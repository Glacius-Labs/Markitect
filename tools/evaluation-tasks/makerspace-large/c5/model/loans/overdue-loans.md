---
kind: std/Rule
---
# Overdue loans

A [[loans/loan]] is **overdue on date D** when its status is `active` and D is after its `dueOn`. On `dueOn` itself it is not overdue, and a returned loan is never overdue. The `overdue` field reports this for the date of the request time of the response. The **days overdue** of a loan are the number of calendar days from `dueOn` to the date of the request time.

A member with a loan that is overdue on a date may not borrow another tool ([[loans/borrow-tool]]) and may not check in to a machine ([[bookings/check-in]]). Reservations and check-out are not gated by it. Operations of other modules that judge it say so on their own pages.

Example: a loan borrowed at `2026-03-02T10:00Z` has `dueOn` `2026-03-16`. It is not overdue on `2026-03-16`. It is overdue on `2026-03-17`, when its days overdue are 1, and on `2026-03-19`, when they are 3. Once it is returned it is no longer overdue.
