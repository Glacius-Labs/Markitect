package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func TestSelectedHostProjectorUsesBoundEntrypointAndTarget(t *testing.T) {
	markdown := canonical.ProjectionRequest{Binding: canonical.ProjectionBinding{Module: "custom-docs"}, ModulePin: canonical.Pin{Name: "custom-docs", Version: "9.2.0"}, Projector: canonical.ProjectorRegistration{ID: "markdown-documentation", Version: "1.0.0", Target: "markdown"}}
	candidate, err := selectedHostProjector(markdown)
	if err != nil || candidate {
		t.Fatalf("Markdown entrypoint dispatch failed: candidate=%v err=%v", candidate, err)
	}
	dotnet := canonical.ProjectionRequest{Binding: canonical.ProjectionBinding{Module: "custom-dotnet"}, ModulePin: canonical.Pin{Name: "custom-dotnet", Version: "2.3.0"}, Projector: canonical.ProjectorRegistration{ID: "dotnet-source", Version: "1.0.0", Target: "dotnet"}}
	candidate, err = selectedHostProjector(dotnet)
	if err != nil || !candidate {
		t.Fatalf(".NET entrypoint dispatch failed: candidate=%v err=%v", candidate, err)
	}
}

func TestSelectedHostProjectorRejectsMismatchedStaticEntrypoints(t *testing.T) {
	cases := []canonical.ProjectorRegistration{
		{ID: "markdown-documentation", Version: "1.0.0", Target: "dotnet"},
		{ID: "dotnet-source", Version: "1.0.0", Target: "markdown"},
		{ID: "dotnet-source", Version: "2.0.0", Target: "dotnet"},
		{ID: "unknown", Version: "1.0.0", Target: "dotnet"},
	}
	for _, projector := range cases {
		if _, err := selectedHostProjector(canonical.ProjectionRequest{Projector: projector}); err == nil {
			t.Fatalf("unsupported Projector registration was accepted: %+v", projector)
		}
	}
}

func TestDecodeCanonicalCandidateRejectsAmbiguousOrUnsafeFiles(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{name: "duplicate paths", data: `{"requestDigest":"sha256:x","files":[{"path":"src/a.cs","content":"a"},{"path":"src/a.cs","content":"b"}]}`},
		{name: "empty files", data: `{"requestDigest":"sha256:x","files":[]}`},
		{name: "unknown fields", data: `{"requestDigest":"sha256:x","files":[{"path":"src/a.cs","content":"a","unexpected":true}]}`},
		{name: "trailing value", data: `{"requestDigest":"sha256:x","files":[{"path":"src/a.cs","content":"a"}]} {}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeCanonicalCandidate([]byte(tc.data)); err == nil {
				t.Fatal("ambiguous or invalid candidate was accepted")
			}
		})
	}

	candidate := CanonicalCandidate{RequestDigest: "sha256:x", Files: []CanonicalCandidateFile{{Path: "../outside.cs", Content: "x"}}}
	if _, _, err := candidateMap(candidate, "src/"); err == nil {
		t.Fatal("candidate outside exact target prefix was accepted")
	}
}

func TestApplyCanonicalProjectionRequiresExactReviewAndWrite(t *testing.T) {
	if _, err := ApplyCanonicalProjection(".", nil, nil, PreparedCanonicalProjection{CandidateDigest: "sha256:approved"}, "sha256:approved", false); err == nil || !strings.Contains(err.Error(), "write=true") {
		t.Fatalf("apply without explicit write authorization was accepted: %v", err)
	}
	if _, err := ApplyCanonicalProjection(".", nil, nil, PreparedCanonicalProjection{CandidateDigest: "sha256:approved"}, "sha256:other", true); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("apply with a different reviewed digest was accepted: %v", err)
	}
}

func TestDecodeCanonicalCandidateRejectsDuplicateJSONMembers(t *testing.T) {
	cases := []string{
		`{"requestDigest":"sha256:first","requestDigest":"sha256:last","files":[{"path":"src/a.cs","content":"a"}]}`,
		`{"requestDigest":"sha256:x","files":[{"path":"src/a.cs","path":"src/b.cs","content":"a"}]}`,
	}
	for _, input := range cases {
		if _, err := decodeCanonicalCandidate([]byte(input)); err == nil {
			t.Fatalf("candidate with a duplicate JSON member was accepted: %s", input)
		}
	}
}

func TestProjectionPlanConfigDigestBindsExactCheckCommands(t *testing.T) {
	first := []authoring.Check{{Name: "build", Run: []string{"go", "test", "./..."}}}
	changed := []authoring.Check{{Name: "build", Run: []string{"go", "test", "./internal/host"}}}
	if projectionConfigDigest("sha256:request", first) == projectionConfigDigest("sha256:request", changed) {
		t.Fatal("plan config digest did not bind the exact verification command")
	}
}

func TestCanonicalProjectionTargetFilesFiltersObservedSnapshot(t *testing.T) {
	projection := core.Definition{Spec: map[string]any{"target": map[string]any{"path": "src/"}}}
	observed := map[string][]byte{"src/Orders.cs": []byte("selected"), "src-old/Orders.cs": []byte("sibling"), "docs/readme.md": []byte("outside")}
	files, err := canonicalProjectionTargetFiles(projection, observed)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || string(files["src/Orders.cs"]) != "selected" {
		t.Fatalf("target filter returned unrelated snapshot files: %#v", files)
	}
	if _, _, err := candidateMap(CanonicalCandidate{Files: []CanonicalCandidateFile{{Path: "src-old/Orders.cs", Content: "outside"}}}, "src"); err == nil {
		t.Fatal("candidate with a sibling-prefix path was accepted")
	}
}

func TestCandidateArtifactModesDefaultAndValidate(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"0644", snapshot.RegularMode}, {"0755", snapshot.ExecutableMode}} {
		got, err := canonicalCandidateMode(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("agent candidate mode %s: got %s, err %v", tc.input, got, err)
		}
	}
	if _, err := canonicalCandidateMode("0600"); err == nil {
		t.Fatal("private agent candidate mode was treated as a Git artifact mode")
	}
	files, modes, err := candidateMap(CanonicalCandidate{Files: []CanonicalCandidateFile{{Path: "hooks/pre-commit", Content: "#!/bin/sh\n", Mode: snapshot.ExecutableMode}}}, "hooks")
	if err != nil || string(files["hooks/pre-commit"]) != "#!/bin/sh\n" || modes["hooks/pre-commit"] != snapshot.ExecutableMode {
		t.Fatalf("executable candidate mode lost: files=%v modes=%v err=%v", files, modes, err)
	}
	_, modes, err = candidateMap(CanonicalCandidate{Files: []CanonicalCandidateFile{{Path: "docs/readme.md", Content: "text"}}}, "")
	if err != nil || modes["docs/readme.md"] != snapshot.RegularMode {
		t.Fatalf("omitted mode did not default to regular: modes=%v err=%v", modes, err)
	}
	if _, _, err := candidateMap(CanonicalCandidate{Files: []CanonicalCandidateFile{{Path: "x", Content: "x", Mode: "100664"}}}, ""); err == nil {
		t.Fatal("unsupported candidate mode was accepted")
	}
}

func TestCanonicalProjectionPostWriteReadbackSelectsWrittenArtifacts(t *testing.T) {
	root, revision := canonicalWorkflowFixtureRepository(t)
	fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-workflow/canonical.yaml", true)
	if err != nil {
		t.Fatalf("load fixed canonical source: %v", err)
	}
	observed, err := source.Load(root, "")
	if err != nil {
		t.Fatalf("load initial working snapshot: %v", err)
	}
	identity := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "example", Name: "workflow-markdown"}
	toolDigest := sha256Prefix(sha256Hex([]byte("selected post-write readback test")))
	prepared, err := PrepareCanonicalProjection(fixed, observed, identity, "readback-test/1", toolDigest, nil, fixed.Config.Checks...)
	if err != nil || prepared.Plan == nil || len(prepared.Outputs) != 1 {
		t.Fatalf("prepare deterministic Projection: outputs=%d err=%v", len(prepared.Outputs), err)
	}
	written := make([]string, 0, len(prepared.Outputs))
	for name, content := range prepared.Outputs {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(name))), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), content, 0644); err != nil {
			t.Fatal(err)
		}
		written = append(written, name)
	}
	large := filepath.Join(root, "docs", "represented", "unrelated-large.bin")
	if err := os.MkdirAll(filepath.Dir(large), 0755); err != nil {
		t.Fatal(err)
	}
	largeFile, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := largeFile.Truncate(source.DefaultMaxFileBytes + 1); err != nil {
		_ = largeFile.Close()
		t.Fatal(err)
	}
	if err := largeFile.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Load(root, ""); err == nil {
		t.Fatal("full working-tree acquisition unexpectedly accepted the oversized unrelated target")
	}

	actual, err := observeCanonicalProjectionOutputs(root, written)
	if err != nil {
		t.Fatalf("observe selected materialized artifacts: %v", err)
	}
	if len(actual.Files) != len(written) || len(actual.Files) != 1 {
		t.Fatalf("post-write observation escaped written paths: got %v, want %v", actual.Files, written)
	}
	record, err := buildCanonicalProjectionRecord(prepared, observed, actual, written, records.StateMaterializedUnverified)
	if err != nil {
		t.Fatalf("build record from selected materialized bytes: %v", err)
	}
	name := written[0]
	artifact := record.Artifacts[0]
	if artifact.Path != name || artifact.Digest != sha256Prefix(sha256Hex(prepared.Outputs[name])) || artifact.Mode != snapshot.RegularMode {
		t.Fatalf("record does not bind exact materialized output: artifact=%+v", artifact)
	}
}
