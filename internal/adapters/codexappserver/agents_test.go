package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const dedicatedLogin = `{"auth_mode":"chatgpt","tokens":{"id_token":"private-id","access_token":"private-access","refresh_token":"private-refresh","account_id":"selected-account"}}`

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

	mu            sync.Mutex
	spec          container.ContainerSpec
	start         frame.Start
	writes        []string
	replies       []frame.ToolReply
	stops         int
	starts        int
	closed        bool
	homeCalls     []string
	installed     []byte
	initializeErr error
	cleanupErr    error

	closeErr       error
	toolArgs       string
	useTool        bool
	toolThread     string
	toolName       string
	hang           bool
	usage          bool
	finalText      string
	finalPhase     *string
	commandItems   []map[string]any
	terminalStatus string
	resumeMismatch bool
	threadError    bool
	errorText      string
	stopFail       bool
	notifications  []string
}

func newRuntime() *fakeRuntime {
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 64), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16), toolName: "ask", toolArgs: `{"to":"a","text":"question"}`, finalText: `{"decision":"submitted","findings":[]}`}
}
func (f *fakeRuntime) emit(raw string) { f.stream.chunks <- []byte(raw + "\n") }
func (f *fakeRuntime) Initialize(_ context.Context, req frame.Initialize) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.homeCalls = append(f.homeCalls, "initialize")
	f.installed = bytes.Clone(req.CodexAuth)
	return f.initializeErr
}
func (f *fakeRuntime) ClearPrivateHome(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.homeCalls = append(f.homeCalls, "clear")
	return f.cleanupErr
}
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.start = s
	f.starts++
	f.homeCalls = append(f.homeCalls, "start")
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
		ID     int
	}
	_ = json.Unmarshal(data, &m)
	switch m.Method {
	case "initialize":
		f.emit(`{"id":1,"result":` + codecInitialization + `}`)
	case "thread/start", "thread/resume":
		if f.threadError {
			f.emit(`{"id":2,"error":{"code":-32000,"message":"thread unavailable"}}`)
			break
		}
		thread := codecThread
		if f.resumeMismatch {
			thread = strings.ReplaceAll(thread, string(codecThreadID), "wrong-thread")
		}
		f.emit(`{"id":2,"result":{"thread":` + thread + `,"approvalPolicy":"never","approvalsReviewer":"user","cwd":"/work","model":"admitted-model","modelProvider":"markitect","sandbox":{"type":"dangerFullAccess"}}}`)
	case "turn/start":
		f.emit(fmt.Sprintf(`{"id":%d,"result":{"turn":{"id":"%s","status":"inProgress","items":[],"error":null}}}`, m.ID, codecTurnID))
		for _, notification := range f.notifications {
			f.emit(notification)
		}
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
	phase := "final_answer"
	if f.finalPhase != nil {
		phase = *f.finalPhase
	}
	for _, command := range f.commandItems {
		data, _ := json.Marshal(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": codecThreadID, "turnId": codecTurnID, "item": command}})
		f.emit(string(data))
	}
	item, _ := json.Marshal(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": codecThreadID, "turnId": codecTurnID, "item": map[string]any{"id": "answer", "type": "agentMessage", "phase": phase, "text": f.finalText}}})
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
	f.homeCalls = append(f.homeCalls, "stop")
	f.mu.Unlock()
	if f.stopFail {
		return frame.Exited{}, io.ErrClosedPipe
	}
	return frame.Exited{Code: -1, Signal: "terminated"}, nil
}
func (f *fakeRuntime) Close() error {
	f.mu.Lock()
	f.closed = true
	f.homeCalls = append(f.homeCalls, "close")
	f.mu.Unlock()
	f.stream.close()
	return f.closeErr
}

func (f *fakeRuntime) ImageID() string { return fixtureImage }
func testSessionSpec(t *testing.T) run.SessionSpec {
	t.Helper()
	r := testkit.NewRepo(t)
	core := "../../../docs/design/canonical-model/minimal-language/exchange-example/dependencies/core"
	if err := fs.WalkDir(os.DirFS(core), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(filepath.Join(core, filepath.FromSlash(p)))
		if err == nil {
			r.Write(".markitect/dependencies/core/"+p, string(data))
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	r.Write(".markitect/model.yaml", "formatVersion: 1\nid: native-fixture\nroot: model\ndependencies: {markitect-core: '1.0.0'}\n")
	r.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0.0', digest: 'sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1'}\n")
	r.Write(".markitect/model/a/thing.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Thing\nkind: Core.Kind\nbody: Preserve the native fixture obligation.\nspec: {}\n")
	r.Write(".markitect/execution.yaml", "apiVersion: markitect.execution/v2\ninstructions: {'.': [guide.md]}\nchecks: []\n")
	r.Write("guide.md", "SELECTED MANAGER GUIDANCE ✓\n")
	r.Write(".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts: []\n")
	r.Commit("native fixture")
	repo, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := project.NewPreparationService(project.NewModelService(repo, repo)).Capture(context.Background(), project.CaptureRequest{Selection: project.InputSelection{Repository: r.Dir, Local: project.ModelSelection{Boundary: ".", ManifestPath: testPath(t, ".markitect/model.yaml")}, Dependencies: []project.ModelSelection{{Boundary: ".markitect/dependencies/core", ManifestPath: testPath(t, "model.yaml")}}}}, model.Limits{})
	a, ok := result.Prepared()
	if err != nil || !ok {
		t.Fatalf("actual fixture preparation: %v %s %+v", err, result.Status(), result.Attempts())
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	manager, _ := p.Topology().Manager(".")
	tools := run.Tools{Delegate: func(context.Context, string, string, []run.Finding) error { return nil }, Await: func(context.Context) (run.Event, error) {
		return run.Event{Question: &run.Question{ID: "pending", From: "a", To: ".", Text: "counter-question"}}, nil
	}, Review: func(context.Context, string, string, run.Decision, []run.Finding) error { return nil }, Ask: func(context.Context, string, string) (string, error) { return "new-id", nil }, Reply: func(context.Context, string, string) error { return nil }, ReadNotes: func(context.Context) ([]run.Note, error) { return nil, nil }, AppendNote: func(context.Context, string, string) (run.Note, error) { return run.Note{ID: "note"}, nil }}
	return run.SessionSpec{RunID: "native-run", Manager: manager, Prepared: p, Mandate: run.Implement, Workspace: &sessionCopy{location: run.CopyLocation{Volume: "private-run-volume", Subpath: "copies/root"}}, Tools: tools}
}

type sessionCopy struct {
	run.Copy
	location run.CopyLocation
}

func (c *sessionCopy) Location() run.CopyLocation { return c.location }
func agentsFor(t *testing.T, rt *fakeRuntime) *Agents {
	t.Helper()
	p := preparedExecution(t, executionRuntimeProfile(filepath.Join(t.TempDir(), "engine")), imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	a, err := NewAgentsWithRuntime(func(_ context.Context, s container.ContainerSpec) (RuntimeSession, error) {
		rt.spec = s
		return rt, nil
	}, p, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "exact-runtime-credential" {
			t.Fatal("wrong reference")
		}
		return RuntimeCredential{Mode: "dedicated-runtime-login", RuntimeLogin: json.RawMessage(dedicatedLogin)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func openSession(t *testing.T, rt *fakeRuntime, spec run.SessionSpec) *nativeSession {
	t.Helper()
	s, err := agentsFor(t, rt).Open(context.Background(), spec)
	if err != nil {
		if s != nil {
			_ = s.Close(context.Background())
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return s.(*nativeSession)
}
func TestSessionUsesHarnessCopySubpathAndPersistentNativeThread(t *testing.T) {
	rt := newRuntime()
	s := openSession(t, rt, testSessionSpec(t))
	if rt.starts != 1 || len(rt.writes) != 2 {
		t.Fatal("Open eagerly started inference", rt.starts, rt.writes)
	}
	if got := rt.spec.Mounts[0]; got != (container.Mount{Volume: "private-run-volume", Subpath: "copies/root", Target: "/work", NoCopy: true}) {
		t.Fatal("copy projection", got)
	}
	if rt.spec.Mounts[1].Target != "/home/agent" || rt.spec.Mounts[1].Volume == rt.spec.Mounts[0].Volume {
		t.Fatal("home aliases workspace")
	}
	for _, task := range []string{"first task", "recall first task"} {
		res, err := s.Exchange(context.Background(), run.Message{Task: task})
		if err != nil || res.Decision != run.Submitted {
			t.Fatal(res, err)
		}
	}
	starts := 0
	for _, raw := range rt.writes {
		var m struct {
			Method string
			Params struct {
				BaseInstructions, DeveloperInstructions string
				OutputSchema                            json.RawMessage
			}
		}
		_ = json.Unmarshal([]byte(raw), &m)
		if m.Method == "thread/start" {
			starts++
			for _, part := range []string{"Prepared binding:", "Complete Manager hierarchy", "SELECTED MANAGER GUIDANCE ✓", "Initial Findings", "implement"} {
				if !strings.Contains(m.Params.DeveloperInstructions, part) {
					t.Fatalf("missing harness %q", part)
				}
			}
			if !strings.Contains(m.Params.BaseInstructions, "checked canonical model is authoritative") {
				t.Fatal("rules absent")
			}
		}
		if m.Method == "turn/start" && len(m.Params.OutputSchema) == 0 {
			t.Fatal("result schema absent")
		}
	}
	if starts != 1 {
		t.Fatal("new native thread per Exchange", starts)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt.homeCalls, []string{"initialize", "start", "stop", "clear", "close"}) {
		t.Fatal(rt.homeCalls)
	}
}
func TestSessionCloseCancelsAndDrainsInboxCallback(t *testing.T) {
	rt := newRuntime()
	rt.useTool = true
	rt.toolName = "await"
	rt.toolArgs = "{}"
	spec := testSessionSpec(t)
	entered := make(chan struct{})
	drained := make(chan struct{})
	spec.Tools.Await = func(ctx context.Context) (run.Event, error) {
		close(entered)
		<-ctx.Done()
		close(drained)
		return run.Event{}, ctx.Err()
	}
	s := openSession(t, rt, spec)
	done := make(chan error, 1)
	go func() { _, e := s.Exchange(context.Background(), run.Message{Task: "wait"}); done <- e }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no callback")
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-drained:
	default:
		t.Fatal("callback not drained")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Exchange active")
	}
}
func TestSessionPartialStartupRetainsHandleAndUncertainClose(t *testing.T) {
	rt := newRuntime()
	rt.initializeErr = errors.New("private-access startup")
	rt.closeErr = errors.New("close uncertain")
	s, err := agentsFor(t, rt).Open(context.Background(), testSessionSpec(t))
	if s == nil || err == nil || strings.Contains(err.Error(), "private-access") {
		t.Fatal(s, err)
	}
	if err := s.Close(context.Background()); err == nil || !strings.Contains(err.Error(), "close uncertain") {
		t.Fatal(err)
	}
	if rt.starts != 0 || !rt.closed {
		t.Fatal("failed setup started provider")
	}
}
func TestSessionNativeFailuresCloseBeforeExchangeReturns(t *testing.T) {
	for _, kind := range []string{"malformed", "decision", "provider", "secret"} {
		t.Run(kind, func(t *testing.T) {
			rt := newRuntime()
			switch kind {
			case "malformed":
				rt.finalText = "not JSON"
			case "decision":
				rt.finalText = `{"decision":"success","findings":[]}`
			case "provider":
				rt.terminalStatus = "failed"
			case "secret":
				rt.errorText = "private-access"
			}
			s := openSession(t, rt, testSessionSpec(t))
			_, err := s.Exchange(context.Background(), run.Message{Task: "work"})
			if err == nil || !rt.closed || strings.Contains(err.Error(), "private-access") {
				t.Fatal(err, rt.closed)
			}
		})
	}
}
func TestSessionAdmissionRejectsDirectoryAndForgedManagerBeforeCredentials(t *testing.T) {
	spec := testSessionSpec(t)
	rt := newRuntime()
	a := agentsFor(t, rt)
	calls := 0
	a.credentials = func(context.Context, string) (RuntimeCredential, error) { calls++; return RuntimeCredential{}, nil }
	spec.Workspace = &sessionCopy{location: run.CopyLocation{Directory: t.TempDir()}}
	if s, e := a.Open(context.Background(), spec); e == nil || s != nil || calls != 0 {
		t.Fatal("Directory admitted")
	}
	spec.Workspace = &sessionCopy{location: run.CopyLocation{Volume: "volume", Subpath: "copy"}}
	spec.Manager.Children = nil
	if s, e := a.Open(context.Background(), spec); e == nil || s != nil || calls != 0 {
		t.Fatal("forged Manager admitted")
	}
}
func TestSessionInboxProjectionRefusalAndForeignCallRejection(t *testing.T) {
	for _, kind := range []string{"ask", "await", "reply", "foreign thread", "foreign tool", "refused"} {
		t.Run(kind, func(t *testing.T) {
			rt := newRuntime()
			rt.useTool = true
			spec := testSessionSpec(t)
			switch kind {
			case "await":
				rt.toolName = "await"
				rt.toolArgs = "{}"
			case "reply":
				rt.toolName = "reply"
				rt.toolArgs = `{"id":"incoming","text":"answer"}`
			case "foreign thread":
				rt.toolThread = "foreign"
			case "foreign tool":
				rt.toolName = "host_shell"
			case "refused":
				spec.Tools.Ask = func(context.Context, string, string) (string, error) { return "", errors.New("refused private-access") }
			}
			s := openSession(t, rt, spec)
			_, err := s.Exchange(context.Background(), run.Message{Task: "coordinate"})
			if strings.HasPrefix(kind, "foreign") {
				if err == nil || len(rt.replies) != 0 {
					t.Fatal("foreign call accepted")
				}
				return
			}
			if err != nil || len(rt.replies) != 1 || strings.Contains(string(rt.replies[0].Result), "private-access") {
				t.Fatal(err, rt.replies)
			}
		})
	}
}
