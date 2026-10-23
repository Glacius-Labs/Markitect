package codexappserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var inboxNative = flag.Bool("markitect-private-inbox", false, "explicit finite authority for one native session with a synthetic inbox counterpart")

// TestInboxNativeProbe uses the actual Codex/agentd MCP path. Its counterpart
// is a four-call test fixture, not a second Manager, product inbox or Run.
func TestInboxNativeProbe(t *testing.T) {
	if !*inboxNative {
		t.Skip("UNEXECUTED: separate finite native inbox dispatch required")
	}
	runPrivateNativeProbe(t, true)
}

type nativeInboxEvidence struct {
	SyntheticCounterpart bool
	Q1, Q2               string
	Thread               string
	Steps                []string
	MessageThreads       []string
	ToolCallIDs          []uint64
	ToolThreads          []string
	TranscriptSHA256     string
	RetainedTask         bool
}

type nativeInboxProbe struct {
	mu                   sync.Mutex
	task, answer, q1, q2 string
	failed               bool
	observed             nativeInboxEvidence
}

func newNativeInboxProbe(task, answer string) *nativeInboxProbe {
	return &nativeInboxProbe{task: "task-" + task, answer: "synthetic-answer-" + answer,
		q1: "synthetic-Q1", q2: "synthetic-Q2",
		observed: nativeInboxEvidence{SyntheticCounterpart: true, Q1: "synthetic-Q1", Q2: "synthetic-Q2"}}
}

func (*nativeInboxProbe) instructions() string {
	return "This is a technical inbox probe with an explicitly synthetic counterpart, not another real Manager. " +
		"Use only inbox_ask, inbox_await and inbox_reply. Ask returns a question ID immediately. " +
		"Await takes no arguments and can deliver an incoming Question while your own question remains unanswered. " +
		"Answer an incoming Question using inbox_reply with its exact ID, then await again for your original Answer. " +
		"Do not issue parallel tool calls or repeat a call. On the first message make exactly four calls: ask, await, reply, await. " +
		"Then state the original assignment token and the received answer verbatim. On the second message use no tools and recall both values."
}

func (*nativeInboxProbe) tools() []run.ToolSpec {
	return []run.ToolSpec{
		{Name: "inbox_ask", Description: "Ask the explicitly synthetic counterpart and receive a correlation ID immediately.", InputSchema: json.RawMessage(`{"type":"object","properties":{"to":{"type":"string"},"text":{"type":"string"}},"required":["to","text"],"additionalProperties":false}`)},
		{Name: "inbox_await", Description: "Receive the next inbox event: Question or Answer. Your own question can remain pending.", InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)},
		{Name: "inbox_reply", Description: "Answer the incoming question using its exact correlation ID and your own answer text.", InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"text":{"type":"string"}},"required":["id","text"],"additionalProperties":false}`)},
	}
}

func (p *nativeInboxProbe) handle(ctx context.Context, name string, raw json.RawMessage) run.ToolAnswer {
	p.mu.Lock()
	defer p.mu.Unlock()
	refuse := func() run.ToolAnswer {
		p.failed = true
		return run.ToolAnswer{Refused: true, Text: "inbox probe refused invalid order, correlation or payload"}
	}
	if p.failed || ctx.Err() != nil || len(raw) > 4096 {
		return refuse()
	}
	var text string
	switch len(p.observed.Steps) {
	case 0:
		var args struct{ To, Text string }
		if name != "inbox_ask" || strictValue(raw, &args) != nil || args.To != "synthetic-counterpart" || strings.TrimSpace(args.Text) == "" {
			return refuse()
		}
		text = p.q1
	case 1:
		if name != "inbox_await" || strictValue(raw, &struct{}{}) != nil {
			return refuse()
		}
		// The original Q1 remains pending. This is an independent incoming Q2.
		event, _ := json.Marshal(map[string]any{"Question": map[string]string{"ID": p.q2, "From": "synthetic-counterpart", "To": "native-manager", "Text": "Reply with exactly the original assignment token from your first user message."}})
		text = string(event)
	case 2:
		var args struct{ ID, Text string }
		if name != "inbox_reply" || strictValue(raw, &args) != nil || args.ID != p.q2 || args.Text != p.task {
			return refuse()
		}
		text = `{"accepted":true}`
	case 3:
		if name != "inbox_await" || strictValue(raw, &struct{}{}) != nil {
			return refuse()
		}
		// Only the native-authored correct Reply(Q2) unlocks Answer(Q1).
		event, _ := json.Marshal(map[string]any{"Answer": map[string]string{"ID": p.q1, "From": "synthetic-counterpart", "Text": p.answer}})
		text = string(event)
	default:
		return refuse()
	}
	p.observed.Steps = append(p.observed.Steps, name)
	return run.ToolAnswer{Text: text}
}

func (p *nativeInboxProbe) exercise(ctx context.Context, session run.AssignmentSession) error {
	for i, message := range []string{
		"Your original assignment token is " + p.task + ". Ask synthetic-counterpart for its answer token and follow the inbox probe instructions. Keep your original assignment token in memory.",
		"Without tools, recall the original assignment token and the received answer token verbatim, and identify the counterpart as synthetic.",
	} {
		result, err := session.Message(ctx, message, 45*time.Second)
		p.mu.Lock()
		p.observed.MessageThreads = append(p.observed.MessageThreads, result.Conversation)
		if i == 0 {
			p.observed.Thread = result.Conversation
		}
		valid := err == nil && result.Completed && result.Conversation != "" && result.Conversation == p.observed.Thread &&
			!p.failed && len(p.observed.Steps) == 4 && len(result.Commands) == 0 &&
			strings.Contains(result.Final, p.task) && strings.Contains(result.Final, p.answer)
		if i == 1 && valid {
			p.observed.RetainedTask = true
		}
		p.mu.Unlock()
		if !valid {
			return errors.New("native inbox order, answer, completion or same-session recall failed")
		}
	}
	return nil
}

// Correlate the actual recorded MCP calls/replies, not fixture-authored IDs or
// inferred events. The production adapter independently checks thread metadata.
func (p *nativeInboxProbe) verifyTranscript(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if err != nil || len(raw) > 16<<20 {
		return errors.New("inbox transcript exceeds bound or cannot be read")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observed.ToolCallIDs, p.observed.ToolThreads = nil, nil
	replies := map[uint64]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var entry struct{ At, Direction, Data string }
		if strictValue(line, &entry) != nil {
			return errors.New("invalid inbox transcript entry")
		}
		switch entry.Direction {
		case "tool-call":
			call, err := frame.DecodeToolCall([]byte(entry.Data))
			var meta struct {
				ThreadID string `json:"threadId"`
			}
			n := len(p.observed.ToolCallIDs)
			if err != nil || n >= len(p.observed.Steps) || call.Name != p.observed.Steps[n] || json.Unmarshal(call.Meta, &meta) != nil || meta.ThreadID != p.observed.Thread || meta.ThreadID == "" {
				return errors.New("inbox tool thread/order mismatch")
			}
			for _, id := range p.observed.ToolCallIDs {
				if id == call.ID {
					return errors.New("duplicate inbox tool call ID")
				}
			}
			p.observed.ToolCallIDs = append(p.observed.ToolCallIDs, call.ID)
			p.observed.ToolThreads = append(p.observed.ToolThreads, meta.ThreadID)
		case "tool-reply":
			reply, err := frame.DecodeToolReply([]byte(entry.Data))
			ids := p.observed.ToolCallIDs
			if err != nil || len(ids) == 0 || reply.ID != ids[len(ids)-1] || replies[reply.ID] || reply.Error != nil {
				return errors.New("inbox tool reply mismatch")
			}
			replies[reply.ID] = true
		}
	}
	if len(p.observed.ToolCallIDs) != 4 || len(replies) != 4 || p.failed || !p.observed.RetainedTask {
		return errors.New("inbox transcript incomplete")
	}
	sum := sha256.Sum256(raw)
	p.observed.TranscriptSHA256 = hex.EncodeToString(sum[:])
	return nil
}

func (p *nativeInboxProbe) evidence() *nativeInboxEvidence {
	p.mu.Lock()
	defer p.mu.Unlock()
	e := p.observed
	e.Steps = append([]string(nil), e.Steps...)
	e.MessageThreads = append([]string(nil), e.MessageThreads...)
	e.ToolCallIDs = append([]uint64(nil), e.ToolCallIDs...)
	e.ToolThreads = append([]string(nil), e.ToolThreads...)
	return &e
}

func inboxFixtureCalls(p *nativeInboxProbe) []frame.ToolCall {
	return []frame.ToolCall{
		{ID: 11, Name: "inbox_ask", Arguments: json.RawMessage(`{"to":"synthetic-counterpart","text":"What is your answer token?"}`)},
		{ID: 12, Name: "inbox_await", Arguments: json.RawMessage(`{}`)},
		{ID: 13, Name: "inbox_reply", Arguments: json.RawMessage(`{"id":"` + p.q2 + `","text":"` + p.task + `"}`)},
		{ID: 14, Name: "inbox_await", Arguments: json.RawMessage(`{}`)},
	}
}

// These deterministic checks protect the probe's oracle from false positives;
// they supply no inference, engine or two-Manager evidence.
func TestInboxProbeRejectsWrongOrderCorrelationAndPayload(t *testing.T) {
	for _, tc := range []struct {
		name       string
		before     int
		tool, args string
	}{
		{"await-before-ask", 0, "inbox_await", `{}`},
		{"wrong-recipient", 0, "inbox_ask", `{"to":"real-manager","text":"question"}`},
		{"unknown-field", 0, "inbox_ask", `{"to":"synthetic-counterpart","text":"question","fallback":true}`},
		{"duplicate-key", 0, "inbox_ask", `{"to":"synthetic-counterpart","text":"question","text":"other"}`},
		{"answer-before-native-reply", 2, "inbox_await", `{}`},
		{"foreign-question", 2, "inbox_reply", `{"id":"synthetic-Q1","text":"task-task"}`},
		{"wrong-native-answer", 2, "inbox_reply", `{"id":"synthetic-Q2","text":"wrong"}`},
		{"duplicate-reply", 3, "inbox_reply", `{"id":"synthetic-Q2","text":"task-task"}`},
		{"extra-call", 4, "inbox_await", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newNativeInboxProbe("task", "answer")
			calls := inboxFixtureCalls(p)
			for _, c := range calls[:tc.before] {
				if p.handle(context.Background(), c.Name, c.Arguments).Refused {
					t.Fatal("fixture rejected")
				}
			}
			if !p.handle(context.Background(), tc.tool, json.RawMessage(tc.args)).Refused {
				t.Fatal("invalid exchange accepted")
			}
			if !p.handle(context.Background(), calls[0].Name, calls[0].Arguments).Refused {
				t.Fatal("failed probe recovered silently")
			}
		})
	}
	p := newNativeInboxProbe("task", "answer")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !p.handle(ctx, "inbox_ask", inboxFixtureCalls(p)[0].Arguments).Refused {
		t.Fatal("canceled call accepted")
	}
}

func TestInboxProbeBindsActualTranscriptThreadAndReplies(t *testing.T) {
	for _, mode := range []string{"valid", "foreign-thread", "duplicate-call", "foreign-reply", "missing-reply", "missing-recall"} {
		t.Run(mode, func(t *testing.T) {
			p := newNativeInboxProbe("task", "answer")
			p.observed.Thread, p.observed.RetainedTask = "thread-actual", mode != "missing-recall"
			var transcript bytes.Buffer
			appendEntry := func(direction string, data []byte) {
				line, err := json.Marshal(struct{ At, Direction, Data string }{"now", direction, string(data)})
				if err != nil {
					t.Fatal(err)
				}
				transcript.Write(line)
				transcript.WriteByte('\n')
			}
			for i, c := range inboxFixtureCalls(p) {
				answer := p.handle(context.Background(), c.Name, c.Arguments)
				if answer.Refused {
					t.Fatal("fixture rejected")
				}
				if i == 2 && mode == "foreign-thread" {
					c.Meta = json.RawMessage(`{"threadId":"thread-other"}`)
				} else {
					c.Meta = json.RawMessage(`{"threadId":"thread-actual"}`)
				}
				if i == 2 && mode == "duplicate-call" {
					c.ID = 11
				}
				call, err := frame.EncodeToolCall(c)
				if err != nil {
					t.Fatal(err)
				}
				appendEntry("tool-call", call)
				if i == 3 && mode == "missing-reply" {
					continue
				}
				id := c.ID
				if i == 2 && mode == "foreign-reply" {
					id = 99
				}
				result, _ := json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": answer.Text}}, "isError": false})
				reply, err := frame.EncodeToolReply(frame.ToolReply{ID: id, Result: result})
				if err != nil {
					t.Fatal(err)
				}
				appendEntry("tool-reply", reply)
			}
			path := t.TempDir() + string(os.PathSeparator) + "transcript.jsonl"
			if err := os.WriteFile(path, transcript.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			err := p.verifyTranscript(path)
			if (err == nil) != (mode == "valid") {
				t.Fatalf("mode %s: %v", mode, err)
			}
			if mode == "valid" && (p.evidence().TranscriptSHA256 == "" || len(p.evidence().ToolThreads) != 4) {
				t.Fatal("observations missing")
			}
		})
	}
}
