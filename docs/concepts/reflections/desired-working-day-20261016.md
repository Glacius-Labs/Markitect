# A working day with delegated realization

Date: 16 October 2026. Desired user experience developed in the [owner discussion](../sources/working-day-discussion-20261016.md). The owner asked to preserve the scenario and assess its prerequisites. The concrete interaction choices below are assistant proposals, not additional accepted decisions or delivery commitments.

The person changes and approves intent. The organization realizes it, handles ordinary implementation and review work, and requests a decision when its authority or the model is insufficient. The person can inspect progress without coordinating every Agent exchange.

This develops the [organization and trust reflection](organization-and-trust-20261016.md). The [Apply concept](../../design/canonical-model/manager-apply-concept.md) owns realization behavior. The [concept register](../register.md) remains the owner of decisions and status.

Technical consolidation on the same date gives [configuration](../../design/canonical-model/manager-apply-configuration.md) ownership of exact overrides and the [execution contract](../../design/canonical-model/manager-apply-execution.md) ownership of budget, time, retry and lifecycle rules. The examples below preserve the working-day rationale, not a parallel runtime specification. The dated prerequisite assessment remains historical; proposed technical contracts are not implementation evidence.

## Morning model change

The owner changes the library's admission limit from five open loans to three. Existing loans remain intact and returnable. Draft editing does not itself start realization. An explicit instruction starts Apply against a selected model revision and the existing realization.

A short acknowledgement states the intended change, selected revision, scope and existing execution authority. It should not request permissions already granted. The first concept reconciles the full selected local model, including reviewed no-change results; it does not yet promise that only changed Packages need assessment.

The root delegates outcomes to the responsible Managers. Lending owns borrowing behavior, while related membership, catalog and reminder obligations receive appropriate assessment. The first strategy permits one active writing task. Work can advance through ready tasks when another task is waiting; the experience does not require concurrent writers.

The owner sees a compact status such as:

> In progress: lower the admission limit to three. Lending is updating its realization; related return and reminder behavior is being assessed. No owner decision is currently needed.

The task tree, actual prompts, candidate differences and evidence remain available on demand. The overview derives from the same history and must distinguish observed progress from an Agent's prediction.

## Decision authority and escalation

The [authority discussion](../sources/working-day-discussion-20261016.md#decide-within-authority-and-escalate) proposes two separate decisions: who may resolve an issue, and when the owner needs notification. A question can require owner authority without requiring an immediate interruption. A serious execution failure can require prompt notification even when the business rule is clear.

### Route the decision by mandate

| Situation | Responsible action | Library example |
| --- | --- | --- |
| The answer already exists | Retrieve the relevant model or applicable decision and follow it. | The model already allows a suspended member to return a book. |
| The choice is within implementation freedom | The Manager decides, explains material choices and submits the result to review. | Structure the borrowing check within the required architecture and behavior. |
| Several areas must coordinate | The nearest common parent coordinates within its own mandate. | Lending and Members agree how the existing membership-status contract is supplied. |
| The choice exceeds the mandate | Escalate to the authorized decision owner, ultimately the human owner. | Introduce an exception allowing another loan after the limit has been reached. |

Hierarchy alone does not grant more authority over intended behavior. The root cannot invent a missing business rule because its children disagree. Coordination does not transfer artifact ownership or authorize a model edit during Apply.

Before escalating, a Manager asks:

1. Is knowledge missing, or is a decision missing? Use bounded reading, checks and organizational questions to obtain existing information first.
2. Which rule or explicit implementation freedom permits the choice? Ground material decisions in that basis.
3. Does the choice preserve binding meaning, obligations and agreed limits? Technical convenience does not authorize weakening promised behavior.

Several valid implementations do not require an owner decision. Several materially different business policies without an authorized selection rule may require one. Confidence in an answer never supplies missing authority. Uncertain authority should be clarified through the organization before taking the dependent action.

### Keep the mandate small and explicit

Use common project rules with few Manager-specific exceptions. Derive the effective mandate from the existing model and execution configuration rather than maintaining a parallel policy document. Four parts suffice for this proposal:

- Responsibility: the outcomes and artifacts owned by the Manager.
- Decision freedom: implementation choices it may make locally.
- Reserved decisions: choices requiring another authorized actor.
- Limits: budgets, permitted actions and escalation conditions.

### Derive the Lending mandate from its sources

The [concrete mandate discussion](../sources/working-day-discussion-20261016.md#derive-a-concrete-lending-mandate) proposes a readable view assembled from existing sources. Each displayed obligation or permission identifies its source and bound revision. The view is not another authored policy or a selected configuration schema.

An illustrative view for Lending reads:

> **Assignment:** Realize lending as code, tests and documentation, including integration of lending/reminders through its responsible child Manager.
>
> **Binding basis:** Apply your local Definitions and connected obligations, including borrowing policy, membership eligibility, copy availability and returns. Their original passages are supplied in context.
>
> **Decision freedom:** Choose internal implementation and tests within the applicable architecture and interface constraints. Do not introduce additional business restrictions.
>
> **Write authority:** Change artifacts assigned to you. Reserve new paths through the Host. Request changes to another Manager's artifacts through the common parent.
>
> **Limits:** This Apply changes artifacts only. The model, execution configuration and your authority remain unchanged. Effective budgets and required independent reviews apply.
>
> **Escalation:** Retrieve missing context first and coordinate dependencies internally. Present missing business decisions, contradictory obligations or necessary mandate changes with a concrete proposal to the authorized decision owner.

The maintained sources are:

| Information | Single source to maintain |
| --- | --- |
| Loan limits, due dates, membership eligibility and return behavior | Canonical model Definitions. |
| Required outputs, architecture and binding interfaces | Canonical model Definitions. |
| Package Managers and parent relationships | Organization derived from the acquired Package structure under the realization strategy. |
| Actual artifact ownership | Versioned realization registry. |
| Teams, execution profiles, budgets and permitted creation areas | Project execution configuration outside canonical Definition roots. |
| Artifact-only Apply, mandatory review and publication guards | Shared operation contract enforced by the Host, not repeated per-Manager configuration. |
| This assignment's goal and additional constraints | Assignment/task records bound to the effective inputs and existing authority. |
| Timing and grouping of notifications | Notification settings, separate from decision permissions. Their exact format remains open. |

For example, [loan-policy](../../design/canonical-model/library-example/m1/model/lending/loan-policy.yaml) owns `spec.maxOpenLoans`, currently five in M1. The mandate refers to that policy; it does not independently prescribe five. A context view can display the bound value. Changing the canonical value to three gives the successor Apply its new basis without editing a second mandate rule. Existing original model text remains available; a summary does not replace it.

Likewise, the operation contract establishes that a Manager writes only its own artifacts, and the registry identifies those artifacts. Instructions can explain this boundary but cannot expand it. Notification preferences never expand decision authority. The reviewer receives the same source-bound mandate, current model, actual candidate and relevant explanations; an implementer's self-authored permission is not evidence of authority.

If the project later defines durable rights to change business rules, those rights belong to its intended governance in the canonical model. Executor choice and budgets remain execution configuration. This does not add delegated model-change rights to artifact Apply or require a general permission language for the first implementation.

Configuration and assignment constraints cannot silently weaken binding model obligations or mandatory operation gates. Conflicting inputs must be exposed for resolution. A displayed mandate preserves the sources and their distinct roles rather than introducing a precedence rule that lets one input erase another.

### Resolve conflicts through applicability and authority

The [conflict discussion](../sources/working-day-discussion-20261016.md#handle-conflicting-inputs) proposes checking applicability and compatibility before escalating. Do not introduce a blanket ordering such as model over assignment over configuration: it could silently discard an authorized request to change the model. Each source has its own responsibility. The model describes intended behavior, configuration supplies permitted execution choices, the assignment specifies the requested outcome, and the operation contract fixes boundaries such as a read-only model during Apply.

A conflict exists when applicable requirements cannot be satisfied together in the same case. First check their scope, conditions and bound revisions. Different statements about different situations need not conflict. A default and an explicit override can coexist under the documented configuration rules; that is not permission to invent precedence for arbitrary prose.

| Situation | Treatment |
| --- | --- |
| A project default selects profile A; an exact Manager override selects B | Apply the defined override rule and record the effective configuration. |
| The model leaves an implementation choice open; the assignment adds a permitted constraint | Apply the compatible constraint within the existing mandate. |
| The budget may not cover the requested work | Assess feasibility and authorized alternatives. Do not weaken business behavior or expand the budget silently. |
| The model allows five loans; the assignment requests three | Determine whether this authorizes a model change or conflicts with the fixed basis of an artifact-only Apply. |
| Configuration requests skipping a mandatory review | The Host rejects the invalid setting; it does not silently choose a substitute configuration. |

Use existing authority for determinate cases: retrieve missing context, distinguish a stale report from the bound model, apply declared configuration resolution, or perform an already authorized change at its proper source. Record the treatment briefly without asking the owner to approve the same decision again. An apparent wording correction that changes meaning is not merely cleanup.

For a real unresolved conflict, use the existing question or finding record with five items:

1. The exact conflicting source passages or values and their revisions.
2. The concrete case in which they cannot both hold.
3. The dependent work and scope that cannot complete.
4. A proposed correction at the proper source.
5. The actor authorized to decide that correction.

The responsible Manager investigates; a common parent coordinates cross-area cases. Dependent work waits while other ready work can continue within its mandate. An unresolved required conflict blocks aggregate success. No separate conflict database or Manager hierarchy is needed.

For example, “Change the model to three loans and realize it” already states the owner's decision. Within the existing mandate, prepare or perform the model change through the separate model operation and start Apply on the new basis. Do not ask again whether three was intended. By contrast, “Just limit the code to three” during an Apply bound to five is not local implementation freedom. Clarify whether a model change is authorized rather than silently changing behavior or discarding the instruction. Artifact Apply itself remains model-read-only in both cases.

A rejected invalid configuration needs a precise diagnostic, not necessarily a new business decision. Correct it autonomously only when the replacement is determinate and authorized. Otherwise identify the missing choice or authority. Record a new effective input binding where required; do not mutate fixed run inputs under the label of conflict resolution.

Independent review checks that the resolution considered all applicable obligations, came from an authorized actor and changed the correct source. Removing one conflicting statement from a prompt does not resolve the underlying conflict. Deterministic validation handles structural cases; semantic compatibility remains a reasoned, reviewed judgment.

Internally resolved cases appear in the report with their basis. Questions needing owner judgment use the concrete question format below and the separate notification policy. The intended sequence is: establish applicability and compatibility, use existing authority, and escalate only the missing decision or permission.

### Choose notification timing separately

| Situation | Owner attention |
| --- | --- |
| A decision blocks the whole assignment, or waiting has significant consequences | Send a timely, focused decision request. |
| A decision blocks part of the work while other useful work remains | Keep it visibly pending and group it with other questions at an appropriate time. |
| The Manager decided within its mandate | Preserve the reasoning for inspection and the result report; no answer is required. |

State when an answer is needed, if there is a real deadline, and what happens without it. Do not invent urgency. Grouping notifications does not permit dependent work to proceed on an assumed answer. Silence is not consent, and a required unresolved question still blocks aggregate completion. Significant execution incidents can need prompt notification independently of whether an owner decision is needed.

### Review the authority as well as the result

The independent reviewer examines material decisions against both the current obligations and the effective mandate: did the Manager have authority to make this choice, and does the result still satisfy the binding requirements? Relevant explanations and mandate references belong in existing reports and evidence, not a new approval system for every implementation detail.

An unauthorized assumption becomes a finding and prevents acceptance of the affected result until resolved. A reviewer cannot grant missing authority by approving the code. The Host enforces structural limits such as write ownership, budgets and operation targets; semantic judgments about authority remain reviewable but fallible Agent work.

The intended balance is symmetrical: Managers explain why they may decide locally, and an escalation explains why the organization cannot resolve the issue within its mandate. Neither constant questioning nor silent expansion of authority is a successful outcome.

## A question requiring the owner

Suppose a broader library also supports renewals and the model does not say whether a member already above the new limit may renew. This is a hypothetical extension: the authored M1 explicitly excludes renewals. The five-to-three change in the existing example does not require inventing this ambiguity.

If a binding answer already exists, Managers use it. If a necessary decision is missing, the responsible Manager routes a concise question through its parent:

> Decision needed: may members above the new limit renew existing loans? We suggest allowing renewals under the existing renewal conditions, because the new limit concerns additional borrowing. The answer affects renewal behavior and its checks. Unrelated assessment can continue.

The question includes the source of the ambiguity, recommendation, alternatives and consequences, affected work and why it exceeds the current task's authority. Silence is not approval. A required unresolved question prevents aggregate completion, even if other work finishes.

An answer changing intended behavior is recorded in the canonical model through a separate authorized model operation. Apply cannot edit its read-only model. A new Apply binds the updated model and reassesses; old candidates and evidence remain inspectable but do not automatically count as current approval. An answer that only points to an existing rule can instead supply recorded context without changing intent.

### A question the owner can answer directly

The [question-format discussion](../sources/working-day-discussion-20261016.md#make-questions-directly-actionable) proposes five parts. The owner should be able to decide without reconstructing Agent history or drafting the resulting model change.

| Part | Required content |
| --- | --- |
| Decision | One concrete question that needs the owner's judgment. |
| Reason | The missing or conflicting model passages and why the existing mandate cannot resolve them. |
| Recommendation and alternatives | A recommended answer, its consequences and meaningful alternatives. Identify alternatives that still need elaboration. |
| Exact model change | The proposed wording and the identified passages or structured values it replaces or adds. Show the complete intended edit, not merely a paraphrase of its effect. |
| Effect on work | Which tasks wait, which can continue, and what accepting the proposal will trigger. |

For a context-only question, explicitly state that no model edit is proposed. A recommendation is not a selected answer, and silence grants no approval.

### Example decision request

Assume an intentionally inconsistent future library draft says both that existing loans remain intact and that members may never have more than three open loans. The authored library example already resolves this case; this illustration does not change it.

**Decision needed: existing loans above the new limit**

The draft lowers the limit from five to three but contains the two conflicting statements above. Lending needs to know what happens to a member who already has four loans.

**Recommendation A: apply the limit when admitting new loans.** Existing loans remain intact. Additional borrowing is refused until the resulting count would fit the limit; returns remain possible.

**Proposed replacement for the conflicting admission passage:**

> A new loan is allowed only if the resulting number of open loans is at most three. Lowering the limit leaves existing loans unchanged. Returns remain permitted regardless of the current loan limit.

Other borrowing restrictions remain in force. In an actual request, identify the exact Definition and prior passage and include any accompanying structured-value edit. Where the policy already stores the limit as a structured value, show that value change and retain the model's reference to it rather than introducing a duplicate numeric authority in prose.

**Alternative B: require existing loans to be reduced as well.** This still needs decisions about deadlines, which books must be returned and enforcement. Selecting elaboration of B does not approve those unspecified consequences.

**Work impact:** Lending waits for this decision; independent ready work may continue. Accepting A authorizes the displayed model edit within the existing mandate. Once the authorized model operation completes, a successor Apply uses the retained work and assesses it against the new basis.

**Responses:** Accept A; elaborate B; decide differently; defer.

### Bind the answer and report the continuation

Record the question identity and version, assignment and affected tasks, exact model basis and locations, proposed edit, and the owner's answer with its source and authority. Existing question and history records can carry these references; this does not require a separate decision system.

Accepting A authorizes the exact displayed proposal within the existing mandate. Do not request the same confirmation again. A free-text answer can authorize a different edit when its meaning and scope are clear; otherwise return only the remaining material ambiguity for resolution. Elaborating an alternative authorizes preparation, not adoption. Deferral leaves the question open.

A changed proposal gets a new version. An answer to an older version must not approve a revised proposal implicitly. Before applying the edit, recheck its model basis and targets. If intervening changes make the proposal stale, preserve the answer and assess applicability; present any materially changed choice again rather than silently rewriting its meaning.

Distinguish answer received, authorized model edit recorded, and successor work actually started. Link the decision to the resulting model revision and successor run. Failure to record the edit or start the run must remain visible; an accepted answer alone does not prove either happened.

After those steps have succeeded, a concise acknowledgement can read:

> Decision adopted and the model updated as shown. The previous Apply has been superseded. Its successor uses the retained work and assesses it against the new model. Lending has resumed. No further decision is required.

The continuation below governs retained work and renewed approval. The acknowledgement reports observed state; it does not grant new merge, release or deployment authority.

## Continue the assignment across model changes

The [follow-up discussion](../sources/working-day-discussion-20261016.md#continue-after-an-owner-answer) proposes keeping the user's assignment stable while recording each Apply against its own fixed inputs. A new model basis requires a new Apply run, not recreating all existing work. This is a proposed interaction contract; it does not change the current Apply contract or establish implemented reuse.

| Level | What persists or changes |
| --- | --- |
| Assignment | The user's intended outcome, such as lowering the admission limit, remains the same assignment across questions and successive runs. It provides continuity and references the mandate and its recorded changes. It grants no extra authority. |
| Apply run | Each run binds its own model, configuration, source base and candidate history. New intent starts a new run linked to the previous one; no answer silently replaces a running input binding. Ordinary rework within one fixed basis remains a task attempt, not a new assignment. |
| Existing work | Exact candidate artifacts and their matching registry can become starting material for the successor. Their provenance, incompleteness and prior findings remain visible. Adoption as starting material is not approval for the new basis. |
| Approval and evidence | Reviews and checks retain their original subjects and inputs. They are historical evidence, not automatically current approval. Required assessment, review and verification must support the new result. |

The assignment can initially be a stable identity and links over existing records. It does not need another scheduler, hierarchy or canonical Kind.

### From question to continuation

1. **Record and route the question.** Give it a stable identity, source passages, affected task, requested decision and proposed answer. Only dependent work waits; other ready work may proceed within the fixed run basis. A required unanswered question prevents aggregate completion.
2. **Classify the answer by its effect.** A reference to an existing binding rule can supply recorded context in the same run. A new or changed obligation requires a model change. Calling an answer a clarification does not exempt it from this distinction. Deferral leaves the dependent work open; silence never resolves it.
3. **Make changed intent explicit.** Where useful, include the exact proposed model edit in the question so one answer can authorize both the decision and that edit. Prepare a revised proposal when the answer differs. Ask again only if the resulting change is not covered by the answer and existing mandate. Apply remains artifact-only; an authorized model operation owns the edit.
4. **Establish a safe handoff.** Prevent further dispatch and publication by the superseded run. Stop or finish active writing under controlled conditions and retain its observed candidate state. An uncertain writer must be reconciled before workspace handoff. Keep the old run and its incomplete outcome; do not relabel it successful.
5. **Prepare the successor from retained work.** Capture the new model and current repository basis. Explicitly select compatible candidate artifacts and registry changes from the predecessor, with their exact bases and findings. Validate ownership, current scope and composition; conflicting work returns to its accountable Manager. Incomplete work can be useful starting material but remains unapproved.
6. **Reassess and complete.** Supply the answer, model difference and previous work to the relevant Managers. Retain the initial policy of full selected-model reconciliation, allowing justified no-change deliveries. Perform required independent reviews, integration, checks and final verification against the current subjects before guarded publication and read-back.

Carried work stays unpublished until the successor completes. Its candidate base is distinct from the actual captured repository source used by publication guards. A predecessor's unpublished edits must not silently become the presumed contents of the live checkout. Unrelated repository changes are part of current acquisition and conflict handling, not permission to overwrite them.

A successor also retains the assignment's consumed effort and remaining authorized limits. Starting another run must not silently reset a finite mandate or create an unlimited retry loop. Exact budgeting and record encodings remain design work.

### A simple rule for carrying work forward

The [follow-up on reuse](../sources/working-day-discussion-20261016.md#choose-what-to-carry-forward) separates technical admission from semantic assessment: retain traceable work as starting material, then have the responsible Manager and independent reviewer assess it through the normal workflow. No separate mechanism needs to prove semantic independence before useful work can be retained.

The Host checks that the exact source and candidate base are known, no uncontrolled writer remains, artifacts and registry agree, ownership is valid, and composition neither overwrites newer changes nor hides conflicts. Incomplete work can satisfy these conditions; its open findings and missing evidence remain attached. Conflicts return to the accountable Manager, with both versions preserved. The Host does not choose their business meaning.

The Manager receives the retained candidate, current model, model difference and decision, and open findings. It records one of four judgments for the relevant work in its existing assessment, with a short reason and references to the affected Definitions and artifacts:

| Judgment | Meaning |
| --- | --- |
| Keep (Beibehalten) | The existing realization still meets the current obligations; explain why it can remain unchanged. |
| Adapt (Anpassen) | Useful parts remain; identify the changes needed for the current obligations. |
| Replace (Ersetzen) | The old solution relies on a discarded assumption or is no longer a useful basis; explain what must replace it. Preserve its historical candidate. |
| Unresolved (Ungeklärt) | Information or a decision is missing; state what is needed and which work cannot complete. |

These are judgments about retained work, not new canonical Kinds or replacements for the Definition-assessment census. They can be recorded in existing reports. A judgment can change when review or integration reveals new evidence.

A fresh independent reviewer assesses the judgment against the current model, actual candidate and relevant connected obligations. Keep is a positive, reviewable claim, not a shortcut to acceptance. Previous findings help the assessment; a previous pass does not discharge it. A required unresolved judgment prevents completion of the affected scope and aggregate.

Initially, perform all required reviews and executable checks for the new run, including integration and final verification. Each result binds to the current subject and inputs. Historical evidence remains available but is not relabeled as current approval. An unchanged file can still have changed obligations or assumptions through another area.

For the library's five-to-three change, the following are illustrative judgments, not preapproved outcomes:

| Retained work | Likely judgment | Current assessment |
| --- | --- | --- |
| Catalog and copy identities | Keep | Confirm current requirements and the interfaces used by lending still fit. |
| Borrowing admission | Adapt | Check the new limit, preserved existing loans and refusal without side effects. |
| Return behavior | Keep | Confirm returns remain allowed above the new limit and for suspended members. |
| Documentation stating the old limit | Adapt | Align the description with the current model. |
| Composed application | Integrate the area results | Verify connected behavior and run the required checks on the current aggregate. |

This policy saves reconstruction and implementation effort while accepting renewed assessment cost. Consider skipping specific checks only after real runs expose expensive redundancy and a justified reuse policy exists. The initial rule is: reuse the work, justify its current suitability, and establish approval anew.

### Library illustration

Suppose a future draft says both that existing loans remain intact and that no member may have more than three open loans. Lending identifies the contradiction and asks whether a member with four existing loans may keep them while further borrowing is refused. This is an intentionally inconsistent hypothetical draft; the authored library model already resolves this case and should not generate that question.

The owner confirms that existing loans remain intact and authorizes correction of the conflicting model passage. The assignment remains “lower the admission limit.” Apply R1 retains its fixed model and recorded work; R2 binds the corrected model. Lending starts from its existing candidate and adjusts the conflicting behavior. Catalog assesses its retained realization and may return a reviewed no-change result. Earlier passes are not copied into R2 as current approval.

The owner sees:

> Decision recorded and model updated. Existing work is retained. Lending is adjusting the affected behavior; the other areas are assessing their current work against the new basis. No further answer is required.

This first approach saves reconstruction and rewriting while keeping broad assessment. Automatically skipping supposedly unaffected reviews would require a later, justified invalidation policy. An unchanged file alone does not show that its obligations or assumptions stayed unchanged.

## Ordinary review stays inside the organization

Lending implements the change. An independent reviewer examines the actual candidate against its obligations. If a new eligibility check accidentally blocks returns for suspended members, the finding goes back to Lending. Correction, renewed checks and independent review follow within the configured finite limits.

The owner can see “review found a return regression; correction in progress,” but need not answer. Parents coordinate cross-area dependencies and integration. They cannot conceal an incomplete child result in a positive summary.

Interruption is warranted when a decision exceeds authority, a required model ambiguity remains, work cannot converge, or a limit prevents further progress. The request should explain what would unblock the work. An ordinary failed attempt that the organization can resolve within its mandate is visible history rather than a demand for owner attention.

### Detect stalled work and bound internal correction

The [stagnation discussion](../sources/working-day-discussion-20261016.md#recognize-stagnation-without-endless-retries) proposes combining hard limits with a short progress assessment. The Apply concept already provides finite budgets and proposes three review attempts per work or integration task. The additional question is whether another attempt has a concrete reason to succeed or produce useful evidence. This proposal does not add another retry allowance outside those limits.

After an unsuccessful attempt, the Manager records three points in its existing report:

1. What remains unresolved?
2. What was demonstrably improved or learned since the previous attempt?
3. What will the next attempt do differently, and what observable result should it produce?

Progress can mean resolving a finding, reproducing a cause or ruling out a hypothesis. A new prompt, more changed files or a promise to try again does not establish progress by itself. Diagnostic findings must be relevant to resolving the task, not a justification for indefinite exploration.

| Signal | Possible explanation |
| --- | --- |
| The same finding survives repeated corrections | The cause or affected execution path is not understood. |
| Fixing A reintroduces B, then fixing B reintroduces A | Local solutions conflict or shared assumptions are wrong. |
| Managers wait on each other | A dependency cycle or unresolved interface agreement prevents progress. |
| An execution deadline passes without a usable outcome | Execution is stuck, failed or uncertain; inspect its actual state. |
| Further attempts repeat the same assumption without new evidence | The approach may be cycling rather than progressing. |

The consolidated first-Apply trigger is the second rejected candidate submission, before a third submission under the default three-total-round limit. This replaces the earlier tentative “second correction of the same finding,” which could leave no round for the course correction. It requests parent planning, not an extra attempt or automatic owner interruption. Do not wait for it when a hard limit, unknown writer or missing authority already requires stopping. The Host observes counts; Managers/reviewers judge causes and useful changes of approach. No semantic scoring subsystem is required.

The parent receives a compact account and may choose a bounded course correction within its mandate: retrieve specific missing context, isolate a small reproduction, settle an existing interface contract with the affected Managers, or test a different solution hypothesis. For execution uncertainty, first inspect the known session and side effects. Do not dispatch a duplicate writer merely because a response timed out.

Another attempt requires a concrete change of approach and a checkable target, within the remaining task and assignment limits. Parent investigation and alternative attempts also consume budget. New task IDs, Agent changes or successor Apply runs must not reset consumed assignment effort. Exhausted limits cannot be bypassed by relabeling the same work. If no justified attempt fits the remaining mandate, stop the affected work and retain its candidates and evidence.

Three outcomes remain distinct:

| Outcome | Treatment |
| --- | --- |
| Internally resolvable | Continue with the identified, authorized next step within remaining limits. Report progress without requesting an owner decision. |
| Decision required | Ask for the specific missing business decision, authority or necessary budget extension, with consequences and a proposed next step. No extension is assumed. |
| Not successfully completed | End the affected work in a controlled way when no viable route is found within the limits. Report the result honestly; do not invent a business question or mark the assignment complete. |

Independent ready work can continue only where it remains useful and authorized. Unresolved required work still prevents aggregate success. Controlled termination records whether execution actually stopped; uncertain side effects remain explicit.

For example, Lending fixes borrowing by suspended members but accidentally blocks their returns. The next correction restores returns and reopens the borrowing defect. The parent requests a reproduction covering both behaviors against the already explicit rules. This needs no new owner decision. If the model genuinely lacked a required rule, that specific gap would instead be escalated.

A compact status might read:

> Lending is not making progress. Two corrections have reproduced the same conflict. The parent is checking both operation sequences together within the remaining budget. No owner decision is currently needed.

If the internal correction also fails, the next report states the attempted approaches, retained result, unresolved issue and whether any concrete owner decision can help. Technical failure is not automatically a question the owner can answer. The aim is to justify further work and stop when justification or authority is missing, rather than consume the entire budget blindly or escalate the first ordinary failure.

## Share an assignment budget and protect completion work

The [budget discussion](../sources/working-day-discussion-20261016.md#budget-the-assignment-and-its-managers) proposes one shared assignment budget with bounded allocations and protected review, integration and completion capacity. Delegating work does not create more budget. The root plans allocation; the Host enforces the authorized total and records consumption. Successor Apply runs retain the same assignment's consumed effort and remaining limits.

### Illustrative allocation

For a small library assignment, an experimental plan could divide 60 bounded Agent steps as follows. These figures are neither measured requirements nor accepted defaults. Before dispatch, assess whether the actual Manager tree, required reviews and checks can reasonably fit the plan; a percentage split alone does not establish feasibility.

| Purpose | Agent steps |
| --- | ---: |
| Investigation, implementation and coordination | 24 |
| Independent reviews, integration and final verification | 18 |
| Corrections including their renewed assessment | 12 |
| Unallocated reserve | 6 |

An Agent step is a bounded execution with supplied work and an observed result. Continuing an existing session also consumes steps; a long-lived session cannot bypass accounting. Steps are a planning unit, not equal units of money or work. The operation also needs per-execution deadlines, a total active-execution limit, bounded review attempts and, where sufficiently observable and enforceable, a cost ceiling. Checks and other tools need execution limits too.

Waiting for an owner answer does not itself consume Agent steps; an external delivery deadline continues independently. Repeated polling or additional Agent investigation remains actual work. Report unknown cost as unknown rather than claiming a monetary guarantee from step counts.

### Allocate by work and release unused capacity

Plan by expected effort and required evidence rather than equal amounts per Manager. An area needing only assessment differs from an implementation change with integration obligations. Give each Manager enough for a useful bounded segment, then reassess progress before further allocation.

For example, Lending could receive up to six implementation steps, with independent review capacity reserved separately. Any delegation to Reminders draws from an explicit allocation within the same total; it does not multiply Lending's allowance. Count child consumption once, not again as the parent's own consumption. Unused capacity returns to the shared pool through the recorded allocation process.

### Protect review and integration

Implementers cannot spend protected review capacity on additional implementation. The root may revise reservations only when completed work releases capacity or a justified plan still covers every remaining required review, integration and final check. Reservations must reflect the current obligations, including changes introduced during the assignment.

Before authorizing substantial extra work, ask whether the remaining budget can plausibly cover both that work and its assessment, integration and completion. If it cannot, choose a simpler realization within the existing requirements, retain an incomplete result, or request a specific extension. Budget pressure never authorizes dropping required review or weakening business rules.

Treat a correction as a complete package: targeted repair, independent reassessment and any affected integration checks. A justified change of approach from the stagnation assessment is still required. Reserve use does not reset finite review-round limits or authorize endless attempts under new task IDs.

### Reallocate internally and escalate extensions

The root can move genuinely freed capacity from Catalog to Lending within the existing mandate, provided total limits and required completion capacity remain covered. This needs a recorded reason, not another owner approval. An increase beyond the authorized mandate needs an explicit decision.

An illustrative request is:

> Budget decision needed: the borrowing change is implemented, but one integration defect remains. We propose up to four additional steps for the isolated correction and renewed checks. There is insufficient capacity within the current budget. Without an extension, this run remains incomplete, with retained work and no publication.

The request states what the extra capacity is for, why the proposed attempt is justified, what remains uncertain and what happens without approval. No extension is assumed. Existing execution uncertainty must still be reconciled before another writer starts.

A compact budget report shows consumed capacity, already allocated but unconsumed capacity, protected capacity awaiting allocation, and freely available capacity. These are non-overlapping amounts in the same ledger; review allocations retain their protected purpose without being counted twice. Reservations are not incurred cost. Parent summaries must likewise distinguish own consumption from descendant consumption.

One shared budget, protected completion capacity and bounded allocations reduce the risk of spending everything on implementation. They do not guarantee success or a precise completion estimate. The [execution contract](../../design/canonical-model/manager-apply-execution.md) now owns the proposed ledger/admission encoding, including ordinary versus protected work and active time versus an optional delivery deadline. Implementation remains future work; these are not new canonical business rules. A run can finish Incomplete while its open assignment awaits an extension; continuation preserves that historical result.

## Result and further steering

A completion report states what changed, the model and exact candidate assessed, required review/check results, remaining limitations and publication outcome. For example:

> Applied and read back: additional borrowing is refused when three loans are already open. Existing loans remain intact and returnable. Boundary cases and connected behavior were checked. A return regression found during review was corrected. No required findings remain open in the assessed scope. These artifacts have not been deployed.

Applied artifacts, a merged branch, a release and an observed production deployment are distinct outcomes. The first Apply does not implicitly merge or deploy. A later delivery workflow would need explicit responsibilities and existing authority for those actions.

The owner should eventually be able to stop work, change priority or submit revised intent. Each action needs an observable acknowledgement and result. A request to stop is not proof that a provider process has stopped. Revised intent cannot silently replace a running operation's model binding; the safe initial behavior is to retain the old run and start a new bound attempt with fresh assessment.

| Information | Expected owner involvement |
| --- | --- |
| Progress, normal findings and bounded correction | Available to inspect; no response required. |
| Completed outcomes, costs where known and remaining limits | Concise completion or requested periodic report. |
| Required decision, exhausted mandate or unresolved execution state | Targeted interruption with the needed decision and affected scope. |

The product should allow the owner to be available for judgment without requiring continuous availability for routine coordination. Wall-clock examples in the discussion were illustrative, not a completion-time promise.

## A compact organization status report

The [status-report discussion](../sources/working-day-discussion-20261016.md#see-progress-without-reading-agent-history) proposes one screen with four stable blocks: overall state, required decisions, progress, and problems or limits. Link to changes, evidence, the task tree and history for inspection. The owner should quickly see whether action is needed, what changed, what blocks progress and what happens next.

The heading identifies the assignment, model revision, Apply run and observation time. Reports are projections of recorded work, not a second manually maintained status system. They expose missing or stale observations rather than presenting them as current progress.

### Illustrative intermediate report

The example below uses an intentionally inconsistent future draft. The authored library model already specifies that lowering the limit preserves existing loans. Times, counts and outcomes are illustrative, not execution evidence.

**Library: lower the admission limit to three**

**In progress: one owner decision needed**

Model M2; Apply R2; updated at 10:40.

The existing realization is being adapted. Catalog and membership have local assessment results. Lending awaits a business decision. No artifacts have been applied or deployed.

**Required decision**

Should existing loans above the new limit remain intact? Two draft passages conflict. Recommendation: preserve existing loans and restrict additional borrowing. The decision entry links to the exact proposed model edit and response controls.

This blocks Lending and completion of the assignment. Independent checks can continue. No external deadline is known.

**Progress**

| Area | State | Result or next step |
| --- | --- | --- |
| Catalog | Locally verified, unchanged | Retained realization fits the current assessed obligations; aggregate integration remains pending. |
| Members | Locally verified | Suspension behavior remains intact; aggregate integration remains pending. |
| Lending | Waiting for decision | Candidate work exists; the model conflict remains open. |
| Reminders | Assessment in progress | Check interaction with existing loans. |
| Aggregate | Not yet verified | Integrate after the required area deliveries are complete. |

**Problems and limits**

- Resolved locally: review found an unintended restriction on returns. The correction was checked locally; aggregate integration is still pending.
- Open: the model conflict identified in the decision above. This is the same question, not another request.
- Budget: 18 of 60 authorized Agent steps used. No reliable completion estimate is available.

**Next:** after the decision, establish the model basis and continue from retained work under the continuation rules above.

The report offers drill-down access to actual changes, check/review results, the task tree and execution history. It does not require the owner to read those details to understand the current state.

### Keep verification and delivery states distinct

| Displayed state | Evidence required and scope |
| --- | --- |
| Locally verified (lokal geprüft) | Required local assessment, independent review and checks support the exact area candidate and current inputs. This does not establish subtree integration or whole-system verification. |
| Aggregate verified (insgesamt geprüft) | Required deliveries, integration reviews, checks and final verification support the exact aggregate and selected scope, with no unresolved required findings. Publication has not necessarily occurred. |
| Applied (übernommen) | Guarded publication and read-back confirm the verified artifact and registry result. This is not a Git merge, release or deployment. Carrying work into a successor as starting material is not this state. |
| Deployed (ausgerollt) | A separate authorized deployment has an observed outcome bound to the version and target environment. Operational verification and its limits are reported separately; an Apply result cannot establish deployment. |

These are proposed display meanings, not a new state-machine schema or an automatic progression guaranteed to succeed. Source changes can make earlier evidence historical. Partial publication, failed deployment or unknown outcomes must remain explicit rather than being rounded up to a successful state.

### Reporting rules

- Derive status from observed events and matching evidence. An implementer's success claim alone cannot produce “verified.”
- Reference one question identity from decision, area and problem entries. Do not multiply a single blocker into several apparent owner requests.
- Highlight meaningful changes since the previous report. Unchanged non-actionable states do not trigger repeated notifications.
- Report counts with their actual scope, such as “four of five areas locally verified.” Do not translate them into “80 percent complete”; integration may dominate the remaining work.
- Distinguish a corrected local finding from a verified overall result. State who acts next and what is needed to unblock work.
- Show only observed usage and justified deadlines or estimates. Unknown cost, progress or completion time stays unknown.

## Prerequisites against the current concept

Assessment basis: committed design at `bc53b58b2319d91581ea98457acb4a9534d95bfe`, especially the [Apply concept](../../design/canonical-model/manager-apply-concept.md), [Go architecture outline](../../design/canonical-model/manager-apply-go-architecture.md) and [implementation plan](../../design/canonical-model/manager-apply-implementation-plan.md). Architecture work is ongoing. This assessment concerns the written contracts, not a fresh audit of product code or a claim about implementation progress.

The central realization mechanisms are already described: Package responsibility, bounded context, artifact ownership, independent local and integration review, aggregate verification, finite budgets, needs-decision outcomes, durable history, interrupted-run reconciliation and stale-input refusal. They need concrete contracts, implementation and evidence; they should not be presented as missing ideas.

| Area | Already described | Smallest remaining specification for the desired day |
| --- | --- | --- |
| Starting a job and its mandate | Explicit Apply instruction, fixed model/configuration, artifact-only target, path ownership and finite budgets. | State how the owner selects and releases a draft revision, sees the effective mandate and changes it. A compact view of existing configuration can suffice; no new authorization framework is implied. |
| Owner questions and answers | Open questions, needs-decision outcomes, parent routing and separate model changes. | The continuation proposal above connects questions, answers, model changes and successor runs. Its concrete records and transitions still need design. |
| Stopping and changing work | Cancellation/session ports, historical attempts, recovery and stale refusal. | Define user-visible stop acknowledgement and actual stopped/uncertain result. The continuation proposal retains work in a successor run without inheriting approval; automatic review reuse and live reprioritization can wait. |
| Attention and progress | Concise reports, task tree, event timeline, usage and waiting distinctions. | Define which events interrupt, which are grouped into reports, and how unresolved waiting becomes visible without repeated alerts. Deadlines, budget treatment while waiting and the responsible next actor must be explicit. A CLI report or small view is enough initially. |
| Completion beyond artifacts | Guarded publication and read-back; merge/release/deploy expressly excluded from implicit Apply. | Keep the first promise at applied artifacts. Specify later delivery and operational verification separately if the desired mandate includes running software. |

These are gaps in the interaction contract or deliberately later scope, not reasons to replace the Manager architecture. Existing run, task, question and journal records should carry them where possible. A new cockpit, experience database, organization layer or Government mechanism is not required to demonstrate this day.

## What would establish that it works

The planned real M1 Apply and subsequent M2 Apply remain the first evidence. Inspect actual code, behavior, tests, documentation and history, including preserved behavior. Passing generated tests alone is insufficient.

After that base works, exercise a small owner-interaction scenario: a required question appears, independent work remains correctly bounded, an answer is recorded at the right authority, and the resulting run cannot publish against an obsolete model. Add interruption/recovery and an exhausted review budget. The result should make clear what the owner must decide without requiring transcript archaeology.

Across later changes, measure missed obligations, unnecessary questions, owner attention and correction outcomes. These observations determine whether routine supervision can safely decrease. Experience selection and operational doctor checks remain separate extensions, not prerequisites for the initial Apply demonstration.

Return to [product reflections](README.md).
