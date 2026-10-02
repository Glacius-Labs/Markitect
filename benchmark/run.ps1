param(
    [Parameter(Mandatory)][string]$CurrentBinary,
    [Parameter(Mandatory)][string]$PreviousBinary,
    [Parameter(Mandatory)][string]$CurrentTag,
    [Parameter(Mandatory)][string]$PreviousTag,
    [Parameter(Mandatory)][string]$CurrentFixtureRoot,
    [Parameter(Mandatory)][string]$PreviousFixtureRoot,
    [Parameter(Mandatory)][string]$OutputDirectory
)
$ErrorActionPreference = 'Stop'

function Get-FixtureVersion([string]$root) {
    $version = Split-Path -Leaf ([IO.Path]::GetFullPath($root).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar))
    if ($version -notmatch '^v[1-9][0-9]*$') { throw "Fixture root '$root' must end in an immutable version directory such as v1 or v2" }
    return $version
}

function Get-FixtureDigest([string]$root) {
    $fullRoot = [IO.Path]::GetFullPath($root)
    $rows = foreach ($file in (Get-ChildItem -LiteralPath $fullRoot -Force -File -Recurse | Sort-Object { $_.FullName.Substring($fullRoot.Length).Replace('\', '/') })) {
        $relative = $file.FullName.Substring($fullRoot.Length).TrimStart([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar).Replace('\', '/')
        $digest = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        "$relative`0$digest"
    }
    $text = [string]::Join("`n", [string[]]$rows)
    $hash = [Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes($text))
    return [Convert]::ToHexString($hash).ToLowerInvariant()
}

function Copy-FixtureTree([string]$source, [string]$destination) {
    New-Item -ItemType Directory -Force -Path $destination | Out-Null
    foreach ($entry in (Get-ChildItem -LiteralPath $source -Force)) {
        if ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Fixture contains an unsupported reparse point: $($entry.FullName)" }
        $target = Join-Path $destination $entry.Name
        if ($entry.PSIsContainer) {
            Copy-FixtureTree $entry.FullName $target
        } else {
            Copy-Item -LiteralPath $entry.FullName -Destination $target -Force
        }
    }
}

function Find-RollbackText([string]$root) {
    $foundResources = [Collections.Generic.List[string]]::new()
    foreach ($file in (Get-ChildItem -LiteralPath $root -Force -File -Recurse | Where-Object { $_.Extension -in @('.yaml', '.yml') })) {
        $content = [IO.File]::ReadAllText($file.FullName)
        if ($content -match '(?m)^kind:\s*Text\s*$' -and
            $content -match '(?m)^\s+name:\s*rollback-procedure\s*$' -and
            $content -match '(?m)^\s+namespace:\s*sample\s*$') {
            $foundResources.Add($file.FullName)
        }
    }
    if ($foundResources.Count -ne 1) { throw "Expected one sample/Text/rollback-procedure resource under '$root'; found $($foundResources.Count)" }
    return $foundResources[0]
}

function Prepare-Fixture([string]$fixtureRoot, [string]$destination, [string]$binary, [string]$tag, [string]$fixtureVersion, [string]$fixtureDigest, [string[]]$baselineBinaries) {
    $root = [IO.Path]::GetFullPath($fixtureRoot)
    if (-not (Test-Path -LiteralPath $root -PathType Container)) { throw "Fixture root does not exist: $root" }
    $work = Join-Path $destination 'worktree'
    if (Test-Path -LiteralPath $work) { throw "Benchmark worktree already exists: $work; choose a fresh output directory" }
    Copy-FixtureTree $root $work
    git -C $work init -b feature/benchmark | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not initialize fixture repository' }
    git -C $work config user.name 'Markitect Benchmark'
    if ($LASTEXITCODE -ne 0) { throw 'Could not configure fixture repository identity' }
    git -C $work config user.email 'markitect-benchmark@users.noreply.github.com'
    if ($LASTEXITCODE -ne 0) { throw 'Could not configure fixture repository identity' }
    git -C $work config core.autocrlf false
    if ($LASTEXITCODE -ne 0) { throw 'Could not configure fixture line endings' }
    git -C $work add .
    if ($LASTEXITCODE -ne 0) { throw 'Could not stage fixture baseline' }
    git -C $work commit -m "Fixture $fixtureVersion baseline" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not commit fixture baseline' }
    $base = (git -C $work rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $base -notmatch '^[0-9a-f]{40}$') { throw 'Could not resolve fixture baseline commit' }

    foreach ($baselineBinary in $baselineBinaries) {
        $checkOutput = (& $baselineBinary check --repo $work --revision $base 2>&1 | Out-String).Trim()
        if ($LASTEXITCODE -ne 0) { throw "A release binary could not check the unchanged $fixtureVersion baseline ($base): $checkOutput" }
    }

    $source = Find-RollbackText $work
    $original = [IO.File]::ReadAllText($source)
    $body = $original.Replace('Pause the rollout', 'Stop the rollout')
    if ($body -eq $original) { throw "Fixture $fixtureVersion mutation had no effect" }
    [IO.File]::WriteAllText($source, $body)
    & $binary render --repo $work --write | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Release $tag could not render its $fixtureVersion candidate fixture" }
    git -C $work add .
    if ($LASTEXITCODE -ne 0) { throw 'Could not stage fixture candidate' }
    git -C $work commit -m "Fixture $fixtureVersion candidate" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not commit fixture candidate' }
    $candidate = (git -C $work rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $candidate -notmatch '^[0-9a-f]{40}$') { throw 'Could not resolve fixture candidate commit' }

    return [pscustomobject]@{
        work = $work; version = $fixtureVersion; root = $root; digest = $fixtureDigest
        baseCommit = $base; candidateCommit = $candidate; preparedByTag = $tag
    }
}

function Measure-Command([string]$binary, [string[]]$arguments, [string]$tag, [string]$fixtureVersion, [string]$fixtureRoot, [string]$fixtureDigest, [string]$baseCommit, [string]$candidateCommit, [string]$operation, [string]$phase, [int]$iteration, [int]$sequence) {
    $start = [Diagnostics.ProcessStartInfo]::new($binary)
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    foreach ($argument in $arguments) { [void]$start.ArgumentList.Add($argument) }
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $start
    $timer = [Diagnostics.Stopwatch]::StartNew()
    [void]$process.Start()
    $stdout = $process.StandardOutput.ReadToEndAsync()
    $stderr = $process.StandardError.ReadToEndAsync()
    $peak = [int64]0
    do {
        try { $process.Refresh(); $peak = [Math]::Max($peak, [int64]$process.PeakWorkingSet64) } catch { }
        if (-not $process.HasExited) { Start-Sleep -Milliseconds 10 }
    } while (-not $process.HasExited)
    $process.WaitForExit()
    try { $process.Refresh(); $peak = [Math]::Max($peak, [int64]$process.PeakWorkingSet64) } catch { }
    $timer.Stop()
    $out = $stdout.GetAwaiter().GetResult()
    $err = $stderr.GetAwaiter().GetResult()
    $exit = $process.ExitCode
    $status = if ($out -match '(?m)^status: ([^\r\n]+)') { $Matches[1] } else { '' }
    [pscustomobject]@{
        tag = $tag; fixtureVersion = $fixtureVersion; fixtureRoot = $fixtureRoot; fixtureDigest = $fixtureDigest
        baseCommit = $baseCommit; candidateCommit = $candidateCommit
        operation = $operation; phase = $phase; iteration = $iteration; sequence = $sequence
        durationMs = [Math]::Round($timer.Elapsed.TotalMilliseconds, 3)
        peakWorkingSetBytes = $(if ($peak -gt 0) { $peak } else { $null }); exitCode = $exit; status = $status
        stdout = $out.Substring(0, [Math]::Min($out.Length, 4096))
        stderr = $err.Substring(0, [Math]::Min($err.Length, 4096))
    }
}

New-Item -ItemType Directory -Force $OutputDirectory | Out-Null
$currentVersion = Get-FixtureVersion $CurrentFixtureRoot
$previousVersion = Get-FixtureVersion $PreviousFixtureRoot
$currentDigest = Get-FixtureDigest $CurrentFixtureRoot
$previousDigest = Get-FixtureDigest $PreviousFixtureRoot
$sameFixture = ($currentVersion -eq $previousVersion -and $currentDigest -eq $previousDigest)
if ($currentVersion -eq $previousVersion -and -not $sameFixture) {
    throw "Fixture roots both claim $currentVersion but their contents differ; preserve fixture versions and introduce a new version for changed workloads"
}

$current = $null
$previous = $null
if ($sameFixture) {
    $shared = Prepare-Fixture $CurrentFixtureRoot (Join-Path $OutputDirectory 'shared-fixture') $CurrentBinary $CurrentTag $currentVersion $currentDigest @($CurrentBinary, $PreviousBinary)
    $current = $shared
    $previous = $shared
} else {
    $current = Prepare-Fixture $CurrentFixtureRoot (Join-Path $OutputDirectory 'current-fixture') $CurrentBinary $CurrentTag $currentVersion $currentDigest @($CurrentBinary)
    $previous = Prepare-Fixture $PreviousFixtureRoot (Join-Path $OutputDirectory 'previous-fixture') $PreviousBinary $PreviousTag $previousVersion $previousDigest @($PreviousBinary)
}

$operations = [ordered]@{
    check = 'check'
    render = 'render'
    context = 'context'
    impact = 'impact'
    verify = 'verify'
}
$contexts = @(
    [pscustomobject]@{ tag = $CurrentTag; binary = $CurrentBinary; fixture = $current },
    [pscustomobject]@{ tag = $PreviousTag; binary = $PreviousBinary; fixture = $previous }
)
$results = [Collections.Generic.List[object]]::new()
$sequence = 0
$operationIndex = 0
foreach ($operation in $operations.Keys) {
    for ($i = 0; $i -le 3; $i++) {
        $first = ($operationIndex + $i) % 2
        foreach ($index in @($first, (1 - $first))) {
            $entry = $contexts[$index]
            $fixture = $entry.fixture
            $argsForRun = [Collections.Generic.List[string]]::new()
            $argsForRun.Add([string]$operations[$operation])
            $argsForRun.Add('--repo')
            $argsForRun.Add($fixture.work)
            switch ($operation) {
                context { $argsForRun.AddRange([string[]]@('--namespace', 'sample', '--kind', 'Skill', '--name', 'rollback-review')) }
                impact { $argsForRun.AddRange([string[]]@('--base', $fixture.baseCommit, '--revision', $fixture.candidateCommit)) }
                verify { $argsForRun.AddRange([string[]]@('--revision', $fixture.candidateCommit)) }
            }
            $sequence++
            $phase = if ($i -eq 0) { 'cold' } else { 'warm' }
            $results.Add((Measure-Command $entry.binary ([string[]]$argsForRun.ToArray()) $entry.tag $fixture.version $fixture.root $fixture.digest $fixture.baseCommit $fixture.candidateCommit $operation $phase $i $sequence))
        }
    }
    $operationIndex++
}

$goVersion = (& go version).Trim()
$entries = @(
    [ordered]@{ role = 'current'; tag = $CurrentTag; fixtureVersion = $current.version; fixtureRoot = $current.root; fixtureDigest = $current.digest; baseCommit = $current.baseCommit; candidateCommit = $current.candidateCommit; preparedByTag = $current.preparedByTag },
    [ordered]@{ role = 'previous'; tag = $PreviousTag; fixtureVersion = $previous.version; fixtureRoot = $previous.root; fixtureDigest = $previous.digest; baseCommit = $previous.baseCommit; candidateCommit = $previous.candidateCommit; preparedByTag = $previous.preparedByTag }
)
$comparability = if ($sameFixture) { 'same fixture version and identical fixture bytes; runtime differences can be compared cautiously' } else { "fixture versions/workloads differ ($previousVersion versus $currentVersion); no apples-to-apples performance claim" }
$record = [ordered]@{
    schemaVersion = 2; sameFixture = $sameFixture; comparable = $sameFixture; comparability = $comparability
    platform = [Runtime.InteropServices.RuntimeInformation]::OSDescription
    architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    runnerOS = $env:RUNNER_OS; runnerArchitecture = $env:RUNNER_ARCH
    runnerImageOS = $env:ImageOS; runnerImageVersion = $env:ImageVersion
    runnerVersion = $env:RUNNER_VERSION; powershellVersion = $PSVersionTable.PSVersion.ToString(); goVersion = $goVersion
    currentTag = $CurrentTag; previousTag = $PreviousTag; entries = $entries
    coldDefinition = 'first fresh process for each measured command on the runner; OS caches are not cleared'
    warmDefinition = 'three later fresh processes on the same runner; release order alternates per operation and iteration'
    results = @($results)
}
$record | ConvertTo-Json -Depth 12 | Set-Content (Join-Path $OutputDirectory 'results.json') -Encoding utf8

$summary = [Collections.Generic.List[string]]::new()
$summary.Add("# Markitect release benchmark: $CurrentTag versus $PreviousTag")
$summary.Add('')
$summary.Add("Current fixture: $($current.version) at $($current.root); base $($current.baseCommit), candidate $($current.candidateCommit).")
$summary.Add("Previous fixture: $($previous.version) at $($previous.root); base $($previous.baseCommit), candidate $($previous.candidateCommit).")
$summary.Add('')
$summary.Add("$comparability. $($record.platform); first-run and three subsequent fresh-process runs. Release order alternates; no cache reset or performance gate.")
$summary.Add('')
$summary.Add('| Command | Cold ms, previous/current | Warm median ms, previous/current | Warm range ms, previous/current | Change | Peak MiB, previous/current | Result |')
$summary.Add('|---|---:|---:|---:|---:|---:|---|')
foreach ($operation in $operations.Keys) {
    $old = @($results | Where-Object { $_.tag -eq $PreviousTag -and $_.operation -eq $operation -and $_.phase -eq 'warm' } | Sort-Object durationMs)
    $new = @($results | Where-Object { $_.tag -eq $CurrentTag -and $_.operation -eq $operation -and $_.phase -eq 'warm' } | Sort-Object durationMs)
    $before = $old[1].durationMs; $after = $new[1].durationMs
    $change = if ($sameFixture -and $before -gt 0) { (100 * ($after - $before) / $before).ToString('+0.0;-0.0;0.0', [Globalization.CultureInfo]::InvariantCulture) + '%' } else { 'n/a' }
    $oldCold = @($results | Where-Object { $_.tag -eq $PreviousTag -and $_.operation -eq $operation -and $_.phase -eq 'cold' })[0].durationMs
    $newCold = @($results | Where-Object { $_.tag -eq $CurrentTag -and $_.operation -eq $operation -and $_.phase -eq 'cold' })[0].durationMs
    $oldPeak = [Math]::Round(((@($results | Where-Object { $_.tag -eq $PreviousTag -and $_.operation -eq $operation }) | Measure-Object peakWorkingSetBytes -Maximum).Maximum / 1MB), 1)
    $newPeak = [Math]::Round(((@($results | Where-Object { $_.tag -eq $CurrentTag -and $_.operation -eq $operation }) | Measure-Object peakWorkingSetBytes -Maximum).Maximum / 1MB), 1)
    $pass = if (@($results | Where-Object { $_.operation -eq $operation -and $_.exitCode -ne 0 }).Count -eq 0) { 'passed' } else { 'failed' }
    $summary.Add("| $operation | $oldCold / $newCold | $before / $after | $($old[0].durationMs)-$($old[2].durationMs) / $($new[0].durationMs)-$($new[2].durationMs) | $change | $oldPeak / $newPeak | $pass |")
}
$summary.Add('')
$missingMemory = @($results | Where-Object { $null -eq $_.peakWorkingSetBytes }).Count
$summary.Add("Raw duration, peak memory, exit code, status and bounded output for every run are in results.json. Peak memory was unavailable for $missingMemory short-lived processes. Differences include runner and cache noise; inspect them before attributing a regression.")
$summary | Set-Content (Join-Path $OutputDirectory 'summary.md') -Encoding utf8
if (@($results | Where-Object { $_.exitCode -ne 0 }).Count -gt 0) { throw 'One or more benchmark commands failed; inspect results.json' }
