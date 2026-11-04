package agents_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
	"github.com/Glacius-Labs/Markitect/internal/agents/codex"
)

// --- a scripted shell ---

// execution is one command the shell was asked to run, with what it was fed.
type execution struct {
	agentport.Exec
	stdin string
}

// fakeShell plays the container's shell: the login report, and the stream-json
// output of a session, which the play function writes.
type fakeShell struct {
	mu    sync.Mutex
	execs []execution
	// auth plays `claude auth status`; the default is the token's login.
	auth func(e agentport.Exec) error
	// play plays a session; it ends when it returns.
	play func(ctx context.Context, e agentport.Exec) error
}

func (f *fakeShell) Exec(ctx context.Context, e agentport.Exec) error {
	var stdin string
	if e.Stdin != nil {
		raw, _ := io.ReadAll(e.Stdin)
		stdin = string(raw)
	}
	f.mu.Lock()
	f.execs = append(f.execs, execution{e, stdin})
	f.mu.Unlock()
	if len(e.Argv) >= 2 && e.Argv[len(e.Argv)-2] == "auth" && e.Argv[len(e.Argv)-1] == "status" {
		if f.auth != nil {
			return f.auth(e)
		}
		fmt.Fprintln(e.Stdout, `{"loggedIn":true,"authMethod":"oauth_token"}`)
		return nil
	}
	return f.play(ctx, e)
}

// sessions returns the executions that were sessions, not login checks.
func (f *fakeShell) sessions() []execution {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []execution
	for _, e := range f.execs {
		if !(len(e.Argv) >= 2 && e.Argv[len(e.Argv)-2] == "auth") {
			out = append(out, e)
		}
	}
	return out
}

func (f *fakeShell) checks() []execution {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []execution
	for _, e := range f.execs {
		if len(e.Argv) >= 2 && e.Argv[len(e.Argv)-2] == "auth" {
			out = append(out, e)
		}
	}
	return out
}

// lines plays stream-json lines.
func lines(w io.Writer, lines ...string) {
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}

// plays is a session that writes the lines and ends.
func plays(ls ...string) func(context.Context, agentport.Exec) error {
	return func(_ context.Context, e agentport.Exec) error {
		lines(e.Stdout, ls...)
		return nil
	}
}

// sessionInit is the init line of a session whose launch held as asked.
func sessionInit() string {
	return `{"type":"system","subtype":"init","session_id":"s-1","model":"claude-sonnet-5-5","claude_code_version":"` + claude.Version +
		`","apiKeySource":"none","permissionMode":"bypassPermissions","tools":["Agent","Bash","Edit","Glob","Grep","Read","TodoWrite","Write"],` +
		`"agents":["claude","Explore","general-purpose","Plan","statusline-setup"],"skills":[],"plugins":[],"slash_commands":[],"mcp_servers":[]}`
}

// sessionResult is the result line of a session; the per-model figures are the
// conversation's whole total.
func sessionResult(text string, input, created, read, output int) string {
	return fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"result":%q,"session_id":"s-1","usage":{"input_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1,"output_tokens":1},`+
		`"modelUsage":{"claude-sonnet-5-5":{"inputTokens":%d,"outputTokens":%d,"cacheReadInputTokens":%d,"cacheCreationInputTokens":%d}}}`, text, input, output, read, created)
}

const failedResult = `{"type":"result","subtype":"success","is_error":true,"result":"API Error","session_id":"s-1","api_error_status":null}`

func newSessions(t *testing.T, transcripts string) *agents.Sessions {
	t.Helper()
	g, err := agents.NewSessions(claude.Provider{}, "claude-sonnet-5-5", "high", chatToken, transcripts, 0)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func aSession() agentport.Session {
	return agentport.Session{Run: "r1", Role: agentport.Implementer, User: agentport.ImplementerUser, Round: 1, Dir: "/work", Prompt: "Realize the change.", Deadline: time.Now().Add(time.Minute)}
}

func TestAFinishedSessionReturnsItsConversationClosingWordsAndWhatItSaw(t *testing.T) {
	sh := &fakeShell{play: plays(sessionInit(),
		`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m1","model":"claude-sonnet-5-5","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/work/README.md"}},{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]},"session_id":"s-1"}`,
		`{"type":"assistant","parent_tool_use_id":"toolu_1","message":{"id":"m2","model":"claude-sonnet-5-5","content":[{"type":"tool_use","name":"Bash","input":{"command":"git status"}}]},"session_id":"s-1"}`,
		sessionResult("The change is made.", 10, 20, 30, 4))}
	end := newSessions(t, "").Run(context.Background(), sh, aSession())
	if end.Reason != agentport.Finished || end.Class != "" || end.Text != "The change is made." || end.Conversation != "s-1" || end.Agent != "claude:claude-sonnet-5-5" {
		t.Fatalf("end %+v", end)
	}
	if !slices.Equal(end.Models, []string{"claude-sonnet-5-5"}) {
		t.Errorf("models %v", end.Models)
	}
	if !end.Seen.Measured || !slices.Equal(end.Seen.Opened, []string{"/work/README.md"}) || !slices.Equal(end.Seen.Commands, []string{"go test ./...", "git status"}) {
		t.Errorf("seen %+v", end.Seen)
	}
	if end.Usage != (agentport.Usage{InputTokens: 60, OutputTokens: 4}) {
		t.Errorf("usage %+v", end.Usage)
	}
}

func TestTheTokenGoesInTheSecretsAndNeverInTheEnvironmentOrTheCommandLine(t *testing.T) {
	sh := &fakeShell{play: plays(sessionInit(), sessionResult("ok", 1, 1, 1, 1))}
	s := aSession()
	s.Role = agentport.Reviewer
	if end := newSessions(t, "").Run(context.Background(), sh, s); end.Reason != agentport.Finished {
		t.Fatalf("end %+v", end)
	}
	want := "CLAUDE_CODE_OAUTH_TOKEN=" + chatToken
	if len(sh.checks()) != 1 || len(sh.sessions()) != 1 {
		t.Fatalf("executions %+v", sh.execs)
	}
	for _, e := range sh.execs {
		if !slices.Equal(e.Secrets, []string{want}) {
			t.Errorf("%q: secrets %q", e.Argv, e.Secrets)
		}
		if strings.Contains(strings.Join(e.Env, "\n"), chatToken) || strings.Contains(strings.Join(e.Argv, " "), chatToken) || strings.Contains(e.stdin, chatToken) {
			t.Errorf("%q carries the token outside the secrets: env %q stdin %q", e.Argv, e.Env, e.stdin)
		}
		if e.User != agentport.ImplementerUser {
			t.Errorf("%q runs as %q", e.Argv, e.User)
		}
	}
	// The session runs in the Command's folder, with the prompt on its input;
	// the login check runs in the role's home.
	run := sh.sessions()[0]
	if run.Dir != "/work" || !strings.Contains(run.stdin, "Realize the change.") || !slices.Contains(run.Env, "CLAUDE_CONFIG_DIR=/home/implementer/.claude") {
		t.Errorf("session %+v", run)
	}
	if check := sh.checks()[0]; check.Dir != "/home/implementer" || !slices.Contains(check.Env, "CLAUDE_CONFIG_DIR=/home/implementer/.claude") {
		t.Errorf("login check %+v", check)
	}
}

func TestTheLoginIsCheckedOncePerRoleAndRun(t *testing.T) {
	sh := &fakeShell{play: plays(sessionInit(), sessionResult("ok", 1, 1, 1, 1))}
	g := newSessions(t, "")
	run := func(runID string, role agentport.Role, round int) {
		s := aSession()
		s.Run, s.Role, s.Round = runID, role, round
		s.User = agentport.ImplementerUser
		if role == agentport.Reviewer {
			s.User = agentport.ReviewerUser(1)
		}
		if end := g.Run(context.Background(), sh, s); end.Reason != agentport.Finished {
			t.Fatalf("%s %s %d: %+v", runID, role, round, end)
		}
	}
	run("r1", agentport.Implementer, 1)
	run("r1", agentport.Implementer, 2)
	if len(sh.checks()) != 1 {
		t.Fatalf("a second session of the role checked the login again: %d checks", len(sh.checks()))
	}
	run("r1", agentport.Reviewer, 1)
	if len(sh.checks()) != 2 {
		t.Fatalf("the other role did not check its login: %d checks", len(sh.checks()))
	}
	run("r2", agentport.Implementer, 1)
	if len(sh.checks()) != 3 || len(sh.sessions()) != 4 {
		t.Fatalf("a new Run did not check the login: %d checks, %d sessions", len(sh.checks()), len(sh.sessions()))
	}
}

func TestARefusedLoginIsALoginFailureAndStartsNoSession(t *testing.T) {
	for name, auth := range map[string]func(agentport.Exec) error{
		"api key": func(e agentport.Exec) error {
			fmt.Fprintln(e.Stdout, `{"loggedIn":true,"authMethod":"api_key"}`)
			return nil
		},
		"not logged in": func(e agentport.Exec) error {
			fmt.Fprintln(e.Stdout, `{"loggedIn":false,"authMethod":"none"}`)
			return nil
		},
		"no report": func(e agentport.Exec) error {
			fmt.Fprintln(e.Stderr, "cannot reach the account with "+chatToken)
			return &agentport.ExitError{Code: 1}
		},
	} {
		sh := &fakeShell{auth: auth, play: plays(sessionInit(), sessionResult("ok", 1, 1, 1, 1))}
		g := newSessions(t, "")
		end := g.Run(context.Background(), sh, aSession())
		if end.Reason != agentport.Failed || end.Class != agentport.Login || end.Detail == "" || strings.Contains(end.Detail, chatToken) {
			t.Errorf("%s: end %+v", name, end)
		}
		if len(sh.sessions()) != 0 {
			t.Errorf("%s: a session started on a refused login", name)
		}
		// A refused login is not remembered as checked.
		sh.auth = nil
		if end := g.Run(context.Background(), sh, aSession()); end.Reason != agentport.Finished {
			t.Errorf("%s: the next session after a good login: %+v", name, end)
		}
	}
}

func TestSessionsNeedAProviderWithATokenRouteAndTheUsersToken(t *testing.T) {
	if _, err := agents.NewSessions(claude.Provider{}, "claude-sonnet-5-5", "high", "", "", 0); err == nil {
		t.Error("sessions without a token")
	}
	if _, err := agents.NewSessions(codex.Provider{}, "gpt-6-sol", "high", chatToken, "", 0); err == nil || !strings.Contains(err.Error(), "codex") {
		t.Errorf("sessions on a provider with no token route: %v", err)
	}
	if _, err := agents.NewSessions(nil, "claude-sonnet-5-5", "high", chatToken, "", 0); err == nil {
		t.Error("sessions without a provider")
	}
	if _, err := agents.NewSessions(claude.Provider{}, "", "high", chatToken, "", 0); err == nil {
		t.Error("sessions without a model")
	}
	if _, err := agents.NewSessions(claude.Provider{}, "claude-sonnet-5-5", "", chatToken, "", 0); err == nil {
		t.Error("sessions without an effort")
	}
}

func TestABreachOfTheLaunchContractFailsTheSessionAndStopsIt(t *testing.T) {
	stopped := make(chan struct{})
	sh := &fakeShell{play: func(ctx context.Context, e agentport.Exec) error {
		lines(e.Stdout, strings.Replace(sessionInit(), `"mcp_servers":[]`, `"mcp_servers":[{"name":"project-db","status":"connected"}]`, 1))
		// The shell ends the process when the session's context ends.
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	}}
	end := newSessions(t, "").Run(context.Background(), sh, aSession())
	if end.Reason != agentport.Failed || end.Class != agentport.Other || !strings.Contains(end.Detail, "project-db") {
		t.Fatalf("end %+v", end)
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the session was not stopped")
	}
}

func TestAFailedResultKeepsItsFailureClass(t *testing.T) {
	for _, c := range []struct {
		error string
		want  agentport.FailureClass
	}{
		{"overloaded", agentport.Capacity},
		{"rate_limit", agentport.RateLimit},
		{"server_error", agentport.ServerError},
		{"authentication_failed", agentport.Login},
		{"invalid_request", agentport.Other},
	} {
		sh := &fakeShell{play: plays(sessionInit(), `{"type":"assistant","parent_tool_use_id":null,"error":"`+c.error+`","message":{}}`, failedResult)}
		end := newSessions(t, "").Run(context.Background(), sh, aSession())
		if end.Reason != agentport.Failed || end.Class != c.want || !strings.Contains(end.Detail, "API Error") {
			t.Errorf("%s: end %+v", c.error, end)
		}
	}

	// A session without any result is the host's failure, not the provider's.
	sh := &fakeShell{play: plays(sessionInit())}
	end := newSessions(t, "").Run(context.Background(), sh, aSession())
	if end.Reason != agentport.Failed || end.Class != agentport.Other || !strings.Contains(end.Detail, "without a result") {
		t.Errorf("no result: %+v", end)
	}
	sh = &fakeShell{play: func(_ context.Context, e agentport.Exec) error {
		lines(e.Stdout, sessionInit())
		fmt.Fprintln(e.Stderr, "Segmentation fault")
		return &agentport.ExitError{Code: 139}
	}}
	end = newSessions(t, "").Run(context.Background(), sh, aSession())
	if end.Reason != agentport.Failed || end.Class != agentport.Other || !strings.Contains(end.Detail, "exit status 139") || !strings.Contains(end.Detail, "Segmentation fault") {
		t.Errorf("a crash: %+v", end)
	}
}

func TestADeadlineEndsTheSessionAsDeadlineReached(t *testing.T) {
	sh := &fakeShell{play: func(ctx context.Context, e agentport.Exec) error {
		lines(e.Stdout, sessionInit())
		<-ctx.Done()
		return ctx.Err()
	}}
	s := aSession()
	s.Deadline = time.Now().Add(200 * time.Millisecond)
	end := newSessions(t, "").Run(context.Background(), sh, s)
	if end.Reason != agentport.DeadlineReached || end.Detail == "" {
		t.Fatalf("end %+v", end)
	}
	// The session had started, so its conversation is kept for a later one.
	if end.Conversation != "s-1" {
		t.Errorf("the conversation of a stopped session is %q", end.Conversation)
	}

	// A deadline that passed before the session started never starts it.
	sh = &fakeShell{auth: func(e agentport.Exec) error { return errors.New("the command could not run") }, play: plays(sessionInit())}
	s.Deadline = time.Now().Add(-time.Second)
	end = newSessions(t, "").Run(context.Background(), sh, s)
	if end.Reason != agentport.DeadlineReached || len(sh.sessions()) != 0 {
		t.Errorf("a late start %+v", end)
	}
}

func TestACancelledContextEndsTheSessionAsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sh := &fakeShell{play: func(ctx context.Context, e agentport.Exec) error {
		lines(e.Stdout, sessionInit())
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}}
	end := newSessions(t, "").Run(ctx, sh, aSession())
	if end.Reason != agentport.Cancelled || end.Detail == "" {
		t.Fatalf("end %+v", end)
	}

	// A context that ended before the session started never starts it.
	done, stop := context.WithCancel(context.Background())
	stop()
	sh = &fakeShell{auth: func(e agentport.Exec) error { return errors.New("the command could not run") }, play: plays(sessionInit())}
	end = newSessions(t, "").Run(done, sh, aSession())
	if end.Reason != agentport.Cancelled || len(sh.sessions()) != 0 {
		t.Errorf("a cancelled start %+v", end)
	}
}

func TestTheTokenNeverAppearsInTheTranscriptOrTheEnd(t *testing.T) {
	dir := t.TempDir()
	token := chatToken
	assistant := func(content string) string {
		return `{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m1","model":"claude-sonnet-5-5","content":[` + content + `]},"session_id":"s-1"}`
	}
	say := assistant(`{"type":"text","text":"The token is ` + token + `."},{"type":"tool_use","name":"Bash","input":{"command":"echo ` + token + `"}}`)

	finished := &fakeShell{play: plays(sessionInit(), say, sessionResult("Done with "+token, 1, 1, 1, 1))}
	end := newSessions(t, dir).Run(context.Background(), finished, aSession())
	failed := &fakeShell{play: plays(sessionInit(), say, strings.Replace(failedResult, "API Error", "API Error for "+token, 1))}
	s := aSession()
	s.Round = 2
	endFailed := newSessions(t, dir).Run(context.Background(), failed, s)

	if end.Reason != agentport.Finished || endFailed.Reason != agentport.Failed {
		t.Fatalf("ends %+v %+v", end, endFailed)
	}
	for name, got := range map[string]string{
		"text": end.Text, "detail": endFailed.Detail, "commands": strings.Join(end.Seen.Commands, "\n"), "failed commands": strings.Join(endFailed.Seen.Commands, "\n"),
	} {
		if strings.Contains(got, token) {
			t.Errorf("the %s holds the token: %q", name, got)
		}
	}
	if !strings.Contains(end.Text, "[redacted]") || !strings.Contains(endFailed.Detail, "[redacted]") || end.Seen.Commands[0] != "echo [redacted]" {
		t.Errorf("redaction left no mark: %q %q %q", end.Text, endFailed.Detail, end.Seen.Commands)
	}
	for _, name := range []string{"implementer-1.jsonl", "implementer-2.jsonl"} {
		raw, err := os.ReadFile(filepath.Join(dir, "r1", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), token) || !strings.Contains(string(raw), "[redacted]") || !strings.Contains(string(raw), `"subtype":"init"`) {
			t.Errorf("%s: %s", name, raw)
		}
	}
}

func TestAContinuedSessionCountsOnlyWhatItAddedToTheConversation(t *testing.T) {
	g := newSessions(t, "")
	play := func(result string) *fakeShell {
		return &fakeShell{play: plays(sessionInit(), result)}
	}
	first := g.Run(context.Background(), play(sessionResult("one", 10, 20, 30, 4)), aSession())
	if first.Usage != (agentport.Usage{InputTokens: 60, OutputTokens: 4}) {
		t.Fatalf("the first session %+v", first)
	}

	// The continued session reports the conversation's whole total.
	s := aSession()
	s.Round, s.Conversation = 2, first.Conversation
	second := g.Run(context.Background(), play(sessionResult("two", 12, 25, 60, 9)), s)
	if second.Reason != agentport.Finished || second.Usage != (agentport.Usage{InputTokens: 37, OutputTokens: 5}) {
		t.Fatalf("the continued session %+v", second)
	}
	s.Round = 3
	third := g.Run(context.Background(), play(sessionResult("three", 12, 25, 70, 19)), s)
	if third.Usage != (agentport.Usage{InputTokens: 10, OutputTokens: 10}) {
		t.Fatalf("the next continued session %+v", third)
	}

	// A total that fell counts as nothing rather than a negative number.
	s.Round = 4
	fell := g.Run(context.Background(), play(sessionResult("four", 1, 1, 1, 1)), s)
	if fell.Usage != (agentport.Usage{}) {
		t.Errorf("a total that fell %+v", fell)
	}

	// Another conversation starts from zero.
	reviewer := aSession()
	reviewer.Role = agentport.Reviewer
	elsewhere := &fakeShell{play: plays(strings.ReplaceAll(sessionInit(), "s-1", "s-2"), strings.ReplaceAll(sessionResult("other", 1, 1, 1, 1), "s-1", "s-2"))}
	other := g.Run(context.Background(), elsewhere, reviewer)
	if other.Usage != (agentport.Usage{InputTokens: 3, OutputTokens: 1}) {
		t.Errorf("another conversation %+v", other)
	}
}
