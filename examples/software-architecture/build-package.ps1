param(
    [ValidateSet('1.0.0', '1.1.0', '2.0.0', '2.1.0')]
    [string] $Version = '1.1.0'
)

$ErrorActionPreference = 'Stop'
$projectRoot = $PSScriptRoot
$repositoryRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$sourceDirectory = switch ($Version) {
    '1.0.0' { 'architecture-package-v1' }
    '1.1.0' { 'architecture-package-v1.1.0' }
    '2.0.0' { 'architecture-package-v2' }
    '2.1.0' { 'architecture-package-v2.1.0' }
}
$sourceCoordinate = "fixture:software-architecture-package-$Version"
$sourceRoot = Join-Path $projectRoot $sourceDirectory
$packageVersion = $Version
$archivePath = Join-Path $projectRoot ".markitect/packages/software-architecture-$packageVersion.zip"
if (Test-Path -LiteralPath $archivePath) { throw "Refusing to overwrite existing archive: $archivePath" }

$temporaryBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$temporaryRoot = Join-Path $temporaryBase ("markitect-software-architecture-" + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $temporaryRoot
$validateTemporaryRoot = {
    $full = [IO.Path]::GetFullPath($temporaryRoot)
    $base = $temporaryBase.TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if (-not $full.StartsWith($base, [StringComparison]::OrdinalIgnoreCase) -or
        -not [IO.Path]::GetFileName($full).StartsWith('markitect-software-architecture-', [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to use a temporary path outside the expected prefix: $full"
    }
    $item = Get-Item -LiteralPath $full -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Refusing reparse-point temporary directory: $full" }
    return $full
}
try {
    $temporaryRoot = & $validateTemporaryRoot
    Get-ChildItem -LiteralPath $sourceRoot -Force | Copy-Item -Destination $temporaryRoot -Recurse
    git -C $temporaryRoot init -b main | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not initialize isolated package-source repository.' }
    git -C $temporaryRoot config user.name 'Markitect software architecture fixture' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not configure isolated package-source author name.' }
    git -C $temporaryRoot config user.email 'fixture@example.invalid' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not configure isolated package-source author email.' }
    git -C $temporaryRoot add --all | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not stage isolated package-source fixture.' }
    git -C $temporaryRoot commit -m "Build local software architecture package $packageVersion" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not commit isolated package-source fixture.' }
    $revision = (git -C $temporaryRoot rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $revision.Length -ne 40) { throw 'Could not resolve isolated package-source snapshot.' }

    Push-Location $repositoryRoot
    try {
        $env:GOCACHE = Join-Path $env:TEMP 'markitect-software-architecture-gocache'
        go run ./src/cmd/markitect-legacy pack --repo $temporaryRoot --revision $revision --output $archivePath
        if ($LASTEXITCODE -ne 0) { throw 'Markitect could not build the package archive.' }
    }
    finally { Pop-Location }
    $digest = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    Write-Output "archive: .markitect/packages/software-architecture-$packageVersion.zip"
    Write-Output "sha256: $digest"
    Write-Output "pin source: $sourceCoordinate (local fixture coordinate; not the temporary Git revision)"
}
finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        $safeTemporaryRoot = & $validateTemporaryRoot
        Remove-Item -LiteralPath $safeTemporaryRoot -Recurse -Force
    }
}
