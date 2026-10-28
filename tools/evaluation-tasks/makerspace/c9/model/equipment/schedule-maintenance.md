---
kind: Operation
---
# Schedule maintenance

Creates the window with status `scheduled` and the next maintenance window identifier. A reservation is active when its reported status is `reserved` or `checkedIn`.

- **Wire name:** `scheduleMaintenance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `start`: timestamp
  - `end`: timestamp
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `reservation_conflict` when the window overlaps an active reservation of the machine.
- **Result:**
  - `window`: a [[equipment/maintenance-window]] record
- **Permissions:** staff all, admin all, steward all.
- **Audit target:** the new `windowId`.
