package run_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agentport/agentfake"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// scriptHappyPath lets catalog and Root realize the book change.
func scriptHappyPath(f *fixture, rootChecker agentfake.Step) {
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	if rootChecker == nil {
		rootChecker = a.Review(nil)
	}
	a.On(".", checker, rootChecker)
}

// flakyGit fails the landing push on the target: after the push took
// effect, or without pushing, and then without being able to read the
// target again.
type flakyGit struct {
	run.Git
	pushed   bool // the push takes effect before it fails
	unread   bool // after the failed push, the target cannot be read
	attempts int
}

func (g *flakyGit) Push(ctx context.Context, branch, commit, old string) error {
	if branch != "main" {
		return g.Git.Push(ctx, branch, commit, old)
	}
	g.attempts++
	if g.pushed {
		if err := g.Git.Push(ctx, branch, commit, old); err != nil {
			return err
		}
	}
	return errors.New("connection reset")
}

func (g *flakyGit) Fetch(ctx context.Context, branch string) (string, error) {
	if g.unread && g.attempts > 0 {
		return "", errors.New("connection refused")
	}
	return g.Git.Fetch(ctx, branch)
}

func TestLandingPushThatFailedAfterItTookEffectLanded(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.engine.Git = &flakyGit{Git: f.host, pushed: true}
	scriptHappyPath(f, nil)
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || f.tip("main") != r.Landed {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

func TestLandingPushWithUnknownOutcomeIsUncertain(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	f.engine.Git = &flakyGit{Git: f.host, unread: true}
	scriptHappyPath(f, nil)
	r := f.run(run.Request{Base: base})
	if r.State != run.StateIncomplete || r.Outcome != run.NotLanded || !strings.Contains(r.Why, "uncertain") {
		t.Fatalf("%s/%s: %s", r.Outcome, r.State, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("the target changed")
	}
}

func TestLandingRefusesARewrittenTarget(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	scriptHappyPath(f, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		// The owner drops the Run's start from the target.
		testkit.Git(t, f.src, "push", "-q", "--force", "origin", base+":main")
		return f.agents.Review(nil)(ctx, s, h)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "rewritten") || f.tip("main") != base {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// A login file that a parent's merge commit adds is in the history that
// landing pushes, so the work does not land.
func TestLoginFileInAMergeDoesNotLand(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		for _, step := range []func() error{
			func() error { _, err := w.Git("fetch", "-q", "origin", "managers/catalog"); return err },
			func() error { _, err := w.Git("merge", "-q", "--no-commit", "--no-ff", "FETCH_HEAD"); return err },
			func() error { return w.Write(".credentials.json", "{}\n") },
			func() error { _, err := w.Git("add", "-A"); return err },
			func() error { _, err := w.Git("commit", "-q", "-m", "merge catalog"); return err },
			func() error { return w.Remove(".credentials.json") },
			func() error { return w.Commit(agentfake.SubmissionMessage(".", "Submit", word("conforms")(s))) },
			w.Push,
		} {
			if err := step(); err != nil {
				return agentfake.Fail(err)
			}
		}
		return agentfake.Done()
	})
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, ".credentials.json") || f.tip("main") != start {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// A parent that ignores its reminder too leaves the Run where nobody can go
// on, and the Result names the open question.
func TestIgnoredReminderNamesTheOpenQuestion(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
		_, err := h.Ask(ctx, "Is the isbn optional?")
		return err
	}))
	a.On(".", maker, a.Submit(nil, nil, word("conforms")), a.Submit(nil, nil, word("conforms")))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateIncomplete || !strings.Contains(r.Why, ". did not answer m1") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

func TestRequestsTheEngineRefuses(t *testing.T) {
	f := newFixture(t)
	start := f.tip("main")
	ok := run.Limits{Deadline: time.Now().Add(time.Minute), Starts: 1, Rounds: 1}
	for name, req := range map[string]run.Request{
		"run name":     {Run: "Bad/Name", Limits: ok},
		"no starts":    {Limits: run.Limits{Deadline: ok.Deadline, Rounds: 1}},
		"no rounds":    {Limits: run.Limits{Deadline: ok.Deadline, Starts: 1}},
		"past":         {Limits: run.Limits{Deadline: time.Now().Add(-time.Minute), Starts: 1, Rounds: 1}},
		"unknown mode": {Mode: "improve", Limits: ok},
	} {
		req.Target, req.Base = "main", start
		if req.Mode == "" {
			req.Mode = agentport.Apply
		}
		if _, err := f.engine.Run(context.Background(), req); !errors.Is(err, run.ErrUsage) {
			t.Errorf("%s: err %v", name, err)
		}
	}
}
