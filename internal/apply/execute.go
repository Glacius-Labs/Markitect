package apply

import (
	"context"
	"errors"
	"fmt"
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
	// Profile returns the session profile of a Manager's role.
	Profile func(manager, role string) (run.SessionProfile, error)
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
	engine, err := run.NewEngine(ctx, run.EngineConfig{Coordinator: co, Journal: c.Journal(), Sessions: ports.Sessions, Workspaces: ports.Workspaces, Checks: ports.Checks,
		Preparation: preparation, Profile: ports.Profile, Clock: ports.Clock, Limits: ports.Limits})
	if err != nil {
		return c.result(nil), err
	}
	if r := co.Current(); r.State == run.Waiting && len(co.OwnerQuestions()) == 0 {
		// The owner answered; the run continues (core rule W2).
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

// start prepares and starts the bound run when none is active. The first run
// opens a new assignment; a later run continues it only on a changed basis.
func (c *ControlledRun) start(ctx context.Context, clock func() time.Time) error {
	co := c.coordinator
	current := co.Current()
	switch current.State {
	case run.Prepared:
	case "", run.Finished:
		assignment, predecessor := fmt.Sprintf("assignment-%.12s", c.basis.Ref().ContentID.String()), ""
		runID := "run-1"
		if current.ID != "" {
			assignment, predecessor = current.Assignment, current.ID
			runID = fmt.Sprintf("run-%d", len(co.Assignments()[len(co.Assignments())-1].Runs)+1)
		}
		prepare := run.PrepareCommand(c.meta("prepare", clock).ID, assignment, runID, predecessor, clock().UTC(), c.basis)
		if err := c.commit(ctx, prepare); err != nil {
			if current.State == run.Finished {
				return fmt.Errorf("the run %s already finished %s on this basis: %w", current.ID, current.Outcome, err)
			}
			return err
		}
	default:
		return nil
	}
	current = co.Current()
	return c.commit(ctx, run.Command{ID: c.meta("start", clock).ID, Kind: run.StartRun, Assignment: current.Assignment, Run: current.ID, ObservedAt: clock().UTC()})
}

// verify runs the required checks on the aggregate, preflights the source
// and records the final evidence check (execution contract, Verifying).
func (c *ControlledRun) verify(ctx context.Context, engine *run.Engine, ports ExecutionPorts) error {
	a, _ := c.prepared.Prepared()
	selections, ok := a.Settings().Checks()
	if !ok {
		return fmt.Errorf("%w: the check selections are incomplete", run.ErrInvalid)
	}
	var required []run.RequiredCheck
	for _, selection := range selections {
		declaration := selection.Declaration()
		if declaration.RequiredBy == nil {
			continue
		}
		request, err := ports.CheckRequest(selection)
		if err != nil {
			return err
		}
		required = append(required, run.RequiredCheck{Name: declaration.ID.Value, Request: request})
	}
	passed, err := engine.Verify(ctx, required)
	if err != nil || c.coordinator.Current().State != run.Verifying {
		return err
	}
	request, err := c.publicationRequest(engine)
	if err != nil {
		return err
	}
	if _, err := ports.Publisher.Preflight(ctx, request, c); err != nil {
		outcome := run.Failed
		if errors.Is(err, ErrCaptureChanged) {
			outcome = run.Stale
		}
		current := c.coordinator.Current()
		return errors.Join(err, c.commit(ctx, run.Command{ID: c.meta("finish", ports.Clock).ID, Kind: run.FinishRun, Assignment: current.Assignment, Run: current.ID, ObservedAt: ports.Clock().UTC(),
			Outcome: outcome, Reason: "the source preflight refused the publication: " + err.Error()}))
	}
	return c.commit(ctx, c.coordinator.VerifyRunCommand(c.meta("verify", ports.Clock), passed))
}

// publicationRequest is the verified aggregate's change set against the
// captured source, including the regenerated registry shards.
func (c *ControlledRun) publicationRequest(engine *run.Engine) (PublicationRequest, error) {
	a, _ := c.prepared.Prepared()
	model := a.Structural().Model()
	inputs, request := model.Inputs(), model.Request()
	captured, ok := a.Structural().Registry().Registry()
	if !ok {
		return PublicationRequest{}, fmt.Errorf("%w: the captured registry is incomplete", run.ErrInvalid)
	}
	aggregate := engine.Aggregate()
	target, err := publicationTree(inputs.Raw.Census, captured, aggregate.Tree(), aggregate.Registry())
	if err != nil {
		return PublicationRequest{}, err
	}
	current := c.coordinator.Current()
	return PublicationRequest{ID: PublicationID(fmt.Sprintf("publication-%s", current.ID)), Run: current.ID, Repository: request.Selection.Repository, Guard: inputs.Guard,
		Changes: snapshot.Compare(inputs.Raw.Census, target)}, nil
}

// publish records the change set, writes it through the publisher and records
// the receipt, which finishes the run (execution contract, Publishing).
func (c *ControlledRun) publish(ctx context.Context, engine *run.Engine, ports ExecutionPorts) (*PublicationReceipt, error) {
	request, err := c.publicationRequest(engine)
	if err != nil {
		return nil, err
	}
	prepare, err := c.coordinator.PreparePublicationCommand(c.meta("prepare-publication", ports.Clock), string(request.ID), request.Changes)
	if err != nil {
		return nil, err
	}
	if err := c.commit(ctx, prepare); err != nil {
		return nil, err
	}
	permit, err := ports.Publisher.Preflight(ctx, request, c)
	var receipt PublicationReceipt
	if err == nil {
		receipt, err = ports.Publisher.Publish(ctx, permit, c)
	} else {
		receipt = PublicationReceipt{ID: request.ID, Outcome: PublicationStale, Reason: err.Error()}
		if !errors.Is(err, ErrCaptureChanged) {
			receipt.Outcome = PublicationFailed
		}
	}
	if receipt.Outcome == "" {
		receipt = PublicationReceipt{ID: request.ID, Outcome: PublicationUnknown, Reason: "the publisher returned no outcome"}
	}
	observed := run.PublicationResult{ID: string(request.ID), Outcome: string(receipt.Outcome), Reason: receipt.Reason}
	for _, p := range receipt.Paths {
		observed.Paths = append(observed.Paths, run.PublishedPathState{Path: p.Path.String(), State: string(p.State)})
	}
	if err != nil && observed.Reason == "" {
		observed.Reason = err.Error()
	}
	cmd, cmdErr := c.coordinator.PublishRunCommand(c.meta("published", ports.Clock), observed)
	if cmdErr != nil {
		return &receipt, errors.Join(err, cmdErr)
	}
	return &receipt, errors.Join(err, c.commit(ctx, cmd))
}
