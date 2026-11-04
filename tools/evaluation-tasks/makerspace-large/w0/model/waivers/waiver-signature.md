---
kind: std/Concept
---
# Waiver signature

A member's signature of one version of a family's waiver. A member can have several signatures for one family, one per version signed; the latest is the one with the highest version.

**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `familyId`: identifier, naming an existing [[equipment/family]]
- `version`: count, at least 1 (the version of the waiver that was signed)
- `signedAt`: timestamp
