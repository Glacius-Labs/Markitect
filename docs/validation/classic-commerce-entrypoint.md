# Public Classic Commerce entrypoint validation

Dated 2026-10-07. This finite local checkpoint validates the [complete public example](../../examples/classic-commerce/README.md), including its canonical model, selected policies, runtime template, native CLI walkthrough and independent .NET behavior probe. It is a source follow-up, not a new release or a replacement for the held v0.14.1-r1 study candidate.

## Frozen inputs

| Input | Exact identity |
|---|---|
| Example implementation and documentation exercised | `e5404893af83f2269b73308e4c905f5facd808fc` |
| Native runtime | Published Markitect v0.14.1 Windows amd64 |
| Runtime source | `7dbd599c81540c8203a1b7f83afbc335174f4f1f` |
| Native executable SHA-256 | `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4` |
| Prepared fixture source, both fresh repositories | `9fdfa9af424dcf9a039434223c99a7b307c995de` |
| Positive materialized evidence | `e4d883782e183ac9497a70a67c482354e614af59` |
| Bad-business materialized evidence | `8e7003bfe136cf99cb3f2d89e99249ec42f208c1` |
| Toolchain used | Existing SDK 8.0.418, compatible .NET 8 runtime, Python 3.13.3, Git 2.52.0.windows.1 |

Preflight `dotnet --info` and `--list-runtimes` inventory records installed .NET runtime 8.0.24. Per-check receipts capture the selected SDK and successful execution; they do not separately capture `RuntimeInformation.FrameworkDescription` from the executing probe.

No toolchain installation, retargeting, provider call or test-only private helper was needed. Git blobs supplied the example inputs. Each mode used its own disposable Git repository, external ledger, actor logs, fixed-check evidence, and build outputs. The driver checked the published executable digest, fixed source revision, clean source, exact reviewed .NET candidate bytes and output paths, then verified those bytes after Apply and Audit. Executor and Verifier were explicitly deterministic **protocol test doubles**; the .NET check and separately authored console probe ran for real.

## Observed native lifecycle

The first and only smoke attempt completed both controls. Inspection, proposal, Execute and guarded Apply exited 0. Apply reported `materialized-unverified` in both cases. Fresh Verify used the saved Apply-result JSON in a separate native process; Audit was a separate read-only invocation.

| Observation | Positive | Compiling bad-business candidate |
|---|---|---|
| Restore and compile candidate, both scopes | Passed | Passed |
| Restore and compile independent probe, both scopes | Passed | Passed |
| Independent behavior probe, both scopes | Passed | Failed: expected 37.50, received 15.50 |
| Synthetic protocol Verifier runs, both scopes | Passed | Passed |
| Native Verify | Exit 0, `passed` | Exit 1, `failed` |
| Native Audit | Exit 0, `complete` | Exit 2, `incomplete` |

The bad candidate adds quantity to price instead of multiplying. Its failure therefore demonstrates that a successful compiler and synthetic Verifier response do not override a failed project-owned behavior check. It demonstrates refused successful verification and incomplete closure after materialization; it does not demonstrate refused Apply, rollback or general semantic verification.

The positive probe covers normal/fractional/zero-price totals, invalid quantity/price exceptions, and the explicit application boundary. These are finite declared cases; decimal overflow is outside this checkpoint.

## Review and source gates

Independent Luna High static review passed before execution, covering the canonical intent, fixed checks, runtime, documented flow and driver. A separate post-run evidence review passed on the captured compiler/probe and native outputs after clarifying installed versus executing runtime evidence. Review receipts retain file hashes and scope limits.

At implementation source `e5404893af83f2269b73308e4c905f5facd808fc`, the following focused checks passed: fixed-revision root Project check and impact, canonical model compilation, managed-artifact accounting, the existing canonical Apply/Verify/repair example regression, architecture import tests, module checks, and `git diff --check`. The post-smoke source change adds only this validation report, documentation links and its artifact entry; its own fixed-revision structure/accounting results belong in the handoff receipt.

**The initial checkpoint at `b09d6b75a202f4fc6fe4855878a152aec938c29d` left full integration gates pending.** No full Go suite, root `verify`, vet/build/package matrix or Windows/Linux CI result was claimed for that checkpoint. The Worker held the exclusive full-suite slot then. Subsequent integration outcomes are recorded against their exact source revision in a separate versioned local handoff; this historical report does not transfer a prior gate result to changed source. The Linux commands are public rerun instructions; the real-compiler smoke recorded here ran on Windows only.

The subsequent integration source selectively adopts the generic bounded check-budget change from `028c137dac733c0a9c8539727e14bdba55fca789`, with no Government behavior or validation note. Project `go-tests` explicitly declares a maximum 1800-second command limit and `-timeout=30m`; omitted budgets stay at 600 seconds. This is an unreleased source capability, not a change to the immutable native v0.14.1 runtime used above. The schema is regenerated and the Classic CI fingerprint is recomputed from this branch's exact CI bytes. YAML and canonical controller JSON retain their existing naming conventions for gate results. A passing fixed root Verify supplies its declared fresh full Go-suite gate; an identical standalone invocation need not run a second time. Remaining local contribution commands and externally hosted/platform gates are listed separately in the source-bound integration handoff.

## Retained evidence and limits

The local evidence handoff is frozen separately under `.artifacts/classic-public-example/` in the follow-up worktree, with raw stdout/stderr, argv/cwd/exit/durations, runtime and candidate digests, fixed-check receipts, toolchain inventory, independent reviews, source bundle, checksum manifest and archive hash. It is excluded from source and is not a public release asset. Captured receipt paths identify the original runs; reruns use new output directories rather than rewriting old evidence.

No general autonomous engineering reliability, provider quality, economic benefit or human acceptance is established. No daemon, new runtime, study adapter or Government change was introduced. This checkpoint makes the existing Classic alpha independently runnable and provides bounded positive and meaningful negative execution evidence.


## Intent-change follow-up, 2026-10-08

The [owner-intent walkthrough](../../examples/classic-commerce/intent-change.md) closes two public-entry gaps: mapping a concrete request to reviewed canonical intent and explicit check/file owners, and following a later accepted business-rule change through the existing native controller. The example changes minimum order quantity from one to two. Its .NET policy now refers business behavior to the UseCase purpose rather than maintaining another quantity rule. No Core, production Host, provider, scheduler or acceptance lifecycle was added.

| Input or result | Exact identity |
|---|---|
| First scripted example source | `3a30e207f9a3ffef4f5dfc6473a7eaf762e589f2` |
| Second scripted example source | `03e0d755a6497609fcd4a2bceec81e5c9336b95f` |
| Corrected helper / seven-test source | `11967c2c7b5bc5a6304d98425c4d5b6741adb5ae` |
| Native runtime | Published v0.14.1 Windows amd64; SHA-256 `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4` |
| Second attempt initial accepted fixture | `4c8d65f927f5db481a26a6eb32cb577ade4a5c53` |
| Second attempt initial materialized evidence | `7e5f1886409257b7e7222a2ebbc83f875f3bf3f5` |
| Accepted minimum-two fixture source | `146721c7e4ad0bd2da43ddbf4ad79e9c3ba2daf6` |
| Minimum-two materialized evidence | `e3301b2b79ece2c7860a4ddabc5ca786da1f9c1d` |
| Exact accepted two-file diff SHA-256 | `a877d9f581a5cc11d9eb53d0acfae20915d3b64d0cba722275c8f62752b4258e` |
| First incomplete receipt SHA-256 | `034368169c834f96ad1546f922c7465236325b6834dca33421f342ad7d31bf60` |
| Second incomplete receipt SHA-256 | `c34d3fd85b18bf6c744ccb57b3912770f998e17ab43d3b18efcff0c5c421bdd2` |
| Successful continuation receipt SHA-256 | `4d6de2ef93c2ac7c22defffc2b4e24d27adca442c30777eef4d194d824109029` |

Both initial cycles compiled and passed fresh Verify plus Audit. Attempt 1 then stopped because the example expected exit 1 for an invalid saved-run source binding; the native CLI correctly returned exit 2 and the exact binding diagnostic. Commit `03e0d75` corrected only that expectation. The no-mutation assertions after the unexpected exit did not run in attempt 1.

Attempt 2 passed read-only impact/reconcile planning, exposed incomplete old assurance under changed intent, and rejected the old Apply at the CLI saved-run binding boundary. Exact HEAD, index, all four target bytes and ledger bytes stayed unchanged through that refusal. Fresh Execute and Apply succeeded. Apply wrote only the changed handler and Markdown; project-file and EffectAxis bytes were retained. The example then stopped because its reused first-materialization assertion expected all four paths in `written`.

Commit `11967c2` corrects the helper's write-set expectation without weakening full-target checks. The baseline default still requires all reviewed paths. The changed cycle derives its expected writes from baseline versus reviewed bytes and requires exactly handler plus Markdown. Seven focused stdlib tests pass, including strict/default/subset refusal, real fixture markers, explicit change selection, quota refusal and timeout-output preservation.

No third complete pass ran. A separately recorded continuation consumed the exact saved attempt-2 Apply JSON and unchanged runtime/repository. The independently reviewed corrected read-only postcondition passed first. One native `controller-verify --apply-result ... --write` then passed both `commerce-dotnet` and `commerce-markdown`; each scope ran actual SDK selection, offline restore/build of candidate and probe, and the finite behavior probe. Two synthetic protocol Verifier responses remain separate from those compiler/behavior results. A subsequent read-only Audit was complete, and final all-four target-byte, HEAD/index and history checks passed. Both original receipts remain incomplete and byte-identical.

The continuation wrapper's optional `assuranceScopes` summary contains null entries because VerificationResults do not expose a `scopeId` field. This summary is not used as evidence: the native `verifierRuns[].scopeId`, Audit scope/record/result bindings and raw fixed-check receipts identify the two scopes. The raw receipt is preserved with a separate metadata note.

The cumulative development allocation is closed: two full attempts, eleven native mutating starts, nine deterministic role reservations and **196.744 seconds** of outer smoke time, within 2/12/24/1800 limits. There were no model/provider calls, tool installations, publication, study-fixture edits or study-candidate substitutions. The continuation adds one Verify start; it does not repeat Execute or Apply. Source tests and normal contribution checks are recorded separately from this smoke allocation.

The local source checks passed: fixed check/context/impact and canonical model/context at `3a30e20`; managed-artifact accounting, configured Module checks, schema, formatting, vet, build, module verification and architecture imports. The first standalone Module invocation omitted required `--hooks`/`--pipelines`; that diagnostic remains preserved, and the corrected declared argv passed. The seven-test correction is source-bound to `11967c2`. A new complete Go suite and hosted Windows/Linux CI were not run for this example/documentation follow-up; the shared heavy slot belongs to Worker. The earlier exact-base `8927704` full-suite evidence is not relabeled as a new-candidate result.

The native initial-to-change journey is demonstrated with the explicit saved-Apply continuation. **The corrected public script has not completed an uninterrupted replay.** Keep that replay and release/platform gates open for a separately bounded next allocation. This checkpoint does not establish real-agent quality, authenticated owner acceptance, complete repository semantics, comparative benefit or stable release support.
