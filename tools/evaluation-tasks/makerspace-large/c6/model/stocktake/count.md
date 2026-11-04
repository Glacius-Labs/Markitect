---
kind: std/Concept
---
# Count

The count of one material in one [[stocktake/stocktake]]: how many units were found on the shelf. A stocktake holds at most one count per material. `difference` is `null` until the stocktake is closed; closing sets it, and it stays.

**Fields**, in the formats of [[values]]:

- `materialId`: identifier, naming an existing [[materials/material]]
- `counted`: integer from `0` to `100000000` (the number of units counted)
- `difference`: integer, or `null`. It is `counted` minus the material's stock at closing, so it is negative when units are missing and may be 0.
