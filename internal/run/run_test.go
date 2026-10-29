package run_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agentport/agentfake"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const (
	maker   = agentport.Maker
	checker = agentport.Checker
)

// word gives every item the same verdict.
func word(w string) func(agentport.Start) []agentfake.Verdict {
	return func(s agentport.Start) []agentfake.Verdict {
		var vs []agentfake.Verdict
		for _, it := range s.Checklist {
			vs = append(vs, agentfake.Verdict{Item: it.Key, Word: w, Evidence: "checked " + it.Key})
		}
		return vs
	}
}

// changeBook changes the catalog's book Definition, so catalog and Root
// work and members does not.
func changeBook(f *fixture) (base, start string) {
	base = f.tip("main")
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "A book has a title and an `isbn`.")})
	return base, f.commit("book has an isbn")
}

func items(r *run.Result) string {
	var parts []string
	for _, it := range r.Items {
		parts = append(parts, it.Manager+":"+it.Item+"="+it.Verdict)
	}
	return strings.Join(parts, " ")
}

func TestApplyLandsOnUnmovedTarget(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Title, ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.Outcome != run.Landed || r.State != run.StateLanded {
		t.Fatalf("outcome %s/%s: %s", r.Outcome, r.State, r.Why)
	}
	if got := f.tip("main"); got != r.Landed {
		t.Fatalf("main is at %s, the Result says it landed %s", got, r.Landed)
	}
	if !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the approved change did not land")
	}
	page := f.show("main", project.ResultsDir+"/"+r.Run+".md")
	if !strings.Contains(page, "catalog/book") || !strings.Contains(page, "# Run "+r.Run+": landed") {
		t.Fatalf("Result page:\n%s", page)
	}
	if r.Start != start || r.Base != base {
		t.Fatalf("base %s start %s", r.Base, r.Start)
	}
	if items(r) != "catalog:catalog/book=changed" {
		t.Fatalf("items %s", items(r))
	}
	for _, it := range r.Items {
		if !it.Approved || it.Checker == "" || it.Evidence == "" {
			t.Fatalf("item %+v", it)
		}
	}
	if !r.RequiredChecks.Ran || !r.RequiredChecks.WholeRepository {
		t.Fatalf("required checks %+v", r.RequiredChecks)
	}
	if f.tip(run.RunBranchPrefix+r.Run) != "" || r.Branch != "" {
		t.Fatal("the landed Run's branch was not deleted")
	}
	var notTaking []string
	for _, p := range r.NotParticipating {
		notTaking = append(notTaking, p.Package)
	}
	if strings.Join(notTaking, " ") != "members members/loans" {
		t.Fatalf("not taking part: %v", notTaking)
	}
	// The landed work is the default base of the next Run, and nothing
	// changed since.
	again := f.run(run.Request{})
	if work := strings.TrimSpace(testkit.Git(t, f.target, "rev-parse", r.Landed+"^")); again.State != run.StateNothing || again.Base != work {
		t.Fatalf("next Run %s from %s: %s", again.State, again.Base, again.Why)
	}
}

func TestCheckerRejectsAndMakerFixes(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker,
		a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Title string }\n"}, word("changed")),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Review || !strings.Contains(s.Inbox[0].Text, "no isbn") {
				return agentfake.Fail(errorf("the second start has no rejection: %+v", s.Inbox))
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Title, ISBN string }\n"}, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, a.Review(map[string]string{"catalog/book": "no isbn"}), a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the fixed work did not land")
	}
	rounds := 0
	for _, st := range r.Starts {
		if st.Manager == "catalog" && st.Role == maker {
			rounds++
		}
	}
	if rounds != 2 {
		t.Fatalf("catalog's maker started %d times", rounds)
	}
}

func TestEscalationSettledByParent(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	var question string
	var rootEnded atomic.Bool
	a.On("catalog", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			m, err := h.Escalate(ctx, "Must an isbn be unique across copies?")
			question = m.ID
			return err
		}),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Answer || s.Inbox[0].ReplyTo != question {
				return agentfake.Fail(errorf("no answer: %+v", s.Inbox))
			}
			// The answer settles the matter only once Root's start has
			// ended, which could still have changed Root's files.
			if !rootEnded.Load() {
				return agentfake.Fail(errorf("catalog started again while Root's answering start ran"))
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Title, ISBN string }\n"}, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Escalation {
				return errorf("Root got %+v", s.Inbox)
			}
			_, err := h.Answer(ctx, s.Inbox[0].ID, "Per title, as the model says nothing more: realization freedom.")
			time.Sleep(500 * time.Millisecond)
			rootEnded.Store(true)
			return err
		}),
		a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if len(r.Messages) < 2 || r.Messages[0].Kind != agentport.Escalation {
		t.Fatalf("messages %+v", r.Messages)
	}
}

// The shape of the T7 smoke's launch 3: Root has no items, so catalog starts
// at once; the change needs a file of Root's, so catalog escalates and Root
// settles it with a commit of its own. Root's settlement is checked before
// its child is approved, catalog continues from that approval, and only then
// does Root integrate catalog's approved work. Before, catalog restarted on
// the answer without Root's fix and Root's submission waited for catalog's
// approval, so neither could converge.
func TestEscalationSettledByAParentsCommit(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	readme := "# Library\n\nBooks carry an isbn.\n"
	var escalation string
	a.On("catalog", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			m, err := h.Escalate(ctx, "The isbn must be in README.md, which is Root's file.")
			escalation = m.ID
			return err
		}),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			var answered, told bool
			for _, msg := range s.Inbox {
				answered = answered || (msg.Kind == agentport.Answer && msg.ReplyTo == escalation)
				told = told || strings.Contains(msg.Text, s.Git.Base)
			}
			if !answered || !told {
				return agentfake.Fail(errorf("catalog starts again without the answer and its parent's approval %s: %+v", s.Git.Base, s.Inbox))
			}
			// A submission must hold the commit its work continues from.
			return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		},
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || !strings.Contains(s.Inbox[0].Text, "does not contain "+s.Git.Base) {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit([]string{"root"}, nil, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		if data, err := os.ReadFile(filepath.Join(w.Dir, "README.md")); err != nil || string(data) != readme {
			return a.Review(map[string]string{"catalog/book": "README.md does not name the isbn"})(ctx, s, h)
		}
		return a.Review(nil)(ctx, s, h)
	})
	a.On(".", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Escalation || !strings.Contains(s.Task, "catalog") {
				return agentfake.Fail(errorf("Root got %+v, task %q", s.Inbox, s.Task))
			}
			if _, err := h.Answer(ctx, s.Inbox[0].ID, "README.md names the isbn now; continue from my approved work."); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, map[string]string{"README.md": readme}, word("conforms"))(ctx, s, h)
		},
		a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil), a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if f.show("main", "README.md") != strings.TrimSpace(readme) || !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the settlement or the child's work did not land")
	}
	// Root's settlement was approved before catalog started again, and
	// catalog was approved before Root integrated it.
	if got := startOrder(a); got != "catalog maker, . maker, . checker, catalog maker, catalog maker, catalog checker, . maker, . checker" {
		t.Fatalf("order: %s", got)
	}
}

// startOrder lists the starts in the order they began.
func startOrder(a *agentfake.Agents) string {
	var order []string
	for _, st := range a.Starts() {
		order = append(order, st.Manager.ID+" "+string(st.Role))
	}
	return strings.Join(order, ", ")
}

// pushWork returns a maker Step that pushes a work commit, no submission.
func pushWork(a *agentfake.Agents, files map[string]string) agentfake.Step {
	return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		for f, content := range files {
			if err := w.Write(f, content); err != nil {
				return agentfake.Fail(err)
			}
		}
		if err := w.Commit("work in progress"); err != nil {
			return agentfake.Fail(err)
		}
		if err := w.Push(); err != nil {
			return agentfake.Fail(err)
		}
		return agentfake.Done()
	}
}

// readmeReview approves only a submission whose README.md is readme.
func readmeReview(a *agentfake.Agents, readme string) agentfake.Step {
	return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		if data, err := os.ReadFile(filepath.Join(w.Dir, "README.md")); err != nil || string(data) != readme {
			return a.Review(map[string]string{"catalog/book": "README.md does not name the isbn"})(ctx, s, h)
		}
		return a.Review(nil)(ctx, s, h)
	}
}

// A child that escalates and submits in the same start, and a parent that
// answers and pushes only a work commit: the parent is still asked for a
// submission, and the child's submission from before the settlement is never
// reviewed.
func TestASettlementOutlivesStaleWork(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	readme := "# Library\n\nBooks carry an isbn.\n"
	a.On("catalog", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if _, err := h.Escalate(ctx, "The isbn must be in README.md, which is Root's file."); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		},
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 2 || !strings.Contains(s.Inbox[1].Text, s.Git.Base) {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit([]string{"root"}, nil, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, readmeReview(a, readme))
	a.On(".", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if _, err := h.Answer(ctx, s.Inbox[0].ID, "README.md names the isbn now."); err != nil {
				return agentfake.Fail(err)
			}
			return pushWork(a, map[string]string{"README.md": readme})(ctx, s, h)
		},
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || !strings.Contains(s.Inbox[0].Text, "which is no submission") {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit(nil, nil, word("conforms"))(ctx, s, h)
		},
		a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil), a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if got := startOrder(a); got != "catalog maker, . maker, . maker, . checker, catalog maker, catalog checker, . maker, . checker" {
		t.Fatalf("order: %s", got)
	}
}

// An approved model-problem verdict escalates as the tool does: the parent's
// settlement with a commit is checked first, and the child continues from it.
// loans escalates to members, whose settling commit names a model problem and
// so escalates to Root. Root and then members answer without a change; loans
// still continues from members' approved settlement.
func TestAnAnswerKeepsTheParentsSettlement(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{
		m + "members/member.md":     def("std/Concept", "Member", "A person with a card."),
		m + "members/loans/loan.md": def("std/Concept", "Loan", "A [[members/member]] borrows with a card."),
	})
	f.commit("cards")
	a := f.agents
	var escalation string
	a.On("members/loans", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			m, err := h.Escalate(ctx, "The card type is in src/members/member.go, members' file.")
			escalation = m.ID
			return err
		}),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			var told bool
			for _, msg := range s.Inbox {
				told = told || (strings.Contains(msg.Text, s.Git.Base) && strings.Contains(msg.Text, escalation))
			}
			if !told {
				return agentfake.Fail(errorf("loans starts again from %s without members' settlement: %+v", s.Git.Base, s.Inbox))
			}
			return a.Submit([]string{"managers/members"}, map[string]string{"src/members/loans/loan.go": "package loans\n\nconst Card = true\n"}, word("changed"))(ctx, s, h)
		})
	a.On("members/loans", checker, a.Review(nil))
	answer := func(ctx context.Context, s agentport.Start, h agentport.Host) error {
		for _, msg := range s.Inbox {
			if msg.Kind == agentport.Escalation {
				_, err := h.Answer(ctx, msg.ID, "Realization freedom; nothing more to change.")
				return err
			}
		}
		return errorf("%s got %+v", s.Manager.ID, s.Inbox)
	}
	a.On("members", maker,
		a.Submit(nil, nil, word("conforms")),
		a.Submit(nil, map[string]string{"src/members/member.go": "package members\n\ntype Card string\n"}, word("model-problem")),
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Answer {
				return errorf("members got %+v", s.Inbox)
			}
			_, err := h.Answer(ctx, escalation, "Realization freedom; nothing more to change.")
			return err
		}),
		a.Submit([]string{"managers/members.loans"}, nil, word("conforms")))
	a.On("members", checker, a.Review(nil), a.Review(nil), a.Review(nil))
	a.On(".", maker, agentfake.Call(answer), a.Submit([]string{"managers/members"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show("main", "src/members/member.go"), "Card") || !strings.Contains(f.show("main", "src/members/loans/loan.go"), "Card") {
		t.Fatal("the settlement or the child's work did not land")
	}
}

func TestAModelProblemSettledByAParentsCommit(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	readme := "# Library\n\nBooks carry an isbn.\n"
	a.On("catalog", maker,
		a.Submit(nil, nil, word("model-problem")),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 2 || !strings.Contains(s.Inbox[1].Text, s.Git.Base) {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit([]string{"root"}, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, a.Review(nil), readmeReview(a, readme))
	a.On(".", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Escalation {
				return agentfake.Fail(errorf("Root got %+v", s.Inbox))
			}
			if _, err := h.Answer(ctx, s.Inbox[0].ID, "README.md names the isbn now."); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, map[string]string{"README.md": readme}, word("conforms"))(ctx, s, h)
		},
		a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil), a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if got := startOrder(a); got != "catalog maker, catalog checker, . maker, . checker, catalog maker, catalog checker, . maker, . checker" {
		t.Fatalf("order: %s", got)
	}
}

func TestModelDefectEndsTheRunAtRoot(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
		_, err := h.Escalate(ctx, "book contradicts the loan limit")
		return err
	}))
	a.On(".", maker, agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
		_, err := h.Escalate(ctx, "The model contradicts itself on books. Recommendation: say which rule wins.")
		return err
	}))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateModelDefect || r.Outcome != run.NotLanded {
		t.Fatalf("%s/%s: %s", r.Outcome, r.State, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("a model defect changed the target")
	}
	if len(r.OwnerDecisions) != 1 || r.OwnerDecisions[0].Recommendation != "say which rule wins." {
		t.Fatalf("decisions %+v", r.OwnerDecisions)
	}
	if kept := f.tip(run.RunBranchPrefix + r.Run); kept == "" || !strings.Contains(f.show(kept, run.ResultFile(r.Run)), "model-defect") {
		t.Fatal("the Result is not kept on the Run branch")
	}
}

// holding returns a Step that fails unless the start's clone holds every
// file, then goes on with next.
func (f *fixture) holding(files []string, next agentfake.Step) agentfake.Step {
	return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := f.agents.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(w.Dir, filepath.FromSlash(file))); err != nil {
				return agentfake.Fail(errorf("%s %s starts without %s from base %s", s.Manager.ID, s.Role, file, s.Git.Base))
			}
		}
		return next(ctx, s, h)
	}
}

// The Root rule changes, so every Manager has items and works parent first
// (concept 2.0 §6.2): Root submits its own files, members and catalog start
// from Root's approval, loans from members' approval, and the approved work
// merges upward again.
func TestParentsWorkFirstAndMergeChildrenUpToRoot(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{m + "loan-limit.md": def("std/Rule", "Loan limit", "At most `3` open loans.")})
	f.commit("limit 3")
	a := f.agents
	rootPage := table + "| README.md | [[project]] | Overview |\n| shared.txt | [[project]] | Shared |\n"
	membersPage := "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n| src/members/shared.txt | [[members/member]] | Shared |\n"
	a.On(".", maker,
		a.Submit(nil, map[string]string{"shared.txt": "limit 3\n", project.RootPage: rootPage}, word("changed")),
		a.Submit([]string{"managers/members", "managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil), a.Review(nil))
	a.On("members", maker,
		f.holding([]string{"shared.txt"}, a.Submit(nil, map[string]string{"src/members/shared.txt": "x\n", project.RegistryDir + "/members.md": membersPage}, word("conforms"))),
		a.Submit([]string{"managers/members.loans"}, nil, word("conforms")))
	a.On("members", checker, a.Review(nil), a.Review(nil))
	a.On("members/loans", maker, f.holding([]string{"shared.txt", "src/members/shared.txt"},
		a.Submit(nil, map[string]string{"src/members/loans/loan.go": "package loans\n\nconst Limit = 3\n"}, word("changed"))))
	a.On("members/loans", checker, a.Review(nil))
	a.On("catalog", maker, f.holding([]string{"shared.txt"}, a.Submit(nil, nil, word("conforms"))))
	a.On("catalog", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show("main", "src/members/loans/loan.go"), "Limit = 3") || !strings.Contains(f.show("main", "src/members/shared.txt"), "x") {
		t.Fatal("the children's work did not land")
	}
	// No child started before its parent's first approval.
	first := map[string]int{}
	for i, st := range r.Starts {
		if _, ok := first[st.Manager+" "+string(st.Role)]; !ok {
			first[st.Manager+" "+string(st.Role)] = i
		}
	}
	for _, pair := range [][2]string{{".", "members"}, {".", "catalog"}, {"members", "members/loans"}} {
		if first[pair[0]+" checker"] > first[pair[1]+" maker"] {
			t.Fatalf("%s started before %s's approval: %v", pair[1], pair[0], first)
		}
	}
	want := map[string]bool{"members/loans:loan-limit": true, "members:loan-limit": true, "catalog:loan-limit": true, ".:loan-limit": true}
	for _, it := range r.Items {
		delete(want, it.Manager+":"+it.Item)
	}
	if len(want) > 0 {
		t.Fatalf("missing items %v in %s", want, items(r))
	}
}

// Root has no items when only the book changes: it does not gate catalog,
// which starts at once from the Run's start.
func TestParentWithoutItemsDoesNotGate(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		for _, st := range a.Starts() {
			if st.Manager.ID == "." {
				return agentfake.Fail(errorf("Root started before catalog"))
			}
		}
		if s.Git.Base != start {
			return agentfake.Fail(errorf("catalog starts from %s, not the Run's start %s", s.Git.Base, start))
		}
		return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
	})
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		if len(s.Checklist) != 0 {
			return agentfake.Fail(errorf("Root has items %v", s.Checklist))
		}
		return a.Submit([]string{"managers/catalog"}, nil, word("conforms"))(ctx, s, h)
	})
	a.On(".", checker, a.Review(nil))
	if r := f.run(run.Request{Base: base}); r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

func TestParentMustMergeChildApproval(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker,
		a.Submit(nil, nil, word("conforms")),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || !strings.Contains(s.Inbox[0].Text, "does not contain catalog's approved work") {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit([]string{"managers/catalog"}, nil, word("conforms"))(ctx, s, h)
		})
	a.On(".", checker, a.Review(nil))
	if r := f.run(run.Request{Base: base}); r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// moveDuring returns a Step that pushes a commit to the target before it
// approves, as a colleague would while the Run works.
func (f *fixture) approveAfter(move map[string]string) agentfake.Step {
	return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		f.pull()
		f.write(move)
		f.commit("a colleague's change")
		return f.agents.Review(nil)(ctx, s, h)
	}
}

func TestLandingMergesAMovedTarget(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, f.approveAfter(map[string]string{"src/members/member.go": "package members\n\n// Member is a person.\n"}))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show("main", "src/members/member.go"), "a person") || !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the merged state did not land")
	}
	if r.RequiredChecks.Commit == "" || !r.RequiredChecks.Ran {
		t.Fatalf("checks on the merged state %+v", r.RequiredChecks)
	}
	if r.Start != start || !strings.Contains(r.Why, "moved") {
		t.Fatalf("why %q", r.Why)
	}
}

func TestLandingRefusesAMovedTargetThatTouchesTheImpact(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, f.approveAfter(map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Name string }\n"}))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || r.Outcome != run.NotLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(r.Why, "src/catalog/book.go") {
		t.Fatalf("why %q", r.Why)
	}
	kept := f.tip(run.RunBranchPrefix + r.Run)
	if kept == "" || !strings.Contains(f.show(kept, "src/catalog/book.go"), "ISBN") {
		t.Fatal("the approved work is not kept on the Run branch")
	}
	if strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the work landed")
	}
}

func TestLandingRefusesAFailingMergedState(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	// The colleague's change is registered and outside the impact, but it
	// breaks the required check.
	a.On(".", checker, f.approveAfter(map[string]string{
		"src/members/FAIL":                  "x\n",
		project.RegistryDir + "/members.md": "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n| src/members/FAIL | [[members/member]] | Marker |\n",
	}))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "required checks fail") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// The host compiles the state it lands: a merge of two states that each
// compile, but together give two Packages the same pattern, does not land.
func TestLandingRefusesAMergedStateThatDoesNotCompile(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	catalogPage := "---\npaths: [src/catalog/**, src/shared/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n"
	a.On("catalog", maker, a.Submit(nil, map[string]string{
		"src/catalog/book.go":               "package catalog\n\ntype Book struct{ ISBN string }\n",
		project.RegistryDir + "/catalog.md": catalogPage,
	}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	// The colleague's change compiles alone and is outside the impact.
	a.On(".", checker, f.approveAfter(map[string]string{
		"src/shared/x.go":                   "package shared\n",
		project.RegistryDir + "/members.md": "---\npaths: [src/members/**, src/shared/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n| src/shared/x.go | [[members/member]] | Shared |\n",
	}))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "the merged state does not compile") || !strings.Contains(r.Why, "src/shared/**") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatal("the work landed")
	}
}

func TestVerifyWithNoChangesStartsNothing(t *testing.T) {
	f := newFixture(t)
	start := f.tip("main")
	r := f.run(run.Request{Mode: agentport.Verify, Base: start})
	if r.State != run.StateNothing || len(r.Starts) != 0 {
		t.Fatalf("%s with %d starts", r.State, len(r.Starts))
	}
}

func TestVerifyGivesVerdictsWithoutChanges(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", checker, a.Verdicts(word("conforms")))
	a.On(".", checker, a.Verdicts(word("conforms")))
	r := f.run(run.Request{Mode: agentport.Verify, Base: base})
	if r.State != run.StateVerified || r.Outcome != run.NotLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("verify changed the target")
	}
	for _, it := range r.Items {
		if !it.Approved || it.Verdict != "conforms" {
			t.Fatalf("item %+v", it)
		}
	}
	for _, st := range r.Starts {
		if st.Role != checker {
			t.Fatalf("verify started a %s", st.Role)
		}
	}
	// The required checks ran on the whole repository, as in every Run.
	if rc := r.RequiredChecks; !rc.Ran || !rc.WholeRepository || rc.Commit == "" || len(rc.Checks) != 1 {
		t.Fatalf("required checks %+v", rc)
	}
}

func TestVerifyReportsFailingRequiredChecks(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{
		m + "catalog/book.md":               def("std/Concept", "Book", "A book has a title and an `isbn`."),
		"src/catalog/FAIL":                  "x\n",
		project.RegistryDir + "/catalog.md": "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n| src/catalog/FAIL | [[catalog/book]] | Marker |\n",
	})
	f.commit("book has an isbn, and the checks fail")
	a := f.agents
	a.On("catalog", checker, a.Verdicts(word("conforms")))
	a.On(".", checker, a.Verdicts(word("conforms")))
	r := f.run(run.Request{Mode: agentport.Verify, Base: base})
	if r.State != run.StateFindings || !strings.Contains(r.Why, "required checks that did not pass") || len(r.Findings) != 0 {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if rc := r.RequiredChecks; rc.Ran || rc.Commit == "" || len(rc.Checks) != 1 || rc.Checks[0].Passed {
		t.Fatalf("required checks %+v", rc)
	}
	if !strings.Contains(string(run.Page(r)), "and did not pass") {
		t.Fatal("the Result page does not say the checks did not pass")
	}
}

func TestApplyWithFindings(t *testing.T) {
	f := newFixture(t)
	start := f.tip("main")
	findings, ds := analysis.ParseFindings("findings.md", []byte("# Findings\n\n## [[catalog/book]]\n\nObserved: no title field.\n\nExpected: a title.\n"))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	a := f.agents
	a.On("catalog", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		if len(s.Checklist) != 1 || s.Checklist[0].Part != agentport.Finding || !strings.Contains(s.Checklist[0].Why, "no title field") {
			return agentfake.Fail(errorf("checklist %+v", s.Checklist))
		}
		return a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Title string }\n"}, word("changed"))(ctx, s, h)
	})
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: start, Findings: findings})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if len(r.Items) != 1 || r.Items[0].Part != agentport.Finding {
		t.Fatalf("items %+v", r.Items)
	}
}

func TestBudgetStopsTheRun(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	r := f.run(run.Request{Base: base, Limits: run.Limits{Deadline: time.Now().Add(time.Minute), Starts: 1, Rounds: 4, Start: time.Minute}})
	if r.State != run.StateIncomplete || !strings.Contains(r.Why, "budget") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("an incomplete Run changed the target")
	}
	for _, it := range r.Items {
		if it.Approved {
			t.Fatalf("an incomplete Run reports %s approved", it.Item)
		}
	}
}

func TestDeadlineReachesTheAgent(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	f.agents.On("catalog", maker, agentfake.Hang())
	began := time.Now()
	r := f.run(run.Request{Base: base, Limits: run.Limits{Deadline: time.Now().Add(3 * time.Second), Starts: 10, Rounds: 4, Start: time.Minute}})
	if r.State != run.StateIncomplete || !strings.Contains(r.Why, "deadline") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if time.Since(began) > time.Minute {
		t.Fatal("the deadline did not stop the agent")
	}
	if len(r.Starts) != 1 || r.Starts[0].Reason != agentport.DeadlineReached {
		t.Fatalf("starts %+v", r.Starts)
	}
	if f.tip("main") != start {
		t.Fatal("the target changed")
	}
}

func TestNoProgressStops(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	idle := agentfake.Call(func(context.Context, agentport.Start, agentport.Host) error { return nil })
	f.agents.On("catalog", maker, idle, idle)
	r := f.run(run.Request{Base: base})
	if r.State != run.StateIncomplete || !strings.Contains(r.Why, "no progress") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

func TestInitialGate(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{"unregistered.txt": "x\n"})
	f.commit("an unregistered file")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateCheckFailed || len(r.Starts) != 0 || len(f.remote.started) != 0 {
		t.Fatalf("%s with %d starts: %s", r.State, len(r.Starts), r.Why)
	}
}

func TestDefaultBaseNeedsALandedRun(t *testing.T) {
	f := newFixture(t)
	_, err := f.engine.Run(context.Background(), run.Request{Mode: agentport.Apply, Target: "main"})
	if err == nil || !strings.Contains(err.Error(), "--base") {
		t.Fatalf("err %v", err)
	}
}

func TestHostToolsRefuseOutsideTheTree(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		if _, err := h.Assign(ctx, "members", "work"); err == nil {
			return agentfake.Fail(errorf("a leaf assigned work"))
		}
		if _, err := h.Answer(ctx, "m99", "x"); err == nil {
			return agentfake.Fail(errorf("an answer to nothing was taken"))
		}
		if _, err := h.Note(ctx, strings.Repeat("x", run.MaxNoteText+1)); err == nil {
			return agentfake.Fail(errorf("an oversized Note was taken"))
		}
		if _, err := h.Note(ctx, "the book file has no tests"); err != nil {
			return agentfake.Fail(err)
		}
		return a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed"))(ctx, s, h)
	})
	a.On("catalog", checker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		if _, err := h.Ask(ctx, "may I?"); err == nil {
			return agentfake.Fail(errorf("a checker asked"))
		}
		// A checker is fresh: it leaves Notes but does not read its maker's.
		if len(s.Tools) != 1 || s.Tools[0] != agentport.ToolNote {
			return agentfake.Fail(errorf("checker tools %v", s.Tools))
		}
		if page, err := h.Notes(ctx, ""); err == nil {
			return agentfake.Fail(errorf("a checker read the Notes: %+v", page))
		}
		if _, err := h.Note(ctx, "book.go has no doc comment"); err != nil {
			return agentfake.Fail(err)
		}
		return a.Review(nil)(ctx, s, h)
	})
	a.On(".", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		if _, err := h.Ask(ctx, "anyone?"); err == nil {
			return agentfake.Fail(errorf("Root asked a parent"))
		}
		if page, err := h.Notes(ctx, ""); err != nil || len(page.Notes) != 2 {
			return agentfake.Fail(errorf("notes %+v %v", page, err))
		}
		return a.Submit([]string{"managers/catalog"}, nil, word("conforms"))(ctx, s, h)
	})
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || len(r.Notes) != 2 {
		t.Fatalf("%s with %d notes: %s", r.State, len(r.Notes), r.Why)
	}
}

// A target branch checked out in a working tree is never updated: Git
// refuses the push, and the work stays on its Run branch.
func TestLandingNeverTouchesAWorkingTree(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	host, err := rungit.Open(context.Background(), filepath.Join(testkit.TempDir(t), "host"), f.src)
	if err != nil {
		t.Fatal(err)
	}
	f.engine.Git = host
	// Even where Git would update the working tree itself, the host does
	// not push to a checked-out branch.
	testkit.Git(t, f.src, "config", "receive.denyCurrentBranch", "updateInstead")
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ ISBN string }\n"}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker, a.Submit([]string{"managers/catalog"}, nil, word("conforms")))
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "refused") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if head := testkit.Git(t, f.src, "rev-parse", "HEAD"); head != start {
		t.Fatalf("HEAD moved to %s", head)
	}
	if status := testkit.Git(t, f.src, "status", "--porcelain"); status != "" {
		t.Fatalf("the working tree changed:\n%s", status)
	}
	if testkit.Git(t, f.src, "rev-parse", "--verify", "--quiet", "refs/heads/"+run.RunBranchPrefix+r.Run) == "" {
		t.Fatal("the work is not on its Run branch")
	}
}

// A verify checker's deviates verdict becomes a Finding, and the Findings
// page it writes is the input of apply --findings.
func TestVerifyFindingsFeedApply(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", checker, a.Verdicts(func(s agentport.Start) []agentfake.Verdict {
		return []agentfake.Verdict{{Item: "catalog/book", Word: "deviates", Evidence: "Observed: book.go has no isbn. Expected: a book has an `isbn`."}}
	}))
	a.On(".", checker, a.Verdicts(word("conforms")))
	r := f.run(run.Request{Mode: agentport.Verify, Base: base})
	if r.State != run.StateFindings || len(r.Findings) != 1 {
		t.Fatalf("%s with %+v: %s", r.State, r.Findings, r.Why)
	}
	got, ds := analysis.ParseFindings("findings.md", run.FindingsPage(r))
	if len(ds) > 0 || len(got) != 1 || got[0].Definition != "catalog/book" || got[0].Observed != "book.go has no isbn." || !strings.Contains(got[0].Expected, "isbn") {
		t.Fatalf("Findings page parsed as %+v %v", got, ds)
	}
}
