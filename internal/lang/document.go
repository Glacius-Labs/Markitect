// Package lang is the spec language over supplied files: it parses documents,
// with their title, type line, sections and links, loads a spec from a file
// system and checks it.
//
// It reads only the file system it is handed and writes nothing. The rules it
// implements are in docs/guides/language-reference.md.
package lang

import (
	"bytes"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Document is one Markdown file of a spec. Line numbers count from 1; 0 means
// that the line is absent.
type Document struct {
	// Path is the document's path below the spec folder, with forward slashes.
	Path string
	// Title is the text of the level-1 heading on the first line with content.
	Title     string
	TitleLine int
	// SecondTitleLine is the first further level-1 heading after the title.
	SecondTitleLine int
	// ContentLine is the first line with content, where the title belongs.
	ContentLine int
	// TypeLine is the line starting with "Type:" in the slot after the title.
	TypeLine int
	// TypeRaw is the path its link names as written, and TypePath that path
	// resolved below the spec folder. Both are empty when the type line is
	// absent or is not exactly one link to a .md path.
	TypeRaw  string
	TypePath string
	Sections []Section
	Links    []Link
}

// Section is a level-2 heading and the lines under it up to the next level-1
// or level-2 heading.
type Section struct {
	Heading string
	Line    int
	Body    []string
}

// HasText reports whether the section holds a line with content.
func (s Section) HasText() bool {
	for _, l := range s.Body {
		if trim(l) != "" {
			return true
		}
	}
	return false
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
// type: each of its level-2 headings, without the "(optional)" marker.
func (d *Document) Declared() []Declared {
	out := make([]Declared, 0, len(d.Sections))
	for _, s := range d.Sections {
		name, optional := cutOptional(s.Heading)
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
	return strings.TrimRightFunc(heading[:n], isSpace), true
}

// Parse reads one document from its bytes. A leading byte order mark is
// dropped, invalid UTF-8 is replaced, and lines end with LF, CRLF or CR.
func Parse(docPath string, src []byte) *Document {
	text := strings.ToValidUTF8(string(bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))), string(utf8.RuneError))
	doc := &Document{Path: docPath}
	slot := 0        // 0: no content yet, 1: the title slot passed, 2: the type-line slot passed
	sec := -1        // index of the current section
	var fence string // the opening fence while inside a fenced code block
	for i, line := range splitLines(text) {
		n := i + 1
		code := fence != ""
		if code {
			if s := trim(line); s != "" && strings.Trim(s, fence[:1]) == "" && len(s) >= len(fence) {
				fence = ""
			}
		} else if f := openingFence(line); f != "" {
			fence, code = f, true
		}
		level, heading := 0, ""
		if !code {
			level, heading = parseHeading(line)
		}
		if trim(line) != "" {
			switch slot {
			case 0:
				slot = 1
				doc.ContentLine = n
				if level == 1 {
					doc.Title, doc.TitleLine = heading, n
					continue
				}
			case 1:
				slot = 2
				if doc.TitleLine != 0 && !code && strings.HasPrefix(line, "Type:") {
					doc.TypeLine = n
					if dest, ok := soleLink(trim(line[len("Type:"):])); ok {
						if raw, _, ok := localPath(dest); ok {
							doc.TypeRaw, doc.TypePath = raw, resolve(docPath, raw)
						}
					}
					continue
				}
			}
		}
		switch {
		case level == 1:
			if doc.SecondTitleLine == 0 {
				doc.SecondTitleLine = n
			}
			sec = -1
		case level == 2:
			doc.Sections = append(doc.Sections, Section{Heading: heading, Line: n})
			sec = len(doc.Sections) - 1
		case sec >= 0:
			doc.Sections[sec].Body = append(doc.Sections[sec].Body, line)
		}
		if code {
			continue
		}
		for _, m := range scanInline(line) {
			if m.code || m.image {
				continue
			}
			if raw, _, ok := localPath(m.dest); ok {
				name := ""
				if sec >= 0 {
					name = doc.Sections[sec].Heading
				}
				doc.Links = append(doc.Links, Link{Target: resolve(docPath, raw), Raw: raw, Section: name, Line: n})
			}
		}
	}
	return doc
}

// splitLines splits text at LF, CRLF and CR. A final line break ends the last
// line rather than opening an empty one.
func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// leadingSpaces returns how many spaces start s, or -1 when more than three do.
func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	if n > 3 {
		return -1
	}
	return n
}

// openingFence returns the run of three or more backticks or tildes that
// opens a fenced code block on this line, after at most three spaces.
func openingFence(line string) string {
	i := leadingSpaces(line)
	if i < 0 || i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return ""
	}
	j := i
	for j < len(line) && line[j] == line[i] {
		j++
	}
	if j-i < 3 {
		return ""
	}
	return line[i:j]
}

// parseHeading returns the level and text of an ATX heading: at most three
// spaces, one to six #, at least one space or tab, then text that starts with
// a non-space; trailing spaces and tabs are dropped. Level 0 means no heading.
func parseHeading(line string) (int, string) {
	i := leadingSpaces(line)
	if i < 0 {
		return 0, ""
	}
	j := i
	for j < len(line) && line[j] == '#' {
		j++
	}
	level := j - i
	if level < 1 || level > 6 || j >= len(line) || (line[j] != ' ' && line[j] != '\t') {
		return 0, ""
	}
	for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
		j++
	}
	r, _ := utf8.DecodeRuneInString(line[j:])
	if j >= len(line) || isSpace(r) {
		return 0, ""
	}
	return level, strings.TrimRight(line[j:], " \t")
}

// isSpace reports whether r is white space: Unicode white space and the
// separators U+001C to U+001F.
func isSpace(r rune) bool { return unicode.IsSpace(r) || r >= 0x1c && r <= 0x1f }

func trim(s string) string { return strings.TrimFunc(s, isSpace) }

// localPath returns the path and the #section of a link destination that
// names a .md file; ok is false for a web address, mailto:, a bare #section
// or any other file.
func localPath(dest string) (raw, fragment string, ok bool) {
	d := dest
	if strings.HasPrefix(dest, "<") {
		_, size := utf8.DecodeLastRuneInString(dest)
		if len(dest)-size < 1 {
			d = ""
		} else {
			d = dest[1 : len(dest)-size]
		}
	}
	raw, _, _ = strings.Cut(d, "#")
	if hasScheme(raw) || !strings.HasSuffix(strings.ToLower(unescape(raw)), ".md") {
		return "", "", false
	}
	return raw, d[len(raw):], true
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

// unescape decodes %XX sequences and leaves any other % as it is.
func unescape(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			b = append(b, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		b = append(b, s[i])
	}
	return strings.ToValidUTF8(string(b), string(utf8.RuneError))
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
	p := unescape(raw)
	if strings.HasPrefix(p, "/") {
		return path.Clean(p)
	}
	return path.Join(path.Dir(src), p)
}
