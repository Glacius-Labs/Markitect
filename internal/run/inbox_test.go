package run

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func inboxExecution() (*execution, *managerState, *managerState) {
	ctx, cancel := context.WithCancel(context.Background())
	a := &managerState{manager: organization.Manager{Package: ".", Children: []string{"a"}}, running: true, working: true, accepting: true, inbox: make(chan Event, 1)}
	b := &managerState{manager: organization.Manager{Package: "a", Parent: "."}, running: true, working: true, accepting: true, inbox: make(chan Event, 1)}
	x := &execution{ctx: ctx, cancel: cancel, request: Request{ID: "inbox-test", Allowance: NewAllowance(time.Now().Add(time.Second), 10)}, states: map[string]*managerState{".": a, "a": b}, questions: map[string]Question{}, answered: map[string]bool{}}
	return x, a, b
}

func TestConcurrentLateFindingCannotBeOverwrittenByCandidateCompletion(t *testing.T) {
	for i := 0; i < 100; i++ {
		x, parent, child := inboxExecution()
		parent.mu.Lock()
		var started, done sync.WaitGroup
		started.Add(2)
		done.Add(2)
		go func() {
			defer done.Done()
			started.Done()
			parent.finish(Commit{ID: "fixed", Tree: "tree"}, Response{Decision: Submitted, Findings: []Finding{{Manager: ".", Text: "ordinary submission"}}}, "")
		}()
		go func() {
			defer done.Done()
			started.Done()
			if err := x.observeLate(child, Response{Decision: ModelBlocker, Findings: []Finding{{Text: "late required blocker"}}}); err != nil {
				t.Error(err)
			}
		}()
		started.Wait()
		parent.mu.Unlock()
		done.Wait()
		x.cancel()
		if parent.handoff.Decision != ModelBlocker || len(parent.handoff.Findings) != 2 {
			t.Fatal("candidate completion lost concurrent required finding", parent.handoff)
		}
	}
}

func TestAnswerPublicationIsRetryableAndAcknowledgedOnlyByRequester(t *testing.T) {
	x, a, b := inboxExecution()
	defer x.cancel()
	id, err := x.ask(x.ctx, a, "a", "actual question")
	if err != nil {
		t.Fatal(err)
	}
	event, err := x.tools(b).Await(x.ctx)
	if err != nil || event.Question.ID != id {
		t.Fatal(event, err)
	}
	// Saturating the asker's inbox cannot hold the global correlation lock.
	a.inbox <- Event{Question: &Question{ID: "unrelated"}}
	if err = x.reply(x.ctx, b, id, "actual answer"); err == nil {
		t.Fatal("full inbox fabricated publication")
	}
	if x.answered[id] || len(x.questions) != 1 {
		t.Fatal("failed publication consumed correlation")
	}
	cancelled, cancel := context.WithCancel(x.ctx)
	cancel()
	if err = x.reply(cancelled, b, id, "canceled"); err == nil {
		t.Fatal("canceled reply published")
	}
	<-a.inbox
	if err = x.reply(x.ctx, a, id, "forged"); err == nil {
		t.Fatal("foreign Manager replied")
	}
	if err = x.reply(x.ctx, b, id, "actual answer"); err != nil {
		t.Fatal(err)
	}
	if len(x.questions) != 1 || !x.answered[id] || x.pendingInputLocked("a") || !x.pendingInputLocked(".") {
		t.Fatal("answered inbound question caused extra inference or unread answer counted as resolved")
	}
	if err = x.reply(x.ctx, b, id, "duplicate"); err == nil {
		t.Fatal("duplicate reply admitted")
	}
	event, err = x.tools(a).Await(x.ctx)
	if err != nil || event.Answer.ID != id || event.Answer.Text != "actual answer" || len(x.questions) != 0 {
		t.Fatal(event, err)
	}
}

func TestCanceledQuestionEnqueueLeavesNoCorrelation(t *testing.T) {
	x, a, b := inboxExecution()
	defer x.cancel()
	b.inbox <- Event{}
	ctx, cancel := context.WithTimeout(x.ctx, time.Millisecond)
	defer cancel()
	id, err := x.ask(ctx, a, "a", "cannot enqueue")
	if err == nil || id != "" || len(x.questions) != 0 {
		t.Fatal("unpublished question escaped", id, err)
	}
}

func TestLateBlockerInvalidatesPreviouslyAcceptedHierarchy(t *testing.T) {
	x, a, b := inboxExecution()
	defer x.cancel()
	a.handoff = Handoff{Manager: ".", Decision: Submitted}
	b.handoff = Handoff{Manager: "a", Decision: Submitted, Review: &ParentReview{Decision: Approve}}
	if err := x.observeLate(b, Response{}); err != nil || b.handoff.Review.Decision != Approve {
		t.Fatal("answer-only message changed acceptance", err)
	}
	if err := x.observeLate(b, Response{Decision: ModelBlocker, Findings: []Finding{{Text: "required model condition contradicts itself"}}}); err != nil {
		t.Fatal(err)
	}
	if a.handoff.Decision != ModelBlocker || b.handoff.Review.Decision != ModelBlocker || a.handoff.Findings[0].Manager != "a" {
		t.Fatal("late blocker was discarded")
	}
	if concreteFindings([]Finding{{Text: " \n"}}) {
		t.Fatal("blank finding authorized correction")
	}
}

func TestOwnModelBlockerDominatesChildRejectAtCompletionAndAggregation(t *testing.T) {
	x, parent, child := inboxExecution()
	defer x.cancel()
	child.working = false
	child.judgment = Reject
	child.handoff = Handoff{Manager: "a", Decision: Reject, Review: &ParentReview{Decision: Reject, Findings: []Finding{{Text: "child issue"}}}}
	h := parent.finish(Commit{ID: "fixed", Tree: "tree"}, Response{Decision: ModelBlocker, Findings: []Finding{{Manager: ".", Text: "own fixed-model contradiction"}}}, Reject)
	if h.Decision != ModelBlocker {
		t.Fatal("child rejection downgraded own blocker", h)
	}
	decision, err := x.aggregate(parent)
	if err != nil || decision != ModelBlocker || parent.handoff.Decision != ModelBlocker {
		t.Fatal("aggregation downgraded own blocker", decision, err)
	}
}
