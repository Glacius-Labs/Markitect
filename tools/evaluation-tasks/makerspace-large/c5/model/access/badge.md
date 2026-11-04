---
kind: std/Concept
---
# Badge

A member's door badge, generated identifier `bdg-N`. A member has at most one active badge. A badge is **inside** from [[access/enter-door]] until [[access/leave-door]] or [[access/revoke-badge]]; it starts outside. A revoked badge is never inside and never active again.

**Fields**, in the formats of [[values]]:

- `badgeId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `status`: `active` or `revoked`
- `inside`: flag

A record is identified by `badgeId`.
