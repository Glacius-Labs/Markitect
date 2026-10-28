# Packaging and distribution

The owner retired the old Project/Domain releases and binary downloads on 17 October 2026. There is currently no supported binary release of the new product. The Git tags no longer exist on GitHub; the commit behind each is recorded in the [retired release mapping](../docs/validation/retired-releases-20261017.json), and the 18 retired release versions `v0.1.0` to `v0.14.1` must not be reused. [Release identities, asset digests and attestations](../docs/validation/distribution-retirement-20261017.md) are retained. Use [current source](../README.md#run-the-current-source), the [project workflow](../docs/guides/project-workflow.md) and the [delivery plan](../docs/implementation-plan.md).

## What lives here

`packaging/` holds what ships besides the binary: [`runtime/`](runtime/README.md) (the runtime container image and its project-image examples, kept for the Run engine of step T4), `bootstrap/` (the consumer bootstrap `run-markitect.go` that release bundles copy into `.markitect/bootstrap/run.go`, with its test) and this guide. The standard Libraries and the agent skills are built into the binary ([`internal/libraries`](../internal/libraries/), [`internal/skills`](../internal/skills/)), and `markitect init` installs the skills. V1's Linux runner image was removed with step T1; its recipe is at the [base commit](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/packaging/runner/Dockerfile).

## Current source release tooling

Independent `markitect-release` commands prepare local distributions. They do not publish, download an old release, run a provider or establish Apply readiness.

```powershell
go run ./cmd/markitect-release package --repo . --output ABSENT_DIRECTORY
go run ./cmd/markitect-release bundle --repo . --revision FULL_COMMIT --output ABSENT_BUNDLE.zip
go run ./cmd/markitect-release install --repo CONSUMER --bundle BUNDLE.zip --sha256 EXPECTED_SHA256
go run ./cmd/markitect-release version
go run ./cmd/markitect-release licenses
```

`package` writes an absent source directory. `bundle` binds an absent ZIP to the exact source commit. `install` verifies the supplied digest and previews the complete project pin; only `--write` applies it. The source/bootstrap and portable build inject `internal/buildinfo.version`. Root `cmd/` and `internal/` are the supported layout; mixed or legacy `src/` layouts, unsupported files, redirection, nested modules and archive collisions are refused.

## Preview and install

Inspect the source-bound local bundle and its SHA-256, preview the operations, then use `install --write` only within the intended consumer repository. The shared writer lock remains held through complete-pin readback, and the manifest is written last. Partial writes stay visible for inspection; they are never reported as an atomic success. Unchanged pins retain their bytes and modes.

The standalone `.markitect/bootstrap/run.go` imports only the Go standard library because it runs inside the consumer's module. It validates the lock and source archive before building the pinned CLI. Changes to that contract must keep bootstrap and release-side tests aligned.

```powershell
go test ./packaging/bootstrap/run-markitect.go ./packaging/bootstrap/run-markitect_test.go -count=1
```

## Runtime requirements

Use Git and the Go version in `go.mod`. Runtime containers and their separately authorized provider capabilities are described in [the runtime guide](runtime/README.md). Packaging or installation is not a runtime or Apply demonstration.

## Release operations

No binary release of the current product is offered; build from [current source](../README.md#run-the-current-source). The [archive index](../docs/archive.md) lists the pinned pre-reset commits behind the retired releases. Local packaging does not publish. The old download-based benchmark and WinGet validation workflow are retired with their fixtures and manifests. Dated release evidence keeps its original source and environment scope in the archived [production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md).

A future public release needs the owner's explicit release instruction, a fresh version tag, a clean reviewed full commit, passing source and Linux gates, a bundle, native Windows and Linux binaries, provenance, the exact workflow/run/attempt identity and matching bundle, provenance and asset digests. The retained release workflow builds and checks new tagged source; it grants no authority to publish. The earlier GitHub-release and WinGet publisher was removed from Main and stays in Git history: a release restores it or writes a new one, inspects its read-only publication plan before any write, and afterwards verifies the immutable release, its attestations and every attached asset. Never overwrite a tag or asset, and never reuse a historical tag or a retired version. Required historical release evidence and the named archive refs are kept. Release pages and downloads are retired only on the owner's explicit instruction; retired releases keep their recorded source and asset identities and attestations without implying availability. Add download instructions to the README only after a release exists and is verified.

CI transport is bounded: Go test artifacts remain for 14 days to support the flake report; commit-scoped source archives remain for 7 days. Named evidence, failed diagnostics and artifacts needed by current work are excluded from manual historical cleanup. Durable summaries, public attestations and source references belong in Git; temporary artifact URLs are not a permanent evidence store. [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns source gates.

## Historical content-package command

The old product, bootstrap formats, download walkthroughs and `markitect package` command remain at [the pre-retirement guide](https://github.com/Glacius-Labs/Markitect/blob/1f4a1ede214d05fd2fee615b3905207df7702426/integration/README.md), pinned by the archive ref `archive/pre-reset-1f4a1ede`. They are historical contracts, not another active command surface.
