---
kind: std/Concept
---
# Family compliance

How one member stands towards one tool family at the request time. [[compliance/compliance-summary]] returns one for every family.

**Fields**, in the formats of [[values]]:

- `familyId`: identifier, naming an existing [[equipment/family]]
- `certified`: flag, `true` when the member holds a certification for the family that is valid on the date of the request time ([[training/certifications]])
- `consent`: flag, `true` when the member is not a minor on the date of the request time, or holds consent for the family ([[guardians/guardian-consent]])
- `waiverSigned`: flag, `true` when the family has no waiver, or the member's latest signature for the family is of the current version and has not lapsed on the date of the request time ([[waivers/waiver-gate]])
- `ppeHeld`: flag, `true` when the member holds every item the family requires, which includes a family that requires none ([[ppe/ppe-gate]])

A record is identified by `familyId` within one summary.
