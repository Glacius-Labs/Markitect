package project

import (
	"strings"
	"testing"
)

func mustIgnore(t *testing.T, text string) *Ignore {
	t.Helper()
	ig, err := ParseIgnore([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return ig
}

type leaf struct {
	rel    string
	folder bool
	left   bool
}

func checkLeaves(t *testing.T, name, ignore string, cases []leaf) {
	t.Helper()
	ig := mustIgnore(t, ignore)
	for _, c := range cases {
		if got := ig.leaves(c.rel, c.folder); got != c.left {
			kind := "file"
			if c.folder {
				kind = "folder"
			}
			t.Errorf("%s: %s %q is left out %v, want %v", name, kind, c.rel, got, c.left)
		}
	}
}

func TestParseIgnoreErrors(t *testing.T) {
	for text, want := range map[string]string{
		"a\n\\x\n":         `.markitect/ignore:2: bad pattern "\x": a backslash is only the escape of a ! or # at the start of a pattern`,
		"# c\n\na/../b":    `.markitect/ignore:3: bad pattern "a/../b": a pattern stays below the project folder, without ..`,
		"x[\n":             `.markitect/ignore:1: bad pattern "x[": a pattern uses the wildcards *, ? and [a-z]`,
		"a\\b\n":           `.markitect/ignore:1: bad pattern "a\b": a backslash is only the escape of a ! or # at the start of a pattern`,
		"a//b\r\n":         `.markitect/ignore:1: bad pattern "a//b": a pattern has no empty folder name and no .`,
		"//a\n":            `.markitect/ignore:1: bad pattern "//a": a pattern has no empty folder name and no .`,
		"/\n":              `.markitect/ignore:1: bad pattern "/": a pattern is not empty`,
		"!\n":              `.markitect/ignore:1: bad pattern "!": a pattern is not empty`,
		"C:/x\n":           `.markitect/ignore:1: bad pattern "C:/x": a pattern is relative to the project folder and has no drive`,
		"a\n  \t\n./b\n":   `.markitect/ignore:3: bad pattern "./b": a pattern has no empty folder name and no .`,
		"!a/../b\n":        `.markitect/ignore:1: bad pattern "!a/../b": a pattern stays below the project folder, without ..`,
		"**/[\n":           `.markitect/ignore:1: bad pattern "**/[": a pattern uses the wildcards *, ? and [a-z]`,
		"!\\!a\n":          `.markitect/ignore:1: bad pattern "!\!a": a backslash is only the escape of a ! or # at the start of a pattern`,
		"\\ a\n":           `.markitect/ignore:1: bad pattern "\ a": a backslash is only the escape of a ! or # at the start of a pattern`,
		"ok\n/ok/\n!ok/..": `.markitect/ignore:3: bad pattern "!ok/..": a pattern stays below the project folder, without ..`,
	} {
		if _, err := ParseIgnore([]byte(text)); err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %s", text, err, want)
		}
	}
}

func TestIgnoreNames(t *testing.T) {
	// A name without a / matches a file or folder of that name at any depth.
	checkLeaves(t, "name", "node_modules\n.DS_Store\n", []leaf{
		{"node_modules", true, true}, {"a/b/node_modules", true, true}, {"a/node_modules", false, true},
		{"node_modules_x", true, false}, {"x/node_modules/y", false, false},
		{".DS_Store", false, true}, {"a/.DS_Store", false, true}, {"DS_Store", false, false},
	})
	// The wildcards stay within a name, and match a leading dot.
	checkLeaves(t, "wildcards", "*.log\n?x\n[a-c]z\n", []leaf{
		{"x.log", false, true}, {"deep/er/x.log", false, true}, {".hidden.log", false, true}, {"x.log.txt", false, false},
		{"ax", false, true}, {".x", false, true}, {"abx", false, false},
		{"az", false, true}, {"dz", false, false}, {"a/b/cz", true, true},
	})
	// A name with a space is a name; white space at the end of a line is dropped.
	checkLeaves(t, "spaces", "my file.txt \t\r\n", []leaf{{"my file.txt", false, true}, {"d/my file.txt", false, true}, {"my file.txt ", false, false}})
}

func TestIgnoreAnchored(t *testing.T) {
	// A / at the start only anchors; a / in the middle makes a path from the project folder.
	checkLeaves(t, "leading slash", "/build\n", []leaf{{"build", true, true}, {"build", false, true}, {"src/build", true, false}, {"build/x", false, false}})
	checkLeaves(t, "middle slash", "docs/generated\n", []leaf{
		{"docs/generated", true, true}, {"docs/generated", false, true}, {"src/docs/generated", true, false}, {"docs/generated/x", false, false}, {"generated", true, false},
	})
	checkLeaves(t, "anchored and unanchored", "/a.txt\nb.txt\n", []leaf{
		{"a.txt", false, true}, {"d/a.txt", false, false}, {"b.txt", false, true}, {"d/b.txt", false, true},
	})
	// A * is no / : it stays in one folder.
	checkLeaves(t, "star in a path", "src/*/gen\n", []leaf{{"src/a/gen", true, true}, {"src/gen", true, false}, {"src/a/b/gen", true, false}})
}

func TestIgnoreFoldersOnly(t *testing.T) {
	checkLeaves(t, "folders only", "build/\n", []leaf{
		{"build", true, true}, {"src/build", true, true}, {"build", false, false}, {"src/build", false, false},
	})
	checkLeaves(t, "anchored folders only", "/out/\ngen/*/\n", []leaf{
		{"out", true, true}, {"x/out", true, false}, {"out", false, false},
		{"gen/a", true, true}, {"gen/a", false, false}, {"gen/a/b", true, false}, {"x/gen/a", true, false},
	})
}

func TestIgnoreDoubleStar(t *testing.T) {
	checkLeaves(t, "leading", "**/gen\n", []leaf{{"gen", true, true}, {"a/gen", false, true}, {"a/b/gen", true, true}, {"a/genx", true, false}})
	checkLeaves(t, "middle", "a/**/b\n", []leaf{{"a/b", true, true}, {"a/x/b", false, true}, {"a/x/y/b", true, true}, {"b", true, false}, {"x/a/b", true, false}})
	checkLeaves(t, "trailing", "docs/**\n", []leaf{{"docs/a", false, true}, {"docs/a/b", true, true}, {"docs", true, false}, {"x/docs/a", false, false}})
	checkLeaves(t, "all three, folders only", "**/cache/**/\n", []leaf{{"cache/a", true, true}, {"x/cache/a/b", true, true}, {"x/cache/a", false, false}, {"cache", true, false}})
	checkLeaves(t, "alone", "**\n", []leaf{{"a", false, true}, {"a/b", true, true}})
}

func TestIgnoreNegationAndLastMatch(t *testing.T) {
	checkLeaves(t, "take back", "*.log\n!keep.log\n", []leaf{{"a.log", false, true}, {"keep.log", false, false}, {"d/keep.log", false, false}, {"d/x.log", false, true}})
	// The last line that matches decides.
	checkLeaves(t, "three lines", "*.txt\n!a.txt\na.txt\n", []leaf{{"a.txt", false, true}, {"b.txt", false, true}})
	checkLeaves(t, "three lines, back again", "*.txt\na.txt\n!a.txt\n", []leaf{{"a.txt", false, false}, {"b.txt", false, true}})
	checkLeaves(t, "a negation before anything", "!a.txt\n*.txt\n", []leaf{{"a.txt", false, true}})
	checkLeaves(t, "negation of folders only", "x/*\n!x/keep/\n", []leaf{{"x/keep", true, false}, {"x/keep", false, true}, {"x/other", true, true}})
	// A negation takes in only what an earlier line left out; another pattern has no effect.
	checkLeaves(t, "alone", "!a.txt\n", []leaf{{"a.txt", false, false}})
	// The escapes: \! and \# are a literal ! and #, a comment and a take-back are not.
	checkLeaves(t, "escapes", "\\!name\n\\#name\n# a comment\n!name\n", []leaf{
		{"!name", false, true}, {"#name", false, true}, {"name", false, false}, {"# a comment", false, false},
	})
	checkLeaves(t, "a literal ! inside", "a!b\n", []leaf{{"a!b", false, true}})
	// A byte order mark and a carriage return are dropped.
	checkLeaves(t, "bom and crlf", "\xef\xbb\xbf*.tmp\r\n!keep.tmp\r\n", []leaf{{"x.tmp", false, true}, {"keep.tmp", false, false}})
}

func TestIgnoreNothing(t *testing.T) {
	var none *Ignore
	if none.leaves("a", false) || mustIgnore(t, "# only\n\n  \n").leaves("a", true) {
		t.Error("an empty ignore file leaves something out")
	}
}

// walked lists the files of a project: the fixed rules, then the ignore file.
func walked(t *testing.T, files map[string]string, ignore string) string {
	t.Helper()
	got, err := listFiles(tree(files), mustIgnore(t, ignore), Folder{Path: "docs/model"})
	if err != nil {
		t.Fatal(err)
	}
	return names(got)
}

func TestListFilesWithoutAnIgnoreFile(t *testing.T) {
	files := map[string]string{
		"main.go":                  "",
		"docs/model/a.md":          "# A\n",
		"docs/model/img/a.png":     "png",
		"docs/notes.txt":           "",
		"docs/model-old/b.md":      "",
		".git/config":              "",
		".github/workflows/ci.yml": "",
		".hidden":                  "",
		"src/.env":                 "",
		"src/code/x.go":            "",
		".markitect/model":         "docs/model\n",
		".markitect/ignore":        "x\n",
		".Markitect/other":         "x",
		"sub/.markitect/model":     "x",
		"a b/c d.txt":              "",
	}
	// Left out are the model folder and the folder .markitect/ at the top, in any
	// case, and nothing else: a name that starts with a dot is a name like any other.
	want := ".git/config .github/workflows/ci.yml .hidden a b/c d.txt docs/model-old/b.md docs/notes.txt main.go src/.env src/code/x.go sub/.markitect/model"
	if got := walked(t, files, ""); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// The example of the ignore file that leaves out the folders a version control
// and a tool keep, and takes in the skills.
func TestListFilesWithAnIgnoreFile(t *testing.T) {
	files := map[string]string{
		"main.go":                        "",
		".git/HEAD":                      "",
		".git/objects/ab/cd":             "",
		".claude/settings.json":          "",
		".claude/skills/triage/SKILL.md": "",
		".claude/skills/triage/.hidden":  "",
		".claude/skills/.cache/x":        "",
		".claude/worktrees/w/big.txt":    "",
		".github/workflows/ci.yml":       "",
		"build/out.o":                    "",
		"src/build":                      "a file named like the folder",
		"src/x.log":                      "",
		"src/keep.log":                   "",
		"node_modules/lib/index.js":      "",
		"src/node_modules/lib/index.js":  "",
	}
	const ignore = "# the version control and the tools\n.git/\n.claude/*\n!.claude/skills/\n/build/\nnode_modules/\n*.log\n!keep.log\n"
	want := ".claude/skills/.cache/x .claude/skills/triage/.hidden .claude/skills/triage/SKILL.md .github/workflows/ci.yml main.go src/build src/keep.log"
	if got := walked(t, files, ignore); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	// A folder that is left out is not entered: nothing inside can be taken in again.
	const cannot = ".claude/\n!.claude/skills/\n!.claude/skills/triage/SKILL.md\n"
	if got := walked(t, files, cannot); got == "" || strings.Contains(got, ".claude") {
		t.Errorf("a negation under a folder that is left out: %s", got)
	}
	// Last match wins also for a folder: left out, taken in, left out again.
	if got := walked(t, files, ".git/\n!.git/\n.git/\n"); strings.Contains(got, ".git/") {
		t.Errorf("three lines for a folder: %s", got)
	}
	if got := walked(t, files, ".git/\n!.git/\n"); !strings.Contains(got, ".git/HEAD") {
		t.Errorf("a folder taken in again: %s", got)
	}
}
