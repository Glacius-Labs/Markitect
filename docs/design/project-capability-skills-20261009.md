# Capability skills and native project entrypoints

## Accepted Init design — 17 October 2026

**Status:** accepted by the owner in the "Usage, Skills, UX" discussion on 17 October 2026. Init is to install six operation skills for both Codex and Claude Code, short native entrypoints and guidance maintained from one shared source. This section owns the current design. The [9 October record](#historical-design--9-october-2026) below preserves the earlier audit and proposals; its skill set and narrower meaning of Apply are superseded here.

**Scope:** documentation only. This decision does not implement Init, install skills, change the canonical language or the frozen Apply core, or authorize execution. Installed instructions must describe only the capabilities supported by their selected Markitect version. The [current workflow](../project-workflow.md) owns executable behavior; the [delivery plan](../implementation-plan.md) owns delivery scope. The target skill set is not evidence that six corresponding CLI verbs exist.

### Owner source

The owner first specified the normal entry through a coding agent and the Init requirement:

> Ich möchte Markitect normalerweise über meinen Coding-Agenten bedienen.
>
> Wir wollen beim Init direkt die entsprechenden Codex und Claude Skills haben, damit sie jeweils wissen wie hier zu arbeiten ist und wie man das Modell pflegt

After the six-skill proposal, the owner accepted it and requested this record:

> Das klingt gut. Halte diese Aufteilung in unseren Docs fest, inklusive der sechs Skills, der Inhalte von AGENTS.md und CLAUDE.md und der gemeinsamen Pflegequelle. Noch keine Implementierung.

The owner subsequently accepted the [existing-project Init flow](#existing-project-init-flow) described below:

> Das klingt gut. Halte diesen Init-Ablauf in unseren Docs fest, insbesondere die Trennung zwischen technischer Einrichtung, Modellübernahme und meinen Entscheidungen. Weiterhin keine Implementierung.

After discussing the first Apply of an adopted project, the owner accepted the workflow and reuse of the existing finding structure:

> Das klingt gut. Halte diesen Apply-Ablauf in unseren Docs fest, inklusive der Übergabe der Extract-Befunde über die vorhandene Befundstruktur und dem begründeten Beibehalten korrekter Teile. Weiterhin keine Implementierung.

The owner then accepted the first business-rule change in that working day:

> Das klingt gut. Halte diesen Ablauf für die erste fachliche Änderung in unseren Docs fest, inklusive der Modelländerung von fünf auf sieben, der anschließenden Realisierung durch Markitect und meiner Entscheidungspunkte. Weiterhin keine Implementierung.

On 18 October 2026 the owner accepted the subsequent defect-repair workflow:

> Das klingt gut. Halte diesen Fehlerbehebungsablauf in unseren Docs fest, inklusive unverändertem Siebenerlimit, Befundübergabe, unabhängigen Nachweisen und meinen Entscheidungspunkten. Weiterhin keine Implementierung.

### The six installed skills

Both clients receive the same skill names, responsibilities and authority boundaries. Provider-specific packaging does not create different project rules.

| Skill | Trigger | Responsibility and boundary |
| --- | --- | --- |
| `markitect-init` | Set up Markitect or update the project integration. | Inspect the existing project, perform supported setup and install or update both native entrypoints and skill sets. Preserve custom guidance and unrelated files. Setup does not establish that an existing project is correctly modeled or realized. The installed skill also serves later setup work; initial bootstrap cannot depend on a skill that has not been installed yet. |
| `markitect-configure` | Select a provider, change capacity or configure project checks. | Maintain supported project and execution settings, validate profile bindings and explain their provenance. Binding project rules belong in the model. This task does not implicitly authorize changes to personal accounts, credentials or security preferences. |
| `markitect-design` | Explain the model, change intended behavior or propose improvements. | Find and understand the relevant Definitions, clarify consequential missing intent and make authorized model changes with their relationships and constraints. Use structural checks and refresh readable views through their supported writer. Suggestions remain proposals until adoption is authorized. This skill does not itself implement application artifacts. |
| `markitect-check` | Check model structure or investigate compiler, registry or file-accounting diagnostics. | Run the structural preparation, explain diagnostics and repair them within the assignment's authority. Distinguish compilation, references, registry and complete file accounting from semantic implementation correctness. An assessment-only request grants no repair authority. |
| `markitect-apply` | Realize the model, repair a deviation without changing intent, inspect current progress or continue an assignment. | Operate the complete supported Apply workflow: check prerequisites, commission execution, observe progress, relay questions and use supported continuation or recovery. Markitect owns realization, independent review, integration, verification and guarded publication. Status inspection starts no new work; the outer agent does not build another Manager scheduler or bypass publication. |
| `markitect-verify` | Assess whether the realization meets the model, especially when asked to check without repairing. | Examine the actual realization and evidence against the selected model, run authorized checks and report deviations, uncertainty and missing evidence. Do not repair automatically or claim full semantic verification from a few passing checks. This is not an extra approval institution beside Apply's independent review. |

`check` assesses structure; `verify` assesses realization and evidence. Skill names describe user tasks, not a promise of matching CLI verbs. Where a necessary operation is unsupported, report that limit instead of inventing a command, execution result or successful completion.

The [modeling skill guidance](modeling-skill-guidance.md) owns the accepted 19 October 2026 content for `markitect-design`: conversational drafting without requiring YAML inspection, proposal and acceptance boundaries, delegated decisions, architecture and organization principles, and good/bad boundary examples. It is documentation only, not an implemented or installed skill, and does not change the six-skill split.

The outer coding agent chains relevant skills within the existing assignment. For the library's five-to-three limit change, it uses `design` and `check`, commits the authorized model revision within its grant, then commissions `apply`. Existing loans stay valid and returnable because the model says so, not because a skill adds that policy. Apply itself changes only artifacts and does not commit, merge or push; the owner accepts the published result under [DEC-035](../concepts/register.md#dec-035-committed-model-in-owner-commit-accepts-the-result).

The [first business-rule change](canonical-model/manager-apply-library-flow.md#first-business-rule-change-from-five-to-seven) walks through the same handoff for the working day's increase from five to seven: the coding agent maintains the intended model, Markitect realizes it, and the owner decides only unresolved meaning or missing authority under the existing acceptance policy. This scenario does not replace the delivery demonstration's five-to-three change.

The subsequent [eighth-loan defect repair](canonical-model/manager-apply-library-flow.md#defect-repair-an-eighth-loan-despite-the-seven-loan-limit) keeps that model unchanged. The coding agent hands the reported deviation to Apply for investigation, repair and independent evidence; an assessment-only request uses `verify` without repair. No artificial model edit, additional skill or mandatory separate Verify round is needed for an already authorized repair.

One ordinary request must not become six human prompts or repeated permission requests for already authorized steps. Status and continuation belong to the ongoing Apply workflow. Separate `extract`, `cleanup`, `refactor` or `bughunt` skills wait until their actual operations justify independent instructions; suggestions are already discoverable through `design`. This decision does not add those operations to the first Apply.

### What AGENTS.md and CLAUDE.md contain

The Markitect-managed part of each native entrypoint gives only the orientation needed before any project task:

- State that the project uses Markitect and that its canonical model owns binding intent.
- Identify the actual selected model, project overview and configured Markitect invocation. Init fills observed project bindings instead of copying historical example paths.
- Distinguish a change of intent, which needs model work, from a realization defect against unchanged intent.
- Route requests briefly to the six skills above. Let the coding agent choose and chain them.
- Continue already authorized work; ask a concrete question only when material intent or authority is missing. A discussion or suggestion alone is not authorization to change the model.
- Distinguish model change, structural checks, realization assessment, publication and owner acceptance. Point to the applicable project policy rather than duplicating it.

Detailed YAML rules, command options, configuration schemas, review evidence and recovery procedures belong in task-specific skill references and their existing owners. Load only the relevant references. The entrypoints must not become a second manual or an independently maintained copy of project rules. For example, the library's loan limit and required architecture stay in the model. This follows [DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model).

Codex uses `AGENTS.md` with `.agents/skills/<skill-name>/SKILL.md`; Claude Code uses `CLAUDE.md` with `.claude/skills/<skill-name>/SKILL.md`. Keep supported provider-specific metadata distinct while retaining the shared meaning. Existing user-written entrypoint text and unrelated skills remain intact. These are instructions for the outer project agent; they neither replace the inner Managers' compiled context nor prove that an independent reviewer ran.

### One maintained source, two native outputs

The Markitect integration/onboarding area owns one versioned, provider-neutral content set for these skills and the managed entrypoint sections. Provider mappings derive both native outputs deterministically from that source; neither provider output is separately authored. This document owns the agreed split. The implementation location and generation mechanism remain implementation work, not an additional design decision made by this record.

Shared content references the actual owning contracts for model authoring, configuration, structural preparation and Apply. It does not become a second owner of those contracts. Each installed set is tied to the selected Markitect version and actual project bindings. Product-contract changes require review of the affected shared instructions; Init or an explicit integration update installs the corresponding supported version. A model edit does not independently rewrite the native guidance, and a binary change does not silently replace it.

Keep updates idempotent, source/target-bound and explicit about ownership: preserve custom text and unrelated skills, refuse stale writes and surface unmanaged same-name collisions instead of overwriting them. No personal provider configuration is changed merely to make the two clients look alike.

Future validation must establish valid native files and resolving references, consistent meaning in both outputs, preservation of custom content, repeatable updates, collision and stale-source handling, and actual selection of the right instructions for ordinary work, structural diagnostics, assessment-only requests and continuation. File existence or matching generated bytes alone does not demonstrate agent behavior, complete Apply or owner acceptance. This is a validation requirement for later implementation, not a run performed by this documentation change.

### Existing-project Init flow

**Status:** owner-accepted design, 17 October 2026; no implementation or execution is supplied by this record. Init prepares the technical integration and reports what remains before adoption is complete. The outer coding agent interprets project evidence and works with the owner on the model. These are distinct activities even when one user request authorizes both.

The concrete starting point is a small existing library application with Git history, code, tests, a README and custom agent guidance, but no canonical Markitect model. The owner asks: "Set up this project for Markitect. Preserve the existing application and our own guidance."

#### Observe the existing state

| Observed input | Permitted conclusion |
| --- | --- |
| Repository root, revision and local changes | Bind setup to the observed basis and preserve unrelated work. Several plausible project boundaries need a concrete selection. |
| Existing Markitect files | Inspect or update the existing integration instead of initializing it again. File presence alone proves no successful setup or adoption. |
| AGENTS.md, CLAUDE.md and native skill files | Preserve custom content, update identified Markitect-owned content and surface unmanaged same-name collisions. |
| Project manifests, directories and declared build/test commands | Prepare concrete technical entrypoints. Finding a command is not evidence that it ran or passed. |
| Existing documentation, tests and code | Give the coding agent source material for model work. Location, frequency or observed behavior does not establish intended meaning. |
| Explicitly configured execution profiles | Reuse suitable, authorized bindings and report missing choices. Do not infer provider authority from the outer coding agent's account. |

Technical observation and semantic judgment stay distinct. "The code admits five loans" is an observation, not proof that five is the intended rule. The coding agent keeps source evidence, interpretation, uncertainty and proposed intent distinguishable.

#### Perform the authorized technical setup

Init installs or updates the six skills and short entrypoints for both clients from the selected version of the shared content. It uses actual project bindings and reads the written outputs back. Repeated setup leaves an unchanged integration unchanged; source/target changes invalidate an older write plan. Existing application code, custom guidance and unrelated skills are preserved.

If no accepted model exists, the entrypoints state that gap rather than referring to a fictitious selected model or declaring the project ready for Apply. Technical integration may be complete while model adoption remains open. Setup alone starts no provider session or Apply and grants no additional execution authority.

Existing configuration and applicable product defaults may supply omitted technical settings with their provenance. They do not resolve missing business intent, provider authorization or a monetary spending ceiling. The [configuration contract](canonical-model/manager-apply-configuration.md#strategy-defaults-and-their-sources) owns those defaults; Init does not introduce another set or silently widen existing grants.

#### Prepare and adopt the model through the coding agent

Model adoption is a separate `markitect-design` task in the same conversation. "Set up Markitect and prepare a model proposal from our existing requirements" already authorizes preparation of that proposal; no extra confirmation is needed for the same scoped step. A setup-only request does not authorize silently adopting existing behavior as binding intent.

The agent examines relevant requirements, documentation, tests and code, proposes a coherent model and surfaces consequential conflicts. For example: "The README and application allow five loans, but a test expects three. Which limit should apply?" It groups related questions with their sources and a justified recommendation. The owner decides meaning that the assignment or authoritative sources do not already settle. An existing bug must not become a requirement merely because extraction observed it.

The [source-grounded model-adoption workflow](canonical-model/commands.md#model-adoption-without-turning-defects-into-rules) owns the detailed procedure. Under its [accepted completion criterion](canonical-model/commands.md#completion-and-adoption-with-known-deviations), a reviewed model proposal may be adopted with adequately supported known realization deviations. Those findings remain visible, Verify keeps its actual result, and artifact repair is separate work; unresolved material intent or evidence gaps are not waived.

The [first Apply after adoption](canonical-model/manager-apply-library-flow.md#first-apply-after-model-adoption) then reconciles the complete existing realization, preserving justified correct parts. Its [finding handoff](canonical-model/manager-apply-execution.md#extract-findings-at-the-first-apply) uses the existing issue structure with current applicability and evidence assessment; adoption supplies neither artifact approval nor authority to close those findings.

Adoption also needs explicit responsibility and exact file accounting under [DEC-006](../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored). The agent can prepare registry associations within authorized modeling choices; the owner need not approve every registry line. Unclear responsibility boundaries and genuine exceptions need a decision. Init must not ignore unknown files or invent ownership merely to produce a passing report. A registered file is accounted for, not thereby semantically correct.

#### Ask only for decisions or missing authority

| Situation | Owner decision |
| --- | --- |
| Several plausible projects or repository boundaries | Select the project to adopt. |
| Conflicting or missing requirements | Choose the intended rule from a source-bound question. |
| Unclear responsibilities or genuine exceptions | Assess the concrete proposed boundary or exception. |
| Missing provider/model selection or execution authority | Supply the needed choice or grant, unless already settled by valid configuration and existing authority. |
| An unmanaged same-name skill or conflicting custom guidance | Resolve the specific collision without losing the original content. |

The agent handles technical details within clear instructions. Do not turn setup into a questionnaire for every file, demand repeated authorization or require human coordination of routine steps. Do not substitute a product default for a genuinely reserved decision.

#### Report setup and adoption separately

The final concise account states what was actually configured, what model work and file accounting remain, which decisions are needed and which operations actually ran. It distinguishes integration readiness, structural results, semantic assessment and execution readiness without creating another dashboard or editable requirements catalogue. For example:

> Codex and Claude integration configured; custom guidance preserved. Model adoption remains open. Three conflicting requirements need a decision. Existing files are not yet fully assigned to responsibilities. No Apply was started.

Init gives the owner a usable entrypoint; the coding agent conducts model adoption with the owner. One assignment may include both, but technical setup never silently accepts a model, certifies the existing realization or repairs the application. The current executable surface remains owned by the [project workflow](../project-workflow.md); earlier adoption designs below and elsewhere retain their historical scope.

## Historical design — 9 October 2026

The remaining text is the dated audit and earlier design. Its instructions and execution allowances applied to that historical assignment and grant no current work authority. Use the accepted section above for the current Init split.

Direct user request, 9 October 2026: improve Markitect Skills, CLAUDE.md, AGENTS.md and native installation guidance, using several skills for meaningful capabilities/work steps. This extends only the active Designer native-working package. Other owners, studies, automation, Main merge and releases remain stopped. The existing one native product-proof allowance is unchanged; no extra run or budget is implied.

## Source audit before changes

Audited source: Design checkout b61eb445fdceb851640d1fd8c7084a8b7b664403, internal/host/projectonboarding/render.go and onboarding.go, plus docs/project-workflow.md. The actual product checkout has no CLAUDE.md; this audit evaluates the generated adopting-project CLAUDE.md template and its shared workflow, not a nonexistent installed file. Product-repository authoring/engineering-change skills serve another purpose and should not be replaced indiscriminately.

The generated Claude and Codex entries are identical brief paragraphs. Each installs one markitect-model-first skill which directs the agent to load the entire mixed setup/exploration/Brownfield/readiness/delivery/recovery workflow. Commands and guarded lifecycle/authority distinctions are mostly concrete. The main weaknesses are coarse discovery, loading irrelevant stages, terse catchall metadata, and the need to align instructions with the actual forthcoming native worker mode. Existing preview/digest binding and marked-block merging already preserve unmanaged entrypoint prose and reject stale writes.

Editorial quality assessment of the generated entry + referenced workflow (not a runtime score):

| Criterion | Assessment | Reason |
| --- | --- | --- |
| Commands/workflows | 15/20 | Numerous source-backed commands, but no capability-local operational entry |
| Architecture clarity | 15/20 | Model/Manager/lifecycle boundaries explained; outer project agent vs native worker routing needs explicit treatment |
| Non-obvious patterns | 10/15 | Acceptance, exact digests and durable recovery covered; no focused troubleshooting/selection |
| Conciseness | 5/15 | Short entry delegates to a large workflow covering every stage |
| Currency | 15/15 | Audited against the current source; future native-working capability is not claimed |
| Actionability | 10/15 | Concrete command forms, but broad skill and multiple placeholder records increase navigation burden |
| Total | 70/100, B | Editorial assessment only; no agent behavior or installed release validated |

## Proposed product structure

The user's later clarification supersedes the original six broad capability groups. Use eight operational skills, named after the concrete work the agent should perform, with suggest/configure grouping delegated below. The latest direct user decision withdraws the old markitect-model-first compatibility router requirement: remove it if its only purpose is supporting the old entry. Root entrypoints identify the selected project model, route directly to the appropriate skills and state essential shared boundaries without repeating manuals.

| Skill | Trigger and outcome |
| --- | --- |
| markitect-init | Initialize a new project or complete supported setup/onboarding in an existing project; bind the actual executable/runtime and install native guidance. Existing projects are inspected rather than reset. Preserve caller budgets and personal/provider configuration. |
| markitect-extract | Extract a proposed canonical model from existing source through supported Brownfield inference/adoption. Bind actual source and scope; retain uncertainty and distinguish observed code from intended/accepted behavior. Initial adoption remains model-only. |
| markitect-design | Translate ordinary Work Items, bugs, concepts, rules or renames into intent/decisions and canonical YAML changes, impact and scoped readiness/planning. Check and repair the draft within delegated task authority. A design proposal is not accepted implementation. |
| markitect-implement | Carry ordinary implementation Work Items through the supported model-first workflow with real Managers and candidate workspaces. Invoke design/check/verification/application as needed to finish already authorized end-to-end work; do not stop merely because another skill covers the next step. Resume the persisted run after interruption. |
| markitect-cleanup | Improve/refactor the realization while preserving accepted model behavior, public contracts, required checks and ownership. Use the actual cleanup operation; a semantic requirement change is design work. Run appropriate verification before application. |
| markitect-verify | Assess actual realization/candidate against the accepted model, rules, contracts and configured checks; collect real test/review evidence and identify drift or missing evidence. Use supported verify/reconcile paths; report gaps without automatic semantic repair when only assessment was requested. |
| markitect-apply | Inspect and apply an already verified candidate using real preflight, freshness and branch/source/candidate/verification bindings. Preserve required reviews. Apply neither invents an implementation nor implies merge/deployment/release authorization. Explain any difference between the task operation called apply and guarded candidate publication instead of conflating them. |
| markitect-check | Run the structural model/compiler checks for schema, references, ownership and inventory/coverage; interpret and repair actionable diagnostics within task authority, then recheck. Passing structural checks does not prove semantic implementation correctness. |

Setup belongs to init, model editing/planning to design, audit/reconcile to verify, and status/resume/known-cause repair to the relevant ongoing implement/cleanup/apply workflow or a shared supporting reference. Do not create extra top-level recovery/plan/compile skills merely to rename these operations. Structural check and realization verify intentionally remain distinct. Standalone apply is useful for explicit candidate-publication requests; implement remains the natural default for an ordinary authorized Work Item.

The contributor still gives an ordinary prompt such as "Implement this work and merge it back to main." The agent selects and chains the applicable skills itself, respects actual merge authorization and project checks, and completes the task. Skill splitting must not turn one request into eight human prompts, add approval rounds or impose a fixed sequence on unrelated tasks.

Descriptions must discriminate these intents and enable normal implicit discovery. Keep bodies concise with concrete local references, not generic coding advice. Use supporting references for detailed JSON shapes and stage-specific commands. Avoid copied provider-independent manuals in every skill or accidental cycles between skills. The canonical maintained source owns each shared fact once; provider outputs must be generated deterministically from that source.

Suggested native entry content, refined to the actual implementation:

> This repository uses Markitect's selected .markitect/project.yaml model. For ordinary work, select the relevant local Markitect operation skill: init, extract, design, implement, cleanup, verify, apply or check. Persist intent and canonical model changes before delivering implementation. Repair actionable model/check diagnostics within your delegated task and continue the existing workflow. Ask only for material unresolved intent or authority. Use the actual installed command/runtime, real checks and guarded Apply; recover existing work instead of replaying it.

This is a content proposal, not a fixed wording test. The init entry must also explain how to bootstrap when the selector or installed skills do not yet exist.

## Native integration and migration

Generate Codex skills in the supported repository-local .agents/skills layout and Claude skills under .claude/skills. AGENTS.md and CLAUDE.md are concise provider-native entrypoints. Add .codex files only when the actual supported Codex configuration requires them, not to create a decorative duplicate or change personal account/security preferences. Keep ordinary project build/test/domain instructions owned by the adopting repository, rather than inventing generic commands.

Outer project agents may make delegated canonical changes through the guarded model-edit workflow. Inner Manager workers operate on their declared candidate responsibility; native helpers are ordinary scoped helpers, not duplicate Host-scheduled Managers. Do not teach implementation workers to bypass ownership, mutate the control plane or write adopting sources before guarded Apply. Instructions must match the real working mode and describe unsupported behavior honestly.

Preserve user-written entrypoint prose and unrelated skill files. Onboarding updates must remain previewed, digest-bound, idempotent and stale-safe. Remove known Markitect-owned obsolete router assets and update maintained skill metadata along with bodies so outdated catchall descriptions do not remain. Treat conflicting unmanaged same-name skills or ambiguous ownership explicitly; do not overwrite them based only on the name. Maintain one current workflow without a compatibility-only router. References must resolve from the actual installed skill directory, including on Windows/CRLF.

## Verification and handoff

Implement through Designer's canonical product sources and regenerate outputs. Include focused validation of provider parity, native skill discovery metadata, valid frontmatter, relative reference resolution, new and existing-project onboarding, preservation of custom entrypoint prose/skills, metadata migration, repeated installation, target collisions and changed-source rejection. Test meaningful observable invariants, not copies of prose or exact headings. Include a small source-backed intent-routing check and walkthrough for a normal Work Item, a compiler diagnostic and an interrupted run. Do not use private study fixtures or claim simulated walkthroughs as real agent performance.

Existing native working proof can use the resulting installed guidance if implementation order permits; do not restart a consumed proof just for this instruction change. Return exact generated paths, source/check outcomes and capability limits. No automatic Main merge or release. The user has already requested these improvements; proceed within this scope without adding a general approval round for each document.

## Latest user addition: Designer owns the final skill boundaries

The user accepts the operational direction and suggests two further capabilities: suggest for proposals to improve the canonical model, and configure for configuration work. Designer is explicitly authorized to decide whether these deserve their own skills or belong in existing skills, based on distinct triggers, useful instructions and overlap. The eight names above are a working proposal, not an immutable mandated count.

Suggest should produce justified model-improvement proposals, with expected benefit, affected responsibilities/contracts and tradeoffs. A request for suggestions alone does not authorize silently accepting or applying a changed model. Configure concerns supported project/runtime/check/provider bindings and their validation, including existing-project updates; it must preserve task authority and existing budgets, and does not imply global account/security changes. If these remain within design/init, make both intents discoverable and document the routing. Explain the final grouping briefly in the ordinary completed handoff. No extra phase, native proof or study is authorized by this clarification.
