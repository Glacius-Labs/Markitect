package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

type sessionChannel struct {
	c               *Coordinator
	journal         Journal
	tools           ToolHandler
	assignment, run string
	session         SessionID
}

func (c *Coordinator) Session(id SessionID) (SessionView, bool) {
	v, ok := c.state.sessions[id]
	return v, ok
}
func (c *Coordinator) Step(id StepID) (StepView, bool) { v, ok := c.state.steps[id]; return v, ok }

// SessionChannel binds the Host side of one session. tools answers the
// session's tool calls; use RefuseTools where a slice offers none.
func (c *Coordinator) SessionChannel(journal Journal, id SessionID, tools ToolHandler) (SessionIO, error) {
	v, ok := c.state.sessions[id]
	if journal == nil || tools == nil || !ok || v.Run != c.Current().ID || v.Assignment != c.Current().Assignment || v.Terminated {
		return nil, ErrState
	}
	return &sessionChannel{c, journal, tools, v.Assignment, v.Run, id}, nil
}

// payloads reads the exact recorded bytes behind refs from the journal.
func payloads(ctx context.Context, journal Journal, refs ...BlobRef) ([]Payload, error) {
	if journal == nil {
		return nil, ErrState
	}
	out := make([]Payload, len(refs))
	for i, ref := range refs {
		p, err := journal.ReadBlob(ctx, ref)
		if err != nil || !p.Valid() || p.Ref() != ref {
			return nil, ErrObservationLimit
		}
		out[i] = p
	}
	return out, nil
}
func (c *Coordinator) Inspection(session SessionID, step StepID) (InspectionRequest, error) {
	v, ok := c.state.sessions[session]
	t, found := c.state.steps[step]
	if !ok || !found || t.Plan.Session != session || v.Provider == "" || t.Provider == "" || v.Binding.BatchID == "" || t.Binding.BatchID == "" {
		return InspectionRequest{}, ErrObservationLimit
	}
	return InspectionRequest{session, step, v.Provider, t.Provider, v.Binding, t.Binding}, nil
}
func (c *Coordinator) SessionAdmission(ctx context.Context, journal Journal, id SessionID) (SessionAdmission, error) {
	v, ok := c.state.sessions[id]
	if !ok || v.Terminated || v.Ready || v.Bound || v.Uncertain || c.pending != nil || c.recovering || v.Run != c.Current().ID || c.Current().State != Running || c.Current().StopRequested {
		return SessionAdmission{}, ErrState
	}
	for key := range c.state.transports {
		if key.Session == id {
			return SessionAdmission{}, ErrObservationLimit
		}
	}
	p, err := payloads(ctx, journal, v.Plan.Profile, v.Plan.Tools)
	if err != nil {
		return SessionAdmission{}, err
	}
	return SessionAdmission{v.Plan, p[0], p[1], v.Prepared, v.Invocation}, nil
}
func (c *Coordinator) StepAdmission(ctx context.Context, journal Journal, id StepID) (StepAdmission, error) {
	v, ok := c.state.steps[id]
	if !ok || v.Terminal || v.Started || v.Uncertain || c.pending != nil || c.recovering || v.Run != c.Current().ID || c.Current().State != Running || c.Current().StopRequested {
		return StepAdmission{}, ErrState
	}
	if session := c.state.sessions[v.Plan.Session]; session.InitialStep == id && session.InitialSend != "" {
		return StepAdmission{}, ErrObservationLimit
	}
	p, err := payloads(ctx, journal, v.Plan.Input, v.Plan.OutputContract)
	if err != nil {
		return StepAdmission{}, err
	}
	return StepAdmission{v.Plan, c.state.sessions[v.Plan.Session].Plan, p[0], p[1], v.Prepared, v.Invocation}, nil
}
func (c *Coordinator) StopRequest(id SessionID, reason string) (StopRequest, error) {
	v, ok := c.state.sessions[id]
	if !ok || v.Terminated || v.Run != c.Current().ID || !boundedInput(reason) {
		return StopRequest{}, ErrState
	}
	return StopRequest{id, v.Provider, c.state.policy.HostOperationTimeout, reason}, nil
}
func (io *sessionChannel) commit(ctx context.Context, meta RecordMeta, kind string, record SessionRecord, payloads ...Payload) (AppendReceipt, error) {
	record.Session = io.session
	c := Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: kind, Assignment: io.assignment, Run: io.run, Session: record}
	for _, p := range payloads {
		if p.Ref() != (BlobRef{}) {
			c.sessionPayloads = append(c.sessionPayloads, p)
		}
	}
	p, err := io.c.Propose(c)
	if err != nil {
		return AppendReceipt{}, err
	}
	return io.c.Commit(ctx, io.journal, p)
}
func (io *sessionChannel) Bind(ctx context.Context, o SessionObservation) (AppendReceipt, error) {
	if o.Ready || !o.Valid() {
		return AppendReceipt{}, ErrInvalid
	}
	return io.commit(ctx, o.RecordMeta, BindSession, SessionRecord{ProviderSession: o.ProviderSession, Wire: o.Wire.record(), Uncertain: o.Uncertain, Reason: o.Reason}, o.Wire.Journal)
}

func (io *sessionChannel) Ready(ctx context.Context, o SessionReadiness) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ReadySession, SessionRecord{Wire: o.Wire.record(), Reason: o.Reason}, o.Wire.Journal)
}
func (io *sessionChannel) PrepareSend(ctx context.Context, p TransportPreparation) (SendPermit, error) {
	r := SessionRecord{Session: io.session, Step: p.Step, Message: p.Message, Call: p.Call, Control: p.Control, Wire: p.Wire.record()}
	// Re-observing a preparation never authorizes another physical send.
	if old, ok := io.c.state.transports[messageKey{io.session, p.Message}]; ok {
		if old.Record != r {
			return SendPermit{}, ErrConflict
		}
		return SendPermit{Receipt: old.Prepared, Identity: r.Wire.Identity}, ErrObservationLimit
	}
	receipt, err := io.commit(ctx, p.RecordMeta, PrepareTransport, r, p.Wire.Journal)
	return SendPermit{Receipt: receipt, Identity: r.Wire.Identity, Dispatch: err == nil}, err
}
func (io *sessionChannel) ObserveSend(ctx context.Context, o TransportObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveTransport, SessionRecord{Message: o.Message, Outcome: o.Outcome, Reason: o.Reason})
}

// Call records the tool call, lets the Host answer it and records the answer.
// An exact repeat returns the recorded answer without executing again; a
// changed repeat conflicts. A call recorded without an answer stays pending:
// its effect is unknown and only reconciliation can settle it. While the run
// admits no new work, the call is answered with a fixed refusal instead of
// reaching the handler; that refusal settles the call but is never sent.
func (io *sessionChannel) Call(ctx context.Context, call ToolCall) (ToolResult, error) {
	r := SessionRecord{Session: io.session, Step: call.Step, Call: call.Call, Tool: call.Tool, Wire: call.Input.record()}
	if old, ok := io.c.state.tools[toolKey{io.session, call.Call}]; ok {
		if old.Record != r {
			return ToolResult{}, ErrConflict
		}
		if old.Replied.BatchID == "" {
			return ToolResult{Receipt: old.Requested}, ErrObservationLimit
		}
		p, err := payloads(ctx, io.journal, old.Reply.Journal)
		if err != nil {
			return ToolResult{Receipt: old.Replied}, err
		}
		var outcome ToolOutcome
		if err := strictJSON(p[0].Bytes(), &outcome); err != nil {
			return ToolResult{Receipt: old.Replied}, ErrObservationLimit
		}
		return ToolResult{Receipt: old.Replied, Outcome: outcome, Output: p[0]}, nil
	}
	if _, err := io.commit(ctx, call.RecordMeta, RequestTool, r, call.Input.Journal); err != nil {
		return ToolResult{}, err
	}
	limit := io.c.state.sessions[io.session].Plan.MaxWireBytes
	answer, current := io.tools, io.c.Current()
	if io.c.recovering || current.State != Running || current.StopRequested {
		answer = refuseHalted
	}
	outcome, err := answer(ctx, call, limit)
	if err != nil {
		return ToolResult{}, err
	}
	output, err := outcome.payload()
	if err != nil {
		return ToolResult{}, err
	}
	if output.Ref().Size > limit {
		return ToolResult{}, ErrInvalid
	}
	result := ToolResult{Outcome: outcome, Output: output}
	// The Host answers inside the same callback, so the answer shares the
	// call's observation time.
	meta := RecordMeta{ID: replyID(io.session, call.Call), ObservedAt: call.ObservedAt}
	receipt, err := io.commit(ctx, meta, ReplyTool, SessionRecord{Call: call.Call, Wire: outcomeWire(output)}, output)
	if err != nil {
		// A possibly partial receipt is no proof that the answer was recorded.
		return result, err
	}
	result.Receipt = receipt
	return result, nil
}

// replyID derives the record identity of a call's answer from its dedup key,
// so it is always a valid identifier and stable across retries.
func replyID(session SessionID, call ProviderCallID) string {
	sum := sha256.Sum256([]byte(string(session) + "\x00" + string(call)))
	return "tool-reply-" + hex.EncodeToString(sum[:])
}

func outcomeWire(output Payload) WireRecord {
	return WirePayload{Identity: WireIdentity{ContentID: output.Ref().ContentID, Size: output.Ref().Size}, Journal: output}.record()
}

// ToolReconciliation records the Host's established answer for a call that was
// recorded without one: for example the answer Call returned with a recording
// error, or a refusal after proving the call never ran. It is Host
// reconciliation, not an adapter callback, and refuses a call that is not
// pending.
func (c *Coordinator) ToolReconciliation(meta RecordMeta, session SessionID, call ProviderCallID, outcome ToolOutcome) (Command, error) {
	v, ok := c.state.sessions[session]
	tool, found := c.state.tools[toolKey{session, call}]
	if !ok || !found || tool.Replied.BatchID != "" {
		return Command{}, ErrState
	}
	output, err := outcome.payload()
	if err != nil {
		return Command{}, err
	}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: ReplyTool, Assignment: v.Assignment, Run: v.Run,
		Session: SessionRecord{Session: session, Call: call, Wire: outcomeWire(output)}, sessionPayloads: []Payload{output}}, nil
}

func (io *sessionChannel) ObserveStep(ctx context.Context, o StepObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveStep, SessionRecord{Step: o.Step, ProviderStep: o.ProviderStep, Wire: o.Wire.record(), Proposed: o.Proposed.Ref(), Terminal: o.Terminal, Uncertain: o.Uncertain, Reason: o.Reason}, o.Wire.Journal, o.Proposed)
}
func (io *sessionChannel) ObserveStop(ctx context.Context, o StopObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveSessionStop, SessionRecord{Terminated: o.Terminated, Reason: o.Reason})
}

var _ SessionIO = (*sessionChannel)(nil)
