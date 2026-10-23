package run

import (
	"fmt"
	"slices"
)

// binding is a bound dependency of a Pass with BindWaits: the accepted basis or
// result of another Manager that one event of a Manager's work builds on, its
// basis (eventBasis) or its complete result (eventResult), at the accepted
// version proposal. A revision or reopening of that submission resets the
// Manager; a basis it built its own basis on is a start condition of its
// restart. An empty proposal awaits the next accepted version: until then the
// Manager's complete result is refused.
type binding struct {
	target   waitTarget
	serves   int
	proposal string
}

// source is the event a binding constrains: the start for a binding its basis
// builds on (like a declared plan dependency), the result otherwise.
func (b binding) source(manager string) depNode {
	if b.serves == eventBasis {
		return depNode{manager, eventStart}
	}
	return depNode{manager, eventResult}
}

// bindableLocked binds actual dependencies, including ancestor contracts.
// Descendant results already participate in mandatory parent integration.
func (e *engine) bindableLocked(m *member, target waitTarget) bool {
	return m.p.spec.BindWaits && target.manager != m.Path && !(target.stage == stageNameResult && e.within(target.manager, m.Path))
}

// servesLocked is the event of m's work that a wait reported now serves.
func servesLocked(m *member) int {
	if m.stage >= stageDelegated {
		return eventResult
	}
	return eventBasis
}

// bindCycleLocked reports the cycle a new binding of m would close among the
// start, basis and result events of the Pass: the explicit-dependency and integration
// order, the accepted plans' dependencies for this purpose and every binding,
// whether or not its events already happened, because a reset can undo them.
func (e *engine) bindCycleLocked(m *member, b binding) string {
	p := m.p
	edges := dependencyEdges(e.team, p.spec.Purpose, e.acceptedPlansLocked(p))
	add := func(x *member, y binding) {
		from := y.source(x.Path)
		edges[from] = append(edges[from], depNode{y.target.manager, eventOf(y.target.stage)})
	}
	for _, x := range p.members {
		for _, y := range x.bound {
			add(x, y)
		}
	}
	add(m, b)
	from := b.source(m.Path)
	return describeCycle(findCycle(e.team, edges, &from))
}

// bindLocked binds m's work to the accepted submission pr of target, unless the
// binding would close a cycle; it returns that cycle, or empty when it bound.
// A binding replaces m's earlier one on the same target and keeps its start
// condition when the earlier one served m's basis. A binding made in a turn that
// later fails stays: it can only cause an additional reset, never a missed one.
func (e *engine) bindLocked(m *member, target waitTarget, pr *proposal) string {
	b := binding{target: target, serves: servesLocked(m), proposal: pr.id}
	for _, x := range m.bound {
		if x.target == target {
			b.serves = min(b.serves, x.serves)
		}
	}
	if cycle := e.bindCycleLocked(m, b); cycle != "" {
		_ = e.record(Event{Kind: "dependency.unbound", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"on": target.manager, "stage": target.stage, "proposal": pr.id, "cycle": cycle}})
		return cycle
	}
	m.bound = slices.DeleteFunc(m.bound, func(x binding) bool { return x.target == target })
	m.bound = append(m.bound, b)
	e.recordBindingLocked(m, b, pr, false)
	return ""
}

// recordBindingLocked records a binding with the version and stand it is bound
// to.
func (e *engine) recordBindingLocked(m *member, b binding, pr *proposal, rebound bool) {
	stand := m.p.accepted
	if m.p.spec.Writes {
		stand = pr.composite
	}
	serves := stageNameBasis
	if b.serves == eventResult {
		serves = stageNameResult
	}
	data := map[string]any{"on": b.target.manager, "stage": b.target.stage, "serves": serves, "proposal": pr.id, "stand": stand.Digest().String()}
	if rebound {
		data["rebound"] = true
	}
	_ = e.record(Event{Kind: "dependency.bound", Pass: m.p.id, Manager: m.Path, Data: data})
}

// bindWaitLocked binds a satisfied wait of m on the accepted submission pr when
// the Pass binds waits; a binding that would close a cycle is skipped, recorded
// and named to m.
func (e *engine) bindWaitLocked(m *member, target waitTarget, pr *proposal) string {
	if !e.bindableLocked(m, target) {
		return ""
	}
	if cycle := e.bindLocked(m, target, pr); cycle != "" {
		return fmt.Sprintf("\nIt is not recorded as your dependency, because that would close a cycle: %s.", cycle)
	}
	return ""
}

// rebindLocked moves the start bindings of a Manager that starts its productive
// work again after a reset to the accepted versions it now builds on.
func (e *engine) rebindLocked(m *member) {
	for i, b := range m.bound {
		if b.serves != eventBasis {
			continue
		}
		if pr := acceptedOf(m.p.byPath[b.target.manager], b.target.stage); pr != nil && pr.id != b.proposal {
			m.bound[i].proposal = pr.id
			e.recordBindingLocked(m, m.bound[i], pr, true)
		}
	}
}

// targetOf names m's basis or, when basis is false, its result.
func targetOf(m string, basis bool) waitTarget {
	if basis {
		return waitTarget{manager: m, stage: stageNameBasis}
	}
	return waitTarget{manager: m, stage: stageNameResult}
}

// resultConsumersLocked returns the Managers whose result alone is bound to m's
// basis or, when basis is false, to its result. Those whose basis is bound to
// it are start-condition consumers (dependenciesLocked) and are reset.
func (e *engine) resultConsumersLocked(m *member, basis bool) []*member {
	target := targetOf(m.Path, basis)
	var out []*member
	for _, z := range m.p.members {
		if slices.ContainsFunc(z.bound, func(b binding) bool { return b.target == target && b.serves == eventResult }) {
			out = append(out, z)
		}
	}
	return out
}

// unsettleResultLocked returns a Manager whose complete result built on a
// revised or reopened submission of owner to its integration. Its submitted
// result is invalidated, or its accepted result reopened together with that
// result's consumers and its parent's integration; a submission of a turn that
// runs meanwhile is withdrawn when the turn ends. Its basis and its children's
// work stay. The binding awaits the new version, which its next complete result
// needs.
func (e *engine) unsettleResultLocked(z *member, owner string, basis bool, why string) {
	target := targetOf(owner, basis)
	i := slices.IndexFunc(z.bound, func(b binding) bool { return b.target == target && b.serves == eventResult })
	if i < 0 {
		return
	}
	z.bound[i].proposal = ""
	if z.stage < stageDelegated {
		// It reworks its basis; its result is still to come.
		return
	}
	if z.running {
		z.stale = why
	}
	switch z.stage {
	case stageResult:
		if pr := z.pending; pr != nil {
			pr.status = proposalInvalidated
			_ = e.record(Event{Kind: "proposal.invalidated", Pass: z.p.id, Manager: z.Path, Data: map[string]any{"proposal": pr.id, "why": why}})
			e.voidDispositionsLocked(pr, why)
		}
		z.pending, z.stage = nil, stageIntegrate
	case stageDone:
		old := z.result
		old.status = proposalReopened
		z.result, z.stage = nil, stageIntegrate
		_ = e.record(Event{Kind: "result.reopened", Pass: z.p.id, Manager: z.Path, Data: map[string]any{"proposal": old.id, "by": "markitect", "reason": why}})
		e.voidDispositionsLocked(old, why)
		e.invalidateConsumersLocked(z, false, why)
		e.unsettleParentLocked(z, why)
	}
	e.deliverLocked(z, item{text: fmt.Sprintf("The accepted %s of %s that your complete result builds on is no longer valid, because %s. Your basis and your children's work stay: your complete result waits for its new version; integrate that one.", target.stage, owner, why)})
}

// awaitedLocked names the first submission m's complete result builds on that
// is not accepted again yet, or is empty.
func (e *engine) awaitedLocked(m *member) string {
	for _, b := range m.bound {
		if b.serves == eventResult && b.proposal == "" && acceptedOf(m.p.byPath[b.target.manager], b.target.stage) == nil {
			return fmt.Sprintf("the %s of %s", b.target.stage, b.target.manager)
		}
	}
	return ""
}

// resumeLocked binds the awaiting result bindings on owner's newly accepted
// submission pr and tells their Managers, unless a wait of theirs already
// bound it.
func (e *engine) resumeLocked(owner *member, pr *proposal) {
	target := waitTarget{manager: owner.Path, stage: pr.kind}
	for _, x := range owner.p.members {
		for i, b := range x.bound {
			if b.target != target || b.proposal != "" {
				continue
			}
			x.bound[i].proposal = pr.id
			e.recordBindingLocked(x, x.bound[i], pr, true)
			e.deliverLocked(x, item{text: e.renderAcceptedLocked(pr) + "\nThe new version your complete result builds on is accepted; integrate it."})
		}
	}
}

// settleResultBindingsLocked keeps the result bindings of a reset Manager on
// the versions accepted now and lets the others await their next version.
func (e *engine) settleResultBindingsLocked(z *member) {
	for i, b := range z.bound {
		if b.serves != eventResult {
			continue
		}
		pr := acceptedOf(z.p.byPath[b.target.manager], b.target.stage)
		switch {
		case pr == nil:
			z.bound[i].proposal = ""
		case pr.id != b.proposal:
			z.bound[i].proposal = pr.id
			e.recordBindingLocked(z, z.bound[i], pr, true)
		}
	}
}
