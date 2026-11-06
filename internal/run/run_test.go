package run_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

const (
	m     = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library is the fixture project: Root with a Rule, members with a sub-Package
// loans, and catalog, with every file accounted for.
func library() map[string]string {
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return map[string]string{
		project.ManifestFile:                      "libraries:\n  std: 1.0.0\n",
		project.LockFile:                          string(model.RenderLock([]model.LibrarySource{std})),
		".claude/agents/markitect-reviewer.md":    "---\nname: markitect-reviewer\n---\nReview.\n",
		".claude/agents/markitect-builder.md":     "---\nname: markitect-builder\n---\nBuild.\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows."),
		m + "catalog/book.md":                     def("std/Concept", "Book", "A book has a title."),
		project.RootPage:                          table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Member |\n",
		project.RegistryDir + "/members/loans.md": "---\npaths: [src/members/loans/**]\n---\n" + table + "| src/members/loans/loan.go | [[members/loans/loan]], [[loan-limit]] | Loans |\n",
		project.RegistryDir + "/catalog.md":       "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Book |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/loans/loan.go":               "package loans\n\nconst Limit = 5\n",
		"src/catalog/book.go":                     "package catalog\n",
		".gitignore":                              ".claude/worktrees/\n",
	}
}

// fixture is a repository and its engine; the test plays the agents.
type fixture struct {
	t      *testing.T
	repo   *testkit.Repo
	first  string
	engine *run.Engine
	now    time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return newFixtureOf(t, library())
}

func newFixtureOf(t *testing.T, files map[string]string) *fixture {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("config", "user.name", "Owner")
	repo.Git("config", "user.email", "owner@example.invalid")
	repo.Git("config", "core.autocrlf", "false")
	for file, content := range files {
		repo.Write(file, content)
	}
	first := repo.Commit("the library")
	host, err := rungit.Open(context.Background(), repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, repo: repo, first: first}
	f.now = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	f.engine = &run.Engine{Git: host, Store: run.Store{Dir: filepath.Join(host.Dir, "markitect")}, Now: func() time.Time { return f.now }}
	return f
}

// apply changes the loan limit in the spec and applies it.
func (f *fixture) apply() (run.Started, string) {
	f.t.Helper()
	f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
	if err != nil {
		f.t.Fatal(err)
	}
	if s.Run == "" {
		f.t.Fatalf("no run started: %+v", s)
	}
	return s, f.repo.Dir
}

// applyFull is the same with nothing landed yet, so that every Definition is on
// the list.
func (f *fixture) applyFull() (run.Started, string) {
	f.t.Helper()
	f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		f.t.Fatal(err)
	}
	if s.Run == "" {
		f.t.Fatalf("no run started: %+v", s)
	}
	return s, f.repo.Dir
}

func (f *fixture) state(name string) *run.RunState {
	f.t.Helper()
	st, ok, err := f.engine.Store.Load(name)
	if err != nil || !ok {
		f.t.Fatalf("run %s: %v %v", name, ok, err)
	}
	return st
}

func write(t *testing.T, dir, file, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) submit(wt string) run.Verdict {
	f.t.Helper()
	v, err := f.engine.Submit(context.Background(), wt)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func has(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Fatalf("%q not in:\n%s", w, text)
		}
	}
}

// answer answers every item of the review with a word, except where words says
// otherwise, the way a reviewer does.
func (f *fixture) answer(wt string, name string, except map[string]string) string {
	f.t.Helper()
	st := f.state(name)
	var lines []string
	for _, it := range st.Review.Items {
		line, ok := except[it.Key]
		if !ok {
			line = "ok README.md:1"
		}
		lines = append(lines, it.Key+": "+line)
	}
	text, _, err := f.engine.Answer(context.Background(), wt, []byte(strings.Join(lines, "\n")))
	if err != nil {
		f.t.Fatal(err)
	}
	return text
}

// The owner may work in a linked worktree of their own, as the desktop app does.
// The run lands on their branch there, and an early end keeps its work.
func TestFromALinkedWorktree(t *testing.T) {
	t.Run("lands on the owner's branch", func(t *testing.T) {
		f := newFixture(t)
		owner := filepath.Join(testkit.TempDir(t), "owner")
		f.repo.Git("worktree", "add", "-q", "-b", "owner-branch", owner, "HEAD")
		write(t, owner, m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: owner, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.Run == "" {
			t.Fatalf("%+v", s)
		}
		st := f.state(s.Run)
		if st.Branch != "owner-branch" {
			t.Fatalf("branch %q", st.Branch)
		}
		wt := owner
		f.submit(wt)
		f.answer(wt, s.Run, nil)
		if v := f.submit(wt); !v.Done || f.state(s.Run).State != run.StateLanded {
			t.Fatalf("%s", v.Text)
		}
		if head := testkit.Git(t, owner, "rev-parse", "HEAD"); head != f.state(s.Run).Landed {
			t.Fatalf("the owner's worktree is at %s", head)
		}
		if data, err := os.ReadFile(filepath.Join(owner, ".markitect", "results", s.Run+".md")); err != nil || len(data) == 0 {
			t.Fatalf("the Result did not reach the owner's folder: %v", err)
		}
	})
	t.Run("an early end keeps the work", func(t *testing.T) {
		f := newFixture(t)
		owner := filepath.Join(testkit.TempDir(t), "owner")
		f.repo.Git("worktree", "add", "-q", "-b", "owner-branch", owner, "HEAD")
		write(t, owner, m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: owner, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		wt := owner
		write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
		if _, err := f.engine.Stop(context.Background(), owner, "changed my mind", false); err != nil {
			t.Fatal(err)
		}
		has(t, f.repo.Git("show", f.state(s.Run).Kept+":src/members/loans/loan.go"), "Limit = 3")
	})
}

// A change lands through the door: the review, and the owner's branch moved
// forward over the work already in the folder, with the Result in the landing
// commit.
func TestAChangeLands(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	has(t, s.Text, s.Run, "is open", "Build run "+s.Run)
	if got := f.repo.Git("log", "-1", "--format=%s"); !strings.Contains(got, "as markitect apply found it") {
		t.Fatalf("the spec was not committed: %s", got)
	}
	if status := f.repo.Git("status", "--porcelain"); status != "" {
		t.Fatalf("the owner's folder is not clean: %s", status)
	}

	list, err := f.engine.Checklist(context.Background(), wt, false)
	if err != nil {
		t.Fatal(err)
	}
	has(t, list, "Make this project match the spec", "loan-limit", "src/members/loans/loan.go", "do not commit")

	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	v := f.submit(wt)
	has(t, v.Text, "markitect-reviewer", "Review run "+s.Run)
	if v.Done {
		t.Fatal("submit closed the run before a review")
	}
	if head := f.repo.Git("rev-parse", "HEAD"); head != f.state(s.Run).Start {
		t.Fatalf("submit committed on the owner's branch before the review: %s", head)
	}
	review, err := f.engine.Checklist(context.Background(), wt, true)
	if err != nil {
		t.Fatal(err)
	}
	has(t, review, "owe it no benefit of the doubt", "markitect answer", "loan-limit")

	has(t, f.answer(wt, s.Run, nil), "every item has an answer")
	v = f.submit(wt)
	if !v.Done {
		t.Fatalf("did not land: %s", v.Text)
	}
	has(t, v.Text, "Landed on")

	st := f.state(s.Run)
	if st.Open || st.State != run.StateLanded || st.Landed == "" {
		t.Fatalf("state: %+v", st)
	}
	if head := f.repo.Git("rev-parse", "HEAD"); head != st.Landed {
		t.Fatalf("the owner's branch is at %s, not %s", head, st.Landed)
	}
	data, err := os.ReadFile(filepath.Join(f.repo.Dir, ".markitect", "results", s.Run+".md"))
	if err != nil {
		t.Fatal(err)
	}
	has(t, string(data), "landed", "not measured")
	got, err := os.ReadFile(filepath.Join(f.repo.Dir, "src", "members", "loans", "loan.go"))
	if err != nil || !strings.Contains(string(got), "Limit = 3") {
		t.Fatalf("the work is not in the owner's folder: %s %v", got, err)
	}

	// The next apply has nothing to do.
	next, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	if next.State != run.StateNothing {
		t.Fatalf("next apply: %+v", next)
	}
	if status := f.repo.Git("status", "--porcelain"); status != "" {
		t.Fatalf("the folder is not clean after landing: %s", status)
	}
}

func TestApplyStopsBeforeAnyAgent(t *testing.T) {
	t.Run("an error in the spec", func(t *testing.T) {
		f := newFixture(t)
		f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "See [[nothing/here]]."))
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.State != run.StateCheckFailed || s.Run != "" {
			t.Fatalf("%+v", s)
		}
		has(t, s.Text, "errors", "nothing/here")
	})
	t.Run("a change outside the spec", func(t *testing.T) {
		f := newFixture(t)
		f.repo.Write("README.md", "# Changed by hand\n")
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.State != run.StateCheckFailed || s.Run != "" {
			t.Fatalf("%+v", s)
		}
		has(t, s.Text, "README.md", "not committed")
	})
	t.Run("nothing to do", func(t *testing.T) {
		f := newFixture(t)
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.State != run.StateNothing || s.Run != "" {
			t.Fatalf("%+v", s)
		}
	})
	t.Run("no harness", func(t *testing.T) {
		f := newFixture(t)
		f.repo.Git("rm", "-q", ".claude/agents/markitect-reviewer.md")
		f.repo.Commit("no harness")
		f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.State != run.StateCheckFailed || s.Run != "" {
			t.Fatalf("%+v", s)
		}
		has(t, s.Text, "markitect init")
	})
	t.Run("detached HEAD", func(t *testing.T) {
		f := newFixture(t)
		f.repo.Git("checkout", "-q", "--detach")
		if _, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first}); err == nil {
			t.Fatal("apply ran on a detached HEAD")
		}
	})
}

// Row 1 of submit: what a Run may not change.
func TestSubmitRefusesProtectedFiles(t *testing.T) {
	for file, want := range map[string]string{
		m + "loan-limit.md":                    "is the owner's",
		project.IgnoreFile:                     "is the owner's",
		"AGENTS.md":                            "harness",
		"claude.md":                            "harness",
		".claude/skills/markitect-apply/x.md":  "harness",
		".claude/agents/markitect-reviewer.md": "harness",
	} {
		t.Run(file, func(t *testing.T) {
			f := newFixture(t)
			s, wt := f.apply()
			write(t, wt, file, "changed\n")
			v := f.submit(wt)
			has(t, v.Text, want)
			if v.Done || f.state(s.Run).Review != nil {
				t.Fatalf("not refused: %s", v.Text)
			}
		})
	}
}

// check passes a project with nothing built, but the gate does not: a Run on a new
// project is refused until it realizes every Definition.
func TestSubmitRefusesAnUnrealizedDefinitionOnANewProject(t *testing.T) {
	all := library()
	f := newFixtureOf(t, map[string]string{
		project.ManifestFile:                   all[project.ManifestFile],
		project.LockFile:                       all[project.LockFile],
		m + "project.md":                       def("std/Project", "New", "A new project."),
		project.RootPage:                       "| File | Realizes | Why |\n|---|---|---|\n",
		".claude/agents/markitect-reviewer.md": all[".claude/agents/markitect-reviewer.md"],
		".claude/agents/markitect-builder.md":  all[".claude/agents/markitect-builder.md"],
	})
	s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.Run == "" {
		t.Fatalf("no run started: %+v", s)
	}
	wt := f.repo.Dir
	has(t, f.submit(wt).Text, "markitect check fails", "definition.unused")
}

// Row 3: check, of the spec and the realization.
func TestSubmitRefusesWhatCheckRefuses(t *testing.T) {
	f := newFixture(t)
	_, wt := f.apply()
	write(t, wt, "src/members/stray.go", "package members\n")
	has(t, f.submit(wt).Text, "markitect check fails", "src/members/stray.go")

	if err := os.Remove(filepath.Join(wt, "src", "members", "stray.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(wt, "src", "catalog", "book.go")); err != nil {
		t.Fatal(err)
	}
	has(t, f.submit(wt).Text, "markitect check fails", "stale.file", "src/catalog/book.go")
}

// The answers: a bad one is refused at once, and nothing of that call is stored.
func TestAnswerRefusesWhatItCannotAccept(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	has(t, f.answer(wt, s.Run, map[string]string{"loan-limit": "ok nosuch/file.go:1"}), "not accepted", "does not exist")
	if got := f.state(s.Run).Review.Answered; len(got) != 0 {
		t.Fatalf("stored: %+v", got)
	}
	has(t, f.answer(wt, s.Run, map[string]string{"loan-limit": "ok README.md:99"}), "has no line 99")
	has(t, f.answer(wt, s.Run, map[string]string{"loan-limit": "wrong README.md:1"}), "needs an explanation")
	text, complete, err := f.engine.Answer(context.Background(), wt, []byte("project: ok README.md:1\n"))
	if err != nil || complete {
		t.Fatalf("%v %v", complete, err)
	}
	has(t, text, "still have no answer")
}

// Row 2: a reviewer's leftover is not new work, and listing it as the work is the
// implementer's call.
func TestLeftoversDoNotInvalidateAReview(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	f.submit(wt)
	f.answer(wt, s.Run, nil)
	write(t, wt, "out.log", "left by a test run\n")
	v := f.submit(wt)
	has(t, v.Text, "out.log", "List them on the registry if they are the work, otherwise remove them")
	if v.Done {
		t.Fatal("landed with a leftover")
	}
	if err := os.Remove(filepath.Join(wt, "out.log")); err != nil {
		t.Fatal(err)
	}
	if v := f.submit(wt); !v.Done {
		t.Fatalf("did not land after the leftover went: %s", v.Text)
	}
	if st := f.state(s.Run); st.State != run.StateLanded {
		t.Fatalf("%+v", st)
	}
}

// Row 8: wrong answers go back, one round per reviewed state, and a repair
// review asks only about what its diff touched.
func TestRepairRounds(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	wrong := map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- still says 5"}
	f.answer(wt, s.Run, wrong)
	v := f.submit(wt)
	has(t, v.Text, "repair round 1 of 2", "still says 5")
	// Calling again without a change uses no further round.
	has(t, f.submit(wt).Text, "repair round 1 of 2")
	if got := f.state(s.Run).Repairs; got != 1 {
		t.Fatalf("repairs %d", got)
	}

	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	has(t, f.submit(wt).Text, "markitect-reviewer")
	st := f.state(s.Run)
	var keys []string
	for _, it := range st.Review.Items {
		keys = append(keys, it.Key)
	}
	sort.Strings(keys)
	if got := strings.Join(keys, ","); !strings.Contains(got, "loan-limit") || strings.Contains(got, "catalog/book") {
		t.Fatalf("a repair review asks about %s", got)
	}
	review, err := f.engine.Checklist(context.Background(), wt, true)
	if err != nil {
		t.Fatal(err)
	}
	has(t, review, "An earlier review", "still says 5")

	// Second round, then the third wrong answer closes the run.
	f.answer(wt, s.Run, wrong)
	has(t, f.submit(wt).Text, "repair round 2 of 2")
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 4\n")
	has(t, f.submit(wt).Text, "markitect-reviewer")
	f.answer(wt, s.Run, wrong)
	v = f.submit(wt)
	if !v.Done {
		t.Fatalf("not closed: %s", v.Text)
	}
	st = f.state(s.Run)
	if st.Open || st.State != run.StateRefused || st.Kept == "" {
		t.Fatalf("%+v", st)
	}
	if tip := f.repo.Git("rev-parse", st.Kept); tip == "" {
		t.Fatal("the work is not kept")
	}
	if f.repo.Git("rev-parse", "HEAD") == f.repo.Git("rev-parse", st.Kept) {
		t.Fatal("the owner's branch moved")
	}
}

// An implementer that reads the reviewer's report and repairs before calling
// submit again uses a round all the same, and the next review asks only about what
// the repair touched. That is what a real session did.
func TestARepairBeforeSubmitCountsAsARound(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	wrong := map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- still says 5"}
	f.answer(wt, s.Run, wrong)

	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	has(t, f.submit(wt).Text, "markitect-reviewer")
	st := f.state(s.Run)
	if st.Repairs != 1 || len(st.SentBack) != 1 {
		t.Fatalf("repairs %d, sent back %v", st.Repairs, st.SentBack)
	}
	for _, it := range st.Review.Items {
		if it.Key == "catalog/book" {
			t.Fatal("a repair review asks about what the repair did not touch")
		}
	}

	// A third such round closes the run.
	f.answer(wt, s.Run, wrong)
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 4\n")
	has(t, f.submit(wt).Text, "markitect-reviewer")
	f.answer(wt, s.Run, wrong)
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 6\n")
	if v := f.submit(wt); !v.Done {
		t.Fatalf("not closed: %s", v.Text)
	}
	if st := f.state(s.Run); st.State != run.StateRefused {
		t.Fatalf("%+v", st)
	}
}

// A spec problem in a finished review stands, even if the implementer carries on.
func TestASpecProblemStandsThroughARepair(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	f.answer(wt, s.Run, map[string]string{"loan-limit": "spec problem -- the limit contradicts the loan page"})
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	if v := f.submit(wt); !v.Done {
		t.Fatalf("not closed: %s", v.Text)
	}
	if st := f.state(s.Run); st.State != run.StateSpecProblem {
		t.Fatalf("%+v", st)
	}
}

// Row 7: a spec problem ends the run, ahead of any wrong answer, with no round.
func TestASpecProblemEndsTheRun(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	f.answer(wt, s.Run, map[string]string{
		"loan-limit": "spec problem -- the limit contradicts the loan page",
		"project":    "wrong README.md:1 -- says nothing",
	})
	v := f.submit(wt)
	if !v.Done {
		t.Fatalf("%s", v.Text)
	}
	st := f.state(s.Run)
	if st.State != run.StateSpecProblem || st.Repairs != 0 {
		t.Fatalf("%+v", st)
	}
	page := f.repo.Git("show", st.Kept+":.markitect/results/"+s.Run+".md")
	has(t, page, "Decisions for the owner", "contradicts")
}

// One Run at a time per folder, since its agents work there.
func TestOneRunPerFolder(t *testing.T) {
	f := newFixture(t)
	s, _ := f.apply()
	again, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	if again.State != "open" || again.Run != s.Run {
		t.Fatalf("%+v", again)
	}
	has(t, again.Text, "still open", "Build run "+s.Run)
}

func TestStop(t *testing.T) {
	t.Run("keeps the work, on its branch and in the folder", func(t *testing.T) {
		f := newFixture(t)
		s, wt := f.apply()
		write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
		text, err := f.engine.Stop(context.Background(), f.repo.Dir, "changed my mind", false)
		if err != nil {
			t.Fatal(err)
		}
		has(t, text, "without landing", "still in your folder")
		has(t, f.repo.Git("status", "--porcelain"), "loan.go")
		st := f.state(s.Run)
		if st.Open || st.State != run.StateStopped {
			t.Fatalf("%+v", st)
		}
		has(t, f.repo.Git("show", st.Kept+":src/members/loans/loan.go"), "Limit = 3")
	})
	t.Run("the reason is a question for the owner", func(t *testing.T) {
		f := newFixture(t)
		s, wt := f.apply()
		if _, err := f.engine.Stop(context.Background(), wt, "the limit contradicts the loan page", false); err != nil {
			t.Fatal(err)
		}
		st := f.state(s.Run)
		if st.State != run.StateStopped {
			t.Fatalf("%+v", st)
		}
		has(t, f.repo.Git("show", st.Kept+":.markitect/results/"+s.Run+".md"), "Decisions for the owner", "contradicts the loan page")
		has(t, f.submit(wt).Text, "closed")
	})
}

func hookInput(t *testing.T, v map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Only the reviewer answers: an agent cannot mark its own work as done.
func TestAnswerGuard(t *testing.T) {
	call := func(agent, command string) string {
		return string(run.AnswerGuard(hookInput(t, map[string]any{
			"agent_type": agent, "tool_name": "Bash", "tool_input": map[string]any{"command": command},
		})))
	}
	has(t, call("", "markitect answer <<'EOF'\nx\nEOF"), `"permissionDecision":"deny"`, run.ReviewerAgent)
	has(t, call(run.BuilderAgent, "cd . && markitect answer f.txt"), `"permissionDecision":"deny"`)
	if out := call(run.ReviewerAgent, "markitect answer f.txt"); out != "" {
		t.Fatalf("refused the reviewer: %s", out)
	}
	if out := call(run.BuilderAgent, "go test ./..."); out != "" {
		t.Fatalf("%s", out)
	}
	if out := string(run.AnswerGuard([]byte("not json"))); out != "" {
		t.Fatalf("%s", out)
	}
}

func TestVerifyNeedsACleanFolder(t *testing.T) {
	f := newFixture(t)
	write(t, f.repo.Dir, "src/members/loans/loan.go", "package loans\n\nconst Limit = 9\n")
	s, err := f.engine.Verify(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	if s.State != run.StateCheckFailed || s.Run != "" {
		t.Fatalf("%+v", s)
	}
	has(t, s.Text, "not committed", "loan.go")
}

func TestApplySaysWhichAgentToStart(t *testing.T) {
	f := newFixture(t)
	s, _ := f.apply()
	has(t, s.Text, "is open", run.BuilderAgent, "Build run "+s.Run, "markitect status")
	f2 := newFixture(t)
	f2.repo.Write(".markitect/manifest.yaml", "libraries:\n  std: 1.0.0\nmodel: haiku\n")
	f2.repo.Commit("on haiku")
	s2, _ := f2.apply()
	has(t, s2.Text, "on model haiku")
}

func TestSubmitCeiling(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	write(t, wt, "src/members/stray.go", "package members\n")
	var last run.Verdict
	for i := 0; i < 31; i++ {
		last = f.submit(wt)
	}
	if !last.Done {
		t.Fatalf("not closed: %s", last.Text)
	}
	if st := f.state(s.Run); st.State != run.StateRefused {
		t.Fatalf("%+v", st)
	}
}

func TestDeadlineAtSubmit(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	f.now = f.now.Add(7 * time.Hour)
	if v := f.submit(wt); !v.Done {
		t.Fatalf("%s", v.Text)
	}
	if st := f.state(s.Run); st.State != run.StateIncomplete {
		t.Fatalf("%+v", st)
	}
}

func TestNoticeTellsOnce(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	f.submit(wt)
	f.answer(wt, s.Run, nil)
	f.submit(wt)
	ctx := context.Background()
	has(t, f.engine.Notice(ctx, f.repo.Dir), s.Run, "landed")
	if again := f.engine.Notice(ctx, f.repo.Dir); again != "" {
		t.Fatalf("told twice: %s", again)
	}
	if inside := f.engine.Notice(ctx, wt); inside != "" {
		t.Fatalf("told inside the run: %s", inside)
	}
}

func TestVerify(t *testing.T) {
	start := func(t *testing.T) (*fixture, string, string) {
		f := newFixture(t)
		s, err := f.engine.Verify(context.Background(), run.Request{Dir: f.repo.Dir})
		if err != nil {
			t.Fatal(err)
		}
		if s.Run == "" {
			t.Fatalf("%+v", s)
		}
		has(t, s.Text, run.ReviewerAgent, "Review run "+s.Run)
		return f, s.Run, f.repo.Dir
	}
	t.Run("clean", func(t *testing.T) {
		f, name, wt := start(t)
		before := f.repo.Git("rev-parse", "HEAD")
		has(t, f.submit(wt).Text, "no answer yet")
		has(t, f.answer(wt, name, nil), "markitect submit")
		v := f.submit(wt)
		if !v.Done {
			t.Fatalf("%s", v.Text)
		}
		st := f.state(name)
		if st.State != run.StateVerified || st.Kept == "" {
			t.Fatalf("%+v", st)
		}
		if f.repo.Git("rev-parse", "HEAD") != before || f.repo.Git("status", "--porcelain") != "" {
			t.Fatal("verify changed what the owner has")
		}
		has(t, f.repo.Git("log", "-1", "--format=%B", st.Kept), "Run-Command: verify", "Run-Full: true")
	})
	t.Run("findings", func(t *testing.T) {
		f, name, wt := start(t)
		f.answer(wt, name, map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- says 5"})
		v := f.submit(wt)
		if !v.Done {
			t.Fatalf("%s", v.Text)
		}
		st := f.state(name)
		if st.State != run.StateFindings {
			t.Fatalf("%+v", st)
		}
		page := f.repo.Git("show", st.Kept+":.markitect/results/"+name+".md")
		has(t, page, "Findings", "[[loan-limit]]", "says 5")
	})
	t.Run("holds the folder until it ends", func(t *testing.T) {
		f, name, _ := start(t)
		f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
		s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first})
		if err != nil {
			t.Fatal(err)
		}
		if s.State != "open" || s.Run != name {
			t.Fatalf("%+v", s)
		}
		has(t, s.Text, "Review run "+name)
		if _, err := f.engine.Stop(context.Background(), f.repo.Dir, "enough", false); err != nil {
			t.Fatal(err)
		}
		if s, err = f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir, Base: f.first}); err != nil || s.State != "" || s.Run == "" {
			t.Fatalf("%+v %v", s, err)
		}
	})
}

func TestStoreLockIsTakenOverWhenItsProcessHasGone(t *testing.T) {
	store := run.Store{Dir: testkit.TempDir(t)}
	release, err := store.TrySubmitLock("r1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.TrySubmitLock("r1"); err != run.ErrBusy {
		t.Fatalf("a held lock was taken: %v", err)
	}
	if !store.SubmitRunning("r1") {
		t.Fatal("a held lock does not show")
	}
	release()
	if store.SubmitRunning("r1") {
		t.Fatal("a released lock shows")
	}
	// A lock left by a process that no longer exists.
	if err := os.MkdirAll(filepath.Join(store.Dir, "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, "runs", "r2.submit.lock"), []byte("999999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	release, err = store.TrySubmitLock("r2")
	if err != nil {
		t.Fatalf("the stale lock was not taken over: %v", err)
	}
	release()
}

func TestFullWhenNothingHasLanded(t *testing.T) {
	f := newFixture(t)
	f.repo.Write(m+"loan-limit.md", def("std/Rule", "Loan limit", "At most `3` open loans."))
	s, err := f.engine.Apply(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	st := f.state(s.Run)
	if !st.Full || len(st.Items) < 5 {
		t.Fatalf("%+v", st)
	}
}

// A stop does not cut across a submit that may be landing.
func TestStopWaitsForASubmit(t *testing.T) {
	f := newFixture(t)
	s, _ := f.apply()
	release, err := f.engine.Store.TrySubmitLock(s.Run)
	if err != nil {
		t.Fatal(err)
	}
	text, err := f.engine.Stop(context.Background(), f.repo.Dir, "changed my mind", false)
	if err != nil {
		t.Fatal(err)
	}
	has(t, text, "is running", "not stopped")
	if !f.state(s.Run).Open {
		t.Fatalf("stopped anyway: %+v", f.state(s.Run))
	}
	release()
	if _, err := f.engine.Stop(context.Background(), f.repo.Dir, "changed my mind", false); err != nil {
		t.Fatal(err)
	}
	if f.state(s.Run).Open {
		t.Fatal("still open")
	}
}

// Answers stored while a submit holds its copy of the state stand.
func TestAnswersDuringASubmitStand(t *testing.T) {
	f := newFixture(t)
	s, wt := f.applyFull()
	f.submit(wt)
	held := f.state(s.Run)
	has(t, f.answer(wt, s.Run, nil), "every item has an answer")
	if err := f.engine.Store.Settle(held); err != nil {
		t.Fatal(err)
	}
	if st := f.state(s.Run); !st.Review.Complete() || len(st.Answers) == 0 {
		t.Fatalf("the answers are gone: %+v", st.Review)
	}
}

// A kept branch merged by hand is not realized work: it stays in the next impact.
func TestAKeptRunMergedByHandStaysToBeReviewed(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	if _, err := f.engine.Stop(context.Background(), f.repo.Dir, "changed my mind", false); err != nil {
		t.Fatal(err)
	}
	// The work is also in the folder, uncommitted; the owner drops it there first.
	f.repo.Git("checkout", "-q", "--", ".")
	f.repo.Git("merge", "-q", "--ff-only", f.state(s.Run).Kept)
	if _, ok, err := f.engine.LastBase(context.Background(), f.repo.Dir); err != nil || ok {
		t.Fatalf("the merge counts as a landing: %v %v", ok, err)
	}
}

// A landing that was killed after moving the branch finishes on a second call.
func TestALandingThatWasKilledFinishes(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	write(t, wt, "src/members/loans/loan.go", "package loans\n\nconst Limit = 3\n")
	f.submit(wt)
	f.answer(wt, s.Run, nil)
	st := f.state(s.Run)
	// Killed after it moved the branch, before the Result: the folder holds the work.
	f.repo.Git("reset", "-q", "--mixed", st.Review.Commit)
	before := f.repo.Git("rev-parse", "HEAD")
	if v := f.submit(wt); !v.Done {
		t.Fatalf("%s", v.Text)
	}
	if st := f.state(s.Run); st.State != run.StateLanded || f.repo.Git("rev-parse", "HEAD") != before {
		t.Fatalf("%+v", st)
	}
}

// A landing does not delete a verify's Findings.
func TestALandingKeepsAVerifysFindings(t *testing.T) {
	f := newFixture(t)
	v, err := f.engine.Verify(context.Background(), run.Request{Dir: f.repo.Dir})
	if err != nil {
		t.Fatal(err)
	}
	wt := f.repo.Dir
	f.answer(wt, v.Run, map[string]string{"loan-limit": "wrong src/members/loans/loan.go:3 -- says 5"})
	f.submit(wt)
	kept := f.state(v.Run).Kept
	has(t, f.repo.Git("show", kept+":.markitect/results/"+v.Run+"-findings.md"), "[[loan-limit]]", "says 5")

	s, wtApply := f.apply()
	f.submit(wtApply)
	f.answer(wtApply, s.Run, nil)
	if got := f.submit(wtApply); !got.Done {
		t.Fatalf("%s", got.Text)
	}
	if f.repo.Git("rev-parse", kept) == "" {
		t.Fatal("the Findings are gone")
	}
}

func TestStaleLocksAndStrayFiles(t *testing.T) {
	store := run.Store{Dir: testkit.TempDir(t)}
	dir := filepath.Join(store.Dir, "runs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	empty := filepath.Join(dir, "r1.submit.lock")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(empty, old, old); err != nil {
		t.Fatal(err)
	}
	if store.SubmitRunning("r1") {
		t.Fatal("an old empty lock counts as running")
	}
	release, err := store.TrySubmitLock("r1")
	if err != nil {
		t.Fatalf("an old empty lock was not taken over: %v", err)
	}
	release()
	// A lock of a live process counts until it is older than any shell call can be.
	live := filepath.Join(dir, "r2.submit.lock")
	if err := os.WriteFile(live, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if !store.SubmitRunning("r2") {
		t.Fatal("a live lock does not count")
	}
	ancient := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(live, ancient, ancient); err != nil {
		t.Fatal(err)
	}
	if store.SubmitRunning("r2") {
		t.Fatal("a process ID used again keeps a lock for ever")
	}
	// Neither a settings file nor a broken state stops every command.
	if err := os.WriteFile(filepath.Join(dir, "r3.settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "r4.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	all, err := store.All()
	if err != nil || len(all) != 0 {
		t.Fatalf("%+v %v", all, err)
	}
}

// A stalled verify can be resumed and stopped from the owner's folder.
// A session that says the machine lacks something asks the owner nothing about
// the spec.
func TestStopForTheEnvironment(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	if _, err := f.engine.Stop(context.Background(), wt, "the database is not running", true); err != nil {
		t.Fatal(err)
	}
	st := f.state(s.Run)
	if st.State != run.StateIncomplete {
		t.Fatalf("%+v", st)
	}
	page := f.repo.Git("show", st.Kept+":.markitect/results/"+s.Run+".md")
	if strings.Contains(page, "Decisions for the owner") {
		t.Fatalf("a question about the spec: %s", page)
	}
	has(t, page, "the database is not running")
}

func TestStatusCountsCommitsSinceTheResult(t *testing.T) {
	f := newFixture(t)
	s, wt := f.apply()
	f.submit(wt)
	f.answer(wt, s.Run, nil)
	f.submit(wt)
	f.repo.Write("README.md", "# By hand\n")
	f.repo.Commit("a hand fix")
	text, err := f.engine.Status(context.Background(), f.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	has(t, text, "1 commit since its Result")
}
