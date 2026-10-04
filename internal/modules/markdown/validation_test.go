package markdown

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestValidateFunctionalClaimsRequiresOptInDeclaredUniqueQuoteAndReportsConflicts(t *testing.T) {
	makeResource := func(key, value, source, quote string) core.ModelResource {
		return core.ModelResource{Identity: core.ModelIdentity{Kind: "Rule", Name: key, Key: key}, Source: core.ModelSource{Path: "docs/" + key + ".yaml", Line: 2}, Data: map[string]any{
			"files": []any{source}, "assertions": []any{map[string]any{"subject": "release-approval", "predicate": "owner", "value": value, "source": source, "quote": quote}},
		}}
	}
	first := makeResource("operations", "operations", "docs/operations.md", "Operations owns approval.")
	second := makeResource("review", "review", "docs/review.md", "Review owns approval.")
	m := core.SemanticModel{Resources: []core.ModelResource{second, first}}
	files := map[string][]byte{"docs/operations.md": []byte("# Contract\nOperations owns approval.\n"), "docs/review.md": []byte("Review owns approval.\n")}
	if got := Validate(m, Config{}, files); len(got) != 0 {
		t.Fatalf("not opted in = %#v", got)
	}
	config := Config{FunctionalPredicates: []string{"owner"}}
	got := Validate(m, config, files)
	if len(got) != 1 || got[0].Code != "consistency.conflict" || got[0].Path != "docs/review.md" || got[0].Line != 1 || !strings.Contains(got[0].Message, "docs/operations.md:2") {
		t.Fatalf("conflict evidence = %#v", got)
	}
	m.Resources[0].Data["assertions"].([]any)[0].(map[string]any)["quote"] = "absent"
	got = Validate(m, config, files)
	if len(got) != 1 || got[0].Code != "consistency.quote" {
		t.Fatalf("missing quote = %#v", got)
	}
	m.Resources[0].Data["assertions"].([]any)[0].(map[string]any)["quote"] = "Review owns approval."
	delete(files, "docs/review.md")
	got = Validate(m, config, files)
	if len(got) != 1 || got[0].Code != "consistency.source" {
		t.Fatalf("missing source = %#v", got)
	}
}

func TestValidateDocumentationRoutersUsesOnlyConfiguredSnapshotRoots(t *testing.T) {
	files := map[string][]byte{
		"docs/README.md":      []byte("[Area](area/) [Policy](policy.md)\n"),
		"docs/policy.md":      []byte("policy\n"),
		"docs/area/README.md": []byte("[Deep](deep.md)\n"),
		"docs/area/deep.md":   []byte("deep\n"),
		"elsewhere/README.md": []byte("ignored\n"),
	}
	if got := ValidateDocumentationRouters([]string{"docs"}, files); len(got) != 0 {
		t.Fatalf("valid router tree = %#v", got)
	}
	delete(files, "docs/area/README.md")
	files["docs/README.md"] = []byte("[Policy](policy.md)\n")
	got := ValidateDocumentationRouters([]string{"docs", "absent"}, files)
	codes := map[string]bool{}
	for _, d := range got {
		codes[d.Code] = true
	}
	for _, want := range []string{"documentation.root.missing", "documentation.router.unlisted-directory", "documentation.router.missing"} {
		if !codes[want] {
			t.Errorf("missing %s in %#v", want, got)
		}
	}
}

func TestDocumentationRouterLinkBoundaries(t *testing.T) {
	data := []byte("[Inline](<a b.md> \"title\")\n[Ref][guide]\n[guide]: guide.md\n[Nested](func(a).md#part)\n[the [nested] label](nested.md)\n`[Code](missing.md)`\n```md\n[Fence](missing.md)\n```\n")
	got := markdownRouterLinks(data)
	want := []routerLink{{target: "a b.md", line: 1}, {target: "guide.md", line: 2}, {target: "func(a).md#part", line: 4}, {target: "nested.md", line: 5}}
	if len(got) != len(want) {
		t.Fatalf("links %#v want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("links %#v want %#v", got, want)
		}
	}
	for _, test := range []struct {
		target, want   string
		local, invalid bool
	}{{"../a%20b.md?mode=1#part", "docs/a b.md", true, false}, {"x:custom", "", false, false}, {"bad%00.md", "", true, true}, {"../../../outside.md", "", true, true}} {
		got, local, err := normalizeRouterTarget("docs/sub/README.md", test.target)
		if got != test.want || local != test.local || (err != nil) != test.invalid {
			t.Errorf("normalize %q = %q,%t,%v", test.target, got, local, err)
		}
	}
}
