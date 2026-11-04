---
kind: std/Concept
---
# Charge

A charge (invoice line) on a member's account. Its amount follows [[money]] and is fixed when the charge is created.

**Fields**, in the formats of [[values]]:

- `kind`: one of `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee` or `event`
- `ref`: string (what a charge is for: the member's tier, a reservation, a purchase, a course, a deposit, a pass, a rental, a guest pass, an invoice or a ticket identifier)
- `quantity`: count
- `unitPriceCents`: cent amount
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp
