[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $CandidateExecutable,

    [Parameter(Mandatory = $true)]
    [string] $CandidateSourceCommit,

    [Parameter(Mandatory = $true)]
    [string] $ExpectedCandidateSha256,

    [string] $PublicV012Executable
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$bundlePath = Join-Path $repoRoot 'experiments/real-project-adoption/evidence/adopter-snapshots.bundle'
$bundleSha256 = 'af9dbdc750f376b5250b61a451c1d124fe363fd87cce87684804b292149311cc'
$publicV012Sha256 = 'b03424560caa460322e3580785d6abc9dfbf2137df878e03ddedcf76b3a8fa47'
$packageHashes = @{
    v1 = '0c7395055aac7f4c35afcff404481a77e2ce6475c863a435afc2586dd388b1cc'
    v2 = '0ed97c9f0b21e10b5e2a0ff232bdb910bd468ac04f84441ae24d5b7cefb2b504'
}
$revisions = [ordered]@{
    v1 = 'fd94a689aab857704c3a4a45ac2abe0fc0e3185c'
    failingV2 = 'a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f'
    firstValidator = '3e138c146736dad1ddb11dc7c509a61d95ab4398'
    waived = '65c1d959e6f5d1b866effbc5188c353c82b2405d'
    final = 'c507172b24da9005904422c03cc3d66a2b5efcec'
}
$apiVersion = 'architecture.mymeetings.example/v1alpha1'
$encoding = [System.Text.UTF8Encoding]::new($false)

function Get-Sha256([string] $Path) {
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Invoke-Native([string] $FileName, [string[]] $Arguments) {
    $start = [System.Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $FileName
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = [System.Text.UTF8Encoding]::new($false)
    $start.StandardErrorEncoding = [System.Text.UTF8Encoding]::new($false)
    foreach ($argument in $Arguments) { [void] $start.ArgumentList.Add($argument) }

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $start
    if (-not $process.Start()) { throw "Could not start $FileName" }
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    $process.WaitForExit()
    return [pscustomobject]@{
        ExitCode = $process.ExitCode
        Stdout = $stdoutTask.GetAwaiter().GetResult()
        Stderr = $stderrTask.GetAwaiter().GetResult()
    }
}

function Invoke-Git([string[]] $Arguments) {
    $result = Invoke-Native 'git' $Arguments
    if ($result.ExitCode -ne 0) {
        throw "git $($Arguments -join ' ') failed ($($result.ExitCode)): $($result.Stderr) $($result.Stdout)"
    }
    return $result.Stdout.Trim()
}

function Get-CheckoutStatus([string] $Checkout) {
    return Invoke-Git @('-C', $Checkout, 'status', '--porcelain=v1', '--untracked-files=all')
}

function Get-CheckoutFileFingerprint([string] $Checkout) {
    # Include ignored projections and any other regular files. Only Git metadata
    # is excluded; record aggregate evidence, never per-file content or hashes.
    $root = (Resolve-Path -LiteralPath $Checkout).Path
    $records = [Collections.Generic.List[string]]::new()
    foreach ($file in Get-ChildItem -LiteralPath $root -Recurse -File -Force) {
        $relative = [IO.Path]::GetRelativePath($root, $file.FullName).Replace('\', '/')
        if ($relative -match '(?i)^\.git(/|$)') { continue }
        $hash = Get-Sha256 $file.FullName
        $records.Add("$relative`0$($file.Length)`0$hash")
    }
    $sorted = @($records | Sort-Object)
    $bytes = $script:encoding.GetBytes(($sorted -join "`n"))
    $digest = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($bytes)).ToLowerInvariant()
    return [pscustomobject]@{ Sha256 = $digest; FileCount = $sorted.Count }
}

function Assert-CleanCheckout([string] $Checkout, [string] $Operation) {
    $status = Get-CheckoutStatus $Checkout
    if (-not [string]::IsNullOrWhiteSpace($status)) {
        throw "$Operation changed or created files in the replay checkout:`n$status"
    }
}

function Set-FixedRevision([string] $Checkout, [string] $Revision) {
    [void] (Invoke-Git @('-C', $Checkout, 'checkout', '--detach', '--force', $Revision))
    Assert-CleanCheckout $Checkout "Checking out $Revision"
    return Invoke-Git @('-C', $Checkout, 'rev-parse', 'HEAD^{tree}')
}

function Get-TopLevelYamlSection([string] $Text, [string] $SectionName) {
    $lines = $Text -split "`r?`n"
    $start = -1
    for ($index = 0; $index -lt $lines.Count; $index++) {
        if ($lines[$index] -match "^$([regex]::Escape($SectionName)):\s*(?:#.*)?$") {
            $start = $index
            break
        }
    }
    if ($start -lt 0) { throw "Top-level YAML section '$SectionName' was not present." }
    $end = $lines.Count
    for ($index = $start + 1; $index -lt $lines.Count; $index++) {
        if ($lines[$index] -match '^[^\s#][^:]*:\s*(?:#.*)?$') {
            $end = $index
            break
        }
    }
    return ($lines[$start..($end - 1)] -join "`n")
}

function Save-Command([string] $Name, [string] $Revision, [string[]] $Arguments, [int] $ExpectedExitCode) {
    $treeBefore = Set-FixedRevision $script:checkout $Revision
    $statusBefore = Get-CheckoutStatus $script:checkout
    $filesBefore = Get-CheckoutFileFingerprint $script:checkout
    $result = Invoke-Native $script:candidateExe $Arguments
    $treeAfter = Invoke-Git @('-C', $script:checkout, 'rev-parse', 'HEAD^{tree}')
    $statusAfter = Get-CheckoutStatus $script:checkout
    $filesAfter = Get-CheckoutFileFingerprint $script:checkout

    $prefix = Join-Path $script:runDirectory $Name
    [IO.File]::WriteAllText("$prefix.stdout.txt", $result.Stdout, $script:encoding)
    [IO.File]::WriteAllText("$prefix.stderr.txt", $result.Stderr, $script:encoding)
    [IO.File]::WriteAllText("$prefix.exit.txt", "$($result.ExitCode)`n", $script:encoding)

    if ($treeBefore -ne $treeAfter -or $statusBefore -ne $statusAfter -or
        $filesBefore.Sha256 -ne $filesAfter.Sha256 -or $filesBefore.FileCount -ne $filesAfter.FileCount) {
        throw "$Name changed the replay checkout (tree/status/file fingerprint differs before and after)."
    }
    if ($result.ExitCode -ne $ExpectedExitCode) {
        throw "$Name exited $($result.ExitCode); expected $ExpectedExitCode. Captured output: $prefix.stdout.txt"
    }
    $record = [ordered]@{
        name = $Name
        revision = $Revision
        arguments = $Arguments
        expectedExitCode = $ExpectedExitCode
        actualExitCode = $result.ExitCode
        stdoutSha256 = Get-Sha256 "$prefix.stdout.txt"
        stderrSha256 = Get-Sha256 "$prefix.stderr.txt"
        checkoutTreeBefore = $treeBefore
        checkoutTreeAfter = $treeAfter
        checkoutFilesBeforeSha256 = $filesBefore.Sha256
        checkoutFilesAfterSha256 = $filesAfter.Sha256
        checkoutFileCount = $filesBefore.FileCount
        checkoutCleanBefore = [string]::IsNullOrWhiteSpace($statusBefore)
        checkoutCleanAfter = [string]::IsNullOrWhiteSpace($statusAfter)
    }
    $script:records.Add($record)
    return $result.Stdout
}

function Assert-Contains([string] $Text, [string] $Pattern, [string] $Label) {
    if ($Text -notmatch $Pattern) { throw "$Label did not contain expected pattern: $Pattern" }
}

function Assert-Count([string] $Text, [string] $Pattern, [int] $Expected, [string] $Label) {
    $actual = [regex]::Matches($Text, $Pattern).Count
    if ($actual -ne $Expected) { throw "$Label contained $actual matches for '$Pattern'; expected $Expected." }
}

if ($CandidateSourceCommit -notmatch '^[0-9a-f]{40}$') { throw 'CandidateSourceCommit must be a full lowercase Git SHA-1.' }
if ($ExpectedCandidateSha256 -notmatch '^[0-9a-f]{64}$') { throw 'ExpectedCandidateSha256 must be 64 lowercase hex characters.' }
[void] (Invoke-Git @('-C', $repoRoot, 'cat-file', '-e', "$CandidateSourceCommit^{commit}"))
if (-not (Test-Path -LiteralPath $bundlePath -PathType Leaf)) { throw "Missing preserved bundle: $bundlePath" }
if ((Get-Sha256 $bundlePath) -ne $bundleSha256) { throw 'Preserved adopter bundle hash differs from the recorded pilot artifact.' }
if (-not (Test-Path -LiteralPath $CandidateExecutable -PathType Leaf)) { throw "Missing candidate executable: $CandidateExecutable" }
$candidateExe = (Resolve-Path -LiteralPath $CandidateExecutable).Path
if ((Get-Sha256 $candidateExe) -ne $ExpectedCandidateSha256) { throw 'Candidate executable SHA-256 does not match the supplied expected digest.' }

$artifactRoot = Join-Path $repoRoot '.artifacts'
$ignoredCheck = Invoke-Git @('-C', $repoRoot, 'check-ignore', '.artifacts/policy-failure-analysis-probe')
if ([string]::IsNullOrWhiteSpace($ignoredCheck)) { throw '.artifacts must be ignored before replay setup.' }

$runId = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ')
$runDirectory = Join-Path $PSScriptRoot "results/$runId"
$runDirectory = [IO.Path]::GetFullPath($runDirectory)
if (Test-Path -LiteralPath $runDirectory) { throw "Refusing existing evidence path: $runDirectory" }
$workRoot = Join-Path $artifactRoot "policy-failure-analysis/$runId"
$checkout = Join-Path $workRoot 'adopter'
if (Test-Path -LiteralPath $workRoot) { throw "Refusing existing replay work path: $workRoot" }
New-Item -ItemType Directory -Path $runDirectory -Force | Out-Null
New-Item -ItemType Directory -Path (Split-Path -Parent $workRoot) -Force | Out-Null

$script:runDirectory = $runDirectory
$script:checkout = $checkout
$script:candidateExe = $candidateExe
$script:records = [Collections.Generic.List[object]]::new()

try {
    [void] (Invoke-Git @('clone', '--quiet', $bundlePath, $checkout))
    foreach ($revision in $revisions.Values) {
        [void] (Invoke-Git @('-C', $checkout, 'cat-file', '-e', "$revision^{commit}"))
    }

    # Prove both package archives are committed inputs at every fixed state.
    foreach ($revision in $revisions.Values) {
        [void] (Invoke-Git @('-C', $checkout, 'checkout', '--detach', '--force', $revision))
        foreach ($version in @('v1', 'v2')) {
            $archive = Join-Path $checkout ".markitect/packages/mymeetings-architecture-$($version.Substring(1)).0.0.zip"
            if (-not (Test-Path -LiteralPath $archive -PathType Leaf)) { throw "$revision is missing committed $version package archive." }
            if ((Get-Sha256 $archive) -ne $packageHashes[$version]) { throw "$revision has an unexpected $version archive hash." }
        }
        Assert-CleanCheckout $checkout "Validating package archives at $revision"
    }

    # The failing pin is analyzed with no waiver and no edits to canonical state.
    $failingTree = Set-FixedRevision $checkout $revisions.failingV2
    $v1Tree = Invoke-Git @('-C', $checkout, 'rev-parse', "$($revisions.v1)^{tree}")
    $changedPaths = Invoke-Git @('-C', $checkout, 'diff', '--name-only', $revisions.v1, $revisions.failingV2)
    if ($changedPaths -match '(?im)(^|/)(exception|exceptions)(/|\.|$)') {
        throw 'The v1-to-failing-v2 pin transition unexpectedly changes an exception path.'
    }

    $v1Model = Save-Command 'v1-model' $revisions.v1 @('model', '--repo', $checkout, '--revision', $revisions.v1) 0
    Assert-Contains $v1Model '(?m)^\s*structuralStatus:\s*passed\s*$' 'v1 model'
    Assert-Contains $v1Model '(?m)^\s*policyStatus:\s*passed\s*$' 'v1 model'
    [void] (Save-Command 'v1-check-strict' $revisions.v1 @('check', '--repo', $checkout, '--revision', $revisions.v1) 0)

    $modelV2 = Save-Command 'v2-model' $revisions.failingV2 @('model', '--repo', $checkout, '--revision', $revisions.failingV2) 1
    Assert-Contains $modelV2 '(?m)^\s*structuralStatus:\s*passed\s*$' 'Failing v2 model'
    Assert-Contains $modelV2 '(?m)^\s*policyStatus:\s*failed\s*$' 'Failing v2 model'
    Assert-Contains $modelV2 'UseCase/add-meeting-attendee' 'Failing v2 model'
    Assert-Contains $modelV2 'UseCase/cancel-meeting' 'Failing v2 model'
    $v2PolicyResults = Get-TopLevelYamlSection $modelV2 'policyResults'
    Assert-Count $v2PolicyResults 'constraint: selected-commands-require-validator' 2 'Failing v2 policyResults'
    Assert-Count $v2PolicyResults '(?m)^\s+status:\s*failed\s*$' 2 'Failing v2 policyResults'
    if ($v2PolicyResults -match '(?m)^\s+status:\s*waived\s*$') { throw 'Failing v2 model unexpectedly contains a waived result.' }

    $checkV2 = Save-Command 'v2-check-strict' $revisions.failingV2 @('check', '--repo', $checkout, '--revision', $revisions.failingV2) 1
    $contextArgs = @('context', '--api-version', $apiVersion, '--kind', 'UseCase', '--name', 'add-meeting-attendee', '--namespace', 'architecture', '--repo', $checkout, '--revision', $revisions.failingV2)
    [void] (Save-Command 'v2-context-strict' $revisions.failingV2 $contextArgs 1)
    $impactArgs = @('impact', '--base', $revisions.v1, '--repo', $checkout, '--revision', $revisions.failingV2)
    [void] (Save-Command 'v1-to-v2-impact-strict' $revisions.failingV2 $impactArgs 1)

    $contextAnalysisArgs = $contextArgs + @('--analyze-policy-failures')
    $contextAnalysis = Save-Command 'v2-context-analysis' $revisions.failingV2 $contextAnalysisArgs 1
    Assert-Contains $contextAnalysis '(?m)^analysis:' 'Failing v2 diagnostic Context'
    Assert-Contains $contextAnalysis '(?m)^\s*structuralStatus:\s*passed\s*$' 'Failing v2 diagnostic Context'
    Assert-Contains $contextAnalysis '(?m)^\s*policyStatus:\s*failed\s*$' 'Failing v2 diagnostic Context'
    Assert-Contains $contextAnalysis 'selected-commands-require-validator' 'Failing v2 diagnostic Context'
    Assert-Contains $contextAnalysis 'UseCase/add-meeting-attendee' 'Failing v2 diagnostic Context'

    $impactAnalysisArgs = $impactArgs + @('--analyze-policy-failures')
    $impactAnalysis = Save-Command 'v1-to-v2-impact-analysis' $revisions.failingV2 $impactAnalysisArgs 1
    Assert-Contains $impactAnalysis '(?m)^analysis:' 'Failing v2 diagnostic Impact'
    Assert-Contains $impactAnalysis '(?m)^directPolicySubjectCount:\s*2\s*$' 'Failing v2 diagnostic Impact'
    Assert-Contains $impactAnalysis '(?m)^affectedCount:\s*[1-9][0-9]*\s*$' 'Failing v2 diagnostic Impact'
    Assert-Contains $impactAnalysis 'UseCase/add-meeting-attendee' 'Failing v2 diagnostic Impact'
    Assert-Contains $impactAnalysis 'UseCase/cancel-meeting' 'Failing v2 diagnostic Impact'
    Assert-Contains $impactAnalysis 'selected-commands-require-validator' 'Failing v2 diagnostic Impact'

    # One implementation is present; the second failing result must remain visible.
    $modelFirst = Save-Command 'first-validator-model' $revisions.firstValidator @('model', '--repo', $checkout, '--revision', $revisions.firstValidator) 1
    Assert-Contains $modelFirst 'UseCase/cancel-meeting' 'First Validator model'
    $firstPolicyResults = Get-TopLevelYamlSection $modelFirst 'policyResults'
    Assert-Count $firstPolicyResults '(?m)^\s+status:\s*failed\s*$' 1 'First Validator policyResults'
    $firstContextArgs = @('context', '--api-version', $apiVersion, '--kind', 'UseCase', '--name', 'cancel-meeting', '--namespace', 'architecture', '--repo', $checkout, '--revision', $revisions.firstValidator, '--analyze-policy-failures')
    $firstContext = Save-Command 'first-validator-context-analysis' $revisions.firstValidator $firstContextArgs 1
    Assert-Contains $firstContext 'selected-commands-require-validator' 'First Validator diagnostic Context'
    Assert-Contains $firstContext 'UseCase/cancel-meeting' 'First Validator diagnostic Context'

    # The historical, explicit waiver yields a waived policy state, not a failed one.
    $waivedModel = Save-Command 'waiver-model' $revisions.waived @('model', '--repo', $checkout, '--revision', $revisions.waived) 0
    Assert-Contains $waivedModel '(?m)^policyStatus:\s*waived\s*$' 'Waived model'
    $waivedContextArgs = @('context', '--api-version', $apiVersion, '--kind', 'UseCase', '--name', 'cancel-meeting', '--namespace', 'architecture', '--repo', $checkout, '--revision', $revisions.waived)
    [void] (Save-Command 'waiver-context-strict' $revisions.waived $waivedContextArgs 0)
    [void] (Save-Command 'waiver-context-analysis' $revisions.waived ($waivedContextArgs + @('--analyze-policy-failures')) 0)
    $waivedImpactArgs = @('impact', '--base', $revisions.v1, '--repo', $checkout, '--revision', $revisions.waived, '--analyze-policy-failures')
    [void] (Save-Command 'v1-to-waiver-impact-analysis' $revisions.waived $waivedImpactArgs 0)

    $finalModel = Save-Command 'final-model' $revisions.final @('model', '--repo', $checkout, '--revision', $revisions.final) 0
    Assert-Contains $finalModel '(?m)^\s*structuralStatus:\s*passed\s*$' 'Final model'
    Assert-Contains $finalModel '(?m)^\s*policyStatus:\s*passed\s*$' 'Final model'
    [void] (Save-Command 'final-check-strict' $revisions.final @('check', '--repo', $checkout, '--revision', $revisions.final) 0)
    $finalContextArgs = @('context', '--api-version', $apiVersion, '--kind', 'UseCase', '--name', 'add-meeting-attendee', '--namespace', 'architecture', '--repo', $checkout, '--revision', $revisions.final)
    [void] (Save-Command 'final-context-strict' $revisions.final $finalContextArgs 0)
    [void] (Save-Command 'final-context-analysis' $revisions.final ($finalContextArgs + @('--analyze-policy-failures')) 0)
    $finalImpactArgs = @('impact', '--base', $revisions.v1, '--repo', $checkout, '--revision', $revisions.final)
    [void] (Save-Command 'v1-to-final-impact-strict' $revisions.final $finalImpactArgs 0)
    [void] (Save-Command 'v1-to-final-impact-analysis' $revisions.final ($finalImpactArgs + @('--analyze-policy-failures')) 0)

    Assert-CleanCheckout $checkout 'Completed replay'
    $candidateVersion = Invoke-Native $candidateExe @('version')
    if ($candidateVersion.ExitCode -ne 0) { throw "Candidate `version` failed: $($candidateVersion.Stderr)" }
    $v012Info = $null
    if (-not [string]::IsNullOrWhiteSpace($PublicV012Executable)) {
        $publicExe = (Resolve-Path -LiteralPath $PublicV012Executable).Path
        $publicHash = Get-Sha256 $publicExe
        if ($publicHash -ne $publicV012Sha256) { throw 'Optional public v0.12.0 executable does not match the preserved release digest.' }
        $publicVersion = Invoke-Native $publicExe @('version')
        if ($publicVersion.ExitCode -ne 0 -or $publicVersion.Stdout -notmatch '^Markitect 0\.12\.0(?:\s|$)') {
            throw 'Optional public executable is not the expected Markitect v0.12.0 binary.'
        }
        $v012Info = [ordered]@{ sha256 = $publicHash; versionOutput = $publicVersion.Stdout.Trim() }
    }
    $metadata = [ordered]@{
        title = 'Markitect policy failure analysis replay'
        runId = $runId
        createdUtc = [DateTime]::UtcNow.ToString('o')
        bundlePath = [IO.Path]::GetRelativePath($repoRoot, $bundlePath).Replace('\', '/')
        bundleSha256 = Get-Sha256 $bundlePath
        candidateSourceCommit = $CandidateSourceCommit
        candidateExecutableSha256 = Get-Sha256 $candidateExe
        candidateVersionOutput = $candidateVersion.Stdout.Trim()
        publicV012 = $v012Info
        revisions = $revisions
        packageArchiveSha256 = $packageHashes
        failingV2Tree = $failingTree
        v1Tree = $v1Tree
        commands = @($records)
        completed = $true
        evidenceScope = 'Read-only inspection only; captured command output and exit codes do not prove application runtime behavior.'
    }
    [IO.File]::WriteAllText((Join-Path $runDirectory 'run.json'), ($metadata | ConvertTo-Json -Depth 12), $encoding)
    Write-Output "Replay evidence: $runDirectory"
}
catch {
    $failure = [ordered]@{
        title = 'Markitect policy failure analysis replay'
        runId = $runId
        createdUtc = [DateTime]::UtcNow.ToString('o')
        candidateSourceCommit = $CandidateSourceCommit
        candidateExecutableSha256 = Get-Sha256 $candidateExe
        revisions = $revisions
        commands = @($records)
        completed = $false
        failure = $_.Exception.Message
    }
    if (Test-Path -LiteralPath $runDirectory) {
        [IO.File]::WriteAllText((Join-Path $runDirectory 'run.json'), ($failure | ConvertTo-Json -Depth 12), $encoding)
    }
    throw
}
