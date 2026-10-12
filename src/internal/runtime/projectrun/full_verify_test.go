package projectrun

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const (
	fullVerifyHelperEnv  = "MARKITECT_FULL_VERIFY_HELPER"
	fullVerifyCheckEnv   = "MARKITECT_FULL_VERIFY_CHECK"
	fullVerifyBadEnv     = "MARKITECT_FULL_VERIFY_BAD_MANAGER"
	fullVerifyNoUsageEnv = "MARKITECT_FULL_VERIFY_NO_USAGE"
	fullVerifyLogEnv     = "MARKITECT_FULL_VERIFY_LOG"
)

// Process fixture for the same stdin/stdout agentexec protocol used by real
// adapters; it returns typed evidence reports without calling a model.
func TestFullVerifyHelperProcess(t *testing.T) {
	if os.Getenv(fullVerifyHelperEnv) != "1" {
		return
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(raw, &invocation); err != nil {
		t.Fatal(err)
	}
	var contextPayload struct {
		Kind    string `json:"kind"`
		Manager struct {
			Manager struct {
				ID string `json:"id"`
			} `json:"manager"`
		} `json:"manager"`
		RequiredSubjects []string              `json:"requiredSubjects"`
		ChildAssessments []fullChildAssessment `json:"childAssessments"`
		CheckResults     []CheckResult         `json:"checkResults"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextPayload); err != nil {
		t.Fatal(err)
	}
	if contextPayload.Kind != "projectrun-full-verify/v1" || contextPayload.Manager.Manager.ID == "" {
		t.Fatalf("unexpected context: %s", invocation.Request.Context)
	}
	if path := os.Getenv(fullVerifyLogEnv); path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(file).Encode(map[string]any{"managerId": contextPayload.Manager.Manager.ID, "context": contextPayload})
		_ = file.Close()
	}
	assessments := make([]FullAssessment, 0, len(contextPayload.RequiredSubjects))
	for _, subject := range contextPayload.RequiredSubjects {
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: "pass", Detail: "bound project snapshot inspected"})
	}
	status := "pass"
	if bad := os.Getenv(fullVerifyBadEnv); bad != "" && strings.Contains(contextPayload.Manager.Manager.ID, bad) {
		status = "incomplete"
		assessments = nil
	}
	report, err := json.Marshal(fullAuditResponse{Status: status, Summary: "completed bounded Manager audit", Assessments: assessments, Findings: []string{}, Counterexamples: []FullCounterexample{}})
	if err != nil {
		t.Fatal(err)
	}
	var usage *agentexec.Usage
	if noUsage := os.Getenv(fullVerifyNoUsageEnv); noUsage == "" || !strings.Contains(contextPayload.Manager.Manager.ID, noUsage) {
		usage = &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	}
	response, err := json.Marshal(agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: report, Uncertainty: []string{}, Usage: usage})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.Write(append(response, '\n'))
	os.Exit(0)
}

func TestFullVerifyCheckProcess(t *testing.T) {
	if os.Getenv(fullVerifyCheckEnv) != "1" {
		return
	}
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("snapshot check input %s missing: %v", path, err)
		}
	}
}

func TestFullVerifyReviewEvidenceIncludesOnlyCurrentParentIntegrationAndPreservesOriginalBindings(t *testing.T) {
	rootID, targetID, siblingID := "manager:root", "manager:target", "manager:sibling"
	report := projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: rootID, Name: "root"},
		{ID: targetID, Name: "target", Parent: rootID},
		{ID: siblingID, Name: "sibling", Parent: rootID},
		{ID: "unrelated-manager", Name: "unrelated", Parent: "other-root"},
	}}
	finalCandidateID := "candidate-final"
	finalCandidateDigest := "sha256:final-candidate"
	want := fullReviewEvidence{TaskID: "task-parent", ManagerID: rootID, Phase: "integrate",
		ReviewCandidateID: "candidate-reviewed", ReviewCandidateDigest: "sha256:reviewed",
		FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest,
		ScopeDigest: "sha256:scope", InputDigest: "sha256:input", ReceiptRunID: "review-run", Outcome: "pass", Findings: []ReviewFinding{}}
	got := fullAuditIntegrationReviewsForManager(report, targetID, finalCandidateID, finalCandidateDigest, []fullReviewEvidence{
		{ManagerID: targetID, Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
		{ManagerID: siblingID, Phase: "work", Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
		{ManagerID: rootID, Phase: "work", Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
		{ManagerID: rootID, Phase: "integrate", Outcome: "pass", FinalCandidateID: "stale-candidate", FinalCandidateDigest: "sha256:stale"},
		{ManagerID: rootID, Phase: "integrate", Outcome: "fail", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
		want,
		{ManagerID: "unrelated-manager", Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
	})
	if len(got) != 1 {
		t.Fatalf("review evidence was not limited to the current parent integration review: %+v", got)
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Fatalf("parent review provenance or findings changed: got=%+v want=%+v", got[0], want)
	}
	rootReview := want
	rootReview.TaskID, rootReview.ManagerID, rootReview.ReviewCandidateID = "task-root", rootID, "root-reviewed-partial-candidate"
	rootReview.ReviewCandidateDigest, rootReview.ReceiptRunID = "sha256:root-reviewed", "root-integration-receipt"
	rootReviews := fullAuditIntegrationReviewsForManager(report, rootID, finalCandidateID, finalCandidateDigest, []fullReviewEvidence{
		rootReview,
		{ManagerID: targetID, Phase: "integrate", Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
		{ManagerID: rootID, Phase: "work", Outcome: "pass", FinalCandidateID: finalCandidateID, FinalCandidateDigest: finalCandidateDigest},
	})
	if len(rootReviews) != 1 || !reflect.DeepEqual(rootReviews[0], rootReview) {
		t.Fatalf("top-level Manager did not receive its own exact integration review: %+v", rootReviews)
	}
}

func TestFreshReviewEvidenceForVerifySelectsCurrentScopeAndRetainsPartialCandidate(t *testing.T) {
	root := t.TempDir()
	runtimeBytes := writeFiniteRuntimeFile(t, root)
	targetID, artifactID := "manager:inventory", "artifact:inventory"
	documentPath := "src/inventory/docs/greeting.md"
	documentContent := []byte("The greeting is displayed to the caller.\n")
	implementationPath := "src/inventory/implementation.txt"
	project := &projectvalue.Project{
		Root: root, Revision: strings.Repeat("a", 40), Digest: strings.Repeat("a", 64),
		Config: projectvalue.Config{CoverageMode: "selected", InventoryRoots: []string{"src"}},
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{
			documentPath: documentContent, implementationPath: []byte("fixed implementation bytes\n"),
			projectconfig.RuntimePath: runtimeBytes,
		}, Modes: map[string]string{documentPath: snapshot.RegularMode, implementationPath: snapshot.RegularMode, projectconfig.RuntimePath: snapshot.RegularMode}},
		Report: projectmodel.Report{
			Managers:   []projectmodel.Manager{{ID: targetID, Name: "inventory"}},
			Statements: []projectmodel.Statement{{ID: "statement:inventory", Owner: targetID, Public: true, Description: "Own the inventory implementation."}},
			Artifacts:  []projectmodel.Artifact{{ID: artifactID, Owner: targetID, Required: true, Realizes: []string{"statement:inventory"}, Paths: []string{implementationPath, documentPath}}},
			Files: []projectmodel.FileEntry{
				{Path: implementationPath, Mode: snapshot.RegularMode, Owner: targetID, Class: "implementation", Artifacts: []string{artifactID}, Exists: true},
				{Path: documentPath, Mode: snapshot.RegularMode, Owner: targetID, Class: "documentation", Artifacts: []string{artifactID}, Exists: true},
			},
		},
	}
	task := ManagerTask{ID: "task-current", ManagerID: targetID, Goal: "Review the owned artifact.", Artifacts: []string{artifactID}}
	plan := PlanRecord{ID: "plan-current", Goal: "Complete the project artifact.", Operation: OperationApply,
		BaseRevision: project.Revision, Managers: []ManagerTask{task}, BriefingDigests: map[string]string{}, Strictness: map[string]projectconfig.StrictnessProfile{}}
	run := RunReport{ID: "0123456789abcdef0123456789abcdef", Tasks: []ManagerTask{task}}
	currentScope, err := reviewScopeDigest(plan, project, task, "work", run)
	if err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.runDir(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(store.base); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(dir); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(dir, "candidates")); err != nil {
		t.Fatal(err)
	}
	partial := candidateData{ID: "11111111111111111111111111111111", Files: map[string]File{}}
	if err := store.writeCandidate(dir, partial); err != nil {
		t.Fatal(err)
	}
	partial, err = store.readCandidate(dir, partial.ID)
	if err != nil {
		t.Fatal(err)
	}
	final := candidateData{ID: "22222222222222222222222222222222", Files: map[string]File{}}
	if err := store.writeCandidate(dir, final); err != nil {
		t.Fatal(err)
	}
	final, err = store.readCandidate(dir, final.ID)
	if err != nil {
		t.Fatal(err)
	}
	current := ReviewRecord{TaskID: task.ID, ManagerID: task.ManagerID, Phase: "work", CandidateID: partial.ID,
		CandidateDigest: partial.Digest, ScopeDigest: currentScope, InputDigest: "sha256:review-input", Outcome: "pass",
		Findings: []ReviewFinding{}, Receipt: agentexec.Receipt{RunID: "review-run-current"}}
	historicalCandidate := candidateData{ID: "33333333333333333333333333333333", Files: map[string]File{}}
	if err := store.writeCandidate(dir, historicalCandidate); err != nil {
		t.Fatal(err)
	}
	historicalCandidate, err = store.readCandidate(dir, historicalCandidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	historical := current
	historical.CandidateID, historical.CandidateDigest, historical.ScopeDigest, historical.Receipt.RunID = historicalCandidate.ID, historicalCandidate.Digest, "sha256:old-scope", "review-run-old"
	run.Reviews = []ReviewRecord{current, historical}
	host := Host{Load: func(_ string, revision string) (*projectvalue.Project, error) {
		if revision != project.Revision {
			return nil, errors.New("unexpected review revision")
		}
		return project, nil
	}}
	evidence, err := freshReviewEvidenceForVerify(store, dir, project, final, plan, projectconfig.Runtime{Review: &projectconfig.ReviewConfig{}}, run, host)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 1 {
		t.Fatalf("selected review count = %d, want 1: %+v", len(evidence), evidence)
	}
	got := evidence[0]
	if got.ReviewCandidateID != partial.ID || got.ReviewCandidateDigest != partial.Digest || got.FinalCandidateID != final.ID || got.FinalCandidateDigest != final.Digest {
		t.Fatalf("review and final candidate bindings were collapsed or changed: %+v", got)
	}
	if got.ReceiptRunID != current.Receipt.RunID || got.ScopeDigest != currentScope || !reflect.DeepEqual(got.Findings, current.Findings) {
		t.Fatalf("selected review provenance/findings differ from the exact current-scope record: got=%+v want=%+v", got, current)
	}
	documentDigest := rawContentDigest(documentContent)
	if len(got.ReviewedFiles) != 2 || got.ReviewedFiles[0].Path != documentPath || got.ReviewedFiles[0].Mode != "0644" || got.ReviewedFiles[0].ContentDigest != documentDigest {
		t.Fatalf("review scope omitted the exact documentation file reference: %+v", got.ReviewedFiles)
	}
	tampered := run
	tampered.Reviews = append([]ReviewRecord(nil), run.Reviews...)
	tampered.Reviews[0].CandidateDigest = "sha256:wrong-retained-candidate"
	if _, err := freshReviewEvidenceForVerify(store, dir, project, final, plan, projectconfig.Runtime{Review: &projectconfig.ReviewConfig{}}, tampered, host); err == nil {
		t.Fatal("review with a mismatched retained-candidate digest was accepted")
	}
}

type fullVerifyNativeWorkspaceInvoker struct {
	root            string
	called          bool
	workspace       projectworkspace.Handle
	roleTimeout     time.Duration
	requestDigest   string
	requestContext  json.RawMessage
	typedIncomplete bool
}

func (i *fullVerifyNativeWorkspaceInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.called = true
	i.roleTimeout = config.Timeout
	i.requestContext = append(json.RawMessage(nil), request.Context...)
	if options.Workspace == nil {
		return agentexec.RunResult{}, errors.New("native assessment received no owned workspace")
	}
	i.workspace = *options.Workspace
	if sameWorkspacePath(i.workspace.CWD, i.root) {
		return agentexec.RunResult{}, errors.New("native assessment received the adopting checkout as CWD")
	}
	journalBytes, err := os.ReadFile(filepath.Join(i.root, ".markitect", "runs", "private", "workspaces", i.workspace.ID+".json"))
	if err != nil {
		return agentexec.RunResult{}, err
	}
	var journal workspaceJournal
	if err := json.Unmarshal(journalBytes, &journal); err != nil {
		return agentexec.RunResult{}, err
	}
	if len(journal.Request.AllowedPaths) != 0 {
		return agentexec.RunResult{}, errors.New("read-only assessment workspace has writable paths")
	}
	var payload struct {
		RequiredSubjects []string `json:"requiredSubjects"`
	}
	if err := json.Unmarshal(request.Context, &payload); err != nil {
		return agentexec.RunResult{}, err
	}
	assessments := make([]FullAssessment, 0, len(payload.RequiredSubjects))
	for _, subject := range payload.RequiredSubjects {
		outcome := "pass"
		if i.typedIncomplete && len(assessments) == 0 {
			outcome = "incomplete"
		}
		assessments = append(assessments, FullAssessment{Subject: subject, Outcome: outcome, Detail: "fixed audit subject inspected"})
	}
	status := "pass"
	if i.typedIncomplete {
		status = "incomplete"
	}
	report, err := json.Marshal(fullAuditResponse{Status: status, Summary: "native read-only workspace inspected", Assessments: assessments, Findings: []string{}, Counterexamples: []FullCounterexample{}})
	if err != nil {
		return agentexec.RunResult{}, err
	}
	inputDigest, err := digest(request)
	if err != nil {
		return agentexec.RunResult{}, err
	}
	i.requestDigest = inputDigest
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Ptr(10), OutputTokens: int64Ptr(5)}
	response := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: "full-verify-native-run", Nonce: "full-verify-native-nonce",
		Role: request.Role, InputDigest: inputDigest, Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, ReportJSON: report, Uncertainty: []string{}, Usage: usage}
	receipt := terminalWorkspaceReceipt("full-verify-native-run")
	receipt.InputDigest = inputDigest
	receipt.ConfigDigest = "native-fixture-fingerprint"
	receipt.Outcome = agentexec.OutcomeProposed
	receipt.Usage = usage
	return agentexec.RunResult{Response: response, Receipt: receipt}, nil
}

func (*fullVerifyNativeWorkspaceInvoker) Fingerprint(agentexec.Config) (string, error) {
	return "native-fixture-fingerprint", nil
}

func TestFullVerifyNativeAssessmentUsesReadOnlyOwnedWorkspaceWithoutProvider(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	revision := identityHead(t, root)
	project := fullVerifyProjectFacts(t, root, revision)
	var err error
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	managerID := project.Report.Managers[0].ID
	for _, manager := range project.Report.Managers {
		if manager.Parent != "" {
			managerID = manager.ID
			break
		}
	}
	assessmentAgent := workspaceBridgeAgent(t, root)
	assessmentAgent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	assessmentAgent.Model = "gpt-6-luna"
	assessmentAgent.ProviderVersion = "codex-cli 0.162.0"
	assessmentAgent.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	assessmentAgent.Timeout = projectconfig.Duration(time.Hour)
	assessmentAgent.MaxStdoutBytes = 1 << 20
	assessmentAgent.MaxStderrBytes = 1 << 20
	assessmentAgent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
	runtime := projectconfig.Runtime{Agents: map[string]projectconfig.Agent{managerID: assessmentAgent}, Review: &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{managerID: assessmentAgent}},
		Limits: projectconfig.Limits{MaxDepth: 1, MaxStarts: 256, MaxParallel: 2, MaxDuration: projectconfig.Duration(4 * time.Hour), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}}
	invoker := &fullVerifyNativeWorkspaceInvoker{root: root}
	finalID, finalDigest := "final-candidate-id", "sha256:final-candidate"
	parentID := ""
	for _, manager := range project.Report.Managers {
		if manager.ID == managerID {
			parentID = manager.Parent
			break
		}
	}
	parentReview := fullReviewEvidence{TaskID: "parent-task", ManagerID: parentID, Phase: "integrate",
		ReviewCandidateID: "reviewed-partial-candidate", ReviewCandidateDigest: "sha256:partial",
		FinalCandidateID: finalID, FinalCandidateDigest: finalDigest, ScopeDigest: "sha256:scope", InputDigest: "sha256:review-input",
		ReceiptRunID: "review-receipt", Outcome: "pass", Findings: []ReviewFinding{},
		ReviewedFiles: []fullReviewFileReference{{Path: "docs/greeting.md", Mode: "0644", ContentDigest: "sha256:docs-content"}}}
	integrationReviews := fullAuditIntegrationReviewsForManager(project.Report, managerID, finalID, finalDigest, []fullReviewEvidence{parentReview})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	row, err := fullAuditManager(ctx, hostForProjectWithWorkspace(root, project, service), invoker, root, project, runtime,
		managerID, projectconfig.StrictnessProfile{}, BriefingContext{}, nil, nil, integrationReviews, false, "full-verify-test")
	if err != nil {
		t.Fatalf("native Manager audit failed: row=%+v err=%v", row, err)
	}
	if !invoker.called || invoker.roleTimeout != time.Hour || invoker.requestDigest != row.InputDigest {
		t.Fatalf("native audit did not preserve the role configuration and request binding: called=%t timeout=%s digest=%q row=%+v", invoker.called, invoker.roleTimeout, invoker.requestDigest, row)
	}
	if row.Status != "passed" || row.Receipt == nil || row.Receipt.RunID != "full-verify-native-run" || row.Receipt.InputDigest != row.InputDigest {
		t.Fatalf("native audit report did not preserve terminal receipt and validated status: %+v", row)
	}
	if _, err := os.Stat(invoker.workspace.CWD); !os.IsNotExist(err) {
		t.Fatalf("terminal read-only workspace was not closed: %v", err)
	}
	journal := readWorkspaceJournal(t, root, invoker.workspace.ID)
	if journal.State != "closed" || journal.Delta == nil || len(journal.Delta.Changes) != 0 || journal.Receipt.RunID != "full-verify-native-run" {
		t.Fatalf("read-only workspace journal did not retain empty harvested evidence and receipt: %+v", journal)
	}
	var captured struct {
		FreshIntegrationReviews []fullReviewEvidence `json:"freshIntegrationReviews"`
	}
	if err := json.Unmarshal(invoker.requestContext, &captured); err != nil {
		t.Fatalf("decode parent review projection: %v", err)
	}
	if len(captured.FreshIntegrationReviews) != 1 || !reflect.DeepEqual(captured.FreshIntegrationReviews[0], parentReview) {
		t.Fatalf("full audit did not receive the exact parent integration provenance: %+v", captured.FreshIntegrationReviews)
	}
}

func TestFullVerifyInvocationReceivesExplicitCrossManagerArtifactEvidence(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	revision := identityHead(t, root)
	project := fullVerifyProjectFacts(t, root, revision)
	var err error
	if len(project.Report.Artifacts) == 0 || len(project.Report.Managers) < 2 {
		t.Fatalf("fixture lacks cross-Manager model definitions: artifacts=%d managers=%d", len(project.Report.Artifacts), len(project.Report.Managers))
	}
	managerID := project.Report.Artifacts[0].Owner
	otherManagerID := ""
	for _, manager := range project.Report.Managers {
		if manager.ID != managerID {
			otherManagerID = manager.ID
			break
		}
	}
	if otherManagerID == "" {
		t.Fatal("fixture has no second Manager for cross-Manager evidence")
	}
	// The explicit base facts include one child for the native-workspace
	// predicate. This test builds its own child obligations below, so make the
	// other existing Manager a peer before adding that separate child.
	for index := range project.Report.Managers {
		if project.Report.Managers[index].ID == otherManagerID {
			project.Report.Managers[index].Parent = ""
		}
	}
	publicRelation := projectmodel.Statement{ID: "statement:public-relation", Owner: otherManagerID, Public: true, Description: "Public related contract."}
	privateRelation := projectmodel.Statement{ID: "statement:private-relation", Owner: otherManagerID, Public: false, Description: "Private related detail."}
	publicStatement := projectmodel.Statement{ID: "statement:public-contract", Owner: otherManagerID, Public: true, Description: "Public contract realized by the artifact.", Uses: []string{publicRelation.ID, privateRelation.ID}}
	privateStatement := projectmodel.Statement{ID: "statement:private-detail", Owner: otherManagerID, Public: false, Description: "Private implementation detail."}
	unrelatedStatement := projectmodel.Statement{ID: "statement:unrelated", Owner: otherManagerID, Public: true, Description: "Unrelated public contract."}
	// Only the attached foreign Check uses this Statement; the audit must not name it.
	privateCheckInput := projectmodel.Statement{ID: "statement:private-check-input", Owner: otherManagerID, Public: false, Description: "Private input of the other Manager's check."}
	project.Report.Statements = append(project.Report.Statements, publicRelation, privateRelation, publicStatement, privateStatement, unrelatedStatement, privateCheckInput)
	attachedCheck := projectmodel.Check{ID: "check:artifact-check", Owner: otherManagerID, Command: []string{"python", "-m", "unittest"}, Limitation: "Checks the same snapshot.", Uses: []string{publicStatement.ID, privateCheckInput.ID}}
	unrelatedCheck := projectmodel.Check{ID: "check:unrelated", Owner: otherManagerID, Command: []string{"go", "test", "./..."}}
	project.Report.Checks = append(project.Report.Checks, attachedCheck, unrelatedCheck)
	// A child Manager owes the parent integration. Its private Statement must
	// stay hidden; its contract, which the parent also uses, and its Check,
	// which the parent's artifact declares, must each be defined once.
	child := projectmodel.Manager{ID: "manager:child", Name: "child", Parent: managerID}
	project.Report.Managers = append(project.Report.Managers, child)
	childPrivate := projectmodel.Statement{ID: "statement:child-private", Owner: child.ID, Public: false, Description: "Private detail of the child."}
	childContract := projectmodel.Statement{ID: "statement:child-contract", Owner: child.ID, Public: true, Description: "Public contract of the child.", Uses: []string{childPrivate.ID}}
	parentFlow := projectmodel.Statement{ID: "statement:parent-flow", Owner: managerID, Public: false, Description: "Parent behaviour built on the child contract.", Uses: []string{childContract.ID}}
	// The parent does not use this contract, so only the obligations define it.
	childSpare := projectmodel.Statement{ID: "statement:child-spare", Owner: child.ID, Public: true, Description: "Another public contract of the child.", Uses: []string{childPrivate.ID}}
	childPrivateTwo := projectmodel.Statement{ID: "statement:child-private-two", Owner: child.ID, Public: false, Description: "Second private detail of the child."}
	project.Report.Statements = append(project.Report.Statements, childPrivate, childContract, parentFlow, childSpare, childPrivateTwo)
	childArtifact := projectmodel.Artifact{ID: "artifact:child-implementation", Owner: child.ID, Required: true, Realizes: []string{childContract.ID, childPrivate.ID}}
	project.Report.Artifacts = append(project.Report.Artifacts, childArtifact)
	declaredChildCheck := projectmodel.Check{ID: "check:child-declared", Owner: child.ID, Command: []string{"go", "test", "./child/..."}, Limitation: "Child suite.", Uses: []string{childContract.ID, childPrivate.ID}}
	childOnlyCheck := projectmodel.Check{ID: "check:child-only", Owner: child.ID, Command: []string{"go", "test", "./child/internal/..."}, Limitation: "Child internals.", Uses: []string{childPrivate.ID}}
	project.Report.Checks = append(project.Report.Checks, declaredChildCheck, childOnlyCheck)
	for index := range project.Report.Artifacts {
		if project.Report.Artifacts[index].Owner == managerID {
			project.Report.Artifacts[index].Realizes = append(project.Report.Artifacts[index].Realizes, publicStatement.ID, privateStatement.ID)
			project.Report.Artifacts[index].Checks = append(project.Report.Artifacts[index].Checks, attachedCheck.ID, declaredChildCheck.ID)
			break
		}
	}
	owned, err := projectmodel.Context(project.Report, managerID)
	if err != nil {
		t.Fatal(err)
	}
	wantSubjects := fullAuditSubjects(owned, fullIntegrationObligations(project.Report, managerID))
	for _, subject := range []string{"integration:statement:" + childContract.ID, "integration:check:" + declaredChildCheck.ID, "integration:check:" + childOnlyCheck.ID} {
		if !containsString(wantSubjects, subject) {
			t.Fatalf("the child's obligations lack %s: %v", subject, wantSubjects)
		}
	}

	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	assessmentAgent := workspaceBridgeAgent(t, root)
	assessmentAgent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	assessmentAgent.Model = "gpt-6-luna"
	assessmentAgent.ProviderVersion = "codex-cli 0.162.0"
	assessmentAgent.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	assessmentAgent.Timeout = projectconfig.Duration(time.Hour)
	assessmentAgent.MaxStdoutBytes = 1 << 20
	assessmentAgent.MaxStderrBytes = 1 << 20
	assessmentAgent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
	runtime := projectconfig.Runtime{Agents: map[string]projectconfig.Agent{managerID: assessmentAgent}, Review: &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{managerID: assessmentAgent}},
		Limits: projectconfig.Limits{MaxDepth: 1, MaxStarts: 8, MaxParallel: 1, MaxDuration: projectconfig.Duration(time.Hour), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}}
	invoker := &fullVerifyNativeWorkspaceInvoker{root: root}
	failedResult := CheckResult{ID: attachedCheck.ID, Outcome: "failed", ExitCode: 1, Stderr: "assertion failed"}
	results := relevantManagerChecks(project.Report, managerID, []CheckResult{failedResult, {ID: unrelatedCheck.ID, Outcome: "passed"}})
	// The child was audited first; its verdicts include its private Statements.
	childRow := FullManagerAssessment{ManagerID: child.ID, Status: "failed", Summary: "Child audit.", ScopeDigest: "sha256:child", Assessments: []FullAssessment{
		{Subject: "statement:" + childPrivate.ID, Outcome: "fail", Detail: "private detail broken"},
		{Subject: "statement:" + childPrivateTwo.ID, Outcome: "pass", Detail: "private detail holds"},
		{Subject: "statement:" + childContract.ID, Outcome: "pass", Detail: "contract holds"},
		{Subject: "artifact:" + childArtifact.ID, Outcome: "incomplete", Detail: "not established"},
	}}
	childAssessments := fullChildAssessments([]FullManagerAssessment{childRow}, project.Report, managerID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := fullAuditManager(ctx, hostForProjectWithWorkspace(root, project, service), invoker, root, project, runtime,
		managerID, projectconfig.StrictnessProfile{}, BriefingContext{}, childAssessments, results, nil, false, "full-verify-support-test"); err != nil {
		t.Fatalf("Manager audit failed: %v", err)
	}
	var auditContext struct {
		Manager                projectmodel.ManagerContext `json:"manager"`
		IntegrationObligations []fullObligationView        `json:"integrationObligations"`
		ChildAssessments       []fullChildAssessment       `json:"childAssessments"`
		CheckResults           []CheckResult               `json:"checkResults"`
		RequiredSubjects       []string                    `json:"requiredSubjects"`
	}
	if err := json.Unmarshal(invoker.requestContext, &auditContext); err != nil {
		t.Fatalf("decode captured audit context: %v", err)
	}
	for _, hidden := range []string{privateCheckInput.ID, privateRelation.ID, unrelatedStatement.ID, childPrivate.ID, childPrivateTwo.ID} {
		if strings.Contains(string(invoker.requestContext), hidden) {
			t.Fatalf("audit context names %s, which the Manager may not see or does not reference", hidden)
		}
	}
	var auditFields map[string]json.RawMessage
	if err := json.Unmarshal(invoker.requestContext, &auditFields); err != nil {
		t.Fatal(err)
	}
	for _, duplicate := range []string{"supportingStatements", "supportingChecks"} {
		if _, ok := auditFields[duplicate]; ok {
			t.Fatalf("%s duplicates the Manager context; the audit must have one source", duplicate)
		}
	}
	// Each Statement, Artifact and Check definition reaches the audit at most
	// once, wherever it sits in the request; a realized foreign contract comes
	// from manager.contracts with only public relations.
	definitionIDs := map[string]bool{}
	for _, statement := range project.Report.Statements {
		definitionIDs[statement.ID] = true
	}
	for _, artifact := range project.Report.Artifacts {
		definitionIDs[artifact.ID] = true
	}
	for _, check := range project.Report.Checks {
		definitionIDs[check.ID] = true
	}
	definitions := map[string]int{}
	var countDefinitions func(any)
	countDefinitions = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if id, ok := x["id"].(string); ok && definitionIDs[id] {
				if _, hasOwner := x["owner"]; hasOwner {
					definitions[id]++
				}
			}
			for _, value := range x {
				countDefinitions(value)
			}
		case []any:
			for _, item := range x {
				countDefinitions(item)
			}
		}
	}
	var whole any
	if err := json.Unmarshal(invoker.requestContext, &whole); err != nil {
		t.Fatal(err)
	}
	countDefinitions(whole)
	for id, count := range definitions {
		if count != 1 {
			t.Fatalf("%s is defined %d times in the audit context", id, count)
		}
	}
	for _, id := range []string{childContract.ID, childSpare.ID, childArtifact.ID, declaredChildCheck.ID, childOnlyCheck.ID} {
		if definitions[id] != 1 {
			t.Fatalf("the child's %s is defined %d times in the audit context, want once", id, definitions[id])
		}
	}
	// The obligations name every public contract and Check the child owes;
	// the definitions outside the Manager context keep only public relations.
	if len(auditContext.IntegrationObligations) != 1 || auditContext.IntegrationObligations[0].ChildManager != child.ID {
		t.Fatalf("integration obligations = %+v", auditContext.IntegrationObligations)
	}
	obligation := auditContext.IntegrationObligations[0]
	if !reflect.DeepEqual(obligation.Contracts, []string{childContract.ID, childSpare.ID}) {
		t.Fatalf("child contracts in the obligations = %v, want every public contract by ID", obligation.Contracts)
	}
	if len(obligation.ContractDefinitions) != 1 || obligation.ContractDefinitions[0].ID != childSpare.ID || len(obligation.ContractDefinitions[0].Uses) != 0 {
		t.Fatalf("child contract definitions in the obligations = %+v", obligation.ContractDefinitions)
	}
	if !reflect.DeepEqual(obligation.Checks, []string{declaredChildCheck.ID, childOnlyCheck.ID}) {
		t.Fatalf("child checks in the obligations = %v, want every Check by ID", obligation.Checks)
	}
	if len(obligation.CheckDefinitions) != 1 || obligation.CheckDefinitions[0].ID != childOnlyCheck.ID || len(obligation.CheckDefinitions[0].Uses) != 0 {
		t.Fatalf("child check definitions in the obligations = %+v", obligation.CheckDefinitions)
	}
	if len(obligation.Artifacts) != 1 || !reflect.DeepEqual(obligation.Artifacts[0].Realizes, []string{childContract.ID}) {
		t.Fatalf("child artifact in the obligations = %+v", obligation.Artifacts)
	}
	// The child's verdicts on its private Statements arrive counted, not named.
	if len(auditContext.ChildAssessments) != 1 {
		t.Fatalf("child assessments = %+v", auditContext.ChildAssessments)
	}
	childView := auditContext.ChildAssessments[0]
	wantChildSubjects := []fullChildSubject{{Subject: "artifact:" + childArtifact.ID, Outcome: "incomplete"}, {Subject: "statement:" + childContract.ID, Outcome: "pass"}}
	if childView.Status != "failed" || !reflect.DeepEqual(childView.Subjects, wantChildSubjects) {
		t.Fatalf("child assessment subjects = %+v, want %+v", childView, wantChildSubjects)
	}
	if want := (fullPrivateVerdicts{Note: fullPrivateVerdictsNote, Passed: 1, Failed: 1}); childView.PrivateStatements != want {
		t.Fatalf("child private verdicts = %+v, want %+v", childView.PrivateStatements, want)
	}
	var realized *projectmodel.Statement
	for i := range auditContext.Manager.Contracts {
		if auditContext.Manager.Contracts[i].ID == publicStatement.ID {
			realized = &auditContext.Manager.Contracts[i]
		}
	}
	if realized == nil || !reflect.DeepEqual(realized.Uses, []string{publicRelation.ID}) {
		t.Fatalf("realized foreign contract in manager.contracts = %+v", realized)
	}
	// Every Check an owned artifact declares reaches the audit exactly once:
	// owned Checks under checks, other Managers' Checks under foreignChecks.
	checkOwners := map[string]string{}
	for _, check := range project.Report.Checks {
		checkOwners[check.ID] = check.Owner
	}
	wantOwned, wantForeign := map[string]bool{}, map[string]bool{}
	for _, artifact := range project.Report.Artifacts {
		if artifact.Owner == managerID {
			for _, checkID := range artifact.Checks {
				if owner, ok := checkOwners[checkID]; ok && owner == managerID {
					wantOwned[checkID] = true
				} else if ok {
					wantForeign[checkID] = true
				}
			}
		}
	}
	seen := map[string]int{}
	for _, check := range append(append([]projectmodel.Check(nil), auditContext.Manager.Checks...), auditContext.Manager.ForeignChecks...) {
		seen[check.ID]++
	}
	for id := range wantOwned {
		if seen[id] != 1 {
			t.Fatalf("owned declared check %s appears %d times in the audit context", id, seen[id])
		}
	}
	for id := range wantForeign {
		if seen[id] != 1 {
			t.Fatalf("foreign declared check %s appears %d times in the audit context", id, seen[id])
		}
	}
	var attached *projectmodel.Check
	for i := range auditContext.Manager.ForeignChecks {
		if auditContext.Manager.ForeignChecks[i].ID == attachedCheck.ID {
			attached = &auditContext.Manager.ForeignChecks[i]
		}
	}
	if attached == nil || attached.Owner != otherManagerID || attached.Limitation != attachedCheck.Limitation || !reflect.DeepEqual(attached.Uses, []string{publicStatement.ID}) {
		t.Fatalf("attached foreign check in the audit context = %+v", attached)
	}
	if !reflect.DeepEqual(auditContext.CheckResults, []CheckResult{failedResult}) {
		t.Fatalf("invocation changed the attached check result: %+v", auditContext.CheckResults)
	}
	if !reflect.DeepEqual(auditContext.RequiredSubjects, wantSubjects) {
		t.Fatalf("supporting evidence changed the Manager's owned obligations: got=%v want=%v", auditContext.RequiredSubjects, wantSubjects)
	}
}

func TestFullVerifyTypedIncompleteAuditCannotPass(t *testing.T) {
	root := makeProjectRunFixture(t)
	configureWorkspaceBridgeInstructions(t, root)
	revision := identityHead(t, root)
	project := fullVerifyProjectFacts(t, root, revision)
	var err error
	service, err := projectworkspace.NewGitService(t.TempDir(), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	managerID := project.Report.Managers[0].ID
	agent := workspaceBridgeAgent(t, root)
	agent.Command = filepath.Join(t.TempDir(), "codex-app-server")
	agent.Model = "gpt-6-luna"
	agent.ProviderVersion = "codex-cli 0.162.0"
	agent.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "high", MaxEventBytes: 1 << 20}
	agent.Timeout = projectconfig.Duration(time.Hour)
	agent.MaxStdoutBytes = 1 << 20
	agent.MaxStderrBytes = 1 << 20
	agent.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1}
	runtime := projectconfig.Runtime{Agents: map[string]projectconfig.Agent{managerID: agent}, Review: &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{managerID: agent}},
		Limits: projectconfig.Limits{MaxDepth: 1, MaxStarts: 256, MaxParallel: 2, MaxDuration: projectconfig.Duration(4 * time.Hour), MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}}
	invoker := &fullVerifyNativeWorkspaceInvoker{root: root, typedIncomplete: true}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	row, err := fullAuditManager(ctx, hostForProjectWithWorkspace(root, project, service), invoker, root, project, runtime,
		managerID, projectconfig.StrictnessProfile{}, BriefingContext{}, nil, nil, nil, false, "full-verify-incomplete-test")
	if err == nil || row.Status != "incomplete" {
		t.Fatalf("typed incomplete audit was accepted: row=%+v err=%v", row, err)
	}
	if len(row.Assessments) == 0 || row.Assessments[0].Outcome != "incomplete" || row.Receipt == nil {
		t.Fatalf("typed incomplete evidence or terminal receipt was not preserved: %+v", row)
	}
}

// fullVerifyProjectFacts supplies an explicit Runtime input for audit-gate
// tests. Source acquisition and model compilation are exercised separately by
// Application integration tests.
func fullVerifyProjectFacts(t *testing.T, root, revision string) *projectvalue.Project {
	t.Helper()
	const rootManager = "manager:root"
	const childManager = "manager:peer"
	runtimeBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projectconfig.RuntimePath)))
	if err != nil {
		t.Fatalf("read exact Runtime bytes from the workspace fixture: %v", err)
	}
	instructionPath := "docs/agent-instructions.md"
	instructionBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(instructionPath)))
	if err != nil {
		t.Fatalf("read exact instruction document from the workspace fixture: %v", err)
	}
	files := map[string][]byte{
		"docs/greeting.md":        []byte("greeting documentation\n"),
		instructionPath:           instructionBytes,
		"src/greeting.py":         []byte("def greeting(): return 'hello'\n"),
		projectconfig.RuntimePath: runtimeBytes,
	}
	modes := map[string]string{"docs/greeting.md": snapshot.RegularMode, instructionPath: snapshot.RegularMode, "src/greeting.py": snapshot.RegularMode, projectconfig.RuntimePath: snapshot.RegularMode}
	return &projectvalue.Project{
		Root: root, Revision: revision, Digest: strings.Repeat("f", 64),
		Config:   projectvalue.Config{CoverageMode: "full", DocumentPath: instructionPath},
		Snapshot: &snapshot.Snapshot{ID: revision, Files: files, Modes: modes},
		Report: projectmodel.Report{Digest: "sha256:" + strings.Repeat("a", 64), ModelDigest: "sha256:" + strings.Repeat("b", 64),
			Managers: []projectmodel.Manager{{ID: rootManager, Name: "root"}, {ID: childManager, Name: "child", Parent: rootManager}},
			Statements: []projectmodel.Statement{
				{ID: "statement:greeting", Owner: rootManager, Public: true, Description: "Provide a greeting."},
				{ID: "statement:child", Owner: childManager, Public: true, Description: "Support the greeting."},
			},
			Artifacts: []projectmodel.Artifact{{ID: "artifact:greeting", Owner: rootManager, Required: true,
				Realizes: []string{"statement:greeting"}, Paths: []string{"docs/greeting.md", "src/greeting.py"}}},
			Checks: []projectmodel.Check{{ID: "check:greeting", Owner: rootManager, Command: []string{"python", "-m", "unittest"}}},
			Files: []projectmodel.FileEntry{
				{Path: "docs/greeting.md", Mode: snapshot.RegularMode, Owner: rootManager, Class: "documentation", Artifacts: []string{"artifact:greeting"}, Exists: true},
				{Path: "src/greeting.py", Mode: snapshot.RegularMode, Owner: rootManager, Class: "implementation", Artifacts: []string{"artifact:greeting"}, Exists: true},
			},
		},
	}
}

func TestFullVerifyKnownCostOverflowSaturatesExplicitly(t *testing.T) {
	total := int64(math.MaxInt64 - 2)
	if !addFullKnownCost(&total, 3) || total != math.MaxInt64 {
		t.Fatalf("known cost overflow = (%d, false), want saturated MaxInt64 and overflow=true", total)
	}
}

func TestFullVerifyProjectRejectsStaleExpectedSnapshotBeforeAudit(t *testing.T) {
	project := &projectvalue.Project{
		Revision: strings.Repeat("a", 40),
		Snapshot: &snapshot.Snapshot{Files: map[string][]byte{"README.md": []byte("fixed bytes\n")}, Modes: map[string]string{"README.md": snapshot.RegularMode}},
	}
	_, err := FullVerifyProject(context.Background(), Host{}, projectexecutor.ProcessInvoker{}, t.TempDir(), project,
		projectconfig.Runtime{}, FullVerifyBinding{ExpectedSnapshot: strings.Repeat("f", 64)})
	if err != ErrStale {
		t.Fatalf("stale expected snapshot error = %v, want ErrStale", err)
	}
}

func TestFullVerifyStrictnessRequiresTypedEvidenceAndGroundedCounterexamples(t *testing.T) {
	required := []string{"statement:orders", "evidence:negative stock case"}
	base := fullAuditResponse{Status: "pass", Summary: "review complete", Findings: []string{},
		Assessments:     []FullAssessment{{Subject: "statement:orders", Outcome: "pass", Detail: "checked"}, {Subject: "evidence:negative stock case", Outcome: "pass", Detail: "checked"}},
		Counterexamples: []FullCounterexample{{Expected: "reject negative stock", Observed: "negative stock can be submitted", EvidenceRefs: []string{"statement:orders"}}}}
	if err := validateFullAssessments(base, required, 1, required, []string{"src/orders.go"}); err != nil {
		t.Fatalf("valid strict response rejected: %v", err)
	}
	missingEvidence := base
	missingEvidence.Assessments = base.Assessments[:1]
	if err := validateFullAssessments(missingEvidence, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("missing typed evidence subject was accepted")
	}
	inventedSubject := base
	inventedSubject.Assessments = append(append([]FullAssessment(nil), base.Assessments...), FullAssessment{Subject: "Documentation semantics", Outcome: "pass", Detail: "invented scope"})
	if err := validateFullAssessments(inventedSubject, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("invented audit subject was accepted")
	}
	missingCounterexample := base
	missingCounterexample.Counterexamples = nil
	if err := validateFullAssessments(missingCounterexample, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("missing required counterexample was accepted")
	}
	unbound := base
	unbound.Counterexamples = []FullCounterexample{{Expected: "reject negative stock", Observed: "it passed", EvidenceRefs: []string{"unrelated:file"}}}
	if err := validateFullAssessments(unbound, required, 1, required, []string{"src/orders.go"}); err == nil {
		t.Fatal("counterexample with out-of-scope evidence was accepted")
	}
}

func TestFullVerifyResponseSchemaUsesExactDynamicSubjectEnum(t *testing.T) {
	want := []string{"statement:orders", "evidence:negative stock case"}
	var schema struct {
		Properties struct {
			Assessments struct {
				MinItems int `json:"minItems"`
				MaxItems int `json:"maxItems"`
				Items    struct {
					Properties struct {
						Subject struct {
							Enum        []string `json:"enum"`
							Description string   `json:"description"`
						} `json:"subject"`
						Outcome struct {
							Description string `json:"description"`
						} `json:"outcome"`
						Detail struct {
							Description string `json:"description"`
						} `json:"detail"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"assessments"`
			Status struct {
				Description string `json:"description"`
			} `json:"status"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(fullVerifyResponseSchema(want, 0), &schema); err != nil {
		t.Fatalf("response schema is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(schema.Properties.Assessments.Items.Properties.Subject.Enum, want) {
		t.Fatalf("subject enum=%q; want exact request subjects %q", schema.Properties.Assessments.Items.Properties.Subject.Enum, want)
	}
	if schema.Properties.Assessments.MinItems != len(want) || schema.Properties.Assessments.MaxItems != len(want) {
		t.Fatalf("assessment item bounds=(%d,%d); want exact required-subject count %d", schema.Properties.Assessments.MinItems, schema.Properties.Assessments.MaxItems, len(want))
	}
	for name, description := range map[string]string{
		"status":  schema.Properties.Status.Description,
		"subject": schema.Properties.Assessments.Items.Properties.Subject.Description,
		"outcome": schema.Properties.Assessments.Items.Properties.Outcome.Description,
		"detail":  schema.Properties.Assessments.Items.Properties.Detail.Description,
	} {
		if strings.TrimSpace(description) == "" {
			t.Errorf("schema %s is missing its contract description", name)
		}
	}
}

func setupFullVerifyProcesses(t *testing.T) {
	t.Helper()
	setRuntimeFixtureEnv(t, fullVerifyHelperEnv, "1")
	setRuntimeFixtureEnv(t, fullVerifyLogEnv, filepath.Join(testkit.TempDir(t), "full-verify.jsonl"))
	setRuntimeFixtureEnv(t, fullVerifyCheckEnv, "1")
}

func configureFullVerifyRuntime(t *testing.T, root string) {
	t.Helper()
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"-test.run=^TestFullVerifyHelperProcess$"}
		reviewers := make(map[string]projectconfig.Agent, len(runtime.Agents))
		for id, agent := range runtime.Agents {
			agent.Command, agent.Args = executable, args
			agent.Environment = append(agent.Environment, fullVerifyHelperEnv, fullVerifyBadEnv, fullVerifyLogEnv, fullVerifyNoUsageEnv)
			reviewers[id] = agent
		}
		runtime.Review = &projectconfig.ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
		for id, agent := range runtime.Agents {
			agent.Args = []string{"-test.run=^TestFullVerifyCheckProcess$"}
			agent.Environment = append(agent.Environment, fullVerifyCheckEnv)
			runtime.Agents[id] = agent
		}
	})
	// Ensure the helper executable remains available to the pinned check resolver.
}

func int64Ptr(value int64) *int64 { return &value }
