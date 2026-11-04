---
kind: std/Rule
---
# Quote pricing

A [[quotes/quote]] is priced once, when it is sent, at the request time of the sending.

- Each line's `unitPriceCents` is captured: the machine's current rate in cents per minute for a `machine` line ([[equipment/machine]]), the material's current unit price for a `material` line ([[materials/material]]). Later changes of a rate or a price never change a sent quote.
- `subtotalCents` is the sum over the lines of `quantity × unitPriceCents`. No discount applies and nothing is rounded.
- `totalCents` is `(subtotalCents × 120 + 50) div 100`, which is the subtotal plus 20 percent, rounded half up to a whole cent once, as [[money]] states for amounts.

Example: a `machine` line of 30 minutes at 45 cents and a `material` line of 7 units at 236 cents give a subtotal of `1350 + 1652 = 3002` and a total of `(3002 × 120 + 50) div 100 = 3602` cents.

Stock is not reserved or drawn by a quote.
