package claudecode

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEncodingUsesOnlyCallerInputs(t *testing.T) {
	c := newTestCodec(t, maxLineBytes)
	init, err := c.EncodeInitialize("supplied-control")
	if err != nil || string(init) != "{\"type\":\"control_request\",\"request_id\":\"supplied-control\",\"request\":{\"subtype\":\"initialize\"}}\n" {
		t.Fatalf("initialize: %q %v", init, err)
	}
	user, err := c.EncodeUser(UserInput{Text: "literal <text>\n😀"})
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(user, &envelope) != nil || len(envelope) != 3 || string(envelope["parent_tool_use_id"]) != "null" {
		t.Fatalf("invented envelope fields: %s", user)
	}
	var body struct{ Role, Content string }
	if json.Unmarshal(envelope["message"], &body) != nil || body.Role != "user" || body.Content != "literal <text>\n😀" {
		t.Fatalf("user input changed: %s", user)
	}
	user, err = c.EncodeUser(UserInput{Text: "", UUID: "caller-uuid", Session: "requested-session", ClientComposed: true})
	if err != nil || !bytes.HasSuffix(user, []byte("\n")) {
		t.Fatalf("explicit input: %q %v", user, err)
	}
	if json.Unmarshal(user, &envelope) != nil || string(envelope["uuid"]) != `"caller-uuid"` || string(envelope["session_id"]) != `"requested-session"` || string(envelope["client_composed"]) != "true" {
		t.Fatalf("caller fields lost: %s", user)
	}
	for _, bad := range []UserInput{{Text: string([]byte{0xff})}, {UUID: UserMessageUUID(string([]byte{0xff}))}, {Session: RequestedSessionID(string([]byte{0xff}))}} {
		if _, err := c.EncodeUser(bad); !errors.Is(err, ErrProtocol) {
			t.Fatalf("invalid UTF8 replaced: %v", err)
		}
	}
	for _, bad := range []ControlRequestID{"", ControlRequestID(string([]byte{0xff}))} {
		if _, err := c.EncodeInitialize(bad); !errors.Is(err, ErrProtocol) {
			t.Fatalf("invalid ID encoded: %v", err)
		}
	}
	for _, limit := range []int{0, -1, maxLineBytes + 1} {
		if _, err := NewCodec(limit); !errors.Is(err, ErrLimit) {
			t.Fatalf("accepted limit %d", limit)
		}
	}
	if _, err := newTestCodec(t, 32).EncodeUser(UserInput{Text: strings.Repeat("\x00", 32)}); !errors.Is(err, ErrLimit) {
		t.Fatalf("escaped output exceeded limit: %v", err)
	}
	if _, err := newTestCodec(t, 32).EncodeUser(UserInput{Text: strings.Repeat("a", 33)}); !errors.Is(err, ErrLimit) {
		t.Fatalf("oversized text: %v", err)
	}
}

func TestControlAcknowledgmentAndObservedSessionAreSeparate(t *testing.T) {
	m := decodeFixture(t, fixtureInitialize)
	if m.Kind() != ControlResponseMessage {
		t.Fatal(m.Kind())
	}
	r, err := m.ReadInitializeResponse("synthetic-control")
	if err != nil || r.Status != ControlSuccess || r.Version != "2.1.301" {
		t.Fatalf("control observation: %+v %v", r, err)
	}
	if _, err := m.Init(); !errors.Is(err, ErrUnsupported) {
		t.Fatal("control reply supplied a session binding")
	}
	if _, err := m.ReadInitializeResponse("different-control"); !errors.Is(err, ErrProtocol) {
		t.Fatal("wrong control ID matched")
	}
	if !bytes.Contains(m.Raw(), []byte(`"future":{"visible":true}`)) {
		t.Fatal("unknown initialize field lost")
	}
	i, err := decodeFixture(t, fixtureInit).Init()
	if err != nil || i.Session != ObservedSessionID(fixtureSession) || i.UUID != MessageUUID(fixtureUUID) {
		t.Fatalf("init: %+v %v", i, err)
	}
	if _, err := decodeFixture(t, fixtureInit).ReadInitializeResponse("synthetic-control"); err == nil {
		t.Fatal("init treated as control ack")
	}
	e := decodeFixture(t, `{"type":"control_response","response":{"subtype":"error","request_id":"synthetic-control","error":"synthetic refusal"}}`)
	r, err = e.ReadInitializeResponse("synthetic-control")
	if err != nil || r.Status != ControlError || r.Error != "synthetic refusal" {
		t.Fatalf("error reply became ack: %+v %v", r, err)
	}
	for _, mutation := range []struct{ old, new string }{
		{`"commands":[]`, `"commands":null`}, {`"agents":[]`, `"agents":{}`}, {`"models":[]`, `"models":false`},
		{`"output_style":"default"`, `"output_style":null`}, {`"available_output_styles":["default"]`, `"available_output_styles":[null]`},
		{`"account":{}`, `"account":[]`}, {`"claude_code_version":"2.1.301"`, `"claude_code_version":1`},
	} {
		bad := decodeFixture(t, strings.Replace(fixtureInitialize, mutation.old, mutation.new, 1))
		if _, err := bad.ReadInitializeResponse("synthetic-control"); err == nil {
			t.Fatalf("malformed selected initializer accepted: %s", mutation.new)
		}
	}
	ack := decodeFixture(t, `{"type":"control_response","response":{"subtype":"success","request_id":"synthetic-control"}}`)
	if _, err := ack.ReadInitializeResponse("synthetic-control"); err == nil {
		t.Fatal("empty general ack passed initialize projection")
	}
}

func TestResultsPreserveIndependentStatusesAndIdentities(t *testing.T) {
	for _, subtype := range []string{"success", "error_during_execution", "error_max_turns", "error_max_budget_usd", "error_max_structured_output_retries"} {
		t.Run(subtype, func(t *testing.T) {
			frame := fixtureResult
			if subtype != "success" {
				frame = errorResult(subtype)
			}
			m := decodeFixture(t, frame)
			r, err := m.Result()
			if err != nil || r.Subtype != ResultSubtype(subtype) || r.UUID != ResultUUID(fixtureUUID) || r.Session != ObservedSessionID(fixtureSession) {
				t.Fatalf("result: %+v %v", r, err)
			}
			if r.IsError != (subtype != "success") {
				t.Fatal("independent error status lost")
			}
			if !bytes.Contains(m.Raw(), []byte(`"user_message_uuids":["another-user-uuid","caller-user-uuid"]`)) {
				t.Fatal("multiple user echoes lost")
			}
			if subtype != "success" && (len(r.Errors) != 1 || r.Errors[0] != "synthetic refusal") {
				t.Fatal("error data lost")
			}
		})
	}
	m := decodeFixture(t, strings.Replace(fixtureResult, `"is_error":false`, `"is_error":true`, 1))
	r, err := m.Result()
	if err != nil || r.Subtype != ResultSuccess || !r.IsError {
		t.Fatalf("success subtype erased error: %+v %v", r, err)
	}
	// Returned structured data is not a mutable view into retained wire evidence.
	r.StructuredOutput[0] = '['
	again, _ := m.Result()
	if string(again.StructuredOutput) != `{"answer":3}` {
		t.Fatal("result bytes alias internal state")
	}
	unknownReason := decodeFixture(t, strings.Replace(fixtureResult, `"completed"`, `"future-reason"`, 1))
	r, err = unknownReason.Result()
	if err != nil || r.TerminalReason != "future-reason" {
		t.Fatal("uninterpreted terminal reason lost")
	}
}

func TestKindsStayVisibleAndOpaque(t *testing.T) {
	for _, tc := range []struct {
		line string
		kind MessageKind
	}{
		{`{"type":"future","opaque":{"x":1}}`, UnsupportedMessage},
		{`{"type":"system","subtype":"future","opaque":true}`, UnsupportedMessage},
		{strings.Replace(fixtureResult, `"subtype":"success"`, `"subtype":"future"`, 1), UnsupportedMessage},
		{`{"type":"control_response","response":{"subtype":"future","request_id":"x"}}`, UnsupportedMessage},
		{`{"type":"control_request","request_id":"x","request":{"subtype":"mcp_message","message":{"opaque":true}}}`, UnsupportedRequest},
		{`{"type":"user","message":{"role":"user","content":"synthetic"},"parent_tool_use_id":null}`, UserMessage},
		{`{"type":"assistant","message":{"role":"assistant","content":[{"type":"future","opaque":true}]},"uuid":"x","session_id":"s","parent_tool_use_id":null}`, AssistantMessage},
		{`{"type":"stream_event","event":{"type":"future"},"uuid":"x","session_id":"s","parent_tool_use_id":null}`, StreamEventMessage},
	} {
		m := decodeFixture(t, tc.line)
		if m.Kind() != tc.kind || string(m.Raw()) != tc.line+"\n" {
			t.Fatalf("kind/raw: %s %q", m.Kind(), m.Raw())
		}
		if _, err := m.Result(); !errors.Is(err, ErrUnsupported) {
			t.Fatal("non-result yielded result")
		}
	}
}

func TestMalformedSelectedFieldsAreRejected(t *testing.T) {
	for _, line := range []string{
		`null`, `[]`, `{}`, `{"type":null}`,
		`{"type":"system","subtype":"init","session_id":"s","uuid":"x"}`,
		strings.Replace(fixtureInit, `"session_id":"`+fixtureSession+`"`, `"session_id":null`, 1),
		strings.Replace(fixtureInit, `"uuid":"`+fixtureUUID+`"`, `"uuid":""`, 1),
		strings.Replace(fixtureResult, `"is_error":false`, `"is_error":null`, 1),
		strings.Replace(fixtureResult, `"result":"synthetic result"`, `"result":null`, 1),
		strings.Replace(errorResult("error_max_turns"), `"errors":["synthetic refusal"]`, `"errors":[null]`, 1),
		`{"type":"control_response","response":{"subtype":"success","request_id":1}}`,
		`{"type":"control_response","response":{"subtype":"error","request_id":"x","error":null}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"x","response":null}}`,
		`{"type":"control_request","request_id":"x","request":null}`,
		`{"type":"assistant","message":{"role":"assistant","content":"wrong"},"uuid":"x","session_id":"s","parent_tool_use_id":null}`,
		`{"type":"user","message":{"role":"assistant","content":"wrong"},"parent_tool_use_id":null}`,
		`{"type":"stream_event","event":[],"uuid":"x","session_id":"s","parent_tool_use_id":null}`,
	} {
		c := newTestCodec(t, maxLineBytes)
		ms, err := c.Feed([]byte(line + "\n"))
		if err == nil || len(ms) != 1 || ms[0].Kind() != RejectedMessage || string(ms[0].Raw()) != line+"\n" {
			t.Fatalf("malformed selected shape escaped: %s %v", line, err)
		}
	}
}

func TestFragmentedFramesPreserveExactBoundaryAndCopies(t *testing.T) {
	frame := []byte(" \t{\"type\":\"future\",\"text\":\"😀\"} \r\n")
	want := bytes.Clone(frame)
	c := newTestCodec(t, len(frame))
	var messages []Message
	for _, b := range frame {
		ms, err := c.Feed([]byte{b})
		if err != nil {
			t.Fatal(err)
		}
		messages = append(messages, ms...)
	}
	if c.End() != nil || len(messages) != 1 || messages[0].Digest() != sha256.Sum256(want) {
		t.Fatal("fragmented wire identity changed")
	}
	frame[0] = '!'
	raw := messages[0].Raw()
	raw[0] = '!'
	if !bytes.Equal(messages[0].Raw(), want) {
		t.Fatal("raw observation aliases caller bytes")
	}
	ms, err := c.Feed(append(bytes.Clone(want), want...))
	if err != nil || len(ms) != 2 {
		t.Fatalf("multiple frames: %v %d", err, len(ms))
	}
	c = newTestCodec(t, 64)
	fragment := []byte(`{"type":"future"}`)
	if _, err := c.Feed(fragment); err != nil {
		t.Fatal(err)
	}
	fragment[0] = '!'
	pending := c.Pending()
	pending[0] = '!'
	if string(c.Pending()) != `{"type":"future"}` {
		t.Fatal("pending aliases input/output")
	}
	if !errors.Is(c.End(), ErrIncomplete) || bytes.HasSuffix(c.Pending(), []byte("\n")) {
		t.Fatal("End invented a frame boundary")
	}
	if ms, err := c.Feed([]byte("\n")); !errors.Is(err, ErrIncomplete) || len(ms) != 0 {
		t.Fatal("incomplete End did not latch")
	}
}

func TestJSONIdentityGuardsAndDepth(t *testing.T) {
	for _, bad := range []string{
		`{"type":"future","type":"future"}`, `{"type":"future","x":{"a":1,"\u0061":2}}`,
		`{"type":"future","x":"\ud800"}`, `{"type":"future","x":"\udfff"}`,
		`{"type":"future","\ud800":1}`, `{"type":"future"} {"type":"future"}`,
		`{"type":"future"} true`, `{"type":"future","x":NaN}`, `{"type":"future","x":"` + string([]byte{0xff}) + `"}`,
	} {
		c := newTestCodec(t, maxLineBytes)
		if _, err := c.Feed([]byte(bad + "\n")); !errors.Is(err, ErrProtocol) {
			t.Fatalf("JSON guard accepted %q: %v", bad, err)
		}
	}
	for _, good := range []string{`{"type":"future","id":"\ud83d\ude00"}`, `{"type":"future","id":"\\ud800"}`, `{"type":"future","id":"�"}`} {
		decodeFixture(t, good)
	}
	for _, count := range []int{63, 64} {
		line := `{"type":"future","x":` + strings.Repeat("[", count) + "0" + strings.Repeat("]", count) + "}\n"
		_, err := newTestCodec(t, maxLineBytes).Feed([]byte(line))
		if count == 63 && err != nil || count == 64 && !errors.Is(err, ErrLimit) {
			t.Fatalf("compound depth %d: %v", count+1, err)
		}
	}
}

func TestFailureRetainsBoundedEvidenceAndLatches(t *testing.T) {
	good := "{\"type\":\"future\"}\n"
	bad := "{\"type\":null}\n"
	c := newTestCodec(t, 32)
	ms, err := c.Feed([]byte(good + bad + good))
	if !errors.Is(err, ErrProtocol) || len(ms) != 2 || ms[0].Kind() != UnsupportedMessage || ms[1].Kind() != RejectedMessage || string(c.Pending()) != bad {
		t.Fatalf("partial failure: %+v %v", ms, err)
	}
	if again, err := c.Feed([]byte(good)); !errors.Is(err, ErrProtocol) || len(again) != 0 || !errors.Is(c.End(), ErrProtocol) {
		t.Fatal("failure not latched")
	}
	c = newTestCodec(t, len(good))
	if _, err := c.Feed([]byte(good)); err != nil {
		t.Fatal("LF-inclusive exact limit rejected")
	}
	c = newTestCodec(t, len(good)-1)
	ms, err = c.Feed([]byte(good))
	if !errors.Is(err, ErrLimit) || len(ms) != 1 || !ms[0].Truncated() || ms[0].Digest() != sha256.Sum256([]byte(good[:len(good)-1])) {
		t.Fatal("overflow prefix misidentified as full frame")
	}
	c = newTestCodec(t, 32)
	if _, err := c.Feed([]byte("{")); err != nil {
		t.Fatal(err)
	}
	ms, err = c.Feed([]byte(strings.Repeat("x", maxLineBytes*2) + "\n" + good))
	if !errors.Is(err, ErrLimit) || len(ms) != 1 || len(ms[0].Raw()) != 32 || len(c.Pending()) != 32 || !ms[0].Truncated() {
		t.Fatal("large fragment not bounded before retention")
	}
}

func TestRateLimitEventRetainsKnownInformationalEnvelopeAndFrameBound(t *testing.T) {
	raw := []byte(fixtureRateLimit + "\n")
	messages, err := newTestCodec(t, len(raw)).Feed(raw)
	if err != nil || len(messages) != 1 || messages[0].Kind() != RateLimitEventMessage || !bytes.Equal(messages[0].Raw(), raw) || messages[0].Digest() != sha256.Sum256(raw) {
		t.Fatalf("informational observation lost raw identity: %+v %v", messages, err)
	}
	if _, err := messages[0].Result(); !errors.Is(err, ErrUnsupported) {
		t.Fatal("rate event became an outcome", err)
	}
	if _, err := newTestCodec(t, len(raw)-1).Feed(raw); !errors.Is(err, ErrLimit) {
		t.Fatal("rate event escaped frame bound", err)
	}
	for _, mutation := range []struct{ from, to string }{
		{`"rate_limit_info":{"status":"allowed","future":{"visible":true}}`, `"rate_limit_info":null`},
		{`"rate_limit_info":{"status":"allowed","future":{"visible":true}}`, `"rate_limit_info":[]`},
		{`"uuid":"` + fixtureUUID + `"`, `"uuid":""`},
		{`"session_id":"` + fixtureSession + `"`, `"session_id":1`},
		{`,"uuid":"` + fixtureUUID + `"`, ""},
		{`"rate_limit_event",`, `"rate_limit_event","extra":true,`},
	} {
		raw := strings.Replace(fixtureRateLimit, mutation.from, mutation.to, 1)
		messages, err := newTestCodec(t, maxLineBytes).Feed([]byte(raw + "\n"))
		if !errors.Is(err, ErrProtocol) || len(messages) != 1 || messages[0].Kind() != RejectedMessage {
			t.Fatalf("invalid rate envelope accepted: %s %v", raw, err)
		}
	}
}
