package run

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// executePhase lets the owners work in the shared tree until the phase is
// quiescent: no Execute turn runs, no Manager has undelivered items and no
// owner question waits for an answer. A Manager whose turn the budget refused
// does not keep the phase open: the phase ends, the stand is captured and the
// reserved turns verify it. The turns of the phase carry the number of the
// Verify round that follows. It returns early when the run stops, but only
// after every running turn returned, and reports whether the budget refused
// work without admitting any turn: no further round fits then.
func (e *engine) executePhase(round int) (stalled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executeOpen, e.execRound = true, round
	e.budgetLimited, e.phaseTurns = false, 0
	for _, m := range e.team {
		m.refused = false
	}
	e.scheduleLocked()
	for e.ctx.Err() == nil && !e.quiescentLocked() {
		e.cond.Wait()
	}
	e.executeOpen = false
	for e.running > 0 {
		e.cond.Wait()
	}
	return e.budgetLimited && e.phaseTurns == 0
}

func (e *engine) quiescentLocked() bool {
	if e.running > 0 {
		return false
	}
	for _, m := range e.team {
		e.pruneLocked(m)
		// Items of a Manager that is not released wait for its parent; those of a
		// Manager the budget refused cannot be delivered.
		if !m.disabled && m.released && !m.refused && len(m.items) > 0 {
			return false
		}
	}
	for _, q := range e.questions {
		if !q.Answered {
			return false
		}
	}
	return true
}

// scheduleLocked starts an Execute turn for every released, idle Manager with
// undelivered items, while Execute is open.
func (e *engine) scheduleLocked() {
	if !e.executeOpen || e.closing || e.ctx.Err() != nil {
		return
	}
	for _, m := range e.team {
		if m.running || m.disabled || !m.released {
			continue
		}
		e.pruneLocked(m)
		if len(m.items) == 0 {
			continue
		}
		// Keep room for the Verify round that follows the phase and for the
		// first Execute turn of every Manager that has not run yet.
		reserve := e.executeReserveLocked(m)
		if !e.reserveTurnLocked(reserve) {
			m.refused = true
			if !e.budgetLimited {
				e.budgetLimited = true
				_ = e.record(Event{Kind: "execute.budget_limit", Manager: m.Path, Round: e.execRound, Data: map[string]any{"turns": e.turnCount, "agentTurns": e.cfg.Budget.AgentTurns, "reserve": reserve}})
			}
			continue
		}
		m.refused = false
		e.phaseTurns++
		items, first := m.items, !m.executed || m.conversation == ""
		m.items, m.consumed = nil, items
		m.execCount++
		m.running, m.ran, m.execUsed = true, true, true
		ts := &turnState{m: m, kind: ExecuteTurn, id: fmt.Sprintf("%s-r%d-%s-x%d", e.cfg.RunID, e.execRound, m.slug, m.execCount), round: e.execRound}
		turn := Turn{Profile: m.ExecuteProfile, Workspace: e.work, Home: m.execHome, Resume: m.conversation, Prompt: e.executePromptLocked(m, texts(items), first), Tools: executeTools(m.Parent == "", e.cfg.CheckTool)}
		if m.Write.Enforced {
			turn.ReadOnly, turn.Mounts = !m.Write.Base, slices.Clone(m.Write.Mounts)
		}
		e.running++
		e.wg.Add(1)
		go e.executeTurn(m, ts, turn)
	}
}

// executeReserveLocked is the number of turns the budget must keep after an
// Execute turn of m: one whole Verify round and the first Execute turn of every
// other Manager that has not run yet, so that no Manager is starved of its
// first turn by the repair work of others.
func (e *engine) executeReserveLocked(m *mgr) int {
	reserve := len(e.team)
	for _, o := range e.team {
		if o != m && o.execCount == 0 && !o.disabled {
			reserve++
		}
	}
	return reserve
}

// releaseLocked releases the start of a Manager: its initial mandate item goes
// first, before any message its parent queued during its own first turn. The
// release is recorded once per Manager.
func (e *engine) releaseLocked(m *mgr, why string) {
	if m.released {
		return
	}
	m.released = true
	m.items = append([]item{{text: initialItem(m)}}, m.items...)
	data := map[string]any{"why": why}
	if m.Parent != "" {
		data["parent"] = m.Parent
	}
	_ = e.record(Event{Kind: "manager.release", Manager: m.Path, Round: e.execRound, Data: data})
}

func (e *engine) executeTurn(m *mgr, ts *turnState, turn Turn) {
	defer e.wg.Done()
	res, err := e.runTurn(ts, turn, e.executeHandler(m, ts))
	e.mu.Lock()
	defer e.mu.Unlock()
	m.running = false
	e.running--
	if res.Conversation != "" {
		m.conversation = res.Conversation
	}
	if err == nil {
		m.executed = true
	}
	if e.ctx.Err() == nil {
		switch {
		case err == nil:
			m.failures = 0
		case res.Stopped:
			e.failedLocked(m, err)
		}
		e.reportLocked(m, ts, err)
		// The children start when the parent's first turn ended without error. A
		// failed first turn that is retried does not release them yet; a disabled
		// parent has ended its first turn for good.
		if err == nil || m.disabled {
			for _, c := range m.Children {
				e.releaseLocked(e.byPath[c], "the first Execute turn of its parent ended")
			}
		}
	}
	e.cond.Broadcast()
	e.scheduleLocked()
}

// reportLocked delivers a child's summary to its direct parent when the child's
// turn finished with status done, so that the parent integrates the result. A
// turn that ended without finish counts as done, with a note that no summary
// was given.
func (e *engine) reportLocked(m *mgr, ts *turnState, err error) {
	if err != nil || m.Parent == "" || ts.finished && ts.status != "done" {
		return
	}
	p := e.byPath[m.Parent]
	if p.disabled {
		return
	}
	summary := ts.summary
	if !ts.finished {
		summary = "no summary was given, the turn ended without finish"
	}
	p.items = append(p.items, item{text: fmt.Sprintf("Your child %s reports done: %s", contact(m), summary)})
	_ = e.record(Event{Kind: "child.report", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"from": m.Path, "to": p.Path, "summary": summary}})
}

// failedLocked handles a turn that ended with an error after its container
// stopped: the Manager is told once more and stopped after repeated failures.
// The items the failed turn consumed are delivered again, whether or not the
// provider kept a conversation: it may not have processed them. A Manager
// without a finished Execute turn also gets its mandate again.
func (e *engine) failedLocked(m *mgr, err error) {
	m.failures++
	if m.failures >= maxExecuteFailures {
		m.disabled = true
		_ = e.record(Event{Kind: "manager.disabled", Manager: m.Path, Data: map[string]any{"failures": m.failures, "error": err.Error()}})
		return
	}
	m.items = append(m.items, m.consumed...)
	m.items = append(m.items, item{text: fmt.Sprintf("Note: your previous turn ended with an error (%v). Continue where you left off.", err)})
}

func (e *engine) executeHandler(m *mgr, ts *turnState) ToolHandler {
	return func(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
		switch name {
		case toolSend:
			return e.send(ctx, m, ts, args)
		case toolFinish:
			return e.finish(m, ts, args)
		case toolCheck:
			if e.cfg.CheckTool {
				return e.check(ctx, m, ts, args)
			}
		}
		return refuse("Unknown tool %q; in Execute %q and %q are available.", name, toolSend, toolFinish)
	}
}

func (e *engine) finish(m *mgr, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	if in.Status != "done" && in.Status != "waiting" {
		return refuse("status must be \"done\" or \"waiting\".")
	}
	summary, err := toolText("summary", in.Summary)
	if err != nil {
		return refuse("%v", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if ts.finished {
		return refuse("This turn is already finished.")
	}
	ts.finished, ts.status, ts.summary = true, in.Status, summary
	_ = e.record(Event{Kind: "turn.finish", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"status": in.Status, "summary": summary}})
	return ToolAnswer{Text: "Recorded. Your turn ends."}
}

func (e *engine) contactsLocked(m *mgr) string {
	var c []string
	if m.Parent != "" {
		c = append(c, `"parent"`)
	}
	for _, child := range m.Children {
		c = append(c, fmt.Sprintf("%q", child))
	}
	if m.Parent == "" {
		c = append(c, `"owner"`)
	}
	if len(c) == 0 {
		return "none"
	}
	return strings.Join(c, ", ")
}

func (e *engine) send(ctx context.Context, m *mgr, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		To   string `json:"to"`
		Text string `json:"text"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	text, err := toolText("text", in.Text)
	if err != nil {
		return refuse("%v", err)
	}
	e.mu.Lock()
	if ts.finished {
		e.mu.Unlock()
		return refuse("This turn is already finished; no further calls are accepted.")
	}
	if in.To == "owner" && m.Parent == "" {
		e.mu.Unlock()
		return e.askOwner(ctx, m, ts, text)
	}
	defer e.mu.Unlock()
	var to *mgr
	switch {
	case in.To == "parent" && m.Parent != "":
		to = e.byPath[m.Parent]
	default:
		for _, c := range m.Children {
			if c == in.To {
				to = e.byPath[c]
			}
		}
	}
	if to == nil {
		return refuse("%q is not one of your contacts. Your contacts are: %s.", in.To, e.contactsLocked(m))
	}
	if to.disabled {
		return refuse("%s stopped after repeated failed turns and cannot receive messages.", to.Path)
	}
	from := "your child " + contact(m)
	if to.Parent == m.Path {
		from = "your parent " + contact(m)
	}
	to.items = append(to.items, item{text: fmt.Sprintf("Message from %s: %s", from, text)})
	_ = e.record(Event{Kind: "message", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: map[string]any{"from": m.Path, "to": to.Path, "text": text}})
	e.scheduleLocked()
	e.cond.Broadcast()
	return ToolAnswer{Text: "Sent to " + to.Path + "." + e.takeItemsLocked(m)}
}

// takeItemsLocked returns the Manager's undelivered items as tool answer text
// and counts them as delivered.
func (e *engine) takeItemsLocked(m *mgr) string {
	e.pruneLocked(m)
	if len(m.items) == 0 {
		return " No new items for you."
	}
	var b strings.Builder
	list(&b, "New items for you:", texts(m.items))
	m.consumed = append(m.consumed, m.items...)
	m.items = nil
	return " " + b.String()
}

func (e *engine) askOwner(ctx context.Context, m *mgr, ts *turnState, text string) ToolAnswer {
	if e.cfg.Owner == nil {
		return refuse("No human can answer in this run. Decide within the bound owner decisions, or report the open question in your finish summary.")
	}
	e.mu.Lock()
	e.nextQuest++
	q := &Question{ID: fmt.Sprintf("Q%d", e.nextQuest), Text: text, Asked: e.now()}
	e.mu.Unlock()
	if err := e.cfg.Owner.Ask(ctx, *q); err != nil {
		return refuse("The owner could not be reached: %v. Decide within the bound owner decisions, or report the open question.", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.questions = append(e.questions, q)
	_ = e.record(Event{Kind: "question", Manager: m.Path, Turn: ts.id, Round: ts.round, Data: *q})
	if !e.polling && !e.closing {
		e.polling = true
		e.wg.Add(1)
		go e.pollOwner()
	}
	return ToolAnswer{Text: fmt.Sprintf("Question %s was sent to the owner. The answer is delivered to you when it arrives.%s", q.ID, e.takeItemsLocked(m))}
}

// pollOwner asks the owner channel for answers until the run ends.
func (e *engine) pollOwner() {
	defer e.wg.Done()
	tick := time.NewTicker(ownerPollInterval)
	defer tick.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-tick.C:
		}
		e.mu.Lock()
		var open []string
		for _, q := range e.questions {
			if !q.Answered {
				open = append(open, q.ID)
			}
		}
		e.mu.Unlock()
		for _, id := range open {
			answer, ok, err := e.cfg.Owner.Answer(e.ctx, id)
			if err != nil {
				if e.ctx.Err() == nil {
					_ = e.record(Event{Kind: "owner.error", Data: map[string]any{"question": id, "error": err.Error()}})
				}
				continue
			}
			if !ok {
				continue
			}
			e.mu.Lock()
			for _, q := range e.questions {
				if q.ID != id || q.Answered {
					continue
				}
				q.Answer, q.Answered = answer, true
				_ = e.record(Event{Kind: "answer", Manager: e.team[0].Path, Data: *q})
				root := e.team[0]
				root.items = append(root.items, item{text: fmt.Sprintf("Owner answer to your question %s (%q): %s", q.ID, q.Text, answer)})
			}
			e.scheduleLocked()
			e.cond.Broadcast()
			e.mu.Unlock()
		}
	}
}
