---
kind: Interface
---
# Promotions interface

What other modules may use of promotions: its operations, the promotion, redemption and scope records, and the refusals below. Other modules also ask for a member's oldest unused redemption and mark a redemption used.

- **Operations:** [[promotions/add-promotion]], [[promotions/redeem-promotion]], [[promotions/list-member-promotions]], [[promotions/end-promotion]].
- **Uses:** [[members/members-interface]].
- **Records:** [[promotions/promotion]], [[promotions/redemption]], [[promotions/promotion-scope]].
- **Capabilities** that are not operations, as [[promotions/redeemed-promotions|the redeemed promotions rule]] states them:
  - **The oldest unused redemption.** The caller names a member and a scope to cover, `course` or `material`. The answer is the member's oldest unused redemption whose promotion covers that scope, with its `percent`, or none.
  - **Use a redemption.** The caller names the redemption and the `ref` of the charge it is used for. The redemption gets `usedAt` the request time and `usedOn` that `ref`.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_redeemed` | The member has already redeemed the promotion. |
| `promotion_ended` | The promotion has been ended. |
| `promotion_exhausted` | The promotion has been redeemed as often as it allows. |
| `promotion_expired` | The request date is after the promotion's `validUntil`. |
