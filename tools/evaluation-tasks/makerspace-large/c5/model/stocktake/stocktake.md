---
kind: std/Concept
---
# Stocktake

A count of the shop's stock, generated identifier `stk-N`. It is `open` until it is closed. At most one stocktake is open at a time.

**Fields**, in the formats of [[values]]:

- `stocktakeId`: identifier
- `status`: `open` or `closed`
- `counts`: a list of [[stocktake/count]] records, in ascending `materialId` order; empty for a new stocktake

A record is identified by `stocktakeId`.
