package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
)

func (e *runExecution) runIntegrationBatch(actions []managerAction) (RunReport, error) {
	for _, action := range actions {
		if action.phase != "integrate" {
			return failRun(e.store, e.report, fmt.Errorf("unknown scheduled Manager action phase %q", action.phase))
		}
		if report, err := e.runIntegrationAction(action); err != nil {
			return report, err
		}
	}
	return RunReport{}, nil
}

func (e *runExecution) runIntegrationAction(action managerAction) (RunReport, error) {
	task := findTask(e.report.Tasks, action.managerID)
	if task == nil {
		return failRun(e.store, e.report, fmt.Errorf("scheduled Manager %s is absent from the run", action.managerID))
	}
	children := activeChildren(e.report.Tasks, task.ManagerID)
	if len(children) == 0 || task.State == "integrated" || task.State == "complete" {
		return RunReport{}, nil
	}
	if e.runtime.Review != nil && hasPendingReworkInSubtree(e.report.Tasks, task.ManagerID) {
		task.State = "integration-deferred"
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		return RunReport{}, nil
	}
	if err := e.ctx.Err(); err != nil {
		return interruptRun(e.store, e.report, err)
	}
	_, hasRecoveredIntegration := e.recoveredResults[taskActionKey(task.ManagerID, "integrate")]
	if e.starts >= e.runtime.Limits.MaxStarts && !hasRecoveredIntegration {
		return blockRun(e.store, e.report, fmt.Errorf("runtime start limit %d exceeded", e.runtime.Limits.MaxStarts))
	}
	if bindErr := ensureWorkingBinding(e.host, e.root, e.plan); bindErr != nil {
		return supersedeExisting(e.store, e.id, bindErr)
	}
	merged, conflicts, err := mergeChildCandidates(e.store, e.dir, e.report.Tasks, *task, children)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	input, err := projectForCandidate(e.host, e.root, e.project.Snapshot, merged)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	childSummaries := buildChildSummaries(e.store, e.dir, e.report.Tasks, e.report.Reviews, children)
	_, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err := e.invokeIntegration(task, children, merged, conflicts, input, childSummaries)
	if err != nil {
		return report, err
	}
	newLocal, err := newLocalResponseObligations(task.ManagerID, remainingObligations, parsed.Questions, parsed.Risks)
	if err != nil {
		return blockRun(e.store, e.report, fmt.Errorf("Manager %s obligation provenance: %w", task.ManagerID, err))
	}
	forwarded := appendForwardedObligations(remainingObligations, task.ManagerID)
	if err := applyObligationResolutions(&e.report, resolvedObligations); err != nil {
		return blockRun(e.store, e.report, fmt.Errorf("Manager %s obligation resolution: %w", task.ManagerID, err))
	}
	task.Obligations = append(forwarded, newLocal...)
	if len(remainingQ)+len(remainingR) > 0 {
		if err := recordEscalation(&e.report, *task, TaskResponse{Summary: parsed.Summary, Questions: remainingQ, Risks: remainingR, EscalateTo: parsed.EscalateTo}); err != nil {
			return blockRun(e.store, e.report, err)
		}
	}
	task.ReworkRequests = append([]ReworkRequest(nil), parsed.ReworkRequests...)
	if len(remainingQ)+len(remainingR) > 0 {
		markObligationEscalationsEscalated(&e.report, task.ManagerID, task.Obligations)
	}
	task.Questions, task.Risks = uniqueSorted(remainingQ), uniqueSorted(remainingR)
	if len(remainingQ) == 0 && len(remainingR) == 0 {
		task.ReportStatus = "complete"
	} else {
		task.ReportStatus = "partial"
	}
	resolved.ID, err = newID()
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	resolved.Parents = []string{merged.ID}
	if err := e.store.writeCandidate(e.dir, resolved); err != nil {
		return failRun(e.store, e.report, err)
	}
	resolved, err = e.store.readCandidate(e.dir, resolved.ID)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	task.State, task.IntegrationReportID, task.IntegrationCandidateID = "integrated", invocation.ReportID, resolved.ID
	task.IntegratedPaths = unionPaths(task.IntegratedPaths, changedCandidatePaths(merged, resolved))
	task.Summary = parsed.Summary
	e.report.Candidate = candidateRef(resolved, false)
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	e.refreshStarts()
	if e.runtime.Review == nil {
		return RunReport{}, nil
	}
	if len(task.ReworkRequests) > 0 {
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "", "", 0
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		return RunReport{}, nil
	}
	candidateProject, compileErr := finalProjectForCandidate(e.host, e.root, e.project, resolved)
	if compileErr != nil {
		return failRun(e.store, e.report, compileErr)
	}
	if !phaseReviewRequired(candidateProject, *task, e.report.Tasks, "integrate") {
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "not-required", resolved.ID, 0
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		return RunReport{}, nil
	}
	return e.reviewIntegration(task, candidateProject, resolved)
}

func (e *runExecution) invokeIntegration(task *ManagerTask, children []string, merged candidateData, conflicts []string, input *projectvalue.Project, childSummaries []childReport) (agentexec.RunResult, InvocationLog, TaskResponse, candidateData, []string, []string, []Obligation, []Obligation, RunReport, error) {
	var proposal agentexec.RunResult
	var invocation InvocationLog
	var parsed TaskResponse
	var resolved candidateData
	var remainingQ, remainingR []string
	var resolvedObligations, remainingObligations []Obligation
	for {
		var invokeErr error
		fromRecovery := false
		if recovered, ok := e.recoveredResults[taskActionKey(task.ManagerID, "integrate")]; ok {
			fromRecovery = true
			proposal, invocation, invokeErr = recovered.result, recovered.log, recovered.err
			delete(e.recoveredResults, taskActionKey(task.ManagerID, "integrate"))
		} else {
			if e.starts >= e.runtime.Limits.MaxStarts {
				report, err := blockRun(e.store, e.report, fmt.Errorf("runtime start limit %d exceeded before %s integration repair", e.runtime.Limits.MaxStarts, task.ManagerID))
				return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
			}
			if e.spent >= e.runtime.Limits.MaxCostMicros {
				report, err := blockRun(e.store, e.report, fmt.Errorf("estimated cost limit reached before %s integration repair", task.ManagerID))
				return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
			}
			task.State, task.RepairPhase = "integrating", "integrate"
			task.IntegrationAttempts++
			task.Attempts++
			if err := persistState(e.store, &e.report); err != nil {
				return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, RunReport{}, err
			}
			repairRound, repairChecks := repairContext(e.report, *task)
			proposal, invocation, invokeErr = invokeManager(e.ctx, e.host, e.invoker, e.root, e.runtime, e.plan, input, *task, "integrate", children, conflicts, childSummaries, task.RepairDiagnostic, repairRound, repairChecks, e.starts)
			e.starts++
		}
		if invokeErr != nil {
			if !fromRecovery && appendInvocationReceipt(&e.report, invocation) {
				e.spent = totalCost(e.report.Invocations)
			}
			task.State = "uncertain"
			if errors.Is(invokeErr, ErrStale) {
				if err := persistState(e.store, &e.report); err != nil {
					return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, RunReport{}, err
				}
				report, err := supersedeExisting(e.store, e.id, invokeErr)
				return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
			}
			_ = persistState(e.store, &e.report)
			e.refreshStarts()
			if e.ctx.Err() != nil {
				report, err := interruptRun(e.store, e.report, e.ctx.Err())
				return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
			}
			report, err := failRun(e.store, e.report, fmt.Errorf("manager %s integration: %w", task.ManagerID, invokeErr))
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
		}
		if !fromRecovery {
			e.report.Invocations = append(e.report.Invocations, invocation)
		}
		e.spent = totalCost(e.report.Invocations)
		if costExceedsLimit(e.report.Invocations, e.runtime.Limits.MaxCostMicros) {
			report, err := blockRun(e.store, e.report, fmt.Errorf("estimated cost limit exceeded"))
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
		}
		e.refreshStarts()
		validationErr := validateIntegrationProposal(task, children, merged, conflicts, input, e.runtime.Limits, &e.report, proposal, &parsed, &resolved, &remainingQ, &remainingR, &resolvedObligations, &remainingObligations)
		if validationErr == nil {
			task.RepairPhase, task.RepairDiagnostic = "", ""
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, RunReport{}, nil
		}
		var terminal terminalTaskResponseError
		if errors.As(validationErr, &terminal) {
			var report RunReport
			var err error
			if terminal.blocked {
				report, err = blockRun(e.store, e.report, terminal.err)
			} else {
				report, err = failRun(e.store, e.report, terminal.err)
			}
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(validationErr)
		if task.IntegrationAttempts > e.runtime.Limits.MaxRetries {
			report, err := failRun(e.store, e.report, fmt.Errorf("manager %s integration response remained invalid after %d attempt(s): %s", task.ManagerID, task.IntegrationAttempts, task.RepairDiagnostic))
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, report, err
		}
		task.State = "integration-retry-ready"
		if err := persistState(e.store, &e.report); err != nil {
			return proposal, invocation, parsed, resolved, remainingQ, remainingR, resolvedObligations, remainingObligations, RunReport{}, err
		}
	}
}

func validateIntegrationProposal(task *ManagerTask, children []string, merged candidateData, conflicts []string, input *projectvalue.Project, limits projectconfig.Limits, report *RunReport, proposal agentexec.RunResult, parsed *TaskResponse, resolved *candidateData, remainingQ, remainingR *[]string, resolvedObligations, remainingObligations *[]Obligation) error {
	var decodeErr error
	*parsed, decodeErr = decodeTaskResponse(proposal.Response.ReportJSON, "integrate", children)
	if decodeErr != nil {
		return decodeErr
	}
	if parsed.Status == "blocked" || parsed.Status == "failed" || parsed.Status == "no-op" {
		return terminalTaskResponseError{blocked: true, err: fmt.Errorf("manager %s integration is %s: %s", task.ManagerID, parsed.Status, parsed.Summary)}
	}
	if outcomeErr := validateTaskOutcome(proposal.Response.Outcome, *parsed); outcomeErr != nil {
		return outcomeErr
	}
	if parsed.EscalateTo != "" && parsed.EscalateTo != escalationTarget(*task) {
		return fmt.Errorf("Manager %s may escalate only to its nearest empowered recipient %q", task.ManagerID, escalationTarget(*task))
	}
	candidate, candidateErr := applyAgentCandidate(merged, proposal, input.Config, input.Report, *task, "integrate", conflicts, limits, input.Snapshot)
	if candidateErr != nil {
		return candidateErr
	}
	if len(conflicts) > 0 && !proposesEvery(agentCandidatePaths(proposal), conflicts) {
		return fmt.Errorf("manager %s did not resolve child path conflict(s): %s", task.ManagerID, strings.Join(conflicts, ", "))
	}
	outstanding, obligationErr := managerObligationRecords(*task, report.Tasks, children)
	if obligationErr != nil {
		return terminalTaskResponseError{blocked: true, err: fmt.Errorf("manager %s obligations: %w", task.ManagerID, obligationErr)}
	}
	*resolvedObligations, *remainingObligations, obligationErr = resolveObligationRecords(outstanding, parsed.ResolvedQuestions, parsed.ResolvedRisks)
	if obligationErr != nil {
		return fmt.Errorf("manager %s obligation resolution: %w", task.ManagerID, obligationErr)
	}
	*remainingQ, *remainingR = obligationTexts(*remainingObligations)
	*remainingQ = append(*remainingQ, parsed.Questions...)
	*remainingR = append(*remainingR, parsed.Risks...)
	if len(*remainingQ)+len(*remainingR) > 0 {
		if parsed.Status != "partial" || parsed.EscalateTo != escalationTarget(*task) {
			return fmt.Errorf("manager %s has unresolved obligations that did not escalate to its nearest parent", task.ManagerID)
		}
	} else if parsed.Status != "complete" {
		return fmt.Errorf("manager %s integration is partial without unresolved obligations", task.ManagerID)
	}
	if parsed.EscalateTo != "" && len(parsed.Questions)+len(parsed.Risks) == 0 {
		return fmt.Errorf("Manager %s escalation requires an actionable question or risk", task.ManagerID)
	}
	*resolved = candidate
	return nil
}

func (e *runExecution) reviewIntegration(task *ManagerTask, candidateProject *projectvalue.Project, candidate candidateData) (RunReport, error) {
	reviewRound := reviewCount(e.report.Reviews, task.ManagerID, "integrate") + 1
	if reviewRound > e.runtime.Review.MaxRounds {
		return blockRun(e.store, e.report, fmt.Errorf("Manager %s exhausted the cumulative integration review round limit", task.ManagerID))
	}
	task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "invoking", candidate.ID, reviewRound
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	if e.starts >= e.runtime.Limits.MaxStarts || e.spent >= e.runtime.Limits.MaxCostMicros {
		return blockRun(e.store, e.report, fmt.Errorf("review for %s exceeds the cumulative run budget", task.ManagerID))
	}
	startedIndex := -1
	review, reviewLog, reviewErr := invokeReviewer(e.ctx, e.host, e.invoker, e.root, e.plan, e.runtime, candidateProject, *task, "integrate", reviewRound, candidate, func(started InvocationLog) error {
		e.report.Invocations = append(e.report.Invocations, started)
		startedIndex = len(e.report.Invocations) - 1
		return persistState(e.store, &e.report)
	}, e.report)
	e.starts++
	if startedIndex >= 0 {
		e.report.Invocations[startedIndex] = reviewLog
		if reviewErr == nil {
			e.report.Reviews = append(e.report.Reviews, review)
		}
		e.spent = totalCost(e.report.Invocations)
		if costExceedsLimit(e.report.Invocations, e.runtime.Limits.MaxCostMicros) {
			return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit exceeded during integration review"))
		}
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
	}
	if reviewErr != nil {
		task.State, task.ReviewStatus = "uncertain", "uncertain"
		_ = persistState(e.store, &e.report)
		if e.ctx.Err() != nil {
			return interruptRun(e.store, e.report, e.ctx.Err())
		}
		return failRun(e.store, e.report, fmt.Errorf("manager %s integration review: %w", task.ManagerID, reviewErr))
	}
	task.ReviewStatus = review.Outcome
	if review.Outcome != "pass" {
		var findings []string
		for _, finding := range review.Findings {
			findings = append(findings, finding.Path+": "+finding.Expectation+" ("+finding.Grounding+")")
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("integration review findings: %s", strings.Join(findings, "; ")))
		requests, reworkErr := reintegrateAfterRework(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, task, &e.starts, &e.spent)
		if reworkErr != nil {
			return blockRun(e.store, e.report, reworkErr)
		}
		task.ReworkRequests = append(task.ReworkRequests, requests...)
	}
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	return RunReport{}, nil
}
