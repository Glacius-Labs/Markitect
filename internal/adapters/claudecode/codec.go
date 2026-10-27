package claudecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxProtocolMessage bounds one stream-json line, including its LF. One line
// can carry a whole written file or a Run tool result escaped once more, so
// the bound is larger than the 1 MiB agentd frame.
const maxProtocolMessage = 8 << 20

var (
	ErrProtocol            = errors.New("invalid Claude Code stream message")
	ErrProtocolLimit       = errors.New("Claude Code stream message exceeds its byte or depth limit")
	ErrProtocolIncomplete  = errors.New("incomplete Claude Code stream message")
	ErrProtocolUnsupported = errors.New("unsupported Claude Code stream variant")
)

// Codec implements bounded in-memory NDJSON framing of the documented
// stream-json interface of the pinned Claude Code CLI (`--print
// --input-format stream-json --output-format stream-json --verbose`): one
// encoded user message in, and the selected output shapes below. It does not
// run a session, redact secrets or authorize effects. A caller must serialize
// use of a Codec; one Codec reads one provider process.
type Codec struct {
	max     int
	pending []byte
	failure error
}

// NewCodec bounds each entire line, including its LF, to at most 8 MiB.
func NewCodec(maxMessageBytes int) (*Codec, error) {
	if maxMessageBytes <= 0 || maxMessageBytes > maxProtocolMessage {
		return nil, ErrProtocolLimit
	}
	return &Codec{max: maxMessageBytes}, nil
}

// EncodeUser encodes the one user message of a response as the documented
// SDK user message: a text content block with no parent tool use.
func (c *Codec) EncodeUser(text string) ([]byte, error) {
	if text == "" {
		return nil, ErrProtocol
	}
	if err := c.checkText(text); err != nil {
		return nil, err
	}
	type block struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	return c.encode(struct {
		Type    string `json:"type"`
		Message struct {
			Role    string  `json:"role"`
			Content []block `json:"content"`
		} `json:"message"`
		Parent *string `json:"parent_tool_use_id"`
	}{Type: "user", Message: struct {
		Role    string  `json:"role"`
		Content []block `json:"content"`
	}{"user", []block{{"text", text}}}})
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

// MessageKind classifies one output line. Unknown lines stay explicit raw
// observations: they are recorded and never treated as success.
type MessageKind string

const (
	SystemInitMessage     MessageKind = "system/init"
	APIRetryMessage       MessageKind = "system/api_retry"
	SystemMessage         MessageKind = "system"
	AssistantMessage      MessageKind = "assistant"
	UserMessage           MessageKind = "user"
	ResultMessage         MessageKind = "result"
	ControlRequestMessage MessageKind = "control_request"
	UnknownMessage        MessageKind = "unknown"
	RejectedMessage       MessageKind = "rejected"
)

// Init is the consumed projection of the system/init message.
type Init struct {
	Session, Model, Version, APIKeySource, PermissionMode string
	Tools                                                 []string
	MCPServers                                            []MCPServer
}

type MCPServer struct{ Name, Status string }

// APIRetry is a system/api_retry message: Claude Code retries a failed API
// request itself and names the error category.
type APIRetry struct {
	Attempt int
	Error   string
}

// Block is the consumed projection of one content block. ID names a tool use;
// ToolUseID names the tool use a result answers.
type Block struct {
	Type, ID, Name, ToolUseID, Text string
	IsError                         bool
}

// Result is the result message that ends a turn. IsError is independent of
// Subtype: a turn that ended on an API error can report subtype success.
type Result struct {
	Subtype string
	IsError bool
	Text    string
	Errors  []string
	Session string
}

// Message retains copied bytes at the provider NDJSON boundary, including LF.
// Truncated is true only for the bounded prefix of an oversized rejected line.
type Message struct {
	raw       []byte
	kind      MessageKind
	subtype   string
	session   string
	parent    string
	errorKind string
	init      Init
	retry     APIRetry
	blocks    []Block
	result    Result
	truncated bool
}

func (m Message) Raw() []byte       { return bytes.Clone(m.raw) }
func (m Message) Kind() MessageKind { return m.kind }
func (m Message) Subtype() string   { return m.subtype }
func (m Message) Session() string   { return m.session }
func (m Message) Parent() string    { return m.parent }
func (m Message) ErrorKind() string { return m.errorKind }
func (m Message) Init() Init        { return m.init }
func (m Message) Retry() APIRetry   { return m.retry }
func (m Message) Blocks() []Block   { return append([]Block(nil), m.blocks...) }
func (m Message) Result() Result    { return m.result }
func (m Message) Truncated() bool   { return m.truncated }

// Feed accepts arbitrary fragments and returns every complete observation
// before a fatal error, plus the rejected observation. On a malformed or
// oversized line it latches the failure; the caller must stop consuming that
// process. Later bytes in the supplied fragment are not interpreted.
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
		m, err := decodeLine(c.pending)
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
// line for the caller's observation; End never turns it into a message.
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

// decodeLine validates the whole line, including fields it does not consume,
// and projects the selected shapes. An empty line between messages is refused.
func decodeLine(raw []byte) (Message, error) {
	m := Message{raw: bytes.Clone(raw), kind: RejectedMessage}
	o, err := protocolObject(raw)
	if err != nil {
		return m, err
	}
	typ, err := protocolString(o["type"])
	if err != nil || typ == "" {
		return m, ErrProtocol
	}
	if value, ok := o["session_id"]; ok {
		if m.session, err = protocolString(value); err != nil {
			return m, err
		}
	}
	switch typ {
	case "system":
		m.subtype, err = protocolString(o["subtype"])
		if err != nil || m.subtype == "" {
			return m, ErrProtocol
		}
		switch m.subtype {
		case "init":
			m.init, err = parseInit(o)
			m.kind = SystemInitMessage
		case "api_retry":
			m.retry, err = parseRetry(o)
			m.kind = APIRetryMessage
		default:
			m.kind = SystemMessage
		}
	case "assistant", "user":
		if m.parent, err = parseParent(o); err == nil {
			m.blocks, err = parseContent(o["message"], typ)
		}
		if err == nil && typ == "assistant" {
			if value, ok := o["error"]; ok && string(value) != "null" {
				m.errorKind, err = protocolString(value)
			}
		}
		m.kind = MessageKind(typ)
	case "result":
		m.result, err = parseResult(o)
		m.subtype = m.result.Subtype
		m.kind = ResultMessage
	case "control_request":
		m.kind = ControlRequestMessage
	default:
		m.kind = UnknownMessage
	}
	if err != nil {
		m.kind = RejectedMessage
		return m, err
	}
	return m, nil
}

func parseInit(o map[string]json.RawMessage) (Init, error) {
	var i Init
	var err error
	if i.Session, err = protocolString(o["session_id"]); err != nil || i.Session == "" {
		return Init{}, ErrProtocol
	}
	if i.Model, err = protocolString(o["model"]); err != nil {
		return Init{}, err
	}
	for name, target := range map[string]*string{"claude_code_version": &i.Version, "apiKeySource": &i.APIKeySource, "permissionMode": &i.PermissionMode} {
		if value, ok := o[name]; ok {
			if *target, err = protocolString(value); err != nil {
				return Init{}, err
			}
		}
	}
	if err = json.Unmarshal(o["tools"], &i.Tools); err != nil || !array(o["tools"]) {
		return Init{}, ErrProtocol
	}
	var servers []struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err = json.Unmarshal(o["mcp_servers"], &servers); err != nil || !array(o["mcp_servers"]) {
		return Init{}, ErrProtocol
	}
	for _, s := range servers {
		if s.Name == nil || s.Status == nil {
			return Init{}, ErrProtocol
		}
		i.MCPServers = append(i.MCPServers, MCPServer{Name: *s.Name, Status: *s.Status})
	}
	return i, nil
}

func parseRetry(o map[string]json.RawMessage) (APIRetry, error) {
	var r APIRetry
	if err := json.Unmarshal(o["attempt"], &r.Attempt); err != nil || r.Attempt < 1 {
		return APIRetry{}, ErrProtocol
	}
	if value, ok := o["error"]; ok {
		var err error
		if r.Error, err = protocolString(value); err != nil {
			return APIRetry{}, err
		}
	}
	return r, nil
}

// parseParent requires the parent_tool_use_id key: null for the main
// conversation, a tool use ID for a subagent.
func parseParent(o map[string]json.RawMessage) (string, error) {
	value, ok := o["parent_tool_use_id"]
	if !ok {
		return "", ErrProtocol
	}
	if string(bytes.TrimSpace(value)) == "null" {
		return "", nil
	}
	parent, err := protocolString(value)
	if err != nil || parent == "" {
		return "", ErrProtocol
	}
	return parent, nil
}

// parseContent reads message.content: an array of typed blocks, or for a user
// message a plain string.
func parseContent(data []byte, role string) ([]Block, error) {
	message, err := protocolObject(data)
	if err != nil {
		return nil, err
	}
	if got, err := protocolString(message["role"]); err != nil || got != role {
		return nil, ErrProtocol
	}
	content, ok := message["content"]
	if !ok {
		return nil, ErrProtocol
	}
	if !array(content) {
		if text, err := protocolString(content); err == nil && role == "user" {
			return []Block{{Type: "text", Text: text}}, nil
		}
		return nil, ErrProtocol
	}
	var items []json.RawMessage
	if err := json.Unmarshal(content, &items); err != nil {
		return nil, ErrProtocol
	}
	blocks := make([]Block, 0, len(items))
	for _, item := range items {
		fields, err := protocolObject(item)
		if err != nil {
			return nil, err
		}
		var b Block
		if b.Type, err = protocolString(fields["type"]); err != nil || b.Type == "" {
			return nil, ErrProtocol
		}
		for name, target := range map[string]*string{"id": &b.ID, "name": &b.Name, "tool_use_id": &b.ToolUseID} {
			if value, ok := fields[name]; ok {
				if *target, err = protocolString(value); err != nil {
					return nil, err
				}
			}
		}
		if b.Type == "text" {
			if b.Text, err = protocolString(fields["text"]); err != nil {
				return nil, err
			}
		}
		if value, ok := fields["is_error"]; ok && string(value) != "null" {
			if err = json.Unmarshal(value, &b.IsError); err != nil {
				return nil, ErrProtocol
			}
		}
		if b.Type == "tool_use" && (b.ID == "" || b.Name == "") || b.Type == "tool_result" && b.ToolUseID == "" {
			return nil, ErrProtocol
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

func parseResult(o map[string]json.RawMessage) (Result, error) {
	var r Result
	var err error
	if r.Subtype, err = protocolString(o["subtype"]); err != nil || r.Subtype == "" {
		return Result{}, ErrProtocol
	}
	if r.Session, err = protocolString(o["session_id"]); err != nil || r.Session == "" {
		return Result{}, ErrProtocol
	}
	switch string(bytes.TrimSpace(o["is_error"])) {
	case "true":
		r.IsError = true
	case "false":
	default:
		return Result{}, ErrProtocol
	}
	if value, ok := o["result"]; ok {
		if r.Text, err = protocolString(value); err != nil {
			return Result{}, err
		}
	} else if r.Subtype == "success" {
		return Result{}, ErrProtocol
	}
	if value, ok := o["errors"]; ok {
		if !array(value) || json.Unmarshal(value, &r.Errors) != nil {
			return Result{}, ErrProtocol
		}
	}
	return r, nil
}

func array(data []byte) bool {
	data = bytes.TrimSpace(data)
	return len(data) > 0 && data[0] == '['
}

func protocolString(data []byte) (string, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '"' {
		return "", ErrProtocol
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", ErrProtocol
	}
	return s, nil
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
// them before decoding so distinct wire values cannot collapse into one.
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
