package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readOnly makes a file of the project read-only. A user who may write anyway,
// such as root, cannot run the test.
func readOnly(t *testing.T, dir, rel string) {
	t.Helper()
	file := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(file, 0o644) })
	if f, err := os.OpenFile(file, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("this user writes a read-only file")
	}
}

// A document of the work that cannot be written does not stop a move: it stays
// as it is, and the move names it with the reason. Everything else is done.
func TestMoveLeavesAnUnwritableDocumentOfTheWorkAsItIs(t *testing.T) {
	dir := project(t, workFiles())
	const locked = ".markitect/findings/pending/late-fee.md"
	readOnly(t, dir, locked)
	code, out, errOut := run("move", "-C", dir, "lending/renew", "lending/renewal", "--json")
	var doc moveResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	same(t, "work", doc.Work, []string{".markitect/plan/lending/renew-online/feature.md"})
	if len(doc.Left) != 1 || doc.Left[0].Path != locked || !strings.HasPrefix(doc.Left[0].Reason, "cannot be written: ") {
		t.Errorf("left %+v", doc.Left)
	}
	if got := read(t, dir, locked); got != workFiles()[locked] {
		t.Error("the unwritable document changed")
	}
	if !strings.Contains(read(t, dir, ".markitect/plan/lending/renew-online/feature.md"), "lending/renewal.md") {
		t.Error("the writable document was not rewritten")
	}
	// The document left alone now has a dead link, which check names.
	if code, out, _ := run("check", "-C", dir); code != ExitFound || !strings.Contains(out, locked+":6: link.unresolved") || strings.Contains(out, "feature.md") {
		t.Errorf("check: %d %s", code, out)
	}
}

// A write in the work that fails midway undoes the whole move: the spec, the
// work, the state and the history are as they were.
func TestFailedMoveWithTheWorkLeavesEverythingAsItWas(t *testing.T) {
	steps := 0
	fault = func(int) error { steps++; return nil }
	t.Cleanup(func() { fault = nil })
	if code, _, errOut := run("move", "-C", project(t, workFiles()), "lending/renew", "elsewhere/deep/renewal"); code != ExitOK {
		t.Fatal(errOut)
	}
	if steps < 10 {
		t.Fatalf("only %d steps", steps)
	}
	dir := project(t, workFiles())
	before, folders := contents(t, dir), tree(t, dir)
	for k := 1; k <= steps; k++ {
		fault = func(step int) error {
			if step == k {
				return errors.New("injected")
			}
			return nil
		}
		if code, out, errOut := run("move", "-C", dir, "lending/renew", "elsewhere/deep/renewal"); code != ExitFailed || out != "" || !strings.Contains(errOut, "injected") {
			t.Errorf("step %d: %d %q %q", k, code, out, errOut)
		}
		if after := contents(t, dir); !equalMaps(after, before) {
			t.Errorf("step %d changed the project", k)
		}
		if got := tree(t, dir); strings.Join(got, " ") != strings.Join(folders, " ") {
			t.Errorf("step %d left %v, want %v", k, got, folders)
		}
	}
}

// formsOfALink are the forms in which a document of the work names the
// document the-rule, each of which check verifies and move must rewrite.
const formsOfALink = "# Forms\n\nType: [Story](../../docs/spec/packages/factory/story.md)\n\n## Goal\n" +
	"A [plain](../../docs/spec/lending/the-rule.md), an [anchor](../../docs/spec/lending/the-rule.md#limits), a [dot](./../../docs/spec/lending/the-rule.md),\n" +
	"a [detour](../../docs/spec/lending/../lending/the-rule.md), a [query](../../docs/spec/lending/the-rule.md?x=1#y),\n" +
	"an image ![img](../../docs/spec/lending/the-rule.md) and <a href=\"../../docs/spec/lending/the-rule.md\">a tag</a>,\n" +
	"[encoded](../../docs/spec/lending/the%2Drule.md), [angle](<../../docs/spec/lending/the-rule.md>) and [reference][r].\n\n" +
	"[r]: ../../docs/spec/lending/the-rule.md\n"

// move rewrites in the work every form of a link that check resolves there, so
// that a move creates no error in the work.
func TestMoveRewritesEveryFormOfALinkInTheWork(t *testing.T) {
	files := workFiles()
	files["docs/spec/lending/the-rule.md"] = "# The rule\n\nA rule.\n"
	files[".markitect/plan/forms.md"] = formsOfALink
	dir := project(t, files)
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Fatalf("check before: %d %s", code, out)
	}
	cur := "lending/the-rule"
	for _, to := range []string{"lending/rules/the new rule", "lending/rules/rule"} {
		code, out, errOut := run("move", "-C", dir, cur, to)
		cur = to
		if code != ExitOK || !strings.Contains(out, "changed .markitect/plan/forms.md\n") || strings.Contains(out, "left as it is") {
			t.Fatalf("%d %s %s", code, out, errOut)
		}
		if code, out, _ := run("check", "-C", dir); code != ExitOK {
			t.Errorf("check after a move to %s: %d %s", to, code, out)
		}
		got := read(t, dir, ".markitect/plan/forms.md")
		if strings.Contains(got, "the-rule") || strings.Contains(got, "the%2Drule") {
			t.Errorf("a link still names the old path:\n%s", got)
		}
		for _, part := range []string{"?x=1#y)", "#limits)", "a tag</a>", "![img](", "Type: [Story](../../docs/spec/packages/factory/story.md)\n"} {
			if !strings.Contains(got, part) {
				t.Errorf("the move to %s lost %q:\n%s", to, part, got)
			}
		}
		if strings.Contains(to, " ") && (!strings.Contains(got, "<../../docs/spec/lending/rules/the new rule.md>") || !strings.Contains(got, "the%20new%20rule.md")) {
			t.Errorf("a name with spaces is written wrongly:\n%s", got)
		}
	}
}

// A type line of the work follows a type that moves.
func TestMoveRewritesTheTypeLineOfTheWork(t *testing.T) {
	dir := project(t, workFiles())
	if code, out, errOut := run("move", "-C", dir, "packages/factory/finding", "packages/factory/observation"); code != ExitOK || !strings.Contains(out, "changed .markitect/findings/pending/late-fee.md\n") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	if got := read(t, dir, ".markitect/findings/pending/late-fee.md"); !strings.Contains(got, "Type: [Finding](../../../docs/spec/packages/factory/observation.md)\n") {
		t.Error(got)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Errorf("%d %s", code, out)
	}
}

// The README of a folder moves like any document, and the work follows.
func TestMoveOfAFoldersReadmeRewritesTheWork(t *testing.T) {
	files := workFiles()
	files["docs/spec/lending/README.md"] = "# Lending\n\nSee [renew](renew.md).\n"
	files[".markitect/context/capture/area.md"] = "# Area\n\nThe [area](../../../docs/spec/lending/README.md) and its [folder](../../../docs/spec/lending/).\n"
	dir := project(t, files)
	code, out, errOut := run("move", "-C", dir, "lending/README", "lending/index")
	if code != ExitOK || !strings.Contains(out, "changed .markitect/context/capture/area.md\n") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	want := "# Area\n\nThe [area](../../../docs/spec/lending/index.md) and its [folder](../../../docs/spec/lending/).\n"
	if got := read(t, dir, ".markitect/context/capture/area.md"); got != want {
		t.Errorf("got %q", got)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Errorf("%d %s", code, out)
	}
}

// A document of the work in a folder that .gitignore excludes is not read, so
// a move leaves it alone and does not name it.
func TestMoveLeavesAGitIgnoredFolderOfTheWorkAlone(t *testing.T) {
	files := workFiles()
	files[".gitignore"] = "local/\n"
	files[".markitect/context/local/n.md"] = "# Mine\n\nSee [the rule](../../../docs/spec/lending/renew.md).\n"
	dir := project(t, files)
	code, out, errOut := run("move", "-C", dir, "lending/renew", "lending/renewal")
	if code != ExitOK || strings.Contains(out, "local") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	if got := read(t, dir, ".markitect/context/local/n.md"); got != files[".markitect/context/local/n.md"] {
		t.Error("move changed a document that .gitignore excludes")
	}
}

// What check verifies in the spec is what move rewrites there: a link in an
// HTML tag or an image is not verified by check, and so a move cannot make
// check fail.
func TestMoveCreatesNoErrorInTheSpec(t *testing.T) {
	dir := spec(t, map[string]string{
		"a.md": "# A\n\n[plain](b.md), [anchor](b.md#x), [query](b.md?x=1), ![img](b.md), <a href=\"b.md\">tag</a>, [ref][r]\n\n[r]: b.md\n",
		"b.md": "# B\n",
	})
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Fatal(out)
	}
	if code, _, errOut := run("move", "-C", dir, "b", "sub/c"); code != ExitOK {
		t.Fatal(errOut)
	}
	if code, out, _ := run("check", "-C", dir); code != ExitOK {
		t.Errorf("check after the move: %d %s", code, out)
	}
}
