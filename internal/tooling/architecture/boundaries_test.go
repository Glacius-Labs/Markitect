package architecture

import (
	"fmt"
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

func mapEntry(pkg, group string) string {
	return fmt.Sprintf("  - {path: %s, layer: %s, purpose: Fixture., doc: README.md}\n", pkg, group)
}

func TestRepositoryArchitecture(t *testing.T) {
	violations, err := CheckRepository(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Error(v)
	}
}

func TestConcreteResponsibilityBoundaries(t *testing.T) {
	// These consumer examples specify the important boundaries independently
	// of the gate table. Maintenance never receives a general tooling privilege.
	for _, tc := range []struct {
		from, to string
		allowed  bool
	}{
		{"internal/model", "internal/snapshot", false},
		{"internal/snapshot", "internal/model", false},
		{"internal/organization", "internal/model", true},
		{"internal/realization", "internal/snapshot", true},
		{"internal/run", "internal/model", true},
		{"internal/run", "internal/snapshot", true},
		{"internal/run", "internal/organization", true},
		{"internal/run", "internal/realization", true},
		{"internal/run", "internal/apply", false},
		{"internal/run", "internal/adapters/gitrepo", false},
		{"internal/run", "internal/adapters/claudecode", false},
		{"internal/model", "internal/run", false},
		{"internal/apply", "internal/organization", true},
		{"internal/apply", "internal/run", true},
		{"internal/apply", "internal/adapters/gitrepo", false},
		{"internal/modeldocs", "internal/apply", true},
		{"internal/modeldocs", "internal/adapters/documents", false},
		{"internal/adapters/cli", "internal/modeldocs", true},
		{"internal/adapters/cli", "internal/tooling/licenses", true},
		{"internal/adapters/cli", "internal/tooling/releasecli", false},
		{"internal/adapters/cli", "internal/adapters/gitrepo", false},
		{"internal/adapters/documents", "internal/tooling/guardedwrite", true},
		{"internal/adapters/documents", "internal/tooling/source", false},
		{"internal/adapters/checks", "internal/adapters/codexappserver", false},
		// Claude uses the Run session port and the frame DTOs exposed by the
		// container API; agentd execution and other adapters remain separate.
		{"internal/adapters/claudecode", "internal/run", true},
		{"internal/adapters/claudecode", "internal/adapters/container", true},
		{"internal/adapters/claudecode", "internal/agentd/frame", true},
		{"internal/adapters/claudecode", "internal/agentd", false},
		{"internal/adapters/claudecode", "internal/adapters/codexappserver", false},
		{"internal/adapters/claudecode", "internal/adapters/gitrepo", false},
		{"internal/adapters/claudecode", "internal/tooling/source", false},
		// Codex uses the Run session port and the frame DTOs exposed by the
		// container API; agentd execution and other adapters remain separate.
		{"internal/adapters/codexappserver", "internal/adapters/container", true},
		{"internal/adapters/codexappserver", "internal/run", true},
		{"internal/adapters/codexappserver", "internal/agentd/frame", true},
		{"internal/adapters/codexappserver", "internal/agentd", false},
		{"internal/adapters/codexappserver", "internal/adapters/claudecode", false},
		{"internal/adapters/codexappserver", "internal/adapters/gitrepo", false},
		{"internal/adapters/codexappserver", "internal/tooling/source", false},
		{"internal/bootstrap", "internal/adapters/container", false},
		{"internal/adapters/container", "internal/agentd/frame", true},
		{"internal/adapters/container", "internal/agentd", false},
		{"internal/adapters/container", "internal/apply", false},
		{"internal/agentd", "internal/agentd/frame", true},
		{"internal/agentd", "internal/adapters/container", false},
		{"internal/agentd/frame", "internal/model", false},
		{"cmd/markitect-agentd", "internal/agentd", true},
		{"cmd/markitect-agentd", "internal/bootstrap", false},
		{"internal/bootstrap", "internal/adapters/gitrepo", true},
		{"internal/bootstrap", "internal/tooling/release", false},
		{"cmd/markitect", "internal/bootstrap", true},
		{"cmd/markitect", "internal/apply", false},
		{"cmd/markitect-release", "internal/tooling/releasecli", true},
		{"cmd/markitect-release", "internal/bootstrap", false},
		{"internal/tooling/release", "internal/tooling/source", true},
		{"internal/tooling/release", "internal/apply", false},
		{"internal/tooling/modulechecks", "internal/apply", true},
		{"internal/tooling/modulechecks", "internal/adapters/checks", false},
		{"internal/tooling/modules/githooks", "internal/tooling/modules/pipelines", false},
		{"internal/tooling/release/install", "internal/tooling/release", true},
		{"internal/tooling/source", "internal/model", false},
		{"integration", "internal/model", false},
		{"tools/smoke", "internal/bootstrap", false},
	} {
		for _, test := range []bool{false, true} {
			got := Check([]Edge{{From: tc.from, To: tc.to, Test: test}})
			if (len(got) == 0) != tc.allowed {
				t.Errorf("%s -> %s test=%v: %v", tc.from, tc.to, test, got)
			}
		}
	}
	for _, removed := range []string{"src/internal/core", "src/internal/runtime/projectrun", "src/cmd/markitect", "internal/common", "internal/adapters/other", "internal/tooling/other", "internal/modelx"} {
		for _, edge := range []Edge{{From: removed}, {From: "internal/model", To: removed}, {From: removed, To: "internal/testkit", Test: true}} {
			if got := Check([]Edge{edge}); len(got) != 1 {
				t.Errorf("unclassified %v: %v", edge, got)
			}
		}
	}
}

func TestTestsAndPlatformsCannotBypassDirection(t *testing.T) {
	for _, file := range []string{"bad.go", "bad_test.go", "bad_windows.go", "bad_linux.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, map[string]string{
				"README.md": "# Owner\n", CodeMapSource: "packages:\n" + mapEntry("internal/model", "apply-model") + mapEntry("internal/apply", "apply-usecase"),
				"internal/model/" + file:     "package model\nimport _ \"" + ModulePath + "internal/apply\"\n",
				"internal/apply/producer.go": "package apply\n",
			})
			got, err := CheckRepository(root)
			if err != nil || len(got) != 1 || got[0].Edge.File != "internal/model/"+file || got[0].Edge.Line != 2 || !strings.Contains(got[0].Rule, "responsibility direction forbids") {
				t.Fatalf("%v %v", got, err)
			}
		})
	}
}

func TestJournalFixtureEdgesStayTestOnly(t *testing.T) {
	for _, tc := range []struct {
		to               string
		production, test bool
	}{
		{"internal/apply", true, true}, {"internal/run", true, true},
		{"internal/model", false, true}, {"internal/snapshot", false, true}, {"internal/testkit", false, true},
		{"internal/adapters/gitrepo", false, false}, {"internal/adapters/container", false, false},
		{"internal/organization", false, false}, {"internal/adapters/unknown", false, false},
	} {
		for _, test := range []bool{false, true} {
			allowed := tc.production
			if test {
				allowed = tc.test
			}
			if got := Check([]Edge{{From: "internal/adapters/journal", To: tc.to, Test: test}}); (len(got) == 0) != allowed {
				t.Errorf("journal -> %s test=%v: %v", tc.to, test, got)
			}
		}
	}
	for _, file := range []string{"fixture_windows_test.go", "fixture_linux_test.go", "fixture_windows.go", "fixture_linux.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			writeFixture(t, root, map[string]string{
				"README.md": "# Owner\n", CodeMapSource: "packages:\n" + mapEntry("internal/model", "apply-model") + mapEntry("internal/adapters/journal", "apply-adapter"),
				"internal/model/model.go":           "package model\n",
				"internal/adapters/journal/" + file: "package journal\nimport _ \"" + ModulePath + "internal/model\"\n",
			})
			got, err := CheckRepository(root)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if !strings.HasSuffix(file, "_test.go") {
				want = 1
			}
			if len(got) != want {
				t.Fatal("platform classification bypassed test-only boundary", got)
			}
		})
	}
}

func TestBootstrapControlFixtureEdgesStayTestOnly(t *testing.T) {
	for _, tc := range []struct {
		to               string
		production, test bool
	}{
		{"internal/apply", true, true}, {"internal/adapters/gitrepo", true, true},
		{"internal/adapters/journal", false, true}, {"internal/model", false, true},
		{"internal/snapshot", false, true}, {"internal/run", false, true}, {"internal/testkit", false, true},
		{"internal/organization", false, false}, {"internal/realization", false, false},
		{"internal/adapters/container", false, false}, {"internal/adapters/unknown", false, false},
	} {
		for _, test := range []bool{false, true} {
			allowed := tc.production
			if test {
				allowed = tc.test
			}
			if got := Check([]Edge{{From: "internal/bootstrap", To: tc.to, Test: test}}); (len(got) == 0) != allowed {
				t.Errorf("bootstrap -> %s test=%v: %v", tc.to, test, got)
			}
		}
	}
	for _, producer := range []struct{ path, layer, pkg string }{
		{"internal/adapters/journal", "apply-adapter", "journal"},
		{"internal/model", "apply-model", "model"},
		{"internal/snapshot", "apply-snapshot", "snapshot"},
		{"internal/run", "apply-run", "run"},
	} {
		for _, file := range []string{"fixture_windows_test.go", "fixture_linux_test.go", "fixture_windows.go", "fixture_linux.go"} {
			t.Run(producer.pkg+"/"+file, func(t *testing.T) {
				root := t.TempDir()
				entries := mapEntry(producer.path, producer.layer) + mapEntry("internal/bootstrap", "apply-bootstrap")
				writeFixture(t, root, map[string]string{
					"README.md": "# Owner\n", CodeMapSource: "packages:\n" + entries,
					producer.path + "/producer.go": "package " + producer.pkg + "\n",
					"internal/bootstrap/" + file:   "package bootstrap\nimport _ \"" + ModulePath + producer.path + "\"\n",
				})
				got, err := CheckRepository(root)
				if err != nil {
					t.Fatal(err)
				}
				want := 0
				if !strings.HasSuffix(file, "_test.go") {
					want = 1
				}
				if len(got) != want {
					t.Fatal("platform classification bypassed bootstrap test-only boundary", got)
				}
			})
		}
	}
}

func TestInventoryRequiresActualOwnersAndProducers(t *testing.T) {
	for _, tc := range []struct{ name, pkg, file, mapText, want string }{
		{"empty", "", "", "packages: []\n", ""},
		{"unmapped test", "internal/model", "only_test.go", "packages: []\n", "missing package internal/model"},
		{"unmapped platform", "internal/snapshot", "only_windows.go", "packages: []\n", "missing package internal/snapshot"},
		{"stub", "", "", "packages:\n" + mapEntry("internal/model", "apply-model"), "extra entry internal/model"},
		{"wrong owner", "internal/model", "m.go", "packages:\n" + mapEntry("internal/model", "apply-usecase"), "does not match import-gate layer"},
		{"unowned tooling", "internal/tooling/other", "x.go", "packages:\n" + mapEntry("internal/tooling/other", "maintenance"), "does not match import-gate layer"},
		{"unknown layer", "internal/model", "m.go", "packages:\n" + mapEntry("internal/model", "future"), "unknown layer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"README.md": "# Owner\n", CodeMapSource: tc.mapText}
			if tc.pkg != "" {
				files[tc.pkg+"/"+tc.file] = "package fixture\n"
			}
			writeFixture(t, root, files)
			v, err := CheckRepository(root)
			if tc.want == "" {
				if err != nil || len(v) != 0 {
					t.Fatalf("%v %v", v, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%v %v, want %s", v, err, tc.want)
			}
		})
	}
	root := t.TempDir()
	writeFixture(t, root, map[string]string{"README.md": "# Owner\n", CodeMapSource: "packages:\n" + mapEntry("internal/organization", "apply-organization"), "internal/organization/o.go": "package organization\nimport _ \"" + ModulePath + "internal/model\"\n"})
	if _, err := CheckRepository(root); err == nil || !strings.Contains(err.Error(), "has no actual package") {
		t.Fatalf("missing producer accepted: %v", err)
	}
}

func TestSelfImportsRequireExternalTestsAndTestkitStaysStandalone(t *testing.T) {
	for _, tc := range []struct {
		file, pkg string
		want      int
	}{{"p.go", "model", 1}, {"p_test.go", "model", 1}, {"p_test.go", "model_test", 0}} {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{"README.md": "# Owner\n", CodeMapSource: "packages:\n" + mapEntry("internal/model", "apply-model"), "internal/model/" + tc.file: "package " + tc.pkg + "\nimport _ \"" + ModulePath + "internal/model\"\n"})
		v, err := CheckRepository(root)
		if err != nil || len(v) != tc.want {
			t.Fatalf("%+v: %v %v", tc, v, err)
		}
	}
	for _, edge := range []Edge{{From: "internal/model", To: "internal/testkit"}, {From: "internal/testkit", To: "internal/model", Test: true}, {From: "integration", To: "internal/testkit", Test: true}} {
		if len(Check([]Edge{edge})) != 1 {
			t.Errorf("forbidden fixture edge %v", edge)
		}
	}
	if v := Check([]Edge{{From: "internal/model", To: "internal/testkit", Test: true}}); len(v) != 0 {
		t.Fatal(v)
	}
}

func TestDocumentWriterMayOnlyUseReviewedGuardedAPI(t *testing.T) {
	for _, tc := range []struct {
		alias, expr string
		want        int
	}{{"gw", "gw.OpenRoot(\".\")", 0}, {"gw", "gw.CaptureFiles(\".\", nil)", 0}, {"gw", "gw.UnreviewedPrimitive()", 1}, {".", "OpenRoot(\".\")", 1}} {
		root := t.TempDir()
		name := "internal/adapters/documents/w.go"
		writeFixture(t, root, map[string]string{name: "package documents\nimport " + tc.alias + " \"" + ModulePath + GuardedWritePackage + "\"\nfunc f(){" + tc.expr + "}\n"})
		v, err := CheckGuardedWriteUse(root, []Edge{{From: "internal/adapters/documents", To: GuardedWritePackage, File: name}}, CodeMap{})
		if err != nil || len(v) != tc.want {
			t.Fatalf("%+v: %v %v", tc, v, err)
		}
	}
}

func TestProductNestedModulesCannotHidePackages(t *testing.T) {
	for _, name := range []string{"internal/model/go.mod", "cmd/markitect/go.mod", "internal/tooling/source/go.mod"} {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{name: "module hidden\n"})
		if _, err := Packages(root); err == nil || !strings.Contains(err.Error(), "nested Go module") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	root := t.TempDir()
	writeFixture(t, root, map[string]string{"internal/model/testdata/nested/go.mod": "module fixture\n", "internal/model/testdata/nested/a.go": "package fixture\n"})
	if p, err := Packages(root); err != nil || len(p) != 0 {
		t.Fatalf("fixture boundary: %v %v", p, err)
	}
}
