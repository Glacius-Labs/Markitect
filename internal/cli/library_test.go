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

// The module core as a spec, the way a person copies it: on its own and
// below a spec folder as core/. check finds nothing in either, so the module
// passes the rules of the language itself: a README, and under Contents every
// type that a document outside it may link to.
func TestLibrary(t *testing.T) {
	alone, err := fs.Sub(libraryFolder(), "core")
	if err != nil {
		t.Fatal(err)
	}
	dir := initialized(t)
	copyFromLibrary(t, dir, "core")
	for name, fsys := range map[string]fs.FS{"alone": alone, "in a spec": os.DirFS(specOf(dir))} {
		s, err := lang.Load(fsys)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range lang.Check(s) {
			t.Errorf("%s: %s", name, d)
		}
		if len(s.Documents) != 7 {
			t.Errorf("%s: %d documents", name, len(s.Documents))
		}
	}
	s, _ := lang.Load(alone)
	concept, _ := s.Document("concept.md")
	if n := len(concept.Declared()); n != 4 {
		t.Errorf("concept declares %d sections", n)
	}
}

// The reference values name a value for every type of core, which is every
// document of it but its README.
func TestReferenceValuesCoverEveryType(t *testing.T) {
	refs, err := references()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	err = fs.WalkDir(libraryFolder(), "core", func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() && e.Name() != "README.md" {
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

// A README is never an object and never a type: with a type line it is read
// without it, with an extends line too, and a type line that names a README
// names no type. check reports each at its line, show and table treat the
// documents as untyped.
func TestReadmeIsNoObjectAndNoType(t *testing.T) {
	dir := spec(t, map[string]string{
		"types/README.md":  "# Types\n\n## Contents\n- [Member](member.md)\n",
		"types/member.md":  "# Member\n\nA person.\n\n## Card\nThe number.\n",
		"people/README.md": "# People\n\nType: [Member](../types/member.md)\n\n## Card\nNone.\n",
		"people/ada.md":    "# Ada\n\nType: [Types](../types/README.md)\n\n## Card\n1\n",
		"people/bob.md":    "# Bob\n\nType: [Member](../types/member.md)\n\n## Card\n2\n",
	})
	code, out, _ := run("check", "-C", dir)
	want := "people/README.md:3: readme.type: a README has no type line\n" +
		"people/ada.md:3: readme.type: \"types/README.md\" is a README, which is no type\n" +
		"2 errors in 5 documents\n"
	if code != ExitFound || out != want {
		t.Errorf("check: %d\n%s\nwant\n%s", code, out, want)
	}
	_, out, _ = run("check", "-C", dir, "--json")
	if !strings.Contains(out, "\"file\": \"people/README.md\",\n      \"line\": 3,\n      \"code\": \"readme.type\",\n      \"message\": \"a README has no type line\"") {
		t.Errorf("check --json: %s", out)
	}
	_, out, _ = run("show", "-C", dir, "people/README")
	if !strings.HasPrefix(out, "people/README.md\nTitle: People\nType: (none)\n") {
		t.Errorf("show README: %s", out)
	}
	_, out, _ = run("show", "-C", dir, "people/ada", "--json")
	if !strings.Contains(out, "\"type_state\": \"invalid\"") {
		t.Errorf("show of a document naming a README: %s", out)
	}
	_, out, _ = run("table", "-C", dir, "types/member")
	if out != "| Title | Card |\n| --- | --- |\n| Bob | 2 |\n" {
		t.Errorf("table of the type: %s", out)
	}
	_, out, _ = run("show", "-C", dir, "types/README")
	if strings.Contains(out, "Documents:") {
		t.Errorf("show of a README lists objects: %s", out)
	}
}
