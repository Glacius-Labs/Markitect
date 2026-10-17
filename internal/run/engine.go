package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const candidateRecordMedia = "application/vnd.markitect.candidate-record+json"

// SessionProfile is the Host profile a role's sessions run with.
type SessionProfile struct {
	ImageID, CredentialRef string
	Profile                Payload
	MaxWireBytes           int64
}

// EngineLimits bound one step's invocations and observations. A step's own
// timeout is its Manager's bound stepTimeout.
type EngineLimits struct {
	StartTimeout, StopTimeout time.Duration
	HarvestEntries            int
	HarvestBytes              int64
	Record                    realization.CandidateRecordLimits
}

// ProfileRequest asks for the session profile of one step: the Manager, the
// role, the phase and whether its workspace is read-only, so that an adapter
// offers only the tools the step may use.
type ProfileRequest struct {
	Manager, Role string
	Phase         Phase
	ReadOnly      bool
}

// EngineConfig supplies everything the engine needs from the application.
type EngineConfig struct {
	Coordinator *Coordinator
	Journal     Journal
	Sessions    Sessions
	Workspaces  Workspaces
	Checks      Checks
	// ReviewChecks are the configured checks a candidate review may request,
	// by name; their trees are filled in by the engine.
	ReviewChecks map[string]CheckRequest
	Preparation  StepPreparation
	Profile      func(ProfileRequest) (SessionProfile, error)
	Clock        func() time.Time
	Limits       EngineLimits
}

// Engine drives a run's admitted steps through the ports, one at a time (core
// rules S1 to S6, T4). It holds the Host-side facts the reducer does not keep:
// the compiled model, every version of the run aggregate, the path
// reservations and the registry records each task proposed. It decides
// nothing the Coordinator has not admitted, and it stops at the first
// uncertain or failed effect.
type Engine struct {
	cfg      EngineConfig
	compiled model.Result
	versions []realization.Aggregate
}

var (
	errStep        = errors.New("step stopped before its conclusion")
	errNotAdmitted = errors.New("the check was not admitted")
)

// NewEngine binds an engine to its run. It rebuilds the aggregate from the
// captured source and the run's accepted deliveries (core rule D2).
func NewEngine(ctx context.Context, cfg EngineConfig) (*Engine, error) {
	if cfg.Coordinator == nil || cfg.Journal == nil || cfg.Sessions == nil || cfg.Workspaces == nil || cfg.Profile == nil || cfg.Clock == nil {
		return nil, ErrInvalid
	}
	compiled := model.Compile(cfg.Preparation.Binding.ModelInput)
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return nil, err
	}
	settings, err := organization.ResolveSettings(topology, cfg.Preparation.Settings)
	if err != nil {
		return nil, err
	}
	areas, err := creationAreas(settings)
	if err != nil {
		return nil, err
	}
	in := cfg.Preparation.Binding
	facts, err := realization.NewPathFacts(compiled, in.Registry, in.Census, areas, in.Protections)
	if err != nil {
		return nil, err
	}
	base, err := realization.NewCandidateBase(compiled, facts)
	if err != nil {
		return nil, err
	}
	aggregate, err := realization.NewAggregate(base)
	if err != nil {
		return nil, err
	}
	e := &Engine{cfg: cfg, compiled: compiled, versions: []realization.Aggregate{aggregate}}
	var composed []Delivery
	for _, t := range cfg.Coordinator.Tasks() {
		for _, d := range t.Deliveries {
			if d.Candidate != (BlobRef{}) {
				composed = append(composed, d)
			}
		}
	}
	slices.SortFunc(composed, func(a, b Delivery) int { return compareUint(a.Aggregate, b.Aggregate) })
	for _, d := range composed {
		candidate, err := e.candidate(ctx, d.Candidate)
		if err != nil {
			return nil, err
		}
		next, ok := realization.ReplayDelivery(e.aggregate(), candidate).Aggregate()
		if !ok || next.Version() != d.Aggregate {
			return nil, ErrState
		}
		e.versions = append(e.versions, next)
	}
	if uint64(len(e.versions)-1) != cfg.Coordinator.AggregateVersion() {
		return nil, ErrState
	}
	return e, nil
}

func creationAreas(settings organization.Settings) ([]realization.CreationArea, error) {
	var areas []realization.CreationArea
	for _, m := range settings.Managers() {
		owner, err := managerOwner(m.Package)
		if err != nil {
			return nil, err
		}
		for _, directory := range m.CreationRoots.Value {
			area, err := realization.NewCreationArea(owner, directory)
			if err != nil {
				return nil, err
			}
			areas = append(areas, area)
		}
	}
	return areas, nil
}

func managerOwner(manager string) (realization.Owner, error) {
	if manager == organization.Root {
		return realization.RootOwner(), nil
	}
	return realization.PackageOwner(manager)
}

func (e *Engine) aggregate() realization.Aggregate { return e.versions[len(e.versions)-1] }

// Aggregate returns the run's current accepted aggregate.
func (e *Engine) Aggregate() realization.Aggregate { return e.aggregate() }

// Run advances the run until no step is admissible and then records why: the
// root's completion leaves the run Verifying (core rule T13), an owner
// question moves it to Waiting (core rule W1), and an incomplete task, a
// budget that no longer fits the next step or a wait nothing can satisfy
// finishes it Incomplete (core rules T12, T14). Waits that only a parent's
// dispatch could resolve (core rule T12) end the run Incomplete in this
// version.
func (e *Engine) Run(ctx context.Context) error {
	c := e.cfg.Coordinator
	for {
		advanced, err := e.Advance(ctx)
		if errors.Is(err, ErrBudget) {
			return e.finish(ctx, "the budget does not fit the next admissible step")
		}
		if _, failed := errors.AsType[stepFailure](err); failed {
			continue // charged and concluded as failed; the budget bounds retries
		}
		if err != nil {
			return err
		}
		if !advanced {
			break
		}
	}
	if c.Current().State != Running {
		return nil
	}
	var incomplete []string
	for _, t := range c.Tasks() {
		if t.State == TaskIncomplete {
			incomplete = append(incomplete, t.Manager+": "+t.Reason)
		}
	}
	var questions []string
	for _, q := range c.OwnerQuestions() {
		questions = append(questions, q.ID)
	}
	switch {
	case len(incomplete) > 0:
		return e.finish(ctx, "incomplete tasks: "+strings.Join(incomplete, "; "))
	case len(questions) > 0:
		return e.commit(ctx, Command{ID: e.meta("wait").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: WaitRun, Assignment: c.Current().Assignment, Run: c.Current().ID,
			Reason: "open questions to the owner: " + strings.Join(questions, ", "), NextActor: addressedToOwner})
	}
	return e.finish(ctx, "no step is admissible and the remaining waits cannot be satisfied")
}

// RequiredCheck is one configured check that the published tree must pass.
type RequiredCheck struct {
	Name    string
	Request CheckRequest
}

// Verify runs the required checks on the tree that publishing the aggregate
// writes and returns the passing ones, which the final evidence check names.
// A check that already passed on that tree in this run is not run again. A
// failing check, or one the budget no longer fits, finishes the run Incomplete
// with its receipts retained; correcting it through the Managers is not
// supported yet.
func (e *Engine) Verify(ctx context.Context, target snapshot.Tree, required []RequiredCheck) ([]CheckID, error) {
	c := e.cfg.Coordinator
	if c.Current().State != Verifying || len(required) > 0 && e.cfg.Checks == nil {
		return nil, ErrState
	}
	var passed []CheckID
	for _, check := range required {
		if id, ok := e.passedOn(check.Name, target.Digest()); ok {
			passed = append(passed, id)
			continue
		}
		req := check.Request
		req.Tree, req.ID = target, CheckID(fmt.Sprintf("%s-%s-%d", req.ID, c.Current().ID, c.Revision()))
		cmd, err := c.PrepareCheckCommand(e.meta("prepare-check"), check.Name, req, e.checkReservation(req))
		if err != nil {
			return nil, err
		}
		receipt, _, err := e.runCheck(ctx, cmd, req)
		if errors.Is(err, ErrBudget) {
			return nil, e.finish(ctx, "the budget does not fit the required check "+check.Name)
		}
		if err != nil {
			return nil, err
		}
		if !receipt.Passed(req) {
			return nil, e.finish(ctx, "the required check "+check.Name+" did not pass on the aggregate")
		}
		passed = append(passed, req.ID)
	}
	return passed, nil
}

// passedOn finds a verification check of the run that already passed on tree.
func (e *Engine) passedOn(name string, tree snapshot.TreeID) (CheckID, bool) {
	for _, v := range e.cfg.Coordinator.Checks() {
		if v.Record.Task == "" && v.Record.Name == name && v.Record.Tree == tree && v.Passed {
			return v.Record.ID, true
		}
	}
	return "", false
}

// checkReservation holds a check's timeout plus the container's start and stop.
func (e *Engine) checkReservation(req CheckRequest) Reservation {
	return Reservation{ID: "check-" + string(req.ID), Kind: ProjectCheck, Purpose: Assessment, Maximum: req.Timeout + e.cfg.Limits.StartTimeout + e.cfg.Limits.StopTimeout}
}

// runCheck admits one check, runs it and records its receipt. An uncertain
// receipt leaves the run to reconciliation.
// A check whose admission the reducer refuses ran nowhere and returns
// errNotAdmitted with the refusal; it returns the time the check ran.
func (e *Engine) runCheck(ctx context.Context, admit Command, req CheckRequest) (CheckReceipt, time.Duration, error) {
	c := e.cfg.Coordinator
	if err := e.commit(ctx, admit); err != nil {
		if errors.Is(err, ErrUncertain) {
			return CheckReceipt{}, 0, err
		}
		return CheckReceipt{}, 0, fmt.Errorf("%w: %w", errNotAdmitted, err)
	}
	began := e.cfg.Clock()
	receipt, runErr := e.cfg.Checks.Run(ctx, req)
	elapsed := e.cfg.Clock().Sub(began)
	if receipt.ID == "" {
		receipt.ID = req.ID
	}
	if runErr != nil && receipt.Reason == "" {
		receipt.Reason = runErr.Error()
	}
	observed, err := c.ObserveCheckCommand(e.meta("observe-check"), receipt, elapsed)
	if err != nil {
		return receipt, elapsed, err
	}
	if err := e.commit(ctx, observed); err != nil {
		return receipt, elapsed, err
	}
	if receipt.Uncertain {
		return receipt, elapsed, errors.Join(errStep, runErr)
	}
	return receipt, elapsed, nil
}

func (e *Engine) finish(ctx context.Context, reason string) error {
	c := e.cfg.Coordinator
	return e.commit(ctx, Command{ID: e.meta("finish").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: FinishRun, Assignment: c.Current().Assignment, Run: c.Current().ID, Outcome: Incomplete, Reason: reason})
}

// stepVersion is the content a step's workspace starts from.
type stepVersion struct {
	tree     snapshot.Tree
	registry realization.Registry
}

// stepFailure reports a step the Host could not finish but concluded as
// failed (core rule T6): the run can continue with its next step.
type stepFailure struct{ err error }

func (f stepFailure) Error() string {
	return "the step failed and was concluded as failed: " + f.err.Error()
}
func (f stepFailure) Unwrap() error { return f.err }

// stepRun is the progress of one admitted step, which decides what the Host
// must settle when the step cannot finish.
type stepRun struct {
	task         string
	manager      string
	phase        Phase
	session      SessionID
	plan         SessionPlan
	version      stepVersion
	openID       string
	stepID       StepID
	stepInvoke   string
	channel      SessionIO
	began        time.Time
	openEnded    time.Time
	stepBegan    time.Time
	stepEnded    time.Time
	deadline     time.Time
	checked      time.Duration
	pending      []Claim
	opened       bool
	openSettled  bool
	stepPrepared bool
	dispatched   bool
	stepTerminal bool
	stepSettled  bool
}

// Advance runs the run's next admissible step to its conclusion. It reports
// false when nothing is admissible. A step the Host cannot finish is stopped,
// its known effects are settled, and it concludes as failed (core rules S2,
// T6); that error is a stepFailure. An effect whose end is not known leaves
// the run to reconciliation.
func (e *Engine) Advance(ctx context.Context) (bool, error) {
	if err := e.recover(ctx); err != nil {
		return true, err
	}
	c := e.cfg.Coordinator
	item, ok := c.Next()
	if !ok {
		return false, nil
	}
	task, _ := c.Task(item.Manager)
	version, err := e.stepVersion(ctx, task, item.Phase)
	if err != nil {
		return true, err
	}
	sid := SessionID(e.id("s", task.ID))
	workspace := WorkspaceID(e.id("w", task.ID))
	defer e.cfg.Workspaces.Close(context.WithoutCancel(ctx), workspace)
	materialized, err := e.cfg.Workspaces.Materialize(ctx, WorkspaceSpec{ID: workspace, Tree: version.tree})
	if err != nil || materialized.Uncertain || materialized.Tree != version.tree.Digest() {
		return true, errors.Join(errStep, err)
	}
	role, purpose := stepRole(item.Phase)
	profile, err := e.cfg.Profile(ProfileRequest{Manager: item.Manager, Role: role, Phase: item.Phase, ReadOnly: !item.Phase.writing()})
	if err != nil {
		return true, err
	}
	tools, err := ToolCatalog(item.Phase)
	if err != nil {
		return true, err
	}
	s := &stepRun{task: task.ID, manager: item.Manager, phase: item.Phase, session: sid, version: version, openID: "open-" + string(sid)}
	s.plan = SessionPlan{ID: sid, Role: role, Workspace: materialized.Volume, ProviderHome: item.Home, Resume: item.Resume, ImageID: profile.ImageID, CredentialRef: profile.CredentialRef,
		Candidate: version.tree.Digest(), ReadOnly: !item.Phase.writing(), MaxWireBytes: profile.MaxWireBytes, Profile: profile.Profile.Ref(), Tools: tools.Ref()}
	prepared, err := c.PrepareTaskSessionCommand(e.meta("prepare-session"), item, s.plan, Reservation{ID: s.openID, Kind: SessionStart, Purpose: purpose, Maximum: e.cfg.Limits.StartTimeout}, profile.Profile, tools)
	if err != nil {
		return true, err
	}
	if err := e.commit(ctx, prepared); err != nil {
		return true, err
	}
	if err := e.drive(ctx, s, purpose, workspace); err != nil {
		if abandonErr := e.abandon(ctx, s, err); abandonErr != nil {
			return true, errors.Join(err, abandonErr)
		}
		if !s.opened {
			return true, err // nothing was charged, so a retry would cost nothing
		}
		return true, stepFailure{err}
	}
	return true, nil
}

// drive runs one prepared session through its single step (core rule S1).
func (e *Engine) drive(ctx context.Context, s *stepRun, purpose Purpose, workspace WorkspaceID) error {
	c := e.cfg.Coordinator
	channel, err := c.SessionChannel(e.cfg.Journal, s.session, e.tools(s))
	if err != nil {
		return err
	}
	s.channel = channel
	admission, err := c.SessionAdmission(ctx, e.cfg.Journal, s.session)
	if err != nil {
		return err
	}
	s.began = e.cfg.Clock()
	s.opened = true
	opened, err := e.cfg.Sessions.Open(ctx, admission, channel)
	s.openEnded = e.cfg.Clock()
	if err != nil || opened.Uncertain {
		return errors.Join(errStep, err)
	}
	if err := e.settle(ctx, s.openID, s.began); err != nil {
		return err
	}
	s.openSettled = true
	quoted, err := e.quotes(ctx, s.manager)
	if err != nil {
		return err
	}
	s.stepID = StepID(e.id("step", s.task))
	s.stepInvoke = "step-" + string(s.stepID)
	prepStep, err := c.PrepareTaskStepCommand(e.meta("prepare-step"), StepPlan{Session: s.session, ID: s.stepID, SessionBinding: s.plan.Binding()}, Reservation{ID: s.stepInvoke, Kind: AgentStep, Purpose: purpose, Maximum: c.StepTimeout(s.manager)},
		e.cfg.Preparation, StepFacts{Tree: s.version.tree, Registry: s.version.registry, Quoted: quoted})
	if err != nil {
		return err
	}
	if err := e.commit(ctx, prepStep); err != nil {
		return err
	}
	s.stepPrepared = true
	stepAdmission, err := c.StepAdmission(ctx, e.cfg.Journal, s.stepID)
	if err != nil {
		return err
	}
	s.stepBegan = e.cfg.Clock()
	s.dispatched = true
	// The Host ends the step before its reservation runs out, so that the
	// adapter's cleanup stays inside it; the step's records are written even
	// after its deadline (core rule S2).
	deadline := c.StepTimeout(s.manager) - e.cfg.Limits.StopTimeout
	if deadline <= 0 {
		deadline = c.StepTimeout(s.manager) / 2
	}
	s.deadline = s.stepBegan.Add(deadline)
	stepCtx, cancel := context.WithTimeout(ctx, deadline)
	observed, err := e.cfg.Sessions.Step(stepCtx, stepAdmission, detached{channel})
	cancel()
	s.stepEnded = e.cfg.Clock()
	if err != nil || observed.Uncertain || !observed.Terminal {
		return errors.Join(errStep, err)
	}
	s.stepTerminal = true
	// The review checks the step requested are charged as their own
	// invocations, so the step's charge leaves their time out.
	if err := e.settle(ctx, s.stepInvoke, s.stepBegan.Add(s.checked)); err != nil {
		return err
	}
	s.stepSettled = true
	if err := e.stop(ctx, s.session, channel, "the step ended"); err != nil {
		return err
	}
	harvest, err := e.cfg.Workspaces.Harvest(ctx, HarvestRequest{ID: workspace, MaxEntries: e.cfg.Limits.HarvestEntries, MaxBytes: e.cfg.Limits.HarvestBytes})
	if err != nil || !harvest.Complete() {
		return errors.Join(errStep, err)
	}
	return e.conclude(ctx, s, harvest.Tree)
}

// detached records an adapter's observations even after the step's deadline
// cancelled its provider work.
type detached struct{ io SessionIO }

func (d detached) Ready(ctx context.Context, r SessionReadiness) (AppendReceipt, error) {
	return d.io.Ready(context.WithoutCancel(ctx), r)
}
func (d detached) Bind(ctx context.Context, o SessionObservation) (AppendReceipt, error) {
	return d.io.Bind(context.WithoutCancel(ctx), o)
}
func (d detached) PrepareSend(ctx context.Context, p TransportPreparation) (SendPermit, error) {
	return d.io.PrepareSend(context.WithoutCancel(ctx), p)
}
func (d detached) ObserveSend(ctx context.Context, o TransportObservation) (AppendReceipt, error) {
	return d.io.ObserveSend(context.WithoutCancel(ctx), o)
}
func (d detached) Call(ctx context.Context, call ToolCall) (ToolResult, error) {
	return d.io.Call(context.WithoutCancel(ctx), call)
}
func (d detached) ObserveStep(ctx context.Context, o StepObservation) (AppendReceipt, error) {
	return d.io.ObserveStep(context.WithoutCancel(ctx), o)
}
func (d detached) ObserveStop(ctx context.Context, o StopObservation) (AppendReceipt, error) {
	return d.io.ObserveStop(context.WithoutCancel(ctx), o)
}

func (e *Engine) stop(ctx context.Context, sid SessionID, channel SessionIO, reason string) error {
	request, err := e.cfg.Coordinator.StopRequest(sid, reason)
	if err != nil {
		return err
	}
	request.Timeout = e.cfg.Limits.StopTimeout
	observed, err := e.cfg.Sessions.Stop(ctx, request, channel)
	if err != nil || !observed.Terminated {
		return errors.Join(errStep, err)
	}
	return nil
}

// abandon ends a step the Host could not finish. It stops the container; a
// confirmed stop ends all activity in it (core rule S2), so the start and the
// step are settled as observed, an unstarted one is released, and a run that
// an uncertain observation moved to Reconciling resumes. Without a confirmed
// stop the effects are marked uncertain and stay for reconciliation.
// Otherwise the step concludes as failed, as a fault of the Host.
func (e *Engine) abandon(ctx context.Context, s *stepRun, cause error) error {
	ctx = context.WithoutCancel(ctx)
	c := e.cfg.Coordinator
	if s.channel == nil {
		channel, err := c.SessionChannel(e.cfg.Journal, s.session, RefuseTools)
		if err != nil {
			return err
		}
		s.channel = channel
	}
	var errs []error
	if v, _ := c.Session(s.session); !v.Terminated {
		if !s.opened {
			// Nothing was started: the Host's own observation ends the session.
			_, err := s.channel.ObserveStop(ctx, StopObservation{RecordMeta: e.meta("abandoned"), Terminated: true, Reason: "the Host never opened the session"})
			errs = append(errs, err)
		} else {
			errs = append(errs, e.stop(ctx, s.session, s.channel, "the Host ended the step early"))
		}
	}
	v, _ := c.Session(s.session)
	stopped := v.Terminated && !v.Uncertain
	// An invocation is charged until its port call returned; stopping the
	// container afterwards is Host control work.
	settle := func(kind, invocation, reason string, began, ended time.Time) {
		cmd := Command{ID: fmt.Sprintf("%s-%s", kind, invocation), ObservedAt: e.cfg.Clock().UTC(), Kind: kind, Assignment: c.Current().Assignment, Run: c.Current().ID, Invocation: invocation, Reason: reason}
		if kind == SettleInvocation {
			cmd.Elapsed = ended.Sub(began)
		}
		errs = append(errs, e.commit(ctx, cmd))
	}
	switch {
	case !s.opened:
		settle(ReleaseUndispatched, s.openID, "the Host never opened the session", time.Time{}, time.Time{})
	case !s.openSettled && stopped:
		settle(SettleInvocation, s.openID, "the Host stopped the container after a failed start", s.began, s.openEnded)
	case !s.openSettled:
		settle(ObserveUncertain, s.openID, "the session's start is not known", time.Time{}, time.Time{})
	}
	switch {
	case s.stepPrepared && !s.dispatched:
		settle(ReleaseUndispatched, s.stepInvoke, "the Host never dispatched the step", time.Time{}, time.Time{})
	case s.dispatched && !s.stepSettled && stopped:
		settle(SettleInvocation, s.stepInvoke, "the Host stopped the container before the step ended", s.stepBegan.Add(s.checked), s.stepEnded)
	case s.dispatched && !s.stepSettled:
		settle(ObserveUncertain, s.stepInvoke, "the step's end is not known", time.Time{}, time.Time{})
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if r := c.Current(); r.State == Reconciling && !r.StopRequested && stopped {
		if c.Budget().Overrun {
			// Every effect ended, but the run cannot continue (core rule T14).
			return e.commit(ctx, Command{ID: e.meta("finish").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: FinishRun, Assignment: r.Assignment, Run: r.ID, Outcome: Incomplete,
				Reason: "the failed step overran the budget"})
		}
		resume := Command{ID: e.meta("resume").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: ResumeRun, Assignment: r.Assignment, Run: r.ID}
		if err := e.commit(ctx, resume); err != nil {
			return err
		}
	}
	cmd, err := c.ConcludeStepCommand(e.meta("conclude"), s.manager, StepResult{Aggregate: c.AggregateVersion(), Failure: "the Host could not finish the step: " + cause.Error(), HostFault: true})
	if err != nil {
		return err
	}
	return e.commit(ctx, cmd)
}

// recover concludes a step that an earlier error or process left bound to its
// task, once its container is stopped and its effects have ended. A step whose
// effects are still open needs reconciliation first.
func (e *Engine) recover(ctx context.Context) error {
	c := e.cfg.Coordinator
	if c.Current().State != Running {
		return nil // a finished or waiting run admits nothing to recover for
	}
	for _, t := range c.Tasks() {
		if t.Session == "" {
			continue
		}
		if v, _ := c.Session(t.Session); !v.Terminated {
			channel, err := c.SessionChannel(e.cfg.Journal, t.Session, RefuseTools)
			if err != nil {
				return err
			}
			if err := e.stop(ctx, t.Session, channel, "the Host recovers the step"); err != nil {
				return err
			}
		}
		cmd, err := c.ConcludeStepCommand(e.meta("conclude"), t.Manager, StepResult{Aggregate: c.AggregateVersion(), Failure: "the Host lost the step before its conclusion", HostFault: true})
		if err == nil {
			err = e.commit(ctx, cmd)
		}
		if err != nil {
			return fmt.Errorf("the step of %s needs reconciliation: %w", t.Manager, err)
		}
	}
	return nil
}

// stepVersion builds the step's workspace content (core rules D3, T7): the
// current aggregate plus the Manager's own unaccepted changes, nothing for the
// expectations review, and the frozen candidate for the candidate review.
func (e *Engine) stepVersion(ctx context.Context, task TaskView, phase Phase) (stepVersion, error) {
	switch phase {
	case PhaseReviewerExpectations:
		tree, err := snapshot.NewTree(nil)
		return stepVersion{tree: tree}, err
	case PhaseReviewerCandidate:
		if task.Submission.Candidate == (BlobRef{}) {
			if task.Submission.Aggregate >= uint64(len(e.versions)) {
				return stepVersion{}, ErrState
			}
			a := e.versions[task.Submission.Aggregate]
			return stepVersion{a.Tree(), a.Registry()}, nil
		}
		candidate, err := e.candidate(ctx, task.Submission.Candidate)
		if err != nil {
			return stepVersion{}, err
		}
		return stepVersion{candidate.Tree(), candidate.Registry()}, nil
	}
	a := e.aggregate()
	if task.Own != (BlobRef{}) {
		own, err := e.candidate(ctx, task.Own)
		if err != nil {
			return stepVersion{}, err
		}
		replayed, ok := realization.ReplayDelivery(a, own).Aggregate()
		if !ok {
			return stepVersion{}, ErrState
		}
		a = replayed
	}
	return stepVersion{a.Tree(), a.Registry()}, nil
}

func (e *Engine) candidate(ctx context.Context, ref BlobRef) (realization.Candidate, error) {
	p, err := e.cfg.Journal.ReadBlob(ctx, ref)
	if err != nil || p.Ref() != ref {
		return realization.Candidate{}, ErrObservationLimit
	}
	record, err := realization.DecodeCandidateRecord(p.Bytes(), e.cfg.Limits.Record)
	if err != nil {
		return realization.Candidate{}, err
	}
	return record.Observation().Candidate(), nil
}

func (e *Engine) quotes(ctx context.Context, manager string) ([]Payload, error) {
	var out []Payload
	for _, ref := range e.cfg.Coordinator.StepQuotes(manager) {
		p, err := e.cfg.Journal.ReadBlob(ctx, ref)
		if err != nil || p.Ref() != ref {
			return nil, ErrObservationLimit
		}
		out = append(out, p)
	}
	return out, nil
}

// conclude records the step's S4 result and its outcome (core rules S4, T5,
// T6, T8). A read-only step that changed files, or a write outside the
// owner's claims, ends the step as failed, and a failed step's reservations
// and registry proposals are undone. A candidate review that passes composes
// the delivery onto the aggregate first (core rule D2); a refused composition
// is a Host error that finishes the run Failed.
func (e *Engine) conclude(ctx context.Context, s *stepRun, observed snapshot.Tree) error {
	c := e.cfg.Coordinator
	task, _ := c.Task(s.manager)
	result := StepResult{Aggregate: c.AggregateVersion()}
	if ref, ok := c.OutcomeInput(s.manager); ok {
		p, err := e.cfg.Journal.ReadBlob(ctx, ref)
		if err != nil || p.Ref() != ref {
			return ErrObservationLimit
		}
		result.Input = p
	}
	var composed realization.Aggregate
	switch {
	case !s.phase.writing() && !observed.Equal(s.version.tree):
		result.Failure = "a read-only step changed its workspace"
	case s.phase.writing() && (!observed.Equal(s.version.tree) || e.proposes(s)):
		// Registry proposals change the candidate even without file changes.
		record, err := e.capture(s, observed)
		if err != nil {
			result.Failure = "the step's changes are outside its owner's claims: " + err.Error()
			break
		}
		p, err := NewPayload(candidateRecordMedia, "utf-8", record.Bytes())
		if err != nil {
			return err
		}
		result.Candidate, result.Tree = p, observed.Digest()
	case s.phase == PhaseReviewerCandidate && result.Input.Valid() && task.Submission.Candidate != (BlobRef{}):
		verdict, err := c.ReviewVerdict(s.manager, result.Input)
		if err != nil || verdict != VerdictAccept {
			break
		}
		candidate, err := e.candidate(ctx, task.Submission.Candidate)
		if err != nil {
			return err
		}
		replay := realization.ReplayDelivery(e.aggregate(), candidate)
		next, ok := replay.Aggregate()
		if !ok {
			reason := "the accepted delivery of " + s.manager + " cannot be composed: " + findingText(replay.Findings())
			finish := Command{ID: e.meta("finish").ID, ObservedAt: e.cfg.Clock().UTC(), Kind: FinishRun, Assignment: c.Current().Assignment, Run: c.Current().ID, Outcome: Failed, Reason: reason}
			return errors.Join(errors.New(reason), e.commit(ctx, finish))
		}
		composed, result.Aggregate, result.AggregateTree = next, next.Version(), next.Tree().Digest()
	}
	cmd, err := c.ConcludeStepCommand(e.meta("conclude"), s.manager, result)
	if err != nil {
		return err
	}
	if err := e.commit(ctx, cmd); err != nil {
		return err
	}
	if composed.Valid() {
		e.versions = append(e.versions, composed)
	}
	return nil
}

// capture checks the observed tree against the current aggregate and the
// task's claims and proposed registry records (core rule S4).
func (e *Engine) capture(s *stepRun, observed snapshot.Tree) (realization.CandidateRecord, error) {
	owner, err := managerOwner(s.manager)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	base := e.aggregate().CandidateBase()
	claims, err := e.freshClaims(s)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	proposed, err := e.proposals(s.manager, s)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	records := map[snapshot.Path]realization.Registration{}
	for _, r := range base.Registry().Entries() {
		records[r.Artifact().Path()] = r
	}
	for path, r := range proposed {
		records[path] = r
	}
	var entries []realization.Registration
	for _, r := range records {
		entries = append(entries, r)
	}
	registry, err := realization.NewRegistry(entries)
	if err != nil {
		return realization.CandidateRecord{}, err
	}
	result := realization.CaptureCandidate(base, owner, claims, observed, registry)
	candidate, ok := result.Candidate()
	if !ok {
		return realization.CandidateRecord{}, errors.New(findingText(result.Findings()))
	}
	return realization.NewCandidateRecord(e.compiled, candidate, nil, nil, e.cfg.Limits.Record)
}

// tools answers the Host tools of one step. Outcome tools are accepted once
// per step, after the Coordinator's own check; reservations and registry
// proposals take effect for the task; read supplies Definitions and the
// step's files. Tools the core does not support yet are refused with a reason.
func (e *Engine) tools(s *stepRun) ToolHandler {
	manager, phase, version := s.manager, s.phase, s.version
	answered := false
	return func(ctx context.Context, call ToolCall, _ int64) (ToolOutcome, error) {
		refuse := func(reason string) (ToolOutcome, error) { return ToolOutcome{Status: ToolRefused, Reason: reason}, nil }
		ok := func(result any) (ToolOutcome, error) {
			data, err := json.Marshal(result)
			if err != nil {
				return refuse("the result cannot be encoded")
			}
			return ToolOutcome{Status: ToolOK, Result: data}, nil
		}
		if !PhaseTool(phase, call.Tool) {
			return refuse("this tool is not offered in the " + string(phase) + " phase")
		}
		if call.Input.Redacted {
			return refuse("redacted tool input cannot be evaluated")
		}
		args, err := CallArguments(call.Tool, call.Input.Journal.Bytes())
		if err != nil {
			return refuse(err.Error())
		}
		if call.Tool == ToolSend {
			// Messages take effect with the step's conclusion; a reply step
			// ends with its one reply (core rules M1 to M5, T11).
			if phase == PhaseReply && answered {
				return refuse("a reply step sends one reply")
			}
			id, err := e.cfg.Coordinator.CheckSend(manager, args)
			if err != nil {
				return refuse(err.Error())
			}
			answered = answered || phase == PhaseReply
			return ok(map[string]string{"id": id})
		}
		if OutcomeTool(phase, call.Tool) {
			if answered {
				return refuse("this step already has an outcome")
			}
			if err := e.cfg.Coordinator.CheckOutcome(manager, call.Tool, args); err != nil {
				return refuse(err.Error())
			}
			if phase.writing() {
				if reason := e.formMismatch(s); reason != "" {
					return refuse(reason)
				}
			}
			answered = true
			return ok(map[string]bool{"accepted": true})
		}
		switch call.Tool {
		case ToolRequestCheck:
			return e.reviewCheck(ctx, s, call, args)
		case ToolRead:
			var in struct {
				Ref string `json:"ref"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			return e.read(phase, version, in.Ref)
		case ToolReservePath:
			var in struct {
				Path string `json:"path"`
				Form string `json:"form"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			claim := Claim{Tool: ToolReservePath, Path: in.Path, Form: in.Form}
			request, err := requestOf(manager, claim)
			if err != nil {
				return refuse(err.Error())
			}
			// Reserving a path again changes the form of one's own reservation.
			claims, err := e.freshClaimsWithout(s, in.Path)
			if err != nil {
				return refuse(err.Error())
			}
			if _, decision := realization.ReserveNewPath(e.aggregate().PathFacts(), claims, request); decision.Status() != realization.PathAdmissible {
				return refuse(findingText(decision.Findings()) + e.holderHint(s, in.Path))
			}
			s.pending = addClaims(s.pending, []Claim{claim})
			return ok(map[string]bool{"reserved": true})
		case ToolRegister:
			var in struct {
				Path     string   `json:"path"`
				Form     string   `json:"form"`
				Realizes []string `json:"realizes"`
				Why      string   `json:"why"`
			}
			if err := decodeArgs(args, &in); err != nil {
				return refuse(err.Error())
			}
			claim := Claim{Tool: ToolRegister, Path: in.Path, Form: in.Form, Realizes: in.Realizes, Why: in.Why}
			record, err := registrationOf(manager, claim)
			if err != nil {
				return refuse(err.Error())
			}
			// A new file's record carries the form of its reservation (core rule S4).
			if _, registered := e.aggregate().Registry().Lookup(record.Artifact().Path()); !registered {
				if form, reserved := e.reservedForm(s, in.Path); reserved && form != in.Form {
					return refuse(fmt.Sprintf("%s is reserved with form %s; register it with that form, or reserve it again with form %s first", in.Path, form, in.Form))
				}
			}
			if err := e.registrable(s, record); err != nil {
				return refuse(err.Error())
			}
			s.pending = addClaims(s.pending, []Claim{claim})
			return ok(map[string]bool{"registered": true})
		}
		return refuse("the " + call.Tool + " tool is not available in this version of the core")
	}
}

// reviewCheck runs a configured check that a candidate review requests, within
// the step's remaining time, on the
// frozen candidate it reviews, and answers with the receipt's outcome and the
// end of its output, which the review cites as evidence by the check's ID.
func (e *Engine) reviewCheck(ctx context.Context, s *stepRun, call ToolCall, args []byte) (ToolOutcome, error) {
	refuse := func(reason string) (ToolOutcome, error) { return ToolOutcome{Status: ToolRefused, Reason: reason}, nil }
	if s.phase != PhaseReviewerCandidate {
		return refuse("checks run on a submitted candidate during its review")
	}
	var in struct {
		Check string `json:"check"`
	}
	if err := decodeArgs(args, &in); err != nil {
		return refuse(err.Error())
	}
	req, ok := e.cfg.ReviewChecks[in.Check]
	if !ok || e.cfg.Checks == nil {
		names := slices.Sorted(maps.Keys(e.cfg.ReviewChecks))
		return refuse(fmt.Sprintf("%q is not a configured check; the configured checks are %s", in.Check, strings.Join(names, ", ")))
	}
	req.Tree, req.ID = s.version.tree, CheckID(fmt.Sprintf("%s-%s-%s", req.ID, call.Step, call.Call))
	reservation := e.checkReservation(req)
	if left := s.deadline.Sub(e.cfg.Clock()); reservation.Maximum > left {
		return refuse(fmt.Sprintf("the check needs up to %s, and the step has %s left", reservation.Maximum, left.Round(time.Second)))
	}
	admit, err := e.cfg.Coordinator.PrepareReviewCheckCommand(e.meta("prepare-check"), s.manager, in.Check, req, reservation)
	if err != nil {
		return refuse(err.Error())
	}
	receipt, elapsed, err := e.runCheck(ctx, admit, req)
	s.checked += elapsed
	if errors.Is(err, errNotAdmitted) {
		return refuse(err.Error())
	}
	if err != nil {
		return ToolOutcome{}, err
	}
	output := receipt.Output
	if len(output) > reviewCheckOutput {
		output = output[len(output)-reviewCheckOutput:]
	}
	data, err := json.Marshal(map[string]any{"check": string(req.ID), "passed": receipt.Passed(req), "exitCode": receipt.ExitCode, "timedOut": receipt.TimedOut, "outputTail": string(output), "reason": receipt.Reason})
	if err != nil {
		return refuse("the receipt cannot be encoded")
	}
	return ToolOutcome{Status: ToolOK, Result: data}, nil
}

const reviewCheckOutput = 16 << 10

// registrable checks a proposed registry record before it is accepted: its
// Definitions exist in the compiled model, and its path is registered to the
// Manager or reserved by it.
func (e *Engine) registrable(s *stepRun, record realization.Registration) error {
	for _, id := range record.Artifact().Basis() {
		if !e.defined(id) {
			return fmt.Errorf("Definition %s/%s does not exist in the model", id.Package, id.ID)
		}
	}
	path := record.Artifact().Path()
	if current, ok := e.aggregate().Registry().Lookup(path); ok {
		if current.Owner() != record.Owner() {
			return fmt.Errorf("%s is registered to the Manager %s; you may use it but not change it", path, ownerManager(current.Owner()))
		}
		return nil
	}
	if !slices.ContainsFunc(e.claimsOf(s.manager, s), func(c Claim) bool { return c.Tool == ToolReservePath && c.Path == path.String() }) {
		return fmt.Errorf("%s is neither registered to you nor reserved by you; reserve it first", path)
	}
	return nil
}

// proposes reports whether the step's Manager has a registry proposal that
// differs from the aggregate.
func (e *Engine) proposes(s *stepRun) bool {
	proposed, err := e.proposals(s.manager, s)
	return err != nil || len(proposed) > 0
}

func (e *Engine) defined(id model.Identity) bool {
	for _, m := range e.compiled.Models {
		for _, d := range m.Definitions {
			if d.Identity == id {
				return true
			}
		}
	}
	return false
}

// read returns a Definition as package/id or a file of the step's version.
// The expectations review reads Definitions only (core rule T7).
func (e *Engine) read(phase Phase, version stepVersion, ref string) (ToolOutcome, error) {
	refuse := func(reason string) (ToolOutcome, error) { return ToolOutcome{Status: ToolRefused, Reason: reason}, nil }
	if id, ok := definitionIdentity(ref); ok && e.defined(id) {
		for _, m := range e.compiled.Models {
			for _, d := range m.Definitions {
				if d.Identity == id {
					data, err := json.Marshal(map[string]any{"definition": ref, "kind": d.Kind.Package + "/" + d.Kind.ID, "title": d.Title, "body": d.Body, "spec": d.Spec})
					if err != nil {
						return refuse("the Definition cannot be encoded")
					}
					return ToolOutcome{Status: ToolOK, Result: data}, nil
				}
			}
		}
	}
	if phase == PhaseReviewerExpectations {
		return refuse("before the candidate, read supplies Definitions only")
	}
	path, err := snapshot.NewPath(ref)
	if err != nil {
		return refuse("ref names neither a Definition as package/id nor a file path")
	}
	entry, found := version.tree.Lookup(path)
	if !found {
		return refuse("no Definition or file " + ref + " exists in this step's version")
	}
	result := map[string]any{"path": ref, "contentId": entry.ContentID().String(), "bytes": entry.ByteLength()}
	if utf8.Valid(entry.Bytes()) {
		result["text"] = string(entry.Bytes())
	}
	data, err := json.Marshal(result)
	if err != nil {
		return refuse("the file cannot be encoded")
	}
	return ToolOutcome{Status: ToolOK, Result: data}, nil
}

func findingText(findings []realization.Finding) string {
	var parts []string
	for _, f := range findings {
		parts = append(parts, f.Code+": "+f.Message)
	}
	return strings.Join(parts, "; ")
}

func definitionIdentity(s string) (model.Identity, bool) {
	i := strings.LastIndex(s, "/")
	if i <= 0 || i == len(s)-1 {
		return model.Identity{}, false
	}
	return model.Identity{Package: s[:i], ID: s[i+1:]}, true
}

func (e *Engine) id(kind, task string) string {
	return fmt.Sprintf("%s-%s-%d", kind, task[len(task)-12:], e.cfg.Coordinator.Revision())
}

func (e *Engine) meta(kind string) RecordMeta {
	return RecordMeta{ID: fmt.Sprintf("%s-%d", kind, e.cfg.Coordinator.Revision()), ObservedAt: e.cfg.Clock().UTC()}
}

func (e *Engine) settle(ctx context.Context, invocation string, began time.Time) error {
	c := Command{ID: fmt.Sprintf("settle-%s", invocation), ObservedAt: e.cfg.Clock().UTC(), Kind: SettleInvocation, Assignment: e.cfg.Coordinator.Current().Assignment, Run: e.cfg.Coordinator.Current().ID,
		Invocation: invocation, Elapsed: e.cfg.Clock().Sub(began), Reason: "Host observed the invocation's end and elapsed duration"}
	return e.commit(ctx, c)
}

func (e *Engine) commit(ctx context.Context, cmd Command) error {
	p, err := e.cfg.Coordinator.Propose(cmd)
	if err != nil {
		return err
	}
	_, err = e.cfg.Coordinator.Commit(ctx, e.cfg.Journal, p)
	return err
}
