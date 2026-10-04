package markdown

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

type routerLink struct {
	target string
	line   int
}

var referenceDefinition = regexp.MustCompile(`(?m)^ {0,3}\[([^]\n]+)\]:[ \t]*(<[^>\n]*>|[^ \t\n]+)`)
var uriScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// ValidateDocumentationRouters checks only explicitly configured local roots.
// All paths and bytes come from the supplied fixed snapshot, never the host tree.
func ValidateDocumentationRouters(roots []string, files map[string][]byte) []core.Diagnostic {
	if len(roots) == 0 {
		return nil
	}
	allDirectories := map[string]bool{".": len(files) > 0}
	directMarkdown := map[string][]string{}
	for name := range files {
		for dir := path.Dir(name); dir != "."; dir = path.Dir(dir) {
			allDirectories[dir] = true
		}
		if path.Ext(name) == ".md" {
			dir := path.Dir(name)
			directMarkdown[dir] = append(directMarkdown[dir], name)
		}
	}
	var findings []core.Diagnostic
	for _, root := range roots {
		if !allDirectories[root] {
			findings = append(findings, core.Diagnostic{Code: "documentation.root.missing", Path: root, Message: "configured documentation root is absent from the selected snapshot"})
			continue
		}
		participating := map[string]bool{root: true}
		for dir := range directMarkdown {
			if !within(dir, root) {
				continue
			}
			for ancestor := dir; within(ancestor, root); ancestor = path.Dir(ancestor) {
				participating[ancestor] = true
				if ancestor == root {
					break
				}
			}
		}
		children := map[string][]string{}
		for child := range participating {
			if child != root {
				children[path.Dir(child)] = append(children[path.Dir(child)], child)
			}
		}
		for _, dir := range sortedDirectories(participating) {
			router := path.Join(dir, "README.md")
			data, ok := files[router]
			if !ok {
				findings = append(findings, core.Diagnostic{Code: "documentation.router.missing", Path: router, Message: "participating documentation directory needs README.md"})
				continue
			}
			linked := map[string]bool{}
			for _, link := range markdownRouterLinks(data) {
				target, local, err := normalizeRouterTarget(router, link.target)
				if err != nil {
					findings = append(findings, core.Diagnostic{Code: "documentation.link.invalid", Path: router, Line: link.line, Message: fmt.Sprintf("invalid local link %q: %v", link.target, err)})
					continue
				}
				if !local {
					continue
				}
				linked[target] = true
				if _, ok := files[target]; !ok && !allDirectories[target] {
					findings = append(findings, core.Diagnostic{Code: "documentation.link.missing", Path: router, Line: link.line, Message: fmt.Sprintf("local link target %q is absent from the selected snapshot", target)})
				}
			}
			for _, name := range directMarkdown[dir] {
				if name != router && !linked[name] {
					findings = append(findings, core.Diagnostic{Code: "documentation.router.unlisted-file", Path: router, Message: fmt.Sprintf("router does not link to direct Markdown file %q", name)})
				}
			}
			for _, child := range children[dir] {
				if !linked[child] && !linked[path.Join(child, "README.md")] {
					findings = append(findings, core.Diagnostic{Code: "documentation.router.unlisted-directory", Path: router, Message: fmt.Sprintf("router does not link to direct documentation directory %q", child)})
				}
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	return findings
}

func within(name, root string) bool {
	return name == root || strings.HasPrefix(name, strings.TrimSuffix(root, "/")+"/")
}

func sortedDirectories(dirs map[string]bool) []string {
	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

// normalizeRouterTarget returns a repository path only for relative local links.
// Query and fragment delimiters are removed before one percent-decoding pass.
func normalizeRouterTarget(router, destination string) (string, bool, error) {
	if strings.HasPrefix(destination, "//") {
		return "", false, nil
	}
	if uriScheme.MatchString(destination) {
		if len(destination) >= 3 && destination[1] == ':' && (destination[2] == '/' || destination[2] == '\\') {
			return "", true, fmt.Errorf("target is not a portable relative path")
		}
		return "", false, nil
	}
	cut := strings.IndexAny(destination, "?#")
	if cut >= 0 {
		destination = destination[:cut]
	}
	if destination == "" {
		return "", false, nil
	}
	decoded, err := url.PathUnescape(destination)
	if err != nil {
		return "", true, err
	}
	if strings.HasPrefix(decoded, "/") || strings.ContainsAny(decoded, "\\\x00") {
		return "", true, fmt.Errorf("target is not a portable relative path")
	}
	resolved := path.Clean(path.Join(path.Dir(router), decoded))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", true, fmt.Errorf("target escapes the repository")
	}
	return resolved, true, nil
}

// markdownRouterLinks reads inline and reference links outside code spans.
// It deliberately does not interpret prose links as graph relationships.
func markdownRouterLinks(data []byte) []routerLink {
	text := maskMarkdownCode(string(data))
	definitions := map[string]string{}
	definitionLines := map[int]bool{}
	for _, match := range referenceDefinition.FindAllStringSubmatchIndex(text, -1) {
		label := normalizeReferenceLabel(text[match[2]:match[3]])
		if _, exists := definitions[label]; !exists {
			definitions[label] = unescapeMarkdownPath(strings.Trim(text[match[4]:match[5]], "<>"))
		}
		definitionLines[lineNumber(text, match[0])] = true
	}
	var links []routerLink
	for i := 0; i < len(text); i++ {
		if text[i] != '[' || escapedMarkdown(text, i) || definitionLines[lineNumber(text, i)] {
			continue
		}
		end := closingMarkdownBracket(text, i)
		if end < 0 {
			continue
		}
		image := i > 0 && text[i-1] == '!' && !escapedMarkdown(text, i-1)
		line := lineNumber(text, i)
		if end+1 < len(text) && text[end+1] == '(' {
			if target, close, ok := inlineDestination(text, end+2); ok {
				if !image {
					links = append(links, routerLink{target: target, line: line})
				}
				i = close
			}
			continue
		}
		label := text[i+1 : end]
		if end+1 < len(text) && text[end+1] == '[' {
			close := strings.IndexByte(text[end+2:], ']')
			if close < 0 {
				continue
			}
			close += end + 2
			if close > end+2 {
				label = text[end+2 : close]
			}
			i = close
		} else {
			i = end
		}
		if target, ok := definitions[normalizeReferenceLabel(label)]; ok && !image {
			links = append(links, routerLink{target: target, line: line})
		}
	}
	return links
}

func closingMarkdownBracket(text string, open int) int {
	depth := 1
	for i := open + 1; i < len(text); i++ {
		if escapedMarkdown(text, i) {
			continue
		}
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func inlineDestination(text string, start int) (string, int, bool) {
	i := start
	for i < len(text) && markdownWhitespace(text[i]) {
		i++
	}
	if i >= len(text) {
		return "", 0, false
	}
	if text[i] == '<' {
		begin := i + 1
		for i = begin; i < len(text); i++ {
			if text[i] == '\n' || text[i] == '\r' {
				return "", 0, false
			}
			if text[i] == '>' && !escapedMarkdown(text, i) {
				close, ok := inlineLinkEnd(text, i+1)
				return unescapeMarkdownPath(text[begin:i]), close, ok
			}
		}
		return "", 0, false
	}
	begin, nested := i, 0
	for i < len(text) {
		switch text[i] {
		case '\\':
			if i+1 < len(text) {
				i += 2
				continue
			}
		case '(':
			nested++
		case ')':
			if nested == 0 {
				return unescapeMarkdownPath(text[begin:i]), i, true
			}
			nested--
		case ' ', '\t', '\n', '\r':
			if nested == 0 {
				close, ok := inlineLinkEnd(text, i)
				return unescapeMarkdownPath(text[begin:i]), close, ok
			}
		}
		i++
	}
	return "", 0, false
}

func markdownWhitespace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// A destination may be followed only by whitespace, an optional quoted title,
// and the link's closing parenthesis. Arbitrary trailing prose is not a link.
func inlineLinkEnd(text string, at int) (int, bool) {
	i := at
	for i < len(text) && markdownWhitespace(text[i]) {
		i++
	}
	if i >= len(text) {
		return 0, false
	}
	if text[i] == ')' {
		return i, true
	}
	if i == at || (text[i] != '"' && text[i] != '\'' && text[i] != '(') {
		return 0, false
	}
	close := text[i]
	if close == '(' {
		close = ')'
	}
	i++
	for i < len(text) && (text[i] != close || escapedMarkdown(text, i)) {
		if close == ')' && text[i] == '(' && !escapedMarkdown(text, i) {
			return 0, false
		}
		i++
	}
	if i >= len(text) {
		return 0, false
	}
	i++
	for i < len(text) && markdownWhitespace(text[i]) {
		i++
	}
	return i, i < len(text) && text[i] == ')'
}

func unescapeMarkdownPath(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) && strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(value[i+1])) {
			i++
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func escapedMarkdown(text string, index int) bool {
	count := 0
	for index > 0 && text[index-1] == '\\' {
		count++
		index--
	}
	return count%2 != 0
}

func normalizeReferenceLabel(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func lineNumber(text string, at int) int { return 1 + strings.Count(text[:at], "\n") }

func maskMarkdownCode(text string) string {
	masked := []byte(text)
	fence, fenceLen := byte(0), 0
	for start := 0; start < len(masked); {
		end := start
		for end < len(masked) && masked[end] != '\n' {
			end++
		}
		line := strings.TrimLeft(text[start:end], " ")
		indent := end - start - len(line)
		if indent <= 3 && len(line) >= 3 && (line[0] == '`' || line[0] == '~') {
			n := 0
			for n < len(line) && line[n] == line[0] {
				n++
			}
			if n >= 3 {
				if fence == 0 && (line[0] != '`' || !strings.ContainsRune(line[n:], '`')) {
					fence, fenceLen = line[0], n
				} else if line[0] == fence && n >= fenceLen && strings.TrimSpace(line[n:]) == "" {
					fence = 0
				} else if fence == 0 {
					start = end + 1
					continue
				}
				for i := start; i < end; i++ {
					masked[i] = ' '
				}
				start = end + 1
				continue
			}
		}
		if fence != 0 {
			for i := start; i < end; i++ {
				masked[i] = ' '
			}
		}
		start = end + 1
	}
	for i := 0; i < len(masked); i++ {
		if masked[i] != '`' || escapedMarkdown(text, i) {
			continue
		}
		n := 1
		for i+n < len(masked) && masked[i+n] == '`' {
			n++
		}
		close := -1
		for at := i + n; at < len(masked); {
			if masked[at] != '`' {
				at++
				continue
			}
			run := 1
			for at+run < len(masked) && masked[at+run] == '`' {
				run++
			}
			if run == n {
				close = at - i - n
				break
			}
			at += run
		}
		if close < 0 {
			i += n - 1
			continue
		}
		for j := i; j < i+n+close+n; j++ {
			if masked[j] != '\n' {
				masked[j] = ' '
			}
		}
		i += n + close + n - 1
	}
	return string(masked)
}
