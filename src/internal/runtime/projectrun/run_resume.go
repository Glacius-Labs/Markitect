package projectrun

import (
	"errors"
	"fmt"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

// resumeExisting reconciles incomplete durable states before execute creates
// the method deadline, preserving the established native recovery boundary.
func (e *runExecution) resumeExisting() (bool, RunReport, error) {
	if e.report.Status == StatusVerified || e.report.Status == StatusApplied {
		return false, e.report, nil
	}
	if e.report.Status == StatusSuperseded {
		return false, e.report, ErrStale
	}
	if e.report.Status != StatusInterrupted && e.report.Status != StatusRunning && e.report.Status != StatusFailed && e.report.Status != StatusBlocked && e.report.Status != StatusAwaitingExternal {
		return false, e.report, fmt.Errorf("run status %s cannot be resumed", e.report.Status)
	}
	for _, round := range e.report.ManagerReworkRounds {
		if round.Status != "running" {
			continue
		}
		e.report.Status = StatusBlocked
		e.report.Findings = append(e.report.Findings, fmt.Sprintf("manager-directed rework round %d was unfinished at the resume boundary; it will not be replayed automatically", round.Number))
		if err := persistState(e.store, &e.report); err != nil {
			return false, RunReport{}, err
		}
		return false, e.report, fmt.Errorf("unfinished manager-directed rework round %d requires a new plan", round.Number)
	}
	pendingReviewRecovered := false
	for _, task := range e.report.Tasks {
		if task.ReviewStatus == "invoking" || task.ReviewStatus == "uncertain" || task.ReviewStatus == StatusAwaitingExternal {
			reintegrate, recoveryErr := recoverPendingNativeReview(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, task.ManagerID)
			if recoveryErr == nil {
				pendingReviewRecovered = true
				if reintegrate {
					e.recoveredReintegrations = append(e.recoveredReintegrations, task.ManagerID)
				}
				continue
			}
			if errors.Is(recoveryErr, ErrAwaitingExternal) {
				report, err := awaitExternalRun(e.store, e.report)
				return false, report, err
			}
			var unrouted unroutedReviewError
			if errors.As(recoveryErr, &unrouted) {
				report, err := blockRun(e.store, e.report, fmt.Errorf("recovered final review for %s failed and its findings have no rework route: %w", task.ManagerID, unrouted.cause))
				return false, report, err
			}
			if current := findTask(e.report.Tasks, task.ManagerID); current != nil {
				current.State = "uncertain"
			}
			e.report.Status = StatusBlocked
			e.report.Findings = append(e.report.Findings, "prior native reviewer outcome is uncertain for "+task.ManagerID+"; exact-candidate recovery failed and it will not be replayed automatically: "+boundedRepairDiagnostic(recoveryErr))
			if err := persistState(e.store, &e.report); err != nil {
				return false, RunReport{}, err
			}
			return false, e.report, fmt.Errorf("uncertain in-flight reviewer recovery remains blocked for %s: %w", task.ManagerID, recoveryErr)
		}
		if task.State != "invoking" && task.State != "integrating" && task.State != "uncertain" && task.State != StatusAwaitingExternal {
			continue
		}
		phase := task.RepairPhase
		if phase == "" {
			if task.State == "integrating" || task.IntegrationAttempts > 0 && task.CandidateID != "" {
				phase = "integrate"
			} else {
				phase = "work"
			}
		}
		agent, configured := e.runtime.Agents[task.ManagerID]
		config := agentexec.Config{}
		var configErr error
		if configured {
			config, configErr = agent.AgentConfig()
		}
		if !configured || configErr != nil || (config.Transport != projectconfig.TransportCodexAppServer && !externalExecutionEnabled(e.ctx)) || task.ID == "" {
			if current := findTask(e.report.Tasks, task.ManagerID); current != nil {
				current.State = "uncertain"
			}
			e.report.Status = StatusBlocked
			e.report.Findings = append(e.report.Findings, "prior process outcome is uncertain for "+task.ManagerID+"; it will not be replayed automatically")
			if err := persistState(e.store, &e.report); err != nil {
				return false, RunReport{}, err
			}
			return false, e.report, fmt.Errorf("uncertain in-flight manager process requires a new plan: %s", task.ManagerID)
		}
		e.pendingRecoveries = append(e.pendingRecoveries, pendingNativeRecovery{managerID: task.ManagerID, phase: phase, taskID: nativeTaskAttemptID(task, phase)})
	}
	if (e.report.Status == StatusFailed || e.report.Status == StatusBlocked) && len(e.pendingRecoveries) == 0 && !pendingReviewRecovered {
		return false, e.report, fmt.Errorf("run status %s cannot be resumed without an exact native invocation to recover", e.report.Status)
	}
	e.report.Status = StatusRunning
	e.report.UpdatedAt = time.Now().UTC()
	e.report.Revision++
	if err := e.store.appendState(e.report); err != nil {
		return false, RunReport{}, err
	}
	return true, RunReport{}, nil
}
