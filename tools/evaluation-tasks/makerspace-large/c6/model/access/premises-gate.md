---
kind: std/Rule
---
# Premises gate

A member **is on the premises** when the member's active badge is inside. A member with no active badge is not gated: the gate never applies to such a member.

**The gate.** At [[bookings/check-in]], a member who has an active badge that is not inside is refused `not_on_premises`, at the place that operation lists. Check-out is never gated.
