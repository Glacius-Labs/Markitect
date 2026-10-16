package model

import (
	"bytes"
	"sort"
	"strconv"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// bodySourceSpans optionally maps decoded body-link byte ranges back to the
// original YAML notation. It is deliberately independent of body parsing and
// content validity: an unproved map returns nil spans without changing model
// status.
func bodySourceSpans(file Entry, body Value, occurrences []bodyOccurrence) []*SourceSpan {
	result := make([]*SourceSpan, len(occurrences))
	if len(occurrences) == 0 || body.kind != StringValue || file.Kind != Regular || !utf8.Valid(file.Bytes) {
		return result
	}
	prepared, problems := prepareYAMLDirectives(file.Bytes)
	if len(problems) != 0 {
		return result
	}
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(prepared)).Decode(&doc); err != nil || len(doc.Content) != 1 {
		return result
	}
	node := findBodyNode(doc.Content[0], body.position, body.text)
	if node == nil || node.Kind != yaml.ScalarNode || node.Value != body.text {
		return result
	}
	atoms, ok := yamlScalarAtoms(file.Bytes, node, body.text, &doc)
	if !ok {
		return result
	}
	firstAtoms, afterAtoms, valid := bodyRangeAtomBounds(body.text, occurrences)
	startPositions, endPositions, mapped := sourceRangePositions(file.Bytes, atoms, firstAtoms, afterAtoms, valid)
	for index := range result {
		if mapped[index] {
			result[index] = &SourceSpan{Start: startPositions[index], End: endPositions[index]}
		}
	}
	return result
}

func findBodyNode(root *yaml.Node, position Position, value string) *yaml.Node {
	if root.Kind != yaml.MappingNode {
		return nil
	}
	var found *yaml.Node
	for index := 0; index+1 < len(root.Content); index += 2 {
		key, candidate := root.Content[index], root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.Value != "body" || candidate.Kind != yaml.ScalarNode || candidate.Line != position.Line || candidate.Column != position.Column || candidate.Value != value {
			continue
		}
		if found != nil {
			return nil
		}
		found = candidate
	}
	return found
}

type sourceAtom struct {
	start, end int
	value      rune
}

func yamlScalarAtoms(source []byte, node *yaml.Node, decoded string, root *yaml.Node) ([]sourceAtom, bool) {
	start, ok := yamlPositionOffset(source, node.Line, node.Column)
	if !ok || start >= len(source) {
		return nil, false
	}
	var atoms []sourceAtom
	switch {
	case node.Style&yaml.SingleQuotedStyle != 0:
		atoms, ok = quotedAtoms(source, start, '\'', false)
	case node.Style&yaml.DoubleQuotedStyle != 0:
		atoms, ok = quotedAtoms(source, start, '"', true)
	case node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0:
		atoms, ok = blockAtoms(source, start, node, root)
	default:
		end, bounded := nextNodeOffset(source, root, node)
		if !bounded {
			end = len(source)
		}
		atoms, ok = plainAtoms(source, start, end)
	}
	if !ok || !sameNonWhitespaceAtoms(atoms, source, decoded) {
		return nil, false
	}
	return atoms, true
}

func quotedAtoms(source []byte, start int, quote byte, escapes bool) ([]sourceAtom, bool) {
	if start >= len(source) || source[start] != quote {
		return nil, false
	}
	var atoms []sourceAtom
	for index := start + 1; index < len(source); {
		origin := index
		if source[index] == quote {
			if !escapes && index+1 < len(source) && source[index+1] == quote {
				atoms = appendIfNonWhitespace(atoms, sourceAtom{start: index, end: index + 2, value: rune(quote)}, rune(quote))
				index += 2
				continue
			}
			return atoms, true
		}
		if escapes && source[index] == '\\' {
			index++
			if index >= len(source) {
				return nil, false
			}
			escape := source[index]
			index++
			if escape == '\r' || escape == '\n' {
				if escape == '\r' && index < len(source) && source[index] == '\n' {
					index++
				}
				for index < len(source) && (source[index] == ' ' || source[index] == '\t') {
					index++
				}
				continue
			}
			if escape == 'x' || escape == 'u' || escape == 'U' {
				digits := map[byte]int{'x': 2, 'u': 4, 'U': 8}[escape]
				if index+digits > len(source) {
					return nil, false
				}
				n, err := strconv.ParseUint(string(source[index:index+digits]), 16, 32)
				if err != nil || !utf8.ValidRune(rune(n)) {
					return nil, false
				}
				index += digits
				atoms = appendIfNonWhitespace(atoms, sourceAtom{start: origin, end: index, value: rune(n)}, rune(n))
				continue
			}
			value, ok := yamlEscape(escape)
			if !ok {
				return nil, false
			}
			atoms = appendIfNonWhitespace(atoms, sourceAtom{start: origin, end: index, value: value}, value)
			continue
		}
		r, size := utf8.DecodeRune(source[index:])
		if r == utf8.RuneError && size == 1 {
			return nil, false
		}
		atoms = appendIfNonWhitespace(atoms, sourceAtom{start: origin, end: index + size, value: r}, r)
		index += size
	}
	return nil, false
}

func yamlEscape(char byte) (rune, bool) {
	switch char {
	case '0':
		return 0, true
	case 'a':
		return '\a', true
	case 'b':
		return '\b', true
	case 't', '\t':
		return '\t', true
	case 'n':
		return '\n', true
	case 'v':
		return '\v', true
	case 'f':
		return '\f', true
	case 'r':
		return '\r', true
	case 'e':
		return '\x1b', true
	case ' ':
		return ' ', true
	case '"':
		return '"', true
	case '/':
		return '/', true
	case '\\':
		return '\\', true
	case 'N':
		return '\u0085', true
	case '_':
		return '\u00a0', true
	case 'L':
		return '\u2028', true
	case 'P':
		return '\u2029', true
	default:
		return 0, false
	}
}

func appendIfNonWhitespace(atoms []sourceAtom, atom sourceAtom, value rune) []sourceAtom {
	if !whitespaceOnly(string(value)) {
		return append(atoms, atom)
	}
	return atoms
}

func plainAtoms(source []byte, start, end int) ([]sourceAtom, bool) {
	if start < 0 || end < start || end > len(source) {
		return nil, false
	}
	var atoms []sourceAtom
	for index := start; index < end; {
		if source[index] == '#' && (index == start || source[index-1] == ' ' || source[index-1] == '\t' || source[index-1] == '\n' || source[index-1] == '\r') {
			for index < end && source[index] != '\n' && source[index] != '\r' {
				index++
			}
			continue
		}
		r, size := utf8.DecodeRune(source[index:end])
		if r == utf8.RuneError && size == 1 {
			return nil, false
		}
		atoms = appendIfNonWhitespace(atoms, sourceAtom{start: index, end: index + size, value: r}, r)
		index += size
	}
	return atoms, true
}

func blockAtoms(source []byte, start int, node, root *yaml.Node) ([]sourceAtom, bool) {
	headerEnd := lineEnd(source, start)
	if headerEnd >= len(source) {
		return nil, false
	}
	header := string(source[start:headerEnd])
	contentStart := nextLine(source, headerEnd)
	end, bounded := nextNodeOffset(source, root, node)
	if !bounded {
		end = len(source)
	}
	baseIndent := indentation(source, lineStart(source, start), start)
	indent := explicitBlockIndent(header, baseIndent)
	if indent == 0 {
		for cursor := contentStart; cursor < end; {
			line := lineEnd(source, cursor)
			trimmed := bytes.Trim(source[cursor:line], " \t\r")
			if len(trimmed) != 0 {
				indent = leadingSpaces(source[cursor:line])
				break
			}
			cursor = nextLine(source, line)
		}
	}
	if indent <= baseIndent {
		return nil, false
	}
	var atoms []sourceAtom
	for cursor := contentStart; cursor < end; {
		line := lineEnd(source, cursor)
		content := source[cursor:line]
		spaces := leadingSpaces(content)
		if len(bytes.Trim(content, " \t\r")) != 0 && spaces < indent {
			break
		}
		if len(bytes.Trim(content, " \t\r")) != 0 {
			cursor += min(indent, spaces)
		}
		lineStop := line
		for index := cursor; index < lineStop; {
			r, size := utf8.DecodeRune(source[index:lineStop])
			if r == utf8.RuneError && size == 1 {
				return nil, false
			}
			atoms = appendIfNonWhitespace(atoms, sourceAtom{start: index, end: index + size, value: r}, r)
			index += size
		}
		cursor = nextLine(source, line)
	}
	return atoms, true
}

func explicitBlockIndent(header string, base int) int {
	for _, char := range header {
		if char == '#' {
			break
		}
		if char >= '1' && char <= '9' {
			return base + int(char-'0')
		}
	}
	return 0
}

func sameNonWhitespaceAtoms(atoms []sourceAtom, source []byte, decoded string) bool {
	index := 0
	for offset := 0; offset < len(decoded); {
		r, size := utf8.DecodeRuneInString(decoded[offset:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if !whitespaceOnly(string(r)) {
			if index >= len(atoms) || atoms[index].start < 0 || atoms[index].end > len(source) || atoms[index].end <= atoms[index].start {
				return false
			}
			if atoms[index].value != r {
				return false
			}
			index++
		}
		offset += size
	}
	return index == len(atoms)
}

type rangeBoundary struct {
	offset, occurrence int
	end                bool
}

// bodyRangeAtomBounds sorts only the requested boundaries, then walks decoded
// body text once to count non-whitespace atoms before each boundary. This keeps
// batch mapping linear in body size after sorting, even with many links.
func bodyRangeAtomBounds(body string, occurrences []bodyOccurrence) (first, after []int, valid []bool) {
	first = make([]int, len(occurrences))
	after = make([]int, len(occurrences))
	valid = make([]bool, len(occurrences))
	if !utf8.ValidString(body) {
		return first, after, valid
	}
	boundaries := make([]rangeBoundary, 0, len(occurrences)*2)
	for index, occurrence := range occurrences {
		span := occurrence.span
		if span.StartByte < 0 || span.EndByte <= span.StartByte || span.EndByte > len(body) || !utf8.RuneStart(body[span.StartByte]) || (span.EndByte < len(body) && !utf8.RuneStart(body[span.EndByte])) {
			continue
		}
		valid[index] = true
		boundaries = append(boundaries,
			rangeBoundary{offset: span.StartByte, occurrence: index},
			rangeBoundary{offset: span.EndByte, occurrence: index, end: true},
		)
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].offset < boundaries[j].offset })
	cursor, atomCount := 0, 0
	for _, boundary := range boundaries {
		for cursor < boundary.offset {
			r, size := utf8.DecodeRuneInString(body[cursor:])
			if size == 0 || cursor+size > boundary.offset {
				valid[boundary.occurrence] = false
				break
			}
			if !whitespaceOnly(string(r)) {
				atomCount++
			}
			cursor += size
		}
		if cursor != boundary.offset {
			valid[boundary.occurrence] = false
			continue
		}
		if boundary.end {
			after[boundary.occurrence] = atomCount
		} else {
			first[boundary.occurrence] = atomCount
		}
	}
	return first, after, valid
}

type sourceBoundary struct {
	offset, occurrence int
	end                bool
}

// sourceRangePositions maps all selected atom endpoints in one ordered pass
// over the original YAML, so many links do not repeatedly scan its prefix.
func sourceRangePositions(source []byte, atoms []sourceAtom, first, after []int, valid []bool) (starts, ends []Position, mapped []bool) {
	starts = make([]Position, len(first))
	ends = make([]Position, len(first))
	mapped = make([]bool, len(first))
	boundaries := make([]sourceBoundary, 0, len(first)*2)
	for index := range first {
		if !valid[index] || first[index] < 0 || after[index] > len(atoms) || after[index] <= first[index] {
			continue
		}
		boundaries = append(boundaries,
			sourceBoundary{offset: atoms[first[index]].start, occurrence: index},
			sourceBoundary{offset: atoms[after[index]-1].end, occurrence: index, end: true},
		)
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].offset < boundaries[j].offset })
	line, column, cursor := 1, 1, 0
	for _, boundary := range boundaries {
		for cursor < boundary.offset {
			switch source[cursor] {
			case '\r':
				cursor++
				if cursor < len(source) && source[cursor] == '\n' {
					cursor++
				}
				line++
				column = 1
			case '\n':
				cursor++
				line++
				column = 1
			default:
				_, size := utf8.DecodeRune(source[cursor:])
				if size == 0 || cursor+size > boundary.offset {
					cursor = boundary.offset + 1
					break
				}
				cursor += size
				column++
			}
			if cursor > boundary.offset {
				break
			}
		}
		if cursor != boundary.offset {
			continue
		}
		position := Position{Line: line, Column: column}
		if boundary.end {
			ends[boundary.occurrence] = position
		} else {
			starts[boundary.occurrence] = position
		}
		mapped[boundary.occurrence] = starts[boundary.occurrence].Line > 0 && ends[boundary.occurrence].Line > 0
	}
	return starts, ends, mapped
}

func yamlBytePosition(source []byte, offset int) (Position, bool) {
	if offset < 0 || offset > len(source) || (offset < len(source) && !utf8.RuneStart(source[offset])) {
		return Position{}, false
	}
	line, column := 1, 1
	for index := 0; index < offset; {
		if source[index] == '\r' {
			line++
			column = 1
			index++
			if index < offset && source[index] == '\n' {
				index++
			}
			continue
		}
		if source[index] == '\n' {
			line++
			column = 1
			index++
			continue
		}
		_, size := utf8.DecodeRune(source[index:])
		index += size
		column++
	}
	return Position{Line: line, Column: column}, true
}

func yamlPositionOffset(source []byte, line, column int) (int, bool) {
	if line < 1 || column < 1 {
		return 0, false
	}
	start := 0
	for current := 1; current < line; current++ {
		for start < len(source) && source[start] != '\n' && source[start] != '\r' {
			start++
		}
		if start == len(source) {
			return 0, false
		}
		if source[start] == '\r' && start+1 < len(source) && source[start+1] == '\n' {
			start += 2
		} else {
			start++
		}
	}
	end := lineEnd(source, start)
	// yaml.v3 reports columns in Unicode code points, while source spans need
	// original byte offsets.
	offset := start
	for current := 1; current < column; current++ {
		if offset >= end {
			return 0, false
		}
		_, size := utf8.DecodeRune(source[offset:end])
		if size == 0 || size == 1 && source[offset] >= utf8.RuneSelf {
			return 0, false
		}
		offset += size
	}
	return offset, offset <= end && (offset == end || utf8.RuneStart(source[offset]))
}

func nextNodeOffset(source []byte, root, target *yaml.Node) (int, bool) {
	line, column := target.Line, target.Column
	bestLine, bestColumn := 0, 0
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		if node != target && (node.Line > line || node.Line == line && node.Column > column) {
			if bestLine == 0 || node.Line < bestLine || node.Line == bestLine && node.Column < bestColumn {
				bestLine, bestColumn = node.Line, node.Column
			}
		}
		for _, child := range node.Content {
			walk(child)
		}
	}
	walk(root)
	if bestLine == 0 {
		return len(source), false
	}
	return yamlPositionOffset(source, bestLine, bestColumn)
}

func lineStart(source []byte, offset int) int {
	for offset > 0 && source[offset-1] != '\n' && source[offset-1] != '\r' {
		offset--
	}
	return offset
}

func lineEnd(source []byte, offset int) int {
	for offset < len(source) && source[offset] != '\n' && source[offset] != '\r' {
		offset++
	}
	return offset
}

func nextLine(source []byte, end int) int {
	if end < len(source) && source[end] == '\r' {
		end++
	}
	if end < len(source) && source[end] == '\n' {
		end++
	}
	return end
}

func indentation(source []byte, start, end int) int {
	if start < 0 || end < start || end > len(source) {
		return 0
	}
	return leadingSpaces(source[start:end])
}

func leadingSpaces(line []byte) int {
	return len(line) - len(bytes.TrimLeft(line, " "))
}
