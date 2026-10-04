package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestModularServiceOwnershipTableContract(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("CI must execute the PowerShell oracle contract; pwsh is unavailable")
		}
		t.Skip("pwsh is not installed")
	}

	evaluator, err := filepath.Abs(filepath.Join("..", "projects", "modular-service", "oracle", "evaluate.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(evaluator); err != nil {
		t.Fatal(err)
	}

	const script = `
param([string]$EvaluatorPath)
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($EvaluatorPath, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) { throw "Evaluator parse failed: $($parseErrors[0].Message)" }
$function = $ast.Find({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Has-CompleteOwnershipTable' }, $true)
if (-not $function) { throw 'Has-CompleteOwnershipTable was not found' }
Invoke-Expression $function.Extent.Text
$tick = [string][char]96

$candidate = @'
| Module | Owns | Package path | Allowed dependencies |
| --- | --- | --- | --- |
| Orders | Order lifecycle | @TICK@internal/modules/orders@TICK@ | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
| Inventory | Stock and reservations | @TICK@internal/modules/inventory@TICK@ | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
| Billing | Invoice records and idempotency | @TICK@internal/modules/billing@TICK@ | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
'@
$candidate = $candidate.Replace('@TICK@', [string][char]96)
$reordered = @'
| Allowed dependencies | Module | Package path | Owns |
| --- | --- | --- | --- |
| Contracts (@TICK@internal/contracts@TICK@) and Core (@TICK@internal/core@TICK@) | Billing | @TICK@internal/modules/billing@TICK@ | Owns invoice records and idempotency |
| Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) | Orders | @TICK@internal/modules/orders@TICK@ | Owns order lifecycle decisions |
| Core (@TICK@internal/core@TICK@); Contracts (@TICK@internal/contracts@TICK@) | Inventory | @TICK@internal/modules/inventory@TICK@ | Owns stock and reservations |
'@
$reordered = $reordered.Replace('@TICK@', [string][char]96)
$legacyLabels = @'
| Module | Responsibility | Allowed dependencies | Go package |
| --- | --- | --- | --- |
| Orders | Order lifecycle | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) | @TICK@internal/modules/orders@TICK@ |
| Inventory | Stock and reservations | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) | @TICK@internal/modules/inventory@TICK@ |
| Billing | Invoice records and idempotency | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) | @TICK@internal/modules/billing@TICK@ |
'@
$legacyLabels = $legacyLabels.Replace('@TICK@', [string][char]96)
$bareDependencies = $candidate.Replace('Core (' + $tick + 'internal/core' + $tick + '), Contracts (' + $tick + 'internal/contracts' + $tick + ')', 'Core, Contracts')
$mixedDependencies = $candidate.Replace('Core (' + $tick + 'internal/core' + $tick + '), Contracts (' + $tick + 'internal/contracts' + $tick + ')', 'Core, Contracts (' + $tick + 'internal/contracts' + $tick + ')')
$unlabeledWithNotes = @'
| Field A | Field B | Field C | Field D | Notes |
| --- | --- | --- | --- | --- |
| Orders | Order lifecycle owner | @TICK@internal/modules/orders@TICK@ | Core, Contracts | reviewed |
| Inventory | Stock and reservations | Core (@TICK@internal/core@TICK@), Contracts | @TICK@internal/modules/inventory@TICK@ | checked |
| Billing | Core, Contracts | Invoice records and idempotency | @TICK@internal/modules/billing@TICK@ | concise |
'@
$unlabeledWithNotes = $unlabeledWithNotes.Replace('@TICK@', [string][char]96)
$orderDecisionsWording = $candidate.Replace('Order lifecycle |', 'Owns order lifecycle and order decisions |')
$inventoryOwnerWording = $candidate.Replace('Stock and reservations |', 'Stock and reservations owner |')
$billingOwnerWording = $candidate.Replace('Invoice records and idempotency |', 'Invoice records and idempotency owner |')
$wrongOwner = $candidate.Replace('| Orders | Order lifecycle |', '| Orders | Stock and reservations |')
$missingRow = $candidate -replace '(?m)^\| Inventory \|.*\r?\n', ''
$contractsDependency = 'Contracts (' + $tick + 'internal/contracts' + $tick + ')'
$extraDependency = $candidate.Replace($contractsDependency, $contractsDependency + ', Payments (' + $tick + 'internal/payments' + $tick + ')')
$wrongPackage = $candidate.Replace($tick + 'internal/modules/billing' + $tick, $tick + 'internal/modules/inventory' + $tick)
$wrongCasePackage = $candidate.Replace($tick + 'internal/modules/orders' + $tick, $tick + 'internal/modules/Orders' + $tick)
$wrongDependencyPath = $candidate.Replace('Core (' + $tick + 'internal/core' + $tick + ')', 'Core (' + $tick + 'internal/shared' + $tick + ')')
$wrongCaseDependencyPath = $candidate.Replace('Core (' + $tick + 'internal/core' + $tick + ')', 'Core (' + $tick + 'internal/Core' + $tick + ')')
$missingDependency = $candidate.Replace(', Contracts (' + $tick + 'internal/contracts' + $tick + ')', '')
$cases = @(
    @{ name = 'task requested headings and bare responsibility nouns'; text = $candidate; expected = $true },
    @{ name = 'field values in another order'; text = $reordered; expected = $true },
    @{ name = 'recognized equivalent headings'; text = $legacyLabels; expected = $true },
    @{ name = 'bare dependency names'; text = $bareDependencies; expected = $true },
    @{ name = 'optional dependency path annotations'; text = $mixedDependencies; expected = $true },
    @{ name = 'unlabeled values and harmless fifth column'; text = $unlabeledWithNotes; expected = $true },
    @{ name = 'legacy order lifecycle owner wording'; text = $candidate.Replace('Order lifecycle |', 'Order lifecycle owner |'); expected = $true },
    @{ name = 'legacy order decisions wording'; text = $orderDecisionsWording; expected = $true },
    @{ name = 'legacy inventory owner wording'; text = $inventoryOwnerWording; expected = $true },
    @{ name = 'legacy billing owner wording'; text = $billingOwnerWording; expected = $true },
    @{ name = 'wrong owner'; text = $wrongOwner; expected = $false },
    @{ name = 'missing module'; text = $missingRow; expected = $false },
    @{ name = 'extra dependency'; text = $extraDependency; expected = $false },
    @{ name = 'missing dependency'; text = $missingDependency; expected = $false },
    @{ name = 'wrong package'; text = $wrongPackage; expected = $false },
    @{ name = 'wrong package casing'; text = $wrongCasePackage; expected = $false },
    @{ name = 'wrong dependency package'; text = $wrongDependencyPath; expected = $false },
    @{ name = 'wrong dependency casing'; text = $wrongCaseDependencyPath; expected = $false }
)
foreach ($case in $cases) {
    $actual = Has-CompleteOwnershipTable $case.text
    if ($actual -ne $case.expected) { throw "Case '$($case.name)' returned $actual; expected $($case.expected)" }
}
Write-Output "Passed $($cases.Count) ownership-table contract cases."
`
	scriptPath := filepath.Join(t.TempDir(), "ownership-contract.ps1")
	if err := os.WriteFile(scriptPath, []byte(strings.TrimSpace(script)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pwsh, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", scriptPath, evaluator)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PowerShell ownership contract failed: %v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}
