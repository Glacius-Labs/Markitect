---
kind: std/Rule
---
# Guardian consent

A member is a **minor on date D** when the member has a [[guardians/guardianship]] and D is before the 18th anniversary of its `birthDate`. A member with no guardianship is an adult. The 18th anniversary is the same month and day 18 years later; for a birth date of 29 February it is 1 March, because the year 18 years later has no 29 February. A birth date after D is not an error: the member is a minor on D.

Example: with birth date `2008-05-20` the member is a minor on `2026-05-19` and an adult on `2026-05-20`. With birth date `2008-02-29` the member is a minor on `2026-02-28` and an adult on `2026-03-01`.

A guardian's **consent** for a family is held when the family is in `consentFamilyIds`. It stays until it is withdrawn, and it does not depend on the guardian's later state.

**The consent gate.** A member who is a minor on the date D of a request and holds no consent for the relevant family is refused `guardian_consent_missing`. A member who is not a minor on D is never refused. The date D and the family are:

| Operation | Family | D |
| --- | --- | --- |
| `createReservation` | the machine's | the date of `start` |
| `checkIn` | the machine's | the date of the request time |
| `enrollInCourse` | the course's | the date of the request time |
