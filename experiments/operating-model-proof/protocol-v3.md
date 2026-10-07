# Real-agent operating-model proof protocol v3

Protocol ID: operating-model-proof/v3. Frozen before the first v3 invocation.

This version preserves every v1/v2 claim, control, measurement, authority rule, historical outcome and limitation. It makes three public, source-bound fixtures explicit by claim and corrects the C9 execution sequence to match the controller's staged-cohort behavior. It does not change any prior result. No fixture tests or setup checks are capability-trial evidence.

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

Retain v1's protocol checkpoint C5 targeted-acquisition FAIL; its C5 three-projection trial remains NOT RUN. Retain the original three-scope fixture's C9 BLOCKED classification. All pre-v3 NOT RUN, FAIL, BLOCKED and INVALID evidence remains unchanged and visible beside new uniquely bound events. No current v3 run is PASS merely by being configured or prepared.

## Source-bound build and runtime

Every run requires the external CLI build receipt specified by the driver: full `sourceSha`, actual `buildCommand`, integer `exitCode: 0`, and `binaryDigest` matching the exact CLI executable. The source commit must resolve in Markitect's repository. A subsequently observed HEAD is not a build receipt.

Keep the runtime bindings frozen by v1/v2: separate fresh Executor and Verifier, Codex CLI 0.130.0, gpt-5.5, high reasoning, Python 3.13, `internal/tooling/codexrunner/runner.py`, native Windows x64 executable, and total configured runtime files within 256 MiB. Provider-reported usage is included only when actually returned. Raw prompts, provider logs, stderr and candidate payloads remain external; public run records contain bounded observations, actual digests and telemetry only.

The driver and protocol tests validate evidence handling only. They are not capability trials or product checks. Technical evidence does not imply owner acceptance.
