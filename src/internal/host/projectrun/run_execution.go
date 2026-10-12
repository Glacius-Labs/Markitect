package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
)

// runExecution is the invocation-local scheduler state. Its report is the
// existing mutable run state; persistence remains owned by runStore.
type runExecution struct {
	ctx                     context.Context
	host                    Host
	invoker                 Invoker
	root                    string
	id                      string
	store                   *runStore
	dir                     string
	plan                    PlanRecord
	runtime                 projectconfig.Runtime
	project                 *projectvalue.Project
	report                  RunReport
	pendingRecoveries       []pendingNativeRecovery
	recoveredReintegrations []string
	roleBudget              *RoleStartBudget
	starts                  int
	spent                   int64
	baseCandidateID         string
	baseCandidate           candidateData
	boundProject            *projectvalue.Project
	dependencies            map[string][]string
	recoveredResults        map[string]managerInvocationResult
}

// execute keeps the current method budget and cancellation window around all
// post-admission recovery, scheduling, and closure work.
func (e *runExecution) execute() (RunReport, error) {
	deadline := e.report.StartedAt.Add(time.Duration(e.runtime.Limits.MaxDuration))
	if !time.Now().Before(deadline) {
		return blockRun(e.store, e.report, fmt.Errorf("total runtime duration limit exceeded"))
	}
	boundedCtx, cancel := context.WithDeadline(e.ctx, deadline)
	defer cancel()
	e.ctx = boundedCtx
	if report, err := e.bindRuntimeBudget(); err != nil {
		return report, err
	}
	if report, err := e.loadBoundCandidate(); err != nil {
		return report, err
	}
	if report, err := e.recoverNativeInvocations(); err != nil {
		return report, err
	}
	if report, err := e.reintegrateRecoveredManagers(); err != nil {
		return report, err
	}
	if report, err := e.scheduleManagers(); err != nil {
		return report, err
	}
	return e.closeRun()
}

func (e *runExecution) bindRuntimeBudget() (RunReport, error) {
	invoker, roleBudget, err := bindRunHelperBudget(e.invoker, e.host, e.runtime.Limits, e.store, &e.report)
	if err != nil {
		return failRun(e.store, e.report, fmt.Errorf("bind runtime helper budget: %w", err))
	}
	e.invoker, e.roleBudget = invoker, roleBudget
	e.refreshStarts()
	e.spent = totalCost(e.report.Invocations)
	return RunReport{}, nil
}

func (e *runExecution) refreshStarts() {
	e.starts = e.roleBudget.Accounting().ObservedTotal + len(e.report.Checks)
}

func (e *runExecution) loadBoundCandidate() (RunReport, error) {
	e.baseCandidateID = e.plan.InitialCandidateID
	if e.report.ActiveRepairCandidateID != "" {
		e.baseCandidateID = e.report.ActiveRepairCandidateID
	}
	baseCandidate, err := e.store.readCandidate(e.dir, e.baseCandidateID)
	if err != nil {
		return RunReport{}, err
	}
	e.baseCandidate = baseCandidate
	modelCandidate, err := snapshotWithCandidate(e.project.Snapshot, baseCandidate)
	if err != nil {
		return RunReport{}, err
	}
	boundProject, err := e.host.FromSnapshot(e.root, modelCandidate)
	if err != nil {
		return RunReport{}, fmt.Errorf("compile planned candidate model: %w", err)
	}
	if boundProject.Report.ModelDigest != e.plan.ModelDigest || hasErrorFinding(boundProject.Report.Findings) || (e.report.ActiveRepairCandidateID == "" && boundProject.Report.Digest != e.plan.ReportDigest) {
		return supersedeExisting(e.store, e.id, ErrStale)
	}
	e.boundProject = boundProject
	dependencies, err := managerDependencies(boundProject.Report, e.report.Tasks)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	e.dependencies = dependencies
	e.recoveredResults = make(map[string]managerInvocationResult, len(e.pendingRecoveries))
	return RunReport{}, nil
}

func (e *runExecution) recoverNativeInvocations() (RunReport, error) {
	for _, pending := range e.pendingRecoveries {
		task := findTask(e.report.Tasks, pending.managerID)
		if task == nil {
			return blockRun(e.store, e.report, fmt.Errorf("native recovery task %s is absent from the run", pending.managerID))
		}
		agent := e.runtime.Agents[task.ManagerID]
		config, configErr := agent.AgentConfig()
		if configErr != nil {
			return blockRun(e.store, e.report, fmt.Errorf("native recovery configuration for %s is invalid", task.ManagerID))
		}
		input, children, childReports, conflicts, inputErr := recoveryManagerInput(e.store, e.dir, e.host, e.root, e.project.Snapshot, e.boundProject.Report, e.report, e.plan, *task, pending.phase, e.dependencies[task.ManagerID])
		if inputErr != nil {
			return blockRun(e.store, e.report, fmt.Errorf("native recovery input for %s could not be reconstructed", task.ManagerID))
		}
		repairRound, repairChecks := repairContext(e.report, *task)
		request, requestErr := managerInvocationRequest(e.host, e.root, e.runtime, e.plan, input, *task, pending.phase, children, conflicts, childReports, task.RepairDiagnostic, repairRound, repairChecks)
		if requestErr != nil {
			return blockRun(e.store, e.report, fmt.Errorf("native recovery request for %s could not be reconstructed", task.ManagerID))
		}
		binding, bindingErr := originalNativeRecoveryBinding(e.report.ID, e.report.Invocations, task.ID, agentexec.RoleExecutor, pending.phase, request)
		if bindingErr != nil {
			return blockRun(e.store, e.report, fmt.Errorf("native recovery for %s is blocked: %w", task.ManagerID, bindingErr))
		}
		result, found, recoverErr := RecoverProjectAgent(e.ctx, e.host, e.invoker, e.root, pending.taskID, config, e.runtime.Limits, binding, request)
		if recoverErr != nil || !found {
			if result.Receipt.RunID != "" {
				upsertRecoveredInvocation(&e.report, invocationLogForResult(*task, pending.phase, result, agent))
			}
			if current := findTask(e.report.Tasks, task.ManagerID); current != nil {
				current.State = "uncertain"
			}
			cause := fmt.Errorf("native invocation for %s could not be safely recovered", task.ManagerID)
			return blockRun(e.store, e.report, errors.Join(cause, recoverErr))
		}
		invocation := invocationLogForResult(*task, pending.phase, result, agent)
		upsertRecoveredInvocation(&e.report, invocation)
		e.recoveredResults[taskActionKey(task.ManagerID, pending.phase)] = managerInvocationResult{managerID: task.ManagerID, result: result, log: invocation}
		if pending.phase == "integrate" {
			task.State = "integration-retry-ready"
		} else {
			task.State = "work-retry-ready"
		}
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
	}
	e.refreshStarts()
	e.spent = totalCost(e.report.Invocations)
	return RunReport{}, nil
}

func (e *runExecution) reintegrateRecoveredManagers() (RunReport, error) {
	// Failed recovered integration reviews are reworked before scheduling.
	for _, managerID := range e.recoveredReintegrations {
		task := findTask(e.report.Tasks, managerID)
		if task == nil {
			return failRun(e.store, e.report, fmt.Errorf("recovered Manager %s is absent from the run", managerID))
		}
		requests, err := reintegrateAfterRework(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, task, &e.starts, &e.spent)
		if err != nil {
			return blockRun(e.store, e.report, err)
		}
		task.ReworkRequests = append(task.ReworkRequests, requests...)
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
	}
	return RunReport{}, nil
}

func (e *runExecution) scheduleManagers() (RunReport, error) {
	// Work and bottom-up integration are chosen from durable task state each
	// turn so they continue to interleave as dependency trees become ready.
	for {
		actions, err := readyManagerActions(e.report.Tasks, e.dependencies, e.runtime.Limits.MaxParallel)
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		if len(actions) == 0 {
			return RunReport{}, nil
		}
		if actions[0].phase == "work" {
			if report, err := e.runWorkBatch(actions); err != nil {
				return report, err
			}
			continue
		}
		if report, err := e.runIntegrationBatch(actions); err != nil {
			return report, err
		}
	}
}
