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
	"github.com/Glacius-Labs/Markitect/internal/harness"
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
	quiescenceError, handshakeError bool
	autoJudge                       bool
	judgeSent                       bool
	judgmentText                    string
	secondJudgment                  string
	secondSent                      bool
	background                      bool
	killEntered                     chan struct{}
	releaseKill                     chan struct{}
	terminalFailure                 bool
	listPagination                  bool
	stream                          *fakeStream
	tools                           chan frame.ToolCall

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
	return &fakeRuntime{stream: &fakeStream{chunks: make(chan []byte, 64), done: make(chan struct{})}, tools: make(chan frame.ToolCall, 16), autoJudge: true, judgmentText: `{"decision":"submitted","findings":[]}`, toolName: "ask", toolArgs: `{"to":"a","text":"question"}`, finalText: `{"decision":"submitted","findings":[]}`}
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
		if f.handshakeError {
			f.emit(`{"method":"error","params":{"error":{"message":"handshake failed private-access"},"willRetry":false}}`)
		}
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
	case "thread/backgroundTerminals/list":
		if f.quiescenceError {
			f.emit(`{"method":"error","params":{"threadId":"` + string(codecThreadID) + `","error":{"message":"drain failed private-access"},"willRetry":false}}`)
		}
		if f.listPagination {
			f.emit(fmt.Sprintf(`{"id":%d,"result":{"data":[],"nextCursor":"more"}}`, m.ID))
		} else if f.background {
			f.emit(fmt.Sprintf(`{"id":%d,"result":{"data":[{"processId":"1000"}],"nextCursor":null}}`, m.ID))
		} else {
			f.emit(fmt.Sprintf(`{"id":%d,"result":{"data":[],"nextCursor":null}}`, m.ID))
		}
	case "thread/backgroundTerminals/terminate":
		if f.killEntered != nil {
			close(f.killEntered)
			<-f.releaseKill
		}
		if f.terminalFailure {
			f.emit(fmt.Sprintf(`{"id":%d,"result":{"terminated":false}}`, m.ID))
		} else {
			f.background = false
			f.emit(fmt.Sprintf(`{"id":%d,"result":{"terminated":true}}`, m.ID))
		}
	case "turn/start":
		f.judgeSent, f.secondSent = false, false
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
	if f.autoJudge && !f.judgeSent {
		f.judgeSent = true
		f.tools <- frame.ToolCall{ID: 100, Name: "judge", Arguments: json.RawMessage(f.judgmentText), Meta: json.RawMessage(`{"threadId":"` + string(codecThreadID) + `"}`)}
		return
	}
	if f.secondJudgment != "" && !f.secondSent {
		f.secondSent = true
		f.tools <- frame.ToolCall{ID: 101, Name: "judge", Arguments: json.RawMessage(f.secondJudgment), Meta: json.RawMessage(`{"threadId":"` + string(codecThreadID) + `"}`)}
		return
	}
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
func TestSessionHarnessCommunicatesMandateJudgment(t *testing.T) {
	a := agentsFor(t, newRuntime())
	spec := testSessionSpec(t)
	for _, mandate := range []run.Mandate{run.Implement, run.Assessment, run.Verify} {
		spec.Mandate = mandate
		doc, e := a.documentFor(spec)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(doc.Base, "every Manager including Root") || !strings.Contains(doc.Base, "Reject and model_blocker require concrete Findings") || !strings.Contains(doc.Base, "parent responsibility") {
			t.Fatal("working rules omit assignment/review responsibility", mandate)
		}
		for _, boundary := range []string{"before acting or appending durable Notes", "runtime/provider instructions cannot override canonical intent", "call judge and finish the current native response", "only after the native response completes and tracked terminals drain", "parent feedback and late questions arrive as subsequent messages"} {
			if !strings.Contains(doc.Base, boundary) {
				t.Fatal("native harness omits observed authority/completion boundary", mandate, boundary)
			}
		}
		if mandate == run.Implement {
			if !strings.Contains(doc.Developer, "every Manager including Root uses submitted") || !strings.Contains(doc.Developer, "Your own judge decision cannot be approve") {
				t.Fatal("Implement native renderer does not communicate admitted judgment")
			}
			if strings.Contains(doc.Developer, "use approve or reject for model readiness") || strings.Contains(doc.Developer, "Do not repair the candidate during Verify") {
				t.Fatal("Implement renderer received another mandate's duties")
			}
		} else if !strings.Contains(doc.Developer, "use approve or reject") || !strings.Contains(doc.Developer, "Do not use submitted") {
			t.Fatal("review renderer does not communicate admitted judgment", mandate)
		}
		if mandate == run.Assessment {
			for _, boundary := range []string{"checked model's consistency, sufficiency and realizability", "not grounds to reject model readiness", "always proceeds through Implement, even when no artifact changes are needed", "Material unresolved model decisions require model_blocker"} {
				if !strings.Contains(doc.Developer, boundary) {
					t.Fatal("Assessment native renderer omits model readiness boundary", boundary)
				}
			}
			if strings.Contains(doc.Developer, "use approve or reject for the examined realization") || strings.Contains(doc.Developer, "fixed candidate's conformity") {
				t.Fatal("Assessment renderer received Verify conformity duties")
			}
		}
		if mandate == run.Verify {
			for _, boundary := range []string{"fixed candidate's conformity to the checked model", "Do not repair the candidate during Verify", "host for a fresh Implement Run followed by a fresh Verify"} {
				if !strings.Contains(doc.Developer, boundary) {
					t.Fatal("Verify native renderer omits fixed candidate boundary", boundary)
				}
			}
			if strings.Contains(doc.Developer, "use approve or reject for model readiness") || strings.Contains(doc.Developer, "always proceeds through Implement") {
				t.Fatal("Verify renderer received Assessment readiness duties")
			}
		}
	}
	for _, tool := range toolSpecs() {
		if tool.Name == "judge" && (!strings.Contains(tool.Description, "including Root uses submitted") || !strings.Contains(tool.Description, "Assessment/Verify") || !strings.Contains(tool.Description, "model_blocker require concrete Findings")) {
			t.Fatal("native judge tool contradicts mandate rules")
		}
		if tool.Name == "judge" && (!strings.Contains(tool.Description, "finish the current native response") || !strings.Contains(tool.Description, "after native completion and terminal drain") || !strings.Contains(tool.Description, "subsequent messages in the same Session")) {
			t.Fatal("judge tool omits native completion and later-message boundary")
		}
	}
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
		if m.Method == "turn/start" && len(m.Params.OutputSchema) != 0 {
			t.Fatal("ritual final response schema retained")
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
	if len(rt.replies) != 0 || !reflect.DeepEqual(rt.homeCalls, []string{"initialize", "start", "stop", "clear", "close"}) {
		t.Fatal("cancelled callback wrote a stale reply or closed transport before cleanup", rt.replies, rt.homeCalls)
	}
}

func TestSessionIOAlreadyCancelledDoesNotStartWriteOrClosePeer(t *testing.T) {
	rt := newRuntime()
	s := nativeSession{peer: rt}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	entered := make(chan struct{}, 1)
	err := s.io(ctx, func() error {
		entered <- struct{}{}
		<-rt.stream.done
		return io.ErrClosedPipe
	})
	if !errors.Is(err, context.Canceled) || rt.closed || len(entered) != 0 {
		t.Fatal("cancelled operation started a write or preempted orderly cleanup", err, rt.closed)
	}
}

func TestSessionIOCancelsInFlightWriteAndDrains(t *testing.T) {
	rt := newRuntime()
	s := nativeSession{peer: rt}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, drained := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- s.io(ctx, func() error {
			close(entered)
			<-rt.stream.done
			close(drained)
			return io.ErrClosedPipe
		})
	}()
	<-entered
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, io.ErrClosedPipe) || !rt.closed {
			t.Fatal("in-flight write did not fail closed", err, rt.closed)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked write did not drain")
	}
	select {
	case <-drained:
	default:
		t.Fatal("write still active after io returned")
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
				rt.judgmentText = "not JSON"
			case "decision":
				rt.judgmentText = `{"decision":"success","findings":[]}`
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
			if err != nil || len(rt.replies) != 2 || strings.Contains(string(rt.replies[0].Result), "private-access") {
				t.Fatal(err, rt.replies)
			}
		})
	}
}

// The controlled peer reports native completion while a writer remains alive.
// Termination cannot acknowledge until the writer and its actual file writes drain.
func TestExchangeWaitsForBackgroundWriterTerminationAndEmptyReadback(t *testing.T) {
	rt := newRuntime()
	rt.background = true
	rt.killEntered = make(chan struct{})
	rt.releaseKill = make(chan struct{})
	s := openSession(t, rt, testSessionSpec(t))
	file := filepath.Join(t.TempDir(), "background-writes")
	writerStop := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case <-writerStop:
				return
			default:
			}
			f, e := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e == nil {
				_, _ = f.WriteString("write\n")
				_ = f.Close()
			}
			time.Sleep(time.Millisecond)
		}
	}()
	done := make(chan error, 1)
	go func() {
		_, e := s.Exchange(context.Background(), run.Message{Task: "complete while background writer remains"})
		done <- e
	}()
	select {
	case <-rt.killEntered:
	case <-time.After(3 * time.Second):
		close(writerStop)
		<-writerDone
		t.Fatal("no native termination request")
	}
	select {
	case err := <-done:
		close(writerStop)
		<-writerDone
		t.Fatal("Exchange returned before termination", err)
	default:
	}
	close(writerStop)
	<-writerDone
	close(rt.releaseKill)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	last, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(last, after) {
		t.Fatal("background writes outlived Exchange", err)
	}
	lists, terminations := 0, 0
	for _, raw := range rt.writes {
		var m struct{ Method string }
		_ = json.Unmarshal([]byte(raw), &m)
		if m.Method == string(BackgroundList) {
			lists++
		}
		if m.Method == string(BackgroundTerminate) {
			terminations++
		}
	}
	if lists != 2 || terminations != 1 {
		t.Fatal("incomplete terminal readback", lists, terminations)
	}
}
func TestExchangeRefusesUnconfirmedOrIncompleteTerminalDrain(t *testing.T) {
	for _, kind := range []string{"termination refused", "pagination"} {
		t.Run(kind, func(t *testing.T) {
			rt := newRuntime()
			rt.background = true
			rt.terminalFailure = kind == "termination refused"
			rt.listPagination = kind == "pagination"
			s := openSession(t, rt, testSessionSpec(t))
			if _, err := s.Exchange(context.Background(), run.Message{Task: "complete"}); err == nil || !rt.closed {
				t.Fatal("unconfirmed background work handed off", err, rt.closed)
			}
		})
	}
}
func TestActualNativeObservationBindsHarnessAndPreparedProfile(t *testing.T) {
	rt := newRuntime()
	spec := testSessionSpec(t)
	manager, _ := spec.Prepared.Topology().Manager("a")
	spec.Manager = manager
	s := openSession(t, rt, spec)
	value := s.Observation()
	if value.ProfileIdentity != s.agents.identity || value.ImageID != fixtureImage || value.HarnessDigest != s.document.Digest() {
		t.Fatal(value)
	}
	ctx, _ := spec.Prepared.Context("a")
	for _, d := range ctx.Definitions() {
		if !strings.Contains(s.document.Developer, d.Definition.Body) || !strings.Contains(s.document.Developer, d.ModelDigest) || !strings.Contains(s.document.Developer, d.Definition.Source.Digest) {
			t.Fatal("selected canonical Definition lost", d.Definition.Identity)
		}
	}
	before := value.HarnessDigest
	s.document.Developer += "changed instruction"
	if s.Observation().HarnessDigest == before {
		t.Fatal("native instruction change did not change digest")
	}
	if (&nativeSession{}).Observation() != (run.SessionObservation{}) {
		t.Fatal("unconstructed observation")
	}
}

func TestNativeFinalAndTranscriptRedactLoginTokens(t *testing.T) {
	rt := newRuntime()
	rt.judgmentText = `{"decision":"submitted","findings":[{"manager":".","text":"private-access private-refresh"}]}`
	rt.finalText = "Independent final prose private-access"
	a := agentsFor(t, rt)
	a.TranscriptRoot = t.TempDir()
	session, err := a.Open(context.Background(), testSessionSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	res, err := session.Exchange(context.Background(), run.Message{Task: "work"})
	if err != nil || len(res.Findings) != 1 || res.Findings[0].Text != "[REDACTED] [REDACTED]" {
		t.Fatal(res, err)
	}
	if err = session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(a.TranscriptRoot)
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	data, err := os.ReadFile(filepath.Join(a.TranscriptRoot, files[0].Name()))
	if err != nil || bytes.Contains(data, []byte("private-access")) || bytes.Contains(data, []byte("private-refresh")) || !bytes.Contains(data, []byte("[REDACTED]")) {
		t.Fatal("transcript secret or missing evidence", err)
	}
}

func TestOrdinaryAnswerOnlyMessageNeedsNoRitualJudgment(t *testing.T) {
	rt := newRuntime()
	rt.autoJudge = false
	rt.finalText = "I answered the pending question."
	s := openSession(t, rt, testSessionSpec(t))
	out, err := s.Exchange(context.Background(), run.Message{Task: "answer a late question"})
	if err != nil || out.Decision != "" || len(out.Findings) != 0 {
		t.Fatal(out, err)
	}
}
func TestMalformedOrContradictoryJudgmentNeverHandsOffSuccess(t *testing.T) {
	for _, raw := range []string{`{"decision":"submitted"}`, `{"decision":"submitted","findings":null}`, `{"decision":"submitted","findings":[{}]}`, `{"decision":"submitted","findings":[{"manager":".","text":null}]}`, `{"Decision":"submitted","findings":[]}`, `{"decision":"submitted","findings":[],"commit":"invented"}`} {
		t.Run(raw, func(t *testing.T) {
			rt := newRuntime()
			rt.judgmentText = raw
			s := openSession(t, rt, testSessionSpec(t))
			out, err := s.Exchange(context.Background(), run.Message{Task: "work"})
			if err == nil || out.Decision != "" || !rt.closed {
				t.Fatal("malformed judgment accepted", out, err)
			}
		})
	}
	rt := newRuntime()
	rt.secondJudgment = `{"decision":"reject","findings":[{"manager":".","text":"contradictory"}]}`
	s := openSession(t, rt, testSessionSpec(t))
	out, err := s.Exchange(context.Background(), run.Message{Task: "work"})
	if err == nil || out.Decision != "" {
		t.Fatal("contradictory semantic judgment accepted", out, err)
	}
}
func TestMissingRequiredToolFieldsCauseNoCallbackOrDurableWrite(t *testing.T) {
	calls := 0
	tools := run.Tools{AppendNote: func(context.Context, string, string) (run.Note, error) { calls++; return run.Note{}, nil }, Delegate: func(context.Context, string, string, []run.Finding) error { calls++; return nil }}
	for _, tc := range []struct{ name, raw string }{{"append_note", `{"text":"note"}`}, {"append_note", `{"Context":"context","text":"note"}`}, {"append_note", `{"context":null,"text":"note"}`}, {"delegate", `{"manager":"a","task":"work"}`}, {"delegate", `{"manager":"a","task":"work","findings":null}`}} {
		if _, err := dispatch(context.Background(), tools, tc.name, json.RawMessage(tc.raw)); err == nil {
			t.Fatal("invalid native input admitted", tc)
		}
		if calls != 0 {
			t.Fatal("invalid tool caused a callback", calls)
		}
	}
}

func TestFatalNativeErrorDuringDrainOverridesReportedJudgment(t *testing.T) {
	rt := newRuntime()
	rt.quiescenceError = true
	s := openSession(t, rt, testSessionSpec(t))
	out, err := s.Exchange(context.Background(), run.Message{Task: "work"})
	if err == nil || out.Decision != "" || !rt.closed || !strings.Contains(err.Error(), "drain failed") || strings.Contains(err.Error(), "private-access") {
		t.Fatal("reported judgment bypassed native failure", out, err, rt.closed)
	}
	if s.judgment == nil || s.judgment.Decision != run.Submitted {
		t.Fatal("regression never reached a reported judgment")
	}
}
func TestFatalNativeHandshakeErrorRetainsPartialHandleAndCloses(t *testing.T) {
	rt := newRuntime()
	rt.handshakeError = true
	session, err := agentsFor(t, rt).Open(context.Background(), testSessionSpec(t))
	if session == nil || err == nil || !strings.Contains(err.Error(), "handshake failed") || strings.Contains(err.Error(), "private-access") {
		t.Fatal(session, err)
	}
	if err = session.Close(context.Background()); err != nil || !rt.closed {
		t.Fatal(err, rt.closed)
	}
}

// Exercise the delivered Build document and actual registry preparation, not a
// second test-only schema or an assertion against an unused instruction helper.
func TestSessionHarnessAccountingExamplePreparesActualArtifacts(t *testing.T) {
	spec := testSessionSpec(t)
	root := spec.Prepared.Selection().Repository
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(message string) {
		t.Helper()
		testkit.Git(t, root, "add", "--all")
		testkit.Git(t, root, "commit", "--quiet", "--no-verify", "--message", message)
	}
	write(".markitect/model/a/nested/thing.yaml", "imports: [{package: markitect/core, as: Core}]\nid: Thing\nkind: Core.Kind\nbody: Preserve the nested fixture obligation.\nspec: {}\n")
	commit("nested accounting fixture")
	repo, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	selection := spec.Prepared.Selection()
	service := project.NewPreparationService(project.NewModelService(repo, repo))
	capture := func() project.CapturedPreparation {
		t.Helper()
		result, err := service.Capture(context.Background(), project.CaptureRequest{Selection: selection}, model.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	initial := capture()
	a, ok := initial.Prepared()
	if !ok {
		t.Fatalf("initial preparation: %s", initial.Status())
	}
	p, err := project.Prepare(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ manager, shard, artifact string }{
		{".", ".markitect/registry/registry.yaml", "realization/root.go"},
		{"a/nested", ".markitect/registry/a/nested/registry.yaml", "realization/a/nested.go"},
	} {
		t.Run(tc.manager, func(t *testing.T) {
			manager, ok := p.Topology().Manager(tc.manager)
			if !ok {
				t.Fatal("checked Manager missing")
			}
			doc, err := harness.Build(p, manager, "implement", nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(doc.Developer, "Your registry shard is "+tc.shard+".") || !strings.Contains(doc.Developer, "Registry shards under .markitect/registry/ are writable realization accounting") {
				t.Fatal("Implement omits owned writable accounting")
			}
			_, yaml, found := strings.Cut(doc.Developer, "```yaml\n")
			if !found {
				t.Fatal("Implement omits registry shape")
			}
			yaml, _, found = strings.Cut(yaml, "```\n")
			if !found {
				t.Fatal("registry shape is incomplete")
			}
			yaml = strings.NewReplacer("<exact repository-relative file path>", tc.artifact, "<existing Package/ID>", "a/nested/Thing", "<reason this artifact realizes those Definitions>", "Realizes the fixture obligation.").Replace(yaml)
			write(tc.artifact, "package fixture\n")
			commit("unregistered artifact")
			if got := capture(); got.Status() != model.Failed {
				t.Fatalf("unregistered artifact accepted: %s", got.Status())
			} else {
				attempts := got.Attempts()
				accounting, available := attempts[len(attempts)-1].Files().Accounting()
				unexplained := false
				for _, finding := range accounting.Findings() {
					unexplained = unexplained || finding.Code == "file.unexplained" && finding.Path.String() == tc.artifact
				}
				if !available || !unexplained {
					t.Fatalf("missing actual unexplained artifact finding: %+v", accounting.Findings())
				}
			}
			write(tc.shard, yaml)
			commit("register artifact using delivered example")
			if got := capture(); got.Status() != model.Passed {
				t.Fatalf("delivered registry example rejected: %s %+v", got.Status(), got.Attempts())
			}
			for _, mandate := range []string{"assessment", "verify"} {
				review, err := harness.Build(p, manager, mandate, nil)
				if err != nil || strings.Contains(review.Developer, "Artifact accounting for Implement") {
					t.Fatalf("review mandate received repair instructions: %s %v", mandate, err)
				}
			}
		})
	}
}
