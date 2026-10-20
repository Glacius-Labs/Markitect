package authoring

import (
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// FileDiff is one changed file: added, changed or removed, with its line
// difference when both sides are text.
type FileDiff struct {
	Path   string     `json:"path"`
	Action string     `json:"action"`
	Lines  []DiffLine `json:"lines,omitempty"`
	Binary bool       `json:"binary,omitempty"`
}

// DiffLine is one line of a difference: Op is "=" for context, "-" for a
// removed and "+" for an added line. Old and New are one-based line numbers,
// zero where the line does not exist on that side. A run of unchanged lines
// longer than the context is cut, and Gap counts the lines left out.
type DiffLine struct {
	Op   string `json:"op"`
	Old  int    `json:"old,omitempty"`
	New  int    `json:"new,omitempty"`
	Text string `json:"text"`
	Gap  int    `json:"gap,omitempty"`
}

const (
	diffContext  = 3
	maxDiffLines = 4000
)

func fileDiff(c snapshot.Change) FileDiff {
	d := FileDiff{Path: c.Path().String()}
	before, hasBefore := c.Before()
	after, hasAfter := c.After()
	switch {
	case !hasBefore:
		d.Action = "added"
	case !hasAfter:
		d.Action = "removed"
	default:
		d.Action = "changed"
	}
	var old, new []byte
	if hasBefore {
		old = before.Bytes()
	}
	if hasAfter {
		new = after.Bytes()
	}
	if !utf8.Valid(old) || !utf8.Valid(new) {
		d.Binary = true
		return d
	}
	d.Lines = diffLines(splitLines(string(old)), splitLines(string(new)))
	return d
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// diffLines is a longest-common-subsequence difference with cut context. The
// files it compares are small configuration and model documents.
func diffLines(a, b []string) []DiffLine {
	if len(a)*len(b) > maxDiffLines*maxDiffLines {
		var out []DiffLine
		for i, line := range a {
			out = append(out, DiffLine{Op: "-", Old: i + 1, Text: line})
		}
		for i, line := range b {
			out = append(out, DiffLine{Op: "+", New: i + 1, Text: line})
		}
		return out
	}
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var all []DiffLine
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			all = append(all, DiffLine{Op: "=", Old: i + 1, New: j + 1, Text: a[i]})
			i, j = i+1, j+1
		case i < len(a) && (j == len(b) || lcs[i+1][j] >= lcs[i][j+1]):
			// Removed lines come before the lines that replace them.
			all = append(all, DiffLine{Op: "-", Old: i + 1, Text: a[i]})
			i++
		default:
			all = append(all, DiffLine{Op: "+", New: j + 1, Text: b[j]})
			j++
		}
	}
	return cutContext(all)
}

// cutContext keeps diffContext unchanged lines around every change and
// replaces longer unchanged runs by a gap.
func cutContext(all []DiffLine) []DiffLine {
	keep := make([]bool, len(all))
	for i, l := range all {
		if l.Op == "=" {
			continue
		}
		for k := max(0, i-diffContext); k <= min(len(all)-1, i+diffContext); k++ {
			keep[k] = true
		}
	}
	var out []DiffLine
	skipped := 0
	for i, l := range all {
		if keep[i] {
			if skipped > 0 {
				out = append(out, DiffLine{Op: "=", Gap: skipped})
				skipped = 0
			}
			out = append(out, l)
			continue
		}
		skipped++
	}
	if skipped > 0 && len(out) > 0 {
		out = append(out, DiffLine{Op: "=", Gap: skipped})
	}
	return out
}
