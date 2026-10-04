# Commerce engineering guidance

This is a fictional .NET 8 modular monolith. `src/Commerce` is split into Core and Orders, Inventory, and Billing modules. Before changing code, read `docs/architecture.md` and inspect the owning slice. Run the acceptance check after the requested change.

## Architecture

- Domain entities and invariants live under `<Module>/Domain`; requests, handlers, validators, and queries live in focused `<Module>/Application/<UseCase>` slices.
- A module owns its Domain, persistence boundary, and application behavior. Cross-module collaboration uses public `Contracts` only. Do not access another module's `Application`, `Domain`, `Infrastructure`, storage, or tables.
- A command is validated in its own Application slice. Keep aggregate invariants in the owning aggregate even when a validator provides earlier feedback.
- Core may contain stable abstractions used by modules. Product behavior and policy stay in the owning module unless an owner decides otherwise.
- Do not silently change module responsibility, waive an architectural rule, or implement a disputed boundary. Identify the owner and affected contracts, then ask for a decision.
- Preserve behavior unless the task asks for a behavior change. Do not claim database, HTTP, concurrency, or deployment behavior from this in-memory seed.

## Acceptance check

Run from this directory:

```powershell
dotnet run --project checks/Commerce.Checks.csproj
```

The runner covers declared behavior and source-boundary checks. It is deliberately offline and uses no external NuGet packages. Report exactly which checks ran and any gap.

