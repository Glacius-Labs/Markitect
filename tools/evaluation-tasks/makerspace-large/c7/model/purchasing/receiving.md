---
kind: std/Rule
---
# Receiving

A receipt is made against one line of an approved [[purchasing/order]], for part or all of what is still open on the line.

- The line's `receivedQuantity` increases by the received quantity. It never exceeds the line's `quantity`.
- The material's stock increases by the received quantity. The material's unit price does not change; the line's `unitCostCents` is what the shop pays, not what it sells for.
- The order stays `approved` while any line has `receivedQuantity` below `quantity`. When every line has `receivedQuantity` equal to `quantity`, the status becomes `received`, and no further receipt is accepted.
- Ledger postings are made as [[ledger/postings]] states: the amount is the received quantity times the line's `unitCostCents`.

A material that is frozen by a stocktake cannot be received. The supplier's status is not judged at a receipt.

Example: a line of 100 units at 120 cents. A receipt of 40 gives `receivedQuantity` 40, adds 40 to the stock and posts 4800 cents. A receipt of 61 is then refused with `over_receipt`, since 40 + 61 is above 100. A receipt of 60 completes the line.
