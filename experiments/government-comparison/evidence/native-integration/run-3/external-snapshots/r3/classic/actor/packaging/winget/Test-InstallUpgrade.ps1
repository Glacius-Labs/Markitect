#requires -Version 7.2
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $WingetBinary,

    [Parameter(Mandatory = $true)]
    [string] $OutputDirectory,

    [switch] $AllowPackageReplacement,

    [switch] $LeaveInstalled,

    [ValidateSet('local', 'catalog-install', 'catalog-upgrade')]
    [string] $Mode = 'local'
)

$ErrorActionPreference = 'Stop'

$packageId = 'GlaciusLabs.Markitect'
$submittedVersion = '0.5.0'
$submittedSha256 = '6F5B98DAD2EA6003BC0368800691AF70084A44D2C225C26FE12D120BCDA27178'
$submittedUrl = 'https://github.com/Glacius-Labs/Markitect/releases/download/v0.5.0/markitect-v0.5.0-windows-amd64.exe'
$olderVersion = '0.4.1'
$olderSha256 = '419F61E99A3EE09C4D87624B44755BACBCB796244CFFE7A1FA148DA4C79A35F1'
$olderUrl = 'https://github.com/Glacius-Labs/Markitect/releases/download/v0.4.1/markitect-v0.4.1-windows-amd64.exe'
$manifestDirectory = Join-Path $PSScriptRoot 'GlaciusLabs.Markitect\0.5.0'
$oldManifestDirectory = Join-Path $OutputDirectory 'GlaciusLabs.Markitect-0.4.1-test'
$upgradeManifestDirectory = Join-Path $OutputDirectory 'GlaciusLabs.Markitect-0.5.0-upgrade-test'
$catalogMode = $Mode -ne 'local'
$localProductCode = 'GlaciusLabs.Markitect__DefaultSource'
$publicSourceIdentifier = 'Microsoft.Winget.Source_8wekyb3d8bbwe'
$publicProductCode = "$($packageId)_$publicSourceIdentifier"
if ($catalogMode) {
    if ($AllowPackageReplacement -or $LeaveInstalled) {
        throw 'Catalog tests require a fresh disposable runner and always uninstall; replacement and retention are forbidden.'
    }
    if (Test-Path -LiteralPath $OutputDirectory) { throw 'Choose a fresh output directory for catalog evidence.' }
    $olderVersion = '0.5.0'
    $olderSha256 = $submittedSha256
    $olderUrl = $submittedUrl
    if ($Mode -eq 'catalog-upgrade') {
        $submittedVersion = '0.7.0'
        $submittedSha256 = 'D797DBBD26D2C40B819B1519DD554701C407505091C9494BAA30FF1447A4BE0F'
        $submittedUrl = 'https://github.com/Glacius-Labs/Markitect/releases/download/v0.7.0/markitect-v0.7.0-windows-amd64.exe'
    }
    $manifestDirectory = Join-Path $PSScriptRoot "GlaciusLabs.Markitect/$submittedVersion"
}
$portablePackageDirectory = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\GlaciusLabs.Markitect__DefaultSource'
if ($catalogMode) {
    $portablePackageDirectory = Join-Path $env:LOCALAPPDATA "Microsoft/WinGet/Packages/$publicProductCode"
}
$portableLinksDirectory = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Links'
$portableAlias = Join-Path $portableLinksDirectory 'markitect.exe'
$transcriptPath = Join-Path $OutputDirectory 'test-run.log'
$summaryPath = Join-Path $OutputDirectory 'test-summary.yaml'
$script:commandRecords = @()
$script:completedPhases = @()
$script:installedChecks = @()
$testStartedAt = Get-Date
$testStatus = 'not_started'
$failureMessage = $null
$cleanupFailure = $null
$restoreFailure = $null
$summaryFailure = $null
$originalUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')

if (-not (Test-Path -LiteralPath $WingetBinary -PathType Leaf)) {
    throw "WinGet executable not found: $WingetBinary"
}
if (-not [System.Security.Principal.WindowsPrincipal]::new([System.Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this script from an elevated PowerShell session. WinGet requires administrator privileges to enable LocalManifestFiles.'
}
if (-not (Test-Path -LiteralPath $manifestDirectory -PathType Container)) {
    throw "Submitted manifest directory not found: $manifestDirectory"
}
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
if (-not $catalogMode) { New-Item -ItemType Directory -Path $oldManifestDirectory -Force | Out-Null }

function Invoke-WinGet {
    param([Parameter(Mandatory = $true)][string[]] $Arguments)
    $startedAt = Get-Date
    $output = @(& $WingetBinary @Arguments 2>&1)
    $exitCode = $LASTEXITCODE
    $output | ForEach-Object { Write-Host $_ }
    $script:commandRecords += [pscustomobject]@{
        command = "winget $($Arguments -join ' ')"
        startedAt = $startedAt.ToString('o')
        exitCode = $exitCode
        output = @($output | ForEach-Object { [string]$_ })
    }
    if ($exitCode -ne 0) {
        throw "WinGet command failed with exit code ${exitCode}: winget $($Arguments -join ' ')"
    }
    return $output
}

function Assert-ManifestContains {
    param(
        [Parameter(Mandatory = $true)][string] $Path,
        [Parameter(Mandatory = $true)][string[]] $ExpectedLines
    )
    $content = Get-Content -LiteralPath $Path -Raw
    foreach ($line in $ExpectedLines) {
        if ($content -notmatch [regex]::Escape($line)) {
            throw "Submitted manifest is missing expected content '$line': $Path"
        }
    }
}

function Get-MarkitectInventory {
    $arguments = @('list', '--name', 'Markitect', '--exact', '--disable-interactivity')
    if ($catalogMode) { $arguments += @('--source', 'winget') }
    $startedAt = Get-Date
    $output = @(& $WingetBinary @arguments 2>&1)
    $exitCode = $LASTEXITCODE
    $script:commandRecords += [pscustomobject]@{
        command = "winget $($arguments -join ' ')"
        startedAt = $startedAt.ToString('o')
        exitCode = $exitCode
        output = @($output | ForEach-Object { [string]$_ })
    }
    if ($exitCode -ne 0 -and ($output -join "`n") -notmatch 'No installed package found matching input criteria') {
        $output | ForEach-Object { Write-Host $_ }
        throw "WinGet inventory query failed with exit code ${exitCode}."
    }
    return $output
}

function Get-MarkitectRegistrationId {
    param([AllowNull()][AllowEmptyCollection()][AllowEmptyString()][string[]] $Inventory)
    $knownIds = @([regex]::Escape($packageId), ('\S*' + [regex]::Escape($localProductCode)), ('\S*' + [regex]::Escape($publicProductCode))) -join '|'
    $registrationIds = @(
        foreach ($line in $Inventory) {
            if ([string]$line -match "^\s*Markitect\s+($knownIds)\s+\S+(?:\s+\S+){0,2}\s*$") {
                $Matches[1]
            } elseif ($catalogMode -and [string]$line -match '^\s*Markitect\s+') {
                throw 'Found a different Markitect package; catalog tests refuse replacement.'
            }
        }
    )
    if ($registrationIds.Count -gt 1) { throw "Found multiple WinGet registrations for $packageId; refusing to choose one." }
    if ($registrationIds.Count -eq 0) { return $null }
    return $registrationIds[0]
}

function Assert-PublicWingetSource {
    param([Parameter(Mandatory = $true)] $Source)
    if ($Source.Name -ne 'winget' -or $Source.Identifier -ne 'Microsoft.Winget.Source_8wekyb3d8bbwe' -or $Source.Arg -ne 'https://cdn.winget.microsoft.com/cache') {
        throw 'The winget source does not match the expected public community catalog.'
    }
}

function Get-CatalogOperationArguments {
    param(
        [ValidateSet('show', 'install', 'upgrade', 'uninstall')][string] $Operation,
        [string] $Version
    )
    $arguments = @($Operation, '--id', $packageId, '--exact', '--source', 'winget', '--accept-source-agreements', '--disable-interactivity')
    if ($Operation -eq 'uninstall') {
        $arguments += '--purge'
    } else {
        if ($Version -notin @('0.5.0', '0.7.0')) { throw 'Only the authorized catalog versions may be tested.' }
        $arguments += @('--version', $Version)
        if ($Operation -in @('install', 'upgrade')) {
            $arguments += @('--scope', 'user', '--accept-package-agreements')
        }
    }
    return $arguments
}

function Get-LocalManifestFilesEnabled {
    $output = @(& $WingetBinary settings export --disable-interactivity 2>&1)
    $exitCode = $LASTEXITCODE
    $script:commandRecords += [pscustomobject]@{
        command = 'winget settings export --disable-interactivity'
        startedAt = (Get-Date).ToString('o')
        exitCode = $exitCode
        output = @($output | ForEach-Object { [string]$_ })
    }
    if ($exitCode -ne 0) { throw "Could not read WinGet settings export (exit $exitCode)." }
    $settings = ($output -join "`n") | ConvertFrom-Json
    if ($null -eq $settings.adminSettings -or $null -eq $settings.adminSettings.LocalManifestFiles) {
        throw 'WinGet settings export did not include adminSettings.LocalManifestFiles.'
    }
    return [bool]$settings.adminSettings.LocalManifestFiles
}

function Assert-InstalledVersion {
    param(
        [Parameter(Mandatory = $true)][string] $ExpectedVersion,
        [Parameter(Mandatory = $true)][string] $ExpectedSha256
    )
    $inventoryLines = Get-MarkitectInventory
    $registrationId = Get-MarkitectRegistrationId -Inventory $inventoryLines
    $inventory = $inventoryLines -join "`n"
    if (-not $registrationId -or $inventory -notmatch "(?im)^\s*Markitect\s+$([regex]::Escape($registrationId))\s+$([regex]::Escape($ExpectedVersion))(?:\s|$)") {
        throw "WinGet inventory does not show Markitect $ExpectedVersion. Output:`n$inventory"
    }
    $exe = Join-Path $portablePackageDirectory 'markitect.exe'
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
        throw "Portable package executable was not installed: $exe"
    }
    if ((Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash -ne $ExpectedSha256) {
        throw "Installed executable does not match the published release SHA-256: $exe"
    }
    $versionOutput = (& $exe version | Out-String).Trim()
    $versionExit = $LASTEXITCODE
    if ($versionExit -ne 0 -or $versionOutput -ne "Markitect $ExpectedVersion (windows/amd64)") {
        throw "Unexpected Markitect executable result (exit $versionExit): $versionOutput"
    }
    # WinGet normally exposes an alias in Links; package-directory PATH is its
    # supported fallback if symlink creation fails.
    $commandPath = $exe
    $commandDirectory = $portablePackageDirectory
    $route = 'package-directory-fallback'
    if (Test-Path -LiteralPath $portableAlias -PathType Leaf) {
        $aliasItem = Get-Item -LiteralPath $portableAlias -Force
        $target = $aliasItem.ResolveLinkTarget($true)
        if ($null -eq $target -or [IO.Path]::GetFullPath($target.FullName) -ne [IO.Path]::GetFullPath($exe)) {
            throw "WinGet alias does not resolve to this package executable: $portableAlias"
        }
        $commandPath = $portableAlias
        $commandDirectory = $portableLinksDirectory
        $route = 'links-alias'
    }
    $userPath = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_.Trim() } | ForEach-Object {
        [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($_.Trim().Trim('"'))).TrimEnd([char[]]'\/')
    })
    if ($commandDirectory.TrimEnd([char[]]'\/') -notin $userPath) {
        throw "WinGet command directory is missing from the persisted user PATH: $commandDirectory"
    }

    # Check a fresh process using persisted PATH without changing the parent
    # session or the user's environment.
    $checkPath = Join-Path $OutputDirectory 'assert-fresh-command.ps1'
    @'
param([string] $ExpectedVersion, [string] $ExpectedSha256, [string] $ExpectedCommandPath)
$ErrorActionPreference = 'Stop'
$env:Path = [Environment]::ExpandEnvironmentVariables((@(
    [Environment]::GetEnvironmentVariable('Path', 'Machine'),
    [Environment]::GetEnvironmentVariable('Path', 'User')
) | Where-Object { $_ }) -join ';')
$command = Get-Command markitect -ErrorAction Stop
if ($command.CommandType -ne 'Application' -or [IO.Path]::GetFullPath($command.Source) -ne [IO.Path]::GetFullPath($ExpectedCommandPath)) {
    throw "Fresh PATH resolves markitect to an unexpected command: $($command.Source)"
}
if ((Get-FileHash -LiteralPath $command.Source -Algorithm SHA256).Hash -ne $ExpectedSha256) {
    throw 'Fresh PATH command does not match the published release SHA-256.'
}
$actual = (& markitect version | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $actual -ne "Markitect $ExpectedVersion (windows/amd64)") {
    throw "Unexpected fresh PATH version: $actual"
}
[pscustomobject]@{ command = $command.Source; version = $actual; sha256 = $ExpectedSha256 } | ConvertTo-Json -Compress
'@ | Set-Content -LiteralPath $checkPath
    $freshResult = @(& (Join-Path $PSHOME 'pwsh.exe') -NoLogo -NoProfile -NonInteractive -File $checkPath -ExpectedVersion $ExpectedVersion -ExpectedSha256 $ExpectedSha256 -ExpectedCommandPath $commandPath)
    if ($LASTEXITCODE -ne 0) { throw 'Fresh-process markitect command verification failed.' }
    $freshCheck = ($freshResult -join [Environment]::NewLine) | ConvertFrom-Json
    $script:installedChecks += [pscustomobject]@{ version = $ExpectedVersion; sha256 = $ExpectedSha256; route = $route; command = $freshCheck.command }
    Write-Host "Verified Markitect $ExpectedVersion via $route at $($freshCheck.command)"
}

function New-TestUpgradeManifest {
    # Local installs use DefaultSource. Before catalog publication, WinGet's
    # composite-source manifest lookup cannot correlate that registration.
    # Bind only this test copy to the known local product code; keep the
    # canonical community manifests unchanged.
    New-Item -ItemType Directory -Path $upgradeManifestDirectory -Force | Out-Null
    foreach ($name in @('GlaciusLabs.Markitect.installer.yaml', 'GlaciusLabs.Markitect.locale.en-US.yaml', 'GlaciusLabs.Markitect.yaml')) {
        Copy-Item -LiteralPath (Join-Path $manifestDirectory $name) -Destination (Join-Path $upgradeManifestDirectory $name)
    }
    $installerPath = Join-Path $upgradeManifestDirectory 'GlaciusLabs.Markitect.installer.yaml'
    $content = Get-Content -LiteralPath $installerPath -Raw
    $content = $content.Replace('InstallerType: portable', "InstallerType: portable" + [Environment]::NewLine + "ProductCode: $localProductCode")
    Set-Content -LiteralPath $installerPath -Value $content -NoNewline
}

function New-TestOlderManifest {
    @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json
PackageIdentifier: $packageId
PackageVersion: $olderVersion
InstallerType: portable
Commands:
- markitect
Installers:
- Architecture: x64
  InstallerUrl: $olderUrl
  InstallerSha256: $olderSha256
ManifestType: installer
ManifestVersion: 1.12.0
"@ | Set-Content -LiteralPath (Join-Path $oldManifestDirectory 'GlaciusLabs.Markitect.installer.yaml')
    @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.defaultLocale.1.12.0.schema.json
PackageIdentifier: $packageId
PackageVersion: $olderVersion
PackageLocale: en-US
Publisher: Glacius Labs
PublisherUrl: https://github.com/Glacius-Labs
PublisherSupportUrl: https://github.com/Glacius-Labs/Markitect/issues
PackageName: Markitect
PackageUrl: https://github.com/Glacius-Labs/Markitect
License: Apache-2.0
LicenseUrl: https://github.com/Glacius-Labs/Markitect/blob/v0.4.1/LICENSE
ShortDescription: Validate and compile AI-facing engineering resources.
Description: Markitect validates typed resources and explicit dependencies, compiles selected context, and measures change impact across Git revisions.
Tags:
- cli
- documentation
- validation
ReleaseNotesUrl: https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.1
ManifestType: defaultLocale
ManifestVersion: 1.12.0
"@ | Set-Content -LiteralPath (Join-Path $oldManifestDirectory 'GlaciusLabs.Markitect.locale.en-US.yaml')
    @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.version.1.12.0.schema.json
PackageIdentifier: $packageId
PackageVersion: $olderVersion
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.12.0
"@ | Set-Content -LiteralPath (Join-Path $oldManifestDirectory 'GlaciusLabs.Markitect.yaml')
}

$packageCreatedByTest = $false
$settingChanged = $false
$originalLocalManifestSetting = $null
$retainedRegistrationId = $null
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
Start-Transcript -Path $transcriptPath -Force | Out-Null

try {
    $testStatus = 'running'
    $originalLocalManifestSetting = Get-LocalManifestFilesEnabled
    $installerManifest = Join-Path $manifestDirectory 'GlaciusLabs.Markitect.installer.yaml'
    $localeManifest = Join-Path $manifestDirectory 'GlaciusLabs.Markitect.locale.en-US.yaml'
    $versionManifest = Join-Path $manifestDirectory 'GlaciusLabs.Markitect.yaml'
    Assert-ManifestContains -Path $installerManifest -ExpectedLines @(
        "PackageIdentifier: $packageId",
        "PackageVersion: $submittedVersion",
        "InstallerUrl: $submittedUrl",
        "InstallerSha256: $submittedSha256",
        'ManifestVersion: 1.12.0'
    )
    Assert-ManifestContains -Path $localeManifest -ExpectedLines @(
        "PackageIdentifier: $packageId",
        "PackageVersion: $submittedVersion",
        'ManifestType: defaultLocale',
        'ManifestVersion: 1.12.0'
    )
    Assert-ManifestContains -Path $versionManifest -ExpectedLines @(
        "PackageIdentifier: $packageId",
        "PackageVersion: $submittedVersion",
        'DefaultLocale: en-US',
        'ManifestVersion: 1.12.0'
    )

    if ($catalogMode) {
        $source = (Invoke-WinGet -Arguments @('source', 'export', '--name', 'winget')) -join [Environment]::NewLine | ConvertFrom-Json
        Assert-PublicWingetSource -Source $source
        Invoke-WinGet -Arguments (Get-CatalogOperationArguments -Operation show -Version $olderVersion) | Out-Null
        if ($Mode -eq 'catalog-upgrade') {
            Invoke-WinGet -Arguments (Get-CatalogOperationArguments -Operation show -Version $submittedVersion) | Out-Null
        }
        $script:completedPhases += 'confirmed authorized versions are indexed in the public winget source'
        $packageRoot = Join-Path $env:LOCALAPPDATA 'Microsoft/WinGet/Packages'
        if (Get-ChildItem -LiteralPath $packageRoot -Filter "$($packageId)_*" -ErrorAction SilentlyContinue) {
            throw 'A Markitect portable package directory already exists; use a fresh disposable runner.'
        }
    }
    $inventoryLines = Get-MarkitectInventory
    $existingRegistrationId = Get-MarkitectRegistrationId -Inventory $inventoryLines
    if ($existingRegistrationId) {
        if (-not $AllowPackageReplacement) {
            throw 'GlaciusLabs.Markitect is already installed. Remove it manually or pass -AllowPackageReplacement only on a disposable test runner.'
        }
        Write-Host "Removing only existing WinGet registration $existingRegistrationId because -AllowPackageReplacement was passed."
        Invoke-WinGet -Arguments @('uninstall', '--id', $existingRegistrationId, '--exact', '--disable-interactivity') | Out-Null
    }
    if (Get-Item -LiteralPath $portableAlias -Force -ErrorAction SilentlyContinue) {
        throw "A command alias already exists without a removable Markitect registration; refusing to replace it: $portableAlias"
    }
    $script:completedPhases += 'confirmed no pre-existing package or replaced the exact GlaciusLabs.Markitect registration'

    if ($catalogMode) {
        Invoke-WinGet -Arguments @('validate', '--manifest', $manifestDirectory) | Out-Null
        $packageCreatedByTest = $true
        Invoke-WinGet -Arguments (Get-CatalogOperationArguments -Operation install -Version $olderVersion) | Out-Null
        Assert-InstalledVersion -ExpectedVersion $olderVersion -ExpectedSha256 $olderSha256
        $script:completedPhases += "installed and verified public catalog v$olderVersion"
        if ($Mode -eq 'catalog-upgrade') {
            Invoke-WinGet -Arguments (Get-CatalogOperationArguments -Operation upgrade -Version $submittedVersion) | Out-Null
            Assert-InstalledVersion -ExpectedVersion $submittedVersion -ExpectedSha256 $submittedSha256
            $script:completedPhases += "upgraded and verified public catalog v$submittedVersion"
        }
    } else {
        New-TestOlderManifest
        New-TestUpgradeManifest
        Invoke-WinGet -Arguments @('validate', '--manifest', $oldManifestDirectory) | Out-Null
        Invoke-WinGet -Arguments @('validate', '--manifest', $manifestDirectory) | Out-Null
        Invoke-WinGet -Arguments @('validate', '--manifest', $upgradeManifestDirectory) | Out-Null
        $script:completedPhases += 'validated exact submitted 0.5.0 manifests and temporary 0.4.1 manifests'

        $settingChanged = -not $originalLocalManifestSetting
        Invoke-WinGet -Arguments @('settings', '--enable', 'LocalManifestFiles') | Out-Null
        $script:completedPhases += 'enabled LocalManifestFiles for local manifests'
        $packageCreatedByTest = $true
        Invoke-WinGet -Arguments @('install', '--manifest', $oldManifestDirectory, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity') | Out-Null
        Assert-InstalledVersion -ExpectedVersion $olderVersion -ExpectedSha256 $olderSha256
        $script:completedPhases += 'installed and verified published v0.4.1'

        Invoke-WinGet -Arguments @('upgrade', '--manifest', $upgradeManifestDirectory, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity') | Out-Null
        Assert-InstalledVersion -ExpectedVersion $submittedVersion -ExpectedSha256 $submittedSha256
        $script:completedPhases += 'upgraded and verified submitted v0.5.0'
    }
    $testStatus = 'passed'
    Write-Host "WinGet $Mode lifecycle passed for $olderVersion and $submittedVersion."
}
catch {
    $testStatus = 'failed'
    $failureMessage = $_.Exception.Message
    throw
}
finally {
    $shouldCleanPackage = $packageCreatedByTest -and (-not $LeaveInstalled -or $testStatus -ne 'passed')
    if ($shouldCleanPackage) {
        try {
            $installedRegistrationId = Get-MarkitectRegistrationId -Inventory (Get-MarkitectInventory)
            if ($installedRegistrationId) {
                $uninstallArguments = if ($catalogMode) { Get-CatalogOperationArguments -Operation uninstall } else { @('uninstall', '--id', $installedRegistrationId, '--exact', '--disable-interactivity') }
                Invoke-WinGet -Arguments $uninstallArguments | Out-Null
                $script:completedPhases += 'removed test-created package registration'
            }
            if (Get-MarkitectRegistrationId -Inventory (Get-MarkitectInventory)) {
                throw 'The test-created package registration remains after uninstall.'
            }
            if ((Test-Path -LiteralPath (Join-Path $portablePackageDirectory 'markitect.exe')) -or (Get-Item -LiteralPath $portableAlias -Force -ErrorAction SilentlyContinue)) {
                throw 'The test-created executable or command alias remains after uninstall.'
            }
            if ($catalogMode) {
                if (Test-Path -LiteralPath $portablePackageDirectory) { throw 'The public portable package directory remains after purge.' }
                $beforePath = @($originalUserPath -split ';' | Where-Object { $_ } | ForEach-Object { [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($_.Trim().Trim('"'))).TrimEnd([char[]]'\/') })
                $afterPath = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ } | ForEach-Object { [IO.Path]::GetFullPath([Environment]::ExpandEnvironmentVariables($_.Trim().Trim('"'))).TrimEnd([char[]]'\/') })
                foreach ($directory in @($portableLinksDirectory, $portablePackageDirectory)) {
                    if ($directory -notin $beforePath -and $directory -in $afterPath) { throw "A test-created PATH entry remains after uninstall: $directory" }
                }
                $script:completedPhases += 'verified public package directory and test-created PATH entries were removed'
            }
            $script:completedPhases += 'verified test-created executable and alias were removed'
        } catch {
            $cleanupFailure = $_.Exception.Message
            Write-Warning "Could not remove the test-created WinGet package registration: $cleanupFailure"
        }
    }

    if ($settingChanged) {
        $restoreArguments = if ($originalLocalManifestSetting) {
            @('settings', '--enable', 'LocalManifestFiles')
        } else {
            @('settings', '--disable', 'LocalManifestFiles')
        }
        try {
            Invoke-WinGet -Arguments $restoreArguments | Out-Null
            $restoredSetting = Get-LocalManifestFilesEnabled
            if ($restoredSetting -ne $originalLocalManifestSetting) {
                throw 'LocalManifestFiles did not return to its original value.'
            }
            $script:completedPhases += "restored LocalManifestFiles to $originalLocalManifestSetting"
        } catch {
            $restoreFailure = $_.Exception.Message
            Write-Warning "Could not restore LocalManifestFiles to its original state: $restoreFailure"
        }
    }

    if ($cleanupFailure -or $restoreFailure) {
        $testStatus = 'failed'
        $failureMessage = (@($failureMessage, $cleanupFailure, $restoreFailure) | Where-Object { $_ }) -join ' | '
    }
    try {
        [pscustomobject]@{
            packageIdentifier = $packageId
            mode = $Mode
            testedVersions = @(@($olderVersion, $submittedVersion) | Select-Object -Unique)
            status = $testStatus
            startedAt = $testStartedAt.ToString('o')
            finishedAt = (Get-Date).ToString('o')
            originalLocalManifestFiles = $originalLocalManifestSetting
            localManifestFilesRestored = if ($null -eq $originalLocalManifestSetting) { $false } else { (Get-LocalManifestFilesEnabled) -eq $originalLocalManifestSetting }
            retainedInstalledPackage = [bool]($LeaveInstalled -and $testStatus -eq 'passed')
            outputDirectory = $OutputDirectory
            completedPhases = @($script:completedPhases)
            installedChecks = @($script:installedChecks)
            localUpgradeTestProductCode = if ($catalogMode) { $null } else { $localProductCode }
            publicSourceIdentifier = if ($catalogMode) { $publicSourceIdentifier } else { $null }
            userPathBefore = $originalUserPath
            userPathAfter = [Environment]::GetEnvironmentVariable('Path', 'User')
            publicCatalogInstallationTested = [bool]($catalogMode -and $testStatus -eq 'passed')
            publicCatalogUpgradeTested = [bool]($Mode -eq 'catalog-upgrade' -and $testStatus -eq 'passed')
            failure = $failureMessage
            cleanupFailure = $cleanupFailure
            settingRestoreFailure = $restoreFailure
            commands = @($script:commandRecords)
        } | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $summaryPath
    } catch {
        $summaryFailure = $_.Exception.Message
        Write-Warning "Could not write test summary: $summaryFailure"
    }
    Stop-Transcript | Out-Null
    if ($cleanupFailure -or $restoreFailure -or $summaryFailure) {
        $postTestFailures = @($cleanupFailure, $restoreFailure, $summaryFailure) | Where-Object { $_ }
        throw "WinGet test did not complete cleanly: $($postTestFailures -join ' | ')"
    }
}
