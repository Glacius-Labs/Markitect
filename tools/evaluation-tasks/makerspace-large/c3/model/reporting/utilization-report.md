---
kind: Operation
---
# Utilization report

Shows how much each machine was used compared with the time it was available. The report range is the interval from `00:00` of `from` to `00:00` of `to`; `to` itself is excluded. `machines` has one row per machine that exists at the request time.

- `openMinutes`: minutes of the range that lie inside [[bookings/opening-hours|opening hours]]. Every machine gets the full count, regardless of when it was added.
- `maintenanceMinutes`: minutes of the range that lie inside opening hours and inside at least one scheduled [[equipment/maintenance-window|maintenance window]] of the machine. Overlapping windows count once.
- `availableMinutes`: `openMinutes − maintenanceMinutes`.
- `usedMinutes`: for every completed [[bookings/reservation]] of the machine, the minutes of `[checkedInAt, checkedOutAt)` that lie in the range, summed. Opening hours do not limit this count. Reservations that are checked in but not checked out are not counted.
- `utilizationPercent`: `(usedMinutes × 100 × 2 + availableMinutes) div (availableMinutes × 2)`, which is `usedMinutes × 100 / availableMinutes` rounded half up to an integer, or 0 when `availableMinutes` is 0. It can exceed 100.
- `totals`: the sums of `openMinutes`, `maintenanceMinutes`, `availableMinutes` and `usedMinutes` over all rows, and `utilizationPercent` computed from these sums with the same formula. With no machines, all totals are 0.

Rules of other modules that apply here: [[bookings/reservations]] and [[equipment/maintenance]].

- **Wire name:** `utilizationReport`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `from`: date
  - `to`: date
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `to` is not after `from` or the range is longer than 366 days.
- **Result:**
  - `from`: date
  - `to`: date
  - `machines`: a list of [[reporting/utilization-row]] records
  - `totals`: a [[reporting/utilization-totals]] record
- **Permissions:** staff all, admin all.
