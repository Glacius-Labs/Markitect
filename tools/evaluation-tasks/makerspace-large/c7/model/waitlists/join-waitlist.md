---
kind: Operation
---
# Join waitlist

Adds the member to the end of the course's waiting list ([[waitlists/waitlists]]). The entry gets `joinedAt` equal to the request time. The `entry` result is the new entry. A member who completed the course may join.

- **Wire name:** `joinWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `already_enrolled` when the member's enrollment in the course has status `enrolled` ([[training/training-interface]]).
  3. `course_not_full` when the course has a free place.
  4. `already_waiting` when the member is on the waiting list of the course.
- **Result:**
  - `entry`: a [[waitlists/waiting-entry]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.
