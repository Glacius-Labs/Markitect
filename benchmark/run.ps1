param(
    [Parameter(Mandatory)][string]$CurrentBinary,
    [Parameter(Mandatory)][string]$PreviousBinary,
    [Parameter(Mandatory)][string]$CurrentTag,
    [Parameter(Mandatory)][string]$PreviousTag,
    [Parameter(Mandatory)][string]$FixtureRoot,
    [Parameter(Mandatory)][string]$OutputDirectory
)
$ErrorActionPreference = 'Stop'
New-Item -ItemType Directory -Force $OutputDirectory | Out-Null
$work = Join-Path $OutputDirectory 'fixture-worktree'
New-Item -ItemType Directory -Force $work | Out-Null
Copy-Item (Join-Path $FixtureRoot '*') $work -Recurse -Force
git -C $work init -b feature/benchmark | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Could not initialize fixture repository' }
git -C $work config user.name 'Markitect Benchmark'
git -C $work config user.email 'markitect-benchmark@users.noreply.github.com'
git -C $work config core.autocrlf false
git -C $work add .
git -C $work commit -m 'Fixture v1 baseline' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Could not commit fixture baseline' }
$base = (git -C $work rev-parse HEAD).Trim()
$source = Join-Path $work 'docs/text/rollback-procedure.yaml'
$body = [IO.File]::ReadAllText($source)
$body = $body.Replace('Pause the rollout', 'Stop the rollout')
if ($body -eq [IO.File]::ReadAllText($source)) { throw 'Fixture mutation had no effect' }
[IO.File]::WriteAllText($source, $body)
& $CurrentBinary render --repo $work --write | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Could not render candidate fixture' }
git -C $work add .
git -C $work commit -m 'Fixture v1 candidate' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Could not commit fixture candidate' }
$candidate = (git -C $work rev-parse HEAD).Trim()

function Measure-Command([string]$binary, [string[]]$arguments, [string]$tag, [string]$operation, [string]$phase, [int]$iteration, [int]$sequence) {
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
        tag = $tag; operation = $operation; phase = $phase; iteration = $iteration; sequence = $sequence
        durationMs = [Math]::Round($timer.Elapsed.TotalMilliseconds, 3)
        peakWorkingSetBytes = $(if ($peak -gt 0) { $peak } else { $null }); exitCode = $exit; status = $status
        stdout = $out.Substring(0, [Math]::Min($out.Length, 4096))
        stderr = $err.Substring(0, [Math]::Min($err.Length, 4096))
    }
}

$operations = [ordered]@{
    check = @('check', '--repo', $work)
    render = @('render', '--repo', $work)
    context = @('context', '--repo', $work, '--namespace', 'sample', '--kind', 'Skill', '--name', 'rollback-review')
    impact = @('impact', '--repo', $work, '--base', $base, '--revision', $candidate)
    verify = @('verify', '--repo', $work, '--revision', $candidate)
}
$results = [Collections.Generic.List[object]]::new()
$entries = @(@{tag=$PreviousTag; binary=$PreviousBinary}, @{tag=$CurrentTag; binary=$CurrentBinary})
$sequence = 0
$operationIndex = 0
foreach ($operation in $operations.Keys) {
    $arguments = [string[]]$operations[$operation]
    for ($i = 0; $i -le 3; $i++) {
        $first = ($operationIndex + $i) % 2
        foreach ($index in @($first, (1 - $first))) {
            $entry = $entries[$index]
            $sequence++
            $phase = if ($i -eq 0) { 'cold' } else { 'warm' }
            $results.Add((Measure-Command $entry.binary $arguments $entry.tag $operation $phase $i $sequence))
        }
    }
    $operationIndex++
}
$goVersion = (& go version).Trim()
$record = [ordered]@{
    schemaVersion = 1; fixture = 'v1'; platform = [Runtime.InteropServices.RuntimeInformation]::OSDescription
    architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    runnerOS = $env:RUNNER_OS; runnerArchitecture = $env:RUNNER_ARCH
    runnerImageOS = $env:ImageOS; runnerImageVersion = $env:ImageVersion
    runnerVersion = $env:RUNNER_VERSION; powershellVersion = $PSVersionTable.PSVersion.ToString(); goVersion = $goVersion
    baseCommit = $base; candidateCommit = $candidate; currentTag = $CurrentTag; previousTag = $PreviousTag
    coldDefinition = 'first fresh process for each measured command on the runner; OS caches are not cleared'
    warmDefinition = 'three later fresh processes on the same runner; release order alternates per operation and iteration'
    results = @($results)
}
$record | ConvertTo-Json -Depth 12 | Set-Content (Join-Path $OutputDirectory 'results.json') -Encoding utf8
$summary = [Collections.Generic.List[string]]::new()
$summary.Add("# Markitect release benchmark: $CurrentTag versus $PreviousTag")
$summary.Add('')
$summary.Add("Fixture v1; $($record.platform); first-run and three subsequent fresh-process runs. Release order alternates; no cache reset or performance gate.")
$summary.Add('')
$summary.Add('| Command | Cold ms, previous/current | Warm median ms, previous/current | Warm range ms, previous/current | Change | Peak MiB, previous/current | Result |')
$summary.Add('|---|---:|---:|---:|---:|---:|---|')
foreach ($operation in $operations.Keys) {
    $old = @($results | Where-Object { $_.tag -eq $PreviousTag -and $_.operation -eq $operation -and $_.phase -eq 'warm' } | Sort-Object durationMs)
    $new = @($results | Where-Object { $_.tag -eq $CurrentTag -and $_.operation -eq $operation -and $_.phase -eq 'warm' } | Sort-Object durationMs)
    $before = $old[1].durationMs; $after = $new[1].durationMs
    $change = if ($before -gt 0) { (100 * ($after - $before) / $before).ToString('+0.0;-0.0;0.0', [Globalization.CultureInfo]::InvariantCulture) + '%' } else { 'n/a' }
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
