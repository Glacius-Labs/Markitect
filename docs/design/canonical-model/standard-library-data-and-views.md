# Web Library: view, data-access and recovery contracts

Date: 17 October 2026. Detailed design proposal commissioned by the owner after the [logical-data/C4 walkthrough](standard-library.md#logical-data-operating-resources-and-c4-walkthrough). The [discussion record](standard-library.md#source-and-selected-decisions) owns selections and provenance; this document owns the detailed recommendations for the open subjects. The owner requested that they be worked out, not that every recommendation be treated as accepted.

This is a prospective web Library, separate from the existing local, sequential, process-lifetime-only M1/M2 fixture. It changes no canonical sources, compiler, engine, runtime or release. The eight selected base Kinds and their empty specs remain unchanged. Any proposed database choice is for this reference example, not a universal standard-library requirement.

## Design basis and intended result

The preceding example revealed a concrete gap: a load balancer and two API replicas do not establish a shared, consistent lending state. A database selection alone also does not specify transaction boundaries, ambiguous failures, safe schema changes or recoverability. These contracts close the design gap by stating observable obligations and retained freedom.

The first implementation should be able to produce reviewed C4 views, process concurrent requests without violating existing lending obligations, distinguish failure outcomes honestly, evolve existing data without inventing deletion rights, and demonstrate a restore in an isolated target. This document supplies proposed acceptance observations, not executed proof.

## 1. C4 views of one model

Recommendation: the first C4 implementation creates model-derived, independently reviewed views. It does not require a fully deterministic semantic renderer or duplicate C4 Definitions of each project object.

| View obligation | Scope and included subjects |
| --- | --- |
| Library container view | The Library boundary, browser/API applications and logical store, plus explicitly modeled relevant external interactions. |
| API component view | The API boundary, lending responsibility and its declared internal/data-access contracts and relevant collaborators. |
| Production deployment view | Production, intended API/store occurrences, hosting/service resources and load-balancing relationships. |

Author these presentation obligations as ordinary Rule Definitions and assess them with Check Definitions. A new universal View Kind is not required for the first implementation. A view obligation selects purpose, scope and abstraction; it does not redeclare business or architecture facts already owned by source Definitions.

Each generated diagram must retain the exact source model binding and a way to trace included nodes/edges to their owning Definitions and statements, for example concise comments in the artifact. This is derived provenance, not a second editable relationship catalogue. Layout and abbreviated labels may vary while identities and meaning remain correct.

The generating agent/tool reads applicable bodies and links. Review must check that an arrow means what its supporting sentence says, that required subjects/relationships are present, and that unsupported relationships are absent. In particular, a negative reference does not become a positive calls edge; reachability does not become containment; a database engine host does not become the logical database; and a deployment occurrence does not create another logical application.

Deterministic checks may validate source identity, links, syntax and any declared machine-readable facts. They do not prove interpretation of prose or full C4 conformance. The [C4 mapping](standard-library.md#c4-mapping-without-duplicate-project-objects) remains the interpretation to assess. A future fully deterministic renderer needs a separate sufficient relationship contract, justified by that consumer. No such schema, parser or Property is introduced here.

## 2. Data access and the admission boundary

Recommendation: expose a data-access contract to the lending responsibility. Business interpretation stays with the existing UseCases/Rules; the provider supplies the transaction and observation guarantees. Do not expose only a separate can-borrow check followed by an uncoordinated insert.

An illustrative Definition in Package library:

```yaml
imports:
  - package: markitect/software
    as: Software
id: library-data-access
kind: Software.Interface
body: >-
  Provides transaction boundaries and read-only inspection of [[library-data]]
  for [[library-api]]. Domain decisions remain with the responsibilities
  realizing the applicable UseCases.
  A state-changing operation evaluates its conditions and commits its changes
  as one coordinated operation under [[atomic-admission]].
  Success is returned only after a confirmed durable commit.
  Known aborts leave no partial domain changes.
  An uncertain commit outcome is reported under [[failure-outcomes]].
  Subsequent inspections observe completed changes regardless of the API
  instance serving them, and each inspection uses a coherent committed state.
spec: {}
```

This is an illustrative interface contract, not a complete compiled fixture or a selected programming-language signature. Exact consumer-owned methods, result types and query shape are implementation design within its meaning. Canonical intent must be extended if an implementation needs new observable behavior.

The proposed atomic-admission Rule coordinates the existing membership, availability and lending-policy conditions with their resulting changes across every API instance. The entire operation must admit a valid serial interpretation, including relevant concurrent membership/status changes and returns. No success or other externally effective action may escape a transaction that is later retried or aborted. ID format remains free; sequence gaps are not a promise violation in themselves.

Read-only inspection must be internally coherent and must not use an unrelated stale replica/cache after a completed operation. This does not require every command to contend on one global application lock. The implementation may choose a concurrency mechanism that satisfies the observable contract.

## 3. Allowed technology and the reference database

Reusable standard-library contract: require the actual consistency, integrity, durability, migration and recovery capabilities; do not hardcode a vendor into generic DataStore or Interface meaning.

Reference web Library recommendation: begin with PostgreSQL and one authoritative writable service, with serializable transactions for relevant state-changing operations and coherent reads from that authority. This gives a concrete first realization without implementing several database providers. It is a proposal, not an owner-selected database or a claim of high availability. Other engines require their own conformance evidence against the same contract rather than a vendor-name equivalence claim.

PostgreSQL's [transaction isolation documentation](https://www.postgresql.org/docs/current/transaction-iso.html) describes Serializable as equivalent serial execution for committed transactions and requires applications to handle serialization failures. This supports the reference approach; it does not prove that a particular application correctly encloses all relevant decisions and writes.

Pin the actual tested engine/client versions and relevant runtime settings when implementing the reference. An image-based deployment also pins the immutable image; a managed service records its corresponding tested configuration. Neither latest labels nor the current documentation URL are an executable dependency pin. Do not silently switch durability off: PostgreSQL documents [non-durable settings](https://www.postgresql.org/docs/current/non-durability.html) as tradeoffs that can lose data.

For this proposal, acknowledged commits must survive an API replacement and a normal database-process restart with the retained storage. Loss of the storage/service or a failover is a different recovery question and receives no unselected zero-loss promise. A database configuration, migration version or recovery state that does not satisfy the selected contract must not make the application ready for writes. An unavailable store must not cause an empty substitute library to start.

## 4. Outcomes, retry and unknown commit

Use the following distinctions at the data-access boundary and preserve them through the prospective HTTP contract. These are proposed web-Library outcomes, not changes to the existing JSON Lines fixture's response schema.

| Situation | Required meaning and behavior |
| --- | --- |
| Domain refusal | An actual applicable business condition prevents the operation. No partial domain change; report that condition without inventing one. |
| Technical failure with confirmed non-application | The operation did not commit. Report a technical failure, retain the reason and leave domain state unchanged. It is not a domain refusal. |
| Confirmed success | The complete operation durably committed; report its actual result. |
| Outcome unknown | The request may have committed, for example if the connection is lost during commit without a conclusive outcome. Do not promise unchanged state, return invented success, or automatically replay the mutation. |

Internal transaction retry is permitted only after a confirmed abort, for the complete decision-and-write unit, without an already released response or non-transactional external effect. Use finite configured attempt/time bounds. Exhaustion returns a technical outcome, not a fictional reached-loan-limit refusal.

PostgreSQL's [failure-handling documentation](https://www.postgresql.org/docs/current/mvcc-serialization-failure-handling.html) identifies serialization failures as SQLSTATE 40001 and requires retrying the complete transaction, including its decision logic. Other error classes must be classified deliberately; a blanket retry of every exception is not this contract.

An HTTP/client/load-balancer retry is a separate operation boundary. It is not made safe by an internal database retry policy. Correlation IDs remain correlation, not an implicit idempotency key. Existing state inspection can show current records but cannot always identify the exact outcome of an unknown request amid concurrent work.

First-scope recommendation: no automatic replay after an unknown commit. If transparent retry or exact result lookup is required, explicitly add a request-identity/idempotency contract with an outcome record committed atomically with the operation, collision rules, scope and retention. This is a possible later extension, not an implicit guarantee. A safe unknown outcome is an honest limitation; it is not proof of no effect.

## 5. Schema evolution and rollout

Reference recommendation: start with planned maintenance for schema changes, rather than promising mixed-version rolling compatibility without a requirement. The following describes a proposed procedure; this chat does not execute it.

1. Bind the intended schema/migration sources, expected starting state, target application and relevant model basis. Previously applied migration contents are immutable; check actual recorded revision/content, including unexpected newer or altered state.
2. For existing data, establish a recoverable pre-change checkpoint. Stop admission of new writes on every API instance, drain admitted work and acquire exclusive migration execution. Two application startups cannot independently race the same migration.
3. Apply reviewed migration steps with a durable record of actual completion. Restrict the initial reference implementation to transaction-compatible changes and commit each step's schema/data changes and applied-step record together. After a lost connection, inspect the actual schema/history before deciding what remains to apply. A non-transactional step needs its own interruption/recovery design before use.
4. Verify the resulting schema and retained domain data. Preserve stable IDs, links, dates, closed loans and existing open loans. A lower admission limit does not authorize deletion or forced return.
5. Start the schema-compatible application revision and verify required observations before reopening writes. A partial or failed migration retains its actual state and refuses write readiness; it does not reset, blindly replay or fabricate completion.

Do not automatically reverse a migration. An application rollback must first be compatible with the actual schema/data; otherwise use an explicitly reviewed forward repair or a recovery plan with its visible recovery point and potential loss. A restore cannot be advertised as a lossless undo of all later activity.

This procedure provides a simple first compatibility boundary. If uninterrupted service later becomes a requirement, model the allowed old/new combinations and transition invariants and add distinguishing checks. Do not assume the initial maintenance procedure already provides them.

## 6. Backup and recovery

Reusable requirement: recovery needs a usable consistency point, sufficient contents/configuration to restore, protection appropriate to the data, and a demonstrated compatible restore. A backup file or successful exit code alone does not establish these properties.

For an isolated reference restore exercise, a consistent PostgreSQL logical export is a useful initial candidate. PostgreSQL's [pg_dump documentation](https://www.postgresql.org/docs/current/app-pgdump.html) states that exports can be consistent during concurrent use, but that the tool covers one database and omits cluster-wide objects. Do not silently assume it contains roles, service configuration or all dependencies needed for operation. The same documentation cautions against treating it as a universal regular production-backup method. The broader [backup documentation](https://www.postgresql.org/docs/current/backup.html) distinguishes logical exports, filesystem backups and continuous archiving.

The reference exercise should bind the backup contents/integrity and known consistency point, schema/migration revision, required engine/configuration and associated model/application basis. Missing observations stay missing; do not invent a compatible revision. Separately account for required roles, privileges, extensions and configuration without copying live secret values into the model or diagrams. Store a recovery copy separately from the failure boundary it is intended to survive.

Restore into an isolated target, not over the current operational store during a verification exercise. Check IDs, references, dates, open/closed state and derived availability, then exercise representative supported operations with a compatible application. Existing over-limit loans must remain returnable after a deliberately lower policy. Record the actual recovered consistency point and observed restore duration. If migration to a newer target is part of recovery, assess it explicitly rather than claim the old backup already has the new schema.

Do not select numeric recovery loss/time objectives, retention or schedule solely because they are common defaults. The project must state acceptable data loss, recovery time, protected failure cases and retention purpose. Choose the production backup/archive schedule and technique against those requirements and actual evidence. A demo restore supports a bounded recoverability claim; it establishes neither zero loss, automatic failover nor a production recovery target. These project-specific decisions do not block defining the general library vocabulary.

## 7. Small distinguishing acceptance set

The following are proposed Scenario/Check subjects, not an executed test plan or a second requirements catalogue. Each must link to the actual canonical owner once those Definitions are authored. Expand only when a concrete uncovered obligation requires it.

| Subject | Observation that distinguishes a correct realization |
| --- | --- |
| C4 views | Correct subjects, boundaries and supported edge meanings at the exact selected model revision; no duplicate project identities or inferred unsupported arrows. |
| Two instances, one available copy | Normal concurrent requests produce one committed loan and one legitimate refusal, with one open loan and unavailable copy. |
| Two instances, last free member slot | Different-copy requests still preserve the member's actual model-bound limit. |
| Definite abort versus unknown commit | A known abort leaves no partial domain state; an ambiguous commit does not become unchanged-state evidence or blind replay. Fault observations must actually establish which boundary occurred. |
| API/database-process restart | Acknowledged changes remain under the stated retained-storage scope; no empty fallback. |
| Migration with retained history | Compatible target preserves old IDs/dates/open and closed loans and refuses readiness after incomplete work. |
| Isolated restore | An actual recovered consistency point yields coherent data and representative correct operations under a bound compatible application. |

Source/unit checks, actual concurrent database execution, provider/runtime checks, C4 semantic review and recovery execution are separate evidence classes. None is supplied by the documentation link checker or structural self-model check.

## What this adds to the library design

Most obligations use the selected Interface, Rule, Scenario and Check Kinds. Example Definition subjects are library-data-access, atomic-admission, failure-outcomes, schema-evolution, recovery-policy and c4-view-conformance. Actual tests, diagrams and migration sources realize them and receive registry assignments when created. Do not add one universal Kind for each procedure merely because the example needs a named obligation.

The [first companion slice](standard-library-architecture-platform.md) now owns the selected DataStore, Deployment and OperatingResource contracts and their Package assignments, all with empty specs and body links. The detailed Web Library data/operation recommendations in this document remain separate proposals. Additional fields should follow a demonstrated consumer. The proposed first C4 path needs no new structured relationship schema. A later deterministic renderer or exact engine/replica configuration validator can justify a separate Property proposal.

The design recommendation is concrete enough to review and implement the reference contracts once selected. Database/provider code, live concurrency, migration and restore exercises, numeric operating goals and production readiness remain distinct work. No database, backup or deployment operation was performed in this task.
