package projectrun

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

type queuedManagerRework struct {
	requester string
	target    string
	request   ReworkRequest
}

// runManagerReworkRounds honors only manager-directed requests validated
// against the requester's active direct children. Each requested subtree is
// rerun through its existing child tree, then only the affected ancestor chain
// is reintegrated. Other child candidate identities and ledgers remain intact.
func runManagerReworkRounds(ctx context.Context, host Host, invoker Invoker, root string, store *runStore, dir string, plan PlanRecord, runtime Runtime, base *Project, report *RunReport, starts *int, spent *int64) error {
	if runtime.Review == nil {
		for _, task := range report.Tasks {
			if len(task.ReworkRequests) > 0 {
				return fmt.Errorf("manager-directed rework requires enabled review runtime")
			}
		}
		return nil
	}
	pending, err := takePendingReworkRequests(report)
	if err != nil {
		return err
	}
	for round := len(report.ManagerReworkRounds) + 1; len(pending) > 0; round++ {
		if round > runtime.Review.MaxManagerRounds {
			return fmt.Errorf("manager-directed rework round limit %d reached", runtime.Review.MaxManagerRounds)
		}
		ledger := ManagerReworkRound{Number: round, Status: "running", Requests: []ManagerReworkRequestRecord{}}
		for _, queued := range pending {
			parent := findTask(report.Tasks, queued.requester)
			if parent == nil || !containsString(activeChildren(report.Tasks, parent.ManagerID), queued.target) {
				return fmt.Errorf("manager %s rework target %s is not an active direct child", queued.requester, queued.target)
			}
			ledger.Requests = append(ledger.Requests, ManagerReworkRequestRecord{Requester: queued.requester, Request: queued.request, Status: "invoking"})
		}
		report.ManagerReworkRounds = append(report.ManagerReworkRounds, ledger)
		if err := persistState(store, report); err != nil {
			return err
		}
		ancestors := map[string]bool{}
		for i, queued := range pending {
			if err := executeReworkSubtree(ctx, host, invoker, root, store, dir, plan, runtime, base, report, queued.target, queued.request.Goal, queued.request.Reason, starts, spent); err != nil {
				report.ManagerReworkRounds[len(report.ManagerReworkRounds)-1].Status = "blocked"
				return err
			}
			child := findTask(report.Tasks, queued.target)
			if child == nil {
				return fmt.Errorf("reworked child %s disappeared from plan", queued.target)
			}
			cursor := findTask(report.Tasks, queued.requester)
			for cursor != nil {
				ancestors[cursor.ManagerID] = true
				if cursor.ParentTask == "" {
					break
				}
				cursor = findTask(report.Tasks, cursor.ParentTask)
			}
			report.ManagerReworkRounds[len(report.ManagerReworkRounds)-1].Requests[i].Status = "completed"
			candidateID := child.IntegrationCandidateID
			if candidateID == "" {
				candidateID = child.CandidateID
			}
			report.ManagerReworkRounds[len(report.ManagerReworkRounds)-1].Requests[i].CandidateID = candidateID
			if err := persistState(store, report); err != nil {
				return err
			}
		}
		var ordered []ManagerTask
		for id := range ancestors {
			if task := findTask(report.Tasks, id); task != nil {
				ordered = append(ordered, *task)
			}
		}
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].Depth != ordered[j].Depth {
				return ordered[i].Depth > ordered[j].Depth
			}
			return ordered[i].ManagerID < ordered[j].ManagerID
		})
		for _, ancestor := range ordered {
			task := findTask(report.Tasks, ancestor.ManagerID)
			if task == nil {
				continue
			}
			if hasPendingReworkInSubtree(report.Tasks, task.ManagerID) {
				// A lower ancestor just emitted another valid request. Leave this
				// higher ancestor for the next existing bounded rework round.
				continue
			}
			if _, err := reintegrateAfterRework(ctx, host, invoker, root, store, dir, plan, runtime, base, report, task, starts, spent); err != nil {
				return err
			}
		}
		report.ManagerReworkRounds[len(report.ManagerReworkRounds)-1].Status = "completed"
		if err := persistState(store, report); err != nil {
			return err
		}
		pending, err = takePendingReworkRequests(report)
		if err != nil {
			return err
		}
	}
	for _, task := range report.Tasks {
		if len(task.ReworkRequests) > 0 {
			return fmt.Errorf("manager %s has unprocessed rework requests at closure", task.ManagerID)
		}
	}
	return nil
}

func takePendingReworkRequests(report *RunReport) ([]queuedManagerRework, error) {
	var pending []queuedManagerRework
	seen := map[string]ReworkRequest{}
	for i := range report.Tasks {
		task := &report.Tasks[i]
		for _, request := range task.ReworkRequests {
			key := task.ManagerID + "\x00" + request.ManagerID
			if prior, ok := seen[key]; ok {
				if prior != request {
					return nil, fmt.Errorf("manager %s emitted conflicting rework requests for child %s", task.ManagerID, request.ManagerID)
				}
				continue
			}
			seen[key] = request
			pending = append(pending, queuedManagerRework{requester: task.ManagerID, target: request.ManagerID, request: request})
		}
		task.ReworkRequests = nil
	}
	return pending, nil
}

// hasPendingReworkInSubtree reports whether a Manager or any active descendant
// has an explicit direct-child request waiting for the bounded rework loop.
// Ancestor integrations must wait too, or they would review a candidate built
// from the still-stale descendant output.
func hasPendingReworkInSubtree(tasks []ManagerTask, managerID string) bool {
	for i := range tasks {
		if len(tasks[i].ReworkRequests) == 0 {
			continue
		}
		cursor := &tasks[i]
		for cursor != nil {
			if cursor.ManagerID == managerID {
				return true
			}
			if cursor.ParentTask == "" {
				break
			}
			cursor = findTask(tasks, cursor.ParentTask)
		}
	}
	return false
}

func executeReworkSubtree(ctx context.Context, host Host, invoker Invoker, root string, store *runStore, dir string, plan PlanRecord, runtime Runtime, base *Project, report *RunReport, managerID, goal, reason string, starts *int, spent *int64) error {
	task := findTask(report.Tasks, managerID)
	if task == nil {
		return fmt.Errorf("rework target %s is absent", managerID)
	}
	if len(task.Questions)+len(task.Risks) != 0 || len(task.Obligations) != 0 {
		return fmt.Errorf("rework target %s has unresolved obligations; targeted work cannot clear them", managerID)
	}
	children := activeChildren(report.Tasks, managerID)
	currentID := task.CandidateID
	if currentID == "" {
		currentID = plan.InitialCandidateID
	}
	current, err := store.readCandidate(dir, currentID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(goal) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("manager rework requires a bounded goal and reason")
	}
	task.Goal = goal
	task.RepairDiagnostic = "Manager-directed rework: " + reason
	reviewPassed := false
	reviewRoundBase := reviewCount(report.Reviews, task.ManagerID, "work")
	if reviewRoundBase >= runtime.Review.MaxRounds {
		return fmt.Errorf("targeted work for %s exhausted the cumulative review round limit", managerID)
	}
	for round := reviewRoundBase + 1; round <= runtime.Review.MaxRounds; round++ {
		if err := ensureRemaining(ctx, runtime, *starts, *spent, report.Invocations, "targeted manager work"); err != nil {
			return err
		}
		input, err := projectForCandidate(host, root, base.Snapshot, current)
		if err != nil {
			return err
		}
		task.State, task.RepairPhase = "invoking", "work"
		task.WorkAttempts++
		task.Attempts++
		if err := persistState(store, report); err != nil {
			return err
		}
		proposal, invocation, invokeErr := invokeManager(ctx, host, invoker, root, runtime, plan, input, *task, "work", children, nil, nil, task.RepairDiagnostic, 0, nil, *starts)
		*starts++
		if invokeErr != nil {
			if appendInvocationReceipt(report, invocation) {
				*spent = totalCost(report.Invocations)
			}
			task.State = "uncertain"
			_ = persistState(store, report)
			return fmt.Errorf("targeted work for %s: %w", managerID, invokeErr)
		}
		report.Invocations = append(report.Invocations, invocation)
		*spent = totalCost(report.Invocations)
		if costExceedsLimit(report.Invocations, runtime.Limits.MaxCostMicros) {
			return fmt.Errorf("estimated cost limit exceeded during targeted work")
		}
		parsed, err := decodeTaskResponse(proposal.Response.ReportJSON, "work", children)
		if err != nil || validateTaskOutcome(proposal.Response.Outcome, parsed) != nil || parsed.Status != "complete" && parsed.Status != "no-op" || len(parsed.Questions)+len(parsed.Risks) != 0 {
			if err == nil {
				err = fmt.Errorf("rework manager report is incomplete or unresolved")
			}
			return fmt.Errorf("targeted work response for %s: %w", managerID, err)
		}
		if parsed.Status == "no-op" && len(proposal.Response.CandidateFiles) > 0 {
			return fmt.Errorf("targeted work response for %s claimed no-op while proposing files", managerID)
		}
		candidate, err := applyProposal(current, proposal.Response.CandidateFiles, input.Config, input.Report, *task, "work", nil, runtime.Limits, input.Snapshot)
		if err != nil {
			return err
		}
		candidate.ID, err = newID()
		if err != nil {
			return err
		}
		candidate.Parents = []string{current.ID}
		if err := store.writeCandidate(dir, candidate); err != nil {
			return err
		}
		candidate, err = store.readCandidate(dir, candidate.ID)
		if err != nil {
			return err
		}
		task.CandidateID, task.ReportID = candidate.ID, invocation.ReportID
		task.WrittenPaths = unionPaths(task.WrittenPaths, proposalPaths(proposal.Response.CandidateFiles))
		task.Summary, task.Questions, task.Risks, task.Delegations, task.ReportStatus = parsed.Summary, parsed.Questions, parsed.Risks, parsed.Delegations, parsed.Status
		task.Obligations, err = newLocalObligations(task.ManagerID, parsed.Questions, parsed.Risks)
		if err != nil {
			return fmt.Errorf("targeted work obligation provenance for %s: %w", task.ManagerID, err)
		}
		for _, delegation := range parsed.Delegations {
			child := findTask(report.Tasks, delegation.ManagerID)
			if child == nil {
				return fmt.Errorf("rework delegation target %s is not active", delegation.ManagerID)
			}
			child.Goal = delegation.Goal
		}
		report.Candidate = candidateRef(candidate, false)
		if err := persistState(store, report); err != nil {
			return err
		}
		candidateProject, err := projectForCandidate(host, root, base.Snapshot, candidate)
		if err != nil {
			return err
		}
		if !reviewRequired(candidateProject, *task) {
			task.State, task.RepairPhase, task.RepairDiagnostic, task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "worked", "", "", "not-required", candidate.ID, 0
			current = candidate
			reviewPassed = true
			if err := persistState(store, report); err != nil {
				return err
			}
			break
		}
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "invoking", candidate.ID, round
		if err := persistState(store, report); err != nil {
			return err
		}
		if err := ensureRemaining(ctx, runtime, *starts, *spent, report.Invocations, "targeted reviewer"); err != nil {
			return err
		}
		review, _, err := recordReview(ctx, host, invoker, root, store, report, plan, runtime, candidateProject, *task, "work", round, candidate)
		*starts++
		*spent = totalCost(report.Invocations)
		if costExceedsLimit(report.Invocations, runtime.Limits.MaxCostMicros) {
			return fmt.Errorf("estimated cost limit exceeded during targeted review")
		}
		if err != nil {
			task.State, task.ReviewStatus = "uncertain", "uncertain"
			_ = persistState(store, report)
			return err
		}
		if review.Outcome == "pass" {
			task.State, task.RepairPhase, task.RepairDiagnostic, task.ReviewStatus = "worked", "", "", "pass"
			current = candidate
			reviewPassed = true
			if err := persistState(store, report); err != nil {
				return err
			}
			break
		}
		var findings []string
		for _, finding := range review.Findings {
			findings = append(findings, finding.Path+": "+finding.Expectation+" ("+finding.Grounding+")")
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("%s; reviewer findings: %s", task.RepairDiagnostic, strings.Join(findings, "; ")))
		current = candidate
	}
	if !reviewPassed {
		return fmt.Errorf("targeted work for %s reached review round limit", managerID)
	}
	for _, childID := range children {
		child := findTask(report.Tasks, childID)
		if child == nil {
			return fmt.Errorf("rework subtree child %s is absent", childID)
		}
		if err := executeReworkSubtree(ctx, host, invoker, root, store, dir, plan, runtime, base, report, childID, child.Goal, "routed from direct parent rework task", starts, spent); err != nil {
			return err
		}
	}
	if len(children) > 0 {
		_, err := reintegrateAfterRework(ctx, host, invoker, root, store, dir, plan, runtime, base, report, task, starts, spent)
		return err
	}
	return nil
}

func reintegrateAfterRework(ctx context.Context, host Host, invoker Invoker, root string, store *runStore, dir string, plan PlanRecord, runtime Runtime, base *Project, report *RunReport, task *ManagerTask, starts *int, spent *int64) ([]ReworkRequest, error) {
	children := activeChildren(report.Tasks, task.ManagerID)
	if len(children) == 0 {
		return nil, nil
	}
	merged, conflicts, err := mergeChildCandidates(store, dir, report.Tasks, *task, children)
	if err != nil {
		return nil, err
	}
	input, err := projectForCandidate(host, root, base.Snapshot, merged)
	if err != nil {
		return nil, err
	}
	childReports := buildChildSummaries(store, dir, report.Tasks, report.Reviews, children)
	if err := ensureRemaining(ctx, runtime, *starts, *spent, report.Invocations, "targeted integration"); err != nil {
		return nil, err
	}
	task.State, task.RepairPhase = "integrating", "integrate"
	task.IntegrationAttempts++
	task.Attempts++
	if err := persistState(store, report); err != nil {
		return nil, err
	}
	diagnostic := task.RepairDiagnostic
	if diagnostic == "" {
		diagnostic = "Manager-directed rework reintegration"
	}
	proposal, invocation, err := invokeManager(ctx, host, invoker, root, runtime, plan, input, *task, "integrate", children, conflicts, childReports, diagnostic, 0, nil, *starts)
	*starts++
	if err != nil {
		if appendInvocationReceipt(report, invocation) {
			*spent = totalCost(report.Invocations)
		}
		task.State = "uncertain"
		_ = persistState(store, report)
		return nil, err
	}
	report.Invocations = append(report.Invocations, invocation)
	*spent = totalCost(report.Invocations)
	if costExceedsLimit(report.Invocations, runtime.Limits.MaxCostMicros) {
		return nil, fmt.Errorf("estimated cost limit exceeded during targeted integration")
	}
	parsed, err := decodeTaskResponse(proposal.Response.ReportJSON, "integrate", children)
	if err != nil {
		return nil, err
	}
	if err := validateTaskOutcome(proposal.Response.Outcome, parsed); err != nil {
		return nil, err
	}
	if parsed.EscalateTo != "" && parsed.EscalateTo != escalationTarget(*task) {
		return nil, fmt.Errorf("Manager %s may escalate only to its nearest empowered recipient %q", task.ManagerID, escalationTarget(*task))
	}
	outstanding, err := managerObligationRecords(*task, report.Tasks, children)
	if err != nil {
		return nil, fmt.Errorf("manager %s reintegration obligations: %w", task.ManagerID, err)
	}
	resolvedObligations, remainingObligations, err := resolveObligationRecords(outstanding, parsed.ResolvedQuestions, parsed.ResolvedRisks)
	if err != nil {
		return nil, fmt.Errorf("manager %s reintegration resolution: %w", task.ManagerID, err)
	}
	remainingQ, remainingR := obligationTexts(remainingObligations)
	remainingQ = append(remainingQ, parsed.Questions...)
	remainingR = append(remainingR, parsed.Risks...)
	if len(remainingQ)+len(remainingR) > 0 {
		if parsed.Status != "partial" || parsed.EscalateTo != escalationTarget(*task) {
			return nil, fmt.Errorf("manager %s reintegration has unresolved obligations that did not escalate to its nearest parent", task.ManagerID)
		}
	} else if parsed.Status != "complete" || parsed.EscalateTo != "" {
		return nil, fmt.Errorf("manager %s reintegration is not complete after resolving obligations", task.ManagerID)
	}
	candidate, err := applyProposal(merged, proposal.Response.CandidateFiles, input.Config, input.Report, *task, "integrate", conflicts, runtime.Limits, input.Snapshot)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 && !proposesEvery(proposal.Response.CandidateFiles, conflicts) {
		return nil, fmt.Errorf("manager %s did not resolve integration conflict paths", task.ManagerID)
	}
	candidate.ID, err = newID()
	if err != nil {
		return nil, err
	}
	candidate.Parents = childCandidateIDs(report.Tasks, children)
	if err := store.writeCandidate(dir, candidate); err != nil {
		return nil, err
	}
	candidate, err = store.readCandidate(dir, candidate.ID)
	if err != nil {
		return nil, err
	}
	task.State, task.IntegrationReportID, task.IntegrationCandidateID = "integrated", invocation.ReportID, candidate.ID
	task.IntegratedPaths = unionPaths(task.IntegratedPaths, changedCandidatePaths(merged, candidate))
	task.Summary = parsed.Summary
	newLocal, err := newLocalResponseObligations(task.ManagerID, remainingObligations, parsed.Questions, parsed.Risks)
	if err != nil {
		return nil, fmt.Errorf("manager %s reintegration obligation provenance: %w", task.ManagerID, err)
	}
	if err := applyObligationResolutions(report, resolvedObligations); err != nil {
		return nil, fmt.Errorf("manager %s reintegration obligation resolution: %w", task.ManagerID, err)
	}
	task.Obligations = append(appendForwardedObligations(remainingObligations, task.ManagerID), newLocal...)
	task.Questions, task.Risks = uniqueSorted(remainingQ), uniqueSorted(remainingR)
	if len(task.Questions)+len(task.Risks) == 0 {
		task.ReportStatus = "complete"
	} else {
		task.ReportStatus = "partial"
		if err := recordEscalation(report, *task, TaskResponse{Summary: parsed.Summary, Questions: task.Questions, Risks: task.Risks, EscalateTo: parsed.EscalateTo}); err != nil {
			return nil, err
		}
		markObligationEscalationsEscalated(report, task.ManagerID, task.Obligations)
	}
	task.ReworkRequests = append([]ReworkRequest(nil), parsed.ReworkRequests...)
	report.Candidate = candidateRef(candidate, false)
	if err := persistState(store, report); err != nil {
		return nil, err
	}
	if len(task.ReworkRequests) > 0 {
		// The parent explicitly requested another child correction. This
		// candidate is not yet the post-rework integration result, so defer its
		// reviewer until the bounded manager-rework loop has applied the request.
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "", "", 0
		if err := persistState(store, report); err != nil {
			return nil, err
		}
		return append([]ReworkRequest(nil), parsed.ReworkRequests...), nil
	}
	compiled, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		return nil, err
	}
	if !reviewRequired(compiled, *task) {
		task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "not-required", candidate.ID, 0
		if err := persistState(store, report); err != nil {
			return nil, err
		}
		return append([]ReworkRequest(nil), parsed.ReworkRequests...), nil
	}
	reviewRound := reviewCount(report.Reviews, task.ManagerID, "integrate") + 1
	if reviewRound > runtime.Review.MaxRounds {
		return nil, fmt.Errorf("manager %s exhausted the cumulative integration review round limit", task.ManagerID)
	}
	task.ReviewStatus, task.ReviewCandidateID, task.ReviewRound = "invoking", candidate.ID, reviewRound
	if err := persistState(store, report); err != nil {
		return nil, err
	}
	if err := ensureRemaining(ctx, runtime, *starts, *spent, report.Invocations, "targeted integration reviewer"); err != nil {
		return nil, err
	}
	review, _, err := recordReview(ctx, host, invoker, root, store, report, plan, runtime, compiled, *task, "integrate", reviewRound, candidate)
	*starts++
	*spent = totalCost(report.Invocations)
	if costExceedsLimit(report.Invocations, runtime.Limits.MaxCostMicros) {
		return nil, fmt.Errorf("estimated cost limit exceeded during targeted integration review")
	}
	if err != nil {
		task.State, task.ReviewStatus = "uncertain", "uncertain"
		_ = persistState(store, report)
		return nil, err
	}
	task.ReviewStatus = review.Outcome
	if review.Outcome != "pass" {
		var findings []string
		for _, finding := range review.Findings {
			findings = append(findings, finding.Path+": "+finding.Expectation+" ("+finding.Grounding+")")
		}
		task.RepairDiagnostic = boundedRepairDiagnostic(fmt.Errorf("integration review findings: %s", strings.Join(findings, "; ")))
		if reviewCount(report.Reviews, task.ManagerID, "integrate") >= runtime.Review.MaxRounds {
			return nil, fmt.Errorf("manager %s exhausted the cumulative integration review round limit", task.ManagerID)
		}
		return reintegrateAfterRework(ctx, host, invoker, root, store, dir, plan, runtime, base, report, task, starts, spent)
	}
	task.RepairDiagnostic = ""
	if err := persistState(store, report); err != nil {
		return nil, err
	}
	return append([]ReworkRequest(nil), parsed.ReworkRequests...), nil
}

func unionPaths(groups ...[]string) []string {
	set := map[string]bool{}
	for _, group := range groups {
		for _, path := range group {
			set[path] = true
		}
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func recordReview(ctx context.Context, host Host, invoker Invoker, root string, store *runStore, report *RunReport, plan PlanRecord, runtime Runtime, project *Project, task ManagerTask, phase string, round int, candidate candidateData) (ReviewRecord, InvocationLog, error) {
	startedIndex := -1
	review, invocation, err := invokeReviewer(ctx, host, invoker, root, plan, runtime, project, task, phase, round, candidate, func(started InvocationLog) error {
		report.Invocations = append(report.Invocations, started)
		startedIndex = len(report.Invocations) - 1
		return persistState(store, report)
	}, *report)
	if startedIndex >= 0 {
		report.Invocations[startedIndex] = invocation
		if err == nil {
			report.Reviews = append(report.Reviews, review)
		}
		if err := persistState(store, report); err != nil {
			return review, invocation, err
		}
	}
	return review, invocation, err
}

func ensureRemaining(ctx context.Context, runtime Runtime, starts int, spent int64, invocations []InvocationLog, stage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if starts >= runtime.Limits.MaxStarts {
		return fmt.Errorf("runtime start limit reached before %s", stage)
	}
	if knownCostOverflow(invocations) || spent >= runtime.Limits.MaxCostMicros {
		return fmt.Errorf("estimated cost limit reached before %s", stage)
	}
	return nil
}

var _ = agentexec.OutcomeProposed
