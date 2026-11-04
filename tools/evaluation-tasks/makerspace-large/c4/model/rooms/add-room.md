---
kind: Operation
---
# Add room

Creates the room. It has no holds.

- **Wire name:** `addRoom`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `roomId`: identifier
  - `name`: text
  - `capacity`: integer from `1` to `1000` (the number of people a room holds)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a room with `roomId` exists.
- **Result:**
  - `room`: a [[rooms/room]] record
- **Permissions:** admin all.
- **Audit target:** `roomId`.
