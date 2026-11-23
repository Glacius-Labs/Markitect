# Decisions

The decisions that hold for Markitect, grouped by topic, and the questions still open. The [concept](concept/README.md) describes the product they add up to, and the [vision](vision.md) says why.

- This page lists only what holds. A decision changes only through a new decision: it takes the next free number, and the entries it replaces leave the page in the same change. Git keeps every earlier wording.
- Do not reopen a decision without new evidence. With such evidence, add an open question that cites it and leave the decision as it is.
- An entry keeps its ID, and the IDs it absorbed keep an anchor on it, unless the anchor names a retired design.
- If another page differs from a decision, the decision holds: correct the page.

## Purpose

<a id="dec-101-people-architect-their-project-in-markdown"></a>

### DEC-119 People architect their project in a typed, declarative specification language

- People design, define and model their project in Markdown: what its concepts are and mean, how they relate, its interfaces, structure, rules and way of working.
- The spec is a typed, declarative specification language in Markdown. It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it, as Terraform makes infrastructure match its configuration.
- It is a specification language, as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by a tool; its content is prose that an agent interprets. It is never a programming language: there the result follows from the text, and here only the structure is checked ([DEC-138](#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation takes the place of code as what people work on. What matters in code stays as relevant there: consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- An agent can be wrong, so its results are checked ([DEC-125](#dec-125-trust-is-measured-not-granted)).
- Reopen when readers repeatedly take the image to mean that Markitect executes the spec.

### DEC-154 Markitect is a language, its default and a tool that only reads the spec

- Markitect is three things: the language in which a spec is written and built up ([DEC-121](#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)); the default that ships with it, the packages `core`, `software` and `company` and the templates `project`, `product`, `architecture`, `software` and `company`, with the reference values and the question catalog of each package; and the tool `markitect`, which helps working with the spec.
- The tool checks the spec, analyses it and helps reading it. It never writes and keeps no state. Anything more is a separate tool or a script of a project. Its commands are flat, and each does one thing on its own: `check`, `show`, `table`, `topic`, `questions`, `diff`, `version` and `licenses`.
- `diff OLD [NEW]` names what is new, gone or changed in the spec of NEW against the project OLD, with a diff for what changed, and nothing else: the plain comparison of two states of a spec, under a name that says what it does, where it was `compare` before.
- Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run. Commands are named after what they do.
- A project is the nearest folder upward that holds the file `.markitect/spec`, whose one line names the spec folder. A person writes that file, and nothing creates it. The tool reads nothing else from `.markitect/`.
- The packages and templates are plain files in the repository, which a person copies into a spec folder; the spec owns its copy. The reference values and the question catalogs are read from the tool's version and never copied.
- The way a project works is not part of Markitect. Findings, briefings, meetings, stories, features, reports, reviewing, accepting, where the work lies and how planning goes belong to a project's way of working: the project describes it in its own spec, with general types, and skills, agents and scripts realize that spec.
- A change to a spec is seen through the ordinary diff of the repository, like every change.
- Nothing of Markitect stands in an artifact or in the spec: no comment in code, no front matter. Deleting `.markitect/spec` and the tool leaves an ordinary repository, with its documentation in Markdown.
- So the tool has no `status`, `confirm`, `move`, `init`, `apply` or `new`, and no state, history, verdicts, fingerprints, evidence, counter-checks, change signals of a history, lock, pinned version, ignore file or work folders. The `factory` package, the `factory` template and the skills of the standard leave the library until a way of working is designed anew as part of the default.
- The plugin is unchanged: its launcher fetches the tool's release and starts it ([DEC-135](#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language)).
- Reopen when a project cannot tell from the spec and the repository's diff what a change reaches without a stored state; or when setting up a project, by hand or by a skill, proves too error-prone for a tool that never writes.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool checks the spec, so that people and agents work from a spec that is sound in its structure; how the project realizes and verifies it is the project's own way of working ([DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).
- Delegated realization is the purpose the spec serves: the spec is there to be built from, not to be a validator's input for its own sake. Markitect makes it possible with the language, the default and the tool, and ships none of the realization; a project's way of working does it.
- Judge Markitect as a way of developing software, not as a linter or a context tool.

<a id="dec-002-trust-is-measured-against-todays-practice"></a><a id="cpt-001-trust-comes-from-structure"></a><a id="cpt-001-federalism"></a>

### DEC-137 Trust is measured against today's practice

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that, not compiler-grade guarantees.
- Trust comes from structure, not from infallible agents: one binding spec, checked by a tool, against which every result can be checked. Measuring how well that check works is the work of a project's way of working ([DEC-125](#dec-125-trust-is-measured-not-granted)).
- Claims read "more reliable than the baseline", never "correct". Use the smallest way of working that earns the trust: more report fields or completeness declarations do not by themselves make work more correct.

<a id="dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart"></a><a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a><a id="cpt-002-tests-are-the-projects-and-they-are-realizations-too"></a><a id="cpt-002-tests-are-realizations-too"></a>

### DEC-138 The compiler is a borrowed idea, and two claims stay apart

- The spec's structure is checked deterministically: titles, types, sections and links. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *The spec is sound in its structure* is checked by the tool. *The project realizes the spec correctly* can only be supported by evidence, which the project's way of working brings, such as an independent review and the project's tests.
- The tests a builder writes are realizations of the spec too, so a green one is evidence, not proof.

<a id="dec-009-how-markitect-is-judged"></a><a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a>

### DEC-120 How Markitect is judged, and when it is proven

- The aim is one clean and simple version that works reliably, not to prove that one version beats another.
- The long-term questions: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to architecture?
- The method is judged apart from any runtime: a failure of Markitect's own tool or default is a product finding, a failure of the environment it runs in a limit of the measurement.
- Being proven unlocks the conduits that wait for it ([DEC-136](#dec-136-later-ideas-wait-as-conduits-with-a-trigger)). When Markitect counts as proven is open ([OQ-015](#open-questions)). The [concept](concept/what-is-shown.md) says what is shown and what is not.

<a id="dec-032-quality-simplicity-and-stability-over-effort-and-compatibility"></a><a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a><a id="dec-038-learn-through-complete-workflows"></a>

### DEC-139 Quality, simplicity and stability over effort and compatibility

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions while Markitect is early. Compatibility counts from the first project run by people who do not develop Markitect. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure.
- Simpler is more robust, and simplicity is earned through insight. A project's way of working uses the coding agents' own features, skills and `AGENTS.md`, and Markitect builds no machinery around them.
- A substantial redesign names the new observation behind it and the result that would show it is better, and asks first whether a mechanism is needed at all or something simpler would do.

<a id="dec-098-one-concept-describes-the-product"></a>

### DEC-140 One concept describes the product

- The [concept](concept/README.md) is the single description of the product and the target that Main builds: the language, the default that ships with it, and the tool that checks the spec, analyses it and helps reading it ([DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).
- Earlier concepts are kept on archive branches ([history](history.md)), not on Main.

## The spec

### DEC-121 A spec is plain Markdown with types, packages and templates

- A document is a Markdown file with a title and `##` sections, without YAML front matter. It is known by its path below the spec folder. Folders are free, so a spec may be ordered by feature. File names use lowercase letters, digits and hyphens; the readable name is the title.
- A type is a document whose `##` sections are the template of its documents; the text under each says what belongs there, and `(optional)` marks a section that is not required. A document names its type under the title, `Type: [Service](../types/service.md)`. Types and documents relate as classes and objects do. A value, such as a state or an owner, stands in the first line of its section; there are no lines in the head but the type line.
- Links are ordinary relative Markdown links, which work on GitHub, in editors and in note tools alike. A link is no import, and no section means anything special to the tool.
- There is no module layer: no modules declared by an anchor, no rules on the direction of links, no visibility and no binding by place. What belongs together shares a folder. A part of the spec is a folder, and its README is its table of contents.
- Types come from packages, folders of types such as `core`, `software` and `company`. Structure comes from templates, folders of documents with empty sections, which can be copied in at any level: a whole project, one part of the domain, or a field such as fire protection. Each type says before its sections what it is and when to use it, and under each section what belongs there, with an example. A spec owns its copy of each package and may change it.
- `check` checks titles, types, missing or empty sections and links that point to no document. What the language does not catch, contradictions between values, gaps and wrong values, a reader finds. Value forms and check rules are left out on purpose; something new joins only when a real project needs it.
- An agent writes the structure, sections and links; it is not derived from the prose.
- Writing rule, not checked: every fact stands in exactly one place, and everywhere else links to it.
- An ontology language, a modelling language and front matter with fields were weighed. None was chosen: Markdown is what people and agents already write.
- The [language](concept/the-language.md) page describes it.
- Reopen when a project repeatedly shows errors that a value form or check rule would have caught and the reviewer misses; when a software project shows a layer break that a reviewer misses and direction rules would have caught, or changes measurably reach wider without modules; when projects change their package copies so much that taking a newer package from the library is no longer possible and that does harm; or when an audit shows that agents' documents contradict their own prose more often than before.

<a id="dec-079-the-spec-is-as-explicit-as-good-documentation"></a><a id="dec-063-the-spec-is-the-authority-and-its-defects-go-to-the-owner"></a>

### DEC-122 The spec declares what is fixed, like an interface

- The spec declares everything that is fixed about what it describes. Fixed is what someone relies on or what was decided. What could differ without bothering anyone is free and stays out, so the spec never becomes a transcription of the code. An interface is the image: it says what holds and hides how it is done. Decisions about the inside, such as modules or technology, belong in it too.
- What belongs in it follows from what it describes, not from who reads it. For software the binding part is what whoever relies on it depends on: inputs and their validation, data shapes, the answer to a refused call, stored data. Detail one can observe but not reasonably depend on, such as message wording or identifier formats, is free unless the project fixes it.
- The spec describes the end state, not the way there, and not the work that changes it: a backlog, a plan, tasks towards a goal or a project risk stay out. Running processes and tasks of an operation, the way of working of a project and goals with a date belong in, since they say what should hold. Where the work itself lies and in which form is part of that way of working ([DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).
- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines, configuration and skills alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- A software project's spec starts from three of the five templates: `project`, with purpose, context, guiding principles and quality; `product`, with capabilities, what the product can do for its users, each of the type Capability of the `software` package, structured but with no fixed chapters; and `architecture`, with building blocks and the rules for building. A business starts from the `company` template instead. The `software` template, with Purpose, Context, Domain, Interfaces, Quality and Architecture, is a checklist for `project`, not an outline. The software package holds Assumptions in Context, with who ensures each and what is therefore not built; the answer the caller gets for each condition under which an Operation is refused; the nine characteristics of ISO/IEC 25010 as optional sections of Quality, which a template lists empty so that `check` names each until it says what it asks or why nothing; and Deployment in Architecture. The Decision type has an optional section "Reopen when".
- A business case, scenarios and operational tasks wait until a project needs them.
- The [content](concept/what-goes-in-a-spec.md) page describes it.
- Reopen when a builder or reviewer in a software project misses something about operation that the spec should have said; when a section of the software package stays empty or says only "nothing" in real projects, which removes it; or when plans or task lists appear in a spec and go stale after every change.

### DEC-123 A good spec has five properties and shows them in four sources

- A good spec is, after the characteristics of good requirements in ISO/IEC/IEEE 29148: **necessary**, it states only what is fixed; **complete**, nothing fixed is missing; **consistent**, no two statements exclude each other; **unambiguous and verifiable**, two readers understand the same and a checker can say yes or no, with every statement as strong as possible; and **structured**, one thing per document, each fact on its level, and what has facts of its own in a document of its own.
- Coupling, cohesion, abstractness and link density are no measures of a spec. There are no fixed thresholds.
- Four sources show how good a spec is. The spec itself: checked by `check`, shown by a topic view and signals, read by a person or agent against the five properties, and questioned with competency questions answered only from the spec with a citation. Its changes, as the repository's history shows them: six signals, a wide change, documents that change together without a link, links dragged along, a hotspot, back and forth, and weakening, with mechanical changes left out; the tool computes none of them. What is built from it. And the questions it raises.
- Every problem found names its place and the property it violates: one of the five, or **correct**, when a person sees a result and says it was not meant so. Only use shows correctness.
- Every number is shown against the reference value of its type, which ships with the package, and against the rest of the project. Reference values come from proven specs; a proven spec is one that the people who answer for it name as a reference.
- Question catalogs ship with packages and templates. A project's own questions lie beside the spec, not in it. An answered question from use becomes a fixed question, and a general one can move into the package.
- The [quality](concept/is-the-spec-good.md) page describes it.
- Reopen when problems found repeatedly show something bad that falls under none of the five properties or correctness, the property is not assigned reliably, or nobody reads the evaluation; when a signal of the spec itself shows almost only false alarms on real specs; when a change signal never points at a real problem over a series of changes on a real project; when reference values show almost only false alarms, or differ so much between projects that none carries; or when answers stay unreliable despite citations, or a catalog grows without ever finding anything.

## Trust

### DEC-125 Trust is measured, not granted

- Markitect aims at trust without approvals: no trust comes from a person looking through the work. A mechanism counts only when it delivers a fact that can turn out negative without anyone reading: a run, a figure, a difference between artifact and spec. Another agent's opinion is no such fact. The image is a testing laboratory.
- The question has two halves. Whether the artifacts match the spec can be secured without people. Whether the spec says what is meant shows only in use.
- Trust comes from convergence over time, not from a perfect run: a single run may leave errors, and the measure is that the differences between artifacts and spec shrink.
- Which mechanisms earn that trust in a project, and how they are measured, is part of its way of working, which the project states in its spec ([DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).
- Reopen when problems found in use repeatedly show late that the spec said something other than was meant, which gives the second half a building block of its own, never an approval.

## Overview

### DEC-131 The tool shows, Claude explains

- There is no user interface for now. The spec is readable, the tool runs natively, and the agent describes what the tool shows. A user interface comes once Markitect is proven ([DEC-120](#dec-120-how-markitect-is-judged-and-when-it-is-proven)). A dashboard or a handbook that a project wants is an artifact of the project, built from its spec, not a user interface of Markitect.
- What people should not have to take on an agent's word does not come from an agent. The tool, and the project's version control where changes are concerned, show; the agent explains, with a citation for each statement, and says what it did not cover.
- An explanation of the whole, a part, a document or a question comes only from the spec, with citations, and otherwise says "not stated".
- Views are generic and come from the language: the topic view, a table per type, a document with its links, and `diff`. They work for every spec. Views per package come with a user interface.
- The [tool](concept/the-tool.md) page describes it.
- Reopen, before Markitect is proven, when people repeatedly ask for the same picture that the chat cannot show, or a user needs Markitect without an agent; when explanations repeatedly say something wrong despite citations, so the tool shows more; or when explanations of a package repeatedly need the same picture, which the package then ships as a view.

## The tool and its delivery

<a id="dec-097-markitect-uses-no-git"></a>

### DEC-097 Markitect neither needs nor knows version control

- Markitect neither needs nor knows Git or any other version control. The tool makes no repository, commits nothing and reads no history; whoever works on the project takes every step of its version control.
- The tool reads folders: `.markitect/spec` and the spec folder it names, as they lie, and nothing else of the project.
- A change to a spec is seen through the repository's ordinary diff. `diff` compares two folders, however their owner made the older one ([DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).

### DEC-135 The tool stays in Go and is rebuilt for the language

- The tool is rebuilt for the language of [DEC-121](#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates), in Go. It must be one program without a runtime that runs on Windows, Linux and macOS and that the plugin and CI download simply, and the release path, checksums and conventions exist.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. It needs no key and no account.
- A shared language for a user interface and an agent SDK does not count: the tool starts no agent and there is no user interface.
- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and a provenance file as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused. The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.
- The plugin only fetches the tool: its launcher fetches the release of the plugin's own version, which is the tool's version, and starts it.
- Reopen when a user interface comes and needs the same code as the tool.

## Outlook

<a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-136 Later ideas wait as conduits with a trigger

- Each later idea stays unbuilt and has a trigger, the event that brings it in, as an architect lays conduits in a shell for cables that may come later. No decision in force blocks one of them.
- The [roadmap](roadmap.md#conduits) keeps them in one list: a user interface with other clients, telemetry for reference values, and variants per context. Detailed planning waits until its trigger comes.
- Gamification, Kubernetes or etcd as a base, and an offline classifier of claims and evidence are left out.
- Reopen when a trigger comes and a decision in force turns out to block its idea. Then that decision is reopened, not built around.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a><a id="dec-070-parallel-writers-where-useful"></a><a id="dec-077-linux-ci-run-on-request"></a><a id="dec-077-development-pace-and-linux-first-execution"></a>

### DEC-076 Markitect is developed as an ordinary software project

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one page per topic. Target projects keep their spec and `.markitect/spec`; product code and tests that handle them use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages.
- Sessions work in parallel where their files do not overlap; shared files have one writer at a time.
- CI runs on Linux only, with the race detector, started on request. Markitect's own tests run on Linux; for users, Windows is a main path ([DEC-135](#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language)).
- [CONTRIBUTING](../CONTRIBUTING.md) is the development process.

### DEC-069 Documentation is written in English

- All documentation on Main is in English.

<a id="dec-071-history-leaves-main-through-the-archive"></a><a id="dec-071-repository-structure-and-archive"></a>

### DEC-071 Main holds only the current state

- Main's documentation describes what holds today, as one concept, and quotes no one. Superseded designs, plans, decisions and dated records leave Main in the change that supersedes them; Git keeps them, and each earlier concept has an archive branch ([history](history.md)).
- A page that relies on a dated result says it in one sentence and links its record: the PR that reported it, or a file at a full commit SHA.

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-015 | When does Markitect count as proven? A user interface and telemetry for reference values wait for it. | [DEC-120](#dec-120-how-markitect-is-judged-and-when-it-is-proven), [DEC-154](#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec) |
