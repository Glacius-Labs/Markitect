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

type completion struct {
	handoff Handoff
	err     error
}
type managerState struct {
	manager                      organization.Manager
	copy                         Copy
	session                      Session
	base                         Commit
	handoff                      Handoff
	mu                           sync.Mutex
	running, delivered, assigned bool
	accepting                    bool
	working                      bool
	lateDecision                 Decision
	judgment                     Decision
	idle                         chan struct{}
	done                         chan completion
	inbox                        chan Event
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
	active         sync.WaitGroup
	questions      map[string]Question
	answered       map[string]bool
	sequence       int
	closing        bool
	asyncErr       error
	checks         []CheckObservation
	receiptsClosed bool
	calls          sync.WaitGroup
}

func (e *Executor) Execute(ctx context.Context, r Request) (Result, error) {
	x := &execution{executor: e, request: r, states: map[string]*managerState{}, questions: map[string]Question{}, answered: map[string]bool{}, result: Result{ID: r.ID, Project: r.Project, Binding: r.Prepared.Binding(), Mandate: r.Mandate, Input: r.Input, Technical: Incomplete}}
	if err := validateRequest(r); err != nil {
		return x.result, err
	}
	if e == nil || e.Sessions == nil || e.Workspaces == nil || e.Records == nil {
		return x.result, fmt.Errorf("Run ports unavailable")
	}
	x.ctx, x.cancel = context.WithDeadline(ctx, r.Allowance.Deadline)
	defer x.cancel()
	for _, m := range r.Prepared.Topology().Managers() {
		x.states[m.Package] = &managerState{manager: m, base: Commit{r.Input.Commit, r.Input.Tree}, done: make(chan completion, len(m.Children)), inbox: make(chan Event, 256), accepting: true, idle: make(chan struct{})}
	}
	used := r.Allowance.Used()
	service, workErr := e.Workspaces.Open(x.ctx, r)
	x.workspace = service
	if workErr == nil && service == nil {
		workErr = fmt.Errorf("workspace startup returned no handle")
	}
	// Complete environment and communication tree exists before ordinary work.
	if workErr == nil {
		for _, m := range r.Prepared.Topology().Managers() {
			s := x.states[m.Package]
			s.copy, workErr = service.Copy(x.ctx, m.Package, s.base)
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
	if workErr == nil {
		// Reserve every primary conversation before a child can ask another Manager
		// a question; wakeups must not start a second initial Exchange in that slot.
		for _, state := range x.states {
			state.running = true
			state.working = true
		}
		x.states[organization.Root].assigned = true
		for _, m := range r.Prepared.Topology().Managers() {
			if m.Package == organization.Root {
				continue
			}
			if workErr = x.startWorker(x.states[m.Package], "Receive your parent assignment with Await; respond to incoming questions while waiting.", nil); workErr != nil {
				break
			}
		}
		if workErr == nil {
			root, workErr = x.work(x.states[organization.Root], "Complete the bound mandate with your complete Manager hierarchy.", r.Findings)
		}
		x.finishedTask(x.states[organization.Root], workErr)
		if workErr == nil {
			done := make(chan struct{})
			go func() { x.active.Wait(); close(done) }()
			select {
			case <-done:
			case <-x.ctx.Done():
				workErr = x.ctx.Err()
			}
		}
		x.mu.Lock()

		if x.asyncErr != nil {
			workErr = errors.Join(workErr, x.asyncErr)
		}
		if len(x.questions) > 0 && workErr == nil {
			workErr = fmt.Errorf("Run ended with unresolved Manager questions")
		}
		x.closing = true
		x.mu.Unlock()
	}

	if workErr == nil {
		root.Decision, workErr = x.aggregate(x.states[organization.Root])
		if workErr == nil && r.Mandate == Implement {
			for _, m := range r.Prepared.Topology().Managers() {
				state := x.states[m.Package]
				latest, err := state.copy.Snapshot(x.ctx)
				if err != nil {
					workErr = err
					break
				}
				state.mu.Lock()
				fixed := state.handoff.Result
				state.mu.Unlock()
				if latest != fixed {
					workErr = fmt.Errorf("%w: Manager %s changed after its fixed handoff", ErrObservationIncomplete, m.Package)
					break
				}
			}
		}
	}

	if workErr == nil {
		x.result.Decision = root.Decision
		fixed := root.Result
		if r.Mandate != Implement {
			fixed = Commit{r.Input.Commit, r.Input.Tree}
		}
		// Review scratch edits never become product. Invalid input can be rejected.
		if r.Mandate != Implement {
			// Assessment judges model readiness before realization exists. Keep
			// fixed structural validation, but do not demand passing product tests.
			checkErr := x.check(x.states[organization.Root], fixed, r.Mandate == Verify)
			if checkErr != nil && (root.Decision == Approve || !errors.Is(checkErr, project.ErrCandidateInvalid)) {
				workErr = checkErr
			}
		}
		if workErr == nil {
			ref, err := service.Retain(x.ctx, x.states[organization.Root].copy, fixed)
			if err != nil {
				workErr = fmt.Errorf("retain independent candidate: %w", err)
			} else if ref.Commit != fixed.ID || ref.Tree != fixed.Tree || ref.Binding != r.Prepared.Binding() || ref.Project != r.Project {
				workErr = fmt.Errorf("retained candidate binding mismatch")
			} else {
				x.result.Candidate = &ref
			}
		}
	}
	if workErr != nil {
		x.result.Errors = append(x.result.Errors, workErr.Error())
		x.result.Technical = Failed
		if x.ctx.Err() != nil || errors.Is(workErr, ErrUnavailable) || errors.Is(workErr, ErrBudget) || errors.Is(workErr, context.Canceled) || errors.Is(workErr, context.DeadlineExceeded) || errors.Is(workErr, ErrObservationIncomplete) || errors.Is(workErr, project.ErrIncompleteInputs) {
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
				sessionClosures <- sessionClosure{"session:" + s.manager.Package, boundedClose(sessionCleanup, s.session.Close)}
			}()
		}
	}
	closures := map[string]error{}
	for i := 0; i < sessionCount; i++ {
		c := <-sessionClosures
		closures[c.name] = c.err
	}
	for _, m := range r.Prepared.Topology().Managers() {
		name := "session:" + m.Package
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
	x.result.Cleanup = append(x.result.Cleanup, Closure{Resource: "manager-workers", Closed: workersClosed})
	for _, m := range r.Prepared.Topology().Managers() {
		s := x.states[m.Package]
		if s.copy != nil {
			closeOne("copy:"+m.Package, s.copy.Close)
		}
		if workersClosed && s.handoff.Manager != "" {
			x.result.Handoffs = append(x.result.Handoffs, s.handoff)
			x.result.Findings = append(x.result.Findings, s.handoff.Findings...)
			if s.handoff.Review != nil {
				x.result.Findings = append(x.result.Findings, s.handoff.Review.Findings...)
			}
		}
	}

	if service != nil {
		closeOne("git-service", service.Close)
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

func (x *execution) toolContext(caller context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(x.ctx)
	stop := context.AfterFunc(caller, cancel)
	return ctx, func() { stop(); cancel() }
}

// A Session may handle ordinary tool errors, but cannot erase a mandatory
// technical port failure by ignoring the returned error.
func (x *execution) stopPort(err error) {
	if err == nil {
		return
	}
	x.mu.Lock()
	x.asyncErr = errors.Join(x.asyncErr, err)
	x.cancel()
	x.mu.Unlock()
}

func (x *execution) tools(s *managerState) Tools {
	return Tools{
		Ask:   func(ctx context.Context, to, text string) (string, error) { return x.ask(ctx, s, to, text) },
		Reply: func(ctx context.Context, id, text string) error { return x.reply(ctx, s, id, text) },

		ReadNotes: func(caller context.Context) ([]Note, error) {
			ctx, stop := x.toolContext(caller)
			defer stop()
			notes, err := x.executor.Records.Notes(ctx, x.request.Project)
			x.stopPort(err)
			return notes, err
		},
		AppendNote: func(caller context.Context, scope, text string) (Note, error) {
			ctx, stop := x.toolContext(caller)
			defer stop()
			if text == "" {
				return Note{}, fmt.Errorf("empty Note")
			}
			note, err := x.executor.Records.Append(ctx, Note{Project: x.request.Project, Run: x.request.ID, Manager: s.manager.Package, Context: scope, Text: text})
			x.stopPort(err)
			return note, err
		},
		Delegate: func(caller context.Context, child, task string, feedback []Finding) error {
			ctx, stop := x.toolContext(caller)
			defer stop()
			target, ok := x.states[child]
			if !ok || target.manager.Parent != s.manager.Package || task == "" {
				return fmt.Errorf("delegation requires direct child and concrete task")
			}
			for {
				target.mu.Lock()
				if !target.assigned || !target.running || target.working {
					break
				}
				idle := target.idle
				target.mu.Unlock()
				select {
				case <-idle:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			defer target.mu.Unlock()
			if target.assigned && (target.running || !target.delivered || target.handoff.Review == nil) {
				return fmt.Errorf("child task active or accepted")
			}
			if target.assigned && !concreteFindings(feedback) {
				return fmt.Errorf("correction of completed child requires findings")
			}
			if target.lateDecision == ModelBlocker || target.judgment == ModelBlocker || target.handoff.Review != nil && target.handoff.Review.Decision == ModelBlocker {
				return fmt.Errorf("a model blocker requires a new model and Run")
			}
			s.mu.Lock()
			base := Commit{x.request.Input.Commit, x.request.Input.Tree}
			var err error
			if x.request.Mandate == Implement {
				base, err = s.copy.Snapshot(ctx)
				if err == nil {
					err = target.copy.Update(ctx, s.copy, base)
					if errors.Is(err, ErrMergeConflict) {
						feedback = append(append([]Finding(nil), feedback...), Finding{Manager: s.manager.Package, Text: "Resolve the preserved child-side Git merge conflict against the current parent baseline: " + err.Error()})
						err = nil
					}
				}
			}
			s.mu.Unlock()
			if err != nil {
				if !errors.Is(err, ErrMergeConflict) {
					x.stopPort(err)
				}
				return err
			}

			// Git may return after cancellation. Admission and state publication share
			// the same lock as shutdown and every worker/call registration.
			x.mu.Lock()
			if err = ctx.Err(); err == nil {
				err = x.admissionErrorLocked()
			}
			if err != nil {
				x.mu.Unlock()
				return err
			}
			if target.lateDecision == Reject {
				target.lateDecision = ""
			}
			target.base = base
			first := !target.assigned
			target.assigned = true
			target.accepting = true
			target.delivered = false
			target.handoff.Review = nil
			if first {
				x.mu.Unlock()
				assignment := Assignment{task, append([]Finding(nil), feedback...), base}
				select {
				case target.inbox <- Event{Assignment: &assignment}:
					return nil
				case <-ctx.Done():
					target.assigned = false
					target.delivered = false
					return ctx.Err()
				}
			}
			target.running = true
			target.working = true
			target.idle = make(chan struct{})
			x.active.Add(1)
			x.mu.Unlock()
			x.launchWorker(target, task, feedback)
			return nil
		},

		Await: func(caller context.Context) (Event, error) {
			ctx, stop := x.toolContext(caller)
			defer stop()
			select {
			case event := <-s.inbox:
				if event.Answer != nil {
					x.mu.Lock()
					q, ok := x.questions[event.Answer.ID]
					if !ok || q.From != s.manager.Package || q.To != event.Answer.From || !x.answered[q.ID] {
						x.mu.Unlock()
						return Event{}, fmt.Errorf("answer correlation mismatch")
					}
					delete(x.questions, q.ID)
					delete(x.answered, q.ID)
					x.mu.Unlock()
				}
				return event, nil
			case c := <-s.done:
				if c.err != nil {
					return Event{}, c.err
				}
				child := x.states[c.handoff.Manager]
				child.mu.Lock()
				child.delivered = true
				child.mu.Unlock()
				s.mu.Lock()
				diff, err := s.copy.Review(ctx, child.copy, c.handoff.Base, c.handoff.Result)
				s.mu.Unlock()
				if err != nil {
					x.stopPort(err)
					return Event{}, err
				}
				c.handoff.Diff = diff
				child.mu.Lock()
				child.handoff.Diff = diff
				child.mu.Unlock()
				return Event{Handoff: &c.handoff}, nil
			case <-ctx.Done():
				return Event{}, ctx.Err()
			}
		},

		Review: func(caller context.Context, child, commit string, decision Decision, findings []Finding) error {
			ctx, stop := x.toolContext(caller)
			defer stop()
			target, ok := x.states[child]
			if !ok || target.manager.Parent != s.manager.Package {
				return fmt.Errorf("review requires direct child")
			}
			target.mu.Lock()
			defer target.mu.Unlock()
			if target.working || !target.delivered || target.handoff.Result.ID != commit || target.handoff.Review != nil {
				return fmt.Errorf("review requires exact unreviewed fixed child")
			}
			if decision != Approve && decision != Reject && decision != ModelBlocker {
				return fmt.Errorf("review decision missing")
			}
			if decision != Approve && !concreteFindings(findings) {
				return fmt.Errorf("rejection/model blocker requires findings")
			}
			for i := range findings {
				findings[i].Manager = s.manager.Package
			}
			if decision == Approve && x.request.Mandate == Implement && target.handoff.Decision == Submitted {
				s.mu.Lock()
				before, err := s.copy.Snapshot(ctx)
				if err == nil {
					var after Commit
					after, err = s.copy.Integrate(ctx, target.copy, target.handoff.Result)
					if err == nil {
						s.handoff.Children = append(s.handoff.Children, Integration{child, target.handoff.Base, target.handoff.Result, before, after})
					}
				}
				s.mu.Unlock()
				if err != nil {
					if !errors.Is(err, ErrMergeConflict) {
						x.stopPort(err)
					}
					target.handoff.Review = &ParentReview{Decision: Reject, Findings: []Finding{{Manager: s.manager.Package, Text: "Git integration failed: " + err.Error()}}}
					return err
				}
			}
			target.handoff.Review = &ParentReview{decision, append([]Finding(nil), findings...)}
			return nil
		},
	}
}

func (x *execution) work(s *managerState, task string, feedback []Finding) (Handoff, error) {
	for {
		if err := x.ctx.Err(); err != nil {
			return Handoff{}, err
		}
		if err := x.request.Allowance.take(); err != nil {
			return Handoff{}, err
		}
		r, err := x.exchange(s, Message{Task: task, Feedback: feedback})
		if err != nil {
			return Handoff{}, err
		}
		if r.Decision != Approve && r.Decision != Reject && r.Decision != ModelBlocker && r.Decision != Submitted {
			return Handoff{}, fmt.Errorf("Manager %s supplied no semantic result", s.manager.Package)
		}
		if x.request.Mandate != Implement && r.Decision == Submitted {
			return Handoff{}, fmt.Errorf("review mandate requires judgment")
		}
		if x.request.Mandate == Implement && r.Decision == Approve {
			return Handoff{}, fmt.Errorf("Implement submits candidate; cannot approve publication")
		}
		if r.Decision != Submitted && r.Decision != Approve && !concreteFindings(r.Findings) {
			return Handoff{}, fmt.Errorf("rejection/model blocker requires findings")
		}
		feedback = nil

		s.mu.Lock()
		assigned := s.assigned
		s.mu.Unlock()
		if !assigned {
			feedback = []Finding{{Manager: s.manager.Package, Text: "Await your direct parent assignment before submitting."}}
			continue
		}
		for _, child := range s.manager.Children {
			c := x.states[child]
			c.mu.Lock()
			if c.handoff.Manager == "" || c.handoff.Review == nil || c.working {
				feedback = append(feedback, Finding{Manager: s.manager.Package, Text: "Complete delegation and review for child " + child})
			}
			c.mu.Unlock()
		}
		if len(feedback) != 0 {
			continue
		}
		commit, err := s.copy.Snapshot(x.ctx)
		if errors.Is(err, ErrMergeConflict) {
			feedback = []Finding{{Manager: s.manager.Package, Text: err.Error()}}
			continue
		}
		if err != nil {
			return Handoff{}, err
		}
		if x.request.Mandate == Implement && r.Decision == Submitted {
			if err = x.check(s, commit, true); err != nil {
				if !errors.Is(err, project.ErrCandidateInvalid) {
					return Handoff{}, err
				}
				feedback = []Finding{{Manager: s.manager.Package, Text: err.Error()}}
				continue
			}
		}
		s.mu.Lock()
		x.mu.Lock()
		for _, q := range x.questions {
			if q.From == s.manager.Package || q.To == s.manager.Package && !x.answered[q.ID] {
				feedback = append(feedback, Finding{Manager: s.manager.Package, Text: "Resolve pending question " + q.ID + " before completing."})
			}
		}
		x.mu.Unlock()
		s.mu.Unlock()
		if len(feedback) != 0 {
			continue
		}
		for i := range r.Findings {
			r.Findings[i].Manager = s.manager.Package
		}
		childDecision := Decision("")
		for _, child := range s.manager.Children {
			c := x.states[child]
			c.mu.Lock()
			childDecision = requiredDecision(childDecision, c.handoff.Decision)
			childDecision = requiredDecision(childDecision, c.handoff.Review.Decision)
			c.mu.Unlock()
		}
		h := s.finish(commit, r, childDecision)
		return h, nil
	}
}
func (x *execution) check(s *managerState, commit Commit, required bool) error {
	a, err := s.copy.Inspect(x.ctx, commit)
	if err == nil {
		err = x.request.Prepared.ValidateCandidate(a)
	}
	if err != nil {
		return err
	}
	if !required {
		return nil
	}
	declarations := x.request.Prepared.ChecksFor(s.manager.Package)
	if len(declarations) != 0 && x.executor.Checks == nil {
		return fmt.Errorf("declared checks require concrete runner")
	}
	if x.executor.Checks != nil {
		observed, checkErr := x.executor.Checks.Check(x.ctx, declarations, s.copy, commit)
		if err := x.recordChecks(observed); err != nil {
			return errors.Join(checkErr, err)
		}
		if checkErr != nil {
			return checkErr
		}
		expected := map[string]bool{}
		for _, d := range declarations {
			expected[d.ID.Value] = true
		}
		for _, c := range observed {
			if !expected[c.ID] || c.Commit != commit.ID || c.Receipt.ID != CheckID(c.ID) || c.Receipt.Tree != a.Model().Inputs().Raw.Census.Digest() {
				return fmt.Errorf("check %s did not pass exactly once on fixed commit", c.ID)
			}
			receipt := c.Receipt
			dirs, err := project.CheckDirectories(a.Model().Inputs().Raw)
			if err != nil {
				return err
			}
			if !sameDirectories(receipt.Directories, dirs) {
				return fmt.Errorf("%w: check directory inventory changed", ErrObservationIncomplete)
			}
			if !receipt.Started || !receipt.Exited || receipt.Uncertain || receipt.TimedOut || !receipt.CopiesClosed || !imageIdentity(receipt.ImageID) || c.Error != "" {
				return fmt.Errorf("%w: check %s", ErrObservationIncomplete, c.ID)
			}
			if receipt.ExitCode != 0 {
				return fmt.Errorf("%w: check %s exited %d", project.ErrCandidateInvalid, c.ID, receipt.ExitCode)
			}
			delete(expected, c.ID)
		}
		if len(expected) != 0 {
			return fmt.Errorf("declared check observations incomplete")
		}
	}
	return nil
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
func (x *execution) startWorker(s *managerState, task string, feedback []Finding) error {
	s.mu.Lock()
	x.mu.Lock()
	if err := x.admissionErrorLocked(); err != nil {
		x.mu.Unlock()
		s.mu.Unlock()
		return err
	}
	s.running = true
	s.working = true
	x.active.Add(1)
	x.mu.Unlock()
	s.mu.Unlock()
	x.launchWorker(s, task, feedback)
	return nil
}

// launchWorker starts only work already registered under the admission lock.
func (x *execution) launchWorker(s *managerState, task string, feedback []Finding) {
	go func() {
		defer x.active.Done()
		h, err := x.work(s, task, feedback)
		if err != nil {
			x.stopPort(err)
		}
		x.finishedTask(s, err)
		x.states[s.manager.Parent].done <- completion{h, err}
	}()
}

func concreteFindings(findings []Finding) bool {
	for _, f := range findings {
		if strings.TrimSpace(f.Text) != "" {
			return true
		}
	}
	return false
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
