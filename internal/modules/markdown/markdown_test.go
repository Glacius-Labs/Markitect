package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestRenderPresentsUnknownKindGenericallyAndDeterministically(t *testing.T) {
	schema := core.Schema{
		APIVersion: "example.test/v1",
		Purpose:    "Project-owned widget semantics.",
		Kinds: map[string]core.Kind{
			"Widget": {
				Purpose: "A project-specific deployable unit.",
				Properties: map[string]core.Property{
					"replicas": {Purpose: "Desired instance count.", Type: core.TypeInteger, MinCount: 1, MaxCount: 1},
					"labels": {Purpose: "Operator annotations.", Type: core.TypeObject, MinCount: 0, MaxCount: 1, Properties: map[string]core.Property{
						"team": {Purpose: "Owning team.", Type: core.TypeString, MinCount: 0, MaxCount: 1},
					}},
				},
			},
		},
	}
	defs := []core.Definition{
		{APIVersion: "example.test/v1", Kind: "Widget", Metadata: core.Metadata{Name: "second", Namespace: "apps"}, Purpose: "A second widget.", Spec: map[string]any{"replicas": 2}},
		{APIVersion: "example.test/v1", Kind: "Widget", Metadata: core.Metadata{Name: "first", Namespace: "apps"}, Purpose: "The first widget.", Spec: map[string]any{"labels": map[string]any{"team": "ops"}, "replicas": 1}},
	}

	first := Render(Input{Definitions: defs, Schemas: []core.Schema{schema}, TargetPath: "docs/widgets.md"})
	second := Render(Input{Definitions: []core.Definition{defs[1], defs[0]}, Schemas: []core.Schema{schema}, TargetPath: "docs/widgets.md"})
	if len(first.Diagnostics) != 0 || len(first.Files) != 1 {
		t.Fatalf("unexpected result: %#v", first)
	}
	if !bytes.Equal(first.Files["docs/widgets.md"], second.Files["docs/widgets.md"]) {
		t.Fatal("rendering changed with Definition input order")
	}
	rendered := string(first.Files["docs/widgets.md"])
	for _, want := range []string{"Widget", "Project-owned widget semantics.", "Desired instance count.", "labels.team", "The first widget.", "A second widget."} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered Markdown does not contain %q", want)
		}
	}
}

func TestRenderRequiresExactTargetAndNonemptyScope(t *testing.T) {
	if result := Render(Input{Definitions: []core.Definition{{}}, TargetPath: " "}); len(result.Diagnostics) == 0 || len(result.Files) != 0 {
		t.Fatalf("expected target diagnostic, got %#v", result)
	}
	if result := Render(Input{TargetPath: "docs/model.md"}); len(result.Diagnostics) == 0 || len(result.Files) != 0 {
		t.Fatalf("expected empty-scope diagnostic, got %#v", result)
	}
}
