package run

import (
	"fmt"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// prepareReview binds a review Pass to one immutable candidate: the sealed
// stand of the preceding Pass, captured after every writer stopped, is
// materialized read-only, passes the Model check and the File check, and the
// required checks run on exactly that tree. Failed checks become genuine
// findings of the Orchestrator; earlier ones held only for their candidate.
func (e *engine) prepareReview(p *pass) bool {
	id := WorkspaceID(fmt.Sprintf("%s-%s-c", e.cfg.RunID, p.id))
	volume, ok := e.materialize(id, p.tree, nil)
	if !ok {
		e.closeVolume(id)
		return false
	}
	e.mu.Lock()
	p.candidate, p.volume = id, volume
	_ = e.record(Event{Kind: "candidate", Pass: p.id, Data: map[string]any{"tree": p.tree.Digest().String(), "workspace": id, "volume": volume, "entries": p.tree.EntryCount()}})
	for _, f := range e.findings {
		if f.RaisedBy == MarkitectFinding && !f.Closed {
			f.Closed, f.ClosedPass, f.ClosedReason = true, p.id, "a check finding holds for exactly the candidate it was observed on"
			_ = e.record(Event{Kind: "finding.closed", Pass: p.id, Manager: f.Owner, Data: *f})
		}
	}
	e.mu.Unlock()
	if p.spec.Assesses {
		// The assessment judges the model of a stand whose entry checks passed;
		// the gates and checks judge candidates and artifacts.
		return true
	}
	if !e.gate(p, p.tree, "on the candidate") {
		return false
	}
	results := e.runChecks(p, p.tree)
	if e.ctx.Err() != nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	p.checks = results
	for i, r := range results {
		_ = e.record(Event{Kind: "check", Pass: p.id, Data: r})
		if r.Passed {
			continue
		}
		chk := e.cfg.Checks[i]
		observed := fmt.Sprintf("exit code %d", r.ExitCode)
		if r.Reason != "" {
			observed += ", " + r.Reason
		}
		if r.Output != "" {
			observed += "\n" + r.Output
		}
		e.addFindingLocked(Finding{Owner: chk.Owner, RaisedBy: MarkitectFinding, Obligation: fmt.Sprintf("Required check %q (required by %s) passes on the candidate", chk.Name, chk.RequiredBy), Observed: observed, Blocking: true, Correction: p.spec.defaultCorrection()})
	}
	return true
}

func (e *engine) runChecks(p *pass, tree snapshot.Tree) []CheckResult {
	results := make([]CheckResult, len(e.cfg.Checks))
	var wg sync.WaitGroup
	for i, c := range e.cfg.Checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = e.runCheck(p, tree, c)
		}()
	}
	wg.Wait()
	return results
}

func (e *engine) runCheck(p *pass, tree snapshot.Tree, c Check) CheckResult {
	req := c.Request
	req.ID, req.Tree = CheckID(fmt.Sprintf("%s-%s-%s", e.cfg.RunID, p.id, c.Name)), tree
	rec, err := e.cfg.Runner.Run(e.ctx, req)
	res := CheckResult{Name: c.Name, Passed: err == nil && rec.Passed(req), ExitCode: rec.ExitCode, Output: tail(rec.Output, outputTail)}
	switch {
	case err != nil:
		// A check that cannot run on a candidate, such as a missing check
		// directory, is a failed check, not a failed run.
		res.Reason = "the check could not run: " + err.Error()
	case res.Passed:
	case rec.TimedOut:
		res.Reason = "timed out"
	case rec.Uncertain:
		res.Reason = "the outcome is uncertain: " + rec.Reason
	case !rec.Started || !rec.Exited:
		res.Reason = "the check did not run to an exit: " + rec.Reason
	case rec.ExitCode == 0:
		res.Reason = "the receipt does not match the request"
	}
	return res
}
