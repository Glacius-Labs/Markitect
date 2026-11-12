package run_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// plan is the slices of a status as one line each: the Packages, the count
// and the command.
func plan(st run.Status) []string {
	var out []string
	for _, s := range st.Slices {
		name := strings.Join(s.Packages, "+")
		if s.Rest {
			name = "rest"
		}
		out = append(out, fmt.Sprintf("%s %d: %s", name, s.Items, s.Command))
	}
	return out
}

func (p *plain) sliced(size int) run.Status {
	p.t.Helper()
	st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, SliceSize: size})
	if err != nil {
		p.t.Fatal(err)
	}
	return st
}

func same(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("slices:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A short list is one slice. A long one goes Package by Package, a Package
// after the one above it and after those it depends on, neighbours joined
// while they stay within the size, and ends with what is verified last: what
// no file realizes, and what a Kind declares so (DEC-112).
func TestALongListIsBuiltInSlices(t *testing.T) {
	p := newPlain(t)
	p.write(m+"kinds/Promise.md", "---\nkind: Kind\nlast: true\n---\n# Promise\n\nJudged on the whole.\n")
	p.write(m+"fast.md", def("kinds/Promise", "Fast", "An answer comes at once."))
	p.write(project.RootPage, table+"| README.md | [[project]], [[fast]] | Overview |\n")

	// Ten items: the Kind and the Rule of Root, two in each Package below,
	// and the Project and the promise, which are verified last.
	st := p.sliced(0)
	if len(st.Items) != 10 {
		t.Fatalf("items:\n%s", st.Text())
	}
	same(t, plan(st), []string{"rest 10: markitect status"})
	has(t, st.Plan(), "The list is built in one slice (10 items):", "1. markitect status\n")

	st = p.sliced(4)
	same(t, plan(st), []string{
		"Root+catalog 4: markitect status --slice Root,catalog",
		"members+members/loans 4: markitect status --slice Root,catalog,members,members/loans",
		"rest 2: markitect status",
	})
	has(t, st.Plan(),
		"The list is built in 3 slices, one after the other, each built and verified before the next:",
		"1. markitect status --slice Root,catalog\n   4 items of the Packages Root and catalog\n",
		"2. markitect status --slice Root,catalog,members,members/loans\n   4 items of the Packages members and members/loans, and whatever of the Packages before is on the list again\n",
		"3. markitect status\n   what is left then: 2 items verified last, on the whole project",
	)
	for _, it := range st.Items {
		if last := it.Definition == "project" || it.Definition == "fast"; it.Last != last {
			t.Errorf("%s: last %v", it.Definition, it.Last)
		}
	}

	// A Package comes after one it depends on, whatever its name.
	p.write(m+"catalog/book.md", def("std/Concept", "Book", "A book a [[members/member]] may borrow."))
	same(t, plan(p.sliced(2)), []string{
		"Root 2: markitect status --slice Root",
		"members 2: markitect status --slice Root,members",
		"catalog 2: markitect status --slice Root,members,catalog",
		"members/loans 2: markitect status --slice Root,members,catalog,members/loans",
		"rest 2: markitect status",
	})

	// A Package with more items than the size is one slice all the same.
	same(t, plan(p.sliced(1))[:2], []string{
		"Root 2: markitect status --slice Root",
		"members 2: markitect status --slice Root,members",
	})
}

// status limited to a slice lists the items of its Packages that are not
// verified last, with the errors in the realization that whoever builds now
// settles, and says how many items the other slices hold and how many
// Definitions no file realizes yet. What is verified drops out, a slice that
// is settled is clean though the list is not, and what waits for the owner
// is no work of a slice.
func TestStatusOfOneSlice(t *testing.T) {
	p := newPlain(t)
	p.write("src/unlisted.go", "package src\n")
	p.write(m+"members/loans/renewal.md", def("std/Concept", "Renewal", "A loan runs on."))
	of := func(names ...string) run.Status {
		t.Helper()
		st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Slice: names})
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	st := of("Root", "catalog")
	if listed(st) != "catalog/book,catalog/catalog,loan-limit" || st.ItemsElsewhere != 6 || st.ErrorsElsewhere != 1 || len(st.Errors) != 1 || st.Errors[0].Code != "file.unregistered" || st.Clean() {
		t.Fatalf("the slice:\n%s", st.Text())
	}
	has(t, st.Text(),
		"To verify in the slice of the Packages Root and catalog (3 of 9 items):",
		"Errors in the realization (1), for whoever builds:",
		"src/unlisted.go",
		"Not listed: 6 items of other slices, 1 Definition that no file realizes yet; markitect status lists all of it.",
	)
	if strings.Contains(st.Text(), "project@") {
		t.Fatalf("the slice lists what is verified last:\n%s", st.Text())
	}
	for _, it := range st.Items {
		if text, ok := p.review(it.Definition, run.VerdictOK, it.Fingerprint, []string{"README.md:1"}, ""); !ok {
			t.Fatal(text)
		}
	}
	// A file no registry page lists is the work of the slice that added it:
	// the slice is not settled with it.
	if st := of("Root", "catalog"); st.Clean() || len(st.Items) != 0 {
		t.Fatalf("a slice with an unlisted file:\n%s", st.Text())
	}
	p.remove("src/unlisted.go")
	st = of("Root", "catalog")
	if !st.Clean() || p.status(false).Clean() {
		t.Fatalf("after verifying the slice:\n%s", st.Text())
	}
	has(t, st.Text(), "Nothing to verify in the slice of the Packages Root and catalog: every item of it is verified at its current inputs.")

	// What a later slice changes puts an item of an earlier one back on the
	// list. The later slice's command names the Packages before its own too,
	// so the item is its work, and the slice is settled only with it.
	p.write("src/members/loans/loan.go", "package loans\n\nconst Limit = 6\n")
	same(t, plan(p.sliced(2)), []string{
		"Root 1: markitect status --slice Root,catalog",
		"members 2: markitect status --slice Root,catalog,members",
		"members/loans 3: markitect status --slice Root,catalog,members,members/loans",
		"rest 1: markitect status",
	})
	if st := of("Root", "catalog", "members"); listed(st) != "loan-limit,members/member,members/members" {
		t.Fatalf("the slice after a change that reaches back:\n%s", st.Text())
	}
	back := p.item(p.status(false), "loan-limit")
	if text, ok := p.review(back.Definition, run.VerdictOK, back.Fingerprint, []string{"README.md:1"}, ""); !ok {
		t.Fatal(text)
	}

	// The plan counts what a builder can settle: a spec-problem waits for
	// the owner.
	it := p.item(p.status(false), "members/member")
	if text, ok := p.review(it.Definition, run.VerdictSpecProblem, it.Fingerprint, nil, "who is a member?"); !ok {
		t.Fatal(text)
	}
	same(t, plan(p.sliced(2)), []string{
		"members 1: markitect status --slice Root,catalog,members",
		"members/loans 3: markitect status --slice Root,catalog,members,members/loans",
		"rest 1: markitect status",
	})
	if st := of("members"); listed(st) != "members/member,members/members" || st.ForTheOwner() {
		t.Fatalf("a slice lists its spec-problem too:\n%s", st.Text())
	}
	// A slice with nothing left but spec-problems says so, whatever the
	// later slices still hold: a Definition no file realizes yet is theirs.
	other := p.item(p.status(false), "members/members")
	if text, ok := p.review(other.Definition, run.VerdictOK, other.Fingerprint, []string{"README.md:1"}, ""); !ok {
		t.Fatal(text)
	}
	if st := of("Root", "catalog", "members"); st.Clean() || !st.ForTheOwner() || st.ErrorsElsewhere != 1 {
		t.Fatalf("a slice with only a spec-problem left:\n%s", st.Text())
	} else {
		has(t, st.Text(), "Only spec-problems are left in the slice of the Packages Root, catalog and members")
	}

	// A Definition gone from the spec is the last slice's: no slice of
	// Packages lists it, and the plan counts it with the rest.
	p.remove(m + "catalog/book.md")
	gone := p.status(false)
	if p.item(gone, "catalog/book").State != run.StateGone {
		t.Fatalf("the book is not gone:\n%s", gone.Text())
	}
	if st := of("Root", "catalog"); strings.Contains(listed(st), "catalog/book") {
		t.Fatalf("a slice lists what is gone:\n%s", st.Text())
	}
	same(t, plan(p.sliced(1)), []string{
		"members/loans 3: markitect status --slice Root,catalog,members,members/loans",
		"rest 2: markitect status",
	})

	if _, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Slice: []string{"Root", "loans"}}); !errors.Is(err, run.ErrUsage) || !strings.Contains(err.Error(), `no Package "loans"`) {
		t.Fatalf("a Package the spec does not have: %v", err)
	}
	// A spec with errors has no list to cut.
	broken := newPlainOf(t, map[string]string{project.ManifestFile: "libraries:\n  std: 2.1.0\n", project.SpecDir + "/x.md": "no front matter\n"})
	st, err := broken.engine.Status(context.Background(), run.StatusRequest{Dir: broken.dir, Slice: []string{"nowhere"}})
	if err != nil || len(st.SpecErrors) == 0 || len(st.Slices) != 0 {
		t.Fatalf("a spec with errors: %v\n%s", err, st.Text())
	}
	has(t, st.Plan(), "so there is no list until it compiles")
}

// Nothing to build is no slice, and neither is a list of nothing but what
// waits for the owner. An error in the realization alone is one slice: a
// builder settles it.
func TestNothingToBuildIsNoSlice(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	st := p.sliced(1)
	if len(st.Slices) != 0 {
		t.Fatalf("slices: %+v", st.Slices)
	}
	has(t, st.Plan(), "Nothing to verify")

	p.write("src/unlisted.go", "package src\n")
	st = p.sliced(1)
	same(t, plan(st), []string{"rest 0: markitect status"})
	has(t, st.Plan(), "The list is built in one slice (0 items and 1 error in the realization):", "1. markitect status\n")
	p.remove("src/unlisted.go")

	p.write(m+"members/member.md", def("std/Concept", "Member", "A person who may borrow."))
	it := p.item(p.status(false), "members/member")
	if text, ok := p.review(it.Definition, run.VerdictSpecProblem, it.Fingerprint, nil, "who may borrow?"); !ok {
		t.Fatal(text)
	}
	for _, other := range p.status(false).Items {
		if other.State == run.StateSpecProblem {
			continue
		}
		if text, ok := p.review(other.Definition, run.VerdictOK, other.Fingerprint, []string{"README.md:1"}, ""); !ok {
			t.Fatal(text)
		}
	}
	st = p.sliced(1)
	if len(st.Slices) != 0 || !st.ForTheOwner() {
		t.Fatalf("only a spec-problem left: %+v\n%s", st.Slices, st.Text())
	}
	has(t, st.Plan(), "Only spec-problems are left")
}
