# Library callable contract

Date: 16 October 2026. Proposed realization interface for the first [two-Apply Playground](../manager-apply-playground.md). This fixes the interface choice deliberately left open by [library/realization](m1/model/library/realization.yaml); it does not change M1's business rules or claim an implemented program.

## Program and build boundary

Use a local JSON Lines program: one request on standard input, one response on standard output, in request order. One process represents one initially empty library. Operations are sequential and share that process's state. Closing input ends the process normally with exit code zero. Start a fresh process for each independent case. There is no network service, machine-clock input, database or state persistence requirement.

Choose Go for this first example. The generated repository contains a Go module at `realization/go.mod` and a thin executable entrypoint at `realization/cmd/library`. From `realization/`, the external runner builds with `go build -o <absolute-output-path> ./cmd/library` and runs generated tests with `go test ./...`. The Go toolchain is pinned by the experiment manifest before execution. Standard-library-only dependencies keep this first example self-contained. Internal packages, domain design and test layout remain realization choices; Markitect's own Go architecture does not prescribe the library's internals.

The model, public interface, execution configuration and dependency inputs are read-only Apply inputs. Managers produce code, tests and concise usage documentation below `realization/`, with ordinary registry ownership. The public contract is supplied as a frozen, shared realization constraint, not inferred from hidden tests. It is not a handwritten library implementation or a new Core Kind.

The built program runs from a copy of `realization/`, with no authoring model, Markitect journal, evaluator source or expected-result files mounted. Any data/configuration it needs must be part of its registered realization. It receives no M1/M2 label or policy override. Its observed policy must come from the realization produced by Apply. No reset, state injection, policy-change or evaluation-only operation is provided.

## Wire format

The contract identifier is `markitect.library-jsonl/v1`, bound by the experiment manifest rather than sent as a business request. UTF-8 lines contain JSON objects. Standard output contains responses only; diagnostics use standard error. No startup banner or unsolicited output is allowed. Each complete input line gets exactly one complete response line before the next request is sent.

```json
{"id":"q1","op":"registerMember","args":{"memberId":"member-1","name":"Ada"}}
```

```json
{"id":"q1","status":"ok","result":{"member":{"memberId":"member-1","name":"Ada","status":"active"}}}
```

`id` is a nonempty request correlation string, echoed exactly. It is not a domain identifier or an idempotency key. Reusing it does not change an operation's meaning. `op` names one operation below; `args` is an object. Field and operation names are case-sensitive. Required domain arguments must be present with the stated type. Unknown request/argument fields are invalid requests; response extensions are allowed and ignored by this contract's evaluator.

Expected domain failures have `status: refused` and a nonempty, understandable `reason`, with no `result`. No exact wording, error-code taxonomy or precedence between simultaneous failures is required. Missing/invalid domain arguments in a recognized operation are refusals. The process continues and observable library state remains unchanged.

```json
{"id":"q2","status":"refused","reason":"The requested copy is already on loan."}
```

A malformed JSON line, invalid request envelope or unknown operation instead returns `status: invalidRequest` and a nonempty `reason`. Echo a valid request `id` if recoverable from a parsed envelope, otherwise use `id: null`. Continue at the next line without changing library state. This is transport behavior, not another library rule. A crash, hang, extra response or success-shaped error fails the callable contract.

Domain identifiers are nonempty strings compared exactly; do not trim, case-fold or normalize them. Names and titles must contain a non-whitespace character and are returned as supplied. Member status is exactly `active` or `suspended`. Dates use `YYYY-MM-DD`, Gregorian calendar years 0001 through 9999, without times or time zones. Evaluation uses dates whose required arithmetic stays within that range. This chooses a wire representation for M1's explicit calendar dates.

## Records and operations

| Record | Required fields |
| --- | --- |
| Member | `memberId`, `name`, `status` |
| Copy | `copyId`, `title`, `available` (boolean) |
| Loan | `loanId`, `memberId`, `copyId`, `borrowDate`, `dueDate`, `status` (`open` or `closed`), `returnDate` (null while open, date while closed) |
| Overdue notice | `loanId`, `memberId`, `copyId`, `dueDate` |

Identifiers and dates in records are strings. Counts are integers. Lists may be in any order. A loan ID is generated by the implementation, is nonempty, is unique across that library's loans and remains stable. The evaluator captures returned IDs and uses symbolic references to them; no UUID format, deterministic generation or particular value is required.

Every successful response has `status: ok` and the `result` fields below. Records describe the state after the operation. Inspection changes no state. Refusals must be observable both in the response and in subsequent inspection.

| Operation | Required `args` | Successful `result` | Canonical basis |
| --- | --- | --- | --- |
| `registerMember` | `memberId`, `name` | `member` | [register-member](m1/model/members/register-member.yaml) |
| `setMemberStatus` | `memberId`, `status` | `member` | [set-member-status](m1/model/members/set-member-status.yaml) |
| `inspectMember` | `memberId` | `member` | [inspect-member](m1/model/members/inspect-member.yaml) |
| `registerCopy` | `copyId`, `title` | `copy` | [register-copy](m1/model/catalog/register-copy.yaml) |
| `inspectCopy` | `copyId` | `copy` | [inspect-copy](m1/model/catalog/inspect-copy.yaml) |
| `borrow` | `memberId`, `copyId`, `borrowDate` | `loan` | [borrow](m1/model/lending/borrow.yaml) |
| `return` | `loanId`, `returnDate` | `outcome` (`returned` or `alreadyReturned`), `loan` | [return](m1/model/lending/return.yaml) |
| `inspectLoan` | `loanId` | `loan` | [inspect-loans](m1/model/lending/inspect-loans.yaml) |
| `inspectOpenLoans` | `memberId` | `memberId`, `count`, `loans` (open Loan records) | [inspect-loans](m1/model/lending/inspect-loans.yaml) |
| `assessOverdue` | `assessmentDate` | `notices` (Overdue notice records) | [assess-overdue](m1/model/lending/reminders/assess-overdue.yaml) |

An already returned loan is a successful repeat with `outcome: alreadyReturned`; it preserves the original return date. The ordinary validation rules still apply, including a valid supplied date on or after the borrowing date. Returning a known loan never substitutes a copy ID for its loan ID.

```json
{"id":"q3","op":"borrow","args":{"memberId":"member-1","copyId":"copy-1","borrowDate":"2026-10-01"}}
```

```json
{"id":"q3","status":"ok","result":{"loan":{"loanId":"generated-loan-a","memberId":"member-1","copyId":"copy-1","borrowDate":"2026-10-01","dueDate":"2026-10-15","status":"open","returnDate":null}}}
```

This example assumes `copy-1` was registered and available. `generated-loan-a` illustrates an opaque returned value, not an expected literal. The request may be repeated with another available copy; a repeated borrow of the same currently loaned copy is refused. Error reasons remain free text.

## Contract completion and limits

The [independent evaluation contract](evaluation-contract.md) owns expected behavior and coverage; this document owns the callable shapes. Both must be frozen before real Apply. There is no automatic extraction of expected results from generated code or tests.

Runtime state belongs to one program process. A2 inherits A1's repository artifacts and registry, not a live A1 library process. This contract does not claim data migration across versions, preservation after a restart or a hot policy update. The evaluator must state the resulting [limit on above-limit loan preservation evidence](evaluation-contract.md#existing-loans-and-the-process-boundary) explicitly. Adding checkpoint/restore merely to make that case reachable would expand the first example's requirements.

Return to [M1](README.md) or the [Playground contract](../manager-apply-playground.md).
