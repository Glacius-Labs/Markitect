# Packaging and distribution

Releases start with `0.15.0` ([DEC-095](../docs/decisions.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)); see [release operations](#release-operations). The 18 versions `v0.1.0` to `v0.14.1` belong to retired releases of an earlier product and must not be reused; their tags no longer exist, and the commit behind each is in the [retired release mapping](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/retired-releases-20261017.json).

## What lives here

`packaging/` holds what ships besides the binary: `bootstrap/` (the consumer bootstrap `run-markitect.go` that release bundles copy into `.markitect/bootstrap/run.go`, with its test) and this guide. The standard Libraries and the agent skills are built into the binary ([`internal/libraries`](../internal/libraries/), [`internal/skills`](../internal/skills/)), and `markitect init` installs the skills.

## Current source release tooling

Independent `markitect-release` commands prepare local distributions. They do not publish, download a release or run an agent.

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

Use Git and the Go version in `go.mod`.

## Release operations

A release is a tag `vX.Y.Z` on a commit of `main`. The [release workflow](../.github/workflows/release.yaml) checks the tag, the source version and the plugin version, runs the quality gates and the bundle smoke, builds `markitect-vX.Y.Z-OS-ARCH` (with `.exe` on Windows) for Linux, macOS and Windows on amd64 and arm64, writes `markitect-vX.Y.Z-checksums.txt` in the format of `sha256sum` and a provenance file, publishes all of it with the source bundle as a GitHub release, and then starts markitect through the plugin's launcher and asks `markitect update --check` for the newest release, against the published files. A manual run of the workflow is a dry run: it runs the gates and builds and checks every asset, but skips what belongs to a tag (the checks of the tag and of main's ancestry, the publish and the check of the published release) and publishes no release. The source version and the plugin version are compared in a dry run too. The names are a contract with `markitect update` (`internal/adapters/upgrade`) and the plugin's launcher (`plugin/bin/markitect`).

To cut a release:

1. In one change, set `version` in `internal/buildinfo/version.go` and `version` in `plugin/.claude-plugin/plugin.json` to `X.Y.Z`. The source and the plugin always say the same: a test refuses a difference, and the workflow checks it again. Never reuse a retired version, `0.1.0` to `0.14.1`. Review it, run the checks of [CONTRIBUTING](../CONTRIBUTING.md#check-a-change) and merge it.
2. Tag the merge commit `vX.Y.Z` and push the tag. The workflow does the rest; it publishes only when every check passed. Wait until its last job, the check of the published release, is green: a release that fails it is marked as a pre-release.
3. Point the catalog at it. The marketplace `glacius-labs` is the repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins), and its entry for `markitect` pins the plugin to a tag and to the commit of that tag. In a clone of it run `python3 scripts/pin.py markitect vX.Y.Z` and `python3 scripts/check_marketplace.py`, commit and push; the check also runs there in CI and refuses a pin that does not match a published release with a plugin of that version. Until then, users keep the previous release.
4. In a following change, set both versions to the next development version (`X.Y+1.0-dev`). The launcher of a development plugin refuses to fetch anything, and nobody installs one, since the catalog pins tags.

The binary is not signed. `markitect update` and the launcher check the download against the checksums of the same release, which guards against a damaged or swapped file, and the connection is HTTPS to GitHub. [Artifact attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifacts/using-artifact-attestations-to-establish-provenance-for-builds) would add that a binary was built by this workflow. Local packaging with `markitect-release` does not publish.

A public release needs a fresh version tag, a clean reviewed full commit on `main`, passing source and Linux gates, a bundle, the binaries, provenance, the exact workflow, run and attempt identity, and matching bundle, provenance and asset digests. The release workflow checks all of that on the tagged commit before it publishes, and the publish job rechecks the assets and the remote tag. The workflow publishes with `gh release create`. A release that fails the check after it is public is marked as a pre-release, so `markitect update` does not offer it. Never overwrite a tag or asset, and never reuse a historical tag or a retired version. Release pages and downloads are retired only on the product owner's explicit instruction.

CI transport is bounded: Go test artifacts remain for 14 days to support the flake report; commit-scoped source archives remain for 7 days. Named evidence, failed diagnostics and artifacts needed by current work are excluded from manual historical cleanup. Durable summaries, public attestations and source references belong in Git; temporary artifact URLs are not a permanent evidence store. [CONTRIBUTING](../CONTRIBUTING.md#check-a-change) owns source gates.
