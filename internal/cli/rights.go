package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// checksDocument is the JSON document of status with --step or --target.
type checksDocument struct {
	header
	// Step is the role of the step checked, "" without --step.
	Step string `json:"step"`
	// Errors keep the landing check from giving an answer.
	Errors     []diagnostic `json:"errors"`
	Violations []diagnostic `json:"violations"`
	// Landing says whether the landing check ran; Added are the entries the
	// branch adds.
	Landing bool        `json:"landing"`
	Added   []addedJSON `json:"added"`
}

type addedJSON struct {
	Kind    string   `json:"kind"`
	Doc     string   `json:"doc"`
	Changes []string `json:"changes"`
}

// entries is a number of entries.
func entries(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

// waitingStatus is the status of a loaded project with its waiting entries,
// and what keeps it from giving a list.
func waitingStatus(w *workspace) (*state.Status, []diagnostic, error) {
	s, problems, err := evaluate(w)
	if err != nil {
		return nil, nil, err
	}
	f, err := factoryOf(w)
	if err != nil {
		return nil, nil, err
	}
	errs := blockers(w, f, problems)
	if len(errs) == 0 {
		s.Wait(f.Reports)
	}
	return s, errs, nil
}

// statusChecks checks what a step wrote against the write table, with
// --step and --before, and whether the branch lands on its target, with
// --target. It changes nothing; the tests and the counter-checks are the
// runner's part of the landing rule.
func statusChecks(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	doc := checksDocument{header: newHeader("status"), Step: o.step, Errors: []diagnostic{}, Violations: []diagnostic{}, Added: []addedJSON{}}
	var target *workspace
	if o.target != "" {
		dir, err := projectAt(o.target)
		if err != nil {
			return ExitFailed, err
		}
		if target, err = loadDir(dir); err != nil {
			return ExitFailed, err
		}
	}
	var b strings.Builder
	if o.step != "" {
		dir, err := projectAt(o.before)
		if err != nil {
			return ExitFailed, err
		}
		before, err := loadDir(dir)
		if err != nil {
			return ExitFailed, err
		}
		control := before
		if target != nil {
			control = target
		}
		violations, err := state.CheckStep(o.step, before.Project, w.Project, control.Project)
		if err != nil {
			return ExitFailed, err
		}
		for _, v := range violations {
			doc.Violations = append(doc.Violations, diagnostic{File: v.Path, Line: 0, Code: state.CodeWriteDenied, Message: v.Message})
			b.WriteString(v.String() + "\n")
		}
		fmt.Fprintf(&b, "step %s: %s\n", o.step, plural(len(violations), "change")+" the write table denies")
	}
	if target != nil {
		doc.Landing = true
		s, errs, err := waitingStatus(w)
		if err != nil {
			return ExitFailed, err
		}
		doc.Errors = append(doc.Errors, errs...)
		if len(errs) > 0 {
			for _, d := range errs {
				b.WriteString(d.String() + "\n")
			}
			b.WriteString("landing: no answer while check finds errors or the state has errors\n")
		} else {
			t, _, err := waitingStatus(target)
			if err != nil {
				return ExitFailed, err
			}
			added := state.Landing(s, t)
			for _, a := range added {
				doc.Added = append(doc.Added, addedJSON{a.Kind, a.Doc, nonNil(a.Changes)})
				line := "adds " + a.Kind + ": " + a.Doc
				if len(a.Changes) > 0 {
					line += ": " + strings.Join(a.Changes, ", ")
				}
				b.WriteString(line + "\n")
			}
			fmt.Fprintf(&b, "landing: %s the target does not show; the tests and the counter-checks are the runner's to run\n", entries(len(added)))
		}
	}
	code, err := output(o, stdout, doc, b.String())
	if err == nil && len(doc.Violations)+len(doc.Errors)+len(doc.Added) > 0 {
		code = ExitFound
	}
	return code, err
}
