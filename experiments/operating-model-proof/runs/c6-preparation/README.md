# C6 five-target proof preparation

Status: provider-backed execution blocked before process start by automatic review. No Executor or Verifier/provider invocation occurred. The original packet had incorrect source fixture paths and is invalid for approval. Use the corrected metadata-derived scope packet: [corrected blocked Codex proposal scope](blocked-codex-propose-scope-corrected.md). The original rejection record remains at [historical packet](blocked-codex-propose-scope.md).

## Frozen inputs

- Host implementation/build source: `17ad4205f15efbf7ae6ccd04b2d1085201602cd3`.
- Windows executable: `markitect-workflow-modes.exe`, SHA-256 `8d179620a679fdfa04c92888ce2639755aa2bd91d15b7531ed709e17cb64bf76`.
- Build receipt SHA-256: `4a7ca77cfbfb83c1e1b5e3f98b755f1de128d072372499438cd9e2d2ab0f3375`.
- Proof-input commit: `f256ab776987c808fb8d537758c2dfd2b1831352`, an additive fixture commit carrying five exact single-Projection Source configs. These configs are necessary because the controller has no Projection selector and assurance roots do not narrow proposals.
- All five frozen-binary `canonical --action request` calls at the proof-input commit returned `status: bound`, each with exactly the same four Definition identities: Rule `retain-source-authority`, Process `review-workflow-change`, Responsibility `workflow-owner`, and Gate `source-review`. The check action was read-only and made no provider calls.

## Independent target configurations

| Config | Projection | Module | Request digest | Target prefix | Expected artifact and mode |
| --- | --- | --- | --- | --- | --- |
| `canonical-codex-only.yaml` | `workflow-codex` | `markitect-agent-rules-codex` | `sha256:04b545214125293a2b1859c23cbeeb5047bc89845966b15d7b65925c9d5d07d6` | `agent/` | `agent/AGENTS.md`, 100644 |
| `canonical-claude-only.yaml` | `workflow-claude` | `markitect-agent-rules-claude` | `sha256:6432364f92fdae05c84adf6c49bb5e41b471183ea065f09ef0cbeb8550bf37fc` | `assistant/` | `assistant/CLAUDE.md`, 100644 |
| `canonical-markdown-only.yaml` | `workflow-markdown` | `markitect-markdown` | `sha256:42a09fe0a877cf983b6f421d984e6096a02994ff7d41bf6aed803d50e13f8689` | `docs/represented/workflow.md` | `docs/represented/workflow.md/index.md`, 100644 |
| `canonical-githooks-only.yaml` | `workflow-githooks` | `markitect-git-hooks` | `sha256:8e1ebb32b728c208882950abddd1932556c669a5ecfdc3a1a35c0fc865828c6d` | `.githooks/` | `.githooks/pre-commit`, 100755 |
| `canonical-azurepipelines-only.yaml` | `workflow-azurepipelines` | `markitect-azure-pipelines` | `sha256:f11b5d9f73482ab9a7ec02c731738e63b3439b39eaa9d1c16bb5884a52270aef` | `ci/azure/` | `ci/azure/azure-pipelines.yml`, 100644 |

Each Source config contains the same four canonical Definitions and workflow Schema, Foundation 1.2.0, one target's four explicit ProjectionPolicies, one Projection Definition, one exact module pin/binding, and the same named `canonical-workflow-check` argv. Four source items compose each single target artifact; each canonical item is projected independently to all five target artifacts.

## Execution controls for the next phase

- Use a separate disposable clone per single-Projection config, all rooted at proof-input commit `f256ab7…`; target output bytes from one run cannot enter another run's selected target snapshot.
- Freeze and record the CLI build and canonical source revisions separately. Keep each run's record store and private logs outside the clone, use exact scope/root configuration, and do not publish raw runner logs.
- For each Windows-compatible target: perform read-only proposal, deterministic Module preparation, save/review exact digest-bound candidate, guarded apply, and Verify at the object-only immutable EvidenceRevision returned by Apply. Do not synthesize a Git HEAD/index commit to stand in for evidence. Verify must execute the fixed `canonical-workflow-check` and the separately configured fresh Verifier; retain sanitized receipts and digests.
- Negative control: in a disposable clone add one unowned sibling under a target prefix and confirm proposal/apply refuses it without mutation. In a separate clone explicitly exclude a non-overlapping manual sibling with a reason; confirm it is reported as excluded and remains byte/mode-identical after successful materialization.
- Windows Git cannot establish executable mode or shell execution for `.githooks/pre-commit`. A Linux checkout with `core.filemode=true` is required to prove Git mode 100755 and actual hook invocation. Until that run succeeds, report the Git Hooks execute/mode claim as blocked.
- Deterministic Module output is not an Executor invocation. Do not claim the five target exercise used an Executor unless a configured real Executor invocation actually occurred. No Provider semantic sufficiency or human acceptance claim follows from fixed checks or a fresh Verifier receipt.

