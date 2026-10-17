package architecture

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestTargetDirectionMatrix(t *testing.T) {
	// Expectations name concrete consumers independently of the gate table.
	allowed := map[string]string{
		"internal/model":                   "",
		"internal/snapshot":                "",
		"internal/organization":            "internal/model internal/snapshot",
		"internal/realization":             "internal/model internal/snapshot",
		"internal/run":                     "internal/model internal/snapshot internal/organization internal/realization",
		"internal/apply":                   "internal/model internal/snapshot internal/organization internal/realization internal/run",
		"internal/adapters/cli":            "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply",
		"internal/adapters/mcp":            "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply",
		"internal/adapters/gitrepo":        "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply",
		"internal/adapters/journal":        "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply",
		"internal/adapters/codexappserver": "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply internal/adapters/process",
		"internal/adapters/checks":         "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply internal/adapters/process",
		"internal/adapters/process":        "",
		"internal/bootstrap":               "internal/model internal/snapshot internal/organization internal/realization internal/run internal/apply internal/adapters/cli internal/adapters/mcp internal/adapters/gitrepo internal/adapters/journal internal/adapters/codexappserver internal/adapters/checks internal/adapters/process",
		"cmd/markitect":                    "internal/bootstrap",
	}
	for from, targets := range allowed {
		for to := range allowed {
			for _, test := range []bool{false, true} {
				e := Edge{File: "direction.go", Line: 3, From: from, To: to, Test: test}
				want := contains(strings.Fields(targets), to) || from == to && test
				if got := Check([]Edge{e}); (len(got) == 0) != want {
					t.Errorf("%s -> %s test=%v: allowed=%v, got %v", from, to, test, want, got)
				}
			}
		}
		if got := Check([]Edge{{From: from, To: from + "/private"}}); len(got) != 0 {
			t.Errorf("own subtree %s: %v", from, got)
		}
		for _, old := range []string{"src/internal/core", "src/internal/application/projectapp", "src/internal/runtime/projectrun", "src/internal/infrastructure/source", "src/internal/tooling/architecture", "src/internal/testkit", "src/cmd/markitect", "src/harness/examples"} {
			for _, e := range []Edge{{From: from, To: old}, {From: old, To: from}, {From: from, To: old, Test: true}, {From: old, To: from, Test: true}} {
				if got := Check([]Edge{e}); len(got) != 1 {
					t.Errorf("cross-generation edge %+v: %v", e, got)
				}
			}
		}
	}
}

func TestTargetRepositoryScansProductionTestsAndPlatforms(t *testing.T) {
	for _, file := range []string{"bad.go", "bad_test.go", "bad_windows.go", "bad_linux.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeLayerDirectionFixture(t, root, map[string]string{
				"README.md":              "# Owner\n",
				CodeMapSource:            "packages:\n" + targetMapEntry("internal/model", "apply-model") + targetMapEntry("internal/run", "apply-run"),
				"internal/model/" + file: "package model\n\nimport _ \"" + ModulePath + "internal/run\"\n",
				"internal/run/run.go":    "package run\n",
			})
			got, err := CheckRepository(root)
			want := []Violation{{Edge{"internal/model/" + file, 3, "internal/model", "internal/run", strings.HasSuffix(file, "_test.go")}, "Apply responsibility direction forbids apply-model -> apply-run"}}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, %v; want %v", got, err, want)
			}
		})
	}
}

func TestTargetInventoryCannotBeBypassed(t *testing.T) {
	for _, test := range []struct{ name, pkg, file, mapText, problem string }{
		{"empty target areas", "", "", "packages: []\n", ""},
		{"missing test package", "internal/model/conformance", "only_test.go", "packages: []\n", "missing package internal/model/conformance"},
		{"missing platform package", "internal/snapshot", "only_windows.go", "packages: []\n", "missing package internal/snapshot"},
		{"future stub entry", "", "", "packages:\n" + targetMapEntry("internal/model", "apply-model"), "extra entry internal/model"},
		{"wrong current layer", "internal/model", "m.go", "packages:\n" + targetMapEntry("internal/model", "core"), "does not match import-gate layer"},
		{"wrong target layer", "internal/model", "m.go", "packages:\n" + targetMapEntry("internal/model", "apply-run"), "does not match import-gate layer"},
		{"unknown layer", "internal/model", "m.go", "packages:\n" + targetMapEntry("internal/model", "future"), "unknown layer"},
		{"unknown root", "internal/common", "x.go", "packages:\n" + targetMapEntry("internal/common", "apply-model"), "does not match import-gate layer"},
		{"unknown adapter", "internal/adapters/other", "x_test.go", "packages:\n" + targetMapEntry("internal/adapters/other", "apply-adapter"), "does not match import-gate layer"},
		{"unknown command", "cmd/other", "x_linux.go", "packages:\n" + targetMapEntry("cmd/other", "apply-command"), "does not match import-gate layer"},
		{"prefix collision", "internal/modelx", "x.go", "packages:\n" + targetMapEntry("internal/modelx", "apply-model"), "does not match import-gate layer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"README.md": "# Owner\n", CodeMapSource: test.mapText}
			if test.pkg != "" {
				files[test.pkg+"/"+test.file] = "package fixture\n"
			}
			writeLayerDirectionFixture(t, root, files)
			got, err := CheckRepository(root)
			if test.problem == "" {
				if err != nil || len(got) != 0 {
					t.Fatalf("empty target requires no stubs: %v %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.problem) || len(got) != 0 {
				t.Fatalf("got %v %v; want inventory error %q", got, err, test.problem)
			}
		})
	}
}

func targetMapEntry(pkg, layer string) string {
	return fmt.Sprintf("  - {path: %s, layer: %s, purpose: Fixture., doc: README.md}\n", pkg, layer)
}

func TestTargetCommandAndAdapterRepositoryEdges(t *testing.T) {
	for _, test := range []struct {
		from, to, fromLayer, toLayer string
		allowed                      bool
	}{
		{"cmd/markitect", "internal/bootstrap", "apply-command", "apply-bootstrap", true},
		{"cmd/markitect", "internal/model", "apply-command", "apply-model", false},
		{"internal/adapters/checks", "internal/adapters/process", "apply-adapter", "apply-process", true},
		{"internal/adapters/journal", "internal/adapters/process", "apply-adapter", "apply-process", false},
		{"internal/adapters/journal", "src/internal/core", "apply-adapter", "core", false},
		{"src/internal/application/projectapp", "internal/model", "application", "apply-model", false},
	} {
		t.Run(test.from+"_to_"+test.to, func(t *testing.T) {
			root := t.TempDir()
			m := CodeMap{Packages: []MappedPackage{
				{Path: test.from, Layer: test.fromLayer, Purpose: "Fixture.", Doc: "README.md"},
				{Path: test.to, Layer: test.toLayer, Purpose: "Fixture.", Doc: "README.md"},
			}}
			// Sort independently using the public layer ordering required by the map.
			if layerRank(test.fromLayer) > layerRank(test.toLayer) {
				m.Packages[0], m.Packages[1] = m.Packages[1], m.Packages[0]
			}
			mapText := "packages:\n"
			for _, p := range m.Packages {
				mapText += targetMapEntry(p.Path, p.Layer)
			}
			file := test.from + "/direction_windows.go"
			writeLayerDirectionFixture(t, root, map[string]string{
				"README.md": "# Owner\n", CodeMapSource: mapText,
				file:                         "package fixture\n\nimport _ \"" + ModulePath + test.to + "\"\n",
				test.to + "/fixture_test.go": "package fixture\n",
			})
			got, err := CheckRepository(root)
			if err != nil || (len(got) == 0) != test.allowed {
				t.Fatalf("allowed=%v: %v %v", test.allowed, got, err)
			}
		})
	}
}

func TestTargetTestsMayImportSharedTestkit(t *testing.T) {
	for _, from := range []string{"internal/model", "internal/adapters/gitrepo", "cmd/markitect"} {
		if got := Check([]Edge{{From: from, To: "internal/testkit", Test: true}}); len(got) != 0 {
			t.Errorf("test in %s importing shared testkit: %v", from, got)
		}
		if got := Check([]Edge{{From: from, To: "internal/testkit"}}); len(got) != 1 {
			t.Errorf("production code in %s importing shared testkit: %v", from, got)
		}
	}
	for _, to := range []string{"internal/model", "src/internal/core"} {
		if got := Check([]Edge{{From: "internal/testkit", To: to}}); len(got) != 1 {
			t.Errorf("shared testkit importing product package %s: %v", to, got)
		}
	}
}

func layerRank(name string) int {
	for i, l := range Layers {
		if l.Name == name {
			return i
		}
	}
	return -1
}

func TestTargetImportRequiresAnActualProducer(t *testing.T) {
	root := t.TempDir()
	writeLayerDirectionFixture(t, root, map[string]string{
		"README.md":           "# Owner\n",
		CodeMapSource:         "packages:\n" + targetMapEntry("internal/run", "apply-run"),
		"internal/run/run.go": "package run\n\nimport _ \"" + ModulePath + "internal/model\"\n",
	})
	got, err := CheckRepository(root)
	if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "target import internal/model has no actual package") {
		t.Fatalf("absent producer: %v %v", got, err)
	}
}

func TestTargetSelfImportRequiresExternalTestPackage(t *testing.T) {
	for _, pkg := range []string{"model", "model_test"} {
		t.Run(pkg, func(t *testing.T) {
			root := t.TempDir()
			writeLayerDirectionFixture(t, root, map[string]string{
				"README.md":                    "# Owner\n",
				CodeMapSource:                  "packages:\n" + targetMapEntry("internal/model", "apply-model"),
				"internal/model/model_test.go": "package " + pkg + "\n\nimport _ \"" + ModulePath + "internal/model\"\n",
			})
			got, err := CheckRepository(root)
			if err != nil || (len(got) == 0) != (pkg == "model_test") {
				t.Fatalf("package %s: %v %v", pkg, got, err)
			}
		})
	}
}

func TestTargetNestedModuleCannotHidePackage(t *testing.T) {
	for _, dir := range []string{"internal/model/hidden", "cmd/markitect/hidden", "internal/unclassified"} {
		t.Run(dir, func(t *testing.T) {
			root := t.TempDir()
			writeLayerDirectionFixture(t, root, map[string]string{
				"go.mod":           "module github.com/Glacius-Labs/Markitect\n",
				CodeMapSource:      "packages: []\n",
				dir + "/go.mod":    "module example.org/hidden\n",
				dir + "/hidden.go": "package hidden\n",
			})
			got, err := CheckRepository(root)
			if len(got) != 0 || err == nil || !strings.Contains(err.Error(), "nested Go module "+dir+"/go.mod") {
				t.Fatalf("nested product module escaped inventory: %v %v", got, err)
			}
		})
	}
}

func TestTargetNestedModuleRulePreservesFixtureBoundaries(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"go.mod": "module github.com/Glacius-Labs/Markitect\n", CodeMapSource: "packages: []\n"}
	for _, dir := range []string{"internal/model/testdata/fixture", "cmd/markitect/testdata/fixture", "internal/model/_fixtures", "internal/model/.hidden", "internal/model/vendor/fixture", "internal/model/node_modules/fixture"} {
		files[dir+"/go.mod"] = "module example.org/fixture\n"
		files[dir+"/fixture.go"] = "package fixture\n"
	}
	writeLayerDirectionFixture(t, root, files)
	// Inventory follows Go's hidden-directory rules. Repository inspection
	// separately retains its existing rules for hidden source and fixtures.
	got, err := Packages(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("excluded fixture inventory: %v %v", got, err)
	}
	fixtureRoot := t.TempDir()
	writeLayerDirectionFixture(t, fixtureRoot, map[string]string{
		CodeMapSource:                                "packages: []\n",
		"internal/model/testdata/fixture/go.mod":     "module example.org/fixture\n",
		"internal/model/testdata/fixture/fixture.go": "package fixture\n",
	})
	violations, err := CheckRepository(fixtureRoot)
	if err != nil || len(violations) != 0 {
		t.Fatalf("testdata boundary: %v %v", violations, err)
	}
}
