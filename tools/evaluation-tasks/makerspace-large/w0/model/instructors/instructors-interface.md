---
kind: Interface
---
# Instructors interface

What other modules may use of instructors: its operations and the qualification record. Other modules also ask whether a user holds a qualification for a family and whether a user holds a qualification for any family. Instructors in turn needs to know whether a user is the instructor of a scheduled class of a family that starts after a given time; it declares that need, and the protocol module supplies the answer from classes.

- **Operations:** [[instructors/qualify-instructor]], [[instructors/withdraw-instructor]], [[instructors/list-instructors]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[training/training-interface]].
- **Records:** [[instructors/qualification]].

What other modules ask of instructors that is not an operation:

- Whether a user holds a qualification for a given family: yes when a [[instructors/qualification]] with that `userId` and `familyId` exists.
- Whether a user holds a qualification for any family: yes when a qualification with that `userId` exists.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `instructor_has_classes` | The user is the instructor of a scheduled class of the family that starts after the request time. |
| `not_eligible` | The user may not be qualified for the family: an admin, or a member with no certification valid on the request date. |
| `not_instructor` | The user holds no qualification for the family, or, where an operation says so, for any family. |
