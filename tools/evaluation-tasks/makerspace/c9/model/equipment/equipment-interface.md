---
kind: Interface
---
# Equipment interface

What other modules may use of equipment: its operations, the family, machine and maintenance window records, and the machine rate type. Other modules also ask for a machine's family and current rate and for the scheduled windows of a machine that overlap an interval. Equipment in turn needs to know whether an active reservation of a machine overlaps an interval; it declares that need, and the protocol module supplies the answer from bookings.

- **Operations:** [[equipment/add-family]], [[equipment/add-machine]], [[equipment/set-machine-rate]], [[equipment/list-machines]], [[equipment/schedule-maintenance]], [[equipment/cancel-maintenance]].
- **Records:** [[equipment/family]], [[equipment/machine]], [[equipment/maintenance-window]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `reservation_conflict` | An active reservation overlaps the maintenance window. |
