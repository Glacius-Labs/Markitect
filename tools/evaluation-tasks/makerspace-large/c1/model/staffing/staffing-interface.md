---
kind: Interface
---
# Staffing interface

What other modules may use of staffing: its operations, the shift and supervision records and the refusal codes below. Other modules also ask whether an interval is covered by scheduled shifts, whether a family is supervised, and whether a user has a scheduled shift that contains a point in time. The rule is [[staffing/supervision]]. Staffing in turn needs the active reservations of all machines of a family, each with its machine, `start` and `end`; it declares that need, and the protocol module supplies the answer from bookings.

- **Operations:** [[staffing/add-shift]], [[staffing/cancel-shift]], [[staffing/list-shifts]], [[staffing/set-supervision]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]].
- **Records:** [[staffing/shift]], [[staffing/supervision-setting]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `not_staff` | The user is a member. |
| `shift_needed` | Cancelling the shift would leave an active reservation of a supervised family without cover. |
| `shift_overlap` | Another scheduled shift of the user overlaps the shift. |
| `shift_too_long` | The shift is longer than 720 minutes. |
| `supervisor_required` | The machine's family is supervised and the slot is not covered by scheduled shifts. |
