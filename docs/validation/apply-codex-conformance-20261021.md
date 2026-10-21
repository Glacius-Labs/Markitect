# Bounded Codex Apply retest, 21 October 2026

This records the completed series, not current Main runtime acceptance. Source was `f38e5f2fc319d7668d9cd43104bde34f1ecb39fa`; manifest SHA256 was `bd4e13b12e44183f98ff29b18498cd405e9a0233a86335c96e8883a372dd2aee`; external runner SHA256 was `dc987465cbfe81d7f4778b6b7b0f1d5df170b9a64da62a9efd6560bfa38265fd`. All roles used Codex App Server, `gpt-6.1-sol`, configured effort `high`, through dedicated runtime login `mk-codex-runtime-auth`. No Claude, desktop credentials, API-key fallback or model fallback was used.

## Results

| Attempt | Run | Outcome |
| --- | --- | --- |
| R01, A2 v2 preservation | `20261021T162529Z-73835adf` | Incomplete before Verify finished. Implement sealed satisfied; the unpublished candidate changed eight paths. Unchanged target and its 22/22 external checks do not prove preservation. |
| R02, A4-gap / S-QV | `20261021T164050Z-c36b5f91` | Correct Root-confirmed `model_gap` about the absent maximum name length. Failed in the first Assessment; no Implement, publication, target change or external answer. |
| R03, A1 v2 repair | `20261021T164649Z-3fcb7e7a` | Incomplete before Verify finished. Implement sealed satisfied, but the name defect remained in both target and unpublished candidate. No publication. |

All three starts were consumed, without retry. All 129 Engine TurnRecords had observed stops; final cleanup found no active Markitect containers or active attempt reservation. Own work volumes were retained. The 644 provider usage notifications are a different unit from Engine turns and are not a billing count. Costs are unknown; the adapter's raw zero does not prove zero cost.

Recorded monotonic attempt durations were 943.196, 358.186 and 940.861 seconds, totalling 2242.243 seconds; pessimistic ledger charges totalled 2244 seconds. Recorded UTC reservation spans totalled 2136.568 seconds. Both measures are preserved; their discrepancy is unresolved, not normalized away. Each recorded measure was within the old 20-minute attempt and 60-minute summed ceilings. Archival/reporting followed execution closure.

## Why Implement closed

R03's model permits trimmed nonempty names up to 60 Unicode code points; it does not exclude option-like names. The faulty CLI parser can reject values such as `-Bolt` before Catalog's valid name handling receives them. Catalog's P22 claimed preservation using `add_item`, existing unit tests and general CLI integration. Depot's P25 claimed complete input and CLI handling using source citations and the expanded 35-test suite. Root's P27 accepted all five children, the suite and selected integration probes. None established the missing condition across CLI parsing and the Catalog contract. The defect file `realization/app.py` retained SHA256 `faa7ad9f050c0d60c0fa71e7bb4979c6efe64955d128c530f1b5a1ff3ff25e2b` in start, target and candidate.

At the frozen source, `internal/run/proposals.go` validates complete own Definition coverage; `coverageOf` makes per-claim evidence compulsory for Review. `internal/run/pass.go` aggregates accepted reports, open obligations/findings and structural gates; Review adds required checks and adverse-receipt handling. These are real safeguards, but they do not infer all material natural-language conditions or validate the sufficiency of a Manager's semantic claim. Root having no own Definitions does not eliminate its integration duty.

This was an incorrect Implement completeness claim, not a completed false Verify. The final publication guard held. Whether a completed independent Verify would have detected the defect is unproved by this series.

## Reference uncertainty

R01 and R03 also changed large-order-ID handling and validation of overreserved persisted state. Offline independent review did not establish all those edits as unnecessary. Unknown huge IDs may already have the required public refusal result; tests of a private parser or exact free error text impose stronger assumptions. Rejecting cross-section-invalid persisted data is a plausible reading of malformed-data requirements, but the model is not explicit enough to settle the dispute. Adjudicate and version these controls before treating the reference as conforming. Keep this historical result unchanged.

## Evidence location

Original series root in WSL Ubuntu: `/home/user/mk-sa-codex-followup-f38e5f2f-20261021-01`. The versioned export is under `C:/Users/Consiliari/.codex/visualizations/2026/10/21/01a15ec1-05e1-7180-85a6-7d994e0918f1/codex-per-turn-retest/completed-20261021-v1/`: `REPORT.txt`, `summary.json`, `SHA256SUMS`, `DELIVERY-SHA256SUMS`, `inventory.json`, the independent offline review and the evidence archive. Local locations are provenance, not portable repository prerequisites. A successor without those files must report the evidence gap rather than invent transcript details. Credential directories were excluded; the export's bounded pattern scan is not a universal secret-freedom guarantee.

The [accepted correction](../design/canonical-model/manager-apply-workflow.md#evidence-backed-acceptance-after-the-codex-retest) and [implementation handoff](../development/codex-apply-conformance-handoff-20261021.md) are subsequent work. These three attempts establish neither general reliability nor a model comparison, hosted CI, release, production readiness or owner acceptance.
