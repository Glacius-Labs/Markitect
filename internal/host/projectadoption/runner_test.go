package projectadoption

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
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
	artifact := invocation.Request.Artifacts[0]
	if artifact.Path != "evidence/implementation.txt" || string(artifact.Content) != "package orders\nfunc Cancel() {}\n" {
		os.Exit(34)
	}
	claim := Claim{ID: "implementation-observation", ScopeID: "orders", Kind: "observation", Method: "static-source",
		Statement:   "The selected source declares a cancellation function.",
		Evidence:    []EvidenceRef{{EvidenceID: "implementation", StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}},
		Uncertainty: []string{"This static source observation does not establish runtime behavior."}}
	draft := DistillationDraft{
		Claims: []Claim{claim}, Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []Question{},
		Scopes:   []ScopeProposal{{ID: "orders", Name: "Order management", ClaimIDs: []string{claim.ID}}},
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
	root, discovery := distillationDiscovery(t)
	selectedBefore, err := os.ReadFile(filepath.Join(root, "src", "orders", "cancel.go"))
	if err != nil {
		t.Fatal(err)
	}
	report, receipt, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), testDistillationOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Method != "agent-assisted" || report.DiscoveryDigest != discovery.Digest || report.SchemaDigest != receipt.SchemaDigest {
		t.Fatalf("generated report did not bind its source, method and schema: %+v", report)
	}
	if report.RunnerIdentity != receipt.RunnerIdentity || report.RunnerDigest != receipt.RunnerDigest || report.RunnerDigest != digestWithoutPrefix(receipt.Execution.ConfigDigest) || receipt.RunnerDigest == "" {
		t.Fatalf("report runner binding is not derived from the executed receipt: report=%+v receipt=%+v", report, receipt)
	}
	if receipt.Execution.Outcome != agentexec.OutcomeProposed || receipt.ExecutionReceiptDigest != digestValue(receipt.Execution) || receipt.EstimatedCostMicros != 2 {
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
	root, discovery := distillationDiscovery(t)
	for _, mode := range []string{"missing-usage", "bad-grounding", "over-budget"} {
		config := testDistillationConfig(t)
		t.Setenv(distillationModeEnv, mode)
		_, _, err := GenerateDistillation(context.Background(), root, discovery, config, testDistillationOptions(t))
		if err == nil {
			t.Errorf("mode %q should produce an incomplete or invalid distillation", mode)
		}
	}
}

func TestGenerateDistillationRejectsUnsafeBoundsAndStaleDiscovery(t *testing.T) {
	root, discovery := distillationDiscovery(t)
	config := testDistillationConfig(t)
	config.EnvironmentAllowlist = nil
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, config, testDistillationOptions(t)); err == nil || !strings.Contains(err.Error(), "explicit environment allowlist") {
		t.Fatalf("nil environment allowlist error = %v", err)
	}
	options := testDistillationOptions(t)
	options.MaxCostMicros = 0
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "cost ceiling") {
		t.Fatalf("unbounded cost options error = %v", err)
	}
	options = testDistillationOptions(t)
	options.TempParent = root
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "outside the source repository") {
		t.Fatalf("repository-local temporary parent error = %v", err)
	}
	options = testDistillationOptions(t)
	options.MaxStdoutBytes = maxDistillationStdout + 1
	if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "stdout limit") {
		t.Fatalf("unbounded stdout option error = %v", err)
	}
	alias := filepath.Join(t.TempDir(), "source-alias")
	if err := os.Symlink(root, alias); err == nil {
		options = testDistillationOptions(t)
		options.PrivateLogDirectory = alias
		if _, _, err := GenerateDistillation(context.Background(), root, discovery, testDistillationConfig(t), options); err == nil || !strings.Contains(err.Error(), "outside the source repository") {
			t.Fatalf("repository alias private-log directory error = %v", err)
		}
	}
	changed := discovery
	changed.Evidence = append([]Evidence(nil), discovery.Evidence...)
	changed.Evidence[0].Content += "changed"
	SealDiscovery(&changed)
	if _, _, err := GenerateDistillation(context.Background(), root, changed, testDistillationConfig(t), testDistillationOptions(t)); err == nil {
		t.Fatal("discovery with evidence that differs from the fixed commit should be rejected")
	}
}

func distillationDiscovery(t *testing.T) (string, Discovery) {
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
	return root, discovery
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

func testDistillationOptions(t *testing.T) DistillationRunOptions {
	t.Helper()
	private := t.TempDir()
	return DistillationRunOptions{
		MaxTimeout: 6 * time.Second, MaxStdoutBytes: 2 << 20, MaxStderrBytes: 128 << 10,
		MaxCostMicros: 1_000, InputPriceMicrosPerMillion: 10_000, OutputPriceMicrosPerMillion: 10_000,
		TempParent: os.TempDir(), PrivateLogDirectory: filepath.Join(private, "private-logs"),
	}
}

func int64Pointer(value int64) *int64 { return &value }
