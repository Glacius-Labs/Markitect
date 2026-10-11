package projectmodel

import (
	"encoding/json"
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

// decisionFixture adds Decision definitions to the standard fixture.
func decisionFixture(t *testing.T, decisions ...core.Definition) (Report, core.Model, []File) {
	t.Helper()
	model, files := fixture(t, true, true, true)
	compiled, diagnostics := core.Compile(model.Schemas, append(copyDefinitions(model.Definitions), decisions...), "decisions")
	if len(diagnostics) != 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	return Analyze(compiled, files), compiled, files
}

func decisionDefinition(namespace, name, subjectNamespace, subject, text string) core.Definition {
	return core.Definition{APIVersion: APIVersion, Kind: decisionKind, Metadata: core.Metadata{Namespace: namespace, Name: name}, Purpose: "A recorded decision.", Spec: map[string]any{
		"subject":  map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": subjectNamespace, "name": subject},
		"decision": text,
		"reason":   "Stock must stay exact.",
		"actor":    map[string]any{"apiVersion": APIVersion, "kind": managerKind, "namespace": namespace, "name": namespace},
	}}
}

// DEC-022: Decisions are in the Report, and a report without them keeps its
// JSON and digest exactly as before Decisions were projected.
func TestAnalyzeProjectsDecisionsOnlyWhenDeclared(t *testing.T) {
	plain, _, _ := decisionFixture(t)
	data, err := json.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"decisions"`) {
		t.Fatalf("a report without Decisions gained a decisions field: %s", data)
	}
	before := digest(struct {
		API, Model, Inventory, Status string
		Managers                      []Manager
		Statements                    []Statement
		Artifacts                     []Artifact
		Checks                        []Check
		Files                         []FileEntry
		Findings                      []Finding
		Unknown                       []string
	}{plain.APIVersion, plain.ModelDigest, plain.InventoryDigest, plain.Status, plain.Managers, plain.Statements, plain.Artifacts, plain.Checks, plain.Files, plain.Findings, plain.Unknown})
	if plain.Digest != before {
		t.Fatalf("report digest without Decisions changed: %s, was %s", plain.Digest, before)
	}
	r, _, _ := decisionFixture(t, decisionDefinition("inventory", "release-once", "inventory", "release-reservation", "Release each reservation exactly once."))
	inventoryID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"}).Key()
	contractID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()
	if r.Status != "succeeded" || len(r.Decisions) != 1 || r.Decisions[0].Owner != inventoryID || r.Decisions[0].Subject != contractID || r.Decisions[0].Actor != inventoryID || r.Decisions[0].Decision != "Release each reservation exactly once." {
		t.Fatalf("decision not projected: status=%s decisions=%+v findings=%+v", r.Status, r.Decisions, r.Findings)
	}
}

// DEC-022: a Manager's Context shows its own Decisions and the Statements they
// decide on; another Manager's Decisions stay out of it.
func TestContextShowsOwnDecisionsAndTheirSubjects(t *testing.T) {
	r, _, _ := decisionFixture(t,
		decisionDefinition("inventory", "release-once", "inventory", "release-reservation", "Release each reservation exactly once."),
		decisionDefinition("orders", "cancel-releases", "inventory", "release-reservation", "Cancellation always releases."),
	)
	contractID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()
	for namespace, want := range map[string]string{"inventory": "release-once", "orders": "cancel-releases"} {
		ctx, err := Context(r, (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: namespace, Name: namespace}).Key())
		if err != nil {
			t.Fatal(err)
		}
		if len(ctx.Decisions) != 1 || ctx.Decisions[0].Name != want {
			t.Fatalf("%s sees decisions %+v, want only its own %s", namespace, ctx.Decisions, want)
		}
		if namespace == "orders" && (len(ctx.Contracts) != 1 || ctx.Contracts[0].ID != contractID) {
			t.Fatalf("orders does not see the contract its decision is about: %+v", ctx.Contracts)
		}
	}
}

// DEC-022: a Decision may not decide on another Manager's private Statement,
// as no other reference may; Context would otherwise reveal it.
func TestAnalyzeRejectsDecisionOnForeignPrivateStatement(t *testing.T) {
	r, _, _ := decisionFixture(t, decisionDefinition("inventory", "orders-cancel", "orders", "cancel-order", "Inventory decides on cancellation."))
	if r.Status != "failed" || !hasFinding(r.Findings, "reference.private-cross-manager") {
		t.Fatalf("decision on a foreign private statement was accepted: status=%s findings=%+v", r.Status, r.Findings)
	}
}

// DEC-022: a Decision change is a change of its subject, routed with the
// Decision's owner, and no longer widens to the whole project.
func TestImpactRoutesDecisionChangeThroughItsSubject(t *testing.T) {
	decision := decisionDefinition("orders", "cancel-releases", "inventory", "release-reservation", "Cancellation always releases.")
	without, _, _ := decisionFixture(t)
	base, compiled, files := decisionFixture(t, decision)
	changedDefinitions := copyDefinitions(compiled.Definitions)
	for i := range changedDefinitions {
		if changedDefinitions[i].Kind == decisionKind {
			changedDefinitions[i].Spec["decision"] = "Cancellation releases unless already released."
		}
	}
	changedModel, diagnostics := core.Compile(compiled.Schemas, changedDefinitions, "decision-changed")
	if len(diagnostics) != 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	decisionID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: decisionKind, Namespace: "orders", Name: "cancel-releases"}).Key()
	contractID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	inventoryID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"}).Key()
	for name, impact := range map[string]ChangeImpact{
		"decision text changed": Impact(base, Analyze(changedModel, files)),
		"decision added":        Impact(without, base),
		"decision removed":      Impact(base, without),
	} {
		if len(impact.Unknown) != 0 {
			t.Fatalf("%s widened to the whole project: %v", name, impact.Unknown)
		}
		if !contains(impact.ChangedDefinitions, decisionID) || !contains(impact.AffectedStatements, contractID) || !contains(impact.Managers, ordersID) || !contains(impact.Managers, inventoryID) || !contains(impact.Files, "src/inventory/release.go") || !hasFinding(impact.Findings, "impact.decision-change") {
			t.Fatalf("%s was not routed through its subject and owner: changed=%v statements=%v managers=%v files=%v", name, impact.ChangedDefinitions, impact.AffectedStatements, impact.Managers, impact.Files)
		}
	}
}

// DEC-022: like a change of the subject itself, a Decision change routes the
// subject's owner, even when nothing else reaches that Manager.
func TestImpactRoutesDecisionSubjectOwner(t *testing.T) {
	model, files := fixture(t, true, true, true)
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "stock-unit"}, Purpose: "Unit of stock.", Spec: map[string]any{"category": "concept", "description": "Pieces.", "public": true}},
		decisionDefinition("orders", "count-pieces", "inventory", "stock-unit", "Orders count pieces."),
	)
	analyze := func(text string) Report {
		for i := range definitions {
			if definitions[i].Kind == decisionKind {
				definitions[i].Spec["decision"] = text
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, copyDefinitions(definitions), "subject-owner")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	impact := Impact(analyze("Orders count pieces."), analyze("Orders count packs."))
	inventoryID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "inventory", Name: "inventory"}).Key()
	if len(impact.Unknown) != 0 || !contains(impact.Managers, inventoryID) {
		t.Fatalf("decision change did not route its subject's owner: managers=%v unknown=%v", impact.Managers, impact.Unknown)
	}
}

// sourcedFixture is the standard fixture with each definition in its own model file.
func sourcedFixture(t *testing.T) ([]core.Definition, core.Model, []File) {
	t.Helper()
	model, files := fixture(t, true, true, true)
	definitions := copyDefinitions(model.Definitions)
	for i := range definitions {
		definitions[i].Source.Path = ".markitect/model/" + definitions[i].Metadata.Namespace + "/" + definitions[i].Metadata.Name + ".yaml"
	}
	return definitions, model, files
}

func analyzeDefinitions(t *testing.T, model core.Model, definitions []core.Definition, files []File) Report {
	t.Helper()
	compiled, diagnostics := core.Compile(model.Schemas, definitions, "sourced")
	if len(diagnostics) != 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	return Analyze(compiled, files)
}

// DEC-021: a purpose change is a change of its definition and routes exactly
// like a description change of the same Statement.
func TestImpactRoutesPurposeChangeLikeDescriptionChange(t *testing.T) {
	definitions, model, files := sourcedFixture(t)
	base := analyzeDefinitions(t, model, definitions, files)
	edited := func(edit func(*core.Definition)) Report {
		changed := copyDefinitions(definitions)
		for i := range changed {
			if changed[i].Metadata.Name == "release-reservation" {
				edit(&changed[i])
			}
		}
		return analyzeDefinitions(t, model, changed, files)
	}
	purpose := Impact(base, edited(func(d *core.Definition) { d.Purpose = "Public inventory contract, now with expiry." }))
	description := Impact(base, edited(func(d *core.Definition) { d.Spec["description"] = "Release reservation at most once." }))
	if len(purpose.Unknown) != 0 {
		t.Fatalf("purpose change widened to the whole project: %v", purpose.Unknown)
	}
	for name, sets := range map[string][2][]string{
		"changed definitions": {purpose.ChangedDefinitions, description.ChangedDefinitions},
		"statements":          {purpose.AffectedStatements, description.AffectedStatements},
		"managers":            {purpose.Managers, description.Managers},
		"files":               {purpose.Files, description.Files},
		"checks":              {purpose.Checks, description.Checks},
	} {
		if strings.Join(sets[0], "|") != strings.Join(sets[1], "|") {
			t.Fatalf("%s: purpose change %v, description change %v", name, sets[0], sets[1])
		}
	}
}

// DEC-021: an edit that changes only how a definition is written routes its
// model file, the file's owner and the definitions in that file, and nothing
// that depends on their meaning.
func TestImpactRoutesMeaningFreeEditThroughItsFile(t *testing.T) {
	definitions, model, files := sourcedFixture(t)
	contract := map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "release-reservation"}
	for i := range definitions {
		if definitions[i].Kind == checkKind {
			definitions[i].Spec["uses"] = []any{contract, map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "orders", "name": "cancel-order"}}
		}
	}
	files = append(files, File{Path: ".markitect/model/orders/cancel-order-tests.yaml", Digest: "sha256:check", Mode: "100644"})
	base := analyzeDefinitions(t, model, definitions, files)
	reordered := copyDefinitions(definitions)
	for i := range reordered {
		if reordered[i].Kind == checkKind {
			uses := append([]any(nil), reordered[i].Spec["uses"].([]any)...)
			uses[0], uses[1] = uses[1], uses[0]
			reordered[i].Spec["uses"] = uses
		}
	}
	impact := Impact(base, analyzeDefinitions(t, model, reordered, files))
	checkID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: checkKind, Namespace: "orders", Name: "cancel-order-tests"}).Key()
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	if len(impact.Unknown) != 0 || len(impact.ChangedDefinitions) != 0 {
		t.Fatalf("meaning-free edit was treated as a semantic change: changed=%v unknown=%v", impact.ChangedDefinitions, impact.Unknown)
	}
	if !contains(impact.Files, ".markitect/model/orders/cancel-order-tests.yaml") || !contains(impact.Checks, checkID) || !contains(impact.Managers, ordersID) || !contains(impact.Managers, rootManagerKey()) {
		t.Fatalf("meaning-free edit was not routed through its file: files=%v checks=%v managers=%v", impact.Files, impact.Checks, impact.Managers)
	}
	if len(impact.AffectedStatements) != 0 || contains(impact.Files, "src/inventory/release.go") || contains(impact.Files, "src/orders/cancel.go") {
		t.Fatalf("meaning-free edit routed what depends on meaning: statements=%v files=%v", impact.AffectedStatements, impact.Files)
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

// BUG-01: the root namespace is shallower than a top-level one, so a selector
// the root delegates unchanged belongs to the delegate, as at every deeper level.
func TestAnalyzeGivesEqualSelectorToTheDelegateBelowRoot(t *testing.T) {
	ref := func(namespace, name string) map[string]any {
		return map[string]any{"apiVersion": APIVersion, "kind": managerKind, "namespace": namespace, "name": name}
	}
	root := core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Name: "root"}, Purpose: "Root manager.", Spec: map[string]any{"owns": []any{"src/", "docs/"}}}
	backend := core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Namespace: "backend", Name: "backend"}, Purpose: "Backend manager.", Spec: map[string]any{"parent": ref("", "root"), "owns": []any{"src/"}}}
	api := core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Namespace: "backend.api", Name: "api"}, Purpose: "API manager.", Spec: map[string]any{"parent": ref("backend", "backend"), "owns": []any{"src/"}}}
	inventory := []File{{Path: "src/main.go", Digest: "sha256:main", Mode: "100644"}, {Path: "docs/guide.md", Digest: "sha256:guide", Mode: "100644"}}
	for _, tc := range []struct {
		name        string
		definitions []core.Definition
		owner       string
	}{
		{"root and top-level delegate", []core.Definition{root, backend}, "backend"},
		{"two delegation levels", []core.Definition{root, backend, api}, "backend.api"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model, diagnostics := core.Compile([]core.Schema{Schema()}, tc.definitions, "delegated-selector")
			if len(diagnostics) != 0 {
				t.Fatalf("compile: %+v", diagnostics)
			}
			r := Analyze(model, inventory)
			if r.Status != "succeeded" {
				t.Fatalf("delegating an identical selector is valid: status=%s findings=%+v", r.Status, r.Findings)
			}
			owners := map[string]string{}
			for _, f := range r.Files {
				owners[f.Path] = f.Owner
			}
			want := core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: tc.owner, Name: tc.owner[strings.LastIndex(tc.owner, ".")+1:]}.Key()
			if owners["src/main.go"] != want || owners["docs/guide.md"] != rootManagerKey() {
				t.Fatalf("owners = %v, want src/main.go by %s and docs/guide.md by root", owners, want)
			}
		})
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

// BUG-01: a foreign public Statement that the Manager's own Check uses or own
// Artifact realizes is a Contract too, with its private relations hidden.
func TestContextIncludesContractsOfOwnChecksAndArtifacts(t *testing.T) {
	model, files := fixture(t, true, true, true)
	contract := map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "release-reservation"}
	cancelOrder := map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "orders", "name": "cancel-order"}
	contractID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "inventory", Name: "release-reservation"}).Key()
	ordersID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	for _, tc := range []struct {
		name     string
		edit     func(*core.Definition)
		contract bool
	}{
		{"no reference", func(*core.Definition) {}, false},
		{"own check uses", func(d *core.Definition) {
			if d.Kind == checkKind {
				d.Spec["uses"] = []any{cancelOrder, contract}
			}
		}, true},
		{"own artifact realizes", func(d *core.Definition) {
			if d.Kind == artifactKind && d.Metadata.Namespace == "orders" {
				d.Spec["realizes"] = []any{cancelOrder, contract}
			}
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definitions := append(copyDefinitions(model.Definitions), core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "internal-guard"}, Purpose: "Private implementation detail.", Spec: map[string]any{"category": "rule", "description": "Internal inventory guard."}})
			for i := range definitions {
				switch {
				case definitions[i].Kind == statementKind && definitions[i].Metadata.Name == "cancel-order":
					delete(definitions[i].Spec, "requires")
				case definitions[i].Kind == statementKind && definitions[i].Metadata.Name == "release-reservation":
					definitions[i].Spec["uses"] = []any{map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "internal-guard"}}
				}
				tc.edit(&definitions[i])
			}
			candidate, diagnostics := core.Compile(model.Schemas, definitions, "own-references")
			if len(diagnostics) != 0 {
				t.Fatalf("compile: %+v", diagnostics)
			}
			r := Analyze(candidate, files)
			if r.Status != "succeeded" {
				t.Fatalf("fixture must analyze cleanly: %s %+v", r.Status, r.Findings)
			}
			ctx, err := Context(r, ordersID)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(ctx.Contracts) == 1 && ctx.Contracts[0].ID == contractID; got != tc.contract {
				t.Fatalf("contracts = %+v, want release-reservation: %v", ctx.Contracts, tc.contract)
			}
			if tc.contract && (len(ctx.Contracts[0].Uses) != 0 || len(ctx.Contracts[0].Requires) != 0) {
				t.Fatalf("contract leaked private relation identities: %+v", ctx.Contracts[0])
			}
		})
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

// BUG-01: moving a definition to another model file routes both files, their
// owner and what each declares; Source is not in the model digest, so no
// other rule names the move (DEC-021).
func TestImpactRoutesBothModelFilesOfAMovedDefinition(t *testing.T) {
	definitions, model, files := sourcedFixture(t)
	base := analyzeDefinitions(t, model, definitions, files)
	moved := copyDefinitions(definitions)
	var from string
	for i := range moved {
		if moved[i].Metadata.Name == "cancel-order-code" {
			from = moved[i].Source.Path
			moved[i].Source.Path = ".markitect/model/orders/moved.yaml"
		}
	}
	impact := Impact(base, analyzeDefinitions(t, model, moved, files))
	if len(impact.Unknown) != 0 || len(impact.ChangedDefinitions) != 0 || !contains(impact.Files, from) || !contains(impact.Files, ".markitect/model/orders/moved.yaml") {
		t.Fatalf("moved definition: changed=%v files=%v unknown=%v, want both model files", impact.ChangedDefinitions, impact.Files, impact.Unknown)
	}
}

// BUG-01: when a Statement changes, the Manager whose own Decision is about it
// is routed; that Statement is one of its Contracts.
func TestImpactRoutesTheOwnerOfADecisionOnAChangedStatement(t *testing.T) {
	model, files := fixture(t, true, true, true)
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "stock-unit"}, Purpose: "Unit of stock.", Spec: map[string]any{"category": "concept", "description": "Pieces.", "public": true}},
		decisionDefinition("orders", "count-pieces", "inventory", "stock-unit", "Orders count pieces."),
	)
	analyze := func(description string) Report {
		changed := copyDefinitions(definitions)
		for i := range changed {
			if changed[i].Metadata.Name == "stock-unit" {
				changed[i].Spec["description"] = description
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, changed, "decision-subject")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	impact := Impact(analyze("Pieces."), analyze("Packs of ten pieces."))
	orders := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"}).Key()
	if len(impact.Unknown) != 0 || !contains(impact.Managers, orders) {
		t.Fatalf("the owner of a decision on the changed statement is not routed: managers=%v unknown=%v", impact.Managers, impact.Unknown)
	}
}

// BUG-01: a Check that must run again because an Artifact declares it routes
// its owner, like a Check that exercises the changed Statement.
func TestImpactRoutesTheOwnerOfAnArtifactCheck(t *testing.T) {
	model, files := fixture(t, true, true, true)
	root := map[string]any{"apiVersion": APIVersion, "kind": managerKind, "namespace": "", "name": "root"}
	smokeRef := map[string]any{"apiVersion": APIVersion, "kind": checkKind, "namespace": "qa", "name": "smoke"}
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Namespace: "qa", Name: "qa"}, Purpose: "Quality.", Spec: map[string]any{"parent": root, "owns": []any{"qa/"}}},
		core.Definition{APIVersion: APIVersion, Kind: checkKind, Metadata: core.Metadata{Namespace: "qa", Name: "smoke"}, Purpose: "Smoke test.", Spec: map[string]any{"command": []any{"go", "test", "./qa"}}},
	)
	for i := range definitions {
		if definitions[i].Metadata.Name == "cancel-order-code" {
			definitions[i].Spec["checks"] = append(append([]any(nil), definitions[i].Spec["checks"].([]any)...), smokeRef)
		}
	}
	analyze := func(description string) Report {
		changed := copyDefinitions(definitions)
		for i := range changed {
			if changed[i].Metadata.Name == "cancel-order" {
				changed[i].Spec["description"] = description
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, changed, "artifact-check")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	impact := Impact(analyze("Before."), analyze("After."))
	smoke := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: checkKind, Namespace: "qa", Name: "smoke"}).Key()
	qa := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: "qa", Name: "qa"}).Key()
	if !contains(impact.Checks, smoke) || !contains(impact.Managers, qa) {
		t.Fatalf("artifact check or its owner not routed: checks=%v managers=%v", impact.Checks, impact.Managers)
	}
}
