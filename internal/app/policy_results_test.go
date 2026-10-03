package app

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestPolicyExceptionRemainsVisibleInModelAndSelectedContext(t *testing.T) {
	s := &snapshot.Snapshot{ID: "policy-fixture", Files: map[string][]byte{
		"markitect.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: policy-fixture}
spec:
  areas: [{name: engineering, path: resources}]
  domains: [domain.yaml]
`),
		"domain.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: engineering}
spec:
  apiVersion: policy.example.org/v1
  kinds:
    Module:
      properties: {intent: {type: string}}
      required: [intent]
  constraints:
    - name: approved-intent
      select: {kind: Module}
      assert: {op: equal, field: intent, value: approved}
`),
		"resources/legacy.yaml":  []byte("apiVersion: policy.example.org/v1\nkind: Module\nmetadata: {name: legacy, namespace: engineering}\nspec: {intent: legacy}\n"),
		"resources/current.yaml": []byte("apiVersion: policy.example.org/v1\nkind: Module\nmetadata: {name: current, namespace: engineering}\nspec: {intent: approved}\n"),
	}}
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	var failure core.PolicyResult
	for _, result := range p.Graph.PolicyResults {
		if result.Status == "failed" {
			failure = result
		}
	}
	if failure.Subject == "" {
		t.Fatalf("missing subject-bound failure: %+v", p.Graph.PolicyResults)
	}
	p.Graph.Project.Spec.PolicyExceptions = []core.PolicyException{{
		Name: "reviewed-legacy", APIVersion: failure.APIVersion, Constraint: failure.Constraint,
		Subject: failure.Subject, ConstraintDigest: failure.ConstraintDigest, SubjectDigest: failure.SubjectDigest,
		Rationale: "Synthetic exercise: migrate in a separate change.", Owner: "fixture-owner",
		Decision: "fixture-decision; does not authenticate a human reviewer",
	}}
	s.Files["markitect.yaml"], err = format.Encode(p.Graph.Project)
	if err != nil {
		t.Fatal(err)
	}
	p, err = Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("bound exception did not validate: %+v", p.Diagnostics)
	}
	model, err := CompileModel(p)
	if err != nil || model.ValidationStatus != "passed" {
		t.Fatalf("exception model: %v, %+v", err, model)
	}
	if model.StructuralStatus != "passed" || model.PolicyStatus != "waived" {
		t.Fatalf("explicit waiver did not remain visible on the policy status axis: %+v", model)
	}
	if len(model.PolicyResults) != 2 {
		t.Fatalf("model lost policy outcomes: %+v", model.PolicyResults)
	}
	ctx, err := CompileContext(p, failure.Subject, "fixture", "sha256:fixture-tool")
	if err != nil || len(ctx.PolicyResults) != 1 {
		t.Fatalf("selected context outcomes: %v, %+v", err, ctx)
	}
	outcome := ctx.PolicyResults[0]
	if outcome.Status != "waived" || outcome.ExceptionName != "reviewed-legacy" || outcome.SubjectDigest != failure.SubjectDigest || outcome.Message == "" {
		t.Fatalf("waiver hides its original finding: %+v", outcome)
	}
	encoded, err := YAML(ctx)
	if err != nil || !strings.Contains(string(encoded), "fixture-decision") || !strings.Contains(string(encoded), "intent: legacy") {
		t.Fatalf("context lost the declared intent or decision: %v\n%s", err, encoded)
	}
	// Returned model outcomes must not share mutable slice storage with the graph.
	model.PolicyResults[0].Status = "changed by caller"
	if p.Graph.PolicyResults[0].Status == "changed by caller" {
		t.Fatal("model caller changed the compiler's policy results")
	}
}

func TestSemanticModelSeparatesStructuralAndPolicyStatus(t *testing.T) {
	s := &snapshot.Snapshot{ID: "policy-analysis", Files: map[string][]byte{
		"markitect.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: policy-analysis}
spec:
  areas: [{name: engineering, path: resources}]
  domains: [domain.yaml]
`),
		"domain.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: engineering}
spec:
  apiVersion: policy.example.org/v1
  kinds:
    Module:
      properties: {intent: {type: string}}
  constraints:
    - name: path
      select: {kind: Module}
      assert: {op: equal, field: intent, value: approved}
`),
		"resources/module.yaml": []byte("apiVersion: policy.example.org/v1\nkind: Module\nmetadata: {name: legacy, namespace: engineering}\nspec: {intent: legacy}\n"),
	}}
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.StructuralDiagnostics(); len(got) != 0 {
		t.Fatalf("ordinary policy failure became structural: %+v", got)
	}
	model, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	if model.ValidationStatus != "failed" || model.StructuralStatus != "passed" || model.PolicyStatus != "failed" {
		t.Fatalf("model did not expose independent validation axes: %+v", model)
	}
	if len(model.Diagnostics) != 1 || model.Diagnostics[0].PolicyResult == nil || model.Diagnostics[0].PolicyResult.Constraint != "path" {
		t.Fatalf("model did not retain policy result provenance: %+v", model.Diagnostics)
	}
	model.Diagnostics[0].PolicyResult.Subject = "mutated"
	if p.Diagnostics[0].PolicyResult.Subject == "mutated" {
		t.Fatal("model diagnostic result reference aliases the parsed project")
	}

	// A code that looks like a policy diagnostic remains structural without a
	// generated graph diagnostic, even if a caller supplies a matching identity.
	failed := p.Graph.PolicyResults[0]
	ref := &core.PolicyResultRef{APIVersion: failed.APIVersion, Constraint: failed.Constraint, Subject: failed.Subject}
	p.Diagnostics = append(p.Diagnostics,
		core.Diagnostic{Code: "constraint.path", Message: "unrelated structural finding"},
		core.Diagnostic{Code: "constraint.path", Message: "forged identity", PolicyResult: ref},
	)
	if got := p.StructuralDiagnostics(); len(got) != 2 || got[0].Message != "unrelated structural finding" || got[1].Message != "forged identity" {
		t.Fatalf("classification accepted a diagnostic that was not produced by graph policy evaluation: %+v", got)
	}
}
