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

func TestGovernmentApplicationFixtureCannotImportProduct(t *testing.T) {
	for _, from := range []string{"examples/government/inventory", "examples/government-g2/inventory", "examples/government-g2/runner", "examples/government-g3/runner", "examples/government-g3/quantity", "examples/government-g3/price", "examples/government-g3/integration", "examples/government-g4/runner", "examples/government-g4/invoice", "examples/government-g5/gitproxy"} {
		for _, target := range []string{"internal/core", "internal/host/government", "internal/modules/dotnet"} {
			findings := Check([]Edge{{From: from, To: target}})
			if len(findings) != 1 || findings[0].Rule != "adopting-code fixture may not import Markitect product packages" {
				t.Fatalf("fixture boundary weakened for %s: %+v", target, findings)
			}
		}
	}
}

func TestCoreExternalDependencyRequiresApproval(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "core")
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

func TestInspectTestFirstDoesNotHideProductionFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "examples")
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
	if len(findings) != 1 || findings[0].Edge.File != "examples/z.go" || findings[0].Edge.Test || findings[0].Rule != "unclassified product package" {
		t.Fatalf("test-first order hid the production file: %v", findings)
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

func TestCanonicalWorkflowCheckIsIsolatedAdopterCode(t *testing.T) {
	if findings := Check([]Edge{{From: "examples/canonical-workflow/check"}}); len(findings) != 0 {
		t.Fatal(findings)
	}
	for _, target := range []string{"internal/core", "internal/host", "internal/modules/githooks"} {
		findings := Check([]Edge{{From: "examples/canonical-workflow/check", To: target}})
		if len(findings) != 1 || findings[0].Rule != "adopting-code fixture may not import Markitect product packages" {
			t.Fatalf("project-owned check acquired product coupling: %v", findings)
		}
	}
}
