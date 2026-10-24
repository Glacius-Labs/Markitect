package run

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// inboxExecution builds Root ".", its child "a" and grandchild "a/deep" with
// controlled ports; no Session runs unless a test supplies one.
func inboxExecution() (*execution, *managerState, *managerState, *managerState) {
	ctx, cancel := context.WithCancel(context.Background())
	root := &managerState{manager: organization.Manager{Package: ".", Children: []string{"a"}}, signal: make(chan struct{}, 1)}
	a := &managerState{manager: organization.Manager{Package: "a", Parent: ".", Children: []string{"a/deep"}}, signal: make(chan struct{}, 1)}
	deep := &managerState{manager: organization.Manager{Package: "a/deep", Parent: "a"}, signal: make(chan struct{}, 1)}
	x := &execution{ctx: ctx, cancel: cancel, settled: make(chan struct{}, 1), request: Request{ID: "inbox-test", Mandate: Implement, Allowance: NewAllowance(time.Now().Add(time.Minute), 10)}, states: map[string]*managerState{".": root, "a": a, "a/deep": deep}, questions: map[string]*question{}}
	return x, root, a, deep
}

// accepted puts s into the reviewed phase with an approved fixed handoff and
// the judgment that produced it.
func accepted(s *managerState, commit string) {
	s.a.phase = reviewed
	s.a.version = 1
	s.a.judgment = &judgment{decision: Submitted}
	s.a.handoff = Handoff{Manager: s.manager.Package, Result: Commit{ID: commit, Tree: commit + "-tree"}, Decision: Submitted, Review: &ParentReview{Decision: Approve}}
	s.a.gen, s.a.sent = 1, 1
}

func TestLateRejectInvalidatesAcceptanceWithoutRejectingAncestors(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.closing = true // observe state only; no driver starts
	accepted(deep, "deep")
	a.a.phase, a.a.version = active, 1
	a.a.judgment = &judgment{decision: Submitted}
	root.a.phase, root.a.version = active, 1
	x.mu.Lock()
	x.lateLocked(deep, Reject, origin([]Finding{{Text: "late required correction"}}, "a/deep"))
	x.mu.Unlock()
	if deep.a.phase != fixed || deep.a.handoff.Review != nil || deep.a.handoff.Decision != Reject || deep.a.late != Reject || deep.a.gen != 2 {
		t.Fatal("late finding did not stay in force or keep the old acceptance", deep.a)
	}
	if f := deep.a.handoff.Findings; len(f) != 1 || f[0].Manager != "a/deep" {
		t.Fatal("late Finding lost its host-set origin", f)
	}
	if a.a.judgment != nil || len(a.a.feedback) != 1 || !strings.Contains(a.a.feedback[0].Text, "a/deep reported reject") {
		t.Fatal("parent judgment resting on the acceptance stayed valid", a.a)
	}
	if len(a.queue) != 1 || a.queue[0].child != "a/deep" || a.queue[0].gen != 2 {
		t.Fatal("parent did not receive the changed handoff", a.queue)
	}
	if root.a.phase != active || root.a.handoff.Decision != "" || len(root.a.feedback) != 0 {
		t.Fatal("late finding touched an ancestor above the nearest active one", root.a)
	}
	// A correction resolves the late Reject; a model blocker never is.
	deep.a.review(ParentReview{Decision: Reject})
	deep.a.assign(Assignment{Task: "correct"})
	if deep.a.late != "" || deep.a.phase != assigned {
		t.Fatal("correction kept the resolved late Reject", deep.a)
	}
}

func TestLateBlockerReopensHandedOffAncestorsUpToTheActiveOne(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.closing = true
	accepted(deep, "deep")
	accepted(a, "a")
	root.a.phase, root.a.version = active, 1
	root.a.judgment = &judgment{decision: Submitted}
	x.mu.Lock()
	x.lateLocked(deep, ModelBlocker, []Finding{{Manager: "a/deep", Text: "required model condition contradicts itself"}})
	x.mu.Unlock()
	if !deep.a.blocked() || deep.a.handoff.Decision != ModelBlocker {
		t.Fatal("late blocker was discarded", deep.a)
	}
	// a's handoff rests on its acceptance of a/deep: a works again, and Root's
	// judgment resting on a's handoff no longer holds. No decision is forced.
	if a.a.phase != active || a.a.judgment != nil || a.a.handoff.Decision != Submitted || len(a.a.feedback) != 1 {
		t.Fatal("handed-off parent was not reopened or its decision was forced", a.a)
	}
	if root.a.judgment != nil || root.a.phase != active || len(root.a.feedback) != 1 {
		t.Fatal("Root judgment resting on the reopened handoff stayed valid", root.a)
	}
	x.mu.Lock()
	err := x.childrenLocked(root)
	x.mu.Unlock()
	if err == nil || !strings.Contains(err.Error(), "child a is still working") {
		t.Fatal("Root could judge over a reopened child", err)
	}
	if concreteFindings([]Finding{{Text: " \n"}}) {
		t.Fatal("blank finding authorized correction")
	}
}

func TestOwnModelBlockerDominatesChildRejectAtCompletionAndAggregation(t *testing.T) {
	x, root, a, _ := inboxExecution()
	defer x.cancel()
	a.a.phase = reviewed
	a.a.handoff = Handoff{Manager: "a", Decision: Reject, Review: &ParentReview{Decision: Reject, Findings: []Finding{{Text: "child issue"}}}}
	if got := requiredDecision(ModelBlocker, x.childOutcomeLocked(root)); got != ModelBlocker {
		t.Fatal("child rejection downgraded own blocker", got)
	}
	if got := requiredDecision(Submitted, x.childOutcomeLocked(root)); got != Reject {
		t.Fatal("rejected child became submitted", got)
	}
}

func TestAskFollowsTheHierarchyAndAwaitNeverWaitsForNothing(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.closing = true // questions are refused at admission; refusal names come first
	for _, tc := range []struct {
		from *managerState
		to   string
		want string
	}{{deep, ".", "(a)"}, {root, "a/deep", "(a)"}, {a, "b", "(., a/deep)"}, {deep, "a/deep", "(a)"}} {
		_, err := x.ask(context.Background(), tc.from, tc.to, "question")
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "through the Managers in between") {
			t.Fatal("question outside the hierarchy was not refused with its addressees", tc.from.manager.Package, tc.to, err)
		}
	}
	x.closing = false
	if _, err := x.await(context.Background(), a); err == nil || !strings.Contains(err.Error(), "nothing can arrive") {
		t.Fatal("await waited without anything that can arrive", err)
	}
	// A working child that is idle itself waits for a: nothing can arrive.
	deep.a.phase = active
	x.questions["q"] = &question{Question: Question{ID: "q", From: "a/deep", To: "a", Text: "open"}, delivered: true}
	if _, err := x.await(context.Background(), a); err == nil || !strings.Contains(err.Error(), "answer your open questions: q from a/deep") {
		t.Fatal("await waited although every other Manager waits for it", err)
	}
	deep.running = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := x.await(ctx, a); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("await did not wait for a working child", err)
	}
}

func TestQuestionAnswerNeitherJudgesNorChangesTheAssignment(t *testing.T) {
	x, root, a, _ := inboxExecution()
	defer x.cancel()
	accepted(a, "a")
	root.a.phase, root.a.version = active, 1
	before := a.a
	// Keep drivers from starting: Sessions are absent in this unit test.
	root.running, a.running = true, true
	id, err := x.ask(context.Background(), root, "a", "Explain your fixed handoff.")
	if err != nil {
		t.Fatal(err)
	}
	event, err := x.await(context.Background(), a)
	if err != nil || event.Question == nil || event.Question.ID != id {
		t.Fatal(event, err)
	}
	if err = x.reply(context.Background(), root, id, "forged"); err == nil {
		t.Fatal("foreign Manager replied")
	}
	if err = x.reply(context.Background(), a, id, "answer"); err != nil {
		t.Fatal(err)
	}
	if err = x.reply(context.Background(), a, id, "duplicate"); err == nil {
		t.Fatal("duplicate reply admitted")
	}
	event, err = x.await(context.Background(), root)
	if err != nil || event.Answer == nil || event.Answer.Text != "answer" || len(x.questions) != 0 {
		t.Fatal(event, err)
	}
	if !reflect.DeepEqual(a.a, before) || root.a.judgment != nil || root.a.phase != active {
		t.Fatal("question and answer changed an assignment", a.a, root.a)
	}
}

// These are controlled port peers. The real Git/record Executor test separately
// proves that a Session ignoring an integration error cannot persist success.
type toolFailureCopy struct {
	Copy
	stage   string
	failure error
}

func (c toolFailureCopy) Snapshot(context.Context) (Commit, error) {
	if c.stage == "snapshot" {
		return Commit{}, c.failure
	}
	return Commit{ID: "fixed", Tree: "tree"}, nil
}
func (c toolFailureCopy) Update(context.Context, Copy, Commit) error {
	if c.stage == "update" {
		return c.failure
	}
	return nil
}
func (c toolFailureCopy) Review(context.Context, Copy, Commit, Commit) (string, error) {
	if c.stage == "diff" {
		return "", c.failure
	}
	return "full diff", nil
}
func (c toolFailureCopy) Integrate(context.Context, Copy, Commit) (Commit, error) {
	if c.stage == "integrate" {
		return Commit{}, c.failure
	}
	return Commit{ID: "fixed", Tree: "tree"}, nil
}

type toolFailureRecords struct {
	Records
	failure error
}

func (r toolFailureRecords) Notes(context.Context, string) ([]Note, error) { return nil, r.failure }
func (r toolFailureRecords) Append(context.Context, Note) (Note, error)    { return Note{}, r.failure }

func TestIgnoredToolPortErrorsStillStopExecution(t *testing.T) {
	for _, stage := range []string{"notes", "append", "delegate-snapshot", "update", "diff", "review-snapshot", "integrate", "judge-snapshot", "check-snapshot"} {
		t.Run(stage, func(t *testing.T) {
			x, _, parent, child := inboxExecution()
			defer x.cancel()
			injected := errors.New("actual port failure " + stage)
			x.executor = &Executor{Records: toolFailureRecords{failure: injected}}
			parent.copy = toolFailureCopy{failure: injected}
			child.copy = toolFailureCopy{failure: injected}
			parent.a.phase, parent.a.version = active, 1
			// No driver starts: this test calls the tools directly.
			parent.running, child.running = true, true
			switch stage {
			case "delegate-snapshot", "review-snapshot", "judge-snapshot", "check-snapshot":
				parent.copy = toolFailureCopy{stage: "snapshot", failure: injected}
				if stage == "judge-snapshot" {
					accepted(child, "deep")
				}
			case "diff":
				parent.copy = toolFailureCopy{stage: "diff", failure: injected}
			case "integrate":
				parent.copy = toolFailureCopy{stage: "integrate", failure: injected}
			case "update":
				child.copy = toolFailureCopy{stage: "update", failure: injected}
			}
			tools := x.tools(parent)
			switch stage {
			case "notes":
				_, _ = tools.ReadNotes(context.Background())
			case "append":
				_, _ = tools.AppendNote(context.Background(), "scope", "durable note")
			case "delegate-snapshot":
				_ = tools.Delegate(context.Background(), "a/deep", "concrete assignment", nil)
			case "update":
				if err := tools.Delegate(context.Background(), "a/deep", "concrete assignment", nil); err != nil {
					t.Fatal(err)
				}
				_, _ = x.tools(child).Await(context.Background())
			case "diff":
				child.a.fix(Handoff{Manager: "a/deep", Result: Commit{ID: "fixed", Tree: "tree"}, Decision: Submitted})
				x.queueHandoffLocked(child)
				_, _ = tools.Await(context.Background())
			case "judge-snapshot":
				_ = tools.Judge(context.Background(), Submitted, nil)
			case "check-snapshot":
				_, _ = tools.Check(context.Background())
			default:
				child.a.fix(Handoff{Manager: "a/deep", Result: Commit{ID: "fixed", Tree: "tree"}, Decision: Submitted, Check: &Check{Valid: true}})
				child.a.sent = child.a.gen
				_ = tools.Review(context.Background(), "a/deep", "fixed", Approve, nil)
			}
			if x.ctx.Err() == nil || !errors.Is(x.asyncErr, injected) {
				t.Fatal("ignored port failure lost", x.ctx.Err(), x.asyncErr)
			}
		})
	}
}

func TestTerminalReceiptCutRejectsLateCallbacksAndDetachesStoredValues(t *testing.T) {
	x, _, _, _ := inboxExecution()
	defer x.cancel()
	observations := []CheckObservation{{ID: "observed", Receipt: CheckReceipt{Output: []byte("captured"), Directories: []string{"empty"}}}}
	if err := x.recordChecks(observations); err != nil {
		t.Fatal(err)
	}
	observations[0].Receipt.Output[0] = 'X'
	observations[0].Receipt.Directories[0] = "changed"
	x.result.Checks = x.finishChecks()
	terminal := copyResult(x.result)
	if string(terminal.Checks[0].Receipt.Output) != "captured" || terminal.Checks[0].Receipt.Directories[0] != "empty" {
		t.Fatal("producer owns terminal receipt", terminal)
	}
	saved := copyResult(terminal)
	// Deterministically exercise the same callback during Save and after return.
	for _, phase := range []string{"during Save", "after return"} {
		if err := x.recordChecks([]CheckObservation{{ID: phase}}); !errors.Is(err, ErrObservationIncomplete) {
			t.Fatal("late receipt accepted", phase, err)
		}
		if !reflect.DeepEqual(saved, terminal) || len(x.checks) != 1 {
			t.Fatal("terminal receipt collection changed", phase)
		}
	}
	x.result.Checks[0].Receipt.Output[0] = 'Y'
	x.checks[0].Receipt.Directories[0] = "late"
	if !reflect.DeepEqual(saved, terminal) {
		t.Fatal("worker collection owns persisted/caller result")
	}
}

type admissionSession struct{ calls atomic.Int32 }

type timedMessageSession struct{}

func (timedMessageSession) Exchange(ctx context.Context, _ Message) error {
	<-ctx.Done()
	// Even a provider returning without error at timeout cannot erase it.
	return nil
}
func (timedMessageSession) Close(context.Context) error { return nil }

func TestExchangeEnforcesPerMessageLimitWithinOriginalRunDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	x := &execution{ctx: ctx, request: Request{MessageTimeout: 20 * time.Millisecond}}
	start := time.Now()
	err := x.exchange(&managerState{session: timedMessageSession{}}, Message{Feedback: []Finding{{Text: "bounded"}}})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second || ctx.Err() != nil {
		t.Fatal("message ignored its own limit or consumed whole-Run deadline", err, ctx.Err())
	}
	x.calls.Wait()
}

func (s *admissionSession) Exchange(context.Context, Message) error {
	s.calls.Add(1)
	return errors.New("unexpected admitted Session")
}
func (*admissionSession) Close(context.Context) error { return nil }

type admissionCopy struct {
	toolFailureCopy
	entered, release chan struct{}
}

func (c admissionCopy) Snapshot(context.Context) (Commit, error) {
	close(c.entered)
	<-c.release
	return Commit{ID: "fixed", Tree: "tree"}, nil
}

func TestDelegateCannotPublishAssignmentAfterExternalGitReturnsToShutdown(t *testing.T) {
	for _, correction := range []bool{false, true} {
		for _, stop := range []string{"closing", "context"} {
			t.Run(fmt.Sprintf("correction=%v/%s", correction, stop), func(t *testing.T) {
				x, _, parent, child := inboxExecution()
				defer x.cancel()
				parent.a.phase, parent.a.version = active, 1
				entered, release := make(chan struct{}), make(chan struct{})
				parent.copy = admissionCopy{entered: entered, release: release}
				session := &admissionSession{}
				child.session = session
				if correction {
					accepted(child, "old")
				}
				before := child.a
				returned := make(chan error, 1)
				go func() {
					returned <- x.tools(parent).Delegate(context.Background(), "a/deep", "late task", []Finding{{Text: "concrete correction"}})
				}()
				<-entered
				x.mu.Lock()
				if stop == "closing" {
					x.closing = true
				} else {
					x.cancel()
				}
				x.mu.Unlock()
				close(release)
				select {
				case err := <-returned:
					if err == nil {
						t.Fatal("late delegation accepted")
					}
				case <-time.After(time.Second):
					t.Fatal("admission kept lock across external Git")
				}
				x.active.Wait()
				x.calls.Wait()
				x.mu.Lock()
				defer x.mu.Unlock()
				if !reflect.DeepEqual(before, child.a) || child.running || len(child.queue) != 0 || session.calls.Load() != 0 {
					t.Fatal("late delegation changed task state", child.a)
				}
			})
		}
	}
}

func TestShutdownRefusesDriverAndExchangeAdmission(t *testing.T) {
	for _, stop := range []string{"closing", "context"} {
		t.Run(stop, func(t *testing.T) {
			x, _, _, child := inboxExecution()
			defer x.cancel()
			session := &admissionSession{}
			child.session = session
			x.questions["pending"] = &question{Question: Question{ID: "pending", From: "a", To: "a/deep", Text: "queued question"}}
			x.mu.Lock()
			if stop == "closing" {
				x.closing = true
			} else {
				x.cancel()
			}
			x.queueLocked(child, entry{event: Event{Question: &x.questions["pending"].Question}})
			x.settleLocked()
			x.mu.Unlock()
			if err := x.exchange(child, Message{Feedback: []Finding{{Text: "late exchange"}}}); err == nil {
				t.Fatal("late Exchange admitted")
			}
			x.active.Wait()
			x.calls.Wait()
			if child.running || session.calls.Load() != 0 {
				t.Fatal("shutdown launched native work")
			}
		})
	}
}

// A judgment rests on the reviewed children. Delegating a correction after it
// invalidates it, so no handoff is fixed from the old judgment.
func TestDelegationAfterJudgmentInvalidatesIt(t *testing.T) {
	x, _, a, deep := inboxExecution()
	defer x.cancel()
	x.executor = &Executor{}
	a.copy, deep.copy = toolFailureCopy{}, toolFailureCopy{}
	a.running, deep.running = true, true
	accepted(deep, "deep")
	a.a.phase, a.a.version = active, 1
	old := &judgment{decision: Submitted, state: Commit{ID: "fixed", Tree: "tree"}}
	a.a.judge(old)
	if err := x.tools(a).Delegate(context.Background(), "a/deep", "correct", []Finding{{Text: "concrete correction"}}); err != nil {
		t.Fatal(err)
	}
	if a.a.judgment != nil || len(a.a.feedback) != 1 || a.a.feedback[0].Text != "you delegated to a/deep after your judgment" {
		t.Fatal("judgment survived a later delegation", a.a)
	}
	if err := x.fix(a, old); err != nil || a.a.phase != active {
		t.Fatal("a handoff was fixed from the invalidated judgment", a.a.phase, err)
	}
}

// A reopened Manager's own late model blocker stays in force when it judges
// and fixes its handoff again.
func TestReopenedManagerKeepsItsOwnLateBlocker(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.request.Mandate = Verify
	x.executor = &Executor{}
	a.copy = toolFailureCopy{}
	root.running = true
	accepted(deep, "deep")
	a.a.phase, a.a.late = active, ModelBlocker
	j := &judgment{decision: Approve}
	a.a.judge(j)
	if err := x.fix(a, j); err != nil {
		t.Fatal(err)
	}
	if a.a.phase != fixed || a.a.handoff.Decision != ModelBlocker {
		t.Fatal("own late model blocker was lost on reopening", a.a.phase, a.a.handoff.Decision)
	}
}
