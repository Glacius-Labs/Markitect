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

// onTopGit lands the Run's commit, lets a colleague land on top of it and
// then reports the push as failed, so the target's new tip is neither the
// Run's commit nor the tip it pushed over.
type onTopGit struct {
	run.Git
	landed string
}

func (g *onTopGit) Push(ctx context.Context, branch, commit, old string) error {
	if branch != "main" {
		return g.Git.Push(ctx, branch, commit, old)
	}
	if err := g.Git.Push(ctx, branch, commit, old); err != nil {
		return err
	}
	next, err := g.Git.Write(ctx, commit, map[string][]byte{"README.md": []byte("# Library, as a colleague wrote it\n")}, "a colleague's change")
	if err != nil {
		return err
	}
	if err := g.Git.Push(ctx, branch, next, commit); err != nil {
		return err
	}
	g.landed = commit
	return errors.New("connection reset")
}

func TestLandingPushThatFailedUnderLaterWorkLanded(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	g := &onTopGit{Git: f.host}
	f.engine.Git = g
	scriptHappyPath(f, nil)
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || r.Outcome != run.Landed || r.Landed != g.landed {
		t.Fatalf("%s/%s, landed %s, pushed %s: %s", r.Outcome, r.State, r.Landed, g.landed, r.Why)
	}
	if strings.TrimSpace(testkit.Git(t, f.target, "rev-parse", "main^")) != g.landed {
		t.Fatal("the colleague's change is not on top of the landing")
	}
}

// branchDeleteFails cannot delete Run branches.
type branchDeleteFails struct{ run.Git }

func (g branchDeleteFails) Push(ctx context.Context, branch, commit, old string) error {
	if strings.HasPrefix(branch, run.RunBranchPrefix) && commit == "" {
		return errors.New("branch deletion refused")
	}
	return g.Git.Push(ctx, branch, commit, old)
}

// runAll runs a Run that may leave cleanup behind.
func (f *fixture) runAll(req run.Request) *run.Result {
	f.t.Helper()
	req.Target, req.Limits = "main", run.Limits{Deadline: time.Now().Add(5 * time.Minute), Starts: 40, Rounds: 4, Start: 2 * time.Minute}
	r, err := f.engine.Run(context.Background(), req)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}

// The landed page is part of the landing commit, so what the cleanup after
// landing leaves is in the returned Result only, and the page says so; what
// the page states agrees with the returned Result.
func TestALandedPageAgreesWithTheResult(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.engine.Git = branchDeleteFails{f.host}
	f.remote.closeErr = errors.New("remote removal refused")
	scriptHappyPath(f, nil)
	r := f.runAll(run.Request{Base: base, Mode: agentport.Apply})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	stored := f.show(r.Landed, run.ResultFile(r.Run))
	if page := strings.TrimSpace(string(run.Page(r))); stored != page {
		t.Fatalf("the stored page differs from the returned Result's:\n%s\n---\n%s", stored, page)
	}
	if len(r.Cleanup) != 0 || len(r.AfterLanding) != 2 || !strings.Contains(r.AfterLanding[0], "branch deletion refused") || !strings.Contains(r.AfterLanding[1], "remote removal refused") {
		t.Fatalf("cleanup %q, after landing %q", r.Cleanup, r.AfterLanding)
	}
	if !strings.Contains(stored, "`afterLanding`") {
		t.Fatalf("the landed page does not say where the cleanup after landing is reported:\n%s", stored)
	}
}

// A Run that keeps its work removes its remote and leaves the lock before it
// writes its page, so the page says what they left.
func TestAKeptPageAgreesWithTheResult(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.remote.closeErr = errors.New("remote removal refused")
	a := f.agents
	a.On("catalog", checker, a.Verdicts(word("conforms")))
	a.On(".", checker, a.Verdicts(word("conforms")))
	r := f.runAll(run.Request{Base: base, Mode: agentport.Verify})
	if r.State != run.StateVerified || r.AfterLanding != nil {
		t.Fatalf("%s: %s, after landing %q", r.State, r.Why, r.AfterLanding)
	}
	stored := f.show(run.RunBranchPrefix+r.Run, run.ResultFile(r.Run))
	if page := strings.TrimSpace(string(run.Page(r))); stored != page {
		t.Fatalf("the stored page differs from the returned Result's:\n%s\n---\n%s", stored, page)
	}
	if !strings.Contains(stored, "remote removal refused") {
		t.Fatalf("the page omits the remote's cleanup:\n%s", stored)
	}
}

// An apply that ends kept in landing leaves the lock and removes its remote
// before its page is written, as every Run that does not land.
func TestAKeptApplyPageAgreesWithTheResult(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.remote.closeErr = errors.New("remote removal refused")
	scriptHappyPath(f, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		testkit.Git(t, f.src, "push", "-q", "--force", "origin", base+":main")
		return f.agents.Review(nil)(ctx, s, h)
	})
	r := f.runAll(run.Request{Base: base, Mode: agentport.Apply})
	if r.State != run.StateKept || r.AfterLanding != nil || len(r.Replaced) != 0 {
		t.Fatalf("%s: %s, after landing %q, replaced %v", r.State, r.Why, r.AfterLanding, r.Replaced)
	}
	stored := f.show(run.RunBranchPrefix+r.Run, run.ResultFile(r.Run))
	if page := strings.TrimSpace(string(run.Page(r))); stored != page {
		t.Fatalf("the stored page differs from the returned Result's:\n%s\n---\n%s", stored, page)
	}
	if !strings.Contains(stored, "remote removal refused") || len(r.Cleanup) != 1 {
		t.Fatalf("cleanup %q; page:\n%s", r.Cleanup, stored)
	}
	// The Run left the lock: a second Run of its name may enter.
	if data, _, err := f.host.Lock(context.Background(), "main"); err != nil || strings.Contains(string(data), r.Run) {
		t.Fatalf("lock record %s: %v", data, err)
	}
}
