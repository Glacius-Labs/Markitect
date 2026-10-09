
param([string]$Launcher)
$tokensForValidation = $null
$errorsForValidation = $null
[System.Management.Automation.Language.Parser]::ParseFile($Launcher,[ref]$tokensForValidation,[ref]$errorsForValidation) | Out-Null
[ordered]@{errors=@($errorsForValidation).Count;purpose='AST only; launcher not executed'} | ConvertTo-Json -Compress
if (@($errorsForValidation).Count -ne 0) { exit 1 }
