package run

import (
	"context"
	"errors"
	"fmt"
)

// ask publishes the correlation and event under the target's lifetime lock.
// No ID escapes for an event that was not actually enqueued.
func (x *execution) ask(caller context.Context, from *managerState, to, text string) (string, error) {
	ctx, stop := x.toolContext(caller)
	defer stop()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	target, ok := x.states[to]
	if !ok || target == from || text == "" {
		return "", fmt.Errorf("invalid Manager question")
	}
	if err := x.request.Allowance.take(); err != nil {
		return "", err
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	x.mu.Lock()
	if x.closing || !target.accepting {
		x.mu.Unlock()
		return "", fmt.Errorf("Run question target closed")
	}
	x.sequence++
	id := fmt.Sprintf("%s-q%d", x.request.ID, x.sequence)
	q := Question{id, from.manager.Package, to, text}
	x.questions[id] = q
	x.mu.Unlock()
	select {
	case target.inbox <- Event{Question: &q}:
		x.wakeLocked(target)
		return id, nil
	case <-ctx.Done():
		x.mu.Lock()
		delete(x.questions, id)
		x.mu.Unlock()
		return "", ctx.Err()
	}
}
func (x *execution) reply(caller context.Context, from *managerState, id, text string) error {
	ctx, stop := x.toolContext(caller)
	defer stop()
	if err := ctx.Err(); err != nil {
		return err
	}
	if text == "" {
		return fmt.Errorf("empty Manager answer")
	}
	// Keep the correlation locked until actual publication. A duplicate or
	// canceled response can neither consume nor fabricate another Manager's ID.
	x.mu.Lock()
	q, ok := x.questions[id]
	if !ok || q.To != from.manager.Package || x.answered[id] {
		x.mu.Unlock()
		return fmt.Errorf("unknown, closed or foreign question")
	}
	answer := Answer{id, from.manager.Package, text}
	target := x.states[q.From]
	select {
	case target.inbox <- Event{Answer: &answer}:
		x.answered[id] = true
		x.mu.Unlock()
		target.mu.Lock()
		x.wakeLocked(target)
		target.mu.Unlock()
		return nil
	default:
		x.mu.Unlock()
		return fmt.Errorf("Manager answer inbox full; retry the same question ID")
	}

}
func (x *execution) hasQuestionsLocked(manager string) bool {
	for _, q := range x.questions {
		if q.From == manager || q.To == manager {
			return true
		}
	}
	return false
}
func (x *execution) pendingInputLocked(manager string) bool {
	for id, q := range x.questions {
		if q.To == manager && !x.answered[id] || q.From == manager && x.answered[id] {
			return true
		}
	}
	return false
}
func (x *execution) finishedTask(s *managerState, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	s.working = false
	close(s.idle)
	if err == nil {
		x.wakeLocked(s)
	}
}

// A submitted child retains its ordinary conversation. A later parent question
// wakes it with an ordinary message, without nested provider inference or any
// change to the already fixed Git handoff.
func (x *execution) wakeLocked(s *managerState) {
	x.mu.Lock()
	needed := !x.closing && x.pendingInputLocked(s.manager.Package)
	x.mu.Unlock()
	if s.running || !needed || !s.accepting {
		return
	}
	s.running = true
	s.idle = make(chan struct{})
	x.active.Add(1)
	go func() {
		defer x.active.Done()
		for {
			err := x.request.Allowance.take()
			if err == nil {
				var response Response
				response, err = x.exchange(s, Message{Task: "Answer pending Manager questions using Await and Reply. Preserve the existing fixed handoff."})
				if err == nil {
					err = x.observeLate(s, response)
				}
			}
			s.mu.Lock()
			x.mu.Lock()
			needed := x.pendingInputLocked(s.manager.Package)
			if err != nil {
				x.asyncErr = errors.Join(x.asyncErr, err)
				x.cancel()
			}
			if err != nil || !needed || x.closing {
				s.running = false
				close(s.idle)
				x.mu.Unlock()
				s.mu.Unlock()
				return
			}
			x.mu.Unlock()
			s.mu.Unlock()
		}
	}()
}

type sessionResponse struct {
	response Response
	err      error
}

func (x *execution) exchange(s *managerState, m Message) (Response, error) {
	returned := make(chan sessionResponse, 1)
	x.calls.Add(1)
	go func() {
		defer x.calls.Done()
		r, err := s.session.Exchange(x.ctx, m)
		returned <- sessionResponse{r, err}
	}()
	select {
	case r := <-returned:
		return r.response, r.err
	case <-x.ctx.Done():
		return Response{}, x.ctx.Err()
	}
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

// A later required finding remains semantic authority even when the Git result
// was already accepted. Invalidate affected acceptance through the real parents.
func (x *execution) observeLate(s *managerState, r Response) error {
	if r.Decision != "" && r.Decision != Approve && r.Decision != Reject && r.Decision != ModelBlocker {
		return fmt.Errorf("invalid late Manager judgment")
	}
	if r.Decision != Reject && r.Decision != ModelBlocker && !concreteFindings(r.Findings) {
		return nil
	}
	if !concreteFindings(r.Findings) {
		return fmt.Errorf("late rejection or blocker requires concrete findings")
	}
	decision := Reject
	if r.Decision == ModelBlocker {
		decision = ModelBlocker
	}
	findings := append([]Finding(nil), r.Findings...)
	for i := range findings {
		findings[i].Manager = s.manager.Package
	}
	for current := s; current != nil; current = x.states[current.manager.Parent] {
		current.mu.Lock()
		if current.handoff.Decision != ModelBlocker {
			current.handoff.Decision = decision
		}
		current.handoff.Findings = append(current.handoff.Findings, findings...)
		if current.handoff.Review != nil {
			current.handoff.Review.Decision = decision
			current.handoff.Review.Findings = append(current.handoff.Review.Findings, findings...)
		}
		current.mu.Unlock()
		if current.manager.Package == "." {
			break
		}
	}
	x.mu.Lock()
	if x.lateDecision != ModelBlocker {
		x.lateDecision = decision
	}
	x.mu.Unlock()
	return nil
}
