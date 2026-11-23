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
