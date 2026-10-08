# Markdown projection independence audit

**Source baseline:** `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`

**Scope:** the built-in `markdown` projection and its boundary with human-owned project artifacts. This is a source-level test/evidence slice; it does not change product behavior or shared renderer/reconciliation code.

## Finding

No projection defect was found in the audited ownership path. With `markdown` explicitly selected, `internal/render` derives resource views, Domain views, and navigation from the validated graph and records canonical owners for generated paths. Captured files are used by prose navigation to resolve links and distinguish existing ordinary files from generated companion aliases; their Markdown bodies are not copied into generated views. A prose link remains navigation and does not add a graph relationship.

The new `internal/render/parallel_wave_docs_test.go` negative control changes the body of a linked human-owned README while keeping its path fixed. It asserts that generated resource bytes and owner identity remain stable across repeated generation, that the human body is not copied, and that the prose link creates no typed relationship. This pins the independence claim at the renderer seam without asserting that the file is irrelevant to link resolution.

## Ownership and stale-output evidence

- `MarkdownViewPaths` and `DomainMarkdownPaths` reject case-folded generated-path collisions. `GenerateWithOwners` rejects generated outputs that collide with canonical resource paths, also case-insensitively. Existing tests cover same-view source collisions, malformed Area traversal, and a case-insensitive source/output collision.
- Existing app-level projection tests cover an unmarked human file at a desired output path: planning is incomplete, contains no replacing operation, apply fails, and the file bytes remain unchanged.
- Existing app-level output checks report a marked generated file with no current output owner as stale. Reconciliation refuses incomplete/stale application so removal stays a deliberate migration decision.
- Generated Markdown remains derived from canonical Markitect resources and Domain definitions. It is not read as canonical input to another adapter.

These ownership and stale-output behaviors span the shared `internal/app` projection planner/writer. They were inspected as existing contracts and were not changed in this slice.

## Coupling and limitations

The Markdown helper/test seam is isolated in `internal/render`; the path inventory, collision check, renderer dispatch, and local reconcile lifecycle are shared with Codex and Claude targets and remain coordinator-owned. No Core change, adapter chain, generated-document input, or source-structure inference is warranted by this audit.

Ownership is based on Markitect's generated marker and fixed snapshot/path checks. A marker is a detectable ownership convention, not proof of authorship. These tests establish deterministic local projection behavior for the exercised graph and file cases; they do not establish semantic correctness of prose, human acceptance, correctness of every possible path/configuration, or runtime behavior of unrelated adapters.

## Focused validation

```powershell
go test ./internal/render -run TestParallelWaveMarkdownProjectionIgnoresHumanDocumentBody -count=1
go test ./internal/render ./internal/app -run 'Test(MarkdownViewsRejectDeterministicSourceAndOutputCollisions|GenerateRejectsCaseInsensitiveSourceOutputCollision|ProjectionPlanReportsUnmanagedCollisionBeforeApply|CheckOutputsRejectsOrphanedAggregateRuleAdapter)$' -count=1
```
