---
kind: Operation
---
# Withdraw instructor

Removes the qualification of the user for the family. The `qualification` result is the record that was removed. Classes already held in the past stay as they are. A class that starts at the request time, or earlier, does not prevent the withdrawal.

- **Wire name:** `withdrawInstructor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier, naming an existing [[members/user]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `not_instructor` when the user holds no qualification for the family.
  2. `instructor_has_classes` when the user is the instructor of a scheduled class whose course belongs to the family and whose `start` is after the request time (classes interface).
- **Result:**
  - `qualification`: a [[instructors/qualification]] record
- **Permissions:** admin all.
- **Audit target:** `userId/familyId`.
