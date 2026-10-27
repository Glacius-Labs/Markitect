# Makerspace requirements: after C9

This document describes the complete product after C9.

## Overview

Makerspace runs the day-to-day operations of a fictional community workshop. It is one local program. Front-desk tools send it requests on behalf of the person who acts; the program decides, records and answers.

- **Members** join on a membership tier and pay a monthly fee that depends on the tier. Staff can suspend a member; a suspended member is not in good standing.
- **Training:** courses certify members for a tool family, such as laser cutters.
- **Equipment:** machines belong to a tool family and have a rate per minute. Staff schedule maintenance windows during which a machine cannot be reserved.
- **Bookings:** members reserve machines in 15-minute steps inside opening hours, check in when they start and check out when they finish. They are charged for every metered minute at the machine's rate, minus a tier discount.
- **Materials:** a small shop sells materials from stock.
- **Billing:** once a month staff generate invoices: one per member, with the monthly fee and all charges of that month. An invoice is due 14 days after it is issued. Members pay their invoices; an overpayment becomes credit, and credit reduces the next invoice. An invoice more than 30 days overdue blocks reservations and purchases until it is paid.
- **Safety:** staff record safety incidents per machine and can lock a machine after an incident.
- **Reporting:** staff and admins read utilization and revenue reports.

Four policies apply to every operation: authorization, audit, money and time.

Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, notifications, scheduled jobs, deletion of records and integration with other systems.

The Go module lives at `realization/go.mod`; the program's main package lives at `realization/cmd/makerspace`. Only the Go standard library is used. The program is divided into the modules of its architecture; each module is one Go package under `realization/`, named after the module, for example `realization/billing`; `realization/protocol` and `realization/cmd/makerspace` belong to the protocol module.

## Architecture

The program is written in Go; the module root is `realization`. `go build ./cmd/makerspace` builds it and `go test ./...` runs its tests. No dependency beyond the language's standard library is permitted.

Each module below is one Go package under the module root, named after it.

| Module | May depend on | Rule |
| --- | --- | --- |
| `billing` | members, policies | The billing module depends only on members, policies. |
| `bookings` | members, training, equipment, billing, safety, policies | The bookings module depends only on members, training, equipment, billing, safety, policies. It depends on safety because a lockout belongs to an open incident. |
| `equipment` | policies | The equipment module depends only on policies. Where maintenance must know about reservations, equipment declares what it needs and the protocol module supplies it from bookings, so equipment never depends on bookings. |
| `materials` | members, billing, policies | The materials module depends only on members, billing, policies. |
| `members` | policies | The members module depends only on policies. |
| `policies` | nothing | The policies module depends on no other module of the project; every other module may depend on it. |
| `protocol` | policies, members, equipment, billing, training, bookings, materials, safety, reporting | The protocol module is the outer layer: it reads requests, runs the checks every operation shares and calls the area modules. It may depend on every other module; no module depends on it. |
| `reporting` | equipment, bookings, billing, policies | The reporting module depends only on equipment, bookings, billing, policies. |
| `safety` | equipment, policies | The safety module depends only on equipment, policies. |
| `training` | members, equipment, billing, policies | The training module depends only on members, equipment, billing, policies. |

A dependency is any use of another module's records, rules or operations, in the documentation and in the code. The standard library is not counted.

- **Audit** is realized in `protocol`: The protocol module appends the audit entry after a mutation is answered `ok`, using the operation's audit target, and after a `forbidden` refusal; area modules do not write audit entries.
- **Authorization** is realized in `protocol`: Checks 2, 4 and 6 of the order of checks run once, in the protocol module, before any operation's own rules; area modules do not check roles or ownership themselves.
- **Time** is realized in `protocol`: The protocol module keeps the request-time order (check 3) and passes the request time to the area modules as their only notion of now.

- **Members interface:** What other modules may use of members: its operations, the user and member records, tier and standing, and the refusal `member_suspended`. Other modules also ask whether a member exists, whether the member is in good standing, and the member's tier with its monthly fee and usage discount. Operations: `registerUser`, `registerMember`, `getMember`, `changeTier`, `setMemberStanding`. Types: user, member, tier, standing.
- **Training interface:** What other modules may use of training: its operations, the course, enrollment and certification records, and the refusal `not_certified`. Other modules also ask whether a member holds a certification for a family that is not revoked. Operations: `addCourse`, `enrollInCourse`, `completeCourse`, `revokeCertification`, `listCertifications`. Types: course, enrollment, certification.
- **Equipment interface:** What other modules may use of equipment: its operations and the family, machine and maintenance window records. Other modules also ask for a machine's family and current rate and for the scheduled windows of a machine that overlap an interval. Equipment in turn needs to know whether an active reservation of a machine overlaps an interval; it declares that need, and the protocol module supplies the answer from bookings. Operations: `addFamily`, `addMachine`, `setMachineRate`, `listMachines`, `scheduleMaintenance`, `cancelMaintenance`. Types: family, machine, maintenance window.
- **Bookings interface:** What other modules may use of bookings: its operations and the reservation and lockout records. Other modules also ask whether an active reservation of a machine overlaps an interval, for the completed sessions of a machine and for the lockouts of a machine. Operations: `createReservation`, `cancelReservation`, `checkIn`, `checkOut`, `listMemberReservations`, `getMachineSchedule`, `lockMachine`. Types: reservation, lockout, opening hours of day.
- **Materials interface:** What other modules may use of materials: its operations and the material and purchase records. Operations: `addMaterial`, `restockMaterial`, `purchaseMaterial`, `listMaterials`. Types: material, purchase.
- **Billing interface:** What other modules may use of billing: its operations, the charge, invoice, payment and account records, and the refusal `account_overdue`. Other modules create a charge for a member through billing, and billing computes its amount by money. Other modules also grant lockout credits through billing and ask whether a member is blocked. Operations: `generateInvoices`, `listInvoices`, `recordPayment`, `issueCredit`, `getAccount`. Types: charge, invoice, payment, account.
- **Safety interface:** What other modules may use of safety: its operations and the incident record. Operations: `reportIncident`, `resolveIncident`, `listIncidents`. Types: incident.
- **Reporting interface:** What other modules may use of reporting: its operations and its report rows. Operations: `utilizationReport`, `revenueReport`. Types: utilization row, utilization totals.

## Program interface

Each argument has a field type: an identifier, text, timestamp, date or month as their definitions state, an enumeration (a JSON string equal to one of the listed values, compared exactly), or an integer.

An integer is a JSON number written as an optional minus sign followed by digits, without a fraction part or exponent (`10.0` and `1e3` are not integers), within the range its type states. Every range in this document starts at 0 or higher, so any value written with a minus sign, including `-0`, is out of range.

An argument that is missing, `null`, of the wrong JSON type or outside its format or range is refused with `invalid_argument`. Each argument is checked on its own. Relationships between arguments, such as an end that is not after a start, and relationships to stored state are checked later and have their own codes.

These formats and ranges apply to arguments, including `at`. An integer argument outside its range is out of range however many digits it has. Values the program computes can lie outside these ranges; computed values states the rules for those values.

`actor` is an identifier argument and `at` is a timestamp argument of every operation. An `actor` that is missing, `null`, not a string or not a valid identifier, such as `"bad id!"` or a string of 65 characters, is refused with `invalid_argument` at check 1, like any other argument. Only a valid identifier that names no user is refused with `unknown_actor`, at check 2.

*Applies to the protocol package.*

A new process contains exactly one user: identifier `admin`, name `Administrator`, role `admin`. There are no members, families, machines, courses, materials, reservations, lockouts, invoices, incidents or audit entries.

The program generates the identifiers of these records:

| Record | Format |
| --- | --- |
| Reservation | `res-1`, `res-2`, … |
| Maintenance window | `mnt-1`, `mnt-2`, … |
| Purchase | `pur-1`, `pur-2`, … |
| Invoice | `inv-1`, `inv-2`, … |
| Payment | `pay-1`, `pay-2`, … |
| Incident | `inc-1`, `inc-2`, … |
| Lockout | `lock-1`, `lock-2`, … |

Each kind has its own counter. The first record of a kind gets number 1 and each further record of that kind gets the next number, in decimal without leading zeros. Numbers are never reused. Refused and invalid requests consume no number.

*Applies to the protocol package.*

The response is `status: "invalidRequest"` with a nonempty `reason` when:

- the line is not valid JSON, is empty, or is not a JSON object;
- `id` is missing, is not a string or is empty;
- `op` is missing, is not a string or names no operation of this document;
- `args` is missing or is not a JSON object;
- the request object has a member other than `id`, `op` and `args`;
- `args` has a member that is not an argument of the named operation, including a member whose name differs from a stated argument only in letter case.

The response's `id` is the request's `id` when the line is a JSON object whose `id` member is a nonempty string; otherwise it is `null`. A last line that ends without a line feed is still a request and gets a response. Behavior for JSON objects with duplicate member names and for lines that are not valid UTF-8 is not specified and is not tested. An invalid request changes nothing: no state, no audit entry, no identifier counter and no request-time order.

A request whose `args` lack a stated argument, or carry an argument with a wrong value, is not invalid in this sense. It is refused with code `invalid_argument`.

*Applies to the protocol package.*

A well-formed request is checked in the following order. The first check that fails decides the refusal and its code.

1. **Arguments:** every argument is present and valid by argument values → otherwise `invalid_argument`.
2. **Actor:** `actor` identifies an existing user of any role → otherwise `unknown_actor`.
3. **Request time:** `at` is not earlier than the request time of the most recent request that was answered `ok` (time) → otherwise `time_regression`.
4. **Role permission:** the actor's role may use the operation at all, as "all" or "own" in the operation's permissions → otherwise `forbidden`.
5. **Referenced records:** every argument that refers to a record names one that exists, checked in the order the operation lists its arguments → otherwise `not_found`. An argument that names a member must name a user whose role is `member`; the identifier of a staff, admin or steward user does not name a member.
6. **Ownership:** when the actor's permission for the operation is "own", the record concerned belongs to the actor (authorization) → otherwise `forbidden`.
7. **Operation rules:** the refusals listed for the operation, in the order listed.

When a check fails, the response is `status: "refused"` with the code and a reason, and nothing else happens, except that a `forbidden` refusal appends its audit entry: no record is created or changed, no identifier number is consumed, no other audit entry is written and the request-time order is not advanced. When every check passes, all of the operation's effects take place together and the response is `status: "ok"`.

*Applies to every package.*

- The program reads requests from standard input, one JSON object per line, and writes exactly one response line per request to standard output, in request order. It writes and flushes each response before it reads the next request. Standard output carries responses only: no banner, prompt or log line. Diagnostics, if any, go to standard error. When standard input closes, the program exits with exit code zero.
- One process holds one makerspace, which starts in the initial state. Requests are processed one at a time, in order.
- The program never reads the machine clock, the network, environment-dependent data or files at run time, and writes no files. All time comes from the requests. Given the same request lines, it produces the same response lines.
- There is no reset, state import, clock override or other test-only operation.
- Request lines up to 64 KiB must be accepted.

*Applies to the protocol package.*

A `result` object has exactly the results its operation lists, and every object inside it has exactly the fields stated for its type. There are never fewer or additional members. Nullable fields are present with the value `null` when they have no value. Lists are JSON arrays; an empty list is `[]`, never `null`. Amounts, counts and minutes are JSON integers. The order of members inside a JSON object is free; the order of list elements is fixed by the order of listed results.

*Applies to the protocol package.*

A request is a UTF-8 JSON object with exactly three members:

```json
{"id":"q1","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"Ada Byron","tier":"plus"}}
```

- `id`: a nonempty string chosen by the caller. It is echoed in the response. It is only a correlation value, not a domain identifier and not an idempotency key; two requests with the same `id` are two separate requests.
- `op`: the wire name of one operation of this document. Names are case-sensitive.
- `args`: a JSON object with the operation's arguments. Every operation takes `actor` (identifier, the acting user) and `at` (timestamp, the request time) in addition to the arguments it lists. All arguments are required; no operation has optional arguments.

*Applies to the protocol package.*

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

*Applies to the protocol package.*

## Shared rules and policies

Operations are either mutations or reads, as each operation states. Every request to a mutation that is answered `ok` appends exactly one audit entry with outcome `accepted`, even when the operation changes no data, such as setting a member's standing to its current value, and every request refused with `forbidden` appends exactly one entry with outcome `refused`. This holds for every operation, mutations and reads alike, and for both kinds of `forbidden`: a role without permission (check 4) and an ownership violation (check 6). Reads answered `ok`, other refusals and invalid requests append nothing. Entries are never changed or removed.

An audit entry has these fields: `seq`, the entry's position in the log, starting at 1 and increasing by 1 for each entry of the one shared sequence; `at`, the request time; `actor`, the acting user's identifier; `actorRole`, that user's role; `op`, the requested operation's name; `target`, a string that identifies the record the operation acted on, as each mutation's audit target states, or `null` in a refusal entry; `outcome`, `accepted` or `refused`.

A refusal entry changes nothing else: no other record, no identifier counter, and the request-time order is still set only by requests answered `ok`. A refusal entry can therefore carry a later `at` than the accepted entry that follows it; `listAuditEntries` still returns entries in `seq` order. A request that fails an earlier check than the permission checks, such as `time_regression`, is not recorded, even when the actor also lacks permission.

Example: member `ada` sends `listAuditEntries` at `2026-03-05T10:00Z`. The response is a `forbidden` refusal, and the log gains `{"seq": 7, "at": "2026-03-05T10:00Z", "actor": "ada", "actorRole": "member", "op": "listAuditEntries", "target": null, "outcome": "refused"}` (with `seq` the next number).

Admins read the log with `listAuditEntries`.

*Applies to every package.*

Every user has exactly one role, fixed when the user is registered: member, staff, admin or steward.

For each operation and role, the operation's permissions state one permission:

- **all:** the role may use the operation for any record.
- **own:** the role may use the operation only for records that belong to the actor.
- **none** (no permission listed for the role): the role may not use the operation.

A record concerned belongs to the actor when:

- the operation has a `memberId` argument and it equals the actor's identifier;
- the operation has a `reservationId` argument and the reservation's member is the actor;
- the operation has an `invoiceId` argument and the invoice's member is the actor.

A "none" permission is refused at check 4, an ownership violation at check 6 of the order of checks. Both use the code forbidden.

*Applies to every package.*

**Booking granularity is 15 minutes.** A reservation's start and end must each fall on a minute that is a multiple of 15 within the hour (`:00`, `:15`, `:30` or `:45`). A reservation lasts at least one granularity step (15 minutes) and at most 240 minutes. Check-in and check-out times are not tied to the granularity.

*Applies to the bookings package.*

Request times and time arguments lie between 2000 and 2099, but some values the program computes from them lie later. Integers grow with the number of requests. This rule and integer limits, identifier limits and request limits cover every computed value.

- **Times after 2099.** A computed timestamp or date can lie after 2099. It uses the same format as its type, with a four-digit year, and the Gregorian calendar continues as usual: 2100 is not a leap year, so `2099-03-01` plus 365 days is `2100-03-01`. The latest value a result can show is a `dueAt` of `2100-01-14T23:59Z` (14 days after `2099-12-31T23:59Z`). Points used only in comparisons can also lie after 2099: the first minute after the month `2099-12` and after the date `2099-12-31` are both `2100-01-01T00:00Z`, and the block compares the request time with `dueAt` plus 43200 minutes, a point as late as `2100-02-13T23:59Z`. No computed time lies before `2000-01-01T00:00Z`.
- **The end of the calendar.** No request time is later than `2099-12-31T23:59Z`. The following consequences follow from the rules of this document and are stated so that they are read only one way:
  - `2099-11` is the last month that can be invoiced. `generateInvoices` for `2099-12` is never accepted: when it reaches the operation rules (check 7), it is refused with `month_not_ended`. The revenue report for `2099-12` always has `invoicesGenerated` `false`.
  - An invoice whose `dueAt` lies after 2099 is never overdue.
  - An invoice whose `dueAt` is `2099-12-01T23:59Z` or later can never block.
  - The last day a utilization report can include is `2099-12-30`, because `to` is at most `2099-12-31` and is excluded from the range.
  - A maintenance window or a lockout ends at the latest at `2099-12-31T23:59Z`, so the last minute of 2099 cannot be covered. Reservations end by closing time and are not affected.

*Applies to every package.*

- **Generated identifiers stay valid identifiers.** A generated identifier is its prefix followed by a number. Within the integer limit it has at most 21 characters (a prefix of at most five characters, such as `lock-`, and at most 16 digits), so it always meets the identifier format and can be passed back as an argument.
- **Audit targets are not identifiers.** An audit `target` is a plain string, or `null` in a refusal entry. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format.

*Applies to every package.*

- **Integers are exact and never negative.** Within the integer limit stated under "Large sums and counters" below, no computed integer is negative. Stock cannot fall below 0 (`insufficient_stock`). Credit and payments are applied at most up to an invoice's balance. Metered minutes are at least 0, because a check-out cannot precede its check-in in the request-time order. Maintenance and lockout minutes are part of the open minutes they are subtracted from, so `availableMinutes` is never negative.
- **Single lines cannot overflow.** The argument ranges keep every line amount and the product inside its formula below 2^53. The largest is a usage line: metered minutes are fewer than 52,596,000 (from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`), and at 10000 cents per minute the product `minutes × 10000 × 100` stays below 5.26 × 10^13. A material line's product is at most `10000 × 1000000 × 100` = 10^12.
- **Large sums and counters.** Sums and counters grow with the number of requests: invoice totals, balances, credit balances including lockout credits, stock, report sums and minute totals, `seq` and the numbers in generated identifiers. The program keeps every integer exact up to 9007199254740991 (2^53 − 1); signed 64-bit integer arithmetic satisfies this. Once a stored or returned integer, or an intermediate value of a formula in this document, would exceed 2^53 − 1, the rest of the run is not specified and is not tested. Reaching that limit takes at least tens of thousands of requests.

*Applies to every package.*

Identifiers are ordered by ASCII byte value (so `-` < `.` < digits < uppercase letters < `_` < lowercase letters, and `M10` < `M9`). Listed results are ordered as follows:

| List | Order |
| --- | --- |
| `listCertifications` → `certifications` | `familyId` ascending |
| `listMachines` → `machines` | `machineId` ascending |
| `getMachineSchedule` → `reservations` | `start` ascending, then creation order |
| `getMachineSchedule` → `maintenance` | `start` ascending, then creation order |
| `getMachineSchedule` → `lockouts` | `start` ascending, then creation order |
| `listMemberReservations` → `reservations` | `start` ascending, then creation order |
| `lockMachine` → `cancelledReservations` and a lockout's `cancelledReservationIds` | creation order |
| `listMaterials` → `materials` | `materialId` ascending |
| `generateInvoices` → `invoices` | `memberId` ascending |
| `listInvoices` → `invoices` | `month` ascending |
| invoice → `lines` | the membership line first, then the other lines by their `at` ascending, then in the order the charges were created |
| `listIncidents` → `incidents` | creation order |
| `utilizationReport` → `machines` | `machineId` ascending |
| `listAuditEntries` → `entries` | `seq` ascending |

"Creation order" means the order in which records of a kind were created, which is the order of their generated numbers.

*Applies to every package.*

- All amounts are integer numbers of cents, represented as JSON integers. There are no fractions of a cent, no floating-point amounts and no currency field. Stored and returned amounts are never negative.
- Every charge is a line with a `quantity`, a `unitPriceCents` and a `discountPercent`. Its amount is
  `amountCents = (quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`,
  where `div` is integer division. This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again.
  Example: 37 minutes at 45 cents with a 10 % discount: `(37 × 45 × 90 + 50) div 100 = 149900 div 100 = 1499` cents.
- A charge's amount is fixed when the charge is created. Later changes to a rate, a tier or a price never change existing charges.
- Each member has a credit balance, initially 0. It increases by credits issued by an admin, by lockout credits and by overpayments. It decreases only when credit is applied to a new invoice. It is never negative. Credit is never paid out.

*Applies to every package.*

**Opening hours** are the same every week:

| Day | Opens | Closes | Open minutes |
| --- | --- | --- | --- |
| Monday to Thursday | 08:00 | 22:00 | 840 |
| Friday | 08:00 | 20:00 | 720 |
| Saturday | 09:00 | 17:00 | 480 |
| Sunday | 12:00 | 16:00 | 240 |

The opening period of a day is the interval `[opens, closes)` on that date. A reservation is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Every day has an opening period. The hours apply to every date, including dates before this change.

*Applies to the bookings and reporting packages.*

- **Request size and depth.** The 64 KiB request lines are 65,536 bytes, not counting the line feed. Longer lines, and lines with any JSON value nested more than 100 levels deep in arrays or objects, are not specified and are not tested.
- **Counting depth.** Levels count arrays and objects only. The outermost array or object of the line, normally the request object, is level 1, and an array or object directly inside a level-n array or object is level n + 1. So `args` is level 2, and an array or object given as an argument value is level 3. Strings, numbers, booleans and `null` add no level. A JSON value is nested more than 100 levels deep when it is an array or object at level 101 or deeper, or lies inside one. Up to level 100, all other rules apply in full. A line that is not valid JSON is answered by the invalid-request rules whatever its depth.

*Applies to every package.*

- All times are UTC with a precision of one minute. Timestamps, dates and months use the formats timestamp, date and month. The day of the week of a timestamp is that of its UTC date. The date of a timestamp is its UTC calendar date.
- **Request time:** `at` is the time at which the request happens. It is the program's only notion of "now". The request time of every request must not be earlier than the request time of the most recent request answered `ok`; an equal time is allowed. Refused and invalid requests do not move this point. Before the first `ok` response, any valid `at` is accepted.
- **Intervals are half-open.** An interval `[start, end)` contains its start minute and excludes its end minute. Two intervals `[a, b)` and `[c, d)` overlap exactly when `a < d` and `c < b`; intervals that only touch, such as 10:00–11:00 and 11:00–12:00, do not overlap. The length of an interval in minutes is `end − start`.
- **Day and date arithmetic:** a day is 1440 minutes; "N days after timestamp t" is `t + N × 1440` minutes. "N days after date d" is the calendar date N days later, so `2027-03-01` plus 365 days is `2028-02-29`.
- **Durations.** A reservation lasts at most 240 minutes. Maintenance windows, lockouts and metered sessions have no maximum length apart from the timestamp range.
- **States that depend on time** are evaluated at the request time of the request that reads or uses them: a reservation that is `missed`, an invoice that is `overdue` and a member who is blocked.

*Applies to every package.*

### Value types

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| amount cents | integer (cents) | 0 or more | An amount of money; never negative. |
| audit target text | string |  | A plain string that identifies the record an operation acted on. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format. |
| clock time | string |  | A time of day `HH:MM`. |
| count | integer | 0 or more | A count, quantity or sequence number; never negative. |
| date | string |  | A JSON string `YYYY-MM-DD` naming a valid calendar date from `2000-01-01` to `2099-12-31`. `2026-02-29` is not valid; `2028-02-29` is. |
| flag | boolean |  | A yes-or-no value. |
| identifier | string |  | A JSON string of 1 to 64 characters from `A`–`Z`, `a`–`z`, `0`–`9`, `.`, `_` and `-`, whose first character is a letter or digit. Identifiers are compared exactly and are case-sensitive: `Ada` and `ada` are different identifiers. |
| minute count | integer (minute) | 0 or more | A number of minutes; never negative. |
| month | string |  | A JSON string `YYYY-MM` from `2000-01` to `2099-12`. |
| operation name | string |  | The name of an operation, as in a request's `op`. |
| percent | integer | 0 or more | A discount in percent. |
| text | string |  | A JSON string that contains at least one character without the Unicode `White_Space` property. It is stored and returned exactly as supplied, without trimming. |
| timestamp | string |  | A JSON string `YYYY-MM-DDTHH:MMZ` naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`, for example `2026-03-02T09:00Z`. Hours are `00` to `23`, minutes `00` to `59`; there are no seconds. |

Enumeration audit outcome: Whether an audit entry records an accepted mutation or a `forbidden` refusal.

- `accepted`
- `refused`

Enumeration user role: The role of a user; see authorization.

- `member`
- `staff`
- `admin`
- `steward`

- Unit cents (`ct`): One cent; all amounts are integer numbers of cents.
- Unit minute (`min`): One minute of time.

#### Record: audit entry

One entry of the audit log. Entries are never changed or removed. `target` is `null` in a refusal entry.

| Field | Type |
| --- | --- |
| `seq` | integer 0 or more |
| `at` | timestamp |
| `actor` | identifier |
| `actorRole` | enumeration `member`, `staff`, `admin` or `steward` |
| `op` | operation name |
| `target` | audit target text, nullable |
| `outcome` | enumeration `accepted` or `refused` |

### Roles

- `member`: Members are the workshop's customers.
- `staff`: Staff run the front desk and the workshop floor.
- `admin`: Admins also manage the catalog, credits and users.
- `steward`: Stewards look after the machines and safety on the workshop floor. A steward is a user, not a member: a steward's identifier does not name a member, and a steward has no membership, invoices or credit. A steward has no access to billing.

### Shared refusal codes

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

### Operations

#### `listAuditEntries`

Read. Returns all audit entries in `seq` order.

- Arguments: none, besides `actor` and `at`.
- Result: `entries` (list of audit entry record).
- Permissions: member none, staff none, admin all, steward none.

### Audited operations

An operation that changes state and therefore names the target of its audit entry.

The `target` of the operation's audit entry: the name of an argument or of a new identifier whose value is the target; `a/b` joins two values with a slash, for example `laser-101/ada` for course `laser-101` and member `ada`; `month` is the month string, for example `2026-03`.

## Members

A membership has a tier and a standing. The fee of the tier is charged once for every invoiced month, without proration. The usage discount applies only to usage charges, never to course or material charges.

A member is **in good standing** when the standing is `good`. A suspended member cannot make reservations, check in, buy materials or enroll in courses. Suspension does not cancel existing reservations or enrollments. A suspended member can still cancel reservations, check out, see their records and pay invoices, and still receives the monthly fee. Reinstating good standing removes these restrictions; all other rules still apply.

*Applies to every package.*

The registry holds every user: the initial admin, staff, admins and stewards registered by an admin, and members registered by staff or admins. Users, including members, share one identifier space. Users are never deleted and their identifier, name and role never change. A steward is a user, not a member.

*Applies to the members package.*

Enumeration staff role choice: The roles `registerUser` can give.

- `staff`
- `admin`
- `steward`

Enumeration standing: A member's standing.

- `good`: In good standing.
- `suspended`: Not in good standing.

Enumeration tier: A membership tier. It sets the monthly fee and the discount on machine usage.

- `basic`: Monthly fee 2500 cents, usage discount 0 %.
- `plus`: Monthly fee 4900 cents, usage discount 10 %.
- `pro`: Monthly fee 9900 cents, usage discount 25 %.

| Code | Meaning |
| --- | --- |
| `member_suspended` | The member is not in good standing. |

#### Record: member

A user with role `member` and a membership.

Identified by `memberId`.

| Field | Type |
| --- | --- |
| `memberId` | identifier |
| `name` | text |
| `tier` | enumeration `basic`, `plus` or `pro` |
| `standing` | enumeration `good` or `suspended` |
| `registeredAt` | timestamp |

#### Record: user

A user of the program: the initial admin, staff, admins and stewards registered by an admin, and members registered by staff or admins.

Identified by `userId`.

| Field | Type |
| --- | --- |
| `userId` | identifier |
| `name` | text |
| `role` | enumeration `member`, `staff`, `admin` or `steward` |

#### `changeTier`

Mutation. Sets the member's tier. Setting the current tier succeeds without a change. The new tier applies to membership fees of invoices generated afterwards and to usage charges created by later check-outs.

- Arguments: `memberId` (identifier, an existing member); `tier` (enumeration `basic`, `plus` or `pro`), besides `actor` and `at`.
- Result: `member` (member record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `memberId`.

#### `getMember`

Read. Returns the member.

- Arguments: `memberId` (identifier, an existing member), besides `actor` and `at`.
- Result: `member` (member record).
- Permissions: member own, staff all, admin all, steward none.

#### `registerMember`

Mutation. Registers a member with a membership: creates a user with role `member`, the given tier, standing `good`, `registeredAt` equal to the request time and a credit balance of 0.

- Arguments: `memberId` (identifier); `name` (text); `tier` (enumeration `basic`, `plus` or `pro`), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when any user has the identifier `memberId`.
- Result: `member` (member record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `memberId`.

#### `registerUser`

Mutation. Registers a staff member, an admin or a steward: creates the user.

- Arguments: `userId` (identifier); `name` (text); `role` (enumeration `staff`, `admin` or `steward`), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when any user, including a member, has the identifier `userId`.
- Result: `user` (user record).
- Permissions: member none, staff none, admin all, steward none.
- Audit target: `userId`.

#### `setMemberStanding`

Mutation. Suspends or reinstates the member: sets the standing. Setting the current standing succeeds without a change.

- Arguments: `memberId` (identifier, an existing member); `standing` (enumeration `good` or `suspended`), besides `actor` and `at`.
- Result: `member` (member record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `memberId`.

## Training

A member has at most one certification per tool family. Completing a course issues it: `issuedOn` is the date of the request time. If the member already has a certification for that family, revoked or not, it is replaced by the newly issued, valid one. A certification has no expiry: it stays valid from its issue until it is revoked.

A certification is **valid on date D** when it is not revoked and `issuedOn ≤ D`. Reservations and check-ins always lie on or after the issue date, so in practice they require a certification for the machine's family that is not revoked. Its `status` is `revoked` when revoked, otherwise `valid`.

*Applies to the training and bookings packages.*

A course teaches the safe use of one tool family. It has a fee and a capacity. A member enrolls; staff record completion, which certifies the member for the course's family. There is one enrollment record per course and member. It is created by the first enrollment; a later enrollment after completion reuses and resets it.

The course's `enrolledCount` is the number of its enrollments with status `enrolled`. A place becomes free when an enrollment is completed.

*Applies to the training package.*

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| capacity | integer | 1 to 1000 | The number of places of a course. |
| fee cents | integer (cents) | 0 to 1000000 | A course fee. |

Enumeration certification status: The status of a certification.

- `valid`
- `revoked`

Enumeration enrollment status: The status of an enrollment.

- `enrolled`
- `completed`

| Code | Meaning |
| --- | --- |
| `already_enrolled` | The member is already enrolled in the course. |
| `course_full` | The course has no free place. |
| `not_certified` | The member has no certification for the machine's family that is not revoked, or there is no certification to revoke. |
| `not_enrolled` | The member is not enrolled in the course. |

#### Record: certification

A member's certification for one tool family.

| Field | Type |
| --- | --- |
| `memberId` | identifier, an existing member |
| `familyId` | identifier, an existing family |
| `issuedOn` | date |
| `status` | enumeration `valid` or `revoked` |

#### Record: course

A course that teaches the safe use of one tool family, with a fee and a capacity.

Identified by `courseId`.

| Field | Type |
| --- | --- |
| `courseId` | identifier |
| `familyId` | identifier, an existing family |
| `title` | text |
| `feeCents` | integer 0 to 1000000 |
| `capacity` | integer 1 to 1000 |
| `enrolledCount` | integer 0 or more |

#### Record: enrollment

The enrollment of one member in one course. `completedAt` is `null` while enrolled.

| Field | Type |
| --- | --- |
| `courseId` | identifier, an existing course |
| `memberId` | identifier, an existing member |
| `status` | enumeration `enrolled` or `completed` |
| `enrolledAt` | timestamp |
| `completedAt` | timestamp, nullable |

#### `addCourse`

Mutation. Creates the course with `enrolledCount` 0.

- Arguments: `courseId` (identifier); `familyId` (identifier, an existing family); `title` (text); `feeCents` (integer 0 to 1000000); `capacity` (integer 1 to 1000), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when a course with `courseId` exists.
- Result: `course` (course record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `courseId`.

#### `completeCourse`

Mutation. Records that the member completed the course: the enrollment gets status `completed` and `completedAt` equal to the request time. The member's certification for the course's family is issued as certifications states. Standing is not checked.

- Arguments: `courseId` (identifier, an existing course); `memberId` (identifier, an existing member), besides `actor` and `at`.
- Refusals, in this order: `not_enrolled` when the member has no enrollment in this course with status `enrolled`.
- Result: `enrollment` (enrollment record); `certification` (certification record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `courseId/memberId`.

#### `enrollInCourse`

Mutation. Enrolls the member in the course. The enrollment gets status `enrolled`, `enrolledAt` equal to the request time and `completedAt` `null`. A course charge is created for the member: `kind` `course`, `ref` the course identifier, `quantity` 1, `unitPriceCents` the course fee, `discountPercent` 0, `at` the request time. A course with fee 0 creates a charge of 0 cents. The `course` result is the course after the change.

- Arguments: `courseId` (identifier, an existing course); `memberId` (identifier, an existing member), besides `actor` and `at`.
- Refusals, in this order: `member_suspended` when the member is not in good standing; `already_enrolled` when the member's enrollment in this course has status `enrolled`; `course_full` when `enrolledCount` equals `capacity`.
- Result: `enrollment` (enrollment record); `course` (course record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `courseId/memberId`.

#### `listCertifications`

Read. Returns all of the member's certifications, including revoked ones.

- Arguments: `memberId` (identifier, an existing member), besides `actor` and `at`.
- Result: `memberId` (identifier); `certifications` (list of certification record).
- Permissions: member own, staff all, admin all, steward none.

#### `revokeCertification`

Mutation. Withdraws the member's certification, for example after a safety incident. The certification becomes revoked; its date does not change. Existing reservations stay, but check-in requires a certification that is not revoked.

- Arguments: `memberId` (identifier, an existing member); `familyId` (identifier, an existing family), besides `actor` and `at`.
- Refusals, in this order: `not_certified` when the member has no certification for the family or it is already revoked.
- Result: `certification` (certification record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `memberId/familyId`.

## Equipment

The catalog holds tool families and machines. Each machine belongs to one family and has a rate in cents per minute. Families and machines are never removed.

*Applies to the equipment package.*

A maintenance window takes a machine out of service for an interval `[start, end)`. While a scheduled window overlaps a slot, the machine cannot be reserved for it, and since every use starts with a reservation and its check-in, the machine cannot be used in the window. The one exception is a session that is already checked in and runs past its reservation's end: it is not interrupted, keeps metering until check-out and counts in `usedMinutes` of the utilization report. Windows may use any minute, are not limited to opening hours and may overlap each other.

*Applies to the equipment, bookings and reporting packages.*

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| rate cents per minute | integer (cents) | 1 to 10000 | A machine's rate per minute. |

Enumeration window status: The status of a maintenance window.

- `scheduled`
- `cancelled`

| Code | Meaning |
| --- | --- |
| `reservation_conflict` | An active reservation overlaps the maintenance window. |

#### Record: family

A tool family, such as laser cutters.

Identified by `familyId`.

| Field | Type |
| --- | --- |
| `familyId` | identifier |
| `name` | text |

#### Record: machine

A machine of one family with a rate in cents per minute.

Identified by `machineId`.

| Field | Type |
| --- | --- |
| `machineId` | identifier |
| `name` | text |
| `familyId` | identifier, an existing family |
| `rateCentsPerMinute` | integer 1 to 10000 |

#### Record: maintenance window

A maintenance window of a machine, generated identifier `mnt-N`.

Identified by `windowId`.

| Field | Type |
| --- | --- |
| `windowId` | identifier |
| `machineId` | identifier, an existing machine |
| `start` | timestamp |
| `end` | timestamp |
| `reason` | text |
| `status` | enumeration `scheduled` or `cancelled` |

#### `addFamily`

Mutation. Creates the family.

- Arguments: `familyId` (identifier); `name` (text), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when a family with `familyId` exists.
- Result: `family` (family record).
- Permissions: member none, staff none, admin all, steward none.
- Audit target: `familyId`.

#### `addMachine`

Mutation. Creates the machine.

- Arguments: `machineId` (identifier); `name` (text); `familyId` (identifier, an existing family); `rateCentsPerMinute` (integer 1 to 10000), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when a machine with `machineId` exists.
- Result: `machine` (machine record).
- Permissions: member none, staff none, admin all, steward none.
- Audit target: `machineId`.

#### `cancelMaintenance`

Mutation. The window gets status `cancelled`. A cancelled window no longer blocks reservations and no longer counts in schedules or reports.

- Arguments: `windowId` (identifier, an existing maintenance window), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the window is already cancelled or its start is at or before the request time.
- Result: `window` (maintenance window record).
- Permissions: member none, staff all, admin all, steward all.
- Audit target: `windowId`.

#### `listMachines`

Read. Returns all machines.

- Arguments: none, besides `actor` and `at`.
- Result: `machines` (list of machine record).
- Permissions: member all, staff all, admin all, steward all.

#### `scheduleMaintenance`

Mutation. Creates the window with status `scheduled` and the next maintenance window identifier. A reservation is active when its reported status is `reserved` or `checkedIn`.

- Arguments: `machineId` (identifier, an existing machine); `start` (timestamp); `end` (timestamp); `reason` (text), besides `actor` and `at`.
- Refusals, in this order: `invalid_interval` when `end` is not after `start`; `starts_in_past` when `start` is before the request time; `reservation_conflict` when the window overlaps an active reservation of the machine.
- Result: `window` (maintenance window record).
- Permissions: member none, staff all, admin all, steward all.
- Audit target: `new windowId`.

#### `setMachineRate`

Mutation. Sets the rate. The rate is captured at check-in, so the new rate applies to check-ins after the change. Reservations already checked in keep their captured rate, and existing charges do not change.

- Arguments: `machineId` (identifier, an existing machine); `rateCentsPerMinute` (integer 1 to 10000), besides `actor` and `at`.
- Result: `machine` (machine record).
- Permissions: member none, staff none, admin all, steward none.
- Audit target: `machineId`.

## Bookings

A lockout belongs to one open incident and locks that incident's machine for an interval `[start, end)`. Like maintenance windows, lockouts may use any minute, are not limited to opening hours and may overlap each other and maintenance windows. An incident can have several lockouts. A lockout cannot be changed, cancelled or ended early; resolving its incident does not end it. An incident itself blocks nothing; only its lockouts do.

A lockout prevents new reservations of the machine in its interval and is excluded from the machine's available time in the utilization report. It never interrupts a checked-in session: a session already checked in when the lockout is created, and a session that overruns its reservation's end into a lockout, both keep metering until check-out, and all their metered minutes count. Scheduling maintenance that overlaps a lockout is allowed, and a lockout never causes `reservation_conflict`.

*Applies to the bookings, reporting and safety packages.*

The member checks in when starting to use the machine and checks out when done. The program meters the minutes in between and charges them at the rate captured at check-in, with the member's tier discount.

*Applies to the bookings package.*

A new reservation is accepted only when all of the following hold. These rules connect members, training, equipment and time:

1. its slot follows the booking granularity and duration limits and starts at or after the request time;
2. the slot lies inside the opening hours of its start date;
3. the member is in good standing (membership);
4. the member is not blocked by a long-overdue invoice (blocked members);
5. the member holds a certification for the machine's family that is not revoked (certifications);
6. no scheduled maintenance window of the machine overlaps the slot (maintenance);
7. no lockout of the machine overlaps the slot (lockouts);
8. no other active reservation of the machine overlaps the slot;
9. no other active reservation of the member, on any machine, overlaps the slot.

*Applies to the bookings package.*

A reservation books one machine for one member for an interval `[start, end)` on one day. Its stored status moves from `reserved` to `checkedIn` to `completed`, or from `reserved` to `cancelled`. Its reported status is the stored status, except that a stored `reserved` reservation whose `end` is at or before the request time is reported as `missed`. A missed reservation can no longer be checked in or cancelled and causes no charge.

A reservation is **active** when its reported status is `reserved` or `checkedIn`. Only active reservations block a machine or a member. Conflicts are judged on the reserved interval `[start, end)`, not on the actual check-in and check-out times.

*Applies to the bookings, equipment and reporting packages.*

Enumeration reservation status: The status of a reservation; see reservations for stored and reported status.

- `reserved`
- `missed`
- `checkedIn`
- `completed`
- `cancelled`

| Code | Meaning |
| --- | --- |
| `check_in_too_early` | The reservation has not started yet. |
| `invalid_slot` | A reservation does not follow the booking granularity or its duration limits. |
| `machine_booked` | Another active reservation of the machine overlaps the reservation. |
| `machine_in_maintenance` | A maintenance window of the machine overlaps the reservation. |
| `machine_locked` | A lockout of the machine overlaps the reservation. |
| `member_double_booked` | Another active reservation of the member overlaps the reservation. |
| `outside_opening_hours` | A reservation is not inside the opening hours of its day. |

#### Record: lockout

A lockout of one incident's machine, generated identifier `lock-N`.

Identified by `lockoutId`.

| Field | Type |
| --- | --- |
| `lockoutId` | identifier |
| `incidentId` | identifier, an existing incident |
| `machineId` | identifier, an existing machine |
| `start` | timestamp |
| `end` | timestamp |
| `cancelledReservationIds` | list of identifier, an existing reservation |

#### Record: opening hours of day

The opening hours of one day.

| Field | Type |
| --- | --- |
| `open` | clock time |
| `close` | clock time |

#### Record: reservation

A reservation of one machine for one member, generated identifier `res-N`. `checkedInAt` and `rateCentsPerMinute` are `null` until check-in; `checkedOutAt`, `meteredMinutes` and `amountCents` are `null` until check-out.

Identified by `reservationId`.

| Field | Type |
| --- | --- |
| `reservationId` | identifier |
| `memberId` | identifier, an existing member |
| `machineId` | identifier, an existing machine |
| `start` | timestamp |
| `end` | timestamp |
| `status` | enumeration `reserved`, `missed`, `checkedIn`, `completed` or `cancelled` |
| `checkedInAt` | timestamp, nullable |
| `checkedOutAt` | timestamp, nullable |
| `rateCentsPerMinute` | integer 1 to 10000, nullable |
| `meteredMinutes` | integer 0 or more, nullable |
| `amountCents` | integer 0 or more, nullable |

#### `cancelReservation`

Mutation. The reservation gets status `cancelled`. There is no fee and no credit.

- Arguments: `reservationId` (identifier, an existing reservation), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the reported status is not `reserved`.
- Result: `reservation` (reservation record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `reservationId`.

#### `checkIn`

Mutation. Status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the machine's current rate. A missed, cancelled, checked-in or completed reservation is refused with `invalid_state`.

- Arguments: `reservationId` (identifier, an existing reservation), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the reported status is not `reserved`; `check_in_too_early` when the request time is before `start`; `member_suspended` when the member is not in good standing; `not_certified` when the member has no certification for the machine's family that is not revoked.
- Result: `reservation` (reservation record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `reservationId`.

#### `checkOut`

Mutation. Status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. A usage charge is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the metered minutes, `unitPriceCents` the captured rate, `discountPercent` the discount of the member's tier at check-out, `at` the request time. A charge of 0 cents is still created. The reservation's `amountCents` is the charge's amount. Standing is not checked.

- Arguments: `reservationId` (identifier, an existing reservation), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the status is not `checkedIn`.
- Result: `reservation` (reservation record); `charge` (charge record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `reservationId`.

#### `createReservation`

Mutation. Creates the reservation with status `reserved` and the next reservation identifier.

Examples: on Sunday `2026-03-08`, `12:00`–`12:15` is accepted and `11:45`–`12:15` is refused with `outside_opening_hours`. On Friday `2026-03-06`, `19:45`–`20:00` is inside opening hours and `19:45`–`20:15` is not. On Tuesday `2026-03-03`, `10:15`–`10:30` is accepted and `10:10`–`10:30` is refused with `invalid_slot`.

- Arguments: `memberId` (identifier, an existing member); `machineId` (identifier, an existing machine); `start` (timestamp); `end` (timestamp), besides `actor` and `at`.
- Refusals, in this order: `invalid_interval` when `end` is not after `start`; `invalid_slot` when `start` or `end` is not on a 15-minute boundary or the length is outside 15 to 240 minutes; `starts_in_past` when `start` is before the request time; `outside_opening_hours` when the slot is not inside the opening hours of its start date; `member_suspended` when the member is not in good standing; `account_overdue` when the member is blocked; `not_certified` when the member has no certification for the machine's family that is not revoked; `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot; `machine_locked` when a lockout of the machine overlaps the slot; `machine_booked` when another active reservation of the machine overlaps the slot; `member_double_booked` when another active reservation of the member overlaps the slot.
- Result: `reservation` (reservation record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `new reservationId`.

#### `getMachineSchedule`

Read. Shows one machine's day: `openingHours` holds that day's opening and closing times; every day has opening hours; `reservations` are the machine's reservations whose `start` falls on that date, in every status except `cancelled`; `maintenance` are the machine's scheduled windows that overlap the interval from `00:00` of that date to `00:00` of the next date; `lockouts` are the machine's lockouts that overlap the same interval.

- Arguments: `machineId` (identifier, an existing machine); `date` (date), besides `actor` and `at`.
- Result: `machineId` (identifier); `date` (date); `openingHours` (opening hours of day record); `reservations` (list of reservation record); `maintenance` (list of maintenance window record); `lockouts` (list of lockout record).
- Permissions: member all, staff all, admin all, steward all.

#### `listMemberReservations`

Read. Returns all of the member's reservations in every status.

- Arguments: `memberId` (identifier, an existing member), besides `actor` and `at`.
- Result: `memberId` (identifier); `reservations` (list of reservation record).
- Permissions: member own, staff all, admin all, steward none.

#### `lockMachine`

Mutation. Locks the machine of an open incident:

1. A lockout is created for the incident's machine, with the next lockout identifier.
2. Every reservation of that machine whose reported status is `reserved` and whose interval overlaps the lockout becomes `cancelled`. These are the reservations of the window that are not yet used: not checked in and not missed. This includes a reservation whose start has already passed but which is not yet checked in and not yet missed. Reservations that are checked in, completed, missed or already cancelled are not changed.
3. For each reservation cancelled this way, the reservation's member receives a lockout credit of 1000 cents: the member's credit balance increases by 1000, whatever the member's standing.

The request writes exactly one audit entry, for `lockMachine`; the cancellations and credits are part of that request and have no entries of their own. `cancelledReservations` are the reservations cancelled by this request, after the change, in creation order. Lockout credits granted by a steward take effect as for staff; this is a safety action with a fixed consequence, not billing access.

- Arguments: `incidentId` (identifier, an existing incident); `start` (timestamp); `end` (timestamp), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the incident is resolved; `invalid_interval` when `end` is not after `start`; `starts_in_past` when `start` is before the request time.
- Result: `lockout` (lockout record); `cancelledReservations` (list of reservation record).
- Permissions: member none, staff all, admin all, steward all.
- Audit target: `new lockoutId`.

## Materials

Material purchases get a bulk discount that depends on the quantity of the single purchase:

| Quantity of the purchase | `discountPercent` |
| --- | --- |
| 1 to 9 | 0 |
| 10 to 24 | 5 |
| 25 or more | 12 |

Quantities of separate purchases, also by the same member or of the same material, are never added together. The table is the same for every material and every member; the membership tier plays no part in it, the tier's usage discount never applies to materials, and the bulk discount never applies to usage.

Examples, with the money formula: 9 units at 199 give `(9 × 199 × 100 + 50) div 100` = 1791; 10 units at 199 give `(10 × 199 × 95 + 50) div 100` = 1891; 25 units at 37 give `(25 × 37 × 88 + 50) div 100` = 814.

*Applies to the materials package.*

The shop sells materials by unit. Each material has a unit price and a stock count, which starts at 0. Stock never goes below 0.

*Applies to the materials package.*

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| purchase quantity | integer | 1 to 10000 | The number of units bought in one request. |
| restock quantity | integer | 1 to 100000 | The number of units added to stock in one request. |
| unit price cents | integer (cents) | 1 to 1000000 | The price of one unit of a material. |

| Code | Meaning |
| --- | --- |
| `insufficient_stock` | The material's stock is lower than the requested quantity. |

#### Record: material

A material sold by unit, with a unit price and a stock count.

Identified by `materialId`.

| Field | Type |
| --- | --- |
| `materialId` | identifier |
| `name` | text |
| `unitPriceCents` | integer 1 to 1000000 |
| `stock` | integer 0 or more |

#### Record: purchase

A sale of material to a member, generated identifier `pur-N`.

Identified by `purchaseId`.

| Field | Type |
| --- | --- |
| `purchaseId` | identifier |
| `memberId` | identifier, an existing member |
| `materialId` | identifier, an existing material |
| `quantity` | integer 1 to 10000 |
| `unitPriceCents` | integer 1 to 1000000 |
| `discountPercent` | integer 0 or more |
| `amountCents` | integer 0 or more |
| `at` | timestamp |

#### `addMaterial`

Mutation. Creates the material with stock 0.

- Arguments: `materialId` (identifier); `name` (text); `unitPriceCents` (integer 1 to 1000000), besides `actor` and `at`.
- Refusals, in this order: `already_exists` when a material with `materialId` exists.
- Result: `material` (material record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `materialId`.

#### `listMaterials`

Read. Returns all materials.

- Arguments: none, besides `actor` and `at`.
- Result: `materials` (list of material record).
- Permissions: member all, staff all, admin all, steward all.

#### `purchaseMaterial`

Mutation. Sells material to the member. The stock decreases by `quantity`. A purchase is created with the next purchase identifier, the material's current unit price, the bulk discount of its quantity as `discountPercent`, the amount by money and `at` the request time. A material charge is created for the member: `kind` `material`, `ref` the purchase identifier, and the purchase's `quantity`, `unitPriceCents`, `discountPercent`, `amountCents` and `at`. The `material` result is the material after the change.

- Arguments: `memberId` (identifier, an existing member); `materialId` (identifier, an existing material); `quantity` (integer 1 to 10000), besides `actor` and `at`.
- Refusals, in this order: `member_suspended` when the member is not in good standing; `account_overdue` when the member is blocked; `insufficient_stock` when the stock is lower than `quantity`.
- Result: `purchase` (purchase record); `material` (material record).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `new purchaseId`.

#### `restockMaterial`

Mutation. The stock increases by `quantity`.

- Arguments: `materialId` (identifier, an existing material); `quantity` (integer 1 to 100000), besides `actor` and `at`.
- Result: `material` (material record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `materialId`.

## Billing

**Charges.** Usage, course and material charges collect on the member's account until they are invoiced. A charge belongs to the month of its `at`. A charge is unbilled until it appears on an invoice.

*Applies to the billing, bookings, training, materials and reporting packages.*

**Credit.** Each member has a credit balance, initially 0. It increases by overpayments, by credits an admin issues as a goodwill gesture and by lockout credits. Credit is applied automatically to invoices generated later, as issuing invoices describes in step 5; it is never applied to an invoice that already exists and is never paid out.

*Applies to the billing package.*

**Due date, overdue and block.** An invoice is **overdue** at a request time when its status is `open` and the request time is at or after `dueAt`. The `overdue` field reports this at the request time of the response. A member is **blocked** at a request time when at least one of the member's invoices has status `open` and the request time is more than 30 days after its `dueAt`, that is, later than `dueAt` plus 43200 minutes; at exactly `dueAt` plus 30 days the member is not yet blocked. The block ends only when each such invoice reaches status `paid`: a partial payment that leaves a balance does not end it, and since credit is never applied to an existing invoice, issuing credit does not end it either. A blocked member cannot make reservations or buy materials (`account_overdue`), whoever acts: the member, staff or an admin. The block is not a standing; existing reservations stay, and check-in, check-out, cancellation, course enrollment, payments, credits and invoice generation are not blocked.

Example: an invoice issued at `2026-04-01T08:00Z` is due at `2026-04-15T08:00Z`. While it is open, a reservation request at `2026-05-15T08:00Z` is not blocked; one at `2026-05-15T08:01Z` is refused with `account_overdue`. Apart from the block, being overdue has no effect.

*Applies to the billing, bookings, materials and reporting packages.*

Billing is organized around the invoice. An invoice collects one member's charges for one calendar month. It is due 14 days after it is issued, receives payments until it is settled, and passes any overpayment to the member's credit, which in turn settles part of the member's later invoices.

*Applies to the billing package.*

**Issuing invoices.** Invoices are generated per calendar month, for all members at once, when staff request it. Generation for a month is possible once the month has ended at the request time, and only once per month. Months can be generated in any order. For a month M, every member whose `registeredAt` is before the first minute of the month after M receives one invoice, whatever the member's standing. The invoice is built as follows:

1. `invoiceId` is the next invoice identifier. Members are processed in ascending `memberId` order.
2. `issuedAt` is the request time; `dueAt` is 14 days after `issuedAt`.
3. `lines` starts with the membership line: `kind` `membership`, `ref` the member's current tier, `quantity` 1, `unitPriceCents` the tier's monthly fee, `discountPercent` 0, `at` the request time. All of the member's unbilled charges whose `at` lies in month M follow, ordered as the order of listed results states. These charges are no longer unbilled.
4. `totalCents` is the sum of the line amounts.
5. `creditAppliedCents` is the smaller of the member's credit balance and `totalCents`. The credit balance decreases by that amount.
6. `paidCents` is 0. `balanceCents` is `totalCents − creditAppliedCents − paidCents`.
7. `status` is `paid` when `balanceCents` is 0, otherwise `open`.

Because request times never go backwards and generation needs the month to have ended, no charge can be created for a month after that month was invoiced.

Worked example: member `ada` on tier `plus` uses a machine with rate 45 from `2026-03-03T10:00Z` to `10:37`. At check-out a usage charge of 1499 cents is created. Generating `2026-03` at `2026-04-01T08:00Z` gives `ada` an invoice with the membership line (4900) and the usage line (1499): `totalCents` 6399, due `2026-04-15T08:00Z`.

*Applies to the billing and reporting packages.*

**Paying an invoice.** A payment pays one invoice. The part of it that exceeds the invoice's balance becomes credit on the member's account. Paying after the due date is allowed; an invoice that is `paid` takes no further payment.

*Applies to the billing package.*

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| charge ref | string |  | What a charge is for: the member's tier, a reservation, a purchase or a course identifier. |
| payment cents | integer (cents) | 1 to 100000000 | An amount paid or credited in one request. |

Enumeration charge kind: The kind of a charge.

- `membership`
- `usage`
- `material`
- `course`

Enumeration invoice status: The status of an invoice.

- `open`
- `paid`

| Code | Meaning |
| --- | --- |
| `account_overdue` | The member has an invoice that is more than 30 days overdue. |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |
| `month_not_ended` | The month to invoice has not ended at the request time. |

#### Record: account

A member's account summary.

| Field | Type |
| --- | --- |
| `memberId` | identifier, an existing member |
| `creditBalanceCents` | integer 0 or more |
| `openBalanceCents` | integer 0 or more |
| `overdueBalanceCents` | integer 0 or more |
| `unbilledCents` | integer 0 or more |
| `blocked` | boolean |

#### Record: charge

A charge (invoice line) on a member's account. Its amount follows money and is fixed when the charge is created.

| Field | Type |
| --- | --- |
| `kind` | enumeration `membership`, `usage`, `material` or `course` |
| `ref` | charge ref |
| `quantity` | integer 0 or more |
| `unitPriceCents` | integer 0 or more |
| `discountPercent` | integer 0 or more |
| `amountCents` | integer 0 or more |
| `at` | timestamp |

#### Record: invoice

An invoice of one member for one month, generated identifier `inv-N`.

Identified by `invoiceId`.

| Field | Type |
| --- | --- |
| `invoiceId` | identifier |
| `memberId` | identifier, an existing member |
| `month` | month |
| `issuedAt` | timestamp |
| `dueAt` | timestamp |
| `lines` | list of charge record |
| `totalCents` | integer 0 or more |
| `creditAppliedCents` | integer 0 or more |
| `paidCents` | integer 0 or more |
| `balanceCents` | integer 0 or more |
| `status` | enumeration `open` or `paid` |
| `overdue` | boolean |

#### Record: payment

A payment of one invoice, generated identifier `pay-N`.

Identified by `paymentId`.

| Field | Type |
| --- | --- |
| `paymentId` | identifier |
| `invoiceId` | identifier, an existing invoice |
| `memberId` | identifier, an existing member |
| `amountCents` | integer 1 to 100000000 |
| `appliedCents` | integer 0 or more |
| `creditedCents` | integer 0 or more |
| `at` | timestamp |

#### `generateInvoices`

Mutation. Creates the invoices as issuing invoices describes and marks the month as invoiced, even when there is no member to invoice.

- Arguments: `month` (month), besides `actor` and `at`.
- Refusals, in this order: `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`); `already_invoiced` when invoices for this month were generated before.
- Result: `month` (month); `invoices` (list of invoice record).
- Permissions: member none, staff all, admin all, steward none.
- Audit target: `month`.

#### `getAccount`

Read. Returns the member's account: `creditBalanceCents` is the member's credit balance; `openBalanceCents` is the sum of `balanceCents` over the member's invoices with status `open`; `overdueBalanceCents` is the same sum over the member's overdue invoices; `unbilledCents` is the sum of the amounts of the member's unbilled charges; `blocked` states whether the member is blocked at the request time.

- Arguments: `memberId` (identifier, an existing member), besides `actor` and `at`.
- Result: `account` (account record).
- Permissions: member own, staff all, admin all, steward none.

#### `issueCredit`

Mutation. The member's credit balance increases by `amountCents`. The credit counts as issued at the request time for the revenue report. `creditBalanceCents` is the member's credit balance after the change.

- Arguments: `memberId` (identifier, an existing member); `amountCents` (integer 1 to 100000000); `reason` (text), besides `actor` and `at`.
- Result: `memberId` (identifier); `amountCents` (integer 1 to 100000000); `creditBalanceCents` (integer 0 or more).
- Permissions: member none, staff none, admin all, steward none.
- Audit target: `memberId`.

#### `listInvoices`

Read. Returns all of the member's invoices.

- Arguments: `memberId` (identifier, an existing member), besides `actor` and `at`.
- Result: `memberId` (identifier); `invoices` (list of invoice record).
- Permissions: member own, staff all, admin all, steward none.

#### `recordPayment`

Mutation. `appliedCents` is the smaller of `amountCents` and the invoice's `balanceCents`; `creditedCents` is `amountCents − appliedCents`. The invoice's `paidCents` increases and its `balanceCents` decreases by `appliedCents`; when `balanceCents` reaches 0 its status becomes `paid`. The member's credit balance increases by `creditedCents`. A payment record is created with the next payment identifier, `memberId` the invoice's member (whoever acts) and `at` the request time. Paying after the due date is allowed. The `invoice` result is the invoice after the change; `creditBalanceCents` is the member's credit balance after the change.

- Arguments: `invoiceId` (identifier, an existing invoice); `amountCents` (integer 1 to 100000000), besides `actor` and `at`.
- Refusals, in this order: `invoice_paid` when the invoice's status is `paid`.
- Result: `payment` (payment record); `invoice` (invoice record); `creditBalanceCents` (integer 0 or more).
- Permissions: member own, staff all, admin all, steward none.
- Audit target: `new paymentId`.

## Safety

Staff record safety incidents against machines. An incident itself blocks nothing; staff can lock its machine with a lockout while it is open.

*Applies to the safety package.*

Enumeration incident status: The status of an incident.

- `open`
- `resolved`

Enumeration severity: The severity of an incident.

- `low`
- `medium`
- `high`

#### Record: incident

A safety incident recorded against a machine, generated identifier `inc-N`. `resolution` and `resolvedAt` are `null` while open.

Identified by `incidentId`.

| Field | Type |
| --- | --- |
| `incidentId` | identifier |
| `machineId` | identifier, an existing machine |
| `severity` | enumeration `low`, `medium` or `high` |
| `description` | text |
| `reportedBy` | identifier |
| `reportedAt` | timestamp |
| `status` | enumeration `open` or `resolved` |
| `resolution` | text, nullable |
| `resolvedAt` | timestamp, nullable |

#### `listIncidents`

Read. Returns all incidents.

- Arguments: none, besides `actor` and `at`.
- Result: `incidents` (list of incident record).
- Permissions: member none, staff all, admin all, steward all.

#### `reportIncident`

Mutation. Creates an incident with the next incident identifier, status `open`, `reportedBy` the actor (a steward's identifier when a steward reports it), `reportedAt` the request time, and `resolution` and `resolvedAt` `null`.

- Arguments: `machineId` (identifier, an existing machine); `severity` (enumeration `low`, `medium` or `high`); `description` (text), besides `actor` and `at`.
- Result: `incident` (incident record).
- Permissions: member none, staff all, admin all, steward all.
- Audit target: `new incidentId`.

#### `resolveIncident`

Mutation. Status `resolved`, `resolution` as supplied, `resolvedAt` the request time.

- Arguments: `incidentId` (identifier, an existing incident); `resolution` (text), besides `actor` and `at`.
- Refusals, in this order: `invalid_state` when the incident is already resolved.
- Result: `incident` (incident record).
- Permissions: member none, staff all, admin all, steward all.
- Audit target: `incidentId`.

## Reporting

Reports are read-only and are calculated from the current state at the request time.

*Applies to the reporting package.*

| Value type | Base | Range | Rule |
| --- | --- | --- | --- |
| utilization percent | integer | 0 or more | A utilization in percent; it can exceed 100. |

#### Record: utilization row

One machine's row of the utilization report.

| Field | Type |
| --- | --- |
| `machineId` | identifier, an existing machine |
| `openMinutes` | integer 0 or more |
| `maintenanceMinutes` | integer 0 or more |
| `lockoutMinutes` | integer 0 or more |
| `availableMinutes` | integer 0 or more |
| `usedMinutes` | integer 0 or more |
| `utilizationPercent` | integer 0 or more |

#### Record: utilization totals

The totals of the utilization report.

| Field | Type |
| --- | --- |
| `openMinutes` | integer 0 or more |
| `maintenanceMinutes` | integer 0 or more |
| `lockoutMinutes` | integer 0 or more |
| `availableMinutes` | integer 0 or more |
| `usedMinutes` | integer 0 or more |
| `utilizationPercent` | integer 0 or more |

#### `revenueReport`

Read. Summarizes billing for one month:

- `invoicesGenerated`: whether invoices for the month have been generated;
- `invoiceCount`: the number of invoices of the month;
- `membershipCents`, `usageCents`, `materialCents`, `courseCents`: the sums of the line amounts of each kind on the invoices of the month;
- `invoicedCents`: the sum of those four;
- `creditAppliedCents`: the sum of `creditAppliedCents` over the invoices of the month;
- `outstandingCents`: the sum of `balanceCents` over the invoices of the month at the request time;
- `paymentsCents`: the sum of `amountCents` of all payments whose `at` lies in the month, for any invoice, including the parts that became credit;
- `creditsIssuedCents`: the sum of credits issued with `issueCredit` whose request time lies in the month, plus the lockout credits of `lockMachine` requests whose request time lies in the month. Credit from overpayments is not included.

For a month without generated invoices, the invoice figures are 0 and `invoicesGenerated` is `false`; payments and issued credits are still reported.

- Arguments: `month` (month), besides `actor` and `at`.
- Result: `month` (month); `invoicesGenerated` (boolean); `invoiceCount` (integer 0 or more); `membershipCents` (integer 0 or more); `usageCents` (integer 0 or more); `materialCents` (integer 0 or more); `courseCents` (integer 0 or more); `invoicedCents` (integer 0 or more); `creditAppliedCents` (integer 0 or more); `outstandingCents` (integer 0 or more); `paymentsCents` (integer 0 or more); `creditsIssuedCents` (integer 0 or more).
- Permissions: member none, staff all, admin all, steward none.

#### `utilizationReport`

Read. Shows how much each machine was used compared with the time it was available. The report range is the interval from `00:00` of `from` to `00:00` of `to`; `to` itself is excluded. `machines` has one row per machine that exists at the request time.

- `openMinutes`: minutes of the range that lie inside opening hours. Every machine gets the full count, regardless of when it was added.
- `maintenanceMinutes`: minutes of the range that lie inside opening hours and inside at least one scheduled maintenance window of the machine. Overlapping windows count once.
- `lockoutMinutes`: minutes of the range that lie inside opening hours and inside at least one lockout of the machine. Overlapping lockouts count once.
- `availableMinutes`: `openMinutes` minus the minutes of the range that lie inside opening hours and inside at least one scheduled maintenance window or lockout of the machine, each minute counted once. A minute covered by both appears in `maintenanceMinutes` and `lockoutMinutes` but is subtracted once.
- `usedMinutes`: for every completed reservation of the machine, the minutes of `[checkedInAt, checkedOutAt)` that lie in the range, summed. Opening hours and lockouts do not limit this count. Reservations that are checked in but not checked out are not counted.
- `utilizationPercent`: `(usedMinutes × 100 × 2 + availableMinutes) div (availableMinutes × 2)`, which is `usedMinutes × 100 / availableMinutes` rounded half up to an integer, or 0 when `availableMinutes` is 0. It can exceed 100.
- `totals`: the sums of `openMinutes`, `maintenanceMinutes`, `lockoutMinutes`, `availableMinutes` and `usedMinutes` over all rows, and `utilizationPercent` computed from these sums with the same formula. With no machines, all totals are 0.

- Arguments: `from` (date); `to` (date), besides `actor` and `at`.
- Refusals, in this order: `invalid_interval` when `to` is not after `from` or the range is longer than 366 days.
- Result: `from` (date); `to` (date); `machines` (list of utilization row record); `totals` (utilization totals record).
- Permissions: member none, staff all, admin all, steward none.

## Delivery

`realization/README.md` explains how to build, run and test the program, shows a short example session and states the scope. Code, tests and documentation agree with this document.

*Applies to every package.*

The tests run with `go test ./...` from `realization/`. They cover, for every operation, a successful request and each refusal code the operation lists; the four policies, including the permission matrix and audit entries; the boundary rules of time (half-open intervals, opening hours, granularity and duration limits) and of money (rounding per line); and the cross-area rules of reservations and invoices. Tests and documentation do not expect or describe an expiry of certifications; a certification issued long before a reservation date is still accepted.

*Applies to every package.*
