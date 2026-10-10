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

var repositoryRoot = filepath.Join("..", "..", "..", "..")

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

// Every map layer names at least one real gate layer, and every gate layer
// except "unknown" is reachable, so the finer layers can drive the gate.
func TestCodeMapLayersMatchGateLayers(t *testing.T) {
	gate := map[string]bool{"core": true, "host": true, "module": true, "infrastructure": true, "tooling": true, "cli": true, "fixture": true, "bootstrap": true, "harness-tests": true, "harness-runtime": true, "testkit": true}
	reached := map[string]bool{}
	names := map[string]bool{}
	for _, l := range Layers {
		if names[l.Name] || l.Title == "" || l.Meaning == "" || len(l.Gate) == 0 {
			t.Errorf("layer %q must be unique and have a title, meaning and gate layers", l.Name)
		}
		names[l.Name] = true
		for _, g := range l.Gate {
			if !gate[g] {
				t.Errorf("layer %q names unknown gate layer %q", l.Name, g)
			}
			reached[g] = true
		}
	}
	if !reflect.DeepEqual(reached, gate) {
		t.Errorf("gate layers without a map layer: reached %v of %v", reached, gate)
	}
	for _, c := range []struct {
		path, layer string
		ok          bool
	}{
		{"src/internal/host/projectcoverage", "core", true},
		{"src/internal/modules/projectmodel", "core", true},
		{"src/internal/modules/markdown", "legacy", true},
		{"src/internal/host/projectrun", "runtime", true},
		{"src/internal/modules/markdown", "application", false},
		{"src/cmd/markitect", "legacy", false},
		{"src/internal/core", "legacy", false},
		{"runs/c11checktools/original", "fixture", true},
		{"src/internal/unowned", "legacy", false},
	} {
		gateLayer, _ := layer(c.path)
		var allowed []string
		for _, l := range Layers {
			if l.Name == c.layer {
				allowed = l.Gate
			}
		}
		if contains(allowed, gateLayer) != c.ok {
			t.Errorf("%s as %s: gate layer %q, want compatible=%v", c.path, c.layer, gateLayer, c.ok)
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
		"go.mod":                         "module example\n",
		"src/internal/core/c.go":         "package core\n",
		"src/internal/host/h.go":         "package host\n",
		"src/internal/host/missing/m.go": "package missing\n",
		"docs/owner.md":                  "# Owner\n\n## Real section\n\n```text\n# Not a heading\n```\n\n<a id=\"explicit\"></a>\n",
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
		{Path: "src/internal/core", Layer: "core", Purpose: "Core.", Doc: "docs/owner.md#real-section"},
		{Path: "src/internal/host", Layer: "legacy", Purpose: "Host.", Doc: "docs/owner.md#explicit"},
		{Path: "src/internal/host/missing", Layer: "legacy", Purpose: "Missing.", Doc: "docs/owner.md"},
	}}
	if problems, err := CheckCodeMap(root, valid); err != nil || len(problems) != 0 {
		t.Fatalf("valid map: %v %v", problems, err)
	}
	bad := CodeMap{Packages: []MappedPackage{
		{Path: "src/internal/host", Layer: "legacy", Purpose: "Host.", Doc: "docs/owner.md"},
		{Path: "src/internal/core", Layer: "application", Purpose: "Core.", Doc: "docs/owner.md#not-a-heading"},
		{Path: "src/internal/gone", Layer: "core", Purpose: "", Doc: "docs/absent.md"},
		{Path: "src/internal/gone", Layer: "nonsense", Purpose: "a | b", Doc: "../outside.md"},
	}}
	problems, err := CheckCodeMap(root, bad)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(problems, "\n")
	for _, want := range []string{
		`entry 2 (src/internal/core): layer "application" does not match import-gate layer "core"`,
		`entry 2 (src/internal/core): entries must be ordered by layer, then by path`,
		`entry 2 (src/internal/core): owning document: docs/owner.md has no anchor #not-a-heading`,
		`entry 3 (src/internal/gone): layer "core" does not match import-gate layer "unknown"`,
		`entry 3 (src/internal/gone): entries must be ordered by layer, then by path`,
		`entry 3 (src/internal/gone): purpose must be one non-empty line`,
		`entry 3 (src/internal/gone): owning document: docs/absent.md does not exist`,
		`entry 4 (src/internal/gone): duplicate path`,
		`entry 4 (src/internal/gone): unknown layer "nonsense"`,
		`entry 4 (src/internal/gone): purpose must be one non-empty line`,
		`entry 4 (src/internal/gone): owning document: "../outside.md" is not a clean repository-relative path`,
		"missing package src/internal/host/missing: add an entry to " + CodeMapSource,
		"extra entry src/internal/gone: no Go package exists there; remove it from " + CodeMapSource,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing problem %q in:\n%s", want, text)
		}
	}
	if len(problems) != 13 {
		t.Errorf("got %d problems, want 13:\n%s", len(problems), text)
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
		{Path: "src/internal/core", Layer: "core", Purpose: "Compiles.", Doc: "docs/architecture.md#go"},
		{Path: "src/cmd/markitect", Layer: "cli", Purpose: "Runs.", Doc: "CONTRIBUTING.md"},
		{Path: "src/internal/tooling/release", Layer: "tooling", Purpose: "Builds.", Doc: "docs/development/modules.md"},
	}}))
	for _, want := range []string{
		"| `src/internal/core` | Compiles. | [docs/architecture.md](../architecture.md#go) |",
		"| `src/cmd/markitect` | Runs. | [CONTRIBUTING.md](../../CONTRIBUTING.md) |",
		"| `src/internal/tooling/release` | Builds. | [docs/development/modules.md](modules.md) |",
		"[codemap.yaml](../../src/internal/tooling/architecture/codemap.yaml)",
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
