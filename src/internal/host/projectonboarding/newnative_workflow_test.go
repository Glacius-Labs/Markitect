package projectonboarding

import (
	"strings"
	"testing"
)

func TestGeneratedWorkflowUsesBoundedTypedMCPDelivery(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	workflow := fileFor(t, Plan{Files: files}, workflowPath).Content
	for _, tool := range []string{
		"use model, check, and context", "with explore and write:false", "using ready", "Use edit to preview",
		"Use adopt and its run stage", "use deliver to advance", "use plan, run, verify, and apply (preview, then write)",
		"recover with status, resume, or repair",
	} {
		if !strings.Contains(workflow, tool) {
			t.Errorf("shared workflow does not route through typed MCP tool %s", tool)
		}
	}
	for _, invariant := range []string{
		"typed and closed", "fixed to the repository root", "write:false", "write:true and expect.",
		"returned plan.digest", "returned writePlan.digest", "same durable scope",
		"Never create a replacement run", "Technical checks, semantic evidence, and human acceptance are distinct",
	} {
		if !strings.Contains(workflow, invariant) {
			t.Errorf("shared workflow omits MCP or acceptance invariant %q", invariant)
		}
	}
	for _, obsolete := range []string{
		"Use the native CLI stages below", "brownfield-action start", "markitect project edit --repo", "project_", "project mcp", "expectedDigest",
		"--execution-mode native-work", "--codex-profile luna-high", "helperLimit", "every --request, --output, and --input",
	} {
		if strings.Contains(workflow, obsolete) {
			t.Errorf("shared workflow retains obsolete default-path guidance %q", obsolete)
		}
	}
}

func TestGeneratedOuterClientAndInnerWorkerGuidanceIsProviderSpecific(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	codexEntry := fileFor(t, Plan{Files: files}, "AGENTS.md").Content
	claudeEntry := fileFor(t, Plan{Files: files}, "CLAUDE.md").Content
	for _, required := range []string{
		"ordinary issue, bug, feature, or backlog Work Item", "codex mcp add markitect",
		"ABSOLUTE_MARKITECT_EXECUTABLE", "ABSOLUTE_PROJECT_ROOT", "Codex CLI 0.162.0 App Server",
		"gpt-6-luna", "at `high`", "outer agent must not schedule duplicate Managers",
	} {
		if !strings.Contains(codexEntry, required) {
			t.Errorf("Codex entry is missing supported client guidance %q", required)
		}
	}
	for _, required := range []string{
		"ordinary issue, bug, feature, or backlog Work Item", "Claude Code 2.1.295",
		"claude mcp add --transport stdio markitect", "claude mcp list", "`/mcp`",
		"Claude Code is not used as an inner worker", "Codex CLI 0.162.0 App Server",
		"Onboard does not configure MCP or change global client settings",
	} {
		if !strings.Contains(claudeEntry, required) {
			t.Errorf("Claude entry is missing supported client guidance %q", required)
		}
	}

	for _, providerRoot := range []string{".agents/skills", ".claude/skills"} {
		guide := fileFor(t, Plan{Files: files}, providerRoot+"/markitect-implement/references/operating-guide.md").Content
		for _, required := range []string{
			"outer Codex or Claude Code agent", "Codex CLI 0.162.0 App Server", "ordinary standard file, shell, and test tools",
			"bounded native helper starts", "selected runtime policy", "Inner workers must not edit the canonical control plane",
		} {
			if !strings.Contains(guide, required) {
				t.Errorf("%s implementation guide is missing worker boundary %q", providerRoot, required)
			}
		}
		for _, obsolete := range []string{"--execution-mode native-work", "--codex-profile luna-high", "helperLimit", "CLI exec JSONL"} {
			if strings.Contains(guide, obsolete) {
				t.Errorf("%s implementation guide retains obsolete worker claim %q", providerRoot, obsolete)
			}
		}
	}
}
