package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
)

// Shared Note limits: a Note stays far below one tool reply, and a page of
// Notes fits the tool transport (concept 1.0 §19).
const (
	MaxNoteText   = 16 << 10
	NotePageCount = 100
	NotePageBytes = 128 << 10
)

// host is the tools of one agent start. Every call returns at once; it only
// records a message or a Note and wakes the schedule.
type host struct {
	s     *schedule
	n     *node
	role  agentport.Role
	tools map[agentport.Tool]bool
	// closed is set under the schedule's lock once the start has ended.
	closed bool
}

var _ agentport.Host = (*host)(nil)

func (h *host) close() {
	h.s.mu.Lock()
	h.closed = true
	h.s.mu.Unlock()
}

// enter checks that the start may call the tool and locks the schedule.
func (h *host) enter(t agentport.Tool) error {
	if !h.tools[t] {
		return fmt.Errorf("the tool %s is not yours in this start; yours are %s", t, h.list())
	}
	h.s.mu.Lock()
	if h.closed {
		h.s.mu.Unlock()
		return fmt.Errorf("this start has ended; tools answer only while it runs")
	}
	if h.s.stopped {
		h.s.mu.Unlock()
		return fmt.Errorf("the Run has stopped (%s); end your work", h.s.why)
	}
	return nil
}

func (h *host) list() string {
	var names []string
	for _, t := range []agentport.Tool{agentport.ToolAsk, agentport.ToolEscalate, agentport.ToolAnswer, agentport.ToolAssign, agentport.ToolNote, agentport.ToolNotes} {
		if h.tools[t] {
			names = append(names, string(t))
		}
	}
	return strings.Join(names, ", ")
}

func (h *host) Ask(_ context.Context, text string) (agentport.Message, error) {
	return h.up(agentport.ToolAsk, agentport.Question, text)
}

func (h *host) Escalate(_ context.Context, text string) (agentport.Message, error) {
	return h.up(agentport.ToolEscalate, agentport.Escalation, text)
}

// up sends a question or escalation to the parent. Root has no parent: its
// question is refused, and its escalation is a model defect that ends the
// Run without landing.
func (h *host) up(t agentport.Tool, kind, text string) (agentport.Message, error) {
	if strings.TrimSpace(text) == "" {
		return agentport.Message{}, fmt.Errorf("say what you %s", t)
	}
	if err := h.enter(t); err != nil {
		return agentport.Message{}, err
	}
	s, n := h.s, h.n
	defer s.mu.Unlock()
	if n.parent == nil {
		if kind == agentport.Question {
			return agentport.Message{}, fmt.Errorf("Root has no parent to ask; decide within the model, or escalate if the model itself is defective")
		}
		// The owner reads it on the Result; it reaches no one during the Run.
		m := s.deliver(agentport.Message{Kind: kind, From: n.name, To: "", Text: text})
		s.decisions = append(s.decisions, OwnerDecision{Manager: n.name, Matter: text, Recommendation: recommendation(text)})
		s.stopLocked(StateModelDefect, "Root escalated a model defect; the Run ends without landing")
		return m, nil
	}
	m := s.send(n, kind, n.parent.name, "", text)
	n.waitFor[m.ID] = true
	n.sent = true
	return m, nil
}

func (h *host) Answer(_ context.Context, to, text string) (agentport.Message, error) {
	if err := h.enter(agentport.ToolAnswer); err != nil {
		return agentport.Message{}, err
	}
	s, n := h.s, h.n
	defer s.mu.Unlock()
	q, ok := n.unanswered[to]
	if !ok {
		open := "none"
		if ids := sortedKeys(n.unanswered); len(ids) > 0 {
			open = strings.Join(ids, ", ")
		}
		return agentport.Message{}, fmt.Errorf("%q is no open question or escalation to you; open ones: %s", to, open)
	}
	delete(n.unanswered, to)
	asker := s.byName(q.From)
	delete(asker.waitFor, to)
	m := s.deliver(agentport.Message{Kind: agentport.Answer, From: n.name, To: q.From, ReplyTo: to, Text: text})
	n.sent = true
	return m, nil
}

func (h *host) Assign(_ context.Context, child, text string) (agentport.Message, error) {
	if err := h.enter(agentport.ToolAssign); err != nil {
		return agentport.Message{}, err
	}
	s, n := h.s, h.n
	defer s.mu.Unlock()
	var c *node
	var names []string
	for _, ch := range n.children {
		names = append(names, ch.name)
		if ch.name == child {
			c = ch
		}
	}
	if c == nil {
		return agentport.Message{}, fmt.Errorf("%q is no working child of yours; yours are %s", child, strings.Join(names, ", "))
	}
	c.approved = false
	m := s.deliver(agentport.Message{Kind: agentport.Assignment, From: n.name, To: c.name, Text: text})
	n.sent = true
	return m, nil
}

func (h *host) Note(_ context.Context, text string) (agentport.Note, error) {
	if len(text) > MaxNoteText {
		return agentport.Note{}, fmt.Errorf("a Note holds at most %d bytes; yours has %d. Split it into several Notes", MaxNoteText, len(text))
	}
	if strings.TrimSpace(text) == "" {
		return agentport.Note{}, fmt.Errorf("a Note needs text")
	}
	if err := h.enter(agentport.ToolNote); err != nil {
		return agentport.Note{}, err
	}
	s := h.s
	defer s.mu.Unlock()
	note := agentport.Note{ID: fmt.Sprintf("n%06d", len(s.notes)+1), Manager: h.n.name, Role: h.role, Text: text, Time: s.e.now().UTC()}
	s.notes = append(s.notes, note)
	return note, nil
}

func (h *host) Notes(_ context.Context, after string) (agentport.NotePage, error) {
	if err := h.enter(agentport.ToolNotes); err != nil {
		return agentport.NotePage{}, err
	}
	s := h.s
	defer s.mu.Unlock()
	page := agentport.NotePage{Notes: []agentport.Note{}}
	size := 0
	for _, note := range s.notes {
		if note.ID <= after {
			continue
		}
		if len(page.Notes) == NotePageCount || (len(page.Notes) > 0 && size+len(note.Text) > NotePageBytes) {
			page.Next = page.Notes[len(page.Notes)-1].ID
			break
		}
		page.Notes = append(page.Notes, note)
		size += len(note.Text)
	}
	return page, nil
}
