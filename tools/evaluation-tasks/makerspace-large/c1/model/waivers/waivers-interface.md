---
kind: Interface
---
# Waivers interface

What other modules may use of waivers: its operations, the waiver and signature records and the refusal codes below. Other modules also ask whether a family has a waiver, and whether a member has signed the current version of a family's waiver. The rule is [[waivers/waiver-gate]].

- **Operations:** [[waivers/publish-waiver]], [[waivers/get-waiver]], [[waivers/sign-waiver]], [[waivers/list-member-waivers]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[guardians/guardians-interface]].
- **Records:** [[waivers/waiver]], [[waivers/waiver-signature]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_signed` | The member has already signed the current version of the family's waiver. |
| `guardian_required` | The member is a minor and no guardian has consented for the family. |
| `no_waiver` | The family has no waiver. |
| `waiver_missing` | The member has not signed the current version of the family's waiver. |
