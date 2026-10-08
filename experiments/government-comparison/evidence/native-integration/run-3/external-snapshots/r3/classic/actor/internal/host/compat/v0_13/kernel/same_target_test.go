package core

import (
	"reflect"
	"strings"
	"testing"
)

const sameTargetAPI = "equality.tests.example/v1"

func sameTargetDomain() DomainDefinition {
	ref := func(kind string) PropertyDefinition { return PropertyDefinition{Type: "ref", RefKind: kind} }
	return DomainDefinition{
		APIVersion: sameTargetAPI,
		Kinds: map[string]KindDefinition{
			"UseCase": {Properties: map[string]PropertyDefinition{
				"module": {Type: "array", Items: ptrProperty(ref("Module"))}, "feature": ref("Feature"),
			}},
			"Feature": {Properties: map[string]PropertyDefinition{"module": ref("Module")}},
			"Module":  {Properties: map[string]PropertyDefinition{"name": {Type: "string"}}},
		},
		Relations: map[string]RelationDefinition{
			"belongsToModule": {Field: "module", SourceKinds: []string{"UseCase", "Feature"}, TargetKinds: []string{"Module"}},
			"realizesFeature": {Field: "feature", SourceKinds: []string{"UseCase"}, TargetKinds: []string{"Feature"}},
		},
		Constraints: []ConstraintDefinition{{
			Name: "feature-module-agrees", Select: ResourceSelector{Kind: "UseCase"},
			Assert: ConstraintAssertion{Op: "same-target", Left: []string{"belongsToModule"}, Right: []string{"realizesFeature", "belongsToModule"}},
		}},
	}
}

func sameTargetResources(useCaseModule, featureModule string) []*Resource {
	ref := func(kind, name string) map[string]any {
		return map[string]any{"apiVersion": sameTargetAPI, "kind": kind, "name": name}
	}
	return []*Resource{
		{APIVersion: sameTargetAPI, Kind: "UseCase", Metadata: Metadata{Name: "CreateOrder", Namespace: "app", Labels: map[string]string{"feature-ownership": "required"}}, Path: "resources/app/usecases/create-order.yaml", Line: 1, Data: map[string]any{"module": []any{ref("Module", useCaseModule)}, "feature": ref("Feature", "order-management")}},
		{APIVersion: sameTargetAPI, Kind: "Feature", Metadata: Metadata{Name: "order-management", Namespace: "app"}, Path: "resources/app/features/order-management.yaml", Line: 1, Data: map[string]any{"module": ref("Module", featureModule)}},
		{APIVersion: sameTargetAPI, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "app"}, Path: "resources/app/modules/orders.yaml", Data: map[string]any{"name": "Orders"}},
		{APIVersion: sameTargetAPI, Kind: "Module", Metadata: Metadata{Name: "billing", Namespace: "app"}, Path: "resources/app/modules/billing.yaml", Data: map[string]any{"name": "Billing"}},
	}
}

func ptrProperty(p PropertyDefinition) *PropertyDefinition { return &p }

func buildSameTargetGraph(t *testing.T, domain DomainDefinition, resources []*Resource) *Graph {
	return buildSameTargetGraphWithPolicy(t, domain, resources, nil)
}

func buildSameTargetGraphWithPolicy(t *testing.T, domain DomainDefinition, resources []*Resource, exceptions []PolicyException) *Graph {
	t.Helper()
	registry := NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	relationships, diagnostics := ResolveTypedRelationships(resources, registry)
	graph := BuildNormalized(resources, registry, relationships, exceptions, "", nil)
	graph.Diagnostics = append(graph.Diagnostics, diagnostics...)
	return graph
}

func TestSameTargetComparesResolvedGraphIdentityAndEmitsTrace(t *testing.T) {
	graph := buildSameTargetGraph(t, sameTargetDomain(), sameTargetResources("orders", "orders"))
	result := policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	if result.Status != PolicyPassed || result.Comparison == nil || result.Comparison.Left.Target != result.Comparison.Right.Target {
		t.Fatalf("matching paths did not pass with a trace: %+v", result)
	}
	if len(result.Comparison.Left.Steps) != 1 || len(result.Comparison.Right.Steps) != 2 {
		t.Fatalf("expected both ordered path traces: %+v", result.Comparison)
	}
	if len(graph.PolicyDependencies) != 6 {
		t.Fatalf("expected source and resolved target dependency evidence for each path hop, got %+v", graph.PolicyDependencies)
	}

	graph = buildSameTargetGraph(t, sameTargetDomain(), sameTargetResources("orders", "billing"))
	result = policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	if result.Status != PolicyFailed || result.Comparison == nil || result.Comparison.Left.Target == result.Comparison.Right.Target {
		t.Fatalf("different canonical targets did not fail: %+v", result)
	}
	found := false
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == "constraint.feature-module-agrees" {
			found = diagnostic.PolicyResult != nil && diagnostic.PolicyResult.APIVersion == sameTargetAPI && diagnostic.PolicyResult.Constraint == result.Constraint && diagnostic.PolicyResult.Subject == result.Subject
		}
	}
	if !found {
		t.Fatalf("same-target policy failure diagnostic lacks its exact result identity: %+v", graph.Diagnostics)
	}
}

func TestSameTargetRejectsInvalidPathStructurallyWithoutPolicyResult(t *testing.T) {
	resources := sameTargetResources("orders", "orders")
	resources[1].Data["module"] = nil // The selected Feature exists, but its second hop is absent.
	graph := buildSameTargetGraph(t, sameTargetDomain(), resources)
	if hasCode(graph, "constraint.path") == false {
		t.Fatalf("missing selected path hop did not produce structural diagnostic: %#v", graph.Diagnostics)
	}
	if result := policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder"); result.Constraint != "" {
		t.Fatalf("invalid path must not create a waivable PolicyResult: %+v", result)
	}
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == "constraint.path" && diagnostic.PolicyResult != nil {
			t.Fatalf("structural same-target traversal failure was tagged as policy failure: %+v", diagnostic)
		}
	}
	if len(graph.PolicyDependencies) == 0 {
		t.Fatal("invalid traversal must retain dependency evidence for the reached prefix")
	}
}

func TestSameTargetRejectsDuplicateRawReferenceEvenIfItResolvesToSameTarget(t *testing.T) {
	resources := sameTargetResources("orders", "orders")
	resources[0].Data["module"] = []any{
		map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "orders"},
		map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "orders"},
	}
	graph := buildSameTargetGraph(t, sameTargetDomain(), resources)
	if !hasCode(graph, "constraint.path") || policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder").Constraint != "" {
		t.Fatalf("duplicate raw declarations should invalidate the path even if graph resolution deduplicates them: %#v", graph.Diagnostics)
	}
}

func TestSameTargetRejectsTwoDistinctResolvedTargetsForOneHop(t *testing.T) {
	resources := sameTargetResources("orders", "orders")
	resources[0].Data["module"] = []any{
		map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "orders"},
		map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "billing"},
	}
	graph := buildSameTargetGraph(t, sameTargetDomain(), resources)
	if !hasCode(graph, "constraint.path") || policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder").Constraint != "" {
		t.Fatalf("two valid distinct targets should invalidate an exactly-one path: %#v", graph.Diagnostics)
	}
}

func TestSameTargetStructuralPathFailuresAreNotWaivable(t *testing.T) {
	baseline := buildSameTargetGraph(t, sameTargetDomain(), sameTargetResources("orders", "orders"))
	passing := policyResultFor(baseline, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	cases := []struct {
		name   string
		mutate func([]*Resource)
	}{
		{"missing selected left target", func(r []*Resource) { r[0].Data["module"] = []any{} }},
		{"unresolved right terminal", func(r []*Resource) {
			r[1].Data["module"] = map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "absent"}
		}},
		{"wrong kind", func(r []*Resource) {
			r[0].Data["feature"] = map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "name": "orders"}
		}},
		{"missing intermediate hop", func(r []*Resource) { r[1].Data["module"] = nil }},
		{"wrong api version", func(r []*Resource) {
			r[1].Data["module"] = map[string]any{"apiVersion": "other.tests.example/v1", "kind": "Module", "name": "orders"}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			resources := sameTargetResources("orders", "orders")
			test.mutate(resources)
			exceptions := []PolicyException{{Name: "old-waiver", APIVersion: passing.APIVersion, Constraint: passing.Constraint, Subject: passing.Subject, ConstraintDigest: passing.ConstraintDigest, SubjectDigest: passing.SubjectDigest, Rationale: "Review pending", Owner: "architecture", Decision: "accepted"}}
			graph := buildSameTargetGraphWithPolicy(t, sameTargetDomain(), resources, exceptions)
			if !hasCode(graph, "constraint.path") || policyResultFor(graph, passing.Constraint, passing.Subject).Constraint != "" || !hasCode(graph, "policy.exception.not-waivable") {
				t.Fatalf("structural path failure produced a result or was waivable: %#v", graph.Diagnostics)
			}
			for _, diagnostic := range graph.Diagnostics {
				if diagnostic.Code == "constraint.path" && (!strings.Contains(diagnostic.Message, passing.Subject) || !strings.Contains(diagnostic.Message, "left") && !strings.Contains(diagnostic.Message, "right")) {
					t.Errorf("structural diagnostic lacks subject/operand trace: %#v", diagnostic)
				}
			}
		})
	}
}

func TestSameTargetValidatesPathTypingAndAssertionShapeAtRegistration(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DomainDefinition)
	}{
		{"missing selector kind", func(d *DomainDefinition) { d.Constraints[0].Select.Kind = "" }},
		{"empty path", func(d *DomainDefinition) { d.Constraints[0].Assert.Left = nil }},
		{"depth three", func(d *DomainDefinition) {
			d.Constraints[0].Assert.Left = []string{"belongsToModule", "belongsToModule", "belongsToModule"}
		}},
		{"undefined relation", func(d *DomainDefinition) { d.Constraints[0].Assert.Left = []string{"missingRelation"} }},
		{"unrelated source kind", func(d *DomainDefinition) {
			d.Relations["belongsToModule"] = RelationDefinition{Field: "module", SourceKinds: []string{"Feature"}, TargetKinds: []string{"Module"}}
		}},
		{"wildcard target", func(d *DomainDefinition) {
			r := d.Relations["belongsToModule"]
			r.TargetKinds = []string{"*"}
			d.Relations["belongsToModule"] = r
		}},
		{"second hop source mismatch", func(d *DomainDefinition) {
			r := d.Relations["belongsToModule"]
			r.SourceKinds = []string{"UseCase"}
			d.Relations["belongsToModule"] = r
		}},
		{"every possible target must be accepted", func(d *DomainDefinition) {
			d.Kinds["Other"] = KindDefinition{Properties: map[string]PropertyDefinition{"name": {Type: "string"}}}
			r := d.Relations["realizesFeature"]
			r.TargetKinds = []string{"Feature", "Other"}
			d.Relations["realizesFeature"] = r
		}},
		{"no terminal overlap", func(d *DomainDefinition) {
			feature := d.Kinds["Feature"]
			feature.Properties["other"] = PropertyDefinition{Type: "ref", RefKind: "Other"}
			d.Kinds["Feature"] = feature
			d.Kinds["Other"] = KindDefinition{Properties: map[string]PropertyDefinition{"name": {Type: "string"}}}
			d.Relations["belongsToOther"] = RelationDefinition{Field: "other", SourceKinds: []string{"Feature"}, TargetKinds: []string{"Other"}}
			d.Constraints[0].Assert.Right[1] = "belongsToOther"
		}},
		{"irrelevant field", func(d *DomainDefinition) { d.Constraints[0].Assert.Field = "oops" }},
		{"left on legacy op", func(d *DomainDefinition) {
			d.Constraints[0].Assert.Op = "present"
			d.Constraints[0].Assert.Field = "name"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			domain := sameTargetDomain()
			test.edit(&domain)
			if err := NewRegistry().AddDomain(domain); err == nil {
				t.Fatal("expected invalid same-target definition to be rejected")
			}
		})
	}
}

func TestSameTargetSubjectDigestBindsIntermediateContentAndExceptionCannotWaiveInvalidPath(t *testing.T) {
	domain := sameTargetDomain()
	resources := sameTargetResources("orders", "orders")
	baseline := buildSameTargetGraph(t, domain, resources)
	finding := policyResultFor(baseline, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	if finding.Status != PolicyPassed {
		t.Fatalf("baseline path did not pass: %+v", finding)
	}
	resources[1].Data["module"] = nil
	exceptions := []PolicyException{{Name: "path-waiver", APIVersion: sameTargetAPI, Constraint: finding.Constraint, Subject: finding.Subject, ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "Historical exception", Owner: "architecture", Decision: "accepted"}}
	graph := buildSameTargetGraphWithPolicy(t, domain, resources, exceptions)
	if !hasCode(graph, "policy.exception.not-waivable") || !hasCode(graph, "constraint.path") {
		t.Fatalf("invalid path should reject the old exception and keep structural finding: %#v", graph.Diagnostics)
	}
	if strings.Contains(policyResultFor(graph, finding.Constraint, finding.Subject).Status, PolicyWaived) {
		t.Fatal("invalid path was waived")
	}
}

func TestSameTargetFailureCanBeWaivedAndBindsIntermediateContent(t *testing.T) {
	domain := sameTargetDomain()
	feature := domain.Kinds["Feature"]
	feature.Properties["note"] = PropertyDefinition{Type: "string"}
	domain.Kinds["Feature"] = feature
	resources := sameTargetResources("orders", "billing")
	resources[1].Data["note"] = "reviewed"
	baseline := buildSameTargetGraph(t, domain, resources)
	finding := policyResultFor(baseline, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	if finding.Status != PolicyFailed || finding.Comparison == nil {
		t.Fatalf("mismatch should produce a failed trace: %+v", finding)
	}
	originalMessage := finding.Message
	exceptions := []PolicyException{{Name: "known-module-drift", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: finding.Subject, ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "Feature ownership migration is scheduled", Owner: "architecture", Decision: "accepted until migration"}}
	waived := buildSameTargetGraphWithPolicy(t, domain, resources, exceptions)
	waivedResult := policyResultFor(waived, finding.Constraint, finding.Subject)
	if waivedResult.Status != PolicyWaived || waivedResult.Message != originalMessage || waivedResult.Comparison == nil {
		t.Fatalf("exception did not waive only the policy mismatch while preserving explanation: %+v", waivedResult)
	}
	resources[1].Data["note"] = "changed intermediate content"
	stale := buildSameTargetGraphWithPolicy(t, domain, resources, exceptions)
	if !hasCode(stale, "policy.exception.stale") || policyResultFor(stale, finding.Constraint, finding.Subject).Status != PolicyFailed {
		t.Fatalf("editing intermediate resource content must stale exact-path evidence: %#v", stale.Diagnostics)
	}
}

func TestSameTargetConstraintDigestBindsPathsRelationsAndConsumedSchema(t *testing.T) {
	domain := sameTargetDomain()
	resources := sameTargetResources("orders", "billing")
	baseline := buildSameTargetGraph(t, domain, resources)
	finding := policyResultFor(baseline, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	mutations := []struct {
		name string
		edit func(*DomainDefinition)
	}{
		{"left path", func(d *DomainDefinition) {
			d.Relations["ownsModule"] = d.Relations["belongsToModule"]
			d.Constraints[0].Assert.Left[0] = "ownsModule"
		}},
		{"right path", func(d *DomainDefinition) {
			d.Relations["featureModule"] = d.Relations["belongsToModule"]
			d.Constraints[0].Assert.Right[1] = "featureModule"
		}},
		{"full relation descriptor", func(d *DomainDefinition) {
			r := d.Relations["belongsToModule"]
			r.Context = true
			d.Relations["belongsToModule"] = r
		}},
		{"consumed property schema", func(d *DomainDefinition) {
			useCase := d.Kinds["UseCase"]
			items := *useCase.Properties["module"].Items
			items.RefKind = ""
			module := useCase.Properties["module"]
			module.Items = &items
			useCase.Properties["module"] = module
			d.Kinds["UseCase"] = useCase
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			changed := sameTargetDomain()
			mutation.edit(&changed)
			resources := sameTargetResources("orders", "billing")
			exceptions := []PolicyException{{Name: "old-policy", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: finding.Subject, ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "Previous relationship meaning was reviewed", Owner: "architecture", Decision: "accepted"}}
			graph := buildSameTargetGraphWithPolicy(t, changed, resources, exceptions)
			if !hasCode(graph, "policy.exception.stale") || policyResultFor(graph, finding.Constraint, finding.Subject).Status != PolicyFailed {
				t.Fatalf("changing %s must stale the bound exception: %#v", mutation.name, graph.Diagnostics)
			}
		})
	}
}

func TestSameTargetUsesCanonicalNamespaceIdentity(t *testing.T) {
	resources := sameTargetResources("orders", "orders")
	resources = append(resources, &Resource{APIVersion: sameTargetAPI, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "other"}, Path: "resources/other/modules/orders.yaml", Data: map[string]any{"name": "Other Orders"}})
	resources[0].Data["module"] = map[string]any{"apiVersion": sameTargetAPI, "kind": "Module", "namespace": "other", "name": "orders"}
	graph := buildSameTargetGraph(t, sameTargetDomain(), resources)
	result := policyResultFor(graph, "feature-module-agrees", "app/equality.tests.example/v1/UseCase/CreateOrder")
	if result.Status != PolicyFailed || result.Comparison == nil || result.Comparison.Left.Target == result.Comparison.Right.Target {
		t.Fatalf("same kind/name in different namespaces must remain different canonical targets: result=%+v all=%+v diagnostics=%+v", result, graph.PolicyResults, graph.Diagnostics)
	}
}

func TestSameTargetUsesCanonicalPackageIdentity(t *testing.T) {
	registry := NewRegistry()
	if err := registry.AddDomain(sameTargetDomain()); err != nil {
		t.Fatal(err)
	}
	ref := func(kind, name, packageName string) map[string]any {
		return map[string]any{"apiVersion": sameTargetAPI, "kind": kind, "name": name, "package": packageName}
	}
	subject := &Resource{APIVersion: sameTargetAPI, Kind: "UseCase", Metadata: Metadata{Name: "CreateOrder", Namespace: "app"}, Package: "pkg-a", Data: map[string]any{"module": []any{ref("Module", "orders", "pkg-a")}, "feature": ref("Feature", "order-management", "pkg-a")}}
	feature := &Resource{APIVersion: sameTargetAPI, Kind: "Feature", Metadata: Metadata{Name: "order-management", Namespace: "app"}, Package: "pkg-a", Data: map[string]any{"module": ref("Module", "orders", "pkg-b")}}
	left := &Resource{APIVersion: sameTargetAPI, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "app"}, Package: "pkg-a", Data: map[string]any{"name": "Package A Orders"}}
	right := &Resource{APIVersion: sameTargetAPI, Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "app"}, Package: "pkg-b", Data: map[string]any{"name": "Package B Orders"}}
	resources := []*Resource{subject, feature, left, right}
	graph := &Graph{Resources: map[string]*Resource{}, Registry: registry, invalidPolicyPaths: map[string]bool{}}
	for _, resource := range resources {
		graph.Resources[resource.GraphKey()] = resource
	}
	graph.Relationships = []Relationship{
		{From: subject.GraphKey(), To: left.GraphKey(), Relation: "belongsToModule", DomainAPIVersion: sameTargetAPI},
		{From: subject.GraphKey(), To: feature.GraphKey(), Relation: "realizesFeature", DomainAPIVersion: sameTargetAPI},
		{From: feature.GraphKey(), To: right.GraphKey(), Relation: "belongsToModule", DomainAPIVersion: sameTargetAPI},
	}
	graph.EvaluateConstraints(nil, "")
	result := policyResultFor(graph, "feature-module-agrees", subject.GraphKey())
	if result.Status != PolicyFailed || result.Comparison == nil || result.Comparison.Left.Target == result.Comparison.Right.Target {
		t.Fatalf("same identity in separate packages must remain distinct canonical targets: %+v", result)
	}
}

func TestSameTargetBoundedWalkTerminatesAcrossCycleAndCycleFindingRemainsStructural(t *testing.T) {
	domain := sameTargetDomain()
	feature := domain.Kinds["Feature"]
	feature.Properties["parent"] = PropertyDefinition{Type: "ref", RefKind: "Feature"}
	domain.Kinds["Feature"] = feature
	domain.Relations["parentFeature"] = RelationDefinition{Field: "parent", SourceKinds: []string{"Feature"}, TargetKinds: []string{"Feature"}}
	domain.Constraints = append(domain.Constraints, ConstraintDefinition{Name: "finite-feature-walk", Select: ResourceSelector{Kind: "Feature"}, Assert: ConstraintAssertion{Op: "same-target", Left: []string{"parentFeature", "parentFeature"}, Right: []string{"parentFeature", "parentFeature"}}})
	resources := sameTargetResources("orders", "orders")
	resources[1].Data["parent"] = map[string]any{"apiVersion": sameTargetAPI, "kind": "Feature", "name": "feature-two"}
	resources = append(resources, &Resource{APIVersion: sameTargetAPI, Kind: "Feature", Metadata: Metadata{Name: "feature-two", Namespace: "app"}, Path: "resources/app/features/feature-two.yaml", Data: map[string]any{"parent": map[string]any{"apiVersion": sameTargetAPI, "kind": "Feature", "name": "order-management"}}})
	graph := buildSameTargetGraph(t, domain, resources)
	result := policyResultFor(graph, "finite-feature-walk", "app/equality.tests.example/v1/Feature/order-management")
	if result.Status != PolicyPassed || result.Comparison == nil || len(result.Comparison.Left.Steps) != 2 || result.Comparison.Left.Target != "app/equality.tests.example/v1/Feature/order-management" {
		t.Fatalf("bounded traversal did not terminate across a graph cycle: result=%+v all=%+v relationships=%+v diagnostics=%+v", result, graph.PolicyResults, graph.Relationships, graph.Diagnostics)
	}

	domain.Relations["parentFeature"] = RelationDefinition{Field: "parent", SourceKinds: []string{"Feature"}, TargetKinds: []string{"Feature"}, Acyclic: true}
	baseline := buildSameTargetGraph(t, domain, resources)
	result = policyResultFor(baseline, "finite-feature-walk", "app/equality.tests.example/v1/Feature/order-management")
	exceptions := []PolicyException{{Name: "cannot-waive-cycle", APIVersion: sameTargetAPI, Constraint: "finite-feature-walk", Subject: "app/equality.tests.example/v1/Feature/order-management", ConstraintDigest: result.ConstraintDigest, SubjectDigest: result.SubjectDigest, Rationale: "Cycle waiver attempt", Owner: "architecture", Decision: "requested"}}
	graph = buildSameTargetGraphWithPolicy(t, domain, resources, exceptions)
	if !hasCode(graph, "relation.cycle") || !hasCode(graph, "policy.exception.unneeded") {
		t.Fatalf("declared acyclic cycle must remain structural and cannot be waived: %#v", graph.Diagnostics)
	}
}

func TestSameTargetResultsAndDependenciesAreDeterministicOnRebuildAndReevaluation(t *testing.T) {
	domain := sameTargetDomain()
	resources := sameTargetResources("orders", "billing")
	first := buildSameTargetGraph(t, domain, resources)
	secondResources := append([]*Resource(nil), resources...)
	for left, right := 0, len(secondResources)-1; left < right; left, right = left+1, right-1 {
		secondResources[left], secondResources[right] = secondResources[right], secondResources[left]
	}
	second := buildSameTargetGraph(t, domain, secondResources)
	if !reflect.DeepEqual(first.PolicyResults, second.PolicyResults) || !reflect.DeepEqual(first.PolicyDependencies, second.PolicyDependencies) {
		t.Fatalf("resource input order changed policy results or dependency evidence\nfirst: %+v\nsecond: %+v", first.PolicyResults, second.PolicyResults)
	}
	results := append([]PolicyResult(nil), first.PolicyResults...)
	dependencies := append([]PolicyDependency(nil), first.PolicyDependencies...)
	first.EvaluateConstraints(nil, "")
	if !reflect.DeepEqual(results, first.PolicyResults) || !reflect.DeepEqual(dependencies, first.PolicyDependencies) {
		t.Fatalf("reevaluation accumulated or reordered same-target evidence\nfirst: %+v\nsecond: %+v", results, first.PolicyResults)
	}
}

func policyResultFor(graph *Graph, constraint, subject string) PolicyResult {
	for _, result := range graph.PolicyResults {
		if result.Constraint == constraint && result.Subject == subject {
			return result
		}
	}
	return PolicyResult{}
}
