package model

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// bodyHTMLContext tracks only the inline HTML link state relevant to body
// links. The native Goldmark parser remains responsible for recognizing tags.
type bodyHTMLContext struct {
	source    []byte
	linkDepth int
}

type bodyHTMLParser struct {
	parser.InlineParser
	context *bodyHTMLContext
}

func (p *bodyHTMLParser) Parse(parent ast.Node, reader text.Reader, context parser.Context) ast.Node {
	node := p.InlineParser.Parse(parent, reader, context)
	raw, ok := node.(*ast.RawHTML)
	if !ok {
		return node
	}
	if isHTMLAnchorOpen(raw.Segments.Value(p.context.source)) {
		p.context.linkDepth++
	} else if isHTMLAnchorClose(raw.Segments.Value(p.context.source)) {
		p.context.linkDepth--
	}
	return node
}

func (p *bodyHTMLParser) CloseBlock(parent ast.Node, reader text.Reader, context parser.Context) {
	if closer, ok := p.InlineParser.(parser.CloseBlocker); ok {
		closer.CloseBlock(parent, reader, context)
	}
	p.context.linkDepth = 0
}

// These checks run only on RawHTML nodes returned by Goldmark's own parser,
// which has already validated the complete tag. Match the link delimiters
// recognized by the language probe without parsing attributes or HTML markup.
func isHTMLAnchorOpen(raw []byte) bool {
	return len(raw) >= 3 && raw[0] == '<' && isHTMLAnchorLetter(raw[1]) && (raw[2] == '>' || isHTMLSpace(raw[2]))
}

func isHTMLAnchorClose(raw []byte) bool {
	if len(raw) < 4 || raw[0] != '<' || raw[1] != '/' || !isHTMLAnchorLetter(raw[2]) {
		return false
	}
	for _, char := range raw[3:] {
		if char == '>' {
			return true
		}
		if !isHTMLSpace(char) {
			return false
		}
	}
	return false
}

func isHTMLAnchorLetter(char byte) bool {
	return char == 'a' || char == 'A'
}

func isHTMLSpace(char byte) bool {
	switch char {
	case ' ', '\t', '\r', '\n', '\f':
		return true
	default:
		return false
	}
}
