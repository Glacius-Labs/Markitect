---
kind: Operation
---
# Get machine schedule

Shows one machine's day: `openingHours` holds that day's opening and closing times; every day has opening hours; `reservations` are the machine's reservations whose `start` falls on that date, in every status except `cancelled`; `maintenance` are the machine's scheduled windows that overlap the interval from `00:00` of that date to `00:00` of the next date; `lockouts` are the machine's lockouts that overlap the same interval.

- **Wire name:** `getMachineSchedule`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `date`: date
- **Result:**
  - `machineId`: identifier
  - `date`: date
  - `openingHours`: a [[bookings/opening-hours-of-day]] record
  - `reservations`: a list of [[bookings/reservation]] records
  - `maintenance`: a list of [[equipment/maintenance-window]] records
  - `lockouts`: a list of [[bookings/lockout]] records
- **Permissions:** member all, staff all, admin all, steward all.
