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
		"members+members/loans 4: markitect status --slice members,members/loans",
		"rest 2: markitect status",
	})
	has(t, st.Plan(),
		"The list is built in 3 slices, one after the other, each built and verified before the next:",
		"1. markitect status --slice Root,catalog\n   4 items of the Packages Root and catalog\n",
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
		"members 2: markitect status --slice members",
		"catalog 2: markitect status --slice catalog",
		"members/loans 2: markitect status --slice members/loans",
		"rest 2: markitect status",
	})

	// A Package with more items than the size is one slice all the same.
	same(t, plan(p.sliced(1))[:2], []string{
		"Root 2: markitect status --slice Root",
		"members 2: markitect status --slice members",
	})
}

// status limited to a slice lists the items of its Packages that are not
// verified last, and says how many the other slices hold. What is verified
// drops out, a slice that is settled is clean though the list is not, and
// what waits for the owner is no work of a slice.
func TestStatusOfOneSlice(t *testing.T) {
	p := newPlain(t)
	p.write("src/unlisted.go", "package src\n")
	of := func(names ...string) run.Status {
		t.Helper()
		st, err := p.engine.Status(context.Background(), run.StatusRequest{Dir: p.dir, Slice: names})
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	st := of("Root", "catalog")
	if listed(st) != "catalog/book,catalog/catalog,loan-limit" || st.ItemsElsewhere != 5 || st.ErrorsElsewhere != 1 || len(st.Errors) != 0 || st.Clean() {
		t.Fatalf("the slice:\n%s", st.Text())
	}
	has(t, st.Text(),
		"To verify in the slice of the Packages Root and catalog (3 of 8 items):",
		"Not listed: 5 items of other slices; markitect status lists them.",
		"Not listed: 1 error in the realization",
	)
	if strings.Contains(st.Text(), "project@") {
		t.Fatalf("the slice lists what is verified last:\n%s", st.Text())
	}
	for _, it := range st.Items {
		if text, ok := p.review(it.Definition, run.VerdictOK, it.Fingerprint, []string{"README.md:1"}, ""); !ok {
			t.Fatal(text)
		}
	}
	st = of("Root", "catalog")
	if !st.Clean() || p.status(false).Clean() {
		t.Fatalf("after verifying the slice:\n%s", st.Text())
	}
	has(t, st.Text(), "Nothing to verify in the slice of the Packages Root and catalog: every item of it is verified at its current inputs.")

	// The plan counts what a builder can settle: a spec-problem waits for
	// the owner.
	it := p.item(p.status(false), "members/member")
	if text, ok := p.review(it.Definition, run.VerdictSpecProblem, it.Fingerprint, nil, "who is a member?"); !ok {
		t.Fatal(text)
	}
	same(t, plan(p.sliced(2)), []string{
		"members 1: markitect status --slice members",
		"members/loans 2: markitect status --slice members/loans",
		"rest 1: markitect status",
	})
	if st := of("members"); listed(st) != "members/member,members/members" {
		t.Fatalf("a slice lists its spec-problem too:\n%s", st.Text())
	}

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

// Nothing to build is no slice.
func TestNothingToBuildIsNoSlice(t *testing.T) {
	p := newPlain(t)
	p.verifyAll()
	st := p.sliced(1)
	if len(st.Slices) != 0 {
		t.Fatalf("slices: %+v", st.Slices)
	}
	has(t, st.Plan(), "Nothing to verify")
}
