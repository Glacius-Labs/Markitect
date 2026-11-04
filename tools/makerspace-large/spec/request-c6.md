# Change request: Renewing a tool loan

Members want to keep a tool a little longer and have to come to the desk just to say so. Let them renew a loan themselves, twice at most.

The loan record gets a new field, `renewals`: count, the number of times the loan has been renewed, 0 when the loan is made.

Renewing moves `dueOn` to the date 14 days after its current value and raises `renewals` by 1. Nothing is charged. No shift is needed, and certification, consent and waiver are not judged again. Ownership is judged on `loanId`: a loan belongs to the actor when its member is the actor.

Example: a loan with `dueOn` `2026-03-16` renewed at `2026-03-16T23:59Z` has `dueOn` `2026-03-30`. The same request at `2026-03-17T00:00Z` is refused with `loan_overdue`.

- **Wire name:** `renewLoan`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `loanId`: identifier, naming an existing loan
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the loan's status is `returned`.
  2. `member_suspended` when the loan's member is not in good standing (members interface).
  3. `loan_overdue` when this loan is overdue on the date of the request time. Another overdue loan of the member does not matter here.
  4. `renewal_limit` when `renewals` is 2.
- **Result:**
  - `loan`: a loan record, after the change
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `loanId`.

New refusal code: `renewal_limit`, "The loan has been renewed twice."
