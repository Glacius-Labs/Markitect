package spec

import (
	"strings"
	"testing"
)

// stdKinds is a small stand-in for the std Library.
var stdKinds = LibrarySource{Name: "std", Version: "1.0.0", Dir: "std", Files: []File{
	{Path: "Rule.md", Data: []byte("---\nkind: Kind\n---\n# Rule\n\nA rule.\n")},
	{Path: "Concept.md", Data: []byte("---\nkind: Kind\n---\n# Concept\n\nA concept; see [[std/Rule]].\n")},
}}

func def(kind, body string) string {
	return "---\nkind: " + kind + "\n---\n" + body
}

// anchor is the anchor of a Package: the Definition of kind Package that
// carries its folder's name.
func anchor(title string) string {
	return def("Package", "# "+title+"\n")
}

func compile(t *testing.T, files map[string]string, libs ...LibrarySource) (*Spec, Diagnostics) {
	t.Helper()
	src := Source{Dir: ".markitect/spec", Libraries: append([]LibrarySource{stdKinds}, libs...)}
	for p, data := range files {
		src.Files = append(src.Files, File{Path: p, Data: []byte(data)})
	}
	return Compile(src)
}

func codes(ds Diagnostics) []string {
	var out []string
	for _, d := range ds {
		out = append(out, d.Code)
	}
	return out
}

func expect(t *testing.T, ds Diagnostics, code, fragment string) Diagnostic {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.String(), fragment) {
			return d
		}
	}
	t.Fatalf("no %s containing %q in:\n%v", code, fragment, ds)
	return Diagnostic{}
}

func clean(t *testing.T, ds Diagnostics) {
	t.Helper()
	if len(ds) != 0 {
		t.Fatalf("unexpected diagnostics:\n%v", ds)
	}
}

func TestCompileValidSpec(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"project.md":         def("std/Concept", "# Lending library\n\nMembers borrow copies. See [[members]] and [[catalog]].\n"),
		"policy.md":          def("std/Rule", "# Loan policy\n\nAt most five loans.\n"),
		"members/members.md": anchor("Members"),
		"members/member.md":  def("std/Concept", "# Member\n\nA person; follows [[policy|the policy]].\n"),
		"members/status.md":  "---\nkind: std/Concept\ninternal: true\n---\n# Status\n\nInternal detail of [[members/member]].\n",
		"members/a/a.md":     anchor("A"),
		"members/a/check.md": def("std/Rule", "# Check\n\nUses [[members/status]], visible below its Package.\n"),
		"catalog/catalog.md": anchor("Catalog"),
		"catalog/Copy.md":    def("Kind", "# Copy\n\nA project Kind.\n"),
		"catalog/book.md":    def("catalog/Copy", "# Book\n\nBorrowed by a [[members/member#Status|member]].\n\n`[[ignored/in/code]]`\n\n```\n[[also/ignored]]\n```\n"),
	})
	clean(t, ds)
	if got := m.Definitions["catalog/book"].Kind; got != "catalog/Copy" {
		t.Fatalf("book kind = %q", got)
	}
	if got := m.Definitions["members/member"].Kind; got != "std/Concept" {
		t.Fatalf("member kind = %q", got)
	}
	uses := m.Edges["catalog"]["members"]
	if len(uses) != 1 || uses[0].From != "catalog/book" || uses[0].Line != 6 {
		t.Fatalf("catalog -> members uses = %+v", uses)
	}
	if len(m.Edges[RootPackage]) != 0 {
		t.Fatalf("naming child Packages made edges: %+v", m.Edges[RootPackage])
	}
	if got := m.Packages[RootPackage].Children; strings.Join(got, ",") != "catalog,members" {
		t.Fatalf("root children = %v", got)
	}
	if got := m.Packages["members"].Definitions; strings.Join(got, ",") != "members/member,members/members,members/status" {
		t.Fatalf("members definitions = %v", got)
	}
	if got := m.Definitions["catalog/book"].Title; got != "Book" {
		t.Fatalf("title = %q", got)
	}
	if got := m.Packages["members/a"]; got.Anchor != "members/a/a" || got.Parent != "members" {
		t.Fatalf("members/a = %+v", got)
	}
	if got := m.Packages[RootPackage].Anchor; got != "" {
		t.Fatalf("Root has the anchor %q", got)
	}
}

// A folder is a namespace and nothing more: what lies in it belongs to the
// Package above. A Package is the folder its anchor lies in.
func TestCompileAFolderIsANamespace(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"project.md":                     def("std/Concept", "# Lending library\n\nIts aim is [[goals/fast]]; its parts are [[members]] and [[catalog]].\n"),
		"goals/fast.md":                  def("std/Rule", "# Fast\n"),
		"members/members.md":             anchor("Members"),
		"members/concepts/member.md":     def("std/Concept", "# Member\n\nHolds a [[members/concepts/card/card|card]] and follows [[members/rules/limit]].\n"),
		"members/rules/limit.md":         def("std/Rule", "# Limit\n\nFor every [[members/concepts/member]].\n"),
		"members/rules/secret.md":        "---\nkind: std/Rule\ninternal: true\n---\n# Secret\n",
		"members/concepts/card/card.md":  anchor("Card"),
		"members/concepts/card/digit.md": def("std/Rule", "# Digit\n\nUses [[members/rules/secret]], visible below its Package.\n"),
		"catalog/catalog.md":             anchor("Catalog"),
		"catalog/operations/lend.md":     def("std/Rule", "# Lend\n\nTo a [[members/concepts/member]].\n"),
	})
	// The member's link down into the Package card is the one error.
	if len(ds) != 1 {
		t.Fatalf("diagnostics:\n%v", ds)
	}
	expect(t, ds, "link.downward", "members links down to members/concepts/card/card inside its child Package members/concepts/card")
	if got := strings.Join(m.SortedPackages(), ","); got != ",catalog,members,members/concepts/card" {
		t.Fatalf("Packages = %q", got)
	}
	for id, pkg := range map[string]string{
		"goals/fast":                  RootPackage,
		"members/concepts/member":     "members",
		"members/rules/limit":         "members",
		"members/concepts/card/digit": "members/concepts/card",
		"catalog/operations/lend":     "catalog",
	} {
		if got := m.Definitions[id].Package; got != pkg {
			t.Errorf("%s lies in %q, want %q", id, got, pkg)
		}
	}
	card := m.Packages["members/concepts/card"]
	if card.Parent != "members" || card.Anchor != "members/concepts/card/card" {
		t.Fatalf("card = %+v", card)
	}
	if got := strings.Join(m.Packages["members"].Children, ","); got != "members/concepts/card" {
		t.Fatalf("children of members = %q", got)
	}
	if got := strings.Join(m.Successors("members/concepts/card"), ","); got != "members" {
		t.Fatalf("card depends on %q", got)
	}
	if got := strings.Join(m.Packages[RootPackage].Definitions, ","); got != "goals/fast,project" {
		t.Fatalf("Root holds %q", got)
	}
	// The two folders of members make no edge between them, and the link from
	// catalog is an edge between the Packages.
	if len(m.Edges["members"]) != 0 || len(m.Edges["catalog"]["members"]) != 1 {
		t.Fatalf("edges = %+v", m.Edges)
	}
	if m.Parent("members/concepts/card") != "members" || m.Parent("members/concepts") != RootPackage {
		t.Fatalf("Parent is wrong")
	}
	if m.PackageOf("members/rules/limit") != "members" || m.PackageOf("members/rules/gone") != "members" || m.PackageOf("members/gone/away") != "members" || m.PackageOf("old/gone") != RootPackage || m.PackageOf("gone") != RootPackage || m.PackageOf("goals/gone") != RootPackage {
		t.Fatalf("PackageOf is wrong")
	}
}

// A thing with parts is a folder that holds a Definition of its own name and
// its parts beside it. That makes no Package: the thing, its parts and the
// Definitions around them link each other freely, as in one namespace tree.
func TestCompileAThingWithPartsIsNoPackage(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"members/members.md":                             anchor("Members"),
		"members/operations/lend/lend.md":                def("std/Rule", "# Lend\n\nAllowed when [[members/operations/lend/conditions/in-stock]].\n"),
		"members/operations/lend/conditions/in-stock.md": def("std/Rule", "# In stock\n\nA condition of [[members/operations/lend/lend]].\n"),
		"members/rules/stock.md":                         def("std/Rule", "# Stock\n\nSee [[members/operations/lend/conditions/in-stock]].\n"),
	})
	clean(t, ds)
	if got := strings.Join(m.SortedPackages(), ","); got != ",members" {
		t.Fatalf("Packages = %q", got)
	}
	if !m.PlainFolders["members/operations/lend"] || m.Definitions["members/operations/lend/lend"].Package != "members" {
		t.Fatalf("lend is not a plain folder of members: %v", m.PlainFolders)
	}
	if len(m.Edges) != 0 {
		t.Fatalf("links inside one Package made edges: %+v", m.Edges)
	}
	// The folder itself is no link target, and the message names the thing.
	_, ds = compile(t, map[string]string{
		"members/members.md":              anchor("Members"),
		"members/operations/lend/lend.md": def("std/Rule", "# Lend\n"),
		"members/rules/stock.md":          def("std/Rule", "# Stock\n\nSee [[members/operations/lend]].\n"),
	})
	expect(t, ds, "link.unresolved", `link "members/operations/lend" names a folder, which is no Package; the Definition of that name is members/operations/lend/lend`)
}

// A broken anchor is one error: its folder stays a Package, so what lies in
// it and what links to it report nothing of their own. A kind that names an
// anchor is no Kind, and Core's two Kinds are written as they are.
func TestCompileABrokenAnchorIsOneError(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"project.md":         def("std/Rule", "# Project\n\nParts: [[catalog]].\n"),
		"catalog/catalog.md": "---\nkind: Package\n# never closed\n",
		"catalog/book.md":    def("std/Rule", "# Book\n\nBorrowed by a [[members/member]].\n"),
		"members/members.md": anchor("Members"),
		"members/member.md":  def("std/Rule", "# Member\n"),
	})
	if len(ds) != 1 || ds[0].Code != "definition.front_matter" {
		t.Fatalf("one broken anchor must give one error, got:\n%v", ds)
	}
	if m.Packages["catalog"] == nil || m.Definitions["catalog/book"].Package != "catalog" {
		t.Fatalf("Packages = %v", m.SortedPackages())
	}
	_, ds = compile(t, map[string]string{
		"members/members.md": anchor("Members"),
		"members/chair.md":   def("members/members", "# Chair, whose kind is an anchor\n"),
		"members/clerk.md":   def("package", "# Clerk, whose kind is misspelt\n"),
	})
	expect(t, ds, "kind.not_a_kind", "members/chair.md:2")
	expect(t, ds, "kind.unresolved", `kind "package" resolves to nothing; Core's two Kinds are written Kind and Package`)
	if len(ds) != 2 {
		t.Fatalf("diagnostics:\n%v", ds)
	}
}

// An anchor carries the name of its folder, and the spec folder needs none.
func TestCompilePackageDefinitions(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"root.md":            anchor("Root"),
		"members/about.md":   anchor("Members"),
		"members/member.md":  def("std/Rule", "# Member\n"),
		"Catalog/catalog.md": anchor("Catalog"),
		"loans/loans.md":     anchor("Loans"),
		"loans/loan.md":      def("Package.md", "# Loan, a second anchor\n"),
		"Package/x.md":       def("std/Rule", "# In a folder named as Core's Package\n"),
		"usage.md":           def("std/Rule", "# Usage\n\nAn anchor is a [[Package]] and of a [[Kind]].\n"),
	})
	expect(t, ds, "spec.package", ".markitect/spec/root.md: spec.package: a Package Definition makes the folder it lies in a Package; the spec folder is the Root Package already")
	expect(t, ds, "spec.package", ".markitect/spec/members/about.md: spec.package: a Package Definition carries the name of its folder, so this one is members/members.md")
	expect(t, ds, "spec.package", ".markitect/spec/Catalog/catalog.md: spec.package: a Package Definition carries the name of its folder, so this one is Catalog/Catalog.md")
	expect(t, ds, "spec.package", ".markitect/spec/loans/loan.md: spec.package: a Package Definition carries the name of its folder, so this one is loans/loans.md")
	expect(t, ds, "spec.reserved", `"Package" is Core's Package`)
	if len(ds) != 5 {
		t.Fatalf("diagnostics:\n%v", ds)
	}
	if got := strings.Join(m.SortedPackages(), ","); got != ",loans" {
		t.Fatalf("Packages = %q", got)
	}
	if got := m.Definitions["loans/loans"]; got.Kind != CorePackage || m.Definitions[CorePackage].Kind != CoreKind {
		t.Fatalf("the anchor is of %q", got.Kind)
	}
}

func TestCompileAFolderIsNoLinkTarget(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"project.md":                 def("std/Rule", "# Project\n\nSee [[members/rules]] and [[catalog]].\n"),
		"members/members.md":         anchor("Members"),
		"members/rules/limit.md":     def("std/Rule", "# Limit\n"),
		"members/rules/secret.md":    "---\nkind: std/Rule\ninternal: true\n---\n# Secret\n",
		"members/kinds/Role.md":      def("Kind", "# Role\n"),
		"members/kinds/treasurer.md": def("members/kinds", "# Treasurer\n"),
		"catalog/book.md":            def("std/Rule", "# Book\n\nNot for [[members/rules/secret]].\n"),
	})
	expect(t, ds, "link.unresolved", `link "members/rules" names a folder that is no Package: it holds no Definition of kind Package, members/rules/rules.md; link a Definition in it or the Package above`)
	expect(t, ds, "link.unresolved", `link "catalog" names a folder that is no Package`)
	expect(t, ds, "kind.unresolved", `kind "members/kinds" names a folder that is no Package: it holds no Definition of kind Package, members/kinds/kinds.md; name a Kind in it`)
	// catalog has no anchor, so its book lies in Root, above members.
	expect(t, ds, "link.downward", "Root links down to members/rules/secret")
	if len(ds) != 4 {
		t.Fatalf("diagnostics:\n%v", ds)
	}
}

func TestCompileFrontMatterErrors(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"none.md":     "# No front matter\n",
		"unknown.md":  "---\nkind: std/Rule\nowner: me\n---\n# Unknown key\n",
		"internal.md": "---\nkind: std/Rule\ninternal: yes please\n---\n# Bad internal\n",
		"open.md":     "---\nkind: std/Rule\n# Never closed\n",
		"nokind.md":   "---\ninternal: true\n---\n# No kind\n",
		"listkind.md": "---\nkind: [std/Rule]\n---\n# List kind\n",
		"yaml.md":     "---\nkind: : :\n---\n# Bad YAML\n",
	})
	expect(t, ds, "definition.front_matter", "none.md:1")
	expect(t, ds, "definition.front_matter", `unknown front matter key "owner"`)
	expect(t, ds, "definition.front_matter", "internal.md:3")
	expect(t, ds, "definition.front_matter", "never closed")
	expect(t, ds, "definition.front_matter", "nokind.md:1")
	expect(t, ds, "definition.front_matter", "listkind.md:2")
	expect(t, ds, "definition.front_matter", "not valid YAML")
}

func TestCompileTitleErrors(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"untitled.md": def("std/Rule", "Text before any heading.\n"),
		"level2.md":   def("std/Rule", "## Second level\n"),
		"twice.md":    def("std/Rule", "# One\n\ntext\n\n# Two\n"),
	})
	expect(t, ds, "definition.title", "untitled.md:4")
	expect(t, ds, "definition.title", "level2.md")
	expect(t, ds, "definition.title", "twice.md:8")
}

func TestCompileFileAndNameErrors(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"diagram.png":    "binary",
		"bad name.md":    def("std/Rule", "# Bad\n"),
		"Member.md":      def("std/Rule", "# Member\n"),
		"member.md":      def("std/Rule", "# member\n"),
		"loans.md":       def("std/Rule", "# Loans\n"),
		"loans/loan.md":  def("std/Rule", "# Loan\n"),
		"std/thing.md":   def("std/Rule", "# Shadows std\n"),
		"Kind.md":        def("Kind", "# Shadows Core\n"),
		"Kind/inside.md": def("std/Rule", "# Inside\n"),
		"ok/.hidden.md":  def("std/Rule", "# Hidden\n"),
		"ok/visible.md":  def("std/Rule", "# Visible\n"),
	})
	expect(t, ds, "spec.file", "diagram.png")
	expect(t, ds, "spec.name", "bad name.md")
	expect(t, ds, "spec.name", ".hidden.md")
	expect(t, ds, "spec.collision", "file Member and file member")
	expect(t, ds, "spec.collision", "file loans and folder loans")
	expect(t, ds, "spec.reserved", `"std" is the name of a Library`)
	expect(t, ds, "spec.reserved", `"Kind" is Core's Kind`)
}

func TestCompileKindErrors(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"a.md":       def("std/Missing", "# A\n"),
		"b.md":       def("nowhere/Thing", "# B\n"),
		"c.md":       def("a", "# C, whose kind is a Rule\n"),
		"d.md":       def("pkg", "# D, whose kind is a Package\n"),
		"pkg/pkg.md": anchor("Pkg"),
		"pkg/e.md":   def("std/Rule", "# E\n"),
	})
	expect(t, ds, "kind.unresolved", `kind "std/Missing": Library std has no Kind`)
	expect(t, ds, "kind.unresolved", `kind "nowhere/Thing"`)
	expect(t, ds, "kind.not_a_kind", "c.md:2")
	expect(t, ds, "kind.not_a_kind", "d.md:2")
}

func TestCompileLinkErrors(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"a.md": def("std/Rule", "# A\n\n[[missing/def]] and [[../escape]] and [[/abs]] and [[std]] and [[std/Nope]].\n"),
	})
	expect(t, ds, "link.unresolved", `"missing/def"`)
	expect(t, ds, "link.syntax", `"../escape"`)
	expect(t, ds, "link.syntax", `"/abs"`)
	expect(t, ds, "link.unresolved", `"std"`)
	expect(t, ds, "link.unresolved", `"std/Nope"`)
}

func TestCompileVisibility(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"members/members.md":  anchor("Members"),
		"members/status.md":   "---\nkind: std/Rule\ninternal: true\n---\n# Status\n",
		"members/sub/sub.md":  anchor("Sub"),
		"members/sub/use.md":  def("std/Rule", "# Use\n\n[[members/status]] is visible here.\n"),
		"members/peer.md":     def("std/Rule", "# Peer\n\n[[members/status]] is visible here.\n"),
		"catalog/catalog.md":  anchor("Catalog"),
		"catalog/outsider.md": def("std/Rule", "# Outsider\n\nLinks [[members/status]].\n"),
	})
	d := expect(t, ds, "link.internal", "members/status is internal to members")
	if d.File != ".markitect/spec/catalog/outsider.md" || d.Line != 6 {
		t.Fatalf("diagnostic at %s:%d", d.File, d.Line)
	}
	if len(ds) != 1 {
		t.Fatalf("expected one diagnostic, got:\n%v", ds)
	}
}

func TestCompileInternalKindIsVisibilityBound(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"members/members.md": anchor("Members"),
		"members/Secret.md":  "---\nkind: Kind\ninternal: true\n---\n# Secret\n",
		"catalog/catalog.md": anchor("Catalog"),
		"catalog/x.md":       def("members/Secret", "# X\n"),
	})
	expect(t, ds, "link.internal", "catalog/x.md:2")
}

func TestCompileDownwardLinks(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"project.md":         def("std/Rule", "# Project\n\nRealized in [[members]]; details in [[members/member]].\n"),
		"members/members.md": anchor("Members"),
		"members/member.md":  def("std/Rule", "# Member\n"),
		"members/Role.md":    def("Kind", "# Role\n"),
		"admin.md":           def("members/Role", "# Admin, a Root Definition of a child's Kind\n"),
	})
	expect(t, ds, "link.downward", "project.md:6")
	expect(t, ds, "link.downward", "admin.md:2")
	for _, d := range ds {
		if d.Code == "package.cycle" {
			t.Fatalf("a downward link is reported once, as downward: %v", d)
		}
	}
}

func TestCompileSidewaysCycle(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"members/members.md": anchor("Members"),
		"members/member.md":  def("std/Rule", "# Member\n\nBorrows a [[catalog/book]].\n"),
		"catalog/catalog.md": anchor("Catalog"),
		"catalog/book.md":    def("std/Rule", "# Book\n\nLent to a [[members/member]].\n"),
	})
	d := expect(t, ds, "package.cycle", "catalog, members")
	if !strings.Contains(d.Message, "catalog/book links [[members/member]]") || !strings.Contains(d.Message, "members/member links [[catalog/book]]") {
		t.Fatalf("cycle path not described: %s", d.Message)
	}
}

func TestCompileCycleThroughParent(t *testing.T) {
	// a links c; c links into a's child a/b; a/b depends on a by location.
	_, ds := compile(t, map[string]string{
		"a/a.md":   anchor("A"),
		"a/x.md":   def("std/Rule", "# X\n\nNeeds [[c/y]].\n"),
		"a/b/b.md": anchor("B"),
		"a/b/z.md": def("std/Rule", "# Z\n"),
		"c/c.md":   anchor("C"),
		"c/y.md":   def("std/Rule", "# Y\n\nUses [[a/b/z]].\n"),
	})
	d := expect(t, ds, "package.cycle", "a, a/b, c")
	if !strings.Contains(d.Message, "a/b lies inside a") {
		t.Fatalf("implicit edge not described: %s", d.Message)
	}
}

func TestCompileKindUseIsAnEdge(t *testing.T) {
	_, ds := compile(t, map[string]string{
		"members/members.md": anchor("Members"),
		"members/Role.md":    def("Kind", "# Role\n\nSee [[catalog/book]].\n"),
		"catalog/catalog.md": anchor("Catalog"),
		"catalog/book.md":    def("std/Rule", "# Book\n"),
		"catalog/curator.md": def("members/Role", "# Curator\n"),
	})
	expect(t, ds, "package.cycle", "catalog, members")
}

func TestCompileUpAndSidewaysAreFine(t *testing.T) {
	m, ds := compile(t, map[string]string{
		"rule.md":            def("std/Rule", "# Rule\n"),
		"members/members.md": anchor("Members"),
		"members/member.md":  def("std/Rule", "# Member\n\nFollows [[rule]].\n"),
		"catalog/catalog.md": anchor("Catalog"),
		"catalog/book.md":    def("std/Rule", "# Book\n\nLent to a [[members/member]], see [[members]].\n"),
		"catalog/a/a.md":     anchor("A"),
		"catalog/a/b/b.md":   anchor("B"),
		"catalog/a/b/c.md":   def("std/Rule", "# C\n\nDeep, uses [[catalog/book]] and [[rule]].\n"),
	})
	clean(t, ds)
	if len(m.Edges["catalog"]["members"]) != 2 {
		t.Fatalf("edges = %+v", m.Edges)
	}
	if got := m.Successors("catalog/a/b"); strings.Join(got, ",") != "catalog/a" {
		t.Fatalf("successors = %v", got)
	}
}

func TestCompileLibraryErrors(t *testing.T) {
	bad := LibrarySource{Name: "web", Version: "1.0.0", Dir: ".markitect/libraries/web", Files: []File{
		{Path: "Page.md", Data: []byte("---\nkind: Kind\ninternal: true\n---\n# Page\n")},
		{Path: "Home.md", Data: []byte("---\nkind: web/Page\n---\n# Home, not a Kind\n")},
		{Path: "nested/Thing.md", Data: []byte("---\nkind: Kind\n---\n# Nested\n")},
		{Path: "Link.md", Data: []byte("---\nkind: Kind\n---\n# Link\n\nInto the spec: [[members/member]], other Library: [[rest/Endpoint]].\n")},
	}}
	_, ds := compile(t, map[string]string{"members/member.md": def("std/Rule", "# Member\n")}, bad,
		LibrarySource{Name: "Bad Name", Dir: "x"}, LibrarySource{Name: "Package", Dir: "y"},
		LibrarySource{Name: "parts", Version: "1.0.0", Dir: "parts", Files: []File{{Path: "Area.md", Data: []byte("---\nkind: Package\n---\n# Area, an anchor in a Library\n")}}})
	expect(t, ds, "library.name", `"Package" cannot name a Library`)
	expect(t, ds, "library.not_a_kind", "parts/Area.md")
	expect(t, ds, "library.internal", "Page.md")
	expect(t, ds, "library.not_a_kind", "Home.md")
	expect(t, ds, "library.file", "nested/Thing.md")
	expect(t, ds, "link.unresolved", `"members/member": a Library Definition may link only`)
	expect(t, ds, "link.unresolved", `"rest/Endpoint"`)
	expect(t, ds, "library.name", `"Bad Name"`)
}

func TestCompileLibraryCrossLinks(t *testing.T) {
	rest := LibrarySource{Name: "rest", Version: "1.0.0", Dir: "rest", Files: []File{
		{Path: "Endpoint.md", Data: []byte("---\nkind: Kind\n---\n# Endpoint\n\nRealizes a [[std/Concept]]; it is a [[Kind]].\n")},
	}}
	m, ds := compile(t, map[string]string{"api/get.md": def("rest/Endpoint", "# GET /books\n")}, rest)
	clean(t, ds)
	if m.Definitions["api/get"].Kind != "rest/Endpoint" || len(m.Edges) != 0 {
		t.Fatalf("library use made a Package edge: %+v", m.Edges)
	}
}

func TestDiagnosticsSortedAndUnique(t *testing.T) {
	ds := Diagnostics{{Code: "b", File: "x", Line: 2}, {Code: "a", File: "x", Line: 2}, {Code: "a", File: "x", Line: 2}, {Code: "z", File: "a"}}
	got := strings.Join(codes(ds.Sorted()), ",")
	if got != "z,a,b" {
		t.Fatalf("sorted = %s", got)
	}
}
