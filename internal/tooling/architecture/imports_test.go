package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForbiddenEdges(t *testing.T) {
	for _, e := range []Edge{{"core.go", 3, "internal/core", "internal/modules/pipelines", false}, {"a.go", 4, "internal/modules/a", "internal/modules/b", false}, {"a_test.go", 5, "internal/modules/a", "internal/host", true}, {"main.go", 6, "cmd/markitect", "internal/core", false}, {"host.go", 7, "internal/host", "cmd/markitect", false}} {
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
	edges, err := Inspect(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range Check(edges) {
		t.Error(v.String())
	}
}

func TestUnclassifiedPackageWithoutLocalImports(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "internal", "unowned", "file.go")
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("package unowned\nimport _ \"fmt\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if v := Check(edges); len(v) != 1 || v[0].Rule != "unclassified product package" {
		t.Fatalf("missing package finding: %v", v)
	}
}
func TestHarnessCannotHideProductOrSelfImport(t *testing.T) {
	for _, e := range []Edge{{From: "internal/core", To: "internal/core"}, {From: "internal/host", To: "examples"}, {From: "examples", To: "internal/core", Test: false}, {From: "examples/arbitrary-new-runtime", To: "internal/host"}} {
		if len(Check([]Edge{e})) != 1 {
			t.Fatalf("unexpected pass: %v", e)
		}
	}
	if v := Check([]Edge{{From: "examples", To: "internal/host", Test: true}}); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestAdoptingCodeFixtureHasNoProductDependencyPrivilege(t *testing.T) {
	const fixture = "examples/documentation/docs/implementation/src"
	if got := Check([]Edge{{From: fixture}}); len(got) != 0 {
		t.Fatal(got)
	}
	for _, target := range []string{"internal/core", "internal/host", "internal/modules/a"} {
		if got := Check([]Edge{{From: fixture, To: target}}); len(got) != 1 {
			t.Fatalf("hidden product edge: %s: %v", target, got)
		}
	}
}

// The public single-file bootstrap remains in integration for the supported
// consumer download path. Its tooling owner grants no product import privilege.
func TestStandaloneBootstrapToolingHasNoProductDependencies(t *testing.T) {
	if got := Check([]Edge{{From: "integration"}}); len(got) != 0 {
		t.Fatal(got)
	}
	for _, target := range []string{"internal/core", "internal/host", "internal/modules/a"} {
		if got := Check([]Edge{{From: "integration", To: target}}); len(got) != 1 {
			t.Fatalf("bootstrap import privilege: %s: %v", target, got)
		}
	}
}
