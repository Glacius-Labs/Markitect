# Small library

A local Go program that manages one small library: registered members,
physical book copies and loans, with overdue assessment. It reads JSON Lines
requests on standard input and writes one JSON Lines response per request on
standard output (contract `markitect.library-jsonl/v1`). It uses only the Go
standard library.

## Build and run

From this folder (`realization/`):

```sh
go build ./cmd/library
./library          # on Windows: library.exe
```

The program reads one request per line, writes and flushes the response, and
then reads the next line. Closing standard input ends the program with exit
code zero. Diagnostics go to standard error; there is no startup banner.

```json
{"id":"q1","op":"registerMember","args":{"memberId":"member-1","name":"Ada"}}
{"id":"q1","status":"ok","result":{"member":{"memberId":"member-1","name":"Ada"}}}
```

## Test

```sh
go vet ./...
go test ./...
```

## Scope

One process holds one library, which starts empty. All state lives in memory
and ends with the process. There is no network service, database, machine
clock, authentication, reservation, fee or message delivery, and no
test-only, reset or policy-override operation. Members and copies are never
deleted, and only a loan's due date and renewal count (through renewal) and
loan return state change.

### Requests and responses

`id` is a nonempty string that is echoed exactly. `args` must be an object.
Field, argument and operation names are case-sensitive.

- `status: ok` carries a `result` that describes the state after the
  operation.
- `status: refused` carries a nonempty `reason` and no `result`. It answers
  a domain failure: an unknown or duplicate record, a copy already on loan, a reached borrowing limit, a due date after 9999-12-31, an
  invalid date, a renewal that is not admitted, a loan class other than
  `standard` or `short`, or a required argument that is missing, `null` or
  not a string (as is a supplied `loanClass` that is `null` or not a string).
  The state is unchanged.
- `status: invalidRequest` carries a nonempty `reason`. It answers a
  malformed JSON line, an invalid envelope (`id` missing or not a string,
  `args` missing or not an object), an unknown operation, and any unknown
  request or argument field, including a known name in the wrong case. The
  `id` is echoed when the line is an object with a string `id`, and is `null`
  otherwise. Processing continues with the next line and the state is
  unchanged.

Identifiers are nonempty strings compared exactly (never trimmed, case-folded
or normalized). Names and titles need a non-whitespace character and are kept
as supplied. Dates are `YYYY-MM-DD` in the years 0001 to 9999.

### Operations

| Operation | Required `args` | `result` |
| --- | --- | --- |
| `registerMember` | `memberId`, `name` | `member` |
| `inspectMember` | `memberId` | `member` |
| `registerCopy` | `copyId`, `title`, optional `loanClass` (`standard` or `short`; `standard` when omitted) | `copy` |
| `inspectCopy` | `copyId` | `copy` |
| `borrow` | `memberId`, `copyId`, `borrowDate` | `loan` |
| `renew` | `loanId`, `renewalDate` | `loan` |
| `return` | `loanId`, `returnDate` | `outcome` (`returned` or `alreadyReturned`), `loan` |
| `inspectLoan` | `loanId` | `loan` |
| `inspectOpenLoans` | `memberId` | `memberId`, `count`, `loans` |
| `assessOverdue` | `assessmentDate` | `notices` |

Records:

- member: `memberId`, `name`
- copy: `copyId`, `title`, `loanClass` (`standard` or `short`), `available`
- loan: `loanId`, `memberId`, `copyId`, `borrowDate`, `dueDate`, `status`
  (`open` or `closed`), `returnDate` (`null` while open), `renewals` (integer,
  `0` until renewed)
- overdue notice: `loanId`, `memberId`, `copyId`, `dueDate`

Lists have no guaranteed order.

### Rules

- **Lending policy.** A member may hold at most 3 open loans; the due date is
  the borrowing date plus the loan period of the copy's loan class: 14
  calendar days for a `standard` copy, 7 for a `short` copy. Overdue loans
  count toward the limit. A due date after 9999-12-31 is refused, never
  clamped. A loan can be renewed at most once; a renewal replaces the loan's
  due date with the later one.
- **Loan class.** Registration fixes a copy's loan class; without one the copy
  is `standard`, and anything but exactly `standard` or `short` is refused
  without registering the copy.
- **Borrowing** needs a registered member and a registered, available copy. A copy is available exactly when it has no open loan.
- **Renewal** needs an open loan, a renewal date on or after the borrowing date
  and on or before the current due date, and fewer renewals than the maximum. It moves the due date forward by the loan period
  of the loan's copy, counted from the current due date, adds one to the
  renewal count and returns the loan. A new due date after 9999-12-31 is
  refused. The open-loan count, the copy's availability and the borrowing date
  do not change.
- **Membership** is the only requirement on a member: a registered member may
  borrow, renew and return whenever the other conditions hold. Members have
  just an identifier and a name.
- **Return** is by loan identifier, never by copy identifier. The return date
  must not be before the borrowing date. Returning a closed loan again
  succeeds with `alreadyReturned`, keeps the original return date and changes
  nothing; a copy lent again in the meantime stays unavailable.
- **Overdue assessment** reports exactly the open loans whose due date is
  strictly before the assessment date. It sends nothing and changes nothing.
- Loan identifiers are generated by the program (`loan-1`, `loan-2`, ...),
  unique and never reused.

## Layout

- `cmd/library`: the thin entry point.
- `internal/protocol`: the wire format, request checking and dispatch.
- `internal/library`: members, copies, loans, lending policy, overdue
  assessment.
- `internal/calendar`: the date type and its arithmetic.

Each package has its tests next to it.
