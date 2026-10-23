package run

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Message kinds of the tool send.
const (
	sendQuestion = "question"
	sendAnswer   = "answer"
	sendInfo     = "info"
)

// contactsLocked lists a Manager's contacts for send: its direct parent and
// children. Nobody outside the tree answers during a run.
func (e *engine) contactsLocked(m *mgr) []string {
	var c []string
	if m.Parent != "" {
		c = append(c, "parent")
	}
	return append(c, m.Children...)
}

// resolveContact maps a send target to a Manager path. Only direct parent and
// child edges exist.
func (e *engine) resolveContact(m *mgr, to string) (string, bool) {
	switch {
	case to == "parent" && m.Parent != "":
		return m.Parent, true
	case slices.Contains(m.Children, to):
		return to, true
	}
	return "", false
}

// closedRun is what a Root that tries to reach beyond the tree is told: in a
// Model assessment a material problem of the model is a finding of the Pass,
// in any other Pass a problem that ends the run.
func closedRun(spec PassSpec) string {
	const closed = "Nobody outside the Manager tree answers during a run, and the model cannot change within it. "
	if spec.Assesses {
		return closed + "A material decision the model leaves open without delegating it, an unusable contract, a contradiction or an erroneous statement is a genuine finding of this assessment, with the correction its guidance names; your integrated result confirms it."
	}
	return closed + "A public decision the model leaves open without delegating it, a contradiction or an error is a model problem: state the confirmed problem in the final assessment problem field, naming the Definitions concerned; the model is corrected outside the run and a new run starts."
}

// send serves the tool send: a question, an answer to a question addressed to
// the sender, or information, along one direct edge of the Manager tree. A
// cross-branch exchange travels up to the common ancestor and down again, one
// correlated question per edge.
func (e *engine) send(ctx context.Context, m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		To    string `json:"to"`
		Kind  string `json:"kind"`
		Text  string `json:"text"`
		About string `json:"about"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	text, err := toolText("text", in.Text)
	if err != nil {
		return refuse("%v", err)
	}
	e.mu.Lock()
	to, ok := e.resolveContact(m.mgr, in.To)
	if !ok {
		contacts := e.contactsLocked(m.mgr)
		e.mu.Unlock()
		if m.Parent == "" && in.To != "" && !slices.Contains(m.Children, in.To) {
			return refuse("%s", closedRun(m.p.spec))
		}
		return refuse("%q is not one of your contacts. Your contacts are: %s. Other Managers are reached through your parent or children.", in.To, strings.Join(contacts, ", "))
	}
	if target := m.p.byPath[to]; target != nil && target.stage == stageStart && in.Kind == sendQuestion {
		e.mu.Unlock()
		return refuse("%s has not started in this Pass: its actual prerequisites are being re-established. Ask it then, or share information.", to)
	}
	switch in.Kind {
	case sendQuestion:
		answer, _ := e.ask(ctx, m, ts, to, text, strings.TrimSpace(in.About))
		return answer
	case sendAnswer:
		defer e.mu.Unlock()
		return e.answerLocked(m, ts, to, text, strings.TrimSpace(in.About))
	case sendInfo:
		defer e.mu.Unlock()
		if in.About != "" {
			return refuse("Information goes to a Manager without about.")
		}
		target := m.p.byPath[to]
		_ = e.record(Event{Kind: "message", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"from": m.Path, "to": to, "text": text}})
		e.deliverLocked(target, item{kind: itemMessage, text: fmt.Sprintf("Message from %s: %s", relation(m.mgr, target.mgr), text)})
		e.scheduleLocked()
		return ToolAnswer{Text: "Sent to " + to + "." + e.takeItemsLocked(m, ts)}
	}
	e.mu.Unlock()
	return refuse("kind must be question, answer or info.")
}

// relation describes the sender as its recipient sees it.
func relation(from, to *mgr) string {
	if to.Parent == from.Path {
		return "your parent " + contactOf(from)
	}
	return "your child " + contactOf(from)
}

// ask records a question to an adjacent Manager. about correlates a forwarded exchange: it names an open question addressed to
// the sender that this one serves. It is called with the lock held and
// releases it. It returns the new question's identity.
func (e *engine) ask(ctx context.Context, m *member, ts *turnState, to, text, about string) (ToolAnswer, string) {
	if about != "" {
		q := e.questionLocked(about)
		if q == nil || q.To != m.Path || q.Answered || q.Lapsed {
			e.mu.Unlock()
			return refuse("about names an open question addressed to you; %q is none.", about), ""
		}
	}
	e.nextQuestion++
	q := &Question{ID: fmt.Sprintf("Q%d", e.nextQuestion), From: m.Path, To: to, About: about, Pass: m.p.id, Text: text, Asked: e.now()}
	e.questions = append(e.questions, q)
	_ = e.record(Event{Kind: "question.asked", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: *q})
	target := m.p.byPath[to]
	e.deliverLocked(target, item{kind: itemMessage, text: fmt.Sprintf("Question %s from %s: %s\n  Answer it with send kind answer, about %s; or ask your own contact first with about %s.", q.ID, relation(m.mgr, target.mgr), text, q.ID, q.ID)})
	e.scheduleLocked()
	answer := fmt.Sprintf("Asked %s. The answer reaches you as an item; use wait with for %s when you cannot continue without it.%s", q.ID, q.ID, e.takeItemsLocked(m, ts))
	e.mu.Unlock()
	return ToolAnswer{Text: answer}, q.ID
}

// answerLocked records an answer to an open question addressed to the sender;
// it travels back along the edge the question came.
func (e *engine) answerLocked(m *member, ts *turnState, to, text, about string) ToolAnswer {
	q := e.questionLocked(about)
	switch {
	case q == nil || q.To != m.Path:
		return refuse("about names a question addressed to you; %q is none.", about)
	case q.Answered || q.Lapsed:
		return refuse("%s is already answered or lapsed.", q.ID)
	case q.From != to:
		return refuse("The answer to %s goes back to its asker %s.", q.ID, q.From)
	}
	q.Answer, q.Answered, q.AnsweredAt = text, true, e.now()
	_ = e.record(Event{Kind: "question.answered", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: *q})
	asker := e.cur.byPath[q.From]
	e.deliverLocked(asker, item{kind: itemMessage, text: fmt.Sprintf("Answer to your question %s (%q) from %s: %s", q.ID, q.Text, relation(m.mgr, asker.mgr), text)})
	e.scheduleLocked()
	return ToolAnswer{Text: "Answered " + q.ID + "." + e.takeItemsLocked(m, ts)}
}

// escalate serves the tool escalate: a classified problem and the decision the
// Manager needs, sent as a question to its parent. It adds no route of its own;
// above Root there is none.
func (e *engine) escalate(ctx context.Context, m *member, ts *turnState, args json.RawMessage) ToolAnswer {
	var in struct {
		Category string `json:"category"`
		Text     string `json:"text"`
	}
	if err := decodeStrict(args, &in); err != nil {
		return refuse("%v", err)
	}
	if !slices.Contains(problemCategories, in.Category) {
		return refuse("category must be one of %s.", strings.Join(problemCategories, ", "))
	}
	text, err := toolText("text", in.Text)
	if err != nil {
		return refuse("%v", err)
	}
	e.mu.Lock()
	to := m.Parent
	if to == "" {
		e.mu.Unlock()
		return refuse("%s", closedRun(m.p.spec))
	}
	x := e.addProblemLocked(Problem{Manager: m.Path, Category: in.Category, Text: text, Escalated: true})
	answer, id := e.ask(ctx, m, ts, to, fmt.Sprintf("Escalated %s problem: %s", in.Category, text), "")
	e.mu.Lock()
	x.Question = id
	e.mu.Unlock()
	return answer
}

// takeItemsLocked returns a Manager's undelivered items as tool answer text
// and counts them as delivered.
func (e *engine) takeItemsLocked(m *member, ts *turnState) string {
	e.pruneLocked(m)
	// The assignment and, in a turn that started blocked, everything but
	// messages stays for the turn that does the work.
	stays := func(it item) bool {
		return it.kind == itemAssignment || ts.blocked && it.kind != itemMessage
	}
	var texts []string
	var taken []item
	for _, it := range m.items {
		if !stays(it) {
			texts = append(texts, it.text)
			taken = append(taken, it)
		}
	}
	if len(texts) == 0 {
		return " No new items for you."
	}
	m.consumed = append(m.consumed, taken...)
	m.items = slices.DeleteFunc(m.items, func(it item) bool { return !stays(it) })
	// These obligations were delivered during this Turn, just like those in
	// its initial prompt. Retain them for the end-of-Turn reminder if undecided.
	// A blocked Turn only drains messages; productive items remain queued.
	if !ts.blocked {
		for _, d := range e.demandsLocked(m) {
			if !slices.Contains(ts.demands, d) {
				ts.demands = append(ts.demands, d)
			}
		}
	}
	var b strings.Builder
	list(&b, "New items for you:", texts)
	return " " + b.String()
}
