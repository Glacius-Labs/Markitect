package claudecode

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

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
	secretAnswer     json.RawMessage
	halted           bool
	readyFail        bool
	bindFail         bool
	stepFail         bool
	ready            []run.SessionReadiness
}

func receipt() run.AppendReceipt { return run.AppendReceipt{BatchID: "recorded", Revision: 1} }
func (f *fakeChannel) Ready(_ context.Context, o run.SessionReadiness) (run.AppendReceipt, error) {
	f.ready = append(f.ready, o)
	if f.readyFail {
		return run.AppendReceipt{}, errors.New("ready append failed")
	}
	return receipt(), nil
}
func (f *fakeChannel) Bind(_ context.Context, o run.SessionObservation) (run.AppendReceipt, error) {
	f.bindings = append(f.bindings, o)
	if f.bindFail {
		return run.AppendReceipt{}, errors.New("bind append failed")
	}
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
	if f.secretAnswer != nil {
		outcome.Result = f.secretAnswer
	}
	if f.refuse {
		outcome = run.ToolOutcome{Status: run.ToolRefused, Reason: "Host refused this call"}
	}
	if f.halted {
		outcome = run.ToolOutcome{Status: run.ToolRefused, Reason: "the run admits no new work"}
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
	if f.stepFail {
		return run.AppendReceipt{}, errors.New("step append failed")
	}
	return receipt(), nil
}
func (f *fakeChannel) ObserveStop(_ context.Context, o run.StopObservation) (run.AppendReceipt, error) {
	f.stops = append(f.stops, o)
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
	exitSignal                                                string
	finalText                                                 string
	toolArguments                                             json.RawMessage
	stream                                                    *fakeStream
	tools                                                     chan frame.ToolCall
	writes                                                    [][]byte
	replies                                                   []frame.ToolReply
	start                                                     frame.Start
	journal                                                   *fakeChannel
	useTool, writeFail, stopFail, cancelTurn                  bool
	noInit, wrongInit, resultWrong, wrongVersion, errorResult bool
	stopCalls                                                 int
}

func peer(f *fakeChannel) *fakeRuntime {
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 32), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16), journal: f}
}
func (f *fakeRuntime) emit(raw string) { f.stream.chunks <- []byte(raw + "\n") }
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	f.start = s
	return frame.Started{PID: 123}, nil
}
func (f *fakeRuntime) Stdout() io.Reader                { return f.stream }
func (f *fakeRuntime) ToolCalls() <-chan frame.ToolCall { return f.tools }
func (f *fakeRuntime) WriteInput(data []byte) error {
	if len(f.journal.preparations) != len(f.writes)+len(f.replies)+1 {
		return errors.New("write lacked fresh preparation")
	}
	f.writes = append(f.writes, bytes.Clone(data))
	if f.writeFail {
		return errors.New("partial write secret-token")
	}
	var m struct {
		Type      string
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(data, &m)
	if m.Type == "control_request" {
		f.emit(`{"type":"control_response","response":{"subtype":"success","request_id":"markitect-initialize","response":{"commands":[],"agents":[],"models":[],"output_style":"default","available_output_styles":[],"account":{},"claude_code_version":"2.1.301"}}}`)
		return nil
	}
	if !f.noInit {
		session := "conversation"
		if f.wrongInit {
			session = "foreign-conversation"
		}
		version := "2.1.301"
		if f.wrongVersion {
			version = "wrong"
		}
		f.emit(`{"type":"system","subtype":"init","session_id":"` + session + `","uuid":"init-event","claude_code_version":"` + version + `"}`)
	}
	if f.cancelTurn {
		return nil
	}
	if f.useTool {
		args := f.toolArguments
		if args == nil {
			args = json.RawMessage(`{"summary":"done"}`)
		}
		f.tools <- frame.ToolCall{ID: 1, Name: "submit", Arguments: args}
	} else {
		f.finish()
	}
	return nil
}
func (f *fakeRuntime) finish() {
	text := "ordinary final text is not an outcome"
	if f.finalText != "" {
		text = f.finalText
	}
	session := "conversation"
	if f.resultWrong {
		session = "foreign-conversation"
	}
	data, _ := json.Marshal(map[string]any{"type": "result", "subtype": "success", "session_id": session, "uuid": "result-event-not-a-turn", "is_error": f.errorResult, "result": text})
	f.emit(string(data))
}
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.replies = append(f.replies, r)
	f.finish()
	return nil
}
func (f *fakeRuntime) Stop(context.Context, int64) (frame.Exited, error) {
	f.stopCalls++
	if f.stopFail {
		return frame.Exited{}, context.DeadlineExceeded
	}
	signal := f.exitSignal
	if signal == "" {
		signal = "terminated"
	}
	return frame.Exited{Code: -1, Signal: signal}, nil
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
	p, err := NewSessionProfile(SessionProfile{Version: 1, Model: "admitted-model", Effort: "high", SystemPrompt: "Use the exact Host instruction.", BuiltinTools: []string{"Read", "Write"}, Store: "store", CPUs: 1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	tools := payloadForSession(t, "application/vnd.markitect.tools+json", []byte(`{"version":1,"tools":[{"name":"submit","description":"Submit to Host","inputSchema":{"type":"object"}}]}`))
	plan := run.SessionPlan{ID: "session", Role: "manager", Workspace: "workspace", ProviderHome: "provider-home", ImageID: "sha256:" + strings.Repeat("a", 64), CredentialRef: "exact-credential", MaxWireBytes: 1 << 20, Profile: p.Ref(), Tools: tools.Ref()}
	plan.Candidate[0] = 1
	a := run.SessionAdmission{Plan: plan, Profile: p, Tools: tools, Receipt: receipt(), Invocation: "open"}
	input := payloadForSession(t, "application/vnd.markitect.explore-input+json", []byte(`{"exact":"input"}`))
	contract, err := run.NewStepContract(run.PhaseExplore)
	if err != nil {
		t.Fatal(err)
	}
	b := run.StepAdmission{Session: plan, Plan: run.StepPlan{Session: plan.ID, ID: "step", SessionBinding: plan.Binding(), Input: input.Ref(), OutputContract: contract.Ref()}, Input: input, OutputContract: contract, Receipt: receipt(), Invocation: "step"}
	return a, b
}

func TestClaudeSessionProfileFactoryRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name, role string
		readOnly   bool
		tools      []string
	}{
		{"read-only-manager", "manager", true, []string{"Read"}},
		{"fresh-reviewer", "reviewer", true, []string{"Read"}},
		{"writing-manager", "manager", false, []string{"Read", "Write"}},
		{"explicit-no-builtins", "manager", true, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := sessionAdmissions(t)
			var profile SessionProfile
			if err := json.Unmarshal(a.Profile.Bytes(), &profile); err != nil {
				t.Fatal(err)
			}
			profile.BuiltinTools = tc.tools
			profile.Network = "recorded-network"
			payload, err := NewSessionProfile(profile)
			if err != nil {
				t.Fatal(err)
			}
			want, err := json.Marshal(profile)
			if err != nil {
				t.Fatal(err)
			}
			if !payload.Valid() || payload.Ref().MediaType != "application/vnd.markitect.claude-session-profile+json" || payload.Ref().Encoding != "utf-8" || !bytes.Equal(payload.Bytes(), want) {
				t.Fatal("factory changed explicit profile bytes or interpretation")
			}
			a.Profile, a.Plan.Profile = payload, payload.Ref()
			a.Plan.Role, a.Plan.ReadOnly = tc.role, tc.readOnly
			got, tools, err := readSessionProfile(a)
			if err != nil || !reflect.DeepEqual(got, profile) || len(tools) != 1 || tools[0].Name != "submit" {
				t.Fatalf("profile did not round-trip: %+v %v", got, err)
			}
		})
	}
}

func TestClaudeSessionProfileFactoryAndReaderRejectInvalidSettings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*SessionProfile)
	}{
		{"version", func(p *SessionProfile) { p.Version = 0 }},
		{"model", func(p *SessionProfile) { p.Model = " " }},
		{"model-utf8", func(p *SessionProfile) { p.Model = string([]byte{0xff}) }},
		{"effort", func(p *SessionProfile) { p.Effort = "medium" }},
		{"system-prompt", func(p *SessionProfile) { p.SystemPrompt = "\t" }},
		{"system-prompt-utf8", func(p *SessionProfile) { p.SystemPrompt = string([]byte{0xff}) }},
		{"absent-builtins", func(p *SessionProfile) { p.BuiltinTools = nil }},
		{"unknown-builtin", func(p *SessionProfile) { p.BuiltinTools = []string{"Bash"} }},
		{"duplicate-builtin", func(p *SessionProfile) { p.BuiltinTools = []string{"Read", "Read"} }},
		{"store", func(p *SessionProfile) { p.Store = "" }},
		{"cpus", func(p *SessionProfile) { p.CPUs = 0 }},
		{"cpus-nan", func(p *SessionProfile) { p.CPUs = math.NaN() }},
		{"cpus-infinity", func(p *SessionProfile) { p.CPUs = math.Inf(1) }},
		{"memory", func(p *SessionProfile) { p.MemoryBytes = 0 }},
		{"pids", func(p *SessionProfile) { p.Pids = 0 }},
		{"tmp", func(p *SessionProfile) { p.TmpBytes = 0 }},
		{"host-network", func(p *SessionProfile) { p.Network = "host" }},
		{"network-name", func(p *SessionProfile) { p.Network = "bad/network" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := sessionAdmissions(t)
			var profile SessionProfile
			if err := json.Unmarshal(a.Profile.Bytes(), &profile); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&profile)
			payload, err := NewSessionProfile(profile)
			if !errors.Is(err, ErrSession) || payload.Valid() {
				t.Fatalf("invalid settings received a payload: %+v %v", payload.Ref(), err)
			}
			// Go's encoder replaces invalid UTF-8 and cannot encode nonfinite
			// numbers. Only exact encodable settings can reach this reader check.
			if strings.HasSuffix(tc.name, "-utf8") || tc.name == "cpus-nan" || tc.name == "cpus-infinity" {
				return
			}
			raw, err := json.Marshal(profile)
			if err != nil {
				t.Fatal(err)
			}
			a.Profile = payloadForSession(t, "application/vnd.markitect.claude-session-profile+json", raw)
			a.Plan.Profile = a.Profile.Ref()
			if _, _, err := readSessionProfile(a); !errors.Is(err, ErrSession) {
				t.Fatal("admission accepted invalid settings")
			}
		})
	}
}

func TestClaudeSessionProfileFactoryDoesNotAuthorizeWritingInReadOnlyPlan(t *testing.T) {
	a, _ := sessionAdmissions(t)
	// The factory admits explicit Write globally; the plan still refuses it.
	a.Plan.ReadOnly = true
	if _, _, err := readSessionProfile(a); !errors.Is(err, ErrSession) {
		t.Fatal("factory bypassed the read-only plan")
	}
}

func sessionsForTest(t *testing.T, f *fakeRuntime) *Sessions {
	t.Helper()
	s, err := NewSessionsWithRuntime(func(_ context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		expected := container.ContainerSpec{Name: "markitect-session", Image: "sha256:" + strings.Repeat("a", 64), Labels: map[string]string{"markitect.store": "store"}, User: "1000:1000", Workdir: "/work", Mounts: []container.Mount{{Volume: "workspace", Target: "/work", NoCopy: true}, {Volume: "provider-home", Target: "/home/agent"}}, CPUs: 1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20}
		if !reflect.DeepEqual(spec, expected) {
			t.Errorf("unbound runtime spec %+v", spec)
		}
		return f, nil
	}, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "exact-credential" {
			t.Errorf("unbound credential %s", ref)
		}
		return RuntimeCredential{Token: "secret-token"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return s
}
func TestClaudeSingleAdmittedStepAcknowledgedToolsAndObservedExit(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		t.Run(fmt.Sprint(refuse), func(t *testing.T) {
			channel := &fakeChannel{refuse: refuse}
			runtime := peer(channel)
			runtime.useTool = true
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			opened, err := s.Open(ctx, a, channel)
			if err != nil || !opened.Ready || opened.ProviderSession != "" || len(channel.bindings) != 0 {
				t.Fatalf("open %+v %v", opened, err)
			}
			expectedArgv := []string{"/usr/local/bin/claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}`, "--tools", "Read,Write", "--allowedTools", "Read,Write,mcp__markitect__submit", "--permission-mode", "dontAsk", "--model", "admitted-model", "--effort", "high", "--setting-sources", "", "--system-prompt", "Use the exact Host instruction."}
			if len(runtime.start.Env) != 1 || runtime.start.Env[0] != "CLAUDE_CODE_OAUTH_TOKEN=secret-token" || !reflect.DeepEqual(runtime.start.Argv, expectedArgv) || runtime.start.Dir != "/work" {
				t.Fatalf("provider invocation %+v", runtime.start)
			}
			step, err := s.Step(ctx, b, channel)
			if err != nil || !step.Terminal || step.Uncertain || step.Proposed.Valid() || !strings.HasPrefix(string(step.ProviderStep), "claude-single-step-") || step.ProviderStep == "result-event-not-a-turn" {
				t.Fatalf("step %+v %v", step, err)
			}
			if len(channel.calls) != 1 || channel.calls[0].Call != "claude-channel-call-1" || len(runtime.replies) != 1 || len(channel.bindings) != 1 || channel.bindings[0].ProviderSession != "conversation" {
				t.Fatal("tool/session correlation absent")
			}
			if !bytes.Contains(runtime.writes[1], []byte(`Exact input bytes:\n{\"exact\":\"input\"}`)) {
				t.Fatalf("input changed: %s", runtime.writes[1])
			}
			if _, err = s.Step(ctx, b, channel); err == nil {
				t.Fatal("step replay admitted")
			}
			if _, err = s.Open(ctx, a, channel); err == nil {
				t.Fatal("session replay admitted")
			}
			n := len(runtime.writes)
			entry, _ := s.entry(a.Plan.ID)
			inspection := run.InspectionRequest{Session: a.Plan.ID, Step: b.Plan.ID, ProviderSession: "conversation", ProviderStep: step.ProviderStep, SessionBinding: entry.bound, StepBinding: entry.observed}
			observed, err := s.Inspect(ctx, inspection)
			if !errors.Is(err, run.ErrObservationLimit) || !observed.Uncertain || len(runtime.writes) != n {
				t.Fatal("Inspect invented history or wrote")
			}
			stopped, err := s.Stop(ctx, run.StopRequest{Session: a.Plan.ID, ProviderSession: "conversation", Timeout: time.Second}, channel)
			if err != nil || !stopped.Terminated || runtime.stopCalls != 1 {
				t.Fatalf("stop %+v %v", stopped, err)
			}
		})
	}
}

func TestClaudePermitAndObservationFailuresNeverReplay(t *testing.T) {
	for _, mode := range []string{"deny", "identity", "receipt", "error", "observation"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			ctx := context.Background()
			if _, err := s.Open(ctx, a, channel); err != nil {
				t.Fatal(err)
			}
			if mode == "observation" {
				channel.observeFail = true
			} else {
				channel.permitMode = mode
			}
			out, err := s.Step(ctx, b, channel)
			if err == nil || !out.Uncertain || out.Terminal {
				t.Fatalf("failed effect claimed %+v %v", out, err)
			}
			writes := len(runtime.writes)
			if mode != "observation" && writes != 1 {
				t.Fatal("unpermitted step wrote")
			}
			if _, err = s.Step(ctx, b, channel); err == nil || len(runtime.writes) != writes {
				t.Fatal("failed step replayed")
			}
		})
	}
}
func TestClaudeUnacknowledgedToolsAndHaltedRunNeverReply(t *testing.T) {
	for _, halted := range []bool{false, true} {
		t.Run(fmt.Sprint(halted), func(t *testing.T) {
			channel := &fakeChannel{callFail: !halted, halted: halted}
			runtime := peer(channel)
			runtime.useTool = true
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			if _, err := s.Open(context.Background(), a, channel); err != nil {
				t.Fatal(err)
			}
			out, err := s.Step(context.Background(), b, channel)
			if err == nil || !out.Uncertain || len(channel.calls) != 1 || len(runtime.replies) != 0 || len(runtime.writes) != 2 {
				t.Fatalf("unacknowledged answer sent %+v %v", out, err)
			}
		})
	}
}
func TestClaudeObservedSessionAndVersionRequired(t *testing.T) {
	for _, mode := range []string{"no-init", "resume-mismatch", "version", "result-session", "bind-append"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			switch mode {
			case "no-init":
				runtime.noInit = true
			case "resume-mismatch":
				a.Plan.Resume = "requested-conversation"
				b.Session = a.Plan
				b.Plan.SessionBinding = a.Plan.Binding()
			case "version":
				runtime.wrongVersion = true
			case "result-session":
				runtime.resultWrong = true
			case "bind-append":
				channel.bindFail = true
			}
			if _, err := s.Open(context.Background(), a, channel); err != nil {
				t.Fatal(err)
			}
			out, err := s.Step(context.Background(), b, channel)
			if err == nil || !out.Uncertain || out.Terminal {
				t.Fatalf("unproven identity accepted %+v %v", out, err)
			}
			if mode == "resume-mismatch" && !strings.Contains(strings.Join(runtime.start.Argv, " "), "--resume requested-conversation") {
				t.Fatal("requested resume not literal")
			}
			if mode == "no-init" || mode == "version" || mode == "resume-mismatch" {
				if out.ProviderStep != "" {
					t.Fatal("adapter handle created before actual session binding")
				}
			}
		})
	}
}
func TestClaudeMatchingResumeAndErrorResult(t *testing.T) {
	for _, errorResult := range []bool{false, true} {
		t.Run(fmt.Sprint(errorResult), func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.errorResult = errorResult
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			a.Plan.Resume = "conversation"
			b.Session = a.Plan
			b.Plan.SessionBinding = a.Plan.Binding()
			if _, err := s.Open(context.Background(), a, channel); err != nil {
				t.Fatal(err)
			}
			out, err := s.Step(context.Background(), b, channel)
			if err != nil || !out.Terminal || out.Proposed.Valid() {
				t.Fatalf("terminal observation %+v %v", out, err)
			}
		})
	}
}
func TestClaudeContractAndBindingRefusedBeforeDispatch(t *testing.T) {
	for _, mode := range []string{"contract", "legacy-contract", "changed-outcomes", "input-ref", "session-binding", "session-plan"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			s := sessionsForTest(t, runtime)
			a, b := sessionAdmissions(t)
			if _, err := s.Open(context.Background(), a, channel); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "contract":
				b.OutputContract = payloadForSession(t, "application/json", []byte(`{"unsupported":"work"}`))
				b.Plan.OutputContract = b.OutputContract.Ref()
			case "legacy-contract":
				b.OutputContract = payloadForSession(t, "application/vnd.markitect.explore-contract+json", []byte(`{"Format":"markitect.explore-report/v1","Fields":{},"AdditionalFields":false}`))
				b.Plan.OutputContract = b.OutputContract.Ref()
			case "changed-outcomes":
				b.OutputContract = payloadForSession(t, run.StepContractMedia, []byte(`{"Format":"markitect.step-outcome/v1","Phase":"work","Outcomes":["report"]}`))
				b.Plan.OutputContract = b.OutputContract.Ref()
			case "input-ref":
				b.Plan.Input.Size++
			case "session-binding":
				b.Plan.SessionBinding[0]++
			case "session-plan":
				b.Session.Workspace = "foreign"
			}
			if _, err := s.Step(context.Background(), b, channel); err == nil || len(runtime.writes) != 1 {
				t.Fatal("unsupported contract or binding dispatched")
			}
		})
	}
}
func TestClaudeCancellationAndStopUncertainty(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	runtime.cancelTurn = true
	s := sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	if _, err := s.Open(context.Background(), a, channel); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	out, err := s.Step(ctx, b, channel)
	if err == nil || !out.Uncertain || out.Terminal {
		t.Fatalf("cancellation claimed terminal %+v %v", out, err)
	}
	runtime.stopFail = true
	stopped, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, ProviderSession: "conversation", Timeout: time.Second}, channel)
	if err == nil || stopped.Terminated || len(channel.stops) != 1 || channel.stops[0].Terminated {
		t.Fatal("cancellation/stop timeout claimed exit")
	}
}
func TestClaudeCredentialAndProfileHaveNoFallback(t *testing.T) {
	a, _ := sessionAdmissions(t)
	for _, mode := range []string{"empty-token", "resolver-error", "profile", "review-resume", "review-write"} {
		t.Run(mode, func(t *testing.T) {
			channel := &fakeChannel{}
			starts := 0
			candidate := a
			switch mode {
			case "profile":
				candidate.Profile = payloadForSession(t, "application/vnd.markitect.claude-session-profile+json", []byte(`{"version":1,"model":"x","effort":"ambient"}`))
				candidate.Plan.Profile = candidate.Profile.Ref()
			case "review-resume":
				candidate.Plan.Role = "reviewer"
				candidate.Plan.ReadOnly = true
				candidate.Plan.Resume = "old"
			case "review-write":
				candidate.Plan.Role = "reviewer"
				candidate.Plan.ReadOnly = true
			}
			s, err := NewSessionsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) { starts++; return nil, nil }, func(context.Context, string) (RuntimeCredential, error) {
				if mode == "resolver-error" {
					return RuntimeCredential{}, errors.New("do not expose resolver detail")
				}
				return RuntimeCredential{}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Open(context.Background(), candidate, channel); err == nil || starts != 0 {
				t.Fatal("invalid profile/credential fell back")
			}
		})
	}
}
func TestClaudeKnownSecretRedactionKeepsTrueWireIdentity(t *testing.T) {
	channel := &fakeChannel{}
	runtime := peer(channel)
	s := sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	if _, err := s.Open(context.Background(), a, channel); err != nil {
		t.Fatal(err)
	}
	entry, _ := s.entry(a.Plan.ID)
	raw := []byte(`{"secret":"secret-token"}`)
	w := entry.wire(raw)
	if !w.Redacted || bytes.Contains(w.Journal.Bytes(), []byte("secret-token")) || w.Identity.ContentID != sha256.Sum256(raw) || w.Identity.Size != int64(len(raw)) || w.ObservationLimit == "" {
		t.Fatal("redaction lost wire identity")
	}
	runtime.writeFail = true
	out, err := s.Step(context.Background(), b, channel)
	if err == nil || strings.Contains(err.Error(), "secret-token") || !out.Uncertain {
		t.Fatal("error disclosed credential or lost uncertainty")
	}
}

func TestClaudeStopRetainsPhysicalHandleAfterUnacknowledgedBind(t *testing.T) {
	channel := &fakeChannel{bindFail: true}
	runtime := peer(channel)
	s := sessionsForTest(t, runtime)
	a, b := sessionAdmissions(t)
	if _, err := s.Open(context.Background(), a, channel); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Step(context.Background(), b, channel); err == nil {
		t.Fatal("unacknowledged binding accepted")
	}
	// RUN cannot name an unacknowledged provider ID. The admitted Host Session
	// still owns the physical container and must remain stoppable.
	out, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, Timeout: time.Second}, channel)
	if err != nil || !out.Terminated || runtime.stopCalls != 1 {
		t.Fatalf("lost physical stop handle %+v %v", out, err)
	}
}

func TestClaudeFreshReviewerMountAndReadOnlyTools(t *testing.T) {
	a, _ := sessionAdmissions(t)
	a.Plan.Role = "reviewer"
	a.Plan.ReadOnly = true
	var profile SessionProfile
	if err := json.Unmarshal(a.Profile.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	profile.BuiltinTools = []string{"Read"}
	raw, _ := json.Marshal(profile)
	a.Profile = payloadForSession(t, "application/vnd.markitect.claude-session-profile+json", raw)
	a.Plan.Profile = a.Profile.Ref()
	parsed, tools, err := readSessionProfile(a)
	if err != nil {
		t.Fatal(err)
	}
	spec := sessionSpec(a.Plan, parsed)
	argv := sessionArgv(a.Plan, parsed, tools)
	if !spec.Mounts[0].ReadOnly || !spec.Mounts[0].NoCopy || spec.Mounts[1].ReadOnly || spec.Mounts[1].NoCopy || strings.Contains(strings.Join(argv, " "), "--resume") || strings.Contains(strings.Join(argv, " "), "Write") {
		t.Fatalf("reviewer inherited writing or conversation state: %+v %v", spec, argv)
	}
}

func TestClaudeNativeSessionAndStopReasonCannotJournalKnownToken(t *testing.T) {
	t.Run("session", func(t *testing.T) {
		channel := &fakeChannel{}
		runtime := peer(channel)
		runtime.noInit = true
		runtime.cancelTurn = true
		s := sessionsForTest(t, runtime)
		a, b := sessionAdmissions(t)
		if _, err := s.Open(context.Background(), a, channel); err != nil {
			t.Fatal(err)
		}
		runtime.emit(`{"type":"system","subtype":"init","session_id":"secret-token","uuid":"init-event","claude_code_version":"2.1.301"}`)
		out, err := s.Step(context.Background(), b, channel)
		if err == nil || !out.Uncertain || len(channel.bindings) != 0 || bytes.Contains(out.Wire.Journal.Bytes(), []byte("secret-token")) {
			t.Fatal("known credential escaped wire redaction through native session field")
		}
	})
	t.Run("stop", func(t *testing.T) {
		channel := &fakeChannel{}
		runtime := peer(channel)
		runtime.exitSignal = "secret-token"
		s := sessionsForTest(t, runtime)
		a, _ := sessionAdmissions(t)
		if _, err := s.Open(context.Background(), a, channel); err != nil {
			t.Fatal(err)
		}
		out, err := s.Stop(context.Background(), run.StopRequest{Session: a.Plan.ID, Timeout: time.Second}, channel)
		if err != nil || !out.Terminated || strings.Contains(out.Reason, "secret-token") || len(channel.stops) != 1 || strings.Contains(channel.stops[0].Reason, "secret-token") {
			t.Fatal("known credential entered stop journal reason")
		}
	})

}

func TestClaudeStepPhasesUseHostToolsAndIgnoreFinalText(t *testing.T) {
	for _, phase := range []run.Phase{run.PhaseExplore, run.PhaseWork, run.PhaseIntegrate, run.PhaseReply, run.PhaseReviewerExpectations, run.PhaseReviewerCandidate} {
		t.Run(string(phase), func(t *testing.T) {
			channel := &fakeChannel{}
			runtime := peer(channel)
			runtime.finalText = `{"Version":1,"Summary":"this final report is not an outcome","secret":"secret-token"}`
			a, b := sessionAdmissions(t)
			catalog, err := run.ToolCatalog(phase)
			if err != nil {
				t.Fatal(err)
			}
			a.Tools = catalog
			a.Plan.Tools = catalog.Ref()
			b.Session = a.Plan
			b.Plan.SessionBinding = a.Plan.Binding()
			b.OutputContract, err = run.NewStepContract(phase)
			if err != nil {
				t.Fatal(err)
			}
			b.Plan.OutputContract = b.OutputContract.Ref()
			s := sessionsForTest(t, runtime)
			if _, err = s.Open(context.Background(), a, channel); err != nil {
				t.Fatal(err)
			}
			var expected struct {
				Tools []frame.ToolSpec `json:"tools"`
			}
			if err = json.Unmarshal(catalog.Bytes(), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runtime.start.Tools, expected.Tools) {
				t.Fatal("admitted catalogue changed")
			}
			out, err := s.Step(context.Background(), b, channel)
			if err != nil || !out.Terminal || out.Proposed.Valid() || len(channel.calls) != 0 || !out.Wire.Redacted || bytes.Contains(out.Wire.Journal.Bytes(), []byte("secret-token")) {
				t.Fatalf("final text became an outcome or leaked: %+v %v", out, err)
			}
			for _, observed := range channel.steps {
				if observed.Proposed.Valid() {
					t.Fatal("journal proposal from final text")
				}
			}
		})
	}
}
func TestClaudeKnownSecretToolCallRefusedBeforeHost(t *testing.T) {
	for _, arguments := range []json.RawMessage{json.RawMessage(`{"summary":"secret-token"}`), json.RawMessage(`{"summary":"\u0073ecret-token"}`)} {
		channel := &fakeChannel{}
		runtime := peer(channel)
		runtime.useTool = true
		runtime.toolArguments = arguments
		s := sessionsForTest(t, runtime)
		a, b := sessionAdmissions(t)
		if _, err := s.Open(context.Background(), a, channel); err != nil {
			t.Fatal(err)
		}
		out, err := s.Step(context.Background(), b, channel)
		if err == nil || !out.Uncertain || len(channel.calls) != 0 || len(runtime.replies) != 0 || strings.Contains(err.Error(), "secret-token") {
			t.Fatal("known secret tool call escaped refusal")
		}
	}
}
func TestClaudeKnownSecretHostAnswerRefusedBeforeReply(t *testing.T) {
	for _, answer := range []json.RawMessage{json.RawMessage(`{"echo":"secret-token"}`), json.RawMessage(`{"echo":"\u0073ecret-token"}`)} {
		channel := &fakeChannel{secretAnswer: answer}
		runtime := peer(channel)
		runtime.useTool = true
		s := sessionsForTest(t, runtime)
		a, b := sessionAdmissions(t)
		if _, err := s.Open(context.Background(), a, channel); err != nil {
			t.Fatal(err)
		}
		out, err := s.Step(context.Background(), b, channel)
		if err == nil || !out.Uncertain || len(channel.calls) != 1 || len(runtime.replies) != 0 || strings.Contains(err.Error(), "secret-token") {
			t.Fatal("known secret Host answer reached provider")
		}
	}
}
