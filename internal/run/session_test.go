package run_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func meta(id string) run.RecordMeta { return run.RecordMeta{ID: id, ObservedAt: observed} }
func currentSessionBinding(c *run.Coordinator) snapshot.ContentID {
	v, _ := c.Session("s")
	return v.Plan.Binding()
}
func payload(t *testing.T, data string) run.Payload {
	t.Helper()
	p, err := run.NewPayload("application/json", "utf-8", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func wire(t *testing.T, data string) run.WirePayload {
	p := payload(t, data)
	return run.WirePayload{Identity: run.WireIdentity{ContentID: p.Ref().ContentID, Size: p.Ref().Size}, Journal: p}
}
func sessionPreparation(t *testing.T, id run.SessionID) run.Command {
	profile, tools := payload(t, `{"model":"configured","credentials":"credential-ref"}`), payload(t, `{"host-tool":"supplied-schema"}`)
	plan := run.SessionPlan{ID: id, Role: "manager", Workspace: "linux-workspace", Candidate: snapshot.TreeID{1}, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 4096, Profile: profile.Ref(), Tools: tools.Ref()}
	r := run.Reservation{ID: "open-" + string(id), Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Minute}
	return run.PrepareSessionCommand(meta("prepare-"+string(id)), "a", "r", plan, r, profile, tools)
}
func preparedSession(t *testing.T) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j := running(t)
	commit(t, c, j, sessionPreparation(t, "s"))
	io, err := c.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	return c, j, io
}
func send(t *testing.T, io run.SessionIO, id run.MessageID, step run.StepID) {
	t.Helper()
	p, err := io.PrepareSend(context.Background(), run.TransportPreparation{RecordMeta: meta("prepare-wire-" + string(id)), Message: id, Step: step, Wire: wire(t, `{"request":"known"}`)})
	if err != nil || !p.Dispatch || p.Receipt.BatchID == "" {
		t.Fatal("not durably admitted", p, err)
	}
	_, err = io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("sent-" + string(id)), Message: id, Outcome: run.Sent, Reason: "observed write"})
	if err != nil {
		t.Fatal(err)
	}
}
func boundSession(t *testing.T) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j, io := preparedSession(t)
	send(t, io, "initialize", "")
	_, err := io.Bind(context.Background(), run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider-session", Wire: wire(t, `{"init":"provider-session"}`), Reason: "observed init"})
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	return c, j, io
}
func activeStep(t *testing.T) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j, io := boundSession(t)
	input, contract := payload(t, `{"compiled":"turn"}`), payload(t, `{"result":"proposal-only"}`)
	plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "step", Input: input.Ref(), OutputContract: contract.Ref()}
	commit(t, c, j, run.PrepareStepCommand(meta("prepare-step"), "a", "r", plan, run.Reservation{ID: "step-invocation", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, input, contract))
	send(t, io, "turn", "step")
	_, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("step-started"), Step: "step", ProviderStep: "provider-turn", Wire: wire(t, `{"turn":"started"}`), Reason: "known turn"})
	if err != nil {
		t.Fatal(err)
	}
	return c, j, io
}

func TestSessionAdmissionAndReferencesCommitAtomically(t *testing.T) {
	c, j := running(t)
	cmd := sessionPreparation(t, "s")
	before := c.Budget()
	p, err := c.Propose(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if c.Budget() != before {
		t.Fatal("proposal consumed startup")
	}
	events := p.Batch().Events()
	if len(events) != 2 || events[0].Kind != run.ReserveInvocation || events[1].Kind != run.PrepareSession {
		t.Fatal("admission not atomic", events)
	}
	refs, err := p.Batch().SessionRefs()
	if err != nil || len(refs) != 2 {
		t.Fatal("profile/tool closure", refs, err)
	}
	// A physical adapter cannot acknowledge a batch whose required blob is
	// missing or corrupt. This process-local fake claims no fsync durability.
	if _, err = j.Append(context.Background(), p.Batch()); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("missing payload acknowledged", err)
	}
	for _, ref := range refs {
		j.blobs[ref] = payload(t, "wrong bytes")
	}
	if _, err = j.Append(context.Background(), p.Batch()); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("corrupt payload acknowledged", err)
	}
	for _, ref := range refs {
		delete(j.blobs, ref)
	}
	commit(t, c, j, cmd)
	if c.Budget().Held.SessionStarts != 1 {
		t.Fatal("startup not held")
	}
	a, err := c.SessionAdmission("s")
	if err != nil || a.Plan.ID != "s" || a.Receipt.BatchID == "" {
		t.Fatal(a, err)
	}
	if _, err := c.Inspection("s", "missing"); !errors.Is(err, run.ErrObservationLimit) {
		t.Fatal("invented provider handles", err)
	}
	refs[0] = run.BlobRef{}
	again, _ := p.Batch().SessionRefs()
	if !again[0].Valid() {
		t.Fatal("reference slice aliases")
	}
}

func TestSendRequiresExactAcknowledgementAndCannotReplay(t *testing.T) {
	for _, mode := range []string{"absent", "lost", "wrong"} {
		t.Run(mode, func(t *testing.T) {
			c, j, io := preparedSession(t)
			j.appendMode = mode
			prep := run.TransportPreparation{RecordMeta: meta("prepare-wire"), Message: "m", Wire: wire(t, `{"send":"original"}`)}
			permit, err := io.PrepareSend(context.Background(), prep)
			if err == nil || permit.Dispatch {
				t.Fatal("failed acknowledgement authorized send", permit, err)
			}
			if c.Budget().Held.SessionStarts != 1 {
				t.Fatal("callback failure released hold")
			}
			if _, pending := c.Pending(); !pending {
				t.Fatal("uncertain callback disappeared")
			}
			j.appendMode = ""
			if err := c.ReconcilePending(context.Background(), j); err != nil {
				t.Fatal(err)
			}
			if mode == "absent" {
				return
			} // Proven absent: no external call was allowed.
			permit, err = io.PrepareSend(context.Background(), prep)
			if !errors.Is(err, run.ErrObservationLimit) || permit.Dispatch || permit.Receipt.BatchID == "" {
				t.Fatal("known preparation resent", permit, err)
			}
			c2 := restored(t, &readJournal{store: j})
			if !c2.RecoveryRequired() {
				t.Fatal("recovery forgot possible dispatch")
			}
			io2, err := c2.SessionChannel(j, "s")
			if err != nil {
				t.Fatal(err)
			}
			permit, err = io2.PrepareSend(context.Background(), prep)
			if !errors.Is(err, run.ErrObservationLimit) || permit.Dispatch {
				t.Fatal("restoration authorized resend", err)
			}
		})
	}
}

func TestLiveSessionAndUnknownStopBlockFinishEvenAfterBudgetSettlement(t *testing.T) {
	c, j, io := boundSession(t)
	f := command("finish", run.FinishRun)
	f.Outcome = run.Incomplete
	f.Reason = "retain work"
	if _, err := c.Propose(f); !errors.Is(err, run.ErrState) {
		t.Fatal("settled startup meant stopped session", err)
	}
	_, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("cancel"), Reason: "cancellation requested; termination unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Propose(f); !errors.Is(err, run.ErrState) {
		t.Fatal("cancellation meant termination", err)
	}
	c2 := restored(t, &readJournal{store: j})
	if !c2.RecoveryRequired() {
		t.Fatal("live provider forgotten")
	}
	io2, err := c2.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io2.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("terminated"), Terminated: true, Reason: "Host observed container/provider terminated"})
	if err != nil {
		t.Fatal(err)
	}
	if c2.RecoveryRequired() {
		t.Fatal("known termination did not clear observation barrier")
	}
	commit(t, c2, j, f)
}

func TestChannelBoundToolDedupRetainsPendingAndExistingReply(t *testing.T) {
	c, j, io := activeStep(t)
	call := run.ToolCall{RecordMeta: meta("tool-ingress"), Step: "step", Call: "provider-call", Tool: "supplied-host-tool", Input: wire(t, `{"read":"target"}`)}
	r, err := io.ToolRequest(context.Background(), call)
	if err != nil || !r.Fresh {
		t.Fatal("ingress not recorded", r, err)
	}
	before, calls := c.Revision(), j.appendCalls
	call.RecordMeta = meta("duplicate-ingress")
	r, err = io.ToolRequest(context.Background(), call)
	if !errors.Is(err, run.ErrObservationLimit) || r.Fresh || r.Replied {
		t.Fatal("unresolved callback re-executed", r, err)
	}
	if c.Revision() != before || j.appendCalls != calls {
		t.Fatal("duplicate appended")
	}
	changed := call
	changed.Input = wire(t, `{"read":"different"}`)
	if _, err := io.ToolRequest(context.Background(), changed); !errors.Is(err, run.ErrConflict) {
		t.Fatal("call identity changed content", err)
	}
	reply := run.ToolReply{RecordMeta: meta("reply"), Call: call.Call, Output: wire(t, `{"result":"known reply"}`)}
	receipt, err := io.ToolReply(context.Background(), reply)
	if err != nil {
		t.Fatal(err)
	}
	r, err = io.ToolRequest(context.Background(), call)
	if err != nil || !r.Replied || r.Fresh || r.Receipt != receipt || !bytes.Equal(r.Payload.Bytes(), reply.Output.Journal.Bytes()) {
		t.Fatal("existing reply not reused", r, err)
	}
	c2 := restored(t, &readJournal{store: j})
	io2, err := c2.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	r, err = io2.ToolRequest(context.Background(), call)
	if err != nil || r.Fresh || r.Receipt != receipt {
		t.Fatal("restart lost tool dedup", r, err)
	}
	inspection, err := c2.Inspection("s", "step")
	if err != nil || inspection.ProviderSession != "provider-session" || inspection.ProviderStep != "provider-turn" || inspection.SessionBinding.BatchID == "" || inspection.StepBinding.BatchID == "" {
		t.Fatal("inspection not exact", inspection, err)
	}
	if _, err := c2.Inspection("s", "other-step"); !errors.Is(err, run.ErrObservationLimit) {
		t.Fatal("inspection guessed turn", err)
	}
}

func TestRedactedWireStoresPermittedRepresentationWithoutReplayAuthority(t *testing.T) {
	c, j, io := activeStep(t)
	call := run.ToolCall{RecordMeta: meta("call"), Step: "step", Call: "call", Tool: "schema", Input: wire(t, `{"request":"nonsecret"}`)}
	if _, err := io.ToolRequest(context.Background(), call); err != nil {
		t.Fatal(err)
	}
	// The supplied hash is the identity of a hypothetical real wire, not a
	// BlobRef. No secret value is constructed or supplied to pure RUN.
	identity := run.WireIdentity{ContentID: snapshot.ContentID(sha256.Sum256([]byte("opaque wire identity fixture"))), Size: 80}
	redacted := run.WirePayload{Identity: identity, Journal: payload(t, `{"credential":"[REDACTED]"}`), Redacted: true, ObservationLimit: "Host redacted known credential before append; original bytes unavailable for replay"}
	_, err := io.ToolReply(context.Background(), run.ToolReply{RecordMeta: meta("reply"), Call: "call", Output: redacted})
	if err != nil {
		t.Fatal(err)
	}
	refs, err := j.batches[len(j.batches)-1].SessionRefs()
	if err != nil || len(refs) != 1 || refs[0] != redacted.Journal.Ref() || refs[0].ContentID == identity.ContentID {
		t.Fatal("wire identity confused with stored bytes", refs, err)
	}
	r, err := io.ToolRequest(context.Background(), call)
	if !errors.Is(err, run.ErrObservationLimit) || !r.Replied || r.Payload.Valid() || r.Reply.Identity != identity {
		t.Fatal("redacted reply became original wire", r, err)
	}
	c2 := restored(t, &readJournal{store: j})
	if v, _ := c2.Session("s"); v.Plan.CredentialRef != "credential-ref" {
		t.Fatal("credential reference lost")
	}
	if c.Revision() != c2.Revision() {
		t.Fatal("redacted history not restored")
	}
}

func TestTerminalStepAndStoppedSessionStillRetainUnresolvedToolActivity(t *testing.T) {
	c, j, io := activeStep(t)
	call := run.ToolCall{RecordMeta: meta("tool"), Step: "step", Call: "call", Tool: "schema", Input: wire(t, `{"request":"bounded"}`)}
	if _, err := io.ToolRequest(context.Background(), call); err != nil {
		t.Fatal(err)
	}
	terminal := run.StepObservation{RecordMeta: meta("terminal"), Step: "step", ProviderStep: "provider-turn", Wire: wire(t, `{"terminal":true}`), Terminal: true, Reason: "provider result"}
	if _, err := io.ObserveStep(context.Background(), terminal); !errors.Is(err, run.ErrState) {
		t.Fatal("provider final cleared pending tool", err)
	}
	if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stop"), Terminated: true, Reason: "observed termination"}); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("step-invocation", time.Second))
	f := command("finish", run.FinishRun)
	f.Reason = "retain"
	f.Outcome = run.Incomplete
	if _, err := c.Propose(f); !errors.Is(err, run.ErrState) {
		t.Fatal("provider stop cleared Host tool uncertainty", err)
	}
	if _, err := io.ToolReply(context.Background(), run.ToolReply{RecordMeta: meta("tool-reconciled"), Call: "call", Output: wire(t, `{"observation":"proven not executed"}`)}); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, f)
	c2 := restored(t, &readJournal{store: j})
	if c2.RecoveryRequired() || !reflect.DeepEqual(c.Assignments(), c2.Assignments()) {
		t.Fatal("terminal reconstruction changed facts")
	}
}

func TestSessionAdmissionsRefuseAfterStopRequest(t *testing.T) {
	for _, kind := range []string{"session", "step"} {
		t.Run(kind, func(t *testing.T) {
			var c *run.Coordinator
			var j *memoryJournal
			if kind == "session" {
				c, j, _ = preparedSession(t)
			} else {
				c, j, _ = boundSession(t)
				input, contract := payload(t, "input"), payload(t, "contract")
				plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "unstarted", Input: input.Ref(), OutputContract: contract.Ref()}
				commit(t, c, j, run.PrepareStepCommand(meta("step-prep"), "a", "r", plan, run.Reservation{ID: "step-hold", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, input, contract))
			}
			stop := command("stop", run.StopRun)
			stop.Reason = "owner stop"
			commit(t, c, j, stop)
			var err error
			if kind == "session" {
				_, err = c.SessionAdmission("s")
			} else {
				_, err = c.StepAdmission("unstarted")
			}
			if !errors.Is(err, run.ErrState) {
				t.Fatal("stop request allowed new external launch", err)
			}
		})
	}
}

func TestRecoveryAllowsBoundedStopSendWithoutClearingUnknownOrdinarySend(t *testing.T) {
	c, j, io := activeStep(t)
	p := run.TransportPreparation{RecordMeta: meta("uncertain-send"), Message: "uncertain", Step: "step", Wire: wire(t, "possible outbound notification")}
	if _, err := io.PrepareSend(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("send-unknown"), Message: p.Message, Outcome: run.SendUncertain, Reason: "connection failed after possible write"}); err != nil {
		t.Fatal(err)
	}
	c2 := restored(t, &readJournal{store: j})
	before := c2.Budget()
	io2, err := c2.SessionChannel(j, "s")
	if err != nil {
		t.Fatal(err)
	}
	control := run.TransportPreparation{RecordMeta: meta("stop-wire"), Message: "stop", Step: "step", Control: true, Wire: wire(t, "bounded interrupt")}
	permit, err := io2.PrepareSend(context.Background(), control)
	if err != nil || !permit.Dispatch {
		t.Fatal("uncertain prior send blocked termination control", permit, err)
	}
	if !c2.RecoveryRequired() || c2.Budget() != before || c2.Budget().Held.Steps != 1 {
		t.Fatal("control erased uncertainty or charged new work")
	}
	request, err := c2.StopRequest("s", "bounded termination")
	if err != nil || request.Timeout != fixture(t).Policy.HostOperationTimeout {
		t.Fatal(request, err)
	}
	if _, err := io2.ObserveStop(context.Background(), run.StopObservation{RecordMeta: meta("stopped"), Terminated: true, Reason: "observed container termination"}); err != nil {
		t.Fatal(err)
	}
	if !c2.RecoveryRequired() {
		t.Fatal("termination released elapsed hold without observation")
	}
	commit(t, c2, j, settle("step-invocation", time.Second))
	if c2.RecoveryRequired() {
		t.Fatal("known quiescence did not clear recovery")
	}
	_ = c
}

func TestUncertainStepRejectsMalformedPartialProviderHandle(t *testing.T) {
	c, j, io := boundSession(t)
	input, contract := payload(t, "input"), payload(t, "contract")
	plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "step", Input: input.Ref(), OutputContract: contract.Ref()}
	commit(t, c, j, run.PrepareStepCommand(meta("prepare-step"), "a", "r", plan, run.Reservation{ID: "step-effect", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, input, contract))
	before := c.Revision()
	_, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("unknown"), Step: "step", ProviderStep: "bad\x00handle", Uncertain: true, Reason: "incomplete provider observation"})
	if !errors.Is(err, run.ErrInvalid) || c.Revision() != before {
		t.Fatal("invalid partial handle persisted", err)
	}
	if _, err := c.Inspection("s", "step"); !errors.Is(err, run.ErrObservationLimit) {
		t.Fatal("invalid handle became usable", err)
	}
}

func TestStepRequiresExplicitMatchingSessionPlanBinding(t *testing.T) {
	c, j, _ := boundSession(t)
	input, contract := payload(t, "input"), payload(t, "contract")
	v, _ := c.Session("s")
	changed := v.Plan
	changed.Workspace = "different-workspace"
	plan := run.StepPlan{Session: "s", ID: "step", SessionBinding: changed.Binding(), Input: input.Ref(), OutputContract: contract.Ref()}
	before := c.Budget()
	cmd := run.PrepareStepCommand(meta("step-prep"), "a", "r", plan, run.Reservation{ID: "step-hold", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, input, contract)
	if _, err := c.Propose(cmd); !errors.Is(err, run.ErrState) || c.Budget() != before {
		t.Fatal("changed session basis continued", err)
	}
	plan.SessionBinding = v.Plan.Binding()
	commit(t, c, j, run.PrepareStepCommand(meta("step-prep"), "a", "r", plan, cmd.Reservation, input, contract))
	admission, err := c.StepAdmission("step")
	if err != nil || admission.Session != v.Plan || admission.Plan.SessionBinding != v.Plan.Binding() {
		t.Fatal("admission lost fixed plan", admission, err)
	}
}

func TestSessionLimitsAndRestoreRefuseMissingOrCorruptRequiredBytes(t *testing.T) {
	c, j, io := preparedSession(t)
	bad := sessionPreparation(t, "other")
	bad.Session.Plan.Profile = run.BlobRef{}
	if _, err := c.Propose(bad); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("missing mandatory reference accepted", err)
	}
	if refs, err := (run.AppendBatch{}).SessionRefs(); err == nil || len(refs) != 0 {
		t.Fatal("invalid batch exposed partial closure", err)
	}
	oversize := run.TransportPreparation{RecordMeta: meta("oversize"), Message: "large", Wire: wire(t, string(bytes.Repeat([]byte("x"), 4097)))}
	if p, err := io.PrepareSend(context.Background(), oversize); !errors.Is(err, run.ErrInvalid) || p.Dispatch {
		t.Fatal("profile wire bound ignored", err)
	}
	v, _ := c.Session("s")
	original := j.blobs[v.Plan.Profile]
	delete(j.blobs, v.Plan.Profile)
	if restored, err := run.Restore(context.Background(), &readJournal{store: j}, "repo", restoreLimits); err == nil || restored != nil {
		t.Fatal("missing required session bytes restored")
	}
	j.blobs[v.Plan.Profile] = payload(t, "corrupt")
	if restored, err := run.Restore(context.Background(), &readJournal{store: j}, "repo", restoreLimits); !errors.Is(err, run.ErrInvalid) || restored != nil {
		t.Fatal("corrupt required session bytes restored", err)
	}
	j.blobs[v.Plan.Profile] = original
}

// portFake exercises the four provider-neutral operations through durable Host
// callbacks. It starts no runtime/provider and establishes no real handshake.
type portFake struct {
	t     *testing.T
	calls []string
}

func (f *portFake) Open(ctx context.Context, a run.SessionAdmission, io run.SessionIO) (run.SessionObservation, error) {
	f.calls = append(f.calls, "Open")
	if a.Receipt.BatchID == "" || a.Plan.ID != "s" {
		return run.SessionObservation{}, run.ErrInvalid
	}
	send(f.t, io, "initialize", "")
	o := run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider-session", Wire: wire(f.t, "provider init"), Reason: "observed init"}
	_, err := io.Bind(ctx, o)
	return o, err
}
func (f *portFake) Step(ctx context.Context, a run.StepAdmission, io run.SessionIO) (run.StepObservation, error) {
	f.calls = append(f.calls, "Step")
	if a.Receipt.BatchID == "" || a.Plan.ID != "step" {
		return run.StepObservation{}, run.ErrInvalid
	}
	send(f.t, io, "turn", "step")
	o := run.StepObservation{RecordMeta: meta("terminal"), Step: "step", ProviderStep: "provider-turn", Wire: wire(f.t, "terminal"), Proposed: payload(f.t, "proposal"), Terminal: true, Reason: "observed terminal turn"}
	_, err := io.ObserveStep(ctx, o)
	return o, err
}
func (f *portFake) Inspect(_ context.Context, r run.InspectionRequest) (run.StepObservation, error) {
	f.calls = append(f.calls, "Inspect")
	if r.ProviderSession != "provider-session" || r.ProviderStep != "provider-turn" || r.SessionBinding.BatchID == "" || r.StepBinding.BatchID == "" {
		return run.StepObservation{}, run.ErrObservationLimit
	}
	return run.StepObservation{Step: r.Step, ProviderStep: r.ProviderStep, Terminal: true}, nil
}
func (f *portFake) Stop(ctx context.Context, r run.StopRequest, io run.SessionIO) (run.StopObservation, error) {
	f.calls = append(f.calls, "Stop")
	if r.Timeout <= 0 {
		return run.StopObservation{}, run.ErrInvalid
	}
	o := run.StopObservation{RecordMeta: meta("stopped"), Terminated: true, Reason: "fake Host observed provider termination"}
	_, err := io.ObserveStop(ctx, o)
	return o, err
}

var _ run.Sessions = (*portFake)(nil)

func TestProviderNeutralPortMapsOpenStepInspectStopWithoutReplay(t *testing.T) {
	c, j, io := preparedSession(t)
	fake := &portFake{t: t}
	ctx := context.Background()
	a, err := c.SessionAdmission("s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Open(ctx, a, io); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	input, contract := payload(t, "compiled input"), payload(t, "output contract")
	plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "step", Input: input.Ref(), OutputContract: contract.Ref()}
	commit(t, c, j, run.PrepareStepCommand(meta("prepare-step"), "a", "r", plan, run.Reservation{ID: "step-effect", Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, input, contract))
	step, err := c.StepAdmission("step")
	if err != nil {
		t.Fatal(err)
	}
	if result, err := fake.Step(ctx, step, io); err != nil || !result.Terminal {
		t.Fatal(result, err)
	}
	commit(t, c, j, settle("step-effect", time.Second))
	inspection, err := c.Inspection("s", "step")
	if err != nil {
		t.Fatal(err)
	}
	head, calls := c.Revision(), j.appendCalls
	if _, err := fake.Inspect(ctx, inspection); err != nil {
		t.Fatal(err)
	}
	if c.Revision() != head || j.appendCalls != calls {
		t.Fatal("inspect wrote or resent")
	}
	stop, err := c.StopRequest("s", "end known session")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Stop(ctx, stop, io); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.calls, []string{"Open", "Step", "Inspect", "Stop"}) {
		t.Fatal(fake.calls)
	}
	finishIncomplete(t, c, j)
	c2 := restored(t, &readJournal{store: j})
	s, _ := c2.Session("s")
	st, _ := c2.Step("step")
	if !s.Terminated || !st.Terminal || st.Proposed == (run.BlobRef{}) || c2.RecoveryRequired() {
		t.Fatal("port observations not reconstructed", s, st)
	}
}
