---
kind: Operation
---
# Leave waitlist

Removes the member's entry from the course's waiting list. The `entry` result is the entry that was removed. Standing is not checked.

- **Wire name:** `leaveWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `not_waiting` when the member is not on the waiting list of the course.
- **Result:**
  - `entry`: a [[waitlists/waiting-entry]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.
