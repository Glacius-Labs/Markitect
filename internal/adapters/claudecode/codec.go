// Package claudecode contains the selected Claude Code 2.1.301 wire codec
// independently of the unqualified replacement native Session consumer.
package claudecode

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
)

const maxLineBytes = 1 << 20

var (
	ErrProtocol    = errors.New("invalid Claude stream protocol message")
	ErrLimit       = errors.New("Claude stream protocol byte or depth limit exceeded")
	ErrIncomplete  = errors.New("incomplete Claude NDJSON frame")
	ErrUnsupported = errors.New("unsupported Claude stream protocol variant")
)

// These identities have different meanings even when their strings coincide.
// None is an Orchestrator step identity or evidence of an exactly bound provider step.
type ControlRequestID string
type RequestedSessionID string
type ObservedSessionID string
type UserMessageUUID string
type MessageUUID string
type ResultUUID string

type UserInput struct {
	Text           string
	UUID           UserMessageUUID
	Session        RequestedSessionID
	ClientComposed bool
}

// Codec is single-consumer, in-memory NDJSON framing. It holds no request map,
// session, clock, I/O, credentials, dispatch authority or accumulated history.
type Codec struct {
	max     int
	pending []byte
	failure error
}

func NewCodec(limit int) (*Codec, error) {
	if limit < 1 || limit > maxLineBytes {
		return nil, ErrLimit
	}
	return &Codec{max: limit}, nil
}

// EncodeInitialize uses only a supplied request identity and a minimal request.
func (c *Codec) EncodeInitialize(id ControlRequestID) ([]byte, error) {
	if id == "" {
		return nil, ErrProtocol
	}
	if err := c.text(string(id)); err != nil {
		return nil, err
	}
	return c.encode(struct {
		Type    string           `json:"type"`
		ID      ControlRequestID `json:"request_id"`
		Request struct {
			Subtype string `json:"subtype"`
		} `json:"request"`
	}{"control_request", id, struct {
		Subtype string `json:"subtype"`
	}{"initialize"}})
}

func (c *Codec) EncodeUser(p UserInput) ([]byte, error) {
	if err := c.text(p.Text, string(p.UUID), string(p.Session)); err != nil {
		return nil, err
	}
	return c.encode(struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Parent         *string            `json:"parent_tool_use_id"`
		UUID           UserMessageUUID    `json:"uuid,omitempty"`
		Session        RequestedSessionID `json:"session_id,omitempty"`
		ClientComposed bool               `json:"client_composed,omitempty"`
	}{"user", struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{"user", p.Text}, nil, p.UUID, p.Session, p.ClientComposed})
}

// Validate before encoding/json can replace malformed UTF-8 or allocate an
// escaped representation of text larger than the configured frame limit.
func (c *Codec) text(values ...string) error {
	if c == nil || c.max < 1 {
		return ErrProtocol
	}
	remaining := c.max
	for _, value := range values {
		if !utf8.ValidString(value) {
			return ErrProtocol
		}
		if len(value) > remaining {
			return ErrLimit
		}
		remaining -= len(value)
	}
	return nil
}
func (c *Codec) encode(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, ErrProtocol
	}
	if len(b) >= c.max {
		return nil, ErrLimit
	}
	return append(b, '\n'), nil
}

type MessageKind string

const (
	ControlResponseMessage MessageKind = "control-response"
	SystemInitMessage      MessageKind = "system-init"
	ThinkingTokensMessage  MessageKind = "thinking-tokens"
	// CompactBoundaryMessage marks that the provider compacted the
	// conversation: what came before survives only in its summary.
	CompactBoundaryMessage MessageKind = "compact-boundary"
	UserMessage            MessageKind = "user"
	AssistantMessage       MessageKind = "assistant"
	StreamEventMessage     MessageKind = "stream-event"
	RateLimitEventMessage  MessageKind = "rate-limit-event"
	ResultMessage          MessageKind = "result"
	UnsupportedMessage     MessageKind = "unsupported-message"
	UnsupportedRequest     MessageKind = "unsupported-control-request"
	RejectedMessage        MessageKind = "rejected"
)

type ControlStatus string

const (
	ControlSuccess ControlStatus = "success"
	ControlError   ControlStatus = "error"
)

// InitializeResponse is a matching control observation, never RUN readiness or
// a session binding. Error is the provider's text when Status == ControlError.
type InitializeResponse struct {
	Request ControlRequestID
	Status  ControlStatus
	Error   string
	Version string
}
type InitObservation struct {
	Session ObservedSessionID
	UUID    MessageUUID
	Version string
}
type ResultSubtype string

const (
	ResultSuccess           ResultSubtype = "success"
	ResultExecutionError    ResultSubtype = "error_during_execution"
	ResultMaxTurns          ResultSubtype = "error_max_turns"
	ResultMaxBudget         ResultSubtype = "error_max_budget_usd"
	ResultStructuredRetries ResultSubtype = "error_max_structured_output_retries"
)

// ResultObservation is protocol data, not task acceptance or process termination.
// IsError is independent of Subtype. TerminalReason is uninterpreted status text.
type ResultObservation struct {
	Session          ObservedSessionID
	UUID             ResultUUID
	Subtype          ResultSubtype
	IsError          bool
	TerminalReason   string
	Text             string
	Errors           []string
	StructuredOutput json.RawMessage
}

// Message owns the exact provider frame including LF and optional CR. Truncated
// marks a rejected prefix: its digest identifies only that prefix, not a frame.
type Message struct {
	raw       []byte
	kind      MessageKind
	subtype   string
	control   ControlRequestID
	truncated bool
}

func (m Message) Raw() []byte                 { return bytes.Clone(m.raw) }
func (m Message) Digest() [32]byte            { return sha256.Sum256(m.raw) }
func (m Message) Kind() MessageKind           { return m.kind }
func (m Message) Subtype() string             { return m.subtype }
func (m Message) ControlID() ControlRequestID { return m.control }
func (m Message) Truncated() bool             { return m.truncated }

// Feed retains at most limit pending bytes, without cloning the entire fragment.
// It returns preceding complete observations and one rejected observation on
// failure. Failure latches; remaining bytes in this and later feeds are ignored.
func (c *Codec) Feed(fragment []byte) ([]Message, error) {
	if c == nil || c.max < 1 {
		return nil, ErrProtocol
	}
	if c.failure != nil {
		return nil, c.failure
	}
	var out []Message
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
			c.failure = ErrLimit
			return append(out, Message{raw: bytes.Clone(c.pending), kind: RejectedMessage, truncated: true}), c.failure
		}
		c.pending = append(c.pending, fragment[:n]...)
		fragment = fragment[n:]
		if !complete {
			break
		}
		m, err := decodeLine(c.pending)
		out = append(out, m)
		if err != nil {
			c.failure = err
			return out, err
		}
		c.pending = nil
	}
	return out, nil
}

// End never invents LF. An incomplete final frame also latches failure.
func (c *Codec) End() error {
	if c == nil || c.max < 1 {
		return ErrProtocol
	}
	if c.failure == nil && len(c.pending) != 0 {
		c.failure = ErrIncomplete
	}
	return c.failure
}
func (c *Codec) Pending() []byte {
	if c == nil {
		return nil
	}
	return bytes.Clone(c.pending)
}

func decodeLine(raw []byte) (Message, error) {
	m := Message{raw: bytes.Clone(raw), kind: RejectedMessage}
	if err := validJSON(raw); err != nil {
		return m, err
	}
	o, err := object(raw)
	if err != nil {
		return m, err
	}
	typ, err := nonempty(o["type"])
	if err != nil {
		return m, err
	}
	m.kind = UnsupportedMessage
	switch typ {
	case "control_request":
		id, e := nonempty(o["request_id"])
		p, pe := object(o["request"])
		if e != nil || pe != nil {
			return rejected(m, ErrProtocol)
		}
		m.subtype, err = nonempty(p["subtype"])
		m.control, m.kind = ControlRequestID(id), UnsupportedRequest
	case "control_response":
		p, e := object(o["response"])
		if e != nil {
			return rejected(m, e)
		}
		id, e := nonempty(p["request_id"])
		if e != nil {
			return rejected(m, e)
		}
		m.control = ControlRequestID(id)
		m.subtype, err = nonempty(p["subtype"])
		if err != nil {
			break
		}
		switch ControlStatus(m.subtype) {
		case ControlSuccess:
			// Other control methods may be acknowledged without a payload.
			if body, ok := p["response"]; ok {
				_, err = object(body)
			}
		case ControlError:
			_, err = stringValue(p["error"])
		default:
			return m, nil
		}
		m.kind = ControlResponseMessage
	case "system":
		m.subtype, err = nonempty(o["subtype"])
		if err == nil {
			switch m.subtype {
			case "init":
				_, err = readInit(o)
				m.kind = SystemInitMessage
			case "thinking_tokens":
				// Select only the observed six-key informational shape. The
				// pinned SDK declares numbers, with no integer/sign restriction.
				if len(o) != 6 {
					err = ErrProtocol
				} else {
					err = envelopeIDs(o, true)
					for _, key := range []string{"estimated_tokens", "estimated_tokens_delta"} {
						var value float64
						if bytes.Equal(bytes.TrimSpace(o[key]), []byte("null")) || json.Unmarshal(o[key], &value) != nil {
							err = ErrProtocol
						}
					}
				}
				m.kind = ThinkingTokensMessage
			case "compact_boundary":
				// Classified by its type alone: any shape of it is an observed
				// compaction, which errs toward delivering the context again.
				m.kind = CompactBoundaryMessage
			}
		}
	case "user", "assistant", "stream_event":
		if typ == "stream_event" {
			_, err = object(o["event"])
		} else {
			var p map[string]json.RawMessage
			p, err = object(o["message"])
			if err == nil {
				role, e := stringValue(p["role"])
				if e != nil || role != typ {
					err = ErrProtocol
				}
				if _, ok := p["content"]; !ok {
					err = ErrProtocol
				}
				if err == nil && !array(p["content"]) {
					if typ != "user" {
						err = ErrProtocol
					} else {
						_, err = stringValue(p["content"])
					}
				}
			}
		}
		if err == nil {
			err = envelopeIDs(o, typ != "user")
		}
		if err == nil {
			if parent, ok := o["parent_tool_use_id"]; ok {
				if !bytes.Equal(parent, []byte("null")) {
					_, err = nonempty(parent)
				}
			} else {
				err = ErrProtocol
			}
		}
		switch typ {
		case "user":
			m.kind = UserMessage
		case "assistant":
			m.kind = AssistantMessage
		case "stream_event":
			m.kind = StreamEventMessage
		}
	case "rate_limit_event":
		// SDK 0.3.301 declares this four-key informational envelope. Its
		// subscription payload remains opaque and supplies no step outcome.
		if len(o) != 4 {
			err = ErrProtocol
		} else if _, err = object(o["rate_limit_info"]); err == nil {
			err = envelopeIDs(o, true)
		}
		m.kind = RateLimitEventMessage
	case "result":
		m.subtype, err = nonempty(o["subtype"])
		if err == nil {
			if !knownResult(m.subtype) {
				return m, nil
			}
			_, err = readResult(o)
			m.kind = ResultMessage
		}
	}
	if err != nil {
		return rejected(m, err)
	}
	return m, nil
}
func rejected(m Message, err error) (Message, error) { m.kind = RejectedMessage; return m, err }

func (m Message) ReadInitializeResponse(expected ControlRequestID) (InitializeResponse, error) {
	if expected == "" || m.kind != ControlResponseMessage || expected != m.control {
		return InitializeResponse{}, ErrProtocol
	}
	o, _ := object(m.raw)
	p, _ := object(o["response"])
	r := InitializeResponse{Request: m.control, Status: ControlStatus(m.subtype)}
	if r.Status == ControlError {
		r.Error, _ = stringValue(p["error"])
		return r, nil
	}
	body, err := object(p["response"])
	if err != nil {
		return InitializeResponse{}, err
	}
	for _, key := range []string{"commands", "agents", "models"} {
		if !array(body[key]) {
			return InitializeResponse{}, ErrProtocol
		}
	}
	if _, err := stringValue(body["output_style"]); err != nil {
		return InitializeResponse{}, err
	}
	if _, err := stringsValue(body["available_output_styles"]); err != nil {
		return InitializeResponse{}, err
	}
	if _, err := object(body["account"]); err != nil {
		return InitializeResponse{}, err
	}
	if v, ok := body["claude_code_version"]; ok {
		r.Version, err = nonempty(v)
		if err != nil {
			return InitializeResponse{}, err
		}
	}
	return r, nil
}
func (m Message) Init() (InitObservation, error) {
	if m.kind != SystemInitMessage {
		return InitObservation{}, ErrUnsupported
	}
	o, _ := object(m.raw)
	return readInit(o)
}
func readInit(o map[string]json.RawMessage) (InitObservation, error) {
	session, e1 := nonempty(o["session_id"])
	uuid, e2 := nonempty(o["uuid"])
	version, e3 := nonempty(o["claude_code_version"])
	if e1 != nil || e2 != nil || e3 != nil {
		return InitObservation{}, ErrProtocol
	}
	return InitObservation{ObservedSessionID(session), MessageUUID(uuid), version}, nil
}
func (m Message) Result() (ResultObservation, error) {
	if m.kind != ResultMessage {
		return ResultObservation{}, ErrUnsupported
	}
	o, _ := object(m.raw)
	return readResult(o)
}
func readResult(o map[string]json.RawMessage) (ResultObservation, error) {
	r := ResultObservation{}
	session, e1 := nonempty(o["session_id"])
	uuid, e2 := nonempty(o["uuid"])
	subtype, e3 := nonempty(o["subtype"])
	if e1 != nil || e2 != nil || e3 != nil {
		return r, ErrProtocol
	}
	r.Session, r.UUID, r.Subtype = ObservedSessionID(session), ResultUUID(uuid), ResultSubtype(subtype)
	switch string(o["is_error"]) {
	case "true":
		r.IsError = true
	case "false":
	default:
		return ResultObservation{}, ErrProtocol
	}
	if reason, ok := o["terminal_reason"]; ok {
		var err error
		r.TerminalReason, err = stringValue(reason)
		if err != nil {
			return ResultObservation{}, err
		}
	}
	if text, ok := o["result"]; ok {
		var err error
		r.Text, err = stringValue(text)
		if err != nil {
			return ResultObservation{}, err
		}
	} else if r.Subtype == ResultSuccess {
		return ResultObservation{}, ErrProtocol
	}
	if data, ok := o["errors"]; ok {
		var err error
		r.Errors, err = stringsValue(data)
		if err != nil {
			return ResultObservation{}, err
		}
	} else if r.Subtype != ResultSuccess {
		return ResultObservation{}, ErrProtocol
	}
	r.StructuredOutput = bytes.Clone(o["structured_output"])
	return r, nil
}
func knownResult(s string) bool {
	switch ResultSubtype(s) {
	case ResultSuccess, ResultExecutionError, ResultMaxTurns, ResultMaxBudget, ResultStructuredRetries:
		return true
	}
	return false
}
func envelopeIDs(o map[string]json.RawMessage, required bool) error {
	for _, key := range []string{"uuid", "session_id"} {
		v, ok := o[key]
		if !ok && !required {
			continue
		}
		if _, err := nonempty(v); err != nil {
			return err
		}
	}
	return nil
}
func object(data []byte) (map[string]json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil, ErrProtocol
	}
	var o map[string]json.RawMessage
	if json.Unmarshal(data, &o) != nil {
		return nil, ErrProtocol
	}
	return o, nil
}
func stringValue(data []byte) (string, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '"' {
		return "", ErrProtocol
	}
	var value string
	if json.Unmarshal(data, &value) != nil {
		return "", ErrProtocol
	}
	return value, nil
}
func nonempty(data []byte) (string, error) {
	s, err := stringValue(data)
	if err != nil || s == "" {
		return "", ErrProtocol
	}
	return s, nil
}
func array(data []byte) bool { data = bytes.TrimSpace(data); return len(data) > 0 && data[0] == '[' }
func stringsValue(data []byte) ([]string, error) {
	if !array(data) {
		return nil, ErrProtocol
	}
	var values []json.RawMessage
	if json.Unmarshal(data, &values) != nil {
		return nil, ErrProtocol
	}
	out := make([]string, len(values))
	for i, value := range values {
		var err error
		out[i], err = stringValue(value)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Validate the entire JSON, including opaque fields, before any projection.
func validJSON(data []byte) error {
	if !utf8.Valid(data) || !validEscapes(data) {
		return ErrProtocol
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := jsonValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrProtocol
	}
	return nil
}
func jsonValue(d *json.Decoder, depth int) error {
	token, err := d.Token()
	if err != nil {
		return ErrProtocol
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if depth >= 64 {
		return ErrLimit
	}
	if delim != '{' && delim != '[' {
		return ErrProtocol
	}
	var keys map[string]bool
	if delim == '{' {
		keys = make(map[string]bool)
	}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return ErrProtocol
			}
			s, ok := key.(string)
			if !ok || keys[s] {
				return ErrProtocol
			}
			keys[s] = true
		}
		if err := jsonValue(d, depth+1); err != nil {
			return err
		}
	}
	end, err := d.Token()
	if err != nil || delim == '{' && end != json.Delim('}') || delim == '[' && end != json.Delim(']') {
		return ErrProtocol
	}
	return nil
}

// Reject UTF-16 escapes that encoding/json would replace with U+FFFD. Skip
// escaped backslashes so literal backslash-u text is not mistaken for an escape.
func validEscapes(data []byte) bool {
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
