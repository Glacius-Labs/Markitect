---
kind: Operation
---
# Get machine schedule

Shows one machine's day: `openingHours` holds that day's opening and closing times, or `null` on Sunday; `reservations` are the machine's reservations whose `start` falls on that date, in every status except `cancelled`; `maintenance` are the machine's scheduled windows that overlap the interval from `00:00` of that date to `00:00` of the next date.

- **Wire name:** `getMachineSchedule`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `date`: date
- **Result:**
  - `machineId`: identifier
  - `date`: date
  - `openingHours`: a [[bookings/opening-hours-of-day]] record, or `null`
  - `reservations`: a list of [[bookings/reservation]] records
  - `maintenance`: a list of [[equipment/maintenance-window]] records
- **Permissions:** member all, staff all, admin all.
