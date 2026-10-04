param(
    [Parameter(Mandatory = $true)][string]$Repo,
    [Parameter(Mandatory = $true)][ValidatePattern('^(0[1-9]|1[0-2]|P0[1-2])$')][string]$Task,
    [string]$Base = $env:GAUNTLET_TASK_BASE,
    [string]$PriorTasksThrough = '',
    [Parameter(Mandatory = $true)][string]$Output
)

$ErrorActionPreference = 'Stop'
$repoPath = (Resolve-Path -LiteralPath $Repo).Path
$script:checks = [System.Collections.Generic.List[object]]::new()
$script:failures = [System.Collections.Generic.List[string]]::new()
$script:changed = @()
$script:effectiveTasks = @()
$arm = if (Test-Path -LiteralPath (Join-Path $repoPath 'markitect.yaml')) { 'b' } else { 'a' }
$taskIndex = if ($Task -match '^P01$') { 7 } elseif ($Task -match '^P02$') { 8 } else { [int]$Task }
$parallelTask = $Task -match '^P0[1-2]$'
$decisionRequired = $false

function Test-EffectiveTask([int]$Index) {
    return $script:effectiveTasks -contains $Index
}

function Quote-Yaml([string]$Value) {
    return ConvertTo-Json -InputObject $Value -Compress
}

function Add-Check([string]$Name, [bool]$Passed, [string]$Evidence) {
    $safe = ($Evidence -replace '[\r\n]+', ' ').Trim()
    $script:checks.Add([pscustomobject]@{ name = $Name; passed = $Passed; evidence = $safe })
    if (-not $Passed) { $script:failures.Add($Name) }
}

function Get-TaskCard([string]$TaskId) {
    $taskFile = if ($TaskId -match '^P0[1-2]$') {
        Join-Path $PSScriptRoot '..\parallel-task-set.yaml'
    } else {
        Join-Path $PSScriptRoot '..\tasks\task-set.yaml'
    }
    $yaml = Get-Content -LiteralPath $taskFile -Raw
    $match = [regex]::Match($yaml, '(?ms)^\s*- id: ["'']?' + [regex]::Escape($TaskId) + '["'']?\s*\r?\n(?<body>.*?)(?=^\s*- id:|\z)')
    if (-not $match.Success) { throw "Task $TaskId not found in $taskFile" }
    return $match.Groups['body'].Value
}

function Parse-YamlList([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value)) { return @() }
    return @($Value -split ',\s*' | ForEach-Object { $_.Trim().Trim('"').Trim("'") } | Where-Object { $_ })
}

function Get-TaskScopes([string]$Body, [string]$SelectedArm) {
    $scoped = [regex]::Match($Body, '(?m)^\s+' + [regex]::Escape($SelectedArm) + ': \[(?<items>[^\]]*)\]')
    if ($scoped.Success) { return Parse-YamlList $scoped.Groups['items'].Value }
    $shared = [regex]::Match($Body, '(?m)^\s+allowed_paths: \[(?<items>[^\]]*)\]')
    if ($shared.Success) { return Parse-YamlList $shared.Groups['items'].Value }
    return @()
}

function Invoke-Repo([string]$WorkingDirectory, [string]$Executable, [string[]]$Arguments) {
    Push-Location $WorkingDirectory
    $priorPreference = $ErrorActionPreference
    $priorGoCache = $env:GOCACHE
    $priorPythonBytecode = $env:PYTHONDONTWRITEBYTECODE
    try {
        $ErrorActionPreference = 'Continue'
        $env:GOCACHE = Join-Path ([IO.Path]::GetTempPath()) 'engineering-ops-gauntlet-gocache'
        New-Item -ItemType Directory -Path $env:GOCACHE -Force | Out-Null
        $env:PYTHONDONTWRITEBYTECODE = '1'
        $lines = @(& $Executable @Arguments 2>&1)
        $exit = $LASTEXITCODE
        return [pscustomobject]@{ exit = $exit; output = ($lines -join "`n") }
    } finally {
        $ErrorActionPreference = $priorPreference
        $env:GOCACHE = $priorGoCache
        $env:PYTHONDONTWRITEBYTECODE = $priorPythonBytecode
        Pop-Location
    }
}

function Get-ChangedPaths([string]$Revision) {
    $commit = Invoke-Repo $repoPath 'git' @('cat-file', '-e', ($Revision + '^{commit}') )
    if ($commit.exit -ne 0) { throw "Base does not resolve to a commit: $Revision" }
    $tracked = Invoke-Repo $repoPath 'git' @('diff', '--name-only', '--diff-filter=ACDMRTUXB', $Revision, '--')
    if ($tracked.exit -ne 0) { throw "git diff failed: $($tracked.output)" }
    $untracked = Invoke-Repo $repoPath 'git' @('ls-files', '--others', '--ignored', '--exclude-standard')
    if ($untracked.exit -ne 0) { throw "git ls-files failed: $($untracked.output)" }
    $all = @($tracked.output -split "`n") + @($untracked.output -split "`n")
    return @($all | ForEach-Object { ($_ -replace '\\', '/').Trim() } | Where-Object {
        $_ -and $_ -notmatch '^(\.git|\.cache)(/|$)'
    } | Sort-Object -Unique)
}

function Test-InScope([string]$Path, [string[]]$Scopes) {
    foreach ($scope in $Scopes) {
        $prefix = ($scope -replace '\\', '/').TrimEnd('/')
        if ($Path -ieq $prefix -or $Path.StartsWith($prefix + '/', [StringComparison]::OrdinalIgnoreCase)) { return $true }
    }
    return $false
}

function Copy-SourceTree([string]$Source, [string]$Destination) {
    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    foreach ($entry in Get-ChildItem -LiteralPath $Source -Force) {
        if ($entry.Name -in @('.git', '.cache') -or ($entry.Attributes -band [IO.FileAttributes]::ReparsePoint)) { continue }
        $target = Join-Path $Destination $entry.Name
        if ($entry.PSIsContainer) { Copy-SourceTree $entry.FullName $target }
        else { Copy-Item -LiteralPath $entry.FullName -Destination $target -Force }
    }
}

function Invoke-HiddenVector([string]$TempRepo, [string]$VectorName, [string]$TestName, [string]$RelativeSource, [string]$GoDirectory) {
    $vector = Join-Path $PSScriptRoot ('vectors/' + $VectorName)
    $target = Join-Path $TempRepo ($RelativeSource -replace '/', [IO.Path]::DirectorySeparatorChar)
    if (-not (Test-Path -LiteralPath $target -PathType Container)) { throw "Vector target missing in evaluation copy: $RelativeSource" }
    $testName = $VectorName -replace '\.go\.txt$', '_hidden_test.go'
    $testFile = Join-Path $target $testName
    if (Test-Path -LiteralPath $testFile) { throw "Refusing to overwrite candidate file in temporary copy: $RelativeSource/$testName" }
    Copy-Item -LiteralPath $vector -Destination $testFile
    $result = Invoke-Repo $TempRepo 'go' @('-C', $GoDirectory, 'test', '.', '-run', ('^' + $TestName + '$'), '-count=1')
    Add-Check $VectorName ($result.exit -eq 0) $(if ($result.exit -eq 0) { 'Cumulative fixed hidden behavior vectors passed in an external temporary copy.' } else { $result.output })
}

function Read-RepoText([string]$RelativePath) {
    $path = Join-Path $script:repoPath ($RelativePath -replace '/', [IO.Path]::DirectorySeparatorChar)
    if (Test-Path -LiteralPath $path -PathType Leaf) { return Get-Content -LiteralPath $path -Raw }
    return ''
}

function Test-TextInSource([string[]]$Roots, [string]$Pattern) {
    foreach ($root in $Roots) {
        $path = Join-Path $script:repoPath ($root -replace '/', [IO.Path]::DirectorySeparatorChar)
        if (-not (Test-Path -LiteralPath $path)) { continue }
        $items = if ((Get-Item -LiteralPath $path).PSIsContainer) {
            Get-ChildItem -LiteralPath $path -File -Recurse -Force | Where-Object { $_.FullName -notmatch '[\\/](\.git|\.cache)[\\/]' }
        } else { @(Get-Item -LiteralPath $path) }
        foreach ($item in $items) {
            if ($item.Extension -in @('.md', '.yaml', '.yml', '.json', '.txt')) {
                if ((Get-Content -LiteralPath $item.FullName -Raw) -match $Pattern) { return $true }
            }
        }
    }
    return $false
}

function Test-GoAssertion([string]$Path, [string]$FunctionName, [string[]]$Patterns) {
    $source = Read-RepoText $Path
    $pattern = '(?ms)^func\s+' + [regex]::Escape($FunctionName) + '\s*\(t \*testing\.T\)\s*\{(?<body>.*?)(?=^func\s|\z)'
    $match = [regex]::Match($source, $pattern)
    if (-not $match.Success) { return $false }
    foreach ($required in $Patterns) { if ($match.Groups['body'].Value -notmatch $required) { return $false } }
    return $true
}

function Test-PythonAssertion([string]$Path, [string]$FunctionName, [string[]]$Patterns) {
    $source = Read-RepoText $Path
    $pattern = '(?ms)^\s+def\s+' + [regex]::Escape($FunctionName) + '\s*\(self\):(?<body>.*?)(?=^\s+def\s|\Z)'
    $match = [regex]::Match($source, $pattern)
    if (-not $match.Success) { return $false }
    foreach ($required in $Patterns) { if ($match.Groups['body'].Value -notmatch $required) { return $false } }
    return $true
}

function Test-DeclaredRunbook([string]$RelativePath) {
    if ($arm -eq 'a') {
        $config = Read-RepoText 'docs/engineering/managed-files.yaml'
        return $config -match '(?m)^\s+' + [regex]::Escape($RelativePath) + ':\s+\S'
    }
    $skill = Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml'
    return $skill -match '(?m)^\s+- ' + [regex]::Escape($RelativePath) + '\s*$'
}

function Add-OriginalAssertionChecks([int]$Index) {
    $observeZero = Test-GoAssertion 'tools/process-sentinel/main_test.go' 'TestObserveReturnsObservedChildExitCode' @('observe\(ctx, os\.Args\[0\]', 'err != nil', 'result\.ExitCode != 0')
    $observeFailure = Test-GoAssertion 'tools/process-sentinel/main_test.go' 'TestObserveReportsChildFailure' @('PROCESS_SENTINEL_CHILD', 'fail', 'err != nil', 'result\.ExitCode != 7')
    $observeDeadline = Test-GoAssertion 'tools/process-sentinel/main_test.go' 'TestObserveReportsCancellationDeadline' @('PROCESS_SENTINEL_CHILD', 'wait', '10\s*\*\s*time\.Millisecond', 'err != nil', 'result\.ExitCode != 124')
    $childHelper = Test-GoAssertion 'tools/process-sentinel/main_test.go' 'TestChildProcessHelper' @('case "fail"', 'os\.Exit\(7\)', 'case "wait"', 'time\.Sleep')
    Add-Check 'original-sentinel-test-assertions' ($observeZero -and $observeFailure -and $observeDeadline -and $childHelper) 'Original exit-zero, exit-seven, timeout-124, and fail/wait helper assertions remain present.'

    $parse = Test-GoAssertion 'internal/config/parse_test.go' 'TestParseConfig' @('\{"name":"relay","labels":\{"region":"eu"\}\}', 'err != nil', 'document\.Name != "relay"', 'document\.Labels\["region"\] != "eu"')
    Add-Check 'original-parser-test-assertions' $parse 'Original valid JSON, relay name, and eu region assertions remain present.'

    if ((Test-EffectiveTask 7) -and $arm -eq 'a') {
        $managed = Test-PythonAssertion 'scripts/test_check_operations.py' 'test_managed_roots_and_exact_exclusion' @('internal/config', 'internal/config/parse\.go', 'testdata/vendor-snapshots', 'immutable parser-test input; no production policy ownership')
        $fastChecks = Test-PythonAssertion 'scripts/test_check_operations.py' 'test_hook_and_pipeline_cover_each_declared_fast_check' @('Operations policy', 'Operations checker tests', 'Root module tests', 'Nested module tests', 'go vet \./\.\.\.')
        $checker = Read-RepoText 'scripts/check_operations.py'
        $runs = [regex]::Match($checker, '(?ms)EXPECTED_RUNS\s*=\s*\{(?<body>.*?)\}')
        $expectedCommands = $runs.Success -and $runs.Groups['body'].Value -match 'python scripts/check_operations\.py' -and
            $runs.Groups['body'].Value -match 'python -m unittest discover -s scripts' -and
            $runs.Groups['body'].Value -match 'go test \./\.\.\.' -and
            $runs.Groups['body'].Value -match 'go -C tools/process-sentinel test \./\.\.\.' -and
            $runs.Groups['body'].Value -match 'go vet \./\.\.\.'
        $oldGates = @('python scripts/check_operations.py', 'python -m unittest discover -s scripts', 'go test ./...', 'go -C tools/process-sentinel test ./...')
        $hook = Read-RepoText '.githooks/pre-commit'
        $ci = Read-RepoText '.github/workflows/ci.yaml'
        $allGates = (@($oldGates + 'go vet ./...') | Where-Object { $hook.Contains($_) -and $ci.Contains($_) }).Count -eq 5
        $guidance = @('docs/engineering/agent-rules.md', 'AGENTS.md', '.claude/CLAUDE.md') | ForEach-Object { (Read-RepoText $_) -match '(?i)go vet \./\.\.\.' }
        $ciScalar = $ci -match '(?m)^\s+run: go vet \./\.\.\.\s*$'
        Add-Check 'original-a7-checker-assertions' ($managed -and $fastChecks -and $expectedCommands) 'Managed-artifact/exclusion and all original fast-check expectations remain; vet was added.'
        Add-Check 'a7-guidance-and-gates' ($allGates -and $ciScalar -and (@($guidance | Where-Object { $_ }).Count -eq 3)) 'Canonical and provider guidance mention vet; hook and exact CI scalar retain all prior gates plus vet.'
    }
    if ((Test-EffectiveTask 7) -and $arm -eq 'b') {
        $project = Read-RepoText 'markitect.yaml'
        $checks = @('managed-artifacts', 'hook-and-pipeline-contracts', 'root-tests', 'nested-module-tests') | ForEach-Object { $project -match '(?m)^\s+- name: ' + [regex]::Escape($_) + '\s*$' }
        $verification = Read-RepoText '.markitect/areas/operations/verification.rule.yaml'
        $skill = Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml'
        $codex = Read-RepoText '.agents/skills/engineering-operations/SKILL.md'
        $claude = Read-RepoText '.claude/skills/engineering-operations/SKILL.md'
        $guidance = $verification -match '(?i)go vet \./\.\.\.' -and $verification -match '(?i)go test \./\.\.\.' -and
            $verification -match '(?i)go -C tools/process-sentinel test \./\.\.\.' -and
            $skill -match '(?i)verification' -and $codex -match '(?i)verification\.rule\.yaml' -and $claude -match '(?i)verification\.rule\.yaml'
        $hook = Read-RepoText '.githooks/pre-commit'
        $ci = Read-RepoText '.github/workflows/ci.yaml'
        $oldGates = @('go test ./...', 'go -C tools/process-sentinel test ./...')
        $allGates = (@($oldGates + 'go vet ./...') | Where-Object { $hook.Contains($_) -and $ci.Contains($_) }).Count -eq 3
        Add-Check 'original-b7-check-assertions' (@($checks | Where-Object { $_ }).Count -eq 4) 'All four configured Markitect checks remain enabled.'
        Add-Check 'b7-guidance-and-gates' ($guidance -and $allGates) 'Canonical verification guidance and provider Skill links remain; hook and CI retain root/nested tests plus vet.'
    }
}

function Restore-BaseAssertionFile([string]$TempRepo, [string]$Revision, [string]$RelativePath) {
    $spec = '{0}:{1}' -f $Revision, $RelativePath
    $blob = Invoke-Repo $script:repoPath 'git' @('show', $spec)
    if ($blob.exit -ne 0) { throw "Cannot load fixed task-start assertion file $spec : $($blob.output)" }
    $destination = Join-Path $TempRepo ($RelativePath -replace '/', [IO.Path]::DirectorySeparatorChar)
    $parent = Split-Path -Parent $destination
    if ($parent) { New-Item -ItemType Directory -Path $parent -Force | Out-Null }
    [IO.File]::WriteAllText($destination, $blob.output, [Text.UTF8Encoding]::new($false))
}

function Invoke-FixedBaselineChecks([string]$TempRepo, [int]$Index, [string]$Revision) {
    foreach ($path in @('tools/process-sentinel/main_test.go', 'internal/config/parse_test.go')) {
        Restore-BaseAssertionFile $TempRepo $Revision $path
    }
    $gates = if ($arm -eq 'a') {
        foreach ($path in @('scripts/check_operations.py', 'scripts/test_check_operations.py')) {
            Restore-BaseAssertionFile $TempRepo $Revision $path
        }
        @(
            @{ name = 'operations-policy'; exe = 'python'; args = @('scripts/check_operations.py') },
            @{ name = 'operations-checker-tests'; exe = 'python'; args = @('-m', 'unittest', 'discover', '-s', 'scripts') },
            @{ name = 'root-go-tests'; exe = 'go'; args = @('test', './...') },
            @{ name = 'nested-go-tests'; exe = 'go'; args = @('-C', 'tools/process-sentinel', 'test', './...') }
        )
    } else {
        @(
            @{ name = 'markitect-check'; exe = 'markitect'; args = @('check', '--repo', '.') },
            @{ name = 'markitect-render'; exe = 'markitect'; args = @('render', '--repo', '.', '--check') },
            @{ name = 'hook-pipeline-check'; exe = 'markitect-check-modules'; args = @('--repo', '.', '--hooks', '.markitect/modules/githooks.config', '--pipelines', '.markitect/modules/pipelines.config') },
            @{ name = 'artifact-check'; exe = 'markitect-check-artifacts'; args = @('--repo', '.', '--config', 'markitect-artifacts.yaml') },
            @{ name = 'root-go-tests'; exe = 'go'; args = @('test', './...') },
            @{ name = 'nested-go-tests'; exe = 'go'; args = @('-C', 'tools/process-sentinel', 'test', './...') }
        )
    }
    foreach ($gate in $gates) {
        $result = Invoke-Repo $TempRepo $gate.exe $gate.args
        Add-Check ('fixed-' + $gate.name) ($result.exit -eq 0) $(if ($result.exit -eq 0) { 'Frozen baseline gate passed in the external evaluation copy.' } else { $result.output })
    }
    if (Test-EffectiveTask 7) {
        $vet = Invoke-Repo $TempRepo 'go' @('vet', './...')
        Add-Check 'fixed-go-vet' ($vet.exit -eq 0) $(if ($vet.exit -eq 0) { 'Accepted vet gate passed in the external evaluation copy.' } else { $vet.output })
    }
}

function Add-TaskSpecificChecks([int]$Index) {
    if (Test-EffectiveTask 2) {
        $runbookPath = if (Test-Path -LiteralPath (Join-Path $script:repoPath 'docs/operations/runbooks/child-processes.md')) { 'docs/operations/runbooks/child-processes.md' } else { 'docs/operations/runbooks/process-sentinel.md' }
        $runbook = Read-RepoText $runbookPath
        $runbookIntent = $runbook -match '(?i)(seven|7)[ -]?second' -and $runbook -match '(?i)cancell' -and $runbook -match '(?i)(v1|version 1).{0,80}text|text.{0,80}(v1|version 1)'
        if ($arm -eq 'a') {
            $canonical = Read-RepoText 'docs/engineering/agent-rules.md'
            $codex = Read-RepoText 'AGENTS.md'
            $claude = Read-RepoText '.claude/CLAUDE.md'
            $instructions = $canonical -match '(?i)(seven|7)[ -]?second' -and $codex -match '(?i)(seven|7)[ -]?second' -and $claude -match '(?i)(seven|7)[ -]?second'
        } else {
            $canonical = Read-RepoText '.markitect/areas/operations/process-lifecycle.rule.yaml'
            $sourceSkill = Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml'
            $codex = Read-RepoText '.agents/skills/engineering-operations/SKILL.md'
            $claude = Read-RepoText '.claude/skills/engineering-operations/SKILL.md'
            $instructions = $sourceSkill -match '(?i)process-lifecycle' -and $codex -match '(?i)process-lifecycle\.rule\.yaml' -and $claude -match '(?i)process-lifecycle\.rule\.yaml'
        }
        $deadline = $canonical -match '(?i)(seven|7)[ -]?second'
        Add-Check 'cumulative-deadline-intent' ($deadline -and $instructions -and $runbookIntent) 'Seven-second deadline remains in the arm canonical owner, provider-facing path, and operational runbook; v1 text and cancellation remain documented.'
    }
    if (Test-EffectiveTask 7) {
        $hook = Read-RepoText '.githooks/pre-commit'
        $ci = Read-RepoText '.github/workflows/ci.yaml'
        Add-Check 'cumulative-vet-gates' ($hook -match '(?m)go vet ./\.\.\.' -and $ci -match 'go vet ./\.\.\.') 'go vet ./... remains enabled in both pre-commit and CI.'
    }
    if (Test-EffectiveTask 5) {
        $newPath = Join-Path $script:repoPath 'docs/operations/runbooks/development-start.md'
        $new = Test-Path -LiteralPath $newPath -PathType Leaf
        $old = Test-Path -LiteralPath (Join-Path $script:repoPath 'docs/operations/runbooks/legacy-start.md')
        $text = if ($new) { Get-Content -LiteralPath $newPath -Raw } else { '' }
        $commands = $text -match '(?i)go test \./\.\.\.' -and $text -match '(?i)go -C tools/process-sentinel test \./\.\.\.' -and $text -match '(?i)process-sentinel'
        $stale = Test-TextInSource @('AGENTS.md', '.claude', '.agents', '.markitect', 'docs') 'legacy-start\.md'
        $declared = Test-DeclaredRunbook 'docs/operations/runbooks/development-start.md'
        $staleDeclared = if ($arm -eq 'a') { (Read-RepoText 'docs/engineering/managed-files.yaml') -match '(?m)^\s+docs/operations/runbooks/legacy-start\.md:' } else { (Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml') -match '(?m)^\s+- docs/operations/runbooks/legacy-start\.md\s*$' }
        Add-Check 'development-start-migration' ($new -and -not $old -and $commands -and -not $stale -and $declared -and -not $staleDeclared) 'Replacement runbook is declared, retains root/nested commands, and replaces active legacy references.'
    }
    if (Test-EffectiveTask 8) {
        $runbook = Read-RepoText 'docs/operations/runbooks/lease-renewal.md'
        $hasIdempotency = $runbook -match '(?i)idempot'
        $hasRequestId = $runbook -match '(?i)request[ -]?id'
        $hasTimeout = $runbook -match '(?i)timeout|deadline'
        $secretsStayPrivate = $runbook -match '(?is)(secret|credential).{0,80}(log|logging)|(log|logging).{0,80}(secret|credential)'
        $payloadStaysPrivate = $runbook -match '(?is)payload.{0,80}(log|logging)|(log|logging).{0,80}payload'
        $declared = Test-DeclaredRunbook 'docs/operations/runbooks/lease-renewal.md'
        Add-Check 'lease-renewal-runbook' ($hasIdempotency -and $hasRequestId -and $hasTimeout -and $secretsStayPrivate -and $payloadStaysPrivate -and $declared) 'Runbook covers idempotency, request ID, timeout, secret/payload log privacy, and its arm-owned input declaration.'
    }
    if (Test-EffectiveTask 9) {
        $new = Test-Path -LiteralPath (Join-Path $script:repoPath 'docs/operations/runbooks/child-processes.md') -PathType Leaf
        $old = Test-Path -LiteralPath (Join-Path $script:repoPath 'docs/operations/runbooks/process-sentinel.md')
        $stale = Test-TextInSource @('AGENTS.md', '.claude', '.agents', '.markitect', 'docs') 'process-sentinel\.md'
        $declared = Test-DeclaredRunbook 'docs/operations/runbooks/child-processes.md'
        $staleDeclared = if ($arm -eq 'a') { (Read-RepoText 'docs/engineering/managed-files.yaml') -match '(?m)^\s+docs/operations/runbooks/process-sentinel\.md:' } else { (Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml') -match '(?m)^\s+- docs/operations/runbooks/process-sentinel\.md\s*$' }
        $facts = Test-TextInSource @('docs/operations/runbooks/child-processes.md') '(?i)(v1|version 1).{0,80}text|text.{0,80}(v1|version 1)' -and
            (Test-TextInSource @('docs/operations/runbooks/child-processes.md') '(?i)cancell') -and
            (Test-TextInSource @('docs/operations/runbooks/child-processes.md') '(?i)timeout') -and
            (Test-TextInSource @('docs/operations/runbooks/child-processes.md') '(?i)secret|credential') -and
            (Test-TextInSource @('docs/operations/runbooks/child-processes.md') '(?i)payload')
        Add-Check 'child-processes-runbook-migration' ($new -and -not $old -and -not $stale -and -not $staleDeclared -and $declared -and $facts) 'Runbook renamed, declared, and retains v1 text, cancellation, timeout, and privacy facts.'
    }
    if (Test-EffectiveTask 10) {
        $deleted = -not (Test-Path -LiteralPath (Join-Path $script:repoPath 'docs/operations/runbooks/legacy-bulk-replay.md'))
        $stale = Test-TextInSource @('AGENTS.md', '.claude', '.agents', '.markitect', 'docs') 'legacy-bulk-replay\.md'
        $staleDeclared = if ($arm -eq 'a') { (Read-RepoText 'docs/engineering/managed-files.yaml') -match '(?m)^\s+docs/operations/runbooks/legacy-bulk-replay\.md:' } else { (Read-RepoText '.markitect/areas/operations/engineering-operations.skill.yaml') -match '(?m)^\s+- docs/operations/runbooks/legacy-bulk-replay\.md\s*$' }
        Add-Check 'legacy-bulk-replay-retirement' ($deleted -and -not $stale -and -not $staleDeclared) 'Deleted path and stale ownership references are absent.'
    }
}

try {
    if ($parallelTask) {
        if ($PriorTasksThrough -and $PriorTasksThrough -ne '06') { throw "Parallel task $Task requires PriorTasksThrough=06; received $PriorTasksThrough." }
        $script:effectiveTasks = @(1..6) + @($taskIndex)
    } else {
        $script:effectiveTasks = @(1..$taskIndex)
    }
    $card = Get-TaskCard $Task
    $decisionRequired = $card -match '(?m)^\s+requires_owner_decision: true\s*$'
    $scopes = Get-TaskScopes $card $arm
    $validatorMatch = [regex]::Match($card, '(?m)^\s+validator: \[(?<items>[^\]]*)\]')
    if (-not $validatorMatch.Success) { throw "Task $Task has no declared validator." }
    $validatorCommand = Parse-YamlList $validatorMatch.Groups['items'].Value
    if ($validatorCommand.Count -eq 0) { throw "Task $Task has an empty validator command." }
    if ([string]::IsNullOrWhiteSpace($Base)) { throw 'Exact task-start Base is required; no parent commit is inferred.' }
    $script:changed = Get-ChangedPaths $Base
    $outside = @($script:changed | Where-Object { -not (Test-InScope $_ $scopes) })
    Add-Check 'allowed-paths' ($outside.Count -eq 0) $(if ($outside.Count) { 'Out of scope: ' + ($outside -join ', ') } else { 'All changed and untracked paths fit the declared arm scope.' })

    $fixture = Join-Path $repoPath 'testdata/vendor-snapshots/sample.json'
    $fixtureOk = (Test-Path -LiteralPath $fixture -PathType Leaf) -and ((Get-FileHash -LiteralPath $fixture -Algorithm SHA256).Hash.ToLowerInvariant() -eq 'b700afecd52da6090288e13537d3efd6da7e2743d3b5fc2acd190f6c0efe68e8')
    Add-Check 'protected-golden-fixture' $fixtureOk 'Protected sample.json SHA-256 is fixed by the evaluator.'

    Add-TaskSpecificChecks $taskIndex
    Add-OriginalAssertionChecks $taskIndex
    if ($decisionRequired) {
        Add-Check 'no-unauthorized-mutation' ($script:changed.Count -eq 0) 'Owner decision is pending; this card must not mutate the project.'
    }

    $tempRoot = Join-Path ([IO.Path]::GetTempPath()) ('engineering-ops-oracle-' + [guid]::NewGuid().ToString('N'))
    $tempRepo = Join-Path $tempRoot 'repo'
    try {
        Copy-SourceTree $repoPath $tempRepo
        $validatorArgs = @($validatorCommand | Select-Object -Skip 1)
        $validator = Invoke-Repo $tempRepo $validatorCommand[0] $validatorArgs
        Add-Check 'task-validator' ($validator.exit -eq 0) $(if ($validator.exit -eq 0) { 'Declared public project validator passed in an external evaluation copy.' } else { $validator.output })
        Invoke-FixedBaselineChecks $tempRepo $taskIndex $Base
        if (Test-EffectiveTask 1) { Invoke-HiddenVector $tempRepo 'task01_process_json_test.go.txt' 'TestGauntletSentinelJSONAndTextCompatibility' 'tools/process-sentinel' 'tools/process-sentinel' }
        if (Test-EffectiveTask 2) { Invoke-HiddenVector $tempRepo 'task02_deadline_test.go.txt' 'TestGauntletSevenSecondDefaultAndCancellation' 'tools/process-sentinel' 'tools/process-sentinel' }
        if (Test-EffectiveTask 6) { Invoke-HiddenVector $tempRepo 'task06_relay_version_test.go.txt' 'TestGauntletRelayVersionAndDefaultOutput' 'cmd/relayctl' 'cmd/relayctl' }
        if (Test-EffectiveTask 11) { Invoke-HiddenVector $tempRepo 'task11_fixture_immutable_test.go.txt' 'TestGauntletParserDoesNotMutateFixtureBytes' 'internal/config' 'internal/config' }
    } finally {
        if (Test-Path -LiteralPath $tempRoot) { Remove-Item -LiteralPath $tempRoot -Recurse -Force }
    }

    if ($decisionRequired -and $script:changed.Count -eq 0 -and $script:failures.Count -eq 0) { $status = 'owner_decision_required' }
    elseif ($script:failures.Count -gt 0) { $status = 'failed' }
    else { $status = 'passed' }
    $ownerDecision = if ($decisionRequired) { 'external assessment required; no automatic acceptance' } else { 'not required' }
} catch {
    Add-Check 'evaluator-available' $false ($_.Exception.Message + ' ' + $_.ScriptStackTrace)
    $status = 'unavailable'
    $ownerDecision = if ($decisionRequired) { 'external assessment required; no automatic acceptance' } else { 'not required' }
}

$changedText = $script:changed -join ', '
$checkLines = if ($script:checks.Count -eq 0) { '  []' } else {
    ($script:checks | ForEach-Object { "  - name: $(Quote-Yaml $_.name)`n    passed: $($_.passed.ToString().ToLowerInvariant())`n    evidence: $(Quote-Yaml $_.evidence)" }) -join "`n"
}
$failureLines = if ($script:failures.Count -eq 0) { '  []' } else { ($script:failures | ForEach-Object { '  - ' + (Quote-Yaml $_) }) -join "`n" }
$baseOutput = if ([string]::IsNullOrWhiteSpace($Base)) { 'unavailable' } else { $Base }
$yaml = "task: $(Quote-Yaml $Task)`narm: $arm`nstatus: $status`nbase: $(Quote-Yaml $baseOutput)`nchanged_paths: $(Quote-Yaml $changedText)`nchecks:`n$checkLines`nfailures:`n$failureLines`nowner_decision: $(Quote-Yaml $ownerDecision)"
$outputPath = [IO.Path]::GetFullPath($Output)
$outputDirectory = Split-Path -Parent $outputPath
if ($outputDirectory) { New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null }
[IO.File]::WriteAllText($outputPath, $yaml, [Text.UTF8Encoding]::new($false))
$yaml
