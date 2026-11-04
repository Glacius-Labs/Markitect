---
kind: Operation
---
# End promotion

Ends the promotion: `ended` becomes true, so nobody can redeem it any more. Redemptions already made stay usable ([[promotions/redeemed-promotions]]).

- **Wire name:** `endPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `code`: identifier, naming an existing [[promotions/promotion]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the promotion's `ended` is already true.
- **Result:**
  - `promotion`: a [[promotions/promotion]] record
- **Permissions:** admin all.
- **Audit target:** `code`.
