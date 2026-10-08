# R2 Government post-run errata

This new, separately bound erratum corrects the accounting statement in the earlier Government review without changing that review, its freeze, or any run evidence. It was prepared read-only under the offline R3 assignment `s1-offline-adapter-contract-repair-r3-preparation`; it authorizes no native execution.

## Corrected row count and hash

The earlier [R2 Government post-run review](native-integration-r2-government-postrun.md), SHA-256 `eec23b494f20ebf78fb62bc8b7f525560de3887fb90c2be9e6f8b55703cf7998`, says that the shared `native-starts.sqlite` has four rows and cites SHA-256 `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5`. The cited hash is the final archived database, and that database has **five** rows. The four-row count and five-row database hash were incorrectly combined in the earlier review.

The final original database and its R2 archive copy are byte-identical at SHA-256 `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5`; both pass SQLite `integrity_check`. The archived copy is `evidence/native-integration/run-2/external-snapshots/native-starts.sqlite`. Its ordered entries are:

1. Government `inspect-constitution`, 150 reserved seconds.
2. Government `government-native-positive/queue`, 150 seconds.
3. Classic `classic-native-positive/execute`, 150 seconds.
4. Government `government-native-corrected-r2/queue`, 150 seconds.
5. Classic `classic-native-corrected-r2/execute`, 150 seconds.

The first three rows match the retained pre-R2 `run-1/native-budget-snapshot.json` exactly. The final two rows are the two authorized R2 appends. The archive contains the final five-row database; it does not contain exact bytes for a four-row intermediate database. The earlier count may describe an intermediate observation before the Classic append, but the cited final hash cannot prove that state. For final delivery, use five rows at the cited hash. No historical database was rewritten to produce this correction.

## Controller receipt and metrics contract

The new Government controller booking is terminal and incomplete. Its `controller_runs` row has a persisted `process_receipt` and `receipt_sha256`; its Result contains the native `controllerProcess`, stdout/stderr receipt references, process receipt, bootstrap and released-input receipts. The Government case ledger has zero `controller_processes` rows and zero Actor `attempts`. Its archived SQLite hash is `ed5e804c27339e46eee778786703a6e0865d35057f115f60dbda332c9bc17c9c`.

The current source distinguishes an outer controller run from processes executed as controller steps. `Ledger.finish_controller_dispatch()` stores the terminal controller receipt on `controller_runs`; Government dispatch passes its single native process, process digest, bootstrap digest, and elapsed time there. `Ledger.record_controller_process()` appends an individual action receipt to `controller_processes`; the Classic flow calls it for its `execute` action, giving one incomplete Classic row. Both `controller_runs` and `controller_processes` are returned separately by `Ledger.snapshot()`. The archived Classic ledger is incomplete, has one `execute/incomplete` controller-process row and zero Actor attempts, at SHA-256 `52c6e65e64db90b00d8a65cc5e4d8714c726b0f7052f27187091b89ed1d1b7c6`.

**Finding:** the Government process receipt is not missing from the common durable ledger or Result. A zero row count in `controller_processes` is expected for this one-process outer Government invocation; that table records child/action processes such as Classic `execute`. The shared native-start ledger also records the Government process claim and reservation. Therefore this is not a historical accounting defect and does not warrant a ledger repair or duplicate process row.

There is a presentation constraint: `controller_processes` alone is not a total native-process count. A collector that needs a unified per-process view should normalize the Government outer process from `controller_runs.process_receipt` and Classic action processes from `controller_processes`, with receipt-digest deduplication because Classic’s outer receipt also summarizes its actions. The current read-only ledger snapshot preserves both source records; the smallest future improvement, if a single normalized count is required, belongs in a derived collector view and must not rewrite historical rows.

## Bounded result

This correction only reconciles retained evidence and the current source contract. Both R2 product cases remain incomplete; Government provider/token values remain unknown, and all R2 execution stopped under the first-failure rule. Unused numeric quota does not authorize a retry. S1 and product readiness remain open. No tests, product process, delegate, provider, or database write was performed for this erratum.
