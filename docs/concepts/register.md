# Concept register

This register records Markitect's product-level decisions, clarifications, endorsed directions, promises, assumptions, concepts, future enhancements, ideas and open questions. Each entry has a status and an origin. "Owner" means Markitect's product owner. The [concept record guide](README.md) defines entry types, statuses and the rules for changing entries. The [vision](../vision.md) presents the thesis these entries support.

## Index

| ID | Type | Title | Status |
|---|---|---|---|
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
| [PRM-001 to PRM-007](#promises) | Promise | Target outcomes | Target |
| [ASM-001 to ASM-005](#assumptions-and-hypotheses) | Assumption | Assumptions and hypotheses | See entries |
| [CPT-001](#cpt-001-federalism) | Concept | Federalism | Accepted |
| [CPT-002](#cpt-002-tests-are-realizations-too) | Concept | Tests are realizations too | Accepted |
| [CPT-003](#cpt-003-two-parts-of-the-promise) | Concept | Two parts of the promise | Clarified |
| [ENH-001](#enh-001-observed-code-dependencies-widen-impact) | Enhancement | Observed code dependencies widen impact | Accepted problem; approach open |
| [ENH-002](#enh-002-model-quality-linter-and-metrics) | Enhancement | Model-quality linter and metrics | Accepted direction |
| [ENH-003](#enh-003-test-to-statement-binding-and-test-review) | Enhancement | Test-to-statement binding and test review | Accepted direction |
| [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) | Enhancement | Per-role agent configuration and model mixing | Accepted direction |
| [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method) | Enhancement | Exchangeable executor for the delegated method | Endorsed; design pending |
| [OQ-001 to OQ-005](#open-questions) | Open question | Open questions | Open |
| [IDEA-001](#idea-001-the-model-as-institutional-memory-for-stateless-agents) | Idea | The model as institutional memory for stateless agents | Proposed |

## Decisions, clarifications and endorsed directions

### DEC-001 Delegated realization is central to the product

- **Status:** Accepted. The owner confirmed this reading on 10 October 2026 ([confirmed summary](sources/2026-10-10-idea-review.md#summary-confirmed-by-the-owner)). It matches the [vision](../vision.md) and the [owner description](sources/2026-10-09-owner-product-description.md).
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

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 1](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
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

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 1](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
- **Statement:** "Compiler" names the idea, not a guarantee.
  - Structural compilation of the model is deterministic: types, identities and references. Together with whole-repository coverage ([DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored)), an invalid or incomplete model blocks work before "runtime".
  - Realization by agents is not deterministic. Markitect claims no compiler-equivalent semantic guarantee for it.
  - The owner adds that interpreted languages and bytecode are not necessarily deterministic either.
- **Interpretation (marked):** The owner's sentence about a "real compiler" breaks off without its predicate. The register reads it as follows: a compiler cannot read a naturally described world model, take context into account, fill unspecified gaps, interpret intent and still be deterministic with classical compiler guarantees. See the recorder's note in the source.
- **Consequence:** Trust in realization comes from computed scope, independent review, declared checks and integration verification ([CPT-003](#cpt-003-two-parts-of-the-promise)).

### DEC-004 The Manager hierarchy is core; Government belongs to a later axis

- **Status:** Clarified. On 10 October 2026 the owner confirmed the review's corrected reading, which separates the Government experiment from the Manager hierarchy ([confirmed message](sources/2026-10-10-idea-review.md#summary-confirmed-by-the-owner)). Consistent with the [main-line design for operation scopes](../design/project-world/operation-scopes-and-model-briefings.md) and the [work-item comparison decision](../design/work-item-comparison-20261009.md) of 9 October 2026.
- **Statement:**
  - The first axis, Model → Repository, is part of the core idea. It consists of:
    - the recursive Manager hierarchy;
    - independent review of each Manager's candidate and of each integration result;
    - verification;
    - guarded Apply.
  - The Government idea belongs to a later axis: Idea, Work Item, trigger or support ticket → Model. On that axis, management levels decide model changes within explicit mandates and escalate reserved decisions. Cross-cutting departments (Ressorts) or an instance for unresolved rule interpretation are possible later designs, not prerequisites.
  - The former Government experiment arm added further institutions for cross-cutting assent and dispute resolution. It is not being retested. Its results are not results about the Manager hierarchy.

### DEC-005 Model granularity is the author's responsibility

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 2](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
- **Statement:** How finely the model is written is the developer's decision. A model can be under- or over-specified, just like work items, prompts, documentation and code. Markitect does not impose a granularity. It can support authors with checks, evaluation and recommendations, including AI-assisted ones ([ENH-002](#enh-002-model-quality-linter-and-metrics)).

### DEC-006 Every file is covered or explicitly ignored

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 3](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
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

- **Status:** Clarified, owner, 10 October 2026 ([owner response, point 4](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
- **Statement:**
  - The cost hypothesis targets capable mid-tier models such as Luna and Sonnet. Haiku is a candidate for cost-sensitive use if it proves sufficient.
  - These models are not weak. The owner distinguishes them from the high-end tier: Fable, Opus and Astra.
- **Current source:** Every inner role runs on Luna. Using Sonnet or Haiku for inner roles requires [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) and [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method).

### DEC-008 The delegated method has not yet been tested

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 5](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
- **Statement:**
  - The value of the delegated method has not yet been measured.
  - The comparative attempts of October 2026 (the work-item, Government and Luna comparisons) were dominated by setup, dependency and environment problems and by harness defects. Examples: Windows paths, CRLF line endings, permissions, the Codex CLI.
  - Those attempts allow no conclusion about the concept, positive or negative. The owner expects these partial problems to be solvable.
- **Recorder's note:** Earlier matched comparisons, such as the [AGENTS.md comparison](../validation/agents-md-vs-markitect.md), evaluated an earlier product without the delegated method. Their findings stand for what they measured, including the observed cost of model upkeep. That cost remains a real risk for [PRM-007](#promises).

### DEC-009 The central evaluation question

- **Status:** Accepted, owner, 10 October 2026 ([owner response, point 5](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)). [Measurement](../measurement.md) owns the procedure.
- **Statement (translated from the owner's words):** When people develop against a model or world view:
  - can it be realized reliably in a repository, and can that be checked?
  - can the repository stay contradiction-free, consistent, uniform and clean?
  - does agentic coding become meaningfully more trustworthy?
  - are rules followed better instead of being overlooked?
  - can people move their attention to deciding and modeling while the code reliably reflects the model?
- **Consequence:** [Measurement](../measurement.md#human-attention-and-delegated-work) already requires repeated, realistic change sequences. The review added an emphasis: the benefit is expected over a series of changes on a long-lived project, not on a single greenfield task. That emphasis is a proposal, not yet an owner decision ([OQ-005](#open-questions)).

### DEC-010 Evaluate the method separately from its execution runtime

- **Status:** The owner endorsed the review's proposal on 10 October 2026: „klingt sehr interessant und sinnvoll“ ("sounds very interesting and sensible"; [follow-up dispositions](sources/2026-10-10-idea-review.md#follow-up-proposals-and-owner-dispositions)). The plan is pending. The product side is [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method); the proposed procedure is in [Measurement](../measurement.md#evaluating-the-method-separately-from-its-runtime).
- **Proposal:**
  - The object of evaluation is the delegated method: computed scope, bounded Managers, independent review, integration, verification and guarded Apply. A particular execution runtime is not the object.
  - A failure of Markitect's own runtime is a product finding. A harness or environment failure is a measurement limitation. Neither is an outcome of the method.
  - The product supplies the method. A study harness chooses and configures the executors the product offers; it does not reimplement Markitect's method. This repeats the existing rule in the [work-item comparison decision](../design/work-item-comparison-20261009.md).

## Promises

These are the target outcomes Markitect is built to deliver. Each has the status **Target**: it is to be validated under [DEC-009](#dec-009-the-central-evaluation-question), and none is an established result. Sources are the [owner description](sources/2026-10-09-owner-product-description.md) and the [confirmed summary](sources/2026-10-10-idea-review.md#summary-confirmed-by-the-owner).

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
| ASM-001 | Agents overlook more relevant files and rules as projects grow. Explicit consistency runs miss things, take ever longer and struggle to resolve contradictions. | Owner experience ([description](sources/2026-10-09-owner-product-description.md)); not measured here |
| ASM-002 | A reviewer with fresh context is more critical than self-review in the same thread, even with the same model. | Owner observation, backed by an Anthropic talk and papers the owner cited without recording references ([point 4](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)) |
| ASM-003 | Bounded scope and compiled context let capable mid-tier models do most implementation and review work. Total cost can then fall even after coordination and review. | Hypothesis; untested |
| ASM-004 | Abstraction and management structures proven in human organizations transfer to agents. Pairing an implementer with a reviewer at the lowest level mirrors how teams already work today. | Owner hypothesis ([point 4](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)); not yet properly evaluated for agents |
| ASM-005 | Reviewers that differ from the implementer in model, not only in context, find more defects, because reviewers on the same model share blind spots. | Review proposal, endorsed for evaluation ("Modelle variieren und kombinieren und evaluieren"); to be tested with [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) |

## Concepts

### CPT-001 Federalism

- **Status:** Accepted. The term is the owner's ([point 1](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)); the structure comes from the [owner description](sources/2026-10-09-owner-product-description.md).
- **Statement:**
  - Responsibility is spread over bounded areas, each with its own Manager, scope and context.
  - Each candidate and each integration result is assessed by an independent reviewer.
  - Parent Managers integrate the results.
  - A global consistency check compares all areas with the canonical model.
  - Trust comes from this structure, not from any single agent being infallible.

### CPT-002 Tests are realizations too

- **Status:** Accepted, owner, 10 October 2026 ([point 1](sources/2026-10-10-idea-review.md#owner-response-to-the-five-points)).
- **Statement:** Tests are one form in which the model is expressed. Agents write them, so they can be as wrong as the code. A green agent-written test is evidence, not proof.
- **Agreed direction:** Review tests against the statements they claim to check ([ENH-003](#enh-003-test-to-statement-binding-and-test-review)). Whether that becomes a variant or is built in is still open.

### CPT-003 Two parts of the promise

- **Status:** Clarified. The owner agreed with the diagnosis behind this split: „Im Grunde hast du mit der Bruchstelle recht“ ("basically you are right about the break point"). The two-part wording comes from the review.
- **Statement:** Keep two claims apart:
  - **Nothing in the modeled scope is overlooked.** This can be computed from declared relationships, coverage and impact. Undeclared dependencies in code are a known gap ([ENH-001](#enh-001-observed-code-dependencies-widen-impact)).
  - **Everything is realized correctly.** This can only be supported by evidence: independent review, declared checks and integration verification. It is never guaranteed.

## Future enhancements

These entries record agreed or endorsed direction. Recording an enhancement does not schedule it. The [roadmap](../implementation-plan.md) and the [Product Readiness backlog](../work-items/product-readiness/backlog.yaml) own scheduling and status. When an enhancement is scheduled, the entry gains a link to its work item, and the work item links back here.

### ENH-001 Observed code dependencies widen impact

- **Status:** The problem is accepted (owner, 10 October 2026); the approach is open. The owner: „wir sollten das verfolgen und uns hier etwas einfallen lassen“ ("we should pursue this and come up with something"; [follow-up dispositions](sources/2026-10-10-idea-review.md#follow-up-proposals-and-owner-dispositions)).
- **Problem:** Declared mappings do not contain dependencies between files in code: calls, imports and shared types. Suppose one Manager changes a shared function that another Manager's realization calls. The second Manager's obligations can then be affected without appearing in impact.
- **Direction (proposal):**
  - Optional, language-specific tools or plugins observe such dependencies.
  - Their output enters impact as a fixed, digest-bound observed input.
  - Observation may widen impact. It never silently narrows it.
  - Core stays free of inferred source-code semantics ([AGENTS.md](../../AGENTS.md), [architecture](../architecture.md)).
- **Constraint (owner):** These dependencies are specific to each programming language. They cannot be checked the same deterministic way for every kind of file.

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

### ENH-003 Test-to-statement binding and test review

- **Status:** Accepted direction, owner, 10 October 2026. It may become a variant or be built in.
- **Direction:**
  - Bind tests to the statements or checks they realize.
  - Add a review that asks whether a test actually verifies its statement.
  - Report statements whose only evidence is a test that has not been reviewed ([CPT-002](#cpt-002-tests-are-realizations-too)).

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

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [ENH-001](#enh-001-observed-code-dependencies-widen-impact) |
| OQ-002 | Which metrics and thresholds show that a model is too fine or too coarse? | [ENH-002](#enh-002-model-quality-linter-and-metrics), [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility) |
| OQ-003 | Which model tiers suffice for which roles, and does total cost actually fall? | [ASM-003](#assumptions-and-hypotheses), [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) |
| OQ-004 | How much independence does a different reviewer model add beyond fresh context? | [ASM-005](#assumptions-and-hypotheses) |
| OQ-005 | Should evaluation focus on a series of changes on a long-lived project? If so, what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden holdout checks? | [DEC-009](#dec-009-the-central-evaluation-question), [PRM-001](#promises) |

## Ideas

### IDEA-001 The model as institutional memory for stateless agents

- **Status:** Proposed by the 10 October review; not yet evaluated by the owner.
- **Idea:** In human organizations, management structures work partly because people remember and keep responsibility over time. Agents keep neither. The canonical model can supply this institutional memory and lasting responsibility. That would make a management hierarchy workable for stateless agents and would support [ASM-004](#assumptions-and-hypotheses).
