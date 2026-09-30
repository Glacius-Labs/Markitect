package app

import (
	"testing"

	"markitect/internal/core"
	"markitect/internal/source"
)

func TestCompileContextIncludesOnlySelectedImplementationAndItsExplicitFiles(t *testing.T) {
	project := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "sample"},
		Path:       projectPath,
		Spec: core.Spec{
			Profile: "generic",
			Areas:   []core.Area{{Name: projectNS, Path: "docs/general"}},
			Bindings: []core.Binding{{
				Contract:       core.Ref{Kind: "Contract", Name: "review", Namespace: projectNS},
				Implementation: core.Ref{Kind: "Agent", Name: "reviewer-a", Namespace: projectNS},
			}},
		},
	}
	contract := core.Resource{
		APIVersion: core.APIVersion, Kind: "Contract", Metadata: core.Metadata{Name: "review", Namespace: projectNS},
		Path: "docs/general/contracts/review.yaml",
		Spec: core.Spec{Text: "Review contract.", Kind: "Agent", Input: []string{"change"}, Output: []string{"findings"}, Files: []string{"docs/general/contracts/review.go"}},
	}
	selected := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-a", Namespace: projectNS},
		Path: "docs/general/agents/reviewer-a.yaml",
		Spec: core.Spec{Text: "Selected reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/general/agents/selected-notes.md"}},
	}
	alternative := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-b", Namespace: projectNS},
		Path: "docs/general/agents/reviewer-b.yaml",
		Spec: core.Spec{Text: "Alternative reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/general/agents/alternative-notes.md"}},
	}
	consumer := core.Resource{
		APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review-change", Namespace: projectNS},
		Path: "docs/general/workflows/review-change.yaml",
		Spec: core.Spec{Text: "Review the change.", Needs: []core.Ref{{Name: "review"}}},
	}

	resources := []*core.Resource{&project, &contract, &selected, &alternative, &consumer}
	snapshot := &source.Snapshot{Revision: "fixed-review-snapshot", Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data := encodeResource(t, *resource)
		snapshot.Files[resource.Path] = data
		snapshot.Modes[resource.Path] = "100644"
	}
	texts := map[string]string{
		"docs/general/contracts/review.go":         "type ReviewInput struct{}\n",
		"docs/general/agents/selected-notes.md":    "Selected reference text.\n",
		"docs/general/agents/alternative-notes.md": "Unselected reference text.\n",
	}
	for name, text := range texts {
		snapshot.Files[name] = []byte(text)
		snapshot.Modes[name] = "100644"
	}

	parsed, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected graph/input diagnostics: %#v", parsed.Diagnostics)
	}
	ctx, err := CompileContext(parsed, consumer.Metadata.Namespace+"/Workflow/"+consumer.Metadata.Name, "test", "tool-digest")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.SnapshotDigest != snapshot.Digest() || ctx.ToolDigest != "tool-digest" || ctx.Provisional {
		t.Fatalf("context is not bound to the fixed snapshot/tool: %#v", ctx)
	}
	entries := make(map[string]ContextInput, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		entries[input.Key] = input
	}
	for _, key := range []string{contract.Key(), selected.Key(), consumer.Key(), project.Key()} {
		if _, ok := entries[key]; !ok {
			t.Errorf("context omitted required resource %s", key)
		}
	}
	if _, ok := entries[alternative.Key()]; ok {
		t.Errorf("context included unselected alternative %s", alternative.Key())
	}
	for name, want := range texts {
		key := "file:" + name
		entry, ok := entries[key]
		if name == "docs/general/agents/alternative-notes.md" {
			if ok {
				t.Errorf("context included unselected input file %s", name)
			}
			continue
		}
		if !ok {
			t.Errorf("context omitted declared input file %s", name)
			continue
		}
		if entry.Text != want || entry.Hash != Hash([]byte(want)) {
			t.Errorf("file context content/hash mismatch for %s: %#v", name, entry)
		}
	}
}
