package run

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Verdicts, after go test.
const (
	// VerdictOK: the Definition is implemented, as the evidence shows.
	VerdictOK = "ok"
	// VerdictFail: it is not; the line says why.
	VerdictFail = "fail"
	// VerdictSkip: verified as far as the project can show, with the reason. It is
	// a note, and a note alone does not make status exit 1.
	VerdictSkip = "skip"
	// VerdictSpecProblem: the spec contradicts itself or leaves open behaviour at
	// the project's boundary, and only the owner can settle it.
	VerdictSpecProblem = "spec-problem"
)

// Verdicts lists the verdicts in the order help names them.
var Verdicts = []string{VerdictOK, VerdictFail, VerdictSkip, VerdictSpecProblem}

// Line is one line of the record: the verdict on one Definition at its inputs.
// Cases, Covered and Revisions came with 0.23.0 (DEC-101) and are left out when
// empty, so a line written before reads as no count and no revision.
type Line struct {
	Definition string   `json:"definition"`
	Verdict    string   `json:"verdict"`
	Inputs     Inputs   `json:"inputs"`
	Evidence   []string `json:"evidence,omitempty"`
	Why        string   `json:"why,omitempty"`
	// Cases is how many distinct cases, conditions and exceptions the reviewer
	// counted in the Definition, and Covered how many of them its evidence
	// demonstrates; nil when the reviewer gave no count.
	Cases   *int `json:"cases,omitempty"`
	Covered *int `json:"covered,omitempty"`
	// Revisions counts how often the Definition's own text changed between two
	// verdicts on it: a verdict at another text than every line before it adds one.
	Revisions int `json:"revisions,omitempty"`
}

// Revisions returns, per Definition, how often its own text changed between two
// verdicts on it: of several lines, as after a merge, the most. Definitions with
// none are left out.
func (r Record) Revisions() map[string]int {
	out := map[string]int{}
	for _, l := range r.Lines {
		if l.Revisions > out[l.Definition] {
			out[l.Definition] = l.Revisions
		}
	}
	return out
}

// revisions is the count of a new line at inputs now, after the lines the record
// held for its Definition: theirs, one more when its own text is none of theirs.
func revisions(before []Line, now Inputs) int {
	n, same := 0, len(before) == 0
	for _, l := range before {
		n = max(n, l.Revisions)
		if l.Inputs[InputSelf] == now[InputSelf] {
			same = true
		}
	}
	if !same {
		n++
	}
	return n
}

// Record is the record of a project: its lines, and the lines it could not read,
// such as one written by hand, with their line numbers.
type Record struct {
	Lines      []Line
	Unreadable []int
}

// ParseRecord reads a record: one JSON object per line.
func ParseRecord(data []byte) Record {
	var r Record
	for i, raw := range strings.Split(string(project.Normalize(data)), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var l Line
		if err := json.Unmarshal([]byte(raw), &l); err != nil || !validLine(l) {
			r.Unreadable = append(r.Unreadable, i+1)
			continue
		}
		r.Lines = append(r.Lines, l)
	}
	return r
}

func validLine(l Line) bool {
	if l.Definition == "" || l.Inputs == nil || !knownVerdict(l.Verdict) {
		return false
	}
	return true
}

func knownVerdict(v string) bool {
	for _, w := range Verdicts {
		if v == w {
			return true
		}
	}
	return false
}

// Render writes the lines as the record holds them: one JSON object per line,
// sorted, with no line twice.
func Render(lines []Line) []byte {
	var out []string
	seen := map[string]bool{}
	for _, l := range lines {
		data, err := marshal(l)
		if err != nil || seen[data] {
			continue
		}
		seen[data] = true
		out = append(out, data)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// marshal writes one line without escaping <, > and &, which a reason may hold.
func marshal(l Line) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(l); err != nil {
		return "", err
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// byDefinition groups the lines by Definition.
func (r Record) byDefinition() map[string][]Line {
	out := map[string][]Line{}
	for _, l := range r.Lines {
		out[l.Definition] = append(out[l.Definition], l)
	}
	return out
}

// Change is one input that differs between a line and the folder.
type Change struct {
	Key string `json:"key"`
	// How is "changed", "new" (the folder has it, the line not) or "gone" (the
	// line has it, the folder not).
	How string `json:"how"`
}

func (c Change) String() string {
	if c.How == "changed" {
		return c.Key
	}
	return fmt.Sprintf("%s (%s)", c.Key, c.How)
}

// compare lists what differs between a line's inputs and the folder's: now holds
// the folder's inputs without evidence, and evidence hashes the evidence files.
func compare(line Inputs, now Inputs, evidence func(file string) string) []Change {
	var out []Change
	for _, k := range sortedKeys(now) {
		switch was, ok := line[k]; {
		case !ok:
			out = append(out, Change{k, "new"})
		case was != now[k]:
			out = append(out, Change{k, "changed"})
		}
	}
	for _, k := range sortedKeys(line) {
		if file, ok := strings.CutPrefix(k, InputEvidence); ok {
			if h := evidence(file); h != line[k] {
				how := "changed"
				if h == gone {
					how = "gone"
				}
				out = append(out, Change{k, how})
			}
			continue
		}
		if _, ok := now[k]; !ok {
			out = append(out, Change{k, "gone"})
		}
	}
	return out
}

// settled is what the lines of one Definition say about it at the folder's
// inputs: the line that matches, or else the closest one and what changed, or a
// conflict between matching lines with different verdicts.
type settled struct {
	match    *Line
	closest  *Line
	changes  []Change
	conflict []Line
}

func settle(lines []Line, now Inputs, evidence func(string) string) settled {
	var s settled
	var matching []Line
	best := -1
	for i := range lines {
		changes := compare(lines[i].Inputs, now, evidence)
		if len(changes) == 0 {
			matching = append(matching, lines[i])
			continue
		}
		if best < 0 || len(changes) < len(s.changes) {
			best, s.changes = i, changes
		}
	}
	if len(matching) > 0 {
		s.changes = nil
		s.match = &matching[0]
		for _, l := range matching[1:] {
			if l.Verdict != s.match.Verdict {
				s.conflict = matching
				break
			}
		}
		return s
	}
	if best >= 0 {
		s.closest = &lines[best]
	}
	return s
}
