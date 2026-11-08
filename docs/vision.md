# Product vision

This page says what Markitect is, why it exists, where its trust comes from and what it leaves to people. [Concept 4.0](concepts/v4-concept.md) describes the product in detail, and the [register](concepts/register.md) lists the decisions in force. The vision authorizes no feature and claims no measured outcome.

## Markitect in brief

Read this section first, then [Founding intent](#founding-intent).

**What it is: a programming language for documentation.**
- People maintain one authoritative place per project: the spec, which code and older documents call the model. It is typed prose: Markdown Definitions, each of a Kind, organized in Packages, with links that work like imports. It states the project's concepts, rules, boundaries, interfaces, architecture and way of working, as far as good documentation of the project would.
- Code, tests, configuration and documentation are realizations of the spec. Where they differ from it, the spec is right.
- A deterministic tool, the `markitect` program, checks and analyses the spec as a compiler and a linter check code: `check` compiles it and accounts for every file of the project, `vet` measures its design, and `status` and `mark` keep a record of what was verified. Agents realize the spec and verify the result.

This is the owner's direction of 7 and 8 November 2026 ([DEC-101](concepts/register.md#dec-101-markitect-is-a-programming-language-for-documentation)). The owner expects coding to move from languages such as Go, C# or Python towards documentation and meaning. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

**What it builds on: proven practice.**
- **Requirements traceability.** Tools such as IBM DOORS and Polarion keep a traceability matrix that links requirements to what realizes them, in both directions, and mark a link suspect when either end changes; DO-178C, ISO 26262 and IEC 62304 require such traceability. In Markitect the registry is the matrix: one page per Package links every file to the Definitions it realizes, and `markitect doc` shows a Definition's links both ways. `markitect status` lists the suspect items, every Definition whose text, Kind, links, files or evidence changed since a reviewer last verified it, and `markitect mark` records a verdict on one; an `ok` with its evidence clears it.
- **Architecture metrics.** The spec's design is judged with the principles that keep code changeable: Clean Architecture's Dependency Rule, domain-driven design's bounded contexts, cohesion and coupling, and Robert C. Martin's stability measures ([what makes a good spec](guides/good-spec.md)). `check` refuses a link from a Package to a Definition inside its child, and a cycle between Packages; `vet` warns about coupling, size, hubs and unstable dependencies, and notes unlinked mentions, long Definitions, placement and change hotspots in Adam Tornhill's sense, counted from the record since the tool reads no Git ([DEC-102](concepts/register.md#dec-102-what-vet-measures)).
- **Go's conventions.** The command line follows Go's tools: `vet`, `list`, `doc` and `tidy` do what their Go namesakes do, diagnostics read `file:line: code: message`, and exit codes follow Go's ([DEC-096](concepts/register.md#dec-096-the-tool-computes-and-checks-a-driver-acts)).

**What is new.**
- The traceability is plain text in the repository, beside the code: Markdown that people and agents read directly, typed by Kinds and checked deterministically, with no database and no separate tool to keep in step.
- AI does not take everything off the developer's hands, but it makes affordable what ordinary software projects dropped because of the manual effort and cost, such as keeping a traceability matrix. Agents do the realization and gather the evidence. A builder makes the project match the spec, and a fresh reviewer verifies each listed Definition and records its verdict with the evidence it read.
- The record of what was verified, `.markitect/review.sum`, works like Go's test cache: a verdict holds until one of its inputs changes, so a change puts back on the list only what it reaches ([DEC-099](concepts/register.md#dec-099-a-record-of-what-was-verified)).

**How a change goes** ([concept 4.0, §5](concepts/v4-concept.md#5-what-build-does)).
- A person changes the spec, usually with a coding agent through the `/markitect-design` skill, which asks first when something is open. The person reads the Markdown diff and commits it.
- `/markitect-build` has the person's own Claude Code session start a builder agent in a worktree of its own, so the person can go on designing. The builder makes the project match the spec for what `markitect status` lists and starts a fresh reviewer agent, which records a verdict for each item with `markitect mark`, bound to the state it read. A `fail` stays on the list until the builder has repaired it and it is verified again.
- When `status` lists nothing, the project compiles and every Definition is verified at its current inputs. The session commits the work with the record and merges it into the person's branch.
- `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt` and `/markitect-audit` verify, refactor, hunt for bugs and read the spec itself for contradictions and gaps, and `/markitect-schedule` sets them up as routines. Four roles, the builder, the reviewer, the architect and the auditor, have their rights held by a hook ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)).
- On a long list, a shipped workflow runs one builder and its reviewer per Package in parallel; `harness.yaml` sets the agents' model and effort and each skill's strategy ([DEC-103](concepts/register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)).
- Markitect itself starts no agent, steers no work and uses no Git ([DEC-096](concepts/register.md#dec-096-the-tool-computes-and-checks-a-driver-acts), [DEC-097](concepts/register.md#dec-097-markitect-uses-no-git)).
- A repair that leaves intent unchanged is realized against the same spec, without an invented spec edit.

**The approach: lighter, native and simpler.** Markitect works with Claude Code and Anthropic's own features rather than against them: skills, subagents, hooks, worktrees, a plugin and Claude Code's own scheduling. Earlier versions built their own machinery around the agents: a Manager agent per Package, containers, a Run that held the work and landed it, and a `gate` command that printed each next step. Each version since has dropped some of it. In the owner's experience the simpler version is the more successful and the more robust one, and that simplicity is earned through insight ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

**The honest limit.** "Complete" means complete for the declared links. `status` follows a Definition's own text, its Kind, the Definitions it links to, its registry rows and its evidence, one step deep; it does not read the code's imports ([ENH-001](concepts/register.md#enh-001-observed-code-dependencies-widen-impact)). Prose that names a concept without linking it is a gap: a change to that concept does not put the naming Definition back on the list. `vet` notes such names as `vet.unlinked_mention`, and `/markitect-audit` has the auditor tell real missing links from ordinary words; names a text check does not match, only a careful reader finds. An empty `status` says that every Definition was verified at its current inputs, not that every verdict is right.

**Why: three problems** ([DEC-101](concepts/register.md#dec-101-markitect-is-a-programming-language-for-documentation)).
- **Nobody keeps track of what was decided.** In a long session or a large project, nobody can remember or keep current what was decided when; coding agents lose track of relevant files and rules as projects grow, and explicit consistency runs miss things and scale badly. The spec is the one living statement of what holds.
- **Nobody reads all the documentation.** AI produces more documentation than anyone reads. The spec is small, structured and navigable: Definitions in Packages, with links, and `markitect list` and `doc` to find one's way.
- **Nobody knows whether the realization matches the idea.** Common practice nevertheless trusts the agents: an agent implements a work item, another AI reviews the pull request, and the required human approval is often given almost blind. `status` and `review.sum` say per Definition what a fresh reviewer found, with its evidence.

So the knowledge of what belongs to what must not depend on an agent's memory, and people cannot read every diff without becoming the bottleneck.

**Expected outcomes, to be validated** ([promises](concepts/register.md#promises)).
- Nothing in the spec's scope is silently forgotten.
- Rules are applied where they matter.
- The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable.
- Capable mid-tier models can do most of the delegated work.
- People concentrate on deciding and modeling while the code reliably reflects the spec.

The intended end state is **spec → AI realizes and independently reviews it → people rely on the result without routinely reviewing code diffs themselves**. A fresh reviewer stays part of the process, and human attention goes to the spec, the architecture and the `spec-problem`s. This is a target to earn through evidence, not a claim that the current version has achieved it, and a project's own acceptance policy still applies.

**How to judge it.**
- Judge Markitect's trust against today's practice, not against a compiler's guarantees ([DEC-002](concepts/register.md#dec-002-trust-is-measured-against-todays-practice)).
- "Compiler" names a borrowed idea. The spec's structure compiles deterministically; realization by agents is not deterministic ([DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Markitect's claim is that a typed spec, a list of what to verify computed from it rather than from an agent's memory, a verdict with evidence for every listed Definition, roles whose rights a hook holds and a reviewer with fresh context make delegated work more trustworthy than unstructured agentic work. [Where trust comes from](#where-trust-comes-from) lists these mechanisms.
- Delegated realization is central to the product. It is not an optional layer on top of a validator ([DEC-001](concepts/register.md#dec-001-delegated-realization-is-the-purpose)).
- The current aim is one simple version that works reliably on Sonnet ([DEC-009](concepts/register.md#dec-009-how-markitect-is-judged)). What is shown and what is not is in [concept 4.0, §13](concepts/v4-concept.md#13-what-is-not-yet-shown).

## Founding intent

The owner's [founding intent](concepts/sources/founding-intent-20261026.md) explains why Markitect exists and what the spec is meant to be, and the owner's rationale of 8 November 2026 says why it is typed and what it is for ([DEC-101](concepts/register.md#dec-101-markitect-is-a-programming-language-for-documentation)). The earlier [origin account](concepts/sources/owner-origin-20261015.md) tells the story behind it; [Origin](#origin) summarizes it.

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. Every consistency run took hours and found more inconsistencies. The owner lost track of which rules existed, what was defined and what had entered the project silently.

**Code is a description.** Code has always been a very precise description for an extremely limited but diligent executor. With AI the executor is intelligent, but it still has to be told what is meant and what holds; otherwise it guesses, and may do what nobody wanted. So documentation, prose and prompts become the programming language. Markitect makes their authors express themselves cleanly: define what they mean and what holds, and connect those statements meaningfully.

**What AI changes.** AI does not take everything off the developer's hands. It makes affordable the approaches that ordinary software projects dropped because of the manual effort and cost, such as the traceability matrix: agents keep the registry current and gather the evidence, work that was too costly by hand.

**The answer is a typed language in prose.** Contradictions in a description surface only when someone changes the place where they sit, the same observation as with weakly typed programming languages. So Markitect borrows typing and a compiler as helpers: in a strongly typed language, a compiler and explicit dependencies report issues at compile time instead of leaving them to be found later. The spec applies that to project intent:

| Typed language | Markitect spec |
| --- | --- |
| type, class, function signature | Kind |
| declaration | Definition |
| function body | the Definition's prose: binding text that agents realize |
| import | link: `[[members/member]]` names its target by path |
| package, module | Package |
| standard library, modules | Libraries, named in the manifest and pinned by digest in the lock, as `go.mod` and `go.sum` do |
| compiler, type checker | `markitect check`: structure only, never meaning |
| `go vet`, architecture metrics | `markitect vet` |
| test cache | `.markitect/review.sum`, the record of what was verified |

The language has no typed fields, so exact values such as error codes are written in prose. It has no import statement either: each link is itself a declared dependency, like an import in Go, and a sideways link makes an edge between two Packages ([model language](guides/model-language.md#links-scope-and-visibility)). A Kind is itself a Definition of Kind `Kind`, a Definition is one Markdown file with front matter, and a Package is a folder.

- The core stays clean. Projects define their own Kinds and so form their own language; Libraries supply common Kinds, oriented on what Go or C# ship, so that no project reinvents them. `std`, `rest` and `cli` ship in the binary.
- The compiler checks structural correctness from the core alone. Meaning is judged by agents: the reviewer judges the code against the spec, and the auditor judges the spec itself.
- The spec carries the structure the owner first gave the agents, after Conway's law, Clean Architecture, clean code and modularity: Packages are modules, and links go up the tree or sideways towards stability, never down. [What makes a good spec](guides/good-spec.md) reads this with domain-driven design's bounded contexts and Martin's stability measures.
- People decide intent. AI agents write the spec in conversation through the design skill, which asks first when something is open; the owner reads the Markdown diff and accepts it. The same skill draws a first spec from existing code.

**The aim of the language.** Meet the preconditions that unlock the established knowledge of software architecture for this new language: Definitions with a type, dependencies that are declared, and a structure a tool can check and measure.

**Consequences.**
- Judge a spec as you would judge code: dependencies, coupling and stability, clean architecture and size. A Definition of 500 words is prose compensating for missing Definitions. [What makes a good spec](guides/good-spec.md) states the rules and their measures.
- A spec is as explicit as good documentation of the project ([DEC-079](concepts/register.md#dec-079-the-spec-is-as-explicit-as-good-documentation)). Whether the standard Libraries, the design skill and `vet` lead to good specs in practice is still to be shown. Results of Markitect are only as good as the spec they use.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in ([CPT-001](concepts/register.md#cpt-001-trust-comes-from-structure)). Each mechanism below replaces something that today's practice leaves to chance; [concept 4.0, §7](concepts/v4-concept.md#7-how-sure-we-are) says how firm each one is.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **A list computed from the spec.** `status` lists every Definition whose text, Kind, links, files or evidence changed since a reviewer verified it, and each listed Definition needs a verdict. | An agent deciding from memory which files might be relevant |
| **Roles with bounded rights.** The builder writes code and the registry pages, not the spec; the reviewer changes nothing and alone records; the architect writes the spec, not code; the auditor only reads and reports. A hook holds these rights for subagents. | One agent that writes, judges and records its own work |
| **Independent review with fresh context.** The reviewer takes its list from `status`, reads each Definition, the Definitions it links to and the code, runs the project's tests where they bear on an item, and treats what the builder tells it as claims to check. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** A Rule reaches every Package below the one that defines it, an `ok` means the Rules above hold for that Definition, and a changed Rule is back on the list with every Definition that links to it. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Verdicts bound to what was read.** `mark` refuses a verdict when an input changed since `status` printed the item, the evidence must exist and is hashed, and the record is committed with the work. A later change puts what it touches back on the list, so `markitect status` in CI shows what is not verified. | Approving a pull request because its pipeline is green |

Two claims stay apart ([DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)): *nothing in the spec's scope is overlooked* can be computed for declared links and file accounting; *everything is realized correctly* can only be supported by evidence. Tests belong to the realization: agents write them, so they can be as wrong as the code ([CPT-002](concepts/register.md#cpt-002-tests-are-the-projects-and-they-are-realizations-too)).

## Who does what

| Role | Responsibility |
|---|---|
| People | Decide what should exist and why; choose concepts, architecture, boundaries, rules and processes; define the agents' freedom and the project's acceptance policy; settle every `spec-problem`. |
| Markitect | Compile the spec, account for every file, say which Definitions are to verify because an input changed, record each verdict at the state the reviewer read, and keep agents from recording their own work or changing the spec. It starts no agent, steers no work and uses no Git. |
| Agents | Draft spec changes for the owner to accept, realize the spec, review the result and record verdicts with evidence, and raise what only the owner can settle. An agent's report is not the owner's approval. |

A real change of intent changes the spec first. When intent is unchanged, a bug or a stale file is repaired against the same spec; no artificial spec edit is needed. Drafts and proposals stay drafts until the owner commits the revised spec. A commit is the basis of the spec, not proof of who reviewed or approved it.

The target human role is specification and engineering judgment, not being a permanent agent scheduler, prompt supervisor or reader of every routine diff. Whether that work can actually be reduced is what Markitect has to show.

## Three starting situations

| Situation | Typical use | With Markitect |
|---|---|---|
| 1. A spec, no artifacts | Start a new project by writing its spec first | `/markitect-design` writes the spec with the owner, and `/markitect-build` builds it all, since nothing is verified yet |
| 2. Spec and artifacts | Ongoing development and maintenance | `/markitect-build` realizes what `status` lists |
| 3. Artifacts, no spec | Adopt Markitect for an existing project | `/markitect-design` draws a first spec from the code with the owner, and may hand a large draft to the architect |

`/markitect-design` sees which situation a folder is in. For a spec drawn from code, the binding part is the contract callers rely on, and observed behaviour, proposed intent and open questions stay apart until the owner accepts the result. Afterwards `/markitect-review --all` records where the code deviates, and `/markitect-build` repairs it. A confirmed defect stays a defect against the accepted spec; rewriting intent to get an empty `status` would defeat adoption. The situations can coexist in different areas of one project.

<a id="origin-and-rationale"></a>

## Origin

The owner was developing a project with explicit Core, Module and Product boundaries: Core depending on nothing, Modules only on Core, Products combining Modules. The owner also wanted to specify the working method, review policies, where each use case is documented, and which skills belong to a module. Code, documentation and the agents' skills repeatedly contradicted one another; changes missed rules, and the owner lost the overview ([origin account](concepts/sources/owner-origin-20261015.md)).

| Step | Problem | Resulting idea |
|---|---|---|
| 1 | Architecture rules and working obligations must survive across code, documentation and agent skills | Make the project's intended world explicit, rules included |
| 2 | Several editable descriptions of the same rule drift, and nobody knows which one holds | Keep one authoritative statement of each fact, and realize everything else from it |
| 3 | Missing relationships and inconsistent structure surface late | Borrow typing and a compiler to reject structural errors before the work |
| 4 | One agent cannot keep the whole project's rules and files in mind | Compute what a change reaches from declared links, and give each Definition a fresh reviewer |
| 5 | Reading every change and repeating global consolidation makes the human the bottleneck | Maintain the spec; let agents build, review, clean up and hunt bugs against it |
| 6 | Interpreting new work and maintaining the spec still costs attention | Later, agents may maintain the spec within mandates ([IDEA-002](concepts/register.md#ideas)) |

Earlier versions answered step 4 with a tree of Manager agents, one per Package, that integrated each other's work. Measured runs showed the tree costing five to eight times one strong agent and adding failures, while a computed list and a fresh checker caught the same planted problems without it, though never in the same launches. A project too big for one agent, the tree's actual reason, was not tested. The tree went on 4 November 2026.

## Assumptions

Why the owner expects the method to help, and what would show that it does not. The register's [assumptions](concepts/register.md#assumptions) and [promises](concepts/register.md#promises) keep their status.

| Assumption | What would challenge it |
|---|---|
| One authoritative spec with maintained links reduces the mistakes of keeping things in step | Contradictions still pile up, or keeping the spec and the registry costs as much as the consolidation it replaces |
| Typing catches useful structural errors before agents work | Most consequential failures pass `check` and still need the same late investigation |
| A list computed from declared links keeps continuing obligations in view | What a change really affects falls outside the list, or the list grows to nearly the whole spec on every change |
| A fresh reviewer finds mistakes the builder misses | Reviewers keep accepting the same omissions or wrong readings, and extra calls add no useful evidence |
| A typed spec lets capable mid-tier models, such as Sonnet, do most of the work | Rework and repeated review eat the savings, or only the most expensive models get it right |
| The record and its evidence remove the need to read every routine diff | The owner still needs to review every diff, cannot tell what is unverified, or trades that work for equally costly spec supervision |

## Where it could go

The longer-term picture is the owner's: a cockpit with a personal AI, and a project organization in which the owner models, sets rules and develops ideas while agents keep the project realized and tidy ([VIS-001](concepts/register.md#ideas)). Continuous work already runs outside the tool, as routines, and autonomy grows in steps, each an owner decision: first repairs run on their own and spec proposals wait for the owner; later the owner may grant a mandate within which agents accept spec changes ([DEC-084](concepts/register.md#dec-084-continuous-work-runs-outside-the-tool-and-autonomy-grows-in-steps)). These are recorded, not planned, until a usable, stable version exists.

## Common misreadings

Check an assessment against this table before treating one of these as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the name comes from. Markitect is a programming language for documentation of the whole project, with delegated realization. Agent instructions are one kind of realization among many. |
| Markitect is a new, unproven formalism. | It builds on proven practice: requirements traceability with suspect links, architecture metrics and Go's conventions. What is new is that the traceability is plain, typed text checked deterministically, that agents realize the spec and gather the evidence, and that the record works like Go's test cache ([DEC-101](concepts/register.md#dec-101-markitect-is-a-programming-language-for-documentation)). |
| The deterministic parts are the product, and agent work is scope creep. | Delegated realization is the purpose, and the tool supports it ([DEC-001](concepts/register.md#dec-001-delegated-realization-is-the-purpose)). |
| Markitect orchestrates or steers the agents. | It checks and analyses the spec, says what is to verify and records verdicts. Who starts which agent, when and how often belongs to the driver: the skills, the person's own session or the person. There is no Run, no gate and no limit of rounds in the tool ([DEC-096](concepts/register.md#dec-096-the-tool-computes-and-checks-a-driver-acts)). |
| Markitect needs containers, a token, a background session or a Git host, and it commits and lands work. | It runs natively, and the person's own Claude Code session starts the agents on its account login. It uses no Git: the session commits the work with the record and merges it, and nothing is pushed unless a routine the person set up lands by a push, or a cloud routine that cannot land keeps its branch by pushing it ([DEC-094](concepts/register.md#dec-094-the-users-own-session-runs-the-agents-natively), [DEC-097](concepts/register.md#dec-097-markitect-uses-no-git)). |
| Markitect works around Claude Code with machinery of its own. | It works with Claude Code's own skills, subagents, hooks, worktrees, plugins and scheduled tasks ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)). |
| Markitect runs a tree of Manager agents, one per Package. | That was concept 2.0. Since 4 November 2026 there is no tree: a builder and a fresh reviewer work from a list computed from the spec. Packages are modules of the spec, not agents. |
| An empty `markitect status` proves the project correct. | It means that the spec compiles and that every Definition was verified at its current inputs. That is complete for the declared links only, and a verdict is evidence a reviewer gave, not a proof ([concept 4.0, §7](concepts/v4-concept.md#7-how-sure-we-are)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed idea, and the trust bar is today's practice ([DEC-002](concepts/register.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](concepts/register.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| "More economical models" means weak models. | It means capable mid-tier models such as Sonnet, and Haiku only if it proves sufficient ([PRM-005](concepts/register.md#promises)). |
| Earlier comparisons showed that one agent with ordinary documentation wins. | They measured earlier versions, with the Manager tree, containers and Runs, and they showed that machinery's cost. The current aim is not to beat a baseline but to make one simple version reliable ([DEC-009](concepts/register.md#dec-009-how-markitect-is-judged)). |
| The spec has to mirror the code, with one Definition per file. | Granularity is the author's choice; `vet` measures and warns. A file may realize several Definitions, and every file is listed or ignored ([DEC-079](concepts/register.md#dec-079-the-spec-is-as-explicit-as-good-documentation), [DEC-006](concepts/register.md#dec-006-every-file-is-accounted-for)). |
| Green agent-written tests prove the realization. | Tests are realizations too, and agents write them. The reviewer runs them as evidence ([CPT-002](concepts/register.md#cpt-002-tests-are-the-projects-and-they-are-realizations-too)). |
| Removing routine human code review removes the four-eyes principle. | A fresh reviewer that verifies every Definition a change reaches, and a record that only it may write and that a later change voids, are the means to earn that trust. People keep the spec, the `spec-problem`s and their project's acceptance policy ([PRM-004](concepts/register.md#promises)). |

## Feature decision filter

Ask: **does this make the spec applied more reliably, with less babysitting, inside explicit boundaries?**

A proposal names the human activity it removes, where its fact lives, the agents' remaining freedom, what evidence it adds and what it costs to author and maintain. Reconsider proposals that add a second truth to keep in step, hide an architectural decision in inference, tie the tool to one provider, or add machinery without evidence that the work gets easier. More generated text, adapters or agent throughput alone do not show progress.

## What Markitect leaves to people

Markitect does not decide which architecture is right, which business goal matters, whether a rule is desirable, whether evidence is representative or which risk to accept. It does not understand arbitrary source code, guarantee correctness or the absence of drift, or replace human product and architecture judgment. People keep those decisions and may delegate routine work explicitly; machine evidence states only what its inputs and checks establish.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, rules, skills and agent instructions. The spec is Markdown, typed by Kinds. The origin explains the name; it does not limit the product to agent instructions.
