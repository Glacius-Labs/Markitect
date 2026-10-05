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

function Select-TaskScopes([string]$TaskId, [string]$PriorThrough, [string[]]$TaskScopes, [string[]]$P01Scopes, [string[]]$P02Scopes) {
    if ($TaskId -match '^P0[1-2]$') {
        if ($PriorThrough -and $PriorThrough -ne '06') { throw "Parallel task $TaskId requires PriorTasksThrough=06; received $PriorThrough." }
        return @($TaskScopes)
    }
    if ($TaskId -eq '08' -and $PriorThrough -eq '06') {
        return @($P01Scopes + $P02Scopes | Sort-Object -Unique)
    }
    if ($PriorThrough) { throw "PriorTasksThrough=$PriorThrough is unsupported for task $TaskId." }
    return @($TaskScopes)
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

function Test-A7FastCheckAssertionContract([string]$RepoRoot) {
    # Read only the task's literal dict and literal unittest set; candidate code is never imported.
    $python = @'
import ast
import pathlib
import sys

original = {
    "Operations policy": "python scripts/check_operations.py",
    "Operations checker tests": "python -m unittest discover -s scripts",
    "Root module tests": "go test ./...",
    "Nested module tests": "go -C tools/process-sentinel test ./...",
}

def fail(message):
    print(message, file=sys.stderr)
    raise SystemExit(1)

def read_literal_dict(path, name):
    tree = ast.parse(pathlib.Path(path).read_text(encoding="utf-8"))
    matches = []
    for node in tree.body:
        if isinstance(node, ast.Assign) and any(
            isinstance(target, ast.Name) and target.id == name for target in node.targets
        ):
            matches.append(node.value)
        elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name) and node.target.id == name:
            matches.append(node.value)
    if len(matches) != 1:
        fail(f"expected one literal {name} assignment")
    value = ast.literal_eval(matches[0])
    if type(value) is not dict or any(type(k) is not str or type(v) is not str for k, v in value.items()):
        fail(f"{name} must be a literal string-to-string dict")
    return value

def is_expected_runs_set(node):
    return (
        isinstance(node, ast.Call)
        and isinstance(node.func, ast.Name)
        and node.func.id == "set"
        and len(node.args) == 1
        and isinstance(node.args[0], ast.Name)
        and node.args[0].id == "EXPECTED_RUNS"
    )

checker_path, tests_path = sys.argv[1:3]
runs = read_literal_dict(checker_path, "EXPECTED_RUNS")
if len(runs) != len(original) + 1 or any(runs.get(k) != v for k, v in original.items()):
    fail("the four original EXPECTED_RUNS entries must remain exact with one addition")
added = [(key, value) for key, value in runs.items() if key not in original]
if len(added) != 1 or added[0][1] != "go vet ./...":
    fail("the one added EXPECTED_RUNS entry must map its literal key to go vet ./...")
expected_names = set(runs)

tree = ast.parse(pathlib.Path(tests_path).read_text(encoding="utf-8"))
methods = [n for n in ast.walk(tree) if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))
           and n.name == "test_hook_and_pipeline_cover_each_declared_fast_check"]
if len(methods) != 1:
    fail("expected exactly one declared-fast-check test method")
literal_sets = []
# Only inspect direct method-body statements: nested functions and conditional/dead
# branches are not evidence that the unittest actually asserts the contract.
for statement in methods[0].body:
    node = statement.value if isinstance(statement, ast.Expr) else None
    if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Attribute) or node.func.attr != "assertEqual" or len(node.args) != 2:
        continue
    if not isinstance(node.func.value, ast.Name) or node.func.value.id != "self":
        continue
    for left, right in ((node.args[0], node.args[1]), (node.args[1], node.args[0])):
        if is_expected_runs_set(left) and isinstance(right, ast.Set):
            try:
                value = ast.literal_eval(right)
            except (ValueError, TypeError):
                continue
            if type(value) is set and all(type(item) is str for item in value):
                statement_index = methods[0].body.index(statement)
                def is_bounded_terminator(previous):
                    if isinstance(previous, (ast.Return, ast.Raise)):
                        return True
                    # Reject only an obvious direct literal guard, not general reachability.
                    if isinstance(previous, ast.If) and isinstance(previous.test, ast.Constant) and previous.test.value is True:
                        return any(isinstance(item, (ast.Return, ast.Raise)) for item in previous.body)
                    return False
                if any(is_bounded_terminator(previous) for previous in methods[0].body[:statement_index]):
                    fail("the literal assertion follows a direct or literal-if-True terminating statement")
                literal_sets.append(value)
if len(literal_sets) != 1 or literal_sets[0] != expected_names:
    fail("the test must compare set(EXPECTED_RUNS) with the four original names plus the mapped vet key")
print("PASS")
'@
    $checkerPath = Join-Path $RepoRoot 'scripts/check_operations.py'
    $testsPath = Join-Path $RepoRoot 'scripts/test_check_operations.py'
    Push-Location $RepoRoot
    try {
        $output = @(& python -c $python $checkerPath $testsPath 2>&1)
        $exit = $LASTEXITCODE
        return $exit -eq 0 -and (($output -join "`n").Trim() -eq 'PASS')
    } catch {
        return $false
    } finally {
        Pop-Location
    }
}

function Get-ProjectChecks([string]$Project) {
    # Bounded YAML-subset reader for checks/name/run argv only; other YAML forms are unsupported.
    $lines = $Project -split "`r?`n"
    $checks = [System.Collections.Generic.List[object]]::new()
    $inChecks = $false
    $checkIndent = -1
    $current = $null
    $runIndent = -1
    $argv = [System.Collections.Generic.List[string]]::new()
    foreach ($line in $lines) {
        if (-not $inChecks) {
            if ($line -match '^(?<indent> *)checks:\s*(?:#.*)?$') {
                $inChecks = $true
                $checkIndent = $Matches.indent.Length
            }
            continue
        }
        if ($line -match '^\S') { break }
        if ($line -match '^(?<indent> *)-\s+name:\s*(?<name>[^#]+?)\s*$' -and $Matches.indent.Length -gt $checkIndent) {
            if ($null -ne $current) {
                $current.Argv = @($argv)
                $checks.Add($current)
            }
            $current = [pscustomobject]@{ Name = $Matches.name.Trim().Trim('"').Trim("'"); Argv = @() }
            $runIndent = -1
            $argv = [System.Collections.Generic.List[string]]::new()
            continue
        }
        if ($null -eq $current) { continue }
        if ($line -match '^(?<indent> *)run:\s*(?:#.*)?$') {
            $runIndent = $Matches.indent.Length
            continue
        }
        if ($runIndent -ge 0 -and $line -match '^(?<indent> *)-\s*(?<value>.*?)\s*$' -and $Matches.indent.Length -gt $runIndent) {
            $argv.Add($Matches.value.Trim().Trim('"').Trim("'"))
            continue
        }
        if ($line -match '^\s*\S' -and $line -notmatch '^\s+-\s') { $runIndent = -1 }
    }
    if ($null -ne $current) {
        $current.Argv = @($argv)
        $checks.Add($current)
    }
    return @($checks)
}

function Test-ArgvEquals([string[]]$Actual, [string[]]$Expected) {
    if ($Actual.Count -ne $Expected.Count) { return $false }
    for ($i = 0; $i -lt $Actual.Count; $i++) {
        if ($Actual[$i] -cne $Expected[$i]) { return $false }
    }
    return $true
}

function Test-ProjectCheckArgv([object[]]$Checks, [string]$Name, [string[]]$ExpectedArgv) {
    $matches = @($Checks | Where-Object { $_.Name -ceq $Name })
    return $matches.Count -eq 1 -and (Test-ArgvEquals @($matches[0].Argv) $ExpectedArgv)
}

function Test-ActiveHookCommand([string]$Hook, [string]$Command) {
    $inFunction = $false
    $stopped = $false
    foreach ($line in ($Hook -split "`r?`n")) {
        if ($stopped) { break }
        $trimmed = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed) -or $trimmed.StartsWith('#')) { continue }
        if ($inFunction) {
            if ($trimmed -match '^}\s*(?:#.*)?$') { $inFunction = $false }
            continue
        }
        if ($trimmed -match '^(?:function\s+[A-Za-z_][\w-]*\s*(?:\(\s*\))?|[A-Za-z_][\w-]*\s*\(\s*\))\s*\{') {
            $inFunction = $trimmed -notmatch '\}\s*(?:#.*)?$'
            continue
        }
        $trimmed = ($trimmed -replace '\s+#.*$', '').Trim()
        $segments = @($trimmed -split '\s*(?:&&|;)\s*' | ForEach-Object { $_.Trim() })
        foreach ($segment in $segments) {
            if ($segment -match '^exit(?:\s+\d+)?$') { $stopped = $true; break }
            if ($segment -ceq $Command) { return $true }
        }
    }
    return $false
}

function Get-CIQualitySteps([string]$Ci) {
    # Bounded reader for top-level quality.steps. It preserves every step index and
    # excludes commands with direct false/continue-on-error controls; it is not a YAML parser.
    $lines = $Ci -split "`r?`n"
    $inJobs = $false; $jobsIndent = -1; $inQuality = $false; $qualityIndent = -1
    $inSteps = $false; $stepsIndent = -1; $stepIndent = -1
    $steps = [System.Collections.Generic.List[object]]::new()
    $current = $null
    foreach ($line in $lines) {
        if ($line -match '^\s*#' -or $line.Trim() -eq '') { continue }
        if (-not $inJobs) {
            if ($line -match '^(?<indent> *)jobs:\s*(?:#.*)?$') { $inJobs = $true; $jobsIndent = $Matches.indent.Length }
            continue
        }
        $indent = ([regex]::Match($line, '^\s*')).Length
        if (-not $inQuality) {
            if ($indent -le $jobsIndent) { break }
            if ($line -match '^(?<indent> *)quality:\s*(?:#.*)?$' -and $Matches.indent.Length -gt $jobsIndent) { $inQuality = $true; $qualityIndent = $Matches.indent.Length }
            continue
        }
        if (-not $inSteps) {
            if ($indent -le $qualityIndent) { break }
            if ($line -match '^(?<indent> *)steps:\s*(?:#.*)?$' -and $Matches.indent.Length -gt $qualityIndent) { $inSteps = $true; $stepsIndent = $Matches.indent.Length }
            continue
        }
        if ($indent -le $stepsIndent) { break }
        if ($line -match '^(?<indent> *)-\s*(?<rest>.*)$' -and $Matches.indent.Length -gt $stepsIndent) {
            $candidateIndent = $Matches.indent.Length
            if ($stepIndent -ge 0 -and $candidateIndent -gt $stepIndent) { continue }
            if ($stepIndent -ge 0 -and $candidateIndent -eq $stepIndent) { $steps.Add($current) }
            if ($stepIndent -ge 0 -and $candidateIndent -lt $stepIndent) { return @() }
            $stepIndent = $candidateIndent
            $current = [pscustomobject]@{ Run = $null; Inactive = $false; Malformed = $false }
            if ($Matches.rest -match '^run:\s*(?<value>.*)$') { $current.Run = $Matches.value.Trim() }
            continue
        }
        if ($stepIndent -ge 0 -and $indent -eq ($stepIndent + 2)) {
            if ($line -match '^\s+run:\s*(?<value>.*)$') {
                if ($null -ne $current.Run) { $current.Malformed = $true } else { $current.Run = $Matches.value.Trim() }
            } elseif ($line -match '^\s+if:\s*(?<value>.*?)\s*(?:#.*)?$') {
                $condition = $Matches.value.Trim().Trim('"').Trim("'")
                if ($condition -match '^(?i:false|\$\{\{\s*false\s*\}\})$') { $current.Inactive = $true }
                elseif ($condition -notmatch '^\S+$') { $current.Malformed = $true }
            } elseif ($line -match '^\s+continue-on-error:\s*(?<value>.*?)\s*(?:#.*)?$') {
                $setting = $Matches.value.Trim().Trim('"').Trim("'")
                if ($setting -match '^(?i:true)$') { $current.Inactive = $true }
                elseif ($setting -notmatch '^(?i:false)$') { $current.Malformed = $true }
            }
        }
    }
    if ($stepIndent -ge 0) { $steps.Add($current) }
    return @($steps)
}

function Get-CINormalizedRun([object]$Step) {
    if ($null -eq $Step -or $Step.Inactive -or $Step.Malformed) { return $null }
    $value = [string]$Step.Run
    if ([string]::IsNullOrWhiteSpace($value) -or $value -in @('|', '>', '|-', '>-')) { return $null }
    if ($value -match '^("[''])(.*)\1$') { $value = $Matches[2] }
    if ($value -match '\s+#') { $value = ($value -split '\s+#', 2)[0].Trim() }
    return $value
}

function Test-ActiveCIRunCommand([string]$Ci, [string]$Command) {
    foreach ($step in (Get-CIQualitySteps $Ci)) {
        if ((Get-CINormalizedRun $step) -ceq $Command) { return $true }
    }
    return $false
}
function Test-ActiveCheckGates([string]$Hook, [string]$Ci, [string[]]$Commands) {
    foreach ($command in $Commands) {
        if (-not (Test-ActiveHookCommand $Hook $command) -or -not (Test-ActiveCIRunCommand $Ci $command)) { return $false }
    }
    return $true
}

function Get-PipelineCheckPointer([string]$PipelineConfig, [string]$Name) {
    $lines = $PipelineConfig -split "`r?`n"
    $inExpected = $false
    $expectedIndent = -1
    $entryIndent = -1
    $currentName = ''
    $pointers = [System.Collections.Generic.List[string]]::new()
    foreach ($line in $lines) {
        if (-not $inExpected) {
            if ($line -match '^(?<indent> *)expectedChecks:\s*(?:#.*)?$') {
                $inExpected = $true
                $expectedIndent = $Matches.indent.Length
            }
            continue
        }
        if ($line -match '^\s*#' -or $line.Trim() -eq '') { continue }
        $lineIndent = ([regex]::Match($line, '^\s*')).Length
        if ($lineIndent -le $expectedIndent) { break }
        if ($line -match '^(?<indent> *)-\s+name:\s*(?<name>[^#]+?)\s*$' -and $Matches.indent.Length -gt $expectedIndent) {
            $entryIndent = $Matches.indent.Length
            $currentName = $Matches.name.Trim().Trim('"').Trim("'")
            continue
        }
        if ($currentName -ceq $Name -and $lineIndent -gt $entryIndent -and $line -match '^\s*yamlPath:\s*(?<pointer>[^#]+?)\s*(?:#.*)?$') {
            $pointers.Add($Matches.pointer.Trim().Trim('"').Trim("'"))
        }
    }
    if ($pointers.Count -ne 1) { return $null }
    return $pointers[0]
}

function Get-CIQualityRunAtPointer([string]$Ci, [string]$Pointer) {
    $pointerMatch = [regex]::Match($Pointer, '^/jobs/quality/steps/(?<index>\d+)/run$')
    if (-not $pointerMatch.Success) { return $null }
    $steps = @(Get-CIQualitySteps $Ci)
    $index = [int]$pointerMatch.Groups['index'].Value
    if ($index -lt 0 -or $index -ge $steps.Count) { return $null }
    return Get-CINormalizedRun $steps[$index]
}
function Test-B7VerificationAndGateContract([string]$RepoRoot) {
    $verification = Get-Content -LiteralPath (Join-Path $RepoRoot '.markitect/areas/operations/verification.rule.yaml') -Raw
    $skill = Get-Content -LiteralPath (Join-Path $RepoRoot '.markitect/areas/operations/engineering-operations.skill.yaml') -Raw
    $codex = Get-Content -LiteralPath (Join-Path $RepoRoot '.agents/skills/engineering-operations/SKILL.md') -Raw
    $claude = Get-Content -LiteralPath (Join-Path $RepoRoot '.claude/skills/engineering-operations/SKILL.md') -Raw
    $project = Get-Content -LiteralPath (Join-Path $RepoRoot 'markitect.yaml') -Raw
    $pipelineConfig = Get-Content -LiteralPath (Join-Path $RepoRoot '.markitect/modules/pipelines.config') -Raw
    $hookConfig = Get-Content -LiteralPath (Join-Path $RepoRoot '.markitect/modules/githooks.config') -Raw
    $hook = Get-Content -LiteralPath (Join-Path $RepoRoot '.githooks/pre-commit') -Raw
    $ci = Get-Content -LiteralPath (Join-Path $RepoRoot '.github/workflows/ci.yaml') -Raw

    # Guidance may name checks semantically; their exact argv remains owned by Project and module inputs below.
    $hasGuidance = $verification -match '(?i)go vet|\bvet\b' -and
        ($verification -match '(?i)root\s+(?:module\s+)?Go tests' -or $verification -match '(?i)go test \./\.\.\.') -and
        ($verification -match '(?i)nested[- ]module Go tests' -or $verification -match '(?i)go -C tools/process-sentinel test \./\.\.\.') -and
        $verification -match '(?i)Markitect'
    $hasDeclaredInputs = @('.githooks/pre-commit', '.github/workflows/ci.yaml', '.markitect/modules/githooks.config', '.markitect/modules/pipelines.config') | Where-Object {
        $verification -match '(?m)^\s+- ' + [regex]::Escape($_) + '\s*$'
    }
    $hasSkillRoute = $skill -match '(?m)^\s+- name: verification\s*$' -and
        $codex -match '(?i)verification\.rule\.yaml' -and $claude -match '(?i)verification\.rule\.yaml'

    $projectChecks = Get-ProjectChecks $project
    $projectCheckContracts = @(
        (Test-ProjectCheckArgv $projectChecks 'managed-artifacts' @('markitect-check-artifacts', '--repo', '.', '--config', 'markitect-artifacts.yaml')),
        (Test-ProjectCheckArgv $projectChecks 'hook-and-pipeline-contracts' @('markitect-check-modules', '--repo', '.', '--hooks', '.markitect/modules/githooks.config', '--pipelines', '.markitect/modules/pipelines.config')),
        (Test-ProjectCheckArgv $projectChecks 'root-tests' @('go', 'test', './...')),
        (Test-ProjectCheckArgv $projectChecks 'nested-module-tests' @('go', '-C', 'tools/process-sentinel', 'test', './...'))
    )
    $vetChecks = @($projectChecks | Where-Object { Test-ArgvEquals @($_.Argv) @('go', 'vet', './...') })
    $pipelinePath = $pipelineConfig -match '(?m)^\s+path: \.github/workflows/ci\.yaml\s*$' -and
        $pipelineConfig -match '(?m)^\s+owner: operations/Rule/verification\s*$'
    $rootTestsPointer = Get-PipelineCheckPointer $pipelineConfig 'root-tests'
    $nestedTestsPointer = Get-PipelineCheckPointer $pipelineConfig 'nested-module-tests'
    $rootTestsLinked = $null -ne $rootTestsPointer -and (Get-CIQualityRunAtPointer $ci $rootTestsPointer) -ceq 'go test ./...'
    $nestedTestsLinked = $null -ne $nestedTestsPointer -and (Get-CIQualityRunAtPointer $ci $nestedTestsPointer) -ceq 'go -C tools/process-sentinel test ./...'
    $vetPipelineLink = $false
    if ($vetChecks.Count -eq 1) {
        $vetPointer = Get-PipelineCheckPointer $pipelineConfig $vetChecks[0].Name
        $vetPipelineLink = $null -ne $vetPointer -and (Get-CIQualityRunAtPointer $ci $vetPointer) -ceq 'go vet ./...'
    }
    $hookLink = $hookConfig -match '(?m)^\s+path: \.githooks/pre-commit\s*$' -and
        $hookConfig -match '(?m)^\s+owner: operations/Rule/verification\s*$'
    $hookCommands = @(
        'markitect check --repo .',
        'markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config',
        'markitect-check-artifacts --repo . --config markitect-artifacts.yaml',
        'go vet ./...', 'go test ./...', 'go -C tools/process-sentinel test ./...'
    ) | Where-Object { Test-ActiveHookCommand $hook $_ }
    $ciProjection = Test-ActiveCIRunCommand $ci 'markitect verify --repo . --revision $GITHUB_SHA'
    $originalNames = @('managed-artifacts', 'hook-and-pipeline-contracts', 'root-tests', 'nested-module-tests')
    $presentOriginals = @($projectChecks | Where-Object { $_.Name -cin $originalNames } | Select-Object -ExpandProperty Name -Unique)
    return $hasGuidance -and $hasSkillRoute -and $hasDeclaredInputs.Count -eq 4 -and
        (@($projectCheckContracts | Where-Object { $_ }).Count -eq 4) -and $presentOriginals.Count -eq 4 -and
        $vetChecks.Count -eq 1 -and $pipelinePath -and $rootTestsLinked -and $nestedTestsLinked -and $vetPipelineLink -and $hookLink -and
        $hookCommands.Count -eq 6 -and $ciProjection
}

function Test-LeaseRenewalRunbookContract([string]$Runbook, [bool]$Declared) {
    # These bounded alternatives recognize this runbook's retry contract, not general natural-language equivalence.
    $sameIDRetry = $Runbook -match '(?is)(same|original)\s+request[ -]?id.{0,80}(every|each|all)\s+(retry|retries|attempt)' -or
        $Runbook -match '(?is)(every|each|all)\s+(retry|retries|attempt).{0,80}(same|original)\s+request[ -]?id' -or
        $Runbook -match '(?is)request[ -]?id.{0,220}(retry|retries|attempt|attempts).{0,80}(?:for|of)\s+(?:(?:the|that)\s+)?same\s+(?:logical\s+)?(?:renewal|operation).{0,100}(?:must|should|will)\s+reuse\s+(?:it|that\s+(?:same\s+)?(?:request[ -]?)?id|the\s+(?:same|original)\s+request[ -]?id)\b' -or
        $Runbook -match '(?is)request[ -]?id.{0,160}(?:for|of)\s+(?:(?:the|that)\s+)?same\s+(?:logical\s+)?(?:renewal|operation).{0,120}(retry|retries|attempt|attempts).{0,80}(?:must|should|will)\s+(?:reuse|use)\s+(?:it|that\s+(?:same\s+)?(?:request[ -]?)?id|the\s+(?:same|original)\s+request[ -]?id)\b'
    $freshIDRetry = $Runbook -match '(?is)(?:each|every|all)\s+(?:retry|retries|attempt|attempts).{0,80}(?:must|should|will)\s+(?:use|assign|generate|create)\s+(?:a\s+)?(?:fresh|new|different)\s+(?:request[ -]?)?id\b' -or
        $Runbook -match '(?is)(?:retry|retries|attempt|attempts).{0,80}(?:must|should|will)\s+(?:use|assign|generate|create)\s+(?:a\s+)?(?:fresh|new|different)\s+(?:request[ -]?)?id\b'
    $deduplicatedRetry = $Runbook -match '(?is)(recogniz\w*|deduplicat\w*).{0,100}(repeated|duplicate|same)\s+(operation|renewal)' -or
        $Runbook -match '(?is)(retry|retries).{0,100}(must not|does not|cannot|will not)\s+(create|cause|start).{0,60}(second|duplicate)\s+(renewal|operation)'
    $duplicateRisk = $false
    foreach ($clause in [regex]::Split($Runbook, '[.!?\r\n]+')) {
        $warnsOfRisk = $clause -match '(?is)(retry|retries|repeat|repeated operation).{0,80}(?:may|can|could|will|might)\s+(?!not\b|never\b).{0,40}(create|cause|start).{0,60}(second|duplicate)\s+(renewal|operation)'
        $correctsRisk = $clause -match '(?is)(therefore|thus|so|instead).{0,60}(reuse|use|retry with).{0,40}(same|original)\s+(?:request[ -]?)?id'
        $normativeFreshID = $clause -match '(?i)^\s*(?:retry|use|assign|generate)\s+(?:with\s+)?(?:a\s+)?fresh(?:\s+request)?[ -]?id\b'
        if ($warnsOfRisk -and (-not $correctsRisk -or $normativeFreshID)) {
            $duplicateRisk = $true
            break
        }
    }
    $retentionDuration = $Runbook -match '(?is)(keep|retain|store).{0,80}request[ -]?id.{0,100}\b(for|during|throughout)\b.{0,50}\b(?:\d+\s*(?:hours?|days?|weeks?|months?)|(?:retry|reconciliation|retention)\s+(?:period|window|lifecycle))\b' -or
        $Runbook -match '(?is)request[ -]?id.{0,80}\b(?:for\s+\d+\s*(?:hours?|days?|weeks?|months?)|until\s+(?:the\s+)?(?:final outcome|record expiry|operation (?:ends|completes|is finalized)))\b'
    $recordScope = $Runbook -match '(?is)(keep|retain|store).{0,80}request[ -]?id.{0,100}(?:with|in|on|as part of).{0,50}(?:operation|renewal|request) record' -or
        $Runbook -match '(?is)(operation|renewal|request) record.{0,100}request[ -]?id'
    $retentionPurpose = $Runbook -match '(?is)(keep|retain|store).{0,80}request[ -]?id.{0,140}(retr(?:y|ies|ying)|reconcil|final outcome)'
    $requestIDRetained = $retentionDuration -or $recordScope -or $retentionPurpose
    $timeoutReporting = $false
    $unsafeTimeoutSuccess = $false
    foreach ($clause in [regex]::Split($Runbook, '[.!?\r\n]+')) {
        $hasTimeout = $clause -match '(?i)timeout|time[- ]?out|timed out|times[- ]?out|deadline'
        if ($hasTimeout -and $clause -match '(?i)\b(report|notify|tell|record|escalate|communicate)\b') { $timeoutReporting = $true }
        if ($hasTimeout) {
            foreach ($successMatch in [regex]::Matches($clause, '(?i)\b(success|successful|succeeded|complete|completed)\b')) {
                $prefixStart = [Math]::Max(0, $successMatch.Index - 45)
                $prefix = $clause.Substring($prefixStart, $successMatch.Index - $prefixStart)
                $negatesSuccess = ($prefix -match '(?i)\b(?:not|never|no|rather than|instead of|without|do not|don\x27t)\b[^,;]{0,50}$') -or
                    ($prefix -match ('(?i)\bdon' + [char]0x2019 + 't\b[^,;]{0,50}$'))
                $suffixStart = $successMatch.Index + $successMatch.Length
                $suffix = $clause.Substring($suffixStart)
                $requiresConfirmation = $suffix -match '(?i)^\s+only after.{0,80}(?:final )?outcome.{0,30}(?:confirm|known|verif)'
                if (-not $negatesSuccess -and -not $requiresConfirmation) { $unsafeTimeoutSuccess = $true; break }
            }
        }
    }
    $logNegation = '(?:never|must not|do not|don\x27t|don' + [char]0x2019 + 't)'
    $secretsStayPrivate = ($Runbook -match "(?is)$logNegation.{0,50}(log|logging).{0,100}(secret|credential)|(secret|credential).{0,100}$logNegation.{0,50}(log|logging)") -or
        $Runbook -match '(?is)(?:secrets?|credentials?).{0,50}(?:stay|remain|are kept|kept)\s+out of logs?' -or
        $Runbook -match '(?is)(?:keep|store).{0,30}(?:secrets?|credentials?).{0,30}out of logs?'
    $payloadStaysPrivate = ($Runbook -match "(?is)$logNegation.{0,50}(log|logging).{0,100}payload|payload.{0,100}$logNegation.{0,50}(log|logging)") -or
        $Runbook -match '(?is)payloads?.{0,50}(?:stay|remain|are kept|kept)\s+out of logs?' -or
        $Runbook -match '(?is)(?:keep|store).{0,30}payloads?.{0,30}out of logs?'
    return $sameIDRetry -and -not $freshIDRetry -and $deduplicatedRetry -and -not $duplicateRisk -and $requestIDRetained -and $timeoutReporting -and -not $unsafeTimeoutSuccess -and
        $secretsStayPrivate -and $payloadStaysPrivate -and $Declared
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
        $fastChecks = Test-A7FastCheckAssertionContract $script:repoPath
        $oldGates = @('python scripts/check_operations.py', 'python -m unittest discover -s scripts', 'go test ./...', 'go -C tools/process-sentinel test ./...')
        $hook = Read-RepoText '.githooks/pre-commit'
        $ci = Read-RepoText '.github/workflows/ci.yaml'
        $allGates = Test-ActiveCheckGates $hook $ci @($oldGates + 'go vet ./...')
        $guidance = @('docs/engineering/agent-rules.md', 'AGENTS.md', '.claude/CLAUDE.md') | ForEach-Object { (Read-RepoText $_) -match '(?i)go vet \./\.\.\.' }
        $ciScalar = Test-ActiveCIRunCommand $ci 'go vet ./...'
        Add-Check 'original-a7-checker-assertions' ($managed -and $fastChecks) 'Managed-artifact/exclusion and four original fast-check expectations remain; one mapped vet check was added.'
        Add-Check 'a7-guidance-and-gates' ($allGates -and $ciScalar -and (@($guidance | Where-Object { $_ }).Count -eq 3)) 'Canonical and provider guidance mention vet; hook and exact CI scalar retain all prior gates plus vet.'
    }
    if ((Test-EffectiveTask 7) -and $arm -eq 'b') {
        $project = Read-RepoText 'markitect.yaml'
        $checks = @('managed-artifacts', 'hook-and-pipeline-contracts', 'root-tests', 'nested-module-tests') | ForEach-Object { $project -match '(?m)^\s+- name: ' + [regex]::Escape($_) + '\s*$' }
        $guidance = Test-B7VerificationAndGateContract $script:repoPath
        Add-Check 'original-b7-check-assertions' (@($checks | Where-Object { $_ }).Count -eq 4) 'All four configured Markitect checks remain enabled.'
        Add-Check 'b7-guidance-and-gates' $guidance 'Canonical guidance routes to explicit Project and module check owners; hook and CI retain root/nested tests plus vet.'
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
        Add-Check 'cumulative-vet-gates' (Test-ActiveCheckGates $hook $ci @('go vet ./...')) 'go vet ./... remains an active command in both pre-commit and CI.'
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
        $declared = Test-DeclaredRunbook 'docs/operations/runbooks/lease-renewal.md'
        Add-Check 'lease-renewal-runbook' (Test-LeaseRenewalRunbookContract $runbook $declared) 'Runbook explains same-ID duplicate prevention, ID retention, timeout reporting, private logs, and is declared as an input.'
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
    $taskScopes = Get-TaskScopes $card $arm
    $p01Scopes = @()
    $p02Scopes = @()
    if ($Task -eq '08' -and $PriorTasksThrough -eq '06') {
        $p01Scopes = Get-TaskScopes (Get-TaskCard 'P01') $arm
        $p02Scopes = Get-TaskScopes (Get-TaskCard 'P02') $arm
    }
    $scopes = Select-TaskScopes $Task $PriorTasksThrough $taskScopes $p01Scopes $p02Scopes
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
