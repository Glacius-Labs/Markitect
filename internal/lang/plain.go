package lang

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// firstLine returns the first line of a block's text that shows something,
// as plain text: markup and list markers dropped, links, reference links and
// images shown as their text, HTML comments and tags left out. Code shows its
// first line, a table its first row with cells separated by " | ".
func (p *parser) firstLine(block ast.Node) string {
	for _, line := range strings.Split(p.plain(block), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// plain renders a block as plain text, one line per line of text.
func (p *parser) plain(n ast.Node) string {
	switch n := n.(type) {
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		return string(n.Lines().Value(p.source))
	case *ast.HTMLBlock:
		s := string(n.Lines().Value(p.source))
		if n.HasClosure() {
			s += string(n.ClosureLine.Value(p.source))
		}
		return withoutTags(withoutComments(s))
	case *ast.ThematicBreak, *ast.LinkReferenceDefinition:
		return ""
	case *east.TableRow, *east.TableHeader:
		var cells []string
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			cells = append(cells, strings.TrimSpace(p.inline(c)))
		}
		return strings.Join(cells, " | ") + "\n"
	}
	if n.Type() == ast.TypeBlock && n.FirstChild() != nil && n.FirstChild().Type() == ast.TypeInline {
		return p.inline(n) + "\n"
	}
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b.WriteString(p.plain(c))
	}
	return b.String()
}

// inline renders the inline children of a node, with a line break where the
// text breaks.
func (p *parser) inline(n ast.Node) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			b.Write(c.Segment.Value(p.source))
			if c.SoftLineBreak() || c.HardLineBreak() {
				b.WriteByte('\n')
			}
		case *ast.String:
			b.Write(c.Value)
		case *ast.RawHTML:
			// tags and comments show nothing
		case *ast.AutoLink:
			b.Write(c.URL(p.source))
		default:
			b.WriteString(p.inline(c))
		}
	}
	return b.String()
}

// withoutTags drops what stands in < and > on each line.
func withoutTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>' && in:
			in = false
		case !in || r == '\n':
			b.WriteRune(r)
		}
	}
	return b.String()
}
