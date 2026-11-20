package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// snapshot reads a project as compare reads it: the project, the state as
// status takes it, and the history, with the lines of the state and the
// history it cannot read, which it leaves out.
func snapshot(dir string) (*state.Snapshot, []diagnostic, error) {
	w, err := loadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return nil, nil, err
	}
	events, historyProblems, err := state.ReadHistory(w.FS)
	if err != nil {
		return nil, nil, err
	}
	var notes []diagnostic
	for _, p := range problems {
		notes = append(notes, diagnostic{File: state.StateFile, Line: p.Line, Code: state.CodeUnreadable, Message: p.Message})
	}
	for _, p := range historyProblems {
		notes = append(notes, diagnostic{File: state.HistoryFile, Line: p.Line, Code: state.CodeHistoryUnreadable, Message: p.Message})
	}
	return &state.Snapshot{Project: w.Project, Status: s, History: events}, notes, nil
}

// projectAt is a project folder named on the command line, which must hold
// .markitect/version.
func projectAt(arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", fmt.Errorf("not a folder: %s", arg)
	}
	if !isProject(abs) {
		return "", fmt.Errorf("%s holds no %s", arg, state.VersionFile)
	}
	return abs, nil
}

// compareDocument is the JSON document of compare.
type compareDocument struct {
	header
	New         []string      `json:"new"`
	Gone        []string      `json:"gone"`
	Moved       []movedJSON   `json:"moved"`
	Changed     []changedJSON `json:"changed"`
	Suspect     []suspectJSON `json:"suspect"`
	Verdicts    []verdictJSON `json:"verdicts"`
	Unexplained []string      `json:"unexplained"`
	// Notes are the lines of either project's state and history that compare
	// cannot read and leaves out, with the side, "old" or "new".
	Notes []compareNote `json:"notes"`
}

type compareNote struct {
	Side string `json:"side"`
	diagnostic
}

type movedJSON struct {
	From string `json:"from"`
	To   string `json:"to"`
	Diff string `json:"diff"`
}

type changedJSON struct {
	Doc  string `json:"doc"`
	Diff string `json:"diff"`
}

type suspectJSON struct {
	Doc     string   `json:"doc"`
	Because []string `json:"because"`
}

type verdictJSON struct {
	Doc          string   `json:"doc"`
	OldVerdict   string   `json:"old_verdict"`
	NewVerdict   string   `json:"new_verdict"`
	OldArtifacts []string `json:"old_artifacts"`
	NewArtifacts []string `json:"new_artifacts"`
}

// compare compares the project OLD with NEW, by default the project the
// command works in.
func compare(o options, stdout io.Writer) (int, error) {
	oldDir, err := projectAt(o.positional[0])
	if err != nil {
		return ExitFailed, err
	}
	var newDir string
	if len(o.positional) > 1 {
		newDir, err = projectAt(o.positional[1])
	} else {
		newDir, err = mustProject(o)
	}
	if err != nil {
		return ExitFailed, err
	}
	older, oldNotes, err := snapshot(oldDir)
	if err != nil {
		return ExitFailed, err
	}
	newer, newNotes, err := snapshot(newDir)
	if err != nil {
		return ExitFailed, err
	}
	c := state.Compare(older, newer)
	doc := compareDocument{header: newHeader("compare"), New: nonNil(c.New), Gone: nonNil(c.Gone), Moved: []movedJSON{},
		Changed: []changedJSON{}, Suspect: []suspectJSON{}, Verdicts: []verdictJSON{}, Unexplained: nonNil(c.Unexplained)}
	var b strings.Builder
	section := func(title string, n int) {
		fmt.Fprintf(&b, "## %s (%d)\n\n", title, n)
		if n == 0 {
			b.WriteString("(none)\n")
		}
	}
	b.WriteString("# Comparison\n\n")
	doc.Notes = []compareNote{}
	for _, side := range []struct {
		name  string
		notes []diagnostic
	}{{"old", oldNotes}, {"new", newNotes}} {
		for _, n := range side.notes {
			doc.Notes = append(doc.Notes, compareNote{side.name, n})
			fmt.Fprintf(&b, "note: %s: %s, left out\n", side.name, n)
		}
	}
	if len(doc.Notes) > 0 {
		b.WriteString("\n")
	}
	section("New", len(c.New))
	for _, p := range c.New {
		b.WriteString(p + "\n")
	}
	b.WriteString("\n")
	section("Gone", len(c.Gone))
	for _, p := range c.Gone {
		b.WriteString(p + "\n")
	}
	b.WriteString("\n")
	section("Moved", len(c.Moved))
	for _, m := range c.Moved {
		doc.Moved = append(doc.Moved, movedJSON{m.From, m.To, m.Diff})
		b.WriteString(m.From + " -> " + m.To + "\n" + m.Diff)
	}
	b.WriteString("\n")
	section("Changed", len(c.Changed))
	for _, ch := range c.Changed {
		doc.Changed = append(doc.Changed, changedJSON{ch.Doc, ch.Diff})
		b.WriteString(ch.Doc + "\n" + ch.Diff)
	}
	b.WriteString("\n")
	section("Suspect", len(c.Suspect))
	for _, s := range c.Suspect {
		doc.Suspect = append(doc.Suspect, suspectJSON{s.Doc, s.Because})
		b.WriteString(s.Doc + ": " + strings.Join(s.Because, ", ") + "\n")
	}
	b.WriteString("\n")
	section("Verdicts", len(c.Verdicts))
	for _, v := range c.Verdicts {
		doc.Verdicts = append(doc.Verdicts, verdictJSON{v.Doc, v.OldVerdict, v.NewVerdict, v.OldArtifacts, v.NewArtifacts})
		fmt.Fprintf(&b, "%s: %s -> %s; artifacts %s -> %s\n", v.Doc, orNone(v.OldVerdict, "no row"), orNone(v.NewVerdict, "no row"),
			orNone(strings.Join(v.OldArtifacts, ", "), "none"), orNone(strings.Join(v.NewArtifacts, ", "), "none"))
	}
	b.WriteString("\n")
	section("Unexplained", len(c.Unexplained))
	for _, u := range c.Unexplained {
		b.WriteString(u + "\n")
	}
	code, err := output(o, stdout, doc, b.String())
	if err == nil && c.Differs() {
		code = ExitFound
	}
	return code, err
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func orNone(s, none string) string {
	if s == "" {
		return "(" + none + ")"
	}
	return s
}

// figuresDocument is the JSON document of figures.
type figuresDocument struct {
	header
	Verdicts  int                     `json:"verdicts"`
	Moves     int                     `json:"moves"`
	Changes   []changeJSON            `json:"changes"`
	Signals   map[string][]figureJSON `json:"signals"`
	Reference map[string]string       `json:"reference"`
	Weakening string                  `json:"weakening"`
	History   []diagnostic            `json:"history"`
}

type changeJSON struct {
	Time   string   `json:"time"`
	Docs   []string `json:"docs"`
	Events int      `json:"events"`
}

type figureJSON struct {
	Docs   []string `json:"docs"`
	Count  int      `json:"count"`
	Detail string   `json:"detail"`
}

// figures prints the change signals computed from the history.
func figures(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	events, problems, err := state.ReadHistory(w.FS)
	if err != nil {
		return ExitFailed, err
	}
	f := state.ComputeFigures(w.Spec, events)
	doc := figuresDocument{header: newHeader("figures"), Verdicts: f.Verdicts, Moves: f.Moves, Changes: []changeJSON{},
		Signals: map[string][]figureJSON{}, Reference: f.Reference, Weakening: state.Weakening, History: []diagnostic{}}
	var b strings.Builder
	b.WriteString("# Figures from the history\n\n")
	for _, p := range problems {
		d := diagnostic{File: state.HistoryFile, Line: p.Line, Code: state.CodeHistoryUnreadable, Message: p.Message}
		doc.History = append(doc.History, d)
		fmt.Fprintf(&b, "note: %s\n", d)
	}
	for _, c := range f.Changes {
		doc.Changes = append(doc.Changes, changeJSON{c.Time.UTC().Format(time.RFC3339Nano), c.Docs, c.Events})
	}
	fmt.Fprintf(&b, "%s from %s; moves left out, since they are mechanical: %d\n", plural(len(f.Changes), "change"), plural(f.Verdicts, "verdict"), f.Moves)
	for _, s := range state.ChangeSignals {
		list := f.Signals[s]
		doc.Signals[s] = []figureJSON{}
		fmt.Fprintf(&b, "\n### %s (%d)\n\nreference: %s\n", s, len(list), f.Reference[s])
		for i, fig := range list {
			doc.Signals[s] = append(doc.Signals[s], figureJSON{fig.Docs, fig.Count, fig.Detail})
			if !o.all && i == shownPerSignal {
				fmt.Fprintf(&b, "… %d more; --all shows them\n", len(list)-i)
				continue
			}
			if o.all || i < shownPerSignal {
				b.WriteString(fig.Detail + "\n")
			}
		}
	}
	fmt.Fprintf(&b, "\n### weakening\n\n%s\n", state.Weakening)
	return output(o, stdout, doc, b.String())
}
