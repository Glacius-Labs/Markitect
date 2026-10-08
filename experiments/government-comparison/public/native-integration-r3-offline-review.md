# R3 offline adapter contract review

**Disposition: reviewed for the assigned offline scope.** The Government command binding and Classic native-role identity mapping now fail closed at the inspected boundaries. The six-role regression exercises the shared Authority/bootstrap and reservation code with a mocked delegate runner. This closes the R2 adapter-contract findings for offline mechanics only; it supplies no native or semantic success evidence and authorizes no further run.

## Source and test binding

Reviewed source SHA-256 values:

| File | SHA-256 |
| --- | --- |
| `runtime/government_roles.py` | `d173a721e33278a20a8667bf3defded549e9031f2a3faf6fd91e6f49486582dc` |
| `runtime/government_integration.py` | `bae84564b89e84643bed2601a843b69dd46266a90eae54ebcae046cc98984e6f` |
| `runtime/classic_integration.py` | `ace20174055c28a41d582430abe370bc4ccad4957eb0ca40183fed880a736fbd` |
| `runtime/test_native_adapter_contracts.py` | `49f4315d60476097636a4d5bc8610a9524696da273d91ae896aac5c94e72bb4e` |

The selected focused gate is bound by `evidence/native-integration/run-3-preparation/focused-tests.log` (`b8e620a4d422e36ec9b202deb25e3a14430e57d1189d588ab8d950cac0b3c17a`), which records 18 passing tests. The separate cross-role log (`1225d4edc78892d85e94457b445674b9a80b08690bc46cda58c6d8d8bb2b1167`) records two passing tests. The contract vectors are SHA-256 `cdc6fbb401d6a3fa8028dd8b0143c1544e5b2f712872a58ec2a1eb1c91f6e845`.

The Classic source contract is SHA-256 `e9a017418485b744135d809ba64e95e927de65396d7b78c8809f423802b17cc8`. The R2 erratum is SHA-256 `4dd0e1987bdacc2f328bd65929f59101367ca02cdb397ea62bb64435ebd8d6ab`, with binding record SHA-256 `19dc8dcbe32751d82a48d3cd5ddeee5b3444c0c1e93b5819328e2ff76d0ff7e7`.

## Findings

Government role authorization now emits an explicit absolute `delegate.command` equal to `delegate.argv[0]`, with the executable and delegate files included in the runtime pins. Static authorization preflight checks each configured slot before controller execution; the launch path repeats the command check. Focused tests reject a missing or changed command before reservation. The frozen R2 authorization had no `command` field, so the pinned predecessor reproduction correctly identifies that old guard failure.

Classic maps a request’s Projection and role to the configured slot and compares `Request.scopeIds` to the canonical Definition identity strings. It checks embedded context Projection and scopes, Definition identities, and the authorized slot. It does not confuse those subject identities with assurance labels such as `commerce-dotnet`. The source contract binds DTO, producer, normalization and serializer blobs to Classic runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`; the source sorts model scopes and Definitions by `Identity.Key()` before serialization. The R3 before-fix shape is synthetic: no complete R2 raw Invocation was retained, and the report correctly preserves that limit.

Government producer evidence is bound separately to accepted source `04e225d5caee78c2a198607143863fca1e829750`, including `execution/actors.go`, its callers, and shared agentexec DTO/runner blobs. The producer chooses executor for `execute`, verifier for review/vote, sets `government/<phase>/<slot>` projection identity, and passes caller-supplied scopes into the Request. Execute scopes come from retained actor data; review and vote scopes are derived from the stored plan and cabinet. The test vectors use synthetic context, run IDs, nonces and digest placeholders; they do not claim to recreate missing R2 stdin.

The six-role chain covers all three frozen slots for each arm. It uses the real local `Authority`, bootstrap serialization/digest validation and per-invocation `load_context`, then both role resolvers and durable temporary reservations. All six slots have protocol-echo success and nonzero-process failure translation (12 mocked bounded calls). The delegate boundary is mocked; native fixture-grant admission and Classic product binding are patched. It therefore proves offline adapter sequencing and error translation, not native readiness or actual delegate behavior.

The R2 Government erratum correctly reconciles the final shared native-start ledger as five rows at SHA-256 `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5`. No exact four-row database image remains, so the earlier four-row count cannot be proven from the cited final hash. Government’s terminal outer process receipt is present in `controller_runs` and the Result; its zero `controller_processes` rows are expected for that direct outer process. There is no unified process collector in this evidence. Any future normalized view must include outer receipts and deduplicate Classic action receipts.

## Remaining boundary

The R2 Government and Classic cases remain incomplete. The proposed next diagnostic checkpoint is two additional native starts and 300 reserved seconds; the larger conditional proposal is not an issued grant. No source, product, or delegate process was started for this review. Provider telemetry remains unknown, and S1, native lifecycle success, semantic quality, verifier independence, human acceptance and comparison results remain unproven.

This review is limited to the offline source/test/evidence snapshot described above. It does not clear the proposal for execution or change the historical R1/R2 records.
