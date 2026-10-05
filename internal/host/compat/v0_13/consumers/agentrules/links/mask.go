package links

import (
	"strings"
)

// maskNonMarkdown retains offsets and newlines while replacing syntax where
// Markdown link tokens must not be recognized. It covers fenced and indented
// code, inline code spans, HTML comments, tags, and the CommonMark HTML block
// families most likely to contain literal bracket text. This deliberately
// bounded scanner does not implement every container/list parsing rule.
func maskNonMarkdown(text string) string {
	masked := []byte(text)
	fenceChar, fenceSize := byte(0), 0
	var fenceContainer blockContainer
	var activeList blockContainer
	htmlEnd := ""
	for start := 0; start < len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		line := text[start:end]
		container := describeContainer(line)
		if container.listMarker {
			activeList = container
		} else if activeList.hasList && container.quoteDepth >= activeList.quoteDepth && (strings.TrimSpace(line[container.bodyStart:]) == "" || container.contentIndent >= activeList.listContentIndent) {
			container.hasList = true
			container.listIndent = activeList.listIndent
			container.listContentIndent = activeList.listContentIndent
		} else if strings.TrimSpace(line[container.bodyStart:]) != "" {
			activeList = blockContainer{}
		}
		trimmed, indent := containerContent(line)
		if fenceChar != 0 {
			if fenceContains(fenceContainer, container) {
				mask(masked, start, end)
				if closesFence(trimmed, fenceChar, fenceSize) {
					fenceChar, fenceSize = 0, 0
				}
				start = nextLine(end, len(text))
				continue
			}
			fenceChar, fenceSize = 0, 0
		}
		if htmlEnd != "" {
			mask(masked, start, end)
			if strings.Contains(strings.ToLower(line), htmlEnd) {
				htmlEnd = ""
			}
			start = nextLine(end, len(text))
			continue
		}
		if isIndentedCode(trimmed, indent) {
			mask(masked, start, end)
			start = nextLine(end, len(text))
			continue
		}
		if indent <= 3 && len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			n := countRun(trimmed, trimmed[0])
			if n >= 3 && !(trimmed[0] == '`' && strings.Contains(trimmed[n:], "`")) {
				fenceChar, fenceSize = trimmed[0], n
				fenceContainer = container
				mask(masked, start, end)
				start = nextLine(end, len(text))
				continue
			}
		}
		if indent <= 3 {
			lower := strings.ToLower(trimmed)
			if strings.HasPrefix(trimmed, "<!--") {
				mask(masked, start, end)
				if !strings.Contains(trimmed, "-->") {
					htmlEnd = "-->"
				}
				start = nextLine(end, len(text))
				continue
			}
			if strings.HasPrefix(trimmed, "<?") || strings.HasPrefix(trimmed, "<![cdata[") || isDeclaration(trimmed) {
				mask(masked, start, end)
				start = nextLine(end, len(text))
				continue
			}
			if tag := rawHTMLTag(lower); tag != "" {
				mask(masked, start, end)
				endTag := "</" + tag + ">"
				if !strings.Contains(lower, endTag) {
					htmlEnd = endTag
				}
				start = nextLine(end, len(text))
				continue
			}
			if isBlockHTMLTag(lower) {
				mask(masked, start, end)
				start = nextLine(end, len(text))
				for start < len(text) {
					nextEnd := strings.IndexByte(text[start:], '\n')
					if nextEnd < 0 {
						nextEnd = len(text)
					} else {
						nextEnd += start
					}
					if strings.TrimSpace(text[start:nextEnd]) == "" {
						break
					}
					mask(masked, start, nextEnd)
					start = nextLine(nextEnd, len(text))
				}
				continue
			}
		}
		start = nextLine(end, len(text))
	}
	maskInlineCode(text, masked)
	maskInlineHTML(text, masked)
	return string(masked)
}

func maskInlineCode(text string, masked []byte) {
	for i := 0; i < len(text); i++ {
		if masked[i] != '`' || escaped(text, i) {
			continue
		}
		run := countRun(text[i:], '`')
		closeAt := -1
		for j := i + run; j < len(text); {
			if masked[j] != '`' {
				j++
				continue
			}
			other := countRun(text[j:], '`')
			if other == run {
				closeAt = j
				break
			}
			j += other
		}
		if closeAt < 0 {
			i += run - 1
			continue
		}
		mask(masked, i, closeAt+run)
		i = closeAt + run - 1
	}
}

func maskInlineHTML(text string, masked []byte) {
	comment := false
	for i := 0; i < len(text); i++ {
		if masked[i] == ' ' && text[i] != ' ' || masked[i] == '\n' {
			continue
		}
		if comment {
			if strings.HasPrefix(text[i:], "-->") {
				mask(masked, i, i+3)
				i += 2
				comment = false
			} else {
				mask(masked, i, i+1)
			}
			continue
		}
		if strings.HasPrefix(text[i:], "<!--") {
			comment = true
			mask(masked, i, i+4)
			i += 3
			continue
		}
		if text[i] != '<' || i+1 >= len(text) || !(isASCIILetter(text[i+1]) || text[i+1] == '/' || text[i+1] == '!' || text[i+1] == '?') {
			continue
		}
		quote := byte(0)
		for j := i + 1; j < len(text); j++ {
			if quote != 0 {
				if text[j] == quote {
					quote = 0
				}
				continue
			}
			if text[j] == '\'' || text[j] == '"' {
				quote = text[j]
				continue
			}
			if text[j] == '>' {
				mask(masked, i, j+1)
				i = j
				break
			}
		}
	}
}

func trimIndent(line string) (string, int) {
	i := 0
	for i < len(line) && i < 4 && line[i] == ' ' {
		i++
	}
	return line[i:], i
}

// containerContent removes only blockquote/list markers for code-block
// classification. The caller still masks or reads byte offsets from the
// original complete line.
func containerContent(line string) (string, int) {
	content := line[describeContainer(line).bodyStart:]
	trimmed, spaces := trimIndent(content)
	return trimmed, spaces
}

type blockContainer struct {
	bodyStart         int
	quoteDepth        int
	hasList           bool
	listMarker        bool
	listIndent        int
	listContentIndent int
	contentIndent     int
}

// describeContainer records exact source offsets while removing the common
// blockquote/list prefixes used around fenced blocks and reference definitions.
func describeContainer(line string) blockContainer {
	info := blockContainer{bodyStart: 0}
	i := 0
	for depth := 0; depth < 16 && i < len(line); depth++ {
		spaces := 0
		for i+spaces < len(line) && line[i+spaces] == ' ' && spaces < 4 {
			spaces++
		}
		if spaces > 3 {
			break
		}
		p := i + spaces
		if p < len(line) && line[p] == '>' {
			i = p + 1
			if i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			info.quoteDepth++
			info.bodyStart = i
			continue
		}
		if p >= len(line) {
			break
		}
		markerWidth := listMarkerLength(line[p:])
		if markerWidth == 0 {
			break
		}
		q := p + markerWidth
		spaceCount := 0
		for q < len(line) && (line[q] == ' ' || line[q] == '\t') {
			spaceCount++
			q++
		}
		if spaceCount == 0 {
			break
		}
		info.hasList = true
		info.listMarker = true
		info.listIndent = p - info.bodyStart
		if spaceCount > 4 {
			spaceCount = 1
			q = p + markerWidth + 1
		}
		info.listContentIndent = info.listIndent + markerWidth + spaceCount
		i = q
		info.bodyStart = i
	}
	content := line[info.bodyStart:]
	for len(content) > 0 && content[0] == ' ' {
		info.contentIndent++
		content = content[1:]
	}
	return info
}

func fenceContains(fence, line blockContainer) bool {
	if line.quoteDepth < fence.quoteDepth {
		return false
	}
	if !fence.hasList {
		return true
	}
	if line.listMarker && line.listIndent <= fence.listIndent {
		return false
	}
	if !line.hasList && line.contentIndent < fence.listContentIndent {
		return false
	}
	return true
}

func listMarkerLength(line string) int {
	if len(line) >= 2 && (line[0] == '-' || line[0] == '+' || line[0] == '*') && (line[1] == ' ' || line[1] == '\t') {
		return 1
	}
	i := 0
	for i < len(line) && i < 9 && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(line) && (line[i] == '.' || line[i] == ')') && (line[i+1] == ' ' || line[i+1] == '\t') {
		return i + 1
	}
	return 0
}

func isIndentedCode(line string, spaces int) bool {
	return spaces >= 4 || strings.HasPrefix(line, "\t")
}

func closesFence(line string, ch byte, minimum int) bool {
	trimmed, indent := trimIndent(line)
	if indent > 3 || len(trimmed) < minimum || trimmed[0] != ch {
		return false
	}
	n := countRun(trimmed, ch)
	return n >= minimum && strings.TrimSpace(trimmed[n:]) == ""
}

func countRun(text string, c byte) int {
	i := 0
	for i < len(text) && text[i] == c {
		i++
	}
	return i
}

func nextLine(end, length int) int {
	if end < length {
		return end + 1
	}
	return length
}

func mask(bytes []byte, start, end int) {
	for i := start; i < end && i < len(bytes); i++ {
		if bytes[i] != '\n' && bytes[i] != '\r' {
			bytes[i] = ' '
		}
	}
}

func rawHTMLTag(line string) string {
	for _, tag := range []string{"script", "pre", "style", "textarea"} {
		if strings.HasPrefix(line, "<"+tag) && len(line) > len(tag)+1 && (line[len(tag)+1] == '>' || line[len(tag)+1] == ' ' || line[len(tag)+1] == '\t' || line[len(tag)+1] == '/') {
			return tag
		}
	}
	return ""
}

func isBlockHTMLTag(line string) bool {
	if !strings.HasPrefix(line, "<") {
		return false
	}
	end := strings.IndexByte(line, '>')
	if end < 2 {
		return false
	}
	name := strings.TrimLeft(line[1:end], "/")
	for i, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && (c >= '0' && c <= '9' || c == '-')) {
			name = name[:i]
			break
		}
	}
	switch strings.ToLower(name) {
	case "address", "article", "aside", "base", "basefont", "blockquote", "body", "caption", "center", "col", "colgroup", "dd", "details", "dialog", "dir", "div", "dl", "dt", "fieldset", "figcaption", "figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hr", "html", "iframe", "legend", "li", "link", "main", "menu", "menuitem", "meta", "nav", "noframes", "ol", "optgroup", "option", "p", "param", "search", "section", "summary", "table", "tbody", "td", "tfoot", "th", "thead", "title", "tr", "track", "ul":
		return true
	}
	return false
}

func isDeclaration(line string) bool {
	return len(line) >= 3 && strings.HasPrefix(line, "<!") && line[2] >= 'A' && line[2] <= 'Z'
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
