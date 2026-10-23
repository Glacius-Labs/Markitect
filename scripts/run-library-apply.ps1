<##
.SYNOPSIS
Build and run one fresh canonical Library M1 Apply through the product command.
.EXAMPLE
./scripts/run-library-apply.ps1 -CredentialPath 'C:\private\codex-auth.json'
##>
param(
    [Parameter(Mandatory = $true)][string]$CredentialPath,
    [string]$RunName = '',
    [string]$OutputRoot = (Join-Path ([Environment]::GetFolderPath('MyDocuments')) 'Codex\Markitect-Library-Runs'),
    [int]$MaxMessages = 128,
    [int]$Minutes = 45,
    [int]$MessageTimeoutSeconds = 600
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if ($MaxMessages -lt 1 -or $MaxMessages -gt 128 -or $Minutes -lt 1 -or
    $MessageTimeoutSeconds -lt 1 -or $MessageTimeoutSeconds -gt 3600) {
    throw 'Invocation limits require 1..128 messages, positive minutes, and 1..3600 seconds per message.'
}
$sourceRepo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docker = (Get-Command docker -ErrorAction Stop).Source
$git = (Get-Command git -ErrorAction Stop).Source
$go = (Get-Command go -ErrorAction Stop).Source
$CredentialPath = [IO.Path]::GetFullPath($CredentialPath)
if (-not (Test-Path -LiteralPath $CredentialPath -PathType Leaf)) { throw 'The dedicated Codex runtime login file is missing.' }

# Every invocation creates a new directory before a build or provider effect.
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
$runName = $RunName
if ($runName -eq '') { $runName = 'library-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ') + '-' + [Guid]::NewGuid().ToString('N').Substring(0, 8) }
if ($runName.Length -gt 50 -or $runName -cnotmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]*$') { throw 'RunName must be a safe unique name of at most 50 characters.' }
$runDir = Join-Path $OutputRoot $runName
if (Test-Path -LiteralPath $runDir) { throw "Run directory already exists: $runDir" }
New-Item -ItemType Directory -Path $runDir -ErrorAction Stop | Out-Null
$source = Join-Path $runDir 'source'
$fixture = Join-Path $runDir 'fixture'
$state = Join-Path $runDir 'state'
$binary = Join-Path $runDir 'markitect.exe'

function Invoke-NativeChecked([string]$Label, [scriptblock]$Action) {
    & $Action
    if ($LASTEXITCODE -ne 0) { throw "$Label failed (exit $LASTEXITCODE); evidence remains at $runDir" }
}

try {
    Invoke-NativeChecked 'fetch Main' { & $git -C $sourceRepo fetch origin main }
    $mainHead = (& $git -C $sourceRepo rev-parse 'refs/remotes/origin/main').Trim()
    if ($LASTEXITCODE -ne 0 -or $mainHead -notmatch '^[0-9a-f]{40}$') { throw 'Cannot resolve an exact Main commit.' }
    Invoke-NativeChecked 'clean Main worktree' { & $git -C $sourceRepo worktree add --detach $source $mainHead }
    $sourceStatus = @(& $git -C $source status --porcelain)
    if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect the fresh Main source.' }
    if ($sourceStatus.Count -ne 0) { throw 'Fresh Main source is not clean.' }
    $integratedScript = Join-Path $source 'scripts\run-library-apply.ps1'
    if (-not (Test-Path -LiteralPath $integratedScript -PathType Leaf) -or
        (Get-FileHash -LiteralPath $integratedScript -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $PSCommandPath -Algorithm SHA256).Hash) {
        throw 'The launched script is not the exact version in integrated Main.'
    }
    Push-Location $source
    try {
        # The source hash only binds the build when ambient Go settings cannot
        # replace files or modules (for example GOFLAGS=-overlay=...).
        $goInputNames = @((Get-ChildItem Env: | Where-Object {
            $_.Name -match '^(GO[A-Z0-9_]*|CGO_[A-Z0-9_]*)$'
        } | ForEach-Object { $_.Name }))
        $goInputs = @{}
        foreach ($name in $goInputNames) {
            $goInputs[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
            Remove-Item -LiteralPath "Env:$name"
        }
        try {
            $env:GOFLAGS = ''
            $env:GOWORK = 'off'
            $env:GOENV = 'off'
            $env:CGO_ENABLED = '0'
            Invoke-NativeChecked 'product build' { & $go build -trimpath -buildvcs=true -o $binary ./cmd/markitect }
        } finally {
            foreach ($name in @('GOFLAGS', 'GOWORK', 'GOENV', 'CGO_ENABLED')) { Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue }
            foreach ($name in $goInputNames) { [Environment]::SetEnvironmentVariable($name, $goInputs[$name], 'Process') }
        }
        $version = ((& $binary version) | ConvertFrom-Json).version
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($version)) { throw 'Cannot read product build version.' }
        $productBuild = "$version+$mainHead"
        $short = $mainHead.Substring(0, 12)
        $baseTag = "markitect-library-base:$short"
        $goTag = "markitect-library-go:$short"
        Invoke-NativeChecked 'runtime base build' { & $docker build -f packaging/runtime/Dockerfile --build-arg "MARKITECT_VERSION=$mainHead" -t $baseTag . }
        Invoke-NativeChecked 'runtime Go build' { & $docker build -f packaging/runtime/examples/go/Dockerfile --build-arg "BASE=$baseTag" -t $goTag . }
    } finally { Pop-Location }

    $baseImage = (& $docker image inspect --format '{{.Id}}' $baseTag).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect the base image.' }
    $goImage = (& $docker image inspect --format '{{.Id}}' $goTag).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect the Go image.' }
    $metadata = & $docker run --rm --network none --entrypoint /bin/cat $baseImage /opt/markitect/runtime.env
    if ($LASTEXITCODE -ne 0) { throw 'Cannot read immutable runtime metadata.' }
    $records = @{}
    foreach ($line in $metadata) {
        $parts = $line.Split('=', 2)
        if ($parts.Count -ne 2 -or $records.ContainsKey($parts[0])) { throw 'Invalid runtime metadata.' }
        $records[$parts[0]] = $parts[1]
    }
    if ($records.agentd -ne $mainHead -or $records.codexSchema -notmatch '^sha256:[0-9a-f]{64}$') { throw 'Runtime build metadata does not match Main.' }

    New-Item -ItemType Directory -Path $fixture -ErrorAction Stop | Out-Null
    $control = Join-Path $fixture '.markitect'
    New-Item -ItemType Directory -Path $control -ErrorAction Stop | Out-Null
    $canonical = Join-Path $source 'docs\design\canonical-model\library-example\m1'
    foreach ($file in @('model.yaml', 'model-lock.yaml')) {
        Copy-Item -LiteralPath (Join-Path $canonical $file) -Destination (Join-Path $control $file)
    }
    Copy-Item -LiteralPath (Join-Path $canonical 'model') -Destination $control -Recurse
    $dependencies = Join-Path $control 'dependencies'
    New-Item -ItemType Directory -Path $dependencies -ErrorAction Stop | Out-Null
    Copy-Item -LiteralPath (Join-Path $source 'docs\design\canonical-model\library-example\vocabulary') -Destination (Join-Path $dependencies 'software') -Recurse
    Copy-Item -LiteralPath (Join-Path $source 'docs\design\canonical-model\minimal-language\exchange-example\dependencies\core') -Destination (Join-Path $dependencies 'core') -Recurse
    # Only execution controls are authored here. Canonical M1 bytes stay exact;
    # hidden oracle expectations remain outside every Manager input.
    $execution = @"
apiVersion: markitect.execution/v2
ownership:
  - {path: ./, owner: .}
  - {path: realization/catalog/, owner: catalog}
  - {path: realization/lending/, owner: lending}
  - {path: realization/lending/reminders/, owner: lending/reminders}
  - {path: realization/application/, owner: library}
  - {path: realization/members/, owner: members}
  - {path: realization/cmd/library/, owner: library}
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
"@
    [IO.File]::WriteAllText((Join-Path $control 'execution.yaml'), $execution + "`n", [Text.UTF8Encoding]::new($false))
    New-Item -ItemType Directory -Path (Join-Path $control 'registry') -ErrorAction Stop | Out-Null
    [IO.File]::WriteAllText((Join-Path $control 'registry\registry.yaml'), "apiVersion: markitect.registry/v1`nartifacts: []`n", [Text.UTF8Encoding]::new($false))
    Invoke-NativeChecked 'fixture init' { & $git -C $fixture init --initial-branch=dev/library-m1 }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local user.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local user.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local author.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local author.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local committer.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local committer.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture add' { & $git -C $fixture add --all }
    Invoke-NativeChecked 'fixture commit' { & $git -C $fixture commit -m 'Canonical Library M1 requirement baseline' }
    $fixtureHead = (& $git -C $fixture rev-parse HEAD).Trim()
    $fixtureCheck = Join-Path $runDir 'fixture-check.json'
    & $binary check --repo $fixture --model .markitect/model.yaml --dependency .markitect/dependencies/software/model.yaml --dependency .markitect/dependencies/core/model.yaml --revision HEAD 1> $fixtureCheck
    if ($LASTEXITCODE -ne 0) { throw 'Fresh canonical Library M1 fixture failed structural preparation.' }
    New-Item -ItemType Directory -Path $state -ErrorAction Stop | Out-Null

    $engineLiteral = $docker.Replace("'", "''")
    $profile = @"
apiVersion: markitect.runtime/v1
agentProfiles:
  library-agent:
    codexAppServer:
      profileVersion: 3
      engineExecutable: '$engineLiteral'
      imageID: $goImage
      agentdVersion: $mainHead
      recordedCodexSchemaDigest: $($records.codexSchema)
      instructions: []
      credentialRef: library-account-login
      maxWireBytes: 1048576
      session:
        version: 1
        model: gpt-6-sol
        effort: high
        baseInstructions: ''
        developerInstructions: ''
        store: $runName
        cpus: 1
        memoryBytes: 2147483648
        pids: 192
        tmpBytes: 268435456
        network: ''
checkProfiles:
  library-tests:
    container:
      profileVersion: 1
      engineExecutable: '$engineLiteral'
      imageID: $goImage
      executable: /usr/local/go/bin/go
      argumentPrefix: []
      env:
        - GOTOOLCHAIN=local
        - GOCACHE=/home/agent/.cache/go-build
      limits:
        milliCPUs: 1000
        memoryBytes: 2147483648
        processes: 64
      maxOutputBytes: 1048576
"@
    $profilePath = Join-Path $runDir 'runtime-profile.yaml'
    [IO.File]::WriteAllText($profilePath, $profile + "`n", [Text.UTF8Encoding]::new($false))
    $deadline = [DateTime]::UtcNow.AddMinutes($Minutes).ToString('yyyy-MM-ddTHH:mm:ssZ')
    $config = [ordered]@{
        formatVersion = 3; repositoryIdentity = 'small-library'; productBuild = $productBuild
        stateDirectory = $state; engineExecutable = $docker; deadline = $deadline
        maxManagers = 32; maxMessages = $MaxMessages; messageTimeoutSeconds = $MessageTimeoutSeconds
        cleanupTimeoutSeconds = 60; persistenceTimeoutSeconds = 30
        credentials = @{ 'library-account-login' = @{ provider = 'codex-app-server'; mode = 'dedicated-runtime-login'; path = $CredentialPath } }
    }
    $configPath = Join-Path $runDir 'run.json'
    [IO.File]::WriteAllText($configPath, (($config | ConvertTo-Json -Depth 12) + "`n"), [Text.UTF8Encoding]::new($false))
    $manifest = [ordered]@{
        mainHead = $mainHead; fixtureHead = $fixtureHead; productBuild = $productBuild
        productExecutableSha256 = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
        runtimeProfileSha256 = (Get-FileHash -LiteralPath $profilePath -Algorithm SHA256).Hash.ToLowerInvariant()
        runConfigSha256 = (Get-FileHash -LiteralPath $configPath -Algorithm SHA256).Hash.ToLowerInvariant()
        fixtureCheckSha256 = (Get-FileHash -LiteralPath $fixtureCheck -Algorithm SHA256).Hash.ToLowerInvariant()
        baseImage = $baseImage; goImage = $goImage; runDirectory = $runDir
        deadline = $deadline; maxMessages = $MaxMessages; messageTimeoutSeconds = $MessageTimeoutSeconds
    }
    [IO.File]::WriteAllText((Join-Path $runDir 'start.json'), (($manifest | ConvertTo-Json -Depth 5) + "`n"), [Text.UTF8Encoding]::new($false))

    $stdout = Join-Path $runDir 'apply.stdout.json'
    $stderr = Join-Path $runDir 'apply.stderr.log'
    & $binary apply --repo $fixture --model .markitect/model.yaml --dependency .markitect/dependencies/software/model.yaml --dependency .markitect/dependencies/core/model.yaml --run-config $configPath --runtime-profile $profilePath 1> $stdout 2> $stderr
    $applyExit = $LASTEXITCODE
    $end = [ordered]@{
        endedAt = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
        applyExit = $applyExit
        stdoutSha256 = (Get-FileHash -LiteralPath $stdout -Algorithm SHA256).Hash.ToLowerInvariant()
        stderrSha256 = (Get-FileHash -LiteralPath $stderr -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    [IO.File]::WriteAllText((Join-Path $runDir 'end.json'), (($end | ConvertTo-Json -Depth 3) + "`n"), [Text.UTF8Encoding]::new($false))
    Write-Output "Run directory: $runDir"
    Write-Output "Apply exit: $applyExit"
    exit $applyExit
} catch {
    [IO.File]::WriteAllText((Join-Path $runDir 'launcher-error.txt'), $_.Exception.Message + "`n", [Text.UTF8Encoding]::new($false))
    throw
}
