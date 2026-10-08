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

**Markitect is the canonical desired-state model of an engineering system.** Human owners define durable, provider-independent meaning when it is needed: goals, rules, processes, responsibilities, invariants and other concepts that still make sense if today's providers and file formats disappear. This is a test for canonical ownership, not a requirement to add every such concept to a shared foundation. Schema Modules make selected intent expressible; Projection Modules supply target expertise and tools. Explicit desired representations and project-owned ProjectionPolicies connect meaning to target representations, while project configuration resolves installed target capabilities separately from that intent. Provider Skills, Workflows, agent definitions, human documentation and generated enforcement are normally representations of this meaning.

**Users change canonical intent. Markitect reconciles representations.** The target flow is:

```text
human-owned engineering intent
    -> canonical Markitect Definitions and purpose
    -> compile, compare intent and derive explicit impact
    -> plan reconciliation of desired representations
    -> Executor Agent using resolved target tools
    -> candidate representation
    -> independent Verifier with bounded evidence
    -> pass, repair or decision escalation
```

The desired authority model extends to code, documentation, agent guidance and automation: they represent accepted canonical intent instead of becoming competing semantic owners. Brownfield work adds a reverse inference path: existing representations can inform a candidate intent, but never grant it authority. Owners review and correct that proposal, accept canonical intent, then match exact artifact scopes and verify existing representations before adopting valid bytes as a baseline. Adoption preserves valid bytes; it does not regenerate or normalize them to an Executor's preference. Unknown or excluded artifacts remain explicit and unresolved artifacts are not rewritten or deleted. A partially adopted repository must identify what remains unmodeled, externally owned, excluded or uncertain; installation alone does not govern the whole repository. Humans decide adoption and real intent changes; unchanged intent with a drifting artifact calls for repair of that artifact.

The product direction is an executable engineering-governance layer for autonomous software development. Markdown views, provider instructions, context, checks and adapters serve that purpose. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is faithful, higher-quality autonomous engineering with architectural authority remaining human-owned. More work per unit of human attention is a consequence to measure alongside quality, reliability and model upkeep.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies, processes and owners; define implementation freedom, acceptance requirements and delegated authority; decide genuine ambiguity, conflicts, exceptions and architecture evolution. |
| Markitect | Make selected engineering decisions explicit and versioned; compile declared structure deterministically; orchestrate explicitly supplied checks and policy mechanisms above Core; compile relevant context and impact; preserve provenance; supply canonical meaning to independent projections, checks and reconciliation adapters. |
| AI agents | Understand the task, consume its relevant context, implement within granted boundaries, test/refactor, update owned projections, run configured verification/reconciliation and continue routine work. Propose changes to intent separately and escalate decisions they are not authorized or equipped to make. |

For governed changes, the intended flow is accepted desired state → relevant agent context and bounded checks → materialization or repair of declared representations → observed evidence. An intent change updates its canonical owner and replans before materialization. When intent is unchanged, a bug or stale representation is repaired against that unchanged model; this is not a bypass and does not require a fake model edit. Architecture changes and proposed policies return to their human owner for deliberate review and adoption; observations do not silently rewrite the rules.

This operating model is a target, not a claim that the current release intercepts every repository change or reconciles every artifact. Current workflow guidance and configured checks cover only their declared scope. Applying the target more broadly requires explicit representation ownership, adapters and bounded verification evidence.

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## What executable governance means

One semantic owner exists per accepted engineering fact or representation decision. Natural-language rationale and guidance can be canonical Definition data. Human narrative not yet adopted, vendor inputs and specialist observations remain explicitly classified outside that scope; migrating them is a reviewed intent change, not automatic inference.

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
| Current product line | Classic is the active Markitect development line on `main`. Government remains a separate experiment; it does not redefine the canonical model, add Core semantics or supply a prerequisite for using Classic. The [roadmap](implementation-plan.md) owns exact integrated-source status; a source merge is not a new published CLI version. |
| Published capability | v0.14.1 preserves the stable Project/Domain CLI contract from v0.13.0, including versioned Domains, typed relations, finite assertions, context/impact, packages, PolicyResults/exceptions and configured projections/adapters. The prior read-only analysis, selective preparation/Copy Me, Markitect-first workflow and managed-artifact accounting remain within that compatibility contract. The roadmap and production assessment own exact release coverage. |
| Experimental alpha (v0.14.1) | The canonical reset and controller are bundled as an explicitly experimental preview, separate from stable Project/Domain CLI support. The [v0.14.1 production assessment](production-assessment.md#v0141-published-release) records the exact source, release gates and assets. Bounded existing-target adoption remains limited to declared scope; Apply produces materialized-unverified records that need fresh Verify. Neither the owner claim nor report establishes semantic review, automatic ownership, complete assurance or real-adopter benefit. |
| Aspirational operation | Many continuously working agents, less routine human supervision, architecture-level escalation and sustainable governance at greater scale. No current scheduler, dashboard or universal automatic acceptance is claimed. |
| Unproven benefit | Reduced human intervention, lower total maintenance/synchronization cost, fewer missed updates or defects, better sustained autonomy, token savings and market demand. |

The [real-code adoption pilot](validation/real-project-adoption-pilot.md), [AGENTS.md comparison](validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

The central evaluation question is: **Does ontology-driven reconciliation preserve intent and project quality more reliably than conventional agentic work, and thereby support less routine human supervision?** [Measurement](measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative.

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.


## Research refinement

The [ontology, projection and assurance thesis](research/README.md) is owner-supplied research orientation. Its distinction between semantic authority, implementation freedom, independent evidence and integration obligations guides experiments. The [alignment assessment](design/projection-assurance-direction.md) identifies what this source slice supports and what remains research. It neither expands Core by naming concepts nor changes measured evidence into a benefit claim.
