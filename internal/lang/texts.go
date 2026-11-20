package lang

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// Text is one stretch of prose of a document: a paragraph, the text of a list
// item or a table body row, rendered as plain text. Headings, the type line,
// code, HTML, link reference definitions and footnotes hold none.
type Text struct {
	// Kind is "paragraph", "item" or "row".
	Kind string
	// Section is the heading of the section it stands in, "" before the first.
	Section string
	// Text is the plain text: markup dropped, links and images as their text,
	// line breaks as spaces, the cells of a row joined by " | ".
	Text  string
	Lines []TextLine
	Links []TextLink
}

// TextLine says that the source line Line starts at Offset of Text.
type TextLine struct{ Offset, Line int }

// TextLink is a link in Text, from Start to End, with the path below the spec
// folder it resolves to when it is a link to a .md path, else "".
type TextLink struct {
	Start, End int
	Target     string
}

// LineAt returns the source line of an offset of Text.
func (t Text) LineAt(offset int) int {
	line := 0
	for _, l := range t.Lines {
		if l.Offset > offset {
			break
		}
		line = l.Line
	}
	if line == 0 && len(t.Lines) > 0 {
		line = t.Lines[0].Line
	}
	return line
}

// texts collects the prose of a top-level block.
func (p *parser) texts(block ast.Node, section string) {
	switch n := block.(type) {
	case *ast.Paragraph:
		p.text("paragraph", section, n)
	case *ast.List:
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			t := &Text{Kind: "item", Section: section}
			for c := item.FirstChild(); c != nil; c = c.NextSibling() {
				switch c.Kind() {
				case ast.KindParagraph, ast.KindTextBlock:
					if t.Text != "" {
						t.Text += " "
					}
					p.flow(c, t)
				default:
					p.texts(c, section)
				}
			}
			p.keep(t)
		}
	case *ast.Blockquote:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.texts(c, section)
		}
	case *east.Table:
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			if row.Kind() != east.KindTableRow {
				continue
			}
			t := &Text{Kind: "row", Section: section}
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if cell != row.FirstChild() {
					t.Text += " | "
				}
				p.flow(cell, t)
			}
			p.keep(t)
		}
	}
}

func (p *parser) text(kind, section string, n ast.Node) {
	t := &Text{Kind: kind, Section: section}
	p.flow(n, t)
	p.keep(t)
}

func (p *parser) keep(t *Text) {
	t.Text = strings.TrimRight(t.Text, " ")
	if strings.TrimSpace(t.Text) != "" {
		p.doc.Texts = append(p.doc.Texts, *t)
	}
}

// flow renders the inline content of n into t, leaving out type lines.
func (p *parser) flow(n ast.Node, t *Text) {
	mark := func(offset int) {
		line := p.line(offset)
		if len(t.Lines) == 0 || t.Lines[len(t.Lines)-1].Line != line {
			t.Lines = append(t.Lines, TextLine{Offset: len(t.Text), Line: line})
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			if p.typeLines[p.line(c.Segment.Start)] {
				continue
			}
			mark(c.Segment.Start)
			t.Text += string(c.Segment.Value(p.source))
			if c.SoftLineBreak() || c.HardLineBreak() {
				t.Text += " "
			}
		case *ast.String:
			t.Text += string(c.Value)
		case *ast.RawHTML:
		case *ast.AutoLink:
			mark(c.Pos())
			t.Text += string(c.URL(p.source))
		case *ast.Link:
			if p.typeLines[p.line(c.Pos())] {
				continue
			}
			start := len(t.Text)
			p.flow(c, t)
			target := ""
			if raw, ok := localPath(string(c.Destination)); ok {
				target = resolve(p.doc.Path, raw)
			}
			t.Links = append(t.Links, TextLink{Start: start, End: len(t.Text), Target: target})
		default:
			p.flow(c, t)
		}
	}
}
