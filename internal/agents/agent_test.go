package agents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

const remoteURL = "http://remote:8080/run.git"

// --- a project for the harness ---

type memRepo map[string]string

func (r memRepo) Files() ([]string, error) {
	var out []string
	for f := range r {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}
func (r memRepo) Read(f string) ([]byte, error) {
	if d, ok := r[f]; ok {
		return []byte(d), nil
	}
	return nil, fmt.Errorf("no %s", f)
}
func (memRepo) GitIgnored(string) bool { return false }

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

func library(t *testing.T) (*project.Project, memRepo) {
	t.Helper()
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	m := project.ModelDir + "/"
	table := "| File | Realizes | Why |\n|---|---|---|\n"
	repo := memRepo{
		project.ManifestFile:                      "libraries:\n  std: 1.0.0\n",
		project.LockFile:                          string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                        "checks: []\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]]."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/cards/card.md":               def("std/Concept", "Card", "A [[members/member]] holds one."),
		project.RootPage:                          table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n",
		project.RegistryDir + "/members/cards.md": "---\npaths: [src/members/cards/**]\n---\n" + table + "| src/members/cards/card.go | [[members/cards/card]] | Type |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/cards/card.go":               "package cards\n",
	}
	p, err := project.Load(repo)
	if err != nil || len(p.Errors) > 0 {
		t.Fatalf("fixture: %v %v", err, p.Errors)
	}
	return p, repo
}

// --- a local Git remote standing in for the Run's remote ---

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "core.autocrlf=false", "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// remote makes a bare repository holding the project at base and the
// branches the Run's remote has.
func remote(t *testing.T, repo memRepo) string {
	t.Helper()
	root := t.TempDir()
	seed := filepath.Join(root, "seed")
	for f, data := range repo {
		path := filepath.Join(seed, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, seed, "init", "-q", "-b", "base")
	git(t, seed, "add", "-A")
	git(t, seed, "commit", "-q", "-m", "base")
	bare := filepath.Join(root, "run.git")
	git(t, root, "clone", "-q", "--bare", seed, bare)
	for _, b := range []string{"root", "managers/members", "managers/members.cards"} {
		git(t, bare, "branch", b, "base")
	}
	return bare
}

// --- a scripted container runtime ---

// fakeRuntime runs the agent's Git commands with the host's Git: /work is a
// temporary directory, the global Git configuration and the credential
// store live in a temporary home, and the remote URL leads to a bare
// repository. Provider processes are scripts.
type fakeRuntime struct {
	t       *testing.T
	bare    string
	work    string
	home    string
	script  func(s *fakeSession, argv []string, stdin []byte) int
	auth    string
	mu      sync.Mutex
	volumes map[string]bool
	spec    container.ContainerSpec
	session *fakeSession
	census  []frame.Process
}

func newRuntime(t *testing.T, bare string) *fakeRuntime {
	return &fakeRuntime{t: t, bare: bare, work: filepath.Join(t.TempDir(), "work"), home: t.TempDir(), volumes: map[string]bool{}, auth: `{"loggedIn":true,"authMethod":"claude.ai"}`}
}

func (r *fakeRuntime) CreateVolume(_ context.Context, name string, _ map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.volumes[name] = true
	return nil
}

func (r *fakeRuntime) RemoveVolume(_ context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.volumes[name] {
		return fmt.Errorf("no volume %s", name)
	}
	delete(r.volumes, name)
	return nil
}

func (r *fakeRuntime) Start(_ context.Context, spec container.ContainerSpec) (agents.Session, error) {
	r.spec = spec
	r.session = &fakeSession{rt: r, calls: make(chan frame.ToolCall), replies: make(chan frame.ToolReply)}
	return r.session, nil
}

func (r *fakeRuntime) Exec(ctx context.Context, name, dir string, argv []string, stdin io.Reader, stdout io.Writer) error {
	if name != r.spec.Name || argv[0] != "/usr/bin/git" {
		return fmt.Errorf("unexpected exec %s %q", name, argv)
	}
	args := append([]string(nil), argv[1:]...)
	for i, a := range args {
		if a == "/work" {
			args[i] = r.work
		}
	}
	if dir == "/work" {
		dir = r.work
	}
	if _, err := os.Stat(dir); err != nil {
		dir = filepath.Dir(r.work)
	}
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "url." + filepath.ToSlash(r.bare) + ".insteadOf=" + remoteURL}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+r.home, "GIT_CONFIG_GLOBAL="+filepath.Join(r.home, ".gitconfig"), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return &container.ExecError{Code: exit.ExitCode(), Diagnostics: strings.TrimSpace(stderr.String())}
	}
	return err
}

type fakeSession struct {
	rt          *fakeRuntime
	mu          sync.Mutex
	initialized frame.Initialize
	argv        [][]string
	stdin       bytes.Buffer
	out         *io.PipeReader
	outW        *io.PipeWriter
	exited      chan frame.Exited
	stop        chan struct{}
	calls       chan frame.ToolCall
	replies     chan frame.ToolReply
	nextCall    uint64
	cleared     bool
	closed      bool
}

func (s *fakeSession) Initialize(_ context.Context, i frame.Initialize) error {
	s.initialized = i
	return nil
}
func (s *fakeSession) ClearPrivateHome(context.Context) error { s.cleared = true; return nil }
func (s *fakeSession) StartProvider(_ context.Context, req frame.Start) (frame.Started, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.argv = append(s.argv, req.Argv)
	s.stdin.Reset()
	s.out, s.outW = io.Pipe()
	s.exited = make(chan frame.Exited, 1)
	s.stop = make(chan struct{})
	return frame.Started{PID: 7}, nil
}
func (s *fakeSession) Stdout() io.Reader { return s.out }
func (s *fakeSession) WriteInput(b []byte) error {
	s.stdin.Write(b)
	return nil
}
func (s *fakeSession) CloseInput() error {
	argv := s.argv[len(s.argv)-1]
	stdin := append([]byte(nil), s.stdin.Bytes()...)
	go func() {
		code := 0
		if argv[len(argv)-2] == "auth" && argv[len(argv)-1] == "status" {
			fmt.Fprintln(s.outW, s.rt.auth)
		} else {
			code = s.rt.script(s, argv, stdin)
		}
		s.outW.Close()
		s.exited <- frame.Exited{Code: code}
	}()
	return nil
}
func (s *fakeSession) ToolCalls() <-chan frame.ToolCall { return s.calls }
func (s *fakeSession) Reply(r frame.ToolReply) error {
	s.replies <- r
	return nil
}
func (s *fakeSession) Stop(ctx context.Context, _ int64) (frame.Exited, error) {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	select {
	case e := <-s.exited:
		return e, nil
	case <-ctx.Done():
		return frame.Exited{}, ctx.Err()
	}
}
func (s *fakeSession) Census(context.Context) ([]frame.Process, error) { return s.rt.census, nil }
func (s *fakeSession) Diagnostics() []byte                             { return nil }
func (s *fakeSession) Close() error                                    { s.closed = true; return nil }

// line writes one stream-json line of the scripted provider.
func (s *fakeSession) line(v string) { fmt.Fprintln(s.outW, v) }

// call makes a Manager tool call and returns the tool's text and whether it
// is an error.
func (s *fakeSession) call(name string, args any) (string, bool) {
	raw, _ := json.Marshal(args)
	s.nextCall++
	s.calls <- frame.ToolCall{ID: s.nextCall, Name: name, Arguments: raw}
	reply := <-s.replies
	var result struct {
		Content []struct{ Text string } `json:"content"`
		IsError bool                    `json:"isError"`
	}
	if err := json.Unmarshal(reply.Result, &result); err != nil || len(result.Content) != 1 {
		panic(fmt.Sprintf("tool reply %s", reply.Result))
	}
	return result.Content[0].Text, result.IsError
}

func initLine(tools ...string) string {
	all := []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"}
	for _, t := range tools {
		all = append(all, "mcp__markitect__"+t)
	}
	raw, _ := json.Marshal(all)
	return `{"type":"system","subtype":"init","session_id":"s-1","model":"claude-sonnet-5-5","claude_code_version":"` + claude.Version +
		`","apiKeySource":"none","permissionMode":"bypassPermissions","tools":` + string(raw) +
		`,"agents":[],"skills":[],"plugins":[],"slash_commands":[],"mcp_servers":[{"name":"markitect","status":"connected"}]}`
}

const resultLine = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s-1","usage":{"input_tokens":10,"cache_creation_input_tokens":20,"cache_read_input_tokens":30,"output_tokens":40}}`

// --- the engine's host tools ---

type call struct{ Tool, To, Text string }

// host records the host tool calls and refuses a text that asks for it.
type host struct {
	mu    sync.Mutex
	calls []call
}

func (h *host) record(tool, to, text string) (agentport.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.Contains(text, "refuse") {
		return agentport.Message{}, errors.New("the engine refuses this; ask your parent instead")
	}
	h.calls = append(h.calls, call{tool, to, text})
	return agentport.Message{ID: fmt.Sprintf("m-%d", len(h.calls)), To: to}, nil
}
func (h *host) Ask(_ context.Context, text string) (agentport.Message, error) {
	return h.record("ask", "", text)
}
func (h *host) Escalate(_ context.Context, text string) (agentport.Message, error) {
	return h.record("escalate", "", text)
}
func (h *host) Answer(_ context.Context, to, text string) (agentport.Message, error) {
	return h.record("answer", to, text)
}
func (h *host) Assign(_ context.Context, child, text string) (agentport.Message, error) {
	return h.record("assign", child, text)
}
func (h *host) Note(_ context.Context, text string) (agentport.Note, error) {
	m, err := h.record("note", "", text)
	return agentport.Note{ID: m.ID, Text: text}, err
}
func (h *host) Notes(_ context.Context, after string) (agentport.NotePage, error) {
	h.record("notes", after, "")
	return agentport.NotePage{Notes: []agentport.Note{{ID: "n-1", Manager: "catalog", Role: agentport.Maker, Text: "the README is stale"}}}, nil
}

var allTools = []agentport.Tool{agentport.ToolAsk, agentport.ToolEscalate, agentport.ToolAnswer, agentport.ToolAssign, agentport.ToolNote, agentport.ToolNotes}

func login(t *testing.T, expires time.Time) []byte {
	raw, _ := json.Marshal(map[string]any{"claudeAiOauth": map[string]any{"accessToken": "sk-ant-oat01-ACCESS-SECRET", "refreshToken": "sk-ant-ort01-REFRESH-SECRET", "expiresAt": expires.UnixMilli(), "scopes": []string{"user:inference"}}})
	return raw
}

type fixture struct {
	rt         *fakeRuntime
	host       *host
	spec       agents.Spec
	bare       string
	transcript *bytes.Buffer
}

// setup prepares one agent of the Manager id with its checklist; Root works
// with members, and members with members/cards.
func setup(t *testing.T, role agentport.Role, id string, items ...string) *fixture {
	t.Helper()
	_, repo := library(t)
	bare := remote(t, repo)
	var checklist []agentport.Item
	for _, i := range items {
		checklist = append(checklist, agentport.Item{Key: i, Part: agentport.Own, Definition: i, Why: "changed"})
	}
	manager := agentport.Manager{ID: id, Parent: ".", Children: []string{"members/cards"}}
	if id == "." {
		manager = agentport.Manager{ID: ".", Children: []string{"members"}}
	}
	pkg := harness.Package(id)
	user := strings.ReplaceAll(pkg, "/", ".") + "-" + string(role)
	f := &fixture{rt: newRuntime(t, bare), host: &host{}, bare: bare, transcript: &bytes.Buffer{}}
	f.spec = agents.Spec{
		Name: "markitect-run-r1-" + user, Labels: map[string]string{agents.RunLabel: "r1"}, Image: "sha256:" + strings.Repeat("a", 64),
		Limits:  agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20},
		Network: "evaluation",
		Start: agentport.Start{Run: "r1", Mode: agentport.Apply, Role: role, Manager: manager, Round: 1, Checklist: checklist, Tools: allTools,
			Git:      agentport.GitAccess{URL: remoteURL, Network: "markitect-run-r1", Branch: runremote.Branch(pkg), Base: git(t, bare, "rev-parse", "base"), User: user, Token: "run-token-SECRET-123"},
			Deadline: time.Now().Add(time.Hour)},
		Host:  f.host,
		Login: login(t, time.Now().Add(2*time.Hour)), Model: "claude-sonnet-5-5", Effort: "high", Transcript: f.transcript,
	}
	return f
}

func (f *fixture) start(t *testing.T) *agents.Agent {
	t.Helper()
	a, err := agents.Start(context.Background(), f.rt, claude.Provider{}, f.spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(context.Background()) })
	return a
}

// makerTools are the tools a maker with all host tools has.
var makerTools = []string{"submit", "ask", "escalate", "answer", "assign", "note", "notes"}

func TestStartPreparesTheContainerLoginAndClone(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	a := f.start(t)
	spec := f.rt.spec
	if spec.Network != "evaluation" || len(spec.MoreNetworks) != 1 || spec.MoreNetworks[0] != "markitect-run-r1" || spec.User != "1000:1000" || spec.Workdir != "/work" {
		t.Fatalf("container spec %+v", spec)
	}
	if len(spec.Mounts) != 2 || spec.Mounts[0].Target != "/home/agent" || spec.Mounts[1].Target != "/work" || len(f.rt.volumes) != 2 {
		t.Fatalf("mounts %+v volumes %v", spec.Mounts, f.rt.volumes)
	}
	home := f.rt.session.initialized.Claude
	if home == nil || bytes.Contains(home.Credentials, []byte("REFRESH")) || !bytes.Contains(home.Credentials, []byte("ACCESS-SECRET")) {
		t.Fatalf("private home %+v", home)
	}
	if !strings.Contains(home.SystemPrompt, "maker of the Manager of Package `members`") || !strings.Contains(home.AppendSystemPrompt, "## Child members/cards") {
		t.Fatalf("harness files not installed")
	}
	if got := git(t, f.rt.work, "rev-parse", "--abbrev-ref", "HEAD"); got != "managers/members" {
		t.Fatalf("clone on %s", got)
	}
	creds, _ := os.ReadFile(filepath.Join(f.rt.home, ".git-credentials"))
	if !strings.Contains(string(creds), "members-maker:run-token-SECRET-123@remote") {
		t.Fatalf("credential store %q", creds)
	}
	if err := a.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !f.rt.session.cleared || !f.rt.session.closed || len(f.rt.volumes) != 0 {
		t.Fatalf("close left cleared=%v closed=%v volumes=%v", f.rt.session.cleared, f.rt.session.closed, f.rt.volumes)
	}
}

func TestStartRefusesAnExpiredOrShortLivedLoginAndAnotherAuthMethod(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.spec.Login = login(t, time.Now().Add(-time.Minute))
	var se *agents.StartError
	if _, err := agents.Start(context.Background(), f.rt, claude.Provider{}, f.spec); !errors.As(err, &se) || se.Class != agents.LoginFailed {
		t.Fatalf("expired login: %v", err)
	}
	f.spec.Login = login(t, time.Now().Add(30*time.Minute))
	if _, err := agents.Start(context.Background(), f.rt, claude.Provider{}, f.spec); !errors.As(err, &se) || !strings.Contains(err.Error(), "before the deadline") {
		t.Fatalf("login shorter than the deadline: %v", err)
	}
	f.spec.Login = login(t, time.Now().Add(2*time.Hour))
	f.rt.auth = `{"loggedIn":true,"authMethod":"api_key"}`
	if _, err := agents.Start(context.Background(), f.rt, claude.Provider{}, f.spec); !errors.As(err, &se) || se.Class != agents.LoginFailed {
		t.Fatalf("API key login: %v", err)
	}
	if len(f.rt.volumes) != 0 || !f.rt.session.closed {
		t.Fatalf("a failed start left volumes %v", f.rt.volumes)
	}
}

func TestMakerSubmitsWithTrailersAndTheTurnAnswers(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	var first, second string
	var firstErr, secondErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		os.WriteFile(filepath.Join(s.rt.work, "src", "members", "member.go"), []byte("package members\n\ntype Member struct{}\n"), 0o644)
		first, firstErr = s.call("submit", map[string]any{"summary": "Member", "verdicts": []map[string]string{}})
		second, secondErr = s.call("submit", map[string]any{"summary": "Realize the member\nwith a type", "verdicts": []map[string]string{
			{"item": "members/member", "verdict": "changed", "evidence": "type Member in\nsrc/members/member.go"}}})
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	out, err := a.Send(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !firstErr || !strings.Contains(first, `"members/member" has no verdict`) {
		t.Fatalf("incomplete submission: %v %s", firstErr, first)
	}
	if secondErr || !strings.Contains(second, "submitted ") {
		t.Fatalf("submission: %v %s", secondErr, second)
	}
	if out.End != agents.Answered || out.Conversation != "s-1" || out.Model != "claude-sonnet-5-5" || out.Usage.CacheRead != 30 {
		t.Fatalf("outcome %+v", out)
	}
	msg := git(t, f.bare, "log", "-1", "--format=%B", "managers/members")
	tr, err := runremote.ParseMessage(msg)
	if err != nil || tr.Kind != runremote.Submission || tr.Manager != "members" || len(tr.Verdicts) != 1 || tr.Verdicts[0].Evidence != "type Member in src/members/member.go" {
		t.Fatalf("pushed message %q: %+v %v", msg, tr, err)
	}
	if !strings.HasPrefix(msg, "Realize the member with a type\n\n") {
		t.Fatalf("subject: %q", msg)
	}
	if got := git(t, f.bare, "show", "managers/members:src/members/member.go"); !strings.Contains(got, "type Member") {
		t.Fatalf("pushed tree lacks the change: %s", got)
	}
	argv := f.rt.session.argv[1]
	if strings.Contains(strings.Join(argv, " "), "--resume") || !strings.Contains(strings.Join(argv, " "), "--model claude-sonnet-5-5 --effort high") {
		t.Fatalf("first turn argv %q", argv)
	}
	if !strings.Contains(f.rt.session.stdin.String(), "# Task") {
		t.Fatalf("first turn without the task: %s", f.rt.session.stdin.String())
	}
	// A later turn resumes the conversation and carries only its message.
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(strings.Replace(initLine(makerTools...), "s-1", "s-1", 1))
		s.line(resultLine)
		return 0
	}
	if _, err := a.Send(context.Background(), "Your checker rejected the submission."); err != nil {
		t.Fatal(err)
	}
	argv = f.rt.session.argv[2]
	if !strings.Contains(strings.Join(argv, " "), "--resume s-1") || strings.Contains(f.rt.session.stdin.String(), "# Task") {
		t.Fatalf("later turn argv %q stdin %s", argv, f.rt.session.stdin.String())
	}
	if bytes.Contains(f.transcript.Bytes(), []byte("SECRET")) {
		t.Fatalf("transcript leaks a secret")
	}
	if !bytes.Contains(f.transcript.Bytes(), []byte(`"kind":"tool-call"`)) {
		t.Fatalf("transcript lacks the tool calls")
	}
}

func TestARefusedPushLeavesTheWorkAsItWas(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	hook := filepath.Join(f.bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'refused: required check go-test failed; fix the test and push again' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var text string
	var isErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		os.WriteFile(filepath.Join(s.rt.work, "src", "members", "member.go"), []byte("broken\n"), 0o644)
		text, isErr = s.call("submit", map[string]any{"summary": "x", "verdicts": []map[string]string{{"item": "members/member", "verdict": "changed", "evidence": "e"}}})
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if !isErr || !strings.Contains(text, "required check go-test failed") {
		t.Fatalf("refusal: %v %s", isErr, text)
	}
	if git(t, f.rt.work, "rev-parse", "HEAD") != git(t, f.bare, "rev-parse", "base") {
		t.Fatal("the refused submission stayed committed")
	}
	if status := git(t, f.rt.work, "status", "--porcelain"); !strings.Contains(status, "src/members/member.go") {
		t.Fatalf("the work is gone: %q", status)
	}
}

func TestCheckerReviewsTheTipWithACommitThatChangesNoFile(t *testing.T) {
	f := setup(t, agentport.Checker, "members", "members/member")
	f.spec.Name = "markitect-run-r1-members-checker"
	// The maker's submission at the tip of the branch.
	maker := filepath.Join(t.TempDir(), "maker")
	git(t, filepath.Dir(maker), "clone", "-q", "--branch", "managers/members", f.bare, maker)
	os.WriteFile(filepath.Join(maker, "src", "members", "member.go"), []byte("package members\n\ntype Member struct{}\n"), 0o644)
	git(t, maker, "commit", "-q", "-am", "Member\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | changed | type Member")
	git(t, maker, "push", "-q", "origin", "HEAD:managers/members")
	submission := git(t, f.bare, "rev-parse", "managers/members")
	var texts []string
	var errs []bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine("review"))
		os.WriteFile(filepath.Join(s.rt.work, "README.md"), []byte("edited by the checker\n"), 0o644)
		for _, args := range []map[string]any{
			{"outcome": "approved", "checks": []map[string]string{{"item": "members/member", "answer": "disputed", "evidence": "no test"}}},
			{"outcome": "approved", "checks": []map[string]string{}},
			{"outcome": "approved", "checks": []map[string]string{{"item": "members/member", "answer": "confirmed", "evidence": "read member.go"}}},
		} {
			text, isErr := s.call("review", args)
			texts, errs = append(texts, text), append(errs, isErr)
		}
		text, isErr := s.call("submit", map[string]any{})
		texts, errs = append(texts, text), append(errs, isErr)
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	// The checker's task names the submission and its verdicts as the clone holds them.
	if task := f.rt.session.stdin.String(); !strings.Contains(task, "submission `"+submission+"`") || !strings.Contains(task, "| `members/member` | changed | type Member |") || !strings.Contains(task, "+type Member struct{}") {
		t.Fatalf("checker task: %s", task)
	}
	if !errs[0] || !strings.Contains(texts[0], "an approval disputes no verdict") || !errs[1] || !strings.Contains(texts[1], "has no answer") {
		t.Fatalf("inconsistent reviews accepted: %q", texts)
	}
	if errs[2] {
		t.Fatalf("review refused: %s", texts[2])
	}
	if !errs[3] || !strings.Contains(texts[3], `no tool "submit"`) {
		t.Fatalf("a checker in apply submitted: %q", texts[3])
	}
	review := git(t, f.bare, "rev-parse", "managers/members")
	if git(t, f.bare, "rev-parse", review+"^") != submission || git(t, f.bare, "rev-parse", review+"^{tree}") != git(t, f.bare, "rev-parse", submission+"^{tree}") {
		t.Fatal("the review is not an empty commit on the submission")
	}
	tr, err := runremote.ParseMessage(git(t, f.bare, "log", "-1", "--format=%B", review))
	if err != nil || tr.Kind != runremote.Review || tr.Review != runremote.Approved || len(tr.Checks) != 1 {
		t.Fatalf("review trailers %+v %v", tr, err)
	}
}

func TestHostToolsReachTheEngineAndItsRefusalsTheAgent(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	var texts []string
	var errs []bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		for _, c := range []struct {
			name string
			args map[string]string
		}{
			{"assign", map[string]string{"child": "members/cards", "text": "realize the card limit"}},
			{"ask", map[string]string{"text": "which limit?"}},
			{"answer", map[string]string{"to": "m-7", "text": "three"}},
			{"escalate", map[string]string{"text": "the model contradicts itself"}},
			{"note", map[string]string{"text": "the README is stale"}},
			{"notes", map[string]string{}},
			{"ask", map[string]string{"text": "please refuse"}},
			{"ask", map[string]string{"text": "x", "extra": "y"}},
		} {
			text, isErr := s.call(c.name, c.args)
			texts, errs = append(texts, text), append(errs, isErr)
		}
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	want := []call{{"assign", "members/cards", "realize the card limit"}, {"ask", "", "which limit?"}, {"answer", "m-7", "three"},
		{"escalate", "", "the model contradicts itself"}, {"note", "", "the README is stale"}, {"notes", "", ""}}
	if fmt.Sprint(f.host.calls) != fmt.Sprint(want) {
		t.Fatalf("host calls %+v", f.host.calls)
	}
	for i := 0; i < 6; i++ {
		if errs[i] {
			t.Errorf("call %d refused: %s", i, texts[i])
		}
	}
	if texts[0] != "sent as m-1 to members/cards" || texts[4] != "recorded as Note m-5" || !strings.Contains(texts[5], `"Text":"the README is stale"`) {
		t.Fatalf("results %q", texts[:6])
	}
	if !errs[6] || !strings.Contains(texts[6], "the engine refuses this; ask your parent instead") || !errs[7] || !strings.Contains(texts[7], "schema") {
		t.Fatalf("refusals %q", texts[6:])
	}
}

func TestOnlyTheStartsToolsAreOffered(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.spec.Start.Tools = []agentport.Tool{agentport.ToolEscalate}
	var text string
	var isErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine("submit", "escalate"))
		text, isErr = s.call("ask", map[string]string{"text": "x"})
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if out, err := a.Send(context.Background(), ""); err != nil || out.End != agents.Answered {
		t.Fatalf("outcome %+v %v", out, err)
	}
	if !isErr || !strings.Contains(text, `no tool "ask"`) || len(f.host.calls) != 0 {
		t.Fatalf("ask: %v %s %v", isErr, text, f.host.calls)
	}
}

func TestTurnEndsAtTheDeadline(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		<-s.stop
		return -1
	}
	f.spec.Start.Deadline = time.Now().Add(3 * time.Second)
	a := f.start(t)
	out, err := a.Send(context.Background(), "")
	if err != nil || out.End != agents.Deadline {
		t.Fatalf("outcome %+v %v", out, err)
	}
	f = setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		<-s.stop
		return -1
	}
	a = f.start(t)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	if out, err = a.Send(ctx, ""); err != nil || out.End != agents.Stopped {
		t.Fatalf("outcome %+v %v", out, err)
	}
}

func TestLaunchContractBreachesFailTheTurn(t *testing.T) {
	cases := map[string][]string{
		"native sub-agent": {initLine(makerTools...), `{"type":"assistant","parent_tool_use_id":"toolu_1","message":{"role":"assistant","content":[]}}`},
		"api key":          {strings.Replace(initLine(makerTools...), `"apiKeySource":"none"`, `"apiKeySource":"ANTHROPIC_API_KEY"`, 1)},
		"no init":          {resultLine},
	}
	for name, lines := range cases {
		t.Run(name, func(t *testing.T) {
			f := setup(t, agentport.Maker, "members", "members/member")
			f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
				for _, l := range lines {
					s.line(l)
				}
				<-s.stop
				return -1
			}
			a := f.start(t)
			out, err := a.Send(context.Background(), "")
			if err != nil || out.End != agents.Failed || out.Class != agents.Other {
				t.Fatalf("outcome %+v %v", out, err)
			}
		})
	}
}

func TestProviderFailureIsClassified(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		s.line(`{"type":"system","subtype":"api_retry","attempt":1,"error":"overloaded","session_id":"s-1"}`)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"error":"overloaded","isApiErrorMessage":true,"message":{"role":"assistant","content":[{"type":"text","text":"API Error: 529 overloaded"}]},"session_id":"s-1"}`)
		s.line(`{"type":"result","subtype":"success","is_error":true,"result":"API Error: 529","session_id":"s-1","usage":{}}`)
		return 1
	}
	a := f.start(t)
	out, err := a.Send(context.Background(), "")
	if err != nil || out.End != agents.Failed || out.Class != agents.Capacity {
		t.Fatalf("outcome %+v %v", out, err)
	}
}

func TestProcessesThatOutliveTheTurnAreAnError(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	f.rt.census = []frame.Process{{PID: 9, Command: "sleep"}}
	if _, err := a.Send(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "outlived the turn") {
		t.Fatalf("leftover process accepted: %v", err)
	}
}

func TestVerifyCheckerPushesItsVerdictsWithoutChangingAFile(t *testing.T) {
	f := setup(t, agentport.Checker, "members", "members/member")
	f.spec.Start.Mode = agentport.Verify
	var texts []string
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine("submit"))
		for _, word := range []string{"changed", "deviates"} {
			text, _ := s.call("submit", map[string]any{"verdicts": []map[string]string{{"item": "members/member", "verdict": word, "evidence": "observed no card; expected a card"}}})
			texts = append(texts, text)
		}
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(texts[0], `"changed" is not conforms, deviates, model-problem`) || !strings.Contains(texts[1], "submitted the verdicts") {
		t.Fatalf("verify submits: %q", texts)
	}
	tip := git(t, f.bare, "rev-parse", "managers/members")
	if git(t, f.bare, "rev-parse", tip+"^{tree}") != git(t, f.bare, "rev-parse", "base^{tree}") || git(t, f.bare, "rev-parse", tip+"^") != git(t, f.bare, "rev-parse", "base") {
		t.Fatal("the verdicts changed a file")
	}
	tr, err := runremote.ParseMessage(git(t, f.bare, "log", "-1", "--format=%B", tip))
	if err != nil || tr.Kind != runremote.Submission || tr.Verdicts[0].Word != runremote.Deviates {
		t.Fatalf("trailers %+v %v", tr, err)
	}
}

func TestAnIntegratingParentSubmitsWithItsManagerLineAlone(t *testing.T) {
	f := setup(t, agentport.Maker, ".")
	var text string
	var isErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		text, isErr = s.call("submit", map[string]any{"summary": "Integrate members", "verdicts": []map[string]string{}})
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	tr, err := runremote.ParseMessage(git(t, f.bare, "log", "-1", "--format=%B", "root"))
	if isErr || err != nil || tr.Kind != runremote.Submission || tr.Manager != "." || len(tr.Verdicts) != 0 {
		t.Fatalf("submission %q %+v %v", text, tr, err)
	}
}
