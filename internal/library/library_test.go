package library

import (
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
)

// folder is the library as it lies on disk, the module included.
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

// modules lists the modules of the library: its folders but the two the tool
// reads.
func modules(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, name := range folders(t, folder(), ".") {
		if name != "references" && name != "questions" {
			out = append(out, name)
		}
	}
	return out
}

// The library holds one module, core, beside the folders the tool reads.
func TestLibraryHoldsTheModuleCore(t *testing.T) {
	if got := strings.Join(folders(t, folder(), "."), " "); got != "core questions references" {
		t.Errorf("folders: %s", got)
	}
	types, _ := fs.Glob(folder(), "core/*.md")
	if len(types) != 7 {
		t.Errorf("core holds %v, its README and six types", types)
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
	want := "questions/core.md references/core.md"
	if strings.Join(got, " ") != want {
		t.Errorf("embedded %v", got)
	}
}

func TestEveryFileIsADocumentWithATitleAndLineFeeds(t *testing.T) {
	n := 0
	for _, dir := range []string{"core", "references", "questions"} {
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
	if n != 9 {
		t.Errorf("%d files", n)
	}
}

func TestEveryModuleHasReferencesAndQuestions(t *testing.T) {
	for _, name := range modules(t) {
		refs, err := fs.ReadFile(FS(), "references/"+name+".md")
		if err != nil {
			t.Fatal(err)
		}
		// Every document of a module but its README is a type.
		types, _ := fs.Glob(folder(), name+"/*.md")
		if rows := strings.Count(string(refs), "\n| ") - 1; rows != len(types)-1 {
			t.Errorf("%s: %d reference rows for %d types", name, rows, len(types)-1)
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

// Every embedded file belongs to a module, so that nothing is built in that no
// command reads.
func TestEveryEmbeddedFileBelongsToAModule(t *testing.T) {
	want := modules(t)
	for _, dir := range []string{"references", "questions"} {
		entries, err := fs.ReadDir(FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
		if strings.Join(names, " ") != strings.Join(want, " ") {
			t.Errorf("%s holds %v, the modules are %v", dir, names, want)
		}
	}
}
