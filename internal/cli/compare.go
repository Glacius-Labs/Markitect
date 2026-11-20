package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/lang"
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
	f, err := factoryOf(w)
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
	return &state.Snapshot{Project: w.Project, Status: s, History: events, Reports: f.Reports}, notes, nil
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
	Rewritten   []string      `json:"rewritten"`
	Changed     []changedJSON `json:"changed"`
	Suspect     []suspectJSON `json:"suspect"`
	Verdicts    []verdictJSON `json:"verdicts"`
	Unexplained []string      `json:"unexplained"`
	Reports     []reportJSON  `json:"reports"`
	// Notes are the lines of either project's state and history that compare
	// cannot read and leaves out, with the side, "old" or "new".
	Notes []compareNote `json:"notes"`
}

type compareNote struct {
	Side string `json:"side"`
	diagnostic
}

// reportJSON is a report that is new, closed, changed or gone; closing is
// the first line of a closed report's closing.
type reportJSON struct {
	Name    string `json:"name"`
	Change  string `json:"change"`
	Closing string `json:"closing"`
	Diff    string `json:"diff"`
}

type movedJSON struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Diff       string `json:"diff"`
	Mechanical bool   `json:"mechanical"`
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
	c := state.Compare(older, newer, replay(older, newer))
	doc := compareDocument{header: newHeader("compare"), New: nonNil(c.New), Gone: nonNil(c.Gone), Moved: []movedJSON{}, Rewritten: nonNil(c.Rewritten),
		Changed: []changedJSON{}, Suspect: []suspectJSON{}, Verdicts: []verdictJSON{}, Unexplained: nonNil(c.Unexplained), Reports: []reportJSON{}}
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
		doc.Moved = append(doc.Moved, movedJSON{m.From, m.To, m.Diff, m.Mechanical})
		if m.Mechanical {
			b.WriteString(m.From + " -> " + m.To + " (links rewritten only)\n")
		} else {
			b.WriteString(m.From + " -> " + m.To + "\n" + m.Diff)
		}
	}
	b.WriteString("\n")
	section("Rewritten", len(c.Rewritten))
	for _, p := range c.Rewritten {
		b.WriteString(p + " (links rewritten only)\n")
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
	b.WriteString("\n")
	section("Reports", len(c.Reports))
	for _, r := range c.Reports {
		doc.Reports = append(doc.Reports, reportJSON{r.Name, r.Kind, r.Closing, r.Diff})
		switch r.Kind {
		case state.ReportClosed:
			b.WriteString(r.Kind + ": " + r.Name + ": " + r.Closing + "\n" + r.Diff)
		case state.ReportChanged:
			b.WriteString(r.Kind + ": " + r.Name + "\n" + r.Diff)
		default:
			b.WriteString(r.Kind + ": " + r.Name + "\n")
		}
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
	// The figures of the factory, whose references join reference.
	Samples        []sampleJSON        `json:"samples"`
	Planted        int                 `json:"planted"`
	Hit            int                 `json:"hit"`
	Reverdicts     int                 `json:"reverdicts"`
	Contradictions []contradictionJSON `json:"contradictions"`
	Reports        reportsJSON         `json:"reports"`
	// StoppedJobs is not computed yet and always null.
	StoppedJobs *int `json:"stopped_jobs"`
	// Notes are what check finds in the files of the factory; the figures
	// leave out each value that is not of its form.
	Notes []diagnostic `json:"notes"`
}

type sampleJSON struct {
	Name    string   `json:"name"`
	Day     string   `json:"day"`
	Planted int      `json:"planted"`
	Hit     int      `json:"hit"`
	Missed  []string `json:"missed"`
}

type contradictionJSON struct {
	Doc  string `json:"doc"`
	Was  string `json:"was"`
	Now  string `json:"now"`
	Time string `json:"time"`
}

type reportsJSON struct {
	Total     int             `json:"total"`
	Open      int             `json:"open"`
	Days      []dayJSON       `json:"days"`
	Areas     []areaJSON      `json:"areas"`
	Recurring []recurringJSON `json:"recurring"`
}

type dayJSON struct {
	Day    string `json:"day"`
	Raised int    `json:"raised"`
	Closed int    `json:"closed"`
	Open   int    `json:"open"`
}

type areaJSON struct {
	Area    string `json:"area"`
	Reports int    `json:"reports"`
	Open    int    `json:"open"`
}

type recurringJSON struct {
	Kind    string   `json:"kind"`
	Place   string   `json:"place"`
	Reports []string `json:"reports"`
	Detail  string   `json:"detail"`
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

// figures prints the change signals computed from the history, and the
// figures of the factory computed from the results of control samples, the
// history and the reports.
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
	factory, err := factoryOf(w)
	if err != nil {
		return ExitFailed, err
	}
	factoryFigures(o, &doc, &b, state.ComputeFactoryFigures(w.Project, factory, events), factory)
	return output(o, stdout, doc, b.String())
}

// factoryFigures adds the figures of the factory to the document and the
// text of figures.
func factoryFigures(o options, doc *figuresDocument, b *strings.Builder, ff *state.FactoryFigures, f *state.Factory) {
	doc.Samples, doc.Contradictions, doc.Notes = []sampleJSON{}, []contradictionJSON{}, []diagnostic{}
	doc.Reports = reportsJSON{Total: ff.Reports, Open: ff.Open, Days: []dayJSON{}, Areas: []areaJSON{}, Recurring: []recurringJSON{}}
	doc.Planted, doc.Hit, doc.Reverdicts = ff.Planted, ff.Hit, ff.Reverdicts
	for k, v := range ff.Reference {
		doc.Reference[k] = v
	}
	b.WriteString("\n# Figures of the factory\n")
	if len(f.Diagnostics) > 0 {
		b.WriteString("\n")
	}
	for _, d := range f.Diagnostics {
		n := diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message}
		doc.Notes = append(doc.Notes, n)
		fmt.Fprintf(b, "note: %s\n", n)
	}
	heading := func(name string, n int) {
		fmt.Fprintf(b, "\n### %s (%d)\n\nreference: %s\n", name, n, ff.Reference[name])
	}
	// list prints the first lines of a figure, or every one with --all.
	list := func(lines []string) {
		for i, line := range lines {
			if !o.all && i == shownPerSignal {
				fmt.Fprintf(b, "… %d more; --all shows them\n", len(lines)-i)
				return
			}
			b.WriteString(line + "\n")
		}
	}
	heading("hit-rate", len(ff.Samples))
	var lines []string
	for _, s := range ff.Samples {
		doc.Samples = append(doc.Samples, sampleJSON{s.Name, s.Day, s.Planted, s.Hit, s.Missed})
		line := fmt.Sprintf("%s (%s): %d of %d found", s.Name, orNone(s.Day, "no day"), s.Hit, s.Planted)
		if len(s.Missed) > 0 {
			line += "; missed " + strings.Join(s.Missed, ", ")
		}
		lines = append(lines, line)
	}
	list(lines)
	heading("contradictions", len(ff.Contradictions))
	lines = nil
	for _, c := range ff.Contradictions {
		t := c.Time.UTC().Format(time.RFC3339Nano)
		doc.Contradictions = append(doc.Contradictions, contradictionJSON{c.Doc, c.Was, c.Now, t})
		lines = append(lines, fmt.Sprintf("%s: %s, then %s at %s on the same inputs", c.Doc, c.Was, c.Now, t))
	}
	list(lines)
	heading("open-reports", ff.Open)
	lines = nil
	for _, d := range ff.Days {
		doc.Reports.Days = append(doc.Reports.Days, dayJSON{d.Day, d.Raised, d.Closed, d.Open})
		lines = append(lines, fmt.Sprintf("%s: %d raised, %d closed, %d open", d.Day, d.Raised, d.Closed, d.Open))
	}
	list(lines)
	heading("findings-per-area", len(ff.Areas))
	lines = nil
	for _, a := range ff.Areas {
		doc.Reports.Areas = append(doc.Reports.Areas, areaJSON{a.Name, a.Reports, a.Open})
		lines = append(lines, fmt.Sprintf("%s: %s, %d open", a.Name, plural(a.Reports, "report"), a.Open))
	}
	list(lines)
	heading("recurring-reports", len(ff.Recurring))
	lines = nil
	for _, r := range ff.Recurring {
		doc.Reports.Recurring = append(doc.Reports.Recurring, recurringJSON{r.Kind, r.Place, r.Reports, r.Detail})
		lines = append(lines, r.Detail)
	}
	list(lines)
	fmt.Fprintf(b, "\n### stopped-jobs\n\n%s\n", state.StoppedJobs)
}

// replay makes, from the older project's documents, the texts the newer
// project's moves after the older history's last event alone give the
// documents they rewrote, by their paths in the newer project. It replays
// each move as move computes it, and stops at one it cannot replay.
func replay(older, newer *state.Snapshot) map[string][]byte {
	var last time.Time
	for _, e := range older.History {
		if e.Time.After(last) {
			last = e.Time
		}
	}
	texts := map[string][]byte{}
	for _, d := range older.Project.Spec.Documents {
		texts[d.Path] = d.Source
	}
	rewritten := map[string]bool{}
	for _, e := range newer.History {
		if e.Type != state.EventMove || !e.Time.After(last) {
			continue
		}
		fsys := fstest.MapFS{}
		var docs []*lang.Document
		for p, text := range texts {
			fsys[p] = &fstest.MapFile{Data: text}
			docs = append(docs, lang.Parse(p, text))
		}
		m, err := edit.MoveDocument(lang.NewSpec(docs), fsys, e.From, e.To)
		if err != nil {
			break
		}
		texts[m.To] = texts[m.From]
		delete(texts, m.From)
		if rewritten[m.From] {
			delete(rewritten, m.From)
			rewritten[m.To] = true
		}
		for _, w := range m.Changes.Writes {
			texts[w.Path] = w.Data
			rewritten[w.Path] = true
		}
		rewritten[m.To] = true
	}
	out := map[string][]byte{}
	for p := range rewritten {
		out[p] = texts[p]
	}
	return out
}
