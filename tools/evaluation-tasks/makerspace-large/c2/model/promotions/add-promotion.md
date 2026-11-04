---
kind: Operation
---
# Add promotion

Creates the promotion with `redemptions` 0 and `ended` false.

- **Wire name:** `addPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `code`: identifier
  - `percent`: integer from `1` to `100` (a discount in percent)
  - `scope`: a [[promotions/promotion-scope]]: `course`, `material` or `both`
  - `maxRedemptions`: integer from `1` to `100000` (the number of redemptions a promotion allows)
  - `validUntil`: date
- **Refusals** at check 7, in this order:
  1. `already_exists` when a promotion with `code` exists.
- **Result:**
  - `promotion`: a [[promotions/promotion]] record
- **Permissions:** admin all.
- **Audit target:** `code`.
