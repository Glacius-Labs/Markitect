package lang

import "testing"

// shopWith is the module shop with the README given and the documents cart and
// limit, and a document outside that links the targets given.
func shopWith(readme, links string) map[string]string {
	return map[string]string{
		"shop/README.md":  readme,
		"shop/cart.md":    "# Cart\n",
		"shop/limit.md":   "# Limit\n",
		"other/README.md": "# Other\n",
		"other/a.md":      "# A\n\n" + links + "\n",
	}
}

func internalTo(module, target string) string {
	return `other/a.md:3: link.internal: "shop/` + target + `" is internal to ` + module + `: ` + module + `/README.md does not list it under Contents`
}

func TestWhatContentsShows(t *testing.T) {
	both := "[c](../shop/cart.md) [l](../shop/limit.md)"
	for name, tc := range map[string]struct {
		readme string
		want   []string
	}{
		"a reference link shows its document": {
			"# Shop\n\n## Contents\n- [Cart][c]\n- [Limit]\n\n[c]: cart.md\n[limit]: limit.md\n", nil},
		"a link in a code span shows nothing": {
			"# Shop\n\n## Contents\n- `[Cart](cart.md)`\n- [Limit](limit.md)\n", []string{internalTo("shop", "cart.md")}},
		"a link in a fenced block shows nothing": {
			"# Shop\n\n## Contents\n```\n[Cart](cart.md)\n```\n- [Limit](limit.md)\n", []string{internalTo("shop", "cart.md")}},
		"two sections Contents both count": {
			"# Shop\n\n## Contents\n- [Cart](cart.md)\n\n## Other\nText.\n\n## Contents\n- [Limit](limit.md)\n", nil},
		"a pattern in the second section Contents counts": {
			"# Shop\n\n## Contents\n- [Cart](cart.md)\n\n## Other\nText.\n\n## contents\n- `li*.md`\n", nil},
		"a subsection of Contents belongs to it": {
			"# Shop\n\n## Contents\n### Orders\n- [Cart](cart.md)\n### Rules\n- [Limit](limit.md)\n", nil},
		"a subsection of another section is not Contents": {
			"# Shop\n\n## Contents\n- [Cart](cart.md)\n\n## Notes\n### Contents\n- [Limit](limit.md)\n", []string{internalTo("shop", "limit.md")}},
		"a link spelled with ./ and ../": {
			"# Shop\n\n## Contents\n- [Cart](./cart.md)\n- [Limit](../shop/./limit.md#top)\n", nil},
		"a link to a document of another module shows nothing": {
			"# Shop\n\n## Contents\n- [Other](../other/a.md)\n",
			[]string{
				internalTo("shop", "cart.md"),
				internalTo("shop", "limit.md"),
				`shop/README.md:4: link.internal: "other/a.md" is internal to other: other/README.md does not list it under Contents`,
			}},
	} {
		t.Run(name, func(t *testing.T) { expect(t, checkExactly(t, shopWith(tc.readme, both)), tc.want...) })
	}
}

func TestChainOfThreeLevels(t *testing.T) {
	files := func(top, middle string) map[string]string {
		return map[string]string{
			"a/README.md":     "# A\n\n## Contents\n" + top,
			"a/b/README.md":   "# B\n\n## Contents\n" + middle,
			"a/b/c/README.md": "# C\n\n## Contents\n- [x](x.md)\n",
			"a/b/c/x.md":      "# X\n",
			"a/b/c/y.md":      "# Y\n",
			"out/README.md":   "# Out\n",
			"out/l.md":        "# L\n\n[x](../a/b/c/x.md) [y](../a/b/c/y.md)\n",
		}
	}
	// Every level shows the README of the one below it.
	expect(t, checkExactly(t, files("- [B](b/README.md)\n", "- [C](c/README.md)\n")),
		`out/l.md:3: link.internal: "a/b/c/y.md" is internal to a/b/c: a/b/c/README.md does not list it under Contents`)
	// The upper level does not show the middle README.
	expect(t, checkExactly(t, files("", "- [C](c/README.md)\n")),
		`out/l.md:3: link.internal: "a/b/c/x.md" is internal to a: a/README.md does not list a/b/README.md under Contents`,
		`out/l.md:3: link.internal: "a/b/c/y.md" is internal to a: a/README.md does not list a/b/README.md under Contents`)
	// The middle level does not show the lower README.
	expect(t, checkExactly(t, files("- [B](b/README.md)\n", "")),
		`out/l.md:3: link.internal: "a/b/c/x.md" is internal to a/b: a/b/README.md does not list a/b/c/README.md under Contents`,
		`out/l.md:3: link.internal: "a/b/c/y.md" is internal to a/b: a/b/README.md does not list a/b/c/README.md under Contents`)
	// A module inside sees what is internal to every module above it.
	deep := files("- [B](b/README.md)\n", "- [C](c/README.md)\n")
	deep["a/a-only.md"] = "# A only\n"
	deep["a/b/c/z.md"] = "# Z\n\n[middle](../README.md) [top](../../README.md) [only](../../a-only.md)\n"
	expect(t, checkExactly(t, deep), `out/l.md:3: link.internal: "a/b/c/y.md" is internal to a/b/c: a/b/c/README.md does not list it under Contents`)
}

// A parent lists the README of a module that has none: the cause is the
// missing README, not the list.
func TestListedModuleWithoutReadme(t *testing.T) {
	files := map[string]string{
		"a/README.md":   "# A\n\n## Contents\n- [B](b/README.md)\n",
		"a/b/x.md":      "# X\n",
		"out/README.md": "# Out\n",
		"out/l.md":      "# L\n\n[x](../a/b/x.md)\n",
	}
	expect(t, checkExactly(t, files),
		`a/README.md:4: link.unresolved: no document at "b/README.md"`,
		`a/b/README.md:1: module.readme: the folder a/b holds documents and has no README.md`,
		`out/l.md:3: link.internal: "a/b/x.md" is internal to a: a/b has no README.md`)
}

// Only README.md is a README: a file in another case is an ordinary document,
// and its folder has no README.
func TestOtherSpellingsOfReadmeAreOrdinaryDocuments(t *testing.T) {
	files := map[string]string{
		"a/readme.md": "# A\n\nType: [T](../t/README.md)\n",
		"b/README.MD": "# B\n\nExtends: [T](../t/README.md)\n",
		"t/README.md": "# T\n\n## Contents\n- [Types](types.md)\n",
		"t/types.md":  "# Types\n",
	}
	expect(t, checkExactly(t, files),
		`a/README.md:1: module.readme: the folder a holds documents and has no README.md`,
		`a/readme.md:3: readme.type: "t/README.md" is a README, which is no type`,
		`b/README.MD:3: readme.type: "t/README.md" is a README, which is no type`,
		`b/README.md:1: module.readme: the folder b holds documents and has no README.md`)
	s := specOf(t, files)
	d, _ := s.Document("a/readme.md")
	if len(d.ReadmeHeads) != 0 {
		t.Errorf("readme.md is read as a README: %+v", d)
	}
	d, _ = s.Document("b/README.MD")
	if len(d.ReadmeHeads) != 0 {
		t.Errorf("README.MD is read as a README: %+v", d)
	}
}

func TestReadmeWithBothHeadLines(t *testing.T) {
	files := map[string]string{
		"types/README.md": "# Types\n\n## Contents\n- [Member](member.md)\n",
		"types/member.md": member,
		"a/README.md":     "# A\n\nType: [Member](../types/member.md)\nExtends: [Member](../types/member.md)\n\nText.\n",
		"b/README.md":     "# B\n\nExtends: [Member](../types/member.md)\nType: [Member](../types/member.md)\n",
		"c/README.md":     "# C\n\nType: [Member](../types/member.md)\n\nType: [Member](../types/member.md)\n",
	}
	// Each head line is a readme.type; a line outside the head block is still a
	// stray line.
	expect(t, checkExactly(t, files),
		"a/README.md:3: readme.type: a README has no type line",
		"a/README.md:4: readme.type: a README has no extends line",
		"b/README.md:3: readme.type: a README has no extends line",
		"b/README.md:4: readme.type: a README has no type line",
		"c/README.md:3: readme.type: a README has no type line",
		"c/README.md:5: type.line: a type line stands only on the first line after the title")
}

func TestReadmeTypeAndInternalOnOneLine(t *testing.T) {
	files := map[string]string{
		"shop/README.md":      "# Shop\n",
		"shop/cart/README.md": "# Cart\n",
		"other/README.md":     "# Other\n",
		"other/a.md":          "# A\n\nType: [Cart](../shop/cart/README.md)\n",
	}
	expect(t, checkExactly(t, files),
		`other/a.md:3: readme.type: "shop/cart/README.md" is a README, which is no type`,
		`other/a.md:3: link.internal: "shop/cart/README.md" is internal to shop: shop/README.md does not list it under Contents`)
}
