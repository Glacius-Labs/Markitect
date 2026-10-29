package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agentport/agentfake"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// second is another host on the same target: its own private repository,
// Run remotes and scripted agents, as a second `markitect apply` would have.
func (f *fixture) second() (*run.Engine, *agentfake.Agents) {
	f.t.Helper()
	dir := testkit.TempDir(f.t)
	host, err := rungit.Open(context.Background(), filepath.Join(dir, "host"), f.target)
	if err != nil {
		f.t.Fatal(err)
	}
	a := agentfake.New(filepath.Join(dir, "agents"))
	return &run.Engine{Git: host, Remotes: &remotes{t: f.t, dir: filepath.Join(dir, "remotes")}, Agents: a, Cleanup: time.Minute, LockPoll: 20 * time.Millisecond}, a
}

// held is a maker Step that says it started and waits until released
// before it submits, so a test can start another Run while this one works.
func held(started chan<- struct{}, release <-chan struct{}, then agentfake.Step) agentfake.Step {
	return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return agentfake.Fail(ctx.Err())
		}
		return then(ctx, s, h)
	}
}

type outcome struct {
	r   *run.Result
	err error
}

func limits() run.Limits {
	return run.Limits{Deadline: time.Now().Add(5 * time.Minute), Starts: 40, Rounds: 4, Start: 2 * time.Minute}
}

func runAsync(e *run.Engine, req run.Request) <-chan outcome {
	done := make(chan outcome, 1)
	go func() {
		r, err := e.Run(context.Background(), req)
		done <- outcome{r, err}
	}()
	return done
}

// TestOverlappingRunsTakeTurns runs two applies on one target whose impact
// overlaps: the second waits in the target's lock until the first has
// ended, starts from what the first left, and its landing deletes the first
// Run's kept branch, whose impact it covers.
func TestOverlappingRunsTakeTurns(t *testing.T) {
	f := newFixture(t)
	f.engine.LockPoll = 20 * time.Millisecond
	base, _ := changeBook(f)
	started, release := make(chan struct{}), make(chan struct{})
	a := f.agents
	a.On("catalog", maker, held(started, release, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))
	first := runAsync(f.engine, run.Request{Mode: agentport.Apply, Run: "first", Target: "main", Base: base, Limits: limits()})
	<-started

	// The owner changes the book again while the first Run works.
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "A book has a title, an `isbn` and a `year`.")})
	start := f.commit("book has a year")
	e2, b := f.second()
	b.On("catalog", maker, b.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string; Year int }\n"}, word("changed")))
	b.On("catalog", checker, b.Review(nil))
	b.On(".", maker, b.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	b.On(".", checker, b.Review(nil))
	second := runAsync(e2, run.Request{Mode: agentport.Apply, Run: "second", Target: "main", Base: base, Limits: limits()})

	time.Sleep(500 * time.Millisecond)
	if n := len(b.Starts()); n > 0 {
		t.Fatalf("the second Run started %d agents while an overlapping Run works", n)
	}
	select {
	case o := <-second:
		t.Fatalf("the second Run ended while an overlapping Run works: %+v %v", o.r, o.err)
	default:
	}
	close(release)

	o1 := <-first
	if o1.err != nil {
		t.Fatal(o1.err)
	}
	// The owner's second change touches the first Run's impact, so its
	// work is kept, not landed.
	if o1.r.State != run.StateKept || o1.r.Branch != run.RunBranchPrefix+"first" {
		t.Fatalf("first: %s: %s", o1.r.State, o1.r.Why)
	}
	o2 := <-second
	if o2.err != nil {
		t.Fatal(o2.err)
	}
	r := o2.r
	if r.State != run.StateLanded || r.Start != start {
		t.Fatalf("second: %s from %s: %s", r.State, r.Start, r.Why)
	}
	if len(r.Queue) == 0 || !strings.Contains(r.Queue[0], "first") {
		t.Fatalf("queue %q", r.Queue)
	}
	if strings.Join(r.Replaced, " ") != run.RunBranchPrefix+"first" || f.tip(run.RunBranchPrefix+"first") != "" {
		t.Fatalf("the covered Run branch was not deleted: replaced %v", r.Replaced)
	}
	page := f.show("main", ".markitect/results/second.md")
	if !strings.Contains(page, "## Queue") || !strings.Contains(page, "## Replaced Run branches") {
		t.Fatalf("Result page:\n%s", page)
	}
	if !strings.Contains(f.show("main", "src/catalog/book.go"), "Year") {
		t.Fatal("the second Run's work did not land")
	}
	if data, _, err := f.host.Lock(context.Background(), "main"); err != nil || data != nil {
		t.Fatalf("the lock is not empty after both Runs: %s %v", data, err)
	}
}

// changeLoan changes the loan Definition: loans, members and Root work,
// and catalog does not.
func changeLoan(f *fixture) string {
	f.write(map[string]string{m + "members/loans/loan.md": def("std/Concept", "Loan", "A [[members/member]] borrows a book for `21` days.")})
	return f.commit("a loan lasts 21 days")
}

func scriptLoan(a *agentfake.Agents) {
	a.On("members/loans", maker, a.Submit(nil, map[string]string{"src/members/loans/loan.go": "package loans\n\nconst Days = 21\n"}, word("changed")))
	a.On("members/loans", checker, a.Review(nil))
	a.On("members", maker, a.Submit([]string{"managers/members.loans"}, nil, word("conforms")))
	a.On("members", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/members"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))
}

// TestDisjointRunsWorkSideBySide runs a second apply whose impact does not
// overlap the first's: it starts and lands while the first still works, and
// the first then lands on the moved target.
func TestDisjointRunsWorkSideBySide(t *testing.T) {
	f := newFixture(t)
	f.engine.LockPoll = 20 * time.Millisecond
	base, bookStart := changeBook(f)
	started, release := make(chan struct{}), make(chan struct{})
	a := f.agents
	a.On("catalog", maker, held(started, release, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))
	first := runAsync(f.engine, run.Request{Mode: agentport.Apply, Run: "first", Target: "main", Base: base, Limits: limits()})
	<-started

	changeLoan(f)
	e2, b := f.second()
	scriptLoan(b)
	// Its base is the first Run's start, so its impact is the loan alone.
	r, err := e2.Run(context.Background(), run.Request{Mode: agentport.Apply, Run: "second", Target: "main", Base: bookStart, Limits: limits()})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateLanded || len(r.Queue) > 0 {
		t.Fatalf("second: %s, queue %q: %s", r.State, r.Queue, r.Why)
	}
	close(release)
	o := <-first
	if o.err != nil {
		t.Fatal(o.err)
	}
	if o.r.State != run.StateLanded || !strings.Contains(o.r.Why, "moved") {
		t.Fatalf("first: %s: %s", o.r.State, o.r.Why)
	}
	if !strings.Contains(f.show("main", "src/members/loans/loan.go"), "21") || !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("both Runs' work is not on main")
	}
}

// TestLandingKeepsRunBranchesItDoesNotCover keeps one Run on its branch,
// then lands a Run with another impact: the kept branch stays. The next
// default base is the landing's own work, so a change the landing merged in
// from the moved target still reaches the next Run.
func TestLandingKeepsRunBranchesItDoesNotCover(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, f.approveAfter(map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Name string }\n"}))
	kept := f.run(run.Request{Run: "kept", Base: base})
	if kept.State != run.StateKept {
		t.Fatalf("%s: %s", kept.State, kept.Why)
	}
	message := testkit.Git(t, f.target, "log", "-1", "--format=%B", run.RunBranchPrefix+"kept")
	for _, want := range []string{"Markitect-Run: kept", "Markitect-Start: " + kept.Start, "Markitect-Impact: def:catalog/book", "Markitect-Impact: file:src/catalog/book.go"} {
		if !strings.Contains(message, want) {
			t.Fatalf("the kept Result commit lacks %q:\n%s", want, message)
		}
	}

	f.pull()
	loanBase := f.tip("main")
	loanStart := changeLoan(f)
	scriptLoan(a)
	r := f.run(run.Request{Run: "loan", Base: loanBase})
	if r.State != run.StateLanded || len(r.Replaced) > 0 {
		t.Fatalf("%s, replaced %v: %s", r.State, r.Replaced, r.Why)
	}
	if f.tip(run.RunBranchPrefix+"kept") == "" {
		t.Fatal("a Run branch the landing does not cover was deleted")
	}
	if r.Start != loanStart {
		t.Fatalf("start %s", r.Start)
	}
}

// TestDefaultBaseIsTheLandedWork lands a Run on a moved target: what it
// merged in was not realized by it, so the next Run's impact still holds it.
func TestDefaultBaseIsTheLandedWork(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		// The owner pushes a loan change while Root's checker works.
		f.pull()
		changeLoan(f)
		return a.Review(nil)(ctx, s, h)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || !strings.Contains(r.Why, "moved") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	scriptLoan(a)
	next := f.run(run.Request{})
	if next.State != run.StateLanded {
		t.Fatalf("the merged-in loan change did not reach the next Run: %s: %s", next.State, next.Why)
	}
	if items(next) != "members/loans:members/loans/loan=changed" {
		t.Fatalf("items %s", items(next))
	}
}

// TestTheLock refuses a second Run of the same name, ends a Run that waits
// past its deadline without starting anything, ignores an entry past its
// time and leaves verify out.
func TestTheLock(t *testing.T) {
	f := newFixture(t)
	f.engine.LockPoll = 20 * time.Millisecond
	base, _ := changeBook(f)
	ctx := context.Background()
	set := func(entries ...map[string]any) {
		t.Helper()
		_, version, err := f.host.Lock(ctx, "main")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]any{"format": 1, "runs": entries})
		if err := f.host.SetLock(ctx, "main", data, version); err != nil {
			t.Fatal(err)
		}
	}
	overlapping := []string{"def:catalog/book"}

	set(map[string]any{"run": "taken", "until": time.Now().Add(time.Hour), "impact": []string{"def:members/member"}})
	if _, err := f.engine.Run(ctx, run.Request{Mode: agentport.Apply, Run: "taken", Target: "main", Base: base, Limits: limits()}); !errors.Is(err, run.ErrUsage) {
		t.Fatalf("a second Run named taken: %v", err)
	}

	set(map[string]any{"run": "busy", "until": time.Now().Add(time.Hour), "impact": overlapping})
	lim := limits()
	lim.Deadline = time.Now().Add(time.Second)
	r, err := f.engine.Run(ctx, run.Request{Mode: agentport.Apply, Run: "waits", Target: "main", Base: base, Limits: lim})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateIncomplete || len(f.agents.Starts()) > 0 || len(r.Queue) == 0 || !strings.Contains(r.Queue[0], "busy") {
		t.Fatalf("%s, queue %q: %s", r.State, r.Queue, r.Why)
	}

	// A host that died leaves its entry; past its time it no longer counts.
	set(map[string]any{"run": "dead", "until": time.Now().Add(-time.Minute), "impact": overlapping})
	scriptHappyPath(f, nil)
	if r := f.run(run.Request{Run: "after", Base: base}); r.State != run.StateLanded || len(r.Queue) > 0 {
		t.Fatalf("%s, queue %q: %s", r.State, r.Queue, r.Why)
	}
	if data, _, err := f.host.Lock(ctx, "main"); err != nil || data != nil {
		t.Fatalf("the lock still holds %s %v", data, err)
	}
	// verify takes no place in the lock.
	set(map[string]any{"run": "busy", "until": time.Now().Add(time.Hour), "impact": overlapping})
	a := f.agents
	a.On("catalog", checker, a.Verdicts(word("conforms")))
	a.On(".", checker, a.Verdicts(word("conforms")))
	if r := f.run(run.Request{Mode: agentport.Verify, Run: "check", Base: base}); r.State != run.StateVerified || len(r.Queue) > 0 {
		t.Fatalf("verify: %s, queue %q: %s", r.State, r.Queue, r.Why)
	}
}
