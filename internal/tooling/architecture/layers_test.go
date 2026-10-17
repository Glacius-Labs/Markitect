package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var layerFixtureMap = CodeMap{Packages: []MappedPackage{
	{Path: "src/internal/core/projectmodel/projectcoverage", Layer: "core"},
	{Path: "src/internal/infrastructure/guardedwrite", Layer: "infrastructure"},
	{Path: "src/internal/infrastructure/source", Layer: "infrastructure"},
	{Path: "src/internal/application/projectcoverage", Layer: "application"},
	{Path: "src/internal/application/projectwork", Layer: "application"},
	{Path: "src/internal/runtime/projectrun", Layer: "runtime"},
	{Path: "src/internal/host", Layer: "legacy"},
	{Path: "src/internal/host/compat/v0_13/kernel", Layer: "legacy"},
	{Path: "src/cmd/markitect", Layer: "cli"},
	{Path: "src/harness/examples", Layer: "harness"},
}}

func TestMapLayersForbidProductToLegacy(t *testing.T) {
	for _, e := range []Edge{
		{"coverage.go", 3, "src/internal/core/projectmodel/projectcoverage", "src/internal/host", false},
		{"source.go", 4, "src/internal/infrastructure/source", "src/internal/host/compat/v0_13/kernel", false},
		{"work.go", 5, "src/internal/application/projectwork", "src/internal/host", false},
		{"run.go", 6, "src/internal/runtime/projectrun", "src/internal/host/compat/v0_13/kernel", false},
		{"work_test.go", 7, "src/internal/application/projectwork", "src/internal/host/compat/v0_13/kernel", true},
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
		{From: "src/internal/host", To: "src/internal/application/projectwork"},
		{From: "src/internal/host/compat/v0_13/kernel", To: "src/internal/host"},
		{From: "src/internal/application/projectwork", To: "src/internal/runtime/projectrun"},
		{From: "src/internal/runtime/projectrun", To: "src/internal/infrastructure/guardedwrite"},
		{From: "src/harness/examples", To: "src/internal/host", Test: true},
		{From: "src/cmd/markitect", To: "src/internal/host"},
		{From: "src/internal/host/unmapped", To: "src/internal/host"},
		{From: "src/internal/application/projectwork", To: "src/internal/host/unmapped"},
		{File: "work.go", From: "src/internal/application/projectwork"},
	}
	if got := CheckMapLayers(edges, layerFixtureMap); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestGuardedWriteUseAllowsOnlyTheGuardedAPI(t *testing.T) {
	root := t.TempDir()
	const imp = `"` + ModulePath + GuardedWritePackage + `"`
	files := map[string]string{
		"README.md": "# Fixture\n",
		CodeMapSource: `packages:
  - {path: src/internal/infrastructure/guardedwrite, layer: infrastructure, purpose: Writes., doc: README.md}
  - {path: src/internal/application/projectexplore, layer: application, purpose: Explores., doc: README.md}
  - {path: src/internal/application/projectwork, layer: application, purpose: Works., doc: README.md}
  - {path: src/internal/runtime/projectrun, layer: runtime, purpose: Runs., doc: README.md}
  - {path: src/internal/runtime/projectsetup, layer: runtime, purpose: Sets up., doc: README.md}
  - {path: src/internal/host, layer: legacy, purpose: Legacy., doc: README.md}
`,
		"src/internal/infrastructure/guardedwrite/guardedwrite.go": "package guardedwrite\n\nfunc OpenRoot(string) {}\n",
		"src/internal/infrastructure/guardedwrite/root_test.go":    "package guardedwrite_test\n\nimport " + imp + "\n\nvar _ = guardedwrite.OpenRoot\n",
		"src/internal/application/projectwork/ok.go":               "package projectwork\n\nimport " + imp + "\n\nvar _ = guardedwrite.Apply\nvar _ = guardedwrite.ApplyOrderedChecked\nvar _ = guardedwrite.ApplyRecordChecked\nvar _ []guardedwrite.Change\nvar _ *guardedwrite.Capture\n",
		"src/internal/application/projectwork/bad.go":              "package projectwork\n\nimport " + imp + "\n\nvar _ = guardedwrite.CaptureFiles\nvar _ = guardedwrite.OpenRoot\n",
		"src/internal/runtime/projectrun/alias_test.go":            "package projectrun\n\nimport gw " + imp + "\n\nvar _ = gw.Apply\nvar _ = gw.SafeDestination\n",
		"src/internal/runtime/projectsetup/dot.go":                 "package projectsetup\n\nimport . " + imp + "\n\nvar _ = Apply\n",
		"src/internal/application/projectexplore/blank.go":         "package projectexplore\n\nimport _ " + imp + "\n",
		"src/internal/host/legacy.go":                              "package host\n\nimport " + imp + "\n\nvar _ = guardedwrite.OpenRoot\n",
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
	const api = "product package may use only the guarded write API (Apply, ApplyChecked, ApplyOrderedChecked, ApplyRecordChecked, Capture, CaptureFiles, Change, File, NewStoredNames, Result, StoredNames), not "
	want := []string{
		"src/internal/application/projectwork/bad.go:6: src/internal/application/projectwork imports src/internal/infrastructure/guardedwrite: " + api + "guardedwrite.OpenRoot",
		"src/internal/infrastructure/guardedwrite/root_test.go:3: src/internal/infrastructure/guardedwrite imports src/internal/infrastructure/guardedwrite: self-import is forbidden",
		"src/internal/runtime/projectrun/alias_test.go:6: src/internal/runtime/projectrun imports src/internal/infrastructure/guardedwrite: " + api + "guardedwrite.SafeDestination",
		"src/internal/runtime/projectsetup/dot.go:3: src/internal/runtime/projectsetup imports src/internal/infrastructure/guardedwrite: product package may not dot-import guardedwrite; use qualified names so the gate can check them",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("violations:\n%q\nwant:\n%q", got, want)
	}
	if !violations[2].Edge.Test || violations[3].Edge.Test {
		t.Errorf("test-file marking: %v", violations)
	}
}

func TestCheckRepositoryRequiresCodeMap(t *testing.T) {
	if _, err := CheckRepository(t.TempDir()); err == nil {
		t.Fatal("checked a repository without a code map")
	}
}

func TestMapLayersDirectionMatrix(t *testing.T) {
	packages := map[string]string{
		"core":           "src/internal/core/projectmodel/projectcoverage",
		"infrastructure": "src/internal/infrastructure/guardedwrite",
		"runtime":        "src/internal/runtime/projectrun",
		"application":    "src/internal/application/projectwork",
	}
	for _, test := range []struct {
		from, to string
		allowed  bool
	}{
		{"core", "core", true},
		{"core", "infrastructure", false},
		{"core", "runtime", false},
		{"core", "application", false},
		{"infrastructure", "core", true},
		{"infrastructure", "infrastructure", true},
		{"infrastructure", "runtime", false},
		{"infrastructure", "application", false},
		{"runtime", "core", true},
		{"runtime", "infrastructure", true},
		{"runtime", "runtime", true},
		{"runtime", "application", false},
		{"application", "core", true},
		{"application", "infrastructure", true},
		{"application", "runtime", true},
		{"application", "application", true},
	} {
		for _, file := range []struct {
			name string
			test bool
		}{
			{"direction.go", false},
			{"direction_test.go", true},
			{"direction_windows.go", false},
		} {
			t.Run(test.from+"_to_"+test.to+"/"+file.name, func(t *testing.T) {
				edge := Edge{file.name, 3, packages[test.from], packages[test.to], file.test}
				got := CheckMapLayers([]Edge{edge}, layerFixtureMap)
				if test.allowed {
					if len(got) != 0 {
						t.Fatalf("allowed direction: %v", got)
					}
					return
				}
				want := []Violation{{edge, fmt.Sprintf("product-layer direction forbids %s -> %s", test.from, test.to)}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			})
		}
	}
}

func TestCheckRepositoryLayerDirectionIncludesTestsAndPlatforms(t *testing.T) {
	root := t.TempDir()
	const target = "src/internal/application/projectwork"
	files := map[string]string{
		"README.md":         "# Fixture\n",
		CodeMapSource:       layerDirectionFixtureMap("infrastructure", true, true),
		target + "/work.go": "package projectwork\n",
	}
	var want []Violation
	for _, file := range []struct {
		name string
		test bool
	}{
		{"direction.go", false},
		{"direction_test.go", true},
		{"direction_windows.go", false},
	} {
		name := GuardedWritePackage + "/" + file.name
		files[name] = "package guardedwrite\n\nimport _ \"" + ModulePath + target + "\"\n"
		want = append(want, Violation{Edge{name, 3, GuardedWritePackage, target, file.test}, "product-layer direction forbids infrastructure -> application"})
	}
	writeLayerDirectionFixture(t, root, files)
	got, err := CheckRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCheckRepositoryRejectsIncompleteOrInvalidLayers(t *testing.T) {
	const target = "src/internal/application/projectwork"
	for _, test := range []struct {
		name, layer         string
		source, destination bool
		problem             string
	}{
		{"valid map", "infrastructure", true, true, ""},
		{"missing test-only source", "infrastructure", false, true, "missing package " + GuardedWritePackage + ": add an entry to " + CodeMapSource},
		{"missing platform-only target", "infrastructure", true, false, "missing package " + target + ": add an entry to " + CodeMapSource},
		{"unknown source layer", "outer", true, true, CodeMapSource + ": entry 1 (" + GuardedWritePackage + "): unknown layer \"outer\""},
		{"empty source layer", "", true, true, CodeMapSource + ": entry 1 (" + GuardedWritePackage + "): unknown layer \"\""},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			const file = GuardedWritePackage + "/direction_test.go"
			writeLayerDirectionFixture(t, root, map[string]string{
				"README.md":                 "# Fixture\n",
				CodeMapSource:               layerDirectionFixtureMap(test.layer, test.source, test.destination),
				file:                        "package guardedwrite\n\nimport _ \"" + ModulePath + target + "\"\n",
				target + "/work_windows.go": "package projectwork\n",
			})
			got, err := CheckRepository(root)
			if test.problem != "" {
				want := "invalid code map:\n" + test.problem
				if err == nil || err.Error() != want || len(got) != 0 {
					t.Fatalf("got %v, %v; want no findings and error %q", got, err, want)
				}
				return
			}
			want := []Violation{{Edge{file, 3, GuardedWritePackage, target, true}, "product-layer direction forbids infrastructure -> application"}}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, %v; want %v", got, err, want)
			}
		})
	}
}

func layerDirectionFixtureMap(layer string, source, destination bool) string {
	var entries []string
	if source {
		entries = append(entries, fmt.Sprintf("  - {path: %s, layer: %q, purpose: Writes., doc: README.md}\n", GuardedWritePackage, layer))
	}
	if destination {
		entries = append(entries, "  - {path: src/internal/application/projectwork, layer: application, purpose: Works., doc: README.md}\n")
	}
	return "packages:\n" + strings.Join(entries, "")
}

func writeLayerDirectionFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
