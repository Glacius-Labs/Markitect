---
kind: Operation
---
# Run dunning

Takes the steps of the ladder that are due for the invoices overdue at the request time, as [[dunning/dunning-run|the dunning run rule]] states, with all effects together. `actions` lists the steps taken, in the order they were taken. It is empty when nothing is due.

Rules of other modules that apply here: [[dunning/dunning-run]], [[billing/overdue-invoices]], [[billing/charges-and-invoices]] and [[members/membership]].

- **Wire name:** `runDunning`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Refusals** at check 7, in this order:
  1. `no_ladder` when no ladder has been set.
- **Result:**
  - `actions`: a list of [[dunning/dunning-action]] records
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `dunning`.
