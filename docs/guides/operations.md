# Operations

Current source provides canonical v1 read-only preparation, a guarded generated model view and `apply`. [Project operations](project-operations.md) is the command reference; [distribution](../../packaging/README.md) owns separate maintenance packaging and publication. Dated evidence retains its original source/environment scope in the [production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md).

## Authoring and checking

Use the [Project workflow](project-workflow.md). Keep intended behavior in canonical Definitions, exact realization ownership in the registry and operational declarations outside the model. A commit binds a specification revision without authenticating human approval. Check actual structural preparation and the owning Package context before edits. Implementation-only changes need no artificial intent change.

The complete census includes Git-ignored files. Separate architecture and module gates check explicit dependencies, actual v1 registry facts, selected byte pins and literal workflow command scalars. [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change) owns the current gates. These are source checks, not provider or owner acceptance.

## Rendering and controlled writes

`docs` previews the readable canonical view. Use its nested `view.previewDigest` with `--write --expect` after review. `contentDigest` identifies rendered bytes; `canonicalModelDigest` identifies model meaning. The independent documentation guard binds actual sources and target bytes/mode/existence, rejects protected inputs and historical writes, rechecks the plan and reads back actual bytes. Retain completed paths after any partial failure.

Apply (Agent sessions, container checks, candidate workspaces and guarded artifact publication) is implemented and pending owner acceptance; run it only with explicit authority, as [project operations](project-operations.md#apply) describes. The guarded document write is not Apply. Multi-file/interrupted effects must not be assumed transactional or safe to replay blindly.

[Project operations](project-operations.md#generated-view-and-recovery) states which sources `docs --write` accepts.

## Troubleshooting

| Observation | Response |
| --- | --- |
| Structural check reports an unaccounted file | Register its exact owner/Definition basis or make an exact explained ignore decision; protected inputs cannot be ignored. |
| Known model/control failure | Read the actual diagnostics and repair the authored input before a new preview. |
| Incomplete acquisition or metadata | Preserve observed facts and diagnose the unavailable bytes, budget or source binding. |
| Preview digest changed | Review a fresh source/target-bound preview; never reconstruct the expected token. |
| Document write reports completed paths and an error | Inspect those actual effects and current sources before deciding recovery. |
| Fixed preview refuses current external sources | Use historically bound inputs or an explicit working preview; do not silently substitute today's bytes. |

## Release operations

The owner retired the old releases and binary downloads on 17 October 2026 while preserving required evidence. The Git tags no longer exist on GitHub; the [retired release mapping](../validation/retired-releases-20261017.json) records the commit behind each tag and the [archive index](../archive.md) lists the pinned pre-reset commits. The [retirement record](../validation/distribution-retirement-20261017.md) holds captured release identities and attestations. No binary release of the current product is offered; use [current source](../../README.md#run-the-current-source).

Local `markitect-release package`, `bundle` and `install` remain independent maintenance tools; see [distribution](../../packaging/README.md#current-source-release-tooling). Local packaging does not publish. The old download-based benchmark and WinGet validation workflow are retired with their fixtures and manifests.

A future publication needs explicit owner authority, a fresh version tag, a clean reviewed full commit, passing source and Linux gates, exact workflow/run/attempt identity and matching bundle/provenance/asset digests. The earlier publication tool was removed from Main and stays in Git history; a release restores it or writes a new one, reviews its read-only plan before publishing, and verifies the immutable release and every attached asset. Never overwrite or reuse a historical tag. Add download instructions to the README only after that release exists and is verified.

CI transport is bounded: Go test artifacts remain for 14 days to support the flake report; commit-scoped source archives remain for 7 days. Named evidence, failed diagnostics and artifacts needed by current work are excluded from manual historical cleanup. Durable summaries, public attestations and source references belong in Git; temporary artifact URLs are not a permanent evidence store. [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change) owns source gates.
