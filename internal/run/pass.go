package run

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// stage is where a Manager stands in the base process of one Pass. Every
// purpose and every Manager, Root and leaves included, passes through the same
// stages; a leaf simply has no children to delegate to.
type stage int

const (
	// stageStart: waiting for the parent's accepted basis; not yet released.
	stageStart stage = iota
	// stageBlocked: released by the parent's accepted basis, but its productive
	// work waits for declared dependencies. It may handle permitted messages in
	// communication-only turns with read-only project access.
	stageBlocked
	// stageOwn: working on its own contribution and the basis its children need.
	stageOwn
	// stageBasis: its basis is submitted: technically checked, then decided by
	// the parent (Root's by the Orchestrator).
	stageBasis
	// stageDelegated: its basis is accepted; its children work, and it remains
	// available for their questions and proposals.
	stageDelegated
	// stageIntegrate: every child result is accepted; it integrates them with its
	// own contribution into its complete result.
	stageIntegrate
	// stageResult: its complete result is submitted.
	stageResult
	// stageDone: its complete result is accepted.
	stageDone
)

func (s stage) String() string {
	return [...]string{"start", "blocked", "own", "basis", "delegated", "integrate", "result", "done"}[s]
}

// itemKind says how an undelivered item is shown.
type itemKind int

const (
	itemNote itemKind = iota
	// itemAssignment carries the Pass assignment, rendered when it is delivered.
	itemAssignment
	itemFinding
	// itemMessage is a question, answer or information along the tree: the
	// permitted work of a Manager whose productive work is blocked.
	itemMessage
)

// item is one undelivered piece of work for a Manager. An item of a finding
// lapses once the finding is closed.
type item struct {
	kind    itemKind
	text    string
	finding string
}

// member is one Manager's state in one Pass. Every field is guarded by
// engine.mu.
type member struct {
	*mgr
	p     *pass
	stage stage
	state State
	// waiting names what a Waiting member waits for.
	waiting string
	// waitFor is the dependency its last wait report named and waitOn the
	// accepted submission of another Manager it names; both are cleared when
	// new work arrives.
	waitFor string
	waitOn  *waitTarget
	// bound are the dependencies its waits declared in a Pass with BindWaits:
	// the accepted submissions of other Managers its work builds on.
	bound []binding
	// stale says why the submission of its running turn no longer stands: what
	// its result built on changed during the turn.
	stale string
	// progressed holds its progress summaries in this Pass: a repeated one
	// is no progress.
	progressed []string
	items      []item
	consumed   []item
	running    bool
	checking   bool
	// pending is its submitted basis or result until it is decided.
	pending       *proposal
	basis, result *proposal
	turns         int
	empty         int
	// reviewHome and reviewConversation hold the fresh reviewer of a review
	// Pass; they never carry over to another Pass.
	reviewHome string
	// reviewRehydrate is rehydrate of the fresh reviewer's conversation.
	reviewRehydrate    bool
	reviewConversation string
	helpers            []*helperRun
	nextHelper         int
	// recorded is set once its first state was recorded in the Pass.
	recorded bool
}

// pass is the state of one Pass. Fields are guarded by engine.mu.
type pass struct {
	id     string
	number int
	spec   PassSpec
	// tree is the stand the Pass started from; for a review it is the candidate.
	tree   snapshot.Tree
	volume string
	// candidate is the review Pass's read-only volume.
	candidate WorkspaceID
	// accepted is the stand assembled from the entry stand and every accepted
	// submission's checked area; only the Orchestrator writes it.
	accepted  snapshot.Tree
	members   []*member
	byPath    map[string]*member
	checks    []CheckResult
	running   int
	checking  int
	sealed    bool
	verdict   Verdict
	sealedOn  string
	proposals []*proposal
}

func (p *pass) recordLocked() PassRecord {
	r := PassRecord{ID: p.id, Purpose: p.spec.Purpose, Review: p.spec.Review, Verdict: p.verdict, Sealed: p.sealed, Checks: slices.Clone(p.checks), Tree: p.sealedOn}
	for _, m := range p.members {
		mr := ManagerRecord{Manager: m.Path, State: m.state}
		if m.basis != nil {
			mr.Basis = m.basis.id
		}
		if m.result != nil {
			mr.Result, mr.Verdict, mr.Summary = m.result.id, m.result.env.Verdict, m.result.env.Summary
		}
		r.Managers = append(r.Managers, mr)
	}
	return r
}

func validateSpec(spec PassSpec) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
	}
	switch {
	case !nameRE.MatchString(string(spec.Purpose)):
		return bad("purpose %q is not a valid name", spec.Purpose)
	case strings.TrimSpace(spec.Mandate) == "":
		return bad("pass %q needs a mandate", spec.Purpose)
	case spec.Review && (spec.Writes || spec.Plans || spec.BindWaits):
		return bad("a review Pass is read-only, plans nothing and binds no dependency")
	case spec.Assesses && (!spec.Review || spec.ArtifactFindingsAdvisory):
		return bad("a model assessment is a review Pass of the model, without artifact findings")
	case spec.Review && strings.TrimSpace(spec.Guidance) == "":
		return bad("a review Pass needs guidance on what a genuine finding is")
	case !spec.Review && spec.ArtifactFindingsAdvisory:
		return bad("only a review Pass reports artifact findings")
	case len(spec.Corrections) == 0:
		return bad("pass %q needs the correction purposes of its findings", spec.Purpose)
	}
	seen := map[Purpose]bool{}
	for _, c := range spec.Corrections {
		if !nameRE.MatchString(string(c)) || seen[c] || c == Verify {
			return bad("correction purpose %q is invalid or repeated", c)
		}
		seen[c] = true
	}
	return nil
}

// defaultCorrection is the most local correction purpose: the last listed.
func (s PassSpec) defaultCorrection() Purpose { return s.Corrections[len(s.Corrections)-1] }

func (e *engine) newPassLocked(spec PassSpec) *pass {
	n := len(e.passes) + 1
	p := &pass{id: fmt.Sprintf("p%d", n), number: n, spec: spec, tree: e.stand, accepted: e.stand, volume: e.work, byPath: map[string]*member{}}
	for _, m := range e.team {
		x := &member{mgr: m, p: p, stage: stageStart, state: Waiting}
		if spec.Review {
			x.reviewHome = fmt.Sprintf("%s-%s-%s-r", e.cfg.RunID, p.id, m.slug)
		}
		p.members = append(p.members, x)
		p.byPath[m.Path] = x
	}
	e.passes = append(e.passes, p)
	e.cur = p
	return p
}

// runPass runs the base process of one Pass for the whole required tree and
// seals it.
func (e *engine) runPass(spec PassSpec) PassResult {
	if st, _, _ := e.stopped(); st != "" {
		return PassResult{Stopped: true}
	}
	if err := validateSpec(spec); err != nil {
		e.fail(err)
		return PassResult{Stopped: true}
	}
	e.mu.Lock()
	if e.work == "" {
		e.mu.Unlock()
		e.halt("the shared working tree is not available")
		return PassResult{Stopped: true}
	}
	if len(e.passes) >= e.cfg.Budget.Passes {
		e.mu.Unlock()
		e.halt(fmt.Sprintf("the Pass budget of %d ended", e.cfg.Budget.Passes))
		return PassResult{Stopped: true}
	}
	p := e.newPassLocked(spec)
	e.mu.Unlock()
	if spec.Review && !e.prepareReview(p) {
		return e.passEnded(p)
	}
	e.mu.Lock()
	_ = e.record(Event{Kind: "pass.start", Pass: p.id, Data: map[string]any{"purpose": spec.Purpose, "review": spec.Review, "writes": spec.Writes, "tree": p.tree.Digest().String()}})
	for _, m := range p.members {
		m.waiting = e.parentWaitLocked(m)
		e.recordStateLocked(m, "the Pass started")
	}
	e.releaseLocked(p.members[0], "the Pass started")
	e.scheduleLocked()
	for e.ctx.Err() == nil && !p.closableLocked() {
		if reason := e.stalledLocked(p); reason != "" {
			e.addProblemLocked(Problem{Manager: ".", Category: "dependency", Text: reason})
			e.halt(reason)
			break
		}
		e.cond.Wait()
	}
	// Delegated helpers that never started do not run once the run stopped.
	for p.running > 0 || p.checking > 0 {
		e.cond.Wait()
	}
	e.mu.Unlock()
	if e.ctx.Err() != nil {
		return e.passEnded(p)
	}
	return e.seal(p)
}

// passEnded returns the result of a Pass the run stopped in.
func (e *engine) passEnded(p *pass) PassResult {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p.candidate != "" {
		e.closeVolume(p.candidate)
		p.candidate = ""
	}
	e.verified = nil
	return PassResult{Record: p.recordLocked(), Stopped: true}
}

func (p *pass) closableLocked() bool {
	if p.running > 0 || p.checking > 0 || p.helpersActiveLocked() {
		return false
	}
	for _, m := range p.members {
		if m.stage != stageDone {
			return false
		}
	}
	return true
}

func (p *pass) helpersActiveLocked() bool {
	for _, m := range p.members {
		if len(m.helpers) > 0 {
			return true
		}
	}
	return false
}

// stalledLocked names why the Pass cannot advance when nothing runs, nothing
// is being checked, no Manager has deliverable work and nobody waits for the
// user: a cyclic or orphaned wait. It is empty while the Pass can advance.
func (e *engine) stalledLocked(p *pass) string {
	if p.running > 0 || p.checking > 0 || p.helpersActiveLocked() {
		return ""
	}
	for _, m := range p.members {
		if m.running || m.checking || !m.disabled && e.runnableLocked(m) {
			return ""
		}
	}
	var waits []string
	for _, m := range p.members {
		if m.stage != stageDone {
			what := m.waiting
			if what == "" {
				what = "nothing it can receive"
			}
			waits = append(waits, fmt.Sprintf("%s waits for %s", m.Path, what))
		}
	}
	return "the Pass cannot advance: " + strings.Join(waits, "; ")
}

// parentWaitLocked names the accepted parent basis a Manager still needs before
// it is released in its Pass, or is empty.
func (e *engine) parentWaitLocked(m *member) string {
	if m.Parent != "" {
		if parent := m.p.byPath[m.Parent]; parent.basis == nil {
			return "the accepted basis of " + labelOf(parent.mgr)
		}
	}
	return ""
}

// dependencyWaitLocked names the first declared dependency, or after a reset
// the first submission its basis was bound to, that a released Manager's
// productive work still needs, or is empty.
func (e *engine) dependencyWaitLocked(m *member) string {
	p := m.p
	for _, d := range e.dependenciesLocked(m) {
		y := p.byPath[d.Owner]
		if d.Stage == stageNameBasis && y.basis == nil || d.Stage == stageNameResult && y.result == nil {
			return fmt.Sprintf("the accepted %s of %s (%s)", d.Stage, labelOf(y.mgr), d.Need)
		}
	}
	return ""
}

// dependenciesLocked returns the start conditions of a Manager in its Pass: the
// dependencies of its accepted plan that apply to the purpose of the Pass and
// the submissions its basis in this Pass is bound to.
func (e *engine) dependenciesLocked(m *member) []dependency {
	var out []dependency
	if plan := e.plans[m.Path]; plan != nil && plan.env.Plan != nil && !m.p.spec.Plans {
		for _, d := range plan.env.Plan.Dependencies {
			if d.Pass == m.p.spec.Purpose {
				out = append(out, d)
			}
		}
	}
	for _, b := range m.bound {
		on := b.target
		if b.serves == eventBasis && !slices.ContainsFunc(out, func(d dependency) bool { return d.Owner == on.manager && d.Stage == on.stage }) {
			out = append(out, dependency{Owner: on.manager, Stage: on.stage, Pass: m.p.spec.Purpose, Need: "your basis in this Pass builds on it"})
		}
	}
	return out
}

// releaseLocked advances a Manager in its Pass as far as its conditions allow.
// The accepted parent basis releases it: its assignment is delivered first,
// then the open genuine findings it owns for a work Pass, then what reached it
// while it waited. Its productive work starts once the dependencies of its
// accepted plan for this purpose, and after a reset the submissions its basis
// was bound to, are accepted too; until then it is blocked and may only handle
// permitted messages, read-only.
func (e *engine) releaseLocked(m *member, why string) {
	if m.disabled {
		return
	}
	if m.stage == stageStart {
		if w := e.parentWaitLocked(m); w != "" {
			m.waiting = w
			e.recordStateLocked(m, "its parent's basis is not accepted yet")
			return
		}
		m.stage = stageBlocked
		front := []item{{kind: itemAssignment}}
		if !m.p.spec.Review {
			for _, f := range e.findings {
				// A suspicion is investigated by a review, never worked on.
				if f.Blocking && !f.Closed && f.Owner == m.Path && f.Kind != Suspected {
					front = append(front, item{kind: itemFinding, text: renderFinding(f), finding: f.ID})
				}
			}
		}
		m.items = append(front, m.items...)
		data := map[string]any{"why": why}
		if m.Parent != "" {
			data["parent"] = m.Parent
		}
		_ = e.record(Event{Kind: "manager.release", Pass: m.p.id, Manager: m.Path, Data: data})
	}
	if m.stage != stageBlocked {
		return
	}
	if w := e.dependencyWaitLocked(m); w != "" {
		m.waiting = w
		e.recordStateLocked(m, "its productive work waits for a dependency")
		return
	}
	m.stage, m.waiting = stageOwn, ""
	e.rebindLocked(m)
	if !slices.ContainsFunc(m.items, func(it item) bool { return it.kind == itemAssignment }) {
		// It already handled messages while blocked: its assignment was delivered,
		// so it learns which dependencies it now builds on.
		var deps []string
		for _, d := range e.dependenciesLocked(m) {
			if pr := e.dependencyProposalLocked(m, d); pr != nil {
				deps = append(deps, e.renderAcceptedLocked(pr))
			}
		}
		text := "Your dependencies are accepted; work on your own contribution now."
		if len(deps) > 0 {
			text += "\n" + strings.Join(deps, "\n")
		}
		m.items = append(m.items, item{text: text})
		_ = e.record(Event{Kind: "manager.unblocked", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"why": why}})
	}
	e.recordStateLocked(m, why)
}

// dependencyProposalLocked returns the accepted submission a dependency names.
func (e *engine) dependencyProposalLocked(m *member, d dependency) *proposal {
	y := m.p.byPath[d.Owner]
	if d.Stage == stageNameResult {
		return y.result
	}
	return y.basis
}

// releaseWaitingLocked advances every Manager of the Pass whose conditions may
// now hold.
func (e *engine) releaseWaitingLocked(p *pass, why string) {
	for _, m := range p.members {
		if m.stage == stageStart || m.stage == stageBlocked {
			e.releaseLocked(m, why)
		}
	}
}

// computeState derives a Manager's scheduling state and what it waits for.
func (e *engine) computeState(m *member) (State, string) {
	switch {
	case m.stage == stageDone:
		return Done, ""
	case m.running || m.checking:
		return Working, ""
	case m.stage == stageStart:
		return Waiting, m.waiting
	case m.stage == stageBlocked && !e.runnableLocked(m):
		return Waiting, m.waiting
	case len(m.items) > 0:
		return Ready, ""
	case m.stage == stageBasis || m.stage == stageResult:
		parent := "the Orchestrator"
		if m.Parent != "" {
			parent = labelOf(e.byPath[m.Parent])
		}
		return Waiting, fmt.Sprintf("the decision of %s on %s", parent, m.pending.id)
	case m.waitFor != "":
		return Waiting, m.waitFor
	case len(m.helpers) > 0:
		return Waiting, "the results of its helpers"
	case m.stage == stageDelegated:
		return Waiting, "the results of its children"
	}
	return Ready, ""
}

// recordStateLocked recomputes a Manager's state and records a change of the
// state or of what it waits for.
func (e *engine) recordStateLocked(m *member, why string) {
	state, waiting := e.computeState(m)
	if m.recorded && state == m.state && waiting == m.waiting {
		return
	}
	from := m.state
	m.state, m.recorded = state, true
	if m.stage != stageStart && m.stage != stageBlocked {
		m.waiting = waiting
	}
	data := map[string]any{"from": from, "to": state, "stage": m.stage.String(), "why": why}
	if state == Waiting {
		data["waitingFor"] = waiting
	}
	_ = e.record(Event{Kind: "manager.state_changed", Pass: m.p.id, Manager: m.Path, Data: data})
}

// deliverLocked adds an item for a Manager of the current Pass and wakes it.
func (e *engine) deliverLocked(m *member, it item) {
	m.items = append(m.items, it)
	m.waitFor, m.waitOn = "", nil
	e.recordStateLocked(m, "new work arrived")
	e.cond.Broadcast()
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

func (e *engine) pruneLocked(m *member) {
	m.items = slices.DeleteFunc(m.items, func(it item) bool { return !e.openLocked(it) })
}

// scheduleLocked starts a turn for every idle Manager of the current Pass with
// deliverable work, and every delegated helper whose Manager's turn ended.
func (e *engine) scheduleLocked() {
	p := e.cur
	if p == nil || p.sealed || e.closing || e.ctx.Err() != nil {
		return
	}
	e.startHelpersLocked(p)
	for _, m := range p.members {
		if m.running || m.disabled {
			continue
		}
		e.pruneLocked(m)
		if e.runnableLocked(m) {
			e.startTurnLocked(m)
		}
	}
	e.cond.Broadcast()
}

// runnableLocked reports whether a Manager has work a turn may deliver now. A
// Manager that is not released has none. A blocked Manager may only handle
// messages and the questions addressed to it, with read-only access; its
// assignment waits for its productive start.
func (e *engine) runnableLocked(m *member) bool {
	switch m.stage {
	case stageStart:
		return false
	case stageBlocked:
		return slices.ContainsFunc(m.items, func(it item) bool { return it.kind == itemMessage }) || len(e.demandsLocked(m)) > 0
	}
	return len(m.items) > 0
}

// childrenDoneLocked reports whether every child of a Manager is Done.
func (m *member) childrenDoneLocked() bool {
	for _, c := range m.Children {
		if m.p.byPath[c].stage != stageDone {
			return false
		}
	}
	return true
}

// acceptLocked accepts a checked submission: by the parent's semantic decision,
// or by the Orchestrator for Root. The submitter's checked area becomes part of
// the accepted stand once it still passes the File check against the current
// accepted stand.
func (e *engine) acceptLocked(pr *proposal, by string) {
	m, p := pr.member, pr.member.p
	if pr.status != proposalChecked || m.pending != pr {
		return
	}
	if p.spec.Writes {
		composite, err := overlay(p.accepted, pr.composite, m.Path, e.cfg.OwnerOf)
		if err == nil && composite.Digest() != pr.composite.Digest() {
			// Another submission was accepted since the check: check the
			// composition with the current accepted stand again.
			if findings := ownFindings(e.cfg.FileCheck(composite), m.Path); len(findings) > 0 {
				e.technicalReturnLocked(pr, findings)
				return
			}
		}
		if err != nil {
			e.technicalReturnLocked(pr, []Finding{{Owner: m.Path, Obligation: "the submission composes with the accepted stand", Observed: err.Error()}})
			return
		}
		p.accepted = composite
	}
	pr.status, pr.decidedBy = proposalAccepted, by
	m.pending = nil
	e.takeEffectsLocked(pr)
	view := e.cfg.Registry(p.accepted)
	kind := "basis.accepted"
	if pr.kind == kindResult {
		kind = "result.accepted"
	}
	_ = e.record(Event{Kind: kind, Pass: p.id, Manager: m.Path, Data: map[string]any{"proposal": pr.id, "by": by, "stand": p.accepted.Digest().String(), "registryShards": view.Shards, "registrations": len(view.Registrations)}})
	// Wake exactly the Managers that wait for this acceptance.
	for _, x := range p.members {
		if x.waitOn != nil && x.waitOn.manager == m.Path && x.waitOn.stage == pr.kind {
			note := e.bindWaitLocked(x, *x.waitOn, pr)
			e.deliverLocked(x, item{text: e.renderAcceptedLocked(pr) + "\nThe submission you waited for is accepted; continue your work." + note})
		}
	}
	e.resumeLocked(m, pr)
	switch pr.kind {
	case kindBasis:
		m.basis = pr
		if len(m.Children) == 0 {
			m.stage = stageIntegrate
			e.deliverLocked(m, item{text: fmt.Sprintf("Your basis %s is accepted. You have no children: complete your own contribution of this %s Pass and submit your complete result as its mandate defines it.", pr.id, p.spec.Purpose)})
		} else {
			m.stage = stageDelegated
			e.recordStateLocked(m, "basis accepted")
			for _, c := range m.Children {
				e.releaseLocked(p.byPath[c], "the basis of its parent was accepted")
			}
		}
	case kindResult:
		m.result = pr
		m.stage = stageDone
		e.recordStateLocked(m, "result accepted")
		if m.Parent != "" {
			if parent := p.byPath[m.Parent]; parent.stage == stageDelegated && parent.childrenDoneLocked() {
				parent.stage = stageIntegrate
				e.deliverLocked(parent, item{text: "The complete results of all your children are accepted. Integrate them with your own contribution and submit your complete result."})
			}
		}
	}
	e.releaseWaitingLocked(p, fmt.Sprintf("%s of %s was accepted", pr.id, m.Path))
	e.scheduleLocked()
}

// returnLocked returns a submission to its owner within the current Pass.
func (e *engine) returnLocked(pr *proposal, by, reason string) {
	m := pr.member
	pr.status, pr.decidedBy, pr.reason = proposalReturned, by, reason
	m.pending = nil
	m.stage = stageOwn
	if pr.kind == kindResult {
		m.stage = stageIntegrate
	}
	kind := "basis.returned"
	if pr.kind == kindResult {
		kind = "result.returned"
	}
	_ = e.record(Event{Kind: kind, Pass: m.p.id, Manager: m.Path, Data: map[string]any{"proposal": pr.id, "by": by, "reason": reason}})
	e.deliverLocked(m, item{text: fmt.Sprintf("%s returned your %s %s: %s", by, pr.kind, pr.id, reason)})
	e.scheduleLocked()
}

// technicalReturnLocked returns a submission whose File check failed.
func (e *engine) technicalReturnLocked(pr *proposal, findings []Finding) {
	m := pr.member
	pr.status = proposalFailed
	m.pending = nil
	m.stage = stageOwn
	if pr.kind == kindResult {
		m.stage = stageIntegrate
	}
	var lines []string
	for _, f := range findings[:min(len(findings), maxListed)] {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", f.Owner, f.Obligation, f.Observed))
	}
	if len(findings) > maxListed {
		lines = append(lines, fmt.Sprintf("and %d more", len(findings)-maxListed))
	}
	_ = e.record(Event{Kind: "proposal.checked", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"proposal": pr.id, "passed": false, "findings": lines}})
	e.deliverLocked(m, item{text: fmt.Sprintf("Your %s %s did not pass the File check of your area and registry; it is not accepted. Fix these and submit again:\n%s", pr.kind, pr.id, strings.Join(lines, "\n"))})
	e.scheduleLocked()
}

// reopenLocked lets a parent reopen a Done child's accepted result before the
// Pass closes: the child integrates again and the parent waits for it.
func (e *engine) reopenLocked(parent, child *member, reason string) {
	old := child.result
	old.status = proposalReopened
	child.result = nil
	child.stage = stageIntegrate
	parent.stage = stageDelegated
	_ = e.record(Event{Kind: "result.reopened", Pass: child.p.id, Manager: child.Path, Data: map[string]any{"proposal": old.id, "by": parent.Path, "reason": reason}})
	e.voidDispositionsLocked(old, "its result was reopened")
	e.invalidateConsumersLocked(child, false, fmt.Sprintf("the result %s of %s was reopened", old.id, child.Path))
	e.deliverLocked(child, item{text: fmt.Sprintf("Your parent reopened your accepted result %s: %s", old.id, reason)})
	e.recordStateLocked(parent, "reopened a child result")
	e.scheduleLocked()
}

// reviseLocked lets a Manager revise its accepted basis: the basis and every
// consumer built on it are explicitly invalidated, and the Manager works on its
// basis again.
func (e *engine) reviseLocked(m *member, reason string) {
	old := m.basis
	old.status = proposalInvalidated
	m.basis = nil
	m.stage = stageOwn
	_ = e.record(Event{Kind: "basis.invalidated", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"proposal": old.id, "reason": reason}})
	e.voidDispositionsLocked(old, "its basis was revised")
	e.invalidateConsumersLocked(m, true, fmt.Sprintf("%s revises its basis %s: %s", m.Path, old.id, reason))
	e.deliverLocked(m, item{text: fmt.Sprintf("You revise your accepted basis %s (%s). Rework it and submit the revised basis; your children and dependents wait for it.", old.id, reason)})
	e.scheduleLocked()
}

// invalidateConsumersLocked resets every Manager that started on m's basis (its
// children and its declared or bound dependents) or, when basis is false, on
// its result, together with their own consumers, and unsettles their parents.
func (e *engine) invalidateConsumersLocked(m *member, basis bool, why string) {
	p := m.p
	var consumers []*member
	if basis {
		for _, c := range m.Children {
			consumers = append(consumers, p.byPath[c])
		}
	}
	for _, z := range p.members {
		for _, d := range e.dependenciesLocked(z) {
			if d.Owner == m.Path && (basis && d.Stage == stageNameBasis || !basis && d.Stage == stageNameResult) && !slices.Contains(consumers, z) {
				consumers = append(consumers, z)
			}
		}
	}
	// A Manager whose result alone built on it integrates again: its basis and
	// its children's work stay.
	var results []*member
	for _, z := range e.resultConsumersLocked(m, basis) {
		if !slices.Contains(consumers, z) {
			results = append(results, z)
		}
	}
	for _, z := range consumers {
		e.resetLocked(z, why)
	}
	for _, z := range results {
		e.unsettleResultLocked(z, m.Path, basis, why)
	}
}

// resetLocked puts a Manager back before its start: its submissions of this
// Pass are invalidated and its own consumers are reset in turn. A parent whose
// integrated or accepted result included it is unsettled.
func (e *engine) resetLocked(z *member, why string) {
	if z.stage == stageStart {
		return
	}
	hadBasis, hadResult := z.basis != nil, z.result != nil
	for _, pr := range []*proposal{z.pending, z.basis, z.result} {
		if pr != nil && pr.status != proposalInvalidated {
			pr.status = proposalInvalidated
			_ = e.record(Event{Kind: "proposal.invalidated", Pass: z.p.id, Manager: z.Path, Data: map[string]any{"proposal": pr.id, "why": why}})
			e.voidDispositionsLocked(pr, why)
		}
	}
	z.pending, z.basis, z.result = nil, nil, nil
	// What its basis was bound to stays a start condition; what its result built
	// on stays bound or awaits its next version.
	e.settleResultBindingsLocked(z)
	z.stage, z.waitFor, z.waitOn = stageStart, "", nil
	z.items = []item{{text: "Your earlier work in this Pass was reset because " + why + ". Re-establish your basis against the current accepted stand."}}
	if hadBasis {
		e.invalidateConsumersLocked(z, true, why)
	}
	if hadResult {
		e.invalidateConsumersLocked(z, false, why)
	}
	e.unsettleParentLocked(z, why)
	z.waiting = e.parentWaitLocked(z)
	e.recordStateLocked(z, why)
	e.releaseLocked(z, why)
}

// unsettleParentLocked moves the parent of a reset Manager back to waiting for
// its children: an integrating or submitted result can no longer stand, and an
// accepted one is reopened up the tree.
func (e *engine) unsettleParentLocked(z *member, why string) {
	if z.Parent == "" {
		return
	}
	parent := z.p.byPath[z.Parent]
	switch parent.stage {
	case stageIntegrate:
		parent.stage = stageDelegated
	case stageResult:
		if parent.pending != nil {
			parent.pending.status = proposalInvalidated
			_ = e.record(Event{Kind: "proposal.invalidated", Pass: z.p.id, Manager: parent.Path, Data: map[string]any{"proposal": parent.pending.id, "why": why}})
			parent.pending = nil
		}
		parent.stage = stageDelegated
	case stageDone:
		old := parent.result
		old.status = proposalReopened
		parent.result = nil
		parent.stage = stageDelegated
		_ = e.record(Event{Kind: "result.reopened", Pass: z.p.id, Manager: parent.Path, Data: map[string]any{"proposal": old.id, "by": "markitect", "reason": why}})
		e.voidDispositionsLocked(old, why)
		e.invalidateConsumersLocked(parent, false, why)
		e.unsettleParentLocked(parent, why)
	default:
		return
	}
	e.deliverLocked(parent, item{text: fmt.Sprintf("Your child %s must re-establish its work because %s; wait for its new result.", z.Path, why)})
}

// verdictLocked is the aggregate verdict of a completed Pass: every accepted
// result is satisfied and no genuine finding of the kind the Pass judges stays
// open; a review also needs every required check passing on its candidate.
func (e *engine) verdictLocked(p *pass) Verdict {
	// Root's result integrates the whole tree. A child's needs_correction rests
	// on an open genuine finding, which the findings below decide, so a finding
	// its raiser closed after the child's result no longer holds the Pass back.
	for _, m := range p.members {
		if m.Parent == "" && (m.result == nil || m.result.env.Verdict != Satisfied) {
			return NeedsCorrection
		}
		// An accepted result that leaves an own obligation open needs
		// correction, whatever finding it rests on.
		if m.result != nil && slices.ContainsFunc(m.result.env.Obligations, func(c coverage) bool { return c.Status == coverOpen }) {
			return NeedsCorrection
		}
	}
	for _, f := range e.findings {
		if f.Blocking && !f.Closed && (p.spec.Review || !f.Review && f.RaisedBy != MarkitectFinding) {
			return NeedsCorrection
		}
	}
	if p.spec.Review {
		if len(p.checks) != len(e.checksOf(p.spec)) {
			return NeedsCorrection
		}
		for _, c := range p.checks {
			if !c.Passed {
				return NeedsCorrection
			}
		}
		// An adverse probe receipt on the candidate stays adverse evidence until
		// an independent disposition, whatever the reports cited.
		if len(e.undisposedLocked(p.tree.Digest().String(), ".", "", nil, nil)) > 0 {
			return NeedsCorrection
		}
	}
	return Satisfied
}

// seal performs the whole-stand exit gates of a completed Pass and seals its
// result. A work Pass captures the quiescent shared tree, which must be exactly
// the accepted stand, and runs the Model check and the File check on it; any
// failure ends the run Failed without a repair.
func (e *engine) seal(p *pass) PassResult {
	stand := p.tree
	if !p.spec.Review {
		h, err := e.cfg.Workspaces.Harvest(e.ctx, HarvestRequest{ID: e.workID, MaxEntries: e.cfg.MaxEntries, MaxBytes: e.cfg.MaxBytes})
		if err == nil && h.Uncertain {
			err = fmt.Errorf("the harvest is uncertain: %s", h.Reason)
		}
		if err != nil {
			e.halt(fmt.Sprintf("capture of the stand at the end of %s is uncertain: %v", p.id, err))
			return e.passEnded(p)
		}
		stand = h.Tree
		e.mu.Lock()
		accepted := p.accepted
		e.mu.Unlock()
		if len(h.Unsupported) > 0 {
			e.reject(fmt.Sprintf("the stand at the end of %s holds %d unsupported entries: %s", p.id, len(h.Unsupported), listNames(h.Unsupported)))
			return e.passEnded(p)
		}
		if !stand.Equal(accepted) {
			var paths []string
			for _, c := range snapshot.Compare(accepted, stand) {
				paths = append(paths, c.Path().String())
			}
			e.reject(fmt.Sprintf("the stand at the end of %s differs from its accepted stand in %s", p.id, listNames(paths)))
			return e.passEnded(p)
		}
		if !e.gate(p, stand, "at the end") {
			return e.passEnded(p)
		}
	}
	e.mu.Lock()
	p.verdict = e.verdictLocked(p)
	p.sealed, p.sealedOn = true, stand.Digest().String()
	for _, q := range e.questions {
		if q.Pass == p.id && !q.Answered && !q.Lapsed {
			q.Lapsed = true
			_ = e.record(Event{Kind: "question.lapsed", Pass: p.id, Manager: q.From, Data: *q})
		}
	}
	e.stand = stand
	if p.spec.Plans {
		e.plans = map[string]*proposal{}
		for _, m := range p.members {
			e.plans[m.Path] = m.result
		}
	}
	e.verified = nil
	if p.spec.Review && p.verdict == Satisfied && (!p.spec.Assesses || e.verifiedBeforeLocked(p)) {
		e.verified = &stand
	}
	if confirmed := e.confirmedLocked(p); p.spec.Assesses && p.verdict == NeedsCorrection && len(confirmed) > 0 {
		// Root confirmed what its tree found: the bound model has a material
		// problem, which no Pass of this run can correct. A Pass that only
		// disputed findings keep from being clean is reassessed instead.
		shown := confirmed[:min(len(confirmed), maxListedNames)]
		e.reject(fmt.Sprintf("%s%s (%d confirmed)", ModelAssessmentReason, strings.Join(shown, "; "), len(confirmed)))
	}
	record := p.recordLocked()
	_ = e.record(Event{Kind: "pass.sealed", Pass: p.id, Data: map[string]any{"purpose": p.spec.Purpose, "verdict": p.verdict, "tree": p.sealedOn, "managers": record.Managers}})
	candidate := p.candidate
	p.candidate = ""
	e.mu.Unlock()
	view := e.cfg.Registry(stand)
	_ = e.record(Event{Kind: "registry.view", Pass: p.id, Data: view})
	if candidate != "" {
		e.closeVolume(candidate)
	}
	return PassResult{Record: record, Verdict: record.Verdict, Corrections: e.corrections(!p.spec.Review)}
}

// gate runs the Model check and the File check on a whole stand and ends the
// run Failed when either reports a finding. at says where, such as "at the
// end" of a Pass.
func (e *engine) gate(p *pass, stand snapshot.Tree, kind string) bool {
	model := e.cfg.ModelCheck(stand)
	files := e.cfg.FileCheck(stand)
	data := map[string]any{"tree": stand.Digest().String(), "modelCheck": len(model), "fileCheck": len(files)}
	var lines []string
	for _, f := range append(slices.Clone(model), files...) {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", f.Owner, f.Obligation, f.Observed))
	}
	if len(lines) > 0 {
		data["findings"] = lines[:min(len(lines), 200)]
	}
	_ = e.record(Event{Kind: "gate", Pass: p.id, Data: map[string]any{"at": kind, "result": data}})
	if len(lines) == 0 {
		return true
	}
	shown := lines[:min(len(lines), maxListedNames)]
	e.reject(fmt.Sprintf("the Model check or File check failed %s of %s (%d findings): %s", kind, p.id, len(lines), strings.Join(shown, "; ")))
	return false
}

// overlay composes a stand from the accepted stand and a live tree: every
// path the owner owns comes from the live tree, everything else from the
// accepted stand.
func overlay(accepted, live snapshot.Tree, owner string, ownerOf func(snapshot.Path) string) (snapshot.Tree, error) {
	var entries []snapshot.Entry
	for _, entry := range accepted.Entries() {
		if ownerOf(entry.Path()) != owner {
			entries = append(entries, entry)
		}
	}
	for _, entry := range live.Entries() {
		if ownerOf(entry.Path()) == owner {
			entries = append(entries, entry)
		}
	}
	return snapshot.NewTree(entries)
}

func listNames(names []string) string {
	shown := names[:min(len(names), maxListedNames)]
	quoted := make([]string, len(shown))
	for i, n := range shown {
		quoted[i] = fmt.Sprintf("%q", n)
	}
	text := strings.Join(quoted, ", ")
	if len(names) > len(shown) {
		text += fmt.Sprintf(" and %d more", len(names)-len(shown))
	}
	return text
}

// verifiedBeforeLocked reports whether a clean independent Verify of the same
// stand precedes the Model assessment p with only Model assessments between
// them: a clean assessment of a verified candidate keeps it verified.
func (e *engine) verifiedBeforeLocked(p *pass) bool {
	for i := p.number - 2; i >= 0; i-- {
		if q := e.passes[i]; !q.spec.Assesses {
			return q.spec.Review && q.verdict == Satisfied && q.sealedOn == p.sealedOn
		}
	}
	return false
}

// confirmedLocked names the confirmed findings of a sealed Pass: open, genuine,
// raised by the team, not disputed in it and no mere suspicion, each with its
// correction class.
func (e *engine) confirmedLocked(p *pass) []string {
	var lines []string
	for _, f := range e.findings {
		if f.Blocking && !f.Closed && f.Kind != Suspected && f.RaisedBy != MarkitectFinding && !slices.ContainsFunc(f.Disputes, func(d Dispute) bool { return d.Pass == p.id }) {
			lines = append(lines, fmt.Sprintf("%s [%s, %s] %s: %s", f.ID, f.Owner, f.Correction, f.Obligation, shorten(f.Observed, 400)))
		}
	}
	return lines
}

// shorten cuts a text for a reason to at most n bytes at a rune boundary and
// marks the cut; the complete text stays in the run's record.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + " [...]"
}
