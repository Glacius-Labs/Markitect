package model

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v3"
)

// Page is one Markdown file split into its YAML front matter and its parsed
// body. Line numbers it reports are lines of the whole file.
type Page struct {
	Front     map[string]*yaml.Node // top-level front matter keys; empty without front matter
	FrontLine int                   // line of the opening ---, 0 without front matter
	Body      []byte
	BodyLine  int // file line of the first body byte
	Root      ast.Node
}

var markdown = goldmark.New(goldmark.WithExtensions(extension.Table))

// ParsePage splits data into front matter and body and parses the body as
// CommonMark with GitHub tables. A front matter block opens with a line
// "---" at the very start of the file and closes with the next line "---".
func ParsePage(data []byte) (Page, error) {
	page := Page{Front: map[string]*yaml.Node{}, Body: data, BodyLine: 1}
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		page.Root = markdown.Parser().Parse(text.NewReader(page.Body))
		return page, nil
	}
	page.FrontLine = 1
	rest := data[bytes.IndexByte(data, '\n')+1:]
	offset := len(data) - len(rest)
	line := 2
	for len(rest) > 0 {
		end := bytes.IndexByte(rest, '\n')
		next := len(rest)
		if end >= 0 {
			next = end + 1
		} else {
			end = len(rest)
		}
		if strings.TrimRight(string(rest[:end]), "\r") == "---" {
			front := data[bytes.IndexByte(data, '\n')+1 : offset]
			if err := page.decodeFront(front); err != nil {
				return page, err
			}
			page.Body = rest[next:]
			page.BodyLine = line + 1
			page.Root = markdown.Parser().Parse(text.NewReader(page.Body))
			return page, nil
		}
		rest = rest[next:]
		offset += next
		line++
	}
	return page, fmt.Errorf("the front matter opened on line 1 is never closed with ---")
}

func (p *Page) decodeFront(front []byte) error {
	if len(bytes.TrimSpace(front)) == 0 {
		return nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return fmt.Errorf("the front matter is not valid YAML: %v", err)
	}
	if len(doc.Content) == 0 {
		return nil
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("the front matter must be a mapping of keys to values")
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		if _, dup := p.Front[key.Value]; dup {
			return fmt.Errorf("the front matter repeats the key %q", key.Value)
		}
		value := mapping.Content[i+1]
		value.Line += p.FrontLine // node lines count from the first front matter line
		p.Front[key.Value] = value
	}
	return nil
}

// Line converts an offset into the body to a line of the whole file.
func (p Page) Line(offset int) int {
	if offset > len(p.Body) {
		offset = len(p.Body)
	}
	return p.BodyLine + bytes.Count(p.Body[:offset], []byte("\n"))
}

// Title returns the text and line of the page's main heading: the first block
// must be a level-1 heading, and no other level-1 heading may follow.
func (p Page) Title() (string, int, error) {
	first := p.Root.FirstChild()
	heading, ok := first.(*ast.Heading)
	if !ok || heading.Level != 1 {
		return "", p.BodyLine, fmt.Errorf("the body must start with a level-1 heading, the title")
	}
	for n := first.NextSibling(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 {
			return "", p.Line(n.Lines().At(0).Start), fmt.Errorf("only one level-1 heading is allowed; the title is the first")
		}
	}
	return strings.TrimSpace(string(heading.Lines().Value(p.Body))), p.Line(heading.Lines().At(0).Start), nil
}

// RawLink is a [[target]] link as written, before resolution.
type RawLink struct {
	Target string // the path part: label after | and section after # removed
	Line   int
}

var wikiLink = regexp.MustCompile(`\[\[([^\[\]\n\x00]+?)\]\]`)

// Links returns every [[...]] link of the body outside code, in order.
func (p Page) Links() []RawLink {
	return p.linksIn(p.Root)
}

// LinksIn returns the links of one node of the page's body.
func (p Page) LinksIn(n ast.Node) []RawLink {
	return p.linksIn(n)
}

func (p Page) linksIn(root ast.Node) []RawLink {
	var links []RawLink
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Type() != ast.TypeBlock {
			return ast.WalkContinue, nil
		}
		if first := n.FirstChild(); first == nil || first.Type() != ast.TypeInline {
			return ast.WalkContinue, nil
		}
		var buf []byte
		var offsets []int
		inlineText(n, p.Body, &buf, &offsets)
		for _, m := range wikiLink.FindAllSubmatchIndex(buf, -1) {
			target := string(buf[m[2]:m[3]])
			if i := strings.IndexByte(target, '|'); i >= 0 {
				target = strings.TrimSuffix(target[:i], `\`)
			}
			if i := strings.IndexByte(target, '#'); i >= 0 {
				target = target[:i]
			}
			links = append(links, RawLink{Target: strings.TrimSpace(target), Line: p.Line(offsets[m[0]])})
		}
		return ast.WalkSkipChildren, nil
	})
	return links
}

// InlineText returns the plain text of a node's inline content; code spans
// keep their text.
func (p Page) InlineText(n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(p.Body))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// inlineText flattens inline content into buf, recording each byte's body
// offset. Code spans, raw HTML and autolinks become a NUL byte so that no
// link is found in or across them.
func inlineText(n ast.Node, src []byte, buf *[]byte, offsets *[]int) {
	last := func() int {
		if len(*offsets) == 0 {
			return 0
		}
		return (*offsets)[len(*offsets)-1]
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			for i := t.Segment.Start; i < t.Segment.Stop; i++ {
				*buf = append(*buf, src[i])
				*offsets = append(*offsets, i)
			}
			if t.SoftLineBreak() || t.HardLineBreak() {
				*buf = append(*buf, '\n')
				*offsets = append(*offsets, t.Segment.Stop)
			}
		case *ast.String:
			for range t.Value {
				*offsets = append(*offsets, last())
			}
			*buf = append(*buf, t.Value...)
		case *ast.CodeSpan, *ast.RawHTML, *ast.AutoLink:
			*buf = append(*buf, 0)
			*offsets = append(*offsets, last())
		default:
			inlineText(c, src, buf, offsets)
		}
	}
}

// Words counts the whitespace-separated words of a body.
func Words(body []byte) int {
	return len(strings.Fields(string(body)))
}
