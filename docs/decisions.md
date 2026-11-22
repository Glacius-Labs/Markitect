# Decisions

The decisions that hold for Markitect, grouped by topic, and the questions still open. The [concept](concept/README.md) describes the product they add up to, and the [vision](vision.md) says why.

- This page lists only what holds. A decision changes only through a new decision: it takes the next free number, and the entries it replaces leave the page in the same change. Git keeps every earlier wording.
- Only the product owner adds or changes a decision. An agent's proposal enters as an open question and stays one until the owner decides.
- Do not reopen a decision without new evidence. With such evidence, add an open question that cites it and leave the decision as it is.
- An entry keeps its ID, and the IDs it absorbed keep an anchor on it, unless the anchor names a retired design.
- If another page differs from a decision, the decision holds: correct the page.

## Purpose

<a id="dec-101-people-architect-their-project-in-markdown"></a>

### DEC-119 People architect their project in a typed, declarative specification language

- People design, define and model their project in Markdown: what its concepts are and mean, how they relate, its interfaces, structure, rules and way of working. The Architect is an agent that works out the spec with a person: it drafts, analyses and proposes. What follows from earlier decisions is recorded without a question, and people make the new decisions in conversation ([DEC-145](#dec-145-triage-has-three-exits-and-what-follows-from-what-was-decided-is-recorded-without-a-question)).
- The spec is a typed, declarative specification language in Markdown. It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it, as Terraform makes infrastructure match its configuration.
- It is a specification language, as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by a tool; its content is prose that an agent interprets. It is never a programming language: there the result follows from the text, and here only the structure is checked ([DEC-138](#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation takes the place of code as what people work on. What matters in code stays as relevant there: consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- An agent can be wrong, so every result is checked and recorded ([DEC-125](#dec-125-trust-is-measured-not-granted)).
- Reopen when readers repeatedly take the image to mean that Markitect executes the spec.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool computes and checks, so that the delegated work can be trusted.
- Judge Markitect as a way of developing software, not as a linter or a context tool. Removing delegated realization as scope creep contradicts this decision.

<a id="dec-002-trust-is-measured-against-todays-practice"></a><a id="cpt-001-trust-comes-from-structure"></a><a id="cpt-001-federalism"></a>

### DEC-137 Trust is measured against today's practice

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that, not compiler-grade guarantees.
- Trust comes from structure, not from infallible agents: separate roles, each a fresh session, a list computed from the spec rather than from memory, a check of every document against the spec, and measurements of how well that check works ([DEC-125](#dec-125-trust-is-measured-not-granted)). The [vision](vision.md#where-trust-comes-from) lists the mechanisms.
- Claims read "more reliable than the baseline", never "correct". Use the smallest process that earns the trust: more report fields or completeness declarations do not by themselves make work more correct. A control sample measures the checker and leaves the work as it is.

<a id="dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart"></a><a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a>

### DEC-138 The compiler is a borrowed idea, and two claims stay apart

- The spec's structure is checked deterministically: titles, types, sections and links. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *Nothing in the spec's scope is overlooked* can be computed from links and the mapping of every file; dependencies in code that no link states are a known gap. *Everything is realized correctly* can only be supported by evidence: a fresh reviewer, counter-checks, the project's tests and `check`.

<a id="dec-009-how-markitect-is-judged"></a><a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a>

### DEC-120 How Markitect is judged, and when it is proven

- The aim is one clean and simple version that works reliably, not to prove that one version beats another.
- The long-term questions: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to architecture?
- The method is judged apart from any runtime: a failure of Markitect's own tool or skills is a product finding, a runner or environment failure a limit of the measurement.
- The lab test comes first: two clean passes in a row over the eight change steps of the 5× Makerspace project, every step reaching the main branch with its spec change and its realization together, confirmed, and every hidden case passing.
- Markitect is proven when a project run by people who do not develop Markitect runs four weeks in the circle with real work in CI, and four figures hold; how each is computed is settled with the runs that bring control samples and spot checks:
  - **Work:** stories that change spec and code reach the main branch every week.
  - **Convergence:** open findings and `status` do not grow over the weeks.
  - **Checking:** the hit rate of control samples does not fall, and spot checks do not contradict earlier verdicts more often.
  - **No rescue:** no merged story is reverted in Git except through a finding.
- All four can turn out negative without anyone reading. Being proven unlocks the conduits that wait for it ([DEC-136](#dec-136-later-ideas-wait-as-conduits-with-a-trigger)). The [concept](concept/what-is-shown.md) says what is shown and what is not.
- Reopen when the figures hold but the outside project no longer runs through Markitect.

<a id="dec-032-quality-simplicity-and-stability-over-effort-and-compatibility"></a><a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a><a id="dec-038-learn-through-complete-workflows"></a>

### DEC-139 Quality, simplicity and stability over effort and compatibility

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions while Markitect is early. Compatibility counts from the first project run by people who do not develop Markitect. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure.
- Simpler is more robust, and simplicity is earned through insight. Markitect works with the coding agents' own features, skills and `AGENTS.md`, rather than building machinery around them.
- A substantial redesign names the new observation behind it and the result that would show it is better, and asks first whether a mechanism is needed at all or something simpler would do.

<a id="dec-098-one-concept-describes-the-product"></a>

### DEC-140 One concept describes the product

- The [concept](concept/README.md) is the single description of the product and the target that Main builds: the spec with its templates, the way of working that stands in it, the tool that checks the spec and computes what is built and verified, and the skills that realize the way of working.
- Earlier concepts are kept on archive branches ([history](history.md)), not on Main.

## The spec

### DEC-121 A spec is plain Markdown with types, packages and templates

- A document is a Markdown file with a title and `##` sections, without YAML front matter. It is known by its path below the spec folder. Folders are free, so a spec may be ordered by feature. File names use lowercase letters, digits and hyphens; the readable name is the title.
- A type is a document whose `##` sections are the template of its documents; the text under each says what belongs there, and `(optional)` marks a section that is not required. A document names its type under the title, `Type: [Service](../types/service.md)`. Types and documents relate as classes and objects do. A value, such as a state or an owner, stands in the first line of its section; there are no lines in the head but the type line.
- Links are ordinary relative Markdown links, which work on GitHub, in editors and in note tools alike. A link is no import, and no section means anything special to the tool.
- There is no module layer: no modules declared by an anchor, no rules on the direction of links, no visibility and no binding by place. What belongs together shares a folder. A part of the spec is a folder, and its README is its table of contents.
- Types come from packages, folders of types such as `core`, `software`, `company` and `factory`. Structure comes from templates, folders of documents with empty sections, which can be applied at any level: a whole project, one part of the domain, or a field such as fire protection. Each type says before its sections what it is and when to use it, and under each section what belongs there, with an example. A spec owns its copy of each package, and `apply` never overwrites it.
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
- The spec describes the end state, not the way there, and not the work that changes it: a backlog, a plan, tasks towards a goal or a project risk stay out. The types of the work, such as Story and Finding, stand in the spec; their documents lie in `.markitect/` ([DEC-144](#dec-144-reports-findings-and-plan-items-hold-the-work)). Running processes and tasks of an operation, the way of working of a project and goals with a date belong in, since they say what should hold.
- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines, configuration and skills alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- A spec starts from four templates: `project`, with purpose, context, guiding principles and quality; `product`, with topics, what the product should do, structured but with no fixed chapters; `architecture`, with building blocks and the rules for building; and `factory`, with the way of working. The `software` template, with Purpose, Context, Domain, Interfaces, Quality and Architecture, is a checklist for `project`, not an outline. The software package holds Assumptions in Context, with who ensures each and what is therefore not built; the answer the caller gets for each condition under which an Operation is refused; the nine characteristics of ISO/IEC 25010 as optional sections of Quality, which a template lists empty so that `check` names each until it says what it asks or why nothing; and Deployment in Architecture. The Decision type has an optional section "Reopen when".
- A business case, scenarios and operational tasks wait until a project needs them.
- The [content](concept/what-goes-in-a-spec.md) page describes it.
- Reopen when a builder or reviewer in a software project misses something about operation that the spec should have said; when a section of the software package stays empty or says only "nothing" in real projects, which removes it; or when plans or task lists appear in a spec and go stale after every change.

### DEC-123 A good spec has five properties and shows them in four sources

- A good spec is, after the characteristics of good requirements in ISO/IEC/IEEE 29148: **necessary**, it states only what is fixed; **complete**, nothing fixed is missing; **consistent**, no two statements exclude each other; **unambiguous and verifiable**, two readers understand the same and a checker can say yes or no, with every statement as strong as possible; and **structured**, one thing per document, each fact on its level, and what has facts of its own in a document of its own.
- Coupling, cohesion, abstractness and link density are no measures of a spec. There are no fixed thresholds.
- Four sources show how good a spec is. The spec itself: checked by `check`, shown by a topic view and signals, read by a person or agent against the five properties, and questioned with competency questions answered only from the spec with a citation. Its changes: six signals, a wide change, documents that change together without a link, links dragged along, a hotspot, back and forth, and weakening, with mechanical changes left out. What is built from it. And the questions it raises.
- Every finding names its place and the property it violates: one of the five, or **correct**, when a person sees a result and says it was not meant so. Only use shows correctness.
- Every number is shown against the reference value of its type, which ships with the package, and against the rest of the project. Reference values come from proven specs; a proven spec is one that the people who answer for it name as a reference.
- Question catalogs ship with packages and templates. A project's own questions lie beside the spec, not in it. An answered question from use becomes a fixed question, and a general one can move into the package.
- The [quality](concept/is-the-spec-good.md) page describes it.
- Reopen when findings repeatedly show something bad that falls under none of the five properties or correctness, the property is not assigned reliably, or nobody reads the evaluation; when a signal of the spec itself shows almost only false alarms on real specs; when a change signal never points at a real finding over a series of changes on a real project; when reference values show almost only false alarms, or differ so much between projects that none carries; or when answers stay unreliable despite citations, or a catalog grows without ever finding anything.

## State and trust

<a id="dec-006-every-file-is-accounted-for"></a><a id="dec-006-every-file-is-covered-or-explicitly-ignored"></a><a id="dec-099-a-record-of-what-was-verified"></a><a id="cpt-002-tests-are-the-projects-and-they-are-realizations-too"></a><a id="cpt-002-tests-are-realizations-too"></a><a id="dec-034-removed-intent-removes-its-realization"></a>

### DEC-124 A state maps each document to its artifacts, and `status` is the plan

- Terraform is the image. The spec is the configuration, an interface; the artifacts are its implementations; the state in `.markitect/` is the matrix between them; `check` is `validate`, `status` is Terraform's `plan`, and building a story is `apply`. The unit is one document.
- The state holds one row per document: its artifacts, its counter-checks apart from them, its verdict and the hashes at its last confirmation. The reviewer writes the row with `confirm` and names the artifacts and the evidence. Nothing of Markitect stands in an artifact or in the spec, so deleting Markitect leaves an ordinary repository.
- A change has two stages. A document is **changed** when its own text or one of its artifacts moved: a builder realizes it and a reviewer confirms it. It is **suspect** when a document it links to, or one that links to it, changed, as in DOORS: a reviewer judges whether the change means something here. A changed type makes its documents suspect. Links are ordinary links in both directions; "Applies to" is an ordinary section. The documents of the work in `.markitect/` are not documents of the spec: a link from them into the spec makes nothing suspect.
- Every file is mapped to a document or ignored, by `.gitignore` or by `.markitect/ignore`. Any other file is **unexplained** in `status`, which then is not empty. A file whose document is gone is unexplained too. `status` only shows and is never a gate ([DEC-146](#dec-146-a-spec-change-lands-with-its-realization-and-status-only-shows)).
- `check` compares nothing with artifacts. Everything that compares spec and artifacts is in `status`.
- There are three verdicts: `ok`, `fail` and `skip`. A document recorded `fail` stays on the list as changed until it holds. `confirm` binds a verdict, with its artifacts and evidence, to the state `status` showed, and refuses when an input changed since.
- At each confirmation the history records, as hashes, which documents were changed at that moment, so that the change signals of [DEC-123](#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources) can be computed. What a statement said before comes from Git, read by an agent. `move` rewrites every link to a moved document, carries its row in the state, and records a mechanical event in the history.
- The tests a builder writes are realizations, so a green one is evidence, not proof.
- The [state](concept/the-state.md) page describes it.
- Reopen when reviewers keep reading the same artifact for many small documents and that costs more than one verdict for the whole; when reviewers repeatedly name a document's artifacts incompletely and it shows late, so the builder maps them again, in the same state; when suspect entries make the list so long that reviewers confirm them unread, so only links in certain sections count; when new documents are repeatedly not reached because a rule lists what it applies to, so the tool reads "Applies to"; when unexplained files are almost only files nobody wants to explain, which then go into the ignore file; or when a change signal needs more text than the hashes give and an agent cannot read it reliably from Git.

### DEC-125 Trust is measured, not granted

- There are no approvals, and no trust comes from a person looking through the work. A mechanism counts only when it delivers a fact that can turn out negative without anyone reading: a run, a figure, a difference between artifact and spec. Another agent's opinion is no such fact. The image is a testing laboratory.
- The question has two halves. Whether the artifacts match the spec is secured without people, by confirmation, control samples, counter-checks and use. Whether the spec says what is meant shows only in use, through findings.
- Counter-checks, control samples and spot checks come with the runs, later; building with the way of working does not wait for them.
- **Control samples.** A sampler plants known errors, differences from the spec, in a throwaway copy of the artifacts. The reviewer checks the copy blind. The list of planted errors and the reviewer's verdicts are recorded, and the hit rate is computed from them. The copy never lands.
- **Counter-checks.** A counter-checker derives checks from the spec alone, before and without the artifacts: tests for code, and questions with expected answers for documents such as a handbook. They are files of the project and stay when Markitect is removed. The state names them per document, apart from its artifacts. The builder may read them and not change them, and a failing one is a `fail`. A document's counter-checks are its cases; the reviewer counts none.
- Where a rule of the spec names modules as the code names them, the counter-checker writes architecture tests in the project's own tool. They are red until it is built and run in CI. A rule that names no modules stays with the reviewer.
- **Spot checks and full reviews.** A spot check checks confirmed documents again at random with a fresh reviewer; a verdict that contradicts the earlier one on an unchanged document is a fact, since one of the two is wrong. A full review checks every document, rarely.
- Trust comes from convergence over time, not from a perfect run. Regular runs keep the project clean and bring it closer to the spec; a single run may leave errors.
- The model of each agent stands at its Agent document in the spec, not in a principle: by default a smaller model for the builder and the reviewer and a larger one for the agents on the spec. Control samples and counter-checks show whether another model catches more.
- The agents of the way of working are listed in [DEC-143](#dec-143-the-way-of-working-is-a-circle-for-a-team-with-roles-findings-and-a-plan); the counter-checker and the sampler come with the runs.
- The [trust](concept/trust.md) page describes it.
- Reopen when nobody reads the measured figures or they change nothing in the work; when findings repeatedly show late that the spec said something other than was meant, which gives the second half a building block of its own, never an approval; when the figures per model diverge clearly, which adjusts the default, not the principle; when planted errors are recognizable as artificial and so always found, or the runs cost more than the figure is worth; when counter-checks repeatedly derive what the spec does not say, which is then a signal about the spec, not a reason to drop them; when spot checks never contradict although findings show errors in confirmed documents; when architecture tests are repeatedly so weak that control samples with broken dependencies pass, which calls for observing the imports; or when the differences between artifacts and spec do not shrink over many runs, or runs change the same places back and forth.

## The way of working

<a id="dec-105-a-first-spec-of-an-existing-project-states-what-is-as-what-should-be"></a>

### DEC-143 The way of working is a circle for a team, with roles, findings and a plan

- Around the spec, a team works in one circle, the same at every level. **Observe:** what happens lands as a report in the context. **Triage:** reports become findings, classified and filed. **Briefing:** what needs a person stands there in short, and people decide in conversation. **Record:** decisions are carried out and stand in the plan. **Work:** planning with a person, building without one, accepting. What shows up in the work goes back to Observe.
- People hold roles: Admin, Developer and QA. Each role has its usual way of working: where it usually works, which skills it usually uses, and what another role is usually there for. Persons, roles and who holds which stand in the spec ([DEC-147](#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)).
- Nothing is forbidden. Work outside the usual way of working is a deviation with a level, one of the four importances. The agent at hand says so, asks for the reason and notes both in the context. At `critical`, work that is hard to undo or reaches outside, it stops before acting and asks. Agents that ask no one write the reason themselves and stop at `critical`. Deviations that pile up become a finding: a Smell, or a Factory finding when the reasons are good and the role is described wrongly.
- An agent is a fresh session that runs exactly one skill. The agents are Norman, a person's entry in the morning; the Architect, who works out the spec and the plan with a person; the Builder, the Reviewer and the Analyst; the Observers; the Triager, the Recorder and the Briefer; and the Housekeeper, the daily run.
- The plan is a tree of Area, Feature, Story and Task. An Area is an area of planning with an owner and an optional class, a document of the spec in its `factory` part and never a folder of it. A Feature is owned, a Story is built. Whoever starts on a plan item first claims it on the main branch.
- People decide in conversation: when they plan a Feature or a Story with the Architect, whose last summary they accept as their go, and on findings that need a new decision. Nobody approves before a merge: a story reaches the main branch without a person once spec and plan are clear and builder and reviewer agree. QA accepts what has a surface once it is on the main branch.
- A new project starts with the Architect and a person working through the templates. An existing project gets its first spec from the Architect too, which states what is as what should be: it asks whether the choice of templates fits; for each part of the spec a subagent finds its own context and fills the part, with a report of where each document comes from, why it is laid out so, what was unclear and where sources contradict each other; what is unclear or contradictory is not guessed but becomes a finding and stays open in the document; and the person sees the parts one after another and decides in conversation.
- The `factory` template holds the way of working in detail as a standard, which a project owns and changes like any part of its spec.
- The [circle](concept/the-circle.md) page describes it.
- Reopen when deviations of one kind pile up with good reasons across persons, which changes the role, not the principle; when stories that reached the main branch without a person are repeatedly reverted; or when findings repeatedly stay pending because no one decides them.

### DEC-144 Reports, findings and plan items hold the work

- A **report** is what an agent or an observer writes about one run or session, in the context: each kind of agent has one folder there, named like its skill, with one report per run, named by date, time and its item, and a checkpoint where the agent continues. Every report has the same head: its item, who started it (a person, Norman, the Housekeeper or a routine), for whom, when it started and finished, and its outcome. Every session that works on a plan item ends with a short report. A run that observed nothing writes only its checkpoint.
- A **finding** is one thing that deserves attention, without a solution. It has a type, Requirement, Idea, Question, Bug, Risk, Smell, Planning, Factory or Unclassified; an importance, `critical`, `high`, `normal` or `low`, with its reason; its Area; what it is, its source and its context; and for a Bug or a Risk its likelihood and impact. Its folder is its state: `pending` needs a person, `ready` is released for work, `closed` is done. Every finding ends in `closed`, with its outcome.
- The plan is rooted in the Areas, which are documents of the spec in its `factory` part. A **plan item** is a Feature or a Story below an Area; a Story holds its Tasks. Its identity is its place in the tree, with no number of its own. A Story has a state, `draft`, `ready`, `in progress`, `acceptance` or `done`; it names its origin, a finding or the context, the places in the spec it changes and how, and its result. The reviewer's `confirm` names the files and the evidence for each place, and the commit names the story. So one trace leads from the origin to the file and back.
- The **briefing** is what reaches people, short and ordered by importance: what needs a decision, what was decided for you, what went to work without a decision, and what was decided yesterday. The Briefer writes it daily, weekly and for each meeting. Importance decides when something is shown: `critical` at once, `high` daily, `normal` and `low` weekly.
- Reports, findings, plan items and briefings are documents of the language. Their types stand in the `factory` part of the spec, their documents lie in `.markitect/` ([DEC-149](#dec-149-markitect-holds-the-work-and-init-lays-out-templates-and-skills)), and `check` checks them against their types. They are work, not part of the spec.
- The [work](concept/the-work.md) page describes them.
- Reopen when agents repeatedly miss the forms and triage cannot sort them reliably; when a project must keep its plan in a ticket system, which brings a mapping and keeps the forms; or when reports pile up in the context that no triage reads.

### DEC-145 Triage has three exits, and what follows from what was decided is recorded without a question

- The Triager reads new reports, bundles related statements into one finding per topic and files it. A statement about the spec is held against the spec: what the spec already says makes no finding. A finding that recurs is bundled into a new finding against the concept one level higher, where the problem may lie.
- A finding leaves triage by one of three exits:
  - **The spec already says what is right:** `ready`. The Recorder makes it a story with fixed tasks, and it is built without a person.
  - **The answer follows from what was decided:** `ready`, with the place it follows from. The Recorder records it in the spec of the level concerned, the global level included, without a question, and closes the finding.
  - **A new decision is needed:** `pending`. People decide in conversation, from the briefing, and the Recorder carries out the decision.
- A record under the second exit adds nothing new: it writes down what already held. It reaches the main branch without a build; where the artifacts differ from it, that is a Bug like any other. It shows in the briefing under "decided for you", with its place, and a finding reverses it.
- Which exit a finding takes follows from its type and from what the spec says, not from its size. There is no way to work past the plan: work is a plan item.
- Reopen when findings repeatedly trace back to records people did not want, or the weakening signal fires on records, which narrows the second exit; or when people repeatedly reverse what was decided for them.

### DEC-146 A spec change lands with its realization, and `status` only shows

- When planning changes the spec as it stands, the change lies on the branch of the story that realizes it, not on the main branch; a change that concerns several stories is cut with them, and each carries the part it realizes. The builder continues on that branch, and spec and artifacts reach the main branch together. The plan itself goes to the main branch at once. So the spec on the main branch is never ahead of a planned story. A record under the second exit of triage reaches the main branch without a build, since it adds nothing new ([DEC-145](#dec-145-triage-has-three-exits-and-what-follows-from-what-was-decided-is-recorded-without-a-question)).
- A fresh reviewer checks the build against the plan and the target picture of the story and against every level above, up to the global spec, and returns `comments`. It changes nothing that was built. When builder and reviewer agree, it runs the targeted tests and writes `confirm`. Then the story goes to the main branch.
- `status` only shows. It is never a gate, in CI neither. What it shows, a change past the spec, an unexplained file or changed evidence, becomes a finding like any other observation.
- The history holds only facts: verdicts with their hashes, and moves. How an agent ran is in its report.
- Reopen when stories repeatedly reach the main branch and make something worse that a check before the merge would have caught, which calls for a test in the way of working, not for an approval; when parallel stories change the same place of the spec so often that merges regularly fail; or when people repeatedly need to see on the main branch a spec change that is not built yet.

### DEC-147 The way of working stands in the spec, and skills realize it

- The way of working is part of the spec, in its `factory` part: the circle, the agents, the roles and who holds them, the meetings, the planning with its Areas and classes, the dashboards and handbooks it wants, and the types of the work.
- Skills are artifacts of the spec, as code is of the product: built from it, confirmed by a reviewer and mapped in the state, so that the trace holds for the way of working too. A handbook of the way of working and a dashboard are artifacts built from it as well.
- What an agent does is described completely and step by step in its skill. People need not know the way of working: the agents lead them.
- Skills lie in the repository and hold for Claude Code and Codex alike. Which agents and observers exist, their model, their schedule and whether they ask, stand at their Agent document in the spec. Whoever starts an agent, a person, Norman, the Housekeeper or a routine, reads them there. There is no settings file.
- Nothing is steered through the tool ([DEC-142](#dec-142-the-tool-computes-and-checks-a-runner-acts)).
- A failure in how agents work is fixed in the `factory` part of the spec and in the skill built from it, not with prompt text beside them.
- Reopen when skills built from the spec repeatedly diverge from it in ways the reviewer misses; when Claude Code and Codex need different skills for the same agent, which brings an adapter per provider; or when a model or a schedule must repeatedly differ per person or machine.

## Overview

### DEC-131 The tool shows, Claude explains

- There is no user interface for now. The spec is readable, the tool runs natively, and the agent describes what the tool shows. A user interface comes once Markitect is proven ([DEC-120](#dec-120-how-markitect-is-judged-and-when-it-is-proven)). A dashboard or a handbook that the way of working describes is an artifact of the project, built from its spec, not a user interface of Markitect.
- What people should not have to take on an agent's word does not come from an agent. The tool, and Git where branches are concerned, show; the agent explains, with a citation for each statement, and says what it did not cover. The content of a report or a finding is agents' text and counts only with its source.
- Comparing two states is a command of the tool. It compares two folders, spec, state and work, and names what is new, gone, changed or suspect, with the old and the new text. The tool uses no Git; the skill lays out the other state. It carries "since I last looked" and the preview of a branch against the main branch. A reader's last look is a copy in their profile under `~/.markitect/`, outside the project.
- People have two entries. **Briefing:** what needs a decision, what was decided for them, what went to work and what was decided, as the Briefer writes it from the findings ([DEC-144](#dec-144-reports-findings-and-plan-items-hold-the-work)), and what changed since their last look. **Explain:** the whole, a part, a document or a question, only from the spec, with citations, and otherwise "not stated".
- Views are generic and come from the language: the topic view, a table per type, a document with its artifacts and verdict, `status` and the comparison. They work for every spec. Views per package come with a user interface.
- The [overview](concept/overview.md) page describes it.
- Reopen, before Markitect is proven, when people repeatedly ask for the same picture that the chat cannot show, or a user needs Markitect without an agent; when explanations repeatedly say something wrong despite citations, so the tool shows more; when copies in the profile grow too large, so the profile keeps only a commit; when readers repeatedly do not find "explain the whole", which then gets its own entry; or when explanations of a package repeatedly need the same picture, which the package then ships as a view.

## The tool and its delivery

<a id="dec-096-markitect-computes-and-checks-a-driver-acts"></a><a id="dec-096-the-tool-computes-and-checks-a-driver-acts"></a>

### DEC-142 The tool computes and checks; a runner acts

- The `markitect` program checks and analyses the spec and the documents of the work, says what is to verify, and binds verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project.
- Who starts which agent, when and how often, is the runner's: a person, Norman, the Housekeeper or a routine, as the Agent documents of the spec say ([DEC-147](#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)). Agents and people make every Git step.
- Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run. Commands are named after what they do ([DEC-148](#dec-148-markitect-is-a-tool-a-spec-with-templates-and-skills)).

### DEC-097 Markitect uses no Git

- The tool makes no repository, commits nothing, keeps no branches, lands nothing and reads no Git history. Agents and people do every Git step.
- It reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, read as plain text.
- So it cannot guarantee that what is committed is what was verified. `status` on any checkout shows what differs from the state, and what it shows becomes a finding ([DEC-146](#dec-146-a-spec-change-lands-with-its-realization-and-status-only-shows)).

### DEC-148 Markitect is a tool, a spec with templates, and skills

- Markitect has three parts. **The tool** computes and checks. **The spec with its templates:** the library inside the tool ships packages of types and the templates `project`, `product`, `architecture` and `factory`, and the `factory` template holds the way of working as a standard. **The skills** of the standard realize the way of working, one per agent; they lie in the project as artifacts of its spec.
- The tool writes only what it computes or binds: `confirm` and `move` in the state and the history. `init`, `apply`, `new` and `move` write documents on request. `check` checks the spec and the documents of the work against their types.
- The commands stay flat, as Git's do: `init`, `check`, `show`, `table`, the topic view, `new`, `apply`, `move`, `status`, `confirm`, `compare` and `version`, and the commands for the competency questions and the license notices. The help groups them: look, check, change the spec, confirm, set up.
- The tool steers nothing: it has no landing rule, no check of steps, no write table, no command for reports and no settings.
- The [tool](concept/the-tool.md) and [project](concept/the-project.md) pages describe it.
- Reopen when a skill repeatedly needs a fact that only the tool could compute or bind, which gets a command after the pattern of `confirm`; or when sessions repeatedly go past a rule of the way of working and it shows only late and at a high cost.

### DEC-149 `.markitect/` holds the work, and `init` lays out templates and skills

- A project holds the spec, an ordinary folder of documentation whose path `.markitect/` names, by default `docs/spec/`; the skills of the standard, as artifacts of the spec; and `.markitect/`.
- `.markitect/` holds `version`, `spec`, `ignore`, `state.jsonl`, `history.jsonl` and its own `.gitattributes`, and the work: `context/`, `findings/` with `pending/`, `ready/` and `closed/`, `plan/` and `briefing/`. In `context/` each kind of agent has one folder, named like its skill, with one report per run, named `YYYY-MM-DD-HHMM-<item>.md`, and where needed a `checkpoint.md`; each observer is a kind of its own.
- `init` lays out `.markitect/` with empty folders for the work, the spec with the four templates, the skills of the standard in `.claude/skills/` and `.agents/skills/`, and a short entry in `AGENTS.md`. Skills always lie in the repository, never in the plugin. The plugin only fetches the tool.
- Deleting `.markitect/` and the tool leaves an ordinary repository with documentation, code, tests and skills, the counter-checks included.
- The version is pinned in the project, and everyone who runs the tool uses it, so that sessions and CI compute the same hashes. In Claude Code the plugin fetches the tool in that version; elsewhere a setup step fetches it from the release. A newer version is taken by changing the version line.
- The library is inside the tool: packages, templates with the skills of the standard, reference values and question catalogs. `apply` copies packages and templates into the spec, which owns its copy; reference values and catalogs are read from the tool's version.
- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and a provenance file as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused. The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.
- Reopen when projects change their copies of the standard skills so much that a newer standard can no longer be taken and that does harm; or when users repeatedly work with old versions because nobody raises the line, which lets `status` say so.

<a id="dec-092-a-command-uses-the-users-own-sign-in"></a>

### DEC-134 Agents run on the user's subscription, with or without a session

- Agents run on the login of the user's subscription: in their own session, as a scheduled task, as a cloud task, or in CI with a token that `claude setup-token` makes for a runner without a session. API keys are never used. The token belongs to the subscription of whoever made it.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. Agents run with the rights of whoever runs them, so work that reads untrusted input needs an isolated machine with limited rights around it.
- The skills hold for Claude Code and Codex alike ([DEC-147](#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)). Runs without a session are described for Claude Code only for now, and the tool does not change for another provider.
- Reopen when Anthropic rules out or throttles subscription logins for unattended runs. Then scheduled tasks stay on the user's own machine, and CI waits.

### DEC-135 The tool stays in Go and is rebuilt for the language

- The tool is rebuilt for the language of [DEC-121](#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates), in Go. It must be one program without a runtime that runs on Windows, Linux and macOS and that the plugin and CI download simply, and the release path, checksums and conventions exist.
- A shared language for a user interface and an agent SDK does not count: the tool starts no agent and there is no user interface.
- Reopen when a user interface comes and needs the same code as the tool, or skills need an agent SDK that Go lacks. Then the language of the skills is chosen anew, not that of the tool.

## Outlook

<a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-136 Later ideas wait as conduits with a trigger

- Each later idea stays unbuilt and has a trigger, the event that brings it in, as an architect lays conduits in a shell for cables that may come later. No decision in force blocks one of them.
- The [roadmap](roadmap.md#conduits) keeps them in one list: a user interface with other clients, and providers beyond Claude Code and Codex, telemetry for reference values, imports as further suspects, operation, and variants per context. Detailed planning waits until its trigger comes. Several people with roles are part of the way of working ([DEC-143](#dec-143-the-way-of-working-is-a-circle-for-a-team-with-roles-findings-and-a-plan)), so teams are no conduit.
- Gamification, Kubernetes or etcd as a base, and an offline classifier of claims and evidence are left out.
- Reopen when a trigger comes and a decision in force turns out to block its idea. Then that decision is reopened, not built around.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a><a id="dec-070-parallel-writers-where-useful"></a><a id="dec-077-linux-ci-run-on-request"></a><a id="dec-077-development-pace-and-linux-first-execution"></a>

### DEC-076 Markitect is developed as an ordinary software project

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one page per topic. Target projects keep their spec and `.markitect/`; product code and tests that handle them use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages.
- Sessions work in parallel where their files do not overlap; shared files have one writer at a time.
- CI runs on Linux only, with the race detector, started on request. Markitect's own tests run on Linux; for users, Windows is a main path ([DEC-134](#dec-134-agents-run-on-the-users-subscription-with-or-without-a-session)).
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
| OQ-013 | When the machine lacks a program or a service the work needs, does the state need a verdict of its own beside `skip` and `fail`? The agent ends with its report, and the document stays on the list. | [DEC-124](#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan) |
