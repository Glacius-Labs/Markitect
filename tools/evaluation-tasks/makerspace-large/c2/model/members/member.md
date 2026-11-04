---
kind: std/Concept
---
# Member

A user with role `member` and a membership.

**Fields**, in the formats of [[values]]:

- `memberId`: identifier
- `name`: text
- `tier`: a [[members/tier]]: `basic`, `plus` or `pro`
- `standing`: `good` or `suspended`
- `registeredAt`: timestamp

A record is identified by `memberId`.
