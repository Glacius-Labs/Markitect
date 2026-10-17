package run_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type priorSessionBlob struct {
	Ref   run.BlobRef
	Bytes []byte
}
type priorSessionFixture struct {
	Source, Scope string
	Batches       [][]byte
	Receipts      []run.AppendReceipt
	Blobs         []priorSessionBlob
}

func TestLiteralBoundSessionBeforeReadinessKeepsExactBytesAndReceipts(t *testing.T) {
	// Captured with unmodified production code at the Source SHA, before
	// readiness edits. No current Bind or session constructor regenerates it.
	data, err := os.ReadFile("testdata/session-bound-before-readiness.json")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != "6a994ed2bb8dd775f13c9326b7197e0b91a46082a84357caa8f57e095ca0d9d4" {
		t.Fatal("prechange session fixture changed", err)
	}
	var f priorSessionFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if f.Source != "fbffd7fbf67ab1f4939f623268d401e1fbd2c7ba" || len(f.Batches) != len(f.Receipts) {
		t.Fatal("capture provenance/receipts lost")
	}
	j := journal()
	for _, original := range f.Blobs {
		p, err := run.NewPayload(original.Ref.MediaType, original.Ref.Encoding, original.Bytes)
		if err != nil || p.Ref() != original.Ref {
			t.Fatal("original payload identity changed", err)
		}
		j.blobs[p.Ref()] = p
	}
	for n, original := range f.Batches {
		b, err := run.DecodeBatch(original)
		if err != nil || !bytes.Equal(original, b.Bytes()) || !f.Receipts[n].Matches(b) {
			t.Fatal("literal batch/receipt changed", n, err)
		}
		j.batches = append(j.batches, b)
		j.head = b.Revision()
	}
	reads := &readJournal{store: j}
	c := restored(t, reads)
	session, _ := c.Session("s")
	step, _ := c.Step("step")
	if !session.Bound || session.Ready || session.Readiness.BatchID != "" || session.InitialStep != "" || session.InitialSend != "" || session.Provider != "provider-session" || step.Provider != "provider-turn" || !c.RecoveryRequired() {
		t.Fatal("old session reinterpreted", session, step)
	}
	before := c.Budget()
	for n, b := range j.batches {
		events := b.Events()
		p, err := c.Propose(events[len(events)-1].Body)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := c.Commit(context.Background(), reads, p)
		if err != nil || receipt != f.Receipts[n] {
			t.Fatal("historical retry changed", n, err)
		}
		if !bytes.Equal(b.Bytes(), f.Batches[n]) {
			t.Fatal("historical bytes rewritten")
		}
	}
	if reads.writes != 0 || !reflect.DeepEqual(before, c.Budget()) {
		t.Fatal("historical retry wrote or charged")
	}
}

func unboundSession(t *testing.T, role string) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j := running(t)
	cmd := sessionPreparation(t, c, "s")
	if role == "reviewer" {
		cmd.Session.Plan.Role, cmd.Session.Plan.ReadOnly, cmd.Reservation.Purpose = role, true, run.Assessment
	}
	commit(t, c, j, cmd)
	io, err := c.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	return c, j, io
}

func readyObservation(t *testing.T) run.SessionReadiness {
	return run.SessionReadiness{RecordMeta: meta("ready"), Wire: wire(t, `{"control":"initialized"}`), Reason: "observed control acknowledgment; provider ID not supplied"}
}

func readySession(t *testing.T, role string, settled bool) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j, io := unboundSession(t, role)
	send(t, io, "initialize", "")
	if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
		t.Fatal(err)
	}
	if settled {
		commit(t, c, j, settle("open-s", time.Second))
	}
	return c, j, io
}

func readinessStep(t *testing.T, c *run.Coordinator, id run.StepID) run.Command {
	return stepPreparation(t, c, id, "invoke-"+string(id))
}

func firstSend(t *testing.T) run.TransportPreparation {
	return run.TransportPreparation{RecordMeta: meta("prepare-first-send"), Message: "first-send", Step: "first", Wire: wire(t, `{"user":"actual first compiled turn"}`)}
}

func TestReadinessAdmitsOneActualFirstStepBeforeObservedProviderBinding(t *testing.T) {
	for _, role := range []string{"manager"} {
		t.Run(role, func(t *testing.T) {
			c, j, io := readySession(t, role, true)
			s, _ := c.Session("s")
			if !s.Ready || s.Bound || s.Provider != "" || s.Readiness.BatchID == "" || s.Binding.BatchID != "" {
				t.Fatal("readiness invented binding", s)
			}
			if _, err := c.SessionAdmission("s"); err == nil {
				t.Fatal("ready session reopened")
			}
			if _, err := c.Inspection("s", "first"); !errors.Is(err, run.ErrObservationLimit) {
				t.Fatal("ready session became inspectable", err)
			}
			before := c.Budget()
			cmd := readinessStep(t, c, "first")
			p, err := c.Propose(cmd)
			if err != nil {
				t.Fatal(err)
			}
			if current, _ := c.Session("s"); current.InitialStep != "" || c.Budget() != before {
				t.Fatal("proposal adopted first step")
			}
			if events := p.Batch().Events(); len(events) != 2 || events[0].Kind != run.ReserveInvocation || events[1].Kind != run.PrepareStep {
				t.Fatal("first step admission not atomic")
			}
			if _, err := c.Commit(context.Background(), j, p); err != nil {
				t.Fatal(err)
			}
			if _, err := c.StepAdmission("first"); err != nil {
				t.Fatal(err)
			}
			prep := firstSend(t)
			permit, err := io.PrepareSend(context.Background(), prep)
			if err != nil || !permit.Dispatch || permit.Identity != prep.Wire.Identity {
				t.Fatal(permit, err)
			}
			incoming := run.SessionObservation{RecordMeta: meta("bind-real-id"), ProviderSession: "observed-provider-session", Wire: wire(t, `{"system/init":"observed-provider-session"}`), Reason: "actual init after first user"}
			// An immediate inbound event is retained by the adapter until the
			// actual send outcome is durably observed; it cannot bind early.
			if _, err := io.Bind(context.Background(), incoming); !errors.Is(err, run.ErrState) {
				t.Fatal("init outran send observation", err)
			}
			if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("sent-first"), Message: prep.Message, Outcome: run.Sent, Reason: "observed actual first input write"}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.Bind(context.Background(), incoming); err != nil {
				t.Fatal(err)
			}
			s, _ = c.Session("s")
			if !s.Bound || !s.Ready || s.Provider != incoming.ProviderSession || s.Binding.BatchID == s.Readiness.BatchID || s.InitialStep != "first" || s.InitialSend != prep.Message {
				t.Fatal("binding/readiness provenance lost", s)
			}
			if _, err := c.StepAdmission("first"); err == nil {
				t.Fatal("first step dispatch reissued after binding")
			}
			if _, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("first-result"), Step: "first", ProviderStep: "observed-turn", Wire: wire(t, `{"result":"done"}`), Terminal: true, Reason: "actual result"}); err != nil {
				t.Fatal(err)
			}
			commit(t, c, j, settle("invoke-first", time.Second))
			if _, err := c.Propose(readinessStep(t, c, "second")); !errors.Is(err, run.ErrState) {
				t.Fatal("initial Explore granted a later phase", err)
			}
		})
	}
}

func TestReadinessNeedsObservedAckAndSettledManagerStartup(t *testing.T) {
	for _, role := range []string{"manager"} {
		t.Run(role, func(t *testing.T) {
			c, j, io := unboundSession(t, role)
			if _, err := c.Propose(readinessStep(t, c, "first")); !errors.Is(err, run.ErrState) {
				t.Fatal("step before readiness", err)
			}
			prep := run.TransportPreparation{RecordMeta: meta("prepare-initialize"), Message: "initialize", Wire: wire(t, `{"initialize":true}`)}
			if _, err := io.PrepareSend(context.Background(), prep); err != nil {
				t.Fatal(err)
			}
			if _, err := c.SessionAdmission("s"); !errors.Is(err, run.ErrObservationLimit) {
				t.Fatal("known startup send permitted another Open", err)
			}
			if _, err := io.Ready(context.Background(), readyObservation(t)); !errors.Is(err, run.ErrState) {
				t.Fatal("unresolved startup send acknowledged ready", err)
			}
			if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("initialized-send"), Message: "initialize", Outcome: run.Sent, Reason: "observed initialize write"}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Propose(readinessStep(t, c, "first")); !errors.Is(err, run.ErrState) {
				t.Fatal("unsettled startup admitted first step", err)
			}
			wait := command("wait", run.WaitRun)
			wait.Reason = "idle"
			wait.NextActor = "owner"
			if _, err := c.Propose(wait); !errors.Is(err, run.ErrState) {
				t.Fatal("ready unbound session became quiescent", err)
			}
			finish := command("finish", run.FinishRun)
			finish.Outcome = run.Incomplete
			finish.Reason = "retain"
			if _, err := c.Propose(finish); !errors.Is(err, run.ErrState) {
				t.Fatal("live ready session finished", err)
			}
			commit(t, c, j, settle("open-s", time.Second))
		})
	}
}

func TestInitialSendSlotCannotReplayOrUseAnotherMessageEvenAfterNotSent(t *testing.T) {
	for _, outcome := range []run.SendOutcome{run.Sent, run.NotSent, run.SendUncertain} {
		t.Run(string(outcome), func(t *testing.T) {
			c, j, io := readySession(t, "manager", true)
			commit(t, c, j, readinessStep(t, c, "first"))
			if _, err := c.Propose(readinessStep(t, c, "another")); !errors.Is(err, run.ErrState) {
				t.Fatal("second unbound step admitted", err)
			}
			for _, change := range []func(*run.TransportPreparation){func(p *run.TransportPreparation) { p.Step = "" }, func(p *run.TransportPreparation) { p.Step = "other" }, func(p *run.TransportPreparation) { p.Control = true }} {
				bad := firstSend(t)
				change(&bad)
				if _, err := io.PrepareSend(context.Background(), bad); !errors.Is(err, run.ErrState) {
					t.Fatal("first input bypassed its slot", err)
				}
			}
			prep := firstSend(t)
			permit, err := io.PrepareSend(context.Background(), prep)
			if err != nil || !permit.Dispatch {
				t.Fatal(err)
			}
			if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("send-observed"), Message: prep.Message, Outcome: outcome, Reason: "actual send observation"}); err != nil {
				t.Fatal(err)
			}
			for _, restoredHistory := range []bool{false, true} {
				if restoredHistory {
					c = restored(t, &readJournal{store: j})
					io, err = c.SessionChannel(j, "s")
					if err != nil {
						t.Fatal(err)
					}
				}
				known, err := io.PrepareSend(context.Background(), prep)
				if !errors.Is(err, run.ErrObservationLimit) || known.Dispatch || known.Receipt != permit.Receipt {
					t.Fatal("known send replayed", known, err)
				}
				second := prep
				second.ID = "prepare-second-send"
				second.Message = "second-send"
				if next, err := io.PrepareSend(context.Background(), second); err == nil || next.Dispatch {
					t.Fatal("different message reused initial slot", err)
				}
				if _, err := c.SessionAdmission("s"); err == nil {
					t.Fatal("initial send reopened session")
				}
				if restoredHistory && !c.RecoveryRequired() {
					t.Fatal("restore erased live/first-step barrier")
				}
				if c.Budget().Held.Steps != 1 {
					t.Fatal("initial step hold lost")
				}
			}
		})
	}
}

func TestReadinessDoesNotClearUncertaintyOrReleaseRealStartup(t *testing.T) {
	c, j, io := unboundSession(t, "manager")
	if _, err := io.Bind(context.Background(), run.SessionObservation{RecordMeta: meta("unknown-open"), Uncertain: true, Reason: "actual startup effect not fully observed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
		t.Fatal(err)
	}
	s, _ := c.Session("s")
	if !s.Ready || !s.Uncertain || s.Bound || c.Current().State != run.Reconciling {
		t.Fatal("ready erased uncertainty", s)
	}
	if _, err := c.Propose(settle("open-s", time.Second)); !errors.Is(err, run.ErrState) {
		t.Fatal("uncertain startup settled", err)
	}
	if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("terminated"), Terminated: true, Reason: "Host observed termination"}); err != nil {
		t.Fatal(err)
	}
	release := command("release-open", run.ReleaseUndispatched)
	release.Invocation = "open-s"
	release.Reason = "no outgoing write claimed"
	if _, err := c.Propose(release); !errors.Is(err, run.ErrState) {
		t.Fatal("acknowledged startup released as undispatched", err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	if c.Budget().Consumed.SessionStarts != 1 {
		t.Fatal("real startup not charged")
	}
}

func TestMissingInitRetainsFirstStepAndShutdownRemainsAvailable(t *testing.T) {
	c, j, io := readySession(t, "manager", true)
	commit(t, c, j, readinessStep(t, c, "first"))
	prep := firstSend(t)
	if _, err := io.PrepareSend(context.Background(), prep); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("first-sent"), Message: prep.Message, Outcome: run.Sent, Reason: "input written; init absent"}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("unknown-first"), Step: "first", Uncertain: true, Reason: "no init confirmation after possible user dispatch"}); err != nil {
		t.Fatal(err)
	}
	c = restored(t, &readJournal{store: j})
	io, err := c.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	if c.Current().State != run.Reconciling || !c.RecoveryRequired() || c.Budget().Held.Steps != 1 {
		t.Fatal("missing init lost effect hold")
	}
	stop := run.TransportPreparation{RecordMeta: meta("prepare-stop"), Message: "stop", Control: true, Wire: wire(t, `{"stop":true}`)}
	if p, err := io.PrepareSend(context.Background(), stop); err != nil || !p.Dispatch {
		t.Fatal("bounded shutdown control unavailable", err)
	}
	if _, err := c.StepAdmission("first"); err == nil {
		t.Fatal("unknown first step admitted again")
	}
}

func TestReadinessCannotDischargeGenericStartupInvocationUncertainty(t *testing.T) {
	c, j, io := unboundSession(t, "manager")
	uncertain := command("startup-uncertain", run.ObserveUncertain)
	uncertain.Invocation, uncertain.Reason = "open-s", "startup effect remains uncertain"
	commit(t, c, j, uncertain)
	if _, err := io.Ready(context.Background(), readyObservation(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Propose(settle("open-s", time.Second)); !errors.Is(err, run.ErrState) {
		t.Fatal("readiness alone discharged invocation uncertainty", err)
	}
	if c.Budget().Held.SessionStarts != 1 || c.Current().State != run.Reconciling {
		t.Fatal("uncertain startup hold or state lost")
	}
	c = restored(t, &readJournal{store: j})
	if _, err := c.Propose(settle("open-s", time.Second)); !errors.Is(err, run.ErrState) {
		t.Fatal("restored readiness discharged invocation uncertainty", err)
	}
}

func TestReadinessAppendLossRetainsOriginalStateAndRequiredWireClosure(t *testing.T) {
	for _, mode := range []string{"lost", "absent", "wrong"} {
		t.Run(mode, func(t *testing.T) {
			c, j, io := unboundSession(t, "manager")
			send(t, io, "initialize", "")
			observation := readyObservation(t)
			j.appendMode = mode
			if _, err := io.Ready(context.Background(), observation); !errors.Is(err, run.ErrUncertain) {
				t.Fatal(err)
			}
			if s, _ := c.Session("s"); s.Ready || s.Readiness.BatchID != "" {
				t.Fatal("unconfirmed readiness adopted")
			}
			batch, pending := c.Pending()
			if !pending {
				t.Fatal("lost ack forgotten")
			}
			refs, err := batch.SessionRefs()
			if err != nil || !reflect.DeepEqual(refs, []run.BlobRef{observation.Wire.Journal.Ref()}) {
				t.Fatal("new wire escaped existing closure", refs, err)
			}
			if _, err := c.SessionAdmission("s"); err == nil {
				t.Fatal("lost readiness ack authorized reOpen")
			}
			if _, err := c.Propose(readinessStep(t, c, "first")); err == nil {
				t.Fatal("lost readiness ack authorized Step")
			}
			j.appendMode = ""
			if err := c.ReconcilePending(context.Background(), j); err != nil {
				t.Fatal(err)
			}
			if mode != "absent" {
				if s, _ := c.Session("s"); !s.Ready {
					t.Fatal("committed readiness not reconciled")
				}
				delete(j.blobs, observation.Wire.Journal.Ref())
				if restored, err := run.Restore(context.Background(), &readJournal{store: j}, "repo", restoreLimits); err == nil || restored != nil {
					t.Fatal("missing ready wire restored")
				}
			} else if _, err := c.SessionAdmission("s"); !errors.Is(err, run.ErrObservationLimit) {
				t.Fatal("proven absent record replayed actual startup", err)
			}
		})
	}
}

func TestOpenObservationOutcomesAreExclusiveAndBindCannotConsumeReady(t *testing.T) {
	_, _, io := unboundSession(t, "manager")
	ready := run.SessionObservation{RecordMeta: meta("ready-return"), Ready: true, Wire: wire(t, `{"control":"ready"}`), Reason: "observed control ready"}
	if !ready.Valid() {
		t.Fatal("valid ready Open result rejected")
	}
	if _, err := io.Bind(context.Background(), ready); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("Bind consumed readiness", err)
	}
	for _, mutate := range []func(*run.SessionObservation){func(o *run.SessionObservation) { o.ProviderSession = "requested-uuid" }, func(o *run.SessionObservation) { o.Uncertain = true }, func(o *run.SessionObservation) { o.Wire = run.WirePayload{} }} {
		bad := ready
		mutate(&bad)
		if bad.Valid() {
			t.Fatal("contradictory Open result accepted")
		}
	}
	ready.Ready = false
	if ready.Valid() {
		t.Fatal("empty provider ID inferred readiness")
	}
}

func exploreRunning(t *testing.T, one bool) (*run.Coordinator, *memoryJournal, run.ExplorePreparation) {
	t.Helper()
	inputs := exploreInputs(t, one)
	b, err := run.Bind(inputs.Binding)
	if err != nil {
		t.Fatal(err)
	}
	c, j := runningWith(t, b)
	return c, j, inputs
}
func explorePreparedStep(t *testing.T, c *run.Coordinator, j *memoryJournal, inputs run.ExplorePreparation, manager string, id run.SessionID) (run.SessionIO, run.ExplorePacket, run.StepPlan) {
	t.Helper()
	cmd := exploreSessionPreparation(t, c, inputs, manager, id)
	commit(t, c, j, cmd)
	io, err := c.SessionChannel(j, id)
	if err != nil {
		t.Fatal(err)
	}
	send(t, io, run.MessageID("init-"+string(id)), "")
	_, err = io.Ready(context.Background(), run.SessionReadiness{RecordMeta: meta("ready-" + string(id)), Wire: wire(t, `{"synthetic":"ready"}`), Reason: "synthetic ready"})
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-"+string(id), time.Second))
	plan := run.StepPlan{Session: id, ID: run.StepID("step-" + string(id)), SessionBinding: cmd.Session.Plan.Binding()}
	step, err := c.PrepareExploreStepCommand(meta("prepare-step-"+string(id)), plan, run.Reservation{ID: "invoke-" + string(id), Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, inputs)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, step)
	plan = step.Session.StepPlan
	var packet run.ExplorePacket
	if err := json.Unmarshal(j.blobs[plan.Input].Bytes(), &packet); err != nil {
		t.Fatal(err)
	}
	return io, packet, plan
}
func reportForPacket(p run.ExplorePacket, input run.BlobRef) run.ExploreReport {
	return run.ExploreReport{Version: 1, Assignment: p.Assignment, Run: p.Run, RunBasis: p.RunBasis, Manager: p.Manager, Parent: p.Parent, HandoffID: p.HandoffID, Session: p.Session, SessionBinding: p.SessionBinding, Step: p.Step, InputContentID: input.ContentID, OperationContentID: p.Operation.Product.ContentID, ContextContentID: p.Context.ContentID, Summary: "Inspected complete local area; questions remain explicit.", ProposedChanges: []string{}, Prerequisites: []string{}, Uncertainties: []string{"Semantic uncertainty is not a transport failure."}, Questions: []string{}}
}
func exploreObserveReport(t *testing.T, c *run.Coordinator, j *memoryJournal, io run.SessionIO, p run.ExplorePacket, plan run.StepPlan, edit func(*run.ExploreReport)) {
	t.Helper()
	id := string(p.Session)
	send(t, io, run.MessageID("first-"+id), plan.ID)
	_, err := io.Bind(context.Background(), run.SessionObservation{RecordMeta: meta("bind-" + id), ProviderSession: run.ProviderSessionID("provider-" + id), Wire: wire(t, `{"synthetic":"provider-init"}`), Reason: "synthetic provider observation"})
	if err != nil {
		t.Fatal(err)
	}
	r := reportForPacket(p, plan.Input)
	if edit != nil {
		edit(&r)
	}
	proposal, err := run.NewExploreReportPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("terminal-" + id), Step: plan.ID, ProviderStep: run.ProviderStepID("turn-" + id), Wire: wire(t, `{"synthetic":"terminal"}`), Proposed: proposal, Terminal: true, Reason: "synthetic terminal proposal; no physical provider"})
	if err != nil {
		t.Fatal(err)
	}
}
func exploreStop(t *testing.T, c *run.Coordinator, j *memoryJournal, io run.SessionIO, id run.SessionID) {
	t.Helper()
	commit(t, c, j, settle("invoke-"+string(id), time.Second))
	_, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stop-" + string(id)), Terminated: true, Reason: "synthetic observed stop"})
	if err != nil {
		t.Fatal(err)
	}
}
func TestExploreAllManagersGateUsesActualOrganizationAndNoWorkPermit(t *testing.T) {
	for _, one := range []bool{true, false} {
		t.Run(fmt.Sprint(one), func(t *testing.T) {
			c, j, inputs := exploreRunning(t, one)
			initial := c.InitialExplore()
			if !initial.Available || initial.Clear || initial.ReportsComplete {
				t.Fatal(initial)
			}
			want := 4
			if one {
				want = 2
			}
			if len(initial.Managers) != want {
				t.Fatal("omitted root/empty Manager", initial)
			}
			for n, manager := range initial.Managers {
				if manager.Queued.BatchID == "" || manager.Prepared.BatchID != "" {
					t.Fatal("queued is not delivered", manager)
				}
				id := run.SessionID(fmt.Sprintf("manager-%d", n))
				io, packet, plan := explorePreparedStep(t, c, j, inputs, manager.Manager, id)
				product, operation := organization.ApplyOperation()
				if packet.Operation.Product != product || string(packet.Operation.Bytes) != operation || packet.Context.Package != manager.Manager || packet.Context.ContentID != (snapshot.ContentID(sha256.Sum256(packet.Context.Bytes))) {
					t.Fatal("producer context mismatch")
				}
				selection, err := organization.SelectContext(inputs.Binding.Compilation, manager.Manager, nil)
				if err != nil {
					t.Fatal(err)
				}
				for _, d := range selection.Definitions() {
					if !bytes.Contains(packet.Context.Bytes, []byte(d.Definition.Body)) {
						t.Fatal("compiled body lost", d.Definition.Identity)
					}
				}
				// No report is inferred from a queued/sent input or a prepared context.
				if c.InitialExplore().Managers[n].ReportObserved.BatchID != "" {
					t.Fatal("invented report")
				}
				exploreObserveReport(t, c, j, io, packet, plan, nil)
				if c.InitialExplore().Clear {
					t.Fatal("terminal report left a live provider/hold")
				}
				exploreStop(t, c, j, io, id)
				if n < len(initial.Managers)-1 && c.InitialExplore().Clear {
					t.Fatal("missing Manager bypassed barrier")
				}
			}
			view := c.InitialExplore()
			if !view.ReportsComplete || !view.Clear {
				t.Fatal(view)
			}
			for _, m := range view.Managers {
				if !m.Quiescent || m.Report == nil || len(m.Report.Uncertainties) != 1 {
					t.Fatal(m)
				}
			}
			bare := reserve("bare", run.ToolProcess, run.Assessment, time.Second)
			if _, err := c.Propose(bare); !errors.Is(err, run.ErrState) {
				t.Fatal("clear granted unscoped execution", err)
			}
			writer := sessionPreparation(t, c, "writer")
			if _, err := c.Propose(writer); !errors.Is(err, run.ErrState) {
				t.Fatal("clear granted writer", err)
			}
			reads := &readJournal{store: j}
			restored := restored(t, reads)
			if !reflect.DeepEqual(view, restored.InitialExplore()) || reads.writes != 0 || restored.RecoveryRequired() {
				t.Fatal("restart changed initial census")
			}
			view.Managers[0].Report.Uncertainties[0] = "tampered"
			if c.InitialExplore().Managers[0].Report.Uncertainties[0] == "tampered" {
				t.Fatal("report aliases")
			}
		})
	}
}
func TestExploreWrongOriginAndMissingReportRemainIncomplete(t *testing.T) {
	for _, kind := range []string{"manager", "parent", "run", "input", "operation", "context", "step", "basis"} {
		t.Run(kind, func(t *testing.T) {
			c, j, inputs := exploreRunning(t, true)
			io, p, plan := explorePreparedStep(t, c, j, inputs, ".", "s")
			exploreObserveReport(t, c, j, io, p, plan, func(r *run.ExploreReport) {
				switch kind {
				case "manager":
					r.Manager = "lending"
				case "parent":
					v := "."
					r.Parent = &v
				case "run":
					r.Run = "other"
				case "input":
					r.InputContentID[0]++
				case "operation":
					r.OperationContentID[0]++
				case "context":
					r.ContextContentID[0]++
				case "step":
					r.Step = "other"
				case "basis":
					r.RunBasis[0]++
				}
			})
			exploreStop(t, c, j, io, "s")
			v := c.InitialExplore()
			if v.Clear || v.ReportsComplete || v.Managers[0].Report != nil || v.Managers[0].ReportProblem == "" {
				t.Fatal("mismatched report counted", v)
			}
			if s, _ := c.Step(plan.ID); !s.Terminal || s.Proposed == (run.BlobRef{}) {
				t.Fatal("invalid proposal observation discarded")
			}
		})
	}
}
func TestExploreAdmissionChecksParentReadOnlyCandidateAndCheckedPreparation(t *testing.T) {
	c, j, inputs := exploreRunning(t, false)
	root := exploreSessionPreparation(t, c, inputs, ".", "s")
	for _, kind := range []string{"parent", "writer", "candidate", "cost", "role", "raw"} {
		t.Run(kind, func(t *testing.T) {
			cmd := root
			switch kind {
			case "parent":
				cmd.Manager.Manager = "lending/reminders"
			case "writer":
				cmd.Session.Plan.ReadOnly = false
			case "candidate":
				cmd.Session.Plan.Candidate[0]++
			case "cost":
				cmd.Reservation.Purpose = run.Assessment
			case "role":
				cmd.Session.Plan.Role = "reviewer"
			case "raw":
				cmd = run.PrepareSessionCommand(meta("raw"), "a", "r", root.Session.Plan, root.Reservation, j.blobs[root.Session.Plan.Profile], j.blobs[root.Session.Plan.Tools])
			}
			if _, err := c.Propose(cmd); err == nil {
				t.Fatal("accepted unclassified/mutated preparation")
			}
		})
	}
	commit(t, c, j, root)
	// The live constructor refuses a duplicate slot, even under a new ID.
	if again, err := c.PrepareExploreSessionCommand(meta("again"), ".", root.Session.Plan, root.Reservation, payload(t, "profile"), payload(t, "tools")); err == nil {
		if _, err = c.Propose(again); err == nil {
			t.Fatal("duplicate slot")
		}
	}
	// A child cannot skip its parent's attributed initial ingress.
	profile, tools := payload(t, "profile"), payload(t, "tools")
	plan := root.Session.Plan
	plan.ID = "child"
	if child, err := c.PrepareExploreSessionCommand(meta("child"), "lending", plan, run.Reservation{ID: "child", Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Second}, profile, tools); err == nil {
		if _, err = c.Propose(child); err == nil {
			t.Fatal("child shortcut")
		}
	}
}
func TestExplorePreparationRecompilesOriginalsAndMatchesExactBoundSettings(t *testing.T) {
	c, j, inputs := exploreRunning(t, true)
	cmd := exploreSessionPreparation(t, c, inputs, ".", "s")
	commit(t, c, j, cmd)
	io, _ := c.SessionChannel(j, "s")
	send(t, io, "init", "")
	_, err := io.Ready(context.Background(), readyObservation(t))
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	for _, kind := range []string{"model", "settings", "basis", "context-limit"} {
		t.Run(kind, func(t *testing.T) {
			altered := inputs
			switch kind {
			case "model":
				altered.Binding.ModelInput.Root.Entries = append([]model.Entry(nil), inputs.Binding.ModelInput.Root.Entries...)
				altered.Binding.ModelInput.Root.Entries[1].Bytes = []byte("id: changed\n")
			case "settings":
				altered.Settings.Defaults.RejectionLimit.Value = 3
			case "basis":
				altered.Binding.Source.Guard = "other"
			case "context-limit":
				altered.Binding.MaxContextBytes.Value = 1
			}
			_, err := c.PrepareExploreStepCommand(meta("bad-"+kind), run.StepPlan{Session: "s", ID: "first", SessionBinding: cmd.Session.Plan.Binding()}, run.Reservation{ID: "first", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Second}, altered)
			if err == nil {
				t.Fatal("accepted unchecked changed producer input")
			}
		})
	}
}
func TestExploreV3DemandIncludesRootEmptyAndTwoPhysicalReviewPhases(t *testing.T) {
	for _, one := range []bool{true, false} {
		inputs := exploreInputs(t, one)
		want := int64(6)
		if one {
			want = 3
		}
		b, err := run.Bind(inputs.Binding)
		if err != nil || b.RequiredReviews() != want {
			t.Fatal(b.RequiredReviews(), err)
		}
		for _, kind := range []string{"starts", "steps"} {
			bad := inputs.Binding
			if kind == "starts" {
				bad.Policy.Reserve.SessionStarts = 2*want - 1
			} else {
				bad.Policy.Reserve.Steps = 2*want - 1
			}
			if _, err := run.Bind(bad); !errors.Is(err, run.ErrInvalid) {
				t.Fatal("underfunded review phases accepted", kind, err)
			}
		}
	}
}

func TestExploreBindsScopeLimitOwnerAndArtifactInputs(t *testing.T) {
	var facts organization.SettingsFacts
	input := fixtureWith(t, func(m *model.Input) {
		for _, d := range []struct{ path, text string }{
			{"model/lending/applies.yaml", "imports: [{package: markitect/core, as: Core}]\nid: applies\nkind: Core.Property\nbody: Explicit modeled scope.\nspec: {type: string, minCount: 0, maxCount: -1}\n"},
			{"model/lending/scoped-rule.yaml", "imports: [{package: markitect/core, as: Core}]\nid: ScopedRule\nkind: Core.Kind\nbody: Scoped obligation type.\nspec: {properties: [applies]}\n"},
			{"model/lending/reminders/global.yaml", "imports: [{package: lending}]\nid: global\nkind: lending/ScopedRule\nbody: Global scoped initial assessment obligation.\nspec: {applies: ['.']}\n"},
		} {
			m.Root.Entries = append(m.Root.Entries, model.Entry{Path: d.path, Kind: model.Regular, Bytes: []byte(d.text)})
		}
	}, func(f *organization.SettingsFacts) { facts = *f })
	input.Scope = []organization.Setting[organization.ScopeRule]{{Value: organization.ScopeRule{Kind: model.Identity{Package: "lending", ID: "ScopedRule"}, Property: model.Identity{Package: "lending", ID: "applies"}}, Source: source(input.Documents[0], "/scope/0")}}
	b, err := run.Bind(input)
	if err != nil {
		t.Fatal(err)
	}
	bytesBefore := b.Payload().Bytes()
	for _, kind := range []string{"scope-source", "scope-value", "limit-value", "limit-source", "missing-source", "missing-limit", "nil-scope", "empty-scope"} {
		t.Run(kind, func(t *testing.T) {
			changed := input
			changed.Scope = append([]organization.Setting[organization.ScopeRule](nil), input.Scope...)
			invalid := false
			switch kind {
			case "scope-source":
				changed.Scope[0].Source = source(input.Documents[1], "/scope/0")
			case "scope-value":
				changed.Scope[0].Value.Property.ID = "absent"
				invalid = true
			case "limit-value":
				changed.MaxContextBytes.Value++
			case "limit-source":
				changed.MaxContextBytes.Source = source(input.Documents[1], "/limits/maxContextBytes")
			case "missing-source":
				changed.Scope[0].Source.ContentID[0]++
				invalid = true
			case "missing-limit":
				changed.MaxContextBytes = organization.Setting[int64]{}
				invalid = true
			case "nil-scope":
				changed.Scope = nil
			case "empty-scope":
				changed.Scope = []organization.Setting[organization.ScopeRule]{}
			}
			next, err := run.Bind(changed)
			if invalid {
				if err == nil {
					t.Fatal("invalid source/selection accepted")
				}
			} else if err != nil || next.Ref() == b.Ref() {
				t.Fatal("selected scope/limit not in basis", err)
			}
		})
	}
	c, j := runningWith(t, b)
	inputs := run.ExplorePreparation{Binding: input, Settings: facts}
	io, p, plan := explorePreparedStep(t, c, j, inputs, ".", "s")
	if !bytes.Contains(p.Context.Bytes, []byte("Global scoped initial assessment obligation.")) || !bytes.Contains(p.Context.Bytes, input.OwnerRequest.Bytes()) {
		t.Fatal("scoped model or actual root request omitted")
	}
	changed := inputs
	changed.Binding.Scope = nil
	if _, err := c.PrepareExploreStepCommand(meta("substitute-scope"), plan, run.Reservation{ID: "substitute", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Second}, changed); err == nil {
		t.Fatal("scope replaced under unchanged basis")
	}
	input.Scope = append([]organization.Setting[organization.ScopeRule](nil), input.Scope...)
	input.Scope[0].Value.Kind.ID = "mutated"
	if !bytes.Equal(bytesBefore, b.Payload().Bytes()) {
		t.Fatal("scope aliases basis")
	}
	exploreObserveReport(t, c, j, io, p, plan, nil)
	exploreStop(t, c, j, io, "s")
	// The owning child receives the existing registered artifact's actual bytes.
	io, p, plan = explorePreparedStep(t, c, j, inputs, "lending", "child")
	if !bytes.Contains(p.Context.Bytes, []byte("package library\n")) {
		t.Fatal("own artifact omitted from Explore context")
	}
	_ = io
	_ = plan
}

func TestExploreTerminalReportNeedsExactAcknowledgementAndQuiescence(t *testing.T) {
	c, j, inputs := exploreRunning(t, true)
	io, p, plan := explorePreparedStep(t, c, j, inputs, ".", "s")
	send(t, io, "first", plan.ID)
	if _, err := io.Bind(context.Background(), run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider", Wire: wire(t, "provider init"), Reason: "synthetic provider"}); err != nil {
		t.Fatal(err)
	}
	proposal, err := run.NewExploreReportPayload(reportForPacket(p, plan.Input))
	if err != nil {
		t.Fatal(err)
	}
	observation := run.StepObservation{RecordMeta: meta("terminal"), Step: plan.ID, ProviderStep: "turn", Wire: wire(t, "terminal"), Proposed: proposal, Terminal: true, Reason: "synthetic terminal"}
	before := c.InitialExplore()
	j.appendMode = "lost"
	if _, err := io.ObserveStep(context.Background(), observation); !errors.Is(err, run.ErrUncertain) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, c.InitialExplore()) {
		t.Fatal("unacknowledged report adopted")
	}
	j.appendMode = ""
	recovered := restored(t, &readJournal{store: j})
	if recovered.InitialExplore().Managers[0].ReportObserved.BatchID == "" || recovered.InitialExplore().Clear || !recovered.RecoveryRequired() {
		t.Fatal("restart lost known commit or invented quiescence")
	}
	if err := c.ReconcilePending(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if c.InitialExplore().Managers[0].ReportObserved.BatchID == "" || c.InitialExplore().Clear {
		t.Fatal("reconciliation did not adopt only confirmed report")
	}
	calls, budget := j.appendCalls, c.Budget()
	if _, err := io.ObserveStep(context.Background(), observation); err != nil {
		t.Fatal(err)
	}
	if j.appendCalls != calls || c.Budget() != budget {
		t.Fatal("duplicate report appended/charged")
	}
	exploreStop(t, c, j, io, "s")
	if c.InitialExplore().Clear {
		t.Fatal("missing child omitted")
	}
}
func TestExploreUncertainOrMissingOutputNeverCompletesInitialCensus(t *testing.T) {
	for _, kind := range []string{"uncertain", "missing", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			c, j, inputs := exploreRunning(t, true)
			io, _, plan := explorePreparedStep(t, c, j, inputs, ".", "s")
			send(t, io, "first", plan.ID)
			if _, err := io.Bind(context.Background(), run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider", Wire: wire(t, "init"), Reason: "provider observation"}); err != nil {
				t.Fatal(err)
			}
			o := run.StepObservation{RecordMeta: meta("result"), Step: plan.ID, ProviderStep: "turn", Wire: wire(t, "result"), Terminal: true, Reason: "observed output limit"}
			if kind == "uncertain" {
				o.Uncertain = true
				o.Terminal = false
			} else if kind == "malformed" {
				o.Proposed = payload(t, `{"invented":"success"}`)
			}
			if _, err := io.ObserveStep(context.Background(), o); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stop"), Terminated: true, Reason: "Host observed stop"}); err != nil {
				t.Fatal(err)
			}
			commit(t, c, j, settle("invoke-s", time.Second))
			v := c.InitialExplore()
			if v.Clear || v.ReportsComplete || v.Managers[0].ReportObserved.BatchID != "" {
				t.Fatal("failed participation completed", v)
			}
		})
	}
}
func TestExploreUnexpectedToolIngressIsRecordedWithoutExecutionAuthority(t *testing.T) {
	c, j, io := activeStep(t)
	stop := command("stop", run.StopRun)
	stop.Reason = "stop requested"
	commit(t, c, j, stop)
	call := run.ToolCall{RecordMeta: meta("late-known-step-call"), Step: "step", Call: "mutating", Tool: "write-workspace", Input: wire(t, `{"path":"owned.go"}`)}
	receipt, err := io.ToolRequest(context.Background(), call)
	if err != nil || !receipt.Fresh || receipt.Receipt.BatchID == "" {
		t.Fatal("known incoming callback was lost", err)
	}
	if _, err := c.Propose(reserve("execute", run.ToolProcess, run.Writing, time.Second)); err == nil {
		t.Fatal("Fresh granted execution")
	}
	if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stopped"), Terminated: true, Reason: "observed stop"}); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("step-invocation", time.Second))
	c = restored(t, &readJournal{store: j})
	if !c.RecoveryRequired() || c.InitialExplore().Clear {
		t.Fatal("unresolved callback disappeared at restart")
	}
}
