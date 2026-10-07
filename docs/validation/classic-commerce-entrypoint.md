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

**Full integration gates remain pending for this follow-up.** No fresh full `go test ./... -count=1`, full root `verify`, vet/build/package matrix, or Windows/Linux CI result is claimed for this candidate. The Worker owns the exclusive full-suite slot. Prior release gates are evidence for their prior source, not this new example. The Linux commands are public rerun instructions; this new real-compiler smoke was run on Windows only.

## Retained evidence and limits

The local evidence handoff is frozen separately under `.artifacts/classic-public-example/` in the follow-up worktree, with raw stdout/stderr, argv/cwd/exit/durations, runtime and candidate digests, fixed-check receipts, toolchain inventory, independent reviews, source bundle, checksum manifest and archive hash. It is excluded from source and is not a public release asset. Captured receipt paths identify the original runs; reruns use new output directories rather than rewriting old evidence.

No general autonomous engineering reliability, provider quality, economic benefit or human acceptance is established. No daemon, new runtime, study adapter or Government change was introduced. This checkpoint makes the existing Classic alpha independently runnable and provides bounded positive and meaningful negative execution evidence.
