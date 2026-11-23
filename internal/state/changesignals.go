package state

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The change signals the history gives, in the order topic shows them.
var ChangeSignals = []string{"wide-change", "unlinked-change", "dragged-links", "hotspot", "back-and-forth"}

// Weakening is why the change signals do not compute the sixth change signal.
const Weakening = "not computed: a strong statement gathering conditions shows only in its old and new text, which an agent reads from Git; the history holds hashes"

// CloseTime is how near in time a verdict must follow the one before it to
// belong to the same change.
const CloseTime = time.Hour

// Change is one change of the spec: the documents changed together, by
// their current paths, at the time of its first verdict, from one or more
// verdict events.
type Change struct {
	Time   time.Time
	Docs   []string
	Events int
}

// Signal is one finding of a change signal: the documents it is about, the
// number it is ranked by, and what it means here.
type Signal struct {
	Docs   []string
	Count  int
	Detail string
}

// HistorySignals are the change signals computed from the history.
type HistorySignals struct {
	// Verdicts and Moves count the events read; moves are left out.
	Verdicts, Moves int
	Changes         []Change
	Signals         map[string][]Signal
	// Reference is the project's own reference for each signal.
	Reference map[string]string
}

// ComputeHistorySignals computes the change signals of the history against the
// spec as it is.
//
// A change is the document of a verdict event and the documents its changed
// list names, each only when its text is not the last one the history
// recorded for it in any event: a document waiting in the list, recorded
// fail, with a changed artifact or gone counts in no change until its text
// changes. A document recorded gone ends there; a later document at its
// path, new or moved there, is another. A verdict joins the change before it when it follows that
// change's last verdict within CloseTime and its documents are among those
// of the change's first verdict, as when a reviewer confirms the documents
// of one change one after the other. Move events are no changes: the history
// follows a document through its moves to its current path, and the
// documents a move rewrote count only when a verdict names them.
func ComputeHistorySignals(spec *lang.Spec, events []Event) *HistorySignals {
	f := &HistorySignals{Signals: map[string][]Signal{}, Reference: map[string]string{}}
	ids := map[string]int{}
	var paths []string
	id := func(p string) int {
		if i, ok := ids[p]; ok {
			return i
		}
		ids[p] = len(paths)
		paths = append(paths, p)
		return ids[p]
	}
	type open struct {
		first map[int]bool
		docs  map[int]bool
		start time.Time
		last  time.Time
		count int
	}
	var changes []*open
	selves := map[int][]string{}
	addSelf := func(i int, h string) {
		if h != "" && (len(selves[i]) == 0 || selves[i][len(selves[i])-1] != h) {
			selves[i] = append(selves[i], h)
		}
	}
	for _, e := range events {
		switch e.Type {
		case EventMove:
			f.Moves++
			i := id(e.From)
			delete(ids, e.From)
			ids[e.To] = i
			paths[i] = e.To
		case EventVerdict:
			f.Verdicts++
			set := map[int]bool{}
			// A document counts when its text is not the last one the
			// history recorded for it: a document waiting, recorded fail,
			// with an edited artifact or gone changed no text. A document
			// recorded gone ends; a later one at its path is another.
			var ended []string
			consider := func(doc, self string) {
				if self == "" {
					ended = append(ended, doc)
					return
				}
				i := id(doc)
				if seq := selves[i]; len(seq) == 0 || seq[len(seq)-1] != self {
					set[i] = true
				}
				addSelf(i, self)
			}
			consider(e.Doc, e.Self)
			for _, c := range e.Changed {
				consider(c.Doc, c.Self)
			}
			for _, doc := range ended {
				delete(ids, doc)
			}
			if len(set) == 0 {
				continue
			}
			if n := len(changes); n > 0 {
				ch := changes[n-1]
				if e.Time.Sub(ch.last) <= CloseTime && subset(set, ch.first) {
					ch.last, ch.count = e.Time, ch.count+1
					continue
				}
			}
			changes = append(changes, &open{first: set, docs: set, start: e.Time, last: e.Time, count: 1})
		}
	}
	name := func(set map[int]bool) []string {
		var out []string
		for i := range set {
			out = append(out, paths[i])
		}
		sort.Strings(out)
		return out
	}
	for _, ch := range changes {
		f.Changes = append(f.Changes, Change{Time: ch.start, Docs: name(ch.docs), Events: ch.count})
	}
	f.wide()
	links, linkedFrom := linkSets(spec)
	f.unlinked(spec, links)
	f.dragged(spec, linkedFrom)
	f.hotspots(spec, linkedFrom)
	f.backAndForth(selves, paths)
	for _, s := range ChangeSignals {
		if f.Signals[s] == nil {
			f.Signals[s] = []Signal{}
		}
	}
	return f
}

func subset(a, b map[int]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// linkSets are the links of the spec: whether two documents are linked in
// either direction, and the documents that link to each.
func linkSets(spec *lang.Spec) (map[[2]string]bool, map[string]map[string]bool) {
	links := map[[2]string]bool{}
	from := map[string]map[string]bool{}
	for _, d := range spec.Documents {
		for _, l := range d.Links {
			if _, ok := spec.Document(l.Target); !ok || l.Target == d.Path {
				continue
			}
			links[pair(d.Path, l.Target)] = true
			if from[l.Target] == nil {
				from[l.Target] = map[string]bool{}
			}
			from[l.Target][d.Path] = true
		}
	}
	return links, from
}

func pair(a, b string) [2]string {
	if b < a {
		a, b = b, a
	}
	return [2]string{a, b}
}

// changesOf counts the changes each document is in.
func (f *HistorySignals) changesOf() map[string]int {
	out := map[string]int{}
	for _, c := range f.Changes {
		for _, d := range c.Docs {
			out[d]++
		}
	}
	return out
}

// wide ranks the changes by their size against the project's other changes.
func (f *HistorySignals) wide() {
	var sizes []int
	for _, c := range f.Changes {
		sizes = append(sizes, len(c.Docs))
	}
	ref := spread(sizes, "documents per change")
	f.Reference["wide-change"] = ref
	var out []Signal
	for _, c := range f.Changes {
		out = append(out, Signal{Docs: c.Docs, Count: len(c.Docs),
			Detail: fmt.Sprintf("%s changed together at %s (%s): %s", count(len(c.Docs), "document"), c.Time.UTC().Format(time.RFC3339), ref, strings.Join(c.Docs, ", "))})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	f.Signals["wide-change"] = out
}

// unlinked lists the pairs of documents of the spec that changed together in
// two changes or more while no link joins them.
func (f *HistorySignals) unlinked(spec *lang.Spec, links map[[2]string]bool) {
	together := map[[2]string]int{}
	for _, c := range f.Changes {
		for i, a := range c.Docs {
			for _, b := range c.Docs[i+1:] {
				together[pair(a, b)]++
			}
		}
	}
	var out []Signal
	for p, n := range together {
		_, okA := spec.Document(p[0])
		_, okB := spec.Document(p[1])
		if n < 2 || !okA || !okB || links[p] {
			continue
		}
		out = append(out, Signal{Docs: []string{p[0], p[1]}, Count: n,
			Detail: fmt.Sprintf("%s and %s changed together in %d of %s, with no link between them", p[0], p[1], n, count(len(f.Changes), "change"))})
	}
	sortSignals(out)
	f.Reference["unlinked-change"] = count(len(f.Changes), "change")
	f.Signals["unlinked-change"] = out
}

// dragged lists the documents of the spec whose changes came, two times or
// more, with changes of documents that link to them.
func (f *HistorySignals) dragged(spec *lang.Spec, linkedFrom map[string]map[string]bool) {
	counts := f.changesOf()
	var out []Signal
	for _, d := range spec.Documents {
		n, with := 0, map[string]bool{}
		for _, c := range f.Changes {
			in, dragged := false, []string{}
			for _, x := range c.Docs {
				if x == d.Path {
					in = true
				} else if linkedFrom[d.Path][x] {
					dragged = append(dragged, x)
				}
			}
			if in && len(dragged) > 0 {
				n++
				for _, x := range dragged {
					with[x] = true
				}
			}
		}
		if n >= 2 {
			out = append(out, Signal{Docs: append([]string{d.Path}, sortedKeys(with)...), Count: n,
				Detail: fmt.Sprintf("%s: %d of its %s came with documents that link to it: %s", d.Path, n, count(counts[d.Path], "change"), strings.Join(sortedKeys(with), ", "))})
		}
	}
	sortSignals(out)
	f.Reference["dragged-links"] = count(len(f.Changes), "change")
	f.Signals["dragged-links"] = out
}

// hotspots ranks the documents of the spec by how often they changed times
// how many documents link to them.
func (f *HistorySignals) hotspots(spec *lang.Spec, linkedFrom map[string]map[string]bool) {
	counts := f.changesOf()
	var changed, linked []int
	var out []Signal
	for _, d := range spec.Documents {
		n, in := counts[d.Path], len(linkedFrom[d.Path])
		changed, linked = append(changed, n), append(linked, in)
		if n*in > 0 {
			out = append(out, Signal{Docs: []string{d.Path}, Count: n * in,
				Detail: fmt.Sprintf("%s: changed %s × linked from %s = %d", d.Path, count(n, "time"), count(in, "document"), n*in)})
		}
	}
	sortSignals(out)
	f.Reference["hotspot"] = fmt.Sprintf("changes per document: median %s; documents linking to one: median %s", median(changed), median(linked))
	f.Signals["hotspot"] = out
}

// backAndForth lists the documents whose text returned to an earlier
// version after another one.
func (f *HistorySignals) backAndForth(selves map[int][]string, paths []string) {
	var out []Signal
	tracked := 0
	for i, seq := range selves {
		tracked++
		seen, returns := map[string]bool{}, 0
		for _, h := range seq {
			if seen[h] {
				returns++
			}
			seen[h] = true
		}
		if returns > 0 {
			out = append(out, Signal{Docs: []string{paths[i]}, Count: returns,
				Detail: fmt.Sprintf("%s: its text returned to an earlier version %s in %s", paths[i], count(returns, "time"), count(len(seq), "version"))})
		}
	}
	sortSignals(out)
	f.Reference["back-and-forth"] = count(tracked, "document") + " with versions in the history"
	f.Signals["back-and-forth"] = out
}

// count is a number and a noun, in the plural unless the number is one.
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// sortSignals sorts by count, highest first, then by the documents.
func sortSignals(list []Signal) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return strings.Join(list[i].Docs, "\n") < strings.Join(list[j].Docs, "\n")
	})
}

// spread is the median and range of a list of numbers.
func spread(values []int, unit string) string {
	if len(values) == 0 {
		return "no changes"
	}
	lo, hi := values[0], values[0]
	for _, v := range values {
		lo, hi = min(lo, v), max(hi, v)
	}
	return fmt.Sprintf("median %s, range %d–%d %s", median(values), lo, hi, unit)
}

func median(values []int) string {
	if len(values) == 0 {
		return "0"
	}
	s := append([]int(nil), values...)
	sort.Ints(s)
	n := len(s)
	if n%2 == 1 {
		return fmt.Sprint(s[n/2])
	}
	return fmt.Sprintf("%g", float64(s[n/2-1]+s[n/2])/2)
}
