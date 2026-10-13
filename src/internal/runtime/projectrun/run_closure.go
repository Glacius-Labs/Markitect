package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"fmt"
	"strings"
	"time"
)

// closeRun completes manager-directed rework, fresh final reviews, and guarded
// candidate closure after the state-driven scheduler has no ready actions.
func (e *runExecution) closeRun() (RunReport, error) {
	if err := runManagerReworkRounds(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, &e.starts, &e.spent); err != nil {
		return blockRun(e.store, e.report, err)
	}
	rootCandidateID := findRootCandidate(e.report.Tasks)
	if rootCandidateID == "" {
		rootCandidateID = e.report.Candidate.ID
	}
	finalCandidate, err := e.store.readCandidate(e.dir, rootCandidateID)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	if report, err := e.reviewFinalManagers(&finalCandidate, &rootCandidateID); err != nil {
		return report, err
	}
	if err := validateReportClosure(e.report); err != nil {
		return blockRun(e.store, e.report, err)
	}
	finalCandidate, err = completeCandidateDocument(e.host, e.root, e.store, e.dir, e.project, finalCandidate)
	if err != nil {
		return blockRun(e.store, e.report, err)
	}
	if err := validateFinalCandidate(e.host, e.root, e.project, finalCandidate, e.plan); err != nil {
		return blockRun(e.store, e.report, err)
	}
	if err := requireFreshReviews(e.host, e.root, e.store, e.dir, e.project, finalCandidate, e.plan, e.runtime, e.report); err != nil {
		return blockRun(e.store, e.report, err)
	}
	if e.report.ActiveRepairCandidateID != "" {
		seed, seedErr := e.store.readCandidate(e.dir, e.report.ActiveRepairCandidateID)
		if seedErr != nil {
			return failRun(e.store, e.report, seedErr)
		}
		before, snapshotErr := snapshotWithCandidate(e.project.Snapshot, seed)
		if snapshotErr != nil {
			return failRun(e.store, e.report, snapshotErr)
		}
		after, snapshotErr := snapshotWithCandidate(e.project.Snapshot, finalCandidate)
		if snapshotErr != nil {
			return failRun(e.store, e.report, snapshotErr)
		}
		if before.Digest() == after.Digest() {
			return failRun(e.store, e.report, fmt.Errorf("repair round produced no source or model change; the unchanged failed candidate cannot be verified again"))
		}
		last := &e.report.RepairRounds[len(e.report.RepairRounds)-1]
		last.CandidateID, last.Status = finalCandidate.ID, "integrated"
		e.report.ActiveRepairCandidateID = ""
	}
	e.report.Candidate = candidateRef(finalCandidate, true)
	e.report.Status = StatusIntegrated
	e.report.UpdatedAt = time.Now().UTC()
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	return e.report, nil
}

func (e *runExecution) reviewFinalManagers(finalCandidate *candidateData, rootCandidateID *string) (RunReport, error) {
	if e.runtime.Review == nil {
		return RunReport{}, nil
	}
	finalProject, err := finalProjectForCandidate(e.host, e.root, e.project, *finalCandidate)
	if err != nil {
		return failRun(e.store, e.report, err)
	}
	for i := range e.report.Tasks {
		task := &e.report.Tasks[i]
		phase := "work"
		if len(activeChildren(e.report.Tasks, task.ManagerID)) > 0 {
			phase = "integrate"
		}
		if !phaseReviewRequired(finalProject, *task, e.report.Tasks, phase) {
			task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "not-required", finalCandidate.ID, 0
			if err := persistState(e.store, &e.report); err != nil {
				return RunReport{}, err
			}
			continue
		}
		_, _, fresh, freshnessErr := freshPassedReviewForScope(e.host, e.root, e.store, e.dir, e.plan, e.report, finalProject, *finalCandidate, *task, phase)
		if freshnessErr != nil {
			return failRun(e.store, e.report, freshnessErr)
		}
		if fresh {
			continue
		}
		reviewRound := reviewCount(e.report.Reviews, task.ManagerID, phase) + 1
		if reviewRound > e.runtime.Review.MaxRounds {
			return blockRun(e.store, e.report, fmt.Errorf("Manager %s exhausted the cumulative final review round limit", task.ManagerID))
		}
		if e.starts >= e.runtime.Limits.MaxStarts || e.spent >= e.runtime.Limits.MaxCostMicros {
			return blockRun(e.store, e.report, fmt.Errorf("final review for %s exceeds the cumulative run budget", task.ManagerID))
		}
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "invoking", finalCandidate.ID, reviewRound
		if err := persistState(e.store, &e.report); err != nil {
			return RunReport{}, err
		}
		if report, err := e.reviewFinalManager(task, finalProject, *finalCandidate, phase, reviewRound, finalProject, finalCandidate, rootCandidateID, &i); err != nil {
			return report, err
		}
	}
	return RunReport{}, nil
}

func (e *runExecution) reviewFinalManager(task *ManagerTask, finalProject *projectvalue.Project, finalCandidate candidateData, phase string, reviewRound int, candidateProject *projectvalue.Project, candidate *candidateData, rootCandidateID *string, index *int) (RunReport, error) {
	startedIndex := -1
	review, reviewLog, reviewErr := invokeReviewer(e.ctx, e.host, e.invoker, e.root, e.plan, e.runtime, finalProject, *task, phase, reviewRound, finalCandidate, e.report)
	startedIndex = invocationAttemptIndex(e.report.Invocations, reviewLog)
	e.refreshStarts()
	if startedIndex >= 0 {
		e.report.Invocations[startedIndex] = reviewLog
		if reviewErr == nil {
			e.report.Reviews = append(e.report.Reviews, review)
		}
		e.spent = totalCost(e.report.Invocations)
		if costExceedsLimit(e.report.Invocations, e.runtime.Limits.MaxCostMicros) {
			return blockRun(e.store, e.report, fmt.Errorf("estimated cost limit exceeded during final review"))
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
		return failRun(e.store, e.report, fmt.Errorf("final review for %s: %w", task.ManagerID, reviewErr))
	}
	task.ReviewStatus = review.Outcome
	if review.Outcome != "pass" {
		requester, requests, routeErr := routeReviewFindings(*task, review, finalProject.Report, e.report.Tasks)
		if routeErr != nil {
			if task.ParentTask != "" || len(activeChildren(e.report.Tasks, task.ManagerID)) == 0 {
				return blockRun(e.store, e.report, routeErr)
			}
			var diagnostics []string
			for _, finding := range review.Findings {
				diagnostics = append(diagnostics, fmt.Sprintf("%s: %s (%s)", finding.Path, finding.Expectation, finding.Grounding))
			}
			task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("final review findings: %s", strings.Join(diagnostics, "; ")))
			_, reworkErr := reintegrateAfterRework(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, task, &e.starts, &e.spent)
			if reworkErr != nil {
				return blockRun(e.store, e.report, reworkErr)
			}
			requester = task.ManagerID
		}
		parent := findTask(e.report.Tasks, requester)
		if parent == nil {
			return blockRun(e.store, e.report, fmt.Errorf("review findings for %s have no empowered direct parent", task.ManagerID))
		}
		parent.ReworkRequests = append(parent.ReworkRequests, requests...)
		if err := runManagerReworkRounds(e.ctx, e.host, e.invoker, e.root, e.store, e.dir, e.plan, e.runtime, e.project, &e.report, &e.starts, &e.spent); err != nil {
			return blockRun(e.store, e.report, err)
		}
		*rootCandidateID = findRootCandidate(e.report.Tasks)
		updated, err := e.store.readCandidate(e.dir, *rootCandidateID)
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		*candidate = updated
		updatedProject, err := finalProjectForCandidate(e.host, e.root, e.project, updated)
		if err != nil {
			return failRun(e.store, e.report, err)
		}
		*candidateProject = *updatedProject
		*index = -1
		return RunReport{}, nil
	}
	if err := persistState(e.store, &e.report); err != nil {
		return RunReport{}, err
	}
	return RunReport{}, nil
}
