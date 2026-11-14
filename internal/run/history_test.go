package run_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The history (DEC-114) is read in time order, whatever order a merge left its
// lines in, and vet's measures of change are derived from its verdicts.

var t0 = time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)

// event is the line of a verdict event at second s.
func event(s int, def, self string, pending ...string) string {
	p := ""
	if len(pending) > 0 {
		p = fmt.Sprintf(`,"pending":["%s"]`, strings.Join(pending, `","`))
	}
	if self != "" {
		self = fmt.Sprintf(`,"self":%q`, self)
	}
	return fmt.Sprintf(`{"time":%q,"type":"verdict","definition":%q,"verdict":"ok"%s%s}`, t0.Add(time.Duration(s)*time.Second).Format(time.RFC3339Nano), def, self, p)
}

func derived(lines ...string) run.Changes {
	return run.ParseHistory([]byte(strings.Join(lines, "\n") + "\n")).Changes()
}

// After a merge of two branches the history holds the lines of both, one
// branch after the other, and a line both hold maybe twice: it is read by
// time, each line once. A line it cannot read counts nothing, and an event of a
// type it does not know is kept and changes nothing.
func TestParseHistory(t *testing.T) {
	base := event(1, "a", "a1")
	ours := []string{event(2, "a", "a2"), event(5, "a", "a3")}
	theirs := []string{event(3, "b", "b1"), event(4, "b", "b2")}
	data := strings.Join(append(append(append([]string{base}, ours...), theirs...), base, "not json", `{"type":"verdict"}`, fmt.Sprintf(`{"time":%q,"type":"run-started","run":"r1"}`, t0.Add(6*time.Second).Format(time.RFC3339))), "\r\n")
	h := run.ParseHistory([]byte(data))
	var order []string
	for _, e := range h.Events {
		order = append(order, e.Definition+e.Self+e.Type[:1])
	}
	if got := strings.Join(order, " "); got != "aa1v aa2v bb1v bb2v aa3v r" {
		t.Fatalf("order: %s", got)
	}
	if c := h.Changes(); c.Revisions["a"] != 2 || c.Revisions["b"] != 1 {
		t.Fatalf("revisions: %v", c.Revisions)
	}
}

// A verdict at another own text than the Definition's verdict before it is a
// revision; a first verdict is none, and a gone Definition's verdict, which has
// no own text, is none and leaves the text before it as the one to compare.
func TestRevisionsFromTheHistory(t *testing.T) {
	c := derived(
		event(1, "a", "x"), event(2, "a", "x"), event(3, "a", "y"),
		event(4, "a", ""), event(5, "a", "y"), event(6, "a", "x"),
		event(7, "b", "z"),
	)
	if len(c.Revisions) != 1 || c.Revisions["a"] != 2 {
		t.Fatalf("revisions: %v", c.Revisions)
	}
}

// A revision counts a co-change with each Definition pending beside it, once a
// round: the partner's own verdict later sees it verified. Each change counts
// once with each partner, in either order.
func TestCochangesFromTheHistory(t *testing.T) {
	first := []string{event(1, "d", "d0"), event(2, "x", "x0"), event(3, "y", "y0")}
	cases := []struct {
		name  string
		lines []string
		want  int
	}{
		{"one round", []string{event(10, "d", "d1", "x"), event(11, "x", "x1")}, 1},
		{"a first verdict is no revision", []string{event(10, "z", "z0", "x"), event(11, "x", "x1")}, 0},
		// x changes once and waits while d is revised three times.
		{"a waiting change counts once", []string{event(10, "d", "d1", "x"), event(11, "d", "d2", "x"), event(12, "d", "d3", "x"), event(13, "x", "x1")}, 1},
		// d's change counted with x; x changes again before d's verdict, here
		// d's second verdict at the same change of d.
		{"nor the other way", []string{event(10, "x", "x1", "d"), event(11, "d", "d1", "x")}, 1},
		// A new change of x after its verdict counts again.
		{"two rounds", []string{event(10, "d", "d1", "x"), event(11, "x", "x1"), event(12, "x", "x2", "d"), event(13, "d", "d2")}, 2},
	}
	for _, tc := range cases {
		c := derived(append(append([]string{}, first...), tc.lines...)...)
		if c.Cochanges["d"]["x"] != tc.want || c.Cochanges["x"]["d"] != tc.want {
			t.Errorf("%s: %v", tc.name, c.Cochanges)
		}
	}
}

// A round of more than 30 Definitions counts nothing: no verdict while more
// than 30 are pending beside it, the revision included, nor the first verdict
// after, and what that one names as left of the round counts with nothing
// until its own verdict, a Definition that changed beside the round included.
// Definitions that change after the round count, however long ago their last
// verdict was, and so does a member of the round once verified again.
func TestLargeRoundsCountNothing(t *testing.T) {
	var ids []string
	var lines []string
	for i := 0; i < 31; i++ {
		id := fmt.Sprintf("x%02d", i)
		ids = append(ids, id)
		lines = append(lines, event(i, id, id+"-0"))
	}
	lines = append(lines, event(40, "d", "d0"), event(41, "z", "z0"), event(42, "q", "q0"), event(43, "r", "r0"))
	// A round of 30: d and 29 others.
	small := append(append([]string{}, lines...), event(50, "d", "d1", ids[:29]...))
	if c := derived(small...); len(c.Cochanges["d"]) != 29 {
		t.Fatalf("a round of 30: %v", c.Cochanges["d"])
	}
	// A round of 32: d, x00 to x29 and z, which changed beside them. d's verdict
	// gives only the number; x00's still sees 31 with itself; x01's names the 29
	// left, and they are verified one by one, z last.
	at := func(s int, n int) string {
		return fmt.Sprintf(`{"time":%q,"type":"verdict","definition":%q,"verdict":"ok","self":"%s-1","pending_count":%d}`, t0.Add(time.Duration(s)*time.Second).Format(time.RFC3339), map[int]string{50: "d", 51: "x00"}[s], map[int]string{50: "d", 51: "x00"}[s], n)
	}
	large := append(append([]string{}, lines...), at(50, 31), at(51, 30))
	for i := 1; i < 30; i++ {
		large = append(large, event(51+i, ids[i], ids[i]+"-1", append(append([]string{}, ids[i+1:30]...), "z")...))
	}
	large = append(large, event(90, "z", "z1"))
	// w changes after x01's verdict, while the round shrinks: x10's verdict
	// sees it pending, and its own sees x20 and z pending. It is no part of the
	// round, and counts with none of it.
	large = append(large, event(44, "w", "w0"))
	for i, l := range large {
		if strings.Contains(l, `"definition":"x10"`) && strings.Contains(l, `"self":"x10-1"`) {
			large[i] = strings.Replace(l, `"pending":["`, `"pending":["w","`, 1)
		}
	}
	large = append(large, fmt.Sprintf(`{"time":%q,"type":"verdict","definition":"w","verdict":"ok","self":"w1","pending":["x20","z"]}`, t0.Add(70500*time.Millisecond).Format(time.RFC3339Nano)))
	// q and r change together after the round.
	large = append(large, event(95, "q", "q1", "r"), event(96, "r", "r1"))
	if c := derived(large...); len(c.Cochanges) != 2 || c.Cochanges["q"]["r"] != 1 {
		t.Fatalf("a round of 32 counts, or what changed after it does not: %v", c.Cochanges)
	}
	// After its verdict, a Definition of the large round counts again.
	again := append(append([]string{}, large...), event(100, ids[5], ids[5]+"-2", "z"), event(101, "z", "z2"))
	if c := derived(again...); c.Cochanges[ids[5]]["z"] != 1 {
		t.Fatalf("after the large round: %v", c.Cochanges)
	}
}

// A Definition pending beside a revision that has no verdict in the history
// yet, as in a project that had a record before its history, counts nothing:
// its own verdict will be its first, no revision.
func TestCochangesNeedAVerdictBefore(t *testing.T) {
	c := derived(event(1, "d", "d0"), event(2, "d", "d1", "x"), event(3, "x", "x0"))
	if len(c.Cochanges) != 0 {
		t.Fatalf("a partner without a verdict before counts: %v", c.Cochanges)
	}
}

// The counts of a run's start and end: a gone Definition is left to verify but
// no item of the spec, a skip is verified as far as it goes, and the rest is
// verified.
func TestCountsOf(t *testing.T) {
	st := run.Status{Units: 5, Items: []run.Item{{State: run.StateGone}, {State: run.StateFail}, {State: run.StateSpecProblem}, {State: run.StateNew}}, Skipped: []run.Item{{State: run.StateSkip}}}
	got := run.CountsOf(st)
	if want := (run.RunCounts{Verified: 1, Fail: 1, SpecProblem: 1, Skip: 1, Left: 4}); got != want {
		t.Fatalf("counts: %+v, want %+v", got, want)
	}
}
