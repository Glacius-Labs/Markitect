package projectcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"go.yaml.in/yaml/v3"
)

const distillHelperEnv = "MARKITECT_PROJECTCLI_DISTILL_HELPER"
const distillCalledEnv = "MARKITECT_PROJECTCLI_DISTILL_CALLED"
const distillBarrierEnv = "MARKITECT_PROJECTCLI_DISTILL_BARRIER"

func TestGeneratedDistillationPersistsOnlyReceiptWhenProposalIsRejected(t *testing.T) {
	repo, args := prepareGeneratedDistillationFixture(t)
	marker := filepath.Join(t.TempDir(), "provider-called")
	t.Setenv(distillHelperEnv, "1")
	t.Setenv(distillCalledEnv, marker)

	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code != 1 {
		t.Fatalf("rejected generated distillation exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	var result struct {
		Status      string `json:"status"`
		ReceiptPath string `json:"receiptPath"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Status != "rejected" || !strings.HasPrefix(result.ReceiptPath, ".markitect/drafts/distillation.receipt.") {
		t.Fatalf("rejected response omitted receipt status/path: %s", out.String())
	}
	if strings.Contains(errout.String(), "provider-private-failure") || strings.Contains(errout.String(), "decode executor reportJson") {
		t.Fatalf("CLI exposed private provider or report details: %s", errout.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("deterministic subprocess was not invoked: %v", err)
	}
	receiptBytes, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(result.ReceiptPath)))
	if err != nil {
		t.Fatalf("execution receipt was not persisted: %v", err)
	}
	var receipt projectadoption.DistillationReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatalf("decode execution receipt: %v", err)
	}
	if receipt.Execution.RunID == "" || receipt.EstimatedCostMicros <= 0 || receipt.RunnerIdentity != "agentexec/fixture-readonly/1/receipt-readonly-test" {
		t.Fatalf("persisted receipt lacks execution/cost accounting: %+v", receipt)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/drafts/distillation.json"))); !os.IsNotExist(err) {
		t.Fatalf("rejected proposal was persisted as an accepted report: %v", err)
	}
}

func TestGeneratedDistillationPreflightsBothOutputRecordsBeforeProvider(t *testing.T) {
	repo, args := prepareGeneratedDistillationFixture(t)
	marker := filepath.Join(t.TempDir(), "provider-called")
	t.Setenv(distillHelperEnv, "1")
	t.Setenv(distillCalledEnv, marker)
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(".markitect/drafts/distillation.json")), []byte("occupied"), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code == 0 || !strings.Contains(errout.String(), "refusing to overwrite existing Markitect record") {
		t.Fatalf("occupied output preflight exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("provider ran despite an occupied output path: %v", err)
	}
	if records := findDistillationReceiptPaths(t, repo); len(records) != 0 {
		t.Fatalf("output preflight failure wrote receipts: %v", records)
	}
}

func TestConcurrentGeneratedDistillationsPreserveBothUniqueReceipts(t *testing.T) {
	repo, args := prepareGeneratedDistillationFixture(t)
	barrier := filepath.Join(t.TempDir(), "barrier")
	if err := os.Mkdir(barrier, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(distillHelperEnv, "1")
	t.Setenv(distillCalledEnv, "")
	t.Setenv(distillBarrierEnv, barrier)
	type callResult struct {
		code   int
		out    string
		errout string
	}
	results := make(chan callResult, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var out, errout bytes.Buffer
			code := Run(append([]string(nil), args...), &out, &errout)
			results <- callResult{code: code, out: out.String(), errout: errout.String()}
		}()
	}
	wait.Wait()
	close(results)
	receiptPaths := map[string]bool{}
	for result := range results {
		if result.code != 1 {
			t.Fatalf("concurrent rejected call exit=%d stderr=%s stdout=%s", result.code, result.errout, result.out)
		}
		var record struct {
			Status      string `json:"status"`
			ReceiptPath string `json:"receiptPath"`
		}
		if err := json.Unmarshal([]byte(result.out), &record); err != nil || record.Status != "rejected" || record.ReceiptPath == "" {
			t.Fatalf("concurrent call omitted rejected receipt record: %s err=%v", result.out, err)
		}
		if receiptPaths[record.ReceiptPath] {
			t.Fatalf("concurrent invocations shared receipt path %q", record.ReceiptPath)
		}
		receiptPaths[record.ReceiptPath] = true
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(record.ReceiptPath)))
		if err != nil {
			t.Fatalf("concurrent receipt %s not preserved: %v", record.ReceiptPath, err)
		}
		var receipt projectadoption.DistillationReceipt
		if err := json.Unmarshal(data, &receipt); err != nil || receipt.Execution.RunID == "" {
			t.Fatalf("invalid concurrent receipt %s: runID=%q err=%v", record.ReceiptPath, receipt.Execution.RunID, err)
		}
	}
	if len(receiptPaths) != 2 {
		t.Fatalf("preserved %d unique receipts, want 2", len(receiptPaths))
	}
	entries, err := os.ReadDir(barrier)
	if err != nil || len(entries) != 2 {
		t.Fatalf("provider barrier observed %d invocations, want 2; err=%v", len(entries), err)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(".markitect/drafts/distillation.json"))); !os.IsNotExist(err) {
		t.Fatalf("invalid concurrent proposals wrote shared report: %v", err)
	}
}

func TestGeneratedDistillationRejectsNativeManagerWithoutReadOnlyBindingBeforeProvider(t *testing.T) {
	repo, args := prepareGeneratedDistillationFixtureWithReview(t, false)
	marker := filepath.Join(t.TempDir(), "provider-called")
	t.Setenv(distillHelperEnv, "1")
	t.Setenv(distillCalledEnv, marker)
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code == 0 || !strings.Contains(errout.String(), "requires a configured read-only assessment binding") {
		t.Fatalf("distillation without read-only binding exit=%d stderr=%s stdout=%s", code, errout.String(), out.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("provider ran without a read-only binding: %v", err)
	}
	if records := findDistillationReceiptPaths(t, repo); len(records) != 0 {
		t.Fatalf("missing read-only binding wrote receipts: %v", records)
	}
}

func TestProjectCLIDistillationHelperProcess(t *testing.T) {
	if os.Getenv(distillHelperEnv) != "1" {
		return
	}
	if marker := os.Getenv(distillCalledEnv); marker != "" {
		if err := os.WriteFile(marker, []byte("called"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, "test helper marker failed")
			os.Exit(9)
		}
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		fmt.Fprintln(os.Stderr, "test helper invocation failed")
		os.Exit(10)
	}
	if barrier := os.Getenv(distillBarrierEnv); barrier != "" {
		if err := os.WriteFile(filepath.Join(barrier, invocation.RunID), []byte("ready"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, "test helper barrier failed")
			os.Exit(12)
		}
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			entries, err := os.ReadDir(barrier)
			if err == nil && len(entries) >= 2 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		entries, err := os.ReadDir(barrier)
		if err != nil || len(entries) < 2 {
			fmt.Fprintln(os.Stderr, "test helper barrier timed out")
			os.Exit(13)
		}
	}
	inputTokens, outputTokens := int64(120), int64(45)
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeProposed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		ReportJSON: json.RawMessage(`{}`), Uncertainty: []string{},
		Usage: &agentexec.Usage{Source: "provider-reported", InputTokens: &inputTokens, OutputTokens: &outputTokens},
	}
	fmt.Fprintln(os.Stderr, "provider-private-failure")
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(11)
	}
	os.Exit(0)
}

func prepareGeneratedDistillationFixture(t *testing.T) (string, []string) {
	return prepareGeneratedDistillationFixtureWithReview(t, true)
}

func prepareGeneratedDistillationFixtureWithReview(t *testing.T, withReview bool) (string, []string) {
	t.Helper()
	repo := copyProjectWorld(t)
	initial, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	rootManager := ""
	for _, manager := range initial.Report.Managers {
		if manager.Parent == "" && manager.Namespace == "" {
			rootManager = manager.ID
			break
		}
	}
	if rootManager == "" {
		t.Fatal("project fixture has no root Manager")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	readOnlyAgent := projectrun.Agent{
		Command: executable, Args: []string{"-test.run=TestProjectCLIDistillationHelperProcess"},
		Model: "receipt-readonly-test", ProviderVersion: "fixture-readonly/1", Timeout: projectrun.Duration(20 * time.Second),
		MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
		Environment: []string{distillHelperEnv, distillCalledEnv, distillBarrierEnv},
		Pricing:     projectrun.Pricing{InputMicrosPerMillion: 2, OutputMicrosPerMillion: 4},
	}
	nativeAgent := readOnlyAgent
	nativeAgent.Model = "native-manager-must-not-run"
	nativeAgent.Args = []string{"-invalid-native-manager-transport"}
	nativeAgent.Environment = []string{}
	nativeAgent.WorkspaceMode = "scoped"
	nativeAgent.InstructionPaths = []string{"AGENTS.md"}
	runtimeConfig := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: map[string]projectrun.Agent{rootManager: nativeAgent},
		Limits: projectrun.Limits{
			MaxDepth: 2, MaxStarts: 24, MaxRetries: 0, MaxParallel: 1,
			MaxDuration: projectrun.Duration(2 * time.Minute), MaxCostMicros: 100000,
			MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 1 << 20,
		},
	}
	if withReview {
		runtimeConfig.Review = &projectrun.ReviewConfig{
			Agents: map[string]projectrun.Agent{rootManager: readOnlyAgent}, MaxRounds: 1, MaxManagerRounds: 1,
		}
	}
	runtimeBytes, err := yaml.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(projectwork.RuntimePath)), runtimeBytes, 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", projectwork.RuntimePath)
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "configure deterministic test runner")
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "receipt-discovery", Purpose: "Test generated receipt persistence",
		Review: "receipt-review", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation", Path: "docs/cancellation.md", Reason: "Fixture evidence", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	}
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	discoveryBytes, err := projectadoption.EncodeDiscovery(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/distillation-discovery.json", discoveryBytes); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"distill", "--repo", repo, "--discovery", ".markitect/drafts/distillation-discovery.json",
		"--generate", "--write", "--output", ".markitect/drafts/distillation.json",
		"--input-micros-per-million", "2", "--output-micros-per-million", "4", "--max-cost-micros", "100000",
	}
	return repo, args
}

func findDistillationReceiptPaths(t *testing.T, repo string) []string {
	t.Helper()
	directory := filepath.Join(repo, filepath.FromSlash(".markitect/drafts"))
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var result []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "distillation.receipt.") && strings.HasSuffix(entry.Name(), ".json") {
			result = append(result, filepath.ToSlash(filepath.Join(".markitect/drafts", entry.Name())))
		}
	}
	return result
}
