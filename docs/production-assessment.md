# Markitect production assessment

**Recorded:** 2026-10-01. This dated assessment records the release evidence verified at that time and distinguishes it from source behavior. It reports only the Markitect source, its distributions, and source-owned gates; project adoption and acceptance are outside its scope.

## v0.7.0 published release

The immutable [v0.7.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.7.0), release ID `401286207`, binds [source commit `857a77f`](https://github.com/Glacius-Labs/Markitect/commit/857a77f86b3a4056c7973fed49d12ce2e6bf9b28), merged from [PR 31](https://github.com/Glacius-Labs/Markitect/pull/31). Its fixed-run context request binds a committed task snapshot and explicitly selected source paths to one full Git revision, reporting selected hashes and missing inputs. This records product capability only; no adopting-project implementation effectiveness or time/cost savings are claimed here. The annotated tag is unsigned; the immutable release and its asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36911228378](https://github.com/Glacius-Labs/Markitect/actions/runs/36911228378) | Windows/Linux source gates passed for the exact reviewed source tree; its tree is identical to the release commit |
| Tagged release | [Run 36913950360](https://github.com/Glacius-Labs/Markitect/actions/runs/36913950360), attempt 1 | Windows/Linux source and bundle install smoke gates passed; exact-source assets and provenance were assembled |
| Owner publisher | Release `401286207` | `published-verified`; owner preflight, run artifact, tag, payload digests, immutable release, and each asset attestation were verified |
| Public distribution metadata | `markitect-release distribution --tag v0.7.0 --repo . --write` | Public immutable release and asset attestations were checked; README install blocks and three versioned WinGet manifests were generated from the attested assets |
| Public downloads | Versioned unauthenticated Windows/Linux download URLs | Distribution verification matched the four downloaded asset digests to provenance and the successful release run |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.7.0-bundle.zip` | `461ae101e3cc50f15b315382162fff1f75f4d91266a0e96790871e80b93c9db5` |
| `markitect-v0.7.0-linux-amd64` | `384718c6574b28ffdcce121a17bda167d963996c3c954e0c27a33493cd9725c8` |
| `markitect-v0.7.0-windows-amd64.exe` | `d797dbbd26d2c40b819b1519dd554701c407505091c9494baa30ff1447a4be0f` |
| `markitect-v0.7.0-provenance.yaml` | `08f0ab103e33f0f8acbf5a1428ccc85b6e552ec5ed11d22136886610f4cba0f6` |

The generated WinGet manifests are local versioned metadata; no community catalog submission or package-manager installation result is claimed. The release gates establish the Markitect source and package checks that ran, not adopting-project acceptance.

## v0.6.0 published release

The immutable [v0.6.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.6.0), release ID `401237667`, binds [source commit 810a86f](https://github.com/Glacius-Labs/Markitect/commit/810a86ffe13aef065f38e70ae49d40911f447af2) from [PR 27](https://github.com/Glacius-Labs/Markitect/pull/27). It ships canonical-owner authoring guidance, optional snapshot router validation and the provider diagnostic/shared-agent-link corrections integrated after v0.5.0. The tag is annotated and unsigned; GitHub release and all asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36905108369](https://github.com/Glacius-Labs/Markitect/actions/runs/36905108369) | Windows/Linux tests, vet, build, schema, executable examples and package smoke passed |
| Tagged release | [Run 36905582096](https://github.com/Glacius-Labs/Markitect/actions/runs/36905582096), attempt 1 | Deterministic bundle and both-platform bundle install/bootstrap gates passed; four exact-source assets assembled |
| Owner publisher | Release `401237667` | `published-verified`: owner plan approved, run artifact/tag/manifest checked, immutable release and every asset attestation verified |
| Public downloads | Versioned unauthenticated Windows/Linux download URLs | Bytes matched the attested hashes; Windows executable reported `Markitect 0.6.0 (windows/amd64)` |
| Published native benchmark | [Run 36906676251](https://github.com/Glacius-Labs/Markitect/actions/runs/36906676251) | Windows/Linux fixture v1 completed for attested v0.6.0 and v0.5.0 binaries; raw JSON and summaries retained as run artifacts |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.6.0-bundle.zip` | `6fc7ad77b13e3b25d5666f7136a7a631b62bfcd931a5454610b96de12a2981d1` |
| `markitect-v0.6.0-linux-amd64` | `52f017f191a21ef352e602da15e7db7ea5d71b0fa0d0fdcaae9d166def00948c` |
| `markitect-v0.6.0-windows-amd64.exe` | `7bd08165e11e2cf315307c49b65d7ab08f61d75dc8a178ad229bfea450067280` |
| `markitect-v0.6.0-provenance.yaml` | `b13009121cab1e748db5afe5c04a39a9c298e7d794233301ba4f2f8b18aa19ae` |

Local Windows tests exposed transient file-sharing locks during fixture cleanup. The release candidate retries only Windows access-denied/sharing-violation cleanup failures within a bounded two-second window and preserves the final error. The complete local test suite and both platform gates passed afterward. This changes test cleanup, not production refusal or write behavior.

Benchmark timings remain diagnostic and do not establish a speed gain or an adopting project's semantic/human acceptance. Router checking establishes snapshot navigation structure; the actual canonical-owner authoring decision remains a separate observed acceptance case. Consumer pins and project-local acceptance records belong in the adopting repository.

## v0.5.0 published release

The immutable [v0.5.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.5.0), release ID 401138862, points to [source commit 7883734](https://github.com/Glacius-Labs/Markitect/commit/78837346d0cb27c95b3a10f0a469b2709b757a83) from [PR 17](https://github.com/Glacius-Labs/Markitect/pull/17). It adds explicitly selected provider adapters and opt-in sourced consistency checks. The tag object itself is unsigned; the immutable-release attestation was verified separately.

| Gate | Evidence | Result |
|---|---|---|
| Source CI | [Main run 36887739125](https://github.com/Glacius-Labs/Markitect/actions/runs/36887739125) | Windows and Linux source checks passed at the release commit |
| Release CI | [Run 36889429215](https://github.com/Glacius-Labs/Markitect/actions/runs/36889429215) | Source checks, bundle build and installation smoke passed on Windows and Linux; run artifact contained four exact-source assets |
| Published release | Release 401138862 | Release is immutable and not a draft; tag target is the source commit; `gh release verify` and all four local `gh release verify-asset` checks succeeded |
| Native Windows asset | Attested executable | `version` reported 0.5.0 and `check --repo examples/minimal` passed |
| Post-publication benchmark | [Run 36890129330](https://github.com/Glacius-Labs/Markitect/actions/runs/36890129330) | Failed during job setup because one upload-artifact action pin was one character short; no measurements were made |
| Recovery benchmark | [Run 36891661794](https://github.com/Glacius-Labs/Markitect/actions/runs/36891661794) | Repaired workflow ran on main for the same immutable v0.5.0 tag; Windows and Linux fixture v1 passed and each uploaded raw JSON and a summary |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.5.0-bundle.zip` | `25c995f40a8e704197ee4245dc8daf37255adfb2dc2af6524c25d9b638c97da0` |
| `markitect-v0.5.0-linux-amd64` | `e9eaa87fa36bb30578aaed0bef5bae2639689b39ac5bcd35c2d0632323d794cc` |
| `markitect-v0.5.0-windows-amd64.exe` | `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178` |
| `markitect-v0.5.0-provenance.yaml` | `1f0713059bc6962e41d1b1dc4013e1ec479ed69924582e550608552c53e20281` |

The verified provenance names source commit 78837346d0cb27c95b3a10f0a469b2709b757a83 and workflow run 36889429215. The failed benchmark is separate from release validity. The recovery run measured the same attested v0.5.0 and v0.4.1 binaries without changing either release; its short timings and range differences do not establish a speed gain or regression. An adopting project's installed pin and acceptance remain unverified here.

## v0.4.1 published release

The immutable [`v0.4.1` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.1), release ID `401051061`, points to source commit [`f424432`](https://github.com/Glacius-Labs/Markitect/commit/f4244324b04d4dd748ae13cea43e2455b251a5de) from [PR 12](https://github.com/Glacius-Labs/Markitect/pull/12). The release adds Apache-2.0 licensing and repository guidance; product behavior remains that of v0.4.0.

| Gate | Evidence | Result |
|---|---|---|
| Candidate and main CI | [Main run 36874976547](https://github.com/Glacius-Labs/Markitect/actions/runs/36874976547) | Windows and Linux source checks passed at the release commit |
| Release CI | [Run 36875604747](https://github.com/Glacius-Labs/Markitect/actions/runs/36875604747) | Source gates, bundle and installation smoke passed on Windows and Linux |
| Owner publisher | Release `401051061` | Exact run artifact and tag checked; release published immutable; a repeat `gh release verify` and each of the four `gh release verify-asset` commands succeeded |
| Native Windows asset | Released executable, digest below | `version` reported 0.4.1; `licenses` showed third-party notices and the Markitect Apache-2.0 reference |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.4.1-bundle.zip` | `440e0829e405c4a793eaa02fe102cdabc6d596ed142d3d472f35182d37c13eab` |
| `markitect-v0.4.1-linux-amd64` | `a02d0eb0ada42d95012da0965d471e6e31c4f5d607ac849fb8bd29d85b9b65a3` |
| `markitect-v0.4.1-windows-amd64.exe` | `419f61e99a3ee09c4d87624b44755bacbcb796244cffe7a1fa148da4c79a35f1` |
| `markitect-v0.4.1-provenance.yaml` | `5bfcf67cb3f30a16f5ca7a879bfd1533762000319133e702a171e2db01798c56` |

The publisher's immediate post-publication attestation request returned an error, reporting `published-unverified`. The subsequent direct GitHub release and asset verification succeeded without changing the immutable release. This records the transient verification result instead of claiming that the publisher itself completed successfully.

## v0.4.0 published release

The immutable [`v0.4.0` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.0) was verified against source commit [`1f0d17b`](https://github.com/Glacius-Labs/Markitect/commit/1f0d17bff22143afc6b2b63b4ea8f17486c818e6), release ID `400944013`. Its tree matches reviewed candidate `d7bb86434559bce52a9f7d8078d656c04ecdb024` from [PR 9](https://github.com/Glacius-Labs/Markitect/pull/9).

| Gate | Evidence | Result |
|---|---|---|
| Candidate CI | [Run 36860632114](https://github.com/Glacius-Labs/Markitect/actions/runs/36860632114) | Windows and Linux passed, including packaged commands and project initialization |
| Main CI | [Run 36860951014](https://github.com/Glacius-Labs/Markitect/actions/runs/36860951014) | Passed at the release source commit |
| Release CI | [Run 36861035636](https://github.com/Glacius-Labs/Markitect/actions/runs/36861035636) | Source gates, deterministic bundle and installation/bootstrap smoke passed on both platforms |
| Owner publisher | Release `400944013` | `published-verified`: exact workflow artifact, tag/commit, immutable release and all four asset attestations verified |
| Native Windows asset | Downloaded release executable, digest below | Version/notices, read-only init preview, initialization and structural check passed; fixed verification correctly returned incomplete evidence for the new Project without declared checks |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.4.0-bundle.zip` | `7966d53a1fe81e2b65d1896f267d708ea6ab95dc08f041476664193a5f10d4c3` |
| `markitect-v0.4.0-linux-amd64` | `8487c3f3258992e89d890ddf036945ce4f026f504d3ed7d80b133b4c3b576608` |
| `markitect-v0.4.0-windows-amd64.exe` | `67d62b1ebdd1224dc18591c72c98c7d8e3f395cedd137ffff1ed3cd66eebc719` |
| `markitect-v0.4.0-provenance.yaml` | `0a852d3b89a5430eaf882dfc6b0bebd53d8748cce6a8030da5302da366fd35a5` |

The earlier [candidate run 36860209819](https://github.com/Glacius-Labs/Markitect/actions/runs/36860209819) failed the packaged notices command because the bootstrap incorrectly inserted `--repo`. The repair preserves repository-independent notices and help arguments and adds regression coverage. That failed result is retained; only the later candidate passed the final gate.

An independent review found and confirmed the Windows short-path repair at `070f51c1828a123530167d985890039579b28068`; its six reviewed implementation/test files are unchanged in the release. The coordinating reviewer inspected the later notices and bootstrap follow-ups. The [authoring pilot](authoring-pilot.md) was interrupted by a provider usage limit after two completed actors, and its planned blind semantic review did not run. Those measurements make no productivity or cost claim and are not release attestations.

## v0.3.1 published release

At the time this assessment was recorded, the immutable [`v0.3.1` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.3.1) was verified against source commit [`52f0bd6`](https://github.com/Glacius-Labs/Markitect/commit/52f0bd64ce64759be10ffdee180c2cf7c939d49b), release ID `400495147`. Main CI [run 36787681871](https://github.com/Glacius-Labs/Markitect/actions/runs/36787681871) and Release CI [run 36787725843](https://github.com/Glacius-Labs/Markitect/actions/runs/36787725843) passed for that release source. This release predates the minimal initialization behavior described in the current source.

## Project initialization behavior

Version 0.4.0 adds minimal project initialization. Preview is read-only; write recomputes and validates its plan, requires a named non-protected branch, and creates only the Project file and one area README. Structural checking is not complete verification: an initialized project with no owner-declared checks must still receive `incomplete-evidence` from `verify`. The product also provides a code-to-documentation example, bundled third-party notices, and corrected Windows short/long-path handling. Multi-file writes remain non-transactional and the resource API remains alpha.

## v0.2.0 release

The immutable [`v0.2.0` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.2.0) is verified against source commit [`8ff0af4`](https://github.com/Glacius-Labs/Markitect/commit/8ff0af4). GitHub Actions [run 36777150514](https://github.com/Glacius-Labs/Markitect/actions/runs/36777150514) passed for that exact release source. This confirms distribution availability; it does not establish acceptance by an adopting project.

## v0.3.0 source scope

This version adds deterministic offline content archives, exact Project pins, explicit exports, origin-aware dependencies and context, and read-only imported resources. It also recognizes orphaned aggregate rule-adapter outputs during drift checks. The package source and consumer examples exercise the public CLI; the archive, graph, context, impact and review tests cover integrity failures and boundary isolation.

The release requires a reviewed fixed commit, the complete Windows/Linux CI matrix, deterministic bundle and installation smoke tests, and verified immutable release assets. The release's provenance asset records the exact source and hosted run; use that record for publication status rather than treating this source assessment as a release receipt. See [Content packages](content-packages.md) for supported behavior and [Operations](operations.md) for the publication procedure.

### v0.3.1 correction

The cycle detector also rejects an implementation that needs a Contract bound directly back to itself. Version 0.3.0 omitted that self-selected execution relationship and could accept this invalid graph. Regression tests cover both Project-local and package-local bindings; existing immutable releases remain unchanged.

## Historical v0.1.0 source and release evidence

The release is bound to merge commit [`45e0017a3d25d2abd464de934b5ef3a802732d0a`](https://github.com/Glacius-Labs/Markitect/commit/45e0017a3d25d2abd464de934b5ef3a802732d0a), whose tree matches reviewed candidate [`4adb45964a172a77a1355c6ea5ac2596c325b90d`](https://github.com/Glacius-Labs/Markitect/commit/4adb45964a172a77a1355c6ea5ac2596c325b90d).

| Gate | Evidence | Result |
|---|---|---|
| Pull request CI | [Run 36762498178](https://github.com/Glacius-Labs/Markitect/actions/runs/36762498178) | Windows and Linux passed |
| Main CI | [Run 36762893903](https://github.com/Glacius-Labs/Markitect/actions/runs/36762893903) | Passed for the merge commit |
| Release build and smoke tests | [Run 36762948855](https://github.com/Glacius-Labs/Markitect/actions/runs/36762948855) | Source gates and bundle/bootstrap checks passed on both platforms |
| Immutable release | [`v0.1.0`](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.1.0), release ID `400355906` | Published 2026-09-30; tag resolves to the merge commit; release and all four asset attestations verified |

The immutable release contains the complete v0.1.0 distribution. Its outer bundle digest is `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6` and the nested source archive digest is `c9bc71b0939982e58d2edfe14e18fb5221c0d947dcac4e672e90f2ae612a2628`.

| Asset | SHA-256 |
|---|---|
| `markitect-v0.1.0-bundle.zip` | `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6` |
| `markitect-v0.1.0-linux-amd64` | `2c338b3601efe54026d5266e592f69cb3e677bfba051f21182d837229a98581f` |
| `markitect-v0.1.0-windows-amd64.exe` | `a6dab3a59c83c29a0f1317982516177be848055b024e07455cafe025847f3431` |
| `markitect-v0.1.0-provenance.yaml` | `41af323fef7b1c1040c25541eb989616724446c5c3222e21fc00046bf2d1cb6a` |

The provenance records source tag/commit, workflow run and attempt, toolchain, and payload digests. It is self-declared; GitHub's immutable-release attestation is authoritative for the tag, commit, and attached assets. GitHub's autogenerated source archive is not the Markitect distribution bundle.

## Product boundary at v0.1.0

The release contained the Go CLI, typed YAML resources and schemas, a dependency graph, structural queries, fixed-snapshot context and impact, review-evidence eligibility, generic and explicitly selected provider views, core authoring resources, and the versioned release installer. The source and bootstrap ran on Windows amd64 and Linux amd64 with Go 1.27.1 or later. The native release binary could preview or apply installation without Go.

The checks establish the product gates that ran. They do not prove prose is true or complete, infer a project's policy, certify an agent's runtime behavior, or constitute an adopting project's decision. The v0.1.0 assets above remain an immutable historical pin; source version 0.2.0 and later versions do not alter them.

## Historical publication route

The tagged workflow built and tested the exact source and uploaded a run-scoped transport artifact. It did not publish a GitHub Release. The Actions `GITHUB_TOKEN` could not read the admin-only immutable-release setting (HTTP 403), so an authorized owner used the local Go publisher and existing GitHub CLI authentication. No personal access token was stored in workflow secrets. The publisher checked the run artifact, exact tag target, asset digests, and immutable result before reporting success.
