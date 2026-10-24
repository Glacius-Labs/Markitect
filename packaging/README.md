# Packaging and distribution

The owner retired the old Project/Domain releases and binary downloads on 17 October 2026. There is currently no supported binary release of the new product. The Git tags no longer exist on GitHub; the commit behind each is recorded in the [retired release mapping](../docs/validation/retired-releases-20261017.json), and the 18 retired release versions `v0.1.0` to `v0.14.1` must not be reused. [Release identities, asset digests and attestations](../docs/validation/distribution-retirement-20261017.md) are retained. Use [current source](../README.md#run-the-current-source), [Project workflow](../docs/guides/project-workflow.md) and the [delivery plan](../docs/implementation-plan.md).

## What lives here

`packaging/` holds what ships besides the binary: [`runtime/`](runtime/README.md) (the runtime container image and its project-image examples), `bootstrap/` (the consumer bootstrap `run-markitect.go` that release bundles copy into `.markitect/bootstrap/run.go`, with its test) and this guide.

## Current source release tooling

Independent `markitect-release` commands prepare local distributions. They do not publish, download an old release, run a provider or establish Apply readiness.

```powershell
go run ./cmd/markitect-release package --repo . --output ABSENT_DIRECTORY
go run ./cmd/markitect-release bundle --repo . --revision FULL_COMMIT --output ABSENT_BUNDLE.zip
go run ./cmd/markitect-release install --repo CONSUMER --bundle BUNDLE.zip --sha256 EXPECTED_SHA256
go run ./cmd/markitect-release version
go run ./cmd/markitect-release licenses
```

`package` writes an absent source directory. `bundle` binds an absent ZIP to the exact source commit. `install` verifies the supplied digest and previews the complete project pin; only `--write` applies it. The source/bootstrap and portable build inject `internal/buildinfo.version`. Root `cmd/` and `internal/` are the supported layout; mixed or legacy `src/` layouts are refused.

## Release contents and verification

A future public release needs a fresh version tag, exact reviewed source, passing platform gates, a bundle, native Windows/Linux binaries and provenance. The publication tool verifies source/run/attempt identity, asset bytes, immutable-release policy and attestations. It never overwrites a tag or asset. Retired version tags must not be reused.

The README keeps its release-generation markers for a future verified distribution. `markitect-release distribution --tag NEW_TAG --repo .` checks an actually published immutable release; `--write` updates local installation text and generated WinGet manifests. `--export-winget ABSENT_DIRECTORY` exports manifests for a separately reviewed catalog submission. No old manifest or catalog-validation workflow remains active on Main. Do not call these commands with a retired release tag.

## Preview and install

Inspect the source-bound local bundle and its SHA-256, preview the operations, then use `install --write` only within the intended consumer repository. The shared writer lock remains held through complete-pin readback, and the manifest is written last. Partial writes stay visible for inspection; they are never reported as an atomic success. Unchanged pins retain their bytes and modes.

The standalone `.markitect/bootstrap/run.go` imports only the Go standard library because it runs inside the consumer's module. It validates the lock and source archive before building the pinned CLI. Changes to that contract must keep bootstrap and release-side tests aligned.

```powershell
go test ./packaging/bootstrap/run-markitect.go ./packaging/bootstrap/run-markitect_test.go -count=1
```

## Runtime requirements

Use Git and the Go version in `go.mod`. Runtime containers and their separately authorized provider capabilities are described in [the runtime guide](runtime/README.md). Packaging or installation is not a runtime or Apply demonstration.

## Owner publication

[Operations](../docs/guides/operations.md#release-operations) owns publication requirements. The retained release workflow builds and checks new tagged source; it grants no authority to publish. Publication requires the owner's explicit release instruction and exact source/run/assets. Inspect the read-only publication plan before invoking its write operation, and verify the resulting release and every asset afterward.

## Historical content-package command

The old product, bootstrap formats, download walkthroughs and `markitect package` command remain at [the pre-retirement guide](https://github.com/Glacius-Labs/Markitect/blob/1f4a1ede214d05fd2fee615b3905207df7702426/integration/README.md), pinned by the archive ref `archive/pre-reset-1f4a1ede`. They are historical contracts, not another active command surface.
