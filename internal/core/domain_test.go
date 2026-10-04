package core

import "testing"

func testSoftwareDomain() DomainDefinition {
	return DomainDefinition{
		Name: "software", APIVersion: "software.markitect.org/v1alpha1",
		Kinds: map[string]KindDefinition{
			"Module": {Required: []string{"layer"}, Properties: map[string]PropertyDefinition{
				"layer":     {Type: "string", Enum: []any{"application", "core"}},
				"dependsOn": {Type: "array", Items: &PropertyDefinition{Type: "ref"}},
			}},
			"Core": {Required: []string{"purpose"}, Properties: map[string]PropertyDefinition{"purpose": {Type: "string"}}},
		},
		Relations: map[string]RelationDefinition{
			"dependsOn": {Field: "dependsOn", SourceKinds: []string{"Module"}, TargetKinds: []string{"Module", "Core"}, Context: true, Invalidate: true, Acyclic: true},
		},
		Constraints: []ConstraintDefinition{{Name: "application-modules-use-core", Select: ResourceSelector{Kind: "Module", Labels: map[string]string{"layer": "application"}}, Assert: ConstraintAssertion{Op: "allowed-targets", Relation: "dependsOn", Values: []any{"Core"}}}},
	}
}

func TestRegistryRegistersClosedVersionedDomainAndReturnsCopies(t *testing.T) {
	r := NewRegistry()
	domain := testSoftwareDomain()
	if err := r.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	definition, ok := r.Lookup(domain.APIVersion, "Module")
	if !ok {
		t.Fatal("registered kind not found")
	}
	domain.Kinds["Module"].Properties["newField"] = PropertyDefinition{Type: "string"}
	definition.Properties["localMutation"] = PropertyDefinition{Type: "string"}
	again, _ := r.Lookup("software.markitect.org/v1alpha1", "Module")
	if _, ok := again.Properties["newField"]; ok {
		t.Fatal("registry retained caller mutation")
	}
	if _, ok := again.Properties["localMutation"]; ok {
		t.Fatal("lookup exposed registry map")
	}
	if err := r.AddDomain(DomainDefinition{APIVersion: "software.markitect.org/v1alpha2", Kinds: map[string]KindDefinition{"Module": {Properties: map[string]PropertyDefinition{"name": {Type: "string"}}}}}); err == nil {
		t.Fatal("accepted competing active version in one API group")
	}
	if err := r.ValidateData("software.markitect.org/v1alpha1", "Module", map[string]any{"layer": "application", "unknown": true}); err == nil {
		t.Fatal("accepted undeclared resource field")
	}
}

func TestBuildWithRegistrySeparatesContextInvalidationAndConstraintEvaluation(t *testing.T) {
	registry := NewRegistry()
	domain := testSoftwareDomain()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	module := &Resource{APIVersion: domain.APIVersion, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "engineering", Labels: map[string]string{"layer": "application"}}, Path: "domains/module.yaml", Data: map[string]any{"layer": "application", "dependsOn": []any{map[string]any{"kind": "Core", "name": "core"}}}}
	coreResource := &Resource{APIVersion: domain.APIVersion, Kind: "Core", Metadata: Metadata{Name: "core", Namespace: "engineering"}, Path: "domains/core.yaml", Data: map[string]any{"purpose": "shared foundation"}}
	graph := buildGenericTestGraph(t, registry, []*Resource{module, coreResource}, nil, "")
	if hasCode(graph, "constraint.application-modules-use-core") {
		t.Fatalf("Core target unexpectedly violates policy: %#v", graph.Diagnostics)
	}
	if len(graph.Edges[module.GraphKey()]) != 1 || graph.Edges[module.GraphKey()][0] != coreResource.GraphKey() {
		t.Fatalf("context edges = %#v", graph.Edges[module.GraphKey()])
	}
	if len(graph.InvalidationEdges[module.GraphKey()]) != 1 || graph.InvalidationEdges[module.GraphKey()][0] != coreResource.GraphKey() {
		t.Fatalf("invalidation edges = %#v", graph.InvalidationEdges[module.GraphKey()])
	}
	module.Data["dependsOn"] = []any{map[string]any{"kind": "Module", "name": "orders"}}
	cycle := buildGenericTestGraph(t, registry, []*Resource{module, coreResource}, nil, "")
	if !hasCode(cycle, "relation.cycle") {
		t.Fatalf("expected typed relation cycle diagnostic, got %#v", cycle.Diagnostics)
	}
	module.Data["dependsOn"] = []any{map[string]any{"kind": "Module", "name": "orders"}, map[string]any{"kind": "Core", "name": "core"}}
	duplicatePolicy := buildGenericTestGraph(t, registry, []*Resource{module, coreResource}, nil, "")
	if !hasCode(duplicatePolicy, "constraint.application-modules-use-core") {
		t.Fatalf("allowed-targets constraint did not reject Module target: %#v", duplicatePolicy.Diagnostics)
	}
}

func TestDomainSelectorsRequireLabelPresence(t *testing.T) {
	registry := NewRegistry()
	domain := testSoftwareDomain()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	module := &Resource{APIVersion: domain.APIVersion, Kind: "Module", Metadata: Metadata{Name: "unlabeled", Namespace: "engineering"}, Data: map[string]any{"layer": "application"}}
	graph := buildGenericTestGraph(t, registry, []*Resource{module}, nil, "")
	if hasCode(graph, "constraint.application-modules-use-core") {
		t.Fatalf("absent label incorrectly matched empty selector value: %#v", graph.Diagnostics)
	}
}

func TestRelationConstraintsValidateAndRespectSourceKinds(t *testing.T) {
	t.Run("reject selector kind outside relation sources", func(t *testing.T) {
		domain := testSoftwareDomain()
		domain.Constraints[0].Select = ResourceSelector{Kind: "Core"}
		if err := NewRegistry().AddDomain(domain); err == nil {
			t.Fatal("accepted relation constraint selecting a non-source kind")
		}
	})

	t.Run("unqualified selector counts only relation sources", func(t *testing.T) {
		domain := testSoftwareDomain()
		module := domain.Kinds["Module"]
		module.Properties["dependsOn"].Items.RefKind = "Module"
		domain.Kinds["Module"] = module
		coreKind := domain.Kinds["Core"]
		coreKind.Properties["dependsOn"] = PropertyDefinition{Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: "Module"}}
		domain.Kinds["Core"] = coreKind
		domain.Constraints = []ConstraintDefinition{{Name: "module-dependency-count", Assert: ConstraintAssertion{Op: "count", Relation: "dependsOn", Min: intPointer(2)}}}
		registry := NewRegistry()
		if err := registry.AddDomain(domain); err != nil {
			t.Fatal(err)
		}
		moduleResource := &Resource{APIVersion: domain.APIVersion, Kind: "Module", Metadata: Metadata{Name: "module", Namespace: "engineering"}, Data: map[string]any{"layer": "application", "dependsOn": []any{map[string]any{"kind": "Module", "name": "target"}}}}
		coreResource := &Resource{APIVersion: domain.APIVersion, Kind: "Core", Metadata: Metadata{Name: "core", Namespace: "engineering"}, Data: map[string]any{"purpose": "shared", "dependsOn": []any{map[string]any{"kind": "Module", "name": "target"}, map[string]any{"kind": "Module", "name": "module"}}}}
		target := &Resource{APIVersion: domain.APIVersion, Kind: "Module", Metadata: Metadata{Name: "target", Namespace: "engineering"}, Data: map[string]any{"layer": "core"}}
		graph := buildGenericTestGraph(t, registry, []*Resource{moduleResource, coreResource, target}, nil, "")
		if !hasCode(graph, "constraint.module-dependency-count") {
			t.Fatalf("count included non-source Core relation values: %#v", graph.Diagnostics)
		}
	})
}

func intPointer(v int) *int { return &v }

func buildGenericTestGraph(t *testing.T, registry *Registry, resources []*Resource, exceptions []PolicyException, policyDate string) *Graph {
	t.Helper()
	relationships, diagnostics := ResolveTypedRelationships(resources, registry)
	graph := BuildNormalized(resources, registry, relationships, exceptions, policyDate, nil)
	graph.Diagnostics = append(graph.Diagnostics, diagnostics...)
	return graph
}

func hasCode(graph *Graph, code string) bool {
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
