---
kind: Operation
---
# Charge overdue loans

Takes every loan that is overdue on the date of the request time ([[loans/overdue-loans]]), in ascending loan number: the number `N` of `lon-N`, compared as a number. For each one whose days overdue are more than its `chargedDays`, it creates a late-fee charge ([[loans/late-fee-charge]]), and when `chargedDays` was 0 before this charge it also posts a notice of kind `loan` with `ref` the loan identifier for the loan's member. Standing is not checked. `charges` lists the charges created; it is empty when nothing is due.

Rules of other modules that apply here: [[notices/posting-notices]] and [[billing/charges-and-invoices]].

- **Wire name:** `chargeOverdueLoans`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `charges`: a list of [[billing/charge]] records
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `overdueLoans`.
