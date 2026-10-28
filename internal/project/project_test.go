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
		ChecksFile:                      "checks:\n  - name: test\n    command: go test ./...\n    timeout: 5m\n    environment: golang:1.27\n",
		ModelDir + "/project.md":        "---\nkind: std/Project\n---\n# Demo\n\nSee [[members]].\n",
		ModelDir + "/members/member.md": "---\nkind: std/Concept\n---\n# Member\n",
		RootPage:                        "| File | Realizes | Why |\n|---|---|---|\n| README.md | [[project]] | Overview |\n",
		RegistryDir + "/members.md":     "---\npaths: [src/**]\n---\n| File | Realizes | Why |\n|---|---|---|\n| src/member.go | [[members/member]] | Type |\n",
		IgnoreFile:                      "| Path | Why |\n|---|---|\n| LICENSE | Legal text |\n",
		ResultsDir + "/run-1.md":        "# Result\n",
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
	if len(p.Checks) != 1 || p.Checks[0].Directory != "." || p.Checks[0].Timeout.Minutes() != 5 {
		t.Fatalf("checks = %+v", p.Checks)
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
	p := load(t, r)
	expect(t, p.Errors, "project.file", ".markitect/notes.txt")
	expect(t, p.Errors, "registry.page", ".markitect/registry/members.yaml")
}

func TestLoadMissingControlFiles(t *testing.T) {
	r := valid(t)
	delete(r.files, ManifestFile)
	delete(r.files, ChecksFile)
	p := load(t, r)
	expect(t, p.Errors, "manifest.missing", "run markitect init")
	expect(t, p.Errors, "execution.missing", "run markitect init")

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
	for _, d := range p.Errors {
		t.Errorf("unexpected: %v", d)
	}
	if p.Model.Definitions["home"].Kind != "web/Page" {
		t.Fatalf("kind = %q", p.Model.Definitions["home"].Kind)
	}

	r.files[LibraryDir+"/web/Page.md"] += "Changed.\n"
	expect(t, load(t, r).Errors, "lock.mismatch", "content of Library web 0.2.0 does not match")
}

func TestParseChecksErrors(t *testing.T) {
	_, ds := ParseChecks("execution.yaml", []byte(`checks:
  - name: ok
    command: make
    timeout: 1m
    environment: debian
  - name: ok
    command: make
    timeout: 1m
    environment: debian
  - name: "bad name"
  - name: nocommand
    timeout: 1m
  - name: abs
    command: make
    directory: /etc
    timeout: 1m
    environment: x
  - name: notimeout
    command: make
    environment: x
  - name: noenv
    command: make
    timeout: 1m
  - name: extra
    command: make
    timeout: 1m
    environment: x
    image: y
  - just text
other: 1
`))
	for _, want := range []string{"named twice", "needs a name", "needs a command", "directory must be repository-relative", "needs a timeout", "needs an environment", `unknown check key "image"`, "each check is a mapping", `unknown key "other"`} {
		expect(t, ds, "execution.syntax", want)
	}
	_, ds = ParseChecks("execution.yaml", []byte("checks: {}\n"))
	expect(t, ds, "execution.syntax", "checks must be a list")
	_, ds = ParseChecks("execution.yaml", []byte("checks: [\n"))
	expect(t, ds, "execution.syntax", "not valid YAML")
}
