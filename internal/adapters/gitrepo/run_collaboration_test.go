package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// collaboration runs controlled Sessions over real local Git. Each Manager
// follows a standard well-behaved flow; a test script takes over selected
// messages to reproduce one counterexample of concept §13.3.
type collaboration struct {
	mu        sync.Mutex
	managers  map[string]*collaborator
	script    func(context.Context, *collaborator, run.Message) (bool, error)
	candidate string
	stops     bool  // the Run is expected to end without technical completion
	err       error // its returned error
}

type collaborator struct {
	owner     *collaboration
	spec      run.SessionSpec
	messages  []run.Message
	judgments int
	tasks     []run.Assignment
	open      bool
	started   bool
	handoffs  []run.Handoff
	pending   []run.Handoff
	approved  map[string]bool
	answers   map[string]string
	refusals  []string
	executed  bool // the current response ran a native command, as an adapter observes it
}

func (c *collaboration) Open(_ context.Context, spec run.SessionSpec) (run.Session, error) {
	m := &collaborator{owner: c, spec: spec, approved: map[string]bool{}, answers: map[string]string{}}
	c.mu.Lock()
	if c.managers == nil {
		c.managers = map[string]*collaborator{}
	}
	c.managers[spec.Manager.Package] = m
	c.mu.Unlock()
	return m, nil
}

func (c *collaboration) manager(name string) *collaborator {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.managers[name]
}

func (*collaborator) Close(context.Context) error { return nil }

func (m *collaborator) Acted() bool { return m.executed }

func (m *collaborator) Exchange(ctx context.Context, message run.Message) error {
	m.owner.mu.Lock()
	m.messages = append(m.messages, message)
	m.owner.mu.Unlock()
	m.executed = false
	if m.owner.script != nil {
		if handled, err := m.owner.script(ctx, m, message); handled || err != nil {
			return err
		}
	}
	for _, e := range message.Events {
		if err := m.handle(ctx, e); err != nil {
			return err
		}
	}
	// A host diagnostic or a child handoff concerns an active assignment whose
	// judgment is missing or no longer holds.
	if len(message.Feedback) != 0 || len(m.pending) != 0 {
		m.open = true
	}
	if !m.open {
		return nil
	}
	return m.complete(ctx)
}

func (m *collaborator) handle(ctx context.Context, e run.Event) error {
	switch {
	case e.Assignment != nil:
		m.tasks = append(m.tasks, *e.Assignment)
		m.open = true
	case e.Question != nil:
		return m.spec.Tools.Reply(ctx, e.Question.ID, "answer from "+m.spec.Manager.Package)
	case e.Answer != nil:
		m.answers[e.Answer.ID] = e.Answer.Text
	case e.Handoff != nil:
		m.handoffs = append(m.handoffs, *e.Handoff)
		m.pending = append(m.pending, *e.Handoff)
		m.approved[e.Handoff.Manager] = false
	}
	return nil
}

func (m *collaborator) next(ctx context.Context) error {
	e, err := m.spec.Tools.Await(ctx)
	if err != nil {
		return err
	}
	return m.handle(ctx, e)
}

// ask sends one question along the hierarchy and waits for its answer.
func (m *collaborator) ask(ctx context.Context, to, text string) (string, error) {
	id, err := m.spec.Tools.Ask(ctx, to, text)
	for err == nil && m.answers[id] == "" {
		err = m.next(ctx)
	}
	return m.answers[id], err
}

func (m *collaborator) handoff(ctx context.Context) (run.Handoff, error) {
	for len(m.pending) == 0 {
		if err := m.next(ctx); err != nil {
			return run.Handoff{}, err
		}
	}
	h := m.pending[0]
	m.pending = m.pending[1:]
	return h, nil
}

// complete delegates on the first task, approves every direct child's
// latest fixed handoff and records the mandate's ordinary judgment.
func (m *collaborator) complete(ctx context.Context) error {
	if !m.started {
		m.started = true
		for _, child := range m.spec.Manager.Children {
			if err := m.spec.Tools.Delegate(ctx, child, "Realize your modeled responsibility.", nil); err != nil {
				return err
			}
			m.approved[child] = false
		}
	}
	decision := run.Submitted
	if m.spec.Mandate != run.Implement {
		decision = run.Approve
	}
	for {
		for m.unapproved() || len(m.pending) != 0 {
			h, err := m.handoff(ctx)
			if err != nil {
				return err
			}
			if err = m.spec.Tools.Review(ctx, h.Manager, h.Result.ID, run.Approve, nil); err != nil {
				return err
			}
			m.approved[h.Manager] = true
		}
		err := m.judge(ctx, decision, nil)
		if err == nil || !strings.HasPrefix(err.Error(), "child ") {
			return err
		}
		// The refusal names a child that works again: await its handoff.
		m.refusals = append(m.refusals, err.Error())
		if err = m.next(ctx); err != nil {
			return err
		}
	}
}

func (m *collaborator) unapproved() bool {
	for _, child := range m.spec.Manager.Children {
		if !m.approved[child] {
			return true
		}
	}
	return false
}

func (m *collaborator) judge(ctx context.Context, decision run.Decision, findings []run.Finding) error {
	err := m.spec.Tools.Judge(ctx, decision, findings)
	if err == nil {
		m.judgments++
		m.open = false
	}
	return err
}

func (m *collaborator) write(name, content string) error {
	return os.WriteFile(filepath.Join(m.spec.Workspace.Location().Directory, filepath.FromSlash(name)), []byte(content), 0600)
}

func (c *collaboration) execute(t *testing.T, id string, mandate run.Mandate) run.Result {
	t.Helper()
	_, prepared, git, input := runFixture(t)
	records, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	request := run.Request{ID: id, Project: "fixture", Prepared: prepared, Input: input, Mandate: mandate, MaxManagers: 5, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	result, err := (&run.Executor{Sessions: c, Workspaces: git, Records: records}).Execute(context.Background(), request)
	if c.stops {
		c.assertNoHostNarration(t)
		c.err = err
		return result
	}
	if err != nil || result.Technical != run.Completed {
		t.Fatalf("Run did not complete: %s %v %v", result.Technical, result.Errors, err)
	}
	for _, closure := range result.Cleanup {
		if !closure.Closed {
			t.Fatal("live resource after Run", closure)
		}
	}
	c.assertNoHostNarration(t)
	if mandate == run.Implement {
		destination := t.TempDir()
		if err = git.Reopen(context.Background(), *result.Candidate, destination); err != nil {
			t.Fatal(err)
		}
		c.candidate = destination
	}
	return result
}

func (c *collaboration) read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.candidate, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// V17: host messages carry events and concrete diagnostics, never an
// explanation of the host's own process states.
func (c *collaboration) assertNoHostNarration(t *testing.T) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	for name, m := range c.managers {
		for _, message := range m.messages {
			if len(message.Events) == 0 && len(message.Feedback) == 0 {
				t.Fatal("empty host message", name)
			}
			for _, f := range message.Feedback {
				// Naming a tool as the next step is allowed; describing the
				// host's process is not.
				for _, narration := range []string{"native response", "continuation", "drain", "terminal", "subsequent message", "same Session", "late message", "fresh mandate-appropriate", "judge call", "inbox", "driver", "schedul"} {
					if strings.Contains(strings.ToLower(f.Text), narration) {
						t.Fatal("host message explains host state", name, f.Text)
					}
				}
			}
		}
	}
}

func handoffOf(result run.Result, manager string) run.Handoff {
	for _, h := range result.Handoffs {
		if h.Manager == manager {
			return h
		}
	}
	return run.Handoff{}
}

func feedbackText(message run.Message) string {
	var parts []string
	for _, f := range message.Feedback {
		parts = append(parts, f.Text)
	}
	return strings.Join(parts, "\n")
}

// §13.3: a question answered before the first judgment requires no judgment
// and is no failure; an answered question is no new assignment or judgment;
// later communication does not change a reviewed fixed handoff.
func TestQuestionsBeforeAndAfterJudgmentNeedNoJudgment(t *testing.T) {
	for _, mandate := range []run.Mandate{run.Assessment, run.Implement, run.Verify} {
		t.Run(string(mandate), func(t *testing.T) { questionsBeforeAndAfterJudgment(t, mandate) })
	}
}

func questionsBeforeAndAfterJudgment(t *testing.T, mandate run.Mandate) {
	aIdle, aAnswered := make(chan struct{}), make(chan struct{})
	var reviewedB run.Handoff
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		switch m.spec.Manager.Package {
		case "a":
			switch len(m.messages) {
			case 1:
				// Delegate and end the response without judging or waiting.
				if err := m.handle(ctx, message.Events[0]); err != nil {
					return true, err
				}
				m.started = true
				m.approved["a/deep"] = false
				err := m.spec.Tools.Delegate(ctx, "a/deep", "Realize your modeled responsibility.", nil)
				close(aIdle)
				return true, err
			case 2:
				if len(message.Events) != 1 || message.Events[0].Question == nil || len(message.Feedback) != 0 {
					return true, fmt.Errorf("question before the first judgment came with more than the question: %+v", message)
				}
				err := m.handle(ctx, message.Events[0])
				close(aAnswered)
				return true, err
			}
		case "a/deep":
			if len(m.messages) == 1 {
				select {
				case <-aAnswered:
				case <-ctx.Done():
					return true, ctx.Err()
				}
			}
		case ".":
			if len(m.messages) != 1 {
				return false, nil
			}
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			m.started = true
			for _, child := range m.spec.Manager.Children {
				m.approved[child] = false
				if err := m.spec.Tools.Delegate(ctx, child, "Realize your modeled responsibility.", nil); err != nil {
					return true, err
				}
			}
			select {
			case <-aIdle:
			case <-ctx.Done():
				return true, ctx.Err()
			}
			if answer, err := m.ask(ctx, "a", "What do you need before your first judgment?"); err != nil || answer != "answer from a" {
				return true, fmt.Errorf("early answer %q: %v", answer, err)
			}
			if err := m.complete(ctx); err != nil {
				return true, err
			}
			for _, h := range m.handoffs {
				if h.Manager == "b" {
					reviewedB = h
				}
			}
			answer, err := m.ask(ctx, "b", "Explain your reviewed fixed handoff.")
			if err == nil && answer != "answer from b" {
				err = fmt.Errorf("late answer %q", answer)
			}
			return true, err
		}
		return false, nil
	}
	result := c.execute(t, "questions-"+string(mandate), mandate)
	if want := map[run.Mandate]run.Decision{run.Assessment: run.Approve, run.Implement: run.Submitted, run.Verify: run.Approve}[mandate]; result.Decision != want {
		t.Fatal("Run outcome changed", result.Decision)
	}
	a := c.manager("a")
	if a.judgments != 1 || len(a.messages) != 3 || a.messages[2].Events[0].Handoff == nil || len(a.messages[2].Feedback) != 0 {
		t.Fatal("answering before the first judgment demanded a judgment or a continuation", a.judgments, a.messages)
	}
	b := c.manager("b")
	if b.judgments != 1 || len(b.messages) != 2 || b.messages[1].Events[0].Question == nil {
		t.Fatal("an answered question became a new assignment or judgment", b.judgments, len(b.messages))
	}
	if final := handoffOf(result, "b"); final.Result != reviewedB.Result || final.Review == nil || final.Review.Decision != run.Approve {
		t.Fatal("later communication changed the reviewed fixed handoff", final.Result, reviewedB.Result)
	}
}

// §13.3: new mandatory Findings invalidate the previous judgment. A correction
// is a new task; the old judgment is not reused even for an unchanged state.
func TestCorrectionFindingsRequireANewJudgment(t *testing.T) {
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		switch m.spec.Manager.Package {
		case "a":
			if len(m.messages) != 1 {
				return false, nil
			}
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			m.started = true
			if err := m.spec.Tools.Delegate(ctx, "a/deep", "Realize your modeled responsibility.", nil); err != nil {
				return true, err
			}
			first, err := m.handoff(ctx)
			if err != nil {
				return true, err
			}
			findings := []run.Finding{{Text: "out/deep.txt must read corrected."}}
			if err = m.spec.Tools.Review(ctx, "a/deep", first.Result.ID, run.Reject, findings); err != nil {
				return true, err
			}
			if err = m.spec.Tools.Delegate(ctx, "a/deep", "Correct out/deep.txt.", findings); err != nil {
				return true, err
			}
			second, err := m.handoff(ctx)
			if err != nil {
				return true, err
			}
			if second.Result == first.Result || !strings.Contains(second.Diff, "+corrected") {
				return true, fmt.Errorf("correction handoff reused the first state")
			}
			if err = m.spec.Tools.Review(ctx, "a/deep", second.Result.ID, run.Approve, nil); err != nil {
				return true, err
			}
			return true, m.judge(ctx, run.Submitted, nil)
		case "a/deep":
			switch len(m.messages) {
			case 1:
				if err := m.write("out/deep.txt", "first\n"); err != nil {
					return true, err
				}
				return true, m.judge(ctx, run.Submitted, nil)
			case 2:
				// The correction arrives; this response ends without a judgment
				// and without any change to the previously judged state.
				if len(message.Events) != 1 || message.Events[0].Assignment == nil || message.Events[0].Assignment.Findings[0].Manager != "a" {
					return true, fmt.Errorf("correction lost its Findings or origin: %+v", message)
				}
				return true, nil
			case 3:
				if len(message.Events) != 0 || feedbackText(message) != "Continue your assignment: Correct out/deep.txt." {
					return true, fmt.Errorf("active assignment was not continued plainly: %+v", message)
				}
				if err := m.write("out/deep.txt", "corrected\n"); err != nil {
					return true, err
				}
				return true, m.judge(ctx, run.Submitted, nil)
			}
		}
		return false, nil
	}
	result := c.execute(t, "new-findings", run.Implement)
	deep := c.manager("a/deep")
	if result.Decision != run.Submitted || deep.judgments != 2 || len(deep.messages) != 3 || len(c.manager("a").handoffs) != 2 {
		t.Fatal("the previous judgment completed the corrected assignment", result.Decision, deep.judgments, len(deep.messages))
	}
	if got := c.read(t, "out/deep.txt"); got != "corrected\n" {
		t.Fatal("corrected work missing", got)
	}
}

// §13.3: an approval of the old assessed state does not apply to the new one.
func TestStateChangeAfterJudgmentRequiresANewJudgment(t *testing.T) {
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if m.spec.Manager.Package != "a/deep" {
			return false, nil
		}
		switch len(m.messages) {
		case 1:
			if err := m.write("out/deep.txt", "judged\n"); err != nil {
				return true, err
			}
			if err := m.judge(ctx, run.Submitted, nil); err != nil {
				return true, err
			}
			return true, m.write("out/deep.txt", "after judgment\n")
		case 2:
			if len(message.Events) != 0 || !strings.Contains(feedbackText(message), "your working state changed after your judgment") || !strings.Contains(feedbackText(message), "check your current state and judge again") {
				return true, fmt.Errorf("changed state did not invalidate the judgment: %+v", message)
			}
			return true, m.judge(ctx, run.Submitted, nil)
		}
		return false, nil
	}
	result := c.execute(t, "changed-state", run.Implement)
	deep := c.manager("a/deep")
	handoffs := c.manager("a").handoffs
	if result.Decision != run.Submitted || deep.judgments != 2 || len(handoffs) != 1 || !strings.Contains(handoffs[0].Diff, "+after judgment") {
		t.Fatal("a judgment of the old state completed the new one", deep.judgments, len(handoffs))
	}
	if got := c.read(t, "out/deep.txt"); got != "after judgment\n" {
		t.Fatal("handoff is not the judged state", got)
	}
}

// §13.3 at Root completion: Run relies on Root's earlier judgment only after
// checking it again; a later change of Root's state refuses completion.
func TestRootCompletionRefusesAnEarlierJudgmentAfterItsStateChanged(t *testing.T) {
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		switch m.spec.Manager.Package {
		case ".":
			switch len(m.messages) {
			case 1:
				for _, e := range message.Events {
					if err := m.handle(ctx, e); err != nil {
						return true, err
					}
				}
				if err := m.complete(ctx); err != nil {
					return true, err
				}
				// End the response with this question open.
				_, err := m.spec.Tools.Ask(ctx, "b", "Confirm your reviewed handoff.")
				return true, err
			case 2:
				// b's answer and b's request arrive after Root judged.
				for _, e := range message.Events {
					if e.Question != nil {
						if err := m.write(".gitignore", "*.ignored\n# reviewed by Root\n"); err != nil {
							return true, err
						}
					}
					if err := m.handle(ctx, e); err != nil {
						return true, err
					}
				}
				return true, nil
			case 3:
				if len(message.Events) != 0 || !strings.Contains(feedbackText(message), "your working state changed after your judgment") || !strings.Contains(feedbackText(message), "check your current state and judge again") {
					return true, fmt.Errorf("Root completion relied on its earlier judgment: %+v", message)
				}
				return true, m.judge(ctx, run.Submitted, nil)
			}
		case "b":
			if len(m.messages) == 2 {
				if err := m.handle(ctx, message.Events[0]); err != nil {
					return true, err
				}
				_, err := m.spec.Tools.Ask(ctx, ".", "Record the review in .gitignore.")
				return true, err
			}
		}
		return false, nil
	}
	result := c.execute(t, "root-state", run.Implement)
	root := c.manager(".")
	if result.Decision != run.Submitted || root.judgments != 2 || len(root.messages) != 3 {
		t.Fatal("Root completion accepted a judgment of an older state", root.judgments, len(root.messages))
	}
	if got := c.read(t, ".gitignore"); !strings.Contains(got, "# reviewed by Root") {
		t.Fatal("Root candidate is not the re-judged state", got)
	}
}

// §13.3: a late adverse finding stays in force. It invalidates the parent's
// acceptance and every judgment resting on it, up to Root, which must judge
// again before it can complete. A late model blocker prevents success.
func TestLateFindingInvalidatesAcceptanceAndJudgmentsAbove(t *testing.T) {
	for _, late := range []run.Decision{run.Reject, run.ModelBlocker} {
		t.Run(string(late), func(t *testing.T) {
			var stale run.Handoff
			var correction error
			c := &collaboration{}
			c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
				switch m.spec.Manager.Package {
				case ".":
					if len(m.messages) != 1 {
						return false, nil
					}
					for _, e := range message.Events {
						if err := m.handle(ctx, e); err != nil {
							return true, err
						}
					}
					if err := m.complete(ctx); err != nil {
						return true, err
					}
					// Root judged; it ends the response with this question open.
					_, err := m.spec.Tools.Ask(ctx, "a", "Reassess with your child.")
					return true, err
				case "a":
					if len(m.messages) != 2 {
						return false, nil
					}
					question := message.Events[0].Question
					if question == nil {
						return true, fmt.Errorf("expected Root's question: %+v", message)
					}
					if answer, err := m.ask(ctx, "a/deep", "Reassess the already accepted handoff."); err != nil || answer != "late finding" {
						return true, fmt.Errorf("late answer %q: %v", answer, err)
					}
					// a/deep's late judgment reopened this handed-off assignment.
					h, err := m.handoff(ctx)
					if err != nil {
						return true, err
					}
					stale = h
					if h.Decision != late || h.Review != nil {
						return true, fmt.Errorf("late finding not in force: %+v", h)
					}
					if late == run.Reject {
						findings := []run.Finding{{Text: "Resolve the late finding."}}
						if err = m.spec.Tools.Review(ctx, "a/deep", h.Result.ID, run.Reject, findings); err != nil {
							return true, err
						}
						if err = m.spec.Tools.Delegate(ctx, "a/deep", "Resolve the late finding.", findings); err != nil {
							return true, err
						}
						if h, err = m.handoff(ctx); err != nil {
							return true, err
						}
					} else {
						correction = m.spec.Tools.Delegate(ctx, "a/deep", "Resolve the late finding.", []run.Finding{{Text: "Resolve it."}})
					}
					if err = m.spec.Tools.Review(ctx, "a/deep", h.Result.ID, run.Approve, nil); err != nil {
						return true, err
					}
					if err = m.spec.Tools.Reply(ctx, question.ID, "reassessed"); err != nil {
						return true, err
					}
					return true, m.judge(ctx, run.Submitted, nil)
				case "a/deep":
					if len(m.messages) == 2 && message.Events[0].Question != nil {
						// Judge before answering: a awaits the changed handoff once
						// it has the answer, and would otherwise be refused.
						if err := m.spec.Tools.Judge(ctx, late, []run.Finding{{Text: "The accepted handoff refutes an assumption."}}); err != nil {
							return true, err
						}
						return true, m.spec.Tools.Reply(ctx, message.Events[0].Question.ID, "late finding")
					}
				}
				return false, nil
			}
			result := c.execute(t, "late-"+string(late), run.Implement)
			root, a := c.manager("."), c.manager("a")
			if root.judgments != 2 || a.judgments != 2 || len(root.handoffs) != 3 {
				t.Fatal("judgments resting on the refuted acceptance were reused", root.judgments, a.judgments, len(root.handoffs))
			}
			// Root was idle when its judgment stopped holding; its next message
			// names the concrete reason together with a's renewed handoff.
			explained := false
			for _, message := range root.messages {
				explained = explained || strings.Contains(feedbackText(message), "a/deep reported "+string(late)+" about its fixed handoff "+stale.Result.ID+", so a works on its assignment again; await its renewed handoff, review it and judge again")
			}
			if !explained {
				t.Fatal("Root's invalidated judgment was not explained", root.messages)
			}
			if late == run.Reject {
				if result.Decision != run.Submitted || handoffOf(result, "a/deep").Decision != run.Submitted || c.manager("a/deep").judgments != 2 {
					t.Fatal("corrected late Reject remained latched", result.Decision, handoffOf(result, "a/deep"))
				}
				return
			}
			if correction == nil || !strings.Contains(correction.Error(), "model blocker") || result.Decision != run.ModelBlocker || handoffOf(result, "a").Decision != run.ModelBlocker {
				t.Fatal("late model blocker did not prevent successful completion", correction, result.Decision)
			}
		})
	}
}

// V03: questions follow the hierarchy; sibling and level-skipping questions
// are refused with the allowed addressees.
func TestQuestionsOutsideTheHierarchyAreRefused(t *testing.T) {
	c := &collaboration{}
	refusals := map[string]string{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if len(m.messages) != 1 {
			return false, nil
		}
		targets := map[string]string{"a": "b", "a/deep": ".", ".": "a/deep", "b/empty": "a"}
		if to, ok := targets[m.spec.Manager.Package]; ok {
			_, err := m.spec.Tools.Ask(ctx, to, "Is this allowed?")
			if err == nil {
				return true, fmt.Errorf("%s asked %s outside the hierarchy", m.spec.Manager.Package, to)
			}
			c.mu.Lock()
			refusals[m.spec.Manager.Package] = err.Error()
			c.mu.Unlock()
		}
		if m.spec.Manager.Package == "a/deep" {
			if answer, err := m.ask(ctx, "a", "Direct parent question."); err != nil || answer != "answer from a" {
				return true, fmt.Errorf("direct parent question failed: %q %v", answer, err)
			}
		}
		return false, nil
	}
	result := c.execute(t, "hierarchy", run.Assessment)
	if result.Decision != run.Approve {
		t.Fatal(result.Decision)
	}
	for manager, allowed := range map[string]string{"a": "(., a/deep)", "a/deep": "(a)", ".": "(a, b)", "b/empty": "(b)"} {
		if !strings.Contains(refusals[manager], allowed) || !strings.Contains(refusals[manager], "through the Managers in between") {
			t.Fatal("refusal does not name the allowed addressees", manager, refusals[manager])
		}
	}
}

// A response that only handled communication neither ends the assignment
// falsely nor leaves it stalled: Run continues it once, naming the task.
func TestCommunicationOnlyResponseIsContinuedNotEnded(t *testing.T) {
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if m.spec.Manager.Package != "b/empty" {
			return false, nil
		}
		switch len(m.messages) {
		case 1:
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			answer, err := m.ask(ctx, "b", "Which bytes does an empty Package own?")
			if err == nil && answer != "answer from b" {
				err = fmt.Errorf("answer %q", answer)
			}
			return true, err
		case 2:
			if len(message.Events) != 0 || feedbackText(message) != "Continue your assignment: Realize your modeled responsibility." {
				return true, fmt.Errorf("open assignment was not continued purposefully: %+v", message)
			}
			return true, m.judge(ctx, run.Submitted, nil)
		}
		return false, nil
	}
	result := c.execute(t, "communication-only", run.Implement)
	if result.Decision != run.Submitted || len(c.manager("b/empty").messages) != 2 || len(c.manager("b").handoffs) != 1 {
		t.Fatal("communication ended the assignment, or it stalled", result.Decision, len(c.manager("b/empty").messages), len(c.manager("b").handoffs))
	}
}

// A child builds in its copy and submits with the build output still there,
// and its parent builds in its own copy before integrating. Neither output
// enters a handoff, an integration or the candidate, so the child's first
// submission passes its check and is accepted. An unaccounted product file
// still fails it (TestInvalidChildCandidateReachesParentAndIsCorrected).
func TestBuildOutputsNeverReachHandoffsOrTheCandidate(t *testing.T) {
	elf := "\x7fELF\x02\x01\x01\x00go build output"
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if len(message.Events) == 0 || message.Events[0].Assignment == nil {
			return false, nil
		}
		switch m.spec.Manager.Package {
		case "a":
			return false, m.write("out/a-tool", elf)
		case "a/deep":
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			for _, file := range [][2]string{{"out/deep.txt", "built\n"}, {"out/deep-tool", elf}} {
				if err := m.write(file[0], file[1]); err != nil {
					return true, err
				}
			}
			if err := m.judge(ctx, run.Submitted, nil); err != nil {
				return true, err
			}
			status, err := localCommand(ctx, m.spec.Workspace.Location().Directory, nil, "status", "--porcelain", "--untracked-files=all")
			if err == nil && status != "?? out/deep-tool" {
				err = fmt.Errorf("build output left the child's copy or became tracked: %q", status)
			}
			return true, err
		}
		return false, nil
	}
	result := c.execute(t, "build-outputs", run.Implement)
	deep, a := c.manager("a/deep"), c.manager("a")
	if result.Decision != run.Submitted || len(deep.tasks) != 1 || deep.judgments != 1 || len(a.handoffs) != 1 || len(a.refusals) != 0 {
		t.Fatal("build output caused a correction", result.Decision, len(deep.tasks), deep.judgments, len(a.handoffs), a.refusals)
	}
	if h := a.handoffs[0]; h.Check == nil || !h.Check.Valid || strings.Contains(h.Diff, "deep-tool") || !strings.Contains(h.Diff, "+built") {
		t.Fatal("child handoff carried its build output or lost its change", h.Check, h.Diff)
	}
	if c.read(t, "out/deep.txt") != "built\n" {
		t.Fatal("accepted child change lost")
	}
	for _, name := range []string{"out/deep-tool", "out/a-tool"} {
		if _, err := os.Stat(filepath.Join(c.candidate, filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Fatal("build output reached the candidate", name, err)
		}
	}
	history, err := localCommand(context.Background(), c.candidate, nil, "rev-list", "--objects", "--all")
	if err != nil || strings.Contains(history, "-tool") {
		t.Fatal("build output reached the candidate's history", err)
	}
}

// Continued responses that do nothing at all do not loop: no tool call, no
// native execution and no working-state change. The Run ends technically
// incomplete with a diagnostic naming the Manager.
func TestRepeatedNoProgressEndsTheRunIncomplete(t *testing.T) {
	c := &collaboration{stops: true}
	c.script = func(_ context.Context, m *collaborator, _ run.Message) (bool, error) {
		return m.spec.Manager.Package == "a/deep", nil
	}
	result := c.execute(t, "no-progress", run.Implement)
	deep := c.manager("a/deep")
	if !errors.Is(c.err, run.ErrNoProgress) || result.Technical != run.Incomplete || result.Candidate != nil || result.Messages > 20 {
		t.Fatal("idle Manager looped or ended as success", result.Technical, result.Messages, c.err)
	}
	if !strings.Contains(strings.Join(result.Errors, " "), "Manager a/deep took no action in 2 continued responses") {
		t.Fatal("stop lacks its concrete diagnostic", result.Errors)
	}
	if len(deep.messages) != 3 || feedbackText(deep.messages[1]) != "Continue your assignment: Realize your modeled responsibility." || feedbackText(deep.messages[2]) != "Your last response took no action; work on your assignment or judge it: Realize your modeled responsibility." {
		t.Fatal("continuations were not bounded and concrete", deep.messages)
	}
}

// Read-only review is activity, not standstill: a reviewer spends continued
// responses only on check, read_notes, native commands and a refused await,
// none of which changes its copy, and then judges without being stopped.
func TestReadOnlyReviewIsNotStopped(t *testing.T) {
	for _, mandate := range []run.Mandate{run.Assessment, run.Verify} {
		t.Run(string(mandate), func(t *testing.T) {
			c := &collaboration{}
			c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
				if m.spec.Manager.Package != "a/deep" {
					return false, nil
				}
				step := len(m.messages)
				if step == 1 {
					return true, m.handle(ctx, message.Events[0])
				}
				if feedbackText(message) != "Continue your assignment: Realize your modeled responsibility." {
					return true, fmt.Errorf("read-only review was treated as standstill: %+v", message)
				}
				switch step {
				case 2:
					check, err := m.spec.Tools.Check(ctx)
					if err == nil && !check.Valid {
						err = fmt.Errorf("fixed input fails its check: %s", check.Diagnostic)
					}
					return true, err
				case 3:
					_, err := m.spec.Tools.ReadNotes(ctx, "")
					return true, err
				case 4:
					// Reading files and running tests in the shell, as the
					// adapter observes it; nothing changes in the copy.
					m.executed = true
					return true, nil
				case 5:
					if _, err := m.spec.Tools.Await(ctx); err == nil || !strings.Contains(err.Error(), "nothing can arrive") {
						return true, fmt.Errorf("await was not refused: %v", err)
					}
					return true, nil
				}
				return true, m.judge(ctx, run.Approve, nil)
			}
			result := c.execute(t, "read-only-"+string(mandate), mandate)
			deep := c.manager("a/deep")
			if result.Decision != run.Approve || deep.judgments != 1 || len(deep.messages) != 6 {
				t.Fatal("read-only reviewer was stopped or judged early", result.Decision, deep.judgments, len(deep.messages))
			}
		})
	}
}

// Work in the private copy is progress even without a Run tool call.
func TestProgressingContinuationsAreNotStopped(t *testing.T) {
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if m.spec.Manager.Package != "a/deep" {
			return false, nil
		}
		step := len(m.messages)
		if step > 1 && feedbackText(message) != "Continue your assignment: Realize your modeled responsibility." {
			return true, fmt.Errorf("progress was treated as idleness: %+v", message)
		}
		if step < 4 {
			return true, m.write("out/deep.txt", fmt.Sprintf("step %d\n", step))
		}
		return true, m.judge(ctx, run.Submitted, nil)
	}
	result := c.execute(t, "progress", run.Implement)
	if result.Decision != run.Submitted || len(c.manager("a/deep").messages) != 4 || c.read(t, "out/deep.txt") != "step 3\n" {
		t.Fatal("progressing Manager was stopped or lost its work", result.Decision, len(c.manager("a/deep").messages))
	}
}
