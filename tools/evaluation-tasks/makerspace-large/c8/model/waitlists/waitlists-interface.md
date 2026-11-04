---
kind: Interface
---
# Waitlists interface

What other modules may use of waitlists: its operations and the waiting entry and skipped entry records. No other module asks anything else of waitlists.

- **Operations:** [[waitlists/join-waitlist]], [[waitlists/leave-waitlist]], [[waitlists/list-waitlist]], [[waitlists/promote-waitlist]].
- **Uses:** [[members/members-interface]], [[training/training-interface]], [[notices/notices-interface]].
- **Records:** [[waitlists/waiting-entry]], [[waitlists/skipped-entry]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_waiting` | The member is already on the waiting list of the course. |
| `course_not_full` | The course has a free place, so there is nothing to wait for. |
| `not_waiting` | The member is not on the waiting list of the course. |
