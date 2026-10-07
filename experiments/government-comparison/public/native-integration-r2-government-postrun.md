# Government corrected r2 post-run review

Review date: 2026-10-08. This is an independent read-only reconciliation of the single authorized Government queue start. The run is incomplete. The grant's stop rule applies: no resume, replay, repair, or additional Government start is authorized by this result.

## Finding

The native CLI process returned exit code 0 with no stop reason after 2.2807656000368297 seconds, but the queue, job, run, controller, and outer result are all `incomplete`. One native queue actor started; no actor remained in flight. The failure occurred in the first delegated wrapper before the delegate command ran: the wrapper raised `ValueError('delegate argv does not begin with its bound command')` at `runtime/government_roles.py:691`. The native CLI's process exit is therefore not a successful queue or role outcome.

The diagnostic evidence distinguishes three things: one product queue process, one wrapper invocation recorded in the diagnostic starts database, and zero delegate attempts in the common case ledger. There are no role receipts or role votes. The queue result records no candidate commit and identifies the run report's missing evidence identity as an additional reason the translator preserved the result as incomplete. Nothing here supports semantic success, independent review, acceptance, or a positive case that would unlock a resume.

## Bound raw evidence

The queue output was checkpointed before any later action. All 18 original-to-snapshot pairs in `evidence/native-integration/run-2/government/queue-checkpoint.json` match their recorded SHA-256 values. Key hashes:

| Evidence | SHA-256 |
|---|---|
| `evidence/native-integration/run-2/government/queue-result.json` | `37f09b9c8e390c92e023321e8da6b06e190eb39f18c9d3d06f3a5b88e77322dc` |
| `evidence/native-integration/run-2/government/queue-driver.log` | `e0be5a368b85d75ebcc787dc14f5be34de1fce374b66243999ae21edf707a080` |
| `evidence/native-integration/run-2/government/queue-checkpoint.json` | `79e1766fdf6aa39d221fba70c1fcf2bc3f4468b0b84f15439817c61b0707c01c` |
| external `government/controller-evidence/controller-bootstrap.json` | `10b8abf6cea21db089b8f7524bff14b7403735fee9f335f7836ac5a37fd5d55f` |
| external `government/controller-evidence/process/process.json` | `d77935a87421fb307b6ef2b75768029dc4f2caf7795cdc0c243b7ecb5798e188` |
| external process `stdout.log` (native process) | `e951718e67fba246c0c058600dfcc8d71f6883670d7f84d86abf4e85e625edf9` |
| external process `stderr.log` (native process, empty) | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| wrapper diagnostic `wrapper-000001/start.json` | `476d6a118472d251881cfa2e8cb110b45ff5a0543b1357ff9212574284c97d13` |
| wrapper diagnostic `wrapper-000001/finished.json` | `63ff28200a27128e994f3fc032be38d5dd08014fe2fcc6dbd69badda9ceba175` |
| wrapper diagnostic `wrapper-000001/stderr.log` | `5f3b56e9b6f4f85d29fe8f230695413118545da6f58b60351ad57f223dba6bd0` |
| wrapper diagnostic `wrapper-000001/stdout.log` (empty) | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

The wrapper start is bound to the r2 correction key and hash, pinned wrapper source hash, exact Request path, diagnostics-config hash, and invocation number 1. Its `providerUsage` is null. The stderr contains the traceback and the bound-command validation error; this is the first failure in the delegate wrapper, not an assertion that the native executable failed to launch.

## Read-only ledger reconciliation

SQLite files were opened read-only and passed `PRAGMA integrity_check`:

- The r2 Government case ledger at `...native-metadata-fixtures-20261008-r2\government\fixture-ledger.sqlite` has one dispatch (`finished` as a controller API phase), one controller run (`incomplete`), zero controller process rows, and zero `attempts` rows. The dispatch result is also `incomplete`. This records the reserved controller outcome without a phantom product-process or delegate-attempt row.
- The wrapper diagnostics database `...r2\government\wrapper-diagnostics\starts.sqlite` has exactly one start row. The wrapper's terminal diagnostic receipt closes that invocation with stdout empty and the error stderr digest above.
- The shared native-start database `...native-metadata-fixtures-20261008\native-starts.sqlite` has four total rows: the three prior starts plus this one Government `government-native-corrected-r2/queue` claim, reserved for 150 seconds. The new process receipt records one automatic-retry-free queue start and the matching exit code/wall time. The database SHA-256 after append is `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5`.
- The original r1 Government fixture ledger remains byte-for-byte unchanged at SHA-256 `c00c116064ecc8cd910c72ffb6f6e250b403fded24e20f85e8b55d38f87140a5`. Its previously running historical controller row was not altered by this r2 case.

The external r2 Government case ledger SHA-256 is `ed5e804c27339e46eee778786703a6e0865d35057f115f60dbda332c9bc17c9c`; the wrapper-start database SHA-256 is `4c5672873cae2d9f32c2341543b27054db9de1a0d8d40eef9ad61f859c5f2c90`.

## Usage and interpretation

The captured queue summary reports `nativeActorStarts=1`, `nativeInFlightActors=0`, and `nativeRolesObserved=[]`. `providerRequests`, `providerTurns`, `reportedInputPlusOutputTokens`, and `resolvedProviderModel` are null. Raw input, output, cached tokens, and tool-call counters each have `known=false`, `unknown=true`; their numeric zero fields are placeholders and must not be interpreted as observed zero usage. Observed actor wall time is 194 ms. The parent native process separately reports 2.2807656 seconds. Arbitrary descendant aggregate wall time remains unmeasured.

The failed delegate boundary explains why there is one wrapper start but zero recorded delegate attempts and no returned provider usage. It does not establish that no external activity occurred beyond what these receipts observe. Treat S1 as open; do not claim provider-zero, token-zero, successful Government execution, or resume eligibility.
