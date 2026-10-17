package run

import "context"

type sessionChannel struct {
	c               *Coordinator
	journal         Journal
	assignment, run string
	session         SessionID
}

func (c *Coordinator) Session(id SessionID) (SessionView, bool) {
	v, ok := c.state.sessions[id]
	return v, ok
}
func (c *Coordinator) Step(id StepID) (StepView, bool) { v, ok := c.state.steps[id]; return v, ok }
func (c *Coordinator) SessionChannel(journal Journal, id SessionID) (SessionIO, error) {
	v, ok := c.state.sessions[id]
	if journal == nil || !ok || v.Run != c.Current().ID || v.Assignment != c.Current().Assignment || v.Terminated {
		return nil, ErrState
	}
	return &sessionChannel{c, journal, v.Assignment, v.Run, id}, nil
}
func (c *Coordinator) Inspection(session SessionID, step StepID) (InspectionRequest, error) {
	v, ok := c.state.sessions[session]
	t, found := c.state.steps[step]
	if !ok || !found || t.Plan.Session != session || v.Provider == "" || t.Provider == "" || v.Binding.BatchID == "" || t.Binding.BatchID == "" {
		return InspectionRequest{}, ErrObservationLimit
	}
	return InspectionRequest{session, step, v.Provider, t.Provider, v.Binding, t.Binding}, nil
}
func (c *Coordinator) SessionAdmission(id SessionID) (SessionAdmission, error) {
	v, ok := c.state.sessions[id]
	if !ok || v.Terminated || v.Bound || v.Uncertain || c.pending != nil || c.recovering || v.Run != c.Current().ID || c.Current().State != Running || c.Current().StopRequested {
		return SessionAdmission{}, ErrState
	}
	return SessionAdmission{v.Plan, v.Prepared, v.Invocation}, nil
}
func (c *Coordinator) StepAdmission(id StepID) (StepAdmission, error) {
	v, ok := c.state.steps[id]
	if !ok || v.Terminal || v.Started || v.Uncertain || c.pending != nil || c.recovering || v.Run != c.Current().ID || c.Current().State != Running || c.Current().StopRequested {
		return StepAdmission{}, ErrState
	}
	return StepAdmission{v.Plan, c.state.sessions[v.Plan.Session].Plan, v.Prepared, v.Invocation}, nil
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
	return io.commit(ctx, o.RecordMeta, BindSession, SessionRecord{ProviderSession: o.ProviderSession, Wire: o.Wire.record(), Uncertain: o.Uncertain, Reason: o.Reason}, o.Wire.Journal)
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
func (io *sessionChannel) ToolRequest(ctx context.Context, call ToolCall) (ToolReceipt, error) {
	r := SessionRecord{Session: io.session, Step: call.Step, Call: call.Call, Tool: call.Tool, Wire: call.Input.record()}
	if old, ok := io.c.state.tools[toolKey{io.session, call.Call}]; ok {
		if old.Record != r {
			return ToolReceipt{}, ErrConflict
		}
		result := ToolReceipt{Receipt: old.Requested, Replied: old.Replied.BatchID != "", Reply: old.Reply}
		if !result.Replied {
			return result, ErrObservationLimit
		}
		result.Receipt = old.Replied
		if old.Reply.Redacted {
			return result, ErrObservationLimit
		}
		p, err := io.journal.ReadBlob(ctx, old.Reply.Journal)
		if err != nil || !p.Valid() || p.Ref() != old.Reply.Journal {
			return result, ErrObservationLimit
		}
		result.Payload = p
		return result, nil
	}
	receipt, err := io.commit(ctx, call.RecordMeta, RequestTool, r, call.Input.Journal)
	return ToolReceipt{Receipt: receipt, Fresh: err == nil}, err
}
func (io *sessionChannel) ToolReply(ctx context.Context, r ToolReply) (AppendReceipt, error) {
	if old, ok := io.c.state.tools[toolKey{io.session, r.Call}]; ok && old.Replied.BatchID != "" {
		if old.Reply != r.Output.record() {
			return AppendReceipt{}, ErrConflict
		}
		return old.Replied, nil
	}
	return io.commit(ctx, r.RecordMeta, ReplyTool, SessionRecord{Call: r.Call, Wire: r.Output.record()}, r.Output.Journal)
}
func (io *sessionChannel) ObserveStep(ctx context.Context, o StepObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveStep, SessionRecord{Step: o.Step, ProviderStep: o.ProviderStep, Wire: o.Wire.record(), Proposed: o.Proposed.Ref(), Terminal: o.Terminal, Uncertain: o.Uncertain, Reason: o.Reason}, o.Wire.Journal, o.Proposed)
}
func (io *sessionChannel) ObserveStop(ctx context.Context, o StopObservation) (AppendReceipt, error) {
	return io.commit(ctx, o.RecordMeta, ObserveSessionStop, SessionRecord{Terminated: o.Terminated, Reason: o.Reason})
}

var _ SessionIO = (*sessionChannel)(nil)
