package render

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestMarkdownViewsUseAreaRelativeKindQualifiedPathsAndIndices(t *testing.T) {
	rule := resource("Rule", "docs/policy/nested/privacy.yaml", "privacy", "policy", core.Spec{Text: "Keep data safe."})
	skill := resource("Skill", "docs/policy/skills/review.skill.yml", "review", "policy", core.Spec{Text: "Review changes.", Rules: []core.Ref{{Name: "privacy"}}})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "policy", Path: "docs/policy"}}})
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{rule.Key(): rule, skill.Key(): skill}, ResourceAreas: map[string]core.Area{rule.GraphKey(): {Name: "policy", Path: "docs/policy"}, skill.GraphKey(): {Name: "policy", Path: "docs/policy"}}}
	outputs, owners, err := GenerateWithOwners(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"docs/markitect/policy/nested/privacy.rule.md",
		"docs/markitect/policy/skills/review.skill.md",
		"docs/markitect/policy/README.md",
		"docs/markitect/policy/nested/README.md",
		"docs/markitect/policy/skills/README.md",
		"docs/markitect/README.md",
	} {
		if outputs[name] == nil {
			t.Errorf("missing Markdown output %q; got %v", name, outputPaths(outputs))
		}
	}
	view := string(outputs["docs/markitect/policy/skills/review.skill.md"])
	if !strings.Contains(view, "[Rule: privacy](../nested/privacy.rule.md)") {
		t.Errorf("Markdown dependency does not link to the generated local view: %s", view)
	}
	if !strings.Contains(string(outputs["docs/markitect/policy/README.md"]), "skills/README.md") {
		t.Error("Area index does not link to the nested skills index")
	}
	if got := strings.Join(owners["docs/markitect/README.md"], ","); got != project.Key() {
		t.Errorf("root index owners = %q, want project owner %q", got, project.Key())
	}
}

func TestMarkdownLinksEscapeURIPathSegments(t *testing.T) {
	rule := resource("Rule", "docs/area/policy #1 (draft).yaml", "privacy", "area", core.Spec{})
	workflow := resource("Workflow", "docs/area/run.yaml", "run", "area", core.Spec{Uses: []core.Ref{{Kind: "Rule", Name: "privacy"}}})
	p := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "area", Path: "docs/area"}}})
	g := &core.Graph{Project: p, Resources: map[string]*core.Resource{rule.Key(): rule, workflow.Key(): workflow}, ResourceAreas: map[string]core.Area{rule.GraphKey(): {Name: "area", Path: "docs/area"}, workflow.GraphKey(): {Name: "area", Path: "docs/area"}}}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["docs/markitect/area/run.workflow.md"])
	for _, escaped := range []string{"policy%20%231%20%28draft%29.rule.md"} {
		if !strings.Contains(body, escaped) {
			t.Errorf("link target did not URI-escape path segment: %q missing from %s", escaped, body)
		}
	}
}

func TestMarkdownViewPathDeduplicatesMatchingKindSuffix(t *testing.T) {
	r := resource("Skill", "docs/area/skills/review.skill.yaml", "review", "area", core.Spec{})
	p := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "area", Path: "docs/area"}}})
	g := &core.Graph{Project: p, Resources: map[string]*core.Resource{r.Key(): r}, ResourceAreas: map[string]core.Area{r.GraphKey(): {Name: "area", Path: "docs/area"}}}
	got, err := MarkdownViewPath(g, r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "docs/markitect/area/skills/review.skill.md" {
		t.Fatalf("MarkdownViewPath = %q", got)
	}
}

func TestMarkdownViewsRejectDeterministicSourceAndOutputCollisions(t *testing.T) {
	a := resource("Text", "docs/area/a.yaml", "a", "area", core.Spec{})
	b := resource("Text", "docs/area/a.yml", "b", "area", core.Spec{})
	p := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "area", Path: "docs/area"}}})
	g := &core.Graph{Project: p, Resources: map[string]*core.Resource{a.Key(): a, b.Key(): b}, ResourceAreas: map[string]core.Area{a.GraphKey(): {Name: "area", Path: "docs/area"}, b.GraphKey(): {Name: "area", Path: "docs/area"}}}
	if _, err := Generate(g); err == nil || !strings.Contains(err.Error(), "Markdown view path collision") {
		t.Fatalf("same output from .yaml/.yml sources should collide deterministically, got %v", err)
	}
}

func TestMarkdownViewCannotEscapeReservedRootWithMalformedAreaName(t *testing.T) {
	r := resource("Text", "docs/area/note.yaml", "note", "..", core.Spec{})
	p := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "..", Path: "docs/area"}}})
	g := &core.Graph{Project: p, Resources: map[string]*core.Resource{r.Key(): r}, ResourceAreas: map[string]core.Area{r.GraphKey(): {Name: "..", Path: "docs/area"}}}
	if output, err := MarkdownViewPath(g, r); err == nil || strings.HasPrefix(output, "docs/markitect/") {
		t.Fatalf("malformed Area escaped or was accepted: path=%q error=%v", output, err)
	}
}
