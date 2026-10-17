# Markitect source release assessment

**Recorded:** 2026-09-30. This assessment preserves general source-packaging and verifier findings from release candidate `0.1.0-rc.3`. It predates immutable `v0.1.0`; the later [production assessment](production-assessment.md) records that release. It contains no project-adoption or acceptance report.

## Findings retained in the product

| Finding | Source repair and general lesson |
|---|---|
| Inherited Git repository variables could redirect fixed reads and branch checks. | Bind every Git operation to the selected repository, discard inherited `GIT_*` overrides, disable replacement objects, and test foreign repositories and replacement refs. |
| A Go verification command could inherit environment options and appear successful without executing tests. | Clear Go selection/workspace overrides, force test execution with `-count=1`, and verify that expected tests ran. Abnormal exits must remain gate failures. |
| Filtering all `GO*` variables also removed unrelated credentials. | Filter an explicit list of compiler/build variables; preserve unrelated environment needed by dependencies. Document the minimum Go version and toolchain selection even on cache hits. |
| A bootstrap fixture compared temporary paths without platform normalization. | Normalize expected paths with the platform path library while preserving the test's failure condition. The correction passed actual Windows and Linux release gates. |

The fixed-snapshot verifier bounds each direct gate process to ten minutes, captures at most one MiB of combined output, and bounds pipe waiting. Cancellation targets the direct child only; arbitrary descendants are not guaranteed to stop. These bounds are operational limits, not a security sandbox.

## Verified v0.1.0-rc.3 source package

The downloaded four-file package was checked against canonical source blobs and its lock:

| File | SHA-256 |
|---|---|
| `tools/markitect/source.zip` | `eeb26389b9a8dcdd3954abe209ae8dcfd44ec64643b03dbde59afde07a80efd2` |
| `markitect.lock.yaml` | `2c5f0f20227289b20735b4a6edbea6cd3f5c305e0c97d4bac24b71dc487a0c8a` |
| `scripts/run-markitect.go` | `43df110dc7b7d105223224b37b9b968f9fa00b0c345c2c3cfdf2215c62e37545` |
| `scripts/markitect-bootstrap_test.go` | `b20e34d9fe00c6520db8ea63f4c6023ec7e3796a749270b21e0561e63b7c60ba` |

The downloaded test passed with a noncanonical temporary directory. The downloaded bootstrap built and passed the embedded-authoring and minimal-example smoke checks. A preceding test-only path normalization correction had a separate successful Windows/Linux run. These are historical RC3 package checks, not a substitute for the later v0.1.0 release evidence.

## Scope at that checkpoint

- Windows amd64 and Linux amd64 were exercised with Git and Go 1.27.1 or newer.
- CI uploaded a private, short-lived artifact; it was transport, not a GitHub Release.
- Project-owned outputs and decisions remained outside the source-package acceptance boundary.
- The source release did not settle a public license, public API domain, or public distribution policy.
