package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForbiddenEdges(t *testing.T) {
	for _, e := range []Edge{{"core.go", 3, "internal/core", "internal/modules/pipelines", false}, {"a.go", 4, "internal/modules/a", "internal/modules/b", false}, {"a_test.go", 5, "internal/modules/a", "internal/host", true}, {"main.go", 6, "cmd/markitect", "internal/core", false}} {
		v := Check([]Edge{e})
		if len(v) != 1 || !strings.Contains(v[0].String(), e.From+" imports "+e.To) {
			t.Fatalf("missing exact edge: %v", v)
		}
	}
}
func TestPermittedEdges(t *testing.T) {
	edges := []Edge{{From: "internal/modules/a", To: "internal/core"}, {From: "internal/modules/a", To: "internal/modules/a/private"}, {From: "internal/host", To: "internal/modules/a"}, {From: "cmd/markitect", To: "internal/host/cli"}}
	if v := Check(edges); len(v) != 0 {
		t.Fatal(v)
	}
}
func TestInspectAllPlatformsAndTests(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"internal/core/bad_windows.go", "internal/modules/a/bad_test.go"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package fixture\nimport _ \""+ModulePath+"internal/modules/b\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(Check(edges)) != 2 {
		t.Fatalf("edges: %v", edges)
	}
}
func TestRepositoryArchitecture(t *testing.T) {
	edges, err := Inspect(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range Check(edges) {
		t.Error(v.String())
	}
}
