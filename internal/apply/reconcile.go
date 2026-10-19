package apply

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ReconcileRequest is everything one Apply runs with besides the captured
// preparation: the invocation's explicit values and the Host's concrete ports.
// Selection is the capture's acquisition location, which publication uses.
// Deadline, when set, is the operator's absolute wall-clock limit; the run's
// own deadline is the earlier of it and the start plus the project's
// executionTime. CheckRequest builds the complete request of one selected
// check; the engine fills each candidate's tree.
type ReconcileRequest struct {
	RunID          string
	Prepared       CapturedSettingsAttempt
	Selection      InputSelection
	OwnerRequest   string
	OwnerDecisions []string
	Deadline       time.Time
	MaxEntries     int
	MaxBytes       int64
	CheckRequest   func(CheckSelection) (run.CheckRequest, error)

	Agents     run.Agents
	Workspaces run.Workspaces
	Runner     run.Checks
	Owner      run.Owner
	Evidence   run.Evidence
	Publisher  Publisher
	Control    ControlCheck
	Clock      func() time.Time
	// RemoveHome deletes one provider-home volume a turn used. The homes are
	// runtime volumes the Agents adapters mount by name, not Workspaces.
	RemoveHome func(context.Context, string) error
	// RequireVolumeSubpaths asks the container engine whether it mounts a
	// directory of a volume. Reconcile calls it before any effect when a Manager's
	// write plan is enforced, because the plan runs as such mounts; it is
	// required then.
	RequireVolumeSubpaths func(context.Context) error
}

// ReconcileResult reports the run and, for a verified candidate, its
// publication. RetainedWork names the shared work volume that was kept: the run
// did not verify, or the verified candidate was not published; it holds the
// work for inspection and a repeated publication.
type ReconcileResult struct {
	Outcome      run.Outcome
	Receipt      *PublicationReceipt
	RetainedWork string
}

// Reconcile runs the permanent Manager team over the captured repository and
// publishes a verified candidate through the guarded Publisher. It never
// commits. Nothing is published unless the run is Verified.
func Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error) {
	cfg, err := reconcileConfig(req, req.now())
	if err != nil {
		return ReconcileResult{}, err
	}
	if slices.ContainsFunc(cfg.Team, func(m run.Manager) bool { return m.Write.Enforced }) {
		if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
			return ReconcileResult{}, err
		}
	}
	outcome, runErr := run.Reconcile(ctx, cfg)
	return req.conclude(ctx, cfg.Start, outcome, runErr)
}

// requireVolumeSubpaths runs the container engine's check that it mounts a
// directory of a volume, which enforced write plans need; the check is required.
func requireVolumeSubpaths(ctx context.Context, check func(context.Context) error) error {
	if check == nil {
		return errors.New("enforced ownership runs as volume subpath mounts: a container engine check for them is required")
	}
	if err := check(ctx); err != nil {
		return fmt.Errorf("enforced ownership cannot run: %w", err)
	}
	return nil
}

// runDeadline is the run's absolute deadline: the project's execution time
// from now, or the operator's earlier deadline when one is set.
func runDeadline(now time.Time, budget time.Duration, operator time.Time) time.Time {
	deadline := now.Add(budget)
	if !operator.IsZero() && operator.Before(deadline) {
		return operator
	}
	return deadline
}

// conclude publishes a verified outcome and cleans up.
func (r ReconcileRequest) conclude(ctx context.Context, start snapshot.Tree, outcome run.Outcome, runErr error) (ReconcileResult, error) {
	result := ReconcileResult{Outcome: outcome}
	var err error
	result.Receipt, result.RetainedWork, err = r.ports().conclude(ctx, outcome, runErr, func(ctx context.Context) (*PublicationReceipt, error) {
		return r.publishChanges(ctx, snapshot.Compare(start, outcome.Candidate))
	})
	return result, err
}

// publishChanges publishes the verified candidate's changes of this run.
func (r ReconcileRequest) publishChanges(ctx context.Context, changes []snapshot.Change) (*PublicationReceipt, error) {
	return publish(ctx, r.Publisher, r.Control, newPublicationRequest(r.RunID, r.Selection, r.Prepared, changes))
}

func reconcileConfig(req ReconcileRequest, now time.Time) (run.Config, error) {
	prepared := req.Prepared
	candidates, err := NewCandidates(prepared)
	if err != nil {
		return run.Config{}, err
	}
	settings, ok := prepared.Settings().Settings()
	if !ok {
		return run.Config{}, errors.New("resolved settings are required")
	}
	budget, ok := prepared.Settings().Budget()
	if !ok {
		return run.Config{}, errors.New("the execution budget is required")
	}
	selections, ok := prepared.Structural().Contexts().Selections()
	if !ok {
		return run.Config{}, errors.New("complete context selections are required")
	}
	selected, ok := prepared.Settings().Checks()
	if !ok {
		return run.Config{}, errors.New("configured check selections are incomplete")
	}
	topology, err := organization.BuildTopology(prepared.Structural().Model().compiled)
	if err != nil {
		return run.Config{}, err
	}
	var checks []run.Check
	var briefChecks []briefCheck
	var requiredIDs []model.Identity
	requiredChecks := make(map[model.Identity][]string)
	for _, check := range selected {
		request, err := req.CheckRequest(check)
		if err != nil {
			return run.Config{}, fmt.Errorf("check %q: %w", check.Declaration().ID.Value, err)
		}
		declaration := check.Declaration()
		requiredBy := ""
		if declaration.RequiredBy != nil {
			requiredBy = identityText(declaration.RequiredBy.Value)
			id := declaration.RequiredBy.Value
			if _, seen := requiredChecks[id]; !seen {
				requiredIDs = append(requiredIDs, id)
			}
			requiredChecks[id] = append(requiredChecks[id], declaration.ID.Value)
		}
		checks = append(checks, run.Check{Name: declaration.ID.Value, RequiredBy: requiredBy, Owner: organization.Root, Request: request})
		briefChecks = append(briefChecks, briefCheck{Name: declaration.ID.Value, Command: strings.Join(append([]string{request.Executable}, request.Args...), " "), Directory: request.Directory.String(), RequiredBy: requiredBy})
	}
	var required []requiredDefinition
	for _, id := range requiredIDs {
		definition, found := compiledDefinition(prepared.Structural().Model().compiled, id)
		if !found {
			return run.Config{}, fmt.Errorf("check %q is required by %s, which the compiled model does not define", requiredChecks[id][0], identityText(id))
		}
		required = append(required, requiredDefinition{Definition: definition, Checks: requiredChecks[id]})
	}
	var team []run.Manager
	for _, manager := range topology.Managers() {
		managerSettings, ok := settings.Manager(manager.Package)
		if !ok {
			return run.Config{}, fmt.Errorf("manager %q has no resolved settings", manager.Package)
		}
		selection := selectionOf(selections, manager.Package)
		children := briefChildren(manager, topology.Managers(), selections, candidates.Areas)
		write := candidates.WritePlan(manager.Package)
		var manageRequired []requiredDefinition
		if !slices.ContainsFunc(selection.Definitions(), func(d organization.ContextDefinition) bool { return d.Definition.Identity.Package == manager.Package }) {
			manageRequired = required
		}
		member := run.Manager{
			Path: manager.Package, Parent: manager.Parent, Children: manager.Children,
			Role:           managerRole(manager, selection),
			ExecuteProfile: managerSettings.Execute.Profile.Value.Name,
			VerifyProfile:  managerSettings.Verify.Profile.Value.Name,
			Brief: renderBrief(briefInput{Manager: manager, Selection: selection, Settings: managerSettings,
				Areas: candidates.Areas(manager.Package), Shard: candidates.Shard(manager.Package), Checks: briefChecks, Required: manageRequired,
				Children: children, Enforced: write.Enforced}),
			Write: write,
		}
		team = append(team, member)
	}
	deadline := runDeadline(now, budget.ExecutionTime.Value, req.Deadline)
	cfg := run.Config{
		RunID: req.RunID, Team: team, Start: prepared.Structural().Model().Inputs().Raw.Census,
		OwnerRequest: req.OwnerRequest, OwnerDecisions: req.OwnerDecisions, Checks: checks,
		Account:    func(tree snapshot.Tree) []run.Finding { return accountFindings(candidates.Account(tree)) },
		OwnerOf:    candidates.OwnerOf,
		Budget:     run.Budget{Deadline: deadline, TurnTimeout: budget.TurnTimeout.Value, AgentTurns: budget.AgentTurns.Value, Rounds: budget.Rounds.Value},
		MaxEntries: req.MaxEntries, MaxBytes: req.MaxBytes,
		Agents: req.Agents, Workspaces: req.Workspaces, Runner: req.Runner, Owner: req.Owner, Evidence: req.Evidence, Clock: req.Clock,
		// Execute turns can see the accounting of the shared tree, such as a shard
		// that does not parse or an unregistered file, before Verify does.
		CheckTool: true,
	}
	return cfg, nil
}

// compiledDefinition returns the exact compiled Definition of an identity.
func compiledDefinition(compiled model.Result, id model.Identity) (model.Definition, bool) {
	for _, owner := range compiled.Models {
		for _, definition := range owner.Definitions {
			if definition.Identity == id {
				return definition, true
			}
		}
	}
	return model.Definition{}, false
}

// accountFindings maps structural candidate findings to blocking run findings
// raised by Markitect itself.
func accountFindings(findings []CandidateFinding) []run.Finding {
	out := make([]run.Finding, 0, len(findings))
	for _, f := range findings {
		observed := f.Message
		if f.Path != "" {
			observed = f.Path + ": " + f.Message
		}
		out = append(out, run.Finding{Owner: f.Owner, RaisedBy: run.MarkitectFinding, Obligation: f.Code, Observed: observed, Blocking: true})
	}
	return out
}

// newPublicationRequest is the publication of a run's verified changes against
// the capture's source guard. run is the run identity the receipt carries.
func newPublicationRequest(run string, selection InputSelection, prepared CapturedSettingsAttempt, changes []snapshot.Change) PublicationRequest {
	inputs := prepared.Structural().Model().Inputs()
	return PublicationRequest{ID: PublicationID(run), Run: run, Repository: selection.Repository, Guard: inputs.Guard, Changes: changes}
}

// publish publishes the verified candidate's changes. A stale source or a change
// set the target cannot represent is a definite receipt without any write.
func publish(ctx context.Context, publisher Publisher, control ControlCheck, request PublicationRequest) (*PublicationReceipt, error) {
	permit, err := publisher.Preflight(ctx, request, control)
	switch {
	case errors.Is(err, ErrCaptureChanged):
		return &PublicationReceipt{ID: request.ID, Outcome: PublicationStale, Reason: err.Error()}, nil
	case errors.Is(err, ErrUnpublishable):
		return &PublicationReceipt{ID: request.ID, Outcome: PublicationFailed, Reason: err.Error()}, nil
	case err != nil:
		return nil, fmt.Errorf("publication preflight: %w", err)
	}
	receipt, err := publisher.Publish(ctx, permit, control)
	if receipt.ID == "" && receipt.Outcome == "" {
		return nil, err
	}
	return &receipt, err
}

func (r ReconcileRequest) now() time.Time {
	if r.Clock == nil {
		return time.Now()
	}
	return r.Clock()
}

// runPorts is what ending one run needs: the ports that hold its work volume
// "<runID>-work", its evidence and its provider homes. The Apply request and
// each phase of an Extract provide one.
type runPorts struct {
	runID      string
	workspaces run.Workspaces
	evidence   run.Evidence
	clock      func() time.Time
	removeHome func(context.Context, string) error
	maxEntries int
	maxBytes   int64
}

func (r ReconcileRequest) ports() runPorts {
	return runPorts{runID: r.RunID, workspaces: r.Workspaces, evidence: r.Evidence, clock: r.Clock, removeHome: r.RemoveHome, maxEntries: r.MaxEntries, maxBytes: r.MaxBytes}
}

func (p runPorts) now() time.Time {
	if p.clock == nil {
		return time.Now()
	}
	return p.clock()
}

func (p runPorts) workVolume() string { return p.runID + "-work" }

// conclude ends a run after run.Reconcile. The turns' homes are of no later
// use. A verified outcome is published by publishVerified, and the shared work
// volume is closed only once that published (applied or no change); after a run
// that did not verify, or any other receipt or a publication error, the volume
// is kept and named as retained work. Cleanup survives a cancelled context.
func (p runPorts) conclude(ctx context.Context, outcome run.Outcome, runErr error, publishVerified func(context.Context) (*PublicationReceipt, error)) (receipt *PublicationReceipt, retained string, err error) {
	p.removeHomes(ctx, outcome.Homes)
	if outcome.Status != run.Verified || runErr != nil {
		return nil, p.retainedWork(ctx), runErr
	}
	receipt, err = publishVerified(ctx)
	if err == nil && receipt != nil && (receipt.Outcome == PublicationApplied || receipt.Outcome == PublicationNoChange) {
		p.closeWork(ctx)
	} else {
		retained = p.retainedWork(ctx)
	}
	return receipt, retained, err
}

// closeWork closes the shared work volume and records the observation.
func (p runPorts) closeWork(ctx context.Context) {
	volume := p.workVolume()
	observation, err := p.workspaces.Close(context.WithoutCancel(ctx), run.WorkspaceID(volume))
	data := map[string]any{"workspace": volume, "closed": observation.Closed, "uncertain": observation.Uncertain}
	if err != nil {
		data["error"] = err.Error()
	}
	_ = p.evidence.Record(run.Event{At: p.now(), Kind: "workspace.close", Data: data})
}

// removeHomes deletes the provider homes and records each observation.
func (p runPorts) removeHomes(ctx context.Context, homes []string) {
	if p.removeHome == nil {
		return
	}
	for _, home := range homes {
		data := map[string]any{"home": home, "removed": true}
		if err := p.removeHome(context.WithoutCancel(ctx), home); err != nil {
			data["removed"], data["error"] = false, err.Error()
		}
		_ = p.evidence.Record(run.Event{At: p.now(), Kind: "home.remove", Data: data})
	}
}

// retainedWork returns the name of the run's shared work volume when it still
// exists, so the work can be inspected or published again. The run closes it
// itself when it ended before the volume held anything.
func (p runPorts) retainedWork(ctx context.Context) string {
	id := run.WorkspaceID(p.workVolume())
	probe, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	if _, err := p.workspaces.Peek(probe, run.HarvestRequest{ID: id, MaxEntries: p.maxEntries, MaxBytes: p.maxBytes}); err != nil {
		return ""
	}
	return string(id)
}
