package app

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func TestFunctionalAssertionsRequireOptInAndExactSource(t *testing.T) {
	project := &core.Resource{Kind: "Project", Spec: core.Spec{Consistency: &core.Consistency{FunctionalPredicates: []string{"owner"}}}}
	first := &core.Resource{Kind: "Rule", Metadata: core.Metadata{Name: "operations", Namespace: "sample"}, Path: "docs/operations.yaml", Line: 2, Spec: core.Spec{Files: []string{"docs/operations.md"}, Assertions: []core.Assertion{{Subject: "release-approval", Predicate: "owner", Value: "operations", Source: "docs/operations.md", Quote: "Operations owns approval."}}}}
	second := &core.Resource{Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "sample"}, Path: "docs/review.yaml", Line: 3, Spec: core.Spec{Files: []string{"docs/review.md"}, Assertions: []core.Assertion{{Subject: "release-approval", Predicate: "owner", Value: "review", Source: "docs/review.md", Quote: "Review owns approval."}}}}
	p := &Project{Snapshot: &snapshot.Snapshot{Files: map[string][]byte{"docs/operations.md": []byte("# Contract\nOperations owns approval.\n"), "docs/review.md": []byte("Review owns approval.\n")}}, Graph: &core.Graph{Project: project, Resources: map[string]*core.Resource{"a": first, "b": second}}}
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
	second.Spec.Assertions[0].Quote = "Review owns approval."
	delete(p.Snapshot.Files, "docs/review.md")
	findings = CheckConsistency(p)
	if len(findings) != 1 || findings[0].Code != "consistency.source" {
		t.Fatalf("missing declared source was accepted: %#v", findings)
	}
	p.Snapshot.Files["docs/review.md"] = []byte("Review owns approval.\nReview owns approval.\n")
	findings = CheckConsistency(p)
	if len(findings) != 1 || findings[0].Code != "consistency.quote" {
		t.Fatalf("ambiguous repeated quote was accepted: %#v", findings)
	}
}

func TestFunctionalAssertionsReportEachConflictingOwner(t *testing.T) {
	project := &core.Resource{Kind: "Project", Spec: core.Spec{Consistency: &core.Consistency{FunctionalPredicates: []string{"owner"}}}}
	resources := map[string]*core.Resource{}
	files := map[string][]byte{}
	for _, item := range []struct{ name, value string }{{"first", "operations"}, {"second", "review"}, {"third", "security"}} {
		path := "docs/" + item.name + ".md"
		quote := item.name + " owns approval."
		files[path] = []byte(quote + "\n")
		r := &core.Resource{Kind: "Rule", Metadata: core.Metadata{Name: item.name}, Path: "docs/" + item.name + ".yaml", Spec: core.Spec{Files: []string{path}, Assertions: []core.Assertion{{Subject: "release-approval", Predicate: "owner", Value: item.value, Source: path, Quote: quote}}}}
		resources[r.Key()] = r
	}
	p := &Project{Snapshot: &snapshot.Snapshot{Files: files}, Graph: &core.Graph{Project: project, Resources: resources}}
	findings := CheckConsistency(p)
	if len(findings) != 2 || findings[0].Code != "consistency.conflict" || findings[0].Path != "docs/second.md" || findings[1].Code != "consistency.conflict" || findings[1].Path != "docs/third.md" {
		t.Fatalf("conflicts were not reported in stable owner order: %#v", findings)
	}
}
