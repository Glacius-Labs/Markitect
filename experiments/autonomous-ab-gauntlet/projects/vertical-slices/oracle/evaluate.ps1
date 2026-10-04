param(
    [Parameter(Mandatory = $true)][string]$Repo,
    [Parameter(Mandatory = $true)][ValidatePattern('^(0[1-9]|1[0-2])$')][string]$Task,
    [Parameter(Mandatory = $true)][string]$Base,
    [Parameter(Mandatory = $true)][string]$Output
)
$ErrorActionPreference = 'Stop'
$repoPath = (Resolve-Path -LiteralPath $Repo).Path
$script:checks = [System.Collections.Generic.List[object]]::new()
$script:failures = [System.Collections.Generic.List[string]]::new()
$script:manual = [System.Collections.Generic.List[string]]::new()
$script:commands = [System.Collections.Generic.List[string]]::new()
$script:taskId = [int]$Task
$arm = if (Test-Path -LiteralPath (Join-Path $repoPath 'markitect.yaml')) { 'b' } else { 'a' }

function Add-Check([string]$Name, [string]$Status, [string]$Evidence) {
    $script:checks.Add([pscustomobject]@{ name = $Name; status = $Status; evidence = $Evidence })
    if ($Status -eq 'failed') { $script:failures.Add($Name) }
    if ($Status -eq 'manual') { $script:manual.Add($Name) }
}
function Read-RepoFile([string]$Relative) {
    $path = Join-Path $script:repoPath $Relative
    if (Test-Path -LiteralPath $path -PathType Leaf) { return Get-Content -LiteralPath $path -Raw }
    return ''
}
function Invoke-Repo([string]$Exe, [string[]]$Args) {
    $script:commands.Add(($Exe + ' ' + ($Args -join ' ')))
    Push-Location $script:repoPath
    try { & $Exe @Args 2>&1 | Out-String; return $LASTEXITCODE }
    finally { Pop-Location }
}
function Get-AllowedPaths {
    $yaml = Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\task-set.yaml') -Raw
    $match = [regex]::Match($yaml, '(?ms)^  - id: "' + [regex]::Escape($Task) + '"\r?\n(?<body>.*?)(?=^  - id:|\z)')
    if (-not $match.Success) { throw "Task $Task is missing from task-set.yaml" }
    $body = $match.Groups['body'].Value
    $armMatch = [regex]::Match($body, '(?m)^    allowed_paths_by_arm: \{a: \[(?<a>[^\]]*)\], b: \[(?<b>[^\]]*)\]\}')
    if ($armMatch.Success) { $raw = $armMatch.Groups[$arm].Value }
    else {
        $rawMatch = [regex]::Match($body, '(?m)^    allowed_paths: \[(?<items>[^\]]*)\]')
        $raw = $rawMatch.Groups['items'].Value
    }
    if ([string]::IsNullOrWhiteSpace($raw)) { return @() }
    return @($raw -split ',\s*' | ForEach-Object { $_.Trim().Trim('"').Trim("'").Replace('\','/') } | Where-Object { $_ })
}
function Get-ChangedPaths {
    Push-Location $script:repoPath
    try {
        $script:commands.Add("git diff --name-only $Base plus untracked paths (excluding build caches and Git metadata)")
        $paths = @(& git diff --name-only $Base 2>$null)
        $paths += @(& git ls-files --others --exclude-standard 2>$null)
        $paths += @(& git ls-files --others --ignored --exclude-standard 2>$null)
        return @($paths | ForEach-Object { $_.Replace('\','/') } | Where-Object {
            $path = $_
            $isGitMetadata = $path -match '(^|/)\.git(/|$)'
            $isDeclaredDotNetOutput = @('src/Commerce/bin','src/Commerce/obj','checks/bin','checks/obj') | Where-Object { $path -eq $_ -or $path.StartsWith($_ + '/', [StringComparison]::OrdinalIgnoreCase) }
            $path -and -not $isGitMetadata -and -not $isDeclaredDotNetOutput
        } | Sort-Object -Unique)
    }
    finally { Pop-Location }
}
function Test-IsAllowed([string]$Path, [string[]]$Scopes) {
    foreach ($scope in $Scopes) {
        if ($Path.Equals($scope, [StringComparison]::OrdinalIgnoreCase) -or $Path.StartsWith($scope.TrimEnd('/') + '/', [StringComparison]::OrdinalIgnoreCase)) { return $true }
    }
    return $false
}
function Get-TextAtRevision([string]$Revision, [string]$Path) {
    Push-Location $script:repoPath
    try { $text = & git show ($Revision + ':' + $Path) 2>$null; if ($LASTEXITCODE -eq 0) { return ($text -join "`n") }; return $null }
    finally { Pop-Location }
}
function Test-TaskOneVector {
    $source = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'vectors\task01.cs.txt') -Raw
    $tempRoot = Join-Path ([IO.Path]::GetTempPath()) ('vertical-slices-oracle-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tempRoot | Out-Null
    try {
        $project = @"
<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>net8.0</TargetFramework><ImplicitUsings>enable</ImplicitUsings><Nullable>enable</Nullable></PropertyGroup><ItemGroup><ProjectReference Include="$($script:repoPath.Replace('&','&amp;'))\src\Commerce\Commerce.csproj" /></ItemGroup></Project>
"@
        Set-Content -LiteralPath (Join-Path $tempRoot 'Oracle.csproj') -Value $project -Encoding utf8
        Set-Content -LiteralPath (Join-Path $tempRoot 'NuGet.Config') -Value '<configuration><packageSources><clear /></packageSources></configuration>' -Encoding utf8
        Set-Content -LiteralPath (Join-Path $tempRoot 'Program.cs') -Value $source -Encoding utf8
        $script:commands.Add('dotnet run --project <evaluator-owned-temp>/Oracle.csproj --configuration Release')
        Push-Location $script:repoPath
        try {
            $previousTask = $env:GAUNTLET_TASK
            $env:GAUNTLET_TASK = $Task
            $result = & dotnet run --project (Join-Path $tempRoot 'Oracle.csproj') --configuration Release 2>&1 | Out-String
            $exit = $LASTEXITCODE
            if ($null -eq $previousTask) { Remove-Item Env:GAUNTLET_TASK -ErrorAction SilentlyContinue } else { $env:GAUNTLET_TASK = $previousTask }
            $status = if ($exit -eq 0) { 'passed' } elseif ($exit -eq 2) { 'manual' } else { 'failed' }
            return [pscustomobject]@{ status = $status; evidence = ($result.Trim() -replace '[\r\n]+',' | '); output = $result }
        }
        finally {
            if ($null -eq $previousTask) { Remove-Item Env:GAUNTLET_TASK -ErrorAction SilentlyContinue } else { $env:GAUNTLET_TASK = $previousTask }
            Pop-Location
        }
    }
    finally { Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue }
}

$scopes = Get-AllowedPaths
$baseCommit = ''
Push-Location $repoPath
try {
    $baseCommit = (& git rev-parse --verify ($Base + '^{commit}') 2>$null | Out-String).Trim()
    $baseExit = $LASTEXITCODE
}
finally { Pop-Location }
Add-Check 'immutable-base-available' $(if ($baseExit -eq 0 -and $baseCommit -eq $Base) { 'passed' } else { 'failed' }) $(if ($baseExit -eq 0 -and $baseCommit -eq $Base) { 'Supplied full base commit resolves exactly in the candidate repository.' } else { 'Supplied base must resolve as the exact full commit id in the candidate repository.' })
$changed = Get-ChangedPaths
$outside = @($changed | Where-Object { -not (Test-IsAllowed $_ $scopes) })
Add-Check 'arm-specific-scope' $(if ($outside.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($outside.Count -eq 0) { "All changed paths fit $arm scopes: $($scopes -join ', ')" } else { "Out-of-scope paths: $($outside -join ', ')" })

# Published v1/v2 package fixture archives are immutable historical inputs on every card.
$fixturePaths = @('.markitect/packages/software-architecture-1.0.0.zip', '.markitect/packages/software-architecture-2.0.0.zip')
$fixtureViolations = [System.Collections.Generic.List[string]]::new()
foreach ($fixture in $fixturePaths) {
    $file = Join-Path $repoPath $fixture
    if (-not (Test-Path -LiteralPath $file)) { continue }
    Push-Location $repoPath
    try {
        $expected = (& git rev-parse ($Base + ':' + $fixture) 2>$null | Out-String).Trim()
        $actual = (& git hash-object -- $fixture 2>$null | Out-String).Trim()
        if ($expected -and $actual -and $expected -ne $actual) { $fixtureViolations.Add($fixture) }
        elseif (-not $expected) { $fixtureViolations.Add("$fixture (baseline blob unavailable)") }
    }
    finally { Pop-Location }
}
Add-Check 'published-fixture-bytes-preserved' $(if ($fixtureViolations.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($fixtureViolations.Count -eq 0) { 'Published v1/v2 fixture archives match their task-start Git blob IDs.' } else { "Published fixture archive mismatch: $($fixtureViolations -join ', ')" })

# A broad allowed path is not permission to revise accepted architecture. Implementation-only and
# escalation cards keep canonical architecture inputs unchanged. Intent cards are checked below.
$modelPaths = @($changed | Where-Object { $_ -match '^(resources/|markitect\.yaml$|\.markitect/(areas/|packages/)|docs/markitect/|\.agents/|\.codex/)' })
if ($script:taskId -in @(1, 2, 4, 9, 10, 12)) {
    $blockedModelPaths = $modelPaths
    if ($script:taskId -in @(9, 10, 12)) { $blockedModelPaths += @($changed | Where-Object { $_ -eq 'docs/architecture.md' }) }
    Add-Check 'no-unrequested-architecture-change' $(if ($blockedModelPaths.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($blockedModelPaths.Count -eq 0) { 'No canonical model, package, policy, or generated architecture paths changed for this implementation-only or escalation card.' } else { "Architecture-owned paths changed without an intent change: $($blockedModelPaths -join ', ')" })
}
if ($script:taskId -in @(3, 5, 7)) {
    $feature = switch ($script:taskId) { 3 { 'cancel-order' } 5 { 'get-order-summary' } 7 { 'list-open-orders' } }
    $resourceChanges = @($changed | Where-Object { $_ -match '^resources/' })
    $expectedUseCase = "resources/usecase-$feature.yaml"
    $expectedHandler = "resources/handler-$feature.yaml"
    $hasUseCase = (Test-Path -LiteralPath (Join-Path $repoPath $expectedUseCase))
    $hasHandler = (Test-Path -LiteralPath (Join-Path $repoPath $expectedHandler))
    $unexpectedResource = @($resourceChanges | Where-Object { $_ -notin @($expectedUseCase, $expectedHandler) })
    $forbiddenModelPaths = @($changed | Where-Object { $_ -match '^(markitect\.yaml$|\.markitect/(areas/|packages/))' })
    $validBModel = $hasUseCase -and $hasHandler -and $unexpectedResource.Count -eq 0 -and $forbiddenModelPaths.Count -eq 0
    Add-Check 'task-specific-usecase-owner' $(if ($arm -eq 'b' -and $validBModel) { 'passed' } elseif ($arm -eq 'a') { 'manual' } else { 'failed' }) $(if ($arm -eq 'a') { 'Arm A architecture documentation ownership requires independent assessor review.' } elseif ($validBModel) { "Added only the requested UseCase and Handler owners for $feature." } else { "Expected $expectedUseCase and $expectedHandler; unexpected resource/model changes: $(@($unexpectedResource + $forbiddenModelPaths) -join ', ')" })
}
if ($script:taskId -eq 6) {
    $useCasePath = 'resources/usecase-create-order.yaml'
    $validatorPath = 'resources/validator-create-order.yaml'
    $resourceChanges = @($changed | Where-Object { $_ -match '^resources/' })
    $unexpectedResource = @($resourceChanges | Where-Object { $_ -notin @($useCasePath, $validatorPath) })
    $forbiddenModelPaths = @($changed | Where-Object { $_ -match '^(markitect\.yaml$|\.markitect/(areas/|packages/))' })
    $hasValidator = Test-Path -LiteralPath (Join-Path $repoPath $validatorPath)
    $validBModel = $hasValidator -and $resourceChanges -contains $useCasePath -and $unexpectedResource.Count -eq 0 -and $forbiddenModelPaths.Count -eq 0
    Add-Check 'place-order-validator-owner' $(if ($arm -eq 'b' -and $validBModel) { 'passed' } elseif ($arm -eq 'a') { 'manual' } else { 'failed' }) $(if ($arm -eq 'a') { 'Arm A architecture documentation ownership requires independent assessor review.' } elseif ($validBModel) { 'Created the Validator owner and attached it only to create-order.' } else { "Expected $useCasePath and $validatorPath; unexpected resource/model changes: $(@($unexpectedResource + $forbiddenModelPaths) -join ', ')" })
}
if ($script:taskId -eq 8) {
    $handlerPath = 'resources/handler-create-order.yaml'
    $handlerText = Read-RepoFile $handlerPath
    $useCaseText = Read-RepoFile 'resources/usecase-create-order.yaml'
    $renamePresent = (Test-Path -LiteralPath (Join-Path $repoPath $handlerPath)) -and $handlerText -match 'name:\s*submit-order-handler' -and $useCaseText -match 'name:\s*submit-order-handler'
    $resourceChanges = @($changed | Where-Object { $_ -match '^resources/' })
    $unexpectedResource = @($resourceChanges | Where-Object { $_ -notin @($handlerPath,'resources/usecase-create-order.yaml') })
    $forbiddenModelPaths = @($changed | Where-Object { $_ -match '^(markitect\.yaml$|\.markitect/(areas/|packages/))' })
    $validBModel = $renamePresent -and $unexpectedResource.Count -eq 0 -and $forbiddenModelPaths.Count -eq 0
    Add-Check 'handler-identity-owner' $(if ($arm -eq 'b' -and $validBModel) { 'passed' } elseif ($arm -eq 'a') { 'manual' } else { 'failed' }) $(if ($arm -eq 'a') { 'Arm A architecture documentation ownership requires independent assessor review.' } elseif ($validBModel) { 'Handler identity and UseCase reference are updated in typed resources.' } else { "Expected identity update only in Handler and UseCase owners; unexpected resource/model changes: $(@($unexpectedResource + $forbiddenModelPaths) -join ', ')" })
}
if ($script:taskId -eq 11) {
    $resourceChanges = @($changed | Where-Object { $_ -match '^resources/' })
    $allowedMigrationResources = @('resources/usecase-create-order.yaml','resources/validator-create-order.yaml')
    $unexpectedMigrationResources = @($resourceChanges | Where-Object { $_ -notin $allowedMigrationResources })
    Add-Check 'no-unapproved-policy-weakening' $(if ($unexpectedMigrationResources.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($unexpectedMigrationResources.Count -eq 0) { 'Only the selected create-order Validator owner/relation may change when the frozen v2 finding requires it; all other canonical resource bytes retain their prior meaning.' } else { "Unapproved architecture resources changed: $($unexpectedMigrationResources -join ', ')" })
    $unexpectedMigrationPaths = @($changed | Where-Object {
        $p = $_
        -not ($p -match '^src/Commerce/Orders/Application/PlaceOrder/' -or
              $p -match '^checks/' -or
              $p -match '^markitect\.yaml$' -or
              $p -match '^docs/architecture\.md$' -or
              $p -match '^docs/markitect/' -or
              $p -match '^\.agents/' -or
              $p -match '^\.codex/')
    })
    $forbiddenModelPaths = @($changed | Where-Object { $_ -match '^\.markitect/(areas/|packages/)' })
    $unexpectedMigrationPaths += $forbiddenModelPaths
    Add-Check 'migration-card-specific-scope' $(if ($unexpectedMigrationPaths.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($unexpectedMigrationPaths.Count -eq 0) { 'Migration changes are limited to the selected command slice, its checks, exact project pin, and generated/documented views.' } else { "Unexpected paths for migration: $($unexpectedMigrationPaths -join ', ')" })
    if ($arm -eq 'b') {
        $config = Read-RepoFile 'markitect.yaml'
        $packageBlock = if ($config -match '(?ms)packages:\s*(?<block>.*?)(?=\r?\n\S|\z)') { $Matches['block'] } else { '' }
        $pinVersion = $packageBlock -match '(?m)^\s*version:\s*2\.0\.0\s*$'
        $pinArchive = $packageBlock -match '(?m)^\s*archive:\s*\.markitect/packages/software-architecture-2\.0\.0\.zip\s*$'
        $pinHashMatch = [regex]::Match($packageBlock, '(?m)^\s*sha256:\s*([0-9a-f]{64})\s*$')
        $archiveFile = Join-Path $repoPath '.markitect/packages/software-architecture-2.0.0.zip'
        $archiveHash = if (Test-Path -LiteralPath $archiveFile) { (Get-FileHash -LiteralPath $archiveFile -Algorithm SHA256).Hash.ToLowerInvariant() } else { '' }
        $hashMatches = $pinHashMatch.Success -and $pinHashMatch.Groups[1].Value -eq $archiveHash
        Add-Check 'exact-v2-pin-and-archive-digest' $(if ($pinVersion -and $pinArchive -and $hashMatches) { 'passed' } else { 'failed' }) 'Candidate pins software-architecture 2.0.0 at its exact archive path and matching SHA-256.'
        $beforeCase = Get-TextAtRevision $Base 'resources/usecase-create-order.yaml'
        $afterCase = Read-RepoFile 'resources/usecase-create-order.yaml'
        $beforeConfig = Get-TextAtRevision $Base 'markitect.yaml'
        $normalizedBeforeConfig = [regex]::Replace($beforeConfig, '(?m)^(\s*(?:version|archive|sha256):\s*).+$', '$1<pin>')
        $normalizedAfterConfig = [regex]::Replace($config, '(?m)^(\s*(?:version|archive|sha256):\s*).+$', '$1<pin>')
        $configStable = $beforeConfig -and $normalizedBeforeConfig -ceq $normalizedAfterConfig
        $normalizedBeforeCase = [regex]::Replace($beforeCase, '(?ms)^\s*validators:\s*(?:\[\]|(?:\r?\n\s+.*)+)\s*$', '')
        $normalizedAfterCase = [regex]::Replace($afterCase, '(?ms)^\s*validators:\s*(?:\[\]|(?:\r?\n\s+.*)+)\s*$', '')
        $caseStable = $beforeCase -and $afterCase -and $normalizedBeforeCase.Trim() -ceq $normalizedAfterCase.Trim()
        Add-Check 'selected-cohort-and-existing-meaning-preserved' $(if ($configStable -and $caseStable) { 'passed' } else { 'failed' }) 'Only package pin bytes and the selected create-order validators relation may change; project selection and other UseCase meaning remain fixed.'
        $beforeValidatorsMatch = [regex]::Match($beforeCase, '(?ms)^\s*validators:\s*(?<block>.*)$')
        $afterValidatorsMatch = [regex]::Match($afterCase, '(?ms)^\s*validators:\s*(?<block>.*)$')
        $beforeValidators = if ($beforeValidatorsMatch.Success) { $beforeValidatorsMatch.Groups['block'].Value.Trim() } else { '' }
        $afterValidators = if ($afterValidatorsMatch.Success) { $afterValidatorsMatch.Groups['block'].Value.Trim() } else { '' }
        $beforeHasValidator = -not [string]::IsNullOrWhiteSpace($beforeValidators) -and $beforeValidators -ne '[]'
        $afterHasValidator = -not [string]::IsNullOrWhiteSpace($afterValidators) -and $afterValidators -ne '[]'
        $conditionalValidatorState = $afterHasValidator -and (-not $beforeHasValidator -or $beforeValidators -ceq $afterValidators)
        Add-Check 'validator-added-only-for-real-v2-gap' $(if ($conditionalValidatorState) { 'passed' } else { 'failed' }) 'Keep an existing selected Validator unchanged; add one only when the task-start selected Command had none.'
        $beforeValidator = Get-TextAtRevision $Base 'resources/validator-create-order.yaml'
        $afterValidator = Read-RepoFile 'resources/validator-create-order.yaml'
        $validatorOwnerStable = if ($beforeValidator) { $beforeValidator -ceq $afterValidator } else { $true }
        Add-Check 'validator-owner-preserved' $(if ($validatorOwnerStable) { 'passed' } else { 'failed' }) 'An existing Validator owner keeps its task-start meaning.'
    }
}

# Any newly added exception or selector broadening is an independent failure, even if ordinary policy checks pass.
$exceptionPaths = @($changed | Where-Object { $_ -match '(^|/)(exceptions?)(/|\.|$)' })
Add-Check 'no-unapproved-exception' $(if ($exceptionPaths.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($exceptionPaths.Count -eq 0) { 'No exception artifacts were added or changed.' } else { "Exception paths require explicit review: $($exceptionPaths -join ', ')" })

# Check core/domain/module boundaries using source namespaces and imports, independently of actor-authored checks.
$ordersRoot = Join-Path $repoPath 'src/Commerce/Orders'
$sourceFiles = if (Test-Path -LiteralPath $ordersRoot) { @(Get-ChildItem -LiteralPath $ordersRoot -Recurse -File -Filter '*.cs') } else { @() }
$boundaryViolations = [System.Collections.Generic.List[string]]::new()
foreach ($file in $sourceFiles) {
    $text = Get-Content -LiteralPath $file.FullName -Raw
    if ($file.FullName -match '\\Orders\\Domain\\' -and $text -match '(?m)^\s*(using\s+Commerce\.(Orders\.Application|Inventory|Billing|Core\.Time)|namespace\s+Commerce\.(Orders\.Application|Inventory|Billing|Core\.Time))') { $boundaryViolations.Add($file.FullName) }
    if ($file.FullName -match '\\Orders\\Domain\\' -and $text -match '\bCommerce\.Core\.Time\.') { $boundaryViolations.Add($file.FullName) }
    if ($file.FullName -match '\\Orders\\' -and $text -match '(?m)^\s*(using|namespace)\s+Commerce\.(Inventory\.Application|Inventory\.Domain|Billing\.Application|Billing\.Domain)(\.|\s|;)') { $boundaryViolations.Add($file.FullName) }
    if ($file.FullName -match '\\Orders\\Domain\\' -and $text -match '\b(DateTime\.Now|DateTime\.UtcNow|DateTimeOffset\.Now|DateTimeOffset\.UtcNow)\b') { $boundaryViolations.Add($file.FullName) }
}
Add-Check 'domain-and-module-boundaries' $(if ($boundaryViolations.Count -eq 0) { 'passed' } else { 'failed' }) $(if ($boundaryViolations.Count -eq 0) { 'Orders Domain has no application/module dependency or ambient system clock; Orders uses only published module contracts.' } else { "Boundary violations: $($boundaryViolations | Sort-Object -Unique -join ', ')" })

# Run the independent, evaluator-owned public behavior vector for the first task.
if ($script:taskId -ge 1) {
    $vector = Test-TaskOneVector
    Add-Check 'task01-order-total-vector' $vector.status $vector.evidence
    $checkSources = if (Test-Path -LiteralPath (Join-Path $repoPath 'checks')) { @(Get-ChildItem -LiteralPath (Join-Path $repoPath 'checks') -Recurse -File -Filter '*.cs' | ForEach-Object { Get-Content -LiteralPath $_.FullName -Raw }) -join "`n" } else { '' }
    $hasRegression = $checkSources -match '(?s)25(?:\.0+)?m' -and $checkSources -match '(?i)(Quantity|OrderLine)' -and $checkSources -match 'Total'
    Add-Check 'task01-regression-assertion' 'manual' $(if ($hasRegression) { 'A source-text signal for the requested 25.00 regression assertion exists; a blinded assessor must verify it is an active meaningful check.' } else { 'No obvious 25.00 quantity regression assertion was found; a blinded assessor must inspect checks without treating actor tests as behavioral proof.' })
}

Add-Check 'baseline-assertion-preservation' 'manual' 'Compare checks/Program.cs from the exact task-start revision with the candidate; preserve all six original assertion meanings. Allow the task01 total-bug correction and public API/test-framework refactors, with independent behavior vectors as separate evidence.'

$script:commands.Add('dotnet run --project checks/Commerce.Checks.csproj --configuration Release')
Push-Location $repoPath
try {
    $validatorText = (& dotnet run --project checks/Commerce.Checks.csproj --configuration Release 2>&1 | Out-String)
    $validatorExit = $LASTEXITCODE
}
finally { Pop-Location }
Add-Check 'acceptance-validator' $(if ($validatorExit -eq 0) { 'passed' } else { 'failed' }) ("dotnet run --project checks/Commerce.Checks.csproj --configuration Release; exit=$validatorExit; " + (($validatorText -replace '[\r\n]+',' | ').Trim()))

# Later requested outcomes need task-specific runnable public API contracts. Keep gaps explicit; never let
# a green actor-authored validator or prose artifact count as independent behavioral evidence.
if ($script:taskId -ge 2 -and $script:taskId -le 8) {
    Add-Check 'task-specific-public-behavior-vectors' 'manual' "Independent task $Task behavior probes are unavailable until a stable public entrypoint can be invoked without assuming an unrequested hidden API; see evaluator rubric vectors."
}
if ($script:taskId -ge 2) {
    Add-Check 'cumulative-prior-behavior-vectors' 'manual' "Only task 01 has a frozen runnable public API probe in this oracle; prior task vectors remain unavailable pending the frozen assessor procedure."
}
if ($script:taskId -eq 11) {
    Add-Check 'migration-findings-and-v2-result' 'manual' 'Review the actual v1 and v2 fixed-snapshot findings against the exact unchanged selected cohort. A task06-compliant state may have zero findings before and after; do not invent a missing validator or claim implementation work that was unnecessary.'
}
if ($script:taskId -in @(9,10,12)) {
    $codeChanges = @($changed | Where-Object { $_ -match '^src/Commerce/.*\.cs$' })
    $codeEvidence = if ($codeChanges.Count -eq 0) { 'No implementation code changed while the owner decision is pending.' } else { 'Implementation changed before decision: ' + ($codeChanges -join ', ') }
    Add-Check 'no-speculative-code-change' $(if ($codeChanges.Count -eq 0) { 'passed' } else { 'failed' }) $codeEvidence
    Add-Check 'owner-decision-pending' 'manual' 'Output remains owner-decision-required; response quality and the required human decision cannot be accepted automatically.'
    Add-Check 'escalation-response-fields' 'manual' 'A separate blinded assessor must compare the final response to this cards frozen response_fields; no answer is auto-accepted.'
}

$status = if ($script:failures.Count -gt 0) { 'failed' } elseif ($script:taskId -in @(9,10,12)) { 'owner-decision-required' } elseif ($script:manual.Count -gt 0) { 'manual-review-required' } else { 'passed' }
$newline = [Environment]::NewLine
$quote = [string][char]34
$checkYaml = ($script:checks | ForEach-Object {
    $safe = $_.evidence.Replace([string][char]34, [string][char]39)
    '  - name: ' + $_.name + $newline + '    status: ' + $_.status + $newline + '    evidence: ' + $quote + $safe + $quote
}) -join $newline
$failureYaml = if ($script:failures.Count) { ($script:failures | ForEach-Object { '  - ' + $_ }) -join $newline } else { '  []' }
$manualYaml = if ($script:manual.Count) { ($script:manual | ForEach-Object { '  - ' + $_ }) -join $newline } else { '  []' }
$changedYaml = if ($changed.Count) { ($changed | ForEach-Object { '  - ' + $quote + $_ + $quote }) -join $newline } else { '  []' }
$commandYaml = if ($script:commands.Count) { ($script:commands | ForEach-Object { '  - ' + $quote + $_ + $quote }) -join $newline } else { '  []' }
$ownerDecision = if ($script:taskId -in @(9,10,12)) { 'required; no automatic acceptance' } else { 'not required' }
$yamlLines = @(
    ('task: ' + $quote + $Task + $quote)
    ('arm: ' + $arm)
    ('status: ' + $status)
    ('base: ' + $quote + $Base + $quote)
    'changed_paths:'
    $changedYaml
    'checks:'
    $checkYaml
    'failures:'
    $failureYaml
    'manual_review:'
    $manualYaml
    'commands:'
    $commandYaml
    ('owner_decision: ' + $ownerDecision)
)
$yaml = $yamlLines -join $newline
Set-Content -LiteralPath $Output -Value $yaml -Encoding utf8
$yaml



