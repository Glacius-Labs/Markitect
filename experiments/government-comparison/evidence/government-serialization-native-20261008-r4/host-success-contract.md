# Government R4 queue success and recovery contract

Date: 2026-10-08  
Scope: offline, source-grounded preflight criteria for one Government Queue followed, only after a positive completed queue, by one Resume/Replay readback of that same existing queue. The R4 grant authorizes this terminal idempotent readback; it does not authorize unfinished/interruption recovery. This records contract predicates, not evidence of an executed R4 result. No native CLI, controller, actor, delegate, provider, or test process was started for this assessment.

## Pins and source basis

Overseer pins for this assessment: Government source commit `04e225d5caee78c2a198607143863fca1e829750`; accepted binary SHA-256 `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`; corrected deterministic delegate SHA-256 `e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9`. These pin values are from the Overseer message; this document does not attest a newly executed binary.

The source files below are from the pinned Government commit. Blob IDs identify the exact inspected content.

| Source | Blob ID | Relevant lines |
| --- | --- | --- |
| `internal/host/cli/government.go` | `17d0a22d39c1cc019e5f6944e63ce06a41cc54a7` | 18-79 |
| `internal/host/government/execution/queue.go` | `7e1ef7f5a934093308a4dc5aae603b1579fbbcc9` | 16-87, 119-203, 270-407, 410-458 |
| `internal/host/government/execution/queue_input.go` | `924ebef88e9fad34d06940800ec52c3b6955a8b4` | 19-132, 207-238 |
| `internal/host/government/execution/queue_journal.go` | `3d255b9d904cb0af95a87de374ddea2608a83d04` | 42-82, 84-150, 154-238, 239-314 |
| `internal/host/government/execution/run.go` | `f8ca2ca796a9b4f81990dd10fcae2f88d042f790` | 40-72, 347-482, 505-521, 577-607 |
| `internal/host/government/execution/recursive.go` | `7ea0ace06d72cfdbc311433812dd3cc911669ff5` | 272-300 |
| `internal/host/verify.go` | `f00566e6935e8556cac6d5daa3c1eb7c2eb79ec0` | 28-37 |
| `internal/host/government/bindings.go` | `88b534871d42691d0ea626000c3869835d5ab5ac` | 118-227, 230-280 |
| `internal/host/government/promotion.go` | `ddfc778c688653cdc3fd522eabf18216b8c630ed` | 20-47, 65-192, 195-250 |
| `internal/host/government/promotion_recovery.go` | `feb7a3b86dd353808067b7002a61ea01a003b219` | 46-145 |

## One Queue invocation and success predicate

The native command is exactly the `government` subcommand with explicit repository, backlog, action, and write intent:

```text
"<pinned-markitect.exe>" government --repo "<absolute-repository>" --action queue --backlog "<absolute-external-backlog.json>" --write
```

The Queue CLI rejects `--config`, `--order`, `--runtime`, or `--queue` for this action, requires an absolute backlog path and `--write`, emits a JSON `QueueReport`, and returns process status 1 when its call returns an error (`cli/government.go:55-79`). A process exit code of zero only means that call returned no error; the scheduler can return a `blocked` or `incomplete` report without an error. Therefore exit code 0 is not a success predicate.

For the one-job R4 scope, evaluator success requires all of the following observed values and bindings:

1. The emitted QueueReport has the expected pinned queue API version, exactly one job with the preselected job ID, `status: "complete"`, `nextStep: "none"`, and no queue error. The job is `state: "accepted-scoped"`, `nextStep: "complete"`, with a nonempty `runId`, report path, and report digest. `scheduleQueue` sets the job accepted-scoped only when the native run report is `accepted-scoped` and `runErr == nil`; overall queue status is complete only when all jobs are terminal and none is blocked or incomplete (`queue.go:344-407`). Preserve the emitted report and verify the report digest against the referenced report bytes; do not infer success from stdout shape or process code alone.
2. The referenced run report has `status: "accepted-scoped"`, `stage: "complete"`, a nonempty candidate commit/tree, material candidate, evidence, acceptance decision, and promotion result whose status is `"promoted"`. The report must have the complete frozen-cabinet votes and evidence/decision links below. A report ending at execute, checks, independent review, ressort votes, or promote is not accepted completion (`run.go:347-482`).
3. Every configured fresh technical check completes without error against the materialized candidate. The report retains the resulting checks; `freshChecks` fences control between checks, checks remaining run budget, and returns an error on failed verification (`recursive.go:272-300`; `verify.go:28-37`; `run.go:385-395`).
4. Every integration review selected by the frozen plan is explicitly passed with no uncertainty, and includes exactly one passing observation for every required review scope. Missing, duplicate, rejected, or unexplained observations fail review (`run.go:399-414,505-521`). This is a separate predicate from the final Ressort vote.
5. Evidence is for the exact material candidate and a positive round, with valid content-derived identity over nonempty immutable result/report digests (`run.go:415-427`; `bindings.go:230-257`). The material candidate binds prior Constitution, repository tree, model, plan, check definitions, and tool pins (`bindings.go:20-44`).
6. Each member of the frozen cabinet, including the specifically selected Ressort, supplies one final vote bound to that exact material candidate ID, evidence ID, and evidence round. The vote role response must have outcome `passed`, no uncertainty, and exactly one observation with subject `government-vote`, observation outcome `passed`, and a valid vote-body JSON detail. The body has `outcome`, `reason`, `materialCandidateId`, `evidenceId`, and `round`; it must not carry unknown fields. Body outcome may be `assent` or `assent-unaffected` for acceptance; `objection` or `incomplete` prevents the acceptance decision. The host binds the vote to the frozen Ressort, prior Mandate and mandate digest, configured slot, plus the vote actor receipt run ID (`run.go:577-607`; `bindings.go:118-146,260-280`).
7. The acceptance decision contains exactly one unique, valid vote from every frozen cabinet member, all votes are positive (`assent` or `assent-unaffected`), and its candidate/evidence/round/prior-authority digest match the same material and evidence (`bindings.go:149-227`). For a one-member frozen cabinet this means exactly that member's one final positive vote; do not choose or substitute a Ressort after seeing the result.
8. Promotion binds the managed active ref, runtime expected-old commit, candidate commit and tree, material candidate ID, evidence ID, decision ID, report state directory, and run ID idempotency key. Success requires no error, `Promotion.Status == "promoted"`, active-ref readback equal to the candidate commit, and the durable completion record (`run.go:457-482`; `promotion.go:20-47,107-192`). A stale-base or incomplete promotion is not success.

The single-job finite backlog must be an external UTF-8 JSON file with `apiVersion: "markitect.government-queue/v1alpha1"`, normalized absolute existing external `stateDirectory`, finite `limits`, and an explicit `jobs` array containing exactly the planned job. The job requires `id`, repository-relative safe `configPath` and `orderPath`, absolute external `runtimePath`, and explicit `dependsOn` array. The parser rejects missing/null required values, unknown fields, duplicate keys, invalid or duplicate job IDs, unsafe paths, missing dependencies, cycles, and out-of-bound limits (`queue_input.go:19-132`). Queue state must be outside the repository and Git metadata; backlog/runtime files must also be external (`queue_input.go:135-204`). Queue freezes raw backlog bytes, runtime bytes/fingerprint/tool pins, repository identity, limits, and job bindings (`queue.go:119-142,205-267`).

## R4 terminal Resume/Replay verification and general recovery boundary

There is no separate `replay` CLI action in this held command contract. Resume is the only queue continuation action; it reconstructs state from the existing hash-chained `events.jsonl` journal. Its exact CLI form is:

```text
"<pinned-markitect.exe>" government --repo "<absolute-repository>" --action resume --backlog "<same-absolute-backlog.json>" --queue "<absolute-existing-queue-directory>" --write
```

For this R4 sequence, issue this command exactly once only after Queue has satisfied the positive success predicates above. Reuse its exact backlog path and the absolute `queueDirectory` from that QueueReport. This is a terminal readback check, not a retry or recovery attempt. Resume requires the same backlog bytes, Git repository identity, frozen job bindings, exact runtime bytes/fingerprint, and tool pins; mismatch marks the queue stale/blocked (`cli/government.go:55-69`; `queue.go:145-203`; `queue_input.go:207-238`). Journal loading checks sequence, previous digest, event digest, event ordering, and terminal report counters/bindings (`queue_journal.go:84-150,208-238`).

The R4 Resume/Replay verifier must confirm that the report returned by Resume is the same terminal report as Queue: queue ID/directory, backlog digest, limits, exactly the same accepted-scoped job identity/run ID/report path/report digest, status `complete`, terminal journal sequence/digest, and usage/start/repair counters are preserved. Confirm after readback that the native active ref still names the promoted candidate and that actor-start and role-ledger counts have not increased and no new role process was created. These are evaluator observations over the before/after evidence; the terminal Resume implementation itself reconstructs from the journal and returns the persisted report without running the scheduler.

Resume is recovery, not actor replay. An interrupted `running` job is eligible for promotion-only recovery only when its durable checkpoint is at `promote`/`amendment-promote` (or complete with promoted result) and candidate, evidence, decision, IDs, candidate commit/tree, active ref, base, runtime pin, and tool pins all match (`queue.go:410-424`). `RecoverPromotion` requires the exact stored promotion intent and exact tokenized ref-history edge; it never retries compare-and-swap (`promotion_recovery.go:46-145`). If that checkpoint/binding does not qualify, recovery marks the job incomplete and explicitly records that actor effects were not replayed (`queue.go:443-458`). Scheduling skips accepted-scoped, blocked, and incomplete jobs (`queue.go:270-280`); it can only continue other eligible queued jobs in an unfinished queue.

After a `queue-finished` journal event, replay sets `TerminalStop`. `ResumeQueue` then returns the persisted report instead of scheduling (`queue_journal.go:208-238`; `queue.go:195-202`). The scheduler also sets terminal stop after writing its queue-finished record for every finished attempt, including incomplete attempts (`queue.go:373-407`). Thus the positive R4 Resume is intended to verify this terminal no-relaunch behavior after a successful complete Queue. An incomplete or blocked Queue does not satisfy the R4 precondition; do not Resume it in this task.

The source also supports a distinct general interruption-recovery path, outside this R4 grant: an unfinished queue may attempt promotion-only recovery for an exact matching promote checkpoint; otherwise a running job becomes incomplete and actor effects are expressly not replayed (`queue.go:410-458`). `RecoverPromotion` requires an exact stored intent and tokenized ref-history edge and never retries compare-and-swap (`promotion_recovery.go:46-145`). Do not invoke this interruption path for R4; only the positive completed-queue readback is authorized here.

## Evidence boundary

These are source-level acceptance and recovery predicates for the supplied pins. They do not claim that the R4 Queue was run, that an OS return code establishes product success, or that a positive technical/run report establishes semantic quality or human acceptance. Any later product result must be evaluated from the actual bound QueueReport, journal, referenced run report and digest, actor receipts, vote, decision, promotion intent/completion, and active-ref readback.

