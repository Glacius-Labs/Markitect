package realization

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func testModel(t *testing.T) *model.Model {
	t.Helper()
	kinds := model.LibrarySource{Name: "std", Version: "1.0.0", Dir: "std", Files: []model.File{
		{Path: "Rule.md", Data: []byte("---\nkind: Kind\n---\n# Rule\n")},
	}}
	src := model.Source{Dir: ".markitect/model", Libraries: []model.LibrarySource{kinds}}
	for _, p := range []string{"project", "members/member", "members/loans/loan", "catalog/book"} {
		src.Files = append(src.Files, model.File{Path: p + ".md", Data: []byte("---\nkind: std/Rule\n---\n# " + p + "\n")})
	}
	m, ds := model.Compile(src)
	if len(ds) != 0 {
		t.Fatalf("fixture model: %v", ds)
	}
	return m
}

func page(pkg, body string) PageFile {
	file := ".markitect/registry.md"
	if pkg != "" {
		file = ".markitect/registry/" + pkg + ".md"
	}
	return PageFile{Package: pkg, File: file, Data: []byte(body)}
}

const table = "| File | Realizes | Why |\n|---|---|---|\n"

func account(t *testing.T, files []string, ignored []string, pages ...PageFile) (*Registry, model.Diagnostics) {
	t.Helper()
	in := Input{Model: testModel(t), Pages: pages, Files: files, OwnDir: ".markitect",
		GitIgnored: func(f string) bool {
			for _, i := range ignored {
				if i == f {
					return true
				}
			}
			return false
		}}
	return Account(in)
}

func expect(t *testing.T, ds model.Diagnostics, code, fragment string) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.String(), fragment) {
			return
		}
	}
	t.Fatalf("no %s containing %q in:\n%v", code, fragment, ds)
}

func TestAccountOwnershipAndEntries(t *testing.T) {
	reg, ds := account(t,
		[]string{"README.md", "src/members/member.go", "src/members/loans/loan.go", "docs/members.md", "src/catalog/book.go", "src/members/loans/special.go"},
		nil,
		page("", "# Root\n\n"+table+"| `README.md` | [[project]] [[members/member]] | Shared overview |\n| `src/catalog/book.go` | [[catalog/book]] | No catalog page yet |\n"),
		page("members", "---\npaths:\n  - src/members/**\n  - docs/members.md\n---\n# Members\n\n"+table+
			"| `src/members/member.go` | [[members/member]] | The type |\n| docs/members.md | [[members/member]] | Docs |\n"),
		page("members/loans", "---\npaths:\n  - src/members/loans/**\n  - src/members/loans/special.go\n---\n"+table+
			"| `src/members/loans/loan.go` | [[members/loans/loan]] | Loans |\n| `src/members/loans/special.go` | | Exact path |\n"),
	)
	if len(ds) != 0 {
		t.Fatalf("diagnostics:\n%v", ds)
	}
	want := map[string]string{"README.md": "", "src/members/member.go": "members", "src/members/loans/loan.go": "members/loans", "docs/members.md": "members", "src/catalog/book.go": ""}
	for f, owner := range want {
		if reg.Owner[f] != owner {
			t.Errorf("owner of %s = %q, want %q", f, reg.Owner[f], owner)
		}
	}
	if got := reg.Entries["README.md"].Realizes; strings.Join(got, ",") != "project,members/member" {
		t.Fatalf("README realizes %v", got)
	}
	if e := reg.Entries["src/members/member.go"]; e.Reason != "The type" || e.Line != 10 {
		t.Fatalf("entry = %+v", e)
	}
	if len(reg.Stale) != 0 {
		t.Fatalf("stale = %+v", reg.Stale)
	}
}

func TestAccountUnregisteredFile(t *testing.T) {
	_, ds := account(t, []string{"src/members/new.go", "notes.txt"}, nil,
		page("members", "---\npaths: [src/members/**]\n---\n# Members\n"))
	expect(t, ds, "file.unregistered", "src/members/new.go: file.unregistered: neither registered nor ignored: list it on .markitect/registry/members.md, the page of its owner members")
	expect(t, ds, "file.unregistered", "notes.txt: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root")
}

func TestAccountIgnorePage(t *testing.T) {
	ignore := model.File{Path: ".markitect/ignore.md", Data: []byte("# Ignored\n\n| Path | Why |\n|---|---|\n| `.claude/skills/**` | Installed by markitect init |\n| LICENSE | Legal text |\n")}
	in := Input{Model: testModel(t), Ignore: &ignore, OwnDir: ".markitect",
		Files: []string{".claude/skills/markitect-modeling/SKILL.md", "LICENSE"},
		Pages: []PageFile{page("", table+"| LICENSE | | Also registered |\n")}}
	reg, ds := Account(in)
	expect(t, ds, "registry.ignored_file", "LICENSE is also on the ignore page")
	if !reg.Ignored[".claude/skills/markitect-modeling/SKILL.md"] || len(ds) != 1 {
		t.Fatalf("ignored = %v, diagnostics = %v", reg.Ignored, ds)
	}
}

func TestAccountIgnoreSyntax(t *testing.T) {
	ignore := model.File{Path: ".markitect/ignore.md", Data: []byte("---\nx: 1\n---\n| Path | Why |\n|---|---|\n| /abs | reason |\n| ok.txt | |\n| .markitect/model/x.md | own |\n")}
	_, ds := Account(Input{Model: testModel(t), Ignore: &ignore, OwnDir: ".markitect"})
	expect(t, ds, "ignore.syntax", "no front matter")
	expect(t, ds, "ignore.syntax", `"/abs"`)
	expect(t, ds, "ignore.syntax", "with a reason")
	expect(t, ds, "ignore.syntax", "accounted automatically")
}

func TestAccountDuplicatePatternAcrossPackages(t *testing.T) {
	_, ds := account(t, nil, nil,
		page("members", "---\npaths: [src/shared/**]\n---\n"),
		page("catalog", "---\npaths:\n  - src/shared/**\n  - src/catalog/**\n  - src/catalog/**\n---\n"))
	expect(t, ds, "registry.duplicate_pattern", ".markitect/registry/members.md:2: registry.duplicate_pattern: src/shared/** is already claimed by catalog")
	expect(t, ds, "registry.duplicate_pattern", ".markitect/registry/catalog.md:5: registry.duplicate_pattern: src/catalog/** is already claimed by catalog")
}

func TestAccountAmbiguousOwner(t *testing.T) {
	_, ds := account(t, []string{"src/x.go"}, nil,
		page("members", "---\npaths: [src/*.go]\n---\n"),
		page("catalog", "---\npaths: ['src/x.*o']\n---\n"))
	expect(t, ds, "registry.ambiguous_owner", "src/x.go: registry.ambiguous_owner: equally specific patterns of catalog and members")
}

func TestAccountEntryErrors(t *testing.T) {
	_, ds := account(t, []string{"a.go", "src/members/m.go"}, []string{"build/out.bin"},
		page("", table+
			"| a.go | [[project]] | first |\n"+
			"| a.go | [[project]] | again |\n"+
			"| src/members/m.go | [[members/member]] | wrong page |\n"+
			"| build/out.bin | | ignored by Git |\n"+
			"| .markitect/lock.yaml | | own file |\n"+
			"| ../outside | | bad path |\n"+
			"| b.go | [[members]] [[std/Rule]] [[bad path]] | links |\n"),
		page("members", "---\npaths: [src/members/**]\nowner: x\n---\n"+table+"| x | y |\n"))
	expect(t, ds, "registry.duplicate_entry", "a.go is already listed on .markitect/registry.md line 3")
	expect(t, ds, "registry.wrong_owner", "src/members/m.go is owned by members, so it belongs on .markitect/registry/members.md")
	expect(t, ds, "registry.git_ignored", "build/out.bin is ignored by Git")
	expect(t, ds, "registry.own_file", ".markitect/lock.yaml lies in .markitect")
	expect(t, ds, "registry.table", `"../outside"`)
	expect(t, ds, "registry.link", "[[members]] is not a Definition")
	expect(t, ds, "registry.link", "[[std/Rule]] is not a Definition")
	expect(t, ds, "registry.link", "[[bad path]] is not a model path")
	expect(t, ds, "registry.front_matter", `unknown key "owner"`)
	expect(t, ds, "file.unregistered", "src/members/m.go")
}

func TestAccountPatternErrors(t *testing.T) {
	_, ds := account(t, nil, nil,
		page("", "---\npaths: [src/**]\n---\n"),
		page("members", "---\npaths:\n  - /abs\n  - src/a**b\n  - src/[ab]\n  - - nested\n---\n"),
		page("catalog", "---\npaths: src/**\n---\n"))
	expect(t, ds, "registry.pattern", "Root owns every file")
	expect(t, ds, "registry.pattern", `"/abs"`)
	expect(t, ds, "registry.pattern", "** must be a whole segment")
	expect(t, ds, "registry.pattern", "[ ] { } are not allowed")
	expect(t, ds, "registry.pattern", "each path pattern is a string")
	expect(t, ds, "registry.front_matter", "paths must be a list")
}

func TestAccountStale(t *testing.T) {
	reg, ds := account(t, []string{"src/members/m.go", "src/gone/g.go"}, nil,
		page("members", "---\npaths: [src/members/**]\n---\n"+table+
			"| src/members/m.go | [[members/member]] [[members/removed]] | one link is stale |\n"+
			"| src/members/deleted.go | [[members/member]] | the file is gone |\n"),
		page("gone", "---\npaths: [src/gone/**]\n---\n"+table+"| src/gone/g.go | [[gone/x]] | page of a removed Package |\n"))
	if len(ds) != 0 {
		t.Fatalf("stale realization is no error:\n%v", ds)
	}
	var got []string
	for _, s := range reg.Stale {
		got = append(got, s.Code+" "+s.Package+" "+s.File+" "+s.Target)
	}
	want := []string{
		"stale.page   ",              // the orphaned page itself, for Root
		"stale.page  src/gone/g.go ", // its file, now Root's
		"stale.definition members src/members/m.go members/removed",
		"stale.file members src/members/deleted.go ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("stale:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestPatternMatchAndSpecificity(t *testing.T) {
	cases := []struct {
		pattern, file string
		match         bool
	}{
		{"src/**", "src/a/b.go", true},
		{"src/**", "src", false},
		{"src/**/x.go", "src/x.go", true},
		{"src/**/x.go", "src/a/b/x.go", true},
		{"docs/billing*.md", "docs/billing-api.md", true},
		{"docs/billing*.md", "docs/sub/billing.md", false},
		{"a?c", "abc", true},
		{"README.md", "README.md", true},
		{"README.md", "docs/README.md", false},
	}
	for _, c := range cases {
		if got := (Pattern{Text: c.pattern}).Match(c.file); got != c.match {
			t.Errorf("%s ~ %s = %v", c.pattern, c.file, got)
		}
	}
	if !((Pattern{Text: "src/billing/**"}).Specificity() > (Pattern{Text: "src/**/*.go"}).Specificity()) {
		t.Error("longer literal prefix must be more specific")
	}
	if !((Pattern{Text: "a/b"}).Specificity() > (Pattern{Text: "a/**/b"}).Specificity()) {
		t.Error("an exact path must beat any pattern")
	}
}
