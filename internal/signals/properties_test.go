package signals

import "testing"

// A type that has a property type line, or that a property type line names, is a
// type like any other: the text under its sections is no statement, its title
// is no mention, and the line itself is no prose.
func TestTypesByPropertyTypeLines(t *testing.T) {
	a := Analyze(spec(t, map[string]string{
		"types/shop.md":    "# Shop\n\nWhere a customer buys things.\n\n## Orders\nType: [Order](order.md)\n\nEvery order is listed here, always.\n\n## Currency\nType: `euro`, `dollar`\n\nOnly these two are allowed.\n",
		"types/order.md":   "# Order\n\nWhat a customer buys.\n\n## Items\nEach item has a price.\n",
		"types/orphan.md":  "# Orphan\n\n## State\nType: `a`\n\nAll states of the orphan.\n",
		"notes.md":         "# Notes\n\nA shop sells things, an order lists items, and the orphan waits.\n",
		"shops/town.md":    "# Town\n\nType: [Shop](../types/shop.md)\n\n## Orders\nNone yet.\n\n## Currency\neuro\n",
		"shops/corner.md":  "# Corner\n\nType: [Shop](../types/shop.md)\n\n## Orders\nNone.\n\n## Currency\ndollar\n",
		"shops/readme.txt": "ignored",
	}), References{})
	var got []string
	for _, st := range a.Statements {
		got = append(got, st.Doc+"|"+st.Text)
	}
	eq(t, got, []string{
		"notes.md|A shop sells things, an order lists items, and the orphan waits.",
		"shops/corner.md|None.",
		"shops/corner.md|dollar",
		"shops/town.md|None yet.",
		"shops/town.md|euro",
		"types/order.md|What a customer buys.",
		"types/shop.md|Where a customer buys things.",
	})
	// The titles of types are no mentions: Shop, Order and Orphan are types, one by
	// the line that names it, one by being named, one by its own line.
	eq(t, a.Mentions[0], []string(nil))
	for path, kind := range map[string]string{"types/shop": "type", "types/order": "type", "types/orphan": "type", "shops/town": "document", "notes": "document"} {
		if topic, ok := a.Topic(path); !ok || topic.Kind != kind {
			t.Errorf("%s is %s", path, topic.Kind)
		}
	}
	topic, _ := a.Topic("types/shop")
	eq(t, topic.Documents, []string{"shops/corner.md", "shops/town.md"})
}

// A property type line in a document with a type line is an error and makes it
// no type; it is no prose either.
func TestPropertyTypeLineOfAnObjectMakesNoType(t *testing.T) {
	a := Analyze(spec(t, map[string]string{
		"types/kind.md": "# Kind\n\nA kind.\n\n## Name\nWho.\n",
		"thing.md":      "# Thing\n\nType: [Kind](types/kind.md)\n\n## Name\nType: `x`\n\nAda\n",
	}), References{})
	if topic, _ := a.Topic("thing"); topic.Kind != "document" {
		t.Errorf("thing is %s", topic.Kind)
	}
	var got []string
	for _, st := range a.Statements {
		got = append(got, st.Doc+"|"+st.Text)
	}
	eq(t, got, []string{"thing.md|Ada", "types/kind.md|A kind."})
}
