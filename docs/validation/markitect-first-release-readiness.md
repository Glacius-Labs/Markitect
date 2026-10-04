# Markitect-first v0.13.0 release readiness

**Status:** candidate work in progress; release evidence pending. This record defines the proposed scope and the evidence required before publishing v0.13.0. It is not a release receipt. Published v0.12.0 remains the current stable contract until a v0.13.0 release is published and independently verified.

The additive pre-1.0 version is v0.13.0. The canonical [Markitect-first Change workflow](../../internal/authoring/resources/workflow-markitect-first-change.yaml) owns the protocol; [Usage](../usage.md#markitect-first-release-candidate) and [Markitect-first](../markitect-first.md) describe the user-facing entrypoint and accounting check. The root Project and generated provider Skills make that workflow executable from the repository's engineering entrypoints. The [managed artifact coverage design](../design/managed-artifact-coverage.md) defines the bounded path-accounting helper.

The passing main CI run **37170101983** is bound to baseline `5cc9718c740185f61a92fbaf3f26cd8b7a17d031`. It predates the current candidate changes and is not evidence for the final release tree. Candidate commit, final Windows/Linux CI, tagged release workflow, independent review, public assets, and publication are all pending.

## Proposed release scope

| Capability | v0.13.0 classification | Acceptance evidence required for this release | Current record |
|---|---|---|---|
| Read-only analysis of ordinary PolicyResult failures | Supported, bounded CLI behavior | Preserve strict defaults and structural blockers; diagnostic Context/Impact must remain explicitly opted in and retain policy-failure exit status. Run its focused fixture and packaged command gates on the exact final candidate. | Existing source and validation are documented; final-candidate gate evidence pending. |
| Selective `prepare` and `copy-me` | Supported alpha-versioned byte/reference infrastructure | Exercise a real owner-selected, approved scope and capture; validate a separate candidate and its supplied decision bindings from captured bytes; preserve the no-acquisition, no-inference, no-automatic-adoption boundary. Include the CLI fixture in source and packaged gates. | Source validation and a sanitized public replay are documented. The source roadmap reports an owner-approved private capture and four candidates; restricted evidence identifiers and final-candidate packaged results are pending. No private adopter material is reproduced here. |
| Markitect-first Change workflow and generated Skills | Supported canonical workflow | Verify the canonical Workflow is included by `markitect authoring`, generated Codex/Claude entrypoints point to it, and root Project checks/formatting/context/verification pass on the fixed candidate. | Implemented in the current candidate working tree; final generated-output and fixed-snapshot proof pending. |
| Standalone artifact-coverage helper | Supported, bounded path accounting | Build the helper from the pinned source package; exercise declared roots, owners, exclusions, missing/unowned paths, generated-output ownership, and nested independent Projects. Run the dogfood helper against a fixed candidate revision. Pair coverage output with `check` for generated-byte freshness; accounting alone does not prove semantic correctness or freshness. | Root Project and coverage helper are present in the candidate working tree. The nested independent-project stale-output correction is underway; its review and candidate gates remain pending. The helper is a source-package Go command, not an additional release binary asset. |
| .NET and source-acquisition hardening | Supported bounded checks | Retain literal-project-reference namespace and ambiguity controls; prove selected Git blobs remain local-only, exact, and digest-bound. Run focused tests, source-package gates, and full Windows/Linux CI on the final candidate. | Prior wave evidence exists for .NET hardening; selective acquisition evidence is documented. Final-candidate gate evidence pending. |

The preparation and Copy Me exercise must show that a real owner-approved scope can produce a reviewable bounded candidate using only its selected evidence. Formal candidate acceptance, canonical adoption, and an adopter-owned implementation diff are separate human decisions. The Konfyra adoption track may remain pending after release; release evidence must not describe it as accepted or adopted. See the [selective handoff validation](selective-adoption-handoff.md) and [real-project pilot](real-project-adoption-pilot.md) for the existing evidence limits and preserved negative findings.

## Experimental and excluded scope

| Capability | Classification | Release treatment |
|---|---|---|
| Standalone GitHub and Azure DevOps consumers | Experimental, source-only; offline supplied-capture consumers | Do not advertise as live provider integrations or stable provider-state verification. Captures are supplied evidence, not authenticated or current remote state. No provider Apply is included. |
| MCP interface | Experimental | Keep its existing experimental status and evaluation limits; it is not a supported stable workflow in this release. |
| Broader provider protocol, Core/Domain/SPI expansion, or live provider Apply | Deferred/out of scope | No new graph composition or reusable Core primitive is justified by this release scope. Do not expand authority or infer project-specific meaning. |
| Product benefit, adoption, or reduced human review claims | Not established | Do not infer productivity, correctness, or business benefit from source tests, path accounting, capture validation, or release gates. |

## Release acceptance matrix

Every result below must bind to the same final full commit. The baseline CI run above does not satisfy any final-candidate row.

| Gate | Required evidence | Status / identifier |
|---|---|---|
| Candidate identity and scope | Clean, reviewed full commit on `main`, version `0.13.0`, exact scope above, and no unreviewed release-only edits | Pending |
| Independent review | Independent full-diff review of the exact candidate, including the nested-project output-ownership fix, generated Skills, coverage helper, release metadata, and documented boundaries | Pending |
| Real preparation and candidate exercise | Restricted owner-approved scope/capture reference; byte/reference validation result; candidate/queue/handoff digests or approved evidence locator; confirmation that only selected captured bytes were used | Pending; retain private evidence outside this repository |
| Adoption disposition | Explicitly record that adoption/acceptance is a separate decision; if Konfyra has not accepted or applied the candidate, state that it remains pending | Pending owner disposition; not a generic release blocker |
| CI source gates | `go mod verify`, `go test ./...`, `go vet ./...`, `go build ./...`, generated schemas, fixtures/examples, dogfood `check`, formatting, fixed Context, artifact coverage, and configured fixed-snapshot `verify` on Windows and Linux | Pending final candidate SHA and run links |
| Packaged behavior | Source package and standalone bootstrap smoke; policy-analysis and selective-adoption CLI fixtures against packaged source; helper build and dogfood invocation from the exact candidate package | Pending final candidate SHA and Windows/Linux run links |
| Tagged Release workflow | Exact tag `v0.13.0` points to the reviewed candidate; quality, preflight, Windows/Linux bundle installation/bootstrap smoke, and exact-source asset assembly all succeed | Pending tag, run ID/attempt, and result |
| Owner publication preflight | Read-only publisher plan matches the successful run, remote tag, source commit, provenance, and four exact assets; owner reviews the plan | Pending |
| Immutable publication and attestations | Publisher reports verified immutable release; `gh release verify` and each `gh release verify-asset` succeed for bundle, Windows binary, Linux binary, and provenance | Pending release ID and verification results |
| Public distribution and install | Read-only `markitect-release distribution` check succeeds; then regenerate README and canonical WinGet metadata from verified assets, and smoke the downloaded platform binaries/bootstrap | Pending public digests and platform results |
| Final release record | Update the production assessment with exact source SHA, CI/run IDs, release ID, asset digests, attestations, and installation evidence; preserve v0.12.0 evidence unchanged | Pending |

## Exact release gates and procedure

The normal CI source gates include:

```text
go mod verify
go test ./...
go vet ./...
go build ./...
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect check --repo . --revision CANDIDATE
go run ./cmd/markitect format --repo .
go run ./cmd/markitect-check-artifacts --repo . --config markitect-artifacts.yaml
go run ./cmd/markitect context --repo . --revision CANDIDATE --namespace development --kind Skill --name engineering-change
go run ./cmd/markitect verify --repo . --revision CANDIDATE
go test ./examples -run '^TestSelectiveAdoptionCLI$' -count=1
```

CI also checks the existing executable examples, generated resources and projections, package/install/bootstrap commands, and reconciliation controls. The selective adoption CLI replay must run with the candidate source package in both platform jobs (`MARKITECT_ADOPTION_BINARY`); source-only unit tests are not a substitute for that packaged invocation. The tagged Release workflow reuses CI, rebuilds and compares the deterministic bundle, and runs bundle installation/bootstrap smoke on Windows and Linux before retaining the release assets.

After candidate review and successful main CI, the release owner may create and push the exact `v0.13.0` tag to trigger Release CI. The publication commands and fail-closed recovery rules are in [Integration](../../integration/README.md#owner-publication); the release owner must inspect the read-only plan before choosing `--publish`. After verified publication, run:

```powershell
gh release verify v0.13.0 --repo Glacius-Labs/Markitect
go run ./cmd/markitect-release distribution --tag v0.13.0 --repo .
```

Verify all four published assets individually with `gh release verify-asset`, then run the distribution tool with `--write` only after its read-only check succeeds. Record exact public digests before updating install instructions. Never retag, replace an immutable release, or claim publication from a workflow artifact alone.
