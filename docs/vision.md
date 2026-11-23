# Vision

This page says why Markitect exists, where its trust comes from and what it leaves to people. The [concept](concept/README.md) describes the product, and the [decisions](decisions.md) list what holds. The vision authorizes no feature and claims no measured outcome.

## Markitect in brief

Read this section first, then [Why Markitect exists](#why-markitect-exists), [Founding intent](#founding-intent) and [Common misreadings](#common-misreadings).

**What it is: a way to architect a project in Markdown** ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). An architect draws the plan, builders build to it, and an inspection checks the building against the plan. A change of plan goes through the architect. Markitect works the same way, with one difference: the plan lives and moves with every change. The spec is the plan, and it is binding. Agents build to it, and the result is checked against it. Architecture is the whole design: what the project's concepts are and mean, how they relate, its interfaces, structure, rules and way of working.

**Who holds what.**
- People maintain one authoritative place per project: the spec. It is a typed, declarative specification language in Markdown: documents with a type and ordinary links, grouped in modules whose READMEs say what each shows to the others. It states what is fixed about the project: its concepts, rules, boundaries, interfaces, structure and way of working.
- Code, tests, configuration, documentation and the agents' skills are implementations of the spec. Where they differ from it, the spec is right.
- How the team works is the project's own. The project describes its way of working in its spec, with general types, and skills, agents and scripts in the repository realize it ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).
- Markitect itself is the language, the default that ships with it, a module of general types, and a deterministic tool, the `markitect` program, that checks the spec, analyses it and helps reading it. The tool never writes, keeps no state and steers nothing.

Markitect expects people's work to move from writing code to architecture: designing, defining and modeling. What matters in code matters there too: consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

> **Humans architect the system of work. AI performs the work. Both answer to the spec.**

**What it builds on: proven practice.**
- **Declared configuration.** Terraform makes infrastructure match a configuration that states what should exist, and leaves the steps to its providers. The spec is that configuration for the intent of a project, and agents are its providers.
- **Typing.** A statically typed language reports a contradiction before the program runs. The spec has a small type system of its own, and `markitect check` is its type checker.
- **Good requirements.** Good requirements are judged by the characteristics of ISO/IEC/IEEE 29148. Markitect's properties of a good spec and the signals that say where reading pays off follow them.

**What is new.**
- The statement of what the project should be is plain text in the repository, beside the code: Markdown that people and agents read directly, checked deterministically, with no database and no separate tool to keep in step.
- It covers the whole project, not one change, and it stays current after each change is built.

**The approach: lighter and simpler.** Markitect works with the coding agents' own features rather than against them: a project's way of working uses skills in the repository and `AGENTS.md`, which Claude Code and Codex share, subagents, and scheduled and cloud tasks. The tool starts no agent, steers no work and writes nothing. The simpler version is the more robust one, and simplicity is earned through insight ([DEC-139](decisions.md#dec-139-quality-simplicity-and-stability-over-effort-and-compatibility)).

**The honest limit.** `check` sees structure only. An empty `check` says that the spec is sound in its titles, types, sections and links, not that it says what was meant, and not that the project realizes it. Prose that names a document without linking it is a gap that no check sees; a signal shows such mentions.

## Why Markitect exists

**Agents build the software.** A coding agent works like a capable, fast colleague. The result depends on how well it was told what is wanted, what matters and what belongs to the task. Like any colleague, it can forget a step. Much software is now built through such agents alone: the editor stays closed, code is skimmed at most, and pull requests are reviewed by another AI and seldom read.

**Two things were lost.** Writing and reviewing the code oneself brought two things at no extra cost:

- **Trust.** The author knew that the code reflected what was meant.
- **Overview.** The author knew the project's state, how it works and how it is organized.

Reading everything again would bring both back, and it would make people the bottleneck at the speed agents change a project. People's time belongs to ideas, concepts, organization and delegation: the work of a manager.

**Today's substitutes have no measure.** People ask the agent how something works. They start separate sessions to look for bugs, to judge the architecture, to check that everything fits, and after a fix to check that nothing else broke. Each of these passes looks from one angle, and each does find what the implementing session missed. Two things limit them: "everything fits" has nothing to fit against, and a pass runs only when someone thinks of it. So projects add rules, processes and skills. These live in agent instructions and documentation. They are more artifacts to remember, to maintain and to keep uniform, and more for an agent to overlook. Consolidating them with an agent works on a small project, scales badly, and leaves nothing that shows the result is consistent.

**A standing statement of what should be is missing.** A prompt, a user story and most specs in spec-driven development describe one change or one area of the project. They age once the change is made, and nobody maintains them. People work from their conversations with the agent, and nothing in those is settled for good. At no moment is it written down what the project as a whole should be. The project stays nebulous, without a clear outline, and what exists has nothing to converge on. An agent cannot tell how far the project has moved from what was wanted, cannot find differences that crept in earlier, and cannot tell a bug from intended behavior. It guesses, and each guess makes the result less reliable. Getting what one wants takes knowing what one wants and saying it.

**The spec is that statement.** It is the one place where what the project should be is written down, and what is written there is binding. The spec is what a person works on: they decide what should be, and an agent writes it down with them. Code, tests, configuration and documentation are the agents' work, checked against the spec. The aim is work that nobody has to watch and nobody signs off. Today such approvals are mostly nodded through unread, which helps nobody; trust comes instead from facts that can turn out negative without anyone reading. A question an agent should not settle for itself shows that the spec leaves out or contradicts something that matters, and the spec is the place to settle it.

Trust and overview are rebuilt on the spec:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | The spec states the whole project in one place, with links between what relies on what, so what a change reaches can be followed instead of remembered |
| Was it understood correctly? | Every result is checked against the spec, not against a prompt or a memory of a conversation |
| Does the spec say what was meant? | The person answers for the spec and develops it in conversation with an agent. `check` finds structural errors, signals say where reading pays off, and readers and competency questions find contradictions, gaps and unclear statements; use shows the rest |
| How does the project stand, and what changed? | The tool shows the spec, and the agent explains it with a citation for each statement. A change to the spec is seen in the repository's diff, like every change |

How a project checks its results, and measures that checking, is its way of working, which its spec states ([Where trust comes from](#where-trust-comes-from)). [Expected outcomes](#expected-outcomes) says what is still to be shown.

## Founding intent

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. It had explicit boundaries between its core, its components and its product, and a specified working method, review policies and skills per component. Code, documentation and the agents' skills repeatedly contradicted one another; changes missed rules, every consistency run took hours and found more, and nobody could tell any more which rules existed, what was defined and what had entered the project silently.

**The whole project, as far as it is fixed.** The spec covers the whole project: which components and deployments exist, what a concept such as a book is and what can be done with it, the rules and the way of working. It states what is fixed, what someone relies on or what was decided, such as data shapes and the answer to a refused call, and leaves loops and data structures to the agents ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). The concept says [what belongs in it](concept/what-goes-in-a-spec.md) and what happens where it leaves something open.

**One source, and why it is documentation.** A project is many artifacts that work together: code, tests, configuration and documentation. They drift apart. When two of them contradict each other, nothing says which is right, so the agent guesses or a person brings them back in step. Software solves this with a single source of truth and gives it up only where a system must scale across machines. A project's description never has to. Code could be that source: it compiles, it can be tested and it is exact. Code cannot state everything, though. A deployment or a way of working needs documentation, Dockerfiles and agent rules beside the code, and the detail of code makes it rigid. Documentation, kept as one source, can state everything. It lacks a compiler and tests, and it tends to say the same thing in many places.

**Code is a description.** Code has always been a precise description for an extremely limited but diligent executor. With AI the executor is intelligent, but it still has to be told what is meant and what holds; otherwise it guesses, and may do what nobody wanted. So documentation takes the place of code: it is what people work on. Markitect makes its authors express themselves cleanly: define what they mean and what holds, and connect those statements meaningfully.

**The answer is a typed, declarative specification language.** Whether documentation is consistent shows only at run time: a contradiction surfaces when an agent or a person stumbles over it while working. Dynamically typed programming languages have the same problem. Statically typed ones such as Go and C# are valued because a compiler reports such errors early, and the author has to say explicitly what is meant. So the spec gets a small type system of its own: a type says which sections its documents have, and a tool checks titles, types, sections and links. And like a declarative language, the spec says what should hold and leaves how to an executor; here the executor is an agent, so the result is checked, not guaranteed.

| In a typed, declarative language | In a Markitect spec |
| --- | --- |
| type, class | type: a document that declares the sections of its documents |
| object, declaration | document |
| the body | the prose of a section: what holds, which agents realize |
| reference | an ordinary Markdown link |
| package, with exported and internal names | module: a folder whose README shows what other modules may link |
| standard library | the default's module `core`, copied into the spec, which owns its copy |
| type checker, `validate` | `markitect check`: structure only, never meaning |

- The core stays small. Projects define their own types and so form their own vocabulary; the default supplies common ones, so that no project reinvents them.
- The tool checks structure. Meaning is judged by agents and people: readers judge the spec itself, and whoever checks the project judges its artifacts against the spec.
- People decide intent and answer for the spec. Agents write it with them in conversation, and ask first when something is open.

**Design against drift.** A check sees structure. It cannot see that the same fact stands in several places, and such places move apart with every change until it is unclear again what should be. So the spec has a writing rule: every fact stands in one place, and everywhere else links to it. A good spec is necessary, complete, consistent, unambiguous and verifiable, and structured, after the characteristics of good requirements in ISO/IEC/IEEE 29148 ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). Clean code holds that good code reads like well-written prose. Markitect asks the reverse: prose that is built like clean code. Results of Markitect are only as good as the spec they use.

**What the spec is for.** The typed spec is the foundation, and applying it is the purpose. One source settles which side is right when the spec and the project differ. Making them agree takes reconciliation: find what a change reaches, realize it, check it, repeat. How a project does that, and which runs it keeps, is its way of working, which it states in its spec. With the spec as their measure, such runs need not wait for someone to think of them. No single pass has to find everything: a later one can catch what an earlier one missed, and trust grows as the project converges on its spec.

## Where trust comes from

Markitect does not try to make agents infallible. It aims at trust that comes from structure and is measured, not granted ([DEC-137](decisions.md#dec-137-trust-is-measured-against-todays-practice), [DEC-125](decisions.md#dec-125-trust-is-measured-not-granted)): one binding spec, checked by a tool, against which every result can be checked, and facts that can turn out negative without anyone reading. Which mechanisms earn that trust in a project, such as an independent review or checks derived from the spec alone, and how they are measured, is part of its way of working, which the project states in its spec.

Two claims stay apart ([DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)): *the spec is sound in its structure* is checked by the tool; *the project realizes the spec correctly* can only be supported by evidence. The tests a builder writes belong to the realization and can be as wrong as the code.

## Who does what

| Role | Responsibility |
|---|---|
| People | Decide what should exist and why; choose concepts, architecture, boundaries, rules and the way of working; answer for what the spec means. |
| Markitect | Ship the language and a default of general types; check the spec, analyse it and help reading it. The tool writes nothing, keeps no state, starts no agent and steers no work. |
| Agents | Write the spec with people, realize it and check the result, as the project's way of working says. An agent's report is not a person's approval. |

A real change of intent changes the spec first. When intent is unchanged, a bug or a stale file is repaired against the same spec; no artificial spec edit is needed. The target human role is specification and engineering judgment, not being a permanent agent scheduler, prompt supervisor or reader of every routine diff. Whether that work can actually be reduced is what Markitect has to show.

## Expected outcomes

Target outcomes, each to be validated; none is an established result.

- Nothing in the spec's scope is silently forgotten: what a change reaches follows from the spec's links, not from an agent's memory. Unlinked mentions are known gaps.
- Rules are applied where they matter.
- The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable.
- Software becomes understandable: the spec states what the system does and should do, and people can reason over it without reading the code.
- Order pays off: structure and careful definition are rewarded instead of turning into endless cleanup.
- People concentrate on architecture. Their attention moves from code diffs to the decisions only they can make, while the code reliably reflects the spec, and the trust rests on checking that the project's way of working measures, rather than on reading.

The intended end state is **spec → AI realizes it and checks it independently, in a way of working that measures the checking → people rely on the result without reviewing code diffs**. This is a target to earn through evidence, not a claim that Markitect has achieved it.

## Assumptions

Why the method should help, and what would show that it does not.

| Assumption | What would challenge it |
|---|---|
| An agent that is told fully what is wanted, what matters and what belongs to the task builds what was meant | Results match the intent no better with a complete spec than with a short prompt |
| A person can answer for a spec written with an agent, from the conversations and what use shows | Wrong statements stay unnoticed in the spec, are realized faithfully and are found only in the product |
| One authoritative spec with maintained links reduces the mistakes of keeping things in step | Contradictions still pile up, or keeping the spec costs as much as the consolidation it replaces |
| Typing catches useful structural errors before agents work | Most consequential failures pass `check` and still need the same late investigation |
| Forgetting, rather than misunderstanding, is the main problem | Most failures turn out to be wrong readings of a document the agent did see |

## How to judge it

- Judge Markitect's trust against today's practice, not against a compiler's guarantees ([DEC-137](decisions.md#dec-137-trust-is-measured-against-todays-practice)).
- "Compiler" names a borrowed idea. The spec's structure is checked deterministically; realization by agents is not deterministic ([DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Delegated realization is the purpose the spec serves: the spec is there to be built from, not to be checked for its own sake. Markitect makes it possible with the language, the default and the tool, and ships none of the realization; a project's way of working does it ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)).
- The aim is one simple version that works reliably; when it counts as proven is open ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)).

## Common misreadings

Check an assessment against this table before treating one of these as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the project began. With Markitect, people architect the whole project in Markdown, and agents realize it. Agent instructions are one kind of realization among many. |
| Markitect is a tool for architecture documentation. | The spec is the plan the project is built from, and it is binding: agents realize it, and the result is checked against it. Architecture covers the domain too: what a concept is, what can be done with it and how things relate. |
| Markitect is a programming language. | Nothing in the spec is executed. It is a typed, declarative specification language, as TLA+ and Alloy are specification languages: its structure is checked, and agents realize its content ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language), [DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). People's work is architecture, and agents write the code. |
| Markitect is a new, unproven formalism. | It builds on proven practice: a declared configuration as Terraform has it, static typing, and the characteristics of good requirements. What is new is that the statement of what the whole project should be is plain text checked deterministically, and that agents realize it. |
| Markitect is a validator for Markdown, and building from the spec is beside the point. | The spec is there to be built from: delegated realization is the purpose it serves. Markitect ships the language, the default and the tool that make it possible, and none of the realization, which a project's way of working does ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)). |
| Markitect steers the agents through its tool. | The tool checks the spec, analyses it and helps reading it, and steers nothing: it writes nothing and keeps no state. How agents work is the project's way of working, which its spec states and its skills realize; every step of the project's version control belongs to agents and people, and Markitect neither needs nor knows it ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec), [DEC-097](decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)). |
| Markitect needs API keys or a platform of its own. | The tool runs natively on Windows, Linux and macOS and needs no key, no account and no platform, and it starts no agent ([DEC-135](decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language), [DEC-097](decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)). |
| A person approves every change. | Markitect aims at trust without approvals, from facts that can turn out negative without anyone reading, which a project's way of working supplies ([DEC-125](decisions.md#dec-125-trust-is-measured-not-granted)). People keep the intent and the new decisions. |
| Markitect forces a way of working on the project. | Markitect ships no way of working. A project states its own in its spec, with general types, and owns and changes it like any part of the spec ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). |
| Stories and findings belong in the spec. | They are the work that changes the spec, not its target. How a project keeps them, in which form and where, is part of its way of working ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). |
| An empty `markitect check` proves the project correct. | It means that the spec is sound in its structure: titles, types, sections and links. Readers judge its meaning, and only evidence supports that the project realizes it ([DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed idea, and the trust bar is today's practice ([DEC-137](decisions.md#dec-137-trust-is-measured-against-todays-practice), [DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| The spec has to mirror the code. | It states only what is fixed; what nobody relies on stays free, and one document may have many artifacts ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). |
| Green agent-written tests prove the realization. | The tests a builder writes are realizations too, and can be as wrong as the code: they are evidence, not proof ([DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| Removing routine human code review removes the four-eyes principle. | An independent check against the spec, and a measure of how well it works, are the means to earn that trust, as the project's way of working sets them up. People keep the intent and the new decisions. |

## What Markitect leaves to people

Markitect does not decide which architecture is right, which business goal matters, whether a rule is desirable, whether evidence is representative or which risk to accept. It does not understand arbitrary source code, guarantee correctness or the absence of drift, or replace human product and architecture judgment. People keep those decisions; machine evidence states only what its inputs and checks establish.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, rules, skills and agent instructions. The spec is Markdown, typed by its own types. The name says what a person does with Markitect: architect their project in Markdown. The origin does not limit the product to agent instructions.
