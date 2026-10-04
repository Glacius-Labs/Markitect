package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestFindUsesLiteralCaseInsensitiveMatchingAndExactFilters(t *testing.T) {
	general := queryResource("Text", "general", "rollback", "docs/general/rollback.yaml", authoring.Spec{Text: "Migration needs a rollback plan.", Description: "Database changes"})
	other := queryResource("Text", "general", "migration", "docs/general/migration.yaml", authoring.Spec{Text: "Apply the migration."})
	project := queryProject(queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general"}}), general, other)
	matches, err := Find(project, FindQuery{Query: "ROLLBACK", Kind: "Text", Namespace: "general"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Key != "general/Text/rollback" || matches[0].Description != "Database changes" {
		t.Fatalf("unexpected match result: %#v", matches)
	}
	if got := matches[0]; got.Path != general.Path || got.Name != "rollback" {
		t.Fatalf("match lacks concise identity fields: %#v", got)
	}
	all, err := Find(project, FindQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Key != "/Project/project" || all[1].Key != "general/Text/migration" || all[2].Key != "general/Text/rollback" {
		t.Fatalf("empty query did not return all resources in stable key order: %#v", all)
	}
}

func TestExplainUsesLongestOwningAreaAndResolvedRelationships(t *testing.T) {
	projectResource := queryProjectResource([]authoring.Area{
		{Name: "general", Path: "docs/general"},
		{Name: "specific", Path: "docs/general/customer"},
	})
	text := queryResource("Text", "specific", "guide", "docs/general/customer/text/guide.yaml", authoring.Spec{Text: "Specific guidance."})
	workflow := queryResource("Workflow", "specific", "review", "docs/general/customer/workflows/review.yaml", authoring.Spec{Text: "Review workflow.", Uses: []core.Ref{{Kind: "Text", Name: "guide"}}})
	project := queryProject(projectResource, text, workflow)
	result, err := Explain(project, workflow.Key())
	if err != nil {
		t.Fatal(err)
	}
	if result.Area == nil || result.Area.Name != "specific" || result.Area.Path != "docs/general/customer" {
		t.Fatalf("did not report longest owner area: %#v", result.Area)
	}
	if len(result.Outgoing) != 1 || result.Outgoing[0].To != text.Key() || result.Outgoing[0].Relation != "uses" || result.Outgoing[0].Path != workflow.Path {
		t.Fatalf("unexpected resolved outgoing relationship: %#v", result.Outgoing)
	}
	textExplanation, err := Explain(project, text.Key())
	if err != nil {
		t.Fatal(err)
	}
	if len(textExplanation.Incoming) != 1 || textExplanation.Incoming[0].From != workflow.Key() || textExplanation.Incoming[0].Relation != "uses" {
		t.Fatalf("unexpected resolved incoming relationship: %#v", textExplanation.Incoming)
	}
}

func TestExplainContractShowsEveryDeclarationAndSelectedBinding(t *testing.T) {
	projectResource := queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general"}})
	contract := queryResource("Contract", "general", "review", "docs/general/contracts/review.yaml", authoring.Spec{
		Text: "Review contract.", Kind: "Agent", Input: []string{"change"}, Output: []string{"findings"},
		Uses: []core.Ref{{Kind: "Text", Name: "policy"}},
	})
	policy := queryResource("Text", "general", "policy", "docs/general/text/policy.yaml", authoring.Spec{Text: "Policy."})
	selected := queryResource("Agent", "general", "reviewer", "docs/general/agents/reviewer.yaml", authoring.Spec{
		Text: "Selected.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}},
	})
	alternative := queryResource("Agent", "general", "reviewer-alt", "docs/general/agents/reviewer-alt.yaml", authoring.Spec{
		Text: "Alternative.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}},
	})
	consumer := queryResource("Workflow", "general", "consume", "docs/general/workflows/consume.yaml", authoring.Spec{
		Text: "Consumer.", Needs: []core.Ref{{Name: "review"}},
	})
	projectResource.Spec.Bindings = []authoring.Binding{{
		Contract:       core.Ref{Kind: "Contract", Name: "review", Namespace: "general"},
		Implementation: core.Ref{Kind: "Agent", Name: "reviewer", Namespace: "general"},
	}}
	project := queryProject(projectResource, contract, policy, selected, alternative, consumer)
	result, err := Explain(project, contract.Key())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeclaredImplementations) != 2 || result.DeclaredImplementations[0].Key != selected.Key() || result.DeclaredImplementations[1].Key != alternative.Key() {
		t.Fatalf("declared implementations are incomplete or unstable: %#v", result.DeclaredImplementations)
	}
	if result.SelectedImplementation == nil || result.SelectedImplementation.Key != selected.Key() {
		t.Fatalf("selected binding is wrong: %#v", result.SelectedImplementation)
	}
	if len(result.Outgoing) != 1 || result.Outgoing[0].To != policy.Key() || result.Outgoing[0].Relation != "uses" {
		t.Fatalf("unexpected contract outgoing relationships: %#v", result.Outgoing)
	}
	if !containsRelationship(result.Incoming, selected.Key(), contract.Key(), "binding") || !containsRelationship(result.Incoming, alternative.Key(), contract.Key(), "implements") {
		t.Fatalf("contract explanation omitted binding provenance: %#v", result.Incoming)
	}
	implementation, err := Explain(project, selected.Key())
	if err != nil {
		t.Fatal(err)
	}
	if !containsRelationship(implementation.Outgoing, selected.Key(), contract.Key(), "implements") || !containsRelationship(implementation.Outgoing, selected.Key(), contract.Key(), "binding") {
		t.Fatalf("implementation explanation omitted direct Contract relations: %#v", implementation.Outgoing)
	}
}

func TestExplainRelationshipDisappearsWhenReferenceIsRemoved(t *testing.T) {
	text := queryResource("Text", "general", "guide", "docs/general/text/guide.yaml", authoring.Spec{Text: "Guide."})
	workflow := queryResource("Workflow", "general", "review", "docs/general/workflows/review.yaml", authoring.Spec{Text: "Review.", Uses: []core.Ref{{Kind: "Text", Name: "guide"}}})
	withRef := queryProject(queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general"}}), text, workflow)
	withoutWorkflow := *workflow
	withoutWorkflow.Spec.Uses = nil
	withoutRef := queryProject(queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general"}}), text, &withoutWorkflow)
	removed, err := Explain(withoutRef, text.Key())
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Incoming) != 0 || len(withoutRef.Graph.Edges[workflow.Key()]) != 0 {
		t.Fatalf("removed reference remained in graph/explanation: %#v, %#v", removed.Incoming, withoutRef.Graph.Edges[workflow.Key()])
	}
	prior, err := Explain(withRef, text.Key())
	if err != nil || len(prior.Incoming) != 1 {
		t.Fatalf("old graph did not retain prior incoming reference: %#v, %v", prior, err)
	}
}

func TestQueriesRejectProjectsWithDiagnostics(t *testing.T) {
	broken := queryResource("Workflow", "general", "broken", "docs/general/workflows/broken.yaml", authoring.Spec{Text: "Broken.", Uses: []core.Ref{{Kind: "Text", Name: "missing"}}})
	project := queryProject(queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general"}}), broken)
	if len(project.Graph.Diagnostics) == 0 {
		t.Fatal("fixture should contain graph diagnostics")
	}
	if _, err := Find(project, FindQuery{Query: "broken"}); err == nil {
		t.Fatal("Find returned a result over an invalid graph")
	}
	if _, err := Explain(project, broken.Key()); err == nil {
		t.Fatal("Explain returned resolved facts over an invalid graph")
	}
}

func TestGraphRelationshipsAreDeterministicAndDoNotChangeEdges(t *testing.T) {
	projectResource := queryProjectResource([]authoring.Area{{Name: "general", Path: "docs/general", Rules: []core.Ref{{Name: "policy"}}}})
	rule := queryResource("Rule", "general", "policy", "docs/general/rules/policy.yaml", authoring.Spec{Text: "Policy."})
	workflow := queryResource("Workflow", "general", "review", "docs/general/workflows/review.yaml", authoring.Spec{Text: "Review."})
	forward := authoring.Build([]*authoring.Resource{projectResource, rule, workflow})
	reverse := authoring.Build([]*authoring.Resource{workflow, rule, projectResource})
	if !reflect.DeepEqual(forward.Relationships, reverse.Relationships) {
		t.Fatalf("relationship order depends on input order:\n%#v\n%#v", forward.Relationships, reverse.Relationships)
	}
	if !containsRelationship(forward.Relationships, workflow.Key(), rule.Key(), "area.rules") || !contains(forward.Edges[workflow.Key()], rule.Key()) {
		t.Fatalf("provenance changed or omitted the existing adjacency edge: %#v, %#v", forward.Relationships, forward.Edges)
	}
}

func queryProject(resources ...*authoring.Resource) *Project {
	return &Project{Resources: resources, Graph: authoring.Build(resources), InputFiles: map[string][]string{}}
}

func queryProjectResource(areas []authoring.Area) *authoring.Resource {
	return &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "project"}, Path: "markitect.yaml"}, Spec: authoring.Spec{Areas: areas}}
}

func queryResource(kind, namespace, name, path string, spec authoring.Spec) *authoring.Resource {
	return &authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: kind, Metadata: core.Metadata{Name: name, Namespace: namespace}, Path: path}, Spec: spec}
}

func containsRelationship(relationships []core.Relationship, from, to, relation string) bool {
	for _, candidate := range relationships {
		if candidate.From == from && candidate.To == to && candidate.Relation == relation {
			return true
		}
	}
	return false
}
