package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func TestContextProvenanceRetainsEveryReachableRelationDeterministically(t *testing.T) {
	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample"}, Path: projectPath}, Spec: authoring.Spec{Areas: []authoring.Area{{Name: "area", Path: "docs/area", Rules: []core.Ref{{Kind: "Rule", Name: "area-policy"}}}}}}
	agent := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "root", Namespace: "area"}, Path: "docs/area/root.yaml"}, Spec: authoring.Spec{Text: "Root.", Uses: []core.Ref{{Kind: "Workflow", Name: "z-short"}, {Kind: "Workflow", Name: "a-long"}}}}
	long := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "a-long", Namespace: "area"}, Path: "docs/area/a-long.yaml"}, Spec: authoring.Spec{Text: "Long path.", Uses: []core.Ref{{Kind: "Workflow", Name: "deep"}}}}
	short := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "z-short", Namespace: "area"}, Path: "docs/area/z-short.yaml"}, Spec: authoring.Spec{Text: "Short path.", Uses: []core.Ref{{Kind: "Skill", Name: "shared"}}}}
	deep := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "deep", Namespace: "area"}, Path: "docs/area/deep.yaml"}, Spec: authoring.Spec{Text: "Deep path.", Uses: []core.Ref{{Kind: "Skill", Name: "shared"}}}}
	shared := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "shared", Namespace: "area"}, Path: "docs/area/shared.yaml"}, Spec: authoring.Spec{Text: "Shared."}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "area-policy", Namespace: "area"}, Path: "docs/area/policy.yaml"}, Spec: authoring.Spec{Text: "Area rule."}}
	parsed := parseContextProvenanceResources(t, []*authoring.Resource{&project, &agent, &long, &short, &deep, &shared, &rule})
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("fixture diagnostics: %+v", parsed.Diagnostics)
	}

	firstContext, err := CompileContext(parsed, "area/Agent/root", "test")
	if err != nil {
		t.Fatal(err)
	}
	secondContext, err := CompileContext(parsed, "area/Agent/root", "test")
	if err != nil {
		t.Fatal(err)
	}
	if firstContext.Digest != secondContext.Digest {
		t.Fatal("identical relationship provenance did not produce a stable context digest")
	}
	inputs := contextInputsByKey(firstContext)
	sharedInput, ok := inputs["area/Skill/shared"]
	if !ok {
		t.Fatal("context omitted shared Skill")
	}
	if len(sharedInput.Via) != 2 || sharedInput.Via[0].From != "area/Workflow/deep" || sharedInput.Via[1].From != "area/Workflow/z-short" {
		t.Fatalf("context provenance did not retain both deterministic paths: %+v", sharedInput.Via)
	}
	if sharedInput.Reason != "required by area/Workflow/z-short via uses" {
		t.Fatalf("human reason must identify the actual first BFS path, not the lexically first alternate: %q", sharedInput.Reason)
	}
	for _, relation := range sharedInput.Via {
		if relation.Relation != "uses" || relation.Path == "" || relation.DomainAPIVersion != "" {
			t.Errorf("builtin relation provenance is incomplete: %+v", relation)
		}
	}
	ruleInput := inputs["area/Rule/area-policy"]
	foundAreaRule := false
	for _, relation := range ruleInput.Via {
		if relation.Relation == "area.rules" && relation.Path == projectPath {
			foundAreaRule = true
		}
	}
	if !foundAreaRule {
		t.Fatalf("area.rules inclusion lacks provenance: %+v", ruleInput.Via)
	}
}

func TestContextProvenanceOmitsNonContextRelationAndIdentifiesDomainInput(t *testing.T) {
	s := domainInputSnapshot()
	s.Files["domains/software.yaml"] = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: software}
spec:
  apiVersion: software.markitect.io/v1alpha1
  kinds:
    Module:
      required: [intent]
      properties:
        intent: {type: string}
        links: {type: array, items: {type: ref, refKind: Module}}
  relations:
    observes:
      field: links
      sourceKinds: [Module]
      targetKinds: [Module]
      context: false
      invalidate: true
`)
	s.Files["resources/module.yaml"] = []byte(`apiVersion: software.markitect.io/v1alpha1
kind: Module
metadata: {name: survey, namespace: engineering}
spec:
  intent: Collect responses.
  links: [{kind: Module, name: other, namespace: engineering}]
`)
	s.Files["resources/other.yaml"] = []byte(`apiVersion: software.markitect.io/v1alpha1
kind: Module
metadata: {name: other, namespace: engineering}
spec: {intent: Other module.}
`)
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("fixture diagnostics: %+v", p.Diagnostics)
	}
	ctx, err := CompileContext(p, "engineering/software.markitect.io/v1alpha1/Module/survey", "test")
	if err != nil {
		t.Fatal(err)
	}
	inputs := contextInputsByKey(ctx)
	if _, included := inputs["engineering/software.markitect.io/v1alpha1/Module/other"]; included {
		t.Fatal("context:false relation was followed")
	}
	var foundDomain bool
	for _, input := range ctx.Inputs {
		if input.Role != "domain" {
			continue
		}
		foundDomain = true
		if input.DomainName != "software" || input.DomainAPIVersion != "software.markitect.io/v1alpha1" {
			t.Fatalf("domain source identity is missing from context evidence: %+v", input)
		}
	}
	if !foundDomain {
		t.Fatal("selected domain definition missing from context")
	}
}

func parseContextProvenanceResources(t *testing.T, resources []*authoring.Resource) *Project {
	t.Helper()
	s := &snapshot.Snapshot{ID: "context-provenance", Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		s.Files[resource.Path] = encodeResource(t, *resource)
		s.Modes[resource.Path] = "100644"
	}
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func contextInputsByKey(context *Context) map[string]ContextInput {
	inputs := make(map[string]ContextInput, len(context.Inputs))
	for _, input := range context.Inputs {
		inputs[input.Key] = input
	}
	return inputs
}

func TestContextRelationProvenanceIsValueStable(t *testing.T) {
	// The relationship explanation is part of the context fingerprint and is
	// independent from map iteration or the order in which callers inspect it.
	s := domainInputSnapshot()
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	a, err := CompileContext(p, "engineering/software.markitect.io/v1alpha1/Module/survey", "test")
	if err != nil {
		t.Fatal(err)
	}
	b, err := CompileContext(p, "engineering/software.markitect.io/v1alpha1/Module/survey", "test")
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest || !reflect.DeepEqual(a, b) {
		t.Fatal("context provenance or digest is nondeterministic")
	}
}
