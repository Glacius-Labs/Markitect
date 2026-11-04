---
kind: std/Concept
---
# Pack

A pack of one member, sold from a [[packs/pack-product]], generated identifier `pak-N`.

**Fields**, in the formats of [[values]]:

- `packId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `productId`: identifier, naming an existing [[packs/pack-product]]
- `minutesLeft`: minute count (starts as the product's `minutes`; [[packs/pack-draw|the pack draw rule]] lowers it)
- `issuedOn`: date (the date of the request time of the sale)
- `expiresOn`: date (`validDays` days after `issuedOn`)

A record is identified by `packId`.
