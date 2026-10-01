# Markitect production assessment

**Recorded:** 2026-10-01. This dated assessment records the release evidence verified at that time and distinguishes it from source behavior. It reports only the Markitect source, its distributions, and source-owned gates; project adoption and acceptance are outside its scope.

## v0.3.1 published release

At the time this assessment was recorded, the immutable [`v0.3.1` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.3.1) was verified against source commit [`52f0bd6`](https://github.com/Glacius-Labs/Markitect/commit/52f0bd64ce64759be10ffdee180c2cf7c939d49b), release ID `400495147`. Main CI [run 36787681871](https://github.com/Glacius-Labs/Markitect/actions/runs/36787681871) and Release CI [run 36787725843](https://github.com/Glacius-Labs/Markitect/actions/runs/36787725843) passed for that release source. This release predates the minimal initialization behavior described in the current source.

## Project initialization source behavior

The source under assessment adds minimal project initialization. Preview is read-only; write recomputes and validates its plan, requires a named non-protected branch, and creates only the Project file and one area README. Structural checking is not complete verification: an initialized project with no owner-declared checks must still receive `incomplete-evidence` from `verify`. These are source-level claims, not release evidence.

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
