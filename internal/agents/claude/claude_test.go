package claude

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func credentials(oauth map[string]any) []byte {
	raw, _ := json.Marshal(map[string]any{"claudeAiOauth": oauth, "mcpOAuth": map[string]any{"server": "other-secret"}})
	return raw
}

func TestLoginCopiesTheAccountLoginWithoutRefreshAuthority(t *testing.T) {
	l, err := Provider{}.Login(credentials(map[string]any{"accessToken": "sk-ant-oat01-a", "refreshToken": "sk-ant-ort01-r", "expiresAt": now.Add(time.Hour).UnixMilli(), "scopes": []string{"user:inference"}}), now)
	if err != nil {
		t.Fatal(err)
	}
	copy := string(l.Copy)
	if strings.Contains(copy, "ort01") || strings.Contains(copy, "refreshToken") || strings.Contains(copy, "mcpOAuth") || !strings.Contains(copy, "sk-ant-oat01-a") || !strings.Contains(copy, "scopes") {
		t.Fatalf("copy %s", copy)
	}
	if len(l.Secrets) != 2 || !l.Expires.Equal(now.Add(time.Hour)) {
		t.Fatalf("login %+v", l)
	}
}

func TestLoginRefusesAPIKeysExpiredAndForeignFiles(t *testing.T) {
	for name, source := range map[string][]byte{
		"api key":    credentials(map[string]any{"accessToken": "sk-ant-api03-k"}),
		"expired":    credentials(map[string]any{"accessToken": "sk-ant-oat01-a", "expiresAt": now.Add(-time.Second).UnixMilli()}),
		"no oauth":   []byte(`{"other":{}}`),
		"no token":   credentials(map[string]any{"refreshToken": "r"}),
		"not json":   []byte("token"),
		"empty":      nil,
		"bad expiry": credentials(map[string]any{"accessToken": "a", "expiresAt": "soon"}),
	} {
		if _, err := (Provider{}).Login(source, now); err == nil {
			t.Errorf("%s: admitted", name)
		}
	}
}

func TestCheckAuthAdmitsOnlyTheClaudeAILogin(t *testing.T) {
	if err := (Provider{}).CheckAuth([]byte(`{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{`{"loggedIn":false,"authMethod":"none"}`, `{"loggedIn":true,"authMethod":"api_key"}`, `not json`} {
		if (Provider{}).CheckAuth([]byte(out)) == nil {
			t.Errorf("admitted %s", out)
		}
	}
}

func TestTurnLaunchesThePinnedContract(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	argv, env, stdin, err := Provider{}.Turn(agents.Turn{First: true, Message: "do it", Model: "claude-sonnet-5-5", Effort: "high", Deadline: deadline})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"--print --input-format stream-json --output-format stream-json --verbose", "--model claude-sonnet-5-5 --effort high",
		"--permission-mode bypassPermissions", "--tools Bash,Read,Edit,Write,Glob,Grep", "--setting-sources user", "--strict-mcp-config", `"cc-plugin-agents-md@builtin":false`} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv lacks %q", want)
		}
	}
	if strings.Contains(joined, "--resume") {
		t.Error("a first turn resumes")
	}
	envs := strings.Join(env, " ")
	for _, want := range []string{"CLAUDE_CONFIG_DIR=/home/agent/.claude", "CLAUDE_CODE_DISABLE_CLAUDE_MDS=1", "CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS=1", "MCP_TOOL_TIMEOUT="} {
		if !strings.Contains(envs, want) {
			t.Errorf("env lacks %q", want)
		}
	}
	if strings.Contains(envs, "SUBPROCESS_ENV_SCRUB") || strings.Contains(envs, "ANTHROPIC") {
		t.Error("env carries a variable the launch must not")
	}
	var msg struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct{ Text string } `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(stdin, &msg) != nil || msg.Type != "user" || msg.Message.Content[0].Text != "do it" || !strings.HasSuffix(string(stdin), "\n") {
		t.Fatalf("stdin %s", stdin)
	}
	argv, _, _, err = Provider{}.Turn(agents.Turn{Conversation: "s-1", Model: "claude-sonnet-5-5", Effort: "max"})
	if err != nil || !strings.Contains(strings.Join(argv, " "), "--resume s-1") {
		t.Fatalf("resume %q %v", argv, err)
	}
	for _, bad := range []agents.Turn{{Model: "m", Effort: "high"}, {First: true, Model: "m", Effort: "ultra"}, {First: true, Effort: "high"}} {
		if _, _, _, err := (Provider{}).Turn(bad); err == nil {
			t.Errorf("turn %+v launched", bad)
		}
	}
}

func TestHomeCarriesTheHarnessAsSystemPrompts(t *testing.T) {
	i, err := Provider{}.Home(agents.Login{Copy: []byte("{}")}, harness.Document{Base: "base", Context: "context", Task: "task"})
	if err != nil || i.Claude == nil || i.Claude.SystemPrompt != "base" || i.Claude.AppendSystemPrompt != "context" || i.CodexAuth != nil {
		t.Fatalf("home %+v %v", i, err)
	}
	if _, err := (Provider{}).Home(agents.Login{Copy: []byte("{}")}, harness.Document{Base: "b", Context: strings.Repeat("x", 2<<20)}); err == nil {
		t.Fatal("an oversized harness was installed")
	}
}

const initOK = `{"type":"system","subtype":"init","session_id":"s-1","model":"claude-sonnet-5-5","claude_code_version":"2.1.308","apiKeySource":"none","permissionMode":"bypassPermissions","tools":["Bash","Edit","Glob","Grep","Read","Write","mcp__markitect__submit"],"agents":[],"skills":[],"plugins":[],"slash_commands":[],"mcp_servers":[{"name":"markitect","status":"pending"}]}`

func read(t *testing.T, turn agents.Turn, lines ...string) (agents.Outcome, error) {
	t.Helper()
	r := Provider{}.Reader(turn)
	for _, l := range lines {
		if err := r.Line([]byte(l)); err != nil {
			return agents.Outcome{}, err
		}
	}
	return r.Outcome(), nil
}

func TestReaderTellsATurnsProgress(t *testing.T) {
	r := Provider{}.Reader(agents.Turn{}).(agents.ProgressReader)
	for raw, want := range map[string]agents.Progress{
		`{"type":"assistant","message":{"content":[{"type":"text","text":" I will look. "}]}}`:                                                    {Text: "I will look."},
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/work/.markitect/model/x.md"}}]}}`:      {Doing: "Reading .markitect/model/x.md"},
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"markitect  check\n--repo /work/a"}}]}}`:   {Doing: "Running markitect check --repo a"},
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Now."},{"type":"tool_use","name":"Grep","input":{"pattern":"loan"}}]}}`: {Text: "Now.", Doing: "Searching for loan"},
		`{"type":"assistant","error":"overloaded","message":{"content":[{"type":"text","text":"API Error: 529"}]}}`:                               {},
		`{"type":"user","message":{"content":[{"type":"text","text":"tool output"}]}}`:                                                            {},
		`{"type":"result","result":"done"}`: {},
		`not json`:                          {},
	} {
		if got := r.Progress([]byte(raw)); got != want {
			t.Errorf("%s: %+v, want %+v", raw, got, want)
		}
	}
}

func TestReaderAnswersAndCountsUsage(t *testing.T) {
	out, err := read(t, agents.Turn{Tools: []string{"submit"}}, initOK,
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"}}`,
		`{"type":"system","subtype":"thinking_tokens"}`,
		`{"type":"assistant","parent_tool_use_id":null,"message":{"role":"assistant","content":[{"type":"text","text":"ok"}]},"session_id":"s-1"}`,
		`{"type":"tool_progress","parent_tool_use_id":"toolu_1"}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s-1","usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}`)
	if err != nil || out.End != agents.Answered || out.Conversation != "s-1" || out.Model != "claude-sonnet-5-5" || out.Usage != (agents.Usage{Input: 1, CacheCreation: 2, CacheRead: 3, Output: 4}) || out.Text != "done" {
		t.Fatalf("outcome %+v %v", out, err)
	}
}

func TestReaderRefusesWhatTheLaunchMustNotBe(t *testing.T) {
	turn := agents.Turn{Tools: []string{"submit"}}
	for name, lines := range map[string][]string{
		"api key":       {strings.Replace(initOK, `"apiKeySource":"none"`, `"apiKeySource":"ANTHROPIC_API_KEY"`, 1)},
		"version":       {strings.Replace(initOK, "2.1.308", "2.2.0", 1)},
		"plugin":        {strings.Replace(initOK, `"plugins":[]`, `"plugins":[{"name":"cc-plugin-agents-md"}]`, 1)},
		"agent":         {strings.Replace(initOK, `"agents":[]`, `"agents":["general-purpose"]`, 1)},
		"task tool":     {strings.Replace(initOK, `"Bash",`, `"Bash","Task",`, 1)},
		"no server":     {strings.Replace(initOK, `"status":"pending"`, `"status":"failed"`, 1)},
		"permission":    {strings.Replace(initOK, "bypassPermissions", "default", 1)},
		"before init":   {`{"type":"assistant","parent_tool_use_id":null,"message":{}}`},
		"sub-agent":     {initOK, `{"type":"user","parent_tool_use_id":"toolu_9","message":{}}`},
		"control":       {initOK, `{"type":"control_request","request":{}}`},
		"not json":      {initOK, `garbage`},
		"other session": {initOK, `{"type":"assistant","parent_tool_use_id":null,"session_id":"s-2","message":{}}`},
	} {
		if _, err := read(t, turn, lines...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := read(t, agents.Turn{Tools: []string{"submit"}, Conversation: "s-0"}, initOK); err == nil {
		t.Error("a resumed turn accepted another session")
	}
}

func TestReaderClassifiesProviderFailures(t *testing.T) {
	failed := func(category string, status int) string {
		return `{"type":"result","subtype":"success","is_error":true,"result":"API Error","session_id":"s-1","api_error_status":` + map[bool]string{true: "null", false: jsonInt(status)}[status == 0] + `}`
	}
	for _, c := range []struct {
		lines []string
		want  agents.Class
	}{
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"overloaded","message":{}}`, failed("", 0)}, agents.Capacity},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"rate_limit","message":{}}`, failed("", 0)}, agents.RateLimit},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"server_error","message":{}}`, failed("", 0)}, agents.ServerError},
		{[]string{`{"type":"system","subtype":"api_retry","attempt":1,"error":"unknown"}`, failed("", 0)}, agents.ServerError},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"authentication_failed","message":{}}`, failed("", 0)}, agents.LoginFailed},
		{[]string{failed("", 529)}, agents.Capacity},
		{[]string{failed("", 429)}, agents.RateLimit},
		{[]string{failed("", 503)}, agents.ServerError},
		{[]string{`{"type":"assistant","parent_tool_use_id":null,"error":"invalid_request","message":{}}`, failed("", 0)}, agents.Other},
		// A request that failed and then recovered is no failure of the turn.
		{[]string{`{"type":"system","subtype":"api_retry","attempt":1,"error":"overloaded"}`, `{"type":"assistant","parent_tool_use_id":null,"message":{}}`, failed("", 0)}, agents.Other},
	} {
		out, err := read(t, agents.Turn{Tools: []string{"submit"}}, append([]string{initOK}, c.lines...)...)
		if err != nil || out.End != agents.Failed || out.Class != c.want {
			t.Errorf("%v: outcome %+v %v, want %s", c.lines, out, err, c.want)
		}
	}
}

func jsonInt(i int) string {
	raw, _ := json.Marshal(i)
	return string(raw)
}

// A turn the Run stops writes no result: its usage is what the assistant
// messages showed, each message counted once by its last line, and partial,
// since their output counts are those streamed at each message's start.
func TestReaderCountsAStoppedTurnFromItsMessages(t *testing.T) {
	msg := func(id string, in, created, read, output int) string {
		return fmt.Sprintf(`{"type":"assistant","parent_tool_use_id":null,"message":{"id":%q,"role":"assistant","content":[],"usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d}},"session_id":"s-1"}`,
			id, in, created, read, output)
	}
	out, err := read(t, agents.Turn{Tools: []string{"submit"}}, initOK,
		msg("m1", 2, 100, 50, 10), msg("m1", 2, 100, 50, 12), msg("m2", 3, 20, 150, 5))
	if err != nil || out.End != "" || out.Usage != (agents.Usage{Input: 5, CacheCreation: 120, CacheRead: 200, Output: 17, Partial: true}) {
		t.Fatalf("outcome %+v %v", out, err)
	}
	// A turn without any message reports no total either.
	if out, err := read(t, agents.Turn{Tools: []string{"submit"}}, initOK); err != nil || out.Usage != (agents.Usage{Partial: true}) {
		t.Fatalf("a turn without messages: %+v %v", out, err)
	}
	// A failed turn with an error result reports its total.
	out, err = read(t, agents.Turn{Tools: []string{"submit"}}, initOK, msg("m1", 2, 100, 50, 10),
		`{"type":"result","subtype":"success","is_error":true,"result":"API Error: 529","session_id":"s-1","usage":{"input_tokens":2,"cache_creation_input_tokens":100,"cache_read_input_tokens":50,"output_tokens":30}}`)
	if err != nil || out.Usage != (agents.Usage{Input: 2, CacheCreation: 100, CacheRead: 50, Output: 30}) {
		t.Fatalf("an error result: %+v %v", out, err)
	}
	// A turn that answered counts its result, not its messages.
	out, err = read(t, agents.Turn{Tools: []string{"submit"}}, initOK, msg("m1", 2, 100, 50, 10),
		`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s-1","usage":{"input_tokens":2,"cache_creation_input_tokens":100,"cache_read_input_tokens":50,"output_tokens":400}}`)
	if err != nil || out.Usage != (agents.Usage{Input: 2, CacheCreation: 100, CacheRead: 50, Output: 400}) {
		t.Fatalf("outcome %+v %v", out, err)
	}
}
