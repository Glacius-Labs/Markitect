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
			files = append(files,
				FileChange{Path: "AGENTS.md", Content: managedBlock(codexEntry()), Action: ""},
				FileChange{Path: ".agents/skills/markitect-model-first/SKILL.md", Content: skillFile("markitect-model-first", "Use the shared Markitect model-first workflow when exploring or implementing a Markitect-managed project.", skillEntry()), Action: ""},
			)
		case Claude:
			files = append(files,
				FileChange{Path: "CLAUDE.md", Content: managedBlock(claudeEntry()), Action: ""},
				FileChange{Path: ".claude/skills/markitect-model-first/SKILL.md", Content: skillFile("markitect-model-first", "Use the shared Markitect model-first workflow when exploring or implementing a Markitect-managed project.", skillEntry()), Action: ""},
			)
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
	body := fmt.Sprintf(`# Model-first project workflow

This file is the shared workflow for contributors using Codex and Claude. Treat the selected Markitect project model as the canonical description of intended behavior, ownership, artifacts, checks, and relationships. Code, tests, documentation, and infrastructure realize that model; their presence alone does not prove they agree with it.

## Start in the conversation

Let the contributor describe the project or requested change in ordinary language. Do not require them to know Markitect commands or YAML. Begin with the project's purpose and the outcome they want. Model useful concepts, rules, use cases, architecture, and workflow while discussing them; keep accepted decisions distinct from assumptions, suggestions, and open questions. Ask focused follow-up questions when a missing decision blocks the next design or implementation step. Do not invent external facts.

## Explore before implementation

During Explore, prepare model edits and readable explanations, and use structural checks and impact results to find missing references and affected areas. Keep unresolved decisions visible with their scope and whether they block work. A compiler error is a prompt to repair or clarify the draft, not a reason to start implementation. Discuss technology choices, Manager boundaries, shared artifact contracts, and the proposed first-scope file structure with the contributor. Managers own their declared areas; shared artifacts may be realized by several areas, with each file retaining one accountable owner and explicit cross-area contracts.

The project documentation destination is %q. Keep it readable and derived from the current accepted model through Markitect's documented generation path. Preserve independent documents under their named owners.

## First-scope readiness

Before implementation, name the first usable scope and show its proposed file structure, required artifacts, Managers, checks, open decisions, and known evidence gaps. Continue discussing or revising the model until decisions that block this scope are resolved. Make the contributor's review of the proposed structure explicit in the conversation. That acknowledgement is conversation evidence, not identity authentication or a technical access control. Keep unresolved decisions and readiness status visible; do not imply they are durably stored unless the project host provides that state. Explore remains unfinished until a successful first Apply for the agreed scope.

## Implement and close the first Apply

When an accepted model change is committed, generate and review the deterministic project brief for the exact before and after revisions before assigning the next Manager work. Use the markitect project brief command with the accepted decision reference; preview first, then persist against the returned current state digest. Share each Manager's scoped brief and public neighbor contracts; use them to orient the next plan. A brief summarizes declared model changes; it does not infer meaning from prose or prove implementation correctness. A caller-declared decision reference does not authenticate identity.

For an implementation request, use the project's model edit and impact workflows as needed, then plan the bounded goal against the current committed model. Review the plan and Manager assignments with the contributor. Run the delegated implementation, reviews, integration, and declared checks. Complete full verification against the final candidate (supply the CLI's required --write flag for an agent verification invocation) and apply only the verified candidate with its current bindings. If a check or verification fails, report the gap and continue the same bounded workflow; do not call the attempt successful. Apply does not publish or deploy.

After first Apply, use the same model, ownership, impact, planning, run, verification, and Apply workflow for later changes. A new feature can begin another Explore cycle without resetting the project. Keep technical validation, semantic evidence, and human acceptance distinguishable.

## Authority boundary

These repository instructions help the selected agent follow the workflow. They do not prevent a contributor or agent with ordinary repository write access from editing files directly or bypassing Markitect. Rely on the project's guarded Apply and configured repository checks for their declared boundaries; do not claim that native instructions create an operating-system security boundary.
`, documentationPath)
	return managedBlock(body), nil
}

func codexEntry() string {
	return "Read and follow the shared [Markitect model-first workflow](.markitect/workflows/model-first.md). Guide the contributor through Explore in ordinary conversation; do not ask them to issue Markitect commands. Keep assumptions and unresolved decisions visible, and use the workflow's model, impact, planning, verification, and Apply steps."
}

func claudeEntry() string {
	return "Read and follow the shared [Markitect model-first workflow](.markitect/workflows/model-first.md). Guide the contributor through Explore in ordinary conversation; do not ask them to issue Markitect commands. Keep assumptions and unresolved decisions visible, and use the workflow's model, impact, planning, verification, and Apply steps."
}

func skillFile(name, description, entry string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, managedBlock(entry))
}

func skillEntry() string {
	return "For project modeling and implementation, read the repository-root .markitect/workflows/model-first.md file and follow it. This shared workflow covers natural-language Explore, visible open decisions, model and impact work, Manager ownership, first-scope readiness, model-change briefs, implementation, full verification, and Apply."
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
			return "", fmt.Errorf("existing canonical workflow has no Markitect managed block")
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
		if !strings.HasPrefix(frontmatter, "---\n") || !strings.Contains(frontmatter[len("---\n"):], "\n---\n") {
			return "", fmt.Errorf("generated skill has no valid YAML frontmatter")
		}
		block := desired[start : end+len(endMarker)]
		if existing == "" {
			return desired, nil
		}
		if strings.HasPrefix(existing, "---\n") {
			if !strings.Contains(existing[len("---\n"):], "\n---\n") {
				return "", fmt.Errorf("existing skill has malformed YAML frontmatter")
			}
		} else {
			// Keep existing native skill bytes intact and add the frontmatter
			// required for discovery ahead of them before managing our block.
			existing = frontmatter + existing
		}
		return mergeManaged(existing, block)
	}
	return mergeManaged(existing, desired)
}
