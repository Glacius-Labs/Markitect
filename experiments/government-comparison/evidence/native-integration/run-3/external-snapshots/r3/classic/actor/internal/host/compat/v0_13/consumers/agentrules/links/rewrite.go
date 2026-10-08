// Package links rewrites Markdown link destinations without formatting or
// otherwise changing the surrounding source bytes. It is private to Agent
// Rules provider projections.
//
// Rewrite supports CommonMark-style inline links and images and reference
// definitions/references. Its scanner is intentionally limited to the syntax
// needed to find destinations: it is not a Markdown renderer or validator.
package links

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

// Rewrite calls rewrite once for each inline link/image destination and for the
// first definition of each label that is used by a reference link or image.
// Unused and duplicate definitions are left untouched. Destinations are found
// outside Markdown code and HTML. The
// callback receives a destination with Markdown backslash escapes and HTML
// character references decoded. If it returns that same value, the original
// destination bytes are retained exactly. A changed value must be a
// URI-safe Markdown destination; existing angle brackets, titles, whitespace,
// and all bytes outside destination spans are preserved.
func Rewrite(text string, rewrite func(destination string) (string, error)) (string, error) {
	if rewrite == nil {
		return "", fmt.Errorf("markdownlinks: rewrite callback is nil")
	}
	masked := maskNonMarkdown(text)
	definitions, defLines := findDefinitions(text, masked)
	spans := make([]destinationSpan, 0, 8)
	usedDefinitions := make(map[string]bool)
	for i := 0; i < len(text); i++ {
		if masked[i] != '[' || escaped(text, i) || defLines[lineStart(text, i)] {
			continue
		}
		close := closingBracket(text, masked, i)
		if close < 0 {
			continue
		}
		if close+1 < len(text) && text[close+1] == '(' {
			if span, end, ok := parseInline(text, masked, close+2); ok {
				nested, nestedLabels := nestedImageDestinations(text, masked, i, close, definitions)
				spans = append(spans, nested...)
				for _, label := range nestedLabels {
					usedDefinitions[label] = true
				}
				spans = append(spans, span)
				i = end
				continue
			}
		}
		label := text[i+1 : close]
		refEnd := close
		refLabel := label
		if close+1 < len(text) && text[close+1] == '[' {
			refClose := closingBracket(text, masked, close+1)
			if refClose >= 0 {
				if refClose > close+2 {
					refLabel = text[close+2 : refClose]
				}
				refEnd = refClose
			}
		}
		if _, ok := definitions[normalizeLabel(refLabel)]; ok {
			usedDefinitions[normalizeLabel(refLabel)] = true
			nested, nestedLabels := nestedImageDestinations(text, masked, i, close, definitions)
			spans = append(spans, nested...)
			for _, label := range nestedLabels {
				usedDefinitions[label] = true
			}
			i = refEnd
			continue
		}
	}
	for label := range usedDefinitions {
		spans = append(spans, definitions[label].span)
	}
	if len(spans) == 0 {
		return text, nil
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var out strings.Builder
	out.Grow(len(text))
	last := 0
	for _, span := range spans {
		if span.start < last || span.end < span.start || span.end > len(text) {
			continue
		}
		original := decodeDestination(text[span.start:span.end])
		updated, err := rewrite(original)
		if err != nil {
			return "", err
		}
		out.WriteString(text[last:span.start])
		if updated == original {
			out.WriteString(text[span.start:span.end])
		} else {
			out.WriteString(encodeReplacement(updated, span.angled))
		}
		last = span.end
	}
	out.WriteString(text[last:])
	return out.String(), nil
}

type destinationSpan struct {
	start, end int
	angled     bool
}

type referenceDefinition struct {
	span destinationSpan
}

// findDefinitions keeps the first definition for a normalized label, as
// CommonMark does. Duplicate definitions are deliberately not rewrite targets.
func findDefinitions(text, masked string) (map[string]referenceDefinition, map[int]bool) {
	defs := make(map[string]referenceDefinition)
	lines := make(map[int]bool)
	for start := 0; start < len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		contentEnd := end
		if contentEnd > start && text[contentEnd-1] == '\r' {
			contentEnd--
		}
		container := describeContainer(text[start:contentEnd])
		p := start + container.bodyStart
		for p < contentEnd && p-(start+container.bodyStart) < 4 && text[p] == ' ' {
			p++
		}
		if p < contentEnd && masked[p] == '[' && !escaped(text, p) {
			close := closingBracketBounded(text, masked, p, contentEnd)
			if close > p+1 && close+1 < contentEnd && text[close+1] == ':' {
				label := normalizeLabel(text[p+1 : close])
				q := close + 2
				for q < contentEnd && (text[q] == ' ' || text[q] == '\t') {
					q++
				}
				destinationEnd := contentEnd
				if q == contentEnd {
					var nextStart, nextEnd int
					nextStart, nextEnd = nextDefinitionDestinationLine(text, end, container)
					if nextStart >= 0 {
						q, destinationEnd = nextStart, nextEnd
					}
				}
				if span, after, ok := parseDestination(text, q, destinationEnd); ok && validDefinitionSuffix(text, after, destinationEnd) {
					lines[start] = true
					if _, exists := defs[label]; !exists {
						defs[label] = referenceDefinition{span: span}
					}
				}
			}
		}
		if end == len(text) {
			break
		}
		start = end + 1
	}
	return defs, lines
}

// A definition destination may follow its label on the immediately following
// line. Container markers and list continuation indentation are accounted for
// while the returned span remains in original-source byte coordinates.
func nextDefinitionDestinationLine(text string, currentEnd int, current blockContainer) (int, int) {
	if currentEnd >= len(text) || text[currentEnd] != '\n' {
		return -1, -1
	}
	start := currentEnd + 1
	end := strings.IndexByte(text[start:], '\n')
	if end < 0 {
		end = len(text)
	} else {
		end += start
	}
	contentEnd := end
	if contentEnd > start && text[contentEnd-1] == '\r' {
		contentEnd--
	}
	if strings.TrimSpace(text[start:contentEnd]) == "" {
		return -1, -1
	}
	next := describeContainer(text[start:contentEnd])
	if next.quoteDepth < current.quoteDepth {
		return -1, -1
	}
	if current.hasList {
		if next.hasList && next.listIndent <= current.listIndent {
			return -1, -1
		}
		if !next.hasList && next.contentIndent < current.listContentIndent {
			return -1, -1
		}
	}
	p := start + next.bodyStart
	for p < contentEnd && (text[p] == ' ' || text[p] == '\t') {
		p++
	}
	return p, contentEnd
}

func validDefinitionSuffix(text string, after, end int) bool {
	i := after
	for i < end && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	if i == end {
		return true
	}
	if i == after || (text[i] != '\'' && text[i] != '"' && text[i] != '(') {
		return false
	}
	close := text[i]
	if close == '(' {
		close = ')'
	}
	i++
	for i < end {
		if text[i] == close && !escaped(text, i) {
			i++
			for i < end && (text[i] == ' ' || text[i] == '\t') {
				i++
			}
			return i == end
		}
		i++
	}
	return false
}

func parseInline(text, masked string, start int) (destinationSpan, int, bool) {
	limit := len(text)
	for start < limit && markdownSpace(text[start]) {
		start++
	}
	span, after, ok := parseDestination(text, start, limit)
	if !ok {
		return destinationSpan{}, 0, false
	}
	close, ok := parseLinkSuffix(text, after)
	if !ok {
		return destinationSpan{}, 0, false
	}
	// A newline is valid only while it is part of whitespace around a title.
	// parseDestination already limits bare destinations to a single token.
	_ = masked
	return span, close, true
}

// An image may occur inside a link label. The outer link scan advances over
// that label, so collect image destinations there before skipping to its end.
func nestedImageDestinations(text, masked string, open, close int, definitions map[string]referenceDefinition) ([]destinationSpan, []string) {
	var spans []destinationSpan
	var used []string
	for i := open + 1; i < close; i++ {
		if masked[i] != '[' || i == 0 || text[i-1] != '!' || escaped(text, i) || escaped(text, i-1) {
			continue
		}
		end := closingBracketBounded(text, masked, i, close)
		if end < 0 || end+1 >= close {
			continue
		}
		if text[end+1] == '(' {
			if span, _, ok := parseInline(text, masked, end+2); ok {
				spans = append(spans, span)
				continue
			}
		}
		label := text[i+1 : end]
		refEnd := end
		refLabel := label
		if end+1 < close && text[end+1] == '[' {
			refClose := closingBracketBounded(text, masked, end+1, close)
			if refClose >= 0 {
				if refClose > end+2 {
					refLabel = text[end+2 : refClose]
				}
				refEnd = refClose
			}
		}
		if _, ok := definitions[normalizeLabel(refLabel)]; ok {
			used = append(used, normalizeLabel(refLabel))
			i = refEnd
		}
	}
	return spans, used
}

// parseDestination parses an angle or bare destination, allowing an empty
// destination only when the caller's current byte is the link-closing `)`.
func parseDestination(text string, start, limit int) (destinationSpan, int, bool) {
	if start > limit || start < 0 {
		return destinationSpan{}, 0, false
	}
	if start == limit {
		return destinationSpan{}, 0, false
	}
	if text[start] == '<' {
		for i := start + 1; i < limit; i++ {
			if text[i] == '\n' || text[i] == '\r' || text[i] == '<' && !escaped(text, i) {
				return destinationSpan{}, 0, false
			}
			if text[i] == '>' && !escaped(text, i) {
				return destinationSpan{start: start + 1, end: i, angled: true}, i + 1, true
			}
		}
		return destinationSpan{}, 0, false
	}
	if text[start] == ')' {
		return destinationSpan{start: start, end: start}, start, true
	}
	depth := 0
	for i := start; i < limit; i++ {
		c := text[i]
		if c == '\\' && i+1 < limit {
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if depth == 0 {
				return destinationSpan{start: start, end: i}, i, true
			}
			return destinationSpan{}, 0, false
		}
		if c < 0x20 || c == 0x7f || c == '<' || c == '>' {
			return destinationSpan{}, 0, false
		}
		switch c {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return destinationSpan{start: start, end: i}, i, true
			}
			depth--
		case ' ', '\t', '\n', '\r':
			if depth == 0 {
				return destinationSpan{start: start, end: i}, i, true
			}
		}
	}
	if depth == 0 {
		return destinationSpan{start: start, end: limit}, limit, true
	}
	return destinationSpan{}, 0, false
}

func parseLinkSuffix(text string, at int) (int, bool) {
	i := at
	for i < len(text) && markdownSpace(text[i]) {
		i++
	}
	if i < len(text) && text[i] == ')' {
		return i, true
	}
	if i == at || i >= len(text) || (text[i] != '\'' && text[i] != '"' && text[i] != '(') {
		return 0, false
	}
	close := text[i]
	if close == '(' {
		close = ')'
	}
	i++
	for i < len(text) {
		if text[i] == close && !escaped(text, i) {
			i++
			for i < len(text) && markdownSpace(text[i]) {
				i++
			}
			return i, i < len(text) && text[i] == ')'
		}
		if close == ')' && text[i] == '(' && !escaped(text, i) {
			return 0, false
		}
		i++
	}
	return 0, false
}

func closingBracket(text, masked string, open int) int {
	return closingBracketBounded(text, masked, open, len(text))
}

func closingBracketBounded(text, masked string, open, limit int) int {
	depth := 1
	for i := open + 1; i < limit; i++ {
		if masked[i] == ' ' && text[i] != ' ' || escaped(text, i) {
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

func normalizeLabel(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(decodeDestination(value)), " "))
}

func decodeDestination(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) && isASCIIPunctuation(value[i+1]) {
			i++
		}
		b.WriteByte(value[i])
	}
	return html.UnescapeString(b.String())
}

func encodeReplacement(value string, angled bool) string {
	if angled {
		return strings.NewReplacer("&", "&amp;", "<", "%3C", ">", "%3E", "\r", "%0D", "\n", "%0A").Replace(value)
	}
	return strings.NewReplacer("&", "&amp;", "\\", "%5C", "(", "%28", ")", "%29", " ", "%20", "\t", "%09", "\r", "%0D", "\n", "%0A").Replace(value)
}

func isASCIIPunctuation(c byte) bool {
	return c >= '!' && c <= '/' || c >= ':' && c <= '@' || c >= '[' && c <= '`' || c >= '{' && c <= '~'
}

func escaped(text string, at int) bool {
	n := 0
	for at > 0 && text[at-1] == '\\' {
		n++
		at--
	}
	return n%2 == 1
}

func markdownSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func lineStart(text string, at int) int {
	if at > len(text) {
		at = len(text)
	}
	if i := strings.LastIndexByte(text[:at], '\n'); i >= 0 {
		return i + 1
	}
	return 0
}
