package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"unicode"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// IgnoreFile is the file of a project that says what does not belong to its own
// files, by its path below the project folder.
const IgnoreFile = ".markitect/ignore"

// Ignore is the ignore file of a project: the files and folders that diff does
// not look at. It reads like the ignore files of a version control, so that
// someone who has written one can write this. Nothing else is left out but what
// the tool fixes: the folder .markitect/ at the top, the spec folder, links,
// junctions and special files, and a project that lies inside the other. A name
// that starts with a dot is a name like any other. The zero value, a pointer to
// it and nil leave out nothing more.
type Ignore struct {
	rules []ignoreRule
}

type ignoreRule struct {
	// segments are the names of the pattern, each a path.Match pattern or **.
	segments []string
	// negate takes back what an earlier rule leaves out.
	negate bool
	// foldersOnly is set for a pattern that ended with /.
	foldersOnly bool
	// anywhere is set for a pattern without a / at its start or in its middle,
	// which matches a name at any depth; any other is a path from the project
	// folder.
	anywhere bool
}

// ParseIgnore reads an ignore file, one pattern per line:
//
//   - a blank line is skipped, and so is a line that starts with #; white space
//     at the end of a line, a carriage return and a byte order mark at the start
//     are dropped;
//   - a line that starts with ! takes back: what an earlier line leaves out is
//     taken in again when the pattern matches it, and the last line that matches
//     decides; \! and \# at the start of a line stand for a literal ! and #;
//   - a pattern with a / at its start or in its middle is a path from the project
//     folder, a leading / only anchoring it; one without matches a name at any
//     depth; one that ends with / matches folders only;
//   - * is any run of characters within a name, ? one character, [a-z] a class,
//     none of them a /; **/ at the start matches in any folder, /**/ in the
//     middle zero or more folders, /** at the end everything inside.
//
// A folder that is left out is not entered, so that nothing inside it can be
// taken in again: to take in part of a folder, leave out its contents and take
// in the part. A line that is no valid pattern is an error that names the file
// and the line.
func ParseIgnore(data []byte) (*Ignore, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	ig := &Ignore{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r, err := parseIgnoreLine(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: bad pattern \"%s\": %v", IgnoreFile, i+1, line, err)
		}
		ig.rules = append(ig.rules, r)
	}
	return ig, nil
}

func parseIgnoreLine(line string) (ignoreRule, error) {
	var r ignoreRule
	text := line
	switch {
	case strings.HasPrefix(text, `\!`), strings.HasPrefix(text, `\#`):
		text = text[1:]
	case strings.HasPrefix(text, "!"):
		r.negate, text = true, text[1:]
	}
	switch {
	case text == "":
		return r, errors.New("a pattern is not empty")
	case strings.Contains(text, `\`):
		return r, errors.New(`a backslash is only the escape of a ! or # at the start of a pattern`)
	case len(text) >= 2 && text[1] == ':':
		return r, errors.New("a pattern is relative to the project folder and has no drive")
	}
	text, r.foldersOnly = strings.CutSuffix(text, "/")
	body, anchored := strings.CutPrefix(text, "/")
	if body == "" {
		return r, errors.New("a pattern is not empty")
	}
	r.anywhere = !anchored && !strings.Contains(body, "/")
	var err error
	r.segments, err = lang.PatternSegments(body)
	return r, err
}

// ReadIgnore reads the ignore file of the project at fsys; a project without
// one leaves out nothing more.
func ReadIgnore(fsys fs.FS) (*Ignore, error) {
	data, err := fs.ReadFile(fsys, IgnoreFile)
	if errors.Is(err, fs.ErrNotExist) {
		return &Ignore{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", IgnoreFile, unwrapPath(err))
	}
	return ParseIgnore(data)
}

// leaves reports whether the file or folder at rel, a path below the project
// folder, is left out by the ignore file: the last line that matches decides.
func (ig *Ignore) leaves(rel string, folder bool) bool {
	if ig == nil {
		return false
	}
	var names []string
	left := false
	for _, r := range ig.rules {
		if r.foldersOnly && !folder {
			continue
		}
		if names == nil {
			names = strings.Split(rel, "/")
		}
		match := names
		if r.anywhere {
			match = names[len(names)-1:]
		}
		if lang.MatchSegments(r.segments, match) {
			left = !r.negate
		}
	}
	return left
}
