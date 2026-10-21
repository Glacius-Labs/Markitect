package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// contextPrompt is a standing context with what a cut or substitution loses
// first: a recipient line, a decisive final sentence, structured values with
// boundaries, resolved link targets and text beyond 64 KiB.
func contextPrompt() string {
	var b strings.Builder
	b.WriteString("You are the Manager of Package lending/reminders.\n\nlending/lending-policy-copy (lending/LendingPolicy): loan-policy\n")
	for i := range 2000 {
		fmt.Fprintf(&b, "Context line %d of the bound model, größer als 60 Zeichen ✓.\n", i)
	}
	b.WriteString("  maxOpenLoans: 5\n  loanPeriodDays: 14\n  overdue: false\n  links to: library/inputs, lending/loan\n")
	b.WriteString("A lower limit neither deletes existing loans nor forces their return; while at or above that limit, the member cannot start another loan.\n")
	return b.String()
}

// deliveredUser returns the text of the user message among the adapter's
// writes to the provider.
func deliveredUser(writes [][]byte) (string, error) {
	for _, w := range writes {
		var m struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(w, &m) == nil && m.Type == "user" {
			return m.Message.Content, nil
		}
	}
	return "", errors.New("no user message was written to the provider")
}

// sameText reports how a delivered text differs from the expected one.
func sameText(delivered, want string) error {
	if delivered == want {
		return nil
	}
	n := 0
	for n < len(delivered) && n < len(want) && delivered[n] == want[n] {
		n++
	}
	return fmt.Errorf("delivered %d bytes, expected %d; they differ from byte %d", len(delivered), len(want), n)
}

// The Turn's prompt reaches the provider byte for byte and an Orchestrator
// tool answer, such as a looked-up Definition, comes back to it byte for byte:
// the comparison is against the independently built text, not the encoder's
// own output, and it detects a cut suffix, a changed value or another
// recipient.
func TestContextCrossesTheAdapterBoundaryUnchanged(t *testing.T) {
	prompt := contextPrompt()
	answer := strings.Repeat("lending/loan-policy (context owned by lending, bound model): the full text.\n", 3000) + "while at or above that limit, the member cannot start another loan."
	rt := newFakeRuntime()
	rt.calls = []frame.ToolCall{{ID: 1, Name: "look_up", Arguments: json.RawMessage(`{"topic":"definition","id":"lending/loan-policy"}`)}}
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Prompt = prompt
	turn.Tools = []run.ToolSpec{{Name: "look_up", Description: "Look up.", InputSchema: json.RawMessage(`{"type":"object"}`)}}
	if _, err := h.agents.Run(context.Background(), turn, func(context.Context, string, json.RawMessage) run.ToolAnswer {
		return run.ToolAnswer{Text: answer}
	}); err != nil {
		t.Fatal(err)
	}
	delivered, err := deliveredUser(rt.writes)
	if err != nil {
		t.Fatal(err)
	}
	if err := sameText(delivered, prompt); err != nil {
		t.Fatalf("the prompt changed at the adapter boundary: %v", err)
	}
	// The same comparison of the actually written payload detects a faulty
	// transport that cuts, changes or readdresses the message on its way.
	for name, fault := range map[string]func(string) string{
		"cut final sentence": func(s string) string { return strings.TrimSuffix(s, "start another loan.\n") },
		"changed value":      func(s string) string { return strings.Replace(s, "maxOpenLoans: 5", "maxOpenLoans: 4", 1) },
		"other recipient":    func(s string) string { return strings.Replace(s, "Package lending/reminders", "Package lending", 1) },
	} {
		frt := &faultyRuntime{fakeRuntime: newFakeRuntime(), fault: fault}
		agents, err := NewAgentsWithRuntime(func(context.Context, container.ContainerSpec) (RuntimeSession, error) { return frt, nil },
			map[string]Profile{"claude": validProfile()}, func(context.Context, string) (RuntimeCredential, error) {
				return RuntimeCredential{Token: testToken}, nil
			})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := agents.Run(context.Background(), turn, echoHandler(new([]string))); err != nil {
			t.Fatal(err)
		}
		got, err := deliveredUser(frt.writes)
		if err != nil || sameText(got, prompt) == nil {
			t.Errorf("the comparison misses a %s on the wire: %v", name, err)
		}
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if len(rt.replies) != 1 || json.Unmarshal(rt.replies[0].Result, &result) != nil || len(result.Content) != 1 || result.IsError {
		t.Fatalf("tool reply %+v", rt.replies)
	}
	if err := sameText(result.Content[0].Text, answer); err != nil {
		t.Fatalf("the looked-up text changed at the adapter boundary: %v", err)
	}
}

// A prompt the wire limit cannot carry is refused before any container starts,
// never cut to fit.
func TestAnOversizedPromptIsRefusedNotCut(t *testing.T) {
	rt := newFakeRuntime()
	profile := validProfile()
	h := newHarness(t, rt, profile)
	turn := testTurn()
	turn.Prompt = strings.Repeat("x", int(profile.MaxWireBytes))
	res, err := h.agents.Run(context.Background(), turn, echoHandler(&[]string{}))
	if !errors.Is(err, ErrLimit) || !errors.Is(err, run.ErrContextLimit) || !strings.Contains(err.Error(), fmt.Sprintf("the prompt of %d bytes", len(turn.Prompt))) || h.starts != 0 || len(rt.writes) != 0 || !res.Stopped || !strings.Contains(res.Reason, "no container started") {
		t.Fatalf("an oversized prompt: %+v %v, starts %d, writes %d", res, err, h.starts, len(rt.writes))
	}
}

// A compaction the provider reports in its stream reaches the Orchestrator, so
// it can deliver the standing context again; without one nothing is claimed.
func TestAReportedCompactionIsPassedOn(t *testing.T) {
	for _, compacted := range []bool{false, true} {
		rt := newFakeRuntime()
		if compacted {
			rt.extra = []string{`{"type":"system","subtype":"compact_boundary","session_id":"conversation-1","uuid":"cb-1","compact_metadata":{"trigger":"auto","pre_tokens":180000}}`}
		}
		h := newHarness(t, rt, validProfile())
		res, err := h.agents.Run(context.Background(), testTurn(), echoHandler(&[]string{}))
		if err != nil || res.Compacted != compacted {
			t.Fatalf("compacted %v: %+v %v", compacted, res, err)
		}
	}
}

// faultyRuntime is a transport that alters the user message on its way to the
// provider, keeping it valid JSON.
type faultyRuntime struct {
	*fakeRuntime
	fault func(string) string
}

func (f *faultyRuntime) WriteInput(data []byte) error {
	var m map[string]any
	if json.Unmarshal(data, &m) == nil && m["type"] == "user" {
		message, _ := m["message"].(map[string]any)
		content, _ := message["content"].(string)
		message["content"] = f.fault(content)
		changed, err := json.Marshal(m)
		if err != nil {
			return err
		}
		data = append(changed, '\n')
	}
	return f.fakeRuntime.WriteInput(data)
}

// An Orchestrator answer the channel cannot carry is refused to the Agent with
// its size, never cut; a compaction the provider reports is named in the next
// answer, which points to look_up brief.
func TestAnOversizedAnswerIsRefusedAndACompactionNamed(t *testing.T) {
	rt := newFakeRuntime()
	rt.extra = []string{`{"type":"system","subtype":"compact_boundary","session_id":"conversation-1","uuid":"cb-1","compact_metadata":{"trigger":"auto","pre_tokens":180000}}`}
	// The stream and the tool calls are separate channels; the delay lets the
	// compaction arrive first, as it does from a provider that compacts before
	// it calls a tool.
	rt.callDelay = 200 * time.Millisecond
	rt.calls = []frame.ToolCall{
		{ID: 1, Name: "look_up", Arguments: json.RawMessage(`{"topic":"brief"}`)},
		{ID: 2, Name: "look_up", Arguments: json.RawMessage(`{"topic":"brief"}`)},
	}
	h := newHarness(t, rt, validProfile())
	turn := testTurn()
	turn.Tools = []run.ToolSpec{{Name: "look_up", Description: "Look up.", InputSchema: json.RawMessage(`{"type":"object"}`)}}
	answers := []string{strings.Repeat("y", maxToolAnswer+1), "short"}
	n := 0
	if _, err := h.agents.Run(context.Background(), turn, func(context.Context, string, json.RawMessage) run.ToolAnswer {
		n++
		return run.ToolAnswer{Text: answers[n-1]}
	}); err != nil {
		t.Fatal(err)
	}
	var first, second struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if len(rt.replies) != 2 || json.Unmarshal(rt.replies[0].Result, &first) != nil || json.Unmarshal(rt.replies[1].Result, &second) != nil {
		t.Fatalf("replies %+v", rt.replies)
	}
	if !first.IsError || !strings.Contains(first.Content[0].Text, fmt.Sprintf("The answer of %d bytes exceeds", maxToolAnswer+1)) || strings.Contains(first.Content[0].Text, "yyyy") {
		t.Errorf("oversized answer: %+v", first)
	}
	if !strings.Contains(first.Content[0].Text, "look_up topic brief") || strings.Contains(second.Content[0].Text, "compacted") || second.Content[0].Text != "short" {
		t.Errorf("the compaction note: %q then %q", first.Content[0].Text, second.Content[0].Text)
	}
}
