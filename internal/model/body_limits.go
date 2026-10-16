package model

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type markdownLimit struct{ measure string }

type boundedBlockParser struct {
	parser.BlockParser
	limit int
}

// Goldmark's default Open methods do not recursively parse child blocks. Check
// the prospective actual node before it is attached or any children are parsed,
// so Goldmark's later recursive walkBlock never receives an over-budget tree.
func (p *boundedBlockParser) Open(parent ast.Node, reader text.Reader, context parser.Context) (ast.Node, parser.State) {
	node, state := p.BlockParser.Open(parent, reader, context)
	if node != nil {
		depth := 1
		for ancestor := parent; ancestor != nil && ancestor.Kind() != ast.KindDocument; ancestor = ancestor.Parent() {
			depth++
		}
		if depth > p.limit {
			panic(markdownLimit{measure: "block AST depth"})
		}
	}
	return node, state
}

type boundedBracketParser struct {
	parser.InlineParser
	limit   int
	parent  ast.Node
	labels  []ast.Node
	onClose func(ast.Node)
}

// Track only opener calls actually processed by Goldmark's unchanged bracket
// parser. Escaped brackets, code and opaque model atoms do not add label depth.
// Before its recursive containsLink, check the pending label's existing tree
// iteratively; never recursively inspect the AST merely to enforce a budget.
func (p *boundedBracketParser) Parse(parent ast.Node, reader text.Reader, context parser.Context) ast.Node {
	if p.parent != parent {
		p.parent, p.labels = parent, nil
	}
	line, _ := reader.PeekLine()
	opening := len(line) > 0 && (line[0] == '[' || bytes.HasPrefix(line, []byte("![")))
	closing := len(line) > 0 && line[0] == ']'
	if opening && len(p.labels) >= p.limit {
		panic(markdownLimit{measure: "pending bracket labels"})
	}
	if closing && len(p.labels) > 0 {
		checkMarkdownTree(p.labels[len(p.labels)-1], p.limit, true)
	}
	node := p.InlineParser.Parse(parent, reader, context)
	if opening && node != nil {
		p.labels = append(p.labels, node)
	} else if closing && len(p.labels) > 0 {
		p.labels = p.labels[:len(p.labels)-1]
	}
	return node
}

func (p *boundedBracketParser) CloseBlock(parent ast.Node, reader text.Reader, context parser.Context) {
	if closer, ok := p.InlineParser.(parser.CloseBlocker); ok {
		closer.CloseBlock(parent, reader, context)
	}
	p.parent, p.labels = nil, nil
	if p.onClose != nil {
		p.onClose(parent)
	}
}

func checkMarkdownTree(root ast.Node, limit int, siblings bool) {
	type frame struct {
		node  ast.Node
		depth int
	}
	stack := []frame{{node: root, depth: 0}}
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current.depth > limit {
			panic(markdownLimit{measure: "constructed AST depth"})
		}
		for child := current.node.FirstChild(); child != nil; child = child.NextSibling() {
			stack = append(stack, frame{node: child, depth: current.depth + 1})
		}
		if siblings && current.depth == 0 && current.node.NextSibling() != nil {
			stack = append(stack, frame{node: current.node.NextSibling(), depth: 0})
		}
	}
}
