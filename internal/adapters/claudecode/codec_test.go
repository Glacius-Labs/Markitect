package claudecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCodecEncodesOneUserMessage(t *testing.T) {
	c := testCodec(t, 1<<20)
	line, err := c.EncodeUser("{\"events\":[]}\nwith \"quotes\" ✓")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(line, []byte("\n")) || bytes.Count(line, []byte("\n")) != 1 {
		t.Fatalf("one NDJSON line expected: %q", line)
	}
	var got struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content []struct {
				Type, Text string
			} `json:"content"`
		} `json:"message"`
		Parent *string `json:"parent_tool_use_id"`
	}
	if err := json.Unmarshal(line, &got); err != nil || got.Type != "user" || got.Message.Role != "user" || len(got.Message.Content) != 1 || got.Message.Content[0].Type != "text" || got.Message.Content[0].Text != "{\"events\":[]}\nwith \"quotes\" ✓" || got.Parent != nil || !bytes.Contains(line, []byte(`"parent_tool_use_id":null`)) {
		t.Fatalf("user message shape: %v %s", err, line)
	}
	for _, text := range []string{"", "\xff", strings.Repeat("x", 1<<20)} {
		if _, err := c.EncodeUser(text); err == nil {
			t.Fatalf("refusable text %d bytes encoded", len(text))
		}
	}
	if _, err := NewCodec(maxProtocolMessage + 1); !errors.Is(err, ErrProtocolLimit) {
		t.Fatal("line bound above 8 MiB admitted")
	}
}

func TestCodecProjectsTheConsumedOutputShapes(t *testing.T) {
	init := decodeOne(t, fixtureInit(fixtureSession, connectedServers))
	var tools []string
	_ = json.Unmarshal([]byte(fixtureTools), &tools)
	if init.Kind() != SystemInitMessage || init.Session() != fixtureSession || !reflect.DeepEqual(init.Init(), Init{Session: fixtureSession, Model: "claude-sonnet-5", Version: "2.1.308", APIKeySource: "none", PermissionMode: "bypassPermissions", Tools: tools, Agents: []string{}, Skills: []string{}, SlashCommands: []string{}, MCPServers: []MCPServer{{"markitect", "connected"}}}) {
		t.Fatalf("init %+v", init.Init())
	}
	loaded := decodeOne(t, strings.Replace(strings.Replace(fixtureInit(fixtureSession, connectedServers), `"plugins":[]`, `"plugins":[{"name":"p","path":"/x"}]`, 1), `"agents":[]`, `"agents":["general-purpose"]`, 1))
	if !reflect.DeepEqual(loaded.Init().Plugins, []string{"p"}) || !reflect.DeepEqual(loaded.Init().Agents, []string{"general-purpose"}) {
		t.Fatalf("loaded extensions %+v", loaded.Init())
	}
	retry := decodeOne(t, fixtureRetry(fixtureSession, "overloaded"))
	if retry.Kind() != APIRetryMessage || retry.Retry() != (APIRetry{Attempt: 1, Error: "overloaded"}) {
		t.Fatalf("retry %+v", retry.Retry())
	}
	assistant := decodeOne(t, fixtureAssistant(fixtureSession, textBlock("Working."), `{"type":"thinking","thinking":"...","signature":"s"}`, toolUse("toolu_1", "Bash")))
	if assistant.Kind() != AssistantMessage || assistant.Parent() != "" || !reflect.DeepEqual(assistant.Blocks(), []Block{{Type: "text", Text: "Working."}, {Type: "thinking"}, {Type: "tool_use", ID: "toolu_1", Name: "Bash"}}) {
		t.Fatalf("assistant %+v", assistant.Blocks())
	}
	user := decodeOne(t, fixtureToolResult(fixtureSession, "toolu_1"))
	if user.Kind() != UserMessage || !reflect.DeepEqual(user.Blocks(), []Block{{Type: "tool_result", ToolUseID: "toolu_1"}}) {
		t.Fatalf("user %+v", user.Blocks())
	}
	result := decodeOne(t, fixtureResult(fixtureSession, "Done."))
	if result.Kind() != ResultMessage || result.Result().Subtype != "success" || result.Result().IsError || result.Result().Text != "Done." || result.Result().Session != fixtureSession {
		t.Fatalf("result %+v", result.Result())
	}
	failure := fixtureFailure(fixtureSession, "overloaded")
	if m := decodeOne(t, failure[0]); m.ErrorKind() != "overloaded" {
		t.Fatalf("assistant error category %q", m.ErrorKind())
	}
	if m := decodeOne(t, failure[1]); !m.Result().IsError || m.Result().Text != "API Error: 529 Overloaded" {
		t.Fatalf("error result %+v", m.Result())
	}
	execution := decodeOne(t, `{"type":"result","subtype":"error_during_execution","is_error":true,"session_id":"`+fixtureSession+`","errors":["boom"],"num_turns":0,"duration_ms":1,"duration_api_ms":0,"total_cost_usd":0,"usage":{},"modelUsage":{},"permission_denials":[],"stop_reason":null,"uuid":"u"}`)
	if !reflect.DeepEqual(execution.Result(), Result{Subtype: "error_during_execution", IsError: true, Errors: []string{"boom"}, Session: fixtureSession}) {
		t.Fatalf("execution error %+v", execution.Result())
	}
	subagent := decodeOne(t, strings.Replace(fixtureAssistant(fixtureSession, textBlock("child")), `"parent_tool_use_id":null`, `"parent_tool_use_id":"toolu_agent"`, 1))
	if subagent.Parent() != "toolu_agent" {
		t.Fatal("subagent parent lost")
	}
	plain := decodeOne(t, `{"type":"user","message":{"role":"user","content":"plain"},"parent_tool_use_id":null,"session_id":"`+fixtureSession+`"}`)
	if !reflect.DeepEqual(plain.Blocks(), []Block{{Type: "text", Text: "plain"}}) {
		t.Fatalf("plain user content %+v", plain.Blocks())
	}
}

func TestCodecKeepsUnknownAndControlMessagesVisible(t *testing.T) {
	for line, kind := range map[string]MessageKind{
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"},"uuid":"u","session_id":"` + fixtureSession + `"}`:                                  UnknownMessage,
		`{"type":"stream_event","event":{"type":"message_start"},"session_id":"` + fixtureSession + `"}`:                                                       UnknownMessage,
		`{"type":"system","subtype":"compact_boundary","compact_metadata":{"trigger":"auto","pre_tokens":1},"uuid":"u","session_id":"` + fixtureSession + `"}`: SystemMessage,
		`{"type":"control_request","request_id":"r1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{}}}`:                                      ControlRequestMessage,
	} {
		m := decodeOne(t, line)
		if m.Kind() != kind || !bytes.Equal(m.Raw(), []byte(line+"\n")) {
			t.Fatalf("%s classified %s", line, m.Kind())
		}
	}
}

func TestCodecRefusesMalformedOrInconsistentLines(t *testing.T) {
	for _, line := range []string{
		`not json`,
		`[]`,
		`{"subtype":"init"}`,
		`{"type":""}`,
		`{"type":"system"}`,
		`{"type":"system","subtype":"init","session_id":"","model":"m","tools":[],"mcp_servers":[]}`,
		`{"type":"system","subtype":"init","session_id":"s","model":"m","tools":{},"mcp_servers":[]}`,
		`{"type":"system","subtype":"init","session_id":"s","model":"m","tools":[],"mcp_servers":[{"name":"markitect"}]}`,
		`{"type":"system","subtype":"api_retry","attempt":0}`,
		`{"type":"assistant","message":{"role":"assistant","content":[]},"session_id":"s"}`,
		`{"type":"assistant","message":{"role":"user","content":[]},"parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"role":"assistant","content":"text"},"parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]},"parent_tool_use_id":null}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result"}]},"parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"role":"assistant","content":[]},"parent_tool_use_id":""}`,
		`{"type":"result","subtype":"success","is_error":false,"session_id":"s"}`,
		`{"type":"result","subtype":"success","is_error":"no","session_id":"s","result":"r"}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"r"}`,
		`{"type":"result","subtype":"x","is_error":true,"session_id":"s","errors":"boom"}`,
		`{"type":"system","subtype":"init","subtype":"init"}`,
		`{"type":"user","message":{"role":"user","content":"\ud800"},"parent_tool_use_id":null}`,
		"{\"type\":\"user\",\"message\":{\"role\":\"user\",\"content\":\"\xff\"},\"parent_tool_use_id\":null}",
		`{"type":"x"} {"type":"y"}`,
		``,
	} {
		messages, err := testCodec(t, 1<<20).Feed([]byte(line + "\n"))
		if err == nil || len(messages) != 1 || messages[0].Kind() != RejectedMessage || !bytes.Equal(messages[0].Raw(), []byte(line+"\n")) {
			t.Fatalf("line admitted or bytes lost: %q %v %+v", line, err, messages)
		}
	}
}

func TestCodecFragmentsBoundsAndLatchedFailure(t *testing.T) {
	c := testCodec(t, 1<<20)
	whole := fixtureInit(fixtureSession, connectedServers) + "\n" + fixtureResult(fixtureSession, "Done.") + "\n"
	var got []Message
	for i := 0; i < len(whole); i += 7 {
		messages, err := c.Feed([]byte(whole[i:min(i+7, len(whole))]))
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, messages...)
	}
	if len(got) != 2 || got[0].Kind() != SystemInitMessage || got[1].Kind() != ResultMessage || c.End() != nil {
		t.Fatalf("fragmented decoding %d", len(got))
	}
	c = testCodec(t, 64)
	messages, err := c.Feed([]byte(`{"type":"user"` + strings.Repeat(" ", 64)))
	if !errors.Is(err, ErrProtocolLimit) || len(messages) != 1 || !messages[0].Truncated() || len(messages[0].Raw()) != 64 {
		t.Fatalf("oversized line: %v %+v", err, messages)
	}
	if _, err := c.Feed([]byte("{}\n")); !errors.Is(err, ErrProtocolLimit) {
		t.Fatal("failure did not latch")
	}
	c = testCodec(t, 1<<20)
	if _, err := c.Feed([]byte(`{"type":"result"`)); err != nil {
		t.Fatal(err)
	}
	if err := c.End(); !errors.Is(err, ErrProtocolIncomplete) || string(c.Pending()) != `{"type":"result"` {
		t.Fatalf("unfinished line: %v %q", err, c.Pending())
	}
}
