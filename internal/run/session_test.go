package run_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
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
func sessionPreparation(t *testing.T, c *run.Coordinator, id run.SessionID) run.Command {
	return exploreSessionPreparation(t, c, exploreInputs(t, false), ".", id)
}
func stepPreparation(t *testing.T, c *run.Coordinator, id run.StepID, invocation string) run.Command {
	t.Helper()
	v, _ := c.Session("s")
	cmd, err := c.PrepareExploreStepCommand(meta("prepare-"+string(id)), run.StepPlan{Session: "s", ID: id, SessionBinding: v.Plan.Binding()}, run.Reservation{ID: invocation, Kind: run.AgentStep, Purpose: run.Writing, Maximum: time.Minute}, exploreInputs(t, false))
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func preparedSession(t *testing.T) (*run.Coordinator, *memoryJournal, run.SessionIO) {
	t.Helper()
	c, j := running(t)
	commit(t, c, j, sessionPreparation(t, c, "s"))
	io, err := c.SessionChannel(j, "s", run.RefuseTools)
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
	commit(t, c, j, stepPreparation(t, c, plan.ID, "step-invocation"))
	send(t, io, "turn", "step")
	_, err := io.ObserveStep(context.Background(), run.StepObservation{RecordMeta: meta("step-started"), Step: "step", ProviderStep: "provider-turn", Wire: wire(t, `{"turn":"started"}`), Reason: "known turn"})
	if err != nil {
		t.Fatal(err)
	}
	return c, j, io
}

func TestSessionAdmissionAndReferencesCommitAtomically(t *testing.T) {
	c, j := running(t)
	cmd := sessionPreparation(t, c, "s")
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
	a, err := c.SessionAdmission(context.Background(), j, "s")
	if err != nil || a.Plan.ID != "s" || a.Receipt.BatchID == "" || a.Profile.Ref() != a.Plan.Profile || a.Tools.Ref() != a.Plan.Tools || string(a.Tools.Bytes()) != `{"tools":[]}` {
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
			io2, err := c2.SessionChannel(j, "s", run.RefuseTools)
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
	io2, err := c2.SessionChannel(j, "s", run.RefuseTools)
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

// answering returns a handler that answers every call with result and counts
// how often the Host executed it.
func answering(result string, calls *int) run.ToolHandler {
	return func(context.Context, run.ToolCall, int64) (run.ToolOutcome, error) {
		*calls++
		return run.ToolOutcome{Status: run.ToolOK, Result: []byte(result)}, nil
	}
}
func failing(context.Context, run.ToolCall, int64) (run.ToolOutcome, error) {
	return run.ToolOutcome{}, errors.New("Host tool failed after possible effect")
}
func channel(t *testing.T, c *run.Coordinator, j run.Journal, tools run.ToolHandler) run.SessionIO {
	t.Helper()
	io, err := c.SessionChannel(j, "s", tools)
	if err != nil {
		t.Fatal(err)
	}
	return io
}

func TestCallAnswersOnceAndReturnsTheRecordedAnswer(t *testing.T) {
	c, j, _ := activeStep(t)
	executed := 0
	io := channel(t, c, j, answering(`{"read":"content"}`, &executed))
	call := run.ToolCall{RecordMeta: meta("tool-ingress"), Step: "step", Call: "provider-call", Tool: "supplied-host-tool", Input: wire(t, `{"read":"target"}`)}
	first, err := io.Call(context.Background(), call)
	if err != nil || executed != 1 || first.Receipt.BatchID == "" || string(first.Output.Bytes()) != `{"status":"ok","result":{"read":"content"}}` {
		t.Fatal("call not answered and recorded", first, err, executed)
	}
	before, appends := c.Revision(), j.appendCalls
	call.RecordMeta = meta("duplicate-ingress")
	again, err := io.Call(context.Background(), call)
	if err != nil || executed != 1 || again.Receipt != first.Receipt || !bytes.Equal(again.Output.Bytes(), first.Output.Bytes()) {
		t.Fatal("exact repeat executed again or lost its answer", again, err, executed)
	}
	if c.Revision() != before || j.appendCalls != appends {
		t.Fatal("exact repeat appended")
	}
	changed := call
	changed.Input = wire(t, `{"read":"different"}`)
	if _, err := io.Call(context.Background(), changed); !errors.Is(err, run.ErrConflict) || executed != 1 {
		t.Fatal("call identity changed content", err)
	}
	c2 := restored(t, &readJournal{store: j})
	restarted, err := channel(t, c2, j, answering(`{"read":"other"}`, &executed)).Call(context.Background(), call)
	if err != nil || executed != 1 || restarted.Receipt != first.Receipt || !bytes.Equal(restarted.Output.Bytes(), first.Output.Bytes()) {
		t.Fatal("restart lost the recorded answer", restarted, err)
	}
	inspection, err := c2.Inspection("s", "step")
	if err != nil || inspection.ProviderSession != "provider-session" || inspection.ProviderStep != "provider-turn" || inspection.SessionBinding.BatchID == "" || inspection.StepBinding.BatchID == "" {
		t.Fatal("inspection not exact", inspection, err)
	}
	if _, err := c2.Inspection("s", "other-step"); !errors.Is(err, run.ErrObservationLimit) {
		t.Fatal("inspection guessed turn", err)
	}
}

func TestChannelRequiresHandlerAndRefusalIsAnAnswer(t *testing.T) {
	c, j, io := activeStep(t)
	if _, err := c.SessionChannel(j, "s", nil); !errors.Is(err, run.ErrState) {
		t.Fatal("channel without Host tools", err)
	}
	r, err := io.Call(context.Background(), run.ToolCall{RecordMeta: meta("refused"), Step: "step", Call: "call", Tool: "unsupported", Input: wire(t, `{}`)})
	if err != nil || string(r.Output.Bytes()) != `{"status":"refused","reason":"no Host tool is available in this slice"}` {
		t.Fatal("refusal not recorded as an answer", r, err)
	}
	for name, outcome := range map[string]run.ToolOutcome{
		"ok with reason":         {Status: run.ToolOK, Reason: "mixed"},
		"refused without reason": {Status: run.ToolRefused},
		"refused with result":    {Status: run.ToolRefused, Result: []byte(`{}`), Reason: "mixed"},
		"invalid result":         {Status: run.ToolOK, Result: []byte(`{"open"`)},
		"unknown status":         {Status: "maybe"},
		"oversize result":        {Status: run.ToolOK, Result: []byte(`"` + strings.Repeat("x", 1<<20) + `"`)},
	} {
		executed := 0
		handler := func(context.Context, run.ToolCall, int64) (run.ToolOutcome, error) {
			executed++
			return outcome, nil
		}
		call := run.ToolCall{RecordMeta: meta("shape-" + name), Step: "step", Call: run.ProviderCallID("shape-" + name), Tool: "unsupported", Input: wire(t, `{}`)}
		if _, err := channel(t, c, j, handler).Call(context.Background(), call); !errors.Is(err, run.ErrInvalid) || executed != 1 {
			t.Fatal(name, "accepted", err)
		}
		if _, err := channel(t, c, j, handler).Call(context.Background(), call); !errors.Is(err, run.ErrObservationLimit) || executed != 1 {
			t.Fatal(name, "invalid answer did not leave the call pending", err)
		}
	}
}

func TestEveryRecordedAnswerCanBeSentOnceInItsStep(t *testing.T) {
	_, _, io := activeStep(t)
	for _, id := range []run.ProviderCallID{"first", "second"} {
		if _, err := io.Call(context.Background(), run.ToolCall{RecordMeta: meta("call-" + string(id)), Step: "step", Call: id, Tool: "unsupported", Input: wire(t, `{}`)}); err != nil {
			t.Fatal(err)
		}
		frame := wire(t, `{"id":"`+string(id)+`","result":"refused"}`)
		answer := run.TransportPreparation{RecordMeta: meta("answer-" + string(id)), Message: run.MessageID("answer-" + string(id)), Step: "step", Call: id, Wire: frame}
		if p, err := io.PrepareSend(context.Background(), answer); err != nil || !p.Dispatch {
			t.Fatal("recorded answer could not be sent", id, err)
		}
		if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("answered-" + string(id)), Message: answer.Message, Outcome: run.Sent, Reason: "observed write"}); err != nil {
			t.Fatal(err)
		}
	}
	again := run.TransportPreparation{RecordMeta: meta("answer-again"), Message: "answer-again", Step: "step", Call: "first", Wire: wire(t, `{"id":"first","result":"again"}`)}
	if _, err := io.PrepareSend(context.Background(), again); !errors.Is(err, run.ErrState) {
		t.Fatal("answer sent twice", err)
	}
	unanswered := run.TransportPreparation{RecordMeta: meta("answer-unknown"), Message: "answer-unknown", Step: "step", Call: "unknown", Wire: wire(t, `{"id":"unknown"}`)}
	if _, err := io.PrepareSend(context.Background(), unanswered); !errors.Is(err, run.ErrState) {
		t.Fatal("answer without a recorded Host answer", err)
	}
	plain := run.TransportPreparation{RecordMeta: meta("plain"), Message: "plain", Step: "step", Wire: wire(t, `{"user":"another turn"}`)}
	if _, err := io.PrepareSend(context.Background(), plain); !errors.Is(err, run.ErrState) {
		t.Fatal("a second turn input was sent", err)
	}
}

func TestHaltedRunAnswersWithoutReachingTheHandler(t *testing.T) {
	c, j, _ := activeStep(t)
	stop := command("stop", run.StopRun)
	stop.Reason = "owner stop"
	commit(t, c, j, stop)
	executed := 0
	r, err := channel(t, c, j, answering(`{}`, &executed)).Call(context.Background(), run.ToolCall{RecordMeta: meta("after-stop"), Step: "step", Call: "after-stop", Tool: "read", Input: wire(t, `{}`)})
	if err != nil || executed != 0 || r.Outcome.Status != run.ToolRefused || r.Receipt.BatchID == "" {
		t.Fatal("stopped run reached the Host tool", r, err, executed)
	}
}

func TestLostAnswerKeepsOutcomeForHostReconciliation(t *testing.T) {
	c, j, _ := activeStep(t)
	executed := 0
	io := channel(t, c, j, answering(`{"read":"content"}`, &executed))
	if _, err := io.Call(context.Background(), run.ToolCall{RecordMeta: meta("first"), Step: "step", Call: "first", Tool: "read", Input: wire(t, `{}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ToolReconciliation(meta("not-pending"), "s", "first", run.ToolOutcome{Status: run.ToolRefused, Reason: "late"}); !errors.Is(err, run.ErrState) {
		t.Fatal("reconciled an answered call", err)
	}
	call := run.ToolCall{RecordMeta: meta("lost"), Step: "step", Call: "lost", Tool: "read", Input: wire(t, `{}`)}
	j.appendMode = "absent-after-next"
	r, err := io.Call(context.Background(), call)
	j.appendMode = ""
	if err == nil || r.Receipt.BatchID != "" || r.Outcome.Status != run.ToolOK || string(r.Output.Bytes()) != `{"status":"ok","result":{"read":"content"}}` || executed != 2 {
		t.Fatal("lost answer not kept for reconciliation", r, err, executed)
	}
	if err := c.ReconcilePending(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	acked := run.ToolCall{RecordMeta: meta("ack-lost"), Step: "step", Call: "ack-lost", Tool: "read", Input: wire(t, `{}`)}
	j.appendMode = "lost-after-next"
	lost, err := io.Call(context.Background(), acked)
	j.appendMode = ""
	if err == nil || lost.Receipt != (run.AppendReceipt{}) || lost.Outcome.Status != run.ToolOK {
		t.Fatal("lost acknowledgement returned a receipt", lost, err)
	}
	if err := c.ReconcilePending(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if recorded, err := io.Call(context.Background(), acked); err != nil || recorded.Receipt.BatchID == "" || executed != 3 {
		t.Fatal("answer committed behind a lost acknowledgement not adopted", recorded, err)
	}
	if _, err := io.Call(context.Background(), call); !errors.Is(err, run.ErrObservationLimit) || executed != 3 {
		t.Fatal("absent answer executed again", err)
	}
	reconciled, err := c.ToolReconciliation(meta("lost-reconciled"), "s", "lost", r.Outcome)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, reconciled)
	again, err := io.Call(context.Background(), call)
	if err != nil || executed != 3 || !bytes.Equal(again.Output.Bytes(), r.Output.Bytes()) {
		t.Fatal("reconciled answer not returned", again, err)
	}
}

func TestRedactedCallInputStoresOnlyThePermittedRepresentation(t *testing.T) {
	c, j, io := activeStep(t)
	// The supplied hash is the identity of a hypothetical real wire, not a
	// BlobRef. No secret value is constructed or supplied to pure RUN.
	identity := run.WireIdentity{ContentID: snapshot.ContentID(sha256.Sum256([]byte("opaque wire identity fixture"))), Size: 80}
	redacted := run.WirePayload{Identity: identity, Journal: payload(t, `{"credential":"[REDACTED]"}`), Redacted: true, ObservationLimit: "Host redacted known credential before append; original bytes unavailable for replay"}
	call := run.ToolCall{RecordMeta: meta("call"), Step: "step", Call: "call", Tool: "schema", Input: redacted}
	if _, err := io.Call(context.Background(), call); err != nil {
		t.Fatal(err)
	}
	refs, err := j.batches[len(j.batches)-2].SessionRefs()
	if err != nil || len(refs) != 1 || refs[0] != redacted.Journal.Ref() || refs[0].ContentID == identity.ContentID {
		t.Fatal("wire identity confused with stored bytes", refs, err)
	}
	c2 := restored(t, &readJournal{store: j})
	if v, _ := c2.Session("s"); v.Plan.CredentialRef != "credential-ref" {
		t.Fatal("credential reference lost")
	}
	if c.Revision() != c2.Revision() {
		t.Fatal("redacted history not restored")
	}
}

func TestPendingToolCallBlocksCompletionUntilHostReconciliation(t *testing.T) {
	c, j, _ := activeStep(t)
	io := channel(t, c, j, failing)
	call := run.ToolCall{RecordMeta: meta("tool"), Step: "step", Call: "call", Tool: "schema", Input: wire(t, `{"request":"bounded"}`)}
	if _, err := io.Call(context.Background(), call); err == nil {
		t.Fatal("failed Host tool reported an answer")
	}
	executed := 0
	if _, err := channel(t, c, j, answering(`{}`, &executed)).Call(context.Background(), call); !errors.Is(err, run.ErrObservationLimit) || executed != 0 {
		t.Fatal("pending call executed again", err, executed)
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
	reconciled, err := c.ToolReconciliation(meta("tool-reconciled"), "s", "call", run.ToolOutcome{Status: run.ToolRefused, Reason: "proven not executed"})
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, reconciled)
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
				commit(t, c, j, stepPreparation(t, c, plan.ID, "step-hold"))
			}
			stop := command("stop", run.StopRun)
			stop.Reason = "owner stop"
			commit(t, c, j, stop)
			var err error
			if kind == "session" {
				_, err = c.SessionAdmission(context.Background(), j, "s")
			} else {
				_, err = c.StepAdmission(context.Background(), j, "unstarted")
			}
			if !errors.Is(err, run.ErrState) {
				t.Fatal("stop request allowed new external launch", err)
			}
		})
	}
}

func TestRecoveryAllowsBoundedStopSendWithoutClearingUnknownOrdinarySend(t *testing.T) {
	c, j, io := activeStep(t)
	denied, err := io.Call(context.Background(), run.ToolCall{RecordMeta: meta("deny-request"), Step: "step", Call: "deny", Tool: "unsupported", Input: wire(t, "request")})
	if err != nil {
		t.Fatal(err)
	}
	reply := run.WirePayload{Identity: run.WireIdentity{ContentID: denied.Output.Ref().ContentID, Size: denied.Output.Ref().Size}, Journal: denied.Output}
	p := run.TransportPreparation{RecordMeta: meta("uncertain-send"), Message: "uncertain", Step: "step", Call: "deny", Wire: reply}
	if _, err := io.PrepareSend(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("send-unknown"), Message: p.Message, Outcome: run.SendUncertain, Reason: "connection failed after possible write"}); err != nil {
		t.Fatal(err)
	}
	c2 := restored(t, &readJournal{store: j})
	before := c2.Budget()
	io2, err := c2.SessionChannel(j, "s", run.RefuseTools)
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
	if !c2.RecoveryRequired() {
		t.Fatal("stop erased unknown send")
	}
	if _, err := io2.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("stop-sent"), Message: "stop", Outcome: run.Sent, Reason: "observed stop transport"}); err != nil {
		t.Fatal(err)
	}
	if _, err := io2.ObserveSend(context.Background(), run.TransportObservation{RecordMeta: meta("reply-sent"), Message: "uncertain", Outcome: run.Sent, Reason: "explicitly reconciled denial transport"}); err != nil {
		t.Fatal(err)
	}
	if c2.RecoveryRequired() {
		t.Fatal("known quiescence did not clear recovery")
	}
	_ = c
}

func TestUncertainStepRejectsMalformedPartialProviderHandle(t *testing.T) {
	c, j, io := boundSession(t)
	input, contract := payload(t, "input"), payload(t, "contract")
	plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "step", Input: input.Ref(), OutputContract: contract.Ref()}
	commit(t, c, j, stepPreparation(t, c, plan.ID, "step-effect"))
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
	cmd := stepPreparation(t, c, plan.ID, "step-hold")
	cmd.Session.StepPlan.SessionBinding = changed.Binding()
	if _, err := c.Propose(cmd); !errors.Is(err, run.ErrState) || c.Budget() != before {
		t.Fatal("changed session basis continued", err)
	}
	plan.SessionBinding = v.Plan.Binding()
	commit(t, c, j, stepPreparation(t, c, plan.ID, "step-hold"))
	admission, err := c.StepAdmission(context.Background(), j, "step")
	if err != nil || admission.Session != v.Plan || admission.Plan.SessionBinding != v.Plan.Binding() {
		t.Fatal("admission lost fixed plan", admission, err)
	}
}

func TestSessionLimitsAndRestoreRefuseMissingOrCorruptRequiredBytes(t *testing.T) {
	c, j, io := preparedSession(t)
	bad := sessionPreparation(t, c, "other")
	bad.Session.Plan.Profile = run.BlobRef{}
	if _, err := c.Propose(bad); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("missing mandatory reference accepted", err)
	}
	if refs, err := (run.AppendBatch{}).SessionRefs(); err == nil || len(refs) != 0 {
		t.Fatal("invalid batch exposed partial closure", err)
	}
	oversize := run.TransportPreparation{RecordMeta: meta("oversize"), Message: "large", Wire: wire(t, string(bytes.Repeat([]byte("x"), (1<<20)+1)))}
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
	a, err := c.SessionAdmission(context.Background(), j, "s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Open(ctx, a, io); err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, settle("open-s", time.Second))
	input, contract := payload(t, "compiled input"), payload(t, "output contract")
	plan := run.StepPlan{Session: "s", SessionBinding: currentSessionBinding(c), ID: "step", Input: input.Ref(), OutputContract: contract.Ref()}
	commit(t, c, j, stepPreparation(t, c, plan.ID, "step-effect"))
	step, err := c.StepAdmission(context.Background(), j, "step")
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

func resumedSession(t *testing.T) run.SessionIO {
	t.Helper()
	c, j := running(t)
	inputs := exploreInputs(t, false)
	profile, tools := payload(t, `{"profile":"synthetic recorded metadata"}`), payload(t, `{"tools":[]}`)
	plan := run.SessionPlan{ID: "s", Role: "manager", Workspace: "linux-workspace-s", ProviderHome: "provider-home", Resume: "provider-session", Candidate: inputs.Binding.Census.Digest(), ReadOnly: true, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 1 << 20, Profile: profile.Ref(), Tools: tools.Ref()}
	cmd, err := c.PrepareExploreSessionCommand(meta("prepare-s"), ".", plan, run.Reservation{ID: "open-s", Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, cmd)
	return channel(t, c, j, run.RefuseTools)
}

func TestResumedSessionBindsOnlyTheRequestedConversation(t *testing.T) {
	io := resumedSession(t)
	send(t, io, "resume", "")
	other := run.SessionObservation{RecordMeta: meta("bind-other"), ProviderSession: "other-session", Wire: wire(t, `{"thread":"other-session"}`), Reason: "provider started a new thread"}
	if _, err := io.Bind(context.Background(), other); !errors.Is(err, run.ErrState) {
		t.Fatal("a different conversation bound as the resumed one", err)
	}
	same := run.SessionObservation{RecordMeta: meta("bind"), ProviderSession: "provider-session", Wire: wire(t, `{"thread":"provider-session"}`), Reason: "provider resumed the thread"}
	if _, err := io.Bind(context.Background(), same); err != nil {
		t.Fatal(err)
	}
}

func TestResumedSessionReportsAnotherConversationAsUncertain(t *testing.T) {
	io := resumedSession(t)
	send(t, io, "resume", "")
	other := run.SessionObservation{RecordMeta: meta("bind-other"), ProviderSession: "other-session", Wire: wire(t, `{"thread":"other-session"}`), Uncertain: true, Reason: "provider reported a different thread"}
	if _, err := io.Bind(context.Background(), other); err != nil {
		t.Fatal("uncertain observation of another conversation refused", err)
	}
}

func TestStepAdmissionCarriesTheVerifiedRecordedBytes(t *testing.T) {
	c, j, _ := boundSession(t)
	commit(t, c, j, stepPreparation(t, c, "unstarted", "step-hold"))
	a, err := c.StepAdmission(context.Background(), j, "unstarted")
	if err != nil || a.Input.Ref() != a.Plan.Input || a.OutputContract.Ref() != a.Plan.OutputContract || len(a.Input.Bytes()) == 0 {
		t.Fatal("admission lacks the exact step bytes", a, err)
	}
	delete(j.blobs, a.Plan.Input)
	if _, err := c.StepAdmission(context.Background(), j, "unstarted"); !errors.Is(err, run.ErrObservationLimit) {
		t.Fatal("admission without its recorded input", err)
	}
}
