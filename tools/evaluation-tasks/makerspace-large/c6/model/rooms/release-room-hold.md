---
kind: Operation
---
# Release room hold

The hold gets status `released`. The room is free again for the hold's interval, unless another held hold overlaps it. A hold may be released before, during or after its interval, as long as it is held.

- **Wire name:** `releaseRoomHold`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `holdId`: identifier, naming an existing [[rooms/hold]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the hold is already released.
  2. `hold_not_direct` when the hold was not made by [[rooms/hold-room]] ([[rooms/room-holds]]).
- **Result:**
  - `hold`: a [[rooms/hold]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `holdId`.
