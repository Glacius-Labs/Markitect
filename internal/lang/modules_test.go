package lang

import (
	"path"
	"sort"
	"strings"
	"testing"
)

// withReadmes returns the files with the README.md that each of their modules
// lacks: it lists under Contents every document of its folder and the README of
// every module directly below it, so that a fixture of another rule meets no
// rule of the modules. A README that the fixture has stays as it is.
func withReadmes(files map[string]string) map[string]string {
	out := map[string]string{}
	holds := map[string][]string{}
	for p, text := range files {
		out[p] = text
		if IsDocumentName(path.Base(p)) {
			holds[path.Dir(p)] = append(holds[path.Dir(p)], p)
		}
	}
	for dir := range holds {
		readme := path.Join(dir, "README.md")
		if _, ok := files[readme]; ok {
			continue
		}
		shown := append([]string(nil), holds[dir]...)
		for other := range holds {
			if other == dir || !(dir == "." || strings.HasPrefix(other, dir+"/")) {
				continue
			}
			nearest := true
			for between := path.Dir(other); between != dir && between != "."; between = path.Dir(between) {
				if _, ok := holds[between]; ok {
					nearest = false
				}
			}
			if nearest {
				shown = append(shown, path.Join(other, "README.md"))
			}
		}
		sort.Strings(shown)
		title := path.Base(dir)
		if dir == "." {
			title = "Model"
		}
		text := "# " + title + "\n\n## Contents\n"
		for _, p := range shown {
			text += "- [" + path.Base(p) + "](<" + strings.TrimPrefix(strings.TrimPrefix(p, dir), "/") + ">)\n"
		}
		out[readme] = text
	}
	return out
}

func TestWithReadmesLeavesNoModuleWithoutOne(t *testing.T) {
	files := withReadmes(map[string]string{
		"a.md":          "# A\n",
		"x/y/b.md":      "# B\n",
		"x/y/z/c.md":    "# C\n",
		"w/README.md":   "# W\n",
		"w/d.md":        "# D\n",
		"only/note.txt": "x",
	})
	if got := files["README.md"]; !strings.Contains(got, "<a.md>") || !strings.Contains(got, "<x/y/README.md>") || !strings.Contains(got, "<w/README.md>") || strings.Contains(got, "z/") {
		t.Errorf("root README: %q", got)
	}
	if files["w/README.md"] != "# W\n" || files["x/y/z/README.md"] == "" || files["x/README.md"] != "" || files["only/README.md"] != "" {
		t.Errorf("READMEs: %v", files)
	}
	expect(t, checkExactly(t, files))
}

// shop is a module of three documents; its README shows the cart rule and not
// the limit.
func shopFiles(extra map[string]string) map[string]string {
	files := map[string]string{
		"shop/README.md":     "# Shop\n\nHow customers buy.\n\n## Contents\n- [Cart](cart.md)\n",
		"shop/cart.md":       "# Cart\n\nA cart.\n",
		"shop/cart-limit.md": "# Cart limit\n\nAt most ten.\n",
	}
	for p, text := range extra {
		files[p] = text
	}
	return files
}

func TestModuleNeedsAReadme(t *testing.T) {
	expect(t, checkExactly(t, map[string]string{"shop/cart.md": "# Cart\n"}),
		`shop/README.md:1: module.readme: the folder shop holds documents and has no README.md`)
	// A folder that holds only folders, or no Markdown at all, is transparent.
	expect(t, checkExactly(t, map[string]string{
		"a/README.md":      "# A\n\n## Contents\n- [B](x/b/README.md)\n",
		"a/x/b/README.md":  "# B\n",
		"a/x/b/doc.md":     "# Doc\n",
		"a/x/y/image.png":  "",
		"a/x/y/notes.txt":  "text",
		"a/x/y/z/deep.txt": "text",
	}))
	// Without a README in the folder between them, a module below is found by
	// its own folder.
	expect(t, checkExactly(t, map[string]string{
		"a/README.md":   "# A\n",
		"a/x/b/note.md": "# Note\n",
	}), `a/x/b/README.md:1: module.readme: the folder a/x/b holds documents and has no README.md`)
	// Every module is reported once, and the model folder is one when it holds
	// a document directly.
	expect(t, checkExactly(t, map[string]string{
		"top.md":       "# Top\n",
		"a/one.md":     "# One\n",
		"a/two.md":     "# Two\n",
		"b/c/inner.md": "# Inner\n",
	}),
		`README.md:1: module.readme: the model folder holds documents and has no README.md`,
		`a/README.md:1: module.readme: the folder a holds documents and has no README.md`,
		`b/c/README.md:1: module.readme: the folder b/c holds documents and has no README.md`)
	// The model folder holds only folders: no README is needed there.
	expect(t, checkExactly(t, map[string]string{"a/README.md": "# A\n"}))
	// A folder with only a README is a module with a README.
	expect(t, checkExactly(t, map[string]string{"a/README.md": "# A\n", "b/README.md": "# B\n"}))
	// A README is exactly README.md: another name is no manifest.
	expect(t, checkExactly(t, map[string]string{"a/readme.md": "# A\n"}),
		`a/README.md:1: module.readme: the folder a holds documents and has no README.md`)
}

func TestReadmeHasNoTypeOrExtendsLine(t *testing.T) {
	files := map[string]string{
		"types/README.md":  "# Types\n\n## Contents\n- [Member](member.md)\n",
		"types/member.md":  member,
		"people/README.md": "# People\n\nType: [Member](../types/member.md)\n\n## Card\nx\n",
		"people/ada.md":    "# Ada\n\nType: [Member](../types/member.md)\n\n## Card\n1\n",
		"base/README.md":   "# Base\n\nExtends: [Member](../types/member.md)\n\n## Contents\n- [Part](part.md)\n",
		"base/part.md":     "# Part\n",
	}
	// The README is read without that line, so it needs no section of the type
	// it names, and the link on that line is no link of the README.
	expect(t, checkExactly(t, files),
		"base/README.md:3: readme.type: a README has no extends line",
		"people/README.md:3: readme.type: a README has no type line")
}

func TestReadmeIsReadWithoutItsHead(t *testing.T) {
	s := modelOf(t, map[string]string{
		"types/README.md":  "# Types\n\nType: [Member](member.md)\n\n## Contents\n- [Member](member.md)\n",
		"types/member.md":  member,
		"base/README.md":   "# Base\n\nExtends: [Member](../types/member.md)\n",
		"people/README.md": "# People\n",
	})
	for _, p := range []string{"types/README.md", "base/README.md"} {
		d, _ := s.Document(p)
		if d.HasType() || d.HasExtends() || d.TypeLine != 0 || d.ExtendsLine != 0 || s.IsType(d) || len(d.ReadmeHeads) != 1 || d.ReadmeHeads[0].Line != 3 {
			t.Errorf("%s: %+v", p, d)
		}
	}
	member, _ := s.Document("types/member.md")
	if got := s.Objects(member); len(got) != 0 {
		t.Errorf("a README is an object of member: %v", got)
	}
	// Its links to documents stay links of the README, but not the head's.
	d, _ := s.Document("types/README.md")
	if len(d.Links) != 1 || d.Links[0].Section != "Contents" {
		t.Errorf("links %+v", d.Links)
	}
}

func TestTypeLineNamingAReadme(t *testing.T) {
	files := map[string]string{
		"types/README.md": "# Types\n\n## Contents\n- [Member](member.md)\n",
		"types/member.md": member,
		"a/README.md":     "# A\n",
		"a/ada.md":        "# Ada\n\nType: [Types](../types/README.md)\n\n## Card\n1\n",
		"a/base.md":       "# Base\n\nExtends: [Types](../types/README.md)\n",
		"a/own.md":        "# Own\n\nType: [Types](../a/README.md)\n",
		"a/missing.md":    "# Missing\n\nType: [Types](../zzz/README.md)\n",
	}
	expect(t, checkExactly(t, files),
		`a/ada.md:3: readme.type: "types/README.md" is a README, which is no type`,
		`a/base.md:3: readme.type: "types/README.md" is a README, which is no type`,
		`a/missing.md:3: type.unresolved: no document at "../zzz/README.md"`,
		`a/own.md:3: readme.type: "a/README.md" is a README, which is no type`)
	// The document is no object of anything and its sections are not checked.
	s := modelOf(t, files)
	d, _ := s.Document("a/ada.md")
	if d.HasType() || d.TypeLine != 3 || d.HeadReadme != "types/README.md" {
		t.Errorf("%+v", d)
	}
	if r, _ := s.Document("types/README.md"); s.IsType(r) {
		t.Error("a README that a type line names is a type")
	}
}

func TestLinksIntoAModule(t *testing.T) {
	const internal = `"shop/cart-limit.md" is internal to shop: shop/README.md does not list it under Contents`
	for name, tc := range map[string]struct {
		files map[string]string
		want  []string
	}{
		"inside the module everything may link to everything": {
			shopFiles(map[string]string{"shop/cart.md": "# Cart\n\nSee [the limit](cart-limit.md).\n"}), nil},
		"a document outside sees what the README lists, and the README": {
			shopFiles(map[string]string{"other/README.md": "# Other\n", "other/a.md": "# A\n\n[cart](../shop/cart.md) [shop](../shop/README.md)\n"}), nil},
		"a document outside does not see what the README leaves out": {
			shopFiles(map[string]string{"other/README.md": "# Other\n", "other/a.md": "# A\n\nThe [limit](../shop/cart-limit.md).\n"}),
			[]string{"other/a.md:3: link.internal: " + internal}},
		"a link with a section counts as a link to its document": {
			shopFiles(map[string]string{"other/README.md": "# Other\n", "other/a.md": "# A\n\n[cart](../shop/cart.md#top) and [limit](../shop/cart-limit.md#top)\n"}),
			[]string{"other/a.md:3: link.internal: " + internal}},
		"a link that does not resolve is no internal link": {
			shopFiles(map[string]string{"other/README.md": "# Other\n", "other/a.md": "# A\n\n[x](../shop/ghost.md)\n"}),
			[]string{`other/a.md:3: link.unresolved: no document at "../shop/ghost.md"`}},
		"the same document linked twice is reported once": {
			shopFiles(map[string]string{"other/README.md": "# Other\n", "other/a.md": "# A\n\n[l](../shop/cart-limit.md)\n\n[l](../shop/cart-limit.md)\n"}),
			[]string{"other/a.md:3: link.internal: " + internal}},
		"a child links to the internal documents of its parent": {
			map[string]string{
				"shop/README.md":      "# Shop\n\n## Contents\n- [Cart](cart/README.md)\n",
				"shop/rules.md":       "# Rules\n",
				"shop/cart/README.md": "# Cart\n",
				"shop/cart/item.md":   "# Item\n\nBound by [the rules](../rules.md).\n",
			}, nil},
		"a parent does not link to the internal documents of a child": {
			map[string]string{
				"shop/README.md":      "# Shop\n\n## Contents\n- [Cart](cart/README.md)\n",
				"shop/rules.md":       "# Rules\n\nSee [the item](cart/item.md).\n",
				"shop/cart/README.md": "# Cart\n",
				"shop/cart/item.md":   "# Item\n",
			},
			[]string{`shop/rules.md:3: link.internal: "shop/cart/item.md" is internal to shop/cart: shop/cart/README.md does not list it under Contents`}},
		"a parent links to the README of a child that it lists": {
			map[string]string{
				"shop/README.md":      "# Shop\n\n## Contents\n- [Cart](cart/README.md)\n",
				"shop/rules.md":       "# Rules\n\nSee [the cart](cart/README.md).\n",
				"shop/cart/README.md": "# Cart\n",
			}, nil},
		"a sibling's document is internal, its README is not": {
			map[string]string{
				"a/README.md": "# A\n\n[B](../b/README.md)\n",
				"a/one.md":    "# One\n\n[readme](../b/README.md) and [two](../b/two.md)\n",
				"b/README.md": "# B\n",
				"b/two.md":    "# Two\n",
			},
			[]string{`a/one.md:3: link.internal: "b/two.md" is internal to b: b/README.md does not list it under Contents`}},
		"the upper module does not show the README of the lower one": {
			map[string]string{
				"shop/README.md":      "# Shop\n",
				"shop/cart/README.md": "# Cart\n\n## Contents\n- [Item](item.md)\n",
				"shop/cart/item.md":   "# Item\n",
				"other/README.md":     "# Other\n",
				"other/a.md":          "# A\n\n[item](../shop/cart/item.md) and [cart](../shop/cart/README.md)\n",
			},
			[]string{
				`other/a.md:3: link.internal: "shop/cart/item.md" is internal to shop: shop/README.md does not list shop/cart/README.md under Contents`,
				`other/a.md:3: link.internal: "shop/cart/README.md" is internal to shop: shop/README.md does not list it under Contents`,
			}},
		"the upper module shows the README of the lower one": {
			map[string]string{
				"shop/README.md":      "# Shop\n\n## Contents\n- [Cart](cart/README.md)\n",
				"shop/cart/README.md": "# Cart\n\n## Contents\n- [Item](item.md)\n",
				"shop/cart/item.md":   "# Item\n",
				"other/README.md":     "# Other\n",
				"other/a.md":          "# A\n\n[item](../shop/cart/item.md) and [cart](../shop/cart/README.md)\n",
			}, nil},
		"the README of a nested module is not shown": {
			map[string]string{
				"shop/README.md":      "# Shop\n",
				"shop/cart/README.md": "# Cart\n",
				"other/README.md":     "# Other\n",
				"other/a.md":          "# A\n\n[cart](../shop/cart/README.md)\n",
			},
			[]string{`other/a.md:3: link.internal: "shop/cart/README.md" is internal to shop: shop/README.md does not list it under Contents`}},
		"the lower module shows the item but the upper one hides the way": {
			map[string]string{
				"shop/README.md":      "# Shop\n\n## Contents\n- [Item](cart/item.md)\n",
				"shop/cart/README.md": "# Cart\n\n## Contents\n- [Item](item.md)\n",
				"shop/cart/item.md":   "# Item\n",
				"other/README.md":     "# Other\n",
				"other/a.md":          "# A\n\n[item](../shop/cart/item.md)\n",
			},
			[]string{`other/a.md:3: link.internal: "shop/cart/item.md" is internal to shop: shop/README.md does not list shop/cart/README.md under Contents`}},
		"a folder between two modules counts as no level": {
			map[string]string{
				"shop/README.md":            "# Shop\n\n## Contents\n- [Cart](parts/cart/README.md)\n",
				"shop/parts/cart/README.md": "# Cart\n\n## Contents\n- [Item](item.md)\n",
				"shop/parts/cart/item.md":   "# Item\n",
				"other/README.md":           "# Other\n",
				"other/a.md":                "# A\n\n[item](../shop/parts/cart/item.md)\n",
			}, nil},
		"Contents shows a document of the folder only, not one two folders down": {
			map[string]string{
				"shop/README.md":            "# Shop\n\n## Contents\n- [Item](parts/cart/item.md)\n- [Cart](parts/cart/README.md)\n",
				"shop/parts/cart/README.md": "# Cart\n",
				"shop/parts/cart/item.md":   "# Item\n",
				"other/README.md":           "# Other\n",
				"other/a.md":                "# A\n\n[item](../shop/parts/cart/item.md)\n",
			},
			// shop/README.md links the item of cart: reaching into a module is
			// itself an error, and shows nothing.
			[]string{
				`other/a.md:3: link.internal: "shop/parts/cart/item.md" is internal to shop/parts/cart: shop/parts/cart/README.md does not list it under Contents`,
				`shop/README.md:4: link.internal: "shop/parts/cart/item.md" is internal to shop/parts/cart: shop/parts/cart/README.md does not list it under Contents`,
			}},
		"Contents is matched without regard to case": {
			shopFiles(map[string]string{"shop/README.md": "# Shop\n\n## CONTENTS\n- [Limit](cart-limit.md)\n", "other/README.md": "# Other\n", "other/a.md": "# A\n\n[l](../shop/cart-limit.md)\n"}), nil},
		"a link outside Contents shows nothing": {
			shopFiles(map[string]string{"shop/README.md": "# Shop\n\n## Contents\n- [Cart](cart.md)\n\n## See also\n- [Limit](cart-limit.md)\n", "other/README.md": "# Other\n", "other/a.md": "# A\n\n[l](../shop/cart-limit.md)\n"}),
			[]string{"other/a.md:3: link.internal: " + internal}},
		"a link in the README before any section shows nothing": {
			shopFiles(map[string]string{"shop/README.md": "# Shop\n\n[Limit](cart-limit.md)\n\n## Contents\n- [Cart](cart.md)\n", "other/README.md": "# Other\n", "other/a.md": "# A\n\n[l](../shop/cart-limit.md)\n"}),
			[]string{"other/a.md:3: link.internal: " + internal}},
		"a README without Contents shows only itself": {
			map[string]string{"shop/README.md": "# Shop\n", "shop/cart.md": "# Cart\n", "other/README.md": "# Other\n", "other/a.md": "# A\n\n[shop](../shop/README.md) [cart](../shop/cart.md)\n"},
			[]string{`other/a.md:3: link.internal: "shop/cart.md" is internal to shop: shop/README.md does not list it under Contents`}},
		"a module without a README shows nothing": {
			map[string]string{"shop/cart.md": "# Cart\n", "other/README.md": "# Other\n", "other/a.md": "# A\n\n[cart](../shop/cart.md)\n"},
			[]string{
				`other/a.md:3: link.internal: "shop/cart.md" is internal to shop: shop has no README.md`,
				`shop/README.md:1: module.readme: the folder shop holds documents and has no README.md`,
			}},
		"the model folder as a module": {
			map[string]string{
				"README.md":      "# Model\n\n## Contents\n- [Intro](intro.md)\n- [Shop](shop/README.md)\n",
				"intro.md":       "# Intro\n\n[shop](shop/README.md) and [cart](shop/cart.md)\n",
				"hidden.md":      "# Hidden\n",
				"shop/README.md": "# Shop\n",
				"shop/cart.md":   "# Cart\n\nBack to [hidden](../hidden.md) and [intro](../intro.md).\n",
			},
			[]string{`intro.md:3: link.internal: "shop/cart.md" is internal to shop: shop/README.md does not list it under Contents`}},
		"a module links to the internal documents of the model folder": {
			map[string]string{
				"README.md":      "# Model\n",
				"intro.md":       "# Intro\n",
				"shop/README.md": "# Shop\n",
				"shop/cart.md":   "# Cart\n\nBack to [intro](../intro.md).\n",
			}, nil},
		"a module that the model folder does not list links to its internal documents": {
			map[string]string{
				"README.md":   "# Model\n",
				"hidden.md":   "# Hidden\n",
				"a/README.md": "# A\n\n## Contents\n- [Doc](doc.md)\n",
				"a/doc.md":    "# Doc\n\n[hidden](../hidden.md)\n",
			}, nil},
		"the model folder links to the internal document of a module": {
			map[string]string{
				"README.md":      "# Model\n\n[cart](shop/cart.md)\n",
				"shop/README.md": "# Shop\n",
				"shop/cart.md":   "# Cart\n",
			},
			[]string{`README.md:3: link.internal: "shop/cart.md" is internal to shop: shop/README.md does not list it under Contents`}},
		"a document that links to itself": {
			shopFiles(map[string]string{"shop/cart-limit.md": "# Cart limit\n\n[me](cart-limit.md)\n"}), nil},
	} {
		t.Run(name, func(t *testing.T) { expect(t, checkExactly(t, tc.files), tc.want...) })
	}
}

func TestTypeAndExtendsLinksIntoAModule(t *testing.T) {
	const member = "# Member\n\nA person.\n\n## Card\nThe card number.\n"
	typesWith := func(contents string) map[string]string {
		return map[string]string{
			"types/README.md":  "# Types\n\n## Contents\n" + contents,
			"types/member.md":  member,
			"people/README.md": "# People\n",
		}
	}
	// The type line.
	files := typesWith("- [Member](member.md)\n")
	files["people/ada.md"] = "# Ada\n\nType: [Member](../types/member.md)\n\n## Card\n1\n"
	expect(t, checkExactly(t, files))
	files = typesWith("")
	files["people/ada.md"] = "# Ada\n\nType: [Member](../types/member.md)\n\n## Card\n1\n"
	expect(t, checkExactly(t, files),
		`people/ada.md:3: link.internal: "types/member.md" is internal to types: types/README.md does not list it under Contents`)
	// The extends line.
	files = typesWith("- [Member](member.md)\n")
	files["people/student.md"] = "# Student\n\nExtends: [Member](../types/member.md)\n\n## School\nWhere.\n"
	expect(t, checkExactly(t, files))
	files = typesWith("")
	files["people/student.md"] = "# Student\n\nExtends: [Member](../types/member.md)\n\n## School\nWhere.\n"
	expect(t, checkExactly(t, files),
		`people/student.md:3: link.internal: "types/member.md" is internal to types: types/README.md does not list it under Contents`)
	// A type line and a link in the text to the same document: one report, at
	// the type line.
	files = typesWith("")
	files["people/ada.md"] = "# Ada\n\nType: [Member](../types/member.md)\n\n## Card\nSee [member](../types/member.md).\n"
	expect(t, checkExactly(t, files),
		`people/ada.md:3: link.internal: "types/member.md" is internal to types: types/README.md does not list it under Contents`)
	// A type that is unresolved is type.unresolved only.
	files = typesWith("")
	files["people/ada.md"] = "# Ada\n\nType: [Member](../types/nobody.md)\n"
	expect(t, checkExactly(t, files), `people/ada.md:3: type.unresolved: no document at "../types/nobody.md"`)
	// The type of the same module needs no listing.
	files = typesWith("")
	files["types/ada.md"] = "# Ada\n\nType: [Member](member.md)\n\n## Card\n1\n"
	expect(t, checkExactly(t, files))
}

func TestDiagnosticsOfModulesAreSortedWithTheRest(t *testing.T) {
	got := checkExactly(t, map[string]string{
		"b.md":            "# B\n\n[x](ghost.md)\n",
		"sub/a.md":        "# A\n",
		"sub/z/c.md":      "# C\n\n[a](../a.md)\n",
		"sub/z/README.md": "# Z\n",
	})
	expect(t, got,
		`README.md:1: module.readme: the model folder holds documents and has no README.md`,
		`b.md:3: link.unresolved: no document at "ghost.md"`,
		`sub/README.md:1: module.readme: the folder sub holds documents and has no README.md`)
}
