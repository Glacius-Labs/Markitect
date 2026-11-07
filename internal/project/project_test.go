package project

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// memRepo is a repository state held in memory.
type memRepo struct {
	files   map[string]string
	ignored map[string]bool
}

func (r memRepo) Files() ([]string, error) {
	var out []string
	for f := range r.files {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

func (r memRepo) Read(f string) ([]byte, error) {
	data, ok := r.files[f]
	if !ok {
		return nil, fmt.Errorf("no file %s", f)
	}
	return []byte(data), nil
}

func (r memRepo) GitIgnored(f string) bool { return r.ignored[f] }

// stdLock is the lock for std alone.
func stdLock(t *testing.T) string {
	t.Helper()
	files, _ := libraries.Files("std")
	lib := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		lib.Files = append(lib.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return string(model.RenderLock([]model.LibrarySource{lib}))
}

func valid(t *testing.T) memRepo {
	return memRepo{files: map[string]string{
		ManifestFile:                    "libraries:\n  std: 1.0.0\n",
		LockFile:                        stdLock(t),
		ModelDir + "/project.md":        "---\nkind: std/Project\n---\n# Demo\n\nSee [[members]].\n",
		ModelDir + "/members/member.md": "---\nkind: std/Concept\n---\n# Member\n",
		RootPage:                        "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n",
		RegistryDir + "/members.md":     "---\npaths: [src/**]\n---\n| File | Realizes | Why |\n|---|---|---|\n| src/member.go | [[members/member]] | Type |\n",
		IgnoreFile:                      "# Legal text\nLICENSE\n",
		ReviewFile:                      "",
		AttributesFile:                  Attributes,
		Dir + "/tool/lock.yaml":         "installed tool\n",
		"README.md":                     "# Demo\n",
		"src/member.go":                 "package members\n",
		"LICENSE":                       "MIT\n",
	}, ignored: map[string]bool{}}
}

func load(t *testing.T, r memRepo) *Project {
	t.Helper()
	p, err := Load(r)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func expect(t *testing.T, ds model.Diagnostics, code, fragment string) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.String(), fragment) {
			return
		}
	}
	t.Fatalf("no %s containing %q in:\n%v", code, fragment, ds)
}

func TestLoadValidProject(t *testing.T) {
	p := load(t, valid(t))
	if len(p.Errors) != 0 {
		t.Fatalf("errors:\n%v", p.Errors)
	}
	if p.Registry.Owner["src/member.go"] != "members" || !p.Registry.Ignored["LICENSE"] {
		t.Fatalf("registry = %+v", p.Registry)
	}
}

func TestLoadNormalizesCRLF(t *testing.T) {
	r := valid(t)
	for f, data := range r.files {
		if strings.HasPrefix(f, Dir+"/") {
			r.files[f] = strings.ReplaceAll(data, "\n", "\r\n")
		}
	}
	if p := load(t, r); len(p.Errors) != 0 {
		t.Fatalf("CRLF checkout does not compile:\n%v", p.Errors)
	}
}

func TestLoadUnknownOwnFiles(t *testing.T) {
	r := valid(t)
	r.files[Dir+"/notes.txt"] = "x"
	r.files[RegistryDir+"/members.yaml"] = "x"
	r.files[Dir+"/results/gate-20261106-120000-aaaaaaaa.json"] = "{}"
	p := load(t, r)
	expect(t, p.Errors, "project.file", ".markitect/notes.txt")
	expect(t, p.Errors, "registry.page", ".markitect/registry/members.yaml")
	// The certificates of markitect 0.20 are retired, and update turns them into
	// the record.
	expect(t, p.Errors, "project.results", "markitect update")
}

func TestLoadMissingControlFiles(t *testing.T) {
	r := valid(t)
	delete(r.files, ManifestFile)
	p := load(t, r)
	expect(t, p.Errors, "manifest.missing", "run markitect init")

	r = valid(t)
	delete(r.files, LockFile)
	expect(t, load(t, r).Errors, "lock.missing", "run markitect lock")
}

func TestLoadLibraryErrors(t *testing.T) {
	r := valid(t)
	r.files[ManifestFile] = "libraries:\n  std: 2.0.0\n  rest: 1.0.0\n  web: 1.0.0\n  cli: 1.0.0\n"
	r.files[LibraryDir+"/cli/Command.md"] = "---\nkind: Kind\n---\n# Command\n"
	r.files[LibraryDir+"/orphan/Thing.md"] = "---\nkind: Kind\n---\n# Thing\n"
	p := load(t, r)
	expect(t, p.Errors, "library.version", "this markitect ships std 1.0.0, not 2.0.0")
	expect(t, p.Errors, "library.missing", "Library web is neither a standard Library nor committed")
	expect(t, p.Errors, "library.reserved", "cli is a standard Library")
	expect(t, p.Errors, "library.undeclared", "Library orphan")

	r = valid(t)
	r.files[ManifestFile] = "libraries:\n  rest: 1.0.0\n"
	expect(t, load(t, r).Errors, "library.missing", "every project uses the standard Library std")
}

func TestLoadLocalLibraryAndLock(t *testing.T) {
	r := valid(t)
	r.files[ManifestFile] = "libraries:\n  std: 1.0.0\n  web: 0.2.0\n"
	r.files[LibraryDir+"/web/Page.md"] = "---\nkind: Kind\n---\n# Page\n\nA web page, a kind of [[std/Concept]].\n"
	r.files[ModelDir+"/home.md"] = "---\nkind: web/Page\n---\n# Home\n"
	expect(t, load(t, r).Errors, "lock.mismatch", "web 0.2.0 is not in the lock")

	local, err := LocalLibraries(r, []string{LibraryDir + "/web/Page.md", ModelDir + "/home.md"})
	if err != nil {
		t.Fatal(err)
	}
	std, _, _ := LibrarySource(model.LibraryRef{Name: "std", Version: "1.0.0"}, local)
	web, code, _ := LibrarySource(model.LibraryRef{Name: "web", Version: "0.2.0"}, local)
	if code != "" {
		t.Fatal(code)
	}
	r.files[LockFile] = string(model.RenderLock([]model.LibrarySource{std, web}))
	p := load(t, r)
	for _, d := range p.SpecErrors() {
		t.Errorf("unexpected: %v", d)
	}
	if p.Model.Definitions["home"].Kind != "web/Page" {
		t.Fatalf("kind = %q", p.Model.Definitions["home"].Kind)
	}

	r.files[LibraryDir+"/web/Page.md"] += "Changed.\n"
	expect(t, load(t, r).Errors, "lock.mismatch", "content of Library web 0.2.0 does not match")
}

// Spec and code that do not match are errors, apart from errors in the spec: a
// deleted Definition's entry and a Definition nothing uses stop check, not a Run.
func TestRealizationErrors(t *testing.T) {
	r := valid(t)
	r.files[ModelDir+"/unused.md"] = "---\nkind: std/Concept\n---\n# Unused\n"
	r.files[RegistryDir+"/members.md"] = "---\npaths: [src/**]\n---\n| File | Realizes | Why |\n|---|---|---|\n| src/member.go | [[members/member]] | Member |\n| src/gone.go | [[members/member]] | Gone |\n"
	p := load(t, r)
	if len(p.SpecErrors()) != 0 {
		t.Fatalf("spec errors: %v", p.SpecErrors())
	}
	got := map[string]bool{}
	for _, d := range p.RealizationErrors() {
		got[d.Code] = true
	}
	for _, code := range []string{CodeUnused, "stale.file"} {
		if !got[code] {
			t.Errorf("no %s in %v", code, p.RealizationErrors())
		}
	}
}

func TestNothingBuilt(t *testing.T) {
	if p := load(t, valid(t)); p.NothingBuilt() {
		t.Fatal("a built project with no errors")
	}
	fresh := memRepo{files: map[string]string{
		ManifestFile:             "libraries:\n  std: 1.0.0\n",
		LockFile:                 stdLock(t),
		ModelDir + "/project.md": "---\nkind: std/Project\n---\n# Demo\n",
		RootPage:                 "| File | Realizes | Why |\n|---|---|---|\n",
		".gitignore":             "x\n",
	}}
	if p := load(t, fresh); !p.NothingBuilt() {
		t.Fatalf("a new project: %v", p.Errors)
	}
	fresh.files["src/a.go"] = "package a\n"
	if p := load(t, fresh); p.NothingBuilt() {
		t.Fatal("an unlisted file is not nothing")
	}
	delete(fresh.files, "src/a.go")
	fresh.files[ModelDir+"/broken.md"] = "no front matter\n"
	if p := load(t, fresh); p.NothingBuilt() {
		t.Fatal("an error in the spec is not nothing")
	}
}

func TestManifestHoldsTheModel(t *testing.T) {
	r := valid(t)
	r.files[ManifestFile] = "libraries:\n  std: 1.0.0\nmodel: opus\n"
	p := load(t, r)
	if len(p.Errors) != 0 || p.Manifest.Model != "opus" {
		t.Fatalf("%+v %v", p.Manifest, p.Errors)
	}
	r.files[ManifestFile] = "libraries:\n  std: 1.0.0\neffort: max\n"
	expect(t, load(t, r).Errors, "manifest.syntax", `unknown key "effort"`)
}

// The files of the old layout are errors, with no compatibility kept.
func TestOldFilesAreRefused(t *testing.T) {
	r := valid(t)
	r.files[Dir+"/execution.yaml"] = "checks: []\n"
	r.files[Dir+"/ignore.md"] = "| Path | Why |\n"
	p := load(t, r)
	expect(t, p.Errors, "project.file", "holds only")
	if n := len(p.Errors); n != 2 {
		t.Fatalf("%d errors: %v", n, p.Errors)
	}
}

// What init installs, and Git's ignore file, need no registry entry; a page may
// still list them.
func TestTheToolAccountsForItsOwnFiles(t *testing.T) {
	r := valid(t)
	for _, f := range []string{".gitignore", "AGENTS.md", "CLAUDE.md", ".claude/agents/markitect-reviewer.md", ".claude/skills/markitect-start/SKILL.md", ".agents/skills/markitect-start/SKILL.md"} {
		r.files[f] = "x\n"
	}
	r.files[RootPage] += "| .gitignore | [[project]] | Listed by the owner |\n"
	p := load(t, r)
	if len(p.Errors) != 0 {
		t.Fatalf("errors: %v", p.Errors)
	}
	if !p.Registry.Ignored["AGENTS.md"] || p.Registry.Ignored[".gitignore"] || p.Registry.Entries[".gitignore"] == nil {
		t.Fatalf("registry = %+v", p.Registry)
	}
}

func TestIsHarness(t *testing.T) {
	for file, want := range map[string]bool{
		"AGENTS.md": true, "claude.md": true, ".claude/agents/markitect-reviewer.md": true, ".CLAUDE/skills/markitect-apply/SKILL.md": true,
		".gitignore": false, "src/AGENTS.md": false, ".claude/skills/my-own/SKILL.md": false,
	} {
		if got := IsHarness(file); got != want {
			t.Errorf("IsHarness(%q) = %t", file, got)
		}
	}
}
