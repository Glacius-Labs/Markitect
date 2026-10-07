[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][ValidateSet('Prepare', 'Continue', 'Resume')][string] $Action,
    [Parameter(Mandatory = $true)][string] $ProjectDirectory,
    [Parameter(Mandatory = $true)][string] $MarkitectExecutable,
    [string] $AcceptedDigest,
    [string] $QueueDirectory,
    [switch] $MechanicalSmoke
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Utf8([string] $Path, [string] $Text) {
    [IO.File]::WriteAllText($Path, $Text, [Text.UTF8Encoding]::new($false))
}

function Test-IsBelowPath([string] $Child, [string] $Parent) {
    $separator = [IO.Path]::DirectorySeparatorChar
    $alternate = [IO.Path]::AltDirectorySeparatorChar
    $parentPath = [IO.Path]::GetFullPath($Parent).TrimEnd([char[]]@($separator, $alternate))
    $childPath = [IO.Path]::GetFullPath($Child)
    return $childPath.StartsWith($parentPath + $separator, [StringComparison]::OrdinalIgnoreCase)
}

function Invoke-Process([string] $Executable, [string[]] $Arguments, [string] $WorkingDirectory,
    [string] $StdoutPath, [string] $StderrPath, [hashtable] $Environment = @{}, [string] $KillOnMarker = '') {
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Executable; $start.WorkingDirectory = $WorkingDirectory
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true
    foreach ($argument in $Arguments) { $start.ArgumentList.Add($argument) }
    foreach ($key in $Environment.Keys) { $start.Environment[$key] = [string]$Environment[$key] }
    $stdout = [IO.File]::Create($StdoutPath); $stderr = [IO.File]::Create($StderrPath)
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $start
    try {
        if (-not $process.Start()) { throw "Could not start $Executable" }
        $outCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdout)
        $errCopy = $process.StandardError.BaseStream.CopyToAsync($stderr)
        if ($KillOnMarker) {
            $deadline = [DateTime]::UtcNow.AddMinutes(2)
            while (-not [IO.File]::Exists($KillOnMarker) -and -not $process.HasExited -and [DateTime]::UtcNow -lt $deadline) { Start-Sleep -Milliseconds 20 }
            if (-not [IO.File]::Exists($KillOnMarker)) {
                if (-not $process.HasExited) { try { $process.Kill($true) } catch { }; $process.WaitForExit() }
                [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($outCopy, $errCopy))
                $stdout.Flush($true); $stderr.Flush($true)
                throw "Host did not reach the requested update-ref gate: $KillOnMarker"
            }
            $process.Kill($true); $process.WaitForExit()
            [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($outCopy, $errCopy))
            $stdout.Flush($true); $stderr.Flush($true)
            return [pscustomobject]@{ exitCode = $null; killed = $true }
        }
        if (-not $process.WaitForExit(300000)) {
            $process.Kill($true); $process.WaitForExit()
            [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($outCopy, $errCopy))
            $stdout.Flush($true); $stderr.Flush($true)
            return [pscustomobject]@{ exitCode = 124; killed = $false }
        }
        [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($outCopy, $errCopy))
        $stdout.Flush($true); $stderr.Flush($true)
        return [pscustomobject]@{ exitCode = $process.ExitCode; killed = $false }
    } finally { $stdout.Dispose(); $stderr.Dispose(); $process.Dispose() }
}

function Invoke-Markitect([string[]] $Arguments, [string] $Name, [hashtable] $Environment = @{}, [string] $KillOnMarker = '') {
    $invocation = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffffffZ') + '-' + [guid]::NewGuid().ToString('N')
    $out = Join-Path $script:External ($Name + '-' + $invocation + '.stdout.json')
    $err = Join-Path $script:External ($Name + '-' + $invocation + '.stderr.txt')
    $result = Invoke-Process $script:Markitect (@('government','--repo',$script:Project) + $Arguments) $script:Project $out $err $Environment $KillOnMarker
    $value = $null
    try { $value = Get-Content -LiteralPath $out -Raw -Encoding UTF8 | ConvertFrom-Json } catch { }
    return [pscustomobject]@{ exitCode = $result.exitCode; killed = $result.killed; value = $value; stdout = $out; stderr = $err }
}

function Invoke-Git([string[]] $Arguments) {
    $lines = & $script:RealGit -C $script:Project @Arguments
    if ($LASTEXITCODE -ne 0) { throw "git $($Arguments -join ' ') failed with exit $LASTEXITCODE" }
    return (($lines -join "`n").Trim())
}

function Register-ControllerStart([string] $Name, [int] $RoleReservation) {
    $budgetPath = Join-Path $script:External 'budget.json'
    $budget = Get-Content -LiteralPath $budgetPath -Raw | ConvertFrom-Json -AsHashtable
    if ($budget.controllerStarts.Count -ge $budget.maxControllerStarts) { throw 'Walkthrough controller-start budget is exhausted; prepare a fresh project only for an authorized new walk.' }
    if (($budget.reservedRoleStarts + $RoleReservation) -gt $budget.maxRoleStarts) { throw 'Walkthrough deterministic-role budget is exhausted.' }
    $budget.controllerStarts += [ordered]@{ name=$Name; reservedRoleStarts=$RoleReservation; reservedAtUtc=[DateTime]::UtcNow.ToString('o') }
    $budget.reservedRoleStarts += $RoleReservation
    Write-Utf8 $budgetPath (ConvertTo-Json -InputObject $budget -Depth 12)
}

function Get-ModelDigest {
    $result = Invoke-Markitect @('--action','inspect','--format','json','--config','government.yaml') 'inspect'
    if ($result.exitCode -ne 0) { throw "Government inspect is incomplete or invalid ($($result.exitCode)); no digest can be accepted. See $($result.stderr)" }
    if ($null -eq $result.value) { throw "Government inspect did not return valid JSON; no digest can be accepted. See $($result.stderr)" }
    $digest = [string]$result.value.model.digest
    if ($digest -notmatch '^sha256:[0-9a-f]{64}$') { throw 'Inspect did not return a canonical model digest.' }
    if ($result.value.model.findings -or $result.value.survey.coverage -ne 'accounted-within-declared-boundary' -or
        $result.value.survey.unknown -or $result.value.survey.missing -or $result.value.survey.unavailable -or
        $result.value.survey.unassignedSubjects -or $result.value.survey.unrealizedSubjects) { throw 'Inspect did not account for the complete declared project boundary; no digest can be accepted.' }
    return $digest
}

function Assert-PreparedProject {
    foreach ($file in @('intent.yaml','government.yaml','go.mod')) {
        if (-not (Test-Path -LiteralPath (Join-Path $script:Project $file) -PathType Leaf)) { throw "Prepared project is missing $file." }
    }
    if (-not (Test-Path -LiteralPath (Join-Path $script:Project '.git') -PathType Container)) { throw 'ProjectDirectory is not a prepared Git repository.' }
    $top = Invoke-Git @('rev-parse','--show-toplevel')
    if ([IO.Path]::GetFullPath($top) -ne $script:Project) { throw 'Prepared repository root does not equal ProjectDirectory.' }
    if (-not (Test-Path -LiteralPath (Join-Path $script:External 'budget.json') -PathType Leaf)) { throw 'Prepared external run directory or persistent budget record is missing.' }
    if ($script:Project -eq $script:Checkout -or (Test-IsBelowPath $script:Project $script:Checkout)) { throw 'ProjectDirectory must be outside the Markitect checkout.' }
}

function New-RunnerSpec([string] $Slot, [string] $Mode) {
    return @{ slotId = $Slot; command = $script:Runner; args = @($Mode); model = 'deterministic-studio-booking-case'; modelOptions = @{}
        providerVersion = 'fixed-example-v1'; timeoutSeconds = 60; maxStdoutBytes = 1048576; maxStderrBytes = 1048576; runtimeFiles = @() }
}

function New-Runtime([string] $Name, [string] $Ref, [string] $Base, [string] $State, [string] $Temp, [string] $ExecutorMode, [switch] $Amend) {
    New-Item -ItemType Directory -Force -Path $State,$Temp | Out-Null
    $ressort = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'studio' }
    $runtime = [ordered]@{
        apiVersion = 'markitect.government-execution/v1alpha1'; activeRef = $Ref; expectedBase = $Base
        timeoutSeconds = 240; stateDirectory = $State; temporaryDirectory = $Temp
        executor = (New-RunnerSpec 'studio-executor' $ExecutorMode)
        verifier = (New-RunnerSpec 'studio-reviewer' $(if ($ExecutorMode -eq 'independent') { 'review-independent' } else { 'review' }))
        ressorts = @(
            @{ ressort = $ressort + @{ name = 'pricing' }; runner = (New-RunnerSpec 'ressort-pricing' $(if ($ExecutorMode -eq 'independent') { 'assent-unaffected' } else { 'assent' })) },
            @{ ressort = $ressort + @{ name = 'schedule' }; runner = (New-RunnerSpec 'ressort-schedule' 'assent-unaffected') }
        )
        checks = @(@{ name = 'booking-realization'; run = @('go','test','./...','-count=1','-timeout=20s') })
    }
    if ($Amend) { $runtime.amendment = @{ maxRepairs = 0 } }
    $path = Join-Path $script:External ($Name + '.runtime.json')
    Write-Utf8 $path (ConvertTo-Json -InputObject $runtime -Depth 32)
    return $path
}

function New-Order([string] $Path, [string] $Digest, [string] $ActionName, [string] $Namespace, [string] $Subject, [string] $Purpose) {
    $order = [ordered]@{ apiVersion = 'markitect.government-order/v1alpha1'; kind = 'Order'; purpose = $Purpose
        activeConstitution = $Digest; action = $ActionName
        subjects = @(@{ apiVersion = 'markitect.government-example/v1alpha1'; kind = 'Requirement'; namespace = $Namespace; name = $Subject }) }
    Write-Utf8 $Path (ConvertTo-Json -InputObject $order -Depth 16)
}

function New-Backlog([string] $Order, [string] $Runtime, [string] $State) {
    $job = @{ id = 'booking-rule-amendment'; configPath = 'government.yaml'; orderPath = $Order; runtimePath = $Runtime; dependsOn = @() }
    $backlog = [ordered]@{ apiVersion = 'markitect.government-queue/v1alpha1'; stateDirectory = $State
        limits = @{ actorStarts = 4; maxRepairs = 0; maxWallTimeSeconds = 600; maxParallelism = 1 }; jobs = @($job) }
    $path = Join-Path $script:External 'amendment.backlog.json'
    Write-Utf8 $path (ConvertTo-Json -InputObject $backlog -Depth 24)
    return $path
}

if (-not [IO.Path]::IsPathRooted($ProjectDirectory)) { throw 'ProjectDirectory must be an absolute path.' }
if (-not [IO.Path]::IsPathRooted($MarkitectExecutable) -or -not (Test-Path -LiteralPath $MarkitectExecutable -PathType Leaf)) { throw 'MarkitectExecutable must be an absolute executable path.' }
$script:Project = [IO.Path]::GetFullPath($ProjectDirectory)
$script:Markitect = [IO.Path]::GetFullPath($MarkitectExecutable)
$script:Fixture = [IO.Path]::GetFullPath($PSScriptRoot)
$script:Checkout = [IO.Path]::GetFullPath((Join-Path $script:Fixture '..\..'))
$script:RealGit = (Get-Command git -CommandType Application | Select-Object -First 1).Source
$script:External = $script:Project + '.government-run'
$script:Runner = Join-Path $script:External 'bin/studio-runner.exe'
if ($script:Project -eq $script:Checkout -or (Test-IsBelowPath $script:Project $script:Checkout)) { throw 'ProjectDirectory must be outside the Markitect checkout.' }

if ($Action -eq 'Prepare') {
    if (Test-Path -LiteralPath $script:Project) { throw 'Prepare requires a new, nonexistent ProjectDirectory.' }
    if (Test-Path -LiteralPath $script:External) { throw 'Prepare requires a new, nonexistent sibling run directory; prior records are never overwritten.' }
    New-Item -ItemType Directory -Path $script:Project | Out-Null
    Get-ChildItem -LiteralPath $script:Fixture -Force | Where-Object { $_.Name -ne 'walkthrough.ps1' } | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $script:Project -Recurse -Force
    }
    New-Item -ItemType Directory -Path (Join-Path $script:Project 'orders') | Out-Null
    New-Item -ItemType Directory -Path (Join-Path $script:External 'bin'),(Join-Path $script:External 'state'),(Join-Path $script:External 'temporary') | Out-Null
    Write-Utf8 (Join-Path $script:External 'budget.json') (ConvertTo-Json -InputObject ([ordered]@{ apiVersion='markitect.government-project-budget/v1alpha1'; maxControllerStarts=3; maxRoleStarts=8; reservedRoleStarts=0; controllerStarts=@() }) -Depth 8)
    $placeholderDigest = 'sha256:' + ('0' * 64)
    New-Order (Join-Path $script:Project 'orders/implementation-order.json') $placeholderDigest 'implement' 'participant' 'participant-format' 'Implement the separately scoped participant confirmation label.'
    New-Order (Join-Path $script:Project 'orders/amendment-order.json') $placeholderDigest 'amend-model' 'studio' 'booking-rule' 'Add the published booking fee exactly once to the studio total, preserving the fixed fee and protected goal.'
    Invoke-Git @('init','-b','codex/studio-booking') | Out-Null
    Invoke-Git @('config','user.name','Government project walkthrough') | Out-Null
    Invoke-Git @('config','user.email','government-project@example.invalid') | Out-Null
    Invoke-Git @('config','core.autocrlf','false') | Out-Null
    Invoke-Git @('add','--all') | Out-Null
    Invoke-Git @('-c','commit.gpgsign=false','commit','-m','Unaccepted workshop booking model draft') | Out-Null
    $digest = Get-ModelDigest
    Write-Host "Draft model digest: $digest"
    Write-Host 'Review and edit government.yaml and intent.yaml. Prepare records no human assent.'
    Write-Host "After review, run: pwsh -NoProfile -File '$($script:Fixture)\walkthrough.ps1' -Action Continue -ProjectDirectory '$script:Project' -MarkitectExecutable '$script:Markitect' -AcceptedDigest '$digest'"
    exit 0
}

if ($AcceptedDigest -notmatch '^sha256:[0-9a-f]{64}$') { throw "$Action requires -AcceptedDigest sha256:<64 lowercase hex> copied from the reviewed draft." }
Assert-PreparedProject
$branch = Invoke-Git @('branch','--show-current')
if ($branch -ne 'codex/studio-booking') { throw "$Action requires the prepared non-protected branch; found $branch" }
$currentDigest = Get-ModelDigest
if ($currentDigest -cne $AcceptedDigest) { throw "The reviewed digest no longer matches the current model. Current digest is $currentDigest; prepare and review again." }
$acceptancePath = Join-Path $script:Project 'human-acceptance.json'
if ($Action -eq 'Resume') {
    if (-not (Test-Path -LiteralPath $acceptancePath -PathType Leaf)) { throw 'Resume requires the prior explicit model-digest acceptance record.' }
    $record = Get-Content -LiteralPath $acceptancePath -Raw | ConvertFrom-Json
    if ($record.acceptedDigest -cne $AcceptedDigest) { throw 'Resume digest does not match the durable acceptance record.' }
    if (-not [IO.Path]::IsPathRooted($QueueDirectory) -or -not (Test-Path -LiteralPath $QueueDirectory -PathType Container)) { throw 'Resume requires the absolute directory of the existing durable queue.' }
    $queue = [IO.Path]::GetFullPath($QueueDirectory)
    $stateRoot = [IO.Path]::GetFullPath((Join-Path $script:External 'state/amendment'))
    if (-not (Test-IsBelowPath $queue $stateRoot)) { throw 'Resume queue must be below this accepted project''s external amendment state directory.' }
    $backlog = Join-Path $script:External 'amendment.backlog.json'
    if (-not (Test-Path -LiteralPath $backlog -PathType Leaf)) { throw 'Resume backlog is missing; preserve the state and inspect the prior failure.' }
    $shimDirectory = Join-Path $script:External 'git-shim'; $gateDirectory = Join-Path $script:External 'gates/amendment'
    $shim = Join-Path $shimDirectory 'git.exe'
    if (-not (Test-Path -LiteralPath $shim -PathType Leaf) -or -not (Test-Path -LiteralPath $gateDirectory -PathType Container)) { throw 'Resume requires the original durable Git shim and gate directory.' }
    $realGitDirectory = Split-Path -Parent $script:RealGit
    $resumePath = $shimDirectory + [IO.Path]::PathSeparator + $realGitDirectory + [IO.Path]::PathSeparator + $env:PATH
    $resumeEnvironment = @{ PATH=$resumePath; MARKITECT_G5_REAL_GIT=$script:RealGit; MARKITECT_G5_GATE_DIRECTORY=$gateDirectory }
    Register-ControllerStart 'amendment.manual-resume' 0
    $result = Invoke-Markitect @('--action','resume','--backlog',$backlog,'--queue',$queue,'--write') 'amendment-manual-resume' $resumeEnvironment
    if ($result.exitCode -ne 0) { throw "Amendment resume failed ($($result.exitCode)); see $($result.stderr)" }
    $job = @($result.value.jobs | Where-Object { $_.id -eq 'booking-rule-amendment' })[0]
    if ($null -eq $job -or $job.state -notin @('complete','accepted-scoped') -or $result.value.actorStarts -ne 4) { throw 'Manual resume did not complete the frozen amendment without exceeding its four-role queue budget.' }
    $report = Get-Content -LiteralPath $job.reportPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $active = Invoke-Git @('rev-parse','--verify','refs/markitect/government/active/studio-booking^{commit}')
    if ($report.status -ne 'accepted-scoped' -or $report.actors.Count -ne 4 -or $report.priorModelDigest -eq $report.proposedModelDigest -or $active -ne $report.candidateCommit) { throw 'Manual resume did not recover the accepted amended candidate, four-role record, and changed model.' }
    if (@($report.votes | Where-Object { $_.outcome -notin @('assent','assent-unaffected') }).Count -ne 0 -or $report.votes.Count -ne 2) { throw 'Manual resume lacks both frozen Ressort final assents.' }
    if (@($report.checks | Where-Object { $_.ExitCode -ne 0 }).Count -ne 0) { throw 'Manual resume recovered a candidate with a failed realization check.' }
    if (@($report.changedPaths | Where-Object { $_ -in @('government.yaml','studio/studio.go','studio/studio_test.go') }).Count -ne 3) { throw 'Manual resume did not recover the model, implementation, and acceptance-test changes together.' }
    Write-Host "Resumed accepted amendment at commit $active; actor starts remain $($result.value.actorStarts)."
    exit 0
}
$acceptancePath = Join-Path $script:Project 'human-acceptance.json'
if (Test-Path -LiteralPath $acceptancePath) { throw 'This example records one accepted model per project directory; prepare a fresh directory for another acceptance.' }
$implementationOrder = Join-Path $script:Project 'orders/implementation-order.json'
New-Order $implementationOrder $currentDigest 'implement' 'participant' 'participant-format' 'Implement the separately scoped participant confirmation label.'
$amendmentOrder = Join-Path $script:Project 'orders/amendment-order.json'
New-Order $amendmentOrder $currentDigest 'amend-model' 'studio' 'booking-rule' 'Add the published booking fee exactly once to the studio total, preserving the fixed fee and protected goal.'
$implementationPlan = Invoke-Markitect @('--action','plan','--format','json','--config','government.yaml','--order','orders/implementation-order.json') 'plan-implementation'
$amendmentPlan = Invoke-Markitect @('--action','plan','--format','json','--config','government.yaml','--order','orders/amendment-order.json') 'plan-amendment'
foreach ($plan in @($implementationPlan,$amendmentPlan)) {
    if ($plan.exitCode -ne 0 -or $null -eq $plan.value -or $plan.value.status -ne 'planned-within-declared-boundary' -or $plan.value.findings) { throw "A prior-authorized order has no clean scoped plan; see $($plan.stderr)" }
}
if ($MechanicalSmoke) {
    $acceptanceKind = 'mechanical-fixture-acknowledgement'
    $acceptanceStatement = 'The deterministic walkthrough operator supplied the fixture digest to exercise mechanics; this is not human or semantic acceptance.'
    $acceptedBy = 'mechanical-smoke-operator'
} else {
    $acceptanceKind = 'operator-explicit-digest-attestation'
    $acceptanceStatement = 'The operator states that they reviewed the model draft and explicitly supplied its exact digest; identity and model correctness are not authenticated.'
    $acceptedBy = 'operator-supplied-digest-not-authenticated'
}
Write-Utf8 $acceptancePath (ConvertTo-Json -InputObject ([ordered]@{ apiVersion='markitect.government-project-acceptance/v1alpha1'; acceptanceKind=$acceptanceKind; acceptedDigest=$AcceptedDigest; acceptedBy=$acceptedBy; acceptedAtUtc=[DateTime]::UtcNow.ToString('o'); statement=$acceptanceStatement }) -Depth 8)
Invoke-Git @('add','--all') | Out-Null
Invoke-Git @('-c','commit.gpgsign=false','commit','-m',"Accept studio booking model $AcceptedDigest") | Out-Null
$base = Invoke-Git @('rev-parse','--verify','HEAD^{commit}')
$implementationRef = 'refs/markitect/government/active/studio-booking'
Invoke-Git @('update-ref',$implementationRef,$base) | Out-Null

$build = Invoke-Process 'go' @('build','-o',$script:Runner,'./runner') $script:Project (Join-Path $script:External 'runner-build.stdout.txt') (Join-Path $script:External 'runner-build.stderr.txt')
if ($build.exitCode -ne 0) { throw 'Could not build deterministic role adapter; see runner-build output files.' }
$implementationState = Join-Path $script:External 'state/implementation'; $implementationTemp = Join-Path $script:External 'temporary/implementation'
$implementationRuntime = New-Runtime 'implementation' $implementationRef $base $implementationState $implementationTemp 'independent'
Register-ControllerStart 'implementation.run' 4
$implementation = Invoke-Markitect @('--action','run','--config','government.yaml','--order','orders/implementation-order.json','--runtime',$implementationRuntime,'--write') 'implementation'
if ($implementation.exitCode -ne 0) { throw "Bounded implementation run failed ($($implementation.exitCode)); see $($implementation.stderr)" }
if ($implementation.value.status -ne 'accepted-scoped' -or @($implementation.value.actors).Count -ne 4) { throw 'Implementation did not produce an accepted-scoped result with four recorded role starts.' }
$afterImplementation = Invoke-Git @('rev-parse','--verify',"$implementationRef^{commit}")

$amendmentState = Join-Path $script:External 'state/amendment'; $amendmentTemp = Join-Path $script:External 'temporary/amendment'
$amendmentRuntime = New-Runtime 'amendment' $implementationRef $afterImplementation $amendmentState $amendmentTemp 'propose' -Amend
$backlog = New-Backlog 'orders/amendment-order.json' $amendmentRuntime $amendmentState
$shimDirectory = Join-Path $script:External 'git-shim'; $gateDirectory = Join-Path $script:External 'gates/amendment'
New-Item -ItemType Directory -Path $shimDirectory,$gateDirectory | Out-Null
$shim = Join-Path $shimDirectory 'git.exe'
$gateBuild = Invoke-Process 'go' @('build','-o',$shim,'./git-gate') $script:Project (Join-Path $script:External 'gate-build.stdout.txt') (Join-Path $script:External 'gate-build.stderr.txt')
if ($gateBuild.exitCode -ne 0) { throw 'Could not build the local Git interruption gate; see gate-build output files.' }
$realGitDirectory = Split-Path -Parent $script:RealGit
$path = $shimDirectory + [IO.Path]::PathSeparator + $realGitDirectory + [IO.Path]::PathSeparator + $env:PATH
$environment = @{ PATH=$path; MARKITECT_G5_REAL_GIT=$script:RealGit; MARKITECT_G5_GATE_DIRECTORY=$gateDirectory; MARKITECT_G5_GIT_GATE='after' }
$marker = Join-Path $gateDirectory 'paused-after'
Register-ControllerStart 'amendment.queue-interrupted-after-cas' 4
$killed = Invoke-Markitect @('--action','queue','--backlog',$backlog,'--write') 'amendment-killed' $environment $marker
if (-not $killed.killed) { throw 'The amendment queue did not stop at the requested post-update-ref interruption.' }
$queues = @(Get-ChildItem -LiteralPath $amendmentState -Directory -Filter 'government-queue-*')
if ($queues.Count -ne 1) { throw 'Could not identify one durable amendment queue for resume.' }
$resumeEnvironment = @{ PATH=$path; MARKITECT_G5_REAL_GIT=$script:RealGit; MARKITECT_G5_GATE_DIRECTORY=$gateDirectory }
Register-ControllerStart 'amendment.resume-after-cas' 0
$resume = Invoke-Markitect @('--action','resume','--backlog',$backlog,'--queue',$queues[0].FullName,'--write') 'amendment-resume' $resumeEnvironment
if ($resume.exitCode -ne 0) { throw "Amendment resume failed ($($resume.exitCode)); see $($resume.stderr)" }
$job = @($resume.value.jobs | Where-Object { $_.id -eq 'booking-rule-amendment' })[0]
if ($null -eq $job -or $job.state -notin @('complete','accepted-scoped') -or $resume.value.actorStarts -ne 4) { throw 'Resume did not complete the frozen four-role amendment queue.' }
$amendmentReport = Get-Content -LiteralPath $job.reportPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ($amendmentReport.status -ne 'accepted-scoped' -or $amendmentReport.actors.Count -ne 4 -or $amendmentReport.priorModelDigest -eq $amendmentReport.proposedModelDigest) { throw 'Recovered amendment report lacks four role records or a changed model digest.' }
if (@($amendmentReport.votes | Where-Object { $_.outcome -notin @('assent','assent-unaffected') }).Count -ne 0 -or $amendmentReport.votes.Count -ne 2) { throw 'Recovered amendment lacks explicit final assent from both frozen Ressorts.' }
if (@($amendmentReport.checks | Where-Object { $_.ExitCode -ne 0 }).Count -ne 0) { throw 'The amended realization check did not pass.' }
if (@($amendmentReport.changedPaths | Where-Object { $_ -in @('government.yaml','studio/studio.go','studio/studio_test.go') }).Count -ne 3) { throw 'The amendment did not change the model, implementation, and their acceptance test together.' }
$finalCommit = Invoke-Git @('rev-parse','--verify',"$implementationRef^{commit}")
if ($finalCommit -ne $amendmentReport.candidateCommit -or $finalCommit -eq $afterImplementation) { throw 'The final Active ref does not point to the recovered amended candidate.' }
Write-Host "Accepted prior model: $currentDigest"
Write-Host "Final promoted commit after prior-authorized amendment: $finalCommit"
Write-Host "Implementation base: $base; implementation promotion: $afterImplementation"
Write-Host "Interrupted queue resumed: $($queues[0].FullName); cumulative amendment actor starts: $($resume.value.actorStarts)"
