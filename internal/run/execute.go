package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"reflect"
	"strings"
	"sync"
	"time"
)

// rootTask is Root's assignment; the mandate and its rules come from the harness.
const rootTask = "Complete the bound mandate with your complete Manager hierarchy."

type managerState struct {
	manager organization.Manager
	copy    Copy
	session Session
	git     sync.Mutex // serializes Git operations that write this Manager's copy
	// Guarded by execution.mu.
	a            assignment
	running      bool // a driver owns this Session's messages
	progress     bool // an action succeeded during the current response
	stalled      int  // continued responses in a row without progress
	queue        []entry
	signal       chan struct{}
	integrations []Integration
}

// entry is one queued event. A child handoff is materialized on delivery, so
// the parent always receives the handoff's current generation.
type entry struct {
	event Event
	child string
	gen   int
	again bool // an unanswered question delivered once more
}

type question struct {
	Question
	delivered, answered bool
}

type execution struct {
	executor       *Executor
	request        Request
	ctx            context.Context
	cancel         context.CancelFunc
	workspace      WorkspaceService
	mu             sync.Mutex
	states         map[string]*managerState
	result         Result
	active         sync.WaitGroup // Manager drivers
	calls          sync.WaitGroup // Session exchanges
	questions      map[string]*question
	sequence       int
	closing        bool
	asyncErr       error
	checks         []CheckObservation
	receiptsClosed bool
	settled        chan struct{}
	rootCheck      *Check // Root's check of its last judged state
}

func (e *Executor) Execute(ctx context.Context, r Request) (Result, error) {
	x := &execution{executor: e, request: r, states: map[string]*managerState{}, questions: map[string]*question{}, settled: make(chan struct{}, 1), result: Result{ID: r.ID, Project: r.Project, Binding: r.Prepared.Binding(), Mandate: r.Mandate, Input: r.Input, Technical: Incomplete}}
	if err := validateRequest(r); err != nil {
		return x.result, err
	}
	if e == nil || e.Sessions == nil || e.Workspaces == nil || e.Records == nil {
		return x.result, fmt.Errorf("Run ports unavailable")
	}
	x.ctx, x.cancel = context.WithDeadline(ctx, r.Allowance.Deadline)
	defer x.cancel()
	for _, m := range r.Prepared.Topology().Managers() {
		x.states[m.Package] = &managerState{manager: m, signal: make(chan struct{}, 1)}
	}
	used := r.Allowance.Used()
	service, workErr := e.Workspaces.Open(x.ctx, r)
	x.workspace = service
	if workErr == nil && service == nil {
		workErr = fmt.Errorf("workspace startup returned no handle")
	}
	input := Commit{r.Input.Commit, r.Input.Tree}
	// Complete environment and communication tree exists before ordinary work.
	if workErr == nil {
		for _, m := range r.Prepared.Topology().Managers() {
			s := x.states[m.Package]
			s.copy, workErr = service.Copy(x.ctx, m.Package, input)
			if workErr != nil {
				break
			}
			if s.copy == nil || !s.copy.Location().Valid() {
				workErr = fmt.Errorf("missing Manager copy")
				break
			}
			s.session, workErr = e.Sessions.Open(x.ctx, SessionSpec{RunID: r.ID, Manager: m, Mandate: r.Mandate, Prepared: r.Prepared, Workspace: s.copy, Findings: append([]Finding(nil), r.Findings...), Tools: x.tools(s)})
			if workErr != nil {
				break
			}
			if s.session == nil {
				workErr = fmt.Errorf("missing Manager session")
				break
			}
			if observed, ok := s.session.(ObservedSession); ok {
				value := observed.Observation()
				if !imageIdentity(value.ProfileIdentity) || !imageIdentity(value.ImageID) || !imageIdentity(value.HarnessDigest) {
					workErr = fmt.Errorf("native Session observation incomplete")
					break
				}
				x.result.Sessions = append(x.result.Sessions, ManagerSessionObservation{Manager: m.Package, Observation: value})
			}
		}
	}
	var root Handoff
	stopped := false
	if workErr == nil {
		x.mu.Lock()
		s := x.states[organization.Root]
		task := Assignment{Task: rootTask, Findings: append([]Finding(nil), r.Findings...), Base: input}
		s.a.assign(task)
		x.queueLocked(s, entry{event: Event{Assignment: &task}})
		x.mu.Unlock()
		root, workErr = x.complete()
		x.mu.Lock()
		x.closing = true
		if x.asyncErr != nil {
			// The stopping cause replaces the cancellation it triggered.
			workErr, stopped = x.asyncErr, true
		}
		x.mu.Unlock()
	}
	if workErr == nil {
		x.result.Decision = root.Decision
		fixed := root.Result
		if r.Mandate != Implement {
			// Review scratch edits never become product.
			fixed = input
		}
		ref, err := service.Retain(x.ctx, x.states[organization.Root].copy, fixed)
		if err != nil {
			workErr = fmt.Errorf("retain independent candidate: %w", err)
		} else if ref.Commit != fixed.ID || ref.Tree != fixed.Tree || ref.Binding != r.Prepared.Binding() || ref.Project != r.Project {
			workErr = fmt.Errorf("retained candidate binding mismatch")
		} else {
			x.result.Candidate = &ref
		}
	}
	if workErr != nil {
		x.result.Errors = append(x.result.Errors, workErr.Error())
		x.result.Technical = Failed
		if x.ctx.Err() != nil && !stopped || errors.Is(workErr, ErrUnavailable) || errors.Is(workErr, ErrBudget) || errors.Is(workErr, context.Canceled) || errors.Is(workErr, context.DeadlineExceeded) || errors.Is(workErr, ErrObservationIncomplete) || errors.Is(workErr, ErrNoProgress) || errors.Is(workErr, project.ErrIncompleteInputs) {
			x.result.Technical = Incomplete
		}
	} else {
		x.result.Technical = Completed
	}
	x.mu.Lock()
	x.closing = true
	x.mu.Unlock()
	x.cancel()
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.CleanupTimeout)
	defer cancel()
	closeOne := func(name string, close func(context.Context) error) {
		err := boundedClose(cleanup, close)
		c := Closure{Resource: name, Closed: err == nil}
		if err != nil {
			c.Error = err.Error()
			x.result.Errors = append(x.result.Errors, "close "+name+": "+err.Error())
			x.result.Technical = Incomplete
		}
		x.result.Cleanup = append(x.result.Cleanup, c)
	}
	// Attempt every Session close together: one unresponsive adapter cannot
	// spend the shared bound before the other environments receive Close.
	type sessionClosure struct {
		name string
		err  error
	}
	sessionClosures := make(chan sessionClosure, len(x.states))
	sessionCleanup, sessionStop := context.WithTimeout(cleanup, r.CleanupTimeout/2)
	defer sessionStop()
	sessionCount := 0
	for _, m := range r.Prepared.Topology().Managers() {
		s := x.states[m.Package]
		if s.session != nil {
			sessionCount++
			go func() {
				sessionClosures <- sessionClosure{sessionResource(s.manager.Package), boundedClose(sessionCleanup, s.session.Close)}
			}()
		}
	}
	closures := map[string]error{}
	for i := 0; i < sessionCount; i++ {
		c := <-sessionClosures
		closures[c.name] = c.err
	}
	for _, m := range r.Prepared.Topology().Managers() {
		name := sessionResource(m.Package)
		err, ok := closures[name]
		if !ok {
			continue
		}
		c := Closure{Resource: name, Closed: err == nil}
		if err != nil {
			c.Error = err.Error()
			x.result.Errors = append(x.result.Errors, "close "+name+": "+err.Error())
			x.result.Technical = Incomplete
		}
		x.result.Cleanup = append(x.result.Cleanup, c)
	}
	drained := make(chan struct{})
	go func() { x.active.Wait(); x.calls.Wait(); close(drained) }()
	workersClosed := false
	drainCleanup, drainStop := context.WithDeadline(cleanup, time.Now().Add(r.CleanupTimeout/4))
	defer drainStop()
	select {
	case <-drained:
		workersClosed = true
	case <-drainCleanup.Done():
		x.result.Technical = Incomplete
		x.result.Errors = append(x.result.Errors, "Manager workers did not stop within cleanup bound")
	}
	// Cut off late check completion before any terminal result is assembled.
	x.result.Checks = x.finishChecks()
	x.result.Cleanup = append(x.result.Cleanup, Closure{Resource: workersResource, Closed: workersClosed})
	for _, m := range r.Prepared.Topology().Managers() {
		s := x.states[m.Package]
		if s.copy != nil {
			closeOne(copyResource(m.Package), s.copy.Close)
		}
		if !workersClosed {
			continue
		}
		x.mu.Lock()
		h := s.a.handoff
		x.mu.Unlock()
		if h.Manager != "" {
			x.result.Handoffs = append(x.result.Handoffs, h)
			x.result.Findings = append(x.result.Findings, h.Findings...)
			if h.Review != nil {
				x.result.Findings = append(x.result.Findings, h.Review.Findings...)
			}
		}
	}

	if service != nil {
		closeOne(gitServiceResource, service.Close)
	}
	x.result.Messages = r.Allowance.Used() - used
	terminal := copyResult(x.result)
	persist, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), r.PersistenceTimeout)
	defer persistCancel()
	if err := e.Records.Save(persist, terminal); err != nil {
		terminal.Technical = Incomplete
		terminal.Errors = append(terminal.Errors, "save terminal result: "+err.Error())
		return terminal, errors.Join(workErr, err)
	}
	stored, err := e.Records.Read(persist, r.Project, r.ID)
	if err != nil || !reflect.DeepEqual(stored, terminal) {
		if err == nil {
			err = fmt.Errorf("terminal result readback mismatch")
		}
		terminal.Technical = Incomplete
		terminal.Errors = append(terminal.Errors, err.Error())
		return terminal, errors.Join(workErr, err)
	}
	if terminal.Technical != Completed && workErr == nil {
		workErr = fmt.Errorf("Run resources not confirmed closed")
	}
	return terminal, workErr
}

// complete waits until the hierarchy is quiet, then validates Root's judgment.
// An invalid Root completion returns its diagnostic to Root, which continues.
func (x *execution) complete() (Handoff, error) {
	x.mu.Lock()
	x.settleLocked()
	x.mu.Unlock()
	for {
		select {
		case <-x.settled:
		case <-x.ctx.Done():
			return Handoff{}, x.ctx.Err()
		}
		x.mu.Lock()
		quiet := x.settleLocked()
		x.mu.Unlock()
		if !quiet {
			continue
		}
		h, done, err := x.completeRoot()
		if err != nil || done {
			return h, err
		}
		x.mu.Lock()
		x.settleLocked()
		x.mu.Unlock()
	}
}

func (x *execution) toolContext(caller context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(x.ctx)
	stop := context.AfterFunc(caller, cancel)
	return ctx, func() { stop(); cancel() }
}

// A Session may handle ordinary tool errors, but cannot erase a mandatory
// technical port failure by ignoring the returned error. Only the cause that
// stops the Run is kept; cancellation it triggers elsewhere is its consequence.
func (x *execution) stopPort(err error) {
	if err == nil {
		return
	}
	x.mu.Lock()
	if x.ctx.Err() == nil {
		x.asyncErr = errors.Join(x.asyncErr, err)
	}
	x.cancel()
	x.mu.Unlock()
}

func (x *execution) tools(s *managerState) Tools {
	// An action that succeeds is progress on the assignment; reading Notes or
	// running the check is not.
	call := func(caller context.Context, action bool, f func(context.Context) error) error {
		ctx, stop := x.toolContext(caller)
		defer stop()
		err := f(ctx)
		if err == nil && action {
			x.mu.Lock()
			s.progress = true
			x.mu.Unlock()
		}
		return err
	}
	return Tools{
		Delegate: func(caller context.Context, child, task string, findings []Finding) error {
			return call(caller, true, func(ctx context.Context) error { return x.delegate(ctx, s, child, task, findings) })
		},
		Await: func(caller context.Context) (event Event, err error) {
			err = call(caller, true, func(ctx context.Context) error { event, err = x.await(ctx, s); return err })
			return event, err
		},
		Review: func(caller context.Context, child, commit string, decision Decision, findings []Finding) error {
			return call(caller, true, func(ctx context.Context) error { return x.review(ctx, s, child, commit, decision, findings) })
		},
		Ask: func(caller context.Context, to, text string) (id string, err error) {
			err = call(caller, true, func(ctx context.Context) error { id, err = x.ask(ctx, s, to, text); return err })
			return id, err
		},
		Reply: func(caller context.Context, id, text string) error {
			return call(caller, true, func(ctx context.Context) error { return x.reply(ctx, s, id, text) })
		},
		Judge: func(caller context.Context, decision Decision, findings []Finding) error {
			return call(caller, true, func(ctx context.Context) error { return x.judge(ctx, s, decision, findings) })
		},
		Check: func(caller context.Context) (result Check, err error) {
			err = call(caller, false, func(ctx context.Context) error { result, err = x.checkTool(ctx, s); return err })
			return result, err
		},
		// Notes are optional information: a storage failure is reported to the
		// Manager, which sees that nothing was read or kept, and does not stop
		// the Run. Mandatory results are stored separately and still stop it.
		ReadNotes: func(caller context.Context, after string) (page NotePage, err error) {
			err = call(caller, false, func(ctx context.Context) error {
				page, err = x.executor.Records.Notes(ctx, x.request.Project, after)
				return err
			})
			return page, err
		},
		AppendNote: func(caller context.Context, scope, text string) (note Note, err error) {
			err = call(caller, true, func(ctx context.Context) error {
				if text == "" {
					return fmt.Errorf("empty Note; append it again with a text")
				}
				note, err = x.executor.Records.Append(ctx, Note{Project: x.request.Project, Run: x.request.ID, Manager: s.manager.Package, Context: scope, Text: text})
				return err
			})
			return note, err
		},
	}
}

// check runs the shared deterministic check on one fixed state of s's copy:
// structural validity and accounting, then the declared checks placed there.
// Assessment judges model readiness and runs only the structural part.
func (x *execution) check(ctx context.Context, s *managerState, fixed Commit) (*Check, error) {
	result := &Check{Commit: fixed, Checks: []string{}}
	err := x.checkErr(ctx, s, fixed, result)
	if errors.Is(err, project.ErrCandidateInvalid) {
		result.Diagnostic = err.Error()
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	result.Valid = true
	return result, nil
}

func (x *execution) checkErr(ctx context.Context, s *managerState, commit Commit, result *Check) error {
	a, err := s.copy.Inspect(ctx, commit)
	if err == nil {
		err = x.request.Prepared.ValidateCandidate(a)
	}
	if err != nil || x.request.Mandate == Assessment {
		return err
	}
	declarations := x.request.Prepared.ChecksFor(s.manager.Package)
	if len(declarations) != 0 && x.executor.Checks == nil {
		return fmt.Errorf("declared checks require concrete runner")
	}
	if x.executor.Checks == nil {
		return nil
	}
	observed, checkErr := x.executor.Checks.Check(ctx, declarations, s.copy, commit)
	if err := x.recordChecks(observed); err != nil {
		return errors.Join(checkErr, err)
	}
	if checkErr != nil {
		return checkErr
	}
	dirs, err := project.CheckDirectories(a.Model().Inputs().Raw)
	if err != nil {
		return err
	}
	for _, d := range declarations {
		result.Checks = append(result.Checks, d.ID.Value)
	}
	return ValidateCheckObservations(declarations, observed, commit, a.Model().Inputs().Raw.Census, dirs)
}

// x.mu serializes shutdown with all positive WaitGroup registrations.
func (x *execution) admissionErrorLocked() error {
	if err := x.ctx.Err(); err != nil {
		return err
	}
	if x.closing {
		return fmt.Errorf("%w: Run is closing", ErrObservationIncomplete)
	}
	return nil
}

func concreteFindings(findings []Finding) bool {
	for _, f := range findings {
		if strings.TrimSpace(f.Text) != "" {
			return true
		}
	}
	return false
}

// origin sets the host-known author of Findings; agents never name it.
func origin(findings []Finding, manager string) []Finding {
	out := make([]Finding, len(findings))
	for i, f := range findings {
		out[i] = Finding{Manager: manager, Text: f.Text}
	}
	return out
}

func (x *execution) recordChecks(observed []CheckObservation) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.receiptsClosed {
		return fmt.Errorf("%w: terminal check collection is closed", ErrObservationIncomplete)
	}
	x.checks = append(x.checks, copyChecks(observed)...)
	return nil
}
func (x *execution) finishChecks() []CheckObservation {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.receiptsClosed = true
	return copyChecks(x.checks)
}
