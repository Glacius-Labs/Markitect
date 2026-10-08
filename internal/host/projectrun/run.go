package projectrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

// Run executes an authorized, persisted plan. It makes one work invocation per
// manager followed by one actual integration invocation per manager with
// children. Candidate bytes are staged under .markitect/runs and never written
// into the project working tree by this function.
func Run(ctx context.Context, host Host, invoker Invoker, root, planID string) (RunReport, error) {
	return runOrResume(ctx, host, invoker, root, planID, false)
}

// Resume reconciles the latest durable task states before continuing. Completed
// manager work and integration calls are not replayed.
func Resume(ctx context.Context, host Host, invoker Invoker, root, runID string) (RunReport, error) {
	return runOrResume(ctx, host, invoker, root, runID, true)
}

func runOrResume(ctx context.Context, host Host, invoker Invoker, root, id string, resume bool) (RunReport, error) {
	var empty RunReport
	if host.Load == nil || host.FromSnapshot == nil || invoker == nil {
		return empty, fmt.Errorf("project runtime requires Host load/snapshot and an agent invoker")
	}
	store, err := newRunStore(root)
	if err != nil {
		return empty, err
	}
	unlock, err := store.lock()
	if err != nil {
		return empty, err
	}
	defer unlock()
	plan, err := store.readPlan(id)
	if err != nil {
		return empty, err
	}
	if !plan.ExecuteAuthorized {
		return empty, ErrNotRunnable
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		return empty, err
	}
	project, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return supersedeExisting(store, id, fmt.Errorf("reload fixed plan base: %w", err))
	}
	if project.Snapshot == nil || project.Snapshot.Digest() != plan.BaseSnapshot || project.Digest != plan.BaseProjectDigest || project.Report.ModelDigest != plan.BaseModelDigest {
		return supersedeExisting(store, id, ErrStale)
	}
	working, err := host.Load(root, "")
	if err != nil {
		return supersedeExisting(store, id, err)
	}
	if working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot || working.Digest != plan.WorkingProjectDigest {
		return supersedeExisting(store, id, ErrStale)
	}
	if err := repositoryMatches(root, plan); err != nil {
		return supersedeExisting(store, id, err)
	}
	if err := validateRuntimeBinding(invoker, runtime, plan); err != nil {
		return supersedeExisting(store, id, err)
	}
	dir, err := store.runDir(id)
	if err != nil {
		return empty, err
	}
	report, stateErr := store.readLatestState(id)
	if stateErr != nil && !errors.Is(stateErr, ErrNotFound) {
		return empty, stateErr
	}
	if errors.Is(stateErr, ErrNotFound) {
		if resume {
			return empty, ErrNotFound
		}
		if len(plan.Blockers) > 0 {
			return empty, fmt.Errorf("plan has blocking obligations: %s", strings.Join(plan.Blockers, "; "))
		}
		report = RunReport{APIVersion: APIVersion, ID: id, PlanID: id, Status: StatusRunning,
			Mode: ModeControlledLocal, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			BaseRevision: plan.BaseRevision, BaseSnapshot: plan.BaseSnapshot, ModelDigest: plan.ModelDigest,
			RuntimeDigest: plan.RuntimeDigest, Tasks: cloneTasks(plan.Managers), Candidate: CandidateRef{ID: plan.InitialCandidateID,
				Files: map[string]string{}, Integrated: false}, Revision: 1}
		initial, err := store.readCandidate(dir, plan.InitialCandidateID)
		if err != nil {
			return empty, err
		}
		report.Candidate.Snapshot = initial.Digest
		if err := store.appendState(report); err != nil {
			return empty, err
		}
	} else {
		if !resume {
			return report, fmt.Errorf("run %s already has durable state; use Resume", id)
		}
		if report.Status == StatusVerified || report.Status == StatusApplied {
			return report, nil
		}
		if resume && report.Status != StatusInterrupted {
			return report, fmt.Errorf("run status %s cannot be resumed", report.Status)
		}
		if report.Status == StatusSuperseded {
			return report, ErrStale
		}
		for _, task := range report.Tasks {
			if task.State == "invoking" || task.State == "integrating" || task.State == "uncertain" {
				report.Status = StatusBlocked
				report.Findings = append(report.Findings, "prior process outcome is uncertain for "+task.ManagerID+"; it will not be replayed automatically")
				if err := persistState(store, &report); err != nil {
					return empty, err
				}
				return report, fmt.Errorf("uncertain in-flight manager process requires a new plan: %s", task.ManagerID)
			}
		}
		report.Status = StatusRunning
		report.UpdatedAt = time.Now().UTC()
		report.Revision++
		if err := store.appendState(report); err != nil {
			return empty, err
		}
	}
	deadline := report.StartedAt.Add(time.Duration(runtime.Limits.MaxDuration))
	if !time.Now().Before(deadline) {
		return blockRun(store, report, fmt.Errorf("total runtime duration limit exceeded"))
	}
	boundedCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	ctx = boundedCtx
	starts := len(report.Invocations)
	spent := totalCost(report.Invocations)
	baseCandidate, err := store.readCandidate(dir, plan.InitialCandidateID)
	if err != nil {
		return empty, err
	}
	modelCandidate, err := snapshotWithCandidate(project.Snapshot, baseCandidate)
	if err != nil {
		return empty, err
	}
	boundProject, err := host.FromSnapshot(root, modelCandidate)
	if err != nil {
		return empty, fmt.Errorf("compile planned candidate model: %w", err)
	}
	if boundProject.Report.ModelDigest != plan.ModelDigest || boundProject.Report.Digest != plan.ReportDigest {
		return supersedeExisting(store, id, ErrStale)
	}
	// Work is top-down: parent work scopes direct delegation before children run.
	for i := range report.Tasks {
		task := &report.Tasks[i]
		if task.State == "worked" || task.State == "integrated" || task.State == "complete" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return interruptRun(store, report, err)
		}
		if starts >= runtime.Limits.MaxStarts {
			return blockRun(store, report, fmt.Errorf("runtime start limit %d exceeded", runtime.Limits.MaxStarts))
		}
		if bindErr := ensureWorkingBinding(host, root, plan); bindErr != nil {
			return supersedeExisting(store, id, bindErr)
		}
		parentID := task.ParentTask
		currentID := plan.InitialCandidateID
		if parentID != "" {
			parent := findTask(report.Tasks, parentID)
			if parent == nil || parent.CandidateID == "" {
				return failRun(store, report, fmt.Errorf("parent Manager %s has no completed work candidate", parentID))
			}
			currentID = parent.CandidateID
		}
		current, err := store.readCandidate(dir, currentID)
		if err != nil {
			return failRun(store, report, err)
		}
		input, err := projectForCandidate(host, root, project.Snapshot, current)
		if err != nil {
			return failRun(store, report, err)
		}
		children := activeChildren(report.Tasks, task.ManagerID)
		task.State = "invoking"
		task.Attempts++
		if err := persistState(store, &report); err != nil {
			return empty, err
		}
		proposal, invocation, err := invokeManager(ctx, host, invoker, root, runtime, plan, input, *task, "work", children, nil, nil, starts)
		starts++
		if err != nil {
			if errors.Is(err, ErrStale) {
				return supersedeExisting(store, id, err)
			}
			task.State = "uncertain"
			_ = persistState(store, &report)
			if ctx.Err() != nil {
				return interruptRun(store, report, ctx.Err())
			}
			return failRun(store, report, fmt.Errorf("manager %s work: %w", task.ManagerID, err))
		}
		report.Invocations = append(report.Invocations, invocation)
		spent = addCost(spent, invocation.CostMicros)
		if spent > runtime.Limits.MaxCostMicros {
			return blockRun(store, report, fmt.Errorf("estimated cost limit exceeded"))
		}
		parsed, err := decodeTaskResponse(proposal.Response.ReportJSON, "work", children)
		if err != nil {
			return failRun(store, report, err)
		}
		if parsed.Status == "blocked" {
			return blockRun(store, report, fmt.Errorf("manager %s reported blocked: %s", task.ManagerID, parsed.Summary))
		}
		if parsed.Status == "failed" {
			return failRun(store, report, fmt.Errorf("manager %s reported failed: %s", task.ManagerID, parsed.Summary))
		}
		if err := validateTaskOutcome(proposal.Response.Outcome, parsed); err != nil {
			return failRun(store, report, err)
		}
		if parsed.Status == "complete" && (len(parsed.Questions) > 0 || len(parsed.Risks) > 0) {
			return blockRun(store, report, fmt.Errorf("manager %s reported complete with unresolved questions or risks", task.ManagerID))
		}
		if parsed.Status == "partial" && len(parsed.Questions) == 0 && len(parsed.Risks) == 0 {
			return blockRun(store, report, fmt.Errorf("manager %s reported partial without an actionable question or risk", task.ManagerID))
		}
		if parsed.EscalateTo != "" {
			if err := recordEscalation(&report, *task, parsed); err != nil {
				return blockRun(store, report, err)
			}
		}
		for _, delegation := range parsed.Delegations {
			child := findTask(report.Tasks, delegation.ManagerID)
			if child == nil {
				return failRun(store, report, fmt.Errorf("delegation target %s is not in the active plan", delegation.ManagerID))
			}
			child.Goal = delegation.Goal
		}
		if parsed.Status == "no-op" && len(proposal.Response.CandidateFiles) > 0 {
			return failRun(store, report, fmt.Errorf("manager %s claimed no-op while proposing files", task.ManagerID))
		}
		candidate, err := applyProposal(current, proposal.Response.CandidateFiles, input.Report, *task, "work", nil, runtime.Limits)
		if err != nil {
			return failRun(store, report, err)
		}
		candidate.ID, err = newID()
		if err != nil {
			return failRun(store, report, err)
		}
		candidate.Parents = []string{current.ID}
		if err := store.writeCandidate(dir, candidate); err != nil {
			return failRun(store, report, err)
		}
		candidate, err = store.readCandidate(dir, candidate.ID)
		if err != nil {
			return failRun(store, report, err)
		}
		task.State, task.ReportID, task.CandidateID = "worked", invocation.ReportID, candidate.ID
		task.WrittenPaths = proposalPaths(proposal.Response.CandidateFiles)
		task.Summary, task.Questions, task.Risks, task.Delegations, task.ReportStatus = parsed.Summary, parsed.Questions, parsed.Risks, parsed.Delegations, parsed.Status
		report.Candidate = candidateRef(candidate, false)
		if err := persistState(store, &report); err != nil {
			return empty, err
		}
	}
	// Integration is bottom-up. Each parent receives actual child candidates and
	// must report that it integrated them; reports alone cannot close the tree.
	indices := make([]int, len(report.Tasks))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool { return report.Tasks[indices[i]].Depth > report.Tasks[indices[j]].Depth })
	for _, index := range indices {
		task := &report.Tasks[index]
		children := activeChildren(report.Tasks, task.ManagerID)
		if len(children) == 0 || task.State == "integrated" || task.State == "complete" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return interruptRun(store, report, err)
		}
		if starts >= runtime.Limits.MaxStarts {
			return blockRun(store, report, fmt.Errorf("runtime start limit %d exceeded", runtime.Limits.MaxStarts))
		}
		if bindErr := ensureWorkingBinding(host, root, plan); bindErr != nil {
			return supersedeExisting(store, id, bindErr)
		}
		merged, conflicts, err := mergeChildCandidates(store, dir, report.Tasks, *task, children)
		if err != nil {
			return failRun(store, report, err)
		}
		input, err := projectForCandidate(host, root, project.Snapshot, merged)
		if err != nil {
			return failRun(store, report, err)
		}
		childSummaries := buildChildSummaries(store, dir, report.Tasks, children)
		task.State = "integrating"
		if err := persistState(store, &report); err != nil {
			return empty, err
		}
		proposal, invocation, err := invokeManager(ctx, host, invoker, root, runtime, plan, input, *task, "integrate", children, conflicts, childSummaries, starts)
		starts++
		if err != nil {
			if errors.Is(err, ErrStale) {
				return supersedeExisting(store, id, err)
			}
			task.State = "uncertain"
			_ = persistState(store, &report)
			if ctx.Err() != nil {
				return interruptRun(store, report, ctx.Err())
			}
			return failRun(store, report, fmt.Errorf("manager %s integration: %w", task.ManagerID, err))
		}
		report.Invocations = append(report.Invocations, invocation)
		spent = addCost(spent, invocation.CostMicros)
		if spent > runtime.Limits.MaxCostMicros {
			return blockRun(store, report, fmt.Errorf("estimated cost limit exceeded"))
		}
		parsed, err := decodeTaskResponse(proposal.Response.ReportJSON, "integrate", nil)
		if err != nil {
			return failRun(store, report, err)
		}
		if err := validateTaskOutcome(proposal.Response.Outcome, parsed); err != nil {
			return failRun(store, report, err)
		}
		if parsed.Status == "blocked" || parsed.Status == "failed" || parsed.Status == "no-op" {
			return blockRun(store, report, fmt.Errorf("manager %s integration is %s: %s", task.ManagerID, parsed.Status, parsed.Summary))
		}
		resolved, err := applyProposal(merged, proposal.Response.CandidateFiles, input.Report, *task, "integrate", conflicts, runtime.Limits)
		if err != nil {
			return failRun(store, report, err)
		}
		if len(conflicts) > 0 && !proposesEvery(proposal.Response.CandidateFiles, conflicts) {
			return blockRun(store, report, fmt.Errorf("manager %s did not resolve child path conflict(s): %s", task.ManagerID, strings.Join(conflicts, ", ")))
		}
		outstandingQ, outstandingR, obligationsErr := managerObligations(*task, report.Tasks, children)
		if obligationsErr != nil {
			return blockRun(store, report, fmt.Errorf("manager %s obligations: %w", task.ManagerID, obligationsErr))
		}
		resolvedQ, remainingQ, err := resolveObligations(outstandingQ, parsed.ResolvedQuestions)
		if err != nil {
			return blockRun(store, report, fmt.Errorf("manager %s questions: %w", task.ManagerID, err))
		}
		resolvedR, remainingR, err := resolveObligations(outstandingR, parsed.ResolvedRisks)
		if err != nil {
			return blockRun(store, report, fmt.Errorf("manager %s risks: %w", task.ManagerID, err))
		}
		_ = resolvedQ
		_ = resolvedR
		remainingQ = append(remainingQ, parsed.Questions...)
		remainingR = append(remainingR, parsed.Risks...)
		if len(remainingQ)+len(remainingR) > 0 {
			if parsed.Status != "partial" || parsed.EscalateTo != escalationTarget(*task) {
				return blockRun(store, report, fmt.Errorf("manager %s has unresolved obligations that did not escalate to its nearest parent", task.ManagerID))
			}
			if err := recordEscalation(&report, *task, TaskResponse{Summary: parsed.Summary, Questions: remainingQ, Risks: remainingR, EscalateTo: parsed.EscalateTo}); err != nil {
				return blockRun(store, report, err)
			}
		} else if parsed.Status != "complete" {
			return blockRun(store, report, fmt.Errorf("manager %s integration is partial without unresolved obligations", task.ManagerID))
		}
		for _, childID := range children {
			child := findTask(report.Tasks, childID)
			if child != nil && containsAll(parsed.ResolvedQuestions, child.Questions) && containsAll(parsed.ResolvedRisks, child.Risks) {
				child.Questions = []string{}
				child.Risks = []string{}
				if child.ReportStatus == "partial" {
					child.ReportStatus = "complete"
				}
				for i := range report.Escalations {
					if report.Escalations[i].Status == "open" && report.Escalations[i].ToManager == task.ManagerID && containsString(report.Escalations[i].AffectedTasks, child.ID) {
						report.Escalations[i].Status = "resolved"
					}
				}
			}
		}
		if len(remainingQ)+len(remainingR) > 0 {
			for i := range report.Escalations {
				if report.Escalations[i].Status == "open" && report.Escalations[i].ToManager == task.ManagerID {
					report.Escalations[i].Status = "escalated"
				}
			}
		}
		task.Questions = remainingQ
		task.Risks = remainingR
		if len(remainingQ) == 0 && len(remainingR) == 0 {
			task.ReportStatus = "complete"
		} else {
			task.ReportStatus = "partial"
		}
		resolved.ID, err = newID()
		if err != nil {
			return failRun(store, report, err)
		}
		resolved.Parents = childCandidateIDs(report.Tasks, children)
		if err := store.writeCandidate(dir, resolved); err != nil {
			return failRun(store, report, err)
		}
		resolved, err = store.readCandidate(dir, resolved.ID)
		if err != nil {
			return failRun(store, report, err)
		}
		task.State, task.IntegrationReportID, task.IntegrationCandidateID = "integrated", invocation.ReportID, resolved.ID
		task.IntegratedPaths = changedCandidatePaths(merged, resolved)
		task.Summary = parsed.Summary
		report.Candidate = candidateRef(resolved, false)
		if err := persistState(store, &report); err != nil {
			return empty, err
		}
	}
	rootCandidateID := findRootCandidate(report.Tasks)
	if rootCandidateID == "" {
		rootCandidateID = report.Candidate.ID
	}
	finalCandidate, err := store.readCandidate(dir, rootCandidateID)
	if err != nil {
		return failRun(store, report, err)
	}
	for _, task := range report.Tasks {
		if (task.ReportStatus != "complete" && task.ReportStatus != "no-op") || len(task.Questions) > 0 || len(task.Risks) > 0 {
			return blockRun(store, report, fmt.Errorf("manager %s has unresolved report status or obligations", task.ManagerID))
		}
	}
	for _, escalation := range report.Escalations {
		if escalation.Status == "open" {
			return blockRun(store, report, fmt.Errorf("unresolved escalation %s from %s", escalation.ID, escalation.FromManager))
		}
	}
	if err := validateFinalCandidate(host, root, project.Snapshot, finalCandidate, plan); err != nil {
		return blockRun(store, report, err)
	}
	report.Candidate = candidateRef(finalCandidate, true)
	report.Status = StatusIntegrated
	report.UpdatedAt = time.Now().UTC()
	if err := persistState(store, &report); err != nil {
		return empty, err
	}
	return report, nil
}

func invokeManager(ctx context.Context, host Host, invoker Invoker, root string, runtime Runtime, plan PlanRecord, project *Project, task ManagerTask, phase string, activeChildIDs, conflicts []string, childReports []childReport, start int) (agentexec.RunResult, InvocationLog, error) {
	var result agentexec.RunResult
	var log InvocationLog
	configAgent, ok := runtime.Agents[task.ManagerID]
	if !ok {
		return result, log, fmt.Errorf("no configured agent for manager %s", task.ManagerID)
	}
	config, err := configAgent.AgentConfig()
	if err != nil {
		return result, log, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return result, log, context.DeadlineExceeded
		}
		if remaining < config.Timeout {
			config.Timeout = remaining
		}
	}
	managerContext, err := projectmodel.Context(project.Report, task.ManagerID)
	if err != nil {
		return result, log, err
	}
	active := map[string]bool{}
	for _, id := range activeChildIDs {
		active[id] = true
	}
	filteredChildren := managerContext.Children[:0]
	for _, child := range managerContext.Children {
		if active[child.ID] {
			filteredChildren = append(filteredChildren, child)
		}
	}
	managerContext.Children = filteredChildren
	ctxPayload := struct {
		Phase           string                      `json:"phase"`
		Goal            string                      `json:"goal"`
		Manager         projectmodel.ManagerContext `json:"manager"`
		DirectChildren  []string                    `json:"directChildren"`
		ChildReports    []childReport               `json:"childReports,omitempty"`
		ConflictPaths   []string                    `json:"conflictPaths,omitempty"`
		CandidateDigest string                      `json:"candidateDigest"`
		ResponseSchema  json.RawMessage             `json:"responseSchema"`
	}{Phase: phase, Goal: task.Goal, Manager: managerContext, DirectChildren: activeChildrenFromContext(managerContext), ChildReports: childReports, ConflictPaths: conflicts, CandidateDigest: project.Snapshot.Digest(), ResponseSchema: taskResponseSchema(phase)}
	// Parent integration sees only direct child summaries and candidate digests,
	// never their transcripts or private logs.
	_ = start
	contextJSON, err := json.Marshal(ctxPayload)
	if err != nil {
		return result, log, err
	}
	artifacts, err := scopedArtifacts(project, task, configAgent, runtime.Limits, phase)
	if err != nil {
		return result, log, err
	}
	scopeIDs := append([]string{task.ManagerID}, task.Artifacts...)
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: project.Revision, ModelDigest: project.Report.ModelDigest,
		ModulePin: project.Report.Digest, ProjectionID: project.Report.Digest, ScopeIDs: uniqueSorted(scopeIDs), PolicyIDs: append([]string(nil), task.Checks...), Context: contextJSON, Artifacts: artifacts}
	result, err = invoker.Run(ctx, config, request, agentexec.RunOptions{PrivateLogDirectory: filepath.Join(root, ".markitect", "runs", "private")})
	if err != nil {
		return result, log, err
	}
	if result.Response.Role != agentexec.RoleExecutor || result.Receipt.Outcome != result.Response.Outcome || (result.Response.Outcome != agentexec.OutcomeProposed && result.Response.Outcome != agentexec.OutcomeEscalated) {
		return result, log, fmt.Errorf("agent did not return a typed proposed or escalated executor result")
	}
	// A fresh project read after each external process catches concurrent edits
	// without auditing mutable operational state under .markitect/runs.
	fresh, err := host.Load(root, "")
	if err != nil {
		return result, log, err
	}
	if fresh.Snapshot.Digest() != plan.WorkingSnapshot || fresh.Digest != plan.WorkingProjectDigest {
		return result, log, ErrStale
	}
	if err := repositoryMatches(root, plan); err != nil {
		return result, log, err
	}
	configFingerprint, err := invoker.Fingerprint(config)
	if err != nil {
		return result, log, err
	}
	if plan.RuntimeAgents[task.ManagerID] != configFingerprint {
		return result, log, ErrStale
	}
	cost, known := estimateCost(result.Receipt.Usage, configAgent.Pricing)
	if !known {
		return result, log, fmt.Errorf("agent usage is missing; bounded cost cannot be asserted")
	}
	log = InvocationLog{TaskID: task.ID, Role: agentexec.RoleExecutor, Phase: phase, InputDigest: result.Receipt.InputDigest, Receipt: result.Receipt, ReportID: result.Receipt.RunID, Outcome: result.Receipt.Outcome, CostMicros: cost}
	return result, log, nil
}

type childReport struct {
	ManagerID       string   `json:"managerId"`
	Summary         string   `json:"summary"`
	Status          string   `json:"status"`
	Questions       []string `json:"questions"`
	Risks           []string `json:"risks"`
	CandidateDigest string   `json:"candidateDigest"`
}

func scopedArtifacts(project *Project, task ManagerTask, agent Agent, limits Limits, phase string) ([]agentexec.Artifact, error) {
	paths := map[string]bool{}
	for _, entry := range project.Report.Files {
		if entry.Owner == task.ManagerID {
			if _, ok := project.Snapshot.Files[entry.Path]; ok {
				paths[entry.Path] = true
			}
		}
	}
	for _, id := range task.Artifacts {
		for _, artifact := range project.Report.Artifacts {
			if artifact.ID == id {
				for _, path := range artifact.Paths {
					if _, ok := project.Snapshot.Files[path]; ok {
						paths[path] = true
					}
				}
			}
		}
	}
	if phase == "integrate" {
		children := map[string]bool{}
		for _, m := range project.Report.Managers {
			if m.Parent == task.ManagerID {
				children[m.ID] = true
			}
		}
		for _, artifact := range project.Report.Artifacts {
			if !children[artifact.Owner] || !artifact.Required {
				continue
			}
			for _, path := range artifact.Paths {
				if strings.HasSuffix(path, "/") {
					for existing := range project.Snapshot.Files {
						if strings.HasPrefix(existing, path) {
							paths[existing] = true
						}
					}
				} else if _, ok := project.Snapshot.Files[path]; ok {
					paths[path] = true
				}
			}
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	var out []agentexec.Artifact
	var total int64
	for _, path := range ordered {
		content := project.Snapshot.Files[path]
		if int64(len(content)) > limits.MaxCandidateFileBytes {
			return nil, fmt.Errorf("input file %s exceeds per-file limit", path)
		}
		total += int64(len(content))
		if total > limits.MaxCandidateBytes {
			return nil, fmt.Errorf("manager input exceeds candidate byte limit")
		}
		mode := protocolMode(project.Snapshot.Modes[path])
		if mode == "" {
			return nil, fmt.Errorf("input file %s has unsupported source mode %q", path, project.Snapshot.Modes[path])
		}
		d := rawContentDigest(content)
		out = append(out, agentexec.Artifact{Path: path, Mode: mode, Digest: d, Content: append([]byte(nil), content...)})
	}
	return out, nil
}

func applyProposal(base candidateData, proposals []agentexec.CandidateFile, report projectmodel.Report, task ManagerTask, phase string, conflictPaths []string, limits Limits) (candidateData, error) {
	files := map[string]File{}
	for p, f := range base.Files {
		f.Content = append([]byte(nil), f.Content...)
		files[p] = f
	}
	seen := map[string]bool{}
	conflictSet := map[string]bool{}
	if phase == "integrate" {
		for _, path := range conflictPaths {
			conflictSet[path] = true
		}
	}
	var total int64
	for _, p := range proposals {
		if !safeRepoPath(p.Path) {
			return candidateData{}, fmt.Errorf("proposal path is unsafe: %q", p.Path)
		}
		if seen[strings.ToLower(p.Path)] {
			return candidateData{}, fmt.Errorf("duplicate proposal path %s", p.Path)
		}
		seen[strings.ToLower(p.Path)] = true
		if forbiddenRuntimePath(p.Path) {
			return candidateData{}, fmt.Errorf("proposal may not write protected path %s", p.Path)
		}
		owner, known := ownerForPath(report, p.Path)
		if !known || (owner != task.ManagerID && !conflictSet[p.Path]) {
			return candidateData{}, fmt.Errorf("manager %s proposed path owned by %q (phase %s): %s", task.ManagerID, owner, phase, p.Path)
		}
		if int64(len(p.Content)) > limits.MaxCandidateFileBytes {
			return candidateData{}, fmt.Errorf("proposal file %s exceeds size limit", p.Path)
		}
		mode := snapshotMode(p.Mode)
		if mode == "" {
			return candidateData{}, fmt.Errorf("proposal file %s has unsupported mode %q", p.Path, p.Mode)
		}
		data := []byte(p.Content)
		total += int64(len(data))
		if total > limits.MaxCandidateBytes {
			return candidateData{}, fmt.Errorf("proposal exceeds candidate byte limit")
		}
		files[p.Path] = File{Path: p.Path, Mode: mode, Content: data}
	}
	base.Files = files
	return base, nil
}

func forbiddenRuntimePath(path string) bool {
	lower := strings.ToLower(path)
	return lower == ".markitect/project.yaml" || lower == ".markitect/runtime.yaml" || strings.HasPrefix(lower, ".markitect/runs/") || strings.HasPrefix(lower, ".markitect/views/") || strings.HasPrefix(lower, ".markitect/cache/") || strings.HasPrefix(lower, ".markitect/model/")
}
func ownedPath(owns []string, path string) bool {
	for _, own := range owns {
		if pathWithin(path, own) {
			return true
		}
	}
	return false
}
func pathWithin(path, prefix string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "." || prefix == "" {
		return safeRepoPath(path)
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func ownerForPath(report projectmodel.Report, path string) (string, bool) {
	for _, entry := range report.Files {
		if entry.Path == path && entry.Owner != "" {
			return entry.Owner, true
		}
	}
	for _, artifact := range report.Artifacts {
		for _, expected := range artifact.Paths {
			if strings.HasSuffix(expected, "/") {
				if strings.HasPrefix(path, expected) {
					return artifact.Owner, true
				}
			} else if expected == path {
				return artifact.Owner, true
			}
		}
	}
	best := -1
	owner := ""
	ambiguous := false
	for _, manager := range report.Managers {
		for _, prefix := range manager.Owns {
			if pathWithin(path, prefix) {
				score := len(strings.TrimSuffix(prefix, "/"))
				if prefix == "." || prefix == "" {
					score = 0
				}
				if score > best {
					best = score
					owner = manager.ID
					ambiguous = false
				} else if score == best && owner != manager.ID {
					ambiguous = true
				}
			}
		}
	}
	return owner, best >= 0 && !ambiguous
}
func activeChildren(tasks []ManagerTask, parent string) []string {
	var out []string
	for _, t := range tasks {
		if t.ParentTask == parent {
			out = append(out, t.ManagerID)
		}
	}
	sort.Strings(out)
	return out
}
func activeChildrenFromContext(c projectmodel.ManagerContext) []string {
	out := make([]string, 0, len(c.Children))
	for _, ch := range c.Children {
		out = append(out, ch.ID)
	}
	sort.Strings(out)
	return out
}
func projectForCandidate(host Host, root string, base *Snapshot, c candidateData) (*Project, error) {
	snap, err := snapshotWithCandidate(base, c)
	if err != nil {
		return nil, err
	}
	return host.FromSnapshot(root, snap)
}
func candidateSnapshotHash(base *Snapshot, c candidateData) string {
	snap, err := snapshotWithCandidate(base, c)
	if err != nil {
		return ""
	}
	return snap.Digest()
}
func candidateRef(c candidateData, integrated bool) CandidateRef {
	hashes := map[string]string{}
	for p, f := range c.Files {
		if f.Delete {
			hashes[p] = "deleted"
		} else {
			hashes[p] = rawContentDigest(f.Content)
		}
	}
	return CandidateRef{ID: c.ID, Snapshot: c.Digest, Files: hashes, Parents: append([]string(nil), c.Parents...), Integrated: integrated}
}
func persistState(store *runStore, r *RunReport) error {
	r.Revision++
	r.UpdatedAt = time.Now().UTC()
	return store.appendState(*r)
}
func interruptRun(s *runStore, r RunReport, cause error) (RunReport, error) {
	r.Status = StatusInterrupted
	r.Findings = append(r.Findings, cause.Error())
	if err := persistState(s, &r); err != nil {
		return r, err
	}
	return r, cause
}
func blockRun(s *runStore, r RunReport, cause error) (RunReport, error) {
	r.Status = StatusBlocked
	r.Findings = append(r.Findings, cause.Error())
	if err := persistState(s, &r); err != nil {
		return r, err
	}
	return r, cause
}
func failRun(s *runStore, r RunReport, cause error) (RunReport, error) {
	r.Status = StatusFailed
	r.Findings = append(r.Findings, cause.Error())
	if err := persistState(s, &r); err != nil {
		return r, err
	}
	return r, cause
}

func supersedeExisting(store *runStore, id string, cause error) (RunReport, error) {
	report, err := store.readLatestState(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RunReport{}, cause
		}
		return RunReport{}, err
	}
	if report.Status == StatusApplied {
		return report, cause
	}
	if report.Status != StatusSuperseded {
		report.Status = StatusSuperseded
		report.Findings = append(report.Findings, "bound project inputs, runtime or Git target changed: "+cause.Error())
		if persistErr := persistState(store, &report); persistErr != nil {
			return report, persistErr
		}
	}
	return report, cause
}
func cloneTasks(input []ManagerTask) []ManagerTask {
	out := append([]ManagerTask(nil), input...)
	for i := range out {
		out[i].Owns = append([]string(nil), out[i].Owns...)
		out[i].Statements = append([]string(nil), out[i].Statements...)
		out[i].Artifacts = append([]string(nil), out[i].Artifacts...)
		out[i].Checks = append([]string(nil), out[i].Checks...)
	}
	return out
}
func totalCost(logs []InvocationLog) int64 {
	var sum int64
	for _, l := range logs {
		if l.CostMicros < 0 || sum > math.MaxInt64-l.CostMicros {
			return math.MaxInt64
		}
		sum += l.CostMicros
	}
	return sum
}

func addCost(current, next int64) int64 {
	if current < 0 || next < 0 || current > math.MaxInt64-next {
		return math.MaxInt64
	}
	return current + next
}
func estimateCost(usage *agentexec.Usage, p Pricing) (int64, bool) {
	if usage == nil || usage.InputTokens == nil || usage.OutputTokens == nil {
		return 0, false
	}
	in := *usage.InputTokens
	out := *usage.OutputTokens
	if in < 0 || out < 0 {
		return 0, false
	}
	value := new(big.Int).Mul(big.NewInt(in), big.NewInt(p.InputMicrosPerMillion))
	value.Add(value, new(big.Int).Mul(big.NewInt(out), big.NewInt(p.OutputMicrosPerMillion)))
	value.Div(value, big.NewInt(1_000_000))
	if !value.IsInt64() {
		return math.MaxInt64, false
	}
	return value.Int64(), true
}
func rawContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func protocolMode(mode string) string {
	switch mode {
	case "100644":
		return "0644"
	case "100755":
		return "0755"
	default:
		return ""
	}
}
func snapshotMode(mode string) string {
	switch mode {
	case "0644":
		return snapshot.RegularMode
	case "0755":
		return snapshot.ExecutableMode
	default:
		return ""
	}
}
func validateRuntimeBinding(invoker Invoker, runtime Runtime, plan PlanRecord) error {
	d, err := runtimeDigestWithInvoker(invoker, runtime)
	if err != nil {
		return err
	}
	if d != plan.RuntimeDigest {
		return ErrStale
	}
	return nil
}

func repositoryMatches(root string, plan PlanRecord) error {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return err
	}
	if identity.Digest != plan.RepositoryDigest {
		return ErrStale
	}
	head, err := resolveGitHead(root)
	if err != nil {
		return err
	}
	branch, err := resolveGitBranch(root)
	if err != nil {
		return err
	}
	if head != plan.TargetHead || branch != plan.TargetBranch {
		return ErrStale
	}
	return nil
}

func ensureWorkingBinding(host Host, root string, plan PlanRecord) error {
	working, err := host.Load(root, "")
	if err != nil {
		return err
	}
	if working == nil || working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot || working.Digest != plan.WorkingProjectDigest {
		return ErrStale
	}
	return repositoryMatches(root, plan)
}

func mergeChildCandidates(store *runStore, dir string, tasks []ManagerTask, parent ManagerTask, children []string) (candidateData, []string, error) {
	base, err := store.readCandidate(dir, parent.CandidateID)
	if err != nil {
		return base, nil, err
	}
	merged := map[string]File{}
	for p, f := range base.Files {
		merged[p] = f
	}
	touched := map[string]string{}
	conflicts := map[string]bool{}
	var parents []string
	parents = append(parents, base.ID)
	for _, child := range children {
		task := findTask(tasks, child)
		if task == nil {
			continue
		}
		cid := task.IntegrationCandidateID
		if cid == "" {
			cid = task.CandidateID
		}
		if cid == "" {
			return base, nil, fmt.Errorf("child %s has no candidate", child)
		}
		candidate, err := store.readCandidate(dir, cid)
		if err != nil {
			return base, nil, err
		}
		parents = append(parents, cid)
		for p, f := range candidate.Files {
			original, ok := base.Files[p]
			if ok && fileEqual(original, f) {
				continue
			}
			if !ok && f.Delete {
				continue
			}
			if prior, ok := touched[p]; ok && prior != child {
				conflicts[p] = true
				continue
			}
			touched[p] = child
			merged[p] = f
		}
	}
	base.Files = merged
	base.Parents = parents
	var conflictPaths []string
	for p := range conflicts {
		conflictPaths = append(conflictPaths, p)
	}
	sort.Strings(conflictPaths)
	base.ID, err = newID()
	if err != nil {
		return base, nil, err
	}
	if err := store.writeCandidate(dir, base); err != nil {
		return base, nil, err
	}
	return base, conflictPaths, nil
}
func fileEqual(a, b File) bool {
	return a.Path == b.Path && a.Mode == b.Mode && a.Delete == b.Delete && string(a.Content) == string(b.Content)
}

func proposalPaths(proposals []agentexec.CandidateFile) []string {
	paths := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		paths = append(paths, proposal.Path)
	}
	sort.Strings(paths)
	return paths
}

func changedCandidatePaths(before, after candidateData) []string {
	set := map[string]bool{}
	for path := range before.Files {
		set[path] = true
	}
	for path := range after.Files {
		set[path] = true
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		old, oldOK := before.Files[path]
		current, currentOK := after.Files[path]
		if oldOK != currentOK || (oldOK && !fileEqual(old, current)) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}
func findTask(tasks []ManagerTask, id string) *ManagerTask {
	for i := range tasks {
		if tasks[i].ManagerID == id {
			return &tasks[i]
		}
	}
	return nil
}
func childCandidateIDs(tasks []ManagerTask, children []string) []string {
	out := []string{}
	for _, id := range children {
		t := findTask(tasks, id)
		if t != nil {
			cid := t.IntegrationCandidateID
			if cid == "" {
				cid = t.CandidateID
			}
			if cid != "" {
				out = append(out, cid)
			}
		}
	}
	sort.Strings(out)
	return out
}
func buildChildSummaries(store *runStore, dir string, tasks []ManagerTask, children []string) []childReport {
	out := make([]childReport, 0, len(children))
	for _, id := range children {
		t := findTask(tasks, id)
		if t == nil {
			continue
		}
		cid := t.IntegrationCandidateID
		if cid == "" {
			cid = t.CandidateID
		}
		candidate, _ := store.readCandidate(dir, cid)
		out = append(out, childReport{ManagerID: id, Summary: t.Summary, Status: t.ReportStatus, Questions: append([]string(nil), t.Questions...), Risks: append([]string(nil), t.Risks...), CandidateDigest: candidate.Digest})
	}
	return out
}
func resolvesConflicts(c candidateData, paths []string) bool {
	for _, p := range paths {
		if _, ok := c.Files[p]; !ok {
			return false
		}
	}
	return true
}
func proposesEvery(proposals []agentexec.CandidateFile, paths []string) bool {
	seen := map[string]bool{}
	for _, p := range proposals {
		seen[p.Path] = true
	}
	for _, p := range paths {
		if !seen[p] {
			return false
		}
	}
	return true
}
func findRootCandidate(tasks []ManagerTask) string {
	var roots []string
	for _, t := range tasks {
		if t.ParentTask == "" {
			if t.IntegrationCandidateID != "" {
				roots = append(roots, t.IntegrationCandidateID)
			} else if t.CandidateID != "" {
				roots = append(roots, t.CandidateID)
			}
		}
	}
	if len(roots) == 1 {
		return roots[0]
	}
	return ""
}
func validateFinalCandidate(host Host, root string, base *Snapshot, c candidateData, plan PlanRecord) error {
	compiled, err := projectForCandidate(host, root, base, c)
	if err != nil {
		return err
	}
	if compiled.Report.ModelDigest != plan.ModelDigest {
		return fmt.Errorf("candidate changed the planned project model")
	}
	if hasErrorFinding(compiled.Report.Findings) {
		return fmt.Errorf("integrated candidate has structural error findings")
	}
	for _, task := range plan.Managers {
		_ = task
	}
	for _, check := range plan.Checks {
		_ = check
	}
	return nil
}

func validateTaskOutcome(outcome string, response TaskResponse) error {
	if response.EscalateTo != "" && outcome != agentexec.OutcomeEscalated {
		return fmt.Errorf("task report escalation does not match executor outcome")
	}
	if response.EscalateTo == "" && outcome != agentexec.OutcomeProposed {
		return fmt.Errorf("task report without escalation does not match executor outcome")
	}
	return nil
}
func recordEscalation(report *RunReport, task ManagerTask, response TaskResponse) error {
	target := escalationTarget(task)
	if response.EscalateTo != target {
		return fmt.Errorf("Manager %s may escalate only to its nearest empowered recipient %q", task.ManagerID, target)
	}
	parts := append(append([]string{}, response.Questions...), response.Risks...)
	if len(parts) == 0 {
		return fmt.Errorf("escalation from %s has no question or risk", task.ManagerID)
	}
	id, err := newID()
	if err != nil {
		return err
	}
	report.Escalations = append(report.Escalations, Escalation{ID: id, FromManager: task.ManagerID, ToManager: target, Question: strings.Join(uniqueSorted(parts), "; "), AffectedTasks: []string{task.ID}, Status: "open"})
	return nil
}

func escalationTarget(task ManagerTask) string {
	if task.ParentTask == "" {
		return "user"
	}
	return task.ParentTask
}
func managerObligations(parent ManagerTask, tasks []ManagerTask, children []string) ([]string, []string, error) {
	questions := append([]string(nil), parent.Questions...)
	risks := append([]string(nil), parent.Risks...)
	questionOwners := map[string]string{}
	riskOwners := map[string]string{}
	for _, q := range parent.Questions {
		questionOwners[q] = parent.ManagerID
	}
	for _, r := range parent.Risks {
		riskOwners[r] = parent.ManagerID
	}
	for _, id := range children {
		child := findTask(tasks, id)
		if child != nil {
			for _, q := range child.Questions {
				if prior, exists := questionOwners[q]; exists && prior != child.ManagerID {
					return nil, nil, fmt.Errorf("question %q is ambiguous between Managers %s and %s", q, prior, child.ManagerID)
				}
				questionOwners[q] = child.ManagerID
			}
			for _, r := range child.Risks {
				if prior, exists := riskOwners[r]; exists && prior != child.ManagerID {
					return nil, nil, fmt.Errorf("risk %q is ambiguous between Managers %s and %s", r, prior, child.ManagerID)
				}
				riskOwners[r] = child.ManagerID
			}
			questions = append(questions, child.Questions...)
			risks = append(risks, child.Risks...)
		}
	}
	return uniqueSorted(questions), uniqueSorted(risks), nil
}
func resolveObligations(outstanding, resolved []string) ([]string, []string, error) {
	want := map[string]bool{}
	for _, v := range outstanding {
		want[v] = true
	}
	seen := map[string]bool{}
	for _, v := range resolved {
		if seen[v] {
			return nil, nil, fmt.Errorf("resolution duplicates %q", v)
		}
		if !want[v] {
			return nil, nil, fmt.Errorf("resolution invents or repeats obligation %q", v)
		}
		seen[v] = true
	}
	var remaining []string
	for _, v := range outstanding {
		if !seen[v] {
			remaining = append(remaining, v)
		}
	}
	return uniqueSorted(resolved), uniqueSorted(remaining), nil
}
func containsAll(have, want []string) bool {
	set := map[string]bool{}
	for _, v := range have {
		set[v] = true
	}
	for _, v := range want {
		if !set[v] {
			return false
		}
	}
	return true
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Status returns the last durable snapshot and its immutable plan.
func Status(root, runID string) (StatusReport, error) {
	var out StatusReport
	s, err := newRunStore(root)
	if err != nil {
		return out, err
	}
	out.Plan, err = s.readPlan(runID)
	if err != nil {
		return out, err
	}
	out.Run, err = s.readLatestState(runID)
	return out, err
}

var _ = json.RawMessage{}
