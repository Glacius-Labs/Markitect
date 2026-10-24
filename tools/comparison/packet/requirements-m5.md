# Small library requirements

This document describes the small library application to build. Links point to the related sections.

## Library

One small library manages registered [members](#member), physical [book copies](#copy) and [loans](#loan). A trusted operator registers members and copies, borrows, renews and returns copies, inspects current state and requests [overdue notices](#assess-overdue-loans). A member is domain data, not a login account. A new library starts empty; registration is ordinary supported behavior, not hardcoded example data. Operations are processed sequentially against current state. Accepted changes remain available to later operations in that state; persistence across process restarts is outside this example. There are no reservations, member suspensions, fees, lost or damaged books, branches, authentication, automatic scheduling or message delivery. Records are not edited or deleted except for a loan's due date and renewal count through renewal, and loan return state. [The shared input rule](#shared-input-rule) governs all operations and [the delivery rule](#delivery) governs the delivered implementation, tests and documentation.

## Shared input rule

Every operation of [the library](#library) requires its stated inputs. Identifiers are nonempty opaque values compared exactly; differently spelled identifiers are not silently combined. Names and titles contain at least one non-whitespace character. Dates are explicitly supplied valid calendar dates without times or time zones; behavior does not depend on the machine clock. Missing required inputs, empty identifiers, invalid dates and unknown requested records are refused with an understandable reason and no observable state change. Invalid requests do not partially register records or change loans, and expected refusals do not produce unhandled crashes. When multiple conditions fail, reporting any actual failing condition is sufficient. Exact error text, transport and identifier generation are implementation choices. Successful outcomes and refusals are observable through operation results and state inspection. Reads change no state; no ordering of returned loan lists is required.

## Delivery

Across [the library](#library), deliver a callable local implementation of the operations concerning [members](#member), [copies](#copy), [loans](#loan) and [overdue assessment](#assess-overdue-loans), automated tests and concise usage documentation. These delivery obligations apply to every listed area. Tests cover successful operations, expected refusals, borrowing-limit and due-date boundaries for both loan classes, renewals and repeated returns. Documentation explains how to run and test the implementation and its scope. Code, tests and documentation must agree with this document. No user interface, deployment or production database is required. [The callable interface](#callable-interface) fixes the program boundary, language, build and test commands and wire format. Frameworks, internal source layout and domain design remain realization choices; no architecture is inferred from the historical library sketch.

## Callable interface

The [library](#library) is delivered as a local Go program that speaks JSON Lines and
offers every operation in this document; [the shared input rule](#shared-input-rule) governs all of them.

### Program and build boundary

The program reads one request per line on standard input and writes exactly
one response per request on standard output, in request order. It writes and
flushes each response before it reads the next request. One process
holds one initially empty library; operations run sequentially on that
process's state. Closing the input ends the process with exit code zero.
There is no network service, machine-clock input, database or persistence
across processes.

The Go module lives at `realization/go.mod`, with a thin executable entry
point at `realization/cmd/library`. From `realization/`, `go build ./cmd/library`
builds the program and `go test ./...` runs its tests. Only the Go standard
library is used. Internal packages, domain design and test layout are
realization choices. The program needs no files outside `realization/` and
offers no reset, state-injection, policy-override or other test-only
operation; its policy comes only from its realization of
[the lending policy](#lending-policy).

### Wire format

Lines are UTF-8 JSON objects. Standard output carries responses only;
diagnostics go to standard error, and there is no startup banner or
unsolicited output. The contract identifier is `markitect.library-jsonl/v1`.

```json
{"id":"q1","op":"registerMember","args":{"memberId":"member-1","name":"Ada"}}
{"id":"q1","status":"ok","result":{"member":{"memberId":"member-1","name":"Ada"}}}
```

`id` is a nonempty request correlation string, echoed exactly; it is neither
a domain identifier nor an idempotency key. `op` names one operation below
and `args` is an object; a request without `args`, or with `args` that is not
an object, has an invalid envelope. Field and operation names are
case-sensitive. Required arguments must be present with the stated type.
Responses may carry additional fields.

An expected domain failure returns `status: refused` with a nonempty,
understandable `reason` and no `result`. Exact wording, error codes and the
precedence of simultaneous failures are free. Missing or invalid arguments of
a known operation are refusals: a required argument that is missing, `null`
or has an invalid type or value. The process continues and the observable
state is unchanged.

```json
{"id":"q2","status":"refused","reason":"The requested copy is already on loan."}
```

A malformed JSON line, an invalid request envelope or an unknown operation
returns `status: invalidRequest` with a nonempty `reason`, echoing the
request `id` when the line parses as an envelope with a string `id`, and
`null` otherwise. Unknown request or argument fields also make a request
invalid: an otherwise valid request of a known operation that carries an
unknown field, or a field whose name differs from a stated one only in case,
returns `status: invalidRequest`, not a refusal. Processing continues
with the next line without changing state. A crash, hang, extra response or
success-shaped error violates this interface.

Identifiers are nonempty strings compared exactly, never trimmed, case-folded
or normalized. Names and titles contain a non-whitespace character and are
returned as supplied. Dates
use `YYYY-MM-DD` in Gregorian years 0001 to 9999, without times or time
zones.

### Records

| Record | Fields |
| --- | --- |
| [member](#member) | `memberId`, `name` |
| [copy](#copy) | `copyId`, `title`, `loanClass` (`standard` or `short`), `available` (boolean) |
| [loan](#loan) | `loanId`, `memberId`, `copyId`, `borrowDate`, `dueDate`, `status` (`open` or `closed`), `returnDate` (`null` while open, the return date once closed), `renewals` (integer, `0` until renewed) |
| Overdue notice | `loanId`, `memberId`, `copyId`, `dueDate` |

Identifiers and dates are strings and counts are integers. Lists may come in
any order. A loan identifier is generated by the program; it is nonempty,
unique within the library and stable, and its format is free.

### Operations

Every success returns `status: ok` and the `result` fields below, describing
the state after the operation. Inspection changes no state. A refusal is
visible in the response and in later inspection.

| Operation | Required `args` | Successful `result` | Use case |
| --- | --- | --- | --- |
| `registerMember` | `memberId`, `name` | `member` | [member registration](#register-a-member) |
| `inspectMember` | `memberId` | `member` | [member inspection](#inspect-a-member) |
| `registerCopy` | `copyId`, `title` | `copy` | [copy registration](#register-a-copy) |
| `inspectCopy` | `copyId` | `copy` | [copy inspection](#inspect-a-copy) |
| `borrow` | `memberId`, `copyId`, `borrowDate` | `loan` | [borrowing](#borrow) |
| `return` | `loanId`, `returnDate` | `outcome` (`returned` or `alreadyReturned`), `loan` | [return](#return) |
| `inspectLoan` | `loanId` | `loan` | [loan inspection](#inspect-loans) |
| `inspectOpenLoans` | `memberId` | `memberId`, `count`, `loans` (open loans) | [loan inspection](#inspect-loans) |
| `renew` | `loanId`, `renewalDate` | `loan` | [renewal](#renew) |
| `assessOverdue` | `assessmentDate` | `notices` (overdue notices) | [overdue assessment](#assess-overdue-loans) |

`registerCopy` also accepts an optional `loanClass` string, `standard` or
`short`; without it the copy is standard.

Returning an already returned loan succeeds with `outcome: alreadyReturned`
and keeps the original return date; the supplied return date must still be
valid and not before the borrowing date. A loan is always returned by its
loan identifier, never by its copy identifier.

## Members

### Member

A registered member of [the library](#library) has a unique stable identifier and a nonempty name. Different members may have the same name. Identifier and name follow [the shared input rule](#shared-input-rule) and remain unchanged after registration. Members have no status: the library does not suspend members, and every registered member may borrow according to [the eligibility rule](#eligibility). This describes the member concept; it does not pre-create any live members.

### Eligibility

Only a registered [member](#member) may start [a new loan](#borrow) or [renew a loan](#renew). There is no suspension: membership never blocks borrowing, renewal or [return](#return) once the member is registered, and all other borrowing and renewal conditions still apply. Lending must enforce this membership rule when admitting a new loan or a renewal, rather than relying on it being checked somewhere else.

### Register a member

The operator supplies a new identifier and name under [the shared input rule](#shared-input-rule) to register a [member](#member) in [the library](#library). Success creates exactly one member with those values. Reusing an existing member identifier is refused without replacing the existing record or changing other state.

### Inspect a member

Given a known [member](#member) identifier, return that member's identifier and name. Apply [the shared input rule](#shared-input-rule) to unknown or invalid identifiers. Inspection is read-only and never creates a member.

## Copies

### Copy

A copy is one physical book in [the library](#library), with a unique stable identifier, a nonempty title under [the shared input rule](#shared-input-rule) and a loan class, either standard or short. Different copies may have the same title and are borrowed independently. Identifier, title and loan class remain unchanged after registration. The loan class selects the loan period under [the lending policy](#lending-policy). There is no separate title or edition catalogue. The copy's current availability follows [the availability rule](#availability).

### Availability

A registered [copy](#copy) is available exactly when it has no open [loan](#loan). Across all members, at most one open loan may refer to the same copy. [Borrowing](#borrow) must enforce availability and makes the copy unavailable by creating an open loan. [Return](#return) releases the copy only when it closes that copy's current open loan. Availability is derived from loans; there is no independent stock count. Returning an old closed loan cannot release a copy now held by a newer open loan.

### Register a copy

The operator supplies a new identifier, a title and optionally a loan class under [the shared input rule](#shared-input-rule) to register one [copy](#copy) in [the library](#library). Without a loan class the copy is standard; a loan class other than exactly standard or short is refused without changing any state. The copy initially has no loans and is available according to [the availability rule](#availability). Reusing an existing copy identifier is refused without replacing that record or changing any state. Matching an existing title alone is not a duplicate registration.

### Inspect a copy

Given a known [copy](#copy) identifier, return its identifier, title, loan class and current availability under [the availability rule](#availability). Apply [the shared input rule](#shared-input-rule) to unknown or invalid identifiers. Inspection changes no state and never creates a copy.

## Lending

### Lending policy

The lending policy states the numeric admission limit, the calendar-day durations and the renewal limit used for loans in its stated scope. All required values must be positive integers. The individual policy connects the operations and explains how the values apply. Values do not authorize deletion or forced return of existing loans.

| Policy value | Value | Meaning |
| --- | --- | --- |
| Maximum open loans (`maxOpenLoans`) | 3 | Positive integer giving the maximum resulting count of currently open loans per member when admitting a new loan. It is an admission limit, not permission to remove existing loans when they exceed a later lower limit. Exactly one integer value is required. |
| Loan period in calendar days (`loanPeriodDays`) | 14 | Positive integer giving the number of calendar days added to the borrowing date to determine a new loan's due date for a standard copy. Weekends and holidays do not alter this interval. This does not describe a time of day or a scheduled job. Exactly one integer value is required. |
| Short loan period in calendar days (`shortLoanPeriodDays`) | 7 | Positive integer giving the number of calendar days added to the borrowing date to determine a new loan's due date for a short copy. Weekends and holidays do not alter this interval. This does not describe a time of day or a scheduled job. Exactly one integer value is required. |
| Maximum renewals per loan (`maxRenewals`) | 1 | Positive integer giving how many times one loan can be renewed. Each renewal moves its due date forward once. Exactly one integer value is required. |

For [borrowing](#borrow), the resulting current open-loan count of the member must not exceed the value of [maximum open loans](#lending-policy) in the table above. Count every open [loan](#loan), including overdue loans, and exclude closed loans. An overdue loan adds no separate borrowing restriction. A new loan's due date is its borrowing date plus the value of [loan period](#lending-policy) for a standard [copy](#copy), or plus the value of [short loan period](#lending-policy) for a short copy, both in the table above. A loan can be renewed at most as often as the value of [maximum renewals](#lending-policy) allows; each [renewal](#renew) replaces the loan's due date with the later one, which is then the loan's current due date. [Return](#return) remains possible regardless of the current limit or overdue state. A lower limit neither deletes existing loans nor forces their return; while at or above that limit, the member cannot start another loan.

### Loan

A loan associates one registered [member](#member) with one registered [copy](#copy) in [the library](#library). A successful [borrowing](#borrow) creates its own unique stable loan identifier, member identifier, copy identifier, borrowing date and due date under [the lending policy](#lending-policy) and [the shared input rule](#shared-input-rule). It starts open and not yet renewed. A loan records how many times it was renewed; [renewal](#renew) moves its due date forward. [Return](#return) records a return date and closes it. Closed loans stay inspectable, preserve their dates and never have their identifiers reused. Loan records describe current state and retained facts, not a service for reconstructing the entire library at a historical date.

### Borrow

The operator supplies a [member](#member) identifier, a [copy](#copy) identifier and a borrowing date under [the shared input rule](#shared-input-rule) in [the library](#library). Admit the request only when the member is registered under [the eligibility rule](#eligibility), the copy is registered and available under [the availability rule](#availability), the resulting open-loan count satisfies [the lending policy](#lending-policy), and the new loan's due date falls on or before 9999-12-31, the last date of the library's date range. Success creates exactly one open [loan](#loan), returns its identifier and due date, makes the copy unavailable and increases the member's open-loan count by one. Other members and copies remain unchanged. Unknown members or copies, an already borrowed copy, a reached limit, a due date after 9999-12-31 or invalid inputs cause refusal with no loan created and no observable state change. A due date is never clamped, wrapped or given a year after 9999. Repeating a successful request for the same copy is refused because it is now borrowed; this is a new transaction, not a retry-safe command with duplicate-request detection.

### Return

The operator supplies a [loan](#loan) identifier and a valid return date on or after its borrowing date under [the shared input rule](#shared-input-rule). Return an open loan by closing exactly that loan and recording the date. The member's open-loan count decreases by one and the [copy](#copy) becomes available under [the availability rule](#availability). Neither overdue state nor the current limit in [the lending policy](#lending-policy) blocks it. There is no fee or penalty. For a closed loan, a repeated return with a valid date on or after its borrowing date reports already returned and changes nothing, preserving the original return date. If its copy was lent again, the newer loan stays open and the copy stays unavailable. An unknown loan or a date before the borrowing date is refused without changing state. A copy identifier alone is insufficient to identify the transaction being returned.

### Renew

The operator supplies a [loan](#loan) identifier and a renewal date under [the shared input rule](#shared-input-rule). Admit the renewal only when the loan is open, its [member](#member) is registered under [the eligibility rule](#eligibility), the renewal date is on or after the borrowing date and on or before the current due date, the loan has been renewed fewer times than the value of [maximum renewals](#lending-policy) in [the lending policy](#lending-policy), and the new due date falls on or before 9999-12-31. Success moves the due date forward by the loan period of the loan's [copy](#copy) under [the lending policy](#lending-policy), counted from the current due date, adds one to the loan's renewal count and returns the loan. The member's open-loan count, the copy's availability and the borrowing date are unchanged. An unknown loan, a closed loan, a renewal date before the borrowing date or after the current due date, an exhausted renewal count, a new due date after 9999-12-31 or invalid inputs cause refusal with no observable state change.

### Inspect loans

Support inspection of a known [loan](#loan) by its identifier, returning its loan, member and copy identifiers, borrowing and due dates, open or closed state, and return date when closed. Also support inspecting a known [member](#member) for its currently open loans and count, following [the lending policy](#lending-policy). An empty list and zero count are valid. Unknown or invalid requested identifiers are refused under [the shared input rule](#shared-input-rule). Both queries are read-only and describe current state; returned loan order is unrestricted.

## Overdue notices

### Assess overdue loans

Given a valid explicit assessment date under [the shared input rule](#shared-input-rule), calculate overdue notices for [the library](#library). Include exactly the currently open [loans](#loan) whose due dates, established by [the lending policy](#lending-policy), are strictly earlier than the assessment date. A loan due on that date is not yet overdue. Returned loans are excluded. Return exactly one notice per qualifying loan, identifying its loan, member, copy and due date, or an empty list when none qualify. Notice order is unrestricted. Send no message, schedule no job and change no library state. The same state and date yield the same notice contents. This is an assessment of current open loans, not a historical-state query; a loan already returned in current state remains excluded even when assessing an earlier date.
