package agents_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
)

func newAgents(t *testing.T, f *fixture, login []byte) (*agents.Agents, string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "credentials.json")
	if login != nil {
		if err := os.WriteFile(file, login, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	transcripts := filepath.Join(dir, "transcripts")
	g, err := agents.New(agents.Config{Runtime: f.rt, Provider: claude.Provider{}, Image: f.spec.Image, Limits: f.spec.Limits,
		Network: "bridge", LoginFile: file, Model: "claude-sonnet-5-5", Effort: "high", Transcripts: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	return g, transcripts
}

func TestRunEndsFinishedWithUsageTranscriptAndNothingLeft(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		s.line(resultLine)
		return 0
	}
	g, transcripts := newAgents(t, f, f.spec.Login)
	end := g.Run(context.Background(), f.spec.Start, f.host)
	if end.Reason != agentport.Finished || end.Cleanup != "" || end.Agent != "claude:claude-sonnet-5-5" || end.Usage != (agentport.Usage{InputTokens: 60, OutputTokens: 40}) {
		t.Fatalf("end %+v", end)
	}
	if !strings.HasPrefix(f.rt.spec.Name, "markitect-run-r1-members-maker-1-") || f.rt.spec.Labels[agents.RunLabel] != "r1" || f.rt.spec.Network != "bridge" || f.rt.spec.MoreNetworks[0] != "markitect-run-r1" {
		t.Fatalf("container %+v", f.rt.spec)
	}
	if len(f.rt.volumes) != 0 || !f.rt.session.closed || !f.rt.session.cleared {
		t.Fatal("the start left its container, login or volumes")
	}
	data, err := os.ReadFile(filepath.Join(transcripts, "r1", "members-maker-1.jsonl"))
	if err != nil || !strings.Contains(string(data), `"subtype":"init"`) || strings.Contains(string(data), "SECRET") {
		t.Fatalf("transcript %v: %s", err, data)
	}
}

func TestRunNamesTheFailureClassFirst(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		s.line(`{"type":"assistant","parent_tool_use_id":null,"error":"rate_limit","message":{"role":"assistant","content":[]},"session_id":"s-1"}`)
		s.line(`{"type":"result","subtype":"success","is_error":true,"result":"API Error: 429","session_id":"s-1","usage":{}}`)
		return 1
	}
	g, _ := newAgents(t, f, f.spec.Login)
	if end := g.Run(context.Background(), f.spec.Start, f.host); end.Reason != agentport.Failed || !strings.HasPrefix(end.Detail, "rate-limit: ") {
		t.Fatalf("end %+v", end)
	}
	g, _ = newAgents(t, f, nil)
	if end := g.Run(context.Background(), f.spec.Start, f.host); end.Reason != agentport.Failed || !strings.HasPrefix(end.Detail, "login: ") {
		t.Fatalf("no login file: %+v", end)
	}
	expired := setup(t, agentport.Maker, "members", "members/member")
	g, _ = newAgents(t, expired, login(t, time.Now().Add(-time.Hour)))
	if end := g.Run(context.Background(), expired.spec.Start, expired.host); end.Reason != agentport.Failed || !strings.HasPrefix(end.Detail, "login: ") || end.Cleanup != "" {
		t.Fatalf("expired login: %+v", end)
	}
}

func TestRunAtTheDeadlineAndOnCancellation(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		<-s.stop
		return -1
	}
	f.spec.Start.Deadline = time.Now().Add(3 * time.Second)
	g, _ := newAgents(t, f, f.spec.Login)
	if end := g.Run(context.Background(), f.spec.Start, f.host); end.Reason != agentport.DeadlineReached || end.Cleanup != "" {
		t.Fatalf("end %+v", end)
	}
	f = setup(t, agentport.Maker, "members", "members/member")
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		<-s.stop
		return -1
	}
	g, _ = newAgents(t, f, f.spec.Login)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(1500*time.Millisecond, cancel)
	if end := g.Run(ctx, f.spec.Start, f.host); end.Reason != agentport.Cancelled {
		t.Fatalf("end %+v", end)
	}
}

func TestNewNeedsTheWholeConfiguration(t *testing.T) {
	if _, err := agents.New(agents.Config{Provider: claude.Provider{}, Image: "x", LoginFile: "f", Model: "m", Effort: "high"}); err == nil {
		t.Fatal("a configuration without a runtime was accepted")
	}
}
