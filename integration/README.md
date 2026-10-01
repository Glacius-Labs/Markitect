# Distribution and integration

Markitect's distribution unit is a versioned, immutable GitHub Release. Source files and release assets are never retagged or overwritten. Check [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available versions and verify the exact tag, source commit, and assets before use. The current source model and upgrade notes are in [Usage](../docs/usage.md); the [roadmap](../docs/implementation-plan.md) records current source scope.

## Release contents and verification

A Markitect release attaches four files: `markitect-vTAG-bundle.zip`, `markitect-vTAG-windows-amd64.exe`, `markitect-vTAG-linux-amd64`, and `markitect-vTAG-provenance.yaml`. The bundle is limited to 64 MiB and contains five pinned files: `tools/markitect/release.yaml`, `markitect.lock.yaml`, `tools/markitect/source.zip`, and the Go bootstrap plus its paired test under `scripts/`. The manifest binds the other files by SHA-256 and records version, source commit, and source repository. Provenance records tag, commit, workflow run, toolchain, and payload digests. It is a self-declared record; GitHub's immutable-release attestation is authoritative for the published tag, commit, and attached assets.

The public native binaries can be downloaded without GitHub CLI or an account; the [README](../README.md#install-markitect) gives pinned-version, SHA-256-checked commands for installing the CLI. For stronger release provenance verification or for project pinning, use GitHub CLI to verify the exact immutable release, download the bundle, provenance, and matching binary, then verify each local file against the release. GitHub CLI may require authentication. Do not choose `latest` or substitute GitHub's generated source archive for the attached bundle. GitHub documents [immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases), [release verification](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity), and [GitHub CLI release commands](https://cli.github.com/manual/gh_release).

PowerShell example (select only a tag confirmed to exist):

```powershell
gh auth status
$repository = 'github.com/Glacius-Labs/Markitect'
$tag = 'v0.6.0'
gh release verify $tag --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'GitHub release attestation verification failed.' }
$release = gh release view $tag --repo $repository --json tagName,isDraft,isImmutable | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw 'Could not read the selected GitHub release.' }
if ($release.tagName -ne $tag -or $release.isDraft -or -not $release.isImmutable) {
    throw 'The selected release is not the exact immutable release.'
}

$download = Join-Path $env:TEMP "markitect-$tag"
if (Test-Path -LiteralPath $download) { throw "Download directory already exists: $download" }
New-Item -ItemType Directory -Path $download | Out-Null
gh release download $tag --repo $repository `
    --pattern "markitect-$tag-bundle.zip" `
    --pattern "markitect-$tag-windows-amd64.exe" `
    --pattern "markitect-$tag-provenance.yaml" `
    --dir $download
if ($LASTEXITCODE -ne 0) { throw 'Release asset download failed.' }

$bundle = Join-Path $download "markitect-$tag-bundle.zip"
$binary = Join-Path $download "markitect-$tag-windows-amd64.exe"
$provenance = Join-Path $download "markitect-$tag-provenance.yaml"
foreach ($asset in @($provenance, $bundle, $binary)) {
    gh release verify-asset $tag $asset --repo $repository
    if ($LASTEXITCODE -ne 0) { throw "Release asset verification failed: $asset" }
}
Get-Content -LiteralPath $provenance
$expectedBundleSha256 = '<64 lowercase hex characters from verified provenance>'
if ($expectedBundleSha256 -notmatch '^[0-9a-f]{64}$') { throw 'Invalid provenance bundle digest.' }
$actualBundleSha256 = (Get-FileHash -LiteralPath $bundle -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualBundleSha256 -ne $expectedBundleSha256) { throw 'Bundle digest differs from verified provenance.' }
```

Verify the provenance asset before trusting the digest it lists. Check that its tag and source commit agree with the selected release and that its referenced workflow run succeeded for that commit. The signed checks do not replace review of source suitability or project changes.

## Preview and install

`install` verifies the outer bundle SHA-256, exact file set, manifest, lock, and manifest-bound distribution files. Without `--write`, it prints a plan listing all five target paths, operations and digests, bundle version, and source commit. Review the plan before applying it.

Create or select a named feature branch in the project repository and commit a baseline. Installation requires a committed `HEAD`, refuses `main` and `master`, checks staged changes to pin paths, and replaces an existing complete pin only when it still matches the committed tree.

```powershell
$project = (Resolve-Path '<project-repository-root>').Path
& $binary install --repo $project --bundle $bundle --sha256 $expectedBundleSha256
if ($LASTEXITCODE -ne 0) { throw 'Installation preview failed.' }
# Confirm version/sourceCommit and each target path and operation.

& $binary install --repo $project --bundle $bundle --sha256 $expectedBundleSha256 --write
if ($LASTEXITCODE -ne 0) { throw 'Installation did not complete; inspect written paths and recovery guidance.' }
```

The native binary can preview and apply an install without Go. For the Go bootstrap route, extract and inspect the complete bundle, then run it from the project root:

```powershell
Push-Location $project
go test scripts/run-markitect.go scripts/markitect-bootstrap_test.go
if ($LASTEXITCODE -ne 0) { throw 'Bootstrap tests failed.' }
go run scripts/run-markitect.go version
if ($LASTEXITCODE -ne 0) { throw 'Pinned Markitect bootstrap failed.' }
Pop-Location
```

`scripts/run-markitect.go` is intentionally one self-contained Go file: the
installed command invokes that file directly, and CI tests the copied file
with its companion test. The bootstrap validates the lock and source archive
before building the pinned CLI, so its version and path validation repeats
some release-side rules without importing the code it has not yet built.
When changing those rules, check both the release package and bootstrap tests.

The installer writes the complete pin as five individual atomic file replacements; it is not a filesystem transaction. If a later write fails, inspect the returned `written` paths. Do not treat a partial set as resumable. For a fresh install, remove only the listed files known to have been created by that attempt before retrying. For an upgrade, restore the previous complete committed set or use a normal Git revert. Keep a baseline commit so the whole pin can be rolled back together. Do not move or recreate an immutable release tag.

A pin installs the CLI distribution; it does not create project content or choose project policy. Project configuration must match the installed release. The current source model documents explicit checks, areas, imports, renderer targets, and bounded initialization in [Usage](../docs/usage.md) and the [Project schema](../schema/Project.yaml). Check the selected release's version and available commands before relying on source-only functionality. Existing content import scripts belong beside the content they transform. Markitect does not ship an implicit migration adapter.

Text files are validated against canonical LF content; Windows CRLF checkout readback is accepted. The nested source archive is copied and verified byte-for-byte. The installer does not rewrite ZIP contents.

## Runtime requirements

Git is required for immutable snapshots and `install --write`. The native release binary can run install preview/write without Go. The bootstrap is Go source and requires Go 1.27.1 or newer; its first build may download the pinned toolchain and checksum-verified module dependencies. For offline use, provision those inputs before disconnecting. A cache hit still probes the selected toolchain. The outer `go run` needs writable `GOCACHE` and `GOTMPDIR`. The bootstrap uses only the Go standard library and builds Markitect with CGO disabled. Public release assets need no account to download; GitHub CLI is optional for signed release-attestation checks and required by the documented owner-publication workflow.

The verified platform targets are Windows amd64 and Linux amd64. The tagged workflow runs source gates and bundle-install/bootstrap smoke tests on both platforms. Other OS/architecture targets are not claimed.

## Owner publication

Release CI builds and tests but does not create or publish a GitHub Release. It uploads one 30-day run artifact named `markitect-release-SOURCE_SHA` containing four root-level versioned files: bundle ZIP, Windows and Linux amd64 binaries, and provenance YAML. Provenance records tag, source commit, workflow run ID/attempt/URL, toolchain, and hashes for the bundle and binaries.

The Actions `GITHUB_TOKEN` cannot read the admin-only immutable-release setting; the live API returned 403. Therefore an authorized release owner uses their existing authenticated GitHub CLI session locally. No PAT is stored in workflow secrets. Use Go 1.27.1 or later from the exact tagged source checkout.

Download the artifact from its successful release workflow run into a new directory:

```powershell
$repository = 'github.com/Glacius-Labs/Markitect'
gh auth status
if ($LASTEXITCODE -ne 0) { throw 'GitHub CLI authentication is unavailable.' }
$runId = '<successful Release workflow run ID>'
$run = gh run view $runId --repo $repository --json headSha,status,conclusion | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect the selected Release workflow run.' }
if ($run.status -ne 'completed' -or $run.conclusion -ne 'success' -or $run.headSha -notmatch '^[0-9a-f]{40}$') {
    throw 'The selected workflow run is not successful or has no full source commit.'
}
$sourceCommit = $run.headSha
$assets = Join-Path $env:TEMP "markitect-release-$runId"
if (Test-Path -LiteralPath $assets) { throw "Asset directory already exists: $assets" }
New-Item -ItemType Directory -Path $assets | Out-Null
gh run download $runId --repo $repository --name "markitect-release-$sourceCommit" --dir $assets
if ($LASTEXITCODE -ne 0) { throw 'Release artifact download failed.' }
```

From the exact tagged source checkout, first print and inspect the read-only plan. Then explicitly select publication:

```powershell
Set-Location -LiteralPath '<exact tagged Markitect source checkout>'
$tag = 'vX.Y.Z' # Replace with the exact version tag for this release.
go run ./cmd/markitect-release --tag $tag --run $runId --assets $assets
if ($LASTEXITCODE -ne 0) { throw 'Release preflight failed.' }
# Review the exact tag, source commit, run and four assets in the plan.
go run ./cmd/markitect-release --tag $tag --run $runId --assets $assets --publish
if ($LASTEXITCODE -ne 0) { throw 'Publication or final verification failed; inspect the release state.' }
```

Both modes use the existing `gh auth` session. The tool verifies the immutable setting, remote tag target, successful run, provenance, and exact asset set; it downloads that same run artifact again and compares bytes with `--assets`. `--publish` repeats the checks before mutation, creates a draft, uploads and reads back the four assets, checks digests, publishes, then verifies the immutable release and each asset. A successful CI run alone is not a published release.

After publication, each read-only attestation query allows four attempts with delays of one, two, and four seconds. Successful retries are recorded in the publisher result. Release identity and asset digest mismatches still fail immediately. If attestation remains unavailable, the publisher reports `published-unverified`; inspect and verify the immutable release manually rather than rerunning publication.

The publisher fails closed if a release or draft already exists for the tag. If publication fails, inspect the release state and compare tag target, exact asset names, GitHub digests, and provenance against the workflow run. Do not retry blindly or overwrite an existing asset. An authorized owner must choose a reviewed recovery path. A published immutable release is never replaced.

## Low-level source package

`markitect package --repo . --output NEW_DIRECTORY` emits only `tools/markitect/source.zip` and the flat lock. It does not create the release bundle, bootstrap, manifest, binaries, or provenance. It is a development artifact, not the supported versioned distribution.
