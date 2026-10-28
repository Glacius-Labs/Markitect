package runremote

import (
	"path"
	"strings"
)

// Ownership answers which Package owns a repository file. The registry's
// pattern semantics belong to the compiler (step T1); until its registry
// lands, PatternOwnership stands in with the rule the concept states (§4):
// the most specific pattern wins and Root owns every file no Package claims.
type Ownership interface {
	Owner(file string) string
}

// PatternOwnership matches the Managers' patterns of a plan.
type PatternOwnership struct{ plan *Plan }

// NewPatternOwnership returns the stand-in ownership of the plan's patterns.
func NewPatternOwnership(p *Plan) PatternOwnership { return PatternOwnership{p} }

// Owner returns the Package whose matching pattern is the most specific: the
// one with the most literal characters, then the deeper Package, then the
// smaller ID. Root owns what no pattern matches.
func (o PatternOwnership) Owner(file string) string {
	best, bestScore, bestDepth := RootID, -1, -1
	for _, m := range o.plan.Managers {
		for _, pattern := range m.Patterns {
			if !Match(pattern, file) {
				continue
			}
			score, depth := literal(pattern), strings.Count(m.ID, "/")
			if score > bestScore || score == bestScore && (depth > bestDepth || depth == bestDepth && m.ID < best) {
				best, bestScore, bestDepth = m.ID, score, depth
			}
		}
	}
	return best
}

// Match reports whether a slash-separated pattern matches a repository path:
// "**" as a whole segment matches any number of segments, other segments
// match one path segment as path.Match does.
func Match(pattern, file string) bool {
	return match(strings.Split(pattern, "/"), strings.Split(file, "/"))
}

func match(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for i := 0; i <= len(segs); i++ {
				if match(pat[1:], segs[i:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 {
			return false
		}
		if ok, err := path.Match(pat[0], segs[0]); err != nil || !ok {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}

func literal(pattern string) int {
	n := 0
	for _, r := range pattern {
		if !strings.ContainsRune("*?[]", r) {
			n++
		}
	}
	return n
}
