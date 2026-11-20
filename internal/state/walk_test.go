package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// ignoreCase is one rule of Git's gitignore documentation: the files of a
// tree, the ones Git keeps, and whether the names exist on every file system.
type ignoreCase struct {
	name  string
	files map[string]string
	keep  []string
	// portable cases use names that Windows and case-insensitive file systems
	// can hold, so that they also run against Git there.
	portable bool
}

var ignoreCases = []ignoreCase{
	{name: "comments and blank lines", portable: true,
		files: map[string]string{".gitignore": "# a comment\n\n\\#hash\n", "#hash": "", "# a comment": "", "keep": ""},
		keep:  []string{"# a comment", ".gitignore", "keep"}},
	{name: "trailing spaces",
		files: map[string]string{".gitignore": "trail   \nspace\\ \n", "trail": "", "space ": "", "space": ""},
		keep:  []string{".gitignore", "space"}},
	{name: "negation", portable: true,
		files: map[string]string{".gitignore": "*.log\n!keep.log\n", "a.log": "", "keep.log": "", "sub/b.log": "", "sub/keep.log": ""},
		keep:  []string{".gitignore", "keep.log", "sub/keep.log"}},
	{name: "no negation inside an excluded folder", portable: true,
		files: map[string]string{".gitignore": "build/\n!build/keep.txt\n", "build/a.txt": "", "build/keep.txt": "", "keep.txt": ""},
		keep:  []string{".gitignore", "keep.txt"}},
	{name: "an escaped exclamation mark", portable: true,
		files: map[string]string{".gitignore": "\\!bang\n", "!bang": "", "bang": ""},
		keep:  []string{".gitignore", "bang"}},
	{name: "folders only", portable: true,
		files: map[string]string{".gitignore": "cache/\n", "cache/x": "", "a/cache": "", "sub/cache/y": ""},
		keep:  []string{".gitignore", "a/cache"}},
	{name: "anchoring", portable: true,
		files: map[string]string{".gitignore": "/top.txt\nmid/deep.txt\n", "top.txt": "", "sub/top.txt": "", "mid/deep.txt": "", "sub/mid/deep.txt": ""},
		keep:  []string{".gitignore", "sub/mid/deep.txt", "sub/top.txt"}},
	{name: "a star stays in its part", portable: true,
		files: map[string]string{".gitignore": "doc/*.txt\n", "doc/a.txt": "", "doc/sub/b.txt": "", "a.txt": ""},
		keep:  []string{".gitignore", "a.txt", "doc/sub/b.txt"}},
	{name: "leading double star", portable: true,
		files: map[string]string{".gitignore": "**/logs\n**/run/out\n", "logs/a": "", "x/y/logs/b": "", "run/out": "", "x/run/out": "", "x/logs.txt": ""},
		keep:  []string{".gitignore", "x/logs.txt"}},
	{name: "inner double star", portable: true,
		files: map[string]string{".gitignore": "a/**/b\n", "a/b": "", "a/x/b": "", "a/x/y/b": "", "a/xb": "", "c/a/b": ""},
		keep:  []string{".gitignore", "a/xb", "c/a/b"}},
	{name: "trailing double star", portable: true,
		files: map[string]string{".gitignore": "abc/**\n!abc/keep\n", "abc/x": "", "abc/deep/y": "", "abc/keep": ""},
		keep:  []string{".gitignore", "abc/keep"}},
	{name: "brackets and question marks",
		files: map[string]string{".gitignore": "[!a]x\nf[0-9]\n?.c\n", "bx": "", "ax": "", "f1": "", "fa": "", "a.c": "", "ab.c": ""},
		keep:  []string{".gitignore", "ab.c", "ax", "fa"}},
	{name: "brackets", portable: true,
		files: map[string]string{".gitignore": "[!a]x\nf[0-9]\n", "bx": "", "ax": "", "f1": "", "fa": ""},
		keep:  []string{".gitignore", "ax", "fa"}},
	{name: "an escaped star",
		files: map[string]string{".gitignore": "\\*star\n", "*star": "", "xstar": ""},
		keep:  []string{".gitignore", "xstar"}},
	{name: "case counts",
		files: map[string]string{".gitignore": "Readme\n", "Readme": "", "readme": ""},
		keep:  []string{".gitignore", "readme"}},
	{name: "nested ignore files", portable: true,
		files: map[string]string{".gitignore": "*.tmp\n", "sub/.gitignore": "!keep.tmp\n/local\n", "a.tmp": "", "local": "",
			"sub/a.tmp": "", "sub/keep.tmp": "", "sub/local": "", "sub/deeper/local": "", "keep.tmp": ""},
		keep: []string{".gitignore", "local", "sub/.gitignore", "sub/deeper/local", "sub/keep.tmp"}},
	{name: "a deeper file wins", portable: true,
		files: map[string]string{".gitignore": "!*.dat\n", "sub/.gitignore": "*.dat\n", "a.dat": "", "sub/b.dat": ""},
		keep:  []string{".gitignore", "a.dat", "sub/.gitignore"}},
	{name: "byte order mark and CRLF", portable: true,
		files: map[string]string{".gitignore": "\xef\xbb\xbfone\r\ntwo\r\n", "one": "", "two": "", "three": ""},
		keep:  []string{".gitignore", "three"}},
}

func mapFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

func TestWalkFollowsGitignore(t *testing.T) {
	for _, c := range ignoreCases {
		got, err := Walk(mapFS(c.files))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, "|") != strings.Join(c.keep, "|") {
			t.Errorf("%s: got %q, want %q", c.name, got, c.keep)
		}
	}
}

// TestWalkMatchesGit lists each case with Git itself, without any global or
// system setting, and compares. It is skipped where Git is not installed.
func TestWalkMatchesGit(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	empty := filepath.Join(home, "empty")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL="+empty, "GIT_CONFIG_NOSYSTEM=1", "HOME="+home, "XDG_CONFIG_HOME="+home)
	gitIn := func(dir string, args ...string) string {
		cmd := exec.Command(git, args...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return string(out)
	}
	cases := append([]ignoreCase{}, ignoreCases...)
	cases = append(cases, ignoreCase{name: "another repository", portable: true,
		files: map[string]string{".gitignore": "*.o\n", "lib/inner/a.txt": "", "lib/b.txt": "", "c.o": ""}})
	for _, c := range cases {
		if !c.portable && runtime.GOOS == "windows" {
			continue
		}
		dir := t.TempDir()
		for p, text := range c.files {
			file := filepath.Join(dir, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		gitIn(dir, "init", "-q")
		if c.name == "another repository" {
			gitIn(filepath.Join(dir, "lib", "inner"), "init", "-q")
		}
		var want []string
		for _, p := range strings.Split(gitIn(dir, "-c", "core.ignorecase=false", "ls-files", "--others", "--exclude-standard", "-z"), "\x00") {
			if p != "" {
				want = append(want, strings.TrimSuffix(p, "/"))
			}
		}
		sort.Strings(want)
		got, err := Walk(os.DirFS(dir))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s: got %q, Git lists %q", c.name, got, want)
		}
	}
}

func TestWalkLeavesOut(t *testing.T) {
	fsys := mapFS(map[string]string{
		".git/HEAD":                 "ref: refs/heads/main\n",
		".markitect/version":        "1.0.0\n",
		".markitect/ignore":         "*.txt\n!notes.md\nscratch/\n",
		".gitignore":                "notes.md\n*.log\n!keep.log\n",
		"notes.md":                  "",
		"a.txt":                     "",
		"keep.log":                  "",
		"scratch/x.md":              "",
		"src/main.go":               "",
		"src/.git":                  "not a repository\n",
		"repo/.git/HEAD":            "ref: refs/heads/main\n",
		"repo/inside.go":            "",
		"worktree/.git":             "gitdir: ../.git/worktrees/w\n",
		"worktree/inside.go":        "",
		"fake/.git/config":          "",
		"fake/file.go":              "",
		"docs/spec/lending.md":      "",
		"docs/.markitect/version":   "",
		"docs/spec/.git/objects/xx": "",
	})
	got, err := Walk(fsys)
	if err != nil {
		t.Fatal(err)
	}
	// .markitect/ignore and the .gitignore files each exclude on their own: a
	// negation in one brings back nothing the other excludes.
	want := []string{".gitignore", "docs/.markitect/version", "docs/spec/lending.md", "fake/file.go", "keep.log", "repo", "src/main.go", "worktree"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if data, err := Content(fsys, "repo"); err != nil || string(data) != repositoryEntry {
		t.Errorf("repo reads %q %v", data, err)
	}
	if _, err := Content(fsys, "fake"); err == nil {
		t.Error("a folder read as a file")
	}
}

func TestWalkFailsOnAnUnreadableIgnoreFile(t *testing.T) {
	// A folder named .gitignore is no ignore file.
	fsys := mapFS(map[string]string{"sub/.gitignore/x": "", "sub/a.txt": ""})
	if got, err := Walk(fsys); err != nil || strings.Join(got, "|") != "sub/.gitignore/x|sub/a.txt" {
		t.Errorf("got %q %v", got, err)
	}
	fsys = mapFS(map[string]string{"sub/.gitignore": "a.txt\n", "sub/a.txt": ""})
	if _, err := Walk(failing{fsys}); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("an unreadable .gitignore was taken as none: %v", err)
	}
}

// failing fails every read of a file named .gitignore.
type failing struct{ fstest.MapFS }

func (f failing) ReadFile(name string) ([]byte, error) {
	if path.Base(name) == ".gitignore" {
		return nil, errors.New("refused")
	}
	return f.MapFS.ReadFile(name)
}

func TestSymbolicLink(t *testing.T) {
	// A link to a folder is one entry, read as where it points, and a pattern
	// for folders does not match it, as in Git.
	mapped := fstest.MapFS{
		".gitignore": {Data: []byte("linked/\n")},
		"real/a.txt": {Data: []byte("a")},
		"linked":     {Data: []byte("real"), Mode: fs.ModeSymlink},
	}
	if got, err := Walk(mapped); err != nil || strings.Join(got, "|") != ".gitignore|linked|real/a.txt" {
		t.Errorf("got %q %v", got, err)
	}
	if h, err := HashFile(mapped, "linked"); err != nil || h != sha([]byte("real")) {
		t.Errorf("the link hashes %s %v", h, err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real", "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "link")); err != nil {
		t.Skip("this system cannot make a symbolic link:", err)
	}
	fsys := os.DirFS(dir)
	got, err := Walk(fsys)
	if err != nil || strings.Join(got, "|") != "link|real/a.txt" {
		t.Fatalf("got %q %v", got, err)
	}
	if data, err := Content(fsys, "link"); err != nil || string(data) != "real" {
		t.Errorf("the link reads %q %v", data, err)
	}
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestHashIgnoresLineEndingsOfText(t *testing.T) {
	binary := []byte("\x89PNG\r\n\x1a\n\x00\x00\r\n")
	lf := map[string]string{
		"README.md":         "# Library\n\nLends books.\n",
		"src/loans.go":      "package loans\n\nfunc Due() {}\n",
		"docs/spec/loan.md": "# Loan\n\nType: [Concept](../types/concept.md)\n",
		"no-end.txt":        "one\ntwo",
	}
	lfTree, crlfTree := fstest.MapFS{}, fstest.MapFS{}
	for p, text := range lf {
		lfTree[p] = &fstest.MapFile{Data: []byte(text)}
		crlfTree[p] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(text, "\n", "\r\n"))}
	}
	lfTree["logo.png"] = &fstest.MapFile{Data: binary}
	crlfTree["logo.png"] = &fstest.MapFile{Data: binary}
	hashes := func(fsys fstest.MapFS) map[string]string {
		files, err := Walk(fsys)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, f := range files {
			if out[f], err = HashFile(fsys, f); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	a, b := hashes(lfTree), hashes(crlfTree)
	if len(a) != 5 {
		t.Fatalf("hashed %v", a)
	}
	for p, h := range a {
		if b[p] != h {
			t.Errorf("%s: %s with LF, %s with CRLF", p, h, b[p])
		}
	}
	if a["README.md"] != sha([]byte(lf["README.md"])) {
		t.Error("a text file with LF does not hash as its bytes")
	}
	// A binary file hashes as it is: its CRLF stays.
	if a["logo.png"] != sha(binary) || Hash(binary) == Hash(bytes.ReplaceAll(binary, []byte("\r\n"), []byte("\n"))) {
		t.Error("a binary file's CRLF was changed")
	}
	// Text that is not UTF-8 counts as binary.
	latin := []byte("caf\xe9\r\n")
	if Hash(latin) != sha(latin) {
		t.Error("text that is not UTF-8 was changed")
	}
	// A lone CR stays.
	if Hash([]byte("a\rb")) == Hash([]byte("a\nb")) {
		t.Error("a lone CR was changed")
	}
}
