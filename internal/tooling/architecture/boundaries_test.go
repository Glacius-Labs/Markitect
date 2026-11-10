package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// fixtureRules is the real table restricted to the named packages, each owned
// by the fixture's README.md, with the real exceptions between them.
func fixtureRules(t *testing.T, paths ...string) Rules {
	t.Helper()
	r := Rules{Layers: Table.Layers}
	for _, p := range Table.Packages {
		if contains(paths, p.Path) {
			p.Doc = "README.md"
			r.Packages = append(r.Packages, p)
		}
	}
	if len(r.Packages) != len(paths) {
		t.Fatalf("fixture names a package outside the table: %v", paths)
	}
	for _, x := range Table.Exceptions {
		if contains(paths, x.From) && contains(paths, x.To) {
			r.Exceptions = append(r.Exceptions, x)
		}
	}
	return r
}

func TestRepositoryArchitecture(t *testing.T) {
	violations, err := Table.CheckRepository(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Error(v)
	}
}

func TestTableIsValid(t *testing.T) {
	for _, problem := range Table.Validate() {
		t.Error(problem)
	}
}

func TestLayerDirections(t *testing.T) {
	// These examples state the important directions independently of the
	// table's rows: foundations never import outward, adapters share only the
	// container client, entrypoints stay thin and maintenance stays apart.
	const (
		both = iota
		none
		testOnly
	)
	for _, tc := range []struct {
		from, to string
		allowed  int
	}{
		{"internal/realization", "internal/spec", both},
		{"internal/project", "internal/realization", both},
		{"internal/project", "internal/libraries", both},
		{"internal/analysis", "internal/project", both},
		{"internal/adapters/cli", "internal/analysis", both},
		{"internal/adapters/cli", "internal/skills", both},
		{"internal/adapters/cli", "internal/buildinfo", both},
		{"internal/adapters/folder", "internal/project", both},
		{"internal/bootstrap", "internal/adapters/cli", both},
		{"internal/bootstrap", "internal/adapters/folder", both},
		{"cmd/markitect", "internal/bootstrap", both},

		// Foundations never import outward.
		{"internal/spec", "internal/realization", none},
		{"internal/spec", "internal/libraries", none},
		{"internal/spec", "internal/bootstrap", none},
		{"internal/libraries", "internal/spec", none},
		{"internal/skills", "internal/spec", none},
		{"internal/realization", "internal/project", none},
		{"internal/project", "internal/analysis", none},
		{"internal/analysis", "internal/adapters/folder", none},
		{"internal/analysis", "internal/adapters/cli", none},
		// Adapters do not import each other, nor the agent side.
		{"internal/adapters/cli", "internal/adapters/folder", none},
		{"internal/adapters/folder", "internal/adapters/cli", none},
		// Entrypoints stay thin.
		{"cmd/markitect", "internal/adapters/cli", none},
		{"cmd/markitect", "internal/spec", none},
		// Maintenance stays apart from the product.
		{"internal/tooling/architecture", "internal/spec", none},
		{"tools/smoke", "internal/spec", none},
		{"tools/smoke", "internal/bootstrap", none},

		// Test composition only.
		{"internal/run", "internal/adapters/folder", testOnly},
		{"internal/spec", "internal/testkit", testOnly},
		{"internal/adapters/folder", "internal/testkit", testOnly},
	} {
		for _, test := range []bool{false, true} {
			allowed := tc.allowed == both || tc.allowed == testOnly && test
			got := Table.Check([]Edge{{From: tc.from, To: tc.to, Test: test}})
			if (len(got) == 0) != allowed {
				t.Errorf("%s -> %s test=%v: %v", tc.from, tc.to, test, got)
			}
		}
	}
}

func TestUnknownPackagesHaveNoPrivilege(t *testing.T) {
	for _, unknown := range []string{"src/internal/core", "src/cmd/markitect", "benchmark/runner", "internal/common", "internal/adapters/other", "internal/tooling/other", "internal/specx", "internal/agentd/frame/sub", "internal/adapters/folder/sub"} {
		for _, edge := range []Edge{{From: unknown}, {From: "internal/spec", To: unknown}, {From: unknown, To: "internal/testkit", Test: true}, {From: "internal/bootstrap", To: unknown, Test: true}} {
			got := Table.Check([]Edge{edge})
			if len(got) != 1 || !strings.Contains(got[0].Rule, "unclassified package") {
				t.Errorf("unclassified %v: %v", edge, got)
			}
		}
	}
}

func TestForbiddenLayerImportFailsInEveryFile(t *testing.T) {
	for _, file := range []string{"bad.go", "bad_test.go", "bad_windows.go", "bad_linux.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, map[string]string{
				"README.md":                     "# Owner\n",
				"internal/spec/" + file:         "package spec\nimport _ \"" + ModulePath + "internal/analysis\"\n",
				"internal/analysis/producer.go": "package analysis\n",
			})
			got, err := fixtureRules(t, "internal/spec", "internal/analysis").CheckRepository(root)
			if err != nil || len(got) != 1 || got[0].Edge.File != "internal/spec/"+file || got[0].Edge.Line != 2 || got[0].Rule != "layer spec may not import layer service" {
				t.Fatalf("%v %v", got, err)
			}
		})
	}
	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		"README.md":                     "# Owner\n",
		"internal/adapters/folder/f.go": "package folder\nimport _ \"" + ModulePath + "internal/adapters/cli\"\n",
		"internal/adapters/cli/c.go":    "package cli\n",
	})
	got, err := fixtureRules(t, "internal/adapters/folder", "internal/adapters/cli").CheckRepository(root)
	if err != nil || len(got) != 1 || got[0].Rule != "layer adapter may not import layer adapter" {
		t.Fatalf("adapter imported a sibling adapter: %v %v", got, err)
	}
}

func TestTestkitOnlyInTests(t *testing.T) {
	for _, tc := range []struct {
		file, content string
		want          string
	}{
		{"internal/spec/m.go", "package spec\nimport _ \"" + ModulePath + "internal/testkit\"\n", "only test files may import internal/testkit"},
		{"internal/spec/m_windows.go", "package spec\nimport _ \"" + ModulePath + "internal/testkit\"\n", "only test files may import internal/testkit"},
		{"internal/spec/m_test.go", "package spec\nimport _ \"" + ModulePath + "internal/testkit\"\n", ""},
		{"internal/testkit/k.go", "package testkit\nimport _ \"" + ModulePath + "internal/spec\"\n", "layer test-support may not import Markitect packages"},
		{"internal/testkit/k_test.go", "package testkit\nimport _ \"" + ModulePath + "internal/spec\"\n", "layer test-support may not import Markitect packages"},
		{"tools/smoke/b_test.go", "package main\nimport _ \"" + ModulePath + "internal/testkit\"\n", "layer standalone may not import Markitect packages"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, map[string]string{
				"README.md":                    "# Owner\n",
				"internal/spec/producer.go":    "package spec\n",
				"internal/testkit/producer.go": "package testkit\n",
				"tools/smoke/main.go":          "package main\n",
				tc.file:                        tc.content,
			})
			got, err := fixtureRules(t, "internal/spec", "internal/testkit", "tools/smoke").CheckRepository(root)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatal(got)
				}
			} else if len(got) != 1 || got[0].Rule != tc.want {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}

func TestUnknownPackageFails(t *testing.T) {
	// A Go package without a row is a repository error, whatever it imports.
	for _, file := range []string{"internal/extra/x.go", "internal/extra/x_test.go", "internal/spec/sub/x_windows.go", "src/internal/core/c.go", "tools/other/o.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, map[string]string{"README.md": "# Owner\n", "internal/spec/m.go": "package spec\n", file: "package extra\n"})
			if _, err := fixtureRules(t, "internal/spec").CheckRepository(root); err == nil || !strings.Contains(err.Error(), "missing package "+filepath.ToSlash(filepath.Dir(file))) {
				t.Fatalf("unknown package accepted: %v", err)
			}
		})
	}
	// An import of a package the table does not classify is a violation.
	root := t.TempDir()
	writeFixture(t, root, map[string]string{"README.md": "# Owner\n", "internal/realization/o.go": "package realization\nimport _ \"" + ModulePath + "internal/spec\"\n"})
	got, err := fixtureRules(t, "internal/realization").CheckRepository(root)
	if err != nil || len(got) != 1 || !strings.Contains(got[0].Rule, "unclassified package") {
		t.Fatalf("import of an unclassified package accepted: %v %v", got, err)
	}
}

func TestBootstrapCompositionAcrossPlatforms(t *testing.T) {
	for _, producer := range []struct{ path, pkg string }{
		{"internal/project", "project"},
		{"internal/adapters/cli", "cli"},
		{"internal/adapters/folder", "folder"},
		{"internal/spec", "spec"},
	} {
		for _, file := range []string{"fixture_windows_test.go", "fixture_linux_test.go", "fixture_windows.go", "fixture_linux.go"} {
			t.Run(producer.pkg+"/"+file, func(t *testing.T) {
				root := t.TempDir()
				writeFixture(t, root, map[string]string{
					"README.md":                    "# Owner\n",
					producer.path + "/producer.go": "package " + producer.pkg + "\n",
					"internal/bootstrap/" + file:   "package bootstrap\nimport _ \"" + ModulePath + producer.path + "\"\n",
				})
				got, err := fixtureRules(t, producer.path, "internal/bootstrap").CheckRepository(root)
				if err != nil || len(got) != 0 {
					t.Fatalf("bootstrap composition dependency refused: %v %v", got, err)
				}
			})
		}
	}
}

func TestInventoryRequiresActualPackagesAndOwners(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		rules Rules
		want  string
	}{
		{"empty", nil, Rules{Layers: Table.Layers}, ""},
		{"unmapped test", map[string]string{"internal/spec/only_test.go": "package spec\n"}, Rules{Layers: Table.Layers}, "missing package internal/spec"},
		{"unmapped platform", map[string]string{"internal/snapshot/only_windows.go": "package snapshot\n"}, Rules{Layers: Table.Layers}, "missing package internal/snapshot"},
		{"row without package", nil, fixtureRules(t, "internal/spec"), "extra row internal/spec"},
		{"absent owner", map[string]string{"internal/spec/m.go": "package spec\n"}, Rules{Layers: Table.Layers, Packages: []Package{{"internal/spec", "spec", "docs/absent.md"}}}, "owning document: docs/absent.md does not exist"},
		{"unknown layer", map[string]string{"internal/spec/m.go": "package spec\n"}, Rules{Layers: Table.Layers, Packages: []Package{{"internal/spec", "future", "README.md"}}}, `unknown layer "future"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"README.md": "# Owner\n"}
			for name, content := range tc.files {
				files[name] = content
			}
			writeFixture(t, root, files)
			v, err := tc.rules.CheckRepository(root)
			if tc.want == "" {
				if err != nil || len(v) != 0 {
					t.Fatalf("%v %v", v, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%v %v, want %s", v, err, tc.want)
			}
		})
	}
}

func TestValidateReportsEveryTableProblem(t *testing.T) {
	bad := Rules{
		Layers: []Layer{
			{Name: "low", Title: "Low", Meaning: "Low."},
			{Name: "high", Title: "High", Meaning: "High.", Imports: []string{"low", "low", "missing"}},
			{Name: "low", Title: "Again", Meaning: "a | b"},
			{Name: "Bad_Name", Title: "Bad", Meaning: "Bad.", Standalone: true, Imports: []string{"low"}},
			{Name: "fixtures", Title: "Fixtures", Meaning: "Fixtures.", TestOnly: true},
			{Name: "user", Title: "User", Meaning: "User.", Imports: []string{"fixtures"}},
			{Name: "upward", Title: "Upward", Meaning: "Upward.", Imports: []string{"later"}},
			{Name: "later", Title: "Later", Meaning: "Later."},
		},
		Packages: []Package{
			{"internal/high", "high", "README.md"},
			{"internal/low", "low", "README.md"},
			{"internal/low", "low", ""},
			{"../outside", "nowhere", "README.md"},
		},
		Exceptions: []Exception{
			{From: "internal/high", To: "internal/low", Reason: "Already allowed."},
			{From: "internal/low", To: "internal/gone", Reason: "Unknown."},
			{From: "internal/low", To: "internal/low", Reason: "Self."},
			{From: "internal/low", To: "internal/high", Reason: ""},
			{From: "internal/low", To: "internal/high", Reason: "Twice."},
		},
	}
	text := strings.Join(bad.Validate(), "\n")
	for _, want := range []string{
		`layer "Bad_Name": name must be lower-case words joined by hyphens`,
		`layer "low": duplicate`,
		`layer "low": title and meaning must be single non-empty lines`,
		`layer "Bad_Name": a standalone layer imports nothing`,
		`layer "high": imports layer "low" twice`,
		`layer "high": imports unknown layer "missing"`,
		`layer "user": test-only layer "fixtures" needs no import rule`,
		`layer "upward": may import only itself and layers listed before it, not "later"`,
		`package "internal/low": packages must be ordered by layer, then by path`,
		`package "internal/low": duplicate`,
		`package "internal/low": owning document is missing`,
		`package "../outside": path must be a clean repository-relative path`,
		`package "../outside": unknown layer "nowhere"`,
		`exception internal/high -> internal/low: layer high already may import layer low`,
		`exception internal/low -> internal/gone: names an unknown package`,
		`exception internal/low -> internal/low: a package may not except itself`,
		`exception internal/low -> internal/high: reason must be a single non-empty line`,
		`exception internal/low -> internal/high: duplicate`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing problem %q in:\n%s", want, text)
		}
	}
}

func TestSelfImportsRequireExternalTests(t *testing.T) {
	for _, tc := range []struct {
		file, pkg string
		want      int
	}{{"p.go", "spec", 1}, {"p_test.go", "spec", 1}, {"p_test.go", "spec_test", 0}} {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{"README.md": "# Owner\n", "internal/spec/" + tc.file: "package " + tc.pkg + "\nimport _ \"" + ModulePath + "internal/spec\"\n"})
		v, err := fixtureRules(t, "internal/spec").CheckRepository(root)
		if err != nil || len(v) != tc.want {
			t.Fatalf("%+v: %v %v", tc, v, err)
		}
	}
}

func TestProductNestedModulesCannotHidePackages(t *testing.T) {
	for _, name := range []string{"internal/spec/go.mod", "cmd/markitect/go.mod", "internal/tooling/architecture/go.mod"} {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{name: "module hidden\n"})
		if _, err := Packages(root); err == nil || !strings.Contains(err.Error(), "nested Go module") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	root := t.TempDir()
	writeFixture(t, root, map[string]string{"internal/spec/testdata/nested/go.mod": "module fixture\n", "internal/spec/testdata/nested/a.go": "package fixture\n"})
	if p, err := Packages(root); err != nil || len(p) != 0 {
		t.Fatalf("fixture boundary: %v %v", p, err)
	}
}

func TestInspectSkipsNestedModulesOutsideTheProduct(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		"tools/reference/go.mod":          "module reference\n",
		"tools/reference/cmd/x/main.go":   "package main\nimport _ \"reference/internal/y\"\n",
		"tools/reference/internal/y/y.go": "package y\n",
		"tools/harness/h.go":              "package harness\n",
		"internal/spec/nested/go.mod":     "module hidden\n",
		"internal/spec/nested/hidden.go":  "package hidden\n",
	})
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range edges {
		seen[e.From] = true
	}
	if seen["tools/reference/cmd/x"] || seen["tools/reference/internal/y"] || !seen["tools/harness"] || !seen["internal/spec/nested"] {
		t.Fatalf("inspected packages: %v", seen)
	}
}
