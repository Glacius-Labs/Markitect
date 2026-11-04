---
kind: Operation
---
# Set supervision

Sets whether the family is supervised. Setting the current value succeeds without a change. Existing reservations are not touched.

- **Wire name:** `setSupervision`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `required`: flag
- **Result:**
  - `supervision`: a [[staffing/supervision-setting]] record
- **Permissions:** admin all.
- **Audit target:** `familyId`.
