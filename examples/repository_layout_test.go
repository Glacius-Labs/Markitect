package examples

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

func TestRepositoryLayoutExampleCompilesExplicitInputsAndNativeOutput(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	root := filepath.Join(filepath.Dir(sourceFile), "repository-layout")
	project, err := app.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("layout example has graph diagnostics: %#v", project.Diagnostics)
	}
	context, err := app.CompileContext(project, "engineering/Workflow/review-change", "layout-example-test")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"engineering/Workflow/review-change":        ".markitect/areas/engineering/review/review-change.workflow.yaml",
		"engineering/Rule/change-review":            ".markitect/areas/engineering/review/change-review.rule.yaml",
		"shared/Text/principles":                    ".markitect/areas/shared/review/principles.text.yaml",
		"file:docs/engineering/change-procedure.md": "docs/engineering/change-procedure.md",
	}
	for _, input := range context.Inputs {
		key := input.Key
		if input.Resource != nil {
			key = input.Resource.Key()
			if expected, ok := want[key]; ok && input.Resource.Path == expected && input.Resource.Spec.Text != "" {
				delete(want, key)
			}
		} else if expected, ok := want[key]; ok && input.Path == expected && input.Text != "" {
			delete(want, key)
		}
		if input.Key == "file:docs/README.md" || input.Key == "file:docs/engineering/README.md" {
			t.Fatalf("navigation became an implicit input: %#v", input)
		}
	}
	if len(want) != 0 {
		t.Fatalf("context omitted explicit typed or ordinary inputs: %v", want)
	}
	outputs, err := render.Generate(project.Graph, project.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := outputs[".claude/rules/change-review.md"]; !ok || len(outputs) != 1 {
		t.Fatalf("provider-only project must produce exactly its native rule adapter: %v", outputs)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "markitect")); !os.IsNotExist(err) {
		t.Fatalf("provider-only example must not contain a Markdown view tree: %v", err)
	}
	if findings := app.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("layout example has output diagnostics: %#v", findings)
	}
	if findings := app.CheckDocumentationRouters(project); len(findings) != 0 {
		t.Fatalf("layout example has router diagnostics: %#v", findings)
	}
	if changed, err := app.Format(root, project, false); err != nil || len(changed) != 0 {
		t.Fatalf("layout example is not canonically formatted: changed=%v err=%v", changed, err)
	}
}
