package project

import (
	"io/fs"
	"strings"
	"testing"
)

// A name that starts with a dot is left out unless a module names it. The walk
// enters a dot folder only where some Realized in pattern may name something
// below it, so that what nobody names is never opened.

// recording is a file system that keeps the names of what is opened.
type recording struct {
	fs.FS
	opened *[]string
}

func (r recording) Open(name string) (fs.File, error) {
	*r.opened = append(*r.opened, name)
	return r.FS.Open(name)
}

// opens reports whether something at or below the folder was opened.
func opens(opened []string, folder string) bool {
	for _, o := range opened {
		if o == folder || strings.HasPrefix(o, folder+"/") {
			return true
		}
	}
	return false
}

// readme is a README with the given Realized in patterns, one code span each.
func readme(title string, patterns ...string) string {
	text := "# " + title + "\n\nThe " + title + ".\n\n## Realized in\n"
	for _, p := range patterns {
		text += "- `" + p + "`\n"
	}
	return text
}

// dotted is a project whose modules are built in dot folders, besides folders
// that nobody names.
func dotted() map[string]string {
	return map[string]string{
		".markitect/spec":                "docs/spec\n",
		"docs/spec/skills/README.md":     readme("Skills", ".claude/skills/triage/"),
		"docs/spec/ci/README.md":         readme("CI", ".github/workflows/*.yaml"),
		"docs/spec/editor/README.md":     readme("Editor", ".editorconfig"),
		".claude/skills/triage/SKILL.md": "triage",
		".claude/skills/triage/run.sh":   "run",
		".claude/skills/triage/.hidden":  "hidden",
		".claude/skills/other/SKILL.md":  "other",
		".claude/worktrees/w1/big.txt":   "big",
		".git/HEAD":                      "ref",
		".git/objects/ab/cdef":           "object",
		".github/workflows/ci.yaml":      "ci",
		".github/workflows/notes.txt":    "notes",
		".github/workflows/.draft.yaml":  "draft",
		".github/CODEOWNERS":             "owners",
		".editorconfig":                  "root",
		".gitignore":                     "x",
		"src/a.go":                       "package a",
		"src/.env":                       "secret",
	}
}

// diffRecording compares two projects whose files are recorded, and returns the
// rendering and the names opened on both sides.
func diffRecording(t *testing.T, older, newer map[string]string, ignore string) (string, []string) {
	t.Helper()
	ig, err := ParseIgnore([]byte(ignore))
	if err != nil {
		t.Fatal(err)
	}
	var opened []string
	o, n := load(t, older), load(t, newer)
	o.FS, n.FS = recording{o.FS, &opened}, recording{n.FS, &opened}
	d, err := DiffProjects(o, n, ig, Folder{}, Folder{})
	if err != nil {
		t.Fatal(err)
	}
	return render(d), opened
}

func TestDotFolderNamedByAModule(t *testing.T) {
	newer := with(dotted(), map[string]string{
		".claude/skills/triage/SKILL.md": "triage, changed",
		".claude/skills/triage/new.md":   "new",
		".claude/skills/triage/run.sh":   gone,
		// Nobody names these, in either state: not read, not listed, not counted.
		".claude/skills/other/SKILL.md": "other, changed",
		".claude/worktrees/w1/big.txt":  gone,
		".claude/worktrees/w2/new.txt":  "new",
		".git/HEAD":                     "ref, changed",
		".git/objects/ab/new":           "new object",
		".gitignore":                    "changed",
		".github/CODEOWNERS":            "changed",
		".claude/skills/triage/.hidden": "hidden, changed",
		"src/.env":                      "changed",
	})
	// Total of NEW that no module names: src/a.go only.
	want := "skills spec=false files=true\n  changed .claude/skills/triage/SKILL.md\n  new .claude/skills/triage/new.md\n  gone .claude/skills/triage/run.sh\nunassigned:\ntotal 1 differs true\n"
	got, opened := diffRecording(t, dotted(), newer, "")
	if got != want {
		t.Errorf("got\n%s", got)
	}
	// What no pattern could name is never opened; the folders of the skill are.
	for _, folder := range []string{".git", ".claude/worktrees", ".claude/skills/other", ".github/CODEOWNERS"} {
		if opens(opened, folder) {
			t.Errorf("%s was opened", folder)
		}
	}
	if !opens(opened, ".claude/skills/triage") {
		t.Error("the skill was not read")
	}
}

func TestDotFileNamedAsAFilePattern(t *testing.T) {
	newer := with(dotted(), map[string]string{
		".github/workflows/ci.yaml":     "ci, changed",
		".github/workflows/lint.yaml":   "new",
		".github/workflows/notes.txt":   "changed",
		".github/workflows/.draft.yaml": "changed",
		".github/CODEOWNERS":            "changed",
		".editorconfig":                 "changed",
		".gitignore":                    "changed",
	})
	want := "ci spec=false files=true\n  changed .github/workflows/ci.yaml\n  new .github/workflows/lint.yaml\n" +
		"editor spec=false files=true\n  changed .editorconfig\nunassigned:\ntotal 1 differs true\n"
	got, opened := diffRecording(t, dotted(), newer, "")
	if got != want {
		t.Errorf("got\n%s", got)
	}
	if opens(opened, ".git") || opens(opened, ".claude/worktrees") || opens(opened, ".claude/skills/other") {
		t.Errorf("a folder nobody names was opened: %v", opened)
	}
}

// A segment that is a bare wildcard never matches a dot name, so */workflows/
// does not open .github; .*/x/ does open dot folders, because its segment
// starts with a dot.
func TestWildcardSegmentsAndDotFolders(t *testing.T) {
	files := map[string]string{
		".markitect/spec":           "docs/spec\n",
		"docs/spec/flows/README.md": readme("Flows", "*/workflows/"),
		".github/workflows/ci.yaml": "ci",
		"app/workflows/run.yaml":    "run",
		".a/workflows/x.txt":        "x",
	}
	newer := with(files, map[string]string{".github/workflows/ci.yaml": "changed", "app/workflows/run.yaml": "changed", ".a/workflows/x.txt": "changed"})
	got, opened := diffRecording(t, files, newer, "")
	if want := "flows spec=false files=true\n  changed app/workflows/run.yaml\nunassigned:\ntotal 0 differs true\n"; got != want {
		t.Errorf("*/workflows/\n%s", got)
	}
	if opens(opened, ".github") || opens(opened, ".a") {
		t.Errorf("a dot folder was opened by a bare wildcard: %v", opened)
	}

	files = map[string]string{
		".markitect/spec":          "docs/spec\n",
		"docs/spec/dots/README.md": readme("Dots", ".*/x/"),
		".a/x/f.txt":               "f",
		".b/y/g.txt":               "g",
		".c/x/.hidden":             "h",
		".c/x/sub/deep.txt":        "d",
		"plain/x/h.txt":            "h",
	}
	newer = with(files, map[string]string{".a/x/f.txt": "changed", ".b/y/g.txt": "changed", ".c/x/.hidden": "changed", ".c/x/sub/deep.txt": "changed", "plain/x/h.txt": "changed"})
	got, opened = diffRecording(t, files, newer, "")
	want := "dots spec=false files=true\n  changed .a/x/f.txt\n  changed .c/x/sub/deep.txt\nunassigned: changed plain/x/h.txt;\ntotal 1 differs true\n"
	if got != want {
		t.Errorf(".*/x/\n%s", got)
	}
	if opens(opened, ".b/y") || !opens(opened, ".a/x") {
		t.Errorf("walked the wrong dot folders: %v", opened)
	}
}

// The folder .markitect/ is never read, whatever names it; the ignore file wins
// over a module; the spec folder rule is the same.
func TestDotFoldersIgnoreAndMarkitect(t *testing.T) {
	files := with(dotted(), map[string]string{"docs/spec/all/README.md": readme("All", ".*/")})
	newer := with(files, map[string]string{
		".markitect/spec":                "docs/spec\n",
		".markitect/other":               "new",
		".git/HEAD":                      "changed",
		".claude/skills/triage/SKILL.md": "changed",
		".github/workflows/ci.yaml":      "changed",
	})
	got, opened := diffRecording(t, files, newer, "")
	// .*/ names everything below the dot folders of the top, but never a dot name
	// below them, and never .markitect/.
	want := "all spec=false files=true\n  changed .claude/skills/triage/SKILL.md\n  changed .git/HEAD\n  changed .github/workflows/ci.yaml\n" +
		"ci spec=false files=true\n  changed .github/workflows/ci.yaml\n" +
		"skills spec=false files=true\n  changed .claude/skills/triage/SKILL.md\nunassigned:\ntotal 1 differs true\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	for _, o := range opened {
		if o == ".markitect" || strings.HasPrefix(o, ".markitect/") {
			t.Errorf(".markitect was opened: %v", opened)
			break
		}
	}
	// The ignore file leaves a dot folder out, though a module names it.
	newer = with(dotted(), map[string]string{".claude/skills/triage/SKILL.md": "changed", ".github/workflows/ci.yaml": "changed"})
	got, opened = diffRecording(t, dotted(), newer, ".claude/\n")
	if want := "ci spec=false files=true\n  changed .github/workflows/ci.yaml\nunassigned:\ntotal 1 differs true\n"; got != want {
		t.Errorf("ignored .claude/\n%s", got)
	}
	if opens(opened, ".claude") {
		t.Errorf(".claude was opened: %v", opened)
	}
	got, _ = diffRecording(t, dotted(), newer, "triage\n*.yaml\n")
	if want := "unassigned:\ntotal 1 differs false\n"; got != want {
		t.Errorf("ignored by name\n%s", got)
	}
}

// A module and a module below it name files below a dot folder: the lower
// module takes what it names, as for any file.
func TestDotFolderNamedByParentAndChild(t *testing.T) {
	files := map[string]string{
		".markitect/spec":                   "docs/spec\n",
		"docs/spec/skills/README.md":        readme("Skills", ".claude/skills/"),
		"docs/spec/skills/triage/README.md": readme("Triage", ".claude/skills/triage/*.md"),
		".claude/skills/triage/SKILL.md":    "triage",
		".claude/skills/triage/run.sh":      "run",
		".claude/skills/other/SKILL.md":     "other",
		".claude/skills/.cache/x":           "cache",
		".claude/settings.json":             "settings",
	}
	newer := with(files, map[string]string{
		".claude/skills/triage/SKILL.md": "changed",
		".claude/skills/triage/run.sh":   "changed",
		".claude/skills/other/SKILL.md":  "changed",
		".claude/skills/.cache/x":        "changed",
		".claude/settings.json":          "changed",
	})
	want := "skills spec=false files=true\n  changed .claude/skills/other/SKILL.md\n  changed .claude/skills/triage/run.sh\n" +
		"skills/triage spec=false files=true\n  changed .claude/skills/triage/SKILL.md\nunassigned:\ntotal 0 differs true\n"
	got, opened := diffRecording(t, files, newer, "")
	if got != want {
		t.Errorf("got\n%s", got)
	}
	if opens(opened, ".claude/skills/.cache") {
		t.Errorf("a dot folder below a named folder was opened: %v", opened)
	}
}

// A dot file below a plain folder is named by a pattern that spells it, and
// by nothing else.
func TestDotFileBelowAPlainFolder(t *testing.T) {
	files := map[string]string{
		".markitect/spec":         "docs/spec\n",
		"docs/spec/app/README.md": readme("App", "src/*", "cfg/.env"),
		"src/a.go":                "a",
		"src/.env":                "secret",
		"cfg/.env":                "cfg",
		"cfg/.other":              "other",
	}
	newer := with(files, map[string]string{"src/a.go": "changed", "src/.env": "changed", "cfg/.env": "changed", "cfg/.other": "changed"})
	want := "app spec=false files=true\n  changed cfg/.env\n  changed src/a.go\nunassigned:\ntotal 0 differs true\n"
	if got := diffOf(t, files, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}
