---
kind: Operation
---
# Mark attendance

Records whether the member was present at the class. A member has one mark per class: a later mark of the same member for the same class replaces the earlier one, including its `present` value.

- **Wire name:** `markAttendance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `classId`: identifier, naming an existing [[classes/class]]
  - `memberId`: identifier, naming an existing [[members/member]]
  - `present`: flag
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the class is cancelled.
  2. `class_not_started` when the request time is before the `start` of the class. A request at the `start` is accepted.
  3. `not_enrolled` when the member has no enrollment in the course of the class, whatever its status ([[training/training-interface]]).
- **Result:**
  - `attendance`: a [[classes/attendance]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `classId/memberId`.
