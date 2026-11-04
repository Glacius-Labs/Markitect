---
kind: std/Rule
---
# Pack draw

At `checkOut`, minutes are drawn from the member's packs before the usage charge is created.

A pack is **valid on date D** when its `minutesLeft` is above 0 and D is before its `expiresOn`. The date used is the date of the request time of the check-out. A pack is not valid on its `expiresOn` date. Whether the pack's charge is paid, and the member's standing, do not matter.

The draw takes the member's valid packs in ascending `expiresOn` order, packs with the same `expiresOn` in creation order. From each pack in turn it draws the smaller of the pack's `minutesLeft` and the minutes still to cover, and lowers `minutesLeft` by that amount. It stops when the metered minutes are covered or the valid packs are used up. `drawn` is the total drawn. It is 0 when the metered minutes are 0 or no pack is valid. `billable` is the metered minutes less `drawn`.

Example: ada has three packs. `pak-1` expires `2026-05-01` with 25 minutes left, `pak-2` expires `2026-05-01` with 100 minutes left and `pak-3` expires `2026-06-01` with 20 minutes left. She checks out on `2026-04-30` with 50 metered minutes. The draw takes 25 minutes from `pak-1` and 25 from `pak-2`: `drawn` 50, `billable` 0, and `pak-2` has 75 minutes left. If she checks out with 50 metered minutes on `2026-05-01`, `pak-1` and `pak-2` are no longer valid, the draw takes 20 minutes from `pak-3`, `drawn` is 20 and `billable` is 30.
