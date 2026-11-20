package lang

import (
	"strings"
	"unicode/utf8"
)

// inline is a code span or an inline link [text](destination "title") found
// on one line; an image is a link with a leading "!".
type inline struct {
	end   int
	code  bool
	image bool
	dest  string
}

// scanInline returns the code spans and inline links of a line, from left to
// right, none overlapping: whatever a code span or a link covers is not looked
// at again.
func scanInline(line string) []inline {
	var out []inline
	for i := 0; i < len(line); {
		var m inline
		ok := false
		switch line[i] {
		case '`':
			m, ok = codeSpan(line, i)
		case '!', '[':
			m, ok = linkAt(line, i, -1)
		}
		if ok {
			out = append(out, m)
			i = m.end
			continue
		}
		i++
	}
	return out
}

// codeSpan matches a code span at i: a run of backticks, at least one
// character, and the same run again. When the whole run is not closed, a
// shorter run from i is tried, down to a single backtick.
func codeSpan(line string, i int) (inline, bool) {
	n := 0
	for i+n < len(line) && line[i+n] == '`' {
		n++
	}
	for k := n; k >= 1; k-- {
		from := i + k + 1 // the content holds at least one character
		if from > len(line) {
			continue
		}
		if j := strings.Index(line[from:], line[i:i+k]); j >= 0 {
			return inline{end: from + j + k, code: true}, true
		}
	}
	return inline{}, false
}

// soleLink returns the destination when s is exactly one inline link, not an
// image.
func soleLink(s string) (string, bool) {
	if !strings.HasPrefix(s, "[") {
		return "", false
	}
	m, ok := linkAt(s, 0, len(s))
	return m.dest, ok
}

// linkAt matches an inline link at i: an optional "!", the text in brackets,
// in which brackets nest one level, and in parentheses the destination, plain
// or in <...>, and an optional title in quotes. With want >= 0 only a match
// that ends at want counts.
func linkAt(line string, i, want int) (inline, bool) {
	j := i
	image := line[j] == '!'
	if image {
		j++
	}
	if j >= len(line) || line[j] != '[' {
		return inline{}, false
	}
	for j++; j < len(line) && line[j] != ']'; j++ {
		if line[j] == '[' {
			k := strings.IndexByte(line[j+1:], ']')
			if k < 0 {
				return inline{}, false
			}
			j += 1 + k
		}
	}
	if j+1 >= len(line) || line[j+1] != '(' {
		return inline{}, false
	}
	open := j + 2
	spaces := blanks(line, open)
	for lead := spaces; lead >= 0; lead-- {
		p := open + lead
		for _, dest := range destinations(line, p) {
			ends := []int{}
			if t := title(line, dest); t >= 0 {
				ends = append(ends, t)
			}
			ends = append(ends, dest)
			for _, e := range ends {
				e += blanks(line, e)
				if e < len(line) && line[e] == ')' && (want < 0 || e+1 == want) {
					return inline{end: e + 1, image: image, dest: line[p:dest]}, true
				}
			}
		}
	}
	return inline{}, false
}

// destinations returns where a destination starting at p may end, in the
// order they are tried: <...> without < or > inside, then the longest run of
// characters that are neither white space nor ")".
func destinations(line string, p int) []int {
	var out []int
	if p < len(line) && line[p] == '<' {
		if k := strings.IndexAny(line[p+1:], "<>"); k >= 0 && line[p+1+k] == '>' {
			out = append(out, p+1+k+1)
		}
	}
	e := p
	for e < len(line) {
		r, size := utf8.DecodeRuneInString(line[e:])
		if r == ')' || isSpace(r) {
			break
		}
		e += size
	}
	return append(out, e)
}

// title returns the end of a link title after p: spaces or tabs, then text in
// double or single quotes. It returns -1 when there is none.
func title(line string, p int) int {
	b := blanks(line, p)
	q := p + b
	if b == 0 || q >= len(line) || (line[q] != '"' && line[q] != '\'') {
		return -1
	}
	k := strings.IndexByte(line[q+1:], line[q])
	if k < 0 {
		return -1
	}
	return q + 1 + k + 1
}

// blanks counts the spaces and tabs at p.
func blanks(line string, p int) int {
	n := 0
	for p+n < len(line) && (line[p+n] == ' ' || line[p+n] == '\t') {
		n++
	}
	return n
}
