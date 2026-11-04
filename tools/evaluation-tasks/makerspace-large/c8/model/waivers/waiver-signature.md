---
kind: std/Concept
---
# Waiver signature

A member's signature of one version of a family's waiver. A member can have several signatures for one family, also of the same version; the latest is the one recorded last. A signature lapses a year after it is made, as [[waivers/waiver-gate]] defines.

**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `familyId`: identifier, naming an existing [[equipment/family]]
- `version`: count, at least 1 (the version of the waiver that was signed)
- `signedAt`: timestamp
