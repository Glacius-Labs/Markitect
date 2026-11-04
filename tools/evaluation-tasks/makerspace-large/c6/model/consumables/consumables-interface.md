---
kind: Interface
---
# Consumables interface

What other modules may use of consumables: its operations, the consumption row record, and the rule that a session's consumables are drawn at check-out. Other modules also ask the capability listed below. Consumables adds no refusal codes beyond the one in the table.

- **Operations:** [[consumables/set-consumption]], [[consumables/clear-consumption]], [[consumables/list-consumption]].
- **Uses:** [[equipment/equipment-interface]], [[materials/materials-interface]], [[billing/billing-interface]].
- **Records:** [[consumables/consumption-row]].
- **Rules that bind other modules:** [[consumables/consumption]].

Capability other modules ask of consumables:

- The draw for one ended session, which bookings asks at check-out. It is given the reservation identifier, the member, the machine, the metered minutes and the request time, and it does what [[consumables/consumption]] states: it decreases material stock and creates the material charges, in that rule's order. It returns the charges it created, which may be none.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `no_consumption` | The machine has no consumption row for the material. |
