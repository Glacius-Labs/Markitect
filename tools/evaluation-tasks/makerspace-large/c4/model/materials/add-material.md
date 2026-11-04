---
kind: Operation
---
# Add material

Creates the material with stock 0.

- **Wire name:** `addMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `materialId`: identifier
  - `name`: text
  - `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a material with `materialId` exists.
- **Result:**
  - `material`: a [[materials/material]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `materialId`.
