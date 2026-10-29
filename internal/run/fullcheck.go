package run

import (
	"context"
	"fmt"
)

// Trailers of the full-check count (concept 2.0 §6.1). A Result commit on
// a Run branch names its command and state, so a `verify --full` that
// ended verified or with findings is a full check while its branch exists.
// A landing carries the newest full check it knows forward, so that check
// survives the deletion of the Run branch it came from.
const (
	trailerCommand   = "Run-Command"
	trailerState     = "Run-State"
	trailerFullCheck = "Run-Full-Check"
	// fullCheckNone is the Run-Full-Check of a landing that knows no full
	// check.
	fullCheckNone = "none"
)

// fullCheck counts the landings in tip's history after the commit the newest
// full check examined: a landed `apply --full` is its own landing; a
// `verify --full` examined its start; and the newest landing that names one
// carries it forward. With none, every landing counts. A verify's Run
// branch deleted before any landing carried it forward loses its reset, so
// the count errs toward a full check being due.
func (e *Engine) fullCheck(ctx context.Context, target, tip string) FullCheck {
	fc, err := e.countFullCheck(ctx, target, tip)
	if err != nil {
		return FullCheck{Note: fmt.Sprintf("the landings since the last full check could not be read: %v", err)}
	}
	return fc
}

func (e *Engine) countFullCheck(ctx context.Context, target, tip string) (FullCheck, error) {
	landings, err := e.Git.Landings(ctx, tip, "")
	if err != nil {
		return FullCheck{}, err
	}
	var checks []string
	for _, l := range landings {
		c, err := e.Git.Commit(ctx, l)
		if err != nil {
			return FullCheck{}, err
		}
		t := trailers(c.Message)
		if v := t[trailerFull]; len(v) == 1 && v[0] == "true" {
			checks = append(checks, l)
			break
		}
		// Every landing since this count exists names a check or none,
		// so the walk ends at the newest one.
		if v := t[trailerFullCheck]; len(v) == 1 {
			if v[0] != fullCheckNone {
				checks = append(checks, v[0])
			}
			break
		}
	}
	branches, err := e.Git.Branches(ctx, RunBranchPrefix)
	if err != nil {
		return FullCheck{}, err
	}
	for _, branch := range sortedKeys(branches) {
		c, err := e.Git.Commit(ctx, branches[branch])
		if err != nil {
			return FullCheck{}, err
		}
		if start, ok := verifiedWhole(trailers(c.Message), target); ok {
			checks = append(checks, start)
		}
	}
	fc := FullCheck{Landed: len(landings)}
	seen := map[string]bool{}
	for _, check := range checks {
		if seen[check] {
			continue
		}
		seen[check] = true
		// A full check of a history the target no longer holds says
		// nothing about it.
		if ok, err := e.Git.Contains(ctx, tip, check); err != nil {
			return FullCheck{}, err
		} else if !ok {
			continue
		}
		since, err := e.Git.Landings(ctx, tip, check)
		if err != nil {
			return FullCheck{}, err
		}
		if fc.Since == "" || len(since) < fc.Landed {
			fc.Landed, fc.Since = len(since), check
		}
	}
	fc.Due = fc.Landed >= FullCheckEvery
	return fc, nil
}

// verifiedWhole reads the start of a `verify --full` of the target that
// ended verified or with findings from its Run branch's Result commit.
func verifiedWhole(t map[string][]string, target string) (string, bool) {
	one := func(k string) string {
		if v := t[k]; len(v) == 1 {
			return v[0]
		}
		return ""
	}
	state := one(trailerState)
	if one(trailerCommand) != "verify" || one(trailerFull) != "true" || one(trailerTarget) != target ||
		(state != StateVerified && state != StateFindings) || one(trailerStart) == "" {
		return "", false
	}
	return one(trailerStart), true
}

// fullCheckLine is the Result page's account of the count.
func fullCheckLine(r *Result) string {
	fc := r.FullCheck
	switch {
	case fc.Note != "":
		return fc.Note
	case r.State == StateLanded && r.Full:
		return "this landing is a full check, so the count starts again"
	}
	since := "since the first landing"
	if fc.Since != "" {
		since = "since the full check of " + short(fc.Since)
	}
	line := fmt.Sprintf("%d %s landed %s", fc.Landed, plural(fc.Landed, "change", "changes"), since)
	if fc.Due {
		line += fmt.Sprintf("; a full check is due from %d on: `markitect verify --full`", FullCheckEvery)
	}
	return line
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// isFullCheck reports whether this Run is itself a full check once it
// ends: a `verify --full` that ended verified or with findings.
func isFullCheck(r *Result) bool {
	return r.Full && r.Command == "verify" && (r.State == StateVerified || r.State == StateFindings)
}
