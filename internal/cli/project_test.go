package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

// tree lists every file and folder below dir, folders with a trailing slash.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if e.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func TestInitWritesOnlyMarkitectAndTheSpecFolder(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := run("init", "-C", dir)
	if code != ExitOK || out != "created .markitect/\ncreated docs/spec/\n" || errOut != "" {
		t.Fatal(code, out, errOut)
	}
	want := []string{".markitect/", ".markitect/.gitattributes", ".markitect/ignore", ".markitect/spec", ".markitect/version", "docs/", "docs/spec/"}
	if got := tree(t, dir); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("init wrote %v", got)
	}
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, ".markitect", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if read("version") != buildinfo.Version()+"\n" || read("spec") != "docs/spec\n" || read("ignore") != "" ||
		read(".gitattributes") != "state.jsonl merge=union\nhistory.jsonl merge=union\n" {
		t.Error("init wrote other contents")
	}
	before := tree(t, dir)
	if code, out, errOut := run("init", "-C", dir); code != ExitFailed || out != "" || !strings.Contains(errOut, ".markitect/ exists already") {
		t.Error(code, out, errOut)
	}
	if got := tree(t, dir); strings.Join(got, " ") != strings.Join(before, " ") {
		t.Errorf("a refused init changed %v", got)
	}
}

func TestInitWithASpecFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "handbook"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "handbook", "lending.md"), []byte("# Lending\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("init", "-C", dir, `--spec=handbook\`, "--json")
	var doc initResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if code != ExitOK || doc.Kind != "init" || doc.Spec != "handbook" || doc.Version != buildinfo.Version() || strings.Join(doc.Created, " ") != ".markitect/" {
		t.Errorf("got %d %s", code, out)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 1 document\n" {
		t.Error(code, out)
	}
	for _, spec := range []string{"..", "/abs", ".markitect/spec", "."} {
		if code, _, errOut := run("init", "-C", t.TempDir(), "--spec", spec); code != ExitFailed || !strings.Contains(errOut, "spec folder") {
			t.Errorf("%s: %d %q", spec, code, errOut)
		}
	}
	file := t.TempDir()
	if err := os.WriteFile(filepath.Join(file, "spec"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("init", "-C", file, "--spec", "spec"); code != ExitFailed || !strings.Contains(errOut, "is a file") {
		t.Error(code, errOut)
	}
	if got := tree(t, file); strings.Join(got, " ") != "spec" {
		t.Errorf("a refused init left %v", got)
	}
}

// TestFailedInitLeavesNothing fails each step of init in turn and checks that
// the folder is as it was, so that init can run again.
func TestFailedInitLeavesNothing(t *testing.T) {
	steps := 0
	fault = func(int) error { steps++; return nil }
	t.Cleanup(func() { fault = nil })
	if code, _, errOut := run("init", "-C", t.TempDir(), "--spec", "docs/spec/main"); code != ExitOK {
		t.Fatal(errOut)
	}
	if steps < 6 {
		t.Fatalf("only %d steps", steps)
	}
	for k := 1; k <= steps; k++ {
		dir := t.TempDir()
		fault = func(step int) error {
			if step == k {
				return errors.New("injected")
			}
			return nil
		}
		if code, out, errOut := run("init", "-C", dir, "--spec", "docs/spec/main"); code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") {
			t.Errorf("step %d: %d %q %q", k, code, out, errOut)
		}
		if got := tree(t, dir); len(got) != 0 {
			t.Errorf("step %d left %v", k, got)
		}
	}
}

func TestFindProject(t *testing.T) {
	root := filepath.Join("C:", "work")
	if filepath.Separator == '/' {
		root = "/work"
	}
	lending := filepath.Join(root, "lending")
	deep := filepath.Join(lending, "src", "loans")
	for _, tc := range []struct {
		projects []string
		want     string
	}{
		{[]string{lending}, lending},
		{[]string{root, lending}, lending},
		{[]string{root}, root},
		{[]string{deep}, deep},
		{nil, ""},
	} {
		got, ok := findProject(deep, func(dir string) bool {
			for _, p := range tc.projects {
				if p == dir {
					return true
				}
			}
			return false
		})
		if got != tc.want || ok != (tc.want != "") {
			t.Errorf("%v: got %q %v", tc.projects, got, ok)
		}
	}
}

func TestCommandsFindTheProjectUpward(t *testing.T) {
	dir := spec(t, map[string]string{"lending.md": "# Lending\n\nSee [loans](loans.md).\n"})
	sub := filepath.Join(dir, "src", "loans")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	code, out, _ := run("check")
	if code != ExitFound || out != "lending.md:3: link.unresolved: no document at \"loans.md\"\n1 error in 1 document\n" {
		t.Error(code, out)
	}
	if code, out, _ := run("new", filepath.Join("..", "lending"), "loans"); code != ExitFailed || out != "" {
		t.Error("a path below the working directory was taken for one below the spec folder", code, out)
	}
	if code, out, _ := run("show", "lending"); code != ExitOK || !strings.HasPrefix(out, "lending.md@") {
		t.Error(code, out)
	}
	// A missing spec folder is named.
	if err := os.RemoveAll(specOf(dir)); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("check"); code != ExitFailed || !strings.Contains(errOut, "the spec folder docs/spec, which .markitect/spec names, is no folder") {
		t.Error(code, errOut)
	}
}

func TestVersion(t *testing.T) {
	dir := initialized(t)
	code, out, _ := run("version", "-C", dir)
	if code != ExitOK || out != "markitect-next "+buildinfo.Version()+"\nthe project pins "+buildinfo.Version()+"\n" {
		t.Error(code, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".markitect", "version"), []byte("0.1.0\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run("version", "-C", dir); code != ExitFound || !strings.HasSuffix(out, "the project pins 0.1.0, not this version\n") {
		t.Error(code, out)
	}
	_, out, _ = run("version", "-C", dir, "--json")
	var doc versionResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Kind != "version" || doc.Version != buildinfo.Version() || doc.Pinned != "0.1.0" || doc.Matches {
		t.Error(out)
	}
	if code, _, errOut := run("version", "-C", t.TempDir()); code != ExitFailed || !strings.Contains(errOut, "holds no .markitect/version") {
		t.Error(code, errOut)
	}
}

// TestOutsideAProject runs below a .markitect/ folder without a version file,
// as in a home folder that holds the reader's last look, which is no project.
func TestOutsideAProject(t *testing.T) {
	outside := t.TempDir()
	if dir, found := findProject(outside, isProject); found {
		t.Fatalf("%s, above the temporary folder, holds .markitect/version", dir)
	}
	if err := os.MkdirAll(filepath.Join(outside, ".markitect", "last-look"), 0o755); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(outside, "work")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(inside)
	if code, out, _ := run("version"); code != ExitOK || out != "markitect-next "+buildinfo.Version()+"\n" {
		t.Error(code, out)
	}
	if code, _, errOut := run("check"); code != ExitFailed || !strings.Contains(errOut, "upward holds .markitect/version; markitect-next init makes one") {
		t.Error(code, errOut)
	}
}

// link makes link lead to the folder target: a symbolic link, or a junction
// on Windows, where a symbolic link needs a privilege.
func link(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("mklink: %v %s", err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func TestSpecFolderOutsideTheProjectIsRefused(t *testing.T) {
	elsewhere := t.TempDir()
	dir := t.TempDir()
	link(t, elsewhere, filepath.Join(dir, "docs"))
	if code, _, errOut := run("init", "-C", dir); code != ExitFailed || !strings.Contains(errOut, "the spec folder docs/spec lies outside the project") {
		t.Fatal(code, errOut)
	}
	if got := tree(t, elsewhere); len(got) != 0 {
		t.Errorf("a refused init wrote %v outside", got)
	}
	// A link inside the project is fine.
	inside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(inside, "handbook"), 0o755); err != nil {
		t.Fatal(err)
	}
	link(t, filepath.Join(inside, "handbook"), filepath.Join(inside, "docs"))
	if code, _, errOut := run("init", "-C", inside); code != ExitOK {
		t.Fatal(errOut)
	}
	// A spec folder that a link moved outside later stops every command that
	// reads it.
	project := spec(t, map[string]string{"a.md": "# A\n"})
	if err := os.RemoveAll(specOf(project)); err != nil {
		t.Fatal(err)
	}
	link(t, elsewhere, specOf(project))
	for _, args := range [][]string{{"check"}, {"apply", "software"}, {"show", "a"}} {
		if code, _, errOut := run(append(args, "-C", project)...); code != ExitFailed || !strings.Contains(errOut, "lies outside the project") {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
}

func TestReservedSpecFolders(t *testing.T) {
	for spec, want := range map[string]string{".Markitect/x": "lies in .markitect/", "a/.GIT/b": "lies in a .git folder"} {
		if code, _, errOut := run("init", "-C", t.TempDir(), "--spec", spec); code != ExitFailed || !strings.Contains(errOut, want) {
			t.Errorf("%s: %d %q", spec, code, errOut)
		}
	}
}

func TestInitFolderMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := initialized(t)
	info, err := os.Stat(filepath.Join(dir, ".markitect"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf(".markitect/ has mode %v", info.Mode().Perm())
	}
}

func TestProjectFSReachesOutside(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(dir, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{filepath.ToSlash(filepath.Join(dir, "HEAD")), "../HEAD"} {
		host, p, ok := diskProject(project).Outside(name)
		if !ok {
			t.Fatalf("%s: not reached", name)
		}
		if data, err := fs.ReadFile(host, p); err != nil || string(data) != "x" {
			t.Errorf("%s: read %q %v", name, data, err)
		}
	}
}
