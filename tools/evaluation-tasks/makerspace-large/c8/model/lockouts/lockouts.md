---
kind: std/Rule
---
# Lockouts

A [[lockouts/lockout]] takes a machine out of use until it is released. A lockout is **active** while its `releasedAt` is `null`. A machine can have several active lockouts at once; it is **locked out** while it has at least one. Releasing one lockout leaves the others in place.

A lockout has one of three sources ([[lockouts/lockout-source]]). A `manual` lockout is applied and released by staff or an admin with the operations of this module. An `inspection` lockout is applied and released by the inspections module, and a `workOrder` lockout by the workorders module; neither can be released with [[lockouts/release-lockout]].

Lockouts are never removed. A released lockout stays on record with `releasedAt` set to the request time of its release, and `releasedAt` does not change again.
