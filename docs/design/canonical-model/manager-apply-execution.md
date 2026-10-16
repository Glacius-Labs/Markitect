# Budget and lifecycle contract for the first Apply

Date: 16 October 2026. Consolidated proposal requested in the [owner discussion](sources/manager-realization-20261016.md#consolidate-budgets-lifecycle-and-current-developments). This document owns budget accounting, execution admission, run transitions and their status meaning for the first Apply. The [configuration contract](manager-apply-configuration.md) owns the input schema and exact Package overrides; the [journal](manager-apply-journal.md) owns durable encoding, append and recovery storage. These are design contracts, not implemented behavior or completed acceptance evidence.

The [consistency review](../../concepts/reflections/concept-consistency-review-20261016.md) identified the conflicting drafts. The newer [working-day assessment](../../concepts/reflections/working-day-demonstration-20261016.md) adds independent observations, model-owned intent, targeted counterchecks and finding-specific evidence under budget pressure. The rules below incorporate those refinements using existing review/history. Its four scenarios and repeatability proposal remain follow-up evidence after the separate M1/M2 Playground, without more roles, schedulers or library business requirements.

Reflection input inspected through commit `d13732a3ee113e18e8f6cd8570f71c5f4cc0c1ca`, including successor applicability, complete task context and correction of finding accountability through local/integration review. The completed language and M1 remain unchanged. The [delivery plan](../../implementation-plan.md) owns assignments; this document owns the technical contract.

## One owner for each rule

| Question | Governing contract |
| --- | --- |
| Which settings apply to Lending? | Project defaults plus the exact `lending` override, in configuration. An override for Lending does not configure Reminders. |
| What can the assignment afford and admit next? | One assignment ledger and the admission rules below, enforced by `run`. |
| Is a run working, waiting, uncertain or finished? | The run transitions below, reduced from committed Host events. |
| Where are events and exact payloads stored? | The journal; its adapter does not choose execution policy. |
| How are inputs captured, successors handed over and outputs published? | `apply`, using the same run state and current controller binding. |
| What does the owner see? | A projection of these facts and current evidence, without another UI state machine. |

## A shared assignment budget

The assignment grant has four independent caps: prepared runs, possible session starts, Agent steps and active execution time. Project limits bound the grant; task restrictions may narrow it. There are no independent copies of the shared balance per Manager. Root proposes allocations and reallocation; the Host validates and commits them. Reviewer and child consumption is attributed once to the actual execution, with references for parent summaries.

- A run consumes one run unit when its fixed binding is committed as Prepared. A rejected acquisition creates no prepared run and grants no execution.
- A session start reserves one start unit before attempting creation. Resuming an already established session does not consume another start; a replacement session does.
- A step is one Host-dispatched Agent turn, including a continued conversation. Tool callbacks are part of that turn, not fresh Agent steps. Their actual external processes still have bounded execution time and tool/output limits.
- Active execution time is the sum of charged invocation wall durations: session startup, Agent turns, project checks and Agent-requested external tool work. It excludes bounded Host control, idle sessions, parent/owner waiting and time between runs. Distinct overlapping invocations contribute separately; never count descendant work again as parent effort. This is neither CPU time nor an end-to-end delivery clock.

Every charged invocation has a recorded finite maximum duration before dispatch. The Host reserves that maximum, then settles the observed duration and releases unused allowance. If dispatch or termination remains uncertain, its full reservation stays unavailable until reconciliation. An error or elapsed timeout cannot release it. Unknown provider usage remains unknown; these units do not establish a monetary spending ceiling.

Deterministic composition, source acquisition/preflight, journal/control, publication/read-back and emergency reconciliation/termination are Host control operations outside the assignment ledger. Each has finite `hostOperationTimeout` plus operation-specific size/output bounds captured in preparation. They can record failure or stop an effect after Agent capacity is exhausted; they cannot investigate semantics, author a repair or launch a review under the name of cleanup. Record overruns and block work; accounting cannot establish physical termination.

## One ledger with protected completion capacity

For each budget dimension, track four non-overlapping amounts:

| Amount | Meaning |
| --- | --- |
| Consumed | Settled effort, or committed prepared runs. It never decreases. |
| Allocated | Unconsumed capacity assigned to named bounded work, including in-flight/uncertain dispatch reservations. It retains its ordinary-work or protected-completion purpose. |
| Protected | Completion capacity not yet allocated to an execution. |
| Free | Unconsumed, unallocated capacity available for ordinary work or additional protection. |

Within the admitted grant, `consumed + allocated + protected + free = grant`, dimension by dimension. A dispatch reservation is held inside its allocation, not counted once more. Reports show uncertain holds separately within Allocated. Admission preserves that accounting; settlement must still record an actual overrun rather than reject the observation. An overrun sets available capacity to zero and blocks work, with the excess explicitly reported. It is not an invented grant increase or a positive free balance.

The configuration supplies an initial completion reserve. Before the first Agent dispatch, the Host creates an initial plan from the organization, fixed strategy and required checks; bootstrap cannot depend on an unbudgeted root turn. The plan identifies remaining mandatory reviews, integration assessments, final verification and executable checks and their bounded allocations/reserves. Include a specifically commissioned later stage when planning the two-Apply assignment. Root may refine the plan, but the Host cannot admit extra implementation without capacity for its required assessment. Structural coverage of obligations is checked; estimating enough effort remains fallible and reviewable.

For the explicitly commissioned M2 stage, hold its required future run unit in a named stage allocation. It is part of Allocated in the same ledger, not an extra run or review pool. R1 cannot consume it through unrelated retries. This is a small recorded two-stage assignment plan, not a general workflow language.

Protected capacity pays for independent Agent assessment, integration/final review and project checks. Deterministic composition and guarded publication are separately bounded Host control, not another budget dimension. Source authoring, including parent integration code and repair, remains ordinary work. Code-writing capabilities are not granted to completion-funded Agent steps.

Root can move Free to an ordinary allocation, move Protected to a completion allocation, or revise/release unspent capacity with a recorded reason and a still-covered completion plan. Settled consumption and in-flight/uncertain holds cannot be reallocated. Unused ordinary allocations return to Free. Unused completion allocations return to Protected until the plan explicitly releases them. No owner decision is needed for valid internal reallocation within the grant. No step or session can silently draw from the other purpose after exhausting its own allocation.

For illustration, one dimension of a 60-step grant could evolve as follows. This demonstrates accounting only; it is not a claim that 60 steps suffice for the entire library or both Applies.

| Observation | Consumed | Allocated | Protected | Free |
| --- | ---: | ---: | ---: | ---: |
| Initial plan | 0 | 0 | 18 | 42 |
| Allocate six Lending work steps | 0 | 6 | 18 | 36 |
| Four steps settle | 4 | 2 | 18 | 36 |
| Release the unused two | 4 | 0 | 18 | 38 |
| Allocate two review steps from Protected | 4 | 2 | 16 | 38 |
| One review step settles; release its unused step to Protected | 5 | 0 | 17 | 38 |

The earlier 24/18/12/6 split was a planning illustration, not four accounts. A correction now reserves ordinary repair effort and the necessary renewed assessment together in this one ledger. If both do not fit, retain the candidate and report incomplete or request a concrete extension. Required checks and business rules are never reduced to fit the budget.

## Evidence need survives budget pressure

Decide what evidence the current obligation needs separately from whether the assignment can afford it. The existing reviewer selects a small distinguishing case when a named plausible error could violate a consequential boundary, change existing state incorrectly or escape ordinary checks. Start with the canonical basis before reading the implementer's solution; the fresh review can use an initial model-only turn followed by candidate inspection in the same session. Record the actual order and include both turns in the completion plan. This is targeted review work, not another reviewer or a mandatory mutation of every Definition.

Reuse a current adequate case when it already distinguishes the concern. For Lending, useful examples are suspension preserving return rights and repeating return L1 after the copy has been lent as L2. Additional checker counterexamples are selective; all mandatory assessment still applies. Neither low remaining budget nor simple-looking code establishes that a recognized concern is unnecessary.

A necessary missing check is an ordinary blocking `Finding`, with stable identity, model/input/scope basis, requested evidence and originating reviewer. It can precede a candidate; do not invent a candidate reference then. Later independent review explicitly resolves it for the current subject using actual evidence, or supplies a grounded applicability decision referring to the authorized basis. A Manager may propose resolution but cannot remove it. Budget/deadline exhaustion cannot resolve it, and omission from the latest report cannot erase it.

Host acceptance reads findings and resolutions from the relevant task/child/predecessor history. It validates authorized reviewer provenance, actual receipt existence and current input/subject/scope/purpose bindings. A child receipt retains its own candidate plus composition provenance; it does not acquire the aggregate's hash or replace root integration evidence. A generic pass, invented receipt, failed/unknown required check or historical resolution on a changed subject cannot close a current gap. When a specific case is required, the evidence must establish that it ran rather than being skipped; command exit zero alone is insufficient. The reviewer judges the case's semantic adequacy.

Use existing finding/review/check records and their projection, not a verification backlog or editable resolved flag. One receipt can support several findings. A genuine reviewed applicability change is recorded with grounds and preserves the original finding; a scope/model change still needs its authorized transition and fresh required assessment. Initial full reconciliation does not permit narrowing the operation merely to claim success. If evidence cannot be obtained, the run remains incomplete. The Host can enforce declared completeness and bindings, but cannot prove that reviewers never omit or misinterpret a necessary concern.

At a successor, carry unresolved finding identities even when their candidate files are discarded. The current independent review records one applicability judgment: still applies, applies with adjustment, no longer applies or unclear. This is an assessment on the new basis, not another lifecycle. For a limit finding, M2 replaces the six-loan countercase with three successful loans and refusal of the fourth; it does not preserve five as binding intent. A valid no-longer-applies judgment stops blocking the current scope without retroactively passing the old candidate. An unclear judgment remains open. If responsibility disappeared, route assessment through the current parent/root. A materially different replacement finding must be linked and remain required; a replaced label cannot close both concerns. Full reconciliation still assesses new current obligations and transfers no old approval.

Build the current finding view from typed judgments and causal links at a pinned journal revision. For the first Apply, supply the compact complete open/unclear/reassessment set for the task's full responsibility plus recorded dependencies and inherited concerns. Do not minimize it from a Manager's free-text next action. Parent integration receives child blockers/delivery references; unknown routing returns to the common parent/root. The Host carries recorded scope/Definition/provenance relationships, without claiming to discover missing semantic dependencies.

Role context carries current grounds and evidence references, not every old conversation or recursively rewritten summary. The Host retains the complete required set even when delivery is paged; a token cutoff, unread page or fetched-but-unresolved entry cannot discharge an obligation. Required context must actually be available before its dependent action. Record supplied references/revision and refresh changed relevant facts before acceptance. This uses the existing journal projection and context builder, with no second finding database or summarizing Agent.

Rule location, artifact ownership and affected responsibility are distinct. A receiving Manager exposes a mismatch through the common parent; “not mine” cannot close or orphan the finding. The parent records corrected accountability and ordinary child contributions. Cross-area concerns require composed integration evidence, not every neighbor's co-signature or a local pass alone. The Host checks recorded scope/ownership; local reasoning and integration review remain responsible for discovering missing semantic impact.

## Admission, amendments and finite correction

Before any external execution, the Host validates the current run/controller, allowed action and source/candidate binding, allocation purpose, remaining counters, timeout and deadline. The selected maximum duration must fit its remaining allocation and any deadline; reduce it explicitly before preparation or refuse admission. Reservation and dispatch preparation commit together before the call. Recheck time/authority at the actual effect boundary. No call follows a failed or uncertain commit. Settlement is another recorded transition; recovery inspects the known effect instead of replaying the preparation.

Assignment consumption and unresolved reservations survive restart and successors. An allocation is not a fresh grant. An owner amendment can increase the grant within project ceilings, with its source and exact changed values recorded. Changing the effective grant, deadline, configuration or profile requires a successor binding. Internal reallocation does not. Raising a project ceiling requires an authorized configuration edit as well. A new task ID, Manager session or same-goal run cannot bypass an exhausted limit.

`reviewRounds` means total submitted candidate rounds per logical responsibility and review purpose (local, integration or final) within one fixed run; the default is three, including the initial candidate. It does not mean three corrections after an initial attempt. Recreating a task/session preserves that counter. An idempotent repeat of the same submission is not a new round; submitting another candidate for rework is. Retrying a known transport failure against the same submission consumes actual execution allowance, without inventing another semantic candidate or blindly replaying an uncertain call.

For full reconciliation, the counter key is `(run, Manager, review purpose)`, not a replaceable task/session ID. A submitted candidate that fails a required check still consumes its round. A new submission ID with unchanged failing bytes is another round; an exact duplicate ingress returns the existing receipt.

After each rejected candidate, record the unresolved finding, actual progress/evidence and the different next action with its observable target. After the second rejection, the parent must assess that plan before a third submission. At root, root performs the same bounded planning step. This count-based trigger replaces the ambiguous “second correction of the same finding”; the Host need not infer whether differently worded findings share a cause. It grants no extra round. Missing authority or uncertain effects stop dependent work immediately, without waiting for a threshold.

A real successor requires a changed basis or explicit renewed authority, not merely the wish to reset review attempts. Its fresh assessments have current-run counters but use the same remaining assignment budget and carry the unresolved findings. First-version successors repeat all required reviews/checks; previous approvals remain historical.

## Execution allowance and delivery deadline

Use `maxExecutionTime` for the active-execution allowance above. Remove the ambiguous `assignmentTimeout` field. The owner can separately supply an optional absolute UTC `deadline` on the assignment. It continues while waiting and is never silently extended. Step/check/startup timeouts remain local limits; reserve their actual selected maximum before invoking them. A maximum such as ten minutes is a ceiling, not permission to spend that much without an allocation.

When all work is quiescent and waiting for the owner, no Agent step or active execution time is consumed. Status inspection alone starts no Agent. Further analysis, polling through an Agent or running checks is charged work. One waiting Lending task does not stop accounting for useful Catalog work that is still executing.

For example, the organization asks a question at 16:00 after consuming 18 steps and twelve minutes of execution time. An answer at 09:00 the next day leaves those consumed amounts unchanged if nothing executed overnight. With no delivery deadline, it may continue within the existing grant after resolving the question. With a deadline at 18:00 the previous day, the next controller observes expiry and finishes the quiescent run Incomplete before any new work; an authorized extension and successor are required. A context-only answer would otherwise continue the same fixed run; a changed model still needs a successor.

No background timer service is required. While the controller is absent, read-only status can show the last recorded Waiting state plus an expired deadline; only a committed control transition records Finished. Admission/recovery checks time before effects. A later amendment never retroactively authorizes expired or overrun work.

A local timeout requests stop and records the observation. Confirmed termination can permit a justified attempt within remaining limits. Unknown termination enters reconciliation and blocks new work. On assignment exhaustion/deadline, stop new work and perform bounded control cleanup. Completion reserves do not bypass an expired deadline, exhausted global cap or unresolved writer.

The Host supplies explicit clock/elapsed observations to transition logic; the reducer does not read an ambient clock. Within an observed execution use elapsed duration, not subtraction of arbitrary report timestamps. Recovery cannot erase an unobserved execution interval: retain the hold until a justified settlement exists. These execution clocks are unrelated to the library's explicitly supplied business dates.

## One run state machine

Only `run` owns these persisted lifecycle meanings. `apply` requests validated transitions for verification/publication/handoff. The journal stores their committed facts; adapters and status views do not run alternative state machines.

| State | Meaning and permitted next transition |
| --- | --- |
| Prepared | Fixed inputs and a run unit are committed; no Agent work is yet implied. Start authorized work to Running, or finish safely without dispatch. |
| Running | Ready work, checks, local review or upward integration are being coordinated. Some tasks may wait while other work continues. Move to Waiting only when all remaining useful work depends on external input and no effect remains active/uncertain; move to Verifying when required deliveries are accepted. |
| Waiting | Quiescent, with a named next actor and reason: owner decision, authorized model edit or other explicit external prerequisite. Resume Running on an applicable context-only answer; use safe successor handoff when the basis changes. A deadline still applies. |
| Verifying | Required aggregate checks, independent final verification and source preflight. A known correctable finding can return to Running only with a valid correction/assessment allocation and remaining rounds. Exact current passing evidence permits Publishing. |
| Publishing | The guarded candidate/registry effect and read-back are being observed. No new implementation or competing publication is dispatched. Complete to Finished, or enter Reconciling for partial/unknown observations. |
| Reconciling | Stop, interruption, controller loss, partial/unknown publication or uncertain execution requires inspection. Block new work/publication. Use bounded control inspection/stop only. Once effects and authority are established, resume the eligible phase on the same basis or finish with the observed result. |
| Finished | Exactly one immutable outcome below. No further dispatch or publication by this run. Later runs link to it without rewriting its result. |

`Waiting` is not permission to leave a background writer unaccounted for. `Reconciling` is an actionable blocked condition, not a successful terminal outcome. A bounded reconciliation that cannot establish the effect returns that visible blocker; it does not enter a free polling loop. If the process/store is unavailable, inspection labels the last committed state and its observation limit; it cannot invent a completed transition. Same-run recovery requires a still-valid basis, known prior effects and a current controller. New input bindings require a successor. Applied/NoChange can finish only through the publication/read-back boundary, including its established recovery result.

An internal dependency cycle is not external Waiting. The existing parent/root performs a bounded coordination/progress assessment within its mandate, or ends incomplete when no viable authorized step fits. Do not wait indefinitely for Managers to unblock each other or invent an owner decision to label a technical deadlock.

Stop requests, input drift and exhaustion first stop admission and reconcile any active effects. Once quiescent, they can finish. A proven partial publication is retained as actual source for a later operation; the old request is not silently repeated. Complete recovery may establish Applied only when the original required evidence and exact publication/read-back are actually established. A cancellation acknowledgement, process exit or Agent report alone cannot do that.

Cleanup may record an already completed authorized effect after a deadline; it cannot launch missing verification or resume writing under expired authority. If on-time required completion is not established, finish Incomplete with the actual publication facts rather than assume success. An already Finished result remains historical when its old deadline later passes. A grant overrun is likewise retained as a violation, not converted to a successful bounded run merely because the artifact is correct.

## Terminal outcomes and displayed status

| Outcome on Finished | Grounds and next action |
| --- | --- |
| Applied | Current required local/integration/final evidence, guards and read-back establish the exact changed artifact/registry result. |
| NoChange | The same required assessment and current source/read-back establish that neither artifacts nor registry need publication. No edits alone is insufficient. |
| Superseded | An unfinished predecessor is safely replaced by an authorized successor in one committed handoff. No unresolved external effect may be handed off. |
| Stale | Bound inputs drifted; the run is quiescent and cannot publish. Acquire a current successor basis if authorized. |
| Stopped | An explicit owner stop has reached known quiescence. Retain work and evidence. |
| Incomplete | Budget, deadline, review-round exhaustion or absence of a viable authorized correction prevents completing the scope. Record the reason, retained work and any actionable question. |
| Failed | A definitive acquisition/execution/validation/publication failure ends this run. Preserve actual effects and grounds; a known partial publication is a failed result, not an implicit rollback. |

An acquisition rejected before Prepared is an application preparation failure with its available evidence, not a fabricated successful or prepared run. Unknown effects never become Failed merely to remove a blocker: they remain Reconciling until established. Each terminal result records the source/subject and reason; later discovery adds a correction/finding instead of erasing history.

“Needs decision” is a question/attention projection, not a second run outcome. It can appear while the run is Running because other tasks remain useful, or alongside Waiting. “Interrupted/uncertain” projects Reconciling and the actual observation gaps. “Locally checked” and “aggregate verified” describe evidence for exact subjects, not new lifecycle states. Publishing is not Applied. Deployed requires a separate authorized operation outside Apply.

An assignment has an open/closed fact, its commissioned remaining goal/stages, linked runs and amendments. While open, activity is projected as active, waiting, blocked or idle from those facts. Closure requires established quiescence/effects. It closes with Completed only when all commissioned work has successful bound results and no required unresolved obligation. It may instead be explicitly closed Stopped, Incomplete or Failed, retaining unanswered questions/findings. Closed assignments are not silently reopened.

Thus an exhausted run can be Finished/Incomplete while the open assignment waits for a grant decision. An authorized amendment starts a successor without changing that old outcome. If the owner closes the assignment, report that separate fact. A completed R1 remains Applied while the two-stage assignment awaits the M2 edit; R2 does not change R1 to Superseded. The same preservation rule applies to every already Finished outcome.

## Library traces and focused implementation gates

| Library situation | Budget and lifecycle result |
| --- | --- |
| M1 initial realization | Reserve completion capacity, run root/Package work through fresh reviews, close required findings with current evidence, verify and publish/read back A1: R1 Finished/Applied. The commissioned M2 stage remains pending. |
| Model-only five-to-three change | Capture M2 and actual A1, prepare R2 under the same remaining grant, replan completion capacity and repeat current assessment. R1 remains Applied. A2 can finish Applied only after its own gates. |
| An unfinished predecessor had a loan-limit finding | Retain its identity and assess the changed requirement on M2. Obtain current three/four-loan evidence; preserve the old failed result without requiring obsolete five/six-loan behavior in A2. |
| Lending asks; Catalog can continue | Lending waits; the run remains Running. Actual Catalog work is charged. No overall Waiting label hides continuing execution. |
| All work awaits an owner answer | Finish/quiesce effects, then Waiting with the exact question and next actor. No step/time consumption solely from the wait. |
| Review rejects suspension/return behavior twice | Parent examines a concrete repair plan before candidate three. Repair and renewed assessment must both fit. A third rejection yields Incomplete/review limit; renaming the task grants no fourth candidate. |
| Only protected assessment capacity remains | Do not spend it fixing borrowing. Preserve the candidate and finish Incomplete if required authoring cannot proceed; ask for an extension only when there is a concrete justified plan. |
| Required L1/L2 case is unaffordable | Preserve its finding and retained candidate. Ordinary-return success does not discharge it; omit no requirement and publish no unsupported result. |
| Step times out; writer state unknown | Reconciling, reservation held, no replacement writer or publication. Confirm termination/effects before a permitted resume or successor. |
| No-change reconciliation | Full current assessment plus source/read-back, then Finished/NoChange, with no inferred approval reuse. |
| Source drift or partial publish | Stop admission and reconcile. Known drift can finish Stale; known partial publication finishes Failed with exact paths. Unknown effects remain blocked. |

Implement focused deterministic transition/accounting checks for these cases, duplicate reservation/settlement, refusal of wrong-purpose spending, unused-capacity release, no inherited Package override, active-time versus overnight waiting, deadline expiry, round counters surviving task recreation, terminal-result immutability and same-goal successor consumption. Finding checks reject disappearance by omission, old/invented receipts and a generic pass without finding-specific resolution; replay preserves valid current resolution. Check real adapter termination separately; fake events cannot establish physical quiescence.

The external working-day evaluator can compare captured invocations and observations against this contract without importing the product's reducer/accounting implementation. Its controlled negative evidence includes a reset balance, implementation funded from protection, an old approval on new inputs, and a second writer admitted before reconciliation. None is added as a secret library business rule or an extra primary M1/M2 requirement.

Return to [configuration](manager-apply-configuration.md), [journal](manager-apply-journal.md), the [Playground](manager-apply-playground.md) or the [Apply plan](manager-apply-implementation-plan.md).
