package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTree copies a project folder; crlf turns every line end into CRLF.
func copyTree(t *testing.T, from string, crlf bool) string {
	t.Helper()
	to := t.TempDir()
	err := filepath.WalkDir(from, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		text := read(t, from, filepath.ToSlash(rel))
		if crlf {
			text = strings.ReplaceAll(text, "\n", "\r\n")
		}
		write(t, to, filepath.ToSlash(rel), text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return to
}

// lending is a small lending system: a spec of two operations of a type and
// a role.
var lending = map[string]string{
	"types/operation.md": "# Operation\n\nSomething a member does.\n\n## Rule\nWhat holds.\n",
	"lending/borrow.md":  "# Borrow\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA [member](../roles/member.md) borrows at most five books.\n",
	"lending/return.md":  "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within three weeks.\n",
	"roles/member.md":    "# Member\n\nA card holder.\n",
	"lending/holds.md":   "# Holds\n\nA member may hold a book until its [return](return.md).\n",
}

// beforeAndAfter are the lending project and a copy of it after edits: a
// document changed, one gone, one new and one moved.
func beforeAndAfter(t *testing.T) (string, string) {
	t.Helper()
	older := spec(t, lending)
	newer := copyTree(t, older, false)
	write(t, newer, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	if err := os.Remove(filepath.Join(newer, "docs", "spec", "lending", "holds.md")); err != nil {
		t.Fatal(err)
	}
	write(t, newer, "docs/spec/lending/fines.md", "# Fines\n\nA late book costs a fee.\n")
	if err := os.Rename(filepath.Join(newer, "docs", "spec", "roles", "member.md"), filepath.Join(newer, "docs", "spec", "roles", "card-holder.md")); err != nil {
		t.Fatal(err)
	}
	return older, newer
}

func TestDiff(t *testing.T) {
	older, newer := beforeAndAfter(t)
	code, out, errOut := run("diff", "-C", newer, older, "--json")
	if code != ExitFound {
		t.Fatal(code, out, errOut)
	}
	var doc diffDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(doc.New, " "); got != "lending/fines.md roles/card-holder.md" {
		t.Errorf("new %v", doc.New)
	}
	if got := strings.Join(doc.Gone, " "); got != "lending/holds.md roles/member.md" {
		t.Errorf("gone %v", doc.Gone)
	}
	if len(doc.Changed) != 1 || doc.Changed[0].Doc != "lending/return.md" ||
		!strings.Contains(doc.Changed[0].Diff, "--- docs/spec/lending/return.md\n+++ docs/spec/lending/return.md\n@@ -3,4 +3,4 @@\n") ||
		!strings.Contains(doc.Changed[0].Diff, "-A book comes back within three weeks.\n+A book comes back within four weeks.\n") {
		t.Errorf("changed %+v", doc.Changed)
	}
	// The text says the same, and is the same at each run and with CRLF.
	_, text, _ := run("diff", older, newer)
	for _, part := range []string{"# Difference\n\n## New (2)\n\nlending/fines.md\nroles/card-holder.md\n\n", "## Gone (2)\n\nlending/holds.md\nroles/member.md\n\n",
		"## Changed (1)\n\nlending/return.md\n--- docs/spec/lending/return.md\n"} {
		if !strings.Contains(text, part) {
			t.Errorf("missing %q in\n%s", part, text)
		}
	}
	if _, again, _ := run("diff", older, newer); again != text {
		t.Error("two runs differ")
	}
	if _, crlf, _ := run("diff", copyTree(t, older, true), copyTree(t, newer, true)); crlf != text {
		t.Errorf("CRLF differs:\n%s", crlf)
	}
	// A project against itself, or its CRLF copy, differs in nothing.
	code, out, _ = run("diff", older, copyTree(t, older, true))
	if code != ExitOK || strings.Count(out, "(none)") != 5 {
		t.Error(code, out)
	}
	if code, _, errOut := run("diff", t.TempDir()); code != ExitFailed || !strings.Contains(errOut, "holds no .markitect/spec") {
		t.Error(code, errOut)
	}
}

// NEW is by default the project the command works in.
func TestDiffDefaultsToThisProject(t *testing.T) {
	older, newer := beforeAndAfter(t)
	_, want, _ := run("diff", older, newer)
	if code, got, _ := run("diff", "-C", newer, older); code != ExitFound || got != want {
		t.Errorf("got %d\n%s\nwant\n%s", code, got, want)
	}
}

// A document that differs only by a byte order mark and its line ends is
// unchanged.
func TestDiffIgnoresAByteOrderMarkAndLineEnds(t *testing.T) {
	older := spec(t, map[string]string{"a.md": "# A\n\nOne.\n"})
	newer := spec(t, map[string]string{"a.md": "\xef\xbb\xbf# A\r\n\r\nOne.\r\n"})
	code, out, errOut := run("diff", "-C", newer, older)
	if code != ExitOK || errOut != "" || strings.Count(out, "(none)") != 5 {
		t.Errorf("%d %q %q", code, out, errOut)
	}
}

// The spec folders of the two projects may have different names; the diff of a
// changed document names each side by its own spec folder.
func TestDiffComparesSpecFoldersOfDifferentNames(t *testing.T) {
	older := spec(t, map[string]string{"a.md": "# A\n\nOne.\n", "b.md": "# B\n"})
	newer := t.TempDir()
	write(t, newer, ".markitect/spec", "handbook\n")
	write(t, newer, "handbook/a.md", "# A\n\nTwo.\n")
	write(t, newer, "handbook/c.md", "# C\n")
	code, out, errOut := run("diff", "-C", newer, older)
	if code != ExitFound || errOut != "" {
		t.Fatalf("%d %q", code, errOut)
	}
	for _, part := range []string{"## New (1)\n\nc.md\n", "## Gone (1)\n\nb.md\n", "## Changed (1)\n\na.md\n--- docs/spec/a.md\n+++ handbook/a.md\n", "-One.\n+Two.\n"} {
		if !strings.Contains(out, part) {
			t.Errorf("missing %q in\n%s", part, out)
		}
	}
}

// .markitect/spec as a folder is no project, and the message says that it is
// no file, with -C, without it and for the project named to diff.
func TestSpecFileThatIsAFolderIsNoProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".markitect", "spec"), 0o755); err != nil {
		t.Fatal(err)
	}
	const want = ".markitect/spec is no file"
	if code, _, errOut := run("check", "-C", dir); code != ExitFailed || !strings.Contains(errOut, want) || strings.Contains(errOut, "holds no") {
		t.Errorf("-C: %d %q", code, errOut)
	}
	if code, _, errOut := run("diff", dir, "-C", dir); code != ExitFailed || !strings.Contains(errOut, want) {
		t.Errorf("diff: %d %q", code, errOut)
	}
	sub := filepath.Join(dir, "work")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	if code, _, errOut := run("check"); code != ExitFailed || !strings.Contains(errOut, want) || strings.Contains(errOut, "upward holds") {
		t.Errorf("upward: %d %q", code, errOut)
	}
	// The search upward is unchanged: a project further up is still found.
	outer := t.TempDir()
	markitectFile(t, outer)
	write(t, outer, "docs/spec/README.md", readme("A", "a.md"))
	write(t, outer, "docs/spec/a.md", "# A\n")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(filepath.Join(inner, ".markitect", "spec"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(inner)
	if code, out, _ := run("check"); code != ExitOK || out != "0 errors in 2 documents\n" {
		t.Errorf("project above: %d %q", code, out)
	}
}
