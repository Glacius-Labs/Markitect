package markdown

import (
	"reflect"
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
	files["docs/review.md"] = []byte("Review owns approval.\nReview owns approval.\n")
	got = Validate(m, config, files)
	if len(got) != 1 || got[0].Code != "consistency.quote" {
		t.Fatalf("repeated quote = %#v", got)
	}
}

func TestValidateFunctionalClaimsReportsEveryConflictingOwnerInStableOrder(t *testing.T) {
	items := []struct{ key, value string }{{"first", "operations"}, {"second", "review"}, {"third", "security"}}
	m := core.SemanticModel{}
	files := map[string][]byte{}
	for _, item := range items {
		file := "docs/" + item.key + ".md"
		quote := item.key + " owns approval."
		files[file] = []byte(quote + "\n")
		m.Resources = append(m.Resources, core.ModelResource{Identity: core.ModelIdentity{Kind: "Rule", Name: item.key, Key: item.key}, Source: core.ModelSource{Path: "docs/" + item.key + ".yaml"}, Data: map[string]any{"files": []any{file}, "assertions": []any{map[string]any{"subject": "release-approval", "predicate": "owner", "value": item.value, "source": file, "quote": quote}}}})
	}
	findings := Validate(m, Config{FunctionalPredicates: []string{"owner"}}, files)
	if len(findings) != 2 || findings[0].Code != "consistency.conflict" || findings[0].Path != "docs/second.md" || findings[1].Code != "consistency.conflict" || findings[1].Path != "docs/third.md" {
		t.Fatalf("all conflicts should follow stable owner-key order: %#v", findings)
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
	}{{"../a%20b.md?mode=1#part", "docs/a b.md", true, false}, {"plus+name.md", "docs/sub/plus+name.md", true, false}, {"foo%23bar.md#heading", "docs/sub/foo#bar.md", true, false}, {"encoded%252Fslash.md", "docs/sub/encoded%2Fslash.md", true, false}, {"a%2Fb.md", "docs/sub/a/b.md", true, false}, {"./", "docs/sub", true, false}, {"../../", ".", true, false}, {"foo%3Fbar.md", "docs/sub/foo?bar.md", true, false}, {"x:custom", "", false, false}, {"bad%00.md", "", true, true}, {"folder\\file.md", "", true, true}, {"#heading", "", false, false}, {"https://example.org/x", "", false, false}, {"//example.org/x", "", false, false}, {"../../../outside.md", "", true, true}, {"/absolute.md", "", true, true}, {"C:\\absolute.md", "", true, true}, {"C:/absolute.md", "", true, true}, {"bad%ZZ.md", "", true, true}} {
		got, local, err := normalizeRouterTarget("docs/sub/README.md", test.target)
		if got != test.want || local != test.local || (err != nil) != test.invalid {
			t.Errorf("normalize %q = %q,%t,%v", test.target, got, local, err)
		}
	}
}

func TestDocumentationRouterParserBoundaries(t *testing.T) {
	tests := []struct {
		name, markdown string
		want           []routerLink
	}{
		{"shortcut at EOF", "[Guide]: guide.md\n\n[Guide]", []routerLink{{"guide.md", 3}}},
		{"first reference wins", "[g]: first.md\n[g]: second.md\n\n[g]", []routerLink{{"first.md", 4}}},
		{"escaped punctuation", `[Guide](a\(b\).md)`, []routerLink{{"a(b).md", 1}}},
		{"literal backslash", `[Guide](folder\file.md)`, []routerLink{{`folder\file.md`, 1}}},
		{"reference escape", "[g]: a\\(b\\).md\n\n[g]", []routerLink{{"a(b).md", 3}}},
		{"image is not coverage", "![Guide](guide.md)", nil},
		{"reference image is not coverage", "[g]: guide.md\n\n![Guide][g]", nil},
		{"invalid inline suffix", "[Guide](guide.md arbitrary prose)", nil},
		{"parenthesized title", "[Guide](guide.md (title))", []routerLink{{"guide.md", 1}}},
		{"fence closing suffix", "```md\n```text\n[Example](missing.md)\n```\n[Guide](guide.md)", []routerLink{{"guide.md", 5}}},
		{"invalid backtick opener", "```info`text\n[Guide](guide.md)", []routerLink{{"guide.md", 2}}},
		{"escaped backticks", "\\`[Guide](guide.md)\\`", []routerLink{{"guide.md", 1}}},
		{"backslash inside code is literal", "`code\\` [Guide](guide.md)", []routerLink{{"guide.md", 1}}},
		{"exact inline delimiter", "` example `` [Example](missing.md) ` [Guide](guide.md)", []routerLink{{"guide.md", 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := markdownRouterLinks([]byte(tt.markdown)); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("links=%#v want %#v", got, tt.want)
			}
		})
	}
}

func TestDocumentationRootWithOnlyAssetsStillNeedsRouterInModule(t *testing.T) {
	got := ValidateDocumentationRouters([]string{"docs"}, map[string][]byte{"docs/assets/logo.png": []byte("image")})
	if len(got) != 1 || got[0].Code != "documentation.router.missing" || got[0].Path != "docs/README.md" {
		t.Fatalf("asset-only root = %#v", got)
	}
}

func TestDocumentationRouterCanLinkRepositoryRootInModule(t *testing.T) {
	got := ValidateDocumentationRouters([]string{"docs"}, map[string][]byte{"docs/README.md": []byte("[Repository](../)")})
	if len(got) != 0 {
		t.Fatalf("repository root link = %#v", got)
	}
}
