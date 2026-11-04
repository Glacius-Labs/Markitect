---
kind: Operation
---
# List dunning cases

Returns all dunning cases of the member's invoices, in creation order (ascending case number). A member without cases gives an empty list.

- **Wire name:** `listDunningCases`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `cases`: a list of [[dunning/dunning-case]] records
- **Permissions:** member own, staff all, admin all.
