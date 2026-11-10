package architecture

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var updateCodeMap = flag.Bool("update", false, "rewrite "+CodeMapPage+" from "+RulesSource)

var repositoryRoot = filepath.Join("..", "..", "..")

func TestRuleTableCoversRepository(t *testing.T) {
	problems, err := Table.CheckInventory(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range problems {
		t.Error(problem)
	}
}

func TestCodeMapPage(t *testing.T) {
	want, err := Table.RenderCodeMap(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(repositoryRoot, filepath.FromSlash(CodeMapPage))
	if *updateCodeMap {
		if err := os.WriteFile(page, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("%v; generate it with: %s", err, CodeMapUpdate)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != string(want) {
		t.Fatalf("%s is stale; regenerate it with: %s", CodeMapPage, CodeMapUpdate)
	}
}

func TestPackagesFollowGoListRules(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{
		"go.mod", "main.go", "a/a.go", "b/b_test.go", "c/c_windows.go", "a/deep/d.go",
		"testdata/t/t.go", ".hidden/h.go", "_skip/s.go", "vendor/v/v.go", "node_modules/n/n.go",
		"e/_ignored.go", "e/.ignored.go", "f/notes.md", "nested/go.mod", "nested/n.go", "nested/inner/i.go",
	} {
		write(name, "package x\n")
	}
	got, err := Packages(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".", "a", "a/deep", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("packages = %v, want %v", got, want)
	}
}

func TestCheckInventoryReportsEveryProblem(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		"go.mod":                         "module example\n",
		"internal/spec/c.go":             "package spec\n",
		"internal/analysis/h.go":         "package analysis\n",
		"internal/analysis/missing/m.go": "package missing\n",
		"docs/owner.md":                  "# Owner\n\n## Real section\n\n```text\n# Not a heading\n```\n\n<a id=\"explicit\"></a>\n",
	})
	valid := Rules{Layers: Table.Layers, Packages: []Package{
		{"internal/spec", "spec", "docs/owner.md#real-section"},
		{"internal/analysis", "service", "docs/owner.md#explicit"},
	}}
	if problems, err := valid.CheckInventory(root); err != nil || len(problems) != 1 || problems[0] != "missing package internal/analysis/missing: add a row to the rule table" {
		t.Fatalf("valid rows: %v %v", problems, err)
	}
	bad := Rules{Layers: Table.Layers, Packages: []Package{
		{"internal/spec", "spec", "docs/owner.md#not-a-heading"},
		{"internal/analysis", "service", "docs/owner.md"},
		{"internal/gone", "service", "docs/absent.md"},
		{"internal/other", "service", "../outside.md"},
	}}
	problems, err := bad.CheckInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(problems, "\n")
	for _, want := range []string{
		"package internal/spec: owning document: docs/owner.md has no anchor #not-a-heading",
		"package internal/gone: owning document: docs/absent.md does not exist",
		`package internal/other: owning document: "../outside.md" is not a clean repository-relative path`,
		"missing package internal/analysis/missing: add a row to the rule table",
		"extra row internal/gone: no Go package exists there; remove it from the rule table",
		"extra row internal/other: no Go package exists there; remove it from the rule table",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing problem %q in:\n%s", want, text)
		}
	}
	if len(problems) != 6 {
		t.Errorf("got %d problems, want 6:\n%s", len(problems), text)
	}
}

func TestSummariesNeedExactlyOneDocComment(t *testing.T) {
	rules := Rules{Layers: Table.Layers, Packages: []Package{{"internal/spec", "spec", "README.md"}}}
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"one", map[string]string{"a.go": "// Package spec checks specs. More text.\npackage spec\n", "b.go": "package spec\n", "a_test.go": "// Package spec tests.\npackage spec\n"}, "Package spec checks specs."},
		{"none", map[string]string{"a.go": "package spec\n"}, "package internal/spec has no package doc comment"},
		{"build constraint only", map[string]string{"a.go": "//go:build windows\n\npackage spec\n"}, "package internal/spec has no package doc comment"},
		{"two", map[string]string{"a.go": "// Package spec a.\npackage spec\n", "b.go": "// Package spec b.\npackage spec\n"}, "package internal/spec has package doc comments in a.go, b.go; keep one"},
		{"pipe", map[string]string{"a.go": "// Package spec a | b.\npackage spec\n"}, "must be non-empty and contain no '|'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{}
			for name, content := range tc.files {
				files["internal/spec/"+name] = content
			}
			writeFixture(t, root, files)
			got, err := rules.Summaries(root)
			if strings.HasPrefix(tc.want, "Package ") {
				if err != nil || got["internal/spec"] != tc.want {
					t.Fatalf("%v %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%v %v, want %s", got, err, tc.want)
			}
		})
	}
}

func TestRenderCodeMapLinksFromPageDirectory(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		"internal/spec/m.go":                 "// Package spec checks.\npackage spec\n",
		"cmd/markitect/main.go":              "// Command markitect runs.\npackage main\n",
		"internal/tooling/architecture/r.go": "// Package architecture checks.\npackage architecture\n",
		"internal/buildinfo/b.go":            "// Package buildinfo knows the version.\npackage buildinfo\n",
	})
	rules := Rules{Layers: Table.Layers, Packages: []Package{
		{"internal/spec", "spec", "docs/architecture.md#go"},
		{"internal/buildinfo", "build", "docs/development/other.md"},
		{"cmd/markitect", "command", "CONTRIBUTING.md"},
		{"internal/tooling/architecture", "maintenance", "internal/tooling/architecture/README.md"},
	}, Exceptions: []Exception{
		{From: "internal/tooling/architecture", To: "internal/buildinfo", TestOnly: true, Reason: "Tests read the notices."},
		{From: "cmd/markitect", To: "internal/spec", Reason: "A fixture edge."},
	}}
	data, err := rules.RenderCodeMap(root)
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, want := range []string{
		"| `internal/spec` | Package spec checks. | [docs/architecture.md#go](../architecture.md#go) |",
		"| `cmd/markitect` | Command markitect runs. | [CONTRIBUTING.md](../../CONTRIBUTING.md) |",
		"| `internal/buildinfo` | Package buildinfo knows the version. | [docs/development/other.md](other.md) |",
		"| `internal/tooling/architecture` | Package architecture checks. | [internal/tooling/architecture/README.md](../../internal/tooling/architecture/README.md) |",
		"| `spec` | The spec language over supplied bytes: Definitions, Kinds, links, visibility, cycles, manifest and lock. | nothing | 1 |",
		"| `command` | Thin entrypoints to product composition. | `composition` | 1 |",
		"| `test-support` | Hermetic fixtures. Only test files may import it. It imports no Markitect package, not even from tests. | nothing | 0 |",
		"| `cmd/markitect` | `internal/spec` | all | A fixture edge. |\n| `internal/tooling/architecture` | `internal/buildinfo` | tests only | Tests read the notices. |",
		"[rules.go](../../internal/tooling/architecture/rules.go)",
		"[Architecture](../architecture.md#layers)",
		CodeMapUpdate,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("rendered page lacks %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "## Analysis") {
		t.Error("rendered a section for an empty layer")
	}
}
