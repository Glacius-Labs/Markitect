package state

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// workProject is a project whose spec has a type of a story, a rule and some
// code, with files of the work beside them.
func workProject(t *testing.T, work map[string]string) *Project {
	t.Helper()
	files := fstest.MapFS{
		".markitect/version":                  {Data: []byte("1.0.0\n")},
		".markitect/spec":                     {Data: []byte("docs/spec\n")},
		"docs/spec/packages/factory/story.md": {Data: []byte("# Story\n\nOne piece of work.\n\n## Goal\nWhat it reaches.\n\n## Notes (optional)\nAnything else.\n")},
		"docs/spec/lending/renew.md":          {Data: []byte("# Renew\n\nA member renews a loan.\n")},
		"src/renew.go":                        {Data: []byte("package lending\n")},
	}
	for p, text := range work {
		files[p] = &fstest.MapFile{Data: []byte(text)}
	}
	p, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func workDiagnostics(t *testing.T, work map[string]string) []string {
	t.Helper()
	w, err := ReadWork(workProject(t, work))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range w.Diagnostics {
		out = append(out, d.String())
	}
	return out
}

func TestWorkDocumentsPassWithLinksToExistingFilesAndFolders(t *testing.T) {
	p := workProject(t, map[string]string{
		".markitect/plan/lending/renew-online/renew-a-loan.md": "# Renew a loan\n\nType: [Story](../../../../docs/spec/packages/factory/story.md)\n\n## Goal\nMembers renew online, see [the rule](../../../../docs/spec/lending/renew.md) and [the code](../../../../src/renew.go).\n",
		".markitect/context/build/checkpoint.md":               "# Checkpoint\n\nContinue with [the plan item](../../plan/lending/renew-online/renew-a-loan.md).\n",
		".markitect/context/capture/.raw/transcript.md":        "# Raw\n\nSee [nothing](missing.md).\n",
		".markitect/context/capture/note.md":                   "# A dropped note\n\nNo type, no links.\n",
		".markitect/context/capture/transcript.txt":            "see [nothing](missing.md)\n",
		".markitect/findings/pending/.gitkeep":                 "",
		".markitect/briefing/folder-link.md":                   "# Briefing\n\nThe [areas](../../docs/spec/lending/) an [image](../../src/renew.go) and the [project](../../).\n",
	})
	w, err := ReadWork(p)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, d := range w.Documents {
		paths = append(paths, d.Path)
	}
	want := ".markitect/briefing/folder-link.md .markitect/context/build/checkpoint.md .markitect/context/capture/note.md .markitect/plan/lending/renew-online/renew-a-loan.md"
	if strings.Join(paths, " ") != want {
		t.Errorf("documents %v", paths)
	}
	if len(w.Diagnostics) > 0 {
		t.Errorf("diagnostics %v", w.Diagnostics)
	}
}

func TestWorkDefects(t *testing.T) {
	got := workDiagnostics(t, map[string]string{
		".markitect/plan/a-missing-section.md":   "# Story\n\nType: [Story](../../docs/spec/packages/factory/story.md)\n",
		".markitect/plan/b-empty-section.md":     "# Story\n\nType: [Story](../../docs/spec/packages/factory/story.md)\n\n## Goal\n\n<!-- later -->\n",
		".markitect/plan/c-dead-link.md":         "# Dead\n\nSee [it](../../docs/spec/lending/ghost.md) and [code](../../src/ghost.go) and [here](#top).\n",
		".markitect/plan/d-no-type.md":           "# Lost\n\nType: [Story](../../docs/spec/packages/factory/ghost.md)\n",
		".markitect/plan/e-no-title.md":          "Just text, see [the rule](../../docs/spec/lending/renew.md).\n",
		".markitect/plan/f-outside.md":           "# Outside\n\nSee [up](../../../x.md) and [cased](../../SRC/renew.go).\n",
		".markitect/plan/g-bad-type-line.md":     "# Bad\n\nType: no link here\n",
		".markitect/briefing/h-stray.md":         "# Stray\n\nText.\n\nType: [Story](../../docs/spec/packages/factory/story.md)\n",
		".markitect/briefing/i-repeated-link.md": "# Repeat\n\n[a](../../src/ghost.go) [b](../../src/ghost.go)\n",
	})
	want := []string{
		`.markitect/briefing/h-stray.md:5: type.line: a type line stands only on the first line after the title`,
		`.markitect/briefing/i-repeated-link.md:3: link.unresolved: no file at "../../src/ghost.go"`,
		`.markitect/plan/a-missing-section.md:3: section.missing: missing section "Goal" required by packages/factory/story.md`,
		`.markitect/plan/b-empty-section.md:5: section.empty: empty section "Goal"`,
		`.markitect/plan/c-dead-link.md:3: link.unresolved: no file at "../../docs/spec/lending/ghost.md"`,
		`.markitect/plan/c-dead-link.md:3: link.unresolved: no file at "../../src/ghost.go"`,
		`.markitect/plan/d-no-type.md:3: type.unresolved: no document at "../../docs/spec/packages/factory/ghost.md"`,
		`.markitect/plan/e-no-title.md:1: title: missing level-1 title`,
		`.markitect/plan/f-outside.md:3: link.unresolved: no file at "../../../x.md"`,
		`.markitect/plan/f-outside.md:3: link.unresolved: no file at "../../SRC/renew.go"`,
		`.markitect/plan/g-bad-type-line.md:3: type.line: type line must be exactly one link to a document`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestWorkTypeMustBeADocumentOfTheSpec(t *testing.T) {
	// A type outside the spec folder is no type of the spec, even where the
	// file exists.
	got := workDiagnostics(t, map[string]string{
		".markitect/plan/a.md":  "# A\n\nType: [Code](../../src/renew.go)\n",
		".markitect/plan/b.md":  "# B\n\nType: [Note](a.md)\n",
		"docs/spec/ignored.txt": "x",
	})
	for _, line := range got {
		if !strings.Contains(line, lang.CodeTypeLine) && !strings.Contains(line, lang.CodeTypeUnresolved) {
			t.Errorf("unexpected %s", line)
		}
	}
	if len(got) != 2 {
		t.Errorf("got %v", got)
	}
}

func TestMissingWorkFoldersAreNoError(t *testing.T) {
	w, err := ReadWork(workProject(t, nil))
	if err != nil || len(w.Documents) != 0 || len(w.Diagnostics) != 0 {
		t.Errorf("%v %+v", err, w)
	}
}

func TestWorkChangesByPath(t *testing.T) {
	doc := func(p, text string) *lang.Document { return lang.Parse(p, []byte(text)) }
	older := []*lang.Document{doc("a.md", "# A\n"), doc("b.md", "# B\n"), doc("c.md", "# C\n")}
	newer := []*lang.Document{doc("a.md", "# A\n"), doc("b.md", "# B\n\nmore\n"), doc("d.md", "# D\n")}
	var got []string
	for _, c := range compareWork(older, newer) {
		got = append(got, c.Kind+" "+c.Path)
	}
	if strings.Join(got, ", ") != "changed b.md, gone c.md, new d.md" {
		t.Error(got)
	}
}

// A type link whose case differs from the file's names no type of the spec,
// as a link of a spec document would not.
func TestWorkTypeLinkIsCaseExact(t *testing.T) {
	got := workDiagnostics(t, map[string]string{
		".markitect/plan/a.md": "# A\n\nType: [Story](../../docs/spec/packages/factory/Story.md)\n",
	})
	want := `.markitect/plan/a.md:3: type.unresolved: no document at "../../docs/spec/packages/factory/Story.md"`
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %v", got)
	}
}

// failingRead is a project file system in which one file cannot be read.
type failingRead struct {
	fstest.MapFS
	bad string
}

func (f failingRead) ReadFile(name string) ([]byte, error) {
	if name == f.bad {
		return nil, errors.New("denied")
	}
	return f.MapFS.ReadFile(name)
}

// The lenient reading leaves out what it cannot read and names it; the strict
// one, which check uses, fails.
func TestReadWorkLenientNamesWhatItCannotRead(t *testing.T) {
	p := workProject(t, map[string]string{
		".markitect/plan/a.md": "# A\n",
		".markitect/plan/b.md": "# B\n",
	})
	p.FS = failingRead{p.FS.(fstest.MapFS), ".markitect/plan/b.md"}
	if _, err := ReadWork(p); err == nil {
		t.Error("the strict reading did not fail")
	}
	w, err := ReadWorkLenient(p)
	if err != nil || len(w.Documents) != 1 || w.Documents[0].Path != ".markitect/plan/a.md" || strings.Join(w.Unreadable, " ") != ".markitect/plan/b.md" {
		t.Errorf("%v %+v", err, w)
	}
}

// What the .gitignore files exclude is not read, whatever its depth.
func TestWorkLeavesOutWhatGitignoreExcludes(t *testing.T) {
	p := workProject(t, map[string]string{
		".gitignore":                         "*.draft.md\n",
		".markitect/.gitignore":              "local/\n",
		".markitect/plan/idea.draft.md":      "no title\n",
		".markitect/context/local/n.md":      "no title\n",
		".markitect/context/kept/n.md":       "# Kept\n",
		".markitect/context/kept/.gitignore": "/private.md\n",
		".markitect/context/kept/private.md": "no title\n",
	})
	w, err := ReadWork(p)
	if err != nil || len(w.Documents) != 1 || w.Documents[0].Path != ".markitect/context/kept/n.md" || len(w.Diagnostics) != 0 {
		t.Errorf("%v %+v", err, w)
	}
}
