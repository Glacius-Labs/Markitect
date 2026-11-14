package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// ErrUsage marks a verdict the command line cannot ask for, such as an unknown
// one.
var ErrUsage = errors.New("usage")

// MarkRequest is one verdict, at the state the reviewer read.
type MarkRequest struct {
	Dir string
	// Item is DEFINITION@FINGERPRINT, as status printed it.
	Item     string
	Verdict  string
	Evidence []string
	Why      string
	// Cases and Covered are the reviewer's counts of the Definition's cases,
	// conditions and exceptions, and of those its evidence demonstrates; nil when
	// it gave none.
	Cases, Covered *int
	// definition and at are Item's two parts.
	definition, at string
}

// check refuses a request the command line cannot make.
func (r *MarkRequest) check() error {
	def, at, ok := strings.Cut(r.Item, "@")
	r.definition = strings.TrimSuffix(strings.TrimPrefix(def, "[["), "]]")
	r.at = strings.ToLower(at)
	r.Why = oneLine(r.Why)
	switch {
	case !ok || r.definition == "" || len(r.at) < fingerprintLen:
		return fmt.Errorf("%w: name the item as markitect status prints it, DEFINITION@FINGERPRINT, not %q", ErrUsage, r.Item)
	case !knownVerdict(r.Verdict):
		return fmt.Errorf("%w: the verdict is one of %s, not %q", ErrUsage, strings.Join(Verdicts, ", "), r.Verdict)
	case r.Verdict != VerdictOK && r.Why == "":
		return fmt.Errorf("%w: a %s says why, with -m", ErrUsage, r.Verdict)
	case r.Cases != nil && *r.Cases < 0, r.Covered != nil && *r.Covered < 0:
		return fmt.Errorf("%w: --cases and --covered count, so they are 0 or more", ErrUsage)
	case r.Covered != nil && r.Cases == nil:
		return fmt.Errorf("%w: --covered counts some of the cases, so give --cases too", ErrUsage)
	case r.Covered != nil && *r.Covered > *r.Cases:
		return fmt.Errorf("%w: --covered %d is more than --cases %d; the evidence demonstrates at most every case", ErrUsage, *r.Covered, *r.Cases)
	}
	return nil
}

// Mark records a verdict on one Definition at the state the reviewer read. It
// refuses when the inputs changed since status printed the fingerprint, or when
// evidence does not exist in the folder; then it records nothing, and refusal
// says why. Marking a gone Definition ok removes its line.
func (e *Engine) Mark(_ context.Context, req MarkRequest) (text string, recorded bool, err error) {
	if err := req.check(); err != nil {
		return "", false, err
	}
	var evidence []Evidence
	for _, raw := range req.Evidence {
		ev, err := ParseEvidence(raw)
		if err != nil {
			return "", false, fmt.Errorf("%w: %v", ErrUsage, err)
		}
		evidence = append(evidence, ev)
	}
	root, err := e.Folder.Root(req.Dir)
	if err != nil {
		return "", false, err
	}
	dir, err := e.lockDir()
	if err != nil {
		return "", false, err
	}
	release, err := lock(dir, recordKey(root), lockWait)
	if err != nil {
		return "", false, fmt.Errorf("the record of %s is locked: %w", root, err)
	}
	defer release()
	f, err := e.readFolder(root)
	if err != nil {
		return "", false, err
	}
	data, err := readRecordFile(f.root)
	if err != nil {
		return "", false, err
	}
	rec := ParseRecord(data)
	refuse := func(format string, args ...any) (string, bool, error) {
		return fmt.Sprintf(format, args...) + " Nothing is recorded.\n", false, nil
	}
	switch {
	case len(f.cur.SpecErrors()) > 0:
		return refuse("The spec has errors, so nothing can be verified against it; markitect check lists them.")
	case len(rec.Unreadable) > 0:
		return refuse("markitect cannot read every line of %s; the owner fixes or removes the ones markitect status names.", project.ReviewFile)
	}
	v, err := f.see(rec)
	if err != nil {
		return "", false, err
	}
	it, ok := v.items[req.definition]
	inputs := v.now[req.definition]
	if it.State == StateGone {
		inputs = v.goneInputs[req.definition]
	}
	switch {
	case !ok && f.cur.Spec.UnitOf(req.definition) != req.definition:
		return refuse("%s is a part of %s, which is verified as one with its parts: record the verdict on that item, as markitect status prints it.", req.definition, f.cur.Spec.UnitOf(req.definition))
	case !ok:
		return refuse("%s is no Definition of the spec, and the record has no line for it.", req.definition)
	case !strings.HasPrefix(inputs.fingerprint(), req.at):
		return refuse("The inputs of %s are not those of fingerprint %s: they changed since status printed it, or it belongs to another item. Run markitect status again and verify %s as it is now.", req.definition, req.at, req.definition)
	case it.State == StateGone && req.Verdict == VerdictSkip:
		return refuse("%s is gone from the spec: its files are removed or have a new basis (ok), or not (fail).", req.definition)
	case it.State != StateGone && req.Verdict == VerdictOK && len(evidence) == 0:
		return "", false, fmt.Errorf("%w: ok names its evidence with -e, such as -e src/fees.go:40", ErrUsage)
	case it.State != StateGone && (req.Verdict == VerdictOK || req.Verdict == VerdictSkip) && (req.Cases == nil || req.Covered == nil):
		return "", false, fmt.Errorf("%w: %s counts the cases the Definition demands and how many of them the evidence demonstrates, with --cases N --covered M; a Definition that demands no case counts 0 and 0", ErrUsage, req.Verdict)
	}
	line := Line{Definition: req.definition, Verdict: req.Verdict, Why: req.Why, Inputs: Inputs{}, Cases: req.Cases, Covered: req.Covered}
	for k, h := range inputs {
		line.Inputs[k] = h
	}
	before := rec.byDefinition()
	line.Revisions = revisions(before[req.definition], inputs)
	// A round of change is every item whose own text changed since its last
	// verdict. A revision counts a co-change with each other item of its round;
	// that item's own verdict later sees this one verified, and that item's line
	// names this one as counted, so that its one change counts with this one
	// once however often this one is revised before it. A round of more than
	// maxChangeSet counts nothing, and its other items' lines say so, so that
	// they count with nothing while the round shrinks as it is verified.
	revised := line.Revisions > mostRevisions(before[req.definition])
	others := changed(before, v.now, req.definition)
	round := len(others)
	if revised {
		round++
	}
	large := round > maxChangeSet
	var with []string
	if revised && !large && !batched(before[req.definition]) {
		for _, id := range others {
			if !batched(before[id]) && !slices.Contains(mostCounted(before[id]), req.definition) {
				with = append(with, id)
			}
		}
	}
	line.Cochanges = live(req.definition, mostCochanges(before[req.definition]), v.now)
	for _, id := range with {
		line.Cochanges[id]++
	}
	var missing []string
	for _, ev := range evidence {
		h, err := f.hash(ev.File)
		if err != nil {
			return "", false, err
		}
		if h == "" || ev.File == project.ReviewFile {
			missing = append(missing, ev.File)
			continue
		}
		line.Inputs[InputEvidence+ev.File] = h
		line.Evidence = append(line.Evidence, ev.Text)
	}
	if len(missing) > 0 {
		return refuse("The evidence must be files of the project folder, and these are not: %s.", strings.Join(missing, ", "))
	}
	lines := clean(rec, v, req.definition)
	for i := range lines {
		lines[i].Cochanges = live(lines[i].Definition, lines[i].Cochanges, v.now)
		lines[i].Counted = liveIDs(lines[i].Counted, v.now)
		if slices.Contains(with, lines[i].Definition) {
			lines[i].Cochanges[req.definition]++
			lines[i].Counted = append(lines[i].Counted, req.definition)
			slices.Sort(lines[i].Counted)
		}
		if large && slices.Contains(others, lines[i].Definition) {
			lines[i].Batch = true
		}
	}
	removed := it.State == StateGone && req.Verdict == VerdictOK
	dirs := rec.Directories
	if !removed {
		lines = append(lines, line)
		// The directories the new line names, so that status can say later
		// which binding Definition changed since.
		for k := range line.Inputs {
			if name, ok := strings.CutPrefix(k, InputBound); ok {
				dirs = append(dirs, v.dirs[name])
			}
		}
	}
	written := Record{Lines: lines, Directories: dirs}.Render()
	if err := writeAtomic(filepath.Join(f.root, filepath.FromSlash(project.Dir)), filepath.Base(project.ReviewFile), written); err != nil {
		return "", false, err
	}
	if removed {
		return fmt.Sprintf("Recorded: %s is gone and its files are dealt with, so its line is removed.\n", req.definition), true, nil
	}
	return fmt.Sprintf("Recorded: %s %s.\n", req.definition, req.Verdict), true, nil
}

// clean keeps one line per Definition, except the one being marked: of several,
// as after a merge, the one whose inputs match the folder, or both of a conflict,
// or else the closest; a gone Definition keeps one line with every file its lines
// named. A kept line counts the most revisions and co-changes any of its
// Definition's lines counted, so that a merge loses none, names every
// Definition any of them names as counted, and is in a large round when any of
// them is.
func clean(rec Record, v view, except string) []Line {
	var out []Line
	for id, lines := range rec.byDefinition() {
		switch {
		case id == except, v.covered[id]:
		case len(lines) == 1:
			out = append(out, lines[0])
		case v.items[id].State == StateGone:
			merged := lines[0]
			merged.Inputs = Inputs{}
			for k, h := range lines[0].Inputs {
				merged.Inputs[k] = h
			}
			for _, l := range lines[1:] {
				for k, h := range l.Inputs {
					if _, ok := merged.Inputs[k]; !ok && strings.HasPrefix(k, InputFile) {
						merged.Inputs[k] = h
					}
				}
			}
			merged.Revisions, merged.Cochanges, merged.Counted, merged.Batch = mostRevisions(lines), mostCochanges(lines), mostCounted(lines), batched(lines)
			out = append(out, merged)
		default:
			for _, l := range keep(lines, v.now[id], v) {
				l.Revisions, l.Cochanges, l.Counted, l.Batch = mostRevisions(lines), mostCochanges(lines), mostCounted(lines), batched(lines)
				out = append(out, l)
			}
		}
	}
	return out
}

// mostRevisions is the most revisions any of the lines counts.
func mostRevisions(lines []Line) int {
	n := 0
	for _, l := range lines {
		n = max(n, l.Revisions)
	}
	return n
}

// live keeps, in a new map, the co-changes of a line on id with items of the
// spec as it is: a Definition gone from it changes with nothing any more.
func live(id string, cochanges map[string]int, now map[string]Inputs) map[string]int {
	out := map[string]int{}
	if _, ok := now[id]; !ok {
		return out
	}
	for id, n := range cochanges {
		if _, ok := now[id]; ok {
			out[id] = n
		}
	}
	return out
}

// liveIDs keeps, in a new slice, the Definitions that are items of the spec as
// it is.
func liveIDs(ids []string, now map[string]Inputs) []string {
	var out []string
	for _, id := range ids {
		if _, ok := now[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// keep chooses among the lines of one live Definition.
func keep(lines []Line, now Inputs, v view) []Line {
	s := settle(lines, now, v.evidence)
	switch {
	case len(s.conflict) > 0:
		var out []Line
		seen := map[string]bool{}
		for _, l := range s.conflict {
			if !seen[l.Verdict] {
				seen[l.Verdict] = true
				out = append(out, l)
			}
		}
		return out
	case s.match != nil:
		return []Line{*s.match}
	case s.closest != nil:
		return []Line{*s.closest}
	}
	return nil
}

// recordKey names the lock of a project folder's record.
func recordKey(root string) string {
	p := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:8])
}
