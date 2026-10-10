[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $MarkitectBinary,

    # markitect-legacy runs the Project/Domain render, check, context, verify and impact steps.
    [Parameter(Mandatory = $true)]
    [string] $MarkitectLegacyBinary,

    [string] $OutputDirectory
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$fixtureRoot = Join-Path $repoRoot 'examples/onboarding/delivery-service'
$oraclePath = Join-Path $fixtureRoot 'tasks/expected.yaml'
$actorPromptPath = Join-Path $fixtureRoot 'tasks/policy-edit-prompt.md'
$binaryPath = (Resolve-Path -LiteralPath $MarkitectBinary).Path
$binaryHash = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
$legacyBinaryPath = (Resolve-Path -LiteralPath $MarkitectLegacyBinary).Path
$legacyBinaryHash = (Get-FileHash -LiteralPath $legacyBinaryPath -Algorithm SHA256).Hash.ToLowerInvariant()

if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path ([IO.Path]::GetTempPath()) ('markitect-adoption-' + [guid]::NewGuid().ToString('N'))
}
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $OutputDirectory) {
    throw "Output directory already exists; choose a fresh path: $OutputDirectory"
}
$null = New-Item -ItemType Directory -Path $OutputDirectory
$logsDirectory = Join-Path $OutputDirectory 'raw-logs'
$null = New-Item -ItemType Directory -Path $logsDirectory
$workspace = Join-Path $OutputDirectory 'workspace'
$initWorkspace = Join-Path $OutputDirectory 'init-smoke'
$metrics = [ordered]@{}
$startUtc = [DateTime]::UtcNow
$runStatus = 'failed'
$failureMessage = ''
$versionOutput = ''
$baseSha = ''
$candidateSha = ''
$outcomes = [ordered]@{
    initPreviewReadOnly = 'not-completed'
    initWriteCreatedExpectedFiles = 'not-completed'
    baselineCheck = 'not-completed'
    contextExpectedPaths = 'not-completed'
    candidateVerify = 'not-completed'
    impactExpectedSet = 'not-completed'
}

function Read-YamlSequence {
    param([string] $Content, [string] $Key)
    $lines = $Content -split "`r?`n"
    $header = [Array]::FindIndex($lines, [Predicate[string]] { param($line) $line -match "^$([regex]::Escape($Key)):\s*$" })
    if ($header -lt 0) { throw "Expected-results YAML has no '$Key' list." }
    $values = [Collections.Generic.List[string]]::new()
    for ($index = $header + 1; $index -lt $lines.Length; $index++) {
        $line = $lines[$index]
        if ($line -match '^\s+-\s+(.+?)\s*$') {
            $values.Add($Matches[1].Trim().Trim('"').Trim("'"))
            continue
        }
        if ($line.Trim() -eq '' -or $line -match '^\s+#') { continue }
        break
    }
    return $values.ToArray()
}

function Assert-ExactSet {
    param([string[]] $Actual, [string[]] $Expected, [string] $Label)
    $actualSet = @($Actual | Sort-Object -Unique)
    $expectedSet = @($Expected | Sort-Object -Unique)
    if (($actualSet -join "`n") -cne ($expectedSet -join "`n")) {
        throw "$Label differed. Expected [$($expectedSet -join ', ')]; observed [$($actualSet -join ', ')]."
    }
}

function Invoke-Timed {
    param([string] $Name, [scriptblock] $Action)
    $timer = [Diagnostics.Stopwatch]::StartNew()
    try {
        $invocation = & $Action
        if ($invocation -is [Collections.IDictionary] -and $invocation.Contains('exitCode')) {
            $result = [string]$invocation.output
            $exitCode = [int]$invocation.exitCode
        }
        else {
            $result = $invocation | Out-String
            $exitCode = 0
        }
        $result | Set-Content -LiteralPath (Join-Path $logsDirectory "$Name.log") -Encoding utf8
        if ($exitCode -ne 0) { throw "$Name failed with exit code $exitCode. See raw log." }
        return $result
    }
    catch {
        $_ | Out-String | Set-Content -LiteralPath (Join-Path $logsDirectory "$Name-error.log") -Encoding utf8
        throw
    }
    finally {
        $timer.Stop()
        $metrics[$Name] = $timer.ElapsedMilliseconds
    }
}

function Invoke-Git {
    param([string] $Directory, [string[]] $Arguments)
    Push-Location $Directory
    try {
        $text = & git @Arguments 2>&1 | Out-String
        if ($LASTEXITCODE -ne 0) { throw "git $($Arguments -join ' ') failed: $text" }
        return $text.Trim()
    }
    finally { Pop-Location }
}

function Invoke-Markitect {
    param([string] $Name, [string[]] $Arguments)
    Invoke-Timed -Name $Name -Action {
        $commandOutput = & $binaryPath @Arguments 2>&1 | Out-String
        $commandExitCode = $LASTEXITCODE
        return @{ output = $commandOutput; exitCode = $commandExitCode }
    }
}

function Invoke-MarkitectLegacy {
    param([string] $Name, [string[]] $Arguments)
    Invoke-Timed -Name $Name -Action {
        $commandOutput = & $legacyBinaryPath @Arguments 2>&1 | Out-String
        $commandExitCode = $LASTEXITCODE
        return @{ output = $commandOutput; exitCode = $commandExitCode }
    }
}

try {
if (-not (Test-Path -LiteralPath $oraclePath) -or -not (Test-Path -LiteralPath $actorPromptPath)) {
    throw 'The fixed task prompt or expected-results file is missing.'
}
$oracle = Get-Content -LiteralPath $oraclePath -Raw
$policyEditPrompt = Get-Content -LiteralPath $actorPromptPath -Raw
$expectedContextPaths = @(Read-YamlSequence -Content $oracle -Key 'contextInputs')
$expectedChangedPaths = @(Read-YamlSequence -Content $oracle -Key 'expectedChangedPaths')
$expectedAffectedResources = @(Read-YamlSequence -Content $oracle -Key 'expectedAffectedResources')
$expectedUnaffectedResources = @(Read-YamlSequence -Content $oracle -Key 'expectedUnaffectedResources')

$versionOutput = Invoke-Markitect -Name 'version' -Arguments @('version')
if ($versionOutput.Trim() -notmatch '^Markitect\s+\d+\.\d+\.\d+(?:[-+][A-Za-z0-9.-]+)?\s+\([A-Za-z0-9_-]+/[A-Za-z0-9_-]+\)$') {
    throw "Unexpected version output: $versionOutput"
}

# Exercise the model-first project init flow in a fresh named branch.
$null = New-Item -ItemType Directory -Path $initWorkspace
$null = Invoke-Git -Directory $initWorkspace -Arguments @('init', '--initial-branch=feature/adoption-init')
$null = Invoke-Git -Directory $initWorkspace -Arguments @('config', 'user.name', 'Markitect onboarding exercise')
$null = Invoke-Git -Directory $initWorkspace -Arguments @('config', 'user.email', 'onboarding@example.invalid')
$initPaths = @(
    '.markitect/.gitignore',
    '.markitect/model/manager.yaml',
    '.markitect/project.yaml',
    '.markitect/runtime.yaml',
    'docs/markitect/project.md'
)
$preview = Invoke-Markitect -Name 'init-preview' -Arguments @('init', '--repo', $initWorkspace, '--name', 'parcel-support')
$previewPlan = ConvertFrom-Json -InputObject $preview
if ($previewPlan.apiVersion -ne 'project.markitect.example.org/v1alpha1' -or $previewPlan.name -ne 'parcel-support' -or $previewPlan.digest -notmatch '^[0-9a-f]{64}$') {
    throw 'Initialization preview returned an invalid or unexpected model-first plan.'
}
$previewPaths = @($previewPlan.files | ForEach-Object { $_.path })
if (($previewPaths -join "`n") -cne ($initPaths -join "`n")) {
    throw "Initialization preview paths differed. Expected [$($initPaths -join ', ')]; observed [$($previewPaths -join ', ')]."
}
foreach ($path in $initPaths) {
    if (Test-Path -LiteralPath (Join-Path $initWorkspace $path)) {
        throw "Initialization preview created $path."
    }
}
$outcomes.initPreviewReadOnly = 'passed'
$write = Invoke-Markitect -Name 'init-write' -Arguments @('init', '--repo', $initWorkspace, '--name', 'parcel-support', '--expect', $previewPlan.digest, '--write')
$writePlan = ConvertFrom-Json -InputObject $write
if ($writePlan.apiVersion -ne 'project.markitect.example.org/v1alpha1' -or $writePlan.name -ne 'parcel-support' -or $writePlan.digest -ne $previewPlan.digest) {
    throw 'Initialization write did not preserve the preview plan identity.'
}
$writtenPaths = @($writePlan.written)
if (($writtenPaths -join "`n") -cne ($initPaths -join "`n")) {
    throw "Initialization write paths differed. Expected [$($initPaths -join ', ')]; observed [$($writtenPaths -join ', ')]."
}
foreach ($path in $initPaths) {
    if (-not (Test-Path -LiteralPath (Join-Path $initWorkspace $path) -PathType Leaf)) {
        throw "Initialization write did not create $path."
    }
}
if (Test-Path -LiteralPath (Join-Path $initWorkspace 'markitect.yaml')) {
    throw 'Initialization created the retired top-level model.'
}
$outcomes.initWriteCreatedExpectedFiles = 'passed'

# Build a disposable fixed-snapshot fixture without task cards or the oracle in its Git tree.
$null = New-Item -ItemType Directory -Path $workspace
Copy-Item -LiteralPath (Join-Path $fixtureRoot 'markitect.yaml') -Destination $workspace
Copy-Item -LiteralPath (Join-Path $fixtureRoot '.markitect') -Destination $workspace -Recurse -Force
Copy-Item -LiteralPath (Join-Path $fixtureRoot 'docs') -Destination $workspace -Recurse
Copy-Item -LiteralPath (Join-Path $fixtureRoot 'scripts') -Destination $workspace -Recurse
$null = Invoke-Git -Directory $workspace -Arguments @('init', '--initial-branch=feature/adoption-exercise')
$null = Invoke-Git -Directory $workspace -Arguments @('config', 'user.name', 'Markitect onboarding exercise')
$null = Invoke-Git -Directory $workspace -Arguments @('config', 'user.email', 'onboarding@example.invalid')
$null = Invoke-MarkitectLegacy -Name 'render-baseline' -Arguments @('render', '--repo', $workspace, '--write')
$null = Invoke-Git -Directory $workspace -Arguments @('add', '.')
$null = Invoke-Git -Directory $workspace -Arguments @('commit', '-m', 'Baseline parcel support resources')
$baseSha = Invoke-Git -Directory $workspace -Arguments @('rev-parse', 'HEAD')
if ($baseSha -notmatch '^[0-9a-f]{40}$') { throw "Baseline is not a full commit SHA: $baseSha" }

$checkBase = Invoke-MarkitectLegacy -Name 'check-baseline' -Arguments @('check', '--repo', $workspace, '--revision', $baseSha)
$outcomes.baselineCheck = 'passed'
$context = Invoke-MarkitectLegacy -Name 'context-baseline' -Arguments @('context', '--repo', $workspace, '--revision', $baseSha, '--kind', 'Skill', '--namespace', 'support', '--name', 'refund-triage')
$contextPaths = @([regex]::Matches($context, '(?m)^ {6}path:\s*"?([^"\s]+)"?\s*$') | ForEach-Object { $_.Groups[1].Value })
Assert-ExactSet -Actual $contextPaths -Expected $expectedContextPaths -Label 'Context input paths'
$outcomes.contextExpectedPaths = 'passed'

# Apply the fixed task to the canonical Text resource and regenerate its managed view.
$policyPath = Join-Path $workspace '.markitect/areas/support/refund-policy.text.yaml'
$policySource = Get-Content -LiteralPath $policyPath -Raw
$oldWindow = 'up to 14 days after delivery'
if (-not $policySource.Contains($oldWindow)) { throw 'The fixed baseline policy wording has changed; review the task and oracle.' }
$candidateSource = $policySource.Replace($oldWindow, 'up to 30 days after delivery')
[IO.File]::WriteAllText($policyPath, $candidateSource, [Text.UTF8Encoding]::new($false))
$null = Invoke-MarkitectLegacy -Name 'render-candidate' -Arguments @('render', '--repo', $workspace, '--write')
$null = Invoke-Git -Directory $workspace -Arguments @('add', '.markitect/areas/support/refund-policy.text.yaml', 'docs/markitect/support/refund-policy.text.md')
$null = Invoke-Git -Directory $workspace -Arguments @('commit', '-m', 'Extend late parcel refund request window')
$candidateSha = Invoke-Git -Directory $workspace -Arguments @('rev-parse', 'HEAD')
if ($candidateSha -notmatch '^[0-9a-f]{40}$') { throw "Candidate is not a full commit SHA: $candidateSha" }
$null = Invoke-Git -Directory $workspace -Arguments @('bundle', 'create', (Join-Path $OutputDirectory 'fixture.bundle'), 'HEAD')

$null = Invoke-MarkitectLegacy -Name 'check-candidate' -Arguments @('check', '--repo', $workspace, '--revision', $candidateSha)
$null = Invoke-MarkitectLegacy -Name 'verify-candidate' -Arguments @('verify', '--repo', $workspace, '--revision', $candidateSha)
$outcomes.candidateVerify = 'passed'
$impact = Invoke-MarkitectLegacy -Name 'impact-fixed-shas' -Arguments @('impact', '--repo', $workspace, '--base', $baseSha, '--revision', $candidateSha)
Assert-ExactSet -Actual @(Read-YamlSequence -Content $impact -Key 'changed') -Expected $expectedChangedPaths -Label 'Impact changed paths'
$affectedResources = @(Read-YamlSequence -Content $impact -Key 'affected')
Assert-ExactSet -Actual $affectedResources -Expected $expectedAffectedResources -Label 'Impact affected resources'
foreach ($identity in $expectedUnaffectedResources) {
    if ($affectedResources -contains $identity) { throw "Impact incorrectly included unaffected resource: $identity" }
}
$outcomes.impactExpectedSet = 'passed'
$runStatus = 'passed'
}
catch {
    $failureMessage = $_.Exception.Message
    throw
}
finally {

$endUtc = [DateTime]::UtcNow
$durationLines = @($metrics.Keys | ForEach-Object { "  ${_}: $($metrics[$_])" })
$outcomeLines = @($outcomes.Keys | ForEach-Object { "  ${_}: $($outcomes[$_])" })
$yamlBinaryPath = ConvertTo-Json -InputObject $binaryPath -Compress
$yamlLegacyBinaryPath = ConvertTo-Json -InputObject $legacyBinaryPath -Compress
$yamlVersion = ConvertTo-Json -InputObject $versionOutput.Trim() -Compress
$record = @(
    'schemaVersion: 1',
    'exercise: parcel-support-onboarding',
    'classification: machine-cli-exercise',
    "startedUtc: `"$($startUtc.ToString('o'))`"",
    "completedUtc: `"$($endUtc.ToString('o'))`"",
    "binaryPath: $yamlBinaryPath",
    "binarySha256: $binaryHash",
    "legacyBinaryPath: $yamlLegacyBinaryPath",
    "legacyBinarySha256: $legacyBinaryHash",
    "versionOutput: $yamlVersion",
    "fixture: examples/onboarding/delivery-service",
    "baseSha: `"$baseSha`"",
    "candidateSha: `"$candidateSha`"",
    "status: $runStatus",
    'outcomes:'
) -join "`n"
$record += "`n" + ($outcomeLines -join "`n") + "`ndurationsMs:`n" + ($durationLines -join "`n")
if ($failureMessage) { $record += "`nfailure: $(ConvertTo-Json -InputObject $failureMessage -Compress)" }
$recordPath = Join-Path $OutputDirectory 'measurement.yaml'
Set-Content -LiteralPath $recordPath -Value $record -Encoding utf8
Write-Output "Measurement: $recordPath"
Write-Output "Raw logs: $logsDirectory"
}
