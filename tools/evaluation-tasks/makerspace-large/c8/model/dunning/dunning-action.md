---
kind: std/Concept
---
# Dunning action

One step taken by a run for one invoice.

**Fields**, in the formats of [[values]]:

- `invoiceId`: identifier, naming an existing [[billing/invoice]]
- `step`: one of `notice`, `fee` or `suspend`
