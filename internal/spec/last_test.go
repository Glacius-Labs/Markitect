package spec

import "testing"

// A Kind may declare that its Definitions are built and verified last, on
// the whole project. What no file realizes is verified last without saying
// so, since it can only be judged on what the others built.
func TestCompileAKindDeclaresThatItIsVerifiedLast(t *testing.T) {
	m, ds := typed(t, map[string]string{
		"kinds/Promise.md": "---\nkind: Kind\nlast: true\n---\n# Promise\n",
		"kinds/Wish.md":    "---\nkind: Kind\nrealized: false\n---\n# Wish\n",
		"kinds/Note.md":    "---\nkind: Kind\nlast: false\n---\n# Note\n",
		"fast.md":          def("kinds/Promise", "# Fast\n"),
		"more.md":          def("kinds/Wish", "# More\n"),
		"plain.md":         def("kinds/Note", "# Plain\n"),
		"goal.md":          def("typed/Goal", "# Goal\n"),
	})
	clean(t, ds)
	if s := m.Definitions["kinds/Promise"].Schema; s == nil || !s.Last || s.Unrealized || s.Binds {
		t.Fatalf("Promise declares %+v", s)
	}
	for id, want := range map[string]bool{"fast": true, "more": true, "plain": false, "goal": false, "kinds/Promise": false, "nowhere": false} {
		if got := m.VerifiedLast(id); got != want {
			t.Errorf("%s is verified last: %v", id, got)
		}
	}
	_, ds = typed(t, map[string]string{"kinds/Odd.md": "---\nkind: Kind\nlast: soon\n---\n# Odd\n"})
	if d := expect(t, ds, "kind.schema", "last is true or false"); d.Line != 3 {
		t.Fatalf("where: %v", d)
	}
	_, ds = typed(t, map[string]string{"goal.md": "---\nkind: typed/Goal\nlast: true\n---\n# Goal\n"})
	expect(t, ds, "definition.front_matter", "only a Kind declares fields, parts, links, realized, binds and last")
}
