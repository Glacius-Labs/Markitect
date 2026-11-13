package bootstrap

import (
	"strings"
	"testing"
)

// doc and doc --read take the path of a file or folder of the spec as status
// prints it, with the spec folder's prefix, with or without .md, with a leading
// ./ and with backslashes, and print what the same target without the prefix
// prints. A path outside the spec folder stays unknown.
func TestDocTakesAFilePath(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Library")
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	r.Write(".markitect/spec/loan-limit.md", def("std/Rule", "Loan limit", "At most `5` open loans, for [[members]]."))
	r.Write(".markitect/spec/members/members.md", def("Package", "Members", "Who may borrow."))
	r.Write(".markitect/spec/members/member.md", def("std/Concept", "Member", "A person who borrows."))
	r.Write(".markitect/spec/members/rules/fees/fees.md", def("std/Rule", "Fees", "A [[members/member]] pays."))
	r.Write(".markitect/spec/members/rules/fees/amounts/per-day.md", def("std/Rule", "Per day", "`1` euro a day."))

	for _, c := range []struct {
		name string
		id   string
		path []string
	}{
		{"a Definition", "loan-limit", []string{".markitect/spec/loan-limit.md", ".markitect/spec/loan-limit", "./.markitect/spec/loan-limit.md", `.markitect\spec\loan-limit.md`}},
		{"a Definition in a folder", "members/member", []string{".markitect/spec/members/member.md", `.\.markitect\spec\members\member`}},
		{"a thing with parts by its folder", "members/rules/fees", []string{".markitect/spec/members/rules/fees", ".markitect/spec/members/rules/fees/", ".markitect/spec/members/rules/fees/fees.md"}},
		{"a part of a thing", "members/rules/fees/amounts/per-day", []string{".markitect/spec/members/rules/fees/amounts/per-day.md"}},
		{"a Package", "members", []string{".markitect/spec/members", ".markitect/spec/members/"}},
		{"the anchor of a Package", "members/members", []string{".markitect/spec/members/members.md"}},
		{"the Root", "Root", []string{".markitect/spec", ".markitect/spec/"}},
	} {
		want := mustRun(t, r.Dir, "doc", c.id)
		if want == "" {
			t.Fatalf("doc %s printed nothing", c.id)
		}
		for _, p := range c.path {
			if got := mustRun(t, r.Dir, "doc", p); got != want {
				t.Errorf("doc %s of %s differs from doc %s:\n%s\n%s", p, c.name, c.id, got, want)
			}
		}
	}

	for _, c := range []struct {
		id   string
		path []string
	}{
		{"members", []string{".markitect/spec/members", `.markitect\spec\members\`, "./.markitect/spec/members/"}},
		{"members/rules/fees", []string{".markitect/spec/members/rules/fees"}},
	} {
		want := mustRun(t, r.Dir, "doc", "--read", c.id)
		if !strings.Contains(want, "members/") {
			t.Fatalf("doc --read %s:\n%s", c.id, want)
		}
		for _, p := range c.path {
			if got := mustRun(t, r.Dir, "doc", "--read", p); got != want {
				t.Errorf("doc --read %s differs from doc --read %s:\n%s\n%s", p, c.id, got, want)
			}
		}
	}
	if want := mustRun(t, r.Dir, "doc", "--read"); mustRun(t, r.Dir, "doc", "--read", ".markitect/spec") != want {
		t.Error("doc --read of the spec folder differs from doc --read")
	}

	// Outside the spec folder, a path is no target.
	for _, p := range []string{".markitect/libraries/std/Rule.md", ".markitect/registry.md", ".markitect/specs/loan-limit.md", "elsewhere/.markitect/spec/loan-limit.md"} {
		if code, _, errOut := run(t, r.Dir, "doc", p); code != 1 || !strings.Contains(errOut, "doc.unknown") {
			t.Errorf("doc %s: exit %d, %s", p, code, errOut)
		}
		if code, _, errOut := run(t, r.Dir, "doc", "--read", p); code != 1 || !strings.Contains(errOut, "doc.unknown") {
			t.Errorf("doc --read %s: exit %d, %s", p, code, errOut)
		}
	}
	if code, _, errOut := run(t, r.Dir, "doc", ".markitect/spec/nobody.md"); code != 1 || !strings.Contains(errOut, "doc.unknown") {
		t.Errorf("doc of no file of the spec: exit %d, %s", code, errOut)
	}
}

// doc --read --bound prints the folder's text between what binds it from
// outside and what outside it it links, each once, with --json too; --bound
// without --read is refused.
func TestDocReadBound(t *testing.T) {
	emptyHome(t)
	r := newFolder(t)
	mustRun(t, r.Dir, "init", "--name", "Library")
	def := func(kind, title, body string) string {
		return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
	}
	r.Write(".markitect/spec/loan-limit.md", def("std/Rule", "Loan limit", "At most `5` open loans."))
	r.Write(".markitect/spec/members/members.md", def("Package", "Members", "Who may borrow."))
	r.Write(".markitect/spec/members/member.md", def("std/Concept", "Member", "A person at the [[places/desk]], a [[std/Concept]]."))
	r.Write(".markitect/spec/members/card.md", def("std/Concept", "Card", "What a [[members/member]] shows at the [[places/desk]]."))
	r.Write(".markitect/spec/places/places.md", def("Package", "Places", "Where lending happens."))
	r.Write(".markitect/spec/places/desk.md", def("std/Concept", "Desk", "Where a book is handed over."))

	out := mustRun(t, r.Dir, "doc", "--read", "members", "--bound")
	for _, id := range []string{"`loan-limit`", "`members/member`", "`places/desk`"} {
		if strings.Count(out, id) != 1 {
			t.Errorf("%s %d times:\n%s", id, strings.Count(out, id), out)
		}
	}
	if !(strings.Index(out, "`loan-limit`") < strings.Index(out, "`members/member`") && strings.Index(out, "`members/member`") < strings.Index(out, "`places/desk`")) {
		t.Errorf("order:\n%s", out)
	}
	if strings.Contains(out, "`std/Concept`") {
		t.Errorf("a Kind of a Library:\n%s", out)
	}
	if js := mustRun(t, r.Dir, "doc", "--json", "--read", "members", "--bound"); !strings.Contains(js, `"bound_by"`) || !strings.Contains(js, `"linked"`) {
		t.Errorf("JSON:\n%s", js)
	}
	if code, _, errOut := run(t, r.Dir, "doc", "--bound", "members"); code != 2 || !strings.Contains(errOut, "--bound goes with --read") {
		t.Errorf("doc --bound: exit %d, %s", code, errOut)
	}
}
