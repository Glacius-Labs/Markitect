package host

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	canonicalControllerActorEnv  = "MARKITECT_CONTROLLER_TEST_ACTOR"
	canonicalControllerMarkerEnv = "MARKITECT_CONTROLLER_TEST_MARKER"
)

// This child process speaks the agent protocol for lifecycle tests. It does not
// inspect semantics or supply verification evidence; the candidate is merely
// syntactically valid, request-bound test input.
func TestCanonicalControllerFakeActor(t *testing.T) {
	if os.Getenv(canonicalControllerActorEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		os.Exit(31)
	}
	marker := os.Getenv(canonicalControllerMarkerEnv)
	if marker != "" {
		if err := os.WriteFile(marker, []byte("invoked\n"), 0600); err != nil {
			os.Exit(32)
		}
	}
	var contextEnvelope struct {
		Model CanonicalAgentContext `json:"model"`
	}
	if err := json.Unmarshal(invocation.Request.Context, &contextEnvelope); err != nil || contextEnvelope.Model.RequestDigest == "" {
		os.Exit(33)
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{},
		Uncertainty: []string{}, CandidateFiles: []agentexec.CandidateFile{
			{Path: "src/ControllerTest.cs", Mode: "0644", Content: "namespace ControllerLifecycle; public sealed class ControllerTest {}\n"},
			{Path: "src/ControllerTest.csproj", Mode: "0644", Content: "<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup></Project>\n"},
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(34)
	}
	os.Exit(0)
}

func canonicalControllerFixture(t *testing.T) (string, string, CanonicalControllerConfig) {
	t.Helper()
	root, revision, _ := scopedCanonicalFixture(t)
	tempRoot := os.TempDir()
	if runtime.GOOS == "windows" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("resolve user home for external controller state: %v", err)
		}
		// Avoid the shared Windows Temp directory, whose concurrent test usage
		// can prevent canonical parent checks from opening the directory.
		tempRoot = filepath.Join(home, "AppData", "Local")
	}
	tempRoot, err := filepath.EvalSymlinks(tempRoot)
	if err != nil {
		t.Fatalf("resolve external controller parent: %v", err)
	}
	tempRoot, err = realDirectory(tempRoot)
	if err != nil {
		t.Fatalf("validate external controller parent: %v", err)
	}
	external, err := os.MkdirTemp(tempRoot, "markitect-controller-")
	if err != nil {
		t.Fatalf("create canonical external controller directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	cfg := CanonicalControllerConfig{
		APIVersion:     CanonicalControllerAPIVersion,
		RecordStore:    filepath.Join(external, "ledger"),
		PrivateLogs:    filepath.Join(external, "private-logs"),
		ReferenceDepth: 1,
		Executor: CanonicalRunnerConfig{
			Command: os.Args[0], Args: []string{"-test.run=^TestCanonicalControllerFakeActor$"},
			Model: "lifecycle-test-double", ModelOptions: json.RawMessage(`{"purpose":"protocol test only"}`),
			ProviderVersion: "fake-actor/1", TimeoutSeconds: 10, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
		},
		Verifier: CanonicalRunnerConfig{
			Command: os.Args[0], Args: []string{"-test.run=^TestCanonicalControllerFakeActor$"},
			Model: "lifecycle-test-double", ModelOptions: json.RawMessage(`{"purpose":"protocol test only"}`),
			ProviderVersion: "fake-actor/1", TimeoutSeconds: 10, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
		},
	}
	return root, revision, cfg
}

func canonicalControllerExecute(t *testing.T, root, revision string, cfg CanonicalControllerConfig) CanonicalReviewedRun {
	t.Helper()
	marker := filepath.Join(filepath.Dir(cfg.RecordStore), "actor-invoked")
	t.Setenv(canonicalControllerActorEnv, "1")
	t.Setenv(canonicalControllerMarkerEnv, marker)
	run, err := ExecuteCanonicalController(context.Background(), root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg, "test-tool/1", "sha256:"+strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("execute controller: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("protocol actor did not run: %v", err)
	}
	return run
}

func canonicalControllerApply(t *testing.T, root string, cfg CanonicalControllerConfig, run CanonicalReviewedRun) (CanonicalControllerApply, error) {
	t.Helper()
	return ApplyCanonicalController(root, "examples/canonical-projection/canonical.yaml", cfg, run, run.Digest, true)
}

func canonicalControllerForbidden(t *testing.T, root string) []string {
	t.Helper()
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	return canonicalControllerForbiddenRoots(identity)
}

func canonicalControllerSourceState(t *testing.T, root string) (head, index string, files map[string][]byte) {
	t.Helper()
	head = scopedTestGit(t, root, "rev-parse", "HEAD")
	index = scopedTestGit(t, root, "diff", "--cached", "--binary")
	files = map[string][]byte{}
	for _, path := range []string{
		"examples/canonical-projection/canonical.yaml",
		"examples/canonical-projection/definitions/commerce.projection.yaml",
		"examples/canonical-projection/definitions/commerce.markdown-projection.yaml",
		"src/ControllerTest.cs", "src/ControllerTest.csproj",
		"docs/represented/README.md",
	} {
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path))); err == nil {
			files[path] = data
		}
	}
	return head, index, files
}

func TestCanonicalControllerLifecycleIsReadOnlyUntilExactReviewedApply(t *testing.T) {
	root, revision, cfg := canonicalControllerFixture(t)
	beforeHead, beforeIndex, beforeFiles := canonicalControllerSourceState(t, root)
	run := canonicalControllerExecute(t, root, revision, cfg)
	if run.Status != "planned" || len(run.Work) != 2 {
		t.Fatalf("execute status/work = %q/%d; want planned and exact two-scope cohort", run.Status, len(run.Work))
	}
	if _, err := os.Stat(cfg.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("read-only execute created a ledger: %v", err)
	}
	if head, index, files := canonicalControllerSourceState(t, root); head != beforeHead || index != beforeIndex || !equalCanonicalValue(files, beforeFiles) {
		t.Fatal("read-only execute changed source HEAD, index, or adopter files")
	}
	if _, err := ApplyCanonicalController(root, "examples/canonical-projection/canonical.yaml", cfg, run, "wrong-digest", true); err == nil {
		t.Fatal("Apply accepted a digest other than the reviewed run digest")
	}
	if _, err := ApplyCanonicalController(root, "examples/canonical-projection/canonical.yaml", cfg, run, run.Digest, false); err == nil {
		t.Fatal("Apply accepted a call without explicit write intent")
	}
	if head, index, files := canonicalControllerSourceState(t, root); head != beforeHead || index != beforeIndex || !equalCanonicalValue(files, beforeFiles) {
		t.Fatal("refused Apply changed source state")
	}
	applied, err := canonicalControllerApply(t, root, cfg, run)
	if err != nil {
		t.Fatalf("reviewed Apply: %v", err)
	}
	if applied.Status != records.StateMaterializedUnverified || len(applied.Written) == 0 {
		t.Fatalf("Apply result = %q with %v; want materialized-unverified writes", applied.Status, applied.Written)
	}
	if head, index, _ := canonicalControllerSourceState(t, root); head != beforeHead || index != beforeIndex {
		t.Fatal("Apply changed source HEAD or index")
	}
	store, err := recordstore.Open(cfg.RecordStore, canonicalControllerForbidden(t, root))
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ActiveSelection.RecordIDs) != len(run.Work) || len(state.Records) != len(run.Work) {
		t.Fatalf("ledger records/active selection = %d/%d; want %d each", len(state.Records), len(state.ActiveSelection.RecordIDs), len(run.Work))
	}
	for _, record := range state.Records {
		if record.State != records.StateMaterializedUnverified {
			t.Fatalf("controller record falsely claims verification: %#v", record)
		}
	}
	if len(state.Verifications) != 0 {
		t.Fatalf("Apply wrote verification results without a verifier: %d", len(state.Verifications))
	}
	if applied.EvidenceRevision == "" {
		t.Fatal("successful Apply omitted immutable evidence revision")
	}
	if got := scopedTestGit(t, root, "cat-file", "-t", applied.EvidenceRevision); got != "commit" {
		t.Fatalf("evidence revision is not a readable Git commit: %q", got)
	}
	if got := scopedTestGit(t, root, "show", applied.EvidenceRevision+":examples/canonical-projection/canonical.yaml"); !strings.Contains(got, "kind: Source") {
		t.Fatal("immutable evidence commit does not contain canonical source")
	}
	// A second proposal observes active ownership but cannot turn it into PASS.
	proposal, err := ProposeCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Status == "passed" || proposal.Status == "verified" {
		t.Fatalf("active ownership was misreported as verification: %q", proposal.Status)
	}
}

func TestCanonicalControllerStaleInputsRefuseBeforeAdopterWrites(t *testing.T) {
	for _, stale := range []string{"canonical", "target", "runtime", "ledger", "intent"} {
		t.Run(stale, func(t *testing.T) {
			root, revision, cfg := canonicalControllerFixture(t)
			run := canonicalControllerExecute(t, root, revision, cfg)
			switch stale {
			case "canonical":
				scopedTestWrite(t, root, "examples/canonical-projection/definitions/commerce.projection.yaml", "changed canonical bytes\n")
			case "target":
				scopedTestWrite(t, root, "src/existing.cs", "target appeared after review\n")
			case "runtime":
				cfg.Executor.ModelOptions = json.RawMessage(`{"purpose":"changed after review"}`)
			case "ledger":
				store, err := recordstore.Initialize(cfg.RecordStore, canonicalControllerForbidden(t, root))
				if err != nil {
					t.Fatal(err)
				}
				state, err := store.Read()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := store.SelectActive(state.Head, []string{}); err != nil {
					t.Fatal(err)
				}
			case "intent":
				scopedTestWrite(t, root, "examples/canonical-projection/definitions/commerce.projection.yaml", "apiVersion: markitect.foundation/v1\nkind: Projection\nmetadata:\n  namespace: commerce\n  name: changed-intent\nspec: {}\n")
			}
			_, _, before := canonicalControllerSourceState(t, root)
			if _, err := canonicalControllerApply(t, root, cfg, run); err == nil {
				t.Fatalf("Apply accepted stale %s input", stale)
			}
			_, _, after := canonicalControllerSourceState(t, root)
			for path, data := range before {
				if !bytes.Equal(after[path], data) {
					t.Fatalf("stale %s refusal overwrote %s", stale, path)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "src", "ControllerTest.cs")); !os.IsNotExist(err) {
				t.Fatalf("stale %s refusal created candidate artifact: %v", stale, err)
			}
		})
	}
}

func TestCanonicalControllerAssuranceAndSourceBoundsRejectBeforeActor(t *testing.T) {
	for _, invalid := range []string{"cycle", "source-escape"} {
		t.Run(invalid, func(t *testing.T) {
			root, revision, cfg := canonicalControllerFixture(t)
			marker := filepath.Join(filepath.Dir(cfg.RecordStore), "should-not-run")
			t.Setenv(canonicalControllerActorEnv, "1")
			t.Setenv(canonicalControllerMarkerEnv, marker)
			switch invalid {
			case "cycle":
				check := []authoring.Check{{Name: "bounded", Run: []string{"go", "version"}}}
				cfg.AssuranceRoots = []string{"root"}
				cfg.AssuranceScopes = []CanonicalAssuranceScope{
					{ID: "root", ProjectionID: "commerce/Projection/application-dotnet", Children: []string{"child"}, Checks: check},
					{ID: "child", ProjectionID: "commerce/Projection/application-markdown", Children: []string{"root"}, Checks: check},
				}
			case "source-escape":
				cfg.CheckInputs = []string{"../outside.txt"}
			}
			if _, err := ExecuteCanonicalController(context.Background(), root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg, "test-tool/1", "sha256:"+strings.Repeat("a", 64)); err == nil {
				t.Fatalf("invalid %s configuration was accepted", invalid)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("actor ran before %s validation: %v", invalid, err)
			}
		})
	}
}

func TestCanonicalControllerRejectsDuplicateOrOmittedWorkWithoutWrites(t *testing.T) {
	for _, mutation := range []string{"duplicate", "omitted", "extra"} {
		t.Run(mutation, func(t *testing.T) {
			root, revision, cfg := canonicalControllerFixture(t)
			run := canonicalControllerExecute(t, root, revision, cfg)
			switch mutation {
			case "duplicate":
				run.Work = append(run.Work, run.Work[0])
			case "omitted":
				run.Work = run.Work[:len(run.Work)-1]
			case "extra":
				run.Work = append(run.Work, CanonicalControllerWork{ProjectionID: "commerce/Projection/unknown"})
			}
			var err error
			run, err = finalizeCanonicalReviewedRun(run)
			if err != nil {
				t.Fatal(err)
			}
			beforeHead, beforeIndex, beforeFiles := canonicalControllerSourceState(t, root)
			if _, err := canonicalControllerApply(t, root, cfg, run); err == nil {
				t.Fatalf("Apply accepted %s work cohort", mutation)
			}
			afterHead, afterIndex, afterFiles := canonicalControllerSourceState(t, root)
			if afterHead != beforeHead || afterIndex != beforeIndex || !equalCanonicalValue(afterFiles, beforeFiles) {
				t.Fatalf("rejected %s cohort changed source state", mutation)
			}
			if _, err := os.Stat(cfg.RecordStore); !os.IsNotExist(err) {
				t.Fatalf("rejected %s cohort created ledger: %v", mutation, err)
			}
		})
	}
}
