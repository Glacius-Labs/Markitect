package run

import "slices"

// phase is where one Manager's assignment stands. Every change follows one
// named Run event: assign, receive, fix, review, reopen or late finding.
type phase int

const (
	unassigned phase = iota // no task from the parent yet
	assigned                // task queued for delivery
	active                  // task received; the Manager works and judges
	fixed                   // handoff fixed for the parent's review
	reviewed                // parent reviewed the fixed handoff
)

// judgment binds a Manager's decision to one received task and the state it
// assessed: the Implement working copy, otherwise the fixed input candidate.
type judgment struct {
	decision Decision
	findings []Finding
	version  int
	state    Commit
	check    *Check // Root only: the completion check of state
}

// assignment is the explicit per-Manager assignment state of one Run.
type assignment struct {
	phase    phase
	task     Assignment
	version  int // counts received tasks
	judgment *judgment
	handoff  Handoff
	gen      int      // changes of the fixed handoff: fix or late finding
	sent     int      // generation the parent last received
	late     Decision // standing late adverse judgment about the fixed handoff
	feedback []Finding
}

// assign queues a parent task. A correction resolves a late Reject; a model
// blocker is never corrected within the Run.
func (a *assignment) assign(task Assignment) {
	a.phase = assigned
	a.task = task
	if a.late == Reject {
		a.late = ""
	}
}

// receive opens the delivered task. No earlier judgment applies to it, and
// no diagnostic about one is still news.
func (a *assignment) receive(task Assignment) {
	a.phase = active
	a.task = task
	a.version++
	a.judgment = nil
	a.feedback = nil
}

// judge records a judgment; a pending diagnostic about an earlier one is moot.
func (a *assignment) judge(j *judgment) {
	a.judgment = j
	a.feedback = nil
}

// invalidate drops the current judgment and tells the Manager why.
func (a *assignment) invalidate(reason string) {
	if a.judgment == nil {
		return
	}
	a.judgment = nil
	a.feedback = append(a.feedback, Finding{Text: reason})
}

func (a *assignment) fix(h Handoff) {
	a.phase = fixed
	a.handoff = h
	a.gen++
}

func (a *assignment) review(r ParentReview) {
	a.phase = reviewed
	a.handoff.Review = &r
}

// reopen makes a handed-off assignment active again: its fixed handoff rests
// on a child acceptance that no longer holds.
func (a *assignment) reopen(reason string) {
	a.phase = active
	a.invalidate(reason)
}

// lateFinding records an adverse judgment about the fixed handoff. It stays in
// force; the parent's acceptance of the handoff no longer holds.
func (a *assignment) lateFinding(decision Decision, findings []Finding) {
	a.late = requiredDecision(a.late, decision)
	a.handoff.Decision = requiredDecision(a.handoff.Decision, decision)
	a.handoff.Findings = append(slices.Clone(a.handoff.Findings), findings...)
	a.handoff.Review = nil
	a.phase = fixed
	a.gen++
}

// blocked reports a model blocker anywhere in the handoff or its review.
func (a *assignment) blocked() bool {
	return a.late == ModelBlocker || a.handoff.Decision == ModelBlocker || a.handoff.Review != nil && a.handoff.Review.Decision == ModelBlocker
}
