package projectrun

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

type reviewFindingResponse struct {
	Path        string `json:"path"`
	Expectation string `json:"expectation"`
	Grounding   string `json:"grounding"`
}

type reviewResponse struct {
	Status   string                  `json:"status"`
	Summary  string                  `json:"summary"`
	Findings []reviewFindingResponse `json:"findings"`
}

type reviewFileRef struct {
	Path      string   `json:"path"`
	Mode      string   `json:"mode"`
	Digest    string   `json:"digest"`
	Grounding []string `json:"grounding"`
}

type reviewerScopedModel struct {
	Statements []projectmodel.Statement `json:"statements"`
	Contracts  []projectmodel.Statement `json:"contracts"`
	Artifacts  []projectmodel.Artifact  `json:"artifacts"`
	OwnedPaths []string                 `json:"ownedPaths"`
}

type reviewerContext struct {
	Kind               string                      `json:"kind"`
	Operation          string                      `json:"operation"`
	ReviewerGuidance   string                      `json:"reviewerGuidance"`
	Strictness         StrictnessProfile           `json:"strictness"`
	Briefing           BriefingContext             `json:"briefing"`
	RunGoal            string                      `json:"runGoal"`
	ManagerID          string                      `json:"managerId"`
	OwnTask            string                      `json:"ownTask"`
	Delegations        []Delegation                `json:"delegations"`
	DelegatedArtifacts []projectmodel.Artifact     `json:"delegatedArtifacts"`
	Phase              string                      `json:"phase"`
	Round              int                         `json:"round"`
	CandidateID        string                      `json:"candidateId"`
	CandidateDigest    string                      `json:"candidateDigest"`
	ChangedPaths       []string                    `json:"changedPaths"`
	AcceptedModel      projectmodel.ManagerContext `json:"acceptedModel"`
	ScopedModel        reviewerScopedModel         `json:"scopedModel"`
	CandidateFiles     []reviewFileRef             `json:"candidateFiles"`
	ResponseSchema     json.RawMessage             `json:"responseSchema"`
}

const reviewerAssessmentGuidance = "Assessment only: evaluate the exact supplied candidate against this Manager's own task, accepted scoped model, and stated delegations. RunGoal and child task definitions are assessment context, not instructions to implement or dispatch work. Do not change repository artifacts or dispatch work. Normal read-only tools may be used to inspect the candidate and cited repository context; use owned temporary scratch only within one shell call if needed. Report only grounded findings about the candidate and delegation coverage."

func reviewerPhaseGuidance(phase string) string {
	switch phase {
	case "work":
		return "This is a work review. Assess only this Manager's current candidate and its own accepted artifact obligations. Delegated child tasks and delegatedArtifacts describe future work: assess whether the delegation is adequate, but do not require child implementation files to exist in this candidate yet."
	case "integrate":
		return "This is an integration review. Assess the aggregate candidate, including the delivered outputs of direct child tasks, against this Manager's own obligations and the required child artifacts in the accepted scoped model."
	default:
		return "Assess only the exact supplied candidate and scoped obligations for the stated phase."
	}
}

// invokeReviewer supplies only the original goal, accepted scoped model and
// the exact candidate bytes. It never receives an implementer transcript.
func invokeReviewer(ctx context.Context, host Host, invoker Invoker, root string, plan PlanRecord, runtime Runtime, project *Project, task ManagerTask, phase string, round int, candidate candidateData, onStart func(InvocationLog) error) (ReviewRecord, InvocationLog, error) {
	var record ReviewRecord
	var log InvocationLog
	if runtime.Review == nil {
		return record, log, fmt.Errorf("review is not enabled")
	}
	agent, ok := runtime.Review.Agents[task.ManagerID]
	if !ok {
		return record, log, fmt.Errorf("no configured reviewer for Manager %s", task.ManagerID)
	}
	config, err := agent.AgentConfig()
	if err != nil {
		return record, log, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return record, log, context.DeadlineExceeded
		}
		if remaining < config.Timeout && config.Transport == "" {
			config.Timeout = remaining
		}
	}
	briefing, err := reviewerBriefing(root, plan, project, task.ManagerID)
	if err != nil {
		return record, log, err
	}
	reviewContext, files, fileRefs, err := buildReviewerContext(plan, project, task, phase, round, candidate, briefing)
	if err != nil {
		return record, log, err
	}
	accepted := reviewContext.AcceptedModel
	contextJSON, err := json.Marshal(reviewContext)
	if err != nil {
		return record, log, err
	}
	scopeIDs := reviewScopeIDs(task.ManagerID, accepted)
	acceptedDigest, err := digest(accepted)
	if err != nil {
		return record, log, err
	}
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: project.Revision,
		ModelDigest: acceptedDigest, ModulePin: acceptedDigest, ProjectionID: acceptedDigest,
		ScopeIDs: append([]string{}, uniqueSorted(scopeIDs)...), PolicyIDs: append([]string{}, uniqueSorted(task.Checks)...), Context: contextJSON, Artifacts: append([]agentexec.Artifact{}, files...)}
	if err := canonicalizeReviewContext(&request); err != nil {
		return record, log, err
	}
	inputDigest, err := digest(request)
	if err != nil {
		return record, log, err
	}
	log = InvocationLog{TaskID: task.ID, Role: "reviewer", Phase: "review", InputDigest: inputDigest, Outcome: "started"}
	result, recovered, invokeErr := recoverInvocationOnResume(ctx, host, invoker, root, fmt.Sprintf("%s-review-%s-%d", task.ID, phase, round), config, runtime.Limits, request)
	if !recovered && invokeErr == nil {
		if onStart != nil {
			if err := onStart(log); err != nil {
				return record, log, fmt.Errorf("persist reviewer start: %w", err)
			}
		}
		result, invokeErr = invokeProjectAgent(ctx, host, invoker, root, project, agent, fmt.Sprintf("%s-review-%s-%d", task.ID, phase, round), nil, nil, runtime.Limits, config, request)
	}
	if invokeErr == nil && result.Delta != nil && len(result.Delta.Changes) != 0 {
		invokeErr = fmt.Errorf("read-only reviewer changed its owned workspace")
	}
	log = InvocationLog{TaskID: task.ID, Role: "reviewer", Phase: "review", InputDigest: inputDigest,
		Receipt: result.Receipt, ReportID: result.Receipt.RunID, Outcome: result.Receipt.Outcome}
	usageCost, known, overflow := estimateCostDetailed(result.Receipt.Usage, agent.Pricing)
	log.CostMicros, log.CostKnown, log.CostOverflow = usageCost, known, overflow
	if overflow {
		return record, log, fmt.Errorf("reviewer cost estimate exceeds the supported int64 range")
	}
	if known {
	} else if invokeErr == nil && config.Transport != TransportCodexAppServer {
		return record, log, fmt.Errorf("reviewer usage is missing; bounded cost cannot be asserted")
	}
	if invokeErr != nil {
		return record, log, invokeErr
	}
	if err := freshBindings(host, invoker, root, plan, runtime); err != nil {
		return record, log, err
	}
	parsed, err := validateReviewerResponse(result.Response, result.Receipt, inputDigest)
	if err != nil {
		return record, log, err
	}
	findings, err := validateReviewFindings(parsed.Findings, fileRefs)
	if err != nil {
		return record, log, err
	}
	scopeDigest, err := reviewScopeDigest(plan, project, task, phase)
	if err != nil {
		return record, log, err
	}
	record = ReviewRecord{TaskID: task.ID, ManagerID: task.ManagerID, Round: round, Phase: phase, CandidateID: candidate.ID,
		CandidateDigest: candidate.Digest, ScopeDigest: scopeDigest, InputDigest: inputDigest, Outcome: parsed.Status, Findings: findings,
		Receipt: result.Receipt, CostMicros: log.CostMicros, CostKnown: log.CostKnown, CostOverflow: log.CostOverflow, At: time.Now().UTC()}
	return record, log, nil
}

func buildReviewerContext(plan PlanRecord, project *Project, task ManagerTask, phase string, round int, candidate candidateData, briefing BriefingContext) (reviewerContext, []agentexec.Artifact, []reviewFileRef, error) {
	files := reviewCandidateFiles(project, task, plan.Managers, phase)
	if len(files) == 0 {
		// Keep the empty list explicit for a legitimate no-op candidate.
		files = []agentexec.Artifact{}
	}
	accepted, err := scopedReviewModel(project.Report, task.ManagerID, plan.Managers, files, phase)
	if err != nil {
		return reviewerContext{}, nil, nil, err
	}
	fileRefs := reviewFileReferences(project.Report, accepted, files)
	return reviewerContext{Kind: "projectrun-review/v1", Operation: plan.Operation,
		ReviewerGuidance: reviewerAssessmentGuidance + " " + reviewerPhaseGuidance(phase), Strictness: plan.Strictness[task.ManagerID], Briefing: briefing,
		RunGoal: plan.Goal, ManagerID: task.ManagerID, OwnTask: task.Goal, Delegations: append([]Delegation{}, task.Delegations...), DelegatedArtifacts: reviewDelegatedArtifacts(project.Report, plan.Managers, task.ManagerID, phase), Phase: phase, Round: round,
		CandidateID: candidate.ID, CandidateDigest: candidate.Digest, ChangedPaths: reviewChangedPaths(task, plan.Managers, phase), AcceptedModel: accepted,
		ScopedModel:    reviewerScopedModel{Statements: append([]projectmodel.Statement(nil), accepted.Statements...), Contracts: append([]projectmodel.Statement(nil), accepted.Contracts...), Artifacts: append([]projectmodel.Artifact(nil), accepted.Artifacts...), OwnedPaths: reviewScopePaths(files)},
		CandidateFiles: fileRefs, ResponseSchema: reviewResponseSchema()}, files, fileRefs, nil
}

// reviewerBriefing binds accepted change context to the plan. A draft ModelEdit
// has not been accepted, so it must not inherit history from the target model.
func reviewerBriefing(root string, plan PlanRecord, project *Project, managerID string) (BriefingContext, error) {
	if plan.ModelEdit != nil {
		if plan.BriefingDigests[managerID] != "" {
			return BriefingContext{}, ErrStale
		}
		return BriefingContext{Briefings: []projectbriefing.Briefing{}, Events: []projectbriefing.Event{}}, nil
	}
	briefing, err := managerBriefing(root, project.Report.ModelDigest, managerID, project.Revision)
	if err != nil {
		return BriefingContext{}, err
	}
	if briefing.Digest != plan.BriefingDigests[managerID] {
		return BriefingContext{}, ErrStale
	}
	return briefing, nil
}

func validateReviewerResponse(response agentexec.Response, receipt agentexec.Receipt, expectedDigest string) (reviewResponse, error) {
	if response.Role != agentexec.RoleExecutor || response.InputDigest != expectedDigest || receipt.InputDigest != expectedDigest {
		return reviewResponse{}, fmt.Errorf("reviewer response is not bound to the exact request")
	}
	if len(response.CandidateFiles) != 0 || len(response.CandidateJSON) != 0 || len(response.VerifierObservations) != 0 {
		return reviewResponse{}, fmt.Errorf("reviewer returned output outside its read-only report contract")
	}
	if response.Outcome != agentexec.OutcomeProposed || receipt.Outcome != agentexec.OutcomeProposed {
		return reviewResponse{}, fmt.Errorf("reviewer invocation did not complete with a proposed typed report")
	}
	return decodeReviewResponse(response.ReportJSON)
}

func canonicalizeReviewContext(request *agentexec.Request) error {
	decoder := json.NewDecoder(strings.NewReader(string(request.Context)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode review context for canonical request binding: %w", err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("canonicalize review context for request binding: %w", err)
	}
	request.Context = canonical
	return nil
}

// reviewScopeDigest deliberately excludes the global candidate identity so a
// parent can merge an independently reviewed sibling without invalidating it.
// It includes the accepted local contract, task goal, review phase and every
// actual owned file byte and mode supplied to the reviewer.
func reviewScopeDigest(plan PlanRecord, project *Project, task ManagerTask, phase string) (string, error) {
	files := reviewCandidateFiles(project, task, plan.Managers, phase)
	accepted, err := scopedReviewModel(project.Report, task.ManagerID, plan.Managers, files, phase)
	if err != nil {
		return "", err
	}
	fileRefs := reviewFileReferences(project.Report, accepted, files)
	return digest(struct {
		Kind               string                      `json:"kind"`
		Operation          string                      `json:"operation"`
		Strictness         StrictnessProfile           `json:"strictness"`
		BriefingDigest     string                      `json:"briefingDigest"`
		RunGoal            string                      `json:"runGoal"`
		ManagerID          string                      `json:"managerId"`
		OwnTask            string                      `json:"ownTask"`
		Delegations        []Delegation                `json:"delegations"`
		DelegatedArtifacts []projectmodel.Artifact     `json:"delegatedArtifacts"`
		ReviewerGuidance   string                      `json:"reviewerGuidance"`
		Phase              string                      `json:"phase"`
		AcceptedModel      projectmodel.ManagerContext `json:"acceptedModel"`
		Checks             []string                    `json:"checks"`
		ChangedPaths       []string                    `json:"changedPaths"`
		Files              []agentexec.Artifact        `json:"files"`
		FileRefs           []reviewFileRef             `json:"fileRefs"`
	}{"projectrun-review/v2", plan.Operation, plan.Strictness[task.ManagerID], plan.BriefingDigests[task.ManagerID], plan.Goal, task.ManagerID, task.Goal, append([]Delegation(nil), task.Delegations...), reviewDelegatedArtifacts(project.Report, plan.Managers, task.ManagerID, phase), reviewerAssessmentGuidance + " " + reviewerPhaseGuidance(phase), phase, accepted, append([]string(nil), task.Checks...), reviewChangedPaths(task, plan.Managers, phase), files, fileRefs})
}

func scopedReviewModel(report projectmodel.Report, managerID string, tasks []ManagerTask, files []agentexec.Artifact, phase string) (projectmodel.ManagerContext, error) {
	accepted, err := projectmodel.Context(report, managerID)
	if err != nil {
		return accepted, err
	}
	activeIDs := activeChildren(tasks, managerID)
	active := map[string]bool{}
	for _, id := range activeIDs {
		active[id] = true
	}
	children := accepted.Children[:0]
	for _, child := range accepted.Children {
		if active[child.ID] {
			children = append(children, child)
		}
	}
	accepted.Children = children

	statementByID := make(map[string]projectmodel.Statement, len(report.Statements))
	for _, statement := range report.Statements {
		statementByID[statement.ID] = statement
	}
	visibleStatements := make(map[string]bool, len(accepted.Statements)+len(accepted.Contracts))
	for _, statement := range accepted.Statements {
		visibleStatements[statement.ID] = true
	}
	contractByID := make(map[string]projectmodel.Statement)
	for _, contract := range accepted.Contracts {
		contractByID[contract.ID] = contract
	}
	addContract := func(statement projectmodel.Statement) {
		if statement.ID == "" || visibleStatements[statement.ID] {
			return
		}
		contractByID[statement.ID] = statement
	}
	// An admitted candidate file can implement an artifact or public statement
	// owned by another Manager. Carry only the interfaces explicitly related to
	// these exact files; never expand to the foreign Manager's full model.
	selectedPaths := make(map[string]bool, len(files))
	for _, file := range files {
		selectedPaths[file.Path] = true
	}
	relatedArtifacts := make(map[string]bool)
	for _, entry := range report.Files {
		if !selectedPaths[entry.Path] {
			continue
		}
		for _, id := range entry.Artifacts {
			relatedArtifacts[id] = true
		}
		for _, id := range entry.Statements {
			if statement, ok := statementByID[id]; ok && statement.Public {
				addContract(statement)
			}
		}
	}
	for _, artifact := range report.Artifacts {
		for _, path := range artifact.Paths {
			if selectedPaths[path] {
				relatedArtifacts[artifact.ID] = true
				break
			}
		}
	}
	requiredChildArtifactIDs := make(map[string]bool)
	if phase == "integrate" {
		for _, artifact := range requiredChildArtifacts(report, activeIDs) {
			relatedArtifacts[artifact.ID] = true
			requiredChildArtifactIDs[artifact.ID] = true
		}
	}

	artifactByID := make(map[string]projectmodel.Artifact, len(accepted.Artifacts))
	for _, artifact := range accepted.Artifacts {
		artifactByID[artifact.ID] = artifact
	}
	for _, artifact := range report.Artifacts {
		if !relatedArtifacts[artifact.ID] {
			continue
		}
		artifactByID[artifact.ID] = artifact
		for _, id := range artifact.Realizes {
			if statement, ok := statementByID[id]; ok && statement.Public && (requiredChildArtifactIDs[artifact.ID] || selectedArtifactForReview(report, artifact.ID, selectedPaths)) {
				addContract(statement)
			}
		}
	}

	// Exported interface relations must not smuggle private or unrelated
	// statement identifiers into the review context.
	visibleContracts := make(map[string]bool, len(visibleStatements)+len(contractByID))
	for id := range visibleStatements {
		visibleContracts[id] = true
	}
	for id := range contractByID {
		visibleContracts[id] = true
	}
	accepted.Contracts = accepted.Contracts[:0]
	for _, contract := range contractByID {
		contract.Uses = filterVisibleStatementRefs(contract.Uses, visibleContracts)
		contract.Requires = filterVisibleStatementRefs(contract.Requires, visibleContracts)
		accepted.Contracts = append(accepted.Contracts, contract)
	}
	sort.Slice(accepted.Contracts, func(i, j int) bool { return accepted.Contracts[i].ID < accepted.Contracts[j].ID })
	accepted.Artifacts = accepted.Artifacts[:0]
	for _, artifact := range artifactByID {
		artifact.Realizes = filterVisibleStatementRefs(artifact.Realizes, visibleContracts)
		accepted.Artifacts = append(accepted.Artifacts, artifact)
	}
	sort.Slice(accepted.Artifacts, func(i, j int) bool { return accepted.Artifacts[i].ID < accepted.Artifacts[j].ID })
	return accepted, nil
}

func delegatedChildArtifacts(report projectmodel.Report, tasks []ManagerTask, managerID string) []projectmodel.Artifact {
	artifacts := requiredChildArtifacts(report, activeChildren(tasks, managerID))
	if artifacts == nil {
		return []projectmodel.Artifact{}
	}
	return artifacts
}

func reviewDelegatedArtifacts(report projectmodel.Report, tasks []ManagerTask, managerID, phase string) []projectmodel.Artifact {
	if phase != "work" {
		return []projectmodel.Artifact{}
	}
	return delegatedChildArtifacts(report, tasks, managerID)
}

// reviewCandidateFiles expands an integration review to include the actual
// delivered paths from direct children. Work reviews stay limited to this
// Manager's current candidate.
func reviewCandidateFiles(project *Project, task ManagerTask, tasks []ManagerTask, phase string) []agentexec.Artifact {
	files := scopedCandidateFiles(project, task)
	if phase != "integrate" || project == nil || project.Snapshot == nil {
		return files
	}
	selected := make(map[string]bool, len(files))
	for _, file := range files {
		selected[file.Path] = true
	}
	addExisting := func(path string) {
		if projectPathAllowed(project.Config, path) {
			if _, exists := project.Snapshot.Files[path]; exists {
				selected[path] = true
			}
		}
	}
	for _, child := range tasks {
		if child.ParentTask == task.ManagerID {
			for _, path := range unionPaths(child.WrittenPaths, child.IntegratedPaths) {
				addExisting(path)
			}
		}
	}
	for _, artifact := range requiredChildArtifacts(project.Report, activeChildren(tasks, task.ManagerID)) {
		for _, path := range artifact.Paths {
			addExisting(path)
		}
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files = make([]agentexec.Artifact, 0, len(paths))
	for _, path := range paths {
		content := append([]byte(nil), project.Snapshot.Files[path]...)
		mode := protocolMode(project.Snapshot.Modes[path])
		if mode == "" {
			mode = "0644"
		}
		files = append(files, agentexec.Artifact{Path: path, Mode: mode, Digest: rawContentDigest(content), Content: content})
	}
	return files
}

func reviewChangedPaths(task ManagerTask, tasks []ManagerTask, phase string) []string {
	paths := unionPaths(task.WrittenPaths, task.IntegratedPaths)
	if phase != "integrate" {
		return paths
	}
	for _, child := range tasks {
		if child.ParentTask == task.ManagerID {
			paths = unionPaths(paths, child.WrittenPaths, child.IntegratedPaths)
		}
	}
	return paths
}

func selectedArtifactForReview(report projectmodel.Report, artifactID string, selectedPaths map[string]bool) bool {
	for _, entry := range report.Files {
		if !selectedPaths[entry.Path] {
			continue
		}
		for _, id := range entry.Artifacts {
			if id == artifactID {
				return true
			}
		}
	}
	for _, artifact := range report.Artifacts {
		if artifact.ID != artifactID {
			continue
		}
		for _, path := range artifact.Paths {
			if selectedPaths[path] {
				return true
			}
		}
	}
	return false
}

func filterVisibleStatementRefs(ids []string, visible map[string]bool) []string {
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		if visible[id] {
			filtered = append(filtered, id)
		}
	}
	return uniqueSorted(filtered)
}

func reviewScopeIDs(managerID string, accepted projectmodel.ManagerContext) []string {
	ids := []string{managerID}
	for _, statement := range accepted.Statements {
		ids = append(ids, "statement:"+statement.ID)
	}
	for _, contract := range accepted.Contracts {
		ids = append(ids, "statement:"+contract.ID)
	}
	for _, artifact := range accepted.Artifacts {
		ids = append(ids, "artifact:"+artifact.ID)
	}
	return uniqueSorted(ids)
}

func scopedCandidateFiles(project *Project, task ManagerTask) []agentexec.Artifact {
	if project == nil || project.Snapshot == nil {
		return nil
	}
	selected := map[string]bool{}
	for _, entry := range project.Report.Files {
		if entry.Owner == task.ManagerID && projectPathAllowed(project.Config, entry.Path) {
			selected[entry.Path] = true
		}
	}
	for _, artifactID := range task.Artifacts {
		for _, artifact := range project.Report.Artifacts {
			if artifact.ID != artifactID {
				continue
			}
			for _, path := range artifact.Paths {
				owner, known := ownerForPath(project.Report, path)
				if _, exists := project.Snapshot.Files[path]; exists && projectPathAllowed(project.Config, path) && known && owner == task.ManagerID {
					selected[path] = true
				}
			}
		}
	}
	for _, path := range task.IntegratedPaths {
		if _, exists := project.Snapshot.Files[path]; exists && projectPathAllowed(project.Config, path) {
			selected[path] = true
		}
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		if _, exists := project.Snapshot.Files[path]; exists {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	files := make([]agentexec.Artifact, 0, len(paths))
	for _, path := range paths {
		mode := protocolMode(project.Snapshot.Modes[path])
		if mode == "" {
			mode = "0644"
		}
		content := append([]byte(nil), project.Snapshot.Files[path]...)
		files = append(files, agentexec.Artifact{Path: path, Mode: mode, Digest: rawContentDigest(content), Content: content})
	}
	return files
}

// reviewRequired distinguishes pure routing from work with reviewable scope.
// Declared artifacts and paths changed at any point remain mandatory even if
// the final snapshot has no bytes at those paths (for example, a deletion).
func reviewRequired(project *Project, task ManagerTask) bool {
	return project != nil && (len(scopedCandidateFiles(project, task)) > 0 || len(task.Artifacts) > 0 || len(task.WrittenPaths) > 0 || len(task.IntegratedPaths) > 0)
}

func reviewScopePaths(files []agentexec.Artifact) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func reviewFileReferences(report projectmodel.Report, accepted projectmodel.ManagerContext, files []agentexec.Artifact) []reviewFileRef {
	mandateContracts := map[string]bool{}
	if mandate, err := projectmodel.Context(report, accepted.Manager.ID); err == nil {
		for _, contract := range mandate.Contracts {
			mandateContracts[contract.ID] = true
		}
	}
	acceptedStatements := make(map[string]bool, len(accepted.Statements)+len(accepted.Contracts))
	for _, statement := range accepted.Statements {
		acceptedStatements[statement.ID] = true
	}
	for _, contract := range accepted.Contracts {
		acceptedStatements[contract.ID] = true
	}
	artifactByID := make(map[string]projectmodel.Artifact, len(accepted.Artifacts))
	for _, artifact := range accepted.Artifacts {
		artifactByID[artifact.ID] = artifact
	}
	refs := make([]reviewFileRef, 0, len(files))
	for _, file := range files {
		grounding := map[string]bool{}
		for _, statement := range accepted.Statements {
			if statement.ID != "" {
				grounding["statement:"+statement.ID] = true
			}
		}
		for id := range mandateContracts {
			grounding["statement:"+id] = true
		}
		relatedArtifacts := map[string]bool{}
		for _, entry := range report.Files {
			if entry.Path != file.Path {
				continue
			}
			for _, id := range entry.Statements {
				if acceptedStatements[id] {
					grounding["statement:"+id] = true
				}
			}
			for _, id := range entry.Artifacts {
				relatedArtifacts[id] = true
			}
		}
		for id, artifact := range artifactByID {
			coversPath := false
			for _, path := range artifact.Paths {
				if path == file.Path || strings.HasSuffix(path, "/") && strings.HasPrefix(file.Path, path) {
					grounding["artifact-path:"+path] = true
					coversPath = true
				}
			}
			if !relatedArtifacts[id] && !coversPath {
				continue
			}
			for _, statementID := range artifact.Realizes {
				if acceptedStatements[statementID] {
					grounding["statement:"+statementID] = true
				}
			}
		}
		values := make([]string, 0, len(grounding))
		for value := range grounding {
			values = append(values, value)
		}
		sort.Strings(values)
		refs = append(refs, reviewFileRef{Path: file.Path, Mode: file.Mode, Digest: file.Digest, Grounding: values})
	}
	return refs
}

func validateReviewFindings(raw []reviewFindingResponse, refs []reviewFileRef) ([]ReviewFinding, error) {
	allowedPaths := map[string]bool{}
	allowedGrounding := make(map[string]map[string]bool, len(refs))
	for _, ref := range refs {
		allowedPaths[ref.Path] = true
		allowedGrounding[ref.Path] = make(map[string]bool, len(ref.Grounding))
		for _, grounding := range ref.Grounding {
			allowedGrounding[ref.Path][grounding] = true
		}
	}
	findings := make([]ReviewFinding, 0, len(raw))
	for _, finding := range raw {
		if !allowedPaths[finding.Path] || strings.TrimSpace(finding.Expectation) == "" || len(finding.Expectation) > 2048 || !allowedGrounding[finding.Path][finding.Grounding] {
			return nil, fmt.Errorf("reviewer finding is ungrounded or outside the exact candidate scope")
		}
		findings = append(findings, ReviewFinding{Path: finding.Path, Expectation: finding.Expectation, Grounding: finding.Grounding})
	}
	return findings, nil
}

func reviewCount(reviews []ReviewRecord, managerID, phase string) int {
	count := 0
	for _, review := range reviews {
		if review.ManagerID == managerID && review.Phase == phase {
			count++
		}
	}
	return count
}

func routeReviewFindings(reviewed ManagerTask, review ReviewRecord, model projectmodel.Report, tasks []ManagerTask) (string, []ReworkRequest, error) {
	if len(review.Findings) == 0 {
		return "", nil, fmt.Errorf("failed review for %s has no actionable findings", reviewed.ManagerID)
	}
	byTarget := map[string][]string{}
	requester := reviewed.ManagerID
	for _, finding := range review.Findings {
		owner, _ := ownerForPath(model, finding.Path)
		if owner == "" {
			owner = reviewed.ManagerID
		}
		target := findTask(tasks, owner)
		if target == nil {
			target = findTask(tasks, reviewed.ManagerID)
		}
		if target == nil {
			return "", nil, fmt.Errorf("review finding path %s has no active Manager owner", finding.Path)
		}
		if owner == reviewed.ManagerID || target.ManagerID == reviewed.ManagerID {
			requester = reviewed.ParentTask
			if requester == "" {
				return "", nil, fmt.Errorf("review findings in root Manager %s scope cannot be routed to a direct child", reviewed.ManagerID)
			}
		} else {
			cursor := target
			for cursor != nil && cursor.ParentTask != reviewed.ManagerID {
				cursor = findTask(tasks, cursor.ParentTask)
			}
			if cursor == nil {
				return "", nil, fmt.Errorf("review finding owner %s is outside Manager %s's subtree", owner, reviewed.ManagerID)
			}
			target = cursor
		}
		byTarget[target.ManagerID] = append(byTarget[target.ManagerID], fmt.Sprintf("%s: %s (%s)", finding.Path, finding.Expectation, finding.Grounding))
	}
	ids := make([]string, 0, len(byTarget))
	for id := range byTarget {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	requests := make([]ReworkRequest, 0, len(ids))
	for _, id := range ids {
		target := findTask(tasks, id)
		if target == nil {
			return "", nil, fmt.Errorf("review rework target %s disappeared", id)
		}
		reason := boundedRepairDiagnostic(fmt.Errorf("independent final review requires correction: %s", strings.Join(byTarget[id], "; ")))
		requests = append(requests, ReworkRequest{ManagerID: id, Goal: target.Goal, Reason: reason})
	}
	return requester, requests, nil
}

func reviewResponseSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false,"required":["status","summary","findings"],"properties":{"status":{"type":"string","enum":["pass","fail"]},"summary":{"type":"string","minLength":1,"maxLength":4096},"findings":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["path","expectation","grounding"],"properties":{"path":{"type":"string","minLength":1,"maxLength":1024},"expectation":{"type":"string","minLength":1,"maxLength":2048},"grounding":{"type":"string","minLength":1,"maxLength":1024}}}}}}`)
}

func decodeReviewResponse(raw json.RawMessage) (reviewResponse, error) {
	var response reviewResponse
	if len(raw) == 0 {
		return response, fmt.Errorf("reviewer omitted the typed review report")
	}
	if err := validateExactObjectKeys(raw, map[string]bool{"status": true, "summary": true, "findings": true}); err != nil {
		return response, err
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return response, err
	}
	if err := validateArrayObjectKeys(wrapper["findings"], map[string]bool{"path": true, "expectation": true, "grounding": true}); err != nil {
		return response, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return response, err
	}
	if response.Status != "pass" && response.Status != "fail" || strings.TrimSpace(response.Summary) == "" || len(response.Summary) > 4096 || response.Findings == nil {
		return response, fmt.Errorf("reviewer report is incomplete or outside its bounded contract")
	}
	if response.Status == "pass" && len(response.Findings) != 0 || response.Status == "fail" && len(response.Findings) == 0 {
		return response, fmt.Errorf("reviewer report status does not match its findings")
	}
	return response, nil
}

func requireFreshReviews(host Host, root string, store *runStore, dir string, base *Project, candidate candidateData, plan PlanRecord, runtime Runtime, run RunReport) error {
	if runtime.Review == nil {
		return nil
	}
	finalProject, err := projectForCandidate(host, root, base.Snapshot, candidate)
	if err != nil {
		return err
	}
	for _, planned := range plan.Managers {
		task := planned
		if current := findTask(run.Tasks, planned.ManagerID); current != nil {
			task = *current
		}
		phase := "work"
		if len(activeChildren(run.Tasks, task.ManagerID)) > 0 {
			phase = "integrate"
		}
		if !reviewRequired(finalProject, task) {
			continue
		}
		scopeDigest, err := reviewScopeDigest(plan, finalProject, task, phase)
		if err != nil {
			return err
		}
		found := false
		for i := len(run.Reviews) - 1; i >= 0; i-- {
			review := run.Reviews[i]
			if review.ManagerID != task.ManagerID || review.Phase != phase || review.Outcome != "pass" || review.ScopeDigest != scopeDigest {
				continue
			}
			prior, err := store.readCandidate(dir, review.CandidateID)
			if err != nil || prior.Digest != review.CandidateDigest {
				return fmt.Errorf("review for Manager %s is not bound to a retained exact candidate", task.ManagerID)
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("Manager %s has no fresh passed %s review for its final candidate scope", task.ManagerID, phase)
		}
	}
	return nil
}
