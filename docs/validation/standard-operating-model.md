# Standard operating model: executable closure contract

This checkpoint adds a read-only completion audit to the existing source-only canonical controller. The [design](../design/standard-operating-model.md) owns its contract; [Usage](../usage.md#declared-scope-completion-audit-source-only-alpha) owns the command; the [roadmap](../implementation-plan.md#standard-operating-model-checkpoint) owns current status. The implementation starts from source candidate `5808ea586979cb48e20f94157b6ccfa9ed318c60`; it does not change the published release.

## What the executable coverage establishes

The Host tests use the real canonical loader, .NET and Markdown Projection Modules, proposal/execution/Apply flow, external record store, fixed check execution, independent verifier process boundary, parent assurance graph and completion audit. Executor and Verifier processes are deterministic protocol fixtures. Their responses exercise mechanics and evidence binding; their passing judgments do not establish AI semantic quality or genuine independent understanding. Existing historical experiment outcomes remain unchanged.

- [Audit regressions](../../internal/host/canonical_controller_audit_test.go) require configured full observation, account for all source Projections when evidence is absent, retain exact exclusions, detect inventory and selected-byte changes, and report a mixed active/missing Projection set without treating old passing results as closure. The no-evidence case also checks that audit neither invokes the configured actor nor initializes the ledger.
- [Model-change cycle](../../internal/host/canonical_controller_model_change_test.go) changes an actual canonical UseCase purpose, keeps target bytes untouched until Apply, rejects old evidence for the new source, routes work to both representations and closes only with new source/model/result bindings. The Markdown output must contain the changed purpose.
- [Unchanged-intent repair cycle](../../internal/host/canonical_controller_operating_test.go) first verifies and closes both representations, then records a failing leaf judgment. The audit stays incomplete, the repair task binds that exact failed result, and the normal Executor/Apply path changes the leaf artifact bytes while preserving the Markdown artifacts. Fresh local and parent verification closes the scope; the old failure remains in the ledger. The repair actor supplies deterministic test bytes, not a demonstrated semantic fix.
- [CLI boundaries](../../internal/host/cli/canonical_audit_test.go) require fixed source revisions and runtime/configuration inputs, reject mutation and selection arguments even when explicitly empty or false, dispatch through the controller, and reserve exit 0 for complete audit reports.

The inventory/byte readback and ledger recheck detect instability observed during the audit. They are optimistic consistency checks, not an operating-system lock over the repository. A report describes its bound observations; it does not remain valid indefinitely while another actor changes files.

## Live protocol pilot

The [sanitized pilot record](../../experiments/live-operating-pilot/result.json) retains three frozen attempts. It covers one synthetic Commerce area with two Projections: live .NET implementation and a deterministic Markdown representation. This is iterative engineering evidence, not a first-pass success or a general reliability result.

The baseline on source `73728a1` audited complete and recorded 120 behavior calls with no failures; that source already forwarded exact persisted repair IDs and findings in the Host context. The first changed-source Executor response modified the handler but omitted its existing project file; Host correctly escalated `artifact.omitted-owner`, so the attempt stopped before Apply or semantic verification. Source `88eae46` exposed the owned paths and instructed the Executor to return a complete candidate. The second attempt materialized the changed representation. Its first leaf Verifier envelope had `role: null` and was rejected; one bounded same-input transport retry returned the accepted semantic failure. Its 80-call negative holdout had 10 failed checks, all for the controlled quantity-zero defect. The repair findings were bound exactly, but the repair Executor also returned `role: null`; the bridge rejected it and no repair was applied. Source `0567da8` clarified that role comes from `invocation.request.role`. This ambiguity warranted correction, but the evidence does not prove it was the sole cause of either malformed envelope.

The third attempt used the corrected source/runtime and the actual retained faulty artifacts, without resetting the ledger or rewriting earlier responses. A fresh verification reproduced the semantic failure; one ordinary repair consumed the exact persisted findings, changed only `src/Commerce/CreateOrderHandler.cs`, and preserved `Commerce.csproj` and the Markdown output. Apply produced evidence revision `f14198e90f5b0299cb1b0c4aa7b5af8d236546e2`. Fresh verification passed for source `bd06f90c9afcc80f53f5c70d355b39266b6c4331`; the final audit was complete for both Projections with no findings, exclusions, or next steps. A separate 80-call holdout passed with zero failures while the source stayed unchanged. The sanitized record includes the Verify and Audit digests and attempt-specific receipts.

This demonstrates the narrow F1 failure-to-repair transition once, including unchanged accepted intent during repair and preservation of unaffected output. It does not establish a repair success rate or close the larger F1–F3 workflow. The native Codex CLI rejected the requested model at authentication preflight, so actors ran through an experiment-only file queue using fresh collaboration forks and the Host protocol; shared files/tools were not an OS sandbox. Provider version and token usage were unavailable. The controlled quantity-zero defect was deliberately introduced by the harness and is not attributed to the agent.

## Reproduction and interpretation

Run the focused audit and operating tests from this checkout with the contribution toolchain:

```powershell
go test ./internal/host ./internal/host/cli -run 'Test(AuditCanonicalController|CanonicalControllerAudit|CanonicalControllerOperating|CanonicalControllerModelChange)' -count=1
```

The fixtures create isolated temporary repositories and external ledgers using the caller's local filesystem authority. They invoke local test-helper processes and declared Go checks; they do not contact a model provider. On Windows the fixture parent must permit the current process to create and clean up temporary directories. A sandbox access refusal is an environment failure, not a passing test.

The focused suite does not replace the required exact-candidate contribution checks in [CONTRIBUTING](../../CONTRIBUTING.md), including fixed check/context/impact and configured verify. Final gate results must name the tested commit; an older fixture pass is not an exact-current-source gate.

## Remaining product evidence

This is a finite controller capability, not a continuously running autonomous organization. The live pilot is one bounded synthetic case; the larger two-area/shared-rule checkpoint, repeated practical tasks, long-running resumption, and comparison with ordinary agents remain open. Full source gates are reported separately for each exact candidate; the pilot is not a substitute for them. The audit accounts for declared Projections and configured targets. It does not prove that the author chose a sufficient model, that arbitrary prose was interpreted correctly, or that undeclared obligations were discovered.
