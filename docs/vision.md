# Vision

This page says why Markitect exists, where its trust comes from and what it leaves to people. The [concept](concept.md) describes the product, and the [decisions](decisions.md) list what holds. The vision authorizes no feature and claims no measured outcome.

## Markitect in brief

Read this section first, then [Why Markitect exists](#why-markitect-exists), [Founding intent](#founding-intent) and [Common misreadings](#common-misreadings).

**What it is: a way to architect a project in Markdown** ([DEC-101](decisions.md#dec-101-people-architect-their-project-in-markdown)). An architect draws the plan, builders build to it, and an inspection checks the building against the plan. A change of plan goes through the architect. Markitect works the same way. The spec is the plan, and it is binding. The builder agent builds to it, and the reviewer checks the result against it. Architecture is the whole design: what the project's concepts are and mean, how they relate, its interfaces, structure, rules and way of working. Architect is a role that the owner and an agent fill together: the agent drafts, analyses and proposes, and the owner decides.

**The parts.**
- People maintain one authoritative place per project: the spec. It is typed prose: Markdown Definitions, each of a Kind, organized in Packages, with links that work like imports. It states the project's concepts, rules, boundaries, interfaces, structure and way of working, as far as good documentation of the project would.
- Code, tests, configuration and documentation are realizations of the spec. Where they differ from it, the spec is right.
- A deterministic tool, the `markitect` program, checks and analyses the spec as a compiler and a linter check code, and keeps a record of what was verified. Agents realize the spec and verify the result.

Markitect expects people's work to move from writing code to architecture: designing, defining and modeling. What matters in code matters there too: consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

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

## Why Markitect exists

**Agents build the software.** A coding agent works like a capable, fast colleague. The result depends on how well it was told what is wanted, what matters and what belongs to the task. Like any colleague, it can forget a step. Much software is now built through such agents alone: the editor stays closed, code is skimmed at most, and pull requests are reviewed by another AI and seldom read.

**Two things were lost.** Writing and reviewing the code oneself brought two things at no extra cost:

- **Trust.** The author knew that the code reflected what was meant.
- **Overview.** The author knew the project's state, how it works and how it is organized.

Reading everything again would bring both back, and it would make people the bottleneck at the speed agents change a project. People's time belongs to ideas, concepts, organization and delegation: the work of a manager.

**Today's substitutes have no measure.** People ask the agent how something works. They start separate sessions to look for bugs, to judge the architecture, to check that everything fits, and after a fix to check that nothing else broke. Each of these passes looks from one angle, and each does find what the implementing session missed. Two things limit them: "everything fits" has nothing to fit against, and a pass runs only when someone thinks of it. So projects add rules, processes and skills. These live in agent instructions and documentation. They are more artifacts to remember, to maintain and to keep uniform, and more for an agent to overlook. Consolidating them with an agent works on a small project, scales badly, and leaves nothing that shows the result is consistent.

**A standing statement of what should be is missing.** A prompt, a user story and most specs in spec-driven development describe one change or one part of the project. They age once the change is made, and nobody maintains them. People work from their conversations with the agent, and nothing in those is settled for good. At no moment is it written down what the project as a whole should be. The project stays nebulous, without a clear outline, and what exists has nothing to converge on. An agent cannot tell how far the project has moved from what was wanted, cannot find deviations that crept in earlier, and cannot tell a bug from intended behavior. It guesses, and each guess makes the result less reliable. Getting what one wants takes knowing what one wants and saying it.

**The spec is that statement.** It is the one place where what the project should be is written down, and what is written there is binding. The spec is what a person works on: they decide what should be, and an agent writes it down with them. Code, tests, configuration, documentation and the registry are the agents' work, and the reviewer records each verdict. The aim is a build that nobody has to watch, and no sign-off at the end. Today such approvals are mostly nodded through unread, which helps nobody. Verdicts and their evidence direct the agents' attention, and the person reads them when they want to. A question an agent should not settle for itself shows that the spec leaves out or contradicts something that matters. The reviewer records it as a `spec-problem`, and the person settles it.

Trust and overview are rebuilt on the spec:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | `status` computes what is to verify from the spec and the record, like a checklist. Surgeons use checklists for the same reason: they know every step and can still skip one |
| Was it understood correctly? | A fresh reviewer judges the code against the spec and records, per Definition, what it found and the evidence |
| Does the spec say what was meant? | The person answers for the spec. They develop it in conversation with an agent, and read a change or go through it with the agent. `check` finds structural errors, `vet` measures the design, and the auditor reports contradictions, gaps, unclear and duplicated statements |
| How does the project stand, and how is it organized? | The spec shows it. Its structure gives the project's division, modules, architecture and dependencies, through `markitect list`, `doc` and `vet`. It changes only when a person accepts a change, and `status` lists what is not yet verified |

[Where trust comes from](#where-trust-comes-from) lists the mechanisms, and [Expected outcomes](#expected-outcomes) says what is still to be shown.

## Founding intent

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. It had explicit Core, Module and Product boundaries, and a specified working method, review policies and skills per module. Code, documentation and the agents' skills repeatedly contradicted one another; changes missed rules, every consistency run took hours and found more, and nobody could tell any more which rules existed, what was defined and what had entered the project silently.

**The whole project, as good documentation would state it.** The spec covers the whole project: which modules and deployments exist, what a concept such as a book is and what can be done with it, the rules and the way of working. It states what callers rely on, such as data shapes and status codes, and leaves loops and data structures to the agents ([DEC-079](decisions.md#dec-079-the-spec-is-as-explicit-as-good-documentation)). The concept says [what belongs in it](concept.md#the-spec) and what happens where it leaves something open.

**One source, and why it is documentation.** A project is many artifacts that work together: code, tests, configuration and documentation. They drift apart. When two of them contradict each other, nothing says which is right, so the agent guesses or a person brings them back in step. Software solves this with a single source of truth and gives it up only where a system must scale across machines. A project's description never has to. Code could be that source: it compiles, it can be tested and it is exact. Code cannot state everything, though. A deployment or a way of working needs documentation, Dockerfiles and agent rules beside the code, and the detail of code makes it rigid. Documentation, kept as one source, can state everything. It lacks a compiler and tests, and it tends to say the same thing in many places.

**Code is a description.** Code has always been a precise description for an extremely limited but diligent executor. With AI the executor is intelligent, but it still has to be told what is meant and what holds; otherwise it guesses, and may do what nobody wanted. So documentation takes the place of code: it is what people work on. Markitect makes its authors express themselves cleanly: define what they mean and what holds, and connect those statements meaningfully.

**The answer is typed documentation with a compiler.** Whether documentation is consistent shows only at run time: a contradiction surfaces when an agent or a person stumbles over it while working. Dynamically typed programming languages have the same problem. Statically typed ones such as Go and C# are valued because a compiler and explicit dependencies report such errors at compile time, and the author has to say explicitly what is meant. So Markitect borrows typing and a compiler as helpers. The spec applies that to project intent:

| In a typed programming language | In a Markitect spec |
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

- The core stays clean. Projects define their own Kinds and so form their own vocabulary; Libraries supply common Kinds, so that no project reinvents them.
- The compiler checks structural correctness from the core alone. Meaning is judged by agents: the reviewer judges the code against the spec, and the auditor judges the spec itself.
- The spec carries the structure that keeps software changeable, after Conway's law, Clean Architecture, clean code and modularity: Packages are modules, and links go up the tree or sideways towards stability, never down.
- People decide intent and answer for the spec. Agents write it in conversation through the design skill, which asks first when something is open.

**Design against drift.** A compiler checks structure. It cannot see that the same fact stands in several places, and such places move apart with every change until it is unclear again what should be. Code has the same problem and manages it through design. Links are declared dependencies, so the same design principles apply to the spec. Clean code holds that good code reads like well-written prose. Markitect asks the reverse: prose that is built like clean code.

**The aim of the typing.** Meet the preconditions that open the established knowledge of software architecture to the spec: Definitions with a type, dependencies that are declared, and a structure a tool can check and measure. So judge a spec as you would judge code: dependencies, coupling and stability, clean architecture and size. A Definition of 500 words is prose compensating for missing Definitions. Results of Markitect are only as good as the spec they use.

**What the spec is for.** The typed spec is the foundation, and applying it is the purpose. One source settles which side is right when the spec and the project differ. Making them agree takes reconciliation, as infrastructure tools do it with a declared state: compute what a change reaches, realize it, verify it, repeat. Each pass sees from one angle, so Markitect keeps the kinds of pass apart, each with its own question: build, review, cleanup, bug hunt and audit. With the spec as their measure, cleanup, bug hunt and a full review can run as routines on a schedule, without waiting for someone to think of them. No single pass has to find everything: a later one can catch what an earlier one missed, and trust is meant to grow as the project converges on its spec. An audit, started by the person or by Claude, reads the spec itself with fresh eyes and reports what is contradictory, missing, unclear or duplicated.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice)). Each mechanism below replaces something that today's practice leaves to chance; [the concept](concept.md#7-how-sure-we-are) says how firm each one is.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **A list computed from the spec.** `status` lists every Definition whose text, Kind, links, files or evidence changed since a reviewer verified it, and each listed Definition needs a verdict. | An agent deciding from memory which files might be relevant |
| **Roles with bounded rights.** The builder writes code and the registry pages, not the spec; the reviewer changes nothing and alone records; the architect writes the spec, not code; the auditor only reads and reports. A hook holds these rights for subagents. | One agent that writes, judges and records its own work |
| **Independent review with fresh context.** The reviewer takes its list from `status`, reads each Definition, the Definitions it links to and the code, runs the project's tests where they bear on an item, and treats what the builder tells it as claims to check. | Self-review in the same thread, or one generic pull-request review |
| **Rules in the way.** A Rule reaches every Package below the one that defines it, an `ok` means the Rules of its Package and those above it held for that Definition when it was verified, and a changed Rule is back on the list with every Definition that links to it. | Rules in a long instruction file that are overlooked when nothing forces the agent to see them |
| **Verdicts bound to what was read.** `mark` refuses a verdict when an input changed since `status` printed the item, the evidence must exist and is hashed, and the record is committed with the work. A later change puts what it touches back on the list, so `markitect status` in CI shows what is not verified. | Approving a pull request because its pipeline is green |

Two claims stay apart ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)): *nothing in the spec's scope is overlooked* can be computed for declared links and file accounting; *everything is realized correctly* can only be supported by evidence. Tests belong to the realization: agents write them, so they can be as wrong as the code.

## Who does what

| Role | Responsibility |
|---|---|
| People | Decide what should exist and why; choose concepts, architecture, boundaries, rules and processes; define the agents' freedom and the project's acceptance policy; settle every `spec-problem`. |
| Markitect | Compile the spec, account for every file, say which Definitions are to verify because an input changed, record each verdict at the state the reviewer read, and keep the agents that build from changing the spec, and every agent but the reviewer from recording. It starts no agent, steers no work and uses no Git. |
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
- People concentrate on architecture. Their review moves from code diffs to spec changes and `spec-problem`s, while the code reliably reflects the spec. A fresh reviewer stays part of the method, and a project's own acceptance policy still applies.

The intended end state is **spec → AI realizes and independently reviews it → people rely on the result without routinely reviewing code diffs**. This is a target to earn through evidence, not a claim that Markitect has achieved it.

## Assumptions

Why the method should help, and what would show that it does not.

| Assumption | What would challenge it |
|---|---|
| An agent that is told fully what is wanted, what matters and what belongs to the task builds what was meant | Results match the intent no better with a complete spec than with a short prompt |
| A person can answer for a spec written with an agent, from its structure and by going through its changes | Wrong statements stay unnoticed in the spec, are realized faithfully and are found only in the product |
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
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the project began. With Markitect, people architect the whole project in Markdown, and agents realize it. Agent instructions are one kind of realization among many. |
| Markitect is a tool for architecture documentation. | The spec is the plan the project is built from, and it is binding: agents realize it, and a reviewer verifies the result against it. Architecture covers the domain too: what a concept is, what can be done with it and how things relate. |
| Markitect is a programming language. | Nothing in the spec is executed. Typing and a compiler are borrowed to check the spec's structure ([DEC-101](decisions.md#dec-101-people-architect-their-project-in-markdown), [DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). People's work is architecture, and agents write the code. |
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

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, rules, skills and agent instructions. The spec is Markdown, typed by Kinds. The name says what a person does with Markitect: architect their project in Markdown. The origin does not limit the product to agent instructions.
