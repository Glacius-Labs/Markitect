# Makerspace requirements: large (fresh build)

This document describes the complete product for the first delivery.

How to read an operation in this document. Its text states the operation's effect. A list below it states: its wire name, the exact `op` value, and whether the operation is a mutation, which changes state and is audited, or a read; its arguments besides `actor` and `at`, which every operation takes, all of them required; an argument that names an existing record must name one that exists (check 5 of the order of checks); its refusals, its own, checked at check 7 of the order of checks in the order listed, the first that applies deciding the code; its result, the members of a successful `result`, each with its type; its permissions, for each role `all` or `own`, and a role that is not listed may not use the operation; and for a mutation its audit target, the `target` of its audit entry: the name of an argument or of a new identifier whose value is the target, `a/b` joining two values with a slash, for example `laser-101/ada` for course `laser-101` and member `ada`, and `month` being the month string, for example `2026-03`.

What a module's interface states is what other modules may use of it: its operations, its records, the refusal codes it adds and the capabilities other modules ask of it that are not operations. Other modules use nothing else of it; the module's other pages are internal to it, except its rules that bind other modules. It also names the interfaces of the modules it uses.

## Overview

Makerspace runs the day-to-day operations of a fictional community workshop. It is one local program. Front-desk tools send it requests on behalf of the person who acts; the program decides, records and answers.

- **Members** join on a membership tier and pay a monthly fee that depends on the tier. Staff can suspend a member; a suspended member is not in good standing.
- **Training:** courses certify members for a tool family, such as laser cutters. A certification is valid for 365 days.
- **Equipment:** machines belong to a tool family and have a rate per minute. Staff schedule maintenance windows during which a machine cannot be reserved.
- **Bookings:** members reserve machines in 30-minute steps inside opening hours, check in when they start and check out when they finish. They are charged for every metered minute that no pass covers, at the machine's rate, minus a tier discount.
- **Materials:** a small shop sells materials from stock.
- **Billing:** once a month staff generate invoices: one per member, with the monthly fee and all charges of that month. An invoice is due 14 days after it is issued. Members pay their invoices; an overpayment becomes credit, and credit reduces the next invoice.
- **Safety:** staff record safety incidents per machine.
- **Reporting:** staff and admins read utilization and revenue reports.
- **Ledger:** every movement of money is posted to a double-entry journal, which admins read with its trial balance.
- **Notices:** the program keeps notices for members, who read and acknowledge them.
- **Households:** members can form a household with one payer; a household of three or more pays less for the monthly fee.
- **Guardians and waivers:** a minor needs a guardian's consent for a tool family, and a family can require a signed waiver of its current version.
- **Protective equipment:** a family can require items that a member must hold to check in.
- **Lockouts and inspections:** a machine can be locked out, and a machine with an inspection plan cannot be reserved for its due date or later.
- **Staffing:** staff work shifts, and a family can require a supervisor on shift for a whole reservation.
- **Parts, work orders and escalations:** staff repair machines with spare parts under work orders. An incident can be escalated, which locks the machine and cancels what is booked on it.
- **Rooms, instructors and classes:** courses can have classes in rooms, taught by qualified instructors. A course can require attendance and a passed assessment, and it can have a waiting list. Instructors are paid per month.
- **Events:** staff run ticketed events in rooms.
- **Deposits, passes and lockers:** members leave deposits, buy passes of prepaid machine minutes and rent lockers.
- **Promotions and scholarships:** a redeemed promotion or a scholarship award lowers a course or material charge.
- **Refunds and dunning:** admins refund invoice lines as credit, and an explicit run moves overdue invoices through notice, late fee and suspension.
- **Guests, access and volunteering:** members bring guests, and badges open the door. Approved volunteer time becomes credit.
- **Consumables and hazardous materials:** a machine can draw material from stock while it runs, and hazardous materials need a certification and have a monthly limit.
- **Purchasing and stocktake:** staff order materials from suppliers and count stock.
- **Projects, quotes and jobs:** members attach their work to projects, and the workshop quotes and carries out jobs for outside customers.
- **Compliance and finance:** staff read whether a member may use a machine, and financial summaries.

Four policies apply to every operation: authorization, audit, money and time.

Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, the sending of notifications (a notice is a record in the program), scheduled jobs, deletion of records and integration with other systems.

The Go module lives at `realization/go.mod`; the program's main package lives at `realization/cmd/makerspace`. Only the Go standard library is used. The program is divided into the modules of its architecture; each module is one Go package under `realization/`, named after the module, for example `realization/billing`; `realization/protocol` and `realization/cmd/makerspace` belong to the protocol module.

The program is written in Go. From `realization/`, `go build ./cmd/makerspace` builds it and `go test ./...` runs its tests.

The modules are described in the architecture: the area modules, each with its own folder here, the outer layer protocol, and the shared policies module, which this folder describes: values, refusal codes, roles and the project-wide rules.

## Architecture

The program is divided into the modules below, with one-way dependencies; the owner decided this module cut on 27 October 2026. Each module is one Go package under `realization/`, named after the module, for example `realization/billing`. A module may depend only on itself and on the modules listed for it; the Go standard library is not counted.

| Module | Model | May depend on |
| --- | --- | --- |
| `policies` | this folder, the project itself | nothing |
| `members` | members | `policies` |
| `equipment` | equipment | `policies` |
| `ledger` | ledger | `policies` |
| `notices` | notices | `members`, `policies` |
| `households` | households | `members`, `policies` |
| `guardians` | guardians | `members`, `equipment`, `policies` |
| `waivers` | waivers | `members`, `equipment`, `guardians`, `policies` |
| `ppe` | ppe | `members`, `equipment`, `policies` |
| `lockouts` | lockouts | `equipment`, `policies` |
| `inspections` | inspections | `equipment`, `lockouts`, `policies` |
| `staffing` | staffing | `members`, `equipment`, `policies` |
| `parts` | parts | `equipment`, `policies` |
| `rooms` | rooms | `policies` |
| `billing` | billing | `members`, `ledger`, `households`, `policies` |
| `deposits` | deposits | `members`, `billing`, `ledger`, `policies` |
| `passes` | passes | `members`, `billing`, `policies` |
| `promotions` | promotions | `members`, `policies` |
| `scholarships` | scholarships | `members`, `policies` |
| `refunds` | refunds | `members`, `billing`, `ledger`, `policies` |
| `dunning` | dunning | `members`, `billing`, `notices`, `policies` |
| `lockers` | lockers | `members`, `billing`, `policies` |
| `guests` | guests | `members`, `billing`, `policies` |
| `volunteering` | volunteering | `members`, `billing`, `ledger`, `staffing`, `policies` |
| `training` | training | `members`, `equipment`, `billing`, `guardians`, `promotions`, `scholarships`, `policies` |
| `materials` | materials | `members`, `billing`, `promotions`, `policies` |
| `consumables` | consumables | `equipment`, `materials`, `billing`, `policies` |
| `bookings` | bookings | `members`, `training`, `equipment`, `billing`, `guardians`, `waivers`, `ppe`, `lockouts`, `inspections`, `staffing`, `passes`, `consumables`, `policies` |
| `safety` | safety | `equipment`, `policies` |
| `reporting` | reporting | `equipment`, `bookings`, `billing`, `policies` |
| `access` | access | `members`, `billing`, `guests`, `bookings`, `policies` |
| `instructors` | instructors | `members`, `equipment`, `training`, `policies` |
| `classes` | classes | `members`, `training`, `instructors`, `rooms`, `policies` |
| `assessments` | assessments | `members`, `training`, `policies` |
| `waitlists` | waitlists | `members`, `training`, `notices`, `policies` |
| `payouts` | payouts | `members`, `ledger`, `instructors`, `classes`, `policies` |
| `events` | events | `members`, `rooms`, `billing`, `ledger`, `policies` |
| `hazmat` | hazmat | `equipment`, `materials`, `training`, `policies` |
| `purchasing` | purchasing | `materials`, `ledger`, `policies` |
| `stocktake` | stocktake | `materials`, `ledger`, `policies` |
| `workorders` | workorders | `members`, `equipment`, `safety`, `parts`, `lockouts`, `policies` |
| `escalations` | escalations | `equipment`, `safety`, `lockouts`, `bookings`, `workorders`, `notices`, `policies` |
| `projects` | projects | `members`, `bookings`, `materials`, `policies` |
| `quotes` | quotes | `equipment`, `materials`, `policies` |
| `jobs` | jobs | `members`, `equipment`, `billing`, `ledger`, `training`, `quotes`, `policies` |
| `compliance` | compliance | `members`, `equipment`, `guardians`, `waivers`, `ppe`, `training`, `lockouts`, `inspections`, `policies` |
| `finance` | finance | `members`, `ledger`, `billing`, `households`, `refunds`, `policies` |
| `protocol` | protocol | `policies`, `members`, `equipment`, `ledger`, `notices`, `households`, `guardians`, `waivers`, `ppe`, `lockouts`, `inspections`, `staffing`, `parts`, `rooms`, `billing`, `deposits`, `passes`, `promotions`, `scholarships`, `refunds`, `dunning`, `lockers`, `guests`, `volunteering`, `training`, `materials`, `consumables`, `bookings`, `safety`, `reporting`, `access`, `instructors`, `classes`, `assessments`, `waitlists`, `payouts`, `events`, `hazmat`, `purchasing`, `stocktake`, `workorders`, `escalations`, `projects`, `quotes`, `jobs`, `compliance`, `finance` |

- `policies`: The policies module depends on no other module of the project; every other module may depend on it.
- `equipment`: Where maintenance must know about reservations, equipment declares what it needs and the protocol module supplies it from bookings, so equipment never depends on bookings.
- `staffing`: Where staffing must know something of `bookings`, staffing declares what it needs and the protocol module supplies it, so staffing never depends on bookings.
- `training`: Where training must know something of `classes` and `assessments`, training declares what it needs and the protocol module supplies it, so training never depends on them.
- `materials`: Where materials must know something of `hazmat` and `stocktake`, materials declares what it needs and the protocol module supplies it, so materials never depends on them.
- `bookings`: Where bookings must know something of `access`, bookings declares what it needs and the protocol module supplies it, so bookings never depends on access.
- `instructors`: Where instructors must know something of `classes`, instructors declares what it needs and the protocol module supplies it, so instructors never depends on classes.
- `protocol`: The protocol module is the outer layer: it reads requests, runs the checks every operation shares and calls the area modules. It may depend on every other module; no module depends on it.

**Placements.** Some project-wide rules are realized once, in one module:

- The protocol module appends the audit entry after a mutation is answered `ok`, using the operation's audit target; area modules do not write audit entries.
- Checks 2, 4 and 6 of the order of checks run once, in the protocol module, before any operation's own rules; area modules do not check roles or ownership themselves. An operation's own refusals at check 7 may still look at who the actor is, where the operation says so, such as `releaseLockout` and `approveOrder`; the protocol module passes the actor's identifier and role to the area module with the request, so neither depends on `members`.
- The protocol module keeps the request-time order (check 3) and passes the request time to the area modules as their only notion of now.
- The protocol module realizes the whole order of checks, including the argument checks of argument values (check 1) and the referenced records (check 5). These rules sit in the project itself so that every module reads them, not because the `policies` module realizes them.

**Layout.** `realization/go.mod`, `realization/README.md` and the `policies` package `realization/policies/` belong to the project itself; `realization/protocol/` and `realization/cmd/makerspace/` belong to protocol; every other module owns `realization/<module>/`.

## Rules every module follows

### Order of checks

A well-formed request is checked in the following order. The first check that fails decides the refusal and its code.

1. **Arguments:** every argument is present and valid by argument values → otherwise `invalid_argument`.
2. **Actor:** `actor` identifies an existing user of any role → otherwise `unknown_actor`.
3. **Request time:** `at` is not earlier than the request time of the most recent request that was answered `ok` (time) → otherwise `time_regression`.
4. **Role permission:** the actor's role may use the operation at all, as "all" or "own" in the operation's permissions → otherwise `forbidden`.
5. **Referenced records:** every argument that refers to a record names one that exists, checked in the order the operation lists its arguments → otherwise `not_found`. An argument that names a member must name a user whose role is `member`; the identifier of a staff or admin user does not name a member.
6. **Ownership:** when the actor's permission for the operation is "own", the record concerned belongs to the actor (authorization) → otherwise `forbidden`.
7. **Operation rules:** the refusals listed for the operation, in the order listed.

When a check fails, the response is `status: "refused"` with the code and a reason, and nothing else happens: no record is created or changed, no identifier number is consumed, no audit entry is written and the request-time order is not advanced. When every check passes, all of the operation's effects take place together and the response is `status: "ok"`.

### Argument values

Each argument has a field type: an identifier, text, timestamp, date or month as their definitions state, a flag (a JSON `true` or `false`), an enumeration (a JSON string equal to one of the listed values, compared exactly), or an integer.

An integer is a JSON number written as an optional minus sign followed by digits, without a fraction part or exponent (`10.0` and `1e3` are not integers), within the range its type states. Every range in this document starts at 0 or higher, so any value written with a minus sign, including `-0`, is out of range.

An argument that is missing, `null`, of the wrong JSON type or outside its format or range is refused with `invalid_argument`. Each argument is checked on its own. Relationships between arguments, such as an end that is not after a start, and relationships to stored state are checked later and have their own codes.

These formats and ranges apply to arguments, including `at`. An integer argument outside its range is out of range however many digits it has. Values the program computes can lie outside these ranges; computed values states the rules for those values.

`actor` is an identifier argument and `at` is a timestamp argument of every operation. An `actor` that is missing, `null`, not a string or not a valid identifier, such as `"bad id!"` or a string of 65 characters, is refused with `invalid_argument` at check 1, like any other argument. Only a valid identifier that names no user is refused with `unknown_actor`, at check 2.

### Values

The formats of the values that requests, records and results carry. Arguments are checked against them as argument values states.

| Value | JSON type and range | Meaning |
| --- | --- | --- |
| identifier | string | A JSON string of 1 to 64 characters from `A`–`Z`, `a`–`z`, `0`–`9`, `.`, `_` and `-`, whose first character is a letter or digit. Identifiers are compared exactly and are case-sensitive: `Ada` and `ada` are different identifiers. |
| text | string | A JSON string that contains at least one character without the Unicode `White_Space` property. It is stored and returned exactly as supplied, without trimming. |
| timestamp | string | A JSON string `YYYY-MM-DDTHH:MMZ` naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`, for example `2026-03-02T09:00Z`. Hours are `00` to `23`, minutes `00` to `59`; there are no seconds. |
| date | string | A JSON string `YYYY-MM-DD` naming a valid calendar date from `2000-01-01` to `2099-12-31`. `2026-02-29` is not valid; `2028-02-29` is. |
| month | string | A JSON string `YYYY-MM` from `2000-01` to `2099-12`. |
| clock time | string | A time of day `HH:MM`. |
| count | integer at least `0` | A count, quantity or sequence number; never negative. |
| cent amount | integer at least `0`, in cents | An amount of money; never negative. |
| minute count | integer at least `0`, in minutes | A number of minutes; never negative. |
| percent | integer at least `0` | A discount in percent. |
| flag | boolean | A yes-or-no value. |
| operation name | string | The name of an operation, as in a request's `op`. |
| audit target | string | A plain string that identifies the record an operation acted on. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format. |

Units: cents (`ct`): One cent; all amounts are integer numbers of cents; minute (`min`): One minute of time.

### Refusal codes

The refusal codes every module shares. Each area module's interface lists the codes it adds. An operation may answer a code that another module's interface lists when its own refusals name it; the meaning stated there applies unless the operation states another.

| Code | Meaning |
| --- | --- |
| `already_exists` | A record with the supplied identifier already exists. |
| `forbidden` | The actor's role may not use the operation, or may use it only for the actor's own records and the record concerned belongs to another member. |
| `invalid_argument` | An argument is missing, `null`, of the wrong type, or outside its format or range. |
| `invalid_interval` | An interval's end is not after its start, or a report range is longer than allowed. |
| `invalid_state` | The record is not in a state that allows the operation. |
| `not_found` | A referenced record does not exist. |
| `starts_in_past` | An interval starts before the request time. |
| `time_regression` | `at` is earlier than the request time of the latest request answered `ok`. |
| `unknown_actor` | `actor` does not identify a user. |

### Authorization

Every user has exactly one role, fixed when the user is registered: member, staff or admin.

For each operation and role, the operation's permissions state one permission:

- **all:** the role may use the operation for any record.
- **own:** the role may use the operation only for records that belong to the actor.
- **none** (no permission listed for the role): the role may not use the operation.

A record concerned belongs to the actor when:

- the operation has a `memberId` argument and it equals the actor's identifier;
- the operation has a `reservationId` argument and the reservation's member is the actor;
- the operation has an `invoiceId` argument and the invoice's member is the actor;
- the operation has a `depositId`, `rentalId`, `guestPassId`, `badgeId`, `noticeId`, `ticketId` or `entryId` argument and that deposit, locker rental, guest pass, badge, notice, event ticket or volunteer time entry is the actor's;
- the operation has a `guardianId` argument and it equals the actor's identifier; where an operation has `guardianId`, ownership is judged on it alone;
- the operation has an `ownerId` argument and it equals the actor's identifier;
- the operation has a `projectId` argument and the project's owner is the actor; where an operation has `projectId`, ownership is judged on it alone, except create project, whose ownership is judged on `ownerId` alone;
- an operation that says on which of its arguments ownership is judged, such as `admitGuest` on `badgeId`, is judged on that argument alone.

A "none" permission is refused at check 4, an ownership violation at check 6 of the order of checks. Both use the code `forbidden`.

The roles:

- `member`: Members are the workshop's customers.
- `staff`: Staff run the front desk and the workshop floor.
- `admin`: Admins also manage the catalog, credits and users.

### Audit

Operations are either mutations or reads, as each operation states. Every request to a mutation that is answered `ok` appends exactly one audit entry, even when the operation changes no data, such as setting a member's standing to its current value. Reads, refusals and invalid requests never append an entry. Entries are never changed or removed.

An audit entry has these fields: `seq`, the entry's position in the log, starting at 1 and increasing by 1 for each entry; `at`, the request time; `actor`, the acting user's identifier; `op`, the operation name; `target`, a string that identifies the record the operation acted on, as each mutation's audit target states.

Admins read the log with `listAuditEntries`.

### Audit entry

One entry of the audit log. Entries are never changed or removed.

**Fields**, in the formats of values:

- `seq`: count
- `at`: timestamp
- `actor`: identifier
- `op`: operation name
- `target`: audit target

### Time

- All times are UTC with a precision of one minute. Timestamps, dates and months use the formats timestamp, date and month. The day of the week of a timestamp is that of its UTC date. The date of a timestamp is its UTC calendar date.
- **Request time:** `at` is the time at which the request happens. It is the program's only notion of "now". The request time of every request must not be earlier than the request time of the most recent request answered `ok`; an equal time is allowed. Refused and invalid requests do not move this point. Before the first `ok` response, any valid `at` is accepted.
- **Intervals are half-open.** An interval `[start, end)` contains its start minute and excludes its end minute. Two intervals `[a, b)` and `[c, d)` overlap exactly when `a < d` and `c < b`; intervals that only touch, such as 10:00–11:00 and 11:00–12:00, do not overlap. The length of an interval in minutes is `end − start`.
- **Day and date arithmetic:** a day is 1440 minutes; "N days after timestamp t" is `t + N × 1440` minutes. "N days after date d" is the calendar date N days later, so `2027-03-01` plus 365 days is `2028-02-29`.
- **Durations.** A reservation lasts at most 240 minutes. Maintenance windows and metered sessions have no maximum length apart from the timestamp range.
- **States that depend on time** are evaluated at the request time of the request that reads or uses them: a reservation that is `missed`, a certification that is `expired` and an invoice that is `overdue`.

### Computed times

Request times and time arguments lie between 2000 and 2099, but some values the program computes from them lie later. Integers grow with the number of requests. This rule and integer limits, identifier limits and request limits cover every computed value.

- **Times after 2099.** A computed timestamp or date can lie after 2099. It uses the same format as its type, with a four-digit year, and the Gregorian calendar continues as usual: 2100 is not a leap year, so `2099-03-01` plus 365 days is `2100-03-01`. The latest values a result can show are a `dueAt` of `2100-01-14T23:59Z` (14 days after `2099-12-31T23:59Z`) and a certification's `expiresOn` of `2100-12-31` (365 days after `2099-12-31`). A quote's `validUntil` lies at the latest on `2100-01-30`. A pass's `expiresOn` and an inspection plan's `nextDueOn`, at most 3650 days after a date of 2099, lie at the latest on `2109-12-29`. Points used only in comparisons can also lie after 2099: the first minute after the month `2099-12` and after the date `2099-12-31` are both `2100-01-01T00:00Z`. No computed time lies before `2000-01-01T00:00Z`.
- **The end of the calendar.** No request time is later than `2099-12-31T23:59Z`. The following consequences follow from the rules of this document and are stated so that they are read only one way:
  - `2099-11` is the last month that can be invoiced. `generateInvoices` for `2099-12` is never accepted: when it reaches the operation rules (check 7), it is refused with `month_not_ended`. The revenue report for `2099-12` always has `invoicesGenerated` `false`.
  - An invoice whose `dueAt` lies after 2099 is never overdue. A certification whose `expiresOn` lies after 2099 is never reported `expired`.
  - The last day a utilization report can include is `2099-12-30`, because `to` is at most `2099-12-31` and is excluded from the range.
  - A maintenance window ends at the latest at `2099-12-31T23:59Z`, so the last minute of 2099 cannot be covered. Reservations end by closing time and are not affected.

### Money

- All amounts are integer numbers of cents, represented as JSON integers. There are no fractions of a cent, no floating-point amounts and no currency field. Stored and returned amounts are never negative.
- Every charge is a line with a `quantity`, a `unitPriceCents` and a `discountPercent`. Its amount is
  `amountCents = (quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`,
  where `div` is integer division. This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again.
  Example: 37 minutes at 45 cents with a 10 % discount: `(37 × 45 × 90 + 50) div 100 = 149900 div 100 = 1499` cents.
- A charge's amount is fixed when the charge is created. Later changes to a rate, a tier or a price never change existing charges.
- Each member has a credit balance, initially 0. It increases by credits issued by an admin, by overpayments and by the credit other modules add through billing: a released deposit, a refund, approved volunteer time, a cancelled ticket and a completed job. It decreases only when credit is applied to a new invoice. It is never negative. Credit is never paid out.

### List order

Identifiers are ordered by ASCII byte value (so `-` < `.` < digits < uppercase letters < `_` < lowercase letters, and `M10` < `M9`). Listed results are ordered as follows:

| List | Order |
| --- | --- |
| `listCertifications` → `certifications` | `familyId` ascending |
| `listMachines` → `machines` | `machineId` ascending |
| `getMachineSchedule` → `reservations` | `start` ascending, then creation order |
| `getMachineSchedule` → `maintenance` | `start` ascending, then creation order |
| `listMemberReservations` → `reservations` | `start` ascending, then creation order |
| `listMaterials` → `materials` | `materialId` ascending |
| `generateInvoices` → `invoices` | `memberId` ascending |
| `listInvoices` → `invoices` | `month` ascending |
| invoice → `lines` | the membership line first, then the other lines by their `at` ascending, then in the order the charges were created |
| `listIncidents` → `incidents` | creation order |
| `utilizationReport` → `machines` | `machineId` ascending |
| `listAuditEntries` → `entries` | `seq` ascending |
| `listJournalEntries` → `entries` | creation order |
| `getTrialBalance` → `rows` | the listing order of the ledger accounts |
| `listNotices` → `notices` | creation order |
| household → `memberIds` | ascending |
| guardianship → `consentFamilyIds` | ascending |
| `listMemberWaivers` → `signatures` | `familyId` ascending |
| `listMemberPpe` → `items` | `itemId` ascending |
| `listLockouts` → `lockouts` | creation order |
| `listInspectionsDue` → `plans` | `machineId` ascending |
| `listShifts` → `shifts` | `start` ascending, then creation order |
| `listParts` → `parts` | `partId` ascending |
| `getRoomSchedule` → `holds` | `start` ascending, then creation order |
| `listDeposits` → `deposits` | creation order |
| `listMemberPasses` → `passes` | creation order |
| `listMemberPromotions` → `redemptions` | creation order |
| `listAwards` → `awards` | creation order |
| `listRefunds` → `refunds` | creation order |
| `runDunning` → `actions` | invoices by ascending number, the steps of one invoice in the order notice, fee, suspend |
| `listDunningCases` → `cases` | creation order |
| `billLockers` → `charges` | `lockerId` ascending |
| `listGuestPasses` → `guestPasses` | creation order |
| `listVolunteerTime` → `entries` | creation order |
| `listConsumption` → `rows` | `materialId` ascending |
| `listDoorEvents` → `events` | creation order |
| `listInstructors` → `instructors` | `userId` ascending |
| `listCourseClasses` → `classes` | `start` ascending, then creation order |
| `listClassAttendance` → `attendance` | `memberId` ascending |
| `listAttempts` → `attempts` | the order the attempts were recorded |
| `listWaitlist` → `entries` | join order |
| `promoteWaitlist` → `enrollments`, `skipped` | join order |
| `generatePayouts` → `payouts` | `userId` ascending |
| `listPayouts` → `payouts` | creation order |
| `listEventTickets` → `tickets` | creation order |
| `cancelEvent` → `tickets` | creation order |
| `listHazards` → `hazards` | `materialId` ascending |
| `listOrders` → `orders` | creation order |
| order → `lines` | the order the lines were added |
| stocktake → `counts` | `materialId` ascending |
| `listWorkOrders` → `workOrders` | creation order |
| `listEscalations` → `escalations` | creation order |
| escalation → `cancelledReservationIds` | reservation `start` ascending, then creation order |
| project → `collaboratorIds` | ascending |
| project → `reservationIds`, `purchaseIds` | the order attached |
| `getProjectCost` → `byMember` | `memberId` ascending |
| quote → `lines` | the order the lines were added |
| `listJobs` → `jobs` | creation order |
| `checkEligibility` → `blockers` | the order of the refusals of `createReservation` |
| `complianceSummary` → `families` | `familyId` ascending |
| `chargesByKind` → `rows` | the kind order `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event` |
| `agedReceivables` → `rows` | invoices by ascending number |
| `householdStatement` → `rows` | `memberId` ascending |

"Creation order" means the order in which records of a kind were created, which is the order of their generated numbers.

### Identifier limits

- **Generated identifiers stay valid identifiers.** A generated identifier is its prefix followed by a number. Within the integer limit it has at most 20 characters (a four-character prefix and at most 16 digits), so it always meets the identifier format and can be passed back as an argument.
- **Audit targets are not identifiers.** An audit `target` is a plain string. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format.

### Integer limits

- **Integers are exact and never negative.** Within the integer limit stated under "Large sums and counters" below, no computed integer is negative, with one exception: the `difference` of a stocktake count is negative for a shrink. Stock cannot fall below 0 (`insufficient_stock`). Credit and payments are applied at most up to an invoice's balance. Metered minutes are at least 0, because a check-out cannot precede its check-in in the request-time order. Maintenance minutes are part of the open minutes they are subtracted from.
- **Single lines cannot overflow.** The argument ranges keep every line amount and the product inside its formula below 2^53. The largest is a usage line: metered minutes are fewer than 52,596,000 (from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`), and at 10000 cents per minute the product `minutes × 10000 × 100` stays below 5.26 × 10^13. A material line's product is at most `10000 × 1000000 × 100` = 10^12. A material line made by a consumables draw has a quantity limited by stock, not by an argument, so the limit under "Large sums and counters" applies to it.
- **Large sums and counters.** Sums and counters grow with the number of requests: invoice totals, balances, credit balances, stock, report sums and minute totals, `seq` and the numbers in generated identifiers. The program keeps every integer exact up to 9007199254740991 (2^53 − 1); signed 64-bit integer arithmetic satisfies this. Once a stored or returned integer, or an intermediate value of a formula in this document, would exceed 2^53 − 1, the rest of the run is not specified and is not tested. Reaching that limit takes at least tens of thousands of requests.

### Request limits

- **Request size and depth.** The 64 KiB request lines are 65,536 bytes, not counting the line feed. Longer lines, and lines with any JSON value nested more than 100 levels deep in arrays or objects, are not specified and are not tested.
- **Counting depth.** Levels count arrays and objects only. The outermost array or object of the line, normally the request object, is level 1, and an array or object directly inside a level-n array or object is level n + 1. So `args` is level 2, and an array or object given as an argument value is level 3. Strings, numbers, booleans and `null` add no level. A JSON value is nested more than 100 levels deep when it is an array or object at level 101 or deeper, or lies inside one. Up to level 100, all other rules apply in full. A line that is not valid JSON is answered by the invalid-request rules whatever its depth.

### List audit entries

Returns all audit entries in `seq` order.

- **Wire name:** `listAuditEntries`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `entries`: a list of audit entry records
- **Permissions:** admin all.

## Module `protocol`

### Initial state

A new process contains exactly one user: identifier `admin`, name `Administrator`, role `admin`. There are no members, families, machines, courses, materials, reservations, invoices, incidents or audit entries, and no record of any other module. Every family is unsupervised, and the volunteer rate is 0.

The program generates the identifiers of these records:

| Record | Format |
| --- | --- |
| Reservation | `res-1`, `res-2`, … |
| Maintenance window | `mnt-1`, `mnt-2`, … |
| Purchase | `pur-1`, `pur-2`, … |
| Invoice | `inv-1`, `inv-2`, … |
| Payment | `pay-1`, `pay-2`, … |
| Incident | `inc-1`, `inc-2`, … |
| Journal entry | `jrn-1`, `jrn-2`, … |
| Notice | `ntc-1`, `ntc-2`, … |
| Shift | `shf-1`, `shf-2`, … |
| Room hold | `hld-1`, `hld-2`, … |
| Deposit | `dep-1`, `dep-2`, … |
| Pass | `pas-1`, `pas-2`, … |
| Scholarship award | `aws-1`, `aws-2`, … |
| Refund | `rfd-1`, `rfd-2`, … |
| Dunning case | `dun-1`, `dun-2`, … |
| Locker rental | `rnt-1`, `rnt-2`, … |
| Guest pass | `gst-1`, `gst-2`, … |
| Volunteer time entry | `vol-1`, `vol-2`, … |
| Badge | `bdg-1`, `bdg-2`, … |
| Door event | `dor-1`, `dor-2`, … |
| Lockout | `lck-1`, `lck-2`, … |
| Class | `cls-1`, `cls-2`, … |
| Payout | `pyt-1`, `pyt-2`, … |
| Event ticket | `tkt-1`, `tkt-2`, … |
| Purchase order | `po-1`, `po-2`, … |
| Stocktake | `stk-1`, `stk-2`, … |
| Work order | `wko-1`, `wko-2`, … |
| Escalation | `esc-1`, `esc-2`, … |
| Quote | `quo-1`, `quo-2`, … |
| Job | `job-1`, `job-2`, … |

Each kind has its own counter. The first record of a kind gets number 1 and each further record of that kind gets the next number, in decimal without leading zeros. Numbers are never reused. Refused and invalid requests consume no number.

### Invalid requests

The response is `status: "invalidRequest"` with a nonempty `reason` when:

- the line is not valid JSON, is empty, or is not a JSON object;
- `id` is missing, is not a string or is empty;
- `op` is missing, is not a string or names no operation of this document;
- `args` is missing or is not a JSON object;
- the request object has a member other than `id`, `op` and `args`;
- `args` has a member that is not an argument of the named operation, including a member whose name differs from a stated argument only in letter case.

The response's `id` is the request's `id` when the line is a JSON object whose `id` member is a nonempty string; otherwise it is `null`. A last line that ends without a line feed is still a request and gets a response. Behavior for JSON objects with duplicate member names and for lines that are not valid UTF-8 is not specified and is not tested. An invalid request changes nothing: no state, no audit entry, no identifier counter and no request-time order.

A request whose `args` lack a stated argument, or carry an argument with a wrong value, is not invalid in this sense. It is refused with code `invalid_argument`.

### Program

- The program reads requests from standard input, one JSON object per line, and writes exactly one response line per request to standard output, in request order. It writes and flushes each response before it reads the next request. Standard output carries responses only: no banner, prompt or log line. Diagnostics, if any, go to standard error. When standard input closes, the program exits with exit code zero.
- One process holds one makerspace, which starts in the initial state. Requests are processed one at a time, in order.
- The program never reads the machine clock, the network, environment-dependent data or files at run time, and writes no files. All time comes from the requests. Given the same request lines, it produces the same response lines.
- There is no reset, state import, clock override or other test-only operation.
- Request lines up to 64 KiB must be accepted.

### Records in responses

A `result` object has exactly the results its operation lists, and every object inside it has exactly the fields stated for its type. There are never fewer or additional members. Nullable fields are present with the value `null` when they have no value. Lists are JSON arrays; an empty list is `[]`, never `null`. Amounts, counts and minutes are JSON integers. The order of members inside a JSON object is free; the order of list elements is fixed by the order of listed results.

### Requests

A request is a UTF-8 JSON object with exactly three members:

```json
{"id":"q1","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"Ada Byron","tier":"plus"}}
```

- `id`: a nonempty string chosen by the caller. It is echoed in the response. It is only a correlation value, not a domain identifier and not an idempotency key; two requests with the same `id` are two separate requests.
- `op`: the wire name of one operation of this document. Names are case-sensitive.
- `args`: a JSON object with the operation's arguments. Every operation takes `actor` (identifier, the acting user) and `at` (timestamp, the request time) in addition to the arguments it lists. All arguments are required; no operation has optional arguments.

The operations are those of every module's interface, and list audit entries of the project itself.

### Responses

Every response is one JSON object on one line, with exactly the members shown for its status.

Success:

```json
{"id":"q1","status":"ok","result":{"member":{"memberId":"ada","name":"Ada Byron","tier":"plus","standing":"good","registeredAt":"2026-03-02T09:00Z"}}}
```

Refusal of a well-formed request:

```json
{"id":"q2","status":"refused","code":"already_exists","reason":"A user with identifier ada already exists."}
```

`code` is exactly one of the refusal codes of this document. `reason` is a nonempty, understandable explanation; its wording is free.

Invalid request:

```json
{"id":null,"status":"invalidRequest","reason":"The line is not valid JSON."}
```

## Module `members`

### Members interface

What other modules may use of members: its operations, the user and member records, tier and standing, and the refusal `member_suspended`. Other modules also ask whether a member exists, whether the member is in good standing, and the member's tier with its monthly fee and usage discount. They ask for a user's role, and for all members with their `registeredAt`, which billing needs to invoice a month. Dunning sets a member's standing as part of its own operation, which writes no audit entry of its own.

- **Operations:** register user, register member, get member, change tier, set member standing.
- **Records:** user, member, tier.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `member_suspended` | The member is not in good standing. |

### Membership

A membership has a tier and a standing. The fee of the tier is charged once for every invoiced month, without proration. The usage discount applies only to usage charges, never to course or material charges.

A member is **in good standing** when the standing is `good`. A suspended member cannot make reservations, check in, buy materials or enroll in courses. Suspension does not cancel existing reservations or enrollments. A suspended member can still cancel reservations, check out, see their records and pay invoices, and still receives the monthly fee. Reinstating good standing removes these restrictions; all other rules still apply.

### Registry

The registry holds every user: the initial admin, staff and admins registered by an admin, and members registered by staff or admins. Users, including members, share one identifier space. Users are never deleted and their identifier, name and role never change.

### Member

A user with role `member` and a membership.

**Fields**, in the formats of values:

- `memberId`: identifier
- `name`: text
- `tier`: a tier: `basic`, `plus` or `pro`
- `standing`: `good` or `suspended`
- `registeredAt`: timestamp

A record is identified by `memberId`.

### Tier

A membership tier. It sets the monthly fee and the discount on machine usage.

| Value | Meaning |
| --- | --- |
| `basic` | Monthly fee 2500 cents, usage discount 0 %. |
| `plus` | Monthly fee 4900 cents, usage discount 10 %. |
| `pro` | Monthly fee 9900 cents, usage discount 25 %. |

### User

A user of the program: the initial admin, staff and admins registered by an admin, and members registered by staff or admins.

**Fields**, in the formats of values:

- `userId`: identifier
- `name`: text
- `role`: one of the roles of authorization

A record is identified by `userId`.

### Change tier

Sets the member's tier. Setting the current tier succeeds without a change. The new tier applies to membership fees of invoices generated afterwards and to usage charges created by later check-outs.

- **Wire name:** `changeTier`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `tier`: a tier: `basic`, `plus` or `pro`
- **Result:**
  - `member`: a member record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.

### Get member

Returns the member.

- **Wire name:** `getMember`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `member`: a member record
- **Permissions:** member own, staff all, admin all.

### Register member

Registers a member with a membership: creates a user with role `member`, the given tier, standing `good`, `registeredAt` equal to the request time and a credit balance of 0.

- **Wire name:** `registerMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier
  - `name`: text
  - `tier`: a tier: `basic`, `plus` or `pro`
- **Refusals** at check 7, in this order:
  1. `already_exists` when any user has the identifier `memberId`.
- **Result:**
  - `member`: a member record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.

### Register user

Registers a staff member or an admin: creates the user.

- **Wire name:** `registerUser`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier
  - `name`: text
  - `role`: `staff` or `admin`
- **Refusals** at check 7, in this order:
  1. `already_exists` when any user, including a member, has the identifier `userId`.
- **Result:**
  - `user`: a user record
- **Permissions:** admin all.
- **Audit target:** `userId`.

### Set member standing

Suspends or reinstates the member: sets the standing. Setting the current standing succeeds without a change.

- **Wire name:** `setMemberStanding`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `standing`: `good` or `suspended`
- **Result:**
  - `member`: a member record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.

## Module `equipment`

### Equipment interface

What other modules may use of equipment: its operations, the family, machine and maintenance window records, and the machine rate type. Other modules also ask whether a machine or a family exists, for all families in ascending `familyId` order, for a machine's family and current rate and for the scheduled windows of a machine that overlap an interval. Equipment in turn needs to know whether an active reservation of a machine overlaps an interval; it declares that need, and the protocol module supplies the answer from bookings.

- **Operations:** add family, add machine, set machine rate, list machines, schedule maintenance, cancel maintenance.
- **Records:** family, machine, maintenance window.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `reservation_conflict` | An active reservation overlaps the maintenance window. |

### Catalog

The catalog holds tool families and machines. Each machine belongs to one family and has a rate in cents per minute. Families and machines are never removed.

### Maintenance

A maintenance window takes a machine out of service for an interval `[start, end)`. While a scheduled window overlaps a slot, the machine cannot be reserved for it, and since every use starts with a reservation and its check-in, the machine cannot be used in the window. The one exception is a session that is already checked in and runs past its reservation's end: it is not interrupted, keeps metering until check-out and counts in `usedMinutes` of the utilization report. Windows may use any minute, are not limited to opening hours and may overlap each other.

### Family

A tool family, such as laser cutters.

**Fields**, in the formats of values:

- `familyId`: identifier
- `name`: text

A record is identified by `familyId`.

### Machine

A machine of one family with a rate in cents per minute.

**Fields**, in the formats of values:

- `machineId`: identifier
- `name`: text
- `familyId`: identifier, naming an existing family
- `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute)

A record is identified by `machineId`.

### Maintenance window

A maintenance window of a machine, generated identifier `mnt-N`.

**Fields**, in the formats of values:

- `windowId`: identifier
- `machineId`: identifier, naming an existing machine
- `start`: timestamp
- `end`: timestamp
- `reason`: text
- `status`: `scheduled` or `cancelled`

A record is identified by `windowId`.

### Add family

Creates the family.

- **Wire name:** `addFamily`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a family with `familyId` exists.
- **Result:**
  - `family`: a family record
- **Permissions:** admin all.
- **Audit target:** `familyId`.

### Add machine

Creates the machine.

- **Wire name:** `addMachine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier
  - `name`: text
  - `familyId`: identifier, naming an existing family
  - `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a machine with `machineId` exists.
- **Result:**
  - `machine`: a machine record
- **Permissions:** admin all.
- **Audit target:** `machineId`.

### Cancel maintenance

The window gets status `cancelled`. A cancelled window no longer blocks reservations and no longer counts in schedules or reports.

- **Wire name:** `cancelMaintenance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `windowId`: identifier, naming an existing maintenance window
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the window is already cancelled or its start is at or before the request time.
- **Result:**
  - `window`: a maintenance window record
- **Permissions:** staff all, admin all.
- **Audit target:** `windowId`.

### List machines

Returns all machines.

- **Wire name:** `listMachines`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `machines`: a list of machine records
- **Permissions:** member all, staff all, admin all.

### Schedule maintenance

Creates the window with status `scheduled` and the next maintenance window identifier. A reservation is active when its reported status is `reserved` or `checkedIn`. A `reserved` reservation whose `end` is at or before the request time is `missed` and no longer active. Overlap is judged on the reserved interval `[start, end)`, not on the actual check-in and check-out times.

- **Wire name:** `scheduleMaintenance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `start`: timestamp
  - `end`: timestamp
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `reservation_conflict` when the window overlaps an active reservation of the machine.
- **Result:**
  - `window`: a maintenance window record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `windowId`.

### Set machine rate

Sets the rate. The rate is captured at check-in, so the new rate applies to check-ins after the change. Reservations already checked in keep their captured rate, and existing charges do not change.

- **Wire name:** `setMachineRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute)
- **Result:**
  - `machine`: a machine record
- **Permissions:** admin all.
- **Audit target:** `machineId`.

## Module `ledger`

### Ledger interface

What other modules may use of the ledger: its operations, the journal entry, account and trial balance row records, and the refusal `same_account`. Other modules also ask the ledger to post an entry and ask it for two figures of the journal.

- **Operations:** list journal entries, get trial balance, post adjustment.
- **Records:** journal entry, ledger account, trial balance row.
- **Capabilities** that are not operations:
  - **Post an entry.** The caller names a debit account, a credit account, an amount in cents and a memo. The ledger records one journal entry with the next entry identifier and `at` the request time. An amount of 0 posts nothing and consumes no number. The caller names the accounts, amount and memo that the postings table gives for its event; the ledger does not check that the event happened.
  - **Entries of a month.** The number of journal entries whose `at` lies in a month.
  - **Balanced.** Whether the total of all debits equals the total of all credits. It always does.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `same_account` | The debit account and the credit account of an adjustment are the same. |

### Postings

Every event in the table below posts journal entries as part of the operation that causes it. The entries are created together with the operation's other effects. An entry has `at` the request time, takes the next entry identifier, debits the **Debit** account and credits the **Credit** account by the **Amount**, and has the **Memo** as its `memo`. A refused request posts nothing.

An amount of 0 posts nothing and consumes no entry number. Entries are numbered in the order they are posted. Generate invoices takes the invoices in ascending `memberId` order and posts, for each invoice, its lines in order and then its credit entry, before the next invoice. Close stocktake takes the counts in ascending `materialId` order and posts each count's one entry. Every other operation posts in the order of its rows.

| Event | Debit | Credit | Amount | Memo |
| --- | --- | --- | --- | --- |
| `generateInvoices`, for each line of each invoice: invoices in ascending `memberId` order, the lines of an invoice in the order of its `lines` | `receivable` | the revenue account of the line's `kind`, from the table below | the line's `amountCents` | the `invoiceId` |
| `generateInvoices`, credit applied, once per invoice, after that invoice's lines | `creditLiability` | `receivable` | the invoice's `creditAppliedCents` | the `invoiceId` |
| `recordPayment` | `cash` | `receivable` | `appliedCents` | the `paymentId` |
| `recordPayment`, then | `cash` | `creditLiability` | `creditedCents` | the `paymentId` |
| `issueCredit` | `expenseGoodwill` | `creditLiability` | `amountCents` | the `memberId` |
| `releaseDeposit` | `depositLiability` | `creditLiability` | the deposit's `amountCents` | the `depositId` |
| `forfeitDeposit` | `depositLiability` | `revenueDeposits` | the deposit's `amountCents` | the `depositId` |
| `issueRefund` | `revenueRefunds` | `creditLiability` | `amountCents` | the `refundId` |
| `decideVolunteerTime`, approved | `expenseVolunteer` | `creditLiability` | the entry's `creditCents` | the `entryId` |
| `cancelTicket`, and `cancelEvent` for each ticket it cancels, in creation order | `revenueRefunds` | `creditLiability` | the credit, the event's `priceCents` | the `ticketId` |
| `receiveOrder` | `inventory` | `payable` | the quantity received in the request × the line's `unitCostCents` | the `orderId` |
| `closeStocktake`, for each count with a negative `difference`, the counts taken in one pass in ascending `materialId` order | `expenseWriteOff` | `inventory` | the units short × the material's `unitPriceCents` at the request time | the `stocktakeId` |
| `closeStocktake`, for each count with a positive `difference`, in the same pass | `inventory` | `expenseWriteOff` | the units over × the material's `unitPriceCents` at the request time | the `stocktakeId` |
| `generatePayouts`, for each payout, in creation order | `expensePayout` | `payable` | the payout's `amountCents` | the `payoutId` |
| `settlePayout` | `payable` | `cash` | the payout's `amountCents` | the `payoutId` |
| `completeJob` | `expenseMaker` | `creditLiability` | the job's `makerCreditCents` | the `jobId` |
| `recordJobPayment` | `cash` | `revenueJobs` | `amountCents` | the `jobId` |
| post adjustment | as given | as given | as given | as given |

The revenue account of an invoice line, by the `kind` of the line:

| Line `kind` | Credit account |
| --- | --- |
| `membership` | `revenueMembership` |
| `usage` | `revenueUsage` |
| `material` | `revenueMaterial` |
| `course` | `revenueCourse` |
| `deposit` | `depositLiability` |
| `pass` | `revenuePasses` |
| `locker` | `revenueLockers` |
| `guest` | `revenueGuests` |
| `lateFee` | `revenueLateFees` |
| `event` | `revenueEvents` |

The credit of an operation that adds to a member's credit balance is posted as the row of that operation says, with `creditLiability` as the credit account. The units short at a stocktake are the absolute value of a negative `difference`.

**Invariants** the program holds after every request:

- The sum of all debits equals the sum of all credits, because every entry adds its amount to one debit and one credit.
- The `receivable` account's `debitCents` less its `creditCents` equals the sum of `balanceCents` over all invoices (an invoice that is `paid` has balance 0).
- The `creditLiability` account's `creditCents` less its `debitCents` equals the sum of all members' credit balances.

The second and third hold unless a postAdjustment named that account on either side; from then on the difference between the two figures is the net of the adjustments.

**Worked example: an invoice with credit applied.** Member `ada` on tier `plus` is the only member. On `2026-03-10T09:00Z` an admin issues her 2000 cents of credit. Entry `jrn-1` debits `expenseGoodwill` and credits `creditLiability` by 2000, memo `ada`. She has a usage charge of 1499 cents in March (overdue invoices). Generating `2026-03` at `2026-04-01T08:00Z` creates `inv-1` with the membership line 4900 and the usage line 1499, `totalCents` 6399, `creditAppliedCents` 2000 and `balanceCents` 4399. It posts, in this order:

| Entry | Debit | Credit | Amount | Memo |
| --- | --- | --- | --- | --- |
| `jrn-2` | `receivable` | `revenueMembership` | 4900 | `inv-1` |
| `jrn-3` | `receivable` | `revenueUsage` | 1499 | `inv-1` |
| `jrn-4` | `creditLiability` | `receivable` | 2000 | `inv-1` |

`receivable` now has `debitCents` 6399 and `creditCents` 2000, a balance of 4399, which is the balance of `inv-1`. `creditLiability` has `debitCents` 2000 and `creditCents` 2000, a balance of 0 and side `none`, which is Ada's credit balance.

**Worked example: a payment that overpays.** On `2026-04-05T10:00Z` Ada pays 5000 cents on `inv-1`. The payment `pay-1` has `appliedCents` 4399 and `creditedCents` 601. It posts `jrn-5` (debit `cash`, credit `receivable`, 4399, memo `pay-1`) and then `jrn-6` (debit `cash`, credit `creditLiability`, 601, memo `pay-1`). A payment of exactly 4399 posts only `jrn-5`, because the second amount is 0. Now `receivable` has a balance of 0, `creditLiability` has a credit balance of 601, which is Ada's credit balance, and `cash` has `debitCents` 5000. The trial balance has `totalDebitCents` and `totalCreditCents` of 15399 each.

### Journal entry

One posting of the ledger: an amount that debits one account and credits another, generated identifier `jrn-N`. Entries are never changed or removed. Which events post which entries is stated in postings.

**Fields**, in the formats of values:

- `entryId`: identifier
- `at`: timestamp (the request time of the posting)
- `debitAccount`: a ledger account
- `creditAccount`: a ledger account
- `amountCents`: integer of at least `1` cents (a posted amount; an amount of 0 is never posted)
- `memo`: text (what the posting is for: the identifier the postings table names)

A record is identified by `entryId`.

### Ledger account

An account of the ledger. The accounts are fixed; no operation adds or removes one. They are listed here in ascending order, which is the listing order of the trial balance.

| Value | Meaning |
| --- | --- |
| `cash` | Money received from payments and job payments, less payouts settled. |
| `creditLiability` | What the workshop owes its members as credit balances. |
| `depositLiability` | Deposits invoiced and not yet released or forfeited. |
| `expenseGoodwill` | Credit issued by admins. |
| `expenseMaker` | Credit given to makers for completed jobs. |
| `expensePayout` | Instructor payouts generated. |
| `expenseVolunteer` | Credit given for approved volunteer time. |
| `expenseWriteOff` | Stock lost or found at a stocktake. |
| `inventory` | The value of stock received and counted. |
| `payable` | What the workshop owes suppliers and instructors. |
| `receivable` | What members owe on their invoices. |
| `revenueCourse` | Course lines invoiced. |
| `revenueDeposits` | Deposits forfeited. |
| `revenueEvents` | Event lines invoiced. |
| `revenueGuests` | Guest lines invoiced. |
| `revenueJobs` | Job payments received. |
| `revenueLateFees` | Late fee lines invoiced. |
| `revenueLockers` | Locker lines invoiced. |
| `revenueMaterial` | Material lines invoiced. |
| `revenueMembership` | Membership lines invoiced. |
| `revenuePasses` | Pass lines invoiced. |
| `revenueRefunds` | Refunds given as credit. |
| `revenueUsage` | Usage lines invoiced. |

### Trial balance row

One account's row of the trial balance.

**Fields**, in the formats of values:

- `account`: a ledger account
- `debitCents`: cent amount (the sum of `amountCents` of all entries whose `debitAccount` is the account)
- `creditCents`: cent amount (the sum of `amountCents` of all entries whose `creditAccount` is the account)
- `balanceCents`: cent amount (the larger of `debitCents` and `creditCents` minus the smaller)
- `side`: `debit` when `debitCents` is larger, `credit` when `creditCents` is larger, `none` when they are equal

### Get trial balance

Returns one row for every account, including accounts without entries, calculated from all journal entries at the request time. The totals are the sums over all rows. Because every entry debits one account and credits another by the same amount, the two totals are always equal (postings).

- **Wire name:** `getTrialBalance`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `rows`: a list of trial balance row records, one per account, in the listing order of ledger account
  - `totalDebitCents`: cent amount (the sum of `debitCents` over the rows)
  - `totalCreditCents`: cent amount (the sum of `creditCents` over the rows)
- **Permissions:** admin all.

### List journal entries

Returns the journal entries whose `at` lies in the month, that is, whose UTC date is in the month. A month without entries gives an empty list.

- **Wire name:** `listJournalEntries`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Result:**
  - `month`: month
  - `entries`: a list of journal entry records, in creation order (ascending entry number)
- **Permissions:** admin all.

### Post adjustment

Posts one journal entry as given: the next entry identifier, `at` the request time, and the debit account, credit account, amount and memo of the arguments. Nothing else changes. After an adjustment names an account on either side, the invariants of postings that concern that account are no longer promised.

- **Wire name:** `postAdjustment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `debitAccount`: an account of ledger account, one of `cash`, `creditLiability`, `depositLiability`, `expenseGoodwill`, `expenseMaker`, `expensePayout`, `expenseVolunteer`, `expenseWriteOff`, `inventory`, `payable`, `receivable`, `revenueCourse`, `revenueDeposits`, `revenueEvents`, `revenueGuests`, `revenueJobs`, `revenueLateFees`, `revenueLockers`, `revenueMaterial`, `revenueMembership`, `revenuePasses`, `revenueRefunds` or `revenueUsage`
  - `creditAccount`: an account of ledger account, one of `cash`, `creditLiability`, `depositLiability`, `expenseGoodwill`, `expenseMaker`, `expensePayout`, `expenseVolunteer`, `expenseWriteOff`, `inventory`, `payable`, `receivable`, `revenueCourse`, `revenueDeposits`, `revenueEvents`, `revenueGuests`, `revenueJobs`, `revenueLateFees`, `revenueLockers`, `revenueMaterial`, `revenueMembership`, `revenuePasses`, `revenueRefunds` or `revenueUsage`
  - `amountCents`: integer from `1` to `100000000` cents (an amount posted in one adjustment)
  - `memo`: text
- **Refusals** at check 7, in this order:
  1. `same_account` when `debitAccount` equals `creditAccount`.
- **Result:**
  - `entry`: a journal entry record
- **Permissions:** admin all.
- **Audit target:** the new `entryId`.

## Module `notices`

### Notices interface

What other modules may use of notices: its operations and the notice record. Other modules also post a notice for a member: they name the member, a kind other than `manual` and a reference, and the notice is created as posting notices states. A posted notice is part of the posting operation and is not audited on its own.

- **Operations:** post notice, list notices, acknowledge notice.
- **Uses:** members interface.
- **Records:** notice, notice kind.

This module adds no refusal codes.

### Posting notices

An operation of another module may post notices for members as part of its own effects. A posted notice is a notice that is created like the one of post notice, with these differences:

- `kind` is the kind the posting operation names, and `ref` is the reference it names. Neither is `manual` or `null`.
- `subject` is the subject of the kind, as notice kind states.
- `at` is the request time of the posting operation, and `acknowledgedAt` is `null`.
- It takes the next notice identifier from the same counter as the notices posted by hand. Notices posted by one operation are numbered in the order that operation states them.
- Standing is not checked.

A posted notice takes place together with the other effects of the posting operation. A refused request posts none. It adds no audit entry: the entry of the posting operation stands for it.

### Notice

A message recorded for a member, generated identifier `ntc-N`. The program only records notices; it sends nothing. `ref` and `acknowledgedAt` are `null` until set.

**Fields**, in the formats of values:

- `noticeId`: identifier
- `memberId`: identifier, naming an existing member
- `kind`: a notice kind: `manual`, `dunning`, `suspension`, `waitlist` or `cancellation`
- `subject`: text
- `ref`: text, or `null` (what the notice is about: an invoice, a course or a reservation identifier)
- `at`: timestamp
- `acknowledgedAt`: timestamp, or `null`

A record is identified by `noticeId`.

### Notice kind

The kind of a notice. It decides the `subject` of every notice that is not posted by hand. `<ref>` stands for the notice's `ref`.

| Value | Meaning | Subject |
| --- | --- | --- |
| `manual` | Posted by staff with post notice. `ref` is `null`. | As supplied. |
| `dunning` | An invoice is overdue. `ref` is the invoice identifier. | `Invoice <ref> is overdue` |
| `suspension` | The member was suspended because an invoice is overdue. `ref` is the invoice identifier. | `Membership suspended: invoice <ref> is overdue` |
| `waitlist` | The member was enrolled in a course from its waiting list. `ref` is the course identifier. | `Enrolled from the waitlist of course <ref>` |
| `cancellation` | A reservation was cancelled. `ref` is the reservation identifier. | `Reservation <ref> was cancelled` |

Example: a `dunning` notice for invoice `inv-3` has the subject `Invoice inv-3 is overdue`.

### Acknowledge notice

Sets the notice's `acknowledgedAt` to the request time. A notice belongs to the actor when its member is the actor.

- **Wire name:** `acknowledgeNotice`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `noticeId`: identifier, naming an existing notice
- **Refusals** at check 7, in this order:
  1. `invalid_state` when `acknowledgedAt` is not `null`.
- **Result:**
  - `notice`: a notice record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `noticeId`.

### List notices

Returns all notices of the member, acknowledged or not, in creation order.

- **Wire name:** `listNotices`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `notices`: a list of notice records
- **Permissions:** member own, staff all, admin all.

### Post notice

Creates a notice for the member with the next notice identifier: `kind` `manual`, `subject` as supplied, `ref` `null`, `at` the request time and `acknowledgedAt` `null`. Standing is not checked.

- **Wire name:** `postNotice`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `subject`: text
- **Result:**
  - `notice`: a notice record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `noticeId`.

## Module `households`

### Households interface

What other modules may use of households: its operations, the household record and the refusal codes below. Other modules also ask which household a member belongs to, if any, and for that household its payer, its members in ascending order and their number. The rule that binds billing is household discount.

- **Operations:** create household, add household member, remove household member, get household.
- **Uses:** members interface.
- **Records:** household.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_in_household` | The member already belongs to a household. |
| `household_full` | The household already has 6 members. |
| `not_in_household` | The member does not belong to the household. |
| `payer_cannot_leave` | The member is the household's payer. |

### Household discount

The membership line of an invoice that `generateInvoices` creates has `discountPercent` 10 when the member belongs, at the request time of that request, to a household with 3 or more members. Otherwise it is 0. The payer counts as a member. Every member of such a household gets the discount on their own membership line, not only the payer. Households are judged as they are when the request is made, whatever month is invoiced. Only the membership line is affected; the line amount follows money.

Example: a household of 3 whose members are on `basic`, `plus` and `pro` gets membership lines of 2250, 4410 and 8910 cents (`(2500 × 90 + 50) div 100`, `(4900 × 90 + 50) div 100` and `(9900 × 90 + 50) div 100`). With 2 members the same lines are 2500, 4900 and 9900. A member who belongs to no household gets no discount.

### Household membership

A household has 1 to 6 members. A member belongs to at most one household. The payer is always one of its members: the household is created with the payer as its only member, and the payer cannot be removed. Households are never removed and their payer and name never change. A member removed from a household belongs to none and may be added to any household, including the same one.

### Household

Members who share a home and one payer, as household membership states.

**Fields**, in the formats of values:

- `householdId`: identifier
- `name`: text
- `payerId`: identifier, naming an existing member
- `memberIds`: a list of identifiers, each naming an existing member, ascending; it always contains `payerId`

A record is identified by `householdId`.

### Add household member

Adds the member to the household. `memberIds` stays ascending. The payer is not changed.

- **Wire name:** `addHouseholdMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `householdId`: identifier, naming an existing household
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `already_in_household` when the member belongs to a household, this one or another.
  2. `household_full` when the household has 6 members.
- **Result:**
  - `household`: a household record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId/memberId`.

### Create household

Creates the household. Its `memberIds` hold only the payer.

- **Wire name:** `createHousehold`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `householdId`: identifier
  - `name`: text
  - `payerId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `already_exists` when a household with `householdId` exists.
  2. `already_in_household` when the payer belongs to a household.
- **Result:**
  - `household`: a household record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId`.

### Get household

Returns the household.

- **Wire name:** `getHousehold`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `householdId`: identifier, naming an existing household
- **Result:**
  - `household`: a household record
- **Permissions:** staff all, admin all.

### Remove household member

Removes the member from the household. The household stays, even when only the payer is left.

- **Wire name:** `removeHouseholdMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `householdId`: identifier, naming an existing household
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `not_in_household` when the member does not belong to this household.
  2. `payer_cannot_leave` when the member is the household's payer.
- **Result:**
  - `household`: a household record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId/memberId`.

## Module `guardians`

### Guardians interface

What other modules may use of guardians: its operations, the guardianship record and the refusal codes below. Other modules also ask whether a member is a minor on a date, whether a member holds the consent of a guardian for a family, and whether a member is refused `guardian_consent_missing` for a family on a date. The rule is guardian consent.

- **Operations:** declare minor, grant consent, withdraw consent, get guardianship.
- **Uses:** members interface, equipment interface.
- **Records:** guardianship.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `guardian_consent_missing` | The member is a minor on the relevant date and no guardian has consented for the family. |
| `invalid_guardian` | The guardian is the member, or is a minor. |
| `no_consent` | No consent is held for the family. |
| `not_a_declared_minor` | The member has no guardianship record. |
| `not_a_minor` | The member is not a minor on the date of the request time. |
| `not_guardian` | The named guardian is not the member's guardian. |

### Guardian consent

A member is a **minor on date D** when the member has a guardianship and D is before the 18th anniversary of its `birthDate`. A member with no guardianship is an adult. The 18th anniversary is the same month and day 18 years later; for a birth date of 29 February it is 1 March, because the year 18 years later has no 29 February. A birth date after D is not an error: the member is a minor on D.

Example: with birth date `2008-05-20` the member is a minor on `2026-05-19` and an adult on `2026-05-20`. With birth date `2008-02-29` the member is a minor on `2026-02-28` and an adult on `2026-03-01`.

A guardian's **consent** for a family is held when the family is in `consentFamilyIds`. It stays until it is withdrawn, and it does not depend on the guardian's later state.

**The consent gate.** A member who is a minor on the date D of a request and holds no consent for the relevant family is refused `guardian_consent_missing`. A member who is not a minor on D is never refused. The date D and the family are:

| Operation | Family | D |
| --- | --- | --- |
| `createReservation` | the machine's | the date of `start` |
| `checkIn` | the machine's | the date of the request time |
| `enrollInCourse` | the course's | the date of the request time |

### Guardianship

The declaration that a member is a minor, with the guardian and the families the guardian consented to. `consentFamilyIds` is empty until a consent is granted.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `birthDate`: date
- `guardianId`: identifier, naming an existing member
- `consentFamilyIds`: a list of identifiers, each naming an existing family, ascending

A record is identified by `memberId`. A guardianship is never removed and its birth date and guardian never change.

### Declare minor

Creates the member's guardianship with the given birth date and guardian and no consent. Standing is not checked.

Rules of other modules that apply here: guardian consent.

- **Wire name:** `declareMinor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `birthDate`: date
  - `guardianId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `already_exists` when the member has a guardianship.
  2. `not_a_minor` when the birth date makes the member an adult on the date of the request time.
  3. `invalid_guardian` when `guardianId` equals `memberId`, or the guardian is a minor on the date of the request time.
- **Result:**
  - `guardianship`: a guardianship record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.

### Get guardianship

Returns the member's guardianship.

- **Wire name:** `getGuardianship`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `not_a_declared_minor` when the member has no guardianship.
- **Result:**
  - `guardianship`: a guardianship record
- **Permissions:** member own, staff all, admin all.

### Grant consent

Adds the family to the member's `consentFamilyIds`, which stays ascending. Granting a consent that is held already succeeds without a change. The member does not have to be a minor any more. Ownership is judged on `guardianId` alone: the guardian is the actor.

- **Wire name:** `grantConsent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `guardianId`: identifier, naming an existing member
  - `memberId`: identifier, naming an existing member
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `not_guardian` when the member has no guardianship, or its `guardianId` differs from the argument.
- **Result:**
  - `guardianship`: a guardianship record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.

### Withdraw consent

Removes the family from the member's `consentFamilyIds`. Existing reservations stay. Ownership is judged on `guardianId` alone: the guardian is the actor.

Rules of other modules that apply here: guardian consent.

- **Wire name:** `withdrawConsent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `guardianId`: identifier, naming an existing member
  - `memberId`: identifier, naming an existing member
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `not_guardian` when the member has no guardianship, or its `guardianId` differs from the argument.
  2. `no_consent` when the family is not in `consentFamilyIds`.
- **Result:**
  - `guardianship`: a guardianship record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.

## Module `waivers`

### Waivers interface

What other modules may use of waivers: its operations, the waiver and signature records and the refusal codes below. Other modules also ask whether a family has a waiver, and whether a member has signed the current version of a family's waiver. The rule is waiver gate.

- **Operations:** publish waiver, get waiver, sign waiver, list member waivers.
- **Uses:** members interface, equipment interface, guardians interface.
- **Records:** waiver, waiver signature.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_signed` | The member has already signed the current version of the family's waiver. |
| `guardian_required` | The member is a minor and no guardian has consented for the family. |
| `no_waiver` | The family has no waiver. |
| `waiver_missing` | The member has not signed the current version of the family's waiver. |

### Waiver gate

A family has a waiver once one is published. A family without a waiver requires no signature. A member **has signed** a family's waiver when the member's latest waiver signature for the family has the family's current version.

**The gate.** A member who has not signed the waiver of the machine's family, because there is no signature or the latest is of an older version, is refused `waiver_missing`. It applies at `createReservation` and `checkIn`, at the place each lists. Standing, certification and consent do not change it.

Publishing a new version leaves existing reservations in place. Their check-in is refused with `waiver_missing` until the member signs the new version.

### Waiver

The current waiver of one tool family. Publishing a new waiver replaces it, and its version grows by 1.

**Fields**, in the formats of values:

- `familyId`: identifier, naming an existing family
- `version`: count, at least 1 (the first waiver of a family has version 1)
- `text`: text

A record is identified by `familyId`.

### Waiver signature

A member's signature of one version of a family's waiver. A member can have several signatures for one family, one per version signed; the latest is the one with the highest version.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `familyId`: identifier, naming an existing family
- `version`: count, at least 1 (the version of the waiver that was signed)
- `signedAt`: timestamp

### Get waiver

Returns the family's current waiver.

- **Wire name:** `getWaiver`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `no_waiver` when the family has no waiver.
- **Result:**
  - `waiver`: a waiver record
- **Permissions:** member all, staff all, admin all.

### List member waivers

Returns the member's latest signature for each family the member has signed, including signatures of versions that are no longer current. The signatures are ordered by `familyId` ascending.

- **Wire name:** `listMemberWaivers`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `signatures`: a list of waiver signature records
- **Permissions:** member own, staff all, admin all.

### Publish waiver

Makes a new waiver the family's current one: `version` is 1 for the first waiver of the family, otherwise the previous version plus 1, and `text` is as supplied. Earlier versions are no longer returned. Existing signatures stay, and waiver gate decides what they are worth.

- **Wire name:** `publishWaiver`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
  - `text`: text
- **Result:**
  - `waiver`: a waiver record
- **Permissions:** admin all.
- **Audit target:** `familyId`.

### Sign waiver

Records a signature of the family's current waiver: `version` is the current version and `signedAt` the request time. Earlier signatures stay. Standing is not checked. `already_signed` is judged as waiver gate defines having signed.

Rules of other modules that apply here: guardian consent.

- **Wire name:** `signWaiver`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `no_waiver` when the family has no waiver.
  2. `already_signed` when the member has signed the family's waiver.
  3. `guardian_required` when the member is a minor on the date of the request time and holds no consent for the family.
- **Result:**
  - `signature`: a waiver signature record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.

## Module `ppe`

### PPE interface

What other modules may use of ppe: its operations, the item and requirement records and the refusal codes below. Other modules also ask which items a family requires, and whether a member holds an item. The rule is PPE gate.

- **Operations:** add PPE item, require PPE, issue PPE, return PPE, list member PPE.
- **Uses:** members interface, equipment interface.
- **Records:** PPE item, PPE requirement.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_issued` | The member already holds the item. |
| `not_issued` | The member does not hold the item. |
| `out_of_stock` | The item's stock is 0. |
| `ppe_missing` | The member does not hold every item the machine's family requires. |

### PPE gate

A member **holds** an item from issue PPE until return PPE. A member holds at most one unit of an item. Standing does not change what a member holds.

**The gate.** At `checkIn`, a member who does not hold every item that the machine's family requires is refused `ppe_missing`, at the place that operation lists. A family with no requirement never causes it. Reservations are not gated by it.

### PPE item

An item of personal protective equipment, such as safety glasses, that the workshop lends to members.

**Fields**, in the formats of values:

- `itemId`: identifier
- `name`: text
- `stock`: count

A record is identified by `itemId`.

### PPE requirement

The fact that machines of a tool family may be checked in only by members who hold an item. A family can require several items. A requirement is never removed.

**Fields**, in the formats of values:

- `familyId`: identifier, naming an existing family
- `itemId`: identifier, naming an existing PPE item

A record is identified by `familyId` and `itemId` together.

### Add PPE item

Creates the item with the given stock.

- **Wire name:** `addPpeItem`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `itemId`: identifier
  - `name`: text
  - `stock`: integer from `0` to `100000` (the number of units of an item in stock)
- **Refusals** at check 7, in this order:
  1. `already_exists` when an item with `itemId` exists.
- **Result:**
  - `item`: a PPE item record
- **Permissions:** staff all, admin all.
- **Audit target:** `itemId`.

### Issue PPE

The member holds the item. The item's stock decreases by 1. The `item` result is the item after the change.

Rules of other modules that apply here: membership.

- **Wire name:** `issuePpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `itemId`: identifier, naming an existing PPE item
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `already_issued` when the member holds the item.
  3. `out_of_stock` when the item's stock is 0.
- **Result:**
  - `item`: a PPE item record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/itemId`.

### List member PPE

Returns the items the member holds, with their current stock, ordered by `itemId` ascending.

- **Wire name:** `listMemberPpe`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `items`: a list of PPE item records
- **Permissions:** member own, staff all, admin all.

### Require PPE

Makes the item a requirement of the family. Repeating a requirement succeeds without a change.

- **Wire name:** `requirePpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
  - `itemId`: identifier, naming an existing PPE item
- **Result:**
  - `requirement`: a PPE requirement record
- **Permissions:** admin all.
- **Audit target:** `familyId/itemId`.

### Return PPE

The member no longer holds the item. The item's stock increases by 1. The `item` result is the item after the change. Standing is not checked.

- **Wire name:** `returnPpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `itemId`: identifier, naming an existing PPE item
- **Refusals** at check 7, in this order:
  1. `not_issued` when the member does not hold the item.
- **Result:**
  - `item`: a PPE item record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/itemId`.

## Module `lockouts`

### Lockouts interface

What other modules may use of lockouts: its operations, the lockout record and its source, and the rule that a locked-out machine cannot be reserved or checked in. Other modules also ask the capabilities listed below. Lockouts needs nothing from other modules apart from equipment.

- **Operations:** apply lockout, release lockout, list lockouts.
- **Uses:** equipment interface.
- **Records:** lockout, lockout source.
- **Rules that bind other modules:** lockout gate.

Capabilities other modules ask of lockouts. None of them writes an audit entry of its own; the operation that uses it is audited once.

- Whether a machine is locked out at the request time, which is whether it has an active lockout (lockouts). Bookings and compliance ask this.
- Whether a machine has an active lockout of a given source. Inspections asks this for source `inspection`.
- Applying a lockout to a machine with any source, a reason and the user who applies it. It gets the next lockout identifier, `appliedAt` the request time and `releasedAt` `null`, and is returned. Inspections and workorders use this.
- Releasing one lockout by its identifier, whatever its source, and releasing every active lockout of one source on one machine. A release sets `releasedAt` to the request time. A lockout that is already released is left as it is. Inspections and workorders use this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `lockout_not_manual` | The lockout was not applied by hand, so it cannot be released by hand. |
| `machine_locked_out` | The machine has an active lockout at the request time. |
| `not_lockout_holder` | The actor did not apply the lockout and is not an admin. |

### Lockout gate

A machine is **locked out** at a request time when it has at least one active lockout (lockouts). `createReservation` and `checkIn` refuse `machine_locked_out` when the machine is locked out at the request time.

The gate looks at the lockouts as they stand when the request is answered. It does not look at the date of the slot: a lockout that is active now refuses a reservation for any date, and a lockout that is released now does not refuse one. A reservation made earlier stays in place, and its check-in is refused while the machine is locked out.

A session that is already checked in is not interrupted. `checkOut` is never refused because of a lockout, and the metering of the session goes on.

Example: staff apply a lockout to `laser-1` at `2026-03-02T09:00Z`. At `2026-03-02T10:00Z` a reservation of `laser-1` for `2026-04-01` is refused with `machine_locked_out`, although the lockout says nothing about April. Once the lockout is released, the same request is accepted.

### Lockouts

A lockout takes a machine out of use until it is released. A lockout is **active** while its `releasedAt` is `null`. A machine can have several active lockouts at once; it is **locked out** while it has at least one. Releasing one lockout leaves the others in place.

A lockout has one of three sources (lockout source). A `manual` lockout is applied and released by staff or an admin with the operations of this module. An `inspection` lockout is applied and released by the inspections module, and a `workOrder` lockout by the workorders module; neither can be released with release lockout.

Lockouts are never removed. A released lockout stays on record with `releasedAt` set to the request time of its release, and `releasedAt` does not change again.

### Lockout

A lockout of one machine, generated identifier `lck-N`. It is active while `releasedAt` is `null`; see lockouts.

**Fields**, in the formats of values:

- `lockoutId`: identifier
- `machineId`: identifier, naming an existing machine
- `source`: a lockout source: `manual`, `inspection` or `workOrder`
- `reason`: text
- `appliedBy`: identifier (the user who applied it)
- `appliedAt`: timestamp
- `releasedAt`: timestamp, or `null`

A record is identified by `lockoutId`.

### Lockout source

Where a lockout came from. It decides who can release it.

| Value | Meaning |
| --- | --- |
| `manual` | Applied by staff or an admin with `applyLockout`. Released with `releaseLockout`. |
| `inspection` | Applied by a failed inspection. Released by a passed inspection of the same machine. |
| `workOrder` | Applied when a work order is opened. Released when that work order is closed. |

### Apply lockout

Creates a lockout of the machine with the next lockout identifier, `source` `manual`, `reason` as supplied, `appliedBy` the actor, `appliedAt` the request time and `releasedAt` `null`. The machine may already have active lockouts; they stay.

Rules of other modules that apply here: lockout gate.

- **Wire name:** `applyLockout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `reason`: text
- **Result:**
  - `lockout`: a lockout record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `lockoutId`.

### List lockouts

Returns all lockouts of the machine, active and released, in creation order.

- **Wire name:** `listLockouts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
- **Result:**
  - `machineId`: identifier
  - `lockouts`: a list of lockout records
- **Permissions:** staff all, admin all.

### Release lockout

The lockout gets `releasedAt` the request time. Other active lockouts of the machine stay active.

- **Wire name:** `releaseLockout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `lockoutId`: identifier, naming an existing lockout
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the lockout is already released.
  2. `lockout_not_manual` when the lockout's `source` is not `manual`.
  3. `not_lockout_holder` when the actor is not the lockout's `appliedBy` and the actor's role is not admin.
- **Result:**
  - `lockout`: a lockout record
- **Permissions:** staff all, admin all.
- **Audit target:** `lockoutId`.

## Module `inspections`

### Inspections interface

What other modules may use of inspections: its operations, the inspection plan record, and the rule that a machine whose inspection is due cannot be reserved. Other modules also ask the capabilities listed below.

- **Operations:** set inspection plan, record inspection, get inspection status, list inspections due.
- **Uses:** equipment interface, lockouts interface.
- **Records:** inspection plan.
- **Rules that bind other modules:** inspection gate.

Capabilities other modules ask of inspections:

- The inspection plan of a machine, or that it has none. Compliance asks this.
- Whether a machine is due on a date, which is whether it has a plan and the date is on or after the plan's `nextDueOn`. Bookings and compliance ask this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `inspection_due` | The machine has an inspection plan and is due on the relevant date. |
| `no_plan` | The machine has no inspection plan. |

### Inspection cycle

A machine has at most one inspection plan. A machine without a plan is never inspected and never due.

`nextDueOn` is the date `intervalDays` days after `lastPassedOn` (time). It is computed again whenever `intervalDays` or `lastPassedOn` changes. It can lie after 2099 and then has a four-digit year (computed times).

A passed inspection moves `lastPassedOn` to the date of the request time. A failed inspection leaves `lastPassedOn` and `nextDueOn` as they are and sets `failed`. The flag `failed` only records the latest outcome; it blocks nothing by itself. What blocks the machine after a failed inspection is the lockout of source `inspection` that the failure applies (lockouts). Inspections keep at most one active lockout of source `inspection` on a machine: a second failure while one is active applies none, and a pass releases every one.

Example: a plan with `intervalDays` 30 is set on `2026-03-01`. `lastPassedOn` is `2026-03-01` and `nextDueOn` is `2026-03-31`. A passed inspection on `2026-03-31` moves `lastPassedOn` to `2026-03-31` and `nextDueOn` to `2026-04-30`.

### Inspection gate

`createReservation` refuses `inspection_due` when the machine has an inspection plan and the date of the reservation's `start` is on or after the plan's `nextDueOn`. A machine without a plan is never refused. The plan's `failed` flag plays no part here.

Only the date of `start` counts. The request time, the date of `end` and the time of day do not. A reservation made before the machine fell due stays in place, and `checkIn` is not gated on the date, so it is not refused for a due inspection. After a passed inspection moves `nextDueOn`, reservations whose start date is before the new `nextDueOn` are accepted again.

Example: a plan has `nextDueOn` `2026-03-31`. A reservation starting on `2026-03-30` is accepted by this gate; one starting on `2026-03-31` or later is refused with `inspection_due`.

### Inspection plan

The inspection plan of one machine; see inspection cycle for how its dates move.

**Fields**, in the formats of values:

- `machineId`: identifier, naming an existing machine
- `intervalDays`: integer from `1` to `3650` (the days between inspections)
- `lastPassedOn`: date
- `nextDueOn`: date, `intervalDays` days after `lastPassedOn`
- `failed`: flag, `true` when the latest inspection failed

A record is identified by `machineId`.

### Get inspection status

Shows the plan of the machine and whether it is due. `due` is `true` when the date of the request time is on or after the plan's `nextDueOn`, otherwise `false`.

- **Wire name:** `getInspectionStatus`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
- **Refusals** at check 7, in this order:
  1. `no_plan` when the machine has no inspection plan.
- **Result:**
  - `plan`: an inspection plan record
  - `due`: flag
- **Permissions:** staff all, admin all.

### List inspections due

Returns the plans whose `nextDueOn` is on or before `date`, in `machineId` ascending order. A plan whose `nextDueOn` equals `date` is included. Machines without a plan are not listed.

- **Wire name:** `listInspectionsDue`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `date`: date
- **Result:**
  - `date`: date
  - `plans`: a list of inspection plan records
- **Permissions:** staff all, admin all.

### Record inspection

Records the outcome of an inspection of the machine.

- `passed`: `lastPassedOn` becomes the date of the request time, `nextDueOn` is computed again, `failed` becomes `false`, and every active lockout of source `inspection` on the machine is released (lockouts interface). Lockouts of other sources stay.
- `failed`: `failed` becomes `true`; `lastPassedOn` and `nextDueOn` stay. When the machine has no active lockout of source `inspection`, one is applied with `reason` the supplied `note`, `appliedBy` the actor and `appliedAt` the request time; otherwise none is applied.

The `note` is not stored on the plan.

Rules of other modules that apply here: lockout gate.

- **Wire name:** `recordInspection`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `outcome`: `passed` or `failed`
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `no_plan` when the machine has no inspection plan.
- **Result:**
  - `plan`: an inspection plan record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId`.

### Set inspection plan

Sets the inspection interval of the machine. A machine without a plan gets a new plan: `lastPassedOn` is the date of the request time and `failed` is `false`. A machine with a plan keeps its `lastPassedOn` and its `failed`. In both cases `intervalDays` is the supplied value and `nextDueOn` is computed from it (inspection cycle). Setting the current interval succeeds without a change. Lockouts are not touched.

- **Wire name:** `setInspectionPlan`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `intervalDays`: integer from `1` to `3650` (the days between inspections)
- **Result:**
  - `plan`: an inspection plan record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId`.

## Module `staffing`

### Staffing interface

What other modules may use of staffing: its operations, the shift and supervision records and the refusal codes below. Other modules also ask whether an interval is covered by scheduled shifts, whether a family is supervised, and whether a user has a scheduled shift that contains a point in time. The rule is supervision. Staffing in turn needs the active reservations of all machines of a family, each with its machine, `start` and `end`; it declares that need, and the protocol module supplies the answer from bookings.

- **Operations:** add shift, cancel shift, list shifts, set supervision.
- **Uses:** members interface, equipment interface.
- **Records:** shift, supervision setting.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `not_staff` | The user is a member. |
| `shift_needed` | Cancelling the shift would leave an active reservation of a supervised family without cover. |
| `shift_overlap` | Another scheduled shift of the user overlaps the shift. |
| `shift_too_long` | The shift is longer than 720 minutes. |
| `supervisor_required` | The machine's family is supervised and the slot is not covered by scheduled shifts. |

### Supervision

A family is **supervised** when its supervision setting has `required` true. Only scheduled shifts count; cancelled shifts count for nothing.

**Cover.** An interval is **covered** when every minute of it lies in at least one scheduled shift. The shifts may belong to several users and may touch or overlap, so cover can be pieced together. Example: the shifts `09:00`–`12:00` and `12:00`–`15:00` cover `11:00`–`13:00`; the shifts `09:00`–`12:00` and `12:30`–`15:00` do not.

**At reservation.** A reservation of a machine of a supervised family is refused `supervisor_required` when its slot is not covered. This applies at `createReservation`, at the place that operation lists. Turning supervision on does not touch existing reservations, and `checkIn` does not judge it.

**At cancellation.** cancel shift is refused `shift_needed` when an active reservation of a machine of a supervised family overlaps the shift, is covered now and would not be covered if the shift were cancelled. A reservation is active as reservations states, judged on its reserved interval at the request time. A reservation that is not covered now does not stop a cancellation.

Example: the family is supervised, the shifts are `09:00`–`13:00` and `12:00`–`18:00` of one day, and an active reservation runs `12:00`–`13:30`. Cancelling the first shift is allowed, because the second still covers the reservation. Cancelling the second is refused, because `13:00`–`13:30` would be left uncovered.

### Shift

A period in which a staff user or an admin is on duty, generated identifier `shf-N`. A shift is never removed; a cancelled shift counts for nothing.

**Fields**, in the formats of values:

- `shiftId`: identifier
- `userId`: identifier, naming an existing user
- `start`: timestamp
- `end`: timestamp
- `status`: `scheduled` or `cancelled`

A record is identified by `shiftId`.

### Supervision setting

Whether machines of a tool family need a person on duty. Every family has one; a family nobody has set is not supervised.

**Fields**, in the formats of values:

- `familyId`: identifier, naming an existing family
- `required`: flag

A record is identified by `familyId`.

### Add shift

Creates the shift with status `scheduled` and the next shift identifier. A shift may start and end on any minute. The user may be staff or an admin.

- **Wire name:** `addShift`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier, naming an existing user
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `not_staff` when the user is a member.
  2. `invalid_interval` when `end` is not after `start`.
  3. `starts_in_past` when `start` is before the request time.
  4. `shift_too_long` when the shift is longer than 720 minutes; exactly 720 is allowed.
  5. `shift_overlap` when another scheduled shift of the same user overlaps `[start, end)`.
- **Result:**
  - `shift`: a shift record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `shiftId`.

### Cancel shift

The shift gets status `cancelled`. A cancelled shift no longer covers anything, no longer counts in schedules and no longer overlaps other shifts of its user.

Rules of other modules that apply here: reservations.

- **Wire name:** `cancelShift`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `shiftId`: identifier, naming an existing shift
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the shift is already cancelled or its `start` is at or before the request time.
  2. `shift_needed` when supervision says the cancellation would leave an active reservation without cover.
- **Result:**
  - `shift`: a shift record
- **Permissions:** staff all, admin all.
- **Audit target:** `shiftId`.

### List shifts

Returns the scheduled shifts of all users that overlap the date, that is the interval from `00:00` of the date to `00:00` of the next date. They are ordered by `start` ascending, then in creation order.

- **Wire name:** `listShifts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `date`: date
- **Result:**
  - `date`: date
  - `shifts`: a list of shift records
- **Permissions:** staff all, admin all.

### Set supervision

Sets whether the family is supervised. Setting the current value succeeds without a change. Existing reservations are not touched.

- **Wire name:** `setSupervision`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
  - `required`: flag
- **Result:**
  - `supervision`: a supervision setting record
- **Permissions:** admin all.
- **Audit target:** `familyId`.

## Module `parts`

### Parts interface

What other modules may use of parts: its operations and the part record. Other modules also ask for a part by its identifier, with its family, unit cost and stock, and to decrease a part's stock. Parts adds no refusal codes.

- **Operations:** add part, restock part, list parts.
- **Uses:** equipment interface.
- **Records:** part.

### Spare parts

Parts are spare parts that repairs use up. Each part fits the machines of one tool family and has a unit cost and a stock count, which starts at 0. Stock never goes below 0. Parts are never removed. A part is not sold to members and creates no charge.

### Part

A spare part with a unit cost and a stock count; see spare parts.

**Fields**, in the formats of values:

- `partId`: identifier
- `name`: text
- `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit of a part)
- `familyId`: identifier, naming an existing family (the family the part fits)
- `stock`: count

A record is identified by `partId`.

### Add part

Creates the part with stock 0.

- **Wire name:** `addPart`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `partId`: identifier
  - `name`: text
  - `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit of a part)
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `already_exists` when a part with `partId` exists.
- **Result:**
  - `part`: a part record
- **Permissions:** staff all, admin all.
- **Audit target:** `partId`.

### List parts

Returns all parts in `partId` ascending order.

- **Wire name:** `listParts`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `parts`: a list of part records
- **Permissions:** staff all, admin all.

### Restock part

The stock increases by `quantity`.

- **Wire name:** `restockPart`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `partId`: identifier, naming an existing part
  - `quantity`: integer from `1` to `100000` (the number of units added to stock in one request)
- **Result:**
  - `part`: a part record
- **Permissions:** staff all, admin all.
- **Audit target:** `partId`.

## Module `rooms`

### Rooms interface

What other modules may use of rooms: its operations and the room and hold records. Other modules also ask for a room's capacity, whether a room is free for an interval, to hold a room for an interval and to release a hold they made.

- **Operations:** add room, hold room, release room hold, get room schedule.
- **Records:** room, hold.

What other modules ask of rooms that is not an operation:

- The `capacity` of a room.
- Whether a room is **free for** an interval, as room holds states.
- Holding a room for an interval `[start, end)` on behalf of another module. The caller has checked that the room is free for the interval and states the hold's `purpose` and its origin, `class` or `event`. The hold is created as room holds states.
- Releasing a hold that was made that way: its status becomes `released`. This is how a class or an event gives its room back. release room hold does not release such holds.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `hold_not_direct` | The hold was made for a class or an event, not by a direct hold of the room. |
| `room_taken` | A held hold of the room overlaps the interval. |

### Room holds

A room is kept for an interval `[start, end)` by a hold. A hold has status `held` until it is released; a released hold is never held again. Rooms are never removed.

A room is **free for** an interval when none of its held holds overlaps the interval (time). Released holds do not count. Two held holds of one room therefore never overlap. A hold of 10:00 to 11:00 and a hold of 11:00 to 12:00 do not overlap.

Every hold has an **origin**, which is not a field of the record:

- `direct` when hold room made it. Its `purpose` is the text of the request.
- `class` when it was made for a scheduled class. Its `purpose` is `class ` followed by the class identifier, for example `class cls-3`.
- `event` when it was made for a created event. Its `purpose` is `event ` followed by the event identifier, for example `event summer-fair`.

A hold made for a class or an event is released only together with that class or event, when it is cancelled. release room hold refuses it. Making a hold for a class or an event consumes a hold identifier like any other hold.

### Hold

A hold of a room for an interval, generated identifier `hld-N`. The hold keeps the room for `[start, end)` while its status is `held`.

**Fields**, in the formats of values:

- `holdId`: identifier
- `roomId`: identifier, naming an existing room
- `start`: timestamp
- `end`: timestamp
- `purpose`: text
- `status`: `held` or `released`

A record is identified by `holdId`.

### Room

A room that can be held for an interval.

**Fields**, in the formats of values:

- `roomId`: identifier
- `name`: text
- `capacity`: integer from `1` to `1000` (the number of people a room holds)

A record is identified by `roomId`.

### Add room

Creates the room. It has no holds.

- **Wire name:** `addRoom`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `roomId`: identifier
  - `name`: text
  - `capacity`: integer from `1` to `1000` (the number of people a room holds)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a room with `roomId` exists.
- **Result:**
  - `room`: a room record
- **Permissions:** admin all.
- **Audit target:** `roomId`.

### Get room schedule

Shows one room's day: `holds` are the room's held holds that overlap the interval from `00:00` of that date to `00:00` of the next date, in every origin. Released holds are not shown. A hold from `2026-03-03T23:00Z` to `2026-03-04T01:00Z` is shown for both `2026-03-03` and `2026-03-04`; a hold that ends at `2026-03-04T00:00Z` is not shown for `2026-03-04`.

- **Wire name:** `getRoomSchedule`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `roomId`: identifier, naming an existing room
  - `date`: date
- **Result:**
  - `roomId`: identifier
  - `date`: date
  - `holds`: a list of hold records
- **Permissions:** member all, staff all, admin all.

### Hold room

Creates a hold of the room with status `held`, the next hold identifier and `purpose` as supplied. The hold is direct (room holds). The room has no limit on the length of a hold and no opening hours.

Example: the room `lab` has a held hold from `2026-03-03T10:00Z` to `2026-03-03T12:00Z`. A hold from `12:00` to `13:00` is accepted; a hold from `11:30` to `12:30` is refused with `room_taken`.

- **Wire name:** `holdRoom`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `roomId`: identifier, naming an existing room
  - `start`: timestamp
  - `end`: timestamp
  - `purpose`: text
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `room_taken` when the room is not free for `[start, end)` (room holds).
- **Result:**
  - `hold`: a hold record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `holdId`.

### Release room hold

The hold gets status `released`. The room is free again for the hold's interval, unless another held hold overlaps it. A hold may be released before, during or after its interval, as long as it is held.

- **Wire name:** `releaseRoomHold`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `holdId`: identifier, naming an existing hold
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the hold is already released.
  2. `hold_not_direct` when the hold was not made by hold room (room holds).
- **Result:**
  - `hold`: a hold record
- **Permissions:** staff all, admin all.
- **Audit target:** `holdId`.

## Module `billing`

### Billing interface

What other modules may use of billing: its operations and the charge, invoice, payment and account records. Other modules create a charge for a member through billing; billing computes its amount by money. Other modules also use the capabilities listed below.

- **Operations:** generate invoices, list invoices, record payment, issue credit, get account.
- **Uses:** members interface, ledger interface, households interface.
- **Records:** charge, invoice, payment, account.
- **Capabilities** that are not operations:
  - **Create a charge.** The caller names the member and gives the `kind` (any kind of charge), `ref`, `quantity`, `unitPriceCents`, `discountPercent` and `at`. Billing computes `amountCents` by money, fixes it, and keeps the charge unbilled until an invoice takes it (charges and invoices). It returns the charge. A charge of 0 cents is created like any other. Billing refuses nothing; the caller has checked what its own operation requires.
  - **Add credit.** The caller names the member, an amount in cents, the debit account and the memo. The member's credit balance increases by the amount, and the entry that debits the named account and credits `creditLiability` by the amount, with that memo and `at` the request time, is posted (postings). An amount of 0 changes nothing and posts nothing. The credit does not count as a credit issued with issue credit.
  - **Open invoices.** All invoices whose status is `open` at a request time, in ascending invoice number.
  - **Whether a charge is on a paid invoice.** The caller names a charge by its `kind` and `ref`. The answer is yes when a line with that `kind` and `ref` is on an invoice whose status is `paid`, and no when no invoice has such a line or the invoice is `open`.
  - **An invoice.** An invoice by its identifier, with its lines in the order of the invoice. The invoice a member has for a month, if any: a member has at most one invoice for a month. The invoices of a month, in ascending invoice number, which is creation order.
  - **A member's account.** The figures of account: credit balance, open balance, overdue balance and unbilled amount.
  - **Overdue invoices.** All invoices that are overdue at a request time (overdue invoices), in ascending invoice number. The invoice number is the number `N` of `inv-N`, compared as a number.
  - **Whether a month is invoiced.** Whether generate invoices has been accepted for the month.
  - **Payments and credits of a month.** The sum of `amountCents` of all payments whose `at` lies in the month, and the sum of the credits issued with issue credit whose request time lies in the month.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |
| `month_not_ended` | The month has not ended at the request time. |

### Charges and invoices

Charges of every kind except `membership` collect on the member's account until they are invoiced. A charge belongs to the month of its `at`. A charge is **unbilled** until it appears on an invoice.

Invoices are generated per calendar month, for all members at once, when staff request it. Generation for a month is possible once the month has ended at the request time, and only once per month. Months can be generated in any order.

For a month M, every member whose `registeredAt` is before the first minute of the month after M receives one invoice, whatever the member's standing. The invoice is built as follows:

1. `invoiceId` is the next invoice identifier. Members are processed in ascending `memberId` order.
2. `issuedAt` is the request time; `dueAt` is 14 days after `issuedAt`.
3. `lines` starts with the membership line: `kind` `membership`, `ref` the member's current tier, `quantity` 1, `unitPriceCents` the tier's monthly fee, `discountPercent` as household discount states, `at` the request time. All of the member's unbilled charges whose `at` lies in month M follow, ordered as the order of listed results states. These charges are no longer unbilled.
4. `totalCents` is the sum of the line amounts.
5. `creditAppliedCents` is the smaller of the member's credit balance and `totalCents`. The credit balance decreases by that amount.
6. `paidCents` is 0. `balanceCents` is `totalCents − creditAppliedCents − paidCents`.
7. `status` is `paid` when `balanceCents` is 0, otherwise `open`.

### Overdue invoices

Because request times never go backwards and generation needs the month to have ended, no charge can be created for a month after that month was invoiced.

An invoice is **overdue** at a request time when its status is `open` and the request time is at or after `dueAt`. The `overdue` field reports this at the request time of the response. Being overdue has no other effect in billing; operations of other modules that judge it say so on their own pages.

Worked example: member `ada` on tier `plus` uses a machine with rate 45 from `2026-03-03T10:00Z` to `10:37`. At check-out a usage charge of 1499 cents is created. Generating `2026-03` at `2026-04-01T08:00Z` gives `ada` an invoice with the membership line (4900) and the usage line (1499): `totalCents` 6399, due `2026-04-15T08:00Z`.

### Payments and credit

A payment pays one invoice. The part of a payment that exceeds the invoice's balance becomes credit on the member's account. Admins can also issue credit as a goodwill gesture. Credit is applied automatically to the next invoices generated; it is never applied to an invoice that already exists.

### Account

A member's account summary.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `creditBalanceCents`: cent amount
- `openBalanceCents`: cent amount
- `overdueBalanceCents`: cent amount
- `unbilledCents`: cent amount

### Charge

A charge (invoice line) on a member's account. Its amount follows money and is fixed when the charge is created.

**Fields**, in the formats of values:

- `kind`: one of `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee` or `event`
- `ref`: string (what a charge is for: the member's tier, a reservation, a purchase, a course, a deposit, a pass, a rental, a guest pass, an invoice or a ticket identifier)
- `quantity`: count
- `unitPriceCents`: cent amount
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp

### Invoice

An invoice of one member for one month, generated identifier `inv-N`.

**Fields**, in the formats of values:

- `invoiceId`: identifier
- `memberId`: identifier, naming an existing member
- `month`: month
- `issuedAt`: timestamp
- `dueAt`: timestamp
- `lines`: a list of charge records
- `totalCents`: cent amount
- `creditAppliedCents`: cent amount
- `paidCents`: cent amount
- `balanceCents`: cent amount
- `status`: `open` or `paid`
- `overdue`: flag

A record is identified by `invoiceId`.

### Payment

A payment of one invoice, generated identifier `pay-N`.

**Fields**, in the formats of values:

- `paymentId`: identifier
- `invoiceId`: identifier, naming an existing invoice
- `memberId`: identifier, naming an existing member
- `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
- `appliedCents`: cent amount
- `creditedCents`: cent amount
- `at`: timestamp

A record is identified by `paymentId`.

### Generate invoices

Creates the invoices as charges and invoices describes and marks the month as invoiced, even when there is no member to invoice. The entries of the ledger are posted as the postings of the ledger state.

Rules of other modules that apply here: membership, household discount and postings.

- **Wire name:** `generateInvoices`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`).
  2. `already_invoiced` when invoices for this month were generated before.
- **Result:**
  - `month`: month
  - `invoices`: a list of invoice records
- **Permissions:** staff all, admin all.
- **Audit target:** `month`.

### Get account

Returns the member's account: `creditBalanceCents` is the member's credit balance; `openBalanceCents` is the sum of `balanceCents` over the member's invoices with status `open`; `overdueBalanceCents` is the same sum over the member's overdue invoices; `unbilledCents` is the sum of the amounts of the member's unbilled charges.

- **Wire name:** `getAccount`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `account`: a account record
- **Permissions:** member own, staff all, admin all.

### Issue credit

The member's credit balance increases by `amountCents`. The credit counts as issued at the request time for the revenue report. `creditBalanceCents` is the member's credit balance after the change. The entry of the ledger is posted as the postings of the ledger state.

Rules of other modules that apply here: postings.

- **Wire name:** `issueCredit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
  - `reason`: text
- **Result:**
  - `memberId`: identifier
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
  - `creditBalanceCents`: cent amount
- **Permissions:** admin all.
- **Audit target:** `memberId`.

### List invoices

Returns all of the member's invoices.

- **Wire name:** `listInvoices`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `invoices`: a list of invoice records
- **Permissions:** member own, staff all, admin all.

### Record payment

`appliedCents` is the smaller of `amountCents` and the invoice's `balanceCents`; `creditedCents` is `amountCents − appliedCents`. The invoice's `paidCents` increases and its `balanceCents` decreases by `appliedCents`; when `balanceCents` reaches 0 its status becomes `paid`. The member's credit balance increases by `creditedCents`. A payment record is created with the next payment identifier, `memberId` the invoice's member (whoever acts) and `at` the request time. Paying after the due date is allowed. The `invoice` result is the invoice after the change; `creditBalanceCents` is the member's credit balance after the change. The entries of the ledger are posted as the postings of the ledger state.

Rules of other modules that apply here: postings.

- **Wire name:** `recordPayment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `invoiceId`: identifier, naming an existing invoice
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
- **Refusals** at check 7, in this order:
  1. `invoice_paid` when the invoice's status is `paid`.
- **Result:**
  - `payment`: a payment record
  - `invoice`: a invoice record
  - `creditBalanceCents`: cent amount
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `paymentId`.

## Module `deposits`

### Deposits interface

What other modules may use of deposits: its operations, the deposit record and the refusal `deposit_unpaid`. No other module asks anything else of it.

- **Operations:** take deposit, release deposit, forfeit deposit, list deposits.
- **Uses:** members interface, billing interface, ledger interface.
- **Records:** deposit.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `deposit_unpaid` | The deposit's charge is not on an invoice with status `paid`. |

### Deposit

A sum a member leaves with the workshop, generated identifier `dep-N`. A deposit is `held` from the moment it is taken. Releasing it turns it into credit for the member, so its status becomes `released`; forfeiting it keeps the sum for the workshop, so its status becomes `forfeited`. `closedAt` is `null` while the deposit is held.

**Fields**, in the formats of values:

- `depositId`: identifier
- `memberId`: identifier, naming an existing member
- `purpose`: text
- `amountCents`: integer from `1` to `100000000` cents (a deposit amount)
- `status`: one of `held`, `released` or `forfeited`
- `takenAt`: timestamp
- `closedAt`: timestamp, or `null`

A record is identified by `depositId`.

### Forfeit deposit

Forfeits the held deposit: its status becomes `forfeited` and `closedAt` the request time. The member's credit balance does not change. The `reason` is not stored. The entry is posted as the postings of the ledger state.

A deposit is paid as release deposit states.

Rules of other modules that apply here: postings.

- **Wire name:** `forfeitDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `depositId`: identifier, naming an existing deposit
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the deposit's status is not `held`.
  2. `deposit_unpaid` when the deposit is not paid.
- **Result:**
  - `deposit`: a deposit record
- **Permissions:** admin all.
- **Audit target:** `depositId`.

### List deposits

Returns all of the member's deposits, in creation order (ascending deposit number). A member without deposits gives an empty list.

- **Wire name:** `listDeposits`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `deposits`: a list of deposit records
- **Permissions:** member own, staff all, admin all.

### Release deposit

Releases the held deposit: its status becomes `released` and `closedAt` the request time. The member's credit balance increases by the deposit's `amountCents`, and the entry is posted as the postings of the ledger state. `creditBalanceCents` is the member's credit balance after the change. The member's standing is not checked.

A deposit is paid when the invoice that has its charge (the line with `kind` `deposit` and `ref` the deposit identifier) has status `paid`. A deposit whose charge is not yet on any invoice is not paid.

Rules of other modules that apply here: postings and charges and invoices.

- **Wire name:** `releaseDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `depositId`: identifier, naming an existing deposit
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the deposit's status is not `held`.
  2. `deposit_unpaid` when the deposit is not paid.
- **Result:**
  - `deposit`: a deposit record
  - `creditBalanceCents`: cent amount
- **Permissions:** staff all, admin all.
- **Audit target:** `depositId`.

### Take deposit

Takes a deposit from the member. The deposit is created with the next deposit identifier, status `held`, `takenAt` the request time and `closedAt` `null`. A deposit charge is created for the member: `kind` `deposit`, `ref` the deposit identifier, `quantity` 1, `unitPriceCents` the deposit's `amountCents`, `discountPercent` 0 and `at` the request time. The charge is invoiced like any other; the deposit is not paid by this request.

Rules of other modules that apply here: charges and invoices and membership.

- **Wire name:** `takeDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `purpose`: text
  - `amountCents`: integer from `1` to `100000000` cents (a deposit amount)
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
- **Result:**
  - `deposit`: a deposit record
  - `charge`: a charge record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `depositId`.

## Module `passes`

### Passes interface

What other modules may use of passes: its operations and the pass product and pass records. Bookings also asks passes to draw minutes at check-out.

- **Operations:** add pass product, sell pass, list member passes.
- **Uses:** members interface, billing interface.
- **Records:** pass product, pass.
- **Capabilities** that are not operations:
  - **Draw minutes.** The caller names a member and a number of metered minutes. Passes draws from the member's passes as the pass draw rule states, lowers their `minutesLeft` and returns the number of minutes drawn, which is never more than the minutes asked for.

### Pass draw

At `checkOut`, minutes are drawn from the member's passes before the usage charge is created.

A pass is **valid on date D** when its `minutesLeft` is above 0 and D is before its `expiresOn`. The date used is the date of the request time of the check-out. A pass is not valid on its `expiresOn` date. Whether the pass's charge is paid, and the member's standing, do not matter.

The draw takes the member's valid passes in ascending `expiresOn` order, passes with the same `expiresOn` in creation order. From each pass in turn it draws the smaller of the pass's `minutesLeft` and the minutes still to cover, and lowers `minutesLeft` by that amount. It stops when the metered minutes are covered or the valid passes are used up. `drawn` is the total drawn. It is 0 when the metered minutes are 0 or no pass is valid. `billable` is the metered minutes less `drawn`.

Example: ada has three passes. `pas-1` expires `2026-05-01` with 25 minutes left, `pas-2` expires `2026-05-01` with 100 minutes left and `pas-3` expires `2026-06-01` with 20 minutes left. She checks out on `2026-04-30` with 50 metered minutes. The draw takes 25 minutes from `pas-1` and 25 from `pas-2`: `drawn` 50, `billable` 0, and `pas-2` has 75 minutes left. If she checks out with 50 metered minutes on `2026-05-01`, `pas-1` and `pas-2` are no longer valid, the draw takes 20 minutes from `pas-3`, `drawn` is 20 and `billable` is 30.

### Pass

A pass of one member, sold from a pass product, generated identifier `pas-N`.

**Fields**, in the formats of values:

- `passId`: identifier
- `memberId`: identifier, naming an existing member
- `productId`: identifier, naming an existing pass product
- `minutesLeft`: minute count (starts as the product's `minutes`; the pass draw rule lowers it)
- `issuedOn`: date (the date of the request time of the sale)
- `expiresOn`: date (`validDays` days after `issuedOn`)

A record is identified by `passId`.

### Pass product

A product the workshop sells: a block of machine minutes that is valid for a number of days.

**Fields**, in the formats of values:

- `productId`: identifier
- `name`: text
- `minutes`: integer from `1` to `100000` (the minutes of a pass)
- `priceCents`: integer from `1` to `100000000` cents (the price of a pass)
- `validDays`: integer from `1` to `3650` (the days a pass is valid)

A record is identified by `productId`.

### Add pass product

Creates the pass product.

- **Wire name:** `addPassProduct`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `productId`: identifier
  - `name`: text
  - `minutes`: integer from `1` to `100000` (the minutes of a pass)
  - `priceCents`: integer from `1` to `100000000` cents (the price of a pass)
  - `validDays`: integer from `1` to `3650` (the days a pass is valid)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a pass product with `productId` exists.
- **Result:**
  - `product`: a pass product record
- **Permissions:** admin all.
- **Audit target:** `productId`.

### List member passes

Returns all passes of the member, expired and used up ones included, in creation order (ascending pass number). A member without passes gives an empty list.

- **Wire name:** `listMemberPasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `passes`: a list of pass records
- **Permissions:** member own, staff all, admin all.

### Sell pass

Sells the product to the member. A pass is created with the next pass identifier, `minutesLeft` the product's `minutes`, `issuedOn` the date of the request time and `expiresOn` the date `validDays` days after `issuedOn` (time). A pass charge is created for the member: `kind` `pass`, `ref` the pass identifier, `quantity` 1, `unitPriceCents` the product's `priceCents`, `discountPercent` 0 and `at` the request time. The pass can be used before its charge is paid.

Rules of other modules that apply here: charges and invoices and membership.

- **Wire name:** `sellPass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `productId`: identifier, naming an existing pass product
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
- **Result:**
  - `pass`: a pass record
  - `charge`: a charge record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `passId`.

## Module `promotions`

### Promotions interface

What other modules may use of promotions: its operations, the promotion, redemption and scope records, and the refusals below. Other modules also ask for a member's oldest unused redemption and mark a redemption used.

- **Operations:** add promotion, redeem promotion, list member promotions, end promotion.
- **Uses:** members interface.
- **Records:** promotion, redemption, promotion scope.
- **Capabilities** that are not operations, as the redeemed promotions rule states them:
  - **The oldest unused redemption.** The caller names a member and a scope to cover, `course` or `material`. The answer is the member's oldest unused redemption whose promotion covers that scope, with its `percent`, or none.
  - **Use a redemption.** The caller names the redemption and the `ref` of the charge it is used for. The redemption gets `usedAt` the request time and `usedOn` that `ref`.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_redeemed` | The member has already redeemed the promotion. |
| `promotion_ended` | The promotion has been ended. |
| `promotion_exhausted` | The promotion has been redeemed as often as it allows. |
| `promotion_expired` | The request date is after the promotion's `validUntil`. |

### Redeemed promotions

A redemption is **unused** while its `usedAt` is `null`. A redemption **covers** a scope when its promotion's scope is that scope or `both`.

The **oldest unused** redemption of a member for a scope is the first, in creation order of the member's redemptions, that is unused and covers the scope. Creation order is the order in which the member redeemed.

The operation that applies a discount decides whether a redemption is used: course discount for enrolling in a course and `purchaseMaterial` for a material purchase. A redemption that is used gets `usedAt` the request time and `usedOn` the `ref` of the charge it was used for (the course identifier or the purchase identifier). The charge's `discountPercent` is the redemption's `percent`. Using happens together with the operation's other effects, so a refused request uses nothing. A redemption is used at most once.

A redemption stays usable after its promotion has been ended and after the promotion's `validUntil` has passed: these matter only when redeeming.

Example: ada redeems `A` (scope `course`, 20 %), then `B` (scope `both`, 15 %), then `C` (scope `material`, 30 %). She buys material: her oldest unused redemption that covers materials is `B`, because `A` does not cover materials. The purchase gets `discountPercent` 15, and `B` gets `usedOn` the purchase identifier. She then enrolls in a course: her oldest unused redemption that covers courses is now `A`, since `B` is used.

### Promotion

A discount code that members redeem and later use once on a course or material charge.

**Fields**, in the formats of values:

- `code`: identifier
- `percent`: integer from `1` to `100` (a discount in percent)
- `scope`: a promotion scope: `course`, `material` or `both`
- `maxRedemptions`: integer from `1` to `100000` (the number of redemptions a promotion allows)
- `validUntil`: date (the last date on which the promotion can be redeemed)
- `redemptions`: count (the number of redemptions so far)
- `ended`: flag (true once the promotion has been ended)

A record is identified by `code`.

### Promotion scope

What a promotion's percent can be used for. A scope covers a charge when the table says so.

| Value | Meaning |
| --- | --- |
| `course` | Covers course charges, made by enrolling in a course. |
| `material` | Covers material purchases made with `purchaseMaterial`. |
| `both` | Covers both. |

### Redemption

One member's redemption of one promotion. It is unused while `usedAt` is `null`, and it can be used once (redeemed promotions).

**Fields**, in the formats of values:

- `code`: identifier, naming an existing promotion
- `memberId`: identifier, naming an existing member
- `percent`: integer from `1` to `100` (the promotion's percent when the redemption was made)
- `redeemedAt`: timestamp
- `usedAt`: timestamp, or `null`
- `usedOn`: string, or `null` (the `ref` of the charge the redemption was used for)

A redemption is identified by its `code` together with its `memberId`: a member redeems a code at most once.

### Add promotion

Creates the promotion with `redemptions` 0 and `ended` false.

- **Wire name:** `addPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `code`: identifier
  - `percent`: integer from `1` to `100` (a discount in percent)
  - `scope`: a promotion scope: `course`, `material` or `both`
  - `maxRedemptions`: integer from `1` to `100000` (the number of redemptions a promotion allows)
  - `validUntil`: date
- **Refusals** at check 7, in this order:
  1. `already_exists` when a promotion with `code` exists.
- **Result:**
  - `promotion`: a promotion record
- **Permissions:** admin all.
- **Audit target:** `code`.

### End promotion

Ends the promotion: `ended` becomes true, so nobody can redeem it any more. Redemptions already made stay usable (redeemed promotions).

- **Wire name:** `endPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `code`: identifier, naming an existing promotion
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the promotion's `ended` is already true.
- **Result:**
  - `promotion`: a promotion record
- **Permissions:** admin all.
- **Audit target:** `code`.

### List member promotions

Returns all redemptions of the member, used and unused, in the order they were made (creation order). A member without redemptions gives an empty list.

- **Wire name:** `listMemberPromotions`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `redemptions`: a list of redemption records
- **Permissions:** member own, staff all, admin all.

### Redeem promotion

The member redeems the promotion. A redemption is created with `percent` the promotion's percent, `redeemedAt` the request time, `usedAt` `null` and `usedOn` `null`. The promotion's `redemptions` increases by 1. Redeeming on the date `validUntil` is allowed; the day after is not.

Rules of other modules that apply here: membership and redeemed promotions.

- **Wire name:** `redeemPromotion`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `code`: identifier, naming an existing promotion
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `promotion_ended` when the promotion's `ended` is true.
  3. `promotion_expired` when the date of the request time is after `validUntil`.
  4. `promotion_exhausted` when `redemptions` is equal to `maxRedemptions`.
  5. `already_redeemed` when the member has a redemption of this `code`.
- **Result:**
  - `redemption`: a redemption record
  - `promotion`: a promotion record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/code`.

## Module `scholarships`

### Scholarships interface

What other modules may use of scholarships: its operations, the fund and award records and the refusal `already_awarded`. Training also asks for a member's active award with the remaining budget of its fund, and adds to a fund's spent amount.

- **Operations:** add fund, award scholarship, revoke scholarship, list awards.
- **Uses:** members interface.
- **Records:** fund, award.
- **Capabilities** that are not operations, as the course discount rule states them:
  - **The active award.** The caller names a member. The answer is the member's active award with its `percent` and the remaining budget of its fund (`budgetCents` less `spentCents`), or none.
  - **Spend.** The caller names a fund and an amount in cents. The fund's `spentCents` increases by the amount.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_awarded` | The member already has an active award. |

### Course discount

When a member enrolls in a course (`enrollInCourse`, also when `promoteWaitlist` enrolls a member), the course charge's `discountPercent` is decided from two sources, using the course's `feeCents` as the charge's unit price and a quantity of 1:

- **The award.** The member's active award has a percent `p`. Its **discount** is `feeCents − amountCents`, where `amountCents` is the amount the fee would have at `p` by money. The award is **covered** when the discount is at most the remaining budget of its fund, which is `budgetCents` less `spentCents`. A discount equal to the remaining budget is covered. An award that is not covered, or a member without an active award, gives no candidate.
- **The promotion.** The member's oldest unused redemption that covers courses (redeemed promotions) has a percent `q`. A member without one gives no candidate.

The candidate with the higher percent wins; on equal percent the award wins. The comparison is of percents only, whatever amounts result. With one candidate it wins; with none the `discountPercent` is 0. A course with a fee of 0 uses neither source and has `discountPercent` 0.

Only the winner is used. If the award wins, the discount is added to the fund's `spentCents`. If the promotion wins, the redemption gets `usedAt` the request time and `usedOn` the course identifier. The loser is not touched: an award that is not covered gives no discount, does not change the fund, and no refusal results.

Example: course fee 12000. `ada` has an award of 30 % from a fund with budget 10000 and spent 6400. The discount is `12000 − 8400 = 3600`, and the remaining budget is `3600`, so the award is covered. She also has an unused redemption of 30 % that covers courses. The percents are equal, so the award wins: `discountPercent` is 30, the fund's `spentCents` becomes 10000, and the redemption stays unused. With the fund's spent amount at 6401 the award is not covered; the promotion is the only candidate and wins, and the redemption is used with `usedOn` the course identifier. With a redemption of 25 % and the award covered, the award's 30 % is higher and wins; with a redemption of 40 %, the redemption wins and the fund is untouched.

### Award

A scholarship of one member from one fund, generated identifier `aws-N`. It gives a discount of `percent` on course fees while it is active (course discount). A member has at most one active award.

**Fields**, in the formats of values:

- `awardId`: identifier
- `fundId`: identifier, naming an existing fund
- `memberId`: identifier, naming an existing member
- `percent`: integer from `1` to `100` (a discount in percent)
- `status`: `active` or `revoked`

A record is identified by `awardId`.

### Fund

A budget from which course discounts of awards are paid. `spentCents` is the sum of the discounts given so far and starts at 0.

**Fields**, in the formats of values:

- `fundId`: identifier
- `budgetCents`: integer from `1` to `100000000` cents (the budget of a fund)
- `spentCents`: cent amount

A record is identified by `fundId`.

### Add fund

Creates the fund with `spentCents` 0.

- **Wire name:** `addFund`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `fundId`: identifier
  - `budgetCents`: integer from `1` to `100000000` cents (the budget of a fund)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a fund with `fundId` exists.
- **Result:**
  - `fund`: a fund record
- **Permissions:** admin all.
- **Audit target:** `fundId`.

### Award scholarship

Awards the member a scholarship from the fund. The award is created with the next award identifier and status `active`. The member's standing is not checked and the fund's amounts do not change.

- **Wire name:** `awardScholarship`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `fundId`: identifier, naming an existing fund
  - `memberId`: identifier, naming an existing member
  - `percent`: integer from `1` to `100` (a discount in percent)
- **Refusals** at check 7, in this order:
  1. `already_awarded` when the member has an award with status `active`.
- **Result:**
  - `award`: a award record
- **Permissions:** admin all.
- **Audit target:** the new `awardId`.

### List awards

Returns all awards of the member, active and revoked, in creation order (ascending award number). A member without awards gives an empty list.

- **Wire name:** `listAwards`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `awards`: a list of award records
- **Permissions:** member own, staff all, admin all.

### Revoke scholarship

Revokes the award: its status becomes `revoked`. The fund's `spentCents` does not change, and discounts already given stay. A revoked award gives no further discounts, and the member can be awarded again.

- **Wire name:** `revokeScholarship`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `awardId`: identifier, naming an existing award
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the award's status is `revoked`.
- **Result:**
  - `award`: a award record
- **Permissions:** admin all.
- **Audit target:** `awardId`.

## Module `refunds`

### Refunds interface

What other modules may use of refunds: its operations, the refund record and the refusals below. Finance also asks how much has been refunded on a line of an invoice.

- **Operations:** issue refund, list refunds.
- **Uses:** members interface, billing interface, ledger interface.
- **Records:** refund.
- **Capabilities** that are not operations:
  - **Refunded on a line.** The caller names an invoice and a line number. The answer is the sum of `amountCents` of all refunds of that invoice and line, and 0 when there is none.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `invalid_line` | The invoice has no line with the given number. |
| `invoice_not_paid` | The invoice's status is not `paid`. |
| `refund_too_large` | The amount is more than what is left of the line after its earlier refunds. |

### Refund

A refund of part or all of one line of a paid invoice, given as credit, generated identifier `rfd-N`. The invoice and its line are not changed.

**Fields**, in the formats of values:

- `refundId`: identifier
- `invoiceId`: identifier, naming an existing invoice
- `lineNumber`: integer from `1` to `1000` (the position of the line in the invoice's `lines`; 1 is the first line)
- `amountCents`: integer from `1` to `100000000` cents (a refund amount)
- `reason`: text
- `at`: timestamp

A record is identified by `refundId`.

### Issue refund

Refunds `amountCents` of one line of the paid invoice. The line is the one at position `lineNumber` in the invoice's `lines`; position 1 is the membership line. A refund is created with the next refund identifier and `at` the request time. The credit balance of the invoice's member increases by `amountCents`, and the entry is posted as the postings of the ledger state. `creditBalanceCents` is the member's credit balance after the change. The invoice, its lines and its status do not change. The member's standing is not checked.

The **left** of a line is the line's `amountCents` less the sum of `amountCents` of the earlier refunds of the same invoice and line. A line can be refunded in several parts until nothing is left; a line of 0 cents has nothing left.

Rules of other modules that apply here: postings.

- **Wire name:** `issueRefund`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `invoiceId`: identifier, naming an existing invoice
  - `lineNumber`: integer from `1` to `1000` (the position of a line in an invoice; 1 is the first line)
  - `amountCents`: integer from `1` to `100000000` cents (a refund amount)
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invoice_not_paid` when the invoice's status is not `paid`.
  2. `invalid_line` when `lineNumber` is greater than the number of lines of the invoice.
  3. `refund_too_large` when `amountCents` is greater than what is left of the line. An amount equal to what is left is allowed.
- **Result:**
  - `refund`: a refund record
  - `creditBalanceCents`: cent amount
- **Permissions:** admin all.
- **Audit target:** the new `refundId`.

### List refunds

Returns all refunds of the invoice, in creation order (ascending refund number). An invoice without refunds gives an empty list.

- **Wire name:** `listRefunds`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `invoiceId`: identifier, naming an existing invoice
- **Result:**
  - `invoiceId`: identifier
  - `refunds`: a list of refund records
- **Permissions:** member own, staff all, admin all.

## Module `dunning`

### Dunning interface

What other modules may use of dunning: its operations, the ladder, case and action records, and the refusals below. No other module asks anything else of it.

- **Operations:** set dunning ladder, run dunning, list dunning cases.
- **Uses:** members interface, billing interface, notices interface.
- **Records:** dunning ladder, dunning case, dunning action.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `invalid_ladder` | The days of a ladder are not strictly increasing from notice to fee to suspension. |
| `no_ladder` | No ladder has been set. |

### Dunning run

A run takes every invoice that is overdue at the request time (overdue invoices), in ascending invoice number: the number `N` of `inv-N`, compared as a number. For each invoice it takes the steps that are due, in the order below, all of them in the same run when all are due.

The **days overdue** of an invoice are the whole days from its `dueAt` to the request time, rounded down: the number of complete 1440-minute periods between the two timestamps. An invoice overdue for less than one day has 0 days overdue and no step is due, because every ladder day is at least 1.

A step is due when the days overdue are at least its ladder days and the step has not been taken for this invoice before. A step is taken at most once per invoice, whatever the ladder is later. The steps:

1. **Notice**, at `noticeDays` or more. A notice of kind `dunning` is posted for the invoice's member through notices interface with `ref` the invoice identifier and the subject `Invoice <invoiceId> is overdue`.
2. **Fee**, at `feeDays` or more. A late fee charge is created for the member: `kind` `lateFee`, `ref` the invoice identifier, `quantity` 1, `unitPriceCents` the ladder's `feeCents`, `discountPercent` 0 and `at` the request time. The charge is unbilled and is invoiced like any other; it does not change the overdue invoice.
3. **Suspension**, at `suspendDays` or more. The member's standing becomes `suspended`. When the member is already suspended, the standing does not change, but the step still counts as taken. A notice of kind `suspension` is posted with `ref` the invoice identifier; its subject is the one notice kind states.

For each step taken, an action is reported with the invoice identifier and the step (`notice`, `fee` or `suspend`). The invoice's case is created in the first run that takes a step for the invoice, and the step's timestamp (`noticedAt`, `feeChargedAt` or `suspendedAt`) is set to the request time.

Paying the invoice later undoes nothing and reinstates nobody. A member who was reinstated is not suspended again for a step already taken. A paid invoice is not overdue and is not visited.

Example: the ladder is `noticeDays` 3, `feeDays` 7, `suspendDays` 14. Invoice `inv-1` is open with `dueAt` `2026-04-15T08:00Z`. A run at `2026-04-18T07:59Z` finds 2 days overdue and takes no step. A run at `2026-04-18T08:00Z` finds 3 days: it posts the notice, creates `dun-1` with `noticedAt` set and reports `inv-1` `notice`. A run at `2026-04-30T08:00Z` finds 15 days: it charges the fee and suspends the member, reporting `inv-1` `fee` and `inv-1` `suspend` in that order. Another run at the same time reports nothing.

### Dunning action

One step taken by a run for one invoice.

**Fields**, in the formats of values:

- `invoiceId`: identifier, naming an existing invoice
- `step`: one of `notice`, `fee` or `suspend`

### Dunning case

The record of the steps taken for one overdue invoice, generated identifier `dun-N`. There is at most one case per invoice. It is created in the first run that takes a step for the invoice (dunning run). Each timestamp is `null` until its step is taken, and is then the request time of the run that took it.

**Fields**, in the formats of values:

- `caseId`: identifier
- `invoiceId`: identifier, naming an existing invoice
- `memberId`: identifier, naming an existing member (the invoice's member)
- `noticedAt`: timestamp, or `null`
- `feeChargedAt`: timestamp, or `null`
- `suspendedAt`: timestamp, or `null`

A record is identified by `caseId`.

### Dunning ladder

The steps taken against an overdue invoice and the days at which they are taken. There is at most one ladder; none exists until one is set, and setting a ladder replaces it.

**Fields**, in the formats of values:

- `noticeDays`: integer from `1` to `3650` (days overdue at which the notice is posted)
- `feeDays`: integer from `1` to `3650` (days overdue at which the late fee is charged)
- `feeCents`: integer from `1` to `1000000` cents (the late fee)
- `suspendDays`: integer from `1` to `3650` (days overdue at which the member is suspended)

`noticeDays` is less than `feeDays`, and `feeDays` is less than `suspendDays`.

### List dunning cases

Returns all dunning cases of the member's invoices, in creation order (ascending case number). A member without cases gives an empty list.

- **Wire name:** `listDunningCases`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `cases`: a list of dunning case records
- **Permissions:** member own, staff all, admin all.

### Run dunning

Takes the steps of the ladder that are due for the invoices overdue at the request time, as the dunning run rule states, with all effects together. `actions` lists the steps taken, in the order they were taken. It is empty when nothing is due.

Rules of other modules that apply here: dunning run, overdue invoices, charges and invoices and membership.

- **Wire name:** `runDunning`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Refusals** at check 7, in this order:
  1. `no_ladder` when no ladder has been set.
- **Result:**
  - `actions`: a list of dunning action records
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `dunning`.

### Set dunning ladder

Sets the ladder, replacing the one before. Cases that exist are not changed. The new ladder applies to the runs after this request.

- **Wire name:** `setDunningLadder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `noticeDays`: integer from `1` to `3650` (days overdue at which the notice is posted)
  - `feeDays`: integer from `1` to `3650` (days overdue at which the late fee is charged)
  - `feeCents`: integer from `1` to `1000000` cents (the late fee)
  - `suspendDays`: integer from `1` to `3650` (days overdue at which the member is suspended)
- **Refusals** at check 7, in this order:
  1. `invalid_ladder` unless `noticeDays` is less than `feeDays` and `feeDays` is less than `suspendDays`.
- **Result:**
  - `ladder`: a dunning ladder record
- **Permissions:** admin all.
- **Audit target:** the fixed string `ladder`.

## Module `lockers`

### Lockers interface

What other modules may use of lockers: its operations, the locker and rental records and the refusals below. No other module asks anything else of it.

- **Operations:** add locker, rent locker, end locker rental, bill lockers.
- **Uses:** members interface, billing interface.
- **Records:** locker, rental.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_billed` | The lockers of the month have already been billed. |
| `locker_taken` | The locker has an active rental. |
| `not_current_month` | The month is not the month of the request time. |
| `rental_limit` | The member already has 2 active rentals. |

### Locker

A locker that members can rent by the month.

**Fields**, in the formats of values:

- `lockerId`: identifier
- `monthlyCents`: integer from `1` to `1000000` cents (the monthly rent of a locker)

A record is identified by `lockerId`.

### Rental

The rental of one locker by one member, generated identifier `rnt-N`. A locker has at most one active rental. `endedAt` is `null` while the rental is active.

**Fields**, in the formats of values:

- `rentalId`: identifier
- `lockerId`: identifier, naming an existing locker
- `memberId`: identifier, naming an existing member
- `status`: `active` or `ended`
- `startedAt`: timestamp
- `endedAt`: timestamp, or `null`

A record is identified by `rentalId`.

### Add locker

Creates the locker.

- **Wire name:** `addLocker`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `lockerId`: identifier
  - `monthlyCents`: integer from `1` to `1000000` cents (the monthly rent of a locker)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a locker with `lockerId` exists.
- **Result:**
  - `locker`: a locker record
- **Permissions:** admin all.
- **Audit target:** `lockerId`.

### Bill lockers

Charges the rent for the month. One locker charge is created for every rental with status `active` at the request time, in ascending `lockerId` order: `kind` `locker`, `ref` the rental identifier, `quantity` 1, `unitPriceCents` the locker's `monthlyCents` at the request time, `discountPercent` 0 and `at` the request time. The whole rent is charged, whenever in the month the rental started and whatever the member's standing is. The month is then marked as billed, even when there was no active rental. `charges` lists the charges created; it is empty when there was no active rental.

Rules of other modules that apply here: charges and invoices.

- **Wire name:** `billLockers`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `not_current_month` when `month` is not the month of the request time.
  2. `already_billed` when an earlier request to bill the lockers of this month was accepted.
- **Result:**
  - `month`: month
  - `charges`: a list of charge records
- **Permissions:** staff all, admin all.
- **Audit target:** `month`.

### End locker rental

Ends the rental: its status becomes `ended` and `endedAt` the request time. The locker can be rented again. Charges already created stay. The member's standing is not checked.

- **Wire name:** `endLockerRental`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `rentalId`: identifier, naming an existing rental
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the rental's status is `ended`.
- **Result:**
  - `rental`: a rental record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `rentalId`.

### Rent locker

Rents the locker to the member. A rental is created with the next rental identifier, status `active`, `startedAt` the request time and `endedAt` `null`. Nothing is charged by this request; bill lockers charges the rent.

Rules of other modules that apply here: membership.

- **Wire name:** `rentLocker`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `lockerId`: identifier, naming an existing locker
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `locker_taken` when the locker has a rental with status `active`.
  3. `rental_limit` when the member has 2 rentals with status `active`.
- **Result:**
  - `rental`: a rental record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `rentalId`.

## Module `guests`

### Guests interface

What other modules may use of guests: its operations, the guest allowance and guest pass records and the refusal codes below. Other modules also ask for a guest pass by its identifier, with its member, date and status.

- **Operations:** set guest allowance, issue guest pass, revoke guest pass, list guest passes.
- **Uses:** members interface, billing interface.
- **Records:** guest allowance, guest pass.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `date_in_past` | The date is before the date of the request time. |
| `guests_not_allowed` | The member's tier has no guest allowance. |

### Guest passes

A member's tier decides the guest allowance. The passes of a member that **count** for a month are the member's guest passes with status `issued` whose `date` lies in that month; a revoked pass does not count. The month of a pass is the month of its `date`, not of the request time.

A new pass is **free** when the member's tier has an allowance and fewer than `perMonth` passes count for the month of its date. Otherwise, if the tier has an allowance, it is **charged**. With `perMonth` 0 every pass is charged. The tier is the member's tier at the request time. A pass that is charged has `charged` true and one charge; its `feeCents` may be 0, and the charge of 0 cents is still created. Revoking a pass takes no charge back and does not change its `charged` field.

Example: the tier has `perMonth` 2 and `feeCents` 500. The member's passes for `2026-06-10`, `2026-06-11` and `2026-06-12` are free, free and charged 500. After the first is revoked, 2 passes count for June, so a pass for `2026-06-20` is charged. A pass for `2026-07-01` is free, because no pass counts for July.

### Guest allowance

How many guest passes a month a tier includes, and the fee for each further one. A tier has an allowance only after set guest allowance has set one; a tier without one allows no guests.

**Fields**, in the formats of values:

- `tier`: a tier: `basic`, `plus` or `pro`
- `perMonth`: integer from `0` to `100` (the guest passes of one month that are free)
- `feeCents`: integer from `0` to `1000000` cents (the fee of a guest pass beyond the free ones)

A record is identified by `tier`.

### Guest pass

The permission for one guest to visit with a member on one date, generated identifier `gst-N`.

**Fields**, in the formats of values:

- `guestPassId`: identifier
- `memberId`: identifier, naming an existing member
- `guestName`: text
- `date`: date
- `charged`: flag (true when a fee was charged for the pass)
- `status`: `issued` or `revoked`

A record is identified by `guestPassId`.

### Issue guest pass

Creates the guest pass with the next guest pass identifier, the supplied guest name and date and status `issued`. `charged` is set as guest passes states. When the pass is charged, a charge is created for the member: `kind` `guest`, `ref` the new guest pass identifier, `quantity` 1, `unitPriceCents` the allowance's `feeCents`, `discountPercent` 0, `at` the request time.

Rules of other modules that apply here: charges and invoices and membership.

- **Wire name:** `issueGuestPass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `guestName`: text
  - `date`: date
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `date_in_past` when `date` is before the date of the request time; today is allowed.
  3. `guests_not_allowed` when the member's tier has no allowance.
- **Result:**
  - `guestPass`: a guest pass record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `guestPassId`.

### List guest passes

Returns all guest passes of the member, revoked or not, in creation order.

- **Wire name:** `listGuestPasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `guestPasses`: a list of guest pass records
- **Permissions:** member own, staff all, admin all.

### Revoke guest pass

The guest pass gets status `revoked`. No charge is taken back. A pass belongs to the actor when its member is the actor.

- **Wire name:** `revokeGuestPass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `guestPassId`: identifier, naming an existing guest pass
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the pass is already revoked or its `date` is before the date of the request time.
- **Result:**
  - `guestPass`: a guest pass record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `guestPassId`.

### Set guest allowance

Sets the allowance of the tier, replacing an earlier one. Passes that exist are not changed.

- **Wire name:** `setGuestAllowance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `tier`: a tier: `basic`, `plus` or `pro`
  - `perMonth`: integer from `0` to `100` (the guest passes of one month that are free)
  - `feeCents`: integer from `0` to `1000000` cents (the fee of a guest pass beyond the free ones)
- **Result:**
  - `allowance`: a guest allowance record
- **Permissions:** admin all.
- **Audit target:** `tier`.

## Module `volunteering`

### Volunteering interface

What other modules may use of volunteering: its operations, the volunteer time entry record and the refusal codes below. No other module asks anything of it beyond that. Volunteering in turn asks billing to add credit to a member, which posts the entry to the ledger, and staffing whether a user has a scheduled shift that contains a point in time.

- **Operations:** set volunteer rate, log volunteer time, decide volunteer time, list volunteer time.
- **Uses:** members interface, billing interface, ledger interface, staffing interface.
- **Records:** volunteer time entry.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `approver_not_on_shift` | The actor is staff and has no scheduled shift at the request time. |
| `date_in_future` | The date is after the date of the request time. |
| `volunteer_cap_reached` | Approving the entry would take the member's approved minutes of the month above 1200. |

### Volunteer credit

The **rate** is a number of cents per hour from `0` to `100000`. It starts at 0 and is changed only by set volunteer rate. An entry is paid at the rate in force when it is approved; a later change does not touch decided entries.

The credit of an approved entry is `creditCents = (minutes × centsPerHour + 30) div 60`, which is the exact value rounded half up to a whole cent. It is added to the member's credit balance through billing, which posts it as postings states. With a rate of 0 the credit is 0, the entry is still approved, and nothing is posted.

The **cap**: the minutes of a member's entries with status `approved` and a `date` in a month are at most 1200. An approval is refused when the approved minutes of that member in the month of the entry's `date`, plus the entry's `minutes`, would be above 1200; reaching exactly 1200 is allowed. Pending and rejected entries do not count. The month is the month of the entry's `date`, not of the request time.

Example: at 1500 cents per hour, 100 minutes give `(100 × 1500 + 30) div 60` = 2500 cents. At 1 cent per hour, 30 minutes give `(30 × 1 + 30) div 60` = 1 cent. A member with 1100 approved minutes in June can have a 100-minute entry of June approved, and then no more.

### Volunteer time entry

Time a member volunteered, logged by or for the member and decided by staff, generated identifier `vol-N`. `creditCents` is `null` until the entry is approved.

**Fields**, in the formats of values:

- `entryId`: identifier
- `memberId`: identifier, naming an existing member
- `date`: date
- `minutes`: integer from `1` to `720` (the minutes volunteered in one entry)
- `note`: text
- `status`: `pending`, `approved` or `rejected`
- `creditCents`: cent amount, or `null`

A record is identified by `entryId`.

### Decide volunteer time

Sets the entry's status to the decision. A rejection leaves `creditCents` `null`. An approval sets `creditCents` and credits the member as volunteer credit states. The `entry` result is the entry after the change.

The actor is on shift when the actor is staff and has a scheduled shift of their own with `start` at or before the request time and `end` after it. An admin does not need a shift.

Rules of other modules that apply here: postings and money.

- **Wire name:** `decideVolunteerTime`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `entryId`: identifier, naming an existing volunteer time entry
  - `decision`: `approved` or `rejected`
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the entry's status is not `pending`.
  2. `approver_not_on_shift` when the actor is staff and is not on shift.
  3. `volunteer_cap_reached` when the decision is `approved` and the cap of volunteer credit would be exceeded.
- **Result:**
  - `entry`: a volunteer time entry record
- **Permissions:** staff all, admin all.
- **Audit target:** `entryId`.

### List volunteer time

Returns all entries of the member in every status, in creation order.

- **Wire name:** `listVolunteerTime`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `entries`: a list of volunteer time entry records
- **Permissions:** member own, staff all, admin all.

### Log volunteer time

Creates the entry with the next entry identifier, status `pending` and `creditCents` `null`. Standing is not checked.

- **Wire name:** `logVolunteerTime`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `date`: date
  - `minutes`: integer from `1` to `720` (the minutes volunteered in one entry)
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `date_in_future` when `date` is after the date of the request time; today is allowed.
- **Result:**
  - `entry`: a volunteer time entry record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `entryId`.

### Set volunteer rate

Sets the rate that later approvals pay. Entries that are decided already keep their credit. Setting the current value succeeds without a change.

- **Wire name:** `setVolunteerRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `centsPerHour`: integer from `0` to `100000` cents per hour (the credit for one hour of volunteering)
- **Result:**
  - `centsPerHour`: cent amount
- **Permissions:** admin all.
- **Audit target:** the fixed string `rate`.

## Module `training`

### Training interface

What other modules may use of training: its operations, the course, enrollment and certification records, and the refusal `not_certified`. Other modules also ask whether a member holds a certification for a family that is valid on a date, for the family of a course, for a member's enrollment in a course, for the free places of a course, and to enroll a member in a course. Training in turn needs to know whether a member still owes attendance for a course and whether a member has passed the assessment of a course; it declares these needs, and the protocol module supplies the answers from classes and assessments.

- **Operations:** add course, enroll in course, complete course, revoke certification, list certifications.
- **Uses:** members interface, equipment interface, billing interface, guardians interface, promotions interface, scholarships interface.
- **Records:** course, enrollment, certification.

What other modules ask of training that is not an operation:

- Whether a member holds a certification for a family that is valid on a date, as certifications states.
- The family of a course.
- A member's enrollment in a course: whether one exists and, when it does, its `status` and `enrolledAt`, and the number of times the member has enrolled in the course so far. The first enrollment is number 1, and each later enrollment, which reuses and resets the record after a completion, counts once more.
- The free places of a course: its `capacity` less its `enrolledCount`.
- Enrolling a member in a course at a request time, exactly as enroll in course does. It makes the refusals of that operation in their order, from `member_suspended` to `course_full`. When none applies it makes every effect of the operation, the enrollment, the charge and the discounts included, and answers the enrollment. When one applies it answers the refusal code and changes nothing. It is not an operation: the checks of roles and ownership and the audit entry belong to the operation that asks.

What training needs from other modules, which the protocol module supplies:

- From classes: whether a member still owes attendance for a course, given the course, the member, the `enrolledAt` of the member's enrollment and the request time (attendance gate).
- From assessments: whether a member has met the assessment requirement of a course, given the course and the member (assessment gate).

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_enrolled` | The member is already enrolled in the course. |
| `course_full` | The course has no free place. |
| `not_certified` | The member has no certification valid on the relevant date for the machine's family, or there is no certification to revoke. |
| `not_enrolled` | The member is not enrolled in the course. Where an operation of another module names a different condition, that operation states it. |

Training's operations also refuse with `guardian_consent_missing` (guardians interface), `attendance_missing` (classes interface) and `assessment_not_passed` (assessments interface), whose meanings those interfaces state.

### Certifications

A member has at most one certification per tool family. Completing a course issues it: `issuedOn` is the date of the request time and `expiresOn` is 365 days after `issuedOn`. If the member already has a certification for that family, whether valid, expired or revoked, it is replaced by the newly issued one.

A certification is **valid on date D** when it is not revoked and `issuedOn ≤ D < expiresOn`. It is not valid on its `expiresOn` date. Its reported `status` at a request time is `revoked` when revoked, otherwise `expired` when the date of the request time is on or after `expiresOn`, otherwise `valid`.

### Courses and enrollments

A course teaches the safe use of one tool family. It has a fee and a capacity. A member enrolls; staff record completion, which certifies the member for the course's family. There is one enrollment record per course and member. It is created by the first enrollment; a later enrollment after completion reuses and resets it.

The course's `enrolledCount` is the number of its enrollments with status `enrolled`. A place becomes free when an enrollment is completed.

### Certification

A member's certification for one tool family.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `familyId`: identifier, naming an existing family
- `issuedOn`: date
- `expiresOn`: date
- `status`: one of `valid`, `expired` or `revoked`

### Course

A course that teaches the safe use of one tool family, with a fee and a capacity.

**Fields**, in the formats of values:

- `courseId`: identifier
- `familyId`: identifier, naming an existing family
- `title`: text
- `feeCents`: integer from `0` to `1000000` cents (a course fee)
- `capacity`: integer from `1` to `1000` (the number of places of a course)
- `enrolledCount`: count

A record is identified by `courseId`.

### Enrollment

The enrollment of one member in one course. `completedAt` is `null` while enrolled.

**Fields**, in the formats of values:

- `courseId`: identifier, naming an existing course
- `memberId`: identifier, naming an existing member
- `status`: `enrolled` or `completed`
- `enrolledAt`: timestamp
- `completedAt`: timestamp, or `null`

### Add course

Creates the course with `enrolledCount` 0.

- **Wire name:** `addCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier
  - `familyId`: identifier, naming an existing family
  - `title`: text
  - `feeCents`: integer from `0` to `1000000` cents (a course fee)
  - `capacity`: integer from `1` to `1000` (the number of places of a course)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a course with `courseId` exists.
- **Result:**
  - `course`: a course record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.

### Complete course

Records that the member completed the course: the enrollment gets status `completed` and `completedAt` equal to the request time. The member's certification for the course's family is issued as certifications states. Standing is not checked.

Rules of other modules that apply here: attendance gate and assessment gate.

- **Wire name:** `completeCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `not_enrolled` when the member has no enrollment in this course with status `enrolled`.
  2. `attendance_missing` when a class of the course requires the member's attendance and the member did not attend it (attendance gate).
  3. `assessment_not_passed` when the course has an assessment and the member has not passed it in the current enrollment period (assessment gate).
- **Result:**
  - `enrollment`: a enrollment record
  - `certification`: a certification record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId/memberId`.

### Enroll in course

Enrolls the member in the course. The enrollment gets status `enrolled`, `enrolledAt` equal to the request time and `completedAt` `null`. A course charge is created for the member: `kind` `course`, `ref` the course identifier, `quantity` 1, `unitPriceCents` the course fee, `discountPercent` the percent that course discount decides (0 when neither a scholarship award nor a redeemed promotion applies), `at` the request time. A course with fee 0 creates a charge of 0 cents. The `course` result is the course after the change.

Rules of other modules that apply here: charges and invoices, membership, guardian consent, course discount and redeemed promotions.

- **Wire name:** `enrollInCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `guardian_consent_missing` when the member is a minor on the date of the request time and has no consent for the course's family (guardian consent).
  3. `already_enrolled` when the member's enrollment in this course has status `enrolled`.
  4. `course_full` when `enrolledCount` equals `capacity`.
- **Result:**
  - `enrollment`: a enrollment record
  - `course`: a course record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.

### List certifications

Returns all of the member's certifications, including expired and revoked ones.

- **Wire name:** `listCertifications`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `certifications`: a list of certification records
- **Permissions:** member own, staff all, admin all.

### Revoke certification

Withdraws the member's certification, for example after a safety incident. The certification becomes revoked; its dates do not change. Existing reservations stay, but check-in requires a valid certification. An expired certification can be revoked.

- **Wire name:** `revokeCertification`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `not_certified` when the member has no certification for the family or it is already revoked.
- **Result:**
  - `certification`: a certification record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/familyId`.

## Module `materials`

### Materials interface

What other modules may use of materials: its operations and the material and purchase records. Other modules also ask materials for:

- a material's existence, its current unit price and its current stock;
- the units a member bought of a material in a month: the sum of `quantity` over the member's purchases of that material whose `at` lies in that month;
- a purchase's member and amount: for a purchase identifier, the purchase's `memberId` and `amountCents`;
- adding received stock: the stock of a material increases by a quantity;
- setting stock from a count: the stock of a material becomes a given number of units;
- drawing stock for consumables: for a material and a number of units, the stock decreases by the smaller of the units and the stock, and the number drawn is answered; it may be 0;
- whether a material has a count in an open stocktake. Materials answers this from the answer of stocktake that it needs, as below.

Materials in turn needs two answers from modules that depend on it. From hazmat it needs, for a member, a material, a quantity and the request time, which refusal of the hazard gate applies to a purchase, if any. From stocktake it needs whether a material has a count in an open stocktake. It declares these needs, and the protocol module supplies the answers from hazmat interface and stocktake interface.

- **Operations:** add material, restock material, purchase material, list materials.
- **Uses:** members interface, billing interface, promotions interface.
- **Records:** material, purchase.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `insufficient_stock` | The material's stock is lower than the requested quantity. |

`purchaseMaterial` also answers `hazard_not_certified` and `hazard_limit_exceeded`, which hazmat interface states, and `stocktake_open`, which stocktake interface states. `restockMaterial` answers `stocktake_open`.

### Shop

The shop sells materials by unit. Each material has a unit price and a stock count, which starts at 0. Stock never goes below 0.

### Material

A material sold by unit, with a unit price and a stock count.

**Fields**, in the formats of values:

- `materialId`: identifier
- `name`: text
- `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- `stock`: count

A record is identified by `materialId`.

### Purchase

A sale of material to a member, generated identifier `pur-N`.

**Fields**, in the formats of values:

- `purchaseId`: identifier
- `memberId`: identifier, naming an existing member
- `materialId`: identifier, naming an existing material
- `quantity`: integer from `1` to `10000` (the number of units bought in one request)
- `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp

A record is identified by `purchaseId`.

### Add material

Creates the material with stock 0.

- **Wire name:** `addMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `materialId`: identifier
  - `name`: text
  - `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a material with `materialId` exists.
- **Result:**
  - `material`: a material record
- **Permissions:** staff all, admin all.
- **Audit target:** `materialId`.

### List materials

Returns all materials.

- **Wire name:** `listMaterials`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `materials`: a list of material records
- **Permissions:** member all, staff all, admin all.

### Purchase material

Sells material to the member. The stock decreases by `quantity`. A purchase is created with the next purchase identifier, the material's current unit price, the `discountPercent` below, the amount by money and `at` the request time. A material charge is created for the member: `kind` `material`, `ref` the purchase identifier, and the purchase's `quantity`, `unitPriceCents`, `discountPercent`, `amountCents` and `at`. The `material` result is the material after the change.

`discountPercent` is the `percent` of the member's oldest unused redeemed promotion whose scope covers materials (`material` or `both`), otherwise 0. That redemption is then used by this purchase, as redeemed promotions states. A refused request uses no promotion.

Example: 3 units at 250 cents with a redeemed promotion of 20 percent: `(3 × 250 × 80 + 50) div 100 = 600` cents.

Rules of other modules that apply here: charges and invoices, membership, hazard gate, stocktake freeze and redeemed promotions.

- **Wire name:** `purchaseMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `materialId`: identifier, naming an existing material
  - `quantity`: integer from `1` to `10000` (the number of units bought in one request)
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `hazard_not_certified` when the material has a classification and the member has no certification for its family valid on the date of the request time (hazard gate).
  3. `hazard_limit_exceeded` when the material has a classification and the units of the material the member bought in the month of the request time, plus `quantity`, are more than its `monthlyLimit` (hazard gate).
  4. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
  5. `insufficient_stock` when the stock is lower than `quantity`.
- **Result:**
  - `purchase`: a purchase record
  - `material`: a material record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `purchaseId`.

### Restock material

The stock increases by `quantity`.

Rules of other modules that apply here: stocktake freeze.

- **Wire name:** `restockMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `materialId`: identifier, naming an existing material
  - `quantity`: integer from `1` to `100000` (the number of units added to stock in one request)
- **Refusals** at check 7, in this order:
  1. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
- **Result:**
  - `material`: a material record
- **Permissions:** staff all, admin all.
- **Audit target:** `materialId`.

## Module `consumables`

### Consumables interface

What other modules may use of consumables: its operations, the consumption row record, and the rule that a session's consumables are drawn at check-out. Other modules also ask the capability listed below. Consumables adds no refusal codes beyond the one in the table.

- **Operations:** set consumption, clear consumption, list consumption.
- **Uses:** equipment interface, materials interface, billing interface.
- **Records:** consumption row.
- **Rules that bind other modules:** consumption.

Capability other modules ask of consumables:

- The draw for one ended session, which bookings asks at check-out. It is given the reservation identifier, the member, the machine, the metered minutes and the request time, and it does what consumption states: it decreases material stock and creates the material charges, in that rule's order. It returns the charges it created, which may be none.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `no_consumption` | The machine has no consumption row for the material. |

### Consumption

A machine can use up materials while it runs, as its consumption rows say. The draw happens once for each session, in `checkOut`, after the usage charge is created. Rows are read as they stand at that request. For each row of the machine, in ascending `materialId` order:

1. `units` is `(meteredMinutes × unitsPerHour + 59) div 60`: the metered minutes times `unitsPerHour`, divided by 60 and rounded up. It uses the full metered minutes of the reservation, whatever the member's passes covered. With 0 metered minutes `units` is 0.
2. `drawn` is the smaller of `units` and the material's stock at that moment.
3. The material's stock decreases by `drawn`.
4. When `drawn` is above 0, a material charge is created for the member: `kind` `material`, `ref` the reservation identifier, `quantity` `drawn`, `unitPriceCents` the material's current unit price, `discountPercent` 0, `at` the request time, and the amount by money. When `drawn` is 0, no charge is created.

A shortfall, where `drawn` is below `units`, is not an error: the session is still checked out and the missing units are neither charged nor carried over. No promotion applies. The draw creates no purchase, and an open stocktake does not stop it. A machine with no rows draws nothing.

Example: a session of 50 metered minutes on a row of 25 units per hour has `units` `(50 × 25 + 59) div 60 = 1309 div 60 = 21`. With a stock of 15, `drawn` is 15, the stock becomes 0 and the charge has quantity 15.

### Consumption row

How fast one machine uses up one material while it runs; see consumption. A machine has at most one row per material.

**Fields**, in the formats of values:

- `machineId`: identifier, naming an existing machine
- `materialId`: identifier, naming an existing material
- `unitsPerHour`: integer from `1` to `1000` (the units of the material used in one hour)

A record is identified by `machineId` and `materialId` together.

### Clear consumption

Removes the row of the machine and the material. Charges and stock draws made earlier stay as they are.

- **Wire name:** `clearConsumption`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `materialId`: identifier, naming an existing material
- **Refusals** at check 7, in this order:
  1. `no_consumption` when the machine has no row for the material.
- **Result:**
  - `machineId`: identifier
  - `materialId`: identifier
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId/materialId`.

### List consumption

Returns the rows of the machine in `materialId` ascending order.

- **Wire name:** `listConsumption`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
- **Result:**
  - `machineId`: identifier
  - `rows`: a list of consumption row records
- **Permissions:** staff all, admin all.

### Set consumption

Sets the row of the machine and the material. A machine with no row for the material gets one; a machine with a row has it replaced, so `unitsPerHour` is the supplied value. Setting the current value succeeds without a change.

Rules of other modules that apply here: consumption.

- **Wire name:** `setConsumption`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `materialId`: identifier, naming an existing material
  - `unitsPerHour`: integer from `1` to `1000` (the units of the material used in one hour)
- **Result:**
  - `row`: a consumption row record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId/materialId`.

## Module `bookings`

### Bookings interface

What other modules may use of bookings: its operations and the reservation record. Other modules also ask whether an active reservation of a machine overlaps an interval, and for the completed sessions of a machine. Bookings in turn needs to know whether a member has an active badge that is not inside; it declares that need, and the protocol module supplies the answer from access.

- **Operations:** create reservation, cancel reservation, check in, check out, list member reservations, get machine schedule.
- **Uses:** members interface, training interface, equipment interface, billing interface, guardians interface, waivers interface, PPE interface, lockouts interface, inspections interface, staffing interface, passes interface, consumables interface.
- **Records:** reservation, opening hours of day.

Capabilities other modules ask of bookings. None of them writes an audit entry of its own; the operation that uses it is audited once. "Active" and the reported status are as reservations states them, at the request time.

- Whether an active reservation of a machine overlaps an interval `[start, end)`. Equipment asks this.
- The completed sessions of a machine, each with its `checkedInAt` and `checkedOutAt`. Reporting asks this.
- The active reservations of all machines of a family, each with its machine, `start` and `end`. Staffing asks this.
- The reservations of a machine that are still to come, which are those whose reported status is `reserved` and whose `start` is after the request time, in ascending `start` order, then creation order. Escalations asks this.
- Cancelling a reservation whose reported status is `reserved` as part of another operation. The reservation gets status `cancelled` exactly as cancel reservation states, with no fee and no credit. Escalations uses this.
- A reservation by its identifier, as the reservation record with its reported status. Projects asks this.
- The opening hours of a date, as an opening hours of day record for that date's day of the week, or none on a Sunday (opening hours). Access asks this to judge whether a time lies in the opening period of its date.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `check_in_too_early` | The reservation has not started yet. |
| `invalid_slot` | A reservation does not follow the booking granularity or its duration limits. |
| `machine_booked` | Another active reservation of the machine overlaps the reservation. |
| `machine_in_maintenance` | A maintenance window of the machine overlaps the reservation. |
| `member_double_booked` | Another active reservation of the member overlaps the reservation. |
| `outside_opening_hours` | A reservation is not inside the opening hours of its day. |

create reservation and check in also return codes that other modules add: `guardian_consent_missing`, `waiver_missing`, `machine_locked_out`, `inspection_due`, `supervisor_required`, `ppe_missing` and `not_on_premises`. Each is explained in the interface of the module that adds it.

### Booking granularity

**Booking granularity is 30 minutes.** A reservation's start and end must each fall on a minute that is a multiple of 30 within the hour (`:00` or `:30`). A reservation lasts at least one granularity step (30 minutes) and at most 240 minutes.

### Metering

The member checks in when starting to use the machine and checks out when done. The program meters the minutes in between and charges them at the rate captured at check-in, with the member's tier discount.

The member's passes pay for metered minutes first: the minutes a pass draws are not charged (pass draw). The charge is for the remaining, billable minutes. The materials that the machine uses up are charged on the full metered minutes, whatever the passes drew (consumption).

Example: `ada` has tier `plus` (usage discount 10 %). `laser-1` has a rate of 40 cents per minute and the consumption rows `acrylic` at 25 units per hour and `ply` at 6 units per hour. The stock of `acrylic` is 15 units at 30 cents, the stock of `ply` is 100 units at 12 cents. `ada` holds one valid pass with 20 minutes left. She checks in at `2026-03-03T10:00Z` and checks out at `2026-03-03T10:50Z`. `checkOut` then does this, in this order:

1. `meteredMinutes` is 50. The pass draws 20 minutes and has 0 left, so `billable` is 30.
2. The usage charge has quantity 30, unit price 40 and discount 10: `(30 × 40 × 90 + 50) div 100 = 1080` cents. The reservation's `amountCents` is 1080, and the result's `charge` is this charge.
3. `acrylic` has `units` `(50 × 25 + 59) div 60 = 21`; the stock holds 15, so 15 are drawn, the stock becomes 0 and the charge is quantity 15 at 30 cents, `(15 × 30 × 100 + 50) div 100 = 450` cents. The shortfall of 6 is not an error.
4. `ply` has `units` `(50 × 6 + 59) div 60 = 5`; 5 are drawn, the stock becomes 95 and the charge is quantity 5 at 12 cents, 60 cents.

The three charges are created in the order usage, `acrylic`, `ply`. Both material charges have discount 0, `kind` `material` and `ref` the reservation identifier.

### Opening hours

**Opening hours** are the same every week:

| Day | Opens | Closes | Open minutes |
| --- | --- | --- | --- |
| Monday to Friday | 09:00 | 21:00 | 720 |
| Saturday | 10:00 | 18:00 | 480 |
| Sunday | closed | closed | 0 |

The opening period of a day is the interval `[opens, closes)` on that date. A reservation is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Sunday has no opening period.

### Reservation rules

A new reservation is accepted only when all of the following hold. These rules connect members, training, equipment and time, and the records of the workshop floor:

1. its slot follows the booking granularity and duration limits and starts at or after the request time;
2. the slot lies inside the opening hours of its start date;
3. the member is in good standing (membership);
4. a member who is a minor on the date of `start` holds consent for the machine's family (guardian consent);
5. the member holds a certification for the machine's family that is valid on the reservation date, which is the date of `start` (certifications);
6. the member's latest signature for the machine's family is of the current waiver version (waiver gate);
7. the machine has no active lockout at the request time (lockout gate);
8. the date of `start` is before the machine's next inspection date, when it has an inspection plan (inspection gate);
9. no scheduled maintenance window of the machine overlaps the slot (maintenance);
10. the slot is covered by scheduled shifts, when the machine's family is supervised (supervision);
11. no other active reservation of the machine overlaps the slot;
12. no other active reservation of the member, on any machine, overlaps the slot.

Rules 4, 6, 7, 8 and 10 bind only where the record they name exists, as their own pages state.

### Reservations

A reservation books one machine for one member for an interval `[start, end)` on one day. Its stored status moves from `reserved` to `checkedIn` to `completed`, or from `reserved` to `cancelled`. Its reported status is the stored status, except that a stored `reserved` reservation whose `end` is at or before the request time is reported as `missed`. A missed reservation can no longer be checked in or cancelled and causes no charge.

A reservation is **active** when its reported status is `reserved` or `checkedIn`. Only active reservations block a machine or a member. Conflicts are judged on the reserved interval `[start, end)`, not on the actual check-in and check-out times.

### Opening hours of day

The opening hours of one day.

**Fields**, in the formats of values:

- `open`: clock time
- `close`: clock time

### Reservation

A reservation of one machine for one member, generated identifier `res-N`. `checkedInAt` and `rateCentsPerMinute` are `null` until check-in; `checkedOutAt`, `meteredMinutes` and `amountCents` are `null` until check-out.

**Fields**, in the formats of values:

- `reservationId`: identifier
- `memberId`: identifier, naming an existing member
- `machineId`: identifier, naming an existing machine
- `start`: timestamp
- `end`: timestamp
- `status`: one of `reserved`, `missed`, `checkedIn`, `completed` or `cancelled`; see reservations for stored and reported status
- `checkedInAt`: timestamp, or `null`
- `checkedOutAt`: timestamp, or `null`
- `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute), or `null`
- `meteredMinutes`: minute count, or `null`
- `amountCents`: cent amount, or `null`

A record is identified by `reservationId`.

### Cancel reservation

The reservation gets status `cancelled`. There is no fee and no credit.

- **Wire name:** `cancelReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `reservationId`: identifier, naming an existing reservation
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the reported status is not `reserved`.
- **Result:**
  - `reservation`: a reservation record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.

### Check in

Status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the machine's current rate. A missed, cancelled, checked-in or completed reservation is refused with `invalid_state`.

Rules of other modules that apply here: guardian consent, waiver gate, PPE gate, lockout gate and premises gate.

- **Wire name:** `checkIn`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `reservationId`: identifier, naming an existing reservation
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the reported status is not `reserved`.
  2. `check_in_too_early` when the request time is before `start`.
  3. `member_suspended` when the member is not in good standing (members interface).
  4. `guardian_consent_missing` when the member is a minor on the date of the request time and holds no consent for the machine's family (guardian consent).
  5. `not_certified` when the member has no certification for the machine's family valid on the date of the request time (training interface).
  6. `waiver_missing` when the waiver gate refuses the member for the machine's family.
  7. `ppe_missing` when the member does not hold every item that the machine's family requires (PPE gate).
  8. `machine_locked_out` when the machine has an active lockout at the request time (lockout gate).
  9. `not_on_premises` when the member has an active badge that is not inside (premises gate).
- **Result:**
  - `reservation`: a reservation record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.

### Check out

Status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. The following then happen together, in this order:

1. Minutes are drawn from the member's valid passes (pass draw). `drawn` is the number drawn, between 0 and `meteredMinutes`, and `billable` is `meteredMinutes − drawn`.
2. A usage charge is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the billable minutes, `unitPriceCents` the captured rate, `discountPercent` the discount of the member's tier at check-out, `at` the request time. A charge of 0 cents is still created, also when the billable minutes are 0. The reservation's `amountCents` is the charge's amount.
3. The consumables of the machine are drawn from stock (consumption). One material charge is created for each consumption row of the machine whose `drawn` is above 0, in ascending `materialId` order, after the usage charge. They use the full metered minutes, whatever the passes drew, and they do not change the reservation's `amountCents`.

Standing is not checked. A lockout of the machine, an unsigned waiver and the other rules of check-in do not stop a check-out. The result is the same as without passes and consumables: only the usage charge is returned.

Rules of other modules that apply here: charges and invoices, pass draw and consumption.

- **Wire name:** `checkOut`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `reservationId`: identifier, naming an existing reservation
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the status is not `checkedIn`.
- **Result:**
  - `reservation`: a reservation record
  - `charge`: a charge record, the usage charge
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.

### Create reservation

Creates the reservation with status `reserved` and the next reservation identifier.

Example: on Saturday `2026-03-07`, the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not; on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`.

Example of the order: a machine that is locked out and also due for inspection refuses a reservation with `machine_locked_out`, because that refusal comes first.

Rules of other modules that apply here: guardian consent, waiver gate, lockout gate, inspection gate and supervision.

- **Wire name:** `createReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `machineId`: identifier, naming an existing machine
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `invalid_slot` when `start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes.
  3. `starts_in_past` when `start` is before the request time.
  4. `outside_opening_hours` when the slot is not inside the opening hours of its start date.
  5. `member_suspended` when the member is not in good standing (members interface).
  6. `guardian_consent_missing` when the member is a minor on the date of `start` and holds no consent for the machine's family (guardian consent).
  7. `not_certified` when the member has no certification for the machine's family valid on the date of `start` (training interface).
  8. `waiver_missing` when the waiver gate refuses the member for the machine's family.
  9. `machine_locked_out` when the machine has an active lockout at the request time, whatever the date of the slot (lockout gate).
  10. `inspection_due` when the machine has an inspection plan and the date of `start` is on or after its `nextDueOn` (inspection gate).
  11. `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot.
  12. `supervisor_required` when the machine's family is supervised and the slot is not wholly inside the union of the scheduled shifts (supervision).
  13. `machine_booked` when another active reservation of the machine overlaps the slot.
  14. `member_double_booked` when another active reservation of the member overlaps the slot.
- **Result:**
  - `reservation`: a reservation record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `reservationId`.

### Get machine schedule

Shows one machine's day: `openingHours` holds that day's opening and closing times, or `null` on Sunday; `reservations` are the machine's reservations whose `start` falls on that date, in every status except `cancelled`; `maintenance` are the machine's scheduled windows that overlap the interval from `00:00` of that date to `00:00` of the next date.

- **Wire name:** `getMachineSchedule`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `date`: date
- **Result:**
  - `machineId`: identifier
  - `date`: date
  - `openingHours`: a opening hours of day record, or `null`
  - `reservations`: a list of reservation records
  - `maintenance`: a list of maintenance window records
- **Permissions:** member all, staff all, admin all.

### List member reservations

Returns all of the member's reservations in every status.

- **Wire name:** `listMemberReservations`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `reservations`: a list of reservation records
- **Permissions:** member own, staff all, admin all.

## Module `safety`

### Safety interface

What other modules may use of safety: its operations and the incident record. Other modules also ask for an incident by its identifier, with its machine, severity and status.

- **Operations:** report incident, resolve incident, list incidents.
- **Uses:** equipment interface.
- **Records:** incident.

### Incidents

Staff record safety incidents against machines. In this delivery an incident is a record only: it does not block reservations or use. Reporting an incident still blocks nothing; `escalateIncident` is what acts on one.

### Incident

A safety incident recorded against a machine, generated identifier `inc-N`. `resolution` and `resolvedAt` are `null` while open.

**Fields**, in the formats of values:

- `incidentId`: identifier
- `machineId`: identifier, naming an existing machine
- `severity`: one of `low`, `medium` or `high`
- `description`: text
- `reportedBy`: identifier
- `reportedAt`: timestamp
- `status`: `open` or `resolved`
- `resolution`: text, or `null`
- `resolvedAt`: timestamp, or `null`

A record is identified by `incidentId`.

### List incidents

Returns all incidents.

- **Wire name:** `listIncidents`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `incidents`: a list of incident records
- **Permissions:** staff all, admin all.

### Report incident

Creates an incident with the next incident identifier, status `open`, `reportedBy` the actor, `reportedAt` the request time, and `resolution` and `resolvedAt` `null`.

- **Wire name:** `reportIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `severity`: one of `low`, `medium` or `high`
  - `description`: text
- **Result:**
  - `incident`: a incident record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `incidentId`.

### Resolve incident

Status `resolved`, `resolution` as supplied, `resolvedAt` the request time.

- **Wire name:** `resolveIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `incidentId`: identifier, naming an existing incident
  - `resolution`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the incident is already resolved.
- **Result:**
  - `incident`: a incident record
- **Permissions:** staff all, admin all.
- **Audit target:** `incidentId`.

## Module `reporting`

### Reporting interface

What other modules may use of reporting: its operations and its report rows.

- **Operations:** utilization report, revenue report.
- **Uses:** equipment interface, bookings interface, billing interface.
- **Records:** utilization row, utilization totals.

### Reports

Reports are read-only and are calculated from the current state at the request time.

### Utilization row

One machine's row of the utilization report.

**Fields**, in the formats of values:

- `machineId`: identifier, naming an existing machine
- `openMinutes`: minute count
- `maintenanceMinutes`: minute count
- `availableMinutes`: minute count
- `usedMinutes`: minute count
- `utilizationPercent`: integer of at least `0` (a utilization in percent; it can exceed 100)

### Utilization totals

The totals of the utilization report.

**Fields**, in the formats of values:

- `openMinutes`: minute count
- `maintenanceMinutes`: minute count
- `availableMinutes`: minute count
- `usedMinutes`: minute count
- `utilizationPercent`: integer of at least `0` (a utilization in percent; it can exceed 100)

### Revenue report

Summarizes billing for one month:

- `invoicesGenerated`: whether invoices for the month have been generated;
- `invoiceCount`: the number of invoices of the month;
- `membershipCents`, `usageCents`, `materialCents`, `courseCents`: the sums of the line amounts of each kind on the invoices of the month;
- `invoicedCents`: the sum of those four;
- `creditAppliedCents`: the sum of `creditAppliedCents` over the invoices of the month;
- `outstandingCents`: the sum of `balanceCents` over the invoices of the month at the request time;
- `paymentsCents`: the sum of `amountCents` of all payments whose `at` lies in the month, for any invoice, including the parts that became credit;
- `creditsIssuedCents`: the sum of credits issued with `issueCredit` whose request time lies in the month. Credit from overpayments is not included.

For a month without generated invoices, the invoice figures are 0 and `invoicesGenerated` is `false`; payments and issued credits are still reported.

The four kind sums and `invoicedCents` cover only the kinds `membership`, `usage`, `material` and `course`; a line of any other kind is in none of them, and `chargesByKind` covers every kind. `creditAppliedCents`, `outstandingCents` and `paymentsCents` take the invoices and payments as they are, so lines of other kinds count in them.

Rules of other modules that apply here: charges and invoices and overdue invoices.

- **Wire name:** `revenueReport`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Result:**
  - `month`: month
  - `invoicesGenerated`: flag
  - `invoiceCount`: count
  - `membershipCents`: cent amount
  - `usageCents`: cent amount
  - `materialCents`: cent amount
  - `courseCents`: cent amount
  - `invoicedCents`: cent amount
  - `creditAppliedCents`: cent amount
  - `outstandingCents`: cent amount
  - `paymentsCents`: cent amount
  - `creditsIssuedCents`: cent amount
- **Permissions:** staff all, admin all.

### Utilization report

Shows how much each machine was used compared with the time it was available. The report range is the interval from `00:00` of `from` to `00:00` of `to`; `to` itself is excluded. `machines` has one row per machine that exists at the request time.

- `openMinutes`: minutes of the range that lie inside opening hours. Every machine gets the full count, regardless of when it was added.
- `maintenanceMinutes`: minutes of the range that lie inside opening hours and inside at least one scheduled maintenance window of the machine. Overlapping windows count once.
- `availableMinutes`: `openMinutes − maintenanceMinutes`.
- `usedMinutes`: for every completed reservation of the machine, the minutes of `[checkedInAt, checkedOutAt)` that lie in the range, summed. Opening hours do not limit this count. Reservations that are checked in but not checked out are not counted.
- `utilizationPercent`: `(usedMinutes × 100 × 2 + availableMinutes) div (availableMinutes × 2)`, which is `usedMinutes × 100 / availableMinutes` rounded half up to an integer, or 0 when `availableMinutes` is 0. It can exceed 100.
- `totals`: the sums of `openMinutes`, `maintenanceMinutes`, `availableMinutes` and `usedMinutes` over all rows, and `utilizationPercent` computed from these sums with the same formula. With no machines, all totals are 0.

Rules of other modules that apply here: reservations and maintenance.

- **Wire name:** `utilizationReport`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `from`: date
  - `to`: date
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `to` is not after `from` or the range is longer than 366 days.
- **Result:**
  - `from`: date
  - `to`: date
  - `machines`: a list of utilization row records
  - `totals`: a utilization totals record
- **Permissions:** staff all, admin all.

## Module `access`

### Access interface

What other modules may use of access: its operations, the badge and door event records and the refusal codes below. Other modules also ask whether a member has an active badge that is not inside. The rule is premises gate. Access in turn needs the opening period of a date from bookings, the account of a member from billing and the guest pass records of guests. Bookings, which access uses, declares its need for that answer, and the protocol module supplies it from access.

- **Operations:** issue badge, revoke badge, enter door, leave door, admit guest, list door events.
- **Uses:** members interface, billing interface, guests interface, bookings interface.
- **Records:** badge, door event, door event kind.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `account_overdue` | The member has an overdue balance above 0. |
| `already_inside` | The badge is already inside. |
| `badge_active` | The member has an active badge. |
| `guest_already_admitted` | The guest pass has been used to admit its guest already. |
| `not_inside` | The badge is not inside. |
| `not_on_premises` | The member has an active badge that is not inside. |
| `pass_not_today` | The guest pass is for another date than the date of the request time. |
| `pass_of_another_member` | The guest pass was issued to another member than the badge's. |
| `sponsor_not_inside` | The badge of the member who sponsors the guest is not inside. |

### Premises gate

A member **is on the premises** when the member's active badge is inside. A member with no active badge is not gated: the gate never applies to such a member.

**The gate.** At check in, a member who has an active badge that is not inside is refused `not_on_premises`, at the place that operation lists. Check-out is never gated.

### Badge

A member's door badge, generated identifier `bdg-N`. A member has at most one active badge. A badge is **inside** from enter door until leave door or revoke badge; it starts outside. A revoked badge is never inside and never active again.

**Fields**, in the formats of values:

- `badgeId`: identifier
- `memberId`: identifier, naming an existing member
- `status`: `active` or `revoked`
- `inside`: flag

A record is identified by `badgeId`.

### Door event

One passage at the door, generated identifier `dor-N`. `guestPassId` is `null` unless the kind is `guest`.

**Fields**, in the formats of values:

- `eventId`: identifier
- `kind`: a door event kind: `enter`, `leave` or `guest`
- `badgeId`: identifier, naming an existing badge
- `guestPassId`: identifier, naming an existing guest pass, or `null`
- `at`: timestamp

A record is identified by `eventId`.

### Door event kind

The kind of a door event.

| Value | Meaning |
| --- | --- |
| `enter` | The badge's member came in. |
| `leave` | The badge's member went out. |
| `guest` | A guest of the badge's member was admitted with a guest pass. |

### Admit guest

Admits the guest of the pass. A door event is created with the next door event identifier: `kind` `guest`, `badgeId` the badge, `guestPassId` the pass, `at` the request time. The badge does not change. A pass admits its guest once. Ownership is judged on `badgeId` alone: the badge's member is the actor. Standing, opening hours and the account are not checked.

- **Wire name:** `admitGuest`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `guestPassId`: identifier, naming an existing guest pass
  - `badgeId`: identifier, naming an existing badge
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the guest pass is revoked or the badge is revoked.
  2. `pass_not_today` when the pass's `date` is not the date of the request time.
  3. `pass_of_another_member` when the pass's member is not the badge's member.
  4. `sponsor_not_inside` when the badge is not inside.
  5. `guest_already_admitted` when a door event of kind `guest` exists for this pass.
- **Result:**
  - `event`: a door event record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `guestPassId`.

### Enter door

The badge is inside. A door event is created with the next door event identifier: `kind` `enter`, `badgeId` the badge, `guestPassId` `null`, `at` the request time. The `badge` result is the badge after the change.

The request time is in the opening period of its date as opening hours states: the opening time is in, the closing time is out, and on a Sunday no time is in. The member's overdue balance is the `overdueBalanceCents` of the member's account at the request time (get account).

Rules of other modules that apply here: membership, opening hours and overdue invoices.

- **Wire name:** `enterDoor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `badgeId`: identifier, naming an existing badge
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is revoked.
  2. `already_inside` when the badge is inside.
  3. `member_suspended` when the badge's member is not in good standing (members interface).
  4. `outside_opening_hours` when the request time is not in the opening period of its date.
  5. `account_overdue` when the member's overdue balance is above 0.
- **Result:**
  - `badge`: a badge record
  - `event`: a door event record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `badgeId`.

### Issue badge

Creates a badge for the member with the next badge identifier, status `active` and `inside` false. Standing is not checked.

- **Wire name:** `issueBadge`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `badge_active` when the member has an active badge.
- **Result:**
  - `badge`: a badge record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `badgeId`.

### Leave door

The badge is no longer inside. A door event is created with the next door event identifier: `kind` `leave`, `badgeId` the badge, `guestPassId` `null`, `at` the request time. Standing, opening hours and the account are not checked. The `badge` result is the badge after the change.

- **Wire name:** `leaveDoor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `badgeId`: identifier, naming an existing badge
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is revoked.
  2. `not_inside` when the badge is not inside.
- **Result:**
  - `badge`: a badge record
  - `event`: a door event record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `badgeId`.

### List door events

Returns the door events whose `at` lies on the date, in creation order.

- **Wire name:** `listDoorEvents`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `date`: date
- **Result:**
  - `date`: date
  - `events`: a list of door event records
- **Permissions:** staff all, admin all.

### Revoke badge

The badge gets status `revoked` and `inside` false. No door event is created. The member may be issued a new badge afterwards.

- **Wire name:** `revokeBadge`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `badgeId`: identifier, naming an existing badge
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is already revoked.
- **Result:**
  - `badge`: a badge record
- **Permissions:** staff all, admin all.
- **Audit target:** `badgeId`.

## Module `instructors`

### Instructors interface

What other modules may use of instructors: its operations and the qualification record. Other modules also ask whether a user holds a qualification for a family and whether a user holds a qualification for any family. Instructors in turn needs to know whether a user is the instructor of a scheduled class of a family that starts after a given time; it declares that need, and the protocol module supplies the answer from classes.

- **Operations:** qualify instructor, withdraw instructor, list instructors.
- **Uses:** members interface, equipment interface, training interface.
- **Records:** qualification.

What other modules ask of instructors that is not an operation:

- Whether a user holds a qualification for a given family: yes when a qualification with that `userId` and `familyId` exists.
- Whether a user holds a qualification for any family: yes when a qualification with that `userId` exists.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `instructor_has_classes` | The user is the instructor of a scheduled class of the family that starts after the request time. |
| `not_eligible` | The user may not be qualified for the family: an admin, or a member with no certification valid on the request date. |
| `not_instructor` | The user holds no qualification for the family, or, where an operation says so, for any family. |

### Qualifications

A user is an **instructor** for a family while the user holds a qualification for it. A user holds at most one qualification per family and may hold several.

Who may be qualified, judged on the date of the request time when the qualification is made:

- A staff user may be qualified for any family.
- A member may be qualified for a family only when the member holds a certification for it that is valid on that date (certifications).
- An admin is never qualified.

The check is made once. A later expiry or revocation of the member's certification does not end the qualification. A qualification ends only when it is withdrawn, and the user may then be qualified for the family again.

### Qualification

The qualification of one user to teach classes for one tool family.

**Fields**, in the formats of values:

- `userId`: identifier, naming an existing user
- `familyId`: identifier, naming an existing family

A record is identified by `userId` and `familyId` together.

### List instructors

Returns the qualifications for the family, in ascending `userId` order.

- **Wire name:** `listInstructors`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
- **Result:**
  - `familyId`: identifier
  - `instructors`: a list of qualification records
- **Permissions:** member all, staff all, admin all.

### Qualify instructor

Creates the qualification of the user for the family (qualifications).

- **Wire name:** `qualifyInstructor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier, naming an existing user
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `not_eligible` when the user is an admin, or is a member who holds no certification for the family valid on the date of the request time (training interface).
  2. `already_exists` when the user holds a qualification for the family.
- **Result:**
  - `qualification`: a qualification record
- **Permissions:** admin all.
- **Audit target:** `userId/familyId`.

### Withdraw instructor

Removes the qualification of the user for the family. The `qualification` result is the record that was removed. Classes already held in the past stay as they are. A class that starts at the request time, or earlier, does not prevent the withdrawal.

- **Wire name:** `withdrawInstructor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier, naming an existing user
  - `familyId`: identifier, naming an existing family
- **Refusals** at check 7, in this order:
  1. `not_instructor` when the user holds no qualification for the family.
  2. `instructor_has_classes` when the user is the instructor of a scheduled class whose course belongs to the family and whose `start` is after the request time (classes interface).
- **Result:**
  - `qualification`: a qualification record
- **Permissions:** admin all.
- **Audit target:** `userId/familyId`.

## Module `classes`

### Classes interface

What other modules may use of classes: its operations and the class and attendance records. Other modules also ask whether a member still owes attendance for a course, for the scheduled classes of an instructor that start in a month, and whether an instructor has a scheduled class of a family that starts after a given time.

- **Operations:** schedule class, cancel class, mark attendance, list course classes, list class attendance.
- **Uses:** members interface, training interface, instructors interface, rooms interface.
- **Records:** class, attendance.

What other modules ask of classes that is not an operation:

- Whether a member still owes attendance for a course, given the course, the member, the `enrolledAt` of the member's enrollment and the request time: yes when attendance gate finds a class that requires the member's attendance and the member did not attend it.
- The scheduled classes of a user as instructor whose `start` lies in a given month, each with its length in minutes, which is `end − start`.
- Whether a user is the instructor of a scheduled class whose course belongs to a given family and whose `start` is after a given time.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `attendance_missing` | The member did not attend a class of the course that requires the member's attendance. |
| `class_not_started` | The class has not started yet at the request time. |
| `instructor_busy` | Another scheduled class of the instructor overlaps the class. |

### Attendance gate

At complete course a member must have attended the classes of the course that began after the member enrolled.

Let `enrolledAt` be that of the member's enrollment in the course, which has status `enrolled` at that moment. A class **requires** the member's attendance when all of these hold:

- it is a class of the course;
- its status is `scheduled`;
- its `start` is at or before the request time;
- its `start` is at or after `enrolledAt`.

The member **attended** a class when an attendance mark of the member for the class has `present` true. No mark, or a mark with `present` false, is not attendance. A class that has started but not ended already requires attendance.

The completion is refused with `attendance_missing` when some class requires the member's attendance and the member did not attend it. A course with no class that requires attendance passes the gate.

Example: the member enrolled at `2026-03-02T10:00Z`. A class that started on `2026-03-01` does not count, whatever its marks. A class that started at `2026-03-02T10:00Z` counts. A cancelled class does not count. A class that starts after the request time does not count. Of the classes that count, every one needs a mark with `present` true.

### Class schedule

A class is `scheduled` until it is cancelled, and a cancelled class stays cancelled. A class has no capacity of its own and no opening hours, and it is not moved: to change it, cancel it and schedule another.

A scheduled class keeps its room: scheduling it makes a hold of the room for `[start, end)` with origin `class` and purpose `class ` followed by the class identifier, and cancelling it releases that hold (room holds). A cancelled class holds nothing.

An instructor teaches at most one scheduled class at a time: the scheduled classes of one instructor never overlap (time). Classes of different instructors may overlap, in different rooms.

### Attendance

The mark of one member for one class: whether the member was present.

**Fields**, in the formats of values:

- `classId`: identifier, naming an existing class
- `memberId`: identifier, naming an existing member
- `present`: flag

A record is identified by `classId` and `memberId` together.

### Class

One session of a course, taught by one instructor in one room for an interval, generated identifier `cls-N`.

**Fields**, in the formats of values:

- `classId`: identifier
- `courseId`: identifier, naming an existing course
- `instructorId`: identifier, naming an existing user
- `roomId`: identifier, naming an existing room
- `start`: timestamp
- `end`: timestamp
- `status`: `scheduled` or `cancelled`

A record is identified by `classId`.

### Cancel class

The class gets status `cancelled`. Its room hold is released, so the room is free again for the class's interval. Attendance marks of the class stay, but a cancelled class requires no attendance (attendance gate).

- **Wire name:** `cancelClass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `classId`: identifier, naming an existing class
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the class is already cancelled or its `start` is at or before the request time.
- **Result:**
  - `class`: a class record
- **Permissions:** staff all, admin all.
- **Audit target:** `classId`.

### List class attendance

Returns the attendance marks of the class, one for each member who has a mark, in ascending `memberId` order. A class with no marks returns an empty list.

- **Wire name:** `listClassAttendance`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `classId`: identifier, naming an existing class
- **Result:**
  - `classId`: identifier
  - `attendance`: a list of attendance records
- **Permissions:** staff all, admin all.

### List course classes

Returns all classes of the course in every status, with `start` ascending, then in creation order.

- **Wire name:** `listCourseClasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
- **Result:**
  - `courseId`: identifier
  - `classes`: a list of class records
- **Permissions:** member all, staff all, admin all.

### Mark attendance

Records whether the member was present at the class. A member has one mark per class: a later mark of the same member for the same class replaces the earlier one, including its `present` value.

- **Wire name:** `markAttendance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `classId`: identifier, naming an existing class
  - `memberId`: identifier, naming an existing member
  - `present`: flag
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the class is cancelled.
  2. `class_not_started` when the request time is before the `start` of the class. A request at the `start` is accepted.
  3. `not_enrolled` when the member has no enrollment in the course of the class, whatever its status (training interface).
- **Result:**
  - `attendance`: a attendance record
- **Permissions:** staff all, admin all.
- **Audit target:** `classId/memberId`.

### Schedule class

Creates the class with status `scheduled` and the next class identifier. It makes a hold of the room for `[start, end)` as class schedule states. The `class` result is the new class.

Example: the instructor `kim` teaches a class from `10:00` to `12:00`. A class of `kim` from `12:00` to `13:00` is accepted; a class of `kim` from `11:30` to `13:00` in another room is refused with `instructor_busy`. If the second class is in the room of the first, the refusal is `room_taken`.

Rules of other modules that apply here: room holds.

- **Wire name:** `scheduleClass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `instructorId`: identifier, naming an existing user
  - `roomId`: identifier, naming an existing room
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `starts_in_past` when `start` is before the request time.
  3. `not_instructor` when the instructor holds no qualification for the family of the course (instructors interface).
  4. `room_taken` when the room is not free for `[start, end)` (rooms interface).
  5. `instructor_busy` when another scheduled class of the instructor overlaps `[start, end)`.
- **Result:**
  - `class`: a class record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `classId`.

## Module `assessments`

### Assessments interface

What other modules may use of assessments: its operations and the assessment and attempt records. Other modules also ask whether a member has met the assessment requirement of a course.

- **Operations:** define assessment, record attempt, list attempts.
- **Uses:** members interface, training interface.
- **Records:** assessment, attempt.

What other modules ask of assessments that is not an operation:

- Whether a member has met the assessment requirement of a course, given the course and the member: yes when assessment gate passes. Assessments learns the member's current enrollment period from training.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `assessment_not_passed` | The course has an assessment and the member has not passed it in the current enrollment period. |
| `attempts_exhausted` | The member has used all attempts of the current enrollment period. |
| `no_assessment` | The course has no assessment. |

### Assessment gate

At complete course a member must have passed the assessment of the course, if it has one, in the current enrollment period.

The completion is refused with `assessment_not_passed` when the course has an assessment and none of the member's attempts in the current enrollment period has `passed` true (attempts). Attempts of earlier periods do not count. A course without an assessment passes the gate.

Example: the pass mark is 70. The member enrolled, scored 65 and then 70 in the first period and completed the course. After a new enrollment the first period's attempts count for nothing: the completion is refused until an attempt of the new period, numbered 1, has a score of 70 or more.

### Attempts

An **enrollment period** of a member in a course starts with an enrollment and ends with its completion. The first enrollment of the member in the course starts period 1. Every later enrollment, which can only follow a completion, starts the next period, even when it is made at the same time as that completion. The number of a period is the number of the member's enrollments in the course so far; training answers it (training interface).

An attempt belongs to the period in which it was recorded. Its `number` is the count of the attempts of that period so far plus one, so the numbers of a period count 1, 2, 3, and a new period starts again at 1. The attempts of earlier periods do not count in a later one.

An attempt is `passed` when its `score` is at least the `passMark` of the course's assessment. A `passMark` of 0 is passed by every score. Attempts are never changed or removed.

### Assessment

The test a member must pass to complete a course: the lowest passing score and the number of attempts allowed in one enrollment period. A course has at most one assessment, and it is never changed or removed.

**Fields**, in the formats of values:

- `courseId`: identifier, naming an existing course
- `passMark`: integer from `0` to `100` (the lowest score that passes)
- `maxAttempts`: integer from `1` to `10` (the number of attempts allowed in one enrollment period)

A record is identified by `courseId`.

### Attempt

One attempt of a member at the assessment of a course. An attempt has no identifier of its own.

**Fields**, in the formats of values:

- `courseId`: identifier, naming an existing course
- `memberId`: identifier, naming an existing member
- `number`: count, 1 for the first attempt of an enrollment period (attempts)
- `score`: integer from `0` to `100` (the score of the attempt)
- `passed`: flag
- `at`: timestamp

### Define assessment

Creates the assessment of the course. It applies at once to members who are already enrolled: their completion needs a passed attempt, and no attempt was possible before this.

- **Wire name:** `defineAssessment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `passMark`: integer from `0` to `100` (the lowest score that passes)
  - `maxAttempts`: integer from `1` to `10` (the number of attempts allowed in one enrollment period)
- **Refusals** at check 7, in this order:
  1. `already_exists` when the course has an assessment.
- **Result:**
  - `assessment`: a assessment record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.

### List attempts

Returns all attempts of the member for the course, from every enrollment period, in the order they were recorded. A member with no attempts gets an empty list.

- **Wire name:** `listAttempts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `courseId`: identifier
  - `memberId`: identifier
  - `attempts`: a list of attempt records
- **Permissions:** member own, staff all, admin all.

### Record attempt

Records an attempt of the member in the current enrollment period (attempts): `number` is one more than the attempts of that period so far, `score` is as supplied, `passed` is true when `score` is at least the `passMark` of the course's assessment, and `at` is the request time.

Example: with `passMark` 70 and `maxAttempts` 2, a score of 69 is not passed and a score of 70 is passed. After two attempts a third is refused with `attempts_exhausted`, even when one of them passed. After a first attempt that passed, a second is still accepted.

- **Wire name:** `recordAttempt`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
  - `score`: integer from `0` to `100` (the score of the attempt)
- **Refusals** at check 7, in this order:
  1. `no_assessment` when the course has no assessment.
  2. `not_enrolled` when the member has no enrollment in the course with status `enrolled` (training interface).
  3. `attempts_exhausted` when the attempts of the member's current enrollment period equal the `maxAttempts` of the assessment.
- **Result:**
  - `attempt`: a attempt record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId/memberId`.

## Module `waitlists`

### Waitlists interface

What other modules may use of waitlists: its operations and the waiting entry and skipped entry records. No other module asks anything else of waitlists.

- **Operations:** join waitlist, leave waitlist, list waitlist, promote waitlist.
- **Uses:** members interface, training interface, notices interface.
- **Records:** waiting entry, skipped entry.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_waiting` | The member is already on the waiting list of the course. |
| `course_not_full` | The course has a free place, so there is nothing to wait for. |
| `not_waiting` | The member is not on the waiting list of the course. |

### Waitlist promotion

A promotion fills the free places of a course from its waiting list.

It goes through the entries of the course in join order. Before it tries an entry it checks that the course has a free place, which is its `capacity` less its `enrolledCount` (training interface); when it has none, the promotion stops and the entries that were not tried stay as they are. Each entry is tried at most once in a promotion.

To try an entry, the member is enrolled in the course exactly as enroll in course does at the request time of the promotion: the same refusals in the same order, and when none applies the same enrollment, charge and discounts.

- When the enrollment succeeds, the entry leaves the list, the enrollment is added to `enrollments`, and a notice of kind `waitlist` is posted for the member with `ref` the course identifier (notices interface, notice kind).
- When the enrollment is refused, nothing changes: the entry keeps its place in the list. It is added to `skipped` with the refusal code, and the next entry is tried. The codes that can occur are `member_suspended`, `guardian_consent_missing` and `already_enrolled`.

`enrollments` and `skipped` are in join order.

Example: a course has capacity 2 and one enrolled member. The list is `ann` (suspended), `bob`, `cy`. A promotion skips `ann` with `member_suspended`, enrolls `bob`, and stops because the course is full. `cy` is not tried and stays on the list. The result has `enrollments` with the enrollment of `bob` and `skipped` with `ann` and `member_suspended`.

### Waitlists

A course that has no free place can have a waiting list. The list holds at most one waiting entry per member, in the order in which the members joined. Joining puts an entry at the end. A member who leaves and joins again is at the end. An entry leaves the list only when the member leaves it or a promotion enrolls the member (waitlist promotion).

A waiting list is not tied to the course being full: places that become free stay free until a promotion fills them, and a member can enroll directly in a free place. Enrolling directly does not remove the member's entry; a promotion then skips it.

### Skipped entry

A waiting entry that a promotion could not enroll, with the reason.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `code`: text, the refusal code of the enrollment: `member_suspended`, `guardian_consent_missing` or `already_enrolled`

### Waiting entry

The place of one member on the waiting list of one course.

**Fields**, in the formats of values:

- `courseId`: identifier, naming an existing course
- `memberId`: identifier, naming an existing member
- `joinedAt`: timestamp

A record is identified by `courseId` and `memberId` together.

### Join waitlist

Adds the member to the end of the course's waiting list (waitlists). The entry gets `joinedAt` equal to the request time. The `entry` result is the new entry. A member who completed the course may join.

- **Wire name:** `joinWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `already_enrolled` when the member's enrollment in the course has status `enrolled` (training interface).
  3. `course_not_full` when the course has a free place.
  4. `already_waiting` when the member is on the waiting list of the course.
- **Result:**
  - `entry`: a waiting entry record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.

### Leave waitlist

Removes the member's entry from the course's waiting list. The `entry` result is the entry that was removed. Standing is not checked.

- **Wire name:** `leaveWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `not_waiting` when the member is not on the waiting list of the course.
- **Result:**
  - `entry`: a waiting entry record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.

### List waitlist

Returns the entries of the course's waiting list in join order. A course with no entries returns an empty list.

- **Wire name:** `listWaitlist`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
- **Result:**
  - `courseId`: identifier
  - `entries`: a list of waiting entry records
- **Permissions:** staff all, admin all.

### Promote waitlist

Fills the free places of the course from its waiting list as waitlist promotion states. `enrollments` are the enrollments made, in join order; `skipped` are the entries that were refused, in join order. Both are empty lists when nothing happened. A course with no free place, or with no entries, is not an error.

Rules of other modules that apply here: enroll in course and, for the notice, notices interface.

- **Wire name:** `promoteWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `courseId`: identifier, naming an existing course
- **Result:**
  - `courseId`: identifier
  - `enrollments`: a list of enrollment records
  - `skipped`: a list of skipped entry records
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.

## Module `payouts`

### Payouts interface

What other modules may use of payouts: its operations and the instructor rate and payout records. No other module asks anything else of payouts.

- **Operations:** set instructor rate, generate payouts, settle payout, list payouts.
- **Uses:** members interface, ledger interface, instructors interface, classes interface.
- **Records:** instructor rate, payout.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_generated` | Payouts for the month have already been generated. |

generate payouts also uses the code `month_not_ended` that invoicing uses. Here it means that the month has not ended at the request time.

### Payout calculation

The **teaching minutes** of a user in a month are the sum of the lengths in minutes, `end − start`, of the user's scheduled classes whose `start` lies in the month (classes interface). A class counts completely in the month of its `start`, even when it ends in the next month. Cancelled classes do not count. Only classes whose `instructorId` is the user count.

The amount of a payout is `(minutes × centsPerHour + 30) div 60`, where `centsPerHour` is the user's instructor rate at the request time and `div` is integer division. This is the exact value rounded half up to a whole cent, once for the whole month (money). The amount can be 0.

Example: 90 and 45 minutes at 4500 cents per hour give 135 minutes and `(135 × 4500 + 30) div 60 = 10125` cents. One minute at 30 cents per hour gives `(30 + 30) div 60 = 1` cent; at 29 cents per hour it gives 0 cents. A class from `2026-03-31T23:30Z` to `2026-04-01T01:00Z` counts 90 minutes in `2026-03` and nothing in `2026-04`.

### Instructor rate

The pay of one instructor per hour of teaching.

**Fields**, in the formats of values:

- `userId`: identifier, naming an existing user
- `centsPerHour`: integer from `1` to `1000000` cents per hour (an instructor's pay for one hour of teaching)

A record is identified by `userId`.

### Payout

The pay owed to one instructor for the classes of one month, generated identifier `pyt-N`.

**Fields**, in the formats of values:

- `payoutId`: identifier
- `userId`: identifier, naming an existing user
- `month`: month
- `minutes`: minute count
- `amountCents`: cent amount
- `status`: `open` or `settled`

A record is identified by `payoutId`.

### Generate payouts

Creates the payouts of the month and marks the month as generated, even when no payout results. For every user who has a rate, in ascending `userId` order, and has at least one scheduled class whose `start` lies in the month, a payout is created with the next payout identifier, status `open`, and `minutes` and `amountCents` as payout calculation states. A user without a rate, and a user with no such class, gets no payout. A payout of 0 cents is still created. For each payout a posting is made as postings states.

Rules of other modules that apply here: postings.

- **Wire name:** `generatePayouts`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`). The month `2099-12` never ends at a request time.
  2. `already_generated` when payouts for this month were generated before.
- **Result:**
  - `month`: month
  - `payouts`: a list of payout records
- **Permissions:** admin all.
- **Audit target:** `month`.

### List payouts

Returns the payouts of the user in creation order. A user with no payouts gets an empty list.

- **Wire name:** `listPayouts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier, naming an existing user
- **Result:**
  - `userId`: identifier
  - `payouts`: a list of payout records
- **Permissions:** staff all, admin all.

### Set instructor rate

Sets the user's rate and replaces an earlier one. Payouts that are already generated do not change. The rate stays when the user's qualifications are withdrawn later.

- **Wire name:** `setInstructorRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `userId`: identifier, naming an existing user
  - `centsPerHour`: integer from `1` to `1000000` cents per hour (an instructor's pay for one hour of teaching)
- **Refusals** at check 7, in this order:
  1. `not_instructor` when the user holds no qualification for any family (instructors interface).
- **Result:**
  - `rate`: a instructor rate record
- **Permissions:** admin all.
- **Audit target:** `userId`.

### Settle payout

The payout gets status `settled`. A posting is made as postings states. A payout of 0 cents can be settled; it posts nothing.

Rules of other modules that apply here: postings.

- **Wire name:** `settlePayout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `payoutId`: identifier, naming an existing payout
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the payout is already settled.
- **Result:**
  - `payout`: a payout record
- **Permissions:** admin all.
- **Audit target:** `payoutId`.

## Module `events`

### Events interface

What other modules may use of events: its operations and the event and ticket records. No other module asks anything else of events.

- **Operations:** create event, sell ticket, cancel ticket, cancel event, list event tickets.
- **Uses:** members interface, rooms interface, billing interface, ledger interface.
- **Records:** event, ticket.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_ticketed` | The member holds a valid ticket for the event. |
| `capacity_too_large` | The event's capacity is above the capacity of its room. |
| `event_closed` | The event is cancelled, or it starts at or before the request time. |
| `event_sold_out` | All places of the event are taken by valid tickets. |
| `too_late` | The event starts in less than 1440 minutes. |

### Ticket sales

An event sells up to `capacity` tickets at `priceCents` each. Its `sold` is the number of its valid tickets. A sale raises it by one and the cancellation of a ticket lowers it by one, so a cancelled ticket gives its place back. The event is **sold out** when `sold` equals `capacity`. A member holds at most one valid ticket per event and may buy another one after cancelling.

An event is **closed** when it is cancelled or its `start` is at or before the request time. Tickets are sold only for an event that is not closed. A cancelled event stays cancelled; its `sold` is 0.

A ticket's charge is created when it is sold and is never changed. When a ticket is cancelled, by cancel ticket or together with its event by cancel event, the member's credit balance increases by the event's `priceCents`, through the add-credit capability of billing interface, and the posting that postings gives for the event is made. The credit does not depend on whether the charge is invoiced or paid, and a price of 0 gives no credit and no posting.

An event keeps its room: creating it makes a hold of the room for `[start, end)` with origin `event` and purpose `event ` followed by the event identifier, and cancelling it releases that hold (room holds). The capacity of an event is judged against the capacity of its room once, when the event is created.

### Event

A one-off event in a room with a number of tickets and a price, such as an open day.

**Fields**, in the formats of values:

- `eventId`: identifier
- `title`: text
- `roomId`: identifier, naming an existing room
- `start`: timestamp
- `end`: timestamp
- `capacity`: integer from `1` to `1000` (the number of tickets of an event)
- `priceCents`: integer from `0` to `1000000` cents (the price of one ticket)
- `status`: `open` or `cancelled`
- `sold`: count, the number of valid tickets of the event (ticket sales)

A record is identified by `eventId`.

### Ticket

The ticket of one member for one event, generated identifier `tkt-N`.

**Fields**, in the formats of values:

- `ticketId`: identifier
- `eventId`: identifier, naming an existing event
- `memberId`: identifier, naming an existing member
- `status`: `valid` or `cancelled`

A record is identified by `ticketId`.

### Cancel event

The event gets status `cancelled` and its room hold is released. Every valid ticket of the event, in creation order, gets status `cancelled`, with the credit and the posting that ticket sales states for each. `sold` becomes 0. The 1440-minute limit of cancel ticket does not apply. `tickets` are the tickets cancelled by this request, in creation order; the list is empty when the event had no valid ticket.

- **Wire name:** `cancelEvent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `eventId`: identifier, naming an existing event
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the event is already cancelled or its `start` is at or before the request time.
- **Result:**
  - `event`: an event record
  - `tickets`: a list of ticket records
- **Permissions:** staff all, admin all.
- **Audit target:** `eventId`.

### Cancel ticket

The ticket gets status `cancelled` and the event's `sold` decreases by one. The member's credit balance increases by the event's `priceCents` and a posting is made, as ticket sales states. The ticket's charge does not change. The `event` result is the event after the change.

A ticket can be cancelled until 1440 minutes before the event's start. Example: for an event that starts at `2026-03-10T18:00Z`, a request at `2026-03-09T18:00Z` is accepted and a request at `2026-03-09T18:01Z` is refused with `too_late`.

- **Wire name:** `cancelTicket`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `ticketId`: identifier, naming an existing ticket
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the ticket is already cancelled.
  2. `too_late` when the `start` of the ticket's event is less than 1440 minutes after the request time, which includes an event that has started.
- **Result:**
  - `ticket`: an ticket record
  - `event`: an event record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `ticketId`.

### Create event

Creates the event with status `open`, `sold` 0 and the other fields as supplied. It makes a hold of the room for `[start, end)` as ticket sales states. The `event` result is the new event.

Rules of other modules that apply here: room holds.

- **Wire name:** `createEvent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `eventId`: identifier
  - `title`: text
  - `roomId`: identifier, naming an existing room
  - `start`: timestamp
  - `end`: timestamp
  - `capacity`: integer from `1` to `1000` (the number of tickets of an event)
  - `priceCents`: integer from `0` to `1000000` cents (the price of one ticket)
- **Refusals** at check 7, in this order:
  1. `already_exists` when an event with `eventId` exists.
  2. `invalid_interval` when `end` is not after `start`.
  3. `starts_in_past` when `start` is before the request time.
  4. `capacity_too_large` when `capacity` is above the `capacity` of the room. A capacity equal to the room's is accepted.
  5. `room_taken` when the room is not free for `[start, end)` (rooms interface).
- **Result:**
  - `event`: an event record
- **Permissions:** staff all, admin all.
- **Audit target:** `eventId`.

### List event tickets

Returns all tickets of the event in every status, in creation order. An event with no tickets returns an empty list.

- **Wire name:** `listEventTickets`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `eventId`: identifier, naming an existing event
- **Result:**
  - `eventId`: identifier
  - `tickets`: a list of ticket records
- **Permissions:** staff all, admin all.

### Sell ticket

Sells a ticket to the member: a ticket is created with status `valid` and the next ticket identifier, and the event's `sold` increases by one. An event charge is created for the member: `kind` `event`, `ref` the ticket identifier, `quantity` 1, `unitPriceCents` the event's `priceCents`, `discountPercent` 0, `at` the request time. An event with price 0 creates a charge of 0 cents. The `event` result is the event after the change.

Rules of other modules that apply here: charges and invoices and membership.

- **Wire name:** `sellTicket`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `eventId`: identifier, naming an existing event
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing (members interface).
  2. `event_closed` when the event is cancelled or its `start` is at or before the request time.
  3. `already_ticketed` when the member holds a valid ticket for the event.
  4. `event_sold_out` when `sold` equals `capacity`.
- **Result:**
  - `ticket`: a ticket record
  - `event`: an event record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `ticketId`.

## Module `hazmat`

### Hazmat interface

What other modules may use of hazmat: its operations, the classification record and the refusal codes below. Other modules also ask hazmat, for a member, a material, a quantity and the request time, which refusal of the hazard gate applies to a purchase: `hazard_not_certified`, `hazard_limit_exceeded` or none. materials interface declares this need, and the protocol module supplies the answer.

- **Operations:** classify hazard, declassify hazard, list hazards.
- **Uses:** equipment interface, materials interface, training interface.
- **Records:** classification.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `hazard_limit_exceeded` | The member would have bought more units of the hazardous material in the month than its monthly limit allows. |
| `hazard_not_certified` | The member has no certification valid on the date of the request time for the family of the hazardous material. |
| `not_hazardous` | The material has no classification. |

### Hazard gate

A purchase of a material that has a classification passes the hazard gate before the purchase is made. A material without a classification is not gated. The gate has two refusals, judged in this order for the member who buys:

1. `hazard_not_certified` when the member has no certification for the classification's family that is valid on the date of the request time (certifications).
2. `hazard_limit_exceeded` when the units of the material the member bought in the month of the request time, plus the quantity of this purchase, are more than the classification's `monthlyLimit`.

The units bought in the month are the sum of `quantity` over the member's purchases of that material whose `at` lies in the month of the request time. Purchases of other members or of other materials are not counted. The stock that consumables draw is not a purchase and is not counted. Buying exactly `monthlyLimit` units in all is allowed.

The classification in force at the request time is the one judged, whatever it was when earlier purchases were made.

Example: `monthlyLimit` is 10 and a certified member bought 6 units on `2026-03-05`. On `2026-03-20` a purchase of 4 units passes the gate, because 6 + 4 is 10, and a purchase of 5 units is refused with `hazard_limit_exceeded`. On `2026-04-01` the count of the month starts again at 0.

purchase material applies this gate; the answer is supplied to materials as hazmat interface states.

### Classification

The hazard classification of one material: the tool family whose certification a member needs to buy it, and the most units one member may buy of it in a month. A material has at most one classification.

**Fields**, in the formats of values:

- `materialId`: identifier, naming an existing material
- `familyId`: identifier, naming an existing family
- `monthlyLimit`: integer from `1` to `10000` (the most units of the material one member may buy in a month)

A record is identified by `materialId`.

### Classify hazard

Gives the material a classification. When the material already has one, the new classification replaces it. A classification gates the purchases made after it; purchases made earlier are not changed and still count toward the limit.

Rules of other modules that apply here: hazard gate.

- **Wire name:** `classifyHazard`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `materialId`: identifier, naming an existing material
  - `familyId`: identifier, naming an existing family
  - `monthlyLimit`: integer from `1` to `10000` (the most units of the material one member may buy in a month)
- **Result:**
  - `classification`: a classification record
- **Permissions:** admin all.
- **Audit target:** `materialId`.

### Declassify hazard

Removes the material's classification. Later purchases of the material are no longer judged by the hazard gate. The `classification` result is the classification that was removed.

- **Wire name:** `declassifyHazard`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `materialId`: identifier, naming an existing material
- **Refusals** at check 7, in this order:
  1. `not_hazardous` when the material has no classification.
- **Result:**
  - `classification`: a classification record
- **Permissions:** admin all.
- **Audit target:** `materialId`.

### List hazards

Returns all classifications, in ascending `materialId` order. Without any classification the list is empty.

- **Wire name:** `listHazards`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `hazards`: a list of classification records
- **Permissions:** member all, staff all, admin all.

## Module `purchasing`

### Purchasing interface

What other modules may use of purchasing: its operations, the supplier, order and order line records and the refusal codes below. Other modules ask purchasing for nothing else.

- **Operations:** add supplier, set supplier status, create order, add order line, submit order, approve order, receive order, list orders.
- **Uses:** materials interface, ledger interface.
- **Records:** supplier, order, order line.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `approval_requires_admin` | The order's total is above 100000 cents and the actor is staff. |
| `duplicate_line` | The order already has a line for the material. |
| `empty_order` | The order has no line. |
| `not_on_order` | The order has no line for the material. |
| `over_receipt` | The receipt would bring the line's received quantity above its ordered quantity. |
| `self_approval` | The actor created the order. |
| `supplier_blocked` | The supplier is blocked. |

`receiveOrder` also answers `stocktake_open`, which stocktake interface states. Purchasing asks materials whether the material has a count in an open stocktake.

### Receiving

A receipt is made against one line of an approved order, for part or all of what is still open on the line.

- The line's `receivedQuantity` increases by the received quantity. It never exceeds the line's `quantity`.
- The material's stock increases by the received quantity. The material's unit price does not change; the line's `unitCostCents` is what the shop pays, not what it sells for.
- The order stays `approved` while any line has `receivedQuantity` below `quantity`. When every line has `receivedQuantity` equal to `quantity`, the status becomes `received`, and no further receipt is accepted.
- Ledger postings are made as postings states: the amount is the received quantity times the line's `unitCostCents`.

A material that is frozen by a stocktake cannot be received. The supplier's status is not judged at a receipt.

Example: a line of 100 units at 120 cents. A receipt of 40 gives `receivedQuantity` 40, adds 40 to the stock and posts 4800 cents. A receipt of 61 is then refused with `over_receipt`, since 40 + 61 is above 100. A receipt of 60 completes the line.

### Order

An order of materials from one supplier, generated identifier `po-N`. It moves from `draft` to `submitted` to `approved` to `received`.

**Fields**, in the formats of values:

- `orderId`: identifier
- `supplierId`: identifier, naming an existing supplier
- `status`: an order status: `draft`, `submitted`, `approved` or `received`
- `createdBy`: identifier, the user who created the order
- `totalCents`: cent amount, the sum over the lines of `quantity × unitCostCents`; 0 while there is no line
- `lines`: a list of order line records, in the order they were added

A record is identified by `orderId`.

### Order line

One material on an order. An order has at most one line per material.

**Fields**, in the formats of values:

- `materialId`: identifier, naming an existing material
- `quantity`: integer from `1` to `100000` (the number of units ordered)
- `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit from the supplier; it is not the material's unit price)
- `receivedQuantity`: count, the number of units received so far; it starts at 0 and is never above `quantity`

### Order status

The step an order has reached.

| Value | Meaning |
| --- | --- |
| `draft` | The order is being written; lines can be added. |
| `submitted` | The order waits for approval. |
| `approved` | The order is approved; stock can be received against its lines. |
| `received` | Every line is fully received. |

### Supplier

A supplier of materials. A new supplier is `active`.

**Fields**, in the formats of values:

- `supplierId`: identifier
- `name`: text
- `status`: a supplier status: `active` or `blocked`

A record is identified by `supplierId`.

### Supplier status

Whether orders may be created for a supplier and approved.

| Value | Meaning |
| --- | --- |
| `active` | Orders can be created for the supplier and approved. |
| `blocked` | No order can be created for the supplier and no order of the supplier can be approved. Orders in any other step are not affected. |

### Add order line

Adds a line to the order, after its other lines, with `receivedQuantity` 0. The order's `totalCents` increases by `quantity × unitCostCents`; this product is not discounted and not rounded. The supplier's status is not judged.

- **Wire name:** `addOrderLine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `orderId`: identifier, naming an existing order
  - `materialId`: identifier, naming an existing material
  - `quantity`: integer from `1` to `100000` (the number of units ordered)
  - `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit from the supplier)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `draft`.
  2. `duplicate_line` when the order already has a line for `materialId`.
- **Result:**
  - `order`: a order record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.

### Add supplier

Creates the supplier with status `active`.

- **Wire name:** `addSupplier`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `supplierId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a supplier with `supplierId` exists.
- **Result:**
  - `supplier`: a supplier record
- **Permissions:** staff all, admin all.
- **Audit target:** `supplierId`.

### Approve order

The order's status becomes `approved`. Nobody approves an order they created, not even an admin. An order whose `totalCents` is above 100000 cents is approved by an admin only; an order of exactly 100000 cents can be approved by staff. A supplier that is blocked at the request time blocks the approval, whoever acts.

- **Wire name:** `approveOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `orderId`: identifier, naming an existing order
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `submitted`.
  2. `self_approval` when the actor is the order's `createdBy`.
  3. `approval_requires_admin` when `totalCents` is above 100000 and the actor is staff.
  4. `supplier_blocked` when the order's supplier has status `blocked`.
- **Result:**
  - `order`: a order record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.

### Create order

Creates an order for the supplier with the next order identifier, status `draft`, `createdBy` the actor, `totalCents` 0 and no lines.

- **Wire name:** `createOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `supplierId`: identifier, naming an existing supplier
- **Refusals** at check 7, in this order:
  1. `supplier_blocked` when the supplier's status is `blocked`.
- **Result:**
  - `order`: a order record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `orderId`.

### List orders

Returns all orders of the supplier, in creation order, in every status.

- **Wire name:** `listOrders`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `supplierId`: identifier, naming an existing supplier
- **Result:**
  - `supplierId`: identifier
  - `orders`: a list of order records
- **Permissions:** staff all, admin all.

### Receive order

Receives `quantity` units of the material against the order, as receiving states. Ledger postings are made.

Rules of other modules that apply here: receiving, stocktake freeze and postings.

- **Wire name:** `receiveOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `orderId`: identifier, naming an existing order
  - `materialId`: identifier, naming an existing material
  - `quantity`: integer from `1` to `100000` (the number of units received in one request)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `approved`.
  2. `not_on_order` when the order has no line for `materialId`.
  3. `over_receipt` when the line's `receivedQuantity` plus `quantity` is above the line's `quantity`.
  4. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
- **Result:**
  - `order`: a order record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.

### Set supplier status

Sets the supplier's status. Setting the current status succeeds without a change. Orders that exist are not changed; the status is judged when an order is created and when an order is approved.

- **Wire name:** `setSupplierStatus`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `supplierId`: identifier, naming an existing supplier
  - `status`: a supplier status: `active` or `blocked`
- **Result:**
  - `supplier`: a supplier record
- **Permissions:** admin all.
- **Audit target:** `supplierId`.

### Submit order

The order's status becomes `submitted`. Lines can no longer be added.

- **Wire name:** `submitOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `orderId`: identifier, naming an existing order
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `draft`.
  2. `empty_order` when the order has no line.
- **Result:**
  - `order`: a order record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.

## Module `stocktake`

### Stocktake interface

What other modules may use of stocktake: its operations, the stocktake and count records and the refusal codes below. Other modules also ask stocktake whether a material has a count in an open stocktake. materials interface declares this need, and the protocol module supplies the answer.

- **Operations:** start stocktake, record count, close stocktake, get stocktake.
- **Uses:** materials interface, ledger interface.
- **Records:** stocktake, count.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `nothing_counted` | The stocktake has no count. |
| `stocktake_open` | The material has a count in an open stocktake. |
| `stocktake_running` | A stocktake is already open. |

### Stocktake freeze

A material is **counted** when it has a count in an open stocktake. A counted material is frozen: three operations refuse to change its stock with `stocktake_open`, in their own order of refusals:

- purchase material,
- restock material,
- `receiveOrder`.

A material without a count is not frozen, even while a stocktake is open. Recording a count freezes the material at once; replacing the count keeps it frozen. Closing the stocktake ends the freeze of all its materials. A closed stocktake freezes nothing.

The freeze stops nothing else. In particular the stock that consumables draw at check-out still decreases, and closing compares each `counted` with the stock at that moment.

### Count

The count of one material in one stocktake: how many units were found on the shelf. A stocktake holds at most one count per material. `difference` is `null` until the stocktake is closed; closing sets it, and it stays.

**Fields**, in the formats of values:

- `materialId`: identifier, naming an existing material
- `counted`: integer from `0` to `100000000` (the number of units counted)
- `difference`: integer, or `null`. It is `counted` minus the material's stock at closing, so it is negative when units are missing and may be 0.

### Stocktake

A count of the shop's stock, generated identifier `stk-N`. It is `open` until it is closed. At most one stocktake is open at a time.

**Fields**, in the formats of values:

- `stocktakeId`: identifier
- `status`: `open` or `closed`
- `counts`: a list of count records, in ascending `materialId` order; empty for a new stocktake

A record is identified by `stocktakeId`.

### Close stocktake

Closes the stocktake: its status becomes `closed`. For each count, in ascending `materialId` order, `difference` is `counted` minus the material's stock at the request time, and the material's stock becomes `counted`. The freeze of the counted materials ends.

Ledger postings are made, for each count in that order and one per count whose `difference` is not 0, as postings states. The `units` of a posting are the absolute value of `difference` and the price is the material's current unit price at the request time. A negative `difference` is a shrink; a positive `difference` is a gain. A count with `difference` 0 posts nothing.

Example: the stock of a material is 10 and its unit price is 250 cents. Counted 8: `difference` is −2, the stock becomes 8 and the shrink posts 500 cents. Counted 15: `difference` is 5, the stock becomes 15 and the gain posts 1250 cents.

Rules of other modules that apply here: stocktake freeze and postings.

- **Wire name:** `closeStocktake`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `stocktakeId`: identifier, naming an existing stocktake
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the stocktake is closed.
  2. `nothing_counted` when the stocktake has no count.
- **Result:**
  - `stocktake`: a stocktake record
- **Permissions:** staff all, admin all.
- **Audit target:** `stocktakeId`.

### Get stocktake

Returns the stocktake with its counts, in ascending `materialId` order.

- **Wire name:** `getStocktake`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `stocktakeId`: identifier, naming an existing stocktake
- **Result:**
  - `stocktake`: a stocktake record
- **Permissions:** staff all, admin all.

### Record count

Records the count of the material in the stocktake. When the stocktake already has a count of that material, the new `counted` replaces it. The material's stock does not change until the stocktake is closed. From the moment a material has a count in an open stocktake, the stocktake freeze applies to it.

- **Wire name:** `recordCount`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `stocktakeId`: identifier, naming an existing stocktake
  - `materialId`: identifier, naming an existing material
  - `counted`: integer from `0` to `100000000` (the number of units counted)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the stocktake is closed.
- **Result:**
  - `stocktake`: a stocktake record
- **Permissions:** staff all, admin all.
- **Audit target:** `stocktakeId/materialId`.

### Start stocktake

Creates a stocktake with the next stocktake identifier, status `open` and no counts.

- **Wire name:** `startStocktake`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Refusals** at check 7, in this order:
  1. `stocktake_running` when a stocktake with status `open` exists.
- **Result:**
  - `stocktake`: a stocktake record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `stocktakeId`.

## Module `workorders`

### Workorders interface

What other modules may use of workorders: its operations and the work order record. Other modules also ask the capability listed below.

- **Operations:** open work order, link incident, use parts, close work order, list work orders.
- **Uses:** members interface, equipment interface, safety interface, parts interface, lockouts interface.
- **Records:** work order.

Capability other modules ask of workorders:

- Opening a work order for a machine with a summary, for a given opener and already linked to a given incident of that machine. It has the effects of open work order followed by link incident, without their refusals and without an audit entry of its own, and it returns the work order. Escalations uses this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_linked` | The work order is already linked to an incident. |
| `incident_open` | The incident linked to the work order is not resolved. |
| `incident_other_machine` | The incident is about another machine than the work order's. |
| `insufficient_parts` | The part's stock is lower than the requested quantity. |
| `part_wrong_family` | The part fits another family than the work order's machine. |

### Work order lifecycle

A work order is a repair job on one machine. Opening it applies a lockout of source `workOrder` to the machine (lockouts), and the work order keeps that lockout's identifier as `lockoutId`. The machine stays locked out by this lockout until the work order is closed; closing releases exactly that lockout. Other active lockouts of the machine are not touched, so a machine with a manual lockout stays locked out after its work order closes.

While a work order is `open`, it can be linked to one incident of its machine, and spare parts of the machine's family can be used on it. A used part leaves the part's stock and adds quantity times the part's `unitCostCents` to the work order's `partsCostCents`. The cost is only recorded on the work order; no charge is created for anyone.

A work order cannot be closed while its linked incident is `open`. A work order without a linked incident can be closed at once. A closed work order cannot be changed, and work orders are never removed.

Work orders are opened by staff or an admin, or by the escalations module for an escalated incident.

### Work order

A repair job on one machine, generated identifier `wko-N`; see work order lifecycle. `incidentId` is `null` until an incident is linked. `closedAt` and `note` are `null` while the work order is open.

**Fields**, in the formats of values:

- `workOrderId`: identifier
- `machineId`: identifier, naming an existing machine
- `summary`: text
- `status`: `open` or `closed`
- `incidentId`: identifier, naming an existing incident, or `null`
- `lockoutId`: identifier, naming an existing lockout
- `partsCostCents`: cent amount
- `openedBy`: identifier, naming an existing user
- `openedAt`: timestamp
- `closedAt`: timestamp, or `null`
- `note`: text, or `null`

A record is identified by `workOrderId`.

### Close work order

The work order gets status `closed`, `closedAt` the request time and `note` as supplied. The lockout named by its `lockoutId` is released: it gets `releasedAt` the request time. Other lockouts of the machine stay.

Rules of other modules that apply here: lockout gate.

- **Wire name:** `closeWorkOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `workOrderId`: identifier, naming an existing work order
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is already closed.
  2. `incident_open` when the work order has a linked incident whose `status` is `open`.
- **Result:**
  - `workOrder`: a work order record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.

### Link incident

The work order's `incidentId` becomes the supplied incident. The incident may be open or resolved, and several work orders may link the same incident; a work order links only one.

- **Wire name:** `linkIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `workOrderId`: identifier, naming an existing work order
  - `incidentId`: identifier, naming an existing incident
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is closed.
  2. `incident_other_machine` when the incident's `machineId` is not the work order's `machineId`.
  3. `already_linked` when the work order's `incidentId` is not `null`.
- **Result:**
  - `workOrder`: a work order record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.

### List work orders

Returns all work orders of the machine, open and closed, in creation order.

- **Wire name:** `listWorkOrders`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
- **Result:**
  - `machineId`: identifier
  - `workOrders`: a list of work order records
- **Permissions:** staff all, admin all.

### Open work order

Creates a work order with the next work order identifier, `status` `open`, `incidentId` `null`, `partsCostCents` 0, `openedBy` the actor, `openedAt` the request time, and `closedAt` and `note` `null`. It applies a lockout to the machine with source `workOrder`, `reason` the supplied `summary`, `appliedBy` the actor and `appliedAt` the request time, and the work order's `lockoutId` is that lockout's identifier. The machine's other lockouts stay.

Rules of other modules that apply here: lockout gate.

- **Wire name:** `openWorkOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `machineId`: identifier, naming an existing machine
  - `summary`: text
- **Result:**
  - `workOrder`: a work order record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `workOrderId`.

### Use parts

The part's stock decreases by `quantity`, and the work order's `partsCostCents` grows by `quantity × unitCostCents` of the part. The `part` result is the part after the change.

- **Wire name:** `useParts`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `workOrderId`: identifier, naming an existing work order
  - `partId`: identifier, naming an existing part
  - `quantity`: integer from `1` to `1000` (the number of units used in one request)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is closed.
  2. `part_wrong_family` when the part's `familyId` is not the family of the work order's machine.
  3. `insufficient_parts` when the part's stock is lower than `quantity`.
- **Result:**
  - `workOrder`: a work order record
  - `part`: a part record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.

## Module `escalations`

### Escalations interface

What other modules may use of escalations: its operations and the escalation and policy records. No other module asks anything of it beyond these.

- **Operations:** set escalation policy, escalate incident, list escalations.
- **Uses:** equipment interface, safety interface, lockouts interface, bookings interface, workorders interface, notices interface.
- **Records:** escalation, escalation policy.
- **Rules that bind other modules:** escalation effects.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_escalated` | The incident has been escalated before. |
| `below_policy` | The incident's severity is lower than the minimum severity of its family's policy. |
| `no_policy` | The family of the incident's machine has no escalation policy. |

### Escalation effects

An escalation (escalate incident) acts on a incident that a policy says is serious enough. All effects take place together, in this order:

1. A work order is opened for the incident's machine with the summary `Escalated incident <incidentId>`, for example `Escalated incident inc-3`. It is linked to the incident, it holds its own lockout of source `workOrder`, and its opener is the actor, all as open work order and link incident state.
2. Every reservation of the machine whose reported status is `reserved` and whose `start` is after the request time is cancelled. They are taken in ascending `start` order, then in creation order. Each gets status `cancelled`, as cancel reservation states: no fee and no credit. Its member gets a notice of kind `cancellation` and `ref` the reservation identifier, posted as posting notices states; the notice takes its subject from the kind. The notices are numbered in the order of the cancellations.

Reservations that do not meet both conditions are not touched: one that is checked in, one that is `reserved` but has `start` at or before the request time, and one that is missed, completed or already cancelled. A session that is checked in goes on.

The escalation lists the cancelled reservations in `cancelledReservationIds` in the order they were cancelled, which is the order of the notices, and the new work order in `workOrderId`. The lockout, the cancellations and the notices are part of the escalation and are not audited on their own.

### Escalation

The record that an incident was escalated, generated identifier `esc-N`; see escalation effects. An incident is escalated at most once.

**Fields**, in the formats of values:

- `escalationId`: identifier
- `incidentId`: identifier, naming an existing incident
- `machineId`: identifier, naming an existing machine (the incident's machine)
- `workOrderId`: identifier, naming an existing work order (the one the escalation opened)
- `cancelledReservationIds`: a list of identifiers of reservation records, empty when none was cancelled, in the order they were cancelled
- `at`: timestamp (the request time)

A record is identified by `escalationId`.

### Escalation policy

The least severe incident that is escalated for the machines of one tool family. Severities are ordered `low` < `medium` < `high`. A family has at most one policy; families without one cannot have an incident escalated.

**Fields**, in the formats of values:

- `familyId`: identifier, naming an existing family
- `minSeverity`: one of `low`, `medium` or `high`

A record is identified by `familyId`.

### Escalate incident

Creates an escalation with the next escalation identifier, `incidentId` as supplied, `machineId` the incident's machine, `workOrderId` the work order it opens, `cancelledReservationIds` the reservations it cancels and `at` the request time. It has the effects that escalation effects states. The incident itself is not changed: it stays `open` until it is resolved.

- **Wire name:** `escalateIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `incidentId`: identifier, naming an existing incident
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the incident's `status` is `resolved`.
  2. `already_escalated` when an escalation of the incident exists.
  3. `no_policy` when the family of the incident's machine has no escalation policy.
  4. `below_policy` when the incident's `severity` is lower than the policy's `minSeverity`, with `low` < `medium` < `high`. An equal severity is not below.
- **Result:**
  - `escalation`: an escalation record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `escalationId`.

### List escalations

Returns all escalations in creation order.

- **Wire name:** `listEscalations`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `escalations`: a list of escalation records
- **Permissions:** staff all, admin all.

### Set escalation policy

Sets the policy of the family: a family without a policy gets one, and a family with a policy has it replaced. Setting the current value succeeds without a change. Escalations made earlier are not affected.

- **Wire name:** `setEscalationPolicy`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `familyId`: identifier, naming an existing family
  - `minSeverity`: one of `low`, `medium` or `high`
- **Result:**
  - `policy`: an escalation policy record
- **Permissions:** admin all.
- **Audit target:** `familyId`.

## Module `projects`

### Projects interface

What other modules may use of projects: its operations, the project and cost share records and the refusal codes below. Other modules ask projects for nothing else.

- **Operations:** create project, add collaborator, attach reservation, attach purchase, close project, get project cost.
- **Uses:** members interface, bookings interface, materials interface.
- **Records:** project, cost share.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_attached` | The reservation or purchase is already attached to a project. |
| `already_collaborator` | The member is the project's owner or already one of its collaborators. |
| `not_project_member` | The member of the reservation or purchase is neither the owner nor a collaborator of the project. |
| `project_closed` | The project is closed. |

### Attachment

A project holds reservations and purchases of its members, which are its owner and its collaborators.

- Only an open project takes changes. Adding a collaborator or attaching anything to a closed project is refused with `project_closed`. Closing is final, and a closed project can still be read.
- A reservation or a purchase can be attached when its member is the project's owner or one of its collaborators at the request time.
- A reservation or a purchase belongs to at most one project and stays there. Attaching one that is already attached, to this or to any other project, is refused with `already_attached`.
- A reservation can be attached in any status. Only completed reservations count in the cost.

An owner can add collaborators and attach records. A collaborator cannot act on the project as a member: a member acts on a project only as its owner, which authorization states.

### Project cost

The cost of a project is read from the records attached to it, at the request time of the read.

- `usageCents` is the sum of `amountCents` over the attached reservations whose status is `completed`. Reservations in any other status add nothing.
- `materialCents` is the sum of `amountCents` over the attached purchases. The amount of a purchase is the amount after its discount.
- `totalCents` is `usageCents` plus `materialCents`.
- `byMember` has one cost share for each member who has a share, in ascending `memberId` order. A member has a share when at least one attached completed reservation or at least one attached purchase belongs to the member. The member's `usageCents` is the sum over the member's attached completed reservations and the member's `materialCents` is the sum over the member's attached purchases. Summed over `byMember`, the shares equal `usageCents` and `materialCents`.

Example: `ada` has an attached completed reservation of 1499 cents and an attached purchase of 600 cents, and `bo`, a collaborator, has an attached purchase of 250 cents. `usageCents` is 1499, `materialCents` is 850 and `totalCents` is 2349. `byMember` is `ada` with 1499 and 600, then `bo` with 0 and 250.

### Cost share

The share of one member in the cost of a project, as project cost states.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `usageCents`: cent amount
- `materialCents`: cent amount

### Project

A project of a member, who owns it, with collaborators. Reservations and purchases of its members are attached to it so that their cost can be read together. The identifier is chosen by the caller.

**Fields**, in the formats of values:

- `projectId`: identifier
- `ownerId`: identifier, naming an existing member
- `title`: text
- `collaboratorIds`: a list of identifiers, each naming an existing member, in ascending order; the owner is not in it
- `status`: `open` or `closed`
- `reservationIds`: a list of identifiers, each naming an existing reservation, in the order they were attached
- `purchaseIds`: a list of identifiers, each naming an existing purchase, in the order they were attached

A record is identified by `projectId`.

### Add collaborator

Adds the member to the project's `collaboratorIds`, which stay in ascending order. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on `memberId`, as authorization states.

Rules of other modules that apply here: attachment and authorization.

- **Wire name:** `addCollaborator`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier, naming an existing project
  - `memberId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `already_collaborator` when `memberId` is the project's owner or is already in `collaboratorIds`.
- **Result:**
  - `project`: a project record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/memberId`.

### Attach purchase

Adds the purchase to the project's `purchaseIds`, after the ones already there. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on the purchase, as authorization states.

Rules of other modules that apply here: attachment and authorization.

- **Wire name:** `attachPurchase`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier, naming an existing project
  - `purchaseId`: identifier, naming an existing purchase
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `not_project_member` when the purchase's member is neither the project's owner nor in its `collaboratorIds`.
  3. `already_attached` when the purchase is attached to any project.
- **Result:**
  - `project`: a project record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/purchaseId`.

### Attach reservation

Adds the reservation to the project's `reservationIds`, after the ones already there. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on the reservation, as authorization states.

Rules of other modules that apply here: attachment, reservations and authorization.

- **Wire name:** `attachReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier, naming an existing project
  - `reservationId`: identifier, naming an existing reservation
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `not_project_member` when the reservation's member is neither the project's owner nor in its `collaboratorIds`.
  3. `already_attached` when the reservation is attached to any project.
- **Result:**
  - `project`: a project record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/reservationId`.

### Close project

The project's status becomes `closed`. A closed project takes no collaborator and no attachment, and its cost can still be read (attachment). A member acts only on a project the member owns, as authorization states.

- **Wire name:** `closeProject`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier, naming an existing project
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the project's status is `closed`.
- **Result:**
  - `project`: a project record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId`.

### Create project

Creates the project with status `open`, no collaborators and no attached reservations or purchases. The owner's standing is not judged. A member creates only projects of the member's own: ownership is judged on `ownerId`, which authorization states.

Rules of other modules that apply here: authorization.

- **Wire name:** `createProject`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier
  - `ownerId`: identifier, naming an existing member
  - `title`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a project with `projectId` exists.
- **Result:**
  - `project`: a project record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId`.

### Get project cost

Returns the cost of the project as project cost states. A member reads only the cost of a project the member owns, as authorization states.

Rules of other modules that apply here: project cost and authorization.

- **Wire name:** `getProjectCost`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `projectId`: identifier, naming an existing project
- **Result:**
  - `projectId`: identifier
  - `usageCents`: cent amount
  - `materialCents`: cent amount
  - `totalCents`: cent amount
  - `byMember`: a list of cost share records, in ascending `memberId` order
- **Permissions:** member own, staff all, admin all.

## Module `quotes`

### Quotes interface

What other modules may use of quotes: its operations, the customer, quote and quote line records and the refusal codes below. Other modules also ask quotes, for a quote identifier, whether the quote exists and for its customer, its status, its `totalCents` and its lines.

- **Operations:** add customer, create quote, add quote line, send quote, decide quote, get quote.
- **Uses:** equipment interface, materials interface.
- **Records:** customer, quote, quote line.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `empty_quote` | The quote has no line. |
| `quote_expired` | The quote is accepted after the date it is valid until. |
| `unknown_line_ref` | The line names no machine or no material. |

### Quote pricing

A quote is priced once, when it is sent, at the request time of the sending.

- Each line's `unitPriceCents` is captured: the machine's current rate in cents per minute for a `machine` line (machine), the material's current unit price for a `material` line (material). Later changes of a rate or a price never change a sent quote.
- `subtotalCents` is the sum over the lines of `quantity × unitPriceCents`. No discount applies and nothing is rounded.
- `totalCents` is `(subtotalCents × 120 + 50) div 100`, which is the subtotal plus 20 percent, rounded half up to a whole cent once, as money states for amounts.

Example: a `machine` line of 30 minutes at 45 cents and a `material` line of 7 units at 236 cents give a subtotal of `1350 + 1652 = 3002` and a total of `(3002 × 120 + 50) div 100 = 3602` cents.

Stock is not reserved or drawn by a quote.

### Quote validity

A quote that is sent has `sentAt` equal to the request time and `validUntil` equal to the date 30 days after the date of the request time (time).

A sent quote can be accepted on every request date up to and including `validUntil`. On the first date after `validUntil` an acceptance is refused with `quote_expired`. A rejection is never refused for age. A quote that has passed `validUntil` stays `sent`.

Example: a quote sent at `2026-03-02T15:00Z` is valid until `2026-04-01`. It can be accepted at `2026-04-01T23:59Z` and not at `2026-04-02T00:00Z`.

### Customer

A customer of the workshop's made-to-order work. A customer is not a user and not a member; it has no standing and no account.

**Fields**, in the formats of values:

- `customerId`: identifier
- `name`: text

A record is identified by `customerId`.

### Quote

An offer to a customer for machine time and materials, generated identifier `quo-N`. It moves from `draft` to `sent` to `accepted` or `rejected`. `subtotalCents` and `totalCents` are 0, and `sentAt` and `validUntil` are `null`, until the quote is sent.

**Fields**, in the formats of values:

- `quoteId`: identifier
- `customerId`: identifier, naming an existing customer
- `title`: text
- `status`: a quote status: `draft`, `sent`, `accepted` or `rejected`
- `lines`: a list of quote line records, in the order they were added
- `subtotalCents`: cent amount
- `totalCents`: cent amount
- `sentAt`: timestamp, or `null`
- `validUntil`: date, or `null`; it can lie after 2099 (computed times)

A record is identified by `quoteId`.

### Quote line

One line of a quote: machine time or a material. The same `ref` can be on several lines. `unitPriceCents` is `null` until the quote is sent and is fixed by sending, as quote pricing states.

**Fields**, in the formats of values:

- `lineKind`: `machine` or `material`
- `ref`: identifier; for a `machine` line the identifier of an existing machine, for a `material` line the identifier of an existing material
- `quantity`: integer from `1` to `100000`; for a `machine` line the minutes of machine time, for a `material` line the units
- `unitPriceCents`: integer from `1` to `1000000` cents, or `null`; for a `machine` line the machine's rate in cents per minute when the quote was sent, for a `material` line the material's unit price when the quote was sent

### Quote status

The step a quote has reached.

| Value | Meaning |
| --- | --- |
| `draft` | The quote is being written; lines can be added. |
| `sent` | The quote is priced and sent to the customer; it waits for a decision. |
| `accepted` | The customer accepted the quote; a job can start from it. |
| `rejected` | The customer rejected the quote. |

### Add customer

Creates the customer.

- **Wire name:** `addCustomer`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `customerId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a customer with `customerId` exists.
- **Result:**
  - `customer`: a customer record
- **Permissions:** staff all, admin all.
- **Audit target:** `customerId`.

### Add quote line

Adds a quote line to the quote, after its other lines, with `unitPriceCents` `null`. Nothing else of the quote changes. A line is never changed or removed.

- **Wire name:** `addQuoteLine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `quoteId`: identifier, naming an existing quote
  - `lineKind`: `machine` or `material`
  - `ref`: identifier; it is judged by the second refusal below, not at check 5
  - `quantity`: integer from `1` to `100000` (minutes of machine time for a `machine` line, units for a `material` line)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `draft`.
  2. `unknown_line_ref` when `lineKind` is `machine` and no machine has the identifier `ref`, or `lineKind` is `material` and no material has it.
- **Result:**
  - `quote`: a quote record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.

### Create quote

Creates a quote for the customer with the next quote identifier, status `draft`, no lines, `subtotalCents` and `totalCents` 0, and `sentAt` and `validUntil` `null`.

- **Wire name:** `createQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `customerId`: identifier, naming an existing customer
  - `title`: text
- **Result:**
  - `quote`: a quote record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `quoteId`.

### Decide quote

The quote's status becomes `accepted` or `rejected`, as `decision` says. Nothing else of the quote changes.

Rules of other modules that apply here: quote validity.

- **Wire name:** `decideQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `quoteId`: identifier, naming an existing quote
  - `decision`: `accepted` or `rejected`
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `sent`.
  2. `quote_expired` when `decision` is `accepted` and the date of the request time is after `validUntil`.
- **Result:**
  - `quote`: a quote record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.

### Get quote

Returns the quote with its lines in the order they were added.

- **Wire name:** `getQuote`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `quoteId`: identifier, naming an existing quote
- **Result:**
  - `quote`: a quote record
- **Permissions:** staff all, admin all.

### Send quote

Prices and sends the quote: its status becomes `sent`, every line gets its `unitPriceCents`, and `subtotalCents`, `totalCents`, `sentAt` and `validUntil` are set, all as quote pricing and quote validity state.

Rules of other modules that apply here: quote pricing, quote validity and money.

- **Wire name:** `sendQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `quoteId`: identifier, naming an existing quote
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `draft`.
  2. `empty_quote` when the quote has no line.
- **Result:**
  - `quote`: a quote record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.

## Module `jobs`

### Jobs interface

What other modules may use of jobs: its operations, the job record and the refusal codes below. Other modules ask jobs for nothing else.

- **Operations:** start job, complete job, record job payment, list jobs.
- **Uses:** members interface, equipment interface, billing interface, ledger interface, training interface, quotes interface.
- **Records:** job.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_started` | A job already exists for the quote. |
| `job_not_completed` | The job is not completed. |
| `maker_not_certified` | The maker has no certification valid on the date of the request time for the family of a machine of the quote. |
| `overpayment` | The payment would bring the job's paid amount above its total. |
| `quote_not_accepted` | The quote is not accepted. |

### Maker credit

When a job is completed, its maker receives 30 percent of the job's `totalCents` as credit: `makerCreditCents` is `(totalCents × 30 + 50) div 100`, rounded half up once as money states for amounts. The maker's credit balance increases by `makerCreditCents` (billing interface). The credit applies to the maker's next invoices, like any other credit.

The credit is posted to the ledger as postings states. A `makerCreditCents` of 0 changes no balance and posts nothing.

Example: a job of 3602 cents gives `(3602 × 30 + 50) div 100 = 1081` cents. A job of 1 cent gives 0.

### Maker gate

A job starts only from an accepted quote and only with a maker who may do the work.

- A quote starts at most one job. A quote that has a job, started or completed, cannot start another.
- The maker must be in good standing (membership).
- For every `machine` line of the quote, the maker must have a certification for the family of the line's machine that is valid on the date of the request time (certifications). A quote with only `material` lines needs no certification. Several lines with machines of one family need that family's certification once.

The checks run in the order of the refusals of start job.

### Job

The work done for an accepted quote by a member, the maker, generated identifier `job-N`. It is `started` until the maker completes it. `makerCreditCents` is `null` until then.

**Fields**, in the formats of values:

- `jobId`: identifier
- `quoteId`: identifier, naming an existing quote
- `customerId`: identifier, naming an existing customer; the quote's customer
- `makerId`: identifier, naming an existing member
- `status`: `started` or `completed`
- `totalCents`: cent amount; the quote's `totalCents`
- `makerCreditCents`: cent amount, or `null`
- `paidCents`: cent amount; the sum of the payments recorded for the job, 0 at the start

A record is identified by `jobId`.

### Complete job

The job's status becomes `completed`. `makerCreditCents` is set and added to the maker's credit balance, as maker credit states. The maker's standing is not judged.

Rules of other modules that apply here: maker credit, postings and money.

- **Wire name:** `completeJob`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `jobId`: identifier, naming an existing job
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the job's status is not `started`.
- **Result:**
  - `job`: a job record
- **Permissions:** staff all, admin all.
- **Audit target:** `jobId`.

### List jobs

Returns all jobs of the customer, in creation order, in every status.

- **Wire name:** `listJobs`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `customerId`: identifier, naming an existing customer
- **Result:**
  - `customerId`: identifier
  - `jobs`: a list of job records
- **Permissions:** staff all, admin all.

### Record job payment

The job's `paidCents` increases by `amountCents`. The customer pays the job directly; no invoice and no credit is involved. A payment that brings `paidCents` exactly to `totalCents` is accepted, and a job that is fully paid takes no further payment. Ledger postings are made.

Rules of other modules that apply here: postings.

- **Wire name:** `recordJobPayment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `jobId`: identifier, naming an existing job
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid in one request)
- **Refusals** at check 7, in this order:
  1. `job_not_completed` when the job's status is not `completed`.
  2. `overpayment` when `paidCents` plus `amountCents` is above `totalCents`.
- **Result:**
  - `job`: a job record
- **Permissions:** staff all, admin all.
- **Audit target:** `jobId`.

### Start job

Creates a job for the accepted quote with the next job identifier, status `started`, `customerId` and `totalCents` those of the quote, `makerCreditCents` `null` and `paidCents` 0.

Rules of other modules that apply here: maker gate, membership and certifications.

- **Wire name:** `startJob`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `quoteId`: identifier, naming an existing quote
  - `makerId`: identifier, naming an existing member
- **Refusals** at check 7, in this order:
  1. `quote_not_accepted` when the quote's status is not `accepted`.
  2. `already_started` when a job exists for the quote.
  3. `member_suspended` when the maker is not in good standing (members interface).
  4. `maker_not_certified` when the quote has a `machine` line and the maker has no certification valid on the date of the request time for the family of the machine of some such line (maker gate).
- **Result:**
  - `job`: a job record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `jobId`.

## Module `compliance`

### Compliance interface

What other modules may use of compliance: its operations and the family compliance record. No other module asks anything of it beyond these. Compliance only reads; it changes nothing and adds no refusal codes.

- **Operations:** check eligibility, compliance summary.
- **Uses:** members interface, equipment interface, guardians interface, waivers interface, PPE interface, training interface, lockouts interface, inspections interface.
- **Records:** family compliance.

### Eligibility

A member is **eligible** for a machine on a date when none of the following **blockers** holds. Each blocker is judged as `createReservation` judges its refusal of the same name for a slot that starts on that date, at the request time. The blockers are listed in the order of that operation's refusals:

1. `member_suspended`: the member is not in good standing (members interface).
2. `guardian_consent_missing`: the member is a minor on the date and holds no consent for the machine's family (guardian consent).
3. `not_certified`: the member has no certification for the machine's family valid on the date (training interface).
4. `waiver_missing`: the waiver gate refuses the member for the machine's family.
5. `machine_locked_out`: the machine has an active lockout at the request time, whatever the date (lockout gate).
6. `inspection_due`: the machine has an inspection plan and the date is on or after its `nextDueOn` (inspection gate).

The rules that depend on the slot itself are not judged: granularity, opening hours, a start in the past, maintenance, supervision and conflicts with other reservations. The check-in rules `ppe_missing` and `not_on_premises` are not judged either. The date may lie in the past.

Example: a member in good standing, certified from `2026-01-10` for 365 days, asks about a machine on `2026-03-31`. The machine has a plan with `nextDueOn` `2026-03-31` and no lockout. The only blocker is `inspection_due`; for `2026-03-30` the member is eligible.

### Family compliance

How one member stands towards one tool family at the request time. compliance summary returns one for every family.

**Fields**, in the formats of values:

- `familyId`: identifier, naming an existing family
- `certified`: flag, `true` when the member holds a certification for the family that is valid on the date of the request time (certifications)
- `consent`: flag, `true` when the member is not a minor on the date of the request time, or holds consent for the family (guardian consent)
- `waiverSigned`: flag, `true` when the family has no waiver, or the member's latest signature for the family is of the current version (waiver gate)
- `ppeHeld`: flag, `true` when the member holds every item the family requires, which includes a family that requires none (PPE gate)

A record is identified by `familyId` within one summary.

### Check eligibility

Tells whether the member could reserve the machine for a slot that starts on `date`, and what stops them. `blockers` holds the codes of the blockers that hold, in the order eligibility lists them, and is `[]` when none holds. `eligible` is `true` exactly when `blockers` is empty.

- **Wire name:** `checkEligibility`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
  - `machineId`: identifier, naming an existing machine
  - `date`: date
- **Result:**
  - `memberId`: identifier
  - `machineId`: identifier
  - `date`: date
  - `eligible`: flag
  - `blockers`: a list of refusal codes
- **Permissions:** member own, staff all, admin all.

### Compliance summary

Shows where the member stands. `standing` is the member's standing. `minor` is `true` when the member is a minor on the date of the request time (guardian consent), otherwise `false`. `families` has one family compliance row for every family that exists at the request time, in `familyId` ascending order, and is `[]` when there are no families.

- **Wire name:** `complianceSummary`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `memberId`: identifier, naming an existing member
- **Result:**
  - `memberId`: identifier
  - `standing`: `good` or `suspended`
  - `minor`: flag
  - `families`: a list of family compliance records
- **Permissions:** member own, staff all, admin all.

## Module `finance`

### Finance interface

What other modules may use of finance: its four reads and their row records. No other module asks anything else of it.

- **Operations:** charges by kind, finance summary, aged receivables, household statement.
- **Uses:** members interface, ledger interface, billing interface, households interface, refunds interface.
- **Records:** charge kind row, aged receivable row, household statement row.

### Aged receivable row

One open invoice's row of the aged receivables report.

**Fields**, in the formats of values:

- `invoiceId`: identifier, naming an existing invoice
- `memberId`: identifier, naming an existing member
- `balanceCents`: cent amount (the invoice's balance at the request time)
- `daysOverdue`: count (the whole days from the date of the invoice's `dueAt` to the report date, and 0 when the report date is not after that date)

### Charge kind row

One kind's row of the charges-by-kind report.

**Fields**, in the formats of values:

- `kind`: one of `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee` or `event`
- `invoicedCents`: cent amount (the sum of the line amounts of this kind on the invoices of the month)
- `refundedCents`: cent amount (the sum of the refunds of those lines)

### Household statement row

One household member's row of a household statement.

**Fields**, in the formats of values:

- `memberId`: identifier, naming an existing member
- `invoiceId`: identifier, or `null` (the member's invoice for the month; `null` when the member has none)
- `totalCents`: cent amount (the invoice's `totalCents`, and 0 without an invoice)
- `balanceCents`: cent amount (the invoice's `balanceCents` at the request time, and 0 without an invoice)

### Aged receivables

Lists the open invoices by how long they have been overdue on a date. An invoice is included when its status is `open` at the request time, whatever the date is. `daysOverdue` of a row is the number of whole days from the date of the invoice's `dueAt` to `date`: the difference of the two calendar dates, and 0 when `date` is on or before the date of `dueAt`. The time of day of `dueAt` does not matter here. `balanceCents` is the invoice's balance at the request time. `rows` is in ascending invoice number, the number `N` of `inv-N` compared as a number.

Each balance falls in one bucket by its `daysOverdue`: `currentCents` for 0 days, `days1to30Cents` for 1 to 30, `days31to60Cents` for 31 to 60 and `over60Cents` for 61 or more. A bucket is the sum of the balances in it and is 0 when it is empty.

Example: an invoice has `dueAt` `2026-04-15T08:00Z`. For `date` `2026-04-15` its `daysOverdue` is 0 (current), for `2026-04-16` it is 1, for `2026-05-15` it is 30, for `2026-05-16` it is 31, for `2026-06-14` it is 60 and for `2026-06-15` it is 61.

- **Wire name:** `agedReceivables`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `date`: date
- **Result:**
  - `date`: date
  - `rows`: a list of aged receivable row records
  - `currentCents`: cent amount
  - `days1to30Cents`: cent amount
  - `days31to60Cents`: cent amount
  - `over60Cents`: cent amount
- **Permissions:** staff all, admin all.

### Charges by kind

Reports the invoiced charges of a month by kind. The month's invoices are the invoices generated for the month. `rows` has one row for each kind, in the order `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event`, including kinds without lines. For a row:

- `invoicedCents` is the sum of `amountCents` of the lines of that kind on the month's invoices;
- `refundedCents` is the sum of all refunds (refund) of those lines, whenever the refunds were issued. A line is identified by its invoice and its position in the invoice's `lines`.

`totalCents` is the sum of `invoicedCents` over the rows, which is the sum of `totalCents` over the month's invoices. For a month without generated invoices every figure is 0.

- **Wire name:** `chargesByKind`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Result:**
  - `month`: month
  - `rows`: a list of charge kind row records, one per kind, in the order above
  - `totalCents`: cent amount
- **Permissions:** staff all, admin all.

### Finance summary

Summarizes the money of one month, calculated at the request time:

- `invoicedCents`: the sum of `amountCents` of all lines of all kinds on the invoices of the month;
- `refundedCents`: the sum of all refunds (refund) of lines of those invoices, whenever the refunds were issued;
- `paymentsCents`: the sum of `amountCents` of all payments whose `at` lies in the month, as in `revenueReport`;
- `creditsIssuedCents`: the sum of credits issued with issue credit whose request time lies in the month, as in `revenueReport`;
- `journalEntries`: the number of journal entries whose `at` lies in the month;
- `trialBalanced`: true when the total of all debits of the whole journal equals the total of all credits, which the postings of the ledger make true at all times.

For a month without generated invoices, `invoicedCents` and `refundedCents` are 0.

- **Wire name:** `financeSummary`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `month`: month
- **Result:**
  - `month`: month
  - `invoicedCents`: cent amount
  - `refundedCents`: cent amount
  - `paymentsCents`: cent amount
  - `creditsIssuedCents`: cent amount
  - `journalEntries`: count
  - `trialBalanced`: flag
- **Permissions:** admin all.

### Household statement

Shows the month's invoices of the members of a household. `rows` has one row for each member who belongs to the household at the request time, in ascending `memberId` order. A member's invoice is the invoice generated for that member for the month; a member has at most one. A member without one has `invoiceId` `null`, `totalCents` 0 and `balanceCents` 0. `totalCents` and `balanceCents` of the result are the sums over the rows.

- **Wire name:** `householdStatement`, a read.
- **Arguments** besides `actor` and `at`, in the formats of values:
  - `householdId`: identifier, naming an existing household
  - `month`: month
- **Result:**
  - `householdId`: identifier
  - `month`: month
  - `rows`: a list of household statement row records
  - `totalCents`: cent amount
  - `balanceCents`: cent amount
- **Permissions:** staff all, admin all.

## Delivery

### Documentation

`realization/README.md` explains how to build, run and test the program, shows a short example session and states the scope. Code, tests and documentation agree with this document.

### Tests

The tests run with `go test ./...` from `realization/`. They cover, for every operation, a successful request and each refusal code the operation lists; the four policies, including the permission matrix and audit entries; the boundary rules of time (half-open intervals, opening hours, granularity and duration limits) and of money (rounding per line); the cross-area rules of reservations (bookings) and invoices (billing); and the postings of the journal (ledger).
