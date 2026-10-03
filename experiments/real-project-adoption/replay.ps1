[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $UpstreamRoot,

    [Parameter(Mandatory = $true)]
    [string] $BaselineDestination,

    [Parameter(Mandatory = $true)]
    [string] $AdopterDestination,

    [Parameter(Mandatory = $true)]
    [string] $PackageSourceRoot,

    [Parameter(Mandatory = $true)]
    [string] $MarkitectExe,

    [string] $EvidencePath
)

$ErrorActionPreference = 'Stop'
$expectedUpstreamCommit = '91c8ef24b4cb6ef558c95d8267fa07d68c7059f8'
$expectedMarkitectVersion = '0.12.0'
$expectedBaselineBranch = 'codex/pilot-baseline'
$experimentRoot = $PSScriptRoot
$prepareScript = Join-Path $experimentRoot 'prepare.ps1'
$modelRoot = Join-Path $experimentRoot 'model'
$evidencePathWasProvided = -not [string]::IsNullOrWhiteSpace($EvidencePath)

function Resolve-NewPath([string] $Path, [string] $Label) {
    $resolved = [IO.Path]::GetFullPath($Path)
    if (Test-Path -LiteralPath $resolved) {
        throw "Refusing existing $Label path: $resolved"
    }
    return $resolved
}

function Test-PathAtOrBelow([string] $Candidate, [string] $Parent) {
    $candidateFull = [IO.Path]::GetFullPath($Candidate).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar)
    $parentFull = [IO.Path]::GetFullPath($Parent).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar)
    return $candidateFull.Equals($parentFull, [StringComparison]::OrdinalIgnoreCase) -or
        $candidateFull.StartsWith($parentFull + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)
}

function Assert-LastExitCode([string] $Operation) {
    if ($LASTEXITCODE -ne 0) {
        throw "$Operation failed with exit code $LASTEXITCODE."
    }
}

function Invoke-Git([string[]] $Arguments, [string] $Operation) {
    $output = @(& git @Arguments 2>&1)
    Assert-LastExitCode $Operation
    return (($output | ForEach-Object { [string] $_ }) -join "`n").Trim()
}

function Get-TreeHashes([string] $Root) {
    $rootPath = (Resolve-Path -LiteralPath $Root).Path
    $files = @(Get-ChildItem -LiteralPath $rootPath -Recurse -File | Sort-Object FullName)
    $results = [Collections.Generic.List[object]]::new()
    foreach ($file in $files) {
        $relative = [IO.Path]::GetRelativePath($rootPath, $file.FullName).Replace('\', '/')
        $results.Add([ordered]@{
            path = $relative
            bytes = $file.Length
            sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        })
    }
    return @($results)
}

function Copy-TreeIntoNewDirectory([string] $Source, [string] $Destination) {
    if (-not (Test-Path -LiteralPath $Source -PathType Container)) {
        throw "Required blueprint directory is missing: $Source"
    }
    New-Item -ItemType Directory -Path $Destination | Out-Null
    foreach ($item in Get-ChildItem -LiteralPath $Source -Force) {
        Copy-Item -LiteralPath $item.FullName -Destination $Destination -Recurse
    }
}

function Invoke-Markitect([string[]] $Arguments, [string] $Operation) {
    $output = @(& $script:markitectPath @Arguments 2>&1)
    Assert-LastExitCode $Operation
    return (($output | ForEach-Object { [string] $_ }) -join "`n").Trim()
}

if (-not (Test-Path -LiteralPath $UpstreamRoot -PathType Container)) {
    throw "Upstream checkout is missing: $UpstreamRoot"
}
$upstreamPath = (Resolve-Path -LiteralPath $UpstreamRoot).Path
if (-not (Test-Path -LiteralPath $MarkitectExe -PathType Leaf)) {
    throw "Markitect executable is missing: $MarkitectExe"
}
$markitectPath = (Resolve-Path -LiteralPath $MarkitectExe).Path
$script:markitectPath = $markitectPath
$baselinePath = Resolve-NewPath $BaselineDestination 'baseline destination'
$adopterPath = Resolve-NewPath $AdopterDestination 'adopter destination'
$packageRootPath = Resolve-NewPath $PackageSourceRoot 'package source root'
if (-not $evidencePathWasProvided) {
    $EvidencePath = "$adopterPath.replay-evidence.json"
}
$evidenceFullPath = Resolve-NewPath $EvidencePath 'replay evidence'
$outputPaths = @($baselinePath, $adopterPath, $packageRootPath)
for ($left = 0; $left -lt $outputPaths.Count; $left++) {
    if (Test-PathAtOrBelow $outputPaths[$left] $upstreamPath) {
        throw "Replay output must not be inside the upstream checkout: $($outputPaths[$left])"
    }
    for ($right = $left + 1; $right -lt $outputPaths.Count; $right++) {
        if ((Test-PathAtOrBelow $outputPaths[$left] $outputPaths[$right]) -or
            (Test-PathAtOrBelow $outputPaths[$right] $outputPaths[$left])) {
            throw "Replay output paths must be separate and non-nested: $($outputPaths[$left]) and $($outputPaths[$right])"
        }
    }
}
foreach ($outputPath in $outputPaths) {
    if ((Test-PathAtOrBelow $evidenceFullPath $outputPath) -or
        (Test-PathAtOrBelow $outputPath $evidenceFullPath)) {
        throw "Replay evidence must be outside output directories: $evidenceFullPath"
    }
}

if (-not (Test-Path -LiteralPath $prepareScript -PathType Leaf)) {
    throw "Baseline preparation script is missing: $prepareScript"
}
if (-not (Test-Path -LiteralPath (Join-Path $modelRoot 'markitect.yaml') -PathType Leaf)) {
    throw "Project blueprint is missing under $modelRoot"
}
$versionOutput = Invoke-Markitect @('version') 'Reading Markitect version'
if ($versionOutput -notmatch "^Markitect $([regex]::Escape($expectedMarkitectVersion)) ") {
    throw "Markitect executable must report version $expectedMarkitectVersion; received: $versionOutput"
}
$markitectExeHash = (Get-FileHash -LiteralPath $markitectPath -Algorithm SHA256).Hash.ToLowerInvariant()

# prepare.ps1 owns source selection, byte-preserving copies, and baseline creation.
$baselineParent = Split-Path -Parent $baselinePath
if (-not (Test-Path -LiteralPath $baselineParent -PathType Container)) {
    New-Item -ItemType Directory -Path $baselineParent -Force | Out-Null
}
& $prepareScript -SourceRoot $upstreamPath -Destination $baselinePath
if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) {
    Assert-LastExitCode 'Preparing the pristine baseline'
}
$baselineManifestPath = Join-Path $baselinePath 'source-manifest.json'
$baselineManifest = Get-Content -LiteralPath $baselineManifestPath -Raw | ConvertFrom-Json
if ([string]$baselineManifest.upstreamCommit -ne $expectedUpstreamCommit) {
    throw "Prepared baseline is not bound to expected upstream commit $expectedUpstreamCommit."
}
$baselineBranch = Invoke-Git @('-C', $baselinePath, 'branch', '--show-current') 'Reading baseline branch'
if ($baselineBranch -ne $expectedBaselineBranch) {
    throw "Expected baseline branch $expectedBaselineBranch; found $baselineBranch."
}

# Clone the prepared baseline so the adopter starts with the exact extracted source and its manifest.
$adopterParent = Split-Path -Parent $adopterPath
if (-not (Test-Path -LiteralPath $adopterParent -PathType Container)) {
    New-Item -ItemType Directory -Path $adopterParent -Force | Out-Null
}
[void](Invoke-Git @('clone', '--local', '--no-hardlinks', '--branch', $expectedBaselineBranch, $baselinePath, $adopterPath) 'Cloning prepared baseline')
[void](Invoke-Git @('-C', $adopterPath, 'switch', '-c', 'codex/pilot-adopter') 'Creating isolated adopter branch')

$packageParent = Split-Path -Parent $packageRootPath
if (-not (Test-Path -LiteralPath $packageParent -PathType Container)) {
    New-Item -ItemType Directory -Path $packageParent -Force | Out-Null
}
New-Item -ItemType Directory -Path $packageRootPath | Out-Null

$packageRecords = [Collections.Generic.List[object]]::new()
$fixedDate = '2026-10-03T00:00:00Z'
foreach ($packageVersion in @(1, 2)) {
    $sourcePackage = Join-Path (Join-Path $modelRoot 'packages') "v$packageVersion"
    $packageDestination = Join-Path $packageRootPath "v$packageVersion"
    if (Test-Path -LiteralPath $packageDestination) {
        throw "Refusing existing package source directory: $packageDestination"
    }
    Copy-TreeIntoNewDirectory $sourcePackage $packageDestination
    $branch = "codex/adoption-constitution-v$packageVersion"
    [void](Invoke-Git @('-C', $packageDestination, 'init', '--initial-branch', $branch) "Initializing package v$packageVersion")
    [void](Invoke-Git @('-C', $packageDestination, 'add', '--all') "Staging package v$packageVersion")

    $previousAuthorDate = $env:GIT_AUTHOR_DATE
    $previousCommitterDate = $env:GIT_COMMITTER_DATE
    try {
        $env:GIT_AUTHOR_DATE = $fixedDate
        $env:GIT_COMMITTER_DATE = $fixedDate
        [void](Invoke-Git @('-C', $packageDestination, '-c', 'user.name=Markitect Adoption Pilot', '-c', 'user.email=pilot@markitect.invalid', '-c', 'commit.gpgsign=false', 'commit', '-m', "Freeze MyMeetings constitution v$packageVersion") "Freezing package v$packageVersion")
    }
    finally {
        if ($null -eq $previousAuthorDate) { Remove-Item Env:GIT_AUTHOR_DATE -ErrorAction SilentlyContinue } else { $env:GIT_AUTHOR_DATE = $previousAuthorDate }
        if ($null -eq $previousCommitterDate) { Remove-Item Env:GIT_COMMITTER_DATE -ErrorAction SilentlyContinue } else { $env:GIT_COMMITTER_DATE = $previousCommitterDate }
    }

    $packageCommit = Invoke-Git @('-C', $packageDestination, 'rev-parse', 'HEAD') "Reading package v$packageVersion commit"
    $actualPackageBranch = Invoke-Git @('-C', $packageDestination, 'branch', '--show-current') "Reading package v$packageVersion branch"
    if ($actualPackageBranch -ne $branch) { throw "Unexpected package branch for v${packageVersion}: $actualPackageBranch" }
    $packageHashes = Get-TreeHashes $packageDestination
    $archiveRelative = ".markitect/packages/mymeetings-architecture-$packageVersion.0.0.zip"
    $archivePath = Join-Path $adopterPath ($archiveRelative.Replace('/', [IO.Path]::DirectorySeparatorChar))
    $archiveParent = Split-Path -Parent $archivePath
    if (-not (Test-Path -LiteralPath $archiveParent -PathType Container)) {
        New-Item -ItemType Directory -Path $archiveParent -Force | Out-Null
    }
    [void](Invoke-Markitect @('pack', '--repo', $packageDestination, '--revision', $packageCommit, '--output', $archivePath) "Packing package v$packageVersion")
    $packageRecords.Add([ordered]@{
        version = "$packageVersion.0.0"
        branch = $actualPackageBranch
        revision = $packageCommit
        contentFiles = $packageHashes
        archive = $archiveRelative
        archiveBytes = (Get-Item -LiteralPath $archivePath).Length
        archiveSha256 = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    })
}

Copy-TreeIntoNewDirectory (Join-Path $modelRoot 'resources') (Join-Path $adopterPath 'resources')
Copy-TreeIntoNewDirectory (Join-Path $modelRoot 'agent-guidance') (Join-Path $adopterPath 'agent-guidance')
$scriptsPath = Join-Path $adopterPath 'scripts'
New-Item -ItemType Directory -Path $scriptsPath | Out-Null
foreach ($checkFile in @('check-module-project-references.ps1', 'project-map.json')) {
    Copy-Item -LiteralPath (Join-Path (Join-Path $experimentRoot 'checks') $checkFile) -Destination $scriptsPath
}
$workItemsPath = Join-Path $adopterPath 'work-items'
New-Item -ItemType Directory -Path $workItemsPath | Out-Null
$taskRequest = Join-Path (Join-Path $experimentRoot 'discovery') 'work-items/copy-me-query-convention.md'
$contextRun = Join-Path (Join-Path $experimentRoot 'discovery') 'context-run.yaml'
Copy-Item -LiteralPath $taskRequest -Destination $workItemsPath
Copy-Item -LiteralPath $contextRun -Destination $adopterPath

# Keep the authored YAML formatting. Only the package digest token is data-dependent;
# the Project-owned check path is asserted against the copied script target.
$projectPath = Join-Path $adopterPath 'markitect.yaml'
$projectText = Get-Content -LiteralPath (Join-Path $modelRoot 'markitect.yaml') -Raw
$v1Archive = Join-Path $adopterPath '.markitect/packages/mymeetings-architecture-1.0.0.zip'
$v1Hash = (Get-FileHash -LiteralPath $v1Archive -Algorithm SHA256).Hash.ToLowerInvariant()
if (-not $projectText.Contains('PLACEHOLDER_FOR_PACKAGE_BUILDER')) {
    throw 'The Project blueprint no longer contains its expected package digest token.'
}
$projectText = $projectText.Replace('PLACEHOLDER_FOR_PACKAGE_BUILDER', $v1Hash)
$checkScriptPath = 'scripts/check-module-project-references.ps1'
if (-not $projectText.Contains($checkScriptPath)) {
    throw "The Project blueprint must configure the copied check at $checkScriptPath."
}
if (-not (Test-Path -LiteralPath (Join-Path $scriptsPath 'check-module-project-references.ps1') -PathType Leaf)) {
    throw 'The configured project check was not copied into the adopter.'
}
[IO.File]::WriteAllText($projectPath, $projectText, [Text.UTF8Encoding]::new($false))

[void](Invoke-Markitect @('format', '--repo', $adopterPath, '--write') 'Formatting adopter blueprint')
$adopterCommit = Invoke-Git @('-C', $adopterPath, 'rev-parse', 'HEAD') 'Reading adopter baseline revision'
$evidenceParent = Split-Path -Parent $evidenceFullPath
if (-not (Test-Path -LiteralPath $evidenceParent -PathType Container)) {
    New-Item -ItemType Directory -Path $evidenceParent -Force | Out-Null
}
$copiedInputs = @(
    (Join-Path $adopterPath 'markitect.yaml'),
    (Join-Path $adopterPath 'context-run.yaml'),
    (Join-Path $workItemsPath 'copy-me-query-convention.md'),
    (Join-Path $scriptsPath 'check-module-project-references.ps1'),
    (Join-Path $scriptsPath 'project-map.json')
)
$adopterInputHashes = [Collections.Generic.List[object]]::new()
foreach ($inputPath in $copiedInputs) {
    $adopterInputHashes.Add([ordered]@{
        path = [IO.Path]::GetRelativePath($adopterPath, $inputPath).Replace('\', '/')
        bytes = (Get-Item -LiteralPath $inputPath).Length
        sha256 = (Get-FileHash -LiteralPath $inputPath -Algorithm SHA256).Hash.ToLowerInvariant()
    })
}
$replayEvidence = [ordered]@{
    evidenceType = 'markitect-real-project-adoption-replay-setup'
    upstreamRepository = [string]$baselineManifest.upstreamRepository
    upstreamCommit = $expectedUpstreamCommit
    sourceManifestSha256 = (Get-FileHash -LiteralPath $baselineManifestPath -Algorithm SHA256).Hash.ToLowerInvariant()
    baselineBranch = $baselineBranch
    baselineCommit = Invoke-Git @('-C', $baselinePath, 'rev-parse', 'HEAD') 'Recording baseline commit'
    adopterBranch = Invoke-Git @('-C', $adopterPath, 'branch', '--show-current') 'Recording adopter branch'
    adopterStartingCommit = $adopterCommit
    markitectVersion = $expectedMarkitectVersion
    markitectExecutableSha256 = $markitectExeHash
    packages = @($packageRecords)
    adopterInputs = @($adopterInputHashes)
    resources = Get-TreeHashes (Join-Path $adopterPath 'resources')
    agentGuidance = Get-TreeHashes (Join-Path $adopterPath 'agent-guidance')
    projectPin = [ordered]@{package = 'mymeetings-architecture'; version = '1.0.0'; archive = '.markitect/packages/mymeetings-architecture-1.0.0.zip'; sha256 = $v1Hash}
    discovery = [ordered]@{taskCopied = 'work-items/copy-me-query-convention.md'; contextRunCopied = 'context-run.yaml'; candidateDecisionEvidenceCopied = $false; automaticallyAdopted = $false}
    status = 'setup-only; no agent run, human acceptance, or adoption is recorded'
}
[IO.File]::WriteAllText($evidenceFullPath, ($replayEvidence | ConvertTo-Json -Depth 12) + "`n", [Text.UTF8Encoding]::new($false))

Write-Output "Upstream commit: $expectedUpstreamCommit"
Write-Output "Markitect version: $expectedMarkitectVersion"
Write-Output "Baseline branch and commit: $baselineBranch / $($replayEvidence.baselineCommit)"
Write-Output "Adopter branch and starting commit: $($replayEvidence.adopterBranch) / $adopterCommit"
Write-Output "Package pins and content hashes: $evidenceFullPath"
Write-Output 'Setup only: discovery remains a copied task prompt and context run; no agent execution or adoption is recorded.'
