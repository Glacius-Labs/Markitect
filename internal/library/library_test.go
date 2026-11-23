package library

import (
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
)

// folder is the library as it lies on disk, packages and templates included.
func folder() fs.FS { return os.DirFS(".") }

func folders(t *testing.T, fsys fs.FS, dir string) []string {
	t.Helper()
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func TestLibraryHoldsPackagesAndTemplates(t *testing.T) {
	if got := strings.Join(folders(t, folder(), "packages"), " "); got != "company core software" {
		t.Errorf("packages: %s", got)
	}
	if got := strings.Join(folders(t, folder(), "templates"), " "); got != "architecture company product project software" {
		t.Errorf("templates: %s", got)
	}
}

// The binary holds what the commands read at run time, and nothing else.
func TestTheBinaryEmbedsOnlyWhatTheCommandsRead(t *testing.T) {
	var got []string
	err := fs.WalkDir(FS(), ".", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			got = append(got, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "questions/company.md questions/core.md questions/software.md references/company.md references/core.md references/software.md"
	if strings.Join(got, " ") != want {
		t.Errorf("embedded %v", got)
	}
}

func TestEveryFileIsADocumentWithATitleAndLineFeeds(t *testing.T) {
	n := 0
	for _, dir := range []string{"packages", "templates", "references", "questions"} {
		err := fs.WalkDir(folder(), dir, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			n++
			data, err := fs.ReadFile(folder(), p)
			if err != nil {
				return err
			}
			text := string(data)
			if path.Ext(p) != ".md" || !strings.HasPrefix(text, "# ") || strings.Contains(text, "\r") || !strings.HasSuffix(text, "\n") {
				t.Errorf("%s is not a document with a title, LF line ends and a final line break", p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if n < 40 {
		t.Errorf("only %d files", n)
	}
}

func TestEveryPackageHasReferencesAndQuestions(t *testing.T) {
	for _, name := range folders(t, folder(), "packages") {
		refs, err := fs.ReadFile(FS(), "references/"+name+".md")
		if err != nil {
			t.Fatal(err)
		}
		types, _ := fs.Glob(folder(), "packages/"+name+"/*.md")
		if rows := strings.Count(string(refs), "\n| ") - 1; rows != len(types) {
			t.Errorf("%s: %d reference rows for %d types", name, rows, len(types))
		}
		questions, err := fs.ReadFile(FS(), "questions/"+name+".md")
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(questions), "\n- "); n < 5 || n > 10 {
			t.Errorf("%s: %d questions", name, n)
		}
	}
}

// Every embedded file belongs to a package, so that nothing is built in that no
// command reads.
func TestEveryEmbeddedFileBelongsToAPackage(t *testing.T) {
	packages := folders(t, folder(), "packages")
	for _, dir := range []string{"references", "questions"} {
		entries, err := fs.ReadDir(FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
		if strings.Join(names, " ") != strings.Join(packages, " ") {
			t.Errorf("%s holds %v, the packages are %v", dir, names, packages)
		}
	}
}
