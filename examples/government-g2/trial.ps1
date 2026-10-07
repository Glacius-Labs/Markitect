[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $OutputDirectory,

    [Parameter(Mandatory = $true)]
    [string] $MarkitectExecutable
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-ExternalProcess {
    param(
        [Parameter(Mandatory = $true)][string] $Executable,
        [Parameter(Mandatory = $true)][string[]] $Arguments,
        [Parameter(Mandatory = $true)][string] $WorkingDirectory,
        [Parameter(Mandatory = $true)][string] $StdoutPath,
        [Parameter(Mandatory = $true)][string] $StderrPath
    )

    $start = [System.Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Executable
    $start.WorkingDirectory = $WorkingDirectory
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    # All configured arguments are simple tokens or filesystem paths, which
    # cannot contain a quote on supported Windows filesystems.
    $start.Arguments = (($Arguments | ForEach-Object { '"' + $_.Replace('"', '\"') + '"' }) -join ' ')

    $stdout = [System.IO.File]::Create($StdoutPath)
    $stderr = [System.IO.File]::Create($StderrPath)
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $start
    try {
        if (-not $process.Start()) {
            throw "Could not start $Executable"
        }
        $stdoutCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdout)
        $stderrCopy = $process.StandardError.BaseStream.CopyToAsync($stderr)
        $process.WaitForExit()
        [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]] @($stdoutCopy, $stderrCopy))
        return $process.ExitCode
    }
    finally {
        $stdout.Dispose()
        $stderr.Dispose()
        $process.Dispose()
    }
}

function Invoke-Git {
    param([Parameter(Mandatory = $true)][string[]] $GitArguments)
    $nativeArguments = @('-C', $script:Repository) + $GitArguments
    $lines = & git @nativeArguments
    if ($LASTEXITCODE -ne 0) {
        throw "git $($GitArguments -join ' ') failed with exit $LASTEXITCODE"
    }
    return (($lines -join "`n").Trim())
}

if (-not [System.IO.Path]::IsPathRooted($OutputDirectory)) {
    throw 'OutputDirectory must be an absolute path to an existing directory.'
}
if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
    throw 'OutputDirectory must already exist.'
}
$resolvedOutput = (Resolve-Path -LiteralPath $OutputDirectory).Path
$outputItem = Get-Item -LiteralPath $resolvedOutput
if ($outputItem.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
    throw 'OutputDirectory must resolve directly to an existing directory, not a reparse point.'
}

if (-not [System.IO.Path]::IsPathRooted($MarkitectExecutable)) {
    throw 'MarkitectExecutable must be an absolute source-built executable path.'
}
if (-not (Test-Path -LiteralPath $MarkitectExecutable -PathType Leaf)) {
    throw 'MarkitectExecutable does not exist.'
}
$resolvedMarkitect = (Resolve-Path -LiteralPath $MarkitectExecutable).Path

$fixture = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '.'))
$checkout = [System.IO.Path]::GetFullPath((Join-Path $fixture '..\..'))
$comparison = [System.StringComparison]::OrdinalIgnoreCase
$separator = [System.IO.Path]::DirectorySeparatorChar
$checkoutPrefix = $checkout.TrimEnd([char[]] @('\', '/')) + $separator
if ($resolvedOutput.Equals($checkout, $comparison) -or $resolvedOutput.StartsWith($checkoutPrefix, $comparison)) {
    throw 'OutputDirectory must be outside the Markitect checkout.'
}

$trialName = 'government-g2-trial-' + [guid]::NewGuid().ToString('N')
$script:TrialRoot = Join-Path $resolvedOutput $trialName
if (Test-Path -LiteralPath $script:TrialRoot) {
    throw 'The generated trial directory already exists; refusing to reuse it.'
}
New-Item -ItemType Directory -Path $script:TrialRoot | Out-Null
$script:Repository = Join-Path $script:TrialRoot 'repo'
$binDirectory = Join-Path $script:TrialRoot 'bin'
$stateDirectory = Join-Path $script:TrialRoot 'state'
$temporaryDirectory = Join-Path $script:TrialRoot 'temporary'
foreach ($directory in @($script:Repository, $binDirectory, $stateDirectory, $temporaryDirectory)) {
    New-Item -ItemType Directory -Path $directory | Out-Null
}

$orderText = Get-Content -LiteralPath (Join-Path $fixture 'order.yaml') -Raw
if ($orderText.Contains('REPLACE_AFTER_GOVERNMENT_INSPECT')) {
    throw 'Pin order.yaml to the current Constitution digest before running the trial.'
}

# Copy an explicit fixture allowlist. The harness is included but excluded by
# Government inventory; README.md has its own declared exclusion as well.
$fixtureFiles = @(
    'README.md',
    'government.yaml',
    'order.yaml',
    'go.mod',
    'trial.ps1',
    'runner/main.go',
    'inventory/reservation.go',
    'inventory/reservation_test.go'
)
foreach ($relativePath in $fixtureFiles) {
    $sourcePath = Join-Path $fixture ($relativePath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
        throw "Fixture input is missing: $relativePath"
    }
    $destinationPath = Join-Path $script:Repository ($relativePath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    $destinationDirectory = Split-Path -Parent $destinationPath
    New-Item -ItemType Directory -Path $destinationDirectory -Force | Out-Null
    Copy-Item -LiteralPath $sourcePath -Destination $destinationPath
}

$runnerExecutable = Join-Path $binDirectory 'government-g2-runner.exe'
$buildExit = Invoke-ExternalProcess -Executable 'go' -Arguments @('build', '-o', $runnerExecutable, './runner') -WorkingDirectory $fixture `
    -StdoutPath (Join-Path $script:TrialRoot 'runner-build.stdout.txt') -StderrPath (Join-Path $script:TrialRoot 'runner-build.stderr.txt')
if ($buildExit -ne 0) {
    throw "Building the fixture runner failed with exit $buildExit; see runner-build.stdout.txt and runner-build.stderr.txt in $script:TrialRoot"
}

Invoke-Git @('init', '-b', 'main') | Out-Null
Invoke-Git @('config', 'user.name', 'Government G2 mechanics trial') | Out-Null
Invoke-Git @('config', 'user.email', 'government-g2@example.invalid') | Out-Null
Invoke-Git @('config', 'core.autocrlf', 'false') | Out-Null
Invoke-Git @('add', '--all') | Out-Null
Invoke-Git @('-c', 'commit.gpgsign=false', 'commit', '-m', 'G2 prior active fixture') | Out-Null
$base = Invoke-Git @('rev-parse', '--verify', 'HEAD^{commit}')
$activeRef = 'refs/markitect/government/active/example'
Invoke-Git @('update-ref', $activeRef, $base) | Out-Null

$runnerSpec = {
    param([string] $slot, [string] $mode)
    return @{
        slotId = $slot
        command = $runnerExecutable
        args = @($mode)
        model = 'deterministic-g2-mechanics-fixture'
        modelOptions = @{}
        providerVersion = 'fixture-v1'
        timeoutSeconds = 120
        maxStdoutBytes = 1048576
        maxStderrBytes = 1048576
        runtimeFiles = @()
    }
}
$runtime = @{
    apiVersion = 'markitect.government-execution/v1alpha1'
    activeRef = $activeRef
    expectedBase = $base
    timeoutSeconds = 1800
    stateDirectory = $stateDirectory
    temporaryDirectory = $temporaryDirectory
    executor = (& $runnerSpec 'writer' 'executor')
    verifier = (& $runnerSpec 'independent-review' 'verifier')
    ressorts = @(
        @{
            ressort = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'inventory'; name = 'correctness' }
            runner = (& $runnerSpec 'ressort-correctness' 'assent')
        },
        @{
            ressort = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'inventory'; name = 'maintainability' }
            runner = (& $runnerSpec 'ressort-maintainability' 'assent-unaffected')
        }
    )
    checks = @(
        @{ name = 'inventory-overflow'; run = @('go', 'test', './inventory', '-count=1'); timeoutSeconds = 60 }
    )
}
$runtimePath = Join-Path $script:TrialRoot 'runtime.json'
$runtimeJson = ConvertTo-Json -InputObject $runtime -Depth 20
[System.IO.File]::WriteAllText($runtimePath, $runtimeJson, [System.Text.UTF8Encoding]::new($false))

$stdoutPath = Join-Path $script:TrialRoot 'markitect.stdout.json'
$stderrPath = Join-Path $script:TrialRoot 'markitect.stderr.txt'
$markitectArguments = @(
    'government', '--action', 'run', '--repo', $script:Repository,
    '--config', 'government.yaml', '--order', 'order.yaml',
    '--runtime', $runtimePath, '--write'
)
$nativeExit = Invoke-ExternalProcess -Executable $resolvedMarkitect -Arguments $markitectArguments -WorkingDirectory $checkout `
    -StdoutPath $stdoutPath -StderrPath $stderrPath

$report = $null
$parseError = $null
try {
    $report = Get-Content -LiteralPath $stdoutPath -Raw -Encoding UTF8 | ConvertFrom-Json
}
catch {
    $parseError = $_.Exception.Message
}
$finalHead = Invoke-Git @('rev-parse', '--verify', 'HEAD^{commit}')
$active = Invoke-Git @('rev-parse', '--verify', "$activeRef^{commit}")
$sourceHeadLines = & git -C $checkout rev-parse --verify 'HEAD^{commit}'
$sourceHeadExit = $LASTEXITCODE
if ($sourceHeadExit -ne 0) {
    throw "Could not read the Markitect source checkout HEAD (git exit $sourceHeadExit)."
}
$sourceHead = (($sourceHeadLines -join "`n").Trim())
$sourceStatusLines = & git -C $checkout status --porcelain
$sourceStatusExit = $LASTEXITCODE
if ($sourceStatusExit -ne 0) {
    throw "Could not read the Markitect source checkout status (git exit $sourceStatusExit)."
}
$sourceStatus = @($sourceStatusLines | ForEach-Object { $_.ToString() })
$markitectHash = (Get-FileHash -LiteralPath $resolvedMarkitect -Algorithm SHA256).Hash.ToLowerInvariant()
$runnerHash = (Get-FileHash -LiteralPath $runnerExecutable -Algorithm SHA256).Hash.ToLowerInvariant()
$resultHash = (Get-FileHash -LiteralPath $stdoutPath -Algorithm SHA256).Hash.ToLowerInvariant()
$runtimeHash = (Get-FileHash -LiteralPath $runtimePath -Algorithm SHA256).Hash.ToLowerInvariant()
$promotionStatus = $null
$candidateCommit = $null
$runId = $null
$reportStatus = $null
$reportStage = $null
$reportTimeout = $null
if ($null -ne $report) {
    $runId = $report.runId
    $reportStatus = $report.status
    $reportStage = $report.stage
    $reportTimeout = $report.timeoutSeconds
    $candidateCommit = $report.candidateCommit
    if ($null -ne $report.promotion) {
        $promotionStatus = $report.promotion.status
    }
}
$accepted = $nativeExit -eq 0 -and $null -eq $parseError -and $reportStatus -eq 'accepted-scoped' -and
    $reportStage -eq 'complete' -and $promotionStatus -eq 'promoted' -and $active -eq $candidateCommit -and $finalHead -eq $base
$accepted = $accepted -and $reportTimeout -eq 1800
$summary = [ordered]@{
    status = if ($accepted) { 'passed' } else { 'failed' }
    nativeExitCode = $nativeExit
    reportStatus = $reportStatus
    reportStage = $reportStage
    runtimeTimeoutSeconds = $reportTimeout
    parseError = $parseError
    runId = $runId
    sourceCheckoutHead = $sourceHead
    sourceCheckoutDirty = ($sourceStatus.Count -gt 0)
    sourceCheckoutStatus = $sourceStatus
    markitectExecutable = $resolvedMarkitect
    markitectBinarySha256 = "sha256:$markitectHash"
    runnerBinarySha256 = "sha256:$runnerHash"
    runtimeJsonSha256 = "sha256:$runtimeHash"
    base = $base
    candidateCommit = $candidateCommit
    finalHead = $finalHead
    activeRef = $activeRef
    activeRevision = $active
    promotion = $promotionStatus
    resultSha256 = "sha256:$resultHash"
    reportPath = if ($null -ne $report) { $report.reportPath } else { $null }
    stdoutPath = $stdoutPath
    stderrPath = $stderrPath
    trialDirectory = $script:TrialRoot
}
$summaryPath = Join-Path $script:TrialRoot 'summary.json'
$summaryJson = ConvertTo-Json -InputObject $summary -Depth 20
[System.IO.File]::WriteAllText($summaryPath, $summaryJson, [System.Text.UTF8Encoding]::new($false))
Write-Output $summaryJson
Write-Output "Summary: $summaryPath"
if (-not $accepted) {
    Write-Error "Government G2 mechanics trial did not meet the acceptance checks. Inspect $stdoutPath and $stderrPath."
    exit 1
}
