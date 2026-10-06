# Standard operating model: executable closure contract

This checkpoint adds a read-only completion audit to the existing source-only canonical controller. The [design](../design/standard-operating-model.md) owns its contract; [Usage](../usage.md#declared-scope-completion-audit-source-only-alpha) owns the command; the [roadmap](../implementation-plan.md#standard-operating-model-checkpoint) owns current status. The implementation starts from source candidate `5808ea586979cb48e20f94157b6ccfa9ed318c60`; it does not change the published release.

## What the executable coverage establishes

The Host tests use the real canonical loader, .NET and Markdown Projection Modules, proposal/execution/Apply flow, external record store, fixed check execution, independent verifier process boundary, parent assurance graph and completion audit. Executor and Verifier processes are deterministic protocol fixtures. Their responses exercise mechanics and evidence binding; their passing judgments do not establish AI semantic quality or genuine independent understanding. Existing historical experiment outcomes remain unchanged.

- [Audit regressions](../../internal/host/canonical_controller_audit_test.go) require configured full observation, account for all source Projections when evidence is absent, retain exact exclusions, detect inventory and selected-byte changes, and report a mixed active/missing Projection set without treating old passing results as closure. The no-evidence case also checks that audit neither invokes the configured actor nor initializes the ledger.
- [Model-change cycle](../../internal/host/canonical_controller_model_change_test.go) changes an actual canonical UseCase purpose, keeps target bytes untouched until Apply, rejects old evidence for the new source, routes work to both representations and closes only with new source/model/result bindings. The Markdown output must contain the changed purpose.
- [Unchanged-intent repair cycle](../../internal/host/canonical_controller_operating_test.go) first verifies and closes both representations, then records a failing leaf judgment. The audit stays incomplete, the repair task binds that exact failed result, and the normal Executor/Apply path changes the leaf artifact bytes while preserving the Markdown artifacts. Fresh local and parent verification closes the scope; the old failure remains in the ledger. The repair actor supplies deterministic test bytes, not a demonstrated semantic fix.
- [CLI boundaries](../../internal/host/cli/canonical_audit_test.go) require fixed source revisions and runtime/configuration inputs, reject mutation and selection arguments even when explicitly empty or false, dispatch through the controller, and reserve exit 0 for complete audit reports.

The inventory/byte readback and ledger recheck detect instability observed during the audit. They are optimistic consistency checks, not an operating-system lock over the repository. A report describes its bound observations; it does not remain valid indefinitely while another actor changes files.

## Reproduction and interpretation

Run the focused audit and operating tests from this checkout with the contribution toolchain:

```powershell
go test ./internal/host ./internal/host/cli -run 'Test(AuditCanonicalController|CanonicalControllerAudit|CanonicalControllerOperating|CanonicalControllerModelChange)' -count=1
```

The fixtures create isolated temporary repositories and external ledgers using the caller's local filesystem authority. They invoke local test-helper processes and declared Go checks; they do not contact a model provider. On Windows the fixture parent must permit the current process to create and clean up temporary directories. A sandbox access refusal is an environment failure, not a passing test.

The focused suite does not replace the required exact-candidate contribution checks in [CONTRIBUTING](../../CONTRIBUTING.md), including fixed check/context/impact and configured verify. Final gate results must name the tested commit; an older fixture pass is not an exact-current-source gate.

## Remaining product evidence

This is a finite controller capability, not a continuously running autonomous organization. A live Executor/Verifier run on meaningful requirements, repeated practical tasks, long-running resumption and retry behavior, and a comparison with ordinary agents remain separate work. The audit accounts for declared Projections and configured targets. It does not prove that the author chose a sufficient model, that arbitrary prose was interpreted correctly, or that undeclared obligations were discovered.
