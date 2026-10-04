# Markitect-first v0.13.0 release readiness

**Status:** published and independently verified on 2026-10-04. The immutable [v0.13.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.13.0), release ID `402873994`, binds source `ec312e35c15012c07acb6838c03e9316c5373dde`. This is the first stable release intended to support the Markitect-first engineering workflow and the current adoption/verification primitives. It does not establish reduced human attention, general architecture safety, productivity, complete autonomy or adopter acceptance.

The additive pre-1.0 minor version promotes supported CLI, workflow and alpha-versioned record contracts. The finite Domain kernel remains unchanged from v0.12.0. Earlier tags, published assets and historical package versions remain intact. [Vision](../vision.md) owns the product thesis; [Measurement](../measurement.md) owns future validation. The canonical [Markitect-first Change workflow](https://github.com/Glacius-Labs/Markitect/blob/ec312e35c15012c07acb6838c03e9316c5373dde/internal/authoring/resources/workflow-markitect-first-change.yaml) owns the protocol; [Usage](../usage.md#markitect-first-release-candidate) and [Markitect-first](../markitect-first.md) describe execution.

## Published scope

| Capability | v0.13.0 classification | Exact proof and limits |
|---|---|---|
| Read-only analysis of ordinary PolicyResult failures | Supported explicit CLI behavior | Opt-in Context/Impact retain failure exit status; structural errors, Check, Verify and reconciliation remain strict. Source and packaged policy-analysis fixtures passed on Windows/Linux. |
| Selective `prepare` and `copy-me` | Supported alpha-versioned byte/reference infrastructure | Exact selected local Git blobs, exclusive external storage and immutable handoff; pure evidence/candidate/decision consistency validation. Packaged CLI lifecycle passed on both platforms. No acquisition by Copy Me, reviewer authentication, interpretation quality or automatic adoption claim. |
| Markitect-first Workflow and generated Skills | Supported canonical guidance | Portable authoring includes it; generated Codex/Claude entrypoints derive from canonical owners. A fresh no-history actor completed one bounded implementation-only replay; root intent-first and configured verification checkpoints are recorded separately. This is not continuous enforcement of every agent action. |
| Standalone managed-artifact helper | Supported source-package command | Closed versioned configuration, literal roots, exact tooling ownership and file exclusions with reasons. Builds and root invocation passed from packaged source on both platforms. No fifth binary asset; build the reviewed helper from the exact source package. Accounting does not prove generated freshness or source semantics. |
| .NET/source hardening and native projection inventory | Supported bounded implementation behavior | Existing specialized .NET checks retain their narrow evidence boundary. Exact selective Git acquisition blocks implicit fetching. Check and native reconciliation share renderer stale inventory and recognize disjoint independent nested Projects without importing their graph. |

Standalone GitHub/Azure supplied-capture consumers and MCP remain experimental. They do not establish live provider state or support provider Apply. Broader graph queries, Pattern/Trait/Composition, inheritance, automatic inference/adoption, source-language semantics and background operation remain out of scope. No new Core/Domain/SPI semantic primitive was required.

## Commit-bound acceptance matrix

The initial passing baseline CI run `37170101983` predates this work and is not final-candidate proof. The following reviewed candidates, identical integrated trees and exact tagged source provide the release evidence.

| Gate | Exact evidence | Result |
|---|---|---|
| Protocol/coverage independent review and PR gates | [PR 74](https://github.com/Glacius-Labs/Markitect/pull/74), reviewed `688b0950a8b31e2a4f6c6980927a7903876875f9`; [CI 37179389649](https://github.com/Glacius-Labs/Markitect/actions/runs/37179389649); integrated `e192c8c79a25c05db725b82535316583c2a0621b` has the same tree `c6d78a60a0f89651468bf50f348855cd324952ba` | Independent review found no remaining blocker; Windows/Linux passed. Earlier discovered defects and failed attempts remain in the dogfood record. |
| Final schema-owner correction and independent review | [PR 75](https://github.com/Glacius-Labs/Markitect/pull/75), reviewed `f1c1428867ef09902590599588a64af537f9528d`; [CI 37180770786](https://github.com/Glacius-Labs/Markitect/actions/runs/37180770786) | No remaining blocker; Windows/Linux passed. Final integrated source has the same tree `52a37a5e3f7484a2c3e5de93360b3348f4b7ae7d`. |
| Final main source gates | [Main CI 37181183052](https://github.com/Glacius-Labs/Markitect/actions/runs/37181183052), source `ec312e35c15012c07acb6838c03e9316c5373dde` | Go tests, vet, build, module verification, schemas, executable examples, root check/format/context/coverage/configured fixed Verify and packaged commands passed on Windows/Linux. |
| Actual main source artifact | Download from main CI above; nested source ZIP SHA-256 `721e641d2c2ec43bc01c4f74cb735c558c8c9857cc7b623e64500ada26e35142` | Equal to release bundle's nested source archive; bootstrap, helper, root authoring Project and canonical Workflow members inspected. |
| Tagged Release workflow | [Run 37181716891](https://github.com/Glacius-Labs/Markitect/actions/runs/37181716891), attempt 1, exact source above | All quality, tag/version/main-ancestry, Windows/Linux bundle install/bootstrap smoke and asset assembly jobs passed. |
| Publication preflight | Read-only publisher plan for exact tag, run and four payload digests; no prior release/draft; local final bundle equals run-scoped bundle | Reviewed before explicit publication under the owner's conditional authorization. |
| Immutable publication | Release `402873994`, source above | Publisher returned `published-verified`; public API reports immutable, non-draft, published 2026-10-04T06:14:38Z. Annotated tag is unsigned; release attestation is verified separately. |
| Independent public verification | Separate download of all four public assets; `gh release verify v0.13.0` and each `gh release verify-asset` | All succeeded; each digest matches the table below. Public Windows binary reports 0.13.0, checks local minimal fixture and produces portable authoring preview. Local fixture check is provisional; fixed packaged source and platform evidence are separate. |
| Public distribution metadata | `markitect-release distribution --tag v0.13.0 --repo . --write` | Public assets, provenance and attestations verified before generated README install/example sections and three canonical WinGet manifests were written. Initial read-only run correctly rejected stale v0.12.0 README metadata. Native `winget validate` succeeded; catalog availability is not established. |
| Published bundle upgrade smoke | Separate isolated public-bundle v0.12.0 → v0.13.0 replay | Passed: previews preserved snapshot/worktree, complete upgrade hashes matched, pinned bootstrap tests/version and fixed check passed at isolated fixture `b7d9691281e1589ad51e779496ab058a370d7f26`. [Production assessment](../production-assessment.md#v0130-published-release) records the limits and retained failed setup. Tagged CI supplies separate Windows/Linux proof. |
| Restricted preparation/candidate exercise | Owner-approved private capture, retained externally; 36 files and four byte/reference-valid candidates | Exact scope identities and source cleanliness verified. Published Windows binary revalidated the unchanged capture/queue. No private identifiers or source bytes are published here. |
| Adoption disposition | Real owner decisions, adopter diff and task/rubric | Pending; no candidate accepted or adopted. Not a generic release blocker. |

## Public payloads

| Asset | Independently verified SHA-256 |
|---|---|
| `markitect-v0.13.0-bundle.zip` | `a14e7ce2619f9cd4bc633e23beaaf49ba83d9681bd9957075803abb04644f953` |
| `markitect-v0.13.0-linux-amd64` | `7163bea66f539c3bf558770cdbd0745beea6b5b26a50e5f3fc4f1021a22abf2c` |
| `markitect-v0.13.0-provenance.yaml` | `1f3e4ce5bfdc28a2ba0ded6a706a02d157da2fe29e4ff6de4e703693aea90bfd` |
| `markitect-v0.13.0-windows-amd64.exe` | `ac40c666ff549e2f6a7af0923a716278ecf4e019589fdd0de0982259bde17054` |

The bundle remains the five-file project pin: release manifest, lock, exact source archive and two bootstrap files. The source package carries the helper; it is not an additional native release asset. Current alpha API/format labels remain explicit; publication is not a promise of indefinite pre-1.0 compatibility.

## Dogfood and remaining owner gate

[Dogfood](markitect-first-dogfood.md) records implementation-only and desired-intent-first checkpoints, the actual unmanaged-file negative control, deterministic context and native empty-plan convergence. It preserves the missing config/test bytes in Context, conservative ten-resource implementation impact, expensive fixed verification and setup friction. Tooling ownership does not silently become a semantic dependency. Prior MyMeetings and AGENTS.md comparison negative findings remain unchanged.

The private owner packet and next published-pin experiment plan are external. Four candidate dispositions, a genuine recurring coordination burden, ordinary backlog task, expected behavior/rubric and any additional approved evidence are still required. No invented task or automatic adoption substitutes for owner authority. The next comparison must include setup, review and recurring upkeep against simpler owner guidance plus existing architecture tests; no attention-saving conclusion follows from release verification.

## Completion answers

| # | Question | Result |
|---|---|---|
| 1 | Canonical protocol | `core/Workflow/markitect-first-change`, embedded in portable authoring; prose elsewhere points to it |
| 2 | Fresh discovery | Root AGENTS pointer → generated provider Skill → canonical Workflow; fresh no-history replay succeeded after the recorded cache failure |
| 3 | Task start | Fix full BASE; `check`, selected `model/find/explain/context`; inspect configured checks, pins, adapters and uncovered inputs |
| 4 | Change classification | Implementation-only follows accepted intent without model churn; intent/migration changes the desired owner first; ambiguity escalates |
| 5 | Before code | Changed modeled intent requires canonical desired checkpoint, strict check (or explicit failed-policy migration analysis) and reviewed impact before corresponding implementation |
| 6 | Coverage guarantee | Enumerated files in literal managed roots have explicit canonical/input/generated/tooling/excluded status or findings; no source meaning, freshness, secret detection or acceptance guarantee |
| 7 | Configuration | Closed `artifact-coverage/v1alpha1`; nonoverlapping literal roots, exact tooling owner paths, exact exclusions with reasons; no subtree/glob exclusion |
| 8 | Negative control | Actual unaccounted managed file exited 1; after removing only that created file the report passed |
| 9 | Existing semantics | Areas/imports, Workflow/Skill/Rule, exact file inputs, renderer ownership and configured checks were sufficient |
| 10 | Shared primitive | None; no Core/Domain/SPI language expansion |
| 11 | Dogfood changes | Real unknown-flag test expansion; separately committed context proof-limit input before its technical regression; root native empty-plan convergence |
| 12 | Fixed friction | Cache placement/permissions, nested Project output inventory, authoring manifest placement, helper-on-PATH guidance, deterministic overlap diagnostics and schema-generator ownership |
| 13 | Supported scope | Explicit failed-policy read-only analysis, selective prepare/Copy Me byte-reference contracts, bounded .NET/source hardening, portable protocol/Skill and source-package coverage helper |
| 14 | Experimental/deferred | Offline GitHub/Azure consumers and MCP stay experimental; broad Core queries/composition, automatic inference/adoption/provider Apply/background operator remain excluded |
| 15 | Version | v0.13.0, additive pre-1.0 supported CLI/workflow/contracts; finite Domain kernel unchanged from v0.12.0 |
| 16 | Release gates | Passed: final source ec312e35, main CI 37181183052 and tagged Release 37181716891 attempt 1; full Windows/Linux source, packaged and installation/bootstrap gates |
| 17 | Publication | Published immutable v0.13.0 / release 402873994; separate public download, release attestation and all four asset attestations/digests verified |
| 18 | Public contract | Exact tag/source/provenance and four assets; helper source carried in the pinned source archive, no fifth binary asset |
| 19 | Restricted exercise | Approved 36-file capture matched digests and retained source cleanliness; four structurally validated candidates; no owner decision or adoption |
| 20 | Owner decisions | Disposition four candidates, identify recurring coordination burden, select a real backlog task/rubric and approve any new implementation evidence scope |
| 21 | Adopter readiness | Published binary revalidated the capture/queue; exact future pin and matched experiment plan prepared externally; adoption/task execution remains owner-dependent |
| 22 | Smallest remaining blocker | Actual owner disposition plus a controlled real-task/upkeep comparison; technical feasibility has not established saved human attention |