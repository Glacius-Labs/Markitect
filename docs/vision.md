# Vision

This page says why Markitect exists, where its trust comes from and what it leaves to people. The [concept](concept/README.md) describes the product, and the [decisions](decisions.md) list what holds. The vision authorizes no feature and claims no measured outcome.

## Markitect in brief

Read this section first, then [Why Markitect exists](#why-markitect-exists), [Founding intent](#founding-intent) and [Common misreadings](#common-misreadings).

**What it is: a way to architect a project in Markdown** ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). An architect draws the plan, builders build to it, and an inspection checks the building against the plan. A change of plan goes through the architect. Markitect works the same way, with two differences: the plan lives and moves with every change, and the inspection is an independent check whose quality is measured, not the client's acceptance. The spec is the plan, and it is binding. Agents build to it, and a reviewer checks the result against it. Architecture is the whole design: what the project's concepts are and mean, how they relate, its interfaces, structure, rules and way of working. Architect is a role that the owner and an agent fill together: the agent drafts, analyses and proposes, and the owner decides.

**The parts.**
- People maintain one authoritative place per project: the spec. It is a typed, declarative specification language in Markdown: documents with a type, ordinary links, and types from packages. It states what is fixed about the project: its concepts, rules, boundaries, interfaces, structure and way of working.
- Code, tests, configuration and documentation are implementations of the spec. Where they differ from it, the spec is right.
- A deterministic tool, the `markitect` program, checks the spec, keeps a state of which artifacts realize which document and when that was last confirmed, and computes what is to verify. Jobs realize the spec, check the result and measure how well the checking works.

Markitect expects people's work to move from writing code to architecture: designing, defining and modeling. What matters in code matters there too: consistency, cleanliness, an overview of the project without reading its code, and knowing what should be as against what is.

> **Humans architect the system of work. AI performs the work. Markitect keeps them aligned.**

**What it builds on: proven practice.**
- **Declared state.** Terraform makes infrastructure match a configuration, keeps a state of what exists, shows a plan of what would change, and imports or reverts drift. The spec is the configuration, the state maps artifacts to documents, and `markitect status` is the plan.
- **Requirements traceability.** Tools such as IBM DOORS and Polarion keep a traceability matrix that links requirements to what realizes them, in both directions, and mark a link suspect when either end changes; DO-178C, ISO 26262 and IEC 62304 require such traceability. In Markitect the state is the matrix, `status` lists what changed and what is suspect, and a reviewer's verdict clears it.
- **Quality management.** Good requirements are judged by the characteristics of ISO/IEC/IEEE 29148, nonconformities are handled by corrective actions as ISO 9001 describes, and a testing laboratory earns trust with control samples. Markitect's spec, reports and checks follow them.

**What is new.**
- The traceability is plain text in the repository, beside the code: Markdown that people and agents read directly, checked deterministically, with no database and no separate tool to keep in step.
- AI does not take everything off the developer's hands, but it makes affordable what ordinary software projects dropped because of the manual effort and cost: a traceability matrix, independent checks derived from the specification, and control samples of the checking itself.
- A verdict holds until something it rests on changes, so a change puts back on the list only what it reaches.

**The approach: lighter and simpler.** Markitect works with Claude Code's own features rather than against them: a plugin, skills, subagents, scheduled and cloud tasks, and CI. The tool starts no agent, steers no work and uses no Git. The simpler version is the more robust one, and simplicity is earned through insight ([DEC-032](decisions.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

**The honest limit.** "Complete" means complete for the links. `status` follows a document's own text, its artifacts and evidence, its type and the documents linked to it in either direction, one step deep; it does not read the code's imports. Prose that names a document without linking it is a gap: a change to that document does not reach the one that names it. A signal shows such mentions, and spot checks and full reviews are the net. An empty `status` says that every document was confirmed at its current state, not that every verdict is right.

## Why Markitect exists

**Agents build the software.** A coding agent works like a capable, fast colleague. The result depends on how well it was told what is wanted, what matters and what belongs to the task. Like any colleague, it can forget a step. Much software is now built through such agents alone: the editor stays closed, code is skimmed at most, and pull requests are reviewed by another AI and seldom read.

**Two things were lost.** Writing and reviewing the code oneself brought two things at no extra cost:

- **Trust.** The author knew that the code reflected what was meant.
- **Overview.** The author knew the project's state, how it works and how it is organized.

Reading everything again would bring both back, and it would make people the bottleneck at the speed agents change a project. People's time belongs to ideas, concepts, organization and delegation: the work of a manager.

**Today's substitutes have no measure.** People ask the agent how something works. They start separate sessions to look for bugs, to judge the architecture, to check that everything fits, and after a fix to check that nothing else broke. Each of these passes looks from one angle, and each does find what the implementing session missed. Two things limit them: "everything fits" has nothing to fit against, and a pass runs only when someone thinks of it. So projects add rules, processes and skills. These live in agent instructions and documentation. They are more artifacts to remember, to maintain and to keep uniform, and more for an agent to overlook. Consolidating them with an agent works on a small project, scales badly, and leaves nothing that shows the result is consistent.

**A standing statement of what should be is missing.** A prompt, a user story and most specs in spec-driven development describe one change or one part of the project. They age once the change is made, and nobody maintains them. People work from their conversations with the agent, and nothing in those is settled for good. At no moment is it written down what the project as a whole should be. The project stays nebulous, without a clear outline, and what exists has nothing to converge on. An agent cannot tell how far the project has moved from what was wanted, cannot find deviations that crept in earlier, and cannot tell a bug from intended behavior. It guesses, and each guess makes the result less reliable. Getting what one wants takes knowing what one wants and saying it.

**The spec is that statement.** It is the one place where what the project should be is written down, and what is written there is binding. The spec is what a person works on: they decide what should be, and an agent writes it down with them. Code, tests, configuration and documentation are the agents' work: a reviewer confirms each document, and counter-checks and control samples measure that checking. The aim is work that nobody has to watch and nobody signs off. Today such approvals are mostly nodded through unread, which helps nobody; trust comes instead from facts that can turn out negative without anyone reading. A question an agent should not settle for itself shows that the spec leaves out or contradicts something that matters. It becomes a report, and where it needs a new decision, the person chooses between prepared proposals.

Trust and overview are rebuilt on the spec:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | `status` computes from the spec and the state what is changed, suspect or unexplained, like a checklist. Surgeons use checklists for the same reason: they know every step and can still skip one |
| Was it understood correctly? | A fresh reviewer confirms each document with evidence. Counter-checks, derived from the spec alone before anything is built, catch a misreading that builder and reviewer share, and control samples measure how much the reviewer finds |
| Does the spec say what was meant? | The person answers for the spec and develops it in conversation with an agent. Use shows the rest: every finding names the property it violates, "correct" among them, and new decisions come to the person as a choice. `check` finds structural errors, and readers and competency questions find contradictions, gaps and unclear statements |
| How does the project stand, and what changed? | The tool shows, and Claude explains with a citation for each statement: the briefing, explanations from the spec, and what changed since the person last looked. Jobs settle what follows from earlier decisions, so the spec changes without the person too, and the comparison with their last look is the core |

[Where trust comes from](#where-trust-comes-from) lists the mechanisms, and [Expected outcomes](#expected-outcomes) says what is still to be shown.

## Founding intent

**The problem.** A project built almost entirely with coding agents, documented in Markdown with rules, skills and agent instructions, drifted away from its code. It had explicit Core, Module and Product boundaries, and a specified working method, review policies and skills per module. Code, documentation and the agents' skills repeatedly contradicted one another; changes missed rules, every consistency run took hours and found more, and nobody could tell any more which rules existed, what was defined and what had entered the project silently.

**The whole project, as far as it is fixed.** The spec covers the whole project: which modules and deployments exist, what a concept such as a book is and what can be done with it, the rules and the way of working. It states what is fixed, what someone relies on or what was decided, such as data shapes and the answer to a refused call, and leaves loops and data structures to the agents ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). The concept says [what belongs in it](concept/what-goes-in-a-spec.md) and what happens where it leaves something open.

**One source, and why it is documentation.** A project is many artifacts that work together: code, tests, configuration and documentation. They drift apart. When two of them contradict each other, nothing says which is right, so the agent guesses or a person brings them back in step. Software solves this with a single source of truth and gives it up only where a system must scale across machines. A project's description never has to. Code could be that source: it compiles, it can be tested and it is exact. Code cannot state everything, though. A deployment or a way of working needs documentation, Dockerfiles and agent rules beside the code, and the detail of code makes it rigid. Documentation, kept as one source, can state everything. It lacks a compiler and tests, and it tends to say the same thing in many places.

**Code is a description.** Code has always been a precise description for an extremely limited but diligent executor. With AI the executor is intelligent, but it still has to be told what is meant and what holds; otherwise it guesses, and may do what nobody wanted. So documentation takes the place of code: it is what people work on. Markitect makes its authors express themselves cleanly: define what they mean and what holds, and connect those statements meaningfully.

**The answer is a typed, declarative specification language.** Whether documentation is consistent shows only at run time: a contradiction surfaces when an agent or a person stumbles over it while working. Dynamically typed programming languages have the same problem. Statically typed ones such as Go and C# are valued because a compiler reports such errors early, and the author has to say explicitly what is meant. So the spec gets a small type system of its own: a type says which sections its documents have, and a tool checks titles, types, sections and links. And like a declarative language, the spec says what should hold and leaves how to an executor; here the executor is an agent, so the result is checked, not guaranteed.

| In a typed, declarative language | In a Markitect spec |
| --- | --- |
| type, class | type: a document whose sections are the template |
| object, declaration | document |
| the body | the prose of a section: what holds, which agents realize |
| reference | an ordinary Markdown link |
| standard library | packages, copied into the spec, which owns its copy |
| type checker, `validate` | `markitect check`: structure only, never meaning |
| `plan` against the state | `markitect status` |
| test cache | the state in `.markitect/`, which binds each verdict to what it rests on |

- The core stays small. Projects define their own types and so form their own vocabulary; packages supply common ones, so that no project reinvents them.
- The tool checks structure. Meaning is judged by agents: the reviewer judges the artifacts against the spec, readers judge the spec itself, and counter-checks test the artifacts against what the spec alone says.
- People decide intent and answer for the spec. Agents write it with them in conversation, and ask first when something is open.

**Design against drift.** A check sees structure. It cannot see that the same fact stands in several places, and such places move apart with every change until it is unclear again what should be. So the spec has a writing rule: every fact stands in one place, and everywhere else links to it. A good spec is necessary, complete, consistent, unambiguous and verifiable, and structured, after the characteristics of good requirements in ISO/IEC/IEEE 29148 ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). Clean code holds that good code reads like well-written prose. Markitect asks the reverse: prose that is built like clean code. Results of Markitect are only as good as the spec they use.

**What the spec is for.** The typed spec is the foundation, and applying it is the purpose. One source settles which side is right when the spec and the project differ. Making them agree takes reconciliation, as Terraform does it with a declared state: compute what a change reaches, realize it, confirm it, repeat. Each pass sees from one angle, so Markitect keeps the kinds of job apart, each with its own question: build, review, cleanup, improve, bug hunt, and reading the spec. With the spec as their measure, they run on a schedule, without waiting for someone to think of them. No single pass has to find everything: a later one can catch what an earlier one missed, and trust grows as the project converges on its spec and the measurements show that the checking works.

## Where trust comes from

Markitect does not try to make agents infallible. It changes the structure they work in, and measures it ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice), [DEC-125](decisions.md#dec-125-trust-is-measured-not-granted)). Each mechanism below replaces something that today's practice leaves to chance; [the concept](concept/trust.md) says how firm each one is.

| Markitect mechanism | What it replaces in today's practice |
|---|---|
| **A list computed from the spec.** `status` lists every document whose text, artifacts, evidence, type or linked documents changed since a reviewer confirmed it, and every file that is mapped to no document. | An agent deciding from memory which files might be relevant |
| **Roles as separate steps.** Each role runs as a step of its own, and the tool checks every change against a write table read from the target branch: the builder writes artifacts, the counter-checker its checks, the reviewer only verdicts, the architect the spec. | One agent that writes, judges and records its own work |
| **Independent review with fresh context.** The reviewer takes its list from `status`, reads each document, what it links to and the artifacts, runs the tests and the counter-checks, and treats what the builder tells it as claims to check. | Self-review in the same thread, or one generic pull-request review |
| **Checks derived from the spec alone.** A counter-checker writes tests or questions from the spec before anything is built, and the builder may not change them. Where a rule names modules, they are architecture tests. | Tests written by the same agent that wrote the code |
| **Measured checking.** Control samples plant known deviations and give the reviewer's hit rate; spot checks check confirmed documents again, and a contradiction on an unchanged document is a fact. | Trusting a review because it said everything is fine |
| **One landing rule.** A branch lands only when `status` shows nothing new, the tests and counter-checks pass and every step stayed in its write area. In CI this is a fact no agent can skip. | Approving a pull request because its pipeline is green |

Two claims stay apart ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)): *nothing in the spec's scope is overlooked* can be computed for links and the mapping of every file; *everything is realized correctly* can only be supported by evidence. The tests a builder writes belong to the realization and can be as wrong as the code; counter-checks come from the spec alone.

## Who does what

| Role | Responsibility |
|---|---|
| People | Decide what should exist and why; choose concepts, architecture, boundaries, rules and processes; make the new decisions they have not handed over, as a choice between prepared proposals; answer for what the spec means. |
| Markitect | Check the spec, map every file, say which documents are changed or suspect, bind each verdict to the state the reviewer read, compute the factory's figures, and check each step of a job against the write table. It starts no agent, steers no work and uses no Git. |
| Agents | Write the spec with people and settle what follows from earlier decisions, realize the spec, derive counter-checks, confirm with evidence, plant control samples, and report what needs a change. An agent's report is not a person's approval. |

A real change of intent changes the spec first. When intent is unchanged, a bug or a stale file is repaired against the same spec; no artificial spec edit is needed. The target human role is specification and engineering judgment, not being a permanent agent scheduler, prompt supervisor or reader of every routine diff. Whether that work can actually be reduced is what Markitect has to show.

## Expected outcomes

Target outcomes, each to be validated; none is an established result.

- Nothing in the spec's scope is silently forgotten: what is to verify comes from the spec and the state, not from an agent's memory. This covers links; dependencies in code are narrowed by architecture tests where the spec's rules name modules, and unlinked mentions are known gaps.
- Rules are applied where they matter.
- The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable.
- Software becomes understandable: the spec states what the system does and should do, and people can reason over it without reading the code.
- Order pays off: structure and careful definition are rewarded instead of turning into endless cleanup.
- People concentrate on architecture. Their attention moves from code diffs to the decisions only they can make, while the code reliably reflects the spec, and the trust rests on measured checking rather than on reading.

The intended end state is **spec → AI realizes and independently checks it, and the checking is measured → people rely on the result without reviewing code diffs**. This is a target to earn through evidence, not a claim that Markitect has achieved it.

## Assumptions

Why the method should help, and what would show that it does not.

| Assumption | What would challenge it |
|---|---|
| An agent that is told fully what is wanted, what matters and what belongs to the task builds what was meant | Results match the intent no better with a complete spec than with a short prompt |
| A person can answer for a spec written with an agent, from the conversations, the briefing and the reports from use | Wrong statements stay unnoticed in the spec, are realized faithfully and are found only in the product |
| One authoritative spec with maintained links reduces the mistakes of keeping things in step | Contradictions still pile up, or keeping the spec and the state costs as much as the consolidation it replaces |
| Typing catches useful structural errors before agents work | Most consequential failures pass `check` and still need the same late investigation |
| A list computed from links keeps continuing obligations in view | What a change really affects falls outside the list, or the list grows to nearly the whole spec on every change |
| A reviewer with fresh context finds mistakes the builder misses, even on the same model | Control samples show a low hit rate, or reviewers keep accepting the same omissions |
| Checks derived from the spec alone catch the misreadings that builder and reviewer share | Counter-checks pass where all three read the spec the same wrong way, and the error shows only in use |
| Forgetting, rather than misunderstanding, is the main problem | Most failures turn out to be wrong readings of a document the agent did see |
| What works in human teams and testing laboratories transfers to agents: separate roles, clear responsibility and control samples | Separate roles add hand-offs and cost without catching more |
| The state, its evidence and the measured checking remove the need to read every routine diff | People still need to review every diff, cannot tell what is unverified, or trade that work for equally costly supervision of the spec |

## How to judge it

- Judge Markitect's trust against today's practice, not against a compiler's guarantees ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice)).
- "Compiler" names a borrowed idea. The spec's structure is checked deterministically; realization by agents is not deterministic ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Delegated realization is central to the product. It is not an optional layer on top of a validator ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)).
- The aim is one simple version that works reliably, shown first in a lab test and then proven on a project that is not the owner's ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)).

## Common misreadings

Check an assessment against this table before treating one of these as a finding.

| Misreading | Correct reading |
|---|---|
| Markitect is a linter or context tool for AI instructions such as rules, skills and `AGENTS.md`. | That is where the project began. With Markitect, people architect the whole project in Markdown, and agents realize it. Agent instructions are one kind of realization among many. |
| Markitect is a tool for architecture documentation. | The spec is the plan the project is built from, and it is binding: agents realize it, and a reviewer verifies the result against it. Architecture covers the domain too: what a concept is, what can be done with it and how things relate. |
| Markitect is a programming language. | Nothing in the spec is executed. It is a typed, declarative specification language, as TLA+ and Alloy are specification languages: its structure is checked, and agents realize its content ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language), [DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). People's work is architecture, and agents write the code. |
| Markitect is a new, unproven formalism. | It builds on proven practice: declared state as Terraform keeps it, requirements traceability with suspect links, and quality management with corrective actions and control samples. What is new is that the traceability is plain text checked deterministically, and that agents realize the spec, check it and are measured doing so. |
| The deterministic parts are the product, and agent work is scope creep. | Delegated realization is the purpose, and the tool supports it ([DEC-001](decisions.md#dec-001-delegated-realization-is-the-purpose)). |
| Markitect orchestrates or steers the agents. | The tool checks, computes and binds verdicts. Which job runs when, and every Git step, belong to the runner ([DEC-096](decisions.md#dec-096-the-tool-computes-and-checks-a-runner-acts)). |
| Markitect needs API keys or a platform of its own. | Jobs run on the user's Claude subscription, in their own session, as a scheduled or cloud task, or in CI with a token of that subscription; API keys are never used. The tool uses no Git, and the runner makes every Git step ([DEC-134](decisions.md#dec-134-jobs-run-on-the-users-subscription-with-or-without-a-session), [DEC-097](decisions.md#dec-097-markitect-uses-no-git)). |
| The owner approves every change. | There are no approvals. The owner sets the intent in conversations and makes new decisions as a choice; everything else lands by the landing rule ([DEC-125](decisions.md#dec-125-trust-is-measured-not-granted), [DEC-127](decisions.md#dec-127-one-landing-rule-and-a-spec-change-lands-with-its-realization)). |
| An empty `markitect status` proves the project correct. | It means that the spec is correct in itself and every document was confirmed at its current state. That is complete for the links only, and a verdict is evidence a reviewer gave, not a proof ([concept, §7](concept/trust.md)). |
| Without compiler-grade guarantees the idea fails. | The compiler is a borrowed idea, and the trust bar is today's practice ([DEC-002](decisions.md#dec-002-trust-is-measured-against-todays-practice), [DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)). |
| The spec has to mirror the code. | It states only what is fixed; what nobody relies on stays free, and one document may have many artifacts ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface), [DEC-124](decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)). |
| Green agent-written tests prove the realization. | The tests a builder writes are realizations too. Counter-checks come from the spec alone, before the code, and the builder may not change them ([DEC-125](decisions.md#dec-125-trust-is-measured-not-granted)). |
| Removing routine human code review removes the four-eyes principle. | A fresh reviewer, checks derived from the spec alone, measured checking and a landing rule outside every agent are the means to earn that trust. People keep the intent and the new decisions. |

## What Markitect leaves to people

Markitect does not decide which architecture is right, which business goal matters, whether a rule is desirable, whether evidence is representative or which risk to accept. It does not understand arbitrary source code, guarantee correctness or the absence of drift, or replace human product and architecture judgment. People keep those decisions and may hand routine kinds of decision over explicitly; machine evidence states only what its inputs and checks establish.

## Origin of the name

**Markitect = Markdown + Architect.** The project began with architecture and process knowledge in readable documentation, rules, skills and agent instructions. The spec is Markdown, typed by its own types. The name says what a person does with Markitect: architect their project in Markdown. The origin does not limit the product to agent instructions.
