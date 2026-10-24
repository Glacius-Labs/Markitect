# Markitect's origin and rationale

This page owns the causal account of why Markitect was conceived and how its mechanisms are expected to address the originating problems. It organizes the product owner's [retrospective account of 15 October 2026](sources/owner-origin-20261015.md), earlier [product description](sources/owner-product-description-20261009.md) and [restatement of the foundations and goal on 17 October 2026](sources/foundations-and-trust-20261017.md). The sequence is the owner's account; the dates when its individual stages first occurred are not established here.

The [vision](../vision.md) owns the product thesis. The [register](register.md) owns decisions and their status, promises, assumptions and ideas. This explanation adds provenance and connects those entries; it does not create a second decision register, schedule work or claim that the method has demonstrated its benefits. Current behavior belongs to [Architecture](../architecture.md), implementation status to the [roadmap](../implementation-plan.md), and evaluation procedure to [Measurement](../guides/measurement.md).

## The originating project

The owner was developing a project with explicit Core, Module and Product boundaries. Its rules included Core depending on nothing, Modules depending only on Core, Products combining and adapting Modules, and Modules remaining independent of individual Products. These were the adopting project's architectural choices, not a required architecture for every Markitect project.

The owner also wanted to specify the working method, review policies, a documentation location for each use case, the skills belonging to a module, an Area, required documentation and its shape. The intended system therefore covered more than application behavior: architecture, development practice and the relationships among code, documentation and contributor guidance all mattered.

In the owner's experience, those representations repeatedly contradicted one another. Changes missed or violated rules, and documentation suffered from what the owner called "Bleeding". This account preserves that term without inventing a precise failure classification for it. Repeated AI consolidation found further drift, took a long time, and eventually left the owner unsure what applied and what was missing. These are reported observations about the originating project, not a measured failure rate for AI development in general.

## How the idea followed from the problem

| Step | Problem or realization | Resulting idea |
|---|---|---|
| 1. Describe a governed project | Architecture rules and working obligations must survive across code, documentation and Claude/Codex skills. | Make the project's intended world explicit, including engineering and working rules. |
| 2. Encounter conflicting representations | Several editable descriptions of the same rule drift, making its authority unclear. | Keep one authoritative definition of a fact and reconcile its realizations against it. |
| 3. Discover errors late | Missing relationships or inconsistent structure are noticed during implementation or expensive consolidation. | Borrow typing and compilation to reject expressible structural errors before execution. |
| 4. Ask one agent to remember everything | A growing context leaves relevant rules, files or consequences overlooked. | Use bounded responsibilities, delegated agents and accountable integration. |
| 5. Seek useful control | Manually reading every change and repeating global consolidation becomes the human bottleneck. | Maintain the model; use Apply, Verify, Cleanup and related operations to realize and inspect it. |
| 6. Anticipate the next bottleneck | If realization becomes dependable, interpreting new work and maintaining the model still consumes human attention. | Government could additionally delegate model maintenance and decisions within explicit authority. |

The base picture is:

```text
Canonical model -> Apply -> Repository
```

Here Apply is the abstract realization mechanism, not a claim that one current CLI command alone performs every step. Code, tests, documentation, configuration and agent guidance can be different realizations of the same intent. One authoritative definition does not require one physical file, nor identical text in every realization.

The Government extension is a separate transition:

```text
Concern or work item -> authorized model maintenance -> canonical model
```

Manager coordination realizes the accepted model. Government would additionally decide some changes to that model within a delegated mandate. The user's president/court analogy describes the desired division of work and escalation; it does not mandate a political structure or confer authority on an agent. [DEC-004](register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) and [IDEA-002](register.md#idea-002-government-delegated-model-maintenance) preserve that distinction.

## Problems the account identifies

| Problem | What was difficult for the owner | Existing connection |
|---|---|---|
| Unclear authoritative intent | Contradicting representations obscure which rule currently applies. | Canonical model and delegated realization: [DEC-001](register.md#dec-001-delegated-realization-is-central-to-the-product). |
| Missed continuing obligations | A change can work locally while violating another architecture, documentation or workflow requirement. | Reported omission problem: [ASM-001](register.md#assumptions-and-hypotheses); continuing specification: [CPT-004](register.md#cpt-004-the-specification-stays-effective). |
| Late discovery of structural problems | Relationships and invalid combinations emerge only while work is being performed. | Limited compiler analogy: [DEC-003](register.md#dec-003-the-compiler-is-a-borrowed-concept). |
| Concentrated cognitive load | One agent must hold the whole project's relevant rules and files in mind. | Bounded responsibility and integration: [CPT-001](register.md#cpt-001-federalism). |
| Repeated manual synchronization and review | Consolidation and reading changes consume time without giving a lasting overview. | Human focus and comprehensibility: [PRM-004, PRM-006 and PRM-007](register.md#promises). |
| Remaining model-maintenance work | New work items and technical findings still need interpretation and decisions about the intended system. | Delegated model maintenance: [IDEA-002](register.md#idea-002-government-delegated-model-maintenance). |

This account does not establish that forgetting is always more important than misunderstanding. Both an omitted relationship and a wrong interpretation can produce drift; [ASM-006](register.md#assumptions-and-hypotheses) remains a separate, untested assessment hypothesis.

## Concepts and their intended contribution

| Concept | What it means in this rationale | Why it might help; its limit |
|---|---|---|
| Canonical project model | The applicable project intent has an authoritative owner; realizations reference it rather than independently redefining the same rule. | Removes ambiguity about which statement governs. A wrong or incomplete model can still be implemented consistently. |
| Typing and structural compilation | Explicit identities, types and relationships can be validated before realization. | Moves expressible structural failures earlier. It does not prove arbitrary prose, model completeness or software correctness. |
| Explicit realization relationships and impact | Describe what belongs to what, including a rule's implementation, tests, documentation and guidance. | Makes declared consequences traceable. Undeclared relationships and code dependencies can still be missed ([ENH-001](register.md#enh-001-observed-code-dependencies-widen-impact)). |
| Managers and bounded delegation | Durable responsibility supplies a limited task and context; the parent remains accountable for integration. | Reduces reliance on one all-remembering "God Agent". A hierarchy alone does not ensure good task boundaries or correct composition. |
| Independent review and verification | Assess both individual contributions and their integration against the applicable obligations. | Supplies additional evidence and opportunities to find omissions. Reviewers can share blind spots; tests are realizations too ([CPT-002](register.md#cpt-002-tests-are-realizations-too)). |
| Apply, Verify and Cleanup | Realize intended changes, examine conformity and improve the repository within the applicable model. | Offers repeatable work against a known basis. These conceptual roles are not new CLI contracts or promises of unattended acceptance. |
| Government | Delegate selected model-maintenance decisions, preserve human reservations and report relevant matters. | Could reduce routine model work. Correct procedure does not prove that the selected intent is right; the model may grow harder to maintain. |

The goal is to use explicit structure and ordinary deterministic software where they fit, then give agents bounded interpretation and realization work. Deterministic and semantic checks have different jobs. The compiler analogy is borrowed from programming languages; it does not make a natural-language specification executable or complete merely by attaching types.

## Foundations restated on 17 October 2026

The marketing discussion also records the owner's [conceptual influences](sources/conceptual-influences-20261017.md) and [product-goal clarification](sources/product-goals-clarification-20261017.md). These supply additional provenance for the foundations below: lower cost is not the product goal, and routine human implementation review should become unnecessary through trustworthy realization.

The owner named the following influences and reaffirmed the goal: a clean, uniform and clear repository, without drift, forgotten rules, inconsistencies or contradictions; model first; trustworthy AI results through system and structure; ultimately modeling followed by AI realization that needs no routine human code review. The [source](sources/foundations-and-trust-20261017.md#owner-statement) preserves the wording. The table connects those influences to existing choices; it introduces no new roles, language semantics or delivery scope.

| Foundation | Intended contribution and current design connection |
|---|---|
| Four eyes and different perspectives | An independent reviewer asks whether the candidate satisfies its obligations; a parent's integration review asks whether the parts work together. Distinct questions matter more than Agent count. [CPT-001](register.md#cpt-001-federalism), [ENH-006](register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes). |
| Responsibility, delegation and escalation | Each Manager owns a bounded area and integrates its direct children. Questions and issues reach the responsible owner. The Orchestrator guarantees participation and the fixed process. [DEC-025](register.md#dec-025-package-managers-organize-realization), [DEC-039](register.md#dec-039-deterministic-manager-process). |
| Divide and conquer; abstraction and need to know | Decompose meaningfully, give specialists sufficient local detail, and let parents combine that detail into a broader view. Summaries and targeted reads connect breadth and depth; bounded context must not hide a needed contract. [DEC-028](register.md#dec-028-context-follows-need-to-know-and-starts-simple), [OQ-006](register.md#open-questions). |
| Model and projections; one source of truth | The repository's code, tests, documentation and configuration realize one intended world. They may have different shapes while expressing the same intent. A generated readable view is a projection with an exact rendering contract; Agent-written code needs semantic assessment. [DEC-001](register.md#dec-001-delegated-realization-is-central-to-the-product), [CPT-003](register.md#cpt-003-two-parts-of-the-promise). |
| Planning, system and structure; trust the process | Deterministic compilation and orchestration handle explicit structure and mandatory workflow. AI supplies bounded interpretation and realization judgment. Trust depends on useful findings, corrections and faithful results across the whole workflow. [DEC-039](register.md#dec-039-deterministic-manager-process). |
| Stakeholders and incentives | Different concerns should be represented by distinct questions and access to relevant evidence. Grounded objections, justified no-change and escalation are useful outcomes. Incentive design is an influence, not an implemented scoring or reward system. The first team remains Manager plus reviewer. [Organization and trust](reflections/organization-and-trust-20261016.md#limits-of-the-analogy-and-future-autonomy). |
| Type systems and compilation | Move structural failures into preparation: invalid types, references and combinations should fail before Agent work. Prose obligations still require interpretation and independent assessment. [DEC-003](register.md#dec-003-the-compiler-is-a-borrowed-concept). |
| Many capable lower-cost Agents | Bounded work and systematic collaboration should let models such as Luna, Sonnet or Haiku contribute instead of concentrating everything in a single expensive Agent. Suitability, total coordination cost and result quality remain questions for evidence. [DEC-007](register.md#dec-007-more-economical-models-means-capable-mid-tier-models), [ASM-003](register.md#assumptions-and-hypotheses). |
| Documentation-driven and specification-driven development; Clean Architecture and the Dependency Rule | Make intended meaning explicit, separate responsibilities and enforce dependency direction. The Markdown origin explains the name; canonical YAML preserves typed structure. These influences do not prescribe one software architecture for every adopting project. [Origin of the name](../vision.md#origin-of-the-name), [Go architecture](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/history/apply-design-before-dec-056/manager-apply-go-architecture.md). |

The goal of removing routine human code review concerns the owner's work. Independent Agent review remains central. Humans keep responsibility for intended meaning, architecture, unresolved decisions and the project's acceptance policy. The [vision](../vision.md#markitect-in-brief) owns this product goal, [PRM-004](register.md#promises) records it, and [Measurement](../guides/measurement.md#human-attention-and-delegated-work) owns how reduced human intervention is assessed. Neither a typed model nor agreement among Agents proves that the intended world is complete or right.

## Assumptions and hoped-for effects

The following are an editorial synthesis of why the owner expects the method to help. They are not new accepted register entries or measured results. References retain the existing statuses; the counter-observations make the reasoning open to correction without creating a new study or gate.

| Assumption or expected effect | Existing basis | Observation that would challenge it |
|---|---|---|
| One authoritative statement with maintained realization links reduces synchronization mistakes. | [DEC-001](register.md#dec-001-delegated-realization-is-central-to-the-product), [PRM-003](register.md#promises). | Contradictions still accumulate, or maintaining the model and links costs as much as the earlier consolidation. |
| Typing catches useful structural errors before agent execution. | [DEC-003](register.md#dec-003-the-compiler-is-a-borrowed-concept). | Most consequential failures pass structural checks and still require the same late global investigation. |
| Declared relationships provide useful scope and preserve continuing obligations. | [PRM-001](register.md#promises), [CPT-004](register.md#cpt-004-the-specification-stays-effective), [ENH-001](register.md#enh-001-observed-code-dependencies-widen-impact). | Relevant consumers or representations fall outside impact, or conservative scope repeatedly expands to nearly the whole project. |
| Bounded Managers and parent integration make complex work more dependable. | [ASM-004](register.md#assumptions-and-hypotheses), [CPT-001](register.md#cpt-001-federalism). | Handoffs introduce errors or coordination cost without improving coverage or integration quality. |
| Independent reviewers reveal mistakes the implementing agent misses. | [ASM-002 and ASM-005](register.md#assumptions-and-hypotheses). | Reviewers repeatedly accept the same omissions or wrong interpretations; extra calls add no useful evidence. |
| Bounded context lets capable, less expensive models handle routine work efficiently. | [ASM-003](register.md#assumptions-and-hypotheses), [PRM-005](register.md#promises). | Rework, review, integration and model upkeep consume any savings. Lower total cost is a possible benefit, not the primary success condition. |
| Model maintenance and explicit evidence reduce the need to inspect every routine PR while preserving control. | [PRM-004, PRM-006 and PRM-007](register.md#promises). | The owner still needs exhaustive diff review, cannot identify unverified obligations, or trades that work for equally expensive model supervision. |
| Government can further reduce routine attention without degrading intent. | [IDEA-002](register.md#idea-002-government-delegated-model-maintenance). | Autonomous decisions misinterpret needs, exceed delegation or create more rules, review and escalation work than they remove. |

The existing [measurement procedure](../guides/measurement.md#human-attention-and-delegated-work) compares quality, omissions and human attention as well as the full cost of setup, modeling, implementation, review, integration and recovery. Success cannot be inferred from more agents, more documents, green self-authored tests or a well-formed decision receipt. Current evidence remains in the [vision's evidence summary](../vision.md#current-basis-and-unproven-benefit) and the source-bound [validation records](../validation/README.md).

## What this account settles and leaves open

The owner confirms the reason for the product: maintaining a clear intended world and using structure, typing, delegated responsibility and checks to make its realization more controllable. Government follows as a possible extension to the maintenance of that world. The original project rules explain the need; they are not prescribed Markitect vocabulary.

The [language v1](../design/language-contract.md) and [Apply design](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/README.md#apply-design) own the selected language and realization rules. A project still chooses its model granularity, intended relationships, implementation freedom and acceptance policy. The owner's restatement clarifies the motivation and goal; it adds no runtime mechanism or authority for autonomous model changes, and it does not establish product superiority over conventional agentic development.
