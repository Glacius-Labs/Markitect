package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// requiredDecision propagates required adverse findings without treating a
// previous ordinary submission as fresh acceptance during correction.
func requiredDecision(current, next Decision) Decision {
	if current == ModelBlocker || next == ModelBlocker {
		return ModelBlocker
	}
	if current == Reject || next == Reject {
		return Reject
	}
	return current
}

// childOutcomeLocked combines the reviewed handoffs of s's direct children.
// A child's handoff decision already carries its own subtree.
func (x *execution) childOutcomeLocked(s *managerState) Decision {
	outcome := Decision("")
	for _, name := range s.manager.Children {
		h := x.states[name].a.handoff
		outcome = requiredDecision(outcome, h.Decision)
		if h.Review != nil {
			outcome = requiredDecision(outcome, h.Review.Decision)
		}
	}
	return outcome
}

// workingLocked refuses parent actions outside s's active assignment: a fixed
// handoff must not change underneath the parent's review.
func (x *execution) workingLocked(s *managerState) error {
	if err := x.admissionErrorLocked(); err != nil {
		return err
	}
	switch s.a.phase {
	case active:
		return nil
	case fixed, reviewed:
		return fmt.Errorf("your handoff %s is fixed; you act on your children again only within a new assignment", s.a.handoff.Result.ID)
	}
	return fmt.Errorf("you have no active assignment")
}

// childrenLocked names the first direct child whose work blocks s's judgment.
func (x *execution) childrenLocked(s *managerState) error {
	for _, name := range s.manager.Children {
		a := x.states[name].a
		switch a.phase {
		case unassigned:
			return fmt.Errorf("child %s has no assignment yet", name)
		case assigned, active:
			return fmt.Errorf("child %s is still working on its assignment", name)
		case fixed:
			return fmt.Errorf("the review of child %s's fixed handoff %s is missing", name, a.handoff.Result.ID)
		}
	}
	return nil
}

func (x *execution) decisionError(d Decision, findings []Finding) error {
	if x.request.Mandate == Implement && d != Submitted && d != Reject && d != ModelBlocker {
		return fmt.Errorf("an Implement judgment is submitted, reject or model_blocker")
	}
	if x.request.Mandate != Implement && d != Approve && d != Reject && d != ModelBlocker {
		return fmt.Errorf("a %s judgment is approve, reject or model_blocker", x.request.Mandate)
	}
	if d != Submitted && d != Approve && !concreteFindings(findings) {
		return fmt.Errorf("reject and model_blocker need concrete Findings")
	}
	return nil
}

// judge records s's judgment of its active assignment, bound to the task and
// the assessed state. A judgment about an already fixed handoff can only be a
// late adverse finding. Root's judgment also passes its completion check.
func (x *execution) judge(ctx context.Context, s *managerState, decision Decision, findings []Finding) error {
	findings = origin(findings, s.manager.Package)
	x.mu.Lock()
	if err := x.admissionErrorLocked(); err != nil {
		x.mu.Unlock()
		return err
	}
	switch s.a.phase {
	case unassigned, assigned:
		x.mu.Unlock()
		return fmt.Errorf("you have no active assignment to judge")
	case fixed, reviewed:
		defer x.mu.Unlock()
		if decision != Reject && decision != ModelBlocker || !concreteFindings(findings) {
			return fmt.Errorf("your handoff %s is fixed; a later judgment of it is reject or model_blocker with concrete Findings", s.a.handoff.Result.ID)
		}
		x.lateLocked(s, decision, findings)
		return nil
	}
	err := x.decisionError(decision, findings)
	if err == nil {
		err = x.childrenLocked(s)
	}
	version := s.a.version
	x.mu.Unlock()
	if err != nil {
		return err
	}
	j := &judgment{decision: decision, findings: findings, version: version, state: Commit{x.request.Input.Commit, x.request.Input.Tree}}
	if x.request.Mandate == Implement {
		s.git.Lock()
		j.state, err = s.copy.Snapshot(ctx)
		s.git.Unlock()
		if errors.Is(err, ErrMergeConflict) {
			return err
		}
		if err != nil {
			x.stopPort(err)
			return err
		}
	}
	if s.manager.Package == organization.Root && (x.request.Mandate != Implement || decision == Submitted) {
		if j.check, err = x.check(ctx, s, j.state); err != nil {
			x.stopPort(err)
			return err
		}
		if !j.check.Valid && (decision == Submitted || decision == Approve) {
			return fmt.Errorf("your state %s fails its check: %s", j.state.ID, j.check.Diagnostic)
		}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.admissionErrorLocked(); err != nil {
		return err
	}
	if s.a.phase != active || s.a.version != version {
		return fmt.Errorf("your assignment changed while you judged it")
	}
	if err := x.childrenLocked(s); err != nil {
		return err
	}
	s.a.judge(j)
	return nil
}

// lateLocked records a late adverse judgment about s's fixed handoff. It stays
// in force. The parent's acceptance and every judgment resting on it no longer
// hold: each ancestor that already handed off works on its assignment again,
// up to the nearest active ancestor, which loses its judgment.
func (x *execution) lateLocked(s *managerState, decision Decision, findings []Finding) {
	s.a.lateFinding(decision, findings)
	x.queueHandoffLocked(s)
	reason := fmt.Sprintf("%s reported %s about its fixed handoff %s", s.manager.Package, decision, s.a.handoff.Result.ID)
	for p := x.states[s.manager.Parent]; p != nil; p = x.states[p.manager.Parent] {
		if p.a.phase != fixed && p.a.phase != reviewed {
			p.a.invalidate(reason)
			x.scheduleLocked(p)
			return
		}
		p.a.reopen(reason)
		x.scheduleLocked(p)
	}
}

// fix turns s's judged active assignment into its fixed handoff after the
// native response ended. A changed assessed state invalidates the judgment.
// Every Implement submission carries its check result to the parent, valid or not.
func (x *execution) fix(s *managerState, j *judgment) error {
	s.git.Lock()
	state, err := s.copy.Snapshot(x.ctx)
	s.git.Unlock()
	var check *Check
	switch {
	case errors.Is(err, ErrMergeConflict):
		x.mu.Lock()
		if s.a.judgment == j {
			s.a.invalidate(err.Error())
		}
		x.mu.Unlock()
		return nil
	case err != nil:
		return err
	case x.request.Mandate == Implement && state != j.state:
		x.mu.Lock()
		if s.a.judgment == j {
			s.a.invalidate(fmt.Sprintf("your working state changed after your judgment: %s became %s", j.state.ID, state.ID))
		}
		x.mu.Unlock()
		return nil
	case x.request.Mandate == Implement && j.decision == Submitted:
		if check, err = x.check(x.ctx, s, state); err != nil {
			return err
		}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if s.a.judgment != j || s.a.phase != active {
		return nil
	}
	s.a.fix(Handoff{Manager: s.manager.Package, Base: s.a.task.Base, Result: state, Children: slices.Clone(s.integrations), Findings: slices.Clone(j.findings), Decision: requiredDecision(j.decision, x.childOutcomeLocked(s)), Check: check})
	x.queueHandoffLocked(s)
	return nil
}

// completeRoot relies on Root's earlier judgment only after checking it again.
// An invalid completion returns its diagnostic to Root and is not done.
func (x *execution) completeRoot() (Handoff, bool, error) {
	s := x.root()
	x.mu.Lock()
	j := s.a.judgment
	if j == nil {
		x.mu.Unlock()
		return Handoff{}, false, fmt.Errorf("%w: the hierarchy stopped without Root's judgment", ErrObservationIncomplete)
	}
	if err := x.childrenLocked(s); err != nil {
		s.a.invalidate(err.Error())
		x.mu.Unlock()
		return Handoff{}, false, nil
	}
	x.mu.Unlock()
	s.git.Lock()
	state, err := s.copy.Snapshot(x.ctx)
	s.git.Unlock()
	if errors.Is(err, ErrMergeConflict) || err == nil && x.request.Mandate == Implement && state != j.state {
		reason := fmt.Sprintf("your working state changed after your judgment: %s became %s", j.state.ID, state.ID)
		if err != nil {
			reason = err.Error()
		}
		x.mu.Lock()
		s.a.invalidate(reason)
		x.mu.Unlock()
		return Handoff{}, false, nil
	}
	if err != nil {
		return Handoff{}, false, err
	}
	if x.request.Mandate == Implement {
		for _, m := range x.request.Prepared.Topology().Managers() {
			c := x.states[m.Package]
			if c == s {
				continue
			}
			c.git.Lock()
			latest, err := c.copy.Snapshot(x.ctx)
			c.git.Unlock()
			if err != nil {
				return Handoff{}, false, err
			}
			x.mu.Lock()
			handed := c.a.handoff.Result
			x.mu.Unlock()
			if latest != handed {
				return Handoff{}, false, fmt.Errorf("%w: Manager %s changed after its fixed handoff", ErrObservationIncomplete, m.Package)
			}
		}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if s.a.judgment != j {
		return Handoff{}, false, nil
	}
	h := Handoff{Manager: s.manager.Package, Base: s.a.task.Base, Result: state, Children: slices.Clone(s.integrations), Findings: slices.Clone(j.findings), Decision: requiredDecision(j.decision, x.childOutcomeLocked(s)), Check: j.check}
	s.a.fix(h)
	return h, true, nil
}

// review decides on a direct child's fixed handoff. An Implement submission
// is integrated on approval and only when its check passed.
func (x *execution) review(ctx context.Context, s *managerState, child, commit string, decision Decision, findings []Finding) error {
	target, ok := x.states[child]
	if !ok || target.manager.Parent != s.manager.Package {
		return fmt.Errorf("review covers only your direct children (%s)", strings.Join(s.manager.Children, ", "))
	}
	if decision != Approve && decision != Reject && decision != ModelBlocker {
		return fmt.Errorf("a review is approve, reject or model_blocker")
	}
	if decision != Approve && !concreteFindings(findings) {
		return fmt.Errorf("reject and model_blocker need concrete Findings")
	}
	findings = origin(findings, s.manager.Package)
	x.mu.Lock()
	if err := x.workingLocked(s); err != nil {
		x.mu.Unlock()
		return err
	}
	a := target.a
	gen := a.gen
	switch {
	case a.phase == reviewed:
		x.mu.Unlock()
		return fmt.Errorf("child %s's fixed handoff %s is already reviewed", child, a.handoff.Result.ID)
	case a.phase != fixed:
		x.mu.Unlock()
		return fmt.Errorf("child %s has no fixed handoff to review", child)
	case a.sent != a.gen:
		x.mu.Unlock()
		return fmt.Errorf("child %s's fixed handoff changed after you received it; await it again", child)
	case a.handoff.Result.ID != commit:
		x.mu.Unlock()
		return fmt.Errorf("child %s's fixed handoff is %s, not %s", child, a.handoff.Result.ID, commit)
	}
	integrate := decision == Approve && x.request.Mandate == Implement && a.handoff.Decision == Submitted
	if integrate && (a.handoff.Check == nil || !a.handoff.Check.Valid) {
		x.mu.Unlock()
		diagnostic := "no check result"
		if a.handoff.Check != nil {
			diagnostic = a.handoff.Check.Diagnostic
		}
		return fmt.Errorf("child %s's fixed handoff %s fails its check and cannot be accepted or integrated: %s; reject it with Findings or report a model blocker", child, commit, diagnostic)
	}
	x.mu.Unlock()
	if integrate {
		s.git.Lock()
		before, err := s.copy.Snapshot(ctx)
		var after Commit
		if err == nil {
			after, err = s.copy.Integrate(ctx, target.copy, a.handoff.Result)
		}
		s.git.Unlock()
		if errors.Is(err, ErrMergeConflict) {
			x.mu.Lock()
			defer x.mu.Unlock()
			if target.a.phase == fixed && target.a.gen == gen {
				target.a.review(ParentReview{Decision: Reject, Findings: []Finding{{Manager: s.manager.Package, Text: "Git integration failed: " + err.Error()}}})
			}
			return fmt.Errorf("%w; the review of %s is recorded as reject; delegate a correction with Findings to merge your current baseline into the child's copy", err, child)
		}
		if err != nil {
			x.stopPort(err)
			return err
		}
		x.mu.Lock()
		s.integrations = append(s.integrations, Integration{child, a.handoff.Base, a.handoff.Result, before, after})
		x.mu.Unlock()
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.workingLocked(s); err != nil {
		return err
	}
	if target.a.phase != fixed || target.a.gen != gen {
		return fmt.Errorf("child %s's fixed handoff changed during your review; await it again", child)
	}
	target.a.review(ParentReview{Decision: decision, Findings: findings})
	return nil
}

// delegate queues a task for a direct child: its first assignment, or after a
// review a correction with concrete Findings. In Implement the parent's current
// working state becomes the child's baseline.
func (x *execution) delegate(ctx context.Context, s *managerState, child, task string, findings []Finding) error {
	target, ok := x.states[child]
	if !ok || target.manager.Parent != s.manager.Package {
		return fmt.Errorf("delegate reaches only your direct children (%s)", strings.Join(s.manager.Children, ", "))
	}
	if strings.TrimSpace(task) == "" {
		return fmt.Errorf("a delegation needs a concrete task")
	}
	findings = origin(findings, s.manager.Package)
	refusal := func(a assignment) error {
		switch {
		case a.phase == assigned || a.phase == active:
			return fmt.Errorf("child %s is still working on its assignment", child)
		case a.phase != unassigned && a.blocked():
			return fmt.Errorf("child %s reported a model blocker; it needs a new model and Run", child)
		case a.phase == fixed:
			return fmt.Errorf("review child %s's fixed handoff %s before a correction", child, a.handoff.Result.ID)
		case a.phase == reviewed && !concreteFindings(findings):
			return fmt.Errorf("a correction needs concrete Findings")
		}
		return nil
	}
	x.mu.Lock()
	err := x.workingLocked(s)
	if err == nil {
		err = refusal(target.a)
	}
	x.mu.Unlock()
	if err != nil {
		return err
	}
	base := Commit{x.request.Input.Commit, x.request.Input.Tree}
	if x.request.Mandate == Implement {
		s.git.Lock()
		base, err = s.copy.Snapshot(ctx)
		s.git.Unlock()
		if errors.Is(err, ErrMergeConflict) {
			return err
		}
		if err != nil {
			x.stopPort(err)
			return err
		}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.workingLocked(s); err != nil {
		return err
	}
	if err := refusal(target.a); err != nil {
		return err
	}
	assignment := Assignment{Task: task, Findings: findings, Base: base}
	target.a.assign(assignment)
	x.queueLocked(target, entry{event: Event{Assignment: &assignment}})
	return nil
}

// checkTool runs the shared deterministic check on the caller's assessed
// state: its current Implement working copy, otherwise the fixed input.
func (x *execution) checkTool(ctx context.Context, s *managerState) (Check, error) {
	state := Commit{x.request.Input.Commit, x.request.Input.Tree}
	if x.request.Mandate == Implement {
		var err error
		s.git.Lock()
		state, err = s.copy.Snapshot(ctx)
		s.git.Unlock()
		if errors.Is(err, ErrMergeConflict) {
			return Check{Diagnostic: err.Error(), Checks: []string{}}, nil
		}
		if err != nil {
			x.stopPort(err)
			return Check{}, err
		}
	}
	result, err := x.check(ctx, s, state)
	if err != nil {
		x.stopPort(err)
		return Check{}, err
	}
	return *result, nil
}

func copyChecks(observed []CheckObservation) []CheckObservation {
	copied := slices.Clone(observed)
	for i := range copied {
		copied[i].Receipt.Output = slices.Clone(copied[i].Receipt.Output)
		copied[i].Receipt.Directories = slices.Clone(copied[i].Receipt.Directories)
	}
	return copied
}

func copyHandoff(h Handoff) Handoff {
	h.Children = slices.Clone(h.Children)
	h.Findings = slices.Clone(h.Findings)
	if h.Review != nil {
		review := *h.Review
		review.Findings = slices.Clone(review.Findings)
		h.Review = &review
	}
	if h.Check != nil {
		check := *h.Check
		check.Checks = slices.Clone(check.Checks)
		h.Check = &check
	}
	return h
}

// The terminal result owns every mutable value passed to persistence and the
// caller. Bounded cleanup may leave a worker alive; it owns no part of this value.
func copyResult(r Result) Result {
	r.Sessions = slices.Clone(r.Sessions)
	r.Findings = slices.Clone(r.Findings)
	r.Handoffs = slices.Clone(r.Handoffs)
	for i := range r.Handoffs {
		r.Handoffs[i] = copyHandoff(r.Handoffs[i])
	}
	r.Checks = copyChecks(r.Checks)
	r.Cleanup = slices.Clone(r.Cleanup)
	r.Errors = slices.Clone(r.Errors)
	if r.Candidate != nil {
		ref := *r.Candidate
		r.Candidate = &ref
	}
	return r
}
