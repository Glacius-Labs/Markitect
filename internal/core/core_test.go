package core

import "testing"

const testAPI = "test.example/v1"

func ptr(v int) *int { return &v }
func genericDomain() DomainDefinition {
	ref := PropertyDefinition{Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: "Node"}}
	return DomainDefinition{APIVersion: testAPI, Kinds: map[string]KindDefinition{"Node": {Properties: map[string]PropertyDefinition{"dependsOn": ref, "layer": {Type: "string"}}}}, Relations: map[string]RelationDefinition{"dependsOn": {Field: "dependsOn", SourceKinds: []string{"Node"}, TargetKinds: []string{"Node"}, Context: true, Invalidate: true, Acyclic: true}}, Constraints: []ConstraintDefinition{{Name: "layer-required", Select: ResourceSelector{Kind: "Node"}, Assert: ConstraintAssertion{Op: "present", Field: "layer"}}}}
}
func mustRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	if err := r.AddDomain(genericDomain()); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestRegistryIsEmptyUntilVocabularyIsSupplied(t *testing.T) {
	r := NewRegistry()
	if len(r.Domains()) != 0 || r.IsKnownKind(APIVersion, "Project") || r.IsAPIVersionRegistered(APIVersion) {
		t.Fatalf("Core registry contains built-in authoring vocabulary: %#v", r.Domains())
	}
}
func TestBuildNormalizedResolvesGenericRelationsAndConstraints(t *testing.T) {
	r := mustRegistry(t)
	resources := []*Resource{{APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "a", Namespace: "n"}, Data: map[string]any{"layer": "app", "dependsOn": []any{map[string]any{"kind": "Node", "name": "b"}}}}, {APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "b", Namespace: "n"}, Data: map[string]any{"layer": "core"}}}
	rels, relationDiagnostics := ResolveTypedRelationships(resources, r)
	if len(relationDiagnostics) != 0 {
		t.Fatalf("reference resolution diagnostics: %#v", relationDiagnostics)
	}
	g := BuildNormalized(resources, r, rels, nil, "", nil)
	if len(g.Edges[resources[0].GraphKey()]) != 1 || g.Edges[resources[0].GraphKey()][0] != resources[1].GraphKey() {
		t.Fatalf("typed relationship did not resolve: %#v", g.Edges)
	}
	if len(g.PolicyResults) != 2 || g.PolicyResults[0].Status != PolicyPassed {
		t.Fatalf("generic constraints not evaluated: %#v", g.PolicyResults)
	}
}
func TestBuildNormalizedReportsGenericRelationCycle(t *testing.T) {
	r := mustRegistry(t)
	resources := []*Resource{{APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "a", Namespace: "n"}, Data: map[string]any{"layer": "app", "dependsOn": []any{map[string]any{"kind": "Node", "name": "b"}}}}, {APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "b", Namespace: "n"}, Data: map[string]any{"layer": "core", "dependsOn": []any{map[string]any{"kind": "Node", "name": "a"}}}}}
	rels, relationDiagnostics := ResolveTypedRelationships(resources, r)
	if len(relationDiagnostics) != 0 {
		t.Fatalf("reference resolution diagnostics: %#v", relationDiagnostics)
	}
	g := BuildNormalized(resources, r, rels, nil, "", nil)
	found := false
	for _, d := range g.Diagnostics {
		if d.Code == "relation.cycle" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected generic declared-cycle diagnostic, got %#v", g.Diagnostics)
	}
}
func TestPolicyExceptionsAreExplicitGenericInputs(t *testing.T) {
	r := mustRegistry(t)
	resources := []*Resource{{APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "a", Namespace: "n"}, Data: map[string]any{}}}
	first := BuildNormalized(resources, r, nil, nil, "", nil)
	result := first.PolicyResults[0]
	exception := PolicyException{Name: "waiver", APIVersion: testAPI, Constraint: result.Constraint, Subject: result.Subject, ConstraintDigest: result.ConstraintDigest, SubjectDigest: result.SubjectDigest, Rationale: "bounded exception", Owner: "team", Decision: "accepted", ExpiresOn: "2026-10-03"}
	waived := BuildNormalized(resources, r, nil, []PolicyException{exception}, "2026-10-01", nil)
	if waived.PolicyResults[0].Status != PolicyWaived || waived.PolicyResults[0].PolicyDate != "2026-10-01" {
		t.Fatalf("explicit waiver was not applied: %#v", waived.PolicyResults)
	}
	expired := BuildNormalized(resources, r, nil, []PolicyException{exception}, "2026-10-03", nil)
	if expired.PolicyResults[0].Status != PolicyFailed || !hasDiagnostic(expired, "policy.exception.expired") {
		t.Fatalf("expiry boundary did not remain exclusive: %#v", expired)
	}
}
func hasDiagnostic(g *Graph, code string) bool {
	for _, d := range g.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestGenericSameTargetConstraintSurvivesNormalizedGraphBoundary(t *testing.T) {
	api := "same.example/v1"
	ref := func(kind string) PropertyDefinition { return PropertyDefinition{Type: "ref", RefKind: kind} }
	arrayRef := func(kind string) PropertyDefinition {
		return PropertyDefinition{Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: kind}}
	}
	domain := DomainDefinition{APIVersion: api, Kinds: map[string]KindDefinition{
		"Case":    {Properties: map[string]PropertyDefinition{"module": arrayRef("Module"), "feature": ref("Feature")}},
		"Feature": {Properties: map[string]PropertyDefinition{"module": ref("Module")}},
		"Module":  {Properties: map[string]PropertyDefinition{"label": {Type: "string"}}},
	}, Relations: map[string]RelationDefinition{
		"belongs":  {Field: "module", SourceKinds: []string{"Case", "Feature"}, TargetKinds: []string{"Module"}},
		"realizes": {Field: "feature", SourceKinds: []string{"Case"}, TargetKinds: []string{"Feature"}},
	}, Constraints: []ConstraintDefinition{{Name: "same-module", Select: ResourceSelector{Kind: "Case"}, Assert: ConstraintAssertion{Op: "same-target", Left: []string{"belongs"}, Right: []string{"realizes", "belongs"}}}}}
	registry := NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	refValue := func(kind, name string) map[string]any {
		return map[string]any{"apiVersion": api, "kind": kind, "name": name}
	}
	resources := []*Resource{
		{APIVersion: api, Kind: "Case", Metadata: Metadata{Name: "create", Namespace: "app"}, Data: map[string]any{"module": []any{refValue("Module", "orders")}, "feature": refValue("Feature", "ordering")}},
		{APIVersion: api, Kind: "Feature", Metadata: Metadata{Name: "ordering", Namespace: "app"}, Data: map[string]any{"module": refValue("Module", "orders")}},
		{APIVersion: api, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "app"}, Data: map[string]any{"label": "Orders"}},
	}
	rels, relationDiagnostics := ResolveTypedRelationships(resources, registry)
	if len(relationDiagnostics) != 0 {
		t.Fatalf("reference resolution diagnostics: %#v", relationDiagnostics)
	}
	graph := BuildNormalized(resources, registry, rels, nil, "", nil)
	if len(graph.PolicyResults) != 1 || graph.PolicyResults[0].Status != PolicyPassed || graph.PolicyResults[0].Comparison == nil {
		t.Fatalf("generic same-target result missing: %#v %#v", graph.PolicyResults, graph.Diagnostics)
	}
}

func TestOpaqueHostDigestEncodingIsCopiedAndUsedOnlyForDigest(t *testing.T) {
	registry := mustRegistry(t)
	resource := &Resource{APIVersion: testAPI, Kind: "Node", Metadata: Metadata{Name: "node", Namespace: "n"}, Data: map[string]any{}}
	encoded := []byte("legacy-canonical-subject")
	key := resource.GraphKey()
	graph := BuildNormalized([]*Resource{resource}, registry, nil, nil, "", map[string][]byte{key: encoded})
	want, err := digestYAML(struct {
		Identity string `yaml:"identity"`
		Resource string `yaml:"resource"`
	}{key, string(encoded)})
	if err != nil {
		t.Fatal(err)
	}
	if graph.PolicyResults[0].SubjectDigest != want || graph.PolicyResults[0].Status != PolicyFailed {
		t.Fatalf("encoding affected semantic evaluation or digest: %+v", graph.PolicyResults[0])
	}
	encoded[0] = 'X'
	if graph.PolicyResults[0].SubjectDigest != want {
		t.Fatal("Core retained caller-mutable encoding bytes")
	}
}
