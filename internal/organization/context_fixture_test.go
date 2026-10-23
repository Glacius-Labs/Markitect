package organization_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func richContextInput(t *testing.T) model.Input {
	t.Helper()
	input := localFixture(t, nil, "empty", "software")
	input.Root.Entries[0].Bytes = []byte(strings.Replace(string(input.Root.Entries[0].Bytes), "id: local-project\n", "id: local-project\nversion: '1.0'\n", 1))
	for _, d := range []struct{ id, kind, spec string }{
		{"Record", "Kind", "{properties: [edge, payload, unused, data]}"},
		{"Rule", "Kind", "{properties: [applies]}"},
		{"edge", "Property", "{name: links, type: reference, target: Record, minCount: 0, maxCount: -1}"},
		{"payload", "Property", "{type: object, properties: [edge, note], minCount: 0, maxCount: 1}"},
		{"note", "Property", "{type: string, minCount: 0, maxCount: 1}"},
		{"unused", "Property", "{type: string, minCount: 0, maxCount: 1}"},
		{"data", "Property", "{name: properties, type: string, minCount: 0, maxCount: 1}"},
		{"applies", "Property", "{name: covers, type: string, minCount: 0, maxCount: -1}"},
	} {
		input.Root.Entries = append(input.Root.Entries, model.Entry{Path: "model/types/" + strings.ToLower(d.id) + ".yaml", Kind: model.Regular,
			Bytes: []byte(fmt.Sprintf("imports: [{package: markitect/core, as: Core}]\nid: %s\nkind: Core.%s\nbody: Explicit context contract fixture.\nspec: %s\n", d.id, d.kind, d.spec))})
	}
	for _, d := range []struct{ pkg, id, kind, imports, body, spec string }{
		{"a", "own", "Record", ", {package: b}", "Own [[b/near|Neighbor]] and [[b/near]].", "{links: [b/near, b/near], payload: {links: [b/near], note: text}, properties: literal}"},
		{"a", "z-other", "Record", "", "Another local Definition.", "{}"},
		{"a/child", "child", "Record", "", "Separate child.", "{}"},
		{"aother", "other", "Record", "", "Separate sibling.", "{}"},
		{"b", "near", "Record", ", {package: c}", "Second hop [[c/far]].", "{links: [c/far]}"},
		{"c", "far", "Record", "", "Unselected second hop.", "{}"},
		{"incoming", "sender", "Record", ", {package: a}", "Incoming [[a/own]].", "{payload: {links: [a/own]}}"},
		{"rules", "global", "Rule", ", {package: c}", "Global rule links [[c/far]].", "{covers: ['.']}"},
		{"rules", "ancestor", "Rule", "", "Ancestor scope.", "{covers: [a]}"},
		{"rules", "repeat", "Rule", "", "Repeated scope.", "{covers: [a, a]}"},
		{"rules", "sibling", "Rule", "", "Sibling scope.", "{covers: [aother]}"},
		{"rules", "absent", "Rule", "", "No modeled scope.", "{}"},
		{"rules", "empty", "Rule", "", "Explicit empty modeled scope.", "{covers: []}"},
		{"é/子", "ö", "Record", "", "Exact UTF-8 Package.", "{}"},
	} {
		input.Root.Entries = append(input.Root.Entries, model.Entry{Path: "model/" + d.pkg + "/" + d.id + ".yaml", Kind: model.Regular,
			Bytes: []byte(fmt.Sprintf("imports: [{package: types}%s]\nid: %s\nkind: types/%s\ntitle: Original title\nbody: '%s'\nspec: %s\n", d.imports, d.id, d.kind, d.body, d.spec))})
	}
	// An independently valid supplied model can depend on this versioned local
	// model without being its dependency. Its incoming edge and global-looking
	// scope must not extend the local selection merely because it was supplied.
	input.Dependencies = append(input.Dependencies, model.SourceModel{Origin: "fixture/unused", ManifestPath: "model.yaml", Entries: []model.Entry{
		{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: unused-context\nversion: '0.1'\nroot: model\ndependencies: {local-project: '1.0'}\n")},
		{Path: "model/unused/foreign.yaml", Kind: model.Regular, Bytes: []byte("imports: [{package: types}, {package: a}]\nid: foreign\nkind: types/Rule\nbody: 'Foreign incoming [[a/own]].'\nspec: {covers: ['.']}\n")},
	}})
	return input
}

func scopeSelector() organization.Setting[organization.ScopeRule] {
	return setting(organization.ScopeRule{Kind: contextID("types", "Rule"), Property: contextID("types", "applies")}, "/scope/0")
}

func findCheckedDefinition(t *testing.T, checked *model.Result, id model.Identity) *model.Definition {
	t.Helper()
	for i := range checked.Models {
		for j := range checked.Models[i].Definitions {
			if checked.Models[i].Definitions[j].Identity == id {
				return &checked.Models[i].Definitions[j]
			}
		}
	}
	t.Fatalf("missing fixture Definition %+v", id)
	return nil
}
