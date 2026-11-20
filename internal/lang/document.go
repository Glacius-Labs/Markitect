// Package lang is the spec language over supplied files: it parses documents,
// with their title, type line, sections and links, loads a spec from a file
// system and checks it.
//
// It reads only the file system it is handed and writes nothing. Documents are
// read as CommonMark with GitHub's extensions, as GitHub renders them. The
// rules it implements are in docs/guides/language-reference.md.
package lang

import (
	"bytes"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Document is one Markdown file of a spec. Line numbers count from 1; 0 means
// that the line is absent.
type Document struct {
	// Path is the document's path below the spec folder, with forward slashes.
	Path string
	// Title is the text of the level-1 heading that is the document's first
	// block.
	Title     string
	TitleLine int
	// SecondTitleLine is the first further level-1 heading after the title.
	SecondTitleLine int
	// ContentLine is the line of the first block, where the title belongs.
	ContentLine int
	// TypeLine is the line starting with "Type:" in the slot after the title.
	TypeLine int
	// TypeRaw is the path its link names as written, and TypePath that path
	// resolved below the spec folder. Both are empty when the type line is
	// absent or is not exactly one link to a .md path.
	TypeRaw  string
	TypePath string
	// StrayTypeLines are the other lines of text that start with "Type:".
	StrayTypeLines []int
	Sections       []Section
	// Links are the links to .md paths outside type lines.
	Links []Link
	// Refs are all destinations of links to .md paths, type lines, footnotes
	// and link reference definitions included, with their place in Source.
	Refs []Ref
	// Source is the document's bytes as read.
	Source []byte
}

// Section is a level-2 heading at the top level of a document and the blocks
// under it up to the next level-1 or level-2 heading.
type Section struct {
	Heading string
	Line    int
	// HasText is whether a block under the heading shows something other than
	// white space and HTML comments.
	HasText bool
	// First is the first line under the heading that shows something, without
	// surrounding white space, as written.
	First string
}

// Link is a link from a document to a .md path.
type Link struct {
	// Target is the linked path resolved below the spec folder; it may lie
	// outside it, as "../x.md", or be absolute.
	Target string
	// Raw is the path as written, without its #section.
	Raw string
	// Section is the heading of the section the link stands in, or "".
	Section string
	Line    int
}

// HasType reports whether the document names a type with a valid type line.
func (d *Document) HasType() bool { return d.TypePath != "" }

// Section returns the document's first section whose heading equals name
// without regard to case.
func (d *Document) Section(name string) (Section, bool) {
	for _, s := range d.Sections {
		if strings.EqualFold(s.Heading, name) {
			return s, true
		}
	}
	return Section{}, false
}

// Declared is one section a type declares for its documents.
type Declared struct {
	Name     string // the heading without its "(optional)" marker
	Optional bool
}

// Declared returns the sections the document declares when it serves as a
// type: each of its level-2 headings, without the "(optional)" marker. A name
// declared more than once, in any case, is one section, optional only when
// every declaration is.
func (d *Document) Declared() []Declared {
	out := make([]Declared, 0, len(d.Sections))
next:
	for _, s := range d.Sections {
		name, optional := cutOptional(s.Heading)
		for i := range out {
			if strings.EqualFold(out[i].Name, name) {
				out[i].Optional = out[i].Optional && optional
				continue next
			}
		}
		out = append(out, Declared{Name: name, Optional: optional})
	}
	return out
}

const optionalMarker = "(optional)"

func cutOptional(heading string) (string, bool) {
	n := len(heading) - len(optionalMarker)
	if n < 0 || !strings.EqualFold(heading[n:], optionalMarker) {
		return heading, false
	}
	return strings.TrimRightFunc(heading[:n], unicode.IsSpace), true
}

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))

const typePrefix = "Type:"

// Parse reads one document from its bytes as CommonMark with GitHub's
// extensions. A leading byte order mark is dropped, invalid UTF-8 is replaced,
// and lines end with LF, CRLF or CR.
func Parse(docPath string, src []byte) *Document {
	source, orig := normalize(src)
	p := &parser{doc: &Document{Path: docPath, Source: src}, source: source, orig: orig, starts: lineStarts(source)}
	root := markdown.Parser().Parse(text.NewReader(source))
	p.read(root)
	p.refs(root)
	return p.doc
}

type parser struct {
	doc       *Document
	source    []byte // the normalized text
	orig      []int  // the offset in Document.Source of each offset of source
	starts    []int  // the offset of each line
	typeLines map[int]bool
}

// line returns the line of a byte offset.
func (p *parser) line(offset int) int { return sort.SearchInts(p.starts, offset+1) }

func lineStarts(source []byte) []int {
	starts := []int{0}
	for i, c := range source {
		if c == '\n' && i+1 < len(source) {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func (p *parser) read(root ast.Node) {
	d := p.doc
	p.typeLines = map[int]bool{}
	first := nextBlock(root.FirstChild())
	if first != nil {
		d.ContentLine = p.line(first.Pos())
		if h, ok := first.(*ast.Heading); ok && h.Level == 1 {
			if title := p.headingText(h); title != "" {
				d.Title, d.TitleLine = title, d.ContentLine
			}
		}
	}
	if d.TitleLine != 0 {
		if para, ok := nextBlock(first.NextSibling()).(*ast.Paragraph); ok {
			seg := para.Lines().At(0)
			if line := p.lineText(seg); strings.HasPrefix(line, typePrefix) {
				d.TypeLine = p.line(seg.Start)
				p.typeLines[d.TypeLine] = true
				if raw, ok := soleLink(strings.TrimSpace(line[len(typePrefix):])); ok {
					d.TypeRaw, d.TypePath = raw, resolve(d.Path, raw)
				}
			}
		}
	}
	// Stray type lines first, in paragraphs and in the text of tight list items,
	// so that their links are skipped below.
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || (n.Kind() != ast.KindParagraph && n.Kind() != ast.KindTextBlock) {
			return ast.WalkContinue, nil
		}
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			if l := p.line(seg.Start); !p.typeLines[l] && strings.HasPrefix(p.lineText(seg), typePrefix) {
				d.StrayTypeLines = append(d.StrayTypeLines, l)
				p.typeLines[l] = true
			}
		}
		return ast.WalkContinue, nil
	})
	section := ""
	sec := -1
	var bodies [][2]int // the first and last line under each section's heading
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if !isBlock(n) {
			p.links(n, "")
			continue
		}
		if h, ok := n.(*ast.Heading); ok && h.Level <= 2 {
			line := p.line(h.Pos())
			if sec >= 0 && bodies[sec][1] == 0 {
				bodies[sec][1] = line - 1
			}
			if h.Level == 1 {
				if n != first && d.SecondTitleLine == 0 {
					d.SecondTitleLine = line
				}
				section, sec = "", -1
			} else {
				section = p.headingText(h)
				d.Sections = append(d.Sections, Section{Heading: section, Line: line})
				sec = len(d.Sections) - 1
				bodies = append(bodies, [2]int{p.headingEnd(h) + 1, 0})
			}
		} else if sec >= 0 && p.hasText(n) {
			d.Sections[sec].HasText = true
		}
		p.links(n, section)
	}
	for i, b := range bodies {
		if !d.Sections[i].HasText {
			continue
		}
		if b[1] == 0 {
			b[1] = len(p.starts)
		}
		var body []string
		for l := b[0]; l <= b[1]; l++ {
			body = append(body, p.lineAt(l))
		}
		for _, line := range strings.Split(withoutComments(strings.Join(body, "\n")), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				d.Sections[i].First = line
				break
			}
		}
	}
}

// withoutComments drops the HTML comments of s and keeps its line breaks; an
// unclosed comment runs to the end.
func withoutComments(s string) string {
	var b strings.Builder
	for {
		before, rest, ok := strings.Cut(s, "<!--")
		b.WriteString(before)
		if !ok {
			return b.String()
		}
		comment, after, closed := strings.Cut(rest, "-->")
		b.WriteString(strings.Repeat("\n", strings.Count(comment, "\n")))
		if !closed {
			return b.String()
		}
		s = after
	}
}

// lineAt returns the text of a line without its line break.
func (p *parser) lineAt(line int) string {
	if line < 1 || line > len(p.starts) {
		return ""
	}
	end := len(p.source)
	if line < len(p.starts) {
		end = p.starts[line]
	}
	return strings.TrimSuffix(string(p.source[p.starts[line-1]:end]), "\n")
}

// headingEnd returns the last line of a heading: its own for a heading with
// #, the underline for an underlined one.
func (p *parser) headingEnd(h *ast.Heading) int {
	line := p.line(h.Pos())
	if strings.HasPrefix(strings.TrimLeft(p.lineAt(line), " "), "#") || h.Lines().Len() == 0 {
		return line
	}
	return p.line(h.Lines().At(h.Lines().Len()-1).Start) + 1
}

// isBlock reports whether a top-level node is a block of the text: link
// reference definitions are not, nor is the list that gathers the footnote
// definitions at the end.
func isBlock(n ast.Node) bool {
	return n.Kind() != ast.KindLinkReferenceDefinition && n.Kind() != east.KindFootnoteList
}

// nextBlock returns n or the first block after it, or nil.
func nextBlock(n ast.Node) ast.Node {
	for n != nil && !isBlock(n) {
		n = n.NextSibling()
	}
	return n
}

// hasText reports whether a block shows something: text other than white
// space and HTML comments, code, a rule, an image or a link.
func (p *parser) hasText(block ast.Node) bool {
	found := false
	_ = ast.Walk(block, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Text:
			found = visible(string(n.Segment.Value(p.source)))
		case *ast.String:
			found = visible(string(n.Value))
		case *ast.RawHTML:
			found = visible(string(n.Segments.Value(p.source)))
		case *ast.HTMLBlock:
			s := string(n.Lines().Value(p.source))
			if n.HasClosure() {
				s += string(n.ClosureLine.Value(p.source))
			}
			found = visible(s)
		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.ThematicBreak, *ast.CodeSpan, *ast.Image, *ast.Link, *ast.AutoLink:
			found = true
		}
		if found {
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

// visible reports whether s holds a character other than white space outside
// HTML comments; an unclosed comment runs to the end.
func visible(s string) bool {
	for {
		before, rest, ok := strings.Cut(s, "<!--")
		if strings.TrimFunc(before, unicode.IsSpace) != "" {
			return true
		}
		if !ok {
			return false
		}
		if _, after, closed := strings.Cut(rest, "-->"); closed {
			s = after
		} else {
			return false
		}
	}
}

// links collects the links to .md paths in a block, outside the type lines.
func (p *parser) links(block ast.Node, section string) {
	_ = ast.Walk(block, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := n.(*ast.Link)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		line := p.line(link.Pos())
		if raw, ok := localPath(string(link.Destination)); ok && !p.typeLines[line] {
			p.doc.Links = append(p.doc.Links, Link{Target: resolve(p.doc.Path, raw), Raw: raw, Section: section, Line: line})
		}
		return ast.WalkContinue, nil
	})
}

// headingText returns a heading's text as written, its lines joined by a
// space.
func (p *parser) headingText(h *ast.Heading) string {
	var parts []string
	for i := 0; i < h.Lines().Len(); i++ {
		seg := h.Lines().At(i)
		if s := strings.TrimSpace(string(seg.Value(p.source))); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// lineText returns the text of a paragraph line without its line break and
// leading white space.
func (p *parser) lineText(seg text.Segment) string {
	return strings.TrimLeftFunc(strings.TrimRight(string(seg.Value(p.source)), "\n"), unicode.IsSpace)
}

// soleLink returns the path of s when s is exactly one inline link, not an
// image, to a .md path.
func soleLink(s string) (string, bool) {
	root := markdown.Parser().Parse(text.NewReader([]byte(s)))
	para, ok := root.FirstChild().(*ast.Paragraph)
	if !ok || para.NextSibling() != nil {
		return "", false
	}
	link, ok := para.FirstChild().(*ast.Link)
	if !ok || link.NextSibling() != nil {
		return "", false
	}
	raw, ok := localPath(string(link.Destination))
	return raw, ok
}

// localPath returns the path of a link destination that names a .md file;
// ok is false for a web address, mailto:, a bare #section or any other file.
func localPath(dest string) (string, bool) {
	raw, _, _ := strings.Cut(dest, "#")
	if hasScheme(raw) || !strings.HasSuffix(strings.ToLower(decode(raw)), ".md") {
		return "", false
	}
	return raw, true
}

// hasScheme reports whether s starts with a URI scheme and a colon.
func hasScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '.' || c == '-'):
		case i > 0 && c == ':':
			return true
		default:
			return false
		}
	}
	return false
}

// decode turns a destination as written into a path: backslash escapes and
// character references are resolved and %XX sequences decoded; any other % is
// left as it is.
func decode(s string) string {
	b := util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations([]byte(s))))
	if !bytes.Contains(b, []byte("%")) {
		return string(b)
	}
	var out []byte
	for i := 0; i < len(b); i++ {
		if b[i] == '%' && i+2 < len(b) && isHex(b[i+1]) && isHex(b[i+2]) {
			out = append(out, unhex(b[i+1])<<4|unhex(b[i+2]))
			i += 2
			continue
		}
		out = append(out, b[i])
	}
	return strings.ToValidUTF8(string(out), string(utf8.RuneError))
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func unhex(c byte) byte {
	switch {
	case c >= 'a':
		return c - 'a' + 10
	case c >= 'A':
		return c - 'A' + 10
	}
	return c - '0'
}

// resolve resolves a link path, as written in the document at src, to a path
// below the spec folder.
func resolve(src, raw string) string {
	p := decode(raw)
	if strings.HasPrefix(p, "/") {
		return path.Clean(p)
	}
	return path.Join(path.Dir(src), p)
}
