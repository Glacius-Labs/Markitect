[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string] $OutputDirectory,
    [Parameter(Mandatory = $true)][string] $MarkitectExecutable,
    [ValidateRange(0, 2)][int] $MaxRepairs = 1,
    [ValidateRange(1, 4096)][int] $MaxCalls = 32,
    [switch] $OutsideScope
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
    foreach ($argument in $Arguments) { $start.ArgumentList.Add($argument) }
    $stdout = [System.IO.File]::Create($StdoutPath)
    $stderr = [System.IO.File]::Create($StderrPath)
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $start
    try {
        if (-not $process.Start()) { throw "Could not start $Executable" }
        $stdoutCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdout)
        $stderrCopy = $process.StandardError.BaseStream.CopyToAsync($stderr)
        if (-not $process.WaitForExit(1800000)) {
            try { $process.Kill($true) } catch { }
            $process.WaitForExit()
            [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]] @($stdoutCopy, $stderrCopy))
            $stdout.Flush($true)
            $stderr.Flush($true)
            return 124
        }
        [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]] @($stdoutCopy, $stderrCopy))
        $stdout.Flush($true)
        $stderr.Flush($true)
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
    if ($LASTEXITCODE -ne 0) { throw "git $($GitArguments -join ' ') failed with exit $LASTEXITCODE" }
    return (($lines -join "`n").Trim())
}

function Get-OptionalProperty {
    param([AllowNull()][object] $Object, [Parameter(Mandatory = $true)][string] $Name)
    if ($null -eq $Object) { return }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property -or $null -eq $property.Value) { return }
    return $property.Value
}

function New-RunnerSpec {
    param([string] $Slot, [string] $Mode, [string] $ParallelToken = '')
    $actorArgs = @($Mode)
    if ($Mode -in @('quantity', 'price', 'outside-scope')) { $actorArgs += $ParallelToken }
    return @{
        slotId = $Slot; command = $script:RunnerExecutable; args = $actorArgs
        model = 'deterministic-g3-recursion-mechanics-fixture'; modelOptions = @{}
        providerVersion = 'fixture-v1'; timeoutSeconds = 120
        maxStdoutBytes = 1048576; maxStderrBytes = 1048576; runtimeFiles = @()
    }
}

if (-not [System.IO.Path]::IsPathRooted($OutputDirectory) -or -not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
    throw 'OutputDirectory must be an absolute path to an existing directory.'
}
$resolvedOutput = (Resolve-Path -LiteralPath $OutputDirectory).Path
$outputItem = Get-Item -LiteralPath $resolvedOutput
if ($outputItem.Attributes -band [System.IO.FileAttributes]::ReparsePoint) { throw 'OutputDirectory must not be a reparse point.' }
if (-not [System.IO.Path]::IsPathRooted($MarkitectExecutable) -or -not (Test-Path -LiteralPath $MarkitectExecutable -PathType Leaf)) {
    throw 'MarkitectExecutable must be an absolute source-built executable path.'
}
$resolvedMarkitect = (Resolve-Path -LiteralPath $MarkitectExecutable).Path
$fixture = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '.'))
$checkout = [System.IO.Path]::GetFullPath((Join-Path $fixture '..\..'))
$comparison = [System.StringComparison]::OrdinalIgnoreCase
$checkoutPrefix = $checkout.TrimEnd([char[]] @('\', '/')) + [System.IO.Path]::DirectorySeparatorChar
if ($resolvedOutput.Equals($checkout, $comparison) -or $resolvedOutput.StartsWith($checkoutPrefix, $comparison)) {
    throw 'OutputDirectory must be outside the Markitect checkout.'
}

$script:TrialRoot = Join-Path $resolvedOutput ('government-g3-trial-' + [guid]::NewGuid().ToString('N'))
if (Test-Path -LiteralPath $script:TrialRoot) { throw 'Refusing to reuse an existing trial directory.' }
New-Item -ItemType Directory -Path $script:TrialRoot | Out-Null
$script:Repository = Join-Path $script:TrialRoot 'repo'
$binDirectory = Join-Path $script:TrialRoot 'bin'
$stateDirectory = Join-Path $script:TrialRoot 'state'
$temporaryDirectory = Join-Path $script:TrialRoot 'temporary'
foreach ($directory in @($script:Repository, $binDirectory, $stateDirectory, $temporaryDirectory)) { New-Item -ItemType Directory -Path $directory | Out-Null }

$fixtureFiles = @(
    'README.md', 'government.yaml', 'order.yaml', 'go.mod', 'trial.ps1',
    'runner/main.go', 'quantity/quantity.txt', 'quantity/quantity_test.go',
    'price/price.txt', 'price/price_test.go', 'integration/invoice.txt', 'integration/invoice_test.go'
)
foreach ($relativePath in $fixtureFiles) {
    $sourcePath = Join-Path $fixture ($relativePath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) { throw "Fixture input is missing: $relativePath" }
    $destinationPath = Join-Path $script:Repository ($relativePath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    New-Item -ItemType Directory -Path (Split-Path -Parent $destinationPath) -Force | Out-Null
    Copy-Item -LiteralPath $sourcePath -Destination $destinationPath
}

# Bind the prior Constitution from the actual source-built CLI so editing the
# fixture cannot leave a stale hand-entered model digest.
$inspectOut = Join-Path $script:TrialRoot 'inspect.stdout.yaml'
$inspectErr = Join-Path $script:TrialRoot 'inspect.stderr.txt'
$inspectExit = Invoke-ExternalProcess -Executable $resolvedMarkitect -Arguments @('government', '--action', 'inspect', '--repo', $script:Repository, '--config', 'government.yaml') `
    -WorkingDirectory $checkout -StdoutPath $inspectOut -StderrPath $inspectErr
$inspectText = [System.IO.File]::ReadAllText($inspectOut)
$digestMatch = [regex]::Match($inspectText, '(?ms)^model:\s*\r?\n\s+digest:\s+(sha256:[0-9a-f]{64})')
if (-not $digestMatch.Success) { throw "Could not read the model digest from inspect output (exit $inspectExit). See $inspectOut" }
$orderPath = Join-Path $script:Repository 'order.yaml'
$orderText = [System.IO.File]::ReadAllText($orderPath)
$orderText = [regex]::Replace($orderText, '(?m)^activeConstitution:\s*.*$', ('activeConstitution: ' + $digestMatch.Groups[1].Value))
[System.IO.File]::WriteAllText($orderPath, $orderText, [System.Text.UTF8Encoding]::new($false))

$script:RunnerExecutable = Join-Path $binDirectory 'government-g3-runner.exe'
$buildExit = Invoke-ExternalProcess -Executable 'go' -Arguments @('build', '-o', $script:RunnerExecutable, './runner') -WorkingDirectory $fixture `
    -StdoutPath (Join-Path $script:TrialRoot 'runner-build.stdout.txt') -StderrPath (Join-Path $script:TrialRoot 'runner-build.stderr.txt')
if ($buildExit -ne 0) { throw "Building the runner failed with exit $buildExit; see the trial directory $script:TrialRoot" }

Invoke-Git @('init', '-b', 'main') | Out-Null
Invoke-Git @('config', 'user.name', 'Government G3 mechanics trial') | Out-Null
Invoke-Git @('config', 'user.email', 'government-g3@example.invalid') | Out-Null
Invoke-Git @('config', 'core.autocrlf', 'false') | Out-Null
Invoke-Git @('add', '--all') | Out-Null
Invoke-Git @('-c', 'commit.gpgsign=false', 'commit', '-m', 'G3 prior active fixture') | Out-Null
$base = Invoke-Git @('rev-parse', '--verify', 'HEAD^{commit}')
$activeRef = 'refs/markitect/government/active/example'
Invoke-Git @('update-ref', $activeRef, $base) | Out-Null

$quantityArea = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Area'; namespace = 'invoice'; name = 'quantity' }
$priceArea = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Area'; namespace = 'invoice'; name = 'price' }
$parallelToken = 'g3-' + [guid]::NewGuid().ToString('N')
$quantityMode = if ($OutsideScope) { 'outside-scope' } else { 'quantity' }
$runner = {
    param([string] $slot, [string] $mode, [string] $token = '')
    return New-RunnerSpec -Slot $slot -Mode $mode -ParallelToken $token
}
$runtime = @{
    apiVersion = 'markitect.government-execution/v1alpha1'
    activeRef = $activeRef; expectedBase = $base; timeoutSeconds = 1800
    stateDirectory = $stateDirectory; temporaryDirectory = $temporaryDirectory
    executor = (& $runner 'root-executor' 'root')
    verifier = (& $runner 'root-review' 'root')
    ressorts = @(
        @{ ressort = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'invoice'; name = 'arithmetic' }; runner = (& $runner 'ressort-arithmetic' 'assent') },
        @{ ressort = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'invoice'; name = 'unaffected' }; runner = (& $runner 'ressort-unaffected' 'assent-unaffected') }
    )
    checks = @(@{ name = 'invoice-composition'; run = @('go', 'test', './...', '-count=1'); timeoutSeconds = 90 })
    recursion = @{
        limits = @{ maxDepth = 2; maxFanout = 2; maxCalls = $MaxCalls }
        parallelism = 2
        maxRepairs = $MaxRepairs
        areas = @(
            @{ area = $quantityArea; executor = (& $runner 'quantity-executor' $quantityMode $parallelToken); verifier = (& $runner 'quantity-review' 'quantity'); checks = @(@{ name = 'quantity-local'; run = @('go', 'test', './quantity', '-count=1'); timeoutSeconds = 45 }) },
            @{ area = $priceArea; executor = (& $runner 'price-executor' 'price' $parallelToken); verifier = (& $runner 'price-review' 'price'); checks = @(@{ name = 'price-local'; run = @('go', 'test', './price', '-count=1'); timeoutSeconds = 45 }) }
        )
    }
}
$runtimePath = Join-Path $script:TrialRoot 'runtime.json'
$runtimeJson = ConvertTo-Json -InputObject $runtime -Depth 32
[System.IO.File]::WriteAllText($runtimePath, $runtimeJson, [System.Text.UTF8Encoding]::new($false))

$stdoutPath = Join-Path $script:TrialRoot 'markitect.stdout.json'
$stderrPath = Join-Path $script:TrialRoot 'markitect.stderr.txt'
$markitectArguments = @('government', '--action', 'run', '--repo', $script:Repository, '--config', 'government.yaml', '--order', 'order.yaml', '--runtime', $runtimePath, '--write')
$watch = [System.Diagnostics.Stopwatch]::StartNew()
$nativeExit = Invoke-ExternalProcess -Executable $resolvedMarkitect -Arguments $markitectArguments -WorkingDirectory $checkout -StdoutPath $stdoutPath -StderrPath $stderrPath
$watch.Stop()
$report = $null
$parseError = $null
try { $report = Get-Content -LiteralPath $stdoutPath -Raw -Encoding UTF8 | ConvertFrom-Json } catch { $parseError = $_.Exception.Message }
$finalHead = Invoke-Git @('rev-parse', '--verify', 'HEAD^{commit}')
$active = Invoke-Git @('rev-parse', '--verify', "$activeRef^{commit}")
$sourceHead = ((& git -C $checkout rev-parse --verify 'HEAD^{commit}') -join "`n").Trim()
if ($LASTEXITCODE -ne 0) { throw 'Could not read source checkout HEAD.' }
$sourceStatus = @(& git -C $checkout status --porcelain | ForEach-Object { $_.ToString() })
if ($LASTEXITCODE -ne 0) { throw 'Could not read source checkout status.' }
$markitectHash = (Get-FileHash -LiteralPath $resolvedMarkitect -Algorithm SHA256).Hash.ToLowerInvariant()
$runnerHash = (Get-FileHash -LiteralPath $script:RunnerExecutable -Algorithm SHA256).Hash.ToLowerInvariant()
$runtimeHash = (Get-FileHash -LiteralPath $runtimePath -Algorithm SHA256).Hash.ToLowerInvariant()
$resultHash = (Get-FileHash -LiteralPath $stdoutPath -Algorithm SHA256).Hash.ToLowerInvariant()
$stderrHash = (Get-FileHash -LiteralPath $stderrPath -Algorithm SHA256).Hash.ToLowerInvariant()
$status = Get-OptionalProperty $report 'status'
$stage = Get-OptionalProperty $report 'stage'
$candidateCommit = Get-OptionalProperty $report 'candidateCommit'
$promotionRecord = Get-OptionalProperty $report 'promotion'
$promotion = Get-OptionalProperty $promotionRecord 'status'
$actorRows = @(Get-OptionalProperty $report 'actors' | Where-Object { $null -ne $_ })
$childExecution = @($actorRows | Where-Object { $_.slotId -in @('quantity-executor', 'price-executor') })
$childWall = 0
foreach ($actor in $childExecution) {
    $resultRecord = Get-OptionalProperty $actor 'result'
    $receiptRecord = Get-OptionalProperty $resultRecord 'receipt'
    $wall = Get-OptionalProperty $receiptRecord 'wallTimeMilliseconds'
    if ($null -ne $wall) { $childWall += [int64]$wall }
}
$actorLogs = @()
$reportPathValue = Get-OptionalProperty $report 'reportPath'
$reportDirectory = if ($reportPathValue) { Split-Path -Parent $reportPathValue } else { '' }
foreach ($actor in $childExecution) {
    $resultRecord = Get-OptionalProperty $actor 'result'
    $receiptRecord = Get-OptionalProperty $resultRecord 'receipt'
    $actorRunId = Get-OptionalProperty $receiptRecord 'runId'
    if (-not $reportDirectory -or -not $actorRunId) { continue }
    $privateLogPath = Join-Path $reportDirectory ($actorRunId + '.jsonl')
    if (Test-Path -LiteralPath $privateLogPath -PathType Leaf) {
        $logHash = (Get-FileHash -LiteralPath $privateLogPath -Algorithm SHA256).Hash.ToLowerInvariant()
        $events = @([System.IO.File]::ReadAllLines($privateLogPath) | Where-Object { $_.Trim().Length -gt 0 } | ForEach-Object { ConvertFrom-Json $_ })
        $privateLogDigest = Get-OptionalProperty $receiptRecord 'privateLogDigest'
        $actorLogs += [pscustomobject]@{ slotId = $actor.slotId; runId = $actorRunId; receiptPrivateLogDigest = $privateLogDigest; fileSha256 = "sha256:$logHash"; digestMatchesReceipt = ($privateLogDigest -eq "sha256:$logHash"); path = $privateLogPath; events = $events }
    }
}
$parallelGroups = @($actorLogs | ForEach-Object { $_.events } | ForEach-Object { $_ } | Where-Object { (Get-OptionalProperty $_ 'token') -eq $parallelToken } | Group-Object generation)
$overlapGroups = @()
foreach ($group in $parallelGroups) {
	$quantityEvents = @($group.Group | Where-Object { $_.side -eq 'quantity' })
	$priceEvents = @($group.Group | Where-Object { $_.side -eq 'price' })
    $qStart = @($quantityEvents | Where-Object { $_.event -eq 'process-start' } | Select-Object -First 1)
    $pStart = @($priceEvents | Where-Object { $_.event -eq 'process-start' } | Select-Object -First 1)
    $qOverlap = @($quantityEvents | Where-Object { $_.event -eq 'sibling-process-overlap' } | Select-Object -First 1)
    $pOverlap = @($priceEvents | Where-Object { $_.event -eq 'sibling-process-overlap' } | Select-Object -First 1)
    $qFinish = @($quantityEvents | Where-Object { $_.event -eq 'process-finish' } | Select-Object -First 1)
    $pFinish = @($priceEvents | Where-Object { $_.event -eq 'process-finish' } | Select-Object -First 1)
    if ($qStart.Count -and $pStart.Count -and $qOverlap.Count -and $pOverlap.Count -and $qFinish.Count -and $pFinish.Count) {
        # ConvertFrom-Json can return DateTime values. Preserve their fractional
        # seconds instead of implicitly formatting them as culture-specific text.
        $overlapGroups += [pscustomobject]@{ generation = $group.Name; quantity = $quantityEvents; price = $priceEvents; quantityRunId = $qStart[0].runId; priceRunId = $pStart[0].runId; intervalOverlap = ([DateTimeOffset]$qStart[0].atUtc -lt [DateTimeOffset]$pFinish[0].atUtc -and [DateTimeOffset]$pStart[0].atUtc -lt [DateTimeOffset]$qFinish[0].atUtc) }
    }
}
$markerEvidence = @()
foreach ($markerPath in @($actorLogs | ForEach-Object { $_.events } | ForEach-Object { $_.markerPath } | Where-Object { $_ } | Sort-Object -Unique)) {
    if (Test-Path -LiteralPath $markerPath -PathType Leaf) {
        $markerLines = [System.IO.File]::ReadAllLines($markerPath)
        $markerEvidence += [pscustomobject]@{ path = $markerPath; sha256 = 'sha256:' + (Get-FileHash -LiteralPath $markerPath -Algorithm SHA256).Hash.ToLowerInvariant(); events = @($markerLines | ForEach-Object { ConvertFrom-Json $_ }) }
    }
}
$parallelismEvidence = (@($overlapGroups | Where-Object { $_.intervalOverlap -and $_.quantityRunId -ne $_.priceRunId }).Count -gt 0 -and @($actorLogs | Where-Object { -not $_.digestMatchesReceipt }).Count -eq 0)
$candidateInvoice = $null
$reportWorkspace = Get-OptionalProperty $report 'workspace'
$rootArea = Get-OptionalProperty $report 'rootArea'
$rootAttempts = @(Get-OptionalProperty $rootArea 'attempts')
$rootAttemptEvidence = @()
if ($rootAttempts.Count -gt 0) { $rootAttemptEvidence = $rootAttempts }
$votes = @(Get-OptionalProperty $report 'votes')
$candidateRecord = Get-OptionalProperty $report 'candidate'
$candidateID = Get-OptionalProperty $candidateRecord 'id'
$evidenceRecord = Get-OptionalProperty $report 'evidence'
$evidenceID = Get-OptionalProperty $evidenceRecord 'id'
$decisionRecord = Get-OptionalProperty $report 'decision'
$decisionID = Get-OptionalProperty $decisionRecord 'id'
$areaWorkspace = $null
if ($rootAttempts.Count -gt 0) { $areaWorkspace = Get-OptionalProperty $rootAttempts[$rootAttempts.Count - 1] 'workspace' }
if ($areaWorkspace -and (Test-Path -LiteralPath (Join-Path $areaWorkspace 'integration\invoice.txt'))) {
    $candidateInvoice = [System.IO.File]::ReadAllText((Join-Path $areaWorkspace 'integration\invoice.txt')).Trim()
} elseif ($reportWorkspace -and (Test-Path -LiteralPath (Join-Path $reportWorkspace 'integration\invoice.txt'))) {
    $candidateInvoice = [System.IO.File]::ReadAllText((Join-Path $reportWorkspace 'integration\invoice.txt')).Trim()
}
$reportError = Get-OptionalProperty $report 'error'
$firstAttemptReview = $null
$lastAttemptReview = $null
if ($rootAttempts.Count -gt 0) {
    $firstReview = Get-OptionalProperty $rootAttempts[0] 'review'
    $firstAttemptReview = Get-OptionalProperty $firstReview 'outcome'
    $lastReview = Get-OptionalProperty $rootAttempts[$rootAttempts.Count - 1] 'review'
    $lastAttemptReview = Get-OptionalProperty $lastReview 'outcome'
}
$firstChildren = @()
if ($rootAttempts.Count -gt 0) { $firstChildren = @(Get-OptionalProperty $rootAttempts[0] 'children') }
$childrenPassed = ($firstChildren.Count -eq 2 -and @($firstChildren | Where-Object {
    $child = $_
    $childAttempts = @(Get-OptionalProperty $child 'attempts')
    $childStatus = Get-OptionalProperty $child 'status'
    if ($childStatus -ne 'passed-scoped' -or $childAttempts.Count -ne 1) { return $true }
    $childChecks = @(Get-OptionalProperty $childAttempts[0] 'checks')
    $childReview = Get-OptionalProperty $childAttempts[0] 'review'
    $childReviewOutcome = Get-OptionalProperty $childReview 'outcome'
    ($childChecks.Count -eq 0 -or @($childChecks | Where-Object { (Get-OptionalProperty $_ 'exitCode') -ne 0 }).Count -gt 0 -or $childReviewOutcome -ne 'passed')
}).Count -eq 0)
$firstRootCheckFailed = $false
$repairRootChecksPassed = $false
$repairReviewPassed = $false
$firstCandidateID = $null
$lastCandidateID = $null
if ($rootAttempts.Count -gt 0) {
    $firstCandidateID = Get-OptionalProperty $rootAttempts[0] 'candidateId'
    $firstChecks = @(Get-OptionalProperty $rootAttempts[0] 'checks')
    $firstRootCheckFailed = ($firstChecks.Count -gt 0 -and @($firstChecks | Where-Object { (Get-OptionalProperty $_ 'exitCode') -ne 0 }).Count -gt 0)
}
if ($rootAttempts.Count -gt 1) {
    $lastAttempt = $rootAttempts[$rootAttempts.Count - 1]
    $lastCandidateID = Get-OptionalProperty $lastAttempt 'candidateId'
    $lastChecks = @(Get-OptionalProperty $lastAttempt 'checks')
    $lastReview = Get-OptionalProperty $lastAttempt 'review'
    $lastReviewOutcome = Get-OptionalProperty $lastReview 'outcome'
    $repairRootChecksPassed = ($lastChecks.Count -gt 0 -and @($lastChecks | Where-Object { (Get-OptionalProperty $_ 'exitCode') -ne 0 }).Count -eq 0)
    $repairReviewPassed = ($lastReviewOutcome -eq 'passed')
}
$votesFresh = ($votes.Count -eq 2 -and $candidateID -and $evidenceID -and @($votes | Where-Object { $_.materialCandidateId -ne $candidateID -or $_.evidenceId -ne $evidenceID }).Count -eq 0)
$voteOutcomes = @($votes | ForEach-Object { $_.outcome })
$unanimousRootVotes = ($votesFresh -and $voteOutcomes -contains 'assent' -and $voteOutcomes -contains 'assent-unaffected')
$expected = if ($MaxRepairs -eq 0 -or $MaxCalls -lt 15 -or $OutsideScope) {
    $boundedCondition = ($nativeExit -ne 0 -and $null -eq $parseError -and $active -eq $base -and $finalHead -eq $base -and $promotion -ne 'promoted' -and $null -eq $evidenceID -and $null -eq $decisionID -and $votes.Count -eq 0)
    if ($MaxRepairs -eq 0) {
        $boundedCondition = $boundedCondition -and $candidateInvoice -eq 'total=11' -and $rootAttempts.Count -eq 1 -and $firstRootCheckFailed -and $firstAttemptReview -eq 'failed' -and $childrenPassed -and $null -eq $evidenceID -and $votes.Count -eq 0
    }
    if ($MaxCalls -lt 15) {
        $rootAreaError = Get-OptionalProperty $rootArea 'error'
        $boundedCondition = $boundedCondition -and ($reportError -match 'global actor invocation limit exhausted' -or $rootAreaError -match 'global actor invocation limit exhausted') -and $actorRows.Count -le $MaxCalls
    }
    if ($OutsideScope) {
        $rootAreaError = Get-OptionalProperty $rootArea 'error'
        $boundedCondition = $boundedCondition -and ($reportError -match 'outside frozen Writer scope' -or $rootAreaError -match 'outside frozen Writer scope')
    }
    $boundedCondition
} else {
    ($nativeExit -eq 0 -and $null -eq $parseError -and $status -eq 'accepted-scoped' -and $stage -eq 'complete' -and $promotion -eq 'promoted' -and $active -eq $candidateCommit -and $finalHead -eq $base -and $candidateInvoice -eq 'total=12' -and $parallelismEvidence -and $rootAttempts.Count -eq 2 -and $firstAttemptReview -eq 'failed' -and $firstRootCheckFailed -and $lastAttemptReview -eq 'passed' -and $repairRootChecksPassed -and $repairReviewPassed -and $childrenPassed -and $lastCandidateID -ne $firstCandidateID -and $votesFresh -and $unanimousRootVotes -and $decisionID)
}
$expectedOutcome = 'repaired-root-promoted'
if ($OutsideScope) { $expectedOutcome = 'child-writer-boundary-rejection' } elseif ($MaxCalls -lt 15) { $expectedOutcome = 'global-invocation-budget-exhaustion' } elseif ($MaxRepairs -eq 0) { $expectedOutcome = 'bounded-repair-exhaustion-blocks-promotion' }
$summary = [ordered]@{
    status = if ($expected) { if ($expectedOutcome -eq 'repaired-root-promoted') { 'passed' } else { 'expected-rejection-passed' } } else { 'failed' }
    expectedOutcome = $expectedOutcome
    maxRepairs = $MaxRepairs; maxCalls = $MaxCalls; outsideScope = [bool]$OutsideScope; nativeExitCode = $nativeExit; reportStatus = $status; reportStage = $stage; parseError = $parseError
    wallTimeMilliseconds = $watch.ElapsedMilliseconds; childExecutionReceiptWallMilliseconds = $childWall; childParallelToken = $parallelToken
    observedIndependentChildProcesses = $childExecution.Count; parallelismEvidence = $parallelismEvidence
    childActorPrivateLogs = $actorLogs; childParallelIntervals = $overlapGroups; childMarkerFiles = $markerEvidence
    sourceCheckoutHead = $sourceHead; sourceCheckoutDirty = ($sourceStatus.Count -gt 0); sourceCheckoutStatus = $sourceStatus
    markitectExecutable = $resolvedMarkitect; markitectBinarySha256 = "sha256:$markitectHash"; runnerBinarySha256 = "sha256:$runnerHash"
    runtimeJsonSha256 = "sha256:$runtimeHash"; resultSha256 = "sha256:$resultHash"; stderrSha256 = "sha256:$stderrHash"
    base = $base; candidateCommit = $candidateCommit; finalHead = $finalHead; activeRef = $activeRef; activeRevision = $active; promotion = $promotion
    candidateTree = Get-OptionalProperty $report 'candidateTree'; candidateId = $candidateID; evidenceId = $evidenceID; decisionId = $decisionID
    candidateInvoice = $candidateInvoice; rootAttempts = $rootAttemptEvidence; childAreasInitiallyPassed = $childrenPassed
    initialRootCheckFailed = $firstRootCheckFailed; initialRootReviewOutcome = $firstAttemptReview; repairRootChecksPassed = $repairRootChecksPassed
    repairRootReviewOutcome = $lastAttemptReview; firstRootCandidateId = $firstCandidateID; repairedRootCandidateId = $lastCandidateID
    votesBoundToFinalEvidence = $votesFresh; unanimousOutcomes = $voteOutcomes; votes = $votes; actors = $actorRows; reportPath = Get-OptionalProperty $report 'reportPath'
    stdoutPath = $stdoutPath; stderrPath = $stderrPath; runtimePath = $runtimePath; trialDirectory = $script:TrialRoot
}
$summaryPath = Join-Path $script:TrialRoot 'summary.json'
[System.IO.File]::WriteAllText($summaryPath, (ConvertTo-Json -InputObject $summary -Depth 32), [System.Text.UTF8Encoding]::new($false))
$summaryJson = ConvertTo-Json -InputObject $summary -Depth 32
Write-Output $summaryJson
Write-Output "Summary: $summaryPath"
if (-not $expected) { Write-Error "Government G3 trial did not meet its expected outcome. Inspect $stdoutPath and $stderrPath."; exit 1 }
