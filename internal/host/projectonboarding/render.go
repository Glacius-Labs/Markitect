package projectonboarding

import (
	"fmt"
	"sort"
	"strings"
)

const beginMarker = "<!-- BEGIN MARKITECT MODEL-FIRST -->"
const endMarker = "<!-- END MARKITECT MODEL-FIRST -->"

func renderFiles(options Options) ([]FileChange, error) {
	workflow, err := renderWorkflow(options.DocumentationPath)
	if err != nil {
		return nil, err
	}
	files := []FileChange{{Path: workflowPath, Action: "", Content: workflow}}
	for _, provider := range options.Providers {
		switch provider {
		case Codex:
			files = append(files, providerFiles(".agents/skills", codexEntry())...)
		case Claude:
			files = append(files, providerFiles(".claude/skills", claudeEntry())...)
		}
	}
	for i := range files {
		if files[i].Path == workflowPath {
			continue
		}
		files[i].Content = strings.TrimSuffix(files[i].Content, "\n") + "\n"
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func renderWorkflow(documentationPath string) (string, error) {
	body := fmt.Sprintf(`# Markitect project workflow

This shared guide is for Codex and Claude Code as outer project agents. The selected project is .markitect/project.yaml; its canonical model is YAML and its runtime is .markitect/runtime.yaml. Use the installed Markitect MCP server for project operations. The server is fixed to the repository root selected at startup; tool arguments cannot redirect it. MCP input schemas are typed and closed. Follow each schema and the latest Host result instead of hand-building CLI record files or inventing IDs and digests.

## Start with an ordinary Work Item

For a short issue, bug, feature, or backlog prompt, inspect repository and Git context, then use project_index, project_check, and project_context as needed. Clarify only material intent, scope, acceptance, or authority; otherwise state a bounded assumption and continue. Keep accepted decisions, assumptions, options, unresolved questions, owners, and blockers in the durable exploration record and active task.

Create or update the typed one-scope record with project_explore and write:false. Review its selected-model binding and returned plan.digest; persist that exact plan with write:true and expectedDigest. Read the scope using project_readiness. If the structure needs acknowledgement, preview first and write only with the returned writePlan.digest and true actor/provenance. Never impersonate a user or imply identity authentication. A draft or exploration record is not accepted model intent.

Minimal new exploration input record (pass as record to project_explore):

%[2]sjson
{
  "apiVersion": "markitect.example.org/project-exploration/v1alpha1",
  "id": "cancel-order",
  "status": "active",
  "request": "Support safe order cancellation.",
  "scopes": [{"id":"cancel-order","name":"Order cancellation","goal":"Allow eligible orders to be canceled and restore reserved stock once.","operation":"apply","managerIds":["<existing-manager-id>"]}],
  "decisions": [],
  "drafts": [],
  "structureAcknowledgements": [],
  "completions": []
}
%[2]s

Use project_edit to preview a bounded canonical model mutation and inspect its structural report and impact. Write only with that exact digest. Repair actionable project_check findings within delegated authority. Under committed-model policy, only the valid selected canonical model at committed HEAD is accepted. Satisfy required checks/review and commit the model on the task branch before readiness and implementation; this does not claim human approval.

For existing code, inspect and cite actual files, history, configuration, and checks. Use project_brownfield and project_brownfield_run for fixed-source discovery, bounded Manager proposals/integration, explicit decisions, and model-only adoption. Preserve evidence ownership, exact child assignments, uncertainty, and unresolved scopes. Initial adoption never changes application source; cleanup or delivery is a separate authorized operation. Observed code is evidence about implementation, not proof of intended behavior.

When the scope is ready and the original Work Item authorizes execution, use project_deliver to advance that same durable scope through planning, Manager work/integration, verification, guarded preflight, and Apply. Inspect every returned stage. For staged work, use project_plan, project_run, project_verify, project_preflight, and project_apply; recover with project_status, project_resume, or project_repair on the same run after interruption or a known corrected failure. Never create a replacement run or blindly replay an unknown outcome. Apply does not merge, publish, or deploy.

Technical checks, semantic evidence, and human acceptance are distinct. A passing check or successful Apply alone does not establish semantic correctness or acceptance of the requested outcome. The project documentation destination is %[1]q; keep it readable and generated from accepted model intent. Project guidance does not prevent repository writers with ordinary repository write access from bypassing tools; guarded Apply and checks enforce only their declared boundaries. These instructions do not create an operating-system security boundary.
`, documentationPath, strings.Repeat(string(rune(96)), 3))
	return managedBlock(body), nil
}

func codexEntry() string {
	return rootEntry("Codex", "codex")
}

func claudeEntry() string {
	return rootEntry("Claude Code", "claude")
}

func rootEntry(provider, providerFlag string) string {
	clientSetup := ""
	switch provider {
	case "Codex":
		clientSetup = "Connect the outer Codex client to this repository's Markitect MCP server with `codex mcp add markitect -- ABSOLUTE_MARKITECT_EXECUTABLE project mcp --repo ABSOLUTE_PROJECT_ROOT`; confirm with `codex mcp list` or `/mcp` (see the [Codex MCP guide](https://developers.openai.com/codex/mcp)). Markitect schedules inner Manager and reviewer work through Codex CLI 0.162.0 App Server using `gpt-6-luna` at `high`; the outer agent must not schedule duplicate Managers."
	case "Claude Code":
		clientSetup = "For Claude Code 2.1.295 as the outer project agent, connect its MCP client with `claude mcp add --transport stdio markitect -- ABSOLUTE_MARKITECT_EXECUTABLE project mcp --repo ABSOLUTE_PROJECT_ROOT`; confirm with `claude mcp list` and `/mcp` (see the [Claude Code MCP guide](https://code.claude.com/docs/en/mcp)). Markitect's supported inner Manager and reviewer runtime is Codex CLI 0.162.0 App Server using `gpt-6-luna` at `high`; Claude Code is not used as an inner worker."
	}
	return fmt.Sprintf("This repository uses the selected Markitect model in `.markitect/project.yaml`. For an ordinary issue, bug, feature, or backlog Work Item, start with `markitect-implement`; use `markitect-init`, `markitect-extract`, `markitect-design`, `markitect-suggest`, `markitect-configure`, `markitect-cleanup`, `markitect-verify`, `markitect-apply`, or `markitect-check` when that is the specific task. Follow the shared [project workflow](.markitect/workflows/model-first.md) and use Markitect MCP as the outer project-operation interface. %s Quote executable and repository-root placeholders for your shell; the server fixes its authority to that selected root. For a new project or first setup, preview `markitect project init --repo ABSOLUTE_PROJECT_ROOT --name NAME` and use that operation's documented write option; then preview `markitect project onboard --repo ABSOLUTE_PROJECT_ROOT --provider %s` and write only with its returned digest. Onboard does not configure MCP or change global client settings.", clientSetup, providerFlag)
}

func skillFile(name, description, entry string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n%s\n", name, description, managedBlock(entry))
}

func providerFiles(skillRoot, entry string) []FileChange {
	rootEntryPath := "CLAUDE.md"
	if strings.HasPrefix(skillRoot, ".agents/") {
		rootEntryPath = "AGENTS.md"
	}
	files := []FileChange{{Path: rootEntryPath, Content: managedBlock(entry)}}
	for _, capability := range capabilitySkills() {
		base := strings.TrimSuffix(skillRoot, "/") + "/" + capability.name
		files = append(files,
			FileChange{Path: base + "/SKILL.md", Content: skillFile(capability.name, capability.description, capability.body)},
			FileChange{Path: base + "/references/operating-guide.md", Content: managedBlock(capabilityGuide(capability))},
		)
		if capability.name == "markitect-implement" {
			files = append(files, FileChange{Path: base + "/references/recovery.md", Content: managedBlock(recoveryReference())})
		}
	}
	return files
}

func managedBlock(body string) string {
	body = strings.TrimSpace(body)
	return beginMarker + "\n" + body + "\n" + endMarker
}

// mergeManaged preserves every existing byte outside the marked region. A
// missing region is appended; partial, repeated, nested, or reversed markers
// are refused so hand-written material is never guessed at or discarded.
func mergeManaged(existing, desired string) (string, error) {
	starts := strings.Count(existing, beginMarker)
	ends := strings.Count(existing, endMarker)
	if starts == 0 && ends == 0 {
		if existing == "" {
			if strings.HasSuffix(desired, "\n") {
				return desired, nil
			}
			return desired + "\n", nil
		}
		separator := ""
		if !strings.HasSuffix(existing, "\n") {
			separator = "\n"
		}
		if strings.HasSuffix(desired, "\n") {
			return existing + separator + desired, nil
		}
		return existing + separator + desired + "\n", nil
	}
	if starts != 1 || ends != 1 {
		return "", fmt.Errorf("managed onboarding block must have exactly one begin and end marker")
	}
	start := strings.Index(existing, beginMarker)
	end := strings.Index(existing, endMarker)
	if end < start+len(beginMarker) {
		return "", fmt.Errorf("managed onboarding block markers are malformed or reversed")
	}
	if strings.Contains(existing[start+len(beginMarker):end], beginMarker) || strings.Contains(existing[start+len(beginMarker):end], endMarker) {
		return "", fmt.Errorf("managed onboarding block contains nested markers")
	}
	suffixStart := end + len(endMarker)
	suffix := existing[suffixStart:]
	if strings.HasSuffix(desired, "\n") && (strings.HasPrefix(suffix, "\n") || strings.HasPrefix(suffix, "\r\n")) {
		desired = strings.TrimSuffix(desired, "\n")
	}
	return existing[:start] + desired + suffix, nil
}

func mergeFile(filePath, existing, desired string) (string, error) {
	if filePath == workflowPath {
		if !strings.Contains(existing, beginMarker) && existing != desired && existing != desired+"\n" {
			return "", fmt.Errorf("existing generated workflow %s has no Markitect managed block", filePath)
		}
		return mergeManaged(existing, desired)
	}
	if strings.HasSuffix(filePath, "/SKILL.md") {
		start := strings.Index(desired, beginMarker)
		end := strings.Index(desired, endMarker)
		if start < 0 || end < start {
			return "", fmt.Errorf("generated skill has no valid managed block")
		}
		frontmatter := desired[:start]
		if !hasYAMLFrontmatter(frontmatter) {
			return "", fmt.Errorf("generated skill has no valid YAML frontmatter")
		}
		block := desired[start : end+len(endMarker)]
		if existing == "" {
			return desired, nil
		}
		if hasFrontmatterOpening(existing) {
			if !hasYAMLFrontmatter(existing) {
				return "", fmt.Errorf("existing skill has malformed YAML frontmatter")
			}
		} else if strings.HasPrefix(existing, "---") {
			return "", fmt.Errorf("existing skill has malformed YAML frontmatter")
		} else {
			return "", fmt.Errorf("target skill %s already exists without a Markitect managed block; refusing to replace or adopt it", filePath)
		}
		if !strings.Contains(existing, beginMarker) || !strings.Contains(existing, endMarker) {
			return "", fmt.Errorf("target skill %s already exists without a Markitect managed block; refusing to replace or adopt it", filePath)
		}
		if err := validateSkillIdentity(filePath, existing); err != nil {
			return "", err
		}
		return mergeManaged(existing, block)
	}
	return mergeManaged(existing, desired)
}

func validateSkillIdentity(filePath, existing string) error {
	expectedName := filePath[strings.LastIndex(filePath[:strings.LastIndex(filePath, "/")], "/")+1 : strings.LastIndex(filePath, "/")]
	actualName := ""
	lines := strings.Split(existing, "\n")
	for index := 1; index < len(lines); index++ {
		line := strings.TrimSuffix(lines[index], "\r")
		if line == "---" {
			break
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			actualName = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")), "\"'")
			break
		}
	}
	if actualName != expectedName {
		return fmt.Errorf("target skill %s has frontmatter name %q; refusing same-path skill collision", filePath, actualName)
	}
	return nil
}

func hasFrontmatterOpening(content string) bool {
	return strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n")
}

func hasYAMLFrontmatter(content string) bool {
	openingLength := 0
	switch {
	case strings.HasPrefix(content, "---\n"):
		openingLength = len("---\n")
	case strings.HasPrefix(content, "---\r\n"):
		openingLength = len("---\r\n")
	default:
		return false
	}
	body := content[openingLength:]
	return strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") ||
		strings.Contains(body, "\n---\n") || strings.Contains(body, "\n---\r\n") ||
		strings.HasSuffix(body, "\n---") || strings.HasSuffix(body, "\r\n---")
}
