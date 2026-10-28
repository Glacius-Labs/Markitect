---
kind: Operation
---
# Purchase material

Sells material to the member. The stock decreases by `quantity`. A purchase is created with the next purchase identifier, the material's current unit price, the [[materials/bulk-discount|bulk discount]] of its quantity as `discountPercent`, the amount by [[money]] and `at` the request time. A material [[billing/charge]] is created for the member: `kind` `material`, `ref` the purchase identifier, and the purchase's `quantity`, `unitPriceCents`, `discountPercent`, `amountCents` and `at`. The `material` result is the material after the change.

Rules of other modules that apply here: [[billing/charges]], [[billing/due-date-and-block]] and [[members/membership]].

- **Wire name:** `purchaseMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `10000` (the number of units bought in one request)
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `account_overdue` when the member is blocked ([[billing/billing-interface]]).
  3. `insufficient_stock` when the stock is lower than `quantity`.
- **Result:**
  - `purchase`: a [[materials/purchase]] record
  - `material`: a [[materials/material]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `purchaseId`.
