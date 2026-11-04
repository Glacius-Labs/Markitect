---
kind: std/Concept
---
# Deposit

A sum a member leaves with the workshop, generated identifier `dep-N`. A deposit is `held` from the moment it is taken. Releasing it turns it into credit for the member, so its status becomes `released`; forfeiting it keeps the sum for the workshop, so its status becomes `forfeited`. `closedAt` is `null` while the deposit is held.

**Fields**, in the formats of [[values]]:

- `depositId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `purpose`: text
- `amountCents`: integer from `1` to `100000000` cents (a deposit amount)
- `status`: one of `held`, `released` or `forfeited`
- `takenAt`: timestamp
- `closedAt`: timestamp, or `null`

A record is identified by `depositId`.
