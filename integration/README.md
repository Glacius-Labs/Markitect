# Consumer integration

Markitect's distribution unit is one versioned, immutable GitHub Release. The tagged workflow builds a deterministic source bundle from the exact commit, tests installation on Windows and Linux amd64, and uploads a run-scoped artifact. An authorized release owner publishes that exact artifact through the local release tool. The source declares version `0.1.0`; that string alone does not prove a release was published or accepted. Install only an exact immutable release whose metadata and successful run identify the intended source commit.

## Release contents and verification

The Release has four attached assets: `markitect-vTAG-bundle.zip`, `markitect-vTAG-windows-amd64.exe`, `markitect-vTAG-linux-amd64`, and `markitect-vTAG-provenance.yaml`. The bundle is a ZIP limited to 64 MiB. It contains five pinned files: the `release.yaml` manifest, `markitect.lock.yaml`, the source archive at `tools/markitect/source.zip`, and the bootstrap and its test under `scripts/`. The manifest binds the four other files by SHA-256 and records the version, source commit, and source repository. The provenance YAML records the tag, commit, workflow run, toolchain, and digest of the bundle and both binaries. It is a self-declared digest and run record; GitHub's immutable-release attestation is separate and authoritative for the published tag, commit, and attached assets.

Use the native binary for your platform. Authenticate to the private repository with GitHub CLI, verify the exact tag's immutable release, download the bundle, provenance, and binary, and verify each downloaded file against that release. Never select `latest` or use GitHub's automatically generated source archive in place of the attached bundle. GitHub documents [immutable release protections](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases) and [release verification](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity); [GitHub CLI release commands](https://cli.github.com/manual/gh_release) provide the corresponding checks.

PowerShell example (replace the candidate tag only after its release has been published):

```powershell
gh auth status
$repository = 'Glacius-Labs/Markitect'
$tag = 'v0.1.0'
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
gh release verify-asset $tag $provenance --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'Provenance asset did not verify against the release.' }
gh release verify-asset $tag $bundle --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'Bundle asset did not verify against the release.' }
gh release verify-asset $tag $binary --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'Windows binary did not verify against the release.' }

# Read the verified provenance. Confirm its tag and source commit, inspect its
# workflowRun in GitHub, and copy the bundle SHA-256 from its bundle asset entry.
Get-Content -LiteralPath $provenance
$expectedBundleSha256 = '<64 lowercase hex characters from verified provenance>'
if ($expectedBundleSha256 -notmatch '^[0-9a-f]{64}$') { throw 'Invalid provenance bundle digest.' }
$actualBundleSha256 = (Get-FileHash -LiteralPath $bundle -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualBundleSha256 -ne $expectedBundleSha256) { throw 'Bundle digest differs from verified provenance.' }
```

`gh release verify` checks GitHub's release attestation, which binds the release tag, commit, and assets. `gh release verify-asset` checks that each local file exactly matches an asset for that release. The provenance asset itself must be verified before trusting the bundle digest it lists. Check that its tag and source commit agree with the selected release, and that the referenced release workflow run completed successfully for that commit. The signed release checks do not replace review of the source or consumer changes.

## Preview and install

`install` verifies the outer bundle SHA-256, its exact file set, the release manifest, the lock, and all four manifest-bound distribution files. With no `--write`, it only prints a plan. The plan lists all five target paths, the operation and expected digest for each, the bundle version, and source commit. Review this plan before applying it.

Create or select a named feature branch in the consumer and commit a baseline first. The write requires a committed Git `HEAD`, refuses `main` and `master`, checks for staged changes to pin paths, and allows a full existing release pin set to be replaced only when it still matches committed `HEAD`. The private GitHub repository currently cannot have server-side branch rules configured under its plan (the API responds 403); Markitect's branch-name check is not a substitute for GitHub-enforced protection or normal review.

```powershell
$consumer = (Resolve-Path '<consumer-repository-root>').Path
& $binary install --repo $consumer --bundle $bundle --sha256 $expectedBundleSha256
if ($LASTEXITCODE -ne 0) { throw 'Installation preview failed.' }
# Inspect the YAML plan: version/sourceCommit must match the verified release,
# and all five target paths and actions must be expected.

& $binary install --repo $consumer --bundle $bundle --sha256 $expectedBundleSha256 --write
if ($LASTEXITCODE -ne 0) { throw 'Installation did not complete; inspect the returned plan and recovery guidance.' }
# This check applies when $consumer already has a configured markitect.yaml.
& $binary check --repo $consumer
if ($LASTEXITCODE -ne 0) { throw 'Installed Markitect check failed.' }

Push-Location $consumer
go test scripts/run-markitect.go scripts/markitect-bootstrap_test.go
if ($LASTEXITCODE -ne 0) { throw 'Installed bootstrap tests failed.' }
go run scripts/run-markitect.go version
if ($LASTEXITCODE -ne 0) { throw 'Installed source bootstrap failed.' }
Pop-Location
```

The five files are written atomically one at a time; this is not a filesystem transaction. If a later write fails, the command returns the paths already written and recovery guidance. Do not use a partial set. A partial set is refused on the next install, so reapplying is not an automatic resume. For a fresh install, inspect and remove the listed newly created files, then rerun preview and apply. For an upgrade, restore the prior complete committed pin set or revert the integration commit through the consumer's normal Git route. Once the installation is committed, rollback is a new Git revert commit that restores all five pins together; do not move or recreate an immutable release tag.

`install` pins the CLI distribution; it does not create a `markitect.yaml` Project. For an already configured Markitect project, run `check` after installation as shown above. To adopt an existing Konfyra or Cockpit repository, first install the tool, then preview migration explicitly with `migrate --repo $consumer --profile konfyra` or `migrate --repo $consumer --profile cockpit`; inspect the planned file list and, for Konfyra, the candidate dependency list. After review, add `--write` on the feature branch and inspect the resulting diff. Migration does not infer normative dependencies from prose. Run the relevant provider renderer and checks after reviewing the migration. A generic new Project must be configured separately; the minimal example is a starting point, not an install initializer.

Text files in the bundle are validated against their canonical LF form; the installer accepts CRLF readback from a Windows checkout. The nested `source.zip` is copied and checked byte-for-byte. Installation does not rewrite or normalize ZIP contents.

The installer can upgrade a complete, committed `0.1.0-rc.3` four-file pin set. It refuses partial, changed, staged, or unknown pin sets rather than guessing ownership. Other legacy installations require an explicit migration.

## Runtime requirements

Git is required for immutable snapshots and for `install --write`. The downloaded native release binary can run the install preview and write without Go installed. The installed consumer bootstrap is Go source: running it requires Go 1.27.1 or newer. The first bootstrap build may download the exact Go toolchain and checksum-verified module dependency; for offline use, provision those exact toolchain and module inputs before disconnecting. A cache hit still probes the selected toolchain. The outer `go run` also needs writable `GOCACHE` and `GOTMPDIR`. The bootstrap uses only the Go standard library; `CGO` is disabled for its Markitect build. GitHub CLI and private-repository read access are needed only to obtain and verify release assets.

The platform release targets are Windows amd64 and Linux amd64. The tagged workflow runs source quality checks on both, then installs and smoke-tests the same deterministic bundle on both before uploading its run artifact. The authorized owner tool handles later publication. Other OS/architecture targets are not claimed.

## Owner publication

The tagged Release workflow runs source quality gates and bundle/bootstrap smoke tests on Windows and Linux amd64, then uploads one run artifact named `markitect-release-SOURCE_SHA` with 30-day retention. The artifact contains four root-level files with versioned names: bundle ZIP, Windows and Linux amd64 binaries, and provenance YAML. The provenance records the tag, source commit, workflow run ID/attempt/URL, toolchain and SHA-256 digests for the bundle and binaries. GitHub Actions only builds and tests; it does not create a draft or mutate a Release.

The Actions `GITHUB_TOKEN` cannot read the admin-only immutable-release setting (the live API returned 403). The release job therefore stops at its run artifact; no PAT is exported to or stored in workflow secrets. An authorized owner publishes locally using their existing GitHub CLI authentication. Check that `gh auth status` shows the verified owner/admin account; use Go 1.27.1 or newer and the exact tagged source checkout.

Download the artifact from the completed Release workflow run into a new directory. The run has one artifact, so this downloads that run's artifact files without requiring manual hash entry:

```powershell
$repository = 'github.com/Glacius-Labs/Markitect'
gh auth status
if ($LASTEXITCODE -ne 0) { throw 'GitHub CLI authentication is unavailable.' }
$runId = '<successful Release workflow run ID>'
$run = gh run view $runId --repo $repository --json headSha,status,conclusion | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect the selected Release workflow run.' }
if ($run.status -ne 'completed' -or $run.conclusion -ne 'success' -or $run.headSha -notmatch '^[0-9a-f]{40}$') {
    throw 'The selected Release workflow run is not successful or has no full source commit.'
}
$sourceCommit = $run.headSha
$assets = Join-Path $env:TEMP "markitect-release-$runId"
if (Test-Path -LiteralPath $assets) { throw "Asset directory already exists: $assets" }
New-Item -ItemType Directory -Path $assets | Out-Null
gh run download $runId --repo $repository --name "markitect-release-$sourceCommit" --dir $assets
if ($LASTEXITCODE -ne 0) { throw 'Release artifact download failed.' }

# Run from the exact tagged Markitect source checkout. This prints a read-only
# plan; inspect its tag, run, source commit, asset names and verified actions.
Set-Location -LiteralPath '<exact tagged Markitect source checkout>'
go run ./cmd/markitect-release --tag v0.1.0 --run $runId --assets $assets
if ($LASTEXITCODE -ne 0) { throw 'Release preflight failed.' }

# This is the explicit owner publication step. Use it only after reviewing
# the complete plan printed by the command above.
go run ./cmd/markitect-release --tag v0.1.0 --run $runId --assets $assets --publish
if ($LASTEXITCODE -ne 0) { throw 'Release publication or final verification failed; inspect the release state.' }
```

Both modes authenticate through the existing `gh auth` session. The tool independently checks the live immutable-release setting, exact remote tag target, selected successful Release workflow run, provenance and four-file set; it fetches the same run artifact again with that authentication and byte-compares it with `--assets`. The default mode prints a full read-only plan. `--publish` repeats the preflight immediately before mutation, creates an empty draft, uploads the four files, checks the names and GitHub SHA-256 values, rechecks tag/settings, publishes, then verifies the immutable-release attestation and every asset. The owner does not copy or type hashes. A successful CI run alone is not a published or accepted Release; completion requires successful owner publication and verification for the exact tag/source commit.

The release tool fails closed if any release or draft already exists for the tag; CI itself creates no draft. If the publish command exits with an error, inspect the actual release state and verify the tag and assets; do not assume it remains a draft or retry blindly. The tool does not resume, delete or clobber existing releases. Compare any draft's resolved tag commit, exact asset names, GitHub API digests and provenance against the completed run. If anything is incomplete or mismatched, stop and have the authorized owner choose a reviewed correction/discard path before retrying. A published immutable Release is never replaced or retried.

## Legacy package command

`markitect package --repo . --output NEW_DIRECTORY` remains a low-level development command that writes only `tools/markitect/source.zip` and the flat lock. It does not produce the release bundle, bootstrap files, `release.yaml`, binaries, or provenance; use the tagged Release workflow for consumer installation. No content-package format is part of this tool lock.
