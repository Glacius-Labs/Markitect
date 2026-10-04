package authoring

import "testing"

func packageTestInputs(content ...*Resource) ([]*Resource, *Resource) {
	project := project(Area{Name: "app", Path: "docs/app"}, Area{Name: "other", Path: "docs/other"})
	project.Spec.Packages = []PackagePin{{Name: "shared", Version: "1.2.3", Source: "git:example/repo@0123456789abcdef", Archive: "packages/shared.zip", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	manifest := &Resource{
		APIVersion: APIVersion, Kind: "Package", Metadata: Metadata{Name: "shared"},
		Path: "markitect-package.yaml", Package: "shared",
		Spec: Spec{Version: "1.2.3", Areas: []Area{{Name: "shared", Path: "content/shared"}}},
	}
	resources := []*Resource{project, manifest}
	for _, resource := range content {
		resource.Package = "shared"
		resources = append(resources, resource)
	}
	return resources, manifest
}

func packageText(name, file string) *Resource {
	r := resource("Text", "shared", name, file)
	r.Spec.Text = "Package content."
	return r
}

func hasDiagnosticAt(g *Graph, code, path string) bool {
	for _, diagnostic := range g.Diagnostics {
		if diagnostic.Code == code && diagnostic.Path == path {
			return true
		}
	}
	return false
}

func TestPackageOriginsPreserveLocalIdentityAndRequireExports(t *testing.T) {
	packaged := packageText("policy", "content/shared/policy.yaml")
	manifestResources, manifest := packageTestInputs(packaged)
	manifestResources[0].Spec.Areas[0].Name = "shared"
	manifest.Spec.Exports = []Ref{{Kind: "Text", Namespace: "shared", Name: "policy"}}
	local := resource("Text", "shared", "policy", "docs/app/policy.yaml")
	local.Spec.Text = "Local content."
	consumer := resource("Skill", "shared", "review", "docs/app/review.yaml")
	consumer.Spec.Uses = []Ref{{Kind: "Text", Package: "shared", Namespace: "shared", Name: "policy"}}
	manifestResources = append(manifestResources, local, consumer)
	graph := Build(manifestResources)
	if len(graph.Diagnostics) != 0 {
		t.Fatalf("valid package graph produced diagnostics: %#v", graph.Diagnostics)
	}
	if packaged.Key() != local.Key() {
		t.Fatal("fixture does not contain equal package-local identities")
	}
	if packaged.GraphKey() == local.GraphKey() || graph.Resources[packaged.GraphKey()] != packaged || graph.Resources[local.GraphKey()] != local {
		t.Fatalf("origin-qualified identities collided: %#v", graph.Resources)
	}
	if graph.Packages["shared"] != manifest {
		t.Fatalf("manifest is not available by package identity: %#v", graph.Packages)
	}
	if !graph.IsExported(packaged) || graph.IsExported(local) {
		t.Fatal("export membership did not follow package origin")
	}
	if !hasEdge(graph, consumer.GraphKey(), packaged.GraphKey()) {
		t.Fatalf("qualified package dependency edge missing: %#v", graph.Edges)
	}
}

func TestPackagePrivateAndTransitiveReferencesAreRejected(t *testing.T) {
	private := packageText("private", "content/shared/private.yaml")
	resourceWithTransitiveRef := resource("Workflow", "shared", "transitive", "content/shared/transitive.yaml")
	resourceWithTransitiveRef.Spec.Uses = []Ref{{Kind: "Text", Package: "shared", Namespace: "shared", Name: "private"}}
	resources, manifest := packageTestInputs(private, resourceWithTransitiveRef)
	manifest.Spec.Exports = []Ref{{Kind: "Text", Namespace: "shared", Name: "private-public-alias"}}
	consumer := resource("Skill", "app", "consumer", "docs/app/consumer.yaml")
	consumer.Spec.Uses = []Ref{{Kind: "Text", Package: "shared", Namespace: "shared", Name: "private"}}
	resources = append(resources, consumer)
	graph := Build(resources)
	if !hasCode(graph, "package.transitive-reference") {
		t.Fatalf("transitive package reference was accepted: %#v", graph.Diagnostics)
	}
	if !hasCode(graph, "package.reference-private") {
		t.Fatalf("consumer reference to an unexported resource was accepted: %#v", graph.Diagnostics)
	}
}

func TestImportedRuleCheckRequiresSelectionAndStaysWithinItsDeclaredScope(t *testing.T) {
	rule := resource("Rule", "shared", "entrypoints", "content/shared/entrypoints.yaml")
	rule.Spec.Text = "Check workflow entrypoints."
	rule.Spec.Check = "workflow-has-entrypoint"
	inside := resource("Workflow", "app", "selected-area", "docs/app/selected-area.yaml")
	outside := resource("Workflow", "other", "unrelated-area", "docs/other/unrelated-area.yaml")
	resources, manifest := packageTestInputs(rule)
	manifest.Spec.Exports = []Ref{{Kind: "Rule", Namespace: "shared", Name: "entrypoints"}}
	resources = append(resources, inside, outside)
	withoutSelection := Build(resources)
	if hasCode(withoutSelection, "workflow.entrypoint") {
		t.Fatalf("importing an unselected Rule activated its check: %#v", withoutSelection.Diagnostics)
	}
	selected := resource("Workflow", "app", "selects-rule", "docs/app/selects-rule.yaml")
	selected.Spec.Rules = []Ref{{Kind: "Rule", Package: "shared", Namespace: "shared", Name: "entrypoints"}}
	resources = append(resources, selected)
	withSelection := Build(resources)
	if !hasDiagnosticAt(withSelection, "workflow.entrypoint", selected.Path) {
		t.Fatalf("explicit resource-level imported Rule did not govern its declaring Workflow: %#v", withSelection.Diagnostics)
	}
	if hasDiagnosticAt(withSelection, "workflow.entrypoint", inside.Path) || hasDiagnosticAt(withSelection, "workflow.entrypoint", outside.Path) {
		t.Fatalf("resource-level rule selection escaped its declaring Workflow: %#v", withSelection.Diagnostics)
	}
	if !hasEdge(withSelection, selected.GraphKey(), rule.GraphKey()) {
		t.Fatalf("explicit imported Rule relationship is missing: %#v", withSelection.Edges)
	}
}

func TestLocalRuleCheckDoesNotImposeLocalPolicyOnImportedWorkflows(t *testing.T) {
	rule := resource("Rule", "app", "entrypoints", "docs/app/entrypoints.yaml")
	rule.Spec.Check = "workflow-has-entrypoint"
	localWorkflow := resource("Workflow", "app", "unused-local", "docs/app/unused-local.yaml")
	packageWorkflow := resource("Workflow", "shared", "unused-package", "content/shared/unused-package.yaml")
	resources, manifest := packageTestInputs(packageWorkflow)
	manifest.Spec.Exports = []Ref{{Kind: "Workflow", Namespace: "shared", Name: packageWorkflow.Metadata.Name}}
	resources = append(resources, rule, localWorkflow)
	graph := Build(resources)
	if !hasDiagnosticAt(graph, "workflow.entrypoint", localWorkflow.Path) {
		t.Fatalf("local rule stopped governing local workflows: %#v", graph.Diagnostics)
	}
	if hasDiagnosticAt(graph, "workflow.entrypoint", packageWorkflow.Path) {
		t.Fatalf("local rule imposed local policy on imported workflow: %#v", graph.Diagnostics)
	}
}

func TestImportedAreaRuleGovernsNestedAreas(t *testing.T) {
	rule := resource("Rule", "shared", "entrypoints", "content/shared/entrypoints.yaml")
	rule.Spec.Check = "workflow-has-entrypoint"
	workflow := resource("Workflow", "app", "nested", "docs/app/nested.yaml")
	resources, manifest := packageTestInputs(rule)
	manifest.Spec.Exports = []Ref{{Kind: "Rule", Namespace: "shared", Name: "entrypoints"}}
	resources[0].Spec.Areas = append(resources[0].Spec.Areas, Area{Name: "parent", Path: "docs", Rules: []Ref{{Kind: "Rule", Package: "shared", Namespace: "shared", Name: "entrypoints"}}})
	graph := Build(append(resources, workflow))
	if !hasDiagnosticAt(graph, "workflow.entrypoint", workflow.Path) {
		t.Fatalf("inherited package rule did not govern the nested area: %#v", graph.Diagnostics)
	}
}

func TestPackageDiagnosticsIdentifyTheirOrigin(t *testing.T) {
	broken := resource("Workflow", "shared", "broken", "content/shared/broken.yaml")
	broken.Spec.Uses = []Ref{{Kind: "Text", Name: "missing"}}
	resources, _ := packageTestInputs(broken)
	graph := Build(resources)
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Path == broken.Path && diagnostic.Package == "shared" {
			return
		}
	}
	t.Fatalf("package diagnostic has no origin: %#v", graph.Diagnostics)
}

func TestImportedProviderNamesDoNotConflictWithLocalProviders(t *testing.T) {
	packageSkill := resource("Skill", "shared", "assistant", "content/shared/assistant.yaml")
	packageSkill.Spec.Text = "Reusable package skill."
	localSkill := resource("Skill", "app", "assistant", "docs/app/assistant.yaml")
	localSkill.Spec.Text = "Local skill."
	resources, manifest := packageTestInputs(packageSkill)
	manifest.Spec.Exports = []Ref{{Kind: "Skill", Namespace: "shared", Name: "assistant"}}
	resources = append(resources, localSkill)
	graph := Build(resources)
	if hasCode(graph, "provider.name") {
		t.Fatalf("imported provider name collided with a local provider: %#v", graph.Diagnostics)
	}
}

func TestConsumerCannotOverridePackageLocalBinding(t *testing.T) {
	contract := resource("Contract", "shared", "review", "content/shared/review.yaml")
	contract.Spec.Kind, contract.Spec.Input, contract.Spec.Output = "Skill", []string{"change"}, []string{"findings"}
	implementation := resource("Skill", "shared", "reviewer", "content/shared/reviewer.yaml")
	implementation.Spec.Text = "Package implementation."
	implementation.Spec.Input, implementation.Spec.Output = []string{"change"}, []string{"findings"}
	implementation.Spec.Implements = []Ref{{Kind: "Contract", Namespace: "shared", Name: "review"}}
	resources, manifest := packageTestInputs(contract, implementation)
	manifest.Spec.Exports = []Ref{{Kind: "Contract", Namespace: "shared", Name: "review"}, {Kind: "Skill", Namespace: "shared", Name: "reviewer"}}
	manifest.Spec.Bindings = []Binding{{
		Contract:       Ref{Kind: "Contract", Namespace: "shared", Name: "review"},
		Implementation: Ref{Kind: "Skill", Namespace: "shared", Name: "reviewer"},
	}}
	caller := resource("Workflow", "shared", "caller", "content/shared/caller.yaml")
	caller.Package = "shared"
	caller.Spec.Needs = []Ref{{Kind: "Contract", Name: "review"}}
	resources = append(resources, caller)
	valid := Build(resources)
	if len(valid.Diagnostics) != 0 || !hasEdge(valid, caller.GraphKey(), implementation.GraphKey()) {
		t.Fatalf("package-local binding failed: diagnostics=%#v edges=%#v", valid.Diagnostics, valid.Edges)
	}
	local := resource("Skill", "app", "local-reviewer", "docs/app/local-reviewer.yaml")
	local.Spec.Text = "Local override."
	local.Spec.Input, local.Spec.Output = []string{"change"}, []string{"findings"}
	local.Spec.Implements = []Ref{{Kind: "Contract", Package: "shared", Namespace: "shared", Name: "review"}}
	resources = append(resources, local)
	project := resources[0]
	project.Spec.Bindings = []Binding{{
		Contract:       Ref{Kind: "Contract", Package: "shared", Namespace: "shared", Name: "review"},
		Implementation: Ref{Kind: "Skill", Namespace: "app", Name: "local-reviewer"},
	}}
	graph := Build(resources)
	if !hasCode(graph, "binding.package-override") {
		t.Fatalf("consumer Project replaced a package-local binding: %#v", graph.Diagnostics)
	}
	manifest.Spec.Bindings = nil
	fromConsumer := Build(resources)
	if !hasCode(fromConsumer, "binding.package-scope") || hasEdge(fromConsumer, caller.GraphKey(), local.GraphKey()) {
		t.Fatalf("package requirement escaped into a consumer binding: diagnostics=%#v edges=%#v", fromConsumer.Diagnostics, fromConsumer.Edges)
	}
}

func TestPackageRuntimeCyclesUseOriginQualifiedGraphKeys(t *testing.T) {
	a := resource("Workflow", "shared", "a", "content/shared/a.yaml")
	b := resource("Workflow", "shared", "b", "content/shared/b.yaml")
	a.Spec.Uses = []Ref{{Kind: "Workflow", Name: "b"}}
	b.Spec.Uses = []Ref{{Kind: "Workflow", Name: "a"}}
	resources, manifest := packageTestInputs(a, b)
	manifest.Spec.Exports = []Ref{{Kind: "Workflow", Namespace: "shared", Name: "a"}}
	graph := Build(resources)
	if !hasCode(graph, "graph.cycle") {
		t.Fatalf("package-local runtime cycle was not found: %#v", graph.Diagnostics)
	}
	if !hasEdge(graph, a.GraphKey(), b.GraphKey()) || !hasEdge(graph, b.GraphKey(), a.GraphKey()) {
		t.Fatalf("package cycle edges are not origin-qualified: %#v", graph.Edges)
	}
}

func TestSelfSelectedBindingIsARuntimeCycle(t *testing.T) {
	for _, packaged := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "package"}[packaged], func(t *testing.T) {
			contract := resource("Contract", "shared", "assessment", "content/shared/assessment.yaml")
			contract.Spec.Kind = "Skill"
			implementation := resource("Skill", "shared", "assessor", "content/shared/assessor.yaml")
			ref := Ref{Kind: "Contract", Namespace: "shared", Name: "assessment"}
			implementation.Spec.Implements = []Ref{ref}
			implementation.Spec.Needs = []Ref{ref}
			binding := Binding{Contract: ref, Implementation: Ref{Kind: "Skill", Namespace: "shared", Name: "assessor"}}
			var resources []*Resource
			if packaged {
				var manifest *Resource
				resources, manifest = packageTestInputs(contract, implementation)
				manifest.Spec.Bindings = []Binding{binding}
			} else {
				p := project(Area{Name: "shared", Path: "content/shared"})
				p.Spec.Bindings = []Binding{binding}
				resources = []*Resource{p, contract, implementation}
			}
			graph := Build(resources)
			if !hasCode(graph, "graph.cycle") {
				t.Fatalf("self-selected implementation was accepted: %#v", graph.Diagnostics)
			}
		})
	}
}

func TestPackagePinsRequireExactVersionDigestAndSafeUniqueArchivePath(t *testing.T) {
	tests := []struct {
		name string
		edit func(*PackagePin)
		want string
	}{
		{name: "floating version", edit: func(p *PackagePin) { p.Version = ">=1.2.3" }, want: "package.version"},
		{name: "leading zero version", edit: func(p *PackagePin) { p.Version = "01.2.3" }, want: "package.version"},
		{name: "unsafe archive", edit: func(p *PackagePin) { p.Archive = "../shared.zip" }, want: "package.archive"},
		{name: "uppercase digest", edit: func(p *PackagePin) { p.SHA256 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" }, want: "package.digest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, manifest := project(Area{Name: "shared", Path: "content/shared"}), &Resource{
				APIVersion: APIVersion, Kind: "Package", Metadata: Metadata{Name: "shared"},
				Path: "markitect-package.yaml", Package: "shared", Spec: Spec{Version: "1.2.3", Areas: []Area{{Name: "shared", Path: "content/shared"}}},
			}
			project.Spec.Packages = []PackagePin{{Name: "shared", Version: "1.2.3", Source: "git:repo@commit", Archive: "packages/shared.zip", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
			test.edit(&project.Spec.Packages[0])
			graph := Build([]*Resource{project, manifest})
			if !hasCode(graph, test.want) {
				t.Fatalf("expected %s for invalid pin, got %#v", test.want, graph.Diagnostics)
			}
		})
	}
}
