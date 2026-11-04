---
kind: std/Concept
---
# Payout

The pay owed to one instructor for the classes of one month, generated identifier `pyt-N`.

**Fields**, in the formats of [[values]]:

- `payoutId`: identifier
- `userId`: identifier, naming an existing [[members/user]]
- `month`: month
- `minutes`: minute count
- `amountCents`: cent amount
- `status`: `open` or `settled`

A record is identified by `payoutId`.
