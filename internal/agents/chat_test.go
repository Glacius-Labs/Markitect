package agents_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/agents/codex"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

// chatToken stands for the user's token from `claude setup-token`.
const chatToken = "sk-ant-oat01-ACCESS-SECRET"

// --- a scripted container runtime ---

// git runs the host's Git in dir.
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

// fakeRuntime runs the agent's Git commands with the host's Git: /work is a
// temporary directory and the global Git configuration lives in a temporary
// home. Provider processes are scripts.
type fakeRuntime struct {
	t       *testing.T
	work    string
	home    string
	script  func(s *fakeSession, argv []string, stdin []byte) int
	auth    string
	mu      sync.Mutex
	volumes map[string]bool
	spec    container.ContainerSpec
	session *fakeSession
}

func newRuntime(t *testing.T) *fakeRuntime {
	return &fakeRuntime{t: t, work: filepath.Join(t.TempDir(), "work"), home: t.TempDir(), volumes: map[string]bool{}, auth: `{"loggedIn":true,"authMethod":"oauth_token"}`}
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
	r.session = &fakeSession{rt: r}
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
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
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

// fakeSession is the agentd connection of a chat's container. It records what
// the host sent and plays the provider's output.
type fakeSession struct {
	rt          *fakeRuntime
	mu          sync.Mutex
	initialized frame.Initialize
	argv        [][]string
	envs        [][]string
	stdin       bytes.Buffer
	out         *io.PipeReader
	outW        *io.PipeWriter
	exited      chan frame.Exited
	stop        chan struct{}
	cleared     bool
	closed      bool
}

// Initialize refuses what agentd refuses.
func (s *fakeSession) Initialize(_ context.Context, i frame.Initialize) error {
	s.initialized = i
	_, err := frame.EncodeControl(frame.ControlMessage{Type: frame.TypeInitialize, Initialize: &i})
	return err
}
func (s *fakeSession) ClearPrivateHome(context.Context) error { s.cleared = true; return nil }
func (s *fakeSession) StartProvider(_ context.Context, req frame.Start) (frame.Started, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.argv = append(s.argv, req.Argv)
	s.envs = append(s.envs, req.Env)
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
func (s *fakeSession) ToolCalls() <-chan frame.ToolCall                { return nil }
func (s *fakeSession) Reply(frame.ToolReply) error                     { return nil }
func (s *fakeSession) Census(context.Context) ([]frame.Process, error) { return nil, nil }
func (s *fakeSession) Diagnostics() []byte                             { return nil }
func (s *fakeSession) Close() error                                    { s.closed = true; return nil }
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

// line writes one stream-json line of the scripted provider.
func (s *fakeSession) line(v string) { fmt.Fprintln(s.outW, v) }

// initLine is the init line of a chat turn whose launch held as asked.
func initLine() string {
	raw, _ := json.Marshal([]string{"Bash", "Read", "Edit", "Write", "Glob", "Grep"})
	return `{"type":"system","subtype":"init","session_id":"s-1","model":"claude-sonnet-5-5","claude_code_version":"` + claude.Version +
		`","apiKeySource":"none","permissionMode":"bypassPermissions","tools":` + string(raw) +
		`,"agents":[],"skills":[],"plugins":[],"slash_commands":[],"mcp_servers":[{"name":"markitect","status":"connected"}]}`
}

const resultLine = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s-1","usage":{"input_tokens":10,"cache_creation_input_tokens":20,"cache_read_input_tokens":30,"output_tokens":40}}`

// chatTree is a working tree with a binary file and names that need care.
var chatTree = map[string]string{
	".markitect/model/project.md":        "---\nkind: std/Project\n---\n# Library\n",
	".markitect/model/members/member.md": "---\nkind: std/Concept\n---\n# Member\n",
	"src/a.go":                           "package a\n",
	"docs/with space.md":                 "# Spaces\n",
	"docs/größe.md":                      "# Umlaut\n",
	"assets/logo.bin":                    string([]byte{0, 1, 2, 255, '\n', 0}),
}

func chatSpec(t *testing.T, pending ...agents.ChatFile) agents.ChatSpec {
	var files []string
	for f := range chatTree {
		files = append(files, f)
	}
	sort.Strings(files)
	return agents.ChatSpec{
		Name: "markitect-chat-0123456789abcdef", Labels: map[string]string{"markitect.chat": "0123456789abcdef"}, Image: "sha256:" + strings.Repeat("a", 64),
		Limits: agents.Limits{CPUs: 1, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20}, Network: "markitect-provider",
		Token: chatToken, Model: "claude-sonnet-5-5", Effort: "high",
		Skill: harness.ChatModeling, Earlier: "owner: Members borrow books.\n\nagent: Noted.",
		Files: files,
		Read: func(f string) ([]byte, error) {
			data, ok := chatTree[f]
			if !ok {
				return nil, os.ErrNotExist
			}
			return []byte(data), nil
		},
		Pending: pending,
		Turn:    30 * time.Minute,
	}
}

func readWork(t *testing.T, rt *fakeRuntime, f string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rt.work, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAChatCopiesTheWorkingTreeAndProposesItsChanges(t *testing.T) {
	rt := newRuntime(t)
	pending := []agents.ChatFile{
		{Path: ".markitect/model/members/member.md", Data: []byte("---\nkind: std/Concept\n---\n# Member\n\nSomeone who borrows.\n")},
		{Path: ".markitect/model/fee.md", Data: []byte("---\nkind: std/Rule\n---\n# Late fee\n")},
		{Path: ".markitect/model/project.md", Removed: true},
	}
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t, pending...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })

	// The copy holds every file as it is, with the pending proposal on top;
	// the start holds the working tree.
	if got := readWork(t, rt, "assets/logo.bin"); got != chatTree["assets/logo.bin"] {
		t.Errorf("the binary file arrived as %q", got)
	}
	for _, f := range []string{"docs/with space.md", "docs/größe.md", "src/a.go"} {
		if got := readWork(t, rt, f); got != chatTree[f] {
			t.Errorf("%s arrived as %q", f, got)
		}
	}
	if got := readWork(t, rt, ".markitect/model/members/member.md"); !strings.Contains(got, "Someone who borrows.") {
		t.Errorf("the pending change is not in the copy: %q", got)
	}
	if _, err := os.Stat(filepath.Join(rt.work, ".markitect", "model", "project.md")); !os.IsNotExist(err) {
		t.Errorf("the pending removal is not in the copy: %v", err)
	}
	if got := git(t, rt.work, "show", harness.ChatStart+":.markitect/model/members/member.md"); got != strings.TrimSpace(chatTree[".markitect/model/members/member.md"]) {
		t.Errorf("the start holds %q", got)
	}

	// The chat's documents: its role and skill, no Manager tools, and the
	// conversation so far before the owner's first message.
	if home := rt.session.initialized.Claude; home == nil || !strings.Contains(home.SystemPrompt, "# Markitect chat") || !strings.Contains(home.AppendSystemPrompt, "markitect-modeling/SKILL.md") {
		t.Fatalf("the chat's home %+v", rt.session.initialized)
	}
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		os.WriteFile(filepath.Join(s.rt.work, ".markitect", "model", "loan.md"), []byte("---\nkind: std/Concept\n---\n# Loan\n"), 0o644)
		os.WriteFile(filepath.Join(s.rt.work, "src", "a.go"), []byte("package a // touched\n"), 0o644)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"I added the Loan."`, 1))
		return 0
	}
	var heard []agents.Progress
	out, err := chat.Send(context.Background(), "Add the loan.", func(p agents.Progress) { heard = append(heard, p) })
	if err != nil || out.End != agents.Answered || out.Text != "I added the Loan." {
		t.Fatalf("turn %+v %v", out, err)
	}
	if len(heard) != 1 || heard[0].Text != "I added the Loan." {
		t.Errorf("the final answer was heard as %+v", heard)
	}
	if in := rt.session.stdin.String(); !strings.Contains(in, "The conversation so far") || !strings.Contains(in, "Members borrow books.") || !strings.Contains(in, "Add the loan.") {
		t.Errorf("first message %s", in)
	}
	if argv := strings.Join(rt.session.argv[1], " "); strings.Contains(argv, "mcp__markitect__") {
		t.Errorf("a chat offers a Manager tool: %s", argv)
	}

	model, outside, err := chat.Changes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range model {
		got[f.Path] = string(f.Data)
		if f.Removed {
			got[f.Path] = "removed"
		}
	}
	if len(got) != 4 || got[".markitect/model/project.md"] != "removed" || !strings.Contains(got[".markitect/model/loan.md"], "# Loan") || !strings.Contains(got[".markitect/model/fee.md"], "Late fee") || !strings.Contains(got[".markitect/model/members/member.md"], "borrows") {
		t.Errorf("model changes %v", got)
	}
	if strings.Join(outside, ",") != "src/a.go" {
		t.Errorf("outside %v", outside)
	}

	// An accept makes the copy's model its start, while the change outside
	// stays a change; a reject puts the copy back.
	if err := chat.Settle(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if model, outside, err := chat.Changes(context.Background()); err != nil || len(model) != 0 || strings.Join(outside, ",") != "src/a.go" {
		t.Fatalf("after the accept: %v %v %v", model, outside, err)
	}
	os.WriteFile(filepath.Join(rt.work, ".markitect", "model", "loan.md"), []byte("changed again\n"), 0o644)
	os.WriteFile(filepath.Join(rt.work, ".markitect", "model", "stray.md"), []byte("stray\n"), 0o644)
	if err := chat.Settle(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if model, _, err := chat.Changes(context.Background()); err != nil || len(model) != 0 {
		t.Fatalf("after the reject: %v %v", model, err)
	}
	if got := readWork(t, rt, ".markitect/model/loan.md"); !strings.Contains(got, "# Loan") {
		t.Errorf("the reject left %q", got)
	}
}

// The copy is compared as it is on disk: the project's line-end attributes
// make no unchanged file look changed, and a nested repository is left out.
func TestAChatComparesTheCopyAsItIsOnDisk(t *testing.T) {
	saved := maps.Clone(chatTree)
	t.Cleanup(func() { chatTree = saved })
	chatTree[".gitattributes"] = "* text eol=lf\n"
	chatTree[".markitect/model/windows.md"] = "---\r\nkind: std/Concept\r\n---\r\n# Windows\r\n"
	rt := newRuntime(t)
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })
	if model, outside, err := chat.Changes(context.Background()); err != nil || len(model)+len(outside) != 0 {
		t.Fatalf("an untouched copy changed: %v %v %v", model, outside, err)
	}
	nested := filepath.Join(rt.work, ".markitect", "model", "nested")
	git(t, filepath.Dir(nested), "init", "--quiet", nested)
	os.WriteFile(filepath.Join(nested, "x.md"), []byte("x\n"), 0o644)
	if model, _, err := chat.Changes(context.Background()); err != nil {
		t.Fatalf("a nested repository: %v %v", model, err)
	}

	// A turn is heard as it runs: its messages and what it does, the final
	// answer once though it repeats the last message, and never the token.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Your token is sk-ant-oat01-ACCESS-SECRET."}]},"session_id":"s-1"}`)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"echo sk-ant-oat01-ACCESS-SECRET"}}]},"session_id":"s-1"}`)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Done."}]},"session_id":"s-1"}`)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Done."`, 1))
		return 0
	}
	var heard []agents.Progress
	out, err := chat.Send(context.Background(), "Tell me a secret.", func(p agents.Progress) { heard = append(heard, p) })
	if err != nil || out.Text != "Done." || len(heard) != 3 || heard[0].Text != "Your token is [redacted]." || heard[1].Doing != "Running echo [redacted]" || heard[2].Text != "Done." {
		t.Fatalf("the turn %+v, heard %+v, %v", out, heard, err)
	}

	// An activity is shortened only after its secret is gone, and a long one
	// stays one short line. The limit falls inside the secret, which would be
	// no longer recognized if it were cut first.
	secret := "sk-ant-oat01-ACCESS-SECRET"
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		long := strings.Repeat("x", 90) + " " + secret + " " + strings.Repeat("y", 100)
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"` + long + `"}}]},"session_id":"s-1"}`)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":""`, 1))
		return 0
	}
	heard = nil
	if _, err := chat.Send(context.Background(), "A long one.", func(p agents.Progress) { heard = append(heard, p) }); err != nil || len(heard) != 1 {
		t.Fatalf("heard %+v, %v", heard, err)
	}
	if d := heard[0].Doing; strings.Contains(d, "sk-ant") || !strings.Contains(d, "[redacted]") || len([]rune(d)) > 110 || !strings.HasPrefix(d, "Running xxx") {
		t.Errorf("the activity %q", d)
	}

	// A provider that reports none of its messages along the way still has
	// its final answer heard, redacted.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Your token is sk-ant-oat01-ACCESS-SECRET."`, 1))
		return 0
	}
	heard = nil
	if out, err := chat.Send(context.Background(), "Again.", func(p agents.Progress) { heard = append(heard, p) }); err != nil || strings.Contains(out.Text, "SECRET") || len(heard) != 1 || strings.Contains(heard[0].Text, "SECRET") {
		t.Fatalf("the answer %q, heard %+v, %v", out.Text, heard, err)
	}
}

func TestAChatPassesTheTokenToTheProviderThroughTheEnvironmentOnly(t *testing.T) {
	rt := newRuntime(t)
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Hello."`, 1))
		return 0
	}
	if out, err := chat.Send(context.Background(), "Hello", func(agents.Progress) {}); err != nil || out.End != agents.Answered {
		t.Fatalf("turn %+v %v", out, err)
	}

	// The login check and the turn both start the CLI with the token in their
	// environment; nothing else carries it.
	want := "CLAUDE_CODE_OAUTH_TOKEN=" + chatToken
	s := rt.session
	if len(s.argv) != 2 || len(s.envs) != 2 {
		t.Fatalf("provider starts: %q", s.argv)
	}
	for i, env := range s.envs {
		if !slices.Contains(env, want) {
			t.Errorf("start %d lacks the token in its environment: %q", i, env)
		}
		if strings.Contains(strings.Join(s.argv[i], " "), chatToken) {
			t.Errorf("start %d carries the token on its command line: %q", i, s.argv[i])
		}
	}
	if raw, _ := json.Marshal(s.initialized); strings.Contains(string(raw), chatToken) {
		t.Errorf("the token went through Initialize: %s", raw)
	}
	if strings.Contains(s.stdin.String(), chatToken) {
		t.Error("the token went to the provider's standard input")
	}
	if strings.Contains(fmt.Sprintf("%+v", rt.spec), chatToken) {
		t.Errorf("the container's spec carries the token: %+v", rt.spec)
	}
}

func TestAChatWithoutATokenOrARouteRefusesToStart(t *testing.T) {
	// No token at all.
	rt := newRuntime(t)
	spec := chatSpec(t)
	spec.Token = ""
	var start *agents.StartError
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); !errors.As(err, &start) || start.Class != agents.LoginFailed {
		t.Fatalf("start without a token: %v", err)
	}
	// A provider with no token route, whatever the token.
	if _, err := agents.StartChat(context.Background(), rt, codex.Provider{}, chatSpec(t)); !errors.As(err, &start) || start.Class != agents.LoginFailed {
		t.Fatalf("start on Codex: %v", err)
	}
	if rt.session != nil || len(rt.volumes) != 0 {
		t.Errorf("a refused chat created a container or volumes: %v %v", rt.session, rt.volumes)
	}

	// A login check that does not report the account login ends the start
	// and removes what it created.
	rt.auth = `{"loggedIn":true,"authMethod":"api_key"}`
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, chatSpec(t)); !errors.As(err, &start) || start.Class != agents.LoginFailed {
		t.Fatalf("start with an API key login: %v", err)
	}
	if len(rt.volumes) != 0 || !rt.session.closed || !rt.session.cleared {
		t.Errorf("a refused start left %v, closed %v, cleared %v", rt.volumes, rt.session.closed, rt.session.cleared)
	}
}

func TestAChatRedactsTheTokenFromItsTranscriptAndErrors(t *testing.T) {
	rt := newRuntime(t)
	var transcript bytes.Buffer
	spec := chatSpec(t)
	spec.Transcript = &transcript
	chat, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = chat.Close(context.Background()) })

	// The token in what the owner typed, in a message and in a tool call.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine())
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"You wrote ` + chatToken + `."}]},"session_id":"s-1"}`)
		s.line(strings.Replace(resultLine, `"result":"done"`, `"result":"Again `+chatToken+`"`, 1))
		return 0
	}
	out, err := chat.Send(context.Background(), "My token is "+chatToken, func(agents.Progress) {})
	if err != nil || out.End != agents.Answered || strings.Contains(out.Text, chatToken) {
		t.Fatalf("turn %+v %v", out, err)
	}

	// A launch that breaks the contract fails the turn; its reason names the
	// offending value, which is the token here.
	rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(strings.Replace(initLine(), `"apiKeySource":"none"`, `"apiKeySource":"`+chatToken+`"`, 1))
		return 0
	}
	out, err = chat.Send(context.Background(), "Again", func(agents.Progress) {})
	if err != nil || out.End != agents.Failed || out.Class != agents.Other || out.Detail == "" || strings.Contains(out.Detail, chatToken) {
		t.Fatalf("a failed turn %+v %v", out, err)
	}
	if got := transcript.String(); !strings.Contains(got, "[redacted]") || strings.Contains(got, chatToken) {
		t.Errorf("the transcript holds the token or no redaction: %s", got)
	}
}

func TestAChatNeedsItsSettings(t *testing.T) {
	rt := newRuntime(t)
	spec := chatSpec(t)
	spec.Network = ""
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); err == nil {
		t.Error("a chat without the provider's network started")
	}
	spec = chatSpec(t)
	spec.Skill = "poetry"
	if _, err := agents.StartChat(context.Background(), rt, claude.Provider{}, spec); err == nil {
		t.Error("a chat with an unknown skill started")
	}
}

func TestImportPathQuotesWhatFastImportNeeds(t *testing.T) {
	for in, want := range map[string]string{
		"a/b.md":      `"a/b.md"`,
		`a "q".md`:    `"a \"q\".md"`,
		`back\slash`:  `"back\\slash"`,
		"line\nbreak": `"line\nbreak"`,
		"tab\there":   `"tab\011here"`,
		"größe.md":    `"größe.md"`,
	} {
		if got := agents.ImportPath(in); got != want {
			t.Errorf("%q: %s, want %s", in, got, want)
		}
	}
}
