package examples

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func TestMinimalExampleCompilesBoundContextAndRenderedViews(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	root := filepath.Join(filepath.Dir(sourceFile), "minimal")
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) > 0 {
		t.Fatalf("minimal example has graph diagnostics: %#v", project.Diagnostics)
	}

	wantKinds := map[string]bool{
		"Text": false, "Rule": false, "Contract": false,
		"Workflow": false, "Skill": false, "Agent": false,
	}
	for _, resource := range project.Resources {
		if _, ok := wantKinds[resource.Kind]; ok {
			wantKinds[resource.Kind] = true
		}
	}
	for kind, found := range wantKinds {
		if !found {
			t.Errorf("example is missing content kind %s", kind)
		}
	}

	context, err := app.CompileContext(project, "sample/Skill/rollback-review", "example-test")
	if err != nil {
		t.Fatal(err)
	}
	seenAgent, seenContract, seenInput := false, false, false
	for _, input := range context.Inputs {
		if input.Resource != nil {
			switch input.Resource.Key() {
			case "sample/Agent/rollback-reviewer":
				seenAgent = true
			case "sample/Contract/rollback-assessment":
				seenContract = true
			}
		}
		if input.Key == "file:docs/inputs/rollback-change.txt" {
			seenInput = input.Text != "" && input.Path == "docs/inputs/rollback-change.txt"
		}
	}
	if !seenAgent || !seenContract || !seenInput {
		t.Fatalf("compiled context omitted bound implementation, Contract, or declared input (Agent=%t Contract=%t file=%t)", seenAgent, seenContract, seenInput)
	}

	if findings := app.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("checked-in rendered views have drift or are missing: %#v", findings)
	}

	// A copied example is an authoring baseline. Formatting a local edit must
	// not rewrite unrelated resources and turn its impact into a project change.
	changed, err := app.Format(root, project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("example must start in canonical format, changed: %v", changed)
	}
}

func TestConsistencyConflictExampleReportsSourceAndOwners(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	root := filepath.Join(filepath.Dir(sourceFile), "consistency-conflict")
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("unexpected graph diagnostics: %#v", project.Diagnostics)
	}
	findings := app.CheckOutputs(project)
	if len(findings) != 1 || findings[0].Code != "consistency.conflict" || findings[0].Path != "docs/review-policy.md" || findings[0].Line != 3 || !strings.Contains(findings[0].Message, "docs/operations-policy.md:3") || !strings.Contains(findings[0].Message, "sample/Workflow/review-approval") {
		t.Fatalf("expected one sourced conflict with both owners: %#v", findings)
	}
}

func TestDocumentationPlacementExampleHasValidRouters(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	root := filepath.Join(filepath.Dir(sourceFile), "documentation-placement")
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("placement example has resource diagnostics: %#v", project.Diagnostics)
	}
	if findings := app.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("placement example has output diagnostics: %#v", findings)
	}
	if findings := app.CheckDocumentationRouters(project); len(findings) != 0 {
		t.Fatalf("placement example has router diagnostics: %#v", findings)
	}
}
