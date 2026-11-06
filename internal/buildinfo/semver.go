package buildinfo

import (
	"fmt"
	"strconv"
	"strings"
)

// IsDev reports whether version is a development version such as 0.0.0-dev or
// 0.16.0-dev: what the source declares between two releases. A binary built
// from it is not a release and is never replaced by an update on its own.
func IsDev(version string) bool {
	v, err := parseSemver(version)
	return err == nil && v.pre == "dev"
}

// Newer reports whether the release latest is newer than the version current.
// A release is newer than a development version of the same number, and a
// development version is older than any later release.
func Newer(latest, current string) (bool, error) {
	l, err := parseSemver(latest)
	if err != nil {
		return false, err
	}
	c, err := parseSemver(current)
	if err != nil {
		return false, err
	}
	return l.compare(c) > 0, nil
}

// semver is a version of the form MAJOR.MINOR.PATCH[-PRERELEASE]. A leading v
// and build metadata after + are accepted and ignored.
type semver struct {
	major, minor, patch int
	pre                 string
}

func parseSemver(s string) (semver, error) {
	orig := s
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v semver
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
		if v.pre == "" {
			return semver{}, fmt.Errorf("%q is not a version", orig)
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("%q is not a version", orig)
	}
	nums := [3]*int{&v.major, &v.minor, &v.patch}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || (len(p) > 1 && p[0] == '0') {
			return semver{}, fmt.Errorf("%q is not a version", orig)
		}
		*nums[i] = n
	}
	return v, nil
}

func (a semver) compare(b semver) int {
	for _, d := range []int{a.major - b.major, a.minor - b.minor, a.patch - b.patch} {
		if d != 0 {
			return d
		}
	}
	switch {
	case a.pre == b.pre:
		return 0
	case a.pre == "": // a release comes after its pre-releases
		return 1
	case b.pre == "":
		return -1
	}
	return comparePre(a.pre, b.pre)
}

// comparePre orders pre-release parts as semantic versioning does: identifier by
// identifier, numbers by value and before text, text by its letters, and a part
// that is the start of a longer one first. So rc.2 comes before rc.10.
func comparePre(a, b string) int {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(x) && i < len(y); i++ {
		m, mErr := strconv.Atoi(x[i])
		n, nErr := strconv.Atoi(y[i])
		var d int
		switch {
		case mErr == nil && nErr == nil:
			d = m - n
		case mErr == nil:
			d = -1
		case nErr == nil:
			d = 1
		default:
			d = strings.Compare(x[i], y[i])
		}
		if d != 0 {
			return d
		}
	}
	return len(x) - len(y)
}
