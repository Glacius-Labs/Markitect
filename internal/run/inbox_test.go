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
	if a.a.judgment != nil || len(a.a.feedback) != 1 || !strings.Contains(a.a.feedback[0].Text, "a/deep reported reject") || !strings.HasSuffix(a.a.feedback[0].Text, "; review its changed handoff and judge again") {
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
	if root.a.judgment != nil || root.a.phase != active || len(root.a.feedback) != 1 || !strings.HasSuffix(root.a.feedback[0].Text, "so a works on its assignment again; await its renewed handoff, review it and judge again") {
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
	if _, err := x.await(context.Background(), a); err == nil || !strings.Contains(err.Error(), "nothing can arrive") || !strings.Contains(err.Error(), "you have no open work to wait for") {
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

type messageSession func(context.Context, Message) error

func (f messageSession) Exchange(ctx context.Context, m Message) error { return f(ctx, m) }
func (messageSession) Close(context.Context) error                     { return nil }

// A question takes no allowance of its own (phase C M-c3-C4: 231 asks used 256
// of 256 messages). A storm of questions reaches its idle receiver batched in
// one message, which is the only message charged.
func TestQuestionStormCostsOnlyItsBatchedDeliveryMessage(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	// The askers stay in their own responses; a is busy until all have asked.
	root.running, a.running, deep.running = true, true, true
	for _, from := range []*managerState{root, deep} {
		for i := range maxOpenQuestions {
			if _, err := x.tools(from).Ask(context.Background(), "a", fmt.Sprintf("question %d from %s", i, from.manager.Package)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if used := x.request.Allowance.Used(); used != 0 {
		t.Fatal("questions took allowance of their own", used)
	}
	var messages []Message
	tools := x.tools(a)
	a.session = messageSession(func(ctx context.Context, m Message) error {
		messages = append(messages, m)
		for _, e := range m.Events {
			if e.Question != nil {
				if err := tools.Reply(ctx, e.Question.ID, "answer"); err != nil {
					return err
				}
			}
		}
		return nil
	})
	x.mu.Lock()
	a.running = false
	x.scheduleLocked(a)
	x.mu.Unlock()
	x.active.Wait()
	x.calls.Wait()
	if len(messages) != 1 || len(messages[0].Events) != 2*maxOpenQuestions || x.request.Allowance.Used() != 1 {
		t.Fatal("the question storm was not delivered in one charged message", len(messages), x.request.Allowance.Used())
	}
	if len(root.queue) != maxOpenQuestions || len(deep.queue) != maxOpenQuestions {
		t.Fatal("answers did not reach the askers", len(root.queue), len(deep.queue))
	}
}

// Each Manager has at most maxOpenQuestions questions whose answer has not
// reached it. One more is refused with the reason and the next step, takes
// nothing and asks nothing; an answer makes room again once it is received.
func TestOpenQuestionCapRefusesWithReasonAndNextStep(t *testing.T) {
	x, root, a, _ := inboxExecution()
	defer x.cancel()
	root.running, a.running = true, true
	ctx := context.Background()
	var first string
	for i := range maxOpenQuestions {
		id, err := x.tools(root).Ask(ctx, "a", fmt.Sprintf("question %d", i))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = id
		}
	}
	if _, err := x.tools(root).Ask(ctx, "a", "one more"); err == nil || err.Error() != "you have 8 open questions; wait for answers with await before you ask again" {
		t.Fatal("a question above the cap was not refused with its next step", err)
	}
	if len(x.questions) != maxOpenQuestions || len(a.queue) != maxOpenQuestions || x.request.Allowance.Used() != 0 {
		t.Fatal("the refused question was kept, delivered or charged", len(x.questions), len(a.queue))
	}
	// The cap counts only the asker's own questions.
	if _, err := x.tools(a).Ask(ctx, ".", "child question"); err != nil {
		t.Fatal("questions to a Manager counted against it", err)
	}
	if err := x.tools(a).Reply(ctx, first, "answer"); err != nil {
		t.Fatal(err)
	}
	if _, err := x.tools(root).Ask(ctx, "a", "next question"); err == nil {
		t.Fatal("an answer root has not received made room")
	}
	for {
		e, err := x.tools(root).Await(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if e.Answer != nil && e.Answer.ID == first {
			break
		}
	}
	if _, err := x.tools(root).Ask(ctx, "a", "next question"); err != nil {
		t.Fatal("a received answer still counted as open", err)
	}
}

// WP5 (phase C M-c3-C8: a child answered its parent's questions with ask
// instead of reply and handed off; the host delivered them again until the
// no-progress rule stopped the Run). A question its addressee leaves
// unanswered at its handoff is closed, and the asker receives the reason and
// next step instead of an answer. A later question still reaches the
// handed-off Manager, which may answer it or report a late finding; left
// unanswered, it is closed the same way and never delivered again.
func TestQuestionsAManagerNoLongerAnswersAreClosedForTheAsker(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.request.Mandate = Assessment
	x.executor = &Executor{}
	a.copy, deep.copy = toolFailureCopy{}, toolFailureCopy{}
	// No driver starts: this test calls the tools and the end of deep's response.
	root.running, a.running, deep.running = true, true, true
	a.a.phase, a.a.version = active, 1
	deep.a.phase, deep.a.version = active, 1
	ctx := context.Background()
	closed := func(id string) {
		t.Helper()
		want := "a/deep has handed off without answering your question " + id + "; continue from its handoff or ask your parent"
		// A closure that never comes fails here instead of blocking the test.
		bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		for {
			e, err := x.tools(a).Await(bounded)
			if err != nil {
				t.Fatal(err)
			}
			if e.Answer != nil && e.Answer.ID == id {
				if e.Answer.Closed != want || e.Answer.Text != "" || e.Answer.From != "a/deep" {
					t.Fatalf("question %s closed without its reason and next step: %+v", id, e.Answer)
				}
				return
			}
		}
	}
	first, err := x.tools(a).Ask(ctx, "a/deep", "What do you need?")
	if err != nil {
		t.Fatal(err)
	}
	if e, err := x.tools(deep).Await(ctx); err != nil || e.Question == nil || e.Question.ID != first {
		t.Fatal("the question did not reach its addressee", e, err)
	}
	// deep answers with ask instead of reply, judges and hands off.
	if _, err := x.tools(deep).Ask(ctx, "a", "Nothing more."); err != nil {
		t.Fatal(err)
	}
	if err := x.tools(deep).Judge(ctx, Approve, nil); err != nil {
		t.Fatal(err)
	}
	if err := x.responded(deep); err != nil || deep.a.phase != fixed {
		t.Fatal("deep did not hand off", deep.a.phase, err)
	}
	closed(first)
	second, err := x.tools(a).Ask(ctx, "a/deep", "Explain your fixed handoff.")
	if err != nil {
		t.Fatal(err)
	}
	if e, err := x.tools(deep).Await(ctx); err != nil || e.Question == nil || e.Question.ID != second {
		t.Fatal("a question after the handoff did not reach the handed-off Manager", e, err)
	}
	// deep's response ends without a reply.
	if err := x.responded(deep); err != nil || len(deep.queue) != 0 {
		t.Fatal("the unanswered question was delivered again", deep.queue, err)
	}
	closed(second)
	if len(x.questions) != 1 || x.questions["inbox-test-q2"] == nil {
		t.Fatal("closed questions stayed open or deep's own question was lost", x.questions)
	}
}

// A question that reaches a Manager after its judgment but before its handoff
// is fixed is not closed unseen: the handed-off Manager receives it once and
// may still answer it.
func TestAQuestionQueuedDuringTheHandoffStillReachesTheManager(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	x.request.Mandate = Assessment
	x.executor = &Executor{}
	a.copy, deep.copy = toolFailureCopy{}, toolFailureCopy{}
	root.running, a.running, deep.running = true, true, true
	a.a.phase, a.a.version = active, 1
	deep.a.phase, deep.a.version = active, 1
	ctx := context.Background()
	if err := x.tools(deep).Judge(ctx, Approve, nil); err != nil {
		t.Fatal(err)
	}
	id, err := x.tools(a).Ask(ctx, "a/deep", "Reassess your handoff.")
	if err != nil {
		t.Fatal(err)
	}
	if err := x.responded(deep); err != nil || deep.a.phase != fixed {
		t.Fatal("deep did not hand off", deep.a.phase, err)
	}
	if q := x.questions[id]; q == nil || q.answered {
		t.Fatal("a question deep never received was closed at its handoff")
	}
	if e, err := x.tools(deep).Await(ctx); err != nil || e.Question == nil || e.Question.ID != id {
		t.Fatal("the question did not reach the handed-off Manager", e, err)
	}
	if err := x.tools(deep).Reply(ctx, id, "Still valid."); err != nil {
		t.Fatal("the handed-off Manager could not answer", err)
	}
}

// The no-progress stop applies only to a Manager with an open assignment: a
// handed-off Manager's inaction never stops the Run, while an open assignment
// that stands still still does.
func TestNoProgressStopsOnlyAnOpenAssignment(t *testing.T) {
	x, _, _, deep := inboxExecution()
	defer x.cancel()
	deep.copy = toolFailureCopy{}
	state := Commit{ID: "fixed", Tree: "tree"}
	accepted(deep, "deep")
	for range 2 * maxIdle {
		if err := x.acted(deep, true, state); err != nil {
			t.Fatal("a handed-off Manager's inaction stopped the Run", err)
		}
	}
	deep.a.phase, deep.a.task = active, Assignment{Task: "realize"}
	var err error
	for range maxIdle {
		err = x.acted(deep, true, state)
	}
	if !errors.Is(err, ErrNoProgress) || !strings.Contains(err.Error(), `on its assignment "realize"`) {
		t.Fatal("an open assignment that stood still was not stopped", err)
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

func (r toolFailureRecords) Notes(context.Context, string, string) (NotePage, error) {
	return NotePage{}, r.failure
}
func (r toolFailureRecords) Append(context.Context, Note) (Note, error) { return Note{}, r.failure }

func TestIgnoredToolPortErrorsStillStopExecution(t *testing.T) {
	for _, stage := range []string{"delegate-snapshot", "update", "diff", "review-snapshot", "integrate", "judge-snapshot", "check-snapshot"} {
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

// Notes are optional information: their storage failures reach the Manager as
// tool errors and leave the Run running.
func TestNotesStorageFailuresAreToolErrorsNotRunFailures(t *testing.T) {
	x, _, parent, _ := inboxExecution()
	defer x.cancel()
	injected := errors.New("actual Notes storage failure")
	x.executor = &Executor{Records: toolFailureRecords{failure: injected}}
	tools := x.tools(parent)
	if _, err := tools.ReadNotes(context.Background(), ""); !errors.Is(err, injected) {
		t.Fatal("read failure not reported to the Manager", err)
	}
	if _, err := tools.AppendNote(context.Background(), "scope", "durable note"); !errors.Is(err, injected) {
		t.Fatal("append failure not reported to the Manager", err)
	}
	if x.ctx.Err() != nil || x.asyncErr != nil {
		t.Fatal("optional Notes failure stopped the Run", x.ctx.Err(), x.asyncErr)
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

func TestExchangeEnforcesIdleLimitWithinOriginalRunDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	x := &execution{ctx: ctx, request: Request{ProviderIdleTimeout: 20 * time.Millisecond}}
	start := time.Now()
	err := x.exchange(&managerState{session: timedMessageSession{}}, Message{Feedback: []Finding{{Text: "bounded"}}})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second || ctx.Err() != nil {
		t.Fatal("message ignored its own limit or consumed whole-Run deadline", err, ctx.Err())
	}
	x.calls.Wait()
}

type scriptedSession func(context.Context) error

func (f scriptedSession) Exchange(ctx context.Context, _ Message) error { return f(ctx) }
func (scriptedSession) Close(context.Context) error                     { return nil }

// ProviderIdleTimeout bounds provider work, not the host serving tool calls: a
// parent may await its working children beyond it, provider work after such a
// call stays bounded, and the Run deadline bounds the waiting.
func TestProviderIdleTimeoutCountsProviderWorkNotServedToolCalls(t *testing.T) {
	const limit = 80 * time.Millisecond
	setup := func(run time.Duration) (*execution, *managerState) {
		x, _, a, deep := inboxExecution()
		x.cancel()
		x.ctx, x.cancel = context.WithTimeout(context.Background(), run)
		x.request.ProviderIdleTimeout = limit
		// a awaits its working child inside its own response.
		deep.a.phase, a.running, deep.running = active, true, true
		return x, a
	}
	question := func(x *execution, a *managerState, after time.Duration) {
		time.Sleep(after)
		x.mu.Lock()
		defer x.mu.Unlock()
		x.questions["q"] = &question{Question: Question{ID: "q", From: "a/deep", To: "a", Text: "child question"}}
		x.queueLocked(a, entry{event: Event{Question: &x.questions["q"].Question}})
	}
	t.Run("await outlasts the limit", func(t *testing.T) {
		x, a := setup(time.Minute)
		defer x.cancel()
		tools := x.tools(a)
		a.session = scriptedSession(func(ctx context.Context) error {
			e, err := tools.Await(ctx)
			if err == nil && e.Question == nil {
				err = fmt.Errorf("await returned %+v", e)
			}
			return err
		})
		start := time.Now()
		go question(x, a, 4*limit)
		if err := x.exchange(a, Message{Feedback: []Finding{{Text: "work"}}}); err != nil || time.Since(start) < 4*limit {
			t.Fatal("a served await counted as provider work", err, time.Since(start))
		}
		x.calls.Wait()
	})
	t.Run("provider stall after a served call", func(t *testing.T) {
		x, a := setup(time.Minute)
		defer x.cancel()
		tools := x.tools(a)
		a.session = scriptedSession(func(ctx context.Context) error {
			if _, err := tools.Await(ctx); err != nil {
				return err
			}
			<-ctx.Done()
			return nil
		})
		start := time.Now()
		go question(x, a, 3*limit)
		err := x.exchange(a, Message{Feedback: []Finding{{Text: "work"}}})
		// The question arrives after 3*limit, so only an uncounted await gets
		// there; the stall after it then gets the whole limit.
		if elapsed := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || elapsed < 4*limit || elapsed > 4*limit+time.Second || x.ctx.Err() != nil {
			t.Fatal("a stall after a served call escaped or misused the limit", err, elapsed)
		}
		x.calls.Wait()
	})
	t.Run("a call ending during a parallel served call", func(t *testing.T) {
		x, a := setup(time.Minute)
		defer x.cancel()
		x.executor = &Executor{Records: toolFailureRecords{}}
		tools := x.tools(a)
		a.session = scriptedSession(func(ctx context.Context) error {
			awaited := make(chan error, 1)
			go func() { _, err := tools.Await(ctx); awaited <- err }()
			// read_notes starts and ends while await is still served.
			clock := ctx.Value(clockKey{}).(*idleClock)
			for serving := 0; serving == 0; {
				clock.mu.Lock()
				serving = clock.serving
				clock.mu.Unlock()
			}
			if _, err := tools.ReadNotes(ctx, ""); err != nil {
				return err
			}
			return <-awaited
		})
		start := time.Now()
		go question(x, a, 4*limit)
		if err := x.exchange(a, Message{Feedback: []Finding{{Text: "work"}}}); err != nil || time.Since(start) < 4*limit {
			t.Fatal("the clock ran while a parallel call was still served", err, time.Since(start))
		}
		x.calls.Wait()
	})
	t.Run("the Run deadline bounds waiting", func(t *testing.T) {
		x, a := setup(4 * limit)
		defer x.cancel()
		tools := x.tools(a)
		a.session = scriptedSession(func(ctx context.Context) error {
			_, err := tools.Await(ctx)
			return err
		})
		start := time.Now()
		err := x.exchange(a, Message{Feedback: []Finding{{Text: "work"}}})
		if elapsed := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || elapsed < 3*limit || x.ctx.Err() == nil {
			t.Fatal("host waiting outlived the Run deadline or ended at the message limit", err, elapsed)
		}
		x.calls.Wait()
	})
}

// The idle limit restarts at every served tool call (phase C chain 3: Root hit
// a whole-response limit six times while it kept delegating, awaiting and
// reviewing). Active stretches longer than the limit together never stop a
// response, a stall after them ends it after the limit, and the Run deadline
// still ends a response that never stalls. No call is judged useful or not.
func TestProviderIdleTimeoutRestartsAtEveryServedToolCall(t *testing.T) {
	const limit = 200 * time.Millisecond
	setup := func(run time.Duration) (*execution, *managerState, Tools) {
		x, _, a, _ := inboxExecution()
		x.cancel()
		x.ctx, x.cancel = context.WithTimeout(context.Background(), run)
		x.request.ProviderIdleTimeout = limit
		x.executor = &Executor{Records: toolFailureRecords{}}
		return x, a, x.tools(a)
	}
	// One active stretch: provider work for a quarter of the limit, then a
	// served read_notes call.
	stretch := func(ctx context.Context, tools Tools) error {
		time.Sleep(limit / 4)
		_, err := tools.ReadNotes(ctx, "")
		return err
	}
	work := Message{Feedback: []Finding{{Text: "work"}}}
	t.Run("active stretches exceed the limit together", func(t *testing.T) {
		x, a, tools := setup(time.Minute)
		defer x.cancel()
		a.session = scriptedSession(func(ctx context.Context) error {
			for range 8 {
				if err := stretch(ctx, tools); err != nil {
					return err
				}
			}
			return nil
		})
		start := time.Now()
		if err := x.exchange(a, work); err != nil || time.Since(start) < 2*limit {
			t.Fatal("served tool calls did not restart the idle limit", err, time.Since(start))
		}
		x.calls.Wait()
	})
	t.Run("a stall after active stretches", func(t *testing.T) {
		x, a, tools := setup(time.Minute)
		defer x.cancel()
		a.session = scriptedSession(func(ctx context.Context) error {
			for range 8 {
				if err := stretch(ctx, tools); err != nil {
					return err
				}
			}
			<-ctx.Done()
			return nil
		})
		start := time.Now()
		err := x.exchange(a, work)
		if elapsed := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || elapsed < 3*limit || elapsed > 3*limit+time.Second || x.ctx.Err() != nil {
			t.Fatal("a stall after active stretches escaped or misused the limit", err, elapsed)
		}
		x.calls.Wait()
	})
	t.Run("the Run deadline ends a response that never stalls", func(t *testing.T) {
		x, a, tools := setup(3 * limit)
		defer x.cancel()
		a.session = scriptedSession(func(ctx context.Context) error {
			for ctx.Err() == nil {
				if err := stretch(ctx, tools); err != nil {
					return err
				}
			}
			return ctx.Err()
		})
		start := time.Now()
		err := x.exchange(a, work)
		if elapsed := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || elapsed < 2*limit || x.ctx.Err() == nil {
			t.Fatal("an active response outlived the Run deadline or ended at the idle limit", err, elapsed)
		}
		x.calls.Wait()
	})
}

// Parallel tool calls: the clock stays paused while any call is served, and
// only the end of the last one restarts it at its full limit.
func TestIdleClockPausesWhileAnyParallelCallIsServed(t *testing.T) {
	const limit = 100 * time.Millisecond
	expired := make(chan struct{})
	clock := startClock(limit, func() { close(expired) })
	defer clock.stop()
	first, second := clock.serve(), clock.serve()
	first()
	select {
	case <-expired:
		t.Fatal("the clock ran while a parallel call was still served")
	case <-time.After(2 * limit):
	}
	start := time.Now()
	second()
	select {
	case <-expired:
		if elapsed := time.Since(start); elapsed < limit {
			t.Fatal("the clock restarted below its full limit", elapsed)
		}
	case <-time.After(limit + time.Second):
		t.Fatal("the clock did not restart after the last call")
	}
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
	if a.a.judgment != nil || len(a.a.feedback) != 1 || a.a.feedback[0].Text != "you delegated to a/deep after your judgment; await its new handoff, review it and judge again" {
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

// The last other driver releasing wakes a blocked await, which then refuses
// instead of waiting for an answer only its caller can give.
func TestReleasedDriverWakesBlockedAwait(t *testing.T) {
	x, _, a, deep := inboxExecution()
	defer x.cancel()
	// a awaits inside its own response, so its driver runs too.
	deep.a.phase, deep.running, a.running = active, true, true
	x.questions["q"] = &question{Question: Question{ID: "q", From: "a/deep", To: "a", Text: "open"}, delivered: true}
	returned := make(chan error, 1)
	go func() { _, err := x.await(context.Background(), a); returned <- err }()
	select {
	case err := <-returned:
		t.Fatal("await returned while another driver ran", err)
	case <-time.After(50 * time.Millisecond):
	}
	x.mu.Lock()
	deep.running = false
	x.settleLocked()
	x.mu.Unlock()
	select {
	case err := <-returned:
		if err == nil || !strings.Contains(err.Error(), "answer your open questions: q from a/deep") {
			t.Fatal("await did not name the open question", err)
		}
	case <-time.After(time.Second):
		t.Fatal("released driver left await blocked")
	}
}

type awaitResult struct {
	event Event
	err   error
}

// awaitAsync runs s's await in its own goroutine.
func awaitAsync(ctx context.Context, x *execution, s *managerState) <-chan awaitResult {
	out := make(chan awaitResult, 1)
	go func() {
		event, err := x.await(ctx, s)
		out <- awaitResult{event, err}
	}()
	return out
}

// blockedInAwait returns once s blocks in the await that reports to returned;
// an await that returns instead, or never blocks, fails the test.
func blockedInAwait(t *testing.T, x *execution, s *managerState, returned <-chan awaitResult) {
	t.Helper()
	bound := time.After(5 * time.Second)
	for {
		x.mu.Lock()
		blocked := s.awaiting > 0
		x.mu.Unlock()
		if blocked {
			return
		}
		select {
		case r := <-returned:
			t.Fatal(s.manager.Package, "returned from await instead of blocking", r.event, r.err)
		case <-bound:
			t.Fatal(s.manager.Package, "never blocked in await")
		case <-time.After(time.Millisecond):
		}
	}
}

// Native control run: a child asked Root, Root awaited the child's handoff
// without replying and the child awaited the answer. Each counted the other as
// running, so both blocked until the Run deadline 76 minutes later. A Manager
// blocked in await waits: whichever of them awaits last, the parent is refused
// with the question it owes, and its reply reaches the waiting child.
func TestMutualAwaitRefusesTheManagerThatOwesTheAnswer(t *testing.T) {
	for _, parentFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("parent awaits first %v", parentFirst), func(t *testing.T) {
			x, root, a, _ := inboxExecution()
			defer x.cancel()
			// Both work inside their responses; a/deep has no task and stays idle.
			root.running, a.running = true, true
			root.a.phase, root.a.version = active, 1
			a.a.phase, a.a.version = active, 1
			// A deadlock ends every await here at this bound instead of hanging.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			id, err := x.tools(a).Ask(ctx, ".", "Which interface do you expect?")
			if err != nil {
				t.Fatal(err)
			}
			if e, err := x.await(ctx, root); err != nil || e.Question == nil || e.Question.ID != id {
				t.Fatal("the question did not reach the parent", e, err)
			}
			var child <-chan awaitResult
			var refusal error
			if parentFirst {
				parent := awaitAsync(ctx, x, root)
				blockedInAwait(t, x, root, parent)
				child = awaitAsync(ctx, x, a)
				refusal = (<-parent).err
			} else {
				child = awaitAsync(ctx, x, a)
				blockedInAwait(t, x, a, child)
				_, refusal = x.await(ctx, root)
			}
			want := "nothing can arrive: every other Manager is waiting; answer your open questions: " + id + " from a"
			if refusal == nil || refusal.Error() != want {
				t.Fatal("the parent was not refused with the question it owes", refusal)
			}
			if err := x.tools(root).Reply(ctx, id, "The one the model names."); err != nil {
				t.Fatal(err)
			}
			if r := <-child; r.err != nil || r.event.Answer == nil || r.event.Answer.ID != id {
				t.Fatal("the waiting child did not receive the answer", r.event, r.err)
			}
			x.mu.Lock()
			defer x.mu.Unlock()
			if root.awaiting != 0 || a.awaiting != 0 {
				t.Fatal("a returned await still counts as blocked", root.awaiting, a.awaiting)
			}
		})
	}
}

// A Manager blocked in await waits, but it does not make another Manager's
// await refuse while a third Manager really runs: that one can still send.
func TestBlockedAwaitDoesNotRefuseWhileAnotherManagerRuns(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	root.running, a.running, deep.running = true, true, true
	root.a.phase, a.a.phase, deep.a.phase = active, active, active
	rootCtx, stopRoot := context.WithCancel(context.Background())
	defer stopRoot()
	parent := awaitAsync(rootCtx, x, root)
	blockedInAwait(t, x, root, parent)
	// a blocks too and wakes Root, which checks again; a/deep still works.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := x.await(ctx, a); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("await refused although a/deep is running", err)
	}
	select {
	case r := <-parent:
		t.Fatal("Root's await returned although a/deep is running", r.event, r.err)
	default:
	}
	stopRoot()
	if r := <-parent; !errors.Is(r.err, context.Canceled) {
		t.Fatal("Root's await refused although a/deep is running", r.err)
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if root.awaiting != 0 || a.awaiting != 0 {
		t.Fatal("a returned await still counts as blocked", root.awaiting, a.awaiting)
	}
}

// A refused await names the next step from its children's state. A child's
// late finding changes that state at the same time, so the refusal reads it
// under the Run lock; the race detector reports a read after unlocking.
func TestAwaitRefusalReadsChildStateUnderTheRunLock(t *testing.T) {
	x, _, a, deep := inboxExecution()
	defer x.cancel()
	a.a.phase, a.a.version = active, 1
	deep.a.fix(Handoff{Manager: "a/deep", Result: Commit{ID: "c", Tree: "t"}, Decision: Submitted})
	changed := make(chan struct{})
	go func() {
		defer close(changed)
		for range 200 {
			x.mu.Lock()
			deep.a.lateFinding(Reject, []Finding{{Manager: "a/deep", Text: "late required correction"}})
			x.mu.Unlock()
		}
	}()
	for range 200 {
		if _, err := x.await(context.Background(), a); err == nil || !strings.Contains(err.Error(), "the review of child a/deep's fixed handoff c is missing") {
			t.Fatal("await did not refuse with the child's fixed handoff", err)
		}
	}
	<-changed
}

// §3.7: a refused call names the concrete reason and the possible next step.
func TestRefusalsNameReasonAndNextStep(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	root.running, a.running, deep.running = true, true, true
	tools := x.tools(a)
	a.a.phase, a.a.version = active, 1
	for name, tc := range map[string]struct {
		call func() error
		want string
	}{
		"judge with an unassigned child": {func() error { return tools.Judge(context.Background(), Submitted, nil) }, "child a/deep has no assignment yet; delegate its task"},
		"judge reject without Findings":  {func() error { return tools.Judge(context.Background(), Reject, nil) }, "judge again with at least one Finding text"},
		"review an unassigned child":     {func() error { return tools.Review(context.Background(), "a/deep", "c", Approve, nil) }, "child a/deep has no assignment yet; delegate its task first"},
		"review as a childless Manager":  {func() error { return x.tools(deep).Review(context.Background(), "x", "c", Approve, nil) }, "review covers only direct children, and you have none"},
		"delegate without a task":        {func() error { return tools.Delegate(context.Background(), "a/deep", " ", nil) }, "delegate again with a task text"},
		"reply to an unknown question":   {func() error { return tools.Reply(context.Background(), "missing", "answer") }, "reply only to question IDs delivered to you"},
		"await with nothing to arrive":   {func() error { _, err := tools.Await(context.Background()); return err }, "none of your questions is open; child a/deep has no assignment yet; delegate its task"},
		"judge without an assignment":    {func() error { return x.tools(deep).Judge(context.Background(), Submitted, nil) }, "a reply to a question needs no judgment"},
		"delegate without an assignment": {func() error { return x.tools(deep).Delegate(context.Background(), "x", "task", nil) }, "delegate reaches only direct children, and you have none"},
		"act from a fixed handoff": {func() error {
			accepted(root, "root")
			return x.tools(root).Delegate(context.Background(), "a", "task", nil)
		}, "reply to questions addressed to you"},
	} {
		if err := tc.call(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatal(name, err)
		}
	}
}

// Next steps depend on the state they are given in: each one names an action
// the tools then accept.
func TestRefusalNextStepsFollowTheState(t *testing.T) {
	x, root, a, deep := inboxExecution()
	defer x.cancel()
	root.running, a.running, deep.running = true, true, true
	a.a.phase, a.a.version = active, 1
	// A handoff fixed but not yet received is awaited, not reviewed.
	deep.a.fix(Handoff{Manager: "a/deep", Result: Commit{ID: "c", Tree: "t"}, Decision: Submitted})
	if err := x.tools(a).Review(context.Background(), "a/deep", "c", Approve, nil); err == nil || !strings.Contains(err.Error(), "has not reached you yet; await it, then review it") {
		t.Fatal(err)
	}
	// A model blocker is judged, not corrected.
	accepted(deep, "deep")
	deep.a.handoff.Decision = ModelBlocker
	if err := x.tools(a).Delegate(context.Background(), "a/deep", "correct", []Finding{{Text: "f"}}); err == nil || !strings.Contains(err.Error(), "judge your assignment, after reviewing its handoff if you have not") {
		t.Fatal(err)
	}
	// A queued correction is awaited.
	deep.a.phase = assigned
	if err := x.tools(deep).Judge(context.Background(), Submitted, nil); err == nil || err.Error() != "your new assignment is waiting; await it" {
		t.Fatal(err)
	}
	// Verify cannot repair its fixed input to pass the check.
	x.request.Mandate = Verify
	x.request.Input.Commit, x.request.Input.Tree = "input", "input-tree"
	accepted(a, "a")
	root.a.phase, root.a.version = active, 1
	root.checked = &Check{Commit: Commit{ID: "input", Tree: "input-tree"}, Diagnostic: "file.unexplained"}
	if err := x.tools(root).Judge(context.Background(), Approve, nil); err == nil || err.Error() != "the fixed input candidate input fails its check: file.unexplained; judge reject or model_blocker with Findings" {
		t.Fatal(err)
	}
}
