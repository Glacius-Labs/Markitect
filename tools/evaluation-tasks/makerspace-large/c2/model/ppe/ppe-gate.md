---
kind: std/Rule
---
# PPE gate

A member **holds** an item from [[ppe/issue-ppe]] until [[ppe/return-ppe]]. A member holds at most one unit of an item. Standing does not change what a member holds.

**The gate.** At `checkIn`, a member who does not hold every item that the machine's family requires is refused `ppe_missing`, at the place that operation lists. A family with no requirement never causes it. Reservations are not gated by it.
