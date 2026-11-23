package lang

import (
	"strings"
	"testing"
	"testing/fstest"
)

const (
	artifact = "# Artifact\n\nSomething a team makes.\n\n## Owner\nWho answers for it.\n\n## Notes (optional)\nAnything else.\n"
	website  = "# Website\n\nExtends: [Artifact](artifact.md)\n\nA site people open in a browser.\n\n## Pages\nThe pages, each a page.\n"
)

func specOf(t *testing.T, files map[string]string) *Spec {
	t.Helper()
	fsys := fstest.MapFS{}
	for p, text := range files {
		fsys[p] = &fstest.MapFile{Data: []byte(text)}
	}
	s, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExtendsLineIsRead(t *testing.T) {
	d := Parse("types/website.md", []byte("# Website\n\nExtends: [Artifact](artifact.md#owner \"t\")\n\nText [x](ghost.md).\n"))
	if d.ExtendsLine != 3 || d.ExtendsRaw != "artifact.md" || d.ExtendsPath != "types/artifact.md" || !d.HasExtends() || d.TypeLine != 0 || d.HasType() {
		t.Errorf("parsed %+v", d)
	}
	// The link of the extends line is no link of the document.
	if len(d.Links) != 1 || d.Links[0].Raw != "ghost.md" {
		t.Errorf("links %+v", d.Links)
	}
	// The extends line is no prose either.
	if len(d.Texts) != 1 || d.Texts[0].Text != "Text x." {
		t.Errorf("texts %+v", d.Texts)
	}
	both := Parse("a.md", []byte("# A\n\nType: [T](t.md)\nExtends: [B](b.md)\n"))
	if both.TypeLine != 3 || both.ExtendsLine != 0 || len(both.StrayExtendsLines) != 1 || both.StrayExtendsLines[0] != 4 {
		t.Errorf("parsed %+v", both)
	}
}

func TestExtendsAType(t *testing.T) {
	// A type that extends a base, and that no document names as its type yet, is
	// no object of its base and gives no diagnostic.
	expect(t, checkFiles(t, map[string]string{"artifact.md": artifact, "website.md": website}))
	// Its sections are declarations: they may stay empty.
	expect(t, checkFiles(t, map[string]string{"artifact.md": artifact, "website.md": "# Website\n\nExtends: [Artifact](artifact.md)\n\n## Pages\n\n"}))
	// The extends line can carry a section and a title, as the type line can.
	expect(t, checkFiles(t, map[string]string{"artifact.md": artifact, "website.md": "# Website\n\nExtends: [Artifact](artifact.md#owner \"base\")  \n\n## Pages\nP.\n"}))
}

func TestChainOfThreeTypes(t *testing.T) {
	files := map[string]string{
		"artifact.md": artifact,
		"website.md":  website,
		"blog.md":     "# Blog\n\nExtends: [Website](website.md)\n\nA website of posts.\n\n## Feed\nWhere the posts are listed.\n\n## Pages\nThe pages, posts included.\n",
		"deep.md":     "# Deep\n\nType: [Blog](blog.md)\n\n## Owner\nO.\n\n## Pages\nP.\n\n## Feed\nF.\n",
		"top.md":      "# Top\n\nType: [Blog](blog.md)\n\n## Pages\nP.\n\n## Feed\nF.\n",
		"mid.md":      "# Mid\n\nType: [Blog](blog.md)\n\n## Owner\nO.\n\n## Feed\nF.\n",
		"empty.md":    "# Empty\n\nType: [Blog](blog.md)\n\n## Owner\n\n## Pages\nP.\n\n## Feed\n\n## Notes\n\n",
		"site.md":     "# Site\n\nType: [Website](website.md)\n\n## Pages\nP.\n\n## Owner\nO.\n",
	}
	expect(t, checkFiles(t, files),
		`empty.md:5: section.empty: empty section "Owner"`,
		`empty.md:10: section.empty: empty section "Feed"`,
		`empty.md:12: section.empty: empty section "Notes"`,
		`mid.md:3: section.missing: missing section "Pages" required by blog.md`,
		`top.md:3: section.missing: missing section "Owner" required by artifact.md`,
	)
	s := specOf(t, files)
	var names []string
	for _, c := range s.ChainSections(mustDoc(t, s, "blog.md")) {
		names = append(names, c.Name+"/"+c.DeclaredBy+"/"+c.RequiredBy)
	}
	// The base's sections first, then what each derived type adds; Pages stays
	// where website put it and is described by blog.
	expect(t, names, "Owner/artifact.md/artifact.md", "Notes/artifact.md/", "Pages/blog.md/blog.md", "Feed/blog.md/blog.md")
	var chain []string
	for _, c := range s.Chain(mustDoc(t, s, "blog.md")) {
		chain = append(chain, c.Path)
	}
	expect(t, chain, "blog.md", "website.md", "artifact.md")
	var objects []string
	for _, o := range s.Objects(mustDoc(t, s, "artifact.md")) {
		objects = append(objects, o.Path)
	}
	expect(t, objects, "deep.md", "empty.md", "mid.md", "site.md", "top.md")
	if !s.IsType(mustDoc(t, s, "artifact.md")) || !s.IsType(mustDoc(t, s, "website.md")) || s.IsType(mustDoc(t, s, "top.md")) {
		t.Error("types are the base, the derived type and nothing else")
	}
}

func mustDoc(t *testing.T, s *Spec, p string) *Document {
	t.Helper()
	d, ok := s.Document(p)
	if !ok {
		t.Fatalf("no document %s", p)
	}
	return d
}

func TestSectionTightened(t *testing.T) {
	// A derived type makes an optional section of its base required: allowed,
	// and the objects need it, by the type that requires it.
	derived := "# Website\n\nExtends: [Artifact](artifact.md)\n\n## Notes\nThe notes are no longer optional.\n\n## Pages\nP.\n"
	expect(t, checkFiles(t, map[string]string{
		"artifact.md": artifact,
		"website.md":  derived,
		"site.md":     "# Site\n\nType: [Website](website.md)\n\n## Owner\nO.\n\n## Pages\nP.\n",
		"ok.md":       "# Ok\n\nType: [Website](website.md)\n\n## Owner\nO.\n\n## Pages\nP.\n\n## notes\nN.\n",
	}), `site.md:3: section.missing: missing section "Notes" required by website.md`)
}

func TestSectionLoosened(t *testing.T) {
	files := map[string]string{
		"artifact.md": artifact,
		"website.md":  "# Website\n\nExtends: [Artifact](artifact.md)\n\n## Pages\nP.\n\n## owner (OPTIONAL)\nNo longer needed.\n",
		"blog.md":     "# Blog\n\nExtends: [Website](website.md)\n\n## Owner (optional)\nStill not.\n",
		"site.md":     "# Site\n\nType: [Blog](blog.md)\n\n## Pages\nP.\n",
	}
	expect(t, checkFiles(t, files),
		`blog.md:5: extends.loosens: section "Owner" is required by artifact.md`,
		// The section stays required for the objects: the loosening has no effect.
		`site.md:3: section.missing: missing section "Owner" required by artifact.md`,
		`website.md:8: extends.loosens: section "owner" is required by artifact.md`,
	)
	// An optional section above that nobody requires may stay optional below, and a
	// section a type up the chain marks optional but a middle one requires is
	// reported against the middle one.
	expect(t, checkFiles(t, map[string]string{
		"a.md": "# A\n\nX.\n\n## Q (optional)\nQ.\n",
		"b.md": "# B\n\nExtends: [A](a.md)\n\n## Q\nQ.\n",
		"c.md": "# C\n\nExtends: [B](b.md)\n\n## Q (optional)\nQ.\n",
	}), `c.md:5: extends.loosens: section "Q" is required by b.md`)
}

func TestExtendsCycles(t *testing.T) {
	cycle := "the chain of extends lines comes back to itself"
	files := map[string]string{
		"a.md":      "# A\n\nExtends: [B](b.md)\n\n## One\nO.\n",
		"b.md":      "# B\n\nExtends: [A](a.md)\n\n## Two\nT.\n",
		"self.md":   "# Self\n\nExtends: [Self](self.md)\n\n## Three\nT.\n",
		"into.md":   "# Into\n\nExtends: [A](a.md)\n\n## Four\nF.\n",
		"obj.md":    "# Obj\n\nType: [A](a.md)\n\n## Two\nT.\n",
		"obj2.md":   "# Obj2\n\nType: [Into](into.md)\n\n## Four\nF.\n",
		"object.md": "# Object\n\nType: [Self](self.md)\n\n## Four\nF.\n",
	}
	expect(t, checkFiles(t, files),
		"a.md:3: extends.cycle: "+cycle,
		"b.md:3: extends.cycle: "+cycle,
		// The sections that can be collected without looping.
		`obj.md:3: section.missing: missing section "One" required by a.md`,
		`obj2.md:3: section.missing: missing section "Two" required by b.md`,
		`obj2.md:3: section.missing: missing section "One" required by a.md`,
		`object.md:3: section.missing: missing section "Three" required by self.md`,
		"self.md:3: extends.cycle: "+cycle,
	)
	s := specOf(t, files)
	var objects []string
	for _, o := range s.Objects(mustDoc(t, s, "b.md")) {
		objects = append(objects, o.Path)
	}
	expect(t, objects, "obj.md", "obj2.md")
}

func TestExtendsLineErrors(t *testing.T) {
	bad := "extends line must be exactly one link to a document"
	stray := "an extends line stands only on the first line after the title"
	both := "a document has a type line or an extends line, not both"
	expect(t, checkFiles(t, map[string]string{
		"artifact.md": artifact,
		"role.md":     "# Role\n\nA role.\n\n## Owner\nO.\n",
		// Not exactly one link.
		"extra.md":    "# A\n\nExtends: [Artifact](artifact.md), see below\n",
		"no-space.md": "# A\n\nExtends:[Artifact](artifact.md)\n\n## Pages\nP.\n",
		"nested.md":   "# A\n\nExtends: [[Artifact]](artifact.md#owner)\n",
		"angle.md":    "# A\n\nExtends: <artifact.md>\n",
		"image.md":    "# A\n\nExtends: ![Artifact](artifact.md)\n",
		"code.md":     "# A\n\nExtends: `[Artifact](artifact.md)`\n",
		"web.md":      "# A\n\nExtends: [Artifact](https://example.com/artifact.md)\n",
		"empty.md":    "# A\n\nExtends:\n",
		// Unresolved, and the link counts only as the base.
		"unresolved.md": "# A\n\nExtends: [Ghost](ghost-base.md)\n",
		// Placed in another place: an error, and its link counts only there.
		"late.md":     "# A\n\ntext\nExtends: [Artifact](ghost.md)\n",
		"listed.md":   "# A\n\nExtends: [Artifact](artifact.md)\n\n## Pages\n- Extends: [Artifact](ghost.md)\n",
		"second.md":   "# A\n\nExtends: [Artifact](artifact.md)\n\nExtends: [Artifact](ghost.md)\n",
		"untitled.md": "Hello\nExtends: [Artifact](artifact.md)\n",
		"later.md":    "# A\n\nWhat this is.\n\nExtends: [Artifact](ghost.md)\n",
		// Beside a type line, in either order.
		"type-first.md":    "# A\n\nType: [Role](role.md)\nExtends: [Artifact](ghost.md)\n\n## Owner\nO.\n",
		"typed-after.md":   "# A\n\nType: [Role](role.md)\n\nExtends: [Artifact](ghost.md)\n\n## Owner\nO.\n",
		"typed-deep.md":    "# A\n\nType: [Role](role.md)\n\n## Owner\nO.\n\ntext\nExtends: [Artifact](ghost.md)\n",
		"typed-list.md":    "# A\n\nType: [Role](role.md)\n\n- Extends: [Artifact](ghost.md)\n\n## Owner\nO.\n",
		"type-second.md":   "# A\n\nExtends: [Artifact](artifact.md)\n\nType: [Role](ghost.md)\n",
		"extends-first.md": "# A\n\nExtends: [Artifact](artifact.md)\nType: [Role](ghost.md)\n",
		// In code, in lower case or indented as code, it is text.
		"fenced.md":   "# A\n\n```\nExtends: [Artifact](ghost.md)\n```\n\n    Extends: [Artifact](ghost2.md)\n",
		"spanned.md":  "# A\n\nText `Extends: [Artifact](ghost.md)`\n",
		"lower.md":    "# A\n\nextends: [Artifact](ghost-lower.md)\n",
		"indented.md": "# A\n\n Extends: [Artifact](artifact.md)\n\n## Pages\nP.\n",
	}),
		"angle.md:3: extends.line: "+bad,
		"code.md:3: extends.line: "+bad,
		"empty.md:3: extends.line: "+bad,
		"extends-first.md:4: type.line: "+both,
		"extra.md:3: extends.line: "+bad,
		"image.md:3: extends.line: "+bad,
		"late.md:4: extends.line: "+stray,
		"later.md:5: extends.line: "+stray,
		"listed.md:6: extends.line: "+stray,
		`lower.md:3: link.unresolved: no document at "ghost-lower.md"`,
		"second.md:5: extends.line: "+stray,
		"type-first.md:4: extends.line: "+both,
		"type-second.md:5: type.line: a type line stands only on the first line after the title",
		"typed-after.md:5: extends.line: "+stray,
		"typed-deep.md:9: extends.line: "+stray,
		"typed-list.md:5: extends.line: "+stray,
		`unresolved.md:3: extends.unresolved: no document at "ghost-base.md"`,
		"untitled.md:1: title: missing level-1 title",
		"untitled.md:2: extends.line: "+stray,
		"web.md:3: extends.line: "+bad,
	)
}

func TestTypeLineStaysForObjects(t *testing.T) {
	// A type that is the object of another type inherits nothing from it: its
	// own sections are those of its documents, and no chain passes through Type.
	files := map[string]string{
		"meta.md": "# Meta\n\nA kind of type.\n\n## Purpose\nWhy.\n",
		"loan.md": "# Loan\n\nType: [Meta](meta.md)\n\n## Purpose\nLending.\n\n## Due\nWhen.\n",
		"hold.md": "# Hold\n\nType: [Loan](loan.md)\n\n## Purpose\nA hold.\n\n## Due\nTomorrow.\n",
	}
	expect(t, checkFiles(t, files))
	s := specOf(t, files)
	if got := len(s.ChainSections(mustDoc(t, s, "loan.md"))); got != 2 {
		t.Errorf("loan has %d sections", got)
	}
	var objects []string
	for _, o := range s.Objects(mustDoc(t, s, "meta.md")) {
		objects = append(objects, o.Path)
	}
	expect(t, objects, "loan.md")
}

func TestExtendsWithUnresolvedOrBrokenBase(t *testing.T) {
	// Objects of a type whose base is missing are checked against what can be
	// collected.
	expect(t, checkFiles(t, map[string]string{
		"website.md":  "# Website\n\nExtends: [Artifact](artifact.md)\n\n## Pages\nP.\n",
		"site.md":     "# Site\n\nType: [Website](website.md)\n\n## Pages\n\n",
		"bad.md":      "# Bad\n\nExtends: nothing\n\n## Pages\nP.\n",
		"bad-site.md": "# Bad site\n\nType: [Bad](bad.md)\n",
	}),
		`bad-site.md:3: section.missing: missing section "Pages" required by bad.md`,
		"bad.md:3: extends.line: extends line must be exactly one link to a document",
		`site.md:5: section.empty: empty section "Pages"`,
		`website.md:3: extends.unresolved: no document at "artifact.md"`,
	)
}

func TestSectionsMatchWithoutCase(t *testing.T) {
	files := map[string]string{
		"base.md":    "# Base\n\nB.\n\n## Due Date\nWhen.\n\n## notes (optional)\nN.\n",
		"derived.md": "# Derived\n\nExtends: [Base](base.md)\n\n## DUE DATE\nThe day, more exactly.\n\n## Notes (Optional)\nN.\n\n## Own\nO.\n",
		"doc.md":     "# Doc\n\nType: [Derived](derived.md)\n\n## due date\nX.\n",
		"doc2.md":    "# Doc2\n\nType: [Derived](derived.md)\n\n## own\nX.\n",
	}
	expect(t, checkFiles(t, files),
		`doc.md:3: section.missing: missing section "Own" required by derived.md`,
		`doc2.md:3: section.missing: missing section "DUE DATE" required by derived.md`,
	)
	s := specOf(t, files)
	var got []string
	for _, c := range s.ChainSections(mustDoc(t, s, "derived.md")) {
		got = append(got, strings.Join([]string{c.Name, c.DeclaredBy, c.RequiredBy}, "/"))
	}
	expect(t, got, "DUE DATE/derived.md/derived.md", "Notes/derived.md/", "Own/derived.md/derived.md")
}

func TestInvalidExtendsLineMakesNoType(t *testing.T) {
	files := map[string]string{
		"base.md":   "# Base\n\nB.\n\n## One\nO.\n",
		"bad.md":    "# Bad\n\nExtends: the borrowing rules of the house\n\n## Two\nT.\n",
		"lost.md":   "# Lost\n\nExtends: [Nowhere](nowhere.md)\n\n## Three\nT.\n",
		"fine.md":   "# Fine\n\nExtends: [Base](base.md)\n",
		"object.md": "# Object\n\nType: [Bad](bad.md)\n",
	}
	expect(t, checkFiles(t, files),
		"bad.md:3: extends.line: extends line must be exactly one link to a document",
		`lost.md:3: extends.unresolved: no document at "nowhere.md"`,
		`object.md:3: section.missing: missing section "Two" required by bad.md`,
	)
	s := specOf(t, files)
	// bad.md is a type only because object.md names it.
	for p, want := range map[string]bool{"base.md": true, "fine.md": true, "lost.md": true, "bad.md": true, "object.md": false} {
		if got := s.IsType(mustDoc(t, s, p)); got != want {
			t.Errorf("IsType(%s) = %v", p, got)
		}
	}
	solo := specOf(t, map[string]string{"bad.md": files["bad.md"]})
	if solo.IsType(mustDoc(t, solo, "bad.md")) {
		t.Error("an invalid extends line makes a type")
	}
}
