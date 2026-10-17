package codexappserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCodecEncodesSelectedPinnedRequests(t *testing.T) {
	c := testCodec(t, 4096)
	initialize, err := c.EncodeInitialize(stringID(t, "host-request"), ClientInfo{Name: "markitect", Version: "synthetic"})
	if err != nil || string(initialize) != "{\"id\":\"host-request\",\"method\":\"initialize\",\"params\":{\"clientInfo\":{\"name\":\"markitect\",\"version\":\"synthetic\"}}}\n" {
		t.Fatalf("initialize: %s %v", initialize, err)
	}
	initialized, err := c.EncodeInitialized()
	if err != nil || string(initialized) != "{\"method\":\"initialized\"}\n" {
		t.Fatalf("initialized: %s %v", initialized, err)
	}
	thread, err := c.EncodeThreadStart(IntegerRequestID(17), ThreadStartParams{Model: "configured-model", Cwd: "/work", ApprovalPolicy: "never", Sandbox: "danger-full-access", BaseInstructions: "explicit\ninstruction", DeveloperInstructions: ""})
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		ID     int
		Method string
		Params map[string]json.RawMessage
	}
	if err := json.Unmarshal(thread, &request); err != nil || request.ID != 17 || request.Method != "thread/start" || string(request.Params["baseInstructions"]) != `"explicit\ninstruction"` || string(request.Params["developerInstructions"]) != `""` || len(request.Params) != 6 {
		t.Fatalf("thread shape: %s %v", thread, err)
	}
	turn, err := c.EncodeTurnStart(IntegerRequestID(18), TurnStartParams{Thread: codecThreadID, Text: "unchanged operation\nexplicit context", Effort: "configured-effort", OutputSchema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"],"additionalProperties":false}`)})
	if err != nil {
		t.Fatal(err)
	}
	var turnRequest struct {
		ID     int
		Method string
		Params struct {
			Thread       string `json:"threadId"`
			Input        []struct{ Type, Text string }
			Effort       string
			OutputSchema json.RawMessage
		}
	}
	if err := json.Unmarshal(turn, &turnRequest); err != nil || turnRequest.ID != 18 || turnRequest.Method != "turn/start" || turnRequest.Params.Thread != string(codecThreadID) || len(turnRequest.Params.Input) != 1 || turnRequest.Params.Input[0].Type != "text" || turnRequest.Params.Input[0].Text != "unchanged operation\nexplicit context" || turnRequest.Params.Effort != "configured-effort" || !json.Valid(turnRequest.Params.OutputSchema) {
		t.Fatalf("turn shape: %s %v", turn, err)
	}
	interrupt, err := c.EncodeTurnInterrupt(IntegerRequestID(19), codecThreadID, codecTurnID)
	if err != nil || !bytes.Contains(interrupt, []byte(`"turnId":"`+string(codecTurnID)+`"`)) || bytes.Count(interrupt, []byte{'\n'}) != 1 {
		t.Fatalf("interrupt: %s %v", interrupt, err)
	}
}

// A resume asks for the thread without its turns: the adapter never reads the
// history, and a long Manager thread would outgrow the wire limit.
func TestCodecResumeExcludesTheThreadHistory(t *testing.T) {
	c := testCodec(t, 1<<20)
	params := ThreadStartParams{Model: "configured-model", Cwd: "/work", ApprovalPolicy: "never", Sandbox: "danger-full-access"}
	var request struct {
		Method string
		Params map[string]json.RawMessage
	}
	resume, err := c.EncodePersistentThread(IntegerRequestID(2), params, codecThreadID)
	if err != nil || json.Unmarshal(resume, &request) != nil || request.Method != "thread/resume" || string(request.Params["threadId"]) != `"`+string(codecThreadID)+`"` || string(request.Params["excludeTurns"]) != "true" {
		t.Fatalf("resume shape: %s %v", resume, err)
	}
	start, err := c.EncodePersistentThread(IntegerRequestID(2), params, "")
	request.Params = nil
	if err != nil || json.Unmarshal(start, &request) != nil || request.Method != "thread/start" || request.Params["excludeTurns"] != nil || string(request.Params["ephemeral"]) != "false" {
		t.Fatalf("start shape: %s %v", start, err)
	}
}

func TestCodecFragmentsRetainExactCopiedNDJSONIdentity(t *testing.T) {
	original := []byte(" \t{\"id\":17,\"result\":" + codecInitialization + "} \r\n")
	c := testCodec(t, len(original))
	var messages []Message
	for i := range original {
		got, err := c.Feed(original[i : i+1])
		if err != nil {
			t.Fatal(err)
		}
		messages = append(messages, got...)
	}
	if len(messages) != 1 || c.End() != nil || !bytes.Equal(messages[0].Raw(), original) || messages[0].Digest() != sha256.Sum256(original) {
		t.Fatal("fragmented raw boundary changed")
	}
	want := bytes.Clone(original)
	original[0] = '!'
	returned := messages[0].Raw()
	returned[1] = '!'
	if !bytes.Equal(messages[0].Raw(), want) {
		t.Fatal("raw bytes alias input or returned copy")
	}
	r, err := messages[0].ReadResponse(IntegerRequestID(17), Initialize)
	if err != nil || r.Initialize.CodexHome != "/home/agent/.codex" || r.Initialize.PlatformOS != "linux" {
		t.Fatal(r, err)
	}
	if _, err := messages[0].ReadResponse(stringID(t, "17"), Initialize); err == nil {
		t.Fatal("numeric ID became a string ID")
	}
	if _, err := messages[0].ReadResponse(IntegerRequestID(18), Initialize); err == nil {
		t.Fatal("different request admitted")
	}
}

func TestCodecMalformedLinesPreserveRejectedBytesAndLatchFailure(t *testing.T) {
	for name, line := range map[string]string{
		"invalid UTF8": "{\"method\":\"bad\xff\"}",
		"blank":        "", "array": `[]`, "null": `null`,
		"duplicate":         `{"id":1,"id":2,"result":{}}`,
		"escaped duplicate": `{"id":1,"\u0069d":2,"result":{}}`,
		"nested duplicate":  `{"id":1,"result":{"outer":[{"a":1,"a":2}]}}`,
		"trailing":          `{"id":1,"result":{}} {"method":"later"}`,
		"bad JSON":          `{"id":1,"result":`,
		"null id":           `{"id":null,"result":{}}`,
		"boolean id":        `{"id":true,"result":{}}`,
		"fraction id":       `{"id":1.5,"result":{}}`,
		"exponent id":       `{"id":1e2,"result":{}}`,
		"overflow id":       `{"id":9223372036854775808,"result":{}}`,
		"method null":       `{"method":null}`, "no union arm": `{}`,
		"response params":    `{"id":1,"result":{},"params":{}}`,
		"request result":     `{"id":1,"method":"unknown","result":{}}`,
		"both result error":  `{"id":1,"result":{},"error":{"code":1,"message":"bad"}}`,
		"error code type":    `{"id":1,"error":{"code":"1","message":"bad"}}`,
		"error message null": `{"id":1,"error":{"code":1,"message":null}}`,
		"deep":               `{"id":1,"result":` + strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65) + `}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := testCodec(t, 4096)
			prefix := "{\"method\":\"visible/before\"}\n"
			messages, err := c.Feed([]byte(prefix + line + "\n{\"method\":\"not/interpreted\"}\n"))
			if err == nil || len(messages) != 2 || messages[0].Kind() != UnknownNotification || messages[1].Kind() != RejectedMessage || string(messages[1].Raw()) != line+"\n" || messages[1].Truncated() {
				t.Fatalf("lost rejected or earlier observation: %v %+v", err, messages)
			}
			later, nextErr := c.Feed([]byte("{\"method\":\"later\"}\n"))
			if len(later) != 0 || nextErr != err || c.End() != err {
				t.Fatal("fatal decoder silently continued", nextErr)
			}
		})
	}
}

func TestCodecUnknownMessagesRemainVisibleWithoutExecutionOrSuccess(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		kind MessageKind
	}{
		{`{"method":"future/critical","params":{"status":"completed","id":"untrusted"}}`, UnknownNotification},
		{`{"id":"server-call","method":"item/tool/call","params":{"name":"never-executed"}}`, UnsupportedRequest},
	} {
		m := codecMessage(t, tc.raw)
		if m.Kind() != tc.kind || string(m.Raw()) != tc.raw+"\n" || m.Thread() != "" || m.Turn() != (TurnObservation{}) {
			t.Fatal("unknown variant acquired known semantics", m.Kind())
		}
		if _, err := m.ReadResponse(stringID(t, "server-call"), TurnStart); err == nil {
			t.Fatal("unsupported request became success")
		}
	}
	m := codecMessage(t, `{"id":-1,"error":{"code":-32000,"message":"provider refused","data":{"retained":true}}}`)
	code, message, err := m.RPCError()
	if err != nil || m.Kind() != ErrorMessage || code != -32000 || message != "provider refused" || !bytes.Contains(m.Raw(), []byte(`"retained":true`)) {
		t.Fatal("error lost", code, message, err)
	}
	if _, err := m.ReadResponse(IntegerRequestID(-1), ThreadStart); err == nil {
		t.Fatal("provider error became success")
	}
}

func TestCodecObservedProviderHandlesAreNotRequestOrHostIDs(t *testing.T) {
	m := codecMessage(t, `{"id":"host-open-request","result":{"approvalPolicy":"never","approvalsReviewer":"user","cwd":"/work","model":"configured-model","modelProvider":"openai","sandbox":{"type":"dangerFullAccess"},"thread":`+codecThread+`}}`)
	r, err := m.ReadResponse(stringID(t, "host-open-request"), ThreadStart)
	if err != nil || r.Thread != codecThreadID || string(r.Thread) == "host-open-request" {
		t.Fatal("provider identity invented", r, err)
	}
	n := codecMessage(t, `{"method":"thread/started","params":{"thread":`+codecThread+`}}`)
	if n.Kind() != ThreadStartedMessage || n.Thread() != codecThreadID {
		t.Fatal("thread notification lost")
	}
	turn := codecMessage(t, `{"id":2,"result":{"turn":{"id":"`+string(codecTurnID)+`","items":[],"status":"inProgress"}}}`)
	got, err := turn.ReadResponse(IntegerRequestID(2), TurnStart)
	if err != nil || got.Turn.Turn != codecTurnID || got.Turn.Status != TurnInProgress || got.Turn.Thread != "" {
		t.Fatal("response invented thread correlation", got, err)
	}
	if _, err := turn.ReadResponse(IntegerRequestID(2), RequestMethod("future/start")); !errors.Is(err, ErrProtocolUnsupported) {
		t.Fatal("unselected response method accepted", err)
	}
}

func TestCodecTurnEventsKeepTerminalStatusesDistinctAndRawItems(t *testing.T) {
	for _, status := range []TurnStatus{TurnInProgress, TurnCompleted, TurnFailed, TurnInterrupted} {
		method := "turn/completed"
		kind := TurnCompletedMessage
		if status == TurnInProgress {
			method, kind = "turn/started", TurnStartedMessage
		}
		raw := `{"method":"` + method + `","params":{"threadId":"` + string(codecThreadID) + `","turn":{"id":"` + string(codecTurnID) + `","items":[{"type":"futureItem","opaque":"retained"}],"status":"` + string(status) + `"}}}`
		m := codecMessage(t, raw)
		if m.Kind() != kind || m.Turn() != (TurnObservation{codecThreadID, codecTurnID, status}) || !bytes.Contains(m.Raw(), []byte(`"futureItem"`)) {
			t.Fatal("terminal state or unparsed item lost", m.Kind(), m.Turn())
		}
	}
	for _, turn := range []string{`{"id":"t","items":[],"status":"futureStatus"}`, `{"id":"t","items":null,"status":"completed"}`, `{"id":null,"items":[],"status":"completed"}`, `{"id":"t","items":[],"status":"inProgress"}`} {
		line := `{"method":"turn/completed","params":{"threadId":"thread","turn":` + turn + `}}` + "\n"
		messages, err := testCodec(t, 4096).Feed([]byte(line))
		if err == nil || len(messages) != 1 || messages[0].Kind() != RejectedMessage || messages[0].Turn() != (TurnObservation{}) || string(messages[0].Raw()) != line {
			t.Fatal("unsupported/malformed turn became terminal", messages, err)
		}
	}
}

func TestCodecResponseProjectionRejectsMissingOrWrongConsumedFields(t *testing.T) {
	for _, tc := range []struct {
		result string
		method RequestMethod
	}{
		{`{}`, Initialize}, {strings.Replace(codecInitialization, `"linux"`, `null`, 1), Initialize},
		{`{"thread":` + codecThread + `}`, ThreadStart}, {`{"turn":{"id":"t","status":"completed"}}`, TurnStart},
		{`{"turn":{"id":"t","items":[],"status":"future"}}`, TurnStart}, {`{"unexpected":true}`, TurnInterrupt},
	} {
		m := codecMessage(t, `{"id":1,"result":`+tc.result+`}`)
		if _, err := m.ReadResponse(IntegerRequestID(1), tc.method); err == nil {
			t.Fatal("incomplete projection accepted", tc.result)
		}
	}
	if _, err := codecMessage(t, `{"id":1,"result":{}}`).ReadResponse(IntegerRequestID(1), TurnInterrupt); err != nil {
		t.Fatal(err)
	}
}

func TestCodecBoundsIncludeLFAndPreserveUnfinishedOrOversizedPrefix(t *testing.T) {
	line := "{\"method\":\"unknown\"}\n"
	for _, n := range []int{0, -1, maxProtocolMessage + 1} {
		if _, err := NewCodec(n); !errors.Is(err, ErrProtocolLimit) {
			t.Fatal("invalid limit", n, err)
		}
	}
	c := testCodec(t, len(line))
	messages, err := c.Feed([]byte(line + line))
	if err != nil || len(messages) != 2 {
		t.Fatal("limit applied to chunk instead of message", err)
	}
	c = testCodec(t, len(line)-1)
	messages, err = c.Feed([]byte(line + strings.Repeat("x", 10000)))
	if !errors.Is(err, ErrProtocolLimit) || len(messages) != 1 || !messages[0].Truncated() || len(messages[0].Raw()) != len(line)-1 || string(messages[0].Raw()) != line[:len(line)-1] {
		t.Fatal("unbounded or misleading oversized observation", err)
	}
	c = testCodec(t, 4096)
	partial := `{"id":1,"result":{}}`
	if got, err := c.Feed([]byte(partial)); err != nil || len(got) != 0 || !errors.Is(c.End(), ErrProtocolIncomplete) {
		t.Fatal("partial frame was completed implicitly", err)
	}
	copyOfPending := c.Pending()
	copyOfPending[0] = '!'
	if string(c.Pending()) != partial {
		t.Fatal("pending data aliases")
	}
	if got, err := c.Feed([]byte{'\n'}); err != nil || len(got) != 1 || c.End() != nil {
		t.Fatal("fragment completion failed", err)
	}
	encoded, err := testCodec(t, 100).EncodeInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testCodec(t, len(encoded)-1).EncodeInitialized(); !errors.Is(err, ErrProtocolLimit) {
		t.Fatal("encoded LF escaped limit", err)
	}
}

func TestCodecOutgoingRefusesInvalidUTF8IDsShapesAndUnboundedSchema(t *testing.T) {
	c := testCodec(t, 512)
	if _, err := StringRequestID("bad\xff"); err == nil {
		t.Fatal("invalid ID")
	}
	for _, client := range []ClientInfo{{Name: "", Version: "v"}, {Name: "n", Version: "\xff"}} {
		if _, err := c.EncodeInitialize(IntegerRequestID(1), client); err == nil {
			t.Fatal("invalid client", client)
		}
	}
	if _, err := c.EncodeInitialize(RequestID{}, ClientInfo{Name: "n", Version: "v"}); err == nil {
		t.Fatal("zero request ID")
	}
	if _, err := c.EncodeTurnInterrupt(IntegerRequestID(1), "", "t"); err == nil {
		t.Fatal("empty thread")
	}
	for _, params := range []TurnStartParams{
		{Thread: "thread", Text: "\xff"}, {Thread: "\xff", Text: "text"}, {Thread: "thread", Effort: "\xff"},
		{Thread: "thread", OutputSchema: json.RawMessage(`{"a":1,"a":2}`)}, {Thread: "thread", OutputSchema: json.RawMessage(`[]`)},
		{Thread: "thread", OutputSchema: json.RawMessage(strings.Repeat(" ", 513))}, {Thread: "thread", Text: strings.Repeat("\"", 300)},
	} {
		if _, err := c.EncodeTurnStart(IntegerRequestID(1), params); err == nil {
			t.Fatal("invalid outgoing turn")
		}
	}
	if _, err := c.EncodeThreadStart(IntegerRequestID(1), ThreadStartParams{Model: "m", Cwd: "/work", ApprovalPolicy: "automatic", Sandbox: "danger-full-access"}); !errors.Is(err, ErrProtocolUnsupported) {
		t.Fatal("unsupported policy accepted", err)
	}
}

func TestCodecUnicodeEscapesCannotAliasWireIDs(t *testing.T) {
	for _, escaped := range []string{`\ud800`, `\udfff`, `\ud800\ud800`, `\udfff\ud800`, `\ud800x`, `\ud800\\udc00`} {
		for _, line := range []string{
			`{"id":"` + escaped + `","result":` + codecInitialization + `}`,
			`{"method":"turn/completed","params":{"threadId":"` + escaped + `","turn":{"id":"t","items":[],"status":"completed"}}}`,
			`{"method":"turn/completed","params":{"threadId":"thread","turn":{"id":"` + escaped + `","items":[],"status":"completed"}}}`,
		} {
			messages, err := testCodec(t, 4096).Feed([]byte(line + "\n"))
			if !errors.Is(err, ErrProtocol) || len(messages) != 1 || messages[0].Kind() != RejectedMessage || messages[0].Turn() != (TurnObservation{}) || string(messages[0].Raw()) != line+"\n" {
				t.Fatal("lossy escaped identity accepted", line, err)
			}
			if _, err := messages[0].ReadResponse(stringID(t, "�"), Initialize); err == nil {
				t.Fatal("lone surrogate matched real replacement character")
			}
		}
	}
	for _, tc := range []struct{ wire, value string }{{`�`, "�"}, {`\ud83d\ude00`, "😀"}, {`\\ud800`, `\ud800`}} {
		m := codecMessage(t, `{"id":"`+tc.wire+`","result":`+codecInitialization+`}`)
		if _, err := m.ReadResponse(stringID(t, tc.value), Initialize); err != nil {
			t.Fatal("valid identity rejected", tc.wire, err)
		}
	}
	// A multi-byte UTF-8 code point may span any input fragment boundary.
	c := testCodec(t, 4096)
	line := []byte("{\"method\":\"future/😀\"}\n")
	var messages []Message
	for i := range line {
		got, err := c.Feed(line[i : i+1])
		if err != nil {
			t.Fatal(err)
		}
		messages = append(messages, got...)
	}
	if len(messages) != 1 || messages[0].Method() != "future/😀" || !bytes.Equal(messages[0].Raw(), line) {
		t.Fatal("split UTF-8 code point lost")
	}
}
