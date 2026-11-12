package run_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// status names below its list what binds the listed items without a link, by
// Package, for the list as it is printed: the whole list, --all, one Package and
// one slice each name what binds their own items. A list with no bound item
// names nothing.

const bindingHeading = "What binds these items without a link; each is verified against it, and markitect doc ID prints one:\n\n"

// withCards is the fixture project with a Rule in members, beside the Rule of
// Root.
func withCards(t *testing.T) *plain {
	t.Helper()
	files := library()
	files[m+"members/rules/cards.md"] = def("std/Rule", "Cards", "Every member carries a card.")
	files[project.RegistryDir+"/members.md"] += "| src/members/cards.go | [[members/rules/cards]] | Cards |\n"
	files["src/members/cards.go"] = "package members\n"
	return newPlainOf(t, files)
}

func (p *plain) statusOf(req run.StatusRequest) run.Status {
	p.t.Helper()
	req.Dir = p.dir
	st, err := p.engine.Status(context.Background(), req)
	if err != nil {
		p.t.Fatal(err)
	}
	return st
}

// names is the binding as one line per Package.
func names(st run.Status) string {
	var lines []string
	for _, b := range st.Binding {
		lines = append(lines, b.Package+": "+strings.Join(b.Definitions, ", "))
	}
	return strings.Join(lines, "\n")
}

func TestStatusNamesWhatBindsTheListedItems(t *testing.T) {
	p := withCards(t)

	// The whole list: Root and members, each with its binding things, below the
	// items and above the errors.
	st := p.status(false)
	if got, want := names(st), "Root: loan-limit\nmembers: members/rules/cards"; got != want {
		t.Fatalf("binding of the list:\n%s\nwant\n%s", got, want)
	}
	block := bindingHeading + "Root: loan-limit\nmembers: members/rules/cards\n"
	has(t, st.Text(), "\n"+block)
	if text := st.Text(); !strings.HasSuffix(text, block) || strings.Index(text, "To verify") > strings.Index(text, bindingHeading) {
		t.Fatalf("the block is not below the items:\n%s", text)
	}
	p.write("stray.txt", "not in the registry\n")
	text := p.status(false).Text()
	if at, errs := strings.Index(text, bindingHeading), strings.Index(text, "Errors in the realization"); at < 0 || errs < at {
		t.Fatalf("the block does not stand before the errors:\n%s", text)
	}
	p.remove("stray.txt")

	// One Package names what binds its own items: catalog is bound by Root alone,
	// members by both.
	st = p.statusOf(run.StatusRequest{Package: "catalog"})
	if got := names(st); got != "Root: loan-limit" {
		t.Fatalf("binding of catalog:\n%s", got)
	}
	has(t, st.Text(), "\n"+bindingHeading+"Root: loan-limit\n")
	if got := names(p.statusOf(run.StatusRequest{Package: "members/loans"})); got != "Root: loan-limit\nmembers: members/rules/cards" {
		t.Fatalf("binding of members/loans:\n%s", got)
	}

	// A slice names what binds the items of its Packages.
	if got := names(p.statusOf(run.StatusRequest{Slice: []string{"catalog"}})); got != "Root: loan-limit" {
		t.Fatalf("binding of the slice catalog:\n%s", got)
	}
	if got := names(p.statusOf(run.StatusRequest{Slice: []string{"members", "catalog"}})); got != "Root: loan-limit\nmembers: members/rules/cards" {
		t.Fatalf("binding of the slice members and catalog:\n%s", got)
	}

	// --all lists everything again, and names the same.
	p.verifyAll()
	if got := names(p.statusOf(run.StatusRequest{All: true})); got != "Root: loan-limit\nmembers: members/rules/cards" {
		t.Fatalf("binding of --all:\n%s", got)
	}

	// Verified, the list is empty and prints no block.
	st = p.status(false)
	if len(st.Binding) != 0 || strings.Contains(st.Text(), "What binds") {
		t.Fatalf("an empty list names a binding:\n%s", st.Text())
	}

	// A list of items nothing binds, as the Project that no file realizes,
	// names nothing.
	p.write(m+"project.md", def("std/Project", "Library", "Parts: [[members]], [[catalog]]. Changed."))
	st = p.status(false)
	if got := listed(st); got != "project" {
		t.Fatalf("listed %q:\n%s", got, st.Text())
	}
	if len(st.Binding) != 0 || strings.Contains(st.Text(), "What binds") {
		t.Fatalf("an unbound item names a binding:\n%s", st.Text())
	}

	// A changed Rule lists what it binds, and the block still says what to read:
	// catalog is not listed, and Root's Rule binds the listed items of members.
	p.verifyAll()
	p.write(m+"members/rules/cards.md", def("std/Rule", "Cards", "Every member carries a card with a photo."))
	st = p.status(false)
	if got := names(st); got != "Root: loan-limit\nmembers: members/rules/cards" {
		t.Fatalf("binding after a change of the Rule of members:\n%s", got)
	}
}
