package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func TestCheckReportsReviewedDeviationWithoutHidingViolation(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "markitect.yaml", []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: policy-report}
spec:
  areas: [{name: engineering, path: resources}]
  domains: [domain.yaml]
`))
	writeRepoFile(t, root, "domain.yaml", []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: report}
spec:
  apiVersion: report.example.org/v1
  kinds:
    Module:
      properties: {state: {type: string}}
      required: [state]
  constraints:
    - name: active-module
      select: {kind: Module}
      assert: {op: equal, field: state, value: active}
`))
	writeRepoFile(t, root, "resources/legacy.yaml", []byte("apiVersion: report.example.org/v1\nkind: Module\nmetadata: {name: legacy, namespace: engineering}\nspec: {state: legacy}\n"))
	code, out, errout := invoke("check", "--repo", root)
	if code != 1 {
		t.Fatalf("unwaived check exit=%d: %s\n%s", code, out, errout)
	}
	failed := decodeYAML[report](t, out)
	if len(failed.PolicyResults) != 1 || failed.PolicyResults[0].Status != "failed" {
		t.Fatalf("missing concrete failure: %+v", failed)
	}
	result := failed.PolicyResults[0]
	p, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Graph.Project.Spec.PolicyExceptions = []core.PolicyException{{
		Name: "legacy-migration", APIVersion: result.APIVersion, Constraint: result.Constraint,
		Subject: result.Subject, ConstraintDigest: result.ConstraintDigest, SubjectDigest: result.SubjectDigest,
		Rationale: "Synthetic migration exercise.", Owner: "fixture-owner", Decision: "fixture-decision",
	}}
	config, err := authoring.Encode(p.Graph.Project)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, root, "markitect.yaml", config)
	code, out, errout = invoke("check", "--repo", root)
	if code != 0 {
		t.Fatalf("bound deviation check exit=%d: %s\n%s", code, out, errout)
	}
	waived := decodeYAML[report](t, out)
	if waived.Status != "passed" || len(waived.PolicyResults) != 1 || waived.PolicyResults[0].Status != "waived" || waived.PolicyResults[0].Message != result.Message {
		t.Fatalf("check hides the original violation: %+v", waived)
	}
}
