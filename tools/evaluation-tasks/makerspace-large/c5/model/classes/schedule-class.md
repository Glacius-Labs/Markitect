---
kind: Operation
---
# Schedule class

Creates the class with status `scheduled` and the next class identifier. It makes a hold of the room for `[start, end)` as [[classes/class-schedule]] states. The `class` result is the new class.

Example: the instructor `kim` teaches a class from `10:00` to `12:00`. A class of `kim` from `12:00` to `13:00` is accepted; a class of `kim` from `11:30` to `13:00` in another room is refused with `instructor_busy`. If the second class is in the room of the first, the refusal is `room_taken`.

Rules of other modules that apply here: [[rooms/room-holds]].

- **Wire name:** `scheduleClass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `instructorId`: identifier, naming an existing [[members/user]]
  - `roomId`: identifier, naming an existing [[rooms/room]]
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `not_instructor` when the instructor holds no qualification for the family of the course ([[instructors/instructors-interface]]).
  4. `room_taken` when the room is not free for `[start, end)` ([[rooms/rooms-interface]]).
  5. `instructor_busy` when another scheduled class of the instructor overlaps `[start, end)`.
- **Result:**
  - `class`: a [[classes/class]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `classId`.
