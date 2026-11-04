---
kind: std/Rule
internal: true
---
# Metering

The member checks in when starting to use the machine and checks out when done. The program meters the minutes in between and charges them at the rate captured at check-in, with the member's tier discount.

The member's packs pay for metered minutes first: the minutes a pack draws are not charged ([[packs/pack-draw]]). The charge is for the remaining, billable minutes. The materials that the machine uses up are charged on the full metered minutes, whatever the packs drew ([[consumables/consumption]]).

Example: `ada` has tier `plus` (usage discount 10 %). `laser-1` has a rate of 40 cents per minute and the consumption rows `acrylic` at 25 units per hour and `ply` at 6 units per hour. The stock of `acrylic` is 15 units at 30 cents, the stock of `ply` is 100 units at 12 cents. `ada` holds one valid pack with 20 minutes left. She checks in at `2026-03-03T10:00Z` and checks out at `2026-03-03T10:50Z`. `checkOut` then does this, in this order:

1. `meteredMinutes` is 50. The pack draws 20 minutes and has 0 left, so `billable` is 30.
2. The usage charge has quantity 30, unit price 40 and discount 10: `(30 × 40 × 90 + 50) div 100 = 1080` cents. The reservation's `amountCents` is 1080, and the result's `charge` is this charge.
3. `acrylic` has `units` `(50 × 25 + 59) div 60 = 21`; the stock holds 15, so 15 are drawn, the stock becomes 0 and the charge is quantity 15 at 30 cents, `(15 × 30 × 100 + 50) div 100 = 450` cents. The shortfall of 6 is not an error.
4. `ply` has `units` `(50 × 6 + 59) div 60 = 5`; 5 are drawn, the stock becomes 95 and the charge is quantity 5 at 12 cents, 60 cents.

The three charges are created in the order usage, `acrylic`, `ply`. Both material charges have discount 0, `kind` `material` and `ref` the reservation identifier.
