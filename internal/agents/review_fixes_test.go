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
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// The deadline passes while submit waits for the remote's hook: the tool
// call is cut off and awaited, and the start ends at its deadline, not as a
// failure, with nothing of the call left behind.
func TestTheDeadlineCutsOffARunningToolCall(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	hook := filepath.Join(f.bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nsleep 12\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The hook's sleep outlives the cut-off Git client; Windows keeps the
	// repository busy until it ends.
	hookEnd := time.Now().Add(15 * time.Second)
	t.Cleanup(func() { time.Sleep(time.Until(hookEnd)) })
	var text string
	var isErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine(makerTools...))
		text, isErr = s.call("submit", map[string]any{"summary": "slow", "verdicts": []map[string]string{{"item": "members/member", "verdict": "conforms", "evidence": "e"}}})
		<-s.stop
		return -1
	}
	f.spec.Start.Deadline = time.Now().Add(4 * time.Second)
	a := f.start(t)
	began := time.Now()
	out, err := a.Send(context.Background(), "")
	if err != nil || out.End != agents.Deadline {
		t.Fatalf("outcome %+v %v", out, err)
	}
	if !isErr || time.Since(began) > 10*time.Second {
		t.Fatalf("the cut-off call answered %v %q after %s", isErr, text, time.Since(began))
	}
}

func TestACheckerRejectsAnIntegratingParentWithoutADispute(t *testing.T) {
	f := setup(t, agentport.Checker, ".")
	maker := filepath.Join(t.TempDir(), "maker")
	git(t, filepath.Dir(maker), "clone", "-q", "--branch", "root", f.bare, maker)
	git(t, maker, "commit", "-q", "--allow-empty", "-m", "Integrate\n\nMarkitect-Manager: .")
	git(t, maker, "push", "-q", "origin", "HEAD:root")
	var text string
	var isErr bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		s.line(initLine("review", "ask", "escalate", "answer", "assign", "note", "notes"))
		text, isErr = s.call("review", map[string]any{"outcome": "rejected", "checks": []map[string]string{}})
		s.line(resultLine)
		return 0
	}
	a := f.start(t)
	if _, err := a.Send(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if isErr {
		t.Fatalf("rejection refused: %s", text)
	}
	tr, err := runremote.ParseMessage(git(t, f.bare, "log", "-1", "--format=%B", "root"))
	if err != nil || tr.Review != runremote.Rejected {
		t.Fatalf("review %+v %v", tr, err)
	}
}

func TestAStartPastItsDeadlineEndsAtTheDeadline(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	f.spec.Start.Deadline = time.Now().Add(-time.Second)
	g, _ := newAgents(t, f, login(t, time.Now().Add(time.Hour)))
	end := g.Run(context.Background(), f.spec.Start, f.host)
	if end.Reason != agentport.DeadlineReached || end.Cleanup != "" || !strings.Contains(end.Detail, "deadline") && !strings.Contains(end.Detail, "context") {
		t.Fatalf("end %+v", end)
	}
}
