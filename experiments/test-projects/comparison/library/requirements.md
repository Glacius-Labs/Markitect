# Small library requirements

Build a local application for one small library. A trusted operator registers
members and physical book copies, changes member status, borrows and returns
copies, inspects current state and requests overdue notices. Members are domain
records, not login accounts.

## Scope and state

A new process starts with an empty library. Registrations are ordinary supported
operations; do not hardcode example records. Process operations sequentially
against current state. Successful changes remain available to subsequent
operations in that process. Persistence across process restarts is outside scope.

The application has no reservations, renewals, fees, lost or damaged books,
branches, authentication, automatic scheduling or message delivery. Records are
not edited or deleted except for member status and loan return state. No user
interface, deployment or production database is required.

The member, copy, lending and overdue operations all obey the shared input,
delivery and callable-interface requirements below. The relationships between
these areas are part of the requirements: lending enforces membership eligibility
and copy availability, returns preserve member rights and current copy state,
and overdue assessment uses current loans, their due dates and member identity.

## Members

A member has a unique, stable identifier, a name containing at least one
non-whitespace character and a status of exactly `active` or `suspended`.
Different members may have the same name. Registration creates exactly one
active member with the supplied identifier and name. Those values remain
unchanged after registration. Reusing an existing member identifier is refused
without replacing that record or changing any other state.

A known member can be suspended or restored to active status. Setting the
current status again succeeds without another change. An unknown member or a
status other than `active` or `suspended` is refused without changing state;
changing status never registers a missing member. A successful status change
preserves all other member data and every loan, including its dates and open
state.

Only registered active members may start a new loan. Lending must check this
rule when admitting each loan. Suspension prevents new borrowing and does not
close existing loans, alter their due dates or prevent returns. Restoring active
status removes the suspension restriction; all ordinary borrowing conditions
still apply.

Inspecting a known member returns its identifier, name and current status.
Inspection changes nothing and does not create a member.

## Physical copies

A copy is one physical book with a unique, stable identifier and a title
containing at least one non-whitespace character. Identifier and title remain
unchanged after registration. Different copies may have the same title and can
be borrowed independently. There is no separate title or edition catalogue.

Registration creates exactly one copy with the supplied values, initially
available with no loans. Reusing an existing copy identifier is refused without
replacing the record or changing any state. Matching a title alone is not a
duplicate registration.

A registered copy is available exactly when it has no open loan. Availability
is derived from loans; there is no independent stock count. Across all members,
at most one open loan may refer to a copy. Borrowing enforces availability by
creating an open loan. Returning a loan releases its copy only when that return
closes the copy's current open loan. Returning an old closed loan cannot release
a copy held by a newer open loan.

Inspecting a known copy returns its identifier, title and current availability.
Inspection changes nothing and does not create a copy.

## Lending policy and loans

The lending policy has two required positive integer values:

| Policy value | Value | Meaning |
| --- | --- | --- |
| Maximum open loans per member | 5 | Maximum resulting current open-loan count when admitting a new loan. |
| Loan period in calendar days | 14 | Calendar days added to the supplied borrowing date for the due date. |

Count all currently open loans, including overdue loans, and exclude closed
loans. An overdue loan adds no separate borrowing restriction. Weekends and
holidays do not alter the loan period. The policy concerns calendar dates, not
a time of day or a scheduled job.

The maximum is an admission limit. A later lower limit neither deletes existing
loans nor forces their return. A member at or above the current limit cannot
start another loan; returns remain possible regardless of the limit or overdue
state. No runtime policy override is provided.

A successful borrowing creates exactly one loan associating a registered
member with a registered copy. The application generates its own nonempty,
unique, stable loan identifier; its format is free. A loan contains that
identifier, member and copy identifiers, borrowing and due dates, open or closed
status and its return date. It starts open with a null return date. Returning it
records its return date and closes it. Closed loans remain inspectable, retain
their dates and never have their identifiers reused.

The due date is the borrowing date plus the policy's calendar-day period.
Dates are supplied explicitly; the application does not read the machine clock.
If that addition produces a due date outside the supported years 0001 to 9999,
refuse the borrowing with no observable state change, even when the supplied
borrowing date itself is valid. Use the ordinary refusal response below.
Loan counts and availability describe current state. Retained facts do not
provide a service for reconstructing the entire library at a historical date.

## Borrowing

The operator supplies a member identifier, copy identifier and borrowing date.
Admit the request only if the member is registered and active, the copy is
registered and available, and the resulting current open-loan count satisfies
the lending policy.

On success, create exactly one open loan, return its identifier and due date,
make its copy unavailable and increase the member's current open-loan count by
one. Other members and copies remain unchanged.

Unknown members or copies, suspension, an already borrowed copy, a reached
limit or invalid inputs cause refusal without creating a loan or changing any
observable state. If several conditions fail, reporting any actual failing
condition is sufficient.

Borrowing is a new transaction, without retry keys or duplicate-request
detection. Repeating a successful request for the same copy is refused because
that copy is now borrowed.

## Returns

The operator supplies a loan identifier and a valid return date on or after its
borrowing date. A copy identifier alone does not identify the loan to return.
An unknown loan or a date before the borrowing date is refused without changing
state.

Returning an open loan closes exactly that loan and records its return date.
The member's open-loan count decreases by one and the copy becomes available.
Member status, overdue state and the current loan limit do not block return.
There is no fee or penalty.

A repeated return of a closed loan succeeds with an already-returned outcome
and changes nothing, preserving its original return date. The supplied date
must still be valid and on or after the borrowing date. If the copy has since
been lent again, the newer loan remains open and the copy remains unavailable.

## Loan inspection and overdue assessment

Inspecting a known loan by identifier returns its loan, member and copy
identifiers, borrowing and due dates, open or closed status and return date.
Inspecting a known member's open loans returns that member identifier, all its
currently open loans and their count. An empty list and zero count are valid.
Unknown or invalid identifiers are refused. Both inspections change nothing;
loan list order is unrestricted.

For a valid explicit assessment date, return exactly one notice per currently
open loan whose due date is strictly earlier than that date. A loan due on the
assessment date is not overdue. Returned loans are excluded; suspension does
not remove a member's open overdue loan from the result. Each notice identifies
its loan, member, copy and due date. Return an empty list when none qualify.
Notice order is unrestricted. The same state and date yield the same contents.

Overdue assessment changes no state, schedules no job and sends no message.
It assesses current open loans. A loan already returned in current state stays
excluded even when the assessment date is earlier than its return.

## Shared input rules

Every operation requires its stated inputs with the types below. Identifiers
are nonempty opaque strings, compared exactly. Do not trim, case-fold, normalize
or silently combine differently spelled identifiers. Names and titles contain
a non-whitespace character and are returned as supplied.

Dates use `YYYY-MM-DD` and are valid Gregorian calendar dates in years 0001 to
9999, without times or time zones. Missing required inputs, empty identifiers,
invalid dates and unknown requested records are refused with an understandable
reason and no observable state change. Invalid requests never partially
register records or modify loans. Expected refusals do not crash the program.
Exact reason wording, error codes and precedence among actual failures are free.

Operation results and later state inspection make successes and refusals
observable. All reads leave state unchanged. No ordering of loan lists is
required.

## Local Go program

Deliver a local Go program using only the Go standard library. The module is
at `realization/go.mod`, with a thin executable entry point at
`realization/cmd/library`. From `realization/`, `go build ./cmd/library` builds
the program and `go test ./...` runs its tests. Internal packages, domain design,
source layout and test layout are implementation choices; no architecture is
inferred from another Library sketch.

The program needs no files outside `realization/`. It has no network service,
machine-clock input, database or persistence across processes. It offers no
reset, state-injection, policy-override or other test-only operation. Its policy
comes from this implementation of the lending requirements.

Read one request per line from standard input and write exactly one response
per request to standard output, in request order. Write and flush each response
before reading the next request. One process holds one initially empty library
and processes requests sequentially. Closing standard input ends the process
with exit code zero.

## JSON Lines interface

Lines are UTF-8 JSON objects. Standard output contains responses only; diagnostics
go to standard error. There is no startup banner or unsolicited output. The
contract identifier is `markitect.library-jsonl/v1`.

A request has exactly these envelope fields:

| Field | Type and meaning |
| --- | --- |
| `id` | Nonempty correlation string, echoed exactly. It is neither a domain identifier nor an idempotency key. |
| `op` | String naming one operation in the table below. |
| `args` | Object containing that operation's arguments. |

Field and operation names are case-sensitive. Required arguments must be
present with the stated type. Unknown envelope or argument fields invalidate
the request. Responses may contain extra fields.

A success returns `id`, `status: ok` and a `result` object with the fields below,
describing state after the operation. An expected domain failure returns `id`,
`status: refused` and a nonempty understandable `reason`, with no `result`.
Missing or invalid arguments of a known operation are refusals. Processing
continues with no state change.

A malformed JSON line, invalid request envelope or unknown operation returns
`status: invalidRequest` with a nonempty `reason`. Echo `id` when the line parses
as an envelope with a string `id`; otherwise return `id: null`. Continue with
the next line without changing state. A crash, hang, extra response or
success-shaped error violates the interface.

### Records and types

Identifiers, names, titles, statuses and dates are strings. Counts are integers.
`available` is a boolean. Lists may appear in any order.

| Record | Required fields |
| --- | --- |
| Member | `memberId`, `name`, `status` (`active` or `suspended`) |
| Copy | `copyId`, `title`, `available` |
| Loan | `loanId`, `memberId`, `copyId`, `borrowDate`, `dueDate`, `status` (`open` or `closed`), `returnDate` (`null` while open, a date string when closed) |
| Overdue notice | `loanId`, `memberId`, `copyId`, `dueDate` |

### Operations

Every listed argument is required and is a string. The result column names
fields of the successful `result` object, using the record types above.

| Operation | `args` fields | Successful `result` fields |
| --- | --- | --- |
| `registerMember` | `memberId`, `name` | `member` |
| `setMemberStatus` | `memberId`, `status` | `member` |
| `inspectMember` | `memberId` | `member` |
| `registerCopy` | `copyId`, `title` | `copy` |
| `inspectCopy` | `copyId` | `copy` |
| `borrow` | `memberId`, `copyId`, `borrowDate` | `loan` |
| `return` | `loanId`, `returnDate` | `outcome` (`returned` or `alreadyReturned`), `loan` |
| `inspectLoan` | `loanId` | `loan` |
| `inspectOpenLoans` | `memberId` | `memberId`, `count`, `loans` (open loan records) |
| `assessOverdue` | `assessmentDate` | `notices` (overdue notice records) |

Returning an already returned loan uses `status: ok` and
`outcome: alreadyReturned`, preserves its original return date and obeys the
same supplied-date validity requirement.

## Delivery

Deliver the callable implementation, automated tests and concise usage
documentation covering members, copies, loans and overdue assessment. Tests
cover successful operations, expected refusals, borrowing-limit and due-date
boundaries, suspension across membership and lending, and repeated returns.
Documentation explains how to run and test the application and its scope.
Code, tests and documentation must agree with these requirements.
