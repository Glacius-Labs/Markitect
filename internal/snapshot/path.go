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

// NewPath validates without changing spelling. Components must be nonempty and
// cannot be dot segments, Windows device names, or end in a dot or space.
// Backslashes, colons, Windows wildcard/reserved punctuation and control characters
// are forbidden. These rules are independent of the host OS; they do not resolve
// symlinks or grant authority to read or write a path.
func NewPath(value string) (Path, error) {
	invalid := func() (Path, error) {
		return Path{}, fmt.Errorf("%w: %q", ErrInvalidPath, value)
	}
	if value == "" || !utf8.ValidString(value) {
		return invalid()
	}
	for _, r := range value {
		if unicode.IsControl(r) || strings.ContainsRune(`\:<>"|?*`, r) {
			return invalid()
		}
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return invalid()
		}
		// Device names remain reserved with an extension on Windows.
		base, _, _ := strings.Cut(part, ".")
		base = strings.ToUpper(base)
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" ||
			(len([]rune(base)) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) &&
				strings.ContainsRune("123456789¹²³", []rune(base)[3])) {
			return invalid()
		}
	}
	return Path{value: value}, nil
}

// String returns the exact validated representation, or empty for the zero value.
func (p Path) String() string { return p.value }

// foldPath gives each Unicode simple-fold cycle a single representative. Unlike
// strings.ToLower it also covers aliases such as Greek sigma and final sigma.
func foldPath(value string) string {
	return strings.Map(func(r rune) rune {
		minimum := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < minimum {
				minimum = next
			}
		}
		return minimum
	}, value)
}
