package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

type stepItem struct {
	Thread string `json:"threadId"`
	Turn   string `json:"turnId"`
	Item   struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Server    string          `json:"server"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
		Text      string          `json:"text"`
		Phase     string          `json:"phase"`
	} `json:"item"`
}
type stepEvent struct {
	message Message
	err     error
}

func (s *Sessions) Step(ctx context.Context, a run.StepAdmission, channel run.SessionIO) (out run.StepObservation, err error) {
	e, err := s.entry(a.Plan.Session)
	if err != nil || channel == nil {
		return out, run.ErrObservationLimit
	}
	if !e.mu.TryLock() {
		return out, run.ErrObservationLimit
	}
	defer e.mu.Unlock()
	if e.stopped || e.peer == nil || e.bound.BatchID == "" || e.step != "" || a.Session != e.plan || a.Plan.SessionBinding != e.plan.Binding() || a.Plan.ID == "" || a.Receipt.BatchID == "" || a.Invocation == "" || !a.Input.Valid() || a.Input.Ref() != a.Plan.Input || !a.OutputContract.Valid() || a.OutputContract.Ref() != a.Plan.OutputContract {
		return out, ErrSession
	}
	// Contract interpretation precedes any provider write. The field catalogue
	// is sent literally as instructions, never mislabelled as JSON Schema.
	if err = exploreContract(a.OutputContract); err != nil {
		return out, err
	}
	e.step = a.Plan.ID
	var pending []byte
	defer func() {
		if err != nil && !out.Terminal {
			out = run.StepObservation{RecordMeta: e.meta(), Step: e.step, ProviderStep: run.ProviderStepID(e.turn), Uncertain: true, Reason: "step observation incomplete; no automatic replay"}
			if len(pending) > 0 {
				out.Wire = e.wire(pending)
			}
			_, recordErr := channel.ObserveStep(context.WithoutCancel(ctx), out)
			err = errors.Join(e.safe(err), recordErr)
		}
	}()
	prompt := "Perform exactly one admitted Markitect step. Return the requested report as one JSON object in your final answer. The Host will assess it separately.\nExact input bytes:\n" + string(a.Input.Bytes()) + "\nExact output contract bytes (a field catalogue, not JSON Schema):\n" + string(a.OutputContract.Bytes())
	id := IntegerRequestID(3)
	raw, err := e.codec.EncodeTurnStart(id, TurnStartParams{Thread: e.thread, Text: prompt, Effort: e.profile.Effort})
	if err != nil {
		return out, err
	}
	if err = e.send(ctx, channel, e.step, "", raw, func() error { return e.peer.WriteInput(raw) }); err != nil {
		return out, err
	}
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := make(chan stepEvent)
	go func() {
		defer close(events)
		for {
			m, readErr := e.nextMessage(readCtx)
			select {
			case events <- stepEvent{m, readErr}:
			case <-readCtx.Done():
				return
			}
			if readErr != nil {
				return
			}
		}
	}()
	tools := e.peer.ToolCalls()
	var waiting []frame.ToolCall
	calls := map[string]stepItem{}
	seenCalls := map[uint64]bool{}
	lastAnswer := ""
	responseSeen := false
	for {
		// MCP and provider stdout are distinct runtime queues. A call can arrive
		// before its item notification; hold it until both identities are known.
		for i := 0; i < len(waiting); {
			call := waiting[i]
			var meta struct {
				Thread string `json:"threadId"`
				Call   string `json:"callId"`
			}
			if _, parseErr := protocolObject(call.Meta); parseErr != nil || json.Unmarshal(call.Meta, &meta) != nil || meta.Thread != string(e.thread) || meta.Call == "" {
				return out, ErrProtocol
			}
			item, ok := calls[meta.Call]
			if !ok || e.turn == "" {
				i++
				continue
			}
			if item.Thread != meta.Thread || item.Turn != string(e.turn) || item.Item.Server != "markitect" || item.Item.Tool != call.Name || !jsonEqual(item.Item.Arguments, call.Arguments) {
				return out, ErrProtocol
			}
			if err = e.answerTool(ctx, channel, call, run.ProviderCallID(meta.Call)); err != nil {
				return out, err
			}
			waiting = append(waiting[:i], waiting[i+1:]...)
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case call, ok := <-tools:
			if !ok {
				tools = nil
				continue
			}
			if call.ID == 0 || seenCalls[call.ID] || len(waiting) >= 16 {
				return out, ErrProtocol
			}
			seenCalls[call.ID] = true
			waiting = append(waiting, call)
		case event, ok := <-events:
			if !ok {
				return out, ErrProtocolIncomplete
			}
			m := event.message
			pending = append(pending, m.Raw()...)
			if int64(len(pending)) > e.plan.MaxWireBytes {
				pending = pending[:e.plan.MaxWireBytes]
				return out, ErrProtocolLimit
			}
			if event.err != nil {
				return out, event.err
			}
			terminal := false
			switch m.kind {
			case ResponseMessage, ErrorMessage:
				if responseSeen {
					return out, ErrProtocol
				}
				r, readErr := m.ReadResponse(id, TurnStart)
				if readErr != nil {
					return out, readErr
				}
				if e.turn != "" && e.turn != r.Turn.Turn {
					return out, ErrProtocol
				}
				e.turn = r.Turn.Turn
				responseSeen = true
				terminal = r.Turn.Status != TurnInProgress
			case TurnStartedMessage, TurnCompletedMessage:
				t := m.Turn()
				if t.Thread != e.thread || e.turn != "" && e.turn != t.Turn {
					return out, ErrProtocol
				}
				e.turn = t.Turn
				terminal = m.kind == TurnCompletedMessage
			case ThreadStartedMessage:
				if m.Thread() != e.thread {
					return out, ErrProtocol
				}
			case UnknownNotification:
				if m.method == "error" {
					return out, ErrProtocol
				}
				if m.method == "item/started" || m.method == "item/completed" {
					var item stepItem
					if json.Unmarshal(m.body, &item) != nil || item.Thread != string(e.thread) || item.Turn == "" || e.turn != "" && item.Turn != string(e.turn) {
						return out, ErrProtocol
					}
					e.turn = TurnID(item.Turn)
					if item.Item.Type == "mcpToolCall" {
						if item.Item.ID == "" {
							return out, ErrProtocol
						}
						calls[item.Item.ID] = item
					}
					if m.method == "item/completed" && item.Item.Type == "agentMessage" && (item.Item.Phase == "" || item.Item.Phase == "final_answer") {
						lastAnswer = item.Item.Text
					}
				}
			default:
				return out, ErrProtocolUnsupported
			}
			if e.turn == "" {
				continue
			}
			out = run.StepObservation{RecordMeta: e.meta(), Step: e.step, ProviderStep: run.ProviderStepID(e.turn), Wire: e.wire(pending), Reason: "raw provider observation; no task acceptance"}
			var projectionErr error
			if terminal {
				// Runtime queues have no cross-channel ordering. Never accept a
				// terminal event while ignoring an already queued MCP call.
				select {
				case _, ok := <-tools:
					if ok {
						return out, ErrProtocolIncomplete
					}
				default:
				}
				if len(waiting) > 0 {
					return out, ErrProtocolIncomplete
				}
				out.Terminal = true
				status := m.Turn().Status
				if m.kind == ResponseMessage {
					r, _ := m.ReadResponse(id, TurnStart)
					status = r.Turn.Status
				}
				out.Reason = "provider terminal status=" + string(status) + "; proposal requires Host assessment"
				if status == TurnCompleted {
					out.Proposed, projectionErr = projectExplore(lastAnswer, e)
					if projectionErr != nil {
						out.Reason = "provider completed; final report could not be projected"
					}
				}
			}
			e.observed, err = channel.ObserveStep(ctx, out)
			if err == nil && e.observed.BatchID == "" {
				err = run.ErrObservationLimit
			}
			if err != nil || terminal {
				return out, errors.Join(err, projectionErr)
			}
			pending = nil
		}
	}
}

func jsonEqual(a, b []byte) bool {
	var x, y any
	if _, err := protocolObject(a); err != nil {
		return false
	}
	if _, err := protocolObject(b); err != nil {
		return false
	}
	da := json.NewDecoder(bytes.NewReader(a))
	da.UseNumber()
	db := json.NewDecoder(bytes.NewReader(b))
	db.UseNumber()
	if da.Decode(&x) != nil || db.Decode(&y) != nil {
		return false
	}
	aa, _ := json.Marshal(x)
	bb, _ := json.Marshal(y)
	return bytes.Equal(aa, bb)
}
func (e *sessionEntry) answerTool(ctx context.Context, channel run.SessionIO, call frame.ToolCall, providerID run.ProviderCallID) error {
	allowed := false
	for _, tool := range e.tools {
		if tool.Name == call.Name {
			allowed = true
		}
	}
	if !allowed {
		return ErrProtocol
	}
	// This is the reconstructed, typed agentd tool-call payload, not a claimed
	// original MCP or provider NDJSON envelope.
	input, err := frame.EncodeToolCall(call)
	if err != nil {
		return err
	}
	if int64(len(input)) > e.plan.MaxWireBytes {
		return ErrProtocolLimit
	}
	result, err := channel.Call(ctx, run.ToolCall{RecordMeta: e.meta(), Step: e.step, Call: providerID, Tool: call.Name, Input: e.wire(input)})
	if err != nil || result.Receipt.BatchID == "" || !result.Output.Valid() {
		return errors.Join(err, run.ErrObservationLimit)
	}
	// Validate the Host's closed recorded representation before returning its
	// exact bytes in MCP text. A refusal is a tool error, never an accepted reply.
	encoded, err := json.Marshal(result.Outcome)
	if err != nil || !bytes.Equal(encoded, result.Output.Bytes()) {
		return ErrProtocol
	}
	var mcp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if result.Outcome.Status != run.ToolOK && result.Outcome.Status != run.ToolRefused {
		return ErrProtocol
	}
	if result.Outcome.Status == run.ToolRefused && result.Outcome.Reason == "the run admits no new work" {
		return run.ErrState
	}
	mcp.Content = append(mcp.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{"text", string(result.Output.Bytes())})
	mcp.IsError = result.Outcome.Status == run.ToolRefused
	answer, _ := json.Marshal(mcp)
	reply := frame.ToolReply{ID: call.ID, Result: answer}
	wire, err := frame.EncodeToolReply(reply)
	if err != nil {
		return err
	}
	return e.send(ctx, channel, e.step, providerID, wire, func() error { return e.peer.Reply(reply) })
}

func exploreContract(p run.Payload) error {
	if p.Ref().MediaType != "application/vnd.markitect.explore-contract+json" || p.Ref().Encoding != "utf-8" {
		return fmt.Errorf("%w: unsupported output contract", ErrSession)
	}
	var contract struct {
		Format           string
		Fields           map[string]string
		AdditionalFields bool
	}
	if strictValue(p.Bytes(), &contract) != nil || contract.Format != "markitect.explore-report/v1" || contract.AdditionalFields {
		return ErrSession
	}
	// Require the supported catalogue's field types, without interpreting it as
	// provider JSON Schema or accepting a future changed contract silently.
	expected := map[string]string{"Version": "integer=1", "Assignment": "string", "Run": "string", "RunBasis": "ContentID", "Manager": "string", "Parent": "string|null", "HandoffID": "string", "Session": "SessionID", "SessionBinding": "ContentID", "Step": "StepID", "InputContentID": "ContentID", "OperationContentID": "ContentID", "ContextContentID": "ContentID", "Summary": "nonblank-string", "ProposedChanges": "string[]", "Prerequisites": "string[]", "Uncertainties": "string[]", "Questions": "string[]"}
	if len(contract.Fields) != len(expected) {
		return ErrSession
	}
	for k, v := range expected {
		if contract.Fields[k] != v {
			return ErrSession
		}
	}
	return nil
}
func projectExplore(text string, e *sessionEntry) (run.Payload, error) {
	if strings.TrimSpace(text) == "" || int64(len(text)) > e.plan.MaxWireBytes || !bytes.Equal(e.redact([]byte(text)), []byte(text)) {
		return run.Payload{}, ErrSession
	}
	var report run.ExploreReport
	if strictValue([]byte(text), &report) != nil {
		return run.Payload{}, ErrSession
	}
	proposal, err := run.NewExploreReportPayload(report)
	if err != nil || !bytes.Equal(e.redact(proposal.Bytes()), proposal.Bytes()) {
		return run.Payload{}, ErrSession
	}
	return proposal, nil
}
