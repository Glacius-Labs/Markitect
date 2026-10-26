package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

const contextScope = "scope: [{kind: policy/ScopedRule, property: policy/appliesTo}]\n"

// Extend actual M1 sources and check them normally; no constructed Model check
// results, context facts or runtime identities stand in for the bound sources.
func scopedContextInputs(t *testing.T, target, scope string) (project.InputSelection, project.RawInputs) {
	t.Helper()
	content := strings.ReplaceAll(documentedExecution(t), "\r\n", "\n") + scope
	selection, raw := executionInputs(t, content)
	for _, definition := range []struct{ name, content string }{
		{"policy/kind", "imports: [{package: markitect/core, as: Core}]\nid: ScopedRule\nkind: Core.Kind\nbody: Rule with explicit Package scope.\nspec: {properties: [appliesTo]}\n"},
		{"policy/property", "imports: [{package: markitect/core, as: Core}]\nid: appliesTo\nkind: Core.Property\nbody: Explicit local Package paths.\nspec: {valueType: string, minCount: 0, maxCount: -1}\n"},
		{"rules/global", "imports: [{package: policy}, {package: extra}]\nid: global\nkind: policy/ScopedRule\ntitle: Global rule\nbody: 'Scoped rule references [[extra/far]].'\nspec: {appliesTo: ['.']}\n"},
		{"rules/ancestor", fmt.Sprintf("imports: [{package: policy}]\nid: ancestor\nkind: policy/ScopedRule\nbody: Ancestor rule.\nspec: {appliesTo: [%q, %q]}\n", target, target)},
		{"rules/sibling", "imports: [{package: policy}]\nid: sibling\nkind: policy/ScopedRule\nbody: Catalog only.\nspec: {appliesTo: [catalog]}\n"},
		{"rules/absent", "imports: [{package: policy}]\nid: absent\nkind: policy/ScopedRule\nbody: No scope selection.\nspec: {}\n"},
		{"rules/empty", "imports: [{package: policy}]\nid: empty\nkind: policy/ScopedRule\nbody: Empty scope selection.\nspec: {appliesTo: []}\n"},
		{"lending/direct", "imports: [{package: software}, {package: extra}]\nid: direct\nkind: software/Concept\nbody: 'Direct [[extra/near|Neighbor]].'\nspec: {}\n"},
		{"extra/near", "imports: [{package: software}]\nid: near\nkind: software/Concept\nbody: 'Second hop [[far]].'\nspec: {}\n"},
		{"extra/far", "imports: [{package: software}]\nid: far\nkind: software/Concept\nbody: Further business edge.\nspec: {}\n"},
		{"unrelated/own", "imports: [{package: software}]\nid: own\nkind: software/Concept\nbody: No relation to lending or root.\nspec: {}\n"},
	} {
		raw.Local.Source.Entries = append(raw.Local.Source.Entries, model.Entry{Path: "model/" + definition.name + ".yaml", Kind: model.Regular, Bytes: []byte(definition.content)})
	}
	refreshLocalFiles(t, &raw)
	// refreshLocalFiles reconstructs the model acquisition area. Execution is a
	// separate bound input within that area and must retain its actual file mode.
	censusFile(t, &raw, ".markitect/execution.yaml", raw.Execution[0].File.Mode(), raw.Execution[0].File.Bytes())
	return selection, raw
}
