---
kind: std/Concept
---
# Charge

A charge (invoice line) on a member's account. Its amount follows [[money]] and is fixed when the charge is created.

**Fields**, in the formats of [[values]]:

- `kind`: one of `membership`, `usage`, `material` or `course`
- `ref`: string (what a charge is for: the member's tier, a reservation, a purchase or a course identifier)
- `quantity`: count
- `unitPriceCents`: cent amount
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp
