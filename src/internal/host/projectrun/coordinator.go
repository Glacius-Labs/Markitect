package projectrun

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

type managerInvocationCall struct {
	managerID string
	invoke    func(context.Context) (agentexec.RunResult, InvocationLog, error)
}

func nativeTaskAttemptID(task ManagerTask, phase string) string {
	attempt := task.WorkAttempts
	if phase == "integrate" {
		attempt = task.IntegrationAttempts
	}
	return fmt.Sprintf("%s-%s-%d", task.ID, phase, attempt)
}

func batchInvocationError(results []managerInvocationResult) error {
	var failures []error
	var cancellations []error
	for _, result := range results {
		if result.err == nil {
			continue
		}
		wrapped := fmt.Errorf("manager %s: %w", result.managerID, result.err)
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			cancellations = append(cancellations, wrapped)
		} else {
			failures = append(failures, wrapped)
		}
	}
	if len(failures) == 0 {
		failures = cancellations
	}
	return errors.Join(failures...)
}

type managerInvocationResult struct {
	managerID string
	result    agentexec.RunResult
	log       InvocationLog
	err       error
}

type preparedWorkAction struct {
	current  candidateData
	input    *Project
	children []string
	result   managerInvocationResult
}

type pendingNativeRecovery struct {
	managerID string
	phase     string
	taskID    string
}

func taskActionKey(managerID, phase string) string { return managerID + "\x00" + phase }

func recoveryManagerInput(store *runStore, dir string, host Host, root string, base *Snapshot, model projectmodel.Report, report RunReport, plan PlanRecord, task ManagerTask, phase string, dependencyIDs []string) (*Project, []string, []childReport, []string, error) {
	children := activeChildren(report.Tasks, task.ManagerID)
	var input *Project
	var conflicts []string
	var childReports []childReport
	if phase == "work" {
		currentID := plan.InitialCandidateID
		if report.ActiveRepairCandidateID != "" {
			currentID = report.ActiveRepairCandidateID
		}
		currentID, err := workCandidateID(report.Tasks, task, currentID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		current, err := store.readCandidate(dir, currentID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		merged, err := mergeDependencyInputs(store, dir, report.Tasks, current, dependencyIDs)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		input, err = projectForCandidate(host, root, base, merged)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else if phase == "integrate" {
		merged, paths, err := mergeChildCandidates(store, dir, report.Tasks, task, children)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		conflicts = paths
		input, err = projectForCandidate(host, root, base, merged)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		childReports = buildChildSummaries(store, dir, report.Tasks, report.Reviews, children)
	} else {
		return nil, nil, nil, nil, fmt.Errorf("unsupported recovery phase %q", phase)
	}
	if input.Report.ModelDigest != model.ModelDigest {
		return nil, nil, nil, nil, ErrStale
	}
	return input, children, childReports, conflicts, nil
}

// invokeManagerBatch starts every already-reserved action before waiting for
// results. A failing action cancels the shared owned context; all dispatched
// calls are still collected and returned in stable Manager ID order.
func invokeManagerBatch(ctx context.Context, calls []managerInvocationCall) []managerInvocationResult {
	ordered := append([]managerInvocationCall(nil), calls...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].managerID < ordered[j].managerID })
	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]managerInvocationResult, len(ordered))
	var wg sync.WaitGroup
	for index, call := range ordered {
		index, call := index, call
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, log, err := call.invoke(batchCtx)
			results[index] = managerInvocationResult{managerID: call.managerID, result: result, log: log, err: err}
			if err != nil {
				cancel()
			}
		}()
	}
	wg.Wait()
	return results
}

// workCandidateID selects the branch a Manager work turn starts from: the run
// base or the parent's work candidate, except that review rework continues
// from the reviewed candidate. Fresh and recovered turns share it so a resumed
// run rebuilds the same rework input as the uninterrupted review loop.
func workCandidateID(tasks []ManagerTask, task ManagerTask, baseID string) (string, error) {
	currentID := baseID
	if task.ParentTask != "" {
		parent := findTask(tasks, task.ParentTask)
		if parent == nil || parent.CandidateID == "" {
			return "", fmt.Errorf("parent Manager %s has no completed work candidate", task.ParentTask)
		}
		currentID = parent.CandidateID
	}
	if task.CandidateID != "" && task.ReviewStatus == "rework-requested" {
		currentID = task.CandidateID
	}
	return currentID, nil
}

func dependencyCandidateID(task ManagerTask) string {
	if task.IntegrationCandidateID != "" {
		return task.IntegrationCandidateID
	}
	return task.CandidateID
}

// mergeDependencyInputs layers only each prerequisite's output delta over the
// dependent's current branch. The returned candidate is an invocation input;
// callers must continue storing the dependent's own candidate from its branch
// base so read-only dependency bytes are not misattributed as its edits.
func mergeDependencyInputs(store *runStore, dir string, tasks []ManagerTask, base candidateData, dependencyIDs []string) (candidateData, error) {
	merged := make(map[string]File, len(base.Files))
	for path, file := range base.Files {
		merged[path] = file
	}
	parents := append([]string(nil), base.Parents...)
	for _, dependencyID := range dependencyIDs {
		task := findTask(tasks, dependencyID)
		if task == nil {
			return candidateData{}, fmt.Errorf("dependency Manager %s is absent from the run", dependencyID)
		}
		outputID := dependencyCandidateID(*task)
		if outputID == "" {
			return candidateData{}, fmt.Errorf("dependency Manager %s has no integrated candidate", dependencyID)
		}
		output, err := store.readCandidate(dir, outputID)
		if err != nil {
			return candidateData{}, err
		}
		work, err := store.readCandidate(dir, task.CandidateID)
		if err != nil {
			return candidateData{}, err
		}
		if len(work.Parents) == 0 {
			return candidateData{}, fmt.Errorf("dependency Manager %s candidate has no branch baseline", dependencyID)
		}
		baseline, err := store.readCandidate(dir, work.Parents[0])
		if err != nil {
			return candidateData{}, err
		}
		paths := make(map[string]bool, len(baseline.Files)+len(output.Files))
		for path := range baseline.Files {
			paths[path] = true
		}
		for path := range output.Files {
			paths[path] = true
		}
		ordered := make([]string, 0, len(paths))
		for path := range paths {
			ordered = append(ordered, path)
		}
		sort.Strings(ordered)
		for _, path := range ordered {
			before, hadBefore := baseline.Files[path]
			after, hasAfter := output.Files[path]
			if hadBefore == hasAfter && (!hadBefore || fileEqual(before, after)) {
				continue
			}
			current, hasCurrent := merged[path]
			if hasCurrent != hadBefore || (hasCurrent && !fileEqual(current, before)) {
				if hasCurrent == hasAfter && (!hasCurrent || fileEqual(current, after)) {
					continue
				}
				return candidateData{}, fmt.Errorf("dependency candidate conflict on %s from Manager %s", path, dependencyID)
			}
			if hasAfter {
				merged[path] = after
			} else {
				delete(merged, path)
			}
		}
		parents = append(parents, outputID)
	}
	base.Files = merged
	base.Parents = parents
	base.ID = ""
	base.Digest = ""
	return base, nil
}
