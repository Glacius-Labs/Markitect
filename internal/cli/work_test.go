package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// factoryTypes are the types of the work in the spec's own copy of the
// factory package.
var factoryTypes = map[string]string{
	"docs/spec/packages/factory/feature.md":  "# Feature\n\nAn idea to build.\n\n## Target picture\nWhat it looks like when it is done.\n",
	"docs/spec/packages/factory/story.md":    "# Story\n\nFunctionally complete in itself.\n\n## Goal\nWhat it reaches.\n\n## Tasks (optional)\nThe steps.\n",
	"docs/spec/packages/factory/finding.md":  "# Finding\n\nOne thing that deserves attention.\n\n## What\nWhat it is.\n",
	"docs/spec/packages/factory/report.md":   "# Report\n\nWhat a run wrote about itself.\n\n## Outcome\nHow it ended.\n",
	"docs/spec/packages/factory/briefing.md": "# Briefing\n\nWhat reaches people.\n\n## Needs a decision\nThe findings that wait.\n",
}

// typed is an item of the work of a type of the factory package, in a folder
// that lies up folders below the project folder.
func typed(up int, typ, section, body string) string {
	return "# An item\n\nType: [" + typ + "](" + strings.Repeat("../", up) + "docs/spec/packages/factory/" + strings.ToLower(typ) + ".md)\n\n## " + section + "\n" + body + "\n"
}

// workFiles is a project with a document of each type of the work, a dropped
// note and a checkpoint without a type, and what they link.
func workFiles() map[string]string {
	files := map[string]string{
		"docs/spec/lending/renew.md": "# Renew\n\nA member renews a loan.\n",
		"src/renew.go":               "package lending\n",

		".markitect/plan/lending/renew-online/feature.md":      typed(4, "Feature", "Target picture", "Renewing online, see [the rule](../../../../docs/spec/lending/renew.md)."),
		".markitect/plan/lending/renew-online/renew-a-loan.md": typed(4, "Story", "Goal", "Members renew at [the code](../../../../src/renew.go)."),
		".markitect/findings/pending/late-fee.md":              typed(3, "Finding", "What", "Fees are unclear, see [the rule](../../../docs/spec/lending/renew.md)."),
		".markitect/context/build/2026-11-20-0900-renew.md":    typed(3, "Report", "Outcome", "Merged."),
		".markitect/briefing/2026-11-21.md":                    typed(2, "Briefing", "Needs a decision", "See [the finding](../findings/pending/late-fee.md)."),
		".markitect/context/capture/dropped-note.md":           "# A dropped note\n\nNothing to see, no type.\n",
		".markitect/context/build/checkpoint.md":               "# Checkpoint\n\nContinue at [the story](../../plan/lending/renew-online/renew-a-loan.md).\n",
		".markitect/context/capture/.raw/transcript.md":        "Raw text with [a dead link](nowhere.md).\n",
		".markitect/context/capture/transcript.txt":            "Not Markdown: [x](nowhere.md)\n",
	}
	for p, text := range factoryTypes {
		files[p] = text
	}
	return files
}

func TestCheckChecksTheDocumentsOfTheWork(t *testing.T) {
	dir := project(t, workFiles())
	code, out, errOut := run("check", "-C", dir)
	if code != ExitOK || out != "0 errors in 6 documents, 7 documents of the work\n" || errOut != "" {
		t.Fatalf("got %d %q %q", code, out, errOut)
	}
	_, js, _ := run("check", "-C", dir, "--json")
	var doc checkDocument
	if err := json.Unmarshal([]byte(js), &doc); err != nil || doc.Documents != 6 || doc.Work != 7 || len(doc.Diagnostics) != 0 {
		t.Errorf("%v %s", err, js)
	}
}

func TestEachDefectOfTheWorkHasItsDiagnostic(t *testing.T) {
	dir := project(t, workFiles())
	write(t, dir, ".markitect/plan/lending/renew-online/no-goal.md", "# No goal\n\nType: [Story](../../../../docs/spec/packages/factory/story.md)\n")
	write(t, dir, ".markitect/plan/lending/renew-online/empty-goal.md", "# Empty goal\n\nType: [Story](../../../../docs/spec/packages/factory/story.md)\n\n## Goal\n\n")
	write(t, dir, ".markitect/findings/ready/dead-link.md", "# Dead link\n\nSee [the rule](../../../docs/spec/lending/gone.md) and [the code](../../../src/gone.go).\n")
	write(t, dir, ".markitect/findings/closed/no-type.md", "# No type\n\nType: [Finding](../../../docs/spec/packages/factory/ghost.md)\n")
	write(t, dir, ".markitect/briefing/no-title.md", "Text before any title.\n")
	code, out, errOut := run("check", "-C", dir)
	want := strings.Join([]string{
		`.markitect/briefing/no-title.md:1: title: missing level-1 title`,
		`.markitect/findings/closed/no-type.md:3: type.unresolved: no document at "../../../docs/spec/packages/factory/ghost.md"`,
		`.markitect/findings/ready/dead-link.md:3: link.unresolved: no file at "../../../docs/spec/lending/gone.md"`,
		`.markitect/findings/ready/dead-link.md:3: link.unresolved: no file at "../../../src/gone.go"`,
		`.markitect/plan/lending/renew-online/empty-goal.md:5: section.empty: empty section "Goal"`,
		`.markitect/plan/lending/renew-online/no-goal.md:3: section.missing: missing section "Goal" required by packages/factory/story.md`,
		"6 errors in 6 documents, 12 documents of the work", ""}, "\n")
	if code != ExitFound || out != want || errOut != "" {
		t.Errorf("got %d\n%s%s\nwant\n%s", code, out, errOut, want)
	}
	// What check finds, in the work or in the spec, blocks neither status nor
	// confirm: both are about the spec and its artifacts.
	code, out, _ = run("status", "-C", dir)
	if code != ExitFound || strings.Contains(out, ".markitect") || strings.Contains(out, "no list while") || !strings.Contains(out, "changed: lending/renew.md@") {
		t.Errorf("status: %d %s", code, out)
	}
	if code, out, errOut := run("confirm", "-C", dir, "lending/renew.md@"+fingerprint(t, dir, "lending/renew.md"), "ok", "-e", "src/renew.go"); code != ExitOK {
		t.Errorf("confirm: %d %s %s", code, out, errOut)
	}
}

// The work is no part of the spec: it never appears in status, a link from it
// makes nothing suspect, and show and topic count no link from it.
func TestTheWorkIsNotPartOfTheSpec(t *testing.T) {
	dir := project(t, workFiles())
	l := statusOf(t, dir)
	for _, line := range l.summary() {
		if strings.Contains(line, ".markitect") || strings.Contains(line, "plan/") || strings.Contains(line, "findings/") {
			t.Errorf("status lists %s", line)
		}
	}
	// Every document of the spec is confirmed; then the work changes.
	for _, line := range l.summary() {
		if doc, ok := strings.CutPrefix(line, "changed "); ok {
			doc, _, _ = strings.Cut(doc, ":")
			verdict(t, dir, doc, "ok", "-e", "src/renew.go")
		}
	}
	for _, line := range statusOf(t, dir).summary() {
		if !strings.HasPrefix(line, "unexplained ") {
			t.Fatalf("status lists %s", line)
		}
	}
	write(t, dir, ".markitect/findings/pending/late-fee.md", typed(3, "Finding", "What", "Changed, see [the rule](../../../docs/spec/lending/renew.md)."))
	write(t, dir, ".markitect/context/new-note.md", "# New\n\nSee [the rule](../../docs/spec/lending/renew.md).\n")
	for _, line := range statusOf(t, dir).summary() {
		if !strings.HasPrefix(line, "unexplained ") {
			t.Errorf("a change of the work shows in status: %s", line)
		}
	}
	_, out, _ := run("show", "-C", dir, "lending/renew")
	if !strings.Contains(out, "Linked from: (none)\n") {
		t.Errorf("show counts a link from the work:\n%s", out)
	}
	_, out, _ = run("topic", "-C", dir, "lending/renew")
	if strings.Contains(out, ".markitect") {
		t.Errorf("topic counts the work:\n%s", out)
	}
	// confirm binds documents of the spec and refuses one of the work.
	code, _, errOut := run("confirm", "-C", dir, ".markitect/findings/pending/late-fee.md@00000000", "ok", "-e", "src/renew.go")
	if code != ExitFound || !strings.Contains(errOut, "is a document of the work") {
		t.Errorf("%d %s", code, errOut)
	}
}

func TestMissingFoldersOfTheWorkAreNoError(t *testing.T) {
	dir := project(t, map[string]string{"docs/spec/a.md": "# A\n"})
	for _, f := range []string{"context", "findings", "plan", "briefing"} {
		if err := os.RemoveAll(filepath.Join(dir, ".markitect", f)); err != nil {
			t.Fatal(err)
		}
	}
	if code, out, errOut := run("check", "-C", dir); code != ExitOK || out != "0 errors in 1 document\n" {
		t.Error(code, out, errOut)
	}
}

// compare names the documents of the work that are new, gone or changed, by
// path.
func TestCompareNamesTheDocumentsOfTheWork(t *testing.T) {
	older := project(t, workFiles())
	newer := copyTree(t, older, false)
	write(t, newer, ".markitect/context/build/checkpoint.md", "# Checkpoint\n\nDone.\n")
	write(t, newer, ".markitect/findings/pending/new-one.md", "# New one\n\nText.\n")
	if err := os.Remove(filepath.Join(newer, ".markitect", "context", "capture", "dropped-note.md")); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("compare", older, newer, "--json")
	var doc compareDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitFound {
		t.Fatal(code, out)
	}
	var got []string
	for _, w := range doc.Work {
		got = append(got, w.Change+" "+w.Path)
	}
	same(t, "work", got, []string{"changed .markitect/context/build/checkpoint.md", "gone .markitect/context/capture/dropped-note.md", "new .markitect/findings/pending/new-one.md"})
	_, text, _ := run("compare", older, newer)
	if !strings.Contains(text, "## Work (3)\n\nchanged: .markitect/context/build/checkpoint.md\ngone: .markitect/context/capture/dropped-note.md\nnew: .markitect/findings/pending/new-one.md\n") {
		t.Error(text)
	}
	// Nothing of the spec differs, so nothing else is listed.
	if len(doc.New)+len(doc.Gone)+len(doc.Changed)+len(doc.Suspect)+len(doc.Verdicts) != 0 {
		t.Errorf("%+v", doc)
	}
	if code, out, _ := run("compare", older, copyTree(t, older, true)); code != ExitOK {
		t.Error(code, out)
	}
}

// move rewrites the links of the work to the document as it rewrites those of
// the spec, type lines included, so that check stays clean.
func TestMoveRewritesTheLinksOfTheWork(t *testing.T) {
	dir := project(t, workFiles())
	code, out, errOut := run("move", "-C", dir, "lending/renew", "lending/renewal")
	want := "moved lending/renew.md -> lending/renewal.md\nchanged .markitect/findings/pending/late-fee.md\nchanged .markitect/plan/lending/renew-online/feature.md\n"
	if code != ExitOK || out != want {
		t.Fatalf("%d %s %s\nwant\n%s", code, out, errOut, want)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK || out != "0 errors in 6 documents, 7 documents of the work\n" {
		t.Errorf("check after the move: %d %s", code, out)
	}
	files := workFiles()
	if got, want := read(t, dir, ".markitect/findings/pending/late-fee.md"), strings.Replace(files[".markitect/findings/pending/late-fee.md"], "lending/renew.md", "lending/renewal.md", 1); got != want {
		t.Errorf("late-fee.md is\n%q\nwant\n%q", got, want)
	}
	if got := read(t, dir, ".markitect/context/build/checkpoint.md"); got != files[".markitect/context/build/checkpoint.md"] {
		t.Error("move changed a document of the work that links nothing that moved")
	}
	// The type line of a document of the work names a type that moves too.
	code, out, _ = run("move", "-C", dir, "packages/factory/story", "packages/factory/tale", "--json")
	var doc moveResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK {
		t.Fatal(code, out)
	}
	same(t, "work", doc.Work, []string{".markitect/plan/lending/renew-online/renew-a-loan.md"})
	if len(doc.Left) != 0 || len(doc.Changed) != 0 {
		t.Errorf("%+v", doc)
	}
	if got := read(t, dir, ".markitect/plan/lending/renew-online/renew-a-loan.md"); !strings.Contains(got, "Type: [Story](../../../../docs/spec/packages/factory/tale.md)\n") {
		t.Error(got)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Errorf("check after the second move: %d %s", code, out)
	}
	// A move brings no change to the state of the spec because of the work.
	for _, line := range statusOf(t, dir).summary() {
		if strings.Contains(line, ".markitect") {
			t.Errorf("status lists %s", line)
		}
	}
}

// A document of the work that cannot be read does not stop a move: it stays
// as it is and the move names it.
func TestMoveLeavesAnUnreadableDocumentOfTheWorkAsItIs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := project(t, workFiles())
	locked := filepath.Join(dir, ".markitect", "findings", "pending", "late-fee.md")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })
	if _, err := os.ReadFile(locked); err == nil {
		t.Skip("this user reads a file without permission")
	}
	code, out, errOut := run("move", "-C", dir, "lending/renew", "lending/renewal")
	if code != ExitOK || !strings.Contains(out, "changed .markitect/plan/lending/renew-online/feature.md\n") ||
		!strings.Contains(out, "left as it is .markitect/findings/pending/late-fee.md: could not be read, so it was not searched for links to this document\n") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
}

// What the project's .gitignore files exclude is not read, as the walk does not
// read it: check and compare skip it. .markitect/ignore does not apply, and a
// folder that starts with a dot stays skipped.
func TestTheWorkIsReadLikeTheRestOfTheProject(t *testing.T) {
	files := workFiles()
	files[".gitignore"] = "local/\n*.draft.md\n"
	files[".markitect/context/local/notes.md"] = "no title, [dead](gone.md)\n"
	files[".markitect/plan/lending/idea.draft.md"] = "no title either\n"
	older := project(t, files)
	if code, out, _ := run("check", "-C", older); code != ExitOK || out != "0 errors in 6 documents, 7 documents of the work\n" {
		t.Errorf("check: %d %s", code, out)
	}
	// .markitect/ignore names files that realize nothing; it does not hide a
	// document of the work from check.
	write(t, older, ".markitect/context/capture/no-title.md", "text without a title\n")
	write(t, older, ".markitect/ignore", ".markitect/context/capture/no-title.md\n")
	if code, out, _ := run("check", "-C", older); code != ExitFound || !strings.Contains(out, ".markitect/context/capture/no-title.md:1: title: missing level-1 title") {
		t.Errorf("check with .markitect/ignore: %d %s", code, out)
	}
	if err := os.Remove(filepath.Join(older, ".markitect", "context", "capture", "no-title.md")); err != nil {
		t.Fatal(err)
	}
	newer := copyTree(t, older, false)
	write(t, newer, ".markitect/context/local/more.md", "ignored too\n")
	write(t, newer, ".markitect/plan/lending/other.draft.md", "ignored too\n")
	write(t, newer, ".markitect/context/capture/.raw/new.md", "a hidden folder\n")
	if code, out, _ := run("compare", older, newer); code != ExitOK || !strings.Contains(out, "## Work (0)\n") {
		t.Errorf("compare: %d %s", code, out)
	}
	// Excluding the folder of the work excludes what lies in it.
	write(t, newer, ".gitignore", "local/\n*.draft.md\n.markitect/plan/\n")
	if code, out, _ := run("check", "-C", newer); code != ExitOK || !strings.Contains(out, "5 documents of the work") {
		t.Errorf("check with an ignored folder: %d %s", code, out)
	}
}
