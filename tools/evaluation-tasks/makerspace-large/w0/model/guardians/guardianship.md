---
kind: std/Concept
---
# Guardianship

The declaration that a member is a minor, with the guardian and the families the guardian consented to. `consentFamilyIds` is empty until a consent is granted.

**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `birthDate`: date
- `guardianId`: identifier, naming an existing [[members/member]]
- `consentFamilyIds`: a list of identifiers, each naming an existing [[equipment/family]], ascending

A record is identified by `memberId`. A guardianship is never removed and its birth date and guardian never change.
