---
kind: std/Rule
---
# Tests

The tests run with `go test ./...` from `realization/`. They cover, for every operation, a successful request and each refusal code the operation lists; the four policies, including the permission matrix and audit entries; the boundary rules of [[time]] (half-open intervals, opening hours, granularity and duration limits) and of [[money]] (rounding per line); and the cross-area rules of reservations ([[bookings]]) and invoices ([[billing]]). Tests and documentation do not expect or describe an expiry of certifications; a certification issued long before a reservation date is still accepted.
