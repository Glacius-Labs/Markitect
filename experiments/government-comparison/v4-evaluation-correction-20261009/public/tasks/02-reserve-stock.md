# Task 2 — Reserve inventory atomically

Add `GET /inventory/{sku}`. Its JSON response contains `sku`, `onHand`, `reserved`, and `available`, where `available` equals `onHand - reserved`. Initial WIDGET stock is 10 and initial GADGET stock is 6. An unknown SKU returns HTTP 404 with error code `unknown_sku`, preserving the Brownfield read contract.

Change successful order creation to reserve every requested unit atomically. The order remains `accepted`. If any requested quantity is unavailable, return HTTP 409 with error code `insufficient_stock`; do not create an accepted order or change any item's reservation. Concurrent requests must not reserve the same unit twice.

Repeating a key with the same canonical request returns HTTP 200 with the same order ID and no additional reservation. A different canonical request with that key returns HTTP 409 `idempotency_conflict` and changes no state. Keep all task 1 status codes, error codes, totals, and Brownfield behavior. Add checks for multi-SKU all-or-nothing behavior and competing requests at a stock boundary.
