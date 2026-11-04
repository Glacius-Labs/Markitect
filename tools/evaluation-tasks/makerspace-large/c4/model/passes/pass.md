---
kind: std/Concept
---
# Pass

A pass of one member, sold from a [[passes/pass-product]], generated identifier `pas-N`.

**Fields**, in the formats of [[values]]:

- `passId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `productId`: identifier, naming an existing [[passes/pass-product]]
- `minutesLeft`: minute count (starts as the product's `minutes`; [[passes/pass-draw|the pass draw rule]] lowers it)
- `issuedOn`: date (the date of the request time of the sale)
- `expiresOn`: date (`validDays` days after `issuedOn`)

A record is identified by `passId`.
