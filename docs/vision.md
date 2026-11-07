# Product vision

This is the canonical owner of Markitect's product thesis, intended division of responsibility, delegated method, origin and long-term operating model. The [documentation map](README.md#canonical-status-and-evidence-owners) names the owners of the other subjects, such as technical boundaries, delivery status, evaluation and individual decisions. This vision does not authorize a feature, weaken an acceptance gate or establish a measured outcome.

## Markitect in brief

Read this section first, then [Founding intent](#founding-intent). The rest of this page develops them, and the [concept register](concepts/register.md) holds the individual decisions behind them.

**What it is: a programming language for documentation.**
- People maintain one authoritative place per project: the spec, which code and older documents call the canonical model. It is typed prose: Markdown Definitions, each of a Kind, organized in Packages, with links that work like imports. It states the project's concepts, rules, boundaries, interfaces, architecture and way of working, as far as good documentation of the project would.
- Code, tests, configuration and documentation are realizations of the spec. Where they differ from it, the spec is right.
- A deterministic tool, the `markitect` program, checks and analyses the spec as a compiler and a linter check code: `check` compiles it and accounts for every file of the project, `vet` measures its design, and `status` and `mark` keep a record of what was verified. Agents realize the spec and verify the result.

This is the owner's direction of 7 and 8 November 2026 ([VIS-002](concepts/register.md#vis-002-a-programming-language-for-documentation)). The owner expects coding to move from languages such as Go, C# or Python towards documentation and meaning. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

**What it builds on: proven practice.**
- **Requirements traceability.** Tools such as IBM DOORS and Polarion keep a traceability matrix that links requirements to what realizes them, in both directions, and mark a link suspect when either end changes; DO-178C, ISO 26262 and IEC 62304 require such traceability. In Markitect the registry is the matrix: one page per Package links every file to the Definitions it realizes, and `markitect doc` shows a Definition's links both ways. `markitect status` lists the suspect items, every Definition whose text, Kind, links, files or evidence changed since a reviewer last verified it, and `markitect mark` clears one with a verdict and its evidence.
- **Architecture metrics.** The spec's design is judged with the principles that keep code changeable: Clean Architecture's Dependency Rule, domain-driven design's bounded contexts, cohesion and coupling, and Robert C. Martin's stability measures ([what makes a good spec](guides/good-spec.md)). `check` refuses a link from a Package to a Definition inside its child, and a cycle between Packages; `vet` warns about coupling, size, reach and unstable dependencies. Change hotspots in Adam Tornhill's sense need a project's history, which the tool does not read, so they stay a later idea ([`vet` plan](concepts/vet-plan.md)).
- **Go's conventions.** The command line follows Go's tools: `vet`, `list`, `doc` and `tidy` do what their Go namesakes do, diagnostics read `file:line: code: message`, and exit codes follow Go's ([tooling plan](concepts/tooling-plan.md#the-commands)).

**What is new.**
- The traceability is plain text in the repository, beside the code: Markdown that people and agents read directly, typed by Kinds and checked deterministically, with no database and no separate tool to keep in step.
- Agents do the realization and gather the evidence. A builder makes the project match the spec, and a fresh reviewer verifies each listed Definition and records its verdict with the evidence it read.
- The record of what was verified, `.markitect/review.sum`, works like Go's test cache: a verdict holds until one of its inputs changes, so a change puts back on the list only what it reaches ([DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)).

**How a change goes** ([concept 4.0, §5](concepts/v4-concept.md#5-what-build-does)).
- A person changes the spec, usually with a coding agent through the `/markitect-design` skill, which asks first when something is open. The person reads the Markdown diff and commits it.
- `/markitect-build` has the person's own Claude Code session start a builder agent in a worktree of its own, so the person can go on designing. The builder makes the project match the spec for what `markitect status` lists and starts a fresh reviewer agent, which records a verdict for each item with `markitect mark`, bound to the state it read. A `fail` stays on the list until the builder has repaired it and it is verified again.
- When `status` lists nothing, the project compiles and every Definition is verified at its current inputs. The session commits the work with the record and merges it into the person's branch.
- `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt` and `/markitect-audit` verify, refactor, hunt for bugs and read the spec itself for contradictions and gaps, and `/markitect-schedule` sets them up as routines. Four roles, the builder, the reviewer, the architect and the auditor, have their rights held by a hook ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)).
- Markitect itself starts no agent, steers no work and uses no Git. It has no Run, no gate and no state outside the project ([DEC-096](concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](concepts/register.md#dec-097-markitect-uses-no-git), [DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)).
- A repair that leaves intent unchanged is realized against the same spec, without an invented spec edit.

**The approach: lighter, native and simpler.** Markitect works with Claude Code and Anthropic's own features rather than against them: skills, subagents, hooks, worktrees, a plugin and Claude Code's own scheduling. Earlier versions built their own machinery around the agents: a Manager agent per Package, containers, a Run that held the work and landed it, and a `gate` command that printed each next step. Each version since has dropped some of it ([concept 4.0, §15](concepts/v4-concept.md#15-what-this-overturns)). In the owner's experience the simpler version is the more successful and the more robust one, and that simplicity is earned through insight, not assumed at the start.

**The honest limit.** "Complete" means complete for the declared links. `status` follows a Definition's own text, its Kind, the Definitions it links to, its registry rows and its evidence, one step deep; it does not read the code's imports ([ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact)). Prose that names a concept without linking it is a gap: a change to that concept does not put the naming Definition back on the list. The [`vet` plan](concepts/vet-plan.md#unlinked-mentions) proposes that `vet` flag such names as `vet.unlinked_mention`; the owner has not decided it yet, so until then only a careful reader finds them. An empty `status` says that every Definition was verified at its current inputs, not that every verdict is right.

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

The intended end state is **model intent → AI realizes and independently reviews it → people can rely on the result without routinely reviewing code diffs themselves**. Automated four-eyes review and integration remain part of the process. Human attention goes to model quality, architecture and escalated decisions. This is the owner's [restated goal of 17 October 2026](concepts/sources/foundations-and-trust-20261017.md#owner-statement), already recorded in [PRM-004](concepts/register.md#promises); it is a target to earn through evidence, not a claim that the current version has achieved it. The project's acceptance policy still applies.

These are [promises](concepts/register.md#promises), not established results.

**How to judge it.**
- Judge Markitect's trust against today's practice, not against a compiler's guarantees. Comparative studies still use a strong conventional workflow that is allowed to win.
- "Compiler" names a borrowed idea. The spec's structure compiles deterministically: a spec with errors gives no list to verify, and a file that belongs nowhere is an error. Realization by agents is not deterministic.
- Markitect's claim is that a typed spec, a list of what to verify computed from it rather than from an agent's memory, a verdict with evidence for every listed Definition, roles whose rights a hook holds and a reviewer with fresh context make delegated work more trustworthy than unstructured agentic work. [Where trust comes from](#where-trust-comes-from) lists these mechanisms.
- Delegated realization is central to the product. It is not an optional layer on top of a validator.
- Markitect's product owner has endorsed separating the method from any particular execution runtime. The tool computes and checks, and a driver, by default the person's own Claude Code session on its account login, starts the agents; the `markitect` program runs natively, with no container. The current target is [concept 4.0](concepts/v4-concept.md) ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)); the [delivery plan](implementation-plan.md) owns what is built. A smoke or a source test alone does not establish acceptance. See [DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime).
- [Common misreadings](#common-misreadings) lists the readings to avoid.

## Founding intent

The owner's [founding intent](concepts/sources/founding-intent-20261026.md) explains why Markitect exists and what the canonical model is meant to be. Earlier accounts are the [origin account](concepts/sources/owner-origin-20261015.md) and the [conceptual influences](concepts/sources/conceptual-influences-20261017.md); [Origin and rationale](#origin-and-rationale) organizes them.

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. Every consistency run took hours and found more inconsistencies. The owner lost track of which rules existed, what was defined and what had entered the project silently.

**The answer is a typed language in prose.** In a strongly typed language, a compiler and explicit dependencies report issues at compile time instead of leaving them to be found later. The canonical model applies that to project intent:

| Typed language | Markitect spec |
| --- | --- |
| type, interface | Kind |
| declaration | Definition |
| function body | the Definition's prose: binding text that agents realize |
| import | link: `[[members/member]]` names its target by path |
| package, module | Package |
| standard library, modules | Libraries, named in the manifest and pinned by digest in the lock, as `go.mod` and `go.sum` do |
| compiler, type checker | `markitect check`: structure only, never meaning |
| `go vet`, architecture metrics | `markitect vet` |
| test cache | `.markitect/review.sum`, the record of what was verified |

The owner first gave this analogy for V1's typed YAML, which also had typed Properties as fields, and the last two rows came with the record and the Go names the owner chose for the tool ([DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). The Markdown language of today has no Properties, so exact values such as error codes are written in prose. It has no import statement either: each link is itself the declared dependency, an edge between Packages, as an import is in Go ([model language](guides/model-language.md#links-scope-and-visibility)). A Kind is itself a Definition of Kind `Kind`, a Definition is one Markdown file with front matter, and a Package is a folder.

- The core stays clean. Projects define their own Kinds and so form their own language; Libraries supply common Kinds, oriented on what Go or C# ship, so that no project reinvents them. `std`, `rest` and `cli` ship in the binary.
- The compiler checks structural correctness from the core alone. Meaning is judged by agents: the reviewer judges the code against the spec, and the auditor judges the spec itself.
- The spec is organized as code is: Packages are modules, links go up the tree or sideways towards stability and never down, and a Package's Definitions are in its area's own words, after Clean Architecture, domain-driven design and modularity ([what makes a good spec](guides/good-spec.md)). Concept 2.0 also gave each Package a responsible Manager agent; that tree went with concept 3.0 ([DEC-091](concepts/register.md#dec-091-concept-30-is-the-target)).
- People decide intent. AI agents write the spec in conversation through the design skill, which asks first when something is open; the owner reads the Markdown diff and accepts it. The same skill draws a first spec from existing code. It ships inside the `markitect` binary, and `markitect init` installs it ([concept 4.0 §8](concepts/v4-concept.md#8-other-commands)).

**Consequences.**
- Judge a spec as you would judge code: dependencies, coupling and stability, mutation testing, CRAP, clean architecture and size. A Definition of 500 words is prose compensating for missing Definitions. [What makes a good spec](guides/good-spec.md) states the rules and their measures.
- How explicit a spec is, is settled: as explicit as good documentation of the project ([DEC-079](concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). The standard Libraries and the design skill exist, and `vet` measures a spec like code ([model language](guides/model-language.md#vet-warnings)). Whether they lead to good specs in practice is still to be shown. Evaluations of Markitect are only as good as the spec they use.

## Primary goal and prerequisites

The owner's [21 October clarification](concepts/sources/primary-goal-and-conformance-evaluation-20261021.md#owner-statement-primary-goal) makes the priority explicit: earn trust in Agents through structured, systematic realization and independent checking of the canonical model across **all project artifacts**. Nothing material should be overlooked or contradicted. Responsibility is divided by area; independent local review and parent integration/review must establish how the parts fit. Sustained consistency, uniformity and cleanliness are the resulting long-term project outcome. This clarifies the existing thesis; it does not establish achieved reliability or commission optional cleanup of conforming artifacts.

Three prerequisites belong to that goal: the model can adequately express the intended software project; its structure and meaning can be assessed for consistency, completeness relative to explicit purpose, and uniformity; and people can maintain it comfortably, mainly through Agents and readable views. A self-consistent model cannot reveal an entirely omitted user need without that purpose or requirement being supplied. Readable progress and results should expose assessed scope, evidence and unresolved matters, rather than imply universal correctness through a green status alone.

The record of what was verified carries the semantic obligation: every Definition whose inputs changed needs a verdict with evidence from a fresh reviewer ([DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)); [measurement](guides/measurement.md#evidence-for-the-primary-goal) defines discriminating evidence. Neither an empty `status` nor successful tests alone establish that every material condition is realized.

## Proportionate Agent process

The owner's [22 October clarification](concepts/sources/proportionate-agent-process-20261022.md) reaffirms DEC-002: the practical baseline is giving a coding Agent a task and trusting its work. Markitect should improve that trust through explicit model intent, bounded ownership, independent perspectives and parent integration. It is not a formal proof system. Agents judge meaning and parents review responsibility and integration; the Host supplies execution, communication, retained results and finite limits. The clarification asked for no technical write protection at every boundary. Concept 4.0 enforces a few boundaries technically: a hook holds the four roles' rights for subagents, `mark` binds a verdict to what the reviewer read, and `status` gives no list while the spec has errors ([§7](concepts/v4-concept.md#7-how-sure-we-are)).

Use the smallest process that supports those responsibilities. More report fields, mandatory probes or completeness declarations do not themselves improve correctness. Tests and inspected artifacts support professional judgment; uncertainty about a concrete required property remains visible, but an imagined possibility of another defect must not create endless review. Parents assess composition and challenge relevant child evidence without routinely repeating all child work. Judge the method by actual consistency, detected and repaired defects, false findings, completed workflows and human intervention against the ordinary Agent baseline.

Markitect itself uses no Git ([DEC-097](concepts/register.md#dec-097-markitect-uses-no-git)). The owner's session commits the builder's work with the record on the worktree's branch and merges it into the owner's branch, and the owner pushes. Concept 2.0 handed work on as Git commits through a Run's remote ([DEC-078](concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)), after the owner's [adoption of Git handoffs on 22 October](concepts/sources/agent-workspaces-and-handoffs-20261022.md) had replaced V1's Pass/Turn and shared-tree design; both, and V1's [handoff workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-workflow.md), are history. Documentation, implementation and demonstrated benefit remain separate.

## The thesis

> **Define the desired project world once. Reconcile its representations without losing intent.**

The primary goal is quality and consistency during autonomous engineering: fewer forgotten obligations, overlooked consequences, conflicting representations and ignored rules. The project ontology expresses accepted concepts, relationships, purpose and constraints; the governed repository is its representation. A change to that world should drive impact, fanout, reconciliation and independent verification. [The delegated method](#the-delegated-method) defines that cycle.

Human attention is the central bottleneck. Better independent evidence and explicit delegation should reduce routine supervision while preserving human ownership of meaningful decisions. The owner's [priority clarification](concepts/sources/human-attention-and-baseline-20261017.md#attention-and-time) treats calls to capable lower-cost models such as Luna as inexpensive relative to a developer inspecting work, detecting an omission and prompting repairs. Additional Agents are justified when they improve fidelity and reduce that attention; Agent count alone is no defect or success metric. Token and monetary cost remain observations, not the primary optimization target.

The practical question is: **Does what the AI built actually match what I intended?** The organization should provide a dependable answer without requiring routine manual inspection or an extra review request made just for reassurance. Agents can overlook obligations as well as misunderstand them; responsibilities, compiled context, independent perspectives and integration address both. Human management practice inspires the structure. Its effectiveness with Agents is what the product must demonstrate.

Elapsed time is a separate concern. Parallel work can shorten the critical path where responsibilities and dependencies permit it. The builder works in a worktree of its own while the owner goes on designing, and reviewers can work in parallel, one per Package; the effectiveness and scalability of this remain to be demonstrated. Measure actual human attention and wall-clock time separately, including review, correction and waiting.

Markitect's working assumption is that AI implementation capacity will grow and become increasingly parallel, while human engineering judgment remains scarce. If humans must implement, synchronize and review every routine change, that attention becomes the bottleneck. Architecture matters more as implementation becomes cheaper: increasing output without explicit boundaries can also increase accidental coupling, inconsistent policy and architecture drift.

These are product assumptions to test, not economic or productivity conclusions established by this repository. Agent capability, task complexity, verification coverage and model-maintenance cost can change the outcome.

Humans should spend more attention defining goals, concepts, architecture, invariants, policies, processes, responsibilities and allowed degrees of freedom. AI agents should perform as much of the remaining implementation, testing, refactoring, documentation, migration, release work and routine maintenance as their granted authority and available evidence permit.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

The owner's [foundations](#foundations-restated-on-17-october-2026) connect model-first and specification-driven development, typing, architectural boundaries, divide and conquer, bounded delegation, escalation, independent perspectives and capable lower-cost agents. Their common purpose is a clear, uniform repository whose code, tests, documentation and configuration continue to express the same model. Typing catches structural inconsistencies early; the record lists every Definition a change reaches and needs a verdict for each; agents interpret meaning, realize the spec and review the result. Each part has a different job.

## Product definition

**Markitect is the spec of a project's intended world, a tool that checks it and records what was verified, and a default harness that has agents realize and verify it.** A project organizes concepts, rules, architecture, interfaces and decisions into Packages. Markitect keeps a registry, one Markdown page per Package, that names the paths the Package owns and links every realizing file to the Definitions it realizes; the builder keeps it current. Every other file is Markitect's own, such as `.markitect/` and the harness `init` installs, is excluded by the project's `.gitignore`, or is listed in Markitect's ignore file ([model language](guides/model-language.md#the-realization-record)). The spec lives under `.markitect/` in the adopting repository, as Markdown that people and agents read directly. Provider, editor and file-format choices are implementation details unless the project deliberately makes them part of its intent.

**Owners change the spec. Agents realize and verify it.** The flow is:

```text
design the change with /markitect-design, which asks first when something is open
    -> the owner reads the Markdown diff and commits the spec
    -> /markitect-build: a builder in its own worktree takes what status lists
    -> the builder makes the project match; the hook keeps it out of the spec and the record
    -> a fresh reviewer verifies each item and records it with mark; a fail goes back to the builder
    -> status lists nothing: the session commits work and record and merges them
```

The committed spec revision is Markitect's accepted repository specification for that revision. Drafts, distillation reports and uncommitted edits remain proposals. Git commits and digests bind bytes and history; they do not authenticate who approved the model or prove that a human reviewed it. Projects define their decision and acceptance process outside that machine identity boundary. Brownfield work can use existing artifacts to inform a proposal, but inference never grants the proposal authority. Owners review and correct the model, commit the selected specification, then map artifact scopes and assess existing realizations. Unmodeled, excluded, externally owned and uncertain paths remain explicit. A partial model must not imply repository-wide governance.

The product direction is an executable engineering-governance workflow for AI-assisted software development. Readable documentation, native agent instructions, bounded context, implementation tools and checks serve the model-first process. Their existence or volume is not the product's success criterion.

Markitect lets humans program the shape and rules of software development instead of repeatedly supervising each implementation detail. AI should implement the project's architecture rather than invent a competing architecture on every task. The desired benefit is faithful, higher-quality autonomous engineering with architectural authority remaining human-owned. More work per unit of human attention is a consequence to measure alongside quality, reliability and model upkeep.

## Three starting situations

There are **three starting situations, but only two directions: Apply and Extract**, accepted by the owner on 19 October 2026 ([DEC-057](concepts/register.md#dec-057-three-fundamental-starting-situations), [process naming decision](concepts/sources/three-starting-situations-20261019.md#owner-selects-apply-and-extract)). The spec describes intended meaning and structure; artifacts realize it.

| Direction | From and to | Goal |
|---|---|---|
| Apply | Spec -> Artifacts | Bring artifacts into agreement with the accepted spec, whether they must first be created or already exist. |
| Extract | Artifacts -> Spec | Develop a spec of the existing project's meaning and structure for the owner to accept, in conversation with the owner. |

| Situation | Typical use | Today |
|---|---|---|
| 1. Spec exists; artifacts do not | Start a new project by writing its spec first. | `/markitect-build` builds the whole spec, since nothing is verified yet. |
| 2. Spec and artifacts exist | Normal ongoing development and maintenance. | `/markitect-build` realizes what `status` lists. |
| 3. Artifacts exist; spec does not | Adopt Markitect for an existing project. | `/markitect-design` draws a first spec from the code with the owner, and may delegate a large draft to the architect agent. |

The `/markitect-design` skill sees which situation a folder is in: an empty project, code without a spec, or an existing spec ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). Building a new project and realizing a change are the same build; the first time, `status` lists every Definition. These names describe the high-level direction of work; the [project workflow](guides/project-workflow.md) owns how to use the skills.

**Build from the spec.** The spec need not prescribe every file or implementation step: it states the intended meaning, structure and deliberate constraints, leaving the agreed realization freedom. A gap with no effect at the project's boundary is the builder's to decide. Where the spec contradicts itself or leaves open behaviour at the boundary, the reviewer records a `spec-problem` with the question for the owner, and the Definition stays on the list until the owner settles it ([concept 4.0, §5](concepts/v4-concept.md#5-what-build-does)).

**Spec and artifacts exist.** This is the normal working state. Missing artifacts are created and existing ones are updated where required; files that already conform are kept. A Definition gone from the spec stays on the list, with the files that realized it, until they are removed or have a new basis in the registry. A spec change supplies new intent; a bug or other realization deviation is repaired against the unchanged spec, without an invented spec edit.

**Draw the spec from existing artifacts.** This is Extract, the reverse direction. For a spec drawn from existing code, the binding part is the contract callers rely on; details a caller can observe but does not rely on stay free ([DEC-061](concepts/register.md#dec-061-what-an-extracted-model-contains)). Observed behavior, proposed intent and open questions stay apart, and the owner accepts the result like any other spec change. Existing artifacts stay unchanged while the spec is drawn. Afterwards `/markitect-review --all` verifies every Definition and records where the code deviates, and `/markitect-build` repairs it. A confirmed realization defect remains a defect against the accepted spec; rewriting intent to obtain an empty `status` would defeat adoption.

The situations can coexist in different areas of one project, so every assessment names its scope. An empty folder enters situation 1 once a first spec has been written and accepted.

**History:** concept 2.0 realized a change in one Run, in which impact decided which Manager agents worked, Root prepared the project skeleton, children built their areas recursively and parents integrated, and the host landed what Root's checker approved ([concept 2.0, §6](concepts/v2-concept.md#6-runs)). V1 designed Extract as a process of the Manager tree, with provisional Managers, rounds and iterative Verify ([Extract workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md#extract-the-existing-stand)), and its [start-from-model direction](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) built recursively from Root. Both are records; the [V1 concept](concepts/v1-concept.md) holds V1's Runs.

## Human, Markitect and agent responsibilities

| Role | Responsibility |
|---|---|
| Humans | Decide what should exist and why; choose concepts, architecture, boundaries, invariants, policies and processes; define implementation freedom and acceptance requirements; maintain the project's approval process and resolve decisions outside delegated authority. |
| Markitect | Compile the spec, account for every file, say which Definitions are to verify because an input changed, record each verdict at the state the reviewer read, and keep agents from recording their own work or changing the spec. It starts no agent, steers no work and uses no Git ([DEC-096](concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](concepts/register.md#dec-097-markitect-uses-no-git), [DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). |
| AI agents | Explore and propose intent, implement assigned responsibilities, test and integrate within granted boundaries, report evidence, and raise unresolved decisions. An agent report is not owner approval. |

For governed changes, the flow is accepted spec → `status` lists what is to verify → a builder realizes, a fresh reviewer verifies and records every item → `status` lists nothing, and the person's session commits the work with the record ([concept 4.0, §5](concepts/v4-concept.md#5-what-build-does)). A real intent change updates its canonical owner before implementation. When intent is unchanged, a bug or stale artifact is repaired against that same spec; no artificial spec edit is needed. Exploration, distillation and other proposals remain drafts until the owner chooses to commit the revised spec. A commit is the specification basis, not proof of who reviewed or approved it.

Release 0.22.0 carries this workflow on Claude Code, covered by tests; no real agent run of its skills has been read yet ([concept 4.0, §14](concepts/v4-concept.md#14-what-is-not-yet-shown)). The [delivery plan](implementation-plan.md) and delivery PRs own implementation progress. Source tests and trials of the skills do not establish reliability or owner acceptance. Checks establish only their declared scope; none proves semantic correctness, human acceptance or success for every adopting repository. The agents run with the user's rights on the user's machine, so the work must not read untrusted input unless a container or a machine with limited rights is put around it ([concept 4.0, §7 and §12](concepts/v4-concept.md#7-how-sure-we-are)).

The target human role is specification and engineering judgment. Being a permanent agent scheduler, prompt supervisor, manual synchronizer or reader of every routine PR should not be necessary merely to keep declared intent consistent. Whether those activities can actually be reduced is a validation question. A project still defines which changes require human acceptance; passing technical checks do not grant authority or authenticate a reviewer.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in, which Markitect's product owner calls federalism ([CPT-001](concepts/register.md#cpt-001-federalism)). Each mechanism below replaces something that today's common practice leaves to chance. These are the method's mechanisms. [Architecture](architecture.md) and the current guides describe implemented behavior, and [concept 4.0, §7](concepts/v4-concept.md#7-how-sure-we-are) says how firm each one is. Actual checks remain bound to their source and scope.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **A list computed from the spec.** The spec and the registry make declared obligations and files explicit. `status` lists every Definition whose text, Kind, links, files or evidence changed since a reviewer verified it, and each listed Definition needs a verdict. | An agent deciding from memory which files might be relevant |
| **Roles with bounded rights.** The builder writes code and the registry pages, not the spec; the reviewer changes nothing and alone records; the architect writes the spec, not code; the auditor only reads and reports to the owner. A hook holds these rights for subagents ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). | One agent that writes, judges and records its own work |
| **Independent review with fresh context.** The reviewer starts with fresh context, takes its list from `status`, reads each Definition, the Definitions it links to and the code, runs the project's tests where they bear on an item, and treats what the builder tells it as claims to check. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** A Rule reaches every Package below the one that defines it. An `ok` means the Definition is implemented and the Rules above its Package hold for it, and a changed Rule is back on the list with every Definition that links to it. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Verdicts bound to what was read.** `mark` refuses a verdict when an input changed since `status` printed the item, the evidence must exist and is hashed, and the record is committed with the work. A later change puts what it touches back on the list, so `markitect status` in CI shows what is not verified. | Approving a pull request because its pipeline is green |

[The delegated method](#the-delegated-method) calls the implementing and assessing roles Executor and independent Verifier.

Two claims must be kept apart ([CPT-003](concepts/register.md#cpt-003-two-parts-of-the-promise)):
- **Nothing in the modeled scope is overlooked.** For declared relationships and file coverage this can be computed. Undeclared dependencies in code are a known gap ([ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact)), and so is prose that names a Definition without linking it ([the honest limit](#markitect-in-brief)).
- **Everything is realized correctly.** This can only be supported by evidence.

Tests belong to the realization. Agents write them, so they can be as wrong as the code. Reviewing tests against the statements they claim to check is an agreed direction ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)).

## The delegated method

The method has three responsibilities. They are responsibilities, not three Core primitives or three people. The delegated method has not yet been tested over a series of changes ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)); see [current basis and unproven benefit](#current-basis-and-unproven-benefit).

| Responsibility | What it does |
|---|---|
| Intent ownership | Accepts the concepts, rules, purpose and allowed freedom. Decides real intent changes, unresolved conflicts and exceptions. |
| Execution | Takes the applicable context, implements the affected work as a bounded candidate, and repairs failed representations without redefining success. |
| Independent verification | Checks the candidate against the accepted obligations with evidence suited to the target. Exposes violations, missing coverage and uncertainty, and returns pass, repair or an owner decision. |

In concept 4.0 the builder executes, while the fresh reviewer, `markitect check` and the project's tests assess the result ([where trust comes from](#where-trust-comes-from)). An Executor must not be the sole judge of its own success. Verification needs the accepted obligations and evidence about the candidate, including its integration, not only tests written by the Executor. Deterministic tools can add evidence. An agent label or a second model call alone does not make verification independent.

**Valid representations.** Code, documentation, agent instructions and other files represent the accepted intent; they do not hold their own copies of its rules. A representation is correct when it is one of the valid representations of that intent. Different structures or names can be equally valid where the policy allows them, so the allowed freedom is explicit. Byte equality applies only to targets with an exact deterministic contract. Keep representations that are already valid instead of rewriting them to an agent's taste.

**Current target:** [concept 4.0](concepts/v4-concept.md) ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)), which Main implements. Where this page still speaks of Managers, makers, checkers and parents, mainly in [Origin and rationale](#origin-and-rationale), it describes [concept 2.0](concepts/v2-concept.md), a record; concept 4.0 governs where they differ. V1's design, its Run host and its demonstrations stay in the [V1 concept](concepts/v1-concept.md) and in Git history.

**The change cycle** ([concept 4.0, §5](concepts/v4-concept.md#5-what-build-does)):

1. The owner commits a spec change and runs `/markitect-build`. Their session starts the builder in a worktree of its own, from the last commit of their branch.
2. The builder runs `markitect status`: every Definition with a changed input or no record yet, those recorded `fail` or `spec-problem`, those gone from the spec, and the errors in the realization. A spec with errors gives no list.
3. The builder makes the project match the spec for what the list holds and keeps the registry current. Where the spec contradicts itself or leaves open behaviour at the project's boundary, it builds nothing for that Definition and tells the reviewer.
4. The builder starts a fresh reviewer, which records a verdict for each item with `markitect mark`, bound to the fingerprint it read. The builder repairs what failed and has it verified again, until `status` lists nothing or only `spec-problem`s, or three rounds in a row have not made the list shorter.
5. The owner's session commits the work with `.markitect/review.sum` on the worktree's branch and, when `status` is empty there, merges it into the owner's branch; otherwise it asks the owner. It pushes nothing. Human acceptance stays the project's own policy.

`/markitect-review` runs reviewers alone, one per Package on a long list; they record verdicts and change no file.

**Only what a change reaches.** A verdict holds until one of its inputs changes, so a change puts back on the list only the Definitions it reaches, and `/markitect-review --all` verifies everything again. The record makes every listed Definition need a verdict; whether a verdict is right is a matter of review and evidence, not of mechanical proof.

**Rules, process and evidence.** A precise rule can have a deterministic check. A broad rule, such as consistent error handling, needs a declared surface, a concrete interpretation and independent evidence; a few examples do not prove it everywhere. Check the agent's process as well as its files: a compliant final file does not prove that the agent started from the model, used its context or handled intent first. A planted defect shows only that a check catches that defect. How often the method works needs repeated real tasks ([measurement](guides/measurement.md)).

**Adoption.** For an existing project, inferred intent is a proposal ([product definition](#product-definition)). Keep valid bytes, record unknown ownership and missing evidence, and never change the model just to bless accidental drift.

## What executable governance means

One canonical owner exists per modeled engineering fact or representation decision. Rationale and guidance may be prose inside the project model. Unadopted narrative, vendor material and specialist observations remain outside accepted intent until explicitly incorporated. The project tree makes responsibility, artifact relationships and decision scope visible; it does not infer source-code semantics or turn every repository file into a model fact.

The new structural Core does not contain an engineering assertion DSL. Checks, policies and assurance live above Core; the historical finite Domain/policy contract belongs to the earlier Project/Domain line, which is not developed further ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)). Project-owned checks and target tools can provide specialized implementation evidence. Rule and Workflow prose remains instruction and rationale where no executable assertion exists; writing it in a Definition does not enforce the process or prove that an agent followed it.

Context helps agents find applicable intent; impact helps reviewers understand declared consequences; versioned packages make architecture evolution explicit; narrow exceptions record deliberate deviations; reconciliation can align owned representations. None proves that all relevant intent was modeled, that an agent obeyed it, or that arbitrary application code is correct.

Continuous autonomous agents are the desired consumer operating model. They may call bounded Markitect operations from their own runtime, orchestration or CI. An Executor produces a candidate representation using appropriate tools, including deterministic renderers when useful: permitted implementation freedom must be explicit and project-owned checks must state what they establish. This does not require a continuously running Core, hidden policy activation or a background mutation controller. The [delivery plan](implementation-plan.md) defines the current sequence. Release 0.22.0 has the skills for design, build, review, cleanup, bug hunts, audits and routines ([project workflow](guides/project-workflow.md#7-build-and-review)); `security`, `improve` and a CI check with `markitect status` come in step 4 of [concept 4.0's order of work](concepts/v4-concept.md#13-order-of-work). V1's Runs and the earlier Pass/shared-tree implementation are retired; their design is in the [V1 concept](concepts/v1-concept.md) and their source in Git history.

## Desired operating model

Humans should eventually review decisions at the level of architecture, ambiguous requirements, unresolved policy conflicts, deliberate exceptions and exceptional risk. Agents should handle ordinary implementation details within the project's declared freedom.

The organization must make a justified no-change result, a useful objection and timely escalation count as good work. Different stakeholders and perspectives need distinct questions and the context to answer them; adding Agent titles or rewarding completion alone supplies no such control. These are motivations for the existing [team direction](concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes), not a requirement to add more roles, separate hierarchies, scores or incentive machinery to the harness.

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
| Apply, Verify and Cleanup | Realize intended changes, examine conformity and improve the repository within the applicable model. | Offers repeatable work against a known basis. Concept 4.0 makes them skills: build, review, cleanup and bug hunt, which routines can run on a schedule ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)); none promises unattended acceptance. |
| Government | Delegate selected model-maintenance decisions, preserve human reservations and report relevant matters. | Could reduce routine model work. Correct procedure does not prove that the selected intent is right; the model may grow harder to maintain. |

The goal is to use explicit structure and ordinary deterministic software where they fit, then give agents bounded interpretation and realization work. Deterministic and semantic checks have different jobs. The compiler analogy is borrowed from programming languages; it does not make a natural-language specification executable or complete merely by attaching types.

### Foundations restated on 17 October 2026

The marketing discussion also records the owner's [conceptual influences](concepts/sources/conceptual-influences-20261017.md) and [product-goal clarification](concepts/sources/product-goals-clarification-20261017.md). These supply additional provenance for the foundations below: lower cost is not the product goal, and routine human implementation review should become unnecessary through trustworthy realization.

The owner named the following influences and reaffirmed the goal: a clean, uniform and clear repository, without drift, forgotten rules, inconsistencies or contradictions; model first; trustworthy AI results through system and structure; ultimately modeling followed by AI realization that needs no routine human code review. The [source](concepts/sources/foundations-and-trust-20261017.md#owner-statement) preserves the wording. The table connects those influences to existing choices; it introduces no new roles, language semantics or delivery scope.

| Foundation | Intended contribution and current design connection |
|---|---|
| Four eyes and different perspectives | A fresh reviewer asks whether the work satisfies each listed Definition; the auditor asks whether the spec itself holds together. Distinct questions matter more than Agent count. [CPT-001](concepts/register.md#cpt-001-federalism), [ENH-006](concepts/register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes). |
| Responsibility, delegation and escalation | Each role has bounded rights, which the hook holds: the builder builds, the reviewer judges and records, the architect drafts the spec on delegation, and the auditor reports to the owner. A `spec-problem` escalates a question to the owner. Concept 2.0 gave each Package a Manager that integrated its children, and V1 had every Manager take part in a fixed process; the Manager tree went with concept 3.0. [DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), [DEC-091](concepts/register.md#dec-091-concept-30-is-the-target), [DEC-025](concepts/register.md#dec-025-package-managers-organize-realization). |
| Divide and conquer; abstraction and need to know | Decompose meaningfully into Packages and give each agent sufficient local detail: a reviewer reads each listed Definition with what it links to, and `/markitect-review` gives a long list one reviewer per Package. Summaries and targeted reads connect breadth and depth; bounded context must not hide a needed contract. [DEC-028](concepts/register.md#dec-028-context-follows-need-to-know-and-starts-simple), [OQ-006](concepts/register.md#open-questions). |
| Model and projections; one source of truth | The repository's code, tests, documentation and configuration realize one intended world. They may have different shapes while expressing the same intent. A generated readable view is a projection with an exact rendering contract; Agent-written code needs semantic assessment. [DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [CPT-003](concepts/register.md#cpt-003-two-parts-of-the-promise). |
| Planning, system and structure; trust the process | Deterministic compilation, the record of what was verified and the hook handle explicit structure. AI supplies bounded interpretation and realization judgment. Trust depends on useful findings, corrections and faithful results across the whole workflow. [DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache). |
| Stakeholders and incentives | Different concerns should be represented by distinct questions and access to relevant evidence. Grounded objections, justified no-change and escalation are useful outcomes. Incentive design is an influence, not an implemented scoring or reward system. Each piece of work pairs a builder with a fresh reviewer. [Organization and trust](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/reflections/organization-and-trust-20261016.md#limits-of-the-analogy-and-future-autonomy). |
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

The [model language guide](guides/model-language.md) and [concept 4.0](concepts/v4-concept.md) own the language and the realization rules. A project still chooses its model granularity, intended relationships, implementation freedom and acceptance policy. The owner's restatement clarifies the motivation and goal; it adds no runtime mechanism or authority for autonomous model changes, and it does not establish product superiority over conventional agentic development.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, Rules, Skills, Workflows and agent instructions. Its direction is to turn informal engineering knowledge into structured, versioned architecture and policy, then into an executable engineering system. The name records that origin. Concept 2.0 makes the model Markdown again: each Definition is a Markdown file whose short front matter names its Kind ([DEC-080](concepts/register.md#dec-080-the-model-is-typed-documentation)). The origin explains the name; it does not limit the product to agent instructions.

## Common misreadings

The readings below came up repeatedly, most recently in the [idea review of 10 October 2026](concepts/sources/idea-review-20261010.md). Check an assessment against this table before treating any of them as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the name comes from. Markitect is a programming language for documentation of the whole project, with delegated realization. Agent instructions are one kind of realization among many. |
| Markitect is a new, unproven formalism. | It builds on proven practice: requirements traceability with suspect links, architecture metrics and Go's conventions. What is new is that the traceability is plain, typed text checked deterministically, that agents realize the spec and gather the evidence, and that the record works like Go's test cache ([Markitect in brief](#markitect-in-brief), [VIS-002](concepts/register.md#vis-002-a-programming-language-for-documentation)). |
| The deterministic parts are the product, and agent work is scope creep. | Delegated realization is the purpose, and the tool supports it. The tool computes and checks, and a driver, by default the person's own session, starts a builder and a fresh reviewer ([DEC-001](concepts/register.md#dec-001-delegated-realization-is-central-to-the-product), [DEC-096](concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)). |
| Markitect orchestrates or steers the agents. | It supports agent work and does not steer it. It checks and analyses the spec, says what is to verify and records verdicts. Who starts which agent, when and how often belongs to the driver: the skills, the person's own session or the person. There is no Run, no `gate` and no limit of rounds in the tool ([DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). |
| Markitect needs containers, a token, a background session or a Git host, and it commits and lands work. | It runs natively, and the person's own Claude Code session starts the agents on its account login. It uses no Git: it makes no repository, commits nothing and keeps no branches. The session commits the work with the record and merges it; nothing is pushed unless a routine the person set up lands by a push ([DEC-097](concepts/register.md#dec-097-markitect-uses-no-git), [DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). |
| Markitect works around Claude Code with machinery of its own. | It works with Claude Code and Anthropic's own features: skills, subagents, hooks, worktrees, a plugin and scheduled tasks. Earlier versions built Manager trees, containers and Runs; each version since has dropped some of that machinery ([concept 4.0, §15](concepts/v4-concept.md#15-what-this-overturns)). |
| An empty `markitect status` proves the project correct. | It means that the spec compiles and that every Definition was verified at its current inputs. That is complete for the declared links only: prose that names a Definition without linking it, and the code's own imports, are not followed. A verdict is evidence a reviewer gave, not a proof ([concept 4.0, §7](concepts/v4-concept.md#7-how-sure-we-are)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed concept, and the trust bar is today's practice ([DEC-002](concepts/register.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept)). |
| The Manager hierarchy is the Government experiment. | Concept 2.0's Manager hierarchy belonged to the core axis, Spec → Repository, and went with concept 3.0 ([DEC-091](concepts/register.md#dec-091-concept-30-is-the-target)). Government belongs to a later axis, in which management levels decide spec changes within mandates. Results of the former Government experiment arm are not results about the Manager hierarchy ([DEC-004](concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)). |
| "More economical models" means weak models. | It means capable mid-tier models such as Luna or Sonnet, and Haiku only if it proves sufficient ([DEC-007](concepts/register.md#dec-007-more-economical-models-means-capable-mid-tier-models)). |
| Earlier comparisons showed that conventional agentic coding wins. | Environment, runtime and harness failures dominated the October 2026 comparison attempts, and none of them exercised the delegated method over a series of changes. They allow no conclusion about the concept. Earlier matched comparisons tested an earlier product; their findings, including the cost of model upkeep, stand ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). |
| The model has to mirror the code, with one definition per file. | Granularity is the author's choice. A file may realize several definitions, and every file must be covered or explicitly ignored ([DEC-005](concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility), [DEC-006](concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). |
| Green agent-written tests prove the realization. | Tests are realizations too. Agents write them, so they can be wrong. Reviewing them against the statements they claim to verify is an agreed direction ([CPT-002](concepts/register.md#cpt-002-tests-are-realizations-too)). |
| The goal of removing routine human code review removes the four-eyes principle. | A fresh reviewer that verifies every Definition a change reaches, and a record that only it may write and that a later change voids, are the means to earn that trust ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)). People retain model decisions, escalation and their project's acceptance policy ([PRM-004](concepts/register.md#promises)). |

## Feature decision filter

Ask: **Does this improve fidelity to accepted intent and reliable autonomous reconciliation inside explicit boundaries, with enough independent evidence to reduce routine human supervision?**

A proposal should identify the human activity it aims to remove, its canonical owner, the agent's remaining freedom, the exact evidence/check boundary and the new authoring or maintenance cost. Deterministic checks, task context, impact, policy evolution, exceptions, discovery and reconciliation can support this goal, but each needs a concrete workflow and evaluation.

Reconsider proposals that introduce another manually synchronized truth owner, hide architectural decisions in inference, specialize Core for one provider, or add machinery without evidence that coordination gets easier. More generated text, adapters, language operators or agent throughput alone do not demonstrate progress. This filter does not override the constitution's bar for new semantics or authorize a new feature wave.

For concept 4.0, the aim is one simple version that works reliably on Sonnet, no longer a proof that one version beats another ([§1](concepts/v4-concept.md#1-the-aim)). Step 3 of its [order of work](concepts/v4-concept.md#13-order-of-work) asks for two clean passes in a row over the 5× project's eight change steps: every step ends with nothing left to verify without the owner, all hidden cases pass and no refusal was overridden. A successful run is bounded feasibility evidence. Reduced routine human review and dependable use of cheaper models need repeated quality and attention evidence under [Measurement](guides/measurement.md#human-attention-and-delegated-work). Keep the first strategy small enough to learn this before adding roles, parallelism or reuse mechanisms.

## Current basis and unproven benefit

| Category | What is established or intended |
|---|---|
| Product direction | A single spec-first user workflow centered on the `.markitect/` folder and its record of what was verified. The [roadmap](implementation-plan.md) records exact source and release status. |
| Current source | Concept 4.0 is the target, and Main carries its build ([delivery plan](implementation-plan.md)): the Markdown spec language with `check`, `vet`, `list`, `doc` and `tidy`, and `status` and `mark` over the record of what was verified, with the skills `markitect-design`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and `/markitect-schedule` and four agents as the default driver, released as 0.22.0 on 7 November 2026. Concept 2.0's Run engine and concept 3.0's rebuild are in Git history and on the branch `archive/main-concept-3-20261106` ([archive](archive.md#archive-points)). V1's design is in the [V1 concept](concepts/v1-concept.md), its source in Git history. The preceding source generation remains in [Git](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). |
| Published compatibility | The earlier Project/Domain CLI and Projection alpha remain in the pinned pre-reset commits listed in the [archive index](archive.md); the former Git tags no longer exist. Their release pages and binary downloads were retired by the owner on 17 October 2026. The contracts and recorded evidence are historical; there is no second active product UX. See the [production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md#v0141-published-release). |
| Acceptance boundary | A committed model revision is the accepted specification used for that revision. Commit/provenance values bind data but do not authenticate a human decision. Passing checks, generated docs and agent reports are evidence with declared scope, not approval. |
| Aspirational operation | More parallel agents, less routine supervision, architecture-level escalation and sustainable governance at greater scale. No universal autonomous acceptance is claimed. The agents run with the user's rights on the user's machine, so untrusted input needs a container or a machine with limited rights around them ([concept 4.0, §7](concepts/v4-concept.md#7-how-sure-we-are)). |
| Unproven benefit | Reduced human intervention, lower total maintenance cost, fewer missed obligations or defects, better sustained autonomy, token savings and market demand remain unmeasured. Historical comparisons are preserved, including their negative or inconclusive findings. |

The [real-code adoption pilot](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/real-project-adoption-pilot.md), [AGENTS.md comparison](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/agents-md-vs-markitect.md) and [parallel-wave consumer inventory](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/parallel-wave-konfyra.md) are evidence, not marketing demonstrations. The comparison remains inconclusive: additional model/projection maintenance was observed, and no independent truth owner or concrete human review step was removed. Missing context and broad impact remain meaningful findings. The parallel wave proved bounded subsystem independence, not reduced human coordination cost.

These records evaluated earlier product forms, not the delegated Manager method over a series of changes. The October 2026 comparison attempts of the delegated method were dominated by environment, runtime and harness failures. They allow no conclusion about the concept, positive or negative ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). Markitect's product owner has endorsed evaluating the method separately from any single execution runtime; the plan is pending ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)).

The central evaluation question is: **Does ontology-driven reconciliation preserve intent and project quality more reliably than conventional agentic work, and thereby support less routine human supervision?** [Measurement](guides/measurement.md#human-attention-and-delegated-work) defines future metrics and falsification. A materially simpler owner document plus architecture tests remains a valid alternative. Markitect's product owner phrases the same question in [DEC-009](concepts/register.md#dec-009-the-central-evaluation-question): when people develop against a model, can the code reliably reflect that model? Can the repository stay contradiction-free, consistent, uniform and clean? Does agentic coding become meaningfully more trustworthy? Can people concentrate on deciding and modeling?

## Decisions Markitect deliberately leaves to people

Markitect does not decide which architecture is correct, which business goal matters, whether a policy is desirable, whether evidence is representative or which exceptional risk to accept. It does not understand arbitrary source-code semantics in Core, guarantee correctness or absence of drift, replace human product/architecture judgment or guarantee complete agent autonomy. Humans retain those decisions and may explicitly delegate routine work; machine evidence states only what its fixed inputs and declared checks establish.


## Research refinement

The [ontology, projection and assurance thesis](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/research/README.md) is owner-supplied research orientation. Its distinction between semantic authority, implementation freedom, independent evidence and integration obligations guides experiments. The [alignment assessment](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/projection-assurance-direction.md) identifies what this source slice supports and what remains research. It neither expands Core by naming concepts nor changes measured evidence into a benefit claim.
