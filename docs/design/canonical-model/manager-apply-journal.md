# Journal and continuation contracts for Apply

Date: 16 October 2026. Proposed P01 runtime records and persistence for [configuration and mandates](manager-apply-configuration.md) and [execution budgets/lifecycle](manager-apply-execution.md). This incorporates [continuing assignments, questions and work reuse](../../concepts/reflections/desired-working-day-20261016.md). The execution contract owns transition meaning; this document owns recorded facts, storage and atomic append. This is a replacement design, not a current storage format, executed library run or historical-record migration.

## One authoritative history

Use one local SQLite database per governed repository state store, with assignment streams, committed append batches, typed events and immutable byte payloads. The journal owns the recorded control history. Reports and current-state views derive from it; they cannot separately authorize execution. Candidate manifests refer to immutable artifact/registry payloads, while ordinary workspaces remain disposable materializations.

The adapter is `internal/adapters/journal`, implementing `run.Journal` through `database/sql` and a pinned `modernc.org/sqlite` driver. That driver provides a CGo-free SQLite implementation ([driver documentation](https://pkg.go.dev/modernc.org/sqlite)). Pure model, organization, realization and run contracts import neither SQL nor driver types. The previous generic `filestore` proposal is replaced for this responsibility.

This adds a database dependency to obtain one explicit transaction boundary for events, correlation and budget reservations. It replaces the proposed custom multi-file journal layout. Keep useful private-directory/identity/lease ideas from the old record store; SQLite access does not automatically inherit its filesystem guarantees. No message broker, ORM or event-sourcing framework is required.

## Physical store and commit contract

Use a private local state directory explicitly selected outside the governed source tree. Bind the store to a repository identity; a copied database does not automatically authorize another checkout. Select WAL mode, full synchronization and foreign-key enforcement, and verify the effective settings. SQLite documents the durability role of `synchronous=FULL` in WAL mode ([synchronization](https://www.sqlite.org/pragma.html#pragma_synchronous)); WAL requires local same-host access and proper handling of its auxiliary files ([WAL](https://www.sqlite.org/wal.html)). Pin and test the driver/embedded engine, including its applicable fixes, before adoption. No current driver version is added by this design.

One controller holds the repository execution lease; inspection can use bounded read snapshots. Do not hold a SQL transaction while an Agent or check executes. A busy/failed/uncertain commit prevents the dependent external effect. A lease timeout alone does not establish that an earlier controller or child writer stopped. Recovery first reconciles known activity.

Make that lifetime explicit through one additional consumer-owned port:

```go
// Package apply. A Host controller session is distinct from an Agent session.
type Control interface {
	Acquire(context.Context, ControlRequest) (ControlSession, error)
}

type ControlSession interface {
	Binding() ControlBinding
	Journal() run.Journal
	Check(context.Context) error
	Close() error
}
```

`ControlRequest` selects the captured repository identity and explicit state directory. Acquisition takes an exclusive process-lifetime lock and binds a validated store plus new controller epoch; its binding is a technical lifecycle capability, not decision authority. `apply` revalidates captured inputs under that lease before dispatch, creates `run` with the returned journal and closes control on exit. The journal adapter implements this port; SQL transactions remain short. Read-only inspection uses the same store format without acquiring execution authority.

A failed acquisition grants no dispatch. If it returns a partial session with an error, the caller still closes that handle and preserves the failure; it must not leak a held repository lock. `Close` releases only controller resources, not an assertion that all previously launched processes terminated.

Anchor the execution lock and active store registration in repository control metadata, independently of the selected database directory. An alternate `--state-dir` cannot bypass the lock or hide unresolved work in another store. Initial store binding uses checked creation; a missing/replaced registered database requires explicit reconciliation rather than automatic creation of empty history. State initialization failures before first dispatch remain distinguishable from loss of an established store.

Append requires the current live controller binding. Publication requests/permits carry that binding and a run-scoped Host authorization check; the publisher checks both at the write boundary. `ControlSession.Check` verifies the held lease, store identity and epoch. The additional Host check validates that the permit's run is still current and eligible to publish; a live controller alone cannot authorize a superseded run. Apply serializes handoff with publication while holding it. Releasing the controller does not erase pending effects or prove child termination; the next acquisition can inspect them but cannot dispatch dependent work until reconciled. No CLI/MCP handler owns this lifecycle.

| Stored relation | Essential columns and constraints |
| --- | --- |
| `store_meta` | Format version, bound repository/store identity and controller epoch. Unknown format or identity fails opening; no fallback to old records. |
| `streams` | Assignment ID primary key and current nonnegative revision. First append uses expected revision zero. |
| `blobs` | SHA-256 of exact bytes as primary key, byte length and original bytes. Reads recompute the digest; an existing key with inconsistent bytes is corruption. Media meaning comes from the typed referring record. |
| `batches` | Assignment ID plus batch ID unique; content digest, expected revision and committed sequence interval. An exact repeat returns the original receipt. |
| `events` | Assignment ID plus sequence primary key, globally unique event ID, batch ID, kind/version, optional run/task/session/step references, UTC observation time, causal predecessors and body blob reference. Foreign keys bind stream, batch and body. |

Append verifies its expected revision, inserts the batch/events and advances the stream head in one transaction. Required blobs must already be durable or be inserted in that transaction. After acknowledgement, the entire batch is visible; an incomplete batch cannot become an observed committed prefix. Append never edits/deletes an earlier event. Corrections use new events.

Look up an existing batch ID before rejecting an old expected revision: an identical retry returns the prior result; different bytes under the same ID fail. If commit outcome is uncertain, query that exact batch before proceeding. Absence must be established from the validated store; an inaccessible database is not evidence of absence. No database retry resends provider work.

Persist exact versioned UTF-8 JSON for Host records, rejecting duplicate/unknown fields and bounded-size violations. Encode known fields in a fixed order and preserve order in lists; IDs are opaque strings, counters are bounded nonnegative integers and observation timestamps are UTC. Batch identity hashes the defined encoded append request, not a regenerated report. Raw prompts/provider wire bytes retain their own byte identity; do not normalize them through the Host JSON codec. This storage codec does not redefine canonical-model bytes.

State is reduced from committed typed control events. Large prompt/transport blobs are loaded on demand, not folded into every state read. Rebuilding a projection performs no external actions. A read pins the committed revision and returns ordered pages; timestamp order is not invented causal order across streams. A database commit records observations, not a transaction with Codex, the filesystem or deployment.

The first version retains all referenced payloads, including rejected candidates. No automatic cleanup removes in-progress evidence. Backups/exports must capture a consistent database including required WAL state; copying only the live main file is not an export protocol. A later retention policy needs explicit reference accounting.

## Stream and value contracts

Change the earlier run-only append scope to an assignment stream. This is necessary for atomic successor creation, inherited budget and questions answered between runs. Run IDs remain mandatory for run-scoped events.

```go
// Package run. Bodies are closed, versioned event values, not arbitrary maps.
type AppendBatch struct {
	Assignment       AssignmentID
	ID               BatchID
	ExpectedRevision uint64
	Events           []Event
}

type Event struct {
	ID       EventID
	Actor    ActorID
	Run      RunID
	Task     TaskID
	Session  SessionID
	Step     StepID
	At       time.Time
	CausedBy []EventID
	Body     EventBody
}
```

Optional identities use explicit absence in the storage codec. `EventBody` is a closed tagged contract with kind/version-specific validation. An assignment event may have no run; an answered question still identifies its originating run/task in its body. `EventQuery` selects the assignment, optional run/batch and a sequence interval; its cursor pins a committed revision. `AppendReceipt` identifies the committed batch and sequence range. `Put`/`ReadBlob` and the four-method journal port stay narrow; domain transitions are owned by `run`.

| Record | Required data |
| --- | --- |
| Assignment | Owner input reference, goal/commissioned remaining stages, constraints, authority source, repository identity, optional absolute deadline, cap/grant revisions, ledger and completion-plan references, amendments, linked runs and closure fact. |
| Run binding | Model closure/lock, captured repository tree/census and publication guard, config/prepared profiles, instructions, initial registry, operation contract identity and product build. |
| Retained work | Exact predecessor run/candidate, original artifact base, matching registry, selected composition, open findings and carry-forward reason. Old approval is never promoted. |
| Task | Current run/Manager, parent task, attempt, outcome, complete responsibility, candidate base, effective mandate reference, remaining constraints and any predecessor-task link. |
| Mandate | Immutable rendered view, original source references/revisions, effective configuration and registry/task basis. Reviewer receives the corresponding authority basis. |
| Question/proposal | Stable question ID, version, originating task/run, affected work, model passages/basis, decision/reason, alternatives, exact proposed before/after source edits and effect on work. |
| Answer | Question/version/proposal identity, exact owner input, actual ingress/actor authority, selected disposition and any interpreted answer, with unresolved ambiguity explicit. |
| Candidate/review/check | Existing exact input/tree/registry/subject bindings, assessments, actual findings and observed results. Carry-forward judgment uses keep/adapt/replace/unresolved within existing assessments. |
| Required finding/assessment | Stable finding reference, issuing role/session, task/scope and original model/input basis, requested evidence and optional original candidate; current reviewer applicability judgment, adjusted requirement/replacement references where needed, exact subject and resolution/evidence grounds. Open/resolved is a history projection, not an editable flag. |
| Publication | Known publication ID, current run and captured source guard, verified candidate/registry, prepared request, actual completed paths and read-back/uncertainty. |
| Budget | Allocation/plan/reservation ID, assignment/grant revision, purpose, responsible scope, resource quantities, effect identity, finite execution duration, actual settlement or unresolved hold, and release/reallocation grounds. |

Before/after model edits are exact captured file bytes or exact identified source fragments with matching prior content identities. A changed proposal gets a new version. An explanation of desired behavior alone is not a concrete edit authorization. Original bytes and the owner's response stay retrievable.

## Budget dispatch and lifecycle

`run` validates transitions against the authoritative [execution contract](manager-apply-execution.md). The expected revision prevents two controllers from committing from the same state. Budget reservation and dispatch preparation are one committed batch before an external call. A reservation belongs to an existing allocation and its ordinary/protected purpose; it is not another copy of its allowance. A prepared run consumes its run unit when committed.

An uncertain invocation retains its full hold. If inspection proves no dispatch occurred, an explicit release restores unconsumed capacity; settled consumption never decreases. Known completion settles actual execution and releases only unused held capacity to its proper pool. Provider usage and estimates are separate observations. Cap/deadline amendments preserve consumption, hold identities and their authority source; their changed effective basis requires a successor. No amendment or new run resets the ledger.

| Level | Authoritative interpretation |
| --- | --- |
| Assignment | Open/closed fact, linked runs, remaining commissioned work and amendments. Activity labels derive from these facts; closure is distinct from an individual run finishing. |
| Run | Exactly the states and Finished outcomes in the execution contract. Needs decision is question attention; aggregate verified is evidence; unknown execution/publication requires Reconciling. No second state enumeration is defined here. |
| Task | Existing local work/check/review/integration phases plus question/child waits. Logical responsibility and review-purpose counters survive task recreation. Ordinary rework stays in the same fixed run. |
| Question | Open, answered pending binding/action, resolved, deferred, stale or replaced by a new version. Recording an answer does not imply its model effect succeeded. |

The initial event catalogue has these version-1 tags. Each has a closed payload codec; state-change payloads identify prior/resulting state and grounds, validated against the current transition. Do not infer state solely from a human message saying “done.”

| Family | Event tags |
| --- | --- |
| Assignment and run | `assignment.created`, `assignment.amended`, `assignment.finished`, `run.prepared`, `run.state-changed`, `run.finished` |
| Task and delivery | `task.created`, `task.state-changed`, `attempt.started`, `delivery.accepted` |
| Budget | `budget.plan-recorded`, `budget.allocated`, `budget.reallocated`, `budget.reserved`, `budget.settled`, `budget.released` |
| Execution and transport | `session.prepared`, `session.observed`, `step.prepared`, `transport.prepared`, `transport.observed`, `tool.requested`, `tool.replied` |
| Realization and evidence | `candidate.captured`, `assessment.recorded`, `review.recorded`, `check.prepared`, `check.observed`, `composition.recorded` |
| Questions and intent | `question.proposed`, `question.state-changed`, `answer.received`, `answer.bound`, `model-change.observed` |
| Publication | `publication.prepared`, `publication.observed` |

`run.finished` records the single immutable Finished outcome and grounds. Supersession uses that event with outcome Superseded, not a competing finish/state tag. Assignment closure is recorded separately. Typed budget events enforce identity/idempotency, pool conservation and purpose; reducers do not infer a grant or completed effect from a narrative report. A completion plan references outstanding required review/check obligations and recorded release reasons.

Use `assessment.recorded`/`review.recorded` for early model-first concerns, findings and explicit resolutions; no extra finding stream or task lifecycle is needed. Parent/predecessor references preserve outstanding requirements rather than copying another mutable list. An old resolution remains historical when the new subject/basis requires current assessment. Retain selected case output/assertion references where execution of that case is required; a generic check pass cannot stand in for missing evidence. The execution contract owns the acceptance guard and its semantic limits.

## Questions answers and model changes

1. The responsible Manager raises a source-grounded question through the parent. Host records its exact version, proposal and affected tasks. Other ready work can continue, but required unresolved work prevents aggregate completion.
2. The application receives the actual owner's answer. It records the raw answer and binds it to the displayed question/proposal version. Repeated identical ingress IDs are deduplicated. An Agent-provided actor name supplies no owner authority.
3. The answer's effect is classified with a recorded reason. Existing-rule context can be supplied in a later step of the same fixed run. Elaborating an alternative authorizes preparation only. Deferral leaves dependent work waiting. A new obligation or authority/configuration change needs a new bound basis.
4. Acceptance authorizes the exact displayed edit within the existing mandate; do not request the same permission again. A different free-text answer can authorize a clear different edit. Remaining material ambiguity receives a revised proposal. On source drift, withhold the edit and record revalidation; preserve the answer and ask again only when the choice materially changes or authority is insufficient.
5. Artifact Apply never edits its model/configuration. An authorized model operation or external owner/caller performs the exact approved edit. Record its actual result, source and new model identity separately from receiving the answer. The first Apply runtime can expose this pending step; a complete model-edit command framework is not a prerequisite.
6. When the new model exists and continuation is authorized, establish the safe handoff below and start a successor. Acknowledge model updated, old run superseded and work resumed only after those respective observations exist. A partial acknowledgement describes precisely which step remains pending.

Notifications are projections of these facts and separate settings, not another source of permission. A reviewer checks material decisions against the recorded mandate and answer scope. Approved code cannot authorize an otherwise unauthorized business-policy change.

## Safe successor and publication basis

Use the repository execution lease to serialize supersession and publication. Stop new dispatch from the predecessor, quiesce or reconcile active writers and finish/inspect any publication already in progress. No new controller takes over merely because a lease deadline elapsed. Preserve actual partial checkout state and acquire it afresh.

Capture the new model/configuration and actual repository source. Validate selected retained work for exact bases, valid ownership/registry, protected inputs and conflicts. Incomplete work may be admitted with its missing evidence and findings. Conflicts return to their accountable Manager; there is no semantic automatic merge.

For an unfinished predecessor, commit its Finished/Superseded result, successor binding, selected retained-work reference and shared budget reservation together. Only then dispatch the successor's root. Every previously Finished run retains its outcome; a later run links to it without rewriting it. In particular, Applied R1 is not Superseded by R2. A crash leaves either the old committed state or the new one; unresolved external effects still require inspection. Tasks and sessions in the successor receive new identities with provenance links.

Keep these three values distinct:

| Value | Library meaning |
| --- | --- |
| Captured repository source | The actual checkout and registry, such as published A1 or the unchanged pre-publication base. This supplies the publication guard. |
| Retained work | An unpublished predecessor candidate and matching registry selected as starting material. It may contain incomplete changes. |
| Current candidate | The new run's assessed result after composing/adapting that starting material. This is what reviews/checks and final publication concern. |

A private candidate never becomes the presumed contents of the real checkout. Final publication compares against the successor's acquired source, writes the verified difference and reads back. Apply holds the execution lease and serializes handoff with publication; preflight permits are bound to the current run/controller epoch and cannot be reused after supersession. Database transactions do not make multi-file publication atomic.

Initially repeat required reviews/checks, integration and final verification for the successor. Keep/adapt/replace/unresolved explains retained work; it does not replace Definition assessments or current evidence. A new run never copies a prior pass into current approval.

## Library traces

### Two actual model versions as the first planned experiment

The owner can commission one assignment covering both planned stages. Run R1 binds M1, produces reviewed A1 and finishes Applied. The open assignment awaits its commissioned M2 stage. The owner/caller changes only `loan-policy.spec.maxOpenLoans` from five to three. Run R2 binds M2 and actual A1, preserves the assignment ledger and replans its completion reserve. Lending adapts admission; other areas can propose justified no-change results. All required current assessments and checks run before A2 is published. The proposed [Playground](manager-apply-playground.md) owns the interface/independent evidence; no question is invented for a rule M1 already resolves.

### A hypothetical missing decision during work

An intentionally inconsistent future draft might both preserve existing loans and require all existing holdings to fit a lowered limit. This is not the authored M1/M2. Lending raises Q1/version 1 with the conflicting passages and an exact proposed correction retaining the structured policy as the numeric authority. Root routes it to the owner because its mandate cannot choose a new business rule.

The owner accepts that displayed correction. The journal distinguishes answer received, edit authorized, actual model revision observed, old run superseded and successor started. A predecessor's unfinished lending candidate remains useful starting material. Its earlier reviews remain historical; the successor assesses it again. If no meaningful new evidence can be produced before the remaining budget expires, the assignment reports incomplete and asks for an explicit next decision rather than resetting its limits.

### Example inspectable timeline

| Observation | What it establishes |
| --- | --- |
| Task delegated; exact lending prompt and mandate stored | Who was commissioned, with which inputs and authority. |
| Step prepared; send outcome observed or unknown | What might have executed; preparation alone is not delivery. |
| Candidate C1 captured; reviewer requests changes | Actual bytes and the grounded issue, such as allowing a sixth loan. |
| C2 captured and locally reviewed; reminders integrated | Current local and child evidence, still distinct from final approval. |
| Q1 answered; model edit pending | An answer exists, while intent has not yet been changed. |
| Successor prepared with retained C2 and current source | Which work was reused, which basis changed and what allowance remains. |
| Current aggregate verified; publication/read-back observed | The exact machine-assessed result and actual checkout outcome. |

## Ownership and focused gates

`run` owns assignment/question/budget transitions and event meaning. `organization` owns rendered configuration/context/mandates from supplied facts. `realization` owns artifact/registry/composition values. `apply` owns capture, safe successor preparation and guarded publication. The journal adapter owns SQL, byte storage and atomic append; it makes no semantic decision about a loan, answer or review. CLI/MCP expose the same services and journal projections.

Implement gates for atomic batch/head updates, exact duplicate handling, conflicting/stale revisions, unknown commit reconciliation, blob corruption, process failure around dispatch, answered-old-proposal refusal, context-only continuation, superseded publication refusal, consumed-budget continuity, retained/source-base separation and meaningful carry-forward review. Pin and exercise the actual Go driver and supported filesystems; a format probe does not establish durability. Then test real role communication and the two real Applies separately.

Future experience notes may cite this history; a doctor may supply separate operational evidence. Neither is required to complete the first journal or turns an old observation into a binding rule. A rich cockpit, distributed coordinator and general Government/permission language remain outside this first implementation.

Return to the [configuration](manager-apply-configuration.md), [ports](manager-apply-port-contracts.md), [rebuild plan](manager-apply-rebuild-plan.md) or [Apply plan](manager-apply-implementation-plan.md).
