package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// defaultFolder is the default as plain files, the way a person finds it in the
// repository: the whole spec folder that a project copies, with the module
// core, the way of working and the modules that start empty.
func defaultFolder() fs.FS { return os.DirFS(filepath.Join("..", "..", "plugin", "default")) }

// copyFromDefault copies folders of the default, such as core, to the same
// paths below the spec folder of a project. The folder "." copies the whole
// default.
func copyFromDefault(t *testing.T, project string, folders ...string) {
	t.Helper()
	for _, folder := range folders {
		err := fs.WalkDir(defaultFolder(), folder, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			data, err := fs.ReadFile(defaultFolder(), p)
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

// documentsOf counts the documents of a folder of the default.
func documentsOf(t *testing.T, folder string) int {
	t.Helper()
	n := 0
	err := fs.WalkDir(defaultFolder(), folder, func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() && strings.HasSuffix(p, ".md") {
			n++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// The default as a spec, the way a person copies it: the module core on its own,
// the whole default on its own and the whole default as the spec folder of a
// project. check finds nothing in any of them, so the default passes the rules of
// the language itself: a README in every folder, and under Contents every type
// that a document outside it may link to.
func TestTheDefaultChecksClean(t *testing.T) {
	core, err := fs.Sub(defaultFolder(), "core")
	if err != nil {
		t.Fatal(err)
	}
	dir := initialized(t)
	copyFromDefault(t, dir, ".")
	cases := []struct {
		name string
		fsys fs.FS
		want int
	}{
		{"core alone", core, documentsOf(t, "core")},
		{"the default alone", defaultFolder(), documentsOf(t, ".")},
		{"the default as a spec", os.DirFS(specOf(dir)), documentsOf(t, ".")},
	}
	for _, c := range cases {
		s, err := lang.Load(c.fsys)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range lang.Check(s) {
			t.Errorf("%s: %s", c.name, d)
		}
		if len(s.Documents) != c.want {
			t.Errorf("%s: %d documents, want %d", c.name, len(s.Documents), c.want)
		}
	}
	if n := documentsOf(t, "core"); n != 14 {
		t.Errorf("core holds %d documents, its README and thirteen types", n)
	}
	s, _ := lang.Load(core)
	concept, _ := s.Document("concept.md")
	if n := len(concept.Declared()); n != 4 {
		t.Errorf("concept declares %d sections", n)
	}
}

// The default holds documents and nothing else: only Markdown files, and no
// .markitect folder, which belongs to the project that copies it.
func TestTheDefaultHoldsOnlyMarkdown(t *testing.T) {
	n := 0
	err := fs.WalkDir(defaultFolder(), ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && strings.HasPrefix(e.Name(), ".") && p != "." {
			t.Errorf("%s is a hidden folder", p)
		}
		if !e.IsDir() {
			n++
			if !strings.HasSuffix(p, ".md") {
				t.Errorf("%s is no Markdown file", p)
			}
			data, err := fs.ReadFile(defaultFolder(), p)
			if err != nil {
				return err
			}
			if text := string(data); !strings.HasPrefix(text, "# ") || strings.Contains(text, "\r") || !strings.HasSuffix(text, "\n") {
				t.Errorf("%s is not a document with a title, LF line ends and a final line break", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != documentsOf(t, ".") || n == 0 {
		t.Errorf("%d files", n)
	}
}

// The reference values name a type of core each, and every one of them is a
// document of core; the catalog stays as it is and covers six of its types.
func TestReferenceValuesNameTypesOfCore(t *testing.T) {
	refs, err := references()
	if err != nil {
		t.Fatal(err)
	}
	for key := range refs {
		module, name, _ := strings.Cut(key, "/")
		if module != "core" {
			t.Errorf("reference values for %s, which is in no module of the default but core", key)
		}
		if _, err := fs.Stat(defaultFolder(), module+"/"+name+".md"); err != nil {
			t.Errorf("reference values for %s, which is no type of the default", key)
		}
		if _, ok := refs.For(module + "/" + name + ".md"); !ok {
			t.Errorf("no reference value found for %s", key)
		}
	}
	if len(refs) != 6 {
		t.Errorf("%d types with reference values", len(refs))
	}
}

// A project whose spec folder is the whole default: every command that reads a
// spec runs on it, in text and in JSON, and prints what it should. The question
// catalog speaks of the module core alone, however many modules the default has.
func TestTheCommandsRunOnTheWholeDefault(t *testing.T) {
	dir := initialized(t)
	copyFromDefault(t, dir, ".")
	total := documentsOf(t, ".")
	names := "way-of-working/work/names"

	// exec runs a command that must succeed and print nothing on stderr.
	exec := func(args ...string) string {
		t.Helper()
		code, out, errOut := run(append(args, "-C", dir)...)
		if code != ExitOK || errOut != "" {
			t.Fatalf("%v: exit %d\n%s%s", args, code, out, errOut)
		}
		return out
	}
	asJSON := func(args ...string) map[string]any {
		t.Helper()
		var doc map[string]any
		out := exec(append(args, "--json")...)
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("%v --json: %v\n%.300s", args, err, out)
		}
		return doc
	}

	if out := exec("check"); out != fmt.Sprintf("0 errors in %d documents\n", total) {
		t.Errorf("check: %s", out)
	}
	if doc := asJSON("check"); doc["kind"] != "check" || doc["documents"] != float64(total) || len(doc["diagnostics"].([]any)) != 0 {
		t.Errorf("check --json: %v", doc)
	}

	out := exec("table")
	if lines := strings.Count(out, "\n"); lines != total+2 || !strings.Contains(out, "| core/rule.md |  | Rule |\n") || !strings.Contains(out, "| way-of-working/work/names.md | core/rule.md | Names in the work |\n") {
		t.Errorf("table: %d lines\n%.400s", lines, out)
	}
	if doc := asJSON("table"); doc["kind"] != "table" {
		t.Errorf("table --json: %.300v", doc)
	}
	if out := exec("table", "core/rule"); !strings.HasPrefix(out, "| Title | Applies to | Rule |") || !strings.Contains(out, "| Names in the work |") {
		t.Errorf("table of a type: %.400s", out)
	}

	topic := exec("topic")
	if !strings.HasPrefix(topic, "# Signals of the spec\n") {
		t.Errorf("topic: %.300s", topic)
	}
	if all := exec("topic", "--all"); !strings.HasPrefix(all, "# Signals of the spec\n") || len(all) < len(topic) {
		t.Errorf("topic --all: %.300s", all)
	}
	if doc := asJSON("topic"); doc["kind"] != "topic" || doc["topic_kind"] != "spec" || len(doc["signals"].(map[string]any)) == 0 {
		t.Errorf("topic --json: %.300v", doc)
	}
	asJSON("topic", "--all")
	if doc := asJSON("topic", "core/rule"); doc["topic_kind"] != "type" || len(doc["typed_documents"].([]any)) == 0 {
		t.Errorf("topic of a type --json: %.300v", doc)
	}
	if out := exec("topic", names); !strings.Contains(out, "way-of-working/work/names.md") {
		t.Errorf("topic of a document: %.300s", out)
	}

	if out := exec("questions"); !strings.HasPrefix(out, "## core\n\n1. ") || strings.Contains(out, "\n## ") {
		t.Errorf("questions: %s", out)
	}
	if doc := asJSON("questions"); len(doc["modules"].([]any)) != 1 {
		t.Errorf("questions --json: %v", doc)
	}

	if out := exec("show", names); !strings.HasPrefix(out, "way-of-working/work/names.md\nTitle: Names in the work\nType: core/rule.md\n") || !strings.Contains(out, "Links to:\n") {
		t.Errorf("show: %s", out)
	}
	if doc := asJSON("show", names); doc["kind"] != "show" || doc["title"] != "Names in the work" {
		t.Errorf("show --json: %.300v", doc)
	}
}

// A project laid out by hand, the way a person does: core copied into the spec
// folder, and a module of their own whose documents are of core's types. The
// questions of core are found by the folder name core directly below the spec
// folder, and so are core's reference values, which topic shows beside the
// numbers of a document; a module of the project's own has neither.
func TestAHandLaidProjectHasTheQuestionsAndReferenceValuesOfCore(t *testing.T) {
	dir := initialized(t)
	copyFromDefault(t, dir, "core")
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
	copyFromDefault(t, moved, "core")
	if err := os.Rename(filepath.Join(specOf(moved), "core"), filepath.Join(specOf(moved), "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run("questions", "-C", moved); code != ExitOK || !strings.Contains(out, "The spec has no module of the default, so there are no questions.") {
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
