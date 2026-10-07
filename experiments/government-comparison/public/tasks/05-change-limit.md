# Task 5 — Apply the changed order limit and align entry points

The order quantity limit has changed from 10 to **6 inclusive**. Apply it to every active order entry point. A sum above six returns HTTP 400 with code `order_limit`; a sum of six remains valid when inventory allows it.

Add `POST /legacy/orders` as an equivalent alias for `POST /orders` in Greenfield. In Brownfield, preserve the existing legacy client behavior while applying the changed limit to both routes. Both endpoints accept the same JSON and `Idempotency-Key` and produce equivalent status, order, idempotency, reservation, and persistence behavior.

Update code, checks, and affected run/documentation text so there is one active limit. Preserve lifecycle behavior and valid neighboring inputs.
