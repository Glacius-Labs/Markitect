package apply

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionPorts are the adapters a controlled run executes through. The Host
// composes them; apply chooses no provider, credential or image.
type ExecutionPorts struct {
	Sessions   run.Sessions
	Workspaces run.Workspaces
	Checks     run.Checks
	Publisher  Publisher
	// Profile returns the session profile of one step; RoleProfile names the
	// prepared profile it is built from.
	Profile func(run.ProfileRequest) (run.SessionProfile, error)
	// CheckRequest builds the request of one configured check from its check
	// profile; the tree is filled in by run.
	CheckRequest func(CheckSelection) (run.CheckRequest, error)
	Clock        func() time.Time
	Limits       run.EngineLimits
}

func (p ExecutionPorts) valid() bool {
	return p.Sessions != nil && p.Workspaces != nil && p.Checks != nil && p.Publisher != nil && p.Profile != nil && p.CheckRequest != nil && p.Clock != nil
}

// ExecutionResult is the run's state when Execute returned.
type ExecutionResult struct {
	Run         run.RunView
	Tasks       []run.TaskView
	Issues      []run.Issue
	Checks      []run.CheckView
	Publication *PublicationReceipt
}

// Execute starts the bound run when no run is active and drives it: the
// Managers work until the root is done, the required checks run on the
// aggregate, the source is preflighted and the verified aggregate is
// published into the working tree. It returns when the run finished, waits
// for the owner, or needs reconciliation. A run that already finished on this
// basis is not started again.
func (c *ControlledRun) Execute(ctx context.Context, ports ExecutionPorts) (ExecutionResult, error) {
	result := ExecutionResult{}
	if err := c.Check(ctx); err != nil {
		return result, err
	}
	if !ports.valid() {
		return result, run.ErrInvalid
	}
	co := c.coordinator
	if co.RecoveryRequired() {
		return c.result(nil), fmt.Errorf("%w: the journal holds effects that need reconciliation first", run.ErrRecovery)
	}
	if err := c.start(ctx, ports.Clock); err != nil {
		return c.result(nil), err
	}
	preparation, err := c.prepared.stepPreparation(c.facts)
	if err != nil {
		return c.result(nil), err
	}
	reviewChecks, err := c.reviewChecks(ports)
	if err != nil {
		return c.result(nil), err
	}
	engine, err := run.NewEngine(ctx, run.EngineConfig{Coordinator: co, Journal: c.Journal(), Sessions: ports.Sessions, Workspaces: ports.Workspaces, Checks: ports.Checks,
		ReviewChecks: reviewChecks, Preparation: preparation, Profile: ports.Profile, Clock: ports.Clock, Limits: ports.Limits})
	if err != nil {
		return c.result(nil), err
	}
	if r := co.Current(); co.Resumable() {
		// The owner answered; the run continues (core rules W1, W2).
		if err := c.commit(ctx, run.Command{ID: c.meta("resume", ports.Clock).ID, Kind: run.ResumeRun, Assignment: r.Assignment, Run: r.ID, ObservedAt: ports.Clock().UTC()}); err != nil {
			return c.result(nil), err
		}
	}
	if co.Current().State == run.Running {
		if err := engine.Run(ctx); err != nil {
			return c.result(nil), err
		}
	}
	if co.Current().State == run.Verifying {
		if err := c.verify(ctx, engine, ports); err != nil {
			return c.result(nil), err
		}
	}
	if co.Current().State == run.Publishing {
		receipt, err := c.publish(ctx, engine, ports)
		return c.result(receipt), err
	}
	return c.result(nil), nil
}

// Answer records the owner's answer to an open question to the owner (core
// rule W2). The next Execute continues the run.
func (c *ControlledRun) Answer(ctx context.Context, question, answer string, clock func() time.Time) error {
	if err := c.Check(ctx); err != nil {
		return err
	}
	cmd, err := c.coordinator.AnswerCommand(c.meta("answer", clock), question, answer)
	if err != nil {
		return err
	}
	return c.commit(ctx, cmd)
}

// RoleProfile returns the prepared Agent profile a Manager's role runs with,
// from which the Host composes the session profile of its adapter.
func (c *ControlledRun) RoleProfile(manager, role string) (PreparedProfile, error) {
	a, ok := c.prepared.Prepared()
	settings, settled := a.Settings().Settings()
	if !ok || !settled {
		return PreparedProfile{}, fmt.Errorf("%w: resolved settings required", run.ErrInvalid)
	}
	m, found := settings.Manager(manager)
	if !found {
		return PreparedProfile{}, fmt.Errorf("%w: no Manager %q", run.ErrInvalid, manager)
	}
	selected := m.Manager.Profile.Value
	if role == "reviewer" {
		selected = m.Reviewer.Profile.Value
	}
	for _, p := range a.Settings().ProfileObservations() {
		if p.Input().Namespace() == "agentProfiles" && p.Input().Name() == selected.Name {
			return p, nil
		}
	}
	return PreparedProfile{}, fmt.Errorf("%w: no prepared Agent profile %q", run.ErrInvalid, selected.Name)
}

func (c *ControlledRun) result(receipt *PublicationReceipt) ExecutionResult {
	r := ExecutionResult{Run: c.coordinator.Current(), Tasks: c.coordinator.Tasks(), Checks: c.coordinator.Checks(), Publication: receipt}
	for _, t := range r.Tasks {
		r.Issues = append(r.Issues, c.coordinator.Issues(t.Manager)...)
	}
	return r
}

func (c *ControlledRun) commit(ctx context.Context, cmd run.Command) error {
	p, err := c.coordinator.Propose(cmd)
	if err != nil {
		return err
	}
	_, err = c.coordinator.Commit(ctx, c.Journal(), p)
	return err
}

func (c *ControlledRun) meta(kind string, clock func() time.Time) run.RecordMeta {
	return run.RecordMeta{ID: fmt.Sprintf("%s-%d", kind, c.coordinator.Revision()), ObservedAt: clock().UTC()}
}

// start prepares and starts the bound run when none is active. An active run
// bound to another capture finishes Stale first, because its source changed
// (execution contract, outcomes). A finished run continues its assignment on
// a changed basis; after Applied or NoChange, the assignment is closed and a
// new one opens.
func (c *ControlledRun) start(ctx context.Context, clock func() time.Time) error {
	co := c.coordinator
	current := co.Current()
	if current.ID != "" && current.State != run.Finished && current.Basis != c.basis.Ref() {
		if err := c.finish(ctx, run.Stale, "the captured source changed since the run was prepared", clock); err != nil {
			return fmt.Errorf("the run %s is bound to another capture and cannot finish yet: %w", current.ID, err)
		}
		current = co.Current()
	}
	switch current.State {
	case run.Prepared:
	case "", run.Finished:
		if current.ID != "" && current.Basis == c.basis.Ref() && (current.Outcome == run.Applied || current.Outcome == run.NoChange) {
			return fmt.Errorf("%w: the run %s already finished %s on this basis", run.ErrState, current.ID, current.Outcome)
		}
		assignments := co.Assignments()
		closed := len(assignments) > 0 && assignments[len(assignments)-1].Closure.CommandID != ""
		if current.ID != "" && !closed && (current.Outcome == run.Applied || current.Outcome == run.NoChange) {
			closure := run.Command{ID: c.meta("close", clock).ID, Kind: run.CloseAssignment, Assignment: current.Assignment, Run: current.ID, ObservedAt: clock().UTC(),
				OwnerInput: "The owner requested a new Apply after the assignment finished " + string(current.Outcome) + ".", Quiescence: "No session, step, check or invocation is open."}
			if err := c.commit(ctx, closure); err != nil {
				return err
			}
			closed = true
		}
		assignment, predecessor := fmt.Sprintf("assignment-%d", co.Revision()), ""
		if current.ID != "" && !closed {
			assignment, predecessor = current.Assignment, current.ID
		}
		prepare := run.PrepareCommand(c.meta("prepare", clock).ID, assignment, fmt.Sprintf("run-%d", co.Revision()), predecessor, clock().UTC(), c.basis)
		if err := c.commit(ctx, prepare); err != nil {
			if predecessor != "" {
				return fmt.Errorf("the run %s finished %s and cannot continue on this basis: %w", current.ID, current.Outcome, err)
			}
			return err
		}
	default:
		return nil
	}
	current = co.Current()
	return c.commit(ctx, run.Command{ID: c.meta("start", clock).ID, Kind: run.StartRun, Assignment: current.Assignment, Run: current.ID, ObservedAt: clock().UTC()})
}

func (c *ControlledRun) finish(ctx context.Context, outcome run.Outcome, reason string, clock func() time.Time) error {
	r := c.coordinator.Current()
	return c.commit(ctx, run.Command{ID: c.meta("finish", clock).ID, Kind: run.FinishRun, Assignment: r.Assignment, Run: r.ID, ObservedAt: clock().UTC(), Outcome: outcome, Reason: reason})
}

// reviewChecks are the configured checks, by name, that candidate reviews may
// request.
func (c *ControlledRun) reviewChecks(ports ExecutionPorts) (map[string]run.CheckRequest, error) {
	a, _ := c.prepared.Prepared()
	selections, ok := a.Settings().Checks()
	if !ok {
		return nil, fmt.Errorf("%w: the check selections are incomplete", run.ErrInvalid)
	}
	out := map[string]run.CheckRequest{}
	for _, selection := range selections {
		request, err := ports.CheckRequest(selection)
		if err != nil {
			return nil, err
		}
		out[selection.Declaration().ID.Value] = request
	}
	return out, nil
}

// verify runs the required checks on the tree that publishing the aggregate
// writes, preflights its publication and records the final evidence check
// (execution contract, Verifying). A source that changed finishes the run
// Stale, and a change set the target cannot represent finishes it Failed;
// any other preflight error leaves the run Verifying for a later attempt.
func (c *ControlledRun) verify(ctx context.Context, engine *run.Engine, ports ExecutionPorts) error {
	a, _ := c.prepared.Prepared()
	selections, ok := a.Settings().Checks()
	if !ok {
		return fmt.Errorf("%w: the check selections are incomplete", run.ErrInvalid)
	}
	requiredNames := c.coordinator.RequiredChecks()
	var required []run.RequiredCheck
	for _, selection := range selections {
		declaration := selection.Declaration()
		if !slices.Contains(requiredNames, declaration.ID.Value) {
			continue
		}
		request, err := ports.CheckRequest(selection)
		if err != nil {
			return err
		}
		required = append(required, run.RequiredCheck{Name: declaration.ID.Value, Request: request})
	}
	target, err := c.publicationTarget(engine)
	if err != nil {
		return err
	}
	passed, err := engine.Verify(ctx, target, required)
	if err != nil || c.coordinator.Current().State != run.Verifying {
		return err
	}
	if _, err := ports.Publisher.Preflight(ctx, c.publicationRequest(target), c); err != nil {
		switch {
		case errors.Is(err, ErrCaptureChanged):
			return errors.Join(err, c.finish(ctx, run.Stale, "the source changed since its capture: "+err.Error(), ports.Clock))
		case errors.Is(err, ErrUnpublishable):
			return errors.Join(err, c.finish(ctx, run.Failed, "the target cannot represent the change set: "+err.Error(), ports.Clock))
		}
		return err
	}
	return c.commit(ctx, c.coordinator.VerifyRunCommand(c.meta("verify", ports.Clock), target.Digest(), passed))
}

// publicationTarget is the tree that publishing the aggregate writes: its
// files and every registry shard whose records changed, regenerated.
func (c *ControlledRun) publicationTarget(engine *run.Engine) (snapshot.Tree, error) {
	a, _ := c.prepared.Prepared()
	captured, ok := a.Structural().Registry().Registry()
	if !ok {
		return snapshot.Tree{}, fmt.Errorf("%w: the captured registry is incomplete", run.ErrInvalid)
	}
	aggregate := engine.Aggregate()
	return publicationTree(a.Structural().Model().Inputs().Raw.Census, captured, aggregate.Tree(), aggregate.Registry())
}

// publicationRequest is the change set from the captured source to target.
func (c *ControlledRun) publicationRequest(target snapshot.Tree) PublicationRequest {
	a, _ := c.prepared.Prepared()
	model := a.Structural().Model()
	inputs, request := model.Inputs(), model.Request()
	current := c.coordinator.Current()
	return PublicationRequest{ID: PublicationID(fmt.Sprintf("publication-%s", current.ID)), Run: current.ID, Repository: request.Selection.Repository, Guard: inputs.Guard,
		Changes: snapshot.Compare(inputs.Raw.Census, target)}
}

// publish writes exactly the verified tree: it records the change set before
// the first write, publishes it and records the receipt, which finishes the
// run (execution contract, Publishing). A publication that an earlier process
// prepared but never recorded is inspected instead of written again.
func (c *ControlledRun) publish(ctx context.Context, engine *run.Engine, ports ExecutionPorts) (*PublicationReceipt, error) {
	co := c.coordinator
	target, err := c.publicationTarget(engine)
	if err != nil {
		return nil, err
	}
	if target.Digest() != co.Verification().Tree {
		return nil, fmt.Errorf("%w: the publication target differs from the verified tree", run.ErrState)
	}
	request := c.publicationRequest(target)
	var receipt PublicationReceipt
	if prepared := co.Publication(); prepared.ID != "" {
		if prepared.ID != string(request.ID) {
			return nil, fmt.Errorf("%w: the prepared publication is another one", run.ErrState)
		}
		if receipt, err = ports.Publisher.Inspect(ctx, request); err != nil {
			return &receipt, err
		}
	} else {
		// Nothing is journaled or written before the preflight passes, so a
		// failed preflight leaves no publication to reconcile.
		permit, err := ports.Publisher.Preflight(ctx, request, c)
		switch {
		case errors.Is(err, ErrCaptureChanged):
			return nil, errors.Join(err, c.finish(ctx, run.Stale, "the source changed since its capture: "+err.Error(), ports.Clock))
		case errors.Is(err, ErrUnpublishable):
			return nil, errors.Join(err, c.finish(ctx, run.Failed, "the target cannot represent the change set: "+err.Error(), ports.Clock))
		case err != nil:
			return nil, err // nothing was journaled; the next Execute publishes again
		}
		cmd, err := co.PreparePublicationCommand(c.meta("prepare-publication", ports.Clock), string(request.ID), target.Digest(), request.Changes)
		if err != nil {
			return nil, err
		}
		if err := c.commit(ctx, cmd); err != nil {
			return nil, err
		}
		receipt, err = ports.Publisher.Publish(ctx, permit, c)
		if receipt.Outcome == "" {
			receipt = PublicationReceipt{ID: request.ID, Outcome: PublicationUnknown, Reason: errors.Join(errors.New("the publisher returned no outcome"), err).Error()}
		}
	}
	observed := run.PublicationResult{ID: string(request.ID), Outcome: string(receipt.Outcome), Reason: receipt.Reason}
	for _, p := range receipt.Paths {
		observed.Paths = append(observed.Paths, run.PublishedPathState{Path: p.Path.String(), State: string(p.State)})
	}
	cmd, err := co.PublishRunCommand(c.meta("published", ports.Clock), observed)
	if err != nil {
		return &receipt, err
	}
	return &receipt, c.commit(ctx, cmd)
}
