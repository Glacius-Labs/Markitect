# Brownfield fixture provenance

This is a purpose-built study fixture authored for the Markitect Government comparison. It is not copied from a production or customer repository and does not claim to represent a real legacy service. Its source begins in this study package; the enclosing Markitect transfer snapshot was `b9e17c9896f14b4b1268e4bcb005395f5d324384` when fixture work began.

The fixture gives each actor a functioning order-creation path and one visible pre-existing route, `POST /legacy/orders`, whose behavior must remain understandable while the new order, reservation, persistence, and lifecycle tasks are completed. No canonical model for that path is supplied. The initial behavior below is the known contract, not hidden task information.

## Toolchain and dependencies

- Target framework: `net10.0`.
- SDK: `10.0.103`, pinned by the fixture's `global.json`; the host has SDK `10.0.103` and .NET/ASP.NET Core runtime `10.0.3` installed.
- Direct packages: `Microsoft.Data.Sqlite` `10.0.3`, `SQLitePCLRaw.bundle_e_sqlite3` `3.0.5`, and `SQLitePCLRaw.core` `3.0.5`, each pinned exactly in the project and `packages.lock.json`.
- The lock resolves the native SQLite package `SQLite` `3.53.4`; the engine returned `3.53.4` from `SELECT sqlite_version()` using the fixture's resolved Microsoft.Data.Sqlite/SQLitePCLRaw assemblies.
- `NuGet.Config` clears machine sources and selects NuGet.org, so actors restore the same public dependency source. Locked restore, including the native package graph, succeeds on NuGet.org. The host's default Azure Artifacts feed returned HTTP 401 before this fixture-local config was added.
- The exact SQLitePCLRaw 3.x override avoids the vulnerable 2.1.11 native package selected transitively by Microsoft.Data.Sqlite 10.0.3; the final locked restore and build reported no package vulnerability warning.

## Known behavior at the initial snapshot

- `GET /health` returns `{ "status": "ok" }`.
- `GET /catalog` returns the case-sensitive SKUs and integer-cent prices: `WIDGET` at 1250 cents and `GADGET` at 775 cents.
- `GET /inventory/{sku}` returns `sku`, `onHand`, `reserved: 0`, and `available` equal to on-hand stock. Unknown SKUs return 404 with `{ "code", "message" }`.
- `POST /orders` and the existing `POST /legacy/orders` use the same behavior. A request has `items`, each with a case-sensitive SKU and positive integer quantity, and requires a nonblank `Idempotency-Key` header. Empty orders, invalid quantities, unknown SKUs, or a total combined quantity above 10 are rejected with HTTP 400 and an `{ "code", "message" }` error.
- Duplicate lines are combined. Item order does not affect idempotency comparison. A first accepted key returns 201; repeating the key with the same semantic request returns the same order and ID with 200; using that key for different content returns 409. Accepted orders have an opaque unique string ID, normalized item lines, an exact integer-cent total, and status `accepted`.
- `GET /orders/{id}` reads accepted orders; an unknown order returns 404 with code `not_found`. The catalog, order, and idempotency records are stored in local SQLite. `ORDERS_DB` selects the database path; otherwise the app uses `orders.db` in its working directory. `ASPNETCORE_URLS` controls the listener.

## Explicit starting gaps

The initial fixture accepts an order without reserving stock. It does not reject orders based on currently available inventory, provide reservation concurrency guarantees, implement cancellation/fulfillment, or reconcile a canonical model. Those are later task concerns. The alias route is deliberately useful existing behavior whose compatibility should be checked throughout the tasks. Do not infer that stock is reserved from the inventory response's zero value.

The source and provenance establish only the fixture's authored contract and observed checks. They are not evidence of real-world migration history, customer use, or production operational characteristics.

## Known ownership and input coverage

Scientist is the author/owner of all reference fixture sources and their public
evidence packet; this is provenance, not a supplied canonical project model.

| Source | Observed responsibility |
|---|---|
| `src/Orders.Api/Program.cs` | HTTP wiring, both order entry points, status/error mapping |
| `src/Orders.Api/OrdersStore.cs` | SQLite schema/catalog, validation, exact totals, order and idempotency persistence |
| `src/Orders.Api/Contracts.cs` | Existing HTTP request/response record shapes |
| `global.json`, project, lock, `NuGet.Config` | Toolchain selection, explicit dependency graph and restore source |
| `.gitignore`, settings, launch profile, `README.md` | Local development and run instructions |

Vendor package bytes remain external inputs identified by the lock hashes.
There are no supplied Markitect Definitions, area/mandate assignments or projection
ownership records for any source. The legacy entry point is visibly present and
unmodeled; the actor must discover/account for it and preserve its valid behavior.
No historical database is shipped. Checks use new isolated test databases and
separately exercise restart durability; no undeclared production migration corpus
is part of this case.
