# Budget and lifecycle contract for the first Apply

This contract owns budget accounting, execution admission, findings that block completion, run transitions and their displayed meaning. The [configuration](manager-apply-configuration.md) owns the input fields and Package overrides; the [journal](manager-apply-journal.md) owns durable records and storage. The `run` package implements these rules.

[DEC-043](../../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) extends this target to named review teams, bounded arbitration and complete product defaults. Existing source and historical single-reviewer policies retain their actual supported version; this document change does not migrate them.

## Assignments and runs

An assignment is one owner request, such as "Apply the current library model". An Apply invocation either starts a new assignment or continues an open one by naming it. An assignment holds the budget and links its runs. A run binds one fixed set of inputs: model, configuration, repository source and profiles. Ordinary rework stays inside the run. A further run in the same assignment is always a **continuation**: a run finished without completing the assignment (any outcome except Applied and NoChange), and the owner continues the assignment with a recorded extension, a changed basis or a working tree cleaned up after a partial publication. The finished run keeps its outcome; the new run links to it. When an owner answer requires a model edit, the run first quiesces and finishes Incomplete with the reason "awaiting model change" and its retained work; after the owner has committed the edit, a continuation starts. There is no handoff between unfinished runs.

A repository has at most one unfinished run. A new Apply on a repository whose run is still unfinished is refused until that run finishes or is stopped. An interrupted run is **resumed**, not continued: resuming reopens the same run, reconciles its effects first and keeps its basis.

Here, the **run input basis** is the immutable captured model, configuration, source and profiles. Each step's workspace starts from the accepted aggregate current at its admission ([D3](manager-apply-core.md#deliveries-and-composition)); a newer aggregate is not a new Apply or a continuation run. A new conversation or execution session changes neither the run input basis nor the assignment's allowance.

The two-Apply demonstration therefore uses two ordinary invocations: R1 on M1 and, after the model edit, R2 on M2 against the committed A1. Each is its own assignment, and both are reported. No budget carries over between them, and no approval does either.

## A shared assignment budget

The assignment has four caps: prepared runs, session starts, Agent steps and active execution time. The configuration supplies their defaults and hard project ceilings. There are no per-Manager copies of the balance, and Managers do not manage budgets; the Orchestrator accounts for every invocation once, against the assignment.

- A run consumes one run unit when its fixed binding is committed as Prepared. A rejected acquisition prepares no run.
- Starting a new execution session consumes one start unit, including a replacement container that resumes a known conversation. In the first Apply every step starts its own container and therefore consumes a start unit ([S6](manager-apply-core.md#steps)). Conversation continuity grants no free execution or new allowance.
- A step is one Orchestrator-dispatched Agent turn, including a continued conversation. Tool callbacks belong to their turn.
- Active execution time is the sum of charged invocation durations: session startup, Agent turns, project checks and Agent-requested tool processes. It excludes Orchestrator control work, idle sessions and waiting for the owner or for other tasks. It is neither CPU time nor a delivery clock.

Every charged invocation has a finite maximum duration recorded before dispatch. The Orchestrator reserves that maximum, then settles the observed duration and releases the rest. If dispatch or termination is uncertain, the full reservation stays held until reconciliation; an error or timeout cannot release it. Unknown provider usage stays unknown. These units do not establish a monetary ceiling. The Orchestrator supplies clock and elapsed-time observations to the transition logic, which reads no ambient clock; within an invocation the observed elapsed duration counts, never a difference of report timestamps.

Composition, source acquisition and preflight, journal control, publication with read-back and emergency termination are Orchestrator control operations outside the budget. Each has a finite `hostOperationTimeout` and size limits fixed at preparation. They can record failures and stop effects after Agent capacity is exhausted; they cannot investigate, repair or review.

## One ledger with protected completion capacity

The configuration sets a completion reserve for session starts, steps and execution time. The rule is one admission check:

- **Ordinary Manager work** (Explore, Plan, coordination, implementation, rework and parent integration authoring) is admitted only if the capacity left after its reservation still covers the completion reserve. Read-only Manager work is not an independent review.
- **Assessment work** (every named local, integration and intake review, conflict arbitration and required checks) may use any remaining capacity, including the reserve.

Role, permitted effects and budget class are separate. In the first Apply steps run one at a time ([S5](manager-apply-core.md#steps)); concurrent read-only Manager work is a later optimization that must preserve the completion reserve. Orchestrator admission and physical workspace/tool enforcement must agree; a read-only label does not establish either. Existing source's `Writing`/`Assessment` coupling needs a deliberate technical migration, not relabeling Explore as review.

Before a run is Prepared, the Orchestrator checks the complete review census. It has one local subject per Manager, including root, empty and intermediate Managers, and one integration subject for each Manager with children. When the captured source has unexplained files, it also has one intake subject ([T2](manager-apply-core.md#manager-tasks)). Each review has the [two stages](manager-apply-concept.md#distinct-review-responsibilities) of expectations before implementation evidence and subsequent candidate assessment. The versioned run policy binds the protected starts and steps; batching or a no-change claim cannot reduce that preparation floor. Insufficient capacity is a preparation failure and consumes no run unit. The floor is structural, not a promise that the reserve will cover every correction or required check. Under the team target, sum both stages for every active reviewer of every subject, including the root intake team when needed. An on-demand arbiter must fit the remaining assessment capacity; preparation does not invent free dispute capacity. The default numeric reserve can be insufficient for a larger team/topology, in which case preparation explains the required floor and the explicit cap/reserve changes needed rather than granting them automatically.

Source reference: at Main `d60ae93dedb39da112f382a284b33f0ca24c6c08`, `run.managerReviewDemand` binds V3 to `N + P` subjects and `2 * (N + P)` starts and steps, where `N` includes all Managers and `P` counts Managers with children. This conservative startup floor does not require each review stage to use a separate session. Older records retain their original versioned demand and bytes. The source floor does not establish a working review engine or sufficient execution-time capacity.

Settled consumption never decreases. Settlement records an actual overrun instead of rejecting it; an overrun sets the remaining capacity to zero, blocks further work and is reported. Waiting consumes nothing: while the run is Waiting, no container runs and no step or execution time is charged ([W1](manager-apply-core.md#while-a-run-waits)). Status inspection starts no Agent.

Budget pressure never waives evidence. When required ordinary Manager work no longer fits, it is not admitted; eligible assessment work may still use the reserve. If no permitted work can complete the outstanding obligations, retain the candidate and finish Incomplete after reconciling effects. The owner can extend the assignment's caps within the project ceilings; the extension is recorded and used by the continuation run. A run that ended Incomplete stays Incomplete.

## Idle execution and resumption

An idle Manager is a logical responsibility without admissible work and without active or uncertain effects ([S2](manager-apply-core.md#steps)). Preserve its task, conversation identity, own unaccepted changes, findings and counters in the existing records. Queuing a message does not itself start a container: ordinary readiness and budget admission select its next step.

The warm limit of the first Apply is zero ([S6](manager-apply-core.md#steps)): every container stops when its step ends. The next step starts a new one, which consumes a session-start unit and startup time even when it resumes the same conversation; otherwise it uses a visibly new conversation with the Orchestrator-built handoff ([S3](manager-apply-core.md#steps)). A later strategy that keeps idle containers warm needs its own recorded finite policy and proof that no autonomous artifact effect happens between steps. No new container or handoff replenishes assignment capacity or clears findings or rejection counts.

Idle waiting has no Agent-step or active-execution-time charge; storage and future startup are still real resource costs. There is no cost-free-resume claim. Stopping a container does not by itself settle outstanding Orchestrator-side or external effects ([S2](manager-apply-core.md#steps)).

## Evidence need survives budget pressure

What evidence an obligation needs is decided separately from whether the budget can afford it. A reviewer who sees a plausible error that could violate a consequential boundary selects a small distinguishing case, starting from the model before reading the implementer's solution. A current case that already distinguishes the concern is reused. For lending, useful cases are a suspended member who can still return a book, and a repeated return of loan L1 after the copy has been lent again as L2.

The [technical risk derivation duty](manager-apply-concept.md#technical-risk-derivation) applies both before and after candidate access. Candidate assessment must investigate failure modes exposed by the implementation or composed boundaries and supplement the original cases when necessary. Preserve the original recorded expectations; use existing findings to request additional cases and the responsible Manager's ordinary rework to supply them. A prior expectation set or a passing suite is not a ceiling on review.

Existing assessments, findings and evidence references carry the reasoning from the affected obligation through the suspected failure to the distinguishing case and actual observation. For a consequential case selected as necessary, the reviewer must cite current evidence that addresses it or explain on the authorized basis why it is not applicable. Missing necessary evidence remains a blocking finding. A hypothesis stays a concern until examined, not a claimed observed defect. Match evidence to the boundary: a fake or sequential test cannot establish a concurrency or database guarantee it did not exercise. If the required observation cannot be obtained within the available authority, environment or budget, retain the unresolved work rather than declare success. This adds no separate ledger or result format.

A concern a reviewer records is a **finding**: an Orchestrator-recorded identity, the affected responsibility, its model and input basis, the requested evidence and the originating reviewer. A necessary missing check is a blocking finding. A finding can exist before any candidate. It has three states:

| State | How it is reached |
| --- | --- |
| Open | Recorded by a reviewer. Omission from a later report does not change it. |
| Resolved | A later independent review cites actual current evidence for the current subject. |
| Not applicable | A later independent review records grounds on the authorized basis, for example a changed model. |

A finding is **blocking** or **advisory**. Only blocking findings reject a submission and prevent acceptance. Advisory findings, such as a suggested simplification, are recorded for the owning Manager and never block.

A Manager may propose a resolution but cannot record one. Budget or rejection-limit exhaustion cannot resolve a finding. A generic pass, an invented receipt, a failed or unknown check, or an old resolution on a changed subject cannot close it. When a specific case is required, its evidence must show that the case ran; an exit code of zero alone is not enough. If a requirement changes in substance, the old finding becomes not applicable and a new, linked finding carries the changed requirement. Open findings carry into a continuation run even when their candidate files are discarded.

Role context carries the complete set of open findings for the task's responsibility and its recorded dependencies, with their grounds and evidence references. Paging the set is allowed; an unread page cannot discharge an obligation. A receiving Manager that is not responsible for a finding reports that through the common parent, which records the corrected accountability; "not mine" cannot orphan it.

### Extract findings at the first Apply

The owner accepted the [post-adoption workflow](manager-apply-library-flow.md#first-apply-after-model-adoption) and reuse of this finding structure ([source](../project-capability-skills-20261009.md#owner-source)). Carry known realization deviations from model adoption into the first Apply through the existing [issue shape](manager-apply-core.md#issues), not a separate repair list, approval ledger or run state machine.

- Preserve the originating report and identity as provenance, affected Definitions, the original model/source/candidate basis, evidence, reason and requested correction or observation. Reuse existing references; do not manufacture a reviewer identity, receipt or runtime observation.
- Bind applicability to the adopted model and the actual captured realization for this Apply. Check the responsible Manager and relevant dependencies. Historical review or model adoption does not approve the current artifacts. A changed source or meaning requires current assessment, not blind reuse of an earlier conclusion.
- Represent still-applicable findings with their current subject, responsible owner, blocking/advisory severity and open state under the existing issue rules. Source notes that are not reviewer findings inform independent assessment; they do not silently become authenticated reviewer verdicts. Only an independent review with current grounds can resolve a finding, mark it not applicable or correct its severity.
- Supply the findings and their evidence to the ordinary Explore and review contexts. They guide investigation without narrowing the complete Manager responsibility to the known list. Unclear applicability or responsibility stays visible until assessed; forwarding a finding cannot orphan it.
- A Manager may propose a repair or resolution. A specific required case needs current evidence that it ran and addressed the concern. A generic pass, old green test, unchanged file or omitted report cannot close the issue. If the underlying realization has already changed, independent review establishes its current effect instead of repairing by rote.

This is an input handoff into the existing process, not a new approval round or automatic replay of an Extract run. Exact persistence, identity mapping and producer bindings remain implementation work; the documented semantics grant no new model writes or execution rights. The adopted model remains read-only during Apply. Any necessary model correction follows the separate owner decision and model-change path.

## Review conflicts and arbitration

Configuration selects the [reviewArbiter](manager-apply-configuration.md#resolve-the-team) for each Manager from project defaults and exact overrides. The Parent coordinates concerns but neither it nor the implementing Manager can close its own blocking findings. The Orchestrator admits the configured arbiter only when a Manager or reviewer has explicitly recorded a disputed blocking finding, its conflicting assessments and grounds. A reviewer preferring another valid design raises an advisory recommendation, not an invented binding requirement.

The arbiter starts a fresh independent read-only review of the exact frozen subject. Supply the relevant model passages, applicable mandate, all disputed findings, both arguments, current evidence and required context through the existing captured access rules. Its decision names each finding and basis. It can confirm a defect, independently resolve or mark an objection not applicable with evidence, correct its severity to advisory with grounds, identify a necessary missing check, or ask the owner to settle unclear/contradictory intent. Missing evidence leaves the relevant blocker open until the named evidence is assessed; silence, a generic pass or exhausted budget cannot close it.

The ruling settles that dispute for that basis without requiring the original reviewer to agree personally. Preserve every original judgment and append the ruling. Reopening the same closed dispute on unchanged inputs needs concrete new evidence. The arbiter does not implement, change intent, waive checks or arbitrate its own ruling. There is no recursive appeal hierarchy and no repeated unchanged-dispute loop. An unresolved question goes to the owner under ordinary question rules; work that cannot finish within the existing limits ends Incomplete with retained findings and candidates.

All configured reviewers must provide complete valid assessments of the bound subject. Acceptance then depends on every blocking finding being settled with current independent grounds and all other required gates, not a vote, a majority or personal unanimity. A valid arbiter ruling can settle a contrary review; it cannot substitute for a missing assessment or a failed check. A failed, invalid or uncertain assessment keeps its existing failed/reconciliation meaning and is not counted as an invented rejection.

For example, a Library reviewer asks to forbid every action by a suspended member, while another preserves returns. The model explicitly permits returns. The Manager cites that passage and supplies the discriminating return check; the arbiter independently assesses its current result and closes the excessive restriction as not applicable. If the passages themselves conflict, it instead asks the owner which intent is binding. No artifact Apply silently edits the policy.

Arbitration uses the assignment ledger and assessment reserve, with finite admitted duration, actual charges and uncertainty holds. It grants no extra starts, steps, time, attempts or rejection allowance. One complete submitted candidate rejected for unresolved blocking findings consumes at most one rejection per review purpose, however many reviewers object. Duplicate group results count once. A ruling alone does not constitute another submitted candidate or reset any counter. After two rejections the Parent still assesses the repair plan; configured exhaustion still finishes the run Incomplete.

## Accepted deliveries and dependent work

Every local Manager receives the same operation before relevance can be assessed; a dependency blocks dependent work, not initial participation. Explore and the write barrier follow [T1](manager-apply-core.md#manager-tasks) and [T2](manager-apply-core.md#manager-tasks) within the [fixed Manager cycle](manager-apply-concept.md#fixed-manager-cycle). Local plans and dependency resolution then progress without a global Plan barrier. The Orchestrator admits ready work in the order of [T4](manager-apply-core.md#manager-tasks), not by an Agent's global plan.

A prerequisite uses a normal delivery with explicit limited scope and outstanding obligations. It has the ordinary candidate, checks, independent review and Orchestrator acceptance; it adds no separate delivery lifecycle. Acceptance makes only that scope usable and does not complete the Manager or subtree. The exact representation must preserve this distinction before recursive implementation.

Only the Orchestrator issues an accepted delivery ([D1](manager-apply-core.md#deliveries-and-composition)). A task that depends on another area implements against the producer's result only after the Orchestrator has accepted it; every later step then sees it in the current aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). An Agent's completion message is not enough.

Initial Explore, planning and independent work need not wait for that delivery. A reviewer pass alone never establishes Orchestrator acceptance. Uncertain writer effects must be reconciled before the workspace admits another step ([S2](manager-apply-core.md#steps)).

Accepted deliveries are immutable. A later blocking finding prevents further dependent use, acceptance and publication of that delivery and affected consumer results. It does not prohibit the owned correction and independent reassessment needed to resolve the finding. Admit such work only with its exact findings, permitted scope, current basis and remaining budget; it cannot use a blocked prerequisite as if it were valid. Uncertain effects still block handoff and must be reconciled first. A correction is an issue addressed to the owner ([D5](manager-apply-core.md#deliveries-and-composition)); its accepted replacement reaches affected consumers through the current aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). Rework resets no budget or rejection counter, and earlier acceptance remains historical evidence for its original subject.

Reuse a review only when its complete bound subject and inputs remain unchanged; composition onto a newer aggregate keeps or renews a local review as [D4](manager-apply-core.md#deliveries-and-composition) defines. An Agent's assertion of no impact is insufficient; changed bindings need renewed assessment and unknown effects broaden it. Current integration evidence must cover any changed aggregate.

## Admission, amendments and finite correction

Before any external execution, the Orchestrator checks the current run and controller, the allowed action, the source and candidate binding, the admission rule above, the remaining caps and the maximum duration. The reservation and the dispatch record commit together before the call. No call follows a failed or uncertain commit. Settlement is another recorded transition; recovery inspects the known effect instead of replaying it.

A failed call is retried automatically only when the adapter proves that nothing was sent, at most `limits.maxTransientRetries` times, and each attempt is charged; otherwise the invocation is uncertain and is reconciled. A Manager conversation ends after `limits.maxSessionSteps` steps; the next step starts a fresh conversation with the Orchestrator-built handoff ([S3](manager-apply-core.md#steps)).

The owner selected a rejection limit instead of a submission limit ([confirmation](../../concepts/sources/manager-process-clarification-20261017.md#review-budget-confirmation)) and subsequently approved naming it `rejectionLimit`, with `maxRejectionLimit` as its project ceiling. It is the maximum number of rejected candidate submissions per run, Manager and review purpose (local, integration or intake) since that task's last accepted delivery for the purpose ([DEC-050](../../concepts/register.md#dec-050-an-accepted-delivery-restarts-the-rejection-count)); its default remains three. This supersedes the earlier submission-count design and the misleading `reviewRounds`/`maxReviewRounds` names. The rejection counter and review engine remain unimplemented; preparing the numeric setting is not proof of either. Runs retain the policy bound at preparation; historical bytes and evidence are not renamed or reinterpreted.

The Orchestrator fixes the counter identity `(run, Manager, review purpose)` from the organization and fixed process. All limited deliveries, later completion work and corrections for that Manager share the corresponding counter. New tasks, sessions, candidate versions, claimed sub-responsibilities or submission IDs cannot reset or split it. Successful deliveries neither increase nor decrease it. They still consume the shared session, step and execution-time budget, as do checks and reviews; success never replenishes assignment capacity. An attempt number records a submission but is not the rejection counter.

A submitted candidate rejected for blocking findings or a failing required check consumes one rejection for its review purpose, regardless of the number of findings. Repeated delivery of the same submission and result returns its recorded receipt and counts once. A newly submitted candidate rejected again counts again, even with unchanged bytes. Failed or uncertain execution is not an invented review verdict: preserve its observations, charges and reconciliation requirements. An unresolved model decision blocks completion rather than granting further uncharged attempts.

After each rejection, the Manager records unresolved findings, progress and a different next action with an observable target. After two rejections, the parent assesses that plan before further permitted correction; at the root, the root does. On reaching the configured rejection limit, the task becomes `incomplete`; the Orchestrator admits no new work and finishes the run Incomplete with all work retained ([T8](manager-apply-core.md#manager-tasks), [T14](manager-apply-core.md#manager-tasks)). With the default of three, the third rejection exhausts the limit. A lower configured limit can exhaust it earlier. Parent assessment grants no extra attempt; missing authority or uncertain effects stop dependent work at once.

A later finding against an accepted delivery blocks that delivery and its consumers as above; it does not retroactively turn a successful submission into a rejection. If a new submitted candidate fails assessment, count that rejection once under its current purpose. Local and integration counters are separate because they judge different obligations. Required evidence and finite shared budget remain mandatory even when every limited delivery passes.

A continuation run needs changed run inputs or a recorded extension, never just the wish to reset rejection counts. Its reviews and checks run again in full; earlier approvals remain history.

## Step basis and continued conversations

A step starts on a confirmed workspace version built from the current aggregate and the Manager's own unaccepted changes, and records the aggregate version it saw ([D3](manager-apply-core.md#deliveries-and-composition)). Nothing changes that workspace while the step runs ([S6](manager-apply-core.md#steps)), and its resulting changes are recorded before a successor step is admitted ([S4](manager-apply-core.md#steps)). An action attributed to a stale step cannot modify the new workspace or produce current evidence; a duplicate observation of an old completed action may only recover its original receipt.

Run inputs, Manager identity, findings and rejection counters survive a conversation handoff ([S3](manager-apply-core.md#steps)). Idle dialogue does not consume execution time, but each step's container startup and work keep their accounting. Unknown outstanding effects retain their holds ([S2](manager-apply-core.md#steps)).

A recorded materialization is not proof that it happened. A physical update without durable confirmation is not permission to continue. Reopening after interruption reconciles the exact prior and resulting trees, own changes, aggregate version and actual workspace before any new step. No automatic replay or Agent acknowledgement resolves an uncertain materialization.

## One run state machine

Only `run` owns these lifecycle meanings. `apply` requests transitions for verification and publication. The journal stores the committed facts, and views derive from them.

| State | Meaning and permitted next transition |
| --- | --- |
| Prepared | Fixed inputs and a run unit are committed. Start work (Running) or finish without dispatch. |
| Running | Work, checks, reviews or integration are being coordinated; some tasks may wait while others continue. Move to Waiting only when all remaining useful work depends on external input and no effect is active or uncertain. Move to Verifying when the root is done ([T13](manager-apply-core.md#manager-tasks)). When a task becomes incomplete, or an answer changes the captured run inputs, admission stops, running steps end or are reconciled, and the run finishes Incomplete with all work retained ([T14](manager-apply-core.md#manager-tasks), [W3](manager-apply-core.md#while-a-run-waits)). |
| Waiting | No container runs ([W1](manager-apply-core.md#while-a-run-waits)), with a named next actor and reason: an owner decision, the owner's approval of publication or another external prerequisite. A context-only answer resumes Running, a granted approval resumes Publishing, and an answer that changes the captured run inputs finishes the run Incomplete. |
| Verifying | Required aggregate checks, the Orchestrator's final evidence check and source preflight. A correctable finding returns to Running only if rejection allowance and budget remain. Current passing evidence permits Publishing. Where the configuration requires the owner's approval, the run first waits on an approval question: approval resumes Publishing, and a declined approval finishes the run Incomplete with its retained candidate. |
| Publishing | The guarded write and read-back are being observed. No new implementation or competing publication is dispatched. Complete to Finished, or enter Reconciling on partial or unknown observations. |
| Reconciling | A stop, interruption, controller loss, partial or unknown publication or uncertain execution needs inspection. New work and publication are blocked; only bounded inspection and termination run. Once effects are known, resume the eligible phase on the same basis or finish with the observed result. |
| Finished | Exactly one immutable outcome below. No further dispatch or publication. Later runs link to it without changing it. |

Reconciling is a blocked condition, not a successful end. An inspection that cannot establish the effect leaves the blocker visible. If the store is unavailable, inspection shows the last committed state and its observation limit. An internal dependency cycle is not Waiting: the common parent or the root assesses it within its mandate, or the run ends Incomplete.

Stop requests, input drift and exhaustion first stop admission and reconcile active effects; then the run can finish. A proven partial publication stays in the working tree; the owner commits or reverts it before the next run, which captures that state. The old request is not repeated. Applied requires the original required evidence plus exact publication and read-back. A cancellation acknowledgement, process exit or Agent report never establishes it.

## Terminal outcomes and displayed status

| Outcome on Finished | Grounds |
| --- | --- |
| Applied | Current local and integration reviews, the final evidence check, guards and read-back establish the exact changed artifacts and registry. |
| NoChange | The same assessment and current source and read-back establish that nothing needs publication. Absence of edits alone is not enough. |
| Stale | Inputs drifted; the run is quiescent and cannot publish. |
| Stopped | An explicit owner stop reached known quiescence. Work and evidence are retained. |
| Incomplete | Budget or rejection-limit exhaustion, an unanswered question, a needed model change, a declined publication approval or the lack of a viable authorized correction prevents completion. The reason, retained work and open questions are recorded. |
| Failed | A definitive acquisition, execution, validation or publication failure. Actual effects are preserved; a known partial publication is Failed, not rolled back. |

An acquisition rejected before Prepared is a preparation failure, not a run. Unknown effects never become Failed just to clear a blocker; they stay Reconciling until established. Later discoveries add findings instead of rewriting results.

"Needs decision" is attention on an open question, not an outcome; it can appear while the run is Running or Waiting. "Locally verified" and "aggregate verified" describe evidence for exact subjects, not states. Publishing is not Applied, and Applied is not deployed; deployment is a separate operation outside Apply.

An assignment closes Completed by itself when its last run finished Applied or NoChange with no required obligation open. Otherwise it stays open until the owner continues it or closes it; closing records the last run's outcome, and open questions stay recorded with the closed assignment. An assignment with an unfinished run cannot close. A closed assignment is not reopened; new work is a new assignment.

## Library traces and focused gates

| Library situation | Budget and lifecycle result |
| --- | --- |
| M1 initial realization | Root and Package work run through fresh reviews; required findings close with current evidence; verification and read-back publish A1. R1 finishes Applied. |
| Model-only five-to-three change | A new Apply invocation captures M2 and the actual A1 and reassesses everything. R2 can finish Applied only after its own gates. R1 stays Applied. |
| A run ended awaiting a model change while a loan-limit finding was open | The continuation carries the finding; the reviewer marks it not applicable on M2 and records the new three-loan requirement as a linked finding with current evidence. |
| Lending asks; catalog can continue | Lending waits; the run stays Running, and catalog's work is charged. |
| All work awaits an owner answer | Effects quiesce; the run is Waiting with the exact question. Nothing is charged for the wait. |
| Review rejects suspension or return behavior twice | The parent assesses a concrete repair plan before further permitted correction. At the default limit a third rejection makes the task incomplete, and the run finishes Incomplete ([T14](manager-apply-core.md#manager-tasks)); renaming tasks or splitting deliveries grants no fresh counter. |
| A skeleton and a final delivery both pass | Both consume shared execution capacity but no rejection allowance. An intervening rejection remains counted after the later pass. |
| Only the completion reserve remains | Ordinary Manager work is refused; eligible reviews and checks may continue. Required unfinished Manager work leaves the run Incomplete. |
| A required L1/L2 case cannot be afforded | The finding stays open; a passing ordinary return does not discharge it, and nothing unsupported is published. |
| A step times out; writer state unknown | Reconciling with the reservation held; no replacement writer or publication until termination and effects are known. |
| Source drift or partial publication | Admission stops and the run reconciles. Known drift finishes Stale; a known partial write finishes Failed with exact paths. Unknown effects stay blocked. |

Implement deterministic transition and accounting checks for these cases, plus duplicate reservation and settlement, refusal of ordinary Manager work that would use the reserve, release of unused time, no inherited Package override, no charge while waiting, rejection counters surviving task recreation, advisory findings that never block, retries only after proven non-dispatch, a fresh conversation at the step limit, publication waiting for a required approval, immutable outcomes and finding resolution only through independent current evidence. Real adapter termination is tested separately; fake events cannot establish that a process stopped.
