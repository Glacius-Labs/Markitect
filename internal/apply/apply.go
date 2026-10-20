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

// ApplyRequest is everything one Apply runs with besides the captured
// preparation: the invocation's explicit values and the Runtime's concrete
// ports. Selection is the capture's acquisition location, which publication
// uses. Deadline, when set, is the operator's absolute wall-clock limit; the
// run's own deadline is the earlier of it and the start plus the project's
// executionTime. CheckRequest builds the complete request of one selected
// check; the Orchestrator fills each candidate's tree. Owner is the channel to
// the invoking AI that answers Root's questions; nil leaves Root without one.
type ApplyRequest struct {
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
	// Runtime volumes the Agents adapters mount by name, not Workspaces.
	RemoveHome func(context.Context, string) error
	// RequireVolumeSubpaths asks the container engine whether it mounts a
	// directory or file of a volume. Apply calls it before any effect, because
	// every write plan is enforced through such mounts.
	RequireVolumeSubpaths func(context.Context) error
}

// ApplyResult reports the run and, for a verified candidate, its publication.
// RetainedWork names the shared work volume that was kept: the run did not
// verify, or the verified candidate was not published; it holds the work for
// inspection and a repeated publication.
type ApplyResult struct {
	Outcome      run.Outcome
	Receipt      *PublicationReceipt
	RetainedWork string
}

// The Pass declarations of Apply: one base process, four purposes. Only their
// mandate, effects, payload and acceptance differ.
var applyCorrections = []run.Purpose{run.Plan, run.Prepare, run.Implement}

const planMandate = "Plan your own responsibility concretely: for each of your own obligations the method and the intended evidence; your own artifacts and interfaces; the inputs you need with their owners; the outputs you provide and to whom; the order and start conditions of your work; your dependencies on other Managers in the Prepare and Implement Passes, each on the accepted basis when its contract suffices and on the complete result only when you cannot start without the finished realization; the other Managers' results your own complete result integrates; blockers and assumptions that need clarification. As a parent, also plan how you coordinate your children and integrate their results. Do not plan your children's obligations: they plan their own, and you review their plans for complete responsibilities, compatible interfaces, sound dependencies and fit, and for public values or behaviors they settled that the model does not state. A public behavior the complete linked model leaves materially open without delegating it, such as an unstated limit or the choice between a refusal and a partial fulfilment, is no implementation choice: never settle it by a conventional default; ask your parent with the concrete alternatives and their consequences, so that it reaches the user through Root. A value the model grants as freedom, or fixes by its wording, stays yours to decide and document. Your basis is what your children plan against: as Root, the project constraints, responsibility boundaries and coordination questions; as a parent, your own approach and the contracts you offer. The project is read-only in this Pass; plans are evidence of the run, not files. Your result is your complete plan, integrated with your children's accepted plans: an own obligation is done when its method and evidence are planned. Nothing is implemented in this Pass; a defect of the existing artifacts is planned work for the later Passes, recorded in your plan, never a reason to hold or return a plan. Existing artifacts and tests that already meet an obligation are its evidence as they are, and the independent Verify probes its claims itself: plan a change only for an observed model violation or a realization or evidence the model requires and the artifacts lack, and keep optional hardening and additional tests of conforming behavior as notes."

const prepareMandate = "Supply the usable foundation that your own obligations and accepted plan require before implementation: interfaces, configuration, build basis, documentation or test scaffolding in your own area, following the project's conventions. Preserve an existing foundation that already fits. Your basis is the foundation your children build on. Register every new or changed file, or a directory holding them, in your local registry in the same change. Never realize your children's or another area's obligations; scaffolding is no behavioral realization, and final behavioral tests need not pass yet. Your result is your complete foundation, integrated with your children's: an own obligation is done when the foundation it needs is in place; its behavior follows in the Implement Pass."

const implementMandate = "Realize every applicable own obligation in your own area: code, tests, documentation and configuration, using other owners' realizations through their contracts instead of rebuilding them. Preserve what already conforms; no change is a valid result for a conforming area. Change only what an observed model violation or a missing required realization or evidence needs; optional hardening the model does not require stays a note, not a change. Keep your local registry exact. Your basis is what your children need from your own realization; your complete result covers every own obligation with its artifacts and evidence, after you integrated your children's accepted results and their interactions. Run the project's build and tests yourself. A change you make comes with a test that discriminates it; tests that only add coverage of behavior that already conforms are optional notes, unless the model requires that evidence and the artifacts lack it, since the independent Verify probes the claims itself. When your work needs another Manager's result that is not accepted yet, report wait for that Manager instead of progress."

const verifyMandate = "Assess the immutable candidate independently against the bound model. Before you read the implementation or its tests, derive from the model the distinct claims of each own obligation, the freedom it leaves and the plausible counterexamples it permits: unusual permitted inputs, rejected inputs, exact boundaries, state changes and interactions with other areas. Then try to falsify each material claim through the public interface, including parsing, persistence and integration, not only through internal functions: run such cases as probes with the tool check, each stating its claim and the outcome the model requires, so that the Orchestrator keeps the receipt. Cover every done or preserved own obligation claim by claim with a passing receipt on this candidate, or with an inspected source location and its reasoning where execution does not apply; a passing suite or an author's summary covers no claim by itself. An adverse receipt is a genuine finding when it shows a violation; dispose of it with the model's reason only when your own probe was wrong: its expectation, its assertion, or its execution, such as a probe too large for the check's time limit, which also needs a passing probe of the same claim. Your basis is your own assessment plus the review context your children's reviewers need; your result integrates your children's assessments and the cross-area correctness of their interaction: read their receipts and assertions, ask which plausible violation could still pass them, probe the interactions yourself and return an unsupported claim to its reviewer. Raise every genuine finding with its owner, the obligation, the observed evidence and the earliest kind of correction it needs, and reassess the findings you raised earlier. Never invent a finding or demand optional work; a concrete question about a required claim that you cannot settle goes to your parent."

// rootModelDuty is Root's own duty in an Apply: the bound model is fixed, so a
// model Root judges erroneous or contradictory ends the run.
const rootModelDuty = "As Root you end the run as soon as you judge the bound model erroneous or contradictory: report kind problem, category model, abort true, naming the Definitions concerned. Nobody can change the model during the run: never ask the user to change or override it. When the complete linked model leaves a public behavior materially open and does not delegate it, such as an unstated limit, ask the user with the concrete alternatives instead of letting a Manager pick a default, and accept a plan that depends on it only once the value comes from the model or an answer; an answer within the model lets the work continue."

const verifyGuidance = "A genuine finding is an observed model violation, a missing required realization or evidence item, or a failed required check; cite the obligation and the observed behavior or source. A Definition realized outside its owner's artifacts, or a change outside a Manager's own area, is a model violation; as a parent, check how your children actually use each other's realizations and whether their receipts and assertions cover the claims and public boundaries, and return an unsupported success claim to its reviewer. A missing required realization or evidence item is genuine; doubt alone is not. Optional improvements the model does not require are notes: set blocking to false, and never soften a genuine finding into a note. When another reviewer, below you or about your own responsibility, raised a finding you independently judge erroneous, dispute it with reason and evidence; it stays open until a fresh review of its responsibility disposes of it."

func applyWork() []run.PassSpec {
	return []run.PassSpec{
		{Purpose: run.Plan, Mandate: planMandate, Plans: true, Corrections: applyCorrections},
		{Purpose: run.Prepare, Mandate: prepareMandate, Writes: true, Corrections: applyCorrections},
		{Purpose: run.Implement, Mandate: implementMandate, Writes: true, Corrections: applyCorrections},
	}
}

func applyVerify() run.PassSpec {
	return run.PassSpec{Purpose: run.Verify, Mandate: verifyMandate, Review: true, Guidance: verifyGuidance, Corrections: applyCorrections}
}

// Apply realizes the bound model: after the strict Model check and File check
// of the capture, it runs Plan, Prepare and Implement through the one recursive
// Pass process, then fresh independent Verify of the captured candidate. A
// genuine finding returns to the earliest necessary work Pass and a complete
// fresh Verify follows; a finding disputed with evidence is reassessed on the
// unchanged candidate by a new Verify. Only a clean Verify is published, through
// the guarded Publisher with read-back. It never commits.
func Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	cfg, err := applyConfig(req, req.now())
	if err != nil {
		return ApplyResult{}, err
	}
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return ApplyResult{}, err
	}
	r, err := run.Open(ctx, cfg)
	if err != nil {
		return ApplyResult{}, err
	}
	reason := compose(r, applyWork(), applyVerify())
	outcome, runErr := r.Finish(reason)
	return req.conclude(ctx, cfg.Start, outcome, runErr)
}

// VerifyCaptured runs only the independent Verify Pass of the one Pass process
// on the captured stand, with the same Briefs, checks, ownership, gates and
// budget as Apply but without Execute and without publication. It is the
// evaluation entry for assessing an immutable known stand through the existing
// Pass API (whether fresh reviewers detect a known fault or accept a
// conforming stand); no Command uses it. The shared work volume and the
// provider homes are removed afterwards.
func VerifyCaptured(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	cfg, err := applyConfig(req, req.now())
	if err != nil {
		return ApplyResult{}, err
	}
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return ApplyResult{}, err
	}
	r, err := run.Open(ctx, cfg)
	if err != nil {
		return ApplyResult{}, err
	}
	reason := ""
	if res := r.Pass(applyVerify()); !res.Stopped && res.Verdict != run.Satisfied {
		reason = "the independent Verify of the captured stand needs correction"
	}
	outcome, runErr := r.Finish(reason)
	ports := req.ports()
	ports.removeHomes(ctx, outcome.Homes)
	ports.closeWork(ctx)
	return ApplyResult{Outcome: outcome}, runErr
}

// compose runs the work Passes in order and then the review, as the shared
// workflow of Apply and Extract. The open work findings after a work Pass
// repeat it or restart the work at an earlier purpose they name; a later
// purpose they name comes in order, so no forward Pass is skipped and no review
// starts while a work finding is open. A review that needs correction
// restarts the work at the earliest purpose its open findings name, after any
// reassessment of disputed findings on the unchanged candidate; a clean review
// ends the run verified. The Pass budget bounds every repetition. The reason
// explains an unverified end that the run did not stop by itself.
func compose(r *run.Run, work []run.PassSpec, review run.PassSpec) string {
	next := 0
	for {
		for next < len(work) {
			res := r.Pass(work[next])
			if res.Stopped {
				return ""
			}
			next = followWork(work, next, res)
		}
		res := r.Pass(review)
		for !res.Stopped && res.Verdict == run.NeedsCorrection && r.Reassess() {
			res = r.Pass(review)
		}
		if res.Stopped || res.Verdict == run.Satisfied {
			return ""
		}
		next = earliest(work, r.Corrections(), len(work)-1)
	}
}

// followWork returns the index of the work Pass after the sealed work Pass at
// index current: the earliest purpose up to current that the open work
// findings name, otherwise the next one.
func followWork(work []run.PassSpec, current int, res run.PassResult) int {
	if res.Verdict == run.NeedsCorrection {
		if i := earliest(work, res.Corrections, current+1); i <= current {
			return i
		}
	}
	return current + 1
}

// earliest returns the index of the earliest work purpose the corrections
// name, or fallback when they name none.
func earliest(work []run.PassSpec, corrections []run.Purpose, fallback int) int {
	for i, spec := range work {
		if slices.Contains(corrections, spec.Purpose) {
			return i
		}
	}
	return fallback
}

// requireVolumeSubpaths runs the container engine's check that it mounts a
// directory or file of a volume, which enforced write plans need.
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
func (r ApplyRequest) conclude(ctx context.Context, start snapshot.Tree, outcome run.Outcome, runErr error) (ApplyResult, error) {
	result := ApplyResult{Outcome: outcome}
	var err error
	result.Receipt, result.RetainedWork, err = r.ports().conclude(ctx, outcome, runErr, func(ctx context.Context) (*PublicationReceipt, error) {
		return r.publishChanges(ctx, snapshot.Compare(start, outcome.Candidate))
	})
	return result, err
}

// publishChanges publishes the verified candidate's changes of this run.
func (r ApplyRequest) publishChanges(ctx context.Context, changes []snapshot.Change) (*PublicationReceipt, error) {
	return publish(ctx, r.Publisher, r.Control, newPublicationRequest(r.RunID, r.Selection, r.Prepared, changes))
}

// applyConfig binds the run: the required organization with its Briefs,
// obligations, enforced write plans and team roles, the required checks, the
// gates and the budget. Undeclared or unenforceable ownership refuses the run
// before any effect.
func applyConfig(req ApplyRequest, now time.Time) (run.Config, error) {
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
	checked := prepared.Structural().Model().checked
	topology, err := organization.BuildTopology(checked)
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
		// The project's arguments follow the execution profile's own; a probe
		// keeps the profile's and replaces the project's.
		project := declaration.Args.Value
		profileArgs := len(request.Args) - len(project)
		if profileArgs < 0 || !slices.Equal(request.Args[profileArgs:], project) {
			return run.Config{}, fmt.Errorf("check %q: its request does not end with the declared arguments", declaration.ID.Value)
		}
		checks = append(checks, run.Check{Name: declaration.ID.Value, RequiredBy: requiredBy, Owner: organization.Root, Request: request, ProfileArgs: profileArgs})
		briefChecks = append(briefChecks, briefCheck{Name: declaration.ID.Value, Command: strings.Join(append([]string{request.Executable}, request.Args...), " "), Directory: request.Directory.String(), RequiredBy: requiredBy})
	}
	var required []requiredDefinition
	for _, id := range requiredIDs {
		definition, found := checkedDefinition(checked, id)
		if !found {
			return run.Config{}, fmt.Errorf("check %q is required by %s, which the checked model does not define", requiredChecks[id][0], identityText(id))
		}
		required = append(required, requiredDefinition{Definition: definition, Checks: requiredChecks[id]})
	}
	var team []run.Manager
	for _, manager := range topology.Managers() {
		managerSettings, ok := settings.Manager(manager.Package)
		if !ok {
			return run.Config{}, fmt.Errorf("manager %q has no resolved settings", manager.Package)
		}
		write, err := candidates.WritePlan(manager.Package)
		if err != nil {
			return run.Config{}, err
		}
		selection := selectionOf(selections, manager.Package)
		var obligations []string
		for _, d := range selection.Definitions() {
			if d.Definition.Identity.Package == manager.Package {
				obligations = append(obligations, identityText(d.Definition.Identity))
			}
		}
		input := briefInput{Manager: manager, Selection: selection, Settings: managerSettings,
			Areas: candidates.Areas(manager.Package), Shard: candidates.Shard(manager.Package), Checks: briefChecks,
			Children: briefChildren(manager, topology.Managers(), selections, candidates.Areas)}
		if manager.Package == organization.Root {
			input.Required, input.Project = required, projectObligations(checked)
		}
		team = append(team, run.Manager{
			Path: manager.Package, Parent: manager.Parent, Children: manager.Children,
			Role:           managerRole(manager, selection),
			Brief:          renderBrief(input),
			Obligations:    obligations,
			ExecuteProfile: managerSettings.Execute.Profile.Value.Name,
			VerifyProfile:  managerSettings.Verify.Profile.Value.Name,
			Write:          write,
			Helpers:        helpersOf(managerSettings),
			MaxHelpers:     managerSettings.MaxHelpers,
		})
	}
	deadline := runDeadline(now, budget.ExecutionTime.Value, req.Deadline)
	cfg := run.Config{
		RunID: req.RunID, Team: team, Start: prepared.Structural().Model().Inputs().Raw.Census,
		OwnerRequest: req.OwnerRequest, OwnerDecisions: req.OwnerDecisions, RootGuidance: rootModelDuty, Checks: checks,
		ModelCheck: func(tree snapshot.Tree) []run.Finding { return gateFindings(candidates.ModelCheck(tree)) },
		FileCheck:  func(tree snapshot.Tree) []run.Finding { return gateFindings(candidates.Account(tree)) },
		OwnerOf:    candidates.OwnerOf,
		Registry:   candidates.RegistryView,
		Definition: definitionLookup(checked),
		Budget:     run.Budget{Deadline: deadline, TurnTimeout: budget.TurnTimeout.Value, AgentTurns: budget.AgentTurns.Value, Passes: budget.Passes.Value},
		MaxEntries: req.MaxEntries, MaxBytes: req.MaxBytes,
		Agents: req.Agents, Workspaces: req.Workspaces, Runner: req.Runner, Owner: req.Owner, Evidence: req.Evidence, Clock: req.Clock,
	}
	return cfg, nil
}

// helpersOf binds a Manager's resolved helper roles with their instructions.
func helpersOf(settings organization.ManagerSettings) []run.Helper {
	var helpers []run.Helper
	for _, h := range settings.Helpers {
		var instructions []string
		for _, instruction := range h.Instructions {
			instructions = append(instructions, strings.TrimSpace(string(instruction.Value.Bytes)))
		}
		helpers = append(helpers, run.Helper{Role: h.Role, Profile: h.Profile.Value.Name, Instructions: strings.Join(instructions, "\n\n"), Writes: h.Writes})
	}
	return helpers
}

// checkedDefinition returns the exact checked Definition of an identity.
func checkedDefinition(checked model.Result, id model.Identity) (model.Definition, bool) {
	for _, owner := range checked.Models {
		for _, definition := range owner.Definitions {
			if definition.Identity == id {
				return definition, true
			}
		}
	}
	return model.Definition{}, false
}

// definitionLookup serves look_up of a Definition of the bound model.
func definitionLookup(checked model.Result) func(string) (run.ModelDefinition, bool) {
	return func(id string) (run.ModelDefinition, bool) {
		i := strings.LastIndexByte(id, '/')
		if i <= 0 {
			return run.ModelDefinition{}, false
		}
		definition, found := checkedDefinition(checked, model.Identity{Package: id[:i], ID: id[i+1:]})
		if !found {
			return run.ModelDefinition{}, false
		}
		var b strings.Builder
		writeDefinition(&b, organization.ContextDefinition{Definition: definition}, true, organization.Manager{}, nil)
		return run.ModelDefinition{ID: id, Owner: definition.Identity.Package, Text: strings.TrimSpace(b.String())}, true
	}
}

// gateFindings maps structural findings to blocking findings of the
// Orchestrator.
func gateFindings(findings []CandidateFinding) []run.Finding {
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

func (r ApplyRequest) now() time.Time {
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

func (r ApplyRequest) ports() runPorts {
	return runPorts{runID: r.RunID, workspaces: r.Workspaces, evidence: r.Evidence, clock: r.Clock, removeHome: r.RemoveHome, maxEntries: r.MaxEntries, maxBytes: r.MaxBytes}
}

func (p runPorts) now() time.Time {
	if p.clock == nil {
		return time.Now()
	}
	return p.clock()
}

func (p runPorts) workVolume() string { return p.runID + "-work" }

// conclude ends a run after its Passes. The turns' homes are of no later use. A
// verified outcome is published by publishVerified, and the shared work volume
// is closed only once that published (applied or no change); after a run that
// did not verify, or any other receipt or a publication error, the volume is
// kept and named as retained work. Cleanup survives a cancelled context.
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
