package claude

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agents"
)

func launch() agents.Launch {
	return agents.Launch{Home: "/home/implementer", Prompt: "Realize the change.", Model: "claude-sonnet-5-5", Effort: "high"}
}

// option returns the argument that follows flag.
func option(t *testing.T, argv []string, flag string) string {
	t.Helper()
	i := slices.Index(argv, flag)
	if i < 0 || i+1 >= len(argv) {
		t.Fatalf("argv lacks %s: %q", flag, argv)
	}
	return argv[i+1]
}

func TestLaunchKeepsEverythingOfTheRepositoryOut(t *testing.T) {
	argv, _, stdin, err := Provider{}.Launch(launch())
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"--print --input-format stream-json --output-format stream-json --verbose", "--model claude-sonnet-5-5 --effort high",
		"--permission-mode bypassPermissions", "--permission-prompts none", "--disable-slash-commands", "--setting-sources user", "--strict-mcp-config"} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv lacks %q", want)
		}
	}
	// Only the user's settings load, the project's .mcp.json connects nothing,
	// and the launch brings no MCP server of its own.
	for _, not := range []string{"--bare", "--mcp-config", "--resume", "--system-prompt-file", "--append-system-prompt-file"} {
		if slices.Contains(argv, not) {
			t.Errorf("argv holds %s", not)
		}
	}
	// A session may use its own subagents; the Agent tool is on the list.
	tools := strings.Split(option(t, argv, "--tools"), ",")
	for _, want := range []string{"Bash", "Read", "Edit", "Write", "Glob", "Grep", "Agent"} {
		if !slices.Contains(tools, want) {
			t.Errorf("the tools %q lack %s", tools, want)
		}
	}
	// The settings switch every hook off, whatever a settings file says.
	var settings struct {
		DisableAllHooks bool            `json:"disableAllHooks"`
		EnabledPlugins  map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal([]byte(option(t, argv, "--settings")), &settings); err != nil || !settings.DisableAllHooks {
		t.Fatalf("settings %q: %v", option(t, argv, "--settings"), err)
	}
	if on, ok := settings.EnabledPlugins["cc-plugin-agents-md@builtin"]; !ok || on {
		t.Errorf("the built-in plugins are not switched off: %v", settings.EnabledPlugins)
	}
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(stdin, &msg) != nil || msg.Type != "user" || msg.Message.Content[0].Text != "Realize the change." || !strings.HasSuffix(string(stdin), "\n") {
		t.Fatalf("stdin %s", stdin)
	}
}

func TestLaunchForcesSubagentsOntoTheModelAndHoldsNoToken(t *testing.T) {
	_, env, _, err := Provider{}.Launch(launch())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CLAUDE_CODE_SUBAGENT_MODEL=claude-sonnet-5-5", "CLAUDE_CODE_SUBAGENT_MODEL_FORCE=1", "CLAUDE_CONFIG_DIR=/home/implementer/.claude", "HOME=/home/implementer",
		"CLAUDE_CODE_DISABLE_CLAUDE_MDS=1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"} {
		if !slices.Contains(env, want) {
			t.Errorf("env lacks %q: %q", want, env)
		}
	}
	// The token reaches a session only through the Command's secrets.
	joined := strings.Join(env, " ")
	if strings.Contains(joined, tokenVariable) || strings.Contains(joined, "ANTHROPIC_API_KEY") || strings.Contains(joined, "ANTHROPIC_AUTH_TOKEN") || strings.Contains(joined, "TOKEN") {
		t.Errorf("env carries a credential variable: %q", env)
	}

	// Each role has a configuration folder of its own below its home.
	_, env, _, err = Provider{}.Launch(agents.Launch{Home: "/home/reviewer", Prompt: "p", Model: "claude-haiku-5", Effort: "low"})
	if err != nil || !slices.Contains(env, "CLAUDE_CONFIG_DIR=/home/reviewer/.claude") || !slices.Contains(env, "CLAUDE_CODE_SUBAGENT_MODEL=claude-haiku-5") {
		t.Fatalf("the reviewer's env %q %v", env, err)
	}
}

func TestLaunchContinuesAConversationAndRefusesBadSettings(t *testing.T) {
	l := launch()
	l.Conversation = "s-1"
	argv, _, _, err := Provider{}.Launch(l)
	if err != nil || option(t, argv, "--resume") != "s-1" {
		t.Fatalf("resume %q %v", argv, err)
	}
	for name, bad := range map[string]agents.Launch{
		"effort":       {Home: "/h", Model: "m", Effort: "ultra"},
		"no model":     {Home: "/h", Effort: "high"},
		"model spaces": {Home: "/h", Model: "two words", Effort: "high"},
	} {
		if _, _, _, err := (Provider{}).Launch(bad); err == nil {
			t.Errorf("%s: launched", name)
		}
	}
}

func TestTheTokenReachesTheCLIThroughItsOwnVariable(t *testing.T) {
	if v := (Provider{}).TokenVariable(); v != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Errorf("the token variable is %q", v)
	}
}

func TestAuthStatusUsesTheSessionUsersHome(t *testing.T) {
	argv, env := Provider{}.AuthStatus("/home/reviewer")
	if len(argv) != 3 || argv[1] != "auth" || argv[2] != "status" {
		t.Fatalf("auth status %q", argv)
	}
	if !slices.Contains(env, "CLAUDE_CONFIG_DIR=/home/reviewer/.claude") || !slices.Contains(env, "HOME=/home/reviewer") {
		t.Errorf("env %q", env)
	}
	if strings.Contains(strings.Join(env, " "), tokenVariable) {
		t.Errorf("env carries the token: %q", env)
	}
}

// sessionInit is the init line of a session whose launch held as asked.
const sessionInit = `{"type":"system","subtype":"init","session_id":"s-1","model":"claude-sonnet-5-5","claude_code_version":"2.1.308","apiKeySource":"none","permissionMode":"bypassPermissions","tools":["Agent","Bash","Edit","Glob","Grep","Read","TodoWrite","Write"],"agents":["claude","Explore","general-purpose","Plan","statusline-setup"],"skills":[],"plugins":[],"slash_commands":[],"mcp_servers":[]}`

func readSession(t *testing.T, l agents.Launch, lines ...string) (agents.SessionReader, error) {
	t.Helper()
	r := Provider{}.SessionReader(l)
	for _, line := range lines {
		if err := r.Line([]byte(line)); err != nil {
			return r, err
		}
	}
	return r, nil
}

func TestSessionReaderRefusesWhatTheRepositoryMustNotBring(t *testing.T) {
	for name, init := range map[string]string{
		"mcp server":       strings.Replace(sessionInit, `"mcp_servers":[]`, `"mcp_servers":[{"name":"project-db","status":"connected"}]`, 1),
		"mcp tool":         strings.Replace(sessionInit, `"Write"]`, `"Write","mcp__project-db__query"]`, 1),
		"project subagent": strings.Replace(sessionInit, `"agents":["claude"`, `"agents":["reviewer-from-project","claude"`, 1),
		"skill":            strings.Replace(sessionInit, `"skills":[]`, `"skills":["deploy"]`, 1),
		"plugin":           strings.Replace(sessionInit, `"plugins":[]`, `"plugins":[{"name":"cc-plugin-agents-md"}]`, 1),
		"slash command":    strings.Replace(sessionInit, `"slash_commands":[]`, `"slash_commands":["/ship"]`, 1),
		"api key":          strings.Replace(sessionInit, `"apiKeySource":"none"`, `"apiKeySource":"ANTHROPIC_API_KEY"`, 1),
		"version":          strings.Replace(sessionInit, "2.1.308", "2.2.0", 1),
		"permission mode":  strings.Replace(sessionInit, "bypassPermissions", "default", 1),
	} {
		if init == sessionInit {
			t.Fatalf("%s: the planted change did not apply", name)
		}
		if _, err := readSession(t, launch(), init); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// A resumed session must be the conversation it was asked to continue.
	l := launch()
	l.Conversation = "s-0"
	if _, err := readSession(t, l, sessionInit); err == nil {
		t.Error("a resumed session accepted another session ID")
	}
	l.Conversation = "s-1"
	if _, err := readSession(t, l, sessionInit); err != nil {
		t.Errorf("a resumed session with the same ID: %v", err)
	}
	for name, lines := range map[string][]string{
		"before init": {`{"type":"assistant","parent_tool_use_id":null,"message":{}}`},
		"control":     {sessionInit, `{"type":"control_request","request":{}}`},
		"not json":    {sessionInit, `garbage`},
		"no type":     {`{"subtype":"init"}`},
	} {
		if _, err := readSession(t, launch(), lines...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSessionReaderAcceptsTheCLIsOwnSubagents(t *testing.T) {
	for _, name := range builtinAgents {
		init := strings.Replace(sessionInit, `"agents":[`, `"agents":["`+name+`",`, 1)
		if _, err := readSession(t, launch(), init); err != nil {
			t.Errorf("the CLI's own subagent %s: %v", name, err)
		}
	}
	if _, err := readSession(t, launch(), strings.Replace(sessionInit, `"agents":["claude","Explore","general-purpose","Plan","statusline-setup"]`, `"agents":[]`, 1)); err != nil {
		t.Errorf("a launch without subagents: %v", err)
	}
}

func TestSessionReaderCountsSubagentsWorkAndListsEveryModel(t *testing.T) {
	r, err := readSession(t, launch(), sessionInit,
		`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m1","model":"claude-sonnet-5-5","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/work/README.md"}},{"type":"tool_use","name":"Agent","input":{"prompt":"look"}}]},"session_id":"s-1"}`,
		// A subagent's messages carry the id of the tool call that started them.
		`{"type":"assistant","parent_tool_use_id":"toolu_1","message":{"id":"m2","model":"claude-sonnet-5-5","content":[{"type":"tool_use","name":"Bash","input":{"command":"git log"}},{"type":"tool_use","name":"Read","input":{"file_path":"/work/src/a.go"}}]},"session_id":"s-1"}`,
		`{"type":"user","parent_tool_use_id":"toolu_1","message":{}}`,
		`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m3","model":"<synthetic>","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}},{"type":"tool_use","name":"Task","input":{}}]},"session_id":"s-1"}`,
		`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m4","model":"claude-sonnet-5-5","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/work/README.md"}},{"type":"text","text":"done"}]},"session_id":"s-1"}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"All done.","session_id":"s-1","usage":{"input_tokens":1,"cache_creation_input_tokens":1,"cache_read_input_tokens":1,"output_tokens":1},`+
			`"modelUsage":{"claude-sonnet-5-5":{"inputTokens":10,"outputTokens":4,"cacheReadInputTokens":30,"cacheCreationInputTokens":20}}}`)
	if err != nil {
		t.Fatal(err)
	}
	seen := r.Seen()
	if !seen.Measured || !slices.Equal(seen.Opened, []string{"/work/README.md", "/work/src/a.go"}) || !slices.Equal(seen.Commands, []string{"git log", "go test ./..."}) || seen.Subagents != 2 {
		t.Errorf("seen %+v", seen)
	}
	// The synthetic model never answered, so it is not listed.
	if got := r.Models(); !slices.Equal(got, []string{"claude-sonnet-5-5"}) {
		t.Errorf("models %v", got)
	}
	// The usage is the per-model figures' sum, which holds the subagents; the
	// result's own usage does not.
	out := r.Outcome()
	if out.End != agents.Answered || out.Text != "All done." || out.Conversation != "s-1" || out.Model != "claude-sonnet-5-5" ||
		out.Usage != (agents.Usage{Input: 10, CacheCreation: 20, CacheRead: 30, Output: 4}) {
		t.Errorf("outcome %+v", out)
	}
}

func TestSessionReaderFallsBackToTheResultsOwnUsage(t *testing.T) {
	// A result without per-model figures falls back to its own usage.
	r, err := readSession(t, launch(), sessionInit,
		`{"type":"result","subtype":"success","is_error":false,"result":"ok","session_id":"s-1","usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}`)
	if err != nil || r.Outcome().Usage != (agents.Usage{Input: 1, CacheCreation: 2, CacheRead: 3, Output: 4}) {
		t.Fatalf("outcome %+v %v", r.Outcome(), err)
	}
}

func TestSessionReaderEndsAndClassifiesLikeATurn(t *testing.T) {
	failed := `{"type":"result","subtype":"success","is_error":true,"result":"API Error","session_id":"s-1","api_error_status":null}`
	for _, c := range []struct {
		lines []string
		want  agents.Class
	}{
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"overloaded","message":{}}`, failed}, agents.Capacity},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"rate_limit","message":{}}`, failed}, agents.RateLimit},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"authentication_failed","message":{}}`, failed}, agents.LoginFailed},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"invalid_request","message":{}}`, failed}, agents.Other},
		// A subagent's failed request does not decide how the session ended.
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"overloaded","message":{}}`, `{"type":"assistant","parent_tool_use_id":"toolu_1","message":{}}`, failed}, agents.Capacity},
	} {
		r, err := readSession(t, launch(), append([]string{sessionInit}, c.lines...)...)
		if out := r.Outcome(); err != nil || out.End != agents.Failed || out.Class != c.want {
			t.Errorf("%v: outcome %+v %v, want %s", c.lines, out, err, c.want)
		}
	}
	// A session that was cut off writes no result; its usage is a lower bound
	// from the messages.
	r, err := readSession(t, launch(), sessionInit,
		`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m1","usage":{"input_tokens":2,"cache_creation_input_tokens":100,"cache_read_input_tokens":50,"output_tokens":10}},"session_id":"s-1"}`)
	if out := r.Outcome(); err != nil || out.End != "" || out.Usage != (agents.Usage{Input: 2, CacheCreation: 100, CacheRead: 50, Output: 10, Partial: true}) {
		t.Fatalf("a cut-off session %+v %v", r.Outcome(), err)
	}
}

// A subagent's messages are the session's own work: unlike a turn, the session
// reader accepts them.
func TestSessionReaderAcceptsSubagentMessages(t *testing.T) {
	_, err := readSession(t, launch(), sessionInit,
		`{"type":"assistant","parent_tool_use_id":"toolu_9","message":{"content":[{"type":"text","text":"working"}]},"session_id":"s-1"}`,
		`{"type":"user","parent_tool_use_id":"toolu_9","message":{}}`)
	if err != nil {
		t.Errorf("a subagent's messages: %v", err)
	}
}

// A session runs on its one model, its subagents included: another that answers
// or reports ends it, and so does a launch on another model.
func TestSessionReaderRefusesAnotherModel(t *testing.T) {
	for name, lines := range map[string][]string{
		"a subagent's answer": {sessionInit, `{"type":"assistant","parent_tool_use_id":"toolu_1","message":{"id":"m2","model":"claude-opus-5-5","content":[]}}`},
		"the launch":          {strings.Replace(sessionInit, `"model":"claude-sonnet-5-5"`, `"model":"claude-opus-5-5"`, 1)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readSession(t, launch(), lines...); err == nil || !strings.Contains(err.Error(), "claude-opus-5-5") {
				t.Fatalf("%v", err)
			}
		})
	}
}

// A session a budget or turn limit stopped did not finish, whatever is_error says.
func TestSessionReaderDoesNotTakeABudgetStopForAFinish(t *testing.T) {
	r, err := readSession(t, launch(), sessionInit, `{"type":"result","subtype":"error_max_budget_usd","is_error":false,"session_id":"s-1","usage":{}}`)
	if err != nil {
		t.Fatal(err)
	}
	if out := r.Outcome(); out.End != agents.Failed || !strings.Contains(out.Detail, "error_max_budget_usd") {
		t.Fatalf("outcome %+v", out)
	}
}

// Without a shell a session has no tool that runs a command, and it can still
// write its note.
func TestALaunchWithoutAShellHasNoToolThatRunsACommand(t *testing.T) {
	l := launch()
	argv, _, _, err := (Provider{}).Launch(l)
	if err != nil {
		t.Fatal(err)
	}
	if tools := option(t, argv, "--tools"); !strings.Contains(tools, "Bash") {
		t.Fatalf("a usual session lost its shell: %s", tools)
	}
	l.NoShell = true
	argv, _, _, err = (Provider{}).Launch(l)
	if err != nil {
		t.Fatal(err)
	}
	tools := option(t, argv, "--tools")
	if strings.Contains(tools, "Bash") || !strings.Contains(tools, "Write") || !strings.Contains(tools, "Read") {
		t.Fatalf("tools without a shell: %s", tools)
	}
}
