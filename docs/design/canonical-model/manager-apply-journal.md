# Journal and continuation contracts for Apply

This contract owns the recorded facts, their storage and atomic append, questions and answers, and continuation runs. The [execution contract](manager-apply-execution.md) owns what transitions mean; the [configuration](manager-apply-configuration.md) owns the inputs.

## One authoritative history

One local SQLite database per governed repository holds one ordered event stream, committed append batches, typed events and immutable byte payloads. The journal is the recorded control history. Reports and current-state views derive from it and cannot authorize execution on their own. Candidate manifests refer to immutable artifact and registry payloads; workspaces remain disposable materializations.

The adapter is `internal/adapters/journal`. It implements `run.Journal` and `apply.Control` through `database/sql` and a pinned, CGo-free `modernc.org/sqlite` driver ([driver documentation](https://pkg.go.dev/modernc.org/sqlite)). The pure model, organization, realization and run packages import no SQL or driver types. SQLite gives one transaction boundary for events, correlation and budget reservations; no message broker, ORM or event-sourcing framework is needed.

## Physical store and commit contract

The store lives in a private state directory outside the governed repository and is bound to that repository's identity; a copied database does not authorize another checkout. Use WAL mode, full synchronization and foreign keys, and verify the effective settings ([synchronization](https://www.sqlite.org/pragma.html#pragma_synchronous), [WAL](https://www.sqlite.org/wal.html)). Pin and test the driver and embedded engine before adoption.

One controller per repository holds the execution lock; inspection uses bounded read snapshots. No SQL transaction stays open while an Agent or check runs. A busy, failed or uncertain commit prevents the dependent external effect. A lapsed lock alone never proves that an earlier controller or its child processes stopped; recovery reconciles known activity first.

The lifetime is explicit through one consumer-owned port:

```go
// Package apply. An Orchestrator controller session is distinct from an Agent session.
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

`ControlRequest` names the captured repository identity and the state directory. Acquisition takes an exclusive process-lifetime lock and binds a validated store with a new controller epoch; the binding is a lifecycle capability, not decision authority. `apply` revalidates captured inputs under the lock, creates `run` with the returned journal and closes control on exit. A failed acquisition grants no dispatch; a partial handle returned with an error is still closed. `Close` releases controller resources only; it does not assert that launched processes terminated.

The lock and the active store registration live in small repository control metadata under `.markitect/state/`, which Git ignores and coverage classifies as Markitect state. It is excluded from source staleness and from the clean-working-tree check, and an alternate `--state-dir` cannot bypass the lock or hide unresolved work. The first store binding uses checked creation; a missing or replaced registered database needs explicit reconciliation, not a fresh empty history.

Append requires the current controller binding. Publication permits carry that binding plus a run-scoped Orchestrator check, and the publisher verifies both at the write boundary; a live controller alone cannot authorize a finished run. CLI and MCP handlers never own this lifecycle.

| Stored relation | Essential columns and constraints |
| --- | --- |
| `store_meta` | Format version, bound repository and store identity, controller epoch. Unknown format or identity fails opening. |
| `stream_head` | The single current revision of the repository's stream. The first append expects revision zero. |
| `blobs` | SHA-256 of the exact bytes as key, length and bytes. Reads recompute the digest; inconsistent bytes are corruption. |
| `batches` | Unique batch ID; content digest, expected revision and committed sequence interval. An exact repeat returns the original receipt. |
| `events` | Sequence as key; globally unique event ID, batch ID, kind and version, assignment reference, optional run, task, session and step references, UTC observation time, causal predecessors and body blob reference. |

Append checks the expected revision, inserts the batch and events and advances the stream head in one transaction. Required blobs are already durable or inserted in the same transaction. After acknowledgement the whole batch is visible; no partial batch becomes a committed prefix. Events are never edited or deleted; corrections are new events.

Before rejecting a stale expected revision, look up the batch ID: an identical retry returns the earlier result, different bytes under the same ID fail. If a commit outcome is uncertain, query that batch before proceeding; an unreachable database is not evidence of absence. No database retry resends provider work.

Orchestrator records are versioned UTF-8 JSON with fixed field order, rejecting duplicate or unknown fields and oversize values. IDs are opaque strings, counters bounded nonnegative integers, timestamps UTC. Raw prompts and provider wire bytes keep their own byte identity. State is reduced from committed typed events; large blobs load on demand, and rebuilding a projection performs no external action. A read pins a committed revision and returns ordered pages. Timestamps never invent a causal order; the sequence and causal links do.

The first version retains every referenced payload, including rejected candidates. A backup or export must capture a consistent database including its WAL state.

## Stream and value contracts

A repository has at most one unfinished run, so one stream per repository is enough: a continuation run, its inherited budget and questions answered between runs commit atomically, and every event names its assignment. Run IDs remain mandatory on run-scoped events.

```go
// Package run. Bodies are closed, versioned event values, not arbitrary maps.
type AppendBatch struct {
	ID               BatchID
	ExpectedRevision uint64
	Events           []Event
}

type Event struct {
	ID         EventID
	Assignment AssignmentID
	Actor      ActorID
	Run      RunID
	Task     TaskID
	Session  SessionID
	Step     StepID
	At       time.Time
	CausedBy []EventID
	Body     EventBody
}
```

Optional identities use explicit absence in the codec. `EventBody` is a closed tagged contract with kind- and version-specific validation. `EventQuery` selects an optional assignment, run or batch and a sequence interval; its cursor pins a revision. The four-method journal port stays narrow; domain transitions belong to `run`.

| Record | Required data |
| --- | --- |
| Assignment | Owner input, goal, constraints, authority source, repository identity, caps and extensions, linked runs and closure. |
| Run binding | Model closure and lock, captured repository tree and census with its publication guard, configuration and prepared profiles, instructions, initial registry, operation contract and product build. |
| Retained work | Exact predecessor run and candidate, original base, matching registry, open findings and the reason for carrying it forward. Old approval is never promoted. |
| Task | Run, Manager, parent task, attempt, outcome, complete responsibility, candidate base, context package reference, constraints and predecessor link. |
| Supplied inputs | Exact prompt and context, step workspace version, the aggregate version it started from and own starting changes, execution-session/conversation lineage and later supplied replies. |
| Context package | Immutable rendered package including its mandate section, with source references, effective configuration, registry and task basis, and the product prompt template identity. |
| Question and proposal | Stable ID and version, originating task and run, affected work, model passages, decision and reason, alternatives, exact before and after source edits and the effect on work. |
| Answer | Question version, exact owner input, ingress and authority, selected disposition and any remaining ambiguity. |
| Candidate, review and check | Exact input, tree, registry and subject bindings, assessments, findings and observed results; carry-forward judgments for retained work. |
| Finding | Stable ID, issuing reviewer and session, responsibility, model and input basis, requested evidence, optional candidate, and later state changes with their evidence or grounds. Its state is a projection of history, not an editable flag. |
| Publication | Publication ID, run and captured source guard, verified candidate and registry, prepared request, completed paths and read-back or uncertainty. |
| Budget | Reservation ID, assignment and cap revision, budget class (ordinary Manager work or assessment), separately bound role/permitted effects, resource quantities, effect identity, maximum duration, settlement or unresolved hold. |

The table specifies target meaning, not a rewrite of persisted codecs. Existing `Writing`/`Assessment` literals retain their recorded version and accounting behavior until an explicit producer migration supplies the separated target contract. Historical records, original bytes and retry identities remain intact.

Proposed model edits are exact file bytes or identified source fragments with their prior content identities. A changed proposal gets a new version. A description of desired behavior alone does not authorize a concrete edit.

## Budget dispatch and lifecycle

`run` validates every transition against the [execution contract](manager-apply-execution.md). The expected revision prevents two controllers from committing from the same state. The budget reservation and the dispatch record are one committed batch before the external call. A prepared run consumes its run unit on commit. An uncertain invocation keeps its hold; if inspection proves that no dispatch happened, an explicit release restores the capacity. Settled consumption never decreases, and no extension or new run resets it.

| Level | Recorded meaning |
| --- | --- |
| Assignment | Open or closed, linked runs, caps and extensions. Activity labels derive from these facts. |
| Run | Exactly the states and outcomes of the execution contract. |
| Task | The states of the [core task machine](manager-apply-core.md#manager-tasks). Rejection counters are kept per run, Manager and review purpose. |
| Question | Open, answered, deferred or replaced; a stale proposal is replaced by a new version. Recording an answer does not mean its model effect happened. |

Illustrative event families and tags for the target workflow, each requiring a closed versioned payload codec. This table does not declare them implemented or redefine an existing source format version:

| Family | Event tags |
| --- | --- |
| Assignment and run | `assignment.created`, `assignment.extended`, `assignment.closed`, `run.prepared`, `run.state-changed`, `run.finished` |
| Task and delivery | `task.created`, `task.state-changed`, `attempt.started`, `delivery.accepted` |
| Budget | `budget.reserved`, `budget.settled`, `budget.released` |
| Execution and transport | `session.prepared`, `session.observed`, `step.prepared`, `transport.prepared`, `transport.observed`, `tool.requested`, `tool.replied` |
| Realization and evidence | `candidate.captured`, `assessment.recorded`, `review.recorded`, `finding.recorded`, `finding.state-changed`, `check.prepared`, `check.observed`, `composition.recorded` |
| Questions and intent | `question.proposed`, `question.state-changed`, `answer.received`, `answer.bound`, `model-change.observed` |
| Publication | `publication.prepared`, `publication.observed` |

`run.finished` records the single immutable outcome. A finding state changes only through `finding.state-changed` issued from an independent review with its evidence or grounds. No state is inferred from a message saying "done".

## Questions answers and model changes

1. The responsible Manager raises a source-grounded question addressed to the owner, which goes through the application ([M4](manager-apply-core.md#messages)). The Orchestrator records its version, proposal and affected tasks. Other ready work continues; required unresolved work prevents aggregate completion.
2. The application receives the owner's answer, records it raw and binds it to the displayed question version. Repeated ingress is deduplicated. An Agent-supplied actor name carries no owner authority.
3. The answer's effect is classified with a reason. Context about an existing rule can be supplied in a later step of the same run. Asking for an alternative to be elaborated authorizes preparation only. Deferral leaves dependent work waiting. A new obligation or authority change needs a new basis.
4. Accepting authorizes the exact displayed edit; the same permission is not asked again. A clear free-text answer can authorize a different edit; remaining material ambiguity gets a revised proposal. If the source drifted, the edit is withheld and revalidated, and the owner is asked again only if the choice materially changed.
5. Artifact Apply never edits its model or configuration. If the answer changes the captured run input basis, the run quiesces and finishes Incomplete with the reason "awaiting model change". In the first version the owner or the calling agent applies and commits the exact approved edit; Markitect records the actual result and new model identity separately from the answer and compares the observed model difference with the approved patch. A deviation is shown to the owner but not blocked, because the owner holds that authority. A pending or failed edit stays visible. An accepted artifact delivery under unchanged run inputs instead reaches later steps through the current aggregate ([D3](manager-apply-core.md#deliveries-and-composition)).
6. Once the edit is committed, the owner continues the assignment and the continuation below starts. "Model updated" and "work resumed" are each reported only after it is observed.

Notifications are projections of these facts, not a source of permission. Approved code cannot authorize an unauthorized business-policy change.

## Continuation and publication basis

A continuation starts only after its predecessor has finished; the controller lock serializes runs and publication. The continuation captures the new model, configuration and actual repository source. Validate the selected retained work for exact bases, ownership and registry, protected inputs and conflicts; incomplete work may be retained with its findings. Conflicts return to their accountable Manager; there is no semantic automatic merge.

The continuation's binding, the retained-work reference and the shared budget commit together; only then is its root dispatched. A crash leaves either no continuation or a complete one. The finished predecessor keeps its outcome; the continuation links to it.

| Value | Library meaning |
| --- | --- |
| Captured repository source | The actual checkout and registry, such as committed A1. It supplies the publication guard. |
| Retained work | An unpublished predecessor candidate and registry selected as starting material. It may be incomplete. |
| Current candidate | The continuation's assessed result after adapting that material. Reviews, checks and publication concern this. |

A private candidate never becomes the presumed content of the real checkout. Final publication compares against the continuation's acquired source, writes the verified difference and reads it back. Database transactions do not make multi-file publication atomic.

The continuation repeats all required reviews, checks and integration and the final evidence check. For retained work, the Manager records one judgment per affected area, reviewed like any other assessment:

| Judgment | Meaning |
| --- | --- |
| Keep | The realization still meets the current obligations; the reason is stated. |
| Adapt | Useful parts remain; the needed changes are named. |
| Replace | It relies on a discarded assumption; what replaces it is named. The old candidate stays in history. |
| Unresolved | Information or a decision is missing; what is needed is named. |

## Library traces

**Two model versions.** R1 binds M1, produces reviewed A1 and finishes Applied. The owner changes only `loan-policy.spec.maxOpenLoans` from five to three and starts a new Apply. R2 binds M2 and the actual A1; lending adapts admission, other areas can deliver justified no-change results, and all required assessments run before A2 is published. The [Playground](manager-apply-playground.md) owns the independent evidence.

**A missing decision during work.** A deliberately inconsistent future draft both preserves existing loans and requires all holdings to fit a lowered limit; this is not M1 or M2. Lending raises question Q1 version 1 with the conflicting passages and an exact correction that keeps the structured policy as the numeric authority. It goes to the owner through the application ([M4](manager-apply-core.md#messages)), and the owner accepts. The journal separates answer received, run finished Incomplete awaiting the model change, edit committed, model revision observed and continuation started. Lending's unfinished candidate is retained and assessed again; if no meaningful evidence fits the remaining budget, the run ends Incomplete.

**An inspectable timeline.**

| Observation | What it establishes |
| --- | --- |
| Unchanged operation propagated; exact lending context and message observations stored | Which direct edge supplied the operation, which Manager actually received it, and what remains pending. |
| Step prepared; send outcome observed or unknown | What may have executed; preparation is not delivery. |
| Candidate C1 captured; reviewer requests changes | Actual bytes and the grounded issue, such as a sixth loan being allowed. |
| C2 captured and locally reviewed; reminders integrated | Current local and child evidence, still short of final approval. |
| Q1 answered; model edit pending | An answer exists; intent has not changed yet. |
| Continuation prepared with retained C2 and current source | What was reused, which basis changed and what capacity remains. |
| Aggregate verified; publication and read-back observed | The exact machine-assessed result and the actual checkout. |

## Ownership and focused gates

`run` owns assignment, question and budget transitions and event meaning. `organization` renders configuration, context and mandates. `realization` owns artifact, registry and composition values. `apply` owns capture, continuation preparation and guarded publication. The journal adapter owns SQL, byte storage and atomic append and makes no semantic decision. CLI and MCP expose the same services and projections.

Gates: atomic batch and head updates, exact duplicates, conflicting and stale revisions, unknown-commit reconciliation, blob corruption, process failure around dispatch, refusal of answers to an old proposal version, a context-only answer resuming the same run, resuming an interrupted run, refusal of a finished run's unused publication permit, budget continuity across continuations and separation of retained work from captured source. Exercise the pinned driver on the supported filesystems; a format probe does not establish durability.
