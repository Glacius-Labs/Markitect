package signals

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The signals, in the order they are shown.
const (
	Weak               = "weak"
	StrongStatements   = "strong"
	Complexity         = "complexity"
	Size               = "size"
	TextBeforeSections = "text-before-sections"
	LinkGroup          = "link-group"
	ScatteredRules     = "scattered-rules"
	UnlinkedMention    = "unlinked-mention"
)

// Names lists the signals in the order they are shown.
var Names = []string{Weak, StrongStatements, Complexity, Size, TextBeforeSections, LinkGroup, ScatteredRules, UnlinkedMention}

// Finding is one entry of a signal's list, at a line of a document.
type Finding struct {
	Signal string
	Path   string
	Line   int
	Detail string
	// Docs are the documents it concerns, its own first.
	Docs []string
	rank float64
}

// String renders the finding as path:line: signal: detail.
func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", f.Path, f.Line, f.Signal, f.Detail)
}

// Analysis is what the signals found in a spec.
type Analysis struct {
	Spec       *lang.Spec
	Statements []Statement
	// Findings holds each signal's list, strongest first.
	Findings map[string][]Finding
	// Mentions holds, per statement index, the documents it names by title.
	Mentions [][]string
	links    map[string]map[string]bool // the documents each document links to
	inbound  map[string]map[string]bool // the documents that link to or mention each document
	refs     References
}

// Analyze computes the statements and the signals of a spec, with refs as the
// reference values by type.
func Analyze(spec *lang.Spec, refs References) *Analysis {
	a := &Analysis{Spec: spec, Findings: map[string][]Finding{}, links: map[string]map[string]bool{},
		inbound: map[string]map[string]bool{}, refs: refs}
	for _, d := range spec.Documents {
		a.Statements = append(a.Statements, Statements(d)...)
		a.links[d.Path] = map[string]bool{}
		for _, r := range d.Refs {
			a.links[d.Path][r.Target] = true
		}
	}
	titles := newTitles(spec)
	for _, st := range a.Statements {
		a.Mentions = append(a.Mentions, titles.find(st.Words, st.Doc))
	}
	for from, targets := range a.links {
		for t := range targets {
			a.addInbound(t, from)
		}
	}
	for i, st := range a.Statements {
		for _, t := range a.Mentions[i] {
			a.addInbound(t, st.Doc)
		}
	}
	a.weak()
	a.strong()
	a.complexity()
	a.size()
	a.textBeforeSections()
	a.linkGroups()
	a.scatteredRules()
	a.unlinkedMentions()
	for name, list := range a.Findings {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].rank != list[j].rank {
				return list[i].rank > list[j].rank
			}
			if list[i].Path != list[j].Path {
				return list[i].Path < list[j].Path
			}
			return list[i].Line < list[j].Line
		})
		a.Findings[name] = list
	}
	return a
}

func (a *Analysis) addInbound(to, from string) {
	if to == from {
		return
	}
	if a.inbound[to] == nil {
		a.inbound[to] = map[string]bool{}
	}
	a.inbound[to][from] = true
}

func (a *Analysis) add(f Finding) { a.Findings[f.Signal] = append(a.Findings[f.Signal], f) }

// refersTo returns the documents a statement links to or names by title.
func (a *Analysis) refersTo(i int) []string {
	out := append([]string(nil), a.Statements[i].Targets...)
	for _, m := range a.Mentions[i] {
		if !contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}

func quoteText(s string) string {
	const max = 100
	if r := []rune(s); len(r) > max {
		s = string(r[:max-1]) + "…"
	}
	return `"` + s + `"`
}

func (a *Analysis) weak() {
	for _, st := range a.Statements {
		n := len(st.Conditions) + len(st.Vague)
		if n == 0 {
			continue
		}
		var parts []string
		if len(st.Conditions) > 0 {
			parts = append(parts, "conditions "+strings.Join(st.Conditions, ", "))
		}
		if len(st.Vague) > 0 {
			parts = append(parts, "vague "+strings.Join(st.Vague, ", "))
		}
		a.add(Finding{Signal: Weak, Path: st.Doc, Line: st.Line, Docs: []string{st.Doc}, rank: float64(n),
			Detail: fmt.Sprintf("%d (%s): %s", n, strings.Join(parts, "; "), quoteText(st.Text))})
	}
}

func (a *Analysis) strong() {
	for i, st := range a.Statements {
		if len(st.Strong) == 0 {
			continue
		}
		subjects := a.refersTo(i)
		if len(subjects) == 0 {
			subjects = []string{st.Doc}
		}
		best, reach := subjects[0], -1
		for _, s := range subjects {
			if n := len(a.inbound[s]); n > reach {
				best, reach = s, n
			}
		}
		a.add(Finding{Signal: StrongStatements, Path: st.Doc, Line: st.Line, Docs: append([]string{st.Doc}, subjects...), rank: float64(reach),
			Detail: fmt.Sprintf("%s about %s, which %s refer to: %s", strings.Join(st.Strong, ", "), best, plural(reach, "document"), quoteText(st.Text))})
	}
}

func (a *Analysis) complexity() {
	var counts []int
	for _, st := range a.Statements {
		counts = append(counts, len(st.Words))
	}
	med, _, max := spread(counts)
	for _, st := range a.Statements {
		ref := ""
		if r, ok := a.reference(st.Doc); ok {
			ref = fmt.Sprintf("; reference %d–%d", r.Words[0], r.Words[1])
		}
		n := len(st.Words)
		a.add(Finding{Signal: Complexity, Path: st.Doc, Line: st.Line, Docs: []string{st.Doc},
			rank:   float64(n) + float64(len(st.Conditions))/10 + float64(len(st.Targets))/100,
			Detail: fmt.Sprintf("%s (project median %s, max %d%s), %s, %s", plural(n, "word"), med, max, ref, plural(len(st.Targets), "term"), plural(len(st.Conditions), "condition"))})
	}
}

// reference returns the reference values of a document's type, if its type
// is a type of a library package.
func (a *Analysis) reference(doc string) (Reference, bool) {
	d, ok := a.Spec.Document(doc)
	if !ok || !d.HasType() {
		return Reference{}, false
	}
	return a.refs.For(d.TypePath)
}

func (a *Analysis) size() {
	count := map[string]int{}
	for _, st := range a.Statements {
		count[st.Doc]++
	}
	byType := map[string][]int{}
	for _, d := range a.Spec.Documents {
		if d.HasType() {
			byType[d.TypePath] = append(byType[d.TypePath], count[d.Path])
		}
	}
	for _, d := range a.Spec.Documents {
		if !d.HasType() {
			continue
		}
		med, low, high := spread(byType[d.TypePath])
		ref := ""
		if r, ok := a.refs.For(d.TypePath); ok {
			ref = fmt.Sprintf("; reference %d–%d", r.Statements[0], r.Statements[1])
		}
		n := count[d.Path]
		a.add(Finding{Signal: Size, Path: d.Path, Line: max(d.TitleLine, 1), Docs: []string{d.Path}, rank: float64(n) / max(median(byType[d.TypePath]), 1),
			Detail: fmt.Sprintf("%s (documents of type %s: median %s, range %d–%d, %s%s)", plural(n, "statement"), d.TypePath, med, low, high, plural(len(byType[d.TypePath]), "document"), ref)})
	}
}

func (a *Analysis) textBeforeSections() {
	first := map[string]int{}
	n := map[string]int{}
	for _, st := range a.Statements {
		d, _ := a.Spec.Document(st.Doc)
		if d == nil || !d.HasType() || st.Section != "" || len(d.Sections) > 0 && st.Line > d.Sections[0].Line {
			continue
		}
		if n[st.Doc] == 0 {
			first[st.Doc] = st.Line
		}
		n[st.Doc]++
	}
	for doc, k := range n {
		a.add(Finding{Signal: TextBeforeSections, Path: doc, Line: first[doc], Docs: []string{doc}, rank: float64(k),
			Detail: plural(k, "statement") + " before the first section"})
	}
}

func (a *Analysis) linkGroups() {
	type place struct {
		doc     string
		line    int
		targets map[string]bool
	}
	bySection := map[string][]place{}
	for _, d := range a.Spec.Documents {
		for _, s := range d.Sections {
			targets := map[string]bool{}
			for _, l := range d.Links {
				if l.Section == s.Heading {
					targets[l.Target] = true
				}
			}
			if len(targets) >= 3 {
				key := strings.ToLower(s.Heading)
				bySection[key] = append(bySection[key], place{d.Path, s.Line, targets})
			}
		}
	}
	for _, places := range bySection {
		type group struct {
			targets []string
			docs    []place
		}
		groups := map[string]*group{}
		for i := range places {
			for j := i + 1; j < len(places); j++ {
				var common []string
				for t := range places[i].targets {
					if places[j].targets[t] {
						common = append(common, t)
					}
				}
				if len(common) < 3 {
					continue
				}
				sort.Strings(common)
				key := strings.Join(common, "\n")
				if groups[key] == nil {
					g := &group{targets: common}
					for _, p := range places {
						all := true
						for _, t := range common {
							all = all && p.targets[t]
						}
						if all {
							g.docs = append(g.docs, p)
						}
					}
					groups[key] = g
				}
			}
		}
		for key, g := range groups {
			maximal := true
			for other, h := range groups {
				if other != key && len(h.docs) == len(g.docs) && len(h.targets) > len(g.targets) && subset(g.targets, h.targets) {
					maximal = false
				}
			}
			if !maximal {
				continue
			}
			var docs []string
			for _, p := range g.docs {
				docs = append(docs, p.doc)
			}
			heading := sectionHeading(a.Spec, g.docs[0].doc, g.docs[0].line)
			a.add(Finding{Signal: LinkGroup, Path: g.docs[0].doc, Line: g.docs[0].line, Docs: append(append([]string(nil), docs...), g.targets...),
				rank:   float64(len(docs)) + float64(len(g.targets))/100,
				Detail: fmt.Sprintf("section %q links the same %d documents in %s: %s (links: %s)", heading, len(g.targets), plural(len(docs), "document"), strings.Join(docs, ", "), strings.Join(g.targets, ", "))})
		}
	}
}

func sectionHeading(spec *lang.Spec, doc string, line int) string {
	d, _ := spec.Document(doc)
	for _, s := range d.Sections {
		if s.Line == line {
			return s.Heading
		}
	}
	return ""
}

func subset(small, big []string) bool {
	for _, s := range small {
		if !contains(big, s) {
			return false
		}
	}
	return true
}

func (a *Analysis) scatteredRules() {
	from := map[string]map[string]bool{}
	for i, st := range a.Statements {
		if len(st.Conditions) == 0 {
			continue
		}
		for _, x := range a.refersTo(i) {
			if x == st.Doc {
				continue
			}
			if _, ok := a.Spec.Document(x); !ok {
				continue
			}
			if from[x] == nil {
				from[x] = map[string]bool{}
			}
			from[x][st.Doc] = true
		}
	}
	for x, docs := range from {
		list := sortedKeys(docs)
		d, _ := a.Spec.Document(x)
		a.add(Finding{Signal: ScatteredRules, Path: x, Line: max(d.TitleLine, 1), Docs: append([]string{x}, list...), rank: float64(len(list)),
			Detail: fmt.Sprintf("conditional statements in %s refer to it: %s", plural(len(list), "other document"), strings.Join(list, ", "))})
	}
}

func (a *Analysis) unlinkedMentions() {
	type pair struct{ doc, target string }
	count := map[pair]int{}
	first := map[pair]int{}
	for i, st := range a.Statements {
		for _, t := range a.Mentions[i] {
			if a.links[st.Doc][t] {
				continue
			}
			p := pair{st.Doc, t}
			if count[p] == 0 {
				first[p] = st.Line
			}
			count[p]++
		}
	}
	for p, n := range count {
		d, _ := a.Spec.Document(p.target)
		a.add(Finding{Signal: UnlinkedMention, Path: p.doc, Line: first[p], Docs: []string{p.doc, p.target}, rank: float64(n),
			Detail: fmt.Sprintf("%q (%s) %s, never linked", d.Title, p.target, plural(n, "time"))})
	}
}

// spread returns the median, the lowest and the highest of counts.
func spread(counts []int) (string, int, int) {
	if len(counts) == 0 {
		return "0", 0, 0
	}
	m := median(counts)
	s := sorted(counts)
	med := fmt.Sprintf("%g", m)
	return med, s[0], s[len(s)-1]
}

func median(counts []int) float64 {
	if len(counts) == 0 {
		return 0
	}
	s := sorted(counts)
	if len(s)%2 == 1 {
		return float64(s[len(s)/2])
	}
	return float64(s[len(s)/2-1]+s[len(s)/2]) / 2
}

func sorted(counts []int) []int {
	s := append([]int(nil), counts...)
	sort.Ints(s)
	return s
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	if strings.HasSuffix(word, "document") || !strings.HasSuffix(word, "s") {
		return fmt.Sprintf("%d %ss", n, word)
	}
	return fmt.Sprintf("%d %s", n, word)
}
