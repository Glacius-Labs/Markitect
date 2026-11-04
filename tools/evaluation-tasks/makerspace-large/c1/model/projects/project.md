---
kind: std/Concept
---
# Project

A project of a member, who owns it, with collaborators. Reservations and purchases of its members are attached to it so that their cost can be read together. The identifier is chosen by the caller.

**Fields**, in the formats of [[values]]:

- `projectId`: identifier
- `ownerId`: identifier, naming an existing [[members/member]]
- `title`: text
- `collaboratorIds`: a list of identifiers, each naming an existing [[members/member]], in ascending order; the owner is not in it
- `status`: `open` or `closed`
- `reservationIds`: a list of identifiers, each naming an existing [[bookings/reservation]], in the order they were attached
- `purchaseIds`: a list of identifiers, each naming an existing [[materials/purchase]], in the order they were attached

A record is identified by `projectId`.
