package container_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// These capability probes answer the runtime questions of the step-basis
// design with a real Claude Code session. They need a Claude credential in
// addition to the engine and runtime image.

// claudeConversation drives one stream-JSON Claude Code process.
type claudeConversation struct {
	t         *testing.T
	s         *container.Session
	events    *bufio.Scanner
	sessionID string
}

func startClaude(t *testing.T, r *runtimeSmoke, s *container.Session, resume string) *claudeConversation {
	t.Helper()
	args := []string{"claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{"markitect":{"type":"stdio","command":"` + container.AgentdPath + `","args":["mcp"]}}}`,
		"--tools", "Read,Write", "--allowedTools", "mcp__markitect__submit_work", "Read", "Write", "--permission-mode", "dontAsk"}
	if resume != "" {
		args = append(args, "--resume", resume)
	}
	if _, err := s.StartProvider(r.ctx, frame.Start{Argv: args, Dir: "/work", Env: credentialEnv("ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"), Tools: []frame.ToolSpec{hostTool}}); err != nil {
		t.Fatal(err)
	}
	go answerTools(s, make(chan frame.ToolCall, 16))
	events := bufio.NewScanner(s.Stdout())
	events.Buffer(make([]byte, 64<<10), frame.MaxPayload)
	return &claudeConversation{t: t, s: s, events: events}
}

// turn sends one user message and returns the final result text.
func (c *claudeConversation) turn(prompt string) string {
	c.t.Helper()
	message, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": prompt}})
	if err := c.s.WriteInput(append(message, '\n')); err != nil {
		c.t.Fatal(err)
	}
	for c.events.Scan() {
		var e struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
			IsError   bool   `json:"is_error"`
			Result    string `json:"result"`
		}
		if json.Unmarshal(c.events.Bytes(), &e) != nil {
			continue
		}
		if e.SessionID != "" {
			c.sessionID = e.SessionID
		}
		if e.Type == "result" {
			if e.IsError {
				c.t.Fatalf("Claude Code result error: %s; diagnostics %q", e.Result, c.s.Diagnostics())
			}
			return e.Result
		}
	}
	c.t.Fatalf("Claude Code output ended; diagnostics %q", c.s.Diagnostics())
	return ""
}

func needClaudeCredential(t *testing.T) {
	if len(credentialEnv("ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN")) == 0 {
		t.Skip("set ANTHROPIC_API_KEY or CLAUDE_CODE_OAUTH_TOKEN")
	}
}

// TestClaudeConversationContinuesInANewEnvironment ends a conversation's
// container, advances the workspace basis while no session runs, and resumes
// the conversation from its retained provider home in a new container.
func TestClaudeConversationContinuesInANewEnvironment(t *testing.T) {
	r := newRuntimeSmoke(t)
	needClaudeCredential(t)
	word := fmt.Sprintf("lantern-%d", time.Now().UnixNano()%100000)
	mounts := r.volumes("resume")

	first := r.start("resume-a", mounts)
	a := startClaude(t, r, first, "")
	a.turn("Remember this code word for later: " + word + ". Do not write any file. Reply with OK.")
	if a.sessionID == "" {
		t.Fatal("no Claude session ID observed")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first environment: %v", err)
	}

	r.writeFile("resume-basis", mounts, "/work/basis.txt", "basis-2\n")

	second := r.start("resume-b", mounts)
	b := startClaude(t, r, second, a.sessionID)
	answer := b.turn("What was the code word I gave you earlier, and what does the file basis.txt in the working directory contain? Answer in one line.")
	if !strings.Contains(answer, word) || !strings.Contains(answer, "basis-2") {
		t.Fatalf("resumed answer %q lacks the code word %s or the new basis", answer, word)
	}
}

// TestClaudeSessionSeesAQuiescentBasisAdvance keeps one execution session,
// checks that only the provider's own processes remain after a turn, then
// changes the workspace through a helper container and continues.
func TestClaudeSessionSeesAQuiescentBasisAdvance(t *testing.T) {
	r := newRuntimeSmoke(t)
	needClaudeCredential(t)
	mounts := r.volumes("advance")
	s := r.start("advance", mounts)
	c := startClaude(t, r, s, "")
	c.turn("Read the file basis.txt in the working directory if it exists and say what it contains, or say MISSING. Do not write files.")

	// The Host waits for quiescence: only the provider's own processes remain.
	deadline := time.Now().Add(15 * time.Second)
	for {
		processes, err := s.Census(r.ctx)
		if err != nil {
			t.Fatal(err)
		}
		if quiescent(processes) {
			t.Logf("quiescent census after the turn: %+v", processes)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the session did not become quiescent: %+v", processes)
		}
		time.Sleep(200 * time.Millisecond)
	}

	r.writeFile("advance-basis", mounts, "/work/basis.txt", "basis-3\n")
	if answer := c.turn("Read basis.txt again now and give its exact content."); !strings.Contains(answer, "basis-3") {
		t.Fatalf("the continued session did not see the advanced basis: %q", answer)
	}
}

// quiescent accepts only Claude Code itself and its MCP bridge; an ended
// process awaiting its parent's wait does not count.
func quiescent(processes []frame.Process) bool {
	for _, p := range processes {
		if p.State == "Z" {
			continue
		}
		if !strings.Contains(p.Command, "claude") && !strings.Contains(p.Command, "node") && !strings.Contains(p.Command, "markitect-agent") {
			return false
		}
	}
	return true
}
