package projectmodel

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func fixture(t *testing.T, includeArtifact bool, artifactRequired bool, includeFiles bool) (core.Model, []File) {
	t.Helper()
	api := APIVersion
	id := func(kind, namespace, name string) core.DefinitionIdentity {
		return core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: namespace, Name: name}
	}
	ref := func(i core.DefinitionIdentity) map[string]any {
		return map[string]any{"apiVersion": i.APIVersion, "kind": i.Kind, "namespace": i.Namespace, "name": i.Name}
	}
	rootID := id(managerKind, "", "root")
	ordersID := id(managerKind, "orders", "orders")
	inventoryID := id(managerKind, "inventory", "inventory")
	orderStatement := id(statementKind, "orders", "cancel-order")
	contract := id(statementKind, "inventory", "release-reservation")
	checkID := id(checkKind, "orders", "cancel-order-tests")
	artifactID := id(artifactKind, "orders", "cancel-order-code")
	sharedArtifactID := id(artifactKind, "inventory", "release-code")
	defs := []core.Definition{
		{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Name: rootID.Name}, Purpose: "Root project manager.", Spec: map[string]any{"owns": []any{"."}}},
		{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Namespace: "orders", Name: ordersID.Name}, Purpose: "Orders manager.", Spec: map[string]any{"parent": ref(rootID), "owns": []any{"src/orders/"}, "instructions": "Keep order state consistent."}},
		{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Namespace: "inventory", Name: inventoryID.Name}, Purpose: "Inventory manager.", Spec: map[string]any{"parent": ref(rootID), "owns": []any{"src/inventory/"}}},
		{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: orderStatement.Name}, Purpose: "Order cancellation.", Spec: map[string]any{"category": "use-case", "description": "Cancel before shipment.", "requires": []any{ref(contract)}}},
		{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: contract.Name}, Purpose: "Public inventory contract.", Spec: map[string]any{"category": "rule", "description": "Release reservation once.", "public": true}},
		{APIVersion: api, Kind: checkKind, Metadata: core.Metadata{Namespace: "orders", Name: checkID.Name}, Purpose: "Run cancellation tests.", Spec: map[string]any{"command": []any{"go", "test", "./orders"}, "uses": []any{ref(orderStatement)}, "limitation": "Does not prove production behavior."}},
	}
	if includeArtifact {
		defs = append(defs,
			core.Definition{APIVersion: api, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: artifactID.Name}, Purpose: "Cancellation implementation.", Spec: map[string]any{"role": "implementation", "realizes": []any{ref(orderStatement)}, "paths": []any{"src/orders/cancel.go"}, "checks": []any{ref(checkID)}, "required": artifactRequired}},
			core.Definition{APIVersion: api, Kind: artifactKind, Metadata: core.Metadata{Namespace: "inventory", Name: sharedArtifactID.Name}, Purpose: "Reservation implementation.", Spec: map[string]any{"role": "implementation", "realizes": []any{ref(contract)}, "paths": []any{"src/inventory/release.go"}, "required": true}},
		)
	}
	model, diagnostics := core.Compile([]core.Schema{Schema()}, defs, "fixture")
	if len(diagnostics) != 0 {
		t.Fatalf("compile fixture: %+v", diagnostics)
	}
	files := []File{{Path: ".markitect/project.yaml", Digest: "sha256:manifest", Mode: "100644"}}
	if includeFiles {
		files = append(files, File{Path: "src/orders/cancel.go", Digest: "sha256:cancel", Mode: "100644"}, File{Path: "src/inventory/release.go", Digest: "sha256:release", Mode: "100644"})
	}
	return model, files
}

func TestSchemaUsesTypedFullIdentityReferences(t *testing.T) {
	s := Schema()
	if s.APIVersion != APIVersion || len(s.Kinds) != 5 {
		t.Fatalf("unexpected schema: %#v", s)
	}
	parent := s.Kinds[managerKind].Properties["parent"]
	if parent.Type != core.TypeReference || parent.Target == nil || parent.Target.Kind != managerKind {
		t.Fatalf("parent is not a typed Manager reference: %#v", parent)
	}
	requires := s.Kinds[statementKind].Properties["requires"]
	if requires.Type != core.TypeReference || requires.Target == nil || requires.Target.Kind != statementKind {
		t.Fatalf("requires is not a typed Statement reference: %#v", requires)
	}
}

func TestAnalyzeTracksManyToManyFileMeaningAndDeterministicDigest(t *testing.T) {
	model, files := fixture(t, true, true, true)
	definitions := append([]core.Definition(nil), model.Definitions...)
	definitions = append(definitions,
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "cancellation-audit"}, Purpose: "Audit cancellation history.", Spec: map[string]any{"category": "rule", "description": "Record every accepted cancellation."}},
		core.Definition{APIVersion: APIVersion, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: "cancellation-audit-doc"}, Purpose: "Document the cancellation audit.", Spec: map[string]any{"role": "documentation", "realizes": []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "orders", "name": "cancellation-audit"}}, "paths": []any{"src/orders/cancel.go"}}},
	)
	var diagnostics []core.Diagnostic
	model, diagnostics = core.Compile(model.Schemas, definitions, "fixture-many-meanings")
	if len(diagnostics) != 0 {
		t.Fatalf("compile many-meaning fixture: %+v", diagnostics)
	}
	first := Analyze(model, files)
	second := Analyze(model, []File{files[2], files[0], files[1]})
	if first.Digest != second.Digest {
		t.Fatalf("input order changed report digest: %s != %s", first.Digest, second.Digest)
	}
	if first.Status != "succeeded" {
		t.Fatalf("unexpected status %q: %+v", first.Status, first.Findings)
	}
	var mapped FileEntry
	for _, f := range first.Files {
		if f.Path == "src/orders/cancel.go" {
			mapped = f
		}
	}
	if !mapped.Exists || len(mapped.Artifacts) != 2 || len(mapped.Statements) != 2 || len(mapped.Checks) != 1 {
		t.Fatalf("file index lost its model meanings: %+v", mapped)
	}
	if mapped.Owner == "" {
		t.Fatalf("file ownership was not resolved: %+v", mapped)
	}
}

func TestImpactUsesAddsContextWhileRequiresAddsCoverage(t *testing.T) {
	baseModel, files := fixture(t, true, true, true)
	baseDefs := copyDefinitions(baseModel.Definitions)
	for i := range baseDefs {
		if baseDefs[i].Kind == statementKind && baseDefs[i].Metadata.Namespace == "orders" {
			delete(baseDefs[i].Spec, "requires")
		}
	}
	noDependencyModel, diagnostics := core.Compile(baseModel.Schemas, baseDefs, "no-dependency")
	if len(diagnostics) != 0 {
		t.Fatalf("compile no-dependency model: %+v", diagnostics)
	}
	base := Analyze(noDependencyModel, files)
	usesDefs := copyDefinitions(noDependencyModel.Definitions)
	for i := range usesDefs {
		if usesDefs[i].Kind == statementKind && usesDefs[i].Metadata.Namespace == "orders" {
			usesDefs[i].Spec["uses"] = []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "release-reservation"}}
		}
	}
	usesModel, diagnostics := core.Compile(baseModel.Schemas, usesDefs, "uses-change")
	if len(diagnostics) != 0 {
		t.Fatalf("compile uses model: %+v", diagnostics)
	}
	usesImpact := Impact(base, Analyze(usesModel, files))
	if contains(usesImpact.Files, "src/inventory/release.go") {
		t.Fatalf("uses was mistaken for implementation coverage: %v", usesImpact.Files)
	}
	if len(usesImpact.AffectedStatements) < 2 {
		t.Fatalf("uses context omitted its target: %v", usesImpact.AffectedStatements)
	}
	requiresImpact := Impact(base, Analyze(baseModel, files))
	if !contains(requiresImpact.Files, "src/inventory/release.go") {
		t.Fatalf("requires did not include target artifact coverage: %v", requiresImpact.Files)
	}
}

func TestAnalyzeKeepsRequiredArtifactWhenMappingIsAbsent(t *testing.T) {
	model, files := fixture(t, true, true, false)
	r := Analyze(model, files)
	if r.Status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", r.Status)
	}
	if !hasFinding(r.Findings, "coverage.required-artifact-missing") {
		t.Fatalf("missing required artifact finding: %+v", r.Findings)
	}
	if !hasFinding(r.Findings, "coverage.required-artifact-missing") {
		t.Fatal("removing the file inventory must not remove its requirement")
	}
}

func TestAnalyzeRejectsSiblingOverlapAndUnsafePaths(t *testing.T) {
	model, files := fixture(t, false, false, false)
	for i := range model.Definitions {
		if model.Definitions[i].Kind == managerKind && model.Definitions[i].Metadata.Namespace == "inventory" {
			model.Definitions[i].Spec["owns"] = []any{"src/areas/orders/"}
		}
		if model.Definitions[i].Kind == managerKind && model.Definitions[i].Metadata.Namespace == "orders" {
			model.Definitions[i].Spec["owns"] = []any{"src/areas/", "../escape"}
		}
	}
	// Core Model is immutable by contract; this deliberate structural corruption exercises fail-closed analysis.
	r := Analyze(model, append(files, File{Path: "src\\bad.go"}))
	if r.Status != "failed" || !hasFinding(r.Findings, "ownership.sibling-overlap") || !hasFinding(r.Findings, "path.ownership-invalid") || !hasFinding(r.Findings, "path.inventory-invalid") {
		t.Fatalf("unsafe ownership was not rejected: status=%s findings=%+v", r.Status, r.Findings)
	}
}

func TestAnalyzeRejectsPrivateCrossManagerReference(t *testing.T) {
	model, files := fixture(t, false, false, true)
	for i := range model.Definitions {
		if model.Definitions[i].Kind == statementKind && model.Definitions[i].Metadata.Namespace == "inventory" {
			model.Definitions[i].Spec["public"] = false
		}
	}
	r := Analyze(model, files)
	if r.Status != "failed" || !hasFinding(r.Findings, "reference.private-cross-manager") {
		t.Fatalf("private cross-manager reference was not rejected: %+v", r.Findings)
	}
}

func TestImpactRetainsRemovedRequiredObligationAndRoutesOwners(t *testing.T) {
	oldModel, oldFiles := fixture(t, true, true, true)
	newModel, newFiles := fixture(t, false, false, false)
	oldReport, newReport := Analyze(oldModel, oldFiles), Analyze(newModel, newFiles)
	impact := Impact(oldReport, newReport)
	if len(impact.ChangedDefinitions) == 0 {
		t.Fatal("removing artifact definitions must be visible")
	}
	if !contains(impact.Files, "src/orders/cancel.go") || !contains(impact.Files, "src/inventory/release.go") {
		t.Fatalf("removed artifact paths were lost: %v", impact.Files)
	}
	if !contains(impact.AffectedStatements, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()) {
		t.Fatalf("removed required contract not affected: %v", impact.AffectedStatements)
	}
	if !contains(impact.Managers, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Name: "root"}).Key()) {
		t.Fatalf("manager ancestry missing: %v", impact.Managers)
	}
}

func TestImpactNewRequirementIncludesUnchangedExpectedPathAndCheck(t *testing.T) {
	oldModel, files := fixture(t, true, false, true)
	newModel, _ := fixture(t, true, true, true)
	impact := Impact(Analyze(oldModel, files), Analyze(newModel, files))
	if !contains(impact.Files, "src/orders/cancel.go") {
		t.Fatalf("newly required artifact path missing from impact: %v", impact.Files)
	}
	if len(impact.Checks) == 0 {
		t.Fatalf("new requirement did not route its declared check: %v", impact.Checks)
	}
}

func TestContextIncludesDirectPublicContractsAndExcludesSiblingInternals(t *testing.T) {
	model, files := fixture(t, true, true, true)
	r := Analyze(model, files)
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	ctx, err := Context(r, ordersID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Statements) != 1 || ctx.Statements[0].Namespace != "orders" {
		t.Fatalf("manager received unrelated statements: %+v", ctx.Statements)
	}
	if len(ctx.Contracts) != 1 || !ctx.Contracts[0].Public || ctx.Contracts[0].Namespace != "inventory" {
		t.Fatalf("direct public contract missing: %+v", ctx.Contracts)
	}
	for _, a := range ctx.Artifacts {
		if a.Owner == "" || strings.Contains(a.ID, "release-code") {
			t.Fatalf("sibling artifact leaked into context: %+v", a)
		}
	}
	if len(ctx.Children) != 0 {
		t.Fatalf("unexpected children: %+v", ctx.Children)
	}
	if _, err = Context(r, "missing"); err != ErrManagerNotFound {
		t.Fatalf("missing manager error=%v", err)
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func copyDefinitions(definitions []core.Definition) []core.Definition {
	copy := append([]core.Definition(nil), definitions...)
	for i := range copy {
		spec := map[string]any{}
		for key, value := range copy[i].Spec {
			spec[key] = value
		}
		copy[i].Spec = spec
	}
	return copy
}

func TestAnalyzeExplicitlyReportsUnknownInventory(t *testing.T) {
	model, files := fixture(t, false, false, false)
	for i := range model.Definitions {
		if model.Definitions[i].Kind == managerKind && model.Definitions[i].Metadata.Namespace == "" {
			model.Definitions[i].Spec["owns"] = []any{".markitect/", "src/orders/", "src/inventory/"}
		}
	}
	r := Analyze(model, append(files, File{Path: "loose.txt", Digest: "sha256:loose"}))
	if r.Status != "incomplete" || len(r.Unknown) != 1 || r.Unknown[0] != "loose.txt" {
		t.Fatalf("unknown inventory hidden: status=%s unknown=%v", r.Status, r.Unknown)
	}
}
