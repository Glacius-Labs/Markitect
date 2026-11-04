---
kind: Operation
---
# Cancel shift

The shift gets status `cancelled`. A cancelled shift no longer covers anything, no longer counts in schedules and no longer overlaps other shifts of its user.

Rules of other modules that apply here: reservations.

- **Wire name:** `cancelShift`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `shiftId`: identifier, naming an existing [[staffing/shift]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the shift is already cancelled or its `start` is at or before the request time.
  2. `shift_needed` when [[staffing/supervision]] says the cancellation would leave an active reservation without cover.
- **Result:**
  - `shift`: a [[staffing/shift]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `shiftId`.
