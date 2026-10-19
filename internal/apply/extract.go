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
	Structure    run.Outcome
	Model        *run.Outcome
	Receipt      *PublicationReceipt
	RetainedWork string
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
	if req.RequireVolumeSubpaths == nil {
		return result, errors.New("enforced ownership runs as volume subpath mounts: a container engine check for them is required")
	}
	if err := req.RequireVolumeSubpaths(ctx); err != nil {
		return result, fmt.Errorf("enforced ownership cannot run: %w", err)
	}
	now := req.now()
	deadline := now.Add(start.budget.ExecutionTime.Value)
	if !req.Deadline.IsZero() && req.Deadline.Before(deadline) {
		deadline = req.Deadline
	}

	structure, err := newExtractAccounting(extractStructure, start)
	if err != nil {
		return result, err
	}
	phase := req.phase("structure")
	cfg, err := phase.config(structure, start, deadline, start.budget.AgentTurns.Value)
	if err != nil {
		return result, err
	}
	outcome, runErr := run.Reconcile(ctx, cfg)
	result.Structure = outcome
	phase.cleanup(ctx, outcome, "structure")
	if outcome.Status != run.Verified || runErr != nil {
		result.RetainedWork = retainedWork(context.WithoutCancel(ctx), phase.reconcile())
		return result, runErr
	}
	closeVolume(context.WithoutCancel(ctx), phase.reconcile(), phase.reconcile().workVolume())

	next, err := start.next(outcome.Candidate)
	if err != nil {
		return result, err
	}
	model, err := newExtractAccounting(extractModel, next)
	if err != nil {
		return result, err
	}
	phase = req.phase("model")
	remaining := start.budget.AgentTurns.Value - len(outcome.Turns)
	if remaining < 1 {
		result.Model = &run.Outcome{Status: run.Incomplete, Reason: "the agent-turn budget was spent by the structure phase"}
		return result, nil
	}
	cfg, err = phase.config(model, next, deadline, remaining)
	if err != nil {
		return result, err
	}
	modelOutcome, runErr := run.Reconcile(ctx, cfg)
	result.Model = &modelOutcome
	phase.cleanup(ctx, modelOutcome, "model")
	if modelOutcome.Status != run.Verified || runErr != nil {
		result.RetainedWork = retainedWork(context.WithoutCancel(ctx), phase.reconcile())
		return result, runErr
	}
	changes := snapshot.Compare(start.tree, modelOutcome.Candidate)
	if err := requireModelChangesOnly(changes); err != nil {
		result.RetainedWork = retainedWork(context.WithoutCancel(ctx), phase.reconcile())
		return result, err
	}
	reconcile := phase.reconcile()
	reconcile.RunID = req.RunID
	receipt, err := publish(ctx, reconcile, changes)
	result.Receipt = receipt
	if err == nil && receipt != nil && (receipt.Outcome == PublicationApplied || receipt.Outcome == PublicationNoChange) {
		closeVolume(context.WithoutCancel(ctx), phase.reconcile(), phase.reconcile().workVolume())
	} else {
		result.RetainedWork = retainedWork(context.WithoutCancel(ctx), phase.reconcile())
	}
	return result, err
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

// ExtractDeviations returns the open non-blocking findings that record an
// artifact deviating from a justified model statement.
func ExtractDeviations(findings []run.Finding) []run.Finding {
	var deviations []run.Finding
	for _, f := range findings {
		if !f.Blocking && !f.Closed && strings.HasPrefix(strings.ToLower(strings.TrimSpace(f.Obligation)), strings.ToLower(RealizationDeviation)) {
			deviations = append(deviations, f)
		}
	}
	return deviations
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
}

func (r ExtractRequest) phase(name string) extractPhaseRun {
	return extractPhaseRun{req: r, runID: r.RunID + "-" + name}
}

// reconcile returns the request the shared Apply helpers (home removal, volume
// closing and retention) use for this phase.
func (p extractPhaseRun) reconcile() ReconcileRequest {
	r := p.req
	return ReconcileRequest{RunID: p.runID, Prepared: r.Prepared, Selection: r.Selection, MaxEntries: r.MaxEntries, MaxBytes: r.MaxBytes,
		Workspaces: r.Workspaces, Evidence: r.Evidence, Publisher: r.Publisher, Control: r.Control, Clock: r.Clock, RemoveHome: r.RemoveHome}
}

// cleanup removes the turns' provider homes and records the phase's outcome.
func (p extractPhaseRun) cleanup(ctx context.Context, outcome run.Outcome, name string) {
	cleanup := context.WithoutCancel(ctx)
	for _, home := range outcome.Homes {
		removeHome(cleanup, p.reconcile(), home)
	}
	_ = p.req.Evidence.Record(run.Event{At: p.req.now(), Kind: "extract.phase", Data: map[string]any{"phase": name, "status": outcome.Status, "reason": outcome.Reason, "rounds": len(outcome.Rounds), "turns": len(outcome.Turns)}})
}

// config builds the engine configuration of one phase from its start.
func (p extractPhaseRun) config(accounting extractAccounting, stage extractStage, deadline time.Time, agentTurns int) (run.Config, error) {
	team, err := accounting.team(stage)
	if err != nil {
		return run.Config{}, err
	}
	mandate, guidance := structureMandate, structureVerifyGuidance
	if accounting.phase == extractModel {
		mandate, guidance = modelMandate, modelVerifyGuidance
	}
	r := p.req
	return run.Config{
		RunID: p.runID, Team: team, Start: stage.tree, OwnerRequest: r.OwnerRequest, OwnerDecisions: r.OwnerDecisions,
		Account:    func(tree snapshot.Tree) []run.Finding { return accountFindings(accounting.Account(tree)) },
		OwnerOf:    accounting.OwnerOf,
		Budget:     run.Budget{Deadline: deadline, TurnTimeout: stage.budget.TurnTimeout.Value, AgentTurns: agentTurns, Rounds: stage.budget.Rounds.Value},
		MaxEntries: r.MaxEntries, MaxBytes: r.MaxBytes,
		Agents: r.Agents, Workspaces: r.Workspaces, Owner: r.Owner, Evidence: r.Evidence, Clock: r.Clock,
		Mandate: mandate, VerifyGuidance: guidance, CheckTool: true,
	}, nil
}

// team lists the Managers of the phase, Root first and every parent before its
// children: Root alone in the structure phase, the proposed tree in the model
// phase.
func (a extractAccounting) team(stage extractStage) ([]run.Manager, error) {
	structural := stage.structural
	compiled := structural.Model().compiled
	checks := a.declaredChecks()
	if a.phase == extractStructure {
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
		brief := renderStructureBrief(structureBriefInput{Roots: roots, Exceptions: layout.Exceptions(), Dependencies: dependencies, Artifacts: artifactPaths(stage.tree), Kinds: vocabularyOf(compiled), Settings: root, Checks: checks})
		return []run.Manager{{Path: organization.Root, Role: "proposes the first model structure of the whole project", Brief: brief,
			ExecuteProfile: root.Execute.Profile.Value.Name, VerifyProfile: root.Verify.Profile.Value.Name, Write: a.WritePlan(organization.Root)}}, nil
	}
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return nil, err
	}
	selections, ok := structural.Contexts().Selections()
	if !ok {
		return nil, errors.New("complete context selections are required")
	}
	selectionOf := func(path string) (selection organization.ContextSelection) {
		for _, candidate := range selections {
			if candidate.Package() == path {
				selection = candidate
			}
		}
		return selection
	}
	kinds := vocabularyOf(compiled)
	artifacts := artifactPaths(stage.tree)
	var team []run.Manager
	for _, manager := range topology.Managers() {
		settings, ok := stage.settings.Manager(manager.Package)
		if !ok {
			return nil, fmt.Errorf("manager %q has no resolved settings", manager.Package)
		}
		selection := selectionOf(manager.Package)
		var children []briefChild
		for _, child := range manager.Children {
			entry := briefChild{Path: child, Areas: a.base.Areas(child)}
			for _, definition := range selectionOf(child).Definitions() {
				if definition.Definition.Identity.Package == child {
					entry.Definitions = append(entry.Definitions, identityText(definition.Definition.Identity)+": "+definitionTitle(definition.Definition))
				}
			}
			children = append(children, entry)
		}
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
