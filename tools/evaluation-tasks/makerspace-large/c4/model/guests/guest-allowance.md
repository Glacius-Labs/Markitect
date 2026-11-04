---
kind: std/Concept
---
# Guest allowance

How many guest passes a month a tier includes, and the fee for each further one. A tier has an allowance only after [[guests/set-guest-allowance]] has set one; a tier without one allows no guests.

**Fields**, in the formats of [[values]]:

- `tier`: a [[members/tier]]: `basic`, `plus` or `pro`
- `perMonth`: integer from `0` to `100` (the guest passes of one month that are free)
- `feeCents`: integer from `0` to `1000000` cents (the fee of a guest pass beyond the free ones)

A record is identified by `tier`.
