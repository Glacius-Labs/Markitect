package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"io"
	"io/fs"
	"math/rand"
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

	// failures fails the leading turns with these codexErrorInfo values; the
	// flags set what the failing turn did before its error.
	failures                       []string
	failAfterTool, failed, unended bool
	failRunning, pendingFailure    string
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
		if len(f.failures) > 0 {
			info := f.failures[0]
			f.failures = f.failures[1:]
			f.failTurn(info)
			break
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

// failTurn plays a turn that Codex fails with a provider error it will not
// retry itself, after the actions the flags select.
func (f *fakeRuntime) failTurn(info string) {
	item := func(method, kind, status string) string {
		return `{"method":"` + method + `","params":{"threadId":"` + string(codecThreadID) + `","turnId":"` + string(codecTurnID) + `","item":{"id":"item-1","type":"` + kind + `","status":"` + status + `"}}}`
	}
	switch {
	case f.failAfterTool:
		f.pendingFailure = info
		f.tools <- frame.ToolCall{ID: 9, Name: "check", Arguments: json.RawMessage(`{}`), Meta: json.RawMessage(`{"threadId":"` + string(codecThreadID) + `"}`)}
		return
	case f.failRunning != "":
		f.emit(item("item/started", f.failRunning, "inProgress"))
	case f.failed:
		f.emit(item("item/started", "commandExecution", "inProgress"))
		f.emit(item("item/completed", "commandExecution", "failed"))
	}
	f.fail(info)
}
func (f *fakeRuntime) fail(info string) {
	reported := `{"message":"Selected model is at capacity.","codexErrorInfo":` + info + `,"additionalDetails":null}`
	f.emit(`{"method":"error","params":{"threadId":"` + string(codecThreadID) + `","turnId":"` + string(codecTurnID) + `","error":` + reported + `,"willRetry":false}}`)
	if !f.unended {
		f.emit(`{"method":"turn/completed","params":{"threadId":"` + string(codecThreadID) + `","turn":{"id":"` + string(codecTurnID) + `","status":"failed","items":[],"error":` + reported + `}}}`)
	}
}
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.mu.Lock()
	f.replies = append(f.replies, r)
	f.mu.Unlock()
	if info := f.pendingFailure; info != "" {
		f.pendingFailure = ""
		f.fail(info)
		return nil
	}
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

// testOrigin is the bound input of harness documents built outside a Run.
var testOrigin = run.Origin{Input: project.CandidateRef{Commit: "input-commit", Tree: "input-tree"}}

func testSessionSpec(t *testing.T) run.SessionSpec {
	t.Helper()
	r := testkit.NewRepo(t)
	core := "../../../testdata/core"
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
	r.Write(".markitect/model.yaml", "formatVersion: 1\nid: native-fixture\nroot: model\ndependencies: {markitect-core: '1.1.0'}\n")
	r.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.1.0', digest: 'sha256:9552a7cdd560d8822854c286cd74d7dbb5d4277ee1fd16b25ef275213868e043'}\n")
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
	}, Review: func(context.Context, string, string, run.Decision, []run.Finding) error { return nil }, Ask: func(context.Context, string, string) (string, error) { return "new-id", nil }, Reply: func(context.Context, string, string) error { return nil }, ReadNotes: func(context.Context, string) (run.NotePage, error) { return run.NotePage{}, nil }, AppendNote: func(context.Context, string, string) (run.Note, error) { return run.Note{ID: "note"}, nil }, Judge: func(context.Context, run.Decision, []run.Finding) error { return nil }, Check: func(context.Context) (run.Check, error) { return run.Check{Valid: true}, nil }}
	return run.SessionSpec{RunID: "native-run", Manager: manager, Prepared: p, Mandate: run.Implement, Origin: testOrigin, Workspace: &sessionCopy{location: run.CopyLocation{Volume: "private-run-volume", Subpath: "copies/root"}}, Tools: tools}
}

// Exercise the real Git review producer and native Event/MCP/frame consumer.
// No Engine, provider or native Session is started by this controlled test.
func TestBinaryChildReviewFitsActualAwaitToolFrame(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	spec := testSessionSpec(t)
	git := &gitrepo.LocalGit{StateRoot: testkit.TempDir(t), TempRoot: testkit.TempDir(t)}
	input, err := git.Capture(ctx, "native-fixture", spec.Prepared)
	if err != nil {
		t.Fatal(err)
	}
	service, err := git.Open(ctx, run.Request{ID: "binary-review", Project: input.Project, Prepared: spec.Prepared, Input: input})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := service.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	base := run.Commit{ID: input.Commit, Tree: input.Tree}
	parent, err := service.Copy(ctx, ".", base)
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Copy(ctx, "a", base)
	if err != nil {
		t.Fatal(err)
	}
	location := child.Location().Directory
	binary := make([]byte, 2<<20)
	_, _ = rand.New(rand.NewSource(1)).Read(binary)
	binary[0] = 0 // Git must classify this as binary, not a large text diff.
	if err := os.WriteFile(filepath.Join(location, "review.bin"), binary, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(location, "guide.md"), []byte("Complete <&> review text.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	testkit.Git(t, location, "add", "review.bin")
	testkit.Git(t, location, "update-index", "--chmod=+x", "review.bin")
	fixed, err := child.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := parent.Review(ctx, child, base, fixed)
	if err != nil {
		t.Fatal(err)
	}
	handoff := run.Handoff{Manager: "a", Base: base, Result: fixed, Diff: diff, Decision: run.Approve, Findings: []run.Finding{{Manager: "a", Text: "Actual fixed child reviewed."}}}
	event := run.Event{Handoff: &handoff}
	spec.Tools.Await = func(context.Context) (run.Event, error) { return event, nil }
	rt := newRuntime()
	defer rt.Close()
	native := &nativeSession{thread: "review-thread", spec: spec, peer: rt}
	if err := native.answer(ctx, frame.ToolCall{ID: 3, Name: "await", Arguments: json.RawMessage(`{}`), Meta: json.RawMessage(`{"threadId":"review-thread"}`)}); err != nil {
		t.Fatal(err)
	}
	if len(rt.replies) != 1 {
		t.Fatal("expected one actual native tool response")
	}
	payload, err := frame.EncodeToolReply(rt.replies[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := frame.Write(io.Discard, frame.Frame{Channel: frame.Tool, Payload: payload}); err != nil {
		t.Fatalf("actual Await Event/MCP reply is %d bytes: %v", len(payload), err)
	}
	t.Logf("actual review %d bytes; encoded Await Event/MCP frame %d bytes", len(diff), len(payload))
	var content struct {
		Content []struct{ Text string }
		IsError bool
	}
	var decoded run.Event
	if err := json.Unmarshal(rt.replies[0].Result, &content); err != nil || content.IsError || len(content.Content) != 1 {
		t.Fatal("invalid actual MCP result", err)
	}
	if err := json.Unmarshal([]byte(content.Content[0].Text), &decoded); err != nil || !reflect.DeepEqual(decoded, event) {
		t.Fatal("Event serialization lost exact handoff metadata or review context", err)
	}
	blob := testkit.Git(t, location, "rev-parse", fixed.ID+":review.bin")
	for _, required := range []string{"diff --git a/review.bin b/review.bin", "new file mode 100755", strings.Repeat("0", len(blob)) + ".." + blob, "Binary files /dev/null and b/review.bin differ"} {
		if !strings.Contains(diff, required) {
			t.Fatal("review omitted binary path, mode or complete object identity", required)
		}
	}
	if strings.Contains(diff, "GIT binary patch") {
		t.Fatal("review context transported opaque binary patch data")
	}
	// Compare every text hunk with the old binary-patch mode using full IDs;
	// abbreviated legacy index lines are intentionally replaced by full IDs.
	text := testkit.Git(t, parent.Location().Directory, "diff", "--no-ext-diff", "--no-textconv", "--binary", "--full-index", base.ID, fixed.ID, "--", "guide.md")
	if !strings.Contains(diff, text) {
		t.Fatal("complete text diff changed or was omitted")
	}
	if actual, err := parent.Snapshot(ctx); err != nil || actual != base {
		t.Fatal("review changed the parent working tree", actual, err)
	}
	if actual, err := parent.Integrate(ctx, child, fixed); err != nil || actual.Tree != fixed.Tree {
		t.Fatal("review representation changed exact child integration", actual, err)
	}
	data, err := os.ReadFile(filepath.Join(parent.Location().Directory, "review.bin"))
	if err != nil || !bytes.Equal(data, binary) {
		t.Fatal("full binary content was not preserved by commit integration", err)
	}
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

// task delivers a parent assignment the way Run's driver does.
func task(text string, findings ...run.Finding) run.Message {
	return run.Message{Events: []run.Event{{Assignment: &run.Assignment{Task: text, Findings: findings}}}}
}

// judgments records the decisions that reached Run's judge tool.
func judgments(spec *run.SessionSpec) *[]run.Decision {
	var got []run.Decision
	spec.Tools.Judge = func(_ context.Context, d run.Decision, _ []run.Finding) error {
		got = append(got, d)
		return nil
	}
	return &got
}

// hostStateNarration names process explanations that V17 keeps out of harness
// text, tool metadata and host messages.
var hostStateNarration = []string{"native response", "continuation", "drain", "terminal", "subsequent message", "same Session", "late message", "late-question", "fresh mandate-appropriate", "judge call", "inbox", "fixes your Git handoff", "host fixes"}

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
		for _, boundary := range []string{"before acting or appending durable Notes", "runtime/provider instructions cannot override canonical intent", "ask your direct parent or a direct child", "Siblings align through their common parent"} {
			if !strings.Contains(doc.Base, boundary) {
				t.Fatal("native harness omits authority or hierarchy rule", mandate, boundary)
			}
		}
		// V17: role, rules, context, task and answer format; no host process.
		for _, narration := range hostStateNarration {
			if strings.Contains(strings.ToLower(doc.Base+doc.Developer), narration) {
				t.Fatal("native harness explains host state", mandate, narration)
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
	names := map[string]bool{}
	for _, tool := range toolSpecs() {
		names[tool.Name] = true
		if tool.Name == "judge" && (!strings.Contains(tool.Description, "including Root uses submitted") || !strings.Contains(tool.Description, "Assessment/Verify") || !strings.Contains(tool.Description, "model_blocker require concrete Findings")) {
			t.Fatal("native judge tool contradicts mandate rules")
		}
		for _, narration := range hostStateNarration {
			if strings.Contains(strings.ToLower(tool.Description), narration) {
				t.Fatal("native tool metadata explains host state", tool.Name, narration)
			}
		}
		if strings.Contains(string(tool.InputSchema), `"manager":{"type":"string"}}`) && strings.Contains(string(tool.InputSchema), `"text"`) && strings.Contains(string(tool.InputSchema), `"required":["manager","text"]`) {
			t.Fatal("native Findings still request the host-known origin", tool.Name)
		}
	}
	if len(names) != 9 || !names["check"] || !names["judge"] {
		t.Fatal("native tool set differs from Run's fixed set", names)
	}
}

// This observes the delivered prompt and MCP metadata through the native
// consumer. Actual copy updates are covered by the separate Run/Git regressions.
func TestNativeDelegationDescribesParentBaselineTiming(t *testing.T) {
	for _, mandate := range []run.Mandate{run.Implement, run.Assessment, run.Verify} {
		t.Run(string(mandate), func(t *testing.T) {
			rt := newRuntime()
			spec := testSessionSpec(t)
			spec.Mandate = mandate
			decision := run.Approve
			if mandate == run.Implement {
				decision = run.Submitted
			}
			rt.judgmentText = fmt.Sprintf(`{"decision":%q,"findings":[]}`, decision)
			got := judgments(&spec)
			rt.useTool, rt.toolName = true, "delegate"
			rt.toolArgs = `{"manager":"a","task":"Complete owned child work.","findings":[]}`
			calls := 0
			var child, delegated string
			var findings []run.Finding
			spec.Tools.Delegate = func(_ context.Context, manager, assignment string, feedback []run.Finding) error {
				calls++
				child, delegated, findings = manager, assignment, feedback
				return nil
			}
			s := openSession(t, rt, spec)
			if err := s.Exchange(context.Background(), task("Complete the bound mandate.")); err != nil || !reflect.DeepEqual(*got, []run.Decision{decision}) {
				t.Fatal(*got, err)
			}
			if calls != 1 || child != "a" || delegated != "Complete owned child work." || len(findings) != 0 {
				t.Fatal("native delegation callback changed", calls, child, delegated, findings)
			}
			starts := 0
			for _, raw := range rt.writes {
				var message struct {
					Method string
					Params struct{ BaseInstructions string }
				}
				if err := json.Unmarshal([]byte(raw), &message); err != nil {
					t.Fatal(err)
				}
				if message.Method != "thread/start" {
					continue
				}
				starts++
				for _, boundary := range []string{"For Implement, a delegation gives the child your current working state as its baseline", "within your ownership", "reach a child only through a correction", "A correction after your review needs concrete Findings", "merges your current baseline into the child's existing work", "Assessment and Verify keep their fixed input candidate"} {
					if !strings.Contains(message.Params.BaseInstructions, boundary) {
						t.Fatal("delivered native Rules omit copy boundary", boundary)
					}
				}
			}
			delegates, reviews := 0, 0
			for _, tool := range rt.start.Tools {
				if tool.Name == "review" {
					reviews++
					for _, boundary := range []string{"by its commit", "approve accepts it and integrates an Implement submission", "reject returns it with Findings", "model_blocker escalates a material model defect"} {
						if !strings.Contains(tool.Description, boundary) {
							t.Fatal("delivered native review metadata misstates handoff handling", boundary)
						}
					}
				}
				if tool.Name != "delegate" {
					continue
				}
				delegates++
				for _, boundary := range []string{"direct child", "After you reviewed its fixed handoff, a correction needs concrete Findings", "For Implement, the child works from your current working state"} {
					if !strings.Contains(tool.Description, boundary) {
						t.Fatal("delivered native delegate metadata omits copy boundary", boundary)
					}
				}
			}
			if starts != 1 || delegates != 1 || reviews != 1 {
				t.Fatal("native prompt or handoff tool metadata not delivered", starts, delegates, reviews)
			}
		})
	}
}

// Observe preservation and legitimate repair duties at the actual thread/start
// boundary. The scripted provider proves delivery, not semantic obedience.
func TestNativeImplementReconcilesBeforeChangingArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mandate  run.Mandate
		findings []run.Finding
	}{
		{name: "implement-without-initial-findings", mandate: run.Implement},
		{name: "implement-with-required-test-gap", mandate: run.Implement, findings: []run.Finding{{Manager: ".", Text: "Required refusal coverage is missing."}}},
		{name: "assessment", mandate: run.Assessment},
		{name: "verify", mandate: run.Verify},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRuntime()
			spec := testSessionSpec(t)
			spec.Mandate, spec.Findings = tc.mandate, tc.findings
			decision := run.Approve
			if tc.mandate == run.Implement {
				decision = run.Submitted
			}
			rt.judgmentText = fmt.Sprintf(`{"decision":%q,"findings":[]}`, decision)
			got := judgments(&spec)
			s := openSession(t, rt, spec)
			if err := s.Exchange(context.Background(), task("Complete the bound mandate.", tc.findings...)); err != nil || !reflect.DeepEqual(*got, []run.Decision{decision}) {
				t.Fatal("mandate judgment transport changed", *got, err)
			}
			starts := 0
			for _, raw := range rt.writes {
				var message struct {
					Method string
					Params struct{ DeveloperInstructions string }
				}
				if err := json.Unmarshal([]byte(raw), &message); err != nil {
					t.Fatal(err)
				}
				if message.Method != "thread/start" {
					continue
				}
				starts++
				instructions := message.Params.DeveloperInstructions
				_, tail, ok := strings.Cut(instructions, "Initial Findings:\n")
				if !ok {
					t.Fatal("actual Findings were not delivered")
				}
				line, _, _ := strings.Cut(tail, "\n")
				var delivered []run.Finding
				if err := json.Unmarshal([]byte(line), &delivered); err != nil || len(delivered) != len(tc.findings) {
					t.Fatal("delivered Findings changed", delivered, err)
				}
				for i := range delivered {
					if delivered[i] != tc.findings[i] {
						t.Fatal("actual required repair Finding was lost", delivered)
					}
				}
				if !strings.Contains(instructions, "Every mandate uses the full hierarchy") {
					t.Fatal("preservation skipped the required hierarchy")
				}
				if tc.mandate == run.Implement {
					for _, duty := range []string{
						"first reconcile the existing realization",
						"checked model's obligations, required checks and actual Findings",
						"Preserve conforming artifacts, including code, tests, documentation and registry entries",
						"optional coverage, refactoring or cosmetic changes solely to produce a diff",
						"Repair genuine defects and missing required tests you discover",
						"when needed for those modeled obligations, required checks or actual Findings",
						"Assessment approval does not replace this reconciliation",
						"complete all modeled delegation, child reviews and required checks, then submit the unchanged candidate",
					} {
						if !strings.Contains(instructions, duty) {
							t.Fatal("delivered Implement instructions omit reconciliation duty", duty)
						}
					}
				} else if strings.Contains(instructions, "first reconcile the existing realization") || strings.Contains(instructions, "then submit the unchanged candidate") {
					t.Fatal("review mandate received Implement duties", tc.mandate)
				}
			}
			if starts != 1 {
				t.Fatal("native instructions were not delivered exactly once", starts)
			}
		})
	}
}

func TestSessionUsesHarnessCopySubpathAndPersistentNativeThread(t *testing.T) {
	rt := newRuntime()
	spec := testSessionSpec(t)
	got := judgments(&spec)
	s := openSession(t, rt, spec)
	if rt.starts != 1 || len(rt.writes) != 2 {
		t.Fatal("Open eagerly started inference", rt.starts, rt.writes)
	}
	if got := rt.spec.Mounts[0]; got != (container.Mount{Volume: "private-run-volume", Subpath: "copies/root", Target: "/work", NoCopy: true}) {
		t.Fatal("copy projection", got)
	}
	if rt.spec.Mounts[1].Target != "/home/agent" || rt.spec.Mounts[1].Volume == rt.spec.Mounts[0].Volume {
		t.Fatal("home aliases workspace")
	}
	for _, text := range []string{"first task", "recall first task"} {
		if err := s.Exchange(context.Background(), task(text)); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(*got, []run.Decision{run.Submitted, run.Submitted}) {
		t.Fatal("judge did not reach Run's tool", *got)
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
	go func() { done <- s.Exchange(context.Background(), task("wait")) }()
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
	for _, kind := range []string{"provider", "secret"} {
		t.Run(kind, func(t *testing.T) {
			rt := newRuntime()
			switch kind {
			case "provider":
				rt.terminalStatus = "failed"
			case "secret":
				rt.errorText = "private-access"
			}
			s := openSession(t, rt, testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
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
			err := s.Exchange(context.Background(), task("coordinate"))
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
		done <- s.Exchange(context.Background(), task("complete while background writer remains"))
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
			if err := s.Exchange(context.Background(), task("complete")); err == nil || !rt.closed {
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
	rt.judgmentText = `{"decision":"submitted","findings":[{"text":"private-access private-refresh"}]}`
	rt.finalText = "Independent final prose private-access"
	a := agentsFor(t, rt)
	a.TranscriptRoot = t.TempDir()
	spec := testSessionSpec(t)
	var findings []run.Finding
	spec.Tools.Judge = func(_ context.Context, _ run.Decision, f []run.Finding) error { findings = f; return nil }
	session, err := a.Open(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err = session.Exchange(context.Background(), task("work")); err != nil || len(findings) != 1 || findings[0] != (run.Finding{Text: "[REDACTED] [REDACTED]"}) {
		t.Fatal(findings, err)
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

// One persistent Session receives an assignment, a question-only message and a
// host diagnostic. The judgment is a Run tool call; a message that only
// handles a question ends without one and is not a failure.
func TestNativeMessagesCarryEventsWithoutDemandingAJudgment(t *testing.T) {
	for _, mandate := range []run.Mandate{run.Implement, run.Assessment, run.Verify} {
		t.Run(string(mandate), func(t *testing.T) {
			rt := newRuntime()
			spec := testSessionSpec(t)
			spec.Mandate = mandate
			decision := run.Approve
			if mandate == run.Implement {
				decision = run.Submitted
			}
			got := judgments(&spec)
			var replies []string
			spec.Tools.Reply = func(_ context.Context, id, _ string) error { replies = append(replies, id); return nil }
			rt.judgmentText = fmt.Sprintf(`{"decision":%q,"findings":[{"text":"initial work"}]}`, decision)
			s := openSession(t, rt, spec)
			if err := s.Exchange(context.Background(), task("Complete your assignment.")); err != nil || !reflect.DeepEqual(*got, []run.Decision{decision}) {
				t.Fatal(*got, err)
			}
			question := run.Message{Events: []run.Event{{Question: &run.Question{ID: "q-1", From: "a", To: ".", Text: "Explain your fixed handoff."}}}}
			rt.autoJudge = false
			rt.useTool, rt.toolName, rt.toolArgs = true, "reply", `{"id":"q-1","text":"It realizes the obligation."}`
			if err := s.Exchange(context.Background(), question); err != nil || len(*got) != 1 || !reflect.DeepEqual(replies, []string{"q-1"}) {
				t.Fatal("a question-only message demanded or produced a judgment", *got, replies, err)
			}
			rt.useTool, rt.autoJudge = false, true
			diagnostic := run.Message{Feedback: []run.Finding{{Text: "your working state changed after your judgment: a became b"}}}
			if err := s.Exchange(context.Background(), diagnostic); err != nil || len(*got) != 2 {
				t.Fatal("a renewed judgment did not reach Run", *got, err)
			}
			starts, turns := 0, 0
			for _, raw := range rt.writes {
				var message struct {
					Method string
					Params struct {
						Input []struct{ Text string }
					}
				}
				if err := json.Unmarshal([]byte(raw), &message); err != nil {
					t.Fatal(err)
				}
				if message.Method == "thread/start" {
					starts++
				}
				if message.Method != "turn/start" {
					continue
				}
				turns++
				var delivered run.Message
				if len(message.Params.Input) != 1 || json.Unmarshal([]byte(message.Params.Input[0].Text), &delivered) != nil {
					t.Fatal("native message is not the Run message")
				}
				want := []run.Message{task("Complete your assignment."), question, diagnostic}[turns-1]
				if !reflect.DeepEqual(delivered, want) {
					t.Fatal("native message changed", delivered, want)
				}
			}
			if starts != 1 || turns != 3 {
				t.Fatal("messages did not use the same native Session", starts, turns)
			}
		})
	}
}

// Every completed native item but text is activity of the Exchange that
// observed it, whatever its outcome and also on a native subagent thread.
// Messages and reasoning are text; every Exchange starts without activity.
func TestExchangeReportsObservedNativeExecutions(t *testing.T) {
	rt := newRuntime()
	rt.autoJudge = false
	var session run.Session = openSession(t, rt, testSessionSpec(t))
	acting, ok := session.(run.ActingSession)
	if !ok {
		t.Fatal("native Session does not report native executions")
	}
	item := func(thread ThreadID, turn TurnID, value map[string]any) string {
		data, _ := json.Marshal(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": thread, "turnId": turn, "item": value}})
		return string(data)
	}
	subagent := `{"method":"item/started","params":{"threadId":"` + string(codecThreadID) + `","turnId":"` + string(codecTurnID) + `","item":{"id":"spawn","type":"subAgentActivity","kind":"started","agentThreadId":"child-thread"}}}`
	for _, tc := range []struct {
		name          string
		notifications []string
		items         []map[string]any
		acted         bool
	}{
		{"text only", nil, []map[string]any{{"id": "u", "type": "userMessage"}, {"id": "r", "type": "reasoning"}, {"id": "a", "type": "agentMessage", "phase": "commentary", "text": "Reading the input."}}, false},
		{"failed command", nil, []map[string]any{{"id": "c", "type": "commandExecution", "command": "go test ./...", "status": "failed", "exitCode": 1}}, true},
		{"file change", nil, []map[string]any{{"id": "f", "type": "fileChange", "status": "completed"}}, true},
		{"tool call", nil, []map[string]any{{"id": "m", "type": "mcpToolCall", "server": "markitect", "tool": "check", "status": "completed"}}, true},
		{"web search", nil, []map[string]any{{"id": "w", "type": "webSearch", "query": "Go module proxy"}}, true},
		{"subagent command", []string{subagent, item("child-thread", "child-turn", map[string]any{"id": "s", "type": "commandExecution", "status": "completed"})}, nil, true},
		{"next Exchange", nil, nil, false},
	} {
		rt.notifications, rt.commandItems = tc.notifications, tc.items
		if err := session.Exchange(context.Background(), task("Review the fixed input.")); err != nil || acting.Acted() != tc.acted {
			t.Fatal(tc.name, acting.Acted(), err)
		}
	}
}

// Invalid judge input is an ordinary tool refusal: the provider receives the
// reason, Run receives no judgment and the Session stays usable.
func TestInvalidJudgeArgumentsAreRefusedWithoutJudgment(t *testing.T) {
	for _, raw := range []string{`{"decision":"submitted"}`, `{"decision":"submitted","findings":null}`, `{"decision":"submitted","findings":[{}]}`, `{"decision":"submitted","findings":[{"text":null}]}`, `{"decision":"submitted","findings":[{"manager":".","text":"origin is host-known"}]}`, `{"decision":"submitted","findings":[{"text":"work","definition":null}]}`, `{"decision":"submitted","findings":[{"definition":"a/Thing"}]}`, `{"Decision":"submitted","findings":[]}`, `{"decision":"submitted","findings":[],"commit":"invented"}`, `{"decision":"success","findings":[]}`} {
		t.Run(raw, func(t *testing.T) {
			rt := newRuntime()
			rt.judgmentText = raw
			spec := testSessionSpec(t)
			got := judgments(&spec)
			s := openSession(t, rt, spec)
			if err := s.Exchange(context.Background(), task("work")); err != nil || len(*got) != 0 || rt.closed {
				t.Fatal("invalid judgment reached Run or closed the Session", *got, err)
			}
			var content struct{ IsError bool }
			if len(rt.replies) != 1 || json.Unmarshal(rt.replies[0].Result, &content) != nil || !content.IsError || !strings.Contains(string(rt.replies[0].Result), "invalid judge arguments") {
				t.Fatal("refusal did not name its reason", rt.replies)
			}
		})
	}
}

func TestMissingRequiredToolFieldsCauseNoCallbackOrDurableWrite(t *testing.T) {
	calls := 0
	tools := run.Tools{AppendNote: func(context.Context, string, string) (run.Note, error) { calls++; return run.Note{}, nil }, Delegate: func(context.Context, string, string, []run.Finding) error { calls++; return nil }, Judge: func(context.Context, run.Decision, []run.Finding) error { calls++; return nil }, Check: func(context.Context) (run.Check, error) { calls++; return run.Check{}, nil }}
	for _, tc := range []struct{ name, raw string }{{"append_note", `{"text":"note"}`}, {"append_note", `{"Context":"context","text":"note"}`}, {"append_note", `{"context":null,"text":"note"}`}, {"delegate", `{"manager":"a","task":"work"}`}, {"delegate", `{"manager":"a","task":"work","findings":null}`}, {"delegate", `{"manager":"a","task":"work","findings":[{"manager":".","text":"origin is host-known"}]}`}, {"judge", `{"decision":"submitted"}`}, {"judge", `{"decision":"success","findings":[]}`}, {"judge", `{"decision":"submitted","findings":[],"commit":"invented"}`}, {"check", `{"manager":"a"}`}} {
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
	spec := testSessionSpec(t)
	got := judgments(&spec)
	s := openSession(t, rt, spec)
	err := s.Exchange(context.Background(), task("work"))
	if err == nil || !rt.closed || !strings.Contains(err.Error(), "drain failed") || strings.Contains(err.Error(), "private-access") {
		t.Fatal("recorded judgment bypassed native failure", err, rt.closed)
	}
	// Run never fixes a handoff from an Exchange that failed.
	if !reflect.DeepEqual(*got, []run.Decision{run.Submitted}) {
		t.Fatal("regression never reached a recorded judgment", *got)
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
			doc, err := harness.Build(p, manager, "implement", nil, harness.Origin{Commit: "input-commit", Tree: "input-tree"})
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
				review, err := harness.Build(p, manager, mandate, nil, harness.Origin{Commit: "input-commit", Tree: "input-tree"})
				if err != nil || strings.Contains(review.Developer, "Artifact accounting for Implement") {
					t.Fatalf("review mandate received repair instructions: %s %v", mandate, err)
				}
			}
		})
	}
}

// The harness states the bound input to every mandate, the host fact only when
// it holds, each Initial Finding's Definition and Verify's tiers.
func TestHarnessStatesBoundInputVerifiedFactAndFindingDefinitions(t *testing.T) {
	spec := testSessionSpec(t)
	origin := harness.Origin{Commit: "input-commit", Tree: "input-tree"}
	findings := []harness.Finding{{Manager: ".", Definition: "a/Thing", Text: "repair the defect"}}
	for _, mandate := range []string{"assessment", "implement", "verify"} {
		doc, err := harness.Build(spec.Prepared, spec.Manager, mandate, findings, origin)
		if err != nil || !strings.Contains(doc.Developer, `{"commit":"input-commit","tree":"input-tree"}`) || !strings.Contains(doc.Developer, `"definition":"a/Thing"`) || strings.Contains(doc.Developer, "Host fact") {
			t.Fatal(mandate, "bound input, Definition or absent fact wrong", err)
		}
		verified := origin
		verified.Verified = "earlier-verify"
		if doc, err = harness.Build(spec.Prepared, spec.Manager, mandate, findings, verified); err != nil || !strings.Contains(doc.Developer, "Host fact: the stored Verify Run earlier-verify already approved exactly this bound input") {
			t.Fatal(mandate, "host fact missing", err)
		}
	}
	doc, err := harness.Build(spec.Prepared, spec.Manager, "verify", nil, origin)
	if err != nil || !strings.Contains(doc.Developer, "When the host states that the bound input was already verified and you find no defect, any change is a reject") || !strings.Contains(doc.Developer, "Every Finding of a reject names the affected Definition") {
		t.Fatal("Verify tiers missing", err)
	}
	if _, err = harness.Build(spec.Prepared, spec.Manager, "verify", nil, harness.Origin{}); err == nil {
		t.Fatal("harness built without its bound input")
	}
}

// A Finding's optional definition reaches Run unchanged; its origin stays host-known.
func TestFindingDefinitionReachesRun(t *testing.T) {
	var got []run.Finding
	tools := run.Tools{Judge: func(_ context.Context, _ run.Decision, f []run.Finding) error { got = f; return nil }}
	raw := `{"decision":"reject","findings":[{"text":"revert the change","definition":"a/Thing"},{"text":"an observation"}]}`
	if _, err := dispatch(context.Background(), tools, "judge", json.RawMessage(raw)); err != nil || !reflect.DeepEqual(got, []run.Finding{{Definition: "a/Thing", Text: "revert the change"}, {Text: "an observation"}}) {
		t.Fatal(got, err)
	}
}

// Notes whose total exceeds one frame are read page by page through the actual
// native tool path; every encoded reply fits the transport frame, even for
// bytes that need the longest JSON escapes.
func TestReadNotesPagesFitTheToolTransportFrame(t *testing.T) {
	ctx := context.Background()
	store, err := runstore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const count = 120
	for i := range count {
		text := strings.Repeat("\x01", run.MaxNoteText)
		if i%2 == 0 {
			text = strings.Repeat(`"\`, run.MaxNoteText/2)
		}
		if _, err = store.Append(ctx, run.Note{Project: "p", Run: "r", Manager: "m", Context: "c", Text: text}); err != nil {
			t.Fatal(err)
		}
	}
	tools := run.Tools{ReadNotes: func(c context.Context, after string) (run.NotePage, error) { return store.Notes(c, "p", after) }}
	seen, pages := 0, 0
	for after := ""; ; pages++ {
		raw, _ := json.Marshal(map[string]string{"after": after})
		if after == "" {
			raw = []byte(`{}`)
		}
		if err = validateArguments("read_notes", raw); err != nil {
			t.Fatal(err)
		}
		text, err := dispatch(ctx, tools, "read_notes", raw)
		if err != nil {
			t.Fatal(err)
		}
		reply, err := frame.EncodeToolReply(frame.ToolReply{ID: 1, Result: toolResult(text, false)})
		if err != nil || len(reply) > frame.MaxPayload {
			t.Fatal("reply does not fit the frame", len(reply), err)
		}
		var page run.NotePage
		if err = json.Unmarshal([]byte(text), &page); err != nil || len(page.Notes) == 0 {
			t.Fatal("page not decodable or empty", err)
		}
		seen += len(page.Notes)
		if page.Next == "" {
			break
		}
		after = page.Next
	}
	if seen != count || pages < 2 {
		t.Fatal("not every Note was read", seen, pages)
	}
}

func TestReadNotesAcceptsOnlyAnOptionalCursorString(t *testing.T) {
	for _, ok := range []string{`{}`, `{"after":""}`, `{"after":"id"}`} {
		if err := validateArguments("read_notes", []byte(ok)); err != nil {
			t.Fatal(ok, err)
		}
	}
	for _, bad := range []string{`{"after":1}`, `{"after":null}`, `{"cursor":"x"}`, `{"after":"x","extra":"y"}`, `[]`} {
		if err := validateArguments("read_notes", []byte(bad)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
	for _, name := range []string{"await", "check"} {
		if err := validateArguments(name, []byte(`{"after":"x"}`)); err == nil {
			t.Fatal(name, "accepted an argument")
		}
	}
}
