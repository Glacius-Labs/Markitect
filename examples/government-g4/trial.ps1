[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string] $OutputDirectory,
    [Parameter(Mandatory = $true)][string] $MarkitectExecutable,
    [ValidateSet('all', 'amendment', 'veto-repair', 'persistent-veto', 'self-authorization', 'missing-mandate', 'stale-vote', 'independent')][string] $Case = 'all'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-ExternalProcess {
    param([string] $Executable, [string[]] $Arguments, [string] $WorkingDirectory, [string] $StdoutPath, [string] $StderrPath)
    $start = [System.Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Executable; $start.WorkingDirectory = $WorkingDirectory
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true
    foreach ($argument in $Arguments) { $start.ArgumentList.Add($argument) }
    $stdout = [System.IO.File]::Create($StdoutPath); $stderr = [System.IO.File]::Create($StderrPath)
    $process = [System.Diagnostics.Process]::new(); $process.StartInfo = $start
    try {
        if (-not $process.Start()) { throw "Could not start $Executable" }
        $outCopy = $process.StandardOutput.BaseStream.CopyToAsync($stdout)
        $errCopy = $process.StandardError.BaseStream.CopyToAsync($stderr)
        if (-not $process.WaitForExit(1800000)) {
            try { $process.Kill($true) } catch { }
            $process.WaitForExit()
            [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]] @($outCopy, $errCopy))
            $stdout.Flush($true); $stderr.Flush($true)
            return 124
        }
        [System.Threading.Tasks.Task]::WaitAll([System.Threading.Tasks.Task[]] @($outCopy, $errCopy))
        $stdout.Flush($true); $stderr.Flush($true)
        return $process.ExitCode
    } finally { $stdout.Dispose(); $stderr.Dispose(); $process.Dispose() }
}

function Invoke-Git {
    param([string[]] $Arguments)
    $lines = & git -C $script:Repository @Arguments
    if ($LASTEXITCODE -ne 0) { throw "git $($Arguments -join ' ') failed with exit $LASTEXITCODE" }
    return (($lines -join "`n").Trim())
}

function Get-Property {
    param([AllowNull()][object] $Value, [string] $Name)
    if ($null -eq $Value) { return }
    $property = $Value.PSObject.Properties[$Name]
    if ($null -eq $property -or $null -eq $property.Value) { return }
    return $property.Value
}

function Write-Utf8 {
    param([string] $Path, [string] $Text)
    [System.IO.File]::WriteAllText($Path, $Text, [System.Text.UTF8Encoding]::new($false))
}

function New-RunnerSpec {
    param([string] $Slot, [string] $Mode, [string[]] $Arguments = @())
    return @{ slotId = $Slot; command = $script:RunnerExecutable; args = @($Mode) + $Arguments
        model = 'deterministic-g4-model-amendment-mechanics'; modelOptions = @{}; providerVersion = 'fixture-v1'
        timeoutSeconds = 120; maxStdoutBytes = 1048576; maxStderrBytes = 1048576; runtimeFiles = @() }
}

function Get-ModelDigest {
    $stdout = Join-Path $script:TrialRoot 'inspect.stdout.yaml'
    $stderr = Join-Path $script:TrialRoot 'inspect.stderr.txt'
    $exitCode = Invoke-ExternalProcess $script:Markitect @('government', '--action', 'inspect', '--repo', $script:Repository, '--config', 'government.yaml') `
        $script:Checkout $stdout $stderr
    $match = [regex]::Match([System.IO.File]::ReadAllText($stdout), '(?ms)^model:\s*\r?\n\s+digest:\s+(sha256:[0-9a-f]{64})')
    if (-not $match.Success) { throw "Could not read the prior model digest from inspect exit $exitCode; see $stdout and $stderr" }
    return $match.Groups[1].Value
}

function New-OrderFile {
    param([string] $Path, [string] $Digest, [string] $Action, [string] $Namespace, [string] $Name, [string] $Purpose)
    $order = [ordered]@{ apiVersion = 'markitect.government-order/v1alpha1'; kind = 'Order'; purpose = $Purpose
        activeConstitution = $Digest; action = $Action
        subjects = @(@{ apiVersion = 'markitect.government-example/v1alpha1'; kind = 'Requirement'; namespace = $Namespace; name = $Name }) }
    Write-Utf8 $Path (ConvertTo-Json -InputObject $order -Depth 16)
}

function New-Runtime {
    param([string] $OrderName, [string] $ExecutorMode, [switch] $Amend, [string[]] $StaleVoteArguments = @())
    $id = @{ apiVersion = 'markitect.government/v1alpha1'; kind = 'Ressort'; namespace = 'invoice' }
    $arithmetic = $id + @{ name = 'arithmetic' }; $unaffected = $id + @{ name = 'unaffected' }
    $arithmeticMode = if ($StaleVoteArguments.Count) { 'stale-assent' } elseif ($ExecutorMode -eq 'veto-repair') { 'object-once' } elseif ($ExecutorMode -eq 'persistent-veto') { 'persistent-objection' } elseif ($ExecutorMode -eq 'independent') { 'assent-unaffected' } else { 'assent' }
    $unaffectedMode = if ($StaleVoteArguments.Count) { 'stale-assent' } elseif ($ExecutorMode -eq 'independent') { 'assent' } else { 'assent-unaffected' }
    $runtime = [ordered]@{
        apiVersion = 'markitect.government-execution/v1alpha1'; activeRef = $script:ActiveRef; expectedBase = $script:Base
        timeoutSeconds = 1800; stateDirectory = $script:StateDirectory; temporaryDirectory = $script:TemporaryDirectory
        executor = (New-RunnerSpec 'root-executor' $ExecutorMode)
        verifier = (New-RunnerSpec 'root-review' $(if ($ExecutorMode -eq 'independent') { 'review-independent' } else { 'review' }))
        ressorts = @(
            @{ ressort = $arithmetic; runner = (New-RunnerSpec 'ressort-arithmetic' $arithmeticMode $StaleVoteArguments) },
            @{ ressort = $unaffected; runner = (New-RunnerSpec 'ressort-unaffected' $unaffectedMode $StaleVoteArguments) }
        )
        checks = @(@{ name = 'model-realization'; run = @('go', 'test', './...', '-count=1', '-timeout=20m') })
    }
    if ($Amend) { $runtime.amendment = @{ maxRepairs = 1 } }
    $runtimePath = Join-Path $script:TrialRoot ('runtime-' + [IO.Path]::GetFileNameWithoutExtension($OrderName) + '.json')
    Write-Utf8 $runtimePath (ConvertTo-Json -InputObject $runtime -Depth 32)
    return $runtimePath
}

function Invoke-Run {
    param([string] $CaseName, [string] $OrderName, [string] $ExecutorMode, [switch] $Amend, [string[]] $StaleVoteArguments = @())
    $runtimePath = New-Runtime $OrderName $ExecutorMode -Amend:$Amend -StaleVoteArguments $StaleVoteArguments
    $stdout = Join-Path $script:TrialRoot ($CaseName + '.markitect.stdout.json')
    $stderr = Join-Path $script:TrialRoot ($CaseName + '.markitect.stderr.txt')
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $exitCode = Invoke-ExternalProcess $script:Markitect @('government', '--action', 'run', '--repo', $script:Repository, '--config', 'government.yaml', '--order', $OrderName, '--runtime', $runtimePath, '--write') `
        $script:Checkout $stdout $stderr
    $watch.Stop()
    $report = $null; $parseError = $null
    try { $report = Get-Content -LiteralPath $stdout -Raw -Encoding UTF8 | ConvertFrom-Json } catch { $parseError = $_.Exception.Message }
    return [pscustomobject]@{ exitCode = $exitCode; elapsedMilliseconds = $watch.ElapsedMilliseconds; report = $report; parseError = $parseError
        runtimePath = $runtimePath; stdoutPath = $stdout; stderrPath = $stderr; stdoutSha256 = 'sha256:' + (Get-FileHash $stdout -Algorithm SHA256).Hash.ToLowerInvariant()
        stderrSha256 = 'sha256:' + (Get-FileHash $stderr -Algorithm SHA256).Hash.ToLowerInvariant() }
}

function New-Repo {
    param([string] $Name, [string] $SeedCandidateWorkspace = '')
    $script:TrialRoot = Join-Path $script:ResolvedOutput ('government-g4-' + $Name + '-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $script:TrialRoot | Out-Null
    $script:Repository = Join-Path $script:TrialRoot 'repo'; $script:BinDirectory = Join-Path $script:TrialRoot 'bin'
    $script:StateDirectory = Join-Path $script:TrialRoot 'state'; $script:TemporaryDirectory = Join-Path $script:TrialRoot 'temporary'
    foreach ($dir in @($script:Repository, $script:BinDirectory, $script:StateDirectory, $script:TemporaryDirectory)) { New-Item -ItemType Directory -Path $dir | Out-Null }
    $paths = @('README.md','government.yaml','go.mod','trial.ps1','runner/main.go','delivery/rate.txt','invoice/invoice.go','invoice/invoice_test.go','invoice/protected-goal.txt','independent/receipt.txt')
    foreach ($relative in $paths) {
        $destination = Join-Path $script:Repository ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))
        New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
        if ($SeedCandidateWorkspace -and $relative -in @('government.yaml','invoice/invoice.go','invoice/invoice_test.go')) {
            Copy-Item -LiteralPath (Join-Path $SeedCandidateWorkspace ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))) -Destination $destination
        } else {
            Copy-Item -LiteralPath (Join-Path $script:Fixture ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))) -Destination $destination
        }
    }
    if ($Name -eq 'missing-mandate') {
        $sourcePath = Join-Path $script:Repository 'government.yaml'
        $source = Get-Content -LiteralPath $sourcePath -Raw | ConvertFrom-Json -AsHashtable
        foreach ($definition in $source.definitions) {
            if ($definition.kind -eq 'Mandate' -and $definition.metadata.name -eq 'root-prior') {
                $definition.spec.actions = @('implement','review')
            }
        }
        Write-Utf8 $sourcePath (ConvertTo-Json -InputObject $source -Depth 64)
    }
    $script:FixtureModelDigest = Get-ModelDigest
    $subject = 'delivery-rule'
    $purpose = if ($Name -in @('veto-repair','persistent-veto')) { 'Make the invoice delivery charge explicit while preserving the prior order purpose: delivery/rate.txt remains exactly charge=2.' } else { "G4 $Name native fixture order." }
    New-OrderFile (Join-Path $script:Repository 'order.yaml') $script:FixtureModelDigest 'amend-model' 'invoice' $subject $purpose
    New-OrderFile (Join-Path $script:Repository 'conflict-order.yaml') $script:FixtureModelDigest 'amend-model' 'invoice' 'protected-total' 'Attempt to redefine a protected customer amount goal.'
    New-OrderFile (Join-Path $script:Repository 'independent-order.yaml') $script:FixtureModelDigest 'implement' 'receipt' 'receipt-format' 'Change only the disjoint plain receipt label.'
    $script:Checkout = $script:Repository
    & git init -b main $script:Repository | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'git init failed.' }
    Invoke-Git @('config','user.name','Government G4 mechanics fixture') | Out-Null
    Invoke-Git @('config','user.email','government-g4@example.invalid') | Out-Null
    Invoke-Git @('config','core.autocrlf','false') | Out-Null
    Invoke-Git @('add','--all') | Out-Null
    Invoke-Git @('-c','commit.gpgsign=false','commit','-m','G4 prior active fixture') | Out-Null
    $script:Base = Invoke-Git @('rev-parse','--verify','HEAD^{commit}')
    $script:ActiveRef = 'refs/markitect/government/active/g4-example'
    Invoke-Git @('update-ref',$script:ActiveRef,$script:Base) | Out-Null
    $script:RunnerExecutable = Join-Path $script:BinDirectory 'government-g4-runner.exe'
    $buildOut = Join-Path $script:TrialRoot 'runner-build.stdout.txt'; $buildErr = Join-Path $script:TrialRoot 'runner-build.stderr.txt'
    $buildExit = Invoke-ExternalProcess 'go' @('build','-o',$script:RunnerExecutable,'./runner') $script:Fixture $buildOut $buildErr
    if ($buildExit -ne 0) { throw "Runner build failed with exit $buildExit; see $buildOut and $buildErr" }
}

function Complete-Case {
    param([string] $CaseName, [object[]] $Runs, [bool] $Expected, [string] $ExpectedOutcome)
    $finalHead = Invoke-Git @('rev-parse','--verify','HEAD^{commit}')
    $active = Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}")
    $reports = @($Runs | ForEach-Object { $_.report })
    $summary = [ordered]@{ case = $CaseName; status = if ($Expected) { 'passed' } else { 'failed' }; expectedOutcome = $ExpectedOutcome
        sourceCheckout = $script:Repository; base = $script:Base; activeRef = $script:ActiveRef; activeRevision = $active; finalHead = $finalHead
        markitectExecutable = $script:Markitect; markitectBinarySha256 = 'sha256:' + (Get-FileHash $script:Markitect -Algorithm SHA256).Hash.ToLowerInvariant()
        runnerExecutable = $script:RunnerExecutable; runnerBinarySha256 = 'sha256:' + (Get-FileHash $script:RunnerExecutable -Algorithm SHA256).Hash.ToLowerInvariant()
        priorModelDigest = Get-Property $reports[0] 'priorModelDigest'; invocations = @($Runs | ForEach-Object { [ordered]@{ exitCode = $_.exitCode; elapsedMilliseconds = $_.elapsedMilliseconds; parseError = $_.parseError
            runtimePath = $_.runtimePath; stdoutPath = $_.stdoutPath; stdoutSha256 = $_.stdoutSha256; stderrPath = $_.stderrPath; stderrSha256 = $_.stderrSha256; report = $_.report } })
        trialDirectory = $script:TrialRoot }
    $summaryPath = Join-Path $script:TrialRoot 'summary.json'
    $summary['summaryPath'] = $summaryPath
    Write-Utf8 $summaryPath (ConvertTo-Json -InputObject $summary -Depth 64)
    if (-not $Expected) { throw "Government G4 $CaseName did not meet its expected outcome; inspect the retained native records in $script:TrialRoot" }
    return ,$summary
}

if (-not [IO.Path]::IsPathRooted($OutputDirectory) -or -not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) { throw 'OutputDirectory must be an absolute path to an existing directory.' }
if (-not [IO.Path]::IsPathRooted($MarkitectExecutable) -or -not (Test-Path -LiteralPath $MarkitectExecutable -PathType Leaf)) { throw 'MarkitectExecutable must be an absolute source-built executable path.' }
$script:ResolvedOutput = (Resolve-Path -LiteralPath $OutputDirectory).Path
$script:Markitect = (Resolve-Path -LiteralPath $MarkitectExecutable).Path
$script:Fixture = [IO.Path]::GetFullPath($PSScriptRoot)
$checkout = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$prefix = $checkout.TrimEnd([char[]] @('\','/')) + [IO.Path]::DirectorySeparatorChar
if ($script:ResolvedOutput.Equals($checkout,[StringComparison]::OrdinalIgnoreCase) -or $script:ResolvedOutput.StartsWith($prefix,[StringComparison]::OrdinalIgnoreCase)) { throw 'OutputDirectory must be outside the Markitect checkout.' }

$summaries = @()
$cases = if ($Case -eq 'all') { @('amendment','veto-repair','persistent-veto','self-authorization','missing-mandate','stale-vote','independent') } else { @($Case) }
$accepted = $null
foreach ($caseName in $cases) {
    if ($caseName -eq 'stale-vote' -and $null -eq $accepted) { throw 'stale-vote requires -Case all so the script can bind prior votes from its actual accepted native amendment run.' }
    $seedWorkspace = if ($caseName -eq 'stale-vote') { Get-Property $accepted.report 'workspace' } else { '' }
    New-Repo $caseName $seedWorkspace
    if ($caseName -eq 'independent') {
        $blocked = Invoke-Run 'conflict' 'conflict-order.yaml' 'self-authorize' -Amend
        $activeAfterConflict = Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}")
        $independent = Invoke-Run 'independent' 'independent-order.yaml' 'independent'
        $blockedReport = $blocked.report; $independentReport = $independent.report
        $escalations = @(Get-Property $blockedReport 'escalations')
        $activeAfter = Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}")
        $good = ($blocked.exitCode -ne 0 -and (Get-Property $blockedReport 'status') -in @('blocked','blocked-escalation-required') -and $escalations.Count -gt 0 -and
            $independent.exitCode -eq 0 -and (Get-Property $independentReport 'status') -eq 'accepted-scoped' -and $blockedReport.baseRevision -eq $independentReport.baseRevision -and
            $blockedReport.priorModelDigest -eq $independentReport.priorModelDigest -and $activeAfterConflict -eq $script:Base -and $activeAfter -eq $independentReport.candidateCommit -and $independentReport.changedPaths -contains 'independent/receipt.txt' -and
            $independentReport.changedPaths -notcontains 'government.yaml' -and $null -eq (Get-Property $blockedReport 'promotion'))
        $runs = @($blocked,$independent)
        $summary = Complete-Case $caseName $runs $good 'blocked invoice conflict followed by disjoint order success under the same unchanged prior law'
        $summaries += $summary
        continue
    }
    if ($caseName -eq 'stale-vote') {
        $detail = $null
        foreach ($actor in @(Get-Property $accepted.report 'actors')) {
            $response = Get-Property (Get-Property $actor 'result') 'response'
            foreach ($observation in @(Get-Property $response 'verifierObservations')) {
                if ((Get-Property $observation 'subject') -eq 'government-vote') { $detail = Get-Property $observation 'detail'; break }
            }
            if ($detail) { break }
        }
        if (-not $detail) { throw 'Accepted native report did not contain an actual final vote body.' }
        $old = $detail | ConvertFrom-Json
        $voteFile = Join-Path $script:TrialRoot 'prior-accepted-vote.json'
        Write-Utf8 $voteFile $detail
        $oldArgs = @($old.materialCandidateId,$old.evidenceId,[string]$old.round)
        $script:Base = Invoke-Git @('rev-parse','--verify','HEAD^{commit}')
        $script:ActiveRef = 'refs/markitect/government/active/g4-example'
        $run = Invoke-Run 'stale-vote' 'order.yaml' 'propose' -Amend -StaleVoteArguments $oldArgs
        $report = $run.report
        $errorText = [string](Get-Property $report 'error')
        $promotion = Get-Property $report 'promotion'
        $candidate = Get-Property $report 'candidate'; $evidence = Get-Property $report 'evidence'
        $good = ($run.exitCode -ne 0 -and $null -eq $run.parseError -and $errorText -match 'stale or foreign candidate/evidence/round' -and
            (Get-Property $report 'status') -ne 'accepted-scoped' -and $null -eq $promotion -and $script:Base -eq (Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}")) -and
            $candidate.id -ne $old.materialCandidateId -and $evidence.id -ne $old.evidenceId -and @(Get-Property $report 'votes').Count -eq 0)
        $summaries += Complete-Case $caseName @($run) $good 'old actual Ressort vote IDs rejected after candidate and evidence changed'
        continue
    }
    $mode = if ($caseName -eq 'self-authorization') { 'self-authorize' } elseif ($caseName -eq 'missing-mandate') { 'missing-mandate' } else { 'propose' }
    if ($caseName -eq 'veto-repair') { $mode = 'veto-repair' }
    if ($caseName -eq 'persistent-veto') { $mode = 'persistent-veto' }
    $amendment = $caseName -ne 'independent'
    $run = Invoke-Run $caseName 'order.yaml' $mode -Amend:$amendment
    $report = $run.report
    if ($caseName -in @('veto-repair','persistent-veto')) {
        $rounds = @(Get-Property $report 'amendmentRounds')
        $first = if ($rounds.Count -gt 0) { $rounds[0] } else { $null }
        $last = if ($rounds.Count -gt 1) { $rounds[-1] } else { $null }
        $firstVotes = @(Get-Property $first 'votes'); $lastVotes = @(Get-Property $last 'votes')
        $allVoteCounts = ($rounds.Count -eq 2 -and @($rounds | Where-Object { @(Get-Property $_ 'votes').Count -ne 2 }).Count -eq 0)
        $firstObjection = @($firstVotes | Where-Object { (Get-Property $_ 'outcome') -eq 'objection' -or (Get-Property $_ 'decision') -eq 'objection' }).Count -gt 0
        $candidateIDs = @($rounds | ForEach-Object { Get-Property (Get-Property $_ 'candidate') 'id' })
        $evidenceIDs = @($rounds | ForEach-Object { Get-Property (Get-Property $_ 'evidence') 'id' })
        $freshBindings = ($candidateIDs.Count -eq 2 -and $candidateIDs[0] -and $candidateIDs[1] -and $candidateIDs[0] -ne $candidateIDs[1] -and
            $evidenceIDs.Count -eq 2 -and $evidenceIDs[0] -and $evidenceIDs[1] -and $evidenceIDs[0] -ne $evidenceIDs[1])
        $baseUnchanged = $script:Base -eq (Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}"))
        if ($caseName -eq 'veto-repair') {
            $finalAssents = ($lastVotes.Count -eq 2 -and @($lastVotes | Where-Object { (Get-Property $_ 'outcome') -in @('assent','assent-unaffected') -or (Get-Property $_ 'decision') -in @('assent','assent-unaffected') }).Count -eq 2)
            $good = ($run.exitCode -eq 0 -and $null -eq $run.parseError -and (Get-Property $report 'status') -eq 'accepted-scoped' -and
                (Get-Property (Get-Property $report 'promotion') 'status') -eq 'promoted' -and $rounds.Count -eq 2 -and $allVoteCounts -and $firstObjection -and
                $finalAssents -and $freshBindings -and @(Get-Property $last 'checks').Count -gt 0 -and @(Get-Property $last 'reviewActorSequences').Count -gt 0 -and
                (Get-Property $report 'proposedModelDigest') -ne (Get-Property $report 'priorModelDigest'))
            $outcome = 'first current-candidate Ressort objection triggers one substantive repair, fresh all-cabinet assent, and promotion'
        } else {
            $lastObjection = @($lastVotes | Where-Object { (Get-Property $_ 'outcome') -eq 'objection' -or (Get-Property $_ 'decision') -eq 'objection' }).Count -gt 0
            $escalations = @(Get-Property $report 'escalations')
            # Durable Host state layout can vary; the report itself must retain both rounds.
            $concreteEscalation = @($escalations | Where-Object { (Get-Property $_ 'promotionAttempted') -eq $false -and (Get-Property $_ 'requiredDecision') }).Count -gt 0
            $good = ($run.exitCode -ne 0 -and $null -eq $run.parseError -and (Get-Property $report 'status') -in @('blocked','blocked-escalation-required') -and
                $rounds.Count -eq 2 -and $allVoteCounts -and $firstObjection -and $lastObjection -and $freshBindings -and $escalations.Count -gt 0 -and
                $concreteEscalation -and $null -eq (Get-Property $report 'promotion') -and $baseUnchanged -and $null -ne (Get-Property $last 'error'))
            $outcome = 'bounded repair limit retains two vetoed candidate rounds, escalates concretely, and does not promote'
        }
        $summaries += Complete-Case $caseName @($run) $good $outcome
    } elseif ($caseName -eq 'amendment') {
        $rounds = @(Get-Property $report 'amendmentRounds'); $last = if ($rounds.Count) { $rounds[$rounds.Count - 1] } else { $null }
        $votes = @(Get-Property $report 'votes'); $candidate = Get-Property $report 'candidate'
        $good = ($run.exitCode -eq 0 -and $null -eq $run.parseError -and (Get-Property $report 'status') -eq 'accepted-scoped' -and
            (Get-Property (Get-Property $report 'promotion') 'status') -eq 'promoted' -and @(Get-Property $report 'changedPaths') -contains 'government.yaml' -and
            @(Get-Property $report 'changedPaths') -contains 'invoice/invoice.go' -and @(Get-Property $report 'changedPaths') -contains 'invoice/invoice_test.go' -and
            $votes.Count -eq 2 -and @(Get-Property $last 'checks').Count -gt 0 -and @(Get-Property $last 'reviewActorSequences').Count -gt 0 -and
            @(Get-Property $last 'votes').Count -eq 2 -and (Get-Property $report 'proposedModelDigest') -ne (Get-Property $report 'priorModelDigest') -and
            (Get-Property $report 'activeRef') -eq $script:ActiveRef)
        $summary = Complete-Case $caseName @($run) $good 'substantive model plus realization proposal passed fresh checks review and unanimous frozen Ressort votes'
        $accepted = $run
        $summaries += $summary
    } else {
        $assessment = Get-Property $report 'amendmentAssessment'; $escalations = @(Get-Property $report 'escalations')
        $errorText = [string](Get-Property $report 'error')
        $expectedBlock = (Get-Property $report 'status') -in @('blocked','blocked-escalation-required')
        $actors = @(Get-Property $report 'actors')
        if ($caseName -eq 'self-authorization') {
            $executor = @($actors | Where-Object { (Get-Property $_ 'slotId') -eq 'root-executor' } | Select-Object -First 1)
            $executorResponse = if ($executor.Count) { Get-Property (Get-Property $executor[0] 'result') 'response' } else { $null }
            $executorUncertainty = @(Get-Property $executorResponse 'uncertainty')
            $findings = @(Get-Property $assessment 'findings')
            $ownerClaim = @($executorUncertainty | Where-Object { $_ -match 'Owner approved' }).Count -gt 0
            $protectedFinding = @($findings | Where-Object { (Get-Property $_ 'code') -eq 'amendment.protected-subject' }).Count -gt 0
            $expectedBlock = $expectedBlock -and (Get-Property $assessment 'status') -eq 'blocked-escalation-required' -and $actors.Count -ge 1 -and $ownerClaim -and $protectedFinding
        }
        if ($caseName -eq 'missing-mandate') {
            $expectedBlock = $expectedBlock -and $actors.Count -eq 0 -and $errorText -match 'amendment prior-authority plan is blocked'
        }
        $good = ($run.exitCode -ne 0 -and $null -eq $run.parseError -and $expectedBlock -and $escalations.Count -gt 0 -and $null -eq (Get-Property $report 'promotion') -and
            $script:Base -eq (Invoke-Git @('rev-parse','--verify',"$script:ActiveRef^{commit}")) -and $errorText.Length -gt 0)
        $summaries += Complete-Case $caseName @($run) $good 'prior authority blocks self-authorization or missing amend-model delegation with a concrete Owner escalation'
    }
}
foreach ($summary in $summaries) {
    Write-Output (ConvertTo-Json -InputObject $summary -Depth 64)
    Write-Output "Summary: $($summary.summaryPath)"
}
if ($Case -eq 'all') { Write-Output ('All G4 cases: ' + (($summaries | ForEach-Object { $_.status }) -join ', ')) }
