---
kind: Operation
---
# Borrow tool

Creates the loan with the next loan identifier, status `active`, `borrowedAt` the request time, `dueOn` 14 days after the date of `borrowedAt`, `returnedAt` `null`, `chargedDays` 0 and `renewals` 0. The tool's `onLoan` becomes `true`. The other rules for machines, such as PPE, lockouts, inspections and maintenance, do not apply to tools.

The actor is on shift when the actor is staff and has a scheduled [[staffing/shift]] of their own with `start` at or before the request time and `end` after it. An admin does not need a shift.

Rules of other modules that apply here: [[members/membership]], [[guardians/guardian-consent]], [[waivers/waiver-gate]] and [[training/certifications]].

- **Wire name:** `borrowTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `toolId`: identifier, naming an existing [[loans/tool]]
- **Refusals** at check 7, in this order:
  1. `approver_not_on_shift` when the actor is staff and is not on shift.
  2. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  3. `guardian_consent_missing` when the member is a minor on the date of the request time and holds no consent for the tool's family ([[guardians/guardian-consent]]).
  4. `not_certified` when the member has no certification for the tool's family valid on the date of the request time ([[training/training-interface]]).
  5. `waiver_missing` when the [[waivers/waiver-gate]] refuses the member for the tool's family.
  6. `tool_on_loan` when the tool has an active loan.
  7. `loan_overdue` when the member has a loan that is overdue on the date of the request time ([[loans/overdue-loans]]).
  8. `loan_limit` when the member has 3 active loans.
- **Result:**
  - `loan`: a [[loans/loan]] record
  - `tool`: a [[loans/tool]] record, after the change
- **Permissions:** staff all, admin all.
- **Audit target:** the new `loanId`.
