# Autonomous A/B gauntlet

This report is in progress. Development trials are still running; no holdout outcome, overall winner or completed experiment is claimed. The [vision](../vision.md) supplies the product question and [measurement](../measurement.md) supplies the evidence limits. Private Konfyra candidates remain outside this experiment and have not been accepted or adopted.

## Fixed candidate and experiment

The tested source candidate is `adef79d935399f8ac63ad874dbdeab8d15c418a1`, built with Go 1.27.1 for Windows amd64, with embedded version 0.13.0 and an unmodified VCS build. Its CLI SHA-256 is `553b7077ce7b5db2df89567146fc1a877a03cb6edd75459be5a3e0a380676056`; the artifact checker is `e671a89e03c93c153dcbe8694b9c30f86cc0e9512b1dc026332ac24db5e1c9d8` and the module checker is `af14fa820f9c145ff44d1bee3129e15bb35ec3b7aeb26334d65cbff25f0702e1`. These are locally frozen candidate binaries. They are not asserted to be the published release assets.

The independently verified published release remains v0.13.0, targeting `ec312e35c15012c07acb6838c03e9316c5373dde`, published on 2026-10-04. This experiment has not published another release.

The [experiment design](../../experiments/autonomous-ab-gauntlet/design.md), [operator contract](../../experiments/autonomous-ab-gauntlet/operator-contract.md) and [protocol](../../experiments/autonomous-ab-gauntlet/protocol.yaml) define three synthetic paired projects: a Go modular service, a .NET Vertical Slice/DDD application, and an engineering operations repository. Each has 12 planned main tasks and three predeclared trials per arm: 216 planned tasks, consisting of 144 development tasks 01-08 and 72 deferred holdout tasks 09-12. Those are planned denominators, not completed observations. No 15-task trajectory was designed or measured.

Knowledge parity is explicit for [the service](../../experiments/autonomous-ab-gauntlet/projects/modular-service/parity.md), [DDD](../../experiments/autonomous-ab-gauntlet/projects/vertical-slices/parity.md) and [operations](../../experiments/autonomous-ab-gauntlet/projects/engineering-ops/parity.md). A has conventional owner guidance, architecture checks, tests and CI. B has the same intended requirements through Markitect's current model/projections/checks. Mechanism differences remain visible rather than being disguised as equal command counts.

Fresh native actors inherit gpt-6.1-sol/xhigh, use no parent conversation turns, and receive only their current arm/task envelope. Each turn has one complete public-helper invocation and at most two fresh repairs from exact public failure output. The 1,800-second task deadline includes preparation, dispatch queues, tool waits, validation and repairs. Actors commit before fixed-revision checks. Neither root nor operators supply semantic corrections or hidden outcomes to scored actors. Requested read/mutation boundaries are not an OS sandbox proof.

The original [304-input freeze](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/freeze.yaml) has digest `4faee78795986559e91a0e233bd7321a9cf48526d331edba621ca98eee11638a`. The frozen harness binary is `0fa0d12f29aa7746f1e74c6435fec08b63324919f1d01d7eb4046103be4d2ec1`, built from `af2f2d7369e2b86a89b735a34e62bda0d2a5a4ae`. The freeze manifest binds every seed input byte. The [source record](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/source-identities.yaml) binds product/harness builds; native task records separately bind the actual generated seed and task-base Git revisions. External arenas retain raw turns, receipts, captured Git state and ledgers. Public records contain sanitized tooling/fixture evidence, not private adopter source.

## Exclusions and unavailable evidence

| Evidence boundary | Treatment |
|---|---|
| Earlier setup cohorts with unusable paths/freeze inputs | Retained and invalidated; not scored as product outcomes. |
| Original DDD trial1 controller early-finish asymmetry | Both original arms excluded. A single same-freeze paired trial1 rerun occupies a separately identified arena. |
| Original service oracle required formatting/unasked wording | Both arms excluded. The [corrected cohort](../../experiments/autonomous-ab-gauntlet/results/r1-service-oracle-v1/freeze-review.yaml) has a new identity; behavior requirements are preserved and oracle QA is mandatory on Windows/Linux. |
| Operations trial2 reported helper outside its workspace | [Both arms excluded](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/runtime-run-exclusions.yaml); actual content-read telemetry is unavailable. No inference about which hidden bytes were read. |
| Fresh DDD trial2 controller PATH transcription | [Both arms excluded](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3-ddd-t1-control-rerun/runtime-run-exclusions.yaml). Submitted-message transcription is retained; exact process environment/native message byte export are unavailable. |
| Corrected service trial1 B04 dispatch exhausted capacity/deadline | No actor or helper started. No response is fabricated. The downstream chain and B parallel fork are unavailable; this is not a functional product failure. |
| DDD initial B06 captured check bytes differ from ledger | [Initial evidence remains unavailable](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/ddd-initial-capture-control.yaml). Final B06 is intact and independently assessed. |
| DDD raw Windows-path YAML serialization defect | [Explicit pinned projection](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/ddd-serialization-control.yaml) preserves all statuses and manual findings. [28 exports](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/ddd-serialization-validation.yaml) are bound to raw and projected bytes; originals stay unchanged. |

A temporary [capacity control](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/capacity-control.yaml) limits new dispatch to one scored leaf per operator, symmetrically, without interrupting active actors or resetting deadlines. Its [timestamp erratum](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3/capacity-control-timestamp-erratum.yaml) explicitly disqualifies an approximate manually supplied timestamp as telemetry. Queue delay is retained as workflow delay, never human attention.

The coordinator could inspect card definitions, including the reserved subset; holdout outcomes and actor runs remain withheld until the explicit fix/no-fix decision. Consequently this is an outcome holdout, not a claim that task definitions were inaccessible to the coordinator.

## Independent DDD development checkpoint

The [trial1 checkpoint](../../experiments/autonomous-ab-gauntlet/results/r1-paths-v3-ddd-t1-control-rerun/trial1-independent-assessment.yaml) binds separate arm assessments, corrections, source APIs and probe evidence. These are automated assessments, not project-owner acceptance. They do not supply feedback to later actors.

| Dimension, trial1 only | A | B | Interpretation limit |
|---|---|---|---|
| Native completed / planned tasks | 6 / 8 | 3 / 8 | Protocol completion alone does not establish hidden correctness. |
| Protocol-invalid tasks | 2 | 5 | Later passing probes do not rescue an invalid turn. |
| Fresh repair turns | 2 | 2 | This is one trial, not a general repair advantage. |
| Frozen behavior vectors | 20 reported passed; 2 unavailable | All 22 frozen vectors have passing evidence | A's task06 composed dispatch and task08 accumulated contract remain unproven. B's extra repeated report row is not a new requirement. |
| Architecture/model ownership | Combined task-level boundary/ownership rows pass | Missing Validator ownership at06, missing Query/Handler ownership at07, stale Handler identity at08 | Raw row counts differ in grouping and cannot be compared as a score. |

A's standalone Validator rejection and aggregate defense are component evidence. They do not prove an executed application dispatch guarantees validation before handler work. The corrected A report preserves this unknown instead of counting it as passed. B's actual Handler probes pass, but declaring new implementation paths as Skill inputs does not make them canonical architectural owners. This is visible model/workflow drift despite passing graph/artifact commands, not evidence that Core must infer source semantics.

An earlier A clock description and a B report backslash were corrected in separate records, preserving originals. A reported task07 B path violation was disproved by checking its exact native task06 base rather than task05. None of these corrections rescored a frozen expectation.

## What is not yet established

Final task-by-task outcomes, architecture/churn time series, escalation quality, context/impact set comparisons, complete repeated-trial results, and holdout/generalization results await terminal evidence and final analysis. No initial checkpoint is an overall winner.

Actual token use, backend model/tool-call counts, complete file-read audit and human attention are unavailable. Bytes and elapsed workflow intervals will not be converted into those quantities. Model/projection changes are upkeep evidence, not proof that a human synchronization step disappeared. Synthetic fixtures cannot establish production reliability, market demand, productivity or human-attention reduction. The real owner-reviewed adopter experiment remains necessary.
