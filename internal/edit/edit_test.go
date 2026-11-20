package edit

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

func mapFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

func load(t *testing.T, fsys fstest.MapFS) *lang.Spec {
	t.Helper()
	s, err := lang.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// carryOut applies a change set to a copy of fsys as the folder adapter does.
func carryOut(t *testing.T, fsys fstest.MapFS, c Changes) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	for p, f := range fsys {
		out[p] = f
	}
	for _, m := range c.Moves {
		if _, ok := out[m.To]; ok {
			t.Fatalf("move onto %s", m.To)
		}
		out[m.To] = out[m.From]
		delete(out, m.From)
	}
	for _, w := range c.Writes {
		if _, ok := out[w.Path]; ok && w.Create {
			t.Fatalf("create over %s", w.Path)
		}
		out[w.Path] = &fstest.MapFile{Data: w.Data}
	}
	return out
}

func text(fsys fstest.MapFS, p string) string {
	if f, ok := fsys[p]; ok {
		return string(f.Data)
	}
	return "<missing>"
}

func checkClean(t *testing.T, fsys fstest.MapFS) {
	t.Helper()
	if found := lang.Check(load(t, fsys)); len(found) > 0 {
		t.Errorf("check found %v", found)
	}
}

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}

func TestDocumentPath(t *testing.T) {
	for arg, want := range map[string]string{
		"lending/loan": "lending/loan.md", "lending/loan.md": "lending/loan.md", `lending\loan`: "lending/loan.md",
		"./a/../b.MD": "b.MD", " open loans ": "open loans.md",
		"../out": "", "/abs": "", "a/../../out": "", "https://x": "", ".md": "", "sub/.md": "", "": "",
	} {
		got, ok := DocumentPath(arg)
		if want == "" && ok || want != "" && got != want {
			t.Errorf("%q: got %q %v, want %q", arg, got, ok, want)
		}
	}
}

func TestRelAndQuote(t *testing.T) {
	eq(t, rel("types/member.md", "staff/new"), "../../types/member.md")
	eq(t, rel("types/member.md", ""), "types/member.md")
	eq(t, rel("types/member.md", "types"), "member.md")
	eq(t, rel("../outside.md", "sub"), "../../outside.md")
	eq(t, quote("my types/a b(1)ü.md"), "my%20types/a%20b%281%29%C3%BC.md")
}

func TestNewDocument(t *testing.T) {
	fsys := mapFS(map[string]string{
		"types/member.md":   "# Member\n\nA card holder.\n\n## Card\nThe number.\n\n## Card (optional)\nAgain.\n\n## Branch\nThe home branch.\n\n## Notes (optional)\nN.\n",
		"my types/shelf.md": "# Shelf\n\n## Place\nWhere.\n",
		"members/ada.md":    "# Ada\n",
	})
	s := load(t, fsys)
	p, c, err := NewDocument(s, fsys, "types/member.md", "members/grace", "Grace Hopper")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, p, "members/grace.md")
	eq(t, c, Changes{Writes: []Write{{Path: "members/grace.md", Create: true,
		Data: []byte("# Grace Hopper\n\nType: [Member](../types/member.md)\n\n## Card\n\n## Branch\n\n")}}})
	// The title defaults to the file name; a type in a folder with a space is encoded.
	_, c, _ = NewDocument(s, fsys, "my types/shelf.md", "fiction one", "")
	eq(t, string(c.Writes[0].Data), "# fiction one\n\nType: [Shelf](my%20types/shelf.md)\n\n## Place\n\n")
	_, c, _ = NewDocument(s, fsys, "my types/shelf.md", "my types/second", "")
	eq(t, strings.Split(string(c.Writes[0].Data), "\n")[2], "Type: [Shelf](shelf.md)")
	// A new document has empty sections, which check names.
	after := carryOut(t, fsys, mustChanges(NewDocument(s, fsys, "types/member.md", "deep/down/alan", "")))
	var codes []string
	for _, d := range lang.Check(load(t, after)) {
		codes = append(codes, d.String())
	}
	eq(t, codes, []string{
		`deep/down/alan.md:5: section.empty: empty section "Card"`,
		`deep/down/alan.md:7: section.empty: empty section "Branch"`,
	})
	for _, arg := range []string{"members/ada", "members/ada.md", "../x", "/abs/x"} {
		if _, _, err := NewDocument(s, fsys, "types/member.md", arg, ""); !IsRefusal(err) {
			t.Errorf("%s: %v", arg, err)
		}
	}
}

func mustChanges(_ string, c Changes, err error) Changes {
	if err != nil {
		panic(err)
	}
	return c
}

// lending is a small spec to move documents in.
func lending() fstest.MapFS {
	return mapFS(map[string]string{
		"types/post.md":     "# Post\n\nA duty someone holds.\n",
		"types/member.md":   "# Member\n\nA card holder.\n\n## Post\nThe post held.\n",
		"staff/director.md": "# Director\n\nRuns the library.\n",
		"roles/librarian.md": "# Librarian\n\nType: [Post](../types/post.md)\n\n" +
			"Lends books at the [desk](librarian.md#desk), answers to the [director](../staff/director.md), " +
			"see the [site](https://example.org/librarian.md).\nA badge ![badge](../img/badge.png).\n",
		"ada.md": "# Ada\n\nType: [Member](types/member.md)\n\n## Post\n" +
			"[Librarian](roles/librarian.md), [at the desk](roles/librarian.md#desk), [bracketed](<roles/librarian.md>), [Post](types/post.md).\n\n" +
			"A sample in code, `[sample](roles/librarian.md)`, is no link.\n\n~~~md\n[in a block](roles/librarian.md)\n~~~\n",
		"shifts.md": "# Shifts\n\nType: [Librarian](roles/librarian.md)\n",
	})
}

func move(t *testing.T, fsys fstest.MapFS, old, newArg string) (Moved, fstest.MapFS) {
	t.Helper()
	m, err := MoveDocument(load(t, fsys), fsys, old, newArg)
	if err != nil {
		t.Fatal(err)
	}
	return m, carryOut(t, fsys, m.Changes)
}

func TestMoveWithinAFolder(t *testing.T) {
	m, after := move(t, lending(), "roles/librarian.md", "roles/clerk")
	eq(t, m.To, "roles/clerk.md")
	eq(t, m.Changed, []string{"ada.md", "roles/clerk.md", "shifts.md"})
	eq(t, text(after, "roles/librarian.md"), "<missing>")
	eq(t, text(after, "ada.md"), "# Ada\n\nType: [Member](types/member.md)\n\n## Post\n"+
		"[Librarian](roles/clerk.md), [at the desk](roles/clerk.md#desk), [bracketed](<roles/clerk.md>), [Post](types/post.md).\n\n"+
		"A sample in code, `[sample](roles/librarian.md)`, is no link.\n\n~~~md\n[in a block](roles/librarian.md)\n~~~\n")
	eq(t, text(after, "shifts.md"), "# Shifts\n\nType: [Librarian](roles/clerk.md)\n")
	// Only the link to itself changes in the moved document, which stays in its folder.
	eq(t, text(after, "roles/clerk.md"), "# Librarian\n\nType: [Post](../types/post.md)\n\n"+
		"Lends books at the [desk](clerk.md#desk), answers to the [director](../staff/director.md), "+
		"see the [site](https://example.org/librarian.md).\nA badge ![badge](../img/badge.png).\n")
	checkClean(t, after)
}

func TestMoveToAnotherFolder(t *testing.T) {
	m, after := move(t, lending(), "roles/librarian", "staff/desk/clerk.md")
	eq(t, m.Changed, []string{"ada.md", "staff/desk/clerk.md", "shifts.md"})
	if !strings.Contains(text(after, "ada.md"), "[Librarian](staff/desk/clerk.md), [at the desk](staff/desk/clerk.md#desk), [bracketed](<staff/desk/clerk.md>)") {
		t.Error(text(after, "ada.md"))
	}
	// The moved document's own destinations: the type line, a neighbour, itself and the image. Web links stay.
	eq(t, text(after, "staff/desk/clerk.md"), "# Librarian\n\nType: [Post](../../types/post.md)\n\n"+
		"Lends books at the [desk](clerk.md#desk), answers to the [director](../director.md), "+
		"see the [site](https://example.org/librarian.md).\nA badge ![badge](../../img/badge.png).\n")
	checkClean(t, after)
	// Up a folder: every path is computed from each linking document.
	_, after = move(t, lending(), "types/post.md", "post")
	if !strings.Contains(text(after, "ada.md"), "[Post](post.md)") || !strings.Contains(text(after, "roles/librarian.md"), "Type: [Post](../post.md)") {
		t.Error(text(after, "ada.md"), text(after, "roles/librarian.md"))
	}
	checkClean(t, after)
}

func TestMoveCarriesEveryRelativeDestination(t *testing.T) {
	fsys := mapFS(map[string]string{
		"a.md": "# A\n\n![d](diagram.png) and the [pdf](files/r.pdf), [up](../shared/notes.txt), [web](https://example.org/x.png).\n\n" +
			"<img src=\"diagram.png\" alt=\"d\"> <a href='files/r.pdf'>pdf</a> <a href=files/r.pdf#p2>page</a>\n\n" +
			"<div>\n<img SRC = \"img/a b.png\">\n<!-- <img src=\"old.png\"> -->\n</div>\n\n[logo]: img/logo.svg\n",
	})
	_, after := move(t, fsys, "a.md", "sub dir/n.md")
	eq(t, text(after, "sub dir/n.md"), "# A\n\n![d](../diagram.png) and the [pdf](../files/r.pdf), [up](../../shared/notes.txt), [web](https://example.org/x.png).\n\n"+
		"<img src=\"../diagram.png\" alt=\"d\"> <a href='../files/r.pdf'>pdf</a> <a href=../files/r.pdf#p2>page</a>\n\n"+
		"<div>\n<img SRC = \"../img/a%20b.png\">\n<!-- <img src=\"old.png\"> -->\n</div>\n\n[logo]: ../img/logo.svg\n")
	// Within the folder nothing changes but the links to the document itself.
	m, err := MoveDocument(load(t, fsys), fsys, "a.md", "b.md")
	if err != nil || len(m.Changes.Writes) != 0 {
		t.Error(m, err)
	}
	// A value split over two lines of a block quote cannot be placed and refuses a move to another folder.
	fsys["a.md"] = &fstest.MapFile{Data: []byte("# A\n\n> <div><img src=\"diagram\n> .png\"></div>\n")}
	if _, err := MoveDocument(load(t, fsys), fsys, "a.md", "sub/n.md"); !IsRefusal(err) || !strings.Contains(err.Error(), "line 3") {
		t.Error(err)
	}
	// A value on the next line of the same tag is placed.
	fsys["a.md"] = &fstest.MapFile{Data: []byte("# A\n\n<img src=\n\"diagram.png\">\n")}
	_, after = move(t, fsys, "a.md", "sub/n.md")
	eq(t, text(after, "sub/n.md"), "# A\n\n<img src=\n\"../diagram.png\">\n")
}

func TestMoveRewritesOnlyRealAttributes(t *testing.T) {
	doc := "# A\n\n<div>\nSet href=config.md in the file, or src = lib/x.js.\n</div>\n\n" +
		"<pre><code>&lt;img src=\"example.png\"&gt;</code></pre>\n\n" +
		"<a title=\"href=fake.md\" data-src='x.png' href=\"real.md\">real</a>\n\n" +
		"<script>var s = '<img src=\"in-script.png\">';</script>\n"
	_, after := move(t, mapFS(map[string]string{"a.md": doc}), "a.md", "sub/a.md")
	eq(t, text(after, "sub/a.md"), strings.Replace(doc, `href="real.md"`, `href="../real.md"`, 1))
}

func TestMoveDecodesCharacterReferences(t *testing.T) {
	fsys := mapFS(map[string]string{
		"due.md":  "# Due\n",
		"list.md": "# List\n\n[decimal](due&#46;md) [hex](due&#x2E;md#top) [named](due&period;md)\n",
	})
	if found := lang.Check(load(t, fsys)); len(found) != 0 {
		t.Error(found)
	}
	_, after := move(t, fsys, "due.md", "late.md")
	eq(t, text(after, "list.md"), "# List\n\n[decimal](late.md) [hex](late.md#top) [named](late.md)\n")
}

func TestMoveKeepsBytes(t *testing.T) {
	fsys := lending()
	crlf := "\xef\xbb\xbf# Reading room\r\n\r\nType: [Librarian](../roles/librarian.md)\r\n\r\nQuestions go to the [desk](../roles/librarian.md#desk).\r\nNo final line break"
	fsys["rooms/reading.md"] = &fstest.MapFile{Data: []byte(crlf)}
	mover := "\xef\xbb\xbf# Porter\r\n\r\nHands keys to the [desk](librarian.md#desk)\r\nand to the [director](../staff/director.md).\r\n\r\nNo final line break"
	fsys["roles/porter.md"] = &fstest.MapFile{Data: []byte(mover)}
	_, after := move(t, fsys, "roles/librarian.md", "roles/clerk")
	eq(t, text(after, "rooms/reading.md"), strings.ReplaceAll(crlf, "librarian", "clerk"))
	eq(t, text(after, "roles/porter.md"), strings.ReplaceAll(mover, "librarian", "clerk"))
	for _, p := range []string{"types/post.md", "types/member.md", "staff/director.md"} {
		eq(t, text(after, p), text(fsys, p))
	}
	_, after = move(t, after, "roles/porter.md", "lobby/porter")
	eq(t, text(after, "lobby/porter.md"), "\xef\xbb\xbf# Porter\r\n\r\nHands keys to the [desk](../roles/clerk.md#desk)\r\nand to the [director](../staff/director.md).\r\n\r\nNo final line break")
	checkClean(t, after)
}

func TestMoveEncodings(t *testing.T) {
	fsys := lending()
	fsys["open loans/due soon.md"] = &fstest.MapFile{Data: []byte("# Due soon\n\nSee [this](due%20soon.md) and [posts](../types/post.md).\n")}
	fsys["list.md"] = &fstest.MapFile{Data: []byte("# List\n\n[encoded](open%20loans/due%20soon.md#top), [spaced](<open loans/due soon.md>) and [titled](<open loans/due soon.md#top> \"Due list\")\n\n" +
		"A [reference][due] link.\n\n[due]: open%20loans/due%20soon.md\n")}
	_, after := move(t, fsys, "open loans/due soon.md", "overdue/late now")
	eq(t, text(after, "list.md"), "# List\n\n[encoded](overdue/late%20now.md#top), [spaced](<overdue/late now.md>) and [titled](<overdue/late now.md#top> \"Due list\")\n\n"+
		"A [reference][due] link.\n\n[due]: overdue/late%20now.md\n")
	eq(t, text(after, "overdue/late now.md"), "# Due soon\n\nSee [this](late%20now.md) and [posts](../types/post.md).\n")
	checkClean(t, after)
	_, after = move(t, after, "overdue/late now.md", "plain")
	eq(t, text(after, "list.md"), "# List\n\n[encoded](plain.md#top), [spaced](<plain.md>) and [titled](<plain.md#top> \"Due list\")\n\n"+
		"A [reference][due] link.\n\n[due]: plain.md\n")
	// A new name with a space is encoded in a plain destination.
	_, after = move(t, lending(), "staff/director.md", "staff/the head")
	if !strings.Contains(text(after, "roles/librarian.md"), "[director](../staff/the%20head.md)") {
		t.Error(text(after, "roles/librarian.md"))
	}
	checkClean(t, after)
}

func TestMoveRefusals(t *testing.T) {
	fsys := lending()
	fsys["notes"] = &fstest.MapFile{Mode: 0o755 | 1<<31}
	s := load(t, fsys)
	for _, tc := range []struct{ old, new, msg string }{
		{"roles/librarian.md", "ada", "already exists"},
		{"roles/librarian.md", "roles/librarian.md", "already exists"},
		{"Roles/Librarian.md", "x", "unknown path"},
		{"roles/librarian.md", "../out", "invalid path"},
		{"roles/librarian.md", "/abs", "invalid path"},
		{"roles/librarian.md", "a/../../out", "invalid path"},
	} {
		_, err := MoveDocument(s, fsys, tc.old, tc.new)
		if !IsRefusal(err) || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("%s -> %s: %v", tc.old, tc.new, err)
		}
	}
}

// library is a small library of its own for apply.
func library() fstest.MapFS {
	return mapFS(map[string]string{
		"packages/lending/loan.md":      "# Loan\n\nType: [Member](../people/sub/member.md)\n\n## Due\nThe date.\n",
		"packages/lending/notes.txt":    "kept as it is\n",
		"packages/people/sub/member.md": "# Member\n\nHome [branch](../../places/branch.md).\n",
		"packages/places/branch.md":     "# Branch\n",
		"packages/unused/shelf.md":      "# Shelf\n",
		"templates/lib/README.md":       "# Library\n\nType: [Loan](../../packages/lending/loan.md#due)\n\n## Due\n\n[rooms](rooms/README.md) [web](https://example.org/a.md)\n",
		"templates/lib/rooms/README.md": "# Rooms\n\n[up](../README.md) [loan](<../../../packages/lending/loan.md>)\n",
		"templates/reading/README.md":   "# Reading\n",
		"report.md":                     "# Report\n",
	})
}

func TestApply(t *testing.T) {
	spec := fstest.MapFS{}
	a, err := Apply(spec, library(), "lib", "branch")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, a.Documents, []string{"branch/README.md", "branch/rooms/README.md"})
	eq(t, a.Packages, []string{"lending", "people", "places"})
	after := carryOut(t, spec, a.Changes)
	var paths []string
	for p := range after {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	eq(t, paths, []string{"branch/README.md", "branch/rooms/README.md", "packages/lending/loan.md", "packages/lending/notes.txt",
		"packages/people/sub/member.md", "packages/places/branch.md"})
	for _, p := range paths[2:] {
		eq(t, text(after, p), text(library(), p))
	}
	eq(t, text(after, "branch/README.md"), "# Library\n\nType: [Loan](../packages/lending/loan.md#due)\n\n## Due\n\n[rooms](rooms/README.md) [web](https://example.org/a.md)\n")
	eq(t, text(after, "branch/rooms/README.md"), "# Rooms\n\n[up](../README.md) [loan](<../../packages/lending/loan.md>)\n")
	checkClean(t, after)
}

func TestApplyKeepsThePackagesOfTheSpec(t *testing.T) {
	spec := mapFS(map[string]string{"packages/people/mine.md": "# Mine\n"})
	a, err := Apply(spec, library(), "lib", "")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, a.Packages, []string{"lending"}) // people is the spec's and not completed, so places is not reached
	for _, w := range a.Changes.Writes {
		if strings.HasPrefix(w.Path, "packages/people/") {
			t.Error(w.Path)
		}
	}
}

func TestApplyRefusals(t *testing.T) {
	lib := library()
	for _, tc := range []struct {
		name string
		lib  fstest.MapFS
		spec fstest.MapFS
		tpl  string
		want []string
	}{
		{"exists", lib, mapFS(map[string]string{"rooms/README.md": "# Mine\n"}), "lib", []string{"rooms/README.md: already exists"}},
		{"unknown template", lib, nil, "nothing", []string{"unknown template: nothing"}},
		{"outside", lib, nil, "..", []string{"unknown template: .."}},
		{"sub-folder linking its parent", lib, nil, "lib/rooms", []string{"templates/lib/rooms/README.md: link leaves the template: ../README.md"}},
		{"link elsewhere", with(lib, "templates/lib/rooms/README.md", "# Rooms\n\n[the reading room](../../reading/README.md)\n"), nil, "lib",
			[]string{"templates/lib/rooms/README.md: link leaves the template: ../../reading/README.md"}},
		{"package links out and to a missing package", with(with(lib, "packages/places/branch.md", "# Branch\n\n[t](../../templates/lib/README.md)\n"),
			"packages/people/sub/member.md", "# Member\n\n[m](../../missing/m.md) [b](../../places/branch.md)\n"), nil, "lib",
			[]string{"packages/missing: no such package in the library", "packages/places/branch.md: link leaves the template: ../../templates/lib/README.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.spec
			if spec == nil {
				spec = fstest.MapFS{}
			}
			_, err := Apply(spec, tc.lib, tc.tpl, "")
			var r *Refusal
			if !IsRefusal(err) {
				t.Fatalf("got %v", err)
			}
			r = err.(*Refusal)
			got := append([]string(nil), r.Problems...)
			sort.Strings(got)
			eq(t, got, tc.want)
		})
	}
	if _, err := Apply(fstest.MapFS{}, lib, "lib", "../up"); !IsRefusal(err) {
		t.Error(err)
	}
}

func TestApplyLeavesCodeAndImages(t *testing.T) {
	doc := "# Plain\n\nA span `[card](../../card.md)` and a photo ![photo](../../photo.md).\n\n```md\n[shelf](../../shelf.md)\n```\n"
	a, err := Apply(fstest.MapFS{}, with(library(), "templates/plain/README.md", doc), "plain", "")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, len(a.Packages), 0)
	eq(t, string(a.Changes.Writes[0].Data), doc)
}

func with(fsys fstest.MapFS, p, text string) fstest.MapFS {
	out := fstest.MapFS{}
	for k, v := range fsys {
		out[k] = v
	}
	out[p] = &fstest.MapFile{Data: []byte(text)}
	return out
}
