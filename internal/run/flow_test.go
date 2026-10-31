package run_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agentport/agentfake"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

const isbn = "package catalog\n\ntype Book struct{ ISBN string }\n"

// noop is a start that does nothing and finishes.
var noop = agentfake.Call(func(context.Context, agentport.Start, agentport.Host) error { return nil })

// A child asks and submits in one turn; Root answers and submits before the
// child is approved. The early submission is not reviewed; once the child is
// approved and its answer changed nothing, the host merges for Root.
func TestAskAndSubmitWithoutWaiting(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	answered := make(chan struct{})
	a.On("catalog", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if _, err := h.Ask(ctx, "Is the isbn optional?"); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		},
		noop)
	a.On("catalog", checker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		<-answered
		return a.Review(nil)(ctx, s, h)
	})
	a.On(".", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			defer close(answered)
			if len(s.Inbox) != 1 {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			if _, err := h.Answer(ctx, s.Inbox[0].ID, "No, it is required."); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, nil, word("conforms"))(ctx, s, h)
		})
	a.On(".", checker, a.Review(nil))
	if r := f.run(run.Request{Base: base}); r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// The shape of the comparison's transition-gap launch: Root, without items,
// answers a question of members with a submission while members is approved
// for its own files and its child loans still works. Root is not told to
// merge members' approval, which members' next one replaces once it merges
// loans' work; the host merges for Root when all of it is approved.
func TestAParentDoesNotMergeWorkAGrandchildStillChanges(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{m + "members/member.md": def("std/Concept", "Member", "A person with a card.")})
	f.commit("members carry a card")
	a := f.agents
	loansWork, rootDone := make(chan struct{}), make(chan struct{})
	a.On("members", maker,
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if _, err := h.Ask(ctx, "Who issues the card?"); err != nil {
				return agentfake.Fail(err)
			}
			return a.Submit(nil, map[string]string{"src/members/member.go": "package members\n\ntype Member struct{ Card string }\n"}, word("changed"))(ctx, s, h)
		},
		noop,
		a.Submit([]string{"managers/members.loans"}, nil, word("changed")))
	a.On("members", checker, a.Review(nil), a.Review(nil))
	a.On("members/loans", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		close(loansWork)
		<-rootDone
		return a.Submit(nil, map[string]string{"src/members/loans/loan.go": "package loans\n\n// A Member with a card borrows.\n"}, word("changed"))(ctx, s, h)
	})
	a.On("members/loans", checker, a.Review(nil))
	a.On(".", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		defer close(rootDone)
		<-loansWork
		if _, err := h.Answer(ctx, s.Inbox[0].ID, "The library issues it."); err != nil {
			return agentfake.Fail(err)
		}
		return a.Submit(nil, map[string]string{"README.md": "# Library\n\nThe library issues member cards.\n"}, word("conforms"))(ctx, s, h)
	})
	a.On(".", checker, a.Review(nil))

	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	for _, msg := range r.Messages {
		if msg.To == "." && strings.Contains(msg.Text, "does not contain") {
			t.Fatalf("Root was told to merge: %s", msg.Text)
		}
	}
	if len(r.Integrations) != 1 || r.Integrations[0].Submission == "" {
		t.Fatalf("integrations %+v", r.Integrations)
	}
	if !strings.Contains(f.show("main", "src/members/loans/loan.go"), "with a card") || !strings.Contains(f.show("main", "README.md"), "issues member cards") {
		t.Fatal("the work did not land")
	}
}

// A parent that does not answer is reminded, once; the child waits without
// blocking anyone.
func TestUnansweredQuestionIsRemindedOnce(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	var question string
	a.On("catalog", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			m, err := h.Ask(ctx, "Is the isbn optional?")
			question = m.ID
			return err
		}),
		a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	a.On(".", maker,
		// Root submits early and leaves the question open.
		a.Submit(nil, nil, word("conforms")),
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			if len(s.Inbox) != 1 || !strings.Contains(s.Inbox[0].Text, question) {
				return errorf("no reminder: %+v", s.Inbox)
			}
			_, err := h.Answer(ctx, question, "Required.")
			return err
		}))
	a.On(".", checker, a.Review(nil))
	if r := f.run(run.Request{Base: base}); r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// A parent's checker rejects the host's merge of its child's approved work;
// the parent's maker sends it back with assign, and the child's new approval
// reaches the parent again through the host.
func TestParentAssignsACorrection(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker,
		a.Submit(nil, map[string]string{"src/catalog/book.go": "package catalog\n\ntype Book struct{ Isbn string }\n"}, word("changed")),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Assignment || s.Inbox[0].From != "." {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, a.Review(nil), a.Review(nil))
	a.On(".", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Review {
				return errorf("inbox %+v", s.Inbox)
			}
			_, err := h.Assign(ctx, "catalog", "Name the field ISBN, as the project spells it.")
			return err
		}))
	// A submission without verdicts may be rejected without a disputed item.
	a.On(".", checker, a.Review(map[string]string{"-": "the field is spelled Isbn"}), a.Review(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || !strings.Contains(f.show("main", "src/catalog/book.go"), "ISBN") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// An approved model-problem verdict escalates to the parent, which settles
// it; the child then submits again.
func TestApprovedModelProblemEscalates(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	a.On("catalog", maker,
		a.Submit(nil, nil, word("model-problem")),
		func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Answer {
				return agentfake.Fail(errorf("inbox %+v", s.Inbox))
			}
			return a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))(ctx, s, h)
		})
	a.On("catalog", checker, a.Review(nil), a.Review(nil))
	a.On(".", maker,
		agentfake.Call(func(ctx context.Context, s agentport.Start, h agentport.Host) error {
			if len(s.Inbox) != 1 || s.Inbox[0].Kind != agentport.Escalation || s.Inbox[0].From != "catalog" {
				return errorf("inbox %+v", s.Inbox)
			}
			_, err := h.Answer(ctx, s.Inbox[0].ID, "The isbn is a plain string: realization freedom.")
			return err
		}))
	a.On(".", checker, a.Review(nil))
	if r := f.run(run.Request{Base: base}); r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// Root's approved model-problem verdict is a model defect.
func TestRootModelProblemIsAModelDefect(t *testing.T) {
	f := newFixture(t)
	start := f.tip("main")
	findings, _ := analysis.ParseFindings("f.md", []byte("# Findings\n\n## [[loan-limit]]\n\nObserved: five loans.\n\nExpected: the limit holds.\n"))
	a := f.agents
	// The Finding is Root's; members and catalog do not work.
	a.On(".", maker, a.Submit(nil, nil, func(s agentport.Start) []agentfake.Verdict {
		return []agentfake.Verdict{{Item: s.Checklist[0].Key, Word: "model-problem", Evidence: "the limit contradicts the project anchor. Recommendation: pick one."}}
	}))
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: start, Findings: findings})
	if r.State != run.StateModelDefect || len(r.OwnerDecisions) != 1 || r.OwnerDecisions[0].Recommendation != "pick one." {
		t.Fatalf("%s %+v: %s", r.State, r.OwnerDecisions, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("a model defect landed")
	}
}

// A token a maker commits and removes again still sits in the history that
// landing would push, so the work does not land.
func TestCredentialInHistoryDoesNotLand(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
		w, err := a.Clone(ctx, s)
		if err != nil {
			return agentfake.Fail(err)
		}
		for _, step := range []func() error{
			func() error { return w.Write("src/catalog/book.go", "package catalog\n\n// "+s.Git.Token+"\n") },
			func() error { return w.Commit("work") },
			func() error { return w.Write("src/catalog/book.go", isbn) },
			func() error {
				return w.Commit(agentfake.SubmissionMessage(s.Manager.ID, "Submit", word("changed")(s)))
			},
			w.Push,
		} {
			if err := step(); err != nil {
				return agentfake.Fail(err)
			}
		}
		return agentfake.Done()
	})
	a.On("catalog", checker, a.Review(nil))
	a.On(".", checker, a.Review(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "Git token") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if f.tip("main") != start {
		t.Fatal("the work landed")
	}
}

// Usage adds up once, also in the JSON Result.
func TestUsageIsCountedOnce(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	a := f.agents
	used := func(step agentfake.Step) agentfake.Step {
		return func(ctx context.Context, s agentport.Start, h agentport.Host) agentport.End {
			end := step(ctx, s, h)
			end.Usage = agentport.Usage{InputTokens: 100, OutputTokens: 10}
			return end
		}
	}
	a.On("catalog", maker, used(a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed"))))
	a.On("catalog", checker, used(a.Review(nil)))
	a.On(".", checker, used(a.Review(nil)))
	r := f.run(run.Request{Base: base})
	var back run.Result
	data, _ := json.Marshal(r)
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Usage.InputTokens != 300 || back.Usage.OutputTokens != 30 {
		t.Fatalf("usage %+v", back.Usage)
	}
	if !strings.Contains(f.show("main", run.ResultFile(r.Run)), "300 in, 30 out") {
		t.Fatal("the page counts other tokens")
	}
}

// A Run stopped after a child's approval says that this approved work is not
// on its Run branch and lands nothing.
func TestStoppedRunNamesApprovedWorkItDoesNotKeep(t *testing.T) {
	f := newFixture(t)
	// The hooks refuse the host's merge for Root, so Root's maker would
	// have to merge, and the budget stops the Run first.
	failMarker(f)
	base, start := changeBook(f)
	a := f.agents
	a.On("catalog", maker, a.Submit(nil, map[string]string{"src/catalog/book.go": isbn}, word("changed")))
	a.On("catalog", checker, a.Review(nil))
	r := f.run(run.Request{Base: base, Limits: run.Limits{Deadline: time.Now().Add(time.Minute), Starts: 2, Rounds: 4, Start: time.Minute}})
	if r.State != run.StateIncomplete || r.Outcome != run.NotLanded || f.tip("main") != start {
		t.Fatalf("%s/%s: %s", r.Outcome, r.State, r.Why)
	}
	if len(r.Unkept) != 1 || !strings.Contains(r.Unkept[0], "catalog") {
		t.Fatalf("unkept %v", r.Unkept)
	}
}
