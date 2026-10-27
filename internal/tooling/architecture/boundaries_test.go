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
		{"internal/run", "internal/model", false},
		{"internal/run", "internal/snapshot", true},
		{"internal/run", "internal/organization", true},
		{"internal/run", "internal/realization", false},
		{"internal/run", "internal/apply", false},
		{"internal/run", "internal/adapters/gitrepo", false},
		{"internal/run", "internal/adapters/checks", false},
		{"internal/model", "internal/run", false},
		{"internal/apply", "internal/organization", false},
		{"internal/apply", "internal/run", true},
		{"internal/apply", "internal/workflow", true},
		{"internal/workflow", "internal/run", true},
		{"internal/workflow", "internal/adapters/gitrepo", false},
		{"internal/workflow", "internal/adapters/runstore", false},
		{"internal/apply", "internal/adapters/gitrepo", false},
		{"internal/modeldocs", "internal/project", true},
		{"internal/modeldocs", "internal/adapters/documents", false},
		{"internal/adapters/cli", "internal/modeldocs", true},
		{"internal/adapters/cli", "internal/tooling/licenses", true},
		{"internal/adapters/cli", "internal/tooling/releasecli", false},
		{"internal/adapters/cli", "internal/adapters/gitrepo", false},
		{"internal/adapters/documents", "internal/tooling/guardedwrite", true},
		{"internal/adapters/documents", "internal/tooling/source", false},
		{"internal/adapters/checks", "internal/adapters/codexappserver", false},
		// Workspace and check execution consume Run ports and the container
		// API without acquiring provider, frame or sibling adapter privileges.
		{"internal/adapters/checks", "internal/run", true},
		{"internal/adapters/checks", "internal/adapters/container", true},
		{"internal/adapters/checks", "internal/agentd/frame", false},
		{"internal/adapters/checks", "internal/tooling/source", false},
		{"internal/adapters/gitrepo", "internal/run", true},
		{"internal/adapters/gitrepo", "internal/adapters/container", true},
		{"internal/adapters/gitrepo", "internal/agentd/frame", false},
		{"internal/adapters/gitrepo", "internal/adapters/checks", false},
		{"internal/adapters/gitrepo", "internal/adapters/codexappserver", false},
		{"internal/adapters/gitrepo", "internal/tooling/source", false},
		// Codex uses the Run session port and the frame DTOs exposed by the
		// container API; agentd execution and other adapters remain separate.
		{"internal/adapters/codexappserver", "internal/adapters/container", true},
		{"internal/adapters/codexappserver", "internal/run", true},
		{"internal/adapters/codexappserver", "internal/harness", true},
		{"internal/adapters/codexappserver", "internal/project", true},
		{"internal/adapters/codexappserver", "internal/runtimeconfig", true},
		{"internal/adapters/codexappserver", "internal/apply", false},
		{"internal/harness", "internal/project", true},
		{"internal/harness", "internal/run", false},
		{"internal/harness", "internal/adapters/codexappserver", false},
		{"internal/adapters/codexappserver", "internal/agentd/frame", true},
		{"internal/adapters/codexappserver", "internal/agentd", false},
		{"internal/adapters/codexappserver", "internal/adapters/gitrepo", false},
		{"internal/adapters/codexappserver", "internal/adapters/runstore", false},
		{"internal/adapters/codexappserver", "internal/tooling/source", false},
		{"internal/bootstrap", "internal/adapters/container", true},
		{"internal/bootstrap", "internal/model", true},
		{"internal/bootstrap", "internal/snapshot", false},
		{"internal/model", "internal/bootstrap", false},
		{"internal/snapshot", "internal/bootstrap", false},
		{"internal/adapters/container", "internal/agentd/frame", true},
		{"internal/adapters/container", "internal/agentd", false},
		{"internal/adapters/container", "internal/apply", false},
		{"internal/agentd", "internal/agentd/frame", true},
		{"internal/agentd", "internal/adapters/container", false},
		{"internal/agentd/frame", "internal/model", false},
		{"cmd/markitect-agentd", "internal/agentd", true},
		{"cmd/markitect-agentd", "internal/bootstrap", false},
		{"cmd/markitect-runner", "internal/adapters/runner", true},
		{"cmd/markitect-runner", "internal/bootstrap", false},
		{"internal/bootstrap", "internal/adapters/runner", true},
		{"internal/adapters/runner", "internal/bootstrap", false},
		{"internal/adapters/runner", "internal/adapters/container", false},
		{"internal/adapters/cli", "internal/adapters/runner", false},
		{"internal/bootstrap", "internal/adapters/gitrepo", true},
		{"internal/bootstrap", "internal/tooling/release", false},
		{"internal/bootstrap", "internal/tooling/source", false},
		{"cmd/markitect", "internal/bootstrap", true},
		{"cmd/markitect", "internal/apply", false},
		{"cmd/markitect-release", "internal/tooling/releasecli", true},
		{"cmd/markitect-release", "internal/bootstrap", false},
		{"internal/tooling/release", "internal/tooling/source", true},
		{"internal/tooling/release", "internal/apply", false},
		{"internal/tooling/release/install", "internal/tooling/release", true},
		{"internal/tooling/source", "internal/model", false},
		{"packaging/bootstrap", "internal/model", false},
		{"tools/smoke", "internal/bootstrap", false},
	} {
		for _, test := range []bool{false, true} {
			allowed := tc.allowed
			if tc.from == "internal/adapters/gitrepo" && tc.to == "internal/adapters/checks" || tc.from == "internal/adapters/codexappserver" && (tc.to == "internal/adapters/gitrepo" || tc.to == "internal/adapters/runstore") || tc.from == "internal/bootstrap" && tc.to == "internal/snapshot" {
				allowed = test
			} // Actual controlled declared-check composition is test-only.
			got := Check([]Edge{{From: tc.from, To: tc.to, Test: test}})
			if (len(got) == 0) != allowed {
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

func TestRealGitRecordCompositionEdgeStaysTestOnly(t *testing.T) {
	// External gitrepo tests compose the concrete workspace adapter with the
	// check runner; production continues to consume the Run workspace port.
	for _, test := range []bool{false, true} {
		got := Check([]Edge{{From: "internal/adapters/gitrepo", To: "internal/adapters/runstore", Test: test}})
		if (len(got) == 0) != test {
			t.Errorf("gitrepo -> gitrepo test=%v: %v", test, got)
		}
	}
	for _, file := range []string{"fixture_windows_test.go", "fixture_linux_test.go", "fixture_windows.go", "fixture_linux.go"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			test := strings.HasSuffix(file, "_test.go")
			pkg := "gitrepo"
			if test {
				pkg += "_test"
			}
			writeFixture(t, root, map[string]string{
				"README.md": "# Owner\n", CodeMapSource: "packages:\n" + mapEntry("internal/adapters/gitrepo", "apply-adapter") + mapEntry("internal/adapters/runstore", "apply-adapter"),
				"internal/adapters/runstore/producer.go": "package runstore\n",
				"internal/adapters/gitrepo/" + file:      "package " + pkg + "\nimport _ \"" + ModulePath + "internal/adapters/runstore\"\n",
			})
			got, err := CheckRepository(root)
			if err != nil {
				t.Fatal(err)
			}
			if test {
				if len(got) != 0 {
					t.Fatal("external check test composition refused", got)
				}
			} else if len(got) != 1 || got[0].Edge.File != "internal/adapters/gitrepo/"+file || !strings.Contains(got[0].Rule, "responsibility direction forbids") {
				t.Fatal("production check imported workspace adapter", got)
			}
		})
	}
}

func TestCodexProfileImportsAcrossPlatforms(t *testing.T) {
	for _, producer := range []struct{ path, layer, pkg string }{
		{"internal/project", "apply-project", "project"},
		{"internal/runtimeconfig", "runtime-config", "runtimeconfig"},
		{"internal/model", "apply-model", "model"},
		{"internal/organization", "apply-organization", "organization"},
		{"internal/snapshot", "apply-snapshot", "snapshot"},
	} {
		for _, file := range []string{"profile_windows.go", "profile_linux.go", "profile_windows_test.go", "profile_linux_test.go"} {
			t.Run(producer.pkg+"/"+file, func(t *testing.T) {
				root := t.TempDir()
				entries := mapEntry(producer.path, producer.layer) + mapEntry("internal/adapters/codexappserver", "apply-adapter")
				writeFixture(t, root, map[string]string{
					"README.md": "# Owner\n", CodeMapSource: "packages:\n" + entries,
					producer.path + "/producer.go":             "package " + producer.pkg + "\n",
					"internal/adapters/codexappserver/" + file: "package codexappserver\nimport _ \"" + ModulePath + producer.path + "\"\n",
				})
				got, err := CheckRepository(root)
				if err != nil || len(got) != 0 {
					t.Fatalf("Codex profile dependency refused: %v %v", got, err)
				}
			})
		}
	}
}

func TestBootstrapExecutionCompositionPreservesFixtureBoundaries(t *testing.T) {
	// Bootstrap composes actual services using model limits/status and snapshot
	// check values; unrelated responsibilities keep their separate boundaries.
	for _, tc := range []struct {
		to               string
		production, test bool
	}{
		{"internal/apply", true, true}, {"internal/adapters/gitrepo", true, true},
		{"internal/model", true, true},
		{"internal/adapters/codexappserver", true, true},
		{"internal/snapshot", false, true}, {"internal/run", true, true}, {"internal/testkit", false, true},
		{"internal/organization", false, false}, {"internal/realization", false, false},
		{"internal/adapters/container", true, true}, {"internal/adapters/unknown", false, false},
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
		{"internal/project", "apply-project", "project"},
		{"internal/adapters/codexappserver", "apply-adapter", "codexappserver"},
		{"internal/model", "apply-model", "model"},
		{"internal/runtimeconfig", "runtime-config", "runtimeconfig"},
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
				if len(got) != 0 {
					t.Fatal("bootstrap composition dependency refused", got)
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
	for _, edge := range []Edge{{From: "internal/model", To: "internal/testkit"}, {From: "internal/testkit", To: "internal/model", Test: true}, {From: "packaging/bootstrap", To: "internal/testkit", Test: true}} {
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

func TestInspectSkipsNestedModulesOutsideTheProduct(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		"tools/reference/go.mod":          "module reference\n",
		"tools/reference/cmd/x/main.go":   "package main\nimport _ \"reference/internal/y\"\n",
		"tools/reference/internal/y/y.go": "package y\n",
		"tools/harness/h.go":              "package harness\n",
		"internal/model/nested/go.mod":    "module hidden\n",
		"internal/model/nested/hidden.go": "package hidden\n",
	})
	edges, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range edges {
		seen[e.From] = true
	}
	if seen["tools/reference/cmd/x"] || seen["tools/reference/internal/y"] || !seen["tools/harness"] || !seen["internal/model/nested"] {
		t.Fatalf("inspected packages: %v", seen)
	}
}
