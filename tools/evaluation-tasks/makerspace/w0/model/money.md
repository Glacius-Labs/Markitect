---
kind: std/Rule
---
# Money

- All amounts are integer numbers of [[values|cents]], represented as JSON integers. There are no fractions of a cent, no floating-point amounts and no currency field. Stored and returned amounts are never negative.
- Every charge is a line with a `quantity`, a `unitPriceCents` and a `discountPercent`. Its amount is
  `amountCents = (quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`,
  where `div` is integer division. This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again.
  Example: 37 minutes at 45 cents with a 10 % discount: `(37 × 45 × 90 + 50) div 100 = 149900 div 100 = 1499` cents.
- A charge's amount is fixed when the charge is created. Later changes to a rate, a tier or a price never change existing charges.
- Each member has a credit balance, initially 0. It increases by credits issued by an admin and by overpayments. It decreases only when credit is applied to a new invoice. It is never negative. Credit is never paid out.
