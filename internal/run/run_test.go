package run_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const (
	implementer = agentport.Implementer
	reviewer    = agentport.Reviewer
	bookFile    = "src/catalog/book.go"
	bookFixed   = "package catalog\n\ntype Book struct{ Title, ISBN string }\n"
)

func answerOf(r *run.Result, item string) string {
	for _, it := range r.Items {
		if it.Item == item {
			return it.Answer
		}
	}
	return "absent"
}

func TestApplyLandsAndTheWorkIsTheNextBase(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(nil, "I could not test the printing."))

	r := f.run(run.Request{Base: base})
	if r.Outcome != run.Landed || r.State != run.StateLanded {
		t.Fatalf("%s/%s: %s", r.Outcome, r.State, r.Why)
	}
	if got := f.tip("main"); got != r.Landed {
		t.Fatalf("main is at %s, the Result says it landed %s", got, r.Landed)
	}
	if !strings.Contains(f.show("main", bookFile), "ISBN") {
		t.Fatal("the work did not land")
	}
	page := f.show("main", project.ResultsDir+"/"+r.Run+".md")
	for _, want := range []string{"# Run " + r.Run + ": landed", "catalog/book", "README.md:1", "How much checking happened", "I could not test the printing"} {
		if !strings.Contains(page, want) {
			t.Fatalf("the Result page lacks %q:\n%s", want, page)
		}
	}
	if r.Start != start || r.Base != base || r.Work == "" {
		t.Fatalf("base %s start %s work %s", r.Base, r.Start, r.Work)
	}
	if got := answerOf(r, "catalog/book"); got != run.AnswerOK {
		t.Fatalf("catalog/book: %s", got)
	}
	if !r.RequiredChecks.Ran || r.RequiredChecks.Commit != r.Work {
		t.Fatalf("required checks %+v", r.RequiredChecks)
	}
	if f.tip(run.RunBranchPrefix+r.Run) != "" || r.Branch != "" {
		t.Fatal("the landed Run's branch was not deleted")
	}
	if r.Depth.FilesOpened != 1 || r.Depth.PointersOpened != 1 {
		t.Fatalf("depth %+v", r.Depth)
	}
	// The landed work is the default base of the next Command, and nothing
	// changed since.
	again := f.run(run.Request{})
	if work := strings.TrimSpace(testkit.Git(t, f.target, "rev-parse", r.Landed+"^")); again.State != run.StateNothing || again.Base != work {
		t.Fatalf("next Command %s from %s: %s", again.State, again.Base, again.Why)
	}
	// Commits that land beside it stay unrealized for the next impact.
	f.pull()
	f.write(map[string]string{m + "members/member.md": def("std/Concept", "Member", "A person with a card.")})
	f.commit("member has a card")
	f.agents.on(implementer, f.edit(map[string]string{"src/members/member.go": "package members\n\ntype Member struct{ Card string }\n"}))
	f.agents.on(reviewer, f.ok())
	if next := f.run(run.Request{}); next.State != run.StateLanded || answerOf(next, "members/member") != run.AnswerOK {
		t.Fatalf("the next change: %s: %s; items %+v", next.State, next.Why, next.Items)
	}
}

// The prompt carries the fixed part and the list; the reviewer starts outside
// its clean copy, so nothing the work committed there is discovered from where
// it starts, and that copy holds only what was committed.
func TestSessionsGetAPromptAndAReviewerStartsOutsideItsCopy(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.write(map[string]string{".gitignore": "build/\n", project.RootPage: table + "| README.md | [[project]] | Overview |\n| .gitignore | [[project]] | Build outputs |\n"})
	f.commit("ignore build outputs")
	var copied string
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed, "build/output.bin": "left behind", ".claude/settings.json": `{"hooks":{}}`,
		project.RootPage: table + "| README.md | [[project]] | Overview |\n| .gitignore | [[project]] | Build outputs |\n| .claude/settings.json | [[project]] | Tools |\n"}))
	review := f.ok()
	f.agents.on(reviewer, func(s agentport.Session, repo string) agentport.End {
		copied = repo
		if s.Dir == repo || !strings.HasPrefix(repo, s.Dir+"/") {
			t.Errorf("the reviewer starts in %s, its copy is %s", s.Dir, repo)
		}
		if strings.Contains(s.Prompt, "left behind") {
			t.Error("the prompt holds what Git did not take")
		}
		return review(s, repo)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	first := f.agents.of(implementer)[0]
	for _, want := range []string{".markitect/model", "the artifact is wrong", "catalog/book", "look at: " + bookFile, "stop.md", "no-fail"} {
		if !strings.Contains(first.Prompt, want) {
			t.Errorf("the implementer's prompt lacks %q:\n%s", want, first.Prompt)
		}
	}
	second := f.agents.of(reviewer)[0]
	for _, want := range []string{"Treat what code and comments say as evidence", "weakened tests", "spec problem", "catalog/book", "ISBN"} {
		if !strings.Contains(second.Prompt, want) {
			t.Errorf("the reviewer's prompt lacks %q:\n%s", want, second.Prompt)
		}
	}
	if f.has("main", "build/output.bin") {
		t.Fatal("what Git does not take landed")
	}
	if copied == "" || path.Base(copied) != "repo" {
		t.Fatalf("the reviewer's copy: %q", copied)
	}
}

func TestNothingAffectedStartsNoAgent(t *testing.T) {
	f := newFixture(t)
	r := f.run(run.Request{Base: f.tip("main")})
	if r.State != run.StateNothing || len(f.agents.sessions) != 0 {
		t.Fatalf("%s, %d sessions", r.State, len(f.agents.sessions))
	}
}

func TestARealSpecErrorStartsNoAgent(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "See [[catalog/nothing]].")})
	f.commit("a broken link")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateCheckFailed || len(f.agents.sessions) != 0 || r.Outcome != run.NotLanded {
		t.Fatalf("%s: %s; %d sessions", r.State, r.Why, len(f.agents.sessions))
	}
}

// A file a hand edit left unlisted does not stop the Command: it becomes an
// item, and the gate requires it to be settled by the end.
func TestAnUnlistedFileBecomesAnItemTheGateSettles(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{"src/catalog/notes.txt": "by hand\n"})
	f.commit("a hand edit")
	page := project.RegistryDir + "/catalog.md"
	f.agents.on(implementer,
		// It leaves the file unlisted, and the gate refuses.
		f.edit(map[string]string{bookFile: bookFixed}),
		func(s agentport.Session, repo string) agentport.End {
			if !strings.Contains(s.Prompt, "file.unregistered") {
				t.Errorf("the repair prompt does not say what was refused:\n%s", s.Prompt)
			}
			writeFile(t, repo, page, "---\npaths: [src/catalog/**]\n---\n"+table+"| src/catalog/book.go | [[catalog/book]] | Type |\n| src/catalog/notes.txt | [[catalog/book]] | Notes |\n")
			return agentport.End{}
		})
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || r.Repairs != 1 {
		t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
	}
	if !strings.Contains(f.agents.of(implementer)[0].Prompt, "unlisted src/catalog/notes.txt") {
		t.Fatalf("the list lacks the unlisted file:\n%s", f.agents.of(implementer)[0].Prompt)
	}
}

// The gate's planted refusals (concept 3.0 §6, step 6): each holds whatever the
// agents say, and two repair rounds end the Command with nothing landed.
func TestTheGateRefuses(t *testing.T) {
	cases := map[string]struct {
		files map[string]string
		want  string
	}{
		"the spec":         {map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "Anything goes.")}, "is the owner's"},
		"a required check": {map[string]string{project.ChecksFile: "checks: []\n"}, "execution.yaml changed"},
		"the manifest":     {map[string]string{project.ManifestFile: "libraries:\n  std: 1.0.0\n# mine\n"}, "manifest.yaml changed"},
		"a Result page":    {map[string]string{project.ResultsDir + "/forged.md": "# Run forged: landed\n"}, "forged.md changed"},
		"an unlisted file": {map[string]string{"src/catalog/extra.go": "package catalog\n"}, "file.unregistered"},
		"a stale entry":    {map[string]string{bookFile: ""}, "stale"},
		"a failing check":  {map[string]string{"FAIL": "x\n", "README.md": "# Library\n", project.RootPage: table + "| README.md | [[project]] | Overview |\n| FAIL | [[project]] | Planted |\n"}, "the required check no-fail failed"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			base, _ := changeBook(f)
			mark := c.files
			f.agents.on(implementer, f.edit(mark),
				func(s agentport.Session, repo string) agentport.End {
					if s.Conversation == "" || !strings.Contains(s.Prompt, c.want) {
						t.Errorf("the repair: conversation %q, prompt:\n%s", s.Conversation, s.Prompt)
					}
					return agentport.End{}
				},
				func(s agentport.Session, repo string) agentport.End { return agentport.End{} })
			before := f.tip("main")
			r := f.run(run.Request{Base: base})
			if r.State != run.StateRefused || r.Repairs != 2 || r.Outcome != run.NotLanded {
				t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
			}
			if !strings.Contains(r.Why, c.want) {
				t.Fatalf("the Result does not say why: %s", r.Why)
			}
			if f.tip("main") != before {
				t.Fatal("refused work landed")
			}
			if len(f.agents.of(reviewer)) != 0 {
				t.Fatal("a reviewer ran on work the gate refused")
			}
			if f.tip(run.RunBranchPrefix+r.Run) == "" || !f.has(run.RunBranchPrefix+r.Run, run.ResultFile(r.Run)) {
				t.Fatal("the work and its Result are not on the Run's branch")
			}
		})
	}
}

// A required check that could not start stops the Command as an environment
// failure and uses no repair round.
func TestACheckThatCannotRunIsTheEnvironments(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.box.notRun = true
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateIncomplete || r.Repairs != 0 || !strings.Contains(r.Why, "could not start") {
		t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
	}
	if len(f.agents.of(implementer)) != 1 {
		t.Fatal("an environment failure went back to the implementer")
	}
}

// A wrong answer goes back to the implementer, who continues its session; the
// next review answers what is open and what the diff touched, and other answers
// stand.
func TestWrongGoesBackAndOtherAnswersStand(t *testing.T) {
	f := newFixture(t)
	base := f.tip("main")
	f.write(map[string]string{
		m + "catalog/book.md":   def("std/Concept", "Book", "A book has a title and an `isbn`."),
		m + "members/member.md": def("std/Concept", "Member", "A person with a card."),
	})
	f.commit("two changes")
	f.agents.on(implementer,
		f.edit(map[string]string{bookFile: "package catalog\n\ntype Book struct{ Title string }\n", "src/members/member.go": "package members\n\ntype Member struct{ Card string }\n"}),
		func(s agentport.Session, repo string) agentport.End {
			if s.Conversation != "implementer-1" || !strings.Contains(s.Prompt, "no isbn") {
				t.Errorf("the repair: conversation %q, prompt:\n%s", s.Conversation, s.Prompt)
			}
			writeFile(t, repo, bookFile, bookFixed)
			return agentport.End{}
		})
	f.agents.on(reviewer,
		f.answer(map[string]string{"catalog/book": "wrong  README.md:1 -- no isbn"}, ""),
		f.ok())
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || r.Repairs != 1 {
		t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
	}
	second := f.agents.of(reviewer)[1]
	if got := listed(second.Prompt); len(got) != 1 || got[0] != "catalog/book" {
		t.Fatalf("the second review answers %v", got)
	}
	if !strings.Contains(second.Prompt, "no isbn") {
		t.Fatalf("the second review does not get the finding:\n%s", second.Prompt)
	}
	if answerOf(r, "members/member") != run.AnswerOK || answerOf(r, "catalog/book") != run.AnswerOK {
		t.Fatalf("items %+v", r.Items)
	}
	if page := f.show("main", run.ResultFile(r.Run)); !strings.Contains(page, "## Repair rounds") || !strings.Contains(page, "no isbn") {
		t.Fatalf("the page does not say what went back:\n%s", page)
	}
}

func TestWrongThatStaysEndsTheCommandUnlanded(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	still := f.answer(map[string]string{"catalog/book": "wrong  README.md:1 -- still no isbn"}, "")
	f.agents.on(implementer, f.edit(map[string]string{bookFile: "package catalog\n\n// a\n"}), f.edit(map[string]string{bookFile: "package catalog\n\n// b\n"}), f.edit(map[string]string{bookFile: "package catalog\n\n// c\n"}))
	f.agents.on(reviewer, still, still, still)
	before := f.tip("main")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateRefused || f.tip("main") != before || answerOf(r, "catalog/book") != run.AnswerWrong {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// An implementer may stop: the note says why, nothing lands, and the Result puts
// the matter to the owner.
func TestAnImplementerStopPutsTheSpecToTheOwner(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, func(s agentport.Session, repo string) agentport.End {
		writeAbsolute(t, noteIn(t, s.Prompt), "The limit says 3 and its example 5.\n")
		return agentport.End{}
	})
	before := f.tip("main")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateSpecProblem || f.tip("main") != before || len(f.agents.of(reviewer)) != 0 {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if len(r.OwnerDecisions) != 1 || !strings.Contains(r.OwnerDecisions[0].Matter, "says 3") {
		t.Fatalf("decisions %+v", r.OwnerDecisions)
	}
}

func TestASpecProblemAnswerEndsTheCommand(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "spec problem -- the title rule contradicts its example"}, ""))
	before := f.tip("main")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateSpecProblem || f.tip("main") != before || len(r.OwnerDecisions) != 1 {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

// An empty diff with no stop is not a failure: the work goes on to review.
func TestAnEmptyDiffGoesToReview(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	f.agents.on(implementer, f.edit(nil))
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || r.Work != start {
		t.Fatalf("%s, work %s: %s", r.State, r.Work, r.Why)
	}
}

// not shown lands, and the Result lists it.
func TestNotShownLandsAndIsListed(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "not shown -- nothing exercises the isbn"}, ""))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	page := f.show("main", run.ResultFile(r.Run))
	if !strings.Contains(page, "catalog/book: nothing exercises the isbn") {
		t.Fatalf("the Result does not list it:\n%s", page)
	}
}

// Every check of the answers refuses a planted bad answer, once corrected it
// passes, and uncorrected nothing lands (U2).
func TestTheCommandChecksTheAnswers(t *testing.T) {
	cases := map[string]struct {
		line string
		want string
	}{
		"no answer":         {"", "has no answer"},
		"no pointer":        {"ok", "ok needs a pointer"},
		"a missing file":    {"ok  src/nothing.go:1", "does not exist in this commit"},
		"a missing line":    {"ok  README.md:99", "has no line 99"},
		"a missing test":    {"ok  README.md:1 (TestNothing)", "names no test TestNothing"},
		"a file not opened": {"ok  src/catalog/book.go:1", "the transcript shows no read of src/catalog/book.go"},
		"no explanation":    {"wrong  README.md:1", "needs an explanation"},
	}
	for name, c := range cases {
		t.Run(name+" corrected", func(t *testing.T) {
			f := newFixture(t)
			base, _ := changeBook(f)
			f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
			first := f.answer(map[string]string{"catalog/book": c.line}, "")
			f.agents.on(reviewer, first, func(s agentport.Session, repo string) agentport.End {
				if s.Conversation == "" || !strings.Contains(s.Prompt, c.want) {
					t.Errorf("the correction: conversation %q, prompt:\n%s", s.Conversation, s.Prompt)
				}
				writeAbsolute(t, f.box.last.NotePath(agentport.ReviewerUser(1), "answers.md"), "catalog/book: ok  README.md:1\n")
				return agentport.End{Seen: agentport.Seen{Measured: true, Opened: []string{"README.md"}}}
			})
			r := f.run(run.Request{Base: base})
			if r.State != run.StateLanded {
				t.Fatalf("%s: %s", r.State, r.Why)
			}
		})
		t.Run(name+" uncorrected", func(t *testing.T) {
			f := newFixture(t)
			base, _ := changeBook(f)
			f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
			f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": c.line}, ""), func(agentport.Session, string) agentport.End { return agentport.End{} })
			before := f.tip("main")
			r := f.run(run.Request{Base: base})
			if r.State != run.StateIncomplete || f.tip("main") != before || !strings.Contains(r.Why, c.want) {
				t.Fatalf("%s: %s", r.State, r.Why)
			}
			if len(f.agents.of(reviewer)) != 2 {
				t.Fatalf("%d reviewer sessions: one request to correct it", len(f.agents.of(reviewer)))
			}
		})
	}
}

// A reviewer's remark in brackets is no test's name, and a range is a location.
func TestAPointerMayHoldARemarkOrARange(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "ok  README.md:1-1 (the title, and more), README.md:1"}, ""))
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	for _, it := range r.Items {
		if it.Item == "catalog/book" && (len(it.Pointers) != 2 || len(it.Pointers[0].Tests) != 0 || it.Pointers[0].Line != 1) {
			t.Fatalf("pointers %+v", it.Pointers)
		}
	}
}

// A transcript that cannot show a read says "not measured", and refuses nothing.
func TestAReadAShellCommandHidesIsNotMeasured(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	answer := f.answer(map[string]string{"catalog/book": "ok  " + bookFile + ":1"}, "")
	f.agents.on(reviewer, func(s agentport.Session, repo string) agentport.End {
		end := answer(s, repo)
		end.Seen = agentport.Seen{Measured: true, Commands: []string{"cat src/catalog/*.go"}}
		return end
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || r.Depth.PointersNotMeasured != 1 || r.Depth.Commands != 1 {
		t.Fatalf("%s, depth %+v: %s", r.State, r.Depth, r.Why)
	}
}

// A location the registry does not list for the item is reported, and refuses
// nothing.
func TestAPointerOutsideTheRegistryIsAPossibleMissingLink(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || len(r.MissingLinks) != 1 || !strings.Contains(r.MissingLinks[0], "README.md") {
		t.Fatalf("%s, links %v", r.State, r.MissingLinks)
	}
}

func TestProviderFailureEndsTheCommand(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, func(agentport.Session, string) agentport.End {
		return agentport.End{Reason: agentport.Failed, Class: agentport.RateLimit, Detail: "429"}
	})
	before := f.tip("main")
	r := f.run(run.Request{Base: base})
	if r.State != run.StateIncomplete || f.tip("main") != before || r.Sessions[0].Class != agentport.RateLimit {
		t.Fatalf("%s: %s; %+v", r.State, r.Why, r.Sessions)
	}
}

// Moved target, nothing in the impact touched: the clean merge passes the gate
// and lands, and the next impact still holds what merged in.
func TestAMovedTargetLandsACleanMerge(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	review := f.ok()
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, func(s agentport.Session, repo string) agentport.End {
		f.pull()
		f.write(map[string]string{"README.md": "# Library, moved\n"})
		f.commit("beside the Run")
		return review(s, repo)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateLanded || !strings.Contains(r.Why, "clean merge") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show("main", bookFile), "ISBN") || !strings.Contains(f.show("main", "README.md"), "moved") {
		t.Fatal("the merge lost a side")
	}
	if r.RequiredChecks.Commit == r.Work {
		t.Fatal("the required checks did not run on the merged state")
	}
	f.agents.on(implementer, f.edit(nil))
	f.agents.on(reviewer, f.ok())
	if next := f.run(run.Request{}); answerOf(next, "project") == "absent" {
		t.Fatalf("what merged in dropped out of the next impact: %s %+v", next.State, next.Items)
	}
}

// Moved target that touches the impact: the work stays on its branch.
func TestAMovedTargetThatTouchesTheImpactKeepsTheWork(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	review := f.ok()
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, func(s agentport.Session, repo string) agentport.End {
		f.pull()
		f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "A book has a title and an `ean`.")})
		f.commit("the same Definition, beside the Run")
		return review(s, repo)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || r.Outcome != run.NotLanded || f.tip(run.RunBranchPrefix+r.Run) == "" {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if !strings.Contains(f.show(run.RunBranchPrefix+r.Run, run.ResultFile(r.Run)), "impact") {
		t.Fatal("the Result does not say why")
	}
}

// Moved target whose merge fails a required check: the gate on the merged state
// refuses it.
func TestTheGateRunsOnTheMergedState(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	review := f.ok()
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, func(s agentport.Session, repo string) agentport.End {
		f.pull()
		f.write(map[string]string{"FAIL": "x\n", project.RootPage: table + "| README.md | [[project]] | Overview |\n| FAIL | [[project]] | Planted |\n"})
		f.commit("breaks the check, beside the Run")
		return review(s, repo)
	})
	r := f.run(run.Request{Base: base})
	if r.State != run.StateKept || !strings.Contains(r.Why, "merged state") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
}

func TestLandingRefusesTheTokenAndALoginFile(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"the token":    {bookFile: "package catalog\n\n// sk-ant-oat01-planted\n"},
		"a login file": {".credentials.json": "{}\n", project.RootPage: table + "| README.md | [[project]] | Overview |\n| .credentials.json | [[project]] | Planted |\n"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.engine.Secrets = [][]byte{[]byte("sk-ant-oat01-planted")}
			base, _ := changeBook(f)
			f.agents.on(implementer, f.edit(files))
			f.agents.on(reviewer, f.ok())
			before := f.tip("main")
			r := f.run(run.Request{Base: base})
			if r.State != run.StateKept || f.tip("main") != before {
				t.Fatalf("%s: %s", r.State, r.Why)
			}
		})
	}
}

func TestVerifyChangesNothingAndEndsWithFindings(t *testing.T) {
	f := newFixture(t)
	base, start := changeBook(f)
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "wrong  README.md:1 -- the book has no isbn"}, ""))
	r := f.run(run.Request{Mode: agentport.Verify, Base: base})
	if r.State != run.StateFindings || f.tip("main") != start || len(f.agents.of(implementer)) != 0 {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if len(r.Findings) != 1 || r.Findings[0].Definition != "catalog/book" || !strings.Contains(r.Findings[0].Observed, "no isbn") {
		t.Fatalf("findings %+v", r.Findings)
	}
	if !r.RequiredChecks.Ran {
		t.Fatalf("required checks %+v", r.RequiredChecks)
	}
	if !f.has(run.RunBranchPrefix+r.Run, run.ResultFile(r.Run)) {
		t.Fatal("verify's Result is not on its branch")
	}
	list, errs := analysis.ParseFindings("f.md", run.FindingsPage(r))
	if len(errs) > 0 || len(list) != 1 {
		t.Fatalf("the Findings page reads back as %+v, %v", list, errs)
	}
	// Those Findings are apply's items.
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.ok())
	fixed := f.run(run.Request{Base: start, Findings: list})
	if fixed.State != run.StateLanded || answerOf(fixed, "finding 1 catalog/book") != run.AnswerOK {
		t.Fatalf("%s: %+v", fixed.State, fixed.Items)
	}
}

// With --full, one reviewer per Package, side by side, each with its own copy
// and only its own items; the count of changes starts again.
func TestAFullCheckIsOneReviewerPerPackage(t *testing.T) {
	f := newFixture(t)
	f.agents.on(reviewer, f.ok(), f.ok(), f.ok(), f.ok())
	r := f.run(run.Request{Mode: agentport.Verify, Full: true})
	if r.State != run.StateVerified {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	sessions := f.agents.of(reviewer)
	if len(sessions) != 4 {
		t.Fatalf("%d reviewers for 4 Packages", len(sessions))
	}
	dirs, seen := map[string]bool{}, map[string]int{}
	for _, s := range sessions {
		dirs[s.Dir] = true
		for _, k := range listed(s.Prompt) {
			seen[k]++
		}
	}
	if len(dirs) != 4 {
		t.Fatalf("the reviewers share copies: %v", dirs)
	}
	for k, n := range seen {
		if n != 1 {
			t.Fatalf("%s is on %d lists", k, n)
		}
	}
	if len(r.Items) != len(seen) || r.FullCheck.Since != r.Start || r.FullCheck.Landed != 0 {
		t.Fatalf("items %d of %d; full check %+v", len(r.Items), len(seen), r.FullCheck)
	}
}

func TestOneReviewerGetsAFullChecksWholeList(t *testing.T) {
	f := newFixture(t)
	f.agents.on(reviewer, f.ok())
	r := f.run(run.Request{Mode: agentport.Verify, Full: true, OneReviewer: true})
	if r.State != run.StateVerified {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	sessions := f.agents.of(reviewer)
	if len(sessions) != 1 {
		t.Fatalf("%d reviewers", len(sessions))
	}
	if n := len(listed(sessions[0].Prompt)); n != len(r.Items) || n == 0 {
		t.Fatalf("%d items on the list of %d", n, len(r.Items))
	}
}

// The count of changes across Commands, from the target's landing commits and
// Run branches: a landing counts, a verify --full starts the count again at the
// commit it examined, and the next landing carries that forward.
func TestTheFullCheckCountAcrossCommands(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.ok())
	first := f.run(run.Request{Base: base})
	if fc := first.FullCheck; first.State != run.StateLanded || fc.Landed != 1 || fc.Since != "" || fc.Due {
		t.Fatalf("first landing %s: %+v", first.State, fc)
	}
	if page := f.show("main", run.ResultFile(first.Run)); !strings.Contains(page, "1 change landed since the first landing") {
		t.Fatalf("first page:\n%s", page)
	}
	f.agents.on(reviewer, f.ok(), f.ok(), f.ok(), f.ok())
	check := f.run(run.Request{Mode: agentport.Verify, Full: true})
	if fc := check.FullCheck; check.State != run.StateVerified || fc.Landed != 0 || fc.Since != check.Start {
		t.Fatalf("verify --full %s: %+v", check.State, fc)
	}
	f.pull()
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "A book has a title, an `isbn` and an author.")})
	f.commit("an author")
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed + "\n// author\n"}))
	f.agents.on(reviewer, f.ok())
	second := f.run(run.Request{})
	if fc := second.FullCheck; second.State != run.StateLanded || fc.Landed != 1 || fc.Since != check.Start {
		t.Fatalf("second landing %s: %+v", second.State, fc)
	}
}

// A later landing that covers a kept Run's impact deletes its branch.
func TestALandingDeletesTheKeptRunBranchItCovers(t *testing.T) {
	f := newFixture(t)
	base, _ := changeBook(f)
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.answer(map[string]string{"catalog/book": "spec problem -- it needs the owner"}, ""))
	kept := f.run(run.Request{Base: base})
	if kept.State != run.StateSpecProblem || f.tip(run.RunBranchPrefix+kept.Run) == "" {
		t.Fatalf("%s: %s", kept.State, kept.Why)
	}
	f.agents.on(implementer, f.edit(map[string]string{bookFile: bookFixed}))
	f.agents.on(reviewer, f.ok())
	landed := f.run(run.Request{Base: base})
	if landed.State != run.StateLanded || len(landed.Replaced) != 1 || f.tip(run.RunBranchPrefix+kept.Run) != "" {
		t.Fatalf("%s, replaced %v", landed.State, landed.Replaced)
	}
}

func TestARequestTheEngineCannotRun(t *testing.T) {
	f := newFixture(t)
	if _, err := f.engine.Run(t.Context(), run.Request{Mode: agentport.Apply, Target: "main", Base: "no-such-commit"}); err == nil {
		t.Fatal("an unknown base ran")
	}
	if _, err := f.engine.Run(t.Context(), run.Request{Mode: agentport.Apply, Target: "main"}); err == nil {
		t.Fatal("no base and no landing ran")
	}
	if _, err := f.engine.Run(t.Context(), run.Request{Mode: agentport.Apply, Target: "main", Run: "Not A Name"}); err == nil {
		t.Fatal("a bad name ran")
	}
}

// noteIn reads the path of the stop note out of a prompt.
func noteIn(t *testing.T, prompt string) string {
	t.Helper()
	const mark = "stop: write why to "
	i := strings.Index(prompt, mark)
	if i < 0 {
		t.Fatalf("the prompt names no stop note:\n%s", prompt)
	}
	rest := prompt[i+len(mark):]
	return rest[:strings.Index(rest, " and end")]
}

func writeAbsolute(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(path.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
