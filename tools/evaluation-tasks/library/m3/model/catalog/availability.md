---
kind: std/Rule
---
# Availability

A registered [[catalog/copy]] is available exactly when it has no open loan. Across all members, at most one open loan may refer to the same copy.

Lending owns loans and must keep this rule: borrowing must enforce availability and makes the copy unavailable by creating an open loan; returning releases the copy only when it closes that copy's current open loan. Availability is derived from loans; there is no independent stock count. Returning an old closed loan cannot release a copy now held by a newer open loan.
