package codex

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/harness"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func jwt(exp time.Time) string {
	part := func(v any) string {
		raw, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return part(map[string]string{"alg": "RS256"}) + "." + part(map[string]any{"exp": exp.Unix(), "https://api.openai.com/auth": map[string]string{"chatgpt_plan_type": "pro"}}) + ".sig"
}

func auth(apiKey any, access string) []byte {
	raw, _ := json.Marshal(map[string]any{"OPENAI_API_KEY": apiKey, "tokens": map[string]string{"id_token": jwt(now.Add(time.Hour)), "access_token": access, "refresh_token": "rt-secret", "account_id": "acc"}, "last_refresh": "2026-10-28T00:00:00Z"})
	return raw
}

func TestLoginKeepsAnEmptyRefreshTokenForThePinnedCLI(t *testing.T) {
	l, err := Provider{}.Login(auth(nil, jwt(now.Add(time.Hour))), now)
	if err != nil {
		t.Fatal(err)
	}
	var copy struct {
		APIKey *string            `json:"OPENAI_API_KEY"`
		Tokens map[string]*string `json:"tokens"`
	}
	if err := json.Unmarshal(l.Copy, &copy); err != nil {
		t.Fatal(err)
	}
	// Codex 0.162.0 refuses a login without the refresh_token field and
	// reports "Logged in using ChatGPT" with an empty one (probed offline).
	if copy.APIKey != nil || copy.Tokens["refresh_token"] == nil || *copy.Tokens["refresh_token"] != "" || *copy.Tokens["account_id"] != "acc" {
		t.Fatalf("copy %s", l.Copy)
	}
	if strings.Contains(string(l.Copy), "rt-secret") || !l.Expires.Equal(now.Add(time.Hour)) || len(l.Secrets) != 3 {
		t.Fatalf("login %+v", l)
	}
}

func TestLoginRefusesAPIKeysAndExpiredTokens(t *testing.T) {
	for name, source := range map[string][]byte{
		"api key":   auth("sk-proj-x", jwt(now.Add(time.Hour))),
		"expired":   auth(nil, jwt(now.Add(-time.Minute))),
		"no jwt":    auth(nil, "opaque"),
		"no tokens": []byte(`{"OPENAI_API_KEY":null}`),
		"not json":  []byte("x"),
	} {
		if _, err := (Provider{}).Login(source, now); err == nil {
			t.Errorf("%s: admitted", name)
		}
	}
}

func TestCheckAuth(t *testing.T) {
	if err := (Provider{}).CheckAuth([]byte("Logged in using ChatGPT\n")); err != nil {
		t.Fatal(err)
	}
	// The pinned CLI reports on standard error, after any warning.
	argv, _ := Provider{}.AuthStatus()
	if argv[0] != "/bin/sh" || !strings.HasSuffix(argv[2], "login status 2>&1") {
		t.Fatalf("auth status %q", argv)
	}
	if err := (Provider{}).CheckAuth([]byte("WARNING: proceeding, even though we could not create PATH aliases\nLogged in using ChatGPT\n")); err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{"Logged in using an API key - ***", "Not logged in", ""} {
		if (Provider{}).CheckAuth([]byte(out)) == nil {
			t.Errorf("admitted %q", out)
		}
	}
}

func TestTurnLaunchesExecWithTheBaselineSwitches(t *testing.T) {
	doc := harness.Document{Base: "BASE", Context: "CONTEXT", Task: "TASK"}
	argv, env, stdin, err := Provider{}.Turn(agents.Turn{First: true, Message: "TASK", Document: doc, Model: "gpt-6-sol", Effort: "high", Deadline: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"/usr/local/bin/codex exec --json", "--disable multi_agent --disable multi_agent_v2 -c agents.enabled=false", `cli_auth_credentials_store="file"`,
		"--dangerously-bypass-approvals-and-sandbox", "-m gpt-6-sol", `model_reasoning_effort="high"`, `mcp_servers.markitect.args=["mcp"]`, "mcp_servers.markitect.tool_timeout_sec="} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv lacks %q", want)
		}
	}
	if argv[len(argv)-1] != "-" || strings.Contains(joined, "resume") {
		t.Fatalf("argv %q", argv)
	}
	if string(stdin) != "BASE\n\nCONTEXT\n\nTASK" || !strings.Contains(strings.Join(env, " "), "CODEX_HOME=/home/agent/.codex") {
		t.Fatalf("stdin %q env %q", stdin, env)
	}
	argv, _, stdin, err = Provider{}.Turn(agents.Turn{Conversation: "th-1", Message: "again", Document: doc, Model: "gpt-6-sol", Effort: "high"})
	if err != nil || argv[1] != "exec" || argv[2] != "resume" || argv[len(argv)-2] != "th-1" || string(stdin) != "again" {
		t.Fatalf("resume %q %q %v", argv, stdin, err)
	}
	if _, _, _, err := (Provider{}).Turn(agents.Turn{Model: "m", Effort: "high"}); err == nil {
		t.Fatal("a later turn without a thread launched")
	}
}

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

func TestReaderAnswersWithTheThreadAndUsage(t *testing.T) {
	out, err := read(t, agents.Turn{}, `{"type":"thread.started","thread_id":"th-1"}`, `{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_0","type":"mcp_tool_call","server":"markitect","tool":"submit"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":60,"output_tokens":7}}`)
	if err != nil || out.End != agents.Answered || out.Conversation != "th-1" || out.Usage != (agents.Usage{Input: 40, CacheRead: 60, Output: 7}) {
		t.Fatalf("outcome %+v %v", out, err)
	}
	if _, err := read(t, agents.Turn{Conversation: "th-1"}, `{"type":"thread.started","thread_id":"th-2"}`); err == nil {
		t.Fatal("a resumed turn started another thread")
	}
	if _, err := read(t, agents.Turn{}, `{"type":"item.started","item":{"type":"collab_tool_call"}}`); err == nil {
		t.Fatal("native multi-agent work accepted")
	}
	if out, _ := read(t, agents.Turn{}, `{"type":"error","message":"Reconnecting... 1/5 (stream error: 503)"}`, `{"type":"turn.completed","usage":{}}`); out.End != agents.Answered {
		t.Fatalf("a turn that recovered from a reconnect failed: %+v", out)
	}
	if out, _ := read(t, agents.Turn{}, `{"type":"thread.started","thread_id":"th-1"}`); out.End != "" {
		t.Fatalf("a turn without its end answered: %+v", out)
	}
}

func TestReaderClassifiesProviderFailures(t *testing.T) {
	for message, want := range map[string]agents.Class{
		"Selected model is at capacity. Please try a different model.":                 agents.Capacity,
		"exceeded retry limit, last status: 429 Too Many Requests":                     agents.RateLimit,
		"You've hit your usage limit. Upgrade to Pro or try again later.":              agents.RateLimit,
		"unexpected status 401 Unauthorized: token_expired":                            agents.LoginFailed,
		"exceeded retry limit, last status: 500 Internal Server Error":                 agents.ServerError,
		"stream disconnected before completion: error sending request for url":         agents.ServerError,
		"invalid_request_error: the prompt is too long for the model's context window": agents.Other,
	} {
		out, err := read(t, agents.Turn{}, `{"type":"thread.started","thread_id":"th-1"}`, `{"type":"turn.failed","error":{"message":`+quote(message)+`}}`)
		if err != nil || out.End != agents.Failed || out.Class != want {
			t.Errorf("%q: %+v %v, want %s", message, out, err, want)
		}
	}
	out, _ := read(t, agents.Turn{}, `{"type":"error","message":"Reconnecting... 2/5 (stream error: 503 Service Unavailable)"}`)
	if out.Class != agents.ServerError {
		t.Fatalf("error event: %+v", out)
	}
}

func quote(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
