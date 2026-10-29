package run_test

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// changeBookAgain changes the book once more on the target's tip.
func changeBookAgain(f *fixture, text string) {
	f.pull()
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", text)})
	f.commit(text)
}

// The full-check count across Runs, from the target's landing commits and
// Run branches (concept 2.0 §6.1): a landing counts; a verify --full that
// found every item conforming starts the count again at the commit it
// examined; the next landing carries that forward, so the count holds after
// the verify's Run branch is deleted.
func TestTheFullCheckCountAcrossRuns(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	scriptHappyPath(f, nil)
	first := f.run(run.Request{Base: base})
	if fc := first.FullCheck; first.State != run.StateLanded || fc.Landed != 1 || fc.Since != "" || fc.Due {
		t.Fatalf("first landing %s: %+v", first.State, fc)
	}
	if page := f.show("main", run.ResultFile(first.Run)); !strings.Contains(page, "1 change landed since the first landing") {
		t.Fatalf("first page:\n%s", page)
	}

	a := f.agents
	for _, m := range []string{".", "members", "members/loans", "catalog"} {
		a.On(m, checker, a.Verdicts(word("conforms")))
	}
	check := f.run(run.Request{Mode: agentport.Verify, Full: true})
	if fc := check.FullCheck; check.State != run.StateVerified || fc.Landed != 0 || fc.Since != check.Start {
		t.Fatalf("verify --full %s: %+v", check.State, fc)
	}

	changeBookAgain(f, "A book has a title, an `isbn` and an author.")
	scriptHappyPath(f, nil)
	second := f.run(run.Request{})
	if fc := second.FullCheck; second.State != run.StateLanded || fc.Landed != 1 || fc.Since != check.Start {
		t.Fatalf("second landing %s: %+v", second.State, fc)
	}
	if msg := testkit.Git(t, f.target, "log", "-1", "--format=%B", "main"); !strings.Contains(msg, "Run-Full-Check: "+check.Start) {
		t.Fatalf("the landing does not carry the full check forward:\n%s", msg)
	}

	// The verify's Run branch goes; the landing carried its reset.
	testkit.Git(t, f.target, "branch", "-D", run.RunBranchPrefix+check.Run)
	changeBookAgain(f, "A book has a title, an `isbn`, an author and a year.")
	scriptHappyPath(f, nil)
	third := f.run(run.Request{})
	if fc := third.FullCheck; third.State != run.StateLanded || fc.Landed != 2 || fc.Since != check.Start {
		t.Fatalf("third landing %s: %+v", third.State, fc)
	}
}
