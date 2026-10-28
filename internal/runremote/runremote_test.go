package runremote

import (
	"strings"
	"testing"
)

func TestTrailersRoundTrip(t *testing.T) {
	sub := Trailers{Kind: Submission, Manager: "members", Verdicts: []Line{
		{"members/member", Conforms, "member.go unchanged; tests pass"},
		{"Finding F-3", ModelProblem, "the model gives two loan limits"},
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
}

func TestMalformedTrailers(t *testing.T) {
	cases := map[string]string{
		"x\n\nMarkitect-Manager: a\n": "alone is no submission",
		"x\n\nMarkitect-Manager: a\nMarkitect-Manager: b\nMarkitect-Verdict: i | changed | e\n":       "appears twice",
		"x\n\nMarkitect-Manager: a\nMarkitect-Check: i | confirmed | e\n":                             "need a Markitect-Review line",
		"x\n\nMarkitect-Manager: a\nMarkitect-Review: maybe\n":                                        "must be \"approved\" or \"rejected\"",
		"x\n\nMarkitect-Manager: a\nMarkitect-Review: approved\nMarkitect-Verdict: i | changed | e\n": "not both",
		"x\n\nMarkitect-Manager: a\nsome prose here\nMarkitect-Verdict: i | changed | e\n":            "every line of the trailer paragraph",
		"x\n\nMarkitect-Manager: a\nMarkitect-Verdict:  | changed | e\n":                              "names no checklist item",
	}
	for msg, want := range cases {
		if _, err := ParseMessage(msg); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want %q", msg, err, want)
		}
	}
}

func TestBranchNames(t *testing.T) {
	for id, want := range map[string]string{".": "root", "members": "managers/members", "members/cards": "managers/members.cards"} {
		if got := Branch(id); got != want || !validBranch(got) {
			t.Errorf("Branch(%q) = %q, want %q", id, got, want)
		}
	}
	for _, bad := range []string{"managers/.hidden", "managers/a..b", "managers/x.lock", "managers/a b", "managers/a:b"} {
		if validBranch(bad) {
			t.Errorf("%q passes as a branch", bad)
		}
	}
}

func TestPatternOwnership(t *testing.T) {
	p := &Plan{Managers: []Manager{
		{ID: "."},
		{ID: "billing", Parent: ".", Patterns: []string{"src/billing/**", "docs/billing*.md"}},
		{ID: "billing/invoices", Parent: "billing", Patterns: []string{"src/billing/invoices/**"}},
		{ID: "docs", Parent: ".", Patterns: []string{"docs/**"}},
	}}
	o := NewPatternOwnership(p)
	for file, want := range map[string]string{
		"src/billing/a.go":            "billing",
		"src/billing/invoices/x/y.go": "billing/invoices",
		"src/billing":                 ".",
		"docs/billing-guide.md":       "billing",
		"docs/other.md":               "docs",
		"README.md":                   ".",
	} {
		if got := o.Owner(file); got != want {
			t.Errorf("Owner(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestPlanValidation(t *testing.T) {
	hash := strings.Repeat("a", 64)
	valid := func() *Plan {
		return &Plan{Run: "r", Mode: Apply, Base: strings.Repeat("1", 40), Protected: []string{".markitect/**"},
			Checks: []Check{{Name: "test", Command: "go test ./...", TimeoutSeconds: 60}},
			Managers: []Manager{
				{ID: ".", Registry: ".markitect/registry/root.md", Working: true, Checklist: []string{"project"}},
				{ID: "a", Parent: ".", Registry: ".markitect/registry/a.md", Working: true, Checklist: []string{"a/x"}},
			},
			Agents: []Agent{{User: "a-maker", Manager: "a", Role: Maker, TokenSHA256: hash}}}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Plan){
		"Root is missing":           func(p *Plan) { p.Managers = p.Managers[1:] },
		"its parent \".\" does not": func(p *Plan) { p.Managers[0].Working, p.Managers[0].Checklist = false, nil },
		"needs a checklist":         func(p *Plan) { p.Managers[1].Checklist = nil },
		"verify has no makers":      func(p *Plan) { p.Mode = Verify },
		"already has a maker": func(p *Plan) {
			p.Agents = append(p.Agents, Agent{User: "b", Manager: "a", Role: Maker, TokenSHA256: hash})
		},
		"branch \"managers/a.b\" is also": func(p *Plan) {
			p.Managers = append(p.Managers, Manager{ID: "a.b", Parent: ".", Registry: "c.md", Working: true, Checklist: []string{"i"}})
			p.Managers[1].ID = "a/b"
			p.Managers[1].Parent = "."
		},
		"must be a full commit id":        func(p *Plan) { p.Base = "main" },
		"timeoutSeconds must be positive": func(p *Plan) { p.Checks[0].TimeoutSeconds = 0 },
		"not a clean relative path":       func(p *Plan) { p.Checks[0].Directory = "../x" },
		"does not work in this Run":       func(p *Plan) { p.Agents[0].Manager = "nobody" },
		"holds '|' or a line break":       func(p *Plan) { p.Managers[1].Checklist = []string{"a | b"} },
		"the parent links form a cycle": func(p *Plan) {
			p.Managers = append(p.Managers, Manager{ID: "b", Parent: "c", Registry: "b.md"}, Manager{ID: "c", Parent: "b", Registry: "c.md"})
		},
		"tokenSha256 must be 64 hex digits":   func(p *Plan) { p.Agents[0].TokenSHA256 = "secret" },
		"registry page \".markitect/registry": func(p *Plan) { p.Managers[1].Registry = p.Managers[0].Registry },
	}
	for want, mutate := range cases {
		p := valid()
		mutate(p)
		if err := p.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v", want, err)
		}
	}
}
