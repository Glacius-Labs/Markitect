# Cockpit organization read model, 21 October 2026

Scope: package 1 of the [organization handoff](../development/codex-cockpit-org-chart-handoff-20261021.md), on integrated base `f38e5f2fc319d7668d9cd43104bde34f1ecb39fa`. The browser chart follows in a separate package. This is source validation, not provider or owner acceptance.

`GET /api/organization` reuses the project observation for declared values.
The bootstrap projection consumes `organization.BuildTopology` and
`realization.Layout.Describe`; it implements neither tree ancestry nor area
specificity. Declarations preserve the selection source of defaults and exact
overrides, and absent team/helper profiles. Resolution is an explicit,
asynchronous `runFacts` capture of committed source with the selected Runtime
profile, with refusal/progress and source identity kept separate from the draft.

The local bootstrap test uses fake Runtime metadata readers, an override of both
profiles and a team with one explicit and one inherited helper profile. It runs
the product binding path with evidence that refuses at the actual `run.start`,
before materialization or Agent work. It compares every Manager's Execute,
Verify, helper profile identities, helper roles and write flags and `maxHelpers`
against that recorded binding, and checks provider model decoding. The declared
test checks ownership exclusions and exact provenance. HTTP tests check explicit
resolution, sharing an active capture, progress, refusal and unavailable Runtime.

Validation commands and results are recorded in this package's PR. No hosted CI,
Docker build, container, provider, actual Apply workflow or model authoring
session was run. Failed development observations: a DirectoryBoundary conversion
did not compile; the inert test workspace initially lacked its cleanup probe;
the evidence-stop test initially expected an error return where the Run records
the evidence failure in its outcome. These test/setup faults were corrected.

Independent complete-diff review found that a failed File check could hide
declared ownership as absent. The projection now reports the File-check failure
and explicitly states that validated areas are unavailable, with a regression
case. The owner separately authorized exactly two bootstrap imports of the pure
organization and realization packages; the architecture gate and its existing
boundary expectation are updated together. No adapter import boundary changes.
The review also found that the check-level provenance pointed only at the ID
field; it now names the whole declaration document and its content identity.

Local checks: focused bootstrap organization and confirmation tests, cockpit
HTTP tests, architecture imports and the existing bootstrap boundary test,
working-tree structural/accounting preparation, changed-page documentation links
and `git diff --check` passed. The official generated view was written using
preview digest `sha256:90f7fb842e4b8a9ce1ff74d24e758eb1413f405d73c804ad18fcb4abcceabb6c`;
its rendered content digest is
`sha256:a25033e3de77f4502b3dd84eb1d25e84ef46f134f0bd9131f7de9b7c9ca803ee`.

Browser acceptance, real recorded-run inspection and screenshots belong to the
later packages; this read model alone establishes none of them. The canonical
Definition records the commissioned chart as intent, including the later work.
