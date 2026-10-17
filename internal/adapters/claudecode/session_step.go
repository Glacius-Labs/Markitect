package claudecode

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"strings"
)

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
	if e.stopped || e.peer == nil || !e.ready || e.step != "" || a.Session != e.plan || a.Plan.SessionBinding != e.plan.Binding() || a.Plan.ID == "" || a.Receipt.BatchID == "" || a.Invocation == "" || !a.Input.Valid() || a.Input.Ref() != a.Plan.Input || !a.OutputContract.Valid() || a.OutputContract.Ref() != a.Plan.OutputContract {
		return out, ErrSession
	}
	if err = exploreContract(a.OutputContract); err != nil {
		return out, err
	}
	e.step = a.Plan.ID
	var pending []byte
	defer func() {
		if err != nil && !out.Terminal {
			out = run.StepObservation{RecordMeta: e.meta(), Step: e.step, ProviderStep: e.turn, Uncertain: true, Reason: "step observation incomplete; no automatic replay"}
			if len(pending) > 0 {
				out.Wire = e.wire(pending)
			}
			_, recordErr := channel.ObserveStep(context.WithoutCancel(ctx), out)
			err = errors.Join(e.safe(err), recordErr)
		}
	}()
	prompt := "Perform exactly one admitted Markitect step. Return the requested report as one JSON object in your final answer. The Host will assess it separately.\nExact input bytes:\n" + string(a.Input.Bytes()) + "\nExact output contract bytes (a field catalogue, not JSON Schema):\n" + string(a.OutputContract.Bytes())
	raw, err := e.codec.EncodeUser(UserInput{Text: prompt})
	if err != nil {
		return out, err
	}
	if err = e.send(ctx, channel, e.step, "", raw, func() error { return e.peer.WriteInput(raw) }); err != nil {
		return out, err
	}
	readCtx, cancel := context.WithCancel(ctx)
	readerDone := make(chan struct{})
	defer func() { cancel(); <-readerDone }()
	events := make(chan stepEvent)
	go func() {
		defer close(readerDone)
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
	seen := map[uint64]bool{}
	var waiting []frame.ToolCall
	for {
		if e.bound.BatchID != "" {
			for len(waiting) > 0 {
				call := waiting[0]
				waiting = waiting[1:]
				if err = e.answerTool(ctx, channel, call, run.ProviderCallID(fmt.Sprintf("claude-channel-call-%d", call.ID))); err != nil {
					return out, err
				}
			}
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case call, ok := <-tools:
			if !ok {
				tools = nil
				continue
			}
			if call.ID == 0 || seen[call.ID] || len(waiting) >= 16 {
				return out, ErrProtocol
			}
			seen[call.ID] = true
			waiting = append(waiting, call)
		case event, ok := <-events:
			if !ok {
				return out, ErrIncomplete
			}
			m := event.message
			pending = append(pending, m.Raw()...)
			if int64(len(pending)) > e.plan.MaxWireBytes {
				pending = pending[:e.plan.MaxWireBytes]
				return out, ErrLimit
			}
			if event.err != nil {
				return out, event.err
			}
			switch m.kind {
			case SystemInitMessage:
				init, readErr := m.Init()
				if readErr != nil || e.thread != "" || init.Version != "2.1.301" || !bytes.Equal(e.redact([]byte(init.Session)), []byte(init.Session)) || e.plan.Resume != "" && string(init.Session) != string(e.plan.Resume) {
					return out, ErrProtocol
				}
				e.thread = init.Session
				observation := run.SessionObservation{RecordMeta: e.meta(), ProviderSession: run.ProviderSessionID(e.thread), Wire: e.wire(m.Raw()), Reason: "actual Claude system/init session; resume identity requires separate runtime proof"}
				e.bound, err = channel.Bind(ctx, observation)
				if err != nil || e.bound.BatchID == "" {
					return out, errors.Join(err, run.ErrObservationLimit)
				}
				// This opaque adapter handle is backed by actual init plus the exclusive
				// container channel and exactly one admitted write. No UUID is a turn ID.
				e.turn = run.ProviderStepID(fmt.Sprintf("claude-single-step-%x", sha256.Sum256(append(append([]byte(string(e.plan.ID)+"\x00"+string(e.step)+"\x00"), m.Raw()...), raw...))))
				out = run.StepObservation{RecordMeta: e.meta(), Step: e.step, ProviderStep: e.turn, Wire: e.wire(pending), Reason: "adapter single-step channel observed; no native Claude turn identity"}
				e.observed, err = channel.ObserveStep(ctx, out)
				if err != nil || e.observed.BatchID == "" {
					return out, errors.Join(err, run.ErrObservationLimit)
				}
				pending = nil
			case UserMessage, AssistantMessage, StreamEventMessage:
				if e.bound.BatchID == "" {
					return out, ErrProtocol
				}
				body, _ := object(m.Raw())
				if session, ok := body["session_id"]; ok {
					id, readErr := nonempty(session)
					if readErr != nil || id != string(e.thread) {
						return out, ErrProtocol
					}
				}
			case ResultMessage:
				result, readErr := m.Result()
				if readErr != nil || e.bound.BatchID == "" || result.Session != e.thread {
					return out, ErrProtocol
				}
				// The runtime queues have no cross-channel ordering. Do not silently drop
				// an already queued Host call when a terminal result arrives.
				select {
				case _, ok := <-tools:
					if ok {
						return out, ErrIncomplete
					}
				default:
				}
				if len(waiting) > 0 {
					return out, ErrIncomplete
				}
				out = run.StepObservation{RecordMeta: e.meta(), Step: e.step, ProviderStep: e.turn, Wire: e.wire(pending), Terminal: true, Reason: "Claude terminal result; adapter handle, no native turn ID; proposal requires Host assessment"}
				var projectionErr error
				if result.Subtype == ResultSuccess && !result.IsError {
					out.Proposed, projectionErr = projectExplore(result.Text, e)
				}
				e.observed, err = channel.ObserveStep(ctx, out)
				if err == nil && e.observed.BatchID == "" {
					err = run.ErrObservationLimit
				}
				return out, errors.Join(err, projectionErr)
			default:
				return out, ErrUnsupported
			}
		}
	}
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
		return ErrLimit
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
