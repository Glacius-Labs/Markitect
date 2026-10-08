# Government R3 queue failure attribution

Date: 2026-10-08. This is a read-only attribution of the existing single Government R3 queue run. No resume, retry, process, test, product file, ledger, or external artifact was started or changed for this diagnosis.

## Finding

The native queue process itself exited successfully (`returnCode: 0`), but its only job and resulting queue were **incomplete** during the `execute` stage. The direct native error was `executor response contains verifier or inference output`. This is an `agentexec` response-contract rejection, not a failed Go check, timed-out process, or semantic-quality verdict.

The evidence locates the offending field precisely. The one deterministic Executor response contains `"candidateJson":null`, while `"verifierObservations":[]`. The pinned Go host decodes `candidateJson` into `json.RawMessage`; `encoding/json` retains the raw bytes for JSON `null` (`RawMessage.UnmarshalJSON` appends the input bytes). In the Executor branch, `runner.go` rejects whenever `len(response.VerifierObservations) != 0 || len(response.CandidateJSON) != 0`. Thus the empty verifier array is allowed, but the present four-byte RawMessage value `null` satisfies the second condition and produces the exact recorded error. The candidate file array and `outcome:"proposed"` do not cause this check to fail.

At pinned product source commit `04e225d5caee78c2a198607143863fca1e829750`, `internal/host/agentexec/types.go:77-87` declares `CandidateJSON json.RawMessage`; `internal/host/agentexec/runner.go:226,230` strict-decodes the response and immediately calls `validateResponse`; lines 384-390 contain the Executor outcome and rejection above. The corresponding Git blob IDs are `664a36e5984ae2a3360b733f50b40b3c18f41cbb` (`types.go`) and `84893e927a96edd787152a769d5ae55b042e506a` (`runner.go`). These are read from the exact accepted source commit, not inferred from a newer checkout.

The configured fixture delegate is the pinned provider-free Python responder (`deterministic_delegate.py`, SHA-256 `bcb541ab827e293f215e1a621205c68d840271eb263191476dfc3a45463ec9c1`). Its shared response initializer sets `candidateJson` to Python `None`; its JSON serializer emits that as `null`. The role receipt records `providerVersion:"python-fixed-role-response-v1"`, `knownNoProviderCalls:0`, and `status:"protocol-echo-valid"`. The Scientist role bridge at `runtime/government_roles.py` (SHA-256 `de8c60a1891c82b1bfedbea03c0e458102287e75eb1a91cb86d50997d40b8153`) checks the response envelope identities in `_response_usage` and relays the original stdout bytes from `run_role`; it does not reject executor `candidateJson:null`. The delegate's `stdout.log`, wrapper `stdout.log`, and receipt `responseSha256` all have SHA-256 `7d434a9c103ced86de2886a74bdc7ce589c7956b0f4f5bb356fc896f6b9c6092`. This explains why the role bridge reports a valid invocation echo while the stricter native host subsequently rejects the role-specific response.

## Independent report-identity gap

The run report at `results/run-state/government-run--3ff444b05f0b7d0b5293051ba3dbf93c/report.json` has no top-level `evidence` object. Its only actor result has a zero-value `Response` and the same Executor validation error. The Scientist-side translator's `_has_evidence_identity` check requires `evidence.id`, `evidence.materialCandidateId`, and a positive integer `evidence.round`; it therefore preserves the report as receipt-only and adds `Native run report lacks evidence identity; preserved bound queue/run receipts as incomplete.`

That missing-identity classification is a separate downstream observation. It did not cause the native Executor error: the host rejected the response before producing a valid actor result from which an accepted candidate/evidence identity could be carried forward. The queue journal records the incomplete run and job, with no checks, votes, candidate commit, or acceptance. Do not synthesize evidence identity for this failed attempt. Neither the response's proposed candidate bytes nor its deterministic test content establishes semantic quality or S1.

## Counts and stop boundary

Read-only queries of the existing ledgers found exactly one new R3 Government native reservation (`government-native-contract-corrected-r3/queue`, 150 logical reserved seconds), one `government_role_calls` row (`phase=execute`, slot `positive-root-executor`, status `protocol-echo-valid`), one wrapper-diagnostics start, and one delegate process receipt. The native process returned 0 in 2.094 seconds; the delegate returned 0 in 0.142 seconds. The queue recorded `actorStarts:1`, `repairs:0`, `inFlightActors:0`, and terminal journal sequence 9. Its `maxRepairs:0` and terminal incomplete job are consistent with the no-retry stop; this diagnosis authorized no further work.

Provider request/turn counters and token counters remain unknown/null in the native usage summary. The fixture receipt identifies zero provider use for the deterministic delegate; that fixture accounting does not turn the product's unknown provider telemetry into measured zero usage.

## Bound artifact digests

All paths below are beneath `C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r3\government`, except the original allocation ledger. The released Request binds the R1 grant and additive R3-A1 grant/snapshot; its SHA-256 is `d36c48f005bd5721529900588b21ded8b0ed1234db515c0bc23d445f70085794`.

| Artifact | SHA-256 |
| --- | --- |
| `released/request.json` | `d36c48f005bd5721529900588b21ded8b0ed1234db515c0bc23d445f70085794` |
| `released/runtime.json` | `d88fb01e232968a8f64726e66349d111727029668782469a398c4a9b46d4cdb3` |
| `released/role-auth.json` | `27ec92f1d1716b3785fdc311b03a3f854bfbb76692fddd9ae4624bb237cd4170` |
| `wrapper-diagnostics-config.json` | `43b45b68e95035f2a9557df3bc5334bc7abf0630aeec6f3d087579fd18e88428` |
| `controller-evidence/process/process.json` | `e3b8536a1c3cf4f1f6c76f98d1f1c64ab028f40e5a2994a9f843a82a8c50e55a` |
| `controller-evidence/process/stdout.log` | `0f58f63b4445361a80d0f0dad3ed9ad9e7af3c457fea99a3f1e741dee1e2b525` |
| `results/run-state/government-run--3ff444b05f0b7d0b5293051ba3dbf93c/report.json` | `b01dfcca88a170f34932c708061a8b761b2eb3242c97c6014e99a5ba6ce47e98` |
| `results/run-state/government-run--3ff444b05f0b7d0b5293051ba3dbf93c/actor-0001.json` | `a994a9def09e62d1f3ff143d8ea6d657965e4c12ce359852a3a5e8f90eab3f82` |
| `results/queue-state/government-queue--53e82cb12899a87e9d66de9964011511/events.jsonl` | `5eda1f8711c39ee04309a4ebbfed3c4b5cce63fe6878d5969d0c68820b27492d` |
| `role-evidence/579807ab283d8dfd5484b28076523cabd6e666050cdec8ec91cf58732e0dc9f7/process.json` | `87ceea22df9d4e1b09a565dbf6de33110c54067f69fe36ea739b21ec7a86e7ea` |
| `role-evidence/579807ab283d8dfd5484b28076523cabd6e666050cdec8ec91cf58732e0dc9f7/stdout.log` | `7d434a9c103ced86de2886a74bdc7ce589c7956b0f4f5bb356fc896f6b9c6092` |
| `wrapper-diagnostics/wrapper-000001/start.json` | `b608efec00958386af18adaf3d6b907afe96423410f644abc55d3384fa50d07e` |
| `wrapper-diagnostics/wrapper-000001/finished.json` | `0fc57a53b081b1ca6f5d69b19943775a268f575f7e368665fdd26be6c4125fc6` |
| `wrapper-diagnostics/wrapper-000001/stdout.log` | `7d434a9c103ced86de2886a74bdc7ce589c7956b0f4f5bb356fc896f6b9c6092` |
| `wrapper-diagnostics/wrapper-000001/stderr.log` | `d5169feeee345171bec8caad7f44ac4e4b8431970e520e6807f709f3b03ca21e` |
| `government/fixture-ledger.sqlite` | `e3e6afacc42edbf5c3dde12affc886fff6fb31e0ebe58aee8936d3f543966472` |
| Original `native-starts.sqlite` | `5c08ba595cad4e6c8d4185027858a59b62024c6bc68a294b731c272130b3c097` |
| `outer-results/government-native-positive.json` | `73622f16f253ebdf184a9e4e4bb526f281a2c77a1b149ce93769022e42d8c1b7` |
