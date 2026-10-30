package agents_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/agents/claude"
)

// push commits a file on a branch of the bare remote from a clone of its
// own, as another agent would, and returns the new tip.
func push(t *testing.T, bare, branch, file, content, message string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "other")
	git(t, filepath.Dir(dir), "clone", "-q", "--branch", branch, bare, dir)
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(file)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", message)
	git(t, dir, "push", "-q", "origin", "HEAD:"+branch)
	return git(t, bare, "rev-parse", branch)
}

// review answers the tip of a branch of the bare remote with an empty commit,
// as a checker does, and returns it.
func review(t *testing.T, bare, branch, outcome string) string {
	t.Helper()
	tip := git(t, bare, "rev-parse", branch)
	commit := git(t, bare, "commit-tree", tip+"^{tree}", "-p", tip, "-m", "Review members\n\nMarkitect-Manager: members\nMarkitect-Review: "+outcome+"\nMarkitect-Check: members/member | disputed | no test")
	git(t, bare, "update-ref", "refs/heads/"+branch, commit, tip)
	return commit
}

func agentsWith(t *testing.T, f *fixture, limit int64) (*agents.Agents, string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(file, f.spec.Login, 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := agents.New(agents.Config{Runtime: f.rt, Provider: claude.Provider{}, Image: f.spec.Image, Limits: f.spec.Limits,
		Network: "bridge", LoginFile: file, Model: "claude-sonnet-5-5", Effort: "high", ContextLimit: limit})
	if err != nil {
		t.Fatal(err)
	}
	return g, file
}

// A maker's later turn continues its conversation in the same container: the
// login copy is renewed from the host's login as it is now, the clone follows
// its branch past the checker's review, and the turn's message is the new
// task alone, which resumes the session.
func TestALaterMakerTurnContinuesInTheSameContainer(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	var tasks []string
	var resumed []bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		tasks, resumed = append(tasks, string(stdin)), append(resumed, slices.Contains(argv, "--resume") && slices.Contains(argv, "s-1"))
		s.line(initLine(makerTools...))
		if len(tasks) == 1 {
			os.WriteFile(filepath.Join(s.rt.work, "src", "members", "member.go"), []byte("package members\n\ntype Member struct{}\n"), 0o644)
			if text, isErr := s.call("submit", map[string]any{"summary": "Member", "verdicts": []map[string]string{
				{"item": "members/member", "verdict": "changed", "evidence": "type Member"}}}); isErr {
				t.Errorf("submit: %s", text)
			}
		}
		s.line(resultLine)
		return 0
	}
	g, file := agentsWith(t, f, 0)
	if end := g.Run(context.Background(), f.spec.Start, f.host); end.Reason != agentport.Finished || end.Continued || end.Cleanup != "" {
		t.Fatalf("first turn %+v", end)
	}
	if len(f.rt.volumes) != 2 || f.rt.session.closed {
		t.Fatal("the agent ended after a finished turn")
	}
	rejection := review(t, f.bare, "managers/members", "rejected")
	renewed := strings.ReplaceAll(string(f.spec.Login), "ACCESS-SECRET", "ACCESS-RENEWED")
	if err := os.WriteFile(file, []byte(renewed), 0o600); err != nil {
		t.Fatal(err)
	}
	second := f.spec.Start
	second.Round, second.Deadline = 2, time.Now().Add(time.Hour)
	second.Inbox = []agentport.Message{{ID: "m-3", Kind: agentport.Review, To: "members", Text: "Your checker rejected it: no test."}}
	end := g.Run(context.Background(), second, f.host)
	if end.Reason != agentport.Finished || !end.Continued || end.Cleanup != "" {
		t.Fatalf("second turn %+v", end)
	}
	if f.rt.starts != 1 || len(resumed) != 2 || resumed[0] || !resumed[1] {
		t.Fatalf("containers %d, resumed %v", f.rt.starts, resumed)
	}
	if !strings.Contains(tasks[1], "Your checker rejected it: no test.") || strings.Contains(tasks[1], "# Role") || strings.Contains(tasks[1], "# Model") {
		t.Fatalf("second task: %s", tasks[1])
	}
	if got := git(t, f.rt.work, "rev-parse", "HEAD"); got != rejection {
		t.Fatalf("the clone is at %s, not at the review %s", got, rejection)
	}
	copy := string(f.rt.renewed["/home/agent/.claude/.credentials.json"])
	if !strings.Contains(copy, "ACCESS-RENEWED") || strings.Contains(copy, "REFRESH") {
		t.Fatalf("renewed login copy %q", copy)
	}
	if left := g.Close(context.Background(), "r1"); len(left) != 0 || len(f.rt.volumes) != 0 || !f.rt.session.closed || !f.rt.session.cleared {
		t.Fatalf("close left %q, volumes %v", left, f.rt.volumes)
	}
}

// A checker's later turn reviews the new submission in its checkout: the
// working tree becomes the submission, without what a build of the earlier
// one left untracked, and the task asks for every verdict again in full.
func TestALaterCheckerTurnReviewsTheNewSubmissionInFull(t *testing.T) {
	f := setup(t, agentport.Checker, "members", "members/member")
	f.spec.Start.Tools = []agentport.Tool{agentport.ToolNote}
	verdict := "\n\nMarkitect-Manager: members\nMarkitect-Verdict: members/member | changed | type Member"
	push(t, f.bare, "managers/members", "src/members/member.go", "package members\n\ntype Member struct{}\n", "Member"+verdict)
	var tasks, files []string
	var leftover bool
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		tasks = append(tasks, string(stdin))
		data, _ := os.ReadFile(filepath.Join(s.rt.work, "src", "members", "member.go"))
		files = append(files, string(data))
		_, err := os.Stat(filepath.Join(s.rt.work, "build.out"))
		leftover = err == nil
		os.WriteFile(filepath.Join(s.rt.work, "build.out"), []byte("artifact\n"), 0o644)
		s.line(initLine("review", "note"))
		outcome, answer := "rejected", "disputed"
		if len(tasks) == 2 {
			outcome, answer = "approved", "confirmed"
		}
		if text, isErr := s.call("review", map[string]any{"outcome": outcome, "checks": []map[string]string{{"item": "members/member", "answer": answer, "evidence": "read member.go"}}}); isErr {
			t.Errorf("review: %s", text)
		}
		s.line(resultLine)
		return 0
	}
	g, _ := agentsWith(t, f, 0)
	if end := g.Run(context.Background(), f.spec.Start, f.host); end.Reason != agentport.Finished {
		t.Fatalf("first turn %+v", end)
	}
	second := push(t, f.bare, "managers/members", "src/members/member.go", "package members\n\ntype Member struct{ Name string }\n", "Name the member"+verdict)
	s := f.spec.Start
	s.Round, s.Deadline = 2, time.Now().Add(time.Hour)
	if end := g.Run(context.Background(), s, f.host); end.Reason != agentport.Finished || !end.Continued {
		t.Fatalf("second turn %+v", end)
	}
	if f.rt.starts != 1 || !strings.Contains(files[1], "Name string") || leftover {
		t.Fatalf("containers %d, the checkout holds %q, a build's leftover stayed: %v", f.rt.starts, files[1], leftover)
	}
	if strings.Contains(tasks[0], "do not carry over") || !strings.Contains(tasks[1], "do not carry over") || !strings.Contains(tasks[1], "submission `"+second+"`") {
		t.Fatalf("second task: %s", tasks[1])
	}
	if tr := git(t, f.bare, "log", "-1", "--format=%B", "managers/members"); !strings.Contains(tr, "Markitect-Review: approved") {
		t.Fatalf("the second review is not on the branch: %s", tr)
	}
	g.Close(context.Background(), "r1")
}

// An agent whose conversation reached the context limit, or whose turn did
// not finish, ends after it; its next turn starts it fresh from the remote.
func TestAnAgentEndsAtTheContextLimitOrAfterAFailedTurn(t *testing.T) {
	f := setup(t, agentport.Maker, "members", "members/member")
	turn := 0
	f.rt.script = func(s *fakeSession, argv []string, stdin []byte) int {
		turn++
		s.line(initLine(makerTools...))
		if turn == 2 {
			s.line(`{"type":"result","subtype":"success","is_error":true,"result":"API Error: 500","session_id":"s-1","usage":{}}`)
			return 1
		}
		s.line(`{"type":"assistant","parent_tool_use_id":null,"message":{"id":"m1","role":"assistant","content":[],"usage":{"input_tokens":5,"cache_creation_input_tokens":500,"cache_read_input_tokens":1500,"output_tokens":9}},"session_id":"s-1"}`)
		s.line(resultLine)
		return 0
	}
	g, _ := agentsWith(t, f, 2000)
	for i, want := range []agentport.Reason{agentport.Finished, agentport.Failed, agentport.Finished} {
		s := f.spec.Start
		s.Round, s.Deadline = i+1, time.Now().Add(time.Hour)
		end := g.Run(context.Background(), s, f.host)
		if end.Reason != want || end.Continued || end.Cleanup != "" || len(f.rt.volumes) != 0 || !f.rt.session.closed {
			t.Fatalf("turn %d: %+v, volumes %v", i+1, end, f.rt.volumes)
		}
	}
	if f.rt.starts != 3 {
		t.Fatalf("%d containers for three fresh turns", f.rt.starts)
	}
}
