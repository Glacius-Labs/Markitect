---
kind: std/Concept
---
# Guest pass

The permission for one guest to visit with a member on one date, generated identifier `gst-N`.

**Fields**, in the formats of [[values]]:

- `guestPassId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `guestName`: text
- `date`: date
- `charged`: flag (true when a fee was charged for the pass)
- `status`: `issued` or `revoked`

A record is identified by `guestPassId`.
