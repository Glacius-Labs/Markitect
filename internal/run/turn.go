package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
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
	waitOn      *waitTarget
	progress    string
	decided     bool
	revised     bool
	delegations []*helperRun
	// demands are the decisions and answers the turn's prompt asked for; only
	// those count as left open when the turn ends.
	demands []string
	// helperSummary is a helper's reported result.
	helperSummary string
	// blocked says that the turn started while the Manager's productive work
	// was blocked: a communication-only turn.
	blocked bool
	// mode is what the turn was admitted for (TurnRecord.Mode).
	mode string
	// context says how the turn received its standing context
	// (TurnRecord.Context).
	context string
	// probes counts the probes the turn ran; deadline is when its timeout ends
	// it at the latest.
	probes   int
	deadline time.Time
}

// acceptedOf returns a Manager's accepted basis or result of its Pass.
func acceptedOf(m *member, stage string) *proposal {
	if stage == stageNameResult {
		return m.result
	}
	return m.basis
}

// turnMode names what a Manager's next turn is admitted for.
func turnMode(m *member) string {
	switch {
	case m.p.spec.Review:
		return "review"
	case m.stage == stageBlocked:
		return "communication"
	case m.stage == stageOwn || m.stage == stageIntegrate:
		return "work"
	}
	return "coordination"
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
	standing := !m.introduced || m.conversation == "" || m.rehydrate
	if review {
		role, kind, profile, home, resume, letter = "reviewer", VerifyTurn, m.VerifyProfile, m.reviewHome, m.reviewConversation, "r"
		standing = m.reviewConversation == "" || m.reviewRehydrate
	}
	ts := &turnState{m: m, id: fmt.Sprintf("%s-%s-%s-%s%d", e.cfg.RunID, p.id, m.slug, letter, m.turns), role: role, kind: kind, demands: e.demandsLocked(m), blocked: m.stage == stageBlocked, mode: turnMode(m),
		deadline: e.now().Add(e.cfg.Budget.TurnTimeout), context: contextDelivery(standing, resume)}
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
	// A compacted or not continued conversation may have lost the standing
	// context; one that received it again holds it.
	lost := res.Compacted || turn.Resume != "" && res.Conversation != "" && res.Conversation != turn.Resume
	switch {
	case lost && m.p.spec.Review:
		m.reviewRehydrate = true
	case lost:
		m.rehydrate = true
	case err == nil && ts.context != contextHistory && m.p.spec.Review:
		m.reviewRehydrate = false
	case err == nil && ts.context != contextHistory:
		m.rehydrate = false
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

// How a turn received the standing context of its Manager or reviewer.
const (
	// contextBrief: a fresh conversation, whose prompt carries it.
	contextBrief = "brief"
	// contextHistory: a continued conversation that received it earlier.
	contextHistory = "history"
	// contextRehydrated: a continued conversation that may have lost it, whose
	// prompt carries it again.
	contextRehydrated = "rehydrated"
)

func contextDelivery(standing bool, resume string) string {
	switch {
	case !standing:
		return contextHistory
	case resume != "":
		return contextRehydrated
	}
	return contextBrief
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
	m.stale = ""
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
		m.stale = ""
		e.withdrawLocked(ts, "its Manager's work was reset during the turn")
		e.recordStateLocked(m, "its turn ended after a reset")
		return
	}
	// What its result built on changed while the turn ran: its submission does
	// not stand, but the turn reported and is no empty turn.
	stale := m.stale != ""
	if stale {
		e.withdrawLocked(ts, m.stale)
		m.stale = ""
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
		if on := ts.waitOn; on != nil && !submitted {
			m.waitFor = fmt.Sprintf("the accepted %s of %s", on.stage, on.manager)
			if pr := acceptedOf(p.byPath[on.manager], on.stage); pr != nil {
				// It was accepted while the turn ran: nothing is left to wait for.
				m.waitFor = ""
				note := e.bindWaitLocked(m, *on, pr)
				m.items = append(m.items, item{text: e.renderAcceptedLocked(pr) + "\nThe submission you waited for is accepted; continue your work." + note})
			} else {
				m.waitOn = on
			}
		}
	}
	working := m.stage == stageOwn || m.stage == stageIntegrate
	// A progress summary the Manager already reported in this Pass is no
	// progress: it is not continued automatically and counts as an empty turn.
	repeated := ts.progress != "" && slices.Contains(m.progressed, ts.progress)
	if ts.progress != "" && !repeated {
		m.progressed = append(m.progressed, ts.progress)
	}
	// A Manager whose helpers still work continues when their results arrive.
	if ts.progress != "" && !repeated && ts.wait == "" && working && !submitted && len(m.helpers) == 0 {
		m.items = append(m.items, item{text: "Continue your work: " + ts.progress})
	}
	// Only what the turn was shown can be left open by it; what arrived while it
	// ran reaches its next turn.
	var demands []string
	for _, d := range e.demandsLocked(m) {
		if slices.Contains(ts.demands, d) {
			demands = append(demands, d)
		}
	}
	handled := submitted || stale || ts.wait != "" || ts.progress != "" && !repeated || len(ts.delegations) > 0 || ts.revised || ts.decided
	switch {
	case ts.wait != "" || stale || submitted && len(demands) == 0:
		m.empty = 0
	case len(demands) > 0:
		// A turn that decided some of what it was shown still has the rest
		// before it; only a turn that decided nothing counts as empty.
		if ts.decided {
			m.empty = 0
		} else {
			m.empty++
		}
		m.items = append(m.items, item{text: "Still open for you: " + strings.Join(demands, "; ") + ". Decide or answer them, or report wait for what you need first."})
	case working && !handled && len(m.helpers) > 0:
		// Its helpers report to it; their results start its next turn.
		m.empty = 0
	case repeated && working && !handled:
		m.empty++
		m.items = append(m.items, item{text: "Your progress report repeats an earlier one of this Pass, so it shows no progress. If you cannot continue before another Manager's basis or result is accepted, report wait for that Manager; if you need a decision, ask your parent; otherwise submit."})
	case working && !handled:
		m.empty++
		next := "your basis"
		if m.stage == stageIntegrate {
			next = "your complete result"
		}
		m.items = append(m.items, item{text: fmt.Sprintf("Your turn ended without a report. Continue and end the turn with report: progress when you continue later, wait when you need a named answer, child or another Manager's accepted basis or result, or %s.", next)})
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
	_ = e.record(Event{Kind: "turn.start", Pass: pass, Manager: manager, Turn: ts.id, Data: map[string]any{"role": ts.role, "kind": turn.Kind, "mode": ts.mode, "profile": turn.Profile, "workspace": turn.Workspace, "readOnly": turn.ReadOnly, "mounts": turn.Mounts, "home": turn.Home, "resume": turn.Resume,
		"context": ts.context, "prompt": textDigest(turn.Prompt), "promptBytes": len(turn.Prompt)}})
	if e.ctx.Err() != nil {
		e.mu.Lock()
		e.turns = append(e.turns, TurnRecord{ID: ts.id, Manager: manager, Role: ts.role, Pass: pass, Kind: turn.Kind, Mode: ts.mode, Started: started, Ended: e.now(), Stopped: true, Reason: "the run stopped before the turn started"})
		e.mu.Unlock()
		return TurnResult{Stopped: true}, e.ctx.Err()
	}
	res, err := e.cfg.Agents.Run(e.ctx, turn, handler)
	e.mu.Lock()
	rec := TurnRecord{ID: ts.id, Manager: manager, Role: ts.role, Pass: pass, Kind: turn.Kind, Mode: ts.mode, Context: ts.context, Compacted: res.Compacted, Started: res.Started, Ended: res.Ended, Usage: res.Usage, Stopped: res.Stopped, Reports: slices.Clone(ts.reports), Reason: res.Reason}
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
	_ = e.record(Event{Kind: "turn.end", Pass: pass, Manager: manager, Turn: ts.id, Data: map[string]any{"role": ts.role, "kind": rec.Kind, "mode": rec.Mode, "usage": rec.Usage, "stopped": rec.Stopped, "reports": rec.Reports, "probes": ts.probes, "reason": rec.Reason, "conversation": res.Conversation, "compacted": res.Compacted}})
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
			return e.lookUp(m, ts, args)
		case toolCheck:
			if m.p.spec.Writes || e.probesAvailable() {
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
