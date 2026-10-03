# Parallel development baseline

This preparation starts from integrated `main` commit `e9550f5c91430c6a65cbbd4ffcdbb49b48272537`, the merge of [PR 59](https://github.com/Glacius-Labs/Markitect/pull/59). It follows the real-code pilot in PR 57 and policy-failure analysis in [PR 58](https://github.com/Glacius-Labs/Markitect/pull/58). No unfinished product migration was identified by the preparation audits. The comparison remains inconclusive; neither its negative findings nor historical pilot evidence is rewritten.

The **PARALLEL DEVELOPMENT BASELINE** is the exact integration commit of this preparation, including these documents, after both quality jobs pass. The coordinator records that full commit and its gate URLs when integrating/dispatching; implementers must copy the full commit into their assignment and branch from it, rather than resolve moving `main` later. This document cannot embed its own future integration hash. Before integration this is a preparation candidate, not an established baseline.

The current published release remains [v0.12.0](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.12.0), source `83b715c415e39af79715cd8a40b3c79237a4c850`. It includes bounded `same-target`. Integrated source additionally supports explicit read-only policy-failure analysis; that option remains unreleased. The matched [comparison](../validation/agents-md-vs-markitect.md) records workflow observations, not an advantage in total upkeep or productivity. No release, tag, historical package or distribution metadata is changed here.

## Required validation

PR 59's commit above is the source audit base only, not the future dispatch baseline. Every assignment must be stamped with the full verified preparation integration SHA. A workstream's references to PR 59 describe inspected implementation, not permission to start from a tree missing these contracts.

Use the actual [CI workflow](../../.github/workflows/ci.yaml) and [contribution checks](../../CONTRIBUTING.md), not a shorter invented gate. Both Windows and Linux quality jobs cover module verification, Go tests/vet/build, authoring benchmark fixture validation, all nine generated schemas, executable examples/format/model/context, source packaging and standalone bootstrap commands. Windows also replays onboarding; packaged smoke includes projection reconciliation, the external .NET reference adapter and preview/write initialization. Record candidate and integration SHA separately. A main source artifact is temporary transport, not a product release.

PR 59's candidate `9b0ed1b2dc1d2b5e94ada881f6b068e7fae8f794` passed [CI 37141270503](https://github.com/Glacius-Labs/Markitect/actions/runs/37141270503) on both platforms, including package/bootstrap gates. Local study hash/archive verification also passed. A local deep-path package replay reached bootstrap tests but failed spawning the installed Go toolchain; this environment observation is separate from hosted success and is not claimed as a passing local bootstrap gate. Preparation gets fresh commit-bound gates before it becomes a dispatch baseline.

## Scope of preparation

The change consolidates canonical invariants, audits current seams, preserves the supplied proposals and prepares implementer contracts. No Core primitive, adapter implementation, discovery command, adopter policy, automatic inference/adoption or background runtime is added. Review the complete diff for exact file ownership, working-tree cleanliness, generated-artifact drift, public-content paths, cross-links and consistency of proposed/source/released status.
