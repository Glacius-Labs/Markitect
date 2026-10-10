# Project-local projections

> This describes the experimental Project-local projection API bundled with v0.14.1. It remains a preview alongside the preserved v0.13.0 Project/Domain contract.

A local projection is a project-owned representation of explicitly selected Markitect inputs. The contract records its sources, target paths, and materializer. Deterministic materializers can be reproduced by the CLI; AI materializers accept a separately prepared candidate whose exact bytes are reviewed before writing. A candidate or a ready plan is not evidence of convergence or human acceptance.

## Register exact configuration paths

Declare one adapter in the Project. Its configuration has exactly two path entries, both exact repository-relative paths: `contracts` and `coverage`.

```yaml
spec:
  adapters:
    - name: representations
      type: local-projection
      version: v1alpha1
      config:
        contracts: projections.config
        coverage: markitect-artifacts.yaml
```

Paths are explicit and normalized; they are not globs. The contract file uses `apiVersion: markitect.example.org/projections/v1alpha1` and `version: "1"`. Each contract names exact semantic sources, a representation, a materializer and mode (`deterministic` or `ai`), exact target paths, and any verification checks. An AI contract's `verificationChecks` must name checks declared in `Project.spec.checks`. Coverage remains the Project's declared artifact accounting; registering a projection does not infer ownership from prose or target contents.

The CLI requires `--config` and `--coverage` on each `projection` action. When the Project has the adapter registered, both values must match its paths exactly:

```powershell
markitect-legacy projection --repo . --config projections.config --coverage markitect-artifacts.yaml --action observe
```

## Observe and plan

`observe` reports the current projection and accounting state. `plan` emits a concrete YAML plan bound to the fixed snapshot, configuration, desired intent, tool and observed files. Save that output unchanged outside the projection targets; apply recomputes the plan and rejects a stale or modified plan.

```powershell
New-Item -ItemType Directory -Force .artifacts/projections | Out-Null
markitect-legacy projection --repo . --config projections.config --coverage markitect-artifacts.yaml --action plan > .artifacts/projections/plan.yaml
```

The plan is a work description, not a claim that targets are converged. Inspect it and any AI materialization record before authorizing writes. The reviewed digest is evidence about the record's exact file bytes; it is not an identity or authentication of the reviewer.

## Apply a reviewed candidate

Apply is working-tree-only and requires a named non-protected branch for Git checkouts, the saved plan, and explicit `--write`. If the projection has AI targets, provide one candidate record containing every exact AI target path and content. Partial AI candidate files are unsupported. The raw SHA-256 in `--expect` is calculated over the candidate record's exact bytes (including whitespace and line endings), not a normalized YAML representation. Review the complete record, then calculate and pass that digest:

```powershell
(Get-FileHash .artifacts/projections/candidate.yaml -Algorithm SHA256).Hash.ToLowerInvariant()
markitect-legacy projection --repo . --config projections.config --coverage markitect-artifacts.yaml --action apply --plan .artifacts/projections/plan.yaml --report .artifacts/projections/candidate.yaml --expect <RAW_RECORD_SHA256> --write
```

`--report` supplies the candidate record for projection apply; it is paired with `--expect`. For deterministic-only contracts, omit both `--report` and `--expect`. Apply writes only declared targets and returns `materialized-unverified`. A successful write is not verification: commit the resulting candidate, then verify that full immutable commit ID.

Writes are sequential, not a multi-file transaction. If a later target write fails, the report includes the exact paths already written and remains `materialized-unverified`; the command reports failure and does not roll back. Inspect the listed paths and current worktree before deciding how to recover. Do not treat a partial write as a successful apply.

## Verify a committed candidate

Projection verification requires a full immutable Git commit ID (40 or 64 lowercase hexadecimal characters), not a branch, tag, or working tree. It compares the declared projections at that snapshot and runs the Project checks explicitly listed in `Project.spec.checks`. Each contract’s `verificationChecks` selects which of those check results is attached as evidence for that contract.

```powershell
markitect-legacy projection --repo . --revision <FULL_IMMUTABLE_COMMIT_ID> --config projections.config --coverage markitect-artifacts.yaml --action verify
```

The ordinary repository verification command also activates projection verification when the Project registers the adapter:

```powershell
markitect verify --repo . --revision <FULL_IMMUTABLE_COMMIT_ID>
```

`markitect check` is for structural, policy, and artifact-accounting validation and can report observed projection state. It does not run the projection's verification checks and cannot establish whether AI-authored contents are semantically correct. An AI target's presence alone is incomplete evidence; declare meaningful project checks and list the relevant checks on the AI contract. Verification runs only commands explicitly declared in `Project.spec.checks` against the immutable snapshot; `verificationChecks` determines which check results support each contract. This is a bounded command list, not an operating-system sandbox; checks execute with the local caller's authority. A passing result is technical evidence for the configured checks, not external acceptance.

Registered target/config paths are opaque even when their contents use YAML and fall within an Area. They cannot contain resources of an active canonical API. A projection cannot introduce new canonical resources. Domain descriptors may be named explicitly as sources (`domain:<apiVersion>/<name>`); Context includes their existing input provenance, while Impact maps changed Domain/config/package inputs to their declared targets without creating graph edges.

Check integrity covers original bytes and regular-file identity, and executable mode on Unix. Windows does not provide this Unix-mode assurance. Extra temporary outputs are not authority; checks still execute with the caller's local permissions.

Impact also closes over explicitly declared contract prerequisites and records `viaContracts` for dependent review targets. This is bounded contract status/review propagation; Context exposes `dependsOn` without automatically including prerequisite sources or creating semantic/context edges.


## Verification strength and ownership

`converged` means the declared fixed checks and integrity/accounting requirements passed. The CLI explicitly reports `verificationBoundary`: checker independence and semantic sufficiency are not proved. A Project can configure a weak, no-op or candidate-authored check; passing that check is correspondingly weak evidence. Check names, argv tokens and file ownership do not prove a wrapper's full dependency chain.

For meaningful assurance, the human check owner must separately retain an independent expected-behavior/check implementation and its inputs outside AI target ownership. The Dispatch proof does this before candidate generation and preserves its exact hashes; it is a property of that measured setup, not an automatic guarantee for arbitrary contract configs. Do not equate agent-authored tests with independent verification. A future check-input/obligation contract needs focused evidence and design; no such DSL is introduced here.

Changed registered contract, coverage or Project activation input includes the corresponding base/candidate contracts with `viaInputs` provenance, even without a direct resource-key match. Coverage changes conservatively invalidate accounting/evidence review; this does not imply every target needs new bytes.

Read-only Host Plan and Observe compose artifact accounting and required native-renderer state. Their top-level status cannot be converged before immutable Verify runs the declared Project checks: matching contract targets are shown separately, and otherwise matching plans remain incomplete with a verification-not-run cause. A successful plan command means a work plan was produced, not that the candidate passed acceptance.

Verify uses the same composed aggregate status in its nested plan, retaining matching per-contract states separately. Accounting/rendering and required-check causes participate in the final plan digest; a failed required check outside a contract’s selected evidence cannot leave the aggregate plan converged.
