# The large Makerspace: its change steps

The eight change steps of [test 2](../../docs/concepts/v3-concept.md#13-how-we-know-it-works). This page is assessor material: no variant receives it. A variant working from the repository receives only the request (`spec/request-cN.md`), and one working from the spec receives the step's model.

An author who knew only the requirements document wrote the steps, so that they follow from running the workshop and not from how any build came out. The mix was set beforehand: two small changes, three that cross areas, a new area, a removal and a renaming. Each step's "For the assessors only" lists what the request reaches, and those marked as unstated are what the hidden cases must catch without the request saying so.

| # | Title | Kind | Touches an earlier step |
| --- | --- | --- | --- |
| 1 | Bookings in 15-minute steps | small, one area | |
| 2 | A waiver signature lasts a year | across areas, through a shared rule | |
| 3 | A fee for no-shows | across areas, through a shared record | 1: the fee is on reserved minutes, which can now be 15 |
| 4 | Pausing a membership | across areas, through a shared record | 3: a paused member is still charged for a no-show |
| 5 | A tool library | new area | 2 and 4: its gate uses the changed waiver rule and the paused standing |
| 6 | Renewing a tool loan | small, one area | 5: the area itself; 4: a paused member cannot renew |
| 7 | Prepaid minutes become packs | renaming | 3: the kind order and the account order now hold `noShow` and `revenueNoShows` |
| 8 | Shifts and supervision are removed | removal | 5: the shift rule of `borrowTool` goes too |

---

## 1. Bookings in 15-minute steps

### The request

Most jobs on the small laser take a quarter of an hour, yet members have to block half an hour and the machine stands idle for the rest. Please make the booking granularity 15 minutes.

- A reservation's `start` and `end` must each fall on a minute that is a multiple of 15 within the hour: `:00`, `:15`, `:30` or `:45`.
- A reservation lasts at least 15 minutes and, as today, at most 240 minutes.
- `createReservation` keeps its refusal codes and their order. `invalid_slot` is now answered when `start` or `end` is not on a 15-minute boundary or the length is outside 15 to 240 minutes.

Everything else about reservations stays as it is.

### For the assessors only

- **Overview:** the Bookings bullet says 30-minute steps; it becomes 15.
- **Booking granularity:** the step becomes 15 minutes, with the four allowed minutes and a minimum length of 15.
- **Create reservation, refusal 2:** the boundary and the length range change to the new values.
- **Create reservation, first example (unstated):** `10:15`–`11:00` on Tuesday `2026-03-03` is no longer `invalid_slot` and is accepted; the example needs another slot, such as `10:10`–`11:00`.
- **Opening hours (unstated):** the rule stays, and it now meets new slots: `17:45`–`18:00` on a Saturday is accepted and `17:45`–`18:15` is `outside_opening_hours`.
- **Time, intervals (unstated):** `10:00`–`10:15` and `10:15`–`10:30` only touch, so they do not conflict; nothing to change, but it is newly reachable.
- **Time, durations:** the maximum of 240 stays; `10:00`–`14:15` is `invalid_slot`.
- **Bookings interface, Reservation rules, Eligibility:** they name granularity without a number and stay true.
- **Schedule maintenance, Supervision, Utilization report:** they work on any minute and stay as they are, which is why this stays in one area.

Kind: small change in one area, one rule with an example that it invalidates.

---

## 2. A waiver signature lasts a year

### The request

Our insurer no longer accepts waiver signatures older than a year, so from now on a signature lapses.

- A signature's **lapse date** is the date 365 days after the date of its `signedAt`. It counts up to the day before its lapse date and not on the lapse date itself. Example: a signature with `signedAt` `2026-03-02T09:00Z` has the lapse date `2027-03-02`; it counts at `2027-03-01T23:59Z` and does not count at `2027-03-02T00:00Z`.
- A member **has signed** a family's waiver when the member's latest signature for the family has the family's current version and has not lapsed on the date of the request time. The date of a slot plays no part in this.
- A lapsed signature no longer counts as signed wherever the program judges whether a member has signed.
- A member can now have more than one signature of the same version of a waiver, so a member's latest signature for a family is the one recorded last.
- The signature record keeps its fields; the lapse date is neither stored nor shown. There is no new operation and no new refusal code.

### For the assessors only

- **Waiver gate:** the definition of having signed gains the lapse; the gate refuses `waiver_missing` for a lapsed signature as well as a missing or old one.
- **Waiver gate, existing reservations (unstated):** like a new version, a lapse leaves reservations in place, and their check-in is refused until the member signs again.
- **Waiver signature:** "one per version signed" and "the latest is the one with the highest version" no longer hold; the latest is the one recorded last, which was the same thing while versions only grew.
- **Sign waiver, `already_signed` (unstated):** it is judged as the gate defines having signed, so a member whose signature has lapsed can sign the same version again, and the new `signedAt` starts a new year. Before the lapse date it is still `already_signed`.
- **Sign waiver, `guardian_required` (unstated):** a minor without consent who signs again after a lapse is refused as on a first signature; the order of refusals stays.
- **List member waivers (unstated):** the latest signature per family is the one recorded last, so it shows the new `signedAt`. A lapsed signature is still listed: this page lists every family signed, old versions included, and nothing asks to change that.
- **Create reservation, refusal 8 (unstated):** a lapsed signature refuses with `waiver_missing` whatever the slot's date; a signature that lapses tomorrow lets through a slot next month.
- **Check in, refusal 6 (unstated):** it judges the same gate on the date of the check-in request.
- **Check out:** an unsigned waiver does not stop a check-out; a lapsed one does not either.
- **Reservation rules, rule 6:** the text speaks only of the version and needs the lapse.
- **Eligibility, blocker 4 (unstated):** the blocker judges the gate as `createReservation` does, at the request time, so the `date` argument plays no part: a far-future `date` is not blocked while the signature counts today, and a past `date` is blocked when it has lapsed today.
- **Family compliance, `waiverSigned` (unstated):** it must be `false` for a lapsed signature, although the page's wording speaks only of the version; it cites the waiver gate.
- **Time, states that depend on time:** a lapsed signature joins the list.
- **Computed times:** the lapse date is used only in comparisons and can lie after 2099, at most `2100-12-31`; a signature made in 2099 never lapses at a request time.
- **Publish waiver, Get waiver:** unchanged; a new version still needs a new signature, however recent the old one is.
- **Overview:** the Guardians and waivers bullet gains the year.

Kind: change across areas through a shared rule read by bookings, compliance and signing itself; most of the impact does not appear in the request's words.

---

## 3. A fee for no-shows

### The request

Members book machines, do not come, and the slot is lost to everyone else. I want them to pay half the price of the time they blocked. A missed reservation causes a charge from now on, but only through a run that staff start by hand, and never when the machine could not have been used anyway.

The run takes every reservation whose reported status is `missed` at the request time, in ascending reservation number (the number `N` of `res-N`, compared as a number). For each one that no earlier run has charged and whose machine was not locked out during the slot, it creates a charge for the reservation's member: `kind` `noShow`, `ref` the reservation identifier, `quantity` the reserved minutes (`end − start`), `unitPriceCents` the machine's current rate, `discountPercent` 50, `at` the request time, and the amount by money.

- The machine **was locked out during the slot** when a lockout of the machine, of any source, overlaps `[start, end)`, taking the lockout as the interval `[appliedAt, releasedAt)`. A lockout that is not released overlaps when its `appliedAt` is before `end`. Such a reservation is not charged, by this run or any later one.
- The member's standing is not checked.
- The reservation does not change: it stays `missed` and its `amountCents` stays `null`.

Example: at a rate of 45 cents per minute, a missed reservation of 60 minutes gives `(60 × 45 × 50 + 50) div 100 = 1350` cents, and one of 15 minutes gives `(15 × 45 × 50 + 50) div 100 = 338` cents.

- **Wire name:** `chargeNoShows`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `charges`: a list of charge records, the charges created, in ascending reservation number; empty when nothing is due
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `noShows`.

`noShow` is a new kind of charge. It comes last in the kind order, after `event`. Its invoice lines are credited to a new ledger account, `revenueNoShows`, which means "No-show lines invoiced."

### For the assessors only

- **Reservations:** "causes no charge" is replaced; a missed reservation can be charged by the run and still cannot be checked in or cancelled.
- **Time, states that depend on time (unstated):** `missed` is judged at the run's request time, so a reservation whose `end` equals the request time is charged by that run.
- **Reservations (unstated):** cancelled, completed, checked-in and still-reserved reservations are not charged, because they are not `missed`.
- **Bookings interface:** the operation joins the list.
- **Lockouts interface:** bookings now also asks whether a lockout of a machine overlaps an interval; bookings already depends on lockouts, so no new dependency.
- **List order:** a row for `chargeNoShows` → `charges`.
- **Charge:** `kind` gains `noShow`; `ref` gains the missed reservation.
- **Charge kind row (unstated):** `kind` gains `noShow`.
- **Charges by kind and its row in List order (unstated):** `rows` has 11 rows with `noShow` last, and `totalCents` includes it.
- **Postings, revenue account table:** `noShow` → `revenueNoShows`, so `generateInvoices` posts debit `receivable`, credit `revenueNoShows`, memo the `invoiceId`.
- **Ledger account (unstated):** the account takes its ascending place, between `revenueMembership` and `revenuePasses`.
- **Get trial balance (unstated):** one more row, 24 in all, in that place, also when it has no entries.
- **Post adjustment (unstated):** both account arguments list the accounts by name; `revenueNoShows` must now be accepted.
- **Revenue report (unstated):** the four kind sums and `invoicedCents` leave the new kind out, while `creditAppliedCents`, `outstandingCents` and `paymentsCents` include it, as the page says of other kinds.
- **Finance summary (unstated):** `invoicedCents` covers all kinds, so it includes the new one.
- **Charges and invoices (unstated):** the charge is unbilled until the invoice of the month of its `at`, the run's month, whatever month the slot was in.
- **Get account (unstated):** `unbilledCents` grows.
- **Issue refund (unstated):** a no-show line can be refunded like any other.
- **Membership (unstated):** the tier discount does not apply, since it applies only to usage charges; the 50 is the whole discount.
- **Pass draw, Consumption (unstated):** no minutes and no stock are drawn, since both happen only at `checkOut`.
- **Project cost (unstated):** an attached missed reservation still adds nothing, because the cost reads `completed` reservations only.
- **Escalation effects (unstated):** a future reservation it cancels is `cancelled`, not missed; one it leaves because it has started is exempt if it goes missed, since the work order's lockout overlaps it.
- **Time, intervals:** the lockout overlap follows the stated formula, including a lockout applied and released in one minute.
- **Set machine rate (unstated):** the run uses the rate at its own request time, not the rate when the slot was booked.
- **Overview:** the Bookings bullet gains the fee.

Kind: change across areas through a shared record (the charge kind and the ledger accounts); the ledger, the adjustment argument and the finance reports do not appear in the request's words.

---

## 4. Pausing a membership

### The request

Members who go abroad for a few months ask to keep their place without paying the full fee. I want a third standing, `paused`.

- `setMemberStanding` also accepts `paused`. As today, any standing can be set from any other, and setting the current one succeeds without a change. Pausing cancels nothing.
- A paused member is not in good standing. Wherever a member who is not in good standing is refused today, a paused member is refused the same way with the same code, `member_suspended`, and what a suspended member can still do, a paused member can do too.
- The monthly fee of a paused member is 500 cents, whatever the tier. When `generateInvoices` finds a member paused at the request time of that request, whatever month is invoiced, the member's membership line has `kind` `membership`, `ref` the member's current tier, `quantity` 1, `unitPriceCents` 500, `discountPercent` as household discount states and `at` the request time. A suspended member keeps paying the tier's full fee.

Example: a paused member on `pro` pays 500 cents, and 450 cents (`(500 × 90 + 50) div 100`) in a household of 3 or more.

There is no new operation and no new refusal code.

### For the assessors only

- **Overview:** the Members bullet gains the pause.
- **Member:** `standing` has three values, in every result that carries a member.
- **Set member standing:** the argument gains `paused`; other values stay `invalid_argument`.
- **Membership:** the rule gains the paused standing and the exception to "the fee of the tier is charged once".
- **Tier:** the monthly fee of the table does not apply to a paused member.
- **Charges and invoices, step 3:** the membership line of a paused member, with the fee above. The invoice is still made, as for every member.
- **Household discount (unstated):** the discount applies to the 500 cents, and a paused member still counts as a member, so a household of 3 with one paused member gets the discount on all three lines.
- **Postings (unstated):** the line posts to `revenueMembership` at its reduced amount; `membershipCents` of the revenue report follows.
- **Issue refund (unstated):** line 1 of a paused member's invoice has only its reduced amount to refund.
- **Issue PPE, Take deposit, Sell pass, Redeem promotion, Rent locker, Issue guest pass, Enroll in course, Purchase material, Check in, Create reservation, Enter door, Join waitlist, Sell ticket, Start job (unstated):** each answers `member_suspended` for a paused member, at its stated place in the order.
- **Waitlist promotion, Skipped entry (unstated):** a paused member on the list is skipped with `member_suspended`.
- **Maker gate, Reservation rules (unstated):** rule 3 and the maker's standing hold for a paused member as for a suspended one.
- **Eligibility, blocker 1 (unstated):** a paused member has the blocker `member_suspended`.
- **Compliance summary (unstated):** the `standing` result lists only `good` or `suspended` and must now also answer `paused`.
- **Dunning run, step 3 (unstated):** a paused member is not "already suspended", so the suspension step makes the standing `suspended`, and the next invoice has the tier's full fee.
- **Cancel reservation, Check out, Record payment, Sign waiver, Declare minor, Return PPE, Leave door, Admit guest, Issue badge, Log volunteer time, Complete course, Complete job, Create project (unstated):** they do not check standing and stay open to a paused member.
- **Pass draw, Bill lockers (unstated):** standing plays no part; a paused member's passes are drawn and lockers are billed in full.
- **Charge no-shows of step 3 (unstated):** the run does not check standing, so a paused member is charged.
- **Members interface:** billing now also asks whether a member is paused; billing already depends on members.

Kind: change across areas through a shared record (the member's standing), 14 operations and two reads that judge it, and a dunning rule that overrides the new value.

---

## 5. A tool library

### The request

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

### For the assessors only

- **Overview:** a new bullet.
- **Architecture:** a row for `loans`; `bookings` and `protocol` gain it; both stay one-way as the request says.
- **Layout:** `realization/loans/` belongs to the module.
- **Initial state:** a row for `lon-N`; no tools or loans at the start, and the sentence "no record of any other module" already covers them.
- **List order:** the three new rows.
- **Time, states that depend on time:** an overdue loan joins the list, and `overdue` is `false` once returned.
- **Computed times (unstated):** `dueOn` can lie after 2099, at most `2100-01-14`, and appears in results in the same format.
- **Membership (unstated):** a paused or suspended member cannot borrow, which also reaches the paused standing of step 4.
- **Guardian consent, the consent gate:** the table gains `borrowTool`, the tool's family, the date of the request time.
- **Waiver gate:** it now also applies at `borrowTool`, with the tool's family in place of the machine's, and it includes the lapse of step 2 (unstated).
- **Training interface, `not_certified`:** the meaning speaks of a machine's family; `borrowTool` states its own, as Refusal codes allows.
- **Volunteering interface, Staffing interface (unstated):** `approver_not_on_shift` is shared with its meaning unchanged, and `loans` asks staffing the question it already answers, so a shift ending at the request time does not count.
- **Notices, Notice kind:** the new kind with its subject; `kind` in Notice gains it and `ref` gains the loan identifier. **Posting notices (unstated):** the notice takes the next notice number and has no audit entry of its own.
- **Charge:** `ref` of a `lateFee` line can now be a loan identifier. **Dunning run (unstated):** its own late fees are not affected, since they are per invoice.
- **Postings, Revenue report, Charges by kind (unstated):** no new kind or account; the line posts to `revenueLateFees` and counts in the `lateFee` row; in the revenue report it is in none of the four kind sums.
- **Charges and invoices (unstated):** the charges are invoiced in the month of their `at`, like any other.
- **Check in:** refusal 10 and its rule name; **Bookings interface:** `loans` joins Uses and `loan_overdue` joins the codes added by other modules.
- **Check in, order (unstated):** `not_on_premises` still comes before `loan_overdue`.
- **Eligibility (unstated):** `loan_overdue` is a check-in rule and stays out of the blockers, like `ppe_missing` and `not_on_premises`; the sentence naming those two gains it, and results do not change.
- **Create reservation, Check out, Compliance summary:** unchanged.
- **Order of checks (unstated):** `memberId` is judged before `toolId` at check 5, and `memberId` must name a member; a member actor is refused at check 4 for `borrowTool` and `returnTool`.

Kind: new area, connected through its borrow gate (standing, consent, certification, waiver), the shift rule, the late-fee charge and notice, and a new refusal in `checkIn`.

---

## 6. Renewing a tool loan

### The request

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

### For the assessors only

- **Loan (unstated):** the new field appears in every result that carries a loan: `borrowTool`, `returnTool`, `listMemberLoans` and `renewLoan`.
- **Authorization:** `loanId` joins the ownership list. **Order of checks (unstated):** another member's loan is `forbidden` at check 6, before any refusal of check 7; for `returnTool`, which also has `loanId`, a member is still refused at check 4.
- **Overdue, late-fee charge (unstated):** a renewed loan is overdue only after its new `dueOn`, so the `checkIn` refusal, `borrowTool` and `chargeOverdueLoans` all follow it. `chargedDays` is still 0 at a renewal, because an overdue loan cannot be renewed.
- **Computed times (unstated):** the latest `dueOn` rises to `2100-02-11`.
- **Membership (unstated):** a paused member, from step 4, is refused like a suspended one.
- **Tool:** unchanged; `onLoan` stays `true`.

Kind: small change in one area, one operation and one field on what step 5 added, with one ownership line.

---

## 7. Prepaid minutes become packs

### The request

At the desk members keep mixing up their prepaid machine minutes with guest passes. From now on we call a block of prepaid machine minutes a **pack**. This is a renaming only: nothing is to behave differently. Guest passes keep their name and everything that belongs to them, and the words for passing an inspection or an assessment stay too.

| Now | From now on |
| --- | --- |
| operation `addPassProduct` | `addPackProduct` |
| operation `sellPass` | `sellPack` |
| operation `listMemberPasses` | `listMemberPacks` |
| field `passId` | `packId` |
| result `pass` of `sellPack` | `pack` |
| result `passes` of `listMemberPacks` | `packs` |
| charge kind `pass` | `pack` |
| ledger account `revenuePasses` | `revenuePacks` |
| identifiers `pas-1`, `pas-2`, … | `pak-1`, `pak-2`, …, the same counter |
| module `passes` | `packs` |

A pass product is now a pack product. Its `product` result and the fields `productId`, `name`, `minutes`, `priceCents` and `validDays` keep their names, as do `minutesLeft`, `issuedOn` and `expiresOn`. The old names are gone; there are no aliases.

### For the assessors only

- **Invalid requests (unstated):** the three old operation names now name no operation, so they are answered `invalidRequest`.
- **Audit entry (unstated):** `op` holds the new names, and the target of a sale is the new `packId`.
- **Initial state:** the identifier row becomes `Pack`, `pak-1`.
- **Pass, Pass product, Pass draw, Passes interface, Add pass product, List member passes, Sell pass:** the pages and their wire names change; the draw rule stays word for word in effect.
- **List order:** the row for `listMemberPacks`; the kind order of `chargesByKind` has `pack` in the old place of `pass`, with `noShow` from step 3 still last.
- **Charge, Charge kind row, Charges by kind (unstated):** `kind` is `pack` and the row keeps its place; the charge's `ref` is `pak-N`.
- **Postings, revenue account table:** `pack` → `revenuePacks`.
- **Ledger account:** the account is renamed and, in ascending order, keeps its place: `revenueNoShows` < `revenuePacks` < `revenueRefunds`.
- **Get trial balance (unstated):** the row is named `revenuePacks`, in the same place.
- **Post adjustment (unstated):** both account arguments accept `revenuePacks`; `revenuePasses` is now `invalid_argument`.
- **Architecture:** the `passes` row, and the `bookings` and `protocol` rows that name it. Not decided by a behaviour test.
- **Computed times, Overview, Metering, Check out, Consumption, Pass draw example:** the prose and the examples use the new word.
- **Stays, and an over-eager rename is the fault to test for:** `guestPassId`, `guestPass`, `guestPasses`, `issueGuestPass`, `revokeGuestPass`, `listGuestPasses`, `pass_not_today`, `pass_of_another_member`, charge kind `guest`, `revenueGuests`, `passMark`, `passed`, `lastPassedOn`, `assessment_not_passed`.

Kind: renaming with no change in behaviour, reaching the shared charge kind and the ledger accounts, next to a similar word that must stay.

---

## 8. Shifts and supervision are removed

### The request

Staff scheduling has moved to our payroll system, so nobody keeps shifts in here any more. Please remove shifts and supervision whole.

- The operations `addShift`, `cancelShift`, `listShifts` and `setSupervision` are gone, and with them the shift and supervision setting records, the identifier `shf-N` and the refusal codes `not_staff`, `shift_needed`, `shift_overlap`, `shift_too_long` and `supervisor_required`. The module `staffing` leaves the architecture.
- No family is supervised any more, so no reservation is refused for lack of cover. The other refusals of `createReservation` keep their order.
- Nobody needs to be on shift for anything any more: the refusal code `approver_not_on_shift` is gone too, and staff can act at any time.

### For the assessors only

- **Invalid requests (unstated):** the four operation names are answered `invalidRequest`.
- **Overview:** the Staffing bullet goes.
- **Architecture:** the `staffing` row and its note go; the rows of `volunteering`, `bookings`, `loans` and `protocol` lose it.
- **List order:** the `listShifts` row goes.
- **Initial state:** "Every family is unsupervised" and the row for `shf-N` go.
- **Staffing, all its pages:** they go.
- **Reservation rules:** rule 10 goes, and the sentence naming the rules that bind only where a record exists no longer names it.
- **Create reservation:** refusal 12 goes and the rule name leaves its list; `machine_in_maintenance` is followed directly by `machine_booked`. A slot with no shift at all is accepted on any family.
- **Bookings interface (unstated):** the capability "active reservations of all machines of a family", which only staffing asks, goes; `supervisor_required` leaves the codes added by other modules.
- **Volunteering interface:** staffing leaves Uses, and `approver_not_on_shift` leaves the codes.
- **Decide volunteer time (unstated):** refusal 2 and the paragraph on being on shift go; staff with no shift can decide, and `invalid_state` and `volunteer_cap_reached` keep their relative order.
- **Borrow tool of step 5 (unstated):** refusal 1 goes, staff with no shift can hand out a tool, and the other refusals keep their order. `loans` no longer depends on `staffing`.
- **Eligibility:** supervision leaves the list of rules not judged; results do not change.
- **Authorization, Registry (unstated):** the role `staff` and `registerUser` stay; only the shifts go.
- **Return tool, Renew loan:** unchanged, since they never needed a shift.

Kind: removal of a whole feature with a gate in bookings and rules in two other areas that relied on it, one of them added in step 5.
