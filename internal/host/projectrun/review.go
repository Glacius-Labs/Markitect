package projectrun

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
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
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Digest string `json:"digest"`
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
		if remaining < config.Timeout {
			config.Timeout = remaining
		}
	}
	accepted, err := scopedReviewModel(project.Report, task.ManagerID, plan.Managers)
	if err != nil {
		return record, log, err
	}
	files := scopedCandidateFiles(project, task)
	if len(files) == 0 {
		// A legitimate no-op can have no owned file in inventory. Keep an empty
		// array explicit; the candidate digest still binds the review.
		files = []agentexec.Artifact{}
	}
	fileRefs := make([]reviewFileRef, 0, len(files))
	for _, file := range files {
		fileRefs = append(fileRefs, reviewFileRef{Path: file.Path, Mode: file.Mode, Digest: file.Digest})
	}
	responseSchema := reviewResponseSchema()
	contextJSON, err := json.Marshal(struct {
		Kind            string                      `json:"kind"`
		RunGoal         string                      `json:"runGoal"`
		ManagerID       string                      `json:"managerId"`
		OwnTask         string                      `json:"ownTask"`
		Phase           string                      `json:"phase"`
		Round           int                         `json:"round"`
		CandidateID     string                      `json:"candidateId"`
		CandidateDigest string                      `json:"candidateDigest"`
		ChangedPaths    []string                    `json:"changedPaths"`
		AcceptedModel   projectmodel.ManagerContext `json:"acceptedModel"`
		ScopedModel     struct {
			Statements []projectmodel.Statement `json:"statements"`
			Artifacts  []projectmodel.Artifact  `json:"artifacts"`
			OwnedPaths []string                 `json:"ownedPaths"`
		} `json:"scopedModel"`
		CandidateFiles []reviewFileRef `json:"candidateFiles"`
		ResponseSchema json.RawMessage `json:"responseSchema"`
	}{Kind: "projectrun-review/v1", RunGoal: plan.Goal, ManagerID: task.ManagerID, OwnTask: task.Goal, Phase: phase, Round: round,
		CandidateID: candidate.ID, CandidateDigest: candidate.Digest, ChangedPaths: unionPaths(task.WrittenPaths, task.IntegratedPaths), AcceptedModel: accepted,
		ScopedModel: struct {
			Statements []projectmodel.Statement `json:"statements"`
			Artifacts  []projectmodel.Artifact  `json:"artifacts"`
			OwnedPaths []string                 `json:"ownedPaths"`
		}{Statements: append([]projectmodel.Statement(nil), accepted.Statements...), Artifacts: append([]projectmodel.Artifact(nil), accepted.Artifacts...), OwnedPaths: reviewScopePaths(files)},
		CandidateFiles: fileRefs, ResponseSchema: responseSchema})
	if err != nil {
		return record, log, err
	}
	scopeIDs := []string{task.ManagerID}
	for _, statement := range accepted.Statements {
		scopeIDs = append(scopeIDs, "statement:"+statement.ID)
	}
	for _, artifact := range accepted.Artifacts {
		scopeIDs = append(scopeIDs, "artifact:"+artifact.ID)
	}
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
	if onStart != nil {
		if err := onStart(log); err != nil {
			return record, log, fmt.Errorf("persist reviewer start: %w", err)
		}
	}
	result, invokeErr := invoker.Run(ctx, config, request, agentexec.RunOptions{PrivateLogDirectory: filepath.Join(root, ".markitect", "runs", "private")})
	log = InvocationLog{TaskID: task.ID, Role: "reviewer", Phase: "review", InputDigest: inputDigest,
		Receipt: result.Receipt, ReportID: result.Receipt.RunID, Outcome: result.Receipt.Outcome}
	if usageCost, known := estimateCost(result.Receipt.Usage, agent.Pricing); known {
		log.CostMicros = usageCost
	} else if invokeErr == nil {
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
	findings, err := validateReviewFindings(parsed.Findings, accepted, files)
	if err != nil {
		return record, log, err
	}
	scopeDigest, err := reviewScopeDigest(plan, project, task, phase)
	if err != nil {
		return record, log, err
	}
	record = ReviewRecord{TaskID: task.ID, ManagerID: task.ManagerID, Round: round, Phase: phase, CandidateID: candidate.ID,
		CandidateDigest: candidate.Digest, ScopeDigest: scopeDigest, InputDigest: inputDigest, Outcome: parsed.Status, Findings: findings,
		Receipt: result.Receipt, CostMicros: log.CostMicros, At: time.Now().UTC()}
	return record, log, nil
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
	accepted, err := scopedReviewModel(project.Report, task.ManagerID, plan.Managers)
	if err != nil {
		return "", err
	}
	files := scopedCandidateFiles(project, task)
	return digest(struct {
		Kind          string                      `json:"kind"`
		RunGoal       string                      `json:"runGoal"`
		ManagerID     string                      `json:"managerId"`
		OwnTask       string                      `json:"ownTask"`
		Phase         string                      `json:"phase"`
		AcceptedModel projectmodel.ManagerContext `json:"acceptedModel"`
		Checks        []string                    `json:"checks"`
		ChangedPaths  []string                    `json:"changedPaths"`
		Files         []agentexec.Artifact        `json:"files"`
	}{"projectrun-review/v1", plan.Goal, task.ManagerID, task.Goal, phase, accepted, append([]string(nil), task.Checks...), unionPaths(task.WrittenPaths, task.IntegratedPaths), files})
}

func scopedReviewModel(report projectmodel.Report, managerID string, tasks []ManagerTask) (projectmodel.ManagerContext, error) {
	accepted, err := projectmodel.Context(report, managerID)
	if err != nil {
		return accepted, err
	}
	active := map[string]bool{}
	for _, id := range activeChildren(tasks, managerID) {
		active[id] = true
	}
	children := accepted.Children[:0]
	for _, child := range accepted.Children {
		if active[child.ID] {
			children = append(children, child)
		}
	}
	accepted.Children = children
	return accepted, nil
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
				if _, exists := project.Snapshot.Files[path]; exists && projectPathAllowed(project.Config, path) {
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

func reviewGrounding(model projectmodel.ManagerContext) map[string]bool {
	allowed := map[string]bool{}
	for _, statement := range model.Statements {
		if statement.ID != "" {
			allowed["statement:"+statement.ID] = true
		}
	}
	for _, artifact := range model.Artifacts {
		for _, path := range artifact.Paths {
			allowed["artifact-path:"+path] = true
		}
	}
	return allowed
}

func validateReviewFindings(raw []reviewFindingResponse, accepted projectmodel.ManagerContext, files []agentexec.Artifact) ([]ReviewFinding, error) {
	allowedGrounding := reviewGrounding(accepted)
	allowedPaths := map[string]bool{}
	for _, file := range files {
		allowedPaths[file.Path] = true
	}
	findings := make([]ReviewFinding, 0, len(raw))
	for _, finding := range raw {
		if !allowedPaths[finding.Path] || strings.TrimSpace(finding.Expectation) == "" || len(finding.Expectation) > 2048 || !allowedGrounding[finding.Grounding] {
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
		if !reviewRequired(finalProject, task) {
			continue
		}
		phase := "work"
		if len(activeChildren(run.Tasks, task.ManagerID)) > 0 {
			phase = "integrate"
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
