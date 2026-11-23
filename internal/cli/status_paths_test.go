package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// What check finds in a document does not make it a document without a
// fingerprint: status lists it and confirm binds a verdict to it, with what
// can be read. Whatever bytes a document holds, it parses; only a file that
// cannot be read at all stops the project.
func TestStatusAndConfirmTakeADocumentCheckFindsFault(t *testing.T) {
	const kind = "# Kind\n\nA kind of thing.\n\n## Rule\nWhat holds.\n"
	for name, tc := range map[string]struct {
		text  string
		fault bool // check reports something
	}{
		"an empty section":         {"# Thing\n\nType: [Kind](../kind.md)\n\n## Rule\n\n", true},
		"a missing section":        {"# Thing\n\nType: [Kind](../kind.md)\n", true},
		"a dead link":              {"# Thing\n\nSee [nothing](gone.md).\n", true},
		"a type that is no file":   {"# Thing\n\nType: [Ghost](../ghost.md)\n", true},
		"a type line of no link":   {"# Thing\n\nType: none\n", true},
		"no title":                 {"Just words.\n", true},
		"a second title":           {"# One\n\n# Two\n", true},
		"a stray type line":        {"# Thing\n\nWords.\n\nType: [Kind](../kind.md)\n", true},
		"an empty file":            {"", true},
		"bytes that are no UTF-8":  {"# Thing \xff\xfe\n\nWords.\n", false},
		"a byte order mark":        {"\xef\xbb\xbf# Thing\n\nWords.\n", false},
		"only a link to itself":    {"# Thing\n\nSee [me](thing.md).\n", false},
		"a type that is itself":    {"# Thing\n\nType: [Thing](thing.md)\n", false},
		"a heading of a deep kind": {"# Thing\n\n### Rule\nWords.\n", false},
	} {
		t.Run(name, func(t *testing.T) {
			dir := spec(t, map[string]string{"kind.md": kind, "things/thing.md": tc.text})
			if code, out, _ := run("check", "-C", dir); (code == ExitFound) != tc.fault {
				t.Fatalf("check: %d %s", code, out)
			}
			l := statusOf(t, dir)
			e, which := l.entry("things/thing.md")
			if l.code != ExitFound || which != "changed" || len(e.Fingerprint) != 8 || len(l.State) != 0 {
				t.Fatalf("status: %d %v", l.code, l.summary())
			}
			code, out, errOut := run("confirm", "-C", dir, "things/thing.md@"+e.Fingerprint, "skip", "-m", "not looked at yet")
			if code != ExitOK || !strings.HasPrefix(out, "confirmed things/thing.md skip\n") {
				t.Fatalf("confirm: %d %s %s", code, out, errOut)
			}
			if _, which := statusOf(t, dir).entry("things/thing.md"); which != "" {
				t.Error("the document is still listed after its confirm")
			}
		})
	}
}

// A project whose spec document cannot be read at all gives no answer: exit 2,
// naming the file.
func TestAnUnreadableSpecDocumentStopsStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := spec(t, map[string]string{"a.md": "# A\n", "b.md": "# B\n"})
	locked := filepath.Join(specOf(dir), "b.md")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })
	if _, err := os.ReadFile(locked); err == nil {
		t.Skip("this user reads a file without permission")
	}
	if code, out, errOut := run("status", "-C", dir); code != ExitFailed || out != "" || !strings.Contains(errOut, "b.md") {
		t.Errorf("%d %q %q", code, out, errOut)
	}
}

// Without a state file, with an empty one and with a whole one, status
// answers; a state file that is no file, which cannot be read, gives no answer.
func TestAStateFileThatCannotBeReadStopsStatus(t *testing.T) {
	dir := spec(t, map[string]string{"a.md": "# A\n"})
	if err := os.Mkdir(filepath.Join(dir, ".markitect", "state.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := run("status", "-C", dir); code != ExitFailed || out != "" || !strings.Contains(errOut, "state.jsonl") {
		t.Errorf("%d %q %q", code, out, errOut)
	}
	if code, _, errOut := run("confirm", "-C", dir, "a.md@00000000", "skip", "-m", "x"); code != ExitFailed || !strings.Contains(errOut, "state.jsonl") {
		t.Errorf("confirm: %d %q", code, errOut)
	}
}

// changedLending is the confirmed lending project after the borrowing, the
// member and a new file changed.
func changedLending(t *testing.T) string {
	t.Helper()
	dir := confirmed(t)
	write(t, dir, "docs/spec/lending/borrow.md", read(t, dir, "docs/spec/lending/borrow.md")+"\nFive books at most.\n")
	write(t, dir, "src/fines.go", "package lending\n")
	write(t, dir, "tests/new_test.go", "package lending\n")
	return dir
}

func TestStatusForPaths(t *testing.T) {
	dir := changedLending(t)
	all := statusOf(t, dir).summary()
	same(t, "everything", all, []string{
		"changed lending/borrow.md: self", "suspect roles/member.md: link:lending/borrow.md",
		"unexplained src/fines.go", "unexplained tests/new_test.go"})
	for name, tc := range map[string]struct {
		args []string
		want []string
		code int
	}{
		"a document":                                  {[]string{"lending/borrow"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a document with .md":                         {[]string{"lending/borrow.md"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a document by its project path":              {[]string{"docs/spec/lending/borrow.md"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a document that is suspect":                  {[]string{"roles/member"}, []string{"suspect roles/member.md: link:lending/borrow.md"}, ExitFound},
		"a document that is clean":                    {[]string{"types/operation"}, nil, ExitOK},
		"a folder of the spec":                        {[]string{"docs/spec/lending"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a folder of the spec, relative to it":        {[]string{"lending"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a folder of the spec with a slash":           {[]string{"lending/"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"a folder of the spec that holds a suspect":   {[]string{"roles"}, []string{"suspect roles/member.md: link:lending/borrow.md"}, ExitFound},
		"a folder of the spec with nothing listed":    {[]string{"types"}, nil, ExitOK},
		"the folder of the work":                      {[]string{".markitect"}, nil, ExitOK},
		"a folder of the spec and one of the project": {[]string{"lending", "src"}, []string{"changed lending/borrow.md: self", "unexplained src/fines.go"}, ExitFound},
		"a folder with a slash":                       {[]string{"docs/spec/lending/"}, []string{"changed lending/borrow.md: self"}, ExitFound},
		"the spec folder":                             {[]string{"docs/spec"}, all[:2], ExitFound},
		"a folder of the project":                     {[]string{"src"}, []string{"unexplained src/fines.go"}, ExitFound},
		"a file":                                      {[]string{"tests/new_test.go"}, []string{"unexplained tests/new_test.go"}, ExitFound},
		"a file that is mapped":                       {[]string{"src/loans.go"}, nil, ExitOK},
		"several":                                     {[]string{"roles/member", "src", "lending/borrow.md"}, []string{"changed lending/borrow.md: self", "suspect roles/member.md: link:lending/borrow.md", "unexplained src/fines.go"}, ExitFound},
	} {
		t.Run(name, func(t *testing.T) {
			l := statusOf2(t, dir, tc.args...)
			same(t, "the list", l.summary(), tc.want)
			if l.code != tc.code {
				t.Errorf("exit %d, want %d", l.code, tc.code)
			}
			code, out, _ := run(append([]string{"status", "-C", dir}, tc.args...)...)
			var lines int
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "changed: ") || strings.HasPrefix(line, "suspect: ") || strings.HasPrefix(line, "unexplained: ") {
					lines++
				}
			}
			if code != tc.code || lines != len(tc.want) || !strings.Contains(out, "\n") {
				t.Errorf("text: %d\n%s", code, out)
			}
		})
	}
	for _, bad := range []string{"nothing/here", "lending/borrowz", "../outside", "/abs"} {
		if code, out, errOut := run("status", "-C", dir, bad); code != ExitFailed || out != "" || !strings.Contains(errOut, "unknown path: "+bad) {
			t.Errorf("%s: %d %q %q", bad, code, out, errOut)
		}
	}
}

// A document that is gone has its row still, and status names it by its path.
func TestStatusForAPathOfAGoneDocument(t *testing.T) {
	dir := confirmed(t)
	if err := os.Remove(filepath.Join(specOf(dir), "lending", "return.md")); err != nil {
		t.Fatal(err)
	}
	l := statusOf2(t, dir, "lending/return")
	same(t, "the list", l.summary(), []string{"changed lending/return.md: gone"})
}

// statusOf2 is the JSON document of status for the paths.
func statusOf2(t *testing.T, dir string, paths ...string) listed {
	t.Helper()
	code, out, errOut := run(append([]string{"status", "-C", dir, "--json"}, paths...)...)
	var l listed
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		t.Fatalf("%v: %s %s", err, out, errOut)
	}
	l.code = code
	return l
}

// The files named by --file are all the files that realize the document: a
// second confirm replaces the list, it does not add to it, and says so.
func TestConfirmReplacesTheListOfFiles(t *testing.T) {
	dir := project(t, lending)
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "--file", "src/member.go", "-e", "src/loans.go")
	e, _ := statusOf(t, dir).entry("lending/return.md")
	code, out, errOut := run("confirm", "-C", dir, "lending/borrow.md@"+fingerprint(t, dir, "lending/borrow.md"), "ok", "--file", "src/loans.go", "-e", "src/loans.go")
	if code != ExitOK || !strings.Contains(out, "files that realize it, all of them: src/loans.go\n") || strings.Contains(out, "member.go") {
		t.Fatalf("%d %s %s %v", code, out, errOut, e)
	}
	_, show, _ := run("show", "-C", dir, "lending/borrow")
	if !strings.Contains(show, "Artifacts:\n  src/loans.go\n") || strings.Contains(show, "src/member.go") {
		t.Errorf("show:\n%s", show)
	}
	// With no file, the list is empty.
	code, out, _ = run("confirm", "-C", dir, "lending/borrow.md@"+fingerprint(t, dir, "lending/borrow.md"), "ok", "-e", "src/loans.go", "--json")
	var doc confirmResult
	if err := json.Unmarshal([]byte(out), &doc); err != nil || code != ExitOK || len(doc.Files) != 0 || doc.Files == nil {
		t.Errorf("%v %d %s", err, code, out)
	}
	_, usage, _ := run("help")
	if !strings.Contains(usage, "the files that realize the document,\n                              all of them: a later confirm replaces the list") {
		t.Errorf("the usage does not say that --file replaces:\n%s", usage)
	}
}
