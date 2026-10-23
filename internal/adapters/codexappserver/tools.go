package codexappserver

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var errUnknownTool = errors.New("unknown native Run tool")

func validTools(t run.Tools) bool {
	return t.Delegate != nil && t.Await != nil && t.Review != nil && t.Ask != nil && t.Reply != nil && t.ReadNotes != nil && t.AppendNote != nil
}
func toolSpecs() []frame.ToolSpec {
	finding := map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"manager": map[string]string{"type": "string"}, "text": map[string]string{"type": "string"}}, "required": []string{"manager", "text"}, "additionalProperties": false}}
	text := map[string]string{"type": "string"}
	var out []frame.ToolSpec
	add := func(name, description string, properties map[string]any, required ...string) {
		if required == nil {
			required = []string{}
		}
		schema, _ := json.Marshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
		out = append(out, frame.ToolSpec{Name: name, Description: description, InputSchema: schema})
	}
	add("delegate", "Assign a task and Findings to a direct child Manager. Completion arrives through await.", map[string]any{"manager": text, "task": text, "findings": finding}, "manager", "task", "findings")
	add("await", "Wait for the next incoming question, answer, parent assignment or fixed child Git handoff. Answer incoming questions while your own question is pending.", map[string]any{})
	add("review", "Review a fixed direct-child commit: approve integrates it, reject sends Findings for repair, model_blocker escalates a material model defect.", map[string]any{"manager": text, "commit": text, "decision": map[string]any{"type": "string", "enum": []string{"approve", "reject", "model_blocker"}}, "findings": finding}, "manager", "commit", "decision", "findings")
	add("ask", "Ask another Run Manager a question. Returns a new pending ID immediately; use await for its answer and any counter-question.", map[string]any{"to": text, "text": text}, "to", "text")
	add("reply", "Answer the exact incoming question ID authored by another Manager.", map[string]any{"id": text, "text": text}, "id", "text")
	add("read_notes", "Read durable project Notes, including earlier Runs. Notes do not override canonical intent.", map[string]any{})
	add("append_note", "Append a durable observation with its context to shared project Notes. Never include credentials.", map[string]any{"context": text, "text": text}, "context", "text")
	return out
}
func dispatch(ctx context.Context, t run.Tools, name string, raw []byte) (string, error) {
	var value any
	var err error
	switch name {
	case "delegate":
		var a struct {
			Manager, Task string
			Findings      []run.Finding
		}
		if err = strictValue(raw, &a); err == nil {
			err = t.Delegate(ctx, a.Manager, a.Task, a.Findings)
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
			Findings        []run.Finding
		}
		if err = strictValue(raw, &a); err == nil {
			err = t.Review(ctx, a.Manager, a.Commit, a.Decision, a.Findings)
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
	case "read_notes":
		var a struct{}
		if err = strictValue(raw, &a); err == nil {
			value, err = t.ReadNotes(ctx)
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
