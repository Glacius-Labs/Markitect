package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func flagValue(argv []string, flag string) string {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag {
			return argv[i+1]
		}
	}
	return ""
}

// Open admits the Manager, installs the harness and the access-only login
// through private initialization and checks the CLI's login before any
// inference.
func TestOpenInstallsHarnessAndAccessLoginThenChecksTheLogin(t *testing.T) {
	rt := newRuntime()
	a := agentsFor(t, rt, credentialsLogin())
	spec := testSessionSpec(t)
	s := openSession(t, a, spec)
	if !slices.Equal(rt.calls, []string{"initialize", "start", "stop"}) {
		t.Fatalf("startup sequence %v", rt.calls)
	}
	home := rt.setup.Claude
	if rt.setup.CodexAuth != nil || home == nil || home.SystemPrompt != s.document.Base || home.AppendSystemPrompt != s.document.Developer {
		t.Fatal("the native instructions are not the rendered harness")
	}
	if strings.Contains(string(home.Credentials), "refresh") || !strings.Contains(string(home.Credentials), "sk-ant-oat01-private-access") {
		t.Fatalf("Manager login copy: %s", home.Credentials)
	}
	if !strings.Contains(home.AppendSystemPrompt, "SELECTED MANAGER GUIDANCE ✓") || !strings.Contains(home.AppendSystemPrompt, "Literal ${NO_EXPANSION}") {
		t.Fatal("selected Manager and profile instructions are missing from the developer instructions")
	}
	check := rt.processes[0].start
	if !slices.Equal(check.Argv, []string{claudeExecutable, "auth", "status"}) || check.Dir != privateHome || len(check.Tools) != 0 || !slices.Contains(check.Env, "CLAUDE_CONFIG_DIR=/home/agent/.claude") {
		t.Fatalf("login check %+v", check)
	}
	mounts := rt.spec.Mounts
	if len(mounts) != 2 || mounts[0].Volume != "private-run-volume" || mounts[0].Subpath != "copies/root" || mounts[0].Target != "/work" || !mounts[0].NoCopy || mounts[1].Target != privateHome || mounts[1].Volume == mounts[0].Volume || rt.spec.User != "1000:1000" || rt.spec.Image != fixtureImage {
		t.Fatalf("container spec %+v", rt.spec)
	}
	if obs := s.Observation(); obs.ProfileIdentity != a.identity || obs.ImageID != fixtureImage || obs.HarnessDigest != s.document.Digest() {
		t.Fatalf("observation %+v", obs)
	}
}

// One Manager is one conversation: the first response creates it under the
// adapter's session ID and later ones resume it, each process with exactly
// one user message and the full launch controls.
func TestExchangesResumeOneConversationWithOneMessageEach(t *testing.T) {
	rt := newRuntime()
	spec := testSessionSpec(t)
	decisions := judgments(&spec)
	s := openSession(t, agentsFor(t, rt, credentialsLogin()), spec)
	for _, text := range []string{"first assignment", "second assignment"} {
		if err := s.Exchange(context.Background(), task(text)); err != nil {
			t.Fatal(err)
		}
		if !s.Acted() {
			t.Fatal("a completed Run tool use is an action")
		}
	}
	responses := rt.responses()
	if len(responses) != 2 || len(*decisions) != 2 {
		t.Fatalf("responses=%d judgments=%v", len(responses), *decisions)
	}
	first, second := responses[0].start.Argv, responses[1].start.Argv
	if flagValue(first, "--session-id") != s.session || flagValue(first, "--resume") != "" || flagValue(second, "--resume") != s.session || flagValue(second, "--session-id") != "" {
		t.Fatalf("conversation flags %v / %v", first, second)
	}
	want := providerArgv(s.session, true, s.profile.Session)
	if !slices.Equal(second, want) {
		t.Fatalf("launch %v", second)
	}
	for _, flag := range []string{"--print", "--verbose", "--strict-mcp-config", "--disable-slash-commands"} {
		if !slices.Contains(second, flag) {
			t.Fatal("missing launch control", flag)
		}
	}
	for flag, value := range map[string]string{"--input-format": "stream-json", "--output-format": "stream-json", "--model": "sonnet", "--effort": "high", "--system-prompt-file": systemPromptPath, "--append-system-prompt-file": appendSystemPromptPath, "--permission-mode": "bypassPermissions", "--permission-prompts": "none", "--tools": builtinTools, "--setting-sources": "user", "--mcp-config": mcpConfig} {
		if got := flagValue(second, flag); got != value {
			t.Fatalf("%s = %q, want %q", flag, got, value)
		}
	}
	for i, p := range responses {
		var message struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct{ Text string } `json:"content"`
			} `json:"message"`
		}
		if strings.Count(string(p.input), "\n") != 1 || json.Unmarshal(p.input, &message) != nil || message.Type != "user" || len(message.Message.Content) != 1 {
			t.Fatalf("response %d input %q", i, p.input)
		}
		var delivered run.Message
		if err := json.Unmarshal([]byte(message.Message.Content[0].Text), &delivered); err != nil || delivered.Events[0].Assignment == nil {
			t.Fatalf("response %d does not carry Run's message: %v", i, err)
		}
		if p.start.Dir != "/work" || len(p.start.Tools) != 9 || !slices.Contains(p.start.Env, "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1") || !slices.Contains(p.start.Env, "ENABLE_TOOL_SEARCH=false") {
			t.Fatalf("response %d start %+v", i, p.start)
		}
	}
	// Every response ends with collected exit and an empty census.
	if got := strings.Count(strings.Join(rt.calls, ","), "census"); got != 2 {
		t.Fatalf("census after each response: %v", rt.calls)
	}
}

type contextProbe struct{}

// Run serves tool calls with the response's context, so its provider idle
// clock pauses while a call is served. The probe stands for that clock.
func TestToolCallsAreServedWithTheExchangeContext(t *testing.T) {
	rt := newRuntime()
	spec := testSessionSpec(t)
	seen := ""
	spec.Tools.Judge = func(ctx context.Context, d run.Decision, _ []run.Finding) error {
		seen, _ = ctx.Value(contextProbe{}).(string)
		return nil
	}
	s := openSession(t, agentsFor(t, rt, credentialsLogin()), spec)
	ctx := context.WithValue(context.Background(), contextProbe{}, "exchange")
	if err := s.Exchange(ctx, task("judge")); err != nil {
		t.Fatal(err)
	}
	if seen != "exchange" {
		t.Fatal("the tool call was not served with the Exchange context")
	}
	if len(rt.replies) != 1 || !strings.Contains(string(rt.replies[0].Result), `\"ok\":true`) {
		t.Fatalf("tool reply %+v", rt.replies)
	}
}

func TestActedCountsCompletedToolUsesOnly(t *testing.T) {
	for name, tc := range map[string]struct {
		lines []string
		acted bool
	}{
		"text only":          {[]string{fixtureAssistant("SESSION", textBlock("Thinking aloud.")), fixtureResult("SESSION", "done")}, false},
		"completed shell":    {[]string{fixtureAssistant("SESSION", toolUse("toolu_1", "Bash")), fixtureToolResult("SESSION", "toolu_1"), fixtureResult("SESSION", "done")}, true},
		"server web search":  {[]string{fixtureAssistant("SESSION", `{"type":"server_tool_use","id":"srv_1","name":"web_search","input":{}}`, `{"type":"web_search_tool_result","tool_use_id":"srv_1","content":[]}`), fixtureResult("SESSION", "done")}, true},
		"thinking and texts": {[]string{fixtureAssistant("SESSION", `{"type":"thinking","thinking":"x","signature":"s"}`), fixtureResult("SESSION", "done")}, false},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.play = lines(0, append([]string{fixtureInit("SESSION", connectedServers)}, tc.lines...)...)
			s := openSession(t, agentsFor(t, rt, credentialsLogin()), testSessionSpec(t))
			if err := s.Exchange(context.Background(), task("work")); err != nil {
				t.Fatal(err)
			}
			if s.Acted() != tc.acted {
				t.Fatalf("Acted=%v, want %v", s.Acted(), tc.acted)
			}
		})
	}
}

// The token login reaches only the provider environment; transcripts and
// errors carry neither token form.
func TestTokenLoginAndTranscriptRedaction(t *testing.T) {
	rt := newRuntime()
	rt.authStatus = `{"loggedIn":true,"authMethod":"oauth_token"}`
	rt.play = lines(0, fixtureInit("SESSION", connectedServers), fixtureAssistant("SESSION", textBlock("leaked "+fixtureToken)), fixtureResult("SESSION", "done "+fixtureToken))
	a := agentsFor(t, rt, RuntimeCredential{Mode: ModeToken, Login: []byte(fixtureToken + "\n")})
	a.TranscriptRoot = t.TempDir()
	s := openSession(t, a, testSessionSpec(t))
	if rt.setup.Claude.Credentials != nil {
		t.Fatal("a token login installed a credentials file")
	}
	if err := s.Exchange(context.Background(), task("work")); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(rt.responses()[0].start.Env, "CLAUDE_CODE_OAUTH_TOKEN="+fixtureToken) {
		t.Fatal("the token did not reach the provider environment")
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(a.TranscriptRoot)
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	data, err := os.ReadFile(filepath.Join(a.TranscriptRoot, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-long-lived") || !strings.Contains(string(data), "[REDACTED]") {
		t.Fatal("transcript leaked the token or redacted nothing")
	}
	for _, direction := range []string{`"setup"`, `"start"`, `"login"`, `"send"`, `"receive"`, `"exit"`} {
		if !strings.Contains(string(data), `"direction":`+direction) {
			t.Fatal("transcript lacks", direction)
		}
	}
	if strings.Contains(string(data), "someone@example.invalid") {
		t.Fatal("the login status output reached the transcript")
	}
}

// The login check refuses an API key, an API-key helper or another provider
// before any inference, and cleanup still runs.
func TestOtherLoginMethodsAreRefusedBeforeInference(t *testing.T) {
	for _, method := range []string{"api_key", "api_key_helper", "third_party", "none", "oauth_token"} {
		t.Run(method, func(t *testing.T) {
			rt := newRuntime()
			rt.authStatus = `{"authMethod":"` + method + `"}`
			s, err := agentsFor(t, rt, credentialsLogin()).Open(context.Background(), testSessionSpec(t))
			if err == nil || !strings.Contains(err.Error(), method) || s == nil {
				t.Fatalf("login method %s admitted: %v", method, err)
			}
			if err := s.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(rt.responses()) != 0 || !slices.Contains(rt.calls, "clear") || !slices.Contains(rt.calls, "close") {
				t.Fatalf("calls %v", rt.calls)
			}
		})
	}
}

func TestAdmissionRefusesDirectoryForgedManagerAndOversizedHarnessBeforeContainers(t *testing.T) {
	spec := testSessionSpec(t)
	rt := newRuntime()
	a := agentsFor(t, rt, credentialsLogin())
	calls := 0
	resolve := a.credentials
	a.credentials = func(ctx context.Context, ref string) (RuntimeCredential, error) { calls++; return resolve(ctx, ref) }
	spec.Workspace = &sessionCopy{location: run.CopyLocation{Directory: t.TempDir()}}
	if s, err := a.Open(context.Background(), spec); err == nil || s != nil || calls != 0 {
		t.Fatal("Directory admitted")
	}
	spec.Workspace = &sessionCopy{location: run.CopyLocation{Volume: "volume", Subpath: "copy"}}
	forged := spec
	forged.Manager.Children = nil
	if s, err := a.Open(context.Background(), forged); err == nil || s != nil || calls != 0 {
		t.Fatal("forged Manager admitted")
	}
	a.profile.Session.DeveloperInstructions = strings.Repeat("x", frame.MaxPayload)
	s, err := a.Open(context.Background(), spec)
	if err == nil || s != nil || !strings.Contains(err.Error(), "private setup frame") || rt.spec.Name != "" {
		t.Fatalf("oversized harness: %v, container %q", err, rt.spec.Name)
	}
}

func TestStreamViolationsEndTheExchange(t *testing.T) {
	for name, tc := range map[string]struct {
		play func(*fakeProcess)
		want string
	}{
		"no markitect server":  {lines(0, fixtureInit("SESSION", `[{"name":"other","status":"connected"}]`), fixtureResult("SESSION", "done")), "without the markitect MCP server"},
		"failed server":        {lines(0, fixtureInit("SESSION", `[{"name":"markitect","status":"failed"}]`), fixtureResult("SESSION", "done")), `"failed"`},
		"native subagent":      {lines(0, fixtureInit("SESSION", connectedServers), strings.Replace(fixtureAssistant("SESSION", textBlock("child")), `"parent_tool_use_id":null`, `"parent_tool_use_id":"toolu_agent"`, 1), fixtureResult("SESSION", "done")), "subagent"},
		"control request":      {lines(0, fixtureInit("SESSION", connectedServers), `{"type":"control_request","request_id":"r","request":{"subtype":"can_use_tool"}}`), "control request"},
		"other session":        {lines(0, fixtureInit("00000000-0000-4000-8000-0000000000ff", connectedServers)), "another session"},
		"missing result":       {lines(0, fixtureInit("SESSION", connectedServers)), "ended before the response finished"},
		"failed exit":          {lines(1, fixtureInit("SESSION", connectedServers), fixtureResult("SESSION", "done")), "exited with code 1"},
		"final provider error": {lines(1, append([]string{fixtureInit("SESSION", connectedServers)}, fixtureFailure("SESSION", "rate_limit")...)...), "rate_limit"},
		"malformed output":     {lines(0, fixtureInit("SESSION", connectedServers), `{"type":`), ErrProtocol.Error()},
		"unfinished line":      {func(p *fakeProcess) { p.stdout.chunks <- []byte(`{"type":"result"`); p.end(0) }, ErrProtocolIncomplete.Error()},
	} {
		t.Run(name, func(t *testing.T) {
			rt := newRuntime()
			rt.play = tc.play
			s := openSession(t, agentsFor(t, rt, credentialsLogin()), testSessionSpec(t))
			err := s.Exchange(context.Background(), task("work"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if !rt.closed || !slices.Contains(rt.calls, "clear") {
				t.Fatalf("a failed Exchange closes its session: %v", rt.calls)
			}
			if err := s.Exchange(context.Background(), task("again")); err == nil {
				t.Fatal("a closed session accepted another Exchange")
			}
		})
	}
}

// A detached process that outlives the provider could still write to the
// working copy, so the Exchange ends.
func TestDetachedProcessAfterExitEndsTheExchange(t *testing.T) {
	rt := newRuntime()
	rt.census = [][]frame.Process{{{PID: 40, PPID: 1, PGID: 40, State: "S", Command: "node server.js sk-ant-oat01-private-access"}}}
	s := openSession(t, agentsFor(t, rt, credentialsLogin()), testSessionSpec(t))
	err := s.Exchange(context.Background(), task("work"))
	if err == nil || !strings.Contains(err.Error(), "1 native processes remain") || strings.Contains(err.Error(), "private-access") {
		t.Fatalf("detached process: %v", err)
	}
	// A process that ended and awaits reaping, or one that ends in time, is quiescent.
	rt = newRuntime()
	rt.census = [][]frame.Process{{{PID: 41, State: "Z", Command: "sh"}, {PID: 42, State: "S", Command: "git"}}, {{PID: 41, State: "Z", Command: "sh"}}}
	s = openSession(t, agentsFor(t, rt, credentialsLogin()), testSessionSpec(t))
	if err := s.Exchange(context.Background(), task("work")); err != nil {
		t.Fatal(err)
	}
}

func TestCloseCancelsAnActiveExchangeAndStopsTheProcess(t *testing.T) {
	rt := newRuntime()
	entered := make(chan struct{})
	spec := testSessionSpec(t)
	spec.Tools.Await = func(ctx context.Context) (run.Event, error) {
		close(entered)
		<-ctx.Done()
		return run.Event{}, ctx.Err()
	}
	rt.play = func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		p.runTool("toolu_await", "await", "{}")
	}
	s := openSession(t, agentsFor(t, rt, credentialsLogin()), spec)
	done := make(chan error, 1)
	go func() { done <- s.Exchange(context.Background(), task("wait")) }()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("await not entered")
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled Exchange reported success")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not end the Exchange")
	}
	if !slices.Contains(rt.calls, "stop") || !slices.Contains(rt.calls, "clear") || !rt.closed {
		t.Fatalf("closure %v", rt.calls)
	}
}

func TestInvalidToolArgumentsAndUnknownToolsReachNoCallback(t *testing.T) {
	rt := newRuntime()
	spec := testSessionSpec(t)
	decisions := judgments(&spec)
	rt.play = func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		if !p.runTool("toolu_bad", "judge", `{"decision":"submitted","findings":[{"text":"x","extra":1}]}`) {
			return
		}
		p.emit(fixtureResult(p.session, "done"))
		p.end(0)
	}
	s := openSession(t, agentsFor(t, rt, credentialsLogin()), spec)
	if err := s.Exchange(context.Background(), task("judge")); err != nil {
		t.Fatal(err)
	}
	if len(*decisions) != 0 || len(rt.replies) != 1 || !strings.Contains(string(rt.replies[0].Result), `"isError":true`) {
		t.Fatalf("invalid arguments reached Run: %v %+v", *decisions, rt.replies)
	}
	rt = newRuntime()
	rt.play = func(p *fakeProcess) {
		p.emit(fixtureInit(p.session, connectedServers))
		p.call("host_shell", "{}")
	}
	s = openSession(t, agentsFor(t, rt, credentialsLogin()), testSessionSpec(t))
	if err := s.Exchange(context.Background(), task("judge")); !errors.Is(err, ErrProtocol) || len(rt.replies) != 0 {
		t.Fatalf("unknown tool: %v %+v", err, rt.replies)
	}
}

func TestToolSpecsCarryTheRunToolsWithClaudeMeta(t *testing.T) {
	var names []string
	for _, spec := range toolSpecs() {
		names = append(names, spec.Name)
		var meta map[string]any
		if json.Unmarshal(spec.Meta, &meta) != nil || meta["anthropic/maxResultSizeChars"] != float64(500000) || meta["anthropic/alwaysLoad"] != true {
			t.Fatalf("%s meta %s", spec.Name, spec.Meta)
		}
		var schema map[string]any
		if json.Unmarshal(spec.InputSchema, &schema) != nil || schema["additionalProperties"] != false {
			t.Fatalf("%s schema %s", spec.Name, spec.InputSchema)
		}
	}
	if !slices.Equal(names, []string{"delegate", "await", "review", "ask", "reply", "judge", "check", "read_notes", "append_note"}) {
		t.Fatal(names)
	}
}
