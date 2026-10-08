[CmdletBinding()]
param(
    [string] $SourceRoot = (Join-Path (Split-Path -Parent $PSScriptRoot) '..\.artifacts\adoption\upstream'),
    [string] $Destination = (Join-Path (Split-Path -Parent $PSScriptRoot) '..\.artifacts\adoption\baseline-final')
)

$ErrorActionPreference = 'Stop'
$expectedCommit = '91c8ef24b4cb6ef558c95d8267fa07d68c7059f8'
$sourcePath = (Resolve-Path -LiteralPath $SourceRoot).Path
$destinationPath = [IO.Path]::GetFullPath($Destination)

function Assert-LastExitCode([string] $Operation) {
    if ($LASTEXITCODE -ne 0) { throw "$Operation failed with exit code $LASTEXITCODE." }
}

$head = (& git -C $sourcePath rev-parse HEAD).Trim()
Assert-LastExitCode 'Reading upstream HEAD'
if ($head -ne $expectedCommit) { throw "Upstream checkout must be exactly $expectedCommit; found $head." }
$dirtyEntries = @(& git -C $sourcePath status --porcelain)
Assert-LastExitCode 'Checking upstream working tree'
if ($dirtyEntries.Count -ne 0) { throw 'Upstream working tree must be clean; refusing local or untracked source changes.' }
$trackedFiles = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($trackedPath in & git -C $sourcePath ls-files) { [void] $trackedFiles.Add($trackedPath.Replace('\', '/')) }
Assert-LastExitCode 'Listing upstream tracked files'

if (Test-Path -LiteralPath $destinationPath) {
    $existing = @(Get-ChildItem -LiteralPath $destinationPath -Force)
    if ($existing.Count -ne 0) { throw "Refusing to write non-empty destination: $destinationPath" }
} else {
    New-Item -ItemType Directory -Path $destinationPath | Out-Null
}

$selected = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
function Add-SelectedFile([string] $RelativePath) {
    $candidate = Join-Path $sourcePath $RelativePath
    if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) { throw "Required selected source file is missing: $RelativePath" }
    if (-not $trackedFiles.Contains($RelativePath.Replace('\', '/'))) { throw "Selected file is not tracked by the pinned upstream commit: $RelativePath" }
    $full = (Resolve-Path -LiteralPath $candidate).Path
    if (-not $full.StartsWith($sourcePath + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Selected source escaped the upstream checkout: $RelativePath"
    }
    [void] $selected.Add($RelativePath.Replace('\', '/'))
}
function Add-SelectedTree([string] $RelativeRoot, [string[]] $Extensions) {
    $tree = Join-Path $sourcePath $RelativeRoot
    if (-not (Test-Path -LiteralPath $tree -PathType Container)) { throw "Required selected source directory is missing: $RelativeRoot" }
    foreach ($file in Get-ChildItem -LiteralPath $tree -Recurse -File) {
        if ($Extensions -contains $file.Extension.ToLowerInvariant()) {
            Add-SelectedFile ([IO.Path]::GetRelativePath($sourcePath, $file.FullName))
        }
    }
}

Add-SelectedFile 'README.md'
Add-SelectedFile 'LICENSE'
$adrRoot = Join-Path $sourcePath 'docs\architecture-decision-log'
if (-not (Test-Path -LiteralPath $adrRoot -PathType Container)) { throw 'Architecture decision log is missing.' }
foreach ($file in Get-ChildItem -LiteralPath $adrRoot -File -Filter '*.md') {
    Add-SelectedFile ([IO.Path]::GetRelativePath($sourcePath, $file.FullName))
}
foreach ($name in @('global.json', 'Directory.Build.props', 'Directory.Build.targets', 'Directory.Packages.props', '.editorconfig', 'stylecop.json')) {
    Add-SelectedFile (Join-Path 'src' $name)
}
foreach ($module in @('Meetings', 'Payments', 'UserAccess', 'Administration', 'Registrations')) {
    foreach ($area in @('Application', 'Domain', 'IntegrationEvents')) {
        Add-SelectedTree (Join-Path (Join-Path 'src\Modules' $module) $area) @('.cs', '.csproj')
    }
}
foreach ($area in @('Application', 'Domain', 'Infrastructure')) {
    Add-SelectedTree (Join-Path 'src\BuildingBlocks' $area) @('.cs', '.csproj')
}
Add-SelectedFile 'src/Modules/Meetings/Tests/ArchTests/Application/ApplicationTests.cs'
Add-SelectedFile 'src/Database/CompanyName.MyMeetings.Database/Structure/meetings/Tables/MeetingAttendees.sql'
Add-SelectedFile 'src/Database/CompanyName.MyMeetings.Database/Structure/meetings/Views/v_MeetingAttendees.sql'

$orderedPaths = @($selected)
[Array]::Sort($orderedPaths, [StringComparer]::Ordinal)
$records = [Collections.Generic.List[object]]::new()
$selectedByteCount = [long] 0
foreach ($relative in $orderedPaths) {
    $sourceFile = Join-Path $sourcePath ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))
    $targetFile = Join-Path $destinationPath ($relative.Replace('/', [IO.Path]::DirectorySeparatorChar))
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $targetFile) | Out-Null
    [IO.File]::Copy($sourceFile, $targetFile, $false)
    $sourceHash = (Get-FileHash -LiteralPath $sourceFile -Algorithm SHA256).Hash.ToLowerInvariant()
    $copiedHash = (Get-FileHash -LiteralPath $targetFile -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($copiedHash -ne $sourceHash) { throw "Copied source did not preserve exact bytes: $relative" }
    $fileBytes = (Get-Item -LiteralPath $sourceFile).Length
    $selectedByteCount += [long] $fileBytes
    $records.Add([ordered]@{
        path = $relative
        sha256 = $sourceHash
        bytes = $fileBytes
    })
}

$manifest = [ordered]@{
    manifestVersion = 1
    upstreamRepository = 'https://github.com/kgrzybek/modular-monolith-with-ddd'
    upstreamCommit = $expectedCommit
    upstreamRefAtPreparation = 'master'
    license = 'MIT; see LICENSE'
    selectionPolicy = 'experiments/real-project-adoption/source-selection.md'
    selectedFileCount = $records.Count
    selectedBytes = $selectedByteCount
    files = @($records)
}
[IO.File]::WriteAllText((Join-Path $destinationPath 'source-manifest.json'), ($manifest | ConvertTo-Json -Depth 8) + "`n", [Text.UTF8Encoding]::new($false))

$ignoreText = "**/bin/`n**/obj/`n**/TestResults/`n.vs/`n"
[IO.File]::WriteAllText((Join-Path $destinationPath '.gitignore'), $ignoreText, [Text.UTF8Encoding]::new($false))
$snapshotReadme = @"
# Markitect real-project pilot baseline

This is a selected, byte-preserving source snapshot from the MIT-licensed upstream repository recorded in ``source-manifest.json``. The manifest binds every copied upstream file to its SHA-256 and the upstream Git commit. ``source-selection.md`` explains the scope and exclusions.

This baseline intentionally contains no Markitect files, package, context, generated projections, or pilot edits. It is the starting point for the unassisted task run. Build outputs are ignored.
"@
[IO.File]::WriteAllText((Join-Path $destinationPath 'PILOT_BASELINE.md'), $snapshotReadme.Replace("`r`n", "`n") + "`n", [Text.UTF8Encoding]::new($false))

if (Test-Path -LiteralPath (Join-Path $destinationPath '.git')) { throw 'Destination unexpectedly contains Git metadata before baseline initialization.' }
& git -C $destinationPath init --initial-branch=codex/pilot-baseline
Assert-LastExitCode 'Initializing isolated baseline Git repository'
& git -C $destinationPath add --all
Assert-LastExitCode 'Staging isolated baseline files'
& git -C $destinationPath -c user.name='Markitect Adoption Pilot' -c user.email='pilot@markitect.invalid' commit -m 'Create fixed real-project adoption baseline'
Assert-LastExitCode 'Creating isolated baseline commit'

$baselineCommit = (& git -C $destinationPath rev-parse HEAD).Trim()
Assert-LastExitCode 'Reading isolated baseline commit'
$actualBranch = (& git -C $destinationPath branch --show-current).Trim()
Assert-LastExitCode 'Reading isolated baseline branch'
if ($actualBranch -ne 'codex/pilot-baseline') { throw "Unexpected baseline branch: $actualBranch" }
Write-Output "Upstream commit: $head"
Write-Output "Selected upstream files: $($records.Count)"
Write-Output "Selected upstream bytes: $($manifest.selectedBytes)"
Write-Output "Baseline branch: $actualBranch"
Write-Output "Baseline Git commit: $baselineCommit"
Write-Output "Baseline path: $destinationPath"
