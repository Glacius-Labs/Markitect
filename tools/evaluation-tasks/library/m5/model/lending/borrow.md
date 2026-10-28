---
kind: std/Feature
---
# Borrow

The operator supplies a [[members/member]] identifier, a [[catalog/copy]] identifier and a borrowing date under [[inputs]] in the [[project|library]].

Admit the request only when the member is registered under [[members/eligibility]], the copy is registered and available under [[catalog/availability]], the resulting open-loan count satisfies [[lending/loan-policy]], and the new loan's due date falls on or before 9999-12-31, the last date of the library's date range.

Success creates exactly one open [[lending/loan]], returns its identifier and due date, makes the copy unavailable and increases the member's open-loan count by one. Other members and copies remain unchanged.

Unknown members or copies, an already borrowed copy, a reached limit, a due date after 9999-12-31 or invalid inputs cause refusal with no loan created and no observable state change. A due date is never clamped, wrapped or given a year after 9999. Repeating a successful request for the same copy is refused because it is now borrowed; this is a new transaction, not a retry-safe command with duplicate-request detection.

It is reached through the operation `borrow` of the [[callable-interface]].
