# Change request: A tool library

We have a shelf of portable power tools that members keep asking to take home for a fortnight. I want the program to run this as a new area, the tool library. Every tool belongs to a tool family, because the rules for using a family are the same for its tools as for its machines. Staff hand tools out and take them back at the desk, and someone must actually be on duty to hand one out. A late tool costs money per day, and a member who keeps one too long cannot use the machines until it is back.

This is a new module, `loans`. It may depend on `members`, `equipment`, `training`, `guardians`, `waivers`, `staffing`, `billing`, `notices` and `policies`. `bookings` may now also depend on `loans`, and `protocol` depends on it as on every module. Both stay one-way: `loans` does not depend on `bookings`, and none of the modules `loans` uses depends on `loans`.

**Tool.** A tool that can be lent out. A record is identified by `toolId`.

- `toolId`: identifier
- `name`: text
- `familyId`: identifier, naming an existing family
- `lateFeeCentsPerDay`: integer from `1` to `100000` cents (the fee for one day overdue)
- `onLoan`: flag, `true` while the tool has an active loan

**Loan.** The loan of one tool to one member, generated identifier `lon-N` with its own counter. A record is identified by `loanId`. `returnedAt` is `null` while the loan is active.

- `loanId`: identifier
- `toolId`: identifier, naming an existing tool
- `memberId`: identifier, naming an existing member
- `borrowedAt`: timestamp
- `dueOn`: date, 14 days after the date of `borrowedAt`
- `returnedAt`: timestamp, or `null`
- `status`: `active` or `returned`
- `overdue`: flag, reported at the request time of the response
- `chargedDays`: count, the days of late fee charged so far, 0 at the start

**Overdue.** A loan is **overdue on date D** when its status is `active` and D is after its `dueOn`. On `dueOn` itself it is not overdue. Its **days overdue** are the number of calendar days from `dueOn` to the date of the request time.

**The late-fee charge.** A charge for the member of the loan: `kind` `lateFee`, `ref` the loan identifier, `quantity` the days overdue less `chargedDays`, `unitPriceCents` the tool's `lateFeeCentsPerDay`, `discountPercent` 0, `at` the request time. `chargedDays` then becomes the days overdue.

Example: a loan with `borrowedAt` `2026-03-02T10:00Z` of a tool with a fee of 200 cents has `dueOn` `2026-03-16`. It is not overdue on `2026-03-16`. A run on `2026-03-19` charges 3 days, 600 cents. A run on `2026-03-20` charges 1 day, 200 cents. A return on `2026-03-22` charges the remaining 2 days, 400 cents.

**A new notice kind.** `loan`: a tool loan is overdue, `ref` is the loan identifier, subject `Tool loan <ref> is overdue`.

**The refusal codes of this module:**

| Code | Meaning |
| --- | --- |
| `loan_limit` | The member already has 3 active loans. |
| `loan_overdue` | The member has an overdue tool loan. |
| `tool_on_loan` | The tool has an active loan. |

**Check in.** `checkIn` gets a new last refusal, number 10: `loan_overdue` when the member has a loan that is overdue on the date of the request time. `createReservation` and `checkOut` are not gated by it.

**List order.** `listTools` → `tools`: `toolId` ascending. `listMemberLoans` → `loans`: creation order. `chargeOverdueLoans` → `charges`: ascending loan number, the number `N` of `lon-N` compared as a number.

**Add tool.** Creates the tool with `onLoan` `false`.

- **Wire name:** `addTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `toolId`: identifier
  - `name`: text
  - `familyId`: identifier, naming an existing family
  - `lateFeeCentsPerDay`: integer from `1` to `100000` cents
- **Refusals** at check 7, in this order:
  1. `already_exists` when a tool with `toolId` exists.
- **Result:**
  - `tool`: a tool record
- **Permissions:** staff all, admin all.
- **Audit target:** `toolId`.

**List tools.** Returns all tools.

- **Wire name:** `listTools`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `tools`: a list of tool records
- **Permissions:** member all, staff all, admin all.

**Borrow tool.** Creates the loan with the next loan identifier, status `active`, `borrowedAt` the request time, `dueOn` as above, `returnedAt` `null` and `chargedDays` 0. The tool's `onLoan` becomes `true`. The actor is on shift as decide volunteer time states it; an admin does not need a shift. The other rules for machines, such as PPE, lockouts, inspections and maintenance, do not apply to tools.

- **Wire name:** `borrowTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `toolId`: identifier, naming an existing tool
- **Refusals** at check 7, in this order:
  1. `approver_not_on_shift` when the actor is staff and is not on shift.
  2. `member_suspended` when the member is not in good standing (members interface).
  3. `guardian_consent_missing` when the member is a minor on the date of the request time and holds no consent for the tool's family.
  4. `not_certified` when the member has no certification for the tool's family valid on the date of the request time.
  5. `waiver_missing` when the waiver gate refuses the member for the tool's family.
  6. `tool_on_loan` when the tool has an active loan.
  7. `loan_overdue` when the member has a loan that is overdue on the date of the request time.
  8. `loan_limit` when the member has 3 active loans.
- **Result:**
  - `loan`: a loan record
  - `tool`: a tool record, after the change
- **Permissions:** staff all, admin all.
- **Audit target:** the new `loanId`.

**Return tool.** The loan gets status `returned` and `returnedAt` the request time. The tool's `onLoan` becomes `false`. When the loan is overdue on the date of the request time and its days overdue are more than its `chargedDays`, a late-fee charge is created. `charge` is that charge, otherwise `null`. Standing is not checked, no shift is needed and no notice is posted.

- **Wire name:** `returnTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `loanId`: identifier, naming an existing loan
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the loan's status is `returned`.
- **Result:**
  - `loan`: a loan record, after the change
  - `tool`: a tool record, after the change
  - `charge`: a charge record, or `null`
- **Permissions:** staff all, admin all.
- **Audit target:** `loanId`.

**List member loans.** Returns all loans of the member, active and returned, in creation order.

- **Wire name:** `listMemberLoans`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `loans`: a list of loan records
- **Permissions:** member own, staff all, admin all.

**Charge overdue loans.** Takes every loan that is overdue on the date of the request time, in ascending loan number. For each one whose days overdue are more than its `chargedDays`, it creates a late-fee charge, and when `chargedDays` was 0 before this charge it also posts a notice of kind `loan` with `ref` the loan identifier for the loan's member (posting notices). Standing is not checked.

- **Wire name:** `chargeOverdueLoans`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `charges`: a list of charge records, the charges created; empty when nothing is due
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `overdueLoans`.
