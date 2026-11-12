package spec

import "testing"

// A Kind may declare that its Definitions bind their Package and those below
// it without a link.
func TestCompileAKindDeclaresThatItBinds(t *testing.T) {
	m, ds := typed(t, map[string]string{
		"kinds/Law.md":  "---\nkind: Kind\nbinds: true\n---\n# Law\n",
		"kinds/Note.md": "---\nkind: Kind\nbinds: false\n---\n# Note\n",
		"quiet.md":      def("kinds/Law", "# Quiet\n"),
	})
	clean(t, ds)
	if s := m.Definitions["kinds/Law"].Schema; s == nil || !s.Binds || s.Unrealized || s.Limited {
		t.Fatalf("Law declares %+v", s)
	}
	if s := m.Definitions["kinds/Note"].Schema; s == nil || s.Binds {
		t.Fatalf("Note declares %+v", s)
	}
	_, ds = typed(t, map[string]string{"kinds/Odd.md": "---\nkind: Kind\nbinds: below\n---\n# Odd\n"})
	if d := expect(t, ds, "kind.schema", "binds is true or false"); d.Line != 3 {
		t.Fatalf("where: %v", d)
	}
	_, ds = typed(t, map[string]string{"goal.md": "---\nkind: typed/Goal\nbinds: true\n---\n# Goal\n"})
	expect(t, ds, "definition.front_matter", "only a Kind declares fields, parts, links, realized, binds and last")
}
