package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
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
	case !ok:
		return refuse("%s is no Definition of the spec, and the record has no line for it.", req.definition)
	case !strings.HasPrefix(inputs.fingerprint(), req.at):
		return refuse("The inputs of %s are not those of fingerprint %s: they changed since status printed it, or it belongs to another item. Run markitect status again and verify %s as it is now.", req.definition, req.at, req.definition)
	case it.State == StateGone && req.Verdict == VerdictSkip:
		return refuse("%s is gone from the spec: its files are removed or have a new basis (ok), or not (fail).", req.definition)
	case it.State != StateGone && req.Verdict == VerdictOK && len(evidence) == 0:
		return "", false, fmt.Errorf("%w: ok names its evidence with -e, such as -e src/fees.go:40", ErrUsage)
	case it.State != StateGone && (req.Verdict == VerdictOK || req.Verdict == VerdictSkip) && (req.Cases == nil || req.Covered == nil):
		return "", false, fmt.Errorf("%w: %s counts the cases the Definition demands and how many of them the evidence demonstrates, with --cases N --covered M", ErrUsage, req.Verdict)
	}
	line := Line{Definition: req.definition, Verdict: req.Verdict, Why: req.Why, Inputs: Inputs{}, Cases: req.Cases, Covered: req.Covered}
	for k, h := range inputs {
		line.Inputs[k] = h
	}
	line.Revisions = revisions(rec.byDefinition()[req.definition], inputs)
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
	removed := it.State == StateGone && req.Verdict == VerdictOK
	if !removed {
		lines = append(lines, line)
	}
	if err := writeAtomic(filepath.Join(f.root, filepath.FromSlash(project.Dir)), filepath.Base(project.ReviewFile), Render(lines)); err != nil {
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
// named. A kept line counts the most revisions any of its Definition's lines
// counted, so that a merge loses none.
func clean(rec Record, v view, except string) []Line {
	var out []Line
	for id, lines := range rec.byDefinition() {
		switch {
		case id == except:
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
			merged.Revisions = mostRevisions(lines)
			out = append(out, merged)
		default:
			for _, l := range keep(lines, v.now[id], v) {
				l.Revisions = mostRevisions(lines)
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
