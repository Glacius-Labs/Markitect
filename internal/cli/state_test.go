package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// The locks of mark go to a temporary cache folder.
	cache, err := os.MkdirTemp("", "markitect-cache-*")
	if err != nil {
		panic(err)
	}
	cacheDir = func() (string, error) { return cache, nil }
	code := m.Run()
	os.RemoveAll(cache)
	os.Exit(code)
}

// lending is a small lending system: a spec of two operations of a type and
// a role, the code that realizes them and a test.
var lending = map[string]string{
	"docs/spec/types/operation.md": "# Operation\n\nSomething a member does.\n\n## Rule\nWhat holds.\n",
	"docs/spec/lending/borrow.md":  "# Borrow\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA [member](../roles/member.md) borrows at most five books.\n",
	"docs/spec/lending/return.md":  "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within three weeks.\n",
	"docs/spec/roles/member.md":    "# Member\n\nA card holder.\n",
	"src/loans.go":                 "package lending\n\nfunc Borrow() {}\n",
	"src/returns.go":               "package lending\n\nfunc Return() {}\n",
	"src/member.go":                "package lending\n\ntype Member struct{}\n",
	"tests/loans_test.go":          "package lending\n\nfunc TestBorrow() {}\n",
	".gitignore":                   "bin/\n",
	".markitect/ignore":            ".gitignore\nnotes.txt\n",
}

// project writes a project of files, set up by init, and returns its folder.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := initialized(t)
	for p, text := range files {
		write(t, dir, p, text)
	}
	return dir
}

func write(t *testing.T, dir, p, text string) {
	t.Helper()
	file := filepath.Join(dir, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, p string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

// listed is the JSON document of status.
type listed struct {
	Errors      []diagnostic  `json:"errors"`
	Changed     []statusEntry `json:"changed"`
	Suspect     []statusEntry `json:"suspect"`
	Unexplained []string      `json:"unexplained"`
	Notes       []statusNote  `json:"notes"`
	code        int
}

func statusOf(t *testing.T, dir string) listed {
	t.Helper()
	code, out, errOut := run("status", "-C", dir, "--json")
	var l listed
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		t.Fatalf("%v: %s %s", err, out, errOut)
	}
	l.code = code
	return l
}

// entry returns the entry of a document, of either kind, and its kind.
func (l listed) entry(doc string) (statusEntry, string) {
	for _, e := range l.Changed {
		if e.Doc == doc {
			return e, "changed"
		}
	}
	for _, e := range l.Suspect {
		if e.Doc == doc {
			return e, "suspect"
		}
	}
	return statusEntry{}, ""
}

// summary is the list in short: "changed DOC: CHANGES" and "suspect DOC:
// CHANGES" per entry, then "unexplained FILE".
func (l listed) summary() []string {
	var out []string
	for _, e := range l.Changed {
		out = append(out, "changed "+e.Doc+": "+strings.Join(e.Changes, ", "))
	}
	for _, e := range l.Suspect {
		out = append(out, "suspect "+e.Doc+": "+strings.Join(e.Changes, ", "))
	}
	for _, u := range l.Unexplained {
		out = append(out, "unexplained "+u)
	}
	return out
}

func same(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s:\ngot  %q\nwant %q", what, got, want)
	}
}

// fingerprint is the fingerprint status gives a document now, listed or not.
func fingerprint(t *testing.T, dir, doc string) string {
	t.Helper()
	if e, kind := statusOf(t, dir).entry(doc); kind != "" {
		return e.Fingerprint
	}
	w, err := load(options{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := evaluate(w)
	if err != nil {
		t.Fatal(err)
	}
	fp, _, err := s.Fingerprint(doc)
	if err != nil {
		t.Fatal(err)
	}
	return fp
}

// verdict binds a verdict at the fingerprint status gives now and expects
// it to succeed.
func verdict(t *testing.T, dir, doc, v string, flags ...string) {
	t.Helper()
	args := append([]string{"mark", "-C", dir, doc + "@" + fingerprint(t, dir, doc), v}, flags...)
	if code, out, errOut := run(args...); code != ExitOK {
		t.Fatalf("%v: %d %s %s", args, code, out, errOut)
	}
}

// marked is the lending project with every document confirmed, so that
// status lists nothing.
func marked(t *testing.T) string {
	t.Helper()
	dir := project(t, lending)
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "--check", "tests/loans_test.go", "-e", "tests/loans_test.go:3 (TestBorrow)")
	verdict(t, dir, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go:3")
	verdict(t, dir, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go")
	verdict(t, dir, "types/operation.md", "ok", "-e", "docs/spec/types/operation.md")
	if l := statusOf(t, dir); l.code != ExitOK || len(l.summary()) != 0 {
		t.Fatalf("not empty after marking everything: %d %v", l.code, l.summary())
	}
	return dir
}

// Scenario 1: a new document is changed; after its mark the list is empty.
func TestNewDocumentsAreChangedUntilMarked(t *testing.T) {
	dir := project(t, lending)
	l := statusOf(t, dir)
	same(t, "the new project", l.summary(), []string{
		"changed lending/borrow.md: no row", "changed lending/return.md: no row",
		"changed roles/member.md: no row", "changed types/operation.md: no row",
		"unexplained src/loans.go", "unexplained src/member.go", "unexplained src/returns.go", "unexplained tests/loans_test.go"})
	if l.code != ExitFound {
		t.Errorf("exit %d", l.code)
	}
	e, _ := l.entry("lending/borrow.md")
	if len(e.Fingerprint) != 8 || strings.Join(e.LookAt, " ") != "docs/spec/lending/borrow.md" {
		t.Errorf("entry %+v", e)
	}
	code, out, _ := run("status", "-C", dir)
	if code != ExitFound || !strings.Contains(out, "changed: lending/borrow.md@"+e.Fingerprint+": no row\n  look at: docs/spec/lending/borrow.md\n") ||
		!strings.HasSuffix(out, "unexplained: tests/loans_test.go\n4 changed, 0 suspect, 4 unexplained\n") {
		t.Error(code, out)
	}
	dir = marked(t)
	code, out, _ = run("status", "-C", dir)
	if code != ExitOK || out != "0 changed, 0 suspect, 0 unexplained\n" {
		t.Error(code, out)
	}
	state := read(t, dir, ".markitect/state.jsonl")
	lines := strings.Split(strings.TrimSuffix(state, "\n"), "\n")
	if len(lines) != 4 || !sort.StringsAreSorted(lines) {
		t.Fatalf("state:\n%s", state)
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatal(err)
	}
	inputs := row["inputs"].(map[string]any)
	var keys []string
	for k := range inputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	same(t, "the inputs of borrow", keys, []string{"check:tests/loans_test.go", "evidence:tests/loans_test.go", "file:src/loans.go",
		"link:roles/member.md", "self", "type:types/operation.md"})
	if !strings.HasPrefix(lines[0], `{"doc":"lending/borrow.md","verdict":"ok","source":"mark","inputs":{`) ||
		!strings.HasSuffix(lines[0], `,"evidence":["tests/loans_test.go:3"],"reason":""}`) {
		t.Error(lines[0])
	}
}

// Scenario 2: a changed text makes its document changed and the documents
// linked in either direction suspect.
func TestChangedTextMakesLinkedDocumentsSuspect(t *testing.T) {
	dir := marked(t)
	write(t, dir, "docs/spec/lending/borrow.md", read(t, dir, "docs/spec/lending/borrow.md")+"\nA book on hold is not lent.\n")
	same(t, "after borrow changed", statusOf(t, dir).summary(), []string{
		"changed lending/borrow.md: self", "suspect roles/member.md: link:lending/borrow.md"})
	write(t, dir, "docs/spec/roles/member.md", "# Member\n\nA card holder who may [return](../lending/return.md) books.\n")
	same(t, "after member links return", statusOf(t, dir).summary(), []string{
		"changed lending/borrow.md: self, link:roles/member.md", "changed roles/member.md: self, link:lending/borrow.md, link:lending/return.md (new)",
		"suspect lending/return.md: link:roles/member.md (new)"})
}

// Scenario 3: a changed type makes its documents suspect.
func TestChangedTypeMakesItsDocumentsSuspect(t *testing.T) {
	dir := marked(t)
	write(t, dir, "docs/spec/types/operation.md", "# Operation\n\nSomething a member does at the desk.\n\n## Rule\nWhat holds.\n")
	same(t, "after the type changed", statusOf(t, dir).summary(), []string{
		"changed types/operation.md: self, evidence:docs/spec/types/operation.md",
		"suspect lending/borrow.md: type:types/operation.md", "suspect lending/return.md: type:types/operation.md"})
}

// Scenario 4: an edited or deleted artifact, or edited evidence, makes the
// document changed.
func TestChangedArtifactsAndEvidence(t *testing.T) {
	dir := marked(t)
	write(t, dir, "src/loans.go", "package lending\n\nfunc Borrow() { panic(1) }\n")
	if err := os.Remove(filepath.Join(dir, "src", "returns.go")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "docs/spec/types/operation.md", read(t, dir, "docs/spec/types/operation.md"))
	same(t, "after the code changed", statusOf(t, dir).summary(), []string{
		"changed lending/borrow.md: file:src/loans.go", "changed lending/return.md: file:src/returns.go (gone), evidence:src/returns.go (gone)"})
	dir = marked(t)
	write(t, dir, "tests/loans_test.go", "package lending\n\nfunc TestBorrow() { t.Skip() }\n")
	l := statusOf(t, dir)
	same(t, "after the test changed", l.summary(), []string{"changed lending/borrow.md: check:tests/loans_test.go, evidence:tests/loans_test.go"})
	e, _ := l.entry("lending/borrow.md")
	same(t, "where to look", e.LookAt, []string{"docs/spec/lending/borrow.md", "src/loans.go", "tests/loans_test.go", "tests/loans_test.go:3"})
}

// Scenario 5: a new file is unexplained until a mark maps it; ignored files
// are not listed.
func TestNewFilesAreUnexplainedUntilMapped(t *testing.T) {
	dir := marked(t)
	write(t, dir, "src/fines.go", "package lending\n")
	write(t, dir, "bin/lending.exe", "binary")
	write(t, dir, "notes.txt", "to do")
	same(t, "after a new file", statusOf(t, dir).summary(), []string{"unexplained src/fines.go"})
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "--file", "src/fines.go", "--check", "tests/loans_test.go", "-e", "tests/loans_test.go")
	if l := statusOf(t, dir); l.code != ExitOK || len(l.summary()) != 0 {
		t.Errorf("%d %v", l.code, l.summary())
	}
	// The named artifacts replace the earlier list.
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "-e", "src/loans.go")
	same(t, "after a mark with fewer artifacts", statusOf(t, dir).summary(), []string{"unexplained src/fines.go", "unexplained tests/loans_test.go"})
}

// Scenario 6: a deleted document with a row is gone, its files unexplained,
// and ok removes its row.
func TestGoneDocument(t *testing.T) {
	dir := marked(t)
	if err := os.Remove(filepath.Join(dir, "docs", "spec", "lending", "return.md")); err != nil {
		t.Fatal(err)
	}
	l := statusOf(t, dir)
	same(t, "after return was deleted", l.summary(), []string{"changed lending/return.md: gone", "unexplained src/returns.go"})
	e, _ := l.entry("lending/return.md")
	same(t, "where to look", e.LookAt, []string{"src/returns.go", "src/returns.go:3"})
	fp := e.Fingerprint
	for _, args := range [][]string{{"fail", "-m", "still here"}, {"ok", "--file", "src/returns.go"}} {
		if code, _, errOut := run(append([]string{"mark", "-C", dir, "lending/return.md@" + fp}, args...)...); code != ExitFound || !strings.Contains(errOut, "is gone from the spec") {
			t.Errorf("%v: %d %s", args, code, errOut)
		}
	}
	verdict(t, dir, "lending/return.md", "ok")
	if strings.Contains(read(t, dir, ".markitect/state.jsonl"), "lending/return.md") {
		t.Error("the row of a gone document stayed")
	}
	same(t, "after its row went", statusOf(t, dir).summary(), []string{"unexplained src/returns.go"})
}

// Scenario 7: fail stays listed until a later ok.
func TestFailStaysListed(t *testing.T) {
	dir := marked(t)
	verdict(t, dir, "lending/borrow.md", "fail", "--file", "src/loans.go", "-m", "lends a sixth\nbook")
	l := statusOf(t, dir)
	same(t, "after fail", l.summary(), []string{"changed lending/borrow.md: fail: lends a sixth book", "unexplained tests/loans_test.go"})
	if code, out, _ := run("show", "-C", dir, "lending/borrow"); code != ExitOK ||
		!strings.HasSuffix(out, "Verdict: fail\nArtifacts:\n  src/loans.go\nCounter-checks: (none)\nEvidence: (none)\nReason: lends a sixth book\n") {
		t.Error(out)
	}
	verdict(t, dir, "lending/borrow.md", "ok", "--file", "src/loans.go", "--check", "tests/loans_test.go", "-e", "tests/loans_test.go:3-4")
	if l := statusOf(t, dir); l.code != ExitOK {
		t.Error(l.summary())
	}
}

// Scenario 8: a file that changes between status and mark refuses the mark.
func TestMarkRefusesAChangeSinceStatus(t *testing.T) {
	dir := marked(t)
	write(t, dir, "docs/spec/lending/borrow.md", read(t, dir, "docs/spec/lending/borrow.md")+"\nOne more rule.\n")
	write(t, dir, "src/holds.go", "package lending\n")
	e, _ := statusOf(t, dir).entry("lending/borrow.md")
	write(t, dir, "src/holds.go", "package lending\n\nfunc Hold() {}\n")
	before := read(t, dir, ".markitect/state.jsonl")
	code, out, errOut := run("mark", "-C", dir, "lending/borrow.md@"+e.Fingerprint, "ok", "--file", "src/loans.go", "--file", "src/holds.go", "-e", "src/loans.go")
	if code != ExitFound || out != "" || !strings.Contains(errOut, "is not the state now") {
		t.Error(code, out, errOut)
	}
	if read(t, dir, ".markitect/state.jsonl") != before {
		t.Error("a refused mark wrote the state")
	}
}

// Scenario 9: two state files joined, as a union merge leaves them: the row
// that matches the folder wins, and two that match with different verdicts
// are a conflict.
func TestMergedStates(t *testing.T) {
	dir := marked(t)
	base := read(t, dir, ".markitect/state.jsonl")
	write(t, dir, "docs/spec/lending/return.md", "# Return\n\nType: [Operation](../types/operation.md)\n\n## Rule\nA book comes back within four weeks.\n")
	verdict(t, dir, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go")
	theirs := read(t, dir, ".markitect/state.jsonl")
	write(t, dir, ".markitect/state.jsonl", theirs+base)
	if l := statusOf(t, dir); l.code != ExitOK {
		t.Errorf("the matching row did not win: %v", l.summary())
	}
	verdict(t, dir, "lending/return.md", "fail", "--file", "src/returns.go", "-m", "four weeks are too long")
	write(t, dir, ".markitect/state.jsonl", read(t, dir, ".markitect/state.jsonl")+theirs)
	same(t, "after a conflict", statusOf(t, dir).summary(), []string{"changed lending/return.md: conflict: fail, ok"})
	if _, out, _ := run("table", "-C", dir); !strings.Contains(out, "| lending/return.md | fail (rows conflict) | src/returns.go |\n") {
		t.Error(out)
	}
	// Another mark keeps the rows of the conflict; one of its own settles it.
	verdict(t, dir, "roles/member.md", "ok", "--file", "src/member.go", "-e", "src/member.go")
	if n := strings.Count(read(t, dir, ".markitect/state.jsonl"), `"doc":"lending/return.md"`); n != 2 {
		t.Errorf("%d rows of the conflict", n)
	}
	verdict(t, dir, "lending/return.md", "ok", "--file", "src/returns.go", "-e", "src/returns.go")
	if n := strings.Count(read(t, dir, ".markitect/state.jsonl"), `"doc":"lending/return.md"`); n != 1 {
		t.Errorf("%d rows after the conflict was settled", n)
	}
	if l := statusOf(t, dir); l.code != ExitOK {
		t.Error(l.summary())
	}
}

// Scenario 10: while the spec has errors, status gives no list and exits 1,
// and mark refuses; so do both with a line of the state they cannot read.
func TestErrorsGiveNoList(t *testing.T) {
	dir := marked(t)
	fp := fingerprint(t, dir, "roles/member.md")
	write(t, dir, "docs/spec/lending/borrow.md", "# Borrow\n\nSee [the fines](fines.md).\n")
	code, out, _ := run("status", "-C", dir)
	if code != ExitFound || out != "lending/borrow.md:3: link.unresolved: no document at \"fines.md\"\nno list while the spec or the state has errors\n" {
		t.Error(code, out)
	}
	if code, _, errOut := run("mark", "-C", dir, "roles/member.md@"+fp, "ok", "-e", "src/member.go"); code != ExitFound || !strings.Contains(errOut, "link.unresolved") {
		t.Error(code, errOut)
	}
	dir = marked(t)
	fp = fingerprint(t, dir, "roles/member.md")
	write(t, dir, ".markitect/state.jsonl", read(t, dir, ".markitect/state.jsonl")+"{\"doc\":\"x.md\"}\nnot json\n")
	l := statusOf(t, dir)
	if l.code != ExitFound || len(l.Errors) != 2 || l.Errors[0].String() != `.markitect/state.jsonl:5: state.unreadable: verdict "" is none of ok, fail and skip` ||
		!strings.HasPrefix(l.Errors[1].String(), ".markitect/state.jsonl:6: state.unreadable: ") || len(l.Changed) != 0 {
		t.Errorf("%d %v", l.code, l.Errors)
	}
	before := read(t, dir, ".markitect/state.jsonl")
	if code, _, errOut := run("mark", "-C", dir, "roles/member.md@"+fp, "ok", "-e", "src/member.go"); code != ExitFound || !strings.Contains(errOut, "state.unreadable") {
		t.Error(code, errOut)
	}
	if read(t, dir, ".markitect/state.jsonl") != before {
		t.Error("a refused mark wrote the state")
	}
}

// Scenario 11: the same project with LF and with CRLF line ends gives the
// same fingerprints.
func TestLineEndingsDoNotChangeFingerprints(t *testing.T) {
	crlf := map[string]string{}
	for p, text := range lending {
		crlf[p] = strings.ReplaceAll(text, "\n", "\r\n")
	}
	a, b := statusOf(t, project(t, lending)), statusOf(t, project(t, crlf))
	if len(a.Changed) != 4 {
		t.Fatal(a.summary())
	}
	for i := range a.Changed {
		if a.Changed[i].Fingerprint != b.Changed[i].Fingerprint {
			t.Errorf("%s: %s with LF, %s with CRLF", a.Changed[i].Doc, a.Changed[i].Fingerprint, b.Changed[i].Fingerprint)
		}
	}
	// A state written on one system holds on the other.
	dir := marked(t)
	for p, text := range crlf {
		if !strings.HasPrefix(p, ".markitect/") {
			write(t, dir, p, text)
		}
	}
	write(t, dir, ".markitect/state.jsonl", strings.ReplaceAll(read(t, dir, ".markitect/state.jsonl"), "\n", "\r\n"))
	if l := statusOf(t, dir); l.code != ExitOK {
		t.Error(l.summary())
	}
}

func TestMarkForms(t *testing.T) {
	dir := marked(t)
	fp := fingerprint(t, dir, "lending/borrow.md")
	at := "lending/borrow.md@" + fp
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"lending/borrow.md", "ok"}, ExitFailed, "DOCUMENT@FINGERPRINT"},
		{[]string{at, "good"}, ExitFailed, `verdict "good"`},
		{[]string{at, "ok"}, ExitFailed, "ok needs evidence"},
		{[]string{at, "fail"}, ExitFailed, "fail needs a reason"},
		{[]string{at, "skip", "-m", " "}, ExitFailed, "skip needs a reason"},
		{[]string{"nobody.md@" + fp, "ok", "-e", "src/loans.go"}, ExitFound, "unknown document: nobody.md"},
		{[]string{at, "ok", "-e", "src/nothing.go"}, ExitFound, "evidence src/nothing.go is no file of the project"},
		{[]string{at, "ok", "-e", "../outside.go"}, ExitFound, "is not path, path:line"},
		{[]string{at, "ok", "-e", "src/loans.go", "--file", "docs/spec/roles/member.md"}, ExitFound, "lies in the spec folder"},
		{[]string{at, "ok", "-e", "src/loans.go", "--file", ".markitect/version"}, ExitFound, "is no file of the project"},
		{[]string{at, "ok", "-e", "src/loans.go", "--file", "notes.txt"}, ExitFound, "is no file of the project"},
		{[]string{at, "ok", "-e", "src/loans.go", "--file", "src/loans.go", "--check", "src/loans.go"}, ExitFound, "as an artifact and as a counter-check"},
		{[]string{"lending/borrow.md@00000000", "ok", "-e", "src/loans.go"}, ExitFound, "is not the state now"},
	} {
		code, out, errOut := run(append([]string{"mark", "-C", dir}, tc.args...)...)
		if code != tc.code || out != "" || !strings.Contains(errOut, tc.want) {
			t.Errorf("%v: %d %q %q", tc.args, code, out, errOut)
		}
	}
	if code, _, errOut := run("check", "-C", dir, "-e", "x"); code != ExitFailed || !strings.Contains(errOut, "flags of mark") {
		t.Error(code, errOut)
	}
	// A skip at the state it was given is a note, not an entry.
	verdict(t, dir, "roles/member.md", "skip", "--file", "src/member.go", "-m", "no members yet")
	code, out, _ := run("status", "-C", dir)
	if code != ExitOK || out != "note: roles/member.md: skip: no members yet\n0 changed, 0 suspect, 0 unexplained\n" {
		t.Error(code, out)
	}
}

func TestStateAsATable(t *testing.T) {
	dir := marked(t)
	code, out, _ := run("table", "-C", dir)
	want := "| Document | Verdict | Artifacts |\n| --- | --- | --- |\n" +
		"| lending/borrow.md | ok | src/loans.go |\n| lending/return.md | ok | src/returns.go |\n" +
		"| roles/member.md | ok | src/member.go |\n| types/operation.md | ok |  |\n"
	if code != ExitOK || out != want {
		t.Errorf("got %d\n%s", code, out)
	}
	_, out, _ = run("show", "-C", dir, "lending/borrow", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["verdict"] != "ok" || len(doc["artifacts"].([]any)) != 1 || len(doc["counter_checks"].([]any)) != 1 || doc["evidence"].([]any)[0] != "tests/loans_test.go:3" {
		t.Error(out)
	}
}
