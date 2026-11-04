---
kind: Operation
---
# Get waiver

Returns the family's current waiver.

- **Wire name:** `getWaiver`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `no_waiver` when the family has no waiver.
- **Result:**
  - `waiver`: a [[waivers/waiver]] record
- **Permissions:** member all, staff all, admin all.
