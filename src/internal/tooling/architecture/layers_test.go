package architecture

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var layerFixtureMap = CodeMap{Packages: []MappedPackage{
	{Path: "src/internal/host/projectcoverage", Layer: "core"},
	{Path: "src/internal/host/guardedwrite", Layer: "infrastructure"},
	{Path: "src/internal/infrastructure/source", Layer: "infrastructure"},
	{Path: "src/internal/host/projectwork", Layer: "application"},
	{Path: "src/internal/host/projectrun", Layer: "runtime"},
	{Path: "src/internal/host", Layer: "legacy"},
	{Path: "src/internal/host/compat/v0_13/kernel", Layer: "legacy"},
	{Path: "src/cmd/markitect", Layer: "cli"},
	{Path: "src/harness/examples", Layer: "harness"},
}}

func TestMapLayersForbidProductToLegacy(t *testing.T) {
	for _, e := range []Edge{
		{"coverage.go", 3, "src/internal/host/projectcoverage", "src/internal/host", false},
		{"source.go", 4, "src/internal/infrastructure/source", "src/internal/host/compat/v0_13/kernel", false},
		{"work.go", 5, "src/internal/host/projectwork", "src/internal/host", false},
		{"run.go", 6, "src/internal/host/projectrun", "src/internal/host/compat/v0_13/kernel", false},
		{"work_test.go", 7, "src/internal/host/projectwork", "src/internal/host/compat/v0_13/kernel", true},
	} {
		got := CheckMapLayers([]Edge{e}, layerFixtureMap)
		want := []Violation{{e, "product package may not import a legacy package"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s -> %s: got %v", e.From, e.To, got)
		}
	}
}

func TestMapLayersAllowOtherEdges(t *testing.T) {
	edges := []Edge{
		{From: "src/internal/host", To: "src/internal/host/projectwork"},
		{From: "src/internal/host/compat/v0_13/kernel", To: "src/internal/host"},
		{From: "src/internal/host/projectwork", To: "src/internal/host/projectrun"},
		{From: "src/internal/host/projectrun", To: "src/internal/host/guardedwrite"},
		{From: "src/harness/examples", To: "src/internal/host", Test: true},
		{From: "src/cmd/markitect", To: "src/internal/host"},
		{From: "src/internal/host/unmapped", To: "src/internal/host"},
		{From: "src/internal/host/projectwork", To: "src/internal/host/unmapped"},
		{File: "work.go", From: "src/internal/host/projectwork"},
	}
	if got := CheckMapLayers(edges, layerFixtureMap); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestGuardedWriteUseAllowsOnlyTheGuardedAPI(t *testing.T) {
	root := t.TempDir()
	const imp = `"` + ModulePath + GuardedWritePackage + `"`
	files := map[string]string{
		CodeMapSource: `packages:
  - {path: src/internal/host/guardedwrite, layer: infrastructure, purpose: Writes., doc: README.md}
  - {path: src/internal/host/projectexplore, layer: application, purpose: Explores., doc: README.md}
  - {path: src/internal/host/projectwork, layer: application, purpose: Works., doc: README.md}
  - {path: src/internal/host/projectrun, layer: runtime, purpose: Runs., doc: README.md}
  - {path: src/internal/host/projectsetup, layer: runtime, purpose: Sets up., doc: README.md}
  - {path: src/internal/host, layer: legacy, purpose: Legacy., doc: README.md}
`,
		"src/internal/host/guardedwrite/guardedwrite.go": "package guardedwrite\n\nfunc OpenRoot(string) {}\n",
		"src/internal/host/guardedwrite/root_test.go":    "package guardedwrite_test\n\nimport " + imp + "\n\nvar _ = guardedwrite.OpenRoot\n",
		"src/internal/host/projectwork/ok.go":            "package projectwork\n\nimport " + imp + "\n\nvar _ = guardedwrite.Apply\nvar _ []guardedwrite.Change\nvar _ *guardedwrite.Capture\n",
		"src/internal/host/projectwork/bad.go":           "package projectwork\n\nimport " + imp + "\n\nvar _ = guardedwrite.CaptureFiles\nvar _ = guardedwrite.OpenRoot\n",
		"src/internal/host/projectrun/alias_test.go":     "package projectrun\n\nimport gw " + imp + "\n\nvar _ = gw.Apply\nvar _ = gw.SafeDestination\n",
		"src/internal/host/projectsetup/dot.go":          "package projectsetup\n\nimport . " + imp + "\n\nvar _ = Apply\n",
		"src/internal/host/projectexplore/blank.go":      "package projectexplore\n\nimport _ " + imp + "\n",
		"src/internal/host/legacy.go":                    "package host\n\nimport " + imp + "\n\nvar _ = guardedwrite.OpenRoot\n",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	violations, err := CheckRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range violations {
		got = append(got, v.String())
	}
	// The coarse gate already forbids guardedwrite's own external test package
	// from importing it; the selector rule adds no second finding there.
	const api = "product package may use only the guarded write API (Apply, ApplyChecked, Capture, CaptureFiles, Change, File, Result), not "
	want := []string{
		"src/internal/host/guardedwrite/root_test.go:3: src/internal/host/guardedwrite imports src/internal/host/guardedwrite: self-import is forbidden",
		"src/internal/host/projectrun/alias_test.go:6: src/internal/host/projectrun imports src/internal/host/guardedwrite: " + api + "guardedwrite.SafeDestination",
		"src/internal/host/projectsetup/dot.go:3: src/internal/host/projectsetup imports src/internal/host/guardedwrite: product package may not dot-import guardedwrite; use qualified names so the gate can check them",
		"src/internal/host/projectwork/bad.go:6: src/internal/host/projectwork imports src/internal/host/guardedwrite: " + api + "guardedwrite.OpenRoot",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("violations:\n%q\nwant:\n%q", got, want)
	}
	if !violations[1].Edge.Test || violations[2].Edge.Test {
		t.Errorf("test-file marking: %v", violations)
	}
}

func TestCheckRepositoryRequiresCodeMap(t *testing.T) {
	if _, err := CheckRepository(t.TempDir()); err == nil {
		t.Fatal("checked a repository without a code map")
	}
}
