package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// queueLocked delivers an event to s's next await or next message.
func (x *execution) queueLocked(s *managerState, e entry) {
	s.queue = append(s.queue, e)
	select {
	case s.signal <- struct{}{}:
	default:
	}
	x.scheduleLocked(s)
}

func (x *execution) queueHandoffLocked(child *managerState) {
	x.queueLocked(x.states[child.manager.Parent], entry{child: child.manager.Package, gen: child.a.gen})
}

// waitingLocked reports whether something s expects can still arrive: the
// handoff of a working direct child or the answer to its own open question.
func (x *execution) waitingLocked(s *managerState) bool {
	for _, name := range s.manager.Children {
		if p := x.states[name].a.phase; p == assigned || p == active {
			return true
		}
	}
	for _, q := range x.questions {
		if q.From == s.manager.Package && !q.answered {
			return true
		}
	}
	return false
}

// neededLocked reports whether s has something to handle: queued events, or an
// active assignment without judgment that waits for nobody.
func (x *execution) neededLocked(s *managerState) bool {
	if len(s.queue) != 0 {
		return true
	}
	return s.a.phase == active && s.a.judgment == nil && !x.waitingLocked(s)
}

// scheduleLocked starts s's driver when s has something to handle and no
// driver owns its Session yet.
func (x *execution) scheduleLocked(s *managerState) {
	if s.running || x.admissionErrorLocked() != nil || !x.neededLocked(s) {
		return
	}
	s.running = true
	x.active.Add(1)
	go x.drive(s)
}

// settleLocked starts every Manager with something to handle and reports
// whether the whole hierarchy is quiet. A quiet hierarchy signals completion.
// A running Manager blocked in await rechecks whether anything can arrive.
func (x *execution) settleLocked() bool {
	quiet := true
	for _, s := range x.states {
		x.scheduleLocked(s)
		quiet = quiet && !s.running
	}
	for _, s := range x.states {
		if s.running {
			select {
			case s.signal <- struct{}{}:
			default:
			}
		}
	}
	if quiet {
		select {
		case x.settled <- struct{}{}:
		default:
		}
	}
	return quiet
}

// drive sends s one message per native response while s has something to
// handle. The end of a response is no judgment: an active assignment without
// judgment continues once nothing it waits for is outstanding.
func (x *execution) drive(s *managerState) {
	defer x.active.Done()
	for {
		// A message without new events continues an open assignment. Only
		// repeated responses that do nothing at all end the Run.
		message, continued, ok, err := x.next(s)
		var before Commit
		if err == nil && continued {
			before, err = x.state(s)
		}
		if err == nil && ok {
			err = x.request.Allowance.take()
		}
		if err == nil && ok {
			err = x.exchange(s, message)
		}
		if err == nil && ok {
			err = x.acted(s, continued, before)
		}
		if err == nil && ok {
			err = x.responded(s)
		}
		if err != nil {
			// Stop the Run first: no second driver can be admitted afterwards.
			x.stopPort(err)
			x.mu.Lock()
			s.running = false
			x.mu.Unlock()
			return
		}
		if !ok {
			return
		}
	}
}

// next assembles the message for s, or releases s's driver when s has nothing
// to handle. Both happen under one lock so no queued event is missed. The
// message continues s's assignment unless it delivers a new event; a question
// delivered again because it stayed unanswered is not new.
func (x *execution) next(s *managerState) (m Message, continued, ok bool, err error) {
	fresh := false
	for {
		x.mu.Lock()
		if err := x.admissionErrorLocked(); err != nil {
			x.mu.Unlock()
			return m, false, false, err
		}
		if len(s.queue) == 0 {
			m.Feedback = append(m.Feedback, s.a.feedback...)
			s.a.feedback = nil
			open := s.a.phase == active && s.a.judgment == nil && !x.waitingLocked(s)
			if len(m.Events) == 0 && len(m.Feedback) == 0 {
				if !open {
					s.running = false
					x.settleLocked()
					x.mu.Unlock()
					return m, false, false, nil
				}
				text := "Continue your assignment: "
				if s.idle > 0 {
					text = "Your last response took no action; work on your assignment or judge it: "
				}
				m.Feedback = []Finding{{Text: text + s.a.task.Task}}
			}
			x.mu.Unlock()
			return m, !fresh, true, nil
		}
		e := s.queue[0]
		s.queue = s.queue[1:]
		x.mu.Unlock()
		event, delivered, err := x.deliver(x.ctx, s, e)
		if err != nil {
			return m, false, false, err
		}
		if delivered {
			m.Events = append(m.Events, event)
			fresh = fresh || !e.again
		}
	}
}

// maxIdle bounds continued responses in a row that do nothing at all.
const maxIdle = 2

// state observes s's working copy; an unresolved merge has no comparable state.
func (x *execution) state(s *managerState) (Commit, error) {
	s.git.Lock()
	defer s.git.Unlock()
	state, err := s.copy.Snapshot(x.ctx)
	if errors.Is(err, ErrMergeConflict) {
		return Commit{}, nil
	}
	return state, err
}

// acted records whether a continued response did anything: a Manager tool
// call, a native execution its Session observed or a change of the working
// state. What the action achieved is not judged. An event message resets the
// count.
func (x *execution) acted(s *managerState, continued bool, before Commit) error {
	changed := false
	if continued {
		after, err := x.state(s)
		if err != nil {
			return err
		}
		changed = after != before
	}
	executed := false
	if native, ok := s.session.(ActingSession); ok {
		executed = native.Acted()
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	called := s.called
	s.called = false
	// Only an open assignment can stand still. A Manager without one has
	// nothing to continue, and questions it leaves unanswered are closed.
	if !continued || called || executed || changed || s.a.phase != active {
		s.idle = 0
		return nil
	}
	s.idle++
	if s.idle < maxIdle {
		return nil
	}
	return fmt.Errorf("%w: Manager %s took no action in %d continued responses in a row on its assignment %q%s", ErrNoProgress, s.manager.Package, s.idle, s.a.task.Task, x.openQuestionsLocked(s))
}

// responded follows the end of one native response: an active judged
// assignment is fixed first. Then each question s received and left
// unanswered is delivered again while s has an open assignment, and closed
// otherwise. A question s has not received yet keeps its one delivery.
func (x *execution) responded(s *managerState) error {
	x.mu.Lock()
	j := s.a.judgment
	fixable := s.a.phase == active && j != nil && s.manager.Package != organization.Root
	x.mu.Unlock()
	if fixable {
		if err := x.fix(s, j); err != nil {
			return err
		}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	for _, q := range x.sortedQuestionsLocked() {
		if q.To != s.manager.Package || !q.delivered || q.answered {
			continue
		}
		if s.a.phase == active || s.a.phase == assigned {
			q.delivered = false
			x.queueLocked(s, entry{event: Event{Question: &q.Question}, again: true})
		} else {
			x.closeLocked(q)
		}
	}
	return nil
}

func (x *execution) sortedQuestionsLocked() []*question {
	out := make([]*question, 0, len(x.questions))
	for _, q := range x.questions {
		out = append(out, q)
	}
	slices.SortFunc(out, func(a, b *question) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// deliver turns one queued entry into the event s receives. A received task
// opens the assignment; in Implement the child's copy first merges the parent
// baseline fixed at delegation. A superseded handoff generation is skipped.
func (x *execution) deliver(ctx context.Context, s *managerState, e entry) (Event, bool, error) {
	switch {
	case e.event.Assignment != nil:
		task := *e.event.Assignment
		task.Findings = slices.Clone(task.Findings)
		if x.request.Mandate == Implement && s.manager.Package != organization.Root {
			s.git.Lock()
			err := s.copy.Update(ctx, x.states[s.manager.Parent].copy, task.Base)
			s.git.Unlock()
			if errors.Is(err, ErrMergeConflict) {
				task.Findings = append(task.Findings, Finding{Manager: s.manager.Parent, Text: "Resolve the preserved Git merge conflict against your parent's baseline: " + err.Error()})
			} else if err != nil {
				return Event{}, false, err
			}
		}
		x.mu.Lock()
		defer x.mu.Unlock()
		if err := x.admissionErrorLocked(); err != nil {
			return Event{}, false, err
		}
		s.a.receive(task)
		return Event{Assignment: &task}, true, nil
	case e.event.Question != nil:
		x.mu.Lock()
		defer x.mu.Unlock()
		q, ok := x.questions[e.event.Question.ID]
		if !ok || q.answered {
			return Event{}, false, nil
		}
		q.delivered = true
		value := q.Question
		return Event{Question: &value}, true, nil
	case e.event.Answer != nil:
		x.mu.Lock()
		defer x.mu.Unlock()
		q, ok := x.questions[e.event.Answer.ID]
		if !ok || q.From != s.manager.Package || !q.answered {
			return Event{}, false, fmt.Errorf("answer correlation mismatch")
		}
		delete(x.questions, q.ID)
		answer := *e.event.Answer
		return Event{Answer: &answer}, true, nil
	}
	child := x.states[e.child]
	x.mu.Lock()
	if child.a.phase != fixed || child.a.gen != e.gen {
		x.mu.Unlock()
		return Event{}, false, nil
	}
	h := copyHandoff(child.a.handoff)
	x.mu.Unlock()
	s.git.Lock()
	diff, err := s.copy.Review(ctx, child.copy, h.Base, h.Result)
	s.git.Unlock()
	if err != nil {
		x.stopPort(err)
		return Event{}, false, err
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if child.a.phase != fixed || child.a.gen != e.gen {
		return Event{}, false, nil
	}
	child.a.handoff.Diff = diff
	child.a.sent = e.gen
	h.Diff = diff
	return Event{Handoff: &h}, true, nil
}

// await returns the next event. It refuses at once when nothing can arrive,
// so a Manager never waits for an event that no one will send. A Manager
// blocked in await waits too: when s starts blocking, every other blocked
// Manager rechecks, so Managers that wait for each other never all block.
func (x *execution) await(ctx context.Context, s *managerState) (Event, error) {
	blocked := false
	// unblock ends s's blocked wait; the caller holds x.mu. A wakeup that
	// blocks again stays one wait, so it wakes no other Manager.
	unblock := func() {
		if blocked {
			s.awaiting--
			blocked = false
		}
	}
	for {
		x.mu.Lock()
		if err := x.admissionErrorLocked(); err != nil {
			unblock()
			x.mu.Unlock()
			return Event{}, err
		}
		if len(s.queue) != 0 {
			unblock()
			e := s.queue[0]
			s.queue = s.queue[1:]
			x.mu.Unlock()
			event, ok, err := x.deliver(ctx, s, e)
			if err != nil {
				x.stopPort(err)
				return Event{}, err
			}
			if ok {
				return event, nil
			}
			continue
		}
		if !x.waitingLocked(s) {
			unblock()
			next := x.nextStepLocked(s)
			x.mu.Unlock()
			return Event{}, fmt.Errorf("nothing can arrive: no direct child of yours is working and none of your questions is open; %s", next)
		}
		if !x.othersRunningLocked(s) {
			// Everyone else waits: only an answer unblocks them. A Manager with
			// a question it has not answered is refused; while another one,
			// blocked in await, owes an answer, s waits for that answer instead.
			next := x.openQuestionsLocked(s)
			if next != "" || !x.answerOwedLocked() {
				unblock()
				if next == "" {
					next = "; " + x.nextStepLocked(s)
				}
				x.mu.Unlock()
				return Event{}, fmt.Errorf("nothing can arrive: every other Manager is waiting%s", next)
			}
		}
		if !blocked {
			blocked = true
			s.awaiting++
			x.wakeAwaitingLocked(s)
		}
		x.mu.Unlock()
		select {
		case <-s.signal:
		case <-ctx.Done():
			x.mu.Lock()
			unblock()
			x.mu.Unlock()
			return Event{}, ctx.Err()
		}
	}
}

// maxOpenQuestions bounds the questions one Manager has asked whose answer has
// not reached it yet. A question takes no allowance of its own: it costs only
// as part of the message that delivers it, so this bound keeps both the
// addressee's questions and the asker's answers finite.
const maxOpenQuestions = 8

// ask follows the hierarchy: only the direct parent or a direct child.
func (x *execution) ask(ctx context.Context, from *managerState, to, text string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	allowed := slices.Clone(from.manager.Children)
	if from.manager.Parent != "" {
		allowed = append([]string{from.manager.Parent}, allowed...)
	}
	if !slices.Contains(allowed, to) {
		reach := "nobody"
		if len(allowed) != 0 {
			reach = strings.Join(allowed, ", ")
		}
		return "", fmt.Errorf("ask reaches only your direct parent or direct children (%s); reach other Managers through the Managers in between", reach)
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty Manager question; ask again with a question text")
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.admissionErrorLocked(); err != nil {
		return "", err
	}
	// A question stays open until its answer, or the host's closure, is
	// delivered to the asker.
	open := 0
	for _, q := range x.questions {
		if q.From == from.manager.Package {
			open++
		}
	}
	if open >= maxOpenQuestions {
		return "", fmt.Errorf("you have %d open questions; wait for answers with await before you ask again", open)
	}
	x.sequence++
	q := &question{Question: Question{ID: fmt.Sprintf("%s-q%d", x.request.ID, x.sequence), From: from.manager.Package, To: to, Text: text}}
	x.questions[q.ID] = q
	x.queueLocked(x.states[to], entry{event: Event{Question: &q.Question}})
	return q.ID, nil
}

func (x *execution) reply(ctx context.Context, from *managerState, id, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty Manager answer; reply again with an answer text")
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.admissionErrorLocked(); err != nil {
		return err
	}
	q, ok := x.questions[id]
	if !ok || q.To != from.manager.Package || q.answered {
		return fmt.Errorf("no open question %s is addressed to you; reply only to question IDs delivered to you", id)
	}
	q.answered = true
	x.queueLocked(x.states[q.From], entry{event: Event{Answer: &Answer{ID: id, From: from.manager.Package, Text: text}}})
	return nil
}

// closeLocked ends q without an answer because its addressee has no open
// assignment to answer it in: it handed off, or has no task. The asker receives
// the reason and a next step in place of the answer.
func (x *execution) closeLocked(q *question) {
	reason := fmt.Sprintf("%s has no open assignment and did not answer your question %s; continue your work", q.To, q.ID)
	if p := x.states[q.To].a.phase; p == fixed || p == reviewed {
		reason = fmt.Sprintf("%s has handed off without answering your question %s; continue from its handoff", q.To, q.ID)
	}
	asker := x.states[q.From]
	if asker.manager.Parent != "" {
		reason += " or ask your parent"
	}
	q.answered = true
	x.queueLocked(asker, entry{event: Event{Answer: &Answer{ID: q.ID, From: q.To, Closed: reason}}})
}

func (x *execution) root() *managerState { return x.states[organization.Root] }

// othersRunningLocked reports whether another Manager can still send s
// something: its driver runs and it is not blocked in await, or an event it
// has yet to take already wakes it.
func (x *execution) othersRunningLocked(s *managerState) bool {
	for _, other := range x.states {
		if other != s && other.running && (other.awaiting == 0 || len(other.queue) != 0) {
			return true
		}
	}
	return false
}

// wakeAwaitingLocked makes every other Manager blocked in await recheck
// whether anything can still arrive, now that s waits too.
func (x *execution) wakeAwaitingLocked(s *managerState) {
	for _, other := range x.states {
		if other != s && other.awaiting > 0 {
			select {
			case other.signal <- struct{}{}:
			default:
			}
		}
	}
}

// answerOwedLocked reports whether a Manager blocked in await holds a question
// it has not answered. Woken, that Manager is refused with the question named,
// as openQuestionsLocked names it.
func (x *execution) answerOwedLocked() bool {
	for _, q := range x.questions {
		if !q.answered && x.states[q.To].awaiting > 0 {
			return true
		}
	}
	return false
}

// nextStepLocked names what s can do instead of waiting.
func (x *execution) nextStepLocked(s *managerState) string {
	if s.a.phase == active && s.a.judgment == nil {
		if err := x.childrenLocked(s); err != nil {
			return err.Error()
		}
		return "continue your work or judge your assignment"
	}
	return "you have no open work to wait for"
}

// openQuestionsLocked names the questions s received and has not answered.
func (x *execution) openQuestionsLocked(s *managerState) string {
	var open []string
	for _, q := range x.sortedQuestionsLocked() {
		if q.To == s.manager.Package && !q.answered {
			open = append(open, q.ID+" from "+q.From)
		}
	}
	if len(open) == 0 {
		return ""
	}
	return "; answer your open questions: " + strings.Join(open, ", ")
}

// exchange bounds each stretch of provider work in one response without a
// served tool call by ProviderIdleTimeout. Serving a call does not count and
// ends the stretch, so a Manager that keeps delegating, awaiting and reviewing
// is never stopped by it. The Run deadline bounds everything.
func (x *execution) exchange(s *managerState, m Message) error {
	messageContext, cancel := context.WithCancelCause(x.ctx)
	defer cancel(nil)
	if x.request.ProviderIdleTimeout > 0 {
		clock := startClock(x.request.ProviderIdleTimeout, func() { cancel(context.DeadlineExceeded) })
		defer clock.stop()
		messageContext = context.WithValue(messageContext, clockKey{}, clock)
	}
	returned := make(chan error, 1)
	x.mu.Lock()
	if err := x.admissionErrorLocked(); err != nil {
		x.mu.Unlock()
		return err
	}
	x.calls.Add(1)
	x.mu.Unlock()
	go func() {
		defer x.calls.Done()
		returned <- s.session.Exchange(messageContext, m)
	}()
	select {
	case err := <-returned:
		if messageContext.Err() != nil {
			return context.Cause(messageContext)
		}
		return err
	case <-messageContext.Done():
		return context.Cause(messageContext)
	}
}

// idleClock expires once the provider works for its limit without a served
// tool call. It pauses while any tool call is served, and the end of the last
// call in progress restarts it at its full limit. It never judges whether a
// call was useful; stop ends it.
type idleClock struct {
	mu      sync.Mutex
	timer   *time.Timer
	limit   time.Duration
	serving int
	running bool
}

type clockKey struct{}

func startClock(limit time.Duration, expire func()) *idleClock {
	return &idleClock{timer: time.AfterFunc(limit, expire), limit: limit, running: true}
}

// serve pauses the clock until the returned function ends the tool call.
func (c *idleClock) serve() func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.serving == 0 && c.running && !c.timer.Stop() {
		// The limit was reached before this call: the response already ends.
		c.running = false
	}
	c.serving++
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.serving--; c.serving == 0 && c.running {
			c.timer.Reset(c.limit)
		}
	}
}

func (c *idleClock) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	c.timer.Stop()
}

// A blocked adapter cannot prevent terminal persistence. Timeout is an honest
// uncertain observation; it does not claim that a goroutine killed a process.
func boundedClose(ctx context.Context, close func(context.Context) error) error {
	done := make(chan error, 1)
	go func() { done <- close(ctx) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("resource closure unconfirmed: %w", ctx.Err())
	}
}
