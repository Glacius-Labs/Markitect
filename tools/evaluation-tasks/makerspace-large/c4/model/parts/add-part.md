---
kind: Operation
---
# Add part

Creates the part with stock 0.

- **Wire name:** `addPart`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `partId`: identifier
  - `name`: text
  - `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit of a part)
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `already_exists` when a part with `partId` exists.
- **Result:**
  - `part`: a [[parts/part]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `partId`.
