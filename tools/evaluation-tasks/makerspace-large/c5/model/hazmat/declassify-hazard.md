---
kind: Operation
---
# Declassify hazard

Removes the material's [[hazmat/classification]]. Later purchases of the material are no longer judged by the [[hazmat/hazard-gate]]. The `classification` result is the classification that was removed.

- **Wire name:** `declassifyHazard`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `materialId`: identifier, naming an existing [[materials/material]]
- **Refusals** at check 7, in this order:
  1. `not_hazardous` when the material has no classification.
- **Result:**
  - `classification`: a [[hazmat/classification]] record
- **Permissions:** admin all.
- **Audit target:** `materialId`.
