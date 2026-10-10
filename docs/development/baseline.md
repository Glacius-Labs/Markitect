# Parallel development baseline

This is the preserved PR #60 preparation baseline. Its release labels, source paths and dispatch conditions describe that wave. For a new assignment, select the current integrated source and record its full commit as described in the [parallel-work guide](parallel-work.md); use the [roadmap](../implementation-plan.md) and [Product Readiness backlog](../work-items/product-readiness/backlog.yaml) for current status.

This preparation starts from integrated `main` commit `e9550f5c91430c6a65cbbd4ffcdbb49b48272537`, the merge of [PR 59](https://github.com/Glacius-Labs/Markitect/pull/59). It follows the real-code pilot in PR 57 and policy-failure analysis in [PR 58](https://github.com/Glacius-Labs/Markitect/pull/58). No unfinished product migration was identified by the preparation audits. The comparison remains inconclusive; neither its negative findings nor historical pilot evidence is rewritten.

The **PARALLEL DEVELOPMENT BASELINE** is the exact integration commit of this preparation, including these documents, after both quality jobs pass. The coordinator records that full commit and its gate URLs when integrating/dispatching; implementers must copy the full commit into their assignment and branch from it, rather than resolve moving `main` later. This document cannot embed its own future integration hash. Before integration this is a preparation candidate, not an established baseline.

The immutable integration identity is the `mergeCommit.oid` of [PR 60](https://github.com/Glacius-Labs/Markitect/pull/60), once it is merged and its commit-bound main gates pass. This lets a fresh implementer recover the full baseline without chat history:

```text
gh pr view 60 --repo Glacius-Labs/Markitect --json state,mergeCommit
```

Require `state: MERGED` and a full commit ID; verify the main quality checks on that exact ID. Before those conditions hold, do not dispatch implementation from this candidate. Later work may use a newer integrated dependency only when the assignment explicitly records it.

The current published release remains [v0.12.0](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.12.0), source `83b715c415e39af79715cd8a40b3c79237a4c850`. It includes bounded `same-target`. Integrated source additionally supports explicit read-only policy-failure analysis; that option remains unreleased. The matched [comparison](../validation/agents-md-vs-markitect.md) records workflow observations, not an advantage in total upkeep or productivity. No release, tag, historical package or distribution metadata is changed here.

## Required validation

PR 59's commit above is the source audit base only, not the future dispatch baseline. Every assignment must be stamped with the full verified preparation integration SHA. A workstream's references to PR 59 describe inspected implementation, not permission to start from a tree missing these contracts.

Use the actual [CI workflow](../../.github/workflows/ci.yaml) and [contribution checks](../../CONTRIBUTING.md), not a shorter invented gate. The Linux quality job, which gates pull requests, and the nightly Windows job cover module verification, Go tests/vet/build, authoring benchmark fixture validation, all nine generated schemas, executable examples/format/model/context, source packaging and standalone bootstrap commands. Windows also replays onboarding; packaged smoke includes projection reconciliation, the external .NET reference adapter and preview/write initialization. Record candidate and integration SHA separately. A main source artifact is temporary transport, not a product release.

PR 59's candidate `9b0ed1b2dc1d2b5e94ada881f6b068e7fae8f794` passed [CI 37141270503](https://github.com/Glacius-Labs/Markitect/actions/runs/37141270503) on both platforms, including package/bootstrap gates. Its integrated `e9550f5` passed [main CI 37141545557](https://github.com/Glacius-Labs/Markitect/actions/runs/37141545557). Local study hash/archive verification also passed. A local replay first hit toolchain-cache access and then a deep-path bootstrap process-spawn failure. Replaying the unchanged Windows CI run steps with an accessible cache and shorter temporary package root passed on preparation candidate `9752f85057a853056ad3fdc56aa548456ef3c4ff`, including schema/examples, onboarding, package/bootstrap, both reconciliation paths and init. This is a bounded local environment workaround, not a source repair or a guarantee for arbitrary Windows paths. The final preparation head and integration still require their own hosted commit-bound gates.

## Scope of preparation

The change consolidates canonical invariants, audits current seams, preserves the supplied proposals and prepares implementer contracts. No Core primitive, adapter implementation, discovery command, adopter policy, automatic inference/adoption or background runtime is added. Review the complete diff for exact file ownership, working-tree cleanliness, generated-artifact drift, public-content paths, cross-links and consistency of proposed/source/released status.
