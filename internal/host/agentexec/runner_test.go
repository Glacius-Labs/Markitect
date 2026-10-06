package agentexec

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const helperEnv = "MARKITECT_AGENTEXEC_TEST_HELPER"

func TestAgentexecHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	mode := os.Getenv("MARKITECT_AGENTEXEC_TEST_MODE")
	if mode == "timeout" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	if mode == "overflow" {
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", 1024)))
		os.Exit(0)
	}
	if mode == "failure" {
		fmt.Fprintln(os.Stderr, "provider-private-failure")
		os.Exit(7)
	}
	var invocation Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		fmt.Fprintln(os.Stderr, "bad request")
		os.Exit(8)
	}
	if mode == "mutate" {
		root := os.Getenv("MARKITECT_AGENTEXEC_TEST_ROOT")
		_ = os.WriteFile(filepath.Join(root, "input.txt"), []byte("changed"), 0600)
	}
	switch mode {
	case "malformed":
		fmt.Fprint(os.Stdout, `{"role":"executor","role":"verifier"}`)
		os.Exit(0)
	case "unknown":
		fmt.Fprint(os.Stdout, `{"unexpected":true}`)
		os.Exit(0)
	}
	role := invocation.Request.Role
	if override := os.Getenv("MARKITECT_AGENTEXEC_TEST_ROLE"); override != "" {
		role = override
	}
	inputDigest := invocation.InputDigest
	if mode == "wrong-digest" {
		inputDigest = strings.Repeat("0", len(inputDigest))
	}
	outcome := OutcomeProposed
	response := Response{
		APIVersion:           APIVersion,
		RunID:                invocation.RunID,
		Nonce:                invocation.Nonce,
		Role:                 role,
		InputDigest:          inputDigest,
		Outcome:              outcome,
		CandidateFiles:       []CandidateFile{{Path: "candidate.txt", Mode: "0644", Content: "candidate bytes"}},
		EvidenceRefs:         []string{},
		VerifierObservations: []Observation{},
		Uncertainty:          []string{},
	}
	if mode == "bad-evidence" {
		response.EvidenceRefs = []string{"README.md"}
	}
	if mode == "allowed-evidence" {
		response.EvidenceRefs = []string{"scope/a", "policy/a"}
	}
	if role == RoleVerifier {
		response.Outcome = OutcomePassed
		response.CandidateFiles = []CandidateFile{}
		response.VerifierObservations = []Observation{{Subject: "candidate", Outcome: OutcomePassed, Detail: "independent observation"}}
	}
	if role == RoleInfer {
		response.CandidateFiles = []CandidateFile{}
		response.CandidateJSON = json.RawMessage(`{"proposal":"value"}`)
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func testRequest(role string) Request {
	return Request{
		Role:           role,
		SourceRevision: strings.Repeat("a", 40),
		ModelDigest:    "sha256:" + strings.Repeat("b", 64),
		ModulePin:      "example@sha256:" + strings.Repeat("c", 64),
		ProjectionID:   "projection/example",
		ScopeIDs:       []string{"scope/z", "scope/a"},
		PolicyIDs:      []string{"policy/a"},
		Context:        json.RawMessage(`{"number":1,"kind":"Projection"}`),
		Artifacts:      []Artifact{},
	}
}

func testConfig() Config {
	return Config{
		Command:         os.Args[0],
		Args:            []string{"-test.run=TestAgentexecHelperProcess"},
		Model:           "test-model",
		ModelOptions:    json.RawMessage(`{"effort":"high"}`),
		ProviderVersion: "test-helper/1",
		Timeout:         3 * time.Second,
		MaxStdoutBytes:  4096,
		MaxStderrBytes:  4096,
	}
}

func testOptions(t *testing.T) RunOptions {
	t.Helper()
	root := t.TempDir()
	private := filepath.Join(t.TempDir(), "private-logs")
	t.Setenv(helperEnv, "1")
	t.Setenv("MARKITECT_AGENT_PRIVATE_LOG", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROOT", root)
	return RunOptions{InputRoots: []string{root}, TempParent: t.TempDir(), PrivateLogDirectory: private}
}

func runHelper(t *testing.T, mode string, role string, request Request, config Config) (RunResult, error) {
	t.Helper()
	opts := testOptions(t)
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", mode)
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", role)
	return Run(context.Background(), config, request, opts)
}

func TestRunBindsIndependentRolesAndDigests(t *testing.T) {
	execResult, err := runHelper(t, "", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if execResult.Response.Role != RoleExecutor || execResult.Response.InputDigest != execResult.Receipt.InputDigest {
		t.Fatalf("executor response not bound to request: %#v", execResult)
	}
	if len(execResult.Response.CandidateFiles) != 1 || execResult.Receipt.RetryCount != 0 {
		t.Fatalf("unexpected executor result: %#v", execResult)
	}
	verifierResult, err := runHelper(t, "", "", testRequest(RoleVerifier), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if verifierResult.Response.Role != RoleVerifier || verifierResult.Response.Outcome != OutcomePassed ||
		len(verifierResult.Response.CandidateFiles) != 0 || len(verifierResult.Response.VerifierObservations) == 0 {
		t.Fatalf("verifier result reused executor candidate output: %#v", verifierResult.Response)
	}
	inferResult, err := runHelper(t, "", "", testRequest(RoleInfer), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if inferResult.Response.CandidateJSON == nil || len(inferResult.Response.CandidateFiles) != 0 {
		t.Fatalf("unexpected inference result: %#v", inferResult.Response)
	}
}

func TestRunAllowsUnconfiguredInputRoots(t *testing.T) {
	options := testOptions(t)
	options.InputRoots = nil
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	result, err := Run(context.Background(), testConfig(), testRequest(RoleExecutor), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Response.Outcome != OutcomeProposed {
		t.Fatalf("unexpected result without an input-root audit: %#v", result.Response)
	}
}

func TestRunRejectsCrossRoleAndWrongDigestResponses(t *testing.T) {
	_, err := runHelper(t, "", RoleVerifier, testRequest(RoleExecutor), testConfig())
	if err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("expected cross-role rejection, got %v", err)
	}
	_, err = runHelper(t, "wrong-digest", "", testRequest(RoleExecutor), testConfig())
	if err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("expected wrong digest rejection, got %v", err)
	}
}

func TestRunRestrictsEvidenceReferencesToSuppliedInputs(t *testing.T) {
	if _, err := runHelper(t, "bad-evidence", "", testRequest(RoleExecutor), testConfig()); err == nil ||
		!strings.Contains(err.Error(), "not supplied") {
		t.Fatalf("expected unsupplied evidence reference rejection, got %v", err)
	}
	result, err := runHelper(t, "allowed-evidence", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Response.EvidenceRefs) != 2 {
		t.Fatalf("supplied scope and policy references were not accepted: %#v", result.Response.EvidenceRefs)
	}
}

func TestRunRejectsDuplicateUnknownAndMalformedJSON(t *testing.T) {
	for _, mode := range []string{"malformed", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			_, err := runHelper(t, mode, "", testRequest(RoleExecutor), testConfig())
			if err == nil || !strings.Contains(err.Error(), "invalid response") {
				t.Fatalf("expected invalid response error, got %v", err)
			}
		})
	}
	if err := rejectDuplicateKeys([]byte(`{"x":1,"x":2}`)); err == nil {
		t.Fatal("duplicate nested JSON key was accepted")
	}
}

func TestRunRecordsFailureTimeoutAndOutputLimitWithoutRetry(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		result, err := runHelper(t, "failure", "", testRequest(RoleExecutor), testConfig())
		if err == nil || result.Receipt.Outcome != OutcomeFailed || result.Receipt.RetryCount != 0 {
			t.Fatalf("expected one recorded process failure, result=%#v err=%v", result, err)
		}
		if strings.Contains(err.Error(), "provider-private-failure") {
			t.Fatal("private provider diagnostics escaped into public error")
		}
	})
	t.Run("timeout", func(t *testing.T) {
		config := testConfig()
		config.Timeout = 100 * time.Millisecond
		result, err := runHelper(t, "timeout", "", testRequest(RoleExecutor), config)
		if err == nil || result.Receipt.Outcome != OutcomeIncomplete || result.Receipt.RetryCount != 0 {
			t.Fatalf("expected incomplete timeout receipt, result=%#v err=%v", result, err)
		}
	})
	t.Run("overlimit", func(t *testing.T) {
		config := testConfig()
		config.MaxStdoutBytes = 16
		result, err := runHelper(t, "overflow", "", testRequest(RoleExecutor), config)
		if !errors.Is(err, ErrOutputTooLarge) || result.Receipt.Outcome != OutcomeIncomplete {
			t.Fatalf("expected bounded output failure, result=%#v err=%v", result, err)
		}
	})
}

func TestRunAuditsInputRootAfterProcess(t *testing.T) {
	opts := testOptions(t)
	root := opts.InputRoots[0]
	if err := os.WriteFile(filepath.Join(root, "input.txt"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "mutate")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	result, err := Run(context.Background(), testConfig(), testRequest(RoleExecutor), opts)
	if !errors.Is(err, ErrInputChanged) {
		t.Fatalf("expected input mutation rejection, got %v", err)
	}
	if result.Receipt.RetryCount != 0 {
		t.Fatalf("unexpected retry: %#v", result.Receipt)
	}
}

func TestRunRuntimeFileChangesConfigurationDigest(t *testing.T) {
	root := t.TempDir()
	opts := RunOptions{InputRoots: []string{root}, TempParent: t.TempDir(), PrivateLogDirectory: filepath.Join(t.TempDir(), "logs")}
	runtimeDir := t.TempDir()
	script := filepath.Join(runtimeDir, "runner.py")
	content := []byte("print('one')")
	if err := os.WriteFile(script, content, 0644); err != nil {
		t.Fatal(err)
	}
	config := testConfig()
	config.RuntimeFiles = []RuntimeFile{{Path: script, Mode: "0644", Digest: digest(content)}}
	fingerprinted, err := Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperEnv, "1")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	first, err := Run(context.Background(), config, testRequest(RoleExecutor), opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.ConfigDigest != fingerprinted {
		t.Fatalf("Fingerprint returned %s, receipt recorded %s", fingerprinted, first.Receipt.ConfigDigest)
	}
	content = []byte("print('two')")
	if err := os.WriteFile(script, content, 0644); err != nil {
		t.Fatal(err)
	}
	config.RuntimeFiles[0].Digest = digest(content)
	updatedFingerprint, err := Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(context.Background(), config, testRequest(RoleExecutor), opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.ConfigDigest == second.Receipt.ConfigDigest || second.Receipt.ConfigDigest != updatedFingerprint ||
		first.Receipt.RuntimeFilesDigest == second.Receipt.RuntimeFilesDigest {
		t.Fatal("changing the selected runner wrapper did not invalidate config identity")
	}
}

func TestRuntimeFileCombinedBoundRejectsOversizedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized-runtime.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxRuntimeFileBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, err = snapshotRuntimeFiles([]RuntimeFile{{Path: path, Mode: "0644", Digest: "sha256:" + strings.Repeat("0", 64)}})
	if err == nil || !strings.Contains(err.Error(), "256 MiB bound") {
		t.Fatalf("expected bounded oversized runtime rejection, got %v", err)
	}
}

func TestPortableArtifactAliasesRejected(t *testing.T) {
	request := testRequest(RoleExecutor)
	request.Artifacts = []Artifact{
		{Path: "Readme.md", Mode: "0644", Content: []byte("a"), Digest: digest([]byte("a"))},
		{Path: "README.MD", Mode: "0644", Content: []byte("b"), Digest: digest([]byte("b"))},
	}
	if _, _, err := normalizeRequest(request); err == nil {
		t.Fatal("case-folded artifact path alias was accepted")
	}
}

func TestEmptyArtifactIsEncodedAsEmptyBase64(t *testing.T) {
	request := testRequest(RoleExecutor)
	request.Artifacts = []Artifact{{Path: "empty.txt", Mode: "0644", Digest: digest(nil)}}
	_, encoded, err := normalizeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Artifacts []map[string]any `json:"artifacts"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Artifacts[0]["content"] != "" {
		t.Fatalf("empty artifact content must encode as base64 empty string, got %#v", decoded.Artifacts[0]["content"])
	}
}

func TestRunnerReceivesOneClosedInvocationOnStdin(t *testing.T) {
	result, err := runHelper(t, "", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Receipt.InputDigest, "sha256:") ||
		result.Receipt.CommandDigest == "" || result.Receipt.ExecutableDigest == "" ||
		result.Receipt.ConfigDigest == "" || result.Receipt.ContextDigest == "" {
		t.Fatalf("receipt is missing invocation digests: %#v", result.Receipt)
	}
	_ = bufio.ErrInvalidUnreadByte
	_ = exec.ErrNotFound
}
