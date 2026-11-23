package cli

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/library"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// newResult, moveResult and applyResult are the JSON documents of the commands
// that write, with the paths below the spec folder.
type newResult struct {
	header
	Path string `json:"path"`
}

// moveResult is the JSON document of move: the documents of the spec whose
// links it rewrote, below the spec folder; the documents of the work whose
// links it rewrote, below the project folder; and the documents of the work it
// could not rewrite and left as they were, with the reason.
type moveResult struct {
	header
	From    string     `json:"from"`
	To      string     `json:"to"`
	Changed []string   `json:"changed"`
	Work    []string   `json:"work"`
	Left    []leftJSON `json:"left"`
}

type leftJSON struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type applyResult struct {
	header
	Documents []string `json:"documents"`
	Packages  []string `json:"packages"`
}

func newDocument(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	t, err := document(w.Spec, o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	title := ""
	if len(o.positional) > 2 {
		title = o.positional[2]
	}
	p, changes, err := edit.NewDocument(w.Spec, w.SpecFS, t.Path, o.positional[1], title)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(w.specDir(), changes); err != nil {
		return ExitFailed, err
	}
	return output(o, stdout, newResult{newHeader("new"), p}, "created "+p+"\n")
}

// move moves a document and carries the state: under the lock of the state, it
// writes the moved and rewritten files of the spec and of the work, the state
// and the history in one change set, all or nothing.
func move(o options, stdout io.Writer) (int, error) {
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
	rows, problems, err := state.ReadState(w.FS)
	if err != nil {
		return ExitFailed, err
	}
	if blocked := blockers(w, problems); len(blocked) > 0 {
		r := &edit.Refusal{}
		for _, d := range blocked {
			r.Problems = append(r.Problems, d.String())
		}
		r.Problems = append(r.Problems, "nothing is moved while the spec or the state has errors")
		return ExitFailed, r
	}
	m, err := edit.MoveDocument(w.Spec, w.SpecFS, o.positional[0], o.positional[1])
	if err != nil {
		return ExitFailed, err
	}
	// The links of the work to the document are rewritten as those of the
	// spec are; a document of the work that cannot be read or rewritten is
	// left as it is.
	work, err := state.ReadWorkLenient(w.Project)
	if err != nil {
		return ExitFailed, err
	}
	was, now := path.Join(w.SpecFolder, m.From), path.Join(w.SpecFolder, m.To)
	rewrites, left := edit.RelinkWork(work.Documents, was, now)
	// A document of the work that cannot be written stays as it is too: only
	// the spec's own files make the move all or nothing.
	var relinked []edit.Write
	for _, wr := range rewrites {
		if err := canWrite(filepath.Join(dir, filepath.FromSlash(wr.Path))); err != nil {
			left = append(left, edit.Left{Path: wr.Path, Reason: "cannot be written: " + err.Error()})
			continue
		}
		relinked = append(relinked, wr)
	}
	// What cannot be read may link the document or not; the tool cannot know.
	for _, p := range work.Unreadable {
		left = append(left, edit.Left{Path: p, Reason: "could not be read, so it was not searched for links to this document"})
	}
	sort.Slice(left, func(i, j int) bool { return left[i].Path < left[j].Path })
	changes, err := carry(dir, w, m, rows)
	if err != nil {
		return ExitFailed, err
	}
	changes.Writes = append(changes.Writes, relinked...)
	if !lock.held() {
		return ExitFailed, errors.New("another command took over the lock of the state; nothing is moved")
	}
	if err := applyChanges(dir, changes); err != nil {
		return ExitFailed, err
	}
	text := "moved " + m.From + " -> " + m.To + "\n"
	changed, relinkedPaths, leftJSONs := append([]string{}, m.Changed...), []string{}, []leftJSON{}
	for _, p := range changed {
		text += "changed " + p + "\n"
	}
	for _, wr := range relinked {
		relinkedPaths = append(relinkedPaths, wr.Path)
		text += "changed " + wr.Path + "\n"
	}
	for _, l := range left {
		leftJSONs = append(leftJSONs, leftJSON{l.Path, l.Reason})
		text += "left as it is " + l.Path + ": " + l.Reason + "\n"
	}
	return output(o, stdout, moveResult{newHeader("move"), m.From, m.To, changed, relinkedPaths, leftJSONs}, text)
}

func apply(o options, stdout io.Writer) (int, error) {
	dir, err := mustProject(o)
	if err != nil {
		return ExitFailed, err
	}
	folder, err := specFolderOf(dir)
	if err != nil {
		return ExitFailed, err
	}
	target := ""
	if len(o.positional) > 1 {
		target = o.positional[1]
	}
	a, err := edit.Apply(os.DirFS(folder), library.FS(), o.positional[0], target)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(folder, a.Changes); err != nil {
		return ExitFailed, err
	}
	var b strings.Builder
	for _, p := range a.Documents {
		b.WriteString("copied " + p + "\n")
	}
	for _, p := range a.Packages {
		b.WriteString("copied packages/" + p + "/\n")
	}
	return output(o, stdout, applyResult{newHeader("apply"), append([]string{}, a.Documents...), append([]string{}, a.Packages...)}, b.String())
}

// carry turns a move's change set into one below the project folder that
// also writes the state the move carries and the move appended to the history.
func carry(dir string, w *workspace, m edit.Moved, rows []state.Row) (edit.Changes, error) {
	rewritten := map[string]state.Text{}
	for _, wr := range m.Changes.Writes {
		old := wr.Path
		if old == m.To {
			old = m.From
		}
		if d, ok := w.Spec.Document(old); ok {
			rewritten[old] = state.Text{Before: d.Source, After: wr.Data}
		}
	}
	in := func(p string) string { return path.Join(w.SpecFolder, p) }
	var c edit.Changes
	for _, mv := range m.Changes.Moves {
		c.Moves = append(c.Moves, edit.Move{From: in(mv.From), To: in(mv.To)})
	}
	for _, wr := range m.Changes.Writes {
		c.Writes = append(c.Writes, edit.Write{Path: in(wr.Path), Data: wr.Data, Create: wr.Create})
	}
	return withHistory(dir, c, state.MoveRows(rows, m.From, m.To, rewritten),
		state.Event{Time: now(), Type: state.EventMove, From: m.From, To: m.To, Rewritten: m.Changed})
}

// withHistory adds to a change set the state, rewritten from rows when the
// project has a state or rows, and the history with one more event, so that
// both are written with the rest or not at all.
func withHistory(dir string, c edit.Changes, rows []state.Row, event state.Event) (edit.Changes, error) {
	existing := func(name string) ([]byte, bool, error) {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return data, err == nil, err
	}
	_, hasState, err := existing(state.StateFile)
	if err != nil {
		return c, err
	}
	if hasState || len(rows) > 0 {
		c.Writes = append(c.Writes, edit.Write{Path: state.StateFile, Data: state.FormatState(rows), Create: !hasState})
	}
	history, hasHistory, err := existing(state.HistoryFile)
	if err != nil {
		return c, err
	}
	data := append(append(history, lineStart(history)...), event.Line()...)
	c.Writes = append(c.Writes, edit.Write{Path: state.HistoryFile, Data: data, Create: !hasHistory})
	return c, nil
}
