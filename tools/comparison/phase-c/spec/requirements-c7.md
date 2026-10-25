# Makerspace requirements

This document describes the Makerspace program to build. Links point to the related sections.

## Overview

Makerspace runs the day-to-day operations of a fictional community workshop. It is one local [program](#program-build-and-process). Front-desk tools send it [requests](#requests) on behalf of the person who acts; the program decides, records and answers.

- **[Members](#registry)** join on a [membership tier](#memberships) and pay a monthly fee that depends on the tier. Staff can suspend a member; a suspended member is not in [good standing](#good-standing).
- **Training:** [courses](#courses-and-enrollments) certify members for a tool family, such as laser cutters.
- **Equipment:** [machines](#catalog) belong to a tool family and have a rate per minute. Staff schedule [maintenance windows](#maintenance) during which a machine cannot be reserved.
- **Bookings:** members [reserve machines](#reservations) in 30-minute steps inside opening hours, check in when they start and check out when they finish. They are charged for every [metered minute](#check-in-check-out-and-metering) at the machine's rate, minus a tier discount.
- **Materials:** a small [shop](#materials) sells materials from stock.
- **Billing:** once a month staff generate [invoices](#invoices): one per member, with the monthly fee and all charges of that month. An invoice is due 14 days after it is issued; an invoice more than 30 days overdue blocks reservations and purchases until it is paid. Members [pay their invoices](#payments-and-credit); an overpayment becomes credit, and credit reduces the next invoice.
- **Safety:** staff record safety [incidents](#safety) per machine and can [lock a machine](#lockouts) after an incident.
- **Reporting:** staff and admins read utilization and revenue [reports](#reporting).

Four policies apply to every operation: [authorization](#authorization), [audit](#audit), [money](#money) and [time](#time).

Cross-area rules at a glance:

- A [reservation](#reservation-rules) needs a member in good standing who is not blocked, a certification for the machine's family that is valid on the reservation date, a machine without maintenance in the reserved slot, no lockout of the machine overlapping the slot, a free machine, and a slot inside opening hours that follows the booking granularity.
- Machine use produces a usage charge in billing, an audit entry and report data. Course enrollments and material purchases also produce charges.

Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, notifications, scheduled jobs, deletion of records and integration with other systems.

## Program interface

### Program, build and process

- The Go module lives at `realization/go.mod`. The program's main package lives at `realization/cmd/makerspace`. From `realization/`, `go build ./cmd/makerspace` builds the program and `go test ./...` runs its tests. Only the Go standard library is used. Internal package layout and design are the implementer's choice.
- The program reads [requests](#requests) from standard input, one JSON object per line, and writes exactly one [response](#responses) line per request to standard output, in request order. It writes and flushes each response before it reads the next request. Standard output carries responses only: no banner, prompt or log line. Diagnostics, if any, go to standard error. When standard input closes, the program exits with exit code zero.
- One process holds one [makerspace](#overview), which starts in the [initial state](#initial-state-and-generated-identifiers). Requests are processed one at a time, in order.
- The program never reads the machine clock, the network, environment-dependent data or files at run time, and writes no files. All time comes from the requests. Given the same request lines, it produces the same response lines.
- There is no reset, state import, clock override or other test-only operation.
- Request lines up to 64 KiB must be accepted.

### Requests

A request is a UTF-8 JSON object with exactly three members:

```json
{"id":"q1","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"Ada Byron","tier":"plus"}}
```

- `id`: a nonempty string chosen by the caller. It is echoed in the response. It is only a correlation value, not a domain identifier and not an idempotency key; two requests with the same `id` are two separate requests.
- `op`: the name of one operation of this document. Names are case-sensitive.
- `args`: a JSON object with the operation's arguments. Every operation takes `actor` (the identifier of the acting [user](#registry)) and `at` (the [request time](#time)) in addition to its own arguments. All arguments are required; no operation has optional arguments.

### Responses

Every response is one JSON object on one line, with exactly the members shown for its status.

Success:

```json
{"id":"q1","status":"ok","result":{"member":{"memberId":"ada","name":"Ada Byron","tier":"plus","standing":"good","registeredAt":"2026-03-02T09:00Z"}}}
```

Refusal of a well-formed request ([processing a request](#order-of-checks)):

```json
{"id":"q2","status":"refused","code":"already_exists","reason":"A user with identifier ada already exists."}
```

`code` is exactly one of the [refusal codes](#refusal-codes). `reason` is a nonempty, understandable explanation; its wording is free.

Invalid request ([invalid requests](#invalid-requests)):

```json
{"id":null,"status":"invalidRequest","reason":"The line is not valid JSON."}
```

### Invalid requests

The response is `status: "invalidRequest"` with a nonempty `reason` when:

- the line is not valid JSON, is empty, or is not a JSON object;
- `id` is missing, is not a string or is empty;
- `op` is missing, is not a string or names no operation of this document;
- `args` is missing or is not a JSON object;
- the request object has a member other than `id`, `op` and `args`;
- `args` has a member that is not an argument of the named operation, including a member whose name differs from a stated argument only in letter case.

The response's `id` is the request's `id` when the line is a JSON object whose `id` member is a nonempty string; otherwise it is `null`. A last line that ends without a line feed is still a request and gets a response. Behavior for JSON objects with duplicate member names and for lines that are not valid UTF-8 is not specified and is not tested. An invalid request changes nothing: no state, no [audit entry](#audit), no [identifier counter](#initial-state-and-generated-identifiers) and no request-time order ([time](#time)).

A request whose `args` lack a stated argument, or carry an argument with a wrong value, is not invalid in this sense. It is refused with code `invalid_argument` ([processing a request](#order-of-checks)).

### Argument values

| Value type | Rule |
| --- | --- |
| identifier | A JSON string of 1 to 64 characters from `A`–`Z`, `a`–`z`, `0`–`9`, `.`, `_` and `-`, whose first character is a letter or digit. Identifiers are compared exactly and are case-sensitive: `Ada` and `ada` are different identifiers. |
| text | A JSON string that contains at least one character without the Unicode `White_Space` property. It is stored and returned exactly as supplied, without trimming. |
| integer | A JSON number written as an optional minus sign followed by digits, without a fraction part or exponent (`10.0` and `1e3` are not integers), within the range the operation states. Every range in this document starts at 0 or higher, so any value written with a minus sign, including `-0`, is out of range. |
| timestamp | A JSON string `YYYY-MM-DDTHH:MMZ` naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`, for example `2026-03-02T09:00Z`. Hours are `00` to `23`, minutes `00` to `59`; there are no seconds. |
| date | A JSON string `YYYY-MM-DD` naming a valid calendar date from `2000-01-01` to `2099-12-31`. `2026-02-29` is not valid; `2028-02-29` is. |
| month | A JSON string `YYYY-MM` from `2000-01` to `2099-12`. |
| enumeration | A JSON string equal to one of the listed values, compared exactly. |

An argument that is missing, `null`, of the wrong JSON type or outside its format or range is refused with `invalid_argument`. Each argument is checked on its own. Relationships between arguments, such as an end that is not after a start, and relationships to stored state are [checked later](#order-of-checks) and have their own codes.

These formats and ranges apply to arguments, including `at`. An integer argument outside its range is out of range however many digits it has. Values the program computes can lie outside these ranges; [computed values and limits](#computed-values-and-limits) states the rules for those values.

### Initial state and generated identifiers

A new process contains exactly one [user](#registry): identifier `admin`, name `Administrator`, role `admin`. There are no members, families, machines, courses, materials, reservations, invoices, incidents or audit entries.

The program generates the identifiers of these records:

| Record | Format |
| --- | --- |
| [Reservation](#reservations) | `res-1`, `res-2`, … |
| [Maintenance window](#maintenance) | `mnt-1`, `mnt-2`, … |
| [Purchase](#purchasematerial) | `pur-1`, `pur-2`, … |
| [Invoice](#invoices) | `inv-1`, `inv-2`, … |
| [Payment](#payments-and-credit) | `pay-1`, `pay-2`, … |
| [Incident](#safety) | `inc-1`, `inc-2`, … |
| [Lockout](#lockouts) | `lock-1`, `lock-2`, … |

Each kind has its own counter. The first record of a kind gets number 1 and each further record of that kind gets the next number, in decimal without leading zeros. Numbers are never reused. Refused and invalid requests consume no number. "Creation order" in this document means the order in which records of a kind were created, which is the order of their numbers.

### Records in responses

A `result` object has exactly the members its operation names, and every object inside it has exactly the members stated for it: records have the fields of the [record list](#records), and other objects, such as an opening-hours object, a utilization row or the members of a report, have the fields their operation lists. There are never fewer or additional members. Nullable fields are present with the value `null` when they have no value. Lists are JSON arrays; an empty list is `[]`, never `null`. Amounts, counts and minutes are JSON integers. The order of members inside a JSON object is free; the order of list elements is fixed by the [order of listed results](#order-of-listed-results).

### Computed values and limits

Request times and time arguments lie between 2000 and 2099 ([argument values](#argument-values)), but some values the program computes from them lie later. Integers grow with the number of requests. These rules cover every computed value.

- **Times after 2099.** A computed timestamp or date can lie after 2099. It uses the same format as its type, with a four-digit year, and the Gregorian [calendar](#time) continues as usual: 2100 is not a leap year, so `2099-03-01` plus 365 days is `2100-03-01`. The latest computed value a result can show is a `dueAt` of `2100-01-14T23:59Z` (14 days after `2099-12-31T23:59Z`). Points used only in comparisons can also lie after 2099: the first minute after the month `2099-12` and after the date `2099-12-31` are both `2100-01-01T00:00Z`. The [block](#blocked-member) compares the request time with `dueAt` plus 43200 minutes, a point that can lie as late as `2100-02-13T23:59Z`. No computed time lies before `2000-01-01T00:00Z`.
- **The end of the calendar.** No request time is later than `2099-12-31T23:59Z`. The following consequences follow from the rules of this document and are stated so that they are read only one way:
  - `2099-11` is the last month that can be [invoiced](#generateinvoices). `generateInvoices` for `2099-12` is never accepted: when it reaches the operation rules ([order of checks](#order-of-checks), check 7), it is refused with `month_not_ended`. The [revenue report](#revenuereport) for `2099-12` always has `invoicesGenerated` `false`.
  - An [invoice](#invoices) whose `dueAt` lies after 2099 is never overdue.
  - An invoice whose `dueAt` is `2099-12-01T23:59Z` or later can never block, because no request time is later than `2099-12-31T23:59Z`.
  - The last day a [utilization report](#utilizationreport) can include is `2099-12-30`, because `to` is at most `2099-12-31` and is excluded from the range.
  - A [maintenance window](#maintenance) ends at the latest at `2099-12-31T23:59Z`, so the last minute of 2099 cannot be covered. Like a maintenance window, a [lockout](#lockouts) ends at the latest at `2099-12-31T23:59Z`, cannot cover the last minute of 2099 and has no maximum length apart from the timestamp range. [Reservations](#reservations) end by closing time and are not affected.
- **Integers are exact and never negative.** Within the integer limit stated under "Large sums and counters" below, no computed integer is negative. [Stock](#materials) cannot fall below 0 (`insufficient_stock`). [Credit](#credit-balance) and [payments](#recordpayment) are applied at most up to an invoice's balance. [Metered minutes](#check-in-check-out-and-metering) are at least 0, because a check-out cannot precede its check-in in the request-time order. Maintenance minutes are part of the open minutes they are subtracted from. Lockout minutes, like maintenance minutes, are part of the open minutes they are subtracted from, so `availableMinutes` is never negative.
- **Single lines cannot overflow.** The argument ranges keep every line amount and the product inside its formula ([money](#money)) below 2^53. The largest is a [usage line](#checkout): metered minutes are fewer than 52,596,000 (from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`), and at 10000 cents per minute the product `minutes × 10000 × 100` stays below 5.26 × 10^13. A [material line](#purchasematerial)'s product is at most `10000 × 1000000 × 100` = 10^12.
- **Large sums and counters.** Sums and counters grow with the number of requests: invoice totals, balances, credit balances, stock, report sums and minute totals, `seq` and the numbers in generated identifiers. [Lockout credits](#lockmachine) are covered by the rule for large sums. The program keeps every integer exact up to 9007199254740991 (2^53 − 1); signed 64-bit integer arithmetic satisfies this. Once a stored or returned integer, or an intermediate value of a formula in this document, would exceed 2^53 − 1, the rest of the run is not specified and is not tested. Reaching that limit takes at least tens of thousands of requests.
- **Generated identifiers stay valid identifiers.** A [generated identifier](#initial-state-and-generated-identifiers) is its prefix followed by a number. Within the integer limit it has at most 21 characters (a four-character prefix, or the five-character prefix of a [lockout](#lockouts) identifier, and at most 16 digits), so it always meets the identifier format of the [argument values](#argument-values) and can be passed back as an argument.
- **Audit targets are not identifiers.** An [audit](#audit-target-list) `target` is a plain string, or `null` in a refusal entry. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format.
- **Durations.** A reservation lasts at most 240 minutes. Maintenance windows and metered sessions have no maximum length apart from the timestamp range.
- **Request size and depth.** The 64 KiB of the [program rules](#program-build-and-process) are 65,536 bytes, not counting the line feed. Longer lines, and lines with any JSON value nested more than 100 levels deep in arrays or objects, are not specified and are not tested.

## Processing a request

### Order of checks

A well-formed request is checked in the following order. The first check that fails decides the refusal and its code.

1. **Arguments:** every argument is present and valid by the [argument value rules](#argument-values) → otherwise `invalid_argument`.
2. **Actor:** `actor` identifies an existing [user](#registry) of any role → otherwise `unknown_actor`.
3. **Request time:** `at` is not earlier than the request time of the most recent request that was answered `ok` ([time](#time)) → otherwise `time_regression`.
4. **Role permission:** the actor's role may use the operation at all, as "all" or "own" in the [permission matrix](#permission-matrix) → otherwise `forbidden`.
5. **Referenced records:** every argument that names an existing record names one that exists, checked in the order the operation lists its arguments → otherwise `not_found`. An argument that names a member must name a user whose role is `member`; the identifier of a staff or admin user does not name a member.
6. **Ownership:** when the actor's permission for the operation is "own", the record concerned belongs to the actor ([authorization](#authorization)) → otherwise `forbidden`.
7. **Operation rules:** the refusals listed for the operation, in the order listed.

When a check fails, the response is `status: "refused"` with the code and a reason, and nothing else happens: no record is created or changed, no identifier number is consumed, no audit entry is written, except that a `forbidden` refusal appends its [audit entry](#audit), and the request-time order is not advanced. When every check passes, all of the operation's effects take place together and the response is `status: "ok"`.

### Refusal codes

| Code | Meaning |
| --- | --- |
| `invalid_argument` | An argument is missing, `null`, of the wrong type, or outside its format or range. |
| `unknown_actor` | `actor` does not identify a user. |
| `time_regression` | `at` is earlier than the request time of the latest request answered `ok`. |
| `forbidden` | The actor's role may not use the operation, or may use it only for the actor's own records and the record concerned belongs to another member. |
| `not_found` | A referenced record does not exist. |
| `already_exists` | A record with the supplied identifier already exists. |
| `invalid_state` | The record is not in a state that allows the operation. |
| `invalid_interval` | An interval's end is not after its start, or a report range is longer than allowed. |
| `starts_in_past` | An interval starts before the request time. |
| `invalid_slot` | A reservation does not follow the booking granularity or its duration limits. |
| `outside_opening_hours` | A reservation is not inside the opening hours of its day. |
| `member_suspended` | The member is not in good standing. |
| `account_overdue` | The member has an invoice that is more than 30 days overdue. |
| `not_certified` | The member has no certification for the machine's family that is not revoked, or there is no certification to revoke. |
| `already_enrolled` | The member is already enrolled in the course. |
| `course_full` | The course has no free place. |
| `not_enrolled` | The member is not enrolled in the course. |
| `machine_in_maintenance` | A maintenance window of the machine overlaps the reservation. |
| `machine_locked` | A lockout of the machine overlaps the reservation. |
| `machine_booked` | Another active reservation of the machine overlaps the reservation. |
| `member_double_booked` | Another active reservation of the member overlaps the reservation. |
| `reservation_conflict` | An active reservation overlaps the maintenance window. |
| `check_in_too_early` | The reservation has not started yet. |
| `insufficient_stock` | The material's stock is lower than the requested quantity. |
| `month_not_ended` | The month to invoice has not ended at the request time. |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |

## Cross-cutting policies

### Authorization

Every [user](#registry) has exactly one role, fixed when the user is registered: `member`, `staff`, `admin` or `steward`. Members are the workshop's customers. Staff run the front desk and the workshop floor. Admins also manage the catalog, credits and users. Stewards look after the machines and safety on the workshop floor: a steward handles maintenance and incidents and has no access to billing.

For each operation and role, the [permission matrix](#permission-matrix) states one permission:

- **all:** the role may use the operation for any record.
- **own:** the role may use the operation only for records that belong to the actor.
- **none:** the role may not use the operation.

A record concerned belongs to the actor when:

- the operation has a `memberId` argument and it equals the actor's identifier;
- the operation has a `reservationId` argument and the reservation's member is the actor;
- the operation has an `invoiceId` argument and the invoice's member is the actor.

A "none" permission is refused at check 4, an ownership violation at check 6 ([order of checks](#order-of-checks)). Both use the code `forbidden`.

#### Permission matrix

This table is the permission matrix ([authorization](#authorization)). It states, for each operation, its area and the permission of each role.

For every operation without "all" in the Steward column a steward's permission is "none", and the request is refused with `forbidden` at check 4 ([order of checks](#order-of-checks)). In particular a steward may not use any billing operation (`generateInvoices`, `listInvoices`, `recordPayment`, `issueCredit`, `getAccount`) or `revenueReport`, and also not `utilizationReport`, `listAuditEntries`, `registerUser`, any member, training, booking or materials operation without "all" in that column, or the catalog operations `addFamily`, `addMachine` and `setMachineRate`.

| Operation | Area | Member | Staff | Admin | Steward |
| --- | --- | --- | --- | --- | --- |
| [registerUser](#registeruser) | Members | none | none | all | none |
| [registerMember](#registermember) | Members | none | all | all | none |
| [getMember](#getmember) | Members | own | all | all | none |
| [changeTier](#changetier) | Members | none | all | all | none |
| [setMemberStanding](#setmemberstanding) | Members | none | all | all | none |
| [addCourse](#addcourse) | Training | none | all | all | none |
| [enrollInCourse](#enrollincourse) | Training | own | all | all | none |
| [completeCourse](#completecourse) | Training | none | all | all | none |
| [revokeCertification](#revokecertification) | Training | none | all | all | none |
| [listCertifications](#listcertifications) | Training | own | all | all | none |
| [addFamily](#addfamily) | Equipment | none | none | all | none |
| [addMachine](#addmachine) | Equipment | none | none | all | none |
| [setMachineRate](#setmachinerate) | Equipment | none | none | all | none |
| [listMachines](#listmachines) | Equipment | all | all | all | all |
| [scheduleMaintenance](#schedulemaintenance) | Equipment | none | all | all | all |
| [cancelMaintenance](#cancelmaintenance) | Equipment | none | all | all | all |
| [createReservation](#createreservation) | Bookings | own | all | all | none |
| [cancelReservation](#cancelreservation) | Bookings | own | all | all | none |
| [checkIn](#checkin) | Bookings | own | all | all | none |
| [checkOut](#checkout) | Bookings | own | all | all | none |
| [listMemberReservations](#listmemberreservations) | Bookings | own | all | all | none |
| [getMachineSchedule](#getmachineschedule) | Bookings | all | all | all | all |
| [addMaterial](#addmaterial) | Materials | none | all | all | none |
| [restockMaterial](#restockmaterial) | Materials | none | all | all | none |
| [purchaseMaterial](#purchasematerial) | Materials | own | all | all | none |
| [listMaterials](#listmaterials) | Materials | all | all | all | all |
| [generateInvoices](#generateinvoices) | Billing | none | all | all | none |
| [listInvoices](#listinvoices) | Billing | own | all | all | none |
| [recordPayment](#recordpayment) | Billing | own | all | all | none |
| [issueCredit](#issuecredit) | Billing | none | none | all | none |
| [getAccount](#getaccount) | Billing | own | all | all | none |
| [reportIncident](#reportincident) | Safety | none | all | all | all |
| [resolveIncident](#resolveincident) | Safety | none | all | all | all |
| [listIncidents](#listincidents) | Safety | none | all | all | all |
| [lockMachine](#lockmachine) | Safety | none | all | all | all |
| [utilizationReport](#utilizationreport) | Reporting | none | all | all | none |
| [revenueReport](#revenuereport) | Reporting | none | all | all | none |
| [listAuditEntries](#listauditentries) | Audit | none | none | all | none |

### Audit

Operations are either mutations or reads, as the [audit target list](#audit-target-list) states. Every request to a mutation answered `ok` appends exactly one entry with outcome `accepted`, even when the operation changes no data, such as setting a member's standing to its current value, and every request refused with `forbidden` appends exactly one entry with outcome `refused`. Reads answered `ok`, other refusals and invalid requests append nothing. Entries are never changed or removed. A steward's accepted mutations and `forbidden` refusals are recorded like everyone's, with `actorRole` `steward`.

Every request refused with `forbidden` is recorded. This holds for every operation, mutations and reads alike, and for both kinds of `forbidden`: a role without permission (check 4 of [the order of checks](#order-of-checks)) and an ownership violation (check 6).

An audit entry has these fields:

- `seq`: the entry's position in the log, starting at 1 and increasing by 1 for each entry;
- `at`: the request time;
- `actor`: the acting user's identifier;
- `actorRole`: the role of the acting user, `member`, `staff`, `admin` or `steward`;
- `op`: the operation name;
- `target`: a string that identifies the record the operation acted on, as stated per operation in the [audit target list](#audit-target-list); `null` in refusal entries;
- `outcome`: `accepted` for an entry of a mutation answered `ok`, `refused` for an entry of a `forbidden` refusal.

**Content of a refusal entry:** `seq` is the next number of the one shared sequence; `at` is the request time; `actor` is the acting user; `actorRole` is that user's role; `op` is the requested operation; `target` is `null`; `outcome` is `refused`. **Content of an accepted entry:** as stated for each field above, with `outcome` `accepted`.

The response to such a request is still the `forbidden` refusal. Apart from the audit entry, a `forbidden` refusal changes nothing: no other record, no identifier counter, and the request-time order is still set only by requests answered `ok` ([time](#time)). A refusal entry can therefore carry a later `at` than the accepted entry that follows it; `listAuditEntries` still returns entries in `seq` order.

Not recorded are refusals with any other code, including `invalid_argument`, `unknown_actor`, `time_regression`, `not_found` and every operation rule code; invalid requests; and a request that fails an earlier check than the permission checks. The order of checks does not change, so, for example, a request from a member with an earlier `at` than the last accepted request is refused with `time_regression` and is not recorded, even though the member also lacks permission.

Example: member `ada` sends `listAuditEntries` at `2026-03-05T10:00Z`. The response is a `forbidden` refusal, and the log gains `{"seq": 7, "at": "2026-03-05T10:00Z", "actor": "ada", "actorRole": "member", "op": "listAuditEntries", "target": null, "outcome": "refused"}` (with `seq` the next number).

Admins [read the log](#listauditentries) with `listAuditEntries`.

#### Audit target list

This table is the audit target list ([audit](#audit)). "Mutation" operations write an audit entry when answered `ok`; "read" operations never do.

| Operation | Kind | Audit `target` |
| --- | --- | --- |
| [registerUser](#registeruser) | mutation | `userId` |
| [registerMember](#registermember) | mutation | `memberId` |
| [getMember](#getmember) | read | — |
| [changeTier](#changetier) | mutation | `memberId` |
| [setMemberStanding](#setmemberstanding) | mutation | `memberId` |
| [addCourse](#addcourse) | mutation | `courseId` |
| [enrollInCourse](#enrollincourse) | mutation | `courseId/memberId` |
| [completeCourse](#completecourse) | mutation | `courseId/memberId` |
| [revokeCertification](#revokecertification) | mutation | `memberId/familyId` |
| [listCertifications](#listcertifications) | read | — |
| [addFamily](#addfamily) | mutation | `familyId` |
| [addMachine](#addmachine) | mutation | `machineId` |
| [setMachineRate](#setmachinerate) | mutation | `machineId` |
| [listMachines](#listmachines) | read | — |
| [scheduleMaintenance](#schedulemaintenance) | mutation | new `windowId` |
| [cancelMaintenance](#cancelmaintenance) | mutation | `windowId` |
| [createReservation](#createreservation) | mutation | new `reservationId` |
| [cancelReservation](#cancelreservation) | mutation | `reservationId` |
| [checkIn](#checkin) | mutation | `reservationId` |
| [checkOut](#checkout) | mutation | `reservationId` |
| [listMemberReservations](#listmemberreservations) | read | — |
| [getMachineSchedule](#getmachineschedule) | read | — |
| [addMaterial](#addmaterial) | mutation | `materialId` |
| [restockMaterial](#restockmaterial) | mutation | `materialId` |
| [purchaseMaterial](#purchasematerial) | mutation | new `purchaseId` |
| [listMaterials](#listmaterials) | read | — |
| [generateInvoices](#generateinvoices) | mutation | `month` |
| [listInvoices](#listinvoices) | read | — |
| [recordPayment](#recordpayment) | mutation | new `paymentId` |
| [issueCredit](#issuecredit) | mutation | `memberId` |
| [getAccount](#getaccount) | read | — |
| [reportIncident](#reportincident) | mutation | new `incidentId` |
| [resolveIncident](#resolveincident) | mutation | `incidentId` |
| [listIncidents](#listincidents) | read | — |
| [lockMachine](#lockmachine) | mutation | new `lockoutId` |
| [utilizationReport](#utilizationreport) | read | — |
| [revenueReport](#revenuereport) | read | — |
| [listAuditEntries](#listauditentries) | read | — |

In the target column, a name is the value of that argument or new identifier; `a/b` joins two values with a slash, for example `laser-101/ada` for `enrollInCourse` with course `laser-101` and member `ada`; `month` is the month string, for example `2026-03`.

#### `listAuditEntries`

Operation `listAuditEntries`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: none besides `actor` and `at`. Result: `entries`, all [audit entries](#audit) in `seq` order.

### Money

- All amounts are integer numbers of cents, represented as JSON integers. There are no fractions of a cent, no floating-point amounts and no currency field. Stored and returned amounts are never negative.
- Every [charge](#invoices) is a line with a `quantity`, a `unitPriceCents` and a `discountPercent`. Its amount is
  `amountCents = (quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`,
  where `div` is integer division. This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again.
  Example: 37 minutes at 45 cents with a 10 % discount: `(37 × 45 × 90 + 50) div 100 = 149900 div 100 = 1499` cents.
- A charge's amount is fixed when the charge is created. Later changes to a rate, a tier or a price never change existing charges.

#### Credit balance

Each member has a credit balance, initially 0. It increases by [credits issued by an admin](#issuecredit), by [lockout credits](#lockmachine) and by [overpayments](#recordpayment). It decreases only when credit is applied to a new invoice ([charges and invoices](#invoices)). It is never negative. Credit is never paid out.

### Time

- All times are UTC with a precision of one minute. Timestamps, dates and months use the formats of the [argument values](#argument-values). The day of the week of a timestamp is that of its UTC date. The date of a timestamp is its UTC calendar date.
- **Request time:** `at` is the time at which the request happens. It is the program's only notion of "now". The request time of every request must not be earlier than the request time of the most recent request answered `ok`; an equal time is allowed. Refused and invalid requests do not move this point. Before the first `ok` response, any valid `at` is accepted.
- **Intervals are half-open.** An interval `[start, end)` contains its start minute and excludes its end minute. Two intervals `[a, b)` and `[c, d)` overlap exactly when `a < d` and `c < b`; intervals that only touch, such as 10:00–11:00 and 11:00–12:00, do not overlap. The length of an interval in minutes is `end − start`.
- **Day and date arithmetic:** a day is 1440 minutes; "N days after timestamp t" is `t + N × 1440` minutes. "N days after date d" is the calendar date N days later, so `2027-03-01` plus 365 days is `2028-02-29`.
- **States that depend on time** are evaluated at the request time of the request that reads or uses them: a reservation that is `missed` ([reservations](#reservations)), an invoice that is `overdue` ([charges and invoices](#invoices)) and a member who is blocked ([blocked member](#blocked-member)).

#### Opening hours

**Opening hours** are the same every week:

| Day | Opens | Closes | Open minutes |
| --- | --- | --- | --- |
| Monday to Friday | 09:00 | 21:00 | 720 |
| Saturday | 10:00 | 18:00 | 480 |
| Sunday | closed | closed | 0 |

The opening period of a day is the interval `[opens, closes)` on that date. A [reservation](#reservations) is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Sunday has no opening period.

#### Booking granularity

**Booking granularity is 30 minutes.** A [reservation](#reservations)'s start and end must each fall on a minute that is a multiple of 30 within the hour (`:00` or `:30`). A reservation lasts at least one granularity step (30 minutes) and at most 240 minutes.

### Order of listed results

Identifiers are ordered by ASCII byte value (so `-` < `.` < digits < uppercase letters < `_` < lowercase letters, and `M10` < `M9`).

| List | Order |
| --- | --- |
| [listCertifications](#listcertifications) → `certifications` | `familyId` ascending |
| [listMachines](#listmachines) → `machines` | `machineId` ascending |
| [getMachineSchedule](#getmachineschedule) → `reservations` | `start` ascending, then creation order |
| [getMachineSchedule](#getmachineschedule) → `maintenance` | `start` ascending, then creation order |
| [getMachineSchedule](#getmachineschedule) → `lockouts` | `start` ascending, then creation order |
| [listMemberReservations](#listmemberreservations) → `reservations` | `start` ascending, then creation order |
| [listMaterials](#listmaterials) → `materials` | `materialId` ascending |
| [generateInvoices](#generateinvoices) → `invoices` | `memberId` ascending |
| [listInvoices](#listinvoices) → `invoices` | `month` ascending |
| [invoice](#invoices) → `lines` | the membership line first, then the other lines by their `at` ascending, then in the order the charges were created |
| [listIncidents](#listincidents) → `incidents` | creation order |
| [lockMachine](#lockmachine) → `cancelledReservations` and a lockout's `cancelledReservationIds` | creation order |
| [utilizationReport](#utilizationreport) → `machines` | `machineId` ascending |
| [listAuditEntries](#listauditentries) → `entries` | `seq` ascending |

## Records

Every record in a result has exactly these fields.

| Record | Fields |
| --- | --- |
| [user](#registry) | `userId`, `name`, `role` |
| [member](#memberships) | `memberId`, `name`, `tier`, `standing` (`good` or `suspended`), `registeredAt` (timestamp) |
| [family](#catalog) | `familyId`, `name` |
| [course](#courses-and-enrollments) | `courseId`, `familyId`, `title`, `feeCents`, `capacity`, `enrolledCount` |
| [enrollment](#courses-and-enrollments) | `courseId`, `memberId`, `status` (`enrolled` or `completed`), `enrolledAt`, `completedAt` (`null` while enrolled) |
| [certification](#certifications) | `memberId`, `familyId`, `issuedOn` (date), `status` (`valid` or `revoked`) |
| [machine](#catalog) | `machineId`, `name`, `familyId`, `rateCentsPerMinute` |
| [maintenance window](#maintenance) | `windowId`, `machineId`, `start`, `end`, `reason`, `status` (`scheduled` or `cancelled`) |
| [reservation](#reservations) | `reservationId`, `memberId`, `machineId`, `start`, `end`, `status` (`reserved`, `missed`, `checkedIn`, `completed` or `cancelled`), `checkedInAt`, `checkedOutAt`, `rateCentsPerMinute`, `meteredMinutes`, `amountCents` |
| [charge](#invoices) (invoice line) | `kind` (`membership`, `usage`, `material` or `course`), `ref`, `quantity`, `unitPriceCents`, `discountPercent`, `amountCents`, `at` |
| [material](#materials) | `materialId`, `name`, `unitPriceCents`, `stock` |
| [purchase](#purchasematerial) | `purchaseId`, `memberId`, `materialId`, `quantity`, `unitPriceCents`, `discountPercent`, `amountCents`, `at` |
| [invoice](#invoices) | `invoiceId`, `memberId`, `month`, `issuedAt`, `dueAt`, `lines` (charges), `totalCents`, `creditAppliedCents`, `paidCents`, `balanceCents`, `status` (`open` or `paid`), `overdue` (boolean) |
| [payment](#payments-and-credit) | `paymentId`, `invoiceId`, `memberId`, `amountCents`, `appliedCents`, `creditedCents`, `at` |
| [account](#getaccount) | `memberId`, `creditBalanceCents`, `openBalanceCents`, `overdueBalanceCents`, `unbilledCents`, `blocked` |
| [incident](#safety) | `incidentId`, `machineId`, `severity` (`low`, `medium` or `high`), `description`, `reportedBy`, `reportedAt`, `status` (`open` or `resolved`), `resolution` (`null` while open), `resolvedAt` (`null` while open) |
| [lockout](#lockouts) | `lockoutId`, `incidentId`, `machineId`, `start`, `end`, `cancelledReservationIds` (the reservations this lockout cancelled, in creation order) |
| [audit entry](#audit) | `seq`, `at`, `actor`, `actorRole`, `op`, `target` (`null` in refusal entries), `outcome` (`accepted` or `refused`) |

In a reservation, `checkedInAt` and `rateCentsPerMinute` are `null` until check-in; `checkedOutAt`, `meteredMinutes` and `amountCents` are `null` until check-out. The meaning of each field is given with the operations that set it.

## Members

### Registry

The registry holds every user: the [initial admin](#initial-state-and-generated-identifiers), staff, admins and stewards [registered by an admin](#registeruser), and members [registered by staff or admins](#registermember). Users, including members, share one identifier space. Users are never deleted and their identifier, name and [role](#authorization) never change. A steward is a user, not a member: a steward's identifier does not name a member, and a steward has no membership, invoices or credit.

#### `registerUser`

Operation `registerUser`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Registers a staff member, an admin or a steward.

- Arguments: `userId` (identifier), `name` (text), `role` (enumeration `staff`, `admin` or `steward`).
- Refusals: `already_exists` when any user, including a member, has the identifier `userId`.
- Effect: creates the [user](#registry).
- Result: `user`.

#### `registerMember`

Operation `registerMember`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Registers a member with a [membership](#memberships).

- Arguments: `memberId` (identifier), `name` (text), `tier` (enumeration `basic`, `plus` or `pro`).
- Refusals: `already_exists` when any user has the identifier `memberId`.
- Effect: creates a user with role `member`, the given tier, standing `good`, `registeredAt` equal to the [request time](#time) and a [credit balance](#credit-balance) of 0.
- Result: `member`.

#### `getMember`

Operation `getMember`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: `memberId` (identifier, an existing [member](#memberships)). Result: `member`.

### Memberships

A membership has a tier and a [standing](#good-standing). The tier sets the monthly fee and the discount on machine usage:

| Tier | Monthly fee | Usage discount |
| --- | --- | --- |
| `basic` | 2500 cents | 0 % |
| `plus` | 4900 cents | 10 % |
| `pro` | 9900 cents | 25 % |

The fee is charged once for every invoiced month ([charges and invoices](#invoices)), without proration. The usage discount applies only to [usage charges](#checkout), never to course or material charges.

#### Good standing

A member is **in good standing** when the standing is `good`. A suspended member cannot [make reservations](#createreservation), [check in](#checkin), [buy materials](#purchasematerial) or [enroll in courses](#enrollincourse). Suspension does not cancel existing reservations or enrollments. A suspended member can still [cancel reservations](#cancelreservation), [check out](#checkout), see their records and [pay invoices](#recordpayment), and still receives the monthly fee. Reinstating good standing removes these restrictions; all other rules still apply.

#### `changeTier`

Operation `changeTier`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Changes a member's tier.

- Arguments: `memberId` (identifier, an existing member), `tier` (enumeration `basic`, `plus` or `pro`).
- Effect: sets the [tier](#memberships). Setting the current tier succeeds without a change. The new tier applies to membership fees of [invoices generated afterwards](#generateinvoices) and to usage charges created by [later check-outs](#checkout).
- Result: `member`.

#### `setMemberStanding`

Operation `setMemberStanding`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Suspends or reinstates a member.

- Arguments: `memberId` (identifier, an existing member), `standing` (enumeration `good` or `suspended`).
- Effect: sets the [standing](#good-standing). Setting the current standing succeeds without a change.
- Result: `member`.

## Training

### Courses and enrollments

A course teaches the safe use of one [tool family](#catalog). It has a fee and a capacity. A member [enrolls](#enrollincourse); staff [record completion](#completecourse), which [certifies](#certifications) the member for the course's family. There is one enrollment record per course and member. It is created by the first enrollment; a later enrollment after completion reuses and resets it.

The course's `enrolledCount` is the number of its enrollments with status `enrolled`. A place becomes free when an enrollment is completed.

#### `addCourse`

Operation `addCourse`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `courseId` (identifier), `familyId` (identifier, an existing [family](#catalog)), `title` (text), `feeCents` (integer 0 to 1000000), `capacity` (integer 1 to 1000).
- Refusals: `already_exists` when a course with `courseId` exists.
- Effect: creates the [course](#courses-and-enrollments) with `enrolledCount` 0.
- Result: `course`.

#### `enrollInCourse`

Operation `enrollInCourse`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Enrolls a member in a [course](#courses-and-enrollments).

- Arguments: `courseId` (identifier, an existing course), `memberId` (identifier, an existing member).
- Refusals, in this order: `member_suspended` when the member is not in [good standing](#good-standing); `already_enrolled` when the member's enrollment in this course has status `enrolled`; `course_full` when `enrolledCount` equals `capacity`.
- Effect: the enrollment gets status `enrolled`, `enrolledAt` equal to the request time and `completedAt` `null`. A course charge is created for the member ([charges and invoices](#invoices)): `kind` `course`, `ref` the course identifier, `quantity` 1, `unitPriceCents` the course fee, `discountPercent` 0, `at` the request time. A course with fee 0 creates a charge of 0 cents.
- Result: `enrollment` and `course` (after the change).

#### `completeCourse`

Operation `completeCourse`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Records that a member completed a [course](#courses-and-enrollments) and certifies the member.

- Arguments: `courseId` (identifier, an existing course), `memberId` (identifier, an existing member).
- Refusals: `not_enrolled` when the member has no enrollment in this course with status `enrolled`.
- Effect: the enrollment gets status `completed` and `completedAt` equal to the request time. The member's certification for the course's family is issued ([certifications](#certifications)). Standing is not checked.
- Result: `enrollment` and `certification`.

### Certifications

A member has at most one certification per tool family. Certifications have no expiry: a certification stays valid from its issue until it is revoked. [Completing a course](#completecourse) issues it: `issuedOn` is the date of the request time. If the member already has a certification for that family, revoked or not, it is replaced by a new valid one.

A certification is **valid on date D** when it is not revoked and `issuedOn ≤ D`. Reservations and check-ins always lie on or after the issue date, so in practice the reservation rule and the check-in rule only require a certification for the machine's family that is not revoked. Its reported `status` is `revoked` when revoked, otherwise `valid`.

#### `revokeCertification`

Operation `revokeCertification`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Withdraws a member's [certification](#certifications), for example after a safety incident.

- Arguments: `memberId` (identifier, an existing member), `familyId` (identifier, an existing family).
- Refusals: `not_certified` when the member has no certification for the family or it is already revoked.
- Effect: the certification becomes revoked; its dates do not change. Existing reservations stay, but check-in requires a valid certification ([check-in](#checkin)).
- Result: `certification`.

#### `listCertifications`

Operation `listCertifications`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: `memberId` (identifier, an existing member). Result: `memberId` and `certifications`, all of the member's [certifications](#certifications), including revoked ones.

## Equipment

### Catalog

The catalog holds tool families and machines. Each machine belongs to one family and has a rate in cents per minute. Families and machines are never removed.

#### `addFamily`

Operation `addFamily`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `familyId` (identifier), `name` (text).
- Refusals: `already_exists` when a [family](#catalog) with `familyId` exists.
- Result: `family`.

#### `addMachine`

Operation `addMachine`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `machineId` (identifier), `name` (text), `familyId` (identifier, an existing family), `rateCentsPerMinute` (integer 1 to 10000).
- Refusals: `already_exists` when a [machine](#catalog) with `machineId` exists.
- Result: `machine`.

#### `setMachineRate`

Operation `setMachineRate`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `machineId` (identifier, an existing machine), `rateCentsPerMinute` (integer 1 to 10000).
- Effect: sets the [rate](#catalog). The rate is captured at [check-in](#checkin), so the new rate applies to check-ins after the change. Reservations already checked in keep their captured rate, and existing charges do not change.
- Result: `machine`.

#### `listMachines`

Operation `listMachines`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: none besides `actor` and `at`. Result: `machines`, all [machines](#catalog).

### Maintenance

A maintenance window takes a [machine](#catalog) out of service for an interval `[start, end)`. While a scheduled window overlaps a slot, the machine cannot be [reserved](#reservation-rules) for it, and since every use starts with a reservation and its check-in, the machine cannot be used in the window. The one exception is a session that is already checked in and runs past its reservation's end: it is not interrupted, keeps metering until check-out and counts in `usedMinutes` ([utilization report](#utilizationreport)). Windows may use any minute, are not limited to opening hours and may overlap each other.

#### `scheduleMaintenance`

Operation `scheduleMaintenance`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `machineId` (identifier, an existing machine), `start` (timestamp), `end` (timestamp), `reason` (text).
- Refusals, in this order: `invalid_interval` when `end` is not after `start`; `starts_in_past` when `start` is before the request time; `reservation_conflict` when the window [overlaps](#time) an active reservation of the machine (status `reserved` or `checkedIn`, [reservations](#reservations)).
- Effect: creates the [window](#maintenance) with status `scheduled`.
- Result: `window`.

#### `cancelMaintenance`

Operation `cancelMaintenance`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `windowId` (identifier, an existing maintenance window).
- Refusals: `invalid_state` when the window is already cancelled or its start is at or before the request time.
- Effect: the [window](#maintenance) gets status `cancelled`. A cancelled window no longer blocks [reservations](#reservation-rules) and no longer counts in [schedules](#getmachineschedule) or [reports](#utilizationreport).
- Result: `window`.

## Bookings

### Reservations

A reservation books one machine for one member for an interval `[start, end)` on one day. Its stored status moves from `reserved` to `checkedIn` to `completed`, or from `reserved` to `cancelled`. Its reported status is the stored status, except that a stored `reserved` reservation whose `end` is at or before the request time is reported as `missed`. A missed reservation can no longer be checked in or cancelled and causes no charge.

A reservation is **active** when its reported status is `reserved` or `checkedIn`. Only active reservations block a machine or a member. Conflicts are judged on the reserved interval `[start, end)`, not on the actual check-in and check-out times.

#### Reservation rules

[A new reservation](#createreservation) is accepted only when all of the following hold. These rules connect members, training, equipment and time:

1. its slot follows the [booking granularity and duration limits](#booking-granularity) and starts at or after the [request time](#time);
2. the slot lies inside the [opening hours](#opening-hours) of its start date;
3. the member is in [good standing](#good-standing);
4. the member is not [blocked](#blocked-member);
5. the member holds a [certification](#certifications) for the machine's family that is valid on the reservation date, which is the date of `start`;
6. no scheduled [maintenance window](#maintenance) of the machine overlaps the slot;
7. no [lockout](#lockouts) of the machine overlaps the slot;
8. no other active [reservation](#reservations) of the machine overlaps the slot;
9. no other active reservation of the member, on any machine, overlaps the slot.

#### `createReservation`

Operation `createReservation`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `memberId` (identifier, an existing member), `machineId` (identifier, an existing machine), `start` (timestamp), `end` (timestamp).
- Refusals, in this order:
  - `invalid_interval` when `end` is not after `start`;
  - `invalid_slot` when `start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes;
  - `starts_in_past` when `start` is before the request time;
  - `outside_opening_hours` when the slot is not inside the opening hours of its start date;
  - `member_suspended` when the member is not in good standing;
  - `account_overdue` when the member is [blocked](#blocked-member);
  - `not_certified` when the member has no certification for the machine's family valid on the date of `start`;
  - `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot;
  - `machine_locked` when a [lockout](#lockouts) of the machine overlaps the slot;
  - `machine_booked` when another active reservation of the machine overlaps the slot;
  - `member_double_booked` when another active reservation of the member overlaps the slot.
- Effect: creates the [reservation](#reservations) with status `reserved`.
- Result: `reservation`.

Example: on Saturday `2026-03-07`, the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not; on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`.

#### `cancelReservation`

Operation `cancelReservation`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `reservationId` (identifier, an existing reservation).
- Refusals: `invalid_state` when the [reported status](#reservations) is not `reserved`.
- Effect: the reservation gets status `cancelled`. There is no fee and no credit.
- Result: `reservation`.

#### `listMemberReservations`

Operation `listMemberReservations`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: `memberId` (identifier, an existing member). Result: `memberId` and `reservations`, all of the member's [reservations](#reservations) in every status.

#### `getMachineSchedule`

Operation `getMachineSchedule`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Shows one machine's day.

- Arguments: `machineId` (identifier, an existing machine), `date` (date).
- Result: `machineId`; `date`; `openingHours`, an object `{"open": "HH:MM", "close": "HH:MM"}` with [that day's times](#opening-hours), or `null` on Sunday; `reservations`, the machine's [reservations](#reservations) whose `start` falls on that date, in every status except `cancelled`; `maintenance`, the machine's [scheduled windows](#maintenance) that overlap the interval from `00:00` of that date to `00:00` of the next date; `lockouts`, the machine's [lockouts](#lockouts) that overlap the interval from `00:00` of the date to `00:00` of the next date.

### Check-in, check-out and metering

The member [checks in](#checkin) when starting to use the machine and [checks out](#checkout) when done. The program meters the minutes in between and charges them at the rate captured at check-in, with the member's [tier discount](#memberships).

#### `checkIn`

Operation `checkIn`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `reservationId` (identifier, an existing reservation).
- Refusals, in this order: `invalid_state` when the [reported status](#reservations) is not `reserved` (so a missed, cancelled, checked-in or completed reservation is refused); `check_in_too_early` when the request time is before `start`; `member_suspended` when the member is not in [good standing](#good-standing); `not_certified` when the member has no [certification](#certifications) for the machine's family valid on the date of the request time.
- Effect: status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the [machine's current rate](#catalog).
- Result: `reservation`.

#### `checkOut`

Operation `checkOut`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `reservationId` (identifier, an existing reservation).
- Refusals: `invalid_state` when the status is not `checkedIn`.
- Effect: status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. A [usage charge](#invoices) is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the metered minutes, `unitPriceCents` the captured rate, `discountPercent` the [discount of the member's tier](#memberships) at check-out, `at` the request time. A charge of 0 cents is still created. The reservation's `amountCents` is the [charge's amount](#money). Standing is not checked.
- Result: `reservation` and `charge`.

## Materials

The shop sells materials by unit. Each material has a unit price and a stock count, which starts at 0. Stock never goes below 0.

### Bulk price tiers

Material [purchases](#purchasematerial) get a bulk discount that depends on the quantity of the purchase.

| Quantity of the purchase | `discountPercent` |
| --- | --- |
| 1 to 9 | 0 |
| 10 to 24 | 5 |
| 25 or more | 12 |

- The tier is chosen by the `quantity` of the single purchase. Quantities of separate purchases, also by the same member or of the same material, are never added together.
- The purchase's `amountCents` and the material charge are calculated with the formula of [the money policy](#money), rounded half up once per line. The material charge carries the same `discountPercent` and `amountCents` as the purchase.
- The tier table is the same for every material and every member. The member's membership tier plays no part in it.

The bulk discount is a materials rule only: the tier usage discount never applies to materials, and the bulk discount never applies to usage.

Examples, with the formula `(quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`:

| Purchase | Calculation | `amountCents` |
| --- | --- | --- |
| 9 units at 199 | `(9 × 199 × 100 + 50) div 100` | 1791 |
| 10 units at 199 | `(10 × 199 × 95 + 50) div 100` | 1891 |
| 25 units at 37 | `(25 × 37 × 88 + 50) div 100` | 814 |

### `addMaterial`

Operation `addMaterial`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `materialId` (identifier), `name` (text), `unitPriceCents` (integer 1 to 1000000).
- Refusals: `already_exists` when a material with `materialId` exists.
- Effect: creates the [material](#materials) with stock 0.
- Result: `material`.

### `restockMaterial`

Operation `restockMaterial`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `materialId` (identifier, an existing material), `quantity` (integer 1 to 100000).
- Effect: the [stock](#materials) increases by `quantity`.
- Result: `material`.

### `purchaseMaterial`

Operation `purchaseMaterial`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Sells material to a member.

- Arguments: `memberId` (identifier, an existing member), `materialId` (identifier, an existing material), `quantity` (integer 1 to 10000).
- Refusals, in this order: `member_suspended` when the member is not in [good standing](#good-standing); `account_overdue` when the member is [blocked](#blocked-member); `insufficient_stock` when the [stock](#materials) is lower than `quantity`.
- Effect: the stock decreases by `quantity`. A purchase is created with the material's current unit price, `discountPercent` from the [bulk discount table](#bulk-price-tiers), the amount by [the money policy](#money) and `at` the request time. A [material charge](#invoices) is created for the member: `kind` `material`, `ref` the purchase identifier, and the purchase's `quantity`, `unitPriceCents`, `discountPercent`, `amountCents` and `at`.
- Result: `purchase` and `material` (after the change).

### `listMaterials`

Operation `listMaterials`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: none besides `actor` and `at`. Result: `materials`, all [materials](#materials).

## Billing

### Invoices

[Usage](#checkout), [course](#enrollincourse) and [material](#purchasematerial) charges collect on the member's account until they are invoiced. A charge belongs to the month of its `at`. A charge is **unbilled** until it appears on an [invoice](#invoices).

Billing is organized around the invoice. An invoice collects one member's charges for one calendar month. It is due 14 days after it is issued, receives payments until it is settled, and passes any overpayment to the member's credit, which in turn settles part of the member's later invoices.

Invoices are generated per calendar month, for all members at once, when [staff request it](#generateinvoices). Generation for a month is possible once the month has ended at the request time, and only once per month. Months can be generated in any order.

For a month M, every member whose `registeredAt` is before the first minute of the month after M receives one invoice, whatever the member's standing. The invoice is built as follows:

1. `invoiceId` is the [next invoice identifier](#initial-state-and-generated-identifiers). Members are processed in ascending `memberId` order.
2. `issuedAt` is the request time; `dueAt` is [14 days after](#time) `issuedAt`.
3. `lines` starts with the membership line: `kind` `membership`, `ref` the member's current tier, `quantity` 1, `unitPriceCents` the [tier's monthly fee](#memberships), `discountPercent` 0, `at` the request time. All of the member's [unbilled charges](#invoices) whose `at` lies in month M follow, ordered as the [order of listed results](#order-of-listed-results) states. These charges are no longer unbilled.
4. `totalCents` is the sum of the line amounts.
5. `creditAppliedCents` is the smaller of the member's [credit balance](#credit-balance) and `totalCents`. The credit balance decreases by that amount.
6. `paidCents` is 0. `balanceCents` is `totalCents − creditAppliedCents − paidCents`.
7. `status` is `paid` when `balanceCents` is 0, otherwise `open`.

Because request times never go backwards and generation needs the month to have ended, no charge can be created for a month after that month was invoiced.

An invoice is **overdue** at a request time when its status is `open` and the request time is at or after `dueAt`. The `overdue` field reports this at the request time of the response. Apart from the [block](#blocked-member), being overdue has no effect.

Worked example: member `ada` on tier `plus` uses a machine with rate 45 from `2026-03-03T10:00Z` to `10:37`. At check-out a usage charge of 1499 cents is created ([money](#money)). Generating `2026-03` at `2026-04-01T08:00Z` gives `ada` an invoice with the membership line (4900) and the usage line (1499): `totalCents` 6399, due `2026-04-15T08:00Z`.

#### Blocked member

A member is **blocked** at a request time when at least one of the member's [invoices](#invoices) has status `open` and the request time is more than 30 days after its `dueAt`, that is, later than `dueAt` plus 43200 minutes. At exactly `dueAt` plus 30 days the member is not yet blocked; one minute later the member is blocked.

The block ends as soon as no invoice meets that condition, which happens only when each such invoice reaches status `paid`. A partial payment that leaves a balance does not end the block. Credit is never applied to an existing invoice, so issuing credit does not end it either.

A blocked member cannot make new reservations or buy materials: [createReservation](#createreservation) and [purchaseMaterial](#purchasematerial) refuse with `account_overdue`. The block applies whoever acts: a member for themselves, staff or an admin.

The block is not a standing. A blocked member's standing stays `good` unless staff suspend the member. Existing reservations stay. Check-in, check-out, cancellation, course enrollment, payments, credits and invoice generation are not blocked. An invoice is overdue from `dueAt` on, but it blocks only after more than 30 days.

Example: an invoice issued at `2026-04-01T08:00Z` is due at `2026-04-15T08:00Z`. While it is open, a reservation request at `2026-05-15T08:00Z` is not blocked; one at `2026-05-15T08:01Z` is refused with `account_overdue`.

#### Payments and credit

A payment pays one [invoice](#invoices). The part of it that exceeds the invoice's balance becomes credit on the member's account. Paying after the due date is allowed; an invoice that is `paid` takes no further payment.

Each member has a [credit balance](#credit-balance), initially 0. It increases by overpayments, by credits an admin [issues](#issuecredit) as a goodwill gesture and by [lockout credits](#lockmachine). Credit is applied automatically to invoices generated later, as step 5 of [issuing invoices](#invoices) describes; it is never applied to an invoice that already exists and is never paid out.

#### `generateInvoices`

Operation `generateInvoices`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `month` (month).
- Refusals, in this order: `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`); `already_invoiced` when invoices for this month were generated before.
- Effect: creates the invoices as described in [charges and invoices](#invoices) and marks the month as invoiced, even when there is no member to invoice.
- Result: `month` and `invoices`, the invoices created by this request.

#### `listInvoices`

Operation `listInvoices`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: `memberId` (identifier, an existing member). Result: `memberId` and `invoices`, all of the member's [invoices](#invoices).

#### `recordPayment`

Operation `recordPayment`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `invoiceId` (identifier, an existing [invoice](#invoices)), `amountCents` (integer 1 to 100000000).
- Refusals: `invoice_paid` when the invoice's status is `paid`.
- Effect: `appliedCents` is the smaller of `amountCents` and the invoice's `balanceCents`; `creditedCents` is `amountCents − appliedCents`. The invoice's `paidCents` increases and its `balanceCents` decreases by `appliedCents`; when `balanceCents` reaches 0 its status becomes `paid`. The member's [credit balance](#credit-balance) increases by `creditedCents`. A [payment record](#payments-and-credit) is created with `memberId` the invoice's member (whoever acts) and `at` the request time.
- Result: `payment`, `invoice` (after the change) and `creditBalanceCents`, the member's credit balance after the change.

#### `issueCredit`

Operation `issueCredit`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `memberId` (identifier, an existing member), `amountCents` (integer 1 to 100000000), `reason` (text).
- Effect: the member's [credit balance](#credit-balance) increases by `amountCents`. The credit counts as issued at the request time ([reporting](#revenuereport)).
- Result: `memberId`, `amountCents` and `creditBalanceCents`, the member's credit balance after the change.

#### `getAccount`

Operation `getAccount`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: `memberId` (identifier, an existing member). Result: `account`, where:

- `creditBalanceCents` is the member's [credit balance](#credit-balance);
- `openBalanceCents` is the sum of `balanceCents` over the member's [invoices](#invoices) with status `open`;
- `overdueBalanceCents` is the same sum over the member's overdue invoices;
- `unbilledCents` is the sum of the amounts of the member's [unbilled charges](#invoices);
- `blocked` states whether the member is [blocked](#blocked-member) at the request time.

## Safety

Staff record safety incidents against [machines](#catalog). An incident itself blocks nothing; only its [lockouts](#lockouts) do.

### `reportIncident`

Operation `reportIncident`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `machineId` (identifier, an existing machine), `severity` (enumeration `low`, `medium` or `high`), `description` (text).
- Effect: creates an [incident](#safety) with status `open`, `reportedBy` the actor, `reportedAt` the request time, and `resolution` and `resolvedAt` `null`. When a steward reports an incident, `reportedBy` is the steward's identifier.
- Result: `incident`.

### `resolveIncident`

Operation `resolveIncident`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation.

- Arguments: `incidentId` (identifier, an existing incident), `resolution` (text).
- Refusals: `invalid_state` when the [incident](#safety) is already resolved.
- Effect: status `resolved`, `resolution` as supplied, `resolvedAt` the request time.
- Result: `incident`.

### `listIncidents`

Operation `listIncidents`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Arguments: none besides `actor` and `at`. Result: `incidents`, all [incidents](#safety).

### Lockouts

Staff can use an open [incident](#safety) to [lock its machine](#lockmachine) for a time window. A lockout cancels the reservations that have not started being used in that window, gives each affected member a credit, prevents new reservations in the window and is excluded from the machine's available time in the [utilization report](#utilizationreport).

A lockout belongs to one incident and locks that incident's machine for an interval `[start, end)`. Like [maintenance windows](#maintenance), lockouts may use any minute, are not limited to opening hours and may overlap each other and maintenance windows. An incident can have several lockouts. A lockout cannot be changed, cancelled or ended early; resolving its incident does not end it.

A lockout never interrupts a checked-in session: a session already checked in when the lockout is created, and a session that overruns its reservation's end into a lockout, both keep metering until check-out. [Scheduling maintenance](#schedulemaintenance) that overlaps a lockout is allowed, and a lockout never causes `reservation_conflict`.

#### `lockMachine`

Operation `lockMachine`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Mutation. Locks the machine of an open incident.

- Arguments: `incidentId` (identifier, an existing incident), `start` (timestamp), `end` (timestamp).
- Refusals, in this order: `invalid_state` when the [incident](#safety) is resolved; `invalid_interval` when `end` is not after `start`; `starts_in_past` when `start` is before the request time.
- Effect:
  1. A [lockout](#lockouts) is created for the incident's machine.
  2. Every [reservation](#reservations) of that machine whose reported status is `reserved` and whose interval [overlaps](#time) the lockout becomes `cancelled`. These are the reservations of the window that are not yet used: not checked in and not missed. This includes a reservation whose start has already passed but which is not yet checked in and not yet missed. Reservations that are checked in, completed, missed or already cancelled are not changed.
  3. For each reservation cancelled this way, the reservation's member receives a lockout credit of 1000 cents: the member's [credit balance](#credit-balance) increases by 1000. This happens whatever the member's standing.
- Result: `lockout` and `cancelledReservations`, the reservations cancelled by this request (after the change), in creation order.

The request writes exactly one audit entry, for `lockMachine`. The cancellations and credits are part of that request and have no entries of their own.

Lockout credits are intended: a steward may lock a machine with `lockMachine`, and the lockout credits it grants take effect as for staff. This is a safety action with a fixed consequence, not billing access: a steward still cannot read or change invoices, payments, accounts or credit.

## Reporting

Reports are read-only and are calculated from the current state at the [request time](#time).

### `utilizationReport`

Operation `utilizationReport`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Shows how much each [machine](#catalog) was used compared with the time it was available.

- Arguments: `from` (date), `to` (date). The report range is the interval from `00:00` of `from` to `00:00` of `to`; `to` itself is excluded.
- Refusals: `invalid_interval` when `to` is not after `from` or the range is longer than 366 days.
- Result: `from`, `to`, `machines` (one row per machine that exists at the request time) and `totals`. A row has the fields `machineId`, `openMinutes`, `maintenanceMinutes`, `lockoutMinutes`, `availableMinutes`, `usedMinutes` and `utilizationPercent`; `totals` has the same fields without `machineId`.
  - `openMinutes`: minutes of the range that lie inside [opening hours](#opening-hours). Every machine gets the full count, regardless of when it was added.
  - `maintenanceMinutes`: minutes of the range that lie inside opening hours and inside at least one scheduled [maintenance window](#maintenance) of the machine. Overlapping windows count once.
  - `lockoutMinutes`: minutes of the range that lie inside opening hours and inside at least one [lockout](#lockouts) of the machine, overlapping lockouts counting once.
  - `availableMinutes`: `openMinutes` minus the minutes of the range that lie inside opening hours and inside at least one scheduled maintenance window or lockout of the machine, each minute counted once. A minute covered by both maintenance and a lockout appears in both `maintenanceMinutes` and `lockoutMinutes` but is subtracted from `availableMinutes` once.
  - `usedMinutes`: for every completed [reservation](#reservations) of the machine, the minutes of `[checkedInAt, checkedOutAt)` that lie in the range, summed. Opening hours do not limit this count. Reservations that are checked in but not checked out are not counted. The lockout excludes time from the available minutes, not from the used minutes: a session already checked in when the lockout is created, and a session that overruns its reservation's end into a lockout, both keep metering until check-out, and their metered minutes count in full, also those inside the lockout.
  - `utilizationPercent`: `(usedMinutes × 100 × 2 + availableMinutes) div (availableMinutes × 2)`, which is `usedMinutes × 100 / availableMinutes` rounded half up to an integer, or 0 when `availableMinutes` is 0. It can exceed 100.
  - `totals`: the sums of `openMinutes`, `maintenanceMinutes`, `lockoutMinutes`, `availableMinutes` and `usedMinutes` over all rows, and `utilizationPercent` computed from these sums with the same formula. With no machines, all totals are 0.

### `revenueReport`

Operation `revenueReport`. [Request processing](#order-of-checks), [results](#records-in-responses) and [delivery](#delivery) apply.

Read. Summarizes billing for one month.

- Arguments: `month` (month).
- Result fields:
  - `month`;
  - `invoicesGenerated`: whether [invoices](#invoices) for the month have been generated;
  - `invoiceCount`: the number of invoices of the month;
  - `membershipCents`, `usageCents`, `materialCents`, `courseCents`: the sums of the [line amounts](#invoices) of each kind on the invoices of the month;
  - `invoicedCents`: the sum of those four;
  - `creditAppliedCents`: the sum of `creditAppliedCents` over the invoices of the month;
  - `outstandingCents`: the sum of `balanceCents` over the invoices of the month at the request time;
  - `paymentsCents`: the sum of `amountCents` of all [payments](#payments-and-credit) whose `at` lies in the month, for any invoice, including the parts that became credit;
  - `creditsIssuedCents`: the sum of [credits issued](#issuecredit) with `issueCredit` whose request time lies in the month, and of the [lockout credits](#lockmachine) whose `lockMachine` request time lies in the month. Credit from overpayments is not included.
- For a month without generated invoices, the invoice figures are 0 and `invoicesGenerated` is `false`; payments and issued credits are still reported.

## Delivery

Deliver the [program](#program-build-and-process), automated tests and concise usage documentation.

- The tests run with `go test ./...` from `realization/`. They cover, for every operation, a successful request and each refusal code the operation lists; the four policies ([authorization](#authorization), [audit](#audit), [money](#money) and [time](#time)), including the [permission matrix](#permission-matrix) and audit entries; the boundary rules of time (half-open intervals, [opening hours](#opening-hours), [granularity and duration limits](#booking-granularity), rounding per line); and the cross-area rules of [reservations](#reservation-rules) and [charges and invoices](#invoices). Tests and documentation do not expect or describe [certification](#certifications) expiry; tests check that a certification issued long before a reservation date is still accepted.
- `realization/README.md` explains how to build, run and test the program, shows a short example session and states the scope.
- Code, tests and documentation agree with this document.
