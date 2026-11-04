---
kind: std/Rule
---
# Pass draw

At `checkOut`, minutes are drawn from the member's passes before the usage charge is created.

A pass is **valid on date D** when its `minutesLeft` is above 0 and D is before its `expiresOn`. The date used is the date of the request time of the check-out. A pass is not valid on its `expiresOn` date. Whether the pass's charge is paid, and the member's standing, do not matter.

The draw takes the member's valid passes in ascending `expiresOn` order, passes with the same `expiresOn` in creation order. From each pass in turn it draws the smaller of the pass's `minutesLeft` and the minutes still to cover, and lowers `minutesLeft` by that amount. It stops when the metered minutes are covered or the valid passes are used up. `drawn` is the total drawn. It is 0 when the metered minutes are 0 or no pass is valid. `billable` is the metered minutes less `drawn`.

Example: ada has three passes. `pas-1` expires `2026-05-01` with 25 minutes left, `pas-2` expires `2026-05-01` with 100 minutes left and `pas-3` expires `2026-06-01` with 20 minutes left. She checks out on `2026-04-30` with 50 metered minutes. The draw takes 25 minutes from `pas-1` and 25 from `pas-2`: `drawn` 50, `billable` 0, and `pas-2` has 75 minutes left. If she checks out with 50 metered minutes on `2026-05-01`, `pas-1` and `pas-2` are no longer valid, the draw takes 20 minutes from `pas-3`, `drawn` is 20 and `billable` is 30.
