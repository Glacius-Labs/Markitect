package codexappserver

import "testing"

// These are authored synthetic wire examples of the selected pinned schema
// shapes, not captures, copied schemas or evidence of provider execution.
const codecThread = `{"id":"01900000-0000-7000-8000-000000000001","cliVersion":"0.162.0","createdAt":1,"updatedAt":2,"cwd":"/work","ephemeral":false,"modelProvider":"openai","preview":"synthetic","projectId":null,"sessionId":"01900000-0000-7000-8000-000000000003","source":"appServer","status":{"type":"idle"},"turns":[]}`
const codecThreadID ThreadID = "01900000-0000-7000-8000-000000000001"
const codecTurnID TurnID = "01900000-0000-7000-8000-000000000002"
const codecInitialization = `{"codexHome":"/home/agent/.codex","platformFamily":"unix","platformOs":"linux","userAgent":"synthetic/0.162.0"}`

func testCodec(t *testing.T, limit int) *Codec {
	t.Helper()
	c, err := NewCodec(limit)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func codecMessage(t *testing.T, data string) Message {
	t.Helper()
	c := testCodec(t, maxProtocolMessage)
	messages, err := c.Feed([]byte(data + "\n"))
	if err != nil || len(messages) != 1 {
		t.Fatalf("decode fixture: %v %+v", err, messages)
	}
	return messages[0]
}

func stringID(t *testing.T, value string) RequestID {
	t.Helper()
	id, err := StringRequestID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
