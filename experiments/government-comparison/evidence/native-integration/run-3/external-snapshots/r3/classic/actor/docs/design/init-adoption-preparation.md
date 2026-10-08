# Existing-project preparation: earlier Init design proposal

**Status:** Historical first-proof proposal. The current source now has a separate `prepare` command and adoption handoff; the canonical contract is [Selective adoption handoff](selective-adoption-handoff.md), with CLI invocation documented in [Usage](../usage.md). The implementation is under source verification; this note makes no CI or release claim. Published v0.12.0 is unchanged, and the existing new-Project `init` behavior remains separate.

## Goal and boundary

An existing-project preparation flow should accept a human-reviewed, explicit evidence scope and prepare a separate discovery workspace. Keep three activities distinct:

1. **Scope review:** the owner supplies repository roots, full immutable commit IDs, and exact repository-relative file paths. The tool validates these inputs and records the owner's supplied scope and exclusions.
2. **Capture:** acquire only the explicitly selected files at those commits and prepare a handoff in an isolated workspace.
3. **Candidate interpretation:** Copy Me or a person examines the captured evidence and records observations, counterexamples, uncertainty, and candidate guidance. Interpretation does not change the adopting Project.

Do not make repository crawling or automatic root inventory a prerequisite for the first proof. Detected-root inventory can be considered later as an optional metadata-only aid; discovered paths must never become selected automatically. Copy Me consumes the human-selected scope and does not choose or broaden it.

The current init remains a minimal new-Project operation. It still requires absent markitect.yaml and target Area paths and creates only those two planned outputs. Preparation does not adopt, initialize, migrate, or overwrite an existing Project or Area, and it does not change checks, policy, resources, adapters, package pins, provider files, repository instructions, or implementation code.

This follows the existing opaque-artifact rule: selected repository files are exact byte inputs, but Markitect Core does not infer their domain meaning. A candidate remains outside canonical Project state until the Project owner separately makes and reviews an ordinary Project change. No requirement to add context to or activate a non-Markitect adopter is implied.

## Earlier first-proof proposal

Treat preparation as a separate operation in the Init application area, with explicit review and capture stages. Whether the CLI presents a new command or mode is a coordinator decision; neither should change the existing command's defaults or refusal behavior.

### 1. Human scope review

Require the caller to supply each repository root, its full Git commit ID, and the exact paths the owner has selected. Keep multiple repositories and their selections separate. Record exclusions and scope rationale as supplied by the owner. Identity and review fields are traceability claims, not authentication.

Validate repository identity, full commit resolution, exact path syntax, supported file type/encoding, and repository-relative containment before capture. Do not expand globs, infer paths from conventions, follow symlinks, consult provider histories, or read files outside the supplied selection. An input error should fail before selected blob contents are opened. If the owner cannot attest to the scope, stop before capture.

For the first proof, omit automated detection of roots. A later inventory may report candidate repository metadata for the owner to review, but it must remain distinct from selected and excluded scope and must not inspect file contents or confer permission to read them.

### 2. Fixed, selective capture

Resolve each supplied full commit to its canonical commit object and retrieve only the exact selected blobs. Bind each file's repository identity, commit, path, mode, and byte digest to the result. Any changed root identity, commit, path set, or selected bytes invalidates capture and requires renewed scope review.

Reuse the existing opaque snapshot.Snapshot value and deterministic digest/comparison boundary where applicable. Keep the repository commit identity separate from the content digest: the current Snapshot.Digest() intentionally excludes ID, so a handoff must bind both. Do not change the digest algorithm or invent a new general snapshot format.

The current internal/source.Load(root, revision) resolves a repository-wide snapshot. It cannot prove that unselected file bytes were not read. The first proof therefore needs a narrow exact-path acquisition seam, preferably implemented at the existing source boundary with Git blob reads for supplied paths, or an explicitly scoped standalone helper if the coordinator finds a source change unwarranted. Do not call whole-tree Load and then filter the result while claiming unselected bytes were never accessed. Selective acquisition must reject missing paths, trees, symlinks/gitlinks, unsafe aliases, unsupported modes, and non-UTF-8 evidence before emitting a successful handoff.

Do not turn captured paths into resource graph edges. ContextRun can be reused only under its existing explicit-source contract and after Copy Me and the coordinator approve how the exact selection/capture identity flows into it. Per-repository source identities must not be merged.

### 3. Isolated workspace and handoff

Write preparation outputs only to a separately designated workspace outside the adopting repository, configured Areas, package contents, generated outputs, and consumer authority files. Require an explicit destination or a coordinator-approved default. Reject a destination that aliases a source root. Create the destination exclusively, and revalidate source identities, selection, destination identity, and absence of conflicting outputs immediately before writes.

The workspace holds the reviewed selection, capture identities, and Copy Me handoff plus reports required to inspect the capture. Persist raw evidence or a rendered ContextRun report only when the approved handoff requires it and the owner has reviewed the storage and retention consequence. Do not persist credentials, provider histories, secrets, or unapproved personal information. On partial failure, report every created path and leave it for deliberate recovery; never delete source or workspace files automatically.

Copy Me owns evidence interpretation and candidate/decision records. Init preparation owns the explicit scope intake, exact selected-source capture, and isolated handoff. The coordinator defines the shared record and write/merge boundary. Neither workstream may silently broaden or concurrently overwrite shared handoff state.

## Ownership proposed before implementation

This proposal owns only docs/design/init-adoption-preparation.md.

If the coordinator approves behavior and shared contracts, the Init implementation slice should be limited to new, directly relevant files such as:

- internal/app/init_preparation*.go for preparation orchestration and destination safety;
- a focused source acquisition helper only if the coordinator approves that shared seam;
- cmd/markitect/init_preparation.go for the separately approved CLI surface;
- focused new tests beside those packages.

The coordinator retains ownership of existing Init behavior, CLI dispatch and shared options, source snapshot identity and digest semantics, common selection or handoff records, schemas, root dependencies, and integration documentation. Exact implementation filenames and the CLI boundary remain subject to ownership review. No adopter checkout or repository file is in scope.

## Safety evidence and focused tests

After approval, demonstrate both supported paths and refusal cases:

| Scenario | Required evidence |
|---|---|
| New-Project Init preview and write | Existing valid behavior and planned bytes remain unchanged; exclusive creation refuses an existing Project or Area. |
| Scope intake | Scope is human supplied and exact; invalid or broadened paths fail before capture. Intake causes no repository or workspace writes. |
| Selective capture | Every opened blob was explicitly selected and belongs to its pinned full commit; per-repository identity and exact content digest are recorded. |
| Unselected or excluded path | A spy/fake source or filesystem evidence proves its content was not opened; it is absent from capture and handoff and is not described as analyzed. |
| Selection or source drift | Changed commit, path set, root identity, selected bytes, or destination causes refusal before workspace writes and requires renewed review. |
| Unsafe repository shape | Symlinks, gitlinks, unsafe paths, case aliases, unsupported file types/encodings, and missing blobs fail closed with actionable diagnostics. |
| Workspace creation | Outputs are confined to the approved external destination and created exclusively; existing Project, Area, package, and authority files remain byte-identical. |
| Partial failure | The result lists exactly what was created; no automatic rollback or cleanup occurs, and rerun does not overwrite those paths. |
| Candidate handoff | Candidate output stays separate from Project state; there is no automatic adoption, policy inference, or human-acceptance claim. |

Tests should assert absence of mutation and forbidden content acquisition directly, not infer those properties from a successful report. The current Init suite remains the regression boundary; do not change its expected contract as part of preparation without a separate coordinator decision.

## Shared contract requests

Before implementation, the coordinator should approve one handoff boundary with Copy Me that defines:

- per-repository root identity and immutable full-commit identity;
- exact human-selected paths and explicit exclusions, including their reasons;
- source identity, mode, and byte digest for each captured file;
- selection, capture, optional ContextRun, and report identities, and changes that require renewed review;
- whether selected bytes or rendered context are persisted, plus access and retention expectations;
- workspace destination and single-writer/merge behavior for evidence and candidate records.

Owner identity and review fields are supplied claims, not authentication or authorization enforcement. Hashes prove byte identity only. The handoff does not imply evidence completeness, semantic correctness, or candidate acceptance.

The source-acquisition request is narrow: support exact paths at one resolved full Git commit without reading unselected blobs. Keep this outside Core, preserve the existing snapshot digest, avoid a generic provider abstraction, and test the acquisition boundary with positive and negative evidence. If that helper cannot be approved now, a standalone helper must still meet the same no-unselected-read property.

## Questions recorded before the shared contract

- Should preparation be a new CLI command or a distinct mode? This proposal intentionally names neither.
- What is the minimum owner-supplied root identity needed to prevent a path from resolving to a different repository between review and capture?
- How will a full commit and exact blob paths be resolved for linked worktrees, submodules, or repositories with unusual path names?
- Are exact UTF-8 regular files the first-proof limit? How should binary files and explicit external exports be handled?
- Does the handoff persist raw selected bytes, a materialized selected snapshot, a ContextRun report, or immutable references and digests? Define storage, redaction, access, and retention before capture.
- Which source or selection changes invalidate review, and how are workspace edits detected before Copy Me consumes the handoff?
- What happens when the destination is inside another Git checkout, ignored, on a protected branch, or left partially initialized?
- Which workstream is the single writer when Init and Copy Me finish at different times or branches?

These questions preserve the rationale for the original review. The current answers and source ownership are in [Selective adoption handoff](selective-adoption-handoff.md); do not use this historical proposal to infer today's command, record format, or verification status.
