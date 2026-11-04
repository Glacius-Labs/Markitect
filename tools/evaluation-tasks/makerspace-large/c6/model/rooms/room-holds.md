---
kind: std/Rule
---
# Room holds

A [[rooms/room]] is kept for an interval `[start, end)` by a [[rooms/hold]]. A hold has status `held` until it is released; a released hold is never held again. Rooms are never removed.

A room is **free for** an interval when none of its held holds overlaps the interval ([[time]]). Released holds do not count. Two held holds of one room therefore never overlap. A hold of 10:00 to 11:00 and a hold of 11:00 to 12:00 do not overlap.

Every hold has an **origin**, which is not a field of the record:

- `direct` when [[rooms/hold-room]] made it. Its `purpose` is the text of the request.
- `class` when it was made for a scheduled class. Its `purpose` is `class ` followed by the class identifier, for example `class cls-3`.
- `event` when it was made for a created event. Its `purpose` is `event ` followed by the event identifier, for example `event summer-fair`.

A hold made for a class or an event is released only together with that class or event, when it is cancelled. [[rooms/release-room-hold]] refuses it. Making a hold for a class or an event consumes a hold identifier like any other hold.
