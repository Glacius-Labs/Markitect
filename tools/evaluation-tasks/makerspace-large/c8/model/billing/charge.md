---
kind: std/Concept
---
# Charge

A charge (invoice line) on a member's account. Its amount follows [[money]] and is fixed when the charge is created.

**Fields**, in the formats of [[values]]:

- `kind`: one of `membership`, `usage`, `material`, `course`, `deposit`, `pack`, `locker`, `guest`, `lateFee`, `event` or `noShow`
- `ref`: string (what a charge is for: the member's tier, a reservation that was used or missed, a purchase, a course, a deposit, a pack, a rental, a guest pass, an invoice, a loan or a ticket identifier)
- `quantity`: count
- `unitPriceCents`: cent amount
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp
