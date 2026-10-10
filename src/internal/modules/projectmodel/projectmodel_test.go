package projectmodel

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
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

// BUG-01: a Check that exercises a changed Statement must run again, and its
// owner is routed, even when no Artifact declares that Check.
func TestImpactRoutesChecksThatUseAChangedStatement(t *testing.T) {
	model, files := fixture(t, false, false, true)
	root := map[string]any{"apiVersion": APIVersion, "kind": managerKind, "namespace": "", "name": "root"}
	contract := map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "release-reservation"}
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Namespace: "audit", Name: "audit"}, Purpose: "Audits stock.", Spec: map[string]any{"parent": root, "owns": []any{"src/audit/"}}},
		core.Definition{APIVersion: APIVersion, Kind: checkKind, Metadata: core.Metadata{Namespace: "audit", Name: "stock-audit"}, Purpose: "Audit released stock.", Spec: map[string]any{"command": []any{"go", "test", "./audit"}, "uses": []any{contract}}},
	)
	analyze := func(description string) Report {
		for i := range definitions {
			if definitions[i].Metadata.Name == "release-reservation" {
				definitions[i].Spec["description"] = description
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, copyDefinitions(definitions), "check-uses")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	base := analyze("Release reservation once.")
	impact := Impact(base, analyze("Release reservation at most once."))
	auditCheck := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: checkKind, Namespace: "audit", Name: "stock-audit"}).Key()
	auditManager := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "audit", Name: "audit"}).Key()
	if len(impact.Unknown) != 0 || !contains(impact.Checks, auditCheck) || !contains(impact.Managers, auditManager) {
		t.Fatalf("check using the changed contract was not routed: unknown=%v checks=%v managers=%v", impact.Unknown, impact.Checks, impact.Managers)
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

// BUG-01: an absent optional Artifact path leaves an expected-artifact entry.
// That entry is not inventory and must not satisfy a required Artifact that
// expects the same path or a prefix covering it, whichever Artifact sorts first.
func TestAnalyzeReportsRequiredArtifactBehindAbsentOptionalPath(t *testing.T) {
	model, files := fixture(t, false, false, false)
	artifact := func(name string, required bool, path string) core.Definition {
		return core.Definition{APIVersion: APIVersion, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: name}, Purpose: "Expected orders artifact.", Spec: map[string]any{"role": "implementation", "paths": []any{path}, "required": required}}
	}
	for _, tc := range []struct{ name, optional, required, optionalPath, requiredSelector string }{
		{"same path, optional first", "a-optional", "b-required", "src/orders/missing.go", "src/orders/missing.go"},
		{"same path, required first", "b-optional", "a-required", "src/orders/missing.go", "src/orders/missing.go"},
		{"covering prefix, optional first", "a-optional", "b-required", "src/orders/gen/x.go", "src/orders/gen/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definitions := append(copyDefinitions(model.Definitions), artifact(tc.optional, false, tc.optionalPath), artifact(tc.required, true, tc.requiredSelector))
			candidate, diagnostics := core.Compile(model.Schemas, definitions, "optional-and-required")
			if len(diagnostics) != 0 {
				t.Fatalf("compile: %+v", diagnostics)
			}
			r := Analyze(candidate, files)
			requiredID := core.DefinitionIdentity{APIVersion: APIVersion, Kind: artifactKind, Namespace: "orders", Name: tc.required}.Key()
			missing := false
			for _, f := range r.Findings {
				missing = missing || f.Code == "coverage.required-artifact-missing" && f.Subject == requiredID
			}
			if r.Status != "incomplete" || !missing {
				t.Fatalf("required %s is absent but not reported: status=%s findings=%+v", tc.requiredSelector, r.Status, r.Findings)
			}
		})
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

// BUG-01: "./" is not the whole-repository selector "." and matches no file,
// so it must be rejected rather than leave every file unowned.
func TestAnalyzeRejectsDotSlashSelectors(t *testing.T) {
	model, files := fixture(t, true, false, true)
	for i := range model.Definitions {
		if model.Definitions[i].Kind == managerKind && model.Definitions[i].Metadata.Namespace == "" {
			model.Definitions[i].Spec["owns"] = []any{"./"}
		}
		if model.Definitions[i].Kind == artifactKind && model.Definitions[i].Metadata.Namespace == "orders" {
			model.Definitions[i].Spec["paths"] = []any{"./"}
		}
	}
	// Core Model is immutable by contract; this deliberate corruption exercises fail-closed analysis.
	r := Analyze(model, files)
	if r.Status != "failed" || !hasFinding(r.Findings, "path.ownership-invalid") || !hasFinding(r.Findings, "path.artifact-invalid") {
		t.Fatalf("\"./\" selectors were accepted: status=%s unknown=%v findings=%+v", r.Status, r.Unknown, r.Findings)
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
