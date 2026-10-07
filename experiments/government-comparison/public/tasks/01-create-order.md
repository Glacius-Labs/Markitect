# Task 1 — Create a validated order

Implement the initial brief's `GET /health`, `GET /catalog`, `POST /orders`, and `GET /orders/{id}` routes. The maximum sum of line quantities is 10 inclusive. Duplicate SKU lines are combined and requests are canonicalized by SKU.

An empty/missing item list, a missing or blank `Idempotency-Key`, or an invalid/non-positive/non-integer quantity returns HTTP 400 with error code `validation`. An unknown SKU returns HTTP 400 with code `unknown_sku`. A quantity sum above 10 returns HTTP 400 with code `order_limit`. Errors have a non-empty `message`.

A new valid order returns HTTP 201 with status `accepted`, its normalized items, integer-cent total, and ID. The same key and canonical request returns HTTP 200 with the same ID. The same key with a different canonical request returns HTTP 409 with code `idempotency_conflict` and leaves the first result unchanged. `GET` returns the same order object; an unknown ID returns HTTP 404 with code `not_found`.

Preserve working behavior present in the supplied Brownfield candidate. Greenfield starts with an empty repository; this task does not introduce any route beyond those named at its start.
