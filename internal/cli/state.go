package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// evaluate reads the state and the files of a loaded project and compares
// them. problems are the lines of the state it cannot read, which it leaves
// out.
func evaluate(w *workspace) (*state.Status, []state.Problem, error) {
	rows, problems, err := state.ReadState(w.FS)
	if err != nil {
		return nil, nil, err
	}
	folder, err := state.ReadFolder(w.FS)
	if err != nil {
		return nil, nil, err
	}
	s, err := state.Evaluate(w.Project, folder, rows)
	return s, problems, err
}

// workOf reads the documents of the work of a loaded project, with what
// check finds in them.
func workOf(w *workspace) (*state.Work, error) {
	return state.ReadWork(w.Project)
}

// statusDocument is the JSON document of status: what it lists, and the lines
// of the state and of the history it cannot read, which it leaves out. What
// check finds in the spec or in the work does not change it.
type statusDocument struct {
	header
	Changed     []statusEntry `json:"changed"`
	Suspect     []statusEntry `json:"suspect"`
	Unexplained []string      `json:"unexplained"`
	Notes       []statusNote  `json:"notes"`
	// State and History hold the lines of the state and of the history that
	// status cannot read: notes, which do not change the exit code.
	State   []diagnostic `json:"state"`
	History []diagnostic `json:"history"`
}

type statusEntry struct {
	Doc         string   `json:"doc"`
	Fingerprint string   `json:"fingerprint"`
	Changes     []string `json:"changes"`
	LookAt      []string `json:"look_at"`
}

type statusNote struct {
	Doc     string `json:"doc"`
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// stateProblems are the lines of the state that cannot be read, as
// diagnostics.
func stateProblems(problems []state.Problem) []diagnostic {
	out := []diagnostic{}
	for _, p := range problems {
		out = append(out, diagnostic{File: state.StateFile, Line: p.Line, Code: state.CodeUnreadable, Message: p.Message})
	}
	return out
}

func (d diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", d.File, d.Line, d.Code, d.Message)
}

// pathFilter says which entries of status the paths on its command line
// name. A path names a document of the spec, with or without .md, or a file
// or a folder of the project, below the project folder; an entry is shown
// when it is that document or file, or lies below that folder.
type pathFilter struct {
	specFolder string
	docs       []string // paths below the spec folder, with .md
	paths      []string // paths below the project folder
}

// newPathFilter reads the paths, which must each name a document of the spec,
// one that has a row in the state, or a file or a folder of the project.
func newPathFilter(w *workspace, s *state.Status, args []string) (*pathFilter, error) {
	f := &pathFilter{specFolder: w.SpecFolder}
	for _, arg := range args {
		known := false
		if doc, ok := edit.DocumentPath(arg); ok {
			_, inSpec := w.Spec.Document(doc)
			_, _, hasRow := s.Row(doc)
			if inSpec || hasRow {
				f.docs = append(f.docs, doc)
				known = true
			}
		}
		if p, ok := state.ProjectPath(arg); ok {
			if _, err := os.Stat(filepath.Join(w.dir, filepath.FromSlash(p))); err == nil {
				f.paths = append(f.paths, p)
				known = true
			}
		}
		if !known {
			return nil, fmt.Errorf("unknown path: %s", arg)
		}
	}
	return f, nil
}

// doc says whether a document of the spec, by its path below the spec folder,
// is named.
func (f *pathFilter) doc(doc string) bool {
	if f == nil {
		return true
	}
	for _, d := range f.docs {
		if d == doc {
			return true
		}
	}
	return f.file(path.Join(f.specFolder, doc))
}

// file says whether a file of the project, by its path below the project
// folder, is named.
func (f *pathFilter) file(file string) bool {
	if f == nil {
		return true
	}
	for _, p := range f.paths {
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

// status lists what is changed, suspect or unexplained, for the paths on the
// command line or for the whole project. It lists regardless of what check
// finds, with what can be read: a document with an empty section, a dead link
// or a type that resolves to nothing is a document like any other.
func status(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	var filter *pathFilter
	if len(o.positional) > 0 {
		if filter, err = newPathFilter(w, s, o.positional); err != nil {
			return ExitFailed, err
		}
	}
	if err := s.Fingerprints(); err != nil {
		return ExitFailed, err
	}
	doc := statusDocument{header: newHeader("status"), Changed: []statusEntry{}, Suspect: []statusEntry{}, Unexplained: []string{},
		Notes: []statusNote{}, State: stateProblems(problems), History: []diagnostic{}}
	var b strings.Builder
	for _, d := range doc.State {
		fmt.Fprintf(&b, "note: %s, left out\n", d)
	}
	shown := 0
	entries := func(kind string, list []state.Entry) []statusEntry {
		out := []statusEntry{}
		for _, e := range list {
			if !filter.doc(e.Doc) {
				continue
			}
			shown++
			out = append(out, statusEntry{Doc: e.Doc, Fingerprint: e.Fingerprint, Changes: e.Changes, LookAt: e.LookAt})
			fmt.Fprintf(&b, "%s: %s@%s: %s\n", kind, e.Doc, e.Fingerprint, strings.Join(e.Changes, ", "))
			if len(e.LookAt) > 0 {
				fmt.Fprintf(&b, "  look at: %s\n", strings.Join(e.LookAt, ", "))
			}
		}
		return out
	}
	doc.Changed = entries(state.Changed, s.Changed)
	doc.Suspect = entries(state.Suspect, s.Suspect)
	for _, u := range s.Unexplained {
		if filter.file(u) {
			shown++
			doc.Unexplained = append(doc.Unexplained, u)
			fmt.Fprintf(&b, "unexplained: %s\n", u)
		}
	}
	_, historyProblems, err := state.ReadHistory(w.FS)
	if err != nil {
		return ExitFailed, err
	}
	for _, p := range historyProblems {
		d := diagnostic{File: state.HistoryFile, Line: p.Line, Code: state.CodeHistoryUnreadable, Message: p.Message}
		doc.History = append(doc.History, d)
		fmt.Fprintf(&b, "note: %s\n", d)
	}
	for _, n := range s.Notes {
		if filter.doc(n.Doc) {
			doc.Notes = append(doc.Notes, statusNote{n.Doc, n.Verdict, n.Reason})
			fmt.Fprintf(&b, "note: %s: %s: %s\n", n.Doc, n.Verdict, n.Reason)
		}
	}
	fmt.Fprintf(&b, "%d changed, %d suspect, %d unexplained\n", len(doc.Changed), len(doc.Suspect), len(doc.Unexplained))
	code, err := output(o, stdout, doc, b.String())
	if err == nil && shown > 0 {
		code = ExitFound
	}
	return code, err
}

// confirmResult is the JSON document of confirm: the document, the verdict, and
// the files that realize it and its counter-checks, all of them, which replace
// the lists of an earlier confirm.
type confirmResult struct {
	header
	Doc     string   `json:"doc"`
	Verdict string   `json:"verdict"`
	Removed bool     `json:"removed"`
	Files   []string `json:"files"`
	Checks  []string `json:"checks"`
}

func confirm(o options, stdout io.Writer) (int, error) {
	docArg, fingerprint, ok := cutLast(o.positional[0], "@")
	if !ok || fingerprint == "" {
		return ExitFailed, fmt.Errorf("%s: name the document as DOCUMENT@FINGERPRINT, as status prints it", o.positional[0])
	}
	verdict := o.positional[1]
	switch {
	case verdict != state.VerdictOK && verdict != state.VerdictFail && verdict != state.VerdictSkip:
		return ExitFailed, fmt.Errorf("verdict %q is none of ok, fail and skip", verdict)
	case verdict != state.VerdictOK && strings.TrimSpace(o.reason) == "":
		return ExitFailed, fmt.Errorf("%s needs a reason with -m", verdict)
	}
	doc, ok := edit.DocumentPath(docArg)
	if !ok {
		return ExitFound, &state.Refusal{Problems: []string{"unknown document: " + docArg}}
	}
	if strings.HasPrefix(doc, state.Dir+"/") {
		return ExitFound, &state.Refusal{Problems: []string{docArg + " is a document of the work: confirm binds documents of the spec"}}
	}
	dir, err := mustProject(o)
	if err != nil {
		return ExitFailed, err
	}
	lock, err := lockProject(dir)
	if err != nil {
		return ExitFailed, err
	}
	defer lock.release()
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	// What check finds in the spec or in the work does not stop a confirm. A
	// line of the state that cannot be read does: confirming writes the state
	// again, and would lose it.
	if unreadable := stateProblems(problems); len(unreadable) > 0 {
		r := &state.Refusal{}
		for _, d := range unreadable {
			r.Problems = append(r.Problems, d.String())
		}
		r.Problems = append(r.Problems, "nothing is confirmed while the state has lines it cannot read: confirming would lose them")
		return ExitFound, r
	}
	_, inSpec := w.Spec.Document(doc)
	if verdict == state.VerdictOK && inSpec && len(o.evidence) == 0 {
		return ExitFailed, errors.New("ok needs evidence with -e")
	}
	rows, err := state.Confirm(s, state.ConfirmRequest{Doc: doc, Fingerprint: fingerprint, Verdict: verdict,
		Files: o.files, Checks: o.checks, Evidence: o.evidence, Reason: o.reason})
	if err != nil {
		var r *state.Refusal
		if errors.As(err, &r) {
			return ExitFound, err
		}
		return ExitFailed, err
	}
	if !lock.held() {
		return ExitFailed, errors.New("another confirm took over the lock of the state; nothing is written")
	}
	inputs, files, checks := "", []string{}, []string{}
	for _, r := range rows {
		if r.Doc == doc {
			inputs = state.InputsDigest(r.Inputs)
			files, checks = append(files, r.Files()...), append(checks, r.Checks()...)
		}
	}
	changes, err := withHistory(dir, edit.Changes{}, rows, s.VerdictEvent(now(), doc, verdict, inputs))
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(dir, changes); err != nil {
		return ExitFailed, err
	}
	// The files named now are all the files that realize the document: they
	// replace the list of an earlier confirm, they do not add to it.
	text := "confirmed " + doc + " " + verdict + "\n" +
		"files that realize it, all of them: " + orNone(strings.Join(files, ", "), "none") + "\n" +
		"counter-checks, all of them: " + orNone(strings.Join(checks, ", "), "none") + "\n"
	if !inSpec {
		text = "removed the row of " + doc + ", which is gone\n"
	}
	return output(o, stdout, confirmResult{newHeader("confirm"), doc, verdict, !inSpec, files, checks}, text)
}

// cutLast cuts s around the last sep.
func cutLast(s, sep string) (string, string, bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

// now is the clock of the history; tests replace it.
var now = func() time.Time { return time.Now().UTC() }

// lineStart is what goes before a line appended to data: a line end when
// data does not end with one.
func lineStart(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] != '\n' {
		return []byte("\n")
	}
	return nil
}
