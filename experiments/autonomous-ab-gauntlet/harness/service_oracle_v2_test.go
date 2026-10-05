package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceOracleV2OwnershipAndIntegrationScopes(t *testing.T) {
	pwsh := requirePowerShell(t)
	evaluator, err := filepath.Abs(filepath.Join("..", "projects", "modular-service", "oracle", "evaluate.ps1"))
	if err != nil {
		t.Fatal(err)
	}

	const script = `
param([string]$EvaluatorPath)
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($EvaluatorPath, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) { throw "Evaluator parse failed: $($parseErrors[0].Message)" }
foreach ($name in @('Has-CompleteOwnershipTable','Has-MetadataName','Has-Relation','Has-AvailabilityIntent','Get-CardScopes','Test-ParallelIntegration','Get-EffectiveCardScopes')) {
    $function = $ast.Find({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name }, $true)
    if (-not $function) { throw "Required function was not found: $name" }
    Invoke-Expression $function.Extent.Text
}
$script:oracleRoot = Split-Path -Parent $EvaluatorPath
$tick = [string][char]96
$candidate = @'
| Module | Owning package | Owns | Allowed dependencies |
| --- | --- | --- | --- |
| Orders | @TICK@internal/modules/orders@TICK@ | Order lifecycle decisions; coordinates reservation and invoice requests through contracts. | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
| Inventory | @TICK@internal/modules/inventory@TICK@ | Stock and reservations. | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
| Billing | @TICK@internal/modules/billing@TICK@ | Invoice records and idempotency. | Core (@TICK@internal/core@TICK@), Contracts (@TICK@internal/contracts@TICK@) |
'@
$candidate = $candidate.Replace('@TICK@', [string][char]96)
$conjoinedCoordination = $candidate.Replace('Order lifecycle decisions; coordinates reservation and invoice requests through contracts.', 'Owns order lifecycle and coordinates reservation and invoice requests through contracts.')
$caseAndPunctuationVariant = $candidate.Replace('Order lifecycle decisions; coordinates reservation and invoice requests through contracts.', 'ORDER lifecycle decisions; coordinates reservation and invoice requests through contracts')
$formattedOwnershipVariant = $candidate.Replace('Order lifecycle decisions; coordinates reservation and invoice requests through contracts.', '**Order lifecycle decisions**; *coordinates reservation and invoice requests through contracts*.')
$wrongOwner = $candidate.Replace('Order lifecycle decisions;', 'Stock and reservations;')
$contradictoryOwner = $candidate.Replace('Order lifecycle decisions;', 'Order lifecycle decisions; Orders also owns stock and reservations;')
$unrequestedCoordination = $candidate.Replace('coordinates reservation and invoice requests through contracts.', 'coordinates reservation and invoice requests by directly importing Inventory.')
$contractCell = 'Contracts (' + $tick + 'internal/contracts' + $tick + ') |'
$extraDependency = $candidate.Replace($contractCell, 'Contracts (' + $tick + 'internal/contracts' + $tick + '), Payments (' + $tick + 'internal/payments' + $tick + ') |')
$coreDependency = 'Core (' + $tick + 'internal/core' + $tick + ')'
$wrongDependencyPath = $candidate.Replace($coreDependency, 'Core (' + $tick + 'internal/contracts' + $tick + ')')
$namedDependencies = $coreDependency + ', Contracts (' + $tick + 'internal/contracts' + $tick + ')'
$pathDependencies = $candidate.Replace($namedDependencies, 'internal/core, internal/contracts')
$siblingDependency = $candidate.Replace($namedDependencies, 'internal/core, internal/modules/orders')
$nestedCoreDependency = $candidate.Replace($namedDependencies, 'internal/core/private, internal/contracts')
$ordersPackage = $tick + 'internal/modules/orders' + $tick
$inventoryPackage = $tick + 'internal/modules/inventory' + $tick
$wrongPackage = $candidate.Replace($ordersPackage, $inventoryPackage)
$cases = @(
    @{ name = 'captured contract-coordination wording'; text = $candidate; expected = $true },
    @{ name = 'equivalent coordinated ownership with conjunction'; text = $conjoinedCoordination; expected = $true },
    @{ name = 'case and terminal punctuation variation'; text = $caseAndPunctuationVariant; expected = $true },
    @{ name = 'bold and italic ownership prose'; text = $formattedOwnershipVariant; expected = $true },
    @{ name = 'wrong Orders owner'; text = $wrongOwner; expected = $false },
    @{ name = 'correct prefix followed by contradictory ownership'; text = $contradictoryOwner; expected = $false },
    @{ name = 'coordination contradicts module boundary'; text = $unrequestedCoordination; expected = $false },
    @{ name = 'extra dependency'; text = $extraDependency; expected = $false },
    @{ name = 'Core dependency with Contracts path'; text = $wrongDependencyPath; expected = $false },
    @{ name = 'exact Core and Contracts paths'; text = $pathDependencies; expected = $true },
    @{ name = 'sibling module dependency'; text = $siblingDependency; expected = $false },
    @{ name = 'nested unsafe Core path'; text = $nestedCoreDependency; expected = $false },
    @{ name = 'wrong package'; text = $wrongPackage; expected = $false }
)
foreach ($case in $cases) {
    $actual = Has-CompleteOwnershipTable $case.text
    if ($actual -ne $case.expected) { throw "Ownership case '$($case.name)' returned $actual; expected $($case.expected)" }
}

$availability = @(
    'apiVersion: markitect.example.org/v1alpha1',
    'kind: UseCase',
    'metadata:',
    '  name: get-availability',
    'spec:',
    '  module:',
    '    kind: Module',
    '    name: inventory',
    '    namespace: architecture'
) -join [char]10
$wrongAvailabilityName = $availability.Replace('name: get-availability', 'name: list-availability')
$missingAvailabilityOwner = @('apiVersion: markitect.example.org/v1alpha1', 'kind: UseCase', 'metadata:', '  name: get-availability', 'spec:', '  summary: Inventory availability query') -join [char]10
$wrongAvailabilityOwner = $availability.Replace('name: inventory', 'name: billing')
foreach ($case in @(
    @{ name = 'canonical availability identity with typed Inventory owner'; text = $availability; expected = $true },
    @{ name = 'availability identity without required typed owner'; text = $missingAvailabilityOwner; expected = $false },
    @{ name = 'wrong canonical availability identity'; text = $wrongAvailabilityName; expected = $false },
    @{ name = 'availability assigned to wrong module'; text = $wrongAvailabilityOwner; expected = $false }
)) {
    $actual = Has-AvailabilityIntent $case.text
    if ($actual -ne $case.expected) { throw "Availability case '$($case.name)' returned $actual; expected $($case.expected)" }
}

$ordinary08 = Test-ParallelIntegration '08' '06' $false
$integration08 = Test-ParallelIntegration '08' '06' $true
$differentPrior = Test-ParallelIntegration '08' '05' $true
if ($ordinary08 -or -not $integration08 -or $differentPrior) { throw 'Parallel integration recognition did not require explicit task 08 / prior 06 signal' }
$task08Scopes = @(Get-EffectiveCardScopes '08' 'a' $false $ordinary08)
$integrationScopes = @(Get-EffectiveCardScopes '08' 'a' $false $integration08)
$p01Scopes = @(Get-EffectiveCardScopes 'P01' 'a' $true $false)
$p02Scopes = @(Get-EffectiveCardScopes 'P02' 'a' $true $false)
if (($task08Scopes -join ',') -ne 'internal/contracts/billing/,internal/modules/billing/') { throw "Ordinary task 08 scope changed: $($task08Scopes -join ',')" }
if (($integrationScopes -join ',') -ne 'internal/modules/orders/,internal/contracts/billing/,internal/modules/billing/') { throw "Integration scope is not the exact P01/P02 union: $($integrationScopes -join ',')" }
if (($p01Scopes -join ',') -ne 'internal/modules/orders/') { throw "P01 scope was widened: $($p01Scopes -join ',')" }
if (($p02Scopes -join ',') -ne 'internal/contracts/billing/,internal/modules/billing/') { throw "P02 scope was widened: $($p02Scopes -join ',')" }
Write-Output "Passed $($cases.Count) ownership cases, 4 availability-intent cases, and exact parallel-scope cases."
`
	scriptPath := filepath.Join(t.TempDir(), "service-oracle-v2.ps1")
	if err := os.WriteFile(scriptPath, []byte(strings.TrimSpace(script)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pwsh, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", scriptPath, evaluator)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("service oracle contract failed: %v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}

func TestTask01VectorChecksRejectionAndStateNotErrorWording(t *testing.T) {
	vectorPath := filepath.Join("..", "projects", "modular-service", "oracle", "vectors", "task01_test.go.txt")
	vector, err := os.ReadFile(vectorPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(vector), "strings.Contains") {
		t.Fatal("task 01 vector still constrains the error message with a literal substring")
	}

	for _, test := range []struct {
		name   string
		mode   string
		passed bool
	}{
		{name: "generic error rejection with unchanged stock", mode: "reject", passed: true},
		{name: "same-key quantity change accepted", mode: "accept", passed: false},
		{name: "stock mutated while rejecting", mode: "mutate", passed: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := writeTask01VectorFixture(t, string(vector), test.mode)
			cmd := exec.Command("go", "test", "./internal/modules/inventory", "-run", "^TestGauntletIdempotencyVector$", "-count=1")
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			passed := err == nil
			if passed != test.passed {
				t.Fatalf("vector pass=%v, want %v: %v\n%s", passed, test.passed, err, output)
			}
		})
	}
}

func writeTask01VectorFixture(t *testing.T, vector, behavior string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/acme/modular-service\n\ngo 1.22\n",
		"internal/contracts/inventory/reservation.go": `package inventory

type ReserveRequest struct { OrderID, SKU string; Quantity int }
type Reservation struct { ID, OrderID, SKU string; Quantity int }
`,
		"internal/modules/inventory/service.go": fmt.Sprintf(`package inventory

import (
	"context"
	"errors"
	"fmt"
	contract "example.com/acme/modular-service/internal/contracts/inventory"
)

const testBehavior = %q

type Service struct { stock map[string]int; reservations map[string]contract.Reservation; nextID int }
func New(stock map[string]int) *Service {
	copy := make(map[string]int, len(stock))
	for sku, count := range stock { copy[sku] = count }
	return &Service{stock: copy, reservations: make(map[string]contract.Reservation)}
}
func (s *Service) Available(sku string) int { return s.stock[sku] }
func (s *Service) Reserve(_ context.Context, request contract.ReserveRequest) (contract.Reservation, error) {
	key := request.OrderID + "\x00" + request.SKU
	if reservation, ok := s.reservations[key]; ok {
		if reservation.Quantity != request.Quantity {
			switch testBehavior {
			case "accept": return contract.Reservation{}, nil
			case "mutate": s.stock[request.SKU] -= request.Quantity; return contract.Reservation{}, errors.New("key reused with another quantity")
			default: return contract.Reservation{}, errors.New("reservation key reused with a different quantity")
			}
		}
		return reservation, nil
	}
	if s.stock[request.SKU] < request.Quantity { return contract.Reservation{}, errors.New("insufficient stock") }
	s.stock[request.SKU] -= request.Quantity
	s.nextID++
	reservation := contract.Reservation{ID: fmt.Sprintf("reservation-%%d", s.nextID), OrderID: request.OrderID, SKU: request.SKU, Quantity: request.Quantity}
	s.reservations[key] = reservation
	return reservation, nil
}
`, behavior),
		"internal/modules/inventory/gauntlet_hidden_test.go": vector,
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func requirePowerShell(t *testing.T) string {
	t.Helper()
	pwsh, err := exec.LookPath("pwsh")
	if err == nil {
		return pwsh
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Fatal("CI must execute the service oracle contract; pwsh is unavailable")
	}
	t.Skip("pwsh is not installed")
	return ""
}
