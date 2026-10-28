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

// fixtureTools is the launch's exact tool list in the CLI's order.
const fixtureTools = `["Bash","Read","Edit","Write","Glob","Grep","mcp__markitect__delegate","mcp__markitect__await","mcp__markitect__review","mcp__markitect__ask","mcp__markitect__reply","mcp__markitect__judge","mcp__markitect__check","mcp__markitect__read_notes","mcp__markitect__append_note"]`

func fixtureInit(session string, servers string) string {
	return `{"type":"system","subtype":"init","cwd":"/work","session_id":"` + session + `","tools":` + fixtureTools + `,"mcp_servers":` + servers + `,"model":"claude-sonnet-5","permissionMode":"bypassPermissions","slash_commands":[],"apiKeySource":"none","claude_code_version":"2.1.308","output_style":"default","agents":[],"skills":[],"plugins":[],"uuid":"00000000-0000-4000-8000-0000000000a1","future":{"visible":true}}`
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

// capturedInit is the system/init line the pinned CLI printed for the
// adapter's launch arguments and environment, captured in the Go runtime
// image without network, with a dummy login and a stand-in Run tool server
// (K1b, 28 October 2026). Without --settings launchSettings the same launch
// loaded capturedDefaultPlugins instead of none.
const capturedInit = `{"type":"system","subtype":"init","cwd":"/work","session_id":"00000000-0000-4000-8000-000000000001","tools":["Bash","Edit","Glob","Grep","Read","Write","mcp__markitect__append_note","mcp__markitect__ask","mcp__markitect__await","mcp__markitect__check","mcp__markitect__delegate","mcp__markitect__judge","mcp__markitect__read_notes","mcp__markitect__reply","mcp__markitect__review"],"mcp_servers":[{"name":"markitect","status":"connected","source":"dynamic"}],"model":"claude-sonnet-5-5","permissionMode":"bypassPermissions","slash_commands":[],"apiKeySource":"none","claude_code_version":"2.1.308","output_style":"default","agents":[],"skills":[],"plugins":[],"capabilities":["interrupt_receipt_v1","interrupt_cancel_queued_v1","interrupt_send_now_v1","interrupt_requeue_taken_v1","msg_lifecycle_v1","request_marker_lists_v1","lifecycle_wakeup_source_v1","sdk_mcp_tools_list_changed","sdk_mcp_manifests","mcp_read_resource_v1","mcp_tool_ui_meta_v1","ui_surface_v1","mcp_call_result_not_for_model_v1"],"analytics_disabled":true,"product_feedback_disabled":true,"uuid":"d02709de-9120-4245-97f9-fa30a195fe75","original_cwd":"/work","additional_directories":[],"messaging_socket_path":"/tmp/cc-socks/23.sock","fast_mode_state":"off","fast_mode_disabled_reason":"sdk_opt_in_required","per_turn_effort_active":true,"view_mode":"default"}`

const capturedDefaultPlugins = `[{"name":"cc-plugin-agents-md","path":"builtin","source":"cc-plugin-agents-md@builtin"},{"name":"cc-plugin-plugin-authoring","path":"builtin","source":"cc-plugin-plugin-authoring@builtin"}]`

// The pinned CLI's own init line passes the launch check only with the
// built-in plugins turned off, and launchSettings turns off every plugin it
// loads by default.
func TestPinnedCLIInitPassesTheLaunchCheck(t *testing.T) {
	if err := checkInit(decodeOne(t, capturedInit).Init()); err != nil {
		t.Fatal(err)
	}
	loaded := strings.Replace(capturedInit, `"plugins":[]`, `"plugins":`+capturedDefaultPlugins, 1)
	if err := checkInit(decodeOne(t, loaded).Init()); err == nil || !strings.Contains(err.Error(), "none may load") {
		t.Fatalf("built-in plugins loaded: %v", err)
	}
	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	var plugins []struct{ Source string }
	if json.Unmarshal([]byte(launchSettings), &settings) != nil || json.Unmarshal([]byte(capturedDefaultPlugins), &plugins) != nil {
		t.Fatal("settings or capture is not JSON")
	}
	for _, p := range plugins {
		if enabled, ok := settings.EnabledPlugins[p.Source]; !ok || enabled {
			t.Fatalf("launch settings leave %s on", p.Source)
		}
	}
	for name, enabled := range settings.EnabledPlugins {
		if enabled {
			t.Fatalf("launch settings turn %s on", name)
		}
	}
}
