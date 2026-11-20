// Package signals finds where reading a spec pays off: it splits the prose of
// the documents into statements, finds the marker words in them, computes the
// eight signals of the spec itself and gathers the topic view of a document,
// a type or a folder. It judges nothing and has no thresholds; each signal is
// a list, strongest first.
//
// The rules are in docs/guides/signals.md.
package signals

import (
	"strings"
	"unicode"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// The marker words, in English. A phrase is matched as whole words, without
// regard to case, the longest first, so "only if" is a condition and not the
// strong word "only".
var (
	Conditions = []string{"if", "unless", "except", "only if", "otherwise", "provided that", "as long as", "in case"}
	Vague      = []string{"reasonable", "appropriate", "adequate", "sufficient", "good", "usually", "normally", "generally",
		"typically", "often", "sometimes", "as needed", "as appropriate", "as applicable", "if possible", "etc", "and so on",
		"user-friendly", "easy", "simple", "fast", "flexible", "efficient", "approximately", "about", "several", "some",
		"many", "few", "best", "minimal", "maximal"}
	Strong = []string{"every", "each", "all", "always", "never", "only", "none", "no one", "nobody", "must not"}
)

// Statement is one statement of a document: a sentence of a paragraph, a list
// item or a table body row.
type Statement struct {
	Doc string
	// Section is the heading of its section, "" before the first.
	Section string
	Line    int
	Text    string
	// Words are its words in lower case.
	Words []string
	// Targets are the documents its links point to, each once.
	Targets []string
	// Conditions, Vague and Strong are the markers found, in order.
	Conditions, Vague, Strong []string
}

// Statements splits the prose of a document into statements.
func Statements(d *lang.Document) []Statement {
	var out []Statement
	for _, t := range d.Texts {
		spans := [][2]int{{0, len(t.Text)}}
		if t.Kind == "paragraph" {
			spans = sentences(t.Text)
		}
		for _, s := range spans {
			text := t.Text[s[0]:s[1]]
			lead := len(text) - len(strings.TrimLeft(text, " "))
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			st := Statement{Doc: d.Path, Section: t.Section, Line: t.LineAt(s[0] + lead), Text: text, Words: words(text)}
			for _, l := range t.Links {
				if l.Target != "" && l.Start >= s[0] && l.Start < s[1] && !contains(st.Targets, l.Target) {
					st.Targets = append(st.Targets, l.Target)
				}
			}
			st.Conditions, st.Vague, st.Strong = markers(st.Words)
			out = append(out, st)
		}
	}
	return out
}

// abbreviations end with a full stop that ends no sentence.
var abbreviations = []string{"e.g", "i.e", "etc", "vs"}

// sentences splits prose after ., ! or ?, and any closing quotes or brackets,
// followed by white space or the end, but not after an abbreviation.
func sentences(s string) [][2]int {
	var out [][2]int
	start := 0
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(".!?", rune(s[i])) {
			continue
		}
		j := i + 1
		for j < len(s) && strings.ContainsRune(".!?\"')]", rune(s[j])) {
			j++
		}
		if j < len(s) && s[j] != ' ' && s[j] != '\t' {
			continue
		}
		if s[i] == '.' && abbreviated(s[:i]) {
			continue
		}
		out = append(out, [2]int{start, j})
		start, i = j, j
	}
	if start < len(s) {
		out = append(out, [2]int{start, len(s)})
	}
	return out
}

func abbreviated(before string) bool {
	lower := strings.ToLower(before)
	for _, a := range abbreviations {
		if strings.HasSuffix(lower, a) {
			k := len(lower) - len(a)
			if k == 0 || !isWordRune(rune(lower[k-1])) {
				return true
			}
		}
	}
	return false
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// words returns the words of s in lower case: runs of letters and digits,
// joined inside a word by a hyphen or an apostrophe.
func words(s string) []string {
	var out []string
	var cur []rune
	rs := []rune(strings.ToLower(s))
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}
	for i, r := range rs {
		switch {
		case isWordRune(r):
			cur = append(cur, r)
		case (r == '-' || r == '\'' || r == '’') && len(cur) > 0 && i+1 < len(rs) && isWordRune(rs[i+1]):
			cur = append(cur, r)
		default:
			flush()
		}
	}
	flush()
	return out
}

type marker struct {
	words []string
	kind  int // 0 condition, 1 vague, 2 strong
}

var markerList = func() []marker {
	var out []marker
	for kind, list := range [][]string{Conditions, Vague, Strong} {
		for _, m := range list {
			out = append(out, marker{words: words(m), kind: kind})
		}
	}
	return out
}()

// markers finds the markers in a statement's words, the longest phrase first
// at each word.
func markers(ws []string) (conditions, vague, strong []string) {
	for i := 0; i < len(ws); {
		best := -1
		for k, m := range markerList {
			if hasPrefixWords(ws[i:], m.words) && (best < 0 || len(m.words) > len(markerList[best].words)) {
				best = k
			}
		}
		if best < 0 {
			i++
			continue
		}
		m := markerList[best]
		phrase := strings.Join(m.words, " ")
		switch m.kind {
		case 0:
			conditions = append(conditions, phrase)
		case 1:
			vague = append(vague, phrase)
		default:
			strong = append(strong, phrase)
		}
		i += len(m.words)
	}
	return conditions, vague, strong
}

func hasPrefixWords(ws, prefix []string) bool {
	if len(prefix) > len(ws) {
		return false
	}
	for i := range prefix {
		if ws[i] != prefix[i] {
			return false
		}
	}
	return true
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
