---
kind: std/Concept
---
# Dunning ladder

The steps taken against an overdue invoice and the days at which they are taken. There is at most one ladder; none exists until one is set, and setting a ladder replaces it.

**Fields**, in the formats of [[values]]:

- `noticeDays`: integer from `1` to `3650` (days overdue at which the notice is posted)
- `feeDays`: integer from `1` to `3650` (days overdue at which the late fee is charged)
- `feeCents`: integer from `1` to `1000000` cents (the late fee)
- `suspendDays`: integer from `1` to `3650` (days overdue at which the member is suspended)

`noticeDays` is less than `feeDays`, and `feeDays` is less than `suspendDays`.
