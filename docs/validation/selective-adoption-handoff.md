# Selective preparation and Copy Me validation

## Scope

This source iteration starts at main `93181bb9bc1af0e663b3daa1a4ff772822320307`, after PR 72. Published v0.12.0, historical examples/packages/releases and the negative adoption-pilot findings remain unchanged. The [handoff design](../design/selective-adoption-handoff.md) owns the new versioned boundary; [Usage](../usage.md#selective-adoption-preparation-and-copy-me-unreleased-source) owns commands. This is acquisition/storage/reference validation, not a discovery-quality or productivity study.

## Executable evidence

| Control | Observed behavior | Limit |
|---|---|---|
| Source acquisition spy | Two exact selected paths cause two literal metadata lookups and one batch containing only those two blob IDs. Missing/tree/symlink/gitlink/glob/duplicate/alias scope produces no blob-reader call. Explicit selected `vendor` evidence is eligible rather than silently filtered. | Logical Git requests; not instrumentation of internal Git pack/delta disk reads. |
| Fixed-source integrity | Commit selectors/abbreviations are refused; loaded bytes must match selected Git blob OIDs. Separate clone and linked-worktree identities are tested. | Canonical locations/layout/full commit and bytes, not repository-owner authentication or an unforgeable directory incarnation. |
| Storage | Preview creates no workspace. Exact expected digest permits exclusive external creation. Existing destinations, source overlap, unsafe spelling, links/reparse paths and malformed manifests are refused. An injected failure reports created paths and leaves partial state for deliberate recovery. | Caller-controlled local filesystem, not an OS sandbox or retention service. |
| Real CLI fixture | Two selected documentation blobs reach the external handoff; private/unselected sentinel text is absent from the preview/capture. Support, counterexample, qualifying evidence, conflict, uncertainty, coverage and a separate supplied decision validate. `adopted: false` and `unauthenticatedReviewer: true` remain explicit. Source working tree stays unchanged. | Synthetic rule/decision, no authenticated owner approval or discovery accuracy. |
| Freshness | Modified source evidence fails handoff validation; candidate/queue/handoff raw-byte changes stale prior decision bindings. Updated coverage claims retain original preparation coverage and stale old decisions. | An old fixed capture/decision remains a valid historical record for its own bytes; no ambient latest-branch or time-based freshness is inferred. |
| Unselected-only change | A new full commit changes handoff/selection identity while selected snapshot content digest remains stable. A working-tree change does not alter capture at the old commit. | Owner review is needed for a new capture; no implicit revision lookup or scope extension. |
| Authority | A validated `accept` record performs no Project/Domain/pin/check/adapter/provider/document/source mutation. Evidence requests are returned as claims and trigger no acquisition. | A later explicit project-owned adoption review is still necessary. |

`go test ./examples -run '^TestSelectiveAdoptionCLI$' -count=1` builds and runs the CLI; `MARKITECT_ADOPTION_BINARY` runs the same fixture against a source-package binary in both CI platforms. The existing greenfield Init tests remain unmodified and run with the app/CLI suite. Required schema, examples, module/vet/build, source packaging and standalone-bootstrap gates remain in normal CI, with the new packaged adoption replay added.

## Public Konfyra-style replay

The replay selected **one path**, `docs/validation/parallel-wave-konfyra.md`, at the full public Markitect commit `93181bb9bc1af0e663b3daa1a4ff772822320307`. This is the already sanitized inventory, not a private checkout or its underlying application code. No private path, source wording, credential or customer data was acquired. The actual command was the repository's `examples/selective-adoption/replay.go` with an explicitly built candidate binary and this public checkout.

It completed preview → digest-bound capture → Copy Me queue/candidate/decision validation, with no Project or ContextRun requirement. The observed result was `copyMe: validated`, `reviewer: unauthenticated`, `adopted: false`, and explicit uncertainty about report truth and representativeness. No captured report body was printed. The reproducible local handoff identity was `3461343e9da47d7d261fbd93771ebb9de4d253e09b6e69ca2191cd446545e6be`; it includes local repository identity, so another checkout location intentionally produces another identity.

The replay deliberately uses temporary storage and its declared test retention removes that fixture after completion. Product preparation itself never rolls back or deletes partial/completed workspaces. Source-package/bootstrap smoke also passed locally on Windows, including dispatch/help for both new commands; exact candidate Windows/Linux evidence is attached to the integration PR, not inferred from a local result.

This proves that the handoff can transport the selected public mature-engineering inventory shape and validate bounded interpretation records. The interpretation and reviewer are synthetic. It does not prove substantive Konfyra discovery, real owner acceptance, a complete constitution, adoption, upkeep payback or user benefit.

## Completion decisions

1. Preparation requests only selected Git blob contents through the tested acquisition seam.
2. Repository identity binds supplied stable ID, canonical root/Git/common directories, object format and full commit separately from content.
3. Exact paths/reasons/exclusions, modes, byte hashes and selected snapshot digests participate in selection/capture/handoff identity.
4. The external workspace persists only `handoff.yaml` and `evidence/<repository>/<path>` selected bytes; dossier records are separately supplied.
5. Different selection, revision, repository binding, selected bytes/modes or scope/privacy/retention/coverage claims creates a different handoff. Tampering is rejected. Original fixed evidence is not silently reinterpreted.
6. Copy Me cannot acquire more evidence; requests require a new owner-supplied scope.
7. No Project or ContextRun is needed. Optional run/report/build/context identities bind supplied selected artifacts, without recompilation or authentication.
8. IDs, digests, ranges, stances, references, duplicates/conflicts, uncertainty fields, coverage and frequency shape are deterministic checks. Semantic merit remains outside them.
9. Changed captured evidence or bound handoff/queue/candidate bytes invalidates reuse of a prior decision against those changed inputs.
10. No disposition automatically adopts anything.
11. Existing greenfield Init code and regression tests are unchanged.
12. No Core/Domain/schema/SPI semantic contract changed; the new records are standalone adoption infrastructure.
13. Preparation and interpretation now have separate code ownership behind one closed handoff.
14. A separately owner-approved Konfyra scope can now yield an owner-reviewed candidate. The public replay is not that approval or adoption.
15. The next missing step is real owner review of a bounded candidate constitution and a separate project-owned adoption diff, followed by measuring total setup/maintenance and human interventions against the simpler existing guidance/tests baseline.
