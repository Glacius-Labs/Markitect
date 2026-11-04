---
kind: Operation
---
# Approve order

The order's status becomes `approved`. Nobody approves an order they created, not even an admin. An order whose `totalCents` is above 100000 cents is approved by an admin only; an order of exactly 100000 cents can be approved by staff. A supplier that is blocked at the request time blocks the approval, whoever acts.

- **Wire name:** `approveOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `orderId`: identifier, naming an existing [[purchasing/order]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `submitted`.
  2. `self_approval` when the actor is the order's `createdBy`.
  3. `approval_requires_admin` when `totalCents` is above 100000 and the actor is staff.
  4. `supplier_blocked` when the order's supplier has status `blocked`.
- **Result:**
  - `order`: a [[purchasing/order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.
