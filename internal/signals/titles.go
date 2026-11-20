package signals

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// titles finds the titles of documents in a statement's words.
type titles struct {
	byFirst map[string][]title
}

type title struct {
	words []string
	docs  []string
}

// newTitles indexes the titles that count as mentions: every document's title
// except those of types, whose names stand in prose as ordinary words, and
// one-word titles shorter than four letters.
func newTitles(spec *lang.Spec) *titles {
	types := map[string]bool{}
	for _, d := range spec.Documents {
		if d.HasType() {
			types[d.TypePath] = true
		}
	}
	byWords := map[string]*title{}
	for _, d := range spec.Documents {
		ws := words(d.Title)
		if types[d.Path] || len(ws) == 0 || len(ws) == 1 && len([]rune(ws[0])) < 4 {
			continue
		}
		key := strings.Join(ws, " ")
		if byWords[key] == nil {
			byWords[key] = &title{words: ws}
		}
		byWords[key].docs = append(byWords[key].docs, d.Path)
	}
	t := &titles{byFirst: map[string][]title{}}
	for _, tt := range byWords {
		t.byFirst[tt.words[0]] = append(t.byFirst[tt.words[0]], *tt)
	}
	return t
}

// find returns the documents whose titles stand in ws, other than self, each
// once. At each word the longest title counts.
func (t *titles) find(ws []string, self string) []string {
	var out []string
	for i := 0; i < len(ws); {
		var best *title
		for k, c := range t.byFirst[ws[i]] {
			if hasPrefixWords(ws[i:], c.words) && (best == nil || len(c.words) > len(best.words)) {
				best = &t.byFirst[ws[i]][k]
			}
		}
		if best == nil {
			i++
			continue
		}
		for _, d := range best.docs {
			if d != self && !contains(out, d) {
				out = append(out, d)
			}
		}
		i += len(best.words)
	}
	return out
}

// Reference is the reference values of one type: statements per document and
// words per statement, each as low and high.
type Reference struct {
	Statements, Words [2]int
}

// References holds reference values by package and type name, as
// "software/operation".
type References map[string]Reference

// For returns the reference values of the type at typePath, a path below the
// spec folder: packages/PACKAGE/NAME.md matches PACKAGE's NAME.
func (r References) For(typePath string) (Reference, bool) {
	parts := strings.Split(typePath, "/")
	if len(parts) != 3 || parts[0] != "packages" {
		return Reference{}, false
	}
	ref, ok := r[parts[1]+"/"+strings.TrimSuffix(parts[2], ".md")]
	return ref, ok
}

// ParseReferences reads the reference values of a package from its file: a
// Markdown table whose rows hold a type name, the statements per document and
// the words per statement, each as "low–high".
func ParseReferences(pkg string, data []byte, into References) error {
	for n, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if len(cells) != 3 || cells[0] == "Type" || strings.HasPrefix(cells[0], "-") {
			continue
		}
		statements, err1 := parseRange(cells[1])
		words, err2 := parseRange(cells[2])
		if err1 != nil || err2 != nil {
			return fmt.Errorf("%s: line %d: a range is low–high", pkg, n+1)
		}
		into[pkg+"/"+cells[0]] = Reference{Statements: statements, Words: words}
	}
	return nil
}

func parseRange(s string) ([2]int, error) {
	s = strings.ReplaceAll(s, "–", "-")
	low, high, ok := strings.Cut(s, "-")
	a, err1 := strconv.Atoi(strings.TrimSpace(low))
	b, err2 := strconv.Atoi(strings.TrimSpace(high))
	if !ok || err1 != nil || err2 != nil || a > b {
		return [2]int{}, fmt.Errorf("bad range %q", s)
	}
	return [2]int{a, b}, nil
}
