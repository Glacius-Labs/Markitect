---
kind: Operation
---
# Create event

Creates the event with status `open`, `sold` 0 and the other fields as supplied. It makes a hold of the room for `[start, end)` as [[events/ticket-sales]] states. The `event` result is the new event.

Rules of other modules that apply here: [[rooms/room-holds]].

- **Wire name:** `createEvent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `eventId`: identifier
  - `title`: text
  - `roomId`: identifier, naming an existing [[rooms/room]]
  - `start`: timestamp
  - `end`: timestamp
  - `capacity`: integer from `1` to `1000` (the number of tickets of an event)
  - `priceCents`: integer from `0` to `1000000` cents (the price of one ticket)
- **Refusals** at check 7, in this order:
  1. `already_exists` when an event with `eventId` exists.
  2. `invalid_interval` when `end` is not after `start`.
  3. `starts_in_past` when `start` is before the request time.
  4. `capacity_too_large` when `capacity` is above the `capacity` of the room. A capacity equal to the room's is accepted.
  5. `room_taken` when the room is not free for `[start, end)` ([[rooms/rooms-interface]]).
- **Result:**
  - `event`: an [[events/event]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `eventId`.
