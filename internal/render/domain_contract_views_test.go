package render

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestDomainContractViewsProjectSchemaPolicyAndOutcomes(t *testing.T) {
	graph, _, subjects := domainContractGraph(t, []any{"Core"}, []core.PolicyResult{
		{APIVersion: "constitution.example.org/v1alpha1", Constraint: "requires-core", Subject: "policy/constitution.example.org/v1alpha1/Clause/passed", Status: core.PolicyPassed, Message: "Declared dependency target is allowed."},
		{APIVersion: "constitution.example.org/v1alpha1", Constraint: "requires-core", Subject: "policy/constitution.example.org/v1alpha1/Clause/failed", Status: core.PolicyFailed, Message: "found forbidden target Module"},
		{APIVersion: "constitution.example.org/v1alpha1", Constraint: "requires-core", Subject: "policy/constitution.example.org/v1alpha1/Clause/waived", Status: core.PolicyWaived, Message: "found forbidden target Module", ExceptionName: "allow-temporary-module", Rationale: "migration window | pending review\nwith follow-up", Owner: "platform team", Decision: "ADR-42", ExpiresOn: "2027-01-31", PolicyDate: "2026-10-01"},
	})
	outputs, owners, err := GenerateWithOwners(graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := "docs/markitect/_domains/constitution.example.org.v1alpha1.domain.md"
	contract := string(outputs[contractPath])
	for _, expected := range []string{
		"# Domain contract: Constitution",
		"**API version:** `constitution.example.org/v1alpha1`",
		"**Definition source:** `domains/constitution.yaml`",
		"## Normalized schema and policy",
		"requires-core",
		"op: allowed",
		"- Core",
		"**PASSED**",
		"**FAILED**",
		"**WAIVED**",
		"found forbidden target Module",
		"allow-temporary-module",
		"migration window \\| pending review with follow-up",
		"platform team",
		"ADR-42",
		"2027-01-31",
		"2026-10-01",
	} {
		if !strings.Contains(contract, expected) {
			t.Errorf("domain contract omitted %q:\n%s", expected, contract)
		}
	}
	if strings.Contains(contract, "[domains/constitution.yaml]") {
		t.Fatalf("domain contract should show logical source provenance without a source hyperlink:\n%s", contract)
	}
	assertNoTrailingWhitespace(t, contractPath, contract)
	if got := strings.Join(owners[contractPath], ","); got != "domain:constitution.example.org/v1alpha1/Constitution" {
		t.Errorf("domain contract owner = %q", got)
	}
	if outputs["docs/markitect/README.md"] == nil || outputs["docs/markitect/_domains/README.md"] == nil {
		t.Fatalf("domain views are not connected through both generated indices: %v", outputPaths(outputs))
	}
	if !strings.Contains(string(outputs["docs/markitect/README.md"]), "_domains/README.md") {
		t.Fatalf("root Markdown views index does not link to domain contracts:\n%s", outputs["docs/markitect/README.md"])
	}

	waivedView := string(outputs[subjects["waived"]])
	for _, expected := range []string{"**validators:** `[]`", "## Policy outcomes", "**WAIVED**", "found forbidden target Module", "migration window", "platform team", "ADR-42", "2027-01-31", "2026-10-01"} {
		if !strings.Contains(waivedView, expected) {
			t.Errorf("subject view omitted waived outcome field %q:\n%s", expected, waivedView)
		}
	}
	if strings.Contains(waivedView, "Waived: found forbidden target Module") {
		t.Errorf("subject view should preserve the original violation message separately from waiver status:\n%s", waivedView)
	}
	assertNoTrailingWhitespace(t, subjects["waived"], waivedView)
	for _, line := range strings.Split(contract, "\n") {
		if strings.Contains(line, "**WAIVED**") && countUnescapedPipes(line) != 6 {
			t.Errorf("waived result row has incorrect Markdown table cell count: %s", line)
		}
	}
}

func TestDomainContractViewsFollowDefinitionAndRequireMarkdownTarget(t *testing.T) {
	graph, _, _ := domainContractGraph(t, []any{"Core"}, nil)
	first, err := Generate(graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := "docs/markitect/_domains/constitution.example.org.v1alpha1.domain.md"
	graph.Registry = newTestDomainRegistry(t, []any{"Module"})
	second, err := Generate(graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(first[contractPath]) == string(second[contractPath]) || !strings.Contains(string(second[contractPath]), "- Module") {
		t.Fatalf("contract view did not follow the canonical structured policy change:\n%s", second[contractPath])
	}

	graph.Project.Spec.Targets = []string{"codex"}
	providerOnly, err := Generate(graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	for output := range providerOnly {
		if strings.HasPrefix(output, markdownViewsRoot+"/") {
			t.Fatalf("provider-only target unexpectedly generated generic Markdown view %s", output)
		}
	}
}

func TestDomainMarkdownPathEncodesCaseAndPunctuation(t *testing.T) {
	domain := core.DomainDefinition{Name: "Constitution", APIVersion: "constitution.example.org/V_1", Path: "domains/constitution.yaml"}
	got, err := DomainMarkdownPath(domain)
	if err != nil {
		t.Fatal(err)
	}
	if got != "docs/markitect/_domains/constitution.example.org._56_5f1.domain.md" {
		t.Fatalf("DomainMarkdownPath = %q", got)
	}
}

func TestApplicableConstraintsRespectRelationSourcesAndResourceCounts(t *testing.T) {
	registry := core.NewRegistry()
	domain := core.DomainDefinition{
		Name: "relationships", APIVersion: "relationships.example.org/v1", Path: "domains/relationships.yaml",
		Kinds: map[string]core.KindDefinition{
			"Module": {Properties: map[string]core.PropertyDefinition{"dependsOn": {Type: "array", Items: &core.PropertyDefinition{Type: "ref", RefKind: "Core"}}}},
			"Core":   {Properties: map[string]core.PropertyDefinition{"purpose": {Type: "string"}}},
		},
		Relations:   map[string]core.RelationDefinition{"dependsOn": {Field: "dependsOn", SourceKinds: []string{"Module"}, TargetKinds: []string{"Core"}}},
		Constraints: []core.ConstraintDefinition{{Name: "module-has-dependency", Select: core.ResourceSelector{}, Assert: core.ConstraintAssertion{Op: "count", Scope: "resource", Relation: "dependsOn", Min: intPointer(1)}}},
	}
	if err := registry.AddDomain(domain); err != nil {
		t.Fatal(err)
	}
	module := &core.Resource{APIVersion: domain.APIVersion, Kind: "Module", Metadata: core.Metadata{Name: "orders", Namespace: "engineering"}}
	coreResource := &core.Resource{APIVersion: domain.APIVersion, Kind: "Core", Metadata: core.Metadata{Name: "platform", Namespace: "engineering"}}
	graph := &core.Graph{Registry: registry}
	moduleConstraints := applicableConstraints(module, graph)
	if len(moduleConstraints) != 1 || !strings.Contains(moduleConstraints[0].Text, "Each selected resource's count of `dependsOn` relationship targets must be at least 1") {
		t.Fatalf("resource-scoped relation count wording = %#v", moduleConstraints)
	}
	if unrelated := applicableConstraints(coreResource, graph); len(unrelated) != 0 {
		t.Fatalf("constraint for dependsOn source kind Module appeared applicable to Core: %#v", unrelated)
	}
}

func domainContractGraph(t *testing.T, allowed []any, results []core.PolicyResult) (*core.Graph, core.DomainDefinition, map[string]string) {
	t.Helper()
	domain := testContractDomain(allowed)
	registry := newTestDomainRegistry(t, allowed)
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"markdown"}, Areas: []core.Area{{Name: "policy", Path: "resources"}}})
	graph := &core.Graph{Project: project, Registry: registry, Resources: map[string]*core.Resource{project.GraphKey(): project}, ResourceAreas: map[string]core.Area{}, PolicyResults: results}
	subjects := map[string]string{}
	for _, name := range []string{"passed", "failed", "waived"} {
		item := &core.Resource{APIVersion: domain.APIVersion, Kind: "Clause", Metadata: core.Metadata{Name: name, Namespace: "policy"}, Data: map[string]any{"category": name, "validators": []any{}}, Path: "resources/" + name + ".yaml"}
		graph.Resources[item.GraphKey()] = item
		graph.ResourceAreas[item.GraphKey()] = core.Area{Name: "policy", Path: "resources"}
		subjects[name], _ = MarkdownViewPath(graph, item)
	}
	return graph, domain, subjects
}

func newTestDomainRegistry(t *testing.T, allowed []any) *core.Registry {
	t.Helper()
	registry := core.NewRegistry()
	if err := registry.AddDomain(testContractDomain(allowed)); err != nil {
		t.Fatal(err)
	}
	return registry
}

func testContractDomain(allowed []any) core.DomainDefinition {
	return core.DomainDefinition{
		Name: "Constitution", APIVersion: "constitution.example.org/v1alpha1", Path: "domains/constitution.yaml",
		Kinds:       map[string]core.KindDefinition{"Clause": {Required: []string{"category"}, Properties: map[string]core.PropertyDefinition{"category": {Type: "string"}}}},
		Constraints: []core.ConstraintDefinition{{Name: "requires-core", Description: "Clauses may target only declared Core resources.", Select: core.ResourceSelector{Kind: "Clause"}, Assert: core.ConstraintAssertion{Op: "allowed", Field: "category", Values: allowed}}},
	}
}

func intPointer(value int) *int { return &value }

func countUnescapedPipes(line string) int {
	count := 0
	for index := 0; index < len(line); index++ {
		if line[index] != '|' {
			continue
		}
		backslashes := 0
		for previous := index - 1; previous >= 0 && line[previous] == '\\'; previous-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			count++
		}
	}
	return count
}

func assertNoTrailingWhitespace(t *testing.T, name, body string) {
	t.Helper()
	for lineNumber, line := range strings.Split(body, "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("%s:%d has trailing whitespace: %q", name, lineNumber+1, line)
		}
	}
}
