package harness

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
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

func items(defs ...string) []Item {
	var out []Item
	for _, d := range defs {
		out = append(out, Item{Key: d, Part: PartOwn, Definition: d, Detail: "changed"})
	}
	return out
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
	d := render(t, Input{Run: "r1", Mode: Apply, Role: Maker, Package: "members", Project: p, Items: items("members/member"), Working: []string{"", "members", "members/loans"}})
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
	d := render(t, Input{Run: "r1", Mode: Apply, Role: Maker, Package: "catalog", Project: p, Items: items("catalog/book")})
	contains(t, d.Context, "## Linked from these", "### `members/member`", "A person who borrows. Has a card.")
	lacks(t, d.Context, "### `members/secret`", "## Child")
}

func TestRootSeesItsChildrenOnlyAsTableOfContentsAndOwnsUnclaimedPaths(t *testing.T) {
	p := library(t)
	d := render(t, Input{Run: "r1", Mode: Apply, Role: Maker, Package: "", Project: p, Items: items("project"), Working: []string{"", "catalog"}})
	contains(t, d.Base, "of Root in", "branch `root`", "`.markitect/registry.md`", "every path no other Package claims", "`catalog` (branch `managers/catalog`)")
	lacks(t, d.Base, "Your parent", "`members` (branch")
	contains(t, d.Context, "## Child catalog", "## Child members", "- `members/member`", "- `members/loans/loan`")
	lacks(t, d.Context, "members/secret", "### `members/member`", "Inherited")
	contains(t, d.Task, "| `project` | own | changed |")
}

func TestMakerTaskAndAnswerFormat(t *testing.T) {
	p := library(t)
	in := Input{Run: "r1", Mode: Apply, Role: Maker, Package: "members", Project: p, Items: []Item{
		{Key: "members/member", Part: PartOwn, Definition: "members/member", Detail: "changed | renamed"},
		{Key: "loan-limit", Part: PartInherited, Definition: "loan-limit", Detail: "changed"},
	}}
	d := render(t, in)
	contains(t, d.Task, "| `members/member` | own | changed \\| renamed |", "| `loan-limit` | inherited | changed |",
		"Call `submit` with exactly one verdict per checklist item", "`model-problem`", "escalate")
	lacks(t, d.Task, "Submitted verdicts")
}

func TestCheckerSeesSubmissionVerdictsAndDiff(t *testing.T) {
	p := library(t)
	sub := &Submission{Commit: "abc123", Verdicts: []Verdict{{Item: "members/member", Verdict: "changed", Evidence: "member.go ```x```"}},
		Diff: "diff --git a/src/members/member.go b/src/members/member.go\n+type Member struct{}\n"}
	d := render(t, Input{Run: "r1", Mode: Apply, Role: Checker, Package: "members", Project: p, Items: items("members/member"), Submission: sub})
	contains(t, d.Base, "As checker you re-check", "accepts from you only a commit that changes no file")
	lacks(t, d.Base, "As maker")
	contains(t, d.Task, "submission `abc123` on branch `managers/members`", "| `members/member` | changed | member.go ```x``` |",
		"```diff\ndiff --git a/src/members/member.go", "Call `review`")
	sub.Diff = strings.Repeat("x", MaxInlineDiff+1)
	sub.Files = []string{"M src/members/member.go"}
	d = render(t, Input{Run: "r1", Mode: Apply, Role: Checker, Package: "members", Project: p, Items: items("members/member"), Submission: sub})
	contains(t, d.Task, "`git diff base abc123`", "- M src/members/member.go")
	lacks(t, d.Task, "xxxxxxxx")
}

func TestVerifyCheckerSubmitsVerdictsOnTheBase(t *testing.T) {
	p := library(t)
	d := render(t, Input{Run: "r1", Mode: Verify, Role: Checker, Package: "members", Project: p, Items: items("members/member")})
	contains(t, d.Task, "at the Run's base conforms", "Call `submit`")
	if _, err := Render(Input{Run: "r1", Mode: Verify, Role: Maker, Package: "members", Project: p, Items: items("members/member")}); err == nil {
		t.Error("verify has no makers")
	}
}

func TestRenderRefusesWhatNoAgentCanWorkFrom(t *testing.T) {
	p := library(t)
	ok := Input{Run: "r1", Mode: Apply, Role: Maker, Package: "members", Project: p, Items: items("members/member")}
	cases := map[string]func(*Input){
		"no project":        func(in *Input) { in.Project = nil },
		"unknown Package":   func(in *Input) { in.Package = "nope" },
		"empty checklist":   func(in *Input) { in.Items = nil },
		"pipe in key":       func(in *Input) { in.Items = []Item{{Key: "a|b"}} },
		"checker no commit": func(in *Input) { in.Role = Checker },
		"unknown mode":      func(in *Input) { in.Mode = "improve" },
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
	want := []string{"members/member own", "loan-limit inherited", "finding-1:members/member finding", "stale:members/old stale", "stale:src/members/gone.go stale", "stale:.markitect/registry/old.md stale"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("keys %q, want %q", keys, want)
	}
	got := Items(c)
	if got[2].Detail != "observed: no card; expected: a card" || !strings.Contains(got[3].Detail, ":4 links members/old") || !strings.Contains(got[3].Detail, ":5 links") {
		t.Fatalf("details: %+v", got)
	}
}

func TestToolsPerRole(t *testing.T) {
	names := func(ts []Tool) string {
		var out []string
		for _, tool := range ts {
			out = append(out, tool.Name)
		}
		return strings.Join(out, ",")
	}
	for _, c := range []struct {
		mode     Mode
		role     Role
		children bool
		want     string
	}{
		{Apply, Maker, true, "submit,assign,ask,answer,escalate"},
		{Apply, Maker, false, "submit,ask,answer,escalate"},
		{Apply, Checker, false, "review"},
		{Verify, Checker, false, "submit"},
	} {
		if got := names(Tools(c.mode, c.role, c.children)); got != c.want {
			t.Errorf("%s %s: %s, want %s", c.mode, c.role, got, c.want)
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
