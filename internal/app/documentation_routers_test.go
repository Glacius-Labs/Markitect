package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func routerProject(roots []string, files map[string]string) *Project {
	bytes := make(map[string][]byte, len(files))
	for name, value := range files {
		bytes[name] = []byte(value)
	}
	return &Project{
		Snapshot: &source.Snapshot{Revision: strings.Repeat("a", 40), Files: bytes},
		Graph:    &core.Graph{Project: &core.Resource{Spec: core.Spec{Documentation: &core.Documentation{Roots: roots}}}},
	}
}

func TestDocumentationParticipationAndDirectCoverage(t *testing.T) {
	project := routerProject([]string{"docs"}, map[string]string{
		"docs/README.md":            "[Area](area/) [Policy](policy.md) [Shortcut](area/deep.md)\n",
		"docs/policy.md":            "policy\n",
		"docs/area/README.md":       "[Deep](deep.md)\n",
		"docs/area/deep.md":         "deep\n",
		"docs/assets/logo.png":      "image",
		"docs/config/settings.yaml": "key: value\n",
	})
	if got := CheckDocumentationRouters(project); len(got) != 0 {
		t.Fatalf("valid local router tree reported diagnostics: %#v", got)
	}
	delete(project.Snapshot.Files, "docs/area/README.md")
	project.Snapshot.Files["docs/README.md"] = []byte("[Policy](policy.md)\n")
	got := CheckDocumentationRouters(project)
	want := []string{"documentation.router.unlisted-directory", "documentation.router.missing"}
	if len(got) != len(want) || got[0].Code != want[0] || got[1].Code != want[1] {
		t.Fatalf("missing direct router coverage = %#v, want %v", got, want)
	}
	if !reflect.DeepEqual(got, CheckDocumentationRouters(project)) {
		t.Fatal("equal snapshots produced different diagnostic order")
	}
}

func TestDocumentationRootAndDirectFileDiagnostics(t *testing.T) {
	project := routerProject([]string{"docs", "absent"}, map[string]string{
		"docs/README.md": "[Broken](missing.md)\n",
		"docs/page.md":   "page\n",
	})
	got := CheckDocumentationRouters(project)
	codes := map[string]bool{}
	for _, finding := range got {
		codes[finding.Code] = true
	}
	for _, code := range []string{"documentation.root.missing", "documentation.router.unlisted-file", "documentation.link.missing"} {
		if !codes[code] {
			t.Errorf("missing %s in %#v", code, got)
		}
	}
}

func TestNormalizeRouterTarget(t *testing.T) {
	tests := []struct {
		in, want       string
		local, invalid bool
	}{
		{"../a%20b.md?mode=1#part", "docs/a b.md", true, false},
		{"plus+name.md", "docs/sub/plus+name.md", true, false},
		{"foo%23bar.md#heading", "docs/sub/foo#bar.md", true, false},
		{"encoded%252Fslash.md", "docs/sub/encoded%2Fslash.md", true, false},
		{"a%2Fb.md", "docs/sub/a/b.md", true, false},
		{"./", "docs/sub", true, false},
		{"../../", ".", true, false},
		{"foo%3Fbar.md", "docs/sub/foo?bar.md", true, false},
		{"x:custom", "", false, false},
		{"bad%00.md", "", true, true},
		{"folder\\file.md", "", true, true},
		{"#heading", "", false, false},
		{"https://example.org/x", "", false, false},
		{"//example.org/x", "", false, false},
		{"../../../outside.md", "", true, true},
		{"/absolute.md", "", true, true},
		{"C:\\absolute.md", "", true, true},
		{"C:/absolute.md", "", true, true},
		{"bad%ZZ.md", "", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, local, err := normalizeRouterTarget("docs/sub/README.md", tt.in)
			if got != tt.want || local != tt.local || (err != nil) != tt.invalid {
				t.Fatalf("normalize(%q) = %q, %t, %v; want %q, %t, invalid=%t", tt.in, got, local, err, tt.want, tt.local, tt.invalid)
			}
		})
	}
}

func TestMarkdownRouterLinksIgnoreCodeAndResolveReferences(t *testing.T) {
	data := []byte("[Inline](<a b.md> \"title\")\n[Ref][guide]\n[guide]: guide.md\n[Nested](func(a).md#part)\n[the [nested] label](nested.md)\n`[Code](missing.md)`\n```md\n[Fence](missing.md)\n```\n")
	got := markdownRouterLinks(data)
	want := []routerLink{{target: "a b.md", line: 1}, {target: "guide.md", line: 2}, {target: "func(a).md#part", line: 4}, {target: "nested.md", line: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("links = %#v, want %#v", got, want)
	}
}

func TestRouterLinksDoNotBecomeGraphDependencies(t *testing.T) {
	project := routerProject([]string{"docs"}, map[string]string{
		"docs/README.md": "[Other](../other.md)\n",
		"other.md":       "outside the documentation root\n",
	})
	if got := CheckDocumentationRouters(project); len(got) != 0 {
		t.Fatalf("cross-area navigation is allowed: %#v", got)
	}
	if len(project.Graph.Edges) != 0 {
		t.Fatalf("navigation created graph edges: %#v", project.Graph.Edges)
	}
}

func TestMarkdownRouterLinkBoundaries(t *testing.T) {
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
		{"exact inline delimiter", "` example `` [Example](missing.md) ` [Guide](guide.md)", []routerLink{{"guide.md", 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := markdownRouterLinks([]byte(tt.markdown)); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("links = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDocumentationRootWithOnlyAssetsStillNeedsRouter(t *testing.T) {
	p := routerProject([]string{"docs"}, map[string]string{"docs/assets/logo.png": "image"})
	got := CheckDocumentationRouters(p)
	if len(got) != 1 || got[0].Code != "documentation.router.missing" || got[0].Path != "docs/README.md" {
		t.Fatalf("asset-only root = %#v", got)
	}
}

func TestRouterCanLinkRepositoryRoot(t *testing.T) {
	p := routerProject([]string{"docs"}, map[string]string{"docs/README.md": "[Repository](../)"})
	if got := CheckDocumentationRouters(p); len(got) != 0 {
		t.Fatalf("repository root link = %#v", got)
	}
}
