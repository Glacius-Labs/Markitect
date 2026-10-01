$ErrorActionPreference = 'Stop'

$policyPath = Join-Path (Get-Location) 'docs/support/refund-policy.md'
if (-not (Test-Path -LiteralPath $policyPath -PathType Leaf)) {
    throw "Required policy file is missing: $policyPath"
}

$policy = Get-Content -LiteralPath $policyPath -Raw
$required = @(
    '30 days after delivery',
    'photograph of the damage',
    'support lead'
)
foreach ($phrase in $required) {
    if ($policy -notmatch [regex]::Escape($phrase)) {
        throw "Refund policy is missing required text: $phrase"
    }
}
