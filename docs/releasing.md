# Releasing

How a release is made, what it publishes and how to cut one ([DEC-095](decisions.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)). Releases start with `0.15.0`. The 18 versions `v0.1.0` to `v0.14.1` belong to retired releases of an earlier product and must not be reused; their tags no longer exist, and the commit behind each is in the [retired release mapping](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/retired-releases-20261017.json).

## What a tag publishes

A release is a tag `vX.Y.Z` on a commit of `main`. The [release workflow](../.github/workflows/release.yaml) does the following, and publishes only when every step before it passed:

1. It checks the tag, the source version (`markitect version`) and the plugin version, and runs the quality gates of [CI](../CONTRIBUTING.md#check-a-change).
2. It builds `markitect-vX.Y.Z-OS-ARCH` for Linux, macOS and Windows on amd64 and arm64 (with `.exe` on Windows), writes `markitect-vX.Y.Z-checksums.txt` in the format of `sha256sum` and a provenance file, `markitect-vX.Y.Z-provenance.yaml`, and checks that the Linux binary reports the release.
3. It publishes exactly those eight files as a GitHub release, after it has checked the files and the remote tag again. GitHub attaches the source of the tag. A version with a pre-release part, such as `v0.16.0-rc.1`, is published as a pre-release.
4. It starts `markitect` through the plugin's launcher, which downloads the published binary, and asks `markitect update --check` for the newest release. A release that fails this final check is marked as a pre-release, so `markitect update` does not offer it.

The file names are a contract with `markitect update` ([`internal/adapters/upgrade`](../internal/adapters/upgrade/)) and the plugin's launcher ([`plugin/bin/markitect`](../plugin/bin/markitect)).

A manual run of the workflow is a dry run. It runs the gates, builds and checks every file, and compares the source version with the plugin version, but skips what belongs to a tag: the checks of the tag and of the ancestry of `main`, the publication and the check of the published release.

## Cut a release

1. In one change, set `version` in [`internal/buildinfo/version.go`](../internal/buildinfo/version.go) and `version` in [`plugin/.claude-plugin/plugin.json`](../plugin/.claude-plugin/plugin.json) to `X.Y.Z`. The source and the plugin always say the same: a test refuses a difference, and the workflow checks it again. Never reuse a retired version, `0.1.0` to `0.14.1`. Review it, run the checks of [CONTRIBUTING](../CONTRIBUTING.md#check-a-change) and merge it.
2. Tag the merge commit `vX.Y.Z` and push the tag. The workflow does the rest. Wait until its last job, the check of the published release, is green.
3. Point the catalog at it. The marketplace `glacius-labs` is the repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins), and its entry for `markitect` pins the plugin to a tag and to the commit of that tag. In a clone of it run `python3 scripts/pin.py markitect vX.Y.Z` and `python3 scripts/check_marketplace.py`, commit and push; the check also runs there in CI and refuses a pin that does not match a published release with a plugin of that version. Until then, users keep the previous release.
4. In a following change, set both versions to the next development version (`X.Y+1.0-dev`). The launcher of a development plugin refuses to fetch anything, and nobody installs one, since the catalog pins tags.

## Rules

- Never overwrite a tag or a file of a release, and never reuse a historical tag or a retired version. Release pages and downloads are retired only on the product owner's explicit instruction.
- The binary is not signed. `markitect update` and the launcher check the download against the checksums of the same release, which guards against a damaged or swapped file, and the connection is HTTPS to GitHub. [Artifact attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations-to-establish-provenance-for-builds) would add that a binary was built by this workflow.
- A public release needs a fresh version tag, a clean reviewed commit on `main`, passing gates, the six binaries, the checksums, the provenance, and the identity of the workflow, its run and its attempt. The workflow checks this on the tagged commit before it publishes.
- CI keeps its artifacts for a limited time: Go test results for 14 days, to support the flake report, and the files of a release run for 30 days. Durable summaries and source references belong in Git; a temporary artifact URL is not a permanent record.
