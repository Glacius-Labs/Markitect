# Vision

This page says why Markitect exists, where its trust comes from and what it leaves to people. The [concept](concept.md) describes the product, and the [decisions](decisions.md) list what holds. The vision authorizes no feature and claims no measured outcome.

## Markitect in brief

Read this section first, then [Founding intent](#founding-intent) and [Common misreadings](#common-misreadings).

**What it is: a programming language for documentation** ([DEC-101](decisions.md#dec-101-markitect-is-a-programming-language-for-documentation)).
- People maintain one authoritative place per project: the spec. It is typed prose: Markdown Definitions, each of a Kind, organized in Packages, with links that work like imports. It states the project's concepts, rules, boundaries, interfaces, architecture and way of working, as far as good documentation of the project would.
- Code, tests, configuration and documentation are realizations of the spec. Where they differ from it, the spec is right.
- A deterministic tool, the `markitect` program, checks and analyses the spec as a compiler and a linter check code, and keeps a record of what was verified. Agents realize the spec and verify the result.

Markitect expects coding to move from languages such as Go, C# or Python towards documentation and meaning. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

**What it builds on: proven practice.**
- **Requirements traceability.** Tools such as IBM DOORS and Polarion keep a traceability matrix that links requirements to what realizes them, in both directions, and mark a link suspect when either end changes; DO-178C, ISO 26262 and IEC 62304 require such traceability. In Markitect the registry is the matrix, `markitect status` lists the suspect items, and `markitect mark` records a verdict on each; an `ok` with its evidence clears it.
- **Architecture metrics.** The spec's design is judged with the principles that keep code changeable: Clean Architecture's Dependency Rule, domain-driven design's bounded contexts, cohesion and coupling, Robert C. Martin's stability measures and Adam Tornhill's change hotspots ([what makes a good spec](guides/good-spec.md)).
- **Go's conventions.** `vet`, `list`, `doc` and `tidy` do what their Go namesakes do, diagnostics read `file:line: code: message`, exit codes follow Go's, and the record of what was verified works like Go's test cache.

**What is new.**
- The traceability is plain text in the repository, beside the code: Markdown that people and agents read directly, typed by Kinds and checked deterministically, with no database and no separate tool to keep in step.
- AI does not take everything off the developer's hands, but it makes affordable what ordinary software projects dropped because of the manual effort and cost, such as keeping a traceability matrix. Agents do the realization and gather the evidence.
- A verdict holds until one of its inputs changes, so a change puts back on the list only what it reaches.

**The approach: lighter, native and simpler.** Markitect works with Claude Code's own features rather than against them: skills, subagents, hooks, worktrees, a plugin and Claude Code's own scheduling. The tool starts no agent, steers no work and uses no Git. The simpler version is the more robust one, and simplicity is earned through insight ([DEC-032](decisions.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

**The honest limit.** "Complete" means complete for the declared links. `status` follows a Definition's own text, its Kind, the Definitions it links to, its registry rows and its evidence, one step deep; it does not read the code's imports. Prose that names a concept without linking it is a gap: a change to that concept does not put the naming Definition back on the list. `vet` notes such names, and the auditor tells real missing links from ordinary words; names a text check does not match, only a careful reader finds. An empty `status` says that every Definition was verified at its current inputs, not that every verdict is right.

## Why: three problems

- **Nobody keeps track of what was decided.** In a long session or a large project, nobody can remember or keep current what was decided when; coding agents lose track of relevant files and rules as projects grow, and explicit consistency runs miss things and scale badly. The spec is the one living statement of what holds.
- **Nobody reads all the documentation.** AI produces more documentation than anyone reads. The spec is small, structured and navigable: Definitions in Packages, with links, and `markitect list` and `doc` to find one's way.
- **Nobody knows whether the realization matches the idea.** Common practice nevertheless trusts the agents: an agent implements a work item, another AI reviews the pull request, and the required human approval is often given almost blind. `status` and the record say per Definition what a fresh reviewer found, with its evidence.

So the knowledge of what belongs to what must not depend on an agent's memory, and people cannot read every diff without becoming the bottleneck.

## Founding intent

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. It had explicit Core, Module and Product boundaries, and a specified working method, review policies and skills per module. Code, documentation and the agents' skills repeatedly contradicted one another; changes missed rules, every consistency run took hours and found more, and nobody could tell any more which rules existed, what was defined and what had entered the project silently.

**Code is a description.** Code has always been a very precise description for an extremely limited but diligent executor. With AI the executor is intelligent, but it still has to be told what is meant and what holds; otherwise it guesses, and may do what nobody wanted. So documentation, prose and prompts become the programming language. Markitect makes their authors express themselves cleanly: define what they mean and what holds, and connect those statements meaningfully.

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

- The core stays clean. Projects define their own Kinds and so form their own language; Libraries supply common Kinds, so that no project reinvents them.
- The compiler checks structural correctness from the core alone. Meaning is judged by agents: the reviewer judges the code against the spec, and the auditor judges the spec itself.
- The spec carries the structure that keeps software changeable, after Conway's law, Clean Architecture, clean code and modularity: Packages are modules, and links go up the tree or sideways towards stability, never down.
- People decide intent. Agents write the spec in conversation through the design skill, which asks first when something is open; the person reads the Markdown diff and accepts it.

**The aim of the language.** Meet the preconditions that unlock the established knowledge of software architecture for this new language: Definitions with a type, dependencies that are declared, and a structure a tool can check and measure. So judge a spec as you would judge code: dependencies, coupling and stability, clean architecture and size. A Definition of 500 words is prose compensating for missing Definitions. Results of Markitect are only as good as the spec they use.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice)). Each mechanism below replaces something that today's practice leaves to chance; [the concept](concept.md#7-how-sure-we-are) says how firm each one is.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **A list computed from the spec.** `status` lists every Definition whose text, Kind, links, files or evidence changed since a reviewer verified it, and each listed Definition needs a verdict. | An agent deciding from memory which files might be relevant |
| **Roles with bounded rights.** The builder writes code and the registry pages, not the spec; the reviewer changes nothing and alone records; the architect writes the spec, not code; the auditor only reads and reports. A hook holds these rights for subagents. | One agent that writes, judges and records its own work |
| **Independent review with fresh context.** The reviewer takes its list from `status`, reads each Definition, the Definitions it links to and the code, runs the project's tests where they bear on an item, and treats what the builder tells it as claims to check. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** A Rule reaches every Package below the one that defines it, an `ok` means the Rules above hold for that Definition, and a changed Rule is back on the list with every Definition that links to it. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Verdicts bound to what was read.** `mark` refuses a verdict when an input changed since `status` printed the item, the evidence must exist and is hashed, and the record is committed with the work. A later change puts what it touches back on the list, so `markitect status` in CI shows what is not verified. | Approving a pull request because its pipeline is green |

Two claims stay apart ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)): *nothing in the spec's scope is overlooked* can be computed for declared links and file accounting; *everything is realized correctly* can only be supported by evidence. Tests belong to the realization: agents write them, so they can be as wrong as the code.

## Who does what

| Role | Responsibility |
|---|---|
| People | Decide what should exist and why; choose concepts, architecture, boundaries, rules and processes; define the agents' freedom and the project's acceptance policy; settle every `spec-problem`. |
| Markitect | Compile the spec, account for every file, say which Definitions are to verify because an input changed, record each verdict at the state the reviewer read, and keep agents from recording their own work or changing the spec. It starts no agent, steers no work and uses no Git. |
| Agents | Draft spec changes for people to accept, realize the spec, review the result and record verdicts with evidence, and raise what only people can settle. An agent's report is not a person's approval. |

A real change of intent changes the spec first. When intent is unchanged, a bug or a stale file is repaired against the same spec; no artificial spec edit is needed. The target human role is specification and engineering judgment, not being a permanent agent scheduler, prompt supervisor or reader of every routine diff. Whether that work can actually be reduced is what Markitect has to show.

## Expected outcomes

Target outcomes, each to be validated; none is an established result.

- Nothing in the spec's scope is silently forgotten: what is to verify comes from the spec and the record, not from an agent's memory. This covers declared links; code dependencies and unlinked mentions are known gaps.
- Rules are applied where they matter.
- The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable.
- Software becomes understandable: the spec states what the system does and should do, and people can reason over it without reading the code.
- Order pays off: structure and careful definition are rewarded instead of turning into endless cleanup.
- Capable mid-tier models, such as Sonnet, do most of the delegated work; Haiku counts if it proves sufficient. Lower cost is a possible benefit, not the purpose.
- People concentrate on deciding and modeling. Their review moves from code diffs to spec changes and `spec-problem`s, while the code reliably reflects the spec. A fresh reviewer stays part of the method, and a project's own acceptance policy still applies.

The intended end state is **spec → AI realizes and independently reviews it → people rely on the result without routinely reviewing code diffs**. This is a target to earn through evidence, not a claim that Markitect has achieved it.

## Assumptions

Why the method should help, and what would show that it does not.

| Assumption | What would challenge it |
|---|---|
| One authoritative spec with maintained links reduces the mistakes of keeping things in step | Contradictions still pile up, or keeping the spec and the registry costs as much as the consolidation it replaces |
| Typing catches useful structural errors before agents work | Most consequential failures pass `check` and still need the same late investigation |
| A list computed from declared links keeps continuing obligations in view | What a change really affects falls outside the list, or the list grows to nearly the whole spec on every change |
| A reviewer with fresh context finds mistakes the builder misses, even on the same model | Reviewers keep accepting the same omissions or wrong readings, and extra calls add no useful evidence |
| Forgetting, rather than misunderstanding, is the main problem | Most failures turn out to be wrong readings of a Definition the agent did see |
| What works in human teams transfers to agents: four eyes, bounded roles and clear responsibility | Bounded roles add hand-offs and cost without catching more |
| A typed spec lets capable mid-tier models do most of the work | Rework and repeated review eat the savings, or only the most expensive models get it right |
| The record and its evidence remove the need to read every routine diff | People still need to review every diff, cannot tell what is unverified, or trade that work for equally costly spec supervision |

## How to judge it

- Judge Markitect's trust against today's practice, not against a compiler's guarantees ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice)).
- "Compiler" names a borrowed idea. The spec's structure compiles deterministically; realization by agents is not deterministic ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Delegated realization is central to the product. It is not an optional layer on top of a validator ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)).
- The current aim is one simple version that works reliably on Sonnet ([DEC-009](decisions.md#dec-009-how-markitect-is-judged)).

## Common misreadings

Check an assessment against this table before treating one of these as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the name comes from. Markitect is a programming language for documentation of the whole project, with delegated realization. Agent instructions are one kind of realization among many. |
| Markitect is a new, unproven formalism. | It builds on proven practice: requirements traceability with suspect links, architecture metrics and Go's conventions. What is new is that the traceability is plain, typed text checked deterministically, that agents realize the spec and gather the evidence, and that the record works like Go's test cache. |
| The deterministic parts are the product, and agent work is scope creep. | Delegated realization is the purpose, and the tool supports it ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)). |
| Markitect orchestrates or steers the agents. | It checks and analyses the spec, says what is to verify and records verdicts. Who starts which agent, when and how often belongs to the driver: the skills, the person's own session or the person ([DEC-096](decisions.md#dec-096-the-tool-computes-and-checks-a-driver-acts)). |
| Markitect needs a token, a background session or a Git host, and it commits and lands work. | It runs natively, and the person's own Claude Code session starts the agents on its account login. It uses no Git: the session commits the work with the record and merges it, and nothing is pushed unless a routine the person set up lands by a push ([DEC-094](decisions.md#dec-094-the-users-own-session-runs-the-agents-natively), [DEC-097](decisions.md#dec-097-markitect-uses-no-git)). |
| Packages are agents, one per area. | Packages are modules of the spec, not agents. A builder and a fresh reviewer work from a list computed from the spec; on a long list one of each may work per Package. |
| An empty `markitect status` proves the project correct. | It means that the spec compiles and that every Definition was verified at its current inputs. That is complete for the declared links only, and a verdict is evidence a reviewer gave, not a proof ([concept, §7](concept.md#7-how-sure-we-are)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed idea, and the trust bar is today's practice ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| "More economical models" means weak models. | It means capable mid-tier models such as Sonnet, and Haiku only if it proves sufficient. |
| The spec has to mirror the code, with one Definition per file. | Granularity is the author's choice; `vet` measures and warns. A file may realize several Definitions, and every file is listed or ignored ([DEC-079](decisions.md#dec-079-the-spec-is-as-explicit-as-good-documentation), [DEC-006](decisions.md#dec-006-every-file-is-accounted-for)). |
| Green agent-written tests prove the realization. | Tests are realizations too, and agents write them. The reviewer runs them as evidence ([DEC-099](decisions.md#dec-099-a-record-of-what-was-verified)). |
| Removing routine human code review removes the four-eyes principle. | A fresh reviewer that verifies every Definition a change reaches, and a record that only it may write and that a later change voids, are the means to earn that trust. People keep the spec, the `spec-problem`s and their project's acceptance policy. |

## Feature decision filter

Ask: **does this make the spec applied more reliably, with less babysitting, inside explicit boundaries?**

A proposal names the human activity it removes, where its fact lives, the agents' remaining freedom, what evidence it adds and what it costs to author and maintain. Reconsider proposals that add a second truth to keep in step, hide an architectural decision in inference, tie the tool to one provider, or add machinery without evidence that the work gets easier. More generated text, adapters or agent throughput alone do not show progress.

## What Markitect leaves to people

Markitect does not decide which architecture is right, which business goal matters, whether a rule is desirable, whether evidence is representative or which risk to accept. It does not understand arbitrary source code, guarantee correctness or the absence of drift, or replace human product and architecture judgment. People keep those decisions and may delegate routine work explicitly; machine evidence states only what its inputs and checks establish.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, rules, skills and agent instructions. The spec is Markdown, typed by Kinds. The origin explains the name; it does not limit the product to agent instructions.
