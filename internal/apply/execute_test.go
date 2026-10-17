package apply_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// executionWorkspaces keeps step workspaces in memory.
type executionWorkspaces struct {
	trees map[run.WorkspaceID]snapshot.Tree
}

func (w *executionWorkspaces) Materialize(_ context.Context, s run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	w.trees[s.ID] = s.Tree
	return run.WorkspaceObservation{ID: s.ID, Volume: "vol-" + string(s.ID), Tree: s.Tree.Digest()}, nil
}
func (w *executionWorkspaces) Harvest(_ context.Context, r run.HarvestRequest) (run.WorkspaceHarvest, error) {
	return run.WorkspaceHarvest{ID: r.ID, Tree: w.trees[r.ID]}, nil
}
func (w *executionWorkspaces) Close(_ context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	delete(w.trees, id)
	return run.WorkspaceObservation{ID: id, Closed: true}, nil
}

// executionSessions is a scripted adapter whose Agents change nothing: they
// report in Explore, submit complete no-change results and pass every review.
type executionSessions struct {
	t *testing.T
}

// adapterRecords numbers the fake adapter's records across instances, because
// record IDs are unique in the whole journal.
var adapterRecords int

func (f *executionSessions) meta() run.RecordMeta {
	adapterRecords++
	return run.RecordMeta{ID: fmt.Sprintf("adapter-%d", adapterRecords), ObservedAt: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
}

func (f *executionSessions) wire(data string) run.WirePayload {
	p, err := run.NewPayload("application/json", "utf-8", []byte(data))
	if err != nil {
		f.t.Fatal(err)
	}
	return run.WirePayload{Identity: run.WireIdentity{ContentID: p.Ref().ContentID, Size: p.Ref().Size}, Journal: p}
}

func (f *executionSessions) send(ctx context.Context, io run.SessionIO, step run.StepID, call run.ProviderCallID, data string) error {
	m := f.meta()
	if _, err := io.PrepareSend(ctx, run.TransportPreparation{RecordMeta: m, Message: run.MessageID(m.ID), Step: step, Call: call, Wire: f.wire(data)}); err != nil {
		return err
	}
	_, err := io.ObserveSend(ctx, run.TransportObservation{RecordMeta: f.meta(), Message: run.MessageID(m.ID), Outcome: run.Sent, Reason: "observed write"})
	return err
}

func (f *executionSessions) Open(ctx context.Context, a run.SessionAdmission, io run.SessionIO) (run.SessionObservation, error) {
	if err := f.send(ctx, io, "", "", `{"control":"initialize"}`); err != nil {
		return run.SessionObservation{}, err
	}
	if a.Plan.Resume != "" {
		_, err := io.Bind(ctx, run.SessionObservation{RecordMeta: f.meta(), ProviderSession: a.Plan.Resume, Wire: f.wire(`{"resumed":true}`), Reason: "resumed"})
		return run.SessionObservation{ProviderSession: a.Plan.Resume}, err
	}
	_, err := io.Ready(ctx, run.SessionReadiness{RecordMeta: f.meta(), Wire: f.wire(`{"control":"ready"}`), Reason: "observed readiness"})
	return run.SessionObservation{Ready: true}, err
}

func (f *executionSessions) Step(ctx context.Context, a run.StepAdmission, io run.SessionIO) (run.StepObservation, error) {
	var packet run.StepPacket
	if err := json.Unmarshal(a.Input.Bytes(), &packet); err != nil {
		return run.StepObservation{}, err
	}
	step := a.Plan.ID
	if err := f.send(ctx, io, step, "", `{"user":"turn"}`); err != nil {
		return run.StepObservation{}, err
	}
	if a.Session.Resume == "" {
		if _, err := io.Bind(ctx, run.SessionObservation{RecordMeta: f.meta(), ProviderSession: run.ProviderSessionID("provider-" + string(a.Session.ID)), Wire: f.wire(`{"thread":"new"}`), Reason: "observed"}); err != nil {
			return run.StepObservation{}, err
		}
	}
	if _, err := io.ObserveStep(ctx, run.StepObservation{RecordMeta: f.meta(), Step: step, ProviderStep: "turn", Wire: f.wire(`{"turn":"started"}`), Reason: "observed"}); err != nil {
		return run.StepObservation{}, err
	}
	tool, args := run.ToolReport, `{"summary":"Nothing to change.","proposedChanges":[],"prerequisites":[],"uncertainties":[],"questions":[]}`
	switch packet.Phase {
	case run.PhaseWork:
		tool, args = run.ToolSubmit, `{"purpose":"local","scope":"complete","summary":"No change.","assessments":[],"dependencies":[]}`
	case run.PhaseIntegrate:
		tool, args = run.ToolSubmit, `{"purpose":"integration","scope":"complete","summary":"No change.","assessments":[],"dependencies":[]}`
	case run.PhaseReviewerExpectations:
		tool, args = run.ToolRecordExpectations, `{"cases":[{"name":"area","expectation":"Unchanged.","discriminates":"Any change."}]}`
	case run.PhaseReviewerCandidate:
		tool, args = run.ToolSubmitReview, `{"verdict":"pass","summary":"Fine.","evidence":[],"findings":[],"resolutions":[]}`
	}
	call := run.ProviderCallID("call-" + string(step))
	result, err := io.Call(ctx, run.ToolCall{RecordMeta: f.meta(), Step: step, Call: call, Tool: tool, Input: f.wire(fmt.Sprintf(`{"id":1,"name":%q,"arguments":%s}`, tool, args))})
	if err != nil {
		return run.StepObservation{}, err
	}
	if result.Outcome.Status != run.ToolOK {
		f.t.Errorf("%s %s refused: %s", packet.Manager, tool, result.Outcome.Reason)
	}
	if err := f.send(ctx, io, step, call, string(result.Output.Bytes())); err != nil {
		return run.StepObservation{}, err
	}
	o := run.StepObservation{RecordMeta: f.meta(), Step: step, ProviderStep: "turn", Wire: f.wire(`{"turn":"done"}`), Terminal: true, Reason: "observed"}
	_, err = io.ObserveStep(ctx, o)
	return o, err
}

func (f *executionSessions) Inspect(context.Context, run.InspectionRequest) (run.StepObservation, error) {
	return run.StepObservation{}, errors.New("not inspected")
}

func (f *executionSessions) Stop(ctx context.Context, _ run.StopRequest, io run.SessionIO) (run.StopObservation, error) {
	o := run.StopObservation{RecordMeta: f.meta(), Terminated: true, Reason: "observed exit"}
	_, err := io.ObserveStop(ctx, o)
	return o, err
}

type executionChecks struct{ runs int }

func (c *executionChecks) Run(_ context.Context, r run.CheckRequest) (run.CheckReceipt, error) {
	c.runs++
	return run.CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true}, nil
}

type executionPublisher struct {
	requests   []apply.PublicationRequest
	stale      bool
	failing    error
	preflights int
	failOn     int // the preflight that fails transiently, counted from 1
}

func (p *executionPublisher) Preflight(_ context.Context, r apply.PublicationRequest, c apply.ControlCheck) (apply.PublicationPermit, error) {
	if err := c.Check(context.Background()); err != nil {
		return apply.PublicationPermit{}, err
	}
	p.preflights++
	if p.preflights == p.failOn {
		return apply.PublicationPermit{}, errors.New("the repository is busy")
	}
	if p.stale {
		return apply.PublicationPermit{}, apply.ErrCaptureChanged
	}
	if p.failing != nil {
		return apply.PublicationPermit{}, p.failing
	}
	p.requests = append(p.requests, r)
	return apply.PublicationPermit{Request: r, Token: "permit"}, nil
}

func (p *executionPublisher) Publish(_ context.Context, permit apply.PublicationPermit, _ apply.ControlCheck) (apply.PublicationReceipt, error) {
	outcome := apply.PublicationNoChange
	if len(permit.Request.Changes) > 0 {
		outcome = apply.PublicationApplied
	}
	return apply.PublicationReceipt{ID: permit.Request.ID, Outcome: outcome}, nil
}

func (p *executionPublisher) Inspect(context.Context, apply.PublicationRequest) (apply.PublicationReceipt, error) {
	return apply.PublicationReceipt{}, errors.New("not inspected")
}

func executionPorts(t *testing.T, publisher *executionPublisher, checks *executionChecks) apply.ExecutionPorts {
	t.Helper()
	profile, err := run.NewPayload("application/json", "utf-8", []byte(`{"profile":"controlled"}`))
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	return apply.ExecutionPorts{
		Sessions: &executionSessions{t: t}, Workspaces: &executionWorkspaces{trees: map[run.WorkspaceID]snapshot.Tree{}}, Checks: checks, Publisher: publisher,
		Profile: func(run.ProfileRequest) (run.SessionProfile, error) {
			return run.SessionProfile{ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", Profile: profile, MaxWireBytes: 1 << 20}, nil
		},
		CheckRequest: func(s apply.CheckSelection) (run.CheckRequest, error) {
			d := s.Declaration()
			return run.CheckRequest{ID: run.CheckID("check-" + d.ID.Value), ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Executable: "/usr/local/go/bin/go",
				Args: d.Args.Value, Directory: d.Directory.Value, Limits: run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 64}, Timeout: d.Timeout.Value, MaxOutputBytes: 1 << 16}, nil
		},
		Clock: func() time.Time { clock = clock.Add(time.Millisecond); return clock },
		Limits: run.EngineLimits{StartTimeout: time.Minute, StopTimeout: time.Minute, HarvestEntries: 10000, HarvestBytes: 1 << 26,
			Record: realization.CandidateRecordLimits{EncodedBytes: 1 << 24, FileBytes: 1 << 24, ManifestEntries: 10000, MetadataRows: 100000}},
	}
}

// Execute runs the whole first Apply through controlled ports: every Manager
// explores, works and is reviewed, the required checks pass on the aggregate,
// the source is preflighted, and the empty change set publishes as no change.
func TestExecuteRunsTheFirstApplyToItsPublication(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	publisher, checks := &executionPublisher{}, &executionChecks{}
	result, err := c.Execute(context.Background(), executionPorts(t, publisher, checks))
	if err != nil {
		t.Fatal(err)
	}
	if checks.runs != 1 {
		t.Fatal("the required check did not run once on the aggregate", checks.runs)
	}
	if result.Run.State != run.Finished || result.Run.Outcome != run.NoChange {
		t.Fatalf("run did not publish: %+v", result.Run)
	}
	for _, task := range result.Tasks {
		if task.State != run.TaskDone {
			t.Fatalf("task %s not done: %s", task.Manager, task.State)
		}
	}
	if result.Publication == nil || result.Publication.Outcome != apply.PublicationNoChange || len(publisher.requests) != 2 || len(publisher.requests[0].Changes) != 0 {
		t.Fatal("publication not preflighted and published", result.Publication, publisher.requests)
	}
	if again, err := c.Execute(context.Background(), executionPorts(t, publisher, checks)); err == nil || again.Run.ID != result.Run.ID {
		t.Fatal("a finished run started again on the same basis", err)
	}
}

func TestExecuteFinishesStaleWhenTheSourceChanged(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	publisher := &executionPublisher{stale: true}
	result, err := c.Execute(context.Background(), executionPorts(t, publisher, &executionChecks{}))
	if !errors.Is(err, apply.ErrCaptureChanged) || result.Run.State != run.Finished || result.Run.Outcome != run.Stale {
		t.Fatalf("stale source not finished Stale: %+v %v", result.Run, err)
	}
}

// A transient preflight error leaves the run Verifying; the next Execute
// publishes it.
func TestExecuteRetriesATransientPreflightError(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	publisher, checks := &executionPublisher{failing: errors.New("the repository is busy")}, &executionChecks{}
	result, err := c.Execute(context.Background(), executionPorts(t, publisher, checks))
	if err == nil || result.Run.State != run.Verifying {
		t.Fatalf("a transient refusal changed the run: %+v %v", result.Run, err)
	}
	publisher.failing = nil
	result, err = c.Execute(context.Background(), executionPorts(t, publisher, checks))
	if err != nil || result.Run.State != run.Finished || result.Run.Outcome != run.NoChange || checks.runs != 1 {
		t.Fatalf("the retry did not publish once checked: %+v %v checks=%d", result.Run, err, checks.runs)
	}
}

// An active run bound to another capture finishes Stale, and a new run
// starts on the current capture (execution contract, outcomes).
func TestExecuteFinishesARunOfAnotherCaptureStale(t *testing.T) {
	prepared, facts, s, j := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	other := facts
	other.ProductBuild = "another-product-build"
	basis, err := prepared.BindRun(other)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	controlCommit(t, c.Coordinator(), j, run.PrepareCommand("old-prepare", "old", "old-run", "", at, basis))
	controlCommit(t, c.Coordinator(), j, run.Command{ID: "old-start", Kind: run.StartRun, Assignment: "old", Run: "old-run", ObservedAt: at})
	result, err := c.Execute(context.Background(), executionPorts(t, &executionPublisher{}, &executionChecks{}))
	if err != nil || result.Run.ID == "old-run" || result.Run.Outcome != run.NoChange {
		t.Fatalf("a new run did not replace the drifted one: %+v %v", result.Run, err)
	}
	runs := c.Coordinator().Assignments()[0].Runs
	if runs[0].ID != "old-run" || runs[0].Outcome != run.Stale {
		t.Fatalf("the drifted run did not finish Stale: %+v", runs)
	}
}

// A transient error in the publication's own preflight journals nothing, so
// the next Execute publishes instead of reconciling (execution contract,
// Publishing).
func TestExecuteRetriesATransientPublicationPreflight(t *testing.T) {
	prepared, facts, s, _ := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	publisher, checks := &executionPublisher{failOn: 2}, &executionChecks{}
	result, err := c.Execute(context.Background(), executionPorts(t, publisher, checks))
	if err == nil || result.Run.State != run.Publishing || c.Coordinator().Publication().ID != "" {
		t.Fatalf("a transient preflight error journaled the publication: %+v %v", result.Run, err)
	}
	result, err = c.Execute(context.Background(), executionPorts(t, publisher, checks))
	if err != nil || result.Run.State != run.Finished || result.Run.Outcome != run.NoChange {
		t.Fatalf("the retry did not publish: %+v %v", result.Run, err)
	}
}
