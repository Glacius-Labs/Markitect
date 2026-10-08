#requires -Version 7.2
param(
    [Parameter(Mandatory = $true)][string]$ServerPath,
    [Parameter(Mandatory = $true)][string]$MarkitectPath,
    [Parameter(Mandatory = $true)][string]$Repository,
    [Parameter(Mandatory = $true)][string]$Revision,
    [string]$OutputPath = (Join-Path $env:TEMP 'markitect-mcp-stdio-probe.jsonl')
)

$ErrorActionPreference = 'Stop'
$psi = [System.Diagnostics.ProcessStartInfo]::new()
$psi.FileName = (Resolve-Path -LiteralPath $ServerPath).Path
$psi.UseShellExecute = $false
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
foreach ($arg in @('--markitect', (Resolve-Path -LiteralPath $MarkitectPath).Path, '--repo', (Resolve-Path -LiteralPath $Repository).Path, '--revision', $Revision)) {
    $psi.ArgumentList.Add($arg)
}

$requests = @(
    @{ jsonrpc = '2.0'; id = 1; method = 'initialize'; params = @{ protocolVersion = '2025-11-25'; capabilities = @{}; clientInfo = @{ name = 'markitect-mcp-evaluation'; version = '1' } } },
    @{ jsonrpc = '2.0'; method = 'notifications/initialized' },
    @{ jsonrpc = '2.0'; id = 2; method = 'tools/list'; params = @{} },
    @{ jsonrpc = '2.0'; id = 3; method = 'tools/call'; params = @{ name = 'find'; arguments = @{ query = 'refund-triage'; kind = 'Skill'; namespace = 'support' } } },
    @{ jsonrpc = '2.0'; id = 4; method = 'tools/call'; params = @{ name = 'explain'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage' } } },
    @{ jsonrpc = '2.0'; id = 5; method = 'tools/call'; params = @{ name = 'context'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage' } } },
    @{ jsonrpc = '2.0'; id = 6; method = 'tools/call'; params = @{ name = 'unknown'; arguments = @{} } },
    @{ jsonrpc = '2.0'; id = 7; method = 'tools/call'; params = @{ name = 'context'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage'; path = '../docs/support/refund-triage.yaml' } } },
    @{ jsonrpc = '2.0'; id = 8; method = 'tools/call'; params = @{ name = 'context'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage'; package = 'parcel-support' } } },
    @{ jsonrpc = '2.0'; id = 9; method = 'tools/call'; params = @{ name = 'context'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage'; repo = 'C:\invalid\repository' } } },
    @{ jsonrpc = '2.0'; id = 10; method = 'tools/call'; params = @{ name = 'context'; arguments = @{ kind = 'Skill'; namespace = 'support'; name = 'refund-triage'; revision = '0000000000000000000000000000000000000000' } } }
)

$process = [System.Diagnostics.Process]::new()
$process.StartInfo = $psi
if (-not $process.Start()) { throw 'Could not start the MCP stdio server.' }
$stdoutTask = $process.StandardOutput.ReadToEndAsync()
$stderrTask = $process.StandardError.ReadToEndAsync()
foreach ($request in $requests) {
    $process.StandardInput.WriteLine(($request | ConvertTo-Json -Compress -Depth 12))
}
$process.StandardInput.Close()
if (-not $process.WaitForExit(120000)) {
    $process.Kill($true)
    $process.WaitForExit()
    $stdoutTask.Result | Set-Content -LiteralPath $OutputPath -Encoding utf8
    throw 'MCP stdio probe exceeded its 120-second bound.'
}
$stdout = $stdoutTask.Result
$stderr = $stderrTask.Result
Set-Content -LiteralPath $OutputPath -Value $stdout -Encoding utf8
$responses = @($stdout -split "`r?`n" | Where-Object { $_ } | ForEach-Object { $_ | ConvertFrom-Json })
$summary = foreach ($response in $responses) {
    $id = $response.id
    $resultText = ($response.result.content | Where-Object { $_.type -eq 'text' } | ForEach-Object { $_.text }) -join "`n"
    [pscustomobject]@{
        Id = $id
        JsonRpcError = $response.error
        IsToolError = $response.result.isError
        Text = $resultText
        TextBytes = [System.Text.Encoding]::UTF8.GetByteCount($resultText)
    }
}
[pscustomobject]@{
    ExitCode = $process.ExitCode
    ServerStderr = $stderr
    OutputPath = $OutputPath
    Responses = @($summary)
} | ConvertTo-Json -Depth 12
