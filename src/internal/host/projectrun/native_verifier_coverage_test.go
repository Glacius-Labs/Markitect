package projectrun

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

const nativeVerifierFixtureEnv = "MARKITECT_PROJECTRUN_NATIVE_VERIFIER_FIXTURE"

// TestMain exposes this test binary as a protocol-only App Server executable
// for the integration below. No Codex CLI, account, or model is started.
func TestMain(m *testing.M) {
	if os.Getenv(nativeVerifierFixtureEnv) == "1" {
		if len(os.Args) > 1 && os.Args[1] == "--version" {
			fmt.Println(codexappserver.SupportedProviderVersion)
			os.Exit(0)
		}
		serveNativeVerifierFixture()
		os.Exit(0)
	}
	testkit.Main(m)
}

func serveNativeVerifierFixture() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 8<<20)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var message struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			return
		}
		var result any = map[string]any{}
		switch message.Method {
		case "initialize":
			result = map[string]string{"userAgent": "codex/0.162.0"}
		case "initialized":
			continue
		case "thread/start":
			cwd, err := os.Getwd()
			if err != nil {
				return
			}
			result = map[string]any{
				"thread": map[string]any{"id": "thread-1", "sessionId": "session-1", "cliVersion": "0.162.0", "cwd": cwd},
				"model":  "gpt-6-luna", "reasoningEffort": "high", "cwd": cwd,
				"approvalPolicy": "on-request", "sandbox": map[string]any{"type": "readOnly"},
				"instructionSources": []string{},
			}
		case "turn/start":
			var semantic json.RawMessage
			if json.Unmarshal([]byte(os.Getenv("MARKITECT_PROJECTRUN_NATIVE_VERIFIER_RESPONSE")), &semantic) != nil || !json.Valid(semantic) {
				return
			}
			if encoder.Encode(map[string]any{"method": "turn/started", "params": map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "inProgress"}}}) != nil {
				return
			}
			if encoder.Encode(map[string]any{"id": message.ID, "result": map[string]any{"turn": map[string]any{"id": "turn-1", "status": "inProgress"}}}) != nil {
				return
			}
			if encoder.Encode(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "item": map[string]any{"id": "final-1", "type": "agentMessage", "phase": "final_answer", "text": string(semantic)}}}) != nil {
				return
			}
			if encoder.Encode(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}}}) != nil {
				return
			}
			continue
		default:
			return
		}
		if len(message.ID) > 0 && encoder.Encode(map[string]any{"id": message.ID, "result": result}) != nil {
			return
		}
	}
}

func TestNativeVerifierAliasesReachExactProjectCoverageBoundary(t *testing.T) {
	refs := []string{"artifact:spec", "check:unit"}
	allRequestRefs := append([]string(nil), refs...)
	allRequestRefs = append(allRequestRefs, "unit") // PolicyID is allowed request evidence, not required coverage.
	sort.Strings(allRequestRefs)
	aliases := map[string]string{}
	for index, ref := range allRequestRefs {
		aliases[fmt.Sprintf("evidence-000000-%06d", index)] = ref
	}
	aliasFor := func(ref string) string {
		for alias, canonical := range aliases {
			if canonical == ref {
				return alias
			}
		}
		t.Fatalf("fixture has no alias for %q", ref)
		return ""
	}
	selected := []string{aliasFor(refs[0]), aliasFor(refs[1])}
	sort.Strings(selected)

	// Native verifier responses use aliases on the wire; the adapter maps them
	// back to these canonical subjects before project coverage is evaluated.
	requiredSubjects := []string{"spec", "unit"}
	sortedSubjects := append([]string(nil), requiredSubjects...)
	sort.Strings(sortedSubjects)
	subjectAliases := make(map[string]string, len(sortedSubjects))
	for index, subject := range sortedSubjects {
		subjectAliases[subject] = fmt.Sprintf("verifier-subject-000000-%06d", index)
	}
	observations := []agentexec.Observation{
		{Subject: subjectAliases["spec"], Outcome: "passed", Detail: "Required artifact inspected."},
		{Subject: subjectAliases["unit"], Outcome: "passed", Detail: "Required check inspected."},
	}
	makeResponse := func(evidence []string) string {
		wire, err := json.Marshal(map[string]any{"outcome": "passed", "candidateFiles": []any{}, "verifierObservations": observations, "uncertainty": []any{}, "evidenceRefs": evidence})
		if err != nil {
			t.Fatal(err)
		}
		return string(wire)
	}

	tests := []struct {
		name      string
		evidence  []string
		wantStage string
		wantError string
	}{
		{name: "complete expected coverage", evidence: selected},
		{name: "unknown alias rejected by adapter", evidence: []string{"unknown-alias"}, wantStage: "adapter", wantError: "native verifier evidence references contain an unknown alias"},
		{name: "duplicate alias rejected by adapter", evidence: []string{selected[0], selected[0]}, wantStage: "adapter", wantError: "native verifier evidence references contain a duplicate alias"},
		{name: "omitted expected reference rejected by coverage", evidence: selected[:1], wantStage: "coverage"},
		{name: "extra valid policy alias rejected by coverage", evidence: append(append([]string(nil), selected...), aliasFor("unit")), wantStage: "coverage"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(nativeVerifierFixtureEnv, "1")
			t.Setenv("MARKITECT_PROJECTRUN_NATIVE_VERIFIER_RESPONSE", makeResponse(tc.evidence))
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cwd := t.TempDir()
			sourceRevision := strings.Repeat("a", 40)
			config := codexappserver.Config{Command: executable, ProviderVersion: codexappserver.SupportedProviderVersion,
				Model: "gpt-6-luna", ReasoningEffort: "high", Timeout: 10 * time.Second, MaxEventBytes: 1 << 20,
				Helpers: codexappserver.HelperPolicy{Enabled: false}}
			adapter, err := codexappserver.NewAdapter(config, codexappserver.Options{})
			if err != nil {
				t.Fatal(err)
			}
			shared := agentexec.Config{Command: executable, ProviderVersion: config.ProviderVersion, Model: config.Model,
				Timeout: config.Timeout, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 16}
			contextJSON, err := json.Marshal(map[string]any{
				"kind": "project-verify/v1", "requiredSubjects": requiredSubjects,
				"requiredEvidenceRefs": refs,
			})
			if err != nil {
				t.Fatal(err)
			}
			request := agentexec.Request{Role: agentexec.RoleVerifier, SourceRevision: sourceRevision,
				ModelDigest: "sha256:" + strings.Repeat("b", 64), ModulePin: "module@1", ProjectionID: "project",
				ScopeIDs: refs, PolicyIDs: []string{"unit"}, Context: contextJSON, Artifacts: []agentexec.Artifact{}}
			workspace := &projectworkspace.Handle{ID: "verify-test", CWD: filepath.Clean(cwd), BaseSHA: sourceRevision}
			result, err := adapter.Run(context.Background(), shared, request, agentexec.RunOptions{Workspace: workspace})
			if tc.wantStage == "adapter" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("adapter error = %v, want error containing %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("native adapter rejected selected aliases before coverage: %v", err)
			}
			if result.Response.Role != agentexec.RoleVerifier || result.Response.Outcome != agentexec.OutcomePassed {
				t.Fatalf("native response was not bound as the passed verifier result: %#v", result.Response)
			}
			coverageErr := validateVerifierCoverage(result.Response.VerifierObservations, []string{"spec", "unit"}, refs, result.Response.EvidenceRefs)
			if tc.wantStage == "coverage" && coverageErr == nil {
				t.Fatal("incomplete or overbroad model-selected aliases passed exact verifier coverage")
			}
			if tc.wantStage == "" && coverageErr != nil {
				t.Fatalf("complete expected alias selection failed project verifier coverage: %v", coverageErr)
			}
		})
	}
}
