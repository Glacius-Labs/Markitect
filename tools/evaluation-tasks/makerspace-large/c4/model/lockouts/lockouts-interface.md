---
kind: Interface
---
# Lockouts interface

What other modules may use of lockouts: its operations, the lockout record and its source, and the rule that a locked-out machine cannot be reserved or checked in. Other modules also ask the capabilities listed below. Lockouts needs nothing from other modules apart from equipment.

- **Operations:** [[lockouts/apply-lockout]], [[lockouts/release-lockout]], [[lockouts/list-lockouts]].
- **Uses:** [[equipment/equipment-interface]].
- **Records:** [[lockouts/lockout]], [[lockouts/lockout-source]].
- **Rules that bind other modules:** [[lockouts/lockout-gate]].

Capabilities other modules ask of lockouts. None of them writes an audit entry of its own; the operation that uses it is audited once.

- Whether a machine is locked out at the request time, which is whether it has an active lockout ([[lockouts/lockouts]]). Bookings and compliance ask this.
- Whether a lockout of a machine, of any source, overlaps an interval `[start, end)`, taking the lockout as the interval `[appliedAt, releasedAt)`. A lockout that is not released overlaps when its `appliedAt` is before `end`. Bookings asks this.
- Whether a machine has an active lockout of a given source. Inspections asks this for source `inspection`.
- Applying a lockout to a machine with any source, a reason and the user who applies it. It gets the next lockout identifier, `appliedAt` the request time and `releasedAt` `null`, and is returned. Inspections and workorders use this.
- Releasing one lockout by its identifier, whatever its source, and releasing every active lockout of one source on one machine. A release sets `releasedAt` to the request time. A lockout that is already released is left as it is. Inspections and workorders use this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `lockout_not_manual` | The lockout was not applied by hand, so it cannot be released by hand. |
| `machine_locked_out` | The machine has an active lockout at the request time. |
| `not_lockout_holder` | The actor did not apply the lockout and is not an admin. |
