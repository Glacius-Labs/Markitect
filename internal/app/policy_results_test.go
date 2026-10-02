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
