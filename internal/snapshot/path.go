package snapshot

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidPath identifies invalid or unsupported path representations.
var ErrInvalidPath = errors.New("invalid snapshot path")

// Path is an exact validated repository-relative path. The zero value is invalid.
type Path struct{ value string }

// NewPath validates an inspectable slash-separated representation without changing
// spelling. Components must be nonempty and cannot be dot segments. Absolute and
// drive-prefixed paths, backslashes (ambiguous separators), invalid UTF-8 and control
// characters are forbidden. Case, Unicode normalization, punctuation and device
// names are preserved; physical representability belongs to adapters. These rules
// do not resolve symlinks or grant authority to read or write a path.
func NewPath(value string) (Path, error) {
	invalid := func() (Path, error) {
		return Path{}, fmt.Errorf("%w: %q", ErrInvalidPath, value)
	}
	if value == "" || !utf8.ValidString(value) {
		return invalid()
	}
	if len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) {
		return invalid()
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == '\\' {
			return invalid()
		}
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return invalid()
		}
	}
	return Path{value: value}, nil
}

// String returns the exact validated representation, or empty for the zero value.
func (p Path) String() string { return p.value }
