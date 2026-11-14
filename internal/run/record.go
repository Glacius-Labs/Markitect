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
// Cases, Covered and Revisions came with 0.23.0 (DEC-102), Cochanges and Batch
// later; each is left out when empty, so a line written before reads as no
// count, no revision and no co-change.
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
	// Cochanges counts, per other Definition, how often its own text had changed
	// too, and was not verified again yet, when a verdict counted a revision of
	// this one: how often the two changed in one round.
	Cochanges map[string]int `json:"cochanges,omitempty"`
	// Batch says that the Definition's own text changed, since this line, in a
	// round of more than maxChangeSet Definitions, so that the verdict on that
	// text counts no co-change.
	Batch bool `json:"batch,omitempty"`
}

// maxChangeSet is the most Definitions one round of change may hold for mark
// to count its co-changes: a larger batch, such as a reorganisation, says
// nothing about which of them belong together. It is code-maat's largest
// change set.
const maxChangeSet = 30

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

// Cochanges returns, per pair of Definitions, how often both own texts changed
// in one round, in both directions: of several lines, as after a merge, the
// most. Definitions with none are left out.
func (r Record) Cochanges() map[string]map[string]int {
	out := map[string]map[string]int{}
	add := func(a, b string, n int) {
		if out[a] == nil {
			out[a] = map[string]int{}
		}
		out[a][b] = max(out[a][b], n)
	}
	for _, l := range r.Lines {
		for id, n := range l.Cochanges {
			add(l.Definition, id, n)
			add(id, l.Definition, n)
		}
	}
	return out
}

// Counts is what a reviewer counted with an ok or a skip: the cases the
// Definition demands and how many of them the evidence demonstrates.
type Counts struct {
	Cases, Covered int
	// Skip says that the verdict was a skip: verified as far as the project can
	// show.
	Skip bool
}

// Counts returns, per Definition, what reviewers counted with each ok or skip
// the record holds for the Definition's own text as it is: self gives each
// Definition's hash now, as Hash makes it. A line at another text speaks of a
// Definition that has changed since, and a Definition with a fail or a
// spec-problem at this text, or with a line there that carries no counts, has
// none. After a merge the record may hold several lines for one Definition, so
// a Definition may have several counts; whoever reads them chooses.
func (r Record) Counts(self map[string]string) map[string][]Counts {
	out := map[string][]Counts{}
	none := map[string]bool{}
	for _, l := range r.Lines {
		h, ok := self[l.Definition]
		if !ok || l.Inputs[InputSelf] != h {
			continue
		}
		if (l.Verdict != VerdictOK && l.Verdict != VerdictSkip) || l.Cases == nil || l.Covered == nil {
			none[l.Definition] = true
			continue
		}
		out[l.Definition] = append(out[l.Definition], Counts{Cases: *l.Cases, Covered: *l.Covered, Skip: l.Verdict == VerdictSkip})
	}
	for id := range none {
		delete(out, id)
	}
	return out
}

// revisions is the count of a new line at inputs now, after the lines the record
// held for its Definition: theirs, one more when its own text is none of theirs.
// A gone Definition has no own text, so a verdict on it adds none.
func revisions(before []Line, now Inputs) int {
	_, has := now[InputSelf]
	n, same := mostRevisions(before), len(before) == 0 || !has
	for _, l := range before {
		if l.Inputs[InputSelf] == now[InputSelf] {
			same = true
		}
	}
	if !same {
		n++
	}
	return n
}

// changed lists, sorted, the items other than except whose own text is none of
// the texts the record's lines on them hold: what changed since its last
// verdict and is not verified again yet. An item with no line yet has no
// change to count.
func changed(lines map[string][]Line, now map[string]Inputs, except string) []string {
	var out []string
	for _, id := range sortedKeys(now) {
		if id == except || len(lines[id]) == 0 {
			continue
		}
		same := false
		for _, l := range lines[id] {
			if l.Inputs[InputSelf] == now[id][InputSelf] {
				same = true
			}
		}
		if !same {
			out = append(out, id)
		}
	}
	return out
}

// mostCochanges is, per other Definition, the most co-changes any of the lines
// counts.
func mostCochanges(lines []Line) map[string]int {
	out := map[string]int{}
	for _, l := range lines {
		for id, n := range l.Cochanges {
			out[id] = max(out[id], n)
		}
	}
	return out
}

// batched says whether any of the lines is in a round too large to count.
func batched(lines []Line) bool {
	for _, l := range lines {
		if l.Batch {
			return true
		}
	}
	return false
}

// Directory is what bound a Package and every Package below it when a line
// was written: each binding thing of the Package by its ID, with the hash of
// its text and the texts of the parts it covers (DEC-113). Its digest is the
// input bound: of the Package on the lines of what it binds, and the record
// keeps the directory beside them, so that status can name the binding
// Definition that changed once that Definition is verified again.
type Directory struct {
	// Bound is the Package's name, Root for the one at the top.
	Bound  string            `json:"bound"`
	Digest string            `json:"digest"`
	Binds  map[string]string `json:"binds"`
}

// key names a directory by its Package and its digest.
func (d Directory) key() string { return d.Bound + "\x00" + d.Digest }

// Record is the record of a project: its lines, the directories they name, and
// the lines it could not read, such as one written by hand, with their line
// numbers.
type Record struct {
	Lines       []Line
	Directories []Directory
	Unreadable  []int
}

// ParseRecord reads a record: one JSON object per line, a verdict or a
// directory.
func ParseRecord(data []byte) Record {
	var r Record
	for i, raw := range strings.Split(string(project.Normalize(data)), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var l Line
		if err := json.Unmarshal([]byte(raw), &l); err == nil && validLine(l) {
			r.Lines = append(r.Lines, l)
			continue
		}
		var d Directory
		if err := json.Unmarshal([]byte(raw), &d); err == nil && validDirectory(d) {
			r.Directories = append(r.Directories, d)
			continue
		}
		r.Unreadable = append(r.Unreadable, i+1)
	}
	return r
}

// validDirectory says whether a directory names its Package, binds something
// and has the digest of what it binds, so that one changed by hand is not
// taken for what bound a line.
func validDirectory(d Directory) bool {
	return d.Bound != "" && len(d.Binds) > 0 && d.Digest == digest(d.Binds)
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

// Render writes the record as its file holds it: one JSON object per line, the
// lines sorted and after them the directories they name, sorted, with no line
// twice. A directory that no line names is left out.
func (r Record) Render() []byte {
	named := map[string]bool{}
	var lines []any
	for _, l := range r.Lines {
		lines = append(lines, l)
		for k, h := range l.Inputs {
			if name, ok := strings.CutPrefix(k, InputBound); ok {
				named[Directory{Bound: name, Digest: h}.key()] = true
			}
		}
	}
	var dirs []any
	for _, d := range r.Directories {
		if named[d.key()] {
			dirs = append(dirs, d)
		}
	}
	out := append(sortedLines(lines), sortedLines(dirs)...)
	if len(out) == 0 {
		return nil
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// sortedLines writes each value as one line, sorted, with no line twice.
func sortedLines(values []any) []string {
	var out []string
	seen := map[string]bool{}
	for _, v := range values {
		data, err := marshal(v)
		if err != nil || seen[data] {
			continue
		}
		seen[data] = true
		out = append(out, data)
	}
	sort.Strings(out)
	return out
}

// marshal writes one line without escaping <, > and &, which a reason may hold.
func marshal(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
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
