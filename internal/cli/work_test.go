package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	// The errors keep status and confirm from giving a list or writing.
	code, out, _ = run("status", "-C", dir)
	if code != ExitFound || !strings.Contains(out, ".markitect/plan/lending/renew-online/no-goal.md:3: section.missing") || !strings.HasSuffix(out, "no list while check finds errors or the state has errors\n") {
		t.Errorf("status: %d %s", code, out)
	}
	if code, _, errOut := run("confirm", "-C", dir, "lending/renew.md@00000000", "ok", "-e", "src/renew.go"); code != ExitFound || !strings.Contains(errOut, "nothing is confirmed while check finds errors") {
		t.Errorf("confirm: %d %s", code, errOut)
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

// move rewrites links in the spec only: a link from the work to a document
// that moved is dead afterwards, and check names it.
func TestMoveLeavesTheLinksOfTheWorkToCheck(t *testing.T) {
	dir := project(t, workFiles())
	code, out, errOut := run("move", "-C", dir, "lending/renew", "lending/renewal")
	if code != ExitOK || out != "moved lending/renew.md -> lending/renewal.md\n" {
		t.Fatal(code, out, errOut)
	}
	code, out, _ = run("check", "-C", dir)
	if code != ExitFound || !strings.Contains(out, `.markitect/findings/pending/late-fee.md:6: link.unresolved: no file at "../../../docs/spec/lending/renew.md"`) ||
		!strings.Contains(out, "2 errors in 6 documents, 7 documents of the work") {
		t.Errorf("%d\n%s", code, out)
	}
	if got := read(t, dir, ".markitect/findings/pending/late-fee.md"); got != workFiles()[".markitect/findings/pending/late-fee.md"] {
		t.Error("move changed a document of the work")
	}
}
