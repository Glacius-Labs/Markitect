# Independent review: Government Executor response correction

## Finding

The narrow offline correction matches the pinned Government host contract for the exercised Executor and Verifier response shapes. I found no material source-to-sample contradiction in the assigned scope. This review supports only the response-serialization correction and offline contract checks. It does not show that the native Host admitted a response, that a Government run succeeded, or that any proposed material is semantically correct.

## Source and wire evidence

The pinned host source is commit `04e225d5caee78c2a198607143863fca1e829750`. All four saved source files reproduce their declared Git blob IDs and SHA-256s in `offline-evidence-manifest.json` (SHA-256 `f4356a19ec1574661a16a5bd86bd5aa2d5c34495047b947fda51f8c86a235ef7`). In `agentexec/runner.go`, `validateResponse` requires all four arrays to be non-null. For Executor it rejects nonempty `verifierObservations` or `candidateJson`; for Verifier it rejects nonempty `candidateFiles` or `candidateJson`. Since `candidateJson` is `json.RawMessage`, the old literal `null` occupies bytes and is rejected. The same host source validates the response before assigning it to the result; Government execution returns on that invocation error before proposal application and the later evidence-producing stages. That source order supports treating absent evidence identity as downstream of the rejected response, not as a separately established evidence-generation defect.

The corrected fixture omits `candidateJson` for both supported roles, retains all four arrays, and keeps the role-specific forbidden arrays empty. `response_bytes()` emits the serialized response, and CLI `main()` writes those bytes directly. The one Executor plus six review/vote samples are source-derived outputs from that serializer; they are explicitly classified as examples, not process invocations or Host admission. I parsed each pair against its recorded hash and checked role, outcome, identity fields, required non-null arrays, omission of `candidateJson`, and the role-specific empty arrays. All seven pairs passed those structural checks. The four saved host snapshots match their declared blob IDs. No source edit changes the Host, its role validator, or a role-admission rule.

The strict delegate pin in `government_roles.py` is updated to the corrected fixture SHA-256 `e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9`; there is no fallback to the old source hash. The fixture source hash matches that pin. The change is confined to the deterministic fixture responder, its strict digest binding, the new serializer tests, and new offline evidence/handoff text.

## Test record and deviation

The retained focused gate log (SHA-256 `27eb38f258a04afac947fcd03dec286e0e4ed15f376de0a706bff6f61ca4a271`) records the requested eight focused tests passing: six serialized-byte cases and two designated existing response-body checks. The new test file itself has SHA-256 `047daec8a575fe7f6b675a6b350e24850b3288b16b2f4472da800c2958e44aae`.

I accidentally invoked both complete unittest modules rather than the specifically permitted eight-test subset. The first command failed during imports before any test body ran; the second ran 14 tests and passed. The extra six were temporary fixture/preparation and binding tests, including local Git operations; they did not invoke the product executable, native controller, wrapper, delegate, model/provider, or metadata client. The exact commands, six test names, results, and scope are preserved in `reviewer-test-command-record.md` (SHA-256 `4dbcd97037aadfd8bbf130ceba24ec3b35fb54d1a5d825b3e17dd2296fa9bb6f`). The 14-test result is not described as fourteen pure-function tests or as a product run. No further tests were run.

## Limits

The retained R3 stdout remains unchanged and still contains `candidateJson: null`; this review does not rescore that historical incomplete Government attempt. No delegate CLI or process was launched to produce these samples, and no Host validator was implemented or substituted. New experimental runtime consumption is zero: no native starts, wrappers, delegate processes, provider/model runs, metadata sessions, or study cells. A later actual Government attempt requires a separate finite grant. Semantic quality, evidence generation after a valid executor response, real-Actor capability, and S1 remain unverified.
