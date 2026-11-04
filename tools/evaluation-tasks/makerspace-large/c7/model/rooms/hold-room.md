---
kind: Operation
---
# Hold room

Creates a hold of the room with status `held`, the next hold identifier and `purpose` as supplied. The hold is direct ([[rooms/room-holds]]). The room has no limit on the length of a hold and no opening hours.

Example: the room `lab` has a held hold from `2026-03-03T10:00Z` to `2026-03-03T12:00Z`. A hold from `12:00` to `13:00` is accepted; a hold from `11:30` to `12:30` is refused with `room_taken`.

- **Wire name:** `holdRoom`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `roomId`: identifier, naming an existing [[rooms/room]]
  - `start`: timestamp
  - `end`: timestamp
  - `purpose`: text
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `room_taken` when the room is not free for `[start, end)` ([[rooms/room-holds]]).
- **Result:**
  - `hold`: a [[rooms/hold]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `holdId`.
