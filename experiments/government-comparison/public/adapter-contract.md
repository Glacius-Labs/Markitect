# Public study adapter contract v1.1

Owner: Scientist. Product owners provide adapters; Scientist validates and freezes them.
This is an interchange contract, not a claim that any product currently supports it.
The source baseline is Classic `1ea5c76f55526fc4d721e865885436153f48b497`;
study product versions remain unset until readiness inspection.

## Invocation

An adapter is an explicit argv array, invoked without a shell in the actor's repository:
`ADAPTER --request ABSOLUTE_REQUEST_JSON --result ABSOLUTE_RESULT_JSON`.
`--request` and `--result` are appended by the harness. The adapter must terminate
within the request deadline, write a UTF-8 JSON result even on ordinary failure,
and preserve raw receipts/logs under `evidenceDirectory`. Exit 0 means the request
was handled; it is not task acceptance. A missing executable or capability returns
`readiness_gap`, never a fabricated implementation. Exit nonzero is infrastructure
failure and remains recorded. An adapter may wrap the actual product CLI; it must
publish the exact wrapped argv and versions. No shell command strings or implicit cwd.

Operations: `probe` (read-only availability/version/capabilities), `run_task` (one
released task on the same trial), `resume` (same request identity and persisted
run), `stop` (bounded shutdown). Only `probe` is required for this package's fixture
smokes. Stateful product support is reported explicitly rather than simulated.

## Request JSON

Required fields:

```json
{
  "schemaVersion": 1,
  "operation": "probe",
  "mode": "fixture",
  "trialId": "conventional-greenfield-smoke",
  "arm": "conventional",
  "condition": "greenfield",
  "actorRepository": "ABSOLUTE_PATH",
  "baseCommit": "FULL_GIT_SHA",
  "task": null,
  "releasedInputs": [{"path":"ABSOLUTE_PATH", "sha256":"HEX"}],
  "product": null,
  "profile": {"model": null,"reasoning": null,"runnerVersion": null},
  "limits": {"wallSeconds": 30,"maxActorCalls": 0,"maxParallelActors": 1},
  "evidenceDirectory": "ABSOLUTE_PATH"
}
```

Arms: `conventional`, `classic`, `government`. Conditions: `greenfield`,
`brownfield`. Mode `fixture` can never establish a live study result. A live
request must bind the frozen protocol digest, exact runner/model configuration,
deadline, tool rights, product source/binary/module/config digests where relevant,
and the common aggregate resource ledger. `task` identifies only the currently
released card with its digest. No private paths, holdout, future cards, other cell
results, prefabricated canonical project model or study source repository are
provided to actors. Public brief/checks are outside the initially empty Greenfield
Git repository. Modeling, discovery and setup are actor work and charged to trial.

## Result JSON

```json
{
  "schemaVersion":1,
  "trialId":"conventional-greenfield-smoke",
  "requestSha256":"SHA256_OF_EXACT_REQUEST_FILE_BYTES",
  "operation":"probe",
  "mode":"fixture",
  "status":"readiness_gap",
  "candidateCommit":null,
  "capabilities":[],
  "gaps":["No actual actor runner configured"],
  "receipts":[],
  "usage":null
}
```

The result must echo `requestSha256`, the SHA-256 of the exact request file bytes,
including resolved evidence directory, released inputs, product pins and limits.
Matching trial/operation alone cannot establish freshness within a trial. This
required field was added in v1.1 after the early v1 publication.

Statuses: `ready`, `completed`, `incomplete`, `blocked`, `readiness_gap`, `failed`,
`stopped`. `ready` only answers `probe`; `completed` only means adapter work ended.
Scientist independently checks a clean, full candidate SHA and evaluates it after
freeze. It does not accept self-scoring. Product-specific evidence is attached as
receipts `{path,sha256,kind}`; Classic supplies actual model/context/impact,
Executor/Verifier/parent/final-audit evidence. Government additionally supplies
mandate and round identity, selected ministries, explicit final-candidate votes,
promotion/refusal and recovery evidence. Missing records are gaps, not passes.

Usage preserves provider raw counters, their semantics, receipt digests and billing
provenance. Unknown tokens/cost are `null`. Cache/reasoning counters are never
blindly added to input/output. Every attempt, review, subagent, failed call, retry,
setup, initial model and upkeep consumes the same trial ledger. Product adapters
must expose their required supervision and cannot conceal orchestration effort.

## Product-owner handoff

Supply: immutable source/binary/module/config versions; invocation argv and
prerequisites; supported operations and genuine limitations; runnable public
setup/task smoke; output/receipt paths and semantics; interruption/recovery
behavior; mandatory interventions and actual access boundaries; metering route.
Worker/Architect receive this public contract only. Private assessment stays with
Scientist. A documented readiness gap permits independent study preparation to
continue; it does not permit replacing a live arm with fixture responses.
