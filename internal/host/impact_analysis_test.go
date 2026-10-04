package host

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring/contentpackage"
)

func TestAnalyzeImpactSeparatesDirectPolicyChangesFromConservativeAffectedSet(t *testing.T) {
	baseSnapshot := policyImpactSnapshot(t, "", false, true, true)
	candidateSnapshot := clonePolicySnapshot(baseSnapshot)
	policy := `  constraints:
    - name: selected-modules-require-intent
      select: {kind: Module, labels: {governed: yes}}
      assert: {op: equal, field: intent, value: expected}`
	domainPath := "domains/engineering.yaml"
	domain := strings.Replace(string(candidateSnapshot.Files[domainPath]), "  constraints:\n    \n", "", 1)
	candidateSnapshot.Files[domainPath] = []byte(strings.TrimRight(domain, "\r\n") + "\n" + policy + "\n")
	ordersPath := "resources/orders.yaml"
	orders := string(candidateSnapshot.Files[ordersPath])
	candidateSnapshot.Files[ordersPath] = []byte(strings.Replace(orders, "  intent: expected\n", "  intent: changed\n", 1))

	base, candidate := parsePolicyImpactProject(t, baseSnapshot), parsePolicyImpactProject(t, candidateSnapshot)
	if len(base.StructuralDiagnostics()) != 0 || len(candidate.StructuralDiagnostics()) != 0 {
		t.Fatalf("fixture must be structurally valid: base=%+v candidate=%+v", base.StructuralDiagnostics(), candidate.StructuralDiagnostics())
	}
	impact, err := AnalyzeImpact(base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if impact.Analysis == nil || !impact.Analysis.Complete || impact.Analysis.Base.PolicyStatus != "passed" || impact.Analysis.Candidate.PolicyStatus != "failed" {
		t.Fatalf("analysis status did not describe both fixed models: %+v", impact.Analysis)
	}
	if len(impact.PolicyChanges) != 1 || impact.PolicyChanges[0].Base.Status != "not-applicable" || impact.PolicyChanges[0].Base.AbsenceReason != "constraint-not-defined" || impact.PolicyChanges[0].Candidate.Status != "failed" {
		t.Fatalf("expected a not-defined -> failed policy delta: %+v", impact.PolicyChanges)
	}
	if impact.DirectPolicySubjectCount == nil || *impact.DirectPolicySubjectCount != 1 || len(impact.DirectPolicySubjects) != 1 || impact.DirectPolicySubjects[0] != "engineering/engineering.markitect.org/v1alpha1/Module/orders" {
		t.Fatalf("direct result subjects should contain only the failed selected Module: %+v", impact)
	}
	if impact.AffectedCount == nil || *impact.AffectedCount != len(impact.Affected) || *impact.AffectedCount < *impact.DirectPolicySubjectCount || !contains(impact.Affected, "engineering/Workflow/unrelated") {
		t.Fatalf("conservative impact should remain visible and count separately: %+v", impact)
	}
	delta := impact.PolicyChanges[0]
	if delta.Candidate.Result == nil || delta.Candidate.Result.Message == "" || delta.Candidate.DomainInput == nil || delta.Candidate.Definition == nil || delta.Candidate.DomainInput.Digest == "" {
		t.Fatalf("policy delta lost explanation or source provenance: %+v", delta)
	}
	if delta.Base.DomainInput == nil || delta.Base.Definition != nil {
		t.Fatalf("the missing v1 policy should retain its Domain source but have no constraint definition: %+v", delta.Base)
	}
	repeated, err := AnalyzeImpact(base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	firstYAML, err := YAML(impact)
	if err != nil {
		t.Fatal(err)
	}
	repeatedYAML, err := YAML(repeated)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(impact.PolicyChanges, repeated.PolicyChanges) || string(firstYAML) != string(repeatedYAML) {
		t.Fatal("policy delta ordering or serialized impact is nondeterministic")
	}
}

func TestAnalyzeImpactDoesNotCallPackageSourceRelocationPolicyDelta(t *testing.T) {
	baseSnapshot := policyImpactSnapshot(t, "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}", false, true, false)
	candidateSnapshot := clonePolicySnapshot(baseSnapshot)
	domainPath := "domains/engineering.yaml"
	candidateSnapshot.Files[domainPath] = append([]byte("# source-only relocation comment\n"), candidateSnapshot.Files[domainPath]...)
	impact, err := AnalyzeImpact(parsePolicyImpactProject(t, baseSnapshot), parsePolicyImpactProject(t, candidateSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.PolicyChanges) != 0 || impact.DirectPolicySubjectCount == nil || *impact.DirectPolicySubjectCount != 0 || len(impact.DirectPolicySubjects) != 0 {
		t.Fatalf("source bytes changed without semantic policy outcome change: %+v", impact.PolicyChanges)
	}
	if !contains(impact.Affected, "engineering/Workflow/unrelated") || !hasImpactCause(impact, "domain-definition") {
		t.Fatalf("conservative domain-input impact was incorrectly narrowed: %+v", impact)
	}
}

func TestAnalyzeImpactPackagePinProvenanceAndVersionOnlyChange(t *testing.T) {
	baseSnapshot := packagedPolicySnapshot(t, "1.0.0", "fixture:architecture@v1", "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}")
	candidateSnapshot := packagedPolicySnapshot(t, "2.0.0", "fixture:architecture@v2", "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}")
	base := parsePolicyImpactProject(t, baseSnapshot)
	candidate := parsePolicyImpactProject(t, candidateSnapshot)
	impact, err := AnalyzeImpact(base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.PolicyChanges) != 0 || impact.DirectPolicySubjectCount == nil || *impact.DirectPolicySubjectCount != 0 {
		t.Fatalf("package version/source-only change altered normalized results: %+v", impact.PolicyChanges)
	}
	if impact.AffectedCount == nil || *impact.AffectedCount != len(impact.Affected) || *impact.AffectedCount == 0 {
		t.Fatalf("package pin change should still retain conservative impact: %+v", impact)
	}

	oldPackage := packagedPolicySnapshot(t, "1.0.0", "fixture:architecture@v1", "")
	newPackage := packagedPolicySnapshot(t, "2.0.0", "fixture:architecture@v2", "- name: selected-commands-require-validator\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}")
	change, err := AnalyzeImpact(parsePolicyImpactProject(t, oldPackage), parsePolicyImpactProject(t, newPackage))
	if err != nil {
		t.Fatal(err)
	}
	if len(change.PolicyChanges) != 1 {
		t.Fatalf("package update should produce one direct failing policy result: %+v", change.PolicyChanges)
	}
	delta := change.PolicyChanges[0]
	if delta.Base.DomainInput == nil || delta.Base.DomainInput.Package != "architecture" || delta.Base.DomainInput.PackageVersion != "1.0.0" || delta.Candidate.DomainInput == nil || delta.Candidate.DomainInput.PackageVersion != "2.0.0" {
		t.Fatalf("package Domain input provenance is incomplete: %+v", delta)
	}
	if delta.Base.PackagePin == nil || delta.Candidate.PackagePin == nil || delta.Base.PackagePin.Source != "fixture:architecture@v1" || delta.Candidate.PackagePin.Source != "fixture:architecture@v2" || delta.Base.PackagePin.SHA256 == delta.Candidate.PackagePin.SHA256 {
		t.Fatalf("exact package pins are missing from the policy delta sides: %+v", delta)
	}
}

func TestAnalyzeImpactStatusPreservingSubjectDigestChange(t *testing.T) {
	constraint := "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	base := policyImpactSnapshot(t, constraint, false, true, false)
	candidate := clonePolicySnapshot(base)
	path := "resources/orders.yaml"
	data := string(candidate.Files[path])
	data = strings.Replace(data, "metadata:\n  name: orders\n  namespace: engineering\n", "metadata:\n  name: orders\n  namespace: engineering\n  labels: {reviewed: yes}\n", 1)
	candidate.Files[path] = []byte(data)
	impact, err := AnalyzeImpact(parsePolicyImpactProject(t, base), parsePolicyImpactProject(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.PolicyChanges) != 1 || impact.PolicyChanges[0].Base.Status != "passed" || impact.PolicyChanges[0].Candidate.Status != "passed" {
		t.Fatalf("status-preserving subject digest change was omitted: %+v", impact.PolicyChanges)
	}
	if impact.PolicyChanges[0].Base.Result.SubjectDigest == impact.PolicyChanges[0].Candidate.Result.SubjectDigest {
		t.Fatalf("changed subject digest was not retained: %+v", impact.PolicyChanges[0])
	}
}

func TestAnalyzeImpactDistinguishesRemovedConstraintAndAbsentSubject(t *testing.T) {
	constraint := "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	base := policyImpactSnapshot(t, constraint, false, true, false)
	removedConstraint := clonePolicySnapshot(base)
	domainPath := "domains/engineering.yaml"
	domain := string(removedConstraint.Files[domainPath])
	if index := strings.Index(domain, "  constraints:"); index >= 0 {
		domain = domain[:index]
	}
	removedConstraint.Files[domainPath] = []byte(domain)
	constraintChange, err := AnalyzeImpact(parsePolicyImpactProject(t, base), parsePolicyImpactProject(t, removedConstraint))
	if err != nil {
		t.Fatal(err)
	}
	if len(constraintChange.PolicyChanges) != 2 {
		t.Fatalf("removing a selected per-resource policy should record each result removal: %+v", constraintChange.PolicyChanges)
	}
	for _, change := range constraintChange.PolicyChanges {
		if change.Base.Status != "passed" || change.Candidate.Status != "not-applicable" || change.Candidate.AbsenceReason != "constraint-not-defined" || change.Candidate.DomainInput == nil || change.Candidate.Definition != nil {
			t.Fatalf("constraint removal should retain candidate Domain provenance and explicit absence: %+v", change)
		}
	}

	absentSubject := clonePolicySnapshot(base)
	delete(absentSubject.Files, "resources/orders.yaml")
	delete(absentSubject.Files, "resources/add-order.skill.yaml")
	subjectChange, err := AnalyzeImpact(parsePolicyImpactProject(t, base), parsePolicyImpactProject(t, absentSubject))
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range subjectChange.PolicyChanges {
		if strings.HasSuffix(change.Subject, "/Module/orders") {
			if change.Candidate.Status != "not-applicable" || change.Candidate.AbsenceReason != "subject-absent" {
				t.Fatalf("resource removal must have its own absence reason: %+v", change)
			}
			return
		}
	}
	t.Fatalf("removed subject did not produce a delta: %+v", subjectChange.PolicyChanges)
}

func TestAnalyzeImpactCollectionDeltaHasNoInventedSubject(t *testing.T) {
	base := policyImpactSnapshot(t, "- name: module-inventory-count\n  select: {kind: Module}\n  assert: {op: count, min: 1}", false, true, false)
	candidate := clonePolicySnapshot(base)
	domainPath := "domains/engineering.yaml"
	candidate.Files[domainPath] = []byte(strings.Replace(string(candidate.Files[domainPath]), "min: 1", "min: 3", 1))
	impact, err := AnalyzeImpact(parsePolicyImpactProject(t, base), parsePolicyImpactProject(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.PolicyChanges) != 1 || impact.PolicyChanges[0].Subject != "" || impact.PolicyChanges[0].Base.Status != "passed" || impact.PolicyChanges[0].Candidate.Status != "failed" {
		t.Fatalf("collection result should remain a subjectless delta: %+v", impact.PolicyChanges)
	}
	if len(impact.DirectPolicySubjects) != 0 || impact.DirectPolicySubjectCount == nil || *impact.DirectPolicySubjectCount != 0 {
		t.Fatalf("collection-level transition must not invent direct resource subjects: %+v", impact)
	}
}

func TestAnalyzeImpactExplainsPerResourceCollectionResultShapeChanges(t *testing.T) {
	perResource := "- name: changing-scope\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	collection := "- name: changing-scope\n  select: {kind: Module}\n  assert: {op: count, scope: selection, min: 1}"
	perSnapshot := policyImpactSnapshot(t, perResource, false, true, false)
	collectionSnapshot := clonePolicySnapshot(perSnapshot)
	domain := string(collectionSnapshot.Files["domains/engineering.yaml"])
	if index := strings.Index(domain, "  constraints:"); index >= 0 {
		domain = domain[:index]
	}
	collectionSnapshot.Files["domains/engineering.yaml"] = []byte(domain + "  constraints:\n" + indentLines(collection, 4) + "\n")
	perToCollection, err := AnalyzeImpact(parsePolicyImpactProject(t, perSnapshot), parsePolicyImpactProject(t, collectionSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	if len(perToCollection.PolicyChanges) != 3 {
		t.Fatalf("expected two removed per-resource records and one added collection record: %+v", perToCollection.PolicyChanges)
	}
	collectionRecords, subjectRecords := 0, 0
	for _, change := range perToCollection.PolicyChanges {
		if change.Subject == "" {
			collectionRecords++
			if change.Base.Status != "not-applicable" || change.Base.AbsenceReason != "result-scope-changed" || change.Candidate.Status != "passed" {
				t.Fatalf("new collection result should explain old result scope: %+v", change)
			}
		} else {
			subjectRecords++
			if change.Base.Status != "passed" || change.Candidate.Status != "not-applicable" || change.Candidate.AbsenceReason != "result-scope-changed" {
				t.Fatalf("removed per-resource result should explain new result scope: %+v", change)
			}
		}
	}
	if collectionRecords != 1 || subjectRecords != 2 || perToCollection.DirectPolicySubjectCount == nil || *perToCollection.DirectPolicySubjectCount != 2 {
		t.Fatalf("result scopes or direct subjects are wrong: %+v", perToCollection)
	}

	collectionToPer, err := AnalyzeImpact(parsePolicyImpactProject(t, collectionSnapshot), parsePolicyImpactProject(t, perSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	if len(collectionToPer.PolicyChanges) != 3 {
		t.Fatalf("reverse scope transition should also produce three records: %+v", collectionToPer.PolicyChanges)
	}
	for _, change := range collectionToPer.PolicyChanges {
		if change.Subject == "" {
			if change.Base.Status != "passed" || change.Candidate.Status != "not-applicable" || change.Candidate.AbsenceReason != "result-scope-changed" {
				t.Fatalf("removed collection result lost its absence reason: %+v", change)
			}
		} else if change.Base.Status != "not-applicable" || change.Base.AbsenceReason != "result-scope-changed" || change.Candidate.Status != "passed" {
			t.Fatalf("added per-resource result lost its absence reason: %+v", change)
		}
	}
}

func TestAnalyzeImpactReportsPolicyStatusTransitions(t *testing.T) {
	constraint := "- name: module-intent\n  select: {kind: Module, labels: {governed: yes}}\n  assert: {op: equal, field: intent, value: expected}"
	passed := policyImpactSnapshot(t, constraint, false, true, true)
	failed := clonePolicySnapshot(passed)
	failed.Files["resources/orders.yaml"] = []byte(strings.Replace(string(failed.Files["resources/orders.yaml"]), "intent: expected", "intent: changed", 1))
	waived := clonePolicySnapshot(failed)
	baseProject := parsePolicyImpactProject(t, passed)
	failedProject := parsePolicyImpactProject(t, failed)
	var finding corePolicyResult
	for _, result := range failedProject.Graph.PolicyResults {
		if result.Status == "failed" && strings.HasSuffix(result.Subject, "/Module/orders") {
			finding = corePolicyResult{api: result.APIVersion, constraint: result.Constraint, subject: result.Subject, constraintDigest: result.ConstraintDigest, subjectDigest: result.SubjectDigest}
		}
	}
	if finding.subject == "" {
		t.Fatalf("fixture has no failed policy result: %+v", failedProject.Graph.PolicyResults)
	}
	config := string(waived.Files["markitect.yaml"])
	config = strings.Replace(config, "spec:\n", "spec:\n  policyExceptions:\n    - name: temporary-orders-waiver\n      apiVersion: "+finding.api+"\n      constraint: "+finding.constraint+"\n      subject: \""+finding.subject+"\"\n      constraintDigest: "+finding.constraintDigest+"\n      subjectDigest: "+finding.subjectDigest+"\n      rationale: test explicit waiver\n      owner: test owner\n      decision: test decision\n", 1)
	waived.Files["markitect.yaml"] = []byte(config)
	waivedProject := parsePolicyImpactProject(t, waived)
	if len(waivedProject.StructuralDiagnostics()) != 0 {
		t.Fatalf("waiver fixture invalid: %+v", waivedProject.StructuralDiagnostics())
	}

	first, err := AnalyzeImpact(baseProject, failedProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, first, "passed", "failed")
	second, err := AnalyzeImpact(failedProject, waivedProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, second, "failed", "waived")
	third, err := AnalyzeImpact(waivedProject, baseProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, third, "waived", "passed")
	failedToPassed, err := AnalyzeImpact(failedProject, baseProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, failedToPassed, "failed", "passed")
	waivedToFailed, err := AnalyzeImpact(waivedProject, failedProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, waivedToFailed, "waived", "failed")
	unselected := clonePolicySnapshot(failed)
	orders := string(unselected.Files["resources/orders.yaml"])
	orders = strings.Replace(orders, "  labels: {governed: yes}\n", "", 1)
	unselected.Files["resources/orders.yaml"] = []byte(orders)
	unselectedProject := parsePolicyImpactProject(t, unselected)
	fourth, err := AnalyzeImpact(failedProject, unselectedProject)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicyTransition(t, fourth, "failed", "not-applicable")
	if fourth.PolicyChanges[0].Candidate.AbsenceReason != "not-selected" {
		t.Fatalf("selector exit should explain why no result exists: %+v", fourth.PolicyChanges)
	}
}

func packagedPolicySnapshot(t *testing.T, version, source, constraints string) *snapshot.Snapshot {
	t.Helper()
	domain := `apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: architecture}
spec:
  apiVersion: architecture.example.org/v1
  kinds:
    Module:
      required: [intent]
      properties:
        intent: {type: string}
`
	if strings.TrimSpace(constraints) != "" {
		domain += "  constraints:\n" + indentLines(strings.TrimSpace(constraints), 4) + "\n"
	}
	manifest := `apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: architecture}
spec:
  version: ` + version + `
  domains: [domains/architecture.yaml]
  areas: [{name: architecture, path: resources}]
  exports: []
`
	contract := `apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata: {name: architecture-contract, namespace: architecture}
spec: {text: Packaged engineering contract.}
`
	archive, err := contentpackage.Build(map[string][]byte{
		contentpackage.ManifestName: []byte(manifest),
		"domains/architecture.yaml": []byte(domain),
		"resources/contract.yaml":   []byte(contract),
	})
	if err != nil {
		t.Fatal(err)
	}
	pin := core.PackagePin{Name: "architecture", Version: version, Source: source, Archive: "packages/architecture.zip", SHA256: strings.TrimPrefix(Hash(archive), "sha256:")}
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "package-impact"}, Spec: core.Spec{
		Domains:  []string{"package:architecture/domains/architecture.yaml"},
		Areas:    []core.Area{{Name: "engineering", Path: "resources"}},
		Packages: []core.PackagePin{pin},
	}}
	module := `apiVersion: architecture.example.org/v1
kind: Module
metadata: {name: orders, namespace: engineering}
spec: {intent: changed}
`
	projectBytes, err := format.Encode(project)
	if err != nil {
		t.Fatal(err)
	}
	return &snapshot.Snapshot{ID: "package-" + version, Provisional: true, Files: map[string][]byte{
		"markitect.yaml":        projectBytes,
		pin.Archive:             archive,
		"resources/orders.yaml": []byte(module),
	}}
}

func indentLines(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	return prefix + strings.ReplaceAll(text, "\n", "\n"+prefix)
}

func TestAnalyzeImpactBlocksStructuralFailuresOnEitherSide(t *testing.T) {
	valid := policyImpactSnapshot(t, "", false, true, false)
	badCandidate := clonePolicySnapshot(valid)
	badCandidate.Files["resources/orders.yaml"] = []byte(strings.Replace(string(badCandidate.Files["resources/orders.yaml"]), "kind: Core", "kind: MissingKind", 1))
	if _, err := AnalyzeImpact(parsePolicyImpactProject(t, valid), parsePolicyImpactProject(t, badCandidate)); err == nil {
		t.Fatal("structural candidate failure must block diagnostic impact")
	}
	badBase := clonePolicySnapshot(badCandidate)
	if _, err := AnalyzeImpact(parsePolicyImpactProject(t, badBase), parsePolicyImpactProject(t, valid)); err == nil {
		t.Fatal("structural base failure must block diagnostic impact")
	}
}

func assertPolicyTransition(t *testing.T, impact *Impact, from, to string) {
	t.Helper()
	for _, change := range impact.PolicyChanges {
		if strings.HasSuffix(change.Subject, "/Module/orders") {
			if change.Base.Status != from || change.Candidate.Status != to {
				t.Fatalf("wanted %s -> %s, got %+v", from, to, change)
			}
			return
		}
	}
	t.Fatalf("no transition for orders: %+v", impact.PolicyChanges)
}

// Keep the small amount of exception setup above independent from test-only
// struct imports in this file.
type corePolicyResult struct{ api, constraint, subject, constraintDigest, subjectDigest string }
