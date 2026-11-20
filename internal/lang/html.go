package lang

import (
	"strings"

	"github.com/yuin/goldmark/text"
)

// html collects the relative href and src values of the tags in a piece of
// raw HTML, given as the segments of its lines. Only attributes named href or
// src of a tag count: text between tags, escaped markup such as &lt;img&gt;,
// comments, other attributes and the contents of script, style and textarea
// stay as they are. A tag with an href or src whose value cannot be placed in
// the source, because the tag does not close or the value spans a line break
// inside a container, is recorded in UnplacedHTML.
func (p *parser) html(segs []text.Segment) {
	var b []byte
	var offs []int // the source offset of each byte of b
	for _, s := range segs {
		for i := s.Start; i < s.Stop; i++ {
			b = append(b, p.source[i])
			offs = append(offs, i)
		}
	}
	unplaced := func(i int) {
		line := p.line(offs[min(i, len(offs)-1)])
		for _, l := range p.doc.UnplacedHTML {
			if l == line {
				return
			}
		}
		p.doc.UnplacedHTML = append(p.doc.UnplacedHTML, line)
	}
	for i := 0; i < len(b); {
		if b[i] != '<' {
			i++
			continue
		}
		rest := string(b[i:])
		switch {
		case strings.HasPrefix(rest, "<!--"):
			i = skipPast(b, i+4, "-->")
			continue
		case strings.HasPrefix(rest, "<![CDATA["):
			i = skipPast(b, i+9, "]]>")
			continue
		case strings.HasPrefix(rest, "<!"), strings.HasPrefix(rest, "<?"), strings.HasPrefix(rest, "</"):
			i = skipPast(b, i+2, ">")
			continue
		}
		if i+1 >= len(b) || !isLetter(b[i+1]) {
			i++
			continue
		}
		j := i + 1
		for j < len(b) && (isLetter(b[j]) || b[j] >= '0' && b[j] <= '9' || b[j] == '-') {
			j++
		}
		name := strings.ToLower(string(b[i+1 : j]))
		attrs, end, closed := tagAttributes(b, j)
		for _, a := range attrs {
			if a.name != "href" && a.name != "src" || a.start < 0 {
				continue
			}
			value := string(b[a.start:a.end])
			raw, ok := relativePath(value)
			if !ok {
				continue
			}
			if !closed || a.end > a.start && offs[a.end-1]-offs[a.start] != a.end-1-a.start {
				unplaced(a.start)
				continue
			}
			off := offs[a.start]
			p.doc.Paths = append(p.doc.Paths, Ref{Dest: value, Raw: raw, Target: resolve(p.doc.Path, raw), Line: p.line(off),
				Start: p.orig[off], End: p.orig[off+len(value)]})
		}
		i = end
		if closed && (name == "script" || name == "style" || name == "textarea") {
			i = skipPastFold(b, i, "</"+name)
		}
	}
}

// attribute is one attribute of a tag: its lower-case name and where its
// value stands in the tag's text, start -1 without a value.
type attribute struct {
	name       string
	start, end int
}

// tagAttributes reads the attributes of a tag from i, after its name, up to
// its closing >. It returns where reading stopped and whether the tag closed.
func tagAttributes(b []byte, i int) ([]attribute, int, bool) {
	var attrs []attribute
	for {
		for i < len(b) && isHTMLSpace(b[i]) {
			i++
		}
		if i >= len(b) {
			return attrs, i, false
		}
		switch b[i] {
		case '>':
			return attrs, i + 1, true
		case '/':
			i++
			continue
		}
		start := i
		for i < len(b) && !isHTMLSpace(b[i]) && !strings.ContainsRune("/>=\"'", rune(b[i])) {
			i++
		}
		if i == start { // a stray quote
			i++
			continue
		}
		a := attribute{name: strings.ToLower(string(b[start:i])), start: -1}
		k := i
		for k < len(b) && isHTMLSpace(b[k]) {
			k++
		}
		if k < len(b) && b[k] == '=' {
			k++
			for k < len(b) && isHTMLSpace(b[k]) {
				k++
			}
			switch {
			case k < len(b) && (b[k] == '"' || b[k] == '\''):
				q := b[k]
				e := k + 1
				for e < len(b) && b[e] != q {
					e++
				}
				if e >= len(b) {
					return append(attrs, attribute{name: a.name, start: k + 1, end: e}), e, false
				}
				a.start, a.end, i = k+1, e, e+1
			default:
				e := k
				for e < len(b) && !isHTMLSpace(b[e]) && !strings.ContainsRune("\"'=<>`", rune(b[e])) {
					e++
				}
				a.start, a.end, i = k, e, e
			}
		}
		attrs = append(attrs, a)
	}
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isHTMLSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// skipPast returns the index after the first end at or after i, or len(b).
func skipPast(b []byte, i int, end string) int {
	if k := strings.Index(string(b[i:]), end); k >= 0 {
		return i + k + len(end)
	}
	return len(b)
}

// skipPastFold is skipPast without regard to case; it stops before end.
func skipPastFold(b []byte, i int, end string) int {
	if k := strings.Index(strings.ToLower(string(b[i:])), end); k >= 0 {
		return i + k
	}
	return len(b)
}
