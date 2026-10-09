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
				FileChange{Path: ".agents/skills/markitect-model-first/SKILL.md", Content: skillFile("markitect-model-first", skillDescription(), skillEntry()), Action: ""},
			)
		case Claude:
			files = append(files,
				FileChange{Path: "CLAUDE.md", Content: managedBlock(claudeEntry()), Action: ""},
				FileChange{Path: ".claude/skills/markitect-model-first/SKILL.md", Content: skillFile("markitect-model-first", skillDescription(), skillEntry()), Action: ""},
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

## Start from an ordinary work item

When a contributor gives a short Work Item, issue, bug, idea, or casual change request, use this model-first skill as the default starting workflow. Treat the short request as an entry point, not as complete requirements or authorization to skip exploration. Let the contributor explain the project and desired outcome in ordinary language and conversation; do not require Markitect commands or YAML knowledge.

Keep an explicit decision ledger as the work proceeds: accepted intent, assumptions, options considered, unresolved questions, their owners, and which selected scopes they block. Carry that ledger forward in the active work item and recover durable state from the repository model and persisted Markitect run when resuming. Dialogue or a draft is not durable acceptance. Do not invent external facts.

Ask the contributor only when material information is missing and changes intended behavior, scope, acceptance, authority, or readiness; otherwise state a bounded assumption and continue. Do not ask for facts already available from the accepted model or repository.

## Explore before implementation

During Explore, discuss useful concepts, rules, use cases, architecture, and workflow. Prepare model edits and explanations, then use structural checks and impact results to find missing references and affected areas. Keep accepted decisions separate from assumptions, suggestions, and open questions. A compiler error is a prompt to repair or clarify the draft, not a reason to start implementation. Discuss technology choices, Manager boundaries, shared artifact contracts, and a proposed first-scope file structure with the contributor. Managers own their declared areas; a shared artifact can serve several areas, while each file keeps one accountable owner and explicit cross-area contracts.

For an existing codebase, reverse-model it iteratively from inspected files, repository history, configuration, and observed checks. Keep a reverse-modeling ledger that connects each proposed rule, artifact, and owner to inspected paths and records unknowns or confidence gaps. Reconcile that ledger with the contributor; do not infer behavior from names, comments, or directory structure alone. Use explicit transient scopes for areas that remain unmodeled, and never present those areas as covered or conforming. Keep the initial adoption model-only. Once the model is accepted under the repository's policy at a committed model revision, plan any cleanup as a separate operation.

For a Brownfield Work Item, use the structured session to turn that ledger into bounded Manager responsibilities. Inspect the target and source at fixed revisions, run markitect project discover, then start a session from its discovery output and explicit scope statuses. Begin one root iteration with its own inspected evidence IDs and a concise purpose/review. Those own evidence IDs provide raw source content for claims and citations. An optional delegationEvidenceIDs pool gives the root permission only to route those IDs to children; its context exposes them as metadata (ID, path, basis, and digest), not raw content, and metadata does not establish behavior. Keep own and delegation pools disjoint. For new root iterations, set delegationEvidenceIDs explicitly: use [] to permit no child evidence or name only selected child evidence. An omitted root pool retains legacy broad routing over selected Discovery evidence. The durable explicit-empty marker preserves [] across session serialization. In a child BEGIN.json, copy the exact evidenceIds and delegationEvidenceIds assigned by its parent proposal; do not broaden or rewrite either pool. Read the child's bounded context and run its propose phase.

The root context also shows the selected Discovery inventory as metadata only. Do not load all child source into the root evidence pool: child Managers receive raw content only for their own evidenceIds. After each proposal assigns direct children and evidence, begin a separate child iteration and repeat recursively. After all children finish proposals and non-leaf integrations, retrieve the parent's context with phase integrate and run its integration phase. The parent receives each direct child's exact final report, citation references, public contracts, and bound digests; non-leaf reports include the completed descendant integration. Forward those citations when relevant; the parent does not need raw child source. Each Manager is a distinct invocation. For a proposed Manager, runtime configuration may map execution to an accepted ancestor, but the proposed Manager remains the work identity and receives only its bounded context.

Use the native CLI stages below; have the agent create the closed JSON files from current command outputs and the repository's actual decisions. START.json wraps the exact DISCOVERY.json object as discovery and includes scopeStatuses (an explicit empty array when none apply). The begin input is a bare ReverseIterationRequest. CONTEXT.json contains the iterationId; for parent integration it also contains phase integrate. Do not ask the contributor to hand-author proposal, integration, or binding digests. The inputs shown are small shape examples, not reusable IDs:

    {"id":"root-review","managerId":"<existing-manager-id>","evidenceIds":["<root-owned-evidence-id>"],"delegationEvidenceIds":["<selected-child-evidence-id>"],"purpose":"Model the observed order-cancellation behavior.","review":"State evidence, uncertainty, and public contracts."}

    markitect project discover --repo TARGET --request DISCOVERY_REQUEST.json --output DISCOVERY.json
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action start --revision TARGET_COMMIT --input START.json
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action start --revision TARGET_COMMIT --input START.json --expect START_SESSION_DIGEST --write
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action begin --session SESSION_ID --input BEGIN.json --expect SESSION_DIGEST --write
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action context --session SESSION_ID --input CONTEXT.json
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action run --session SESSION_ID --input RUN.json
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action run --session SESSION_ID --input RUN.json --expect PREVIEW_DIGEST --write

For run, use closed JSON such as {"iterationId":"ITERATION_ID","phase":"propose","agentManagerId":"ACCEPTED_RUNTIME_MANAGER_ID"}; use phase integrate for a parent. Preview is no-call and returns previewDigest; execution must confirm that exact digest with --expect. It binds session, context, fixed revisions, runtime selection and budget. Review requestContractDigest across attempts: it binds the stable request schema/instructions while deliberately excluding cumulative remaining-budget counters, which may advance after priced work; previewDigest still binds the current ledger and budget. Each write makes exactly one attempt. There is no automatic retry; only explicitly retry the latest known failed attempt with a final receipt, within cumulative configured retry/start/time/cost limits. Unknown cost, ambiguous durable publication, or an attempt without a known terminal receipt fails closed and is not automatically replayable. The process lock is released by the operating system when the owning process exits; inspect the session overview and run journal before resuming, and do not delete the lock file to force progress.

Review outputs at their intended scope: session fields expose a safe overview of fixed bases, scope status, iteration IDs, and digests; resume and plan also retain typed readiness questions and conflict diagnostics for the outer coordinator to route and decide. These diagnostics contain no full source or private report bodies. Context returns only the requested Manager context; the inner Manager request includes that Manager's bounded context (own selected content and authorized metadata) and, for integration, completed direct-child reports, never the complete session or readiness. Run returns the scoped proposal/integration and attempt receipt. The plan action returns the full model AdoptionPlan for review before model-only Apply. An incomplete/ambiguous publication or unknown invocation outcome is a fail-closed state to inspect, not a signal to repeat the command.

After root integration, have the authorized owner or delegated Manager review the ledger and make explicit scope/question decisions within their authority. If the Work Item does not delegate those decisions, a material issue remains open, or repository policy requires a human decision, ask the contributor. Have the agent build RESOLUTION.json from the current proposal and target bindings; its shape is {"iterationId":"ITERATION_ID","resolution":{"apiVersion":"markitect.example.org/project-resolution/v1alpha1","actor":"ACTOR","authorityClaim":"AUTHORITY","decisionReference":"REAL_PROVENANCE","authenticated":false,"questions":[{"questionId":"QUESTION_ID","scopeId":"SCOPE_ID","disposition":"answer","answer":"DECISION","reason":"WHY"}],"scopes":[{"scopeId":"SCOPE_ID","status":"adopt","reason":"WHY"}]}} plus the binding digests copied from current Host outputs. The provenance fields record a caller assertion and do not authenticate a human. Then use:

    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action resolve --session SESSION_ID --input RESOLUTION.json --expect SESSION_DIGEST --write
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action plan --session SESSION_ID --input PLAN.json
    markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action apply-adoption --session SESSION_ID --input APPLY.json --expect SESSION_DIGEST --write

APPLY.json contains the iteration ID and the reviewed expected plan digest from the plan output. Commit and accept the canonical model under repository policy before readiness or implementation; initial adoption remains model-only.

The project documentation destination is %q. Keep it readable and derived from the current accepted model through Markitect's documented generation path. Preserve independent documents under their named owners.

## First-scope readiness

Before implementation, select a first usable scope and compute its readiness from the current fixed project snapshot: structural validity, scoped ownership and dependencies, required artifacts, configured checks, blocking open decisions, and known evidence gaps. Report what is ready and what remains open; readiness is scoped planning evidence, not proof of implementation correctness. Show the proposed file structure, required artifacts, Managers, and checks, and obtain the review or acknowledgement required by repository policy. When the Work Item delegates ordinary implementation and structure decisions within its scope, the responsible Manager may review and acknowledge using its own actor, delegated authority, and task provenance; never present that assertion as the contributor's acknowledgement or as identity authentication. Ask the contributor only when material intent or authority remains undecided, or policy explicitly requires human review.

Only the canonical model owner and repository policy can accept a model change. A saved proposal or draft is not accepted, and committing a draft by itself does not make it accepted. Satisfy the repository's required review and policy checks, then bind implementation planning to the resulting committed model revision. Do not implement a scope while its blocking decisions or model acceptance remain unresolved.

### Durable exploration records and commands

Each ordinary Work Item gets one durable exploration ID and one named scope ID. Keep those IDs stable while updating or resuming that item. A new Work Item receives new IDs. The input is closed JSON with exactly one scope and explicit arrays; substitute a Manager ID that exists in the current model and JSON-escape it as one string value.

Minimal new exploration input:

%[2]sjson
{
  "apiVersion": "markitect.example.org/project-exploration/v1alpha1",
  "id": "cancel-order",
  "status": "active",
  "request": "Support safe order cancellation.",
  "scopes": [
    {
      "id": "cancel-order",
      "name": "Order cancellation",
      "goal": "Allow eligible orders to be canceled and restore reserved stock once.",
      "operation": "apply",
      "managerIds": ["<existing-manager-id>"]
    }
  ],
  "decisions": [],
  "drafts": [],
  "structureAcknowledgements": [],
  "completions": []
}
%[2]s

On creation, the Host fills the source-binding and record digests. To preview a create or update, then persist only that exact plan:

%[2]stext
markitect project explore --repo PATH --input .markitect/drafts/work-item.json
markitect project explore --repo PATH --input .markitect/drafts/work-item.json --expect PLAN_DIGEST --write
markitect project explore --repo PATH --exploration cancel-order
markitect project explore --repo PATH
%[2]s

The first command returns a write plan; inspect its binding and digest before the second. Use the returned WritePlan digest as PLAN_DIGEST. The last commands read one durable record or list records. Keep status active until the Host records a successful Apply receipt; integration alone is not completion. Never delete or recreate a record to avoid its open decisions or history.

## Implement and close the first Apply

After the model is accepted at a committed revision, use the existing Plan, Run, Verify, and Apply lifecycle in that order. The Host reconciles committed accepted-model history automatically; inspect the exact context for a Manager with `+"`markitect project briefings --repo PATH --manager MANAGER_ID`"+`. Reconcile that Manager's scoped briefings, events, and relevant public neighbor contracts before work. Do not invent or manually aggregate a model delta that the accepted-history mechanism already supplies. A briefing summarizes declared model changes; it does not infer meaning from prose or prove implementation correctness. A caller-declared decision reference does not authenticate identity.

Plan only the bounded goal against the selected accepted revision and review the plan and Manager assignments against the Work Item and repository policy. Follow the ownership tree: a Manager owns its assigned scope and delegates only to active direct children; leaf Managers implement their files, an independent reviewer assesses the exact scoped candidate bytes, and each parent integrates direct-child results against its own contracts. Do not substitute a whole-project implementer for these bounded responsibilities. Ask the contributor only when material intent or authority is undecided or the task or policy requires human review. Run the declared checks and complete full verification against the final candidate. Apply only the verified candidate with its current bindings. If a check or verification fails, report the gap and continue the same persisted run; do not call the attempt successful. Apply does not publish or deploy.

If provider setup must be refreshed after Manager-tree edits, inspect its preview and explicitly preserve the already authorized time, start/retry, and cost limits. Setup may replace runtime limits with defaults; never accept a refresh that silently resets those bounds.

The first successful Apply for the acknowledged scope closes the initial work item. Before that point, keep the work item open and continue its bounded repair, review, verification, and Apply cycle. If interrupted or moved to a new conversation, inspect and resume the existing persisted run from its last durable state; do not replay completed Manager work or create a duplicate run. After first Apply, use the same model, ownership, impact, planning, run, verification, and Apply workflow for later changes. A new feature can begin another Explore cycle without resetting the project. Keep technical validation, semantic evidence, and human acceptance distinguishable.

The CLI forms below expose preview digests and durable IDs; replace each placeholder with the value just returned by the Host.

%[2]stext
markitect project readiness --repo PATH --exploration EXPLORATION_ID --scope SCOPE_ID
markitect project readiness --repo PATH --exploration EXPLORATION_ID --scope SCOPE_ID --acknowledge-structure --actor ACTOR --authority AUTHORITY --decision-ref PROVENANCE --acknowledged-at RFC3339_TIME
markitect project readiness --repo PATH --exploration EXPLORATION_ID --scope SCOPE_ID --acknowledge-structure --actor ACTOR --authority AUTHORITY --decision-ref PROVENANCE --acknowledged-at RFC3339_TIME --expect WRITE_PLAN_DIGEST --write
markitect project deliver --repo PATH --exploration EXPLORATION_ID --scope SCOPE_ID --write
markitect project deliver --repo PATH --exploration EXPLORATION_ID --scope SCOPE_ID --run RUN_ID --write
markitect project status --repo PATH --run RUN_ID
markitect project resume --repo PATH --run RUN_ID --write
markitect project repair --repo PATH --run RUN_ID --write
%[2]s

Read the readiness preview's exact Managers, files, artifacts, checks, blockers, binding digest, structure digest, and write-plan digest. Use the returned writePlan.digest for WRITE_PLAN_DIGEST. Generate one explicit UTC RFC3339 --acknowledged-at value and reuse that exact value and all other acknowledgement arguments for preview and write; changing the timestamp changes the plan. Record only real authority and provenance: an authorized Manager may assert its own delegated authority within the Work Item, citing that task; never claim that assertion is a human acknowledgement. Use --actor user only after an actual user decision, with its real authority and provenance. If the task does not authorize the exact structure or a material blocking decision remains open, ask the contributor; do not invent a decision or acknowledgement.

Project deliver uses the existing durable run: interrupted/running work resumes; integrated work advances through verification and Apply; verified work continues guarded Apply; an already applied run recovers the exploration completion from its immutable receipt. If delivery reports blocked or failed, inspect the same run with project status and resolve the reported cause. Resume interrupted work without replay; use repair only for a failed required check after correcting its cause. Never create a replacement run just because integration, verification, or Apply was interrupted.

## Authority boundary

These repository instructions help the selected agent follow the workflow. They do not prevent a contributor or agent with ordinary repository write access from editing files directly or bypassing Markitect. Rely on the project's guarded Apply and configured repository checks for their declared boundaries; do not claim that native instructions create an operating-system security boundary.
`, documentationPath, strings.Repeat(string(rune(96)), 3))
	return managedBlock(body), nil
}

func codexEntry() string {
	return "For a short Work Item, issue, bug, idea, or change request, use the repository-local Markitect model-first skill and follow the shared [workflow](.markitect/workflows/model-first.md). Begin in ordinary conversation, preserve decisions and open questions, acknowledge computed readiness and the proposed first-scope structure, then use the accepted model and existing Plan/Run/Verify/Apply lifecycle. Resume persisted work without replaying completed work."
}

func claudeEntry() string {
	return "For a short Work Item, issue, bug, idea, or change request, use the repository-local Markitect model-first skill and follow the shared [workflow](.markitect/workflows/model-first.md). Begin in ordinary conversation, preserve decisions and open questions, acknowledge computed readiness and the proposed first-scope structure, then use the accepted model and existing Plan/Run/Verify/Apply lifecycle. Resume persisted work without replaying completed work."
}

func skillDescription() string {
	return "Use for ordinary short Work Items, issues, bugs, and ideas in a Markitect project; explore and persist decisions, establish scoped readiness, then implement through the accepted model-first workflow."
}

func skillFile(name, description, entry string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, managedBlock(entry))
}

func skillEntry() string {
	return "When a short Work Item, issue, bug, or idea arrives, read the repository-root .markitect/workflows/model-first.md and follow it as the default. It covers a persistent decision ledger, iterative brownfield reverse-modeling, repository-policy acceptance at a committed model revision, computed first-scope readiness, bounded Manager and reviewer responsibilities, Plan/Run/Verify/Apply, first-success completion, and resuming persisted work without replay."
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
