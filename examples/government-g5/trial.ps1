[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string] $OutputDirectory,
    [Parameter(Mandatory = $true)][string] $MarkitectExecutable,
    [ValidateSet('all', 'branches', 'stale-backlog', 'interrupt-before', 'interrupt-after')][string] $Case = 'all'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Utf8([string] $Path, [string] $Text) {
    [IO.File]::WriteAllText($Path, $Text, [Text.UTF8Encoding]::new($false))
}

function Invoke-Process {
    param([string] $Executable, [string[]] $Arguments, [string] $WorkingDirectory, [string] $StdoutPath,
        [string] $StderrPath, [hashtable] $Environment = @{}, [string] $KillOnMarker = '')
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
            if (-not [IO.File]::Exists($KillOnMarker)) { throw "Host did not reach promotion gate: $KillOnMarker" }
            $process.Kill($true); $process.WaitForExit()
            [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($outCopy, $errCopy))
            $stdout.Flush($true); $stderr.Flush($true)
            return [pscustomobject]@{ exitCode = $null; killed = $true }
        }
        if (-not $process.WaitForExit(1800000)) {
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

function Invoke-Git([string] $Repo, [string[]] $Arguments) {
    $lines = & $script:RealGit -C $Repo @Arguments
    if ($LASTEXITCODE -ne 0) { throw "git $($Arguments -join ' ') failed with exit $LASTEXITCODE" }
    return (($lines -join "`n").Trim())
}

function Get-Field([AllowNull()][object] $Value, [string] $Name) {
    if ($null -eq $Value) { return $null }
    $property = $Value.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }
    return $property.Value
}

function Invoke-Markitect([string] $Action, [string[]] $Extra, [string] $Name, [hashtable] $Environment = @{}, [string] $KillOnMarker = '') {
    $stdout = Join-Path $script:TrialRoot ($Name + '.stdout.json')
    $stderr = Join-Path $script:TrialRoot ($Name + '.stderr.txt')
    $args = @('government', '--repo', $script:Repo, '--action', $Action) + $Extra + @('--write')
    $result = Invoke-Process $script:Markitect $args $script:Repo $stdout $stderr $Environment $KillOnMarker
    $value = $null; $parseError = $null
    try { $value = Get-Content -LiteralPath $stdout -Raw -Encoding UTF8 | ConvertFrom-Json } catch { $parseError = $_.Exception.Message }
    return [pscustomobject]@{ exitCode = $result.exitCode; killed = $result.killed; value = $value; parseError = $parseError
        stdout = $stdout; stderr = $stderr; stdoutSha256 = 'sha256:' + (Get-FileHash $stdout -Algorithm SHA256).Hash.ToLowerInvariant()
        stderrSha256 = 'sha256:' + (Get-FileHash $stderr -Algorithm SHA256).Hash.ToLowerInvariant() }
}

function New-RunnerSpec([string] $Slot, [string] $Mode) {
    return @{ slotId = $Slot; command = $script:Runner; args = @($Mode); model = 'deterministic-g5-native-queue'; modelOptions = @{}
        providerVersion = 'fixture-v1'; timeoutSeconds = 120; maxStdoutBytes = 1048576; maxStderrBytes = 1048576; runtimeFiles = @() }
}

function New-Runtime([string] $Name, [string] $Ref, [string] $Base, [string] $State, [string] $Temp, [switch] $Amend) {
    $id = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'invoice' }
    $runtime = [ordered]@{
        apiVersion = 'markitect.government-execution/v1alpha1'; activeRef = $Ref; expectedBase = $Base
        timeoutSeconds = 1800; stateDirectory = $State; temporaryDirectory = $Temp
        executor = (New-RunnerSpec 'root-executor' 'independent'); verifier = (New-RunnerSpec 'root-review' 'review-independent')
        ressorts = @(
            @{ ressort = $id + @{ name = 'arithmetic' }; runner = (New-RunnerSpec 'ressort-arithmetic' 'assent-unaffected') },
            @{ ressort = $id + @{ name = 'unaffected' }; runner = (New-RunnerSpec 'ressort-unaffected' 'assent-unaffected') }
        )
        checks = @(@{ name = 'model-realization'; run = @('go', 'test', './...', '-count=1', '-timeout=20m') })
    }
    if ($Amend) { $runtime.amendment = @{ maxRepairs = 0 } }
    $path = Join-Path $script:TrialRoot ($Name + '.runtime.json')
    Write-Utf8 $path (ConvertTo-Json -InputObject $runtime -Depth 32)
    return $path
}

function Get-ModelDigest([string] $Config) {
    $out = Join-Path $script:TrialRoot ('inspect-' + $Config + '.stdout.yaml')
    $err = Join-Path $script:TrialRoot ('inspect-' + $Config + '.stderr.txt')
    $result = Invoke-Process $script:Markitect @('government', '--action', 'inspect', '--repo', $script:Repo, '--config', $Config) $script:Repo $out $err
    $match = [regex]::Match([IO.File]::ReadAllText($out), '(?ms)^model:\s*\r?\n\s+digest:\s+(sha256:[0-9a-f]{64})')
    if (-not $match.Success) { throw "No model digest in inspect output for $Config" }
    return $match.Groups[1].Value
}

function New-Order([string] $Path, [string] $Digest, [string] $Action, [string] $Namespace, [string] $Subject, [string] $Purpose) {
    $order = [ordered]@{ apiVersion = 'markitect.government-order/v1alpha1'; kind = 'Order'; purpose = $Purpose
        activeConstitution = $Digest; action = $Action
        subjects = @(@{ apiVersion = 'markitect.government-example/v1alpha1'; kind = 'Requirement'; namespace = $Namespace; name = $Subject }) }
    Write-Utf8 $Path (ConvertTo-Json -InputObject $order -Depth 16)
}

function New-Trial([string] $Name) {
    $script:TrialRoot = Join-Path $script:Output ('government-g5-' + $Name + '-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $script:TrialRoot | Out-Null
    $script:Repo = Join-Path $script:TrialRoot 'repo'; $script:State = Join-Path $script:TrialRoot 'state'
    $script:Temp = Join-Path $script:TrialRoot 'temporary'; $script:Bin = Join-Path $script:TrialRoot 'bin'
    foreach ($directory in @($script:Repo, $script:State, $script:Temp, $script:Bin, (Join-Path $script:Repo 'orders'))) { New-Item -ItemType Directory -Path $directory | Out-Null }
    Get-ChildItem -LiteralPath $script:Fixture -Force | Where-Object { $_.Name -notin @('trial.ps1','git-gate','README.md') } | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $script:Repo -Recurse -Force
    }
    if ($Name -in @('branches','interrupt-before','interrupt-after')) {
        $blocked = Get-Content -LiteralPath (Join-Path $script:Repo 'government.yaml') -Raw | ConvertFrom-Json -AsHashtable
        foreach ($definition in $blocked.definitions) {
            if ($definition.kind -eq 'Mandate' -and $definition.metadata.name -eq 'root-prior') { $definition.spec.actions = @('implement', 'review') }
        }
        Write-Utf8 (Join-Path $script:Repo 'blocked-government.yaml') (ConvertTo-Json -InputObject $blocked -Depth 64)
    }
    & $script:RealGit init -b main $script:Repo | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'git init failed' }
    Invoke-Git $script:Repo @('config','user.name','Government G5 native queue fixture') | Out-Null
    Invoke-Git $script:Repo @('config','user.email','government-g5@example.invalid') | Out-Null
    Invoke-Git $script:Repo @('config','core.autocrlf','false') | Out-Null
    Invoke-Git $script:Repo @('add','--all') | Out-Null
    Invoke-Git $script:Repo @('-c','commit.gpgsign=false','commit','-m','G5 prior active fixture') | Out-Null
    $base = Invoke-Git $script:Repo @('rev-parse','--verify','HEAD^{commit}')
    $normalDigest = Get-ModelDigest 'government.yaml'
    $orders = Join-Path $script:Repo 'orders'
    New-Order (Join-Path $orders 'independent.yaml') $normalDigest 'implement' 'receipt' 'receipt-format' "G5 $Name independent queue order."
    if ($Name -in @('branches','interrupt-before','interrupt-after')) {
        $blockedDigest = Get-ModelDigest 'blocked-government.yaml'
        New-Order (Join-Path $orders 'blocked.yaml') $blockedDigest 'amend-model' 'invoice' 'protected-total' 'Attempt a protected goal change without prior amendment authority.'
    }
    Invoke-Git $script:Repo @('add','--all') | Out-Null
    Invoke-Git $script:Repo @('-c','commit.gpgsign=false','commit','-m','G5 finite orders') | Out-Null
    $base = Invoke-Git $script:Repo @('rev-parse','--verify','HEAD^{commit}')
    $script:Base = $base
    $script:Refs = @{}
    $script:Runtimes = @{}
    $script:Runner = Join-Path $script:Bin 'government-g5-runner.exe'
    $jobNames = if ($Name -eq 'branches') { @('blocked','dependent','independent') } elseif ($Name -like 'interrupt-*') { @('promotion','independent') } elseif ($Name -like 'stale-*') { @('only') } else { @('promotion') }
    foreach ($job in $jobNames) {
        $ref = 'refs/markitect/government/active/g5-' + $Name + '-' + $job
        Invoke-Git $script:Repo @('update-ref',$ref,$base) | Out-Null
        $script:Refs[$job] = $ref
        $state = Join-Path $script:State $job; $temp = Join-Path $script:Temp $job
        New-Item -ItemType Directory -Path $state,$temp | Out-Null
        $script:Runtimes[$job] = if ($job -eq 'blocked') { New-Runtime ($job) $ref $base $state $temp -Amend } else { New-Runtime ($job) $ref $base $state $temp }
    }
    $buildOut = Join-Path $script:TrialRoot 'runner-build.stdout.txt'; $buildErr = Join-Path $script:TrialRoot 'runner-build.stderr.txt'
    $built = Invoke-Process 'go' @('build','-o',$script:Runner,'./runner') $script:Fixture $buildOut $buildErr
    if ($built.exitCode -ne 0) { throw "Deterministic runner build failed ($($built.exitCode)); see $buildOut and $buildErr" }
    return $base
}

function New-Backlog([string] $Name, [object[]] $Jobs, [int] $ActorStarts = 40) {
    $backlog = [ordered]@{ apiVersion = 'markitect.government-queue/v1alpha1'; stateDirectory = $script:State
        limits = @{ actorStarts = $ActorStarts; maxRepairs = 4; maxWallTimeSeconds = 14400; maxParallelism = 1 }; jobs = $Jobs }
    $path = Join-Path $script:TrialRoot ($Name + '.backlog.json')
    Write-Utf8 $path (ConvertTo-Json -InputObject $backlog -Depth 24)
    return $path
}

function Job([string] $Id, [string] $Order, [string] $Runtime, [string[]] $Depends = @(), [string] $Config = 'government.yaml') {
    return @{ id = $Id; configPath = $Config; orderPath = ('orders/' + $Order + '.yaml'); runtimePath = $Runtime; dependsOn = $Depends }
}

function Start-Queue([string] $Backlog, [string] $Name, [hashtable] $Environment = @{}, [string] $KillOnMarker = '') {
    return Invoke-Markitect 'queue' @('--backlog',$Backlog) $Name $Environment $KillOnMarker
}

function Resume-Queue([string] $Backlog, [string] $Queue, [string] $Name, [hashtable] $Environment = @{}) {
    return Invoke-Markitect 'resume' @('--backlog',$Backlog,'--queue',$Queue) $Name $Environment
}

function Queue-Directory([object] $Report) {
    foreach ($name in @('queueDirectory','directory','reportPath')) {
        $value = Get-Field $Report $name
        if ($value -and (Test-Path -LiteralPath $value -PathType Container)) { return [IO.Path]::GetFullPath($value) }
        if ($value -and (Test-Path -LiteralPath (Split-Path -Parent $value) -PathType Container) -and (Split-Path -Leaf $value) -eq 'report.json') { return [IO.Path]::GetFullPath((Split-Path -Parent $value)) }
    }
    $found = @(Get-ChildItem -LiteralPath $script:State -Directory -Filter 'government-queue-*')
    if ($found.Count -eq 1) { return $found[0].FullName }
    throw 'Queue report did not identify a unique durable queue directory.'
}

function Count-ActorStarts([object] $Report) {
    $usage = Get-Field $Report 'usage'
    foreach ($name in @('actorStarts','actorsStarted','actorStartsConsumed')) {
        $value = Get-Field $usage $name
        if ($null -ne $value) { return [int]$value }
        $value = Get-Field $Report $name
        if ($null -ne $value) { return [int]$value }
    }
    return -1
}

function Job-State([object] $Job) { return Get-Field $Job 'state' }

function Complete-Case([string] $Name, [bool] $Passed, [string] $Expected, [object[]] $Invocations, [hashtable] $Evidence) {
    $summary = [ordered]@{ case = $Name; status = if ($Passed) { 'passed' } else { 'failed' }; expected = $Expected
        sourceCheckout = $script:Repo; baseRevision = $script:Base; invocations = $Invocations; evidence = $Evidence; trialDirectory = $script:TrialRoot }
    $path = Join-Path $script:TrialRoot 'summary.json'; $summary.summaryPath = $path
    Write-Utf8 $path (ConvertTo-Json -InputObject $summary -Depth 64)
    if (-not $Passed) { throw "G5 $Name failed its native assertion; retained evidence: $path" }
    return $summary
}

function Run-Branches {
    $null = New-Trial 'branches'; $repo = $script:Repo
    $jobs = @(
        (Job 'blocked' 'blocked' $script:Runtimes['blocked'] @() 'blocked-government.yaml'),
        (Job 'dependent' 'independent' $script:Runtimes['dependent'] @('blocked')),
        (Job 'independent' 'independent' $script:Runtimes['independent'])
    )
    $backlog = New-Backlog 'branches' $jobs
    $run = Start-Queue $backlog 'queue'
    $directory = Queue-Directory $run.value
    $states = @(Get-Field $run.value 'jobs')
    $blocked = @($states | Where-Object { (Get-Field $_ 'id') -eq 'blocked' })[0]
    $dependent = @($states | Where-Object { (Get-Field $_ 'id') -eq 'dependent' })[0]
    $independent = @($states | Where-Object { (Get-Field $_ 'id') -eq 'independent' })[0]
    $activeIndependent = Invoke-Git $repo @('rev-parse','--verify',"$($script:Refs['independent'])^{commit}")
    $passed = ($run.exitCode -eq 0 -and $run.parseError -eq $null -and (Job-State $blocked) -in @('blocked','incomplete') -and
        (Job-State $dependent) -in @('blocked','dependency-blocked','skipped') -and (Job-State $independent) -in @('complete','accepted-scoped') -and $activeIndependent -ne $script:Base)
    return Complete-Case 'branches' $passed 'a failed prerequisite blocks only its dependent finite order while a disjoint independent order completes' @($run) @{ queueDirectory = $directory; blocked = $blocked; dependent = $dependent; independent = $independent; activeIndependent = $activeIndependent }
}

function Run-StaleBacklog {
    $null = New-Trial 'stale-runtime'
    $runtimeBacklog = New-Backlog 'runtime' @((Job 'only' 'independent' $script:Runtimes['only']))
    $runtimeStarted = Start-Queue $runtimeBacklog 'runtime-queue'
    $runtimeQueue = Queue-Directory $runtimeStarted.value
    $startsBefore = Count-ActorStarts $runtimeStarted.value
    $runtime = Get-Content -LiteralPath $script:Runtimes['only'] -Raw | ConvertFrom-Json -AsHashtable
    $runtime.timeoutSeconds = 1799
    Write-Utf8 $script:Runtimes['only'] (ConvertTo-Json -InputObject $runtime -Depth 32)
    $runtimeResume = Resume-Queue $runtimeBacklog $runtimeQueue 'resume-stale-runtime'
    $startsAfter = Count-ActorStarts $runtimeResume.value
    $runtimeState = (Get-Field (Get-Field $runtimeResume.value 'jobs' | Select-Object -First 1) 'state')
    $runtimeError = Get-Field (@(Get-Field $runtimeResume.value 'jobs')[0]) 'error'
    $runtimeOverallState = Get-Field $runtimeResume.value 'status'
    $runtimePassed = ($runtimeResume.exitCode -ne 0 -and $runtimeOverallState -eq 'blocked' -and $runtimeState -eq 'accepted-scoped' -and $runtimeError -match 'runtime bytes changed' -and $startsBefore -eq 4 -and $startsAfter -eq $startsBefore -and
        (Invoke-Git $script:Repo @('rev-parse','--verify',"$($script:Refs['only'])^{commit}")) -ne $script:Base)
    $runtimeTrial = $script:TrialRoot

    $null = New-Trial 'stale-backlog'
    $backlog = New-Backlog 'stale' @((Job 'only' 'independent' $script:Runtimes['only']))
    $started = Start-Queue $backlog 'queue'
    $queue = Queue-Directory $started.value
    $before = Get-Field $started.value 'journalDigest'
    $activeBeforeResume = Invoke-Git $script:Repo @('rev-parse','--verify',"$($script:Refs['only'])^{commit}")
    $edited = Get-Content -LiteralPath $backlog -Raw | ConvertFrom-Json -AsHashtable
    $edited.limits.maxParallelism = 2
    $changed = Join-Path $script:TrialRoot 'changed.backlog.json'
    Write-Utf8 $changed (ConvertTo-Json -InputObject $edited -Depth 24)
    $resume = Resume-Queue $changed $queue 'resume-stale'
    $after = $resume.value
    $staleState = Get-Field (@(Get-Field $after 'jobs')[0]) 'state'
    $staleError = Get-Field (@(Get-Field $after 'jobs')[0]) 'error'
    $activeAfterResume = Invoke-Git $script:Repo @('rev-parse','--verify',"$($script:Refs['only'])^{commit}")
    $staleOverallState = Get-Field $after 'status'
    $passed = ($resume.exitCode -ne 0 -and $before -ne (Get-Field $after 'journalDigest') -and $staleOverallState -eq 'blocked' -and $staleState -eq 'accepted-scoped' -and $staleError -match 'backlog bytes changed' -and $activeBeforeResume -eq $activeAfterResume)
    $passed = $passed -and $runtimePassed
    return Complete-Case 'stale-backlog' ($passed -and $runtimePassed) 'changed runtime fingerprints and backlog bytes stop future queue work while preserving accepted-scoped historical results, actors, and Active refs' @($runtimeStarted,$runtimeResume,$started,$resume) @{ staleRuntimeTrial = $runtimeTrial; staleRuntimeQueue = $runtimeQueue; runtimeOverallState = $runtimeOverallState; actorStartsBeforeRuntimeChange = $startsBefore; actorStartsAfterRuntimeChange = $startsAfter; runtimeJobState = $runtimeState; runtimeError = $runtimeError; staleBacklogQueue = $queue; originalJournalDigest = $before; staleJournalDigest = Get-Field $after 'journalDigest'; staleOverallState = $staleOverallState; staleJobState = $staleState; staleError = $staleError; activeBeforeResume = $activeBeforeResume; activeAfterResume = $activeAfterResume }
}

function Run-Interruption([string] $Phase) {
    $name = 'interrupt-' + $Phase
    $null = New-Trial $name
    $budget = if ($Phase -eq 'before') { 5 } else { 8 }
    $backlog = New-Backlog $name @((Job 'promotion' 'independent' $script:Runtimes['promotion']),(Job 'independent' 'independent' $script:Runtimes['independent'])) $budget
    $shimDirectory = Join-Path $script:TrialRoot 'git-shim'; $gateDirectory = Join-Path $script:TrialRoot 'git-gate'
    New-Item -ItemType Directory -Path $shimDirectory,$gateDirectory | Out-Null
    $shim = Join-Path $shimDirectory 'git.exe'
    $buildOut = Join-Path $script:TrialRoot 'git-gate-build.stdout.txt'; $buildErr = Join-Path $script:TrialRoot 'git-gate-build.stderr.txt'
    $build = Invoke-Process 'go' @('build','-o',$shim,'./git-gate') $script:Fixture $buildOut $buildErr
    if ($build.exitCode -ne 0) { throw "Git gate build failed; see $buildOut and $buildErr" }
    $trace = Join-Path $script:TrialRoot 'git-update-ref.jsonl'
    $environment = @{ PATH = $shimDirectory + [IO.Path]::PathSeparator + $env:PATH; MARKITECT_G5_REAL_GIT = $script:RealGit
        MARKITECT_G5_GATE_DIRECTORY = $gateDirectory; MARKITECT_G5_GIT_TRACE = $trace; MARKITECT_G5_GIT_GATE = $Phase }
    $marker = Join-Path $gateDirectory ('paused-' + $Phase)
    $killed = Start-Queue $backlog 'queue-killed' $environment $marker
    $queue = Queue-Directory $killed.value
    $ref = $script:Refs['promotion']
    $activeAtKill = Invoke-Git $script:Repo @('rev-parse','--verify',"$ref^{commit}")
    $eventsAtKill = @(); if (Test-Path -LiteralPath $trace) { $eventsAtKill = @(Get-Content -LiteralPath $trace | ForEach-Object { $_ | ConvertFrom-Json }) }
    $laterCommit = ''
    if ($Phase -eq 'after') {
        $gateRecord = Get-Content -LiteralPath (Join-Path $gateDirectory 'paused-after') -Raw | ConvertFrom-Json
        $candidate = [string]$gateRecord.args[-2]
        if ($candidate -ne $activeAtKill) { throw 'The after-CAS marker candidate does not equal the actual Active ref.' }
        $laterLines = & $script:RealGit -C $script:Repo commit-tree "$candidate^{tree}" -p $candidate -m 'G5 fixture advances an already promoted Active ref'
        if ($LASTEXITCODE -ne 0) { throw 'Could not create the G5 external continuation commit.' }
        $laterCommit = (($laterLines -join "`n").Trim())
        & $script:RealGit -C $script:Repo update-ref --create-reflog -m 'G5 fixture later Active-ref advancement' $ref $laterCommit $candidate | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Could not advance the managed Active ref after the killed host.' }
    }
    $activeBeforeResume = Invoke-Git $script:Repo @('rev-parse','--verify',"$ref^{commit}")
    $resumeEnvironment = @{ PATH = $environment.PATH; MARKITECT_G5_REAL_GIT = $script:RealGit; MARKITECT_G5_GATE_DIRECTORY = $gateDirectory; MARKITECT_G5_GIT_TRACE = $trace }
    $resume = Resume-Queue $backlog $queue 'resume-after-kill' $resumeEnvironment
    $repeat = Resume-Queue $backlog $queue 'resume-repeat' $resumeEnvironment
    $activeAfter = Invoke-Git $script:Repo @('rev-parse','--verify',"$ref^{commit}")
    $eventsFinal = @(); if (Test-Path -LiteralPath $trace) { $eventsFinal = @(Get-Content -LiteralPath $trace | ForEach-Object { $_ | ConvertFrom-Json }) }
    $delegated = @($eventsFinal | Where-Object { $_.phase -eq 'delegated' })
    $promotionRef = $script:Refs['promotion']
    $promotionDelegations = @($delegated | Where-Object { $_.args -contains $promotionRef })
    $report = Get-Field $repeat.value 'jobs'
    $jobReport = @($report | Where-Object { (Get-Field $_ 'id') -eq 'promotion' })[0]
    $independentReport = @($report | Where-Object { (Get-Field $_ 'id') -eq 'independent' })[0]
    $usageAfterResume = Count-ActorStarts $resume.value
    $usageAfterRepeat = Count-ActorStarts $repeat.value
    $usage = Get-Field $repeat.value 'usage'
    $usageCounters = @(@('inputTokens','outputTokens','cachedTokens','toolCalls') | ForEach-Object { Get-Field $usage $_ })
    $unknownProviderUsage = @($usageCounters | Where-Object { (Get-Field $_ 'unknown') -ne $true -or (Get-Field $_ 'known') -ne $false -or (Get-Field $_ 'total') -ne 0 }).Count -eq 0
    if ($Phase -eq 'before') {
        $expected = ($activeAtKill -eq $script:Base -and $activeBeforeResume -eq $script:Base -and $activeAfter -eq $script:Base -and $promotionDelegations.Count -eq 0 -and $usageAfterResume -eq 5 -and $usageAfterRepeat -eq $usageAfterResume -and
            (Job-State $independentReport) -in @('budget-exhausted','blocked','pending','incomplete'))
    } else {
        $expected = ($activeAtKill -ne $script:Base -and $activeBeforeResume -eq $laterCommit -and $activeAfter -eq $laterCommit -and $laterCommit -ne $activeAtKill -and $promotionDelegations.Count -eq 1 -and $usageAfterResume -eq 8 -and $usageAfterRepeat -eq $usageAfterResume -and
            (Job-State $jobReport) -in @('complete','accepted-scoped') -and (Job-State $independentReport) -in @('complete','accepted-scoped'))
    }
    $unknownUsageValid = ($unknownProviderUsage -and $usageCounters.Count -eq 4)
    $exitValid = if ($Phase -eq 'after') { $resume.exitCode -eq 0 -and $repeat.exitCode -eq 0 } else { $null -eq $resume.exitCode -or $resume.exitCode -in @(0,1) }
    $passed = ($killed.killed -and $exitValid -and $expected -and $unknownUsageValid)
    return Complete-Case $name $passed "a real Markitect host process is killed $Phase update-ref; resume proves effect state, retains actor starts against the shared budget, preserves unknown provider usage, and repeat cannot issue another CAS or actor" @($killed,$resume,$repeat) @{ queueDirectory = $queue; gate = Get-Content -LiteralPath $marker -Raw | ConvertFrom-Json; eventsAtKill = $eventsAtKill; eventsFinal = $eventsFinal; primaryPromotionDelegations = $promotionDelegations.Count; totalDelegations = $delegated.Count; activeAtKill = $activeAtKill; activeBeforeResume = $activeBeforeResume; laterCommit = $laterCommit; activeAfter = $activeAfter; actorStartsAfterResume = $usageAfterResume; actorStartsAfterRepeat = $usageAfterRepeat; unknownProviderUsage = $usage; promotion = $jobReport; independent = $independentReport }
}

if (-not [IO.Path]::IsPathRooted($OutputDirectory) -or -not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) { throw 'OutputDirectory must be an absolute existing directory.' }
if (-not [IO.Path]::IsPathRooted($MarkitectExecutable) -or -not (Test-Path -LiteralPath $MarkitectExecutable -PathType Leaf)) { throw 'MarkitectExecutable must be an absolute executable path.' }
$script:Output = [IO.Path]::GetFullPath($OutputDirectory); $script:Markitect = [IO.Path]::GetFullPath($MarkitectExecutable)
$script:RealGit = (Get-Command git -CommandType Application | Select-Object -First 1).Source
$script:Fixture = [IO.Path]::GetFullPath($PSScriptRoot)
$checkout = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
if ($script:Output.StartsWith($checkout.TrimEnd('\') + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'OutputDirectory must be outside the source checkout.' }

$summaries = @()
$cases = if ($Case -eq 'all') { @('branches','stale-backlog','interrupt-before','interrupt-after') } else { @($Case) }
foreach ($caseName in $cases) {
    $summaries += switch -Exact ($caseName) {
        'branches' { Run-Branches }
        'stale-backlog' { Run-StaleBacklog }
        'interrupt-before' { Run-Interruption 'before' }
        'interrupt-after' { Run-Interruption 'after' }
    }
}
foreach ($summary in $summaries) { Write-Output (ConvertTo-Json -InputObject $summary -Depth 64); Write-Output "Summary: $($summary.summaryPath)" }
if ($Case -eq 'all') { Write-Output ('All G5 cases: ' + (($summaries | ForEach-Object { $_.status }) -join ', ')) }
