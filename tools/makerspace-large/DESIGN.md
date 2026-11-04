# Makerspace large: design

The owner chose on 3 November 2026 to grow Makerspace to about 5× its size, so that the approaches can be tested at a size that strains one session. This page is the design every author of the project's parts works from. It is assessor material: no variant receives it.

## The rules of the growth

1. **Old behavior stays.** Every one of Makerspace's 37 operations answers exactly as it does today, in the same shape, unless a record of a new module exists that changes it. Makerspace's 54 hidden cases stay as they are and must pass on the grown program. No field is added to an old record. A new rule never applies before the new record that turns it on exists.
2. **Real rules across modules, not copies.** Every new module has at least one rule that reaches another module, and several rules run through many modules.
3. **Requests only.** No clock, no scheduled job. What must happen periodically is an explicit run operation, such as `runDunning`.
4. **One way dependencies.** The table below is a directed graph. Where a lower module must ask an upper one, it declares a need and `protocol` supplies the answer, as `equipment` already does for `bookings`.
5. **Every new generated identifier** has a prefix of at most four characters and a dash, and its own counter.

## Size

48 modules (10 old, 38 new). 188 operations (37 old, 151 new). About 550 checklist items and 60,000 words of model.

## Modules

The order below is a valid build order. Every module also depends on `policies`, not listed. `Needs` names what a module declares and `protocol` supplies.

| Module | Status | Depends on | Needs from (through protocol) | Author cluster |
|---|---|---|---|---|
| `members` | old | | | |
| `equipment` | old | | bookings | |
| `ledger` | new | | | money |
| `notices` | new | members | | people |
| `households` | new | members | | people |
| `guardians` | new | members, equipment | | people |
| `waivers` | new | members, equipment, guardians | | people |
| `ppe` | new | members, equipment | | people |
| `lockouts` | new | equipment | | floor |
| `inspections` | new | equipment, lockouts | | floor |
| `staffing` | new | members, equipment | bookings | people |
| `parts` | new | equipment | | floor |
| `rooms` | new | | | teaching |
| `billing` | changed | members, ledger, households | | money |
| `deposits` | new | members, billing, ledger | | money |
| `passes` | new | members, billing | | money |
| `promotions` | new | members | | money |
| `scholarships` | new | members | | money |
| `refunds` | new | members, billing, ledger | | money |
| `dunning` | new | members, billing, notices | | money |
| `lockers` | new | members, billing | | money |
| `guests` | new | members, billing | | people |
| `volunteering` | new | members, billing, ledger, staffing | | people |
| `training` | changed | members, equipment, billing, guardians, promotions, scholarships | classes, assessments | teaching |
| `materials` | changed | members, billing, promotions | hazmat, stocktake | trade |
| `consumables` | new | equipment, materials, billing | | floor |
| `bookings` | changed | members, training, equipment, billing, guardians, waivers, ppe, lockouts, inspections, staffing, passes, consumables | access | floor |
| `safety` | old | equipment | | |
| `reporting` | old | equipment, bookings, billing | | |
| `access` | new | members, billing, guests, bookings | | people |
| `instructors` | new | members, equipment, training | classes | teaching |
| `classes` | new | members, training, instructors, rooms | | teaching |
| `assessments` | new | members, training | | teaching |
| `waitlists` | new | members, training, notices | | teaching |
| `payouts` | new | members, ledger, instructors, classes | | teaching |
| `events` | new | members, rooms, billing, ledger | | teaching |
| `hazmat` | new | equipment, materials, training | | trade |
| `purchasing` | new | materials, ledger | | trade |
| `stocktake` | new | materials, ledger | | trade |
| `workorders` | new | members, equipment, safety, parts, lockouts | | floor |
| `escalations` | new | equipment, safety, lockouts, bookings, workorders, notices | | floor |
| `projects` | new | members, bookings, materials | | trade |
| `quotes` | new | equipment, materials | | trade |
| `jobs` | new | members, equipment, billing, ledger, training, quotes | | trade |
| `compliance` | new | members, equipment, guardians, waivers, ppe, training, lockouts, inspections | | floor |
| `finance` | new | members, ledger, billing, households, refunds | | money |
| `protocol` | old, grows | every module | | |

`policies` is the shared module. A module named by `Depends on` may be used only through its Interface.

## Changes to old modules

These are the only changes. Each applies only when the new record exists.

### `createReservation`

The refusals become, in this order, new codes in bold:

`invalid_interval`, `invalid_slot`, `starts_in_past`, `outside_opening_hours`, `member_suspended`, **`guardian_consent_missing`**, `not_certified`, **`waiver_missing`**, **`machine_locked_out`**, **`inspection_due`**, `machine_in_maintenance`, **`supervisor_required`**, `machine_booked`, `member_double_booked`.

### `checkIn`

`invalid_state`, `check_in_too_early`, `member_suspended`, **`guardian_consent_missing`**, `not_certified`, **`waiver_missing`**, **`ppe_missing`**, **`machine_locked_out`**, **`not_on_premises`**.

`guardian_consent_missing` is judged on the date of the request time here, and on the date of `start` for `createReservation`.

### `checkOut`

One operation, in this order, all effects together:

1. The reservation's `meteredMinutes` is computed as today.
2. **Passes:** minutes are drawn from the member's valid passes (see `passes`). `billable = meteredMinutes − drawn`.
3. The usage charge is created as today, with `quantity` `billable`. A charge of 0 cents is still created. The reservation's `amountCents` is that charge's amount.
4. **Consumables:** one material charge per consumption row of the machine, in ascending `materialId` order, as `consumables` states. They are created after the usage charge. They use the full metered minutes, whatever the passes drew.

### `enrollInCourse`

Refusals: `member_suspended`, **`guardian_consent_missing`** (for the course's family, on the date of the request time), `already_enrolled`, `course_full`.

The course charge's `discountPercent` is decided by `training` from two sources:

- the member's active scholarship award, when the fund's remaining budget covers the discount it would give;
- the member's oldest unused redeemed promotion whose scope covers courses.

The higher percent wins; on equal percent the award wins. Only the winner is used. A course with fee 0 uses neither. The discount of an award is `feeCents − amountCents`, which is added to the fund's `spentCents`.

### `completeCourse`

Refusals: `not_enrolled`, **`attendance_missing`**, **`assessment_not_passed`**. Their answers come from `classes` and `assessments` through declared needs.

### `purchaseMaterial`

Refusals: `member_suspended`, **`hazard_not_certified`**, **`hazard_limit_exceeded`**, **`stocktake_open`**, `insufficient_stock`.

`discountPercent` is the percent of the member's oldest unused redeemed promotion whose scope covers materials, otherwise 0. It is then used.

### `restockMaterial`

Refusal: **`stocktake_open`**.

### `generateInvoices`

The membership line's `discountPercent` is 10 when the member belongs to a household with 3 or more members at the request time, otherwise 0. Ledger postings are made (see `ledger`). Nothing else changes.

### `recordPayment` and `issueCredit`

Ledger postings are made. Nothing else changes.

### `revenueReport`

Unchanged, and it says so: its sums cover the four kinds it names, and charges of new kinds are in none of its figures. `chargesByKind` covers every kind.

### Shared pages

- **Charge kind** gains `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event`.
- **Authorization:** a record concerned also belongs to the actor when the operation has one of these arguments and the record is the actor's: `guardianId` (equals the actor; where an operation has `guardianId`, ownership is judged on it alone), `depositId`, `rentalId`, `guestPassId`, `badgeId`, `noticeId`, `ticketId`, `entryId` (a volunteer time entry), `ownerId` (equals the actor), `projectId` (the project's owner; where an operation has `projectId`, ownership is judged on it alone). `reservationId` and `invoiceId` stay as they are.
- **Refusal codes:** `invalid_state`, `already_exists`, `member_suspended`, `invalid_interval` and `starts_in_past` keep their meaning and are reused. Every other new code belongs to the module that names it.
- **Project page, overview:** a line per new area. "Notifications" stays out of scope as sending; a notice is a record in the program.
- **Safety:** reporting an incident still blocks nothing; `escalateIncident` is what acts on one.
- **Initial state, generated identifiers, order of listed results:** every new record, prefix and list is added here.

## New modules

Every operation also takes `actor` and `at`. `M` means a mutation, `R` a read. Permissions are given as member/staff/admin; `-` means none. These contracts are binding. An author adds what a page needs to be read one way: result fields, examples, boundary sentences. An author changes none of the names, codes or orders here without saying so in the page's cluster notes.

### `ledger`: double-entry journal

Accounts, a fixed enumeration in this ascending listing order: `cash`, `creditLiability`, `depositLiability`, `expenseGoodwill`, `expenseMaker`, `expensePayout`, `expenseVolunteer`, `expenseWriteOff`, `inventory`, `payable`, `receivable`, `revenueCourse`, `revenueDeposits`, `revenueEvents`, `revenueGuests`, `revenueJobs`, `revenueLateFees`, `revenueLockers`, `revenueMaterial`, `revenueMembership`, `revenuePasses`, `revenueRefunds`, `revenueUsage`.

A journal entry (`jrn-N`): `entryId`, `at`, `debitAccount`, `creditAccount`, `amountCents`, `memo` (text, what the posting is for, as the table says). An amount of 0 posts nothing and consumes no number.

| Event | Debit | Credit | Amount | Memo |
|---|---|---|---|---|
| `generateInvoices`, per invoice line, invoices in `memberId` order, lines in order | `receivable` | by kind: membership `revenueMembership`, usage `revenueUsage`, material `revenueMaterial`, course `revenueCourse`, deposit `depositLiability`, pass `revenuePasses`, locker `revenueLockers`, guest `revenueGuests`, lateFee `revenueLateFees`, event `revenueEvents` | line amount | invoiceId |
| `generateInvoices`, credit applied, after that invoice's lines | `creditLiability` | `receivable` | `creditAppliedCents` | invoiceId |
| `recordPayment` | `cash` | `receivable` | `appliedCents` | paymentId |
| `recordPayment`, then | `cash` | `creditLiability` | `creditedCents` | paymentId |
| `issueCredit` | `expenseGoodwill` | `creditLiability` | amount | memberId |
| `releaseDeposit` | `depositLiability` | `creditLiability` | amount | depositId |
| `forfeitDeposit` | `depositLiability` | `revenueDeposits` | amount | depositId |
| `issueRefund` | `revenueRefunds` | `creditLiability` | amount | refundId |
| `decideVolunteerTime`, approved | `expenseVolunteer` | `creditLiability` | credit | entryId |
| `cancelTicket`, `cancelEvent` per ticket | `revenueRefunds` | `creditLiability` | credit | ticketId |
| `receiveOrder` | `inventory` | `payable` | quantity × unit cost | orderId |
| `closeStocktake`, shrink, per material | `expenseWriteOff` | `inventory` | units × current unit price | stocktakeId |
| `closeStocktake`, gain, per material | `inventory` | `expenseWriteOff` | units × current unit price | stocktakeId |
| `generatePayouts`, per payout | `expensePayout` | `payable` | amount | payoutId |
| `settlePayout` | `payable` | `cash` | amount | payoutId |
| `completeJob` | `expenseMaker` | `creditLiability` | maker's credit | jobId |
| `recordJobPayment` | `cash` | `revenueJobs` | amount | jobId |
| `postAdjustment` | as given | as given | as given | memo as given |

Operations:

- `listJournalEntries(month)` R -/-/all. The entries whose `at` lies in the month, `entryId` creation order.
- `getTrialBalance()` R -/-/all. One row per account in the listing order, with `debitCents`, `creditCents`, `balanceCents` (the larger minus the smaller) and `side` (`debit`, `credit`, or `none` when equal), plus `totalDebitCents`, `totalCreditCents`, always equal.
- `postAdjustment(debitAccount, creditAccount, amountCents, memo)` M -/-/all. Refusal `same_account`. Target: new entryId.

**Invariants** the program must hold at all times: total debits equal total credits. The `receivable` balance equals the sum of `balanceCents` over all open invoices, unless `postAdjustment` touched it. The `creditLiability` balance equals the sum of all members' credit balances, unless `postAdjustment` touched it.

### `notices`

A notice (`ntc-N`): `noticeId`, `memberId`, `kind` (`manual`, `dunning`, `suspension`, `waitlist`, `cancellation`), `subject`, `ref` (text or nullable), `at`, `acknowledgedAt` nullable.

- `postNotice(memberId, subject)` M -/all/all. Kind `manual`, `ref` null. Target: new noticeId.
- `listNotices(memberId)` R own/all/all. Creation order.
- `acknowledgeNotice(noticeId)` M own/all/all. Refusal `invalid_state` when already acknowledged. Target: noticeId.

Other modules post notices through its interface; a posted notice is part of their operation and is not audited on its own.

### `households`

A household: `householdId`, `name`, `payerId`, `memberIds` (ascending). A member belongs to at most one. Size limit 6. The payer is always a member of it.

- `createHousehold(householdId, name, payerId)` M -/all/all. Refusals `already_exists`, `already_in_household`. Target: householdId.
- `addHouseholdMember(householdId, memberId)` M -/all/all. Refusals `already_in_household`, `household_full`. Target: `householdId/memberId`.
- `removeHouseholdMember(householdId, memberId)` M -/all/all. Refusals `not_in_household`, `payer_cannot_leave`. Target: `householdId/memberId`.
- `getHousehold(householdId)` R -/all/all.

### `guardians`

A member is a **minor on date D** when D is before the 18th anniversary of the declared `birthDate`. A member with no declaration is an adult. A date's 18th anniversary of 29 February is 1 March when that year has no 29 February.

A record: `memberId`, `birthDate`, `guardianId`, `consentFamilyIds` (ascending).

- `declareMinor(memberId, birthDate, guardianId)` M -/all/all. Refusals `already_exists`, `not_a_minor` (adult on the request date), `invalid_guardian` (the guardian is the member, or is a minor on the request date). Target: memberId.
- `grantConsent(guardianId, memberId, familyId)` M own/all/all. Refusals `not_guardian`. Granting a consent held already succeeds without a change. Target: `memberId/familyId`.
- `withdrawConsent(guardianId, memberId, familyId)` M own/all/all. Refusals `not_guardian`, `no_consent`. Target: `memberId/familyId`.
- `getGuardianship(memberId)` R own/all/all. Refusal `not_a_declared_minor`.

**Rule:** a minor without consent for the family is refused `guardian_consent_missing` at `enrollInCourse`, `createReservation` and `checkIn`, at the places above.

### `waivers`

Per family at most one current waiver, with `version` 1, 2, … A signature: `memberId`, `familyId`, `version`, `signedAt`.

- `publishWaiver(familyId, text)` M -/-/all. The next version becomes current. Target: familyId.
- `getWaiver(familyId)` R all/all/all. Refusal `no_waiver`.
- `signWaiver(memberId, familyId)` M own/all/all. Refusals `no_waiver`, `already_signed` (the current version), `guardian_required` (the member is a minor on the request date and has no consent for the family). Target: `memberId/familyId`.
- `listMemberWaivers(memberId)` R own/all/all. The latest signature per family, `familyId` ascending.

**Rule:** a member whose latest signature for the machine's family is not of the current version is refused `waiver_missing`. A family without a waiver requires none. A new version leaves existing reservations in place and blocks their check-in until signed.

### `ppe`

An item: `itemId`, `name`, `stock`. A family can require items; a requirement is never removed.

- `addPpeItem(itemId, name, stock)` M -/all/all. Refusal `already_exists`. `stock` 0 to 100000. Target: itemId.
- `requirePpe(familyId, itemId)` M -/-/all. Repeating succeeds without a change. Target: `familyId/itemId`.
- `issuePpe(memberId, itemId)` M -/all/all. Refusals `member_suspended`, `already_issued`, `out_of_stock`. Stock decreases. Target: `memberId/itemId`.
- `returnPpe(memberId, itemId)` M -/all/all. Refusal `not_issued`. Stock increases. Target: `memberId/itemId`.
- `listMemberPpe(memberId)` R own/all/all. Held items, `itemId` ascending.

**Rule:** at `checkIn`, `ppe_missing` when the member does not hold every item the machine's family requires.

### `lockouts`

A lockout (`lck-N`): `lockoutId`, `machineId`, `source` (`manual`, `inspection`, `workOrder`), `reason`, `appliedBy`, `appliedAt`, `releasedAt` nullable. Active while `releasedAt` is null. Several can be active on one machine.

- `applyLockout(machineId, reason)` M -/all/all. Source `manual`. Target: new lockoutId.
- `releaseLockout(lockoutId)` M -/all/all. Refusals `invalid_state` (released), `lockout_not_manual`, `not_lockout_holder` (the actor is not the one who applied it and is not an admin). Target: lockoutId.
- `listLockouts(machineId)` R -/all/all. Creation order.

**Rule:** `machine_locked_out` at `createReservation` and `checkIn` when the machine has an active lockout at the request time, whatever the slot's date. A session already checked in is not interrupted.

### `inspections`

A plan: `machineId`, `intervalDays` (1 to 3650), `lastPassedOn` (date), `nextDueOn` (date), `failed` (boolean).

- `setInspectionPlan(machineId, intervalDays)` M -/all/all. A new plan's `lastPassedOn` is the date of the request time; a changed plan keeps it. Target: machineId.
- `recordInspection(machineId, outcome, note)` M -/all/all. `outcome` `passed` or `failed`. Refusal `no_plan`. `passed`: `lastPassedOn` the request date, `failed` false, and every active lockout of source `inspection` on the machine is released. `failed`: `failed` true, and a lockout of source `inspection` is applied unless one is active. Target: machineId.
- `getInspectionStatus(machineId)` R -/all/all. Refusal `no_plan`.
- `listInspectionsDue(date)` R -/all/all. Plans whose `nextDueOn` is on or before `date`, `machineId` ascending.

`nextDueOn` is `intervalDays` days after `lastPassedOn`.

**Rule:** `inspection_due` at `createReservation` when the machine has a plan and the date of `start` is on or after `nextDueOn`. `checkIn` is not gated on the date.

### `staffing`

A shift (`shf-N`): `shiftId`, `userId`, `start`, `end`, `status` (`scheduled`, `cancelled`).

- `addShift(userId, start, end)` M -/all/all. `userId` names an existing user. Refusals `not_staff` (the user is a member), `invalid_interval`, `starts_in_past`, `shift_too_long` (over 720 minutes), `shift_overlap` (another scheduled shift of the user overlaps). Target: new shiftId.
- `cancelShift(shiftId)` M -/all/all. Refusals `invalid_state` (cancelled, or its start is at or before the request time), `shift_needed` (an active reservation of a supervised family would no longer be covered). Target: shiftId.
- `listShifts(date)` R -/all/all. Scheduled shifts overlapping that date, `start` ascending, then creation order.
- `setSupervision(familyId, required)` M -/-/all. `required` boolean. Setting the current value succeeds. Target: familyId.

**Rule:** `supervisor_required` at `createReservation` when the machine's family is supervised and the slot is not wholly inside the union of scheduled shifts. Coverage may be pieced from several shifts of several people. Turning supervision on does not touch existing reservations; `shift_needed` then protects them.

### `parts`

A part: `partId`, `name`, `unitCostCents` (1 to 1000000), `familyId` (the family it fits), `stock` (starts at 0).

- `addPart(partId, name, unitCostCents, familyId)` M -/all/all. Refusal `already_exists`. Target: partId.
- `restockPart(partId, quantity)` M -/all/all. 1 to 100000. Target: partId.
- `listParts()` R -/all/all. `partId` ascending.

### `rooms`

A room: `roomId`, `name`, `capacity` (1 to 1000). A hold (`hld-N`): `holdId`, `roomId`, `start`, `end`, `purpose`, `status` (`held`, `released`). Holds are made directly, or by `classes` and `events`, whose holds can be released only by cancelling them.

- `addRoom(roomId, name, capacity)` M -/-/all. Refusal `already_exists`. Target: roomId.
- `holdRoom(roomId, start, end, purpose)` M -/all/all. Refusals `invalid_interval`, `starts_in_past`, `room_taken`. Target: new holdId.
- `releaseRoomHold(holdId)` M -/all/all. Refusals `invalid_state` (released), `hold_not_direct`. Target: holdId.
- `getRoomSchedule(roomId, date)` R all/all/all. Held holds overlapping the date, `start` ascending, then creation order.

### `deposits`

A deposit (`dep-N`): `depositId`, `memberId`, `purpose`, `amountCents` (1 to 100000000), `status` (`held`, `released`, `forfeited`), `takenAt`, `closedAt` nullable.

- `takeDeposit(memberId, purpose, amountCents)` M -/all/all. Refusal `member_suspended`. Creates a charge: `kind` `deposit`, `ref` depositId, `quantity` 1, `unitPriceCents` amount, `discountPercent` 0, `at` the request time. Target: new depositId.
- `releaseDeposit(depositId)` M -/all/all. Refusals `invalid_state` (not held), `deposit_unpaid` (its charge is not on an invoice with status `paid`). The member's credit balance increases by the amount. Target: depositId.
- `forfeitDeposit(depositId, reason)` M -/-/all. Refusals `invalid_state` (not held), `deposit_unpaid`. Target: depositId.
- `listDeposits(memberId)` R own/all/all. Creation order.

### `passes`

A product: `productId`, `name`, `minutes` (1 to 100000), `priceCents` (1 to 100000000), `validDays` (1 to 3650). A pass (`pas-N`): `passId`, `memberId`, `productId`, `minutesLeft`, `issuedOn`, `expiresOn` (`validDays` days after `issuedOn`).

- `addPassProduct(productId, name, minutes, priceCents, validDays)` M -/-/all. Refusal `already_exists`. Target: productId.
- `sellPass(memberId, productId)` M own/all/all. Refusal `member_suspended`. Charge: `kind` `pass`, `ref` passId, `quantity` 1, `unitPriceCents` price, `discountPercent` 0. Target: new passId.
- `listMemberPasses(memberId)` R own/all/all. Creation order, all of them.

**Rule:** at `checkOut`, minutes are drawn from the member's passes valid on the date of the request time (date before `expiresOn`), earliest `expiresOn` first, then creation order, until the metered minutes are covered or the passes are empty. A pass need not be paid to be used.

### `promotions`

A promotion: `code` (identifier), `percent` (1 to 100), `scope` (`course`, `material`, `both`), `maxRedemptions` (1 to 100000), `validUntil` (date), `redemptions`, `ended` (boolean). A redemption: `code`, `memberId`, `percent`, `redeemedAt`, `usedAt` nullable, `usedOn` nullable (the charge's `ref`).

- `addPromotion(code, percent, scope, maxRedemptions, validUntil)` M -/-/all. Refusal `already_exists`. Target: code.
- `redeemPromotion(memberId, code)` M own/all/all. Refusals, in this order: `member_suspended`, `promotion_ended`, `promotion_expired` (the request date is after `validUntil`), `promotion_exhausted`, `already_redeemed` (this member, this code). Target: `memberId/code`.
- `listMemberPromotions(memberId)` R own/all/all. Redemptions in creation order.
- `endPromotion(code)` M -/-/all. Refusal `invalid_state` (ended). Redemptions held stay usable. Target: code.

"Oldest unused" means creation order among the member's redemptions with `usedAt` null whose scope covers the charge.

### `scholarships`

A fund: `fundId`, `budgetCents` (1 to 100000000), `spentCents`. An award (`aws-N`): `awardId`, `fundId`, `memberId`, `percent` (1 to 100), `status` (`active`, `revoked`).

- `addFund(fundId, budgetCents)` M -/-/all. Refusal `already_exists`. Target: fundId.
- `awardScholarship(fundId, memberId, percent)` M -/-/all. Refusal `already_awarded` (the member has an active award). Target: new awardId.
- `revokeScholarship(awardId)` M -/-/all. Refusal `invalid_state`. Target: awardId.
- `listAwards(memberId)` R own/all/all. Creation order.

**Rule:** see `enrollInCourse`. A fund that cannot cover the discount gives none, is not touched, and no refusal results.

### `refunds`

A refund (`rfd-N`): `refundId`, `invoiceId`, `lineNumber`, `amountCents`, `reason`, `at`.

- `issueRefund(invoiceId, lineNumber, amountCents, reason)` M -/-/all. `lineNumber` is 1 to 1000; 1 is the first line. Refusals, in this order: `invoice_not_paid`, `invalid_line`, `refund_too_large` (over the line's amount less the line's earlier refunds). The member's credit balance increases by the amount. Target: new refundId.
- `listRefunds(invoiceId)` R own/all/all. Creation order.

### `dunning`

A ladder: `noticeDays`, `feeDays`, `feeCents`, `suspendDays`. A case (`dun-N`), one per invoice that first enters the ladder: `caseId`, `invoiceId`, `memberId`, `noticedAt`, `feeChargedAt`, `suspendedAt` (nullable timestamps).

- `setDunningLadder(noticeDays, feeDays, feeCents, suspendDays)` M -/-/all. Days 1 to 3650, `feeCents` 1 to 1000000. Refusal `invalid_ladder` unless `noticeDays` < `feeDays` < `suspendDays`. Target: the fixed string `ladder`.
- `runDunning()` M -/all/all. Refusal `no_ladder`. Target: the fixed string `dunning`. Result `actions`.
- `listDunningCases(memberId)` R own/all/all. Creation order.

**Rule of a run.** Every overdue invoice at the request time, ascending by its number. Days overdue is the whole days from `dueAt` to the request time, rounded down. In this order, each at most once per invoice and each in the same run if due:

1. at `noticeDays` or more: a notice of kind `dunning`, `ref` the invoiceId;
2. at `feeDays` or more: a charge, `kind` `lateFee`, `ref` the invoiceId, `quantity` 1, `unitPriceCents` `feeCents`, `discountPercent` 0;
3. at `suspendDays` or more: the member's standing becomes `suspended` (unless it already is, which still marks the step done) and a notice of kind `suspension`, `ref` the invoiceId.

An action is reported per step taken: `invoiceId`, `step` (`notice`, `fee`, `suspend`). Paying the invoice later undoes nothing and reinstates nobody.

### `lockers`

A locker: `lockerId`, `monthlyCents` (1 to 1000000). A rental (`rnt-N`): `rentalId`, `lockerId`, `memberId`, `status` (`active`, `ended`), `startedAt`, `endedAt` nullable.

- `addLocker(lockerId, monthlyCents)` M -/-/all. Refusal `already_exists`. Target: lockerId.
- `rentLocker(memberId, lockerId)` M own/all/all. Refusals `member_suspended`, `locker_taken`, `rental_limit` (the member has 2 active rentals). Target: new rentalId.
- `endLockerRental(rentalId)` M own/all/all. Refusal `invalid_state`. Target: rentalId.
- `billLockers(month)` M -/all/all. Refusals `not_current_month` (the month is not the month of the request time), `already_billed`. One charge per rental active at the request time, ascending `lockerId`: `kind` `locker`, `ref` rentalId, `quantity` 1, `unitPriceCents` the locker's current `monthlyCents`, `discountPercent` 0. Target: month. Result `month`, `charges`.

### `guests`

An allowance per tier: `tier`, `perMonth` (0 to 100), `feeCents` (0 to 1000000). A guest pass (`gst-N`): `guestPassId`, `memberId`, `guestName`, `date`, `charged` (boolean), `status` (`issued`, `revoked`).

- `setGuestAllowance(tier, perMonth, feeCents)` M -/-/all. Target: tier.
- `issueGuestPass(memberId, guestName, date)` M own/all/all. Refusals, in this order: `member_suspended`, `date_in_past` (before the request date), `guests_not_allowed` (the member's tier has no allowance). When the member already holds `perMonth` or more issued passes dated in that month, a charge is created: `kind` `guest`, `ref` the new guestPassId, `quantity` 1, `unitPriceCents` `feeCents`, `discountPercent` 0, and `charged` is true. Target: new guestPassId.
- `revokeGuestPass(guestPassId)` M own/all/all. Refusal `invalid_state` (revoked, or its date is before the request date). No charge is taken back. A revoked pass no longer counts. Target: guestPassId.
- `listGuestPasses(memberId)` R own/all/all. Creation order.

### `volunteering`

A rate: `centsPerHour` (0 to 100000), initially 0. An entry (`vol-N`): `entryId`, `memberId`, `date`, `minutes` (1 to 720), `note`, `status` (`pending`, `approved`, `rejected`), `creditCents` nullable.

- `setVolunteerRate(centsPerHour)` M -/-/all. Target: the fixed string `rate`.
- `logVolunteerTime(memberId, date, minutes, note)` M own/all/all. Refusal `date_in_future`. Target: new entryId.
- `decideVolunteerTime(entryId, decision)` M -/all/all. `decision` `approved` or `rejected`. Refusals, in this order: `invalid_state` (not pending), `approver_not_on_shift` (the actor is staff with no scheduled shift overlapping the request time; an admin is exempt), `volunteer_cap_reached` (only on approval: approved minutes of that member in the month of `date` would exceed 1200). Approval: `creditCents` is `(minutes × centsPerHour + 30) div 60`, added to the member's credit balance. Target: entryId.
- `listVolunteerTime(memberId)` R own/all/all. Creation order.

### `consumables`

A row: `machineId`, `materialId`, `unitsPerHour` (1 to 1000).

- `setConsumption(machineId, materialId, unitsPerHour)` M -/all/all. Replaces a row. Target: `machineId/materialId`.
- `clearConsumption(machineId, materialId)` M -/all/all. Refusal `no_consumption`. Target: `machineId/materialId`.
- `listConsumption(machineId)` R -/all/all. `materialId` ascending.

**Rule at `checkOut`.** For each row, `units` is the metered minutes × `unitsPerHour` divided by 60, rounded up. `drawn` is the smaller of `units` and the stock. The stock decreases by `drawn`. When `drawn` is above 0, a charge: `kind` `material`, `ref` the reservationId, `quantity` `drawn`, `unitPriceCents` the material's current price, `discountPercent` 0, `at` the request time. A shortfall is not an error. No promotion applies. An open stocktake does not stop a draw.

### `access`

A badge (`bdg-N`): `badgeId`, `memberId`, `status` (`active`, `revoked`), `inside` (boolean). A door event (`dor-N`): `eventId`, `kind` (`enter`, `leave`, `guest`), `badgeId`, `guestPassId` nullable, `at`.

- `issueBadge(memberId)` M -/all/all. Refusal `badge_active` (the member has an active badge). Target: new badgeId.
- `revokeBadge(badgeId)` M -/all/all. Refusal `invalid_state` (revoked). A revoked badge is no longer inside. Target: badgeId.
- `enterDoor(badgeId)` M own/all/all. Refusals, in this order: `invalid_state` (revoked), `already_inside`, `member_suspended`, `outside_opening_hours` (the request time is not in the opening period of its date), `account_overdue` (the member's `overdueBalanceCents` is above 0). Target: badgeId.
- `leaveDoor(badgeId)` M own/all/all. Refusals `invalid_state` (revoked), `not_inside`. Target: badgeId.
- `admitGuest(guestPassId, badgeId)` M own/all/all. Ownership is judged on `badgeId`. Refusals, in this order: `invalid_state` (the pass is revoked or the badge is revoked), `pass_not_today` (the pass's date is not the request date), `pass_of_another_member`, `sponsor_not_inside`, `guest_already_admitted`. Target: guestPassId.
- `listDoorEvents(date)` R -/all/all. Events whose `at` lies on that date, creation order.

**Rule:** at `checkIn`, `not_on_premises` when the member has an active badge that is not inside. A member with no active badge is not gated. Check-out is never gated.

### `instructors`

A qualification: `userId`, `familyId`.

- `qualifyInstructor(userId, familyId)` M -/-/all. Refusals `not_eligible` (the user is an admin; or a member with no certification for the family valid on the request date), `already_exists`. Target: `userId/familyId`.
- `withdrawInstructor(userId, familyId)` M -/-/all. Refusals `not_instructor`, `instructor_has_classes` (a scheduled class of that family starts after the request time). Target: `userId/familyId`.
- `listInstructors(familyId)` R all/all/all. `userId` ascending.

### `classes`

A class (`cls-N`): `classId`, `courseId`, `instructorId`, `roomId`, `start`, `end`, `status` (`scheduled`, `cancelled`). Attendance: `classId`, `memberId`, `present` (boolean).

- `scheduleClass(courseId, instructorId, roomId, start, end)` M -/all/all. Refusals, in this order: `invalid_interval`, `starts_in_past`, `not_instructor` (no qualification for the course's family), `room_taken`, `instructor_busy` (another scheduled class of the instructor overlaps). Holds the room. Target: new classId.
- `cancelClass(classId)` M -/all/all. Refusal `invalid_state` (cancelled, or its start is at or before the request time). Releases the room. Target: classId.
- `markAttendance(classId, memberId, present)` M -/all/all. Refusals, in this order: `invalid_state` (cancelled), `class_not_started`, `not_enrolled` (the member has no enrollment in the course, of any status). A later mark replaces. Target: `classId/memberId`.
- `listCourseClasses(courseId)` R all/all/all. `start` ascending, then creation order.
- `listClassAttendance(classId)` R -/all/all. `memberId` ascending.

**Rule:** at `completeCourse`, `attendance_missing` when a class of the course that is scheduled, has started at the request time and began at or after the member's `enrolledAt` has no `present` true mark for the member.

### `assessments`

A definition: `courseId`, `passMark` (0 to 100), `maxAttempts` (1 to 10). An attempt: `courseId`, `memberId`, `number`, `score` (0 to 100), `passed`, `at`.

- `defineAssessment(courseId, passMark, maxAttempts)` M -/all/all. Refusal `already_exists`. Target: courseId.
- `recordAttempt(courseId, memberId, score)` M -/all/all. Refusals, in this order: `no_assessment`, `not_enrolled` (no enrollment with status `enrolled`), `attempts_exhausted` (attempts of this enrollment have reached `maxAttempts`). Target: `courseId/memberId`.
- `listAttempts(courseId, memberId)` R own/all/all. All attempts of all enrollment periods, in order.

**Rule:** at `completeCourse`, `assessment_not_passed` when the course has a definition and no attempt since the member's current `enrolledAt` has `passed` true. A new enrollment after completion starts its attempts at number 1 of a new period; the numbers of a period count from 1.

### `waitlists`

A waiting entry: `courseId`, `memberId`, `joinedAt`. In join order.

- `joinWaitlist(courseId, memberId)` M own/all/all. Refusals, in this order: `member_suspended`, `already_enrolled`, `course_not_full`, `already_waiting`. Target: `courseId/memberId`.
- `leaveWaitlist(courseId, memberId)` M own/all/all. Refusal `not_waiting`. Target: `courseId/memberId`.
- `listWaitlist(courseId)` R -/all/all. Join order.
- `promoteWaitlist(courseId)` M -/all/all. Result `enrollments`, `skipped`. Target: courseId.

**Rule of a promotion.** In join order, while the course has a free place: try to enroll the member exactly as `enrollInCourse` does, with its charge and discounts. On success the entry leaves the list and a notice of kind `waitlist`, `ref` the courseId, is posted. On a refusal the member stays on the list and is reported in `skipped` with the code, and the next is tried.

### `payouts`

A rate per instructor: `userId`, `centsPerHour` (1 to 1000000). A payout (`pyt-N`): `payoutId`, `userId`, `month`, `minutes`, `amountCents`, `status` (`open`, `settled`).

- `setInstructorRate(userId, centsPerHour)` M -/-/all. Refusal `not_instructor` (no qualification at all). Target: userId.
- `generatePayouts(month)` M -/-/all. Refusals `month_not_ended`, `already_generated`. Per instructor, ascending `userId`, with a rate and at least one scheduled class whose `start` lies in the month: `minutes` is the sum of those classes' lengths, `amountCents` `(minutes × centsPerHour + 30) div 60`. Target: month.
- `settlePayout(payoutId)` M -/-/all. Refusal `invalid_state`. Target: payoutId.
- `listPayouts(userId)` R -/all/all. Creation order.

### `events`

An event: `eventId`, `title`, `roomId`, `start`, `end`, `capacity`, `priceCents` (0 to 1000000), `status` (`open`, `cancelled`), `sold`. A ticket (`tkt-N`): `ticketId`, `eventId`, `memberId`, `status` (`valid`, `cancelled`).

- `createEvent(eventId, title, roomId, start, end, capacity, priceCents)` M -/all/all. `capacity` 1 to 1000. Refusals, in this order: `already_exists`, `invalid_interval`, `starts_in_past`, `capacity_too_large` (above the room's), `room_taken`. Holds the room. Target: eventId.
- `sellTicket(eventId, memberId)` M own/all/all. Refusals, in this order: `member_suspended`, `event_closed` (cancelled, or its start is at or before the request time), `already_ticketed` (a valid ticket), `event_sold_out`. Charge: `kind` `event`, `ref` ticketId, `quantity` 1, `unitPriceCents` price, `discountPercent` 0. Target: new ticketId.
- `cancelTicket(ticketId)` M own/all/all. Refusals `invalid_state` (cancelled), `too_late` (less than 1440 minutes before the event's start). The member's credit balance increases by the event's `priceCents`. Target: ticketId.
- `cancelEvent(eventId)` M -/all/all. Refusal `invalid_state` (cancelled, or its start is at or before the request time). Every valid ticket, in creation order, is cancelled with the same credit. The room is released. Target: eventId.
- `listEventTickets(eventId)` R -/all/all. Creation order.

### `hazmat`

A classification: `materialId`, `familyId`, `monthlyLimit` (1 to 10000).

- `classifyHazard(materialId, familyId, monthlyLimit)` M -/-/all. Replaces a classification. Target: materialId.
- `declassifyHazard(materialId)` M -/-/all. Refusal `not_hazardous`. Target: materialId.
- `listHazards()` R all/all/all. `materialId` ascending.

**Rule at `purchaseMaterial`:** `hazard_not_certified` when the member has no certification for the classification's family valid on the request date. `hazard_limit_exceeded` when the units of this material the member bought in the request's month, with this quantity, would exceed `monthlyLimit`. Consumable draws are not purchases.

### `purchasing`

A supplier: `supplierId`, `name`, `status` (`active`, `blocked`). An order (`po-N`): `orderId`, `supplierId`, `status` (`draft`, `submitted`, `approved`, `received`), `createdBy`, `totalCents`, `lines` in the order added, each `materialId`, `quantity` (1 to 100000), `unitCostCents` (1 to 1000000), `receivedQuantity`.

- `addSupplier(supplierId, name)` M -/all/all. Refusal `already_exists`. Target: supplierId.
- `setSupplierStatus(supplierId, status)` M -/-/all. Target: supplierId.
- `createOrder(supplierId)` M -/all/all. Refusal `supplier_blocked`. Target: new orderId.
- `addOrderLine(orderId, materialId, quantity, unitCostCents)` M -/all/all. Refusals `invalid_state` (not draft), `duplicate_line`. Target: orderId.
- `submitOrder(orderId)` M -/all/all. Refusals `invalid_state`, `empty_order`. Target: orderId.
- `approveOrder(orderId)` M -/all/all. Refusals, in this order: `invalid_state` (not submitted), `self_approval` (the actor created it), `approval_requires_admin` (`totalCents` over 100000 and the actor is staff), `supplier_blocked`. Target: orderId.
- `receiveOrder(orderId, materialId, quantity)` M -/all/all. Quantity 1 to 100000. Refusals, in this order: `invalid_state` (not approved), `not_on_order`, `over_receipt`, `stocktake_open`. The material's stock increases. When every line is fully received the status becomes `received`. Target: orderId.
- `listOrders(supplierId)` R -/all/all. Creation order.

### `stocktake`

A stocktake (`stk-N`): `stocktakeId`, `status` (`open`, `closed`), `counts` (ascending `materialId`), each `materialId`, `counted` (0 to 100000000), and on closing `difference`.

- `startStocktake()` M -/all/all. Refusal `stocktake_running`. Target: new stocktakeId.
- `recordCount(stocktakeId, materialId, counted)` M -/all/all. Refusal `invalid_state` (closed). A later count replaces. Target: `stocktakeId/materialId`.
- `closeStocktake(stocktakeId)` M -/all/all. Refusals `invalid_state`, `nothing_counted`. Per count, ascending `materialId`: `difference` is `counted − stock` at the request time, and the stock becomes `counted`. Target: stocktakeId.
- `getStocktake(stocktakeId)` R -/all/all.

**Rule:** `stocktake_open` at `purchaseMaterial`, `restockMaterial` and `receiveOrder` when the material has a count in an open stocktake.

### `workorders`

A work order (`wko-N`): `workOrderId`, `machineId`, `summary`, `status` (`open`, `closed`), `incidentId` nullable, `lockoutId`, `partsCostCents`, `openedBy`, `openedAt`, `closedAt` nullable, `note` nullable.

- `openWorkOrder(machineId, summary)` M -/all/all. Applies a lockout of source `workOrder`. Target: new workOrderId.
- `linkIncident(workOrderId, incidentId)` M -/all/all. Refusals, in this order: `invalid_state` (closed), `incident_other_machine`, `already_linked`. Target: workOrderId.
- `useParts(workOrderId, partId, quantity)` M -/all/all. Quantity 1 to 1000. Refusals, in this order: `invalid_state` (closed), `part_wrong_family`, `insufficient_parts`. Stock decreases; `partsCostCents` grows by quantity × the part's `unitCostCents`. Target: workOrderId.
- `closeWorkOrder(workOrderId, note)` M -/all/all. Refusals `invalid_state`, `incident_open` (a linked incident is not resolved). Releases its lockout. Target: workOrderId.
- `listWorkOrders(machineId)` R -/all/all. Creation order.

### `escalations`

A policy per family: `familyId`, `minSeverity`. An escalation (`esc-N`): `escalationId`, `incidentId`, `machineId`, `workOrderId`, `cancelledReservationIds`, `at`.

- `setEscalationPolicy(familyId, minSeverity)` M -/-/all. Replaces. Target: familyId.
- `escalateIncident(incidentId)` M -/all/all. Refusals, in this order: `invalid_state` (resolved), `already_escalated`, `no_policy` (the machine's family has none), `below_policy` (severity under `minSeverity`; `low` < `medium` < `high`). Target: new escalationId.
- `listEscalations()` R -/all/all. Creation order.

**Effects of an escalation, in this order.** A work order is opened with the summary `Escalated incident <incidentId>`, linked to the incident, with its lockout; the opener is the actor. Every reservation of the machine with reported status `reserved` and `start` after the request time, ascending by `start`, then creation order, gets status `cancelled` and its member a notice of kind `cancellation`, `ref` the reservationId. A session checked in is not touched.

### `projects`

A project: `projectId`, `ownerId`, `title`, `collaboratorIds` (ascending), `status` (`open`, `closed`), `reservationIds`, `purchaseIds` (in the order attached).

- `createProject(projectId, ownerId, title)` M own/all/all. Ownership is judged on `ownerId`. Refusal `already_exists`. Target: projectId.
- `addCollaborator(projectId, memberId)` M own/all/all. Refusals `project_closed`, `already_collaborator` (also the owner). Target: `projectId/memberId`.
- `attachReservation(projectId, reservationId)` M own/all/all. Ownership on `projectId`. Refusals, in this order: `project_closed`, `not_project_member` (the reservation's member is neither owner nor collaborator), `already_attached` (to any project). Target: `projectId/reservationId`.
- `attachPurchase(projectId, purchaseId)` M own/all/all. As `attachReservation`. Target: `projectId/purchaseId`.
- `closeProject(projectId)` M own/all/all. Refusal `invalid_state`. Target: projectId.
- `getProjectCost(projectId)` R own/all/all. `usageCents`: the sum of `amountCents` of attached reservations with status `completed`. `materialCents`: the sum over attached purchases. `totalCents`. `byMember`: per member with a share, ascending `memberId`, `usageCents`, `materialCents`.

### `quotes`

A customer: `customerId`, `name`. A quote (`quo-N`): `quoteId`, `customerId`, `title`, `status` (`draft`, `sent`, `accepted`, `rejected`), `lines` in order added, each `lineKind` (`machine`, `material`), `ref`, `quantity` (1 to 100000), `unitPriceCents` nullable until sent; `subtotalCents`, `totalCents`, `sentAt`, `validUntil` (nullable until sent).

- `addCustomer(customerId, name)` M -/all/all. Refusal `already_exists`. Target: customerId.
- `createQuote(customerId, title)` M -/all/all. Target: new quoteId.
- `addQuoteLine(quoteId, lineKind, ref, quantity)` M -/all/all. `ref` is an identifier. Refusals, in this order: `invalid_state` (not draft), `unknown_line_ref` (no such machine or material). Target: quoteId.
- `sendQuote(quoteId)` M -/all/all. Refusals `invalid_state`, `empty_quote`. Prices are captured: a machine's current rate, a material's current price. `subtotalCents` is the sum of quantity × unit price. `totalCents` is `(subtotalCents × 120 + 50) div 100`. `validUntil` is 30 days after the date of the request time. Target: quoteId.
- `decideQuote(quoteId, decision)` M -/all/all. `decision` `accepted` or `rejected`. Refusals `invalid_state` (not sent), `quote_expired` (only on acceptance: the request date is after `validUntil`). Target: quoteId.
- `getQuote(quoteId)` R -/all/all.

### `jobs`

A job (`job-N`): `jobId`, `quoteId`, `customerId`, `makerId`, `status` (`started`, `completed`), `totalCents`, `makerCreditCents` nullable, `paidCents`.

- `startJob(quoteId, makerId)` M -/all/all. Refusals, in this order: `quote_not_accepted`, `already_started`, `member_suspended`, `maker_not_certified` (no certification valid on the request date for the family of some machine line). Target: new jobId.
- `completeJob(jobId)` M -/all/all. Refusal `invalid_state`. `makerCreditCents` is `(totalCents × 30 + 50) div 100`, added to the maker's credit balance. Target: jobId.
- `recordJobPayment(jobId, amountCents)` M -/all/all. Refusals `job_not_completed`, `overpayment` (`paidCents` plus the amount would exceed `totalCents`). Target: jobId.
- `listJobs(customerId)` R -/all/all. Creation order.

### `compliance`

- `checkEligibility(memberId, machineId, date)` R own/all/all. `eligible` (boolean), `blockers` (list of codes). The blockers that hold, judged as `createReservation` would judge a slot starting on `date` at the request time, in the order of its refusals: `member_suspended`, `guardian_consent_missing`, `not_certified`, `waiver_missing`, `machine_locked_out`, `inspection_due`. Slot-bound rules are not judged. `eligible` is true when the list is empty.
- `complianceSummary(memberId)` R own/all/all. `standing`; `minor` (boolean, on the request date); `families`: one row per family, ascending `familyId`, with `certified`, `consent` (true when not a minor, or consent held), `waiverSigned` (true when no waiver), `ppeHeld` (true when every required item is held).

### `finance`

- `chargesByKind(month)` R -/all/all. One row per kind in the order `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event`: `kind`, `invoicedCents` (the sum of line amounts of that kind on the month's invoices), `refundedCents` (the sum of refunds on those lines). `totalCents`.
- `financeSummary(month)` R -/-/all. `invoicedCents`, `refundedCents`, `paymentsCents` (as `revenueReport`), `creditsIssuedCents`, `journalEntries` (count in the month), `trialBalanced` (boolean).
- `agedReceivables(date)` R -/all/all. Open invoices at the request time by days overdue on `date`: rows `invoiceId`, `memberId`, `balanceCents`, `daysOverdue` (whole days from the date of `dueAt` to `date`, 0 when not yet due), ascending by number. Buckets: `currentCents` (0), `days1to30Cents`, `days31to60Cents`, `over60Cents`.
- `householdStatement(householdId, month)` R -/all/all. Per household member, ascending: `memberId`, `invoiceId` nullable, `totalCents`, `balanceCents`. Sums `totalCents`, `balanceCents`.

## Agreed page names

So that pages of different authors can link each other before they exist:

- Interface: `<module>/<module>-interface`. Operation: `<module>/<wire name in kebab case>`, for example `ledger/get-trial-balance`. Record: `<module>/<record name in kebab case>`, for example `lockouts/lockout`, `passes/pass`, `passes/pass-product`, `ledger/journal-entry`.
- Rules that other modules link:

| Page | Holds |
|---|---|
| `ledger/postings` | The postings table and the invariants |
| `guardians/guardian-consent` | What a minor is and the consent gate |
| `waivers/waiver-gate` | The signature gate |
| `ppe/ppe-gate` | The check-in gate |
| `lockouts/lockout-gate` | The gate at reservation and check-in |
| `inspections/inspection-gate` | The due-date gate |
| `staffing/supervision` | Coverage and `shift_needed` |
| `passes/pass-draw` | The draw at check-out |
| `consumables/consumption` | The draw of stock at check-out |
| `access/premises-gate` | The check-in gate |
| `classes/attendance-gate` | The completion gate |
| `assessments/assessment-gate` | The completion gate |
| `hazmat/hazard-gate` | The purchase gate |
| `stocktake/stocktake-freeze` | The freeze of counted materials |
| `promotions/redeemed-promotions` | What "oldest unused" means and how one is used |
| `scholarships/course-discount` | Award against promotion at enrollment |
| `households/household-discount` | The membership line's discount |
| `dunning/dunning-run` | The rule of a run |
| `waitlists/waitlist-promotion` | The rule of a promotion |
| `escalations/escalation-effects` | The effects of an escalation |

## The form of pages

A new module has the same form as an old one:

- `<module>/<module>-interface.md`, kind `Interface`.
- One page per operation, kind `Operation`, file named after the operation in kebab case.
- One page per record, kind `std/Concept`.
- One page per rule, kind `std/Rule`.
- Enumerations and value types where a module has its own.

A rule that binds another module links it, and the operation page lists it under "Rules of other modules that apply here". Registry page: `registry/<module>.md` with `paths: realization/<module>/**`.

Pages follow the existing Makerspace pages word for word in their structure, because `pages.py` parses them.

## Cases

Three classes of hidden case, on top of Makerspace's 54, which stay:

- **G:** one generated case per new operation: arguments, permissions, referenced records, no audit entry on a refusal. Written from the parsed pages.
- **B:** written by hand from the requirements document only, one or more per module: effects, refusal orders, boundaries, result shapes.
- **X:** cross-module chains and invariants: the reservation and check-in gates, `checkOut` with passes and consumables, the money chain through the ledger, dunning through suspension to the gates, escalation, the waitlist through discounts.

## What is not here

Change steps, plants and the other arms. They come after the probe. This project is public and not counted.
