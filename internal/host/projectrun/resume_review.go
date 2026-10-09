package projectrun

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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
	phase := "work"
	switch task.ReviewCandidateID {
	case task.CandidateID:
	case task.IntegrationCandidateID:
		phase = "integrate"
	case report.Candidate.ID:
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
	selected, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		return err
	}
	deadline := report.StartedAt.Add(time.Duration(runtime.Limits.MaxDuration))
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	record, log, err := invokeReviewer(requireNativeRecovery(bounded), host, invoker, root, plan, runtime, selected, *task, phase, task.ReviewRound, candidate, func(InvocationLog) error { return fmt.Errorf("recovery cannot reserve a new reviewer") })
	index := -1
	for i := len(report.Invocations) - 1; i >= 0; i-- {
		prior := report.Invocations[i]
		if prior.TaskID == task.ID && prior.Role == "reviewer" && prior.InputDigest == log.InputDigest {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("original reviewer start is absent from the durable ledger")
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
