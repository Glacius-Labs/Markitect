# Modular service engineering guidance

## Architecture

- `internal/core` owns stable primitives shared by modules.
- `internal/contracts` owns consumer-facing cross-module interfaces and DTOs.
- Each package under `internal/modules/<name>` owns its data and use cases.
- A module may import `core` and `contracts`, but must not import another
  module. Coordinate through a contract and keep composition in `internal/app`.
- Keep application behavior in explicit use cases. Domain decisions belong to
  the owning module; transport and wiring belong to `internal/app` or `cmd`.
- Preserve compatibility of exported contract types unless the task asks for a
  contract change. Update every consumer and test when a contract changes.

## Checks

Run `go test ./...` for the full local feedback loop. The architecture boundary
is independently enforced by `go test ./internal/architecture`; CI runs both.
Document behavior and ownership changes in the owning package README.

## Scope and decisions

Do not bypass contracts by importing a sibling module or reaching into its
storage. Keep changes inside the requested module and its declared contract
consumers. If a request requires a new cross-module dependency, changes who
owns business policy, conflicts with these rules, or leaves the owner unclear,
stop and ask the service owner to decide before changing code. Do not treat a
green test as approval to change architecture.
