# Project-world Shop example

This fixture demonstrates project-owned Markitect files under `.markitect/`, a recursive Manager tree, vertical Sales/Orders/Inventory slices, explicit file ownership and realization, and a runnable cancellation example using Python's standard library and SQLite.

The fixture is intentionally small. It has no configured provider runner, agent credentials, or automatically starting runtime. The empty `.markitect/runtime.yaml` is a placeholder until a user or root Manager proposes a bounded runtime through the reviewed `project edit` flow; the business code and tests run independently.

See the repository guide at [docs/project-workflow.md](../../docs/project-workflow.md) for the project CLI lifecycle.

The repository example is nested inside Markitect, so copy it to an isolated directory before using Git-backed project commands. From the Markitect repository root, the following PowerShell creates a temporary feature-branch checkout and runs the fixture tests plus the native candidate CLI:

```powershell
$markitectRoot = (Resolve-Path .).Path
$fixtureRepo = Join-Path $env:TEMP ("markitect-project-world-" + [guid]::NewGuid().ToString("N"))
Copy-Item .\examples\project-world $fixtureRepo -Recurse
Push-Location $fixtureRepo
git init -b feature/project-world
git config core.autocrlf false
git add .
git commit -m "Initialize project-world fixture"
python -B -m unittest discover -s tests -v
Pop-Location

Push-Location $markitectRoot
go run ./cmd/markitect project check --repo $fixtureRepo
go run ./cmd/markitect project index --repo $fixtureRepo
go run ./cmd/markitect project context --repo $fixtureRepo --manager '["project.markitect.example.org/v1alpha1","Manager","","shop"]'
go run ./cmd/markitect project context --repo $fixtureRepo --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]'
go run ./cmd/markitect project document --repo $fixtureRepo
Pop-Location
```

The candidate command above runs from the Markitect source checkout; it does not change the installed release or any local tool pin. To run only the Python tests from a copy that is already at a Git worktree root:

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

The model is a structural specification. Passing these tests establishes only the finite SQLite behaviors they exercise; it does not establish that the model captures every requirement or that an AI followed the workflow.

The cancellation tests cover idempotency, shipped-order rejection, rollback on release failure, and rollback when the reservation is missing, already released, or duplicated.

The request/response example in `.markitect/agent-fixtures/` is a protocol fixture, not a live invocation or provider configuration. The repository guide shows how a supported agent host can use the CLI as a read/proposal boundary.
