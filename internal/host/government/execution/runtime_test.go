package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
)

func TestValidateRuntimeRequiresCompleteDistinctBoundedSlots(t *testing.T) {
	runtime := testRuntime(t)
	if err := ValidateRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	for _, timeout := range []int{0, maxRunTimeout + 1} {
		invalid := runtime
		invalid.TimeoutSeconds = timeout
		if err := ValidateRuntime(invalid); err == nil {
			t.Errorf("run timeout %d should fail", timeout)
		}
	}

	invalid := runtime
	invalid.Verifier.SlotID = invalid.Executor.SlotID
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("same executor/verifier slot should fail, got %v", err)
	}
	invalid = runtime
	invalid.Ressorts = append(invalid.Ressorts, RessortRunner{Ressort: runtime.Ressorts[0].Ressort, Runner: runnerSpec("reviewer-2")})
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "duplicates a Ressort") {
		t.Fatalf("duplicate Ressort should fail, got %v", err)
	}
	invalid = runtime
	invalid.Ressorts = append([]RessortRunner(nil), runtime.Ressorts...)
	invalid.Ressorts[0].Runner.TimeoutSeconds = 601
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "timeoutSeconds") {
		t.Fatalf("actor timeout above 600 seconds should fail, got %v", err)
	}
	invalid = runtime
	invalid.Checks = append([]authoring.Check(nil), runtime.Checks...)
	invalid.Checks[0].TimeoutSeconds = intPointer(authoring.MaxCheckTimeoutSeconds + 1)
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "timeoutSeconds") {
		t.Fatalf("check timeout above 1800 seconds should fail, got %v", err)
	}
	invalid = runtime
	invalid.TemporaryDirectory = filepath.Join(runtime.StateDirectory, "nested")
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("overlapping private directories should fail, got %v", err)
	}
	invalid = runtime
	invalid.ActiveRef = "refs/heads/main"
	if err := ValidateRuntime(invalid); err == nil {
		t.Fatal("ordinary branch ref should not pass the managed active-ref contract")
	}
	for _, ref := range []string{
		"refs/markitect/government/other/active",
		"refs/markitect/government/active/one/two",
		"refs/markitect/government/active/.hidden",
		"refs/markitect/government/active/name.lock",
	} {
		invalid = runtime
		invalid.ActiveRef = ref
		if err := ValidateRuntime(invalid); err == nil {
			t.Errorf("unsafe active ref %q was accepted", ref)
		}
	}
	invalid = runtime
	invalid.ExpectedBase = "main"
	if err := ValidateRuntime(invalid); err == nil {
		t.Fatal("symbolic expected base should fail the full object ID contract")
	}
	invalid = runtime
	invalid.ExpectedBase = strings.Repeat("B", 40)
	if err := ValidateRuntime(invalid); err == nil {
		t.Fatal("uppercase object ID should fail the lowercase contract")
	}
	validSHA256Base := runtime
	validSHA256Base.ExpectedBase = strings.Repeat("b", 64)
	if err := ValidateRuntime(validSHA256Base); err != nil {
		t.Fatalf("full SHA-256 object ID should be accepted: %v", err)
	}
	invalid = runtime
	invalid.Ressorts = make([]RessortRunner, maxRuntimeRessorts+1)
	for i := range invalid.Ressorts {
		invalid.Ressorts[i] = RessortRunner{
			Ressort: core.DefinitionIdentity{APIVersion: "markitect.government/v1alpha1", Kind: "Ressort", Name: fmt.Sprintf("ressort-%03d", i)},
			Runner:  runnerSpec(fmt.Sprintf("reviewer-%03d", i)),
		}
	}
	if err := ValidateRuntime(invalid); err == nil {
		t.Fatal("more than 128 configured Ressorts should fail")
	}
	invalid = runtime
	invalid.Checks = make([]authoring.Check, maxRuntimeChecks+1)
	for i := range invalid.Checks {
		invalid.Checks[i] = authoring.Check{Name: fmt.Sprintf("check-%03d", i), Run: []string{"go"}}
	}
	if err := ValidateRuntime(invalid); err == nil {
		t.Fatal("more than 128 configured checks should fail")
	}
}

func TestValidateRuntimeBoundsAmendmentRepairRounds(t *testing.T) {
	runtime := testRuntime(t)
	for _, repairs := range []int{0, 1, 2} {
		runtime.Amendment = &AmendmentRuntime{MaxRepairs: repairs}
		if err := ValidateRuntime(runtime); err != nil {
			t.Fatalf("amendment repair bound %d should validate: %v", repairs, err)
		}
	}
	runtime.Amendment = &AmendmentRuntime{MaxRepairs: 3}
	if err := ValidateRuntime(runtime); err == nil || !strings.Contains(err.Error(), "amendment.maxRepairs") {
		t.Fatalf("more than two amendment repairs should fail, got %v", err)
	}
	runtime.Amendment = &AmendmentRuntime{MaxRepairs: -1}
	if err := ValidateRuntime(runtime); err == nil || !strings.Contains(err.Error(), "amendment.maxRepairs") {
		t.Fatalf("negative amendment repairs should fail, got %v", err)
	}
}

func TestFingerprintRuntimeBindsAmendmentRepairLimit(t *testing.T) {
	runtime := testRuntime(t)
	base, err := FingerprintRuntime(runtime)
	if err != nil {
		t.Fatal(err)
	}
	runtime.Amendment = &AmendmentRuntime{MaxRepairs: 1}
	amended, err := FingerprintRuntime(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if amended == base {
		t.Fatal("runtime fingerprint did not bind amendment repair configuration")
	}
}

func TestDecodeRuntimeIsClosedAndRejectsDuplicateOrUnknownKeys(t *testing.T) {
	runtime := testRuntime(t)
	encoded, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRuntime(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Executor.SlotID != runtime.Executor.SlotID || decoded.Ressorts[0].Runner.SlotID != runtime.Ressorts[0].Runner.SlotID {
		t.Fatal("decoded runtime lost configured slots")
	}

	for _, input := range []string{
		`{"apiVersion":"x","apiVersion":"y"}`,
		`{"apiVersion":"x","APIVERSION":"y"}`,
		`{"unexpected":true}`,
	} {
		if _, err := DecodeRuntime([]byte(input)); err == nil {
			t.Errorf("DecodeRuntime(%s) unexpectedly succeeded", input)
		}
	}
	runtime.Checks[0].TimeoutSeconds = intPointer(600)
	withTimeout, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	withNullTimeout := strings.Replace(string(withTimeout), `"timeoutSeconds":600`, `"timeoutSeconds":null`, 1)
	if withNullTimeout == string(withTimeout) {
		t.Fatalf("test did not inject explicit null timeout: %s", withTimeout)
	}
	if _, err := DecodeRuntime([]byte(withNullTimeout)); err == nil {
		t.Fatal("explicit null check timeout must be rejected")
	}
	unicodeAlias := strings.Replace(string(withNullTimeout), `"checks"`, `"checKs"`, 1)
	unicodeAlias = strings.Replace(unicodeAlias, `"timeoutSeconds":null`, `"timeoutſecondſ":null`, 1)
	if _, err := DecodeRuntime([]byte(unicodeAlias)); err == nil {
		t.Fatal("Unicode case-fold aliases must not turn a null check timeout into an omitted default")
	}
}

func TestFingerprintRuntimeBindsAllSlotsChecksAndRuntimeFiles(t *testing.T) {
	runtime := testRuntime(t)
	first, err := FingerprintRuntime(runtime)
	if err != nil {
		t.Fatal(err)
	}
	second, err := FingerprintRuntime(runtime)
	if err != nil || second != first {
		t.Fatalf("fingerprint is not stable: first=%q second=%q err=%v", first, second, err)
	}

	reordered := runtime
	reordered.Checks = append([]authoring.Check(nil), runtime.Checks...)
	reordered.Ressorts = append([]RessortRunner(nil), runtime.Ressorts...)
	// Add a second configured Ressort and check, then prove order is not an
	// identity change because both are canonical sets in this runtime contract.
	secondRunner := runnerSpec("reviewer-2")
	reordered.Ressorts = append(reordered.Ressorts, RessortRunner{
		Ressort: core.DefinitionIdentity{APIVersion: "markitect.government/v1alpha1", Kind: "Ressort", Namespace: "", Name: "architecture"}, Runner: secondRunner,
	})
	reordered.Checks = append(reordered.Checks, authoring.Check{Name: "format", Run: []string{filepath.Base(mustLookPath(t, "go"))}})
	base, err := FingerprintRuntime(reordered)
	if err != nil {
		t.Fatal(err)
	}
	reordered.Ressorts[0], reordered.Ressorts[1] = reordered.Ressorts[1], reordered.Ressorts[0]
	reordered.Checks[0], reordered.Checks[1] = reordered.Checks[1], reordered.Checks[0]
	reversed, err := FingerprintRuntime(reordered)
	if err != nil || base != reversed {
		t.Fatalf("set order changed runtime fingerprint: %q != %q (%v)", base, reversed, err)
	}

	changed := runtime
	changed.Executor.Model = "different-model"
	changedFingerprint, err := FingerprintRuntime(changed)
	if err != nil || changedFingerprint == first {
		t.Fatalf("runner model change did not change fingerprint: %q, %v", changedFingerprint, err)
	}
	changed = runtime
	changed.TimeoutSeconds--
	changedFingerprint, err = FingerprintRuntime(changed)
	if err != nil || changedFingerprint == first {
		t.Fatalf("run timeout change did not change fingerprint: %q, %v", changedFingerprint, err)
	}

	declared := t.TempDir()
	runtimeFile := filepath.Join(declared, "wrapper.py")
	if err := os.WriteFile(runtimeFile, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	runtime.Executor.RuntimeFiles = []agentexec.RuntimeFile{{Path: runtimeFile, Mode: "0644", Digest: sha256Digest([]byte("first"))}}
	if _, err := FingerprintRuntime(runtime); err != nil {
		t.Fatalf("valid declared runtime file rejected: %v", err)
	}
	if err := os.WriteFile(runtimeFile, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := FingerprintRuntime(runtime); err == nil {
		t.Fatal("changed declared runtime file retained a valid fingerprint")
	}
}

func TestRunnerPinsAbsoluteFileArgumentsAndInterpreterScripts(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "runner script.py")
	content := []byte("print('runner')\n")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}
	mode := "0644"
	declared := agentexec.RuntimeFile{Path: filePath, Mode: mode, Digest: sha256Digest(content)}

	python := runnerSpec("python")
	python.Command = "python3.11"
	python.Args = []string{filePath}
	if err := validateRunner(python); err == nil {
		t.Fatal("interpreter script without a runtime-file pin should fail")
	}
	python.RuntimeFiles = []agentexec.RuntimeFile{declared}
	if err := validateRunner(python); err != nil {
		t.Fatalf("declared interpreter script should pass: %v", err)
	}

	inline := runnerSpec("python-inline")
	inline.Command = "python"
	inline.Args = []string{"-c", "print('inline')"}
	if err := validateRunner(inline); err != nil {
		t.Fatalf("inline code should be pinned by argv: %v", err)
	}
	for _, sample := range []struct{ command, flag string }{
		{"ruby", "-e"}, {"perl", "-e"}, {"node", "-e"}, {"sh", "-c"}, {"pwsh", "-Command"},
	} {
		inline.Command = sample.command
		inline.Args = []string{sample.flag, "inline code"}
		if err := validateRunner(inline); err != nil {
			t.Errorf("%s inline mode rejected: %v", sample.command, err)
		}
	}
	inline.Command = "python"
	inline.Args = []string{"-c", ""}
	if err := validateRunner(inline); err == nil {
		t.Fatal("empty inline code should fail")
	}
	inline.Args = []string{"reviewer.py", "-c", "print('late flag')"}
	if err := validateRunner(inline); err == nil {
		t.Fatal("inline flag after an unpinned script must not bypass the script pin")
	}

	absoluteArgument := runnerSpec("absolute-arg")
	absoluteArgument.Args = []string{"--config=" + filePath}
	if err := validateRunner(absoluteArgument); err == nil {
		t.Fatal("absolute file-valued flag argument without runtime pin should fail")
	}
	absoluteArgument.RuntimeFiles = []agentexec.RuntimeFile{declared}
	if err := validateRunner(absoluteArgument); err != nil {
		t.Fatalf("declared absolute file-valued flag should pass: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateRunner(absoluteArgument); err == nil {
		t.Fatal("changed absolute file should fail its declared digest")
	}
}

func TestInvokeUsesFreshAgentexecProcessAndReturnsReceipt(t *testing.T) {
	workspace := t.TempDir()
	state := t.TempDir()
	temporary := t.TempDir()
	request := testRequest(agentexec.RoleExecutor)
	first, err := Invoke(context.Background(), runnerSpec("writer"), request, workspace, state, temporary)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Invoke(context.Background(), runnerSpec("writer"), request, workspace, state, temporary)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.RunID == "" || second.Receipt.RunID == "" || first.Receipt.RunID == second.Receipt.RunID {
		t.Fatalf("invocations did not receive distinct real run receipts: first=%+v second=%+v", first.Receipt, second.Receipt)
	}
	if first.Receipt.Outcome != agentexec.OutcomeProposed || len(first.Response.CandidateFiles) != 1 {
		t.Fatalf("expected actual executor candidate response, got response=%+v receipt=%+v", first.Response, first.Receipt)
	}
	if info, err := os.Stat(state); err != nil || !info.IsDir() {
		t.Fatalf("expected configured private state directory: info=%v err=%v", info, err)
	}
}

func TestInvokeRejectsInferenceAndVerifierCannotReturnCandidate(t *testing.T) {
	workspace := t.TempDir()
	state := t.TempDir()
	temporary := t.TempDir()
	request := testRequest(agentexec.RoleInfer)
	if _, err := Invoke(context.Background(), runnerSpec("writer"), request, workspace, state, temporary); err == nil {
		t.Fatal("inference role must not be accepted by Government invocation")
	}
	request = testRequest(agentexec.RoleVerifier)
	request.Context = json.RawMessage(`{"testMode":"verifier-candidate"}`)
	if _, err := Invoke(context.Background(), runnerSpec("verifier"), request, workspace, state, temporary); err == nil {
		t.Fatal("agentexec should reject candidate files from a verifier response")
	}
}

// TestGovernmentRuntimeRunnerHelper is launched as the configured external
// process by the tests above. It echoes the real invocation envelope and emits
// one controlled response to exercise agentexec's process and receipt boundary.
func TestGovernmentRuntimeRunnerHelper(t *testing.T) {
	if os.Getenv("MARKITECT_AGENT_PRIVATE_LOG") == "" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		t.Fatal(err)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, CandidateFiles: []agentexec.CandidateFile{{Path: "src/value.txt", Mode: "0644", Content: "actual candidate bytes"}},
		EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{},
	}
	if invocation.Request.Role == agentexec.RoleVerifier {
		response.CandidateFiles = []agentexec.CandidateFile{}
		response.Outcome = agentexec.OutcomePassed
		response.VerifierObservations = []agentexec.Observation{{Subject: "src/value.txt", Outcome: agentexec.OutcomePassed, Detail: "controlled verifier result"}}
		if string(invocation.Request.Context) == `{"testMode":"verifier-candidate"}` {
			response.CandidateFiles = []agentexec.CandidateFile{{Path: "src/value.txt", Mode: "0644", Content: "forbidden"}}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		t.Fatal(err)
	}
	// Avoid the go test harness's trailing PASS text contaminating the wire
	// response. This test is only selected by the child command in the tests.
	os.Exit(0)
}

func testRuntime(t *testing.T) Runtime {
	t.Helper()
	executable := mustLookPath(t, "go")
	check := authoring.Check{Name: "tests", Run: []string{filepath.Base(executable)}}
	state := filepath.Join(t.TempDir(), "private-state")
	temporary := filepath.Join(t.TempDir(), "private-temp")
	return Runtime{
		APIVersion: RuntimeVersion, ActiveRef: "refs/markitect/government/active/primary", ExpectedBase: strings.Repeat("a", 40), TimeoutSeconds: 1800,
		StateDirectory: state, TemporaryDirectory: temporary,
		Executor: runnerSpec("writer"), Verifier: runnerSpec("verifier"),
		Ressorts: []RessortRunner{{
			Ressort: core.DefinitionIdentity{APIVersion: "markitect.government/v1alpha1", Kind: "Ressort", Namespace: "", Name: "security"},
			Runner:  runnerSpec("security-reviewer"),
		}}, Checks: []authoring.Check{check},
	}
}

func runnerSpec(slot string) RunnerSpec {
	executable, _ := os.Executable()
	return RunnerSpec{
		SlotID: slot, Command: executable, Args: []string{"-test.run=^TestGovernmentRuntimeRunnerHelper$"},
		Model: "controlled-test-model", ModelOptions: json.RawMessage(`{"effort":"low"}`), ProviderVersion: "test-runner-v1",
		TimeoutSeconds: 20, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
	}
}

func testRequest(role string) agentexec.Request {
	return agentexec.Request{
		Role: role, SourceRevision: strings.Repeat("b", 40), ModelDigest: sha256Digest([]byte("model")),
		ModulePin: "government-test/v1", ProjectionID: "area:root", ScopeIDs: []string{"src/value.txt"}, PolicyIDs: []string{},
		Context: json.RawMessage(`{"task":"test"}`), Artifacts: []agentexec.Artifact{},
	}
}

func mustLookPath(t *testing.T, command string) string {
	t.Helper()
	path, err := exec.LookPath(command)
	if err != nil {
		t.Skipf("required executable %q is unavailable", command)
	}
	return path
}

func sha256Digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func intPointer(value int) *int { return &value }
