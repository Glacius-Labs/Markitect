package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type concurrentAgents struct {
	mu                sync.Mutex
	team              []run.Manager
	opened            map[string]*concurrentSession
	openEntered       chan string
	workEntered       chan string
	reviewEntered     chan string
	questionEntered   chan string
	questionCalled    chan struct{}
	questionAnswered  chan struct{}
	cancelled         chan string
	stopped           chan string
	openGates         map[string]<-chan struct{}
	workGates         map[string]<-chan struct{}
	reviewGates       map[string]<-chan struct{}
	askNow            <-chan struct{}
	failOpen          string
	failWork          string
	lateOpen          string
	consultBusy       bool
	consultRefused    atomic.Bool
	returnChild       string
	returned          atomic.Bool
	correctionEntered chan string
	correctionGate    <-chan struct{}
	parentBusy        atomic.Int32
	overlapped        atomic.Bool
}

func concurrentFixture() ([]run.Manager, *concurrentAgents) {
	team := []run.Manager{
		{Path: ".", Children: []string{"library"}, ExecuteProfile: "exec"},
		{Path: "library", Parent: ".", Children: []string{"library/first", "library/second"}, ExecuteProfile: "exec"},
		{Path: "library/first", Parent: "library", ExecuteProfile: "exec"},
		{Path: "library/second", Parent: "library", ExecuteProfile: "exec"},
	}
	a := &concurrentAgents{team: team, opened: map[string]*concurrentSession{}, openEntered: make(chan string, 16), workEntered: make(chan string, 16), reviewEntered: make(chan string, 16), questionEntered: make(chan string, 16), questionCalled: make(chan struct{}, 1), questionAnswered: make(chan struct{}, 1), cancelled: make(chan string, 16), stopped: make(chan string, 16), correctionEntered: make(chan string, 8), openGates: map[string]<-chan struct{}{}, workGates: map[string]<-chan struct{}{}, reviewGates: map[string]<-chan struct{}{}}
	return team, a
}

type concurrentSession struct {
	owner   *concurrentAgents
	spec    run.AssignmentSpec
	manager run.Manager
	handler run.ToolHandler
}

func (a *concurrentAgents) enterParent() func() {
	if a.parentBusy.Add(1) != 1 {
		a.overlapped.Store(true)
	}
	return func() { a.parentBusy.Add(-1) }
}

func (a *concurrentAgents) OpenAssignment(ctx context.Context, spec run.AssignmentSpec, handler run.ToolHandler) (run.AssignmentSession, error) {
	var index int
	if _, err := fmt.Sscanf(spec.ID, "fan-%d", &index); err != nil || index >= len(a.team) {
		return nil, errors.New("unknown Agent")
	}
	path := a.team[index].Path
	a.openEntered <- path
	if gate := a.openGates[path]; gate != nil {
		if path == a.lateOpen {
			<-gate
		} else if err := waitGate(ctx, gate); err != nil {
			return nil, err
		}
	}
	if path == a.failOpen {
		return nil, errors.New("controlled child startup failure")
	}
	s := &concurrentSession{a, spec, a.team[index], handler}
	a.mu.Lock()
	a.opened[path] = s
	a.mu.Unlock()
	return s, nil
}

func waitGate(ctx context.Context, gate <-chan struct{}) error {
	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *concurrentSession) Message(ctx context.Context, prompt string, _ time.Duration) (run.TurnResult, error) {
	a, path := s.owner, s.manager.Path
	if path == "library" {
		defer a.enterParent()()
	}
	call := func(name string, args map[string]string) error {
		data, _ := json.Marshal(args)
		if answer := s.handler(ctx, name, data); answer.Refused {
			return errors.New(answer.Text)
		}
		return nil
	}
	result := run.TurnResult{Completed: true, Final: "Result of " + path, Commands: []run.CommandObservation{{Command: "test " + path, Status: "passed"}}}
	switch {
	case strings.HasPrefix(prompt, "Your responsibility"):
		a.workEntered <- path
		for _, child := range s.manager.Children {
			if err := call(run.DelegateChildTool, map[string]string{"child": child, "task": "Use the interface established in this parent Go"}); err != nil {
				return result, err
			}
		}
		if path == "library/first" && a.askNow != nil {
			if err := waitGate(ctx, a.askNow); err != nil {
				return result, err
			}
			a.questionCalled <- struct{}{}
			if err := call(run.AskParentTool, map[string]string{"question": "Confirm the parent interface"}); err != nil {
				a.cancelled <- path
				return result, err
			}
			a.questionAnswered <- struct{}{}
		}
		if gate := a.workGates[path]; gate != nil {
			if err := waitGate(ctx, gate); err != nil {
				a.cancelled <- path
				return result, err
			}
		}
		if path == a.failWork {
			return result, errors.New("controlled child work failure")
		}
	case strings.HasPrefix(prompt, "Review your direct child"):
		parts := strings.Fields(prompt)
		child := strings.TrimSuffix(parts[4], "'s")
		a.reviewEntered <- child
		if gate := a.reviewGates[child]; gate != nil {
			if err := waitGate(ctx, gate); err != nil {
				return result, err
			}
		}
		if a.consultBusy && child == "library/second" {
			args, _ := json.Marshal(map[string]string{"child": "library/first", "question": "Confirm the cross-area contract"})
			answer := s.handler(ctx, run.AskChildTool, args)
			if !answer.Refused || !strings.Contains(answer.Text, "working or waiting") {
				return result, fmt.Errorf("busy child consultation did not refuse safely: %+v", answer)
			}
			a.consultRefused.Store(true)
		}
		if child == a.returnChild && a.returned.CompareAndSwap(false, true) {
			return result, call(run.ReturnChildTool, map[string]string{"child": child, "correction": "Repair " + child})
		}
		commit := reviewedResult.FindStringSubmatch(prompt)
		if len(commit) != 2 {
			return result, errors.New("missing fixed child result")
		}
		if err := call(run.AcceptChildTool, map[string]string{"child": child, "commit": commit[1]}); err != nil {
			return result, err
		}
	case strings.Contains(prompt, " asks:"):
		a.questionEntered <- path
		result.Final = "Use the established interface"
	case strings.HasPrefix(prompt, "Repair "):
		a.correctionEntered <- path
		if err := waitGate(ctx, a.correctionGate); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *concurrentSession) Close(context.Context) (run.TurnResult, error) {
	s.owner.stopped <- s.manager.Path
	return run.TurnResult{Stopped: true}, nil
}

type concurrentIntegration struct {
	agents  *concurrentAgents
	entered chan string
	gates   map[string]<-chan struct{}
	mu      sync.Mutex
	order   []string
}

func (g *concurrentIntegration) Changes(_ context.Context, _ run.AssignmentWorkspace, h run.AssignmentHandoff) (string, error) {
	return "actual diff " + h.ResultCommit, nil
}
func (g *concurrentIntegration) Integrate(ctx context.Context, parent, child run.AssignmentWorkspace, _ run.AssignmentHandoff, _ string) (string, error) {
	if parent.ID() == "fan-1" {
		defer g.agents.enterParent()()
	}
	path := "library"
	if child.ID() == "fan-2" {
		path = "library/first"
	}
	if child.ID() == "fan-3" {
		path = "library/second"
	}
	g.entered <- path
	if gate := g.gates[path]; gate != nil {
		if err := waitGate(ctx, gate); err != nil {
			return "", err
		}
	}
	g.mu.Lock()
	g.order = append(g.order, path)
	g.mu.Unlock()
	return strings.Repeat("f", 40), nil
}

type concurrentOutcome struct {
	result run.AssignmentHandoff
	flow   *run.AssignmentHandoffs
	err    error
}

func startConcurrentTree(t *testing.T, ctx context.Context, a *concurrentAgents, g *concurrentIntegration, timeout time.Duration) <-chan concurrentOutcome {
	t.Helper()
	options := run.AssignmentSchedule{IDPrefix: "fan", Task: "Establish interfaces here, then Go to every direct child", MaxMessages: 40, Timeout: timeout, TranscriptDir: t.TempDir()}
	done := make(chan concurrentOutcome, 1)
	go func() {
		result, flow, err := run.RunAssignmentTree(ctx, a.team, &assignmentWorkspacesFake{}, a, g, options)
		done <- concurrentOutcome{result, flow, err}
	}()
	return done
}

func receivePath(t *testing.T, ch <-chan string, wanted string) {
	t.Helper()
	select {
	case got := <-ch:
		if got != wanted {
			t.Fatalf("got %q, want %q", got, wanted)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", wanted)
	}
}

func receivePair(t *testing.T, ch <-chan string, first, second string) {
	t.Helper()
	got := map[string]bool{}
	for len(got) < 2 {
		select {
		case path := <-ch:
			if path == first || path == second {
				got[path] = true
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("starts did not overlap: %v", got)
		}
	}
}

func TestAssignmentChildrenOverlapAndReviewInCompletionOrderWithQueuedQuestions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, a := concurrentFixture()
	open, first, second, review, integrate, ask := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	a.openGates["library/first"], a.openGates["library/second"] = open, open
	a.workGates["library/first"], a.workGates["library/second"] = first, second
	a.reviewGates["library/second"], a.askNow = review, ask
	g := &concurrentIntegration{agents: a, entered: make(chan string, 8), gates: map[string]<-chan struct{}{"library/second": integrate}}
	done := startConcurrentTree(t, ctx, a, g, time.Second)
	receivePair(t, a.openEntered, "library/first", "library/second")
	close(open)
	receivePair(t, a.workEntered, "library/first", "library/second")
	close(second)
	receivePath(t, a.reviewEntered, "library/second") // first is still working
	close(ask)
	select {
	case <-a.questionCalled:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case <-a.questionEntered:
		t.Fatal("question interrupted parent review")
	case <-time.After(20 * time.Millisecond):
	}
	close(review)
	receivePath(t, g.entered, "library/second")
	select {
	case <-a.questionEntered:
		t.Fatal("question split review from integration")
	case <-time.After(20 * time.Millisecond):
	}
	close(integrate)
	receivePath(t, a.questionEntered, "library")
	select {
	case <-a.questionAnswered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	close(first)
	receivePath(t, a.reviewEntered, "library/first")
	outcome := <-done
	if outcome.err != nil || outcome.result.ResultCommit == "" {
		t.Fatalf("tree: %+v", outcome)
	}
	if a.overlapped.Load() {
		t.Fatal("parent conversation and integration overlapped")
	}
	if strings.Join(g.order, ",") != "library/second,library/first,library" {
		t.Fatalf("completion order lost: %v", g.order)
	}
	if len(a.opened) != 4 {
		t.Fatalf("parent conversation/container replaced: %d starts", len(a.opened))
	}
}

func TestAssignmentChildrenCrossedQuestionAndConsultationDoNotDeadlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, a := concurrentFixture()
	first, review, ask := make(chan struct{}), make(chan struct{}), make(chan struct{})
	a.workGates["library/first"], a.reviewGates["library/second"], a.askNow, a.consultBusy = first, review, ask, true
	g := &concurrentIntegration{agents: a, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}
	done := startConcurrentTree(t, ctx, a, g, time.Second)
	receivePath(t, a.reviewEntered, "library/second")
	close(ask)
	select {
	case <-a.questionCalled:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	close(review) // Parent consults First, which is waiting for this same parent.
	receivePath(t, a.questionEntered, "library")
	select {
	case <-a.questionAnswered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	close(first)
	outcome := <-done
	if outcome.err != nil || !a.consultRefused.Load() || a.overlapped.Load() {
		t.Fatalf("crossed questions: %+v", outcome)
	}
}

func TestAssignmentChildrenReturnedCorrectionRejoinsAfterReadySibling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, a := concurrentFixture()
	first, second, corrected := make(chan struct{}), make(chan struct{}), make(chan struct{})
	a.workGates["library/first"], a.workGates["library/second"] = first, second
	a.returnChild, a.correctionGate = "library/second", corrected
	g := &concurrentIntegration{agents: a, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}
	done := startConcurrentTree(t, ctx, a, g, time.Second)
	receivePair(t, a.workEntered, "library/first", "library/second")
	close(second)
	receivePath(t, a.reviewEntered, "library/second")
	receivePath(t, a.correctionEntered, "library/second")
	close(first)
	receivePath(t, a.reviewEntered, "library/first")
	receivePath(t, g.entered, "library/first") // Correction has not finished.
	close(corrected)
	receivePath(t, a.reviewEntered, "library/second")
	outcome := <-done
	if outcome.err != nil || strings.Join(g.order, ",") != "library/first,library/second,library" {
		t.Fatalf("returned correction blocked or reordered sibling: %v %v", g.order, outcome.err)
	}
	results := 0
	for _, result := range outcome.flow.Results() {
		if result.Manager == "library/second" {
			results++
		}
	}
	if results != 2 || len(a.opened) != 4 {
		t.Fatalf("correction lost fixed results or replaced Agent: %d results, %d Agents", results, len(a.opened))
	}
}

func TestAssignmentChildrenRecursiveBranchesCreateAndRetainHomesConcurrently(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, a := concurrentFixture()
	a.team = []run.Manager{
		{Path: ".", Children: []string{"left", "right"}, ExecuteProfile: "exec"},
		{Path: "left", Parent: ".", Children: []string{"left/leaf"}, ExecuteProfile: "exec"},
		{Path: "right", Parent: ".", Children: []string{"right/leaf"}, ExecuteProfile: "exec"},
		{Path: "left/leaf", Parent: "left", ExecuteProfile: "exec"},
		{Path: "right/leaf", Parent: "right", ExecuteProfile: "exec"},
	}
	entered, release := make(chan string, 2), make(chan struct{})
	g := &concurrentIntegration{agents: a, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}
	tree, err := run.OpenAssignmentTree(ctx, a.team, &assignmentWorkspacesFake{}, a, g, run.AssignmentSchedule{IDPrefix: "fan", Task: "Implement", MaxMessages: 40, Timeout: time.Second, TranscriptDir: t.TempDir(), CreateHome: func(ctx context.Context, home string) error {
		if home == "fan-3-home" || home == "fan-4-home" {
			entered <- home
			return waitGate(ctx, release)
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	done := make(chan error, 1)
	go func() { _, err := tree.Implement(ctx); done <- err }()
	receivePair(t, entered, "fan-3-home", "fan-4-home")
	if len(tree.Homes()) != 5 {
		t.Fatalf("concurrent home bookkeeping lost a branch: %v", tree.Homes())
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestAssignmentChildFailureCancelsWorkAndRetainsEveryCopy(t *testing.T) {
	for _, phase := range []string{"open", "work", "queued question", "late open"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, a := concurrentFixture()
			gate := make(chan struct{})
			secondWork := make(chan struct{})
			if phase == "late open" {
				a.openGates["library/first"], a.lateOpen = gate, "library/first"
			} else {
				a.workGates["library/first"] = gate
			}
			if phase == "open" || phase == "late open" {
				a.failOpen = "library/second"
			} else {
				a.workGates["library/second"] = secondWork
				if phase == "work" {
					a.failWork = "library/second"
				}
			}
			g := &concurrentIntegration{agents: a, entered: make(chan string, 8), gates: map[string]<-chan struct{}{}}
			var ask chan struct{}
			if phase == "queued question" {
				ask = make(chan struct{})
				a.askNow = ask
				delete(a.workGates, "library/second")
				a.reviewGates["library/second"] = gate
			}
			done := startConcurrentTree(t, ctx, a, g, 80*time.Millisecond)
			if phase == "work" {
				receivePair(t, a.workEntered, "library/first", "library/second")
				close(secondWork)
			}
			if phase == "queued question" {
				receivePath(t, a.reviewEntered, "library/second")
				close(ask)
				<-a.questionCalled
				cancel()
			}
			outcome := <-done
			if outcome.err == nil || outcome.result.ResultCommit != "" {
				t.Fatalf("failure accepted: %+v", outcome)
			}
			for _, path := range []string{"library/first", "library/second"} {
				if outcome.flow.Workspace(path) == nil {
					t.Fatalf("lost %s copy", path)
				}
			}
			if phase == "late open" {
				if !strings.Contains(outcome.err.Error(), "uncertain") {
					t.Fatalf("late startup was claimed settled: %v", outcome.err)
				}
				close(gate)
				receivePair(t, a.stopped, "library/first", ".")
			} else {
				a.mu.Lock()
				opened := len(a.opened)
				a.mu.Unlock()
				if len(a.stopped) != opened {
					t.Fatalf("stopped %d of %d opened Agents", len(a.stopped), opened)
				}
			}
		})
	}
}
