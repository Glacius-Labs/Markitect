# Product vision

This is the canonical owner of Markitect's product thesis, intended division of responsibility and long-term operating model. [Architecture](architecture.md) owns technical boundaries, the [engineering constitution](engineering-constitution.md) owns kernel invariants, the [roadmap](implementation-plan.md) owns released/source/planned status, and [Operating methodology](operating-methodology.md) owns how the intended process works, and [Measurement](measurement.md) owns staged readiness and how to test benefits. This vision does not authorize a feature, weaken an acceptance gate or establish a measured outcome.

## The thesis

> **Define the desired project world once. Reconcile its representations without losing intent.**

The primary goal is quality and consistency during autonomous engineering: fewer forgotten obligations, overlooked consequences, conflicting representations and ignored rules. The project ontology expresses accepted concepts, relationships, purpose and constraints; the governed repository is its representation. A change to that world should drive impact, fanout, reconciliation and independent verification. The [operating methodology](operating-methodology.md) defines that cycle.

Human attention is scarce. Better independent evidence and explicit delegation should reduce routine supervision while preserving human ownership of meaningful decisions. Saving tokens or elapsed time is a secondary possible benefit, not the primary purpose or a prerequisite for the idea to be useful.

Markitect's working assumption is that AI implementation capacity will grow and become increasingly parallel, while human engineering judgment remains scarce. If humans must implement, synchronize and review every routine change, that attention becomes the bottleneck. Architecture matters more as implementation becomes cheaper: increasing output without explicit boundaries can also increase accidental coupling, inconsistent policy and architecture drift.

These are product assumptions to test, not economic or productivity conclusions established by this repository. Agent capability, task complexity, verification coverage and model-maintenance cost can change the outcome.

Humans should spend more attention defining goals, concepts, architecture, invariants, policies, processes, responsibilities and allowed degrees of freedom. AI agents should perform as much of the remaining implementation, testing, refactoring, documentation, migration, release work and routine maintenance as their granted authority and available evidence permit.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

## Product definition

**Markitect is the canonical model of a project's intended world and the responsibilities for realizing it.** A project organizes concepts, rules, architecture, workflows and decisions into recursively managed vertical slices. Its model also declares expected artifacts, file responsibility, integration contracts and checks. The model lives under `.markitect/` in the adopting repository; the generated readable project document is a view of those sources, not a competing owner. Provider, editor and file-format choices are implementation details unless the project deliberately makes them part of its intent.

**Owners change the model. Managers coordinate realization.** The target flow is:

```text
explore goals, constraints and open decisions
    -> accept and commit the project model
    -> derive affected Managers and bounded work
    -> delegate implementation within declared freedom
    -> integrate child changes and run declared checks
    -> Verify the full required Manager tree
    -> guarded Apply, bounded repair or owner decision
```

The committed model revision is Markitect's accepted repository specification for that revision. Drafts, distillation reports and uncommitted edits remain proposals. Git commits and digests bind bytes and history; they do not authenticate who approved the model or prove that a human reviewed it. Projects define their decision and acceptance process outside that machine identity boundary. Brownfield work can use existing artifacts to inform a proposal, but inference never grants the proposal authority. Owners review and correct the model, commit the selected specification, then map artifact scopes and assess existing realizations. Unmodeled, excluded, externally owned and uncertain paths remain explicit. A partial model must not imply repository-wide governance.

The product direction is an executable engineering-governance workflow for AI-assisted software development. Readable documentation, native agent instructions, bounded context, implementation tools and checks serve the model-first process. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is faithful, higher-quality autonomous engineering with architectural authority remaining human-owned. More work per unit of human attention is a consequence to measure alongside quality, reliability and model upkeep.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies, processes and Managers; define implementation freedom and acceptance requirements; maintain the project's approval process and resolve decisions outside delegated authority. |
| Markitect | Compile the committed project model and fixed repository snapshots; derive bounded context and impact; route Manager work; validate candidate structure and declared evidence; preserve provenance and guard writes. |
| AI agents | Explore and propose intent, implement assigned responsibilities, test and integrate within granted boundaries, report evidence, and raise unresolved decisions. An agent report is not owner approval. |

For governed changes, the intended flow is committed accepted model → relevant Manager context and bounded checks → materialization or repair of declared artifacts → full Verify and observed evidence. A real intent change updates its canonical owner before implementation. When intent is unchanged, a bug or stale artifact is repaired against that same model; no artificial model edit is needed. Exploration, distillation and other proposals remain drafts until the owner chooses to commit the revised model. A commit is the specification basis, not proof of who reviewed or approved it.

This operating model is not a claim that the published release supports the new workflow or that current source closes every user journey. Current source includes durable Explore records, scoped readiness and structure acknowledgements, structured Brownfield proposal/adoption sessions, Manager execution, integration, full verification and guarded Apply. The composed `project deliver` operation advances and resumes that bounded proof. These protocol surfaces do not establish the quality of a natural-language interview, semantic correctness, human acceptance, or a successful outcome for every adopting repository. Configured checks establish only their declared scope. Agent instructions and scoped execution are not an OS sandbox and do not prevent an independently writable process from changing repository files.

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## What executable governance means

One canonical owner exists per modeled engineering fact or representation decision. Rationale and guidance may be prose inside the project model. Unadopted narrative, vendor material and specialist observations remain outside accepted intent until explicitly incorporated. The project tree makes responsibility, artifact relationships and decision scope visible; it does not infer source-code semantics or turn every repository file into a model fact.

The new structural Core does not contain an engineering assertion DSL. Checks, policies and assurance live above Core; the historical finite Domain/policy contract remains supported through compatibility. Project-owned checks and target tools can provide specialized implementation evidence. Rule and Workflow prose remains instruction and rationale where no executable assertion exists; storing it in a typed resource does not enforce the process or prove that an agent followed it.

Context helps agents find applicable intent; impact helps reviewers understand declared consequences; versioned packages make architecture evolution explicit; narrow exceptions record deliberate deviations; reconciliation can align owned representations. None proves that all relevant intent was modeled, that an agent obeyed it, or that arbitrary application code is correct.

Continuous autonomous agents are the desired consumer operating model. They may call bounded Markitect operations from their own runtime, orchestration or CI. An Executor produces a candidate representation using appropriate tools, including deterministic renderers when useful: permitted implementation freedom must be explicit and project-owned checks must state what they establish. This does not require a continuously running Core, hidden policy activation or a background mutation controller. Current Observe/Plan/Apply/Verify trust boundaries remain in force; Apply still requires explicit authority and exact input-bound plans. A plan digest binds inputs and operations; it does not authenticate an owner or prove prior approval.

## Desired operating model

Humans should eventually review decisions at the level of architecture, ambiguous requirements, unresolved policy conflicts, deliberate exceptions and exceptional risk. Agents should handle ordinary implementation details within the project's declared freedom.

An illustrative future summary might say:

```text
Agents completed:                 47 changes
Automatically verified:          42
Need architectural decision:       3
Need explicit policy exception:    1
Ambiguous requirement:             1
```

These invented numbers describe an aspiration, not a current dashboard or measured run. The aim is to focus human attention on the five decisions. “Verified” would still mean only the declared checks and evidence coverage, not complete software correctness or automatic acceptance. The project must explicitly define its delegation and acceptance policy.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, Rules, Skills, Workflows and agent instructions. Its direction is to turn informal engineering knowledge into structured, versioned architecture and policy, then into an executable engineering system. The name records that origin; it does not require Markdown as canonical syntax. Current typed definitions use YAML, while human-readable prose and explicitly selected projections remain useful.

## Feature decision filter

Ask: **Does this improve fidelity to accepted intent and reliable autonomous reconciliation inside explicit boundaries, with enough independent evidence to reduce routine human supervision?**

A proposal should identify the human activity it aims to remove, its canonical owner, the agent's remaining freedom, the exact evidence/check boundary and the new authoring or maintenance cost. Deterministic checks, task context, impact, policy evolution, exceptions, discovery and reconciliation can support this goal, but each needs a concrete workflow and evaluation.

Reconsider proposals that introduce another manually synchronized truth owner, hide architectural decisions in inference, specialize Core for one provider, or add machinery without evidence that coordination gets easier. More generated text, adapters, language operators or agent throughput alone do not demonstrate progress. This filter does not override the constitution's bar for new semantics or authorize a new feature wave.

## Current basis and unproven benefit

| Category | What is established or intended |
|---|---|
| Product direction | A single model-first user workflow centered on the recursively managed `.markitect/` project tree. The [roadmap](implementation-plan.md) records exact source and release status. |
| Current source | The `project` CLI supports model, Manager, artifact, coverage, operation, briefing, onboarding, durable Explore/readiness, structured Brownfield refinement, and composed bounded delivery. See the [workflow](project-workflow.md), [native work-item delivery checklist](design/project-world/native-work-item-delivery.md), and [validation record](validation/project-operations-2026-10-09.md); checklist requirements and protocol tests are not user-journey or benefit proof. |
| Published compatibility | v0.14.1 preserves the earlier Project/Domain CLI and bundles the experimental canonical Projection alpha. These versioned contracts and evidence remain available to existing projects; they are not a second active product UX and are not included in the current model-first source command surface. See the [production assessment](production-assessment.md#v0141-published-release). |
| Acceptance boundary | A committed model revision is the accepted specification used for that revision. Commit/provenance values bind data but do not authenticate a human decision. Passing checks, generated docs and agent reports are evidence with declared scope, not approval. |
| Aspirational operation | More parallel Managers, less routine supervision, architecture-level escalation and sustainable governance at greater scale. No universal autonomous acceptance or OS isolation is claimed. |
| Unproven benefit | Reduced human intervention, lower total maintenance cost, fewer missed obligations or defects, better sustained autonomy, token savings and market demand remain unmeasured. Historical comparisons are preserved, including their negative or inconclusive findings. |

The [real-code adoption pilot](validation/real-project-adoption-pilot.md), [AGENTS.md comparison](validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

The central evaluation question is: **Does ontology-driven reconciliation preserve intent and project quality more reliably than conventional agentic work, and thereby support less routine human supervision?** [Measurement](measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative.

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.


## Research refinement

The [ontology, projection and assurance thesis](research/README.md) is owner-supplied research orientation. Its distinction between semantic authority, implementation freedom, independent evidence and integration obligations guides experiments. The [alignment assessment](design/projection-assurance-direction.md) identifies what this source slice supports and what remains research. It neither expands Core by naming concepts nor changes measured evidence into a benefit claim.
