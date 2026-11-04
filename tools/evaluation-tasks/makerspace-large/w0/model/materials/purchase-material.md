---
kind: Operation
---
# Purchase material

Sells material to the member. The stock decreases by `quantity`. A purchase is created with the next purchase identifier, the material's current unit price, the `discountPercent` below, the amount by [[money]] and `at` the request time. A material [[billing/charge]] is created for the member: `kind` `material`, `ref` the purchase identifier, and the purchase's `quantity`, `unitPriceCents`, `discountPercent`, `amountCents` and `at`. The `material` result is the material after the change.

`discountPercent` is the `percent` of the member's oldest unused redeemed promotion whose scope covers materials (`material` or `both`), otherwise 0. That redemption is then used by this purchase, as [[promotions/redeemed-promotions]] states. A refused request uses no promotion.

Example: 3 units at 250 cents with a redeemed promotion of 20 percent: `(3 × 250 × 80 + 50) div 100 = 600` cents.

Rules of other modules that apply here: [[billing/charges-and-invoices]], [[members/membership]], hazard gate, stocktake freeze and [[promotions/redeemed-promotions]].

- **Wire name:** `purchaseMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `10000` (the number of units bought in one request)
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `hazard_not_certified` when the material has a classification and the member has no certification for its family valid on the date of the request time (hazard gate).
  3. `hazard_limit_exceeded` when the material has a classification and the units of the material the member bought in the month of the request time, plus `quantity`, are more than its `monthlyLimit` (hazard gate).
  4. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
  5. `insufficient_stock` when the stock is lower than `quantity`.
- **Result:**
  - `purchase`: a [[materials/purchase]] record
  - `material`: a [[materials/material]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `purchaseId`.
