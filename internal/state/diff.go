package state

import (
	"bytes"
	"fmt"
	"strings"
)

// diffContext is how many unchanged lines a hunk shows around a change.
const diffContext = 3

// maxDiffCells bounds the table of the line diff; a larger change shows as
// one hunk that removes the old lines and adds the new ones.
const maxDiffCells = 4_000_000

// Diff is a unified line diff of two texts, compared with CRLF as LF and
// without a byte order mark; "" when they are equal. The headers name the
// two sides.
func Diff(oldName, newName string, oldText, newText []byte) string {
	a, b := diffLines(oldText), diffLines(newText)
	ops := lineOps(a, b)
	var out strings.Builder
	changed := false
	for _, op := range ops {
		if op.kind != ' ' {
			changed = true
			break
		}
	}
	if !changed {
		return ""
	}
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", oldName, newName)
	for start := 0; start < len(ops); {
		// Find the next change, and the hunk around it.
		first := start
		for first < len(ops) && ops[first].kind == ' ' {
			first++
		}
		if first == len(ops) {
			break
		}
		from := max(first-diffContext, start)
		end := first
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run == len(ops) || run-end > 2*diffContext {
				end = min(end+diffContext, len(ops))
				break
			}
			end = run
		}
		hunk := ops[from:end]
		oldStart, newStart := ops[from].oldLine, ops[from].newLine
		oldCount, newCount := 0, 0
		for _, op := range hunk {
			if op.kind != '+' {
				oldCount++
			}
			if op.kind != '-' {
				newCount++
			}
		}
		fmt.Fprintf(&out, "@@ -%s +%s @@\n", hunkRange(oldStart, oldCount), hunkRange(newStart, newCount))
		for _, op := range hunk {
			text, last := strings.CutSuffix(op.text, noNewline)
			out.WriteString(string(op.kind) + text + "\n")
			if last {
				out.WriteString("\\ No newline at end of file\n")
			}
		}
		start = end
	}
	return out.String()
}

// hunkRange writes a hunk's start and length as unified diffs do: a range
// of no lines starts at the line before it.
func hunkRange(start, count int) string {
	if count == 0 {
		start--
	}
	if count == 1 {
		return fmt.Sprint(start)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

// noNewline marks a last line without a line end, so that it differs from
// the same line with one.
const noNewline = "\x00no newline"

func diffLines(text []byte) []string {
	text = bytes.TrimPrefix(text, []byte("\xef\xbb\xbf"))
	s := strings.ReplaceAll(string(text), "\r\n", "\n")
	if s == "" {
		return nil
	}
	ended := strings.HasSuffix(s, "\n")
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if !ended {
		lines[len(lines)-1] += noNewline
	}
	return lines
}

// lineOp is one line of the diff: kept (' '), removed ('-') or added ('+'),
// with its line numbers on both sides, counted from 1.
type lineOp struct {
	kind             byte
	text             string
	oldLine, newLine int
}

// lineOps is the shortest edit between two lists of lines, by their longest
// common subsequence.
func lineOps(a, b []string) []lineOp {
	// The common start and end need no table.
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	ma, mb := a[pre:len(a)-suf], b[pre:len(b)-suf]
	var ops []lineOp
	oldLine, newLine := 1, 1
	keep := func(s string) {
		ops = append(ops, lineOp{' ', s, oldLine, newLine})
		oldLine++
		newLine++
	}
	remove := func(s string) {
		ops = append(ops, lineOp{'-', s, oldLine, newLine})
		oldLine++
	}
	add := func(s string) {
		ops = append(ops, lineOp{'+', s, oldLine, newLine})
		newLine++
	}
	for _, s := range a[:pre] {
		keep(s)
	}
	if len(ma)*len(mb) > maxDiffCells {
		for _, s := range ma {
			remove(s)
		}
		for _, s := range mb {
			add(s)
		}
	} else {
		// lcs[i][j] is the longest common subsequence of ma[i:] and mb[j:].
		lcs := make([][]int32, len(ma)+1)
		for i := range lcs {
			lcs[i] = make([]int32, len(mb)+1)
		}
		for i := len(ma) - 1; i >= 0; i-- {
			for j := len(mb) - 1; j >= 0; j-- {
				if ma[i] == mb[j] {
					lcs[i][j] = lcs[i+1][j+1] + 1
				} else {
					lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
				}
			}
		}
		i, j := 0, 0
		for i < len(ma) || j < len(mb) {
			switch {
			case i < len(ma) && j < len(mb) && ma[i] == mb[j]:
				keep(ma[i])
				i++
				j++
			case j == len(mb) || i < len(ma) && lcs[i+1][j] >= lcs[i][j+1]:
				remove(ma[i])
				i++
			default:
				add(mb[j])
				j++
			}
		}
	}
	for _, s := range a[len(a)-suf:] {
		keep(s)
	}
	return ops
}
