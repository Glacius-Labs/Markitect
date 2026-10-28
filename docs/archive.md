# Archive index

Main keeps what the concept 2.0 build, its development and V1's qualification evidence need. Everything else stays at a named archive point and is listed here: designs of earlier product lines, closed experiments, dated validation records, handoffs, strategy and research. Archived files keep their original bytes and wording; DEC-069 applies to the documentation on Main, not to the archive ([DEC-071](concepts/register.md#dec-071-repository-structure-and-archive)).

Links to archived files use the full commit SHA, never a branch or tag name, because names can move or vanish, as the tags did. Each group below links its original folder at the archive point; replace the folder with the file path from the group's description to reach a single file.

## Archive points

| Archive point | Commit | What it holds |
|---|---|---|
| Branch `archive/main-pre-r3c-20261027` | [`e6d9b94a64c2083ee1487414a728a520a87def03`](https://github.com/Glacius-Labs/Markitect/tree/e6d9b94a64c2083ee1487414a728a520a87def03) (tree `0fbc1d063bdada18dfe826e9641a47a3732122cf`) | Main on 27 October 2026, immediately before the documentation steps of the cleanup (R3c). Contains every page removed or merged by R3c, and is in Main's ancestry |
| Branch `archive/main-pre-r2-20261024` | [`2389e252a85e3896da91f5f4649a3394b9222b2c`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c) (tree `fec8e0c2fc16413e2e206cfb220059791338dfc1`) | Main on 24 October 2026, immediately before the repository restructuring (R2). Contains every file removed or moved by R2, and is in Main's ancestry |
| Branch `archive/v1-pre-cutover-20261023` | [`447ac3b152de79b52de80513e14dc94349a060bd`](https://github.com/Glacius-Labs/Markitect/tree/447ac3b152de79b52de80513e14dc94349a060bd) (tree `6bd59e02d929dc94fe82f799cefc0da18ff7984c`) | The source snapshot of 23 October 2026 before the Extract, Cockpit, authoring and Setup retirement; see [its bundle](#bundle-of-the-pre-cutover-archive-point) |
| Main before T1 | [`f23c2e12d9833efa61e5e65c90018016423abe74`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74) | Main on 28 October 2026, immediately before step T1 of the concept 2.0 build. Contains V1's language, Run host, adapters and guides that T1 removed ([below](#removed-by-t1)), and is in Main's ancestry; no separate archive ref or bundle |
| Pre-reset commits | Six, below | History before the reset of 23 October 2026, which Main's history does not contain |

The refs under `archive/` are protected against deletion and force-push by the ruleset "Protect archive refs". The three Main-derived points also exist as Git bundles outside the repository, so a second history reset cannot remove them. Restore an archive point into a new empty destination outside existing checkouts; never reset or overwrite an active implementation checkout to restore history. An archive point holds committed source only, not local credentials, operator state, provider histories or uncommitted planning documents.

### Bundle of the R3c archive point

- File: `C:\Users\Consiliari\Glacius Labs\Markitect-archives\main-pre-r3c-20261027\markitect-e6d9b94a.bundle`
- SHA-256: `12facd617ae41029f0a7b073705acfb742ac16ffac1bce7e30dcb8af0e3be70b`. The bundle holds the full history (615 commits) under the ref `archive/main-pre-r3c-20261027`.
- A restore check (clone, `git fsck --full`, exact commit and tree, empty status) sits beside it in `restore-check`.

Restore from the named reference or, if it is unavailable, from the bundle after verifying its hash:

```text
git clone --branch archive/main-pre-r3c-20261027 https://github.com/Glacius-Labs/Markitect.git NEW_EMPTY_DESTINATION
git clone --branch archive/main-pre-r3c-20261027 PATH_TO_BUNDLE NEW_EMPTY_DESTINATION
```

Require commit `e6d9b94a64c2083ee1487414a728a520a87def03`, tree `0fbc1d063bdada18dfe826e9641a47a3732122cf` and an empty status.

### Bundle of the R2 archive point

- File: `C:\Users\Consiliari\Glacius Labs\Markitect-archives\main-pre-r2-20261024\markitect-2389e252.bundle`
- SHA-256: `8db5fa6899fd0c8941901f4985a405e2bf3c056ea52ff0168e4caf76d30905e9`. The bundle holds the full history (222 commits) under the ref `archive/main-pre-r2-20261024`.
- A restore check (clone, `git fsck --full`, exact commit and tree, empty status) sits beside it in `restore-check`.

Restore from the named reference or, if it is unavailable, from the bundle after verifying its hash:

```text
git clone --branch archive/main-pre-r2-20261024 https://github.com/Glacius-Labs/Markitect.git NEW_EMPTY_DESTINATION
git clone --branch archive/main-pre-r2-20261024 PATH_TO_BUNDLE NEW_EMPTY_DESTINATION
```

Require commit `2389e252a85e3896da91f5f4649a3394b9222b2c`, tree `fec8e0c2fc16413e2e206cfb220059791338dfc1` and an empty status.

### Bundle of the pre-cutover archive point

The pre-cutover point was created and verified before any S1 source removal under the [source implementation approval](concepts/sources/v1-source-implementation-approval-20261023.md). It preserves the integrated source before the Extract, Cockpit/UI, authoring and Setup retirement; it is not a release or runtime-readiness claim. The remote branch was pushed without force and read back at the exact commit. Ordinary implementation cleanup must not delete the ref or the bundle. It is a source snapshot, not a dependency on dangling historical objects: the snapshot commit has no parents, so it establishes no ancestry to the pre-reset commits.

- File: `C:\Users\Consiliari\Glacius Labs\Markitect-archives\v1-pre-cutover-20261023\markitect-447ac3b.bundle`
- SHA-256: `6336E2E48E46F9FA35EB278C66C3BCBA9EB058664951676B56A8336CB40124BB`. The bundle records a complete history and the `refs/heads/main` reference, but no default HEAD, so a clone selects `main` explicitly.
- A restore check (checkout, `git fsck --full`, exact commit and tree, empty status) succeeded and sits beside it in `restore-check`; it is not an implementation worktree. Bundle verification succeeded independently of that checkout. A copy of the restore guide sits beside the bundle as `RESTORE.md`.

Restore from the named reference or, if it has moved or is unavailable, from the bundle after verifying its hash:

```powershell
git clone --branch archive/v1-pre-cutover-20261023 https://github.com/Glacius-Labs/Markitect.git NEW_EMPTY_DESTINATION
$bundlePath = 'C:\Users\Consiliari\Glacius Labs\Markitect-archives\v1-pre-cutover-20261023\markitect-447ac3b.bundle'
Get-FileHash -Algorithm SHA256 -LiteralPath $bundlePath
git clone --branch main $bundlePath 'C:\path\to\new-restored-markitect'
git -C 'C:\path\to\new-restored-markitect' fsck --full
git -C 'C:\path\to\new-restored-markitect' rev-parse HEAD
git -C 'C:\path\to\new-restored-markitect' rev-parse 'HEAD^{tree}'
git -C 'C:\path\to\new-restored-markitect' status --short
```

Require commit `447ac3b152de79b52de80513e14dc94349a060bd`, tree `6bd59e02d929dc94fe82f799cefc0da18ff7984c` and an empty status.

### Pre-reset commits

The history reset of 23 October 2026 removed these commits from every branch. Each is pinned by an archive ref, which keeps it reachable on GitHub. The retired `v0.14.1` distribution and the preceding source generation keep their own contracts; current consumers do not reinterpret those formats or revive their runtime.

| Ref | Commit | Date | What it holds | Cited by |
|---|---|---|---|---|
| `archive/pre-reset-7dbd599c` | [`7dbd599c81540c8203a1b7f83afbc335174f4f1f`](https://github.com/Glacius-Labs/Markitect/tree/7dbd599c81540c8203a1b7f83afbc335174f4f1f) | 7 Oct 2026 | Source of release `v0.14.1`, the last Project/Domain release | [Production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md), [documentation maintenance](development/documentation.md) |
| `archive/pre-reset-3ebe3c6e` | [`3ebe3c6e6d283b998020350334e24a7303b089b7`](https://github.com/Glacius-Labs/Markitect/tree/3ebe3c6e6d283b998020350334e24a7303b089b7) | 16 Oct 2026 | The evening rules and the consistency review of the working-day reflection | [Reflections](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/reflections/README.md), [working-day discussion](concepts/sources/working-day-discussion-20261016.md) |
| `archive/pre-reset-bc4b49e7` | [`bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`](https://github.com/Glacius-Labs/Markitect/tree/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310) | 16 Oct 2026 | Main with the work-item backlog (`docs/work-items/backlog.yaml`) | [Concept register](concepts/register.md) |
| `archive/pre-reset-5ff32069` | [`5ff3206917fa3b7482d9daedb903d57a919f56b8`](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8) | 17 Oct 2026 | The removed product generation before the Apply rebuild | [Architecture](architecture.md), [Vision](vision.md) |
| `archive/pre-reset-1f4a1ede` | [`1f4a1ede214d05fd2fee615b3905207df7702426`](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426) | 17 Oct 2026 | The complete pre-retirement tree, the basis of the [retirement record](validation/distribution-retirement-20261017.md) | [Measurement](guides/measurement.md), [packaging guide](../packaging/README.md), [examples](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/examples/README.md) |
| `archive/pre-reset-0c28097c` | [`0c28097c030bc5d031b501002bc85634cd66a4c8`](https://github.com/Glacius-Labs/Markitect/tree/0c28097c030bc5d031b501002bc85634cd66a4c8) | 20 Oct 2026 | Main when the Apply workflow terminology was recorded | [Apply workflow conversation](concepts/sources/terminology-and-file-accounting-20261020.md) |

The 18 retired release versions `v0.1.0` to `v0.14.1` no longer exist as Git tags. The mapping from each former tag to its commit is [retired-releases-20261017.json](validation/retired-releases-20261017.json); only the commit of `v0.14.1` is pinned.

## Archived groups

The register entries that cite a group are listed for orientation; the register links the individual files.

| Group | Original location at the archive point | What it holds and why it still matters | Register |
|---|---|---|---|
| Pre-V1 product design | [`docs/design/*.md`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design) (32 files) and [`docs/design/project-world/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/project-world) (28 files, with the shop example) | Designs of the Project/Domain and projection line, adoption and Init, the skills, Cockpit, the verb table and the earlier model-first workflow. Cockpit, Setup and authoring are outside V1 | DEC-004, DEC-010, DEC-065, DEC-066, VIS-001 |
| Apply and canonical-model design | [`docs/design/canonical-model/*.md`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model) (35 files) | The Apply designs behind DEC-024 to DEC-067 (concept, plan, runtime, workflow, playground, library flow), the commands design, model consolidation, the standard library and its working-vocabulary pages. Concept 1.0 supersedes the Apply designs | CPT-004, DEC-002, DEC-024 to DEC-026, DEC-031, DEC-033 to DEC-037, DEC-039, DEC-040, DEC-042, DEC-044, DEC-045, DEC-053, DEC-056 to DEC-058, DEC-061 to DEC-064, DEC-067, ENH-001 to ENH-003, ENH-006, IDEA-001, IDEA-017, VIS-001 |
| Language and model design history | [`docs/design/canonical-model/minimal-language/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language) (88 files without the specification and the Core dependency), [`library-example/verify-model.py` and `verification.json`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/library-example), [`docs/design/product-model-refinement/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/product-model-refinement), [`task-management/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/task-management) and [`task-management-model/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/task-management-model) | The evidence behind the language v1 baseline: probes, identities, the exchange example with its verification reports, the Command/command collision example, earlier model drafts and the Tasks counter-probe. Nothing runs the Python probes any more | none |
| Handoffs and contracts | [`docs/development/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/development): the Claude and Codex Apply and Cockpit handoffs of 20 and 21 October, `night-apply-stability-20261021.md`, `shared-contracts.md`, `session-handoff.md` | Pre-V1 assignments and the v0.13 contract table. They grant no current work | DEC-037, DEC-064, ENH-002, IDEA-018, VIS-001 |
| Validation records | [`docs/validation/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation) (77 files including 8 screenshots; the qualification record, the A1 proof and the release evidence stay on Main) | Dated evidence about retired source: the runtime probes of 17 October, Apply, Extract and recursive-Apply runs of 19 to 23 October, Cockpit records, the Project/Domain line, comparisons and pilots | DEC-008, DEC-037, DEC-056 to DEC-061, DEC-064, DEC-066, ENH-006 |
| History | [`docs/history/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/history) (35 files; the release evidence stays on Main) | Earlier plans, handoffs, surveys, assessments, the WinGet and source-release distribution notes and the Apply design before DEC-056 | DEC-003, DEC-017, DEC-030, DEC-035, DEC-037 to DEC-039, DEC-041, DEC-043, DEC-045 to DEC-050, DEC-052 to DEC-055, ENH-005 |
| Strategy and marketing | [`docs/strategy/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/strategy) (33 files), including the [marketing working plan](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/strategy/marketing-plan.md) | Supplied strategy bundles, `explainer.md`, `short-posts.md` and the marketing working plan of 17 October. Restore it from here when marketing work resumes | none |
| Research | [`docs/research/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/research) (6 files) | Supplied research orientation and protocol-v1 capability records | DEC-018, IDEA-016 |
| Work items | [`docs/work-items/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/work-items) | A pointer to the archived surveys | none |
| Apply trials | [`experiments/apply-trials/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/apply-trials) (273 files) | CCF, NIGHT, D-series and plan comparison cases with answer keys, built on the pre-V1 Pass engine; the series is closed | none |
| Recursive Apply demo | [`experiments/recursive-apply-demo/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/recursive-apply-demo) (23 files) | The DEC-056 two-Apply Library demonstration and Extract runs | none |
| Model assessment | [`experiments/model-assessment/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/model-assessment) (8 files) | The Session C method trial of the Pass-process assessment | none |
| Test projects | [`experiments/test-projects/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/test-projects) (183 files) | Library r2, Readinglog, Roombook and the ordinary-documentation comparison with its packet tool. A dispatch that needs them restores them into `testdata/projects/` from here | none |
| Runtime baseline | [`experiments/runtime-baseline/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/runtime-baseline) (`README.md`, `baseline.py`, `codex_login.py`) | The single-agent provider baseline on the pre-V1 runtime (Claude Haiku, Codex Luna). The two files that assess a candidate with the oracle moved to `tools/library-oracle/container/` | none |
| Experiments map | [`experiments/README.md`](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/experiments/README.md) | The index of the former folder | none |

### Accepted later directions

These designs are accepted as later directions but have no V1 consumer. They stay findable here instead of on Main.

| Direction | Where it lives at the archive point |
|---|---|
| Standard library | [`docs/design/canonical-model/standard-library*.md`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model) and its working-vocabulary pages in the same folder |
| Commands (Extract, Verify, Optimize) | [`docs/design/canonical-model/commands.md`](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md) |
| Init, adoption and skills | [`docs/design/`](https://github.com/Glacius-Labs/Markitect/tree/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design): `init-adoption-preparation.md`, `selective-adoption-handoff.md`, `brownfield-representation-adoption.md`, `project-capability-skills-20261009.md`, `modeling-skill-guidance.md` |

## Removed and merged by R3c

R3c consolidated the remaining documentation after the owner decided to develop Markitect as a regular software project ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project); [cleanup inventory](development/cleanup-inventory-20261027.md)). The pages below left Main at the R3c archive point. Both tables list the register entries that cite a page, for orientation.

### Archived pages

| Page | Location at the archive point | What it holds and why it still matters | Register |
|---|---|---|---|
| Product reflections | [`docs/concepts/reflections/`](https://github.com/Glacius-Labs/Markitect/tree/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/reflections) (README and four reflections of 16 October) | Dated discussions of trust through organized agent work, a working day with delegated realization, its demonstration and experience without implicit rules, with owner wording and assistant interpretation kept apart. They discuss the pre-V1 Apply design that concept 1.0 superseded; the owner sources they quote stay on Main | VIS-001, ENH-006, IDEA-001, IDEA-017 |
| Repository structure (R1) | [`docs/development/repository-structure.md`](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/repository-structure.md) | The R1 inventory of every file, the accepted target layout, the archive mechanism and the R2 work list, which R2 executed | DEC-071 |
| V1 bounded assignments | [`docs/development/v1-workstream-assignments.md`](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/v1-workstream-assignments.md) | Writer scopes of delivery-plan steps 2 and 3: the Run contract (D10–D13, D15, D17) and the provider scope (D18). Both steps are done | none |
| Concept conformance audit | [`docs/development/concept-conformance-audit-20261024.md`](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/concept-conformance-audit-20261024.md) | The supplied static audit of Main `6c7d758c` against the V1 concept; origin of A1–A6 and R1–R3, closed by concept 1.0 | none |
| V1 counter-review | [`docs/development/v1-architecture-audit-20261023.md`](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/v1-architecture-audit-20261023.md) | The source-bound counter-review of the v0.17 architecture of 23 October, superseded by concept 1.0 | DEC-067 |
| Production assessment | [`docs/validation/production-assessment.md`](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md) | Source, platform and publisher evidence of the retired releases up to `v0.14.1`, with its limits. The releases no longer exist, so its role as owner of release asset claims has nothing left to own; the [retirement record](validation/distribution-retirement-20261017.md) and the [retired release mapping](validation/retired-releases-20261017.json) stay on Main | none |

### Merged pages

These pages were folded into a current page. Their pre-merge text stays at the archive point, except for the two operations guides: R3c step 12 merged them after the archive point, together with slice L3, so their pre-merge text is linked at Main `1eec3f0e`, the last commit that has them.

| Former page | Now in | Pre-merge text | Register |
|---|---|---|---|
| `docs/development/v1-source-archive-20261023.md` | [Bundle of the pre-cutover archive point](#bundle-of-the-pre-cutover-archive-point) | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/v1-source-archive-20261023.md) | none |
| `docs/design/v1-execution-architecture.md` | [Architecture](architecture.md): its current contracts (package direction, ports, Run lifecycle, candidates and state, checks and review, harness, provider and authentication). Its scope statement for the source cut, the proposed package locations, the stream ownership of the Sessions port, the Phase C rationale and the acceptance plan stay only at the archive point | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/design/v1-execution-architecture.md) | DEC-067, DEC-068 |
| `docs/development/modules.md` | [Architecture](architecture.md): dependency direction, the [import gate](architecture.md#import-gate) and [how to add a package](architecture.md#adding-changing-or-removing-a-package); the package invariants in Foundations and Project, Adapters and State and evidence storage, which T1 removed ([architecture before T1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/architecture.md#foundations-and-project)) | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/modules.md) | none |
| `docs/guides/source-snapshots.md` | [Architecture before T1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/architecture.md#foundations-and-project): snapshot values, capture identity and acquisition limits, removed by T1; the release input in [Distribution](architecture.md#distribution) | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/guides/source-snapshots.md) | none |
| `docs/guides/project-operations.md` | [Project workflow before T1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md#model-commands): the command reference for `check`, `context`, `docs` and `apply`, removed by T1 | [On Main before the merge](https://github.com/Glacius-Labs/Markitect/blob/1eec3f0e043da2a49ddd94222e236c714ba39e8b/docs/guides/project-operations.md) | DEC-044 |
| `docs/guides/operations.md` | [Project workflow before T1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md#operations): authoring, rendering and troubleshooting, removed by T1; [release operations](../packaging/README.md#release-operations) in the distribution guide. Its stale lines about module gates and registry facts of the self-model were dropped | [On Main before the merge](https://github.com/Glacius-Labs/Markitect/blob/1eec3f0e043da2a49ddd94222e236c714ba39e8b/docs/guides/operations.md) | none |
| `internal/adapters/gitrepo/WORKSPACES.md` | [Git repository adapter](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/internal/adapters/gitrepo/README.md#run-git-workspace-service), removed by T1 | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/internal/adapters/gitrepo/WORKSPACES.md) | none |
| `internal/adapters/gitrepo/PUBLISHER.md` | [Git repository adapter](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/internal/adapters/gitrepo/README.md#publication-boundary), removed by T1 | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/internal/adapters/gitrepo/PUBLISHER.md) | none |
| `docs/development/parallel-work.md` | [CONTRIBUTING](../CONTRIBUTING.md#pull-requests-and-parallel-work), without its lines about the Extract stream, project-model coverage, the cutover and the pre-V1 Go boundaries | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/parallel-work.md) | none |
| `docs/development/README.md` | [Documentation map](README.md#development) | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/README.md) | none |
| `docs/concepts/origin-and-rationale.md` | [Vision](vision.md#origin-and-rationale) | [At the archive point](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/origin-and-rationale.md) | CPT-001 and the register index |

## Removed by T1

Step T1 of the [concept 2.0 build](implementation-plan.md#concept-20-build) replaced V1's language and removed V1's Run host, with no compatibility kept ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)). Everything it removed stays at [Main before T1](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74). The [model language guide](guides/model-language.md) describes what replaced the language; T4 builds the new Run engine and T5 the native host.

| Removed | Location before T1 | What it held |
|---|---|---|
| V1 language and its Libraries | [`internal/model`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/internal/model), [`testdata/core`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/testdata/core) (Core 1.2.0), [`libraries/`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/libraries) (`values`, `data`, `api`, `practice`, `architecture`) | Typed YAML Definitions, the v1 compiler, the Core and the standard Libraries; the [language contract](design/language-contract.md) specifies them |
| V1 Run host | [`internal/`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/internal): `organization`, `harness`, `modeldocs`, `run`, `runtimeconfig`, `apply`, `snapshot`, and the old `realization`, `project` and `bootstrap` | Manager topology and context, the harness, the Run executor, Apply and publication, runtime profiles, snapshot values and the generated model view |
| V1 adapters | [`internal/adapters/`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/internal/adapters): `checks`, `claudecode`, `codexappserver`, `documents`, `gitrepo`, `runner`, `runstore`, and the old `cli` | Container checks, the Claude and Codex Session adapters, the document writer, Run Git and publication, the Linux runner launcher, the Run store and the V1 command line |
| Linux runner | [`cmd/markitect-runner`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/cmd/markitect-runner), [`packaging/runner`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/packaging/runner) | The runner entrypoint and image; the [runner design](design/linux-runner.md) records it |
| Configuration examples | [`examples/`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/examples) | The runtime-profile and run-configuration examples of V1's Apply |
| Python analysis tool | [`tools/model-analysis`](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/tools/model-analysis) | Lint signals for v1 models, replaced by `markitect lint` |
| Provider guide | [`docs/guides/provider-adapters.md`](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/provider-adapters.md) | Runtime profiles, native Sessions, account login and the Claude adapter |
| V1 guides | [Architecture](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/architecture.md) and [project workflow](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md) before T1 | V1's Run, Apply, adapter and state architecture; Apply, the runner, configuration, state and evidence |

## Moved, not archived

These files are still on Main under new paths; their content is unchanged apart from path and link updates (the packaging, Library and oracle READMEs and the oracle container scripts also carry the edits their new places need). After R2, slice C1 translated the Core in `testdata/core` into English as `markitect-core` 1.1.0 and moved the Library manifests and lock to it. R3c later archived the production assessment, merged `source-snapshots.md` into the architecture and merged `project-operations.md` and `operations.md` into the project workflow and the distribution guide ([above](#removed-and-merged-by-r3c)).

| Former path | Current path |
|---|---|
| `assets/` | [`docs/assets/`](assets) |
| `docs/project-workflow.md`, `project-operations.md`, `provider-adapters.md`, `operations.md`, `measurement.md`, `source-snapshots.md` | [`docs/guides/`](guides/); T1 later removed `provider-adapters.md` |
| `docs/repository-layout.md` | [`docs/development/repository-layout.md`](development/repository-layout.md) (rewritten) |
| `docs/design/canonical-model/minimal-language/language-contract.md`, `diagnostic-contract.md` | [`docs/design/`](design/) |
| `docs/design/canonical-model/sources/` | [`docs/concepts/sources/`](concepts/sources/) |
| `docs/history/distribution-retirement-20261017.md`, `retired-releases-20261017.json`, `distribution/production-assessment.md` | [`docs/validation/`](validation/) |
| `docs/design/canonical-model/minimal-language/exchange-example/dependencies/core/` | `testdata/core/`, removed by T1 ([before T1](https://github.com/Glacius-Labs/Markitect/tree/f23c2e12d9833efa61e5e65c90018016423abe74/testdata/core)) |
| `docs/design/canonical-model/library-example/` (model M1, vocabulary, business rules) | [`testdata/library/`](../testdata/library/) |
| `docs/design/canonical-model/library-example/evaluation-contract.md`, `experiments/library-playground/` | [`tools/library-oracle/`](../tools/library-oracle/) |
| `experiments/runtime-baseline/assess.py`, `run_oracle.py` | [`tools/library-oracle/container/`](../tools/library-oracle/container/) |
| `integration/README.md`, `run-markitect.go`, `run-markitect_test.go` | [`packaging/README.md`](../packaging/README.md), [`packaging/bootstrap/`](../packaging/bootstrap/) |
