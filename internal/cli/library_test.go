package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// libraryFolder is the library as plain files, the way a person finds it: the
// packages and templates they copy into a spec by hand.
func libraryFolder() fs.FS { return os.DirFS(filepath.Join("..", "library")) }

// copyFromLibrary copies folders of the library, such as packages or
// templates/product, to the same paths below the spec folder of a project.
func copyFromLibrary(t *testing.T, project string, folders ...string) {
	t.Helper()
	for _, folder := range folders {
		err := fs.WalkDir(libraryFolder(), folder, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			data, err := fs.ReadFile(libraryFolder(), p)
			if err != nil {
				return err
			}
			write(t, project, "docs/spec/"+p, string(data))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// templates lists the templates of the library: the folders below templates/.
func templates(t *testing.T) []string {
	t.Helper()
	entries, err := fs.ReadDir(libraryFolder(), "templates")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// A template laid beside the packages it uses, in a spec folder that holds
// packages/ and templates/NAME/, loads, and check finds in it only what an
// unfilled template has: empty sections. The commands that read the spec
// work on it.
func TestEveryTemplateLoadsBesideThePackages(t *testing.T) {
	names := templates(t)
	if len(names) < 5 {
		t.Fatalf("templates %v", names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir := initialized(t)
			copyFromLibrary(t, dir, "packages", "templates/"+name)
			s, err := lang.Load(os.DirFS(specOf(dir)))
			if err != nil {
				t.Fatal(err)
			}
			filled := 0
			for _, d := range s.Documents {
				if strings.HasPrefix(d.Path, "templates/"+name+"/") {
					filled++
				}
			}
			if filled == 0 {
				t.Fatalf("template %s holds no document", name)
			}
			for _, d := range lang.Check(s) {
				if d.Code != lang.CodeSectionEmpty || !strings.HasPrefix(d.Path, "templates/"+name+"/") {
					t.Errorf("%s: %s", name, d)
				}
			}
			for _, args := range [][]string{{"check"}, {"table"}, {"topic"}, {"questions"}} {
				if code, _, errOut := run(append(args, "-C", dir)...); code == ExitFailed {
					t.Errorf("%v: %s", args, errOut)
				}
			}
		})
	}
}

// The packages and templates of the library, as one spec: only the templates
// have empty sections.
func TestLibrary(t *testing.T) {
	s, err := lang.Load(libraryFolder())
	if err != nil {
		t.Fatal(err)
	}
	var packages []string
	for _, d := range s.Documents {
		if strings.HasPrefix(d.Path, "packages/") {
			packages = append(packages, strings.Split(d.Path, "/")[1])
		}
	}
	if got := strings.Join(unique(packages), " "); got != "company core software" {
		t.Errorf("packages %s", got)
	}
	for _, d := range lang.Check(s) {
		if d.Code != lang.CodeSectionEmpty || !strings.HasPrefix(d.Path, "templates/") {
			t.Error(d)
		}
	}
	quality, _ := s.Document("packages/software/quality.md")
	if n := len(quality.Declared()); n != 10 {
		t.Errorf("quality declares %d sections", n)
	}
}

func unique(list []string) []string {
	var out []string
	for _, s := range list {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}

// The reference values name a value for every type of every package.
func TestReferenceValuesCoverEveryType(t *testing.T) {
	refs, err := references()
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(libraryFolder(), "packages", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			if _, ok := refs.For(p); !ok {
				t.Errorf("no reference value for %s", p)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
