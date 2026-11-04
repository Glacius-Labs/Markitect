---
kind: Operation
---
# Publish waiver

Makes a new waiver the family's current one: `version` is 1 for the first waiver of the family, otherwise the previous version plus 1, and `text` is as supplied. Earlier versions are no longer returned. Existing signatures stay, and [[waivers/waiver-gate]] decides what they are worth.

- **Wire name:** `publishWaiver`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `text`: text
- **Result:**
  - `waiver`: a [[waivers/waiver]] record
- **Permissions:** admin all.
- **Audit target:** `familyId`.
