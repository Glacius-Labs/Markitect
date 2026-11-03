# Notes: teaching modules

Modules: `rooms`, `instructors`, `classes`, `assessments`, `waitlists`, `payouts`, `events`; changes to `training` (`enrollInCourse`, `completeCourse`, the interface). 28 new operations, 7 interfaces, 7 registry pages. `pages.py check` and `render` pass.

## (a) Generated identifier prefixes

| Prefix | Record | Module |
|---|---|---|
| `hld-N` | room hold | rooms (holds made for classes and events take numbers from the same counter) |
| `cls-N` | class | classes |
| `pyt-N` | payout | payouts |
| `tkt-N` | event ticket | events |

`roomId`, `eventId` are named by the caller. Qualifications, attendance marks, assessments, attempts, waiting entries and instructor rates have no identifier of their own.

## (b) Lists for the list-order page

| List | Order |
|---|---|
| `getRoomSchedule` → `holds` | `start` ascending, then creation order |
| `listInstructors` → `instructors` | `userId` ascending |
| `listCourseClasses` → `classes` | `start` ascending, then creation order |
| `listClassAttendance` → `attendance` | `memberId` ascending |
| `listAttempts` → `attempts` | the order the attempts were recorded |
| `listWaitlist` → `entries` | join order |
| `promoteWaitlist` → `enrollments`, `skipped` | join order |
| `generatePayouts` → `payouts` | `userId` ascending (which is creation order) |
| `listPayouts` → `payouts` | creation order |
| `listEventTickets` → `tickets` | creation order |
| `cancelEvent` → `tickets` | creation order |

## (c) What other modules' interfaces must offer

- billing: "Create a charge" (events, `event` kind) and "Add credit" with debit account `revenueRefunds` and memo the `ticketId` (events). Both are in the billing interface already.
- ledger: "Post an entry" for the two payout rows. Already there.
- notices: posting a `waitlist` notice with the course as `ref`; the subject comes from `notice-kind`. Already there.
- members: user exists and its role (instructor and payout `userId`), standing (waitlists, events). No new demand.
- equipment: family exists. No new demand.
- protocol: supplies three declared needs. Training from classes (attendance owed) and from assessments (assessment met). Instructors from classes (an instructor has a scheduled class of a family starting after a time).
- Authorization page: `ticketId` is owned when the ticket's member is the actor (`cancelTicket`). Already in DESIGN.
- Initial state: no rooms, holds, qualifications, classes, marks, assessments, attempts, waiting entries, instructor rates, payouts, events or tickets.
- Charge kind page: `event` is used by `sellTicket`. Already in DESIGN.
- Refusal codes: `payouts` uses `month_not_ended`, owned by billing, whose table says "The month to invoice has not ended". Integrator: widen billing's meaning to "The month has not ended at the request time", or give payouts its own code.

## (d) Gaps in DESIGN and what I chose

1. `instructors` needs `classes` for `instructor_has_classes`, but `classes` depends on `instructors` and the table lists no need. I declared the need in the instructors interface (protocol supplies it). Integrator: add `classes` to the Needs column of `instructors` and to protocol's table.
2. `hold_not_direct` needs the origin of a hold, and the hold has no such field. I kept the origin as internal state (`direct`, `class`, `event`) and fixed the purposes of class and event holds: `class <classId>`, `event <eventId>`. No field was added.
3. Result members are not in DESIGN. Mutations return the changed record under its record name (`room`, `hold`, `qualification`, `class`, `attendance`, `assessment`, `attempt`, `entry`, `rate`, `payout`, `event`, `ticket`); `sellTicket` and `cancelTicket` also return `event`; `cancelEvent` returns `event` and `tickets`; `generatePayouts` returns `month` and `payouts`; `promoteWaitlist` returns `courseId`, `enrollments`, `skipped` (`memberId`, `code`). Lists return their key (`courseId`, `classId`, `eventId`, `userId`, `familyId`, `roomId`/`date`) and the list.
4. `passMark`: an attempt passes when `score` is at least `passMark`.
5. Attempt periods: "since the member's current `enrolledAt`" is ambiguous when a completion and a new enrollment share a minute. I define periods by enrollment, and training offers the count of the member's enrollments in the course so far (a capability, not a record field). Assessments keeps it with each attempt.
6. `sold` is the number of valid tickets; cancelling lowers it, so a cancelled ticket gives its place back. A cancelled event has `sold` 0.
7. `withdrawInstructor`: "a scheduled class of that family" means a class the user teaches, of a course of that family, starting after the request time.
8. `generatePayouts`: "per instructor" means every user with a rate and a scheduled class in the month, even when the qualification was withdrawn later. A payout of 0 cents is created and posts nothing. The month counts as generated even when no payout results.
9. `promoteWaitlist` stops when the course has no free place; entries not tried are not in `skipped`. Enrolling directly leaves the entry on the list, so a later promotion skips it with `already_enrolled`.
10. `cancelTicket`: `too_late` includes an event that has started. The ticket's charge is not changed, and the credit is given whatever its invoice state.
11. Attendance gate: a class requires attendance when it is scheduled and its `start` is at or before the request time and at or after the current `enrolledAt`. A class in progress counts.
12. `not_enrolled` is used with three conditions (training: no `enrolled` enrollment; classes: no enrollment of any status; assessments: no `enrolled` enrollment). Training's table says the operation states a different condition.
