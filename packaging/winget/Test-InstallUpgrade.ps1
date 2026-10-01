[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $WingetBinary,

    [Parameter(Mandatory = $true)]
    [string] $OutputDirectory,

    [switch] $AllowPackageReplacement,

    [switch] $LeaveInstalled
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
$portablePackageDirectory = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages\GlaciusLabs.Markitect__DefaultSource'
$transcriptPath = Join-Path $OutputDirectory 'test-run.log'
$summaryPath = Join-Path $OutputDirectory 'test-summary.yaml'
$script:commandRecords = @()
$script:completedPhases = @()
$testStartedAt = Get-Date
$testStatus = 'not_started'
$failureMessage = $null
$cleanupFailure = $null
$restoreFailure = $null
$summaryFailure = $null

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
New-Item -ItemType Directory -Path $oldManifestDirectory -Force | Out-Null

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
    $startedAt = Get-Date
    $output = @(& $WingetBinary list --name Markitect --exact --disable-interactivity 2>&1)
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
    $registrationIds = @(
        foreach ($line in $Inventory) {
            if ([string]$line -match '^\s*Markitect\s+(\S*GlaciusLabs\.Markitect__DefaultSource)\s+\S+\s*$') {
                $Matches[1]
            }
        }
    )
    if ($registrationIds.Count -gt 1) {
        throw "Found multiple WinGet registrations for $packageId; refusing to choose one."
    }
    if ($registrationIds.Count -eq 0) { return $null }
    return $registrationIds[0]
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
    param([Parameter(Mandatory = $true)][string] $ExpectedVersion)
    $inventory = (Get-MarkitectInventory) -join "`n"
    if ($inventory -notmatch "(?im)^\s*Markitect\s+.*GlaciusLabs\.Markitect__DefaultSource\s+$([regex]::Escape($ExpectedVersion))\s*$") {
        throw "WinGet inventory does not show Markitect $ExpectedVersion. Output:`n$inventory"
    }
    $exe = Join-Path $portablePackageDirectory 'markitect.exe'
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
        throw "Portable package executable was not installed: $exe"
    }
    $versionOutput = (& $exe version | Out-String).Trim()
    $versionExit = $LASTEXITCODE
    if ($versionExit -ne 0 -or $versionOutput -ne "Markitect $ExpectedVersion (windows/amd64)") {
        throw "Unexpected Markitect executable result (exit $versionExit): $versionOutput"
    }
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User') -split ';'
    if ($portablePackageDirectory -notin $userPath) {
        throw "WinGet did not add the portable package directory to the user PATH: $portablePackageDirectory"
    }
    Write-Host "Verified Markitect $ExpectedVersion at $exe"
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

    $inventoryLines = Get-MarkitectInventory
    $existingRegistrationId = Get-MarkitectRegistrationId -Inventory $inventoryLines
    if ($existingRegistrationId) {
        if (-not $AllowPackageReplacement) {
            throw 'GlaciusLabs.Markitect is already installed. Remove it manually or pass -AllowPackageReplacement only on a disposable test runner.'
        }
        Write-Host "Removing only existing WinGet registration $existingRegistrationId because -AllowPackageReplacement was passed."
        Invoke-WinGet -Arguments @('uninstall', '--id', $existingRegistrationId, '--exact', '--disable-interactivity') | Out-Null
    }
    $script:completedPhases += 'confirmed no pre-existing package or replaced the exact GlaciusLabs.Markitect registration'

    New-TestOlderManifest
    Invoke-WinGet -Arguments @('validate', '--manifest', $oldManifestDirectory) | Out-Null
    Invoke-WinGet -Arguments @('validate', '--manifest', $manifestDirectory) | Out-Null
    $script:completedPhases += 'validated exact submitted 0.5.0 manifests and temporary 0.4.1 manifests'

    $settingChanged = -not $originalLocalManifestSetting
    Invoke-WinGet -Arguments @('settings', '--enable', 'LocalManifestFiles') | Out-Null
    $script:completedPhases += 'enabled LocalManifestFiles for local manifests'
    $packageCreatedByTest = $true
    Invoke-WinGet -Arguments @('install', '--manifest', $oldManifestDirectory, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity') | Out-Null
    Assert-InstalledVersion -ExpectedVersion $olderVersion
    $script:completedPhases += 'installed and verified published v0.4.1'

    Invoke-WinGet -Arguments @('upgrade', '--manifest', $manifestDirectory, '--scope', 'user', '--accept-package-agreements', '--accept-source-agreements', '--disable-interactivity') | Out-Null
    Assert-InstalledVersion -ExpectedVersion $submittedVersion
    $script:completedPhases += 'upgraded and verified submitted v0.5.0'
    $testStatus = 'passed'
    Write-Host 'WinGet manifest validation, per-user install, and 0.4.1-to-0.5.0 upgrade passed.'
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
                Invoke-WinGet -Arguments @('uninstall', '--id', $installedRegistrationId, '--exact', '--disable-interactivity') | Out-Null
                $script:completedPhases += 'removed test-created package registration'
            }
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
            testedVersions = @($olderVersion, $submittedVersion)
            status = $testStatus
            startedAt = $testStartedAt.ToString('o')
            finishedAt = (Get-Date).ToString('o')
            originalLocalManifestFiles = $originalLocalManifestSetting
            localManifestFilesRestored = if ($null -eq $originalLocalManifestSetting) { $false } else { (Get-LocalManifestFilesEnabled) -eq $originalLocalManifestSetting }
            retainedInstalledPackage = [bool]($LeaveInstalled -and $testStatus -eq 'passed')
            outputDirectory = $OutputDirectory
            completedPhases = @($script:completedPhases)
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
