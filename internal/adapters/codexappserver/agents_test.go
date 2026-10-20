package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const testKey = "secret-key-for-runtime"

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
			// Lines emitted before the stream closed are still delivered;
			// select alone would pick between them and EOF at random.
			select {
			case f.pending = <-f.chunks:
			default:
				return 0, io.EOF
			}
		}
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}
func (f *fakeStream) close() { f.once.Do(func() { close(f.done) }) }

// fakeRuntime plays the provider side of one turn from a script of flags.
type fakeRuntime struct {
	stream *fakeStream
	tools  chan frame.ToolCall

	mu      sync.Mutex
	spec    container.ContainerSpec
	start   frame.Start
	writes  []string
	replies []frame.ToolReply
	stops   int
	closed  bool

	closeErr       error
	toolArgs       string
	useTool        bool
	toolThread     string
	toolName       string
	hang           bool
	usage          bool
	finalText      string
	terminalStatus string
	resumeMismatch bool
	errorText      string
	stopFail       bool
}

func newRuntime() *fakeRuntime {
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 64), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16), toolName: "submit", toolArgs: `{"summary":"done"}`}
}
func (f *fakeRuntime) emit(raw string) { f.stream.chunks <- []byte(raw + "\n") }
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.start = s
	return frame.Started{PID: 123}, nil
}
func (f *fakeRuntime) Stdout() io.Reader                { return f.stream }
func (f *fakeRuntime) ToolCalls() <-chan frame.ToolCall { return f.tools }
func (f *fakeRuntime) WriteInput(data []byte) error {
	f.mu.Lock()
	f.writes = append(f.writes, string(data))
	f.mu.Unlock()
	var m struct {
		Method string
	}
	_ = json.Unmarshal(data, &m)
	switch m.Method {
	case "initialize":
		f.emit(`{"id":1,"result":` + codecInitialization + `}`)
	case "thread/start", "thread/resume":
		thread := codecThread
		if f.resumeMismatch {
			thread = strings.ReplaceAll(thread, string(codecThreadID), "wrong-thread")
		}
		f.emit(`{"id":2,"result":{"thread":` + thread + `,"approvalPolicy":"never","approvalsReviewer":"user","cwd":"/work","model":"admitted-model","modelProvider":"markitect","sandbox":{"type":"dangerFullAccess"}}}`)
	case "turn/start":
		f.emit(`{"id":3,"result":{"turn":{"id":"` + string(codecTurnID) + `","status":"inProgress","items":[],"error":null}}}`)
		switch {
		case f.errorText != "":
			f.emit(`{"method":"error","params":{"error":{"message":"` + f.errorText + `"},"willRetry":false}}`)
		case f.hang:
		case f.useTool:
			thread := string(codecThreadID)
			if f.toolThread != "" {
				thread = f.toolThread
			}
			f.tools <- frame.ToolCall{ID: 1, Name: f.toolName, Arguments: json.RawMessage(f.toolArgs), Meta: json.RawMessage(`{"threadId":"` + thread + `","callId":"call-1"}`)}
		default:
			f.finish()
		}
	}
	return nil
}
func (f *fakeRuntime) finish() {
	if f.usage {
		f.emit(`{"method":"thread/tokenUsage/updated","params":{"threadId":"` + string(codecThreadID) + `","turnId":"` + string(codecTurnID) + `","tokenUsage":{"total":{"totalTokens":0,"inputTokens":0,"cachedInputTokens":0,"outputTokens":0,"reasoningOutputTokens":0},"last":{"totalTokens":150,"inputTokens":100,"cachedInputTokens":40,"outputTokens":50,"reasoningOutputTokens":10,"cacheWriteInputTokens":5}}}}`)
	}
	item, _ := json.Marshal(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": codecThreadID, "turnId": codecTurnID, "item": map[string]any{"id": "answer", "type": "agentMessage", "text": f.finalText}}})
	f.emit(string(item))
	status := f.terminalStatus
	if status == "" {
		status = "completed"
	}
	f.emit(`{"method":"turn/completed","params":{"threadId":"` + string(codecThreadID) + `","turn":{"id":"` + string(codecTurnID) + `","status":"` + status + `","items":[],"error":null}}}`)
}
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.mu.Lock()
	f.replies = append(f.replies, r)
	f.mu.Unlock()
	f.finish()
	return nil
}
func (f *fakeRuntime) Stop(context.Context, int64) (frame.Exited, error) {
	f.mu.Lock()
	f.stops++
	f.mu.Unlock()
	if f.stopFail {
		return frame.Exited{}, io.ErrClosedPipe
	}
	return frame.Exited{Code: -1, Signal: "terminated"}, nil
}
func (f *fakeRuntime) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	f.stream.close()
	return f.closeErr
}

func testProfile() Profile {
	return Profile{ImageID: "sha256:" + strings.Repeat("a", 64), CredentialRef: "exact-credential", MaxWireBytes: 1 << 20,
		Session: SessionProfile{Version: 1, Model: "admitted-model", Effort: "high", Store: "store", CPUs: 1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20}}
}

func testTurn() run.Turn {
	return run.Turn{ID: "turn-1", Manager: "lending", Kind: run.ExecuteTurn, Profile: "codex", Workspace: "workspace", Home: "provider-home", Prompt: "do the work",
		Tools: []run.ToolSpec{{Name: "submit", Description: "Submit the outcome.", InputSchema: json.RawMessage(`{"type":"object"}`)}}, Timeout: 5 * time.Second}
}

func agentsFor(t *testing.T, rt *fakeRuntime) *Agents {
	t.Helper()
	a, err := NewAgentsWithRuntime(func(_ context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		rt.mu.Lock()
		rt.spec = spec
		rt.mu.Unlock()
		return rt, nil
	}, nil, map[string]Profile{"codex": testProfile()}, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "exact-credential" {
			t.Errorf("unbound credential %s", ref)
		}
		return RuntimeCredential{Mode: "api-key", APIKey: testKey}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	a.stopGrace, a.stopTimeout = time.Second, time.Second
	t.Cleanup(func() { _ = rt.Close() })
	return a
}

func okHandler(calls *[]string) run.ToolHandler {
	return func(_ context.Context, name string, args json.RawMessage) run.ToolAnswer {
		*calls = append(*calls, name+" "+string(args))
		return run.ToolAnswer{Text: "accepted"}
	}
}

func TestRunExecuteTurn(t *testing.T) {
	rt := newRuntime()
	rt.useTool, rt.usage, rt.finalText = true, true, "all done "+testKey
	path := filepath.Join(t.TempDir(), "turn.jsonl")
	turn := testTurn()
	turn.Transcript = path
	var calls []string
	res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(&calls))
	if err != nil {
		t.Fatal(err)
	}
	if res.Conversation != string(codecThreadID) || res.Final != "all done [REDACTED]" || !res.Stopped || res.Reason != "provider turn completed" || res.Started.IsZero() || res.Ended.Before(res.Started) {
		t.Fatalf("result %+v", res)
	}
	if want := (run.Usage{Model: "admitted-model", InputTokens: 60, OutputTokens: 50, CacheReadTokens: 40, CacheCreationTokens: 5, ProviderTurns: 1}); res.Usage != want {
		t.Fatalf("usage %+v", res.Usage)
	}
	if len(calls) != 1 || calls[0] != `submit {"summary":"done"}` {
		t.Fatalf("handler calls %v", calls)
	}
	var result struct {
		Content []struct{ Type, Text string }
		IsError bool
	}
	if len(rt.replies) != 1 || json.Unmarshal(rt.replies[0].Result, &result) != nil || rt.replies[0].ID != 1 || len(result.Content) != 1 || result.Content[0].Text != "accepted" || result.IsError {
		t.Fatalf("tool reply %+v", rt.replies)
	}
	spec := rt.spec
	if spec.Name != "markitect-turn-1" || spec.Image != testProfile().ImageID || len(spec.Mounts) != 2 || spec.Mounts[0] != (container.Mount{Volume: "workspace", Target: "/work", NoCopy: true}) || spec.Mounts[1] != (container.Mount{Volume: "provider-home", Target: "/home/agent"}) || spec.Labels["markitect.store"] != "store" {
		t.Fatalf("container spec %+v", spec)
	}
	if len(rt.start.Tools) != 1 || rt.start.Tools[0].Name != "submit" || rt.start.Dir != "/work" || strings.Join(rt.start.Env, " ") != "CODEX_HOME=/home/agent/.codex OPENAI_API_KEY="+testKey {
		t.Fatalf("provider start %+v", rt.start)
	}
	if !strings.Contains(rt.writes[2], `"method":"thread/start"`) || strings.Contains(rt.writes[2], "threadId") || !strings.Contains(rt.writes[2], `"sandbox":"danger-full-access"`) || !strings.Contains(rt.writes[3], "do the work") || !strings.Contains(rt.writes[3], `"effort":"high"`) {
		t.Fatalf("provider writes %q", rt.writes)
	}
	if rt.stops != 1 || !rt.closed {
		t.Fatal("container not stopped")
	}
	data, err := os.ReadFile(path)
	if err != nil || bytes.Contains(data, []byte(testKey)) || !bytes.Contains(data, []byte("[REDACTED]")) {
		t.Fatalf("transcript leaked or missing: %v", err)
	}
	directions := map[string]bool{}
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var l struct{ At, Direction, Data string }
		if json.Unmarshal(line, &l) != nil || l.At == "" || l.Data == "" {
			t.Fatalf("transcript line %q", line)
		}
		directions[l.Direction] = true
	}
	for _, d := range []string{"send", "receive", "tool-call", "tool-reply", "exit"} {
		if !directions[d] {
			t.Fatalf("transcript lacks %s lines", d)
		}
	}
}

func TestRunReportsNoUsageWhenProtocolReportsNone(t *testing.T) {
	rt := newRuntime()
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err != nil || res.Usage != (run.Usage{}) || res.Final != "" || !res.Stopped {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestRunResumesOnlyTheNamedThread(t *testing.T) {
	turn := testTurn()
	turn.Resume = string(codecThreadID)
	rt := newRuntime()
	res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
	if err != nil || res.Conversation != turn.Resume || !strings.Contains(rt.writes[2], `"method":"thread/resume"`) || !strings.Contains(rt.writes[2], `"threadId":"`+turn.Resume+`"`) {
		t.Fatalf("%+v %v %q", res, err, rt.writes)
	}
	rt = newRuntime()
	rt.resumeMismatch = true
	res, err = agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
	if !errors.Is(err, ErrProtocol) || res.Conversation != "" || !res.Stopped || rt.stops != 1 {
		t.Fatalf("resumed another thread: %+v %v", res, err)
	}
}

func TestRunVerifyTurnMountsReadOnly(t *testing.T) {
	turn := testTurn()
	turn.Kind, turn.ReadOnly = run.VerifyTurn, true
	rt := newRuntime()
	if _, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string))); err != nil {
		t.Fatal(err)
	}
	if m := rt.spec.Mounts[0]; m.Target != "/work" || !m.ReadOnly || !m.NoCopy || rt.spec.Mounts[1].ReadOnly {
		t.Fatalf("workspace must be read-only and the provider home writable: %+v", rt.spec.Mounts)
	}
}

func TestRunTurnMountsOverlayTheWorkspaceVolumeInOrder(t *testing.T) {
	root := []run.PathMount{{Path: "lending"}, {Path: "library"}}
	child := []run.PathMount{{Path: "lending", Writable: true}, {Path: "lending/reminders"}}
	for name, tc := range map[string]struct {
		readOnly bool
		mounts   []run.PathMount
		want     []container.Mount
	}{
		"root style: writable base, read-only overlays": {false, root, []container.Mount{
			{Volume: "workspace", Target: "/work", NoCopy: true},
			{Volume: "workspace", Target: "/work/lending", Subpath: "lending", ReadOnly: true, NoCopy: true},
			{Volume: "workspace", Target: "/work/library", Subpath: "library", ReadOnly: true, NoCopy: true},
			{Volume: "provider-home", Target: "/home/agent"}}},
		"child style: read-only base, writable overlays": {true, child, []container.Mount{
			{Volume: "workspace", Target: "/work", ReadOnly: true, NoCopy: true},
			{Volume: "workspace", Target: "/work/lending", Subpath: "lending", NoCopy: true},
			{Volume: "workspace", Target: "/work/lending/reminders", Subpath: "lending/reminders", ReadOnly: true, NoCopy: true},
			{Volume: "provider-home", Target: "/home/agent"}}},
	} {
		t.Run(name, func(t *testing.T) {
			turn := testTurn()
			turn.ReadOnly, turn.Mounts = tc.readOnly, tc.mounts
			rt := newRuntime()
			if _, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string))); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rt.spec.Mounts, tc.want) {
				t.Fatalf("mounts %+v", rt.spec.Mounts)
			}
		})
	}
}

func TestRunVerifyTurnWithReadOnlyMountsStarts(t *testing.T) {
	turn := testTurn()
	turn.Kind, turn.ReadOnly, turn.Mounts = run.VerifyTurn, true, []run.PathMount{{Path: "lending"}}
	rt := newRuntime()
	if _, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string))); err != nil || len(rt.spec.Mounts) != 3 {
		t.Fatalf("%v %+v", err, rt.spec.Mounts)
	}
}

func TestRunRefusalIsAToolErrorNotATurnFailure(t *testing.T) {
	rt := newRuntime()
	rt.useTool = true
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), func(context.Context, string, json.RawMessage) run.ToolAnswer {
		return run.ToolAnswer{Text: "Orchestrator refused this call", Refused: true}
	})
	var result struct {
		Content []struct{ Text string }
		IsError bool
	}
	if err != nil || json.Unmarshal(rt.replies[0].Result, &result) != nil || !result.IsError || result.Content[0].Text != "Orchestrator refused this call" || res.Reason != "provider turn completed" {
		t.Fatalf("%+v %v %+v", res, err, rt.replies)
	}
}

func TestRunRefusesForeignOrUnofferedToolCalls(t *testing.T) {
	for name, set := range map[string]func(*fakeRuntime){
		"foreign thread": func(f *fakeRuntime) { f.toolThread = "foreign-thread" },
		"unoffered tool": func(f *fakeRuntime) { f.toolName = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.useTool = true
			set(rt)
			called := false
			res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), func(context.Context, string, json.RawMessage) run.ToolAnswer {
				called = true
				return run.ToolAnswer{}
			})
			if !errors.Is(err, ErrProtocol) || called || len(rt.replies) != 0 || !res.Stopped || rt.stops != 1 {
				t.Fatalf("%+v %v called=%v", res, err, called)
			}
		})
	}
}

func TestRunProviderFailureKeepsConversation(t *testing.T) {
	for _, status := range []string{"failed", "interrupted"} {
		rt := newRuntime()
		rt.terminalStatus = status
		res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
		if !errors.Is(err, ErrSession) || res.Conversation != string(codecThreadID) || res.Reason != "provider turn "+status || !res.Stopped {
			t.Fatalf("%s: %+v %v", status, res, err)
		}
	}
}

func TestRunProviderErrorTextIsRedacted(t *testing.T) {
	rt := newRuntime()
	rt.errorText = "bad key " + testKey
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err == nil || strings.Contains(err.Error(), testKey) || !strings.Contains(err.Error(), "[REDACTED]") || !res.Stopped {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestRunTimeoutAndCancelStopTheContainer(t *testing.T) {
	rt := newRuntime()
	rt.hang = true
	turn := testTurn()
	turn.Timeout = 100 * time.Millisecond
	res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
	if !errors.Is(err, context.DeadlineExceeded) || res.Reason != "timeout" || !res.Stopped || rt.stops != 1 || !rt.closed || res.Conversation != string(codecThreadID) {
		t.Fatalf("timeout: %+v %v", res, err)
	}
	rt = newRuntime()
	rt.hang = true
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	res, err = agentsFor(t, rt).Run(ctx, testTurn(), okHandler(new([]string)))
	if !errors.Is(err, context.Canceled) || res.Reason != "cancelled" || !res.Stopped || rt.stops != 1 || !rt.closed {
		t.Fatalf("cancel: %+v %v", res, err)
	}
}

func TestRunStoppedOnlyWhenExitWasObserved(t *testing.T) {
	rt := newRuntime()
	rt.stopFail = true
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err == nil || res.Stopped || !rt.closed {
		t.Fatalf("unobserved exit reported as stopped: %+v %v", res, err)
	}
}

func TestRunStoppedOnlyWhenTheContainerEndWasObserved(t *testing.T) {
	rt := newRuntime()
	rt.closeErr = errors.Join(container.ErrForcedClose, errors.New("engine "+testKey))
	res, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err == nil || !strings.Contains(err.Error(), "removed by force") || res.Stopped || !strings.Contains(res.Reason, "container end was not") || !rt.closed {
		t.Fatalf("forced close reported as stopped: %+v %v", res, err)
	}
	if strings.Contains(err.Error(), testKey) || strings.Contains(res.Reason, testKey) {
		t.Fatal("secret leaked")
	}
}

func TestRunRedactsKnownSecretsFromToolArgumentsBeforeTheHandler(t *testing.T) {
	rt := newRuntime()
	rt.useTool, rt.toolArgs = true, `{"summary":"leak `+testKey+`"}`
	var calls []string
	if _, err := agentsFor(t, rt).Run(context.Background(), testTurn(), okHandler(&calls)); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || strings.Contains(calls[0], testKey) || calls[0] != `submit {"summary":"leak [REDACTED]"}` {
		t.Fatalf("handler saw %q", calls)
	}
}

func TestRunStartFailureBeforeAndAfterContainerCreation(t *testing.T) {
	profiles := map[string]Profile{"codex": testProfile()}
	starts := 0
	failing := func(resolver CredentialResolver) *Agents {
		a, err := NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) {
			starts++
			return nil, errors.New("container creation result unknown")
		}, nil, profiles, resolver)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	good := func(context.Context, string) (RuntimeCredential, error) {
		return RuntimeCredential{Mode: "api-key", APIKey: testKey}, nil
	}
	res, err := failing(good).Run(context.Background(), testTurn(), okHandler(new([]string)))
	if err == nil || res.Stopped || starts != 1 {
		t.Fatalf("a failed container start is uncertain: %+v %v", res, err)
	}
	for name, resolver := range map[string]CredentialResolver{
		"resolver error": func(context.Context, string) (RuntimeCredential, error) {
			return RuntimeCredential{}, errors.New("failed")
		},
		"invalid": func(context.Context, string) (RuntimeCredential, error) {
			return RuntimeCredential{Mode: "api-key"}, nil
		},
	} {
		starts = 0
		res, err = failing(resolver).Run(context.Background(), testTurn(), okHandler(new([]string)))
		if !errors.Is(err, ErrSession) || !res.Stopped || starts != 0 {
			t.Fatalf("%s: %+v %v starts=%d", name, res, err, starts)
		}
	}
}

func TestRunRefusesInvalidTurnsBeforeStartingAnything(t *testing.T) {
	for name, change := range map[string]func(*run.Turn){
		"unknown profile":     func(t *run.Turn) { t.Profile = "other" },
		"bad id":              func(t *run.Turn) { t.ID = "a/b" },
		"bad workspace":       func(t *run.Turn) { t.Workspace = "" },
		"same volume":         func(t *run.Turn) { t.Home = t.Workspace },
		"negative timeout":    func(t *run.Turn) { t.Timeout = -1 },
		"unknown kind":        func(t *run.Turn) { t.Kind = "other" },
		"writable verify":     func(t *run.Turn) { t.Kind = run.VerifyTurn },
		"resuming verify":     func(t *run.Turn) { t.Kind, t.ReadOnly, t.Resume = run.VerifyTurn, true, "thread" },
		"duplicate tool":      func(t *run.Turn) { t.Tools = append(t.Tools, t.Tools[0]) },
		"tool without text":   func(t *run.Turn) { t.Tools[0].Description = " " },
		"tool schema":         func(t *run.Turn) { t.Tools[0].InputSchema = json.RawMessage(`[]`) },
		"relative transcript": func(t *run.Turn) { t.Transcript = "turn.jsonl" },
		"mount path escapes":  func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "../lending", Writable: true}} },
		"mount path absolute": func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "/lending"}} },
		"mount path unclean":  func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "lending/./reminders"}} },
		"mount path empty":    func(t *run.Turn) { t.Mounts = []run.PathMount{{}} },
		"mount option inject": func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "a,readonly=false"}} },
		"mount path twice":    func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "a"}, {Path: "a", Writable: true}} },
		"verify writable mount": func(t *run.Turn) {
			t.Kind, t.ReadOnly, t.Mounts = run.VerifyTurn, true, []run.PathMount{{Path: "lending", Writable: true}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			turn := testTurn()
			turn.Tools = append([]run.ToolSpec(nil), turn.Tools...)
			change(&turn)
			rt := newRuntime()
			res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
			if !errors.Is(err, ErrSession) || !res.Stopped || rt.spec.Name != "" {
				t.Fatalf("%+v %v", res, err)
			}
		})
	}
	if _, err := agentsFor(t, newRuntime()).Run(context.Background(), testTurn(), nil); !errors.Is(err, ErrSession) {
		t.Fatal("accepted a nil handler")
	}
}

func TestNewAgentsRefusesInvalidInputs(t *testing.T) {
	resolver := func(context.Context, string) (RuntimeCredential, error) { return RuntimeCredential{}, nil }
	start := func(context.Context, container.ContainerSpec) (RuntimeSession, error) { return nil, nil }
	bad := func(change func(*Profile)) map[string]Profile {
		p := testProfile()
		change(&p)
		return map[string]Profile{"codex": p}
	}
	for name, profiles := range map[string]map[string]Profile{
		"image":      bad(func(p *Profile) { p.ImageID = "latest" }),
		"credential": bad(func(p *Profile) { p.CredentialRef = "" }),
		"wire":       bad(func(p *Profile) { p.MaxWireBytes = maxProtocolMessage + 1 }),
		"settings":   bad(func(p *Profile) { p.Session.Effort = "max" }),
		"name":       {"": testProfile()},
	} {
		if _, err := NewAgentsWithRuntime(start, nil, profiles, resolver); !errors.Is(err, ErrSession) {
			t.Fatalf("%s accepted", name)
		}
	}
	if _, err := NewAgentsWithRuntime(nil, nil, nil, resolver); err == nil {
		t.Fatal("accepted missing starter")
	}
	if _, err := NewAgentsWithRuntime(start, nil, nil, nil); err == nil {
		t.Fatal("accepted missing resolver")
	}
	if _, err := NewAgents(nil, nil, resolver); err == nil {
		t.Fatal("accepted missing engine")
	}
}

func TestRunsMayOverlap(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rt := newRuntime()
			rt.useTool = true
			turn := testTurn()
			turn.ID = "turn-" + string(rune('a'+i))
			res, err := agentsFor(t, rt).Run(context.Background(), turn, okHandler(new([]string)))
			if err != nil || !res.Stopped {
				t.Errorf("%+v %v", res, err)
			}
		}()
	}
	wg.Wait()
}
