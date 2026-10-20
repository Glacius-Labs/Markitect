package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// turnState is what the handlers of one turn share with the Orchestrator. Its
// fields are guarded by engine.mu.
type turnState struct {
	m    *member
	h    *helperRun
	id   string
	role string
	kind TurnKind
	// reports lists the kinds of the turn's accepted report calls.
	reports     []string
	proposal    *proposal
	wait        string
	progress    string
	decided     bool
	revised     bool
	delegations []*helperRun
	// helperSummary is a helper's reported result.
	helperSummary string
}

// startTurnLocked starts a turn of a Manager that delivers all its items.
func (e *engine) startTurnLocked(m *member) {
	p := m.p
	if !e.reserveTurnLocked() {
		e.addProblemLocked(Problem{Manager: m.Path, Category: "technical", Text: "no Agent turn is left in the budget"})
		e.halt(fmt.Sprintf("the agent turn budget of %d ended", e.cfg.Budget.AgentTurns))
		return
	}
	items := m.items
	m.items = nil
	if m.stage == stageBlocked {
		// A communication-only turn leaves the findings to correct for the
		// productive work.
		var deliver, keep []item
		for _, it := range items {
			if it.kind == itemFinding {
				keep = append(keep, it)
			} else {
				deliver = append(deliver, it)
			}
		}
		items, m.items = deliver, keep
	}
	m.consumed = items
	m.running = true
	m.turns++
	p.running++
	review := p.spec.Review
	role, kind, profile, home, resume, letter := "manager", ExecuteTurn, m.ExecuteProfile, m.workHome, m.conversation, "m"
	standing := !m.introduced || m.conversation == ""
	if review {
		role, kind, profile, home, resume, letter = "reviewer", VerifyTurn, m.VerifyProfile, m.reviewHome, m.reviewConversation, "r"
		standing = m.reviewConversation == ""
	}
	ts := &turnState{m: m, id: fmt.Sprintf("%s-%s-%s-%s%d", e.cfg.RunID, p.id, m.slug, letter, m.turns), role: role, kind: kind}
	readOnly, mounts := e.mountsLocked(m)
	turn := Turn{Kind: kind, Profile: profile, Workspace: p.volume, ReadOnly: readOnly, Mounts: mounts, Home: home, Resume: resume,
		Prompt: e.promptLocked(m, items, standing, readOnly, mounts), Tools: e.memberTools(m)}
	e.useHomeLocked(home)
	e.recordStateLocked(m, "a turn started")
	e.wg.Add(1)
	go e.memberTurn(m, ts, turn)
}

// mountsLocked returns where a Manager's next turn may write: its write plan in
// a Pass with writes while it works on its own basis or result, minus the
// subsets its helpers hold; read-only otherwise.
func (e *engine) mountsLocked(m *member) (bool, []PathMount) {
	if !m.p.spec.Writes || m.stage != stageOwn && m.stage != stageIntegrate {
		return true, nil
	}
	mounts := slices.Clone(m.Write.Mounts)
	for _, h := range m.helpers {
		for _, d := range h.paths {
			i := slices.IndexFunc(mounts, func(x PathMount) bool { return x.Path == d.Path })
			if i >= 0 {
				mounts[i].Writable = false
			} else {
				mounts = append(mounts, PathMount{Path: d.Path, File: d.File})
			}
		}
	}
	slices.SortFunc(mounts, func(a, b PathMount) int { return strings.Compare(a.Path, b.Path) })
	return !m.Write.Base, mounts
}

func (e *engine) memberTurn(m *member, ts *turnState, turn Turn) {
	defer e.wg.Done()
	res, err := e.runTurn(ts, turn, e.memberHandler(m, ts))
	e.mu.Lock()
	defer e.mu.Unlock()
	defer e.cond.Broadcast()
	m.running = false
	m.p.running--
	if res.Conversation != "" {
		if m.p.spec.Review {
			m.reviewConversation = res.Conversation
		} else {
			m.conversation = res.Conversation
		}
	}
	if e.ctx.Err() != nil {
		e.withdrawLocked(ts, "the run stopped")
		e.recordStateLocked(m, "the run stopped")
		return
	}
	if err != nil {
		if res.Stopped {
			e.turnFailedLocked(m, ts, err)
			e.scheduleLocked()
		}
		return
	}
	if !m.p.spec.Review {
		m.introduced = true
	}
	m.failures = 0
	e.turnEndedLocked(m, ts)
	e.scheduleLocked()
}

// withdrawLocked discards what a turn submitted or delegated without effect.
func (e *engine) withdrawLocked(ts *turnState, why string) {
	if pr := ts.proposal; pr != nil && pr.status == proposalSubmitted {
		pr.status = proposalWithdrawn
		_ = e.record(Event{Kind: "proposal.withdrawn", Pass: ts.m.p.id, Manager: ts.m.Path, Turn: ts.id, Data: map[string]any{"proposal": pr.id, "why": why}})
	}
	ts.delegations = nil
}

// turnFailedLocked handles a turn that ended with an error after its container
// stopped: its submission and delegations are withdrawn, the items it consumed
// are delivered again and, after repeated failures, the Manager is stopped. A
// required Manager that cannot work ends the run Incomplete.
func (e *engine) turnFailedLocked(m *member, ts *turnState, err error) {
	e.withdrawLocked(ts, "the turn failed")
	m.failures++
	if m.failures >= maxTurnFailures {
		m.disabled = true
		_ = e.record(Event{Kind: "manager.disabled", Pass: m.p.id, Manager: m.Path, Data: map[string]any{"failures": m.failures, "error": err.Error()}})
		e.addProblemLocked(Problem{Manager: m.Path, Category: "technical", Text: fmt.Sprintf("its turns failed %d times: %v", m.failures, err)})
		e.halt(fmt.Sprintf("manager %q stopped after %d failed turns: %v", m.Path, m.failures, err))
		return
	}
	m.items = append(slices.Clone(m.consumed), m.items...)
	m.items = append(m.items, item{text: fmt.Sprintf("Your previous turn ended with an error (%v); nothing it submitted was taken. Continue where you left off.", err)})
	e.recordStateLocked(m, "the turn failed")
}

// turnEndedLocked processes a turn that ended normally. A provider exit never
// completes work: a submission is now checked and decided, a wait or progress
// report sets the next state, and a turn that worked on its own contribution
// without any report, or left a decision or question to it open without a
// wait, is incomplete work that is returned to it with a note.
func (e *engine) turnEndedLocked(m *member, ts *turnState) {
	p := m.p
	if m.stage == stageStart {
		// Its work was reset while the turn ran: nothing of the turn stands.
		e.withdrawLocked(ts, "its Manager's work was reset during the turn")
		e.recordStateLocked(m, "its turn ended after a reset")
		return
	}
	submitted := false
	if pr := ts.proposal; pr != nil && pr.status == proposalSubmitted {
		wantStage := stageOwn
		if pr.kind == kindResult {
			wantStage = stageIntegrate
		}
		if m.stage == wantStage {
			submitted = true
			m.pending = pr
			m.stage = stageBasis
			if pr.kind == kindResult {
				m.stage = stageResult
			}
			if p.spec.Writes {
				m.checking = true
				p.checking++
				e.wg.Add(1)
				go e.checkSubmission(pr)
			} else {
				e.checkedLocked(pr)
			}
		} else {
			e.withdrawLocked(ts, "its Manager's work was reset during the turn")
		}
	}
	for _, h := range ts.delegations {
		m.helpers = append(m.helpers, h)
	}
	if ts.wait != "" {
		m.waitFor = ts.wait
	}
	working := m.stage == stageOwn || m.stage == stageIntegrate
	if ts.progress != "" && working && !submitted {
		m.items = append(m.items, item{text: "Continue your work: " + ts.progress})
	}
	demands := e.demandsLocked(m)
	handled := submitted || ts.wait != "" || ts.progress != "" || len(ts.delegations) > 0 || ts.revised || ts.decided
	switch {
	case ts.wait != "" || submitted && len(demands) == 0:
		m.empty = 0
	case working && !handled:
		m.empty++
		next := "your basis"
		if m.stage == stageIntegrate {
			next = "your complete result"
		}
		m.items = append(m.items, item{text: fmt.Sprintf("Your turn ended without a report. Continue and end the turn with report: progress when you continue later, wait when you need a named answer or child, or %s.", next)})
	case len(demands) > 0 && !ts.decided:
		m.empty++
		m.items = append(m.items, item{text: "Still open for you: " + strings.Join(demands, "; ") + ". Decide or answer them, or report wait for what you need first."})
	default:
		m.empty = 0
	}
	if m.empty > maxEmptyTurns {
		e.addProblemLocked(Problem{Manager: m.Path, Category: "technical", Text: fmt.Sprintf("%d consecutive turns ended without the report its work needs", m.empty)})
		e.halt(fmt.Sprintf("manager %q ended %d consecutive turns without the report its work needs", m.Path, m.empty))
		return
	}
	e.recordStateLocked(m, "its turn ended")
}

// demandsLocked lists what waits for a Manager's own decision or answer:
// proposals of its children and questions addressed to it.
func (e *engine) demandsLocked(m *member) []string {
	var out []string
	for _, c := range m.Children {
		if pr := m.p.byPath[c].pending; pr != nil && pr.status == proposalChecked {
			out = append(out, fmt.Sprintf("decide %s, the %s of %s", pr.id, pr.kind, c))
		}
	}
	for _, q := range e.questions {
		if q.To == m.Path && !q.Answered && !q.Lapsed && !e.forwardedLocked(q) {
			out = append(out, fmt.Sprintf("answer %s from %s", q.ID, q.From))
		}
	}
	return out
}

// forwardedLocked reports whether the recipient of a question forwarded it
// with an open question of its own.
func (e *engine) forwardedLocked(q *Question) bool {
	for _, x := range e.questions {
		if x.About == q.ID && x.From == q.To && !x.Answered && !x.Lapsed {
			return true
		}
	}
	return false
}

// runTurn runs one Agent turn through the Agents port and records it. A turn
// whose container end was not observed stops the run: quiescence is then
// uncertain.
func (e *engine) runTurn(ts *turnState, turn Turn, handler ToolHandler) (TurnResult, error) {
	manager := ts.m.Path
	pass := ts.m.p.id
	turn.ID, turn.Manager, turn.Timeout = ts.id, manager, e.cfg.Budget.TurnTimeout
	turn.Transcript = e.cfg.Evidence.Transcript(ts.id)
	started := e.now()
	_ = e.record(Event{Kind: "turn.start", Pass: pass, Manager: manager, Turn: ts.id, Data: map[string]any{"role": ts.role, "kind": turn.Kind, "profile": turn.Profile, "workspace": turn.Workspace, "readOnly": turn.ReadOnly, "mounts": turn.Mounts, "home": turn.Home, "resume": turn.Resume}})
	if e.ctx.Err() != nil {
		e.mu.Lock()
		e.turns = append(e.turns, TurnRecord{ID: ts.id, Manager: manager, Role: ts.role, Pass: pass, Kind: turn.Kind, Started: started, Ended: e.now(), Stopped: true, Reason: "the run stopped before the turn started"})
		e.mu.Unlock()
		return TurnResult{Stopped: true}, e.ctx.Err()
	}
	res, err := e.cfg.Agents.Run(e.ctx, turn, handler)
	e.mu.Lock()
	rec := TurnRecord{ID: ts.id, Manager: manager, Role: ts.role, Pass: pass, Kind: turn.Kind, Started: res.Started, Ended: res.Ended, Usage: res.Usage, Stopped: res.Stopped, Reports: slices.Clone(ts.reports), Reason: res.Reason}
	if rec.Started.IsZero() {
		rec.Started = started
	}
	if rec.Ended.IsZero() {
		rec.Ended = e.now()
	}
	if err != nil {
		rec.Reason = strings.TrimSpace(rec.Reason + " " + err.Error())
	}
	e.turns = append(e.turns, rec)
	e.usage = e.usage.Add(res.Usage)
	_ = e.record(Event{Kind: "turn.end", Pass: pass, Manager: manager, Turn: ts.id, Data: map[string]any{"role": ts.role, "kind": rec.Kind, "usage": rec.Usage, "stopped": rec.Stopped, "reports": rec.Reports, "reason": rec.Reason, "conversation": res.Conversation}})
	e.mu.Unlock()
	if !res.Stopped {
		e.halt(fmt.Sprintf("turn %s ended without an observed container stop; quiescence is uncertain", ts.id))
	}
	return res, err
}

// memberHandler serves the tools of a Manager's or reviewer's turn.
func (e *engine) memberHandler(m *member, ts *turnState) ToolHandler {
	return func(ctx context.Context, name string, args json.RawMessage) ToolAnswer {
		switch name {
		case toolReport:
			return e.report(m, ts, args)
		case toolSend:
			return e.send(ctx, m, ts, args)
		case toolEscalate:
			return e.escalate(ctx, m, ts, args)
		case toolLookUp:
			return e.lookUp(m, args)
		case toolCheck:
			if m.p.spec.Writes {
				return e.check(ctx, m, ts, args)
			}
		case toolDelegate:
			if len(m.Helpers) > 0 {
				return e.delegate(ctx, m, ts, args)
			}
		}
		var names []string
		for _, t := range e.memberTools(m) {
			names = append(names, fmt.Sprintf("%q", t.Name))
		}
		return refuse("Unknown tool %q; you have %s.", name, strings.Join(names, ", "))
	}
}

// decodeStrict reads one tool call's arguments and refuses unknown fields and
// trailing data.
func decodeStrict(raw json.RawMessage, into any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("the arguments are not valid: %v", err)
	}
	if dec.More() {
		return fmt.Errorf("the arguments have trailing data")
	}
	return nil
}

func refuse(format string, args ...any) ToolAnswer {
	return ToolAnswer{Text: fmt.Sprintf(format, args...), Refused: true}
}

// toolText checks one free-text argument.
func toolText(name, s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "" || !utf8.ValidString(s) || strings.ContainsRune(s, 0):
		return "", fmt.Errorf("%s must be non-empty text", name)
	case len(s) > maxToolText:
		return "", fmt.Errorf("%s is longer than %d bytes", name, maxToolText)
	}
	return s, nil
}

// tail returns the last n bytes of output as valid text.
func tail(b []byte, n int) string {
	cut := false
	if len(b) > n {
		b, cut = b[len(b)-n:], true
		for len(b) > 0 && !utf8.RuneStart(b[0]) {
			b = b[1:]
		}
	}
	s := strings.ToValidUTF8(string(b), "�")
	if cut {
		s = "[output cut]\n" + s
	}
	return s
}
