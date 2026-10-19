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
	outcome, runErr := run.Reconcile(ctx, cfg)
	return conclude(ctx, req, cfg.Start, outcome, runErr)
}

// conclude publishes a verified outcome and cleans up. The turns' homes are of
// no later use. The shared work volume is closed only once the verified
// candidate is published (applied or no change); after a run that did not
// verify, or any other receipt or a publication error, it is kept and reported.
func conclude(ctx context.Context, req ReconcileRequest, start snapshot.Tree, outcome run.Outcome, runErr error) (ReconcileResult, error) {
	result := ReconcileResult{Outcome: outcome}
	cleanup := context.WithoutCancel(ctx)
	for _, home := range outcome.Homes {
		closeVolume(cleanup, req, home)
	}
	if outcome.Status != run.Verified || runErr != nil {
		result.RetainedWork = retainedWork(cleanup, req)
		return result, runErr
	}
	receipt, err := publish(ctx, req, snapshot.Compare(start, outcome.Candidate))
	result.Receipt = receipt
	if err == nil && receipt != nil && (receipt.Outcome == PublicationApplied || receipt.Outcome == PublicationNoChange) {
		closeVolume(cleanup, req, req.workVolume())
	} else {
		result.RetainedWork = retainedWork(cleanup, req)
	}
	return result, err
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
		var selection organization.ContextSelection
		for _, candidate := range selections {
			if candidate.Package() == manager.Package {
				selection = candidate
			}
		}
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
				Areas: candidates.Areas(manager.Package), Shard: candidates.Shard(manager.Package), Checks: briefChecks, Required: manageRequired}),
		}
		team = append(team, member)
	}
	deadline := now.Add(budget.ExecutionTime.Value)
	if !req.Deadline.IsZero() && req.Deadline.Before(deadline) {
		deadline = req.Deadline
	}
	cfg := run.Config{
		RunID: req.RunID, Team: team, Start: prepared.Structural().Model().Inputs().Raw.Census,
		OwnerRequest: req.OwnerRequest, OwnerDecisions: req.OwnerDecisions, Checks: checks,
		Account:    func(tree snapshot.Tree) []run.Finding { return accountFindings(candidates.Account(tree)) },
		OwnerOf:    candidates.OwnerOf,
		Budget:     run.Budget{Deadline: deadline, TurnTimeout: budget.TurnTimeout.Value, AgentTurns: budget.AgentTurns.Value, Rounds: budget.Rounds.Value},
		MaxEntries: req.MaxEntries, MaxBytes: req.MaxBytes,
		Agents: req.Agents, Workspaces: req.Workspaces, Runner: req.Runner, Owner: req.Owner, Evidence: req.Evidence, Clock: req.Clock,
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

// publish publishes the verified candidate's changes. A stale source or a change
// set the target cannot represent is a definite receipt without any write.
func publish(ctx context.Context, req ReconcileRequest, changes []snapshot.Change) (*PublicationReceipt, error) {
	inputs := req.Prepared.Structural().Model().Inputs()
	request := PublicationRequest{ID: PublicationID(req.RunID), Run: req.RunID, Repository: req.Selection.Repository, Guard: inputs.Guard, Changes: changes}
	permit, err := req.Publisher.Preflight(ctx, request, req.Control)
	switch {
	case errors.Is(err, ErrCaptureChanged):
		return &PublicationReceipt{ID: request.ID, Outcome: PublicationStale, Reason: err.Error()}, nil
	case errors.Is(err, ErrUnpublishable):
		return &PublicationReceipt{ID: request.ID, Outcome: PublicationFailed, Reason: err.Error()}, nil
	case err != nil:
		return nil, fmt.Errorf("publication preflight: %w", err)
	}
	receipt, err := req.Publisher.Publish(ctx, permit, req.Control)
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

func (r ReconcileRequest) workVolume() string { return r.RunID + "-work" }

func closeVolume(ctx context.Context, req ReconcileRequest, volume string) {
	observation, err := req.Workspaces.Close(ctx, run.WorkspaceID(volume))
	data := map[string]any{"workspace": volume, "closed": observation.Closed, "uncertain": observation.Uncertain}
	if err != nil {
		data["error"] = err.Error()
	}
	_ = req.Evidence.Record(run.Event{At: req.now(), Kind: "workspace.close", Data: data})
}

// retainedWork returns the name of the run's shared work volume when it still
// exists, so the work can be inspected or published again. The run closes it
// itself when it ended before the volume held anything.
func retainedWork(ctx context.Context, req ReconcileRequest) string {
	id := run.WorkspaceID(req.workVolume())
	probe, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if _, err := req.Workspaces.Peek(probe, run.HarvestRequest{ID: id, MaxEntries: req.MaxEntries, MaxBytes: req.MaxBytes}); err != nil {
		return ""
	}
	return string(id)
}
