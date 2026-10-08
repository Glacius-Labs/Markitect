[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$AdopterRoot,
    [Parameter(Mandatory)][ValidateSet('task1', 'task2', 'task3')][string]$Task,
    [Parameter(Mandatory)][string]$Arm,
    [Parameter(Mandatory)][string]$OutputDirectory,
    [string]$BaselineManifest,
    [switch]$CaptureBaselineOnly
)

$ErrorActionPreference = 'Stop'
$evaluationRoot = $PSScriptRoot
$repoRoot = (Resolve-Path (Join-Path $evaluationRoot '..\..\..')).Path
$adopterPath = (Resolve-Path -LiteralPath $AdopterRoot).Path
$outputPath = [IO.Path]::GetFullPath($OutputDirectory)
$utf8 = [Text.UTF8Encoding]::new($false)
$excludedSegments = @('.git', 'bin', 'obj')

function Get-TrackedCandidateFiles([string]$Root) {
    Get-ChildItem -LiteralPath $Root -File -Recurse -Force | Where-Object {
        $relative = [IO.Path]::GetRelativePath($Root, $_.FullName)
        $parts = $relative -split '[\\/]'
        -not ($parts | Where-Object { $excludedSegments -contains $_ })
    }
}

function Get-TreeManifest([string]$Root) {
    $files = @{}
    foreach ($file in Get-TrackedCandidateFiles $Root) {
        $relative = [IO.Path]::GetRelativePath($Root, $file.FullName).Replace('\', '/')
        $files[$relative] = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    return $files
}

if ($CaptureBaselineOnly) {
    if (Test-Path -LiteralPath $outputPath -PathType Container) { throw "Baseline output path must be a file: $outputPath" }
    if (Test-Path -LiteralPath $outputPath -PathType Leaf) { throw "Refusing to overwrite baseline manifest: $outputPath" }
    $baselineDirectory = Split-Path -Parent $outputPath
    if (!(Test-Path -LiteralPath $baselineDirectory -PathType Container)) { New-Item -ItemType Directory -Path $baselineDirectory -Force | Out-Null }
    $baseline = [ordered]@{
        schemaVersion = 1
        capturedAtUtc = [DateTime]::UtcNow.ToString('o')
        adopterRoot = $adopterPath
        task = $Task
        arm = $Arm
        excludedDirectoryNames = $excludedSegments
        files = (Get-TreeManifest $adopterPath)
    }
    [IO.File]::WriteAllText($outputPath, ($baseline | ConvertTo-Json -Depth 6), $utf8)
    Write-Output "Baseline manifest written: $outputPath"
    exit 0
}

if ([string]::IsNullOrWhiteSpace($BaselineManifest)) { throw 'BaselineManifest is required for evaluation; capture it before the trial with -CaptureBaselineOnly.' }
$baselinePath = (Resolve-Path -LiteralPath $BaselineManifest).Path
$baseline = Get-Content -LiteralPath $baselinePath -Raw | ConvertFrom-Json
if ($baseline.task -ne $Task -or $baseline.arm -ne $Arm) { throw 'Baseline task/arm identity does not match this evaluation.' }
$before = @{}; foreach ($property in $baseline.files.PSObject.Properties) { $before[$property.Name] = [string]$property.Value }

if (!(Test-Path -LiteralPath $outputPath -PathType Container)) { New-Item -ItemType Directory -Path $outputPath -Force | Out-Null }
$sdk = $null
Push-Location (Join-Path $adopterPath 'src')
try { $sdk = (& dotnet --version 2>&1 | Out-String).Trim(); if ($LASTEXITCODE -ne 0) { throw "dotnet --version failed: $sdk" } }
finally { Pop-Location }

function Invoke-Recorded([string]$Name, [string]$WorkingDirectory, [string]$Executable, [string[]]$Arguments) {
    Push-Location $WorkingDirectory
    try {
        $lines = & $Executable @Arguments 2>&1
        $code = $LASTEXITCODE
        $text = ($lines | Out-String)
        $bytes = $utf8.GetBytes($text)
        $digest = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($bytes)).ToLowerInvariant()
        return [ordered]@{ name = $Name; command = ($Executable + ' ' + ($Arguments -join ' ')); exitCode = $code; stdoutAndStderrSha256 = $digest; output = $text }
    }
    finally { Pop-Location }
}

$runner = Join-Path $evaluationRoot 'MeetingsEvaluation.csproj'
$application = Join-Path $adopterPath 'src/Modules/Meetings/Application/CompanyName.MyMeetings.Modules.Meetings.Application.csproj'
$restoreSource = 'https://api.nuget.org/v3/index.json'
$records = [Collections.Generic.List[object]]::new()
$records.Add((Invoke-Recorded 'meetings-application-build' (Join-Path $adopterPath 'src') 'dotnet' @('build', $application, '--nologo', "-p:RestoreSources=$restoreSource")))
$runnerArgs = @('restore', $runner, '--source', $restoreSource, "-p:AdopterRoot=$adopterPath", '--nologo')
$records.Add((Invoke-Recorded 'bounded-test-runner-restore' (Join-Path $adopterPath 'src') 'dotnet' $runnerArgs))
if ($records[0].exitCode -eq 0 -and $records[1].exitCode -eq 0) {
    $testArgs = @('test', $runner, '--no-restore', '--nologo', "-p:AdopterRoot=$adopterPath", '--logger', 'console;verbosity=normal')
    if ($Task -ne 'task2') { $testArgs += @('--filter', 'FullyQualifiedName!~Task2ValidatorRuntimeTests') }
    $records.Add((Invoke-Recorded 'linked-application-architecture-tests' (Join-Path $adopterPath 'src') 'dotnet' $testArgs))
} else {
    $records.Add([ordered]@{ name = 'linked-application-architecture-tests'; skipped = $true; reason = 'Application build or runner restore failed.' })
}

$referenceChecker = Join-Path $repoRoot 'experiments/real-project-adoption/checks/check-module-project-references.ps1'
$referenceRoot = Join-Path ([IO.Path]::GetTempPath()) ('markitect-comparison-reference-eval-' + [guid]::NewGuid().ToString('N'))
try {
    New-Item -ItemType Directory -Path $referenceRoot | Out-Null
    New-Item -ItemType Junction -Path (Join-Path $referenceRoot 'src') -Target (Join-Path $adopterPath 'src') | Out-Null
    Copy-Item -LiteralPath (Join-Path $repoRoot 'experiments/real-project-adoption/evidence/source-manifest.json') -Destination (Join-Path $referenceRoot 'source-manifest.json')
    $referenceArgs = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $referenceChecker, '-ProjectRoot', $referenceRoot)
    $records.Add((Invoke-Recorded 'evaluated-project-reference-policy' $repoRoot 'pwsh' $referenceArgs))
}
finally {
    $sourceJunction = Join-Path $referenceRoot 'src'
    if (Test-Path -LiteralPath $sourceJunction) { Remove-Item -LiteralPath $sourceJunction -Force }
    $shimManifest = Join-Path $referenceRoot 'source-manifest.json'
    if (Test-Path -LiteralPath $shimManifest) { Remove-Item -LiteralPath $shimManifest -Force }
    if (Test-Path -LiteralPath $referenceRoot) { Remove-Item -LiteralPath $referenceRoot -Force }
}

$after = Get-TreeManifest $adopterPath
$modified = @(); $added = @(); $deleted = @()
foreach ($path in $before.Keys) {
    if (!$after.ContainsKey($path)) { $deleted += $path }
    elseif ($after[$path] -ne $before[$path]) { $modified += $path }
}
foreach ($path in $after.Keys) { if (!$before.ContainsKey($path)) { $added += $path } }
$changedCount = $modified.Count + $added.Count + $deleted.Count
$report = [ordered]@{
    schemaVersion = 1
    evidenceType = 'bounded-meetings-build-tests-and-reference-evaluation'
    task = $Task
    arm = $Arm
    capturedAtUtc = [DateTime]::UtcNow.ToString('o')
    adopterRoot = $adopterPath
    baselineManifest = [ordered]@{ path = $baselinePath; sha256 = (Get-FileHash -LiteralPath $baselinePath -Algorithm SHA256).Hash.ToLowerInvariant() }
    dotnetSdk = $sdk
    targetFramework = 'net8.0'
    restoreSource = $restoreSource
    commands = @($records)
    changedFiles = [ordered]@{ count = $changedCount; modified = @($modified | Sort-Object); added = @($added | Sort-Object); deleted = @($deleted | Sort-Object); excludes = @('directory names .git, bin, obj') }
    limitations = @(
        'ApplicationTests.cs is unchanged upstream source linked into a bounded NUnit runner; this is not the upstream test assembly or its full test suite.',
        'The local TestBase retains only members used by ApplicationTests.cs; DomainAssembly and InfrastructureAssembly are omitted because they are unused and omitted infrastructure is unavailable.',
        'Task 2 runtime tests exercise only the two exact validators, empty/non-empty Guid outcomes, and nonpublic visibility; they do not establish registration or broader business validation.',
        'ProjectReference results cover only the operator-owned 18-project owner/layer map and do not establish runtime dependency behavior.',
        'No database or application runtime is available; SQL semantics, including the Task 1 attendee-plus-guests contract, require human code review.',
        'A failed restore/build makes dependent test outcomes inconclusive; do not replace the test run with mocks.'
    )
}
$reportPath = Join-Path $outputPath 'evaluation.json'
[IO.File]::WriteAllText($reportPath, ($report | ConvertTo-Json -Depth 12), $utf8)
Write-Output "Evaluation report written: $reportPath"
if (@($records | Where-Object { $_.Contains('exitCode') -and $_.exitCode -ne 0 }).Count -gt 0) { exit 1 }
exit 0
