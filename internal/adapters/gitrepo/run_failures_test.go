package gitrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type failingRecords struct {
	run.Records
	failSave, mismatch bool
}

func (f failingRecords) Save(ctx context.Context, r run.Result) error {
	if f.failSave {
		return errors.New("injected storage failure")
	}
	return f.Records.Save(ctx, r)
}
func (f failingRecords) Read(ctx context.Context, p, id string) (run.Result, error) {
	r, err := f.Records.Read(ctx, p, id)
	if f.mismatch {
		r.Binding = "substituted"
	}
	return r, err
}
func (f failingRecords) Approvals(ctx context.Context, p, key string) ([]run.Result, error) {
	return f.Records.(*runstore.Store).Approvals(ctx, p, key)
}

type inspectFailure struct{ run.Workspaces }

func (f inspectFailure) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := f.Workspaces.Open(ctx, r)
	return failedService{s}, err
}

type failedService struct{ run.WorkspaceService }

func (f failedService) Copy(ctx context.Context, p string, c run.Commit) (run.Copy, error) {
	copy, err := f.WorkspaceService.Copy(ctx, p, c)
	if p == "." {
		return failedInspection{copy}, err
	}
	return copy, err
}

type failedInspection struct{ run.Copy }

func (f failedInspection) Inspect(context.Context, run.Commit) (project.PreviewPreparationAttempt, error) {
	return project.PreviewPreparationAttempt{}, errors.New("injected acquisition failure")
}

func TestRunFailuresPersistHonestTerminalStateAndCloseRealGit(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, kind := range []string{"partial startup", "cleanup failure", "stuck close", "stuck exchange", "storage failure", "readback substitution", "reject technical acquisition"} {
		t.Run(kind, func(t *testing.T) {
			store, _ := runstore.New(g.StateRoot)
			managers := &controlledManagers{crossed: make(chan struct{})}
			var records run.Records = store
			var git run.Workspaces = g
			release := make(chan struct{})
			var releaseOnce sync.Once
			releaseBlocked := func() { releaseOnce.Do(func() { close(release) }) }
			defer releaseBlocked()
			entered, drained := make(chan struct{}), make(chan struct{})
			deadline := runDeadline(t)
			switch kind {
			case "partial startup":
				managers.failOpen = true
			case "cleanup failure":
				managers.failClose = true
			case "stuck close":
				managers.closeBlock = release
			case "stuck exchange":
				managers.exchangeBlock = release
				managers.exchangeEntered, managers.exchangeDrained = entered, drained
			case "storage failure":
				records = failingRecords{Records: store, failSave: true}
			case "readback substitution":
				records = failingRecords{Records: store, mismatch: true}
			case "reject technical acquisition":
				managers.forceDecision = run.Reject
				git = inspectFailure{g}
			}
			id := "failure-" + fmt.Sprintf("%x", []byte(kind))
			e := run.Executor{Sessions: managers, Workspaces: git, Records: records}
			request := run.Request{MaxManagers: 5, ID: id, Project: "fixture", Prepared: p, Input: ref, Mandate: run.Assessment, Allowance: run.NewAllowance(deadline, 80), CleanupTimeout: time.Second, PersistenceTimeout: time.Second}
			var r run.Result
			var err error
			if kind == "stuck exchange" {
				// Real Git startup has the ordinary fixture allowance. Trigger
				// cancellation only after the intended unresponsive call exists.
				// Expiry before entry says nothing about Exchange drain.
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				type outcome struct {
					value run.Result
					err   error
				}
				returned := make(chan outcome, 1)
				go func() { value, err := e.Execute(ctx, request); returned <- outcome{value, err} }()
				select {
				case <-entered:
				case out := <-returned:
					t.Fatalf("Run ended before the blocked Exchange was entered: %s %v", out.value.Technical, out.err)
				}
				cancel()
				select {
				case out := <-returned:
					r, err = out.value, out.err
				case <-time.After(8 * time.Second):
					t.Fatal("Exchange ignored finite work/cleanup bounds after cancellation")
				}
				select {
				case <-drained:
					t.Fatal("blocked Exchange drained before fixture release")
				default:
				}
			} else {
				r, err = e.Execute(context.Background(), request)
			}
			if err == nil || r.Technical == run.Completed {
				t.Fatalf("technical failure became completed: %+v err=%v", r, err)
			}
			if kind == "reject technical acquisition" && (r.Technical != run.Failed || !strings.Contains(strings.Join(r.Errors, " "), "injected acquisition failure")) {
				t.Fatalf("semantic Reject swallowed technical acquisition: %+v", r)
			}
			if kind != "storage failure" && kind != "readback substitution" {
				fresh, _ := runstore.New(g.StateRoot)
				saved, readErr := fresh.Read(context.Background(), "fixture", id)
				if readErr != nil || saved.Technical == run.Completed || saved.Input != ref {
					t.Fatal("failed terminal record unavailable", saved, readErr)
				}
			}
			for _, s := range managers.sessions {
				if _, err := os.Stat(s.spec.Workspace.Location().Directory); !os.IsNotExist(err) {
					t.Fatalf("private Git copy survived failure %s: %v", kind, err)
				}
			}
			if kind == "stuck close" || kind == "stuck exchange" {
				found := false
				for _, c := range r.Cleanup {
					if !c.Closed {
						found = true
					}
				}
				if !found {
					t.Fatal("unresponsive Session falsely reported closed")
				}
			}
			if kind == "stuck exchange" {
				workersUnclosed := false
				for _, closure := range r.Cleanup {
					if closure.Resource == "manager-workers" && !closure.Closed {
						workersUnclosed = true
					}
				}
				if !workersUnclosed {
					t.Fatal("blocked Exchange was falsely reported drained")
				}
				before, readErr := store.Read(context.Background(), "fixture", id)
				if readErr != nil {
					t.Fatal(readErr)
				}
				releaseBlocked()
				waitRunSignal(t, drained, "released Exchange to drain")
				after, readErr := store.Read(context.Background(), "fixture", id)
				if readErr != nil || !reflect.DeepEqual(before, after) {
					t.Fatal("late Exchange completion changed the stored terminal record", readErr)
				}
			}
		})
	}
}

func TestRunPreflightRefusesInsufficientLimitsBeforeAnyEnvironment(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, kind := range []string{"managers", "messages"} {
		sessions := &controlledManagers{}
		store, _ := runstore.New(g.StateRoot)
		r := run.Request{MaxManagers: 5, ID: "preflight", Project: "fixture", Prepared: p, Input: ref, Mandate: run.Assessment, Allowance: run.NewAllowance(time.Now().Add(time.Minute), 80), CleanupTimeout: time.Second, PersistenceTimeout: time.Second}
		if kind == "managers" {
			r.MaxManagers = 4
		} else {
			r.Allowance = run.NewAllowance(time.Now().Add(time.Minute), 4)
		}
		e := run.Executor{Sessions: sessions, Workspaces: g, Records: store}
		_, err := e.Execute(context.Background(), r)
		if err == nil || len(sessions.sessions) != 0 {
			t.Fatal("insufficient limits started Manager environments")
		}
		entries, err := os.ReadDir(g.TempRoot)
		if err != nil || len(entries) != 0 {
			t.Fatal("preflight allocated Git resources", entries, err)
		}
	}
}

// This controlled check ignores cancellation, just as a bounded native cleanup
// can outlive Run's drain. The error-classification signal proves its returned
// receipt has reached the actual Run consumer before Save is allowed to finish.
type lateCheckError struct {
	once      sync.Once
	processed chan struct{}
}

func (*lateCheckError) Error() string { return "late controlled check completion" }
func (e *lateCheckError) Is(target error) bool {
	if target == project.ErrCandidateInvalid {
		e.once.Do(func() { close(e.processed) })
	}
	return false
}

type namedCheckWorkspaces struct {
	run.Workspaces
	mu    sync.Mutex
	names map[run.Copy]string
}

func (w *namedCheckWorkspaces) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := w.Workspaces.Open(ctx, r)
	return namedCheckService{WorkspaceService: s, owner: w}, err
}

type namedCheckService struct {
	run.WorkspaceService
	owner *namedCheckWorkspaces
}

func (s namedCheckService) Copy(ctx context.Context, name string, base run.Commit) (run.Copy, error) {
	c, err := s.WorkspaceService.Copy(ctx, name, base)
	if c != nil {
		s.owner.mu.Lock()
		s.owner.names[c] = name
		s.owner.mu.Unlock()
	}
	return c, err
}

type lateCandidateCheck struct {
	workspaces       *namedCheckWorkspaces
	entered, release chan struct{}
	failure          *lateCheckError
	once             sync.Once
}

func (c *lateCandidateCheck) Check(_ context.Context, _ []project.ExecutionCheck, copy run.Copy, commit run.Commit) ([]run.CheckObservation, error) {
	c.workspaces.mu.Lock()
	name := c.workspaces.names[copy]
	c.workspaces.mu.Unlock()
	if name != "a/deep" {
		return nil, nil
	}
	c.once.Do(func() { close(c.entered) })
	<-c.release
	return []run.CheckObservation{{Commit: commit.ID, ID: "late", Receipt: run.CheckReceipt{ID: "late", Output: []byte("late observation")}}}, c.failure
}

type heldSaveRecords struct {
	run.Records
	entered, release chan struct{}
}

func (r heldSaveRecords) Save(ctx context.Context, value run.Result) error {
	close(r.entered)
	select {
	case <-r.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return r.Records.Save(ctx, value)
}

// waitRunSignal waits for a point a Run reaches after real Git work; the
// test's own deadline bounds a Run that never reaches it.
func waitRunSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Until(runDeadline(t))):
		t.Fatal("timed out waiting for " + what)
	}
}

func TestLateChildCheckCannotChangeTerminalResultDuringSaveOrAfterReturn(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, phase := range []string{"during-save", "after-return"} {
		t.Run(phase, func(t *testing.T) {
			store, err := runstore.New(g.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			named := &namedCheckWorkspaces{Workspaces: g, names: map[run.Copy]string{}}
			check := &lateCandidateCheck{workspaces: named, entered: make(chan struct{}), release: make(chan struct{}), failure: &lateCheckError{processed: make(chan struct{})}}
			saved := heldSaveRecords{Records: store, entered: make(chan struct{}), release: make(chan struct{})}
			sessions := &controlledManagers{crossed: make(chan struct{})}
			executor := run.Executor{Workspaces: named, Sessions: sessions, Records: saved, Checks: check}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type completion struct {
				result run.Result
				err    error
			}
			done := make(chan completion, 1)
			go func() {
				r, err := executor.Execute(ctx, run.Request{MaxManagers: 5, ID: phase, Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(runDeadline(t), 100), CleanupTimeout: 800 * time.Millisecond, PersistenceTimeout: 5 * time.Second})
				done <- completion{r, err}
			}()
			waitRunSignal(t, check.entered, "child check")
			cancel()
			waitRunSignal(t, saved.entered, "Save after bounded worker drain")
			if phase == "during-save" {
				close(check.release)
				waitRunSignal(t, check.failure.processed, "late check classification during Save")
			}
			close(saved.release)
			var finished completion
			select {
			case finished = <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Run exceeded cleanup/persistence bounds")
			}
			if finished.err == nil || finished.result.Technical != run.Incomplete || len(finished.result.Checks) != 0 {
				t.Fatal("late receipt affected terminal result", finished.result, finished.err)
			}
			initial, err := json.Marshal(finished.result)
			if err != nil {
				t.Fatal(err)
			}
			if phase == "after-return" {
				close(check.release)
				waitRunSignal(t, check.failure.processed, "late check classification after return")
			}
			retained, err := store.Read(context.Background(), "fixture", phase)
			current, marshalErr := json.Marshal(finished.result)
			if err != nil || marshalErr != nil || !reflect.DeepEqual(retained, finished.result) || string(initial) != string(current) {
				t.Fatal("stored/caller terminal snapshot changed", retained, finished.result, err, marshalErr)
			}
			workersOpen := false
			for _, closure := range finished.result.Cleanup {
				if closure.Resource == "manager-workers" && !closure.Closed {
					workersOpen = true
				}
			}
			if !workersOpen {
				t.Fatal("blocked check was falsely reported drained")
			}
		})
	}
}

type workerErrorSessions struct {
	failure               error
	stage                 string
	awaiting, ignored     chan struct{}
	awaitOnce, ignoreOnce sync.Once
}

func (s *workerErrorSessions) Open(_ context.Context, spec run.SessionSpec) (run.Session, error) {
	return workerErrorSession{owner: s, spec: spec}, nil
}

type workerErrorSession struct {
	owner *workerErrorSessions
	spec  run.SessionSpec
}

func (workerErrorSession) Close(context.Context) error { return nil }
func (s workerErrorSession) Exchange(ctx context.Context, _ run.Message) error {
	tools := s.spec.Tools
	// Every refusal and failure below is deliberately ignored by this Session.
	for _, child := range s.spec.Manager.Children {
		if err := tools.Delegate(ctx, child, "Complete owned work.", nil); err != nil {
			_ = tools.Judge(ctx, run.Submitted, nil)
			return nil
		}
	}
	for range s.spec.Manager.Children {
		if s.spec.Manager.Package == "a" {
			s.owner.awaitOnce.Do(func() { close(s.owner.awaiting) })
		}
		event, err := tools.Await(ctx)
		if err != nil {
			s.owner.ignoreOnce.Do(func() { close(s.owner.ignored) })
			_ = tools.Judge(ctx, run.Submitted, nil)
			return nil
		}
		if event.Handoff == nil {
			return fmt.Errorf("child handoff missing")
		}
		if err = tools.Review(ctx, event.Handoff.Manager, event.Handoff.Result.ID, run.Approve, nil); err != nil {
			_ = tools.Judge(ctx, run.Submitted, nil)
			return nil
		}
	}
	if s.spec.Manager.Package == "a/deep" && s.owner.stage == "exchange" {
		<-s.owner.awaiting
		return s.owner.failure
	}
	_ = tools.Judge(ctx, run.Submitted, nil)
	return nil
}

type workerErrorWorkspaces struct {
	run.Workspaces
	owner *workerErrorSessions
}

func (w workerErrorWorkspaces) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	s, err := w.Workspaces.Open(ctx, r)
	return workerErrorService{WorkspaceService: s, owner: w.owner}, err
}

type workerErrorService struct {
	run.WorkspaceService
	owner *workerErrorSessions
}

func (s workerErrorService) Copy(ctx context.Context, name string, base run.Commit) (run.Copy, error) {
	c, err := s.WorkspaceService.Copy(ctx, name, base)
	if name == "a/deep" {
		return workerErrorInspection{Copy: c, owner: s.owner}, err
	}
	return c, err
}

type workerErrorInspection struct {
	run.Copy
	owner *workerErrorSessions
}

func (c workerErrorInspection) Inspect(context.Context, run.Commit) (project.PreviewPreparationAttempt, error) {
	<-c.owner.awaiting
	return project.PreviewPreparationAttempt{}, c.owner.failure
}

func TestParentCannotIgnoreInitialChildWorkerExchangeOrInspectionFailure(t *testing.T) {
	_, p, g, ref := runFixture(t)
	for _, stage := range []string{"exchange", "inspect"} {
		t.Run(stage, func(t *testing.T) {
			injected := errors.New("initial child " + stage + " failure")
			sessions := &workerErrorSessions{failure: injected, stage: stage, awaiting: make(chan struct{}), ignored: make(chan struct{})}
			var workspaces run.Workspaces = g
			if stage == "inspect" {
				workspaces = workerErrorWorkspaces{Workspaces: g, owner: sessions}
			}
			store, err := runstore.New(g.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			executor := run.Executor{Workspaces: workspaces, Sessions: sessions, Records: store}
			result, err := executor.Execute(context.Background(), run.Request{MaxManagers: 5, ID: "worker-" + stage, Project: "fixture", Prepared: p, Input: ref, Mandate: run.Implement, Allowance: run.NewAllowance(runDeadline(t), 50), CleanupTimeout: 2 * time.Second, PersistenceTimeout: 2 * time.Second})
			waitRunSignal(t, sessions.ignored, "parent ignoring Await failure")
			if !errors.Is(err, injected) || result.Technical == run.Completed || result.Messages >= 50 || !strings.Contains(strings.Join(result.Errors, " "), injected.Error()) {
				t.Fatal("child failure lost or exhausted budget", result, err)
			}
			retained, readErr := store.Read(context.Background(), "fixture", "worker-"+stage)
			if readErr != nil || !reflect.DeepEqual(retained, result) {
				t.Fatal("worker failure not durably returned", retained, result, readErr)
			}
		})
	}
}
