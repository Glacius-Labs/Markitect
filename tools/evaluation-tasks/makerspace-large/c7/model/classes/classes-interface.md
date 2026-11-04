---
kind: Interface
---
# Classes interface

What other modules may use of classes: its operations and the class and attendance records. Other modules also ask whether a member still owes attendance for a course, for the scheduled classes of an instructor that start in a month, and whether an instructor has a scheduled class of a family that starts after a given time.

- **Operations:** [[classes/schedule-class]], [[classes/cancel-class]], [[classes/mark-attendance]], [[classes/list-course-classes]], [[classes/list-class-attendance]].
- **Uses:** [[members/members-interface]], [[training/training-interface]], [[instructors/instructors-interface]], [[rooms/rooms-interface]].
- **Records:** [[classes/class]], [[classes/attendance]].

What other modules ask of classes that is not an operation:

- Whether a member still owes attendance for a course, given the course, the member, the `enrolledAt` of the member's enrollment and the request time: yes when [[classes/attendance-gate]] finds a class that requires the member's attendance and the member did not attend it.
- The scheduled classes of a user as instructor whose `start` lies in a given month, each with its length in minutes, which is `end − start`.
- Whether a user is the instructor of a scheduled class whose course belongs to a given family and whose `start` is after a given time.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `attendance_missing` | The member did not attend a class of the course that requires the member's attendance. |
| `class_not_started` | The class has not started yet at the request time. |
| `instructor_busy` | Another scheduled class of the instructor overlaps the class. |
