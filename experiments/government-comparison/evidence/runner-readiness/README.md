# Finite runner readiness receipts

`native-r3/` is the final native wrapper smoke. It binds all current runtime source
and config/pin files, exact request bytes, real Codex/Classic version output,
process-tree control and raw-log hashes. It launches zero providers or Actors.
All three arm results are `readiness_gap`. The generated protocol subset and
hashes of the full protocol output preserve the installed-interface inspection.
Large `ClientRequest.json` schemas and baseline source-gate logs use lossless
`.gz` storage. Decompression reproduces their original bytes; schema digests
refer to those uncompressed bytes.

The top-level raw receipts and `readiness-smoke.json` preserve r2 unchanged.
After r2, Classic's 40-hex Git identity keys were renamed from `*Sha256` to
`*Sha`, without changing identities, and `runner-pin.json` was added. Therefore
r2's one old metadata hash is historical; r3 owns the final source binding.
`validation.json` records that initial checkpoint and historical source gates.

Original absolute paths stay intact in raw requests/results. Map paths relative
to each smoke's `actorRoot` into its matching frozen evidence directory, then
verify the declared SHA-256. Input paths pointing to the Classic packet or runner
are external pinned dependencies and are not rebased. Nothing requires changing
old raw receipt bytes. External directories remain preserved.

`runtime-tests.log`: 13 focused checks passed. `harness-tests.log`: 5 existing
preparation checks passed. Neither is a full Go suite or a live actor result.
`runner-observation.json` separates installed CLI/schema inspection from actual
account/model/access evidence, which remains absent.

The independent review is a code/measurement-boundary assessment. The new
`runner-readiness-freeze.json` hashes source and evidence for this checkpoint;
the final source-validation metadata separately binds its committed source SHA.
S1 is not cleared. The historical full Verify remains incomplete and was not run
again. No holdout or future task card was supplied to an Actor.
