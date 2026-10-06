package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
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
		{name: "unknown fields", data: `{"requestDigest":"sha256:x","files":[{"path":"src/a.cs","content":"a","mode":"100755"}]}`},
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
	if _, err := candidateMap(candidate, "src/"); err == nil {
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
	if _, err := candidateMap(CanonicalCandidate{Files: []CanonicalCandidateFile{{Path: "src-old/Orders.cs", Content: "outside"}}}, "src"); err == nil {
		t.Fatal("candidate with a sibling-prefix path was accepted")
	}
}
