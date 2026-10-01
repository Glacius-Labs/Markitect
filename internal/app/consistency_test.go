package app

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestFunctionalAssertionsRequireOptInAndExactSource(t *testing.T) {
	project := &core.Resource{Kind: "Project", Spec: core.Spec{Consistency: &core.Consistency{FunctionalPredicates: []string{"owner"}}}}
	first := &core.Resource{Kind: "Rule", Metadata: core.Metadata{Name: "operations", Namespace: "sample"}, Path: "docs/operations.yaml", Line: 2, Spec: core.Spec{Files: []string{"docs/operations.md"}, Assertions: []core.Assertion{{Subject: "release-approval", Predicate: "owner", Value: "operations", Source: "docs/operations.md", Quote: "Operations owns approval."}}}}
	second := &core.Resource{Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "sample"}, Path: "docs/review.yaml", Line: 3, Spec: core.Spec{Files: []string{"docs/review.md"}, Assertions: []core.Assertion{{Subject: "release-approval", Predicate: "owner", Value: "review", Source: "docs/review.md", Quote: "Review owns approval."}}}}
	p := &Project{Snapshot: &source.Snapshot{Files: map[string][]byte{"docs/operations.md": []byte("# Contract\nOperations owns approval.\n"), "docs/review.md": []byte("Review owns approval.\n")}}, Graph: &core.Graph{Project: project, Resources: map[string]*core.Resource{"a": first, "b": second}}}
	findings := CheckConsistency(p)
	if len(findings) != 1 || findings[0].Code != "consistency.conflict" || findings[0].Path != "docs/review.md" || findings[0].Line != 1 || !strings.Contains(findings[0].Message, "docs/operations.md:2") {
		t.Fatalf("expected traceable conflict, got %#v", findings)
	}
	project.Spec.Consistency = nil
	if got := CheckConsistency(p); len(got) != 0 {
		t.Fatalf("non-opted-in project reported %#v", got)
	}
	project.Spec.Consistency = &core.Consistency{FunctionalPredicates: []string{"owner"}}
	second.Spec.Assertions[0].Quote = "Unwritten assertion."
	findings = CheckConsistency(p)
	if len(findings) != 1 || findings[0].Code != "consistency.quote" {
		t.Fatalf("missing source quote was accepted: %#v", findings)
	}
}
