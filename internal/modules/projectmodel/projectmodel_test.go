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
	if s.APIVersion != APIVersion || len(s.Kinds) != 6 {
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

func TestAnalyzePreservesLiteralArgvOrderAndDuplicates(t *testing.T) {
	model, files := fixture(t, false, false, true)
	for i := range model.Definitions {
		if model.Definitions[i].Kind == checkKind {
			model.Definitions[i].Spec["command"] = []any{"python", "-m", "unittest", "x", "x"}
		}
	}
	r := Analyze(model, files)
	if len(r.Checks) != 1 || strings.Join(r.Checks[0].Command, "|") != "python|-m|unittest|x|x" {
		t.Fatalf("literal argv was normalized or reordered: %+v", r.Checks)
	}
}

func TestImpactChangedStatementIncludesUnchangedRealizationAndCheck(t *testing.T) {
	model, files := fixture(t, true, true, true)
	base := Analyze(model, files)
	definitions := copyDefinitions(model.Definitions)
	for i := range definitions {
		if definitions[i].Kind == statementKind && definitions[i].Metadata.Namespace == "orders" {
			definitions[i].Spec["description"] = "Cancel before shipment and release the reservation."
		}
	}
	candidate, diagnostics := core.Compile(model.Schemas, definitions, "changed-statement")
	if len(diagnostics) != 0 {
		t.Fatalf("compile changed statement: %+v", diagnostics)
	}
	impact := Impact(base, Analyze(candidate, files))
	if !contains(impact.Files, "src/orders/cancel.go") || len(impact.Checks) == 0 {
		t.Fatalf("unchanged realization/check was omitted: files=%v checks=%v", impact.Files, impact.Checks)
	}
}

func TestImpactDecisionOnlyDeltaRoutesFullDeclaredReview(t *testing.T) {
	model, files := fixture(t, true, true, true)
	r := Analyze(model, files)
	candidate := r
	candidate.ModelDigest = "sha256:decision-only-change"
	impact := Impact(r, candidate)
	if len(impact.Managers) != len(r.Managers) || len(impact.Files) != len(r.Files) || len(impact.Checks) != len(r.Checks) || len(impact.Unknown) == 0 {
		t.Fatalf("unprojected model change did not route broad review: managers=%v files=%v checks=%v unknown=%v", impact.Managers, impact.Files, impact.Checks, impact.Unknown)
	}
}

func TestManagerPurposeIsProjectedAndRoutesScopedImpact(t *testing.T) {
	model, files := fixture(t, true, true, true)
	base := Analyze(model, files)
	definitions := copyDefinitions(model.Definitions)
	for i := range definitions {
		if definitions[i].Kind == managerKind && definitions[i].Metadata.Namespace == "orders" {
			definitions[i].Purpose = "Orders owns fulfilment and cancellation behavior."
		}
	}
	candidate, diagnostics := core.Compile(model.Schemas, definitions, "manager-purpose-change")
	if len(diagnostics) != 0 {
		t.Fatalf("compile manager purpose change: %+v", diagnostics)
	}
	impact := Impact(base, Analyze(candidate, files))
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	if !contains(impact.ChangedDefinitions, ordersID) || !contains(impact.Managers, ordersID) || !contains(impact.Managers, rootManagerKey()) {
		t.Fatalf("Manager purpose change did not route its owner and ancestor: changed=%v managers=%v", impact.ChangedDefinitions, impact.Managers)
	}
	if len(impact.Unknown) != 0 {
		t.Fatalf("Manager purpose change incorrectly fell back to unknown full scope: managers=%v files=%v unknown=%v", impact.Managers, impact.Files, impact.Unknown)
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
	removedRequires := Impact(Analyze(baseModel, files), base)
	if !contains(removedRequires.AffectedStatements, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()) || !contains(removedRequires.Files, "src/inventory/release.go") {
		t.Fatalf("removed requires edge lost its former contract or coverage: statements=%v files=%v", removedRequires.AffectedStatements, removedRequires.Files)
	}
}

func TestImpactTracksNamespaceAndSourcePathMovement(t *testing.T) {
	model, inventory := fixture(t, true, true, true)
	oldID := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"}
	newID := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "cancel-order"}
	baseDefinitions := copyDefinitions(model.Definitions)
	for i := range baseDefinitions {
		if baseDefinitions[i].Identity().Key() == oldID.Key() {
			baseDefinitions[i].Source.Path = ".markitect/model/orders/cancel-order.yaml"
		}
	}
	baseModel, diagnostics := core.Compile(model.Schemas, baseDefinitions, "before-move")
	if len(diagnostics) != 0 {
		t.Fatalf("compile base namespace: %+v", diagnostics)
	}
	inventory = append(inventory, File{Path: ".markitect/model/orders/cancel-order.yaml", Digest: "sha256:model-before", Mode: "100644"})
	base := Analyze(baseModel, inventory)

	candidateDefinitions := copyDefinitions(baseModel.Definitions)
	for i := range candidateDefinitions {
		if candidateDefinitions[i].Identity().Key() == oldID.Key() {
			candidateDefinitions[i].Metadata.Namespace = "inventory"
			candidateDefinitions[i].Source.Path = ".markitect/model/inventory/cancel-order.yaml"
		}
		for key, value := range candidateDefinitions[i].Spec {
			candidateDefinitions[i].Spec[key] = rewriteReference(value, oldID, newID)
		}
	}
	candidateModel, diagnostics := core.Compile(baseModel.Schemas, candidateDefinitions, "after-move")
	if len(diagnostics) != 0 {
		t.Fatalf("compile moved namespace: %+v", diagnostics)
	}
	candidateFiles := append([]File(nil), inventory...)
	for i := range candidateFiles {
		if candidateFiles[i].Path == ".markitect/model/orders/cancel-order.yaml" {
			candidateFiles[i].Path = ".markitect/model/inventory/cancel-order.yaml"
			candidateFiles[i].Digest = "sha256:model-after"
		}
	}
	impact := Impact(base, Analyze(candidateModel, candidateFiles))
	if !contains(impact.ChangedDefinitions, oldID.Key()) || !contains(impact.ChangedDefinitions, newID.Key()) {
		t.Fatalf("namespace identity migration was hidden: %v", impact.ChangedDefinitions)
	}
	if !contains(impact.Files, ".markitect/model/orders/cancel-order.yaml") || !contains(impact.Files, ".markitect/model/inventory/cancel-order.yaml") {
		t.Fatalf("source path movement was hidden: %v", impact.Files)
	}
	ordersID := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}.Key()
	inventoryID := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"}.Key()
	if !contains(impact.Managers, ordersID) || !contains(impact.Managers, inventoryID) {
		t.Fatalf("ownership routing missed old or new manager: %v", impact.Managers)
	}
}

func TestImpactFollowsTransitiveConsumers(t *testing.T) {
	model, files := fixture(t, true, true, true)
	definitions := copyDefinitions(model.Definitions)
	definitions = append(definitions, core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "operator-workflow"}, Purpose: "Operator flow uses cancellation.", Spec: map[string]any{"category": "workflow", "description": "Follow the cancellation workflow.", "uses": []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "orders", "name": "cancel-order"}}}})
	model, diagnostics := core.Compile(model.Schemas, definitions, "transitive-consumer")
	if len(diagnostics) != 0 {
		t.Fatalf("compile consumer fixture: %+v", diagnostics)
	}
	base := Analyze(model, files)
	changedDefinitions := copyDefinitions(model.Definitions)
	for i := range changedDefinitions {
		if changedDefinitions[i].Kind == statementKind && changedDefinitions[i].Metadata.Namespace == "inventory" {
			changedDefinitions[i].Spec["description"] = "Release once and report conflicts."
		}
	}
	candidate, diagnostics := core.Compile(model.Schemas, changedDefinitions, "changed-contract")
	if len(diagnostics) != 0 {
		t.Fatalf("compile changed contract: %+v", diagnostics)
	}
	impact := Impact(base, Analyze(candidate, files))
	if !contains(impact.AffectedStatements, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"}).Key()) || !contains(impact.AffectedStatements, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "operator-workflow"}).Key()) {
		t.Fatalf("transitive consumers were omitted: %v", impact.AffectedStatements)
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

func TestAnalyzeRejectsArtifactRealizingPrivateForeignStatement(t *testing.T) {
	model, files := fixture(t, false, false, true)
	definitions := copyDefinitions(model.Definitions)
	definitions = append(definitions, core.Definition{APIVersion: APIVersion, Kind: artifactKind, Metadata: core.Metadata{Namespace: "inventory", Name: "foreign-realization"}, Purpose: "Invalid cross-manager realization.", Spec: map[string]any{"role": "implementation", "realizes": []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "orders", "name": "cancel-order"}}, "paths": []any{"src/inventory/foreign.go"}}})
	model, diagnostics := core.Compile(model.Schemas, definitions, "private-artifact")
	if len(diagnostics) != 0 {
		t.Fatalf("compile private artifact fixture: %+v", diagnostics)
	}
	r := Analyze(model, files)
	if r.Status != "failed" || !hasFinding(r.Findings, "reference.private-cross-manager") {
		t.Fatalf("private artifact realization was not rejected: %+v", r.Findings)
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
	definitions := copyDefinitions(model.Definitions)
	definitions = append(definitions, core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "internal-guard"}, Purpose: "Private implementation detail.", Spec: map[string]any{"category": "rule", "description": "Internal inventory guard."}})
	for i := range definitions {
		if definitions[i].Kind == statementKind && definitions[i].Metadata.Namespace == "inventory" && definitions[i].Metadata.Name == "release-reservation" {
			definitions[i].Spec["uses"] = []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "internal-guard"}}
		}
	}
	model, diagnostics := core.Compile(model.Schemas, definitions, "private-contract-relation")
	if len(diagnostics) != 0 {
		t.Fatalf("compile contract relation fixture: %+v", diagnostics)
	}
	r := Analyze(model, files)
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	ctx, err := Context(r, ordersID)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Manager.Purpose != "Orders manager." {
		t.Fatalf("manager's own purpose is missing from its context: %+v", ctx.Manager)
	}
	if len(ctx.Statements) != 1 || ctx.Statements[0].Namespace != "orders" {
		t.Fatalf("manager received unrelated statements: %+v", ctx.Statements)
	}
	if len(ctx.Contracts) != 1 || !ctx.Contracts[0].Public || ctx.Contracts[0].Namespace != "inventory" {
		t.Fatalf("direct public contract missing: %+v", ctx.Contracts)
	}
	if len(ctx.Contracts[0].Uses) != 0 || len(ctx.Contracts[0].Requires) != 0 {
		t.Fatalf("public contract leaked private relation identities: %+v", ctx.Contracts[0])
	}
	for _, a := range ctx.Artifacts {
		if a.Owner == "" || strings.Contains(a.ID, "release-code") {
			t.Fatalf("sibling artifact leaked into context: %+v", a)
		}
	}
	if len(ctx.Children) != 0 {
		t.Fatalf("unexpected children: %+v", ctx.Children)
	}
	rootID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Name: "root"}).Key()
	rootContext, err := Context(r, rootID)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range rootContext.Children {
		if child.Purpose == "" {
			t.Fatalf("child purpose is missing from parent context: %+v", child)
		}
		if child.Instructions != "" {
			t.Fatalf("child-local instructions leaked to parent context: %+v", child)
		}
	}
	if _, err = Context(r, "missing"); err != ErrManagerNotFound {
		t.Fatalf("missing manager error=%v", err)
	}
}

func rootManagerKey() string {
	return (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Name: "root"}).Key()
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

func rewriteReference(value any, old, next core.DefinitionIdentity) any {
	switch v := value.(type) {
	case map[string]any:
		if v["apiVersion"] == old.APIVersion && v["kind"] == old.Kind && v["namespace"] == old.Namespace && v["name"] == old.Name {
			return map[string]any{"apiVersion": next.APIVersion, "kind": next.Kind, "namespace": next.Namespace, "name": next.Name}
		}
		for key, item := range v {
			v[key] = rewriteReference(item, old, next)
		}
	case []any:
		for i := range v {
			v[i] = rewriteReference(v[i], old, next)
		}
	}
	return value
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

func TestDecisionAndIdentityChangeAreProjectedAndRouteConcreteImpact(t *testing.T) {
	model, files := fixture(t, true, true, true)
	base := Analyze(model, files)
	defs := copyDefinitions(model.Definitions)
	order := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"}
	actor := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}
	oldDecision := core.DefinitionIdentity{APIVersion: APIVersion, Kind: decisionKind, Namespace: "orders", Name: "accept-cancellation-v1"}
	newDecision := core.DefinitionIdentity{APIVersion: APIVersion, Kind: decisionKind, Namespace: "orders", Name: "accept-cancellation-v2"}
	decision := func(i core.DefinitionIdentity, spec map[string]any) core.Definition {
		return core.Definition{APIVersion: APIVersion, Kind: decisionKind, Metadata: core.Metadata{Namespace: i.Namespace, Name: i.Name}, Purpose: "Record cancellation policy decision.", Spec: spec, Source: core.Source{Path: ".markitect/decisions.yaml", Line: 17}}
	}
	defs = append(defs,
		decision(oldDecision, map[string]any{"subject": refValue(order), "decision": "Accept cancellation before shipment.", "reason": "Preserve inventory consistency.", "actor": refValue(actor)}),
		decision(newDecision, map[string]any{"subject": refValue(order), "decision": "Accept cancellation before shipment.", "reason": "Clarify the existing rule.", "actor": refValue(actor), "supersedes": refValue(oldDecision), "public": true}),
		core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "cancel-order-rename"}, Purpose: "Record an explicit historical identity claim.", Spec: map[string]any{"operation": "renamed", "previous": historicalIdentity(APIVersion, statementKind, "orders", "cancel-order-v0"), "subject": refValue(order), "reason": "The Statement was renamed during model cleanup.", "actorManager": refValue(actor)}, Source: core.Source{Path: ".markitect/decisions.yaml", Line: 17}})
	candidateModel, diagnostics := core.Compile(model.Schemas, defs, "decision-and-identity-change")
	if len(diagnostics) != 0 {
		t.Fatalf("compile additive records: %+v", diagnostics)
	}
	candidate := Analyze(candidateModel, files)
	if candidate.Status != "succeeded" || len(candidate.Decisions) != 2 || len(candidate.IdentityChanges) != 1 {
		t.Fatalf("records were not projected cleanly: status=%s decisions=%+v changes=%+v findings=%+v", candidate.Status, candidate.Decisions, candidate.IdentityChanges, candidate.Findings)
	}
	if candidate.Decisions[0].Public || !candidate.Decisions[1].Public || candidate.Decisions[1].Supersedes != oldDecision.Key() || candidate.Decisions[1].Source.Path != ".markitect/decisions.yaml" {
		t.Fatalf("decision defaults, edge or provenance lost: %+v", candidate.Decisions)
	}
	change := candidate.IdentityChanges[0]
	if change.Previous.Name != "cancel-order-v0" || change.Subject != order.Key() || change.Actor != actor.Key() || change.Source.Line != 17 {
		t.Fatalf("identity change fields or provenance lost: %+v", change)
	}
	impact := Impact(base, candidate)
	if len(impact.Unknown) != 0 || !contains(impact.ChangedDefinitions, newDecision.Key()) || !contains(impact.ChangedDefinitions, change.ID) || !contains(impact.AffectedStatements, order.Key()) || !contains(impact.Files, "src/orders/cancel.go") || !contains(impact.Managers, actor.Key()) {
		t.Fatalf("real record delta did not route concrete scope: %+v", impact)
	}
	ctx, err := Context(candidate, actor.Key())
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Decisions) != 2 || len(ctx.IdentityChanges) != 1 {
		t.Fatalf("Manager context omitted owned records: decisions=%+v changes=%+v", ctx.Decisions, ctx.IdentityChanges)
	}
	removed := candidate
	removed.Decisions = append([]Decision(nil), candidate.Decisions...)
	for i, d := range removed.Decisions {
		if d.ID == oldDecision.Key() {
			removed.Decisions = append(removed.Decisions[:i], removed.Decisions[i+1:]...)
			break
		}
	}
	removed.IdentityChanges = nil
	removed.ModelDigest = "sha256:removed-records"
	removedImpact := Impact(candidate, removed)
	oldIdentity := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order-v0"}
	if len(removedImpact.Unknown) != 0 || !contains(removedImpact.ChangedDefinitions, oldDecision.Key()) || !contains(removedImpact.ChangedDefinitions, change.ID) || !contains(removedImpact.AffectedStatements, oldIdentity.Key()) || !contains(removedImpact.AffectedStatements, order.Key()) || !contains(removedImpact.Files, "src/orders/cancel.go") {
		t.Fatalf("removed decision/history obligations were omitted: %+v", removedImpact)
	}
}

func TestManagerContextFiltersPrivateForeignRecordsAndReferences(t *testing.T) {
	model, files := fixture(t, false, false, false)
	r := Analyze(model, files)
	publicSubject := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}.Key()
	privateDecision := core.DefinitionIdentity{APIVersion: APIVersion, Kind: decisionKind, Namespace: "inventory", Name: "private-record"}.Key()
	r.Decisions = append(r.Decisions,
		Decision{ID: privateDecision, Owner: "foreign-manager", Subject: publicSubject, Public: false},
		Decision{ID: "foreign-public", Owner: "foreign-manager", Subject: publicSubject, Supersedes: privateDecision, Public: true},
		Decision{ID: "own", Owner: (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key(), Subject: (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"}).Key()},
		Decision{ID: "own-hidden-subject", Owner: (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key(), Subject: "private-subject"},
	)
	r.IdentityChanges = append(r.IdentityChanges,
		IdentityChange{ID: "foreign-private-change", Owner: "foreign-manager", Subject: publicSubject, Public: false},
		IdentityChange{ID: "foreign-public-change", Owner: "foreign-manager", Subject: publicSubject, Public: true},
	)
	ordersOwner := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	ctx, err := Context(r, ordersOwner)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Decisions) != 3 || !contains([]string{ctx.Decisions[0].ID, ctx.Decisions[1].ID, ctx.Decisions[2].ID}, "foreign-public") || !contains([]string{ctx.Decisions[0].ID, ctx.Decisions[1].ID, ctx.Decisions[2].ID}, "own") {
		t.Fatalf("Context leaked or omitted a foreign Decision: %+v", ctx.Decisions)
	}
	for _, d := range ctx.Decisions {
		if d.ID == "own-hidden-subject" && d.Subject != "" {
			t.Fatalf("Context exposed an unavailable Decision subject: %+v", d)
		}
		if d.ID == "foreign-public" && d.Supersedes != "" {
			t.Fatalf("Context exposed a private supersedes edge: %+v", d)
		}
	}
	if len(ctx.IdentityChanges) != 1 || ctx.IdentityChanges[0].ID != "foreign-public-change" {
		t.Fatalf("Context leaked or omitted a foreign IdentityChange: %+v", ctx.IdentityChanges)
	}
}

func TestImpactRoutesSubjectOwnerForRootOwnedRecordsAddedAndRemoved(t *testing.T) {
	model, files := fixture(t, true, true, true)
	base := Analyze(model, files)
	defs := copyDefinitions(model.Definitions)
	root := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Name: "root"}
	contract := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}
	decision := core.DefinitionIdentity{APIVersion: APIVersion, Kind: decisionKind, Name: "inventory-contract-decision"}
	change := core.DefinitionIdentity{APIVersion: APIVersion, Kind: identityChangeKind, Name: "inventory-contract-history"}
	defs = append(defs,
		core.Definition{APIVersion: APIVersion, Kind: decisionKind, Metadata: core.Metadata{Name: decision.Name}, Purpose: "Record the project-level contract decision.", Spec: map[string]any{"subject": refValue(contract), "decision": "Keep the reservation contract stable.", "reason": "Consumers rely on this behavior.", "actor": refValue(root)}},
		core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Name: change.Name}, Purpose: "Record the project-level identity transition.", Spec: map[string]any{"operation": "replaced", "previous": historicalIdentity(APIVersion, statementKind, "inventory", "reservation-v0"), "subject": refValue(contract), "reason": "The contract identity was clarified.", "actorManager": refValue(root)}})
	candidateModel, diagnostics := core.Compile(model.Schemas, defs, "root-owned-records")
	if len(diagnostics) != 0 {
		t.Fatalf("compile root-owned records: %+v", diagnostics)
	}
	candidate := Analyze(candidateModel, files)
	if candidate.Status != "succeeded" {
		t.Fatalf("root-owned public references should be valid: %+v", candidate.Findings)
	}
	assertRoutesSubjectOwner := func(name string, impact ChangeImpact) {
		t.Helper()
		inventory := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"}.Key()
		if len(impact.Unknown) != 0 || !contains(impact.Managers, root.Key()) || !contains(impact.Managers, inventory) || !contains(impact.AffectedStatements, contract.Key()) || !contains(impact.Files, "src/inventory/release.go") {
			t.Fatalf("%s did not route the referenced subject owner: managers=%v statements=%v files=%v unknown=%v", name, impact.Managers, impact.AffectedStatements, impact.Files, impact.Unknown)
		}
	}
	assertRoutesSubjectOwner("added root-owned records", Impact(base, candidate))
	assertRoutesSubjectOwner("removed root-owned records", Impact(candidate, base))
}

func TestIdentityChangeValidationReportsInvalidHistoricalIdentityScopeAndCycle(t *testing.T) {
	model, files := fixture(t, false, false, false)
	defs := copyDefinitions(model.Definitions)
	order := core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"}
	ordersManager := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}
	defs = append(defs, core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "bad-previous"}, Purpose: "Invalid historical identity test.", Spec: map[string]any{"operation": "renamed", "previous": historicalIdentity(APIVersion, "OtherKind", "orders", "old"), "subject": refValue(order), "reason": "test", "actorManager": refValue(ordersManager)}})
	bad, diagnostics := core.Compile(model.Schemas, defs, "invalid-previous")
	if len(diagnostics) != 0 {
		t.Fatalf("Core should accept structurally typed historical values: %+v", diagnostics)
	}
	r := Analyze(bad, files)
	if !finding(r, "identity-change.previous-invalid") {
		t.Fatalf("invalid previous identity was not diagnosed: %+v", r.Findings)
	}
	activeDefs := copyDefinitions(model.Definitions)
	activeDefs = append(activeDefs, core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "retired-active"}, Purpose: "Retirement requires a historical identity.", Spec: map[string]any{"operation": "retired", "previous": historicalIdentity(APIVersion, statementKind, "orders", "cancel-order"), "reason": "test", "actorManager": refValue(core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Name: "root"})}})
	activeModel, diagnostics := core.Compile(model.Schemas, activeDefs, "retired-active")
	if len(diagnostics) != 0 {
		t.Fatalf("compile active retirement: %+v", diagnostics)
	}
	activeReport := Analyze(activeModel, files)
	if !finding(activeReport, "identity-change.previous-still-active") {
		t.Fatalf("active retired identity was not diagnosed: %+v", activeReport.Findings)
	}
	if finding(activeReport, "identity-change.actor-out-of-scope") {
		t.Fatalf("ancestor Manager actor should be within scope: %+v", activeReport.Findings)
	}

	foreignDefs := copyDefinitions(model.Definitions)
	for i := range foreignDefs {
		if foreignDefs[i].Kind == statementKind && foreignDefs[i].Metadata.Namespace == "inventory" {
			foreignDefs[i].Spec["public"] = false
		}
	}
	foreignDefs = append(foreignDefs, core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "foreign-private-subject"}, Purpose: "Private foreign subject and out-of-scope actor test.", Spec: map[string]any{"operation": "replaced", "previous": historicalIdentity(APIVersion, statementKind, "orders", "old-cancellation"), "subject": refValue(core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}), "reason": "test", "actorManager": refValue(core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"})}})
	foreignModel, diagnostics := core.Compile(model.Schemas, foreignDefs, "private-foreign-subject")
	if len(diagnostics) != 0 {
		t.Fatalf("compile private foreign subject case: %+v", diagnostics)
	}
	foreignReport := Analyze(foreignModel, files)
	foreignChangeID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: identityChangeKind, Namespace: "orders", Name: "foreign-private-subject"}).Key()
	if !findingSubject(foreignReport, "reference.private-cross-manager", foreignChangeID) || !finding(foreignReport, "identity-change.actor-out-of-scope") {
		t.Fatalf("private foreign subject or sibling Manager actor scope was accepted: %+v", foreignReport.Findings)
	}

	missingSubjectDefs := copyDefinitions(model.Definitions)
	missingSubjectDefs = append(missingSubjectDefs, core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "rename-without-subject"}, Purpose: "A rename names its current target.", Spec: map[string]any{"operation": "renamed", "previous": historicalIdentity(APIVersion, statementKind, "orders", "old-name"), "reason": "test", "actorManager": refValue(core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"})}})
	missingSubject, diagnostics := core.Compile(model.Schemas, missingSubjectDefs, "missing-identity-subject")
	if len(diagnostics) != 0 {
		t.Fatalf("Core should permit semantic subject validation in the projection: %+v", diagnostics)
	}
	if r := Analyze(missingSubject, files); !finding(r, "identity-change.subject-required") {
		t.Fatalf("missing rename target was not diagnosed: %+v", r.Findings)
	}

	cycleDefs := copyDefinitions(model.Definitions)
	for _, name := range []string{"old-a", "old-b"} {
		cycleDefs = append(cycleDefs, core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: name}, Purpose: "Statement retained for replacement history.", Spec: map[string]any{"category": "concept", "description": "Historical replacement target."}})
	}
	for _, pair := range [][2]string{{"old-a", "old-b"}, {"old-b", "old-a"}} {
		cycleDefs = append(cycleDefs, core.Definition{APIVersion: APIVersion, Kind: identityChangeKind, Metadata: core.Metadata{Namespace: "orders", Name: "replace-" + pair[0]}, Purpose: "Create identity cycle.", Spec: map[string]any{"operation": "replaced", "previous": historicalIdentity(APIVersion, statementKind, "orders", pair[0]), "subject": refValue(core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: pair[1]}), "reason": "test", "actorManager": refValue(ordersManager)}})
	}
	cyclic, diagnostics := core.Compile(model.Schemas, cycleDefs, "identity-cycle")
	if len(diagnostics) != 0 {
		t.Fatalf("compile replacement cycle: %+v", diagnostics)
	}
	if r := Analyze(cyclic, files); !finding(r, "identity-change.cycle") {
		t.Fatalf("identity cycle was not diagnosed: %+v", r.Findings)
	}
}

func refValue(identity core.DefinitionIdentity) map[string]any {
	return map[string]any{"apiVersion": identity.APIVersion, "kind": identity.Kind, "namespace": identity.Namespace, "name": identity.Name}
}

func historicalIdentity(apiVersion, kind, namespace, name string) map[string]any {
	return map[string]any{"apiVersion": apiVersion, "kind": kind, "namespace": namespace, "name": name}
}

func finding(report Report, code string) bool {
	for _, value := range report.Findings {
		if value.Code == code {
			return true
		}
	}
	return false
}

func findingSubject(report Report, code, subject string) bool {
	for _, value := range report.Findings {
		if value.Code == code && value.Subject == subject {
			return true
		}
	}
	return false
}
