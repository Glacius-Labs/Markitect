package model

import (
	"bytes"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type bodyOccurrence struct {
	written string
	display *string
	span    BodySpan
}

type bodyProblem struct {
	code, message string
	start         int
	incomplete    bool
}

type bodySyntax struct {
	occurrences []bodyOccurrence
	problems    []bodyProblem
}

// Only the parser's context view replaces NUL. Boundary offsets map back to
// original decoded UTF-8 bytes, distinguishing a real U+FFFD from authored NUL.
type markdownView struct {
	original, view []byte
	offsets        []int
}

func newMarkdownView(body string) markdownView {
	original := []byte(body)
	m := markdownView{original: original, view: original}
	if !bytes.Contains(original, []byte{0}) {
		return m
	}
	m.view = make([]byte, 0, len(original))
	for index, b := range original {
		if b == 0 {
			m.view = append(m.view, 0xef, 0xbf, 0xbd)
			m.offsets = append(m.offsets, index, index, index)
		} else {
			m.view = append(m.view, b)
			m.offsets = append(m.offsets, index)
		}
	}
	m.offsets = append(m.offsets, len(original))
	return m
}

func (m markdownView) originalOffset(index int) int {
	if m.offsets == nil {
		return index
	}
	return m.offsets[index]
}

type bodyCandidate struct {
	ast.BaseInline
	occurrence bodyOccurrence
}

var bodyCandidateKind = ast.NewNodeKind("MarkitectBodyCandidate")

func (*bodyCandidate) Kind() ast.NodeKind { return bodyCandidateKind }
func (*bodyCandidate) Dump([]byte, int)   {}

type bodyInlineParser struct {
	source        markdownView
	malformed     []bodyProblem
	pending       bool
	result        bodySyntax
	rejectedUntil int
}

func (*bodyInlineParser) Trigger() []byte { return []byte{'['} }

func (p *bodyInlineParser) Parse(_ ast.Node, reader text.Reader, _ parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	if segment.Padding != 0 || !bytes.HasPrefix(line, []byte("[[")) {
		return nil
	}
	p.pending = true
	start := p.source.originalOffset(segment.Start)
	if start < p.rejectedUntil {
		return nil
	}
	stop := p.source.originalOffset(min(segment.Stop, len(p.source.view)))
	raw := p.source.original[start:stop]
	length, target, display, issue := scanBodyLink(raw)
	if issue != "" {
		// Leave malformed text entirely to CommonMark. Consuming even its
		// opener could change whether an enclosing multiline label forms.
		p.malformed = append(p.malformed, bodyProblem{code: "body.syntax", message: issue, start: start})
		// Error recovery does not install model atoms inside the malformed
		// candidate. Track its balanced raw brackets through this decoded
		// line; native CommonMark still receives every byte unchanged.
		p.rejectedUntil = start + malformedBodyExtent(raw)
		return nil
	}
	viewLength := length
	if p.source.offsets != nil {
		viewLength = 0
		for p.source.originalOffset(segment.Start+viewLength) < start+length {
			viewLength++
		}
	}
	reader.Advance(viewLength)
	return &bodyCandidate{occurrence: bodyOccurrence{written: target, display: display, span: BodySpan{StartByte: start, EndByte: start + length}}}
}

// This bounds error recovery only. It neither consumes Markdown input nor
// determines protection; only the native completed AST can do that.
func malformedBodyExtent(raw []byte) int {
	depth := 2
	for index := 2; index < len(raw); {
		r, size := utf8.DecodeRune(raw[index:])
		if r == '\r' || r == '\n' || r == 0x85 || r == 0x2028 || r == 0x2029 {
			return index
		}
		switch r {
		case '\\':
			if index+1 < len(raw) && strings.ContainsRune("|[]\\", rune(raw[index+1])) {
				index += 2
				continue
			}
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return index + size
			}
		}
		index += size
	}
	return len(raw)
}

func scanBodyLink(raw []byte) (int, string, *string, string) {
	var target, display strings.Builder
	hasDisplay := false
	for index := 2; index < len(raw); {
		if bytes.HasPrefix(raw[index:], []byte("]]")) {
			written := target.String()
			if !validAuthorReference(written) {
				return 0, "", nil, "target must be an exact nonempty NFC author reference without surrounding whitespace."
			}
			var label *string
			if hasDisplay {
				value := display.String()
				label = &value
				if whitespaceOnly(value) {
					return 0, "", nil, "display text must not be empty or consist only of Unicode 15.1 White_Space."
				}
			}
			return index + 2, written, label, ""
		}
		r, size := utf8.DecodeRune(raw[index:])
		if r == '\r' || r == '\n' || r == 0x85 || r == 0x2028 || r == 0x2029 {
			return 0, "", nil, "an active model link must be one decoded line."
		}
		switch r {
		case '|':
			if hasDisplay {
				return 0, "", nil, "an additional unescaped separator is forbidden."
			}
			hasDisplay = true
		case '\\':
			if !hasDisplay || index+1 >= len(raw) || !strings.ContainsRune("|[]\\", rune(raw[index+1])) {
				return 0, "", nil, "only \\|, \\[, \\] and \\\\ are permitted, and only in display text."
			} else {
				display.WriteByte(raw[index+1])
				index += 2
				continue
			}
		case '[', ']':
			return 0, "", nil, "unescaped or nested brackets are forbidden inside a model link."
		default:
			if hasDisplay {
				display.WriteRune(r)
			} else {
				target.WriteRune(r)
			}
		}
		index += size
	}
	return 0, "", nil, "active model link has no closing ]]."
}

func parseBodySyntax(body string, limits Limits) (result bodySyntax) {
	if len(body) > limits.BodyBytes {
		return bodySyntax{problems: []bodyProblem{{code: "limit.body_bytes", message: "Decoded body exceeds the original UTF-8 byte budget.", incomplete: true}}}
	}
	source := newMarkdownView(body)
	modelParser := &bodyInlineParser{source: source}
	defer func() {
		if caught := recover(); caught != nil {
			if limit, ok := caught.(markdownLimit); ok {
				// Only completed inline blocks have a final protection context.
				// Keep their proven errors/occurrences beside a later refusal.
				result = modelParser.result
				result.problems = append(result.problems, bodyProblem{code: "limit.markdown_nesting", message: "Markdown nesting budget exceeded (" + limit.measure + ").", incomplete: true})
			} else {
				panic(caught)
			}
		}
		sort.Slice(result.occurrences, func(i, j int) bool {
			return result.occurrences[i].span.StartByte < result.occurrences[j].span.StartByte
		})
	}()
	blocks := parser.DefaultBlockParsers()
	for index, block := range blocks {
		blocks[index].Value = &boundedBlockParser{BlockParser: block.Value.(parser.BlockParser), limit: limits.MarkdownNesting}
	}
	inlines := parser.DefaultInlineParsers()
	for index, inline := range inlines {
		if inline.Priority == 200 {
			inlines[index].Value = &boundedBracketParser{InlineParser: inline.Value.(parser.InlineParser), limit: limits.MarkdownNesting, onClose: modelParser.finishBlock}
		}
	}
	inlines = append(inlines, util.Prioritized(modelParser, 150))
	md := parser.NewParser(parser.WithBlockParsers(blocks...), parser.WithInlineParsers(inlines...), parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...))
	root := md.Parse(text.NewReader(source.view))
	checkMarkdownTree(root, limits.MarkdownNesting, false)
	return modelParser.result
}

// Goldmark has processed delimiters and flushed the native pending labels
// before this callback. No incomplete block is ever judged from a partial AST.
func (p *bodyInlineParser) finishBlock(root ast.Node) {
	if !p.pending {
		return
	}
	source := p.source
	type frame struct {
		node       ast.Node
		protected  bool
		labelStart int
	}
	var protectedText []BodySpan
	stack := []frame{{node: root}}
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		protected := current.protected || current.node.Kind() == ast.KindLink || current.node.Kind() == ast.KindImage
		labelStart := current.labelStart
		if current.node.Kind() == ast.KindLink || current.node.Kind() == ast.KindImage {
			labelStart = source.originalOffset(current.node.Pos())
		}
		if node, ok := current.node.(*ast.Text); ok && protected {
			start := source.originalOffset(node.Segment.Start)
			end := source.originalOffset(node.Segment.Stop)
			protectedText = append(protectedText, BodySpan{StartByte: max(start, labelStart+1), EndByte: end})
		}
		if candidate, ok := current.node.(*bodyCandidate); ok && !protected {
			p.result.occurrences = append(p.result.occurrences, candidate.occurrence)
		}
		for child := current.node.LastChild(); child != nil; child = child.PreviousSibling() {
			stack = append(stack, frame{node: child, protected: protected, labelStart: labelStart})
		}
	}
	// Label text segments are in source order and disjoint. Use their original
	// byte ranges, not a guessed regex extent or the parser's pending-label hint.
	sort.Slice(protectedText, func(i, j int) bool { return protectedText[i].StartByte < protectedText[j].StartByte })
	for _, issue := range p.malformed {
		index := sort.Search(len(protectedText), func(i int) bool { return protectedText[i].EndByte > issue.start })
		if index == len(protectedText) || protectedText[index].StartByte > issue.start {
			p.result.problems = append(p.result.problems, issue)
		}
	}
	p.pending, p.malformed = false, nil
}
