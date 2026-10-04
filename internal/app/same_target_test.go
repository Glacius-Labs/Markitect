package app

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

const appSameTargetAPI = "equality.tests.example/v1"

func TestSameTargetImpactTracksPathInputsAndExactConsumers(t *testing.T) {
	beforeSnapshot := appSameTargetSnapshot()
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	path := "resources/features/order-management.yaml"
	afterSnapshot.Files[path] = []byte(strings.Replace(string(afterSnapshot.Files[path]), "name: orders", "name: billing", 1))
	before, after := parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot)
	if len(before.Diagnostics) != 0 || len(after.Diagnostics) == 0 {
		t.Fatalf("expected a valid baseline and mismatched-path finding: before=%+v after=%+v", before.Diagnostics, after.Diagnostics)
	}
	impact := Changes(before, after)
	useCase := appSameTargetKey(t, before, "UseCase", "create-order")
	feature := appSameTargetKey(t, before, "Feature", "order-management")
	workflow := appSameTargetKey(t, before, "Reviewer", "review-order")
	handler := appSameTargetKey(t, before, "Processor", "create-order")
	coordinator := appSameTargetKey(t, before, "Coordinator", "follow-order")
	unrelated := appSameTargetKey(t, before, "Reviewer", "unrelated")
	for _, key := range []string{feature, useCase, workflow, handler, coordinator} {
		if !contains(impact.Affected, key) {
			t.Errorf("changed path input or its exact consumer was omitted: %s; impact=%+v", key, impact)
		}
	}
	if contains(impact.Affected, unrelated) || hasImpactCause(impact, "context-policy-effect") {
		t.Fatalf("same-target input change was broadened by unrelated Context-only relations: %+v", impact)
	}
	foundDependency, foundContext, foundInvalidation, foundChainedContext := false, false, false, false
	for _, cause := range impact.Causes {
		switch cause.Kind {
		case "policy-dependency":
			if cause.Resource == useCase && cause.Constraint == "feature-module-agrees" && cause.To == feature && cause.DomainAPIVersion == appSameTargetAPI && cause.Snapshot == "base" && cause.Path == path {
				foundDependency = true
			}
		case "policy-context":
			if cause.Resource == workflow && cause.To == useCase && cause.Relation == "documentsUseCase" {
				foundContext = true
			}
			if cause.Resource == coordinator && cause.To == handler && cause.Relation == "documentsProcessor" {
				foundChainedContext = true
			}
		case "invalidation":
			if cause.Resource == handler && cause.To == useCase && cause.Relation == "handlesUseCase" {
				foundInvalidation = true
			}
		}
	}
	if !foundDependency || !foundContext || !foundInvalidation || !foundChainedContext {
		t.Fatalf("missing source-identified policy/context/invalidation explanations: dependency=%v context=%v chained-context=%v invalidation=%v causes=%+v", foundDependency, foundContext, foundChainedContext, foundInvalidation, impact.Causes)
	}
}

func TestStructuralDiagnosticsClassifyByFailedResultIdentityNotDiagnosticCode(t *testing.T) {
	snapshot := appSameTargetSnapshot()
	snapshot.Files["domains/equality.yaml"] = []byte(strings.Replace(string(snapshot.Files["domains/equality.yaml"]), "name: feature-module-agrees", "name: path", 1))
	featurePath := "resources/features/order-management.yaml"
	snapshot.Files[featurePath] = []byte(strings.Replace(string(snapshot.Files[featurePath]), "name: orders", "name: billing", 1))
	p := parsePolicyImpactProject(t, snapshot)
	if len(p.Diagnostics) == 0 {
		t.Fatal("fixture should have one failed policy diagnostic")
	}
	if len(p.StructuralDiagnostics()) != 0 {
		t.Fatalf("failed result named path was misclassified from its diagnostic code: %+v", p.StructuralDiagnostics())
	}
	for _, diagnostic := range p.Diagnostics {
		if diagnostic.Code == "constraint.path" && (diagnostic.PolicyResult == nil || diagnostic.PolicyResult.Constraint != "path") {
			t.Fatalf("failed policy diagnostic lacks exact identity: %+v", diagnostic)
		}
	}

	broken := appSameTargetSnapshot()
	broken.Files[featurePath] = []byte(strings.Replace(string(broken.Files[featurePath]), "  module: {apiVersion: equality.tests.example/v1, kind: Module, name: orders}\n", "", 1))
	structurallyInvalid := parsePolicyImpactProject(t, broken)
	if len(structurallyInvalid.StructuralDiagnostics()) == 0 {
		t.Fatal("same-target traversal failure was not retained as structural")
	}
	for _, diagnostic := range structurallyInvalid.Diagnostics {
		if diagnostic.Code == "constraint.path" && diagnostic.PolicyResult != nil {
			t.Fatalf("invalid traversal received a failed-policy identity: %+v", diagnostic)
		}
	}
}

func TestAnalyzeContextAllowsPolicyFailureButBlocksStructuralFailure(t *testing.T) {
	snapshot := appSameTargetSnapshot()
	snapshot.Files["domains/equality.yaml"] = []byte(strings.Replace(string(snapshot.Files["domains/equality.yaml"]), "name: feature-module-agrees", "name: path", 1))
	featurePath := "resources/features/order-management.yaml"
	snapshot.Files[featurePath] = []byte(strings.Replace(string(snapshot.Files[featurePath]), "name: orders", "name: billing", 1))
	p := parsePolicyImpactProject(t, snapshot)
	entry := appSameTargetKey(t, p, "UseCase", "create-order")
	if _, err := CompileContext(p, entry, "test"); err == nil {
		t.Fatal("strict context compilation accepted failed policy")
	}
	first, err := AnalyzeContext(p, entry, "test", "sha256:analysis-tool")
	if err != nil {
		t.Fatalf("analyze policy-failing context: %v", err)
	}
	second, err := AnalyzeContext(p, entry, "test", "sha256:analysis-tool")
	if err != nil {
		t.Fatal(err)
	}
	if first.Analysis == nil || first.Analysis.Mode != PolicyFailureAnalysisMode || !first.Analysis.Complete || first.Analysis.Candidate.StructuralStatus != "passed" || first.Analysis.Candidate.PolicyStatus != "failed" || first.Analysis.Candidate.ValidationStatus != "failed" {
		t.Fatalf("analysis state was not visibly bound to the candidate: %+v", first.Analysis)
	}
	if first.Analysis.Candidate.ModelDigest == "" || first.Analysis.Candidate.ConfigDigest == "" || first.Digest != second.Digest {
		t.Fatalf("analysis omitted model identity or was nondeterministic: first=%+v second=%+v", first, second)
	}
	foundFailure := false
	for _, result := range first.PolicyResults {
		if result.Subject == entry && result.Constraint == "path" && result.Status == "failed" {
			foundFailure = true
		}
	}
	if !foundFailure {
		t.Fatalf("context omitted the selected failed policy result: %+v", first.PolicyResults)
	}
	for _, input := range first.Inputs {
		if input.Key == appSameTargetKey(t, p, "Module", "billing") || input.Key == appSameTargetKey(t, p, "Feature", "order-management") {
			t.Fatalf("policy-only same-target traversal changed Context closure: %+v", input)
		}
	}

	broken := appSameTargetSnapshot()
	broken.Files[featurePath] = []byte(strings.Replace(string(broken.Files[featurePath]), "  module: {apiVersion: equality.tests.example/v1, kind: Module, name: orders}\n", "", 1))
	structurallyInvalid := parsePolicyImpactProject(t, broken)
	brokenEntry := appSameTargetKey(t, structurallyInvalid, "UseCase", "create-order")
	if _, err := AnalyzeContext(structurallyInvalid, brokenEntry, "test"); err == nil {
		t.Fatal("diagnostic context proceeded through structural same-target failure")
	}
}

func TestSameTargetImpactRetainsIncompleteOldAndNewPathPrefixes(t *testing.T) {
	beforeSnapshot := appSameTargetSnapshot()
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	featurePath := "resources/features/order-management.yaml"
	afterSnapshot.Files[featurePath] = []byte(strings.Replace(string(afterSnapshot.Files[featurePath]), "spec:\n  module: {apiVersion: equality.tests.example/v1, kind: Module, name: orders}\n", "spec: {}\n", 1))
	before, after := parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot)
	if len(before.Diagnostics) != 0 || len(after.Diagnostics) == 0 {
		t.Fatalf("expected structurally incomplete candidate traversal: before=%+v after=%+v", before.Diagnostics, after.Diagnostics)
	}
	impact := Changes(before, after)
	useCase := appSameTargetKey(t, before, "UseCase", "create-order")
	feature := appSameTargetKey(t, before, "Feature", "order-management")
	if !contains(impact.Affected, useCase) {
		t.Fatalf("old/new valid path prefix did not invalidate its policy subject: %+v", impact)
	}
	foundOld, foundNew := false, false
	for _, cause := range impact.Causes {
		if cause.Kind == "policy-dependency" && cause.Resource == useCase && cause.To == feature && cause.Constraint == "feature-module-agrees" {
			foundOld = foundOld || cause.Snapshot == "base"
			foundNew = foundNew || cause.Snapshot == "candidate"
		}
	}
	if !foundOld || !foundNew {
		t.Fatalf("policy dependency evidence lost base/candidate provenance for the incomplete prefix: %+v", impact.Causes)
	}
}

func TestSameTargetContextCarriesComparisonWithoutIncludingPathResources(t *testing.T) {
	p := parsePolicyImpactProject(t, appSameTargetSnapshot())
	if len(p.Diagnostics) != 0 {
		t.Fatalf("fixture diagnostics: %+v", p.Diagnostics)
	}
	useCase := appSameTargetKey(t, p, "UseCase", "create-order")
	ctx, err := CompileContext(p, useCase, "test")
	if err != nil {
		t.Fatal(err)
	}
	var comparisonFound bool
	for _, result := range ctx.PolicyResults {
		if result.Constraint == "feature-module-agrees" && result.Subject == useCase {
			comparisonFound = result.Comparison != nil && len(result.Comparison.Left.Steps) == 1 && len(result.Comparison.Right.Steps) == 2
		}
	}
	if !comparisonFound {
		t.Fatalf("context omitted the ordered same-target trace: %+v", ctx.PolicyResults)
	}
	for _, input := range ctx.Inputs {
		if input.Key == appSameTargetKey(t, p, "Feature", "order-management") || input.Key == appSameTargetKey(t, p, "Module", "orders") {
			t.Fatalf("path resource with Context=false was included in context: %s", input.Key)
		}
	}
	changedIntermediate := clonePolicySnapshot(appSameTargetSnapshot())
	changedIntermediate.Files["resources/modules/orders.yaml"] = []byte(strings.Replace(string(changedIntermediate.Files["resources/modules/orders.yaml"]), "name: Orders", "name: Updated orders", 1))
	changedProject := parsePolicyImpactProject(t, changedIntermediate)
	changedContext, err := CompileContext(changedProject, useCase, "test")
	if err != nil {
		t.Fatal(err)
	}
	if changedContext.Digest == ctx.Digest {
		t.Fatal("context fingerprint did not bind the same-target outcome to its non-context path inputs")
	}
	for _, input := range changedContext.Inputs {
		if input.Key == appSameTargetKey(t, changedProject, "Module", "orders") || input.Key == appSameTargetKey(t, changedProject, "Feature", "order-management") {
			t.Fatalf("Context=false path input unexpectedly entered context: %s", input.Key)
		}
	}
	model, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	policyIndex := -1
	for i := range model.PolicyResults {
		if model.PolicyResults[i].Constraint == "feature-module-agrees" && model.PolicyResults[i].Subject == useCase {
			policyIndex = i
			break
		}
	}
	if policyIndex < 0 {
		t.Fatal("semantic model omitted same-target result")
	}
	model.PolicyResults[policyIndex].Comparison.Left.Relations[0] = "mutated"
	model.PolicyResults[policyIndex].Comparison.Left.Steps[0].Relation = "mutated"
	ctx.PolicyResults[0].Comparison.Left.Relations[0] = "context-mutated"
	graphTrace := ""
	for _, result := range p.Graph.PolicyResults {
		if result.Constraint == "feature-module-agrees" && result.Subject == useCase {
			graphTrace = result.Comparison.Left.Relations[0]
		}
	}
	if graphTrace == "mutated" || graphTrace == "context-mutated" || graphTrace != "belongsToModule" {
		t.Fatal("model or context comparison aliases the canonical graph trace")
	}
	recompiled, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	if recompiled.PolicyResults[policyIndex].Comparison.Left.Relations[0] != "belongsToModule" || recompiled.PolicyResults[policyIndex].Comparison.Left.Steps[0].Relation != "belongsToModule" {
		t.Fatalf("consumer mutation contaminated the next normalized model: %+v", recompiled.PolicyResults[policyIndex].Comparison)
	}
}

func TestSameTargetTerminalResourceChangeInvalidatesPolicySubject(t *testing.T) {
	beforeSnapshot := appSameTargetSnapshot()
	afterSnapshot := clonePolicySnapshot(beforeSnapshot)
	terminalPath := "resources/modules/orders.yaml"
	afterSnapshot.Files[terminalPath] = []byte(strings.Replace(string(afterSnapshot.Files[terminalPath]), "name: Orders", "name: Order platform", 1))
	before, after := parsePolicyImpactProject(t, beforeSnapshot), parsePolicyImpactProject(t, afterSnapshot)
	impact := Changes(before, after)
	useCase := appSameTargetKey(t, before, "UseCase", "create-order")
	if !contains(impact.Affected, useCase) || contains(impact.Affected, appSameTargetKey(t, before, "Reviewer", "unrelated")) {
		t.Fatalf("terminal resource content change was not scoped to its path consumer: %+v", impact)
	}
}

func appSameTargetKey(t *testing.T, p *Project, kind, name string) string {
	t.Helper()
	for key, resource := range p.Graph.Resources {
		if resource.Kind == kind && resource.Metadata.Name == name {
			return key
		}
	}
	t.Fatalf("missing %s %s", kind, name)
	return ""
}

func appSameTargetSnapshot() *snapshot.Snapshot {
	domain := `apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: equality}
spec:
  apiVersion: equality.tests.example/v1
  kinds:
    UseCase:
      properties:
        module: {type: ref, refKind: Module}
        feature: {type: ref, refKind: Feature}
    Feature:
      properties:
        module: {type: ref, refKind: Module}
    Module:
      required: [name]
      properties: {name: {type: string}}
    Reviewer:
      properties: {useCase: {type: ref, refKind: UseCase}}
    Processor:
      properties: {useCase: {type: ref, refKind: UseCase}}
    Coordinator:
      properties: {processor: {type: ref, refKind: Processor}}
  relations:
    belongsToModule:
      field: module
      sourceKinds: [UseCase, Feature]
      targetKinds: [Module]
      context: false
      invalidate: false
    realizesFeature:
      field: feature
      sourceKinds: [UseCase]
      targetKinds: [Feature]
      context: false
      invalidate: false
    documentsUseCase:
      field: useCase
      sourceKinds: [Reviewer]
      targetKinds: [UseCase]
      context: true
      invalidate: false
    handlesUseCase:
      field: useCase
      sourceKinds: [Processor]
      targetKinds: [UseCase]
      context: true
      invalidate: true
    documentsProcessor:
      field: processor
      sourceKinds: [Coordinator]
      targetKinds: [Processor]
      context: true
      invalidate: false
  constraints:
    - name: feature-module-agrees
      select: {kind: UseCase}
      assert:
        op: same-target
        left: [belongsToModule]
        right: [realizesFeature, belongsToModule]
`
	project := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: equality}
spec:
  domains: [domains/equality.yaml]
  areas: [{name: app, path: resources}]
`
	ref := func(kind, name string) string {
		return "{apiVersion: " + appSameTargetAPI + ", kind: " + kind + ", name: " + name + "}"
	}
	resource := func(kind, name, fields string) []byte {
		if fields == "" {
			return []byte("apiVersion: " + appSameTargetAPI + "\nkind: " + kind + "\nmetadata: {name: " + name + ", namespace: app}\nspec: {}\n")
		}
		return []byte("apiVersion: " + appSameTargetAPI + "\nkind: " + kind + "\nmetadata: {name: " + name + ", namespace: app}\nspec:\n" + fields)
	}
	return &snapshot.Snapshot{ID: "same-target", Provisional: true, Files: map[string][]byte{
		"markitect.yaml": []byte(project), "domains/equality.yaml": []byte(domain),
		"resources/usecases/create-order.yaml":         resource("UseCase", "create-order", "  module: "+ref("Module", "orders")+"\n  feature: "+ref("Feature", "order-management")+"\n"),
		"resources/features/order-management.yaml":     resource("Feature", "order-management", "  module: "+ref("Module", "orders")+"\n"),
		"resources/modules/orders.yaml":                resource("Module", "orders", "  name: Orders\n"),
		"resources/modules/billing.yaml":               resource("Module", "billing", "  name: Billing\n"),
		"resources/reviewers/review-order.yaml":        resource("Reviewer", "review-order", "  useCase: "+ref("UseCase", "create-order")+"\n"),
		"resources/reviewers/unrelated.yaml":           resource("Reviewer", "unrelated", ""),
		"resources/processors/create-order.yaml":       resource("Processor", "create-order", "  useCase: "+ref("UseCase", "create-order")+"\n"),
		"resources/000-coordinators/follow-order.yaml": resource("Coordinator", "follow-order", "  processor: "+ref("Processor", "create-order")+"\n"),
	}, Modes: map[string]string{}}
}
