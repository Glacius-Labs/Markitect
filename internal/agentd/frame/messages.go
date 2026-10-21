package frame

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ControlType names a control message.
type ControlType string

const (
	// TypeHello is agentd's first message: its frame-protocol version.
	TypeHello ControlType = "hello"
	// TypeStart asks agentd to start the provider CLI.
	TypeStart ControlType = "start"
	// TypeStarted reports the started provider process.
	TypeStarted ControlType = "started"
	// TypeStop asks agentd to stop the provider within a grace period.
	TypeStop ControlType = "stop"
	// TypeExited reports the provider's observed exit.
	TypeExited ControlType = "exited"
	// TypeFailure reports a request agentd could not carry out.
	TypeFailure ControlType = "failure"
	// TypeCensus asks for, and answers with, the processes in the container.
	TypeCensus ControlType = "census"
)

// ControlMessage is one control message. Exactly the field named by Type is
// set. Request numbers an Orchestrator request; agentd copies it into the reply
// (started, failure or census), and leaves it zero on messages it sends on
// its own, such as hello, exited or a failure that ends the session.
type ControlMessage struct {
	Type    ControlType `json:"type"`
	Request uint64      `json:"request,omitempty"`
	Hello   *Hello      `json:"hello,omitempty"`
	Start   *Start      `json:"start,omitempty"`
	Started *Started    `json:"started,omitempty"`
	Stop    *Stop       `json:"stop,omitempty"`
	Exited  *Exited     `json:"exited,omitempty"`
	Failure *Failure    `json:"failure,omitempty"`
	Census  *Census     `json:"census,omitempty"`
}

// Hello announces agentd. Version is informational; only Protocol must match.
type Hello struct {
	Protocol int    `json:"protocol"`
	Version  string `json:"version"`
}

// Start describes the provider process and the Orchestrator tools it may call.
// Env entries are KEY=VALUE pairs added to the provider's environment only;
// they may carry the runtime credential, so the Orchestrator records this message
// with values redacted.
type Start struct {
	Argv  []string   `json:"argv"`
	Dir   string     `json:"dir"`
	Env   []string   `json:"env,omitempty"`
	Tools []ToolSpec `json:"tools"`
}

// Started reports the provider's process ID inside the container.
type Started struct {
	PID int `json:"pid"`
}

// Stop asks for termination: a polite signal to the provider's process
// group, then a kill after GraceMillis.
type Stop struct {
	GraceMillis int64 `json:"graceMillis"`
}

// Exited reports how the provider ended. Signal is empty for an ordinary
// exit; Code is -1 when the process was killed by a signal.
type Exited struct {
	Code   int    `json:"code"`
	Signal string `json:"signal,omitempty"`
}

// Failure reports why agentd refused or could not complete a request.
type Failure struct {
	Message string `json:"message"`
}

// Census lists the processes in the container other than agentd and the
// container's init process. The Orchestrator sends an empty census as its request;
// agentd answers with the observed processes. A census is a snapshot: it
// supports a quiescence judgment but cannot prevent a later process start.
type Census struct {
	Processes []Process `json:"processes,omitempty"`
}

// Process is one observed process inside the container. State is the
// kernel's one-letter state; "Z" marks a process that ended and awaits its
// parent's wait.
type Process struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	PGID    int    `json:"pgid"`
	State   string `json:"state"`
	Command string `json:"command"`
	// Threads is the number of threads of the process when the census knows it.
	Threads int `json:"threads,omitempty"`
}

// ToolSpec is one Orchestrator tool as offered to the provider through MCP.
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolCall is a provider's call of an Orchestrator tool. ID is assigned by agentd and
// is unique per connection; Meta carries the provider's own call metadata.
type ToolCall struct {
	ID        uint64          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Meta      json.RawMessage `json:"meta,omitempty"`
}

// ToolReply answers one ToolCall. Result is an MCP tools/call result and is
// passed to the provider unchanged; Error refuses the call.
type ToolReply struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ToolError      `json:"error,omitempty"`
}

// ToolError is a JSON-RPC error returned to the provider for a refused call.
type ToolError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// EncodeControl validates and encodes a control message as a frame payload.
func EncodeControl(c ControlMessage) ([]byte, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

// DecodeControl decodes and validates a control frame payload.
func DecodeControl(payload []byte) (ControlMessage, error) {
	var c ControlMessage
	if err := decodeStrict(payload, &c); err != nil {
		return ControlMessage{}, fmt.Errorf("decode control message: %w", err)
	}
	if err := c.validate(); err != nil {
		return ControlMessage{}, err
	}
	return c, nil
}

// EncodeToolCall encodes a tool call frame payload.
func EncodeToolCall(c ToolCall) ([]byte, error) {
	if c.Name == "" || !json.Valid(c.Arguments) {
		return nil, errors.New("encode tool call: a name and JSON arguments are required")
	}
	return json.Marshal(c)
}

// DecodeToolCall decodes a tool call frame payload.
func DecodeToolCall(payload []byte) (ToolCall, error) {
	var c ToolCall
	if err := decodeStrict(payload, &c); err != nil {
		return ToolCall{}, fmt.Errorf("decode tool call: %w", err)
	}
	if c.Name == "" || len(c.Arguments) == 0 {
		return ToolCall{}, errors.New("decode tool call: a name and arguments are required")
	}
	return c, nil
}

// EncodeToolReply validates and encodes a tool reply frame payload.
func EncodeToolReply(r ToolReply) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

// DecodeToolReply decodes and validates a tool reply frame payload.
func DecodeToolReply(payload []byte) (ToolReply, error) {
	var r ToolReply
	if err := decodeStrict(payload, &r); err != nil {
		return ToolReply{}, fmt.Errorf("decode tool reply: %w", err)
	}
	if err := r.validate(); err != nil {
		return ToolReply{}, err
	}
	return r, nil
}

func (r ToolReply) validate() error {
	hasResult := len(r.Result) != 0
	if hasResult == (r.Error != nil) {
		return errors.New("tool reply: exactly one of result and error is required")
	}
	if hasResult && !json.Valid(r.Result) {
		return errors.New("tool reply: result is not valid JSON")
	}
	return nil
}

func (c ControlMessage) validate() error {
	set := 0
	for _, present := range []bool{c.Hello != nil, c.Start != nil, c.Started != nil, c.Stop != nil, c.Exited != nil, c.Failure != nil, c.Census != nil} {
		if present {
			set++
		}
	}
	var matches bool
	switch c.Type {
	case TypeHello:
		matches = c.Hello != nil
	case TypeStart:
		matches = c.Start != nil && len(c.Start.Argv) != 0 && c.Start.Dir != ""
	case TypeStarted:
		matches = c.Started != nil
	case TypeStop:
		matches = c.Stop != nil && c.Stop.GraceMillis >= 0
	case TypeExited:
		matches = c.Exited != nil
	case TypeFailure:
		matches = c.Failure != nil
	case TypeCensus:
		matches = c.Census != nil
	default:
		return fmt.Errorf("control message: unknown type %q", c.Type)
	}
	if !matches || set != 1 {
		return fmt.Errorf("control message %q: exactly its own complete body is required", c.Type)
	}
	return nil
}

func decodeStrict(payload []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(payload))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.More() {
		return errors.New("trailing data")
	}
	return nil
}
