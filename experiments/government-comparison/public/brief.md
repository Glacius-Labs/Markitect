# Common initial service brief

Version: 0.2. This text is supplied outside the actor repository with the currently released task card. It is not starter-repository content.

Build a small HTTP service in C# on .NET 10 with local SQLite. The initial Greenfield Git tree is empty. The study supplies an external brief, the initial task card, and public checks for that release; each actor creates and pins the toolchain and dependencies as part of its charged work. The Brownfield cell starts from its separate frozen source candidate and evidence packet. It is a useful existing service, not a production system.

## Initial HTTP and JSON contract

The service binds to the address and SQLite location supplied by the runner. JSON uses camelCase. These initial routes are required:

- `GET /health`: return HTTP 200 while the service can answer requests.
- `GET /catalog`: return a JSON array. Each item has `sku` and `unitPriceCents`; a display name may also be supplied. The required values are `WIDGET` at 1250 cents and `GADGET` at 775 cents.
- `POST /orders`: accept `{"items":[{"sku":"WIDGET","quantity":2}]}` and an `Idempotency-Key` header. A successful new order returns HTTP 201 and a JSON object containing `id`, `status`, `items`, and `totalCents`. Its initial status is `accepted`. Each returned item contains `sku`, `quantity`, and `unitPriceCents`. For two WIDGETs and one GADGET, `totalCents` is 3275.
- `GET /orders/{id}`: return the same order object for a known ID. An unknown ID returns HTTP 404 with `{"code":"not_found","message":"..."}`.

Error responses use `{"code":"...","message":"..."}`. For this initial order behavior, missing/empty items, missing or blank idempotency key, and invalid quantities return HTTP 400 with code `validation`; an unknown SKU returns HTTP 400 with code `unknown_sku`. Error messages must be non-empty and useful. The order quantity limit is stated by the currently released task card. Do not infer requirements from a future card.

A repeated idempotency key with the same canonical request returns HTTP 200 and the same order ID. Canonicalization combines duplicate SKU lines and sorts by SKU, so line order and duplicate-line spelling do not make an otherwise identical request different. Reusing the key for a different canonical request returns HTTP 409 with code `idempotency_conflict`, leaving the original result unchanged.

## Prices and freedom

All amounts are integer euro cents; do not use binary floating point for prices or totals. Quantities are positive integers. The required catalog SKUs and prices above are fixed. Keep the service runnable and provide concise build/run/check instructions.

The implementation may use minimal APIs or controllers, direct SQL or ADO.NET, and a single project or a small number of projects. No DDD, Clean Architecture, repository pattern, mediator, ORM, container, cloud service, or paid service is required. Choose abstractions proportionate to the behavior that has actually been released.
