package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func TestRunContextUsesFixedRevisionAndExactSelectedPaths(t *testing.T) {
	repo := newCLIRepo(t, false)
	manifest := runManifestYAML("work-items/17.md", "internal/orders/OrderService.cs")
	writeRepoFile(t, repo.root, "context-run.yaml", []byte(manifest))
	writeRepoFile(t, repo.root, "work-items/17.md", []byte("# Work item 17\n\nAcceptance criteria: preserve idempotency.\n"))
	writeRepoFile(t, repo.root, "internal/orders/OrderService.cs", []byte("public sealed class OrderService { }\n"))
	git(t, repo.root, "add", "context-run.yaml", "work-items/17.md", "internal/orders/OrderService.cs")
	git(t, repo.root, "commit", "-m", "add fixed run context fixture")
	revision := git(t, repo.root, "rev-parse", "HEAD")
	writeRepoFile(t, repo.root, "internal/orders/OrderService.cs", []byte("working tree must not enter context\n"))

	code, output, stderr := invoke("context", "--repo", repo.root, "--revision", revision, "--run", "context-run.yaml")
	if code != 0 {
		t.Fatalf("context exit=%d stderr=%s output=%s", code, stderr, output)
	}
	ctx := decodeYAML[app.Context](t, output)
	if ctx.Revision != revision || ctx.Provisional || !ctx.Complete || ctx.Status != "complete" || ctx.Run == nil {
		t.Fatalf("run context has wrong fixed snapshot identity: %#v", ctx)
	}
	if ctx.Run.ManifestHash != app.Hash([]byte(manifest)) || ctx.Run.ManifestPath != "context-run.yaml" || ctx.Run.TaskID != "work-item-17" {
		t.Fatalf("run context omitted manifest evidence: %#v", ctx.Run)
	}
	inputs := map[string]app.ContextInput{}
	for _, input := range ctx.Inputs {
		inputs[input.Role+":"+input.Path] = input
	}
	for key, expected := range map[string]string{
		"task:work-items/17.md":                  "# Work item 17\n\nAcceptance criteria: preserve idempotency.\n",
		"source:internal/orders/OrderService.cs": "public sealed class OrderService { }\n",
	} {
		input, ok := inputs[key]
		if !ok || input.Status != "included" || input.Hash != app.Hash([]byte(expected)) || input.Text != expected {
			t.Errorf("input %s = %#v, want exact fixed-snapshot bytes", key, input)
		}
	}
	if strings.Contains(output, "working tree must not enter context") {
		t.Fatal("working-tree edit entered fixed-revision context")
	}
}

func TestRunContextReportsMissingRequiredInputsAsIncomplete(t *testing.T) {
	repo := newCLIRepo(t, false)
	writeRepoFile(t, repo.root, "context-run.yaml", []byte(runManifestYAML("work-items/missing.md", "internal/missing.cs")))
	git(t, repo.root, "add", "context-run.yaml")
	git(t, repo.root, "commit", "-m", "add incomplete run context fixture")
	revision := git(t, repo.root, "rev-parse", "HEAD")

	code, output, stderr := invoke("context", "--repo", repo.root, "--revision", revision, "--run", "context-run.yaml")
	if code != 2 {
		t.Fatalf("incomplete context exit=%d stderr=%s output=%s, want 2", code, stderr, output)
	}
	ctx := decodeYAML[app.Context](t, output)
	if ctx.Complete || ctx.Status != "incomplete" {
		t.Fatalf("missing required inputs were not marked incomplete: %#v", ctx)
	}
	missing := map[string]bool{}
	for _, input := range ctx.Inputs {
		if input.Status == "missing" && input.Required {
			missing[input.Role+":"+input.Path] = true
		}
	}
	if !missing["task:work-items/missing.md"] || !missing["source:internal/missing.cs"] {
		t.Fatalf("missing required task/source not shown in report: %#v", ctx.Inputs)
	}
}

func TestRunContextRequiresFixedRevisionAndRejectsDuplicatePaths(t *testing.T) {
	repo := newCLIRepo(t, false)
	if code, _, _ := invoke("context", "--repo", repo.root, "--run", "context-run.yaml"); code == 0 {
		t.Fatal("run context accepted an omitted fixed revision")
	}
	duplicate := fmt.Sprintf("version: %s\nentry: %s\ntask:\n  id: work-item-17\n  path: work-items/17.md\nsources:\n  - path: work-items/17.md\n    reason: duplicate task\n", app.RunManifestVersion, cliNamespace+"/Skill/entry")
	if _, err := app.ParseRunManifest([]byte(duplicate)); err == nil {
		t.Fatal("run manifest accepted a source path duplicated from the task path")
	}
}

func runManifestYAML(taskPath, sourcePath string) string {
	return fmt.Sprintf("version: %s\nentry: %s\ntask:\n  id: work-item-17\n  path: %s\nsources:\n  - path: %s\n    reason: implementation source selected for this work item\n", app.RunManifestVersion, cliNamespace+"/Skill/entry", taskPath, sourcePath)
}
