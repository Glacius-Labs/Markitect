# Capability proof checkpoint results

Date: 2026-10-06. Result: **C5 FAIL for targeted acquisition; proof phase PAUSED at the owner's stop gate.** The witness harness passes because it reproduces that failure. No product feature was added or changed.

## Claim

A local canonical change should avoid unnecessary unrelated acquisition and execution, not merely avoid unrelated writes. This checkpoint tests the earlier decisive boundary: does the ordinary reconcile path acquire unrelated bytes before selecting scoped work?

## Setup and frozen inputs

Product baseline: integrated main `75031b8eca8161b0a247414741828bfb2e1d9069` (PR #80). Protocol frozen at `e4d8b63b25824f490170fec1e2bc155f26a6eda2`; witness first frozen at `bc97980e6e7bc0b44a0546736a7095e0a23bfce0`. The public `examples/canonical-projection` seed is unchanged. No private Konfyra source or previous sealed holdout was read.

The isolated fixture has eight Definitions, two desired representations (.NET and Markdown), four exact-pinned Modules, two caller-supplied unverified materialization records and one unrelated 100-byte Billing note outside all canonical input lists and projection targets. Fixed Git identity/timestamps and a fixture-local no-conversion attribute make the seed reproducible.

This is the smallest stop witness, **not** the complete requested three-projection Orders/Billing execution experiment. That later trial is NOT RUN. Supplied records do not prove the specimen target artifacts implement any business behavior.

## Canonical revision and Module versions

Real fixture Git revisions:

- Base: `f9f2c23427b80a51bc4a4179c5b27e4729e11dff`.
- Candidate: `985b0a842b4bc532bce2c56c33918844f58e44f8`.

Both runs use the existing 1.0.0 packages `markitect-foundation`, `commerce-example`, `markitect-dotnet` and `markitect-markdown`. Exact immutable pin digests, snapshot/model/plan identities and operational record IDs are in each [observation](runs/windows-2/observation.json). Those are content bindings, not authorization or semantic acceptance.

## Change / mutation

One .NET-only ProjectionPolicy purpose adds an idempotency requirement. No application file, Module package, pin or unrelated Definition changes. The selected representation shares application scope with Markdown, but the policy applies only to .NET.

## Expected behavior

The narrow planner should derive one .NET work proposal and leave Markdown materialization unchanged, reporting conservative evidence freshness separately. Ordinary CLI planning should be read-only. The stronger targeted-acquisition claim would avoid the unrelated note's contents.

## Negative control

Remove only the unrelated note's unique loose Git blob inside the temporary fixture. All selected Definition blobs remain readable at their fixed commit. Call the real CLI again, then restore the object and require the identical plan.

Observed refusal: exit 2, `invalid Git blob size for "unrelated/Billing/review-note.txt"`. Git's full-tree size preflight refused before the content reader opened that missing blob. Therefore this control proves a whole-tree acquisition dependency, **not a cat-file open trace**. Separate positive snapshot observations prove the unrelated exact bytes were acquired when present. This control was not relabeled as syscall instrumentation.

## Observed behavior

The positive and restored real CLI calls exit 0 and emit the same plan as the separately observed public Host pipeline. Repeated planning is deterministic. Two Windows runs reproduce the same fixture revisions, model digests, plan digest and content digest.

| Observation | Measured result | Meaning / limit |
|---|---:|---|
| Changed Definitions | 1 | The .NET policy |
| Semantically affected Definitions | 2 | Policy and its referring .NET Projection; not a traversal-operation count |
| Projection bindings considered | 2 | Binding enumeration, not two agent executions |
| Scoped materialization work proposals | 1 | .NET only |
| No-applicable-materialization representations | 1 | Markdown; no-work does not establish verified convergence |
| Conservative invalidated representations | 2 | Both full-model/revision evidence bindings stale |
| Retained representation needing evidence refresh | 1 | Markdown |
| Base snapshot contents materialized | 21 files / 16,881 bytes | Includes the unrelated note |
| Candidate snapshot contents materialized | 21 files / 16,888 bytes | Includes the unrelated note |
| Working-target snapshot contents materialized | 21 files / 16,888 bytes | Includes the unrelated note |
| Unrelated note bytes acquired per observed stage | 100 | Zero avoidance in this witness |
| Artifact writes by reconcile | 0 | Checked through unchanged working-content digest |
| Executor / Verifier runs | 0 / 0 | No autonomous execution occurred |
| Parent checks | 0 | No recursive behavior proof |
| Token usage, human attention, syscall count | Unavailable | Not inferred from bytes or elapsed test time |

The observed pipeline captures 63 file entries / 50,657 content bytes across its three snapshots. These are snapshot materialization counts, **not unique files, disk-read operation counts, LLM context, tokens or scan timing**. The real CLI path calls the same two canonical source loaders and working snapshot loader; its emitted plan is compared exactly. An extra observer invocation is not counted as a fourth CLI stage.

## Artifacts touched

The witness creates, mutates and restores only its isolated temporary Git fixture. The normal CLI modifies no source/target content and writes no ProjectionRecord. Active records are an explicit external temporary input. Checked-in fixture bytes, historical experiments, Core, Host, Modules, schemas and published artifacts stay unchanged. The repository change contains the harness, declared harness ownership, research references and these evidence files only.

## Context supplied

No agent was asked to implement or verify the specimen. Host receives the closed canonical config and supplied active records, then acquires complete snapshots. The work request carries selected Definitions, Kind contracts and policies. Acquisition size is not agent-context size. Fresh ontology generation and actual Executor/Verifier context measurements are NOT RUN.

## Checks / evidence

- [Windows run 1](runs/windows-1/run.json), [raw log](runs/windows-1/stdout.log): valid initial observation at the frozen witness commit.
- [Windows run 2](runs/windows-2/run.json), [raw log](runs/windows-2/stdout.log): repeated observation with clearer metadata-versus-content commentary.
- The second run preserves the first; its added qualifier does not change setup, mutation, expected classifications or negative control.
- The [file manifest](file-manifest.json) hashes retained evidence bytes.
- [Gates](gates.md) records source-quality checks separately; passing quality gates do not make C5 pass.

## Human interventions

The owner requested this program. Automated investigators inspected public source; the coordinator prepared the deterministic fixture and interpreted the output. No real adopter owner accepted candidate intent. Human attention minutes, review minutes, intervention savings, model/tool telemetry and adoption cost were not instrumented. They remain unavailable.

## Result and limitations

**FAIL:** ordinary reconciliation acquisition is global relative to this selected semantic/target scope.

**PARTIAL existing support:** bounded semantic impact correctly chooses one work proposal and explains separately broadened freshness. This does not rescue the stronger minimal-work claim.

The missing-object negative stops at metadata. No agent execution, actual composition behavior, economic comparison, source-ontology generation reliability or complete repository assurance was tested. No full three-projection trial or benefit outcome is claimed.

## Follow-up and stop decision

Stop before later proof stages. The smallest next action is a focused design assessment of canonical input acquisition, declared target observation, whole-repository inventory identity and conservative evidence refresh. Do not silently change the generic snapshot digest or lose unknown-file visibility. Source identity, input-completeness assumptions and stale-plan rejection must remain explicit.

Only after that assessment and an authorized bounded correction should this same witness be rerun as a new version with both required acquisition and unrelated-read avoidance controls. Until then the whole proof phase is incomplete. No release is prepared or published.

## Answers at this checkpoint

| Question | Evidence-backed answer |
|---|---|
| What is deterministic today? | Typed structural compilation, explicit identity/ref resolution, content bindings, finite impact/plan/record validation and declared checks within fixed supplied inputs; historical test support is bounded. |
| Reliable AI projection? | Unproven. .NET validates supplied candidates; it invokes no model. |
| Module independence? | Static boundaries and disjoint manifest types are enforced; richer independent execution/replacement outcomes remain untested. |
| Correct/minimal derived work? | One correct scoped proposal here; global acquisition and conservative evidence refresh prevent the stronger minimality claim. Autonomous execution is absent. |
| Drift detection/repair? | Existing fixed-scope example and earlier projection evidence remain valid within their stated boundaries; no new C8 trial. |
| Unknown artifacts visible? | Existing ownership/scoped-root tests show visibility; no new repository-wide coverage claim. |
| Recursive integration? | Pure parent evidence composition exists. Actual leaf-pass/parent-behavior-fail trial NOT RUN. |
| Independent AI verifier catch rate? | Unknown; no fresh trials or reliability estimate. |
| Required agent context? | Unknown; no new executor/verifier trial. Snapshot size is not agent context. |
| Brownfield cost? | Unknown. Exact-byte adoption is implemented; real owner review and full adoption costs remain unmeasured. |
| Human-attention improvement? | Unproven; earlier inconclusive A/B remains unchanged. New comparison NOT RUN. |
| Coherence after 30–50 changes? | Unknown; longitudinal trial NOT RUN at stop gate. |
