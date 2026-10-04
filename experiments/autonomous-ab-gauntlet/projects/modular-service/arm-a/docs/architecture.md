# Service architecture

The service has three independently owned modules: Orders, Inventory, and Billing. `internal/core` provides stable shared primitives. `internal/contracts` contains consumer-facing interfaces and values used for cross-module calls. Modules may depend on Core and Contracts; they do not import one another. Composition belongs to `internal/app` and the command entrypoint.

Orders owns order lifecycle decisions and coordinates reservation and invoice requests through contracts. Inventory owns stock and reservations. Billing owns invoice records and idempotency. Module package tests cover their use cases; the architecture test enforces import direction. See [agent guidance](../AGENTS.md) for required feedback commands and escalation boundary.
