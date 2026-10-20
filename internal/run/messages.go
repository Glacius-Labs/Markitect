package run

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Message kinds of the tool send.
const (
	sendQuestion = "question"
	sendAnswer   = "answer"
	sendInfo     = "info"
)

// contactsLocked lists a Manager's contacts for send: its direct parent and
// children, and for Root the user when an owner channel exists.
func (e *engine) contactsLocked(m *mgr) []string {
	var c []string
	if m.Parent != "" {
		c = append(c, "parent")
	}
	c = append(c, m.Children...)
	if m.Parent == "" && e.cfg.Owner != nil {
		c = append(c, User)
	}
	return c
}

// resolveContact maps a send target to a Manager path, or User. Only direct
// parent and child edges, and Root's channel to the user, exist.
func (e *engine) resolveContact(m *mgr, to string) (string, bool) {
	switch {
	case to == "parent" && m.Parent != "":
		return m.Parent, true
	case to == User && m.Parent == "" && e.cfg.Owner != nil:
		return User, true
	case slices.Contains(m.Children, to):
		return to, true
	}
	return "", false
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
		if in.To == User && m.Parent == "" {
			return refuse("Nobody answers questions in this run. Decide within the model and the bound owner decisions; a model that is erroneous or contradictory is reported with report kind problem, category model, abort true.")
		}
		return refuse("%q is not one of your contacts. Your contacts are: %s. Other Managers are reached through your parent or children.", in.To, strings.Join(contacts, ", "))
	}
	if target := m.p.byPath[to]; target != nil && target.stage == stageStart && in.Kind == sendQuestion {
		e.mu.Unlock()
		return refuse("%s has not started in this Pass: it starts once its parent's basis is accepted. Ask it then, or share information.", to)
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
		if to == User || in.About != "" {
			return refuse("Information goes to a Manager without about; ask the user with kind question.")
		}
		target := m.p.byPath[to]
		_ = e.record(Event{Kind: "message", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: map[string]any{"from": m.Path, "to": to, "text": text}})
		e.deliverLocked(target, item{kind: itemMessage, text: fmt.Sprintf("Message from %s: %s", relation(m.mgr, target.mgr), text)})
		e.scheduleLocked()
		return ToolAnswer{Text: "Sent to " + to + "." + e.takeItemsLocked(m)}
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

// ask records a question to an adjacent Manager or, for Root, to the user.
// about correlates a forwarded exchange: it names an open question addressed to
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
	if to != User {
		e.questions = append(e.questions, q)
		_ = e.record(Event{Kind: "question.asked", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: *q})
		target := m.p.byPath[to]
		e.deliverLocked(target, item{kind: itemMessage, text: fmt.Sprintf("Question %s from %s: %s\n  Answer it with send kind answer, about %s; or ask your own contact first with about %s.", q.ID, relation(m.mgr, target.mgr), text, q.ID, q.ID)})
		e.scheduleLocked()
		answer := fmt.Sprintf("Asked %s. The answer reaches you as an item; report wait with for %s when you cannot continue without it.%s", q.ID, q.ID, e.takeItemsLocked(m))
		e.mu.Unlock()
		return ToolAnswer{Text: answer}, q.ID
	}
	e.mu.Unlock()
	if err := e.cfg.Owner.Ask(ctx, *q); err != nil {
		return refuse("The user could not be reached: %v. Decide within the model and the bound owner decisions, or report a problem.", err), ""
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.questions = append(e.questions, q)
	_ = e.record(Event{Kind: "question.asked", Pass: m.p.id, Manager: m.Path, Turn: ts.id, Data: *q})
	if !e.polling && !e.closing {
		e.polling = true
		e.wg.Add(1)
		go e.pollOwner()
	}
	return ToolAnswer{Text: fmt.Sprintf("Asked the user %s. The answer reaches you as an item; report wait with for %s when you cannot continue without it.%s", q.ID, q.ID, e.takeItemsLocked(m))}, q.ID
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
	return ToolAnswer{Text: "Answered " + q.ID + "." + e.takeItemsLocked(m)}
}

// escalate serves the tool escalate: a classified problem and the decision the
// Manager needs, sent as a question to its parent (Root's to the user). It adds
// no route of its own.
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
		if e.cfg.Owner == nil {
			e.mu.Unlock()
			return refuse("Nobody answers questions in this run. Decide within the model and the bound owner decisions; a model that is erroneous or contradictory is reported with report kind problem, category model, abort true.")
		}
		to = User
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
func (e *engine) takeItemsLocked(m *member) string {
	e.pruneLocked(m)
	var texts []string
	for _, it := range m.items {
		if it.kind == itemAssignment {
			continue
		}
		texts = append(texts, it.text)
	}
	if len(texts) == 0 {
		return " No new items for you."
	}
	m.consumed = append(m.consumed, m.items...)
	m.items = slices.DeleteFunc(m.items, func(it item) bool { return it.kind != itemAssignment })
	var b strings.Builder
	list(&b, "New items for you:", texts)
	return " " + b.String()
}

// pollOwner asks the owner channel for answers until the run ends. An answer
// reaches Root as an item.
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
			if q.To == User && !q.Answered && !q.Lapsed {
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
			if q := e.questionLocked(id); q != nil && !q.Answered && !q.Lapsed {
				q.Answer, q.Answered, q.AnsweredAt = answer, true, e.now()
				_ = e.record(Event{Kind: "question.answered", Pass: q.Pass, Manager: User, Data: *q})
				if e.cur != nil && !e.cur.sealed {
					e.deliverLocked(e.cur.members[0], item{kind: itemMessage, text: fmt.Sprintf("Answer of the user to your question %s (%q): %s", q.ID, q.Text, answer)})
				}
				e.scheduleLocked()
			}
			e.cond.Broadcast()
			e.mu.Unlock()
		}
	}
}
