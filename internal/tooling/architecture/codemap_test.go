package architecture

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var updateCodeMap = flag.Bool("update", false, "rewrite "+CodeMapPage+" from "+CodeMapSource)

var repositoryRoot = filepath.Join("..", "..", "..")

func TestCodeMapCoversRepository(t *testing.T) {
	m, err := LoadCodeMap(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	problems, err := CheckCodeMap(repositoryRoot, m)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range problems {
		t.Error(problem)
	}
}

func TestCodeMapPage(t *testing.T) {
	m, err := LoadCodeMap(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	want := RenderCodeMap(m)
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

func TestCodeMapLayersMatchResponsibilities(t *testing.T) {
	for _, r := range targetRoots {
		found := false
		for _, l := range Layers {
			if l.Name == r.layer && contains(l.Gate, r.layer) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing display layer for %s: %s", r.path, r.layer)
		}
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

func TestCheckCodeMapReportsEveryProblem(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":                      "module example\n",
		"internal/model/c.go":         "package core\n",
		"internal/apply/h.go":         "package host\n",
		"internal/apply/missing/m.go": "package missing\n",
		"docs/owner.md":               "# Owner\n\n## Real section\n\n```text\n# Not a heading\n```\n\n<a id=\"explicit\"></a>\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	valid := CodeMap{Packages: []MappedPackage{
		{Path: "internal/model", Layer: "apply-model", Purpose: "Core.", Doc: "docs/owner.md#real-section"},
		{Path: "internal/apply", Layer: "apply-usecase", Purpose: "Host.", Doc: "docs/owner.md#explicit"},
		{Path: "internal/apply/missing", Layer: "apply-usecase", Purpose: "Missing.", Doc: "docs/owner.md"},
	}}
	if problems, err := CheckCodeMap(root, valid); err != nil || len(problems) != 0 {
		t.Fatalf("valid map: %v %v", problems, err)
	}
	bad := CodeMap{Packages: []MappedPackage{
		{Path: "internal/apply", Layer: "apply-usecase", Purpose: "Host.", Doc: "docs/owner.md"},
		{Path: "internal/model", Layer: "apply-adapter", Purpose: "Core.", Doc: "docs/owner.md#not-a-heading"},
		{Path: "internal/gone", Layer: "apply-model", Purpose: "", Doc: "docs/absent.md"},
		{Path: "internal/gone", Layer: "nonsense", Purpose: "a | b", Doc: "../outside.md"},
	}}
	problems, err := CheckCodeMap(root, bad)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(problems, "\n")
	for _, want := range []string{
		`entry 2 (internal/model): layer "apply-adapter" does not match import-gate layer "apply-model"`,
		`entry 2 (internal/model): owning document: docs/owner.md has no anchor #not-a-heading`,
		`entry 3 (internal/gone): layer "apply-model" does not match import-gate layer "unknown"`,
		`entry 3 (internal/gone): entries must be ordered by layer, then by path`,
		`entry 3 (internal/gone): purpose must be one non-empty line`,
		`entry 3 (internal/gone): owning document: docs/absent.md does not exist`,
		`entry 4 (internal/gone): duplicate path`,
		`entry 4 (internal/gone): unknown layer "nonsense"`,
		`entry 4 (internal/gone): purpose must be one non-empty line`,
		`entry 4 (internal/gone): owning document: "../outside.md" is not a clean repository-relative path`,
		"missing package internal/apply/missing: add an entry to " + CodeMapSource,
		"extra entry internal/gone: no Go package exists there; remove it from " + CodeMapSource,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing problem %q in:\n%s", want, text)
		}
	}
	if len(problems) != 12 {
		t.Errorf("got %d problems, want 12:\n%s", len(problems), text)
	}
}

func TestParseCodeMapIsStrict(t *testing.T) {
	for _, input := range []string{
		"packages:\n  - path: a\n    owner: x\n",
		"packages: []\n---\npackages: []\n",
		"other: 1\n",
	} {
		if _, err := ParseCodeMap([]byte(input)); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRenderCodeMapLinksFromPageDirectory(t *testing.T) {
	page := string(RenderCodeMap(CodeMap{Packages: []MappedPackage{
		{Path: "internal/model", Layer: "apply-model", Purpose: "Checks.", Doc: "docs/architecture.md#go"},
		{Path: "cmd/markitect", Layer: "apply-command", Purpose: "Runs.", Doc: "CONTRIBUTING.md"},
		{Path: "internal/tooling/release", Layer: "maintenance", Purpose: "Builds.", Doc: "docs/development/modules.md"},
	}}))
	for _, want := range []string{
		"| `internal/model` | Checks. | [docs/architecture.md](../architecture.md#go) |",
		"| `cmd/markitect` | Runs. | [CONTRIBUTING.md](../../CONTRIBUTING.md) |",
		"| `internal/tooling/release` | Builds. | [docs/development/modules.md](modules.md) |",
		"[codemap.yaml](../../internal/tooling/architecture/codemap.yaml)",
		CodeMapUpdate,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("rendered page lacks %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "## Modules") {
		t.Error("rendered a section for an empty layer")
	}
}
