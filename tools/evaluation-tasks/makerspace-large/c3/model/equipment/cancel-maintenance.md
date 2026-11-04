---
kind: Operation
---
# Cancel maintenance

The window gets status `cancelled`. A cancelled window no longer blocks reservations and no longer counts in schedules or reports.

- **Wire name:** `cancelMaintenance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `windowId`: identifier, naming an existing [[equipment/maintenance-window]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the window is already cancelled or its start is at or before the request time.
- **Result:**
  - `window`: a [[equipment/maintenance-window]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `windowId`.
