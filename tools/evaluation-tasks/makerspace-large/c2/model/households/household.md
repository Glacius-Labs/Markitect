---
kind: std/Concept
---
# Household

Members who share a home and one payer, as [[households/household-membership]] states.

**Fields**, in the formats of [[values]]:

- `householdId`: identifier
- `name`: text
- `payerId`: identifier, naming an existing [[members/member]]
- `memberIds`: a list of identifiers, each naming an existing [[members/member]], ascending; it always contains `payerId`

A record is identified by `householdId`.
