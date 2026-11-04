---
kind: std/Concept
---
# Charge kind row

One kind's row of the charges-by-kind report.

**Fields**, in the formats of [[values]]:

- `kind`: one of `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event` or `noShow`
- `invoicedCents`: cent amount (the sum of the line amounts of this kind on the invoices of the month)
- `refundedCents`: cent amount (the sum of the refunds of those lines)
