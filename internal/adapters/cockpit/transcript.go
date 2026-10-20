package cockpit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	maxTranscriptEntries = 4000
	maxTranscriptText    = 6000
)

// Transcript is one turn's provider stream as the adapter recorded it, with
// secrets already redacted by the producer. Until cuts it at the boundary's
// recorded time; Truncated says that entries or texts were shortened.
type Transcript struct {
	Turn      string            `json:"turn"`
	Until     time.Time         `json:"until,omitzero"`
	Entries   []TranscriptEntry `json:"entries"`
	Later     int               `json:"later"`
	Truncated bool              `json:"truncated"`
	Unread    int               `json:"unread"`
}

// TranscriptEntry is one presented line: who it is from (input, assistant,
// thinking, tool, orchestrator, provider) and its text.
type TranscriptEntry struct {
	At   time.Time `json:"at,omitzero"`
	From string    `json:"from"`
	Name string    `json:"name,omitempty"`
	Text string    `json:"text"`
}

// parseTranscript reads the wrapped provider stream. Lines after until are
// counted, not shown; lines that do not decode are counted as unread.
func parseTranscript(turn string, data []byte, until time.Time) Transcript {
	t := Transcript{Turn: turn, Until: until, Entries: []TranscriptEntry{}}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var w struct {
			At   time.Time       `json:"at"`
			Kind string          `json:"kind"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(line, &w) != nil {
			t.Unread++
			continue
		}
		if !until.IsZero() && w.At.After(until) {
			t.Later++
			continue
		}
		for _, e := range transcriptEntries(w.Kind, w.Data) {
			if len(t.Entries) == maxTranscriptEntries {
				t.Truncated = true
				return t
			}
			e.At = w.At
			if len([]rune(e.Text)) > maxTranscriptText {
				e.Text, t.Truncated = clip(e.Text, maxTranscriptText), true
			}
			t.Entries = append(t.Entries, e)
		}
	}
	return t
}

func transcriptEntries(kind string, data json.RawMessage) []TranscriptEntry {
	var d map[string]json.RawMessage
	_ = json.Unmarshal(data, &d)
	str := func(m map[string]json.RawMessage, key string) string {
		var s string
		_ = json.Unmarshal(m[key], &s)
		return s
	}
	switch kind {
	case "tool_call":
		return []TranscriptEntry{{From: "tool", Name: str(d, "name"), Text: compact(d["arguments"])}}
	case "tool_answer":
		var refused bool
		_ = json.Unmarshal(d["refused"], &refused)
		from := "orchestrator"
		text := str(d, "text")
		if refused {
			text = "Refused: " + text
		}
		return []TranscriptEntry{{From: from, Text: text}}
	case "end":
		var stopped bool
		_ = json.Unmarshal(d["stopped"], &stopped)
		return []TranscriptEntry{{From: "runtime", Text: fmt.Sprintf("Turn ended: %s (container stop observed: %t)", str(d, "reason"), stopped)}}
	case "sent", "provider":
	default:
		return []TranscriptEntry{{From: "runtime", Name: kind, Text: compact(data)}}
	}
	typ := str(d, "type")
	switch typ {
	case "user", "assistant":
		var msg struct {
			Content json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(d["message"], &msg)
		var text string
		if json.Unmarshal(msg.Content, &text) == nil {
			from := "assistant"
			if typ == "user" {
				from = "input"
			}
			return []TranscriptEntry{{From: from, Text: text}}
		}
		var blocks []map[string]json.RawMessage
		_ = json.Unmarshal(msg.Content, &blocks)
		var out []TranscriptEntry
		for _, b := range blocks {
			switch str(b, "type") {
			case "text":
				from := "assistant"
				if typ == "user" {
					from = "input"
				}
				out = append(out, TranscriptEntry{From: from, Text: str(b, "text")})
			case "thinking":
				if s := str(b, "thinking"); strings.TrimSpace(s) != "" {
					out = append(out, TranscriptEntry{From: "thinking", Text: s})
				}
			case "tool_use":
				name := str(b, "name")
				if strings.HasPrefix(name, "mcp__markitect__") {
					continue // the Orchestrator's own tool_call line records it
				}
				out = append(out, TranscriptEntry{From: "tool", Name: name, Text: compact(b["input"])})
			case "tool_result":
				out = append(out, TranscriptEntry{From: "result", Text: toolResult(b["content"])})
			}
		}
		return out
	case "result":
		return []TranscriptEntry{{From: "provider", Name: "result", Text: fmt.Sprintf("%s, stop %s", str(d, "subtype"), str(d, "stop_reason"))}}
	case "system":
		if str(d, "subtype") == "init" {
			return []TranscriptEntry{{From: "provider", Name: "session", Text: "Session started with model " + str(d, "model")}}
		}
		return nil
	case "control_request", "control_response", "rate_limit_event":
		return nil
	}
	return []TranscriptEntry{{From: "provider", Name: typ, Text: compact(data)}}
}

func toolResult(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return compact(content)
}

func compact(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}
