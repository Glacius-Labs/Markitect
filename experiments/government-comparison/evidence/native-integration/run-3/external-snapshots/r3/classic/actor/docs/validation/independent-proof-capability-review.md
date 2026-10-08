# Independent proof-readiness capability review

Review target: integrated Markitect source commit `84fc27ca71f32b8636a28fc1e64ecfc8660c3863` (2026-10-06). This review is a source-level contract review, not a current-head or provider-behavior proof. It does not reuse historical C9 evidence as proof for this revision.

## Concrete controller traceability gap

The controller can persist a passing Verifier result without an observation for each declared verification obligation. `internal/host/canonical_controller_verification.go` constructs `canonicalControllerExactEvidenceRefs` at lines 835–862. It requires `EvidenceRefs` to enumerate each supplied ScopeID, PolicyID, and artifact path, but it checks each observation only for membership in that allowed set. `canonicalAgentCheckOutcome` at lines 864–887 starts with the response-level outcome and only downgrades it for failed, escalated, or incomplete observations. Separately, `internal/host/agentexec/runner.go:393` requires only one observation for a passed Verifier response.

This is reproduced without an AI call by `internal/host/canonical_controller_verification_test.go:74–94`: the test supplies all four references but only one passing observation on `docs/orders.md`, then asserts that the validator accepts it. If the response-level outcome is `passed`, the verifier check remains passed. In particular, fixed-check identities appear in the verifier context but are not subjects in the exact reference set, and no observation coverage check ties them to the result.

This is a structural traceability/acceptance-gate defect. It is not evidence that a provider made a semantically incorrect judgment, and no observation schema can guarantee semantic truth. A bounded repair is to require a passing response to cover every selected ScopeID, PolicyID, fixed-check identity, and materialized target artifact with an explicit passing observation. Reject missing or duplicate subjects as incomplete/escalated; preserve partial observations for failed/incomplete responses, and retain the existing rule that any negative observation prevents overall PASS. Keep the change Host/protocol-local: no Core concept or free-form obligation DSL is needed. Tests should cover single-observation rejection, complete coverage, missing check/artifact coverage, duplicate/extra subjects, and failure/incomplete preservation.

## Boundaries reviewed

- `internal/core` and the product schema tree have no diff against `origin/main` at the review target. Changed `examples/**/schema.yaml` files are package/fixture inputs, not Core schema-language changes.
- The accepted readiness design and current code keep Core and capability Modules separate: Module proposal packages consume Core-facing inputs and do not import Host or sibling Modules. This is an architectural boundary observation, not evidence that every Module result or generated artifact is semantically correct.
- Brownfield inference uses explicitly captured handoff/queue/blob inputs and the existing candidate validator; its result is noncanonical and cannot adopt. Goal modeling consumes caller-provided package bytes, resolves selected exact pins, decodes proposed Definitions, and compiles against the selected Core schemas. Neither flow authenticates a decision-reference claim or constitutes owner acceptance.
- The docs explicitly identify semantic insufficiency escalation as unproven behavior. API shape, protocol helper tests, and a green structural check do not prove model quality, independent judgment, or human acceptance.

## Validation scope

The Markitect `check` and `development/Skill/engineering-change` context passed with fixed input revision `84fc27ca71f32b8636a28fc1e64ecfc8660c3863`. Focused `agentexec`, `recordstore`, `assurance`, `canonical`, `internal/modules/...`, and `internal/core/...` tests passed in the mutable integration checkout. That checkout changed during coordination; these focused test results are not an exact-84fc or final-head quality gate.

The broader `internal/host` test package was attempted but is **INVALID ENV GATE**, not a functional failure: the managed sandbox denied test writes to `.cache`, `AppData\\Local\\markitect-controller-*`, and a temporary directory under the worktree. No permission escalation or repeated host test run was attempted. The specific helper-level traceability reproduction above is an existing deterministic test and does not require those writes.

This note does not establish behavior at later source revisions, real-provider semantic correctness, privacy guarantees, owner authorization, adoption, deployment, or release readiness.

The reviewer output was initially untracked in a different writable checkout after its shared Git index refused writes. The coordinator copied the note into the integration branch; no production change came from that output.
