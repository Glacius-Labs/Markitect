package run

import (
	"errors"
	"fmt"
	"slices"
)

// Message is one message along the Manager tree (core rules M1 to M5): a
// question, a reply or a hint. A question becomes an issue addressed to its
// addressee (core rule I2); a reply resolves it. Path lists the Managers
// between sender and addressee, who see the message in their next step without
// being dispatched (core rule M2). The owner's answers are replies from
// "owner".
type Message struct {
	ID, Kind, From, To, About string
	Path                      []string
	Body                      string
	Step                      StepID
	At                        uint64
	Source                    BlobRef
}

const (
	messageQuestion = "question"
	messageReply    = "reply"
	messageHint     = "hint"
	toParent        = "parent"
)

type sendInput struct {
	Kind string `json:"kind"`
	To   string `json:"to"`
	Body string `json:"body"`
}

// Messages returns the run's messages in sending order.
func (c *Coordinator) Messages() []Message {
	out := slices.Clone(c.state.messages)
	for i := range out {
		out[i].Path = slices.Clone(out[i].Path)
	}
	return out
}

// stepSends returns the answered send calls of the task's current step, in
// call order.
func (s stateValue) stepSends(t taskValue) []ToolView {
	var sends []ToolView
	for key, tool := range s.tools {
		if key.Session == t.Session && t.Step != "" && tool.Record.Step == t.Step && tool.Record.Tool == ToolSend && tool.Replied.BatchID != "" && tool.OK {
			sends = append(sends, tool)
		}
	}
	slices.SortFunc(sends, func(a, b ToolView) int { return compareUint(a.Requested.Revision, b.Requested.Revision) })
	return sends
}

// CheckSend checks a send call of the manager's current step against the tree
// and the run, and returns the identity its message gets when the step
// concludes.
func (c *Coordinator) CheckSend(manager string, args []byte) (string, error) {
	t, ok := c.state.tasks[manager]
	if !ok || t.Step == "" {
		return "", errors.New("this task has no running step")
	}
	var in sendInput
	if err := decodeArgs(args, &in); err != nil {
		return "", err
	}
	m, err := c.state.resolveSend(t, in, len(c.state.stepSends(t))+1)
	return m.ID, err
}

// resolveSend checks the n-th send of a task's step and routes it (core rules
// M1 to M4). A question to "parent" goes to the direct parent; a reply answers
// an open question addressed to the sender; a hint goes to a direct child.
func (s stateValue) resolveSend(t taskValue, in sendInput, n int) (Message, error) {
	if !text(in.Body) || !boundedInput(in.Body) {
		return Message{}, errors.New("a message needs a body of at most 64 KiB")
	}
	if t.Phase == PhaseReply && in.Kind != messageReply {
		return Message{}, errors.New("a reply step can only send replies")
	}
	from := t.Slot.Package
	m := Message{ID: fmt.Sprintf("%s-%s-%d", in.Kind, t.Step, n), Kind: in.Kind, From: from, Body: in.Body}
	switch in.Kind {
	case messageQuestion:
		to := in.To
		if to == toParent {
			if t.Slot.Parent == "" {
				return Message{}, errors.New("the root has no parent; ask the owner")
			}
			to = t.Slot.Parent
		}
		if _, ok := s.tasks[to]; to != addressedToOwner && (!ok || to == from) {
			return Message{}, fmt.Errorf("%q is neither another Manager, parent nor owner", in.To)
		}
		m.To, m.Path = to, s.treePath(from, to)
	case messageReply:
		q, ok := s.issues[in.To]
		if !ok || q.Kind != issueQuestion || q.State != issueOpen || q.To != from {
			return Message{}, fmt.Errorf("%q is not an open question addressed to you", in.To)
		}
		m.To, m.About, m.Path = q.Task, q.ID, s.treePath(from, q.Task)
	case messageHint:
		if child, ok := s.tasks[in.To]; !ok || child.Slot.Parent != from {
			return Message{}, fmt.Errorf("a hint goes to a direct child, and %q is none", in.To)
		}
		m.To = in.To
	default:
		return Message{}, fmt.Errorf("unknown message kind %q", in.Kind)
	}
	return m, nil
}

// treePath lists the Managers between two Managers along the tree, or from a
// Manager up to the root for a question to the owner (core rules M2, M4).
func (s stateValue) treePath(from, to string) []string {
	up := func(p string) []string {
		chain := []string{p}
		for t := s.tasks[p]; t.Slot.Parent != ""; t = s.tasks[t.Slot.Parent] {
			chain = append(chain, t.Slot.Parent)
		}
		return chain
	}
	fromUp := up(from)
	if to == addressedToOwner {
		return fromUp[1:]
	}
	toUp := up(to)
	i := slices.IndexFunc(fromUp, func(p string) bool { return slices.Contains(toUp, p) })
	if i < 0 {
		return nil
	}
	j := slices.Index(toUp, fromUp[i])
	var path []string
	if i > 1 {
		path = append(path, fromUp[1:i]...)
	}
	if lca := fromUp[i]; lca != from && lca != to {
		path = append(path, lca)
	}
	for k := j - 1; k > 0; k-- {
		path = append(path, toUp[k])
	}
	return path
}

// recordSends records the messages of a concluding step in call order and
// applies them: a question becomes an open issue addressed to its addressee,
// and a reply resolves its question, which wakes an asker waiting for it (core
// rules M1 to M5, I2, T10).
func (s *stateValue) recordSends(t taskValue, revision uint64) error {
	for i, call := range s.stepSends(t) {
		if call.Send == nil {
			return ErrState
		}
		m, err := s.resolveSend(t, *call.Send, i+1)
		if err != nil {
			return ErrState
		}
		m.Step, m.At, m.Source = t.Step, revision, call.Record.Wire.Journal
		s.addMessage(m)
	}
	return nil
}

func (s *stateValue) addMessage(m Message) {
	s.messages = append(s.messages, m)
	switch m.Kind {
	case messageQuestion:
		s.addIssue(Issue{ID: m.ID, Kind: issueQuestion, Severity: severityAdvisory, Task: m.From, To: m.To, Request: m.Body, State: issueOpen, Raised: m.Step, Source: m.Source, At: m.At})
	case messageReply:
		q := s.issues[m.About]
		q.State, q.Answer = issueResolved, m.ID
		s.issues[m.About] = q
	}
}

// unanswered returns the open questions addressed to a task that have had no
// reply step yet, oldest first (core rule T11).
func (s stateValue) unanswered(t taskValue) []Issue {
	var out []Issue
	for _, id := range s.issueOrder {
		if q := s.issues[id]; q.Kind == issueQuestion && q.To == t.Slot.Package && q.State == issueOpen && q.ReplyStep == "" {
			out = append(out, q)
		}
	}
	return out
}

// replyStepTaken marks the questions a reply step saw, so that each question
// admits one reply step (core rule T11).
func (s *stateValue) replyStepTaken(t taskValue, step StepID) {
	for _, q := range s.unanswered(t) {
		q.ReplyStep = step
		s.issues[q.ID] = q
	}
}

// waitsForOwner reports whether a task waits for an answer from the owner.
func (s stateValue) waitsForOwner(t taskValue) bool {
	return t.State == TaskBlocked && slices.ContainsFunc(t.Waits, func(w Wait) bool {
		q, ok := s.issues[w.Ref]
		return (w.Kind == waitOwner || w.Kind == waitReply) && ok && q.To == addressedToOwner && q.State == issueOpen
	})
}

// AnswerQuestion records the owner's answer (core rules I2, W2).
const AnswerQuestion = "question.answered"

// AnswerRecord is the owner's answer to one question.
type AnswerRecord struct {
	Question string
	Text     BlobRef
}

func (r AnswerRecord) payloadRefs() []BlobRef {
	if r.Text == (BlobRef{}) {
		return nil
	}
	return []BlobRef{r.Text}
}

const answerMedia = "text/plain"

// OwnerQuestions returns the open questions addressed to the owner.
func (c *Coordinator) OwnerQuestions() []Issue {
	var out []Issue
	for _, id := range c.state.issueOrder {
		if q := c.state.issues[id]; q.Kind == issueQuestion && q.To == addressedToOwner && q.State == issueOpen {
			q.Definitions = slices.Clone(q.Definitions)
			out = append(out, q)
		}
	}
	return out
}

// AnswerCommand records the owner's answer to an open question to the owner.
// An answer to a review's question starts a fresh review of the same frozen
// candidate (core rule T8); any other answer wakes the asker (core rule T10).
func (c *Coordinator) AnswerCommand(meta RecordMeta, question, answer string) (Command, error) {
	if !text(answer) || !boundedInput(answer) {
		return Command{}, ErrInvalid
	}
	p, err := NewPayload(answerMedia, "utf-8", []byte(answer))
	if err != nil {
		return Command{}, err
	}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: AnswerQuestion, Assignment: c.Current().Assignment, Run: c.Current().ID,
		Answer: AnswerRecord{Question: question, Text: p.Ref()}, sessionPayloads: []Payload{p}}, nil
}

func (s *stateValue) answer(c Command) error {
	r := s.current
	q, ok := s.issues[c.Answer.Question]
	body, found := payloadFor(c, c.Answer.Text)
	if r.State != Running && r.State != Waiting || !ok || q.Kind != issueQuestion || q.To != addressedToOwner || q.State != issueOpen {
		return ErrState
	}
	if !found || len(c.sessionPayloads) != 1 || body.Ref().MediaType != answerMedia {
		return ErrInvalid
	}
	revision := s.revision
	s.addMessage(Message{ID: "answer-" + q.ID, Kind: messageReply, From: addressedToOwner, To: q.Task, About: q.ID, Body: string(body.Bytes()), At: revision, Source: c.Answer.Text})
	if t, ok := s.tasks[q.Task]; ok && t.State == TaskBlocked && t.Submission.Proposal != (BlobRef{}) &&
		slices.ContainsFunc(t.Waits, func(w Wait) bool { return w.Ref == q.ID }) {
		t.State, t.Since, t.Waits, t.Review = TaskReviewing, revision, nil, reviewValue{}
		s.tasks[q.Task] = t
	}
	s.advanceTasks(revision)
	return nil
}

// decodeSend reads a recorded send call's input, when it is one.
func decodeSend(r SessionRecord, input Payload) *sendInput {
	if r.Tool != ToolSend || r.Wire.Redacted {
		return nil
	}
	args, err := CallArguments(ToolSend, input.Bytes())
	if err != nil {
		return nil
	}
	var in sendInput
	if decodeArgs(args, &in) != nil {
		return nil
	}
	return &in
}

// pendingQuestions returns the questions the task's running step sent, by
// identity and addressee, before the step concludes.
func (s stateValue) pendingQuestions(t taskValue) map[string]string {
	out := map[string]string{}
	for i, v := range s.stepSends(t) {
		if v.Send == nil || v.Send.Kind != messageQuestion {
			continue
		}
		if m, err := s.resolveSend(t, *v.Send, i+1); err == nil {
			out[m.ID] = m.To
		}
	}
	return out
}
