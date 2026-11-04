package run_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const planted = "sk-ant-oat01-planted"

// Work that holds the token never leaves the host, on any way out: refused work
// keeps only its Result, on the start.
func TestRefusedWorkThatHoldsTheTokenLeavesOnlyItsResult(t *testing.T) {
	f := newFixture(t)
	f.engine.Secrets = [][]byte{[]byte(planted)}
	base, start := changeBook(f)
	leak := f.edit(map[string]string{"debug.txt": planted + "\n"})
	f.agents.on(implementer, leak, f.edit(nil), f.edit(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateRefused {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	branch := run.RunBranchPrefix + r.Run
	if got := strings.TrimSpace(testkit.Git(t, f.target, "rev-parse", branch+"^")); got != start {
		t.Fatalf("the Result sits on %s, not on the start %s", got, start)
	}
	if f.has(branch, "debug.txt") {
		t.Fatal("the token left the host on the Run's branch")
	}
	if !strings.Contains(r.Why, "only this Result is kept") {
		t.Fatalf("the Result does not say so: %s", r.Why)
	}
}

// Nothing a session said carries the token into the Result, on any branch.
func TestTheResultHoldsNoToken(t *testing.T) {
	f := newFixture(t)
	f.engine.Secrets = [][]byte{[]byte(planted)}
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "not shown -- the environment held " + planted}, "I saw "+planted))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	page := f.show("main", run.ResultFile(r.Run))
	if strings.Contains(page, planted) || !strings.Contains(page, "[redacted]") {
		t.Fatalf("the page:\n%s", page)
	}
	if strings.Contains(r.NotTested, planted) || strings.Contains(r.Items[0].Explanation, planted) {
		t.Fatalf("the returned Result holds the token: %+v", r)
	}
}

// verify does not say verified over a file the registry does not account for, or
// over a wrong on an item with no Definition.
func TestVerifyCountsWhatTheRegistryDoesNotSettle(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{"src/catalog/notes.txt": "by hand\n"})
	f.commit("a hand edit")
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Mode: agentport.Verify, Base: base})
	if r.State != run.StateFindings || !strings.Contains(r.Why, "does not settle") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// A variant of the folder's name is the same folder in some checkouts.
func TestACaseVariantOfMarkitectsFolderIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("this file system does not tell the two names apart")
	}
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{".MARKITECT/model/catalog/book.md": "anything\n", project.RootPage: table + "| README.md | [[project]] | Overview |\n| .MARKITECT/model/catalog/book.md | [[project]] | Planted |\n"}), f.edit(nil), f.edit(nil))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateRefused || !strings.Contains(r.Why, "is the owner's") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// A repair that deletes a file and its row changes what an answer rests on, so
// the item is asked again.
func TestARepairThatDeletesALocationAsksTheItemAgain(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	page := project.RegistryDir + "/catalog.md"
	row := "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n"
	f.write(map[string]string{
		"src/catalog/extra.go":  "package catalog\n",
		page:                    row + "| src/catalog/extra.go | [[catalog/book]] | More |\n",
		m + "catalog/book.md":   def("std/Concept", "Book", "A book has a title and an `isbn`."),
		m + "members/member.md": def("std/Concept", "Member", "A person with a card."),
	})
	f.commit("two changes, with a second file for the book")
	f.agents.on(implementer,
		f.edit(map[string]string{bookFile: bookFixed, "src/members/member.go": "package members\n\ntype Member struct{ Card string }\n"}),
		f.edit(map[string]string{"src/catalog/extra.go": "", page: row, "src/members/member.go": "package members\n\ntype Member struct{ Card, Name string }\n"}))
	f.agents.on(reviewer,
		f.answer(map[string]string{"members/member": "wrong  README.md:1 -- the card is optional"}, ""),
		f.ok())
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	second := strings.Join(listed(f.agents.of(reviewer)[1].Prompt), " ")
	if !strings.Contains(second, "catalog/book") || !strings.Contains(second, "members/member") {
		t.Fatalf("the second review answers %s", second)
	}
}

// A reused name is refused before any work is done, and would otherwise lose it.
func TestARunNameWithABranchIsRefused(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "spec problem -- it needs the owner"}, ""))
	if kept := f.run(run.Request{Base: base, Run: "same"}); kept.State != run.StateSpecProblem {
		t.Fatalf("%s: %s", kept.State, kept.Why)
	}
	if _, err := f.engine.Run(t.Context(), run.Request{Mode: agentport.Apply, Target: "main", Base: base, Run: "same"}); err == nil || !strings.Contains(err.Error(), "already has its branch") {
		t.Fatalf("the reused name: %v", err)
	}
}

// Findings need no landing before them: their items do not depend on what changed.
func TestApplyFindingsNeedsNoEarlierLanding(t *testing.T) {
	f := newFixture(t)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Findings: []analysis.Finding{{Definition: "catalog/book", Observed: "no isbn", Expected: "an isbn"}}})
	if r.State != run.StateLanded || answerOf(r, "finding 1 catalog/book") != run.AnswerOK {
		t.Fatalf("%s: %s; %+v", r.State, r.Why, r.Items)
	}
}
