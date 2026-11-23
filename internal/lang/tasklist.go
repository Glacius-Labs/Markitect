package lang

import (
	"regexp"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// taskList is the task list of GitHub's Markdown: a list item that starts with
// [ ] or [x] and white space. The extension of goldmark takes white space after
// the bracket for optional, so that the list item "- [x](x.md)" would lose its
// link to a checked box; GitHub renders that item as a link.
type taskList struct{}

// taskMarker is the marker at the start of a list item: a space or an x in
// brackets, and then white space or the end of the text.
var taskMarker = regexp.MustCompile(`^\[([\sxX])\](?:\s+|$)`)

func (taskList) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(gparser.WithInlineParsers(util.Prioritized(taskCheckBoxParser{}, 0)))
}

type taskCheckBoxParser struct{}

func (taskCheckBoxParser) Trigger() []byte { return []byte{'['} }

func (taskCheckBoxParser) Parse(parent gast.Node, block text.Reader, pc gparser.Context) gast.Node {
	// The list item holds a text block, the parent, and the marker is the
	// start of its first line.
	if parent.Parent() == nil || parent.Parent().FirstChild() != parent || parent.HasChildren() {
		return nil
	}
	if _, ok := parent.Parent().(*gast.ListItem); !ok {
		return nil
	}
	line, _ := block.PeekLine()
	m := taskMarker.FindSubmatchIndex(line)
	if m == nil {
		return nil
	}
	value := line[m[2]]
	block.Advance(m[1])
	return east.NewTaskCheckBox(value == 'x' || value == 'X')
}

func (taskCheckBoxParser) CloseBlock(parent gast.Node, pc gparser.Context) {}
