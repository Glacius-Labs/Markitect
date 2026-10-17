package codexappserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxProtocolMessage = 1 << 20

var (
	ErrProtocol            = errors.New("invalid Codex protocol message")
	ErrProtocolLimit       = errors.New("Codex protocol message exceeds its byte or depth limit")
	ErrProtocolIncomplete  = errors.New("incomplete Codex NDJSON message")
	ErrProtocolUnsupported = errors.New("unsupported Codex protocol variant")
)

// RequestID preserves the protocol's distinct string and signed integer forms.
// Its zero value is invalid. It is neither a Host identity nor a provider handle.
type RequestID struct {
	value   string
	integer bool
	valid   bool
}

func StringRequestID(value string) (RequestID, error) {
	if !utf8.ValidString(value) {
		return RequestID{}, ErrProtocol
	}
	return RequestID{value: value, valid: true}, nil
}
func IntegerRequestID(value int64) RequestID {
	return RequestID{value: strconv.FormatInt(value, 10), integer: true, valid: true}
}
func (id RequestID) MarshalJSON() ([]byte, error) {
	if !id.valid {
		return nil, ErrProtocol
	}
	if id.integer {
		return []byte(id.value), nil
	}
	return json.Marshal(id.value)
}

type ThreadID string
type TurnID string
type RequestMethod string

const (
	Initialize    RequestMethod = "initialize"
	ThreadStart   RequestMethod = "thread/start"
	TurnStart     RequestMethod = "turn/start"
	TurnInterrupt RequestMethod = "turn/interrupt"
)

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ThreadStartParams is a selected stable subset, with no arbitrary config,
// dynamic tools, native delegation, resume, credentials or Host authority.
// Both instruction strings are sent literally, including an explicit empty one.
type ThreadStartParams struct {
	Model                 string `json:"model"`
	Cwd                   string `json:"cwd"`
	ApprovalPolicy        string `json:"approvalPolicy"`
	Sandbox               string `json:"sandbox"`
	BaseInstructions      string `json:"baseInstructions"`
	DeveloperInstructions string `json:"developerInstructions"`
}

type TurnStartParams struct {
	Thread       ThreadID
	Text         string
	Effort       string
	OutputSchema json.RawMessage
}

// Codec implements only bounded in-memory NDJSON framing and selected wire
// shapes from rust-v0.162.0. It does not correlate requests, run a session, redact
// secrets or authorize effects. A caller must serialize use of a Codec.
type Codec struct {
	max     int
	pending []byte
	failure error
}

// NewCodec bounds each entire line, including its LF, to at most 1 MiB.
func NewCodec(maxMessageBytes int) (*Codec, error) {
	if maxMessageBytes <= 0 || maxMessageBytes > maxProtocolMessage {
		return nil, ErrProtocolLimit
	}
	return &Codec{max: maxMessageBytes}, nil
}

func (c *Codec) EncodeInitialize(id RequestID, client ClientInfo) ([]byte, error) {
	if client.Name == "" || client.Version == "" {
		return nil, ErrProtocol
	}
	if err := c.checkText(client.Name, client.Version); err != nil {
		return nil, err
	}
	return c.request(id, Initialize, struct {
		ClientInfo ClientInfo `json:"clientInfo"`
	}{client})
}
func (c *Codec) EncodeInitialized() ([]byte, error) {
	return c.encode(struct {
		Method string `json:"method"`
	}{"initialized"})
}
func (c *Codec) EncodeThreadStart(id RequestID, p ThreadStartParams) ([]byte, error) {
	if p.Model == "" || p.Cwd == "" || !oneOf(p.ApprovalPolicy, "untrusted", "on-request", "never") || !oneOf(p.Sandbox, "read-only", "workspace-write", "danger-full-access") {
		return nil, ErrProtocolUnsupported
	}
	if err := c.checkText(p.Model, p.Cwd, p.ApprovalPolicy, p.Sandbox, p.BaseInstructions, p.DeveloperInstructions); err != nil {
		return nil, err
	}
	return c.request(id, ThreadStart, p)
}
func (c *Codec) EncodeTurnStart(id RequestID, p TurnStartParams) ([]byte, error) {
	if p.Thread == "" {
		return nil, ErrProtocol
	}
	if err := c.checkText(string(p.Thread), p.Text, p.Effort); err != nil {
		return nil, err
	}
	if len(p.OutputSchema) > c.max {
		return nil, ErrProtocolLimit
	}
	if len(p.OutputSchema) != 0 {
		if _, err := protocolObject(p.OutputSchema); err != nil {
			return nil, err
		}
	}
	return c.request(id, TurnStart, struct {
		Thread ThreadID `json:"threadId"`
		Input  []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"input"`
		Effort       string          `json:"effort,omitempty"`
		OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
	}{p.Thread, []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{{"text", p.Text}}, p.Effort, p.OutputSchema})
}
func (c *Codec) EncodeTurnInterrupt(id RequestID, thread ThreadID, turn TurnID) ([]byte, error) {
	if thread == "" || turn == "" {
		return nil, ErrProtocol
	}
	if err := c.checkText(string(thread), string(turn)); err != nil {
		return nil, err
	}
	return c.request(id, TurnInterrupt, struct {
		Thread ThreadID `json:"threadId"`
		Turn   TurnID   `json:"turnId"`
	}{thread, turn})
}
func (c *Codec) request(id RequestID, method RequestMethod, params any) ([]byte, error) {
	if err := c.checkText(id.value); err != nil {
		return nil, err
	}
	return c.encode(struct {
		ID     RequestID     `json:"id"`
		Method RequestMethod `json:"method"`
		Params any           `json:"params"`
	}{id, method, params})
}
func (c *Codec) encode(v any) ([]byte, error) {
	if c == nil || c.max <= 0 {
		return nil, ErrProtocol
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	if len(b)+1 > c.max {
		return nil, ErrProtocolLimit
	}
	if err := protocolJSON(b); err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Check before encoding/json can silently replace invalid UTF-8 strings or
// allocate an unbounded escaped representation of caller-supplied text.
func (c *Codec) checkText(values ...string) error {
	if c == nil || c.max <= 0 {
		return ErrProtocol
	}
	remaining := c.max
	for _, value := range values {
		if !utf8.ValidString(value) {
			return ErrProtocol
		}
		if len(value) > remaining {
			return ErrProtocolLimit
		}
		remaining -= len(value)
	}
	return nil
}

type MessageKind string

const (
	ResponseMessage      MessageKind = "response"
	ErrorMessage         MessageKind = "error"
	ThreadStartedMessage MessageKind = "thread/started"
	TurnStartedMessage   MessageKind = "turn/started"
	TurnCompletedMessage MessageKind = "turn/completed"
	UnknownNotification  MessageKind = "unknown-notification"
	UnsupportedRequest   MessageKind = "unsupported-request"
	RejectedMessage      MessageKind = "rejected"
)

type TurnStatus string

const (
	TurnInProgress  TurnStatus = "inProgress"
	TurnCompleted   TurnStatus = "completed"
	TurnInterrupted TurnStatus = "interrupted"
	TurnFailed      TurnStatus = "failed"
)

type TurnObservation struct {
	Thread ThreadID
	Turn   TurnID
	Status TurnStatus
}
type InitializeResponse struct{ CodexHome, PlatformFamily, PlatformOS, UserAgent string }
type Response struct {
	Method     RequestMethod
	Initialize InitializeResponse
	Thread     ThreadID
	Turn       TurnObservation
}

// Message retains copied bytes at the provider NDJSON boundary, including LF
// (and CR if present). Digest is NOT an engine-frame or redacted-journal hash.
// Truncated is true only for the bounded prefix of an oversized rejected line.
type Message struct {
	raw       []byte
	kind      MessageKind
	method    string
	id        RequestID
	body      json.RawMessage
	thread    ThreadID
	turn      TurnObservation
	truncated bool
}

func (m Message) Raw() []byte           { return bytes.Clone(m.raw) }
func (m Message) Digest() [32]byte      { return sha256.Sum256(m.raw) }
func (m Message) Kind() MessageKind     { return m.kind }
func (m Message) Method() string        { return m.method }
func (m Message) ID() (RequestID, bool) { return m.id, m.id.valid }
func (m Message) Thread() ThreadID      { return m.thread }
func (m Message) Turn() TurnObservation { return m.turn }
func (m Message) Truncated() bool       { return m.truncated }

// Feed accepts arbitrary fragments and returns every complete observation before
// a fatal error, plus the rejected observation. On a malformed or oversized line
// it latches the failure; the caller must stop consuming that connection. Later
// bytes in the supplied fragment are not interpreted. Unknown notifications and
// server requests remain explicit raw observations, never successful responses.
func (c *Codec) Feed(fragment []byte) ([]Message, error) {
	if c == nil || c.max <= 0 {
		return nil, ErrProtocol
	}
	if c.failure != nil {
		return nil, c.failure
	}
	var messages []Message
	for len(fragment) > 0 {
		n := bytes.IndexByte(fragment, '\n')
		complete := n >= 0
		if complete {
			n++
		} else {
			n = len(fragment)
		}
		if n > c.max-len(c.pending) {
			c.pending = append(c.pending, fragment[:c.max-len(c.pending)]...)
			c.failure = ErrProtocolLimit
			return append(messages, Message{raw: bytes.Clone(c.pending), kind: RejectedMessage, truncated: true}), c.failure
		}
		c.pending = append(c.pending, fragment[:n]...)
		fragment = fragment[n:]
		if !complete {
			break
		}
		m, err := decodeProtocolLine(c.pending)
		c.pending = nil
		messages = append(messages, m)
		if err != nil {
			c.failure = err
			return messages, err
		}
	}
	return messages, nil
}

// End requires a clean line boundary. Pending returns a copy of an unfinished
// line for the caller's observation; End never turns it into a complete message.
func (c *Codec) End() error {
	if c == nil || c.max <= 0 {
		return ErrProtocol
	}
	if c.failure != nil {
		return c.failure
	}
	if len(c.pending) != 0 {
		return ErrProtocolIncomplete
	}
	return nil
}
func (c *Codec) Pending() []byte {
	if c == nil {
		return nil
	}
	return bytes.Clone(c.pending)
}

func decodeProtocolLine(raw []byte) (Message, error) {
	m := Message{raw: bytes.Clone(raw), kind: RejectedMessage}
	o, err := protocolObject(raw)
	if err != nil {
		return m, err
	}
	_, hasID := o["id"]
	_, hasMethod := o["method"]
	_, hasResult := o["result"]
	_, hasError := o["error"]
	if hasID {
		m.id, err = parseRequestID(o["id"])
		if err != nil {
			return m, err
		}
	}
	if hasMethod {
		if hasResult || hasError {
			return m, ErrProtocol
		}
		m.method, err = protocolString(o["method"])
		if err != nil || m.method == "" {
			return m, ErrProtocol
		}
		m.body = bytes.Clone(o["params"])
		if hasID {
			m.kind = UnsupportedRequest
			return m, nil
		}
		m.kind = UnknownNotification
		switch m.method {
		case "thread/started":
			p, err := protocolObject(m.body)
			if err == nil {
				m.thread, err = parseThread(p["thread"])
			}
			if err != nil {
				m.kind = RejectedMessage
				return m, err
			}
			m.kind = ThreadStartedMessage
		case "turn/started", "turn/completed":
			m.turn, err = parseTurnNotification(m.body)
			if err != nil {
				return rejected(m), err
			}
			if m.method == "turn/started" {
				if m.turn.Status != TurnInProgress {
					return rejected(m), ErrProtocol
				}
				m.kind = TurnStartedMessage
			} else {
				if m.turn.Status == TurnInProgress {
					return rejected(m), ErrProtocol
				}
				m.kind = TurnCompletedMessage
			}
		}
		return m, nil
	}
	if !hasID || hasResult == hasError {
		return m, ErrProtocol
	}
	if _, ok := o["params"]; ok {
		return m, ErrProtocol
	}
	if hasError {
		if _, _, err := parseRPCError(o["error"]); err != nil {
			return m, err
		}
		m.kind, m.body = ErrorMessage, bytes.Clone(o["error"])
	} else {
		m.kind, m.body = ResponseMessage, bytes.Clone(o["result"])
	}
	return m, nil
}
func rejected(m Message) Message {
	m.kind = RejectedMessage
	m.thread = ""
	m.turn = TurnObservation{}
	return m
}

// ReadResponse requires the expected method and exact typed request ID. It
// validates the consumed projection, not every optional upstream schema field.
// Unconsumed fields and items remain in Raw; this grants no session readiness,
// tool authority, successful task result or provider capability.
func (m Message) ReadResponse(expected RequestID, method RequestMethod) (Response, error) {
	if !expected.valid || m.id != expected {
		return Response{}, ErrProtocol
	}
	if m.kind == ErrorMessage {
		return Response{}, ErrProtocol
	}
	if m.kind != ResponseMessage {
		return Response{}, ErrProtocol
	}
	o, err := protocolObject(m.body)
	if err != nil {
		return Response{}, err
	}
	r := Response{Method: method}
	switch method {
	case Initialize:
		fields := []*string{&r.Initialize.CodexHome, &r.Initialize.PlatformFamily, &r.Initialize.PlatformOS, &r.Initialize.UserAgent}
		for i, name := range []string{"codexHome", "platformFamily", "platformOs", "userAgent"} {
			*fields[i], err = protocolString(o[name])
			if err != nil {
				return Response{}, err
			}
		}
	case ThreadStart:
		r.Thread, err = parseThread(o["thread"])
		if err == nil {
			err = requireFields(o, "approvalPolicy", "approvalsReviewer", "cwd", "model", "modelProvider", "sandbox")
		}
	case TurnStart:
		r.Turn, err = parseTurn(o["turn"])
	case TurnInterrupt:
		if len(o) != 0 {
			err = ErrProtocolUnsupported
		}
	default:
		err = ErrProtocolUnsupported
	}
	if err != nil {
		return Response{}, err
	}
	return r, nil
}

// RPCError returns the provider error's typed code and text without losing Raw
// (which also retains optional error data). It is never a successful response.
func (m Message) RPCError() (int64, string, error) {
	if m.kind != ErrorMessage {
		return 0, "", ErrProtocol
	}
	return parseRPCError(m.body)
}
func parseRPCError(data []byte) (int64, string, error) {
	o, err := protocolObject(data)
	if err != nil {
		return 0, "", err
	}
	code, err := strconv.ParseInt(string(o["code"]), 10, 64)
	if err != nil {
		return 0, "", ErrProtocol
	}
	message, err := protocolString(o["message"])
	return code, message, err
}
func parseRequestID(data []byte) (RequestID, error) {
	if len(data) != 0 && data[0] == '"' {
		s, err := protocolString(data)
		if err != nil {
			return RequestID{}, err
		}
		return StringRequestID(s)
	}
	n, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return RequestID{}, ErrProtocol
	}
	return IntegerRequestID(n), nil
}
func parseThread(data []byte) (ThreadID, error) {
	o, err := protocolObject(data)
	if err != nil {
		return "", err
	}
	if err := requireFields(o, "cliVersion", "createdAt", "cwd", "ephemeral", "id", "modelProvider", "preview", "projectId", "sessionId", "source", "status", "turns", "updatedAt"); err != nil {
		return "", err
	}
	s, err := protocolString(o["id"])
	if err != nil || s == "" {
		return "", ErrProtocol
	}
	return ThreadID(s), nil
}
func parseTurn(data []byte) (TurnObservation, error) {
	o, err := protocolObject(data)
	if err != nil {
		return TurnObservation{}, err
	}
	id, err := protocolString(o["id"])
	if err != nil || id == "" {
		return TurnObservation{}, ErrProtocol
	}
	status, err := protocolString(o["status"])
	if err != nil {
		return TurnObservation{}, err
	}
	if !oneOf(status, "inProgress", "completed", "failed", "interrupted") {
		return TurnObservation{}, ErrProtocolUnsupported
	}
	if len(o["items"]) == 0 || o["items"][0] != '[' {
		return TurnObservation{}, ErrProtocol
	}
	return TurnObservation{Turn: TurnID(id), Status: TurnStatus(status)}, nil
}
func parseTurnNotification(data []byte) (TurnObservation, error) {
	o, err := protocolObject(data)
	if err != nil {
		return TurnObservation{}, err
	}
	t, err := parseTurn(o["turn"])
	if err != nil {
		return TurnObservation{}, err
	}
	thread, err := protocolString(o["threadId"])
	if err != nil || thread == "" {
		return TurnObservation{}, ErrProtocol
	}
	t.Thread = ThreadID(thread)
	return t, nil
}
func protocolString(data []byte) (string, error) {
	if len(data) == 0 || data[0] != '"' {
		return "", ErrProtocol
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", ErrProtocol
	}
	return s, nil
}
func requireFields(o map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		if _, ok := o[name]; !ok {
			return ErrProtocol
		}
	}
	return nil
}
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

// protocolJSON rejects duplicate keys at every nesting level, invalid UTF-8,
// excessive depth and trailing values. Size is bounded separately by the codec.
func protocolJSON(data []byte) error {
	if !utf8.Valid(data) || !pairedUnicodeEscapes(data) {
		return ErrProtocol
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := protocolValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrProtocol
	}
	return nil
}

// encoding/json replaces unpaired UTF-16 surrogate escapes with U+FFFD. Reject
// them before decoding so distinct wire IDs cannot collapse to one typed ID.
// Escaped backslashes are skipped as a pair: "\\\\ud800" is literal text.
func pairedUnicodeEscapes(data []byte) bool {
	for i := 0; i < len(data); {
		if data[i] != '\\' {
			i++
			continue
		}
		if i+1 >= len(data) {
			return false
		}
		if data[i+1] != 'u' {
			i += 2
			continue
		}
		if i+6 > len(data) {
			return false
		}
		value, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
		if err != nil || value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		i += 6
		if value < 0xd800 || value > 0xdbff {
			continue
		}
		if i+6 > len(data) || data[i] != '\\' || data[i+1] != 'u' {
			return false
		}
		low, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
func protocolValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return ErrProtocolLimit
	}
	token, err := d.Token()
	if err != nil {
		return ErrProtocol
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return ErrProtocol
			}
			key, ok := token.(string)
			if !ok || seen[key] {
				return ErrProtocol
			}
			seen[key] = true
			if err := protocolValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := protocolValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrProtocol
	}
	end, err := d.Token()
	if err != nil || delim == '{' && end != json.Delim('}') || delim == '[' && end != json.Delim(']') {
		return ErrProtocol
	}
	return nil
}
func protocolObject(data []byte) (map[string]json.RawMessage, error) {
	if err := protocolJSON(data); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		return nil, ErrProtocol
	}
	var o map[string]json.RawMessage
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, ErrProtocol
	}
	return o, nil
}
