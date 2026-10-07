# Sequential operating pilot

This package records a predeclared three-task continuation from the actual final state of the [two-area pilot](../two-area-operating-pilot/README.md). It retains a stopped first attempt and a separately frozen second attempt that completed all tasks. Original CLI source `74cbb7ca630e221b9b365e13e70165e9c7f59b6a`, runtime, bridge and independent behavior oracle stayed fixed; no product implementation change was made for this sequence.

## Outcome and retained failures

| Task | Completed evidence |
|---|---|
| Orders SKU intent | Orders and Markdown changed; Fulfillment bytes did not. The actual applied libraries passed 700 checks. Attempt 02 resumed those bytes and obtained fresh verification and complete audit after one bounded retry. |
| Shared callback-failure Rule | Both .NET areas and Markdown changed; project files did not. Fresh verification, audit and 724 behavior checks passed. Exact old Fulfillment bytes made audit incomplete and caused an uncaught exception in the behavior process despite a successful build. Exact restoration returned the same complete audit digest. This simulates a stale representation; it is not a planner miss or autonomous repair. |
| Orders drift, unchanged intent | A predeclared whitespace-guard defect compiled but failed 160 of 700 SKU checks. Ordinary reconciliation selected Orders alone. A fresh Executor restored exactly the pre-drift handler; Apply changed only that file. Fresh verification, audit and 724 behavior checks passed; other targets stayed byte-identical. |

Attempt 01 stopped after response-format errors and coordinator corruption of a submitted response, followed by a post-capture edit. The original Fulfillment child JSON had valid escaping; this does not claim that its complete original envelope passed Host validation. The passing SKU holdout never made that incomplete attempt complete. Its frozen [result](attempt-01-result.yaml) is retained as a sanitized derivative, not rewritten as success.

Attempt 02 used actor-owned serialization, exclusive draft creation and atomic hard-link publication. Its first Fulfillment verifier failed to create a response under the default filesystem sandbox; the parent supplied an incorrect opaque check identity. Host refused both. One bounded retry with unchanged leaf inputs and scoped local write authorization passed; the parent reran against current child outcomes. The template is frozen for the attempt; Task 1 has per-job prompt/identity bindings but lacks the later exact per-dispatch instruction receipts. Tasks 2 and 3 retain those receipts. Actor drafts, published responses and captured raw response bytes match for all successful Attempt 02 submissions. Normalized bridge output is a separate serialization.

A premature read-only proposal, a wrong-source Verify preflight and an empty pre-dispatch instruction file are retained as coordination detours with no actor dispatch or live-state mutation from those erroneous steps. A composition sentence was clarified before Task 2 actors started, within the frozen behavioral contract; the intermediate source commit and model receipt remain in history. Root and Luna coordination are not measured human attention. The [completed result](attempt-02-result.yaml) accounts for the source, operations, controls and interventions. Its original freeze-receipt pointer and omitted first-phase provenance are corrected separately in [result corrections](receipts/result-corrections.yaml); the original record remains unchanged. [Planner observations](receipts/t3-planner-observation.yaml) retain the distinction between current Fulfillment representation and Markdown requiring fresh assurance.

Final canonical source is `a2e0ab51eee175a9a5231fc2e833811b86d98281`; final target evidence revision is `ba6074ccd3e06f2b889c0ac8178f8df8e08165b3`. Restored Task 3 target bytes equal the valid Task 2 bytes, so their target evidence revision is equal; new Apply and fresh verification have separate ledger evidence. Final audit is `sha256:1d1bb03f70013c832ca678c2e0cdddf69fe56d343cf78aabf3d56b99240467e3`, complete with no findings.

## Inspect source and repeat behavior checks

The run used Windows, Python 3.13 and .NET SDK 10.0.103. From this package directory, verify `canonical-source.bundle` against the SHA-256 in [source-manifest.yaml](source-manifest.yaml) or [evidence-index.yaml](evidence-index.yaml), then clone to a new disposable directory outside the product checkout:

    Get-FileHash canonical-source.bundle -Algorithm SHA256
    git clone --no-checkout canonical-source.bundle <new-directory>
    git -C <new-directory> switch --detach a2e0ab51eee175a9a5231fc2e833811b86d98281

Copy the five files under `target-samples/` into that new checkout at their corresponding relative paths. Those files are the byte-exact final representations; the Git bundle contains canonical source history, not the operational ledger or an automatically populated live target tree. Then use separate, previously absent output directories:

    python holdout/run_holdout.py --fixture <new-directory> --stage sku --output <external-sku-output>
    python holdout/run_holdout.py --fixture <new-directory> --stage callback-policy --output <external-callback-output>

The holdout builds the two libraries separately and checks caller composition through their public APIs. It is the unchanged pre-candidate oracle. This reproduces technical behavior checks, not the live agent queue or controller history. A negative callback control that terminates before emitting its result has no completed check count; do not interpret a missing count as zero failures.

## Provenance and limits

Protocol, result and receipt `.yaml` files use JSON syntax and are sanitized, reserialized derivatives. The evidence index distinguishes their original external hashes from their packaged hashes. Exact source Git objects/metadata, final target samples, holdout and response-instruction template are preserved byte-for-byte where stated. Machine-local paths and actor identities are redacted or aliased. Raw provider payloads, runtime configuration, full private logs and live queues remain external; their retained digests do not make the public package a complete replay of that environment. The instruction template is an experiment transport instruction, not a native provider adapter.

All live actors were configured as fresh Luna High forks. Provider version and token usage were unavailable; separate supplied contexts are not an OS security boundary. The negative controls were deliberately injected by the experiment. Passing judgments and behavior checks are bounded evidence, not proof of arbitrary semantic correctness, reliable unattended operation, reduced human effort, production adoption or release readiness. Full product-source gates belong to the exact integration commit and are separate from the frozen pilot binary.
