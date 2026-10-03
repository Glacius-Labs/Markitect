[CmdletBinding(DefaultParameterSetName = 'Prepare')]
# FrozenManifest v1 contract: sourceFiles are the shared 646-file app subset;
# trials pin simple-arm revision/task paths and optional Markitect base/overlay revisions;
# assets are explicit hash-bound copies; contextArgs compile read-only task artifacts;
# preparationCommands describe the reviewed T1 projection reconciliation; frozenFiles
# hash all design, task, guidance and oracle files but the harness copies only assets/tasks.
param(
    [Parameter(Mandatory = $true)]
    [string] $SourceBundle,

    [Parameter(Mandatory = $true)]
    [string] $MarkitectExe,

    [Parameter(Mandatory = $true)]
    [string] $RunRoot,

    [Parameter(Mandatory = $true, ParameterSetName = 'Prepare')]
    [switch] $PrepareOnly,

    [Parameter(Mandatory = $true, ParameterSetName = 'Execute')]
    [switch] $Execute,

    [Parameter(Mandatory = $true, ParameterSetName = 'Execute')]
    [Parameter(Mandatory = $true, ParameterSetName = 'BeginExternal')]
    [Parameter(Mandatory = $true, ParameterSetName = 'CompleteExternal')]
    [ValidateSet('t1-simple', 't1-markitect', 't2-simple', 't2-markitect', 't3-simple', 't3-markitect')]
    [string] $RunId,

    [Parameter(Mandatory = $true, ParameterSetName = 'Prepare')]
    [Parameter(Mandatory = $true, ParameterSetName = 'Execute')]
    [string] $FrozenManifest,

    [Parameter(Mandatory = $true, ParameterSetName = 'BeginExternal')]
    [switch] $BeginExternalAgent,

    [Parameter(Mandatory = $true, ParameterSetName = 'BeginExternal')]
    [string] $ExternalFrozenManifest,

    [Parameter(Mandatory = $true, ParameterSetName = 'CompleteExternal')]
    [switch] $CompleteExternalAgent,

    [Parameter(Mandatory = $true, ParameterSetName = 'CompleteExternal')]
    [string] $CompletionFrozenManifest,

    [Parameter(Mandatory = $true, ParameterSetName = 'CompleteExternal')]
    [string] $AgentIdentity,

    [Parameter(Mandatory = $true, ParameterSetName = 'CompleteExternal')]
    [string] $AgentReport,

    [Parameter(Mandatory = $true, ParameterSetName = 'Finalize')]
    [switch] $Finalize,

    [Parameter(Mandatory = $true, ParameterSetName = 'Finalize')]
    [string] $FinalFrozenManifest,

    [Parameter(Mandatory = $true, ParameterSetName = 'ApplyProjections')]
    [switch] $ApplyPreparedProjections,

    [Parameter(Mandatory = $true, ParameterSetName = 'ApplyProjections')]
    [string] $ProjectionFrozenManifest,

    [string[]] $AuditRoots = @()
)

$ErrorActionPreference = 'Stop'
$script:Utf8NoBom = [Text.UTF8Encoding]::new($false)
$script:ExperimentRoot = $PSScriptRoot
$script:RepositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$script:RunRootFull = [IO.Path]::GetFullPath($RunRoot)

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)][string] $FilePath,
        [Parameter(Mandatory = $true)][string[]] $ArgumentList,
        [string] $WorkingDirectory,
        [string] $StandardInput,
        [switch] $AllowFailure
    )

    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $FilePath
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.RedirectStandardInput = $null -ne $StandardInput
    $start.CreateNoWindow = $true
    if ($WorkingDirectory) { $start.WorkingDirectory = $WorkingDirectory }
    foreach ($argument in $ArgumentList) { $start.ArgumentList.Add([string]$argument) }

    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $start
    if (-not $process.Start()) { throw "Could not start $FilePath." }
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    if ($null -ne $StandardInput) {
        $process.StandardInput.Write($StandardInput)
        $process.StandardInput.Close()
    }
    $process.WaitForExit()
    $stdout = $stdoutTask.GetAwaiter().GetResult()
    $stderr = $stderrTask.GetAwaiter().GetResult()
    $result = [pscustomobject]@{ ExitCode = $process.ExitCode; Stdout = $stdout; Stderr = $stderr }
    if (-not $AllowFailure -and $result.ExitCode -ne 0) {
        throw "$FilePath $($ArgumentList -join ' ') failed with exit code $($result.ExitCode).`n$stderr"
    }
    return $result
}

function Invoke-Git {
    param([string[]] $ArgumentList, [string] $WorkingDirectory, [switch] $AllowFailure)
    $gitPath = (Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
    Invoke-Native -FilePath $gitPath -ArgumentList $ArgumentList -WorkingDirectory $WorkingDirectory -AllowFailure:$AllowFailure
}

function Get-Sha256([string] $Path) {
    (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Assert-Hash([string] $Path, [string] $Expected, [string] $Label) {
    $actual = Get-Sha256 $Path
    if ($actual -ne $Expected.ToLowerInvariant()) { throw "$Label SHA-256 mismatch: expected $Expected, found $actual ($Path)." }
}

function Assert-FrozenFiles($Manifest) {
    foreach ($file in $Manifest.frozenFiles) {
        $relative = ([string]$file.path).Replace('\', '/')
        if (-not $relative -or $relative.StartsWith('/') -or $relative -match '(^|/)\.\.?(/|$)') { throw "Invalid frozen file path: $relative" }
        $path = [IO.Path]::GetFullPath((Join-Path $script:RepositoryRoot $relative))
        $prefix = $script:RepositoryRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
        if (-not $path.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) { throw "Frozen file escaped repository: $relative" }
        Assert-Hash $path ([string]$file.sha256) "Frozen file $relative"
    }
}

function Get-WorkspaceAgentInstructions([string] $Workspace) {
    $cursor = [IO.DirectoryInfo]::new([IO.Path]::GetFullPath($Workspace)).Parent
    while ($null -ne $cursor) {
        $inherited = Join-Path $cursor.FullName 'AGENTS.md'
        if (Test-Path -LiteralPath $inherited -PathType Leaf) { throw "Inherited parent AGENTS.md would affect the trial: $inherited" }
        $cursor = $cursor.Parent
    }
    $records = [Collections.Generic.List[object]]::new()
    foreach ($file in Get-ChildItem -LiteralPath $Workspace -Filter 'AGENTS.md' -File -Recurse -Force | Where-Object { $_.FullName -notmatch '[\\/]\.git[\\/]' -and $_.FullName -notmatch '[\\/]\.tools[\\/]' }) {
        $records.Add([ordered]@{ path = [IO.Path]::GetRelativePath($Workspace, $file.FullName).Replace('\', '/'); sha256 = Get-Sha256 $file.FullName; bytes = $file.Length })
    }
    return @($records)
}

function Write-JsonFile([string] $Path, [object] $Value) {
    $json = ConvertTo-Json -InputObject $Value -Depth 30
    [IO.File]::WriteAllText($Path, $json + "`n", $script:Utf8NoBom)
}

function Get-TrialTable($Manifest) {
    $table = @{}
    foreach ($trial in $Manifest.trials) {
        if (-not $trial.id -or -not $trial.revision -or -not $trial.task) { throw 'Each manifest trial requires id, revision, and task.' }
        $table[[string]$trial.id] = $trial
    }
    return $table
}

function Get-AssetRecords($Manifest) {
    if ($null -eq $Manifest.assets) { return @() }
    @($Manifest.assets)
}

function Test-AssetApplies($Asset, [string] $Arm, [string] $Trial, [string] $Phase) {
    if ($Asset.arm -and @($Asset.arm) -notcontains $Arm) { return $false }
    if ($Asset.trial -and @($Asset.trial) -notcontains $Trial) { return $false }
    if ($Asset.phase -and [string]$Asset.phase -ne $Phase) { return $false }
    return $true
}

function Copy-FrozenAssets {
    param($Manifest, [string] $Workspace, [string] $Arm, [string] $Trial, [string] $Phase)
    foreach ($asset in (Get-AssetRecords $Manifest)) {
        if (-not (Test-AssetApplies $asset $Arm $Trial $Phase)) { continue }
        if (-not $asset.source -or -not $asset.destination -or -not $asset.sha256) { throw 'Each applicable frozen asset requires source, destination, and sha256.' }
        if (([string]$asset.source + '/' + [string]$asset.destination) -match '(?i)(oracle|evaluation|grading|judge|answer-key)') { throw 'Oracle/evaluation files may not be copied into agent workspaces.' }
        $sourcePath = [IO.Path]::GetFullPath((Join-Path $script:RepositoryRoot ([string]$asset.source)))
        $repoPrefix = $script:RepositoryRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
        if (-not $sourcePath.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw "Frozen asset source escaped repository: $($asset.source)" }
        if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) { throw "Frozen asset missing: $($asset.source)" }
        Assert-Hash $sourcePath ([string]$asset.sha256) "Frozen asset $($asset.source)"
        $destination = Join-Path $Workspace ([string]$asset.destination)
        $destinationFull = [IO.Path]::GetFullPath($destination)
        $workspacePrefix = $Workspace.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
        if (-not $destinationFull.StartsWith($workspacePrefix, [StringComparison]::OrdinalIgnoreCase)) { throw "Frozen asset destination escaped workspace: $($asset.destination)" }
        $parent = Split-Path -Parent $destinationFull
        if (-not (Test-Path -LiteralPath $parent -PathType Container)) { New-Item -ItemType Directory -Path $parent -Force | Out-Null }
        Copy-Item -LiteralPath $sourcePath -Destination $destinationFull -Force
        Assert-Hash $destinationFull ([string]$asset.sha256) "Copied asset $($asset.destination)"
        if ($destinationFull.Contains([IO.Path]::DirectorySeparatorChar + '.tools' + [IO.Path]::DirectorySeparatorChar) -or $destinationFull.Contains([IO.Path]::DirectorySeparatorChar + '.agent-input' + [IO.Path]::DirectorySeparatorChar)) {
            (Get-Item -LiteralPath $destinationFull).IsReadOnly = $true
        }
    }
}

function Get-FrozenWorkspaceAssetEvidence($Manifest, [string] $Workspace, [string] $Arm, [string] $Trial) {
    $expected = [ordered]@{}
    foreach ($asset in (Get-AssetRecords $Manifest)) {
        if (-not (Test-AssetApplies $asset $Arm $Trial ([string]$asset.phase))) { continue }
        $relative = ([string]$asset.destination).Replace('\', '/')
        $expected[$relative] = [string]$asset.sha256
    }
    $records = [Collections.Generic.List[object]]::new()
    foreach ($relative in $expected.Keys) {
        $path = Join-Path $Workspace ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Frozen workspace asset disappeared: $relative" }
        $actual = Get-Sha256 $path
        if ($actual -ne ([string]$expected[$relative]).ToLowerInvariant()) { throw "Frozen workspace asset changed during preparation: $relative" }
        $records.Add([ordered]@{ path = $relative; sha256 = $actual; bytes = (Get-Item -LiteralPath $path).Length })
    }
    return @($records)
}

function Compare-FrozenWorkspaceAssets($Expected, [string] $Workspace) {
    $changes = [Collections.Generic.List[object]]::new()
    foreach ($asset in $Expected) {
        $path = Join-Path $Workspace ([string]$asset.path.Replace('/', [IO.Path]::DirectorySeparatorChar))
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            $changes.Add([ordered]@{ path = [string]$asset.path; change = 'missing' })
            continue
        }
        $hash = Get-Sha256 $path
        if ($hash -ne [string]$asset.sha256) { $changes.Add([ordered]@{ path = [string]$asset.path; change = 'modified'; expectedSha256 = [string]$asset.sha256; actualSha256 = $hash }) }
    }
    return @($changes)
}

function Compare-CompiledInputs($Expected, [string] $Workspace) {
    $changes = [Collections.Generic.List[object]]::new()
    foreach ($input in $Expected) {
        $path = Join-Path $Workspace ([string]$input.output.Replace('/', [IO.Path]::DirectorySeparatorChar))
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            $changes.Add([ordered]@{ path = [string]$input.output; change = 'missing' })
            continue
        }
        $hash = Get-Sha256 $path
        if ($hash -ne [string]$input.sha256) { $changes.Add([ordered]@{ path = [string]$input.output; change = 'modified'; expectedSha256 = [string]$input.sha256; actualSha256 = $hash }) }
    }
    return @($changes)
}

function Install-AgentReadOnlyInputs([string] $Workspace) {
    $excludePath = Join-Path $Workspace '.git/info/exclude'
    $excludeText = [IO.File]::ReadAllText($excludePath)
    foreach ($line in @('.tools/', '.agent-input/', '.telemetry/')) {
        if (-not $excludeText.Contains($line)) { $excludeText += "`n$line" }
    }
    [IO.File]::WriteAllText($excludePath, $excludeText, $script:Utf8NoBom)
    $inputRoot = Join-Path $Workspace '.agent-input'
    if (Test-Path -LiteralPath $inputRoot -PathType Container) {
        foreach ($file in Get-ChildItem -LiteralPath $inputRoot -File -Recurse) { $file.IsReadOnly = $true }
    }
}

function Install-ReadLedger([string] $Workspace) {
    $toolRoot = Join-Path $Workspace '.tools'
    if (-not (Test-Path -LiteralPath $toolRoot -PathType Container)) { New-Item -ItemType Directory -Path $toolRoot -Force | Out-Null }
    $ledger = @'
param([Parameter(Mandatory=$true)][string]$Path)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$candidate = [IO.Path]::GetFullPath((Join-Path $root $Path))
$resolved = (Resolve-Path -LiteralPath $candidate).Path
$prefix = $root.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $resolved.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Read path must stay inside the trial workspace.' }
if ($candidate -match '^[A-Za-z]:[\\/]' -or $Path.StartsWith('\\') -or $Path -match '(^|[\\/])\.\.?([\\/]|$)') { throw 'Read path must be a normalized workspace-relative path.' }
$cursor = $candidate
while ($cursor.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) {
    if ((Test-Path -LiteralPath $cursor) -and ((Get-Item -LiteralPath $cursor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Read path cannot traverse a reparse point.' }
    $cursor = Split-Path -Parent $cursor
}
if (-not (Test-Path -LiteralPath $resolved -PathType Leaf)) { throw 'Read path must name a regular file.' }
$bytes = [IO.File]::ReadAllBytes($resolved)
$sha = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($bytes)).ToLowerInvariant()
$relative = [IO.Path]::GetRelativePath($root, $resolved).Replace('\','/')
$telemetry = Join-Path $root '.telemetry/reads.jsonl'
$telemetryDirectory = Split-Path -Parent $telemetry
if (Test-Path -LiteralPath $telemetryDirectory) {
    if ((Get-Item -LiteralPath $telemetryDirectory -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Telemetry directory cannot be a reparse point.' }
} else {
    New-Item -ItemType Directory -Path $telemetryDirectory -Force | Out-Null
}
if ((Test-Path -LiteralPath $telemetry) -and ((Get-Item -LiteralPath $telemetry -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Telemetry file cannot be a reparse point.' }
$record = [ordered]@{ path=$relative; bytes=$bytes.Length; sha256=$sha; readType='content' }
[IO.File]::AppendAllText($telemetry, ((ConvertTo-Json -InputObject $record -Compress) + "`n"), [Text.UTF8Encoding]::new($false))
$utf8 = [Text.UTF8Encoding]::new($false, $true)
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
[Console]::Write($utf8.GetString($bytes))
'@
    $helper = Join-Path $toolRoot 'read.ps1'
    [IO.File]::WriteAllText($helper, $ledger, $script:Utf8NoBom)
    (Get-Item -LiteralPath $helper).IsReadOnly = $true
    return $helper
}

function Copy-TaskPrompt($Manifest, $Trial) {
    $taskPath = [string]$Trial.task
    if (-not $taskPath) { throw "Trial $($Trial.id) has no task path." }
    $record = @($Manifest.frozenFiles | Where-Object { ([string]$_.path).Replace('\', '/') -eq $taskPath.Replace('\', '/') }) | Select-Object -First 1
    if (-not $record) { throw "Trial task is not bound in frozenFiles: $taskPath" }
    $full = [IO.Path]::GetFullPath((Join-Path $script:RepositoryRoot $taskPath))
    $repoPrefix = $script:RepositoryRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $full.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw "Frozen task escaped repository: $taskPath" }
    Assert-Hash $full ([string]$record.sha256) "Frozen task $taskPath"
    [IO.File]::ReadAllText($full)
}

function Get-FrozenAgentPrompt($Manifest, $Trial) {
    $genericPrompt = [string]$Manifest.prompts.genericBoundary
    if (-not $Manifest.prompts.genericBoundarySha256) { throw 'Frozen manifest must bind genericBoundarySha256.' }
    $promptHash = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($script:Utf8NoBom.GetBytes($genericPrompt))).ToLowerInvariant()
    if ($promptHash -ne ([string]$Manifest.prompts.genericBoundarySha256).ToLowerInvariant()) { throw 'Frozen generic prompt hash mismatch.' }
    $taskPrompt = Copy-TaskPrompt $Manifest $Trial
    if (-not $genericPrompt -or -not $taskPrompt) { throw 'Frozen manifest must provide prompts.genericBoundary and each trial task path.' }
    $instrumentationPrompt = @'
Work only in this trial workspace. Do not read files from the parent repository, another trial, the operator evidence directory, hidden evaluation material, or original Git history. Read any provided `.agent-input` task context first. For every source or documentation file whose contents you read, use the provided helper from the workspace root: `pwsh -NoProfile -File .tools/read.ps1 -Path <workspace-relative-path>`. This logs content reads with byte counts and SHA-256; metadata searches are not content reads. Keep all writes inside this workspace. When building this .NET project, run from `src` so the nearest `global.json` selects the SDK; report the observed SDK version.
'@
    return $genericPrompt + "`n`n" + $instrumentationPrompt.Trim() + "`n`n" + $taskPrompt
}

function Compile-AgentInputs($Trial, [string] $Workspace, [string] $BaseCommit, [string] $CandidateCommit, [string] $Id) {
    $records = [Collections.Generic.List[object]]::new()
    if (-not $Trial.contextArgs) {
        throw "Markitect trial $Id has no contextArgs; task context scope must be explicitly compiled and frozen."
    }
    $executable = Join-Path $Workspace '.tools/markitect.exe'
    foreach ($spec in @($Trial.contextArgs)) {
        if (-not $spec.args -or -not $spec.output) { throw "Each contextArgs record for $Id requires args and output." }
        if (@($spec.scope).Count -eq 0) { throw "Context scope inputs are required for $Id ($($spec.output))." }
        $outputRelative = ([string]$spec.output).Replace('\', '/')
        if ($outputRelative.StartsWith('/') -or $outputRelative -match '(^|/)\.\.?(/|$)') { throw "Invalid compiled input destination: $outputRelative" }
        $values = @{ repo = $Workspace; executable = $executable; base = $BaseCommit; candidate = $CandidateCommit }
        $args = Expand-Args @($spec.args) $values
        $result = Invoke-Native -FilePath $executable -ArgumentList $args -WorkingDirectory $Workspace -AllowFailure
        $expectedExit = 0
        if ($null -ne $spec.expectedExitCode) { $expectedExit = [int]$spec.expectedExitCode }
        if ($result.ExitCode -ne $expectedExit) { throw "Compiled input '$($spec.output)' for $Id exited $($result.ExitCode); expected $expectedExit. $($result.Stderr)" }
        foreach ($marker in @($spec.stdoutContains)) { if ($marker -and $result.Stdout -notmatch [regex]::Escape([string]$marker)) { throw "Compiled input '$($spec.output)' lacked expected analysis marker: $marker" } }
        $outputPath = Join-Path (Join-Path $Workspace '.agent-input') $outputRelative
        $outputParent = Split-Path -Parent $outputPath
        if (-not (Test-Path -LiteralPath $outputParent -PathType Container)) { New-Item -ItemType Directory -Path $outputParent -Force | Out-Null }
        [IO.File]::WriteAllText($outputPath, $result.Stdout, $script:Utf8NoBom)
        $logDirectory = Join-Path $script:RunRootFull "operator/compiled-inputs/$Id"
        if (-not (Test-Path -LiteralPath $logDirectory -PathType Container)) { New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null }
        $name = [IO.Path]::GetFileName([string]$spec.output)
        [IO.File]::WriteAllText((Join-Path $logDirectory "$name.stdout.txt"), $result.Stdout, $script:Utf8NoBom)
        [IO.File]::WriteAllText((Join-Path $logDirectory "$name.stderr.txt"), $result.Stderr, $script:Utf8NoBom)
        $records.Add([ordered]@{
            command = $args
            expectedExitCode = $expectedExit
            exitCode = $result.ExitCode
            output = ".agent-input/$outputRelative"
            sha256 = Get-Sha256 $outputPath
            bytes = (Get-Item -LiteralPath $outputPath).Length
            scope = $spec.scope
        })
    }
    if ($Id -eq 't2-markitect') {
        if ($records.Count -ne 2 -or @($records | Where-Object { $_.command[0] -eq 'context' }).Count -ne 1 -or @($records | Where-Object { $_.command[0] -eq 'impact' }).Count -ne 1) {
            throw 'T2 Markitect preparation requires exactly one Context and one Impact analysis.'
        }
        foreach ($record in $records) {
            if ($record.command -notcontains '--analyze-policy-failures') { throw 'T2 Context and Impact must use --analyze-policy-failures.' }
            if (@($record.scope).Count -eq 0) { throw 'T2 Context and Impact must record their exact selected resource/revision scope.' }
        }
        foreach ($spec in @($Trial.contextArgs)) {
            if (@($spec.stdoutContains).Count -eq 0) { throw 'T2 Context and Impact require frozen output markers for useful completed analysis.' }
        }
    } elseif ($records.Count -ne 1 -or $records[0].command[0] -ne 'context') {
        throw "$Id Markitect preparation requires exactly one compiled Context input."
    }
    foreach ($file in Get-ChildItem -LiteralPath (Join-Path $Workspace '.agent-input') -File -Recurse) { $file.IsReadOnly = $true }
    return @($records)
}

function Invoke-PreparationCommands($Trial, [string] $Workspace, [string] $BaseCommit, [string] $CandidateCommit, [string] $Id) {
    $records = [Collections.Generic.List[object]]::new()
    if (-not $Trial.preparationCommands) { return [pscustomobject]@{ Completed = @($records); Pending = @() } }
    $evidenceRoot = Join-Path $script:RunRootFull "operator/preparation/$Id"
    if (-not (Test-Path -LiteralPath $evidenceRoot -PathType Container)) { New-Item -ItemType Directory -Path $evidenceRoot -Force | Out-Null }
    $index = 0
    $commands = @($Trial.preparationCommands)
    for ($commandIndex = 0; $commandIndex -lt $commands.Count; $commandIndex++) {
        $spec = $commands[$commandIndex]
        if ($spec.requiresPlanReview -eq $true) {
            return [pscustomobject]@{ Completed = @($records); Pending = @($commands[$commandIndex..($commands.Count - 1)]) }
        }
        if (-not $spec.args -or -not $spec.executable) { throw "Each preparation command for $Id requires executable and args." }
        $values = @{ repo = $Workspace; workspace = $Workspace; executable = (Join-Path $Workspace '.tools/markitect.exe'); markitect = (Join-Path $Workspace '.tools/markitect.exe'); base = $BaseCommit; candidate = $CandidateCommit; evidence = $evidenceRoot; externalEvidenceRoot = $evidenceRoot }
        $commandArgs = Expand-Args @($spec.args) $values
        $executable = [string]$spec.executable
        if ($executable -eq '{markitect}') { $executable = Join-Path $Workspace '.tools/markitect.exe' }
        $workingDirectory = $Workspace
        if ($spec.workingDirectory) { $workingDirectory = [IO.Path]::GetFullPath((Join-Path $Workspace (Expand-Args @([string]$spec.workingDirectory) $values)[0])) }
        $stdin = $null
        if ($spec.stdin) {
            $stdinPath = Join-Path $Workspace ([string]$spec.stdin)
            if (-not (Test-Path -LiteralPath $stdinPath -PathType Leaf)) { throw "Preparation stdin is missing: $($spec.stdin)" }
            $stdin = [IO.File]::ReadAllText($stdinPath)
        }
        $result = Invoke-Native -FilePath $executable -ArgumentList $commandArgs -WorkingDirectory $workingDirectory -StandardInput $stdin -AllowFailure
        $expectedExit = 0
        if ($null -ne $spec.expectedExitCode) { $expectedExit = [int]$spec.expectedExitCode }
        if ($result.ExitCode -ne $expectedExit) { throw "Preparation command '$($spec.name)' exited $($result.ExitCode); expected $expectedExit. $($result.Stderr)" }
        foreach ($pattern in @($spec.stdoutContains)) {
            if ($pattern -and $result.Stdout -notmatch [regex]::Escape([string]$pattern)) { throw "Preparation command '$($spec.name)' did not emit expected text: $pattern" }
        }
        $name = if ($spec.name) { [string]$spec.name } else { "step-$index" }
        $stdoutName = "$name.stdout.txt"
        $stderrName = "$name.stderr.txt"
        [IO.File]::WriteAllText((Join-Path $evidenceRoot $stdoutName), $result.Stdout, $script:Utf8NoBom)
        [IO.File]::WriteAllText((Join-Path $evidenceRoot $stderrName), $result.Stderr, $script:Utf8NoBom)
        $records.Add([ordered]@{ name = $name; executable = $executable; args = $commandArgs; workingDirectory = $workingDirectory; expectedExitCode = $expectedExit; exitCode = $result.ExitCode; stdoutPath = "$Id/$stdoutName"; stdoutSha256 = Get-Sha256 (Join-Path $evidenceRoot $stdoutName); stderrPath = "$Id/$stderrName" })
        $index++
    }
    return [pscustomobject]@{ Completed = @($records); Pending = @() }
}

function Assert-ProjectionPlanTargets([string] $PlanPath) {
    if (-not (Test-Path -LiteralPath $PlanPath -PathType Leaf)) { throw "Projection plan is missing: $PlanPath" }
    $text = [IO.File]::ReadAllText($PlanPath)
    $pathMatches = [regex]::Matches($text, '(?m)^\s*path:\s*["'']?([^"''\r\n]+)')
    if ($pathMatches.Count -lt 1) { throw 'Projection plan contained no file operation paths; refusing to apply.' }
    foreach ($match in $pathMatches) {
        $path = $match.Groups[1].Value.Trim()
        if ($path.Contains('\') -or $path.StartsWith('/') -or $path -match '(^|/)\.\.?(/|$)') { throw "Projection plan contains a non-relative or escaping target: $path" }
        if (-not ($path.StartsWith('.agents/skills/', [StringComparison]::Ordinal) -or $path.StartsWith('docs/markitect/', [StringComparison]::Ordinal))) {
            throw "Projection plan target is outside allowed skill/projection roots: $path"
        }
    }
    return @($pathMatches | ForEach-Object { $_.Groups[1].Value.Trim() })
}

function Get-SourceFiles($Manifest) {
    $sourceFiles = @($Manifest.sourceFiles)
    if ($sourceFiles.Count -lt 1) { throw 'The frozen manifest must list sourceFiles.' }
    $seen = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($item in $sourceFiles) {
        $relative = ([string]$item.path).Replace('\', '/')
        if (-not $relative -or $relative.StartsWith('/') -or $relative.Contains('..') -or -not $item.sha256) { throw "Invalid source file record: $($item | ConvertTo-Json -Compress)" }
        if (-not $seen.Add($relative)) { throw "Duplicate source file in manifest: $relative" }
    }
    return $sourceFiles
}

function Export-SourceSnapshot {
    param($Manifest, $Trial, [string] $Workspace, [string] $Inspector, [string] $Arm)
    $revision = [string]$Trial.revision
    if ($Arm -eq 'markitect' -and $Trial.markitectBaseRevision) { $revision = [string]$Trial.markitectBaseRevision }
    $files = Get-SourceFiles $Manifest
    $archive = Join-Path $script:RunRootFull "operator/source-$($Trial.id)-$Arm.zip"
    $archiveParent = Split-Path -Parent $archive
    if (-not (Test-Path -LiteralPath $archiveParent -PathType Container)) { New-Item -ItemType Directory -Path $archiveParent -Force | Out-Null }
    $args = @('-C', $Inspector, 'archive', '--format=zip', "--output=$archive", $revision)
    if ($Arm -eq 'simple') { $args += @('--', 'src', 'docs/architecture-decision-log', 'README.md', 'LICENSE') }
    [void](Invoke-Git -ArgumentList $args -WorkingDirectory $script:RunRootFull)
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [IO.Compression.ZipFile]::ExtractToDirectory($archive, $Workspace)
    $records = [Collections.Generic.List[object]]::new()
    foreach ($item in $files) {
        $relative = ([string]$item.path).Replace('/', [IO.Path]::DirectorySeparatorChar)
        $path = Join-Path $Workspace $relative
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Archive omitted manifest source: $($item.path)" }
        Assert-Hash $path ([string]$item.sha256) "Exported source $($item.path)"
        $records.Add([ordered]@{ path = ([string]$item.path).Replace('\', '/'); sha256 = Get-Sha256 $path; bytes = (Get-Item -LiteralPath $path).Length })
    }
    $sourceManifest = [ordered]@{
        manifestVersion = 1
        originalRevision = [string]$Trial.revision
        fileCount = $records.Count
        files = @($records)
    }
    $metadataPath = Join-Path $script:RunRootFull "operator/source-manifests/$($Trial.id).json"
    $metadataParent = Split-Path -Parent $metadataPath
    if (-not (Test-Path -LiteralPath $metadataParent -PathType Container)) { New-Item -ItemType Directory -Path $metadataParent -Force | Out-Null }
    Write-JsonFile $metadataPath $sourceManifest
    $fullTreeFiles = @()
    $fullTreeManifestPath = $null
    $fullTreeManifestHash = $null
    if ($Arm -eq 'markitect') {
        $fullTreeFiles = @(Get-ChildItem -LiteralPath $Workspace -File -Recurse | Where-Object { $_.FullName -notmatch '[\\/]\.git[\\/]' } | ForEach-Object {
            [ordered]@{ path = [IO.Path]::GetRelativePath($Workspace, $_.FullName).Replace('\', '/'); sha256 = Get-Sha256 $_.FullName; bytes = $_.Length }
        })
        $fullTreeManifestPath = Join-Path $script:RunRootFull "operator/canonical-tree-manifests/$($Trial.id).json"
        $fullTreeManifestParent = Split-Path -Parent $fullTreeManifestPath
        if (-not (Test-Path -LiteralPath $fullTreeManifestParent -PathType Container)) { New-Item -ItemType Directory -Path $fullTreeManifestParent -Force | Out-Null }
        Write-JsonFile $fullTreeManifestPath ([ordered]@{ manifestVersion = 1; exportedRevision = $revision; fileCount = $fullTreeFiles.Count; files = @($fullTreeFiles) })
        $fullTreeManifestHash = Get-Sha256 $fullTreeManifestPath
    }
    return [pscustomobject]@{ ManifestPath = $metadataPath; Hash = Get-Sha256 $metadataPath; Files = $records; FullTreeFiles = $fullTreeFiles; FullTreeManifestPath = $fullTreeManifestPath; FullTreeManifestHash = $fullTreeManifestHash; ExportRevision = $revision }
}

function Initialize-WorkspaceGit([string] $Workspace, [string] $Branch, [string] $Message) {
    if (-not (Test-Path -LiteralPath (Join-Path $Workspace '.git'))) { [void](Invoke-Git @('init', '--initial-branch', $Branch) $Workspace) }
    [void](Invoke-Git @('add', '--all') $Workspace)
    [void](Invoke-Git @('-c', 'user.name=Comparison Harness', '-c', 'user.email=harness@markitect.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', $Message) $Workspace)
    (Invoke-Git @('rev-parse', 'HEAD') $Workspace).Stdout.Trim()
}

function Get-ExcludedDirectory([string] $Name) {
    @('.git', '.tools', 'bin', 'obj', '.vs', 'node_modules', '.codex-cache', '.nuget', '.artifacts', 'artifacts', 'TestResults', '.pytest_cache', '.mypy_cache', '.ruff_cache', 'cache') -contains $Name
}

function Get-DirectorySnapshot([string] $Root, [switch] $IncludeIgnored) {
    $full = [IO.Path]::GetFullPath($Root)
    if (-not (Test-Path -LiteralPath $full -PathType Container)) { return [ordered]@{ exists = $false; gitStatus = $null; files = @{} } }
    $files = [ordered]@{}
    $gitStatus = $null
    $isGit = Test-Path -LiteralPath (Join-Path $full '.git')
    if ($isGit) {
        $status = Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $full -AllowFailure
        $gitStatus = [ordered]@{ exitCode = $status.ExitCode; text = $status.Stdout }
    }
    $relativePaths = [Collections.Generic.List[string]]::new()
    if ($IncludeIgnored -or -not $isGit) {
        $stack = [Collections.Generic.Stack[string]]::new()
        $stack.Push($full)
        while ($stack.Count -gt 0) {
            $directory = $stack.Pop()
            foreach ($item in Get-ChildItem -LiteralPath $directory -Force) {
                if ($item.PSIsContainer) {
                    if (-not (Get-ExcludedDirectory $item.Name)) { $stack.Push($item.FullName) }
                    continue
                }
                if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { continue }
                $relativePaths.Add([IO.Path]::GetRelativePath($full, $item.FullName).Replace('\', '/'))
            }
        }
    } else {
        $listed = Invoke-Git @('ls-files', '--cached', '--others', '--exclude-standard', '--full-name') $full -AllowFailure
        if ($listed.ExitCode -ne 0) { throw "Could not list tracked/nonignored files under ${full}: $($listed.Stderr)" }
        foreach ($line in $listed.Stdout -split "`r?`n") { if ($line) { $relativePaths.Add($line.Replace('\', '/')) } }
        $ignored = Invoke-Git @('ls-files', '--others', '--ignored', '--exclude-standard', '--full-name') $full -AllowFailure
        if ($ignored.ExitCode -ne 0) { throw "Could not list ignored files under ${full}: $($ignored.Stderr)" }
        foreach ($line in $ignored.Stdout -split "`r?`n") { if ($line) { $relativePaths.Add($line.Replace('\', '/')) } }
    }
    foreach ($relative in $relativePaths) {
        $segments = $relative.Split('/')
        if (@($segments | Where-Object { Get-ExcludedDirectory $_ }).Count -gt 0) { continue }
        $path = Join-Path $full ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
        $item = Get-Item -LiteralPath $path -Force
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { continue }
        $files[$relative] = [ordered]@{ bytes = $item.Length; sha256 = Get-Sha256 $path }
    }
    $coverage = if ($IncludeIgnored) { 'all regular files except .git/.tools and cache/build directories' } elseif ($isGit) { 'tracked plus all untracked regular files except .git/.tools and cache/build directories; Git status retained' } else { 'all regular files except .git/.tools and cache/build directories' }
    return [ordered]@{ exists = $true; root = $full; gitStatus = $gitStatus; fileCoverage = $coverage; files = $files }
}

function Get-AuditSnapshotSet([string[]] $Roots, [string[]] $PreparedWorkspaces) {
    $preparedSet = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($workspace in $PreparedWorkspaces) { [void]$preparedSet.Add([IO.Path]::GetFullPath($workspace)) }
    $snapshots = [ordered]@{}
    foreach ($root in $Roots) {
        $full = [IO.Path]::GetFullPath($root)
        $includeIgnored = $preparedSet.Contains($full)
        $snapshots[$full] = Get-DirectorySnapshot $full -IncludeIgnored:$includeIgnored
    }
    return $snapshots
}

function Get-AuditRoots([string[]] $AdditionalRoots, [string[]] $PreparedWorkspaces) {
    $roots = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    [void]$roots.Add($script:RepositoryRoot)
    foreach ($candidate in $AdditionalRoots) {
        if (-not [string]::IsNullOrWhiteSpace($candidate)) { [void]$roots.Add([IO.Path]::GetFullPath($candidate)) }
    }
    $gitPath = (Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
    $worktrees = Invoke-Native -FilePath $gitPath -ArgumentList @('-C', $script:RepositoryRoot, 'worktree', 'list', '--porcelain') -AllowFailure
    if ($worktrees.ExitCode -eq 0) {
        foreach ($line in $worktrees.Stdout -split "`r?`n") {
            if ($line.StartsWith('worktree ')) { [void]$roots.Add($line.Substring(9).Trim()) }
        }
    }
    foreach ($workspace in $PreparedWorkspaces) { [void]$roots.Add($workspace) }
    return @($roots | Sort-Object)
}

function Get-SnapshotDelta($Before, $After) {
    $changes = [Collections.Generic.List[object]]::new()
    $allRoots = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($key in $Before.Keys) { [void]$allRoots.Add($key) }
    foreach ($key in $After.Keys) { [void]$allRoots.Add($key) }
    foreach ($root in $allRoots) {
        if (-not $Before.Contains($root) -or -not $After.Contains($root)) {
            $changes.Add([ordered]@{ root = $root; path = ''; change = 'root-added-or-removed' })
            continue
        }
        $oldFiles = $Before[$root].files
        $newFiles = $After[$root].files
        $relativePaths = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
        foreach ($path in $oldFiles.Keys) { [void]$relativePaths.Add($path) }
        foreach ($path in $newFiles.Keys) { [void]$relativePaths.Add($path) }
        foreach ($path in $relativePaths) {
            if (-not $oldFiles.Contains($path)) { $changes.Add([ordered]@{ root = $root; path = $path; change = 'created'; after = $newFiles[$path] }); continue }
            if (-not $newFiles.Contains($path)) { $changes.Add([ordered]@{ root = $root; path = $path; change = 'deleted'; before = $oldFiles[$path] }); continue }
            if ($oldFiles[$path].sha256 -ne $newFiles[$path].sha256 -or $oldFiles[$path].bytes -ne $newFiles[$path].bytes) {
                $changes.Add([ordered]@{ root = $root; path = $path; change = 'modified'; before = $oldFiles[$path]; after = $newFiles[$path] })
            }
        }
    }
    return @($changes)
}

function Get-ManifestTrial($Manifest, [string] $Id) {
    $trialId, $arm = $Id.Split('-', 2)
    $table = Get-TrialTable $Manifest
    if (-not $table.ContainsKey($trialId)) { throw "Frozen manifest does not define $trialId." }
    if ($arm -notin @('simple', 'markitect')) { throw "Invalid experiment arm in RunId: $arm" }
    return [pscustomobject]@{ Trial = $table[$trialId]; Arm = $arm; Id = $Id }
}

function Expand-Args([object[]] $ArgumentTemplates, [hashtable] $Values) {
    $result = [Collections.Generic.List[string]]::new()
    foreach ($arg in $ArgumentTemplates) {
        $expanded = [string]$arg
        foreach ($key in $Values.Keys) { $expanded = $expanded.Replace("{$key}", [string]$Values[$key]) }
        $result.Add($expanded)
    }
    return @($result)
}

function Read-EventSummary([string] $JsonLines) {
    $events = [Collections.Generic.List[object]]::new()
    foreach ($line in $JsonLines -split "`r?`n") {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        try { $events.Add(($line | ConvertFrom-Json -AsHashtable)) } catch { }
    }
    $tools = [Collections.Generic.List[string]]::new()
    $models = [Collections.Generic.List[string]]::new()
    $usage = [Collections.Generic.List[object]]::new()
    foreach ($event in $events) {
        $type = [string]$event.type
        if ($type -match 'tool|command') {
            $name = [string]$event.name
            if (-not $name) { $name = [string]$event.tool }
            if ($name) { $tools.Add($name) }
        }
        foreach ($key in @('model', 'model_slug')) {
            if ($event[$key] -is [string] -and $event[$key]) { $models.Add([string]$event[$key]) }
        }
        if ($event.usage) { $usage.Add($event.usage) }
        if ($event.response.usage) { $usage.Add($event.response.usage) }
    }
    $lastMessage = $null
    for ($index = $events.Count - 1; $index -ge 0; $index--) {
        $event = $events[$index]
        if ($event.type -match 'message|output' -and $event.message) { $lastMessage = [string]$event.message; break }
        if ($event.type -eq 'item.completed' -and $event.item.type -eq 'agent_message') { $lastMessage = [string]$event.item.text; break }
    }
    return [ordered]@{ parsedEventCount = $events.Count; modelsObserved = @($models | Select-Object -Unique); toolEvents = @($tools); usageRecords = @($usage); lastMessageAnnotation = $lastMessage }
}

function Assert-RunRoot {
    if (-not [IO.Path]::IsPathRooted($RunRoot)) { throw 'RunRoot must be an absolute path.' }
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $script:RunRootFull.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) { throw "RunRoot must be under the system Temp directory: $tempRoot" }
    $repoPrefix = $script:RepositoryRoot.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if ($script:RunRootFull.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase) -or $script:RunRootFull.Equals($script:RepositoryRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'RunRoot must be outside the Markitect repository.'
    }
}

Assert-RunRoot
$sourceBundleFull = (Resolve-Path -LiteralPath $SourceBundle).Path
$markitectFull = (Resolve-Path -LiteralPath $MarkitectExe).Path
if (-not (Test-Path -LiteralPath $sourceBundleFull -PathType Leaf) -and -not (Test-Path -LiteralPath $sourceBundleFull -PathType Container)) { throw 'SourceBundle must be a local Git bundle or repository directory.' }
if (-not (Test-Path -LiteralPath $markitectFull -PathType Leaf)) { throw 'MarkitectExe must name an existing executable file.' }

if ($PrepareOnly) {
    if (Test-Path -LiteralPath $script:RunRootFull) {
        if (@(Get-ChildItem -LiteralPath $script:RunRootFull -Force).Count -ne 0) { throw "RunRoot must be new or empty: $script:RunRootFull" }
    } else {
        New-Item -ItemType Directory -Path $script:RunRootFull -Force | Out-Null
    }
    $manifestPath = Join-Path $script:RunRootFull 'operator/frozen-manifest.json'
    if ($FrozenManifest) {
        $manifestPath = (Resolve-Path -LiteralPath $FrozenManifest).Path
    } else {
        throw 'PrepareOnly also requires -FrozenManifest so the experiment is bound to frozen design and oracle hashes.'
    }
    $manifestRaw = [IO.File]::ReadAllText($manifestPath)
    $manifest = $manifestRaw | ConvertFrom-Json -AsHashtable
    if ([int]$manifest.schemaVersion -ne 1) { throw 'Unsupported frozen manifest schemaVersion; expected 1.' }
    if (-not $manifest.frozenFiles -or -not $manifest.oracleFreezeSha256) { throw 'Frozen manifest must bind the frozen design files and oracle freeze hash.' }
    if (-not $manifest.prompts.genericBoundarySha256) { throw 'Frozen manifest must bind genericBoundarySha256.' }
    Assert-FrozenFiles $manifest
    Assert-Hash $markitectFull ([string]$manifest.markitect.sha256) 'Integrated Markitect executable'

    $inspector = Join-Path $script:RunRootFull 'operator/source-inspector'
    if (-not (Test-Path -LiteralPath $inspector)) {
        New-Item -ItemType Directory -Path $inspector -Force | Out-Null
        [void](Invoke-Git @('init', '--initial-branch', 'inspect') $inspector)
        $revisions = [Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
        foreach ($trial in $manifest.trials) {
            foreach ($revision in @($trial.revision, $trial.markitectBaseRevision, $trial.markitectOverlayRevision)) {
                if ($revision -and $revisions.Add([string]$revision)) { [void](Invoke-Git @('fetch', $sourceBundleFull, ([string]$revision)) $inspector) }
            }
        }
    }

    $prepared = [Collections.Generic.List[object]]::new()
    $trialTable = Get-TrialTable $manifest
    foreach ($trialId in @('t1', 't2', 't3')) {
        if (-not $trialTable.ContainsKey($trialId)) { throw "Frozen manifest must define $trialId." }
        $trial = $trialTable[$trialId]
        foreach ($arm in @('simple', 'markitect')) {
            $id = "$trialId-$arm"
            $checkout = Join-Path (Join-Path $script:RunRootFull 'workspaces') "$id/checkout"
            if (Test-Path -LiteralPath $checkout) { throw "Refusing to overwrite prepared workspace: $checkout" }
            New-Item -ItemType Directory -Path $checkout -Force | Out-Null
            $sourceEvidence = Export-SourceSnapshot $manifest $trial $checkout $inspector $arm
            Copy-FrozenAssets $manifest $checkout $arm $trialId 'base'
            $branch = "codex/comparison-$id"
            [void](Invoke-Git @('init', '--initial-branch', $branch) $checkout)
            Install-AgentReadOnlyInputs $checkout
            $readHelperPath = Install-ReadLedger $checkout
            $baseCommit = Initialize-WorkspaceGit $checkout $branch "Freeze source snapshot for $id"
            $candidateCommit = $baseCommit
            if ($arm -eq 'markitect' -and $trial.markitectOverlayRevision) {
                $overlayPath = [string]$trial.markitectOverlayPath
                if (-not $overlayPath) { $overlayPath = 'markitect.yaml' }
                if ($overlayPath -ne 'markitect.yaml') { throw 'The canonical comparison overlay is limited to the frozen Project pin file markitect.yaml.' }
                $blob = Invoke-Git @('show', "$($trial.markitectOverlayRevision):$overlayPath") $inspector
                [IO.File]::WriteAllText((Join-Path $checkout $overlayPath), $blob.Stdout, $script:Utf8NoBom)
            }
            Copy-FrozenAssets $manifest $checkout $arm $trialId 'candidate'
            if ((Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $checkout).Stdout.Trim()) {
                [void](Invoke-Git @('add', '--all') $checkout)
                [void](Invoke-Git @('-c', 'user.name=Comparison Harness', '-c', 'user.email=harness@markitect.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', "Freeze model candidate for $id") $checkout)
                $candidateCommit = (Invoke-Git @('rev-parse', 'HEAD') $checkout).Stdout.Trim()
            }
            if ($arm -eq 'markitect') {
                $toolsPath = Join-Path $checkout '.tools'
                New-Item -ItemType Directory -Path $toolsPath -Force | Out-Null
                Copy-Item -LiteralPath $markitectFull -Destination (Join-Path $toolsPath 'markitect.exe')
                Assert-Hash (Join-Path $toolsPath 'markitect.exe') ([string]$manifest.markitect.sha256) 'Workspace Markitect executable'
                (Get-Item -LiteralPath (Join-Path $toolsPath 'markitect.exe')).IsReadOnly = $true
                (Get-Item -LiteralPath (Join-Path $toolsPath 'markitect.exe')).IsReadOnly = $true
            }
            Install-AgentReadOnlyInputs $checkout
            $prepResult = if ($arm -eq 'markitect') { Invoke-PreparationCommands $trial $checkout $baseCommit $candidateCommit $id } else { [pscustomobject]@{ Completed = @(); Pending = @() } }
            $preparationCommands = @($prepResult.Completed)
            $pendingPreparationCommands = @($prepResult.Pending)
            if ($pendingPreparationCommands.Count -eq 0) {
                $dirtyAfterPreparation = (Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $checkout).Stdout.Trim()
                if ($dirtyAfterPreparation) {
                    [void](Invoke-Git @('add', '--all') $checkout)
                    [void](Invoke-Git @('-c', 'user.name=Comparison Harness', '-c', 'user.email=harness@markitect.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', "Prepare Markitect projections for $id") $checkout)
                    $candidateCommit = (Invoke-Git @('rev-parse', 'HEAD') $checkout).Stdout.Trim()
                }
            }
            $compiledInputs = @()
            if ($arm -eq 'markitect' -and $pendingPreparationCommands.Count -eq 0) { $compiledInputs = @(Compile-AgentInputs $trial $checkout $baseCommit $candidateCommit $id) }
            $agentInstructions = @(Get-WorkspaceAgentInstructions $checkout)
            $frozenAssets = @(Get-FrozenWorkspaceAssetEvidence $manifest $checkout $arm $trialId)
            $record = [ordered]@{
                id = $id; task = $trialId; arm = $arm; originalRevision = [string]$trial.revision; exportedRevision = $sourceEvidence.ExportRevision
                overlayRevision = if ($arm -eq 'markitect') { [string]$trial.markitectOverlayRevision } else { $null }
                overlayPath = if ($arm -eq 'markitect' -and $trial.markitectOverlayRevision) { if ($trial.markitectOverlayPath) { [string]$trial.markitectOverlayPath } else { 'markitect.yaml' } } else { $null }
                sourceManifestSha256 = $sourceEvidence.Hash; sourceManifestPath = [IO.Path]::GetRelativePath($script:RunRootFull, $sourceEvidence.ManifestPath).Replace('\', '/')
                fullTreeFileCount = @($sourceEvidence.FullTreeFiles).Count
                fullTreeManifestPath = if ($sourceEvidence.FullTreeManifestPath) { [IO.Path]::GetRelativePath($script:RunRootFull, $sourceEvidence.FullTreeManifestPath).Replace('\', '/') } else { $null }
                fullTreeManifestSha256 = $sourceEvidence.FullTreeManifestHash
                baseCommit = $baseCommit; candidateCommit = $candidateCommit; workspace = $checkout
                compiledInputs = @($compiledInputs)
                frozenWorkspaceAssets = @($frozenAssets)
                preparationCommands = @($preparationCommands)
                pendingPreparationCommands = @($pendingPreparationCommands)
                readHelperSha256 = Get-Sha256 $readHelperPath
                automaticAgentInstructions = @($agentInstructions)
                markitectExecutableSha256 = if ($arm -eq 'markitect') { [string]$manifest.markitect.sha256 } else { $null }
            }
            $prepared.Add($record)
        }
    }
    # Paired arms must begin with byte-identical selected application files, independent of arm model additions.
    foreach ($trialId in @('t1', 't2', 't3')) {
        $pair = @($prepared | Where-Object { $_.task -eq $trialId })
        if ($pair.Count -ne 2 -or $pair[0].sourceManifestSha256 -ne $pair[1].sourceManifestSha256) { throw "Source-byte parity failed for $trialId." }
    }
    $runManifest = [ordered]@{
        evidenceType = 'agents-md-comparison-preparation'
        schemaVersion = 1
        frozenManifestSha256 = Get-Sha256 $manifestPath
        oracleFreezeSha256 = [string]$manifest.oracleFreezeSha256
        sourceBundle = [IO.Path]::GetFullPath($sourceBundleFull)
        sourceBundleSha256 = if (Test-Path -LiteralPath $sourceBundleFull -PathType Leaf) { Get-Sha256 $sourceBundleFull } else { $null }
        markitectExecutableSha256 = [string]$manifest.markitect.sha256
        workspaces = @($prepared)
        auditRoots = @(Get-AuditRoots $AuditRoots @($prepared | ForEach-Object { [string]$_.workspace }))
        status = 'prepared only; no agents have run'
    }
    $manifestOut = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    Write-JsonFile $manifestOut $runManifest
    $wholeBefore = Get-AuditSnapshotSet @($runManifest.auditRoots) @($prepared | ForEach-Object { [string]$_.workspace })
    Write-JsonFile (Join-Path $script:RunRootFull 'operator/whole-experiment-before.json') $wholeBefore
    Write-Output "Prepared six isolated workspaces under $script:RunRootFull. No agent was invoked."
    exit 0
}

if ($Finalize) {
    $finalFrozenPath = (Resolve-Path -LiteralPath $FinalFrozenManifest).Path
    $preparedPath = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    if (-not (Test-Path -LiteralPath $preparedPath -PathType Leaf)) { throw 'RunRoot is not prepared.' }
    $prepared = [IO.File]::ReadAllText($preparedPath) | ConvertFrom-Json -AsHashtable
    if ($prepared.frozenManifestSha256 -ne (Get-Sha256 $finalFrozenPath)) { throw 'Final manifest differs from preparation.' }
    $finalManifest = [IO.File]::ReadAllText($finalFrozenPath) | ConvertFrom-Json -AsHashtable
    Assert-FrozenFiles $finalManifest
    $beforePath = Join-Path $script:RunRootFull 'operator/whole-experiment-before.json'
    if (-not (Test-Path -LiteralPath $beforePath -PathType Leaf)) { throw 'Whole-experiment baseline audit is missing.' }
    $before = [IO.File]::ReadAllText($beforePath) | ConvertFrom-Json -AsHashtable
    $after = Get-AuditSnapshotSet @($prepared.auditRoots) @($prepared.workspaces | ForEach-Object { [string]$_.workspace })
    Write-JsonFile (Join-Path $script:RunRootFull 'operator/whole-experiment-after.json') $after
    $delta = Get-SnapshotDelta $before $after
    Write-JsonFile (Join-Path $script:RunRootFull 'operator/whole-experiment-delta.json') $delta
    Write-Output "Whole-experiment audit completed. Detected $($delta.Count) path change(s); see operator/whole-experiment-delta.json."
    exit 0
}

if ($ApplyPreparedProjections) {
    $projectionManifestPath = (Resolve-Path -LiteralPath $ProjectionFrozenManifest).Path
    $preparedPath = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    if (-not (Test-Path -LiteralPath $preparedPath -PathType Leaf)) { throw 'RunRoot is not prepared.' }
    $manifest = [IO.File]::ReadAllText($projectionManifestPath) | ConvertFrom-Json -AsHashtable
    $prepared = [IO.File]::ReadAllText($preparedPath) | ConvertFrom-Json -AsHashtable
    if ($prepared.frozenManifestSha256 -ne (Get-Sha256 $projectionManifestPath)) { throw 'Projection manifest differs from preparation.' }
    Assert-FrozenFiles $manifest
    $run = @($prepared.workspaces | Where-Object { $_.id -eq 't1-markitect' }) | Select-Object -First 1
    if (-not $run -or -not $run.pendingPreparationCommands) { throw 'No reviewed projection plan is pending for t1-markitect.' }
    $trial = @($manifest.trials | Where-Object { $_.id -eq 't1' }) | Select-Object -First 1
    $pending = @($run.pendingPreparationCommands)
    $reviewSpec = @($pending | Where-Object { $_.reviewPlan } | Select-Object -First 1)
    if (-not $reviewSpec) { throw 'Pending projection commands must identify reviewPlan.' }
    $evidenceRoot = Join-Path $script:RunRootFull 'operator/preparation/t1-markitect'
    $planPath = Join-Path $evidenceRoot ([string]$reviewSpec.reviewPlan)
    $planOutput = @($run.preparationCommands | Where-Object { ([string]$_.stdoutPath).EndsWith([string]$reviewSpec.reviewPlan, [StringComparison]::OrdinalIgnoreCase) }) | Select-Object -First 1
    if (-not $planOutput -or (Get-Sha256 $planPath) -ne [string]$planOutput.stdoutSha256) { throw 'Reviewed plan bytes differ from the captured plan stdout.' }
    if ((Invoke-Git @('rev-parse', 'HEAD') ([string]$run.workspace)).Stdout.Trim() -ne [string]$run.candidateCommit -or (Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') ([string]$run.workspace)).Stdout.Trim()) {
        throw 'T1 Markitect workspace changed after the plan was prepared; refusing stale-plan apply.'
    }
    Assert-Hash (Join-Path $run.workspace '.tools/markitect.exe') ([string]$manifest.markitect.sha256) 'Markitect executable before projection apply'
    $allowedPlanTargets = Assert-ProjectionPlanTargets $planPath
    $completed = [Collections.Generic.List[object]]::new()
    $index = 0
    foreach ($spec in $pending) {
        if (-not $spec.args -or -not $spec.executable) { throw 'Pending projection command requires executable and args.' }
        $values = @{ repo = [string]$run.workspace; workspace = [string]$run.workspace; executable = (Join-Path $run.workspace '.tools/markitect.exe'); markitect = (Join-Path $run.workspace '.tools/markitect.exe'); base = [string]$run.baseCommit; candidate = [string]$run.candidateCommit; evidence = $evidenceRoot; externalEvidenceRoot = $evidenceRoot }
        $args = Expand-Args @($spec.args) $values
        $exe = if ([string]$spec.executable -eq '{markitect}') { Join-Path $run.workspace '.tools/markitect.exe' } else { [string]$spec.executable }
        $result = Invoke-Native -FilePath $exe -ArgumentList $args -WorkingDirectory ([string]$run.workspace) -AllowFailure
        $expected = if ($null -ne $spec.expectedExitCode) { [int]$spec.expectedExitCode } else { 0 }
        if ($result.ExitCode -ne $expected) { throw "Projection preparation '$($spec.name)' exited $($result.ExitCode); expected $expected. $($result.Stderr)" }
        foreach ($pattern in @($spec.stdoutContains)) { if ($pattern -and $result.Stdout -notmatch [regex]::Escape([string]$pattern)) { throw "Projection preparation '$($spec.name)' lacked expected output: $pattern" } }
        $name = if ($spec.name) { [string]$spec.name } else { "step-$index" }
        [IO.File]::WriteAllText((Join-Path $evidenceRoot "$name.stdout.txt"), $result.Stdout, $script:Utf8NoBom)
        [IO.File]::WriteAllText((Join-Path $evidenceRoot "$name.stderr.txt"), $result.Stderr, $script:Utf8NoBom)
        $completed.Add([ordered]@{ name = $name; executable = $exe; args = $args; exitCode = $result.ExitCode; stdoutPath = "$name.stdout.txt"; stderrPath = "$name.stderr.txt" })
        $index++
    }
    $dirty = (Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') ([string]$run.workspace)).Stdout.Trim()
    if ($dirty) {
        [void](Invoke-Git @('add', '--all') ([string]$run.workspace))
        [void](Invoke-Git @('-c', 'user.name=Comparison Harness', '-c', 'user.email=harness@markitect.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', 'Refresh declared Markitect projections for comparison context') ([string]$run.workspace))
        $run.candidateCommit = (Invoke-Git @('rev-parse', 'HEAD') ([string]$run.workspace)).Stdout.Trim()
    }
    $run.preparationCommands = @($run.preparationCommands) + @($completed)
    $run.pendingPreparationCommands = @()
    $run.allowedProjectionTargets = $allowedPlanTargets
    $run.compiledInputs = @(Compile-AgentInputs $trial ([string]$run.workspace) ([string]$run.baseCommit) ([string]$run.candidateCommit) 't1-markitect')
    Write-JsonFile $preparedPath $prepared
    Write-Output "Reviewed and applied the bounded t1-markitect projection plan; candidate is $($run.candidateCommit)."
    exit 0
}

if ($BeginExternalAgent) {
    $externalManifestPath = (Resolve-Path -LiteralPath $ExternalFrozenManifest).Path
    $manifest = [IO.File]::ReadAllText($externalManifestPath) | ConvertFrom-Json -AsHashtable
    Assert-FrozenFiles $manifest
    $preparedPath = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    if (-not (Test-Path -LiteralPath $preparedPath -PathType Leaf)) { throw 'RunRoot is not prepared.' }
    $prepared = [IO.File]::ReadAllText($preparedPath) | ConvertFrom-Json -AsHashtable
    if ($prepared.frozenManifestSha256 -ne (Get-Sha256 $externalManifestPath)) { throw 'External-agent manifest differs from preparation.' }
    foreach ($workspaceRecord in $prepared.workspaces) {
        if ($workspaceRecord.arm -eq 'markitect' -and @($workspaceRecord.compiledInputs).Count -eq 0) { throw "All Markitect contexts must be sealed before any agent starts; missing $($workspaceRecord.id)." }
    }
    $run = @($prepared.workspaces | Where-Object { $_.id -eq $RunId }) | Select-Object -First 1
    if (-not $run) { throw "Prepared manifest has no workspace for $RunId." }
    if (@($run.pendingPreparationCommands).Count -gt 0) { throw "$RunId still has reviewed preparation commands pending." }
    if ($run.arm -eq 'markitect' -and @($run.compiledInputs).Count -eq 0) { throw "$RunId task context is not compiled." }
    $runsRoot = Join-Path $script:RunRootFull 'operator/runs'
    if (Test-Path -LiteralPath $runsRoot -PathType Container) {
        foreach ($existingState in Get-ChildItem -LiteralPath $runsRoot -Filter 'external-state.json' -File -Recurse) {
            $state = [IO.File]::ReadAllText($existingState.FullName) | ConvertFrom-Json -AsHashtable
            if ($state.status -eq 'started') { throw "Another fresh-agent run has not been collected: $($state.runId). Keep trials sequential." }
        }
    }
    $workspace = [string]$run.workspace
    $instructions = @(Get-WorkspaceAgentInstructions $workspace)
    if ((ConvertTo-Json -InputObject @($instructions) -Depth 8 -Compress) -ne (ConvertTo-Json -InputObject @($run.automaticAgentInstructions) -Depth 8 -Compress)) { throw 'Automatic AGENTS.md inputs changed since preparation.' }
    $assetChanges = @(Compare-FrozenWorkspaceAssets @($run.frozenWorkspaceAssets) $workspace)
    if ($assetChanges.Count -gt 0) { throw 'Frozen helper/context assets changed before agent start.' }
    $compiledChanges = @(Compare-CompiledInputs @($run.compiledInputs) $workspace)
    if ($compiledChanges.Count -gt 0) { throw 'Compiled task context changed before agent start.' }
    if ((Invoke-Git @('rev-parse', 'HEAD') $workspace).Stdout.Trim() -ne [string]$run.candidateCommit -or (Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $workspace).Stdout.Trim()) { throw 'Prepared workspace has uncommitted or unexpected changes before agent start.' }
    Assert-Hash (Join-Path $workspace '.tools/read.ps1') ([string]$run.readHelperSha256) 'Prepared read-ledger helper'
    if ($run.arm -eq 'markitect') { Assert-Hash (Join-Path $workspace '.tools/markitect.exe') ([string]$manifest.markitect.sha256) 'Prepared Markitect executable' }
    $runDirectory = Join-Path $runsRoot $RunId
    if (Test-Path -LiteralPath $runDirectory) { throw "RunId already has run evidence: $RunId" }
    New-Item -ItemType Directory -Path $runDirectory -Force | Out-Null
    $before = Get-AuditSnapshotSet @($prepared.auditRoots) @($prepared.workspaces | ForEach-Object { [string]$_.workspace })
    Write-JsonFile (Join-Path $runDirectory 'pre-run-audit.json') $before
    $trial = @($manifest.trials | Where-Object { $_.id -eq $run.task }) | Select-Object -First 1
    $prompt = Get-FrozenAgentPrompt $manifest $trial
    [IO.File]::WriteAllText((Join-Path $runDirectory 'input-prompt.txt'), $prompt, $script:Utf8NoBom)
    $state = [ordered]@{
        status = 'started'
        executionMode = 'fresh-collaboration-agent'
        runId = $RunId
        task = [string]$run.task
        arm = [string]$run.arm
        workspace = $workspace
        preparedManifestSha256 = $prepared.frozenManifestSha256
        frozenManifestSha256 = Get-Sha256 $externalManifestPath
        sourceManifestSha256 = [string]$run.sourceManifestSha256
        candidateCommit = [string]$run.candidateCommit
        promptSha256 = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($script:Utf8NoBom.GetBytes($prompt))).ToLowerInvariant()
        promptPath = 'input-prompt.txt'
        automaticAgentInstructions = @($instructions)
        compiledInputs = @($run.compiledInputs)
        frozenWorkspaceAssets = @($run.frozenWorkspaceAssets)
        compiledInputChanges = @($compiledChanges)
        readHelperSha256Before = Get-Sha256 (Join-Path $workspace '.tools/read.ps1')
        markitectExecutableSha256Before = if ($run.arm -eq 'markitect') { Get-Sha256 (Join-Path $workspace '.tools/markitect.exe') } else { $null }
        startedAtUtc = [DateTimeOffset]::UtcNow.ToString('o')
    }
    Write-JsonFile (Join-Path $runDirectory 'external-state.json') $state
    Write-Output "Fresh-agent input frozen for $RunId. Workspace: $workspace. Prompt: $(Join-Path $runDirectory 'input-prompt.txt')"
    exit 0
}

if ($CompleteExternalAgent) {
    $completionManifestPath = (Resolve-Path -LiteralPath $CompletionFrozenManifest).Path
    $manifest = [IO.File]::ReadAllText($completionManifestPath) | ConvertFrom-Json -AsHashtable
    Assert-FrozenFiles $manifest
    $preparedPath = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    $prepared = [IO.File]::ReadAllText($preparedPath) | ConvertFrom-Json -AsHashtable
    if ($prepared.frozenManifestSha256 -ne (Get-Sha256 $completionManifestPath)) { throw 'Completion manifest differs from preparation.' }
    $runDirectory = Join-Path $script:RunRootFull "operator/runs/$RunId"
    $statePath = Join-Path $runDirectory 'external-state.json'
    if (-not (Test-Path -LiteralPath $statePath -PathType Leaf)) { throw "No external-agent run is pending for $RunId." }
    $state = [IO.File]::ReadAllText($statePath) | ConvertFrom-Json -AsHashtable
    if ($state.status -ne 'started' -or $state.runId -ne $RunId) { throw "External-agent state is not pending for $RunId." }
    if ([string]::IsNullOrWhiteSpace($AgentIdentity)) { throw 'AgentIdentity is required to attribute the collected report.' }
    $reportPath = (Resolve-Path -LiteralPath $AgentReport).Path
    $runRootPrefix = $script:RunRootFull.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $reportPath.StartsWith($runRootPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'AgentReport must be stored inside RunRoot.' }
    $workspace = [string]$state.workspace
    $reportCopy = Join-Path $runDirectory 'agent-report.txt'
    Copy-Item -LiteralPath $reportPath -Destination $reportCopy -Force
    $reportText = [IO.File]::ReadAllText($reportPath)
    $after = Get-AuditSnapshotSet @($prepared.auditRoots) @($prepared.workspaces | ForEach-Object { [string]$_.workspace })
    Write-JsonFile (Join-Path $runDirectory 'post-run-audit.json') $after
    $deltas = Get-SnapshotDelta ([IO.File]::ReadAllText((Join-Path $runDirectory 'pre-run-audit.json')) | ConvertFrom-Json -AsHashtable) $after
    Write-JsonFile (Join-Path $runDirectory 'external-tree-deltas.json') $deltas
    $workspaceStatus = Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $workspace -AllowFailure
    $workspaceHead = Invoke-Git @('rev-parse', 'HEAD') $workspace -AllowFailure
    $workspaceDiff = Invoke-Git @('diff', '--binary', 'HEAD') $workspace -AllowFailure
    [IO.File]::WriteAllText((Join-Path $runDirectory 'workspace.status.txt'), $workspaceStatus.Stdout, $script:Utf8NoBom)
    [IO.File]::WriteAllText((Join-Path $runDirectory 'workspace.diff'), $workspaceDiff.Stdout, $script:Utf8NoBom)
    $workspaceBefore = [IO.File]::ReadAllText((Join-Path $runDirectory 'pre-run-audit.json')) | ConvertFrom-Json -AsHashtable
    $workspaceDelta = Get-SnapshotDelta ([ordered]@{$workspace = $workspaceBefore[$workspace]}) ([ordered]@{$workspace = $after[$workspace]})
    $helperPath = Join-Path $workspace '.tools/read.ps1'
    $helperHashAfter = Get-Sha256 $helperPath
    $assetChanges = @(Compare-FrozenWorkspaceAssets @($state.frozenWorkspaceAssets) $workspace)
    $compiledChanges = @(Compare-CompiledInputs @($state.compiledInputs) $workspace)
    $readLedgerPath = Join-Path $workspace '.telemetry/reads.jsonl'
    $readRecords = @()
    if (Test-Path -LiteralPath $readLedgerPath -PathType Leaf) {
        if ((Get-Item -LiteralPath $readLedgerPath).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Read ledger must be a regular workspace file.' }
        Copy-Item -LiteralPath $readLedgerPath -Destination (Join-Path $runDirectory 'reads.jsonl')
        foreach ($line in [IO.File]::ReadAllLines($readLedgerPath)) { try { $readRecords += ,($line | ConvertFrom-Json -AsHashtable) } catch { } }
    }
    $markitectHashAfter = if ($state.arm -eq 'markitect' -and (Test-Path -LiteralPath (Join-Path $workspace '.tools/markitect.exe'))) { Get-Sha256 (Join-Path $workspace '.tools/markitect.exe') } else { $null }
    $evidence = [ordered]@{
        evidenceType = 'agents-md-comparison-fresh-agent-run'
        executionMode = 'fresh-collaboration-agent'
        runId = $RunId
        agentIdentity = $AgentIdentity
        task = $state.task
        arm = $state.arm
        workspace = $workspace
        preparedManifestSha256 = $state.preparedManifestSha256
        frozenManifestSha256 = $state.frozenManifestSha256
        sourceManifestSha256 = $state.sourceManifestSha256
        candidateCommitBeforeAgent = $state.candidateCommit
        candidateCommitAfterAgent = $workspaceHead.Stdout.Trim()
        startedAtUtc = $state.startedAtUtc
        completedAtUtc = [DateTimeOffset]::UtcNow.ToString('o')
        agentReportPath = 'agent-report.txt'
        agentReportSha256 = Get-Sha256 $reportCopy
        agentReportText = $reportText
        automaticAgentInstructions = @($state.automaticAgentInstructions)
        compiledInputs = @($state.compiledInputs)
        frozenWorkspaceAssets = @($state.frozenWorkspaceAssets)
        frozenWorkspaceAssetChanges = @($assetChanges)
        compiledInputChanges = @($compiledChanges)
        model = $null
        modelEvidenceStatus = 'unavailable from collaboration backend metadata'
        toolEvents = $null
        toolUsageEvidenceStatus = 'unavailable; the collaboration backend exposes only the final agent report'
        tokenUsage = $null
        tokenUsageEvidenceStatus = 'unavailable; no CLI JSONL events or metering were collected'
        workspaceStatus = $workspaceStatus.Stdout
        workspaceDelta = @($workspaceDelta)
        workspaceDiffPath = 'workspace.diff'
        workspaceDiffSha256 = Get-Sha256 (Join-Path $runDirectory 'workspace.diff')
        fileReadLedger = [ordered]@{ recordsPath = if (Test-Path -LiteralPath (Join-Path $runDirectory 'reads.jsonl') -PathType Leaf) { 'reads.jsonl' } else { $null }; helperSha256Before = $state.readHelperSha256Before; helperSha256After = $helperHashAfter; helperMutated = $state.readHelperSha256Before -ne $helperHashAfter; helperRecordedContentReads = @($readRecords).Count; records = @($readRecords); limit = 'Only reads through .tools/read.ps1 are logged.' }
        markitectExecutable = if ($state.arm -eq 'markitect') { [ordered]@{ sha256Before = $state.markitectExecutableSha256Before; sha256After = $markitectHashAfter; mutated = $state.markitectExecutableSha256Before -ne $markitectHashAfter } } else { $null }
        externalTreeDeltas = @($deltas | Where-Object { $_.root -ne $workspace })
        externalWriteDetected = @($deltas | Where-Object { $_.root -ne $workspace }).Count -gt 0
        externalWritePolicy = 'Any detected out-of-workspace change is reported and left in place for review.'
    }
    Write-JsonFile (Join-Path $runDirectory 'run-evidence.json') $evidence
    $state.status = 'complete'
    $state.completedAtUtc = [DateTimeOffset]::UtcNow.ToString('o')
    Write-JsonFile $statePath $state
    Write-Output "Collected $RunId from $AgentIdentity. Evidence: $runDirectory"
    exit 0
}

if ($Execute) {
    $frozenPath = (Resolve-Path -LiteralPath $FrozenManifest).Path
    $manifest = [IO.File]::ReadAllText($frozenPath) | ConvertFrom-Json -AsHashtable
    if ([int]$manifest.schemaVersion -ne 1 -or -not $manifest.oracleFreezeSha256) { throw 'Execution requires a frozen schemaVersion 1 manifest with oracleFreezeSha256.' }
    Assert-FrozenFiles $manifest
    $preparedPath = Join-Path $script:RunRootFull 'operator/prepared-manifest.json'
    if (-not (Test-Path -LiteralPath $preparedPath -PathType Leaf)) { throw 'RunRoot is not prepared. Run PrepareOnly first.' }
    $prepared = [IO.File]::ReadAllText($preparedPath) | ConvertFrom-Json -AsHashtable
    if ($prepared.frozenManifestSha256 -ne (Get-Sha256 $frozenPath) -or $prepared.oracleFreezeSha256 -ne $manifest.oracleFreezeSha256) { throw 'The frozen manifest/oracle binding differs from preparation.' }
    foreach ($workspaceRecord in $prepared.workspaces) {
        if ($workspaceRecord.arm -eq 'markitect' -and @($workspaceRecord.compiledInputs).Count -eq 0) { throw "All Markitect contexts must be sealed before any agent starts; missing $($workspaceRecord.id)." }
    }
    $run = @($prepared.workspaces | Where-Object { $_.id -eq $RunId }) | Select-Object -First 1
    if (-not $run) { throw "Prepared manifest has no workspace for $RunId." }
    $workspace = [string]$run.workspace
    if (-not (Test-Path -LiteralPath $workspace -PathType Container)) { throw "Prepared workspace is missing: $workspace" }
    $trialInfo = Get-ManifestTrial $manifest $RunId
    $trial = $trialInfo.Trial
    if ([string]$trial.revision -ne [string]$run.originalRevision) { throw 'Trial revision changed since preparation.' }
    if ($trialInfo.Arm -eq 'markitect') {
        Assert-Hash (Join-Path $workspace '.tools/markitect.exe') ([string]$manifest.markitect.sha256) 'Prepared Markitect executable'
        if (@($run.compiledInputs).Count -eq 0) { throw "$RunId has not had its frozen task context compiled." }
    }
    if ((Invoke-Git @('rev-parse', 'HEAD') $workspace).Stdout.Trim() -ne [string]$run.candidateCommit -or (Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $workspace).Stdout.Trim()) { throw 'Prepared workspace has uncommitted or unexpected changes before agent start.' }
    $compiledInputChangesBefore = @(Compare-CompiledInputs @($run.compiledInputs) $workspace)
    if ($compiledInputChangesBefore.Count -gt 0) { throw 'Compiled task context changed before agent start.' }
    $currentInstructions = @(Get-WorkspaceAgentInstructions $workspace)
    if ((ConvertTo-Json -InputObject @($currentInstructions) -Depth 8 -Compress) -ne (ConvertTo-Json -InputObject @($run.automaticAgentInstructions) -Depth 8 -Compress)) { throw 'Automatic workspace AGENTS.md inputs changed since preparation.' }
    Assert-Hash (Join-Path $workspace '.tools/read.ps1') ([string]$run.readHelperSha256) 'Prepared read-ledger helper'
    $readHelperHashBefore = Get-Sha256 (Join-Path $workspace '.tools/read.ps1')
    $markitectHashBefore = if ($trialInfo.Arm -eq 'markitect') { Get-Sha256 (Join-Path $workspace '.tools/markitect.exe') } else { $null }

    $auditRootsResolved = @($prepared.auditRoots)
    $before = Get-AuditSnapshotSet $auditRootsResolved @($prepared.workspaces | ForEach-Object { [string]$_.workspace })
    $runDirectory = Join-Path $script:RunRootFull "operator/runs/$RunId"
    if (Test-Path -LiteralPath $runDirectory) { throw "RunId already has logs; refusing duplicate execution: $runDirectory" }
    New-Item -ItemType Directory -Path $runDirectory -Force | Out-Null
    Write-JsonFile (Join-Path $runDirectory 'pre-run-audit.json') $before

    $codexCommand = Get-Command codex.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $codexCommand) {
        $codexCommand = Get-Command codex -All -ErrorAction SilentlyContinue | Where-Object { $_.CommandType -eq 'Application' -and [string]$_.Source -match '\.exe$' } | Select-Object -First 1
    }
    if (-not $codexCommand) { throw 'Could not resolve a native codex.exe executable; refusing to launch through a shell shim.' }
    $codexPath = $codexCommand.Source
    $version = Invoke-Native -FilePath $codexPath -ArgumentList @('--version') -AllowFailure
    [IO.File]::WriteAllText((Join-Path $runDirectory 'codex-version.stdout.txt'), $version.Stdout, $script:Utf8NoBom)
    [IO.File]::WriteAllText((Join-Path $runDirectory 'codex-version.stderr.txt'), $version.Stderr, $script:Utf8NoBom)
    $dotnetEvidence = [ordered]@{ executable = $null; repositoryRoot = $null; sourceDirectory = $null }
    $dotnetCommand = Get-Command dotnet -CommandType Application -ErrorAction SilentlyContinue
    if ($dotnetCommand) {
        $dotnetEvidence.executable = $dotnetCommand.Source
        $repoVersion = Invoke-Native -FilePath $dotnetCommand.Source -ArgumentList @('--version') -WorkingDirectory $workspace -AllowFailure
        $dotnetEvidence.repositoryRoot = [ordered]@{ exitCode = $repoVersion.ExitCode; stdout = $repoVersion.Stdout; stderr = $repoVersion.Stderr }
        if (Test-Path -LiteralPath (Join-Path $workspace 'src') -PathType Container) {
            $srcVersion = Invoke-Native -FilePath $dotnetCommand.Source -ArgumentList @('--version') -WorkingDirectory (Join-Path $workspace 'src') -AllowFailure
            $dotnetEvidence.sourceDirectory = [ordered]@{ exitCode = $srcVersion.ExitCode; stdout = $srcVersion.Stdout; stderr = $srcVersion.Stderr }
        }
    }

    $prompt = Get-FrozenAgentPrompt $manifest $trial
    $promptPath = Join-Path $runDirectory 'input-prompt.txt'
    [IO.File]::WriteAllText($promptPath, $prompt, $script:Utf8NoBom)
    $codexArgs = @('exec', '--ignore-user-config', '--ephemeral', '--json', '--sandbox', 'workspace-write', '--cd', $workspace)
    $started = [DateTimeOffset]::UtcNow
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $agentResult = Invoke-Native -FilePath $codexPath -ArgumentList $codexArgs -WorkingDirectory $workspace -StandardInput $prompt -AllowFailure
    $timer.Stop()
    $finished = [DateTimeOffset]::UtcNow
    [IO.File]::WriteAllText((Join-Path $runDirectory 'codex.stdout.jsonl'), $agentResult.Stdout, $script:Utf8NoBom)
    [IO.File]::WriteAllText((Join-Path $runDirectory 'codex.stderr.txt'), $agentResult.Stderr, $script:Utf8NoBom)

    $summary = Read-EventSummary $agentResult.Stdout
    $after = Get-AuditSnapshotSet $auditRootsResolved @($prepared.workspaces | ForEach-Object { [string]$_.workspace })
    Write-JsonFile (Join-Path $runDirectory 'post-run-audit.json') $after
    $deltas = Get-SnapshotDelta $before $after
    Write-JsonFile (Join-Path $runDirectory 'external-tree-deltas.json') $deltas
    $workspaceFiles = $after[$workspace].files
    $workspaceDelta = Get-SnapshotDelta ([ordered]@{$workspace = $before[$workspace]}) ([ordered]@{$workspace = $after[$workspace]})
    $workspaceStatus = Invoke-Git @('status', '--porcelain=v1', '--untracked-files=all') $workspace -AllowFailure
    $workspaceHead = Invoke-Git @('rev-parse', 'HEAD') $workspace -AllowFailure
    $workspaceDiff = Invoke-Git @('diff', '--binary', 'HEAD') $workspace -AllowFailure
    [IO.File]::WriteAllText((Join-Path $runDirectory 'workspace.status.txt'), $workspaceStatus.Stdout, $script:Utf8NoBom)
    [IO.File]::WriteAllText((Join-Path $runDirectory 'workspace.diff'), $workspaceDiff.Stdout, $script:Utf8NoBom)
    $readLedgerPath = Join-Path $workspace '.telemetry/reads.jsonl'
    $readLedgerRecords = @()
    if (Test-Path -LiteralPath $readLedgerPath -PathType Leaf) {
        Copy-Item -LiteralPath $readLedgerPath -Destination (Join-Path $runDirectory 'reads.jsonl')
        foreach ($line in [IO.File]::ReadAllLines($readLedgerPath)) { try { $readLedgerRecords += ,($line | ConvertFrom-Json -AsHashtable) } catch { } }
    }
    $readHelperHashAfter = Get-Sha256 (Join-Path $workspace '.tools/read.ps1')
    $markitectHashAfter = if ($trialInfo.Arm -eq 'markitect') { Get-Sha256 (Join-Path $workspace '.tools/markitect.exe') } else { $null }
    $compiledInputChangesAfter = @(Compare-CompiledInputs @($run.compiledInputs) $workspace)
    $evidence = [ordered]@{
        evidenceType = 'agents-md-comparison-agent-run'
        runId = $RunId
        arm = $trialInfo.Arm
        task = [string]$trial.task
        sourceRevision = [string]$trial.revision
        sourceManifestSha256 = [string]$run.sourceManifestSha256
        automaticAgentInstructions = @($currentInstructions)
        frozenManifestSha256 = Get-Sha256 $frozenPath
        oracleFreezeSha256 = [string]$manifest.oracleFreezeSha256
        workspace = $workspace
        baseCommit = [string]$run.baseCommit
        candidateCommitBeforeAgent = [string]$run.candidateCommit
        candidateCommitAfterAgent = $workspaceHead.Stdout.Trim()
        finalGitStatusExitCode = $workspaceStatus.ExitCode
        finalDiffExitCode = $workspaceDiff.ExitCode
        finalStatusPath = 'workspace.status.txt'
        finalDiffPath = 'workspace.diff'
        finalGitStatus = $workspaceStatus.Stdout
        finalDiffSha256 = Get-Sha256 (Join-Path $runDirectory 'workspace.diff')
        compiledInputs = @($run.compiledInputs)
        compiledInputChangesBeforeAgent = @($compiledInputChangesBefore)
        compiledInputChangesAfterAgent = @($compiledInputChangesAfter)
        codex = [ordered]@{
            executable = $codexPath
            version = $version.Stdout.Trim()
            versionExitCode = $version.ExitCode
            args = $codexArgs
            config = [ordered]@{ ignoreUserConfig = $true; ephemeral = $true; sandbox = 'workspace-write'; modelOverride = $null }
            exitCode = $agentResult.ExitCode
            startedAtUtc = $started.ToString('o')
            finishedAtUtc = $finished.ToString('o')
            wallTimeMilliseconds = $timer.ElapsedMilliseconds
            stdoutPath = 'codex.stdout.jsonl'
            stderrPath = 'codex.stderr.txt'
            promptPath = 'input-prompt.txt'
            eventSummary = $summary
        }
        dotnetSdk = $dotnetEvidence
        workspaceDelta = @($workspaceDelta)
        fileReadLedger = [ordered]@{
            recordsPath = if (Test-Path -LiteralPath (Join-Path $runDirectory 'reads.jsonl') -PathType Leaf) { 'reads.jsonl' } else { $null }
            helperSha256Before = $readHelperHashBefore
            helperSha256After = $readHelperHashAfter
            helperMutated = $readHelperHashBefore -ne $readHelperHashAfter
            helperRecordedContentReads = @($readLedgerRecords).Count
            records = @($readLedgerRecords)
            limit = 'Records only reads performed through .tools/read.ps1; searches and direct/automatic reads are not represented as content reads.'
        }
        markitectExecutable = if ($trialInfo.Arm -eq 'markitect') { [ordered]@{ sha256Before = $markitectHashBefore; sha256After = $markitectHashAfter; mutated = $markitectHashBefore -ne $markitectHashAfter } } else { $null }
        externalTreeDeltas = @($deltas | Where-Object { $_.root -ne $workspace })
        externalWriteDetected = @($deltas | Where-Object { $_.root -ne $workspace }).Count -gt 0
        completionEvidence = 'Raw JSONL events are retained. Token usage and model are copied only when present in structured CLI events; no self-reported usage is treated as metering.'
    }
    Write-JsonFile (Join-Path $runDirectory 'run-evidence.json') $evidence
    Write-Output "Finished $RunId with codex exit $($agentResult.ExitCode); elapsed $($timer.ElapsedMilliseconds) ms. Evidence: $runDirectory"
}
