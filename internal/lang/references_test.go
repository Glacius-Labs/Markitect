package lang

import (
	"reflect"
	"strconv"
	"testing"
)

func TestPropertyLineVisibility(t *testing.T) {
	files := map[string]string{
		"shop/README.md":     "# Shop\n\n## Contents\n- [Cart](cart.md)\n",
		"shop/cart.md":       "# Cart\n\nA cart.\n\n## Items\nWhat is in it.\n",
		"shop/cart-limit.md": "# Cart limit\n\nAt most ten.\n",
		"README.md":          "# Model\n\n## Contents\n- [Order](order.md)\n- [Shop](shop/README.md)\n",
		"order.md":           "# Order\n\n## Cart\nType: [Cart](shop/cart.md)\n\n## Limit\nType: [Cart limit](shop/cart-limit.md), [Cart](shop/cart.md)\n\n## Both\nType: [Cart limit](shop/cart-limit.md)\n",
	}
	internal := `: link.internal: "shop/cart-limit.md" is internal to shop: shop/README.md does not list it under Contents`
	// What the module does not show is internal, the line included, and a target
	// is reported once, at its first link.
	expect(t, checkExactly(t, files), "order.md:7"+internal)
	// The first link is the first by line, whether a property type line or text.
	files["order.md"] = "# Order\n\n## Cart\nType: [Cart limit](shop/cart-limit.md)\n\n## Other\nSee [the limit](shop/cart-limit.md).\n"
	expect(t, checkExactly(t, files), "order.md:4"+internal)
	files["order.md"] = "# Order\n\nSee [the limit](shop/cart-limit.md).\n\n## Cart\nType: [Cart limit](shop/cart-limit.md)\n"
	expect(t, checkExactly(t, files), "order.md:3"+internal)
	files["shop/README.md"] = "# Shop\n\n## Contents\n- [Cart](cart.md)\n- [Cart limit](cart-limit.md)\n"
	expect(t, checkExactly(t, files))
	// From inside the module nothing is internal.
	files["shop/README.md"] = "# Shop\n\n## Contents\n- [Cart](cart.md)\n"
	files["shop/cart-limit.md"] = "# Cart limit\n\n## Cart\nType: [Cart](cart.md)\n"
	files["order.md"] = "# Order\n"
	expect(t, checkExactly(t, files))
}

func TestReferenceProperty(t *testing.T) {
	files := map[string]string{
		"page.md":    pageType,
		"post.md":    "# Post\n\nExtends: [Page](page.md)\n\nA page of a blog.\n\n## Tags\nThe tags.\n",
		"article.md": "# Article\n\nExtends: [Post](post.md)\n\nA long post.\n",
		"note.md":    "# Note\n\nA note.\n\n## Text\nWords.\n",
		"site.md":    siteTypeP,
		"multi.md":   "# Multi\n\nA site.\n\n## Items\nType: [Page](page.md), [Note](note.md)\n\nThings.\n",
		"start.md":   pageDoc,
		"post1.md":   "# Post 1\n\nType: [Post](post.md)\n\n## Body\nB.\n\n## Tags\nT.\n",
		"deep.md":    "# Deep\n\nType: [Article](article.md)\n\n## Body\nB.\n\n## Tags\nT.\n",
		"note1.md":   "# Note 1\n\nType: [Note](note.md)\n\n## Text\nT.\n",
		"free.md":    "# Free\n\nNo type.\n",
	}
	// home is an object of Site with the given text under Pages, from line 6.
	check := func(pages string, want ...string) {
		t.Helper()
		f := map[string]string{"home.md": site(pages)}
		for k, v := range files {
			f[k] = v
		}
		expect(t, checkFiles(t, f), want...)
	}
	wrong := func(line int, target, what string) string {
		return "home.md:" + strconv.Itoa(line) + `: reference.type: section "Pages" refers to documents of page.md or a type that extends it, but "` + target + `" is ` + what
	}
	// An object of the type, of a derived type, and of one further down.
	check("- [Start](start.md)\n- [A post](post1.md)\n- [Deep](deep.md)\n")
	// Any number of links is allowed, and none, and text around them.
	check("None yet.")
	check("[Start](start.md) and [again](start.md#body), [a post](post1.md).")
	// An object of another type, a type, a README, an untyped document, itself.
	check("[Note](note1.md)", wrong(6, "note1.md", "a document of note.md"))
	check("[Page](page.md)", wrong(6, "page.md", "a type"))
	check("[Post](post.md)", wrong(6, "post.md", "a type"))
	check("[The README](README.md)", wrong(6, "README.md", "a README"))
	check("[Free](free.md)", wrong(6, "free.md", "a document without a type"))
	check("[Home](home.md)", wrong(6, "home.md", "a document of site.md"))
	// One right and one wrong: only the wrong one is reported, at its link, once.
	check("- [Start](start.md)\n- [Note](note1.md)\n- [Note again](note1.md#text)\n", wrong(7, "note1.md", "a document of note.md"))
	check("- [Note](note1.md)\n- [Start](start.md)\n", wrong(6, "note1.md", "a document of note.md"))
	// A link that does not resolve is only link.unresolved; a link to something
	// that is no .md is no link.
	check("[Ghost](ghost.md), [web](https://example.com/x.md), [file](start.txt)", `home.md:6: link.unresolved: no document at "ghost.md"`)
	// A text property, the text before the first section and a section the type
	// does not declare are no reference property: nothing is checked there.
	f := map[string]string{
		"home.md": "# Home\n\nType: [Site](site.md)\n\nBefore [free](free.md).\n\n## Pages\n[Start](start.md)\n\n## Other\n[Note](note1.md)\n\n## Notes\n[Note](note1.md)\n",
		"site.md": "# Site\n\nSee [free](free.md).\n\n## Pages\nType: [Page](page.md)\n\n## Notes (optional)\nText.\n",
	}
	for k, v := range files {
		if f[k] == "" {
			f[k] = v
		}
	}
	expect(t, checkFiles(t, f))
	// Several types: one of them, or a type derived from one.
	f["home.md"] = "# Home\n\nType: [Multi](multi.md)\n\n## Items\n- [Start](start.md)\n- [Note](note1.md)\n- [Post](post1.md)\n- [Free](free.md)\n- [Page](page.md)\n"
	multi := `: reference.type: section "Items" refers to documents of one of page.md, note.md or a type that extends one of them, but `
	expect(t, checkFiles(t, f),
		"home.md:9"+multi+`"free.md" is a document without a type`,
		"home.md:10"+multi+`"page.md" is a type`,
	)
	// The heading of the object may be in another case, and a second heading of
	// the same name is no property.
	f["home.md"] = "# Home\n\nType: [Multi](multi.md)\n\n## ITEMS\n[Start](start.md)\n\n## items\n[Free](free.md)\n"
	expect(t, checkFiles(t, f))
}

func TestReferenceToAnUnresolvedType(t *testing.T) {
	// The line is reported where it stands; the objects are not checked against a
	// type that is no document.
	expect(t, checkFiles(t, map[string]string{
		"site.md": "# Site\n\nA site.\n\n## Pages\nType: [Ghost](ghost.md)\n",
		"home.md": "# Home\n\nType: [Site](site.md)\n\n## Pages\n[Home](home.md)\n[Free](free.md)\n",
		"free.md": "# Free\n",
	}), `site.md:6: property.unresolved: no document at "ghost.md"`)
	expect(t, checkFiles(t, map[string]string{
		"site.md": "# Site\n\nA site.\n\n## Pages\nType: [Model](README.md)\n",
		"home.md": "# Home\n\nType: [Site](site.md)\n\n## Pages\n[Free](free.md)\n",
		"free.md": "# Free\n",
	}), `site.md:6: readme.type: "README.md" is a README, which is no type`)
}

func TestFixedValues(t *testing.T) {
	state := "# State site\n\nA site.\n\n## State\nType: `draft`, `live`, `retired`\n\nWhere it stands.\n\n## Name\nType: `Ada`\n"
	object := func(value, name string) map[string]string {
		return map[string]string{"site.md": state, "home.md": "# Home\n\nType: [State site](site.md)\n\n## State\n" + value + "\n\n## Name\n" + name + "\n"}
	}
	wrong := func(line int, first string) string {
		return "home.md:" + strconv.Itoa(line) + `: property.value: the first line of section "State" is "` + first + "\", not one of `draft`, `live`, `retired`"
	}
	expect(t, checkFiles(t, object("live", "Ada")))
	// What follows the first line is free, and so is a second paragraph.
	expect(t, checkFiles(t, object("draft\nbecause it is new.\n\nMore.", "Ada\n\nAda Lovelace")))
	// Compared exactly, case regarded, the whole first line.
	expect(t, checkFiles(t, object("Live", "Ada")), wrong(6, "Live"))
	expect(t, checkFiles(t, object("live and well", "Ada")), wrong(6, "live and well"))
	expect(t, checkFiles(t, object("wip", "ada")), wrong(6, "wip"),
		"home.md:9: property.value: the first line of section \"Name\" is \"ada\", not one of `Ada`")
	// The first line as plain text, the way table reads it: markup is dropped.
	expect(t, checkFiles(t, object("**live**", "[Ada](site.md)")))
	expect(t, checkFiles(t, object("- live\n- draft", "`Ada`")))
	expect(t, checkFiles(t, object("> live", "Ada")))
	expect(t, checkFiles(t, object("**live** now", "Ada")), wrong(6, "live now"))
	// A section without text stays section.empty and is not checked again.
	expect(t, checkFiles(t, object("", "Ada")), `home.md:5: section.empty: empty section "State"`)
	expect(t, checkFiles(t, object("<!-- later -->", "Ada")), `home.md:5: section.empty: empty section "State"`)
	// A missing section is section.missing only.
	f := object("live", "Ada")
	f["home.md"] = "# Home\n\nType: [State site](site.md)\n\n## Name\nAda\n"
	expect(t, checkFiles(t, f), `home.md:3: section.missing: missing section "State" required by site.md`)
	// The heading in another case is the section.
	f["home.md"] = "# Home\n\nType: [State site](site.md)\n\n## state\nlive\n\n## NAME\nAda\n"
	expect(t, checkFiles(t, f))
	// An optional property is checked when it is there.
	f["site.md"] = "# State site\n\nA site.\n\n## State (optional)\nType: `draft`\n"
	f["home.md"] = "# Home\n\nType: [State site](site.md)\n\n## State\nlive\n"
	expect(t, checkFiles(t, f), "home.md:6: property.value: the first line of section \"State\" is \"live\", not one of `draft`")
	f["home.md"] = "# Home\n\nType: [State site](site.md)\n"
	expect(t, checkFiles(t, f))
}

func TestPropertyTypeComesDownTheChain(t *testing.T) {
	files := map[string]string{
		"a.md": "# A\n\nBase.\n\n## State\nType: `x`, `y`\n\n## Link\nType: [C](c.md)\n\n## Text\nFree.\n",
		// B redeclares every section, without a line: it keeps the type.
		"b.md": "# B\n\nExtends: [A](a.md)\n\nMiddle.\n\n## State\nOne of the states.\n\n## Link\nA link.\n\n## Text\nAnything.\n",
		// C adds a section of its own and states the lines of A again, in another way.
		"c.md": "# C\n\nExtends: [B](b.md)\n\nTop.\n\n## State\nType: `y`, `x`\n\n## Link\nType: [C](./c.md)\n\n## Own\nType: `1`\n",
		"d.md": "# D\n\nType: [C](c.md)\n\n## State\ny\n\n## Link\n[D](d.md)\n\n## Text\nT.\n\n## Own\n1\n",
		"e.md": "# E\n\nType: [C](c.md)\n\n## State\nz\n\n## Link\n[D](d.md), [E](e.md), [A](a.md)\n\n## Text\n[A](a.md)\n\n## Own\n2\n",
		"f.md": "# F\n\nType: [B](b.md)\n\n## State\nz\n\n## Link\n[D](d.md), [F](f.md)\n\n## Text\nT.\n",
	}
	// An object of C is checked against the properties of the whole chain, each
	// with the nearest type that gives one; an object of B has those of A.
	expect(t, checkFiles(t, files),
		"e.md:6: property.value: the first line of section \"State\" is \"z\", not one of `y`, `x`",
		"e.md:9: reference.type: section \"Link\" refers to documents of c.md or a type that extends it, but \"a.md\" is a type",
		"e.md:15: property.value: the first line of section \"Own\" is \"2\", not one of `1`",
		"f.md:6: property.value: the first line of section \"State\" is \"z\", not one of `x`, `y`",
		"f.md:9: reference.type: section \"Link\" refers to documents of c.md or a type that extends it, but \"f.md\" is a document of b.md",
	)
	s := modelOf(t, withReadmes(files))
	got := func(typ, name string) Property { return s.PropertyType(mustDoc(t, s, typ), name) }
	if p := got("b.md", "State"); p.Kind != PropertyValues || !reflect.DeepEqual(p.Values, []string{"x", "y"}) || p.From != "a.md" {
		t.Errorf("b.md State: %+v", p)
	}
	if p := got("c.md", "state"); p.Kind != PropertyValues || !reflect.DeepEqual(p.Values, []string{"y", "x"}) || p.From != "c.md" {
		t.Errorf("c.md state: %+v", p)
	}
	if p := got("c.md", "Link"); p.Kind != PropertyRefs || !reflect.DeepEqual(p.Refs, []string{"c.md"}) {
		t.Errorf("c.md Link: %+v", p)
	}
	if p := got("c.md", "Text"); p.Kind != "" || p.From != "" {
		t.Errorf("c.md Text: %+v", p)
	}
}

func TestPropertyRetyped(t *testing.T) {
	artifact := "# Artifact\n\nSomething.\n\n## Pages\nThe pages.\n\n## State\nType: `a`, `b`\n\n## Link\nType: [Page](page.md)\n"
	derived := func(pages, state, link string) string {
		return "# Derived\n\nExtends: [Artifact](artifact.md)\n\nMore.\n\n## Pages\n" + pages + "\n\n## State\n" + state + "\n\n## Link\n" + link + "\n"
	}
	check := func(doc string, want ...string) {
		t.Helper()
		expect(t, checkFiles(t, map[string]string{"page.md": pageType, "post.md": pageType, "artifact.md": artifact, "derived.md": doc}), want...)
	}
	// A line of the derived type where the base has none, text included.
	check(derived("Type: [Page](page.md)\n\nPages.", "States.", "Links."),
		`derived.md:8: property.retyped: property "Pages" is text in artifact.md, not a reference to page.md`)
	check(derived("Type: `x`", "States.", "Links."),
		"derived.md:8: property.retyped: property \"Pages\" is text in artifact.md, not one of `x`")
	// Another line than the base's, of any kind.
	check(derived("Pages.", "Type: `a`", "Links."),
		"derived.md:11: property.retyped: property \"State\" is one of `a`, `b` in artifact.md, not one of `a`")
	check(derived("Pages.", "Type: [Page](page.md)", "Type: [Post](post.md)"),
		"derived.md:11: property.retyped: property \"State\" is one of `a`, `b` in artifact.md, not a reference to page.md",
		"derived.md:14: property.retyped: property \"Link\" is a reference to page.md in artifact.md, not a reference to post.md")
	check(derived("Pages.", "States.", "Type: [Page](page.md), [Post](post.md)"),
		"derived.md:14: property.retyped: property \"Link\" is a reference to page.md in artifact.md, not a reference to page.md, post.md")
	check(derived("Pages.", "States.", "Type: `a`"),
		"derived.md:14: property.retyped: property \"Link\" is a reference to page.md in artifact.md, not one of `a`")
	// No line keeps the inherited type, and the same line again is fine, in
	// another order and by another path.
	check(derived("Pages.", "States.", "Links."))
	check(derived("Pages.", "Type: `b`, `a`, `b`", "Type: [Page again](./page.md#x)"))
	// The section named in another case is the same property.
	check("# Derived\n\nExtends: [Artifact](artifact.md)\n\n## LINK\nType: [Post](post.md)\n",
		"derived.md:6: property.retyped: property \"LINK\" is a reference to page.md in artifact.md, not a reference to post.md")
	// A section the base does not declare is the derived type's own; a line that
	// is no line is no type, and so no other type.
	check("# Derived\n\nExtends: [Artifact](artifact.md)\n\n## Own\nType: `x`\n\n## State\nType: nonsense\n",
		"derived.md:9: property.line: property type line must be links to documents or code spans, separated by commas")
	// A base that is no document gives nothing to compare with.
	expect(t, checkFiles(t, map[string]string{"z.md": "# Z\n\nExtends: [Ghost](ghost.md)\n\n## P\nType: `x`\n"}),
		`z.md:3: extends.unresolved: no document at "ghost.md"`)
}

func TestPropertyRetypedAlongThreeLevels(t *testing.T) {
	// Each type is compared with what is given above it: a gives text, b a line,
	// c keeps it, d repeats it, e and f change it.
	files := map[string]string{
		"page.md": pageType,
		"a.md":    "# A\n\nBase.\n\n## P\nText.\n",
		"b.md":    "# B\n\nExtends: [A](a.md)\n\nMiddle.\n\n## P\nType: `x`\n",
		"c.md":    "# C\n\nExtends: [B](b.md)\n\nTop.\n\n## P\nStill the same.\n",
		"d.md":    "# D\n\nExtends: [C](c.md)\n\nTop.\n\n## P\nType: `x`\n",
		"e.md":    "# E\n\nExtends: [C](c.md)\n\nTop.\n\n## P\nType: `y`\n",
		"f.md":    "# F\n\nExtends: [D](d.md)\n\nTop.\n\n## P\nType: [Page](page.md)\n",
		// A base with a line, the middle one without, the top one with the same.
		"g.md": "# G\n\nBase.\n\n## P\nType: `x`\n",
		"h.md": "# H\n\nExtends: [G](g.md)\n\nMiddle.\n\n## P\nNo line.\n",
		"i.md": "# I\n\nExtends: [H](h.md)\n\nTop.\n\n## P\nType: `x`\n",
		"j.md": "# J\n\nExtends: [H](h.md)\n\nTop.\n\n## P\nType: `z`\n",
	}
	expect(t, checkFiles(t, files),
		"b.md:8: property.retyped: property \"P\" is text in a.md, not one of `x`",
		"e.md:8: property.retyped: property \"P\" is one of `x` in b.md, not one of `y`",
		"f.md:8: property.retyped: property \"P\" is one of `x` in d.md, not a reference to page.md",
		"j.md:8: property.retyped: property \"P\" is one of `x` in g.md, not one of `z`",
	)
}

func TestFootnoteLinksAreNoReferences(t *testing.T) {
	// A link in a footnote stands outside every section: it is no reference of the
	// last section, and no dependency of the module.
	files := map[string]string{
		"page.md":       pageType,
		"site.md":       siteTypeP,
		"start.md":      pageDoc,
		"free.md":       "# Free\n\nNo type.\n",
		"other/free.md": "# Free\n\nNo type either.\n",
		"home.md":       "# Home\n\nType: [Site](site.md)\n\n## Pages\n[Start](start.md)[^1]\n\n[^1]: See [free](free.md) and [the other](other/free.md).\n",
	}
	expect(t, checkFiles(t, files))
	d := mustDoc(t, modelOf(t, withReadmes(files)), "home.md")
	if len(d.Links) != 3 {
		t.Errorf("links %+v", d.Links)
	}
	for _, l := range d.Links {
		if l.Raw != "start.md" && l.SectionLine != 0 {
			t.Errorf("link %+v", l)
		}
	}
	// The same link in the section is a reference.
	files["home.md"] = "# Home\n\nType: [Site](site.md)\n\n## Pages\n[Start](start.md), [free](free.md)\n"
	expect(t, checkFiles(t, files), `home.md:6: reference.type: section "Pages" refers to documents of page.md or a type that extends it, but "free.md" is a document without a type`)
}

func TestPropertyDeclaredTwice(t *testing.T) {
	stray := "a type line stands only on the first line after the title"
	files := map[string]string{
		"page.md":  pageType,
		"other.md": "# Other\n\nO.\n",
		// The first heading of a name declares the property, with or without a line;
		// a later one, in any case, declares nothing, and its line is a stray line.
		"plain.md": "# Plain\n\nA.\n\n## Pages\nNo line.\n\n## pages\nType: [Page](page.md)\n",
		"first.md": "# First\n\nA.\n\n## Pages\nType: [Page](page.md)\n\n## PAGES\nType: [Other](other.md)\n",
		"bad.md":   "# Bad\n\nA.\n\n## Pages\nType: nonsense\n\n## Pages\nType: [Page](page.md)\n",
	}
	expect(t, checkFiles(t, files),
		"bad.md:6: property.line: property type line must be links to documents or code spans, separated by commas",
		"bad.md:9: type.line: "+stray,
		"first.md:9: type.line: "+stray,
		"plain.md:9: type.line: "+stray,
	)
	s := modelOf(t, withReadmes(files))
	// plain.md and bad.md have no property type; first.md has the first line's.
	if p := s.PropertyType(mustDoc(t, s, "plain.md"), "Pages"); p.Kind != "" {
		t.Errorf("plain: %+v", p)
	}
	if p := s.PropertyType(mustDoc(t, s, "bad.md"), "Pages"); p.Kind != "" {
		t.Errorf("bad: %+v", p)
	}
	if p := s.PropertyType(mustDoc(t, s, "first.md"), "pages"); !reflect.DeepEqual(p.Refs, []string{"page.md"}) {
		t.Errorf("first: %+v", p)
	}
	// Only the first line marks a type: page.md by first.md, nothing by the others.
	for path, want := range map[string]bool{"plain.md": false, "bad.md": false, "first.md": true, "page.md": true, "other.md": false} {
		if got := s.IsType(mustDoc(t, s, path)); got != want {
			t.Errorf("IsType(%s) = %v, want %v", path, got, want)
		}
	}
	// A later heading makes no dependency and is not checked for visibility.
	modules := map[string]string{
		"a/t.md": "# T\n\nA.\n\n## Peer\nText.\n\n## peer\nType: [U](../b/u.md)\n",
		"b/u.md": "# U\n\nB.\n\n## Peer\nType: [T](../a/t.md)\n",
		"b/v.md": "# V\n\nB.\n",
	}
	expect(t, checkFiles(t, modules), "a/t.md:9: type.line: "+stray)
	modules["b/README.md"] = "# B\n\n## Contents\n- [U](u.md)\n"
	modules["a/t.md"] = "# T\n\nA.\n\n## Peer\nText.\n\n## peer\nType: [V](../b/v.md)\n"
	expect(t, checkFiles(t, modules), "a/t.md:9: type.line: "+stray)
	// With the line first, it counts: a cycle and a link that is internal.
	modules["a/t.md"] = "# T\n\nA.\n\n## Peer\nType: [V](../b/v.md)\n\n## peer\nText.\n"
	expect(t, checkFiles(t, modules),
		`a/t.md:6: link.internal: "b/v.md" is internal to b: b/README.md does not list it under Contents`,
		"a/t.md:6: module.cycle: module cycle a -> b -> a: a/t.md:6 declares a reference to b/v.md, b/u.md:6 declares a reference to a/t.md")
}

func TestFixedValueWrittenWithBackticks(t *testing.T) {
	for value, want := range map[string]string{
		"live": "`live`", "a`b": "``a`b``", "`x": "`` `x ``", "x`": "`` x` ``", "a``b`": "``` a``b` ```",
		"a b": "`a b`",
	} {
		if got := QuoteValue(value); got != want {
			t.Errorf("QuoteValue(%q) = %s, want %s", value, got, want)
		}
		// Written so, it reads back as the value.
		d := Parse("t.md", []byte("# T\n\n## A\nType: "+QuoteValue(value)+"\n"))
		if got := d.Sections[0].Property.Values; len(got) != 1 || got[0] != value {
			t.Errorf("%q reads back as %q", value, got)
		}
	}
	expect(t, checkFiles(t, map[string]string{
		"t.md": "# T\n\nA.\n\n## State\nType: ``a`b``, `live`\n",
		"o.md": "# O\n\nType: [T](t.md)\n\n## State\nwip\n",
	}), "o.md:6: property.value: the first line of section \"State\" is \"wip\", not one of ``a`b``, `live`")
}

func TestFixedValueIsReportedAtItsLine(t *testing.T) {
	state := "# T\n\nA.\n\n## State\nType: `a`\n"
	line := func(body string) []string {
		return checkFiles(t, map[string]string{"t.md": state, "o.md": "# O\n\nType: [T](t.md)\n\n## State\n" + body})
	}
	// The first line of the text, not the heading: after a comment, in a list, in
	// code, in a quote and in a table.
	wrong := func(n int, first string) []string {
		return []string{"o.md:" + strconv.Itoa(n) + `: property.value: the first line of section "State" is "` + first + "\", not one of `a`"}
	}
	expect(t, line("wip\n"), wrong(6, "wip")...)
	expect(t, line("<!-- later -->\n\nwip\nmore\n"), wrong(8, "wip")...)
	expect(t, line("\n\n- wip\n- b\n"), wrong(8, "wip")...)
	expect(t, line("```\nwip\n```\n"), wrong(7, "wip")...)
	expect(t, line("> wip\n"), wrong(6, "wip")...)
	expect(t, line("| x |\n|---|\n| y |\n"), wrong(6, "x")...)
	expect(t, line("a\n"))
}
