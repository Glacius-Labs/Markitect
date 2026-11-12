package spec

import "testing"

// A Kind may declare that no file realizes its Definitions.
func TestCompileAKindDeclaresThatItIsNotRealized(t *testing.T) {
	m, ds := typed(t, map[string]string{
		"kinds/Given.md": "---\nkind: Kind\nrealized: false\n---\n# Given\n",
		"kinds/Built.md": "---\nkind: Kind\nrealized: true\n---\n# Built\n",
		"fact.md":        def("kinds/Given", "# Fact\n"),
	})
	clean(t, ds)
	if s := m.Definitions["kinds/Given"].Schema; s == nil || !s.Unrealized || s.HasFields || s.HasParts {
		t.Fatalf("Given declares %+v", s)
	}
	if s := m.Definitions["kinds/Built"].Schema; s == nil || s.Unrealized {
		t.Fatalf("Built declares %+v", s)
	}
	_, ds = typed(t, map[string]string{"kinds/Odd.md": "---\nkind: Kind\nrealized: never\n---\n# Odd\n"})
	if d := expect(t, ds, "kind.schema", "realized is true or false"); d.Line != 3 {
		t.Fatalf("where: %v", d)
	}
	_, ds = typed(t, map[string]string{"goal.md": "---\nkind: typed/Goal\nrealized: false\n---\n# Goal\n"})
	expect(t, ds, "definition.front_matter", "only a Kind declares fields, parts, links, realized, binds and last")
}
