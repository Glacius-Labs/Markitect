package lang

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// lines renders what Check finds as path:line: code: message.
func lines(t *testing.T, s *Spec) []string {
	t.Helper()
	out := []string{}
	for _, d := range Check(s) {
		out = append(out, d.String())
	}
	return out
}

func checkFolder(t *testing.T, folder string) []string {
	t.Helper()
	s, err := Load(os.DirFS(folder))
	if err != nil {
		t.Fatal(err)
	}
	return lines(t, s)
}

// checkFiles checks a spec of the given files, path to content.
func checkFiles(t *testing.T, files map[string]string) []string {
	t.Helper()
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	s, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return lines(t, s)
}

func expect(t *testing.T, got []string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestCheckFixtures(t *testing.T) {
	missing := "type line must be exactly one link to a document"
	cases := map[string][]string{
		// A clean spec of members and branches: a heading in another case, sections in
		// another order, a name with a space linked as %20 and as <...> with a title and a
		// section, and links the check leaves alone: web, mail and ftp addresses, a bare
		// #section, an image, a code span and a fenced block, all to missing files.
		"valid": nil,
		"title": {
			"two-titles.md:3: title: second level-1 heading",
			"untitled.md:1: title: missing level-1 title",
		},
		"type.unresolved": {`loan.md:3: type.unresolved: no document at "reservation.md"`},
		"type.line":       {"reader.md:3: type.line: " + missing, "remote.md:3: type.line: " + missing, "visitor.md:3: type.line: " + missing},
		"section.missing": {`ada.md:3: section.missing: missing section "Card" required by member.md`},
		"section.empty":   {`ada.md:5: section.empty: empty section "Card"`},
		// Paths are relative to the linking document, a path above the spec folder names
		// nothing, and a path linked twice, once with a section, is one report at its
		// first line.
		"link.unresolved": {
			`catalog.md:3: link.unresolved: no document at "archive.md"`,
			`catalog.md:5: link.unresolved: no document at "../city.md"`,
			`shelves/fiction.md:3: link.unresolved: no document at "catalog.md"`,
		},
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) { expect(t, checkFolder(t, "testdata/"+name), want...) })
	}
	s, err := Load(os.DirFS("testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, d := range s.Documents {
		paths = append(paths, d.Path)
	}
	expect(t, paths, "branches/central.md", "members/Grace Hopper.md", "members/ada-lovelace.md",
		"members/alan-turing.md", "types/branch.md", "types/member.md")
}

const member = "# Member\n\nA person.\n\n## Card\nThe card number.\n"

func TestFiles(t *testing.T) {
	t.Run("one file name in two folders", func(t *testing.T) {
		expect(t, checkFiles(t, map[string]string{
			"fiction/index.md": "# Fiction\n",
			"poetry/index.md":  "# Poetry\n\nNext to [fiction](../fiction/index.md).\n",
		}))
	})
	t.Run("only .md files are documents", func(t *testing.T) {
		s, _ := Load(fstest.MapFS{
			"a.md":        {Data: []byte("# A\n\n[cover](cover.jpg) [list](titles.csv)\n")},
			"readme.txt":  {Data: []byte("no title")},
			".md":         {Data: []byte("no title")},
			"B.MD":        {Data: []byte("# B\n")},
			"dir.md/c.md": {Data: []byte("# C\n")},
		})
		var paths []string
		for _, d := range s.Documents {
			paths = append(paths, d.Path)
		}
		expect(t, paths, "B.MD", "a.md", "dir.md/c.md")
		expect(t, lines(t, s))
	})
	t.Run("byte order mark and CRLF", func(t *testing.T) {
		expect(t, checkFiles(t, map[string]string{
			"member.md": "\xef\xbb\xbf# Member\r\n\r\n## Card\r\nThe number.\r\n",
			"ada.md":    "\xef\xbb\xbf# Ada\r\n\r\nType: [Member](member.md)\r\n\r\n[x](ghost.md)\r\n\r\n## Card\r\n\r\n",
		}), `ada.md:5: link.unresolved: no document at "ghost.md"`, `ada.md:7: section.empty: empty section "Card"`)
	})
	t.Run("CR line ends", func(t *testing.T) {
		expect(t, checkFiles(t, map[string]string{
			"member.md": member,
			"ada.md":    "# Ada\r\rType: [Member](member.md)\r\r## Card\r1\r[x](ghost.md)\r",
		}), `ada.md:7: link.unresolved: no document at "ghost.md"`)
	})
	t.Run("invalid UTF-8 is replaced", func(t *testing.T) {
		expect(t, checkFiles(t, map[string]string{"a.md": "# A\xff\n\n[x](gh\xffost.md)\n"}),
			"a.md:3: link.unresolved: no document at \"gh\uFFFDost.md\"")
	})
	t.Run("empty and blank files have no title", func(t *testing.T) {
		expect(t, checkFiles(t, map[string]string{"blank.md": "", "spaces.md": "  \n\t\n", "late.md": "\n\ntext\n\n# Late\n"}),
			"blank.md:1: title: missing level-1 title", "late.md:3: title: missing level-1 title",
			"spaces.md:1: title: missing level-1 title")
	})
}

func TestTitle(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{
		"no-space.md":   "#Title\n",
		"three.md":      "   # Three\n",
		"four.md":       "    # Four\n",
		"seven.md":      "####### Seven\n",
		"empty.md":      "# \n",
		"setext.md":     "Title\n=====\n",
		"fenced.md":     "```\n# In code\n```\n",
		"tab.md":        "#\tTab\n",
		"closing.md":    "# Title #\n",
		"code-h1.md":    "# T\n\n```\n# code\n```\n\n~~~~\n# code\n~~~\n# still code\n~~~~\n",
		"two.md":        "# One\n\n## S\n\n# Two\n\n# Three\n",
		"title-link.md": "# [T](ghost.md)\n\n## [S](ghost2.md)\n",
	}),
		"empty.md:1: title: missing level-1 title",
		"fenced.md:1: title: missing level-1 title",
		"four.md:1: title: missing level-1 title",
		"no-space.md:1: title: missing level-1 title",
		"setext.md:1: title: missing level-1 title",
		"seven.md:1: title: missing level-1 title",
		// The title line is not searched for links; a heading line is.
		`title-link.md:3: link.unresolved: no document at "ghost2.md"`,
		"two.md:5: title: second level-1 heading",
	)
}

func TestTypeLine(t *testing.T) {
	role := "# Role\n\nA role.\n\n## Duty\nWhat.\n"
	bad := "type line must be exactly one link to a document"
	expect(t, checkFiles(t, map[string]string{
		"role.md":     role,
		"extra.md":    "# A\n\nType: [Role](role.md), see below\n\n## Duty\nx\n",
		"no-space.md": "# A\n\nType:[Role](role.md)\n\n## Duty\nx\n",
		"title.md":    "# A\n\nType: [Role](role.md \"t\")   \n\n## Duty\nx\n",
		"padded.md":   "# A\n\nType: [Role]( role.md )\n\n## Duty\nx\n",
		"nested.md":   "# A\n\nType: [[Role]](role.md#duty)\n\n## Duty\nx\n",
		"angle.md":    "# A\n\nType: <role.md>\n",
		"image.md":    "# A\n\nType: ![Role](role.md)\n",
		"code.md":     "# A\n\nType: `[Role](role.md)`\n",
		// Only the first line with content after the title is the type line, written
		// exactly "Type:" at the start; elsewhere such a line is text with a link.
		"lower.md":    "# A\n\ntype: [Role](ghost.md)\n",
		"indented.md": "# A\n\n Type: [Role](ghost.md)\n",
		"late.md":     "# A\n\ntext\nType: [Role](ghost.md)\n",
		"fenced.md":   "# A\n\n```\nType: [Role](role.md)\n```\n",
		"untitled.md": "Hello\nType: [Role](role.md)\n",
		// The link of the type line is not reported again as a link.
		"unresolved.md": "# A\n\nType: [Ghost](ghost-type.md)\n",
	}),
		"angle.md:3: type.line: "+bad,
		"code.md:3: type.line: "+bad,
		"extra.md:3: type.line: "+bad,
		"image.md:3: type.line: "+bad,
		`indented.md:3: link.unresolved: no document at "ghost.md"`,
		`late.md:4: link.unresolved: no document at "ghost.md"`,
		`lower.md:3: link.unresolved: no document at "ghost.md"`,
		`unresolved.md:3: type.unresolved: no document at "ghost-type.md"`,
		"untitled.md:1: title: missing level-1 title",
	)
}

func TestSections(t *testing.T) {
	typ := "# Role\n\nA role.\n\n## Duty\nWhat.\n\n## Size ##\nS.\n"
	expect(t, checkFiles(t, map[string]string{
		"role.md": typ,
		// A level-3 heading, code or a fence line is text of the section; the closing #
		// of a heading belong to its text.
		"h3.md":    "# A\n\nType: [Role](role.md)\n\n## Duty\n### Part\n## size ##\nz\n",
		"fence.md": "# A\n\nType: [Role](role.md)\n\n## duty\n```\n```\n## SIZE ##\nz\n",
		// A level-1 heading ends the section before it.
		"h1.md": "# A\n\nType: [Role](role.md)\n\n## Duty\n# Break\ntext\n## Size ##\nz\n",
		// The first of two sections with the same heading counts.
		"twice.md": "# A\n\nType: [Role](role.md)\n\n## Duty\n\n## Duty\nsecond\n## Size ##\nz\n",
		// No-break space is white space.
		"blank.md": "# A\n\nType: [Role](role.md)\n\n## Duty\n\u00a0\n## Size\nz\n",
		// Headings in code are no sections.
		"code.md": "# A\n\nType: [Role](role.md)\n\n```\n## Duty\na sample\n```\n\n`## Size ##`\n",
	}),
		`blank.md:3: section.missing: missing section "Size ##" required by role.md`,
		`blank.md:5: section.empty: empty section "Duty"`,
		`code.md:3: section.missing: missing section "Duty" required by role.md`,
		`code.md:3: section.missing: missing section "Size ##" required by role.md`,
		`h1.md:5: section.empty: empty section "Duty"`,
		"h1.md:6: title: second level-1 heading",
		`twice.md:5: section.empty: empty section "Duty"`,
	)
}

func TestOptionalSections(t *testing.T) {
	reservation := "# Reservation\n\nA hold on one copy for one member.\n\n## Copy\nThe copy that is held.\n\n" +
		"## Pickup branch (optional)\nWhere the member collects it.\n\n## Reminder (OPTIONAL)\nWhen the member hears.\n\n## Expiry(optional)\nE.\n"
	cases := []struct {
		name, doc string
		want      []string
	}{
		{"absent", "## Copy\nC-17.\n", nil},
		{"present without the marker, in any case", "## Copy\nC-17.\n\n## pickup BRANCH\nCentral.\n## reminder\nBy mail.\n", nil},
		{"present but empty", "## Copy\nC-17.\n\n## Pickup branch\n\n", []string{`hold.md:8: section.empty: empty section "Pickup branch"`}},
		{"required stays required", "## Pickup branch\nCentral.\n", []string{`hold.md:3: section.missing: missing section "Copy" required by reservation.md`}},
		{"marker in the document is part of the heading", "## Copy\nC-17.\n\n## Pickup branch (optional)\n\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expect(t, checkFiles(t, map[string]string{
				"reservation.md": reservation,
				"hold.md":        "# Hold 17\n\nType: [Reservation](reservation.md)\n\n" + tc.doc,
			}), tc.want...)
		})
	}
	d := Parse("reservation.md", []byte(reservation))
	expect(t, []string{renderDeclared(d.Declared())}, "Copy, Pickup branch?, Reminder?, Expiry?")
}

func renderDeclared(ds []Declared) string {
	var parts []string
	for _, d := range ds {
		if d.Optional {
			parts = append(parts, d.Name+"?")
		} else {
			parts = append(parts, d.Name)
		}
	}
	return strings.Join(parts, ", ")
}

func TestLinks(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{
		"b.md":     "# B\n\n## Sec\nx\n",
		"b c.md":   "# BC\n",
		"a%zz.md":  "# Percent\n",
		"inner.md": "# Inner\n",
		"outer.md": "# Outer\n",
		"sub/c.md": "# C\n\n[a](../a.md) [up](../../b.md) [abs](/sub/c.md) [self](c.md#top)\n",
		"a.md": "# A\n\n" +
			"[abs](/b.md) [bad](a%zz.md) [short](b%2.md) [dots](%2e%2e/out.md) [here](./b.md) [back](sub/../b.md)\n" +
			"[t](b.md \"t\") [t2](b.md 't') [pad]( b.md ) [sp](<b c.md>) [pct](b%20c.md) [case](B.md) [ext](b.MD)\n" +
			"[n [m](ghost-inner.md)](outer.md) [](b.md) [e](b.md#) [h](b.md?q) [l](b.md#sec \"t\")\n" +
			"[i](mailto:x.md) [j](C:/x.md) [k](//x.md) [web](https://example.com/x.md) [bare](#sec)\n" +
			"![img](ghost-img.md) !![img2](ghost-img2.md) [g](ghost.md) [again](ghost.md#x) [dot](./ghost.md)\n" +
			"`` x [g1](ghost-open.md) ``a`[g2](ghost-span.md) `a` [g3](ghost-between.md) `b` ```x`` [g4](ghost-after.md)\n" +
			"[q1](ghost-quote.md 'unclosed) [q2](ghost-2.md \"a\"\"b\") [space](ghost 3.md) [tab](\tghost-tab.md\t)\n",
	}),
		`a.md:3: link.unresolved: no document at "/b.md"`,
		`a.md:3: link.unresolved: no document at "b%2.md"`,
		`a.md:3: link.unresolved: no document at "%2e%2e/out.md"`,
		`a.md:4: link.unresolved: no document at "B.md"`,
		`a.md:4: link.unresolved: no document at "b.MD"`,
		`a.md:6: link.unresolved: no document at "//x.md"`,
		`a.md:7: link.unresolved: no document at "ghost.md"`,
		`a.md:7: link.unresolved: no document at "./ghost.md"`,
		`a.md:8: link.unresolved: no document at "ghost-after.md"`,
		`a.md:9: link.unresolved: no document at "ghost-tab.md"`,
		`sub/c.md:3: link.unresolved: no document at "../../b.md"`,
		`sub/c.md:3: link.unresolved: no document at "/sub/c.md"`,
	)
}

func TestFences(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{"f.md": "# F\n\n" +
		"```\n[a](ghost1.md)\n``\n[b](ghost2.md)\n```\n[c](ghost3.md)\n" +
		"    ```\n[d](ghost4.md)\n" +
		"~~~\n[e](ghost5.md)\n``` \n[f](ghost6.md)\n~~~ x\n[g](ghost7.md)\n~~~\n" +
		"````js\n[h](ghost8.md)\n```\n````\n[i](ghost9.md)\n" +
		"```\n[j](unclosed.md)\n"}),
		`f.md:8: link.unresolved: no document at "ghost3.md"`,
		`f.md:10: link.unresolved: no document at "ghost4.md"`,
		`f.md:22: link.unresolved: no document at "ghost9.md"`,
	)
}

func TestParse(t *testing.T) {
	d := Parse("staff/ada.md", []byte("# Ada\n\nType: [Member](../types/member.md)\n\n## Card\n[c](../cards/a%20b.md#n)\n"))
	if d.Title != "Ada" || d.TitleLine != 1 || d.TypeLine != 3 || d.TypeRaw != "../types/member.md" || d.TypePath != "types/member.md" {
		t.Errorf("parsed %+v", d)
	}
	want := []Link{{Target: "cards/a b.md", Raw: "../cards/a%20b.md", Section: "Card", Line: 6}}
	if !reflect.DeepEqual(d.Links, want) {
		t.Errorf("links %+v, want %+v", d.Links, want)
	}
	if len(d.Sections) != 1 || d.Sections[0].Line != 5 || !d.Sections[0].HasText() {
		t.Errorf("sections %+v", d.Sections)
	}
}
