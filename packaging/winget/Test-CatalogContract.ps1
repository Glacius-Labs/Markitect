#requires -Version 7.2
$ErrorActionPreference = 'Stop'

# Load only the pure contract functions; do not execute runner setup or WinGet.
$runner = Join-Path $PSScriptRoot 'Test-InstallUpgrade.ps1'
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($runner, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors.Message -join [Environment]::NewLine) }
foreach ($name in @('Get-MarkitectRegistrationId', 'Get-CatalogOperationArguments', 'Assert-PublicWingetSource')) {
    $definitions = @($ast.FindAll({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name }, $true))
    if ($definitions.Count -ne 1) { throw "Expected one contract function: $name" }
    . ([scriptblock]::Create($definitions[0].Extent.Text))
}

$publicSource = [pscustomobject]@{ Name = 'winget'; Identifier = 'Microsoft.Winget.Source_8wekyb3d8bbwe'; Arg = 'https://cdn.winget.microsoft.com/cache' }
Assert-PublicWingetSource -Source $publicSource
foreach ($invalidSource in @(
    [pscustomobject]@{ Name = 'winget'; Identifier = 'private'; Arg = 'https://cdn.winget.microsoft.com/cache' }
    [pscustomobject]@{ Name = 'winget'; Identifier = 'Microsoft.Winget.Source_8wekyb3d8bbwe'; Arg = 'https://example.invalid/cache' }
)) {
    $rejected = $false
    try { Assert-PublicWingetSource -Source $invalidSource } catch { $rejected = $true }
    if (-not $rejected) { throw 'A different source identity or URL was accepted.' }
}

$packageId = 'GlaciusLabs.Markitect'
$localProductCode = 'GlaciusLabs.Markitect__DefaultSource'
$publicProductCode = 'GlaciusLabs.Markitect_Microsoft.Winget.Source_8wekyb3d8bbwe'
$catalogMode = $false
$localId = "ARP\User\X64\$localProductCode"
if ((Get-MarkitectRegistrationId @("Markitect $localId 0.4.1")) -ne $localId) { throw 'Local ARP registration was not recognized.' }
$catalogMode = $true
foreach ($row in @(
    'Markitect GlaciusLabs.Markitect 0.5.0 winget',
    'Markitect GlaciusLabs.Markitect 0.5.0 0.7.0 winget',
    "Markitect ARP\User\X64\$publicProductCode 0.5.0"
)) {
    if (-not (Get-MarkitectRegistrationId @($row))) { throw "Public registration was not recognized: $row" }
}
if (Get-MarkitectRegistrationId @('Name Id Version Source', '--- ---- ------- ------')) { throw 'Inventory headers were recognized as a package.' }
foreach ($case in @(
    [pscustomobject]@{ Rows = @('Markitect GlaciusLabs.MarkitectEvil 0.5.0 winget') }
    [pscustomobject]@{ Rows = @("Markitect $localId 0.4.1", 'Markitect GlaciusLabs.Markitect 0.5.0 winget') }
)) {
    $rejected = $false
    try { Get-MarkitectRegistrationId -Inventory $case.Rows | Out-Null } catch { $rejected = $true }
    if (-not $rejected) { throw 'An ambiguous or different package was accepted.' }
}

foreach ($operation in @('show', 'install', 'upgrade', 'uninstall')) {
    $arguments = @(Get-CatalogOperationArguments -Operation $operation -Version '0.5.0')
    $expectedValues = @{ '--id' = $packageId; '--source' = 'winget' }
    foreach ($flag in $expectedValues.Keys) {
        $index = [Array]::IndexOf($arguments, $flag)
        if ($index -lt 0 -or $arguments[$index + 1] -ne $expectedValues[$flag]) { throw "Wrong $flag for $operation" }
    }
    if ('--exact' -notin $arguments -or '--disable-interactivity' -notin $arguments -or '--manifest' -in $arguments) { throw 'Public commands must use exact catalog lookup without local manifests.' }
    if ($operation -eq 'uninstall') {
        if ('--purge' -notin $arguments -or '--version' -in $arguments) { throw 'Catalog uninstall must purge the test package without selecting a version.' }
    } else {
        $index = [Array]::IndexOf($arguments, '--version')
        if ($index -lt 0 -or $arguments[$index + 1] -ne '0.5.0') { throw 'A catalog command omitted its fixed release version.' }
    }
}
$rejected = $false
try { Get-CatalogOperationArguments -Operation install -Version '9.9.9' | Out-Null } catch { $rejected = $true }
if (-not $rejected) { throw 'An unauthorized catalog version was accepted.' }

foreach ($forbidden in @('AllowPackageReplacement', 'LeaveInstalled')) {
    $parameters = @{ Mode = 'catalog-install'; WingetBinary = $runner; OutputDirectory = (Join-Path $PSScriptRoot 'guard-must-not-create'); $forbidden = $true }
    $rejected = $false
    try { & $runner @parameters } catch {
        $rejected = $_.Exception.Message -match 'replacement and retention are forbidden'
    }
    if (-not $rejected -or (Test-Path -LiteralPath $parameters.OutputDirectory)) { throw 'Catalog replacement/retention was not rejected before creating output.' }
}
$rejected = $false
try { & $runner -Mode catalog-install -WingetBinary $runner -OutputDirectory $PSScriptRoot } catch {
    $rejected = $_.Exception.Message -match 'fresh output directory'
}
if (-not $rejected) { throw 'An existing evidence directory was accepted.' }

foreach ($version in @('0.5.0', '0.7.0')) {
    $expectedSha = if ($version -eq '0.5.0') { '6F5B98DAD2EA6003BC0368800691AF70084A44D2C225C26FE12D120BCDA27178' } else { 'D797DBBD26D2C40B819B1519DD554701C407505091C9494BAA30FF1447A4BE0F' }
    $manifest = Get-Content -LiteralPath (Join-Path $PSScriptRoot "GlaciusLabs.Markitect/$version/GlaciusLabs.Markitect.installer.yaml") -Raw
    if ($manifest -notmatch "(?m)^\s*InstallerSha256: $expectedSha\s*$") { throw "Catalog test digest is inconsistent with the canonical release: $version" }
}
Write-Output 'Catalog command, registration ambiguity, pinned version and digest contracts passed; no installation was executed.'
