---
kind: std/Concept
---
# Rental

The rental of one locker by one member, generated identifier `rnt-N`. A locker has at most one active rental. `endedAt` is `null` while the rental is active.

**Fields**, in the formats of [[values]]:

- `rentalId`: identifier
- `lockerId`: identifier, naming an existing [[lockers/locker]]
- `memberId`: identifier, naming an existing [[members/member]]
- `status`: `active` or `ended`
- `startedAt`: timestamp
- `endedAt`: timestamp, or `null`

A record is identified by `rentalId`.
