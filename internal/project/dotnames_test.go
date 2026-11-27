package project

import (
	"io/fs"
	"strings"
	"testing"
)

// A name that starts with a dot is a name like any other, for the walk and for
// the patterns of Realized in. What a project does not want to look at, it says
// in its ignore file, as it does for a version control.

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
		".markitect/model":               "docs/model\n",
		"docs/model/skills/README.md":    readme("Skills", ".claude/skills/triage/"),
		"docs/model/ci/README.md":        readme("CI", ".github/workflows/*.yaml"),
		"docs/model/editor/README.md":    readme("Editor", ".editorconfig"),
		".claude/skills/triage/SKILL.md": "triage",
		".claude/skills/triage/run.sh":   "run",
		".claude/skills/triage/.hidden":  "hidden",
		".claude/skills/other/SKILL.md":  "other",
		".claude/worktrees/w1/big.txt":   "big",
		".git/HEAD":                      "ref",
		".git/objects/ab/cdef":           "object",
		".github/workflows/ci.yaml":      "ci",
		".github/workflows/notes.txt":    "notes",
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

// changedDotted changes the files of a module, and files that no module names.
func changedDotted() map[string]string {
	return with(dotted(), map[string]string{
		".claude/skills/triage/SKILL.md": "triage, changed",
		".claude/skills/triage/.hidden":  "hidden, changed",
		".claude/skills/triage/new.md":   "new",
		".claude/skills/triage/run.sh":   gone,
		".claude/skills/other/SKILL.md":  "other, changed",
		".claude/worktrees/w1/big.txt":   gone,
		".claude/worktrees/w2/new.txt":   "new",
		".git/HEAD":                      "ref, changed",
		".git/objects/ab/new":            "new object",
		".gitignore":                     "changed",
		".github/CODEOWNERS":             "changed",
		"src/.env":                       "changed",
	})
}

// A module names a folder below a dot name; the ignore file leaves out the
// version control and the contents of .claude, and takes in the skills.
func TestModuleInADotFolderWithAnIgnoreFile(t *testing.T) {
	const ignore = ".git/\n.claude/*\n!.claude/skills/\n"
	got, opened := diffRecording(t, dotted(), changedDotted(), ignore)
	// What lies below the folder of the pattern is the module's, a dot name too. The
	// other skill is taken in by the ignore file, and no module names it.
	want := "skills model=false files=true\n" +
		"  changed .claude/skills/triage/.hidden\n  changed .claude/skills/triage/SKILL.md\n  new .claude/skills/triage/new.md\n  gone .claude/skills/triage/run.sh\n" +
		"unassigned: changed .claude/skills/other/SKILL.md; changed .github/CODEOWNERS; changed .gitignore; changed src/.env;\ntotal 6 differs true\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	// A folder that is left out is never entered.
	for _, folder := range []string{".git", ".claude/worktrees", ".claude/settings.json"} {
		if opens(opened, folder) {
			t.Errorf("%s was opened", folder)
		}
	}
	if !opens(opened, ".claude/skills/triage") {
		t.Error("the skill was not read")
	}
}

// The skills are a module's, and the rest of .claude is left out: without the
// lines that take them in, the module sees nothing.
func TestModuleInADotFolderLeftOutByTheIgnoreFile(t *testing.T) {
	got, opened := diffRecording(t, dotted(), changedDotted(), ".git/\n.claude/\n")
	want := "unassigned: changed .github/CODEOWNERS; changed .gitignore; changed src/.env;\ntotal 5 differs true\n"
	if got != want {
		t.Errorf("got\n%s", got)
	}
	if opens(opened, ".claude") || opens(opened, ".git") {
		t.Errorf("a folder that is left out was opened: %v", opened)
	}
	// A line that takes back what is below a folder that is left out has no effect.
	got, _ = diffRecording(t, dotted(), changedDotted(), ".git/\n.claude/\n!.claude/skills/\n!.claude/skills/triage/SKILL.md\n")
	if got != want {
		t.Errorf("a negation below a folder that is left out\n%s", got)
	}
}

// Without an ignore file nothing is left out but what the tool fixes: every file
// counts, the dot folders too, and the files no module names are listed.
func TestNoIgnoreFile(t *testing.T) {
	newer := changedDotted()
	newer[".markitect/ignore"] = "never read\n"
	newer[".markitect/extra"] = "never read\n"
	got, opened := diffRecording(t, dotted(), newer, "")
	want := "skills model=false files=true\n" +
		"  changed .claude/skills/triage/.hidden\n  changed .claude/skills/triage/SKILL.md\n  new .claude/skills/triage/new.md\n  gone .claude/skills/triage/run.sh\n" +
		"unassigned: changed .claude/skills/other/SKILL.md; gone .claude/worktrees/w1/big.txt; new .claude/worktrees/w2/new.txt; changed .git/HEAD; new .git/objects/ab/new; changed .github/CODEOWNERS; changed .gitignore; changed src/.env;\n" +
		"total 10 differs true\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if opens(opened, ".markitect") {
		t.Errorf(".markitect was opened: %v", opened)
	}
}

// The folder .markitect/ is never read, whatever names it, and the ignore file
// cannot take it in.
func TestMarkitectFolderIsNeverRead(t *testing.T) {
	files := with(dotted(), map[string]string{"docs/model/all/README.md": readme("All", ".*/", ".markitect/", "**")})
	newer := with(files, map[string]string{".markitect/model": "docs/model\n", ".markitect/other": "new", ".git/HEAD": "changed"})
	got, opened := diffRecording(t, files, newer, "!.markitect/\n")
	if !strings.Contains(got, "  changed .git/HEAD\n") || strings.Contains(got, ".markitect") {
		t.Errorf("got\n%s", got)
	}
	if opens(opened, ".markitect") {
		t.Errorf(".markitect was opened: %v", opened)
	}
}

// Only the Realized in of two READMEs changes between the states and the files
// are the same, those below a dot folder too: no file is new, gone or changed.
func TestOnlyRealizedInChangesAndTheFilesAreTheSame(t *testing.T) {
	newer := with(dotted(), map[string]string{
		"docs/model/skills/README.md": readme("Skills", ".claude/skills/"),
		"docs/model/ci/README.md":     readme("CI", ".github/"),
		"docs/model/editor/README.md": readme("Editor"),
	})
	for _, ignore := range []string{"", ".git/\n.claude/worktrees/\n"} {
		got, _ := diffRecording(t, dotted(), newer, ignore)
		if !strings.HasPrefix(got, "ci model=true files=false\neditor model=true files=false\nskills model=true files=false\nunassigned:\ntotal ") || !strings.HasSuffix(got, " differs true\n") {
			t.Errorf("ignore %q:\n%s", ignore, got)
		}
	}
}

// A module and a module below it name files below a dot folder: the lower
// module takes what it names, as for any file; a folder pattern covers all below.
func TestDotFolderNamedByParentAndChild(t *testing.T) {
	files := map[string]string{
		".markitect/model":                   "docs/model\n",
		"docs/model/skills/README.md":        readme("Skills", ".claude/skills/"),
		"docs/model/skills/triage/README.md": readme("Triage", ".claude/skills/triage/*.md"),
		".claude/skills/triage/SKILL.md":     "triage",
		".claude/skills/triage/run.sh":       "run",
		".claude/skills/other/SKILL.md":      "other",
		".claude/skills/.cache/x":            "cache",
		".claude/settings.json":              "settings",
	}
	newer := with(files, map[string]string{
		".claude/skills/triage/SKILL.md": "changed",
		".claude/skills/triage/run.sh":   "changed",
		".claude/skills/other/SKILL.md":  "changed",
		".claude/skills/.cache/x":        "changed",
		".claude/settings.json":          "changed",
	})
	want := "skills model=false files=true\n  changed .claude/skills/.cache/x\n  changed .claude/skills/other/SKILL.md\n  changed .claude/skills/triage/run.sh\n" +
		"skills/triage model=false files=true\n  changed .claude/skills/triage/SKILL.md\n" +
		"unassigned: changed .claude/settings.json;\ntotal 1 differs true\n"
	if got := diffOf(t, files, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}

// *, ? and a class match a leading dot, a ** goes through dot folders, and a dot
// file at the top is named by a pattern that spells it.
func TestWildcardsAndDotNames(t *testing.T) {
	files := map[string]string{
		".markitect/model":           "docs/model\n",
		"docs/model/flows/README.md": readme("Flows", "*/workflows/", "src/*", "**/gen/", ".editor*"),
		".github/workflows/ci.yaml":  "ci",
		"app/workflows/run.yaml":     "run",
		"src/.env":                   "secret",
		"src/a":                      "a",
		".a/b/gen/x":                 "x",
		".editorconfig":              "root",
		".gitignore":                 "x",
	}
	newer := with(files, map[string]string{
		".github/workflows/ci.yaml": "changed", "app/workflows/run.yaml": "changed", "src/.env": "changed", "src/a": "changed",
		".a/b/gen/x": "changed", ".editorconfig": "changed", ".gitignore": "changed",
	})
	want := "flows model=false files=true\n  changed .a/b/gen/x\n  changed .editorconfig\n  changed .github/workflows/ci.yaml\n  changed app/workflows/run.yaml\n  changed src/.env\n  changed src/a\n" +
		"unassigned: changed .gitignore;\ntotal 1 differs true\n"
	if got := diffOf(t, files, newer, ""); got != want {
		t.Errorf("got\n%s", got)
	}
}
