package lang

import (
	"reflect"
	"strings"
	"testing"
)

const (
	pageType  = "# Page\n\nA page.\n\n## Body\nText.\n"
	pageDoc   = "# Start\n\nType: [Page](page.md)\n\n## Body\nHello.\n"
	siteTypeP = "# Site\n\nA site.\n\n## Pages\nType: [Page](page.md)\n\nThe pages.\n"
)

// site is an object of the type Site, with the given text under Pages.
func site(pages string) string {
	return "# Home\n\nType: [Site](site.md)\n\n## Pages\n" + pages + "\n"
}

func TestPropertyLineIsRead(t *testing.T) {
	d := Parse("types/site.md", []byte("# Site\n\nA site.\n\n## Pages\nType: [Page](page.md#body), [Post](../post.md \"t\")\n\nThe [pages](x.md).\n\n## State\nType: `draft`, ` live `\n\n## Hosting\nWhere.\n"))
	pages, state, hosting := d.Sections[0], d.Sections[1], d.Sections[2]
	wantRefs := []PropertyRef{{Raw: "page.md", Target: "types/page.md"}, {Raw: "../post.md", Target: "post.md"}}
	if pages.Property.Line != 6 || pages.Property.Kind != PropertyRefs || !reflect.DeepEqual(pages.Property.Refs, wantRefs) {
		t.Errorf("pages %+v", pages.Property)
	}
	if state.Property.Line != 11 || state.Property.Kind != PropertyValues || !reflect.DeepEqual(state.Property.Values, []string{"draft", "live"}) {
		t.Errorf("state %+v", state.Property)
	}
	if hosting.Property.Line != 0 || hosting.Property.Kind != "" {
		t.Errorf("hosting %+v", hosting.Property)
	}
	// The line is no stray type line, no link and no prose; the rest is.
	if len(d.StrayTypeLines) != 0 || len(d.Links) != 1 || d.Links[0].Raw != "x.md" || d.Links[0].SectionLine != 5 {
		t.Errorf("stray %v, links %+v", d.StrayTypeLines, d.Links)
	}
	for _, text := range d.Texts {
		if strings.Contains(text.Text, "Type") || strings.Contains(text.Text, "draft") {
			t.Errorf("the line is prose: %q", text.Text)
		}
	}
}

func TestPropertyLineForms(t *testing.T) {
	bad := "property type line must be links to documents or code spans, separated by commas"
	files := map[string]string{
		"page.md": pageType,
		"post.md": pageType,
		// Valid: one link, several, one code span, several, without a space after the
		// comma, and a code span that holds the text of a link, which is a value.
		"one.md":     "# One\n\n## A\nType: [Page](page.md)\n",
		"several.md": "# Several\n\n## A\nType: [Page](page.md), [Post](post.md)\n",
		"tight.md":   "# Tight\n\n## A\nType: [Page](page.md),[Post](post.md)\n",
		"code.md":    "# Code\n\n## A\nType: `a`\n",
		"codes.md":   "# Codes\n\n## A\nType: `a`,`b`, `c`\n",
		// Invalid: a mix, text after the links, a trailing comma, no separator, no
		// value, an image, a word, a web address, another file, markup around a
		// link, a blank code span.
		"mix.md":      "# Mix\n\n## A\nType: [Page](page.md), `b`\n",
		"mix2.md":     "# Mix2\n\n## A\nType: `a`, [Page](page.md)\n",
		"after.md":    "# After\n\n## A\nType: [Page](page.md) and more\n",
		"trailing.md": "# Trailing\n\n## A\nType: [Page](page.md),\n",
		"nosep.md":    "# Nosep\n\n## A\nType: [Page](page.md) [Post](post.md)\n",
		"empty.md":    "# Empty\n\n## A\nType:\n",
		"image.md":    "# Image\n\n## A\nType: ![Page](page.md)\n",
		"word.md":     "# Word\n\n## A\nType: text\n",
		"web.md":      "# Web\n\n## A\nType: [Page](https://example.com/page.md)\n",
		"file.md":     "# File\n\n## A\nType: [Page](page.txt)\n",
		"bold.md":     "# Bold\n\n## A\nType: **[Page](page.md)**\n",
		"inlink.md":   "# Inlink\n\n## A\nType: `[Page](page.md)`\n",
		"blank.md":    "# Blank\n\n## A\nType: ` `\n",
	}
	expect(t, checkFiles(t, files),
		"after.md:4: property.line: "+bad,
		"blank.md:4: property.line: "+bad,
		"bold.md:4: property.line: "+bad,
		"empty.md:4: property.line: "+bad,
		"file.md:4: property.line: "+bad,
		"image.md:4: property.line: "+bad,
		"mix.md:4: property.line: "+bad,
		"mix2.md:4: property.line: "+bad,
		"nosep.md:4: property.line: "+bad,
		"trailing.md:4: property.line: "+bad,
		"web.md:4: property.line: "+bad,
		"word.md:4: property.line: "+bad,
	)
}

func TestPropertyLinePlace(t *testing.T) {
	stray := "a type line stands only on the first line after the title"
	expect(t, checkFiles(t, map[string]string{
		"page.md": pageType,
		// The second line of the block is a stray type line, as anywhere else.
		"second.md": "# Second\n\n## A\nText.\nType: [Page](page.md)\n",
		// Under a level-3 heading the block after the level-2 heading is a heading.
		"deep.md": "# Deep\n\n## A\n### B\nType: [Page](page.md)\n",
		// A line after a blank line is the first line of its own block, not of the
		// block after the heading, but that block is the first one too.
		"blank.md": "# Blank\n\n## A\n\nType: [Page](page.md)\n",
		// After a block of another kind, the line is no property type line.
		"list.md": "# List\n\n## A\n- one\n\nType: [Page](page.md)\n",
		// In code it is text, in a list it is a stray type line.
		"fenced.md": "# Fenced\n\n## A\n```\nType: [Page](page.md)\n```\n",
		"listed.md": "# Listed\n\n## A\n- Type: [Page](page.md)\n",
		// Under the title it is the type line.
		"object.md": "# Object\n\nType: [Page](page.md)\n\n## Body\nText.\n",
		// Only exactly "Type:" counts.
		"lower.md": "# Lower\n\n## A\ntype: [Page](page.md)\n",
		"types.md": "# Types\n\n## A\nTypes: [Page](page.md)\n",
		// A line with a link reference definition before it.
		"defined.md": "# Defined\n\n## A\n[x]: other.md\nType: [Page](page.md)\n",
	}),
		"deep.md:5: type.line: "+stray,
		"list.md:6: type.line: "+stray,
		"listed.md:4: type.line: "+stray,
		"second.md:5: type.line: "+stray,
	)
}

func TestPropertyLineIsRecognizedAsType(t *testing.T) {
	files := map[string]string{
		"page.md":   pageType,
		"refs.md":   "# Refs\n\n## A\nType: [Page](page.md)\n",
		"values.md": "# Values\n\n## A\nType: `x`\n",
		"plain.md":  "# Plain\n\n## A\nNothing.\n",
		"bad.md":    "# Bad\n\n## A\nType: nonsense\n",
		"obj.md":    "# Obj\n\nType: [Page](page.md)\n\n## A\nType: `x`\n",
		"one.md":    "# One\n\nType: [Values](values.md)\n\n## A\nx\n",
	}
	s := modelOf(t, withReadmes(files))
	for path, want := range map[string]bool{
		// A type by its own line alone, and by being named in a line.
		"refs.md": true, "values.md": true, "page.md": true,
		"plain.md": false, "bad.md": false,
		// An object stays an object, and what its line names is no type.
		"obj.md": false, "README.md": false,
	} {
		if got := s.IsType(mustDoc(t, s, path)); got != want {
			t.Errorf("IsType(%s) = %v, want %v", path, got, want)
		}
	}
	// An unused type is no object of anything and no error.
	expect(t, checkFiles(t, map[string]string{"refs.md": "# Refs\n\n## A\nType: `x`\n"}))
	if got := s.Objects(mustDoc(t, s, "refs.md")); len(got) != 0 {
		t.Errorf("objects %v", got)
	}
}

func TestPropertyLineInAnObjectOrAReadme(t *testing.T) {
	object := "a document with a type line has no property type line"
	files := map[string]string{
		"page.md": pageType,
		// The object keeps its type: its missing section is still found, and its
		// line makes it no type and names nothing.
		"obj.md":    "# Obj\n\nType: [Page](page.md)\n\n## Extra\nType: [Ghost](ghost.md)\nx\n",
		"values.md": "# Values\n\nType: [Page](page.md)\n\n## Body\nType: `a`\nText.\n",
		// A document with a broken type line is no better.
		"broken.md": "# Broken\n\nType: [Page](page.md), more\n\n## Extra\nType: `a`\nx\n",
		"README.md": "# Model\n\n## Contents\nType: [Page](page.md)\n\n- [Page](page.md)\n- [Obj](obj.md)\n- [Values](values.md)\n- [Broken](broken.md)\n",
	}
	expect(t, checkExactly(t, files),
		"README.md:4: property.line: a README has no property type line",
		"broken.md:3: type.line: type line must be exactly one link to a document",
		"broken.md:6: property.line: "+object,
		"obj.md:3: section.missing: missing section \"Body\" required by page.md",
		"obj.md:6: property.line: "+object,
		"values.md:6: property.line: "+object,
	)
	s := modelOf(t, files)
	if s.IsType(mustDoc(t, s, "README.md")) || s.IsType(mustDoc(t, s, "obj.md")) || !mustDoc(t, s, "obj.md").HasType() {
		t.Error("an object stays an object, and a README is no type")
	}
	// A README is no type by a line of its own, and the line names nothing.
	if !s.IsType(mustDoc(t, s, "page.md")) {
		t.Error("page.md is named by a type line")
	}
	// The line is read as a line, so it is no link of the README either.
	if got := mustDoc(t, s, "README.md").Links; len(got) != 4 {
		t.Errorf("links %+v", got)
	}
}

func TestPropertyLineLinks(t *testing.T) {
	expect(t, checkFiles(t, map[string]string{
		"page.md":  pageType,
		"a.md":     "# A\n\n## P\nType: [Page](page.md), [Ghost](ghost.md), [Page again](page.md#body), [Ghost](ghost.md#x)\n",
		"b.md":     "# B\n\n## P\nType: [Ghost](ghost.md), [Page](page.md)\n\nA [ghost](nowhere.md).\n",
		"c.md":     "# C\n\n## P\nType: [Model](README.md)\n",
		"d.md":     "# D\n\n## P\nType: [Model](README.md), [Ghost](ghost.md)\n",
		"e/f/g.md": "# G\n\n## P\nType: [Up](../../page.md)\n",
	}),
		`a.md:4: property.unresolved: no document at "ghost.md"`,
		`b.md:4: property.unresolved: no document at "ghost.md"`,
		`b.md:6: link.unresolved: no document at "nowhere.md"`,
		`c.md:4: readme.type: "README.md" is a README, which is no type`,
		`d.md:4: readme.type: "README.md" is a README, which is no type`,
		`d.md:4: property.unresolved: no document at "ghost.md"`,
	)
}
