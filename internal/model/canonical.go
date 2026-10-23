package model

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
)

type canonicalCandidate struct {
	definitions []Definition
	bytes       []byte
	digest      string
}

// Candidates stay private until the selected content closure and the applicable
// root lock have completed. Foreign content and all provenance remain outside
// the own-model projection.
func projectCanonical(scope sourceScope, contracts contractResult, values valueResult, bodies bodyResult) (canonicalCandidate, bool) {
	var candidate canonicalCandidate
	packages := map[string][]canonicalMember{}
	for _, sourceDefinition := range scope.definitions {
		id := sourceDefinition.identity
		contract := contracts.definitions[id]
		spec, hasSpec := values.specs[id]
		links, hasLinks := bodies.links[id]
		if contract == nil || contract.status != Passed || contract.kind == nil || !hasSpec || !hasLinks {
			return canonicalCandidate{}, false
		}
		typed, ok := canonicalTypedValue(spec)
		if !ok || spec.kind != ObjectValue {
			return canonicalCandidate{}, false
		}
		fields := typed.items[1]
		// Only actual Core meta declarations are unordered. An ordinary field
		// called properties or values keeps the order checked by its contract.
		for index := range fields.fields {
			field := &fields.fields[index]
			declaration := contract.category == "kind" && field.name == "properties" || contract.category == "property" && (field.name == "properties" || field.name == "values")
			if declaration {
				sortCanonicalDeclaration(&field.value)
			}
		}
		body, _ := lookup(sourceDefinition.envelope, "body")
		record := canonicalObject(
			canonicalMember{"kind", canonicalReference(*contract.kind)},
			canonicalMember{"body", canonicalString(body.text)},
			canonicalMember{"bodyLinks", canonicalNode{kind: 'a'}},
			canonicalMember{"spec", fields},
		)
		for _, link := range links {
			record.fields[2].value.items = append(record.fields[2].value.items, canonicalReference(link.Target))
		}
		definition := Definition{Identity: id, ModelID: scope.modelID, Kind: *contract.kind, Body: body.text, Spec: append([]Field(nil), spec.fields...), BodyLinks: links, Source: source(scope.origin, sourceDefinition.file)}
		if title, present := lookup(sourceDefinition.envelope, "title"); present {
			text := title.text
			definition.Title = &text
			record.fields = append(record.fields, canonicalMember{"title", canonicalString(text)})
		}
		candidate.definitions = append(candidate.definitions, definition)
		packages[id.Package] = append(packages[id.Package], canonicalMember{id.ID, record})
	}
	sort.Slice(candidate.definitions, func(i, j int) bool {
		a, b := candidate.definitions[i].Identity, candidate.definitions[j].Identity
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.ID < b.ID
	})
	dependencies := canonicalNode{kind: 'o'}
	for _, requirement := range scope.requirements {
		dependencies.fields = append(dependencies.fields, canonicalMember{requirement.id, canonicalString(requirement.version.text)})
	}
	metadata := canonicalObject(canonicalMember{"formatVersion", canonicalNode{kind: 'n', text: "1"}}, canonicalMember{"id", canonicalString(scope.modelID)}, canonicalMember{"dependencies", dependencies})
	if scope.versionDeclared {
		metadata.fields = append(metadata.fields, canonicalMember{"version", canonicalString(scope.version.text)})
	}
	inventory := canonicalNode{kind: 'o'}
	for pkg, definitions := range packages {
		inventory.fields = append(inventory.fields, canonicalMember{pkg, canonicalObject(definitions...)})
	}
	candidate.bytes = encodeCanonical(canonicalObject(canonicalMember{"format", canonicalString("markitect-model-digest/1")}, canonicalMember{"model", metadata}, canonicalMember{"packages", inventory}))
	candidate.digest = fmt.Sprintf("sha256:%x", sha256.Sum256(candidate.bytes))
	return candidate, true
}

func sortCanonicalDeclaration(value *canonicalNode) {
	items := value.items[1].items
	type item struct {
		node    canonicalNode
		encoded []byte
	}
	ordered := make([]item, len(items))
	for index, node := range items {
		ordered[index] = item{node: node, encoded: encodeCanonical(node)}
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i].encoded, ordered[j].encoded) < 0 })
	for index, item := range ordered {
		items[index] = item.node
	}
}
