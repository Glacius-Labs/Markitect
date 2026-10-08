# Independent R3 post-run review

## Disposition

The retained R3 receipts support the reported bounded outcomes, with the qualifications below. I found no material contradiction between the terminal evidence, the frozen inputs, the additive native-start ledger, and the archived external files. This is a mechanics/evidence review only. It does not establish S1, semantic quality, real-Actor capability, study success, or human acceptance.

## Bound evidence and checks

The preflight freeze is `evidence/native-integration/run-3/preflight-freeze.json` (SHA-256 `acd01a42a7a7b511188479e66222430b5d9d628a4661fe7d7c5ff0bf6f5190ba`). Its 102 declared input files still match their recorded hashes. The freeze binds the released R3-A1 grant and Coordinator snapshot, the preflight review, source pins, requests, and runtime inputs. The current admission/runtime source remains the frozen R3 source; the final checkpoint driver adds positive Classic status and raw-stdout binding checks. No post-run source or test execution was needed for this review.

The closed archive manifest (`external-snapshot-manifest.json`, SHA-256 `19d87aea5567520a8bf9d69452d813c4675b5a96ce6e0e0c926bece6148db1b1`) lists 2,303 original/copy pairs. I hashed all 4,606 files: every file exists and all hashes match. The archived native-start database and both R3 role-ledger snapshots pass SQLite `integrity_check`.

## Government

The single authorized Queue process returned 0 without timeout or a stop reason, but its queue and job ended `incomplete` at Execute. The deterministic Executor response passed the role-protocol echo, then the pinned native host rejected its `candidateJson: null` as non-empty `json.RawMessage`, with `executor response contains verifier or inference output`. The separate failure attribution (`government/failure-attribution.md`, SHA-256 `7e86d34400e1c73ba7ffd0378adccfe512b8048d313b6ce090ff0657d197561c`) traces that contract mismatch to the pinned host and delegate serialization. The missing evidence identity in the normalized run report is a downstream qualification, not the cause of the host rejection.

No Resume, retry, repair, or second Government start followed. The outer native receipt is represented once in `controller_runs`; `controller_processes` has zero Government rows. This is consistent with the product's receipt layout and does not mean the native process was absent. The case is negative/incomplete, not a product semantic verdict.

## Classic

The five declared native actions are consistent with the terminal flow: Execute `planned`; Apply `materialized-unverified`; fresh Verify `passed`; Audit `complete` with zero findings and zero next steps; stale Apply replay `refused`, return code 2, and no writes. The overall controller flow is `completed` with no reported errors, no candidate commit, and no human acceptance. The controller took about 92.1 seconds within its 180-second window. Three fixed deterministic role slots ran; these are fixture delegates, not real study Actors.

The original structured independent Execute approval remains unchanged and binds the exact Execute report SHA-256 `4b746df3082120c05f7d0f2310ade6e37e6dbd2c950c4ccf2b14fb59ebcabf9f` and digest `sha256:b87bd40e05f9ddd08cb06e9fbb8305de6a19765c42ddbec6526fd6136f76cfdb` (approval SHA-256 `cbad08e1bf6038ef21224194190720137b140a809db49a6bb9e8228e80874ee3`). The original human-readable review narrative is preserved unchanged at SHA-256 `05efc166447d96ec0122c7ec85507485b8473507ce6cf35a37ec3c4e481d2d12`; it contains literal shell interpolation and a corrupted commit string, so it is not independently readable as written. The separate post-run rendering erratum (SHA-256 `475b071e8cfc2cb7e846d286ddf3a0d053f61779b529b900f5ca4aabd1a48fe7`) supplies the exact observed values, binds them to the unchanged report and structured approval, and explicitly states it is neither a new approval nor retrospective authority. It repairs the narrative record only; it does not change the original decision.

## Accounting and limits

The appended native-start ledger contains 11 cumulative starts and 1,650 logical reserved seconds: the five historical R1/R2 starts plus one Government and five Classic R3 starts. R3 accounts for six starts, four wrapper attempts, four deterministic delegates, and 900 reserved seconds. The Government outer process is counted once from its controller receipt; Classic has five action rows. Historical reported tokens remain a known subtotal of 53,331 with the true total unknown (`null`). Current native/provider usage fields remain unknown where telemetry was not available; fixture-declared zero provider calls do not turn those fields into measured zero.

The run is bounded evidence of the exact disposable fixtures and pinned binaries. It does not prove filesystem isolation for the wrapper, aggregate descendant runtime, general workflow correctness, or semantic validity. No new model/provider call, metadata session, real Actor, or study cell is evidenced. Historical failure states are not rescored, and no further R3 execution is authorized by these results.
