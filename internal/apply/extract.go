package apply

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExtractRequest is everything one Extract runs with besides the captured
// preparation (made by PreparationService.CaptureExtract): the invocation's
// explicit values and the Host's concrete ports. Deadline, when set, is the
// operator's absolute wall-clock limit; the run's own deadline is the earlier
// of it and the start plus the project's executionTime, shared by both phases.
type ExtractRequest struct {
	RunID          string
	Prepared       CapturedSettingsAttempt
	Selection      InputSelection
	OwnerRequest   string
	OwnerDecisions []string
	Deadline       time.Time
	MaxEntries     int
	MaxBytes       int64

	Agents     run.Agents
	Workspaces run.Workspaces
	Owner      run.Owner
	Evidence   run.Evidence
	Publisher  Publisher
	Control    ControlCheck
	Clock      func() time.Time
	// RemoveHome deletes one provider-home volume a turn used.
	RemoveHome func(context.Context, string) error
	// RequireVolumeSubpaths asks the container engine whether it mounts a
	// directory of a volume. Both phases run with enforced write plans, which are
	// such mounts, so it is required.
	RequireVolumeSubpaths func(context.Context) error
}

// ExtractResult reports the two phases and, for a verified model, its
// publication. Model is nil when the structure phase did not verify.
// RetainedWork names the shared work volume that was kept: a phase did not
// verify, or the verified model was not published.
type ExtractResult struct {
	Structure run.Outcome
	Model     *run.Outcome
	Receipt   *PublicationReceipt
	// ProposedChecks are the checks of a verified model's execution document that
	// the start did not have: a proposal for the owner's adoption, never run.
	ProposedChecks []ProposedCheck
	RetainedWork   string
}

// ProposedCheck is one required check the structure phase proposed. Timeout is
// written as the execution document writes durations, for example "2m".
type ProposedCheck struct {
	ID        string   `json:"id"`
	Profile   string   `json:"profile"`
	Directory string   `json:"directory"`
	Args      []string `json:"args"`
	Timeout   string   `json:"timeout"`
}

// Extract develops a model candidate from the existing artifacts in two phases
// over the same engine as Apply. Phase one: Root alone proposes the Packages
// with an overview Definition each and the layout exceptions they need. Phase
// two: the Manager tree of the verified proposal derives the model of every
// Package and registers the artifacts. A Verified second phase is published
// through the guarded Publisher as the changed .markitect model, registry and
// layout files; an artifact is never published and never changed. It never
// commits.
func Extract(ctx context.Context, req ExtractRequest) (ExtractResult, error) {
	var result ExtractResult
	start, err := newExtractStart(req.Prepared)
	if err != nil {
		return result, err
	}
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return result, err
	}
	deadline := runDeadline(req.now(), start.budget.ExecutionTime.Value, req.Deadline)

	// The structure phase may use at most half of the agent turns, so the model
	// phase always keeps the rest; each phase needs one Execute and one Verify
	// turn of Root at least.
	structureTurns := start.budget.AgentTurns.Value / 2
	if structureTurns < 2 {
		return result, fmt.Errorf("an Extract needs at least 4 agent turns, the project allows %d", start.budget.AgentTurns.Value)
	}
	structure, err := newExtractAccounting(extractStructure, start)
	if err != nil {
		return result, err
	}
	structurePhase := req.phase(extractStructure)
	cfg, err := structurePhase.config(structure, start, deadline, structureTurns)
	if err != nil {
		return result, err
	}
	outcome, runErr := run.Reconcile(ctx, cfg)
	result.Structure = outcome
	structurePhase.ports().removeHomes(ctx, outcome.Homes)
	structurePhase.record(outcome)
	if outcome.Status != run.Verified || runErr != nil {
		result.RetainedWork = structurePhase.ports().retainedWork(ctx)
		return result, runErr
	}
	// Until the model phase holds its own start, the verified structure is only
	// in the structure phase's work volume: keep it when the model phase cannot
	// be prepared or started.
	keepStructure := func(err error) (ExtractResult, error) {
		result.RetainedWork = structurePhase.ports().retainedWork(ctx)
		return result, err
	}
	next, err := start.next(outcome.Candidate)
	if err != nil {
		return keepStructure(err)
	}
	model, err := newExtractAccounting(extractModel, next)
	if err != nil {
		return keepStructure(err)
	}
	phase := req.phase(extractModel)
	cfg, err = phase.config(model, next, deadline, start.budget.AgentTurns.Value-len(outcome.Turns))
	if err != nil {
		return keepStructure(err)
	}
	// Both phases share the owner channel: the model phase knows the structure
	// phase's answers and continues its question numbering.
	cfg.Questions = outcome.Questions
	modelOutcome, runErr := run.Reconcile(ctx, cfg)
	result.Model = &modelOutcome
	phase.ports().removeHomes(ctx, modelOutcome.Homes)
	phase.record(modelOutcome)
	verified := modelOutcome.Status == run.Verified && runErr == nil
	if verified {
		// The model phase holds its own start; the structure's volume has no use left.
		structurePhase.ports().closeWork(ctx)
	}
	// The homes are removed above, before the phase is recorded; conclude must
	// not remove them again.
	concluded := modelOutcome
	concluded.Homes = nil
	receipt, retained, err := phase.ports().conclude(ctx, concluded, runErr, func(ctx context.Context) (*PublicationReceipt, error) {
		result.ProposedChecks = proposedChecks(start, modelOutcome.Candidate)
		changes := snapshot.Compare(start.tree, modelOutcome.Candidate)
		if err := requireModelChangesOnly(changes); err != nil {
			// Account blocks every such candidate, so this is a defect, not an
			// incomplete run.
			modelOutcome.Status, modelOutcome.Reason = run.Failed, err.Error()
			return nil, err
		}
		return publish(ctx, req.Publisher, req.Control, newPublicationRequest(req.RunID, req.Selection, req.Prepared, changes))
	})
	result.Receipt, result.RetainedWork = receipt, retained
	if !verified {
		// A model phase that never held its own work volume leaves the structure
		// as the only result.
		if retained == "" {
			return keepStructure(runErr)
		}
		structurePhase.ports().closeWork(ctx)
	}
	return result, err
}

// proposedChecks lists the checks of the candidate's execution document that the
// start's document does not declare.
func proposedChecks(start extractStage, candidate snapshot.Tree) []ProposedCheck {
	declared := func(structural CapturedPreparationAttempt) []ExecutionCheck {
		if checks := structural.execution.authored.Checks; checks != nil {
			return checks.Value
		}
		return nil
	}
	known := make(map[string]bool)
	for _, check := range declared(start.structural) {
		known[check.ID.Value] = true
	}
	proposed := []ProposedCheck{}
	for _, check := range declared(rebindStructural(start.structural.Model(), candidate, extractModel)) {
		if !known[check.ID.Value] {
			proposed = append(proposed, ProposedCheck{ID: check.ID.Value, Profile: check.Profile.Value, Directory: check.Directory.Value.String(), Args: slices.Clone(check.Args.Value), Timeout: durationText(check.Timeout.Value)})
		}
	}
	return proposed
}

// durationText writes a duration without the zero units that time.Duration
// prints, so that 2 minutes read "2m" as in an execution document.
func durationText(d time.Duration) string {
	text := d.String()
	if strings.HasSuffix(text, "m0s") {
		text = strings.TrimSuffix(text, "0s")
	}
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}
	return text
}

// requireModelChangesOnly refuses to publish anything but the model, the
// registry shards and the execution document: an artifact is never published.
func requireModelChangesOnly(changes []snapshot.Change) error {
	for _, change := range changes {
		name := change.Path().String()
		if !strings.HasPrefix(name, modelDirectory) && !strings.HasPrefix(name, registryDirectory) && name != executionFile {
			return fmt.Errorf("refusing to publish %s: an Extract publishes only the model, the registry and the layout", name)
		}
	}
	return nil
}

// ArtifactDeviation is one Definition that artifacts deviate from: the owners,
// the raising Managers and the observed texts of its open artifact findings.
type ArtifactDeviation struct {
	Definition string   `json:"definition"`
	Owners     []string `json:"owners"`
	RaisedBy   []string `json:"raisedBy"`
	Observed   []string `json:"observed"`
}

// ExtractDeviations groups the open findings with the subject artifact by the
// Definition they name, in Definition order: an artifact that deviates from a
// statement documented intent justifies, which the model phase reports instead
// of changing the model.
func ExtractDeviations(findings []run.Finding) []ArtifactDeviation {
	byDefinition := map[string]*ArtifactDeviation{}
	var deviations []*ArtifactDeviation
	for _, f := range findings {
		if f.Subject != "artifact" || f.Closed {
			continue
		}
		d := byDefinition[f.Definition]
		if d == nil {
			d = &ArtifactDeviation{Definition: f.Definition}
			byDefinition[f.Definition] = d
			deviations = append(deviations, d)
		}
		d.Observed = append(d.Observed, f.Observed)
		if !slices.Contains(d.Owners, f.Owner) {
			d.Owners = append(d.Owners, f.Owner)
		}
		if !slices.Contains(d.RaisedBy, f.RaisedBy) {
			d.RaisedBy = append(d.RaisedBy, f.RaisedBy)
		}
	}
	slices.SortFunc(deviations, func(x, y *ArtifactDeviation) int { return strings.Compare(x.Definition, y.Definition) })
	out := make([]ArtifactDeviation, len(deviations))
	for i, d := range deviations {
		slices.Sort(d.Owners)
		slices.Sort(d.RaisedBy)
		out[i] = *d
	}
	return out
}

func (r ExtractRequest) now() time.Time {
	if r.Clock == nil {
		return time.Now()
	}
	return r.Clock()
}

// extractPhaseRun carries the request of one phase: the volumes and homes of a
// phase are named after the run and the phase, so the two never collide.
type extractPhaseRun struct {
	req   ExtractRequest
	runID string
	phase extractPhaseDescriptor
}

func (r ExtractRequest) phase(p extractPhase) extractPhaseRun {
	d := p.descriptor()
	return extractPhaseRun{req: r, runID: r.RunID + "-" + d.name, phase: d}
}

// ports returns what the shared helpers (home removal, volume closing,
// retention and the run's ending) need of this phase's run.
func (p extractPhaseRun) ports() runPorts {
	r := p.req
	return runPorts{runID: p.runID, workspaces: r.Workspaces, evidence: r.Evidence, clock: r.Clock, removeHome: r.RemoveHome, maxEntries: r.MaxEntries, maxBytes: r.MaxBytes}
}

// record records the phase's outcome.
func (p extractPhaseRun) record(outcome run.Outcome) {
	_ = p.req.Evidence.Record(run.Event{At: p.req.now(), Kind: "extract.phase", Data: map[string]any{"phase": p.phase.name, "status": outcome.Status, "reason": outcome.Reason, "rounds": len(outcome.Rounds), "turns": len(outcome.Turns)}})
}

// config builds the engine configuration of one phase from its start.
func (p extractPhaseRun) config(accounting extractAccounting, stage extractStage, deadline time.Time, agentTurns int) (run.Config, error) {
	team, err := accounting.team(stage)
	if err != nil {
		return run.Config{}, err
	}
	r := p.req
	cfg := run.Config{
		RunID: p.runID, Team: team, Start: stage.tree, OwnerRequest: r.OwnerRequest, OwnerDecisions: r.OwnerDecisions,
		Account:    func(tree snapshot.Tree) []run.Finding { return accountFindings(accounting.Account(tree)) },
		OwnerOf:    accounting.OwnerOf,
		Budget:     run.Budget{Deadline: deadline, TurnTimeout: stage.budget.TurnTimeout.Value, AgentTurns: agentTurns, Rounds: stage.budget.Rounds.Value},
		MaxEntries: r.MaxEntries, MaxBytes: r.MaxBytes,
		Agents: r.Agents, Workspaces: r.Workspaces, Owner: r.Owner, Evidence: r.Evidence, Clock: r.Clock,
		Mandate: p.phase.mandate, VerifyGuidance: p.phase.verifyGuidance, CheckTool: true,
	}
	if accounting.phase == extractModel {
		// The artifacts are fixed evidence of the model phase: a deviation of an
		// artifact is reported against a Definition of the candidate, not repaired.
		cfg.ArtifactFindingsAdvisory, cfg.ResolveDefinition = true, accounting.definitionResolver()
	}
	return cfg, nil
}

// team lists the Managers of the phase, Root first and every parent before its
// children: Root alone in the structure phase, the proposed tree in the model
// phase.
func (a extractAccounting) team(stage extractStage) ([]run.Manager, error) {
	if a.phase == extractStructure {
		return a.structureTeam(stage)
	}
	return a.modelTeam(stage)
}

// structureTeam is Root alone, with the Brief of the structure phase.
func (a extractAccounting) structureTeam(stage extractStage) ([]run.Manager, error) {
	structural := stage.structural
	root, ok := stage.settings.Manager(organization.Root)
	if !ok {
		return nil, errors.New("Root has no resolved settings")
	}
	layout := structural.files.layout
	var dependencies []string
	for _, dependency := range structural.Model().inputs.Raw.Dependencies {
		if area := dependency.RepositoryArea; area.String() != "" {
			dependencies = append(dependencies, area.String())
		}
	}
	slices.Sort(dependencies)
	var roots []string
	for _, r := range layout.Roots() {
		roots = append(roots, r.String())
	}
	brief := renderStructureBrief(structureBriefInput{Roots: roots, Exceptions: layout.Exceptions(), Dependencies: dependencies, Artifacts: artifactPaths(stage.tree), Kinds: vocabularyOf(structural.Model().compiled), Settings: root, Checks: a.declaredChecks(), CheckProfiles: stage.checkProfiles})
	return []run.Manager{{Path: organization.Root, Role: "proposes the first model structure of the whole project", Brief: brief,
		ExecuteProfile: root.Execute.Profile.Value.Name, VerifyProfile: root.Verify.Profile.Value.Name, Write: a.WritePlan(organization.Root)}}, nil
}

// modelTeam is the Manager tree of the proposed topology, each with the Brief of
// the model phase.
func (a extractAccounting) modelTeam(stage extractStage) ([]run.Manager, error) {
	structural := stage.structural
	compiled := structural.Model().compiled
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return nil, err
	}
	selections, ok := structural.Contexts().Selections()
	if !ok {
		return nil, errors.New("complete context selections are required")
	}
	kinds := vocabularyOf(compiled)
	checks := a.declaredChecks()
	artifacts := artifactPaths(stage.tree)
	var team []run.Manager
	for _, manager := range topology.Managers() {
		settings, ok := stage.settings.Manager(manager.Package)
		if !ok {
			return nil, fmt.Errorf("manager %q has no resolved settings", manager.Package)
		}
		selection := selectionOf(selections, manager.Package)
		children := briefChildren(manager, topology.Managers(), selections, a.base.Areas)
		var files []string
		for _, p := range artifacts {
			if a.base.OwnerOf(p) == manager.Package {
				files = append(files, p.String())
			}
		}
		team = append(team, run.Manager{
			Path: manager.Package, Parent: manager.Parent, Children: manager.Children,
			Role: managerRole(manager, selection), ExecuteProfile: settings.Execute.Profile.Value.Name, VerifyProfile: settings.Verify.Profile.Value.Name,
			Brief: renderModelBrief(modelBriefInput{Manager: manager, Selection: selection, Settings: settings, Areas: a.base.Areas(manager.Package), Files: files,
				Shard: a.base.Shard(manager.Package), Children: children, Kinds: kinds, Checks: checks}),
			Write: a.WritePlan(manager.Package),
		})
	}
	return team, nil
}

// declaredChecks describes the project's declared checks for the Briefs. Extract
// does not run them: they verify the artifacts, which Extract leaves as they are.
func (a extractAccounting) declaredChecks() []briefCheck {
	declared := a.start.structural.execution.authored.Checks
	if declared == nil {
		return nil
	}
	var checks []briefCheck
	for _, check := range declared.Value {
		checks = append(checks, briefCheck{Name: check.ID.Value, Command: "profile " + check.Profile.Value + ": " + strings.Join(check.Args.Value, " "), Directory: check.Directory.Value.String()})
	}
	return checks
}
