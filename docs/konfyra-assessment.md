# Konfyra Markitect RC3 pilot assessment

Assessment updated: 2026-09-30. This is the current RC3 pilot assessment; the Konfyra delivery dossier `01a0f27a-a479-76bf-886e-6583bbe49200` remains the owner of live delivery and acceptance.

## Current position

| Subject | Fixed identity | Evidence and state |
|---|---|---|
| Current Konfyra candidate | `48d6711fde761de0bad144daf4bfdf8be81b8151`; target `413c0f9cc66adcfb718cefa5e68cdd0092076a41` | Fixed-snapshot Windows verify passed (snapshot `d9113100efe2f909b5b2fbccd73e34e05c2b1d02a432553405817db5f922fd52`); architecture suite 226/226 and both hosted gates passed. |
| Consumer delivery | [Draft PR 3007](https://dev.azure.com/consiliari/Consiliari.Konfyra/_git/Consiliari.Konfyra/pullrequest/3007) | [Governance run 26344](https://dev.azure.com/consiliari/Consiliari.Konfyra/_build/results?buildId=26344) and [required PR-policy run 26345](https://dev.azure.com/consiliari/Consiliari.Konfyra/_build/results?buildId=26345) succeeded. Preview `3d122817e27c86eb3cff111221439fa53196d408` targets `413c…`. PR remains Draft. |
| Markitect source distribution | `d3d6f4e55588f966849de9bf0f71fd3a9e4949f8`, `main`, CI run `36747050405` | Windows/Linux CI passed; downloaded four-file package bootstrap, authoring and example checks passed. Archive SHA-256: `eeb26389b9a8dcdd3954abe209ae8dcfd44ec64643b03dbde59afde07a80efd2`. |

Markitect source ownership is now in private GitHub repository `Glacius-Labs/Markitect`. The candidate has passed the recorded local and hosted technical gates. This does not establish human acceptance, PR completion or Cockpit migration.

## Migration and release evidence

The parity record for migration candidate `5bc189bdc83cdc78ca6dad45b90a32c21ec400ad` compares current master `413c0f9cc66adcfb718cefa5e68cdd0092076a41`: 64 legacy definitions, 66 mechanisms, zero missing, two added, zero moved and zero diagnostics. It also records three text, 35 dependency, two declared-file and two other spec changes, plus one Project policy change. Additions are the documentation-review Contract and review-snapshots Text. Parity requires field-by-field review and does not assert semantic equivalence. The independent mapping review found no blocker in scope, traced ownership and relationships, and found zero provider-metadata changes with provider instruction bodies unchanged; it includes the generated Codex and Claude authoring-skill descriptions.

The final candidate's fixed-snapshot verification is Markitect `0.1.0-rc.3`, `provisional: false`, context digest `4841023ab60d6cf7ef2e81b703642d40dae0db34909dc76fa81556f2a788cfe2` and the snapshot in the table. Independent delta reviews found no issue with (a) the `373…` changes: `.gitattributes` LF rules and C# exact leading provenance-marker stripping, preserving suffix/body; and (b) the sole `373…`→`48d…` line changing temporary test-path identity from `Guid.NewGuid()` to `Guid.CreateVersion7()`. Candidate `48d…` has no migration-resource, Project, Markitect-tool-input or generated provider-output delta from `373…`.

For the EOL change, the `373…` generated files were clean on a second render. A focused probe deleted and recreated two pre-existing CRLF files: index/worktree bytes were LF and matched the correct HEAD blobs; there was no content change or staged diff after index refresh. This corrects the earlier stale-stat observation; it verifies the targeted checkout behavior, not all possible environments.

Repair history: PR pipeline 197 / run `26333` failed at `5bc…` because the C# architecture check included the generated leading provenance marker when comparing the expected Agent instruction body; the Python renderer strips that prefix. The marker helper was corrected and focused tests passed. Run `26341` then found a test fixture using `Guid.NewGuid()`. That one-line fix is in `48d…`; local architecture tests passed 226/226. Hosted governance `26344` and required PR-policy `26345` both succeeded.

## Authoring route and bounded review

The candidate-specific authoring review for `48d…` found no material issue in its bounded procedure question. It records the fixed candidate/context/snapshot/tool digests and checked the compiled authoring skill, workflow and candidate-binding instructions; unchanged ownership/provider sections were carried forward from the preceding report because the context digest matched. The older `373…` review was correctly `review-required` at `48d…` because of the new undeclared C# input. The same-`48d…` reuse record is `reusable`. These advisory results do not assess implementation correctness, the migration, runtime behavior, hosted CI or human acceptance; reuse does not make a semantic-reviewer call, and authoring/orchestration token use was not measured.

A separate practical exercise on branch `exercise/markitect-parallel` used candidate `0e98cabc64a01aa8195697432468a4b4b243ee63`, based on `5bc…`. Through the generated Konfyra authoring route and pinned core authoring/find/explain commands, it clarified the existing LSJV `manage-feature-flags` Skill. Exactly two YAML/Markdown files changed, affecting only `lsjv/Skill/manage-feature-flags`; full governance and fixed-candidate `verify` passed. The compiled General authoring context digest remained `4841023a…cfe2`. Same- and different-commit `review --evidence` checks both reported reusable against the copied real baseline report, with measured durations of 1,536 ms and 1,607 ms; reuse did not invoke a semantic-reviewer call. This does not establish that the authoring/orchestration workflow used no model work; token use was not measured.

Follow-up invalidation probes on this exercise found that a shared-Rule-only change at `8526f862bcbd47786f9a7cbde49bb34f60309361` changed compiled context and required review. An unknown-input-only change at `22d984970d849dbdb90d0441aa0eb26ad43b135e` left compiled context unchanged but changed full-inventory impact and required review. The exercise reported a 94,134-byte General context. These are separate worktree results, not evidence for the release candidate or its required PR check.

Root WorkSync's offline validation of the current merged data reported 216 records and `valid=true`. This is a separate offline data check, not provider runtime or acceptance evidence.

Commit-specific context addresses the consistency-review problem by keeping an in-flight review on one candidate while work continues in another checkout. After integration, use `impact` and compile context for the new fixed SHA. Reuse remains conditional: changed configuration, inventory, dependency closure, undeclared files or external inputs may require a new review. No time or token savings have been established.

The downloaded RC3 bootstrap, `authoring` command and minimal example were smoke-checked as part of the source distribution. The independent authoring-context report itself notes that the reviewer did not run the bundled core `authoring` command while conducting its semantic review; the package smoke check is separate evidence.

## Open acceptance boundaries

- PR 3007 remains Draft: no human acceptance or completion, promotion to `master`, or Cockpit migration has occurred.
- Claude authentication was rechecked as logged out; there is no Claude runtime acceptance evidence.
- Structural checks, migration review and bounded authoring reviews do not prove every prose dependency is declared or provider workflows behave equivalently at runtime.
- No measured token-cost or sustained time saving is available. Practical evidence supports fixed-context reviews and conditional reuse only.

## Historical RC2 evidence

The initial RC2 migration candidate `705f90b072938324dfe7940f9c815ff19d3b79b4` was compared with the then-used baseline `74d18c56e20dbb272dc085a389bfdaf291ee4562`. It established the 64-definition migration and its local checks. A later Go-bootstrap integration candidate `1b56627b1e1d9eb58db87b210d5c59e069b8a5a4` replaced the Markitect-owned Python bootstrap while keeping the RC2 tool archive; its Windows and isolated Linux governance evidence is historical and does not verify the current RC3 candidate.
