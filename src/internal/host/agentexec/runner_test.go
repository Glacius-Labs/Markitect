package agentexec

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	helperEnv = "MARKITECT_AGENTEXEC_TEST_HELPER"

	descendantFixtureReadinessTimeout = 10 * time.Second
	descendantFixtureRunTimeout       = 15 * time.Second
	descendantFixtureIntentionalWait  = 30 * time.Second
)

func TestAgentexecHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	mode := os.Getenv("MARKITECT_AGENTEXEC_TEST_MODE")
	if mode == "failure" {
		fmt.Fprintln(os.Stderr, "provider-private-failure")
		os.Exit(7)
	}
	var invocation Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		fmt.Fprintln(os.Stderr, "bad request")
		os.Exit(8)
	}
	if mode == "private-log-oversized" || mode == "private-log-empty" {
		file, err := os.Create(os.Getenv("MARKITECT_AGENT_PRIVATE_LOG"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "private log create failed")
			os.Exit(11)
		}
		if mode == "private-log-oversized" {
			if err := file.Truncate(maxOutputBound + 1); err != nil {
				_ = file.Close()
				fmt.Fprintln(os.Stderr, "private log truncate failed")
				os.Exit(12)
			}
		}
		if err := file.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "private log close failed")
			os.Exit(13)
		}
	}
	if mode == "timeout" || mode == "overflow" {
		if mode == "timeout" {
			time.Sleep(5 * time.Second)
			os.Exit(0)
		}
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", 1024)))
		os.Exit(0)
	}
	if strings.HasPrefix(mode, "spawn-child-") {
		command := exec.Command(os.Args[0], "-test.run=TestAgentexecHeartbeatChild")
		command.Env = append(os.Environ(), "MARKITECT_AGENTEXEC_HEARTBEAT="+os.Getenv("MARKITECT_AGENTEXEC_TEST_HEARTBEAT"))
		if err := command.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "child start failed")
			os.Exit(9)
		}
		heartbeat := os.Getenv("MARKITECT_AGENTEXEC_TEST_HEARTBEAT")
		deadline := time.Now().Add(descendantFixtureReadinessTimeout)
		ready := false
		for time.Now().Before(deadline) {
			if info, err := os.Stat(heartbeat); err == nil && info.Size() > 0 {
				ready = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !ready {
			fmt.Fprintln(os.Stderr, "heartbeat child did not become ready")
			os.Exit(10)
		}
		if mode == "spawn-child-timeout" || mode == "spawn-child-overflow" {
			if mode == "spawn-child-overflow" {
				_, _ = os.Stdout.Write([]byte(strings.Repeat("x", 1024)))
			}
			time.Sleep(descendantFixtureIntentionalWait)
			os.Exit(0)
		}
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
	if mode == "report-json" {
		response.ReportJSON = json.RawMessage(`{"status":"complete","summary":"task ended"}`)
	}
	if mode == "report-only" {
		response.CandidateFiles = []CandidateFile{}
		response.ReportJSON = json.RawMessage(`{"status":"complete","summary":"no file changes"}`)
	}
	if mode == "empty-proposed" {
		response.CandidateFiles = []CandidateFile{}
	}
	if mode == "bad-report-json" {
		response.ReportJSON = json.RawMessage(`[]`)
	}
	if mode == "wrong-role-report-json" {
		response.ReportJSON = json.RawMessage(`{"status":"complete"}`)
	}
	if mode == "bad-evidence" {
		response.EvidenceRefs = []string{"README.md"}
	}
	if mode == "allowed-evidence" {
		response.EvidenceRefs = []string{"scope/a", "policy/a"}
	}
	if strings.HasPrefix(mode, "native-work") {
		candidatePath := "src/a& café\u2028.go"
		response.CandidateFiles = []CandidateFile{{Path: candidatePath, Mode: "0644", Content: "value <>& café\u2028 line"}}
		contentDigest := sha256.Sum256([]byte(response.CandidateFiles[0].Content))
		deltaJSON := fmt.Sprintf(`[{"digest":"sha256:%x","mode":"0644","path":"%s"}]`, contentDigest, candidatePath)
		response.NativeWork = &NativeWork{
			WorkspaceBaseDigest:  "sha256:" + strings.Repeat("0", 64),
			WorkspaceFinalDigest: "sha256:" + strings.Repeat("1", 64),
			DeltaDigest:          digest([]byte(deltaJSON)),
			ChangedPaths:         []string{candidatePath},
			ToolCalls:            2,
			HelperStarts:         0,
			HelperAccounting:     "disabled",
		}
		switch mode {
		case "native-work-bad-digest":
			response.NativeWork.DeltaDigest = "sha256:" + strings.Repeat("2", 64)
		case "native-work-tampered-paths":
			response.NativeWork.ChangedPaths = []string{"src/other.go"}
		case "native-work-unsorted":
			response.NativeWork.ChangedPaths = []string{"src/z.go", "src/a.go"}
		case "native-work-unsafe-path":
			response.NativeWork.ChangedPaths = []string{"../escape"}
		case "native-work-negative-tools":
			response.NativeWork.ToolCalls = -1
		case "native-work-helper-start":
			response.NativeWork.HelperStarts = 1
		case "native-work-helper-unknown":
			response.NativeWork.HelperAccounting = "reported"
		}
	}
	if mode == "capture-agent-config" || mode == "native-work-capture-agent-config" {
		response.CandidateFiles[0].Content = os.Getenv("MARKITECT_AGENT_CONFIG_JSON")
		if response.NativeWork != nil {
			response.NativeWork.DeltaDigest, _ = nativeDeltaDigest(response.CandidateFiles)
		}
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

func TestAgentexecHeartbeatChild(t *testing.T) {
	path := os.Getenv("MARKITECT_AGENTEXEC_HEARTBEAT")
	if path == "" {
		return
	}
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = file.Write([]byte{'.'})
			_ = file.Close()
		}
		time.Sleep(20 * time.Millisecond)
	}
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

func nativeTaskRequest() Request {
	request := testRequest(RoleExecutor)
	request.Context = json.RawMessage(`{"kind":"projectrun-task/v1","phase":"work","nativeWorkspace":{"apiVersion":"markitect.example.org/native-workspace/v1","instructions":[{"path":"AGENTS.md"}]}}`)
	return request
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

func TestExecutorAcceptsOptionalTypedReportJSON(t *testing.T) {
	result, err := runHelper(t, "report-json", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Response.CandidateFiles) != 1 || string(result.Response.ReportJSON) != `{"status":"complete","summary":"task ended"}` {
		t.Fatalf("executor did not retain candidate files and task report independently: %#v", result.Response)
	}
	legacy, err := runHelper(t, "", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatalf("legacy executor response without reportJson was rejected: %v", err)
	}
	if len(legacy.Response.ReportJSON) != 0 {
		t.Fatalf("legacy executor unexpectedly gained a task report: %#v", legacy.Response)
	}
}

func TestExecutorMayProposeReportWithoutCandidateFiles(t *testing.T) {
	result, err := runHelper(t, "report-only", "", testRequest(RoleExecutor), testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Response.CandidateFiles) != 0 || len(result.Response.ReportJSON) == 0 {
		t.Fatalf("report-only proposal was not preserved: %#v", result.Response)
	}
	if _, err := runHelper(t, "empty-proposed", "", testRequest(RoleExecutor), testConfig()); err == nil {
		t.Fatal("proposed executor response with neither files nor reportJson was accepted")
	}
}

func TestReportJSONRejectsWrongRoleAndMalformedValues(t *testing.T) {
	if _, err := runHelper(t, "bad-report-json", "", testRequest(RoleExecutor), testConfig()); err == nil {
		t.Fatal("non-object reportJson was accepted")
	}
	if _, err := runHelper(t, "wrong-role-report-json", "", testRequest(RoleVerifier), testConfig()); err == nil {
		t.Fatal("verifier reportJson was accepted")
	}
	if _, err := runHelper(t, "wrong-role-report-json", "", testRequest(RoleInfer), testConfig()); err == nil {
		t.Fatal("inference reportJson was accepted")
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

func TestScopedNativeWorkMetadataIsValidatedAndCopiedToReceipt(t *testing.T) {
	config := testConfig()
	config.WorkspaceMode = "scoped"
	result, err := runHelper(t, "native-work", "", nativeTaskRequest(), config)
	if err != nil {
		t.Fatal(err)
	}
	if result.Response.NativeWork == nil || result.Receipt.NativeWork == nil ||
		result.Response.NativeWork.DeltaDigest != result.Receipt.NativeWork.DeltaDigest {
		t.Fatalf("native work metadata did not reach response and receipt: response=%#v receipt=%#v", result.Response.NativeWork, result.Receipt.NativeWork)
	}
	contentDigest := sha256.Sum256([]byte("value <>& café\u2028 line"))
	wantDeltaJSON := fmt.Sprintf(`[{"digest":"sha256:%x","mode":"0644","path":"%s"}]`, contentDigest, "src/a& café\u2028.go")
	if result.Response.NativeWork.DeltaDigest != digest([]byte(wantDeltaJSON)) {
		t.Fatal("native delta digest differs from the Python canonical JSON contract for UTF-8 content")
	}
	result.Response.NativeWork.ChangedPaths[0] = "mutated-after-run"
	if result.Receipt.NativeWork.ChangedPaths[0] != "src/a& café\u2028.go" {
		t.Fatal("receipt native work changedPaths aliases the response")
	}

	if _, err := runHelper(t, "", "", nativeTaskRequest(), config); err == nil || !strings.Contains(err.Error(), "requires nativeWork") {
		t.Fatalf("scoped proposal without nativeWork was accepted: %v", err)
	}
}

func TestNativeDeltaDigestMatchesPythonCompactUTF8Contract(t *testing.T) {
	path := "src/<>& café\u2028.go"
	files := []CandidateFile{
		{Path: "z.txt", Mode: "0644", Content: "last"},
		{Path: path, Mode: "0755", Content: "<>& café\u2028"},
	}
	firstContentDigest := sha256.Sum256([]byte("<>& café\u2028"))
	secondContentDigest := sha256.Sum256([]byte("last"))
	expectedJSON := fmt.Sprintf(
		`[{"digest":"sha256:%x","mode":"0755","path":"%s"},{"digest":"sha256:%x","mode":"0644","path":"z.txt"}]`,
		firstContentDigest, path, secondContentDigest,
	)
	expectedBytesDigest := sha256.Sum256([]byte(expectedJSON))
	want := "sha256:" + hex.EncodeToString(expectedBytesDigest[:])
	got, err := nativeDeltaDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("native delta digest = %s, want Python canonical UTF-8 digest %s", got, want)
	}
}

func TestNativeWorkMetadataIsRestrictedAndStrictlyValidated(t *testing.T) {
	if _, err := runHelper(t, "native-work", "", testRequest(RoleExecutor), testConfig()); err == nil || !strings.Contains(err.Error(), "requires scoped") {
		t.Fatalf("nativeWork on a legacy proposal flow was accepted: %v", err)
	}

	for _, mode := range []string{"native-work-bad-digest", "native-work-tampered-paths", "native-work-unsorted", "native-work-unsafe-path", "native-work-negative-tools", "native-work-helper-start", "native-work-helper-unknown"} {
		t.Run(mode, func(t *testing.T) {
			config := testConfig()
			config.WorkspaceMode = "scoped"
			if _, err := runHelper(t, mode, "", nativeTaskRequest(), config); err == nil {
				t.Fatalf("malformed nativeWork was accepted")
			}
		})
	}

	config := testConfig()
	config.WorkspaceMode = "scoped"
	if _, err := runHelper(t, "", "", testRequest(RoleVerifier), config); err == nil || !strings.Contains(err.Error(), "only for projectrun Manager executor") {
		t.Fatalf("scoped native workspace was accepted for verifier: %v", err)
	}
	if _, err := runHelper(t, "", "", testRequest(RoleExecutor), config); err == nil || !strings.Contains(err.Error(), "projectrun-task/v1") {
		t.Fatalf("scoped native workspace was accepted without native context: %v", err)
	}
}

func TestWorkspaceModeIsBoundIntoConfigurationFingerprint(t *testing.T) {
	legacy := testConfig()
	scoped := testConfig()
	scoped.WorkspaceMode = "scoped"
	legacyFingerprint, err := Fingerprint(legacy)
	if err != nil {
		t.Fatal(err)
	}
	scopedFingerprint, err := Fingerprint(scoped)
	if err != nil {
		t.Fatal(err)
	}
	if legacyFingerprint == scopedFingerprint {
		t.Fatal("scoped workspace mode did not change the normalized configuration fingerprint")
	}
	invalid := testConfig()
	invalid.WorkspaceMode = "unrestricted"
	if _, err := Fingerprint(invalid); err == nil {
		t.Fatal("unsupported workspace mode was accepted")
	}
}

func TestNativeWindowsSandboxBackendIsClosedAndFingerprintBound(t *testing.T) {
	config := testConfig()
	config.Transport = "codex-app-server"
	config.ModelOptions = nil
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":false,"maxStartRequests":0,"maxDepth":0},"maxEventBytes":1024}`)
	baseline, err := Fingerprint(config)
	if err != nil {
		t.Fatalf("fingerprint inherited backend: %v", err)
	}
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","windowsSandboxBackend":"mxc","helpers":{"enabled":false,"maxStartRequests":0,"maxDepth":0},"maxEventBytes":1024}`)
	withMXC, err := Fingerprint(config)
	if err != nil || baseline == withMXC {
		t.Fatalf("mxc backend was not accepted and bound: fingerprint=%q err=%v", withMXC, err)
	}
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","windowsSandboxBackend":"unsafe","helpers":{"enabled":false,"maxStartRequests":0,"maxDepth":0},"maxEventBytes":1024}`)
	if _, err := Fingerprint(config); err == nil || !strings.Contains(err.Error(), "unsupported Windows sandbox backend") {
		t.Fatalf("unsupported backend was accepted: %v", err)
	}
}

func TestWorkspaceModeDoesNotChangeProviderConfigEnvironmentContract(t *testing.T) {
	config := testConfig()
	config.WorkspaceMode = "scoped"
	result, err := runHelper(t, "native-work-capture-agent-config", "", nativeTaskRequest(), config)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"model":"test-model","modelOptions":{"effort":"high"},"providerVersion":"test-helper/1"}`
	if result.Response.CandidateFiles[0].Content != want {
		t.Fatalf("provider config environment contract changed: got %s, want %s", result.Response.CandidateFiles[0].Content, want)
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
	var response Response
	aliasedProtocolKeys := []byte(`{"apiVersion":"` + APIVersion + `","runId":"wrong","RunID":"expected","nonce":"n","role":"executor","inputDigest":"sha256:` + strings.Repeat("0", 64) + `","outcome":"incomplete","candidateFiles":[],"evidenceRefs":[],"verifierObservations":[],"uncertainty":[]}`)
	if err := strictDecode(aliasedProtocolKeys, &response); err == nil || !strings.Contains(err.Error(), "duplicate protocol JSON key") {
		t.Fatalf("case-variant protocol fields were accepted: %v", err)
	}
	opaqueCandidate := []byte(`{"apiVersion":"` + APIVersion + `","runId":"r","nonce":"n","role":"infer","inputDigest":"sha256:` + strings.Repeat("0", 64) + `","outcome":"incomplete","candidateFiles":[],"evidenceRefs":[],"verifierObservations":[],"candidateJson":{"Property":1,"property":2},"uncertainty":[]}`)
	if err := strictDecode(opaqueCandidate, &response); err != nil {
		t.Fatalf("case-distinct opaque candidate properties were rejected: %v", err)
	}
}

func TestStrictDecodeRejectsInvalidUTF8Response(t *testing.T) {
	data := []byte(`{"candidateFiles":[{"path":"candidate.txt","mode":"0644","content":"`)
	data = append(data, 0xff)
	data = append(data, []byte(`"}]}`)...)
	var response Response
	if err := strictDecode(data, &response); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("invalid UTF-8 response was not rejected before decoding: %v", err)
	}
}

func TestNormalizeRootsPreservesDistinctCaseSensitivePaths(t *testing.T) {
	parent := t.TempDir()
	upper := filepath.Join(parent, "Input")
	lower := filepath.Join(parent, "input")
	if err := os.Mkdir(upper, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lower, 0700); err != nil {
		if os.IsExist(err) {
			t.Skip("filesystem is case-insensitive")
		}
		t.Fatal(err)
	}
	upperInfo, err := os.Stat(upper)
	if err != nil {
		t.Fatal(err)
	}
	lowerInfo, err := os.Stat(lower)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(upperInfo, lowerInfo) {
		t.Skip("filesystem aliases paths that differ only by case")
	}

	roots, err := normalizeRoots([]string{upper, lower})
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 {
		t.Fatalf("distinct case-sensitive roots collapsed: %v", roots)
	}
}

func TestRunRejectsOversizedPrivateLog(t *testing.T) {
	options := testOptions(t)
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "private-log-oversized")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	result, err := Run(context.Background(), testConfig(), testRequest(RoleExecutor), options)
	if err == nil || !strings.Contains(err.Error(), "private log could not be read") {
		t.Fatalf("expected explicit private-log read failure, result=%#v err=%v", result, err)
	}
	if result.Receipt.PrivateLogDigest != "" {
		t.Fatalf("unreadable private log unexpectedly has a digest: %#v", result.Receipt)
	}
}

func TestRunDigestsPresentEmptyPrivateLog(t *testing.T) {
	options := testOptions(t)
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "private-log-empty")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	result, err := Run(context.Background(), testConfig(), testRequest(RoleExecutor), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.PrivateLogDigest != digest(nil) {
		t.Fatalf("empty private log digest = %q, want %q", result.Receipt.PrivateLogDigest, digest(nil))
	}
}

func TestVerifyExecutableDigestDetectsPersistentChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.bin")
	original := []byte("runner-one")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyExecutableDigest(path, digest(original)); err != nil {
		t.Fatalf("unchanged executable failed verification: %v", err)
	}
	if err := os.WriteFile(path, []byte("runner-two"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyExecutableDigest(path, digest(original)); err == nil {
		t.Fatal("persistent executable change was not detected")
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
		if !strings.Contains(err.Error(), "configured role timeout") || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout should identify the role deadline while retaining the sentinel: %v", err)
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

func TestNormalizeConfigAcceptsSixtyMinuteRoleTimeoutAndRejectsLonger(t *testing.T) {
	config := testConfig()
	config.Timeout = time.Hour
	if _, err := normalizeConfig(config); err != nil {
		t.Fatalf("sixty-minute role timeout should be accepted: %v", err)
	}
	config.Timeout = time.Hour + time.Nanosecond
	if _, err := normalizeConfig(config); err == nil || !strings.Contains(err.Error(), "at most sixty minutes") {
		t.Fatalf("role timeout above sixty minutes should be rejected clearly: %v", err)
	}
}

func TestRunStopsDescendantProcessesOnTimeoutOverflowAndNormalExit(t *testing.T) {
	for _, mode := range []string{"spawn-child-timeout", "spawn-child-overflow", "spawn-child-normal"} {
		t.Run(mode, func(t *testing.T) {
			opts := testOptions(t)
			heartbeat := filepath.Join(t.TempDir(), "heartbeat")
			t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", mode)
			t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
			t.Setenv("MARKITECT_AGENTEXEC_TEST_HEARTBEAT", heartbeat)
			config := testConfig()
			// All descendant modes get the same bounded run budget. Readiness has
			// its own allowance; the timeout fixture's longer intentional wait keeps
			// runtime timeout behavior distinct from hosted-Windows startup latency.
			config.Timeout = descendantFixtureRunTimeout
			if mode == "spawn-child-overflow" {
				config.MaxStdoutBytes = 16
			}
			result, err := Run(context.Background(), config, testRequest(RoleExecutor), opts)
			switch mode {
			case "spawn-child-timeout":
				if err == nil || !strings.Contains(err.Error(), "configured role timeout") || !errors.Is(err, context.DeadlineExceeded) || result.Receipt.Outcome != OutcomeIncomplete {
					t.Fatalf("expected incomplete timeout, result=%#v err=%v", result, err)
				}
			case "spawn-child-overflow":
				if !errors.Is(err, ErrOutputTooLarge) || result.Receipt.Outcome != OutcomeIncomplete {
					t.Fatalf("expected bounded output failure, result=%#v err=%v", result, err)
				}
			case "spawn-child-normal":
				if err != nil || result.Response.Outcome != OutcomeProposed {
					t.Fatalf("expected successful parent result, result=%#v err=%v", result, err)
				}
			}
			before := heartbeatSize(t, heartbeat)
			time.Sleep(150 * time.Millisecond)
			after := heartbeatSize(t, heartbeat)
			if after != before {
				t.Fatalf("descendant remained active after Run returned: heartbeat grew from %d to %d", before, after)
			}
		})
	}
}

func heartbeatSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("heartbeat was not started: %v", err)
	}
	return info.Size()
}

func TestPrivateLogPathWithinInputRootIsRejectedBeforeMutation(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "protected.txt")
	if err := os.WriteFile(marker, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	opts := RunOptions{InputRoots: []string{root}, TempParent: t.TempDir(), PrivateLogDirectory: filepath.Join(root, "nested", "logs")}
	if _, err := Run(context.Background(), testConfig(), testRequest(RoleExecutor), opts); err == nil {
		t.Fatal("protected private-log location was accepted")
	}
	afterInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "preserve" || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatalf("protected input root changed while rejecting private-log path: bytes=%q before=%v after=%v", contents, beforeInfo.ModTime(), afterInfo.ModTime())
	}
	if _, err := os.Lstat(filepath.Join(root, "nested")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private-log validation created a protected directory: %v", err)
	}
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
	t.Setenv(helperEnv, "1")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	config.EnvironmentAllowlist = stringList(helperEnv, "MARKITECT_AGENTEXEC_TEST_MODE", "MARKITECT_AGENTEXEC_TEST_ROLE")
	config.RuntimeFiles = []RuntimeFile{{Path: script, Mode: "0644", Digest: digest(content)}}
	fingerprinted, err := Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
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

func TestEnvironmentAllowlistSemanticsAndDigest(t *testing.T) {
	selected := []string{"MARKITECT_AGENTEXEC_TEST_SECRET"}
	child, envDigest, err := resolveEnvironment(&selected, []string{
		"MARKITECT_AGENTEXEC_TEST_SECRET=secret-value-must-not-appear",
		"UNSELECTED=value",
		"MARKITECT_AGENT_CONFIG_JSON=ambient-config",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(child) != 1 || child[0] != "MARKITECT_AGENTEXEC_TEST_SECRET=secret-value-must-not-appear" {
		t.Fatalf("allowlist did not select exactly its named environment value: %#v", child)
	}
	if strings.Contains(envDigest, "secret-value-must-not-appear") || strings.Contains(envDigest, "sha256:secret") {
		t.Fatal("environment fingerprint exposed a value")
	}
	empty := []string{}
	child, _, err = resolveEnvironment(&empty, []string{"ONLY=value"})
	if err != nil || len(child) != 0 {
		t.Fatalf("non-nil empty allowlist must inherit nothing, got %#v, %v", child, err)
	}
	child, _, err = resolveEnvironment(nil, []string{"LEGACY=value"})
	if err != nil || len(child) != 1 || child[0] != "LEGACY=value" {
		t.Fatalf("nil allowlist must preserve legacy inheritance, got %#v, %v", child, err)
	}
}

func TestLegacyNilAllowlistFingerprintIgnoresAmbientChanges(t *testing.T) {
	config := testConfig()
	if config.EnvironmentAllowlist != nil {
		t.Fatal("fixture must retain the historical nil allowlist")
	}
	const ambientName = "MARKITECT_AGENTEXEC_LEGACY_FINGERPRINT_TEST"
	t.Setenv(ambientName, "prepared-value")
	_, _, _, _, _, preparedFingerprint, _, preparedEnvironmentDigest, err := fingerprintConfig(config, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(ambientName, "invocation-value")
	_, _, _, _, _, invocationFingerprint, _, invocationEnvironmentDigest, err := fingerprintConfig(config, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if invocationFingerprint != preparedFingerprint {
		t.Fatalf("legacy nil-allowlist fingerprint changed with ambient environment: prepared=%s invocation=%s", preparedFingerprint, invocationFingerprint)
	}
	if invocationEnvironmentDigest == preparedEnvironmentDigest {
		t.Fatal("receipt environment digest did not capture the changed inherited environment")
	}
}

func TestEnvironmentAllowlistIsBoundIntoFingerprintAndReceipt(t *testing.T) {
	config := testConfig()
	config.EnvironmentAllowlist = stringList("MARKITECT_AGENTEXEC_TEST_SECRET", helperEnv, "MARKITECT_AGENTEXEC_TEST_MODE", "MARKITECT_AGENTEXEC_TEST_ROLE")
	t.Setenv(helperEnv, "1")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_MODE", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_ROLE", "")
	t.Setenv("MARKITECT_AGENTEXEC_TEST_SECRET", "secret-value-must-not-appear")
	fingerprint, err := Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_AGENTEXEC_TEST_SECRET", "changed-secret")
	changed, err := Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint == changed {
		t.Fatal("changing an allowlisted effective value did not change the fingerprint")
	}
	t.Setenv("MARKITECT_AGENTEXEC_TEST_SECRET", "secret-value-must-not-appear")
	opts := testOptions(t)
	result, err := Run(context.Background(), config, testRequest(RoleExecutor), opts)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.EnvironmentDigest == "" || strings.Contains(string(encoded), "secret-value-must-not-appear") {
		t.Fatalf("receipt must bind the effective environment without exposing its values: %s", encoded)
	}
}

func stringList(values ...string) *[]string { return &values }

func TestRuntimeFileOver256MiBIsFingerprintedWithStreamingHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-runtime.bin")
	const size = (256 << 20) + 1
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(size); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	declaredDigest := zeroFileDigest(size)
	states, _, err := snapshotRuntimeFiles([]RuntimeFile{{Path: path, Mode: "0644", Digest: declaredDigest}})
	if err != nil {
		t.Fatalf("expected runtime asset over 256 MiB to fingerprint successfully: %v", err)
	}
	if len(states) != 1 || states[0].digest != declaredDigest {
		t.Fatalf("large runtime state did not preserve its declared digest: %#v", states)
	}
}

func TestPrivateExecutableRuntimeAssetKeepsExact0700ModePin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not preserve Unix executable permission bits")
	}
	path := filepath.Join(t.TempDir(), "private-tool")
	content := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(path, content, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	file := RuntimeFile{Path: path, Mode: "0700", Digest: digest(content)}
	config := testConfig()
	config.RuntimeFiles = []RuntimeFile{file}
	if _, err := normalizeConfig(config); err != nil {
		t.Fatalf("private executable mode should be accepted as an exact runtime pin: %v", err)
	}
	states, _, err := snapshotRuntimeFiles([]RuntimeFile{file})
	if err != nil {
		t.Fatalf("private executable mode should match its pinned fingerprint: %v", err)
	}
	if len(states) != 1 || states[0].mode != "0700" || states[0].digest != file.Digest {
		t.Fatalf("runtime fingerprint did not retain exact mode and bytes: %#v", states)
	}
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := snapshotRuntimeFiles([]RuntimeFile{file}); err == nil || !strings.Contains(err.Error(), "differs from its declared digest or mode") {
		t.Fatalf("a mode change must invalidate the exact pin, got %v", err)
	}
}

func TestRuntimeFileIndividualAndCombinedBoundsRejectBeforeHashing(t *testing.T) {
	root := t.TempDir()
	createSparseFile := func(name string, size int64) string {
		t.Helper()
		path := filepath.Join(root, name)
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(size); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	oversized := createSparseFile("oversized-runtime.bin", MaxRuntimeAssetBytes+1)
	_, _, err := snapshotRuntimeFiles([]RuntimeFile{{Path: oversized, Mode: "0644", Digest: "sha256:" + strings.Repeat("0", 64)}})
	if err == nil || !strings.Contains(err.Error(), "512 MiB bound") {
		t.Fatalf("expected per-file runtime bound rejection, got %v", err)
	}

	first := createSparseFile("first-runtime.bin", 1)
	second := createSparseFile("second-runtime.bin", MaxRuntimeAssetBytes)
	files := []RuntimeFile{
		{Path: first, Mode: "0644", Digest: digest([]byte{0})},
		{Path: second, Mode: "0644", Digest: "sha256:" + strings.Repeat("0", 64)},
	}
	_, _, err = snapshotRuntimeFiles(files)
	if err == nil || !strings.Contains(err.Error(), "512 MiB bound") {
		t.Fatalf("expected combined runtime bound rejection, got %v", err)
	}
}

func zeroFileDigest(size int64) string {
	hasher := sha256.New()
	zeros := make([]byte, 64<<10)
	for size > 0 {
		chunk := int64(len(zeros))
		if size < chunk {
			chunk = size
		}
		_, _ = hasher.Write(zeros[:chunk])
		size -= chunk
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
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

func TestEmptyRequestCollectionsHaveOneArrayEncoding(t *testing.T) {
	request := testRequest(RoleExecutor)
	request.Artifacts = nil
	request.ScopeIDs = nil
	request.PolicyIDs = nil
	_, absent, err := normalizeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Artifacts = []Artifact{}
	request.ScopeIDs = []string{}
	request.PolicyIDs = []string{}
	_, empty, err := normalizeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(absent, empty) {
		t.Fatal("nil and empty collections changed request identity")
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(absent, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"artifacts", "scopeIds", "policyIds"} {
		if string(decoded[field]) != "[]" {
			t.Fatalf("%s must be an explicit empty array, got %s", field, decoded[field])
		}
	}
}
