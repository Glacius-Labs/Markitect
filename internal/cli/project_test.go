package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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
	if code, out, _ := run("show", "lending"); code != ExitOK || !strings.HasPrefix(out, "lending.md\n") {
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
	if code, _, errOut := run("version", "-C", t.TempDir()); code != ExitFailed || !strings.Contains(errOut, "holds no .markitect/") {
		t.Error(code, errOut)
	}
}

// TestOutsideAProject runs where no folder from the temporary folder upward
// holds .markitect/, and is skipped elsewhere.
func TestOutsideAProject(t *testing.T) {
	outside := t.TempDir()
	if _, found := findProject(outside, isProject); found {
		t.Skip("a folder above the temporary folder holds .markitect/")
	}
	t.Chdir(outside)
	if code, out, _ := run("version"); code != ExitOK || out != "markitect-next "+buildinfo.Version()+"\n" {
		t.Error(code, out)
	}
	if code, _, errOut := run("check"); code != ExitFailed || !strings.Contains(errOut, "upward holds .markitect/; markitect-next init makes one") {
		t.Error(code, errOut)
	}
}
