package projectrun

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// unroutedReviewError reports a recovered final review whose failure is
// already recorded but whose findings have no empowered rework route.
type unroutedReviewError struct{ cause error }

func (e unroutedReviewError) Error() string { return e.cause.Error() }
func (e unroutedReviewError) Unwrap() error { return e.cause }

// recoverPendingNativeReview reconciles the exact persisted review candidate,
// round and native journal before returning the Manager to the scheduler.
func recoverPendingNativeReview(ctx context.Context, host Host, invoker Invoker, root string, store *runStore, dir string, plan PlanRecord, runtime Runtime, base *Project, report *RunReport, managerID string) error {
	task := findTask(report.Tasks, managerID)
	if task == nil || (task.ReviewStatus != "invoking" && task.ReviewStatus != "uncertain") {
		return nil
	}
	if runtime.Review == nil {
		return fmt.Errorf("pending review has no configured reviewer")
	}
	agent, ok := runtime.Review.Agents[managerID]
	if !ok || agent.Transport != TransportCodexAppServer || task.ReviewCandidateID == "" || task.ReviewRound < 1 {
		return fmt.Errorf("pending review lacks an exact native candidate and round; replay is prohibited")
	}
	phase, final := "work", false
	switch task.ReviewCandidateID {
	case task.CandidateID:
	case task.IntegrationCandidateID:
		phase = "integrate"
	case report.Candidate.ID:
		// Only the final review loop reviews a Manager against the run
		// candidate instead of its own work or integration candidate.
		final = true
		if len(activeChildren(report.Tasks, managerID)) > 0 {
			phase = "integrate"
		}
	default:
		return fmt.Errorf("pending reviewer candidate is outside the persisted task lineage")
	}
	candidate, err := store.readCandidate(dir, task.ReviewCandidateID)
	if err != nil {
		return err
	}
	selected, err := finalProjectForCandidate(host, root, base, candidate)
	if err != nil {
		return err
	}
	deadline := report.StartedAt.Add(time.Duration(runtime.Limits.MaxDuration))
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	record, log, err := invokeReviewer(requireNativeRecovery(bounded, report.ID, report.Invocations), host, invoker, root, plan, runtime, selected, *task, phase, task.ReviewRound, candidate, func(InvocationLog) error { return fmt.Errorf("recovery cannot reserve a new reviewer") }, *report)
	index, matchErr := originalInvocationIndex(report.Invocations, log)
	if matchErr != nil {
		return fmt.Errorf("reconcile original reviewer start: %w", matchErr)
	}
	report.Invocations[index] = retainOriginalReceipt(report.Invocations[index], log)
	if err != nil {
		_ = persistState(store, report)
		return err
	}
	report.Reviews = append(report.Reviews, record)
	task.ReviewStatus = record.Outcome
	if record.Outcome == "pass" {
		task.State = "worked"
		if phase == "integrate" {
			task.State = "integrated"
		}
	} else if final {
		// Rework of this Manager's own branch would never reach the root
		// candidate the reviewer assessed. Route the findings to the empowered
		// parent as the final review loop does; the bounded rework rounds then
		// rebuild the ancestor chain before the final reviews run again.
		task.State = "worked"
		if phase == "integrate" {
			task.State = "integrated"
		}
		requester, requests, routeErr := routeReviewFindings(*task, record, selected.Report, report.Tasks)
		parent := findTask(report.Tasks, requester)
		if routeErr == nil && parent == nil {
			routeErr = fmt.Errorf("review findings for %s have no empowered direct parent", task.ManagerID)
		}
		if routeErr != nil {
			if err := persistState(store, report); err != nil {
				return err
			}
			return unroutedReviewError{cause: routeErr}
		}
		parent.ReworkRequests = append(parent.ReworkRequests, requests...)
	} else {
		var diagnostics []string
		for _, finding := range record.Findings {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %s (%s)", finding.Path, finding.Expectation, finding.Grounding))
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("independent review findings: %s", strings.Join(diagnostics, "; ")))
		task.ReviewStatus = "rework-requested"
		task.State = "review-rework-ready"
		if phase == "integrate" {
			task.State = "integration-retry-ready"
		}
	}
	return persistState(store, report)
}
