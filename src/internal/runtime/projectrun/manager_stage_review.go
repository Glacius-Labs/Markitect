package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

// Prepared reviewer validation is shared by both ordinary candidate reviews
// and Brownfield report reviews. Workflow-specific context stays in builders.
func validatePreparedManagerReview(result agentexec.RunResult, protocolDigest string, refs []reviewFileRef) (reviewResponse, []ReviewFinding, error) {
	parsed, err := validateReviewerResponse(result.Response, result.Receipt, protocolDigest)
	if err != nil {
		return parsed, nil, err
	}
	findings, err := validateReviewFindings(parsed.Findings, refs)
	return parsed, findings, err
}

func managerStageReviewRequest(parent InvocationLog, runID string, spec ManagerStageReviewSpec, round int, raw []byte) (agentexec.Request, []reviewFileRef, string, error) {
	if parent.Attempt == nil || parent.Attempt.ReportRef == nil || spec.Purpose == "" || spec.AcceptedModel.Manager.ID == "" {
		return agentexec.Request{}, nil, "", errors.New("stage review requires real accepted context and exact report evidence")
	}
	acceptedDigest, err := digest(spec.AcceptedModel)
	if err != nil {
		return agentexec.Request{}, nil, "", err
	}
	if spec.AcceptedModelDigest != "" && spec.AcceptedModelDigest != acceptedDigest {
		return agentexec.Request{}, nil, "", errors.New("accepted stage review context digest differs")
	}
	path := ".markitect/runs/" + runID + "/" + parent.Attempt.ReportRef.Path
	reportRef := reviewFileRef{Path: path, Mode: "0644", Digest: parent.Attempt.ReportRef.Digest, Grounding: []string{"report:" + parent.Attempt.ReportRef.Digest, "stage:" + parent.Attempt.IterationID + "/" + parent.Phase}}
	refs := []reviewFileRef{reportRef}
	artifacts := []agentexec.Artifact{{Path: path, Mode: "0644", Digest: parent.Attempt.ReportRef.Digest, Content: append([]byte(nil), raw...)}}
	sourceRefs := []reviewFileRef{}
	seen := map[string]bool{path: true}
	for _, file := range spec.Evidence {
		if seen[file.Path] || file.Path == "" || len(spec.Grounding[file.Path]) == 0 {
			return agentexec.Request{}, nil, "", errors.New("stage reviewer evidence lacks unique assigned grounding")
		}
		if file.Digest != rawContentDigest(file.Content) {
			return agentexec.Request{}, nil, "", errors.New("assigned stage review evidence byte digest differs")
		}
		seen[file.Path] = true
		ref := reviewFileRef{Path: file.Path, Mode: file.Mode, Digest: rawContentDigest(file.Content), Grounding: append([]string(nil), spec.Grounding[file.Path]...)}
		sourceRefs = append(sourceRefs, ref)
		refs = append(refs, ref)
		copy := file
		copy.Content = append([]byte(nil), file.Content...)
		artifacts = append(artifacts, copy)
	}
	reviewContext := reviewerContext{Kind: "projectrun-review/v1", Operation: "adoption", SubjectKind: "manager-stage-report",
		ReviewerGuidance: reviewerAssessmentGuidance + " Assess the exact report proposal and its grounding. Draft hierarchy, YAML, proposed contracts and submitted runtime records are assessment evidence, not accepted model or authenticated execution. Never implement, dispatch Managers or change source/target artifacts. The report artifact is the assessment subject; it is not an implementation candidate.",
		Strictness:       spec.Strictness, RunGoal: spec.Purpose, OwnTask: spec.Purpose, ManagerID: parent.Attempt.ManagerID, Phase: parent.Phase, Round: round, CandidateID: parent.Attempt.ID, CandidateDigest: parent.Attempt.ReportRef.Digest, ChangedPaths: []string{path}, AcceptedModel: spec.AcceptedModel,
		ScopedModel:    reviewerScopedModel{Statements: append([]projectmodel.Statement(nil), spec.AcceptedModel.Statements...), Contracts: append([]projectmodel.Statement(nil), spec.AcceptedModel.Contracts...), Artifacts: append([]projectmodel.Artifact(nil), spec.AcceptedModel.Artifacts...), OwnedPaths: []string{path}},
		CandidateFiles: refs, CandidateChanges: []candidateChange{}, ProposedContext: append(json.RawMessage(nil), spec.ProposedContext...), ChildBindings: append(json.RawMessage(nil), spec.ChildBindings...), SourceEvidence: sourceRefs, ResponseSchema: reviewResponseSchema()}
	contextBytes, err := json.Marshal(reviewContext)
	if err != nil {
		return agentexec.Request{}, nil, "", err
	}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: spec.SourceRevision, ModelDigest: acceptedDigest, ModulePin: acceptedDigest, ProjectionID: acceptedDigest, ScopeIDs: uniqueSorted([]string{parent.Attempt.ManagerID, spec.AcceptedModel.Manager.ID}), PolicyIDs: []string{"selected-evidence-only", "report-only", "no-source-or-target-writes"}, Context: contextBytes, Artifacts: artifacts}
	if err := canonicalizeReviewContext(&request); err != nil {
		return agentexec.Request{}, nil, "", err
	}
	scopeDigest, err := digest(struct {
		Parent, Subject, Accepted, Context, Children string
		Files                                        []reviewFileRef
	}{parent.Attempt.ID, parent.Attempt.ReportRef.Digest, acceptedDigest, parent.Attempt.ContextDigest, rawContentDigest(spec.ChildBindings), refs})
	return request, refs, scopeDigest, err
}

func stageReviewCount(report RunReport, manager, phase string) int {
	count := 0
	for _, log := range report.Invocations {
		if a := log.Attempt; a != nil && a.ParentAttemptID != "" && a.ManagerID == manager && a.Phase == phase {
			count++
		}
	}
	return count
}

func existingManagerStageReview(report RunReport, parent InvocationLog) (ReviewRecord, bool) {
	for _, record := range report.Reviews {
		if record.CandidateID == parent.Attempt.ID && parent.Attempt.ReportRef != nil && record.CandidateDigest == parent.Attempt.ReportRef.Digest && record.TaskID == parent.TaskID && record.ManagerID == parent.Attempt.ManagerID && record.Phase == parent.Phase {
			return record, true
		}
	}
	return ReviewRecord{}, false
}

func passedManagerStageReview(report RunReport, parent InvocationLog) bool {
	record, ok := existingManagerStageReview(report, parent)
	if !ok || record.Outcome != "pass" {
		return false
	}
	for _, log := range report.Invocations {
		if a := log.Attempt; a != nil && a.ParentAttemptID == parent.Attempt.ID && a.Status == "accepted" && a.ReviewSubjectDigest == parent.Attempt.ReportRef.Digest && a.ReviewPolicyDigest == parent.Attempt.ReviewPolicyDigest && log.Receipt.RunID == record.Receipt.RunID && log.CostKnown && !log.CostOverflow {
			return true
		}
	}
	return false
}

func ReviewManagerStage(ctx context.Context, host Host, invoker Invoker, root, runID, attemptID string, spec ManagerStageReviewSpec, expectedStateDigest string) (ManagerStageReviewResult, error) {
	if ctx == nil || host.ValidateManagerStage == nil || invoker == nil {
		return ManagerStageReviewResult{}, errors.New("stage review requires context, invoker and Application fact validation")
	}
	if err := ctx.Err(); err != nil {
		return ManagerStageReviewResult{}, err
	}
	store, err := newManagerStageStore(root)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	unlock, err := store.lock()
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	defer unlock()
	report, err := store.readLatestState(runID)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	if report.ManagerStageBinding == nil || report.Digest != expectedStateDigest {
		return ManagerStageReviewResult{}, errors.New("Manager review state changed")
	}
	var parent InvocationLog
	for _, log := range report.Invocations {
		if log.Attempt != nil && log.Attempt.ID == attemptID && log.Attempt.ParentAttemptID == "" {
			parent = log
			break
		}
	}
	if parent.Attempt == nil || parent.Attempt.ReportRef == nil || !parent.Attempt.ReviewRequired || parent.Attempt.Status != "transport-succeeded" && parent.Attempt.Status != "accepted" {
		return ManagerStageReviewResult{}, errors.New("Manager review has no exact terminal report subject")
	}
	if spec.Validation.SessionID != report.ManagerStageBinding.SessionID || spec.Validation.SessionDigest != parent.Attempt.SessionDigest || spec.Validation.IterationID != parent.Attempt.IterationID || spec.Validation.ManagerID != parent.Attempt.ManagerID || spec.Validation.AgentManagerID != parent.Attempt.AgentManagerID || spec.Validation.Phase != parent.Phase || spec.Validation.ContextDigest != parent.Attempt.ContextDigest || spec.Validation.RequestContractDigest != parent.Attempt.RequestContractDigest || spec.Validation.RuntimeFileDigest != parent.Attempt.RuntimeFileDigest {
		return ManagerStageReviewResult{}, errors.New("Manager review prepared facts differ from its original stage")
	}
	sourceRoot, err := filepath.EvalSymlinks(spec.SourceRoot)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	if !sameRecoveryPath(sourceRoot, root) {
		return ManagerStageReviewResult{}, errors.New("Manager review source differs from its store")
	}
	spec.SourceRevision = report.ManagerStageBinding.SourceRevision
	projection, err := managerStageReviewProjection(invoker, ManagerStageSpec{Validation: spec.Validation, Review: &spec.Review})
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	if projection.PolicyDigest != parent.Attempt.ReviewPolicyDigest {
		return ManagerStageReviewResult{}, errors.New("Manager reviewer policy changed")
	}
	configFP, err := invoker.Fingerprint(spec.Config)
	if err != nil || configFP != projection.ConfigFingerprint {
		return ManagerStageReviewResult{}, errors.New("Manager reviewer fingerprint differs from configured policy")
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		return ManagerStageReviewResult{}, err
	}
	if record, ok := existingManagerStageReview(report, parent); ok {
		return ManagerStageReviewResult{ReviewRecord: record, RunID: runID, ParentAttemptID: attemptID, SubjectDigest: parent.Attempt.ReportRef.Digest, StateDigest: report.Digest, Status: record.Outcome}, nil
	}
	for _, previous := range report.Invocations {
		if previous.Attempt != nil && previous.Attempt.ParentAttemptID == attemptID {
			return recoverManagerStageReview(ctx, host, invoker, store, &report, parent, previous, spec)
		}
	}
	round := stageReviewCount(report, parent.Attempt.ManagerID, parent.Phase) + 1
	if round > spec.Review.MaxRounds {
		return ManagerStageReviewResult{}, errors.New("Manager phase review round ceiling is exhausted")
	}
	budget := frozenManagerStageBudget(report.ManagerStageBinding.Budget, spec.Limits)
	if err := managerStageAdmission(report, budget, spec.Config.Timeout, false); err != nil {
		return ManagerStageReviewResult{}, err
	}
	raw, err := readStageEvidence(store, runID, parent.Attempt.ReportRef)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	request, refs, scope, err := managerStageReviewRequest(parent, runID, spec, round, raw)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	protocol, err := nativeRequestInputDigest(request)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	rawDigest, err := digest(request)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	requestFP, err := fingerprintForRequest(invoker, spec.Config, request)
	if err != nil || requestFP != projection.ConfigFingerprint {
		return ManagerStageReviewResult{}, errors.New("Manager reviewer request fingerprint changed")
	}
	limits := spec.RuntimeLimits
	limits.MaxStarts = budget.MaxStarts
	journal, err := newManagerStageJournal(store, &report, limits)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	id := fmt.Sprintf("%s-review-%d", attemptID, round)
	log := InvocationLog{TaskID: parent.TaskID, Role: "reviewer", Phase: parent.Phase, InputDigest: rawDigest, Outcome: "started", Attempt: &ManagerStageAttempt{ID: id, StageKey: parent.Attempt.StageKey + "\x00review", ParentAttemptID: attemptID, SessionDigest: parent.Attempt.SessionDigest, IterationID: parent.Attempt.IterationID, ManagerID: parent.Attempt.ManagerID, AgentManagerID: parent.Attempt.AgentManagerID, Phase: parent.Phase, ContextDigest: rawContentDigest(request.Context), RequestDigest: rawDigest, RequestContractDigest: parent.Attempt.RequestContractDigest, ConfigFingerprint: requestFP, RuntimeFileDigest: parent.Attempt.RuntimeFileDigest, LimitsDigest: parent.Attempt.LimitsDigest, AcceptedTargetHead: parent.Attempt.AcceptedTargetHead, Bounds: managerStageBounds(spec.Agent, spec.Config), StartedAt: time.Now().UTC(), Status: "started", ReviewRound: round, ReviewSubjectDigest: parent.Attempt.ReportRef.Digest, ReviewPolicyDigest: parent.Attempt.ReviewPolicyDigest}}
	result, log, invokeErr := invokeManagerAttempt(ctx, journal, log, request, spec.Agent, func(call context.Context) (agentexec.RunResult, error) {
		return dispatchReadOnlyManagerStage(call, host, invoker, spec.SourceRoot, spec.TargetRoot, runID, id, spec.Config, request, spec.RuntimeLimits)
	})
	if invokeErr != nil {
		return ManagerStageReviewResult{}, invokeErr
	}
	return finishManagerStageReview(host, journal, parent, log, result, spec, protocol, refs, scope, budget)
}

func finishManagerStageReview(host Host, journal *managerStageJournal, parent, log InvocationLog, result agentexec.RunResult, spec ManagerStageReviewSpec, protocol string, refs []reviewFileRef, scope string, budget ManagerStageBudget) (ManagerStageReviewResult, error) {
	if result.Delta != nil && len(result.Delta.Changes) > 0 {
		return ManagerStageReviewResult{}, errors.New("read-only Manager reviewer changed source files")
	}
	parsed, findings, err := validatePreparedManagerReview(result, protocol, refs)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	if err := host.ValidateManagerStage(spec.Validation); err != nil {
		return ManagerStageReviewResult{}, err
	}
	report := journal.report
	_, _, elapsed, cost, unknown := managerStageUsage(*report)
	if unknown || elapsed > budget.MaxDuration || cost > budget.MaxCostMicros {
		return ManagerStageReviewResult{}, errors.New("Manager review exceeded shared session accounting")
	}
	record := ReviewRecord{TaskID: parent.TaskID, ManagerID: parent.Attempt.ManagerID, Round: log.Attempt.ReviewRound, Phase: parent.Phase, CandidateID: parent.Attempt.ID, CandidateDigest: parent.Attempt.ReportRef.Digest, ScopeDigest: scope, InputDigest: protocol, Outcome: parsed.Status, Findings: findings, Receipt: result.Receipt, CostMicros: log.CostMicros, CostKnown: log.CostKnown, CostOverflow: log.CostOverflow, At: time.Now().UTC()}
	journal.mu.Lock()
	for i, row := range report.Invocations {
		if row.Attempt != nil && row.Attempt.ID == log.Attempt.ID {
			a := *row.Attempt
			a.Status = "accepted"
			a.ValidationDigest = scope
			row.Attempt = &a
			report.Invocations[i] = row
			break
		}
	}
	report.Reviews = append(report.Reviews, record)
	err = persistState(journal.store, report)
	journal.mu.Unlock()
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	current, err := journal.store.readLatestState(report.ID)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	return ManagerStageReviewResult{ReviewRecord: record, RunID: report.ID, ParentAttemptID: parent.Attempt.ID, SubjectDigest: parent.Attempt.ReportRef.Digest, StateDigest: current.Digest, Status: parsed.Status}, nil
}

func recoverManagerStageReview(ctx context.Context, host Host, invoker Invoker, store *runStore, report *RunReport, parent, log InvocationLog, spec ManagerStageReviewSpec) (ManagerStageReviewResult, error) {
	a := log.Attempt
	if a.ReviewSubjectDigest != parent.Attempt.ReportRef.Digest || a.ReviewPolicyDigest != parent.Attempt.ReviewPolicyDigest || a.Bounds != managerStageBounds(spec.Agent, spec.Config) || a.SessionDigest != spec.Validation.SessionDigest || a.RuntimeFileDigest != spec.Validation.RuntimeFileDigest {
		return ManagerStageReviewResult{}, errors.New("original reviewer subject/policy/bounds changed; no replay")
	}
	if a.Status == "transport-failed" || a.Status == "rejected" {
		return ManagerStageReviewResult{}, errors.New("original reviewer failed; no automatic reviewer replay")
	}
	raw, err := readStageEvidence(store, report.ID, parent.Attempt.ReportRef)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	request, refs, scope, err := managerStageReviewRequest(parent, report.ID, spec, a.ReviewRound, raw)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	currentDigest, err := digest(request)
	if err != nil || currentDigest != a.RequestDigest {
		return ManagerStageReviewResult{}, errors.New("original reviewer input is stale")
	}
	originalData, err := readStageEvidence(store, report.ID, a.RequestRef)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	var original agentexec.Request
	if err := json.Unmarshal(originalData, &original); err != nil {
		return ManagerStageReviewResult{}, err
	}
	protocol, err := nativeRequestInputDigest(original)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	budget := frozenManagerStageBudget(report.ManagerStageBinding.Budget, spec.Limits)
	limits := spec.RuntimeLimits
	limits.MaxStarts = budget.MaxStarts
	journal, err := newManagerStageJournal(store, report, limits)
	if err != nil {
		return ManagerStageReviewResult{}, err
	}
	var result agentexec.RunResult
	if a.Status == "started" || a.Status == "uncertain" {
		if spec.Config.Transport != projectconfig.TransportCodexAppServer {
			return ManagerStageReviewResult{}, errors.New("original reviewer is uncertain; no replay")
		}
		recovered, recoveryErr := recoverReadOnlyManagerStage(ctx, host, invoker, spec.SourceRoot, report.ID, a.ID, spec.Config, original, spec.RuntimeLimits, log.Receipt)
		result = recovered
		log.CostMicros, log.CostKnown, log.CostOverflow = managerStageCost(result.Receipt.Usage, a.Bounds)
		log, err = journal.terminal(log, result, recoveryErr)
		if err != nil {
			return ManagerStageReviewResult{}, err
		}
		if recoveryErr != nil || log.Attempt.Status != "transport-succeeded" {
			return ManagerStageReviewResult{}, errors.New("original reviewer terminal remains unresolved; no replay")
		}
	} else {
		result, err = readStageResult(store, *report, log)
		if err != nil {
			return ManagerStageReviewResult{}, err
		}
	}
	return finishManagerStageReview(host, journal, parent, log, result, spec, protocol, refs, scope, budget)
}

func validateStageReviewRecords(store *runStore, report RunReport) error {
	if report.ManagerStageBinding == nil {
		return nil
	}
	for _, record := range report.Reviews {
		var row InvocationLog
		for _, log := range report.Invocations {
			if log.Attempt != nil && log.Attempt.ParentAttemptID == record.CandidateID && log.Receipt.RunID == record.Receipt.RunID {
				row = log
				break
			}
		}
		if row.Attempt == nil || row.Attempt.Status != "accepted" || row.Attempt.ReviewSubjectDigest != record.CandidateDigest || row.Attempt.ValidationDigest != record.ScopeDigest || row.Attempt.ManagerID != record.ManagerID || row.TaskID != record.TaskID || row.Phase != record.Phase || row.Attempt.ReviewRound != record.Round || row.CostMicros != record.CostMicros || row.CostKnown != record.CostKnown || row.CostOverflow != record.CostOverflow {
			return errors.New("Manager ReviewRecord differs from its original attempt")
		}
		output, err := readStageResult(store, report, row)
		if err != nil {
			return err
		}
		input, err := readStageEvidence(store, report.ID, row.Attempt.RequestRef)
		if err != nil {
			return err
		}
		var request agentexec.Request
		if err := json.Unmarshal(input, &request); err != nil {
			return err
		}
		var reviewContext reviewerContext
		if err := json.Unmarshal(request.Context, &reviewContext); err != nil {
			return err
		}
		protocol, err := nativeRequestInputDigest(request)
		if err != nil || protocol != record.InputDigest {
			return errors.New("Manager ReviewRecord input differs")
		}
		parsed, findings, err := validatePreparedManagerReview(output, protocol, reviewContext.CandidateFiles)
		if err != nil {
			return err
		}
		a, _ := digest(findings)
		b, _ := digest(record.Findings)
		x, _ := digest(output.Receipt)
		y, _ := digest(record.Receipt)
		if parsed.Status != record.Outcome || a != b || x != y {
			return errors.New("Manager ReviewRecord verdict differs from retained response")
		}
	}
	return nil
}
