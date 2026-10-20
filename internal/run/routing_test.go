package run

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// A cross-branch question travels only along tree edges: lending asks Root,
// Root forwards it to library with a correlated question, library answers Root
// and Root answers lending. A sibling address is refused.
func TestCrossBranchQuestionTakesTheAncestorRouteBothWays(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	asked, forwarded := "", ""
	var siblingRefusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "lending" && strings.HasPrefix(c.now(), "work on your own contribution") && asked == "":
			a := c.send("library", sendQuestion, "how do you print a loan?", "")
			siblingRefusal = a.Text
			if !a.Refused {
				t.Errorf("a sibling question was sent")
			}
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendQuestion, "text": "how does library print a loan?"})
			asked = "Q1"
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Question Q1 from your child lending") && forwarded == "":
			c.must(toolSend, map[string]any{"to": "library", "kind": sendQuestion, "text": "lending asks how you print a loan", "about": "Q1"})
			forwarded = "Q2"
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q2"})
			for _, id := range c.pendingDecisions() {
				c.must(toolReport, map[string]any{"kind": kindAccept, "proposal": id})
			}
			return nil
		case c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Answer to your question Q2"):
			c.must(toolSend, map[string]any{"to": "lending", "kind": sendAnswer, "text": "library prints JSON lines", "about": "Q1"})
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v: %s", res.Stopped, out.Reason)
	}
	if !strings.Contains(siblingRefusal, "not one of your contacts") {
		t.Errorf("sibling refusal: %q", siblingRefusal)
	}
	byID := map[string]Question{}
	for _, q := range out.Questions {
		byID[q.ID] = q
	}
	q1, q2 := byID["Q1"], byID["Q2"]
	if q1.From != "lending" || q1.To != "." || !q1.Answered || q1.Answer != "library prints JSON lines" {
		t.Errorf("Q1 %+v", q1)
	}
	if q2.From != "." || q2.To != "library" || q2.About != "Q1" || !q2.Answered {
		t.Errorf("Q2 %+v", q2)
	}
	if !strings.Contains(promptOf(f, "lending", "library prints JSON lines"), "Answer to your question Q1") {
		t.Errorf("lending never got the answer")
	}
}

// A parent asks a child while it waits for the child's work, and the child
// answers; an answer to a question that was not addressed to the sender is
// refused.
func TestParentAsksAChildAndForeignAnswersAreRefused(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	asked := false
	var foreign string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && strings.HasPrefix(c.now(), "your children work") && !asked {
			c.must(toolSend, map[string]any{"to": "library", "kind": sendQuestion, "text": "which flags do you parse?"})
			asked = true
		}
		if c.turn.Manager == "lending" && asked && foreign == "" {
			a := c.send("parent", sendAnswer, "none", "Q1")
			foreign = a.Text
			if !a.Refused {
				t.Errorf("lending answered a question addressed to library")
			}
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped {
		t.Fatalf("stopped: %s", out.Reason)
	}
	if len(out.Questions) != 1 || !out.Questions[0].Answered || out.Questions[0].From != "." || out.Questions[0].To != "library" {
		t.Errorf("questions %+v", out.Questions)
	}
}

// Root's question to the user pauses Root, and the run continues with the
// answer; an unanswered question cannot hide behind a complete result.
func TestRootUserQuestionPausesAndContinuesTheSameRun(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	asked := false
	var hidden string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && strings.HasPrefix(c.now(), "work on your own contribution") && !asked {
			c.must(toolSend, map[string]any{"to": User, "kind": sendQuestion, "text": "Which loan limit applies to staff?"})
			asked = true
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			go func() {
				time.Sleep(30 * time.Millisecond)
				f.owner.answer("Q1", "the model's limit applies")
			}()
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped || res.Verdict != Satisfied {
		t.Fatalf("stopped %v: %s", res.Stopped, out.Reason)
	}
	if len(f.owner.questions()) != 1 || !out.Questions[0].Answered || out.Questions[0].To != User {
		t.Fatalf("questions %+v", out.Questions)
	}
	if !strings.Contains(promptOf(f, ".", "the model's limit applies"), "Answer of the user to your question Q1") {
		t.Errorf("Root never saw the answer")
	}
	_ = hidden
}

// A result is refused while its Manager's own question is unanswered.
func TestOpenQuestionBlocksTheCompleteResult(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var refusal string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "library" && strings.HasPrefix(c.now(), "submit your complete result") && refusal == "" {
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendQuestion, "text": "is a flag required?"})
			a := c.result(nil)
			refusal = a.Text
			if !a.Refused {
				t.Errorf("a result with an open question was accepted")
			}
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if res.Stopped {
		t.Fatalf("stopped: %s", out.Reason)
	}
	if !strings.Contains(refusal, "Q1 is unanswered") {
		t.Errorf("refusal %q", refusal)
	}
}

// Root's decision that the model is erroneous ends the run Failed at once;
// a child cannot abort.
func TestRootModelErrorEndsTheRunFailed(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	var childAbort string
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "lending" && childAbort == "" {
			a := c.tool(toolReport, map[string]any{"kind": kindProblem, "category": "model", "text": "contradiction", "abort": true})
			childAbort = a.Text
			if !a.Refused {
				t.Errorf("a child ended the run")
			}
			c.must(toolEscalate, map[string]any{"category": "model", "text": "borrow and reminders contradict each other"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			return nil
		}
		if c.turn.Manager == "." && strings.Contains(c.turn.Prompt, "Escalated model problem") {
			c.must(toolReport, map[string]any{"kind": kindProblem, "category": "model", "text": "lending/borrow contradicts lending/reminders/notice", "abort": true})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !res.Stopped || out.Status != Failed || !strings.HasPrefix(out.Reason, ModelErrorReason) {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
	if out.Candidate.EntryCount() != 0 {
		t.Errorf("a failed run returns no candidate")
	}
	var escalated, aborted bool
	for _, p := range out.Problems {
		escalated = escalated || p.Manager == "lending" && p.Escalated && p.Question == "Q1"
		aborted = aborted || p.Manager == "." && p.Abort
	}
	if !escalated || !aborted {
		t.Errorf("problems %+v", out.Problems)
	}
}

// Without an owner channel nobody answers: Root's question is refused with the
// way out within the model.
func TestWithoutOwnerChannelRootCannotAsk(t *testing.T) {
	f := newFixture(t)
	f.cfg.Owner = nil
	var refusal string
	var mu sync.Mutex
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		if c.turn.Manager == "." && refusal == "" {
			a := c.send(User, sendQuestion, "may I?", "")
			refusal = a.Text
		}
		c.protocol()
		return nil
	}
	r := f.open()
	r.Pass(planSpec())
	f.finish(r, "")
	if !strings.Contains(refusal, "Nobody answers questions") {
		t.Errorf("refusal %q", refusal)
	}
}

// A wait that nothing can resolve is a visible problem, not endless polling.
func TestOrphanedWaitEndsVisiblyIncomplete(t *testing.T) {
	f := newFixture(t)
	var mu sync.Mutex
	asked := false
	f.agents.script = func(c *call) error {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case c.turn.Manager == "library" && !asked:
			c.must(toolSend, map[string]any{"to": "parent", "kind": sendQuestion, "text": "what now?"})
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "Q1"})
			asked = true
			return nil
		case c.turn.Manager == "." && len(c.pendingQuestions()) > 0:
			// Root forwards to the user... no: it waits for library itself.
			c.must(toolReport, map[string]any{"kind": kindWait, "for": "library"})
			return nil
		}
		c.protocol()
		return nil
	}
	r := f.open()
	res := r.Pass(implementSpec())
	out := f.finish(r, "")
	if !res.Stopped || out.Status != Incomplete || !strings.Contains(out.Reason, "cannot advance") || !strings.Contains(out.Reason, "library waits for Q1") {
		t.Fatalf("status %s: %s", out.Status, out.Reason)
	}
}
