# Task 5 â€” Apply the changed order limit and align entry points

The order quantity limit has changed from 10 to **6 inclusive**. Apply it to every active order entry point. A sum above six returns HTTP 400 with code `order_limit`; a sum of six remains valid when inventory allows it.

Add `POST /legacy/orders` as an equivalent alias for `POST /orders` in Greenfield. In Brownfield, preserve the existing legacy client behavior while applying the changed limit to both routes. Both endpoints accept the same JSON and `Idempotency-Key` and produce equivalent status, order, idempotency, reservation, and persistence behavior.

Update code, checks, and affected run/documentation text so there is one active limit. Preserve lifecycle behavior and valid neighboring inputs.

## v4 prospective clarification

The six-unit limit applies to new orders. A canonical retry of an order accepted under the prior limit keeps its original ID and result even when that original order exceeds the new limit. A changed canonical request with the same existing key still conflicts and leaves accepted state unchanged. Accepted earlier orders remain readable and retain their released lifecycle behavior. This clarifies continued idempotency/preservation; it does not require newly accepting an over-limit order.
