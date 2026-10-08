package projectadoption

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

const distillationHelperEnv = "MARKITECT_PROJECTADOPTION_TEST_HELPER"
const distillationModeEnv = "MARKITECT_PROJECTADOPTION_TEST_MODE"

func TestDistillationExecutorHelper(t *testing.T) {
	if os.Getenv(distillationHelperEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		os.Exit(31)
	}
	if invocation.Request.Role != agentexec.RoleExecutor || len(invocation.Request.Artifacts) != 1 {
		os.Exit(32)
	}
	var requestContext map[string]json.RawMessage
	if err := json.Unmarshal(invocation.Request.Context, &requestContext); err != nil || len(requestContext["responseSchema"]) == 0 {
		os.Exit(33)
	}
	var targetContext DistillationTargetContext
	if err := json.Unmarshal(requestContext["targetContext"], &targetContext); err != nil || targetContext.RootManagerID == "" || targetContext.Digest == "" {
		os.Exit(35)
	}
	var prompt struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &prompt); err != nil ||
		!strings.Contains(prompt.Instructions, "commerce.sales.orders") ||
		!strings.Contains(prompt.Instructions, `Root scope proposals MUST have parentId = ""`) ||
		!strings.Contains(prompt.Instructions, "every non-root scope's parentId must name another declared scope ID, and parent relationships must be acyclic") ||
		!strings.Contains(prompt.Instructions, "Every proposed scope must contain at least one grounded claim assigned to it and at least one model-proposal file") ||
		!strings.Contains(prompt.Instructions, "Existing target Managers are guidance only and do not need mirrored as adoption scopes") ||
		!strings.Contains(prompt.Instructions, "observation/static-source") ||
		!strings.Contains(prompt.Instructions, "one-based line bounds") {
		os.Exit(36)
	}
	artifact := invocation.Request.Artifacts[0]
	if artifact.Path != "evidence/implementation.txt" || string(artifact.Content) != "package orders\nfunc Cancel() {}\n" {
		os.Exit(34)
	}
	claim := DistillationDraftClaim{ID: "implementation-observation", ScopeID: "orders", Kind: "observation", Method: "static-source",
		Statement:   "The selected source declares a cancellation function.",
		Evidence:    []EvidenceRef{{EvidenceID: "implementation", StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}},
		Uncertainty: []string{"This static source observation does not establish runtime behavior."}, RuntimeObservationJSON: ""}
	draft := DistillationDraft{
		Claims: []DistillationDraftClaim{claim}, Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []DistillationDraftQuestion{},
		Scopes:   []DistillationDraftScope{{ID: "orders", Name: "Order management", ParentID: "", ClaimIDs: []string{claim.ID}, OwnerCandidate: ""}},
		Proposal: ModelProposal{Goal: "Represent the proposed order scope", Files: []ProposedFile{{ScopeID: "orders", Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\n"}}},
	}
	reportBytes, _ := json.Marshal(draft)
	if os.Getenv(distillationModeEnv) == "bad-grounding" {
		draft.Claims[0].Evidence[0].Excerpt = "func Ship() {}"
		reportBytes, _ = json.Marshal(draft)
	}
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: int64Pointer(100), OutputTokens: int64Pointer(100), ToolCalls: int64Pointer(0)}
	if os.Getenv(distillationModeEnv) == "missing-usage" {
		usage = nil
	}
	if os.Getenv(distillationModeEnv) == "over-budget" {
		usage.InputTokens = int64Pointer(100_000_000)
		usage.OutputTokens = int64Pointer(100_000_000)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleExecutor, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{},
		EvidenceRefs: []string{artifact.Path}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: reportBytes, Uncertainty: []string{"Proposal requires owner review."}, Usage: usage,
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestGenerateDistillationUsesBoundedExecutorAndValidatesReport(t *testing.T) {
	root, discovery, target, _ := distillationDiscovery(t)
	selectedBefore, err := os.ReadFile(filepath.Join(root, "src", "orders", "cancel.go"))
	if err != nil {
		t.Fatal(err)
	}
	report, receipt, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), testDistillationOptions(t, target))
	if err != nil {
		t.Fatal(err)
	}
	if report.Method != "agent-assisted" || report.DiscoveryDigest != discovery.Digest || report.SchemaDigest != receipt.SchemaDigest || report.TargetBasis != target.ProjectDigest || report.TargetRevision != target.Revision || report.TargetContextDigest != target.Digest {
		t.Fatalf("generated report did not bind its source, method and schema: %+v", report)
	}
	if report.RunnerIdentity != receipt.RunnerIdentity || report.RunnerDigest != receipt.RunnerDigest || report.RunnerDigest != digestWithoutPrefix(receipt.Execution.ConfigDigest) || receipt.RunnerDigest == "" {
		t.Fatalf("report runner binding is not derived from the executed receipt: report=%+v receipt=%+v", report, receipt)
	}
	if receipt.Execution.Outcome != agentexec.OutcomeProposed || receipt.ExecutionReceiptDigest != digestValue(receipt.Execution) || receipt.EstimatedCostMicros != 2 || receipt.TargetContextDigest != target.Digest || receipt.DistillationDigest != report.Digest {
		t.Fatalf("execution receipt or conservative cost estimate is incorrect: %+v", receipt)
	}
	if err := ValidateDistillation(discovery, report); err != nil {
		t.Fatalf("generated report should be independently valid: %v", err)
	}
	selectedAfter, err := os.ReadFile(filepath.Join(root, "src", "orders", "cancel.go"))
	if err != nil || string(selectedAfter) != string(selectedBefore) {
		t.Fatalf("distillation changed selected source bytes: read error=%v", err)
	}
}

func TestGenerateDistillationRejectsMissingUsageAndBadGrounding(t *testing.T) {
	root, discovery, target, _ := distillationDiscovery(t)
	for _, mode := range []string{"missing-usage", "bad-grounding", "over-budget"} {
		config := testDistillationConfig(t)
		t.Setenv(distillationModeEnv, mode)
		_, _, err := GenerateDistillation(context.Background(), root, discovery, config, testDistillationOptions(t, target))
		if err == nil {
			t.Errorf("mode %q should produce an incomplete or invalid distillation", mode)
		}
	}
}

func TestGenerateDistillationRejectsUnsafeBoundsAndStaleDiscovery(t *testing.T) {
	root, discovery, target, _ := distillationDiscovery(t)
	config := testDistillationConfig(t)
	config.EnvironmentAllowlist = nil
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, config, testDistillationOptions(t, target)); err == nil || !strings.Contains(err.Error(), "explicit environment allowlist") {
		t.Fatalf("nil environment allowlist error = %v", err)
	}
	options := testDistillationOptions(t, target)
	options.MaxCostMicros = 0
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "cost ceiling") {
		t.Fatalf("unbounded cost options error = %v", err)
	}
	options = testDistillationOptions(t, target)
	options.TempParent = root
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "outside the source repository") {
		t.Fatalf("repository-local temporary parent error = %v", err)
	}
	options = testDistillationOptions(t, target)
	options.MaxStdoutBytes = maxDistillationStdout + 1
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "stdout limit") {
		t.Fatalf("unbounded stdout option error = %v", err)
	}
	alias := filepath.Join(t.TempDir(), "source-alias")
	if err := os.Symlink(root, alias); err == nil {
		options = testDistillationOptions(t, target)
		options.PrivateLogDirectory = alias
		if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "outside the source repository") {
			t.Fatalf("repository alias private-log directory error = %v", err)
		}
	}
	changed := discovery
	changed.Evidence = append([]Evidence(nil), discovery.Evidence...)
	changed.Evidence[0].Content += "changed"
	SealDiscovery(&changed)
	if _, _, err := GenerateDistillation(context.Background(), root, changed, testDistillationConfig(t), testDistillationOptions(t, target)); err == nil {
		t.Fatal("discovery with evidence that differs from the fixed commit should be rejected")
	}
}

func TestDistillationDraftSchemaMatchesBothPythonAdapterContracts(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("Python is unavailable for adapter contract validation")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate the test source")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	schema := distillationDraftJSONSchema()
	var responseSchema any
	if err := json.Unmarshal(schema, &responseSchema); err != nil {
		t.Fatal(err)
	}
	rootSchema := responseSchema.(map[string]any)
	claims := rootSchema["properties"].(map[string]any)["claims"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	claimID := claims["id"].(map[string]any)
	if claimID["minLength"] != float64(1) || claimID["maxLength"] != float64(64) {
		t.Fatalf("provider claim ID bounds = %#v", claimID)
	}
	if got := claims["kind"].(map[string]any)["enum"].([]any); len(got) != 4 {
		t.Fatalf("provider claim kind enum = %#v", got)
	}
	if got := claims["method"].(map[string]any)["enum"].([]any); len(got) != 4 {
		t.Fatalf("provider claim method enum = %#v", got)
	}
	scopes := rootSchema["properties"].(map[string]any)["scopes"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	parentID := scopes["parentId"].(map[string]any)
	if parentID["minLength"] != float64(0) || parentID["maxLength"] != float64(64) {
		t.Fatalf("provider scope parent ID bounds = %#v", parentID)
	}
	draft := DistillationDraft{
		Claims: []DistillationDraftClaim{{
			ID: "cancel-observation", ScopeID: "orders", Kind: "observation", Method: "static-source",
			Statement: "A cancellation function is declared.", Evidence: []EvidenceRef{{EvidenceID: "implementation", StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}},
			Uncertainty: []string{}, RuntimeObservationJSON: "",
		}},
		Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []DistillationDraftQuestion{},
		Scopes:   []DistillationDraftScope{{ID: "orders", Name: "Orders", ParentID: "", ClaimIDs: []string{"cancel-observation"}, OwnerCandidate: ""}},
		Proposal: ModelProposal{Goal: "Represent orders", Files: []ProposedFile{{ScopeID: "orders", Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\n"}}},
	}
	var draftValue any
	if data, err := json.Marshal(draft); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(data, &draftValue); err != nil {
		t.Fatal(err)
	}
	invocation := map[string]any{
		"apiVersion": agentexec.APIVersion, "runId": "run-fixture", "nonce": "nonce-fixture",
		"inputDigest": "sha256:" + strings.Repeat("a", 64),
		"request": map[string]any{
			"role": agentexec.RoleExecutor, "sourceRevision": strings.Repeat("b", 40),
			"modelDigest": "sha256:" + strings.Repeat("c", 64), "modulePin": "project-adoption/v1alpha1",
			"projectionId": "brownfield-distillation", "scopeIds": []string{}, "policyIds": []string{},
			"context": map[string]any{"responseSchema": responseSchema, "testDraft": draftValue}, "artifacts": []any{},
		},
	}
	encoded, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	script := `import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("adapter", sys.argv[1])
adapter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(adapter)
value = adapter.strict_loads(sys.stdin.buffer.read())
validated = adapter.validate_invocation(value)
schema = adapter.task_response_schema(validated)
assert schema == validated["request"]["context"]["responseSchema"]
adapter.validate_report_value(validated["request"]["context"]["testDraft"], schema)
assert validated["request"]["context"]["testDraft"]["scopes"][0]["parentId"] == ""
`
	for _, adapterPath := range []string{
		filepath.Join(repoRoot, "internal", "tooling", "codexrunner", "runner.py"),
		filepath.Join(repoRoot, "internal", "tooling", "clauderunner", "runner.py"),
	} {
		adapterPath := adapterPath
		t.Run(filepath.Base(filepath.Dir(adapterPath)), func(t *testing.T) {
			cmd := exec.Command(python, "-B", "-c", script, adapterPath)
			cmd.Dir = repoRoot
			cmd.Stdin = bytes.NewReader(encoded)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("adapter rejected generated response schema: %v\n%s", err, output)
			}
		})
	}
}

func TestAgentDistillationRequiresFixedTargetBinding(t *testing.T) {
	_, discovery, target, targetProject := distillationDiscovery(t)
	report := Distillation{
		APIVersion: DistillationVersion, DiscoveryDigest: discovery.Digest, SchemaDigest: strings.Repeat("d", 64),
		Claims: []Claim{{ID: "claim", ScopeID: "orders", Kind: "observation", Method: "static-source", Statement: "A cancellation function is declared.",
			Evidence: []EvidenceRef{{EvidenceID: "implementation", StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}}, Uncertainty: []string{}}},
		Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []Question{},
		Scopes:   []ScopeProposal{{ID: "orders", Name: "Orders", ClaimIDs: []string{"claim"}}},
		Proposal: ModelProposal{Goal: "Record the proposed order scope", Files: []ProposedFile{{ScopeID: "orders", Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\n"}}},
	}
	report.Method = "agent-assisted"
	report.RunnerIdentity = "agentexec/fixture/model"
	report.RunnerDigest = strings.Repeat("d", 64)
	report.TargetBasis = target.ProjectDigest
	report.TargetRevision = target.Revision
	report.TargetContextDigest = target.Digest
	SealDistillation(&report)
	if err := ValidateDistillation(discovery, report); err != nil {
		t.Fatalf("complete target-bound agent report rejected: %v", err)
	}
	if err := ValidateDistillationTarget(report, targetProject); err != nil {
		t.Fatalf("matching target project rejected: %v", err)
	}
	unbound := report
	unbound.TargetBasis, unbound.TargetRevision, unbound.TargetContextDigest = "", "", ""
	SealDistillation(&unbound)
	if err := ValidateDistillation(discovery, unbound); err == nil {
		t.Fatal("agent-assisted report without target binding was accepted")
	}
	retargeted := report
	retargeted.TargetRevision = strings.Repeat("e", 40)
	SealDistillation(&retargeted)
	if err := ValidateDistillation(discovery, retargeted); err != nil {
		t.Fatalf("structurally complete retargeted report should validate before target comparison: %v", err)
	}
	if err := ValidateDistillationTarget(retargeted, targetProject); err == nil {
		t.Fatal("report bound to another target revision was accepted")
	}
	badPair := report
	badPair.Claims = append([]Claim(nil), report.Claims...)
	badPair.Claims[0].Method = "documentation"
	SealDistillation(&badPair)
	if err := ValidateDistillation(discovery, badPair); err == nil {
		t.Fatal("claim with a kind/method pair outside the exact allowed set was accepted")
	}
	badID := report
	badID.Scopes = append([]ScopeProposal(nil), report.Scopes...)
	badID.Claims = append([]Claim(nil), report.Claims...)
	badID.Proposal.Files = append([]ProposedFile(nil), report.Proposal.Files...)
	badID.Scopes[0].ID = "commerce.sales.orders"
	badID.Claims[0].ScopeID = "commerce.sales.orders"
	badID.Proposal.Files[0].ScopeID = "commerce.sales.orders"
	SealDistillation(&badID)
	if err := ValidateDistillation(discovery, badID); err == nil {
		t.Fatal("dotted namespace was accepted as a local scope ID")
	}
}

func TestNewPrivateLogDirectoryIsUniqueAndUncreated(t *testing.T) {
	parent := t.TempDir()
	first, err := newPrivateLogDirectory(parent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newPrivateLogDirectory(parent)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("private log directories collided: %q", first)
	}
	for _, candidate := range []string{first, second} {
		if _, err := os.Lstat(candidate); !os.IsNotExist(err) {
			t.Errorf("private log directory must be an uncreated leaf, path=%q error=%v", candidate, err)
		}
	}
}

func TestTargetContextRejectsUncommittedProject(t *testing.T) {
	root, _ := committedRepository(t, map[string]string{"README.md": "target\n"})
	gitRun(t, root, "checkout", "-b", "codex/project-adoption-provisional-target")
	if _, err := projectwork.Init(root, "Provisional target", true); err != nil {
		t.Fatal(err)
	}
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TargetContextForProject(project); err == nil || !strings.Contains(err.Error(), "fixed committed Project revision") {
		t.Fatalf("provisional target context error = %v", err)
	}
}

func distillationDiscovery(t *testing.T) (string, Discovery, DistillationTargetContext, *projectwork.Project) {
	t.Helper()
	root, commit := committedRepository(t, map[string]string{"src/orders/cancel.go": "package orders\nfunc Cancel() {}\n"})
	discovery, err := Discover(root, DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "runner-discovery", Purpose: "Assess the selected order source", Review: "review-52",
		Commit: commit, ScopeRoots: []string{"src/orders"},
		Selected:   []SelectedPath{{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Implementation evidence", Basis: "code"}},
		Exclusions: []PathReason{}, Unselected: []PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	targetRoot, _ := committedRepository(t, map[string]string{"README.md": "accepted target\n"})
	gitRun(t, targetRoot, "checkout", "-b", "codex/project-adoption-target-fixture")
	if _, err := projectwork.Init(targetRoot, "Target fixture", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, targetRoot, "add", "--all")
	gitRun(t, targetRoot, "commit", "--quiet", "-m", "initialize target project")
	targetRevision := gitRun(t, targetRoot, "rev-parse", "HEAD")
	targetProject, err := projectwork.Load(targetRoot, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	target, err := TargetContextForProject(targetProject)
	if err != nil {
		t.Fatal(err)
	}
	return root, discovery, target, targetProject
}

func testDistillationConfig(t *testing.T) agentexec.Config {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	allow := []string{distillationHelperEnv, distillationModeEnv}
	t.Setenv(distillationHelperEnv, "1")
	t.Setenv(distillationModeEnv, "")
	return agentexec.Config{
		Command: executable, Args: []string{"-test.run=^TestDistillationExecutorHelper$"},
		Model: "fixture-model", ModelOptions: json.RawMessage(`{"temperature":0}`), ProviderVersion: "fixture-provider-v1",
		EnvironmentAllowlist: &allow, Timeout: 5 * time.Second,
		MaxStdoutBytes: 2 << 20, MaxStderrBytes: 128 << 10,
	}
}

func testDistillationOptions(t *testing.T, target DistillationTargetContext) DistillationRunOptions {
	t.Helper()
	return DistillationRunOptions{
		MaxTimeout: 6 * time.Second, MaxStdoutBytes: 2 << 20, MaxStderrBytes: 128 << 10,
		MaxCostMicros: 1_000, InputPriceMicrosPerMillion: 10_000, OutputPriceMicrosPerMillion: 10_000,
		TempParent: os.TempDir(), TargetContext: target,
	}
}

func int64Pointer(value int64) *int64 { return &value }
