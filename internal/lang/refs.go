package lang

import (
	"bytes"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
)

// Ref is one destination of a link to a .md path, wherever it stands: in a
// link, on a type line, in a footnote or in a link reference definition. It
// knows where the destination stands in the document's bytes, so that it can
// be rewritten without touching anything else.
type Ref struct {
	// Dest is the destination as written, with its #section, without < and >.
	Dest string
	// Raw is its path as written, without the #section.
	Raw string
	// Target is the path resolved below the spec folder.
	Target string
	Line   int
	// Start and End delimit Dest in the document's bytes; Start is -1 when
	// the destination could not be located.
	Start, End int
	// Angle is whether the destination is written in < and >.
	Angle bool
}

// Fragment returns the #section of the destination as written, or "".
func (r Ref) Fragment() string { return r.Dest[len(r.Raw):] }

// normalize drops a byte order mark, replaces each run of bytes that are not
// UTF-8 with U+FFFD and turns CRLF and CR into LF. orig maps every offset of
// the result, and its end, to the offset in src.
func normalize(src []byte) (text []byte, orig []int) {
	i := 0
	if bytes.HasPrefix(src, []byte("\xef\xbb\xbf")) {
		i = 3
	}
	text = make([]byte, 0, len(src))
	orig = make([]int, 0, len(src)+1)
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\r':
			text, orig = append(text, '\n'), append(orig, i)
			i++
			if i < len(src) && src[i] == '\n' {
				i++
			}
		case c < utf8.RuneSelf:
			text, orig = append(text, c), append(orig, i)
			i++
		default:
			r, size := utf8.DecodeRune(src[i:])
			if r == utf8.RuneError && size == 1 {
				start := i
				for i < len(src) {
					if r, size := utf8.DecodeRune(src[i:]); r != utf8.RuneError || size != 1 {
						break
					}
					i++
				}
				text = utf8.AppendRune(text, utf8.RuneError)
				for len(orig) < len(text) {
					orig = append(orig, start)
				}
				continue
			}
			text = append(text, src[i:i+size]...)
			for k := 0; k < size; k++ {
				orig = append(orig, i+k)
			}
			i += size
		}
	}
	return text, append(orig, len(src))
}

// refs collects every destination of a link to a .md path in the document.
func (p *parser) refs(root ast.Node) {
	seen := map[int]bool{}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest []byte
		switch n := n.(type) {
		case *ast.Link:
			dest = n.Destination
		case *ast.LinkReferenceDefinition:
			dest = n.Destination
		default:
			return ast.WalkContinue, nil
		}
		raw, ok := localPath(string(dest))
		if !ok {
			return ast.WalkContinue, nil
		}
		ref := Ref{Dest: string(dest), Raw: raw, Target: resolve(p.doc.Path, raw), Line: p.line(n.Pos()), Start: -1, End: -1}
		if off, ok := p.offset(dest); ok {
			if seen[off] {
				return ast.WalkContinue, nil
			}
			seen[off] = true
			ref.Start, ref.End = p.orig[off], p.orig[off+len(dest)]
			ref.Angle = off > 0 && p.source[off-1] == '<'
			ref.Line = p.line(off)
		}
		p.doc.Refs = append(p.doc.Refs, ref)
		return ast.WalkContinue, nil
	})
}

// offset returns where a destination stands in the parsed text, when the
// parser handed out a slice of that text rather than a copy.
func (p *parser) offset(dest []byte) (int, bool) {
	if len(dest) == 0 {
		return 0, false
	}
	off := cap(p.source) - cap(dest)
	if off < 0 || off+len(dest) > len(p.source) || &p.source[off] != &dest[0] {
		return 0, false
	}
	return off, true
}
