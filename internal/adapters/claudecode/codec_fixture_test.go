package claudecode

import (
	"strings"
	"testing"
)

// Independently authored synthetic examples, not provider captures, copied SDK
// schemas or proof of runtime support. UUIDs are illustrative caller inputs.
const (
	fixtureSession    = "00000000-0000-4000-8000-000000000001"
	fixtureUUID       = "00000000-0000-4000-8000-000000000002"
	fixtureInitialize = `{"type":"control_response","response":{"subtype":"success","request_id":"synthetic-control","response":{"commands":[],"agents":[],"models":[],"output_style":"default","available_output_styles":["default"],"account":{},"claude_code_version":"2.1.301","future":{"visible":true}}}}`
	fixtureInit       = `{"type":"system","subtype":"init","session_id":"` + fixtureSession + `","uuid":"` + fixtureUUID + `","claude_code_version":"2.1.301","apiKeySource":"none","cwd":"/work","tools":[],"mcp_servers":[],"model":"synthetic","permissionMode":"dontAsk","slash_commands":[],"output_style":"default","skills":[],"plugins":[]}`
	fixtureResult     = `{"type":"result","subtype":"success","session_id":"` + fixtureSession + `","uuid":"` + fixtureUUID + `","is_error":false,"result":"synthetic result","duration_ms":1,"duration_api_ms":1,"num_turns":1,"stop_reason":"end_turn","total_cost_usd":0,"usage":{},"modelUsage":{},"permission_denials":[],"terminal_reason":"completed","structured_output":{"answer":3},"user_message_uuid":"caller-user-uuid","user_message_uuids":["another-user-uuid","caller-user-uuid"]}`
)

func newTestCodec(t *testing.T, limit int) *Codec {
	t.Helper()
	c, err := NewCodec(limit)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func decodeFixture(t *testing.T, line string) Message {
	t.Helper()
	c := newTestCodec(t, maxLineBytes)
	messages, err := c.Feed([]byte(line + "\n"))
	if err != nil || len(messages) != 1 {
		t.Fatalf("fixture: %v, %d messages", err, len(messages))
	}
	return messages[0]
}
func errorResult(subtype string) string {
	s := strings.Replace(fixtureResult, `"subtype":"success"`, `"subtype":"`+subtype+`"`, 1)
	s = strings.Replace(s, `"is_error":false`, `"is_error":true`, 1)
	return strings.Replace(s, `"result":"synthetic result"`, `"errors":["synthetic refusal"]`, 1)
}
