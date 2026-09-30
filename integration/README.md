# Consumer integration

Markitect owns a Go-only bootstrap distributed as `run-markitect.go`. A consumer copies it to `scripts/run-markitect.go` and its test to `scripts/markitect-bootstrap_test.go`. Its existing repository checks may remain in their current language and call this Go entrypoint.

The tool release is a verified source archive. `markitect package --repo . --output NEW_DIRECTORY` creates `tools/markitect/source.zip` and a flat `markitect.lock.yaml` with exact `version`, `source` and SHA-256. Copy the archive and lock together into the consumer. The bootstrap is a separate integration source file.

Run from the consumer repository root:

```powershell
go run scripts/run-markitect.go version
go run scripts/run-markitect.go check
go test scripts/run-markitect.go scripts/markitect-bootstrap_test.go
```

The bootstrap finds the lock from the current directory or its parents, verifies source/archive/cache integrity and builds the pinned Go module. It requires Go; its own code uses only the standard library. The first tool build can acquire the toolchain and checksum-verified Go dependency. Subsequent runs can use verified local inputs and caches. In a restricted shell, configure a writable `GOCACHE` and `GOTMPDIR` before `go run` because the outer Go invocation runs before the bootstrap.

Cache eligibility also includes the actual selected Go toolchain and the bootstrap's build-policy version. The bootstrap probes the toolchain in the verified extracted module, then pins that version for compilation. It builds a portable native executable with CGO disabled; caller `GOFLAGS`, workspace settings, experiments and architecture tuning do not change the binary contract. Module download/checksum configuration and cache locations remain available. Old stamps or a different toolchain cause a rebuild. Source/archive/executable hashes continue to be checked.

Build the bootstrap as a small executable when exact CLI exit codes or repeated invocation speed matter. `go run` itself can map a program's nonzero status to its own exit status; automation should use the built bootstrap/binary when distinguishing Markitect's 1 from 2.

The current lock is a tool-distribution lock. Content-package imports require the future versioned format described in [the design](../docs/refinement.md); do not add unknown fields to this lock.

## Provisioning from GitHub Actions

The private [`Glacius-Labs/Markitect`](https://github.com/Glacius-Labs/Markitect) repository runs CI for pull requests, pushes to `main`, and manual dispatches. The `ubuntu-24.04` and `windows-latest` gates verify the Go module, tests, vet, build, schemas, canonical formatting and minimal example. They also package the source and run the standalone bootstrap against that package, including the compiled authoring Skill. A source artifact is uploaded only after every gate succeeds for a push to `main` or a manual run dispatched on `main`.

The artifact is named `markitect-source-<full-commit-sha>` and contains this exact set:

```text
markitect.lock.yaml
tools/markitect/source.zip
scripts/run-markitect.go
scripts/markitect-bootstrap_test.go
```

It is a short-lived workflow artifact, not a GitHub Release, tag, or automatic consumer update. Artifacts are retained for 30 days. Access to this private repository requires an authenticated GitHub CLI session with repository read access. Select a completed successful `main` run and supply both its run ID and expected 40-character commit SHA; do not download an artifact by “latest”. For example, in PowerShell:

```powershell
gh auth status
$repository = 'Glacius-Labs/Markitect'
gh run list --repo $repository --workflow ci.yaml --branch main --limit 10

$runId = '<completed-run-id>'
$expectedSha = '<40-character-commit-sha>'
if ($expectedSha -notmatch '^[0-9a-f]{40}$') { throw 'Expected a full lowercase commit SHA.' }
$run = gh run view $runId --repo $repository --json headSha,headBranch,event,status,conclusion | ConvertFrom-Json
$allowedEvent = $run.event -eq 'push' -or $run.event -eq 'workflow_dispatch'
if ($run.headSha -ne $expectedSha -or $run.headBranch -ne 'main' -or $run.status -ne 'completed' -or $run.conclusion -ne 'success' -or -not $allowedEvent) {
    throw 'The run is not a successful main-branch source-artifact run for the expected commit.'
}

$artifactName = "markitect-source-$expectedSha"
$download = Join-Path $env:TEMP $artifactName
if (Test-Path -LiteralPath $download) { throw "Download directory already exists: $download" }
New-Item -ItemType Directory -Path $download | Out-Null
gh run download $runId --repo $repository --name $artifactName --dir $download
if ($LASTEXITCODE -ne 0) { throw 'Artifact download failed.' }

Push-Location $download
go test scripts/run-markitect.go scripts/markitect-bootstrap_test.go
if ($LASTEXITCODE -ne 0) { throw 'Standalone bootstrap tests failed.' }
$context = go run scripts/run-markitect.go authoring
if ($LASTEXITCODE -ne 0 -or ($context -join "`n") -notmatch 'core/Skill/authoring') { throw 'The downloaded lock/archive/bootstrap set did not pass its authoring smoke check.' }
Pop-Location
```

The bootstrap verifies that the source archive bytes match the downloaded lock, then builds the pinned module. Only after these checks, review and copy the four files together into the explicitly chosen consumer paths. For example, from the consumer checkout in PowerShell:

```powershell
$consumerRoot = (Resolve-Path '<consumer-repository-root>').Path
New-Item -ItemType Directory -Force -Path (Join-Path $consumerRoot 'tools/markitect'), (Join-Path $consumerRoot 'scripts') | Out-Null
Copy-Item (Join-Path $download 'markitect.lock.yaml') (Join-Path $consumerRoot 'markitect.lock.yaml')
Copy-Item (Join-Path $download 'tools/markitect/source.zip') (Join-Path $consumerRoot 'tools/markitect/source.zip')
Copy-Item (Join-Path $download 'scripts/run-markitect.go') (Join-Path $consumerRoot 'scripts/run-markitect.go')
Copy-Item (Join-Path $download 'scripts/markitect-bootstrap_test.go') (Join-Path $consumerRoot 'scripts/markitect-bootstrap_test.go')
```

Commit that scoped change through the consumer's normal review and acceptance process. Do not configure consumer CI to fetch a floating version, rewrite a consumer checkout, or upgrade consumers automatically; Markitect CI only publishes its own source artifact.

Keep the previous pin commit for rollback. Restore the complete four-file set together and reverse any related content migration before checking the rollback candidate. [Operations and releases](../docs/operations.md) describes failure recovery, upgrade evidence and the supported release boundary.
