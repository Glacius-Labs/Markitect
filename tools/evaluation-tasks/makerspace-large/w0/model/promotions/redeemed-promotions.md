---
kind: std/Rule
---
# Redeemed promotions

A redemption is **unused** while its `usedAt` is `null`. A redemption **covers** a scope when its promotion's [[promotions/promotion-scope|scope]] is that scope or `both`.

The **oldest unused** redemption of a member for a scope is the first, in creation order of the member's redemptions, that is unused and covers the scope. Creation order is the order in which the member redeemed.

The operation that applies a discount decides whether a redemption is used: course discount for enrolling in a course and `purchaseMaterial` for a material purchase. A redemption that is used gets `usedAt` the request time and `usedOn` the `ref` of the charge it was used for (the course identifier or the purchase identifier). The charge's `discountPercent` is the redemption's `percent`. Using happens together with the operation's other effects, so a refused request uses nothing. A redemption is used at most once.

A redemption stays usable after its promotion has been ended and after the promotion's `validUntil` has passed: these matter only when redeeming.

Example: ada redeems `A` (scope `course`, 20 %), then `B` (scope `both`, 15 %), then `C` (scope `material`, 30 %). She buys material: her oldest unused redemption that covers materials is `B`, because `A` does not cover materials. The purchase gets `discountPercent` 15, and `B` gets `usedOn` the purchase identifier. She then enrolls in a course: her oldest unused redemption that covers courses is now `A`, since `B` is used.
