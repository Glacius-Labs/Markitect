---
kind: Operation
---
# Add customer

Creates the [[quotes/customer]].

- **Wire name:** `addCustomer`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `customerId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a customer with `customerId` exists.
- **Result:**
  - `customer`: a [[quotes/customer]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `customerId`.
