package spec

import (
	"reflect"
	"testing"
)

// What binds a Package is each unit of a binding Kind that lies in the
// Package itself, a part with parts of its own among them, and it binds each
// unit a file realizes: not a Definition whose Kind declares that no file
// realizes it, and not a Kind (DEC-113).
func TestWhatBindsAndWhatIsBound(t *testing.T) {
	m, ds := typed(t, map[string]string{
		"kinds/Law.md":                "---\nkind: Kind\nbinds: true\n---\n# Law\n",
		"kinds/Given.md":              "---\nkind: Kind\nbinds: true\nrealized: false\n---\n# Given\n",
		"kinds/Wish.md":               "---\nkind: Kind\nrealized: false\n---\n# Wish\n",
		"kinds/Thing.md":              "---\nkind: Kind\n---\n# Thing\n",
		"laws/order/order.md":         def("kinds/Law", "# Order\n"),
		"laws/order/calm.md":          def("kinds/Law", "# Calm\n"),
		"laws/order/rows/rows.md":     def("kinds/Law", "# Rows\n"),
		"laws/order/rows/straight.md": def("kinds/Law", "# Straight\n"),
		"given.md":                    def("kinds/Given", "# Given\n"),
		"wish.md":                     def("kinds/Wish", "# Wish\n"),
		"members/members.md":          anchor("Members"),
		"members/laws/cards.md":       def("kinds/Law", "# Cards\n"),
		"members/member.md":           def("kinds/Thing", "# Member\n"),
		"members/wishes/more.md":      def("kinds/Wish", "# More\n"),
	})
	clean(t, ds)
	want := map[string][]string{
		RootPackage: {"given", "laws/order/order", "laws/order/rows/rows"},
		"members":   {"members/laws/cards"},
	}
	if got := m.Binding(); !reflect.DeepEqual(got, want) {
		t.Fatalf("binding = %v", got)
	}
	for id, bound := range map[string]bool{
		"laws/order/order": true, "laws/order/calm": true, "laws/order/rows/straight": true, "members/laws/cards": true,
		"members/member": true, "members/members": true,
		"given": false, "wish": false, "members/wishes/more": false, "kinds/Law": false, "typed/Goal": false, "gone": false,
	} {
		if got := m.Bound(id); got != bound {
			t.Errorf("bound %s = %v, want %v", id, got, bound)
		}
	}
}

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
