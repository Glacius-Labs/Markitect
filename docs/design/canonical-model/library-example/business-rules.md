# Business rules for the small library

Date: 16 October 2026. Domain description confirmed by the owner and used to derive [M1](m1/model.yaml). The YAML Definitions now own intended behavior; this text preserves the agreed derivation source. The library records members, lends physical book copies, accepts returns and identifies overdue loans. These rules select no programming language, framework or source layout.

## Scope

There is one library. A trusted operator supplies registrations and performs the supported operations. Member identity is domain data, not a login account. Operations are processed one at a time against the current library state.

A new library starts empty. Members and book copies can be registered through ordinary operations; examples are not hardcoded records. Successful changes remain available to subsequent operations in the same state. Persistence across process restarts is outside this first example.

The example has no reservations, renewals, fees, lost or damaged books, multiple branches, user authentication, automatic scheduling or message delivery. Records are not edited or deleted, apart from member suspension state and loan return state. An overdue loan adds no borrowing restriction beyond being an open loan that counts toward the limit.

## Members

A member has a unique, stable identifier, a nonempty name and a status: active or suspended. Names need not be unique. Registration creates an active member. Reusing an existing identifier is refused and does not overwrite that member.

A known member can be suspended or restored to active status. Setting the current status again succeeds without another state change. An unknown member cannot be suspended or restored.

Only active, registered members may start a new loan. Suspension does not close existing loans, change their due dates or prevent their return. Restoring a member removes this restriction; the ordinary borrowing conditions still apply.

## Book copies

A copy is one physical book with a unique, stable identifier and a nonempty title. Two copies may have the same title and can be borrowed independently. There is no separate title or edition catalogue in this example.

Registering a copy makes it available. Reusing a copy identifier is refused without overwriting the existing copy. A copy is available exactly when it has no open loan. Availability is derived from loans rather than maintained as a separate stock count.

At most one open loan may refer to the same copy, across all members.

## Loans and dates

A successful borrowing creates a loan with its own unique, stable identifier, the member identifier, the copy identifier, a borrowing date and a due date. It is initially open. Returning it records a return date and closes it. Closed loans remain available for inspection; their identifiers are never reused.

The borrowing date is supplied explicitly. The due date is that date plus 14 calendar days. Weekends and holidays do not change it. For example, a loan starting on 1 October is due on 15 October. Dates are calendar dates without times or time zones; the domain does not read the machine clock.

Only open loans count toward a member's current borrowing total. Open overdue loans count too. Closed loans do not count. The count and copy availability describe the current state, not a reconstruction of historical state at a supplied date.

## Borrowing

The operator requests a loan for a member, a copy and a valid borrowing date. The request succeeds only when all of these conditions hold:

- The member is registered and active.
- The copy is registered and available.
- The member has fewer than five open loans before the request, so the resulting total is at most five.

On success, exactly one open loan is created and its identifier and due date are returned. The copy becomes unavailable and the member's open-loan count increases by one. Other members and copies remain unchanged.

An unknown member, unknown copy, suspended member, already borrowed copy or reached loan limit causes refusal. A refused request creates no loan and leaves the observable library state unchanged. If several conditions fail, reporting any actual failing condition is sufficient; error precedence is not a domain rule.

Borrowing is a new transaction, not a repeat-safe command. Repeating a successful request for the same copy is refused because that copy is now borrowed. This example has no retry keys or automatic duplicate-request detection.

## Returns

The operator returns a loan by its loan identifier and a valid return date on or after its borrowing date. A return before the borrowing date is refused. The member's current status, the due date and the current borrowing limit do not prevent a return.

Returning an open loan closes exactly that loan and records its return date. The member's open-loan count decreases by one and the copy becomes available. An overdue return has the same effect, with no fee or other penalty.

A repeated return of an already closed loan, with a valid date on or after its borrowing date, reports that it was already returned and changes nothing. It preserves the original return date. In particular, if the same copy has since been lent again, returning the old loan must not close the newer loan or make its copy available.

An unknown loan identifier is refused without changing state. Returning a copy identifier alone is not supported: the loan identifier makes the intended transaction unambiguous.

## Overdue reminders

The operator supplies a valid assessment date and receives one notice for each currently open loan whose due date is strictly earlier than that date. A loan due on the assessment date is not yet overdue. Returned loans are excluded. A suspended member's open overdue loans are included.

Each notice identifies the loan, member, copy and due date. There is exactly one notice per qualifying loan. When no loan qualifies, the result is empty. Notices can be returned in any order; the evaluation compares their contents.

This operation only calculates a list. It sends no email, schedules no job and changes no library state. Repeating it against the same state and date yields the same notices. The assessment date is used for comparison with due dates, not as a request to reconstruct an earlier library state: a loan already returned in the current state remains excluded.

## Reading state

The operator can inspect a known member and its current status, a known copy and its current availability, a known loan including its dates and open or closed state, and a known member's open loans and count. An empty open-loan list and count zero are valid results.

An unknown requested member, copy or loan is reported as unknown rather than fabricated or silently created. Reads change no state. No ordering of returned loan lists is required.

## Failure behavior

Identifiers are nonempty opaque values compared exactly; two differently spelled identifiers are not silently combined. Names and titles must contain at least one non-whitespace character. Missing required inputs, empty identifiers and invalid calendar dates are refused. No invalid request partially registers a record or changes a loan.

Expected refusals produce an understandable reason, such as an unknown member or reached limit, rather than an unhandled crash. Exact error wording, transport and identifier generation are implementation choices. Accepted operations and refusals must remain observable through their results and state inspection.

## Worked cases

Each row describes an independent case with the stated setup. A request for another book uses a distinct available copy unless the row says otherwise. These cases explain the rules; they are not an executed test report.

| Setup and operation | Expected result |
| --- | --- |
| Register a new member, then register the same identifier again. | First registration succeeds and creates an active member; the second is refused and preserves the original. |
| Register two copies with the same title and different identifiers. | Both are available and can be borrowed independently. |
| An active member with four open loans borrows another copy. | Accept; create one loan, count becomes five, requested copy becomes unavailable. |
| An active member with five open loans borrows another copy. | Refuse; count, loans and copy availability remain unchanged. |
| A suspended member with no loans requests an available copy. | Refuse. Restoring the member makes a later request eligible under the other rules. |
| Borrow with an unknown member or an unknown copy. | Refuse without creating records. |
| A different member requests a copy that already has an open loan. | Refuse; the existing loan is unchanged. |
| Borrow on 1 October. | The returned loan has a due date of 15 October. |
| Suspend a member who has an open loan, then return that loan. | Suspension preserves the loan; return succeeds and releases the copy. |
| Return an open loan with a date before its borrowing date. | Refuse; the loan stays open and the copy unavailable. |
| Return a loan, lend the same copy again, then return the old loan again. | The last operation reports already returned; the new loan remains open and the copy unavailable. |
| Assess an open loan due on 15 October on the 15th, then on the 16th. | No notice on the 15th; exactly one notice on the 16th. |
| Assess the same overdue loan after it has been returned. | No notice; calculation does not change state. |
| Assess a suspended member's open overdue loan. | Include it; suspension does not erase an outstanding loan. |

## Cross area effects

The planned Package responsibilities must be connected in M1. A statement in one area that requires another area's behavior needs a visible model relationship; no Manager may treat its local text as the complete system.

| Area | Shared meaning needed by another area |
| --- | --- |
| Members and lending | Registration and active status gate new borrowing. Suspension preserves open loans and permits returns. |
| Catalog and lending | A copy's identity and availability are shared. One open loan makes the copy unavailable; closing it releases the copy. |
| Lending and lending/reminders | Open or closed state and due dates determine notices. Assessment is read-only and uses the current loan state. |

These are domain relationships. They do not prescribe module APIs, descendant files to edit or additional Manager roles.

Return to the [example overview](README.md).
