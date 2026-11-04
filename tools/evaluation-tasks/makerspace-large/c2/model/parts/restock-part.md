---
kind: Operation
---
# Restock part

The stock increases by `quantity`.

- **Wire name:** `restockPart`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `partId`: identifier, naming an existing [[parts/part]]
  - `quantity`: integer from `1` to `100000` (the number of units added to stock in one request)
- **Result:**
  - `part`: a [[parts/part]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `partId`.
