package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// HistoryFile holds the history: one event per line, only ever appended to.
const HistoryFile = ".markitect/history.jsonl"

// The kinds of event.
const (
	EventVerdict = "verdict"
	EventMove    = "move"
)

// CodeHistoryUnreadable is the code of a line of the history that cannot be
// read.
const CodeHistoryUnreadable = "history.unreadable"

// Event is one line of the history. Time is UTC, written in RFC 3339.
type Event struct {
	Time time.Time `json:"time"`
	Type string    `json:"type"`
	// A verdict: the document, the verdict, its self hash, absent for a gone
	// document, the InputsDigest of its row, absent for a gone document and
	// in events written before the history recorded it, and every other
	// document status listed as changed then.
	Doc     string    `json:"doc,omitempty"`
	Verdict string    `json:"verdict,omitempty"`
	Self    string    `json:"self,omitempty"`
	Inputs  string    `json:"inputs,omitempty"`
	Changed []Pending `json:"changed,omitempty"`
	// A move: the paths before and after, and the documents whose links it
	// rewrote, by their paths after the move.
	From      string   `json:"from,omitempty"`
	To        string   `json:"to,omitempty"`
	Rewritten []string `json:"rewritten,omitempty"`
}

// Pending is a document listed as changed beside a verdict, with the hash of
// its text then; Self is absent for a document that is gone.
type Pending struct {
	Doc  string `json:"doc"`
	Self string `json:"self,omitempty"`
}

// verdictLine and moveLine fix the fields and their order of the events the
// tool writes.
type verdictLine struct {
	Time    string    `json:"time"`
	Type    string    `json:"type"`
	Doc     string    `json:"doc"`
	Verdict string    `json:"verdict"`
	Self    string    `json:"self,omitempty"`
	Inputs  string    `json:"inputs,omitempty"`
	Changed []Pending `json:"changed"`
}

type moveLine struct {
	Time      string   `json:"time"`
	Type      string   `json:"type"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Rewritten []string `json:"rewritten"`
}

// Line is the event as one line of the history, with its line end.
func (e Event) Line() []byte {
	t := e.Time.UTC().Format(time.RFC3339Nano)
	var v any
	switch e.Type {
	case EventVerdict:
		changed := e.Changed
		if changed == nil {
			changed = []Pending{}
		}
		v = verdictLine{t, e.Type, e.Doc, e.Verdict, e.Self, e.Inputs, changed}
	case EventMove:
		rewritten := e.Rewritten
		if rewritten == nil {
			rewritten = []string{}
		}
		v = moveLine{t, e.Type, e.From, e.To, rewritten}
	default:
		panic("unknown event " + e.Type)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return b.Bytes()
}

// HistoryProblem is a line of the history that cannot be read.
type HistoryProblem struct {
	Line    int
	Message string
}

func (p HistoryProblem) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", HistoryFile, p.Line, CodeHistoryUnreadable, p.Message)
}

// ReadHistory reads the history of the project at fsys; a project without
// one has no events.
func ReadHistory(fsys fs.FS) ([]Event, []HistoryProblem, error) {
	data, err := fs.ReadFile(fsys, HistoryFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	events, problems := ParseHistory(data)
	return events, problems, nil
}

// ParseHistory reads the lines of a history: in time order, keeping the
// file's order among equal times, a line that stands twice, as a union merge
// of two sides that both hold it leaves it, once. It skips events of a type
// it does not know and reports a line it cannot read without failing.
func ParseHistory(data []byte) ([]Event, []HistoryProblem) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var events []Event
	var problems []HistoryProblem
	seen := map[string]bool{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		var head struct {
			Time string `json:"time"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &head); err != nil {
			problems = append(problems, HistoryProblem{i + 1, "not a JSON object of an event: " + err.Error()})
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, head.Time)
		if err != nil || head.Type == "" {
			problems = append(problems, HistoryProblem{i + 1, "an event needs a time in RFC 3339 and a type"})
			continue
		}
		if head.Type != EventVerdict && head.Type != EventMove {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			problems = append(problems, HistoryProblem{i + 1, "not an event of type " + head.Type + ": " + err.Error()})
			continue
		}
		e.Time = t.UTC()
		if msg := e.problem(); msg != "" {
			problems = append(problems, HistoryProblem{i + 1, msg})
			continue
		}
		events = append(events, e)
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, problems
}

func (e Event) problem() string {
	switch e.Type {
	case EventVerdict:
		if !cleanPath(e.Doc) || e.Verdict != VerdictOK && e.Verdict != VerdictFail && e.Verdict != VerdictSkip {
			return "a verdict event needs a doc and ok, fail or skip"
		}
		if e.Inputs != "" && !validHash(e.Inputs) {
			return "the inputs of a verdict event are no sha256 hash"
		}
	case EventMove:
		if !cleanPath(e.From) || !cleanPath(e.To) {
			return "a move event needs from and to"
		}
	}
	return ""
}

// VerdictEvent is the event of a verdict on a document, with the
// InputsDigest of the row it binds, "" for a document that is gone, and
// every other document status lists as changed.
func (s *Status) VerdictEvent(at time.Time, doc, verdict, inputs string) Event {
	e := Event{Time: at, Type: EventVerdict, Doc: doc, Verdict: verdict, Inputs: inputs, Changed: []Pending{}}
	if d, ok := s.docs[doc]; ok && d.current != nil {
		e.Self = d.current[InputSelf]
	}
	for _, c := range s.Changed {
		if c.Doc == doc {
			continue
		}
		p := Pending{Doc: c.Doc}
		if d := s.docs[c.Doc]; d.current != nil {
			p.Self = d.current[InputSelf]
		}
		e.Changed = append(e.Changed, p)
	}
	return e
}

// Text is a document's text before and after a move rewrote it.
type Text struct{ Before, After []byte }

// MoveRows carries the rows of the state through a move of the document from
// to to. rewritten holds, by their paths before the move, the documents
// whose text the move rewrote. The moved document's rows take the new path;
// every type: and link: input that names it names the new path; and a self,
// type: or link: hash that matched a rewritten document's text before the
// move takes the hash of its new text, while one that did not stays, so that
// a document changed before the move stays changed. Every row the move
// touched has the source move.
func MoveRows(rows []Row, from, to string, rewritten map[string]Text) []Row {
	before, after := map[string]string{}, map[string]string{}
	for p, t := range rewritten {
		before[p], after[p] = Hash(t.Before), Hash(t.After)
	}
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		n := r
		n.Inputs = Inputs{}
		touched := false
		if r.Doc == from {
			n.Doc, touched = to, true
		}
		for k, h := range r.Inputs {
			target := ""
			switch {
			case k == InputSelf:
				target = r.Doc
			case strings.HasPrefix(k, InputType):
				target = strings.TrimPrefix(k, InputType)
			case strings.HasPrefix(k, InputLink):
				target = strings.TrimPrefix(k, InputLink)
			}
			if target != "" {
				if was, ok := before[target]; ok && h == was {
					h, touched = after[target], true
				}
				if target == from && k != InputSelf {
					k, touched = strings.TrimSuffix(k, from)+to, true
				}
			}
			n.Inputs[k] = h
		}
		if touched {
			n.Source = "move"
		}
		out = append(out, n)
	}
	return out
}
