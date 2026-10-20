# Product vision

This is the canonical owner of Markitect's product thesis, intended division of responsibility, delegated method and long-term operating model. [Architecture](architecture.md) owns technical boundaries and kernel invariants, the [roadmap](implementation-plan.md) owns released/source/planned status, [Measurement](measurement.md) owns staged readiness and how to test benefits, and the [concept record](concepts/README.md) owns individual product decisions, promises, assumptions and agreed but unscheduled directions, each with its origin. This vision does not authorize a feature, weaken an acceptance gate or establish a measured outcome.

## Markitect in brief

Read this section first. The rest of this page develops it, and the [concept register](concepts/register.md) holds the individual decisions behind it.

**What it is.**
- People maintain one authoritative place: the canonical model of the project's intended world. The model holds the project's concepts, rules, boundaries, responsibilities, architecture and way of working.
- Code, tests, configuration and documentation are realizations of that model.
- A change of intent is made to the model, usually in conversation with a coding agent. Markitect then has it applied:
  - It compiles the model and checks that every repository file is accounted for.
  - It derives which responsibilities, obligations and files the change affects.
  - Every local Manager receives the Apply mandate with explicit model context and permanent ownership. Managers work in parallel on one shared tree and communicate only with parent and children. Fresh recursive Verify assesses one immutable closing stand; parents inspect child integration rather than only collecting reports ([DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace)).
  - Fresh Verify conversations assess each responsibility and its integration without the author conversation or a second permanent Manager hierarchy.
  - The result is verified across the required Manager tree before a guarded Apply.
- A repair that leaves intent unchanged is realized against the same model, without an invented model edit.

**Why.**
- Coding agents lose track of relevant files and rules as projects grow. Explicit consistency runs miss things and scale badly.
- Common practice nevertheless trusts these agents: an agent implements a work item, another AI reviews the pull request, and the required human approval is often given almost blind.
- So the knowledge of what belongs to what must not depend on an agent's memory. And people cannot read every diff without becoming the bottleneck.

**Expected outcomes, to be validated.**
- Nothing in the modeled scope is silently forgotten.
- Rules are applied where they matter.
- The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable.
- Capable mid-tier models can do most of the delegated work.
- People concentrate on deciding and modeling while the code reliably reflects the model.

The intended end state is **model intent → AI realizes and independently reviews it → people can rely on the result without routinely reviewing code diffs themselves**. Automated four-eyes review and integration remain part of the process. Human attention goes to model quality, architecture and escalated decisions. This is the owner's [restated goal of 17 October 2026](concepts/sources/foundations-and-trust-20261017.md#owner-statement), already recorded in [PRM-004](concepts/register.md#promises); it is a target to earn through evidence, not a claim that current Apply has achieved it. The project's acceptance policy still applies.

These are [promises](concepts/register.md#promises), not established results.

**How to judge it.**
- Judge Markitect's trust against today's practice, not against a compiler's guarantees. Comparative studies still use a strong conventional workflow that is allowed to win.
- "Compiler" names a borrowed idea. The model's structure compiles deterministically, and an invalid or incompletely covered model blocks work. Realization by agents is not deterministic.
- Markitect's claim is that compiled obligations, guaranteed Manager participation, bounded responsibility and independent review make delegated work more trustworthy than unstructured agentic work. [Where trust comes from](#where-trust-comes-from) lists these mechanisms.
- Delegated realization is central to the product. It is not an optional layer on top of a validator.
- Markitect's product owner has endorsed separating the method from any particular execution runtime. The accepted design puts Codex App Server and Claude Code behind provider-neutral session contracts, with each step in its own Linux container ([DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). The [delivery plan](implementation-plan.md), [provider guide](provider-adapters.md) and delivery PRs distinguish implemented components from a demonstrated Apply; a runtime probe or adapter test alone does not establish the Manager process. See [DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime).
- [Common misreadings](#common-misreadings) lists the readings to avoid.

## The thesis

> **Define the desired project world once. Reconcile its representations without losing intent.**

The primary goal is quality and consistency during autonomous engineering: fewer forgotten obligations, overlooked consequences, conflicting representations and ignored rules. The project ontology expresses accepted concepts, relationships, purpose and constraints; the governed repository is its representation. A change to that world should drive impact, fanout, reconciliation and independent verification. [The delegated method](#the-delegated-method) defines that cycle.

Human attention is the central bottleneck. Better independent evidence and explicit delegation should reduce routine supervision while preserving human ownership of meaningful decisions. The owner's [priority clarification](concepts/sources/human-attention-and-baseline-20261017.md#attention-and-time) treats calls to capable lower-cost models such as Luna as inexpensive relative to a developer inspecting work, detecting an omission and prompting repairs. Additional Agents are justified when they improve fidelity and reduce that attention; Agent count alone is no defect or success metric. Token and monetary cost remain observations, not the primary optimization target.

The practical question is: **Does what the AI built actually match what I intended?** The organization should provide a dependable answer without requiring routine manual inspection or an extra review request made just for reassurance. Agents can overlook obligations as well as misunderstand them; responsibilities, compiled context, independent perspectives and integration address both. Human management practice inspires the structure. Its effectiveness with Agents is what the product must demonstrate.

Elapsed time is a separate concern. Parallel work can shorten the critical path where responsibilities and dependencies permit it. The simplified strategy selects parallel Execute and a shared pause for closing Verify; its shared-tree behavior and scalability remain to be demonstrated. Measure actual human attention and wall-clock time separately, including review, correction and waiting.

Markitect's working assumption is that AI implementation capacity will grow and become increasingly parallel, while human engineering judgment remains scarce. If humans must implement, synchronize and review every routine change, that attention becomes the bottleneck. Architecture matters more as implementation becomes cheaper: increasing output without explicit boundaries can also increase accidental coupling, inconsistent policy and architecture drift.

These are product assumptions to test, not economic or productivity conclusions established by this repository. Agent capability, task complexity, verification coverage and model-maintenance cost can change the outcome.

Humans should spend more attention defining goals, concepts, architecture, invariants, policies, processes, responsibilities and allowed degrees of freedom. AI agents should perform as much of the remaining implementation, testing, refactoring, documentation, migration, release work and routine maintenance as their granted authority and available evidence permit.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

The owner's [foundations](concepts/origin-and-rationale.md#foundations-restated-on-17-october-2026) connect model-first and specification-driven development, typing, architectural boundaries, divide and conquer, bounded delegation, escalation, independent perspectives and capable lower-cost agents. Their common purpose is a clear, uniform repository whose code, tests, documentation and configuration continue to express the same model. Typing catches structural inconsistencies early; the Orchestrator guarantees required participation and process; Agents interpret meaning, realize their areas and review the result. Each part has a different job.

## Product definition

**Markitect is the canonical model of a project's intended world and the responsibilities for realizing it.** A project organizes concepts, rules, architecture, workflows and decisions into Packages. One Manager per Package realizes that slice, and the Managers form a hierarchy that follows the Packages ([DEC-025](concepts/register.md#dec-025-package-managers-organize-realization)). Markitect keeps a registry, built from the Managers' registrations, that links every artifact to the Definitions it realizes; every other file is a Markitect input or explicitly ignored. The model lives under `.markitect/` in the adopting repository; the generated readable project document is a view of those sources, not a competing owner. Provider, editor and file-format choices are implementation details unless the project deliberately makes them part of its intent.

**Owners change the model. Managers coordinate realization.** The target flow is:

```text
explore goals, constraints and open decisions
    -> accept and commit the project model
    -> derive every local Manager and its bounded context
    -> delegate implementation within declared freedom
    -> integrate child changes and run declared checks
    -> Verify the full required Manager tree
    -> guarded Apply, bounded repair or owner decision
```

The committed model revision is Markitect's accepted repository specification for that revision. Drafts, distillation reports and uncommitted edits remain proposals. Git commits and digests bind bytes and history; they do not authenticate who approved the model or prove that a human reviewed it. Projects define their decision and acceptance process outside that machine identity boundary. Brownfield work can use existing artifacts to inform a proposal, but inference never grants the proposal authority. Owners review and correct the model, commit the selected specification, then map artifact scopes and assess existing realizations. Unmodeled, excluded, externally owned and uncertain paths remain explicit. A partial model must not imply repository-wide governance.

The product direction is an executable engineering-governance workflow for AI-assisted software development. Readable documentation, native agent instructions, bounded context, implementation tools and checks serve the model-first process. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is faithful, higher-quality autonomous engineering with architectural authority remaining human-owned. More work per unit of human attention is a consequence to measure alongside quality, reliability and model upkeep.

## Three starting situations

There are **three starting situations, but only two processes: Apply and Extract**, accepted by the owner on 19 October 2026 ([DEC-057](concepts/register.md#dec-057-three-fundamental-starting-situations), [process naming decision](concepts/sources/three-starting-situations-20261019.md#owner-selects-apply-and-extract)). The model describes intended meaning and structure; artifacts realize it.

| Process | Direction | Goal |
|---|---|---|
| Apply | Model -> Artifacts | Bring artifacts into agreement with the accepted model, whether they must first be created or already exist. |
| Extract | Artifacts -> Model | Develop a reviewed model candidate of the existing project's meaning and structure for owner adoption, through derivation, integration, owner clarification and iterative Verify. |

| Situation | Typical use | Process |
|---|---|---|
| 1. Model exists; artifacts do not | Start a new project by modeling it first. | Apply creates the realization. |
| 2. Model and artifacts exist | Normal ongoing development and maintenance. | Apply realizes the model change in the artifacts. |
| 3. Artifacts exist; model does not | Adopt Markitect for an existing project. | Extract derives and refines the model interactively with the owner. |

Building a new project and applying the model to an existing one are starting states of Apply, not separate processes or commands. These names describe the high-level direction of work; they establish neither executable availability nor a new implementation assignment.

Both Apply starting states use **prepare -> Execute and integration from Root down the fixed tree -> capture a complete candidate -> independent Verify -> repair or publication** ([DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify)). There is no initial Verify round. Execute includes examining the available realization and may preserve every byte. Each parent first provides the concrete foundation its children need, then their realization begins. Results are integrated upward and subsequently independently verified; parents reach final completion after child integration. The [runtime contract](design/canonical-model/manager-apply-runtime.md) describes the implemented sequence.

### 1. Build from the model

This is Apply with no existing artifacts. Managers build the project. Root starts by preparing the root-owned project skeleton and shared prerequisites. Its children build their areas on those foundations, and their children do the same recursively. Each Manager supplies the foundations its children need within its own responsibility; it does not implement their areas for them. Results are integrated upward, with each parent responsible for its own work and the interaction of its children's results.

This is a dependency order for construction, not a new globally sequential scheduling rule. Areas whose prerequisites are available can progress under the agreed Manager process. The model need not prescribe every file or implementation step: it states the intended meaning, structure and deliberate constraints, leaving the agreed realization freedom. An unresolved intent question returns to the owner through Root, as a model defect that ends the Apply.

This makes the existing [start-from-model direction](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) explicit at the product level. The [Library walkthrough](design/canonical-model/manager-apply-library-flow.md#first-apply-m1-to-a1) owns the bounded first-Apply example.

### 2. Model and artifacts exist

This is the same Apply process with existing artifacts, the normal working state. Missing artifacts are created, existing artifacts are assessed and updated where required, and artifacts with no remaining justified model basis are removed under the existing removal rules. Managers realize their responsibilities against the current accepted model, integrate their children and verify the result. A model change supplies new intent; a bug or other realization deviation can require repair against the unchanged model. A repair does not need an invented model edit.

Before children begin realization, each parent assesses its existing foundation within Execute against the current model, actual artifacts and known child requirements. It preserves a matching basis or has the responsible owner correct or supplement it, then identifies the concrete usable result to the children. Later gaps or changes are coordinated through the parent before integration completes. This is the [existing-foundation part of Execute](design/canonical-model/manager-apply-concept.md#existing-foundations-during-execute), with the same subsequent independent Verify; it adds no initial Verify or separate case-2 route.

Parents assess their children's actual results and interactions. If children make contradictory claims during Apply or Verify, the parent investigates and resolves the conflict within the accepted intent and delegated freedom. Unsettled meaning or a decision outside that authority returns to the owner through Root. Read-only Verify reports the existing stand; authorized Apply changes the artifacts. The [command design](design/canonical-model/commands.md#one-organization-and-different-instructions) owns that distinction.

### 3. Derive the model from existing artifacts

This is Extract, the reverse direction Artifacts -> Model. Start from a fixed existing repository and named scope, without requiring a complete model. Root derives a first proposed responsibility/Package structure from the artifacts, and the initial Manager tree is formed from that provisional model outline. Parents provide the source and candidate context their children need to investigate. Managers then propose the meaning and structure of their areas and integrate the model upward; discovery can require a visible correction of the initial boundaries. Parents reconcile shared concepts, relationships and contradictory child statements. The process is interactive with the owner, especially where sources conflict or intended meaning cannot be established. Questions reach the owner through Root in understandable terms; the owner need not inspect YAML to decide them. The [first-candidate workflow](design/canonical-model/commands.md#build-the-first-candidate) owns the details. Adoption is the purpose of this starting situation; Extract names the process.

The cycle is **derive first structure -> form provisional Managers -> develop, review and integrate the model with owner clarification -> read-only Verify against existing artifacts -> refine the candidate and Verify again**. Model review judges the proposal's source support and useful description; Verify judges the unchanged artifacts against the exact candidate. Observed behavior, proposed intent, accepted decisions and unresolved questions remain distinct. The [Extract workflow](design/canonical-model/commands.md#extract-the-existing-stand) owns the detailed source, review and refinement boundaries.

Keep the provisional Manager structure stable within each Extract round. Review and integrate structural corrections between rounds, carrying all findings, source and model references, and open questions to the resulting responsibilities. The [round-transition workflow](design/canonical-model/commands.md#keep-the-structure-stable-within-each-extract-round) preserves their status and traceability; an earlier positive assessment does not automatically approve the revised candidate.

Existing artifacts stay unchanged during model derivation. A confirmed realization defect remains a defect against the justified model; rewriting intent to obtain a green Verify would defeat adoption. Essential unresolved model questions or evidence gaps preventing a sound model or assessment block completion for the affected scope. Known, evidenced artifact defects do not by themselves block a sound, reviewed model's readiness for adoption. Root reports [model completion, the actual Verify result and later repair separately](design/canonical-model/commands.md#completion-and-adoption-with-known-deviations), with remaining scope and questions visible. Subsequent authorized artifact repair is case 2.

At the [transition to Apply](design/canonical-model/commands.md#adopt-the-model-and-hand-findings-to-apply), adopt the model and registry as one coherent reviewed stand. Carry known artifact defects into the first regular authorized Apply, with their model references, source evidence and responsibilities preserved. Reassess them against the actual starting artifacts and treat them as repaired only after independent Verify supports resolution on the integrated candidate. Adoption introduces no exception for known defects and no special first-Apply route.

### One organization, explicit scope

Apply and Extract reuse the recursive Manager organization. Apply covers cases 1 and 2 and changes artifacts against the accepted model. Extract covers case 3 and prepares model changes from existing artifacts. For Apply, the accepted model is the specification; for Extract, artifacts supply evidence for proposed intent, with unresolved meaning clarified by the owner. Similar coordination does not merge those write authorities. The cases can coexist in different areas of one project, so every assessment names its scope. An empty project with neither model nor artifacts enters case 1 once an initial model has been developed and accepted.

The [current workflow](project-workflow.md), [delivery plan](implementation-plan.md) and actual validation records own availability and evidence. This high-level account adds no Extract runtime, new lifecycle protocol, provider run or implementation queue.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies, processes and Managers; define implementation freedom and acceptance requirements; maintain the project's approval process and resolve decisions outside delegated authority. |
| Markitect | Compile the committed project model and fixed repository snapshots; derive bounded context and impact; route Manager work; validate candidate structure and declared evidence; preserve provenance and guard writes. |
| AI agents | Explore and propose intent, implement assigned responsibilities, test and integrate within granted boundaries, report evidence, and raise unresolved decisions. An agent report is not owner approval. |

For governed changes, the intended flow is committed accepted model → relevant Manager context and bounded checks → materialization or repair of declared artifacts → full Verify and observed evidence. A real intent change updates its canonical owner before implementation. When intent is unchanged, a bug or stale artifact is repaired against that same model; no artificial model edit is needed. Exploration, distillation and other proposals remain drafts until the owner chooses to commit the revised model. A commit is the specification basis, not proof of who reviewed or approved it.

This operating model does not claim that the published release supports the new workflow or that current source closes every user journey. The [delivery plan](implementation-plan.md) and delivery PRs own implementation progress. Source tests and smokes of Apply do not establish a real provider run, the two-Apply demonstration, a release or owner acceptance. Checks establish only their declared scope; none proves semantic correctness, human acceptance or success for every adopting repository. Runtime isolation has its own concrete boundary under [DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers).

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in, which Markitect's product owner calls federalism ([CPT-001](concepts/register.md#cpt-001-federalism)). Each mechanism below replaces something that today's common practice leaves to chance. These are the method's mechanisms. [Architecture](architecture.md) and current executable guides describe implemented behavior; the [delivery plan](implementation-plan.md) owns the rebuild toward the new design. Actual checks remain bound to their source and scope.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **Compiled responsibility and guaranteed participation.** The model and registry make declared obligations, responsibilities and files explicit. The Orchestrator includes every local Manager in the first Apply. | An agent deciding from memory which files might be relevant |
| **Bounded responsibility.** Each Manager owns a slice of the model, the files that realize it and the context it needs. Its parent keeps the duty to integrate. | One agent holding the whole project in a single context |
| **Independent review with fresh context.** A Manager, or a helper it starts, implements within the Manager's scope. A fresh Verify turn, without the author dialogue, then assesses the one immutable candidate: the four-eyes principle at the lowest level. Parents also assess their children's actual interaction. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** Applicable rules and policies are compiled into the responsible Manager's context, and its fresh Verify turn checks them. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Verification and guarded Apply.** The exact candidate is verified across the required Manager tree before Apply. Unknown or unchecked areas stay visible. | Approving a pull request because its pipeline is green |

[The delegated method](#the-delegated-method) calls the implementing and assessing roles Executor and independent Verifier.

Two claims must be kept apart ([CPT-003](concepts/register.md#cpt-003-two-parts-of-the-promise)):
- **Nothing in the modeled scope is overlooked.** For declared relationships and file coverage this can be computed. Undeclared dependencies in code are a known gap ([ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact)).
- **Everything is realized correctly.** This can only be supported by evidence.

Tests belong to the realization. Agents write them, so they can be as wrong as the code. Reviewing tests against the statements they claim to check is an agreed direction ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)).

## The delegated method

The method has three responsibilities. They are responsibilities, not three Core primitives or three people. The delegated method has not yet been tested over a series of changes ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)); see [current basis and unproven benefit](#current-basis-and-unproven-benefit).

| Responsibility | What it does |
|---|---|
| Intent ownership | Accepts the concepts, rules, purpose and allowed freedom. Decides real intent changes, unresolved conflicts and exceptions. |
| Execution | Takes the applicable context, implements the affected work as a bounded candidate, and repairs failed representations without redefining success. |
| Independent verification | Checks the candidate against the accepted obligations with evidence suited to the target. Exposes violations, missing coverage and uncertainty, and returns pass, repair or an owner decision. |

In the method, a Manager executes, while fresh Verify turns and required checks assess the result ([where trust comes from](#where-trust-comes-from)). The [delivery plan](implementation-plan.md) distinguishes session components from the complete workflow. An Executor must not be the sole judge of its own success. Verification needs the accepted obligations and evidence about the candidate, including its integration, not only tests written by the Executor. Deterministic tools can add evidence. An agent label or a second model call alone does not make verification independent.

**Valid representations.** Code, documentation, agent instructions and other files represent the accepted intent; they do not hold their own copies of its rules. A representation is correct when it is one of the valid representations of that intent. Different structures or names can be equally valid where the policy allows them, so the allowed freedom is explicit. Byte equality applies only to targets with an exact deterministic contract. Keep representations that are already valid instead of rewriting them to an agent's taste.

**Current strategy:** [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace), amended by [DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify), and the [current concept](design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026) select Execute and integration first, then fresh recursive Verify on one immutable candidate. The [runtime contract](design/canonical-model/manager-apply-runtime.md) describes the implementation, and the [repeated demonstrations](validation/recursive-apply-repetitions-20261019.md) are bounded evidence for one small example.

**The change cycle.** This is the target process. [Project workflow](project-workflow.md) describes what current source implements.

1. **Bind and prepare.** Fix the model, controls, repository, permanent ownership and budget. A real intent change updates the model before implementation; a drift repair needs no invented model edit. Block unexplained files and ambiguous ownership before Apply.
2. **Execute and integrate from Root.** Work proceeds down the fixed parent/child tree. Every owner examines the available realization as part of Execute, preserves what fits, creates what is missing and corrects deviations in its code, tests, docs and registry. Each parent first establishes the concrete foundation needed by its direct children and communicates its availability; their realization starts afterward. An existing suitable foundation may be preserved. Siblings whose foundations are available may work in parallel in one shared tree; parents coordinate further dependencies and integrate results upward before final completion. Only Root reaches the human: in Apply by reporting a model defect, which ends the run ([DEC-063](concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). There is no initial Verify or separate no-artifact bootstrap route, no per-Manager merge and no requirement to edit an already conforming area.
3. **Quiesce and capture.** Finish or stop writers and their subprocesses and capture one complete immutable candidate. If the stop or capture is uncertain, retain incomplete work instead of publishing.
4. **Verify recursively.** Fresh conversations assess that same stand, without the author dialogue, children before parents. Parents inspect their own obligations and actual child interactions, not just green reports. Required checks use the same candidate. Concrete model violations, absent model-required evidence and red mandatory checks block.
5. **Repair or apply.** Return findings through the tree to their owners for Execute and integration, then capture and verify a new stand after corrections. Initially repeat the whole Verify round rather than maintain fine-grained invalidation. Complete only with every responsibility assessed, blocking findings closed and guarded publication/read-back of the exact verified bytes. The finite budget bounds repetition; human acceptance remains separate.

**The same method at every scope.** Each parent owns coordination and integration of its direct children. Every local Manager is represented even when no artifact change is needed; a concise independent assessment replaces a separate no-change delivery transaction. The [current Execute-first cycle](design/canonical-model/manager-apply-concept.md#shared-working-tree-and-reconciliation-cycle) owns the precise behavior. Declared links supply applicable context, not inferred ownership or new communication routes.

**Rules, process and evidence.** A precise rule can have a deterministic check. A broad rule, such as consistent error handling, needs a declared surface, a concrete interpretation and independent evidence; a few examples do not prove it everywhere. Check the agent's process as well as its files: a compliant final file does not prove that the agent started from the model, used its context or handled intent first. A planted defect shows only that a check catches that defect. How often the method works needs repeated real tasks ([measurement](measurement.md)).

**Adoption.** For an existing project, inferred intent is a proposal ([product definition](#product-definition)). Keep valid bytes, record unknown ownership and missing evidence, and never change the model just to bless accidental drift.

## What executable governance means

One canonical owner exists per modeled engineering fact or representation decision. Rationale and guidance may be prose inside the project model. Unadopted narrative, vendor material and specialist observations remain outside accepted intent until explicitly incorporated. The project tree makes responsibility, artifact relationships and decision scope visible; it does not infer source-code semantics or turn every repository file into a model fact.

The new structural Core does not contain an engineering assertion DSL. Checks, policies and assurance live above Core; the historical finite Domain/policy contract belongs to the earlier Project/Domain line, which is not developed further ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)). Project-owned checks and target tools can provide specialized implementation evidence. Rule and Workflow prose remains instruction and rationale where no executable assertion exists; storing it in a typed resource does not enforce the process or prove that an agent followed it.

Context helps agents find applicable intent; impact helps reviewers understand declared consequences; versioned packages make architecture evolution explicit; narrow exceptions record deliberate deviations; reconciliation can align owned representations. None proves that all relevant intent was modeled, that an agent obeyed it, or that arbitrary application code is correct.

Continuous autonomous agents are the desired consumer operating model. They may call bounded Markitect operations from their own runtime, orchestration or CI. An Executor produces a candidate representation using appropriate tools, including deterministic renderers when useful: permitted implementation freedom must be explicit and project-owned checks must state what they establish. This does not require a continuously running Core, hidden policy activation or a background mutation controller. The [delivery plan](implementation-plan.md) defines the current sequence. The current CLI does not expose the earlier Observe/Plan/Apply/Verify workflow. Apply is implemented as recursive Passes over a shared working tree ([DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace), [runtime contract](design/canonical-model/manager-apply-runtime.md)), pending owner acceptance.

## Desired operating model

Humans should eventually review decisions at the level of architecture, ambiguous requirements, unresolved policy conflicts, deliberate exceptions and exceptional risk. Agents should handle ordinary implementation details within the project's declared freedom.

The organization must make a justified no-change result, a useful objection and timely escalation count as good work. Different stakeholders and perspectives need distinct questions and the context to answer them; adding Agent titles or rewarding completion alone supplies no such control. These are motivations for the existing [team direction](concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes), not a requirement to add more roles, separate hierarchies, scores or incentive machinery to the first Apply.

An illustrative future summary might say:

```text
Agents completed:                 47 changes
Automatically verified:          42
Need architectural decision:       3
Need explicit policy exception:    1
Ambiguous requirement:             1
```

These invented numbers describe an aspiration, not a current dashboard or measured run. The aim is to focus human attention on the five decisions. “Verified” would still mean only the declared checks and evidence coverage, not complete software correctness or automatic acceptance. The project must explicitly define its delegation and acceptance policy.

Markitect's product owner describes a longer-term picture beyond this thesis: a cockpit with a personal AI, and an autonomous project organization ([VIS-001](concepts/register.md#vis-001-cockpit-personal-ai-and-autonomous-project-organization)). On the later Government axis, management levels would also maintain the model within mandates ([DEC-004](concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)). The related ideas, such as briefings, precedents and dispute handling, are recorded but not planned until four conditions hold: the current work packages are done, case-study numbers exist, Main is clean, and a usable, stable version exists ([DEC-011](concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits)).

## Origin of the name

The [origin and rationale](concepts/origin-and-rationale.md) records the owner's concrete project experience and the progression from conflicting code, documentation and agent skills to a canonical model, structural compilation, delegated Managers and the later Government idea. It preserves the [owner's retrospective account of 15 October 2026](concepts/sources/owner-origin-20261015.md) and separates experience, mechanisms and hoped-for effects; it is not an implementation or benefit claim.

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, Rules, Skills, Workflows and agent instructions. Its direction is to turn informal engineering knowledge into structured, versioned architecture and policy, then into an executable engineering system. The name records that origin; it does not require Markdown as canonical syntax. Current typed definitions use YAML, while human-readable prose and explicitly selected projections remain useful. The origin explains the name; it does not limit the product to agent instructions.

## Common misreadings

The readings below came up repeatedly, most recently in the [idea review of 10 October 2026](concepts/sources/idea-review-20261010.md). Check an assessment against this table before treating any of them as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the name comes from. The product is model-first development of the whole project with delegated realization. Agent instructions are one kind of realization among many. |
| The deterministic parts are the product, and Manager execution is scope creep. | Delegated realization is the purpose, and deterministic preparation supports it. The accepted rebuild separates the method from its runtime behind session ports; component availability and complete Apply remain distinct ([DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers), [delivery plan](implementation-plan.md)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed concept, and the trust bar is today's practice ([DEC-002](concepts/register.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept)). |
| The Manager hierarchy is the Government experiment. | The Manager hierarchy belongs to the core axis, Model → Repository. Government belongs to a later axis, in which management levels decide model changes within mandates. Results of the former Government experiment arm are not results about the Manager hierarchy ([DEC-004](concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)). |
| "More economical models" means weak models. | It means capable mid-tier models such as Luna or Sonnet, and Haiku only if it proves sufficient ([DEC-007](concepts/register.md#dec-007-more-economical-models-means-capable-mid-tier-models)). |
| Earlier comparisons showed that conventional agentic coding wins. | Environment, runtime and harness failures dominated the October 2026 comparison attempts, and none of them exercised the delegated method over a series of changes. They allow no conclusion about the concept. Earlier matched comparisons tested an earlier product; their findings, including the cost of model upkeep, stand ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). |
| The model has to mirror the code, with one definition per file. | Granularity is the author's choice. A file may realize several definitions, and every file must be covered or explicitly ignored ([DEC-005](concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility), [DEC-006](concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). |
| Green agent-written tests prove the realization. | Tests are realizations too. Agents write them, so they can be wrong. Reviewing them against the statements they claim to verify is an agreed direction ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)). |
| The goal of removing routine human code review removes the four-eyes principle. | Independent Agent review, integration review and Orchestrator checks are the means to earn that trust. People retain model decisions, escalation and their project's acceptance policy ([PRM-004](concepts/register.md#promises)). |

## Feature decision filter

Ask: **Does this improve fidelity to accepted intent and reliable autonomous reconciliation inside explicit boundaries, with enough independent evidence to reduce routine human supervision?**

A proposal should identify the human activity it aims to remove, its canonical owner, the agent's remaining freedom, the exact evidence/check boundary and the new authoring or maintenance cost. Deterministic checks, task context, impact, policy evolution, exceptions, discovery and reconciliation can support this goal, but each needs a concrete workflow and evaluation.

Reconsider proposals that introduce another manually synchronized truth owner, hide architectural decisions in inference, specialize Core for one provider, or add machinery without evidence that coordination gets easier. More generated text, adapters, language operators or agent throughput alone do not demonstrate progress. This filter does not override the constitution's bar for new semantics or authorize a new feature wave.

For the first Apply, use the existing [complete-workflow gates](design/canonical-model/manager-apply-implementation-plan.md) to ask whether the organization preserves obligations, finds and corrects a bad delivery through its owner, composes the areas correctly and exposes what still needs a human decision. A successful run is bounded feasibility evidence. Reduced routine human review and dependable use of cheaper models need repeated quality and attention evidence under [Measurement](measurement.md#human-attention-and-delegated-work). Keep the first strategy small enough to learn this before adding roles, parallelism or reuse mechanisms.

## Current basis and unproven benefit

| Category | What is established or intended |
|---|---|
| Product direction | A single model-first user workflow centered on the recursively managed `.markitect/` project tree. The [roadmap](implementation-plan.md) records exact source and release status. |
| Current source | The root-layout CLI implements canonical v1 compilation, complete registry/file preparation, execution declarations and static context, plus explicitly requested source-bound profile metadata/settings and an independently guarded readable model view. The recursive Apply of DEC-056 (with DEC-058 to DEC-060) and a first Extract are implemented, pending owner acceptance ([runtime contract](design/canonical-model/manager-apply-runtime.md)); real library demonstrations are [recorded](validation/recursive-apply-repetitions-20261019.md). The preceding source generation remains in [Git](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). |
| Published compatibility | The earlier Project/Domain CLI and Projection alpha remain in unchanged Git tags. Their release pages and binary downloads were retired by the owner on 17 October 2026. The contracts and recorded evidence are historical; there is no second active product UX. See the [production assessment](history/distribution/production-assessment.md#v0141-published-release). |
| Acceptance boundary | A committed model revision is the accepted specification used for that revision. Commit/provenance values bind data but do not authenticate a human decision. Passing checks, generated docs and agent reports are evidence with declared scope, not approval. |
| Aspirational operation | More parallel Managers, less routine supervision, architecture-level escalation and sustainable governance at greater scale. No universal autonomous acceptance or OS isolation is claimed. |
| Unproven benefit | Reduced human intervention, lower total maintenance cost, fewer missed obligations or defects, better sustained autonomy, token savings and market demand remain unmeasured. Historical comparisons are preserved, including their negative or inconclusive findings. |

The [real-code adoption pilot](validation/real-project-adoption-pilot.md), [AGENTS.md comparison](validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

These records evaluated earlier product forms, not the delegated Manager method over a series of changes. The October 2026 comparison attempts of the delegated method were dominated by environment, runtime and harness failures. They allow no conclusion about the concept, positive or negative ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). Markitect's product owner has endorsed evaluating the method separately from any single execution runtime; the plan is pending ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)).

The central evaluation question is: **Does ontology-driven reconciliation preserve intent and project quality more reliably than conventional agentic work, and thereby support less routine human supervision?** [Measurement](measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative. Markitect's product owner phrases the same question in [DEC-009](concepts/register.md#dec-009-the-central-evaluation-question): when people develop against a model, can the code reliably reflect that model? Can the repository stay contradiction-free, consistent, uniform and clean? Does agentic coding become meaningfully more trustworthy? Can people concentrate on deciding and modeling?

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.


## Research refinement

The [ontology, projection and assurance thesis](research/README.md) is owner-supplied research orientation. Its distinction between semantic authority, implementation freedom, independent evidence and integration obligations guides experiments. The [alignment assessment](design/projection-assurance-direction.md) identifies what this source slice supports and what remains research. It neither expands Core by naming concepts nor changes measured evidence into a benefit claim.
