package run

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Verdict is what submit says. Done means the Run is closed and the session should
// end; otherwise Text is the next step.
type Verdict struct {
	Done bool
	Text string
}

// harnessPaths are the files of the harness, which a Run may not change: an agent
// cannot weaken its own rules in the work it submits.
var harnessPaths = []string{".claude/agents/markitect-", ".claude/skills/markitect-", ".agents/skills/markitect-"}

// Submit is the one door: it looks at the current state and does the first thing
// that applies. A slow one keeps its lock, so a second call only says it is still
// running, and a killed one can be run again.
func (e *Engine) Submit(ctx context.Context, dir string) (Verdict, error) {
	st, err := e.Current(ctx, dir)
	if err != nil {
		return Verdict{}, err
	}
	root, err := e.Git.Root(ctx, dir)
	if err != nil {
		return Verdict{}, err
	}
	if !st.Open {
		return Verdict{Done: true, Text: fmt.Sprintf("Run %s is closed: %s. %s. End now.\n", st.Run, st.State, capitalize(st.Why))}, nil
	}
	release, err := e.Store.TrySubmitLock(st.Run)
	if errors.Is(err, ErrBusy) {
		return Verdict{Text: "markitect submit is still running in another call. Wait for it, and do not start it again.\n"}, nil
	}
	if err != nil {
		return Verdict{}, err
	}
	defer release()
	// The answers may have changed since the look above.
	if st, _, err = e.Store.Load(st.Run); err != nil {
		return Verdict{}, err
	}
	st.Submits++
	st.Blocks = 0
	if st.Submits > maxSubmits {
		why := fmt.Sprintf("markitect submit was called %d times, the most a run gets", maxSubmits)
		return e.closeVerdict(ctx, st, StateRefused, why, nil)
	}
	if e.now().After(st.Deadline) {
		return e.closeVerdict(ctx, st, StateIncomplete, "the run passed its deadline", nil)
	}
	if st.Command == "verify" {
		return e.submitVerify(ctx, st)
	}
	return e.submitApply(ctx, st, root)
}

func (e *Engine) submitApply(ctx context.Context, st *RunState, root string) (Verdict, error) {
	cand, err := e.Git.Candidate(ctx, root, fmt.Sprintf("Markitect run %s: the implementer's work", st.Run))
	if err != nil {
		return Verdict{}, err
	}
	commit, err := e.Git.Commit(ctx, cand)
	if err != nil {
		return Verdict{}, err
	}
	// Row 1: what the Run may not change.
	if refusals, err := e.protected(ctx, st.Start, cand); err != nil {
		return Verdict{}, err
	} else if len(refusals) > 0 {
		return e.refuse(st, "Put these back, or run markitect stop \"why\" if the spec cannot be met:", refusals)
	}
	p, err := project.Load(e.Git.State(ctx, cand))
	if err != nil {
		return Verdict{}, err
	}
	// Row 2: a reviewed state, and only new unlisted files on top of it.
	if left, err := e.leftovers(ctx, st, cand, p); err != nil {
		return Verdict{}, err
	} else if len(left) > 0 {
		return e.refuse(st, "These files are new since the review and not listed. List them on the registry if they are the work, otherwise remove them:", left)
	}
	// Row 3: check, of the spec and the realization.
	if len(p.Errors) > 0 {
		return e.refuse(st, "markitect check fails on your work:", errorLines(p))
	}
	// Rows 4 and 5: the required checks, once per state when they pass.
	if !st.Passed[commit.Tree] {
		results, err := e.runChecks(ctx, cand, p.Checks)
		if err != nil {
			return Verdict{}, err
		}
		st.Checks, st.ChecksOn = results, cand
		var failed []string
		for _, res := range results {
			if res.NotRun {
				return e.closeVerdict(ctx, st, StateIncomplete, fmt.Sprintf("the required check %s could not start (exit %d): %s", res.Name, res.Exit, tail(res.Output)), nil)
			}
			if !res.Passed {
				failed = append(failed, fmt.Sprintf("the required check %s failed:\n%s", res.Name, tail(res.Output)))
			}
		}
		if len(failed) > 0 {
			return e.refuse(st, "The required checks do not pass in a clean copy of your work:", failed)
		}
		if st.Passed == nil {
			st.Passed = map[string]bool{}
		}
		st.Passed[commit.Tree] = true
	}
	// Row 6: no complete review for this state. A review that finished on an
	// earlier state still counts: an implementer that reads its report and repairs
	// before calling submit has used a round like any other, and what the review
	// settled stands.
	if st.Review == nil || st.Review.Tree != commit.Tree {
		if prev := st.Review; prev != nil && prev.Complete() {
			st.Reviewed = prev.Commit
			wrong, decisions := e.verdicts(st)
			if len(decisions) > 0 {
				return e.closeVerdict(ctx, st, StateSpecProblem, "a reviewer found a spec problem: the spec needs the owner", decisions)
			}
			if len(wrong) > 0 && !prev.Counted {
				if st.Repairs == repairRounds {
					return e.closeVerdict(ctx, st, StateRefused, "the repair rounds ran out, and these stay open: "+strings.Join(wrong, "; "), nil)
				}
				prev.Counted = true
				st.Repairs++
				st.SentBack = append(st.SentBack, strings.Join(wrong, "; "))
			}
		}
		if err := e.Git.Adopt(ctx, root, cand); err != nil {
			return Verdict{}, err
		}
		if err := e.startReview(ctx, st, cand, commit.Tree, p); err != nil {
			return Verdict{}, err
		}
		if len(st.Review.Items) > 0 {
			if err := e.Store.Settle(st); err != nil {
				return Verdict{}, err
			}
			return Verdict{Text: reviewerStep(st)}, nil
		}
	}
	rev := st.Review
	if missing := rev.Missing(); len(missing) > 0 {
		if err := e.Store.Settle(st); err != nil {
			return Verdict{}, err
		}
		return Verdict{Text: fmt.Sprintf("The review is not complete: %d items have no answer. Start the %s agent again with the message \"Review run %s.\"\n", len(missing), ReviewerAgent, st.Run)}, nil
	}
	st.Reviewed = rev.Commit
	wrong, decisions := e.verdicts(st)
	// Row 7: a spec problem ends the Run before a round is counted.
	if len(decisions) > 0 {
		return e.closeVerdict(ctx, st, StateSpecProblem, "a reviewer found a spec problem: the spec needs the owner", decisions)
	}
	// Row 8: wrong answers go back, one round per reviewed state.
	if len(wrong) > 0 {
		if !rev.Counted {
			if st.Repairs == repairRounds {
				return e.closeVerdict(ctx, st, StateRefused, "the repair rounds ran out, and these stay open: "+strings.Join(wrong, "; "), nil)
			}
			rev.Counted = true
			st.Repairs++
			st.SentBack = append(st.SentBack, strings.Join(wrong, "; "))
		}
		return e.refuse(st, fmt.Sprintf("The reviewer found these wrong (repair round %d of %d). Fix them and call markitect submit again:", st.Repairs, repairRounds), wrong)
	}
	// Row 9: nothing left.
	return e.land(ctx, st, cand, commit.Tree)
}

// verdicts reads what the latest answers say: what is wrong, and what only the
// owner can settle.
func (e *Engine) verdicts(st *RunState) ([]string, []OwnerDecision) {
	var wrong []string
	var decisions []OwnerDecision
	for _, it := range st.Items {
		switch a := st.Answers[it.Key]; a.Word {
		case AnswerSpecProblem:
			decisions = append(decisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: "Settle it in the spec and run apply again."})
		case AnswerWrong:
			wrong = append(wrong, fmt.Sprintf("%s: %s %s", it.Key, a.Explanation, pointers(a.Pointers)))
		}
	}
	return wrong, decisions
}

// protected says what changed that a Run may not change: the owner's part of
// Markitect's folder, the harness, and links that leave the repository.
func (e *Engine) protected(ctx context.Context, from, commit string) ([]string, error) {
	changed, err := e.Git.Changed(ctx, from, commit)
	if err != nil {
		return nil, err
	}
	links, err := e.Git.Links(ctx, commit)
	if err != nil {
		return nil, err
	}
	var refusals []string
	for _, f := range changed {
		// Without regard to case: a variant of the folder's name is the same
		// folder in a Windows or macOS checkout.
		lower := strings.ToLower(f.Path)
		if strings.HasPrefix(lower, project.Dir+"/") && !registryPage(f.Path) {
			refusals = append(refusals, fmt.Sprintf("%s changed: everything under %s/ except the registry pages is the owner's", f.Path, project.Dir))
		}
		for _, h := range harnessPaths {
			if strings.HasPrefix(lower, h) {
				refusals = append(refusals, fmt.Sprintf("%s changed: the harness is the owner's", f.Path))
			}
		}
		if to, ok := links[f.Path]; ok && leavesTree(f.Path, to) {
			refusals = append(refusals, fmt.Sprintf("%s is a link that leaves the repository, to %s", f.Path, to))
		}
	}
	return refusals, nil
}

// leftovers lists new files that are all a state adds to a reviewed one, when
// none of them is listed: what a reviewer's test run can leave behind.
func (e *Engine) leftovers(ctx context.Context, st *RunState, cand string, p *project.Project) ([]string, error) {
	if st.Review == nil {
		return nil, nil
	}
	changed, err := e.Git.Changed(ctx, st.Review.Commit, cand)
	if err != nil {
		return nil, err
	}
	unlisted := map[string]bool{}
	for _, d := range p.Errors {
		if d.Code == project.CodeUnregistered {
			unlisted[d.File] = true
		}
	}
	var out []string
	for _, f := range changed {
		if f.Change != analysis.Added || !unlisted[f.Path] {
			return nil, nil
		}
		out = append(out, f.Path)
	}
	return out, nil
}

// startReview rebuilds the list from the work and says what this review must
// answer: everything on the first; later, what has no answer or is open, and
// whatever the new diff touched.
func (e *Engine) startReview(ctx context.Context, st *RunState, cand, tree string, p *project.Project) error {
	before, err := project.Load(e.Git.State(ctx, st.Base))
	if err != nil {
		return err
	}
	imp, err := e.impact(ctx, st.Base, cand, before, p, st.Findings, st.Full)
	if err != nil {
		return err
	}
	st.Items = mergeItems(st.Items, checklist(imp, before, p, st.Findings))
	st.Impact = dedupe(append(st.Impact, impactKeys(imp, st.Items)...))
	rev := &Review{Commit: cand, Tree: tree, From: st.Reviewed, Answered: map[string]Answer{}}
	if st.Reviewed == "" {
		rev.Items = st.Items
		st.Review = rev
		return nil
	}
	changed, err := e.Git.Changed(ctx, st.Reviewed, cand)
	if err != nil {
		return err
	}
	touched := map[string]bool{}
	for _, f := range changed {
		touched[f.Path] = true
	}
	for _, it := range st.Items {
		a, answered := st.Answers[it.Key]
		again := !answered || a.Word == AnswerWrong
		for _, f := range it.Look {
			again = again || touched[f]
		}
		for _, ptr := range a.Pointers {
			again = again || touched[ptr.File]
		}
		if a.Word == AnswerWrong {
			rev.Earlier = append(rev.Earlier, fmt.Sprintf("%s: %s", it.Key, a.Explanation))
		}
		if again {
			rev.Items = append(rev.Items, it)
		}
	}
	st.Review = rev
	return nil
}

func reviewerStep(st *RunState) string {
	return fmt.Sprintf(`The work passes the checks and is committed. Now start the %s agent with exactly this message:

    Review run %s.

Wait for it to finish, then call markitect submit again. It has %d items to answer.
`, ReviewerAgent, st.Run, len(st.Review.Items))
}

// submitVerify closes a verify: it only ever reports. Items without an answer are
// listed; otherwise the required checks run once on the commit, what fails
// becomes a Finding, and the Findings and the Result stay on the Run's branch.
func (e *Engine) submitVerify(ctx context.Context, st *RunState) (Verdict, error) {
	rev := st.Review
	if rev == nil {
		return Verdict{}, fmt.Errorf("%w: run %s has nothing to review", ErrUsage, st.Run)
	}
	if missing := rev.Missing(); len(missing) > 0 {
		if err := e.Store.Settle(st); err != nil {
			return Verdict{}, err
		}
		return Verdict{Text: fmt.Sprintf("%d items have no answer yet:\n\n- %s\n", len(missing), strings.Join(missing, "\n- "))}, nil
	}
	p, err := project.Load(e.Git.State(ctx, st.Start))
	if err != nil {
		return Verdict{}, err
	}
	results, err := e.runChecks(ctx, st.Start, p.Checks)
	if err != nil {
		return Verdict{}, err
	}
	st.Checks, st.ChecksOn = results, st.Start
	var findings []ResultFinding
	var decisions []OwnerDecision
	// What does not hold and names no Definition stays in the Result, not in the
	// Findings, which each name one.
	var unsettled []string
	for _, it := range st.Items {
		a := st.Answers[it.Key]
		switch a.Word {
		case AnswerWrong:
			if it.Definition != "" {
				findings = append(findings, ResultFinding{Definition: it.Definition, Observed: strings.TrimSpace(a.Explanation + " " + pointers(a.Pointers)), Expected: "what the Definition says"})
			} else {
				unsettled = append(unsettled, it.Key)
			}
		case AnswerSpecProblem:
			decisions = append(decisions, OwnerDecision{Matter: fmt.Sprintf("%s: %s", it.Key, a.Explanation), Recommendation: "Settle it in the spec."})
		}
	}
	for _, d := range p.RealizationErrors() {
		unsettled = append(unsettled, d.String())
	}
	for _, res := range results {
		switch {
		case res.NotRun:
			unsettled = append(unsettled, fmt.Sprintf("the required check %s could not start", res.Name))
		case !res.Passed:
			unsettled = append(unsettled, fmt.Sprintf("the required check %s failed", res.Name))
		}
	}
	state, why := StateVerified, "every item answered without a Finding, and the required checks pass"
	var found []string
	if n := len(findings); n > 0 {
		found = append(found, fmt.Sprintf("%d Findings", n))
	}
	if n := len(decisions); n > 0 {
		found = append(found, fmt.Sprintf("%d decisions for the owner", n))
	}
	if n := len(unsettled); n > 0 {
		found = append(found, fmt.Sprintf("%d things that do not hold: %s", n, strings.Join(unsettled, "; ")))
	}
	if len(found) > 0 {
		state, why = StateFindings, "verify found "+strings.Join(found, " and ")
	}
	return e.closeVerify(ctx, st, state, why, findings, decisions, unsettled)
}

// FindingsFile is where a verify's Findings page lives on its branch, the input of
// apply --findings.
func FindingsFile(run string) string { return path.Join(project.ResultsDir, run+"-findings.md") }

func (e *Engine) closeVerify(ctx context.Context, st *RunState, state, why string, findings []ResultFinding, decisions []OwnerDecision, unsettled []string) (Verdict, error) {
	r := e.result(ctx, st, state, why, decisions)
	r.Findings, r.Unsettled = findings, unsettled
	if isFullCheck(r) {
		r.FullCheck = FullCheck{Since: r.Start}
	}
	files := map[string][]byte{ResultFile(st.Run): Page(r)}
	if len(findings) > 0 {
		files[FindingsFile(st.Run)] = FindingsPage(r)
	}
	top, err := e.Git.Write(ctx, st.Start, files, resultMessage(r, ""))
	if err != nil {
		return Verdict{}, err
	}
	st.Kept = RunBranchPrefix + st.Run
	if err := e.Git.SetBranch(ctx, st.Kept, top); err != nil {
		return Verdict{}, err
	}
	e.markClosed(st, state, why)
	if err := e.Store.Settle(st); err != nil {
		return Verdict{}, err
	}
	return Verdict{Done: true, Text: fmt.Sprintf("Run %s closed: %s. Its Result is on %s. End now.\n", st.Run, state, st.Kept)}, nil
}

// refuse saves the Run and gives the agent its next step.
func (e *Engine) refuse(st *RunState, head string, lines []string) (Verdict, error) {
	if err := e.Store.Settle(st); err != nil {
		return Verdict{}, err
	}
	var b strings.Builder
	b.WriteString(head)
	b.WriteString("\n\n")
	for _, l := range lines {
		fmt.Fprintf(&b, "- %s\n", l)
	}
	return Verdict{Text: b.String()}, nil
}

func errorLines(p *project.Project) []string {
	var out []string
	for i, d := range p.Errors {
		if i == maxRefusals {
			out = append(out, fmt.Sprintf("and %d more errors", len(p.Errors)-maxRefusals))
			break
		}
		out = append(out, d.String())
	}
	return out
}

func (e *Engine) closeVerdict(ctx context.Context, st *RunState, state, why string, decisions []OwnerDecision) (Verdict, error) {
	if err := e.close(ctx, st, state, why, decisions); err != nil {
		return Verdict{}, err
	}
	return Verdict{Done: true, Text: fmt.Sprintf("Run %s has ended without landing: %s. %s. Its work and Result are on %s. End now.\n", st.Run, state, capitalize(why), st.Kept)}, nil
}

// close ends a Run without landing: what the worktree holds is committed with a
// Result on the Run's own branch, and the Result says what is open.
func (e *Engine) close(ctx context.Context, st *RunState, state, why string, decisions []OwnerDecision) error {
	work := st.Start
	if st.Reviewed != "" {
		work = st.Reviewed
	}
	if root, err := e.Git.WorktreeOf(ctx, worktreeBranch(st.Run)); err == nil && root != "" {
		if cand, err := e.Git.Candidate(ctx, root, fmt.Sprintf("Markitect run %s: the work when it ended", st.Run)); err == nil {
			work = cand
		}
	}
	if st.Command == "verify" {
		work = st.Start
	}
	r := e.result(ctx, st, state, why, decisions)
	r.Work = work
	top, err := e.Git.Write(ctx, work, map[string][]byte{ResultFile(st.Run): Page(r)}, resultMessage(r, ""))
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

func (e *Engine) markClosed(st *RunState, state, why string) {
	st.Open, st.State, st.Why, st.Closed = false, state, why, e.now()
}

func registryPage(file string) bool {
	return file == project.RootPage || strings.HasPrefix(file, project.RegistryDir+"/")
}

// leavesTree says whether a symbolic link at file points outside the repository.
func leavesTree(file, to string) bool {
	if path.IsAbs(to) || strings.Contains(to, `\`) {
		return true
	}
	// A drive letter leaves the tree in a Windows checkout, where Git makes the
	// link as it stands.
	if len(to) > 1 && to[1] == ':' {
		return true
	}
	// A ".." after another part can pass through a link, where the text says
	// nothing about the way.
	inner := false
	for _, part := range strings.Split(to, "/") {
		switch {
		case part == ".." && inner:
			return true
		case part != ".." && part != "" && part != ".":
			inner = true
		}
	}
	target := path.Join(path.Dir(file), to)
	return target == ".." || strings.HasPrefix(target, "../")
}
