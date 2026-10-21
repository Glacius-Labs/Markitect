package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

const testToken = `sk-ant-oat01-te"st\secret`

// fakeStream is a blocking provider stdout.
type fakeStream struct {
	chunks chan []byte
	done   chan struct{}
	once   sync.Once
	rest   []byte
}

func (f *fakeStream) Read(p []byte) (int, error) {
	if len(f.rest) == 0 {
		select {
		case f.rest = <-f.chunks:
		case <-f.done:
			// Lines emitted before the stream closed are still delivered;
			// select alone would pick between them and EOF at random.
			select {
			case f.rest = <-f.chunks:
			default:
				return 0, io.EOF
			}
		}
	}
	n := copy(p, f.rest)
	f.rest = f.rest[n:]
	return n, nil
}
func (f *fakeStream) close() { f.once.Do(func() { close(f.done) }) }

// fakeRuntime plays one provider: it acknowledges initialize, answers the
// prompt with init, the scripted Orchestrator tool calls and a result.
type fakeRuntime struct {
	mu      sync.Mutex
	stream  *fakeStream
	tools   chan frame.ToolCall
	spec    container.ContainerSpec
	start   frame.Start
	writes  [][]byte
	replies []frame.ToolReply

	session   string // init session_id
	calls     []frame.ToolCall
	extra     []string      // provider lines sent after init
	result    string        // override of the result line
	silent    bool          // never sends a result
	endStream bool          // closes stdout after the prompt instead of a result
	callDelay time.Duration // waits before the scripted tool calls
	startErr  error
	stopErr   error
	closeErr  error
	exit      frame.Exited
	stopCalls int
	closed    bool
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 64), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16),
		session: "conversation-1", exit: frame.Exited{Code: -1, Signal: "terminated"}}
}

func (f *fakeRuntime) emit(line string) { f.stream.chunks <- []byte(line + "\n") }
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.start = s
	return frame.Started{PID: 7}, f.startErr
}
func (f *fakeRuntime) Stdout() io.Reader                { return f.stream }
func (f *fakeRuntime) ToolCalls() <-chan frame.ToolCall { return f.tools }
func (f *fakeRuntime) WriteInput(data []byte) error {
	f.mu.Lock()
	f.writes = append(f.writes, append([]byte(nil), data...))
	f.mu.Unlock()
	var m struct{ Type string }
	_ = json.Unmarshal(data, &m)
	switch m.Type {
	case "control_request":
		f.emit(`{"type":"control_response","response":{"subtype":"success","request_id":"markitect-initialize","response":{"commands":[],"agents":[],"models":[],"output_style":"default","available_output_styles":[],"account":{},"claude_code_version":"2.1.301"}}}`)
	case "user":
		f.emit(`{"type":"system","subtype":"init","session_id":"` + f.session + `","uuid":"init-1","claude_code_version":"2.1.301","model":"claude-test-model"}`)
		for _, line := range f.extra {
			f.emit(line)
		}
		switch {
		case f.endStream:
			f.stream.close()
		case len(f.calls) > 0 && f.callDelay > 0:
			go func() {
				time.Sleep(f.callDelay)
				for _, c := range f.calls {
					f.tools <- c
				}
			}()
		case len(f.calls) > 0:
			for _, c := range f.calls {
				f.tools <- c
			}
		case !f.silent:
			f.finish()
		}
	}
	return nil
}
func (f *fakeRuntime) finish() {
	if f.result != "" {
		f.emit(f.result)
		return
	}
	f.emit(`{"type":"result","subtype":"success","session_id":"` + f.session + `","uuid":"res-1","is_error":false,"result":"all done","num_turns":3,"total_cost_usd":0.25,` +
		`"usage":{"input_tokens":11,"output_tokens":22,"cache_read_input_tokens":33,"cache_creation_input_tokens":44}}`)
}
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.mu.Lock()
	f.replies = append(f.replies, r)
	n := len(f.replies)
	f.mu.Unlock()
	if n == len(f.calls) {
		f.finish()
	}
	return nil
}
func (f *fakeRuntime) Stop(context.Context, int64) (frame.Exited, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	if f.stopErr != nil {
		return frame.Exited{}, f.stopErr
	}
	return f.exit, nil
}
func (f *fakeRuntime) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	f.stream.close()
	return f.closeErr
}

type harness struct {
	t       *testing.T
	runtime *fakeRuntime
	starts  int
	agents  *Agents
}

func newHarness(t *testing.T, rt *fakeRuntime, profile Profile) *harness {
	t.Helper()
	h := &harness{t: t, runtime: rt}
	agents, err := NewAgentsWithRuntime(func(_ context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		h.starts++
		rt.spec = spec
		return rt, nil
	}, map[string]Profile{"claude": profile}, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "ref" {
			return RuntimeCredential{}, errors.New("unknown credential")
		}
		return RuntimeCredential{Token: testToken}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	h.agents = agents
	return h
}

func testTurn() run.Turn {
	return run.Turn{ID: "turn-1", Manager: "lending", Kind: run.ExecuteTurn, Profile: "claude", Workspace: "ws-volume", Home: "home-volume", Prompt: "Do the work.",
		Tools: []run.ToolSpec{{Name: "report", Description: "Report to the Orchestrator", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
}

func argAfter(argv []string, flag string) string {
	for i, a := range argv {
		if a == flag && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return "<absent>"
}

func echoHandler(calls *[]string) run.ToolHandler {
	return func(_ context.Context, name string, args json.RawMessage) run.ToolAnswer {
		*calls = append(*calls, name+" "+string(args))
		return run.ToolAnswer{Text: "orchestrator says ok"}
	}
}

func TestExecuteTurnSpecArgvEnvAndResult(t *testing.T) {
	rt := newFakeRuntime()
	rt.calls = []frame.ToolCall{{ID: 1, Name: "report", Arguments: json.RawMessage(`{"x":1}`)}}
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	var seen []string
	res, err := h.agents.Run(context.Background(), turn, echoHandler(&seen))
	if err != nil {
		t.Fatal(err)
	}
	wantSpec := container.ContainerSpec{Name: "markitect-turn-1", Image: fixtureImage, Labels: map[string]string{"markitect.store": "store"}, User: "1000:1000", Workdir: "/work",
		Mounts: []container.Mount{{Volume: "ws-volume", Target: "/work", NoCopy: true}, {Volume: "home-volume", Target: "/home/agent"}},
		CPUs:   1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20}
	if !reflect.DeepEqual(rt.spec, wantSpec) {
		t.Fatalf("spec %+v", rt.spec)
	}
	argv := rt.start.Argv
	for flag, want := range map[string]string{"--tools": "Read,Write,Edit", "--allowedTools": "Read,Write,Edit,mcp__markitect__report", "--permission-mode": "dontAsk",
		"--model": "m", "--effort": "high", "--setting-sources": "", "--system-prompt": "Use the Runtime instruction.", "--resume": "<absent>",
		"--input-format": "stream-json", "--output-format": "stream-json", "--strict-mcp-config": argAfter(argv, "--strict-mcp-config")} {
		if got := argAfter(argv, flag); got != want {
			t.Errorf("%s = %q, want %q", flag, got, want)
		}
	}
	if !strings.Contains(argAfter(argv, "--mcp-config"), `"args":["mcp"]`) || argv[0] != "/usr/local/bin/claude" {
		t.Errorf("argv %q", argv)
	}
	if rt.start.Dir != "/work" || !reflect.DeepEqual(rt.start.Env, []string{"CLAUDE_CODE_OAUTH_TOKEN=" + testToken, mcpOutputTokens}) ||
		len(rt.start.Tools) != 1 || rt.start.Tools[0].Name != "report" {
		t.Errorf("start %+v", rt.start)
	}
	for _, a := range argv {
		if strings.Contains(a, testToken) {
			t.Fatal("token in argv")
		}
	}
	if len(rt.writes) != 2 || !strings.Contains(string(rt.writes[0]), `"initialize"`) || !strings.Contains(string(rt.writes[1]), `"content":"Do the work."`) {
		t.Fatalf("writes %q", rt.writes)
	}
	if !reflect.DeepEqual(seen, []string{`report {"x":1}`}) || len(rt.replies) != 1 || rt.replies[0].ID != 1 || rt.replies[0].Error != nil ||
		string(rt.replies[0].Result) != `{"content":[{"type":"text","text":"orchestrator says ok"}],"isError":false}` {
		t.Fatalf("tool round trip: %v %+v", seen, rt.replies)
	}
	want := run.Usage{Model: "claude-test-model", InputTokens: 11, OutputTokens: 22, CacheReadTokens: 33, CacheCreationTokens: 44, CostUSD: 0.25, ProviderTurns: 3}
	if res.Conversation != "conversation-1" || res.Final != "all done" || res.Usage != want || !res.Stopped || res.Started.IsZero() || res.Ended.Before(res.Started) ||
		!strings.Contains(res.Reason, "provider exit observed code=-1 signal=terminated") {
		t.Fatalf("result %+v", res)
	}
	if rt.stopCalls != 1 || !rt.closed {
		t.Fatal("container not stopped and closed")
	}
}

func TestVerifyTurnIsReadOnlyAndDropsWritingTools(t *testing.T) {
	profile := validProfile()
	profile.Session.BuiltinTools = []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}
	rt := newFakeRuntime()
	h := newHarness(t, rt, profile)
	turn := testTurn()
	turn.Kind, turn.ReadOnly, turn.Tools = run.VerifyTurn, true, nil
	if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil {
		t.Fatal(err)
	}
	if want := (container.Mount{Volume: "ws-volume", Target: "/work", ReadOnly: true, NoCopy: true}); rt.spec.Mounts[0] != want {
		t.Fatalf("mount %+v", rt.spec.Mounts[0])
	}
	if got := argAfter(rt.start.Argv, "--tools"); got != "Read,Glob,Grep,Bash" {
		t.Fatalf("tools %q", got)
	}
	if got := argAfter(rt.start.Argv, "--allowedTools"); got != "Read,Glob,Grep,Bash" {
		t.Fatalf("allowed %q", got)
	}
	// The configured profile is unchanged for later execute turns.
	if got := h.agents.profiles["claude"].Session.BuiltinTools; len(got) != 6 {
		t.Fatal(got)
	}
}

func TestWritingToolsFollowWhatTheTurnCanWrite(t *testing.T) {
	for name, tc := range map[string]struct {
		kind     run.TurnKind
		readOnly bool
		mounts   []run.PathMount
		want     string
	}{
		"enforced execute: read-only base, writable mount": {run.ExecuteTurn, true, []run.PathMount{{Path: "lending", Writable: true}}, "Read,Write,Edit,Glob,Grep,Bash"},
		"read-only base and only read-only mounts":         {run.ExecuteTurn, true, []run.PathMount{{Path: "lending"}}, "Read,Glob,Grep,Bash"},
		"read-only base, no mounts":                        {run.ExecuteTurn, true, nil, "Read,Glob,Grep,Bash"},
		"writable base":                                    {run.ExecuteTurn, false, []run.PathMount{{Path: "lending"}}, "Read,Write,Edit,Glob,Grep,Bash"},
		"verify":                                           {run.VerifyTurn, true, nil, "Read,Glob,Grep,Bash"},
	} {
		t.Run(name, func(t *testing.T) {
			profile := validProfile()
			profile.Session.BuiltinTools = []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}
			rt := newFakeRuntime()
			h := newHarness(t, rt, profile)
			turn := testTurn()
			turn.Kind, turn.ReadOnly, turn.Mounts = tc.kind, tc.readOnly, tc.mounts
			if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil {
				t.Fatal(err)
			}
			if got := argAfter(rt.start.Argv, "--tools"); got != tc.want {
				t.Fatalf("tools %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTurnMountsOverlayTheWorkspaceVolumeInOrder(t *testing.T) {
	root := []run.PathMount{{Path: "lending"}, {Path: "library"}}
	child := []run.PathMount{{Path: "lending", Writable: true}, {Path: "lending/reminders"}}
	for name, tc := range map[string]struct {
		readOnly bool
		mounts   []run.PathMount
		want     []container.Mount
	}{
		"root style: writable base, read-only overlays": {false, root, []container.Mount{
			{Volume: "ws-volume", Target: "/work", NoCopy: true},
			{Volume: "ws-volume", Target: "/work/lending", Subpath: "lending", ReadOnly: true, NoCopy: true},
			{Volume: "ws-volume", Target: "/work/library", Subpath: "library", ReadOnly: true, NoCopy: true},
			{Volume: "home-volume", Target: "/home/agent"}}},
		"child style: read-only base, writable overlays": {true, child, []container.Mount{
			{Volume: "ws-volume", Target: "/work", ReadOnly: true, NoCopy: true},
			{Volume: "ws-volume", Target: "/work/lending", Subpath: "lending", NoCopy: true},
			{Volume: "ws-volume", Target: "/work/lending/reminders", Subpath: "lending/reminders", ReadOnly: true, NoCopy: true},
			{Volume: "home-volume", Target: "/home/agent"}}},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newFakeRuntime()
			h := newHarness(t, rt, validProfile())
			turn := testTurn()
			turn.ReadOnly, turn.Mounts = tc.readOnly, tc.mounts
			if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rt.spec.Mounts, tc.want) {
				t.Fatalf("mounts %+v", rt.spec.Mounts)
			}
		})
	}
}

func TestVerifyTurnWithReadOnlyMountsStarts(t *testing.T) {
	rt := newFakeRuntime()
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Kind, turn.ReadOnly, turn.Tools, turn.Mounts = run.VerifyTurn, true, nil, []run.PathMount{{Path: "lending"}}
	if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil || len(rt.spec.Mounts) != 3 {
		t.Fatalf("%v %+v", err, rt.spec.Mounts)
	}
}

// A resumed turn whose provider starts another conversation would work on
// neither its history nor its standing context: it fails before any tool call
// is served, and reports the conversation it got.
func TestAResumeThatStartsAnotherConversationFails(t *testing.T) {
	rt := newFakeRuntime()
	rt.session = "observed-other"
	rt.calls = []frame.ToolCall{{ID: 1, Name: "report", Arguments: json.RawMessage(`{"kind":"basis"}`)}}
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Resume = "earlier"
	var served []string
	res, err := h.agents.Run(context.Background(), turn, echoHandler(&served))
	if err == nil || !strings.Contains(err.Error(), "instead of continuing earlier") || res.Conversation != "observed-other" || !res.Stopped || len(served) != 0 || argAfter(rt.start.Argv, "--resume") != "earlier" {
		t.Fatalf("%v %+v served %v", err, res, served)
	}
}

func TestVerifyTurnMayResumeItsReviewersConversationAndStaysReadOnly(t *testing.T) {
	rt := newFakeRuntime()
	rt.session = "reviewer-conversation"
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Kind, turn.ReadOnly, turn.Resume = run.VerifyTurn, true, "reviewer-conversation"
	if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil || argAfter(rt.start.Argv, "--resume") != "reviewer-conversation" {
		t.Fatalf("a resuming verify turn was refused: %v %q", err, rt.start.Argv)
	}
	for _, m := range rt.spec.Mounts {
		if m.Target == "/work" && !m.ReadOnly {
			t.Fatalf("the resumed verify turn can write the workspace: %+v", rt.spec.Mounts)
		}
	}
	// Resuming never lifts the read-only requirement.
	turn.ReadOnly = false
	h = newHarness(t, newFakeRuntime(), validProfile())
	if _, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string))); err == nil || h.starts != 0 {
		t.Fatalf("a writable resuming verify turn started: %v starts=%d", err, h.starts)
	}
}

func TestRefusedAnswerIsAToolError(t *testing.T) {
	rt := newFakeRuntime()
	rt.calls = []frame.ToolCall{{ID: 4, Name: "report", Arguments: json.RawMessage(`{}`)}, {ID: 5, Name: "nope", Arguments: json.RawMessage(`{}`)}}
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), func(context.Context, string, json.RawMessage) run.ToolAnswer {
		return run.ToolAnswer{Text: "not allowed now", Refused: true}
	})
	if err != nil || !res.Stopped {
		t.Fatal(err, res)
	}
	if len(rt.replies) != 2 || string(rt.replies[0].Result) != `{"content":[{"type":"text","text":"not allowed now"}],"isError":true}` ||
		rt.replies[1].Error == nil || rt.replies[1].Result != nil {
		t.Fatalf("replies %+v", rt.replies)
	}
}

func TestErrorResultKeepsUsageAndReportsError(t *testing.T) {
	rt := newFakeRuntime()
	rt.result = `{"type":"result","subtype":"error_max_turns","session_id":"conversation-1","uuid":"res","is_error":true,"errors":["too many turns"],"terminal_reason":"max_turns","num_turns":9,"total_cost_usd":1.5,"usage":{"input_tokens":5}}`
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err == nil || !strings.Contains(err.Error(), "error_max_turns") || res.Usage.InputTokens != 5 || res.Usage.ProviderTurns != 9 || res.Conversation != "conversation-1" || !res.Stopped {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestProviderEndingWithoutResultIsIncomplete(t *testing.T) {
	rt := newFakeRuntime()
	rt.endStream = true
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if !errors.Is(err, ErrIncomplete) || !res.Stopped || res.Conversation != "conversation-1" {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestTimeoutStopsTheContainerAndReturnsPartialResult(t *testing.T) {
	rt := newFakeRuntime()
	rt.silent = true
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Timeout = 100 * time.Millisecond
	res, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string)))
	if !errors.Is(err, context.DeadlineExceeded) || !res.Stopped || rt.stopCalls != 1 || !rt.closed || !strings.HasPrefix(res.Reason, "timeout;") || res.Conversation != "conversation-1" {
		t.Fatalf("%v %+v stops=%d", err, res, rt.stopCalls)
	}
}

func TestCancelStopsTheContainerAndReturnsPartialResult(t *testing.T) {
	rt := newFakeRuntime()
	rt.silent = true
	h := newHarness(t, rt, validProfile())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			rt.mu.Lock()
			n := len(rt.writes)
			rt.mu.Unlock()
			if n == 2 {
				cancel()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	res, err := h.agents.Run(ctx, testTurn(), echoHandler(new([]string)))
	if !errors.Is(err, context.Canceled) || !res.Stopped || rt.stopCalls != 1 || !rt.closed || !strings.HasPrefix(res.Reason, "cancelled;") {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestCancelWhileAnOrchestratorToolRuns(t *testing.T) {
	rt := newFakeRuntime()
	rt.calls = []frame.ToolCall{{ID: 1, Name: "report", Arguments: json.RawMessage(`{}`)}}
	h := newHarness(t, rt, validProfile())
	ctx, cancel := context.WithCancel(context.Background())
	res, err := h.agents.Run(ctx, testTurn(), func(ctx context.Context, _ string, _ json.RawMessage) run.ToolAnswer {
		cancel()
		<-ctx.Done()
		return run.ToolAnswer{Text: "late"}
	})
	if !errors.Is(err, context.Canceled) || !res.Stopped || rt.stopCalls != 1 {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestUncertainStopIsReportedAsNotStopped(t *testing.T) {
	rt := newFakeRuntime()
	rt.stopErr = errors.New("engine unreachable " + testToken)
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err == nil || res.Stopped || !strings.Contains(res.Reason, "provider exit not observed") || res.Final != "all done" {
		t.Fatalf("%v %+v", err, res)
	}
	if strings.Contains(err.Error(), testToken) || strings.Contains(res.Reason, testToken) {
		t.Fatal("token leaked")
	}
	if !rt.closed {
		t.Fatal("runtime not closed")
	}
}

func TestForcedCloseIsNotAnObservedContainerEnd(t *testing.T) {
	rt := newFakeRuntime()
	rt.closeErr = errors.Join(container.ErrForcedClose, errors.New("engine "+testToken))
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if !errors.Is(err, container.ErrForcedClose) || res.Stopped || !strings.Contains(res.Reason, "container end was not") || res.Final != "all done" {
		t.Fatalf("%v %+v", err, res)
	}
	if strings.Contains(err.Error(), testToken) || strings.Contains(res.Reason, testToken) {
		t.Fatal("token leaked")
	}
}

// assistantLine is one assistant message with its usage, as the provider streams it.
func assistantLine(id string, in, out, read, create int) string {
	return fmt.Sprintf(`{"type":"assistant","message":{"id":%q,"role":"assistant","content":[{"type":"text","text":"x"}],"usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d}},"parent_tool_use_id":null,"uuid":"u-%s-%d","session_id":"conversation-1"}`, id, in, out, read, create, id, out)
}

func TestStreamedUsageSurvivesATurnWithoutResult(t *testing.T) {
	for name, setup := range map[string]func(*fakeRuntime) (context.Context, context.CancelFunc, time.Duration){
		"timeout": func(*fakeRuntime) (context.Context, context.CancelFunc, time.Duration) {
			return context.Background(), func() {}, 150 * time.Millisecond
		},
		"cancel": func(*fakeRuntime) (context.Context, context.CancelFunc, time.Duration) {
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(150*time.Millisecond, cancel)
			return ctx, cancel, 0
		},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newFakeRuntime()
			rt.silent = true
			// The first message is repeated per content block with growing output.
			rt.extra = []string{assistantLine("m1", 10, 1, 100, 5), assistantLine("m1", 10, 7, 100, 5), assistantLine("m2", 20, 3, 200, 0)}
			h := newHarness(t, rt, validProfile())
			ctx, cancel, timeout := setup(rt)
			defer cancel()
			turn := testTurn()
			turn.Timeout = timeout
			res, err := h.agents.Run(ctx, turn, echoHandler(new([]string)))
			want := run.Usage{Model: "claude-test-model", InputTokens: 30, OutputTokens: 10, CacheReadTokens: 300, CacheCreationTokens: 5, ProviderTurns: 2}
			if err == nil || !res.Stopped || res.Usage != want || !strings.Contains(res.Reason, "usage partial: no result message") {
				t.Fatalf("%v %+v", err, res)
			}
		})
	}
}

func TestResultTotalsReplaceStreamedUsage(t *testing.T) {
	rt := newFakeRuntime()
	rt.extra = []string{assistantLine("m1", 1, 1, 1, 1)}
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	want := run.Usage{Model: "claude-test-model", InputTokens: 11, OutputTokens: 22, CacheReadTokens: 33, CacheCreationTokens: 44, CostUSD: 0.25, ProviderTurns: 3}
	if err != nil || res.Usage != want || strings.Contains(res.Reason, "usage partial") {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestResultWithoutTotalsKeepsStreamedUsageAndSaysPartial(t *testing.T) {
	rt := newFakeRuntime()
	rt.extra = []string{assistantLine("m1", 4, 5, 6, 7)}
	rt.result = `{"type":"result","subtype":"success","session_id":"conversation-1","uuid":"res","is_error":false,"result":"ok","num_turns":1}`
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err != nil || res.Usage.InputTokens != 4 || res.Usage.OutputTokens != 5 || !strings.Contains(res.Reason, "usage partial: the result message carries no usage") {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestStartFailuresReportNoObservedExit(t *testing.T) {
	rt := newFakeRuntime()
	rt.startErr = errors.New("provider refused " + testToken)
	h := newHarness(t, rt, validProfile())
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err == nil || strings.Contains(err.Error(), testToken) || !res.Stopped || rt.stopCalls != 1 {
		t.Fatalf("provider start failure must still stop the container: %v %+v", err, res)
	}
	failing, err := NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) {
		return nil, errors.New("no engine")
	}, map[string]Profile{"claude": validProfile()}, func(context.Context, string) (RuntimeCredential, error) {
		return RuntimeCredential{Token: testToken}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err = failing.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err == nil || res.Stopped || res.Reason == "" {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestInvalidTurnsFailBeforeAnyContainerStarts(t *testing.T) {
	for name, mutate := range map[string]func(*run.Turn){
		"unknown profile":        func(t *run.Turn) { t.Profile = "other" },
		"verify not read-only":   func(t *run.Turn) { t.Kind = run.VerifyTurn },
		"unknown kind":           func(t *run.Turn) { t.Kind = "other" },
		"bad ID":                 func(t *run.Turn) { t.ID = "-bad" },
		"same volumes":           func(t *run.Turn) { t.Home = t.Workspace },
		"empty prompt":           func(t *run.Turn) { t.Prompt = " " },
		"negative timeout":       func(t *run.Turn) { t.Timeout = -1 },
		"duplicate tool":         func(t *run.Turn) { t.Tools = append(t.Tools, t.Tools[0]) },
		"tool without schema":    func(t *run.Turn) { t.Tools[0].InputSchema = json.RawMessage(`[]`) },
		"tool without text":      func(t *run.Turn) { t.Tools[0].Description = "" },
		"relative transcript":    func(t *run.Turn) { t.Transcript = "relative.jsonl" },
		"prompt over wire limit": func(t *run.Turn) { t.Prompt = strings.Repeat("x", maxLineBytes) },
		"mount path escapes":     func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "../lending", Writable: true}} },
		"mount path absolute":    func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "/lending"}} },
		"mount path unclean":     func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "lending/./reminders"}} },
		"mount path empty":       func(t *run.Turn) { t.Mounts = []run.PathMount{{}} },
		"mount option inject":    func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "a,readonly=false"}} },
		"mount path twice":       func(t *run.Turn) { t.Mounts = []run.PathMount{{Path: "a"}, {Path: "a", Writable: true}} },
		"verify writable mount": func(t *run.Turn) {
			t.Kind, t.ReadOnly, t.Mounts = run.VerifyTurn, true, []run.PathMount{{Path: "lending", Writable: true}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, newFakeRuntime(), validProfile())
			turn := testTurn()
			turn.Tools = append([]run.ToolSpec(nil), turn.Tools...)
			mutate(&turn)
			res, err := h.agents.Run(context.Background(), turn, echoHandler(new([]string)))
			if err == nil || h.starts != 0 {
				t.Fatalf("err=%v starts=%d", err, h.starts)
			}
			if !res.Stopped || !strings.HasPrefix(res.Reason, "no container started: ") || res.Started.IsZero() {
				t.Fatalf("a turn that never started is over: %+v", res)
			}
		})
	}
	h := newHarness(t, newFakeRuntime(), validProfile())
	if _, err := h.agents.Run(context.Background(), testTurn(), nil); err == nil || h.starts != 0 {
		t.Fatal("nil handler accepted")
	}
}

func TestCredentialFailureStartsNothing(t *testing.T) {
	profile := validProfile()
	profile.CredentialRef = "missing"
	h := newHarness(t, newFakeRuntime(), profile)
	res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(new([]string)))
	if err == nil || h.starts != 0 || !res.Stopped || !strings.Contains(res.Reason, "no container started: ") || !strings.Contains(res.Reason, "missing") {
		t.Fatalf("err=%v starts=%d res=%+v", err, h.starts, res)
	}
}

func TestConstructorsRefuseInvalidInput(t *testing.T) {
	start := func(context.Context, container.ContainerSpec) (RuntimeSession, error) { return nil, nil }
	creds := func(context.Context, string) (RuntimeCredential, error) { return RuntimeCredential{}, nil }
	bad := validProfile()
	bad.Session.Effort = "max"
	for name, call := range map[string]func() (*Agents, error){
		"nil engine":      func() (*Agents, error) { return NewAgents(nil, nil, creds) },
		"nil start":       func() (*Agents, error) { return NewAgentsWithRuntime(nil, nil, creds) },
		"nil credentials": func() (*Agents, error) { return NewAgentsWithRuntime(start, nil, nil) },
		"invalid profile": func() (*Agents, error) { return NewAgentsWithRuntime(start, map[string]Profile{"claude": bad}, creds) },
		"empty name": func() (*Agents, error) {
			return NewAgentsWithRuntime(start, map[string]Profile{"": validProfile()}, creds)
		},
	} {
		if _, err := call(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestKnownTokenIsRedactedFromToolArgumentsBeforeTheHandler(t *testing.T) {
	escaped, _ := json.Marshal(testToken)
	rt := newFakeRuntime()
	rt.calls = []frame.ToolCall{{ID: 1, Name: "report", Arguments: json.RawMessage(`{"leak":` + string(escaped) + `}`)}}
	h := newHarness(t, rt, validProfile())
	var seen []string
	if _, err := h.agents.Run(context.Background(), testTurn(), echoHandler(&seen)); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || strings.Contains(seen[0], "sk-ant") || !strings.Contains(seen[0], "[REDACTED]") || !json.Valid([]byte(strings.TrimPrefix(seen[0], "report "))) {
		t.Fatalf("handler saw %q", seen)
	}
}

func TestTranscriptRecordsTheTurnWithoutTheToken(t *testing.T) {
	escaped, _ := json.Marshal(testToken)
	rt := newFakeRuntime()
	rt.extra = []string{`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":` + string(escaped) + `}]},"parent_tool_use_id":null,"uuid":"a-1","session_id":"conversation-1"}`}
	rt.calls = []frame.ToolCall{{ID: 1, Name: "report", Arguments: json.RawMessage(`{"leak":` + string(escaped) + `}`)}}
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Transcript = filepath.Join(t.TempDir(), "nested", "deeper", "turn.jsonl")
	_, err := h.agents.Run(context.Background(), turn, func(context.Context, string, json.RawMessage) run.ToolAnswer {
		return run.ToolAnswer{Text: "echo " + testToken}
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(turn.Transcript)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, testToken) || strings.Contains(text, string(escaped[1:len(escaped)-1])) || !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("token not redacted:\n%s", text)
	}
	kinds := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		var entry struct {
			Kind string
			Data json.RawMessage
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil || entry.Kind == "" || len(entry.Data) == 0 {
			t.Fatalf("not JSON Lines: %q", line)
		}
		kinds[entry.Kind]++
	}
	if kinds["sent"] != 2 || kinds["provider"] < 4 || kinds["tool_call"] != 1 || kinds["tool_answer"] != 1 || kinds["end"] != 1 {
		t.Fatalf("kinds %v", kinds)
	}
}

// Every turn runs in a fresh container of its own, stopped and closed before
// Run returns; the next turn of the same conversation starts another one with
// the same provider home and resumes the conversation, so container lifetime
// and conversation are separate.
func TestEveryTurnGetsAFreshContainerAndKeepsItsConversation(t *testing.T) {
	var specs []container.ContainerSpec
	var runtimes []*fakeRuntime
	agents, err := NewAgentsWithRuntime(func(_ context.Context, spec container.ContainerSpec) (RuntimeSession, error) {
		if n := len(runtimes); n > 0 && !runtimes[n-1].closed {
			t.Errorf("container %s started while %s still ran", spec.Name, specs[n-1].Name)
		}
		rt := newFakeRuntime()
		rt.session = "conversation-1"
		specs, runtimes = append(specs, spec), append(runtimes, rt)
		return rt, nil
	}, map[string]Profile{"claude": validProfile()}, func(context.Context, string) (RuntimeCredential, error) {
		return RuntimeCredential{Token: testToken}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first := testTurn()
	res, err := agents.Run(context.Background(), first, echoHandler(new([]string)))
	if err != nil || !res.Stopped || res.Conversation != "conversation-1" {
		t.Fatalf("first turn %+v %v", res, err)
	}
	second := testTurn()
	second.ID, second.Resume = "turn-2", res.Conversation
	if res, err := agents.Run(context.Background(), second, echoHandler(new([]string))); err != nil || !res.Stopped || res.Conversation != "conversation-1" {
		t.Fatalf("second turn %+v %v", res, err)
	}
	if len(specs) != 2 || specs[0].Name == specs[1].Name || !runtimes[0].closed || !runtimes[1].closed {
		t.Fatalf("containers %q, closed %v %v", []string{specs[0].Name, specs[1].Name}, runtimes[0].closed, runtimes[1].closed)
	}
	home := func(spec container.ContainerSpec) string {
		for _, m := range spec.Mounts {
			if m.Target == "/home/agent" {
				return m.Volume
			}
		}
		return ""
	}
	if home(specs[0]) != "home-volume" || home(specs[1]) != "home-volume" || argAfter(runtimes[0].start.Argv, "--resume") != "<absent>" || argAfter(runtimes[1].start.Argv, "--resume") != "conversation-1" {
		t.Fatalf("homes %q %q, resume %q then %q", home(specs[0]), home(specs[1]), argAfter(runtimes[0].start.Argv, "--resume"), argAfter(runtimes[1].start.Argv, "--resume"))
	}
}
