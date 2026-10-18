package apply_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// These controlled ports test the consumer's composition. They provide no
// filesystem lock, physical identity, SQL transaction or durability evidence.
type controlPort struct {
	acquire func(context.Context, apply.ControlRequest) (apply.ControlSession, error)
}

func (p controlPort) Acquire(ctx context.Context, r apply.ControlRequest) (apply.ControlSession, error) {
	return p.acquire(ctx, r)
}

type controlSession struct {
	binding  apply.ControlBinding
	journal  run.Journal
	check    func(context.Context) error
	checks   int
	bindings int
	closes   int
	closeErr error
}

func (s *controlSession) Binding() apply.ControlBinding { s.bindings++; return s.binding }
func (s *controlSession) Journal() run.Journal          { return s.journal }
func (s *controlSession) Check(ctx context.Context) error {
	s.checks++
	if s.check != nil {
		return s.check(ctx)
	}
	return nil
}
func (s *controlSession) Close() error { s.closes++; return s.closeErr }

type controlJournal struct {
	blobs                map[run.BlobRef]run.Payload
	batches              []run.AppendBatch
	puts, appends, reads int
	readErr, appendErr   error
}

func (j *controlJournal) Put(_ context.Context, p run.Payload) (run.BlobRef, error) {
	j.puts++
	if j.blobs == nil {
		j.blobs = map[run.BlobRef]run.Payload{}
	}
	j.blobs[p.Ref()] = p
	return p.Ref(), nil
}
func (j *controlJournal) ReadBlob(_ context.Context, ref run.BlobRef) (run.Payload, error) {
	p, ok := j.blobs[ref]
	if !ok {
		return p, errors.New("controlled missing blob")
	}
	return p, nil
}
func (j *controlJournal) Append(_ context.Context, b run.AppendBatch) (run.AppendReceipt, error) {
	j.appends++
	for _, existing := range j.batches {
		if existing.ID() == b.ID() {
			if !reflect.DeepEqual(existing.Bytes(), b.Bytes()) {
				return run.AppendReceipt{}, run.ErrConflict
			}
			return run.ReceiptFor(existing), j.appendErr
		}
	}
	var head uint64
	if len(j.batches) > 0 {
		head = j.batches[len(j.batches)-1].Revision()
	}
	if b.ExpectedRevision() != head {
		return run.AppendReceipt{}, run.ErrConflict
	}
	j.batches = append(j.batches, b)
	return run.ReceiptFor(b), j.appendErr
}
func (j *controlJournal) Read(_ context.Context, q run.EventQuery) (run.EventPage, error) {
	j.reads++
	var head uint64
	if len(j.batches) > 0 {
		head = j.batches[len(j.batches)-1].Revision()
	}
	if q.Through != 0 {
		head = q.Through
	}
	p := run.EventPage{Revision: head}
	for _, b := range j.batches {
		if b.ExpectedRevision() < q.After || b.Revision() > head || q.BatchID != "" && b.ID() != q.BatchID {
			continue
		}
		if len(p.Batches) == int(q.Limit) {
			p.Next = p.Batches[len(p.Batches)-1].Revision()
			break
		}
		p.Batches = append(p.Batches, b)
	}
	return p, j.readErr
}

var controlOptions = apply.ControlOptions{
	StateDirectory: "explicit-external-state-directory",
	RestoreLimits:  run.RestoreLimits{PageSize: 1, Batches: 20, Events: 40, Bytes: 16 << 20},
}

func controlFixture(t *testing.T) (apply.CapturedSettingsResult, apply.RunBindingFacts, *controlSession, *controlJournal) {
	t.Helper()
	prepared, _, _ := runBindingCapture(t, nil, false)
	facts := runBindingHost(t)
	j := &controlJournal{}
	s := &controlSession{binding: apply.ControlBinding{Repository: facts.RepositoryIdentity, StoreID: "registered-store", Epoch: 4}, journal: j}
	return prepared, facts, s, j
}

func controlSource(t *testing.T, prepared apply.CapturedSettingsResult, s *controlSession) apply.Source {
	t.Helper()
	a, _ := prepared.Prepared()
	m := a.Structural().Model()
	return controlledSource{capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
		if s.checks == 0 || s.closes != 0 || !reflect.DeepEqual(request, m.Request()) {
			t.Fatal("recapture did not use final actual closure under live control")
		}
		observed := m.Inputs()
		observed.Guard = "fresh-equivalent-opaque-guard"
		return observed, nil
	}}
}

func controlAcquire(t *testing.T, prepared apply.CapturedSettingsResult, facts apply.RunBindingFacts, s *controlSession) apply.Control {
	t.Helper()
	a, _ := prepared.Prepared()
	m := a.Structural().Model()
	return controlPort{acquire: func(_ context.Context, request apply.ControlRequest) (apply.ControlSession, error) {
		want := apply.ControlRequest{Repository: m.Request().Selection.Repository, RepositoryIdentity: facts.RepositoryIdentity, Revision: m.Inputs().Head, Guard: m.Inputs().Guard, StateDirectory: controlOptions.StateDirectory}
		if request != want {
			t.Fatalf("control acquisition changed captured facts: got=%+v want=%+v", request, want)
		}
		return s, nil
	}}
}

func openControlled(t *testing.T, prepared apply.CapturedSettingsResult, facts apply.RunBindingFacts, s *controlSession) *apply.ControlledRun {
	t.Helper()
	r, err := apply.NewControlService(controlSource(t, prepared, s), controlAcquire(t, prepared, facts, s)).Open(context.Background(), prepared, facts, controlOptions)
	c, ok := r.Controlled()
	if err != nil || !ok {
		t.Fatalf("open control: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func controlCommit(t *testing.T, c *run.Coordinator, j run.Journal, cmd run.Command) {
	t.Helper()
	p, err := c.Propose(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Commit(context.Background(), j, p); err != nil {
		t.Fatal(err)
	}
}

func TestControlOpenRestoresRecordedHoldsAndKeepsOriginalBasis(t *testing.T) {
	prepared, facts, s, j := controlFixture(t)
	basis, _ := prepared.BindRun(facts)
	original, _ := run.NewCoordinator(facts.RepositoryIdentity)
	at := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	controlCommit(t, original, j, run.PrepareCommand("prepare", "assignment", "run", "", at, basis))
	controlCommit(t, original, j, run.Command{ID: "start", Kind: run.StartRun, Assignment: "assignment", Run: "run", ObservedAt: at})
	a, _ := prepared.Prepared()
	startup := controlStartup(t, original, a.Structural().Model().Inputs().Raw.Census.Digest(), at)
	controlCommit(t, original, j, startup)
	puts, appends := j.puts, j.appends
	c := openControlled(t, prepared, facts, s)
	if c.Binding() != s.binding || c.Basis().Ref() != basis.Ref() || c.Coordinator().Current() != original.Current() || c.Coordinator().Revision() != original.Revision() || !c.Coordinator().RecoveryRequired() || !reflect.DeepEqual(c.Coordinator().Budget(), original.Budget()) {
		t.Fatal("opening lost original capture, confirmed history or unresolved holds")
	}
	if j.puts != puts || j.appends != appends || j.reads != 3 {
		t.Fatal("open wrote history or did not reconstruct bounded pages")
	}
	view := runBindingRead(t, c.Basis())
	if view.Source.Guard != string(a.Structural().Model().Inputs().Guard) {
		t.Fatal("fresh recapture guard replaced original publication basis")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil || s.closes != 1 {
		t.Fatal("close was not idempotent", err)
	}
	if _, err := c.Journal().Put(context.Background(), basis.Payload()); !errors.Is(err, apply.ErrControlClosed) || j.puts != puts {
		t.Fatal("closed handle reached journal", err)
	}
}

func TestControlledJournalRetainsUncertainReceiptAndStopsAfterControlLoss(t *testing.T) {
	prepared, facts, s, j := controlFixture(t)
	c := openControlled(t, prepared, facts, s)
	at := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	p, err := c.Coordinator().Propose(run.PrepareCommand("prepare", "assignment", "run", "", at, c.Basis()))
	if err != nil {
		t.Fatal(err)
	}
	j.appendErr = errors.New("controlled lost acknowledgement")
	receipt, err := c.Coordinator().Commit(context.Background(), c.Journal(), p)
	if !errors.Is(err, run.ErrUncertain) || receipt != run.ReceiptFor(p.Batch()) || c.Coordinator().Revision() != 0 {
		t.Fatal("partial uncertain append outcome was replaced", err)
	}
	if _, ok := c.Coordinator().Pending(); !ok {
		t.Fatal("uncertainty did not stay RUN-owned")
	}
	j.appendErr = nil
	if err := c.Coordinator().ReconcilePending(context.Background(), c.Journal()); err != nil || c.Coordinator().Revision() != p.Batch().Revision() || j.appends != 1 {
		t.Fatal("reconciliation replayed or lost exact committed batch", err)
	}
	puts, appends := j.puts, j.appends
	lost := errors.New("controlled lock lost")
	s.check = func(context.Context) error { return lost }
	if _, err := c.Journal().Put(context.Background(), c.Basis().Payload()); !errors.Is(err, lost) || !errors.Is(err, apply.ErrControlBinding) {
		t.Fatal(err)
	}
	s.check = nil // A later successful check cannot revive this handle.
	if _, err := c.Journal().Append(context.Background(), p.Batch()); !errors.Is(err, apply.ErrControlBinding) || j.puts != puts || j.appends != appends {
		t.Fatal("lost controller reached journal", err)
	}
}

func controlStartup(t *testing.T, c *run.Coordinator, candidate snapshot.TreeID, at time.Time) run.Command {
	t.Helper()
	profile, err := run.NewPayload("application/json", "utf-8", []byte(`{"profile":"controlled"}`))
	if err != nil {
		t.Fatal(err)
	}
	item, ok := c.Next()
	if !ok {
		t.Fatal("no admissible step")
	}
	tools, err := run.ToolCatalog(item.Phase)
	if err != nil {
		t.Fatal(err)
	}
	// The root's first step is a reviewer's expectations step, which sees no
	// candidate.
	if item.Phase != run.PhaseReviewerExpectations {
		t.Fatal("unexpected first step", item)
	}
	empty, _ := snapshot.NewTree(nil)
	candidate = empty.Digest()
	plan := run.SessionPlan{ID: "session", Role: "reviewer", Workspace: "controlled-linux-workspace", ProviderHome: item.Home, Resume: item.Resume, Candidate: candidate, ReadOnly: true, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 1 << 20, Profile: profile.Ref(), Tools: tools.Ref()}
	command, err := c.PrepareTaskSessionCommand(run.RecordMeta{ID: "reserve", ObservedAt: at}, item, plan, run.Reservation{ID: "invocation", Kind: run.SessionStart, Purpose: item.Purpose, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	return command
}
