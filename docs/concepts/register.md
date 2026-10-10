# Concept register

This register records Markitect's long-term vision, product-level decisions, clarifications, endorsed directions, promises, assumptions, concepts, future enhancements, ideas and open questions. Each entry has a status and an origin. "Owner" means Markitect's product owner. The [concept record guide](README.md) defines entry types, statuses and the rules for changing entries. The [vision](../vision.md) presents the thesis these entries support. Earlier collections from the Concepts and Ideas chats are preserved in the [sources](sources/README.md) and mapped [below](#imported-collections).

## Index

| ID | Type | Title | Status |
|---|---|---|---|
| [VIS-001](#vis-001-cockpit-personal-ai-and-autonomous-project-organization) | Vision | Cockpit, personal AI and autonomous project organization | Owner statement; planning deferred |
| [DEC-001](#dec-001-delegated-realization-is-central-to-the-product) | Decision | Delegated realization is central to the product | Accepted |
| [DEC-002](#dec-002-trust-is-measured-against-todays-practice) | Decision | Trust is measured against today's practice | Accepted |
| [DEC-003](#dec-003-the-compiler-is-a-borrowed-concept) | Decision | The compiler is a borrowed concept | Accepted |
| [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) | Clarification | The Manager hierarchy is core; Government belongs to a later axis | Clarified |
| [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility) | Decision | Model granularity is the author's responsibility | Accepted |
| [DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored) | Decision | Every file is covered or explicitly ignored | Accepted |
| [DEC-007](#dec-007-more-economical-models-means-capable-mid-tier-models) | Clarification | "More economical models" means capable mid-tier models | Clarified |
| [DEC-008](#dec-008-the-delegated-method-has-not-yet-been-tested) | Decision | The delegated method has not yet been tested | Accepted |
| [DEC-009](#dec-009-the-central-evaluation-question) | Decision | The central evaluation question | Accepted |
| [DEC-010](#dec-010-evaluate-the-method-separately-from-its-execution-runtime) | Endorsed direction | Evaluate the method separately from its execution runtime | Endorsed; plan pending |
| [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits) | Decision | Detailed planning of future ideas waits | Accepted |
| [DEC-012](#dec-012-government-ideas-evaluation-commissioned) | Decision | Government ideas evaluation commissioned | Accepted; result not adopted |
| [DEC-013](#dec-013-linux-first-for-tests-and-the-playground) | Decision | Linux first for tests and the playground | Accepted |
| [DEC-014](#dec-014-compatibility-does-not-drive-decisions) | Decision | Compatibility does not drive decisions | Accepted |
| [DEC-015](#dec-015-a-fresh-roadmap-earlier-plans-are-inputs) | Decision | A fresh roadmap; earlier plans are inputs | Accepted |
| [DEC-016](#dec-016-a-clean-stable-testable-main-and-uniform-structure-first) | Decision | A clean, stable, testable main and uniform structure first | Accepted |
| [PRM-001 to PRM-007](#promises) | Promise | Target outcomes | Target |
| [ASM-001 to ASM-005](#assumptions-and-hypotheses) | Assumption | Assumptions and hypotheses | See entries |
| [CPT-001](#cpt-001-federalism) | Concept | Federalism | Accepted |
| [CPT-002](#cpt-002-tests-are-realizations-too) | Concept | Tests are realizations too | Accepted |
| [CPT-003](#cpt-003-two-parts-of-the-promise) | Concept | Two parts of the promise | Clarified |
| [CPT-004](#cpt-004-the-specification-stays-effective) | Concept | The specification stays effective | Clarified; mechanism proposed |
| [ENH-001](#enh-001-observed-code-dependencies-widen-impact) | Enhancement | Observed code dependencies widen impact | Accepted problem; approach open |
| [ENH-002](#enh-002-model-quality-linter-and-metrics) | Enhancement | Model-quality linter and metrics | Accepted direction |
| [ENH-003](#enh-003-test-to-statement-binding-and-test-review) | Enhancement | Test-to-statement binding and test review | Accepted direction |
| [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) | Enhancement | Per-role agent configuration and model mixing | Accepted direction |
| [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method) | Enhancement | Exchangeable executor for the delegated method | Endorsed; design pending |
| [OQ-001 to OQ-006](#open-questions) | Open question | Open questions | Open |
| [IDEA-001](#idea-001-the-model-as-institutional-memory-for-stateless-agents) | Idea | The model as institutional memory for stateless agents | Proposed |
| [IDEA-002](#idea-002-government-delegated-model-maintenance) | Idea | Government: delegated model maintenance | Owner idea; parked |
| [IDEA-003](#idea-003-degree-of-freedom-per-manager-and-rule-strictness) | Idea | Degree of freedom per Manager and rule strictness | Owner idea; parked |
| [IDEA-004](#idea-004-project-goals-and-non-functional-priorities) | Idea | Project goals and non-functional priorities | Owner idea; parked |
| [IDEA-005](#idea-005-reasoned-user-decisions-as-project-precedents) | Idea | Reasoned user decisions as project precedents | Owner idea; parked |
| [IDEA-006](#idea-006-dispute-severity-and-higher-review-levels) | Idea | Dispute severity and higher review levels | Owner idea; parked |
| [IDEA-007](#idea-007-managers-escalate-implementation-cases) | Idea | Managers escalate implementation cases | Owner idea; parked |
| [IDEA-008](#idea-008-configurable-local-priorities-at-every-level) | Idea | Configurable local priorities at every level | Owner idea; parked |
| [IDEA-009](#idea-009-regular-briefings) | Idea | Regular briefings | Owner idea; parked |
| [IDEA-010](#idea-010-monitoring-logs-and-data-collection) | Idea | Monitoring, logs and data collection | Owner idea; parked |
| [IDEA-011](#idea-011-kubernetes-or-etcd-as-later-candidates) | Idea | Kubernetes or etcd as later candidates | Owner idea; parked |
| [IDEA-012](#idea-012-visualize-the-governed-realm) | Idea | Visualize the governed realm | Owner idea; parked |
| [IDEA-013](#idea-013-gamification) | Idea | Gamification | Owner idea; parked |
| [IDEA-014](#idea-014-shared-evidence-bound-events) | Idea | Shared evidence-bound events | Proposed by the product chat; parked |
| [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent) | Idea | Cockpit with a conversational personal agent | Owner idea; parked |

## Long-term vision

### VIS-001 Cockpit, personal AI and autonomous project organization

- **Status:** Owner statement of 9 October 2026, relayed by the product chat together with the owner's commission of the Government evaluation ([source](sources/concepts-chat-20261009/government-evaluation-mandate-20261009.md)). Detailed planning is deferred ([DEC-011](#dec-011-detailed-planning-of-future-ideas-waits)).
- **Statement:** The big vision includes a cockpit plus a personal AI, and an autonomous project organization („Cockpit plus persönliche AI und autonome Projektorganisation“).
- **Related:**
  - the Government axis ([IDEA-002](#idea-002-government-delegated-model-maintenance), [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis));
  - the cockpit ([IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent));
  - briefings, monitoring and visualization ([IDEA-009](#idea-009-regular-briefings) to [IDEA-014](#idea-014-shared-evidence-bound-events)).

## Decisions, clarifications and endorsed directions

### DEC-001 Delegated realization is central to the product

- **Status:** Accepted. The owner confirmed this reading on 10 October 2026 ([confirmed summary](sources/idea-review-20261010.md#summary-confirmed-by-the-owner)). It matches the [vision](../vision.md) and the [owner description](sources/owner-product-description-20261009.md).
- **Statement:** Markitect is model-first development with delegated realization.
  - People maintain the canonical model.
  - Markitect compiles the model and derives the affected responsibilities, obligations and files.
  - Delegated agent work realizes the change: recursive Managers, independent review, integration, verification and guarded Apply.
  - The delegated work acts as the backend that turns the model into repository content. The deterministic parts exist to make that work trustworthy.
- **Consequences:**
  - Assess Markitect as a development method, not as a linter or a context tool for agent instructions.
  - Removing delegated realization as scope creep contradicts this decision.
  - Separating the method from any particular execution runtime is endorsed but not yet decided ([DEC-010](#dec-010-evaluate-the-method-separately-from-its-execution-runtime), [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method)).

### DEC-002 Trust is measured against today's practice

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 1](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:** The baseline for trust is today's common practice:
  - an agent implements a work item;
  - another AI reviews the pull request;
  - the required human approval is often given almost blind.

  Markitect aims to earn more trust than this baseline through structure, methodology and federalism ([CPT-001](#cpt-001-federalism)). It does not need compiler-grade guarantees to be valuable.
- **Rationale:** In this practice agents often make mistakes and overlook things. They review themselves poorly in the same thread. They easily miss rules and policies when nothing stops them.
- **Consequences:**
  - Claims are phrased as "more reliable than the baseline", never as correctness guarantees.
  - The trust bar is today's practice. Comparative studies still use a strong, well-equipped conventional workflow that is allowed to win, as [Measurement](../measurement.md#quality-first-comparative-evaluation) requires.

### DEC-003 The compiler is a borrowed concept

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 1](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:** "Compiler" names the idea, not a guarantee.
  - Structural compilation of the model is deterministic: types, identities and references. Together with whole-repository coverage ([DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored)), an invalid or incomplete model blocks work before "runtime".
  - Realization by agents is not deterministic. Markitect claims no compiler-equivalent semantic guarantee for it.
  - The owner adds that interpreted languages and bytecode are not necessarily deterministic either.
- **Interpretation (marked):** The owner's sentence about a "real compiler" breaks off without its predicate. The register reads it as follows: a compiler cannot read a naturally described world model, take context into account, fill unspecified gaps, interpret intent and still be deterministic with classical compiler guarantees. See the recorder's note in the source.
- **Consequence:** Trust in realization comes from computed scope, independent review, declared checks and integration verification ([CPT-003](#cpt-003-two-parts-of-the-promise)).

### DEC-004 The Manager hierarchy is core; Government belongs to a later axis

- **Status:** Clarified. On 10 October 2026 the owner confirmed the review's corrected reading, which separates the Government experiment from the Manager hierarchy ([confirmed message](sources/idea-review-20261010.md#summary-confirmed-by-the-owner)). Consistent with the [main-line design for operation scopes](../design/project-world/operation-scopes-and-model-briefings.md) and the [work-item comparison decision](../design/work-item-comparison-20261009.md) of 9 October 2026.
- **Statement:**
  - The first axis, Model → Repository, is part of the core idea. It consists of:
    - the recursive Manager hierarchy;
    - independent review of each Manager's candidate and of each integration result;
    - verification;
    - guarded Apply.
  - The Government idea belongs to a later axis: Idea, Work Item, trigger or support ticket → Model. On that axis, management levels decide model changes within explicit mandates and escalate reserved decisions. Cross-cutting departments (Ressorts) or an instance for unresolved rule interpretation are possible later designs, not prerequisites.
  - The former Government experiment arm added further institutions for cross-cutting assent and dispute resolution. It is not being retested. Its results are not results about the Manager hierarchy.

### DEC-005 Model granularity is the author's responsibility

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 2](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:** How finely the model is written is the developer's decision. A model can be under- or over-specified, just like work items, prompts, documentation and code. Markitect does not impose a granularity. It can support authors with checks, evaluation and recommendations, including AI-assisted ones ([ENH-002](#enh-002-model-quality-linter-and-metrics)).

### DEC-006 Every file is covered or explicitly ignored

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 3](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Current source:**
  - This is enforced through whole-repository coverage, not inside the structural model compiler.
  - With `coverageMode: full`, the `project init` default, an unclassified file fails `project check`, Plan and full Verify.
  - Ignore rules live in `.markitect/ignore.yaml`; the discussion's `.markitectignore` refers to the same concept.
  - A transitional class counts as accounted for but not conforming.
- **Statement:**
  - Every repository file must be accounted for: realized through the model, classified as a Markitect file, or explicitly ignored. Otherwise the project does not "compile" in the owner's sense.
  - One file may realize several definitions.
  - Coverage proves that nothing is unexplained. It does not prove that the realization is correct.

### DEC-007 "More economical models" means capable mid-tier models

- **Status:** Clarified, owner, 10 October 2026 ([owner response, point 4](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:**
  - The cost hypothesis targets capable mid-tier models such as Luna and Sonnet. Haiku is a candidate for cost-sensitive use if it proves sufficient.
  - These models are not weak. The owner distinguishes them from the high-end tier: Fable, Opus and Astra.
- **Current source:** Every inner role runs on Luna. Using Sonnet or Haiku for inner roles requires [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) and [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method).

### DEC-008 The delegated method has not yet been tested

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 5](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:**
  - The value of the delegated method has not yet been measured.
  - The comparative attempts of October 2026 (the work-item, Government and Luna comparisons) were dominated by setup, dependency and environment problems and by harness defects. Examples: Windows paths, CRLF line endings, permissions, the Codex CLI.
  - Those attempts allow no conclusion about the concept, positive or negative. The owner expects these partial problems to be solvable.
- **Recorder's note:** Earlier matched comparisons, such as the [AGENTS.md comparison](../validation/agents-md-vs-markitect.md), evaluated an earlier product without the delegated method. Their findings stand for what they measured, including the observed cost of model upkeep. That cost remains a real risk for [PRM-007](#promises).

### DEC-009 The central evaluation question

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 5](sources/idea-review-20261010.md#owner-response-to-the-five-points)). [Measurement](../measurement.md) owns the procedure.
- **Statement (translated from the owner's words):** When people develop against a model or world view:
  - can it be realized reliably in a repository, and can that be checked?
  - can the repository stay contradiction-free, consistent, uniform and clean?
  - does agentic coding become meaningfully more trustworthy?
  - are rules followed better instead of being overlooked?
  - can people move their attention to deciding and modeling while the code reliably reflects the model?
- **Consequence:** [Measurement](../measurement.md#human-attention-and-delegated-work) already requires repeated, realistic change sequences. The review added an emphasis: the benefit is expected over a series of changes on a long-lived project, not on a single greenfield task. That emphasis is a proposal, not yet an owner decision ([OQ-005](#open-questions)).

### DEC-010 Evaluate the method separately from its execution runtime

- **Status:** The owner endorsed the review's proposal on 10 October 2026: „klingt sehr interessant und sinnvoll“ ("sounds very interesting and sensible"; [follow-up dispositions](sources/idea-review-20261010.md#follow-up-proposals-and-owner-dispositions)). The plan is pending. The product side is [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method); the proposed procedure is in [Measurement](../measurement.md#evaluating-the-method-separately-from-its-runtime).
- **Proposal:**
  - The object of evaluation is the delegated method: computed scope, bounded Managers, independent review, integration, verification and guarded Apply. A particular execution runtime is not the object.
  - A failure of Markitect's own runtime is a product finding. A harness or environment failure is a measurement limitation. Neither is an outcome of the method.
  - The product supplies the method. A study harness chooses and configures the executors the product offers; it does not reimplement Markitect's method. This repeats the existing rule in the [work-item comparison decision](../design/work-item-comparison-20261009.md).

### DEC-011 Detailed planning of future ideas waits

- **Status:** Accepted, owner, 9 October 2026 ([planning boundary](sources/concepts-chat-20261009/discussion-boundaries-20261009.md)). The owner's wording is in `USER-20261009-01`. The product chat restated the hold in the notes it attached to `-02`, `-03`, `-04` and `-06`.
- **Statement:**
  - The owner wants to discuss Markitect as if it were finished. Topics:
    - how it can be used;
    - how it could work in live operation;
    - how it could influence agentic coding;
    - what future it has;
    - how realistic and feasible it is;
    - brainstorming about ways of working and extensions.
  - Detailed planning waits until four conditions hold:
    - the current work packages are done;
    - case-study numbers are available;
    - Main is clean;
    - a usable, stable Markitect version exists.
  - The parked ideas below ([IDEA-002](#idea-002-government-delegated-model-maintenance) to [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent)) are therefore recorded, not planned.

### DEC-012 Government ideas evaluation commissioned

- **Status:** Accepted as a commission, owner, 9 October 2026, relayed by the product chat ([mandate](sources/concepts-chat-20261009/government-evaluation-mandate-20261009.md); Ideas register `EVAL-20261009-001`). The owner has not adopted any result.
- **Statement:**
  - A Government evaluator assesses the functional Government ideas, especially [IDEA-002](#idea-002-government-delegated-model-maintenance) and [IDEA-004](#idea-004-project-goals-and-non-functional-priorities) to [IDEA-008](#idea-008-configurable-local-priorities-at-every-level). It assumes a working Markitect of small scope.
  - The assessment covers classifying the concepts, compatibility with management, pitfalls, sub-problems, solution options and a reasoned feasibility view.
  - Cockpit, UI and gamification are excluded. No product implementation and no new studies.
  - Fixed basis at dispatch: `codex/product-integration-20261009` at `fc6d09a234572c344279a342416475e788435f1f`, explicitly not the then-stale `main`.
  - The evaluation does not replace a real acceptance of that basis („Die Forschung ersetzt keine echte Basisabnahme“).
  - Per the Concepts chat's reading, the commission does not lift the planning hold in [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits).
- **Work item:** GOV-00 in the [backlog](../work-items/backlog.yaml) settles whether and when Government v1 starts.
- **Result location:** The evaluation lives on branch `codex/government-evaluation-20261009`, not on Main. It is not part of this record and not an adopted design. Record the owner's decision on it here once it is made.

### DEC-013 Linux first for tests and the playground

- **Status:** Accepted, owner, 10 October 2026 ([source](sources/roadmap-planning-20261010.md#windows)).
- **Statement:**
  - Tests, CI and the case playground target Linux first.
  - Windows need not be considered for now if it slows the work or causes too many problems. How to handle Windows is decided later.
- **Consequences:**
  - The required pull-request gate runs on Linux. The Windows job is manual or nightly and does not block merges (backlog CI-02).
  - Windows support is revisited as backlog CI-05.

### DEC-014 Compatibility does not drive decisions

- **Status:** Accepted, owner, 10 October 2026 ([source](sources/roadmap-planning-20261010.md#compatibility)).
- **Statement:** Markitect is in active development and nobody else uses it. Compatibility is not a driver for decisions or implementation.
- **Consequences:**
  - Retained-compatibility surfaces may be removed instead of kept. This covers the earlier Project/Domain command tree, the canonical projection alpha and old example formats.
  - Commands and MCP tools may be renamed without aliases.
  - Documents that promise continued support for those surfaces are corrected (backlog ARCH-04, ARCH-09, ARCH-10, CLI-01).
  - Published releases stay immutable. This is a fact about released files, not a compatibility promise.

### DEC-015 A fresh roadmap; earlier plans are inputs

- **Status:** Accepted, owner, 10 October 2026 ([source](sources/roadmap-planning-20261010.md#a-fresh-roadmap-earlier-plans-are-inputs)).
- **Statement:**
  - The roadmap is built fresh from current main, the vision and the owner's topic list.
  - Plans and roadmaps on other branches or in untracked files are likely based on older states. They are not adopted as they stand.
  - Their ideas and concepts are checked separately against current main and the vision, then recorded here (backlog IDEA-01).

### DEC-016 A clean, stable, testable main and uniform structure first

- **Status:** Accepted, owner, 10 October 2026 ([source](sources/roadmap-planning-20261010.md#priority-a-clean-stable-and-testable-main) and [structure](sources/roadmap-planning-20261010.md#structure-and-conventions)).
- **Statement:**
  - The top priority is a clean, stable and testable main.
  - Test runs must check Markitect itself, not provider CLI specifics, permissions, line endings or other environment problems.
  - The repository needs a well-organized structure in which everything has its place and conventions are uniform.
  - An existing convention is kept only where it is sensible; otherwise it is changed and the affected files are moved.
- **Consequences:** The roadmap's Waves 0 to 2 serve this priority before feature extensions.

## Promises

These are the target outcomes Markitect is built to deliver. Each has the status **Target**: it is to be validated under [DEC-009](#dec-009-the-central-evaluation-question), and none is an established result. Sources are the [owner description](sources/owner-product-description-20261009.md) and the [confirmed summary](sources/idea-review-20261010.md#summary-confirmed-by-the-owner).

| ID | Promise |
|---|---|
| PRM-001 | Nothing in the modeled scope is silently forgotten. Affected obligations, files and responsible Managers come from the model, not from an agent's memory, and unknown impact remains visible. This covers declared relationships; dependencies in code are a known gap ([ENH-001](#enh-001-observed-code-dependencies-widen-impact)). |
| PRM-002 | Rules and policies are applied where they matter. They are compiled into the responsible Manager's context and checked by independent reviewers, so they are not overlooked when nothing forces the agent to see them. |
| PRM-003 | The repository stays contradiction-free, consistent, uniform and clean through many changes, so cleanups and refactorings become dependable. |
| PRM-004 | People concentrate on deciding and modeling. Human review moves from code diffs to model changes and escalated decisions while the code reliably reflects the model. |
| PRM-005 | Capable mid-tier models can do most of the delegated work, so routine delegation does not depend on the most expensive models. Lower total cost is a possible secondary benefit, not the primary purpose ([DEC-007](#dec-007-more-economical-models-means-capable-mid-tier-models), [ASM-003](#assumptions-and-hypotheses)). |
| PRM-006 | Software becomes understandable: the model states what the system does and should do, and people can reason over it. |
| PRM-007 | Order pays off: structure and careful definition are rewarded instead of turning into an endless cleanup. |

## Assumptions and hypotheses

| ID | Assumption | Status and origin |
|---|---|---|
| ASM-001 | Agents overlook more relevant files and rules as projects grow. Explicit consistency runs miss things, take ever longer and struggle to resolve contradictions. | Owner experience ([description](sources/owner-product-description-20261009.md)); not measured here |
| ASM-002 | A reviewer with fresh context is more critical than self-review in the same thread, even with the same model. | Owner observation, backed by an Anthropic talk and papers the owner cited without recording references ([point 4](sources/idea-review-20261010.md#owner-response-to-the-five-points)) |
| ASM-003 | Bounded scope and compiled context let capable mid-tier models do most implementation and review work. Total cost can then fall even after coordination and review. | Hypothesis; untested |
| ASM-004 | Abstraction and management structures proven in human organizations transfer to agents. Pairing an implementer with a reviewer at the lowest level mirrors how teams already work today. | Owner hypothesis ([point 4](sources/idea-review-20261010.md#owner-response-to-the-five-points)); not yet properly evaluated for agents |
| ASM-005 | Reviewers that differ from the implementer in model, not only in context, find more defects, because reviewers on the same model share blind spots. | Review proposal, endorsed for evaluation ("Modelle variieren und kombinieren und evaluieren"); to be tested with [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) |

## Concepts

### CPT-001 Federalism

- **Status:** Accepted. The term is the owner's ([point 1](sources/idea-review-20261010.md#owner-response-to-the-five-points)); the structure comes from the [owner description](sources/owner-product-description-20261009.md).
- **Statement:**
  - Responsibility is spread over bounded areas, each with its own Manager, scope and context.
  - Each candidate and each integration result is assessed by an independent reviewer.
  - Parent Managers integrate the results.
  - A global consistency check compares all areas with the canonical model.
  - Trust comes from this structure, not from any single agent being infallible.

### CPT-002 Tests are realizations too

- **Status:** Accepted, owner, 10 October 2026 ([point 1](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:** Tests are one form in which the model is expressed. Agents write them, so they can be as wrong as the code. A green agent-written test is evidence, not proof.
- **Agreed direction:** Review tests against the statements they claim to check ([ENH-003](#enh-003-test-to-statement-binding-and-test-review)). Whether that becomes a variant or is built in is still open.

### CPT-003 Two parts of the promise

- **Status:** Clarified. The owner agreed with the diagnosis behind this split: „Im Grunde hast du mit der Bruchstelle recht“ ("basically you are right about the break point"). The two-part wording comes from the review.
- **Statement:** Keep two claims apart:
  - **Nothing in the modeled scope is overlooked.** This can be computed from declared relationships, coverage and impact. Undeclared dependencies in code are a known gap ([ENH-001](#enh-001-observed-code-dependencies-widen-impact)).
  - **Everything is realized correctly.** This can only be supported by evidence: independent review, declared checks and integration verification. It is never guaranteed.

### CPT-004 The specification stays effective

- **Status:** The concept is clarified: the owner confirmed it on 10 October 2026 as part of the [confirmed summary](sources/idea-review-20261010.md#summary-confirmed-by-the-owner). The mechanism below is a proposal from the original product chat, not an owner decision ([discussion impulse](sources/concepts-chat-20261009/discussion-impulse-20261009-01.md); Concepts note C17).
- **Statement:** The specification is not a one-off feature document. Every change has to cover the new intent and also keep all obligations that still apply in view.
- **Proposed mechanism:** Each affected modeled obligation gets a responsible party and an explicit verification or fulfillment state, so that no work silently drops out of consideration.
- **Related:** [PRM-001](#promises), [CPT-003](#cpt-003-two-parts-of-the-promise), [OQ-006](#open-questions).

## Future enhancements

These entries record agreed or endorsed direction. Recording an enhancement does not schedule it. The [roadmap](../implementation-plan.md) and the [backlog](../work-items/backlog.yaml) own scheduling and status. When an enhancement is scheduled, the entry gains a link to its work item, and the work item links back here.

### ENH-001 Observed code dependencies widen impact

- **Status:** The problem is accepted (owner, 10 October 2026); the approach is open. The owner: „wir sollten das verfolgen und uns hier etwas einfallen lassen“ ("we should pursue this and come up with something"; [follow-up dispositions](sources/idea-review-20261010.md#follow-up-proposals-and-owner-dispositions)).
- **Problem:** Declared mappings do not contain dependencies between files in code: calls, imports and shared types. Suppose one Manager changes a shared function that another Manager's realization calls. The second Manager's obligations can then be affected without appearing in impact.
- **Direction (proposal):**
  - Optional, language-specific tools or plugins observe such dependencies.
  - Their output enters impact as a fixed, digest-bound observed input.
  - Observation may widen impact. It never silently narrows it.
  - Core stays free of inferred source-code semantics ([AGENTS.md](../../AGENTS.md), [architecture](../architecture.md)).
- **Constraint (owner):** These dependencies are specific to each programming language. They cannot be checked the same deterministic way for every kind of file.
- **Work item:** IDEA-02 in the [backlog](../work-items/backlog.yaml).

### ENH-002 Model-quality linter and metrics

- **Status:** Accepted direction, owner, 10 October 2026.
- **Direction:** Report signals that a model is too fine, too coarse or incomplete:
  - blast radius per change;
  - size of a model diff compared with the code diff it causes;
  - files that realize unusually many definitions;
  - statements or obligations without any check;
  - definitions without realizations;
  - ambiguous or unowned areas.

  Output is findings and recommendations, including AI-assisted ones. They are not hard gates by default ([DEC-005](#dec-005-model-granularity-is-the-authors-responsibility)).
- **Work item:** IDEA-03 in the [backlog](../work-items/backlog.yaml).

### ENH-003 Test-to-statement binding and test review

- **Status:** Accepted direction, owner, 10 October 2026. It may become a variant or be built in.
- **Direction:**
  - Bind tests to the statements or checks they realize.
  - Add a review that asks whether a test actually verifies its statement.
  - Report statements whose only evidence is a test that has not been reviewed ([CPT-002](#cpt-002-tests-are-realizations-too)).
- **Work item:** IDEA-04 in the [backlog](../work-items/backlog.yaml).

### ENH-004 Per-role agent configuration and model mixing

- **Status:** Accepted direction, owner, 10 October 2026. A separate configuration for every agent role was already planned.
- **Current source:**
  - `.markitect/runtime.yaml` binds one executor and one reviewer per Manager. Each binding has its own model, plus a reasoning effort for the App Server or model options for the process transport. One verifier is configured globally.
  - `project setup` generates a single profile and uses it for every role: Codex App Server, `gpt-6-luna`, `high` effort ([provider adapters](../provider-adapters.md#inner-role-execution-and-workspace-boundary)).
  - Helpers inherit their parent's configuration.
  - The Manager implements its own slice; there is no separate implementer role.
- **Direction:**
  - Configure model, reasoning effort and provider per role (Manager or implementer, helper, reviewer, verifier), and possibly per Manager.
  - Allow mixing, for example one model implements and another reviews.
  - Evaluate combinations as an experimental factor ([ASM-005](#assumptions-and-hypotheses), [OQ-003](#open-questions)).
- **Work items:** RUN-01 (pull request #92) and PLAY-06 in the [backlog](../work-items/backlog.yaml).

### ENH-005 Exchangeable executor for the delegated method

- **Status:** Endorsed by the owner on 10 October 2026; the design is pending. This is the product side of [DEC-010](#dec-010-evaluate-the-method-separately-from-its-execution-runtime).
- **Problem:** In practice every delegated role runs through one native runtime, the Codex App Server adapter. Runtime and environment failures therefore block both product use and evaluation of the method.
- **Current source:**
  - Run invokes every role through the `projectrun.Invoker` interface. That interface has two transports:
    - a hand-configurable process transport with a versioned JSON protocol (`agent-execution/v1alpha1`);
    - the native Codex App Server transport.
  - Repository end-to-end tests drive Plan, Run, Review, Verify and guarded Apply through a scripted, model-free process executor.
  - What is missing is product surface:
    - setup for executors other than the native one;
    - executors that cannot report provider token usage, such as scripts or people (process executors must report it today);
    - export of work packets;
    - submission of external candidates and review verdicts;
    - plans that do not depend on executor fingerprints.
- **Direction (proposal):**
  - Separate the deterministic method from the execution runtime.
    - The method covers the compiled model, coverage, impact, Manager scopes and briefings, required reviews, candidate validation, Verify and guarded Apply.
    - The runtime covers process launch, sandboxing, provider sessions and recovery of agent turns.
  - Expose a stable executor boundary. Through it, Markitect hands each role a deterministic work packet: scope, context, obligations, required checks and required reviews. It then accepts the resulting candidate and review verdict and validates them with the same rules as today.
  - The native Codex adapter becomes one executor among others. Further executors, for example Claude Code, can be added without changing the method.
  - Authority does not change. Scope checks, fixed snapshots and guarded Apply apply to every executor, and submitted reviews remain AI evidence, not human acceptance.
- **Work items:** RUN-01 (pull request #92), RUN-02 and RUN-03 in the [backlog](../work-items/backlog.yaml).

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [ENH-001](#enh-001-observed-code-dependencies-widen-impact) |
| OQ-002 | Which metrics and thresholds show that a model is too fine or too coarse? | [ENH-002](#enh-002-model-quality-linter-and-metrics), [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility) |
| OQ-003 | Which model tiers suffice for which roles, and does total cost actually fall? | [ASM-003](#assumptions-and-hypotheses), [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) |
| OQ-004 | How much independence does a different reviewer model add beyond fresh context? | [ASM-005](#assumptions-and-hypotheses) |
| OQ-005 | Should evaluation focus on a series of changes on a long-lived project? If so, what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden holdout checks? | [DEC-009](#dec-009-the-central-evaluation-question), [PRM-001](#promises) |
| OQ-006 | How much current-state context does a bounded Manager need, so that limited responsibility does not become artificial blindness? The original product chat's hypothesis: shared contracts, plus current-state context on demand (Concepts note C19). | [CPT-001](#cpt-001-federalism), [CPT-004](#cpt-004-the-specification-stays-effective) |

## Ideas

### IDEA-001 The model as institutional memory for stateless agents

- **Status:** Proposed by the 10 October review; not yet evaluated by the owner.
- **Idea:** In human organizations, management structures work partly because people remember and keep responsibility over time. Agents keep neither. The canonical model can supply this institutional memory and lasting responsibility. That would make a management hierarchy workable for stateless agents and would support [ASM-004](#assumptions-and-hypotheses).

The ideas below come from the owner's contributions of 9 October 2026. The product chat handed them to the Concepts and Ideas chats, which recorded them. They are parked under [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits): recorded for later examination, not planned. Quoted German is the owner's wording unless marked otherwise. "Open" items are questions raised by the Concepts or Ideas chat, not by the owner, unless marked as the owner's. The [sources](sources/README.md) keep the full text and the chats' analysis. In these ideas, "user" means the person who uses Markitect to run a project.

### IDEA-002 Government: delegated model maintenance

- **Status:** Owner idea (Concepts `USER-20261009-02`; notes C20 to C22). The owner describes it as an earlier idea with a relatively immature implementation somewhere on a branch. An evaluation was commissioned ([DEC-012](#dec-012-government-ideas-evaluation-commissioned)). [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) places it on the later axis.
- **Aim (owner):** As much trust in coding agents as possible, and through that as much AI autonomy as possible, without the repository ending up chaotic and hard to follow.
- **Idea:**
  - If the model can be changed and its realization then relatively reliably checked and applied, the user's task becomes maintaining the model. Attention shifts to every change of the model.
  - Work items and technical topics must be interpreted and worked into the model. Problems found while applying the model could leave the user guiding more than before.
  - Hence a "government" maintains and adapts the model:
    - The user is the "president" („Ich bin der Präsident“), receives briefings and is asked only about relevant topics.
    - "Courts" and "offices" look after maintaining and adapting the model.
    - Departments, for example security, architecture, code hygiene, usability and innovation, step in mainly on model changes and raise objections. They then meet in a kind of court.
  - Apply is the executive; a judiciary and a legislature are added. The result would be an organism with high autonomy and adjustability.
- **The owner's open points:**
  - How well it can be realized and what comes out in practice has to be tested.
  - It must be reconciled with the Manager concept.
  - The model must keep a consistently high quality and must not decay.
- **Source:** [government-as-model-governance](sources/concepts-chat-20261009/government-as-model-governance-20261009.md)

### IDEA-003 Degree of freedom per Manager and rule strictness

- **Status:** Owner idea, discussed earlier and restated on 9 October 2026 (Concepts `USER-20261009-01`). Provisional; not an accepted contract.
- **Idea:** Each Manager can be configured with a degree of freedom or something similar, and rules can carry a "strictness" („Strenge“).
- **Current source:** `.markitect/runtime.yaml` already has a strictness setting, as a project default and per Manager. It asks a Manager for additional evidence and counterexamples.
- **Recorder's interpretation:** That setting covers one possible reading of "strictness": how deeply work is checked. The Concepts chat left other readings open.
- **Related:** [IDEA-008](#idea-008-configurable-local-priorities-at-every-level). Source: [discussion-boundaries](sources/concepts-chat-20261009/discussion-boundaries-20261009.md).

### IDEA-004 Project goals and non-functional priorities

- **Status:** Owner idea (Ideas `IDEA-20261009-008`; Concepts C23).
- **Idea:** Each project has a target picture or a prioritization of its non-functional properties, against which trade-offs can be weighed.
- **Open:**
  - Should this be a ranking, a weighting, minimum targets or a mix?
  - How are project-wide and local goals reconciled?
- **Source:** [decision-framework ideas](sources/concepts-chat-20261009/decision-framework-ideas-20261009.md)

### IDEA-005 Reasoned user decisions as project precedents

- **Status:** Owner idea (Ideas `IDEA-20261009-009`; Concepts C24).
- **Idea:** A project-specific "learning mechanism" collects the user's decisions with their reasons as precedents („Entscheidungen vom Benutzer mit den Begründungen“). Later cases can refer to them, like an interpretation of the law for the "judge".
- **Clarification (Concepts):** This means a citable collection of reasoned decisions. It does not mean model training or automatic rule changes.
- **Open:** When does a precedent still apply after the context changes?
- **Source:** [decision-framework ideas](sources/concepts-chat-20261009/decision-framework-ideas-20261009.md)

### IDEA-006 Dispute severity and higher review levels

- **Status:** Owner idea (Ideas `IDEA-20261009-010`; Concepts C25).
- **Idea:** Disagreements carry a severity, and there can be higher court instances.
- **Open:**
  - Who assigns the severity?
  - What may each level decide?
  - How is unnecessary escalation kept small?
- **Source:** [decision-framework ideas](sources/concepts-chat-20261009/decision-framework-ideas-20261009.md)

### IDEA-007 Managers escalate implementation cases

- **Status:** Owner idea (Ideas `IDEA-20261009-011`; Concepts C26).
- **Idea:** While implementing, a Manager can bring a case to court, escalate it or report it.
- **Related:** The [operating methodology](../operating-methodology.md#the-change-and-reconciliation-cycle) already returns unresolved intent, ownership or authority to its owner. This idea adds a case procedure.
- **Source:** [decision-framework ideas](sources/concepts-chat-20261009/decision-framework-ideas-20261009.md)

### IDEA-008 Configurable local priorities at every level

- **Status:** Owner idea (Ideas `IDEA-20261009-012`; Concepts C27).
- **Idea:** Configuration is possible at every level, and decisions are placed within "local" priorities, goals and importance.
- **Open:** How do local settings relate to binding rules from higher levels, and how does the inherited state stay visible?
- **Source:** [decision-framework ideas](sources/concepts-chat-20261009/decision-framework-ideas-20261009.md)

### IDEA-009 Regular briefings

- **Status:** Owner idea (Ideas `IDEA-20261009-013`; Concepts C28).
- **Idea:** Like the daily briefing of the US president, the user in the "president" role regularly receives a summary of the important events and so keeps an overview.
- **Source check:** The real President's Daily Brief is a daily intelligence summary focused on national security, not a general activity overview. For Markitect it stays a broader analogy ([source check](sources/concepts-chat-20261009/operational-overview-source-check-20261009.md)).
- **Open:**
  - What counts as important?
  - What belongs in a daily update and what is reported when it happens?

### IDEA-010 Monitoring, logs and data collection

- **Status:** Owner idea (Ideas `IDEA-20261009-014`; Concepts C29).
- **Idea:** A future, more autonomous operation needs thought about monitoring, logs and the general collection of data.
- **Open:**
  - Which signals help decisions?
  - How are retention, visibility, noise and cost bounded?
- **Source:** [operational overview ideas](sources/concepts-chat-20261009/operational-overview-ideas-20261009.md)

### IDEA-011 Kubernetes or etcd as later candidates

- **Status:** Owner idea (Ideas `IDEA-20261009-015`; Concepts C30). No technology has been chosen.
- **Idea:** Kubernetes, or at least etcd, could be considered for this operation.
- **Source check:** Kubernetes manages containerized workloads and services. etcd is a consistent distributed key-value store for coordination. They serve different purposes, so a concrete need has to be stated first.
- **Source:** [operational overview ideas](sources/concepts-chat-20261009/operational-overview-ideas-20261009.md)

### IDEA-012 Visualize the governed realm

- **Status:** Owner idea (Ideas `IDEA-20261009-016`; Concepts C31).
- **Idea:** A visualization of one's own "realm" („Reich“).
- **Open:** What should it show (responsibilities, activity, decisions, dependencies or health), and for whom?
- **Source:** [operational overview ideas](sources/concepts-chat-20261009/operational-overview-ideas-20261009.md)

### IDEA-013 Gamification

- **Status:** Owner idea (Ideas `IDEA-20261009-017`; Concepts C31 and C33).
- **Idea:** Gamification would be interesting. One later hypothesis: show verified progress ([IDEA-014](#idea-014-shared-evidence-bound-events)).
- **Open:** What would be gamified, toward what behavior, and how can false incentives be avoided?
- **Source:** [operational overview ideas](sources/concepts-chat-20261009/operational-overview-ideas-20261009.md)

### IDEA-014 Shared evidence-bound events

- **Status:** Proposed by the product chat; a discussion impulse, not a decision.
  - The Concepts chat records it as the owner's hypothesis (C32 and C33).
  - The relayed wording is the product chat's own: „Meine zusätzliche Produkthypothese … Das ist ein Diskussionsimpuls, kein Nutzerentscheid“. The Ideas chat records it accordingly.
  - The owner's next contribution opens with „Ja das klingt absolut interessant finde ich“. Whether that refers to this hypothesis is unclear.
- **Idea:** The same evidence-bound events could feed briefings, history, observability and visualization. Gamification could make verified progress visible.
- **Source:** [operational overview source check](sources/concepts-chat-20261009/operational-overview-source-check-20261009.md)

### IDEA-015 Cockpit with a conversational personal agent

- **Status:** Owner idea (Ideas `IDEA-20261009-018`; Concepts C34). Part of [VIS-001](#vis-001-cockpit-personal-ai-and-autonomous-project-organization).
- **Idea:** A cockpit with a chat interface, connected to an AI agent. The agent can search the system for the user, adjust the model, or show the user directly what interests them in the cockpit.
- **Provisional boundary (attribution unclear; treat as a proposal until the owner confirms it):**
  - Chat and cockpit refer to the same model, decisions and evidence.
  - The agent acts only within delegated authority to change the model, and unresolved material decisions stay visible.
  - A chat request alone grants no new authority.

  The Concepts chat records this as a provisional derivation by the owner („vorläufige Nutzerableitung“). The Ideas chat records it as the Concepts chat's interpretation. No verbatim owner wording for it exists.
- **Source:** [personal agent cockpit](sources/concepts-chat-20261009/personal-agent-cockpit-20261009.md)

## Imported collections

On 9 October 2026 two chats collected the owner's ideas outside this record:
- the Concepts chat, as untracked notes;
- the Ideas chat, on a branch.

Both are preserved verbatim in the [sources](sources/README.md). The tables below map their IDs to the entries here. An earlier ID that maps to an existing entry adds no separate decision. The chats' own analysis (assumptions, limits, open questions) stays in the sources.

### Ideas chat register

| Ideas ID | Entry here |
|---|---|
| IDEA-20261009-001 One canonical model | [DEC-001](#dec-001-delegated-realization-is-central-to-the-product), [PRM-001](#promises) |
| IDEA-20261009-002 Typed relationships make ownership and impact visible | [DEC-003](#dec-003-the-compiler-is-a-borrowed-concept), [DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored), [CPT-003](#cpt-003-two-parts-of-the-promise), [PRM-001](#promises) |
| IDEA-20261009-003 Hierarchical agents and independent checks | [DEC-001](#dec-001-delegated-realization-is-central-to-the-product), [CPT-001](#cpt-001-federalism), [ASM-002](#assumptions-and-hypotheses), [ASM-004](#assumptions-and-hypotheses) |
| IDEA-20261009-004 Gap between what software does and should do | [PRM-006](#promises) |
| IDEA-20261009-005 Change intent once, the project follows | [DEC-001](#dec-001-delegated-realization-is-central-to-the-product), [CPT-004](#cpt-004-the-specification-stays-effective) |
| IDEA-20261009-006 Less costly models | [DEC-007](#dec-007-more-economical-models-means-capable-mid-tier-models), [ASM-003](#assumptions-and-hypotheses), [PRM-005](#promises) |
| IDEA-20261009-007 Reward order without a maintenance trap | [PRM-007](#promises), [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility), [ENH-002](#enh-002-model-quality-linter-and-metrics) |
| IDEA-20261009-008 to -012 | [IDEA-004](#idea-004-project-goals-and-non-functional-priorities) to [IDEA-008](#idea-008-configurable-local-priorities-at-every-level), in order |
| IDEA-20261009-013 to -017 | [IDEA-009](#idea-009-regular-briefings) to [IDEA-013](#idea-013-gamification), in order |
| IDEA-20261009-018 Cockpit | [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent) |
| EVAL-20261009-001 Government evaluation | [DEC-012](#dec-012-government-ideas-evaluation-commissioned) |
| HIST-MARKITECT-20261009-FORMAT-001 Representation framing | Kept as a historical finding in the source. The owner rejected framing Markitect as "structures engineering knowledge and makes it usable as Markdown". This matches the first row of the vision's [common misreadings](../vision.md#common-misreadings). |

### Concepts chat notes

| Concepts note | Entry here |
|---|---|
| C01 One canonical source, C15 spec-driven AI-first development | [DEC-001](#dec-001-delegated-realization-is-central-to-the-product) |
| C02 Target world and realization | [PRM-006](#promises), [CPT-003](#cpt-003-two-parts-of-the-promise) |
| C03 Typed model and early structural checking | [DEC-003](#dec-003-the-compiler-is-a-borrowed-concept) |
| C04 What belongs to what | [DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored) |
| C05 Explicit relationships and change impact | [PRM-001](#promises), [ENH-001](#enh-001-observed-code-dependencies-widen-impact) |
| C06 to C09 Namespaces, responsibility areas, recursive Manager hierarchy, bounded goals | [CPT-001](#cpt-001-federalism), [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) |
| C10 Cheaper models | [DEC-007](#dec-007-more-economical-models-means-capable-mid-tier-models), [ASM-003](#assumptions-and-hypotheses) |
| C11 Independent review, four-eyes principle | [ASM-002](#assumptions-and-hypotheses), [ASM-005](#assumptions-and-hypotheses), [CPT-001](#cpt-001-federalism) |
| C12 Global check | [CPT-001](#cpt-001-federalism) |
| C13 Cleanup and refactoring | [PRM-003](#promises) |
| C14 Trust through verifiable method | [DEC-002](#dec-002-trust-is-measured-against-todays-practice), [PRM-004](#promises) |
| C16 Order as lasting benefit | [PRM-007](#promises) |
| C17 New intent and preserving obligations | [CPT-004](#cpt-004-the-specification-stays-effective) |
| C18 Granularity without copying the code | [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility), [OQ-002](#open-questions) |
| C19 Bounded responsibility with sufficient context | [OQ-006](#open-questions) |
| C20 to C22 Model maintenance and Government | [IDEA-002](#idea-002-government-delegated-model-maintenance) |
| C23 to C27 Decision basis and escalation | [IDEA-004](#idea-004-project-goals-and-non-functional-priorities) to [IDEA-008](#idea-008-configurable-local-priorities-at-every-level), in order |
| C28 to C31 Briefing, monitoring, technology, visualization | [IDEA-009](#idea-009-regular-briefings) to [IDEA-013](#idea-013-gamification) |
| C32 and C33 Evidence-bound events, verified progress | [IDEA-014](#idea-014-shared-evidence-bound-events), [IDEA-013](#idea-013-gamification) |
| C34 Cockpit | [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent) |

The Concepts chat's references for owner contributions map as follows:
- `USER-20261009-01` → [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits), [IDEA-003](#idea-003-degree-of-freedom-per-manager-and-rule-strictness)
- `-02` → [IDEA-002](#idea-002-government-delegated-model-maintenance)
- `-03` → [IDEA-004](#idea-004-project-goals-and-non-functional-priorities) to [IDEA-008](#idea-008-configurable-local-priorities-at-every-level)
- `-04` → [IDEA-009](#idea-009-regular-briefings) to [IDEA-013](#idea-013-gamification)
- `-05` → [IDEA-014](#idea-014-shared-evidence-bound-events); this is the product chat's source check and hypothesis, not an owner contribution
- `-06` → [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent)
- `-07` → [VIS-001](#vis-001-cockpit-personal-ai-and-autonomous-project-organization), [DEC-012](#dec-012-government-ideas-evaluation-commissioned)
