---
kind: std/Concept
---
# Redemption

One member's redemption of one [[promotions/promotion]]. It is unused while `usedAt` is `null`, and it can be used once ([[promotions/redeemed-promotions]]).

**Fields**, in the formats of [[values]]:

- `code`: identifier, naming an existing [[promotions/promotion]]
- `memberId`: identifier, naming an existing [[members/member]]
- `percent`: integer from `1` to `100` (the promotion's percent when the redemption was made)
- `redeemedAt`: timestamp
- `usedAt`: timestamp, or `null`
- `usedOn`: string, or `null` (the `ref` of the charge the redemption was used for)

A redemption is identified by its `code` together with its `memberId`: a member redeems a code at most once.
