---
kind: Operation
---
# Add locker

Creates the locker.

- **Wire name:** `addLocker`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `lockerId`: identifier
  - `monthlyCents`: integer from `1` to `1000000` cents (the monthly rent of a locker)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a locker with `lockerId` exists.
- **Result:**
  - `locker`: a [[lockers/locker]] record
- **Permissions:** admin all.
- **Audit target:** `lockerId`.
