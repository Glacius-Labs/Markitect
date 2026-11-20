# Decisions

The decisions that hold for Markitect, grouped by topic, and the questions still open. The [concept](concept/README.md) describes the product they add up to, and the [vision](vision.md) says why.

- This page lists only what holds. A decision changes only through a new decision: it takes the next free number, and the entries it replaces leave the page in the same change. Git keeps every earlier wording.
- Only the product owner adds or changes a decision. An agent's proposal enters as an open question and stays one until the owner decides.
- Do not reopen a decision without new evidence. With such evidence, add an open question that cites it and leave the decision as it is. Each newer decision names in "Reopen when" the evidence that would reopen it.
- An entry keeps its ID, and the IDs it absorbed keep an anchor on it, unless the anchor names a retired design.
- If another page differs from a decision, the decision holds: correct the page.

## Purpose

<a id="dec-101-people-architect-their-project-in-markdown"></a>

### DEC-119 People architect their project in a typed, declarative specification language

- People design, define and model their project in Markdown: what its concepts are and mean, how they relate, its interfaces, structure, rules and way of working. Architect is a role that the owner and an agent fill together: the agent drafts, analyses and proposes, and the owner decides.
- The spec is a typed, declarative specification language in Markdown. It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it, as Terraform makes infrastructure match its configuration.
- It is a specification language, as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by a tool; its content is prose that an agent interprets. It is never a programming language: there the result follows from the text, and here only the structure is checked ([DEC-003](#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation takes the place of code as what people work on. What matters in code stays as relevant there: consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- An agent can be wrong, so every result is checked and recorded ([DEC-125](#dec-125-trust-is-measured-not-granted)).
- Reopen when readers repeatedly take the image to mean that Markitect executes the spec.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool computes and checks, so that the delegated work can be trusted.
- Judge Markitect as a way of developing software, not as a linter or a context tool. Removing delegated realization as scope creep contradicts this decision.

<a id="cpt-001-trust-comes-from-structure"></a><a id="cpt-001-federalism"></a>

### DEC-002 Trust is measured against today's practice

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that, not compiler-grade guarantees.
- Trust comes from structure, not from infallible agents: separate roles with bounded rights, a list computed from the spec rather than from memory, a check of every document against the spec, and measurements of how well that check works ([DEC-125](#dec-125-trust-is-measured-not-granted)). The [vision](vision.md#where-trust-comes-from) lists the mechanisms.
- Claims read "more reliable than the baseline", never "correct". Use the smallest process that earns the trust: more report fields or completeness declarations do not by themselves make work more correct. A control sample measures the checker and leaves the work as it is.

<a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a>

### DEC-003 The compiler is a borrowed idea, and two claims stay apart

- The spec's structure is checked deterministically: titles, types, sections and links. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *Nothing in the spec's scope is overlooked* can be computed from links and the mapping of every file; dependencies in code that no link states are a known gap. *Everything is realized correctly* can only be supported by evidence: a fresh reviewer, counter-checks, the project's tests and `check`.

<a id="dec-009-how-markitect-is-judged"></a><a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a>

### DEC-120 How Markitect is judged, and when it is proven

- The aim is one clean and simple version that works reliably, not to prove that one version beats another.
- The long-term questions: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to architecture?
- The method is judged apart from any runtime: a failure of Markitect's own tool is a product finding, a runner or environment failure a limit of the measurement.
- The lab test comes first: two clean passes in a row over the eight change steps of the 5× Makerspace project, every step landing by the landing rule and every hidden case passing.
- Markitect is proven when a project that is not the owner's runs four weeks in the loop with real work in CI, and four figures from the tool and Git hold:
  - **Work:** actions that change spec and code land every week.
  - **Convergence:** open reports and `status` do not grow over the weeks.
  - **Checking:** the hit rate of control samples does not fall, and spot checks do not contradict earlier verdicts more often.
  - **No rescue:** no landed action is reverted in Git except through a report.
- All four can turn out negative without anyone reading. Being proven unlocks the conduits that wait for it ([DEC-136](#dec-136-later-ideas-wait-as-conduits-with-a-trigger)). The [concept](concept/what-is-shown.md) says what is shown and what is not.
- Reopen when the figures hold but the outside owner no longer runs the project through Markitect.

<a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a><a id="dec-038-learn-through-complete-workflows"></a>

### DEC-032 Quality, simplicity and stability over effort and compatibility

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions while Markitect is early. Compatibility counts from the first project that is not the owner's. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure.
- Simpler is more robust, and simplicity is earned through insight. Markitect works with Claude Code's own features rather than building machinery around them.
- A substantial redesign names the new observation behind it and the result that would show it is better, and asks first whether a mechanism is needed at all or something simpler would do.

### DEC-098 One concept describes the product

- The [concept](concept/README.md) is the single description of the product and the target that Main builds: the spec, a tool that checks it and computes what is built and verified, the jobs that realize and check it, and the runners that start them.
- Earlier concepts are kept on archive branches ([history](history.md)), not on Main.

## The spec

### DEC-121 A spec is plain Markdown with types, packages and templates

- A document is a Markdown file with a title and `##` sections, without YAML front matter. It is known by its path below the spec folder. Folders are free, so a spec may be ordered by feature. File names use lowercase letters, digits and hyphens; the readable name is the title.
- A type is a document whose `##` sections are the template of its documents; the text under each says what belongs there, and `(optional)` marks a section that is not required. A document names its type under the title, `Type: [Service](../types/service.md)`. Types and documents relate as classes and objects do.
- Links are ordinary relative Markdown links, which work on GitHub, in editors and in note tools alike. A link is no import, and no section means anything special to the tool.
- There is no module layer: no Packages with anchors, no rules on the direction of links, no visibility and no binding by place. What belongs together shares a folder, and each area has a README of an area type whose sections are its table of contents.
- Types come from packages, folders of types such as `core`, `software` and `company`. Structure comes from templates, folders of documents with empty sections, which can be applied at any level: a whole project, one area of the domain, or a field such as fire protection. A spec owns its copy of each package, and `apply` never overwrites it.
- `check` checks titles, types, missing or empty sections and links that point to no document. What the language does not catch, contradictions between values, gaps and wrong values, a reader finds. Value forms and check rules are left out on purpose; something new joins only when a real project needs it.
- An agent writes the structure, sections and links; it is not derived from the prose.
- Writing rule, not checked: every fact stands in exactly one place, and everywhere else links to it.
- An ontology language, a modelling language and front matter with fields were weighed. None was chosen: Markdown is what people and agents already write, and the simplest form read like a handbook and caught everything mechanical.
- The [language](concept/the-language.md) page describes it.
- Reopen when a project repeatedly shows errors that a value form or check rule would have caught and the reviewer misses; when a software project shows a layer break that a reviewer misses and direction rules would have caught, or changes measurably reach wider without modules; when projects change their package copies so much that taking a newer package from the library is no longer possible and that does harm; or when an audit shows that agents' documents contradict their own prose more often than before.

<a id="dec-079-the-spec-is-as-explicit-as-good-documentation"></a>

### DEC-122 The spec declares what is fixed, like an interface

- The spec declares everything that is fixed about what it describes. Fixed is what someone relies on or what was decided. What could differ without bothering anyone is free and stays out, so the spec never becomes a transcription of the code. An interface is the image: it says what holds and hides how it is done. Decisions about the inside, such as modules or technology, belong in it too.
- What belongs in it follows from what it describes, not from who reads it. For software the binding part is what whoever relies on it depends on: inputs and their validation, data shapes, the answer to a refused call, stored data. Detail one can observe but not reasonably depend on, such as message wording or identifier formats, is free unless the project fixes it.
- The spec describes the end state, not the way there, and not the work that changes it: a backlog, a plan, tasks towards a goal or a project risk stay out. Running processes and tasks of an operation, the way of working of a project and goals with a date belong in, since they say what should hold.
- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines and configuration alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- The content model is the area type of a package. For software it is Purpose, Context, Domain, Interfaces, Quality and Architecture. The software package holds Assumptions in Context, with who ensures each and what is therefore not built; the answer the caller gets for each condition under which an Operation is refused; the nine characteristics of ISO/IEC 25010 as optional sections of Quality, which the template lists empty so that `check` names each until it says what it asks or why nothing; and Deployment in Architecture. The Decision type has an optional section "Reopen when".
- A business case, scenarios and operational tasks wait until a project needs them.
- The [content](concept/what-goes-in-a-spec.md) page describes it.
- Reopen when a builder or reviewer in a software project misses something about operation that the spec should have said; when a section of the software package stays empty or says only "nothing" in real projects, which removes it; or when plans or task lists appear in a spec and go stale after every change.

### DEC-123 A good spec has five properties and shows them in four sources

- A good spec is, after the characteristics of good requirements in ISO/IEC/IEEE 29148: **necessary**, it states only what is fixed; **complete**, nothing fixed is missing; **consistent**, no two statements exclude each other; **unambiguous and verifiable**, two readers understand the same and a checker can say yes or no, with every statement as strong as possible; and **structured**, one thing per document, each fact on its level, and what has facts of its own in a document of its own.
- Coupling, cohesion, abstractness and link density are no measures of a spec. There are no fixed thresholds.
- Four sources show how good a spec is. The spec itself: checked by `check`, shown by a topic view and signals, read by a person or agent against the five properties, and questioned with competency questions answered only from the spec with a citation. Its changes: six signals, a wide change, documents that change together without a link, links dragged along, a hotspot, back and forth, and weakening, with mechanical changes left out. What is built from it. And the questions it raises.
- Every finding names its place and the property it violates: one of the five, or **correct**, when the owner sees a result and says it was not meant so. Only use shows correctness.
- Every number is shown against the reference value of its type, which ships with the package, and against the rest of the project. Reference values come from proven specs; a proven spec is one the owner names as a reference.
- Question catalogs ship with packages and templates. A project's own questions lie beside the spec, not in it. An answered question from use becomes a fixed question, and a general one can move into the package.
- The [quality](concept/is-the-spec-good.md) page describes it.
- Reopen when findings repeatedly show something bad that falls under none of the five properties or correctness, or the property is not assigned reliably; when a signal or a reference value shows almost only false alarms on real specs, or reference values differ so much between projects that none carries; or when answers stay unreliable despite citations, or a catalog grows without ever finding anything.

## State and trust

<a id="dec-006-every-file-is-accounted-for"></a><a id="dec-006-every-file-is-covered-or-explicitly-ignored"></a><a id="dec-099-a-record-of-what-was-verified"></a>

### DEC-124 A state maps each document to its artifacts, and `status` is the plan

- Terraform is the image. The spec is the configuration, an interface; the artifacts are its implementations; the state in `.markitect/` is the matrix between them; `check` is `validate`, `status` is `plan`, and a build is `apply`. The unit is one document.
- The state holds one row per document: its artifacts, its counter-checks apart from them, its verdict and the hashes at its last confirmation. The reviewer names the artifacts when it confirms. Nothing of Markitect stands in an artifact or in the spec, so deleting Markitect leaves an ordinary repository.
- A change has two stages. A document is **changed** when its own text or one of its artifacts moved: a builder realizes it and a reviewer confirms it. It is **suspect** when a document it links to, or one that links to it, changed, as in DOORS: a reviewer judges whether the change means something here. A changed type makes its documents suspect. Links are ordinary links in both directions; "Applies to" is an ordinary section.
- Every file is mapped to a document or ignored, by `.gitignore` or by `.markitect/ignore`. Any other file is **unexplained** in `status`, which then is not empty and fails CI. A file whose document is gone is unexplained too.
- `check` checks only the spec. Everything that compares spec and artifacts is in `status`.
- There are three verdicts: `ok`, `fail` and `skip`. `mark` binds a verdict, with its artifacts and evidence, to the state `status` showed, and refuses when an input changed since.
- At each confirmation the history records, as hashes, which documents were changed at that moment, so that the change signals of [DEC-123](#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources) can be computed. What a statement said before comes from Git, read by an agent. `move` rewrites every link to a moved document, carries its row in the state, and records a mechanical event in the history.
- The tests a builder writes are realizations, so a green one is evidence, not proof.
- The [state](concept/the-state.md) page describes it.
- Reopen when reviewers repeatedly name a document's artifacts incompletely and it shows late, so the builder maps them again, in the same state; when suspect entries make the list so long that reviewers confirm them unread, so only links in certain sections count; when new documents are repeatedly not reached because a rule lists what it applies to, so the tool reads "Applies to"; when unexplained files are almost only files nobody wants to explain, which then go into the ignore file; or when a change signal needs more text than the hashes give and an agent cannot read it reliably from Git.

### DEC-125 Trust is measured, not granted

- There are no approvals, and no trust comes from a person looking through the work. A mechanism counts only when it delivers a fact that can turn out negative without anyone reading: a run, a figure, a deviation. Another agent's opinion is no such fact. The image is a testing laboratory.
- The question has two halves. Whether the artifacts match the spec is secured without people, by confirmation, control samples, counter-checks and use. Whether the spec says what is meant shows only in use, through reports.
- **Control samples.** A sampler plants known deviations from the spec in a throwaway copy of the artifacts. The reviewer checks the copy blind. The list of planted deviations and the reviewer's verdicts land as a file, and the tool computes the hit rate. The copy never lands.
- **Counter-checks.** A counter-checker derives checks from the spec alone, before and without the artifacts: tests for code, and questions with expected answers for documents such as a handbook. They are files of the project and stay when Markitect is removed. The state names them per document, apart from its artifacts. The builder may read them and not change them, and a failing one is a `fail`. They replace any count of cases by the reviewer.
- Where a rule of the spec names modules as the code names them, the counter-checker writes architecture tests in the project's own tool. They are red until it is built, run in CI and belong to the landing rule. A rule that names no modules stays with the reviewer.
- **Spot checks and full reviews.** A spot check checks confirmed documents again at random with a fresh reviewer; a verdict that contradicts the earlier one on an unchanged document is a fact, since one of the two is wrong. A full review checks every document, rarely.
- Trust comes from convergence over time, not from a perfect run. Regular jobs keep the project clean and bring it closer to the spec; a single run may leave errors.
- The model per role is a setting, not a principle: by default a smaller model for the builder and the reviewer and a larger one for the roles on the spec. Control samples and counter-checks show whether another model catches more.
- The roles are architect, builder, reviewer, reader, counter-checker and sampler.
- The [trust](concept/trust.md) page describes it.
- Reopen when nobody reads the measured figures or they change nothing in the work; when planted deviations are recognizable as artificial and so always found, or the runs cost more than the figure is worth; when counter-checks repeatedly derive what the spec does not say, which is then a signal about the spec, not a reason to drop them; when spot checks never contradict although reports show errors in confirmed documents; when architecture tests are repeatedly so weak that control samples with broken dependencies pass, which calls for observing the imports; or when deviations do not shrink over many runs, or runs change the same places back and forth.

## The factory

<a id="dec-105-a-first-spec-of-an-existing-project-states-what-is-as-what-should-be"></a><a id="dec-116-the-first-design-walks-every-area-with-the-owner"></a>

### DEC-126 Conversations with the owner, jobs for everything else

- Markitect works as a factory with a quality management loop, without approvals. There are two kinds of work.
- **Conversations** happen only with the owner: the first design, designing together when the owner wants, and choosing between proposals. A conversation never runs as a job.
- **Jobs** are everything else. A job starts from a commit, needs no open session, asks no one during the run, and ends with a branch, rows for the state and reports. It runs by hand, on a schedule or on an event.
- The jobs: intake, triage, architect, build, review, control sample, bug hunt, cleanup, improve, and read and question. Cleanup changes no behaviour. Improve changes behaviour only where the spec leaves it free and adds nothing to the spec. Read and question checks the spec itself.
- The build job runs the counter-checker, then the builder, then the reviewer, until the list is empty; suspect documents go straight to the reviewer. It stops when three rounds in a row leave the list no shorter; what is left becomes a report.
- A long list is cut into slices by folder, with no order, and each slice is merged into the job's branch.
- How often each job runs is a default of the runner, not a rule of the tool.
- A new project starts with the first design: one conversation that walks the areas of the template's area types with the owner, with the questions of their catalogs. It ends with the spec on a branch and the build job starting; what is open becomes reports.
- An existing project without a spec gets a first design in which the project answers instead of the owner. An architect job states what is as what should be, from every artifact, as explicitly as [DEC-122](#dec-122-the-spec-declares-what-is-fixed-like-an-interface) says and not as a transcription. Where a statement does not hold, the spec misread the project and is corrected; the project contradicts itself and the owner says which side holds; or the project does something nothing in it states, which the spec leaves unstated and reports. The owner settles these in a conversation, and a full review then fills the state.
- The [factory](concept/the-factory.md) page describes it.
- Reopen when a job repeatedly gets on only because someone answers during the run, which means the cut between job and conversation is wrong; or when first builds need clearly more rounds without an order of slices, which brings an order by how many documents link to a folder.

### DEC-127 One landing rule, and a spec change lands with its realization

- One rule decides for every job whether its branch lands, with no question to the owner. A branch lands when `status` after it shows nothing that the target branch does not show, apart from documents waiting on the owner; the tests and the counter-checks pass; and the job wrote only its write area. Otherwise the branch stays and there is a report.
- A spec change lands together with its realization, as a pull request with green checks. So the main branch is always built and confirmed, apart from documents waiting on the owner, which are shown and do not fail. `status` on the main branch is then a check for drift: a change past the jobs or an unexplained file.
- The history holds only facts: verdicts with their hashes, moves, and nothing about how a job ran. How a job ran is in its branch and its reports.
- Reopen when reports repeatedly trace back to branches that landed by the rule and still made something worse, which calls for another condition, not an approval; when parallel branches with spec changes contradict each other so often that merges regularly fail, or the owner repeatedly wants to see on the main branch what is not built yet; or when a signal must know which verdicts belonged to one job and that does not follow from time and branch.

### DEC-128 Reports and actions hold everything that needs a change

- Every signal that a change is needed becomes a report: an idea of the owner, a bug report, an incident, a customer's wish, a log, a finding of a job, or a problem with the spec itself. A report says what is wrong or wanted, with evidence, and proposes no solution.
- A report is a document of the language in `.markitect/reports/`, of a report type the tool owns, with the sections source, place, finding with evidence, occurred or risk, rating, violated property, actions, prevention and closing. `check` checks its form.
- Agents report in their answer, and the job writes the file. Only triage adds rating, actions and closing. The source "owner" is set only by a conversation.
- Outside signals pass through intake: a job that reads them isolated and writes a report of fixed form. Their raw text never reaches an agent that may change something.
- Triage sorts each report: valid, a duplicate, which counts as a repetition of the earlier one, or needing nothing, with the reason. It rates by impact and likelihood. Then the switch: where the spec already says what is right, a build job repairs; where the answer follows from what was decided, an architect job changes the spec and the build job realizes it on the same branch; where a new decision is needed, the report escalates to the owner ([DEC-129](#dec-129-the-owner-makes-only-new-decisions-and-the-system-learns-which)).
- An action is a branch, and the report points to it. Every action includes its prevention: a fixed question or a counter-check, so that the same gap shows earlier next time.
- A report is closed when its action has landed, or with the reason why nothing is done. A report that recurs at the same place shows that the action did not work.
- Reports are work, not part of the spec. A document with an open report about it is built no further and waits; `status` shows it as waiting on that report.
- The [reports](concept/reports-and-actions.md) page describes it.
- Reopen when agents repeatedly miss the form of reports and triage cannot sort them reliably, or a project must keep its reports in a ticket system, which brings an adapter and keeps the form; or when prevention piles up questions and counter-checks that never find anything again, which then go.

### DEC-129 The owner makes only new decisions, and the system learns which

- Where the answer follows from what was decided, an architect job changes the spec itself. A new decision is something someone will rely on that nobody decided, a contradiction between two decisions, or the change of a decision. A job that stops without progress and a report that recurs at the same place escalate too.
- An escalated report goes to the owner, who chooses between two or three proposals that agents prepared, each with its effect, none of them built. Only the chosen one is built.
- Each decision of the owner is recorded with its kind, area, proposals, recommendation, choice and comment. From it the system learns which kinds of decision the owner wants to make: from what the owner says, and from behaviour, which it shows as a proposal. A kind the owner has handed over is decided with the recommendation and shown afterwards under "decided for you"; a report reverses it. What the owner does not comment on counts as agreed. Wishes from outside go to the owner until that kind is handed over.
- Learned rules live in `.markitect/`, not in the spec, and only a conversation changes them. A decision made for the owner still lands as a Decision in the spec.
- Reopen when reports repeatedly trace back to spec changes the owner did not want, or the weakening signal fires on changes by agents, which narrows what a job may settle itself; or when the owner repeatedly reverses decisions made for them, which gives that kind back to the owner.

### DEC-130 A change past the spec is back-filled

- A change past the spec shows in `status` as changed documents or unexplained files, on a branch before it lands or on the main branch after a direct push. Where it came from decides what follows.
- From a job, the build job brings the artifacts back to the spec.
- From a conversation, by a person or by Claude for them, it becomes a report "past the spec", with the diff and the commit messages as evidence. Triage sorts it. Where it fills a gap, an architect job writes the back-fill, as a Decision with the reason from the commits where it was a choice; counter-checks are derived from the new text and the reviewer confirms before it lands. Where it contradicts the spec, it is a new decision for the owner: keep it and change the spec, or have the build job revert it. Like drift in Terraform, it is imported or reverted.
- A back-fill shows under "decided for you", and a report reverses it.
- A decision that changes no file is recorded only by the conversation that makes it, by changing the spec. Nothing records chats.
- Reopen when spot checks or later reports repeatedly find back-fills that recorded an error as intent, or the owner reverses back-fills. Then every gap from a conversation goes to the owner as a choice.

<a id="dec-051-agents-get-tasks-the-orchestrator-owns-the-process"></a>

### DEC-051 Agents get tasks, not process

- An agent gets a clear task and an answer format. Order, limits, acceptance and routing to the owner are not explained to agents in prompt text.
- A failure in how agents work with Markitect is fixed in the tool contract, not with more prompt text. Since the tool steers no process, the agents' own files hold their task and answer format, and skills hold no steps an agent must remember.
- The job holds every loop, the build loop included: it starts each role as its own step. The one exception is the first design, whose walk through the areas the conversation with the owner holds.

## Overview

### DEC-131 The tool shows, Claude explains

- There is no user interface for now. The spec is readable, the tool runs natively, and Claude describes what the tool shows. A user interface and clients other than Claude come once Markitect is proven ([DEC-120](#dec-120-how-markitect-is-judged-and-when-it-is-proven)).
- What the owner should not have to take on Claude's word does not come from Claude. The tool, and Git where branches are concerned, show; Claude explains, with a citation for each statement, and says what it did not cover. The content of a report is agents' text and counts only with its evidence.
- Comparing two states is a command of the tool. It compares two folders, spec, state and reports, and names what is new, gone, changed or suspect, with the old and the new text. The tool uses no Git; the runner lays out the other state. It carries "since I last looked", the preview of an idea, its branch against the main branch, and the development in the briefing. A reader's last look is a copy in their profile under `~/.markitect/`, outside the project.
- People have two entries. **Briefing:** what to decide, what was decided for them, ideas, incidents, development, how the factory runs, and what changed since their last look; for every reader, with "to decide" only for the owner. **Explain:** the whole, an area, a document or a question, only from the spec, with citations, and otherwise "not stated".
- Views are generic and come from the language: the topic view, a table per type, a document with its artifacts and verdict, `status` and the comparison. They work for every spec. Views per package come with a user interface.
- The [overview](concept/overview.md) page describes it.
- Reopen, before Markitect is proven, when the owner repeatedly asks for the same picture that the chat cannot show, or a user needs Markitect without Claude; when explanations repeatedly say something wrong despite citations, so the tool shows more; when copies in the profile grow too large, so the profile keeps only a commit; when readers repeatedly do not find "explain the whole", which then gets its own entry; or when explanations of a package repeatedly need the same picture, which the package then ships as a view.

## The tool and its delivery

<a id="dec-096-markitect-computes-and-checks-a-driver-acts"></a><a id="dec-096-the-tool-computes-and-checks-a-driver-acts"></a>

### DEC-096 The tool computes and checks; a runner acts

- The `markitect` program checks and analyses the spec, says what is to verify, and binds verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project.
- Who starts which job and role, when and how often is the runner's, and so is every Git step. A runner may not change the tool, the state's inputs or the write table.
- Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run. Commands are named after what they do ([DEC-132](#dec-132-markitect-is-a-tool-jobs-and-runners)).

### DEC-097 Markitect uses no Git

- The tool makes no repository, commits nothing, keeps no branches, lands nothing and reads no Git history. Agents and people do every Git step.
- It reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, read as plain text.
- So it cannot guarantee that what is committed is what was verified. `status` on any checkout shows what differs from the state, and a CI job can refuse it.

### DEC-132 Markitect is a tool, jobs and runners

- Markitect has three parts, as Terraform has its command line, its pipeline definitions and the services that run them. **The tool** computes and checks. **The jobs** say for each job which roles run in which order, with which task and write area; they are the only part tied to a provider and ship in the Claude Code plugin. **The runners** are where a job runs, the user's choice: their own session, a scheduled task of the desktop app, a cloud task or CI. Markitect ships an entry for each.
- The tool writes only what it computes or binds: `mark` and `move`. What a job reports is a file of fixed form that the tool checks: reports and the results of control samples. The tool computes the factory's figures from the history, the reports and the control samples.
- Rights come from a job's steps. Each role runs as its own step with its own allowed tools. The runner records the state before and after each step, and the tool checks each change against the write table, by file and by row of the state and the history: the builder writes artifacts; the counter-checker the counter-check folder the settings name; the reviewer only verdict rows; the architect the spec, the ignore file and move rows; the job itself reports and control samples with the job as source; triage only rating, actions and closing in reports; and the owner, on a branch from a conversation, also the control files and reports with the owner as source. The write table ships with the tool, and a project may only narrow it.
- The control files are the write table, the settings, the version, the spec path, the learned rules and the CI definition. The tool reads them from the target branch, not from the branch that wants to land.
- The check is a fact only in CI, where the runner is a script no agent can change. In a session the runner is Claude, and the check makes shortcuts visible without proving anything.
- The commands are `init`, `apply`, `new`, `move`, `check`, `show`, `table`, `topics`, `status`, `mark`, `diff`, `figures`, `version` and `licenses`.
- The [tool](concept/the-tool.md) and [jobs and runners](concept/jobs-and-runners.md) pages describe them.
- Reopen when the same job does different things on two runners because a runner has to do more than fetch a commit, call the steps, lay out the states for each step and hand over a branch, which moves the extra into the jobs; when a figure of the factory holds only if the tool binds something a file does not keep, which gets a command after the pattern of `mark`; or when a runner cannot call roles as separate steps, or shortcuts in sessions repeatedly show only at landing and cost much work, which brings back a guard at the moment of writing for that runner, beside the check at landing.

### DEC-133 The project holds only the spec and `.markitect/`, and pins its version

- A project holds two things of Markitect's: the spec, an ordinary folder of documentation whose path `.markitect/` names, by default `docs/spec/`; and `.markitect/`, with the version, the settings, the state, the history, the reports, the control samples, the ignore file, the learned rules and its own `.gitattributes`. Deleting `.markitect/` and the plugin leaves an ordinary repository with documentation, code and tests, the counter-checks included.
- `init` writes only these two. It writes no `AGENTS.md`, no `CLAUDE.md`, nothing in `.claude/` and no line in `.gitignore`. A session learns from the plugin that a folder with `.markitect/` has a spec.
- Settings exist only in the project: the model per role, the cadence, the counter-check folder, and a narrower write table.
- The version is pinned in the project, and every runner uses it, so that sessions and CI compute the same hashes. In Claude Code the plugin fetches the tool in that version; elsewhere a setup step fetches it from the release. A newer version is taken by changing the version line.
- The library is inside the tool: packages, templates, the report type, reference values and question catalogs. `apply` copies packages and templates into the spec, which owns its copy; reference values and catalogs are read from the tool's version.
- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and a provenance file as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused. The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.
- Reopen when sessions in a project repeatedly pass the spec by because they know of it only from the plugin, which lets `init` write one line in `AGENTS.md` saying where the spec is; or when users repeatedly work with old versions because nobody raises the line, which lets `status` say so.

<a id="dec-092-a-command-uses-the-users-own-sign-in"></a>

### DEC-134 Jobs run on the user's subscription, with or without a session

- Jobs run on the login of the user's Claude subscription: in their own session, as a scheduled task, as a cloud task, or in CI with a token that `claude setup-token` makes for a runner without a session. API keys are never used. The token belongs to the subscription of whoever made it.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. Agents run with the rights of whoever runs them, so work that reads untrusted input needs an isolated machine with limited rights around it.
- Claude Code is the only provider for now, and the tool does not change for another.
- Reopen when Anthropic rules out or throttles subscription logins for unattended runs. Then scheduled tasks stay on the user's own machine, and CI waits.

### DEC-135 The tool stays in Go and is rebuilt for the language

- The tool is rebuilt for the language of [DEC-121](#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates), in Go. It must be one program without a runtime that runs on Windows, Linux and macOS and that the plugin and CI download simply, and the release path, checksums and conventions exist.
- A shared language for a user interface and an agent SDK does not count: the tool starts no agent and there is no user interface.
- Reopen when a user interface comes and needs the same code as the tool, or jobs need an agent SDK that Go lacks. Then the language of the jobs is chosen anew, not that of the tool.

## Outlook

<a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-136 Later ideas wait as conduits with a trigger

- Each later idea stays unbuilt and has a trigger, the event that brings it in, as an architect lays conduits in a shell for cables that may come later. No decision in force blocks one of them.
- The [roadmap](roadmap.md#conduits) keeps them in one list: teams, a user interface with other clients and providers, telemetry for reference values, imports as further suspects, operation, and variants per context. Detailed planning waits until its trigger comes.
- Gamification, Kubernetes or etcd as a base, and an offline classifier of claims and evidence are left out.
- Reopen when a trigger comes and a decision in force turns out to block its idea. Then that decision is reopened, not built around.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a><a id="dec-070-parallel-writers-where-useful"></a><a id="dec-077-linux-ci-run-on-request"></a><a id="dec-077-development-pace-and-linux-first-execution"></a>

### DEC-076 Markitect is developed as an ordinary software project

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one page per topic. Target projects keep their spec and `.markitect/`; product code and tests that handle them use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages.
- Sessions work in parallel where their files do not overlap; shared files have one writer at a time.
- CI runs on Linux only, with the race detector, started on request. Markitect's own tests run on Linux; for users, Windows is a main path ([DEC-134](#dec-134-jobs-run-on-the-users-subscription-with-or-without-a-session)).
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
| OQ-013 | When the machine lacks a program or a service the work needs, does the state need a verdict of its own beside `skip` and `fail`? The job ends with a report, and the document stays on the list. | [DEC-124](#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan) |
