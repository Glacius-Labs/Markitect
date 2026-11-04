---
kind: Operation
---
# Cancel class

The class gets status `cancelled`. Its room hold is released, so the room is free again for the class's interval. Attendance marks of the class stay, but a cancelled class requires no attendance ([[classes/attendance-gate]]).

- **Wire name:** `cancelClass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `classId`: identifier, naming an existing [[classes/class]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the class is already cancelled or its `start` is at or before the request time.
- **Result:**
  - `class`: a [[classes/class]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `classId`.
