---
kind: Interface
---
# Rooms interface

What other modules may use of rooms: its operations and the room and hold records. Other modules also ask for a room's capacity, whether a room is free for an interval, to hold a room for an interval and to release a hold they made.

- **Operations:** [[rooms/add-room]], [[rooms/hold-room]], [[rooms/release-room-hold]], [[rooms/get-room-schedule]].
- **Records:** [[rooms/room]], [[rooms/hold]].

What other modules ask of rooms that is not an operation:

- The `capacity` of a room.
- Whether a room is **free for** an interval, as [[rooms/room-holds]] states.
- Holding a room for an interval `[start, end)` on behalf of another module. The caller has checked that the room is free for the interval and states the hold's `purpose` and its origin, `class` or `event`. The hold is created as [[rooms/room-holds]] states.
- Releasing a hold that was made that way: its status becomes `released`. This is how a class or an event gives its room back. [[rooms/release-room-hold]] does not release such holds.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `hold_not_direct` | The hold was made for a class or an event, not by a direct hold of the room. |
| `room_taken` | A held hold of the room overlaps the interval. |
