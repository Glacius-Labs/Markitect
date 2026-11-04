---
kind: std/Concept
---
# Quote line

One line of a [[quotes/quote]]: machine time or a material. The same `ref` can be on several lines. `unitPriceCents` is `null` until the quote is sent and is fixed by sending, as [[quotes/quote-pricing]] states.

**Fields**, in the formats of [[values]]:

- `lineKind`: `machine` or `material`
- `ref`: identifier; for a `machine` line the identifier of an existing [[equipment/machine]], for a `material` line the identifier of an existing [[materials/material]]
- `quantity`: integer from `1` to `100000`; for a `machine` line the minutes of machine time, for a `material` line the units
- `unitPriceCents`: integer from `1` to `1000000` cents, or `null`; for a `machine` line the machine's rate in cents per minute when the quote was sent, for a `material` line the material's unit price when the quote was sent
