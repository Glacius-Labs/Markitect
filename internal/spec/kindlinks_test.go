package spec

import "testing"

// A Kind may declare which Kinds a link of its Definitions may name. The
// limit holds for the Definition's own sentences and for every field that
// does not name its Kinds itself.
func TestCompileAKindLimitsItsLinks(t *testing.T) {
	kinds := map[string]string{
		"kinds/Term.md":  "---\nkind: Kind\nlinks: kinds/Term\n---\n# Term\n",
		"kinds/Fact.md":  "---\nkind: Kind\nlinks: [kinds/Term, kinds/Fact]\nfields:\n  Source:\n  Because: {links: [kinds/Rule, Package]}\n---\n# Fact\n",
		"kinds/Rule.md":  "---\nkind: Kind\n---\n# Rule\n",
		"kinds/Alone.md": "---\nkind: Kind\nlinks:\n---\n# Alone\n",
	}
	with := func(files map[string]string) map[string]string {
		for name, data := range kinds {
			files[name] = data
		}
		return files
	}
	m, ds := typed(t, with(map[string]string{
		"salon.md":       def("kinds/Term", "# Salon\n\nA place with a [[chair]].\n"),
		"chair.md":       def("kinds/Term", "# Chair\n"),
		"open.md":        def("kinds/Fact", "# Open\n\nThe [[salon]] is open, see [[closed]].\n\n**Source:** the [[salon]].\n\n**Because:** [[rule]] and [[area]].\n"),
		"closed.md":      def("kinds/Fact", "# Closed\n"),
		"rule.md":        def("kinds/Rule", "# Rule\n\nA Kind that declares nothing links anything: [[open]], [[salon]].\n"),
		"area/area.md":   anchor("Area"),
		"area/inside.md": def("kinds/Alone", "# Inside\n"),
	}))
	clean(t, ds)
	if s := m.Definitions["kinds/Fact"].Schema; !s.Limited || len(s.Links) != 2 {
		t.Fatalf("Fact declares %+v", s)
	}
	if s := m.Definitions["kinds/Alone"].Schema; !s.Limited || len(s.Links) != 0 {
		t.Fatalf("Alone declares %+v", s)
	}

	_, ds = typed(t, with(map[string]string{
		"salon.md":       def("kinds/Term", "# Salon\n\nSee [[open]].\n"),
		"open.md":        def("kinds/Fact", "# Open\n\nBy [[rule]].\n\n**Source:** [[rule]] and [[area]].\n\n**Because:** the [[salon]].\n"),
		"rule.md":        def("kinds/Rule", "# Rule\n"),
		"area/area.md":   anchor("Area"),
		"area/inside.md": def("kinds/Alone", "# Inside\n\nSee [[area/other]].\n"),
		"area/other.md":  def("kinds/Alone", "# Other\n"),
	}))
	if d := expect(t, ds, "link.target", "a kinds/Term links a kinds/Term; open is a kinds/Fact"); d.Line != 6 {
		t.Fatalf("where: %v", d)
	}
	expect(t, ds, "link.target", "a kinds/Fact links one of kinds/Fact, kinds/Term; rule is a kinds/Rule")
	// A field that does not name its Kinds takes the Kind's.
	expect(t, ds, "link.target", "a kinds/Fact links one of kinds/Fact, kinds/Term; area is a Package")
	// A field that names its Kinds holds to those alone.
	expect(t, ds, "field.target", `"Because" of a kinds/Fact links one of Package, kinds/Rule; salon is a kinds/Term`)
	expect(t, ds, "link.target", "a kinds/Alone links nothing; area/other is a kinds/Alone")
	if len(ds) != 6 {
		t.Fatalf("want six findings:\n%v", ds)
	}

	_, ds = typed(t, with(map[string]string{"kinds/Odd.md": "---\nkind: Kind\nlinks: {a: b}\n---\n# Odd\n"}))
	expect(t, ds, "kind.schema", "links names Kinds by path")
}
