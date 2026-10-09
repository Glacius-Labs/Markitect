# Project-world Shop example

This fixture demonstrates project-owned Markitect files under `.markitect/`, a recursive Manager tree, vertical Sales/Orders/Inventory slices, explicit file ownership and realization, and a runnable cancellation example using Python's standard library and SQLite.

The fixture is intentionally small. It ships without a provider runner or credentials, and no agent starts automatically. The empty `.markitect/runtime.yaml` remains unchanged by the fixture tests. To configure a runner in a disposable copy, use `project setup` from the Markitect source candidate; it creates a reviewed edit proposal without hand-authored YAML and never probes account credentials.

See the repository guide at [docs/project-workflow.md](../../docs/project-workflow.md) for the project CLI lifecycle.

The repository example is nested inside Markitect, so copy it to an isolated directory before using Git-backed project commands. From the Markitect repository root, the following PowerShell creates a temporary feature-branch checkout and runs the fixture tests plus the native candidate CLI:

```powershell
$markitectRoot = (Resolve-Path .).Path
$fixtureRepo = Join-Path $env:TEMP ("markitect-project-world-" + [guid]::NewGuid().ToString("N"))
Copy-Item .\examples\project-world $fixtureRepo -Recurse
Push-Location $fixtureRepo
git init -b feature/project-world
git config core.autocrlf false
git config user.email project-world@example.test
git config user.name "Project World Fixture"
git add .
git commit -m "Initialize project-world fixture"
$modelBasis = (git rev-parse HEAD).Trim()
python -B -m unittest discover -s tests -v
Pop-Location

Push-Location $markitectRoot
go run ./cmd/markitect project check --repo $fixtureRepo
go run ./cmd/markitect project coverage --repo $fixtureRepo
go run ./cmd/markitect project index --repo $fixtureRepo
go run ./cmd/markitect project context --repo $fixtureRepo --manager '["project.markitect.example.org/v1alpha1","Manager","","shop"]'
go run ./cmd/markitect project context --repo $fixtureRepo --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]'
go run ./cmd/markitect project document --repo $fixtureRepo
Pop-Location
```

`coverage` classifies every repository file. The fixture has no broad ignore rules: its README, package modules, transaction coordinator, tests, cancellation guide, and protocol fixtures under `tests/agent-fixtures/` are mapped to their owning Managers through Artifacts. The generated project document is explicitly owned at `docs/markitect/project.md` and does not enter the semantic inventory. The empty runtime keeps conversational execution unconfigured.

The CLI also exposes an onboarding preview and briefing ledger inspection. The onboarding command only prints a reviewed file plan; applying one requires reviewing its digest and passing it back with `--expect PLAN_DIGEST --write`. Briefings are generated only when two committed model revisions and explicit provenance exist; this single-commit fixture intentionally does not fabricate briefing history.

```powershell
Push-Location $markitectRoot
go run ./cmd/markitect project onboard --repo $fixtureRepo --provider both
go run ./cmd/markitect project briefings --repo $fixtureRepo
Pop-Location
```

An empty briefing ledger is not evidence of prior accepted changes. The check, coverage, index, context, document, briefing-list, and onboarding-preview commands above do not invoke a provider. Cleanup and Reconcile likewise only print plan previews.

To prepare that disposable copy for a conversational run, first check local prerequisites. The doctor report deliberately reports authentication as `not-verified`; it never runs a login-status command or reads auth files. The setup preview requires the caller to select the model and explicit cost-estimation weights for their own budget policy:

```powershell
Push-Location $markitectRoot
go run ./cmd/markitect project doctor --repo $fixtureRepo --tool-root $markitectRoot --provider codex
go run ./cmd/markitect project setup --repo $fixtureRepo --tool-root $markitectRoot --provider codex --model MODEL --effort high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET
```

Review `editPlan.digest` and the exact native executable/adapter pins in the preview before applying that same deterministic runtime edit:

```powershell
go run ./cmd/markitect project setup --repo $fixtureRepo --tool-root $markitectRoot --provider codex --model MODEL --effort high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET --expect EDIT_PLAN_DIGEST --write
go run ./cmd/markitect project document --repo $fixtureRepo --write
Pop-Location
Push-Location $fixtureRepo
git add .markitect/runtime.yaml docs/markitect/project.md
git commit -m "Configure project-local Markitect runtime"
$verifiedRevision = (git rev-parse HEAD).Trim()
Pop-Location
Push-Location $markitectRoot
go run ./cmd/markitect project verify --repo $fixtureRepo --revision $verifiedRevision --write
go run ./cmd/markitect project cleanup --repo $fixtureRepo --goal "Improve Shop implementations while preserving the accepted model"
go run ./cmd/markitect project reconcile --repo $fixtureRepo --goal "Reconcile every Shop responsibility against its repository files"
go run ./cmd/markitect project plan --repo $fixtureRepo --goal "Cancel confirmed orders and release their reservation atomically" --since $modelBasis
Pop-Location
```

Replace the uppercase placeholders with the caller's selected model, microcurrency-per-million-token estimates, configured task budget, and digest from the immediately preceding preview. These prices are estimate inputs, not provider quotes or hard invoice ceilings. The native candidate command and the release CLI remain distinct; `go run` does not update an installed release or local tool pin. A live run still requires an existing supported host and account login, which Markitect does not verify.

After committing the intended model/runtime basis, use the captured model commit as `--since` while planning a change. `--since` supplies the comparison baseline; code edits still target the current committed `HEAD`. A plan created without `--since` plans from the current project model without a historical comparison.

The default fixture intentionally leaves the runtime empty. After the optional setup flow, full fixed-revision Verify invokes the configured Manager reviewers and runs declared checks against the chosen revision; it does not create an implementation run or establish that an agent authored code. Cleanup and Reconcile print plans for review. The candidate command above runs from the Markitect source checkout; it does not change the installed release or any local tool pin. To run only the Python tests from a copy that is already at a Git worktree root:

```sh
python3 -B -m unittest discover -s tests -v
```

The model and source layout is:

```text
.markitect/
  project.yaml
  runtime.yaml
  model/
    manager.yaml                         Shop root
    commerce/manager.yaml                Commerce
    commerce/sales/manager.yaml          Sales
    commerce/sales/orders/manager.yaml   Orders
    commerce/sales/inventory/manager.yaml Inventory
    engineering/manager.yaml             Engineering
    ... statements, artifacts, workflow, architecture, and check
src/shop/
  orders/                                order state
  inventory/                             reservation lifecycle
  commerce/cancellation.py               cross-slice transaction
tests/test_cancellation.py
docs/cancellation.md
```

The Manager tree is root `shop`, then `commerce` → `sales` → `orders` and `inventory`, plus the peer `engineering` Manager. The cancellation flow crosses Orders and Inventory: shipped orders are rejected; successful cancellation releases exactly one active reservation in the same transaction; repeating cancellation is idempotent; absent, pre-released, or duplicate reservations cause the operation to fail and the order status to roll back. Those cases are linked to the Inventory artifact and the declared SQLite checks in the model; they remain finite fixture evidence, not proof of broader correctness.

The model is a structural specification. Passing these tests establishes only the finite SQLite behaviors they exercise; it does not establish that the model captures every requirement or that an AI followed the workflow.

The cancellation tests cover idempotency, shipped-order rejection, rollback on release failure, and rollback when the reservation is missing, already released, or duplicated.

The request/response examples in `tests/agent-fixtures/` are protocol fixtures, not a live invocation or provider configuration. The repository guide shows how a supported agent host can use the CLI as a read/proposal boundary.
