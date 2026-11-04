---
kind: std/Rule
internal: true
---
# Payout calculation

The **teaching minutes** of a user in a month are the sum of the lengths in minutes, `end − start`, of the user's scheduled classes whose `start` lies in the month ([[classes/classes-interface]]). A class counts completely in the month of its `start`, even when it ends in the next month. Cancelled classes do not count. Only classes whose `instructorId` is the user count.

The amount of a [[payouts/payout]] is `(minutes × centsPerHour + 30) div 60`, where `centsPerHour` is the user's [[payouts/instructor-rate]] at the request time and `div` is integer division. This is the exact value rounded half up to a whole cent, once for the whole month ([[money]]). The amount can be 0.

Example: 90 and 45 minutes at 4500 cents per hour give 135 minutes and `(135 × 4500 + 30) div 60 = 10125` cents. One minute at 30 cents per hour gives `(30 + 30) div 60 = 1` cent; at 29 cents per hour it gives 0 cents. A class from `2026-03-31T23:30Z` to `2026-04-01T01:00Z` counts 90 minutes in `2026-03` and nothing in `2026-04`.
