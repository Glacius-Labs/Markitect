#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Cell,
    [Parameter(Mandatory)][ValidateSet('Start', 'Continue')][string]$Mode
)
$ErrorActionPreference = 'Stop'
$cellPath = (Resolve-Path -LiteralPath $Cell).Path
$spec = Get-Content -LiteralPath $cellPath -Raw | ConvertFrom-Json
if ($spec.execution_authorized -ne $true -or [string]::IsNullOrWhiteSpace($spec.grant_id)) {
    throw 'No concrete execution grant in the cell manifest. Preparation is not a trial grant.'
}
foreach ($key in @('repository', 'evidence_root', 'codex_home', 'codex_exe',
                   'native_sha256', 'config_sha256', 'initial_main_sha', 'deadline_utc')) {
    if ([string]::IsNullOrWhiteSpace([string]$spec.$key)) { throw "Missing cell field: $key" }
}
if ([int]$spec.max_outer_turns -le 0 -or [int]$spec.max_turn_seconds -le 0) {
    throw 'Positive finite outer-turn and wall-time limits are required.'
}
$repo = (Resolve-Path -LiteralPath $spec.repository).Path
$nativeExe = (Resolve-Path -LiteralPath $spec.codex_exe).Path
$cellHome = (Resolve-Path -LiteralPath $spec.codex_home).Path
$configPath = Join-Path $cellHome 'config.toml'
if ((Get-FileHash -LiteralPath $nativeExe -Algorithm SHA256).Hash -ne $spec.native_sha256) {
    throw 'Native executable changed.'
}
if ((Get-FileHash -LiteralPath $configPath -Algorithm SHA256).Hash -ne $spec.config_sha256) {
    throw 'Frozen cell configuration changed.'
}
foreach ($name in @('AGENTS.md', 'BACKLOG.md')) {
    if (-not (Test-Path -LiteralPath (Join-Path $repo $name) -PathType Leaf)) {
        throw "Missing project entrypoint: $name"
    }
}
$deadline = [DateTimeOffset]::Parse($spec.deadline_utc).ToUniversalTime()
$remainingSeconds = [math]::Floor(($deadline - [DateTimeOffset]::UtcNow).TotalSeconds)
$seconds = [int][math]::Min([int]$spec.max_turn_seconds, $remainingSeconds)
if ($seconds -le 0) { throw 'Cell deadline exhausted.' }
$evidenceRoot = [IO.Path]::GetFullPath($spec.evidence_root)
$repoPrefix = $repo.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if ($evidenceRoot.Equals($repo, [StringComparison]::OrdinalIgnoreCase) -or
    $evidenceRoot.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Dispatcher evidence must be outside the actor repository.'
}
$null = New-Item -ItemType Directory -Path $evidenceRoot -Force
# Exclusive lock: one native outer Actor per cell, no overlap or double dispatch.
$cellLock = [IO.File]::Open((Join-Path $evidenceRoot 'dispatch.lock'),
    [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
try {
    $turns = @(Get-ChildItem -LiteralPath $evidenceRoot -Directory -Filter 'turn-*' | Sort-Object Name)
    if ($turns.Count -ge [int]$spec.max_outer_turns) { throw 'Outer-turn budget exhausted.' }
    if ($Mode -eq 'Start' -and $turns.Count -ne 0) { throw 'Start requires a fresh evidence directory.' }
    if ($Mode -eq 'Continue' -and $turns.Count -eq 0) { throw 'Continue requires an earlier dispatch.' }
    if ($Mode -eq 'Continue') {
        $previousFinal = Join-Path $turns[-1].FullName 'final.json'
        if (Test-Path -LiteralPath $previousFinal) {
            $previous = Get-Content -LiteralPath $previousFinal -Raw | ConvertFrom-Json
            if ($previous.status -in @('completed', 'blocked')) {
                throw 'Terminal actor state; no automatic continuation.'
            }
        }
    }
    $mainBefore = & git -C $repo rev-parse --verify refs/heads/main
    if ($LASTEXITCODE -ne 0) { throw 'Isolated repository requires an existing main branch.' }
    if ($Mode -eq 'Start' -and $mainBefore -ne $spec.initial_main_sha) {
        throw 'Initial main differs from the frozen seed.'
    }
    $turnDir = Join-Path $evidenceRoot ('turn-{0:D4}' -f ($turns.Count + 1))
    $null = New-Item -ItemType Directory -Path $turnDir
    Copy-Item -LiteralPath $cellPath -Destination (Join-Path $turnDir 'cell.json')
    $prompt = if ($Mode -eq 'Start') {
        'Hier liegt das Backlog. Implementiere die Arbeit, halte die Projektregeln ein, prüfe die Ergebnisse und merge die fertigen Änderungen nach main.'
    } else {
        'Setze denselben Auftrag anhand von BACKLOG.md, PROGRESS.md und dem tatsächlichen Repositoryzustand fort.'
    }
    $prompt += "`nDiese Ausführung hat höchstens $seconds Sekunden. Sichere den Arbeitsstand rechtzeitig und liefere den Abschluss im vorgegebenen JSON-Format."
    [IO.File]::WriteAllText((Join-Path $turnDir 'prompt.txt'), $prompt)
    $schemaPath = Join-Path $PSScriptRoot 'final.schema.json'
    $finalPath = Join-Path $turnDir 'final.json'
    $nativeArgs = @('exec', '--model', 'gpt-6-luna', '-c', 'model_reasoning_effort="high"',
        '-c', 'approval_policy="never"', '--sandbox', 'workspace-write', '--json',
        '--output-schema', $schemaPath, '--output-last-message', $finalPath, '-')
    $request = [ordered]@{
        grant_id = $spec.grant_id; mode = $Mode; repository = $repo
        executable = $nativeExe; arguments = $nativeArgs
        executable_sha256 = $spec.native_sha256; config_sha256 = $spec.config_sha256
        schema_sha256 = (Get-FileHash -LiteralPath $schemaPath -Algorithm SHA256).Hash
        model_requested = 'gpt-6-luna'; reasoning_requested = 'high'
        effective_runtime_profile = 'unverified; preserve native receipts and resolve separately'
        codex_home = $cellHome; main_before = $mainBefore
        max_seconds = $seconds; started_utc = [DateTimeOffset]::UtcNow.ToString('o')
    }
    $request | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $turnDir 'request.json') -Encoding utf8
    $psi = [Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $nativeExe
    $psi.WorkingDirectory = $repo
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $psi.RedirectStandardInput = $true
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.StandardInputEncoding = [Text.UTF8Encoding]::new($false)
    $psi.Environment['CODEX_HOME'] = $cellHome
    foreach ($arg in $nativeArgs) { $psi.ArgumentList.Add($arg) }
    $proc = [Diagnostics.Process]::new()
    $proc.StartInfo = $psi
    $stdout = [IO.File]::Create((Join-Path $turnDir 'events.jsonl'))
    $stderr = [IO.File]::Create((Join-Path $turnDir 'stderr.txt'))
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $timedOut = $false
    $failure = $null
    try {
        if (-not $proc.Start()) { throw 'Native process did not start.' }
        $outTask = $proc.StandardOutput.BaseStream.CopyToAsync($stdout)
        $errTask = $proc.StandardError.BaseStream.CopyToAsync($stderr)
        $proc.StandardInput.Write($prompt)
        $proc.StandardInput.Close()
        if (-not $proc.WaitForExit([int]([math]::Min($seconds, 2147483) * 1000))) {
            $timedOut = $true
            $proc.Kill($true)
            $proc.WaitForExit()
        }
        $drain = [Threading.Tasks.Task]::WhenAll([Threading.Tasks.Task[]]@($outTask, $errTask))
        if (-not $drain.Wait(5000)) { throw 'Native streams did not close within five seconds.' }
        $nativeExit = $proc.ExitCode
    } catch {
        $failure = $_.Exception.Message
        $nativeExit = $null
        try { if (-not $proc.HasExited) { $proc.Kill($true); $proc.WaitForExit() } } catch { }
    } finally {
        $watch.Stop()
        $stdout.Dispose(); $stderr.Dispose(); $proc.Dispose()
    }
    $mainAfter = & git -C $repo rev-parse --verify refs/heads/main
    $gitExit = $LASTEXITCODE
    & git -C $repo status --porcelain=v1 --branch |
        Set-Content -LiteralPath (Join-Path $turnDir 'git-status.txt') -Encoding utf8
    $receipt = [ordered]@{
        finished_utc = [DateTimeOffset]::UtcNow.ToString('o')
        elapsed_wall_seconds = $watch.Elapsed.TotalSeconds
        native_exit_code = $nativeExit; timed_out = $timedOut; infrastructure_error = $failure
        main_after = $mainAfter; main_read_exit_code = $gitExit
        actor_final_present = (Test-Path -LiteralPath $finalPath)
        usage_source = 'raw events.jsonl; no inferred child totals or cost'
    }
    $receipt | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $turnDir 'receipt.json') -Encoding utf8
    Write-Output $turnDir
    if ($timedOut) { exit 124 }
    if ($null -ne $failure) { exit 70 }
    if ($nativeExit -ne 0) { exit $nativeExit }
    if (-not (Test-Path -LiteralPath $finalPath)) { exit 65 }
    $finalJson = Get-Content -LiteralPath $finalPath -Raw
    if (-not (Test-Json -Json $finalJson -SchemaFile $schemaPath -ErrorAction SilentlyContinue)) { exit 65 }
} finally {
    $cellLock.Dispose()
}
