package app

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestDocumentationScenarioTracksExplicitCodeInput(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate scenario test source")
	}
	fixtureRoot := filepath.Join(filepath.Dir(sourceFile), "..", "..", "examples", "documentation")
	working, err := source.Load(fixtureRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	fixed := func(revision string) *source.Snapshot {
		snapshot := &source.Snapshot{
			Revision: revision, Provisional: false,
			Files: make(map[string][]byte, len(working.Files)),
			Modes: make(map[string]string, len(working.Modes)),
		}
		for name, data := range working.Files {
			snapshot.Files[name] = append([]byte(nil), data...)
			snapshot.Modes[name] = working.Modes[name]
		}
		return snapshot
	}

	baseSnapshot := fixed(strings.Repeat("a", 40))
	candidateSnapshot := fixed(strings.Repeat("b", 40))
	codePath := "docs/implementation/src/worker.go"
	candidateCode := strings.Replace(string(candidateSnapshot.Files[codePath]), "maxAttempts = 3", "maxAttempts = 5", 1)
	if candidateCode == string(candidateSnapshot.Files[codePath]) {
		t.Fatal("fixture Go source did not contain the expected attempt count")
	}
	candidateSnapshot.Files[codePath] = []byte(candidateCode)

	base, err := Parse(baseSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := Parse(candidateSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	for label, project := range map[string]*Project{"base": base, "candidate": candidate} {
		if len(project.Diagnostics) != 0 {
			t.Errorf("%s has structural diagnostics: %#v", label, project.Diagnostics)
		}
		if findings := CheckOutputs(project); len(findings) != 0 {
			t.Errorf("%s generated views drifted after a source-only change: %#v", label, findings)
		}
	}

	entry := "implementation/Workflow/startup-review"
	ctx, err := CompileContext(candidate, entry, "test")
	if err != nil {
		t.Fatal(err)
	}
	wantContext := map[string]bool{
		candidate.Graph.Project.GraphKey():       true,
		"implementation/Workflow/startup-review": true,
		"implementation/Text/worker-behavior":    true,
		"file:docs/implementation/src/worker.go": true,
	}
	gotContext := make(map[string]bool, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		gotContext[input.Key] = true
		if input.Key == "file:"+codePath && !strings.Contains(input.Text, "maxAttempts = 5") {
			t.Errorf("context did not capture candidate Go source: %q", input.Text)
		}
	}
	if len(gotContext) != len(wantContext) {
		t.Errorf("context contains %d inputs, want exactly %d: %v", len(gotContext), len(wantContext), gotContext)
	}
	for key := range wantContext {
		if !gotContext[key] {
			t.Errorf("context omitted %s: %v", key, gotContext)
		}
	}
	if gotContext["operations/Text/deployment-guide"] {
		t.Error("context included independent documentation from the operations area")
	}

	impact := Changes(base, candidate)
	if len(impact.Changed) != 1 || impact.Changed[0] != codePath {
		t.Fatalf("changed paths = %v, want only %s", impact.Changed, codePath)
	}
	assertAffected(t, impact,
		"implementation/Text/worker-behavior",
		"implementation/Workflow/startup-review",
	)
	if contains(impact.Affected, "operations/Text/deployment-guide") {
		t.Error("impact included independent documentation from the operations area")
	}

	config := ReviewConfig{Question: "Does the documented worker behavior match the source?", PromptVersion: "scenario-v1", Model: "test-model", Effort: "high", AllowReuse: true}
	record, err := RecordReview(base, entry, "test-version", "test-tool", config, "synthetic test report")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := ReuseReview(base, base, record, "test-version", "test-tool", config)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != ReviewReusable {
		t.Fatalf("unchanged evidence status = %s, want reusable", unchanged.Status)
	}
	stale, err := ReuseReview(base, candidate, record, "test-version", "test-tool", config)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Status != ReviewRequired {
		t.Fatalf("source change evidence status = %s, want review-required", stale.Status)
	}
}
