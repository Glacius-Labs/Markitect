package projectrun

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
)

func (e *runExecution) runWorkBatch(actions []managerAction) (RunReport, error) {
	remainingStarts := e.runtime.Limits.MaxStarts - e.starts
	freshActions := 0
	limited := make([]managerAction, 0, len(actions))
	for _, action := range actions {
		if _, recovered := e.recoveredResults[taskActionKey(action.managerID, "work")]; recovered {
			limited = append(limited, action)
			continue
		}
		if freshActions >= remainingStarts {
			continue
		}
		freshActions++
		limited = append(limited, action)
	}
	actions = limited
	if len(actions) == 0 {
		return blockRun(e.store, e.report, fmt.Errorf("runtime start limit %d exceeded", e.runtime.Limits.MaxStarts))
	}
	if freshActions > 0 && e.spent >= e.runtime.Limits.MaxCostMicros {
		return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit reached before Manager work"))
	}
	if bindErr := ensureWorkingBinding(e.host, e.root, e.plan); bindErr != nil {
		return supersedeExisting(e.store, e.id, bindErr)
	}
	prepared := make(map[string]preparedWorkAction, len(actions))
	calls := make([]managerInvocationCall, 0, len(actions))
	for _, action := range actions {
		if recovered, ok := e.recoveredResults[taskActionKey(action.managerID, "work")]; ok {
			task := findTask(e.report.Tasks, action.managerID)
			if task == nil {
				return failRun(e.store, e.report, fmt.Errorf("recovered Manager %s is absent from the run", action.managerID))
			}
			current, readErr := workCandidate(e.store, e.dir, e.report, *task, e.baseCandidateID)
			if readErr != nil {
				return failRun(e.store, e.report, readErr)
			}
			input, children, _, _, inputErr := recoveryManagerInput(e.store, e.dir, e.host, e.root, e.project.Snapshot, e.boundProject.Report, e.report, e.plan, *task, "work", e.dependencies[task.ManagerID])
			if inputErr != nil {
				return failRun(e.store, e.report, inputErr)
			}
			prepared[action.managerID] = preparedWorkAction{current: current, input: input, children: children, result: recovered}
			continue
		}
		task := findTask(e.report.Tasks, action.managerID)
		if task == nil {
			return failRun(e.store, e.report, fmt.Errorf("scheduled Manager %s is absent from the run", action.managerID))
		}
		if err := e.ctx.Err(); err != nil {
			return interruptRun(e.store, e.report, err)
		}
		current, readErr := workCandidate(e.store, e.dir, e.report, *task, e.baseCandidateID)
		if readErr != nil {
			return failRun(e.store, e.report, readErr)
		}
		inputCandidate, mergeErr := mergeDependencyInputs(e.store, e.dir, e.report.Tasks, current, e.dependencies[task.ManagerID])
		if mergeErr != nil {
			return failRun(e.store, e.report, mergeErr)
		}
		input, projectErr := projectForCandidate(e.host, e.root, e.project.Snapshot, inputCandidate)
		if projectErr != nil {
			return failRun(e.store, e.report, projectErr)
		}
		children := activeChildren(e.report.Tasks, task.ManagerID)
		task.State, task.RepairPhase = "invoking", "work"
		task.WorkAttempts++
		task.Attempts++
		start := e.starts
		e.starts++
		taskCopy := *task
		repairRound, repairChecks := repairContext(e.report, taskCopy)
		prepared[action.managerID] = preparedWorkAction{current: current, input: input, children: children}
		calls = append(calls, managerInvocationCall{managerID: action.managerID, invoke: func(callCtx context.Context) (agentexec.RunResult, InvocationLog, error) {
			return invokeManager(callCtx, e.host, e.invoker, e.root, e.runtime, e.plan, input, taskCopy, "work", children, nil, nil,
				taskCopy.RepairDiagnostic, repairRound, repairChecks, start)
		}})
	}
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	results := invokeManagerBatch(e.ctx, calls)
	for _, result := range results {
		prepared[result.managerID] = preparedWorkAction{current: prepared[result.managerID].current, input: prepared[result.managerID].input, children: prepared[result.managerID].children, result: result}
		appendInvocationReceipt(&e.report, result.log)
		if result.err != nil {
			if task := findTask(e.report.Tasks, result.managerID); task != nil {
				task.State = "uncertain"
			}
		}
	}
	e.spent = totalCost(e.report.Invocations)
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	e.refreshStarts()
	if dispatchErr := batchInvocationError(results); dispatchErr != nil {
		return failRun(e.store, e.report, fmt.Errorf("parallel Manager work dispatch failed: %w", dispatchErr))
	}
	for _, action := range actions {
		if report, err := e.runWorkAction(action, prepared[action.managerID]); err != nil {
			return report, err
		}
	}
	return RunReport{}, nil
}

func (e *runExecution) runWorkAction(action managerAction, preparedAction preparedWorkAction) (RunReport, error) {
	task := findTask(e.report.Tasks, action.managerID)
	if task == nil {
		return failRun(e.store, e.report, fmt.Errorf("scheduled Manager %s is absent from the run", action.managerID))
	}
	if task.State == "worked" || task.State == "integrated" || task.State == "complete" || task.State == "integration-retry-ready" {
		return RunReport{}, nil
	}
	if err := e.ctx.Err(); err != nil {
		return interruptRun(e.store, e.report, err)
	}
	current, input, children := preparedAction.current, preparedAction.input, preparedAction.children
	reviewRounds, reviewRoundBase := 1, 0
	if e.runtime.Review != nil {
		reviewRounds = e.runtime.Review.MaxRounds
		reviewRoundBase = reviewCount(e.report.Reviews, task.ManagerID, "work")
		if reviewRoundBase >= reviewRounds {
			return blockRun(e.store, e.report, fmt.Errorf("Manager %s exhausted the cumulative work review round limit", task.ManagerID))
		}
	}
	batchResultPending := true
	var err error
	for reviewRound := reviewRoundBase + 1; reviewRound <= reviewRounds; reviewRound++ {
		var proposal agentexec.RunResult
		var invocation InvocationLog
		var parsed TaskResponse
		var candidate candidateData
		for {
			var invokeErr error
			fromBatch := batchResultPending
			if batchResultPending {
				dispatched := preparedAction.result
				proposal, invocation, invokeErr = dispatched.result, dispatched.log, dispatched.err
				delete(e.recoveredResults, taskActionKey(task.ManagerID, "work"))
				batchResultPending = false
			} else {
				if e.starts >= e.runtime.Limits.MaxStarts {
					return blockRun(e.store, e.report, fmt.Errorf("runtime start limit %d exceeded before %s work repair", e.runtime.Limits.MaxStarts, task.ManagerID))
				}
				if e.spent >= e.runtime.Limits.MaxCostMicros {
					return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit reached before %s work repair", task.ManagerID))
				}
				task.State, task.RepairPhase = "invoking", "work"
				task.WorkAttempts++
				task.Attempts++
				if err := persistState(e.store, &e.report); err != nil {
					return RunReport{}, err
				}
				e.refreshStarts()
				repairRound, repairChecks := repairContext(e.report, *task)
				proposal, invocation, invokeErr = invokeManager(e.ctx, e.host, e.invoker, e.root, e.runtime, e.plan, input, *task, "work", children, nil, nil, task.RepairDiagnostic, repairRound, repairChecks, e.starts)
				e.starts++
			}
			if invokeErr != nil {
				if !fromBatch && appendInvocationReceipt(&e.report, invocation) {
					e.spent = totalCost(e.report.Invocations)
				}
				task.State = "uncertain"
				if errors.Is(invokeErr, ErrStale) {
					if err := persistState(e.store, &e.report); err != nil {
						return RunReport{}, err
					}
					return supersedeExisting(e.store, e.id, invokeErr)
				}
				_ = persistState(e.store, &e.report)
				e.refreshStarts()
				if e.ctx.Err() != nil {
					return interruptRun(e.store, e.report, e.ctx.Err())
				}
				return failRun(e.store, e.report, fmt.Errorf("manager %s work: %w", task.ManagerID, invokeErr))
			}
			if !fromBatch {
				e.report.Invocations = append(e.report.Invocations, invocation)
				e.spent = totalCost(e.report.Invocations)
				if err := persistState(e.store, &e.report); err != nil {
					return RunReport{}, err
				}
				e.refreshStarts()
			}
			if costExceedsLimit(e.report.Invocations, e.runtime.Limits.MaxCostMicros) {
				return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit exceeded"))
			}
			parseErr := validateWorkResponse(task, children, current, input, e.runtime.Limits, proposal, &parsed, &candidate)
			if parseErr == nil {
				task.RepairPhase, task.RepairDiagnostic = "", ""
				break
			}
			var terminal terminalTaskResponseError
			if errors.As(parseErr, &terminal) {
				if terminal.blocked {
					return blockRun(e.store, e.report, terminal.err)
				}
				return failRun(e.store, e.report, terminal.err)
			}
			task.RepairDiagnostic = boundedRepairDiagnostic(parseErr)
			if task.WorkAttempts > e.runtime.Limits.MaxRetries {
				return failRun(e.store, e.report, fmt.Errorf("manager %s work response remained invalid after %d attempt(s): %s", task.ManagerID, task.WorkAttempts, task.RepairDiagnostic))
			}
			task.State = "work-retry-ready"
			if err := persistState(e.store, &e.report); err != nil {
				return RunReport{}, err
			}
		}
		task.Obligations, err = newLocalObligations(task.ManagerID, parsed.Questions, parsed.Risks)
		if err != nil {
			return blockRun(e.store, e.report, fmt.Errorf("Manager %s obligation provenance: %w", task.ManagerID, err))
		}
		if parsed.EscalateTo != "" {
			if err := recordEscalation(&e.report, *task, parsed); err != nil {
				return blockRun(e.store, e.report, err)
			}
		}
		for _, delegation := range parsed.Delegations {
			child := findTask(e.report.Tasks, delegation.ManagerID)
			if child == nil {
				return failRun(e.store, e.report, fmt.Errorf("delegation target %s is not in the active plan", delegation.ManagerID))
			}
			child.Goal = delegation.Goal
		}
		candidate.ID, err = newID()
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		candidate.Parents = []string{current.ID}
		if err := e.store.writeCandidate(e.dir, candidate); err != nil {
			return failRun(e.store, e.report, err)
		}
		candidate, err = e.store.readCandidate(e.dir, candidate.ID)
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		task.State, task.ReportID, task.CandidateID = "worked", invocation.ReportID, candidate.ID
		task.WrittenPaths = unionPaths(task.WrittenPaths, agentCandidatePaths(proposal))
		task.Summary, task.Questions, task.Risks, task.Delegations, task.ReportStatus = parsed.Summary, parsed.Questions, parsed.Risks, parsed.Delegations, parsed.Status
		e.report.Candidate = candidateRef(candidate, false)
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		if e.runtime.Review == nil {
			break
		}
		candidateProject, compileErr := finalProjectForCandidate(e.host, e.root, e.project, candidate)
		if compileErr != nil {
			return failRun(e.store, e.report, compileErr)
		}
		if !reviewRequired(candidateProject, *task) {
			task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "not-required", candidate.ID, 0
			if err := persistState(e.store, &e.report); err != nil {
				return RunReport{}, err
			}
			break
		}
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "invoking", candidate.ID, reviewRound
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		if e.starts >= e.runtime.Limits.MaxStarts || e.spent >= e.runtime.Limits.MaxCostMicros {
			return blockRun(e.store, e.report, fmt.Errorf("review for %s exceeds the cumulative run budget", task.ManagerID))
		}
		startedIndex := -1
		review, reviewLog, reviewErr := invokeReviewer(e.ctx, e.host, e.invoker, e.root, e.plan, e.runtime, candidateProject, *task, "work", reviewRound, candidate, func(started InvocationLog) error {
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
				return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit exceeded during review"))
			}
			if err := persistState(e.store, &e.report); err != nil {
				return RunReport{}, err
			}
			e.refreshStarts()
		}
		if reviewErr != nil {
			task.State, task.ReviewStatus = "uncertain", "uncertain"
			_ = persistState(e.store, &e.report)
			if e.ctx.Err() != nil {
				return interruptRun(e.store, e.report, e.ctx.Err())
			}
			return failRun(e.store, e.report, fmt.Errorf("manager %s review: %w", task.ManagerID, reviewErr))
		}
		task.ReviewStatus = review.Outcome
		if review.Outcome == "pass" {
			if err := persistState(e.store, &e.report); err != nil {
				return RunReport{}, err
			}
			break
		}
		if reviewRound == reviewRounds {
			return blockRun(e.store, e.report, fmt.Errorf("manager %s reached the configured review round limit", task.ManagerID))
		}
		var diagnostics []string
		for _, finding := range review.Findings {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %s (%s)", finding.Path, finding.Expectation, finding.Grounding))
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("independent review findings: %s", strings.Join(diagnostics, "; ")))
		task.ReviewStatus = "rework-requested"
		current = candidate
		reworkInputCandidate, mergeErr := mergeDependencyInputs(e.store, e.dir, e.report.Tasks, current, e.dependencies[task.ManagerID])
		if mergeErr != nil {
			return failRun(e.store, e.report, mergeErr)
		}
		input, err = projectForCandidate(e.host, e.root, e.project.Snapshot, reworkInputCandidate)
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		task.State = "review-rework-ready"
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
	}
	return RunReport{}, nil
}

func validateWorkResponse(task *ManagerTask, children []string, current candidateData, input *Project, limits projectconfig.Limits, proposal agentexec.RunResult, parsed *TaskResponse, candidate *candidateData) error {
	decoded, err := decodeTaskResponse(proposal.Response.ReportJSON, "work", children)
	if err != nil {
		return err
	}
	*parsed = decoded
	if parsed.Status == "blocked" {
		return terminalTaskResponseError{blocked: true, err: fmt.Errorf("manager %s reported blocked: %s", task.ManagerID, parsed.Summary)}
	}
	if parsed.Status == "failed" {
		return terminalTaskResponseError{err: fmt.Errorf("manager %s reported failed: %s", task.ManagerID, parsed.Summary)}
	}
	if outcomeErr := validateTaskOutcome(proposal.Response.Outcome, *parsed); outcomeErr != nil {
		return outcomeErr
	}
	if parsed.Status == "complete" && (len(parsed.Questions) > 0 || len(parsed.Risks) > 0) {
		return fmt.Errorf("manager %s reported complete with unresolved questions or risks", task.ManagerID)
	}
	if parsed.Status == "partial" && len(parsed.Questions) == 0 && len(parsed.Risks) == 0 {
		return fmt.Errorf("manager %s reported partial without an actionable question or risk", task.ManagerID)
	}
	if parsed.EscalateTo != "" && parsed.EscalateTo != escalationTarget(*task) {
		return fmt.Errorf("Manager %s may escalate only to its nearest empowered recipient %q", task.ManagerID, escalationTarget(*task))
	}
	if parsed.Status == "no-op" && (len(proposal.Response.CandidateFiles) > 0 || proposal.Delta != nil && len(proposal.Delta.Changes) > 0) {
		return fmt.Errorf("manager %s claimed no-op while proposing files", task.ManagerID)
	}
	result, err := applyAgentCandidate(current, proposal, input.Config, input.Report, *task, "work", nil, limits, input.Snapshot)
	if err != nil {
		return err
	}
	*candidate = result
	return nil
}
