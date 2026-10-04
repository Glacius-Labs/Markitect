package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func routerProject(roots []string, files map[string]string) *Project {
	bytes := make(map[string][]byte, len(files))
	for name, value := range files {
		bytes[name] = []byte(value)
	}
	return &Project{
		Snapshot: &snapshot.Snapshot{ID: strings.Repeat("a", 40), Files: bytes},
		Graph:    &authoring.Graph{Project: &authoring.Resource{Spec: authoring.Spec{Documentation: &authoring.Documentation{Roots: roots}}}},
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
