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

// ErrUsage marks a review the command line cannot ask for, such as an unknown
// verdict.
var ErrUsage = errors.New("usage")

// ReviewRequest is one verdict, at the state the reviewer read.
type ReviewRequest struct {
	Dir        string
	Definition string
	Verdict    string
	// At is the fingerprint status printed for the item.
	At       string
	Evidence []string
	Why      string
}

// check refuses a request the command line cannot make.
func (r *ReviewRequest) check() error {
	r.Definition = strings.TrimSuffix(strings.TrimPrefix(r.Definition, "[["), "]]")
	r.Why = oneLine(r.Why)
	switch {
	case r.Definition == "":
		return fmt.Errorf("%w: name the Definition", ErrUsage)
	case !knownVerdict(r.Verdict):
		return fmt.Errorf("%w: the verdict is one of %s, not %q", ErrUsage, strings.Join(Verdicts, ", "), r.Verdict)
	case r.At == "":
		return fmt.Errorf("%w: --at names the fingerprint markitect status printed for the item", ErrUsage)
	case r.Verdict != VerdictOK && r.Why == "":
		return fmt.Errorf("%w: a %s says why, after --", ErrUsage, r.Verdict)
	}
	return nil
}

// Review records a verdict on one Definition at the state the reviewer read. It
// refuses when the inputs changed since status printed the fingerprint, or when
// evidence does not exist in the folder; then it records nothing, and refusal
// says why. Reviewing a gone Definition ok removes its line.
func (e *Engine) Review(_ context.Context, req ReviewRequest) (text string, recorded bool, err error) {
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
	if e.Folder == nil {
		return "", false, errors.New("this build reads no project folder")
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
	it, ok := v.items[req.Definition]
	switch {
	case !ok:
		return refuse("%s is no Definition of the spec, and the record has no line for it.", req.Definition)
	case !strings.EqualFold(req.At, it.Fingerprint):
		return refuse("The inputs of %s are not those of fingerprint %s: they changed since status printed it, or it belongs to another item. Run markitect status again and verify %s as it is now.", req.Definition, req.At, req.Definition)
	case it.State == StateGone && req.Verdict == VerdictSkip:
		return refuse("%s is gone from the spec: its files are removed or have a new basis (ok), or not (fail).", req.Definition)
	case it.State != StateGone && req.Verdict == VerdictOK && len(evidence) == 0:
		return "", false, fmt.Errorf("%w: ok names the evidence, such as FILE:LINE", ErrUsage)
	}
	line := Line{Definition: req.Definition, Verdict: req.Verdict, Why: req.Why}
	if it.State == StateGone {
		line.Inputs = Inputs{}
		for k, h := range v.goneInputs[req.Definition] {
			line.Inputs[k] = h
		}
	} else {
		line.Inputs = Inputs{}
		for k, h := range v.now[req.Definition] {
			line.Inputs[k] = h
		}
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
	lines := clean(rec, v, req.Definition)
	removed := it.State == StateGone && req.Verdict == VerdictOK
	if !removed {
		lines = append(lines, line)
	}
	if err := writeAtomic(filepath.Join(f.root, filepath.FromSlash(project.Dir)), filepath.Base(project.ReviewFile), Render(lines)); err != nil {
		return "", false, err
	}
	if removed {
		return fmt.Sprintf("Recorded: %s is gone and its files are dealt with, so its line is removed.\n", req.Definition), true, nil
	}
	return fmt.Sprintf("Recorded: %s %s.\n", req.Definition, req.Verdict), true, nil
}

// clean keeps one line per Definition, except the one being reviewed: of several,
// as after a merge, the one whose inputs match the folder, or both of a conflict,
// or else the closest; a gone Definition keeps one line with every file its lines
// named.
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
			out = append(out, merged)
		default:
			out = append(out, keep(lines, v.now[id], v)...)
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
