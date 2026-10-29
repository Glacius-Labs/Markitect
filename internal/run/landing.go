package run

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// landAttempts bounds how often landing starts again when the target branch
// moves while the Run lands, as a merge queue does.
const landAttempts = 3

// ResultFile is where a Run's Result page lives in the repository.
func ResultFile(run string) string { return path.Join(project.ResultsDir, run+".md") }

// land lands Root's approved work as in a merge queue (concept 2.0 §6.5):
// unchanged target, fast-forward; moved target, merge, but only when the
// merge is clean, compiles, passes the required checks and the incoming
// changes touch nothing in the Run's impact. The host compiles every state it
// lands (§6.3). Otherwise the work stays on its Run branch and the Result
// says why.
func (e *Engine) land(ctx context.Context, r *Result, s *schedule, remote Remote, release func(*[]string)) {
	// A Run that does not land releases its remote and its lock entry
	// before it writes its page, so the page says what they left.
	keepOn := func(ctx context.Context, r *Result, commit, old, state, why string, err error) {
		release(&r.Cleanup)
		e.keepOn(ctx, r, commit, old, state, why, err)
	}
	root := s.nodes[model.RootPackage]
	cand := root.approval
	r.Branch = RunBranchPrefix + r.Run
	// The approved work is safe on its Run branch before landing starts.
	if err := e.Git.Push(ctx, r.Branch, cand, ""); err != nil {
		r.State, r.Why = StateIncomplete, fmt.Sprintf("the approved work could not be pushed to %s: %v", r.Branch, err)
		return
	}
	pushed := cand
	for _, n := range s.order {
		if ok, err := e.Git.Contains(ctx, cand, n.approval); err != nil || !ok {
			keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("Root's approved work does not contain %s's approval %s", n.name, short(n.approval)), err)
			return
		}
	}
	if why, err := e.credentialScan(ctx, s, cand); err != nil || why != "" {
		keepOn(ctx, r, cand, pushed, StateKept, why, err)
		return
	}
	if why, err := e.compile(ctx, cand); err != nil || why != "" {
		keepOn(ctx, r, cand, pushed, StateKept, "Root's approved work "+why, err)
		return
	}
	for attempt := 1; attempt <= landAttempts; attempt++ {
		tip, err := e.Git.Fetch(ctx, r.Target)
		if err != nil {
			keepOn(ctx, r, cand, pushed, StateIncomplete, "", fmt.Errorf("reading the target branch: %w", err))
			return
		}
		state := cand
		if tip != r.Start {
			// Until the merged state is checked, the checks named are those
			// of Root's approved work, which is not what would land.
			if len(s.plan.Checks) > 0 {
				r.RequiredChecks.Note = fmt.Sprintf("%s moved to %s, and no merged state was checked", r.Target, short(tip))
			}
			if ok, err := e.Git.Contains(ctx, tip, r.Start); err != nil || !ok {
				keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("%s was rewritten to %s, which no longer holds the Run's start", r.Target, short(tip)), err)
				return
			}
			why, err := e.incoming(ctx, s, r.Start, tip, cand)
			if err != nil || why != "" {
				keepOn(ctx, r, cand, pushed, StateKept, why, err)
				return
			}
			merged, conflicts, err := e.Git.Merge(ctx, tip, cand, fmt.Sprintf("Markitect Run %s: merge onto %s", r.Run, r.Target))
			if err != nil || len(conflicts) > 0 {
				keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("%s moved to %s and the merge conflicts in %s", r.Target, short(tip), strings.Join(conflicts, ", ")), err)
				return
			}
			if why, err := e.compile(ctx, merged); err != nil || why != "" {
				keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("%s moved to %s and the merged state %s", r.Target, short(tip), why), err)
				return
			}
			if len(s.plan.Checks) > 0 {
				var b bytes.Buffer
				if err := e.Git.Bundle(ctx, merged, &b); err != nil {
					keepOn(ctx, r, cand, pushed, StateIncomplete, "", err)
					return
				}
				results, err := remote.Check(ctx, &b, merged)
				r.RequiredChecks = RequiredChecks{Commit: merged, Checks: results, WholeRepository: true, Ran: err == nil && len(results) == len(s.plan.Checks) && allPassed(results)}
				if err != nil {
					r.RequiredChecks.Note = fmt.Sprintf("the checks on the merged state could not run: %v", err)
					keepOn(ctx, r, cand, pushed, StateKept, "the required checks could not run on the merged state", nil)
					return
				}
				if !r.RequiredChecks.Ran {
					keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("%s moved to %s and the required checks fail on the merged state", r.Target, short(tip)), nil)
					return
				}
			}
			state = merged
		}
		r.Outcome, r.State = Landed, StateLanded
		r.Why = "Root's checker approved the work"
		if tip != r.Start {
			r.Why += fmt.Sprintf("; %s had moved to %s, and the clean merge passed the required checks", r.Target, short(tip))
		}
		// The landed page names no Run branch: it is deleted on landing,
		// with the kept Run branches this landing covers.
		r.Branch = ""
		replaced, err := e.covered(ctx, r)
		if err != nil {
			replaced = nil
			r.Cleanup = append(r.Cleanup, fmt.Sprintf("the kept Run branches could not be read, so none that this landing covers is deleted: %v", err))
		}
		r.Replaced = sortedKeys(replaced)
		page := e.stamp(r)
		commit, err := e.Git.Write(ctx, state, map[string][]byte{ResultFile(r.Run): Page(page)}, resultMessage(r, cand))
		if err != nil {
			keepOn(ctx, r, cand, pushed, StateIncomplete, "", err)
			return
		}
		if err := e.Git.Push(ctx, r.Target, commit, tip); err != nil {
			// Whether the push took effect is read again, on a fresh
			// deadline: a push can fail after the target took it.
			// Others may have landed on top of it before it is read, so
			// it landed when the new tip holds it.
			cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
			now, ferr := e.Git.Fetch(cctx, r.Target)
			holds := now == commit
			if ferr == nil && !holds && now != tip {
				holds, ferr = e.Git.Contains(cctx, now, commit)
			}
			cancel()
			switch {
			case ferr != nil:
				keepOn(ctx, r, cand, pushed, StateIncomplete, fmt.Sprintf("the landing push failed (%v) and %s could not be read again, so whether it landed is uncertain", err, r.Target), ferr)
				return
			case holds:
				// It landed.
			case now != tip:
				// The target moved again while landing: start over.
				continue
			default:
				keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("the target refused the landing push: %v", err), nil)
				return
			}
		}
		// The landed page cannot say what follows it; the returned Result
		// does, under AfterLanding.
		r.Landed, r.AfterLanding = commit, []string{}
		if err := e.Git.Push(ctx, RunBranchPrefix+r.Run, "", pushed); err != nil {
			r.AfterLanding = append(r.AfterLanding, fmt.Sprintf("the landed Run's branch %s could not be deleted: %v", RunBranchPrefix+r.Run, err))
		}
		for _, branch := range r.Replaced {
			if err := e.Git.Push(ctx, branch, "", replaced[branch]); err != nil {
				r.AfterLanding = append(r.AfterLanding, fmt.Sprintf("the covered Run branch %s could not be deleted: %v", branch, err))
			}
		}
		release(&r.AfterLanding)
		return
	}
	keepOn(ctx, r, cand, pushed, StateKept, fmt.Sprintf("%s moved while landing, %d times in a row", r.Target, landAttempts), nil)
}

// keep leaves a Run that does not land on its Run branch: the Result page on
// top of Root's newest state, or of the starting commit.
func (e *Engine) keep(ctx context.Context, r *Result, s *schedule) {
	r.Branch = RunBranchPrefix + r.Run
	top := r.Start
	if tip := s.tips[runremote.RootBranch]; tip != "" && r.Command == "apply" {
		top = tip
	}
	if r.Command == "apply" {
		for _, n := range s.order {
			if n.approval == "" {
				continue
			}
			if ok, err := e.Git.Contains(ctx, top, n.approval); err != nil || !ok {
				r.Unkept = append(r.Unkept, fmt.Sprintf("%s's approved work %s goes with the Run's remote; the Run branch does not hold it", n.name, n.approval))
			}
		}
	}
	e.keepOn(ctx, r, top, "", "", "", nil)
}

// keepOn writes the Result page on top of commit and pushes it to the Run
// branch, which is at old (empty when it does not exist yet). A non-empty
// state and why replace the Result's.
func (e *Engine) keepOn(ctx context.Context, r *Result, commit, old, state, why string, err error) {
	// Keeping the Result is cleanup: it still runs, bounded on its own,
	// when the Run's deadline has passed.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
	defer cancel()
	r.Outcome, r.Branch = NotLanded, RunBranchPrefix+r.Run
	if err != nil {
		state, why = StateIncomplete, strings.TrimSpace(why+" "+err.Error())
	}
	if state != "" {
		r.State, r.Why = state, why
	}
	page := e.stamp(r)
	top, werr := e.Git.Write(ctx, commit, map[string][]byte{ResultFile(r.Run): Page(page)}, resultMessage(r, ""))
	if werr == nil {
		werr = e.Git.Push(ctx, r.Branch, top, old)
	}
	if werr != nil {
		r.Branch = ""
		r.Cleanup = append(r.Cleanup, fmt.Sprintf("the Result could not be kept on %s%s: %v", RunBranchPrefix, r.Run, werr))
	}
}

// incoming says why changes that reached the target after the Run started
// touch the Run's impact, or "" when they do not.
func (e *Engine) incoming(ctx context.Context, s *schedule, start, tip, cand string) (string, error) {
	in, err := e.Git.Changed(ctx, start, tip)
	if err != nil {
		return "", err
	}
	next, err := project.Load(e.Git.State(ctx, tip))
	if err != nil {
		return "", err
	}
	if len(next.Errors) > 0 {
		return fmt.Sprintf("%s moved to %s, which has %d compile errors", s.req.Target, short(tip), len(next.Errors)), nil
	}
	if !reflect.DeepEqual(next.Checks, s.cur.Checks) {
		return fmt.Sprintf("%s moved to %s, which changes the required checks", s.req.Target, short(tip)), nil
	}
	var files []analysis.Change
	for _, c := range in {
		switch {
		case strings.HasPrefix(c.Path, project.ResultsDir+"/"):
		case c.Path == project.ManifestFile || c.Path == project.LockFile:
			return fmt.Sprintf("%s moved to %s, which changes %s", s.req.Target, short(tip), c.Path), nil
		case strings.HasPrefix(c.Path, project.Dir+"/"):
			if pkg, ok := s.plan.Pages[c.Path]; ok && s.nodes[pkg] != nil {
				return fmt.Sprintf("%s moved to %s, which changes the registry page %s of a working Manager", s.req.Target, short(tip), c.Path), nil
			}
		default:
			files = append(files, c)
		}
	}
	inImp := analysis.ComputeImpact(analysis.ImpactInput{Base: s.cur, Current: next, Changed: files})
	ours := map[string]bool{}
	for _, k := range impactKeys(s.imp) {
		ours[k] = true
	}
	mine, err := e.Git.Changed(ctx, start, cand)
	if err != nil {
		return "", err
	}
	for _, c := range mine {
		ours["file:"+c.Path] = true
	}
	var hits []string
	for _, c := range inImp.Definitions {
		if ours["def:"+c.Path] {
			hits = append(hits, "[["+c.Path+"]]")
		}
	}
	for _, it := range inImp.Reached {
		if ours["def:"+it.Definition] {
			hits = append(hits, "[["+it.Definition+"]]")
		}
	}
	for _, f := range inImp.ReachedFiles {
		if ours["file:"+f.File] {
			hits = append(hits, f.File)
		}
	}
	if len(hits) > 0 {
		return fmt.Sprintf("%s moved to %s, and its changes touch the Run's impact: %s; a follow-up Run starts from the new state", s.req.Target, short(tip), strings.Join(dedupe(hits), ", ")), nil
	}
	return "", nil
}

// compile says why a state does not compile, as `markitect check` reports
// it, or "" when it compiles.
func (e *Engine) compile(ctx context.Context, commit string) (string, error) {
	p, err := project.Load(e.Git.State(ctx, commit))
	if err != nil {
		return "", err
	}
	if len(p.Errors) > 0 {
		return fmt.Sprintf("does not compile: check finds %d compile errors at %s; the first: %s", len(p.Errors), short(commit), p.Errors.Sorted()[0].String()), nil
	}
	return "", nil
}

// credentialFiles are file names that hold a provider account's login.
var credentialFiles = []string{".credentials.json", "auth.json"}

// credentialScan refuses landing work that holds one of the Run's own Git
// tokens or a provider login file anywhere in its history, since landing
// pushes every commit; the hooks look for neither.
func (e *Engine) credentialScan(ctx context.Context, s *schedule, cand string) (string, error) {
	added, paths, err := e.Git.History(ctx, s.plan.Base, cand)
	if err != nil {
		return "", err
	}
	for _, roles := range s.creds {
		for _, c := range roles {
			if strings.Contains(added, c.token) {
				return fmt.Sprintf("the approved work holds the Git token of agent %s; it does not land", c.user), nil
			}
		}
	}
	for _, p := range paths {
		for _, name := range credentialFiles {
			if path.Base(p) == name {
				return fmt.Sprintf("the approved work's history touches %s, which looks like a provider login; it does not land", p), nil
			}
		}
	}
	return "", nil
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
