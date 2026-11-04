---
kind: Operation
---
# Add shift

Creates the shift with status `scheduled` and the next shift identifier. A shift may start and end on any minute. The user may be staff or an admin.

- **Wire name:** `addShift`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier, naming an existing [[members/user]]
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `not_staff` when the user is a member.
  2. `invalid_interval` when `end` is not after `start`.
  3. `starts_in_past` when `start` is before the request time.
  4. `shift_too_long` when the shift is longer than 720 minutes; exactly 720 is allowed.
  5. `shift_overlap` when another scheduled shift of the same user overlaps `[start, end)`.
- **Result:**
  - `shift`: a [[staffing/shift]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `shiftId`.
