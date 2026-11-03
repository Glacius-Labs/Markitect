---
kind: std/Rule
---
# Attendance gate

At [[training/complete-course]] a member must have attended the classes of the course that began after the member enrolled.

Let `enrolledAt` be that of the member's enrollment in the course, which has status `enrolled` at that moment. A class **requires** the member's attendance when all of these hold:

- it is a class of the course;
- its status is `scheduled`;
- its `start` is at or before the request time;
- its `start` is at or after `enrolledAt`.

The member **attended** a class when an [[classes/attendance]] mark of the member for the class has `present` true. No mark, or a mark with `present` false, is not attendance. A class that has started but not ended already requires attendance.

The completion is refused with `attendance_missing` when some class requires the member's attendance and the member did not attend it. A course with no class that requires attendance passes the gate.

Example: the member enrolled at `2026-03-02T10:00Z`. A class that started on `2026-03-01` does not count, whatever its marks. A class that started at `2026-03-02T10:00Z` counts. A cancelled class does not count. A class that starts after the request time does not count. Of the classes that count, every one needs a mark with `present` true.
