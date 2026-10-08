#requires -Version 7.2
param(
    [Parameter(Mandatory = $true)][string]$ServerPath,
    [Parameter(Mandatory = $true)][string]$MarkitectPath,
    [Parameter(Mandatory = $true)][string]$Repository,
    [Parameter(Mandatory = $true)][string]$Revision,
    [string]$OutputDirectory = (Join-Path $env:TEMP ('markitect-mcp-transport-' + [guid]::NewGuid().ToString('N')))
)

$ErrorActionPreference = 'Stop'
if (Test-Path -LiteralPath $OutputDirectory) { throw 'Choose a fresh output directory to preserve prior evidence.' }
New-Item -ItemType Directory -Path $OutputDirectory | Out-Null
$cli = (Resolve-Path -LiteralPath $MarkitectPath).Path
$repo = (Resolve-Path -LiteralPath $Repository).Path
$probe = Join-Path $PSScriptRoot 'probe-stdio.ps1'
$calls = @(
    @{ Name = 'find'; Args = @('find', '--repo', $repo, '--revision', $Revision, '--query', 'refund-triage', '--kind', 'Skill', '--namespace', 'support') },
    @{ Name = 'explain'; Args = @('explain', '--repo', $repo, '--revision', $Revision, '--kind', 'Skill', '--namespace', 'support', '--name', 'refund-triage') },
    @{ Name = 'context'; Args = @('context', '--repo', $repo, '--revision', $Revision, '--kind', 'Skill', '--namespace', 'support', '--name', 'refund-triage') }
)
$cliResults = @()
foreach ($call in $calls) {
    $watch = [System.Diagnostics.Stopwatch]::StartNew()
    $lines = & $cli @($call.Args) 2>&1
    $exitCode = $LASTEXITCODE
    $watch.Stop()
    if ($exitCode -ne 0) { throw "Published CLI $($call.Name) failed with exit code $exitCode`: $($lines -join "`n")" }
    # Native CLI writes one terminal newline; PowerShell strips it when collecting lines.
    $text = ($lines -join "`n") + "`n"
    $path = Join-Path $OutputDirectory "cli-$($call.Name).txt"
    Set-Content -LiteralPath $path -Value $text -Encoding utf8 -NoNewline
    $cliResults += [pscustomobject]@{ Name = $call.Name; Text = $text; ElapsedMilliseconds = $watch.ElapsedMilliseconds; Path = $path }
}

$rawMcp = Join-Path $OutputDirectory 'mcp-stdio.jsonl'
$mcp = & $probe -ServerPath $ServerPath -MarkitectPath $cli -Repository $repo -Revision $Revision -OutputPath $rawMcp | ConvertFrom-Json
if ($mcp.ExitCode -ne 0 -or $mcp.ServerStderr) { throw "MCP stdio server failed: $($mcp.ServerStderr)" }
$mcpResults = @($mcp.Responses | Where-Object { $_.Id -in @(3, 4, 5) })
if ($mcpResults.Count -ne 3 -or ($mcpResults | Where-Object IsToolError).Count -ne 0) { throw 'One or more valid MCP tool calls failed.' }
$comparisons = foreach ($index in 0..2) {
    $cliResult = $cliResults[$index]
    $mcpResult = $mcpResults[$index]
    $cliBytes = [System.Text.Encoding]::UTF8.GetBytes($cliResult.Text)
    $mcpBytes = [System.Text.Encoding]::UTF8.GetBytes($mcpResult.Text)
    [pscustomobject]@{
        Tool = $cliResult.Name
        ExactTextEqual = [System.String]::Equals($cliResult.Text, $mcpResult.Text, [System.StringComparison]::Ordinal)
        CliUtf8Bytes = $cliBytes.Length
        McpUtf8Bytes = $mcpBytes.Length
        CliSha256 = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($cliBytes)).ToLowerInvariant()
        McpSha256 = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($mcpBytes)).ToLowerInvariant()
        CliElapsedMilliseconds = $cliResult.ElapsedMilliseconds
        McpTextBytes = $mcpResult.TextBytes
    }
}
$summary = [pscustomobject]@{
    Revision = $Revision
    Repository = $repo
    ServerExitCode = $mcp.ExitCode
    ServerStderr = $mcp.ServerStderr
    RawMcpPath = $rawMcp
    CliOutputs = @($cliResults | Select-Object Name, Path, ElapsedMilliseconds)
    Comparisons = @($comparisons)
    Boundaries = @($mcp.Responses | Where-Object { $_.Id -ge 6 } | Select-Object Id, JsonRpcError, IsToolError, Text, TextBytes)
}
$summaryPath = Join-Path $OutputDirectory 'summary.json'
$summary | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $summaryPath -Encoding utf8
Get-Content -LiteralPath $summaryPath -Raw

$comparisonFailures = @($comparisons | Where-Object { -not $_.ExactTextEqual })
if ($comparisonFailures.Count) { throw 'CLI and MCP tool payloads differed; see preserved summary.' }
$unknownTool = @($mcp.Responses | Where-Object { $_.Id -eq 6 })
if ($unknownTool.Count -ne 1 -or $unknownTool[0].JsonRpcError.code -ne -32601) { throw 'Unknown tool did not produce the expected JSON-RPC rejection.' }
foreach ($id in 7..10) {
    $boundary = @($mcp.Responses | Where-Object { $_.Id -eq $id })
    if ($boundary.Count -ne 1 -or $boundary[0].IsToolError -ne $true -or $boundary[0].Text -notmatch '^unknown argument ') {
        throw "Boundary request $id was not rejected as expected; see preserved summary."
    }
}