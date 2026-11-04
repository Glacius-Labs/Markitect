---
kind: Operation
---
# List payouts

Returns the payouts of the user in creation order. A user with no payouts gets an empty list.

- **Wire name:** `listPayouts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier, naming an existing [[members/user]]
- **Result:**
  - `userId`: identifier
  - `payouts`: a list of [[payouts/payout]] records
- **Permissions:** staff all, admin all.
