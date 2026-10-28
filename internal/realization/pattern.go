package realization

import (
	"fmt"
	"path"
	"strings"
)

// Pattern is one path pattern of a registry or ignore page: a
// repository-relative path whose segments may use * and ?, and whose whole
// segment ** matches any number of folders.
type Pattern struct {
	Text string
	Line int
}

// ValidatePattern reports why text is not a usable pattern.
func ValidatePattern(text string) error {
	if text == "" || strings.HasPrefix(text, "/") || strings.HasSuffix(text, "/") || strings.Contains(text, `\`) {
		return fmt.Errorf("%q: a pattern is a repository-relative path with forward slashes, without a leading or trailing /", text)
	}
	for _, s := range strings.Split(text, "/") {
		switch {
		case s == "" || s == "." || s == "..":
			return fmt.Errorf("%q: empty, . and .. segments are not allowed", text)
		case s != "**" && strings.Contains(s, "**"):
			return fmt.Errorf("%q: ** must be a whole segment", text)
		case strings.ContainsAny(s, "[]{}"):
			return fmt.Errorf("%q: only * , ? and ** are wildcards; [ ] { } are not allowed", text)
		}
		if _, err := path.Match(s, ""); err != nil {
			return fmt.Errorf("%q: %v", text, err)
		}
	}
	return nil
}

// Exact reports whether a pattern names one file without wildcards.
func (p Pattern) Exact() bool { return !strings.ContainsAny(p.Text, "*?") }

// Specificity orders patterns that match the same file: an exact path wins,
// then the pattern with more literal characters (characters other than *, ?
// and /).
func (p Pattern) Specificity() int {
	if p.Exact() {
		return 1 << 30
	}
	n := 0
	for _, r := range p.Text {
		if r != '*' && r != '?' && r != '/' {
			n++
		}
	}
	return n
}

// Match reports whether the pattern matches a repository-relative file path.
func (p Pattern) Match(file string) bool {
	return match(strings.Split(p.Text, "/"), strings.Split(file, "/"))
}

func match(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			if len(pattern) == 1 {
				return len(name) > 0 // a trailing ** matches what lies inside, not the folder itself
			}
			for i := 0; i <= len(name); i++ {
				if match(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, _ := path.Match(pattern[0], name[0]); !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

// ValidFile reports whether a registry entry names a clean
// repository-relative file path.
func ValidFile(file string) bool {
	return file != "" && file == path.Clean(file) && !path.IsAbs(file) && !strings.HasPrefix(file, "../") && file != ".." && file != "." && !strings.Contains(file, `\`)
}
