package run

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// landAttempts bounds how often landing starts again when the owner's branch
// moves or Git's own lock holds while it lands.
const landAttempts = 3

// ResultFile is where a Run's Result page lives in the repository.
func ResultFile(run string) string { return path.Join(project.ResultsDir, run+".md") }

// land brings the owner's branch up. What will land is built off to the side: the
// work if the branch has not moved, otherwise a merge, which must be clean and
// pass check --spec and the required checks. Later commits stay in the next
// impact, so the next Run realizes them. If the branch cannot be moved, the Run
// closes as kept and the next apply tries once more.
func (e *Engine) land(ctx context.Context, st *RunState, cand, tree string) (Verdict, error) {
	st.Reviewed = cand
	why := ""
	for attempt := 1; attempt <= landAttempts; attempt++ {
		var err error
		var landed string
		landed, why, err = e.landOnce(ctx, st, cand)
		if err != nil {
			return Verdict{}, err
		}
		if landed != "" {
			st.Landed = landed
			e.markClosed(st, StateLanded, e.landedWhy(st))
			if err := e.Store.Settle(st); err != nil {
				return Verdict{}, err
			}
			return Verdict{Done: true, Text: fmt.Sprintf("Landed on %s as %s. Nothing is left to do: end now.\n", st.Branch, short(landed))}, nil
		}
		if attempt < landAttempts {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}
	st.RetryLanding = true
	if err := e.closeOn(ctx, st, cand, StateKept, why); err != nil {
		return Verdict{}, err
	}
	return Verdict{Done: true, Text: fmt.Sprintf("The work passed, but %s. It stays on %s, and the next apply tries again. End now.\n", why, st.Kept)}, nil
}

func (e *Engine) landedWhy(st *RunState) string {
	return "the checks passed and the reviewer found nothing wrong"
}

// landOnce tries to land once. It returns the landing commit, or why not.
func (e *Engine) landOnce(ctx context.Context, st *RunState, cand string) (string, string, error) {
	tip, err := e.Git.Tip(ctx, st.Branch)
	if err != nil {
		return "", "", err
	}
	if tip == "" {
		return "", fmt.Sprintf("the branch %s no longer exists", st.Branch), nil
	}
	// A branch the owner moved back must not get what they dropped back by landing.
	if ok, err := e.Git.Contains(ctx, tip, st.Start); err != nil {
		return "", "", err
	} else if !ok {
		return "", fmt.Sprintf("%s was rewritten to %s, which no longer holds the run's start", st.Branch, short(tip)), nil
	}
	// A landing that was killed after it moved the branch has nothing left to do.
	// Work that changed nothing is on the branch from the start, and still lands
	// its Result.
	if cand != st.Start {
		if ok, err := e.Git.Contains(ctx, tip, cand); err != nil {
			return "", "", err
		} else if ok {
			return tip, "", nil
		}
	}
	state := cand
	moved := false
	if ok, err := e.Git.Contains(ctx, cand, tip); err != nil {
		return "", "", err
	} else if !ok {
		moved = true
		merged, conflicts, err := e.Git.Merge(ctx, tip, cand, fmt.Sprintf("Markitect run %s: merge onto %s", st.Run, st.Branch))
		if err != nil {
			return "", "", err
		}
		if len(conflicts) > 0 {
			return "", fmt.Sprintf("%s moved to %s and the merge conflicts in %s", st.Branch, short(tip), strings.Join(conflicts, ", ")), nil
		}
		why, err := e.checkMerged(ctx, merged)
		if err != nil {
			return "", "", err
		}
		if why != "" {
			return "", fmt.Sprintf("%s moved to %s and %s", st.Branch, short(tip), why), nil
		}
		state = merged
	}
	r := e.result(ctx, st, StateLanded, e.landedWhy(st), nil)
	r.Outcome = Landed
	r.Work = cand
	if moved {
		r.Why += fmt.Sprintf("; %s had moved to %s, and the clean merge passed the checks", st.Branch, short(tip))
	}
	replaced, err := e.covered(ctx, r)
	if err != nil {
		replaced = nil
	}
	r.Replaced = sortedKeys(replaced)
	// The count includes this landing; a landed apply --full starts it again.
	r.FullCheck = FullCheck{}
	if !r.Full {
		r.FullCheck = e.fullCheck(ctx, st.Branch, state)
		if r.FullCheck.Note == "" {
			r.FullCheck.Landed++
			r.FullCheck.Due = r.FullCheck.Landed >= FullCheckEvery
		}
	}
	commit, err := e.Git.Write(ctx, state, map[string][]byte{ResultFile(st.Run): Page(r)}, resultMessage(r, cand))
	if err != nil {
		return "", "", err
	}
	if err := e.Git.Advance(ctx, st.Branch, commit, tip); err != nil {
		return "", err.Error(), nil
	}
	for _, branch := range r.Replaced {
		_ = e.Git.DeleteBranch(ctx, branch)
	}
	return commit, "", nil
}

// checkMerged says why a merged state cannot land, or "": the spec must still
// compile. Later spec changes are not realized yet, so errors of the realization
// are expected there.
func (e *Engine) checkMerged(ctx context.Context, merged string) (string, error) {
	p, err := project.Load(e.Git.State(ctx, merged))
	if err != nil {
		return "", err
	}
	if errs := p.SpecErrors(); len(errs) > 0 {
		return fmt.Sprintf("the merged state has %d errors in the spec; the first: %s", len(errs), errs[0].String()), nil
	}
	return "", nil
}

// closeOn ends a Run that passed but cannot land.
func (e *Engine) closeOn(ctx context.Context, st *RunState, cand, state, why string) error {
	r := e.result(ctx, st, state, why, nil)
	r.Work = cand
	top, err := e.Git.Write(ctx, cand, map[string][]byte{ResultFile(st.Run): Page(r)}, resultMessage(r, ""))
	if err != nil {
		return err
	}
	st.Kept = RunBranchPrefix + st.Run
	if err := e.Git.SetBranch(ctx, st.Kept, top); err != nil {
		return err
	}
	e.markClosed(st, state, why)
	return e.Store.Settle(st)
}

// retryLandings tries once more to land the kept work of this branch, and returns
// the new tip when one landed. A second failure leaves the work on its branch: the
// base has not moved, so a fresh Run does the work again and nothing is lost.
func (e *Engine) retryLandings(ctx context.Context, runs []*RunState, branch, start string) string {
	tip := ""
	for _, st := range runs {
		if st.Open || !st.RetryLanding || st.Branch != branch {
			continue
		}
		kept, err := e.Git.Tip(ctx, st.Kept)
		if err != nil || kept == "" {
			continue
		}
		c, err := e.Git.Commit(ctx, kept)
		if err != nil || len(c.Parents) == 0 {
			continue
		}
		landed, _, err := e.landOnce(ctx, st, c.Parents[0])
		_ = e.Store.Update(st.Run, func(s *RunState) error {
			s.RetryLanding = false
			if err == nil && landed != "" {
				s.Landed, s.State, s.Why, s.Told = landed, StateLanded, e.landedWhy(st), false
			}
			return nil
		})
		if err == nil && landed != "" {
			_ = e.Git.DeleteBranch(ctx, st.Kept)
			tip = landed
		}
	}
	return tip
}

// result is the Result of a Run as it stands. The tool writes it from what
// happened, never an agent.
func (e *Engine) result(ctx context.Context, st *RunState, state, why string, decisions []OwnerDecision) *Result {
	r := &Result{Format: ResultFormat, Run: st.Run, Command: st.Command, Outcome: NotLanded, State: state, Why: why, Base: st.Base, Start: st.Start, Work: st.Reviewed,
		Target: st.Branch, Full: st.Full, OwnerDecisions: decisions, Repairs: st.Repairs, SentBack: st.SentBack, MissingLinks: st.MissingFor, ImpactNotes: st.Notes,
		Began: st.Began, Ended: e.now(), impact: st.Impact, Failure: st.Failure}
	if state != StateLanded {
		r.Branch = RunBranchPrefix + st.Run
	}
	r.FullCheck = e.fullCheck(ctx, st.Branch, st.Start)
	for _, it := range st.Items {
		a := st.Answers[it.Key]
		r.Items = append(r.Items, ResultItem{Item: it.Key, Kind: it.Kind, Definition: it.Definition, Look: it.Look, Answer: a.Word, Pointers: a.Pointers, Explanation: a.Explanation})
	}
	sort.SliceStable(r.Items, func(i, j int) bool { return r.Items[i].Item < r.Items[j].Item })
	return r
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
