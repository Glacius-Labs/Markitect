# Budget and lifecycle contract for the first Apply

This contract owns budget accounting, execution admission, findings that block completion, run transitions and their displayed meaning. The [configuration](manager-apply-configuration.md) owns the input fields and Package overrides; the [journal](manager-apply-journal.md) owns durable records and storage. The `run` package implements these rules.

## Assignments and runs

An assignment is one owner request, such as "Apply the current library model". An Apply invocation either starts a new assignment or continues an open one by naming it. An assignment holds the budget and links its runs. A run binds one fixed set of inputs: model, configuration, repository source and profiles. Ordinary rework stays inside the run. A further run in the same assignment is always a **continuation**: a run finished Incomplete, and the owner continues the assignment with a recorded extension or a changed basis. The finished run keeps its outcome; the new run links to it. When an owner answer requires a model edit, the run first quiesces and finishes Incomplete with the reason "awaiting model change" and its retained work; after the owner has committed the edit, a continuation starts. There is no handoff between unfinished runs.

A repository has at most one unfinished run. A new Apply on a repository whose run is still unfinished is refused until that run finishes, is stopped, or the invocation continues its assignment.

The two-Apply demonstration therefore uses two ordinary invocations: R1 on M1 and, after the model edit, R2 on M2 against the published A1. Each is its own assignment, and both are reported. No budget carries over between them, and no approval does either.

## A shared assignment budget

The assignment has four caps: prepared runs, session starts, Agent steps and active execution time. The configuration supplies their defaults and hard project ceilings. There are no per-Manager copies of the balance, and Managers do not manage budgets; the Host accounts for every invocation once, against the assignment.

- A run consumes one run unit when its fixed binding is committed as Prepared. A rejected acquisition prepares no run.
- A session start consumes one start unit. Resuming an established session does not; a replacement session does.
- A step is one Host-dispatched Agent turn, including a continued conversation. Tool callbacks belong to their turn.
- Active execution time is the sum of charged invocation durations: session startup, Agent turns, project checks and Agent-requested tool processes. It excludes Host control work, idle sessions and waiting for the owner or for other tasks. It is neither CPU time nor a delivery clock.

Every charged invocation has a finite maximum duration recorded before dispatch. The Host reserves that maximum, then settles the observed duration and releases the rest. If dispatch or termination is uncertain, the full reservation stays held until reconciliation; an error or timeout cannot release it. Unknown provider usage stays unknown. These units do not establish a monetary ceiling. The Host supplies clock and elapsed-time observations to the transition logic, which reads no ambient clock; within an invocation the observed elapsed duration counts, never a difference of report timestamps.

Composition, source acquisition and preflight, journal control, publication with read-back and emergency termination are Host control operations outside the budget. Each has a finite `hostOperationTimeout` and size limits fixed at preparation. They can record failures and stop effects after Agent capacity is exhausted; they cannot investigate, repair or review.

## One ledger with protected completion capacity

The configuration sets a completion reserve for session starts, steps and execution time. The rule is one admission check:

- **Writing work** (implementation, rework and parent integration authoring) is admitted only if the capacity left after its reservation still covers the completion reserve.
- **Assessment work** (local, integration and final reviews and required checks) may use any remaining capacity, including the reserve.

Before a run is Prepared, the Host rejects a reserve that cannot cover at least one session start and one step for every required local and integration review of the organization. Such a rejection is a preparation failure and consumes no run unit. That check is structural; whether the reserve is generous enough remains a judgment that a run can prove wrong.

Settled consumption never decreases. Settlement records an actual overrun instead of rejecting it; an overrun sets the remaining capacity to zero, blocks further work and is reported. Waiting consumes nothing: while all work is quiescent and waiting for an answer, no step or execution time is charged. Status inspection starts no Agent.

Budget pressure never waives evidence. When writing work no longer fits, the Host retains the candidate and the run finishes Incomplete with its open work. The owner can extend the assignment's caps within the project ceilings; the extension is recorded and used by the continuation run. A run that ended Incomplete stays Incomplete.

## Evidence need survives budget pressure

What evidence an obligation needs is decided separately from whether the budget can afford it. A reviewer who sees a plausible error that could violate a consequential boundary selects a small distinguishing case, starting from the model before reading the implementer's solution. A current case that already distinguishes the concern is reused. For lending, useful cases are a suspended member who can still return a book, and a repeated return of loan L1 after the copy has been lent again as L2.

A necessary missing check is a blocking **finding**: a Host-recorded identity, the affected responsibility, its model and input basis, the requested evidence and the originating reviewer. A finding can exist before any candidate. It has three states:

| State | How it is reached |
| --- | --- |
| Open | Recorded by a reviewer. Omission from a later report does not change it. |
| Resolved | A later independent review cites actual current evidence for the current subject. |
| Not applicable | A later independent review records grounds on the authorized basis, for example a changed model. |

A finding is **blocking** or **advisory**. Only blocking findings prevent acceptance and count toward review rounds. Advisory findings, such as a suggested simplification, are recorded for the owning Manager and never block.

A Manager may propose a resolution but cannot record one. Budget or round exhaustion cannot resolve a finding. A generic pass, an invented receipt, a failed or unknown check, or an old resolution on a changed subject cannot close it. When a specific case is required, its evidence must show that the case ran; an exit code of zero alone is not enough. If a requirement changes in substance, the old finding becomes not applicable and a new, linked finding carries the changed requirement. Open findings carry into a continuation run even when their candidate files are discarded.

Role context carries the complete set of open findings for the task's responsibility and its recorded dependencies, with their grounds and evidence references. Paging the set is allowed; an unread page cannot discharge an obligation. A receiving Manager that is not responsible for a finding reports that through the common parent, which records the corrected accountability; "not mine" cannot orphan it.

## Accepted deliveries and dependent work

Only the Host issues an accepted delivery: the exact candidate, registry revision, completed reviews and checks, and the child deliveries it contains. A task that depends on another area starts only when the producer's accepted delivery has been composed into its base. An Agent's completion message is not enough.

Accepted deliveries are immutable. If a later blocking finding concerns an accepted delivery, then neither that delivery nor its consumers (tasks whose base composes it) can be dispatched further, accepted or published until the finding is resolved. The common parent selects a replacement delivery explicitly and has it composed into the affected consumers. Correcting it is ordinary bounded rework and resets no budget or round counter.

## Admission, amendments and finite correction

Before any external execution, the Host checks the current run and controller, the allowed action, the source and candidate binding, the admission rule above, the remaining caps and the maximum duration. The reservation and the dispatch record commit together before the call. No call follows a failed or uncertain commit. Settlement is another recorded transition; recovery inspects the known effect instead of replaying it.

A failed call is retried automatically only when the adapter proves that nothing was sent, at most `limits.maxTransientRetries` times, and each attempt is charged; otherwise the invocation is uncertain and is reconciled. A Manager session ends after `limits.maxSessionSteps` steps; the next step starts a recorded replacement session from the journal's summary of the task.

`reviewRounds` is the number of candidate submissions per run, Manager and review purpose (local or integration), including the first; the default is three. The counter is keyed by `(run, Manager, review purpose)`, so recreating a task or session does not reset it. An attempt number identifies one submission within a task; the round counter is the limit. A repeat with the same submission ID and the same bytes returns the existing receipt. A new submission ID is a new round, even with unchanged bytes. A candidate that fails a required check still consumes its round.

After each rejection, the Manager records the unresolved finding, the progress made and a different next action with an observable target. After the second rejection, the parent assesses that plan before a third submission; at the root, the root does. This grants no extra round. Missing authority or uncertain effects stop dependent work at once.

A continuation run needs a changed basis or a recorded extension, never just the wish to reset rounds. Its reviews and checks run again in full; earlier approvals remain history.

## One run state machine

Only `run` owns these lifecycle meanings. `apply` requests transitions for verification, publication and handoff. The journal stores the committed facts, and views derive from them.

| State | Meaning and permitted next transition |
| --- | --- |
| Prepared | Fixed inputs and a run unit are committed. Start work (Running) or finish without dispatch. |
| Running | Work, checks, reviews or integration are being coordinated; some tasks may wait while others continue. Move to Waiting only when all remaining useful work depends on external input and no effect is active or uncertain. Move to Verifying when the required deliveries are accepted. |
| Waiting | Quiescent, with a named next actor and reason: an owner decision, the owner's approval of publication, an authorized model edit or another external prerequisite. A context-only answer resumes Running; an answer that changes the basis finishes the run Incomplete once quiescent. |
| Verifying | Required aggregate checks, the Host's final evidence check and source preflight. A correctable finding returns to Running only if rounds and budget remain. Current passing evidence permits Publishing, after the owner's approval where the configuration requires it; a declined approval finishes the run Incomplete with its retained candidate. |
| Publishing | The guarded write and read-back are being observed. No new implementation or competing publication is dispatched. Complete to Finished, or enter Reconciling on partial or unknown observations. |
| Reconciling | A stop, interruption, controller loss, partial or unknown publication or uncertain execution needs inspection. New work and publication are blocked; only bounded inspection and termination run. Once effects are known, resume the eligible phase on the same basis or finish with the observed result. |
| Finished | Exactly one immutable outcome below. No further dispatch or publication. Later runs link to it without changing it. |

Reconciling is a blocked condition, not a successful end. An inspection that cannot establish the effect leaves the blocker visible. If the store is unavailable, inspection shows the last committed state and its observation limit. An internal dependency cycle is not Waiting: the common parent or the root assesses it within its mandate, or the run ends Incomplete.

Stop requests, input drift and exhaustion first stop admission and reconcile active effects; then the run can finish. A proven partial publication becomes the actual source for a later run; the old request is not repeated. Applied requires the original required evidence plus exact publication and read-back. A cancellation acknowledgement, process exit or Agent report never establishes it.

## Terminal outcomes and displayed status

| Outcome on Finished | Grounds |
| --- | --- |
| Applied | Current local, integration and final evidence, guards and read-back establish the exact changed artifacts and registry. |
| NoChange | The same assessment and current source and read-back establish that nothing needs publication. Absence of edits alone is not enough. |
| Stale | Inputs drifted; the run is quiescent and cannot publish. |
| Stopped | An explicit owner stop reached known quiescence. Work and evidence are retained. |
| Incomplete | Budget or round exhaustion, an unanswered question, a needed model change, a declined publication approval or the lack of a viable authorized correction prevents completion. The reason, retained work and open questions are recorded. |
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
| Review rejects suspension or return behavior twice | The parent assesses a concrete repair plan before candidate three. A third rejection ends Incomplete; renaming the task grants no fourth candidate. |
| Only the completion reserve remains | No further writing is admitted. The candidate is retained and the run ends Incomplete with its open work. |
| A required L1/L2 case cannot be afforded | The finding stays open; a passing ordinary return does not discharge it, and nothing unsupported is published. |
| A step times out; writer state unknown | Reconciling with the reservation held; no replacement writer or publication until termination and effects are known. |
| Source drift or partial publication | Admission stops and the run reconciles. Known drift finishes Stale; a known partial write finishes Failed with exact paths. Unknown effects stay blocked. |

Implement deterministic transition and accounting checks for these cases, plus duplicate reservation and settlement, refusal of writing work that would use the reserve, release of unused time, no inherited Package override, no charge while waiting, round counters surviving task recreation, immutable outcomes and finding resolution only through independent current evidence. Real adapter termination is tested separately; fake events cannot establish that a process stopped.
