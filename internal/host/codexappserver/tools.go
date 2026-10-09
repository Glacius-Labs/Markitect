package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"
	"unicode/utf8"
)

// DynamicTool is the pinned 0.162.0 function-tool surface. The Host owns tool
// semantics, scope and any helper reservations; the transport adds no task graph.
type DynamicTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}
type ToolCall struct {
	ThreadID  string          `json:"threadId"`
	TurnID    string          `json:"turnId"`
	CallID    string          `json:"callId"`
	Tool      string          `json:"tool"`
	Namespace string          `json:"namespace"`
	Arguments json.RawMessage `json:"arguments"`
}
type ToolResult struct {
	Success bool
	Text    string
}

var toolName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

func (o Options) validateTools(timeout time.Duration) error {
	if len(o.DynamicTools) == 0 {
		if o.HandleToolCall != nil || o.MaxToolCalls != 0 || o.ToolTimeout != 0 {
			return errors.New("dynamic tool handler requires explicit specs and limits")
		}
		return nil
	}
	if len(o.DynamicTools) > 16 || o.HandleToolCall == nil || o.MaxToolCalls < 1 || o.MaxToolCalls > 64 || o.ToolTimeout <= 0 || o.ToolTimeout > timeout {
		return errors.New("dynamic tools require bounded specs, handler, call count and timeout")
	}
	seen := map[string]bool{}
	for _, tool := range o.DynamicTools {
		var schema map[string]any
		if tool.Type != "function" || !toolName.MatchString(tool.Name) || seen[tool.Name] || len(tool.Description) > 8192 || !utf8.ValidString(tool.Description) || len(tool.InputSchema) > 64<<10 || json.Unmarshal(tool.InputSchema, &schema) != nil || schema["type"] != "object" {
			return errors.New("invalid dynamic tool specification")
		}
		seen[tool.Name] = true
	}
	return nil
}

func (s *session) toolRequest(ctx context.Context, msg envelope) error {
	if msg.Method != "item/tool/call" {
		return ErrApprovalRequired
	}
	var call ToolCall
	if json.Unmarshal(msg.Params, &call) != nil || call.ThreadID != s.h.ThreadID || call.TurnID == "" || call.TurnID != s.h.TurnID || call.CallID == "" || call.Namespace != "" || len(call.Arguments) > 64<<10 {
		return ErrProtocol
	}
	found := false
	for _, v := range s.a.options.DynamicTools {
		if v.Name == call.Tool {
			found = true
		}
	}
	if !found || s.a.options.HandleToolCall == nil {
		return ErrProtocol
	}
	if s.toolCalls[call.CallID] || len(s.toolCalls) >= s.a.options.MaxToolCalls {
		return errors.New("dynamic tool duplicate/call limit")
	}
	s.toolCalls[call.CallID] = true
	ctx, cancel := context.WithTimeout(ctx, s.a.options.ToolTimeout)
	defer cancel()
	type completed struct {
		result ToolResult
		err    error
	}
	done := make(chan completed, 1)
	go func() { result, err := s.a.options.HandleToolCall(ctx, call); done <- completed{result, err} }()
	var output ToolResult
	select {
	case <-ctx.Done():
		return ctx.Err()
	case v := <-done:
		if v.err != nil {
			return v.err
		}
		output = v.result
	}
	if !utf8.ValidString(output.Text) || len(output.Text) > 64<<10 {
		return errors.New("dynamic tool output limit")
	}
	return s.c.send(ctx, map[string]any{"id": msg.ID, "result": map[string]any{"success": output.Success, "contentItems": []any{map[string]string{"type": "inputText", "text": output.Text}}}})
}
