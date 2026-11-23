package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/lang"
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

// statusDocument is the JSON document of status.
type statusDocument struct {
	header
	Errors      []diagnostic  `json:"errors"`
	Changed     []statusEntry `json:"changed"`
	Suspect     []statusEntry `json:"suspect"`
	Unexplained []string      `json:"unexplained"`
	Notes       []statusNote  `json:"notes"`
	// History holds the lines of the history status cannot read: notes,
	// which do not change the exit code.
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

// blockers are the errors of check, in the spec and in the documents of the
// work, and the unreadable lines of the state, which keep status from giving
// a list and confirm from writing. work is nil where only the spec and the
// state count.
func blockers(w *workspace, work *state.Work, problems []state.Problem) []diagnostic {
	out := []diagnostic{}
	found := lang.Check(w.Spec)
	if work != nil {
		found = append(found, work.Diagnostics...)
	}
	for _, d := range found {
		out = append(out, diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message})
	}
	for _, p := range problems {
		out = append(out, diagnostic{File: state.StateFile, Line: p.Line, Code: state.CodeUnreadable, Message: p.Message})
	}
	return out
}

func (d diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", d.File, d.Line, d.Code, d.Message)
}

func status(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	work, err := workOf(w)
	if err != nil {
		return ExitFailed, err
	}
	doc := statusDocument{header: newHeader("status"), Errors: blockers(w, work, problems),
		Changed: []statusEntry{}, Suspect: []statusEntry{}, Unexplained: []string{}, Notes: []statusNote{}, History: []diagnostic{}}
	if err := s.Fingerprints(); err != nil {
		return ExitFailed, err
	}
	var b strings.Builder
	if len(doc.Errors) > 0 {
		for _, d := range doc.Errors {
			b.WriteString(d.String() + "\n")
		}
		b.WriteString("no list while check finds errors or the state has errors\n")
		code, err := output(o, stdout, doc, b.String())
		if err == nil {
			code = ExitFound
		}
		return code, err
	}
	entries := func(kind string, list []state.Entry) []statusEntry {
		out := []statusEntry{}
		for _, e := range list {
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
		doc.Unexplained = append(doc.Unexplained, u)
		fmt.Fprintf(&b, "unexplained: %s\n", u)
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
		doc.Notes = append(doc.Notes, statusNote{n.Doc, n.Verdict, n.Reason})
		fmt.Fprintf(&b, "note: %s: %s: %s\n", n.Doc, n.Verdict, n.Reason)
	}
	fmt.Fprintf(&b, "%d changed, %d suspect, %d unexplained\n", len(s.Changed), len(s.Suspect), len(s.Unexplained))
	code, err := output(o, stdout, doc, b.String())
	if err == nil && s.Lists() {
		code = ExitFound
	}
	return code, err
}

// confirmResult is the JSON document of confirm.
type confirmResult struct {
	header
	Doc     string `json:"doc"`
	Verdict string `json:"verdict"`
	Removed bool   `json:"removed"`
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
	work, err := workOf(w)
	if err != nil {
		return ExitFailed, err
	}
	if blocked := blockers(w, work, problems); len(blocked) > 0 {
		r := &state.Refusal{}
		for _, d := range blocked {
			r.Problems = append(r.Problems, d.String())
		}
		r.Problems = append(r.Problems, "nothing is confirmed while check finds errors or the state has errors")
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
	inputs := ""
	for _, r := range rows {
		if r.Doc == doc {
			inputs = state.InputsDigest(r.Inputs)
		}
	}
	changes, err := withHistory(dir, edit.Changes{}, rows, s.VerdictEvent(now(), doc, verdict, inputs))
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(dir, changes); err != nil {
		return ExitFailed, err
	}
	text := "confirmed " + doc + " " + verdict + "\n"
	if !inSpec {
		text = "removed the row of " + doc + ", which is gone\n"
	}
	return output(o, stdout, confirmResult{newHeader("confirm"), doc, verdict, !inSpec}, text)
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
