package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForbiddenEdges(t *testing.T) {
	for _, e := range []Edge{{"core.go", 3, "src/internal/core", "src/internal/modules/pipelines", false}, {"a.go", 4, "src/internal/modules/a", "src/internal/modules/b", false}, {"a_test.go", 5, "src/internal/modules/a", "src/internal/host", true}, {"main.go", 6, "src/cmd/markitect", "src/internal/core", false}, {"host.go", 7, "src/internal/host", "src/cmd/markitect", false}} {
		v := Check([]Edge{e})
		if len(v) != 1 || !strings.Contains(v[0].String(), e.From+" imports "+e.To) {
			t.Fatalf("missing exact edge: %v", v)
		}
	}
}
func TestPermittedEdges(t *testing.T) {
	edges := []Edge{{From: "src/internal/modules/a", To: "src/internal/core"}, {From: "src/internal/modules/a", To: "src/internal/modules/a/private"}, {From: "src/internal/host", To: "src/internal/modules/a"}, {From: "src/cmd/markitect", To: "src/internal/host/cli"}}
	if v := Check(edges); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestCoreExternalDependencyRequiresApproval(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src/internal", "core")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, imports := range map[string]string{
		"generic.go":       "\"fmt\"\n_ \"go.yaml.in/yaml/v3\"",
		"provider_test.go": "_ \"example.org/provider/sdk\"",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package core\nimport (\n"+imports+"\n)\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	findings := Check(edges)
	if len(findings) != 2 {
		t.Fatalf("Core external dependencies were hidden: %v", findings)
	}
	found := map[string]bool{}
	for _, finding := range findings {
		found[finding.Edge.To] = true
	}
	if !found["go.yaml.in/yaml/v3"] || !found["example.org/provider/sdk"] {
		t.Fatalf("missing exact external dependency: %v", findings)
	}
}
func TestInspectAllPlatformsAndTests(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"src/internal/core/bad_windows.go", "src/internal/modules/a/bad_test.go"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package fixture\nimport _ \""+ModulePath+"src/internal/modules/b\"\n"), 0644); err != nil {
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
	edges, err := Inspect(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range Check(edges) {
		t.Error(v.String())
	}
}

func TestUnclassifiedPackageWithoutLocalImports(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "src/internal", "unowned", "file.go")
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
	for _, e := range []Edge{{From: "src/internal/core", To: "src/internal/core"}, {From: "src/internal/host", To: "src/harness/examples"}, {From: "src/harness/examples", To: "src/internal/core", Test: false}, {From: "examples/arbitrary-new-runtime", To: "src/internal/host"}} {
		if len(Check([]Edge{e})) != 1 {
			t.Fatalf("unexpected pass: %v", e)
		}
	}
	if v := Check([]Edge{{From: "src/harness/examples", To: "src/internal/host", Test: true}}); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestInspectTestFirstDoesNotHideProductionFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src/harness/examples")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"00_test.go", "z.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package examples\nimport _ \"fmt\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	findings := Check(edges)
	if len(findings) != 1 || findings[0].Edge.File != "src/harness/examples/z.go" || findings[0].Edge.Test || findings[0].Rule != "unclassified product package" {
		t.Fatalf("test-first order hid the production file: %v", findings)
	}
}

func TestAdoptingCodeFixtureHasNoProductDependencyPrivilege(t *testing.T) {
	const fixture = "examples/documentation/docs/implementation/src"
	if got := Check([]Edge{{From: fixture}}); len(got) != 0 {
		t.Fatal(got)
	}
	for _, target := range []string{"src/internal/core", "src/internal/host", "src/internal/modules/a"} {
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
	for _, target := range []string{"src/internal/core", "src/internal/host", "src/internal/modules/a"} {
		if got := Check([]Edge{{From: "integration", To: target}}); len(got) != 1 {
			t.Fatalf("bootstrap import privilege: %s: %v", target, got)
		}
	}
}

func TestTestkitIsTestOnlyAndStandalone(t *testing.T) {
	if got := Check([]Edge{{From: "src/internal/testkit"}}); len(got) != 0 {
		t.Fatalf("testkit classification: %v", got)
	}
	for _, from := range []string{"src/internal/core", "src/internal/modules/a", "src/internal/host", "src/internal/infrastructure/source", "src/harness/examples"} {
		if got := Check([]Edge{{From: from, To: "src/internal/testkit", Test: true}}); len(got) != 0 {
			t.Fatalf("test in %s importing testkit: %v", from, got)
		}
		if got := Check([]Edge{{From: from, To: "src/internal/testkit"}}); len(got) != 1 {
			t.Fatalf("production code in %s importing testkit: %v", from, got)
		}
	}
	if got := Check([]Edge{{From: "src/internal/testkit", To: "src/internal/core"}}); len(got) != 1 {
		t.Fatalf("testkit importing a local package: %v", got)
	}
	if got := Check([]Edge{{From: "src/internal/host", To: "src/internal/testkit/gitfixture"}}); len(got) != 1 {
		t.Fatalf("production code importing a testkit subpackage: %v", got)
	}
}
