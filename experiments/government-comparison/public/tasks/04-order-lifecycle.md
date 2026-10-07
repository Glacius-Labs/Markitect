# Task 4 — Complete order lifecycle

Implement `POST /orders/{id}/cancel` and `POST /orders/{id}/fulfill` with durable, consistent inventory effects. A successful order has status `accepted`.

- Cancelling an accepted order changes its status to `cancelled` and releases its reservation exactly once.
- Fulfilling an accepted order changes its status to `fulfilled`; fulfillment consumes the reserved units, reducing `onHand` and `reserved` by the same amount while leaving `available` unchanged.
- Repeating the same terminal action returns HTTP 200 and the existing order object without changing inventory again.
- Trying the opposite action after a terminal transition returns HTTP 409 with code `invalid_transition` and leaves order and inventory unchanged.
- Unknown IDs return HTTP 404 with code `not_found`.

Persist status and inventory changes as one operation. Preserve all released requirements and Brownfield behavior. Add focused checks for each transition, its repeat, its opposite, and inventory before/after.
