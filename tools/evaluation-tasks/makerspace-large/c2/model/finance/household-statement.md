---
kind: Operation
---
# Household statement

Shows the month's invoices of the members of a household. `rows` has one row for each member who belongs to the household at the request time, in ascending `memberId` order. A member's invoice is the invoice generated for that member for the month; a member has at most one. A member without one has `invoiceId` `null`, `totalCents` 0 and `balanceCents` 0. `totalCents` and `balanceCents` of the result are the sums over the rows.

- **Wire name:** `householdStatement`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `householdId`: identifier, naming an existing [[households/household]]
  - `month`: month
- **Result:**
  - `householdId`: identifier
  - `month`: month
  - `rows`: a list of [[finance/household-statement-row]] records
  - `totalCents`: cent amount
  - `balanceCents`: cent amount
- **Permissions:** staff all, admin all.
