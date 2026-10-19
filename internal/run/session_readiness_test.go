package run_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

func unboundSession(t *testing.T, role string) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j := running(t)
	cmd := sessionPreparation(t, c, "s")
	if role == "reviewer" {
		cmd.Session.Plan.Role, cmd.Session.Plan.ReadOnly, cmd.Reservation.Purpose = role, true, run.Assessment
	}
	commit(t, c, j, cmd)
	io, err := c.SessionChannel(j, "s", run.RefuseTools)
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
	t.Parallel()
	for _, role := range []string{"manager"} {
		t.Run(role, func(t *testing.T) {
			c, j, io := readySession(t, role, true)
			s, _ := c.Session("s")
			if !s.Ready || s.Bound || s.Provider != "" || s.Readiness.BatchID == "" || s.Binding.BatchID != "" {
				t.Fatal("readiness invented binding", s)
			}
			if _, err := c.SessionAdmission(context.Background(), j, "s"); err == nil {
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
			if _, err := c.StepAdmission(context.Background(), j, "first"); err != nil {
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
			if _, err := c.StepAdmission(context.Background(), j, "first"); err == nil {
				t.Fatal("first step dispatch reissued after binding")
			}
			if _, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("first-result"), Step: "first", ProviderStep: "observed-turn", Wire: wire(t, `{"result":"done"}`), Terminal: true, Reason: "actual result"}); err != nil {
				t.Fatal(err)
			}
			commit(t, c, j, settle("invoke-first", time.Second))
			if _, err := c.Propose(readinessStep(t, c, "second")); !errors.Is(err, run.ErrState) {
				t.Fatal("the initial step granted a later phase", err)
			}
		})
	}
}

func TestReadinessNeedsObservedAckAndSettledManagerStartup(t *testing.T) {
	t.Parallel()
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
			if _, err := c.SessionAdmission(context.Background(), j, "s"); !errors.Is(err, run.ErrObservationLimit) {
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
	t.Parallel()
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
					io, err = c.SessionChannel(j, "s", run.RefuseTools)
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
				if _, err := c.SessionAdmission(context.Background(), j, "s"); err == nil {
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
	t.Parallel()
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
	t.Parallel()
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
	io, err := c.SessionChannel(j, "s", run.RefuseTools)
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
	if _, err := c.StepAdmission(context.Background(), j, "first"); err == nil {
		t.Fatal("unknown first step admitted again")
	}
}

func TestReadinessCannotDischargeGenericStartupInvocationUncertainty(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
			if _, err := c.SessionAdmission(context.Background(), j, "s"); err == nil {
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
			} else if _, err := c.SessionAdmission(context.Background(), j, "s"); !errors.Is(err, run.ErrObservationLimit) {
				t.Fatal("proven absent record replayed actual startup", err)
			}
		})
	}
}

func TestOpenObservationOutcomesAreExclusiveAndBindCannotConsumeReady(t *testing.T) {
	t.Parallel()
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
