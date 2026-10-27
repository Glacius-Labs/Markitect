package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// fakeStream delivers one provider process's standard output and ends at its
// exit, like the container session's per-process output buffer.
type fakeStream struct {
	chunks  chan []byte
	pending []byte
	done    chan struct{}
	once    sync.Once
}

func newStream() *fakeStream {
	return &fakeStream{chunks: make(chan []byte, 1024), done: make(chan struct{})}
}

func (f *fakeStream) Read(p []byte) (int, error) {
	if len(f.pending) == 0 {
		select {
		case f.pending = <-f.chunks:
		case <-f.done:
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

// fakeProcess is one provider process the fake CLI plays.
type fakeProcess struct {
	rt      *fakeRuntime
	start   frame.Start
	input   []byte
	stdout  *fakeStream
	killed  chan struct{}
	exited  bool
	exit    frame.Exited
	session string
}

func (p *fakeProcess) emit(lines ...string) {
	for _, line := range lines {
		p.stdout.chunks <- []byte(line + "\n")
	}
}

// call sends one Run tool call through the MCP bridge and waits for the host's
// reply, or for the process to be stopped.
func (p *fakeProcess) call(name, args string) (frame.ToolReply, bool) {
	p.rt.mu.Lock()
	p.rt.nextCall++
	id := p.rt.nextCall
	reply := make(chan frame.ToolReply, 1)
	p.rt.waiting[id] = reply
	p.rt.mu.Unlock()
	select {
	case p.rt.tools <- frame.ToolCall{ID: id, Name: name, Arguments: json.RawMessage(args), Meta: json.RawMessage(`{"claudecode/toolUseId":"toolu_run"}`)}:
	case <-p.killed:
		return frame.ToolReply{}, false
	}
	select {
	case r := <-reply:
		return r, true
	case <-p.killed:
		return frame.ToolReply{}, false
	}
}

// runTool plays one Run tool use in the stream around the MCP call.
func (p *fakeProcess) runTool(id, name, args string) bool {
	p.emit(fixtureAssistant(p.session, toolUse(id, "mcp__markitect__"+name)))
	if _, ok := p.call(name, args); !ok {
		return false
	}
	p.emit(fixtureToolResult(p.session, id))
	return true
}

func (p *fakeProcess) end(code int) {
	p.rt.mu.Lock()
	p.exited, p.exit = true, frame.Exited{Code: code}
	p.rt.mu.Unlock()
	p.stdout.close()
}

// fakeRuntime plays agentd and the Claude Code CLI behind the container
// session surface. play scripts each response process after its input closed.
type fakeRuntime struct {
	mu         sync.Mutex
	spec       container.ContainerSpec
	setup      *frame.Initialize
	calls      []string
	processes  []*fakeProcess
	current    *fakeProcess
	tools      chan frame.ToolCall
	waiting    map[uint64]chan frame.ToolReply
	nextCall   uint64
	replies    []frame.ToolReply
	census     [][]frame.Process
	authStatus string
	play       func(*fakeProcess)
	closed     bool
	stopErr    error
	sessionIDs []string
}

func newRuntime() *fakeRuntime {
	return &fakeRuntime{tools: make(chan frame.ToolCall, 16), waiting: map[uint64]chan frame.ToolReply{}, authStatus: `{"loggedIn":true,"authMethod":"claude.ai","email":"someone@example.invalid"}`, play: judged("submitted")}
}

func (f *fakeRuntime) record(call string) {
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
}

func (f *fakeRuntime) ImageID() string { return fixtureImage }
func (f *fakeRuntime) Initialize(_ context.Context, req frame.Initialize) error {
	f.record("initialize")
	f.mu.Lock()
	f.setup = &req
	f.mu.Unlock()
	return nil
}
func (f *fakeRuntime) ClearPrivateHome(context.Context) error {
	f.record("clear")
	return nil
}
func (f *fakeRuntime) StartProvider(_ context.Context, s frame.Start) (frame.Started, error) {
	p := &fakeProcess{rt: f, start: s, stdout: newStream(), killed: make(chan struct{})}
	for i := 0; i+1 < len(s.Argv); i++ {
		if s.Argv[i] == "--session-id" || s.Argv[i] == "--resume" {
			p.session = s.Argv[i+1]
		}
	}
	f.mu.Lock()
	if f.current != nil && !f.current.exited {
		f.mu.Unlock()
		return frame.Started{}, fmt.Errorf("the previous provider has not ended")
	}
	f.processes = append(f.processes, p)
	f.current = p
	f.calls = append(f.calls, "start")
	f.mu.Unlock()
	if slices.Equal(s.Argv, authStatusArgv()) {
		p.emit(f.authStatus)
		p.end(0)
	}
	return frame.Started{PID: 100 + len(f.processes)}, nil
}
func (f *fakeRuntime) Stdout() io.Reader {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current.stdout
}
func (f *fakeRuntime) WriteInput(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current.input = append(f.current.input, data...)
	return nil
}
func (f *fakeRuntime) CloseInput() error {
	f.mu.Lock()
	p := f.current
	f.mu.Unlock()
	if !slices.Equal(p.start.Argv, authStatusArgv()) {
		go f.play(p)
	}
	return nil
}
func (f *fakeRuntime) ToolCalls() <-chan frame.ToolCall { return f.tools }
func (f *fakeRuntime) Reply(r frame.ToolReply) error {
	f.mu.Lock()
	f.replies = append(f.replies, r)
	reply := f.waiting[r.ID]
	delete(f.waiting, r.ID)
	f.mu.Unlock()
	if reply != nil {
		reply <- r
	}
	return nil
}
func (f *fakeRuntime) Stop(context.Context, int64) (frame.Exited, error) {
	f.record("stop")
	if f.stopErr != nil {
		return frame.Exited{}, f.stopErr
	}
	f.mu.Lock()
	p := f.current
	if !p.exited {
		p.exited, p.exit = true, frame.Exited{Code: -1, Signal: "terminated"}
		close(p.killed)
		p.stdout.close()
	}
	f.mu.Unlock()
	return p.exit, nil
}
func (f *fakeRuntime) Census(context.Context) ([]frame.Process, error) {
	f.record("census")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.census) == 0 {
		return nil, nil
	}
	answer := f.census[0]
	if len(f.census) > 1 {
		f.census = f.census[1:]
	}
	return answer, nil
}
func (f *fakeRuntime) Close() error {
	f.record("close")
	f.mu.Lock()
	f.closed = true
	if f.current != nil {
		f.current.stdout.close()
	}
	f.mu.Unlock()
	return nil
}

// responses returns the response processes, without the login check.
func (f *fakeRuntime) responses() []*fakeProcess {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*fakeProcess
	for _, p := range f.processes {
		if !slices.Equal(p.start.Argv, authStatusArgv()) {
			out = append(out, p)
		}
	}
	return out
}

// judged plays a response that judges its assignment through the Run tool
// and ends with a result.
func judged(decision string) func(*fakeProcess) {
	return func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		if !p.runTool("toolu_judge", "judge", `{"decision":"`+decision+`","findings":[]}`) {
			return
		}
		p.emit(fixtureAssistant(p.session, textBlock("Judged.")), fixtureResult(p.session, "Judged."))
		p.end(0)
	}
}

// lines plays a response of fixed output lines and an exit code.
func lines(code int, out ...string) func(*fakeProcess) {
	return func(p *fakeProcess) {
		for _, line := range out {
			p.emit(strings.ReplaceAll(line, "SESSION", p.session))
		}
		p.end(code)
	}
}

const dedicatedCredentials = fixtureCredentials

// testOrigin is the bound input of harness documents built outside a Run.
var testOrigin = run.Origin{Input: project.CandidateRef{Commit: "input-commit", Tree: "input-tree"}}

type sessionCopy struct {
	run.Copy
	location run.CopyLocation
}

func (c *sessionCopy) Location() run.CopyLocation { return c.location }

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
	r.Write(".markitect/model.yaml", "formatVersion: 1\nid: native-fixture\nroot: model\ndependencies: {markitect-core: '1.2.0'}\n")
	r.Write(".markitect/model-lock.yaml", "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.2.0', digest: 'sha256:b338a1d686be91495e893f085d85013cc91f8f740bef8725210adc2e276f000a'}\n")
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

// agentsFor composes the actual adapter over the fake CLI with fast bounds.
func agentsFor(t *testing.T, rt *fakeRuntime, credential RuntimeCredential) *Agents {
	t.Helper()
	p := preparedProfile(t, claudeProfile(filepath.Join(t.TempDir(), "engine")), imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	a, err := NewAgentsWithRuntime(func(_ context.Context, s container.ContainerSpec) (RuntimeSession, error) {
		rt.spec = s
		return rt, nil
	}, p, func(_ context.Context, ref string) (RuntimeCredential, error) {
		if ref != "exact-runtime-credential" {
			t.Fatal("wrong reference")
		}
		return credential, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	a.now = func() time.Time { return credentialNow }
	a.quiet = quiescencePolicy{attempts: 3, interval: time.Millisecond}
	a.retry = retryPolicy{waits: []time.Duration{time.Millisecond, time.Millisecond}, grace: 200 * time.Millisecond}
	return a
}

func credentialsLogin() RuntimeCredential {
	return RuntimeCredential{Mode: ModeCredentials, Login: []byte(dedicatedCredentials)}
}

func openSession(t *testing.T, a *Agents, spec run.SessionSpec) *nativeSession {
	t.Helper()
	s, err := a.Open(context.Background(), spec)
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
func task(text string) run.Message {
	return run.Message{Events: []run.Event{{Assignment: &run.Assignment{Task: text}}}}
}

// judgments records the decisions that reached Run's judge tool.
func judgments(spec *run.SessionSpec) *[]run.Decision {
	var got []run.Decision
	var mu sync.Mutex
	spec.Tools.Judge = func(_ context.Context, d run.Decision, _ []run.Finding) error {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, d)
		return nil
	}
	return &got
}
