package run

import "context"

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
	r := SessionRecord{Session: io.session, Step: p.Step, Message: p.Message, Control: p.Control, Wire: p.Wire.record()}
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
// its effect is unknown and only reconciliation can settle it.
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
		return ToolResult{Receipt: old.Replied, Output: p[0]}, nil
	}
	if _, err := io.commit(ctx, call.RecordMeta, RequestTool, r, call.Input.Journal); err != nil {
		return ToolResult{}, err
	}
	outcome, err := io.tools(ctx, call)
	if err != nil {
		return ToolResult{}, err
	}
	output, err := outcome.payload()
	if err != nil {
		return ToolResult{}, err
	}
	// The Host answers inside the same callback, so the answer shares the
	// call's observation time; its record identity derives from the call's.
	meta := RecordMeta{ID: call.ID + "/reply", ObservedAt: call.ObservedAt}
	wire := WirePayload{Identity: WireIdentity{ContentID: output.Ref().ContentID, Size: output.Ref().Size}, Journal: output}
	receipt, err := io.commit(ctx, meta, ReplyTool, SessionRecord{Call: call.Call, Wire: wire.record()}, output)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Receipt: receipt, Output: output}, nil
}

// ToolReconciliation records the Host's established answer for a call that was
// recorded without one, for example after proving it was never executed. It is
// Host reconciliation, not an adapter callback.
func (c *Coordinator) ToolReconciliation(meta RecordMeta, session SessionID, call ProviderCallID, outcome ToolOutcome) (Command, error) {
	v, ok := c.state.sessions[session]
	if !ok {
		return Command{}, ErrState
	}
	output, err := outcome.payload()
	if err != nil {
		return Command{}, err
	}
	wire := WirePayload{Identity: WireIdentity{ContentID: output.Ref().ContentID, Size: output.Ref().Size}, Journal: output}
	return Command{ID: meta.ID, ObservedAt: meta.ObservedAt, Kind: ReplyTool, Assignment: v.Assignment, Run: v.Run,
		Session: SessionRecord{Session: session, Call: call, Wire: wire.record()}, sessionPayloads: []Payload{output}}, nil
}

func (io *sessionChannel) ObserveStep(ctx context.Context, o StepObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveStep, SessionRecord{Step: o.Step, ProviderStep: o.ProviderStep, Wire: o.Wire.record(), Proposed: o.Proposed.Ref(), Terminal: o.Terminal, Uncertain: o.Uncertain, Reason: o.Reason}, o.Wire.Journal, o.Proposed)
}
func (io *sessionChannel) ObserveStop(ctx context.Context, o StopObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveSessionStop, SessionRecord{Terminated: o.Terminated, Reason: o.Reason})
}

var _ SessionIO = (*sessionChannel)(nil)
