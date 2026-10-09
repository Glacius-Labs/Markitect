package core

import (
	"strings"
	"testing"
)

func policyTestDomain() DomainDefinition {
	return DomainDefinition{
		Name: "policy-tests", APIVersion: "policy.tests.example/v1",
		Kinds: map[string]KindDefinition{
			"Module": {Properties: map[string]PropertyDefinition{
				"intent":     {Type: "string"},
				"code":       {Type: "string"},
				"validators": {Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: "Core"}},
			}},
			"Core": {Properties: map[string]PropertyDefinition{"purpose": {Type: "string"}}},
		},
		Relations: map[string]RelationDefinition{
			"validators": {Field: "validators", SourceKinds: []string{"Module"}, TargetKinds: []string{"Core"}},
		},
		Constraints: []ConstraintDefinition{
			{Name: "module-intent", Select: ResourceSelector{Kind: "Module"}, Assert: ConstraintAssertion{Op: "present", Field: "intent"}},
			{Name: "module-validator-count", Select: ResourceSelector{Kind: "Module"}, Assert: ConstraintAssertion{Op: "count", Scope: "resource", Relation: "validators", Min: intPtr(1)}},
			{Name: "module-count", Select: ResourceSelector{Kind: "Module"}, Assert: ConstraintAssertion{Op: "count", Min: intPtr(2)}},
			{Name: "module-codes-unique", Select: ResourceSelector{Kind: "Module"}, Assert: ConstraintAssertion{Op: "unique", Field: "code"}},
		},
	}
}

func policyTestResources() []*Resource {
	domain := "policy.tests.example/v1"
	return []*Resource{
		{APIVersion: domain, Kind: "Module", Metadata: Metadata{Name: "empty", Namespace: "engineering"}, Path: "modules/empty.yaml", Line: 2, Data: map[string]any{"code": "same"}},
		{APIVersion: domain, Kind: "Module", Metadata: Metadata{Name: "full", Namespace: "engineering"}, Path: "modules/full.yaml", Line: 3, Data: map[string]any{"code": "same", "validators": []any{map[string]any{"kind": "Core", "name": "one"}, map[string]any{"kind": "Core", "name": "two"}}}},
		{APIVersion: domain, Kind: "Core", Metadata: Metadata{Name: "one", Namespace: "engineering"}, Data: map[string]any{"purpose": "one"}},
		{APIVersion: domain, Kind: "Core", Metadata: Metadata{Name: "two", Namespace: "engineering"}, Data: map[string]any{"purpose": "two"}},
	}
}

func buildPolicyTestGraph(exceptions []PolicyException, policyDate string, domain DomainDefinition) *Graph {
	registry := NewRegistry()
	if err := registry.AddDomain(domain); err != nil {
		panic(err)
	}
	resources := policyTestResources()
	relationships, diagnostics := ResolveTypedRelationships(resources, registry)
	if len(diagnostics) != 0 {
		panic(diagnostics)
	}
	return BuildNormalized(resources, registry, relationships, exceptions, policyDate, nil)
}

func TestPolicyResultsReportEverySubjectAndSeparateResourceFromSelectionCounts(t *testing.T) {
	domain := policyTestDomain()
	graph := buildPolicyTestGraph(nil, "", domain)
	results := map[string]map[string]PolicyResult{}
	for _, result := range graph.PolicyResults {
		if results[result.Constraint] == nil {
			results[result.Constraint] = map[string]PolicyResult{}
		}
		results[result.Constraint][result.Subject] = result
	}
	for _, subject := range []string{"engineering/policy.tests.example/v1/Module/empty", "engineering/policy.tests.example/v1/Module/full"} {
		if result := results["module-intent"][subject]; result.Status != PolicyFailed || result.SubjectDigest == "" || result.ConstraintDigest == "" {
			t.Fatalf("missing subject-specific finding for %s: %+v", subject, result)
		}
	}
	if result := results["module-validator-count"]["engineering/policy.tests.example/v1/Module/empty"]; result.Status != PolicyFailed {
		t.Fatalf("zero validators did not fail independently: %+v", result)
	}
	if result := results["module-validator-count"]["engineering/policy.tests.example/v1/Module/full"]; result.Status != PolicyPassed {
		t.Fatalf("two validators incorrectly failed per-resource min bound: %+v", result)
	}
	if result := results["module-count"][""]; result.Status != PolicyPassed {
		t.Fatalf("selection count was not a single collection result: %+v", result)
	}
	if result := results["module-codes-unique"][""]; result.Status != PolicyFailed {
		t.Fatalf("unique did not produce a failed collection result: %+v", result)
	}
	if countCode(graph, "constraint.module-intent") != 2 {
		t.Fatalf("expected a diagnostic for each failing subject, got %#v", graph.Diagnostics)
	}
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code != "constraint.module-intent" {
			continue
		}
		if diagnostic.PolicyResult == nil || diagnostic.PolicyResult.APIVersion != domain.APIVersion || diagnostic.PolicyResult.Constraint != "module-intent" || diagnostic.PolicyResult.Subject == "" {
			t.Fatalf("ordinary failed-policy diagnostic lacks its exact result identity: %+v", diagnostic)
		}
	}
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == "constraint.module-validator-count" && diagnostic.PolicyResult == nil {
			t.Fatalf("collection failure diagnostic lacks its empty-subject result identity: %+v", diagnostic)
		}
	}
}

func TestPolicyExceptionWaivesOneExactFindingAndReportsMetadata(t *testing.T) {
	domain := policyTestDomain()
	base := buildPolicyTestGraph(nil, "", domain)
	var target PolicyResult
	for _, result := range base.PolicyResults {
		if result.Constraint == "module-intent" && result.Subject == "engineering/policy.tests.example/v1/Module/empty" {
			target = result
		}
	}
	if target.Status != PolicyFailed {
		t.Fatalf("fixture did not fail: %+v", target)
	}
	exceptions := []PolicyException{{
		Name: "legacy-intent", APIVersion: target.APIVersion, Constraint: target.Constraint, Subject: target.Subject,
		ConstraintDigest: target.ConstraintDigest, SubjectDigest: target.SubjectDigest, Rationale: "Migration is scheduled separately.", Owner: "architecture", Decision: "accepted for this fixed snapshot",
	}}
	graph := buildPolicyTestGraph(exceptions, "", domain)
	statuses := map[string]string{}
	for _, result := range graph.PolicyResults {
		if result.Constraint == "module-intent" {
			statuses[result.Subject] = result.Status
			if result.Subject == target.Subject && (result.ExceptionName != "legacy-intent" || result.Message != target.Message || result.Rationale == "" || result.Owner == "" || result.Decision == "") {
				t.Fatalf("waived result lost original finding or exception metadata: %+v", result)
			}
		}
	}
	if statuses[target.Subject] != PolicyWaived || statuses["engineering/policy.tests.example/v1/Module/full"] != PolicyFailed {
		t.Fatalf("exception did not bind only to one exact subject: %#v", statuses)
	}
	if countCode(graph, "constraint.module-intent") != 1 {
		t.Fatalf("waiver removed a sibling finding or left its finding diagnostic: %#v", graph.Diagnostics)
	}
	graph.EvaluateConstraints(exceptions, "")
	if statusFor(graph, "module-intent", target.Subject) != PolicyWaived || countCode(graph, "constraint.module-intent") != 1 {
		t.Fatalf("reevaluation accumulated stale policy diagnostics: %#v", graph.Diagnostics)
	}
}

func TestPolicyExceptionDigestBindsRelationDefinition(t *testing.T) {
	domain := policyTestDomain()
	module := domain.Kinds["Module"]
	module.Properties["approvers"] = PropertyDefinition{Type: "array", Items: &PropertyDefinition{Type: "ref", RefKind: "Core"}}
	domain.Kinds["Module"] = module
	baseline := buildPolicyTestGraph(nil, "", domain)
	var finding PolicyResult
	for _, result := range baseline.PolicyResults {
		if result.Constraint == "module-validator-count" && result.Subject == "engineering/policy.tests.example/v1/Module/empty" {
			finding = result
		}
	}
	if finding.Status != PolicyFailed {
		t.Fatalf("fixture did not produce a failing resource-count result: %+v", finding)
	}

	// Keep the policy's API, name, selector, assertion, and subject unchanged,
	// but retarget the named relation to a different declared field.
	relation := domain.Relations["validators"]
	relation.Field = "approvers"
	domain.Relations["validators"] = relation
	exceptions := []PolicyException{{
		Name: "old-validator-waiver", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: finding.Subject,
		ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest,
		Rationale: "The previous relation definition was reviewed.", Owner: "architecture", Decision: "time-boxed exception",
	}}
	updated := buildPolicyTestGraph(exceptions, "", domain)
	if countCode(updated, "policy.exception.stale") != 1 {
		t.Fatalf("changing the relation field must stale the exception: %#v", updated.Diagnostics)
	}
	for _, result := range updated.PolicyResults {
		if result.Constraint == finding.Constraint && result.Subject == finding.Subject && result.Status != PolicyFailed {
			t.Fatalf("old exception incorrectly waived changed relation semantics: %+v", result)
		}
	}
}

func TestPolicyExceptionsRejectStaleAndExpiredEvidenceWithoutClock(t *testing.T) {
	domain := policyTestDomain()
	base := buildPolicyTestGraph(nil, "", domain)
	var finding PolicyResult
	for _, result := range base.PolicyResults {
		if result.Constraint == "module-intent" && result.Subject == "engineering/policy.tests.example/v1/Module/empty" {
			finding = result
		}
	}
	makeException := func(subjectDigest string) []PolicyException {
		return []PolicyException{{Name: "legacy-intent", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: finding.Subject,
			ConstraintDigest: finding.ConstraintDigest, SubjectDigest: subjectDigest, Rationale: "R", Owner: "O", Decision: "D", ExpiresOn: "2026-10-02"}}
	}
	stale := buildPolicyTestGraph(makeException("sha256:"+strings.Repeat("0", 64)), "2026-10-01", domain)
	if !hasCode(stale, "policy.exception.stale") || statusFor(stale, finding.Constraint, finding.Subject) != PolicyFailed {
		t.Fatalf("stale subject binding waived a failure: %#v", stale.Diagnostics)
	}
	expired := buildPolicyTestGraph(makeException(finding.SubjectDigest), "2026-10-02", domain)
	if !hasCode(expired, "policy.exception.expired") || statusFor(expired, finding.Constraint, finding.Subject) != PolicyFailed {
		t.Fatalf("expiresOn was not exclusive at the pinned policy date: %#v", expired.Diagnostics)
	}

	changedPolicy := policyTestDomain()
	for i := range changedPolicy.Constraints {
		if changedPolicy.Constraints[i].Name == finding.Constraint {
			changedPolicy.Constraints[i].Assert.Field = "code"
		}
	}
	stalePolicy := buildPolicyTestGraph(makeException(finding.SubjectDigest), "2026-10-01", changedPolicy)
	if !hasCode(stalePolicy, "policy.exception.stale") {
		t.Fatalf("changed normalized constraint did not stale the exception: %#v", stalePolicy.Diagnostics)
	}
}

func TestPolicySubjectDigestExcludesSourceLocationButIncludesPackageIdentity(t *testing.T) {
	resource := &Resource{APIVersion: "policy.tests.example/v1", Kind: "Module", Metadata: Metadata{Name: "orders", Namespace: "engineering"}, Data: map[string]any{"intent": "stable"}, Path: "first.yaml", Line: 1}
	first, err := digestResource(resource, nil)
	if err != nil {
		t.Fatal(err)
	}
	resource.Path, resource.Line = "second.yaml", 50
	second, err := digestResource(resource, nil)
	if err != nil || first != second {
		t.Fatalf("source location affected canonical subject digest: %q != %q (%v)", first, second, err)
	}
	resource.Package = "patterns"
	third, err := digestResource(resource, nil)
	if err != nil || third == second {
		t.Fatalf("package-qualified subject identity was omitted from digest: %q, %q (%v)", second, third, err)
	}
	resource.Package = ""
	resource.APIVersion = "policy.tests.example/v2"
	fourth, err := digestResource(resource, nil)
	if err != nil || fourth == second {
		t.Fatalf("API-qualified resource identity was omitted from digest: %q, %q (%v)", second, fourth, err)
	}
}

func TestPolicyExceptionsRejectWrongAPIVersionAndNoLongerNeededFindings(t *testing.T) {
	domain := policyTestDomain()
	base := buildPolicyTestGraph(nil, "", domain)
	var finding PolicyResult
	for _, result := range base.PolicyResults {
		if result.Constraint == "module-intent" && result.Subject == "engineering/policy.tests.example/v1/Module/empty" {
			finding = result
		}
	}
	wrongAPI := []PolicyException{{
		Name: "wrong-domain", APIVersion: "other.tests.example/v1", Constraint: finding.Constraint, Subject: finding.Subject,
		ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "R", Owner: "O", Decision: "D",
	}}
	if graph := buildPolicyTestGraph(wrongAPI, "", domain); !hasCode(graph, "policy.exception.unknown") || statusFor(graph, finding.Constraint, finding.Subject) != PolicyFailed {
		t.Fatalf("wrong API identity matched an unrelated finding: %#v", graph.Diagnostics)
	}

	unselectedPolicy := policyTestDomain()
	for i := range unselectedPolicy.Constraints {
		if unselectedPolicy.Constraints[i].Name == finding.Constraint {
			unselectedPolicy.Constraints[i].Select.Labels = map[string]string{"scope": "included"}
		}
	}
	unneeded := []PolicyException{{
		Name: "no-longer-selected", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: finding.Subject,
		ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "R", Owner: "O", Decision: "D",
	}}
	if graph := buildPolicyTestGraph(unneeded, "", unselectedPolicy); !hasCode(graph, "policy.exception.stale") {
		t.Fatalf("changed selector did not stale a no-longer-selected exception: %#v", graph.Diagnostics)
	}
}

func TestPolicyExceptionCannotWaiveCollectionUniqueFinding(t *testing.T) {
	domain := policyTestDomain()
	base := buildPolicyTestGraph(nil, "", domain)
	var finding PolicyResult
	for _, result := range base.PolicyResults {
		if result.Constraint == "module-codes-unique" {
			finding = result
		}
	}
	if finding.Status != PolicyFailed || finding.Subject != "" {
		t.Fatalf("fixture did not produce a collection finding: %+v", finding)
	}
	exceptions := []PolicyException{{
		Name: "unique-override", APIVersion: finding.APIVersion, Constraint: finding.Constraint, Subject: "engineering/policy.tests.example/v1/Module/empty",
		ConstraintDigest: finding.ConstraintDigest, SubjectDigest: finding.SubjectDigest, Rationale: "R", Owner: "O", Decision: "D",
	}}
	graph := buildPolicyTestGraph(exceptions, "", domain)
	if !hasCode(graph, "policy.exception.not-waivable") || statusFor(graph, finding.Constraint, "") != PolicyFailed || !hasCode(graph, "constraint."+finding.Constraint) {
		t.Fatalf("collection-wide finding was waived or lost: results=%#v diagnostics=%#v", graph.PolicyResults, graph.Diagnostics)
	}
}

func intPtr(value int) *int { return &value }

func countCode(graph *Graph, code string) int {
	count := 0
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Code == code {
			count++
		}
	}
	return count
}

func statusFor(graph *Graph, constraint, subject string) string {
	for _, result := range graph.PolicyResults {
		if result.Constraint == constraint && result.Subject == subject {
			return result.Status
		}
	}
	return ""
}
