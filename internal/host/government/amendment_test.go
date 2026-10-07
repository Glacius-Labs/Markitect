package government

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
)

func amendmentFixture(t *testing.T) (Source, Model, Order, Plan) {
	t.Helper()
	source := testSource()
	for i := range source.Schemas {
		if source.Schemas[i].APIVersion == testAPI {
			source.Schemas[i].Kinds["Rule"].Properties["businessRule"] = core.Property{Purpose: "Business rule statement", Type: core.TypeString, MinCount: 0, MaxCount: 1}
		}
	}
	mutate(&source, "Mandate", "orders", func(d *core.Definition) {
		d.Spec["actions"] = []any{"implement", "review", "amend-model"}
	})
	source.Definitions = append(source.Definitions,
		testDef("Artifact", "model", map[string]any{"path": "model/government.yaml", "class": "canonical", "writer": testRef("Area", "orders")}),
		testDef("Realization", "cancel-model", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "model"), "role": "canonical business model"}),
	)
	prior := Compile(source)
	if len(prior.Findings) != 0 {
		t.Fatalf("prior model: %+v", prior.Findings)
	}
	order := testOrder(prior)
	order.Action = "amend-model"
	report := amendmentReport(t)
	plan := BuildPlan(prior, order, report, "model/government.yaml")
	if plan.Status != "planned-within-declared-boundary" {
		t.Fatalf("amendment plan: %+v", plan.Findings)
	}
	return source, prior, order, plan
}

func amendmentReport(t *testing.T) inventory.Report {
	return amendmentReportWithOptions(t, inventory.Options{Roots: []string{"."}})
}

func amendmentReportWithOptions(t *testing.T, options inventory.Options) inventory.Report {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"handler.go":            "package orders\nfunc Cancel() {}\n",
		"handler_test.go":       "package orders\n",
		"model/government.yaml": "kind: GovernmentSource\n",
	}
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	returnReport, err := inventory.Capture(dir, options)
	if err != nil {
		t.Fatal(err)
	}
	return returnReport
}

func amendmentSource(prior Source) Source {
	out := prior
	out.Definitions = append([]core.Definition(nil), prior.Definitions...)
	for i := range out.Definitions {
		out.Definitions[i].Spec = cloneSpec(prior.Definitions[i].Spec)
	}
	out.Schemas = append([]core.Schema(nil), prior.Schemas...)
	return out
}

func cloneSpec(source map[string]any) map[string]any {
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func TestAssessAmendmentAllowsSubstantivePriorDelegationForFreshReview(t *testing.T) {
	source, prior, order, plan := amendmentFixture(t)
	candidate := amendmentSource(source)
	mutate(&candidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Cancellation releases reserved stock" })
	mutate(&candidate, "Realization", "cancel-handler", func(d *core.Definition) { d.Spec["role"] = "release stock after cancellation" })
	for i := range candidate.Definitions {
		candidate.Definitions[i].Source = core.Source{Path: "candidate.yaml", Digest: "sha256:candidate"}
	}
	for i := range candidate.Schemas {
		candidate.Schemas[i].Source = core.Source{Path: "candidate-schema.yaml", Digest: "sha256:candidate-schema"}
	}
	assessment := AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "eligible-for-fresh-review" || len(assessment.Findings) != 0 {
		t.Fatalf("authorized change blocked: %+v", assessment)
	}
	if len(assessment.Changed) != 2 || assessment.Changed[0] != testID("Realization", "cancel-handler") || assessment.Changed[1] != testID("Rule", "cancel") || assessment.Digest == "" {
		t.Fatalf("changed identity or binding missing: %+v", assessment)
	}
	if len(assessment.Subjects) != 1 || assessment.Subjects[0] != testID("Rule", "cancel") {
		t.Fatalf("changed subject scope missing: %+v", assessment.Subjects)
	}
	if assessment.PriorConstitution != prior.Digest || assessment.ProposedConstitution == prior.Digest {
		t.Fatalf("assessment did not bind prior and proposed models: %+v", assessment)
	}
}

func TestAssessAmendmentRejectsSelfAuthorityAndProtectedGoal(t *testing.T) {
	source, prior, order, plan := amendmentFixture(t)
	protectedSource := amendmentSource(source)
	mutate(&protectedSource, "Constitution", "active", func(d *core.Definition) {
		d.Spec["protectedSubjects"] = []any{testRef("Rule", "cancel")}
	})
	protectedPrior := Compile(protectedSource)
	if len(protectedPrior.Findings) != 0 {
		t.Fatalf("protected prior model: %+v", protectedPrior.Findings)
	}
	protectedOrder := testOrder(protectedPrior)
	protectedOrder.Action = "amend-model"
	protectedPlan := BuildPlan(protectedPrior, protectedOrder, amendmentReport(t), "model/government.yaml")
	protectedCandidate := amendmentSource(protectedSource)
	mutate(&protectedCandidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Changed protected goal" })
	protectedAssessment := AssessAmendment(protectedPrior, protectedCandidate, protectedOrder, protectedPlan)
	if protectedAssessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(protectedAssessment, "amendment.protected-subject") {
		t.Fatalf("protected root goal was eligible: %+v", protectedAssessment)
	}

	selfCandidate := amendmentSource(source)
	mutate(&selfCandidate, "Constitution", "active", func(d *core.Definition) {
		d.Purpose = "Candidate changes the governing Constitution purpose"
	})
	selfAssessment := AssessAmendment(prior, selfCandidate, order, plan)
	if selfAssessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(selfAssessment, "amendment.constitution") {
		t.Fatalf("candidate changed its own decision rule: %+v", selfAssessment)
	}
}

func TestAssessAmendmentRequiresPriorMandateAndExactFrozenPlan(t *testing.T) {
	source, prior, order, plan := amendmentFixture(t)
	candidate := amendmentSource(source)
	mutate(&candidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Change without mandate" })
	withoutAuthority := amendmentSource(source)
	mutate(&withoutAuthority, "Mandate", "orders", func(d *core.Definition) { d.Spec["actions"] = []any{"implement", "review"} })
	changedPrior := Compile(withoutAuthority)
	changedOrder := testOrder(changedPrior)
	changedOrder.Action = "amend-model"
	changedPlan := BuildPlan(changedPrior, changedOrder, amendmentReport(t), "model/government.yaml")
	assessment := AssessAmendment(changedPrior, candidate, changedOrder, changedPlan)
	if assessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(assessment, "amendment.authority") {
		t.Fatalf("missing prior amend-model mandate accepted: %+v", assessment)
	}

	alteredOrder := order
	alteredOrder.Purpose = "different request"
	planMismatch := AssessAmendment(prior, candidate, alteredOrder, plan)
	if planMismatch.Status != "blocked-escalation-required" || !hasAmendmentFinding(planMismatch, "amendment.plan") {
		t.Fatalf("stale plan accepted for changed order: %+v", planMismatch)
	}

	alteredBoundary := amendmentSource(source)
	alteredBoundary.Observation = inventory.Options{Roots: []string{"internal"}}
	boundaryAssessment := AssessAmendment(prior, candidate, order, plan)
	if boundaryAssessment.Status != "eligible-for-fresh-review" {
		t.Fatalf("test precondition failed: %+v", boundaryAssessment)
	}
	boundaryAssessment = AssessAmendment(prior, alteredBoundary, order, plan)
	if boundaryAssessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(boundaryAssessment, "amendment.observation") {
		t.Fatalf("candidate changed prior acquisition boundary: %+v", boundaryAssessment)
	}
}

func TestAssessAmendmentRejectsUnsupportedIdentityAndOwnershipChanges(t *testing.T) {
	source, prior, order, plan := amendmentFixture(t)
	newDomain := amendmentSource(source)
	newDomain.Definitions = append(newDomain.Definitions, testDef("Rule", "new", map[string]any{"businessRule": "New subject"}))
	identityAssessment := AssessAmendment(prior, newDomain, order, plan)
	if identityAssessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(identityAssessment, "amendment.domain-identities") {
		t.Fatalf("domain identity expansion accepted: %+v", identityAssessment)
	}

	reassigned := amendmentSource(source)
	mutate(&reassigned, "Responsibility", "cancel", func(d *core.Definition) { d.Spec["area"] = testRef("Area", "cancellation") })
	ownershipAssessment := AssessAmendment(prior, reassigned, order, plan)
	if ownershipAssessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(ownershipAssessment, "amendment.ownership-coverage") {
		t.Fatalf("new owner outside the prior authorized plan accepted: %+v", ownershipAssessment)
	}
}

func TestProtectedSubjectsMustResolveToDomainDefinitions(t *testing.T) {
	source := testSource()
	mutate(&source, "Constitution", "active", func(d *core.Definition) {
		d.Spec["protectedSubjects"] = []any{testRef("Rule", "missing"), testRef("Area", "root")}
	})
	model := Compile(source)
	if !hasFinding(model.Findings, "subject.unresolved") || !hasFinding(model.Findings, "constitution.protected-subject") {
		t.Fatalf("invalid protected subjects were not rejected: %+v", model.Findings)
	}
}

func TestAssessAmendmentTreatsCustomResponsibilityAndRealizationKindsAsDomain(t *testing.T) {
	source, _, _, _ := amendmentFixture(t)
	customAPI := "example.com/v1"
	source.Schemas = append(source.Schemas, core.Schema{APIVersion: customAPI, Purpose: "Custom domain vocabulary", Kinds: map[string]core.Kind{
		"Responsibility": {Purpose: "A domain responsibility term", Properties: map[string]core.Property{
			"note": {Purpose: "Meaning of the domain term", Type: core.TypeString, MinCount: 1, MaxCount: 1},
		}},
		"Realization": {Purpose: "A domain realization term", Properties: map[string]core.Property{
			"note": {Purpose: "Meaning of the domain term", Type: core.TypeString, MinCount: 1, MaxCount: 1},
		}},
	}})
	source.Definitions = append(source.Definitions,
		core.Definition{APIVersion: customAPI, Kind: "Responsibility", Metadata: core.Metadata{Namespace: "shop", Name: "domain-responsibility"}, Purpose: "Domain concept", Spec: map[string]any{"note": "first"}},
		core.Definition{APIVersion: customAPI, Kind: "Realization", Metadata: core.Metadata{Namespace: "shop", Name: "domain-realization"}, Purpose: "Domain concept", Spec: map[string]any{"note": "first"}},
	)
	prior := Compile(source)
	if len(prior.Findings) != 0 {
		t.Fatalf("prior with custom kinds: %+v", prior.Findings)
	}
	order := testOrder(prior)
	order.Action = "amend-model"
	plan := BuildPlan(prior, order, amendmentReport(t), "model/government.yaml")
	candidate := amendmentSource(source)
	for i := range candidate.Definitions {
		if candidate.Definitions[i].APIVersion == customAPI {
			candidate.Definitions[i].Spec["note"] = "changed domain meaning"
		}
	}
	assessment := AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "blocked-escalation-required" || len(assessment.Subjects) != 2 || !hasAmendmentFinding(assessment, "amendment.outside-plan") {
		t.Fatalf("custom domain Kind names bypassed scope classification: %+v", assessment)
	}
}

func TestAssessAmendmentRejectsOffPlanAndForeignRealizationRewires(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		name := "off-plan managed target"
		class := "realization"
		path := "unselected/notes.md"
		if foreign {
			name = "foreign target"
			class = "foreign"
			path = "external/reference.md"
		}
		t.Run(name, func(t *testing.T) {
			source, _, _, _ := amendmentFixture(t)
			artifact := testDef("Artifact", "alternate", map[string]any{"path": path, "class": class, "writer": testRef("Area", "orders")})
			if foreign {
				artifact.Spec = map[string]any{"path": path, "class": class}
			}
			source.Definitions = append(source.Definitions, artifact)
			prior := Compile(source)
			if len(prior.Findings) != 0 {
				t.Fatalf("prior model: %+v", prior.Findings)
			}
			order := testOrder(prior)
			order.Action = "amend-model"
			plan := BuildPlan(prior, order, amendmentReport(t), "model/government.yaml")
			candidate := amendmentSource(source)
			mutate(&candidate, "Realization", "cancel-handler", func(d *core.Definition) { d.Spec["artifact"] = testRef("Artifact", "alternate") })
			assessment := AssessAmendment(prior, candidate, order, plan)
			if assessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(assessment, "amendment.realization-path") {
				t.Fatalf("off-plan/foreign realization rewire accepted: %+v", assessment)
			}
		})
	}
}

func TestAssessAmendmentRequiresManagedRealizationForEachChangedSubject(t *testing.T) {
	source, _, _, _ := amendmentFixture(t)
	source.Definitions = append(source.Definitions, testDef("Artifact", "foreign", map[string]any{"path": "external/rules.md", "class": "foreign"}))
	prior := Compile(source)
	if len(prior.Findings) != 0 {
		t.Fatalf("prior model: %+v", prior.Findings)
	}
	order := testOrder(prior)
	order.Action = "amend-model"
	plan := BuildPlan(prior, order, amendmentReport(t), "model/government.yaml")
	candidate := amendmentSource(source)
	for i := range candidate.Definitions {
		if candidate.Definitions[i].APIVersion == APIVersion && candidate.Definitions[i].Kind == "Realization" && identity(candidate.Definitions[i].Spec["subject"]).Key() == testID("Rule", "cancel").Key() {
			candidate.Definitions[i].Spec["artifact"] = testRef("Artifact", "foreign")
		}
	}
	assessment := AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(assessment, "amendment.realization-required") {
		t.Fatalf("foreign links erased the managed realization obligation: %+v", assessment)
	}
}

func TestAssessAmendmentRetainsUnchangedForeignContextBesideManagedRealization(t *testing.T) {
	source, _, _, _ := amendmentFixture(t)
	source.Definitions = append(source.Definitions,
		testDef("Artifact", "foreign-context", map[string]any{"path": "external/reference.md", "class": "foreign"}),
		testDef("Realization", "cancel-reference", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "foreign-context"), "role": "external business context"}),
	)
	prior := Compile(source)
	if len(prior.Findings) != 0 {
		t.Fatalf("prior model: %+v", prior.Findings)
	}
	order := testOrder(prior)
	order.Action = "amend-model"
	plan := BuildPlan(prior, order, amendmentReport(t), "model/government.yaml")
	candidate := amendmentSource(source)
	mutate(&candidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Changed while retaining external context" })
	assessment := AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "eligible-for-fresh-review" {
		t.Fatalf("unchanged foreign context prevented managed amendment: %+v", assessment.Findings)
	}
}

func TestAssessAmendmentRevalidatesPlanDigestAndModelPath(t *testing.T) {
	source, prior, order, plan := amendmentFixture(t)
	candidate := amendmentSource(source)
	mutate(&candidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Updated business rule" })
	tampered := plan
	tampered.ModelPath = "handler.go"
	tampered.Action = "implement"
	assessment := AssessAmendment(prior, candidate, order, tampered)
	if assessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(assessment, "amendment.plan-digest") || !hasAmendmentFinding(assessment, "amendment.model-path") {
		t.Fatalf("altered plan fields were trusted under the original digest: %+v", assessment)
	}
}

func TestAssessAmendmentNormalizesInventoryBoundaryOrdering(t *testing.T) {
	source, _, _, _ := amendmentFixture(t)
	options := inventory.Options{
		Roots: []string{"."},
		Exclusions: []inventory.Boundary{
			{Path: "tmp", Reason: "temporary outputs"},
			{Path: "cache", Reason: "generated cache"},
		},
	}
	source.Observation = options
	prior := Compile(source)
	order := testOrder(prior)
	order.Action = "amend-model"
	plan := BuildPlan(prior, order, amendmentReportWithOptions(t, options), "model/government.yaml")
	candidate := amendmentSource(source)
	mutate(&candidate, "Rule", "cancel", func(d *core.Definition) { d.Spec["businessRule"] = "Updated business rule" })
	assessment := AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "eligible-for-fresh-review" {
		t.Fatalf("equivalent unsorted boundary ordering rejected: %+v", assessment.Findings)
	}
	candidate.Observation.Exclusions[0].Reason = "different boundary"
	assessment = AssessAmendment(prior, candidate, order, plan)
	if assessment.Status != "blocked-escalation-required" || !hasAmendmentFinding(assessment, "amendment.observation") {
		t.Fatalf("changed acquisition boundary accepted: %+v", assessment)
	}
}

func TestAmendmentClosureFollowsSharedGovernmentRealizations(t *testing.T) {
	source := testSource()
	other := testDef("Rule", "shared-only", map[string]any{})
	source.Definitions = append(source.Definitions,
		other,
		testDef("Responsibility", "shared-only", map[string]any{"subject": testRef("Rule", "shared-only"), "area": testRef("Area", "orders")}),
		testDef("Realization", "shared-only", map[string]any{"subject": testRef("Rule", "shared-only"), "artifact": testRef("Artifact", "handler"), "role": "shared implementation"}),
	)
	prior := Compile(source)
	if len(prior.Findings) != 0 {
		t.Fatalf("shared-realization prior: %+v", prior.Findings)
	}
	closure := amendmentClosure(definitionsByIdentity(prior.Canonical.Definitions), prior.Canonical.Edges, testID("Rule", "cancel"))
	if !amendmentContainsIdentity(closure, testID("Rule", "shared-only")) {
		t.Fatalf("shared Government Realization did not expand the closure: %+v", closure)
	}
}

func hasAmendmentFinding(assessment AmendmentAssessment, code string) bool {
	for _, finding := range assessment.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
