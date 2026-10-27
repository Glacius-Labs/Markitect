package claudecode

import (
	"encoding/json"
	"strings"
	"testing"
)

// Independently authored synthetic lines in the shapes of the documented
// stream-json output of the pinned CLI. They are not provider captures and do
// not prove runtime support; native qualification K1b checks them.
const fixtureSession = "00000000-0000-4000-8000-000000000001"

func fixtureInit(session string, servers string) string {
	return `{"type":"system","subtype":"init","cwd":"/work","session_id":"` + session + `","tools":["Bash","Read","Edit","Write","Glob","Grep","mcp__markitect__await"],"mcp_servers":` + servers + `,"model":"claude-sonnet-5","permissionMode":"bypassPermissions","slash_commands":[],"apiKeySource":"none","claude_code_version":"2.1.308","output_style":"default","agents":[],"skills":[],"plugins":[],"uuid":"00000000-0000-4000-8000-0000000000a1","future":{"visible":true}}`
}

const connectedServers = `[{"name":"markitect","status":"connected"}]`

func fixtureRetry(session, category string) string {
	return `{"type":"system","subtype":"api_retry","attempt":1,"max_retries":10,"retry_delay_ms":500,"error_status":529,"error":"` + category + `","uuid":"00000000-0000-4000-8000-0000000000a2","session_id":"` + session + `"}`
}

func fixtureAssistant(session string, blocks ...string) string {
	return `{"type":"assistant","message":{"id":"msg_01","type":"message","role":"assistant","model":"claude-sonnet-5","content":[` + strings.Join(blocks, ",") + `],"stop_reason":null,"usage":{"input_tokens":1,"output_tokens":1}},"parent_tool_use_id":null,"session_id":"` + session + `","uuid":"00000000-0000-4000-8000-0000000000a3"}`
}

func toolUse(id, name string) string {
	return `{"type":"tool_use","id":"` + id + `","name":"` + name + `","input":{"command":"go test ./..."}}`
}

func textBlock(text string) string {
	data, _ := json.Marshal(map[string]string{"type": "text", "text": text})
	return string(data)
}

func fixtureToolResult(session, id string) string {
	return `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"` + id + `","type":"tool_result","content":"ok","is_error":false}]},"parent_tool_use_id":null,"session_id":"` + session + `","uuid":"00000000-0000-4000-8000-0000000000a4","tool_use_result":{"stdout":"ok"}}`
}

func fixtureResult(session, text string) string {
	data, _ := json.Marshal(text)
	return `{"type":"result","subtype":"success","is_error":false,"duration_ms":10,"duration_api_ms":8,"num_turns":2,"result":` + string(data) + `,"stop_reason":"end_turn","session_id":"` + session + `","total_cost_usd":0,"usage":{},"modelUsage":{},"permission_denials":[],"uuid":"00000000-0000-4000-8000-0000000000a5"}`
}

// fixtureFailure is the end of a turn on an API error after Claude Code's own
// retries: a synthetic assistant message naming the category, then an error
// result.
func fixtureFailure(session, category string) []string {
	return []string{
		`{"type":"assistant","message":{"id":"msg_02","type":"message","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"API Error: 529 Overloaded"}],"stop_reason":"stop_sequence","usage":{}},"parent_tool_use_id":null,"error":"` + category + `","session_id":"` + session + `","uuid":"00000000-0000-4000-8000-0000000000a6"}`,
		`{"type":"result","subtype":"success","is_error":true,"api_error_status":529,"duration_ms":10,"duration_api_ms":8,"num_turns":1,"result":"API Error: 529 Overloaded","stop_reason":null,"session_id":"` + session + `","total_cost_usd":0,"usage":{},"modelUsage":{},"permission_denials":[],"uuid":"00000000-0000-4000-8000-0000000000a7"}`,
	}
}

func testCodec(t *testing.T, limit int) *Codec {
	t.Helper()
	c, err := NewCodec(limit)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func decodeOne(t *testing.T, line string) Message {
	t.Helper()
	messages, err := testCodec(t, maxProtocolMessage).Feed([]byte(line + "\n"))
	if err != nil || len(messages) != 1 {
		t.Fatalf("fixture: %v, %d messages: %s", err, len(messages), line)
	}
	return messages[0]
}
