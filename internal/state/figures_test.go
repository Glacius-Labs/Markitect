package state

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

func TestFigures(t *testing.T) {
	spec := lang.NewSpec([]*lang.Document{
		lang.Parse("a.md", []byte("# A\n")),
		lang.Parse("b.md", []byte("# B\n\nSee [a](a.md).\n")),
		lang.Parse("c.md", []byte("# C\n")),
		lang.Parse("e.md", []byte("# E\n")),
	})
	t0 := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	at := func(d time.Duration) time.Time { return t0.Add(d) }
	h := func(s string) string { return Hash([]byte(s)) }
	v := func(when time.Duration, doc, self string, changed ...Pending) Event {
		return Event{Time: at(when), Type: EventVerdict, Doc: doc, Verdict: "ok", Self: self, Changed: changed}
	}
	events := []Event{
		// One change of A, B and C, confirmed one after the other.
		v(0, "a.md", h("a1"), Pending{"b.md", h("b1")}, Pending{"c.md", h("c1")}),
		v(time.Minute, "b.md", h("b1"), Pending{"c.md", h("c1")}),
		v(2*time.Minute, "c.md", h("c1")),
		// A and B again, later.
		v(2*time.Hour, "a.md", h("a2"), Pending{"b.md", h("b2")}),
		v(2*time.Hour+time.Minute, "b.md", h("b2")),
		// D alone, then moved to E: a move is no change.
		v(4*time.Hour, "d.md", h("d1")),
		{Time: at(5 * time.Hour), Type: EventMove, From: "d.md", To: "e.md", Rewritten: []string{"e.md", "b.md"}},
		// E and A, A back at its first text.
		v(6*time.Hour, "e.md", h("e1"), Pending{"a.md", h("a1")}),
		// A, at a third text, and C, far from the change before.
		v(8*time.Hour, "c.md", h("c2"), Pending{"a.md", h("a3")}),
	}
	f := ComputeFigures(spec, events)
	var changes []string
	for _, c := range f.Changes {
		changes = append(changes, strings.Join(c.Docs, "+"))
	}
	if strings.Join(changes, " ") != "a.md+b.md+c.md a.md+b.md e.md a.md+e.md a.md+c.md" || f.Verdicts != 8 || f.Moves != 1 || f.Changes[0].Events != 1 {
		t.Fatalf("changes %v, %d verdicts, %d moves", changes, f.Verdicts, f.Moves)
	}
	top := func(signal string) string {
		var out []string
		for _, fig := range f.Signals[signal] {
			out = append(out, strings.Join(fig.Docs, "+")+"="+itoa(fig.Count))
		}
		return strings.Join(out, " ")
	}
	for signal, want := range map[string]string{
		"wide-change":     "a.md+b.md+c.md=3 a.md+b.md=2 a.md+e.md=2 a.md+c.md=2 e.md=1",
		"unlinked-change": "a.md+c.md=2",
		"dragged-links":   "a.md+b.md=2",
		"hotspot":         "a.md=4",
		"back-and-forth":  "a.md=1",
	} {
		if got := top(signal); got != want {
			t.Errorf("%s: got %s, want %s", signal, got, want)
		}
	}
	if f.Reference["wide-change"] != "median 2, range 1–3 documents per change" {
		t.Error(f.Reference["wide-change"])
	}
	if d := f.Signals["hotspot"][0].Detail; d != "a.md: changed 4 times × linked from 1 document = 4" {
		t.Error(d)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestNoHistoryNoFigures(t *testing.T) {
	f := ComputeFigures(lang.NewSpec(nil), nil)
	for _, s := range ChangeSignals {
		if f.Signals[s] == nil || len(f.Signals[s]) != 0 {
			t.Errorf("%s: %v", s, f.Signals[s])
		}
	}
	if f.Reference["wide-change"] != "no changes" {
		t.Error(f.Reference)
	}
}

func TestDiff(t *testing.T) {
	old := []byte("# Return\r\n\r\nA book comes back within three weeks.\r\nLate books cost a fee.\r\n")
	new := []byte("# Return\n\nA book comes back within four weeks.\nLate books cost a fee.\nA lost book is paid.\n")
	want := "--- docs/spec/return.md\n+++ docs/spec/return.md\n@@ -1,4 +1,5 @@\n # Return\n \n-A book comes back within three weeks.\n+A book comes back within four weeks.\n Late books cost a fee.\n+A lost book is paid.\n"
	if got := Diff("docs/spec/return.md", "docs/spec/return.md", old, new); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if Diff("a", "b", old, []byte(strings.ReplaceAll(string(old), "\r\n", "\n"))) != "" {
		t.Error("line ends made a diff")
	}
	var long []string
	for i := 0; i < 20; i++ {
		long = append(long, "line")
	}
	a := strings.Join(append(append([]string{"start"}, long...), "end"), "\n") + "\n"
	b := strings.Join(append(append([]string{"begin"}, long...), "finish"), "\n") + "\n"
	got := Diff("a", "b", []byte(a), []byte(b))
	if strings.Count(got, "@@ ") != 2 || !strings.Contains(got, "@@ -1,4 +1,4 @@\n-start\n+begin\n line\n line\n line\n") ||
		!strings.Contains(got, "@@ -19,4 +19,4 @@\n line\n line\n line\n-end\n+finish\n") {
		t.Error(got)
	}
	if got := Diff("a", "b", nil, []byte("one\n")); got != "--- a\n+++ b\n@@ -0,0 +1 @@\n+one\n" {
		t.Error(got)
	}
}

// A document waiting unchanged in several events counts once.
func TestWaitingDocumentsCountOnce(t *testing.T) {
	h := func(s string) string { return Hash([]byte(s)) }
	t0 := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	v := func(hours int, doc string) Event {
		return Event{Time: t0.Add(time.Duration(hours) * time.Hour), Type: EventVerdict, Doc: doc, Verdict: "ok", Self: h(doc),
			Changed: []Pending{{"p.md", h("p")}, {"q.md", h("q")}}}
	}
	f := ComputeFigures(lang.NewSpec(nil), []Event{v(0, "x.md"), v(2, "y.md"), v(4, "z.md")})
	var changes []string
	for _, c := range f.Changes {
		changes = append(changes, strings.Join(c.Docs, "+"))
	}
	if strings.Join(changes, " ") != "p.md+q.md+x.md y.md z.md" {
		t.Error(changes)
	}
}

// A document recorded gone ends; a later one at its path is another, whose
// first text is no return.
func TestReusedPath(t *testing.T) {
	h := func(s string) string { return Hash([]byte(s)) }
	t0 := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	at := func(hours int) time.Time { return t0.Add(time.Duration(hours) * time.Hour) }
	events := []Event{
		{Time: at(0), Type: EventVerdict, Doc: "a.md", Verdict: "ok", Self: h("one")},
		{Time: at(2), Type: EventVerdict, Doc: "a.md", Verdict: "ok", Self: h("two")},
		{Time: at(4), Type: EventVerdict, Doc: "a.md", Verdict: "ok"},
		{Time: at(6), Type: EventVerdict, Doc: "a.md", Verdict: "ok", Self: h("one")},
	}
	f := ComputeFigures(lang.NewSpec(nil), events)
	if len(f.Signals["back-and-forth"]) != 0 || len(f.Changes) != 3 {
		t.Errorf("%+v %+v", f.Signals["back-and-forth"], f.Changes)
	}
}

func TestDiffOfALineEnd(t *testing.T) {
	got := Diff("a", "b", []byte("one\ntwo"), []byte("one\ntwo\n"))
	if got != "--- a\n+++ b\n@@ -1,2 +1,2 @@\n one\n-two\n\\ No newline at end of file\n+two\n" {
		t.Errorf("%q", got)
	}
	if got := Diff("a", "b", []byte("one\ntwo"), []byte("one\r\ntwo")); got != "" {
		t.Errorf("%q", got)
	}
}
