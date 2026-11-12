package analysis

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// Where a link that an unlinked mention asks for would go, when not within the
// Package structure as it is.
const (
	// StructureAgainst: the link would close a Package cycle.
	StructureAgainst = "against"
	// StructureNew: the link would add a Package dependency the spec does not have.
	StructureNew = "new_dependency"
)

// unlinkedMentions finds a Definition's name in another Definition's prose with no
// link to it (DEC-102). A name is a Definition's title or the last segment of its
// ID, hyphens read as spaces, matched as whole words without regard to case, with
// up to two trailing letters for inflection. It reads the text and the structure
// only, with no word list of any language. It returns one note per Definition that
// names others, and one more for each group of names a link to which would go
// against the Package structure or beyond it.
func unlinkedMentions(p *project.Project) []Note {
	sp := p.Spec
	ids := sp.OwnDefinitions()
	byFirst := map[string][]named{}
	nameList := map[string][][]string{}
	for _, id := range ids {
		for _, n := range namesOf(sp.Definitions[id]) {
			byFirst[n[0]] = append(byFirst[n[0]], named{id, n})
			nameList[id] = append(nameList[id], n)
		}
	}
	reach := reachability{sp: sp, seen: map[string]map[string]bool{}}
	var out []Note
	for _, x := range ids {
		d := sp.Definitions[x]
		page, err := spec.ParsePage(d.Data)
		if err != nil {
			continue
		}
		text, lines := page.Prose()
		toks := tokenize(text)
		linked := map[string]bool{}
		for _, l := range d.Links {
			if t := sp.Definitions[l.Target]; !l.Package && t != nil && t.Own() {
				linked[l.Target] = true
			}
		}
		// Each target's first mention that no linked Definition's name covers, as a
		// line of the file.
		first := map[string]int{}
		for _, s := range longestFirst(spans(toks, byFirst)) {
			dd := s.id
			t := sp.Definitions[dd]
			// An anchor carries its Package's name, a word of the domain that prose
			// uses freely: naming it is no reliance on the anchor's text. Its name
			// still covers the words, so that no shorter name inside it is reported.
			if dd == x || linked[dd] || t.Kind == spec.CoreKind || sp.Packages[t.Package].Anchor == dd || !mayLink(d, t) {
				continue
			}
			// A part speaks of the thing it is a part of without linking it.
			if sp.IsPartOf(x, dd) {
				continue
			}
			if _, ok := first[dd]; ok || covered(s.words, linked, nameList) {
				continue
			}
			first[dd] = lines[toks[s.from].start]
		}
		groups := map[string][]string{}
		for dd := range first {
			s := reach.structure(d.Package, sp.Definitions[dd].Package)
			groups[s] = append(groups[s], dd)
		}
		for _, structure := range []string{"", StructureAgainst, StructureNew} {
			targets := groups[structure]
			if len(targets) == 0 {
				continue
			}
			sort.Strings(targets)
			line := 0
			for _, dd := range targets {
				if line == 0 || first[dd] < line {
					line = first[dd]
				}
			}
			it := plural(len(targets), "it", "them")
			var msg string
			switch structure {
			case "":
				msg = "names %s without linking %s"
			case StructureAgainst:
				msg = "names %s without linking %s; " + plural(len(targets), "a link would close a Package cycle", "links would close Package cycles")
			case StructureNew:
				msg = "names %s without linking %s; " + plural(len(targets), "a link would add a Package dependency the spec does not have", "links would add Package dependencies the spec does not have")
			}
			var n spec.Diagnostics
			n.Add("vet.unlinked_mention", d.File, line, msg, names(targets), it)
			out = append(out, Note{Diagnostic: n[0], Targets: targets, Structure: structure})
		}
	}
	return out
}

// named is one name of a Definition, as lowercase words.
type named struct {
	id    string
	words []string
}

// namesOf returns a Definition's names: its title and the last segment of its ID,
// as words.
func namesOf(d *spec.Definition) [][]string {
	var out [][]string
	seen := map[string]bool{}
	segment := d.ID[strings.LastIndexByte(d.ID, '/')+1:]
	for _, s := range []string{d.Title, strings.NewReplacer("-", " ", "_", " ").Replace(segment)} {
		w := wordsOf(s)
		if key := strings.Join(w, " "); len(w) > 0 && !seen[key] {
			seen[key] = true
			out = append(out, w)
		}
	}
	return out
}

// wordsOf splits a text into lowercase words: runs of letters and digits.
func wordsOf(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// token is one word of a text: a run of letters, digits and underscores, as a
// regular expression's \w reads a word.
type token struct {
	text       string // lowercase
	start, end int
	// spaced says that only white space separates it from the token before it.
	spaced bool
	// hyphenBefore and hyphenAfter say that a hyphen touches it, which makes it
	// part of a compound word, not a name of its own.
	hyphenBefore, hyphenAfter bool
}

func wordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

func tokenize(text []byte) []token {
	var out []token
	prevEnd := -1
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRune(text[i:])
		if !wordRune(r) {
			i += size
			continue
		}
		start := i
		for i < len(text) {
			r, size = utf8.DecodeRune(text[i:])
			if !wordRune(r) {
				break
			}
			i += size
		}
		t := token{text: strings.ToLower(string(text[start:i])), start: start, end: i}
		t.hyphenBefore = start > 0 && text[start-1] == '-'
		t.hyphenAfter = i < len(text) && text[i] == '-'
		if prevEnd >= 0 {
			gap := text[prevEnd:start]
			t.spaced = len(strings.TrimSpace(string(gap))) == 0 && !strings.ContainsRune(string(gap), 0)
		}
		out = append(out, t)
		prevEnd = i
	}
	return out
}

// span is one place where a name matches: tokens from to to, inclusive.
type span struct {
	id       string
	from, to int
	start    int
	end      int
	words    []string
}

// spans finds every place where a name of a Definition matches the tokens.
func spans(toks []token, byFirst map[string][]named) []span {
	var out []span
	for i, t := range toks {
		if t.hyphenBefore {
			continue
		}
		for cut := 0; cut <= 2 && cut <= len(t.text); cut++ {
			key := t.text[:len(t.text)-cut]
			if cut > 0 && !asciiLetter(t.text[len(t.text)-cut]) {
				break
			}
			for _, n := range byFirst[key] {
				if cut > 0 && len(n.words) > 1 {
					continue // only the last word takes an ending
				}
				if j, ok := matchRest(toks, i, n.words); ok {
					words := make([]string, 0, j-i+1)
					for k := i; k <= j; k++ {
						words = append(words, toks[k].text)
					}
					out = append(out, span{id: n.id, from: i, to: j, start: t.start, end: toks[j].end, words: words})
				}
			}
		}
	}
	return out
}

// matchRest matches the words of a name after the first at token i, and returns
// the last token. Words are separated by white space only; the last may take up
// to two letters of an ending, and no hyphen may follow it.
func matchRest(toks []token, i int, words []string) (int, bool) {
	j := i + len(words) - 1
	if j >= len(toks) {
		return 0, false
	}
	for k := 1; k < len(words); k++ {
		t := toks[i+k]
		if !t.spaced || (k < len(words)-1 && t.text != words[k]) {
			return 0, false
		}
	}
	if len(words) > 1 && !inflected(toks[j].text, words[len(words)-1]) {
		return 0, false
	}
	return j, !toks[j].hyphenAfter
}

// inflected says whether word is base with up to two letters of an ending.
func inflected(word, base string) bool {
	if !strings.HasPrefix(word, base) || len(word)-len(base) > 2 {
		return false
	}
	for i := len(base); i < len(word); i++ {
		if !asciiLetter(word[i]) {
			return false
		}
	}
	return true
}

func asciiLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

// longestFirst keeps, where names overlap, the longest: "maintenance window" is a
// mention of the window, not of maintenance. Names that cover exactly the same
// words are all kept. The result is in the order of the text.
func longestFirst(all []span) []span {
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.end-a.start != b.end-b.start {
			return a.end-a.start > b.end-b.start
		}
		if a.start != b.start {
			return a.start < b.start
		}
		return a.id < b.id
	})
	var kept []span
	for _, s := range all {
		ok := true
		for _, k := range kept {
			if s.start < k.end && k.start < s.end && (s.start != k.start || s.end != k.end) {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, s)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].start < kept[j].start })
	return kept
}

// covered says whether a linked Definition's name already covers the words, as
// "invoices" beside a linked billing/invoice.
func covered(words []string, linked map[string]bool, nameList map[string][][]string) bool {
	for id := range linked {
		for _, n := range nameList[id] {
			if sameName(words, n) {
				return true
			}
		}
	}
	return false
}

// sameName says whether two names are the same words, but for up to two letters
// of an ending on the last.
func sameName(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := 0; i < len(a)-1; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	x, y := a[len(a)-1], b[len(b)-1]
	return inflected(x, y) || inflected(y, x)
}

// mayLink says whether Definition d may link t at all: not into a child Package,
// and not to a Definition internal to a Package d lies outside of.
func mayLink(d, t *spec.Definition) bool {
	if t.Package != d.Package && spec.Within(t.Package, d.Package) {
		return false
	}
	return !t.Internal || spec.Within(d.Package, t.Package)
}

// reachability answers whether a Package depends on another along the explicit
// and implicit edges, as the cycle check reads them.
type reachability struct {
	sp   *spec.Spec
	seen map[string]map[string]bool
}

// structure says where a link from a Definition in Package from to one in Package
// to would go: "" within the structure as it is (the same Package, one above it,
// or a dependency it has), StructureAgainst when it would close a cycle, and
// StructureNew when it would add a dependency.
func (r reachability) structure(from, to string) string {
	switch {
	case spec.Within(from, to), r.sp.Edges[from][to] != nil:
		return ""
	case r.reaches(to)[from]:
		return StructureAgainst
	}
	return StructureNew
}

// reaches returns every Package p depends on, directly or along a path.
func (r reachability) reaches(p string) map[string]bool {
	if got, ok := r.seen[p]; ok {
		return got
	}
	got := map[string]bool{}
	stack := []string{p}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, s := range r.sp.Successors(q) {
			if !got[s] {
				got[s] = true
				stack = append(stack, s)
			}
		}
	}
	r.seen[p] = got
	return got
}
