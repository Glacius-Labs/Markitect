# Concept register

This register records Markitect's long-term vision, product-level decisions, clarifications, endorsed directions, promises, assumptions, concepts, future enhancements, ideas and open questions. Each entry has a status and an origin. "Owner" means Markitect's product owner. The [concept record guide](README.md) defines entry types, statuses and the rules for changing entries. The [vision](../vision.md) presents the thesis these entries support. Earlier collections from the Concepts and Ideas chats are preserved in the [sources](sources/README.md) and mapped [below](#imported-collections).

**Current Apply reading:** [DEC-064](#dec-064-checked-recursive-apply-passes-and-manager-teams) and the [accepted workflow](../design/canonical-model/manager-apply-workflow.md) own Apply. Main #530 integrated the base Pass process; the [first real runs](../validation/apply-pass-process-20261020.md) include a false-success Depot and blocked Haiku work. The owner has adopted [evidence-driven hardening](../design/canonical-model/manager-apply-workflow.md#evidence-driven-hardening-adopted-20-october-2026): active counterexamples, tool-produced receipts, challenged child evidence, precise dependencies and honest unresolved policy. The [new Claude handoff](../development/claude-apply-hardening-handoff-20261020.md) prepares implementation and bounded evaluation; these refinements are not new runtime evidence. The runtime guide describes the base implementation; the canonical self-model includes adopted target intent. Dated decisions and old-core results keep their scope.

## Index

Earlier work-item IDs and wave names in this register are historical references. Their recorded decisions retain their status; the [current delivery plan](../implementation-plan.md) alone assigns rebuild scope and dependencies.

The owner's [origin account of 15 October 2026](sources/owner-origin-20261015.md) adds the concrete adopting-project experience behind these entries. [Origin and rationale](origin-and-rationale.md) connects its problems, mechanisms and assumptions to the existing IDs without changing their status. The example Core/Module/Product rules belong to that adopting project.

| ID | Type | Title | Status |
|---|---|---|---|
| [VIS-001](#vis-001-cockpit-personal-ai-and-autonomous-project-organization) | Vision | Cockpit, personal AI and autonomous project organization | Owner statement; scoped cockpit concept and Claude handoff commissioned 20 October 2026; broader extensions deferred |
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
| [DEC-017](#dec-017-command-model-top-level-verbs-from-one-verb-table) | Decision | Command model: top-level verbs from one verb table | Accepted |
| [DEC-018](#dec-018-the-knowledge-graph-explains-impact) | Decision | The knowledge graph explains impact | Accepted |
| [DEC-019](#dec-019-government-code-starts-after-wave-2) | Decision | Government code starts after Wave 2 | Accepted |
| [DEC-020](#dec-020-markitect-uses-any-library-that-helps) | Decision | Markitect uses any library that helps | Accepted |
| [DEC-021](#dec-021-purpose-and-meaning-free-edits-route-narrowly) | Decision | Purpose and meaning-free edits route narrowly | Accepted |
| [DEC-022](#dec-022-decisions-are-visible-and-route-through-their-subject) | Decision | Decisions are visible and route through their subject | Accepted |
| [DEC-023](#dec-023-impact-marks-change-and-context-without-dropping-either) | Decision | Impact marks change and context without dropping either | Accepted |
| [DEC-024](#dec-024-first-reliable-apply-rebuild-and-coordinated-start) | Endorsed direction | First reliable Apply rebuild and coordinated start | Endorsed; implementation commissioned |
| [DEC-025](#dec-025-package-managers-organize-realization) | Decision | Package Managers organize realization | Accepted |
| [DEC-026](#dec-026-apply-comes-first-and-changes-only-artifacts) | Decision | Apply comes first and changes only artifacts | Accepted |
| [DEC-027](#dec-027-strict-links-with-an-optional-optimizer) | Decision | Strict links with an optional Optimizer | Accepted |
| [DEC-028](#dec-028-context-follows-need-to-know-and-starts-simple) | Decision | Context follows need to know and starts simple | Accepted |
| [DEC-029](#dec-029-instructions-enter-through-the-root) | Decision | Instructions enter through the root | Accepted |
| [DEC-030](#dec-030-agent-communication-is-traceable) | Decision | Agent communication is traceable | Accepted |
| [DEC-031](#dec-031-the-first-target-is-two-applies-of-the-library-example) | Decision | The first target is two Applies of the library example | Accepted |
| [DEC-032](#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility) | Decision | Quality, simplicity and stability over effort and compatibility | Accepted |
| [DEC-033](#dec-033-binding-intent-lives-only-in-the-model) | Decision | Binding intent lives only in the model | Accepted |
| [DEC-034](#dec-034-removed-intent-removes-its-realization) | Decision | Removed intent removes its realization | Accepted |
| [DEC-035](#dec-035-committed-model-in-owner-commit-accepts-the-result) | Decision | Committed model in, owner commit accepts the result | Accepted |
| [DEC-036](#dec-036-adopted-design-refinements-for-the-first-apply) | Decision | Adopted design refinements for the first Apply | Accepted |
| [DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers) | Decision | Agent sessions and checks run in Linux containers | Accepted |
| [DEC-038](#dec-038-learn-through-complete-workflows) | Decision | Learn through complete workflows | Accepted |
| [DEC-039](#dec-039-deterministic-manager-process) | Decision | Deterministic Manager process | Accepted |
| [DEC-040](#dec-040-concept-freeze-until-the-core-runs) | Decision | Concept freeze until the core runs | Accepted |
| [DEC-041](#dec-041-core-protocol-choices) | Decision | Core protocol choices | Accepted |
| [DEC-042](#dec-042-orchestrator-names-the-deterministic-coordination-component) | Decision | Orchestrator names the deterministic coordination component | Accepted |
| [DEC-043](#dec-043-project-teams-review-arbitration-and-product-defaults) | Decision | Project teams, review arbitration and product defaults | Accepted; implementation pending |
| [DEC-044](#dec-044-commands-reuse-the-manager-organization) | Decision | Commands reuse the Manager organization | Accepted direction; Apply and a first Extract implemented |
| [DEC-045](#dec-045-managers-and-reviewers-derive-technical-risks) | Clarification | Managers and reviewers derive technical risks | Accepted |
| [DEC-046](#dec-046-a-withheld-pass-keeps-its-recorded-expectations) | Decision | A withheld pass keeps its recorded expectations | Accepted |
| [DEC-047](#dec-047-recorded-review-cases-stand-until-their-inputs-change) | Decision | Recorded review cases stand until their inputs change | Accepted |
| [DEC-048](#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose) | Decision | Recorded review cases stand for every delivery of a purpose | Accepted |
| [DEC-049](#dec-049-review-cases-are-recorded-before-the-work-they-assess) | Decision | Review cases are recorded before the work they assess | Accepted |
| [DEC-050](#dec-050-an-accepted-delivery-restarts-the-rejection-count) | Decision | An accepted delivery restarts the rejection count | Accepted |
| [DEC-051](#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) | Decision | Agents get tasks; the Orchestrator owns the process | Accepted |
| [DEC-052](#dec-052-implement-then-review-no-explore-phase) | Decision | Implement then review: no Explore phase | Accepted |
| [DEC-053](#dec-053-owner-decisions-bind-the-whole-run) | Decision | Owner decisions bind the whole run | Accepted |
| [DEC-054](#dec-054-a-rejection-that-finds-fewer-gaps-does-not-count) | Decision | A rejection that finds fewer gaps does not count | Accepted |
| [DEC-055](#dec-055-a-submission-must-pass-its-required-checks) | Decision | A submission must pass its required checks | Accepted |
| [DEC-056](#dec-056-recursive-reconciliation-with-a-shared-workspace) | Decision | Recursive reconciliation with a shared workspace | Accepted; implemented (PR #463) |
| [DEC-057](#dec-057-three-fundamental-starting-situations) | Clarification | Three starting situations, two processes: Apply and Extract | Accepted high-level direction; first Extract implemented |
| [DEC-058](#dec-058-apply-always-executes-before-independent-verify) | Decision | Apply always executes before independent Verify | Accepted; implemented (PR #472) |
| [DEC-059](#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle) | Decision | Enforced responsibilities, build start and an outside oracle | Accepted; implemented (PR #472) |
| [DEC-060](#dec-060-owners-realize-others-use-parents-settle-cross-area-matters) | Decision | Owners realize, others use; parents settle cross-area matters | Accepted; implemented |
| [DEC-061](#dec-061-what-an-extracted-model-contains) | Decision | What an extracted model contains | Accepted; implemented |
| [DEC-062](#dec-062-start-stand-drift-is-the-owners-first-work) | Decision | Start-stand drift is the owner's first work | Superseded by DEC-064; removed from the implementation |
| [DEC-063](#dec-063-the-model-is-authoritative-defects-escalate) | Decision | The model is authoritative; defects escalate | Amended by DEC-064; model authority retained |
| [DEC-064](#dec-064-checked-recursive-apply-passes-and-manager-teams) | Decision | Checked recursive Apply passes and Manager teams | Base integrated; evidence-driven hardening adopted, implementation and new evidence pending |
| [DEC-065](#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration) | Decision | Work through the cockpit: agent-proposed model changes and configuration | Accepted by the owner; implementation started |
| [PRM-001 to PRM-007](#promises) | Promise | Target outcomes | Target |
| [ASM-001 to ASM-006](#assumptions-and-hypotheses) | Assumption | Assumptions and hypotheses | See entries |
| [CPT-001](#cpt-001-federalism) | Concept | Federalism | Accepted |
| [CPT-002](#cpt-002-tests-are-realizations-too) | Concept | Tests are realizations too | Accepted |
| [CPT-003](#cpt-003-two-parts-of-the-promise) | Concept | Two parts of the promise | Clarified |
| [CPT-004](#cpt-004-the-specification-stays-effective) | Concept | The specification stays effective | Clarified; mechanism proposed |
| [ENH-001](#enh-001-observed-code-dependencies-widen-impact) | Enhancement | Observed code dependencies widen impact | Accepted problem; approach open |
| [ENH-002](#enh-002-model-quality-linter-and-metrics) | Enhancement | Model-quality linter and metrics | Accepted direction |
| [ENH-003](#enh-003-test-to-statement-binding-and-test-review) | Enhancement | Test-to-statement binding and test review | Accepted direction |
| [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) | Enhancement | Per-role agent configuration and model mixing | Accepted direction |
| [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method) | Enhancement | Exchangeable executor for the delegated method | Endorsed; increment 1 merged |
| [ENH-006](#enh-006-configurable-teams-for-perspectives-and-four-eyes) | Enhancement | Configurable teams for perspectives and four eyes | Selected DEC-043 slice accepted; other variants deferred |
| [OQ-001 to OQ-010](#open-questions) | Open question | Open questions | Open |
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
| [IDEA-016](#idea-016-optional-offline-claim-and-evidence-classifier) | Idea | Optional offline claim and evidence classifier | Proposed (research R01) |
| [IDEA-017](#idea-017-operational-check-through-the-managers) | Idea | Operational check through the Managers | Owner idea; deferred |

## Long-term vision

### VIS-001 Cockpit, personal AI and autonomous project organization

- **Status:** Owner statement of 9 October 2026, relayed by the product chat together with the owner's commission of the Government evaluation ([source](sources/concepts-chat-20261009/government-evaluation-mandate-20261009.md)). The owner commissioned a scoped cockpit concept and Claude handoff on 20 October 2026 ([source](sources/cockpit-commission-20261020.md)); broader extensions retain the [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits) deferral.
- **Scoped commission, 20 October 2026:** the [cockpit concept](../design/cockpit.md) records a local project/Run surface with Manager organization, causal waits, timeline, details, replay, messages and animated directional arrows. Claude owns its implementation and visual design through the [handoff](../development/claude-cockpit-handoff-20261020.md). This Main delivery documents the direction; it starts no implementation session or provider run and does not adopt personal AI, Government or a new Apply process.
- **Statement:** The big vision includes a cockpit plus a personal AI, and an autonomous project organization („Cockpit plus persönliche AI und autonome Projektorganisation“).
- **Owner, 16 October 2026:** the desired working life is to model, develop ideas and concepts and set rules, and to know reliably that the organization realizes them cleanly and consistently, develops features, finds and fixes bugs and keeps the project tidy („Ich möchte meine Zeit damit verbringen, zu modellieren, Ideen und Konzepte auszuarbeiten, Regeln aufzustellen.“; [source](sources/organization-and-trust-20261016.md#desired-working-life)). The [working-day reflection](reflections/desired-working-day-20261016.md) develops it.
- **Related:**
  - the Government axis ([IDEA-002](#idea-002-government-delegated-model-maintenance), [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis));
  - the cockpit ([IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent));
  - briefings, monitoring and visualization ([IDEA-009](#idea-009-regular-briefings) to [IDEA-014](#idea-014-shared-evidence-bound-events)).
- **Planned extensions:** cockpit, Government and approval reuse are recorded in the [north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions).

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
- **First-Apply clarification:** This consequence and [DEC-010](#dec-010-evaluate-the-method-separately-from-its-execution-runtime) retain earlier "computed scope" wording. Under [DEC-039](#dec-039-deterministic-manager-process), compilation supplies explicit obligations and responsibilities while every local Manager participates. Omitting out-of-scope Managers conflicts with that decision; Orchestrator-verified reuse is a separate proposal. Both remain unresolved under the [core's scope questions](../history/apply-design-before-dec-056/manager-apply-core.md#not-settled-here). The earlier wording grants no exception to first-Apply participation.

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
- **Historical source snapshot (10 October 2026):** The predecessor engine checked coverage separately from its model compiler and supported `coverageMode: full`. The current source compiles canonical v1 and prepares exact repository accounting; it has no legacy `init`, Plan or Verify commands. See the [workflow](../project-workflow.md) and [delivery plan](../implementation-plan.md) for current behavior and sequence.
- **Statement:**
  - Every repository file must be accounted for: realized through the model, classified as a Markitect file, or explicitly ignored. Otherwise the project does not "compile" in the owner's sense.
  - One file may realize several definitions.
  - Coverage proves that nothing is unexplained. It does not prove that the realization is correct.

### DEC-007 "More economical models" means capable mid-tier models

- **Status:** Clarified, owner, 10 October 2026 ([owner response, point 4](sources/idea-review-20261010.md#owner-response-to-the-five-points)).
- **Statement:**
  - The cost hypothesis targets capable mid-tier models such as Luna and Sonnet. Haiku is a candidate for cost-sensitive use if it proves sufficient.
  - These models are not weak. The owner distinguishes them from the high-end tier: Fable, Opus and Astra.
- **Historical source snapshot (10 October 2026):** The predecessor engine defaulted inner roles to Luna and allowed process executors. Those settings and that runtime are not the current source. Current implementation and planned work are owned by the [delivery plan](../implementation-plan.md).

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
- **Owner clarification, 17 October 2026:** Human attention is the bottleneck. Additional lower-cost Agents are useful when they reduce overlooked obligations and the developer's inspection and correction prompts; Agent count alone is not the problem. Elapsed time is separate, with parallelism as a later implementation lever. The primary comparison is ordinary documentation with the same content as the canonical model, given to one Agent to read and implement. The [source](sources/human-attention-and-baseline-20261017.md) preserves the wording; [Measurement](../measurement.md#quality-first-comparative-evaluation) owns the procedure. This clarification changes neither the sequential first Apply nor the feasibility gates.

### DEC-010 Evaluate the method separately from its execution runtime

- **Status:** The owner endorsed the review's proposal on 10 October 2026: „klingt sehr interessant und sinnvoll“ ("sounds very interesting and sensible"; [follow-up dispositions](sources/idea-review-20261010.md#follow-up-proposals-and-owner-dispositions)). The plan is pending. The product side is [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method); the proposed procedure is in [Measurement](../measurement.md#evaluating-the-method-separately-from-its-runtime).
- **Proposal:**
  - The object of evaluation is the delegated method: computed scope, bounded Managers, independent review, integration, verification and guarded Apply. A particular execution runtime is not the object.
  - A failure of Markitect's own runtime is a product finding. A harness or environment failure is a measurement limitation. Neither is an outcome of the method.
  - The product supplies the method. A study harness chooses and configures the executors the product offers; it does not reimplement Markitect's method. This repeats the existing rule in the [work-item comparison decision](../design/work-item-comparison-20261009.md).

### DEC-011 Detailed planning of future ideas waits

- **Status:** Accepted, owner, 9 October 2026 ([planning boundary](sources/concepts-chat-20261009/discussion-boundaries-20261009.md)). For Government, [DEC-019](#dec-019-government-code-starts-after-wave-2) supersedes these conditions. The owner's wording is in `USER-20261009-01`. The product chat restated the hold in the notes it attached to `-02`, `-03`, `-04` and `-06`.
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
- **Historical work item:** GOV-00 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml) settles whether and when Government v1 starts.
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

### DEC-017 Command model: top-level verbs from one verb table

- **Status:** Accepted, owner, 10 October 2026. The owner approved the roadmap's recommendations for CLI-01 ([roadmap](../implementation-plan.md#owner-decisions)). D1, D4 and D6 follow from [DEC-014](#dec-014-compatibility-does-not-drive-decisions).
- **Statement:**
  - **D2:** The `project` noun is dropped. Commands are top-level verbs in the style of kubectl, terraform and git, for example `markitect plan` and `markitect apply`, plus convenience verbs.
  - **One verb table:** a single table drives CLI help, dispatch and MCP registration. MCP tools carry the verb names without a `project_` prefix, and MCP fields are the same arguments as the CLI flags.
  - **D3:** One brownfield path remains, as positional `adopt` stages.
  - **D5:** `--write` only persists. Starting agents needs a separate `--execute`.
  - **D7:** `status` shows a project overview by default, and `status RUN` shows one run.
  - **D8:** MCP gets a read-only server mode.
  - **D9:** Release tooling lives in the separate release binary.
- **Historical work items:** CLI-01 (verb-table specification), CLI-02, CLI-03, MCP-01, TEST-02, AGENT-02 and ARCH-08 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Detail delegated to the specification:**
  - which of today's two brownfield paths survives;
  - the exact verb names;
  - which convenience verbs exist.

  The specification records these choices; anything that changes this decision's direction returns to the owner.
- **Rebuild note (16 October 2026):** the rebuilt command surface keeps these principles; its first verbs are listed in the [Go architecture](../history/apply-design-before-dec-056/manager-apply-go-architecture.md#source-layout).

### DEC-018 The knowledge graph explains impact

- **Status:** Accepted, owner, 10 October 2026, approving the roadmap's recommendation for KG-00.
- **Statement:**
  - The knowledge graph's role is to explain impact: why something is affected, and which statements, decisions and owners connect a change to its consequences. It is not a separate query product.
  - A reduced core is ported to current main. The branch `codex/knowledge-graph` is not revived; it stays as a frozen reference.
  - The existing MCP surface is used. There is no separate knowledge server.
  - The YAML model stays canonical.
  - Whether Decisions become first-class model entries was proposed in KG-02 and is decided in [DEC-022](#dec-022-decisions-are-visible-and-route-through-their-subject).
- **Historical work items:** KG-01 to KG-03 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml). The earlier research item R02 is summarized on the [research page](../research/optional-research-r01-r03-20261009.md).
- **Rebuild note (16 October 2026):** written for the earlier Project model of Statements, Artifacts, Checks and Decisions. How it maps to language v1 and the registry is open ([OQ-009](#open-questions)).

### DEC-019 Government code starts after Wave 2

- **Status:** Accepted, owner, 10 October 2026, approving the roadmap's recommendation for GOV-00. For Government, this supersedes the evidence conditions of [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits).
- **Statement:**
  - Design work on Government may continue.
  - Government code starts once Wave 2 has settled the interface zone (CLI-02) and the runtime zone (ARCH-06).
  - Until then, uncommitted model changes on the Government branch are not integrated.
  - Model schema changes are coordinated through the roadmap's model zone, which has one active owner at a time.
  - Government documents on main follow the repository's conventions ([DEC-016](#dec-016-a-clean-stable-testable-main-and-uniform-structure-first)).
- **Historical work items:** GOV-01 and GOV-02 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).

### DEC-020 Markitect uses any library that helps

- **Status:** Accepted, owner, 10 October 2026 ([source](sources/roadmap-planning-20261010.md#external-libraries)). The question came from a verified BUG-01 finding: the architecture gate checks third-party imports only for Core, two Modules already import a YAML library, and the documents disagree ("standard library only" against "justified external dependencies").
- **Statement:**
  - Markitect uses any library that helps it reach its goal. No layer has a standard-library-only rule.
  - The architecture gate keeps its rules for imports between Markitect's own packages; it does not restrict third-party libraries.
- **Consequences:** the documents that state "standard library only" for Core or Modules are aligned, and the gate's third-party check for Core is removed.
- **Historical work items:** ARCH-03 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).

### DEC-021 Purpose and meaning-free edits route narrowly

- **Status:** Accepted, owner, 10 October 2026, choosing the recommended option of the KG-02 question ([source](sources/roadmap-planning-20261010.md#knowledge-graph-impact-routing)).
- **Statement:**
  - A change to the purpose of a Statement, Artifact or Check is routed like a description change: the definition, its realizations, consumers, owners and ancestors. It no longer widens impact to the whole project.
  - A meaning-free edit, such as a list reorder or an explicit default, routes only through its changed model file: that file's owner and the definitions in it.
  - Combining changes never shrinks impact; the monotonicity test covers the narrowed routing.
- **Consequences:** impact and plans for these edits become much smaller, and impact digests change once, so `--since` plans must be replanned. An obligation written only in a purpose routes narrowly, as one written in a description already does.
- **Historical work items:** KG-02 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Rebuild note (16 October 2026):** written for the earlier Project model of Statements, Artifacts, Checks and Decisions. How it maps to language v1 and the registry is open ([OQ-009](#open-questions)).

### DEC-022 Decisions are visible and route through their subject

- **Status:** Accepted, owner, 10 October 2026, choosing the recommended option of the KG-02 question ([source](sources/roadmap-planning-20261010.md#knowledge-graph-impact-routing)). This settles the open point of [DEC-018](#dec-018-the-knowledge-graph-explains-impact).
- **Statement:**
  - Decisions appear in the Report (omitted when a project has none) and in the owning Manager's Context.
  - A Decision change routes as a change of its subject Statement plus the Decision's owner, instead of a full-project review.
  - The actor is provenance, never authority.
- **Consequences:** ManagerContext digests change for Managers that own Decisions; plans for Decision edits select fewer Managers.
- **Historical work items:** KG-02 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Rebuild note (16 October 2026):** written for the earlier Project model of Statements, Artifacts, Checks and Decisions. How it maps to language v1 and the registry is open ([OQ-009](#open-questions)).

### DEC-023 Impact marks change and context without dropping either

- **Status:** Accepted, owner, 10 October 2026, choosing the recommended option of the KG-02 question ([source](sources/roadmap-planning-20261010.md#knowledge-graph-impact-routing)). The file-priority clarification below was delegated by the user on 11 October 2026 ([source and evidence](sources/dashboard-delegation-20261011.md#dec-023-delegated-clarification)); it is not a historical owner reply.
- **Statement:**
  - Each impact element is classed by a fixed rule. "Change" covers changed definitions and files, their realizations and Checks, direct consumers of changed Statements, and requires targets. "Context" is the rest of the closure.
  - For a changed file, explicitly mapped Statements and Statements realized by its explicitly mapped Artifacts, plus the direct consumers of those Statements, receive `change` priority. Their declared realizations and Checks follow the same rule; file contents do not imply undeclared relationships.
  - Impact sets and plans stay unchanged; the class is a priority signal, never permission to skip.
  - Dropping context-only Managers from plans or briefings waits for measured evidence from the playground (KG-03).
- **Consequences:** the impact result gains fields, so its digest changes once; invalidation and plans do not change.
- **Historical work items:** KG-02 and KG-03 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Rebuild note (16 October 2026):** written for the earlier Project model of Statements, Artifacts, Checks and Decisions. How it maps to language v1 and the registry is open ([OQ-009](#open-questions)).

### DEC-024 First reliable Apply rebuild and coordinated start

- **Status:** Endorsed direction, owner, 16 October 2026 ([original instruction](../design/canonical-model/sources/manager-realization-20261016.md#polish-the-concept-and-start-the-coordinated-rebuild)).
- **Statement:** Consolidate the Manager/canonical-v1 concept, retire the old backlog/dashboard, prepare Main and start bounded parallel development chats for the first reliable library Apply.
- **Consequences:** The [delivery plan](../implementation-plan.md) is the single current scope/dependency plan. Earlier package IDs/queues remain historical. The [technical contracts](../design/canonical-model/README.md) guide a clean rebuild without compatibility obligations. Implement the deterministic foundation and independent oracle in parallel; downstream runtime work waits for reviewed consumer contracts.
- **Boundary:** This commissions development and ordinary reviewed integration, not claims that Apply works. Exact provider identity, finite execution grants, real M1/M2 proof, release/deployment, studies and Government retain their stated gates. Detailed schema choices remain reviewable implementation contracts.

### DEC-025 Package Managers organize realization

- **Status:** Accepted, owner, 16 October 2026 ([strategy](../design/canonical-model/sources/manager-realization-20261016.md#management-and-a-customizable-strategy), [teams and registry](../design/canonical-model/sources/manager-realization-20261016.md#teams-integration-and-artifact-registration), [root Manager](../design/canonical-model/sources/manager-realization-20261016.md#root-manager-and-naming)).
- **Statement:**
  - One Manager per Package forms the hierarchy that realizes the model. Each level is an integration layer with less detail and more overview than the one below.
  - The root is an ordinary Manager („Er sollte jedoch auch einfach nur ein normaler Manager sein“); "project manager" only named its position.
  - The realization strategy is customizable („Generell sollte die Umsetzungsstrategie aber customizable sein“), and every Manager is configurable with its own team of AI Agents.
  - Every Manager registers its artifacts and justifies each with a Definition („Jeder Manager trägt dann jeweils seine Artefakte in eine Art Registry und begründet sie mit einer Definition“); every other file is a Markitect input or explicitly ignored. This refines [DEC-006](#dec-006-every-file-is-covered-or-explicitly-ignored).
- **Owning documents:** the [Apply concept](../design/canonical-model/manager-apply-concept.md). Its first strategy is Manager plus independent reviewer; the other team variants stay in the [north star](../design/canonical-model/manager-realization-north-star.md#team-variants).

### DEC-026 Apply comes first and changes only artifacts

- **Status:** Accepted, owner, 16 October 2026 ([change targets](../design/canonical-model/sources/manager-realization-20261016.md#common-preparation-and-change-targets), [separate operations](../design/canonical-model/sources/manager-realization-20261016.md#separate-operations-reconciliation-and-custom-commands), [Apply first](../design/canonical-model/sources/manager-realization-20261016.md#apply-first-and-delegation-from-the-root)).
- **Statement:**
  - Apply reconciles the artifacts with the model („Ausserdem soltle Apply generell die Artifacts gegen das Modell reconcilen“), whether or not the model changed.
  - Apply must work reliably first; other commands are extensions („zunächst darauf konzentrieren, dass apply zuverlässig funktioniert und die anderen commands als Erweiterungen behandeln“).
  - The owner prefers that no command changes both the model and the artifacts („Ich würde ungerne Commands machen, die sowohl Model als auch Artifacts anfassen“). Apply therefore changes only artifacts.
- **Owning documents:** the Apply concept; later operations are kept in the [north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions).

### DEC-027 Strict links with an optional Optimizer

- **Status:** Accepted, owner, 16 October 2026 ([source](../design/canonical-model/sources/manager-realization-20261016.md#strict-links-and-an-optional-optimizer)).
- **Statement:** Authored connections between Packages are the baseline for compiling context („Lass uns die strenge Variante als Grundlage nehmen“). An optional Optimizer proposes missing connections („einen optionalen Optimizer vorsehen, der fehlende Verknüpfungen vorschlägt“). Apply reports a missing connection but never adds it.

### DEC-028 Context follows need to know and starts simple

- **Status:** Accepted, owner, 16 October 2026 ([need to know](../design/canonical-model/sources/manager-realization-20261016.md#need-to-know), [simple context](../design/canonical-model/sources/manager-realization-20261016.md#preserve-the-variants-and-start-with-simple-context)).
- **Statement:** Need to know keeps Agents free of unnecessary context; it is not a protection against information leaks. Context compilation starts simple and is not optimized early („aufpassen nicht zu sehr zu optimieren“): the own Package's bodies, a See also section with linked Definitions, and files to read.

### DEC-029 Instructions enter through the root

**Updated by [DEC-039](#dec-039-deterministic-manager-process):** root entry remains; parent-authored child operation prompts are superseded by unchanged, Orchestrator-guaranteed propagation. The statement below records the earlier decision.

- **Status:** Accepted, owner, 16 October 2026 ([source](../design/canonical-model/sources/manager-realization-20261016.md#apply-first-and-delegation-from-the-root)). The file-access part was left to refine.
- **Statement:** The operation's instruction goes to the root Manager only. Each parent writes its own prompt for its children, which must not be too specific: it does not tell them which files to touch. Whether higher Managers read descendant files only when someone supplies them „gilt es zu refinen“; the Apply concept chose summaries first plus recorded targeted reads.

### DEC-030 Agent communication is traceable

- **Status:** Accepted, owner, 16 October 2026 ([source](../design/canonical-model/sources/manager-realization-20261016.md#reporting-and-communication-history)).
- **Statement:** It must be possible to trace which Agent received which prompt, what it reported, how the communication went and what flowed and changed in each step.
- **Owning documents:** the Apply concept's reports and the [journal](../history/apply-design-before-dec-056/manager-apply-journal.md).

### DEC-031 The first target is two Applies of the library example

- **Status:** Accepted, owner, 16 October 2026 ([target](../design/canonical-model/sources/manager-realization-20261016.md#two-apply-cycle-and-freedom-to-redesign), [architecture style](../design/canonical-model/sources/manager-realization-20261016.md#sketch-the-go-architecture-with-ports-and-adapters)).
- **Statement:**
  - Model the library completely as a canonical model, run Apply and examine the code; then change the model, run Apply again and examine the result.
  - The Agents must actually be started and communicate.
  - The Go source follows Ports and Adapters in idiomatic Go.
- **Owning documents:** the [acceptance gates](../design/canonical-model/manager-apply-implementation-plan.md) and the [delivery plan](../implementation-plan.md).

### DEC-032 Quality, simplicity and stability over effort and compatibility

- **Status:** Accepted, owner, 16 October 2026 ([rebuild](../design/canonical-model/sources/manager-realization-20261016.md#two-apply-cycle-and-freedom-to-redesign), [adapter and rebuild plan](../design/canonical-model/sources/manager-realization-20261016.md#choose-the-first-real-adapter-and-plan-a-clean-rebuild), [consolidation](sources/consolidation-discussion-20261016.md#quality-over-effort-and-compatibility)). It extends [DEC-014](#dec-014-compatibility-does-not-drive-decisions).
- **Statement:** Cleanliness, architectural quality, quality in general, simplicity and stability have high priority. Effort is not a criterion („Aufwand sollte keine Rolle spielen“), and neither is compatibility with old versions. The rebuild may proceed radically and delete what is no longer needed („Bei der Umsetzung können wir ruhig radikal vorgehen und einfach löschen“).
- **Consequences:** Choose the clean design even when it is more work. Add no adapters, aliases or legacy paths to save effort. Delete obsolete source, tests and documents together.

### DEC-033 Binding intent lives only in the model

- **Status:** Accepted, owner, 16 October 2026, adopting the assistant's recommendations A1 to A3 ([source](sources/consolidation-discussion-20261016.md#adopt-the-recommendations)).
- **Statement:**
  - If violating something makes an artifact wrong, it is model content. This includes realization constraints such as a public interface, the programming language and required build and test commands. The library's callable interface therefore moved into M1.
  - Configuration and custom instructions hold only execution choices and operational guidance. A project rule found there is a review finding and moves into the model. The one accepted repetition is an executable check command the model requires: configuration repeats it literally, names the Definition in `requiredBy`, and review keeps both in step.
  - A shared rule's scope is stated in the model. Configuration may only name the Kind and Property that carry scope; it never lists individual Definitions.
- **Owning document:** the [Apply concept](../design/canonical-model/manager-apply-concept.md#four-kinds-of-input-and-record). It applies the vision's rule of one canonical owner per fact to realization constraints.

### DEC-034 Removed intent removes its realization

- **Status:** Accepted, owner, 16 October 2026, adopting recommendation A4 ([source](sources/consolidation-discussion-20261016.md#adopt-the-recommendations)).
- **Statement:** When a Definition disappears, Apply deletes every artifact whose only basis it was, or the owner Manager records a new justified basis that review accepts. A moved or renamed Definition counts as a removal plus an addition: the link to the old identity is a blocking finding until the owner re-registers the artifact against the new identity with review. Leftover realization of removed intent fails the reconciliation.
- **Owning document:** the [Apply concept](../design/canonical-model/manager-apply-concept.md#removed-and-moved-intent). It serves [PRM-003](#promises).

### DEC-035 Committed model in, owner commit accepts the result

- **Status:** Accepted, owner, 16 October 2026, adopting recommendations B8 and A7 ([source](sources/consolidation-discussion-20261016.md#adopt-the-recommendations), [all recommendations](sources/consolidation-discussion-20261016.md#adopt-all-recommendations)).
- **Statement:** Apply reads the model, configuration, registry and ignore policy from the committed state and refuses to start with uncommitted changes to tracked files or untracked files outside Git-ignored paths. It writes its verified result only into the working tree and never commits, merges or pushes. The owner's commit of the published result is the acceptance, just as a committed model revision is the accepted specification. A project can also require the owner's approval before publication for selected Packages.
- **Owning documents:** the [Apply concept](../design/canonical-model/manager-apply-concept.md#apply-lifecycle) and the [port contracts](../history/apply-design-before-dec-056/manager-apply-port-contracts.md#guarded-publication-port).

### DEC-036 Adopted design refinements for the first Apply

- **Status:** Accepted, owner, 16 October 2026 ([recommendations](sources/consolidation-discussion-20261016.md#adopt-the-recommendations), [all recommendations](sources/consolidation-discussion-20261016.md#adopt-all-recommendations)).
- **Statement:** The owner adopted every item of the assistant's list of 16 October. Besides DEC-033 to DEC-035 these include:
  - the ownership of project-level artifacts, the skeleton and the wiring task;
  - the root's recorded delivery order (superseded by [DEC-039](#dec-039-deterministic-manager-process));
  - the registry layout and Markitect's own initial registry at the cutover;
  - the closed Orchestrator tool catalogue;
  - the Orchestrator's final evidence check instead of a separate verifier Agent;
  - continuation runs instead of a Superseded handoff.

  The source lists every item; the [design map](../design/canonical-model/README.md#apply-design) names the document that owns each subject.
- **Consequence:** Changing an adopted item needs the owner. Implementation can refine details within an item's intent, such as exact schemas or default numbers, through ordinary review.
- **Owning documents:** the [Apply design map](../design/canonical-model/README.md#apply-design) and the documents it lists.

### DEC-037 Agent sessions and checks run in Linux containers

- **Status:** Accepted, owner, 17 October 2026 ([source](sources/runtime-discussion-20261017.md)), after a real handshake with both CLIs ([record](../validation/runtime-handshake-20261017.md)).
- **Statement:**
  - Every Agent session and every project check runs in its own Linux container, started from a runtime image that the Orchestrator profile names by its image ID. The image pins the agent CLIs and the project's toolchain. Real Apply requires Docker or a compatible engine; controlled fake sessions in tests need none.
  - Authority comes from what a container receives. A writer gets its workspace writable, a reviewer the frozen candidate read-only, and a check a disposable copy. Independent expectations never enter a writer's container, and a writer's change to model, configuration or registry fails at Freeze. The container, hardened and running as a non-root user, is the boundary instead of the provider's sandbox, and stopping a session ends its whole process tree.
  - Workspaces live on Linux volumes owned by the runtime. Inputs and candidates move as Git objects through the Orchestrator's private state store, and only guarded publication writes to the owner's working tree.
  - The session helper `markitect-agentd` runs the provider CLI, serves it the Orchestrator tools over MCP, and carries provider protocol, tool calls and control over the container's single stdio connection.
  - Two session adapters implement the same port in the first version: Codex App Server and Claude Code. Orchestrator tools reach both through MCP, not through Codex's experimental dynamic tools.
  - Runtime credentials are API keys or long-lived tokens referenced by the Orchestrator profile, never copied desktop logins. They reach only the provider process and are redacted from the journal; because the Agent can still reach them, they should be scoped and spend-capped.
- **Consequences:**
  - Windows path identity, host process trees, provider sandbox setup and ambient user configuration leave the session path. The Orchestrator talks only to the container engine and Git.
  - The read-only metadata adapter that pins a Windows desktop binary is replaced by metadata the image records at build time.
  - The second session adapter and sandbox profiles move from the later extensions into the first version.
- **Owning document:** the [rebuild plan](../history/apply-design-before-dec-056/manager-apply-rebuild-plan.md#container-runtime). It serves [DEC-013](#dec-013-linux-first-for-tests-and-the-playground) and [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method).

### DEC-038 Learn through complete workflows

- **Status:** Accepted, owner, 17 October 2026 ([original adoption](sources/project-history-working-method-20261017.md#owner-adoption-and-publication-request)), after the [project-history reflection](../history/project-history-20261017.md#follow-up-reflection-and-owner-adoption).
- **Decision:** Adopt the proposed working method for future Markitect development: explain substantial redesign through new observations and a visible improvement criterion; give the accepted foundation coherent actual use against a complete workflow; investigate consequential uncertain boundaries early; review the need and simpler alternatives for mechanisms; and give short workflow-oriented owner updates at milestones. Keep current knowledge at existing owners.
- **Canonical owner:** [repository/learning-driven-workflow](../../.markitect/model/repository/learning-driven-workflow.yaml). The [working-method guide](../development/working-method.md) is its operational realization. The delivery plan retains scopes, sequence and integration; chats/PRs retain candidate evidence.
- **Boundary:** Adoption does not make the assistant's historical assessments measured facts, prove product benefit, waive review/CI/safeguards, create routine approval gates or authorize provider runs, studies or releases. This governs Markitect's own development; it does not impose a new product contract on adopting projects. It preserves DEC-031's two-Apply target and DEC-032's quality priorities.

### DEC-039 Deterministic Manager process

**Updated by [DEC-041](#dec-041-core-protocol-choices):** cross-area requests are forwarded by the Orchestrator instead of dispatched hop by hop. Warm retention and quiescent basis advances are deferred beyond the first Apply. The statement below records the decision as accepted.

**Updated by [DEC-052](#dec-052-implement-then-review-no-explore-phase):** the Explore refinement, the write barrier and the "before the Explore barrier" mount rule below are removed; every Manager starts with a work step. Read-only mounts and mutation denial still apply to steps that may not write.

- **Status:** Accepted, owner, 17 October 2026 ([original clarification and confirmation](sources/manager-process-clarification-20261017.md)). This changes the parent-authored prompts of DEC-029 and the root-planned delivery order adopted in DEC-036. Historical sources remain unchanged.
- **Statement:** Apply supplies one static operation instruction. Deterministic compilation supplies each Manager's explicit model context; configuration and Orchestrator facts complete its template. The Orchestrator guarantees propagation from root through every direct child, recursively, without AI selection, rewriting or omission. Every local Manager assesses its area and returns reviewed change or no-change; intermediate and empty Packages are not skipped. Managers act on messages and communicate only with direct parent or children. Cross-area requests travel up and down that tree, using explicit routing facts or bounded inquiry, never inferred model relationships. Local review, integration, corrections and final evidence/publication follow the fixed Orchestrator process. Parents retain their own work and integration responsibility, including prerequisites needed by waiting children.
- **Why:** The owner observed that the previous design moved already modeled structure back into AI planning. The intended organization should guarantee participation and required workflow steps while leaving realization and semantic judgment to Agents.
- **Canonical owners:** [organization/delegated-context](../../.markitect/model/product/organization/delegated-context.yaml) and [run/confirmed-run-history](../../.markitect/model/product/run/confirmed-run-history.yaml). The [Apply concept](../design/canonical-model/manager-apply-concept.md#tasks-and-delegation) owns the process and its remaining design questions.
- **Confirmed refinement:** The owner selected an initial read-only Explore assessment by every Manager before writing, followed by local plans without a global Plan barrier. The Orchestrator admits eligible work in recorded FIFO order with stable ties. Limited normal deliveries require review, Orchestrator acceptance and a concrete consumer basis while the producer Manager remains open. Parent hints stay separate from the operation and model; parents use broad, traceable child summaries. Dependency waits and resource exclusion remain distinct, with one writer in the first strategy ([confirmation](sources/manager-process-clarification-20261017.md#explore-planning-and-limited-deliveries)). Review reuse requires the unchanged complete bound subject.
- **Confirmed budget refinement:** Limit rejected submissions per run, Manager and review purpose, rather than all submissions. Success neither uses nor resets rejection allowance; all work consumes the finite shared execution budget. At the default limit, the parent assesses the plan after two rejections and the third leaves the affected work incomplete. New tasks or limited deliveries grant no fresh counters ([confirmation](sources/manager-process-clarification-20261017.md#review-budget-confirmation), [execution contract](../history/apply-design-before-dec-056/manager-apply-execution.md#admission-amendments-and-finite-correction)). **Updated by [DEC-050](#dec-050-an-accepted-delivery-restarts-the-rejection-count):** an accepted delivery restarts the count for its purpose.
- **Confirmed continuity refinement:** Keep Manager responsibility and conversation across explicitly versioned step bases where supported, while retaining immutable run inputs and review candidates. Preserve own unaccepted work during a recorded quiescent basis advance; stale actions and unknown effects block progress. Changed execution rights need a newly bound environment with proven resume or an explicit handoff. Separate role, effect authority and budget class. Before the Explore barrier, enforce read-only artifact mounts and Orchestrator mutation denial. Name the target setting `rejectionLimit` and its ceiling `maxRejectionLimit`; preserve historical records rather than reinterpret them. Basic metrics remain in Apply; a later hot-path view is an extension ([confirmation](sources/manager-process-clarification-20261017.md#working-bases-and-session-continuity), [concept](../design/canonical-model/manager-apply-concept.md#working-bases-and-conversation-continuity)).
- **Confirmed coordination refinement:** Versioned bounded child reports preserve delivery/completion status, interfaces, prerequisites, findings, uncertainty and evidence; enforce the parent input limit as well as each report limit, using visible partitioning without dropping obligations. Logical Managers may retain conversations without resident containers; finite warm retention and deterministic stop rules require actual quiescence, charged replacement starts and proven resume or explicit handoff. Process census alone is not quiescence: close admission, reconcile effects and enforce an artifact-write barrier through basis changes. Reuse of accepted Manager assessments and coordination-based modeling hints are later extensions, not changes to initial Explore or current review capacity ([confirmation](sources/manager-process-clarification-20261017.md#bounded-coordination-and-idle-execution), [concept](../design/canonical-model/manager-apply-concept.md#tasks-and-delegation)).
- **Evidence boundary:** This records intent, not a working recursive runtime. The concept includes a fixed-source comparison and distinguishing process cases. Technical realization follows the process; no new Core language feature, provider execution or release is authorized here.

### DEC-040 Concept freeze until the core runs

**Bounded exception:** [DEC-043](#dec-043-project-teams-review-arbitration-and-product-defaults) records the owner's later explicit instruction to integrate team configuration and product defaults into the concept on Main. It changes only that subject; the original freeze remains the boundary for unrelated work.

- **Status:** Accepted, owner, 17 October 2026 ([source](sources/architecture-review-20261017.md#concept-freeze-and-order-of-work)), after the [architecture review](../design/canonical-model/apply-architecture-review-20261017.md).
- **Statement:**
  - The concept is frozen („Das Konzept ist jetzt eingefroren“).
  - It is to be completed first („erst das Konzept fertigstellen“), and the cleanup round should start in any case („Die Aufräumrunde wäre generell gut zu starten unabhängig davon“).
  - Implementation follows once the concept is clean, uniform and worked out („sauber, einheitlich und ausgearbeitet“).
- **Interpretation (marked):** the freeze applies to the documents of the [Apply design map](../design/canonical-model/README.md#apply-design). It answers the review's recommendation to add no new mechanisms until one Package runs end to end. Completing the concept means specifying the core, resolving contradictions and giving each rule one owner, without adding scope.
- **Boundary:** DEC-025 to DEC-039 remain in force. A review proposal that changes an adopted item still needs the owner, as [DEC-036](#dec-036-adopted-design-refinements-for-the-first-apply) requires.
- **Owning document:** the [Apply design map](../design/canonical-model/README.md#apply-design).

### DEC-041 Core protocol choices

**Updated by [DEC-052](#dec-052-implement-then-review-no-explore-phase):** a Manager that finds nothing to change submits its no-change result from its first work step; there is no Explore step.

- **Status:** Accepted, owner, 17 October 2026 ([source](sources/architecture-review-20261017.md#core-protocol-choices)). The owner selected one prepared option for each of six questions.
- **Statement:**
  - A question to a known addressee is forwarded by the Orchestrator along the tree, without dispatching the Managers in between („Host leitet weiter“).
  - Each step starts from the currently accepted aggregate plus the Manager's own open changes („Aktueller Gesamtstand“).
  - A Manager that finds nothing to change during Explore submits a no-change result directly („Ja, direkt einreichen“).
  - When a task cannot finish, the Orchestrator admits no new work and finishes the run Incomplete, retaining the work („Sofort stoppen“).
  - The first Apply stops every container after its step, and warm retention follows later („Erst stoppen, warm später“).
  - While a run waits for the owner, the Orchestrator process may exit; `answer` and `resume` continue the run („Ja, Prozess darf enden“).
- **Changes:**
  - DEC-039's hop-by-hop dispatch of cross-area requests becomes forwarding. Requests still travel along the tree, and every hop is recorded.
  - The concept's explicit consumption of each delivery into a working basis becomes the current aggregate.
  - DEC-039's finite warm-retention policy and quiescent basis advance are deferred. The first Apply keeps no container warm.
  - An incomplete task now finishes the whole run, not only the affected work.
- **Owning document:** the [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md), rules M2, D3, T3, T14, S6 and W2.

### DEC-042 Orchestrator names the deterministic coordination component

**Current disposition, 20 October 2026:** DEC-064 retains Orchestrator and removes the earlier allowance for competing current code/config/CLI names; literal hosts and history retain their meaning.

- **Status:** Accepted, owner, 17 October 2026 ([source](sources/orchestrator-terminology-20261017.md#owner-instruction)).
- **Statement:** Use **Orchestrator** for Markitect's deterministic coordination component previously called Host. The Manager responsibilities and the accepted process are unchanged.
- **Boundary:** Update current documentation and its generated sources. Keep historical sources and literal quotations unchanged. Existing API identifiers, wire/configuration fields and CLI options keep their implemented names; this decision does not dispatch an implementation rename or alter the ongoing Apply work. A host machine remains a host machine.
- **Owning document:** the [Apply glossary](../design/canonical-model/README.md#apply-design).

### DEC-043 Project teams review arbitration and product defaults

- **Status:** Accepted, owner, 17 October 2026, in the Config chat ([requirements and explicit Main publication instruction](sources/configuration-teams-defaults-20261017.md)). Implementation pending; the illustrative target syntax is not a supported v1 payload.
- **Decision:** Project defaults select the Manager's model/effort/instructions, multiple stable named reviewers and a conflict-only reviewArbiter. Exact Package overrides replace supplied fields and append captured instruction lists. New reviewers need complete model/effort; at least one independent reviewer remains active. No ancestor-override inheritance or extra authored Manager hierarchy.
- **Resolution:** All required independent assessments must be complete. Model-grounded blocking findings prevent acceptance; preferences remain advisory. A configured fresh read-only arbiter settles explicit disputes with current grounds, without changing intent, implementing or requiring personal unanimity. An unchanged closed dispute needs new evidence to reopen. The owner resolves unclear intent; unfinishable work remains Incomplete. One submitted candidate counts at most once as rejected per review purpose, regardless of reviewer count; existing budget and exhaustion rules remain.
- **Defaults:** Product-owned versioned finite defaults supply omitted limits, assignment caps and reserve fields. The configuration owner records the accepted starting values, including a 10m step default and 20m ceiling. Preserve original bytes, effective-value provenance and immutable run bindings; validate combinations rather than silently clamp. Values are not measured sufficiency or monetary ceilings.
- **Interim until implemented ([DEC-048](#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose)):** without an arbiter, the prompts let the next independent review settle a reading conflict by the model's text. A recorded case or finding that the text clearly contradicts is marked not applicable, with the passage. **Updated by [DEC-051](#dec-051-agents-get-tasks-the-orchestrator-owns-the-process):** a Manager that keeps a behavior against a case cites the passage in its submission, and no longer asks and waits itself. The candidate review asks the owner, with needs_decision, when the text leaves open a reading that changes observable behavior or when it keeps a reading that the submission disputes with a cited passage. An owner decision reaches only the task that asked, so a parent or sibling review can raise the same reading again. **Updated by [DEC-053](#dec-053-owner-decisions-bind-the-whole-run):** an owner decision now reaches every task of the run.
- **Change boundary:** Supersedes the single-reviewer-only target and the named-team deferral within ENH-006; extends DEC-040 only for this owner-authorized concept delivery. Existing decoded formats, fixtures, source policies and runtime behavior keep their actual version until coordinated implementation. Other team variants remain deferred. Model authority, guaranteed participation, protected inputs, ownership, independent review and guarded publication remain mandatory.
- **Owners:** [Configuration](../history/apply-design-before-dec-056/manager-apply-configuration.md), [review/execution](../history/apply-design-before-dec-056/manager-apply-execution.md#review-conflicts-and-arbitration), [core](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), [ports](../history/apply-design-before-dec-056/manager-apply-port-contracts.md#configured-review-team-bindings) and [journal](../history/apply-design-before-dec-056/manager-apply-journal.md#review-team-and-arbitration-records). No runtime, comparison or study authority is granted.

### DEC-044 Commands reuse the Manager organization

- **Status:** Accepted direction, owner, 17 October 2026 ([command examples](sources/commands-extract-optimize-20261017.md#owner-command-examples), [Verify clarification](sources/commands-extract-optimize-20261017.md#owner-clarification-of-verify), [Extract clarification](sources/commands-extract-optimize-20261017.md#extract-describes-the-existing-stand), [Main publication and Optimize request](sources/commands-extract-optimize-20261017.md#publication-request-and-optimize)).
- **Confirmed model-migration refinement:** The owner accepted meaning-preserving Package moves and merges, explicit v1 identity/reference migration, future Manager-tree derivation without changing the active organization, and jointly reviewed registry/configuration adjustments while retaining realization artifacts ([source and Main instruction](sources/model-migrations-20261017.md#owner-acceptance-and-main-publication), [owning workflow](../design/canonical-model/commands.md#model-restructuring-as-an-explicit-migration)). Existing identity rules, scope, authority and historical evidence remain; publication/staging support is not implemented by this documentation. This resolves the earlier structural-Optimize scope question and adds no implementation assignment.
- **Confirmed Optimize search refinement:** The owner accepted model-wide inventory/search and question-driven targeted reading within the selected local model, explicitly bound dependencies and permitted captured artifacts; loaded context and write authority remain separate. Every Manager's own assessment, parent integration and independent review complement lexical search; bounded results retain continuation and incomplete investigation stays visible. No fixed link-depth cap or semantic-index subsystem is selected ([source and Main instruction](sources/commands-extract-optimize-20261017.md#optimize-search-space-and-targeted-reading), [owning workflow](../design/canonical-model/commands.md#find-candidates-beyond-existing-links)). This resolves the earlier search-boundary question without adding implementation or runtime authority.
- **Statement:** Apply, Cleanup, Bughunt, Refactor, Verify and Custom Commands use operation instructions across the Manager tree, focused through each area's context. Verify only assesses conformity and reports findings; it changes neither model nor artifacts. Extract initially describes the existing repository and presents code/documented-intent contradictions as questions. Its model authorship, appropriate detail, independent review, rework and parent integration reuse existing capabilities; read-only Verify findings guide model refinement. Optimize examines the canonical model for missing links and justified improvements without unsupported invention.
- **Interpretation (marked):** The owner's request to record the resulting Extract proposal adopts reuse of normal model-candidate review rather than a separate permanent completeness catalogue. Verify conformity alone does not establish model depth. Model-quality review remains Extract's responsibility. The owning design distinguishes selected behavior from future settings, staging and execution contracts.
- **Confirmed Optimize refinement:** The owner accepted inventory/search and targeted reading beyond existing links; source-supported, meaning-preserving change proposals; independent review for invented relationships, changed obligations and context effects; normal rework and parent integration; and reviewed improvement, grounded question or justified no-op outcomes ([source](sources/commands-extract-optimize-20261017.md#optimize-workflow-clarification)). The [remaining command questions](../design/canonical-model/commands.md#remaining-command-decisions) are explicitly unselected recommendations, not new assignments.

- **Extract completion clarification:** The owner subsequently accepted the source-grounded adoption workflow and explicitly allowed a reviewed model proposal to be adopted with known realization deviations ([verbatim continuation](sources/commands-extract-optimize-20261017.md#model-adoption-with-known-realization-deviations), [owning criterion](../design/canonical-model/commands.md#completion-and-adoption-with-known-deviations)). This replaces the earlier all-conforming Verify completion requirement. A sound model and adequately supported retained deviations can be ready for adoption; unresolved material intent, inadequate model coverage or evidence gaps cannot. Verify remains read-only and retains its actual result. Established intent is not changed to excuse faulty artifacts, and their repair is separate authorized work.
- **Boundary:** Extends DEC-026 and DEC-027's later-operation direction. This is authority to record and integrate documentation on Main, not to expand the frozen Apply runtime, start implementation packages or execute providers, Docker, Apply, comparisons or studies. The current delivery order, model/artifact separation, canonical authority and mandatory review remain in force.
- **Owner:** [Commands over the Manager organization](../design/canonical-model/commands.md). The [current operations guide](../project-operations.md) continues to own executable availability.

### DEC-045 Managers and reviewers derive technical risks

- **Status:** Accepted clarification, owner, 17 October 2026 ([question and implementation instruction](sources/technical-risk-derivation-20261017.md)).
- **Decision:** Managers and independent reviewers actively derive relevant technical risks and necessary evidence from applicable model obligations, operating conditions and actual realization, including interactions between areas. The owner need not enumerate every failure mode. Managers investigate and repair within their mandate; reviewers independently derive model-based expectations and, after candidate access, examine implementation-specific risks and obtain additional necessary evidence through existing findings and rework.
- **Authority boundary:** A necessary technical consequence of existing intent is realization judgment, not a new project rule requiring its own model edit. Changed business policy, exceptional tradeoffs outside the mandate and missing necessary model relationships retain the existing decision/model-change route. No invented operating conditions, new rights or inferred canonical links.
- **Scope:** Clarifies existing responsibilities and the previously deferred task-premise duty. No new role, register, workflow stage, schema or experiment is introduced. Existing budgets, independent review, arbitration and evidence requirements remain. This authorizes the bounded concept and role-template change, not provider execution or a claim that risk discovery has been demonstrated.
- **Owners:** [Apply concept](../design/canonical-model/manager-apply-concept.md#technical-risk-derivation), [execution evidence](../history/apply-design-before-dec-056/manager-apply-execution.md#evidence-need-survives-budget-pressure), and the canonical [delegated-context Rule](../../.markitect/model/product/organization/delegated-context.yaml).

### DEC-046 A withheld pass keeps its recorded expectations

- **Status:** Accepted, Architect under the owner's authorization, 18 October 2026 ([evidence and owner instruction](sources/withheld-review-expectations-20261018.md)).
- **Decision:** When the Orchestrator withholds a pass (I3, D4, T9), the review's recorded expectations stand for the task's next submission. That submission's review skips the expectations step while the submission's purpose and scope, the task's open findings other than those the withheld passes raised, and the owner's decisions for the task are unchanged. Its candidate step still starts a fresh reviewer conversation and assesses every recorded case.
- **Reason:** The expectations step sees neither the candidate nor the aggregate. Re-recording them for an unchanged resubmission only redraws the cases, and in the library runs it rejected candidates that had just passed, consuming the shared rejection limit.
- **Updated by [DEC-047](#dec-047-recorded-review-cases-stand-until-their-inputs-change):** a rejection also keeps the recorded expectations, and the Manager sees them. **Updated by [DEC-048](#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose):** a pass keeps them as well, and a change of scope no longer records fresh ones.
- **Scope:** Amends T7 for this case only. A rejection, a failed assessment, an owner answer and any changed input still record fresh expectations. Rejection limits, independence and evidence requirements are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), rule T7.

### DEC-047 Recorded review cases stand until their inputs change

- **Status:** Accepted, Architect under the owner's authorization, 18 October 2026 ([evidence and owner instruction](sources/stable-review-cases-20261018.md)).
- **Decision:** A rejected candidate also leaves its review's recorded expectations standing for the task's next submission, as [DEC-046](#dec-046-a-withheld-pass-keeps-its-recorded-expectations) established for a withheld pass. The next review skips the expectations step while the submission's purpose and scope, the task's open findings other than those the reviews against these expectations raised or resolved, and the owner's decisions for the task are unchanged. The Manager's steps show these recorded cases, so that its rework can give each case evidence that shows it ran.
- **Reason:** Review must converge across rounds. In the library runs each resubmission redrew the case set and re-judged every case; a case evidenced twice was judged missing the third time, and two runs ended Incomplete at the rejection limit without an asserted defect.
- **Updated by [DEC-048](#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose):** the recorded cases stand for every later submission with the same purpose, after a pass too and whatever the scope.
- **Scope:** Amends T7 and DEC-046. Every candidate review still assesses every recorded case independently, resolves open findings with current evidence and may raise findings for other gaps. These review cases are not the frozen evaluation expectations of [DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers), which still never enter a writer's container. A failed assessment, an owner answer and any changed input still record fresh expectations. The rejection limit and evidence requirements are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), rule T7.

### DEC-048 Recorded review cases stand for every delivery of a purpose

- **Status:** Accepted, Architect under the owner's authorization, 18 October 2026 ([evidence and owner instruction](sources/review-cases-per-purpose-20261018.md)).
- **Decision:** A task's recorded review expectations stand for each of its later submissions with the same purpose, whatever the last verdict other than a failed assessment and whatever the submitted scope. Their review skips the expectations step while the task's open findings other than those the reviews against these expectations raised or resolved, and the owner's decisions for the task, are unchanged. The expectations step does not see the scope. It records cases for the task's complete responsibility under that purpose, and each candidate review decides which cases apply to the delivered scope. The Manager's steps show the standing cases for each purpose.
- **Reason:** The expectations step sees neither the candidate, the aggregate nor the content of a limited delivery, so a new delivery gave it the same inputs as the last one, and a fresh set only redrew the cases. In R2-A1-08, `library` converged on each of its three limited deliveries, yet each new delivery redrew the cases. The third set judged the model's no-line-limit property missing under a new wording, after two reviews had judged the same tests as evidence for it. The run ended Incomplete at the rejection limit.
- **Cost:** Fewer independent case sets are drawn, so coverage beyond the first set rests on each candidate review's duty to look past the recorded cases and raise findings for other gaps. The pass rates in the source (7 of 8 against kept cases, 5 of 11 on a fresh set's first review) favor kept cases but are confounded, because kept-case reviews follow rework.
- **Scope:** Amends T7, DEC-046 and DEC-047. Purposes keep separate cases, and a changed finding or owner decision still records fresh ones, as does a failed assessment or a failed review step. Every candidate review still assesses every recorded case independently and may raise findings for other gaps. A recorded case is a reviewer's reading of the model, not new intent. A conflict between that reading and the model's text follows the interim rule recorded at [DEC-043](#dec-043-project-teams-review-arbitration-and-product-defaults). The rejection limit and its accounting per run, Manager and purpose, the evidence requirements and the frozen evaluation expectations of [DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers) are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), rule T7.

### DEC-049 Review cases are recorded before the work they assess

**Updated by [DEC-052](#dec-052-implement-then-review-no-explore-phase):** there is no write barrier any more, so "once the write barrier is open" below no longer limits anything. A ready task's expectations step is admissible when the run starts.

- **Status:** Accepted, Architect under the owner's authorization, 18 October 2026 ([evidence and owner instruction](sources/review-cases-before-work-20261018.md)).
- **Decision:** Once the write barrier is open, a task whose next step writes for a purpose gets an expectations step first, unless cases for that purpose still stand ([DEC-048](#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose)). A work step writes for the local purpose; an integration step writes for the integration purpose. The recorded cases stand for the task's submissions with that purpose, and the Manager's writing step shows them. A review of a submission against standing cases has only its candidate step. A reply step is admitted before other items that became admissible with it, so that a question is answered before the asking step's submission is reviewed.
- **Reason:** In R2-A1-08, 6 of 11 fresh case sets were rejected on their first review, for scenarios the Manager could not know. Recording the cases first lets the Manager build against them. The expectations step never sees a candidate, so recording it earlier changes nothing it could see.
- **Cost:** Expectations steps move but are not removed; a task whose inputs change before it submits records its cases once more, for example after an owner answer. The gain is the expected drop in first-review rejections, which a run still has to show. An expectations step before writing is charged as preparation and does not use the completion reserve.
- **Scope:** Amends T4 and T7. The candidate review still assesses every case independently and may raise findings for other gaps. Cases are recorded again when the task's open findings or owner decisions change, and when a review step fails. The rejection limit and the evidence requirements are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), rules T4 and T7.

### DEC-050 An accepted delivery restarts the rejection count

- **Status:** Accepted, owner, 18 October 2026 ([proposal and decision](sources/rejection-count-restart-20261018.md)).
- **Decision:** The rejection limit counts the rejections of a task's candidates for one purpose since the task's last accepted delivery for that purpose. An accepted delivery, limited or complete, restarts the count. The limit, its default of three and its configuration are unchanged.
- **Reason:** The limit stops work that does not converge. An accepted delivery shows that the work converges. In R2-A1-08, `library` repaired each rejection and had its next candidate accepted, yet the third rejection over three deliveries ended the run Incomplete.
- **Scope:** Amends the budget refinement of [DEC-039](#dec-039-deterministic-manager-process) and the execution contract. All work still consumes the finite shared execution budget, which bounds the total cost. A withheld pass is not an accepted delivery.
- **Updated by [DEC-054](#dec-054-a-rejection-that-finds-fewer-gaps-does-not-count):** a rejection that finds fewer gaps than the last one does not count.
- **Owners:** [execution contract](../history/apply-design-before-dec-056/manager-apply-execution.md#admission-amendments-and-finite-correction).

### DEC-051 Agents get tasks; the Orchestrator owns the process

- **Status:** Accepted, owner, 18 October 2026 ([owner instruction](sources/agents-get-tasks-20261018.md)).
- **Decision:** A step's input gives its agent a clear task: its role, the binding Definitions, its files, its children's results, its open findings and review cases, its messages and the answer it must return. Admission, barriers, waits, limits, acceptance and routing to the owner belong to the Orchestrator and are not explained to agents. A process gap is closed in the Orchestrator, through state, structured answers and refusal reasons, not with more prompt text.
- **Reason:** The step inputs had accumulated the Host's rules: admission order, the write barrier, the rejection allowance, owner-question mechanics. The owner expects results to become less reliable as prompts take over organization and orchestration, and wants that loop stopped.
- **Scope:** Applies to every rendered step input and Host note. The model's binding intent, the technical-judgment duty of [DEC-045](#dec-045-managers-and-reviewers-derive-technical-risks) and the provenance the Host records for each section are unchanged. The first change removes the process rules from the role, mandate, phase, review and note texts. The provenance sections the step input still shows, such as record references, limits, profile identities and inclusion reasons, move out of the visible text in a follow-up; the Host keeps recording them. Task facts such as a Manager's creation areas stay visible.
- **Owners:** [organization/delegated-context](../../.markitect/model/product/organization/delegated-context.yaml) rendering in `internal/organization`, and the Host notes in `internal/run`.

### DEC-052 Implement then review: no Explore phase

- **Status:** Accepted, owner, 18 October 2026 ([owner instruction and evidence](sources/implement-review-20261018.md)).
- **Decision:** A Manager's first step is a work step. Every task starts ready. Apply has no Explore step, no write barrier and no intake review. A Manager that needs no change submits a complete no-change result from its first work step. A reviewer records the cases before that step ([DEC-049](#dec-049-review-cases-are-recorded-before-the-work-they-assess)), and the independent review assesses the result against them. No Plan step is added. The owner registers or ignores the unexplained files of the captured source before Apply, and the file check fails a run on an unexplained file.
- **Reason:** In the bound runs R2-A1-04 to R2-A1-08, Explore took 6 steps per run, one for each of the 6 Managers, about 4 % of wall time, and gave no information in an empty repository. What it cost was the write barrier and the prompt rules around it, which [DEC-051](#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) wants out of the prompts. Removing the intake of unexplained files is a design simplification: intake was never implemented, so it cost nothing in those runs.
- **Cost:** The evidence comes only from an empty repository. A parent no longer reads its children's assessments before it starts, and the root learns about files that several Packages need only from messages and deliveries. Later runs have to show whether questions and waits grow. Settling unexplained files is now the owner's job before Apply, which is an adoption burden for existing code.
- **Scope:** Replaces the Explore refinement of [DEC-039](#dec-039-deterministic-manager-process) and the no-change-during-Explore rule of [DEC-041](#dec-041-core-protocol-choices). Amends [DEC-049](#dec-049-review-cases-are-recorded-before-the-work-they-assess). Removes rules T1 to T3 (their IDs stay reserved), the `report` tool and the intake path of D6. Admission order, the review and acceptance rules, the rejection limit, the budget purposes and the messages are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), rules T4 to T7, D6 and the Orchestrator tool table.

### DEC-053 Owner decisions bind the whole run

- **Status:** Accepted, owner, 18 October 2026 ([evidence and owner instruction](sources/run-wide-owner-decisions-20261018.md)).
- **Decision:** An owner's answer binds the whole run: every task's later Manager steps and candidate reviews see it, not only the asking task's; another task's expectations steps do not, so its recorded cases stand. The owner's earlier decisions can be bound with a run as `ownerDecisions` in its configuration; every step sees them from the run's start, so a question that the owner has decided is not asked again. A candidate review judges a recorded case or finding that an owner decision contradicts as not applicable. Only the asking task's recorded review cases are renewed by an answer; bound decisions are inputs of every task's cases from the start.
- **Reason:** In R2-A1-09 the run waited 54 minutes for the owner, on four questions about two topics. The root, `lending` and `library` asked about the same date boundary, because an answer reached only the task that asked.
- **Scope:** Amends I2. Asking the owner, the Waiting state and the owner's authority are unchanged. A bound decision is the owner's own text; the Host neither infers nor matches decisions against questions. Each run binds its own decisions, so a continuation can carry decisions made after the earlier run; changing them ends an open run as stale.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#issues), rule I2, and the [run configuration](../project-workflow.md).

### DEC-054 A rejection that finds fewer gaps does not count

- **Status:** Accepted, owner, 19 October 2026 ([evidence and owner decision](sources/rejection-count-progress-20261019.md)).
- **Decision:** A rejection's gaps are its recorded cases judged missing plus the blocking findings it raises. A rejection counts toward the rejection limit unless its gaps are fewer than those of the task's last rejection for the same purpose. The first rejection always counts. An accepted delivery restarts both the count and the comparison.
- **Reason:** The limit stops work that does not converge, and fewer gaps show convergence. Two Luna runs, R2-A1-10 and R2-A1-11, ended Incomplete at the limit while the rejected Manager's gaps fell; in R2-A1-11 the third rejection found every recorded case evidenced and one blocking finding.
- **Cost:** Blocking findings that an earlier review raised and that stay open are not counted again, so a review that raises fewer new findings counts as progress. The finite shared execution budget bounds such a sequence.
- **Scope:** Amends [DEC-050](#dec-050-an-accepted-delivery-restarts-the-rejection-count) and the execution contract. The limit, its default of three, its configuration, the review rules and the evidence requirements are unchanged. All work still consumes the finite shared execution budget, which bounds the total cost.
- **Owners:** [execution contract](../history/apply-design-before-dec-056/manager-apply-execution.md#admission-amendments-and-finite-correction).

### DEC-055 A submission must pass its required checks

- **Status:** Accepted, Architect under the owner's authorization, 19 October 2026 ([evidence and authorization](sources/submit-required-checks-20261019.md)).
- **Decision:** When a work or integration step submits, the Host runs the run's required checks on the observed workspace before it answers the call. If one fails, the submit is refused with the check's output, and the Manager fixes the build within the step or waits for what it needs. The check is journaled and charged as writing work; it is skipped when it does not fit the step's remaining time or would need the completion reserve, and when it did not run to an observed result. A candidate review still runs its own checks, and only those count as review evidence.
- **Reason:** A review passes only when the required checks pass (T8), so a candidate that fails them cannot be accepted. In R2-A1-12, `members` reached the rejection limit with three such candidates: twice the shared Go module did not exist yet, and once the project did not compile.
- **Cost:** Each submit runs the required checks once more; in R2-A1-12 a check took about 10 seconds. A Manager that cannot make the checks pass alone has to wait for the file it needs or ask its owner, instead of submitting.
- **Scope:** Amends S4 in the way [PR #457](https://github.com/Glacius-Labs/Markitect/pull/457) amended S4 for uncapturable changes. The review rules, the evidence requirements, the rejection limit and the checks' configuration are unchanged.
- **Owners:** [core protocol](../history/apply-design-before-dec-056/manager-apply-core.md#steps), rule S4, and the [execution contract](../history/apply-design-before-dec-056/manager-apply-execution.md).

### DEC-056 Recursive reconciliation with a shared workspace

**Current disposition, 20 October 2026:** DEC-058 removed initial Verify; DEC-064 now specifies the accepted common Pass mechanism and Execute decomposition. Read the current workflow before implementing this dated entry.

- **Status:** Accepted direction, owner, 19 October 2026 ([verbatim owner source](sources/recursive-apply-simplification-20261019.md)). Implemented under the owner's [implementation dispatch](sources/recursive-apply-implementation-20261019.md) and merged (PR #463); pending owner acceptance. The implemented technical contract is [manager-apply-runtime.md](../design/canonical-model/manager-apply-runtime.md); real library runs are recorded in the [demonstration](../validation/recursive-apply-demonstration-20261019.md) and the [repetitions](../validation/recursive-apply-repetitions-20261019.md).
- **Decision:** Keep the permanent recursive Manager organization. Define ownership before Apply through persistent artifact areas with mirrored model structure as the default, explicit persistent exceptions and deepest-owner responsibility. Resolve those areas even before artifact directories exist. Managers create/register their own files in per-area registry shards. Initially trust write boundaries and observe violations; technical ownership enforcement is future work. All communication is strictly parent/child, and only Root asks the human.
- **Start-from-model clarification (owner, 19 October 2026):** The next library demonstration starts without application artifacts or prepared internal APIs. Necessary execution controls and explicit model contracts remain inputs. Root supplies root-owned shared prerequisites; Managers create their own interfaces/artifacts and coordinate dependencies through the tree. No external cross-area skeleton or stub/API preparation may hide this work. See the [owner confirmation](sources/recursive-apply-simplification-20261019.md#default-layout-and-model-only-start) and [library walkthrough](../design/canonical-model/manager-apply-library-flow.md#first-apply-m1-to-a1).
- **Cycle:** Fresh Verify against an immutable starting stand; parallel Execute on a shared working tree where needed; quiesce writers and capture one immutable closing stand; fresh recursive Verify and required checks; repair through owners and repeat, bounded by the overall budget. No global initial audit barrier; no artificial Execute when already valid. Review checks actual artifacts and child interactions. Root completion requires every responsibility, current closure of blocking findings and the exact reviewed/published stand.
- **Later amendment:** [DEC-058](#dec-058-apply-always-executes-before-independent-verify) replaces that initial-Verify entry and Verify-only shortcut with Execute and integration first for every Apply, followed by independent Verify. The dated cycle above records the earlier decision and implemented sequence, not the current accepted target.
- **Simplification:** Replace per-turn private writer composition, path reservations, delivery purpose/scope types, expectations-case administration, separate no-change transactions, mandatory named per-delivery reviewer/arbiter machinery and rejection-counter exceptions. Preserve model authority, complete accounting, independent perspectives, concrete findings, mandatory checks and guarded publication/read-back. The [supersession map](../design/canonical-model/manager-apply-concept.md#implementation-handoff-and-supersession) specifies the boundaries; it does not replace all earlier decisions wholesale.
- **First-demonstration scope confirmation (owner, 19 October 2026):** No additional Fork, team or Skill functionality for the first end-to-end demonstration. Evaluate usefulness later against concrete bottlenecks. Claude's running implementation assignment remains unchanged. This records the [owner's confirmation](sources/recursive-apply-simplification-20261019.md#first-demonstration-scope-after-independent-review), not a new implementation scope or experiment grant.
- **Owners and evidence:** [Apply concept](../design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026), [library walkthrough](../design/canonical-model/manager-apply-library-flow.md#current-library-walkthrough), and canonical product organization/run/accounting Rules. The prior Luna experiment is bounded evidence for prepared parallel work with separate copies, not this shared-workspace design or its scalability. Stop/capture mechanics, mapping schema and native message capabilities need concrete engineering validation. The implementation removed the sequential task machine, deliveries, reviews, expectations, reservations, rejection counters, the journal and its SQLite adapter without compatibility.
- **Runtime model (owner, 19 October 2026):** Self-started implementation, Manager, Verify and review Agents use Sonnet with no Codex or OpenAI inference and no silent model fallback; the implementation session configured `claude-sonnet-5-5` with high effort. This replaces the earlier Luna preference of the library walkthrough for the demonstration; the Codex adapter stays in the product. A complete real test run including both Applies has a shared hard wall-clock limit of 30 minutes, enforced outside the Agent prompts ([source](sources/recursive-apply-implementation-20261019.md)).


### DEC-057 Three fundamental starting situations

- **Status:** Accepted high-level direction, owner, 19 October 2026; subsequently clarified as three starting situations and only two processes, Apply and Extract ([original discussion](sources/three-starting-situations-20261019.md), [process naming and documentation instruction](sources/three-starting-situations-20261019.md#owner-selects-apply-and-extract)).
- **Statement:** Distinguish model without artifacts, model with artifacts, and artifacts without a model. The first two use **Apply: Model -> Artifacts**; the third uses **Extract: Artifacts -> Model**. Building and reconciling are starting states of Apply, not separate processes or commands. These are high-level product semantics, not implementation progress categories.
- **Apply without artifacts:** Root prepares its project skeleton and shared prerequisites; children build their areas recursively on the available foundations. Parents integrate upward. This clarifies dependency order without selecting a new scheduler or transferring child ownership.
- **Apply with artifacts:** Managers reconcile artifacts with the accepted model, including repair against unchanged intent. Missing artifacts are created, required existing artifacts are updated as needed, and obsolete realizations follow the existing removal rules. Parents investigate contradictory child results during Apply or Verify and escalate unresolved intent through Root.
- **Extract:** Root first derives a provisional responsibility/Package structure from the fixed artifacts and selected scope; form Managers from that candidate structure, then derive, review and integrate the model recursively with owner clarification and read-only Verify ([accepted case-3 walkthrough](sources/three-starting-situations-20261019.md#owner-confirms-the-extract-walkthrough)). Source boundaries and candidate context provide the children's investigation basis; wrong or missing responsibilities can require a visible correction to the outline. Existing artifacts remain unchanged during derivation. Model review assesses the proposal, Verify assesses artifact conformity against its exact candidate, and the owner adopts the selected result. Retain supported intent and known realization defects rather than changing meaning to manufacture a pass; later artifact repair belongs to Apply. This clarifies the existing [DEC-044](#dec-044-commands-reuse-the-manager-organization) direction without implementing Extract or its provisional-tree lifecycle.
- **Extract round refinement:** The owner [confirmed](sources/three-starting-situations-20261019.md#owner-confirms-structure-changes-between-extract-rounds) a stable provisional Manager structure within each round, with reviewed and integrated structural corrections adopted between rounds. Carry all derived knowledge, source and model references, findings and open questions to the resulting responsibilities, preserving their status, identity mappings and original evidence bindings. Rebind the next tree and contexts to the revised candidate; earlier positive assessments do not automatically approve it. The [owning workflow](../design/canonical-model/commands.md#keep-the-structure-stable-within-each-extract-round) distinguishes this candidate transition from final owner adoption and leaves technical transition/rebuild protocols open.
- **Extract completion refinement:** The owner [confirmed](sources/three-starting-situations-20261019.md#owner-confirms-extract-completion-criteria) that essential unresolved model questions block completion, while known, evidenced artifact defects do not by themselves. Completion requires a supported, adequately detailed, independently reviewed and integrated model and a completed read-only assessment against the exact candidate and fixed source basis. Material evidence gaps remain blockers; unsupported defect suspicions cannot be relabeled as known deviations. Root distinguishes model readiness for adoption, the actual Verify result and separately authorized later repair, reporting partial scope and retained questions honestly. This confirms [DEC-044](#dec-044-commands-reuse-the-manager-organization)'s adoption boundary; the [owning criteria](../design/canonical-model/commands.md#completion-and-adoption-with-known-deviations) add no implementation or new runtime status fields.
- **Extract to Apply:** The owner [confirmed](sources/three-starting-situations-20261019.md#owner-confirms-the-extract-to-apply-transition) joint adoption of the model and registry, carrying known artifact defects into the first regular authorized Apply, and treating them as repaired only after independent Verify supports their resolution. Adoption makes this case 2 for the selected scope. Preserve findings' model references, evidence and source bindings, reassess applicability against the actual starting artifacts and retain normal Execute-first participation, integration, verification and publication boundaries. Adoption closes no defect and starts no repair by itself; known violations gain no exception from Apply completion requirements. The [owning transition workflow](../design/canonical-model/commands.md#adopt-the-model-and-hand-findings-to-apply) leaves storage and transport details open and adds no implementation.
- **Implementation:** On owner instruction, a first bounded Extract is implemented ([runtime contract](../design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation)) and was run on the library ([record](../validation/library-extract-20261019.md)). Restructuring between rounds and the handoff of findings to Apply are not implemented.
- **Owner and scope:** [Vision: three starting situations](../vision.md#three-starting-situations) owns the high-level account. [DEC-056](#dec-056-recursive-reconciliation-with-a-shared-workspace) and the existing [Extract design](../design/canonical-model/commands.md#extract-the-existing-stand) retain their technical scope. This documentation decision adds no runtime, language construct, implementation assignment or provider/study authority.

### DEC-058 Apply always executes before independent Verify

- **Status:** Accepted direction, owner, 19 October 2026 ([discussion and explicit documentation-only instruction](sources/apply-execute-first-20261019.md)). No implementation is part of this decision. The owner then dispatched the implementation in the Claude session ([DEC-059](#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)); it is implemented and merged (PR #472) and demonstrated three times ([record](../validation/recursive-apply-repetitions-20261019.md)).
- **Decision:** Every Apply uses prepare -> Execute and integration -> capture one immutable candidate -> independent Verify -> correction or guarded publication. There is no initial Verify and no special route for case 1. Structural model, ownership and source-accounting preparation remain before execution.
- **Execute:** Each Manager examines its available artifacts against the model as part of reconciliation, preserves valid realization, creates missing artifacts and corrects deviations within its authority. The initial mandate supplies work without fabricated missing-artifact findings. Even an already conforming area participates; Execute requires assessment, not an edit.
- **Root and recursion:** Apply enters at Root and work proceeds through the existing fixed parent/child hierarchy, with results integrated upward. The owner [clarified the start rule](sources/apply-execute-first-20261019.md#owner-confirms-parent-foundations-before-child-work): a parent first provides the concrete foundation its children need, then their realization begins. An already suitable foundation can be preserved. Siblings may progress in parallel after their needed foundations are available; further dependencies remain parent-coordinated. Parents finish after child integration, so this is not a requirement to complete all parent work before children start, a Root-authored file plan, selective participation or a transfer of child ownership.
- **Existing foundations:** The owner [confirmed case 2](sources/apply-execute-first-20261019.md#owner-confirms-existing-foundation-assessment): parents assess actual existing foundations against the current model and known child needs within Execute, preserving or correcting them before child realization begins. Subsequent gaps and basis changes are parent-coordinated through the artifact owners. This local assessment does not replace the subsequent independent whole-candidate Verify and creates no initial Verify or separate process.
- **Independent assessment:** After Execute and integration, observe writer quiescence and capture the whole candidate. Fresh Verify sessions assess every responsibility, children before parents, and required checks examine that same stand. Findings return to the responsible owners; corrections receive a new whole Verify round. The existing independence, evidence, budget and publication boundaries remain.
- **Supersession and evidence:** Amends the entry order of [DEC-056](#dec-056-recursive-reconciliation-with-a-shared-workspace), keeping [DEC-057](#dec-057-three-fundamental-starting-situations)'s three starting situations and two processes. The [current concept](../design/canonical-model/manager-apply-concept.md#shared-working-tree-and-reconciliation-cycle) owns the accepted target; the [runtime contract](../design/canonical-model/manager-apply-runtime.md) and canonical self-model record its implementation. The decision itself created no runtime change, provider run or demonstration result.

### DEC-059 Enforced responsibilities, build start and an outside oracle

**Current disposition, 20 October 2026:** DEC-064 preserves enforcement and the outside oracle while replacing the mirrored/default and Root-only exact-file restrictions with explicit most-specific directory/file ownership for all Managers.

- **Status:** Accepted, owner, 19 October 2026, after the first real library demonstration ([verbatim owner source](sources/ownership-build-order-oracle-20261019.md), [validation record](../validation/recursive-apply-demonstration-20261019.md)). Implemented and merged (PR #472); demonstrated three times ([record](../validation/recursive-apply-repetitions-20261019.md)).
- **Responsibilities:** A parent must not take over its children's work or tell them they need do nothing. Under a declared layout, ownership is now enforced in the shared working tree: an Execute turn may write only its own areas and registry shard; everything else, including protected inputs and the children's areas, is read-only. Root owns the files directly in an artifact root and every file outside the roots; a subdirectory of an artifact root needs an owning Package directory or a persistent exception. Briefs name each child's areas and Definitions and the parent's role (supply foundations, coordinate, integrate). This amends DEC-056's "initially trust, enforce later" for projects with a declared layout; without a layout ownership stays trusted.
- **Build start:** Superseded within the same day by [DEC-058](#dec-058-apply-always-executes-before-independent-verify)'s single process, which the owner chose to implement right away: every Apply starts at Root, each child begins when its parent's first Execute turn has supplied the foundation, and there is no special route for areas without artifacts.
- **Oracle:** The independent library oracle observes once after each Apply, outside the process. It is no longer a required check and gives the Agents no feedback.

### DEC-060 Owners realize, others use; parents settle cross-area matters

**Current disposition, 20 October 2026:** DEC-064 preserves owner realization and parent coordination; its concrete Plan Pass replaces the planning-only-as-fallback choice below.

- **Status:** Accepted, owner, 19 October 2026, after the [three repeated demonstrations](../validation/recursive-apply-repetitions-20261019.md) ([verbatim owner source](sources/owner-realizes-others-use-20261019.md)). Implemented in the Briefs, the Verify guidance and the self-model, together with the first Extract.
- **Decision:** A Definition is realized by the artifacts of the Manager that owns it. Other areas use that realization rather than rebuilding it, and no Manager touches anything outside its own area. Cross-area matters between children are settled by their parent: it decides which child's realization provides a shared rule or interface and coordinates the others to use it.
- **Verification:** A realization of another area's Definition outside its owner's artifacts, or a change outside the Manager's own area, is a concrete model violation and therefore a blocking finding. The parent's fresh Verify of its children's interaction is responsible for detecting it.
- **Fallback:** A prior planning round to prevent such duplication is named as an option if this rule alone does not suffice. It is not adopted.

### DEC-061 What an extracted model contains

- **Status:** Accepted, owner, 19 October 2026, after the [first library Extract runs](../validation/library-extract-20261019.md) ([verbatim owner source](sources/extract-model-content-20261019.md)). Implemented in the Extract mandates, Verify guidance and Briefs, with proposed checks in the structure phase ([runtime contract](../design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation)); the old/new comparison on the library is [recorded](../validation/library-extract-comparison-20261019.md).
- **Project Package:** Extract's structure phase always proposes one Package that owns the project as a whole: its purpose, scope and non-goals, the delivery obligations (for example tests and documentation) and the technical boundary (for example language, build and test commands, permitted dependencies). Its Manager states these obligations in the model phase. Language v1 forbids Definitions in the model root and Root has no Package, so these obligations need a named Package; it may be a dedicated Package or one that also owns other behavior.
- **Binding versus realization choice:** A statement is binding when it is observable at the project's boundary (its interfaces, inputs and outputs, and the behavior a user or caller can see) or when the artifacts document it explicitly as intended (for example a README rule or a recorded decision). Internal structure, names, wiring, data structures and the order of internal steps are realization choices unless the artifacts document them as intended. The model can name such a choice as free, but does not make it binding.
- **Proposed checks:** Extract may propose required checks derived from documented build and test commands. They are a proposal for the owner's adoption and are not active before it.
- **Clarification from round 2 (implementation, 20 October 2026, pending owner confirmation):** Runs on a second project and three library runs showed that "observable" needs a boundary. The binding part is the contract callers rely on: inputs and their validation, returned and stored data shapes, status and exit codes, stored data and the order in which data was added. Observable details a caller cannot reasonably depend on are free unless the artifacts fix them: message wording and detail, generated identifier formats, which failure is reported when several apply, and field order. The project Package also owns the project-level artifacts directly in the artifact root, through a directory exception, and every Package owns artifacts or has children ([record](../validation/extract-round2-20261020.md)).
- **Consequence:** These criteria guide authoring and the independent model review alike. They refine [DEC-057](#dec-057-three-fundamental-starting-situations)'s Extract without changing its process, completion or adoption boundary.

### DEC-062 Start-stand drift is the owner's first work

**Current disposition, 20 October 2026:** Superseded by DEC-064: initial File check is strictly blocking and does not admit accounting repair work. Historical drift runs below remain evidence for the former contract.

- **Status:** Accepted, owner, 19 October 2026, after a drift scenario stopped in preparation ([verbatim owner source](sources/start-stand-drift-20261019.md)). Implemented in Apply's preparation and engine ([runtime contract](../design/canonical-model/manager-apply-runtime.md#candidate-accounting-and-protected-inputs)).
- **Decision:** Accounting drift of the committed starting stand that a Manager can repair in its own area and registry shard no longer refuses Apply under a declared layout, which names that Manager. Without a layout no Manager owns such a file, and preparation still refuses. Examples are a file that no shard registers, a registration of a missing file or of an unknown Definition, and a registration outside the registrant's area. Each finding reaches the owning Manager as part of its first Execute work, as DEC-058's "correct deviations within its authority" says. The usual whole-candidate Verify then confirms the repair.
- **Still refused:** control faults that no Manager can repair: an invalid or unreadable registry or layout, changed protected inputs, and a file that no Manager owns under the persistent layout (`layout.no_owner`). Only the owner changes the layout, and removing such a file is no repair. A Git-ignored untracked file is local to its developer rather than drift of the committed project; it still needs an entry in `.markitect/ignore.yaml`.
- **Supersedes:** the concept's "Preflight blocks on missing/duplicate registrations" for Apply. `markitect check` and Extract keep their own rules.

### DEC-063 The model is authoritative; defects escalate

**Current disposition, 20 October 2026:** Amended by DEC-064: questions and compatible clarification may proceed through parents to the user. Actual model error/contradiction still requires Root to stop. The blanket question ban and automatic boundary-underspecification refusal below are historical.

- **Status:** Accepted, owner, 19 October 2026, after the contradiction and owner-question scenarios of the Apply evidence ([verbatim owner source](sources/model-defects-escalate-20261019.md)). Implemented after the Apply evidence report ([#492](https://github.com/Glacius-Labs/Markitect/issues/492)): the Apply mandate states the rule, and Root's tool `model_defect` ends the run ([runtime contract](../design/canonical-model/manager-apply-runtime.md#reconcile-cycle)).
- **Decision:** Apply never repairs the model. What the model states always holds, and no answer may contradict it. A Manager that cannot settle a matter within the model escalates it to its parent. The parent settles it when the model allows, for example which child realizes a shared rule, a tool or environment question, or an internal technical choice. Otherwise it escalates further, up to Root.
- **At Root:** A model defect is reported to the owner and ends the run at once, visibly Incomplete with a model-defect report. A model defect is a model that contradicts itself, or a gap that changes behavior at the project boundary. Nothing is published, the work is retained, the owner corrects the model, and a new Apply follows. No run-only answer settles a model defect.
- **No owner questions in Apply (owner, 19 October 2026, [same source](sources/model-defects-escalate-20261019.md#follow-up-one-way-out)):** In the scenario RA-Q2, Root recognized a boundary gap but asked it as an ordinary owner question and continued with the answer. The owner therefore chose one way out: Apply asks the owner nothing mid-run. Whatever Root cannot settle within the model and the bound owner decisions is reported with the model-defect report, and the run ends. Extract keeps its owner questions.
- **No escalation:** A gap without effect at the project boundary is realization freedom ([DEC-061](#dec-061-what-an-extracted-model-contains)'s boundary criterion). The Manager decides it and documents the choice.
- **Amends:** Owner questions of [DEC-056](#dec-056-recursive-reconciliation-with-a-shared-workspace) and run-wide owner decisions of [DEC-053](#dec-053-owner-decisions-bind-the-whole-run). Bound owner decisions remain for decisions within the model and no longer settle a contradiction or a boundary gap of the model. Mid-run owner questions no longer exist in Apply.

### DEC-064 Checked recursive Apply passes and Manager teams

**Hardening refinement, 20 October 2026:** the owner [adopted active counterexample assessment](sources/terminology-and-file-accounting-20261020.md#owner-adoption-evidence-driven-apply-hardening). The [owning workflow](../design/canonical-model/manager-apply-workflow.md#evidence-driven-hardening-adopted-20-october-2026) specifies independently recorded probe evidence, parent challenge, unresolved-policy escalation and the dispositions of all prior Claude proposals. This refines the same Pass engine. The new handoff is prepared; PR #536 remains an incomplete candidate, and no benefit is claimed for untested batching or Luna.

- **Status:** Accepted by the owner, 20 October 2026. [Verbatim adoption and conversation](sources/terminology-and-file-accounting-20261020.md#owner-adoption-and-implementation-preparation). The base process merged in PR #530, Main `780dccb7`, with [first provider-run evidence](../validation/apply-pass-process-20261020.md) including known failures. The [runtime contract](../design/canonical-model/manager-apply-runtime.md) describes that baseline. The adopted hardening refinement still needs implementation and new evaluation; stable Apply, hosted CI and owner acceptance are not established by this concept preparation.
- **Behavior owner:** [Apply workflow and recursive Manager passes](../design/canonical-model/manager-apply-workflow.md). Apply is the complete workflow; Execute groups Plan, Prepare and Implement; Verify is an independent separate Pass afterward. All use one parent-first recursive base process with a shared result envelope/lifecycle, explicit basis acceptance, child work, bidirectional tree questions, parent acceptance of proposals and upward integration. Repeated purposes have new Pass identities; no separate correction engine is introduced.
- **Strict gates:** Model check and separate File check block before work and at accepted Pass boundaries. No automatic model/accounting repair or initial drift exception remains. New owned files may be created during work, but validly registered/covered before accepted basis/completion. Registered semantic artifact defects remain normal owner work.
- **Ownership:** Flexible repository layouts, repository-relative local registries, whole-directory and exact-file selectors, most-specific unambiguous ownership, enforced per-Agent container writes and project-wide reads. Managers author only their own shards; the Orchestrator alone produces the checked read-only versioned global view.
- **Results and communication:** Ready/Working/Waiting/Done scheduling; Problem as a structured report/event; separate Pass verdict and Success/Failed/Incomplete Run outcome. A Turn ending is never completion. Questions travel only between direct parent/child Managers; Root may ask the invoking AI/user. No answer overrides the model; Root aborts actual model errors/contradictions.
- **Completeness:** Every required Manager has its own concrete plan and realization, with parents reviewing fit and integrating rather than taking over child responsibilities. Fresh candidate-bound Verify retains genuine findings until independently closed. Correction uses the needed ordinary work Passes and complete fresh Verify. Clean means no unresolved genuine finding, followed by guarded Publish/read-back.
- **Tools and teams:** Small lookup/check/send/report methods, escalation through the parent, reusable method Skills and a standard Manager/Worker/independent-Reviewer template with configurable specialists. Helpers add no owners, global write authority or cross-tree channels.
- **Accepted clarifications, 20 October 2026:** [Owner instruction](sources/terminology-and-file-accounting-20261020.md#owner-adoption-three-implementation-clarifications) distinguishes immutable authority inputs from mutable owned registry metadata, allows a released Manager to handle routed questions read-only while productive dependencies remain blocked, and permits explicit independent disproof of an erroneous finding on an unchanged candidate. The same Pass lifecycle, genuine-finding blocks and sealed-result history remain intact; the handoff includes acceptance scenarios.
- **Supersession:** Supersedes DEC-062, the blanket question ban/boundary-gap rule in DEC-063, competing-name compatibility in DEC-042, earlier restrictive layout choices, planning only as a fallback, implicit Turn completion and conflicting discussion variants. Retains model authority, owner realization, fixed inputs, observed stop, finite limits, exact candidate checks, outside oracles and guarded publication. See the workflow's explicit preservation table.
- **Delivery:** The [new Claude hardening handoff](../development/claude-apply-hardening-handoff-20261020.md) prepares the owner-started bounded implementation/retest assignment. The [earlier implementation handoff](../development/claude-apply-implementation-handoff-20261020.md) retains its historical scope. The owner requested Main integration of this concept, not integration of draft PR #536's partial code; preparing the handoff starts no Claude/provider session or release. Shared Extract behavior remains protected, and historical results retain their original bindings.

### DEC-065 Work through the cockpit: agent-proposed model changes and configuration

- **Status:** Accepted by the owner, 20 October 2026 ([verbatim source and choices](sources/cockpit-commission-20261020.md#working-through-the-cockpit)). Implementation started in the cockpit stream; nothing is established yet.
- **Decision:** The owner works on the project through the cockpit. A prompt in the cockpit goes to a Claude or Codex Agent session that proposes a change to the canonical model; the Agent and Manager team configuration in `.markitect/execution.yaml` is edited through the cockpit as well.
- **Boundaries:** Every change is a proposal: the cockpit runs the Model check and the File check on it, shows its difference and writes it only after the owner's explicit acceptance, through Apply's guarded publication with read-back, without committing. The owner remains the author of intent; an Agent proposes, it never changes the model on its own. Agents run in Linux containers through the existing Runtime adapters and profiles ([DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers)) and write only the model and registry of their session's workspace. Each prompt is a provider turn whose usage is shown. Accepting a change starts no Apply.
- **Not adopted:** Government and delegated model maintenance ([IDEA-002](#idea-002-government-delegated-model-maintenance)), a personal AI beyond these sessions, approval reuse, and starting or stopping Apply from the cockpit.
- **Behavior owner:** the [cockpit concept](../design/cockpit.md#working-through-the-cockpit). Related: [IDEA-015](#idea-015-cockpit-with-a-conversational-personal-agent), whose chat-driven model adjustment this adopts in this bounded form, and [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing).

## Promises

These are the target outcomes Markitect is built to deliver. Each has the status **Target**: it is to be validated under [DEC-009](#dec-009-the-central-evaluation-question), and none is an established result. Sources are the [owner description](sources/owner-product-description-20261009.md) and the [confirmed summary](sources/idea-review-20261010.md#summary-confirmed-by-the-owner).

| ID | Promise |
|---|---|
| PRM-001 | Nothing in the modeled scope is silently forgotten. Affected obligations, files and responsible Managers come from the model, not from an agent's memory, and unknown impact remains visible. This covers declared relationships; dependencies in code are a known gap ([ENH-001](#enh-001-observed-code-dependencies-widen-impact)). |
| PRM-002 | Rules and policies are applied where they matter. They are compiled into the responsible Manager's context and checked by independent reviewers, so they are not overlooked when nothing forces the agent to see them. |
| PRM-003 | The repository stays contradiction-free, consistent, uniform and clean through many changes, so cleanups and refactorings become dependable. |
| PRM-004 | People concentrate on deciding and modeling. Human review moves from code diffs to model changes and escalated decisions while the code reliably reflects the model. The owner [restated the end goal](sources/foundations-and-trust-20261017.md#owner-statement) on 17 October 2026: routine human code review becomes unnecessary because the realization can be trusted. Independent Agent review remains part of the method; reduced human intervention is a target to validate, with the project's acceptance policy retained. |
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
| ASM-005 | Reviewers that differ from the implementer in model, not only in context, find more defects, because reviewers on the same model share blind spots. | Review proposal, endorsed for evaluation ("Modelle variieren und kombinieren und evaluieren"); to be tested with [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing); the Claude Code adapter of [DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers) lets the default reviewer come from another model family |
| ASM-006 | Forgetting (coverage), rather than misunderstanding, is the main problem. | Independent-assessment hypothesis (claim 17), not an owner comparison; untested ([source note](sources/dashboard-delegation-20261011.md#idea-01-assumption-source-check)) |

## Concepts

### CPT-001 Federalism

- **Status:** Accepted. The term is the owner's ([point 1](sources/idea-review-20261010.md#owner-response-to-the-five-points)); the structure comes from the [owner description](sources/owner-product-description-20261009.md).
- **Statement:**
  - Responsibility is spread over bounded areas, each with its own Manager, scope and context.
  - Each candidate and each integration result is assessed by an independent reviewer.
  - Parent Managers integrate the results.
  - A global consistency check compares all areas with the canonical model.
  - Trust comes from this structure, not from any single agent being infallible.

- **Owner restatement, 17 October 2026:** four eyes, responsibilities, delegation and escalation, divide and conquer, abstraction, need to know and coordinated lower-cost Agents are foundations of this organization. The [source](sources/foundations-and-trust-20261017.md#owner-statement) and [rationale](origin-and-rationale.md#foundations-restated-on-17-october-2026) connect them to the existing decisions; they add no new first-Apply mechanism.

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
- **Design (16 October 2026):** the Definition census of every delivery gives each local Definition a responsible Manager and an explicit disposition ([Apply concept](../design/canonical-model/manager-apply-concept.md#deliveries-and-evidence)).

## Future enhancements

These entries record agreed or endorsed direction. Recording an enhancement does not schedule it. The [delivery plan](../implementation-plan.md) owns scheduling and explicit package scopes; chats/PRs own candidate evidence. Earlier work-item links refer to the retired backlog. When an enhancement is scheduled, the entry gains a link to its work item, and the work item links back here.

### ENH-001 Observed code dependencies widen impact

- **Status:** The problem is accepted (owner, 10 October 2026); the approach is open. The owner: „wir sollten das verfolgen und uns hier etwas einfallen lassen“ ("we should pursue this and come up with something"; [follow-up dispositions](sources/idea-review-20261010.md#follow-up-proposals-and-owner-dispositions)).
- **Problem:** Declared mappings do not contain dependencies between files in code: calls, imports and shared types. Suppose one Manager changes a shared function that another Manager's realization calls. The second Manager's obligations can then be affected without appearing in impact.
- **Direction (proposal):**
  - Optional, language-specific tools or plugins observe such dependencies.
  - Their output enters impact as a fixed, digest-bound observed input.
  - Observation may widen impact. It never silently narrows it.
  - Core stays free of inferred source-code semantics ([AGENTS.md](../../AGENTS.md), [architecture](../architecture.md)).
- **Constraint (owner):** These dependencies are specific to each programming language. They cannot be checked the same deterministic way for every kind of file.
- **Historical work item:** IDEA-02 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Design link (16 October 2026):** observed code dependencies are planned as input to the Optimizer, which proposes missing model links ([north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions)).

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
- **Historical work item:** IDEA-03 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Design link (16 October 2026):** model lint is planned on registry signals ([north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions)).

### ENH-003 Test-to-statement binding and test review

- **Status:** Accepted direction, owner, 10 October 2026. It may become a variant or be built in.
- **Direction:**
  - Bind tests to the statements or checks they realize.
  - Add a review that asks whether a test actually verifies its statement.
  - Report statements whose only evidence is a test that has not been reviewed ([CPT-002](#cpt-002-tests-are-realizations-too)).
- **Historical work item:** IDEA-04 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Design (16 October 2026):** Definition assessments name the tests that would fail if the Definition were violated, and reviewers judge whether they discriminate ([Apply concept](../design/canonical-model/manager-apply-concept.md#deliveries-and-evidence)). Scenario Definitions in the model are a planned extension ([north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions)).

### ENH-004 Per-role agent configuration and model mixing

- **Status:** Accepted direction, owner, 10 October 2026. A separate configuration for every agent role was already planned.
- **Historical source snapshot (10 October 2026):**
  - `.markitect/runtime.yaml` binds one executor and one reviewer per Manager. Each binding has its own model, plus a reasoning effort for the App Server or model options for the process transport. One verifier is configured globally.
  - `config` generates a default profile for every role: Codex App Server, `gpt-6-luna`, `high` effort. Since RUN-01, a setup options record gives Managers, reviewers and the verifier their own provider (`codex` or `process`), model, effort and cost mode, and profiles can be mixed ([role profiles](../provider-adapters.md#role-profiles)).
  - PLAY-06 was integrated as [PR241](https://github.com/Glacius-Labs/Markitect/pull/241) at Main `d083c8f37b6bb1e09a9c5a7947c421651a7e755b` with [push CI38231391134](https://github.com/Glacius-Labs/Markitect/actions/runs/38231391134). Playground manifests selected role executor/model/effort; setup confirmed actual runtime arguments, effective cost mode and requested process version, and reported the configuration. Provider-free checks and fake setup proved this configuration path, not provider honor or a method study. This evidence remains historical and is not the current implementation. The rebuilt source and its sequence belong to the [delivery plan](../implementation-plan.md).
  - Helpers inherit their parent's configuration.
  - The Manager implements its own slice; there is no separate implementer role.
- **Direction:**
  - Configure model, reasoning effort and provider per role (Manager or implementer, helper, reviewer, verifier), and possibly per Manager.
  - Allow mixing, for example one model implements and another reviews.
  - Evaluate combinations as an experimental factor ([ASM-005](#assumptions-and-hypotheses), [OQ-003](#open-questions)).
- **Historical work items:** RUN-01 (merged as pull request #104) and PLAY-06 (merged as pull request #241) in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Rebuild note (16 October 2026):** the rebuilt first strategy has two Agent roles, Manager and reviewer, configured per Manager; there is no separate verifier Agent, because the Orchestrator performs the final evidence check ([DEC-036](#dec-036-adopted-design-refinements-for-the-first-apply)).

### ENH-005 Exchangeable executor for the delegated method

- **Status:** Endorsed by the owner on 10 October 2026. The predecessor source's RUN-01 and RUN-02 increments were merged; their recorded CI result at `e9979084dd085a2cd2715828358ed384feba0383` remains evidence for that source only. This is the product side of [DEC-010](#dec-010-evaluate-the-method-separately-from-its-execution-runtime). The current rebuild and its evidence are owned by the [delivery plan](../implementation-plan.md).
- **Historical motivation (10 October 2026):** In the predecessor source, every delegated role ran through one native runtime, the Codex App Server adapter. Runtime and environment failures therefore blocked both product use and evaluation of the method.
- **Historical source snapshot (10 October 2026):**
  - Internal role invocations use the `projectrun.Invoker` interface. That interface has two transports:
    - a hand-configurable process transport with a versioned JSON protocol (`agent-execution/v1alpha1`);
    - the native Codex App Server transport.
  - Repository end-to-end tests drive Plan, Run, Review, Verify and guarded Apply through a scripted, model-free process executor.
  - Since RUN-01, the role options accepted by `config` select a process executor per role, `costMode: unmetered` covers executors that cannot report token usage, such as scripts or people, and `markitect-exchange-executor` is a reference adapter that hands each invocation to a person, a script or another session through files ([bring your own executor](../provider-adapters.md#bring-your-own-executor)).
  - RUN-02 adds deterministic Manager/reviewer packet export, read-only listing and digest-bound response or fixed Git-candidate submission through CLI/MCP. Submission stores data; a later explicit scheduler continuation validates the response and advances Manager/review work. Verify uses the existing local path, and guarded Apply remains a separate operation. Git candidates include grounded deletions. New Plans bind method policy independently of an installed executor, while actual internal invocations retain executor identity checks.
  - Passed review reuse bound the review's retained parent, scoped bytes/modes, typed operations, deletion preimages and model/helper policy. Integration/rework retained the actual materialized merged input as its direct parent. This was provider-free capability and source-bound technical proof; packetized Verifier, successful interruption recovery, provider studies and human acceptance were separate. These details describe the predecessor implementation, not current source.
- **Direction (proposal):**
  - Separate the deterministic method from the execution runtime.
    - The method covers the compiled model, coverage, impact, Manager scopes and briefings, required reviews, candidate validation, Verify and guarded Apply.
    - The runtime covers process launch, sandboxing, provider sessions and recovery of agent turns.
  - Expose a stable executor boundary. Through it, Markitect hands each role a deterministic work packet: scope, context, obligations, required checks and required reviews. It then accepts the resulting candidate and review verdict and validates them with the same rules as today.
  - The native Codex adapter becomes one executor among others. Further executors, for example Claude Code, can be added without changing the method.
  - Authority does not change. Scope checks, fixed snapshots and guarded Apply apply to every executor, and submitted reviews remain AI evidence, not human acceptance.
- **Current design (17 October 2026):** The rebuild defines a provider-neutral session port `run.Sessions` and two container-based adapters, Codex App Server and Claude Code ([DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers), [rebuild plan](../history/apply-design-before-dec-056/manager-apply-rebuild-plan.md#session-adapters)). The [delivery plan](../implementation-plan.md) owns implementation status and sequencing. This design is not evidence that provider execution or Apply has been implemented or accepted.
- **Historical work items:** RUN-01 (merged as pull request #104), RUN-02, RUN-03 and RUN-05 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml).
- **Design link (17 October 2026):** the Claude Code session adapter moved from the later extensions into the first version ([DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers)).

### ENH-006 Configurable teams for perspectives and four eyes

**Selected slice:** [DEC-043](#dec-043-project-teams-review-arbitration-and-product-defaults) now accepts named reviewer teams, an arbiter and project defaults. The earlier deferral below records the 16 October decision; remaining team variants retain their later status.

- **Historical status (16 October 2026):** Endorsed, owner, 16 October 2026 ([source](sources/organization-and-trust-20261016.md#retain-objections-and-team-design-questions)). At that time the first strategy was Manager plus independent reviewer, with richer teams deferred until after the two-Apply demonstration. DEC-043 supersedes that deferral for named reviewers and arbitration; other team variants remain deferred.
- **Statement (owner):** „Gerade Dinge wie die Erfahrung, verschiedene Blickwinkel bzw Vier Augen Prinzip (realisierbar durch konfigurierbare Teams) etc finde ich sehr wichtig und diese sollten nicht vergessen werden“.
- **Direction:** Configurable teams carry distinct perspectives and the four-eyes principle without a Manager hierarchy per perspective. Roles differ in their question, context, tools and model profile ([organization and trust](reflections/organization-and-trust-20261016.md#different-perspectives-through-configurable-teams), [team variants](../design/canonical-model/manager-realization-north-star.md#team-variants)).
- **Related:** [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing), [IDEA-001](#idea-001-the-model-as-institutional-memory-for-stateless-agents).

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [ENH-001](#enh-001-observed-code-dependencies-widen-impact) |
| OQ-002 | Which metrics and thresholds show that a model is too fine or too coarse? | [ENH-002](#enh-002-model-quality-linter-and-metrics), [DEC-005](#dec-005-model-granularity-is-the-authors-responsibility) |
| OQ-003 | Which model tiers suffice for which roles, and does total cost actually fall? | [ASM-003](#assumptions-and-hypotheses), [ENH-004](#enh-004-per-role-agent-configuration-and-model-mixing) |
| OQ-004 | How much independence does a different reviewer model add beyond fresh context? | [ASM-005](#assumptions-and-hypotheses) |
| OQ-005 | Should evaluation focus on a series of changes on a long-lived project? If so, what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden holdout checks? | [DEC-009](#dec-009-the-central-evaluation-question), [PRM-001](#promises) |
| OQ-006 | How much current-state context does a bounded Manager need, so that limited responsibility does not become artificial blindness? The original product chat's hypothesis: shared contracts, plus current-state context on demand (Concepts note C19). Design: context packages plus recorded targeted reads ([Apply concept](../design/canonical-model/manager-apply-concept.md#context-packages)); evidence from real runs is pending. | [CPT-001](#cpt-001-federalism), [CPT-004](#cpt-004-the-specification-stays-effective) |
| OQ-007 | How reliable is the native delivery path? The combined A01 native smoke passed once (Actual28, on a Windows host) after 27 failed attempts, was not repeated, and never ran on Linux ([A01 record](../validation/a01-native-smoke-20261010.md)). The rebuilt runtime measures this again in containers, starting from the [handshake record](../validation/runtime-handshake-20261017.md). | [DEC-013](#dec-013-linux-first-for-tests-and-the-playground), [ENH-005](#enh-005-exchangeable-executor-for-the-delegated-method) |
| OQ-008 | Should the Manager organization become the application's entry point, with the canonical model as its foundation, and would that better support a later Government axis? Owner framing, 16 October 2026: „Eventuell wäre das sogar die bessere Betrachtungsweise“ ([source](../design/canonical-model/sources/manager-realization-20261016.md#apply-first-and-delegation-from-the-root)). The north star keeps it with the [later extensions](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions). | [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis), [DEC-026](#dec-026-apply-comes-first-and-changes-only-artifacts) |
| OQ-009 | How do DEC-018 and DEC-021 to DEC-023, written for the earlier Project model, map to language v1 and the registry once computed scope is built? | [DEC-018](#dec-018-the-knowledge-graph-explains-impact), [computed scope](../design/canonical-model/manager-apply-concept.md#toward-computed-scope) |
| OQ-010 | How can a Codex subscription login, rather than an API key, serve several concurrent session containers without racing on refresh-token rotation? A candidate is Orchestrator-owned tokens through the App Server's external token refresh; a smoke with concurrent sessions decides. | [DEC-037](#dec-037-agent-sessions-and-checks-run-in-linux-containers) |

## Ideas

### IDEA-001 The model as institutional memory for stateless agents

- **Status:** Proposed by the 10 October review. On 16 October 2026 the owner called experience very important and asked that it not be forgotten ([ENH-006](#enh-006-configurable-teams-for-perspectives-and-four-eyes)); the [experience reflection](reflections/experience-without-implicit-rules-20261016.md) develops it. Not yet decided.
- **Idea:** In human organizations, management structures work partly because people remember and keep responsibility over time. Agents keep neither. The canonical model can supply this institutional memory and lasting responsibility. That would make a management hierarchy workable for stateless agents and would support [ASM-004](#assumptions-and-hypotheses).

The ideas below come from the owner's contributions of 9 October 2026. The product chat handed them to the Concepts and Ideas chats, which recorded them. They are parked under [DEC-011](#dec-011-detailed-planning-of-future-ideas-waits): recorded for later examination, not planned. Quoted German is the owner's wording unless marked otherwise. "Open" items are questions raised by the Concepts or Ideas chat, not by the owner, unless marked as the owner's. The [sources](sources/README.md) keep the full text and the chats' analysis. In these ideas, "user" means the person who uses Markitect to run a project.
- **Planned extension:** experience notes are recorded in the [north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions).

### IDEA-002 Government: delegated model maintenance

- **Status:** Owner idea (Concepts `USER-20261009-02`; notes C20 to C22). The owner describes it as an earlier idea with a relatively immature implementation somewhere on a branch. An evaluation was commissioned ([DEC-012](#dec-012-government-ideas-evaluation-commissioned)). [DEC-004](#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) places it on the later axis. Implementation is planned as GOV-01 after Wave 2 ([DEC-019](#dec-019-government-code-starts-after-wave-2)).
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
- **Historical source snapshot (9 October 2026):** `.markitect/runtime.yaml` held a strictness setting for the predecessor engine. That file is not part of the current source. This idea remains open; the [delivery plan](../implementation-plan.md) owns the current rebuild scope.
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
- **Related:** The [operating methodology](../vision.md#the-delegated-method) already returns unresolved intent, ownership or authority to its owner. This idea adds a case procedure.
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

- **Status:** Owner idea (Ideas `IDEA-20261009-018`; Concepts C34). Part of [VIS-001](#vis-001-cockpit-personal-ai-and-autonomous-project-organization). Its model adjustment through a cockpit chat is adopted in bounded form by [DEC-065](#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration) (20 October 2026); search and a personal agent remain parked.
- **Idea:** A cockpit with a chat interface, connected to an AI agent. The agent can search the system for the user, adjust the model, or show the user directly what interests them in the cockpit.
- **Provisional boundary (attribution unclear; treat as a proposal until the owner confirms it):**
  - Chat and cockpit refer to the same model, decisions and evidence.
  - The agent acts only within delegated authority to change the model, and unresolved material decisions stay visible.
  - A chat request alone grants no new authority.

  The Concepts chat records this as a provisional derivation by the owner („vorläufige Nutzerableitung“). The Ideas chat records it as the Concepts chat's interpretation. No verbatim owner wording for it exists.
- **Source:** [personal agent cockpit](sources/concepts-chat-20261009/personal-agent-cockpit-20261009.md)

### IDEA-016 Optional offline claim and evidence classifier

- **Status:** Proposed by the research item R01 (9 October 2026); not evaluated by the owner.
- **Idea:** The R01 research assessed a strongly typed AI tool for Markitect. It recommended against using it to replace the compiler or Manager gates. An optional, offline classifier of claims and evidence could be examined later.
- **Source:** [optional research R01 to R03](../research/optional-research-r01-r03-20261009.md).

### IDEA-017 Operational check through the Managers

- **Status:** Owner idea and clarification, 16 October 2026 ([idea](sources/experience-discussion-20261016.md#experience-selection-and-runtime-observations), [clarification](sources/experience-discussion-20261016.md#clarification-logs-support-a-check-or-doctor)). Deferred by the [delivery plan](../implementation-plan.md).
- **Idea (owner):** „Was wäre wenn man Logging entsprechend aufbaut, sodass Logs aus laufenden Systemen ebenso durch die Manager geprüft werden können, sodass dort auch diskrepanzen aufgedeckt werden können?“
- **Clarification (owner):** „Die Logs meinte ich eher als eine Art check oder doctor oder sowas über die Manager. Nicht Erfahrungsbezogen“.
- **Developed in:** the [experience reflection](reflections/experience-without-implicit-rules-20261016.md#operational-check-through-the-managers), and planned in the [north star](../design/canonical-model/manager-realization-north-star.md#later-operations-and-extensions). It extends [IDEA-010](#idea-010-monitoring-logs-and-data-collection).

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
