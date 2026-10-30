package harness

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

type memRepo map[string]string

func (r memRepo) Files() ([]string, error) {
	var out []string
	for f := range r {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

func (r memRepo) Read(f string) ([]byte, error) {
	if data, ok := r[f]; ok {
		return []byte(data), nil
	}
	return nil, fmt.Errorf("no %s", f)
}

func (memRepo) GitIgnored(string) bool { return false }

const (
	m     = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library is a small project: Root with a Rule, members with the child
// Package loans, and catalog beside it, which links to members.
func library(t *testing.T) *project.Project {
	t.Helper()
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	repo := memRepo{
		project.ManifestFile:                      "libraries:\n  std: 1.0.0\n",
		project.LockFile:                          string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                        "checks: []\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans. Never more."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person who borrows. Has a card."),
		m + "members/secret.md":                   "---\nkind: std/Concept\ninternal: true\n---\n# Secret\n\nOnly members knows.\n",
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows a book.\n\nMore text."),
		m + "catalog/book.md":                     def("std/Concept", "Book", "Lent to a [[members/member]] under [[loan-limit]]."),
		project.RootPage:                          table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] [[members/secret]] | Type |\n",
		project.RegistryDir + "/members/loans.md": "---\npaths: [src/members/loans/**]\n---\n" + table + "| src/members/loans/loan.go | [[members/loans/loan]] | Type |\n",
		project.RegistryDir + "/catalog.md":       "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/loans/loan.go":               "package loans\n",
		"src/catalog/book.go":                     "package catalog\n",
	}
	p, err := project.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Errors) > 0 {
		t.Fatalf("fixture does not compile: %v", p.Errors)
	}
	return p
}

func items(defs ...string) []agentport.Item {
	var out []agentport.Item
	for _, d := range defs {
		out = append(out, agentport.Item{Key: d, Part: agentport.Own, Definition: d, Why: "changed"})
	}
	return out
}

// start is an agent start for the Manager id with its working children.
func start(mode agentport.Mode, role agentport.Role, id string, children []string, checklist []agentport.Item) agentport.Start {
	parent := ""
	if id != "." {
		parent = runremote.Name(model.ParentOf(id))
	}
	return agentport.Start{Run: "r1", Mode: mode, Role: role, Manager: agentport.Manager{ID: id, Parent: parent, Children: children}, Round: 1, Checklist: checklist}
}

func render(t *testing.T, in Input) Document {
	t.Helper()
	d, err := Render(in)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func contains(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in:\n%s", w, text)
		}
	}
}

func lacks(t *testing.T, text string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(text, u) {
			t.Errorf("unexpected %q in:\n%s", u, text)
		}
	}
}

func TestMakerKnowsOwnInheritedLinkedAndItsChildrenTableOfContents(t *testing.T) {
	p := library(t)
	d := render(t, Input{Start: start(Apply, Maker, "members", []string{"members/loans"}, items("members/member")), Project: p})
	contains(t, d.Base,
		"maker of the Manager of Package `members` in Markitect Run `r1` (apply)",
		"on your Manager's branch `managers/members`",
		"registry page is `.markitect/registry/members.md`",
		"Your area is the paths it claims: `src/members/**`.",
		"Your parent is Root, on branch `root`.",
		"`members/loans` (branch `managers/members.loans`)",
	)
	contains(t, d.Context,
		"## Your Package's Definitions", "### `members/member` (.markitect/model/members/member.md)", "A person who borrows. Has a card.",
		"### `members/secret`", // own internal Definitions are the Manager's own
		"## Inherited from your parents", "### `loan-limit`", "### `project`",
		"## Kinds of the Definitions above", "### `std/Concept` (Library std)",
		"## Child members/loans", "- `members/loans/loan` (.markitect/model/members/loans/loan.md): Loan: A [[members/member]] borrows a book.",
	)
	lacks(t, d.Context, "### `catalog/book`", "More text.")
	if !(strings.Index(d.Context, "### `project`") < strings.Index(d.Context, "## Kinds")) {
		t.Error("inherited Definitions come before the Kinds")
	}
}

func TestLinkedDefinitionsInOtherPackagesAreShownInFull(t *testing.T) {
	p := library(t)
	d := render(t, Input{Start: start(Apply, Maker, "catalog", nil, items("catalog/book")), Project: p})
	contains(t, d.Context, "## Linked from these", "### `members/member`", "A person who borrows. Has a card.")
	lacks(t, d.Context, "### `members/secret`", "## Child")
}

func TestRootSeesItsChildrenOnlyAsTableOfContentsAndOwnsUnclaimedPaths(t *testing.T) {
	p := library(t)
	d := render(t, Input{Start: start(Apply, Maker, ".", []string{"catalog"}, items("project")), Project: p})
	contains(t, d.Base, "of Root in", "branch `root`", "`.markitect/registry.md`", "every path no other Package claims", "`catalog` (branch `managers/catalog`)")
	lacks(t, d.Base, "Your parent", "`members` (branch")
	contains(t, d.Context, "## Child catalog", "## Child members", "- `members/member`", "- `members/loans/loan`")
	lacks(t, d.Context, "members/secret", "### `members/member`", "Inherited")
	contains(t, d.Task, "| `project` | own | changed |")
}

func TestMakerTaskInboxAndAnswerFormat(t *testing.T) {
	p := library(t)
	s := start(Apply, Maker, "members", nil, []agentport.Item{
		{Key: "members/member", Part: agentport.Own, Definition: "members/member", Why: "changed | renamed"},
		{Key: "loan-limit", Part: agentport.Inherited, Definition: "loan-limit", Why: "changed"},
	})
	s.Inbox = []agentport.Message{
		{ID: "m-1", Kind: agentport.Review, From: "members", To: "members", Text: "Rejected: no test at the limit."},
		{ID: "m-2", Kind: agentport.Answer, From: ".", To: "members", ReplyTo: "m-0", Text: "Five, as the Rule says."},
	}
	d := render(t, Input{Start: s, Project: p})
	contains(t, d.Task, "Make the repository conform", "| `members/member` | own | changed \\| renamed |", "| `loan-limit` | inherited | changed |",
		"## Messages for you", "### `m-1`: review from `members`\n\nRejected: no test at the limit.", "### `m-2`: answer from `.`, answering `m-0`",
		"Call `submit` with exactly one verdict per checklist item", "`model-problem`", "escalate", "arrive with a later start")
	lacks(t, d.Task, "Submitted verdicts")
	s.Task = "Correct the submission your checker rejected."
	d = render(t, Input{Start: s, Project: p})
	contains(t, d.Task, "# Task\n\nCorrect the submission your checker rejected.\n\n## Checklist")
	lacks(t, d.Task, "Make the repository conform")
}

func TestCheckerSeesSubmissionVerdictsAndDiff(t *testing.T) {
	p := library(t)
	sub := &Submission{Commit: "abc123", Verdicts: []Verdict{{Item: "members/member", Verdict: "changed", Evidence: "member.go ```x```"}},
		Diff: "diff --git a/src/members/member.go b/src/members/member.go\n+type Member struct{}\n"}
	d := render(t, Input{Start: start(Apply, Checker, "members", nil, items("members/member")), Project: p, Submission: sub})
	contains(t, d.Base, "As checker you re-check", "accepts from you only a commit that changes no file")
	lacks(t, d.Base, "As maker")
	contains(t, d.Task, "submission `abc123` on branch `managers/members`", "| `members/member` | changed | member.go ```x``` |",
		"```diff\ndiff --git a/src/members/member.go", "Call `review`")
	sub.Diff = strings.Repeat("x", MaxInlineDiff+1)
	sub.Files = []string{"M src/members/member.go"}
	d = render(t, Input{Start: start(Apply, Checker, "members", nil, items("members/member")), Project: p, Submission: sub})
	contains(t, d.Task, "`git diff base abc123`", "- M src/members/member.go")
	lacks(t, d.Task, "xxxxxxxx")
}

func TestAParentWithAnEmptyChecklistIntegrates(t *testing.T) {
	p := library(t)
	d := render(t, Input{Start: start(Apply, Maker, ".", []string{"members"}, nil), Project: p})
	contains(t, d.Task, "Your checklist is empty: integrate your children's approved work")
	lacks(t, d.Task, "## Checklist")
}

func TestVerifyCheckerSubmitsVerdictsOnTheBase(t *testing.T) {
	p := library(t)
	d := render(t, Input{Start: start(Verify, Checker, "members", nil, items("members/member")), Project: p})
	contains(t, d.Task, "at the Run's base conforms", "Call `submit`", "`deviates`")
	lacks(t, d.Task, "`changed`")
	if _, err := Render(Input{Start: start(Verify, Maker, "members", nil, items("members/member")), Project: p}); err == nil {
		t.Error("verify has no makers")
	}
}

func TestRenderRefusesWhatNoAgentCanWorkFrom(t *testing.T) {
	p := library(t)
	ok := Input{Start: start(Apply, Maker, "members", nil, items("members/member")), Project: p}
	cases := map[string]func(*Input){
		"no project":        func(in *Input) { in.Project = nil },
		"unknown Package":   func(in *Input) { in.Start.Manager.ID = "nope" },
		"root as empty":     func(in *Input) { in.Start.Manager.ID = "" },
		"pipe in key":       func(in *Input) { in.Start.Checklist = []agentport.Item{{Key: "a|b"}} },
		"checker no commit": func(in *Input) { in.Start.Role = Checker },
		"unknown mode":      func(in *Input) { in.Start.Mode = "improve" },
		"not a child":       func(in *Input) { in.Start.Manager.Children = []string{"catalog"} },
		"compile errors":    func(in *Input) { q := *p; q.Errors = model.Diagnostics{{Code: "x"}}; in.Project = &q },
	}
	for name, mutate := range cases {
		in := ok
		mutate(&in)
		if _, err := Render(in); err == nil {
			t.Errorf("%s: rendered", name)
		}
	}
}

func TestItemsKeyEveryPartForThePlan(t *testing.T) {
	c := analysis.Checklist{
		Package:   "members",
		Own:       []analysis.Item{{Definition: "members/member", Reasons: []string{"changed", "realized by a touched file"}}},
		Inherited: []analysis.Item{{Definition: "loan-limit", Reasons: []string{"changed"}}},
		Findings:  []analysis.Finding{{Definition: "members/member", Observed: "no\ncard", Expected: "a card"}},
		Stale: []realization.Stale{
			{Code: "stale.definition", Package: "members", PageFile: ".markitect/registry/members.md", Line: 4, Target: "members/old"},
			{Code: "stale.definition", Package: "members", PageFile: ".markitect/registry/members.md", Line: 5, Target: "members/old"},
			{Code: "stale.file", Package: "members", PageFile: ".markitect/registry/members.md", File: "src/members/gone.go"},
			{Code: "stale.page", Package: "", PageFile: ".markitect/registry/old.md"},
		},
	}
	var keys []string
	for _, it := range Items(c) {
		keys = append(keys, it.Key+" "+it.Part)
	}
	want := []string{"members/member own", "loan-limit inherited", "finding 1 on members/member finding",
		"stale.definition members/old at .markitect/registry/members.md:4 stale", "stale.definition members/old at .markitect/registry/members.md:5 stale",
		"stale.file src/members/gone.go at .markitect/registry/members.md stale", "stale.page .markitect/registry/old.md stale"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keys %q, want %q", keys, want)
	}
	got := Items(c)
	if got[2].Why != "observed: no card; expected: a card" || !strings.Contains(got[3].Why, ":4 links members/old") || got[6].Why != ".markitect/registry/old.md is the page of a Package that no longer exists" {
		t.Fatalf("details: %+v", got)
	}
}

func TestToolsPerStart(t *testing.T) {
	names := func(ts []Tool) string {
		var out []string
		for _, tool := range ts {
			out = append(out, tool.Name)
		}
		return strings.Join(out, ",")
	}
	all := []agentport.Tool{agentport.ToolAsk, agentport.ToolEscalate, agentport.ToolAnswer, agentport.ToolAssign, agentport.ToolNote, agentport.ToolNotes, "unknown"}
	for _, c := range []struct {
		mode  agentport.Mode
		role  agentport.Role
		tools []agentport.Tool
		want  string
	}{
		{Apply, Maker, all, "submit,ask,escalate,answer,assign,note,notes"},
		{Apply, Maker, []agentport.Tool{agentport.ToolEscalate}, "submit,escalate"},
		{Apply, Checker, []agentport.Tool{agentport.ToolNote}, "review,note"},
		{Verify, Checker, nil, "submit"},
	} {
		s := agentport.Start{Mode: c.mode, Role: c.role, Tools: c.tools}
		if got := names(Tools(s)); got != c.want {
			t.Errorf("%s %s: %s, want %s", c.mode, c.role, got, c.want)
		}
	}
	for _, tool := range Tools(agentport.Start{Mode: Apply, Role: Maker, Tools: all}) {
		var schema map[string]any
		if err := json.Unmarshal(tool.Schema, &schema); err != nil || schema["type"] != "object" {
			t.Errorf("%s: schema %s %v", tool.Name, tool.Schema, err)
		}
	}
}

func TestFenceOutgrowsBackticksInTheText(t *testing.T) {
	if got := fenceFor([]byte("a ```` b")); got != "`````" {
		t.Fatalf("fence %q", got)
	}
	if got := fenceFor([]byte("plain")); got != "```" {
		t.Fatalf("fence %q", got)
	}
}

// Every maker and checker reads what a verdict covers (DEC-088): the model
// in full for each item, with evidence for each example, and the code of its
// own Package only; a parent integrates its children's work by using it.
func TestEveryStartSaysWhatAVerdictCovers(t *testing.T) {
	p := library(t)
	sub := &Submission{Commit: "abc123", Verdicts: []Verdict{{Item: "members/member", Verdict: "changed", Evidence: "member.go"}}, Diff: "diff\n"}
	model := []string{"## What a verdict covers", "its examples included, and the Definitions it links to", "the only right verdict is `model-problem`", "It is no review of the whole model"}

	leaf := start(Apply, Maker, "catalog", nil, items("catalog/book"))
	leaf.Task = "An engine task replaces the default text."
	d := render(t, Input{Start: leaf, Project: p})
	contains(t, d.Task, model...)
	contains(t, d.Task, "For `conforms` and `changed`, the evidence names how each example", "Whether the rest of the program uses them yet is your parent's integration")
	lacks(t, d.Task, "Your children's work", "accepts the submission only when the required checks pass")
	contains(t, d.Task, "refuses a submission that merges your children's work and fails them")

	d = render(t, Input{Start: start(Apply, Checker, "members", []string{"members/loans"}, items("members/member")), Project: p, Submission: sub})
	contains(t, d.Task, model...)
	contains(t, d.Task, "Re-check each example in the item's Definition yourself", "matches part of the Definition but not its example",
		"it is no reason to dispute a verdict", "Dispute the submission when one of its own files builds a child's behavior a second time")

	d = render(t, Input{Start: start(Apply, Maker, "members", []string{"members/loans"}, items("members/member")), Project: p})
	contains(t, d.Task, "Integrate it by using it", "not by building a child's behavior a second time in your own files")

	// A parent that only integrates still hears about its children's work.
	d = render(t, Input{Start: start(Apply, Checker, ".", []string{"members"}, nil), Project: p, Submission: &Submission{Commit: "abc123", Diff: "diff\n"}})
	contains(t, d.Task, "## What a verdict covers", "Dispute the submission when one of its own files builds a child's behavior a second time")
	lacks(t, d.Task, "The model, in full")

	d = render(t, Input{Start: start(Verify, Checker, "members", []string{"members/loans"}, items("members/member")), Project: p})
	contains(t, d.Task, model...)
	contains(t, d.Task, "For `conforms`, the evidence names how each example")
	lacks(t, d.Task, "Your children's work", "`changed`")
}
