package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
)

func TestAnalyzeContextAndImpactBlockStructuralGraphFailures(t *testing.T) {
	cases := []struct {
		name     string
		wantCode string
		mutate   func(*testing.T, *snapshot.Snapshot)
	}{
		{
			name:     "unresolved typed relation reference",
			wantCode: "reference.missing",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				path := "resources/orders.yaml"
				candidate.Files[path] = []byte(strings.Replace(string(candidate.Files[path]), "name: platform", "name: missing-platform", 1))
			},
		},
		{
			name:     "wrong relation target kind",
			wantCode: "reference.kind",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				domain := "domains/engineering.yaml"
				candidate.Files[domain] = []byte(strings.Replace(string(candidate.Files[domain]), "targetKinds: [Core, Module]", "targetKinds: [Core]", 1))
				path := "resources/orders.yaml"
				candidate.Files[path] = []byte(strings.Replace(string(candidate.Files[path]), "dependsOn: [{kind: Core, name: platform, namespace: engineering}]", "dependsOn: [{kind: Module, name: other, namespace: engineering}]", 1))
			},
		},
		{
			name:     "relation cardinality",
			wantCode: "relation.max-targets",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				domain := "domains/engineering.yaml"
				candidate.Files[domain] = []byte(strings.Replace(string(candidate.Files[domain]), "      invalidate: false\n  constraints:", "      invalidate: false\n      maxTargets: 1\n  constraints:", 1))
				path := "resources/orders.yaml"
				candidate.Files[path] = []byte(strings.Replace(string(candidate.Files[path]), "dependsOn: [{kind: Core, name: platform, namespace: engineering}]", "dependsOn: [{kind: Core, name: platform, namespace: engineering}, {kind: Module, name: other, namespace: engineering}]", 1))
			},
		},
		{
			name:     "acyclic relation cycle",
			wantCode: "relation.cycle",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				domain := "domains/engineering.yaml"
				candidate.Files[domain] = []byte(strings.Replace(string(candidate.Files[domain]), "      invalidate: false\n  constraints:", "      invalidate: false\n      acyclic: true\n  constraints:", 1))
				orders := "resources/orders.yaml"
				candidate.Files[orders] = []byte(strings.Replace(string(candidate.Files[orders]), "dependsOn: [{kind: Core, name: platform, namespace: engineering}]", "dependsOn: [{kind: Module, name: other, namespace: engineering}]", 1))
				other := "resources/other.yaml"
				candidate.Files[other] = []byte(strings.Replace(string(candidate.Files[other]), "dependsOn: [{kind: Core, name: platform, namespace: engineering}]", "dependsOn: [{kind: Module, name: orders, namespace: engineering}]", 1))
			},
		},
		{
			name:     "malformed resource",
			wantCode: "parse",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				path := "resources/orders.yaml"
				candidate.Files[path] = []byte(strings.Replace(string(candidate.Files[path]), "  intent: expected\n", "  intent: expected\n  undeclared: true\n", 1))
			},
		},
		{
			name:     "stale exception metadata",
			wantCode: "policy.exception.stale",
			mutate: func(t *testing.T, candidate *snapshot.Snapshot) {
				path := "resources/orders.yaml"
				candidate.Files[path] = []byte(strings.Replace(string(candidate.Files[path]), "intent: expected", "intent: changed", 1))
				p := parsePolicyImpactProject(t, candidate)
				finding := policyFindingFor(t, p, "orders")
				config := string(candidate.Files["markitect.yaml"])
				config = strings.Replace(config, "spec:\n", "spec:\n  policyExceptions:\n    - name: stale-orders\n      apiVersion: "+finding.api+"\n      constraint: "+finding.constraint+"\n      subject: \""+finding.subject+"\"\n      constraintDigest: sha256:"+strings.Repeat("0", 64)+"\n      subjectDigest: "+finding.subjectDigest+"\n      rationale: migration pending\n      owner: architecture\n      decision: accepted for the fixture\n", 1)
				candidate.Files["markitect.yaml"] = []byte(config)
			},
		},
	}

	policy := "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}"
	base := parsePolicyImpactProject(t, policyImpactSnapshot(t, policy, false, false, false))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidateSnapshot := clonePolicySnapshot(policyImpactSnapshot(t, policy, false, false, false))
			tc.mutate(t, candidateSnapshot)
			candidate := parsePolicyImpactProject(t, candidateSnapshot)
			if len(candidate.StructuralDiagnostics()) == 0 {
				t.Fatalf("fixture did not produce structural diagnostics: %+v", candidate.Diagnostics)
			}
			foundCode := false
			for _, diagnostic := range candidate.StructuralDiagnostics() {
				if diagnostic.Code == tc.wantCode {
					foundCode = true
					break
				}
			}
			if !foundCode {
				t.Fatalf("fixture did not exercise %s: %+v", tc.wantCode, candidate.StructuralDiagnostics())
			}
			entry := moduleKey(t, candidate, "other")
			if _, err := AnalyzeContext(candidate, entry, "test"); err == nil {
				t.Fatal("AnalyzeContext proceeded despite structural diagnostics")
			}
			if _, err := AnalyzeImpact(base, candidate); err == nil {
				t.Fatal("AnalyzeImpact accepted structurally invalid candidate")
			}
			if _, err := AnalyzeImpact(candidate, base); err == nil {
				t.Fatal("AnalyzeImpact accepted structurally invalid base")
			}
		})
	}
}

func TestAnalyzeContextAndImpactRejectMalformedDomainAndInvalidException(t *testing.T) {
	t.Run("malformed Domain", func(t *testing.T) {
		candidate := policyImpactSnapshot(t, "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}", false, false, false)
		domain := "domains/engineering.yaml"
		candidate.Files[domain] = []byte(strings.Replace(string(candidate.Files[domain]), "op: equal", "op: arbitrary", 1))
		if _, err := Parse(candidate); err == nil {
			t.Fatal("malformed Domain unexpectedly produced an analyzable Project")
		}
		if _, err := AnalyzeContext(nil, "", "test"); err == nil {
			t.Fatal("AnalyzeContext accepted the missing parsed model")
		}
		if _, err := AnalyzeImpact(nil, nil); err == nil {
			t.Fatal("AnalyzeImpact accepted missing parsed models")
		}
	})

	t.Run("invalid exception metadata", func(t *testing.T) {
		candidate := policyImpactSnapshot(t, "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}", false, false, false)
		config := string(candidate.Files["markitect.yaml"])
		config = strings.Replace(config, "spec:\n", "spec:\n  policyExceptions:\n    - name: Invalid_Name\n      apiVersion: engineering.markitect.org/v1alpha1\n      constraint: module-intent\n      subject: \"engineering/engineering.markitect.org/v1alpha1/Module/orders\"\n      constraintDigest: sha256:"+strings.Repeat("0", 64)+"\n      subjectDigest: sha256:"+strings.Repeat("0", 64)+"\n      rationale: invalid fixture\n      owner: architecture\n      decision: accepted for fixture\n", 1)
		candidate.Files["markitect.yaml"] = []byte(config)
		if _, err := Parse(candidate); err == nil {
			t.Fatal("invalid exception metadata unexpectedly produced an analyzable Project")
		}
	})
}

func moduleKey(t *testing.T, p *Project, name string) string {
	t.Helper()
	for key, resource := range p.Graph.Resources {
		if resource.Kind == "Module" && resource.Metadata.Name == name {
			return key
		}
	}
	t.Fatalf("missing Module %s", name)
	return ""
}

type policyFinding struct {
	api, constraint, subject, constraintDigest, subjectDigest string
}

func policyFindingFor(t *testing.T, p *Project, name string) policyFinding {
	t.Helper()
	for _, result := range p.Graph.Core.PolicyResults {
		if result.Status == "failed" && strings.HasSuffix(result.Subject, "/Module/"+name) {
			return policyFinding{api: result.APIVersion, constraint: result.Constraint, subject: result.Subject, constraintDigest: result.ConstraintDigest, subjectDigest: result.SubjectDigest}
		}
	}
	t.Fatalf("missing failed policy for Module %s: %+v", name, p.Graph.Core.PolicyResults)
	return policyFinding{}
}

func TestAnalyzeContextReportsGlobalFailuresWithoutExpandingSelectedClosure(t *testing.T) {
	policy := "- name: selected-modules-require-intent\n  select: {kind: Module, labels: {governed: yes}}\n  assert: {op: equal, field: intent, value: expected}"
	s := policyImpactSnapshot(t, policy, false, false, true)
	otherPath := "resources/other.yaml"
	other := string(s.Files[otherPath])
	other = strings.Replace(other, "metadata: {name: other, namespace: engineering}", "metadata: {name: other, namespace: engineering, labels: {governed: yes}}", 1)
	other = strings.Replace(other, "intent: expected", "intent: changed", 1)
	s.Files[otherPath] = []byte(other)
	ordersPath := "resources/orders.yaml"
	s.Files[ordersPath] = []byte(strings.Replace(string(s.Files[ordersPath]), "intent: expected", "intent: changed", 1))
	p := parsePolicyImpactProject(t, s)
	entry := moduleKey(t, p, "orders")
	model, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := AnalyzeContext(p, entry, "test")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Analysis.Candidate.FailedPolicyResults != 2 || ctx.Analysis.Candidate.ModelDigest != model.ModelDigest {
		t.Fatalf("analysis header did not bind full-candidate status to the same model: %+v model=%s", ctx.Analysis, model.ModelDigest)
	}
	if len(ctx.PolicyResults) != 1 || ctx.PolicyResults[0].Subject != entry || ctx.PolicyResults[0].Status != "failed" {
		t.Fatalf("context should include only the selected closure's policy result: %+v", ctx.PolicyResults)
	}
	for _, input := range ctx.Inputs {
		if input.Key == moduleKey(t, p, "other") {
			t.Fatalf("unrelated failed subject entered the selected context: %+v", input)
		}
	}
}

func TestAnalyzeContextMarksWaivedStateAndUsesDistinctDigest(t *testing.T) {
	s := policyImpactSnapshot(t, "- name: module-intent\n  select: {kind: Module}\n  assert: {op: equal, field: intent, value: expected}", false, false, false)
	s.Files["resources/orders.yaml"] = []byte(strings.Replace(string(s.Files["resources/orders.yaml"]), "intent: expected", "intent: changed", 1))
	failing := parsePolicyImpactProject(t, s)
	finding := policyFindingFor(t, failing, "orders")
	config := string(s.Files["markitect.yaml"])
	config = strings.Replace(config, "spec:\n", "spec:\n  policyExceptions:\n    - name: orders-waiver\n      apiVersion: "+finding.api+"\n      constraint: "+finding.constraint+"\n      subject: \""+finding.subject+"\"\n      constraintDigest: "+finding.constraintDigest+"\n      subjectDigest: "+finding.subjectDigest+"\n      rationale: explicit test waiver\n      owner: architecture\n      decision: accepted for test\n", 1)
	s.Files["markitect.yaml"] = []byte(config)
	waived := parsePolicyImpactProject(t, s)
	entry := moduleKey(t, waived, "orders")
	ordinary, err := CompileContext(waived, entry, "test")
	if err != nil {
		t.Fatal(err)
	}
	analysis, err := AnalyzeContext(waived, entry, "test")
	if err != nil {
		t.Fatal(err)
	}
	model, err := CompileModel(waived)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Analysis == nil || analysis.Analysis.Candidate.PolicyStatus != "waived" || analysis.Analysis.Candidate.FailedPolicyResults != 0 || analysis.Analysis.Candidate.ModelDigest != model.ModelDigest {
		t.Fatalf("explicit waiver was not accurately summarized: %+v", analysis.Analysis)
	}
	if analysis.Digest == ordinary.Digest {
		t.Fatal("diagnostic analysis digest collided with ordinary context digest")
	}
}
