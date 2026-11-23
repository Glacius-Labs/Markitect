package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

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
	if code, out, _ := run("show", "lending"); code != ExitOK || !strings.HasPrefix(out, "lending.md\nTitle: Lending\n") {
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

// version prints the tool's version and build, in a project or outside one,
// and reads nothing of the project.
func TestVersion(t *testing.T) {
	want := "markitect " + buildinfo.Build() + "\n"
	for name, dir := range map[string]string{"in a project": initialized(t), "outside": t.TempDir()} {
		if code, out, errOut := run("version", "-C", dir); code != ExitOK || out != want || errOut != "" {
			t.Error(name, code, out, errOut)
		}
		_, out, _ := run("version", "-C", dir, "--json")
		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatal(err)
		}
		if len(doc) != 4 || doc["format_version"] != float64(1) || doc["kind"] != "version" || doc["version"] != buildinfo.Version() || doc["build"] != buildinfo.Build() {
			t.Error(name, out)
		}
	}
}

// The license notices print anywhere, inside a project or not.
func TestLicenses(t *testing.T) {
	if code, out, _ := run("licenses"); code != ExitOK || out != buildinfo.Notices || !strings.Contains(out, "goldmark") {
		t.Error(code, out)
	}
	_, out, _ := run("licenses", "--json")
	var doc licensesResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil || doc.Kind != "licenses" || doc.Notices != buildinfo.Notices {
		t.Error(err, out)
	}
}

// TestOutsideAProject runs below a .markitect/ folder without a spec file, as
// in a home folder that holds other things, which is no project. A .markitect/
// folder of a project holds nothing else the tool needs.
func TestOutsideAProject(t *testing.T) {
	outside := t.TempDir()
	if dir, found := findProject(outside, isProject); found {
		t.Fatalf("%s, above the temporary folder, holds .markitect/spec", dir)
	}
	if err := os.MkdirAll(filepath.Join(outside, ".markitect", "last-look"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, outside, ".markitect/version", "0.1.0\n")
	inside := filepath.Join(outside, "work")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(inside)
	if code, out, _ := run("version"); code != ExitOK || out != "markitect "+buildinfo.Build()+"\n" {
		t.Error(code, out)
	}
	if code, _, errOut := run("check"); code != ExitFailed || !strings.Contains(errOut, "upward holds .markitect/spec; that file holds one line, the path of the spec folder") {
		t.Error(code, errOut)
	}
	// With the spec file, the other files of .markitect/ are no concern.
	write(t, outside, ".markitect/spec", "docs\n")
	write(t, outside, ".markitect/state.jsonl", "not json\n")
	write(t, outside, "docs/a.md", "# A\n")
	if code, out, errOut := run("check"); code != ExitOK || out != "0 errors in 1 document\n" {
		t.Error(code, out, errOut)
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
	write(t, elsewhere, "a.md", "# A\n")
	// A link inside the project is fine.
	inside := t.TempDir()
	markitectFile(t, inside)
	if err := os.MkdirAll(filepath.Join(inside, "handbook"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, inside, "handbook/spec/a.md", "# A\n")
	link(t, filepath.Join(inside, "handbook"), filepath.Join(inside, "docs"))
	if code, _, errOut := run("check", "-C", inside); code != ExitOK {
		t.Fatal(errOut)
	}
	// A spec folder that a link moved outside stops every command that reads
	// it.
	project := spec(t, map[string]string{"a.md": "# A\n"})
	if err := os.RemoveAll(specOf(project)); err != nil {
		t.Fatal(err)
	}
	link(t, elsewhere, specOf(project))
	for _, args := range [][]string{{"check"}, {"show", "a"}, {"table"}, {"topic"}, {"questions"}, {"diff", project}} {
		if code, _, errOut := run(append(args, "-C", project)...); code != ExitFailed || !strings.Contains(errOut, "lies outside the project") {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
}

func TestReservedSpecFolders(t *testing.T) {
	for spec, want := range map[string]string{".Markitect/x": "lies in .markitect/", "a/.GIT/b": "lies in a .git folder", "../up": "not a path below the project folder"} {
		dir := t.TempDir()
		write(t, dir, ".markitect/spec", spec+"\n")
		if code, _, errOut := run("check", "-C", dir); code != ExitFailed || !strings.Contains(errOut, want) {
			t.Errorf("%s: %d %q", spec, code, errOut)
		}
	}
}
