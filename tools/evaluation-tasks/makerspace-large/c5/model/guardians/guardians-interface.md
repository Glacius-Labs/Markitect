---
kind: Interface
---
# Guardians interface

What other modules may use of guardians: its operations, the guardianship record and the refusal codes below. Other modules also ask whether a member is a minor on a date, whether a member holds the consent of a guardian for a family, and whether a member is refused `guardian_consent_missing` for a family on a date. The rule is [[guardians/guardian-consent]].

- **Operations:** [[guardians/declare-minor]], [[guardians/grant-consent]], [[guardians/withdraw-consent]], [[guardians/get-guardianship]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]].
- **Records:** [[guardians/guardianship]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `guardian_consent_missing` | The member is a minor on the relevant date and no guardian has consented for the family. |
| `invalid_guardian` | The guardian is the member, or is a minor. |
| `no_consent` | No consent is held for the family. |
| `not_a_declared_minor` | The member has no guardianship record. |
| `not_a_minor` | The member is not a minor on the date of the request time. |
| `not_guardian` | The named guardian is not the member's guardian. |
