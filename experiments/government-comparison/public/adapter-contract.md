# Public study adapter contract v1.2

Owner: Scientist. Scientist implements, maintains, validates and freezes the common
study adapter wrapper: Request/Result translation, exact `requestSha256` binding,
metrics, common resource limits and trial invocation. Worker/Architect supply their
native product invocation paths, supported capabilities/gaps, immutable pins and
actual receipts. This handoff bounds their product-readiness work; Scientist maps
it into the study interchange below.
This is an interchange contract, not a claim that any product currently supports it.
The finite S1 dispatch extension is specified in [s1-dispatch.md](s1-dispatch.md):
operator-pinned grant/protocol authority, counted reservation/process/usage/Result,
conservative recovery and separate mechanical/live gates. It preserves this
fixture probe schema and does not authorize inference or product readiness.
The historical preparation source baseline is `1ea5c76f55526fc4d721e865885436153f48b497`.
Subsequent inspected product identities belong to `runtime/classic-pin.json` and
the runner readiness checkpoint; inspection does not clear the live-study gate.

## Invocation

The Scientist-owned study adapter is an explicit argv array, invoked without a shell in the actor's repository:
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

The native readiness wrapper additionally requires `profile.runnerExecutable`
(absolute pinned Codex executable) for a Conventional version probe and
`product.packetPath` (absolute frozen packet directory) for a Classic version
probe. These are operator inputs; they are never passed to an Actor. The exact
request digest binds them. Missing fields return a capability gap. An actual
Actor call requires the explicit finite S1 grant/protocol gates; mode alone cannot enable it.

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
setup, initial model and upkeep consumes the same trial ledger. Scientist's wrapper
records required product supervision from the native product handoff and preserves
all orchestration effort.

## Product-owner handoff

Supply: immutable source/binary/module/config versions; invocation argv and
prerequisites; supported operations and genuine limitations; runnable public
setup/task smoke; output/receipt paths and semantics; interruption/recovery
behavior; mandatory interventions and actual access boundaries; metering route.
These are native product calls and evidence from Worker/Architect. Scientist owns
their Request/Result adaptation, study-level metrics/limit enforcement and shared
trial shell. A native product command need not accept `--request`/`--result`:
Scientist wraps its documented argv and receipts. Product owners retain ownership
of product defects; study-wrapper defects remain Scientist's responsibility.
Worker/Architect receive this public contract only. Private assessment stays with
Scientist. A documented readiness gap permits independent study preparation to
continue; it does not permit replacing a live arm with fixture responses.
