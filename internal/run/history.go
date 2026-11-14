package run

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The history is the project's process data (DEC-114): what happened, one event
// per line, appended and never rewritten. The record says what is verified now;
// the history says how the work went, and vet derives its measures of change
// from it at read time, so that a threshold tuned later applies to the past too.

// EventVerdict is the type of the event mark appends for each verdict it records.
const EventVerdict = "verdict"

// maxChangeSet is the most Definitions one round of change may hold for its
// co-changes to count: a larger batch, such as a reorganisation, says nothing
// about which of them belong together. It is code-maat's largest change set.
// A verdict event names the pending Definitions only up to it, and beyond it
// gives their number.
const maxChangeSet = 30

// Event is one line of the history: when, and what type of event, with the
// fields of its type. A type the reader does not know is kept and skipped, so
// new types need no format version.
type Event struct {
	Time time.Time `json:"time"`
	Type string    `json:"type"`
	// A verdict: on which Definition, which, the hash of its own text when it
	// has one (a gone Definition has none), and the reviewer's counts.
	Definition string `json:"definition,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
	Self       string `json:"self,omitempty"`
	Cases      *int   `json:"cases,omitempty"`
	Covered    *int   `json:"covered,omitempty"`
	// Pending names, sorted, the other Definitions whose own text had changed
	// since their last verdict and was not verified again yet when the verdict
	// was recorded; PendingCount gives only their number when there were more
	// than maxChangeSet.
	Pending      []string `json:"pending,omitempty"`
	PendingCount int      `json:"pending_count,omitempty"`
}

// pending is how many other Definitions were pending at a verdict.
func (e Event) pending() int {
	if e.PendingCount > 0 {
		return e.PendingCount
	}
	return len(e.Pending)
}

// History is the events of a project's history, in time order.
type History struct {
	Events []Event
}

// ParseHistory reads a history: one JSON object per line. After a merge the
// lines of two branches lie one after the other, so it sorts them by time,
// keeping the order of the file among equal times, and keeps a line that both
// branches hold once. A line it cannot read, or one without a time or a type,
// counts nothing.
func ParseHistory(data []byte) History {
	var h History
	seen := map[string]bool{}
	for _, raw := range strings.Split(string(project.Normalize(data)), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		var e Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil || e.Time.IsZero() || e.Type == "" {
			continue
		}
		h.Events = append(h.Events, e)
	}
	sort.SliceStable(h.Events, func(i, j int) bool { return h.Events[i].Time.Before(h.Events[j].Time) })
	return h
}

// Changes is what the history shows about how a project's Definitions changed
// between verdicts.
type Changes struct {
	// Revisions counts, per Definition, how often its own text changed between
	// two verdicts on it.
	Revisions map[string]int
	// Cochanges counts, per pair of Definitions in both directions, how often
	// their own texts changed in one round.
	Cochanges map[string]map[string]int
}

// Changes derives from the verdicts, in time order, how often each Definition's
// own text changed between two verdicts on it, and how often two Definitions'
// own texts changed in one round (DEC-102).
//
// A verdict at another own text than the Definition's verdict before it is a
// revision. A revision counts a co-change with each Definition pending beside it.
// Each change counts once with each partner: a pending change that counted with
// a Definition counts with it no more, however often that one is revised before
// its verdict, and the verdict on it counts with none it counted with already.
// A round of more than maxChangeSet Definitions, the revision included, counts
// nothing, and neither does any change verified after such a round was seen and
// before its verdict: what changes during a reorganisation is part of it, and a
// round shrinks as it is verified. A Definition's verdict ends its pending
// change. Definitions gone from the spec stay in the counts; whoever reads them
// asks the spec.
func (h History) Changes() Changes {
	c := Changes{Revisions: map[string]int{}, Cochanges: map[string]map[string]int{}}
	last := map[string]string{}
	// counted holds, per Definition, the partners its pending change counted
	// with; large counts the large rounds seen, and largeAt how many had been
	// seen at a Definition's last verdict.
	counted := map[string]map[string]bool{}
	large, largeAt := 0, map[string]int{}
	add := func(a, b string) {
		for _, p := range [][2]string{{a, b}, {b, a}} {
			if c.Cochanges[p[0]] == nil {
				c.Cochanges[p[0]] = map[string]int{}
			}
			c.Cochanges[p[0]][p[1]]++
		}
	}
	for _, e := range h.Events {
		if e.Type != EventVerdict || e.Definition == "" {
			continue
		}
		d := e.Definition
		was, seen := last[d]
		revised := seen && e.Self != "" && was != "" && e.Self != was
		if e.Self != "" {
			last[d] = e.Self
		}
		round := e.pending()
		if revised {
			c.Revisions[d]++
			round++
		}
		if round > maxChangeSet {
			large++
		}
		if revised && large == largeAt[d] {
			for _, x := range e.Pending {
				if x == d || large != largeAt[x] || counted[d][x] || counted[x][d] {
					continue
				}
				add(d, x)
				if counted[x] == nil {
					counted[x] = map[string]bool{}
				}
				counted[x][d] = true
			}
		}
		delete(counted, d)
		largeAt[d] = large
	}
	return c
}

// appendHistory adds an event to a project's history as its last line, in one
// step: a crash leaves the history as it was or with the line. mark calls it
// under the lock of the record.
func appendHistory(root string, e Event) error {
	data, err := readOwnFile(root, project.HistoryFile)
	if err != nil {
		return err
	}
	line, err := marshal(e)
	if err != nil {
		return err
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, line+"\n"...)
	return writeAtomic(filepath.Join(root, filepath.FromSlash(project.Dir)), filepath.Base(project.HistoryFile), data)
}
