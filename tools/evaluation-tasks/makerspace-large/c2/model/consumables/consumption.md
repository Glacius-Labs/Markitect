---
kind: std/Rule
---
# Consumption

A machine can use up [[materials/material|materials]] while it runs, as its [[consumables/consumption-row|consumption rows]] say. The draw happens once for each session, in `checkOut`, after the usage charge is created. Rows are read as they stand at that request. For each row of the machine, in ascending `materialId` order:

1. `units` is `(meteredMinutes × unitsPerHour + 59) div 60`: the metered minutes times `unitsPerHour`, divided by 60 and rounded up. It uses the full metered minutes of the reservation, whatever the member's passes covered. With 0 metered minutes `units` is 0.
2. `drawn` is the smaller of `units` and the material's stock at that moment.
3. The material's stock decreases by `drawn`.
4. When `drawn` is above 0, a material [[billing/charge]] is created for the member: `kind` `material`, `ref` the reservation identifier, `quantity` `drawn`, `unitPriceCents` the material's current unit price, `discountPercent` 0, `at` the request time, and the amount by [[money]]. When `drawn` is 0, no charge is created.

A shortfall, where `drawn` is below `units`, is not an error: the session is still checked out and the missing units are neither charged nor carried over. No promotion applies. The draw creates no [[materials/purchase]], and an open stocktake does not stop it. A machine with no rows draws nothing.

Example: a session of 50 metered minutes on a row of 25 units per hour has `units` `(50 × 25 + 59) div 60 = 1309 div 60 = 21`. With a stock of 15, `drawn` is 15, the stock becomes 0 and the charge has quantity 15.
