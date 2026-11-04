---
kind: Operation
---
# Renew loan

The loan's `dueOn` becomes the date 14 days after its current `dueOn` and its `renewals` rises by 1. Nothing is charged. No shift is needed, and certification, consent and waiver are not judged again. The tool's `onLoan` stays `true`.

Example: a loan with `dueOn` `2026-03-16` renewed at `2026-03-16T23:59Z` has `dueOn` `2026-03-30`. The same request at `2026-03-17T00:00Z` is refused with `loan_overdue` ([[loans/overdue-loans]]).

Rules of other modules that apply here: [[members/membership]].

- **Wire name:** `renewLoan`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `loanId`: identifier, naming an existing [[loans/loan]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the loan's status is `returned`.
  2. `member_suspended` when the loan's member is not in good standing ([[members/members-interface]]).
  3. `loan_overdue` when this loan is overdue on the date of the request time ([[loans/overdue-loans]]). Another overdue loan of the member does not matter here.
  4. `renewal_limit` when `renewals` is 2.
- **Result:**
  - `loan`: a [[loans/loan]] record, after the change
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `loanId`.
