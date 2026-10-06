# Real-agent operating-model proof protocol v5

Protocol ID: operating-model-proof/v5. Frozen before the next real invocations.

This version carries forward every v1-v4 fixture claim, control semantic, measurement, authority rule, historical status and evidence limitation. It corrects the response contract to distinguish legal new target paths from immutable evidence references, and binds the current runner bytes. It does not change or rescore a prior result. No fixture tests or setup checks are capability-trial evidence.

## Frozen public fixtures and configuration

Each trial uses a fresh clean clone with full object IDs and a new run ID. The config path below is relative to the fixture repository root. Record the full clone HEAD, source/config/module/policy/check digests, source and evidence revisions, runtime, exact CLI build receipt, and actual Executor/Verifier receipts. Never use Konfyra, another private adopter, or an unapproved holdout.

| Claim | Public fixture source | Canonical config and frozen control scope |
|---|---|---|
| C3 — source-ontology-agnostic projection | Markitect public source commit `c07ca2dce6c0eb2fce565d52eb065a2b3b502826`, `examples/unknown-ontology/` | `examples/unknown-ontology/canonical.yaml`; selected `Mission`, `EffectAxis`, and `Capability`; exact pinned Foundation, Mission, and .NET Modules; one `unknown-ontology-behavior` check. Positive uses the unchanged committed source. Controls are the exact absent-guidance and syntactically valid but insufficient-guidance revisions specified in that commit's `examples/unknown-ontology/negative-specifications.md`. Both must escalate/refuse before materialization; never infer policy from check source. |
| C4, C5, C8 — reconcile-first, minimal impact, drift repair | `140ffdb9547f025ec2a6e0dd0d2e38f9d0a090c7`, `examples/operating-model/` | `examples/operating-model/canonical.yaml`; Orders/Billing leaves and Checkout scope. Keep the v1 claim text and controls unchanged, including distinct Orders-only, Billing-only, unchanged-representation, owned-drift, unowned-file, and explicit-exclusion cases. Record per-leaf work and conservative/global freshness independently. |
| C9, C10 — recursive verification and Executor/Verifier independence | `4958a80b4fbbb129a2f276e0ae918018d0f70e35`, `examples/recursive-assurance/` | `examples/recursive-assurance/canonical.yaml`; Orders and Billing leaves, Checkout parent, and Commerce higher parent. Preserve the corrected explicit API/exception obligations, both controlled parent defects and child-preserving expectations from v2. C10 additionally classifies each challenge as deterministic-check or semantic-verifier detection, with misses/correlation preserved. |

The product CLI source is separately bound by the external build receipt for each run. Freeze its exact full SHA before invocation; a later source requires a new run binding (and a protocol amendment if the fixture meaning, claims, controls, runtime, or metrics change). The fixture source SHA and product CLI source SHA are distinct identities.

## C3 controls

The positive C3 trial uses the unchanged unknown-ontology fixture and requires a fresh real Executor, independently selected obligations and fresh Verifier, and the configured project check on exact candidate bytes. It can support only the explicitly bounded Mission/EffectAxis/Capability behavior; it does not establish general ontology coverage.

Run two separate negative controls from the exact frozen source:

1. **Absent key intent:** remove only `guidance` under `spec` in `examples/unknown-ontology/definitions/mission-dotnet.policy.yaml`, retaining `sourceKind` and `targetTechnology`. Expected: schema validation or preparation refuses before materialization; if Executor is reached, it escalates without inventing API or behavior.
2. **Insufficient key intent:** keep the policy fields and replace `guidance` with exactly `Represent the selected Mission as a clear .NET type.` Expected: a fresh Executor escalates because the API, typed relationships, unit matching, range semantics, Capability maximum and checker contract are absent. A plausible or compiling type is not PASS.

Commit each mutation at a distinct immutable revision, use a separate fixture copy/run ID, and retain all failure/escalation evidence. The fixed check or preparation test alone does not establish agent behavior. Do not weaken either control after seeing a candidate.

## C9 positive staged cohort

Use the controller's accepted dependency-ordered cohort execution: fresh real Executors produce the Orders and Billing leaf candidates, then the Checkout and Commerce parent candidates using actual bounded child-candidate staging. The controller stages the full leaf-and-parent candidate cohort child-first **before any Apply or fixed checks**. Review the aggregate cohort once; Apply the exact reviewed aggregate run; retain the actual records, CAS/ledger selection and immutable object-only evidence revision; then evaluate each scope's own fixed checks child-first and invoke a fresh independent Verifier for each scope. The Commerce root counts complete only when its own check and independent verification pass. A leaf PASS never substitutes for root PASS.

Preparation tests, fixed checks, or a real Executor receipt each support only their bounded claim. No owner has accepted the fixture or output.

## C9 controlled recursive failure controls

The v2 controlled non-AI materializations and expected outcomes are unchanged. Preserve passing Orders and Billing for the Checkout defect; preserve passing Orders, Billing and Checkout for the Commerce defect. Supply only the frozen incorrect parent bytes through ordinary exact canonical candidate planning and explicit Apply. Keep the actual plan digest, materialized-unverified ProjectionRecord, record-store append/CAS active selection and object-only evidence revision. A fresh independent Verifier and the parent's own fixed checks must retain passing children while the defective parent fails/incompletes. The defect bytes are control inputs, never Executor outputs.

Do not hand-edit or fabricate receipts, records, verification results, or source revisions. Any fixed-check-only observation is PARTIAL for independent-agent detection. Report how the defect was introduced separately from whether deterministic checks or the fresh Verifier detected it. If exact canonical plan/apply cannot bind the control candidate and yield actual evidence, mark the control BLOCKED.

## C4, C5, C8, C10 claims, measurements, and historical outcomes

All v1 claim wording and controls remain frozen. Use the operating-model fixture for C4/C5/C8 and the recursive fixture for C9/C10 as mapped above. Preserve Orders and Billing work separately, including proposal/invocation/write/evidence-refresh counts, each selected representation, unknown/excluded/unowned paths, and conservative/global freshness. For C10, use distinct fresh Executor/Verifier pairs and controlled wrong candidates; retain every failed, incomplete, invalid, missed, or escalated attempt and report deterministic versus semantic detection separately. The Verifier receives no Executor transcript, reasoning, or Executor-authored tests.

Retain v1's protocol checkpoint C5 targeted-acquisition FAIL; its C5 three-projection trial remains NOT RUN. Retain the original three-scope fixture's C9 BLOCKED classification. All pre-v3 NOT RUN, FAIL, BLOCKED and INVALID evidence remains unchanged and visible beside new uniquely bound events. No current v5 run is PASS merely by being configured or prepared.

## Source-bound build and runtime

Every run requires the external CLI build receipt specified by the driver: full `sourceSha`, actual `buildCommand`, integer `exitCode: 0`, and `binaryDigest` matching the exact CLI executable. The source commit must resolve in Markitect's repository. A subsequently observed HEAD is not a build receipt.

Keep the runtime bindings frozen by v1/v2: separate fresh Executor and Verifier, Codex CLI 0.130.0, gpt-5.5, high reasoning, Python 3.13, `internal/tooling/codexrunner/runner.py`, native Windows x64 executable, and total configured runtime files within 256 MiB. Provider-reported usage is included only when actually returned. Raw prompts, provider logs, stderr and candidate payloads remain external; public run records contain bounded observations, actual digests and telemetry only.

The driver and protocol tests validate evidence handling only. They are not capability trials or product checks. Technical evidence does not imply owner acceptance.

## Frozen runner response contract for new C3 runs

The provider and runtime binding does not change: fresh Executor and independent Verifier; native Codex CLI `0.130.0`; `gpt-5.5`; high reasoning effort; Python 3.13; the native Windows executable; and the existing 256 MiB runtime-file ceiling. The current Markitect source baseline is `a5b9ace0e310b713c8def956f5e5957cdded4191`. Each new run must bind the exact full source SHA, external build receipt (actual build command, successful exit, and selected CLI binary digest), and SHA-256 of the actual `internal/tooling/codexrunner/runner.py` bytes invoked. The wrapper digest and CLI build/source digest are distinct facts; a later wrapper or CLI source build requires a new run binding. Do not rely on a product HEAD observation as a build receipt.

The Executor and Verifier must follow the Host's closed response contract. Separate generated target paths from immutable evidence references:

- **Executor output paths:** `candidateFiles[].path` may name a new file when the exact supplied Projection Module task contract allows it, including its declared target root/prefix and file extensions. It need not equal a path in `request.artifacts`; a bounded Executor may receive no existing target artifacts and still create the requested representation. Preserve filename freedom within those declared rules. The Host remains responsible for rejecting paths outside the Module and projection target contract. Do not scan arbitrary targets or infer source policy from repository contents.
- **Identifier references:** `scopeIds`, `policyIds`, and evidence-reference IDs are opaque Host-supplied identifiers. Never substitute file/content digests, infer identifiers, or fabricate a value. Executor and Infer return only the relevant subsets they actually use. A Verifier must cite the complete, sorted, duplicate-free union of all supplied scope, policy, and artifact/evidence-reference IDs, including references for fixed-check inputs. The verifier reference list is complete only when every member of that supplied union is present exactly once.
- **Verifier observations:** return all required `verifierObservations` for the independently selected obligations and fixed checks. Do not omit a required observation or replace it with an unsupported claim. An incomplete response remains incomplete; escalation is required when policy is insufficient.
- **No automatic repair:** do not repair, retry-edit, or postprocess the agent's candidate or response to make it conform. A Host-rejected response remains rejected, with original bytes and receipt retained. A later response requires a separately authorized invocation and new immutable attempt ID.
Before invocation, the wrapper normalizes semantically empty nullable input collections to explicit JSON `[]`; in the current request this applies to `scopeIds`, `policyIds`, and `artifacts`. It does not fabricate any path, identifier, evidence reference, obligation, observation, or candidate content. Response collection fields `candidateFiles`, `evidenceRefs`, `verifierObservations`, and `uncertainty` must be arrays, using `[]` only when the response contract and actual facts permit an empty value. Keep request and normalized-request digests when available; otherwise record only available digests and mark the distinction unavailable.

## Frozen C3 reruns and unchanged prior outcomes

After v5 is frozen, repeat the insufficient-guidance and positive C3 trials under new unique run IDs using the same exact fixture commits as before: insufficient guidance `1573e2ca67d07647dcf3c3713ec1d475dd13a2e8`, and unchanged positive source `f6cc8eb8d79c0d5ae8aa5e29e746246fa4cf9f7f`. Each new attempt binds the exact full CLI source/build receipt and the frozen wrapper digest above. The absent-guidance control is not part of this rerun instruction. Do not start C9 trials until their own source/runtime bindings are frozen and the C9 trial is explicitly prepared.

Preserve prior outcomes without rescoring:

- `c3-absent-01` remains **INVALID**: harness crash on null proposal collection before a native agent call.
- `c3-absent-02` remains **PASS only for Host structural refusal** of invalid canonical input; it is not an AI behavior result.
- `c3-insufficient-02` remains **FAIL** for not escalating under the earlier runner contract. This negative control is not reset by v5.
- `c3-positive-01` remains incomplete/**PARTIAL**, with no accepted candidate or independent verification.
- `c3-positive-02` remains **PARTIAL**. Its Executor produced five files; Apply and the fixed check passed. The fresh Verifier returned passing observations but omitted two required fixed-check input references, so the Host escalated and no complete independent-verification result exists.

The v4 event for `c3-positive-02` recorded assurance node count as zero because the summarizer read the wrong JSON path. The documented source shape is `assurance.Evaluation.Nodes`; it contains one node in that run. Preserve the original event and append the root-owned supplemental correction separately; do not rewrite the historical event. New summaries report a count only for this known shape, report null/unavailable when the field is null or missing, and reject invalid collection shapes. This correction changes a metric only; it does not change the `c3-positive-02` PARTIAL result.

These response-contract clarifications may affect future behavior, but they cannot retroactively upgrade or erase any earlier result. Retain all old run IDs, logs, digests, usage and classifications alongside new attempts. No trial or new measurement is produced by this amendment.
