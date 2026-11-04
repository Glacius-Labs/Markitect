---
kind: Operation
---
# Classify hazard

Gives the material a [[hazmat/classification]]. When the material already has one, the new classification replaces it. A classification gates the purchases made after it; purchases made earlier are not changed and still count toward the limit.

Rules of other modules that apply here: [[hazmat/hazard-gate]].

- **Wire name:** `classifyHazard`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `monthlyLimit`: integer from `1` to `10000` (the most units of the material one member may buy in a month)
- **Result:**
  - `classification`: a [[hazmat/classification]] record
- **Permissions:** admin all.
- **Audit target:** `materialId`.
