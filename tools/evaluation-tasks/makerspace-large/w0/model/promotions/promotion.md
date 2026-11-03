---
kind: std/Concept
---
# Promotion

A discount code that members redeem and later use once on a course or material charge.

**Fields**, in the formats of [[values]]:

- `code`: identifier
- `percent`: integer from `1` to `100` (a discount in percent)
- `scope`: a [[promotions/promotion-scope]]: `course`, `material` or `both`
- `maxRedemptions`: integer from `1` to `100000` (the number of redemptions a promotion allows)
- `validUntil`: date (the last date on which the promotion can be redeemed)
- `redemptions`: count (the number of redemptions so far)
- `ended`: flag (true once the promotion has been ended)

A record is identified by `code`.
