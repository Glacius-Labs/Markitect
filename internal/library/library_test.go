package library

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// folder is the library as it lies on disk, the files the tool embeds.
func folder() fs.FS { return os.DirFS(".") }

// defaultFolder is the default that ships with Markitect as it lies on disk:
// the spec folder that a project copies, beside the plugin and not in the binary.
func defaultFolder() fs.FS { return os.DirFS(filepath.Join("..", "..", "plugin", "default")) }

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

// The library holds the two folders the tool reads, and no module: the modules
// are the folders of the default.
func TestLibraryHoldsOnlyWhatTheToolReads(t *testing.T) {
	if got := strings.Join(folders(t, folder(), "."), " "); got != "questions references" {
		t.Errorf("folders: %s", got)
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
	for _, dir := range []string{"references", "questions"} {
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
	if n != 2 {
		t.Errorf("%d files", n)
	}
}

// Every embedded file belongs to a module of the default, so that nothing is
// built in that no command reads. A module may have neither, one or both: the
// default holds more modules than the tool has values for.
func TestEveryEmbeddedFileBelongsToAModuleOfTheDefault(t *testing.T) {
	modules := map[string]bool{}
	for _, name := range folders(t, defaultFolder(), ".") {
		modules[name] = true
	}
	for _, dir := range []string{"references", "questions"} {
		entries, err := fs.ReadDir(FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if name := strings.TrimSuffix(e.Name(), ".md"); !modules[name] {
				t.Errorf("%s/%s belongs to no module of the default, which are %v", dir, e.Name(), modules)
			}
		}
	}
}

// Every reference value names a type of its module, and the question catalog
// holds a few questions.
func TestReferencesNameTypesOfTheirModuleAndQuestionsAreFew(t *testing.T) {
	entries, err := fs.ReadDir(FS(), "references")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".md")
		refs, err := fs.ReadFile(FS(), "references/"+e.Name())
		if err != nil {
			t.Fatal(err)
		}
		rows := 0
		for _, line := range strings.Split(string(refs), "\n") {
			cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
			if !strings.HasPrefix(line, "| ") || len(cells) != 3 || strings.TrimSpace(cells[0]) == "Type" {
				continue
			}
			rows++
			if _, err := fs.Stat(defaultFolder(), name+"/"+strings.TrimSpace(cells[0])+".md"); err != nil {
				t.Errorf("%s: reference values for %s, which is no type of the module", name, strings.TrimSpace(cells[0]))
			}
		}
		if rows == 0 {
			t.Errorf("%s: no reference values", name)
		}
	}
	entries, err = fs.ReadDir(FS(), "questions")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		questions, err := fs.ReadFile(FS(), "questions/"+e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(questions), "\n- "); n < 5 || n > 10 {
			t.Errorf("%s: %d questions", e.Name(), n)
		}
	}
}
