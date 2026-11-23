package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// libraryFolder is the library as plain files, the way a person finds it: the
// module core, which they copy into a spec by hand.
func libraryFolder() fs.FS { return os.DirFS(filepath.Join("..", "library")) }

// copyFromLibrary copies folders of the library, such as core, to the same
// paths below the spec folder of a project.
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

// The library as one spec: the module core, whose types are well-formed, and
// the reference values and question catalogs beside it, which are plain
// documents. check finds nothing in it.
func TestLibrary(t *testing.T) {
	s, err := lang.Load(libraryFolder())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range lang.Check(s) {
		t.Error(d)
	}
	core := 0
	for _, d := range s.Documents {
		if strings.HasPrefix(d.Path, "core/") {
			core++
		}
	}
	if core != 6 {
		t.Errorf("core holds %d documents", core)
	}
	concept, _ := s.Document("core/concept.md")
	if n := len(concept.Declared()); n != 4 {
		t.Errorf("concept declares %d sections", n)
	}
}

// The reference values name a value for every type of core.
func TestReferenceValuesCoverEveryType(t *testing.T) {
	refs, err := references()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	err = fs.WalkDir(libraryFolder(), "core", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			n++
			if _, ok := refs.For(p); !ok {
				t.Errorf("no reference value for %s", p)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Errorf("%d types", n)
	}
}

// A project laid out by hand, the way a person does: core copied into the spec
// folder, and a module of their own whose documents are of core's types. The
// questions of core are found by the folder name core directly below the spec
// folder, and so are core's reference values, which topic shows beside the
// numbers of a document; a module of the project's own has neither.
func TestAHandLaidProjectHasTheQuestionsAndReferenceValuesOfCore(t *testing.T) {
	dir := initialized(t)
	copyFromLibrary(t, dir, "core")
	write(t, dir, "docs/spec/lending/README.md", "# Lending\n\nHow books are lent.\n")
	write(t, dir, "docs/spec/lending/loan.md", "# Loan\n\nType: [Concept](../core/concept.md)\n\n## Definition\nA member keeps a copy for three weeks. The copy goes back to the shelf. A fee is due after that. The librarian waives it.\n\n## Rules (optional)\nA copy is on loan to one member. A member holds five loans. Nothing else.\n")
	write(t, dir, "docs/spec/lending/own-type.md", "# Own type\n\nSomething of the project.\n\n## Part\nIt.\n")
	write(t, dir, "docs/spec/lending/thing.md", "# Thing\n\nType: [Own type](own-type.md)\n\n## Part\nA part.\n")

	code, out, errOut := run("check", "-C", dir)
	if code != ExitOK {
		t.Fatalf("check: %d\n%s%s", code, out, errOut)
	}

	code, out, _ = run("questions", "-C", dir)
	if code != ExitOK || !strings.HasPrefix(out, "## core\n\n1. Which role is responsible for each rule, and does that role exist?\n") || strings.Contains(out, "## lending") {
		t.Errorf("questions: %d\n%s", code, out)
	}
	_, out, _ = run("questions", "-C", dir, "--json")
	var q struct {
		Kind    string
		Modules []struct {
			Module    string
			Questions []string
		}
	}
	if err := json.Unmarshal([]byte(out), &q); err != nil {
		t.Fatal(err)
	}
	if q.Kind != "questions" || len(q.Modules) != 1 || q.Modules[0].Module != "core" || len(q.Modules[0].Questions) != 6 {
		t.Errorf("questions --json: %s", out)
	}

	// The loan has seven statements, and core's reference values for a concept
	// are 1 to 5 statements of 5 to 25 words, which topic shows beside them. A
	// type of the project's own has no reference values.
	_, out, _ = run("topic", "-C", dir, "--json")
	var topic struct {
		Signals map[string][]struct {
			File   string
			Detail string
		}
	}
	if err := json.Unmarshal([]byte(out), &topic); err != nil {
		t.Fatal(err)
	}
	details := map[string][]string{}
	for _, signal := range []string{"size", "complexity"} {
		for _, f := range topic.Signals[signal] {
			details[signal+" "+f.File] = append(details[signal+" "+f.File], f.Detail)
		}
	}
	wantSize := "7 statements (documents of type core/concept.md: median 7, range 7–7, 1 document; reference 1–5)"
	if got := details["size lending/loan.md"]; len(got) != 1 || got[0] != wantSize {
		t.Errorf("size of the loan: %v, want %s", got, wantSize)
	}
	if got := details["complexity lending/loan.md"]; len(got) != 7 || !strings.HasSuffix(strings.Split(got[0], ", 0 terms")[0], "; reference 5–25)") {
		t.Errorf("complexity of the loan: %v", got)
	}
	if got := details["size lending/thing.md"]; len(got) != 1 || strings.Contains(got[0], "reference") {
		t.Errorf("size of a document of the project's own type: %v", got)
	}

	// The folder of core is found only directly below the spec folder.
	moved := initialized(t)
	copyFromLibrary(t, moved, "core")
	if err := os.Rename(filepath.Join(specOf(moved), "core"), filepath.Join(specOf(moved), "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run("questions", "-C", moved); code != ExitOK || !strings.Contains(out, "The spec has no module of the library, so there are no questions.") {
		t.Errorf("a folder of another name: %d %s", code, out)
	}
}
