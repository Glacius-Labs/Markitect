# Product vision

This is the canonical owner of Markitect's product thesis, intended division of responsibility, delegated method, origin and long-term operating model. The [documentation map](README.md#canonical-status-and-evidence-owners) names the owners of the other subjects, such as technical boundaries, delivery status, evaluation and individual decisions. This vision does not authorize a feature, weaken an acceptance gate or establish a measured outcome.

## Markitect in brief

Read this section first, then [Founding intent](#founding-intent). The rest of this page develops them, and the [concept register](concepts/register.md) holds the individual decisions behind them.

**What it is.**
- People maintain one authoritative place: the canonical model of the project's intended world. The model holds the project's concepts, rules, boundaries, responsibilities, architecture and way of working.
- Code, tests, configuration and documentation are realizations of that model.
- A change of intent is made to the model, usually in conversation with a coding agent. Markitect then has it applied:
  - It compiles the model and checks that every repository file is accounted for.
  - Impact derives which Definitions, files and Packages the change reaches; only the Managers of those Packages work, each with a checklist of the items the change reaches ([DEC-082](concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
  - Each working Manager has a maker and a fresh checker. The maker realizes its part in its own container and Git branch and gives every checklist item a verdict with evidence. The checker reviews the submission against the model and re-checks every verdict: it confirms them or sends the defects back. Parents integrate their children's approved work.
  - Server-side hooks on the Run's remote enforce the technical boundaries: a Manager's own branch and paths, the required checks and a complete checklist.
  - When Root's checker approves, the host lands the result on the target branch. If that branch moved during the Run, the result lands only when the merge is clean, the required checks pass and nothing in the Run's impact changed; otherwise nothing lands and a follow-up Run starts from the new state. Either way the host writes a Result page with every verdict ([concept 2.0, §6](concepts/v2-concept.md#6-runs)).
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
- Markitect's claim is that compiled obligations, participation derived from impact with a verdict for every reached item, bounded responsibility and independent review make delegated work more trustworthy than unstructured agentic work. [Where trust comes from](#where-trust-comes-from) lists these mechanisms.
- Delegated realization is central to the product. It is not an optional layer on top of a validator.
- Markitect's product owner has endorsed separating the method from any particular execution runtime. Providers sit behind a provider-neutral boundary and use account login only; the host runs natively with Docker, and every Manager works in its own container ([DEC-083](concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)). The current target is [concept 2.0](concepts/v2-concept.md) ([DEC-086](concepts/register.md#dec-086-concept-20-is-the-target)); the [delivery plan](implementation-plan.md#concept-20-build) owns what is built. A smoke or a source test alone does not establish acceptance. See [DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime).
- [Common misreadings](#common-misreadings) lists the readings to avoid.

## Founding intent

The owner's [founding intent](concepts/sources/founding-intent-20261026.md) explains why Markitect exists and what the canonical model is meant to be. Earlier accounts are the [origin account](concepts/sources/owner-origin-20261015.md) and the [conceptual influences](concepts/sources/conceptual-influences-20261017.md); [Origin and rationale](#origin-and-rationale) organizes them.

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. Every consistency run took hours and found more inconsistencies. The owner lost track of which rules existed, what was defined and what had entered the project silently.

**The answer is a typed language in prose.** In a strongly typed language, a compiler and explicit dependencies report issues at compile time instead of leaving them to be found later. The canonical model applies that to project intent:

| Typed language | Canonical model |
| --- | --- |
| type, class, function signature | Kind |
| field, parameter | Property |
| declaration | Definition |
| function body | body: binding prose that agents realize |
| namespace, module | Package |
| standard library, packages | Libraries, imported as model dependencies |
| static type checker | compiler: structure only, never meaning |

The owner gave this analogy while the model was still V1's typed YAML. Concept 2.0 keeps it with two changes: there are no typed Properties, so exact values such as error codes are written in prose, and there are no imports, since links name their target by path and Libraries ship in the binary ([concept §2](concepts/v2-concept.md#2-the-model)). A Kind is itself a Definition of Kind `Kind`, a Definition is one Markdown file with front matter, and a Package is a folder.

- The core stays clean. Projects define their own Kinds and so form their own language; Libraries supply common Kinds, oriented on what Go or C# ship, so that no project reinvents them.
- The compiler checks structural correctness from the core alone. Meaning is judged by the agents and the Manager hierarchy.
- The Apply organization follows Conway's law, Clean Architecture, clean code and modularity: Packages are modules, and each has a responsible Manager. Kubernetes and Terraform are further orientation points.
- People decide intent. AI agents write the model in conversation through the modeling skill, which asks first when something is open; the owner reads the Markdown diff and accepts it. Extract, drawing a model from existing code, is its own skill on top of it. Both ship inside the `markitect` binary, and `markitect init` installs them ([concept §8](concepts/v2-concept.md#8-authoring)).

**Consequences.**
- Judge a model as you would judge code: dependencies, mutation testing, CRAP, clean architecture and size. A Definition of 500 words is prose compensating for missing Definitions.
- How explicit a model is, is settled: as explicit as good documentation of the project ([DEC-079](concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). The standard Libraries and the modeling skills exist in their 2.0 form, and `lint` measures a model like code ([model language](guides/model-language.md)). Whether they lead to good models in practice is still to be shown. Evaluations of Markitect are only as good as the model they use.

## Primary goal and prerequisites

The owner's [21 October clarification](concepts/sources/primary-goal-and-conformance-evaluation-20261021.md#owner-statement-primary-goal) makes the priority explicit: earn trust in Agents through structured, systematic realization and independent checking of the canonical model across **all project artifacts**. Nothing material should be overlooked or contradicted. Responsibility is divided by area; independent local review and parent integration/review must establish how the parts fit. Sustained consistency, uniformity and cleanliness are the resulting long-term project outcome. This clarifies the existing thesis; it does not establish achieved reliability or commission optional cleanup of conforming artifacts.

Three prerequisites belong to that goal: the model can adequately express the intended software project; its structure and meaning can be assessed for consistency, completeness relative to explicit purpose, and uniformity; and people can maintain it comfortably, mainly through Agents and readable views. A self-consistent model cannot reveal an entirely omitted user need without that purpose or requirement being supplied. Readable progress and results should expose assessed scope, evidence and unresolved matters, rather than imply universal correctness through a green status alone.

In concept 2.0 the checklist carries the semantic obligation: every item a change reaches needs a verdict with evidence ([§6.2](concepts/v2-concept.md#62-one-run-maker-and-checker)); [measurement](guides/measurement.md#evidence-for-the-primary-goal) defines discriminating evidence. Neither a complete checklist nor successful tests alone establish that every material condition is realized.

## Proportionate Agent process

The owner's [22 October clarification](concepts/sources/proportionate-agent-process-20261022.md) reaffirms DEC-002: the practical baseline is giving a coding Agent a task and trusting its work. Markitect should improve that trust through explicit model intent, bounded ownership, independent perspectives and parent integration. It is not a formal proof system. Agents judge meaning and parents review responsibility and integration; the Host supplies execution, communication, retained results and finite limits. The clarification asked for no technical write protection at every boundary. Concept 2.0 enforces a few boundaries technically: server-side hooks hold each Manager to its own branch and paths, the required checks and a complete checklist ([§6.3](concepts/v2-concept.md#63-git-hooks-and-messages)).

Use the smallest process that supports those responsibilities. More report fields, mandatory probes or completeness declarations do not themselves improve correctness. Tests and inspected artifacts support professional judgment; uncertainty about a concrete required property remains visible, but an imagined possibility of another defect must not create endless review. Parents assess composition and challenge relevant child evidence without routinely repeating all child work. Judge the method by actual consistency, detected and repaired defects, false findings, completed workflows and human intervention against the ordinary Agent baseline.

Handoffs are Git commits through the Run's remote ([DEC-078](concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review), [§6.3](concepts/v2-concept.md#63-git-hooks-and-messages)). This goes back to the owner's [adoption of Git handoffs on 22 October](concepts/sources/agent-workspaces-and-handoffs-20261022.md), which replaced V1's earlier Pass/Turn and shared-tree design; V1's [handoff workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-workflow.md) is history. Documentation, implementation and demonstrated benefit remain separate.

## The thesis

> **Define the desired project world once. Reconcile its representations without losing intent.**

The primary goal is quality and consistency during autonomous engineering: fewer forgotten obligations, overlooked consequences, conflicting representations and ignored rules. The project ontology expresses accepted concepts, relationships, purpose and constraints; the governed repository is its representation. A change to that world should drive impact, fanout, reconciliation and independent verification. [The delegated method](#the-delegated-method) defines that cycle.

Human attention is the central bottleneck. Better independent evidence and explicit delegation should reduce routine supervision while preserving human ownership of meaningful decisions. The owner's [priority clarification](concepts/sources/human-attention-and-baseline-20261017.md#attention-and-time) treats calls to capable lower-cost models such as Luna as inexpensive relative to a developer inspecting work, detecting an omission and prompting repairs. Additional Agents are justified when they improve fidelity and reduce that attention; Agent count alone is no defect or success metric. Token and monetary cost remain observations, not the primary optimization target.

The practical question is: **Does what the AI built actually match what I intended?** The organization should provide a dependable answer without requiring routine manual inspection or an extra review request made just for reassurance. Agents can overlook obligations as well as misunderstand them; responsibilities, compiled context, independent perspectives and integration address both. Human management practice inspires the structure. Its effectiveness with Agents is what the product must demonstrate.

Elapsed time is a separate concern. Parallel work can shorten the critical path where responsibilities and dependencies permit it. Agents may work independently on their own branches and parents integrate their Git handoffs; the effectiveness and scalability of this target remain to be demonstrated. Measure actual human attention and wall-clock time separately, including review, correction and waiting.

Markitect's working assumption is that AI implementation capacity will grow and become increasingly parallel, while human engineering judgment remains scarce. If humans must implement, synchronize and review every routine change, that attention becomes the bottleneck. Architecture matters more as implementation becomes cheaper: increasing output without explicit boundaries can also increase accidental coupling, inconsistent policy and architecture drift.

These are product assumptions to test, not economic or productivity conclusions established by this repository. Agent capability, task complexity, verification coverage and model-maintenance cost can change the outcome.

Humans should spend more attention defining goals, concepts, architecture, invariants, policies, processes, responsibilities and allowed degrees of freedom. AI agents should perform as much of the remaining implementation, testing, refactoring, documentation, migration, release work and routine maintenance as their granted authority and available evidence permit.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

The owner's [foundations](#foundations-restated-on-17-october-2026) connect model-first and specification-driven development, typing, architectural boundaries, divide and conquer, bounded delegation, escalation, independent perspectives and capable lower-cost agents. Their common purpose is a clear, uniform repository whose code, tests, documentation and configuration continue to express the same model. Typing catches structural inconsistencies early; impact and the hooks make the reached Managers take part and give every item a verdict; Agents interpret meaning, realize their areas and review the result. Each part has a different job.

## Product definition

**Markitect is the canonical model of a project's intended world and the responsibilities for realizing it.** A project organizes concepts, rules, architecture, workflows and decisions into Packages. One Manager per Package realizes that slice, and the Managers form a hierarchy that follows the Packages ([DEC-025](concepts/register.md#dec-025-package-managers-organize-realization)). Markitect keeps a registry, one Markdown page per Package maintained by its Manager, that links every realizing file to the Definitions it realizes; every other file is a Markitect input or explicitly ignored ([§4](concepts/v2-concept.md#4-the-realization-record)). The model lives under `.markitect/` in the adopting repository, as Markdown that people and agents read directly. Provider, editor and file-format choices are implementation details unless the project deliberately makes them part of its intent.

**Owners change the model. Managers coordinate realization.** The target flow is:

```text
explore goals, constraints and open decisions with the modeling skill
    -> the owner accepts and commits the model change
    -> check and impact: who works, and each working Manager's checklist
    -> makers realize within declared freedom; hooks enforce paths and required checks
    -> fresh checkers re-check every verdict; parents integrate
    -> Root's checker approves and the host lands, or a model problem ends the Run
    -> a Result page either way
```

The committed model revision is Markitect's accepted repository specification for that revision. Drafts, distillation reports and uncommitted edits remain proposals. Git commits and digests bind bytes and history; they do not authenticate who approved the model or prove that a human reviewed it. Projects define their decision and acceptance process outside that machine identity boundary. Brownfield work can use existing artifacts to inform a proposal, but inference never grants the proposal authority. Owners review and correct the model, commit the selected specification, then map artifact scopes and assess existing realizations. Unmodeled, excluded, externally owned and uncertain paths remain explicit. A partial model must not imply repository-wide governance.

The product direction is an executable engineering-governance workflow for AI-assisted software development. Readable documentation, native agent instructions, bounded context, implementation tools and checks serve the model-first process. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is faithful, higher-quality autonomous engineering with architectural authority remaining human-owned. More work per unit of human attention is a consequence to measure alongside quality, reliability and model upkeep.

## Three starting situations

There are **three starting situations, but only two processes: Apply and Extract**, accepted by the owner on 19 October 2026 ([DEC-057](concepts/register.md#dec-057-three-fundamental-starting-situations), [process naming decision](concepts/sources/three-starting-situations-20261019.md#owner-selects-apply-and-extract)). The model describes intended meaning and structure; artifacts realize it.

| Process | Direction | Goal |
|---|---|---|
| Apply | Model -> Artifacts | Bring artifacts into agreement with the accepted model, whether they must first be created or already exist. |
| Extract | Artifacts -> Model | Develop a model of the existing project's meaning and structure for the owner to accept, through the Extract skill in conversation with the owner. |

| Situation | Typical use | Process |
|---|---|---|
| 1. Model exists; artifacts do not | Start a new project by modeling it first. | Apply creates the realization. |
| 2. Model and artifacts exist | Normal ongoing development and maintenance. | Apply realizes the model change in the artifacts. |
| 3. Artifacts exist; model does not | Adopt Markitect for an existing project. | Extract derives the model in conversation with the owner. |

Building a new project and applying the model to an existing one are starting states of Apply, not separate processes or commands. These names describe the historical high-level direction of work; they establish neither executable availability nor a new implementation assignment. In concept 2.0, Extract is a skill that uses the modeling skill, not a command ([§8](concepts/v2-concept.md#8-authoring)).

Concept 2.0 realizes a change in one Run: impact decides which Managers work, each with a maker and a fresh checker. Once Root's checker approves, the host lands the result; if the target branch moved, only when the merge is clean, the required checks pass and nothing in the Run's impact changed, and otherwise nothing lands ([§6](concepts/v2-concept.md#6-runs)). A Result is written either way. `apply --full` covers the whole tree and also repairs drift ([§7](concepts/v2-concept.md#7-commands-that-start-agents)); the regular full check is `verify --full`, at milestones and after ten landed changes ([§6.1](concepts/v2-concept.md#61-who-works)). The [architecture](architecture.md) and [delivery plan](implementation-plan.md) own what is built. V1's separate Assessment, Implement and Verify Runs are recorded in the [V1 concept](concepts/v1-concept.md).

### 1. Build from the model

This is Apply with no existing artifacts. Managers build the project. Root starts by preparing the root-owned project skeleton and shared prerequisites. Its children build their areas on those foundations, and their children do the same recursively. Each Manager supplies the foundations its children need within its own responsibility; it does not implement their areas for them. Results are integrated upward, with each parent responsible for its own work and the interaction of its children's results.

This is a dependency order for construction, not a new globally sequential scheduling rule. Areas whose prerequisites are available can progress under the agreed Manager process. The model need not prescribe every file or implementation step: it states the intended meaning, structure and deliberate constraints, leaving the agreed realization freedom. A real model defect ends the Run at Root without landing, and the Result names the decision the owner must make ([§6.2](concepts/v2-concept.md#62-one-run-maker-and-checker)).

In concept 2.0 this order follows from §6.2: a parent with items of its own submits its files first, and its children start from that approved commit. V1's [start-from-model direction](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) and its [Library walkthrough](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-library-flow.md#first-apply-m1-to-a1) are history.

### 2. Model and artifacts exist

This is the same Apply process with existing artifacts, the normal working state. Missing artifacts are created, existing artifacts are assessed and updated where required, and artifacts with no remaining justified model basis are removed under the existing removal rules. Managers realize their responsibilities against the current accepted model, integrate their children and verify the result. A model change supplies new intent; a bug or other realization deviation can require repair against the unchanged model. A repair does not need an invented model edit.

A parent with items of its own in the change submits its own files first, and its children start from that approved commit; conforming files are kept ([§6.2](concepts/v2-concept.md#62-one-run-maker-and-checker)). Gaps a child finds later go to its parent as questions or escalations; messages never carry work ([§6.3](concepts/v2-concept.md#63-git-hooks-and-messages)). There is no separate case-2 route. V1's [existing-foundation step within Execute](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-concept.md#existing-foundations-during-execute) and its separate later Verify are history.

Parents assess their children's actual results and interactions. If children make contradictory claims, the parent investigates and settles the conflict if the model allows. A real model defect ends the Run at Root, and the Result names the decision the owner must make. `verify` only checks and changes nothing; `apply` changes the artifacts by landing ([§7](concepts/v2-concept.md#7-commands-that-start-agents)).

### 3. Derive the model from existing artifacts

This is Extract, the reverse direction Artifacts -> Model. In concept 2.0 it is a skill, not a command: it draws a model from existing code in conversation with the owner and uses the modeling skill, which asks first when something is open ([§8](concepts/v2-concept.md#8-authoring)). For a model drawn from existing code, the binding part is the contract callers rely on; details a caller can observe but does not rely on stay free ([DEC-061](concepts/register.md#dec-061-what-an-extracted-model-contains)). Observed behavior, proposed intent and open questions stay apart. The owner accepts the result like any other model change.

Existing artifacts stay unchanged while the model is drawn. A confirmed realization defect remains a defect against the accepted model; rewriting intent to obtain a green `verify` would defeat adoption. After adoption, `verify` reports where the artifacts deviate, and its Findings go into a regular `apply --findings` ([§7](concepts/v2-concept.md#7-commands-that-start-agents)); there is no special first-Apply route.

**History:** V1 designed Extract as a process of the Manager tree, with provisional Managers, rounds and iterative Verify ([DEC-057](concepts/register.md#dec-057-three-fundamental-starting-situations), [Extract workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md#extract-the-existing-stand)). Its command was retired before concept 2.0.

### One organization, explicit scope

Apply covers cases 1 and 2 and changes artifacts against the accepted model; the Manager organization realizes it. Extract covers case 3 and prepares a model change from existing artifacts, for the owner to accept. For Apply, the accepted model is the specification; for Extract, artifacts supply evidence for proposed intent, with unresolved meaning clarified by the owner. The cases can coexist in different areas of one project, so every assessment names its scope. An empty project with neither model nor artifacts enters case 1 once an initial model has been developed and accepted.

The [current workflow](guides/project-workflow.md), [delivery plan](implementation-plan.md) and actual validation records own availability and evidence. This high-level account adds no mechanism of its own.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies, processes and Managers; define implementation freedom and acceptance requirements; maintain the project's approval process and resolve decisions outside delegated authority. |
| Markitect | Compile the model, derive impact and checklists, give each Manager its container and branch, enforce the hooks and required checks, land approved work and write the Result, within finite limits. |
| AI agents | Explore and propose intent, implement assigned responsibilities, test and integrate within granted boundaries, report evidence, and raise unresolved decisions. An agent report is not owner approval. |

For governed changes, the flow is committed accepted model → impact and checklists → makers realize, fresh checkers judge every item, hooks enforce the required checks → landing with a Result. A real intent change updates its canonical owner before implementation. When intent is unchanged, a bug or stale artifact is repaired against that same model; no artificial model edit is needed. Exploration, distillation and other proposals remain drafts until the owner chooses to commit the revised model. A commit is the specification basis, not proof of who reviewed or approved it.

This operating model does not claim that the published release supports the new workflow or that current source closes every user journey. The [delivery plan](implementation-plan.md) and delivery PRs own implementation progress. Source tests and smokes of `apply` do not establish the counted series of [§15](concepts/v2-concept.md#15-acceptance), a release or owner acceptance. Checks establish only their declared scope; none proves semantic correctness, human acceptance or success for every adopting repository. Runtime isolation has its own concrete boundary under [DEC-083](concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers).

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in, which Markitect's product owner calls federalism ([CPT-001](concepts/register.md#cpt-001-federalism)). Each mechanism below replaces something that today's common practice leaves to chance. These are the method's mechanisms. [Architecture](architecture.md) and current executable guides describe implemented behavior; the [delivery plan](implementation-plan.md) owns the rebuild toward the new design. Actual checks remain bound to their source and scope.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **Compiled responsibility and guaranteed participation.** The model and registry make declared obligations, responsibilities and files explicit. Impact decides which Managers work, and every item the change reaches is on a checklist that needs a verdict. | An agent deciding from memory which files might be relevant |
| **Bounded responsibility.** Each Manager owns a slice of the model, the files that realize it and the context it needs. Its parent keeps the duty to integrate. | One agent holding the whole project in a single context |
| **Independent review with fresh context.** A Manager's maker implements within the Manager's scope. A fresh checker, which sees the model, the submission and its diff but not the maker's notes, reviews it, and the parent integrates its children's approved work. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** A Rule reaches every Manager below the Package that defines it. A verdict means the item conforms to it and to the Rules above it, and a changed Rule becomes a checklist item for every Manager below it. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Hooks and guarded landing.** Server-side hooks refuse a submission outside the Manager's paths, one that fails to compile or fails the required checks, and an incomplete checklist. The host lands Root's approved result, after a moved target branch only on a clean merge with passing checks and untouched impact, and records every verdict in the Result. Unknown or unchecked areas stay visible. | Approving a pull request because its pipeline is green |

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

In concept 2.0, each Manager's maker executes, while its fresh checker, its parent's integration and the required checks assess the result ([where trust comes from](#where-trust-comes-from)). An Executor must not be the sole judge of its own success. Verification needs the accepted obligations and evidence about the candidate, including its integration, not only tests written by the Executor. Deterministic tools can add evidence. An agent label or a second model call alone does not make verification independent.

**Valid representations.** Code, documentation, agent instructions and other files represent the accepted intent; they do not hold their own copies of its rules. A representation is correct when it is one of the valid representations of that intent. Different structures or names can be equally valid where the policy allows them, so the allowed freedom is explicit. Byte equality applies only to targets with an exact deterministic contract. Keep representations that are already valid instead of rewriting them to an agent's taste.

**Current target:** [concept 2.0](concepts/v2-concept.md) ([DEC-086](concepts/register.md#dec-086-concept-20-is-the-target)). V1's design, its Run host and its demonstrations stay in the [V1 concept](concepts/v1-concept.md) and in Git history.

**The change cycle** ([concept 2.0, §6](concepts/v2-concept.md#6-runs)):

1. The owner commits a model change. Markitect compiles it and computes impact: the Definitions, files and Packages it reaches, and each working Manager's checklist.
2. A parent with items of its own in the change submits its own files first; its children start from that approved commit.
3. Each maker works in its own container and branch, gives every checklist item a verdict with evidence and pushes a submission. The hooks refuse it unless it stays in the Manager's paths, compiles, passes the required checks and has a verdict for every item.
4. A fresh checker reviews each submission against the model and re-checks every verdict. It confirms them with a commit of its own or sends the defects straight back to the maker. Parents integrate their children's approved work.
5. When Root's checker approves, the host compiles the result, runs the required checks and lands it on the target branch. If the branch moved, the result lands only when the merge is clean, the required checks pass and nothing in the Run's impact changed; otherwise nothing lands and a follow-up Run starts from the new state ([§6.5](concepts/v2-concept.md#65-landing)). A material model problem ends the Run without landing. Either way the host writes a Result page that lists every verdict. Human acceptance stays the project's own policy.

`verify` runs the same way with checkers only; they write the verdicts and change no file ([§7](concepts/v2-concept.md#7-commands-that-start-agents)).

**The same method at every scope.** Each parent coordinates and integrates its direct children. Only the Packages the change reaches take part, and `--full` brings in the whole tree. The checklist makes every reached item need a verdict; whether a verdict is right is a matter of review and evidence, not of mechanical proof.

**Rules, process and evidence.** A precise rule can have a deterministic check. A broad rule, such as consistent error handling, needs a declared surface, a concrete interpretation and independent evidence; a few examples do not prove it everywhere. Check the agent's process as well as its files: a compliant final file does not prove that the agent started from the model, used its context or handled intent first. A planted defect shows only that a check catches that defect. How often the method works needs repeated real tasks ([measurement](guides/measurement.md)).

**Adoption.** For an existing project, inferred intent is a proposal ([product definition](#product-definition)). Keep valid bytes, record unknown ownership and missing evidence, and never change the model just to bless accidental drift.

## What executable governance means

One canonical owner exists per modeled engineering fact or representation decision. Rationale and guidance may be prose inside the project model. Unadopted narrative, vendor material and specialist observations remain outside accepted intent until explicitly incorporated. The project tree makes responsibility, artifact relationships and decision scope visible; it does not infer source-code semantics or turn every repository file into a model fact.

The new structural Core does not contain an engineering assertion DSL. Checks, policies and assurance live above Core; the historical finite Domain/policy contract belongs to the earlier Project/Domain line, which is not developed further ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)). Project-owned checks and target tools can provide specialized implementation evidence. Rule and Workflow prose remains instruction and rationale where no executable assertion exists; writing it in a Definition does not enforce the process or prove that an agent followed it.

Context helps agents find applicable intent; impact helps reviewers understand declared consequences; versioned packages make architecture evolution explicit; narrow exceptions record deliberate deviations; reconciliation can align owned representations. None proves that all relevant intent was modeled, that an agent obeyed it, or that arbitrary application code is correct.

Continuous autonomous agents are the desired consumer operating model. They may call bounded Markitect operations from their own runtime, orchestration or CI. An Executor produces a candidate representation using appropriate tools, including deterministic renderers when useful: permitted implementation freedom must be explicit and project-owned checks must state what they establish. This does not require a continuously running Core, hidden policy activation or a background mutation controller. The [delivery plan](implementation-plan.md) defines the current sequence. `apply` and `verify` are on Main ([project workflow](guides/project-workflow.md#7-apply-and-verify)); the other Commands of [§7](concepts/v2-concept.md#7-commands-that-start-agents) come after acceptance. V1's Runs and the earlier Pass/shared-tree implementation are retired; their design is in the [V1 concept](concepts/v1-concept.md) and their source in Git history.

## Desired operating model

Humans should eventually review decisions at the level of architecture, ambiguous requirements, unresolved policy conflicts, deliberate exceptions and exceptional risk. Agents should handle ordinary implementation details within the project's declared freedom.

The organization must make a justified no-change result, a useful objection and timely escalation count as good work. Different stakeholders and perspectives need distinct questions and the context to answer them; adding Agent titles or rewarding completion alone supplies no such control. These are motivations for the existing [team direction](concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes), not a requirement to add more roles, separate hierarchies, scores or incentive machinery to `apply`.

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

## Origin and rationale

This section gives the causal account of why Markitect was conceived and how its mechanisms are expected to address the originating problems. It organizes the product owner's [retrospective account of 15 October 2026](concepts/sources/owner-origin-20261015.md), earlier [product description](concepts/sources/owner-product-description-20261009.md) and [restatement of the foundations and goal on 17 October 2026](concepts/sources/foundations-and-trust-20261017.md), and traces the progression from conflicting code, documentation and agent skills to a canonical model, structural compilation, delegated Managers and the later Government idea. The sequence is the owner's account; the dates when its individual stages first occurred are not established here. The account separates experience, mechanisms and hoped-for effects and adds provenance to the register's entries; it does not create a second decision register, schedule work or claim that the method has demonstrated its benefits.

### The originating project

The owner was developing a project with explicit Core, Module and Product boundaries. Its rules included Core depending on nothing, Modules depending only on Core, Products combining and adapting Modules, and Modules remaining independent of individual Products. These were the adopting project's architectural choices, not a required architecture for every Markitect project.

The owner also wanted to specify the working method, review policies, a documentation location for each use case, the skills belonging to a module, an Area, required documentation and its shape. The intended system therefore covered more than application behavior: architecture, development practice and the relationships among code, documentation and contributor guidance all mattered.

In the owner's experience, those representations repeatedly contradicted one another, with the drift that [Founding intent](#founding-intent) describes. Changes missed or violated rules, and documentation suffered from what the owner called "Bleeding". This account preserves that term without inventing a precise failure classification for it. These are reported observations about the originating project, not a measured failure rate for AI development in general.

### How the idea followed from the problem

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

Manager coordination realizes the accepted model. Government would additionally decide some changes to that model within a delegated mandate. The user's president/court analogy describes the desired division of work and escalation; it does not mandate a political structure or confer authority on an agent. [DEC-004](concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) and [IDEA-002](concepts/register.md#idea-002-government-delegated-model-maintenance) preserve that distinction.

### Problems the account identifies

| Problem | What was difficult for the owner | Existing connection |
|---|---|---|
| Unclear authoritative intent | Contradicting representations obscure which rule currently applies. | Canonical model and delegated realization: [DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product). |
| Missed continuing obligations | A change can work locally while violating another architecture, documentation or workflow requirement. | Reported omission problem: [ASM-001](concepts/register.md#assumptions-and-hypotheses); continuing specification: [CPT-004](concepts/register.md#cpt-004-the-specification-stays-effective). |
| Late discovery of structural problems | Relationships and invalid combinations emerge only while work is being performed. | Limited compiler analogy: [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept). |
| Concentrated cognitive load | One agent must hold the whole project's relevant rules and files in mind. | Bounded responsibility and integration: [CPT-001](concepts/register.md#cpt-001-federalism). |
| Repeated manual synchronization and review | Consolidation and reading changes consume time without giving a lasting overview. | Human focus and comprehensibility: [PRM-004, PRM-006 and PRM-007](concepts/register.md#promises). |
| Remaining model-maintenance work | New work items and technical findings still need interpretation and decisions about the intended system. | Delegated model maintenance: [IDEA-002](concepts/register.md#idea-002-government-delegated-model-maintenance). |

This account does not establish that forgetting is always more important than misunderstanding. Both an omitted relationship and a wrong interpretation can produce drift; [ASM-006](concepts/register.md#assumptions-and-hypotheses) remains a separate, untested assessment hypothesis.

### Concepts and their intended contribution

| Concept | What it means in this rationale | Why it might help; its limit |
|---|---|---|
| Canonical project model | The applicable project intent has an authoritative owner; realizations reference it rather than independently redefining the same rule. | Removes ambiguity about which statement governs. A wrong or incomplete model can still be implemented consistently. |
| Typing and structural compilation | Explicit identities, types and relationships can be validated before realization. | Moves expressible structural failures earlier. It does not prove arbitrary prose, model completeness or software correctness. |
| Explicit realization relationships and impact | Describe what belongs to what, including a rule's implementation, tests, documentation and guidance. | Makes declared consequences traceable. Undeclared relationships and code dependencies can still be missed ([ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact)). |
| Managers and bounded delegation | Durable responsibility supplies a limited task and context; the parent remains accountable for integration. | Reduces reliance on one all-remembering "God Agent". A hierarchy alone does not ensure good task boundaries or correct composition. |
| Independent review and verification | Assess both individual contributions and their integration against the applicable obligations. | Supplies additional evidence and opportunities to find omissions. Reviewers can share blind spots; tests are realizations too ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)). |
| Apply, Verify and Cleanup | Realize intended changes, examine conformity and improve the repository within the applicable model. | Offers repeatable work against a known basis. Concept 2.0 makes them Commands ([§7](concepts/v2-concept.md#7-commands-that-start-agents)); none promises unattended acceptance. |
| Government | Delegate selected model-maintenance decisions, preserve human reservations and report relevant matters. | Could reduce routine model work. Correct procedure does not prove that the selected intent is right; the model may grow harder to maintain. |

The goal is to use explicit structure and ordinary deterministic software where they fit, then give agents bounded interpretation and realization work. Deterministic and semantic checks have different jobs. The compiler analogy is borrowed from programming languages; it does not make a natural-language specification executable or complete merely by attaching types.

### Foundations restated on 17 October 2026

The marketing discussion also records the owner's [conceptual influences](concepts/sources/conceptual-influences-20261017.md) and [product-goal clarification](concepts/sources/product-goals-clarification-20261017.md). These supply additional provenance for the foundations below: lower cost is not the product goal, and routine human implementation review should become unnecessary through trustworthy realization.

The owner named the following influences and reaffirmed the goal: a clean, uniform and clear repository, without drift, forgotten rules, inconsistencies or contradictions; model first; trustworthy AI results through system and structure; ultimately modeling followed by AI realization that needs no routine human code review. The [source](concepts/sources/foundations-and-trust-20261017.md#owner-statement) preserves the wording. The table connects those influences to existing choices; it introduces no new roles, language semantics or delivery scope.

| Foundation | Intended contribution and current design connection |
|---|---|
| Four eyes and different perspectives | An independent reviewer asks whether the candidate satisfies its obligations; a parent's integration review asks whether the parts work together. Distinct questions matter more than Agent count. [CPT-001](concepts/register.md#cpt-001-federalism), [ENH-006](concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes). |
| Responsibility, delegation and escalation | Each Manager owns a bounded area and integrates its direct children. Questions and issues reach the responsible owner. Impact decides which Managers take part, and the host enforces the order and the hooks; V1 had every Manager take part in a fixed process. [DEC-025](concepts/register.md#dec-025-package-managers-organize-realization), [DEC-082](concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-039](concepts/register.md#dec-039-deterministic-manager-process). |
| Divide and conquer; abstraction and need to know | Decompose meaningfully, give specialists sufficient local detail, and let parents combine that detail into a broader view. Summaries and targeted reads connect breadth and depth; bounded context must not hide a needed contract. [DEC-028](concepts/register.md#dec-028-context-follows-need-to-know-and-starts-simple), [OQ-006](concepts/register.md#open-questions). |
| Model and projections; one source of truth | The repository's code, tests, documentation and configuration realize one intended world. They may have different shapes while expressing the same intent. A generated readable view is a projection with an exact rendering contract; Agent-written code needs semantic assessment. [DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [CPT-003](concepts/register.md#cpt-003-two-parts-of-the-promise). |
| Planning, system and structure; trust the process | Deterministic compilation, impact and the hooks handle explicit structure and the guarantees of a Run. AI supplies bounded interpretation and realization judgment. Trust depends on useful findings, corrections and faithful results across the whole workflow. [DEC-082](concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager). |
| Stakeholders and incentives | Different concerns should be represented by distinct questions and access to relevant evidence. Grounded objections, justified no-change and escalation are useful outcomes. Incentive design is an influence, not an implemented scoring or reward system. Each working Manager is a maker plus a fresh checker. [Organization and trust](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/reflections/organization-and-trust-20261016.md#limits-of-the-analogy-and-future-autonomy). |
| Type systems and compilation | Move structural failures into preparation: invalid types, references and combinations should fail before Agent work. Prose obligations still require interpretation and independent assessment. [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept). |
| Many capable lower-cost Agents | Bounded work and systematic collaboration should let models such as Luna, Sonnet or Haiku contribute instead of concentrating everything in a single expensive Agent. Suitability, total coordination cost and result quality remain questions for evidence. [DEC-007](concepts/register.md#dec-007-more-economical-models-means-capable-mid-tier-models), [ASM-003](concepts/register.md#assumptions-and-hypotheses). |
| Documentation-driven and specification-driven development; Clean Architecture and the Dependency Rule | Make intended meaning explicit, separate responsibilities and enforce dependency direction. The Markdown origin explains the name; the model is Markdown typed by Kinds ([DEC-080](concepts/register.md#dec-080-the-model-is-typed-documentation)). These influences do not prescribe one software architecture for every adopting project. [Origin of the name](#origin-of-the-name), [architecture](architecture.md). |

The goal of removing routine human code review concerns the owner's work: [Markitect in brief](#markitect-in-brief) states it, [PRM-004](concepts/register.md#promises) records it, [Common misreadings](#common-misreadings) keeps independent Agent review and the human responsibilities beside it, and [Measurement](guides/measurement.md#human-attention-and-delegated-work) owns how reduced human intervention is assessed. Neither a typed model nor agreement among Agents proves that the intended world is complete or right.

### Assumptions and hoped-for effects

The following are an editorial synthesis of why the owner expects the method to help. They are not new accepted register entries or measured results. References retain the existing statuses; the counter-observations make the reasoning open to correction without creating a new study or gate.

| Assumption or expected effect | Existing basis | Observation that would challenge it |
|---|---|---|
| One authoritative statement with maintained realization links reduces synchronization mistakes. | [DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [PRM-003](concepts/register.md#promises). | Contradictions still accumulate, or maintaining the model and links costs as much as the earlier consolidation. |
| Typing catches useful structural errors before agent execution. | [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept). | Most consequential failures pass structural checks and still require the same late global investigation. |
| Declared relationships provide useful scope and preserve continuing obligations. | [PRM-001](concepts/register.md#promises), [CPT-004](concepts/register.md#cpt-004-the-specification-stays-effective), [ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact). | Relevant consumers or representations fall outside impact, or conservative scope repeatedly expands to nearly the whole project. |
| Bounded Managers and parent integration make complex work more dependable. | [ASM-004](concepts/register.md#assumptions-and-hypotheses), [CPT-001](concepts/register.md#cpt-001-federalism). | Handoffs introduce errors or coordination cost without improving coverage or integration quality. |
| Independent reviewers reveal mistakes the implementing agent misses. | [ASM-002 and ASM-005](concepts/register.md#assumptions-and-hypotheses). | Reviewers repeatedly accept the same omissions or wrong interpretations; extra calls add no useful evidence. |
| Bounded context lets capable, less expensive models handle routine work efficiently. | [ASM-003](concepts/register.md#assumptions-and-hypotheses), [PRM-005](concepts/register.md#promises). | Rework, review, integration and model upkeep consume any savings. Lower total cost is a possible benefit, not the primary success condition. |
| Model maintenance and explicit evidence reduce the need to inspect every routine PR while preserving control. | [PRM-004, PRM-006 and PRM-007](concepts/register.md#promises). | The owner still needs exhaustive diff review, cannot identify unverified obligations, or trades that work for equally expensive model supervision. |
| Government can further reduce routine attention without degrading intent. | [IDEA-002](concepts/register.md#idea-002-government-delegated-model-maintenance). | Autonomous decisions misinterpret needs, exceed delegation or create more rules, review and escalation work than they remove. |

The existing [measurement procedure](guides/measurement.md#human-attention-and-delegated-work) compares quality, omissions and human attention as well as the full cost of setup, modeling, implementation, review, integration and recovery. Success cannot be inferred from more agents, more documents, green self-authored tests or a well-formed decision receipt. Current evidence remains in the [evidence summary](#current-basis-and-unproven-benefit) and the source-bound [validation records](validation/README.md).

### What this account settles and leaves open

The owner confirms the reason for the product: maintaining a clear intended world and using structure, typing, delegated responsibility and checks to make its realization more controllable. Government follows as a possible extension to the maintenance of that world. The original project rules explain the need; they are not prescribed Markitect vocabulary.

The [model language guide](guides/model-language.md) and [concept 2.0](concepts/v2-concept.md) own the language and the realization rules. A project still chooses its model granularity, intended relationships, implementation freedom and acceptance policy. The owner's restatement clarifies the motivation and goal; it adds no runtime mechanism or authority for autonomous model changes, and it does not establish product superiority over conventional agentic development.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, Rules, Skills, Workflows and agent instructions. Its direction is to turn informal engineering knowledge into structured, versioned architecture and policy, then into an executable engineering system. The name records that origin. Concept 2.0 makes the model Markdown again: each Definition is a Markdown file whose short front matter names its Kind ([DEC-080](concepts/register.md#dec-080-the-model-is-typed-documentation)). The origin explains the name; it does not limit the product to agent instructions.

## Common misreadings

The readings below came up repeatedly, most recently in the [idea review of 10 October 2026](concepts/sources/idea-review-20261010.md). Check an assessment against this table before treating any of them as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the name comes from. The product is model-first development of the whole project with delegated realization. Agent instructions are one kind of realization among many. |
| The deterministic parts are the product, and Manager execution is scope creep. | Delegated realization is the purpose, and deterministic preparation supports it. Providers sit behind a provider-neutral boundary, and every Manager works in its own container ([DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [DEC-083](concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers), [delivery plan](implementation-plan.md)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed concept, and the trust bar is today's practice ([DEC-002](concepts/register.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept)). |
| The Manager hierarchy is the Government experiment. | The Manager hierarchy belongs to the core axis, Model → Repository. Government belongs to a later axis, in which management levels decide model changes within mandates. Results of the former Government experiment arm are not results about the Manager hierarchy ([DEC-004](concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)). |
| "More economical models" means weak models. | It means capable mid-tier models such as Luna or Sonnet, and Haiku only if it proves sufficient ([DEC-007](concepts/register.md#dec-007-more-economical-models-means-capable-mid-tier-models)). |
| Earlier comparisons showed that conventional agentic coding wins. | Environment, runtime and harness failures dominated the October 2026 comparison attempts, and none of them exercised the delegated method over a series of changes. They allow no conclusion about the concept. Earlier matched comparisons tested an earlier product; their findings, including the cost of model upkeep, stand ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). |
| The model has to mirror the code, with one definition per file. | Granularity is the author's choice. A file may realize several definitions, and every file must be covered or explicitly ignored ([DEC-005](concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility), [DEC-006](concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). |
| Green agent-written tests prove the realization. | Tests are realizations too. Agents write them, so they can be wrong. Reviewing them against the statements they claim to verify is an agreed direction ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)). |
| The goal of removing routine human code review removes the four-eyes principle. | Fresh checkers, parent integration, the hooks and the required checks are the means to earn that trust. People retain model decisions, escalation and their project's acceptance policy ([PRM-004](concepts/register.md#promises)). |

## Feature decision filter

Ask: **Does this improve fidelity to accepted intent and reliable autonomous reconciliation inside explicit boundaries, with enough independent evidence to reduce routine human supervision?**

A proposal should identify the human activity it aims to remove, its canonical owner, the agent's remaining freedom, the exact evidence/check boundary and the new authoring or maintenance cost. Deterministic checks, task context, impact, policy evolution, exceptions, discovery and reconciliation can support this goal, but each needs a concrete workflow and evaluation.

Reconsider proposals that introduce another manually synchronized truth owner, hide architectural decisions in inference, specialize Core for one provider, or add machinery without evidence that coordination gets easier. More generated text, adapters, language operators or agent throughput alone do not demonstrate progress. This filter does not override the constitution's bar for new semantics or authorize a new feature wave.

For concept 2.0, the [acceptance criteria](concepts/v2-concept.md#15-acceptance) ask, on a small and a larger project over a series of changes, whether every landed change is correct, no drift is left, nothing is skipped, planted model defects stop the Run, and the results are at least as trustworthy as today's practice: one strong agent with ordinary documentation. The counted series claims only what it measures; time saved by people is measured apart, after it ([§15](concepts/v2-concept.md#15-acceptance)). A successful run is bounded feasibility evidence. Reduced routine human review and dependable use of cheaper models need repeated quality and attention evidence under [Measurement](guides/measurement.md#human-attention-and-delegated-work). Keep the first strategy small enough to learn this before adding roles, parallelism or reuse mechanisms.

## Current basis and unproven benefit

| Category | What is established or intended |
|---|---|
| Product direction | A single model-first user workflow centered on the recursively managed `.markitect/` project tree. The [roadmap](implementation-plan.md) records exact source and release status. |
| Current source | Concept 2.0 is being built in the [delivery plan's steps](implementation-plan.md#concept-20-build): the Markdown model language and its commands, the Run remote and hooks, the Run engine with `apply` and `verify`, and the native host are on Main; the pilot and the counted series are open. V1's design is in the [V1 concept](concepts/v1-concept.md), its source in Git history. The preceding source generation remains in [Git](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). |
| Published compatibility | The earlier Project/Domain CLI and Projection alpha remain in the pinned pre-reset commits listed in the [archive index](archive.md); the former Git tags no longer exist. Their release pages and binary downloads were retired by the owner on 17 October 2026. The contracts and recorded evidence are historical; there is no second active product UX. See the [production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md#v0141-published-release). |
| Acceptance boundary | A committed model revision is the accepted specification used for that revision. Commit/provenance values bind data but do not authenticate a human decision. Passing checks, generated docs and agent reports are evidence with declared scope, not approval. |
| Aspirational operation | More parallel Managers, less routine supervision, architecture-level escalation and sustainable governance at greater scale. No universal autonomous acceptance is claimed; container isolation has its own boundary under [DEC-083](concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers). |
| Unproven benefit | Reduced human intervention, lower total maintenance cost, fewer missed obligations or defects, better sustained autonomy, token savings and market demand remain unmeasured. Historical comparisons are preserved, including their negative or inconclusive findings. |

The [real-code adoption pilot](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/real-project-adoption-pilot.md), [AGENTS.md comparison](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

These records evaluated earlier product forms, not the delegated Manager method over a series of changes. The October 2026 comparison attempts of the delegated method were dominated by environment, runtime and harness failures. They allow no conclusion about the concept, positive or negative ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). Markitect's product owner has endorsed evaluating the method separately from any single execution runtime; the plan is pending ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)).

The central evaluation question is: **Does ontology-driven reconciliation preserve intent and project quality more reliably than conventional agentic work, and thereby support less routine human supervision?** [Measurement](guides/measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative. Markitect's product owner phrases the same question in [DEC-009](concepts/register.md#dec-009-the-central-evaluation-question): when people develop against a model, can the code reliably reflect that model? Can the repository stay contradiction-free, consistent, uniform and clean? Does agentic coding become meaningfully more trustworthy? Can people concentrate on deciding and modeling?

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.


## Research refinement

The [ontology, projection and assurance thesis](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/research/README.md) is owner-supplied research orientation. Its distinction between semantic authority, implementation freedom, independent evidence and integration obligations guides experiments. The [alignment assessment](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/projection-assurance-direction.md) identifies what this source slice supports and what remains research. It neither expands Core by naming concepts nor changes measured evidence into a benefit claim.
