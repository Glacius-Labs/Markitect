# Product vision

This is the canonical owner of Markitect's product thesis, intended division of responsibility and long-term operating model. [Architecture](architecture.md) owns technical boundaries, the [engineering constitution](engineering-constitution.md) owns kernel invariants, the [roadmap](implementation-plan.md) owns released/source/planned status, and [Measurement](measurement.md) owns how to test benefits. This vision does not authorize a feature, weaken an acceptance gate or establish a measured outcome.

## The thesis

> **Implementation is becoming cheap. Human attention is not.**

Markitect's working assumption is that AI implementation capacity will grow and become increasingly parallel, while human engineering judgment remains scarce. If humans must implement, synchronize and review every routine change, that attention becomes the bottleneck. Architecture matters more as implementation becomes cheaper: increasing output without explicit boundaries can also increase accidental coupling, inconsistent policy and architecture drift.

These are product assumptions to test, not economic or productivity conclusions established by this repository. Agent capability, task complexity, verification coverage and model-maintenance cost can change the outcome.

Humans should spend more attention defining goals, concepts, architecture, invariants, policies, processes, responsibilities and allowed degrees of freedom. AI agents should perform as much of the remaining implementation, testing, refactoring, documentation, migration, release work and routine maintenance as their granted authority and available evidence permit.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

## Product definition

**Markitect turns human engineering intent into an explicit, versioned, machine-consumable engineering system so AI agents can build and evolve software within deliberately designed boundaries.**

The product direction is an executable engineering-governance layer for autonomous software development. Markdown views, provider instructions, context, checks and adapters serve that purpose. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is more autonomous engineering work per unit of human attention, with architectural authority remaining human-owned.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies, processes and owners; define implementation freedom, acceptance requirements and delegated authority; decide genuine ambiguity, conflicts, exceptions and architecture evolution. |
| Markitect | Make selected engineering decisions explicit and versioned; validate declared structure and finite policy; compile relevant context and impact; preserve provenance; supply canonical meaning to independent projections, checks and reconciliation adapters. |
| AI agents | Understand the task, consume its relevant context, implement within granted boundaries, test/refactor, update owned projections, run configured verification/reconciliation and continue routine work. Propose changes to intent separately and escalate decisions they are not authorized or equipped to make. |

The intended flow is human decisions → explicit engineering model → relevant agent context and bounded checks → agent work and observed evidence. Architecture changes and proposed policies return to their human owner for deliberate review and adoption; observations do not silently rewrite the rules.

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## What executable governance means

One owner exists per modeled fact or explicit mapping, not for all knowledge in a repository. Human narrative, business meaning and specialist implementation evidence can retain their own owners outside Markitect.

Structured policies can become finite deterministic assertions. Project-owned checks and adapters can provide specialized implementation evidence. Rule and Workflow prose remains instruction and rationale where no executable assertion exists; storing it in a typed resource does not enforce the process or prove that an agent followed it.

Context helps agents find applicable intent; impact helps reviewers understand declared consequences; versioned packages make architecture evolution explicit; narrow exceptions record deliberate deviations; reconciliation can align owned representations. None proves that all relevant intent was modeled, that an agent obeyed it, or that arbitrary application code is correct.

Continuous autonomous agents are the desired consumer operating model. They may call bounded Markitect operations from their own runtime, orchestration or CI. This does not require a continuously running Core, a scheduler, hidden policy activation or a background mutation controller. Current Observe/Plan/Apply/Verify trust boundaries remain in force; Apply still requires explicit authority and exact input-bound plans.

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

Ask: **Does this reduce human execution or supervision while increasing useful agent autonomy inside explicit boundaries, without transferring architectural authority away from humans?**

A proposal should identify the human activity it aims to remove, its canonical owner, the agent's remaining freedom, the exact evidence/check boundary and the new authoring or maintenance cost. Deterministic checks, task context, impact, policy evolution, exceptions, discovery and reconciliation can support this goal, but each needs a concrete workflow and evaluation.

Reconsider proposals that introduce another manually synchronized truth owner, hide architectural decisions in inference, specialize Core for one provider, or add machinery without evidence that coordination gets easier. More generated text, adapters, language operators or agent throughput alone do not demonstrate progress. This filter does not override the constitution's bar for new semantics or authorize a new feature wave.

## Current basis and unproven benefit

| Category | What is established or intended |
|---|---|
| Published capability | v0.12.0 provides generic versioned Domains, typed relations, finite assertions including bounded `same-target`, a normalized model, context/impact, exact-pinned packages, PolicyResults/exceptions, provider projections, configured checks/adapters, explicit reconciliation and the Copy Me proposal/review workflow. The roadmap owns precise coverage. |
| Later integrated source | Explicit read-only analysis of ordinary failed policies and the first parallel wave's bounded adapter/test slices. The GitHub/Azure consumers use supplied offline captures; expanded selective Init/Copy Me preparation remains deferred design. These are not additions to an installed v0.12.0 release. |
| Aspirational operation | Many continuously working agents, less routine human supervision, architecture-level escalation and sustainable governance at greater scale. No current scheduler, dashboard or universal automatic acceptance is claimed. |
| Unproven benefit | Reduced human intervention, lower total maintenance/synchronization cost, fewer missed updates or defects, better sustained autonomy, token savings and market demand. |

The [real-code adoption pilot](validation/real-project-adoption-pilot.md), [AGENTS.md comparison](validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

The central evaluation question is: **Can a small number of humans govern much more autonomous engineering work without losing architectural intent?** [Measurement](measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative.

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.
