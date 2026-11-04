---
kind: Operation
---
# Set dunning ladder

Sets the ladder, replacing the one before. Cases that exist are not changed. The new ladder applies to the runs after this request.

- **Wire name:** `setDunningLadder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `noticeDays`: integer from `1` to `3650` (days overdue at which the notice is posted)
  - `feeDays`: integer from `1` to `3650` (days overdue at which the late fee is charged)
  - `feeCents`: integer from `1` to `1000000` cents (the late fee)
  - `suspendDays`: integer from `1` to `3650` (days overdue at which the member is suspended)
- **Refusals** at check 7, in this order:
  1. `invalid_ladder` unless `noticeDays` is less than `feeDays` and `feeDays` is less than `suspendDays`.
- **Result:**
  - `ladder`: a [[dunning/dunning-ladder]] record
- **Permissions:** admin all.
- **Audit target:** the fixed string `ladder`.
