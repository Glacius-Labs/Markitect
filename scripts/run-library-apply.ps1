<##
.SYNOPSIS
Build and run one fresh Library r2 Apply through the product command.
.EXAMPLE
./scripts/run-library-apply.ps1 -CredentialPath 'C:\private\codex-auth.json'
##>
param(
    [Parameter(Mandatory = $true)][string]$CredentialPath,
    [string]$OutputRoot = (Join-Path ([Environment]::GetFolderPath('MyDocuments')) 'Codex\Markitect-Library-Runs'),
    [int]$MaxMessages = 40,
    [int]$Minutes = 45,
    [int]$MessageTimeoutSeconds = 120
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if ($MaxMessages -lt 1 -or $Minutes -lt 1 -or $MessageTimeoutSeconds -lt 1) { throw 'All invocation limits must be positive.' }
$sourceRepo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docker = (Get-Command docker -ErrorAction Stop).Source
$git = (Get-Command git -ErrorAction Stop).Source
$go = (Get-Command go -ErrorAction Stop).Source
$CredentialPath = [IO.Path]::GetFullPath($CredentialPath)
if (-not (Test-Path -LiteralPath $CredentialPath -PathType Leaf)) { throw 'The dedicated Codex runtime login file is missing.' }

# Every invocation creates a new directory before a build or provider effect.
$OutputRoot = [IO.Path]::GetFullPath($OutputRoot)
New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
$runName = 'library-' + [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ') + '-' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
$runDir = Join-Path $OutputRoot $runName
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
    if ((& $git -C $source status --porcelain).Length -ne 0) { throw 'Fresh Main source is not clean.' }
    $integratedScript = Join-Path $source 'scripts\run-library-apply.ps1'
    if (-not (Test-Path -LiteralPath $integratedScript -PathType Leaf) -or
        (Get-FileHash -LiteralPath $integratedScript -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $PSCommandPath -Algorithm SHA256).Hash) {
        throw 'The launched script is not the exact version in integrated Main.'
    }
    Push-Location $source
    try {
        Invoke-NativeChecked 'product build' { & $go build -trimpath -o $binary ./cmd/markitect }
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
    Copy-Item -LiteralPath (Join-Path $source 'experiments\test-projects\library-r2\project\.markitect') -Destination $fixture -Recurse
    Invoke-NativeChecked 'fixture init' { & $git -C $fixture init --initial-branch=dev/library-r2 }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local user.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local user.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local author.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local author.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local committer.name 'Markitect Library Fixture' }
    Invoke-NativeChecked 'fixture identity' { & $git -C $fixture config --local committer.email 'fixture@markitect.invalid' }
    Invoke-NativeChecked 'fixture add' { & $git -C $fixture add --all }
    Invoke-NativeChecked 'fixture commit' { & $git -C $fixture commit -m 'Library r2 requirement baseline' }
    $fixtureHead = (& $git -C $fixture rev-parse HEAD).Trim()
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
      credentialRef: library-luna-runtime-login
      maxWireBytes: 1048576
      session:
        version: 1
        model: gpt-6-luna
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
        formatVersion = 2; repositoryIdentity = 'library-r2'; productBuild = $productBuild
        ownerRequest = 'Implement the complete fixed Library r2 M1 model and its declared Go JSONL interface. Preserve stated behavior and state boundaries. Ask your parent only when genuine ambiguity blocks conforming work.'
        stateDirectory = $state; engineExecutable = $docker; deadline = $deadline
        maxMessages = $MaxMessages; messageTimeoutSeconds = $MessageTimeoutSeconds
        workspaceHelper = [ordered]@{ name = "markitect-$runName"; image = $baseImage; labels = @{ 'markitect.store' = $runName }; user = '1000:1000'; cpus = 1; memoryBytes = 1073741824; pids = 64; tmpBytes = 268435456; network = '' }
        harvest = @{ maxEntries = 1000; maxBytes = 16777216 }
        credentials = @{ 'library-luna-runtime-login' = @{ provider = 'codex-app-server'; mode = 'dedicated-runtime-login'; path = $CredentialPath } }
    }
    $configPath = Join-Path $runDir 'run.json'
    [IO.File]::WriteAllText($configPath, (($config | ConvertTo-Json -Depth 12) + "`n"), [Text.UTF8Encoding]::new($false))
    $manifest = [ordered]@{
        mainHead = $mainHead; fixtureHead = $fixtureHead; productBuild = $productBuild
        productExecutableSha256 = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
        runtimeProfileSha256 = (Get-FileHash -LiteralPath $profilePath -Algorithm SHA256).Hash.ToLowerInvariant()
        runConfigSha256 = (Get-FileHash -LiteralPath $configPath -Algorithm SHA256).Hash.ToLowerInvariant()
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
