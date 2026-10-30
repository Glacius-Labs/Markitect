package runremote

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func TestTrailersRoundTrip(t *testing.T) {
	sub := Trailers{Kind: Submission, Manager: "members", Verdicts: []Line{
		{"members/member", Conforms, "member.go unchanged; tests pass"},
		{"finding 1 on members/member", ModelProblem, "the model gives two loan limits"},
	}}
	got, err := ParseMessage("Realize members\n\nFree text.\n\n" + sub.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != Submission || got.Manager != "members" || len(got.Verdicts) != 2 || got.Verdicts[1] != sub.Verdicts[1] {
		t.Fatalf("parsed %+v", got)
	}
	rev := Trailers{Kind: Review, Manager: ".", Review: Rejected, Checks: []Line{{"project", Disputed, "README still names the old limit | see line 3"}}}
	got, err = ParseMessage("Review\n\n" + rev.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != Review || got.Review != Rejected || got.Checks[0].Evidence != "README still names the old limit | see line 3" {
		t.Fatalf("parsed %+v", got)
	}
}

func TestTrailersBesideGitTrailersAndContinuationLines(t *testing.T) {
	msg := "Work\n\nMarkitect-Manager: loans\nMarkitect-Verdict: loans/loan | changed | limit 3,\n  tested in loan_test.go\nCo-Authored-By: Someone <a@b>\n"
	got, err := ParseMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdicts[0].Evidence != "limit 3, tested in loan_test.go" {
		t.Fatalf("evidence %q", got.Verdicts[0].Evidence)
	}
	if got, err := ParseMessage("Plain work\n\nCo-Authored-By: Someone <a@b>\n"); err != nil || got.Kind != Plain {
		t.Fatalf("plain message: %+v %v", got, err)
	}
	// A parent that only integrates has no items: its Manager line alone submits.
	if got, err := ParseMessage("Integrate\n\nMarkitect-Manager: .\n"); err != nil || got.Kind != Submission || len(got.Verdicts) != 0 {
		t.Fatalf("submission without items: %+v %v", got, err)
	}
}

func TestMalformedTrailers(t *testing.T) {
	cases := map[string]string{
		"x\n\nMarkitect-Manager: a\nMarkitect-Manager: b\nMarkitect-Verdict: i | changed | e\n":       "appears twice",
		"x\n\nMarkitect-Manager: a\nMarkitect-Check: i | confirmed | e\n":                             "need a Markitect-Review line",
		"x\n\nMarkitect-Manager: a\nMarkitect-Review: maybe\n":                                        "must be \"approved\" or \"rejected\"",
		"x\n\nMarkitect-Manager: a\nMarkitect-Review: approved\nMarkitect-Verdict: i | changed | e\n": "not both",
		"x\n\nMarkitect-Manager: a\nsome prose here\nMarkitect-Verdict: i | changed | e\n":            "every line of the trailer paragraph",
		"x\n\nMarkitect-Manager: a\nMarkitect-Verdict:  | changed | e\n":                              "names no checklist item",
		// Git matches trailer keys without case, so another case is refused, not ignored.
		"x\n\nMarkitect-Manager: a\nmarkitect-review: approved\nMarkitect-Check: i | confirmed | e\n":           "use exactly",
		"x\n\nMARKITECT-VERDICT: i | changed | e\n\nMarkitect-Manager: a\nMarkitect-Verdict: i | changed | e\n": "must form the message's last paragraph",
	}
	for msg, want := range cases {
		if _, err := ParseMessage(msg); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want %q", msg, err, want)
		}
	}
}

func TestBranchNames(t *testing.T) {
	for id, want := range map[string]string{model.RootPackage: "root", "members": "managers/members", "members/cards": "managers/members.cards", "root": "managers/root"} {
		if got := Branch(id); got != want || !validBranch(got) {
			t.Errorf("Branch(%q) = %q, want %q", id, got, want)
		}
	}
	for _, bad := range []string{"managers/.hidden", "managers/a..b", "managers/x.lock", "managers/a b", "managers/a:b"} {
		if validBranch(bad) {
			t.Errorf("%q passes as a branch", bad)
		}
	}
	if Name(model.RootPackage) != "." || Name("members") != "members" {
		t.Fatal("Name")
	}
}

func TestOwnershipFollowsTheRegistry(t *testing.T) {
	p := &Plan{Managers: []Manager{
		{ID: ""},
		{ID: "billing", Patterns: []string{"src/billing/**", "docs/billing*.md"}},
		{ID: "billing/invoices", Patterns: []string{"src/billing/invoices/**"}},
		{ID: "docs", Patterns: []string{"docs/**"}},
		{ID: "other", Patterns: []string{"src/a?/**"}},
		{ID: "third", Patterns: []string{"src/?b/**"}},
	}}
	o := NewOwnership(p)
	for file, want := range map[string]string{
		"src/billing/a.go":            "billing",
		"src/billing/invoices/x/y.go": "billing/invoices",
		"src/billing":                 "",
		"docs/billing-guide.md":       "billing",
		"docs/other.md":               "docs",
		"README.md":                   "",
	} {
		if got, tied := o.Owner(file); got != want || tied != nil {
			t.Errorf("Owner(%q) = %q %v, want %q", file, got, tied, want)
		}
	}
	if _, tied := o.Owner("src/ab/x.go"); !slices.Equal(tied, []string{"other", "third"}) {
		t.Fatalf("a tie between Packages: %v", tied)
	}
}

// memRepo is a project state in memory.
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

func load(t *testing.T, r memRepo) *project.Project {
	t.Helper()
	p, err := project.Load(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Errors) != 0 {
		t.Fatalf("fixture does not compile:\n%v", p.Errors)
	}
	return p
}

func TestPlanForAProjectAndItsImpact(t *testing.T) {
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	table := "| File | Realizes | Why |\n|---|---|---|\n"
	m := project.ModelDir + "/"
	base := memRepo{
		project.ManifestFile:                "libraries:\n  std: 1.0.0\n",
		project.LockFile:                    string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                  "checks:\n  - name: test\n    command: go test ./...\n    timeout: 5m\n    environment: golang:1.27\n",
		m + "project.md":                    def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "members/member.md":             def("std/Concept", "Member", "A person."),
		m + "catalog/book.md":               def("std/Concept", "Book", "A book."),
		m + "shelf/shelf.md":                def("std/Concept", "Shelf", "Holds books."),
		project.RootPage:                    table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md": "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n",
		project.RegistryDir + "/catalog.md": "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n",
		"README.md":                         "# Library\n",
		"src/members/member.go":             "package members\n",
		"src/catalog/book.go":               "package catalog\n",
	}
	cur := memRepo{}
	for k, v := range base {
		cur[k] = v
	}
	cur[m+"members/member.md"] = def("std/Concept", "Member", "A person with a card.")
	bp, cp := load(t, base), load(t, cur)
	imp := analysis.ComputeImpact(analysis.ImpactInput{Base: bp, Current: cp,
		Findings: []analysis.Finding{{Definition: "members/member", Observed: "x", Expected: "y"}}})
	plan, err := PlanFor("r1", Apply, strings.Repeat("a", 40), bp, imp)
	if err != nil {
		t.Fatal(err)
	}
	members, _ := plan.Manager("members")
	catalog, _ := plan.Manager("catalog")
	root, _ := plan.Manager(model.RootPackage)
	if !members.Working || !slices.Equal(members.Checklist, []string{"members/member", "finding 1 on members/member"}) || !slices.Equal(members.Patterns, []string{"src/members/**"}) {
		t.Fatalf("members: %+v", members)
	}
	if catalog.Working || !root.Working {
		t.Fatalf("catalog %+v, Root %+v", catalog, root)
	}
	if plan.Pages[project.RootPage] != model.RootPackage || plan.Pages[project.RegistryDir+"/catalog.md"] != "catalog" {
		t.Fatalf("pages %v", plan.Pages)
	}
	// A Package without a page at the base gets none: its first page and
	// patterns are the owner's (DEC-081).
	if _, ok := plan.Manager("shelf"); !ok {
		t.Fatal("no Manager for shelf")
	}
	if owner, ok := plan.Pages[project.RegistryDir+"/shelf.md"]; ok {
		t.Fatalf("shelf's missing page is mapped to %q", owner)
	}
	if plan.Image() != "golang:1.27" || plan.Checks[0].Timeout != 5*time.Minute {
		t.Fatalf("checks %+v", plan.Checks)
	}
	plan.Agents = []Agent{{User: "members-maker", Manager: "members", Role: Maker, TokenSHA256: strings.Repeat("0", 64)}}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestItemsNameEveryPartOfAChecklist(t *testing.T) {
	got := Items(analysis.Checklist{
		Own: []analysis.Item{{Definition: "a/x"}}, Inherited: []analysis.Item{{Definition: "rule"}},
		Findings: []analysis.Finding{{Definition: "a/x"}, {Definition: "a/x"}},
		Stale: []realization.Stale{
			{Code: "stale.definition", PageFile: ".markitect/registry/a.md", Line: 7, Target: "a/old"},
			{Code: "stale.file", PageFile: ".markitect/registry/a.md", Line: 8, File: "src/a/gone.go"},
			{Code: "stale.page", PageFile: ".markitect/registry/b.md"},
		},
	})
	want := []string{"a/x", "rule", "finding 1 on a/x", "finding 2 on a/x",
		"stale.definition a/old at .markitect/registry/a.md:7", "stale.file src/a/gone.go at .markitect/registry/a.md:8", "stale.page .markitect/registry/b.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("items\n%q\nwant\n%q", got, want)
	}
}

func TestPlanValidation(t *testing.T) {
	hash := strings.Repeat("a", 64)
	valid := func() *Plan {
		return &Plan{Run: "r", Mode: Apply, Base: strings.Repeat("1", 40),
			Checks: []project.Check{{Name: "test", Command: "go test ./...", Timeout: time.Minute, Environment: "golang:1.27"}},
			Pages:  map[string]string{project.RootPage: "", project.RegistryDir + "/a.md": "a"},
			Managers: []Manager{
				{ID: "", Working: true, Checklist: []string{"project"}},
				{ID: "a", Working: true, Checklist: []string{"a/x"}},
			},
			Agents: []Agent{{User: "a-maker", Manager: "a", Role: Maker, TokenSHA256: hash}}}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Plan){
		"Root is missing":           func(p *Plan) { p.Managers = p.Managers[1:] },
		"its parent \".\" does not": func(p *Plan) { p.Managers[0].Working, p.Managers[0].Checklist = false, nil },
		"verify has no makers":      func(p *Plan) { p.Mode = Verify },
		"already has a maker": func(p *Plan) {
			p.Agents = append(p.Agents, Agent{User: "b", Manager: "a", Role: Maker, TokenSHA256: hash})
		},
		"not a Package path below":       func(p *Plan) { p.Managers = append(p.Managers, Manager{ID: "x/y"}) },
		"must be a full SHA-1 commit id": func(p *Plan) { p.Base = strings.Repeat("1", 64) },
		"timeout must be positive":       func(p *Plan) { p.Checks[0].Timeout = 0 },
		"not a clean relative path":      func(p *Plan) { p.Checks[0].Directory = "../x" },
		"names no image":                 func(p *Plan) { p.Checks[0].Environment = "" },
		"runs every check in one image": func(p *Plan) {
			p.Checks = append(p.Checks, project.Check{Name: "lint", Command: "x", Timeout: time.Minute, Environment: "node:22"})
		},
		"does not work in this Run":      func(p *Plan) { p.Agents[0].Manager = "nobody" },
		"holds '|' or a line break":      func(p *Plan) { p.Managers[1].Checklist = []string{"a | b"} },
		"tokenSha256 must be 64":         func(p *Plan) { p.Agents[0].TokenSHA256 = "secret" },
		"not a registry page":            func(p *Plan) { p.Pages[".markitect/model/a.md"] = "a" },
		"Package \"b\" is not a Manager": func(p *Plan) { p.Pages[project.RegistryDir+"/b.md"] = "b" },
		"pattern \"src/[ab]/**\"":        func(p *Plan) { p.Managers[1].Patterns = []string{"src/[ab]/**"} },
	}
	for want, mutate := range cases {
		p := valid()
		mutate(p)
		if err := p.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v", want, err)
		}
	}
}
