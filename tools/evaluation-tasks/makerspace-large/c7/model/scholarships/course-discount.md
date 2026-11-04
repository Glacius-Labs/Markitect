---
kind: std/Rule
---
# Course discount

When a member enrolls in a course (`enrollInCourse`, also when `promoteWaitlist` enrolls a member), the course charge's `discountPercent` is decided from two sources, using the course's `feeCents` as the charge's unit price and a quantity of 1:

- **The award.** The member's active [[scholarships/award]] has a percent `p`. Its **discount** is `feeCents − amountCents`, where `amountCents` is the amount the fee would have at `p` by [[money]]. The award is **covered** when the discount is at most the remaining budget of its [[scholarships/fund]], which is `budgetCents` less `spentCents`. A discount equal to the remaining budget is covered. An award that is not covered, or a member without an active award, gives no candidate.
- **The promotion.** The member's oldest unused redemption that covers courses (redeemed promotions) has a percent `q`. A member without one gives no candidate.

The candidate with the higher percent wins; on equal percent the award wins. The comparison is of percents only, whatever amounts result. With one candidate it wins; with none the `discountPercent` is 0. A course with a fee of 0 uses neither source and has `discountPercent` 0.

Only the winner is used. If the award wins, the discount is added to the fund's `spentCents`. If the promotion wins, the redemption gets `usedAt` the request time and `usedOn` the course identifier. The loser is not touched: an award that is not covered gives no discount, does not change the fund, and no refusal results.

Example: course fee 12000. `ada` has an award of 30 % from a fund with budget 10000 and spent 6400. The discount is `12000 − 8400 = 3600`, and the remaining budget is `3600`, so the award is covered. She also has an unused redemption of 30 % that covers courses. The percents are equal, so the award wins: `discountPercent` is 30, the fund's `spentCents` becomes 10000, and the redemption stays unused. With the fund's spent amount at 6401 the award is not covered; the promotion is the only candidate and wins, and the redemption is used with `usedOn` the course identifier. With a redemption of 25 % and the award covered, the award's 30 % is higher and wins; with a redemption of 40 %, the redemption wins and the fund is untouched.
