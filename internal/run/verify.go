package run

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// verifyRound runs one Verify round on an immutable candidate: structural
// accounting and the required checks first, then fresh Verify turns of all
// Managers with children before parents. It reports whether the round is green.
func (e *engine) verifyRound(n int, c candidate) bool {
	rs := &roundState{n: n, tree: c.tree, volume: c.volume, assessments: map[string]Assessment{}, assessed: map[string]bool{}}
	e.mu.Lock()
	for _, f := range e.findings {
		if f.RaisedBy == MarkitectFinding && !f.Closed {
			f.Closed, f.ClosedRound, f.ClosedReason = true, n, "a structural finding holds for exactly the candidate it was observed on"
			_ = e.record(Event{Kind: "finding.closed", Manager: f.Owner, Round: n, Data: *f})
		}
	}
	_ = e.record(Event{Kind: "round.start", Round: n, Data: map[string]any{"tree": c.tree.Digest().String(), "workspace": c.id}})
	e.mu.Unlock()

	structural := e.cfg.Account(c.tree)
	results := e.runChecks(n, c.tree)
	if e.ctx.Err() != nil {
		return e.endRound(rs, false)
	}
	e.mu.Lock()
	rs.checks = results
	for _, f := range structural {
		rs.structural = append(rs.structural, e.structuralLocked(f, n))
	}
	for _, f := range e.unsupportedFindings(c.unsupported) {
		rs.structural = append(rs.structural, e.addFindingLocked(f, n))
	}
	for i, r := range results {
		_ = e.record(Event{Kind: "check", Round: n, Data: r})
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
		rs.structural = append(rs.structural, e.addFindingLocked(Finding{Owner: chk.Owner, RaisedBy: MarkitectFinding, Obligation: fmt.Sprintf("Required check %q (required by %s) passes on the candidate", chk.Name, chk.RequiredBy), Observed: observed, Blocking: true}, n))
	}
	e.deliverLocked(n)
	e.mu.Unlock()

	done := map[string]chan struct{}{}
	for _, m := range e.team {
		done[m.Path] = make(chan struct{})
	}
	var wg sync.WaitGroup
	for _, m := range e.team {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(done[m.Path])
			for _, child := range m.Children {
				select {
				case <-done[child]:
				case <-e.ctx.Done():
					return
				}
			}
			e.verifyManager(m, rs)
		}()
	}
	wg.Wait()
	return e.endRound(rs, true)
}

// endRound decides whether the round is green and records it.
func (e *engine) endRound(rs *roundState, complete bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	green := complete && e.ctx.Err() == nil
	r := Round{Number: rs.n, Tree: rs.tree.Digest().String(), Checks: rs.checks}
	for _, m := range e.team {
		if rs.assessed[m.Path] {
			r.Assessments = append(r.Assessments, rs.assessments[m.Path])
		} else {
			green = false
		}
	}
	for _, c := range rs.checks {
		green = green && c.Passed
	}
	if len(rs.checks) != len(e.cfg.Checks) {
		green = false
	}
	for _, f := range e.findings {
		if f.Blocking && !f.Closed {
			green = false
		}
	}
	r.Green = green
	e.rounds = append(e.rounds, r)
	_ = e.record(Event{Kind: "round.end", Round: rs.n, Data: map[string]any{"green": green, "tree": r.Tree}})
	e.deliverLocked(rs.n)
	return green
}

func (e *engine) runChecks(n int, tree snapshot.Tree) []CheckResult {
	results := make([]CheckResult, len(e.cfg.Checks))
	var wg sync.WaitGroup
	for i, c := range e.cfg.Checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = e.runCheck(n, tree, c)
		}()
	}
	wg.Wait()
	return results
}

func (e *engine) runCheck(n int, tree snapshot.Tree, c Check) CheckResult {
	req := c.Request
	req.ID, req.Tree = CheckID(fmt.Sprintf("%s-r%d-%s", e.cfg.RunID, n, c.Name)), tree
	rec, err := e.cfg.Runner.Run(e.ctx, req)
	res := CheckResult{Name: c.Name, Passed: err == nil && rec.Passed(req), ExitCode: rec.ExitCode, Output: tail(rec.Output, outputTail)}
	switch {
	case err != nil:
		// A check that cannot run on a candidate, such as a missing check
		// directory before the first Execute, is a failed check, not a failed run.
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

// verifyManager runs the Manager's fresh Verify turn, once more when the first
// ended without a valid assessment.
func (e *engine) verifyManager(m *mgr, rs *roundState) {
	var last error
	for attempt := 1; attempt <= maxVerifyAttempts; attempt++ {
		if e.ctx.Err() != nil {
			return
		}
		e.mu.Lock()
		if !e.reserveTurnLocked(0) {
			e.mu.Unlock()
			e.halt("the agent turn budget ended")
			return
		}
		m.verifyUsed = true
		ts := &turnState{m: m, kind: VerifyTurn, id: fmt.Sprintf("%s-r%d-%s-v%d", e.cfg.RunID, rs.n, m.slug, attempt), round: rs.n}
		prompt := e.verifyPromptLocked(m, rs, attempt > 1)
		e.mu.Unlock()
		res, err := e.runTurn(ts, Turn{Profile: m.VerifyProfile, Workspace: rs.volume, ReadOnly: true, Home: m.verifyHome, Prompt: prompt, Tools: verifyTools()}, e.assessHandler(m, rs, ts))
		e.mu.Lock()
		e.applyReportLocked(m, rs, ts)
		reported := rs.assessed[m.Path]
		e.mu.Unlock()
		if !res.Stopped {
			return
		}
		if err != nil {
			last = err
		}
		if reported {
			return
		}
	}
	if e.ctx.Err() == nil {
		reason := fmt.Sprintf("manager %q did not report an assessment in round %d", m.Path, rs.n)
		if last != nil {
			reason += ": " + last.Error()
		}
		e.halt(reason)
	}
}

type assessment struct {
	Summary   string `json:"summary"`
	Satisfied *bool  `json:"satisfied"`
	Findings  []struct {
		Owner      string `json:"owner"`
		Obligation string `json:"obligation"`
		Observed   string `json:"observed"`
		Blocking   *bool  `json:"blocking"`
	} `json:"findings"`
	Closed []struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	} `json:"closed"`
}

// report is a valid assessment of one Verify turn, kept until the turn ends.
type report struct {
	summary   string
	satisfied bool
	raised    []Finding
	closing   map[string]string
}

// assessHandler serves the Verify tool. It keeps a valid report for the end of
// the turn, where a later valid report replaces an earlier one, and refuses an
// invalid one with the reason so the Agent can retry. Only the Manager's own
// findings decide validity, and no other turn changes them while this one runs.
func (e *engine) assessHandler(m *mgr, rs *roundState, ts *turnState) ToolHandler {
	return func(_ context.Context, name string, args json.RawMessage) ToolAnswer {
		if name != toolAssess {
			return refuse("Unknown tool %q; in Verify only %q is available.", name, toolAssess)
		}
		e.mu.Lock()
		defer e.mu.Unlock()
		var in assessment
		if err := decodeStrict(args, &in); err != nil {
			return refuse("%v", err)
		}
		summary, err := toolText("summary", in.Summary)
		if err != nil {
			return refuse("%v", err)
		}
		if in.Satisfied == nil {
			return refuse("satisfied is required.")
		}
		if len(in.Findings) > maxToolFindings || len(in.Closed) > maxToolFindings {
			return refuse("At most %d findings and %d closures per report.", maxToolFindings, maxToolFindings)
		}
		var adds []Finding
		blocking := 0
		for i, f := range in.Findings {
			if f.Blocking == nil {
				return refuse("findings[%d]: blocking is required.", i)
			}
			if !e.within(f.Owner, m.Path) {
				return refuse("findings[%d]: owner %q must be you (%q) or one of your descendants.", i, f.Owner, m.Path)
			}
			obligation, err := toolText(fmt.Sprintf("findings[%d].obligation", i), f.Obligation)
			if err != nil {
				return refuse("%v", err)
			}
			observed, err := toolText(fmt.Sprintf("findings[%d].observed", i), f.Observed)
			if err != nil {
				return refuse("%v", err)
			}
			adds = append(adds, Finding{Owner: f.Owner, RaisedBy: m.Path, Obligation: obligation, Observed: observed, Blocking: *f.Blocking})
			if *f.Blocking {
				blocking++
			}
		}
		closing := map[string]string{}
		for i, c := range in.Closed {
			f := e.findingLocked(c.ID)
			switch {
			case f == nil || f.RaisedBy != m.Path:
				return refuse("closed[%d]: %q is not a finding you raised.", i, c.ID)
			case f.Closed || closing[c.ID] != "":
				return refuse("closed[%d]: %q is already closed.", i, c.ID)
			}
			reason, err := toolText(fmt.Sprintf("closed[%d].reason", i), c.Reason)
			if err != nil {
				return refuse("%v", err)
			}
			closing[c.ID] = reason
		}
		for _, f := range e.findings {
			if f.RaisedBy == m.Path && f.Blocking && !f.Closed && closing[f.ID] == "" {
				blocking++
			}
		}
		if *in.Satisfied && blocking > 0 {
			return refuse("satisfied is true but %d blocking finding(s) of yours stay open; close them with a reason or set satisfied to false.", blocking)
		}
		if !*in.Satisfied && blocking == 0 {
			return refuse("satisfied is false but none of your findings is blocking; raise the blocking finding or set satisfied to true.")
		}

		replaced := ts.report != nil
		ts.report = &report{summary: summary, satisfied: *in.Satisfied, raised: adds, closing: closing}
		ts.status = "assessed"
		if replaced {
			return ToolAnswer{Text: "Assessment recorded; it replaces your earlier one in this turn."}
		}
		return ToolAnswer{Text: "Assessment recorded. A later assess call in this turn replaces it."}
	}
}

// applyReportLocked applies the last valid report of a Verify turn when the
// turn has ended: closures, new findings and the Manager's assessment.
func (e *engine) applyReportLocked(m *mgr, rs *roundState, ts *turnState) {
	r := ts.report
	if r == nil || rs.assessed[m.Path] {
		return
	}
	for _, id := range sortedKeys(r.closing) {
		if f := e.findingLocked(id); f != nil && !f.Closed {
			f.Closed, f.ClosedRound, f.ClosedReason = true, rs.n, r.closing[id]
			_ = e.record(Event{Kind: "finding.closed", Manager: m.Path, Turn: ts.id, Round: rs.n, Data: *f})
		}
	}
	for _, f := range r.raised {
		e.addFindingLocked(f, rs.n)
	}
	rs.assessed[m.Path] = true
	rs.assessments[m.Path] = Assessment{Manager: m.Path, Turn: ts.id, Satisfied: r.satisfied, Summary: r.summary}
	_ = e.record(Event{Kind: "assessment", Manager: m.Path, Turn: ts.id, Round: rs.n, Data: rs.assessments[m.Path]})
	e.deliverLocked(rs.n)
	e.cond.Broadcast()
}

// within reports whether owner is the manager or one of its descendants.
func (e *engine) within(owner, manager string) bool {
	for m := e.byPath[owner]; m != nil; m = e.byPath[m.Parent] {
		if m.Path == manager {
			return true
		}
	}
	return false
}

func (e *engine) findingLocked(id string) *Finding {
	for _, f := range e.findings {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// structuralLocked raises one structural finding of Account in the round as a
// blocking markitect finding; an unknown owner is replaced by Root.
func (e *engine) structuralLocked(f Finding, round int) *Finding {
	f.Blocking, f.RaisedBy = true, MarkitectFinding
	if _, ok := e.byPath[f.Owner]; !ok {
		f.Observed = fmt.Sprintf("%s (reported for unknown owner %q)", f.Observed, f.Owner)
		f.Owner = "."
	}
	if strings.TrimSpace(f.Obligation) == "" {
		f.Obligation = "Every file is accounted for in the candidate"
	}
	return e.addFindingLocked(f, round)
}

func (e *engine) addFindingLocked(f Finding, round int) *Finding {
	e.nextFinding++
	f.ID, f.Round = fmt.Sprintf("F%d", e.nextFinding), round
	p := &f
	e.findings = append(e.findings, p)
	_ = e.record(Event{Kind: "finding.raised", Manager: f.Owner, Round: round, Data: *p})
	return p
}

// deliverLocked hands every open blocking finding to its owner once per round.
func (e *engine) deliverLocked(round int) {
	for _, f := range e.findings {
		if !f.Blocking || f.Closed || e.delivered[f.ID] == round {
			continue
		}
		e.delivered[f.ID] = round
		o := e.byPath[f.Owner]
		o.items = append(o.items, item{text: renderFinding(f), finding: f.ID})
	}
}

// unsupportedFindings returns one blocking finding per owner for the entries of
// the shared tree that a candidate cannot hold, so the number of entries never
// grows the prompts. A finding names at most maxListedNames entries and counts
// all of them; its owner removes the entries in the next Execute phase.
func (e *engine) unsupportedFindings(names []string) []Finding {
	names = slices.Sorted(slices.Values(names))
	byOwner := map[string][]string{}
	var owners []string
	for _, name := range names {
		owner := "."
		if p, err := snapshot.NewPath(name); err == nil {
			if o := e.cfg.OwnerOf(p); e.byPath[o] != nil {
				owner = o
			}
		}
		if byOwner[owner] == nil {
			owners = append(owners, owner)
		}
		byOwner[owner] = append(byOwner[owner], name)
	}
	var out []Finding
	for _, owner := range owners {
		all := byOwner[owner]
		shown := all[:min(len(all), maxListedNames)]
		quoted := make([]string, len(shown))
		for i, name := range shown {
			quoted[i] = fmt.Sprintf("%q", name)
		}
		listing := strings.Join(quoted, ", ")
		if len(all) > len(shown) {
			listing += fmt.Sprintf(" and %d more", len(all)-len(shown))
		}
		out = append(out, Finding{Owner: owner, RaisedBy: MarkitectFinding, Blocking: true,
			Obligation: "Every entry under /work is a regular file or a directory the candidate can hold",
			Observed:   fmt.Sprintf("%d unsupported entries (symbolic links, devices or other special entries, or names that are not valid paths); remove them: %s", len(all), listing)})
	}
	return out
}

// openLocked reports whether the item still asks for work: an item of a
// finding lapses once the finding is closed.
func (e *engine) openLocked(it item) bool {
	if it.finding == "" {
		return true
	}
	f := e.findingLocked(it.finding)
	return f == nil || !f.Closed
}

// pruneLocked drops the Manager's undelivered items of closed findings.
func (e *engine) pruneLocked(m *mgr) {
	m.items = slices.DeleteFunc(m.items, func(it item) bool { return !e.openLocked(it) })
}

func texts(items []item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.text
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
