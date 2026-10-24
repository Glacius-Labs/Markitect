package codexappserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var errUnknownTool = errors.New("unknown native Run tool")

func validTools(t run.Tools) bool {
	return t.Delegate != nil && t.Await != nil && t.Review != nil && t.Ask != nil && t.Reply != nil && t.Judge != nil && t.Check != nil && t.ReadNotes != nil && t.AppendNote != nil
}

// toolSpecs maps Run's fixed tool set. Descriptions state each tool's purpose
// and arguments; Run's refusals name the reason and the possible next step.
func toolSpecs() []frame.ToolSpec {
	finding := map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"text": map[string]string{"type": "string"}}, "required": []string{"text"}, "additionalProperties": false}}
	text := map[string]string{"type": "string"}
	var out []frame.ToolSpec
	add := func(name, description string, properties map[string]any, required ...string) {
		if required == nil {
			required = []string{}
		}
		schema, _ := json.Marshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
		out = append(out, frame.ToolSpec{Name: name, Description: description, InputSchema: schema})
	}
	add("delegate", "Give a direct child Manager a task with Findings. After you reviewed its fixed handoff, a correction needs concrete Findings. For Implement, the child works from your current working state.", map[string]any{"manager": text, "task": text, "findings": finding}, "manager", "task", "findings")
	add("await", "Receive your next event: an assignment, a question, an answer or a direct child's fixed handoff with its diff and check result.", map[string]any{})
	add("review", "Decide on a direct child's fixed handoff by its commit. approve accepts it and integrates an Implement submission into your working copy; reject returns it with Findings; model_blocker escalates a material model defect.", map[string]any{"manager": text, "commit": text, "decision": map[string]any{"type": "string", "enum": []string{"approve", "reject", "model_blocker"}}, "findings": finding}, "manager", "commit", "decision", "findings")
	add("ask", "Ask your direct parent or a direct child a question. Returns the question ID; the answer arrives through await.", map[string]any{"to": text, "text": text}, "to", "text")
	add("reply", "Answer a question addressed to you, by its ID.", map[string]any{"id": text, "text": text}, "id", "text")
	add("judge", "Record your judgment of your assignment with Findings. Implement: submitted, reject or model_blocker; every Manager including Root uses submitted for completed candidate work, never approve. Assessment/Verify: approve, reject or model_blocker. Reject and model_blocker require concrete Findings. Reviewing a child with review is separate.", map[string]any{"decision": map[string]any{"type": "string", "enum": []string{"approve", "reject", "model_blocker", "submitted"}}, "findings": finding}, "decision", "findings")
	add("check", "Run the deterministic check of your current state: structural validity, accounting and the project checks placed in your copy. It lists the checks that ran.", map[string]any{})
	add("read_notes", "Read durable project Notes, including earlier Runs, oldest first and one page at a time. The reply lists notes and next. Pass next as after to read the following page; an empty next marks the last page. Notes do not override canonical intent.", map[string]any{"after": text})
	add("append_note", fmt.Sprintf("Append a durable observation with its context to shared project Notes. The text may have up to %d bytes and the context up to %d bytes. Never include credentials.", run.MaxNoteText, run.MaxNoteContext), map[string]any{"context": text, "text": text}, "context", "text")
	return out
}

type nativeFinding struct {
	Text string `json:"text"`
}

// findings drops nothing but the authoring identity: Run sets each origin.
func findings(in []nativeFinding) []run.Finding {
	out := make([]run.Finding, len(in))
	for i, f := range in {
		out[i] = run.Finding{Text: f.Text}
	}
	return out
}

func dispatch(ctx context.Context, t run.Tools, name string, raw []byte) (string, error) {
	var value any
	var err error
	if err = validateArguments(name, raw); err != nil {
		if !errors.Is(err, errUnknownTool) {
			err = fmt.Errorf("%w: invalid %s arguments; see the tool's input schema (a Finding is an object with one text string)", err, name)
		}
		return err.Error(), err
	}
	switch name {
	case "delegate":
		var a struct {
			Manager, Task string
			Findings      []nativeFinding
		}
		if err = strictValue(raw, &a); err == nil {
			err = t.Delegate(ctx, a.Manager, a.Task, findings(a.Findings))
		}
	case "await":
		var a struct{}
		if err = strictValue(raw, &a); err == nil {
			value, err = t.Await(ctx)
		}
	case "review":
		var a struct {
			Manager, Commit string
			Decision        run.Decision
			Findings        []nativeFinding
		}
		if err = strictValue(raw, &a); err == nil {
			err = t.Review(ctx, a.Manager, a.Commit, a.Decision, findings(a.Findings))
		}
	case "ask":
		var a struct{ To, Text string }
		if err = strictValue(raw, &a); err == nil {
			value, err = t.Ask(ctx, a.To, a.Text)
		}
	case "reply":
		var a struct{ ID, Text string }
		if err = strictValue(raw, &a); err == nil {
			err = t.Reply(ctx, a.ID, a.Text)
		}
	case "judge":
		var a struct {
			Decision run.Decision
			Findings []nativeFinding
		}
		if err = strictValue(raw, &a); err == nil {
			err = t.Judge(ctx, a.Decision, findings(a.Findings))
		}
	case "check":
		var a struct{}
		if err = strictValue(raw, &a); err == nil {
			value, err = t.Check(ctx)
		}
	case "read_notes":
		var a struct{ After string }
		if err = strictValue(raw, &a); err == nil {
			value, err = t.ReadNotes(ctx, a.After)
		}
	case "append_note":
		var a struct{ Context, Text string }
		if err = strictValue(raw, &a); err == nil {
			value, err = t.AppendNote(ctx, a.Context, a.Text)
		}
	default:
		return "", errUnknownTool
	}
	if err != nil {
		return err.Error(), err
	}
	if value == nil {
		return `{"ok":true}`, nil
	}
	data, err := json.Marshal(value)
	return string(data), err
}

// toolResult is the MCP tools/call result that carries one tool's text.
func toolResult(text string, failed bool) json.RawMessage {
	content, _ := json.Marshal(struct {
		Content []map[string]string `json:"content"`
		IsError bool                `json:"isError"`
	}{[]map[string]string{{"type": "text", "text": text}}, failed})
	return content
}

// Validate the concrete native tool envelope before any callback or durable write.
// These are closed provider-local inputs, not a general JSON Schema framework.
func validateArguments(name string, raw []byte) error {
	var keys []string
	optional := ""
	switch name {
	case "delegate":
		keys = []string{"manager", "task", "findings"}
	case "await", "check":
		keys = []string{}
	case "read_notes":
		keys, optional = []string{}, "after"
	case "review":
		keys = []string{"manager", "commit", "decision", "findings"}
	case "ask":
		keys = []string{"to", "text"}
	case "reply":
		keys = []string{"id", "text"}
	case "append_note":
		keys = []string{"context", "text"}
	case "judge":
		keys = []string{"decision", "findings"}
	default:
		return errUnknownTool
	}
	object, err := protocolObject(raw)
	if err != nil {
		return ErrSession
	}
	// One optional string argument is allowed beyond the required keys.
	if value, ok := object[optional]; ok {
		if _, err = protocolString(value); err != nil {
			return err
		}
		keys = append(keys, optional)
	}
	if len(object) != len(keys) {
		return ErrSession
	}
	for _, key := range keys {
		value, ok := object[key]
		if !ok {
			return ErrSession
		}
		if key == "findings" {
			if err = validateFindings(value); err != nil {
				return err
			}
			continue
		}
		text, err := protocolString(value)
		if err != nil {
			return err
		}
		if key == "decision" && !oneOf(text, "approve", "reject", "model_blocker", "submitted") {
			return ErrSession
		}
		if key == "decision" && name == "review" && text == "submitted" {
			return ErrSession
		}
	}
	return nil
}
func validateFindings(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return ErrSession
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return ErrSession
	}
	for _, item := range items {
		fields, err := protocolObject(item)
		if err != nil || len(fields) != 1 {
			return ErrSession
		}
		value, ok := fields["text"]
		if !ok {
			return ErrSession
		}
		if _, err = protocolString(value); err != nil {
			return err
		}
	}
	return nil
}
