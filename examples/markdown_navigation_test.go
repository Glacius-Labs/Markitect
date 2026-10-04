package examples

import (
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
)

func TestMarkdownNavigationExampleRebasesHumanLinksAndPreservesCode(t *testing.T) {
	root := markdownNavigationRoot(t)
	project, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("navigation fixture has graph diagnostics: %#v", project.Diagnostics)
	}

	const entry = "general/Skill/setup-workspace"
	wantEdges := []string{"general/Workflow/setup-workspace"}
	if got := project.Graph.Edges[entry]; len(got) != len(wantEdges) || len(got) != 0 && got[0] != wantEdges[0] {
		t.Fatalf("ordinary prose links changed the declared dependency graph: got %v, want %v", got, wantEdges)
	}

	outputs, err := host.GenerateOutputs(project)
	if err != nil {
		t.Fatal(err)
	}
	const view = "docs/markitect/general/skills/setup-workspace.skill.md"
	body := string(outputs[view])
	for _, expected := range []string{
		"[setup workflow](../workflows/setup-workspace.workflow.md#steps)",
		"[canonical workflow YAML](../workflows/setup-workspace.workflow.md#steps)",
		"[tool guide](../../../general/tools/README.md)",
		"![Workspace flow][workspace-flow]",
		"[workspace-flow]: ../../../general/images/workspace-flow.svg?view=compact#components \"Workspace flow diagram\"",
		"`../workflows/not-a-link.md`",
		"[This fenced link](../workflows/not-a-link.md)",
		"![This fenced image](../images/not-an-image.svg)",
		"[workspace-flow]: ../images/not-an-image.svg?view=compact#fake",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("rendered companion is missing unchanged or rebased content %q:\n%s", expected, body)
		}
	}
	if strings.Contains(body, "[Tool guide](../tools/README.md)") {
		t.Fatalf("ordinary source-relative prose link was not rebased:\n%s", body)
	}

	for _, target := range []string{
		"../workflows/setup-workspace.workflow.md#steps",
		"../../../general/tools/README.md",
		"../../../general/images/workspace-flow.svg?view=compact#components",
	} {
		assertMarkdownTargetExists(t, view, target, project.Snapshot.Files, outputs)
	}

	provider := string(outputs[".claude/skills/setup-workspace/SKILL.md"])
	if !strings.Contains(provider, "[Workflow: setup-workspace](../../../docs/general/workflows/setup-workspace.yaml)") {
		t.Fatalf("provider dependency must resolve to the canonical source YAML:\n%s", provider)
	}
	assertMarkdownTargetExists(t, ".claude/skills/setup-workspace/SKILL.md", "../../../docs/general/workflows/setup-workspace.yaml", project.Snapshot.Files, outputs)

	if findings := host.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("checked-in navigation outputs have drift or are missing: %#v", findings)
	}
}

func TestMarkdownNavigationPrefersExistingUnmarkedOrdinaryAlias(t *testing.T) {
	root := markdownNavigationRoot(t)
	project, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	files := cloneNavigationFiles(project.Snapshot.Files)
	const alias = "docs/general/workflows/setup-workspace.md"
	files[alias] = []byte("# Human-owned workflow alias\n")

	candidate, err := host.Parse(&snapshot.Snapshot{Files: files})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := host.GenerateOutputs(candidate)
	if err != nil {
		t.Fatal(err)
	}
	const view = "docs/markitect/general/skills/setup-workspace.skill.md"
	body := string(outputs[view])
	want := "[setup workflow](../../../general/workflows/setup-workspace.md#steps)"
	if !strings.Contains(body, want) {
		t.Fatalf("existing unmarked Markdown alias must take precedence over the typed view; missing %q:\n%s", want, body)
	}
	assertMarkdownTargetExists(t, view, "../../../general/workflows/setup-workspace.md#steps", files, outputs)
}

func markdownNavigationRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate example test source")
	}
	return filepath.Join(filepath.Dir(sourceFile), "markdown-navigation")
}

func cloneNavigationFiles(files map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(files))
	for name, data := range files {
		cloned[name] = append([]byte(nil), data...)
	}
	return cloned
}

func assertMarkdownTargetExists(t *testing.T, from, target string, files, outputs map[string][]byte) {
	t.Helper()
	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("invalid generated Markdown URL %q: %v", target, err)
	}
	if parsed.IsAbs() || parsed.Host != "" {
		t.Fatalf("expected repository-relative Markdown URL, got %q", target)
	}
	resolved := path.Clean(path.Join(path.Dir(from), parsed.Path))
	if _, ok := outputs[resolved]; ok {
		return
	}
	if _, ok := files[resolved]; ok {
		return
	}
	t.Errorf("Markdown URL %q from %q resolves to missing repository target %q", target, from, resolved)
}
