package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// lendingFiles is a project with two modules that say where they are built, and the
// files that build them.
var lendingFiles = map[string]string{
	"docs/spec/lending/README.md":     "# Lending\n\nHow members borrow.\n\n## Realized in\n- `src/lending/`\n\n## Contents\n- `*.md`\n",
	"docs/spec/lending/loan-limit.md": "# Loan limit\n\nFive loans.\n",
	"docs/spec/members/README.md":     "# Members\n\nWho borrows.\n\n## Realized in\n- `src/members/*.go`\n",
	"src/lending/loan.go":             "package lending\n",
	"src/lending/old.go":              "package lending\n",
	"src/members/card.go":             "package members\n",
	"tools/build.sh":                  "#!/bin/sh\n",
	"LICENSE":                         "MIT\n",
	".markitect/ignore":               "# generated, and the version control\nout/\n.git/\n.hidden/\n",
	"out/result.txt":                  "left out by the ignore file\n",
	".git/HEAD":                       "ref: refs/heads/main\n",
	"docs/spec/lending/diagram.png":   "png",
	"src/members/.hidden/secret.go":   "package secret\n",
}

// projectOf writes a project of the given files, with .markitect/spec, and
// returns its folder.
func projectOf(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	markitectFile(t, dir)
	for p, text := range files {
		write(t, dir, p, text)
	}
	return dir
}

func changes(files map[string]string, change map[string]string) map[string]string {
	out := map[string]string{}
	for p, s := range files {
		out[p] = s
	}
	for p, s := range change {
		if s == "\x00gone" {
			delete(out, p)
		} else {
			out[p] = s
		}
	}
	return out
}

// snapshotOf reads every file below dir, to show that a command wrote nothing.
func snapshotOf(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if e.IsDir() {
			out[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(p)
		out[filepath.ToSlash(rel)] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func changedLending() map[string]string {
	return changes(lendingFiles, map[string]string{
		"docs/spec/lending/loan-limit.md": "# Loan limit\n\nSix loans.\n",
		"src/lending/loan.go":             "package lending // loans\n",
		"src/lending/old.go":              "\x00gone",
		"src/lending/renew.go":            "package lending\n",
		"src/members/card.go":             "package members // card\n",
		"tools/build.sh":                  "#!/bin/sh\nset -e\n",
		"notes.txt":                       "new\n",
		"out/result.txt":                  "changed, but left out\n",
		".git/HEAD":                       "ref: refs/heads/dev\n",
		"src/members/.hidden/secret.go":   "package secret // changed\n",
		"docs/spec/lending/diagram.png":   "png2",
	})
}

const lendingText = `# Difference

## New (0)

(none)

## Gone (0)

(none)

## Changed (1)

lending/loan-limit.md
--- docs/spec/lending/loan-limit.md
+++ docs/spec/lending/loan-limit.md
@@ -1,3 +1,3 @@
 # Loan limit
 
-Five loans.
+Six loans.

## Modules (2)

lending: spec and files changed (3)
  changed src/lending/loan.go
  gone    src/lending/old.go
  new     src/lending/renew.go

members: files changed (1)
  changed src/members/card.go

## Files no module names (2)

new     notes.txt
changed tools/build.sh

Files of NEW that no module names: 3
`

func TestDiffShowsModulesAndFiles(t *testing.T) {
	older, newer := projectOf(t, lendingFiles), projectOf(t, changedLending())
	beforeOld, beforeNew := snapshotOf(t, older), snapshotOf(t, newer)
	code, out, errOut := run("diff", older, newer)
	if code != ExitFound || errOut != "" || out != lendingText {
		t.Fatalf("%d %q\n%s", code, errOut, out)
	}
	// NEW is by default this project; the text is the same at every run.
	if code, again, _ := run("diff", "-C", newer, older); code != ExitFound || again != out {
		t.Errorf("default NEW: %d\n%s", code, again)
	}
	// The command wrote nothing, in either project.
	if !reflect.DeepEqual(snapshotOf(t, older), beforeOld) || !reflect.DeepEqual(snapshotOf(t, newer), beforeNew) {
		t.Error("a command wrote into a project")
	}

	code, out, _ = run("diff", older, newer, "--json")
	if code != ExitFound {
		t.Fatal(code)
	}
	var doc struct {
		Kind    string `json:"kind"`
		Changed []struct{ Doc string }
		Modules []struct {
			Module       string
			Spec, Files  bool
			NewFiles     []string `json:"new_files"`
			GoneFiles    []string `json:"gone_files"`
			ChangedFiles []string `json:"changed_files"`
		}
		Unassigned struct {
			Changes []struct{ File, Change string }
			Total   int
		}
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Kind != "diff" || len(doc.Changed) != 1 || len(doc.Modules) != 2 {
		t.Fatalf("%+v", doc)
	}
	lending, members := doc.Modules[0], doc.Modules[1]
	if lending.Module != "lending" || !lending.Spec || !lending.Files || !reflect.DeepEqual(lending.NewFiles, []string{"src/lending/renew.go"}) ||
		!reflect.DeepEqual(lending.GoneFiles, []string{"src/lending/old.go"}) || !reflect.DeepEqual(lending.ChangedFiles, []string{"src/lending/loan.go"}) {
		t.Errorf("lending %+v", lending)
	}
	if members.Module != "members" || members.Spec || !members.Files || len(members.NewFiles)+len(members.GoneFiles) != 0 || !reflect.DeepEqual(members.ChangedFiles, []string{"src/members/card.go"}) {
		t.Errorf("members %+v", members)
	}
	if doc.Unassigned.Total != 3 || len(doc.Unassigned.Changes) != 2 || doc.Unassigned.Changes[0].File != "notes.txt" || doc.Unassigned.Changes[0].Change != "new" ||
		doc.Unassigned.Changes[1].File != "tools/build.sh" || doc.Unassigned.Changes[1].Change != "changed" {
		t.Errorf("unassigned %+v", doc.Unassigned)
	}
	// An empty list is [], not null.
	_, out, _ = run("diff", older, older, "--json")
	for _, part := range []string{`"modules": []`, `"changes": []`, `"total": 2`, `"changed": []`} {
		if !strings.Contains(out, part) {
			t.Errorf("missing %s in\n%s", part, out)
		}
	}
	if strings.Contains(out, "null") {
		t.Errorf("null in\n%s", out)
	}
}

// A file that changed makes diff exit 1, with nothing else different; the same
// projects without a change exit 0.
func TestDiffExitCodeFollowsFiles(t *testing.T) {
	older := projectOf(t, lendingFiles)
	same := projectOf(t, lendingFiles)
	if code, out, _ := run("diff", older, same); code != ExitOK || !strings.HasSuffix(out, "## Modules (0)\n\n(none)\n\n## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 2\n") {
		t.Errorf("%d\n%s", code, out)
	}
	for name, change := range map[string]map[string]string{
		"a module file":     {"src/members/card.go": "package members // x\n"},
		"a file of no one":  {"LICENSE": "BSD\n"},
		"a new file":        {"README.txt": "x"},
		"a line end":        {"LICENSE": "MIT\r\n"},
		"an emptied file":   {"LICENSE": ""},
		"a file in a nest":  {"a/b/c/d.txt": "x"},
		"a file of a name":  {"src/lending/new file (1).go": "x"},
		"a binary file":     {"tools/tool.bin": "\x00\x01"},
		"a folder of files": {"data/x.csv": "1"},
	} {
		newer := projectOf(t, changes(lendingFiles, change))
		if code, _, errOut := run("diff", older, newer); code != ExitFound || errOut != "" {
			t.Errorf("%s: %d %q", name, code, errOut)
		}
	}
	// A file that differs only inside the ignore file's reach, the spec folder or a dot name does not count.
	newer := projectOf(t, changes(lendingFiles, map[string]string{"out/other.txt": "x", ".git/ci.yml": "x", "docs/spec/lending/diagram.png": "other", "src/lending/.hidden/x": "x"}))
	if code, out, _ := run("diff", older, newer); code != ExitOK {
		t.Errorf("%d\n%s", code, out)
	}
}

// Two projects that differ only in documents: what is printed about them is what
// it was; what is new is the two headings after it, the modules of the documents
// that changed, the total, and in JSON the fields modules and unassigned.
func TestDiffOfSpecsOnlyKeepsTheOldParts(t *testing.T) {
	older, newer := beforeAndAfter(t)
	_, text, _ := run("diff", older, newer)
	const old = "# Difference\n\n## New (2)\n\nlending/fines.md\nroles/card-holder.md\n\n## Gone (2)\n\nlending/holds.md\nroles/member.md\n\n## Changed (1)\n\nlending/return.md\n"
	if !strings.HasPrefix(text, old) {
		t.Fatalf("the old part changed:\n%s", text)
	}
	rest := text[strings.Index(text, "\n## Modules"):]
	const want = "\n## Modules (2)\n\nlending: spec changed\n\nroles: spec changed\n\n## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 0\n"
	if rest != want {
		t.Errorf("rest:\n%q\nwant\n%q", rest, want)
	}
	// With nothing different at all the new part is the two empty headings and the total.
	_, text, _ = run("diff", older, older)
	if want := "## Changed (0)\n\n(none)\n\n## Modules (0)\n\n(none)\n\n## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 0\n"; !strings.HasSuffix(text, want) {
		t.Errorf("%q", text)
	}
	// The JSON keeps its fields, in their order, and adds two behind them.
	_, out, _ := run("diff", older, newer, "--json")
	var order []string
	dec := json.NewDecoder(strings.NewReader(out))
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	for dec.More() {
		key, _ := dec.Token()
		order = append(order, key.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(order, " "); got != "format_version kind new gone changed modules unassigned" {
		t.Errorf("fields %s", got)
	}
}

func TestDiffReportsABadIgnoreLine(t *testing.T) {
	older := projectOf(t, lendingFiles)
	newer := projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": "# one\nout/\na/../b\n"}))
	code, out, errOut := run("diff", older, newer)
	if code != ExitFailed || out != "" || errOut != "markitect diff: .markitect/ignore:3: bad pattern \"a/../b\": a pattern stays below the project folder, without ..\n" {
		t.Errorf("%d %q %q", code, out, errOut)
	}
	// Only the ignore file of NEW is read, and check reads none.
	if code, _, errOut := run("diff", newer, older); code != ExitOK || errOut != "" {
		t.Errorf("a bad ignore file of OLD: %d %q", code, errOut)
	}
	if code, out, _ := run("check", "-C", newer); code != ExitOK || !strings.HasPrefix(out, "0 errors") {
		t.Errorf("check: %d %q", code, out)
	}
	for text, want := range map[string]string{
		"//abs\n":      `.markitect/ignore:1: bad pattern "//abs": a pattern has no empty folder name and no .`,
		"x\n[\n":       `.markitect/ignore:2: bad pattern "[": a pattern uses the wildcards *, ? and [a-z]`,
		"a\\b\r\n":     `.markitect/ignore:1: bad pattern "a\b": a backslash is only the escape of a ! or # at the start of a pattern`,
		"!x\n!../y\n":  `.markitect/ignore:2: bad pattern "!../y": a pattern stays below the project folder, without ..`,
		"\n\n  \na//b": `.markitect/ignore:4: bad pattern "a//b": a pattern has no empty folder name and no .`,
	} {
		bad := projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": text}))
		if code, _, errOut := run("diff", older, bad); code != ExitFailed || !strings.Contains(errOut, want) {
			t.Errorf("%q: %d %q", text, code, errOut)
		}
	}
}

// The ignore file of NEW is the one that counts, for both sides.
func TestDiffIgnoreFileOfNewCountsForBothSides(t *testing.T) {
	older := projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": "tools/\n", "out/result.txt": "old"}))
	newer := projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": "", "out/result.txt": "new", "tools/build.sh": "#!/bin/sh\n# changed\n"}))
	_, out, _ := run("diff", older, newer)
	for _, part := range []string{"## Files no module names (2)\n\nchanged out/result.txt\nchanged tools/build.sh\n\nFiles of NEW that no module names: 5\n"} {
		if !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
	// Without an ignore file in NEW, one in OLD does nothing, and the other way round.
	older = projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": "*\n"}))
	newer = projectOf(t, changes(lendingFiles, map[string]string{".markitect/ignore": "", "LICENSE": "BSD\n"}))
	if _, out, _ := run("diff", older, newer); !strings.Contains(out, "changed LICENSE\n") {
		t.Errorf("OLD's file counted:\n%s", out)
	}
}

// A project without Realized in: every changed file is one that no module names.
func TestDiffWithoutRealizedIn(t *testing.T) {
	files := map[string]string{"docs/spec/a.md": "# A\n", "src/x.go": "x", "src/y.go": "y"}
	older, newer := projectOf(t, files), projectOf(t, changes(files, map[string]string{"src/x.go": "X", "src/z.go": "z", "src/y.go": "\x00gone"}))
	code, out, _ := run("diff", older, newer)
	want := "## Modules (0)\n\n(none)\n\n## Files no module names (3)\n\nchanged src/x.go\ngone    src/y.go\nnew     src/z.go\n\nFiles of NEW that no module names: 2\n"
	if code != ExitFound || !strings.HasSuffix(out, want) {
		t.Errorf("%d\n%s", code, out)
	}
}

func TestCheckReportsABadRealizedInPattern(t *testing.T) {
	dir := projectOf(t, map[string]string{
		"docs/spec/shop/README.md": "# Shop\n\nThe shop.\n\n## Realized in\n- `src/shop/`\n- `../shop`\n- `C:\\shop`\n- `src//x`\n",
		"docs/spec/shop/a.md":      "# A\n",
	})
	code, out, _ := run("check", "-C", dir)
	want := "shop/README.md:7: readme.pattern: bad pattern \"../shop\": a pattern stays below the project folder, without ..\n" +
		"shop/README.md:8: readme.pattern: bad pattern \"C:\\shop\": a pattern separates folders with /, not \\\n" +
		"shop/README.md:9: readme.pattern: bad pattern \"src//x\": a pattern has no empty folder name and no .\n" +
		"3 errors in 2 documents\n"
	if code != ExitFound || out != want {
		t.Errorf("%d\n%s", code, out)
	}
}

// A symbolic link is never followed, to a file or to a folder.
func TestDiffSkipsSymbolicLinks(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "inside.go", "package lending\n")
	older, newer := projectOf(t, lendingFiles), projectOf(t, lendingFiles)
	for _, dir := range []string{older, newer} {
		if err := os.Symlink(filepath.Join(outside, "inside.go"), filepath.Join(dir, "src", "lending", "link.go")); err != nil {
			t.Skip("no symbolic links here:", err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "src", "lending", "linked")); err != nil {
			t.Skip("no symbolic links here:", err)
		}
	}
	// The target changes behind the links: neither the file nor the folder shows.
	write(t, outside, "inside.go", "package lending // changed\n")
	write(t, outside, "new.go", "package lending\n")
	if code, out, _ := run("diff", older, newer); code != ExitOK || !strings.HasSuffix(out, "Files of NEW that no module names: 2\n") {
		t.Errorf("%d\n%s", code, out)
	}
}

// Only the section Realized in changes between the states, the files are the
// same: the module shows its spec as changed, no file.
func TestDiffRealizedInChangedOnly(t *testing.T) {
	older := projectOf(t, lendingFiles)
	newer := projectOf(t, changes(lendingFiles, map[string]string{
		"docs/spec/members/README.md": "# Members\n\nWho borrows.\n\n## Realized in\n- `src/members/*.go`\n- `tools/`\n- `LICENSE`\n",
	}))
	code, out, _ := run("diff", older, newer)
	if code != ExitFound || !strings.HasSuffix(out, "## Modules (1)\n\nmembers: spec changed\n\n## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 0\n") {
		t.Errorf("%d\n%s", code, out)
	}
	_, out, _ = run("diff", older, newer, "--json")
	if !strings.Contains(out, `"spec": true`) || !strings.Contains(out, `"files": false`) || !strings.Contains(out, `"new_files": []`) {
		t.Errorf("%s", out)
	}
}

// projectIn writes a project of the given files into the folder dir.
func projectIn(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	markitectFile(t, dir)
	for p, text := range files {
		write(t, dir, p, text)
	}
	return dir
}

// A copy kept inside the project is no file of that project: not when OLD lies
// in NEW, and not when NEW lies in OLD.
func TestDiffLeavesOutAProjectThatLiesInside(t *testing.T) {
	outer := projectOf(t, lendingFiles)
	inner := projectIn(t, filepath.Join(outer, "backup", "old"), lendingFiles)
	for name, args := range map[string][]string{"OLD inside NEW": {inner, outer}, "NEW inside OLD": {outer, inner}} {
		code, out, errOut := run(append([]string{"diff"}, args...)...)
		if code != ExitOK || errOut != "" || !strings.HasSuffix(out, "## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 2\n") {
			t.Errorf("%s: %d %q\n%s", name, code, errOut, out)
		}
	}
	// What differs is still found, and the total of NEW does not count the inner project.
	write(t, outer, "src/lending/loan.go", "package lending // changed\n")
	code, out, _ := run("diff", inner, outer)
	if code != ExitFound || !strings.Contains(out, "lending: files changed (1)\n  changed src/lending/loan.go\n") || strings.Contains(out, "backup") || !strings.HasSuffix(out, "Files of NEW that no module names: 2\n") {
		t.Errorf("%d\n%s", code, out)
	}
	// The folder is found by what is on the disk, however its path is written.
	code, out, _ = run("diff", filepath.Join(outer, "backup", "..", "backup", "old"), outer)
	if code != ExitFound || strings.Contains(out, "backup") {
		t.Errorf("%d\n%s", code, out)
	}
}

// The spec folder is left out whatever case .markitect/spec writes its name in,
// on a file system that does not regard case.
func TestDiffLeavesOutTheSpecFolderInAnyCase(t *testing.T) {
	older, newer := projectOf(t, lendingFiles), projectOf(t, changes(lendingFiles, map[string]string{
		"docs/spec/lending/loan-limit.md": "# Loan limit\n\nSix loans.\n", "docs/spec/lending/diagram.png": "png2",
	}))
	for _, dir := range []string{older, newer} {
		write(t, dir, ".markitect/spec", "Docs/Spec\n")
	}
	if _, err := os.Stat(filepath.Join(older, "Docs", "Spec")); err != nil {
		t.Skip("this file system regards case")
	}
	code, out, errOut := run("diff", older, newer)
	if code != ExitFound || errOut != "" || !strings.Contains(out, "## Changed (1)\n\nlending/loan-limit.md\n") ||
		!strings.HasSuffix(out, "## Files no module names (0)\n\n(none)\n\nFiles of NEW that no module names: 2\n") {
		t.Errorf("%d %q\n%s", code, errOut, out)
	}
}
