---
kind: Operation
---
# Redeem promotion

The member redeems the promotion. A redemption is created with `percent` the promotion's percent, `redeemedAt` the request time, `usedAt` `null` and `usedOn` `null`. The promotion's `redemptions` increases by 1. Redeeming on the date `validUntil` is allowed; the day after is not.

Rules of other modules that apply here: [[members/membership]] and [[promotions/redeemed-promotions]].

- **Wire name:** `redeemPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `code`: identifier, naming an existing [[promotions/promotion]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `promotion_ended` when the promotion's `ended` is true.
  3. `promotion_expired` when the date of the request time is after `validUntil`.
  4. `promotion_exhausted` when `redemptions` is equal to `maxRedemptions`.
  5. `already_redeemed` when the member has a redemption of this `code`.
- **Result:**
  - `redemption`: a [[promotions/redemption]] record
  - `promotion`: a [[promotions/promotion]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/code`.
