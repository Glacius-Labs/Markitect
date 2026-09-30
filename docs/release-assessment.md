# Markitect source release assessment

**Recorded:** 2026-09-30  
**Scope:** source distribution candidate `0.1.0-rc.3`; this is an evidence summary, not release authorization.

## Assessment

The reviewed verifier and bootstrap findings were repaired, independently rechecked, merged, and covered by successful hosted Windows and Linux CI. A later test-only correction made the bootstrap root-discovery fixture compare normalized temporary paths. The downloaded distribution was checked as a four-file set and passed the standalone bootstrap test plus the authoring and minimal-example smoke checks.

This supports the bounded private source-distribution workflow for the recorded candidate. It does not establish a stable production release, provider-runtime acceptance, Konfyra acceptance, or human acceptance. The Konfyra assessment remains separate and must be bound to its final consumer candidate.

## Closed review findings

| Finding | Repair and disposition | Evidence |
|---|---|---|
| Inherited Git repository variables could redirect fixed reads and protected-branch checks to a different checkout. | Shared Git invocation binds the selected repository, removes inherited `GIT_*` overrides and disables replacement objects. Regressions cover foreign repositories, replacement refs and refused writes to the actual protected branch. Independent review found no remaining issue in this scope. | PR1 source and Windows/Linux CI; `.artifacts/readiness-review-ff81205.md`. |
| The native Go verification gate could inherit `GOFLAGS` and report success without running tests; abnormal child termination could be mapped to incomplete evidence instead of gate failure. | Candidate `8f3b11a6a67a60293673c46fe6bff4206023173e` clears Go selection/workspace overrides, sets `-count=1`, tests that the snapshot's bootstrap test actually ran, and maps abnormal exits as gate failures. The targeted recheck closed both findings. PR1 merged as `fb000760ccc7b64a15e9c00a6961207fff154127`; its reviewed tree matched the rechecked candidate. | [PR1 main CI, run 36745959339](https://github.com/Glacius-Labs/Markitect/actions/runs/36745959339) — success. Initial and targeted reports: `.artifacts/readiness-review-ff81205.md`, `.artifacts/readiness-review-8f3b11a.md`. |
| The bootstrap's environment filter dropped non-Go credentials such as `GOOGLE_APPLICATION_CREDENTIALS`; prerequisites understated the required Go version; cache reuse still probes the selected toolchain. | An explicit list of compiler/build variables replaces the blanket `GO*` filter, preserving unrelated settings and `GOAUTH`. Documentation now requires Go 1.27.1 or newer and explains that the selected toolchain must be available even for a cache hit. The targeted bootstrap recheck found no new issue. | `.artifacts/bootstrap-review-ff81205.md`, `.artifacts/bootstrap-review-8f3b11a.md`. PR1 CI above exercises the merged source. |
| Bootstrap fixture compared temporary paths without applying the platform's path normalization. | Test-only change `c08d18875e4e8330aa97d30cb3af61db1542a339` compares `discoverRoot` with `filepath.Clean(root)`. It preserves the original failure condition and passes when `GOTMPDIR` is noncanonical. PR2 merged as `d3d6f4e55588f966849de9bf0f71fd3a9e4949f8`. | [PR2 main CI, run 36747050405](https://github.com/Glacius-Labs/Markitect/actions/runs/36747050405) — Windows and Linux success. |

The verifier applies fixed-snapshot profile commands, a ten-minute per-gate limit, a 1 MiB combined-output cap, and a bounded pipe wait. Cancellation targets the direct process; it does not promise to terminate every child a consumer script might create.

## Downloaded distribution identity

The exact four files downloaded from the successful PR2 main run were checked against the reviewed source blobs and package lock:

| File | SHA-256 |
|---|---|
| `tools/markitect/source.zip` | `eeb26389b9a8dcdd3954abe209ae8dcfd44ec64643b03dbde59afde07a80efd2` |
| `markitect.lock.yaml` | `2c5f0f20227289b20735b4a6edbea6cd3f5c305e0c97d4bac24b71dc487a0c8a` |
| `scripts/run-markitect.go` | `43df110dc7b7d105223224b37b9b968f9fa00b0c345c2c3cfdf2215c62e37545` |
| `scripts/markitect-bootstrap_test.go` | `b20e34d9fe00c6520db8ea63f4c6023ec7e3796a749270b21e0561e63b7c60ba` |

The downloaded standalone test passed with a deliberately noncanonical `GOTMPDIR`. The downloaded bootstrap built and passed `authoring` and the minimal-example `check`; the Windows executable reported tool digest `1f2eb13ca12a59914ac4b3fdba40352e60d33952114c3c10735bd2afb32225a0`. The exact four-file artifact and its verification were recorded under `.artifacts/hosted-d3d6f4e5/`.

## Pin upgrade, cache, and rollback evidence

The bounded local RC3 pin exercise used the same source archive SHA above. In an isolated consumer, the new four-file pin was committed as `a756e2878ba4c38cc30d173cc6ca6c1944b454a5`; cold and warm `version`/`check` calls passed, and `authoring` returned the embedded authoring Skill. A normal revert produced `109ecc01b001e67375c5ca75e1262fc8765b08d9`; all four integration-file blobs then matched the RC2 baseline commit `93119c7bf105d9196d0bdc0651c79055e123da75`. See `.artifacts/pin-upgrade-8f3b11a-report.md` for the exact local commands and cache digests.

That upgrade exercise predates PR2's test-fixture correction: its paired test has SHA-256 `6141b824ba0d91401eec6f6309dbbe8422d26a11dc75acc45863484ffda52bd3`. Source archive, lock, bootstrap runtime and executable are identical to the final set above. The corrected paired test was verified separately through the actual final download; the earlier rollback report has not been relabelled as a test of different bytes.

Archive, extracted-source, cached-executable and build-stamp digests are checked by the bootstrap; cache eligibility is tied to the source digest, actual Go toolchain, build policy and executable digest. The final hosted CI runs the repository and standalone bootstrap suites. The bounded local upgrade exercise did not independently repeat the archive-tamper case; that exercise alone is not evidence for the tamper gate.

## Release boundary

- The exercised support boundary is Windows amd64 and Linux amd64 with Git and Go 1.27.1 or newer. Konfyra and Cockpit repository profiles also require Python for their existing consumer-owned checks.
- The hosted artifact is a private, short-lived workflow artifact (30-day retention), not a GitHub Release or tag. Consumers choose and review their own pin update; CI does not update consumer repositories.
- Consumers continue to own provider rendering/runtime behavior and their own CI and human acceptance. A passing Markitect check or verification does not replace those decisions.
- Public release still requires decisions on the API domain, license, and source provenance/signing policy. The release candidate evidence does not close those decisions or imply a production release.

