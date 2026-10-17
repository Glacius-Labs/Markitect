package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const sessionContract = `{"Format":"markitect.explore-report/v1","Fields":{"Version":"integer=1","Assignment":"string","Run":"string","RunBasis":"ContentID","Manager":"string","Parent":"string|null","HandoffID":"string","Session":"SessionID","SessionBinding":"ContentID","Step":"StepID","InputContentID":"ContentID","OperationContentID":"ContentID","ContextContentID":"ContentID","Summary":"nonblank-string","ProposedChanges":"string[]","Prerequisites":"string[]","Uncertainties":"string[]","Questions":"string[]"},"AdditionalFields":false}`

type fakeChannel struct {
	preparations     []run.TransportPreparation
	sends            []run.TransportObservation
	bindings         []run.SessionObservation
	steps            []run.StepObservation
	calls            []run.ToolCall
	stops            []run.StopObservation
	permitMode       string
	observeFail      bool
	observeNoReceipt bool
	callFail         bool
	refuse           bool
	stopRecordError  error
}

func receipt() run.AppendReceipt { return run.AppendReceipt{BatchID: "recorded", Revision: 1} }
func (f *fakeChannel) Ready(context.Context, run.SessionReadiness) (run.AppendReceipt, error) {
	return receipt(), nil
}
func (f *fakeChannel) Bind(_ context.Context, o run.SessionObservation) (run.AppendReceipt, error) {
	f.bindings = append(f.bindings, o)
	return receipt(), nil
}
func (f *fakeChannel) PrepareSend(_ context.Context, p run.TransportPreparation) (run.SendPermit, error) {
	f.preparations = append(f.preparations, p)
	permit := run.SendPermit{Receipt: receipt(), Identity: p.Wire.Identity, Dispatch: true}
	switch f.permitMode {
	case "deny":
		permit.Dispatch = false
	case "identity":
		permit.Identity.Size++
	case "receipt":
		permit.Receipt = run.AppendReceipt{}
	case "error":
		return run.SendPermit{}, errors.New("append failed")
	}
	return permit, nil
}
func (f *fakeChannel) ObserveSend(_ context.Context, o run.TransportObservation) (run.AppendReceipt, error) {
	f.sends = append(f.sends, o)
	if f.observeFail {
		return run.AppendReceipt{}, errors.New("observation append failed")
	}
	if f.observeNoReceipt {
		return run.AppendReceipt{}, nil
	}
	return receipt(), nil
}
func (f *fakeChannel) Call(_ context.Context, c run.ToolCall) (run.ToolResult, error) {
	f.calls = append(f.calls, c)
	outcome := run.ToolOutcome{Status: run.ToolOK, Result: json.RawMessage(`{"echo":"host-value"}`)}
	if f.refuse {
		outcome = run.ToolOutcome{Status: run.ToolRefused, Reason: "Host refused this call"}
	}
	data, _ := json.Marshal(outcome)
	p, _ := run.NewPayload("application/vnd.markitect.tool-outcome+json", "utf-8", data)
	r := run.ToolResult{Receipt: receipt(), Outcome: outcome, Output: p}
	if f.callFail {
		r.Receipt = run.AppendReceipt{}
		return r, errors.New("answer not recorded")
	}
	return r, nil
}
func (f *fakeChannel) ObserveStep(_ context.Context, o run.StepObservation) (run.AppendReceipt, error) {
	f.steps = append(f.steps, o)
	return receipt(), nil
}
func (f *fakeChannel) ObserveStop(_ context.Context, o run.StopObservation) (run.AppendReceipt, error) {
	f.stops = append(f.stops, o)
	if f.stopRecordError != nil {
		return run.AppendReceipt{}, f.stopRecordError
	}
	return receipt(), nil
}

type fakeStream struct {
	chunks  chan []byte
	pending []byte
	done    chan struct{}
	once    sync.Once
}

func (f *fakeStream) Read(p []byte) (int, error) {
	if len(f.pending) == 0 {
		select {
		case f.pending = <-f.chunks:
		case <-f.done:
			return 0, io.EOF
		}
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}
func (f *fakeStream) close() { f.once.Do(func() { close(f.done) }) }

type fakeRuntime struct {
	stream         *fakeStream
	tools          chan frame.ToolCall
	writes         [][]byte
	replies        []frame.ToolReply
	start          frame.Start
	journal        *fakeChannel
	useTool        bool
	badTool        bool
	writeFail      bool
	stopFail       bool
	resumeMismatch bool
	cancelTurn     bool
	writeBlock     <-chan struct{}
	startBlocked   bool
	stopBlocked    bool
	secretField    string
	stopCalls      int
}

func peer(f *fakeChannel) *fakeRuntime {
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 32), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16), journal: f}
}
func (f *fakeRuntime) emit(raw string) {
	if f.secretField == "thread-marker" && strings.Contains(raw, `"id":2`) {
		raw = strings.ReplaceAll(raw, string(codecThreadID), "prefix-[REDACTED]")
	}
	if f.secretField == "thread-response" && strings.Contains(raw, `"id":2`) {
		raw = strings.ReplaceAll(raw, string(codecThreadID), "prefix-secret-key-for-runtime")
	}
	if f.secretField == "turn-response" && strings.Contains(raw, `"id":3`) {
		raw = strings.ReplaceAll(raw, string(codecTurnID), "prefix-secret-key-for-runtime")
	}
	if f.secretField == "call" {
		raw = strings.ReplaceAll(raw, "actual-call", "prefix-secret-key-for-runtime")
	}
	f.stream.chunks <- []byte(raw + "\n")
}
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	f.start = s
	if f.startBlocked {
		<-f.stream.done
		return frame.Started{}, io.ErrClosedPipe
	}
	return frame.Started{PID: 123}, nil
}
func (f *fakeRuntime) Stdout() io.Reader                { return f.stream }
func (f *fakeRuntime) ToolCalls() <-chan frame.ToolCall { return f.tools }
func (f *fakeRuntime) WriteInput(data []byte) error {
	if f.writeBlock != nil {
		select {
		case <-f.writeBlock:
		case <-f.stream.done:
			return io.ErrClosedPipe
		}
	}
	if len(f.journal.preparations) != len(f.writes)+len(f.replies)+1 {
		return errors.New("write lacked fresh preparation")
	}
	f.writes = append(f.writes, bytes.Clone(data))
	if f.writeFail {
		return errors.New("partial write")
	}
	var m struct {
		ID     int
		Method string
		Params map[string]json.RawMessage
	}
	_ = json.Unmarshal(data, &m)
	switch m.Method {
	case "initialize":
		f.emit(`{"id":1,"result":` + codecInitialization + `}`)
	case "thread/start", "thread/resume":
		if f.secretField == "thread-notification" {
			f.emit(`{"method":"thread/started","params":{"thread":` + strings.ReplaceAll(codecThread, string(codecThreadID), "prefix-secret-key-for-runtime") + `}}`)
		}
		thread := codecThread
		if f.resumeMismatch {
			thread = strings.ReplaceAll(thread, string(codecThreadID), "wrong-thread")
		}
		f.emit(`{"id":2,"result":{"thread":` + thread + `,"approvalPolicy":"never","approvalsReviewer":"user","cwd":"/work","model":"admitted-model","modelProvider":"markitect","sandbox":{"type":"dangerFullAccess"}}}`)
	case "turn/start":
		if f.secretField == "turn-notification" {
			f.emit(`{"method":"turn/started","params":{"threadId":"` + string(codecThreadID) + `","turn":{"id":"prefix-secret-key-for-runtime","status":"inProgress","items":[],"error":null}}}`)
		}
		if f.secretField == "turn-item" {
			f.emit(`{"method":"item/started","params":{"threadId":"` + string(codecThreadID) + `","turnId":"prefix-secret-key-for-runtime","item":{"id":"observed-item","type":"agentMessage","text":"observed"}}}`)
		}
		f.emit(`{"id":3,"result":{"turn":{"id":"` + string(codecTurnID) + `","status":"inProgress","items":[],"error":null}}}`)
		if f.cancelTurn {
			return nil
		}
		if f.useTool {
			thread := string(codecThreadID)
			if f.badTool {
				thread = "foreign-thread"
			}
			// Deliberately enqueue MCP first, before its stdout identity event.
			callID := "actual-call"
			if f.secretField == "call" {
				callID = "prefix-secret-key-for-runtime"
			}
			f.tools <- frame.ToolCall{ID: 1, Name: "submit_work", Arguments: json.RawMessage(`{"summary":"done"}`), Meta: json.RawMessage(`{"threadId":"` + thread + `","callId":"` + callID + `"}`)}
			f.emit(`{"method":"item/started","params":{"threadId":"` + string(codecThreadID) + `","turnId":"` + string(codecTurnID) + `","item":{"id":"actual-call","type":"mcpToolCall","server":"markitect","tool":"submit_work","arguments":{"summary":"done"},"status":"inProgress"}}}`)
		} else {
			f.finish()
		}
	}
	return nil
}
func (f *fakeRuntime) finish() {
	report := run.ExploreReport{Version: 1, Summary: "observed", ProposedChanges: []string{}, Prerequisites: []string{}, Uncertainties: []string{}, Questions: []string{}}
	p, _ := run.NewExploreReportPayload(report)
	item := map[string]any{"method": "item/completed", "params": map[string]any{"threadId": codecThreadID, "turnId": codecTurnID, "item": map[string]any{"id": "answer", "type": "agentMessage", "phase": "final_answer", "text": string(p.Bytes())}}}
	data, _ := json.Marshal(item)
	f.emit(string(data))
	f.emit(`{"method":"turn/completed","params":{"threadId":"` + string(codecThreadID) + `","turn":{"id":"` + string(codecTurnID) + `","status":"completed","items":[],"error":null}}}`)
}
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.replies = append(f.replies, r)
	f.finish()
	return nil
}
func (f *fakeRuntime) Stop(context.Context, int64) (frame.Exited, error) {
	f.stopCalls++
	if f.stopBlocked {
		<-f.stream.done
		return frame.Exited{}, io.ErrClosedPipe
	}
	if f.stopFail {
		return frame.Exited{}, context.DeadlineExceeded
	}
	if f.secretField == "stop-signal" {
		return frame.Exited{Code: -1, Signal: "prefix-secret-key-for-runtime"}, nil
	}
	if f.secretField == "stop-marker" {
		return frame.Exited{Code: -1, Signal: "prefix-[REDACTED]"}, nil
	}
	if f.secretField == "stop-marker-nested" {
		return frame.Exited{Code: -1, Signal: "[RED[REDACTED]ACTED]"}, nil
	}
	return frame.Exited{Code: -1, Signal: "terminated"}, nil
}
func (f *fakeRuntime) Close() error { f.stream.close(); return nil }

func payloadForSession(t *testing.T, media string, data []byte) run.Payload {
	t.Helper()
	p, err := run.NewPayload(media, "utf-8", data)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func sessionAdmissions(t *testing.T) (run.SessionAdmission, run.StepAdmission) {
	t.Helper()
	profile, _ := json.Marshal(SessionProfile{Version: 1, Model: "admitted-model", Effort: "high", Store: "store", CPUs: 1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20})
	p := payloadForSession(t, "application/vnd.markitect.codex-session-profile+json", profile)
	tools := payloadForSession(t, "application/vnd.markitect.tools+json", []byte(`{"version":1,"tools":[{"name":"submit_work","description":"Submit to Host","inputSchema":{"type":"object"}}]}`))
	plan := run.SessionPlan{ID: "session", Role: "manager", Workspace: "workspace", ProviderHome: "provider-home", ImageID: "sha256:" + strings.Repeat("a", 64), CredentialRef: "exact-credential", Candidate: snapshot.TreeID{1}, MaxWireBytes: 1 << 20, Profile: p.Ref(), Tools: tools.Ref()}
	a := run.SessionAdmission{Plan: plan, Profile: p, Tools: tools, Receipt: receipt(), Invocation: "open"}
	input := payloadForSession(t, "application/vnd.markitect.explore-input+json", []byte(`{"exact":"input"}`))
	contract := payloadForSession(t, "application/vnd.markitect.explore-contract+json", []byte(sessionContract))
	b := run.StepAdmission{Session: plan, Plan: run.StepPlan{Session: plan.ID, ID: "step", SessionBinding: plan.Binding(), Input: input.Ref(), OutputContract: contract.Ref()}, Input: input, OutputContract: contract, Receipt: receipt(), Invocation: "step"}
	return a, b
}
func sessionsForTest(t *testing.T, f *fakeRuntime) *Sessions {
	t.Helper()
	s, err := NewSessionsWithRuntime(func(_ context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		if spec.Image != "sha256:"+strings.Repeat("a", 64) || len(spec.Mounts) != 2 || spec.Mounts[0].Volume != "workspace" || spec.Mounts[1].Volume != "provider-home" {
			t.Errorf("unbound runtime spec: %+v", spec)
		}
		return f, nil
	}, nil, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "exact-credential" {
			t.Errorf("unbound credential %s", ref)
		}
		return RuntimeCredential{Mode: "api-key", APIKey: "secret-key-for-runtime"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return s
}

func TestAdmittedCodexSessionOneStepHostToolsAndObservedExit(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		t.Run(map[bool]string{false: "answer", true: "refusal"}[refuse], func(t *testing.T) {
			channel := &fakeChannel{refuse: refuse}
			runtime := peer(channel)
			runtime.useTool = true
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			opened, err := s.Open(ctx, a, channel)
			if err != nil || opened.ProviderSession != run.ProviderSessionID(codecThreadID) {
				t.Fatalf("open %+v %v", opened, err)
			}
			if len(runtime.start.Env) != 2 || !strings.Contains(runtime.start.Env[1], "secret-key-for-runtime") || runtime.start.Argv[0] != "/usr/local/bin/codex" {
				t.Fatalf("provider start %+v", runtime.start)
			}
			step, err := s.Step(ctx, b, channel)
			if err != nil || !step.Terminal || step.ProviderStep != run.ProviderStepID(codecTurnID) || !step.Proposed.Valid() {
				t.Fatalf("step %+v %v", step, err)
			}
			if len(channel.calls) != 1 || channel.calls[0].Call != "actual-call" || len(runtime.replies) != 1 {
				t.Fatalf("tool exchange %+v %+v", channel.calls, runtime.replies)
			}
			var reply struct {
				Content []struct{ Text string }
				IsError bool
			}
			if json.Unmarshal(runtime.replies[0].Result, &reply) != nil || reply.IsError != refuse || len(reply.Content) != 1 {
				t.Fatal("Host outcome not preserved")
			}
			var outcome run.ToolOutcome
			_ = json.Unmarshal([]byte(reply.Content[0].Text), &outcome)
			if refuse && outcome.Reason != "Host refused this call" || !refuse && string(outcome.Result) != `{"echo":"host-value"}` {
				t.Fatal("Host answer changed")
			}
			last := channel.preparations[len(channel.preparations)-1]
			if last.Call != "actual-call" || last.Step != "step" {
				t.Fatal("tool answer lacks association")
			}
			if _, err = s.Step(ctx, b, channel); !errors.Is(err, ErrSession) {
				t.Fatal("second step admitted")
			}
			before := len(runtime.writes)
			_, err = s.Inspect(ctx, run.InspectionRequest{Session: a.Plan.ID, Step: b.Plan.ID, ProviderSession: opened.ProviderSession, ProviderStep: step.ProviderStep, SessionBinding: receipt(), StepBinding: receipt()})
			if !errors.Is(err, run.ErrObservationLimit) || len(runtime.writes) != before {
				t.Fatal("Inspect sent or recreated")
			}
			stopped, err := s.Stop(ctx, run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
			if err != nil || !stopped.Terminated || !strings.Contains(stopped.Reason, "signal=terminated") {
				t.Fatalf("stop %+v %v", stopped, err)
			}
		})
	}
}

func TestCodexRequiresExactAcknowledgedPermitsAndNeverRetriesPartialWrites(t *testing.T) {
	for _, mode := range []string{"deny", "identity", "receipt", "error", "partial", "observation", "observation-receipt"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			s := sessionsForTest(t, runtime)
			a, _ := sessionAdmissions(t)
			if mode == "partial" {
				runtime.writeFail = true
			} else if mode == "observation" {
				channel.observeFail = true
			} else if mode == "observation-receipt" {
				channel.observeNoReceipt = true
			} else {
				channel.permitMode = mode
			}
			_, err := s.Open(context.Background(), a, channel)
			if err == nil {
				t.Fatal("unacknowledged send accepted")
			}
			want := 0
			if mode == "partial" || mode == "observation" || mode == "observation-receipt" {
				want = 1
			}
			if len(runtime.writes) != want {
				t.Fatalf("writes %d want %d", len(runtime.writes), want)
			}
			if mode == "partial" && (len(channel.sends) != 1 || channel.sends[0].Outcome != run.SendUncertain) {
				t.Fatal("partial write lost uncertainty")
			}
			_, err = s.Open(context.Background(), a, channel)
			if !errors.Is(err, run.ErrObservationLimit) || len(runtime.writes) != want {
				t.Fatal("Open replayed")
			}
		})
	}
}

func TestCodexToolCorrelationAndUnrecordedHostOutcomeStopDispatch(t *testing.T) {
	for _, mode := range []string{"foreign-thread", "answer-append-failure"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{callFail: mode == "answer-append-failure"}
			runtime := peer(channel)
			runtime.useTool = true
			runtime.badTool = mode == "foreign-thread"
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if _, err := s.Open(ctx, a, channel); err != nil {
				t.Fatal(err)
			}
			result, err := s.Step(ctx, b, channel)
			if err == nil || !result.Uncertain || len(runtime.replies) != 0 {
				t.Fatalf("unsafe tool effect %+v %v", result, err)
			}
			if mode == "foreign-thread" && len(channel.calls) != 0 {
				t.Fatal("foreign call reached Host")
			}
			if mode == "answer-append-failure" && len(channel.calls) != 1 {
				t.Fatal("Host callback not observed")
			}
		})
	}
}

func TestCodexResumeMismatchAdmissionBindingAndReadOnlyMount(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	runtime.resumeMismatch = true
	s := sessionsForTest(t, runtime)
	a, _ := sessionAdmissions(t)
	a.Plan.Resume = run.ProviderSessionID(codecThreadID)
	result, err := s.Open(context.Background(), a, channel)
	if err == nil || !result.Uncertain || result.ProviderSession != "wrong-thread" {
		t.Fatalf("resume mismatch %+v %v", result, err)
	}
	channel = &fakeChannel{}
	runtime = peer(channel)
	s = sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	a.Profile = b.Input
	if _, err = s.Open(context.Background(), a, channel); err == nil || len(runtime.writes) != 0 {
		t.Fatal("different payload bytes admitted")
	}
	a, _ = sessionAdmissions(t)
	a.Plan.Role = "reviewer"
	a.Plan.ReadOnly = true
	profile, _, err := readSessionProfile(a)
	if err != nil || !sessionSpec(a.Plan, profile).Mounts[0].ReadOnly {
		t.Fatal("reviewer workspace writable")
	}
	a.Plan.Resume = "retained-thread"
	if _, _, err = readSessionProfile(a); err == nil {
		t.Fatal("reviewer resumed")
	}
}

func TestCodexCancellationStopUncertaintyAndSecretRedaction(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	runtime.cancelTurn = true
	runtime.stopFail = true
	s := sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	opened, err := s.Open(context.Background(), a, channel)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := s.Step(ctx, b, channel)
	if err == nil || !result.Uncertain {
		t.Fatalf("cancel lost uncertainty %+v %v", result, err)
	}
	stopped, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
	if err == nil || stopped.Terminated {
		t.Fatalf("stop fabricated exit %+v %v", stopped, err)
	}
	e, _ := s.entry(a.Plan.ID)
	wire := e.wire([]byte(`{"echo":"secret-key-for-runtime"}`))
	if !wire.Redacted || bytes.Contains(wire.Journal.Bytes(), e.secrets[0]) || wire.Identity.ContentID == wire.Journal.Ref().ContentID {
		t.Fatal("secret or true identity lost")
	}
}

func TestCodexToolArgumentCorrelationPreservesIntegers(t *testing.T) {
	if jsonEqual([]byte(`{"n":9007199254740992}`), []byte(`{"n":9007199254740993}`)) {
		t.Fatal("different exact integers collapsed")
	}
	if !jsonEqual([]byte(`{"n":9007199254740993,"a":true}`), []byte(`{"a":true,"n":9007199254740993}`)) {
		t.Fatal("identical arguments depend on object field order")
	}
	if jsonEqual([]byte(`{"n":1,"n":2}`), []byte(`{"n":2}`)) {
		t.Fatal("duplicate arguments keys collapsed")
	}
}

func TestCodexCanceledPhysicalWriteReturnsUncertaintyThenStop(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	s := sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	opened, err := s.Open(context.Background(), a, channel)
	if err != nil {
		t.Fatal(err)
	}
	runtime.writeBlock = make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	out, err := s.Step(ctx, b, channel)
	if err == nil || !out.Uncertain || channel.sends[len(channel.sends)-1].Outcome != run.SendUncertain {
		t.Fatalf("blocked write cancellation %+v %v", out, err)
	}
	stopped, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
	if err != nil || !stopped.Terminated {
		t.Fatalf("stop after blocked write %+v %v", stopped, err)
	}
}

func TestCodexBlockedRuntimeControlsRespectCancellation(t *testing.T) {
	for _, operation := range []string{"start", "stop"} {
		t.Run(operation, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.startBlocked = operation == "start"
			s := sessionsForTest(t, runtime)
			a, _ := sessionAdmissions(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			opened, err := s.Open(ctx, a, channel)
			if operation == "start" {
				if err == nil || !opened.Uncertain {
					t.Fatalf("blocked start %+v %v", opened, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			runtime.stopBlocked = operation == "stop"
			stopped, stopErr := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: 20 * time.Millisecond}, channel)
			if operation == "stop" && (stopErr == nil || stopped.Terminated) {
				t.Fatalf("blocked stop %+v %v", stopped, stopErr)
			}
		})
	}
}

func TestCodexSecretEchoCannotEscapeThroughCallbackIdentityFields(t *testing.T) {
	for _, field := range []string{"thread-response", "thread-notification", "turn-response", "turn-notification", "turn-item", "call"} {
		t.Run(field, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.secretField = field
			runtime.useTool = field == "call"
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			opened, err := s.Open(ctx, a, channel)
			if strings.HasPrefix(field, "thread-") {
				if err == nil || !opened.Uncertain || opened.ProviderSession != "" {
					t.Fatalf("unsafe thread accepted: uncertain=%t bound=%t error=%v", opened.Uncertain, opened.ProviderSession != "", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				step, stepErr := s.Step(ctx, b, channel)
				if stepErr == nil || !step.Uncertain {
					t.Fatalf("unsafe native identity accepted: uncertain=%t error=%v", step.Uncertain, stepErr)
				}
				if field != "call" && step.ProviderStep != "" {
					t.Fatal("unsafe turn replaced by invented identity")
				}
			}
			// Assert the actual values handed to every journal callback, including
			// uncertain deferred observations, rather than just Wire.Redacted.
			for _, bound := range channel.bindings {
				if strings.Contains(string(bound.ProviderSession), "secret-key-for-runtime") || bytes.Contains(bound.Wire.Journal.Bytes(), []byte("secret-key-for-runtime")) {
					t.Fatal("thread secret reached Bind")
				}
			}
			for _, step := range channel.steps {
				if strings.Contains(string(step.ProviderStep), "secret-key-for-runtime") || bytes.Contains(step.Wire.Journal.Bytes(), []byte("secret-key-for-runtime")) {
					t.Fatal("turn secret reached ObserveStep")
				}
			}
			if field == "call" && len(channel.calls) != 0 {
				t.Fatal("unsafe native call ID reached Host callback")
			}
			for _, prepared := range channel.preparations {
				if strings.Contains(string(prepared.Call), "secret-key-for-runtime") {
					t.Fatal("call secret reached PrepareSend")
				}
			}
		})
	}
}

func TestCodexStopSignalSecretIsRedactedBeforeObservation(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	runtime.secretField = "stop-signal"
	s := sessionsForTest(t, runtime)
	a, _ := sessionAdmissions(t)
	opened, err := s.Open(context.Background(), a, channel)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
	if err != nil || !stopped.Terminated || len(channel.stops) != 1 || strings.Contains(stopped.Reason, "secret-key-for-runtime") || strings.Contains(channel.stops[0].Reason, "secret-key-for-runtime") || !strings.Contains(stopped.Reason, "[REDACTED]") {
		t.Fatalf("unsafe stop observation %+v %v", stopped, err)
	}
}

func TestCodexStopJournalErrorCannotReturnKnownCredential(t *testing.T) {
	channel := &fakeChannel{stopRecordError: errors.New("journal refused secret-key-for-runtime")}
	runtime := peer(channel)
	s := sessionsForTest(t, runtime)
	a, _ := sessionAdmissions(t)
	opened, err := s.Open(context.Background(), a, channel)
	if err != nil {
		t.Fatal(err)
	}
	stopped, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
	if !stopped.Terminated || err == nil || strings.Contains(err.Error(), "secret-key-for-runtime") || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatal("stop journal error escaped credential redaction")
	}
}

func TestCodexCredentialCannotEqualRedactionMarker(t *testing.T) {
	for _, field := range []string{"thread-marker", "stop-marker", "stop-marker-nested"} {
		t.Run(field, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.secretField = field
			s, err := NewSessionsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) { return runtime, nil }, nil, func(context.Context, string) (RuntimeCredential, error) {
				return RuntimeCredential{Mode: "api-key", APIKey: "[REDACTED]"}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = runtime.Close() })
			a, _ := sessionAdmissions(t)
			opened, openErr := s.Open(context.Background(), a, channel)
			if field == "thread-marker" {
				if openErr == nil || !opened.Uncertain || opened.ProviderSession != "" {
					t.Fatal("replacement token accepted as native identity")
				}
				for _, bound := range channel.bindings {
					if strings.Contains(string(bound.ProviderSession), "[REDACTED]") || bytes.Contains(bound.Wire.Journal.Bytes(), []byte("[REDACTED]")) {
						t.Fatal("marker-valued credential reached Bind")
					}
				}
			} else {
				if openErr != nil {
					t.Fatal(openErr)
				}
				stopped, stopErr := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: opened.ProviderSession, Timeout: time.Second}, channel)
				if stopErr != nil || !stopped.Terminated || strings.Contains(stopped.Reason, "[REDACTED]") || strings.Contains(channel.stops[0].Reason, "[REDACTED]") {
					t.Fatal("marker-valued credential reached ObserveStop")
				}
			}
		})
	}
}
