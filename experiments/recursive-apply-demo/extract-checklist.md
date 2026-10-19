# Extract assessment checklist (library)

A fixed rubric for judging an extracted library model against the human-written model. Fix it before results exist, keep it unchanged across a series, and give it to an assessor who did not take part in the runs. The reference is Library M2: the [M1 model](../../docs/design/canonical-model/library-example/m1/model) with `maxOpenLoans` 3 instead of 5. The realization is the evidence the Extract saw.

## How to score

For each obligation, give one of:
- **covered (2):** the extracted model states it as binding, in substance; wording and placement are free.
- **partial (1):** part of it is stated, or it is only implied by an example or a "basis" paragraph.
- **missing (0):** not stated.
- **wrong:** the model states something that contradicts it. Count it separately and score it 0.

Then list every **over-specification**: a binding statement that constrains a future realization beyond the reference and beyond what is observable at the program boundary or explicitly documented in the realization. Give each one a category. A statement that only restates the reference more precisely is not over-specification. A statement the model marks as a free choice ("is not specified", "is free") is not over-specification either.

| Category | Examples |
| --- | --- |
| X1 order | refusal precedence, order of lists or notices where the reference leaves it free |
| X2 internals | packages, types, function or constant names, wiring, internal views, data structures |
| X3 corner cases | parser or validation specifics that neither the reference nor the README fixes |
| X4 non-functional | concurrency, performance or resource claims |
| X5 evidence in intent | "basis" paragraphs, file names or code citations inside binding Definitions |
| X6 other | anything else that a reasonable owner would call a realization choice |

Finally, judge structure: the Packages, whether one Package owns the project-wide obligations (purpose, scope and non-goals, delivery, technical boundary), and whether the policy values live in one structured place or are repeated in prose.

## Obligations

### Project

| ID | Obligation |
| --- | --- |
| P01 | One small library manages members, physical copies and loans for a trusted operator; a member is domain data, not a login account |
| P02 | A new library starts empty; registration is ordinary behavior, not hardcoded data |
| P03 | Operations run sequentially on one process's state; nothing persists across processes |
| P04 | Non-goals: no reservations, renewals, fees, lost or damaged books, branches, authentication, automatic scheduling or message delivery |
| P05 | Records are not edited or deleted except member status and loan return state |
| P06 | Delivery: a callable local implementation, automated tests and concise usage documentation |
| P07 | Tests cover successes, expected refusals, limit and due-date boundaries, suspension across membership and lending, and repeated returns |
| P08 | Documentation explains how to run and test the implementation and its scope |
| P09 | Code, tests and documentation agree with the model |
| P10 | No user interface, deployment or production database is required |
| P11 | Go module at `realization/go.mod`, entry point `realization/cmd/library`; from `realization/`, `go build ./cmd/library` builds and `go test ./...` tests |
| P12 | Only the Go standard library is used |
| P13 | No files outside `realization/` are needed; no reset, state-injection, policy-override or other test-only operation |
| P14 | Internal packages, domain design, test layout and frameworks are stated as realization choices |

### Interface

| ID | Obligation |
| --- | --- |
| I01 | JSON Lines on standard input and output: one response per request, in request order, each written and flushed before the next request is read |
| I02 | Closing the input ends the process with exit code zero; no network service, machine clock or database |
| I03 | UTF-8 JSON objects; standard output carries responses only, diagnostics go to standard error, no banner |
| I04 | Contract identifier `markitect.library-jsonl/v1` |
| I05 | Envelope: `id` a nonempty string echoed exactly and not an idempotency key; `op`; `args` an object; case-sensitive names; unknown request or argument fields are invalid; responses may carry additional fields |
| I06 | `refused`: nonempty reason, no result, wording and precedence free; missing or invalid arguments of a known operation are refusals; state unchanged |
| I07 | `invalidRequest` for a malformed line, an invalid envelope or an unknown operation; echoes `id` when it parses as a string, `null` otherwise; processing continues; state unchanged |
| I08 | Identifiers compared exactly, never trimmed, case-folded or normalized; names and titles have a non-whitespace character and are returned as supplied; status exactly `active` or `suspended`; dates `YYYY-MM-DD`, years 0001 to 9999 |
| I09 | Record fields of member, copy, loan (with `returnDate` null while open) and overdue notice; identifiers and dates are strings, counts integers; lists in any order; loan identifier generated, unique, stable, format free |
| I10 | The ten operations with their required arguments and result fields |
| I11 | Returning a returned loan succeeds with `alreadyReturned` and keeps the original date, the supplied date must still be valid and not before the borrow date; a loan is returned by its loan identifier only |

### Behavior

| ID | Obligation |
| --- | --- |
| R01 | Missing inputs, empty identifiers, invalid dates and unknown records are refused with a reason and no observable change; no partial change; no crash |
| R02 | Member: unique stable identifier, nonempty name, status; equal names allowed; registration creates an active member; identifier and name unchanged |
| R03 | Registering an existing member identifier is refused without replacing the record |
| R04 | Setting a status: known member, `active` or `suspended`; repeating the current status succeeds; unknown member or other status refused; never registers a member |
| R05 | Inspecting a member is read-only and returns identifier, name and status |
| R06 | Eligibility: only a registered active member may borrow; suspension keeps every loan and does not prevent return; restoring removes the restriction; lending enforces it at admission |
| R07 | Copy: unique stable identifier, nonempty title; equal titles allowed; unchanged after registration; no title catalogue |
| R08 | Registering a copy makes it available; an existing copy identifier is refused; an equal title alone is no duplicate |
| R09 | Inspecting a copy is read-only and returns identifier, title and availability |
| R10 | Availability: available exactly when no open loan refers to it; at most one open loan per copy; derived from loans; returning an old closed loan never releases a copy held by a newer loan |
| R11 | Borrow admits only a registered active member, a registered available copy and a count within the policy; success creates exactly one open loan, returns identifier and due date, makes the copy unavailable, raises the count by one, changes nothing else |
| R12 | Borrow refusals leave no loan and no change; repeating a successful borrow is refused, not deduplicated |
| R13 | Policy: at most 3 open loans per member after admission, counting open loans including overdue ones and excluding closed ones; overdue adds no separate restriction |
| R14 | Due date is the borrow date plus 14 calendar days; weekends and holidays do not change it |
| R15 | A lower limit neither deletes loans nor forces their return; at or above the limit no new loan starts |
| R16 | Loan: identifier, member, copy, borrow and due date; starts open; return records the date and closes it; closed loans stay inspectable; identifiers are never reused |
| R17 | Return: a valid date on or after the borrow date closes exactly that loan, lowers the count, frees the copy; suspension, overdue state and the limit never block it; no fee |
| R18 | A repeated return reports already returned, changes nothing and keeps the original date; a newer loan of the same copy stays open |
| R19 | An unknown loan or a return date before the borrow date is refused without change; a copy identifier alone cannot identify the loan |
| R20 | Inspecting a loan and a member's open loans: the stated fields, empty list and zero count valid, read-only, order free |
| R21 | Overdue assessment: valid explicit date; open loans due strictly before it; due on the date is not overdue; returned loans excluded; suspended members' loans included; one notice per loan; order free; no message, job or change; same state and date give the same notices; current state, not history |

## Seeded scenario

Only for the `seeded` source (`rx_seed.py`). Score each item yes or no.

| ID | Expected outcome |
| --- | --- |
| S1 | The model keeps the limit of 3 open loans; no binding statement permits a fourth |
| S2 | A realization deviation names the limit Definition and the code that admits a fourth loan |
| S3 | An owner question about the loan period cites both sources (README.md 14 days, LENDING.md 21 days) |
| S4 | After the scripted answer (21 days) the model states 21 days |
| S5 | Realization deviations name the code (and README.md, tests) that still use 14 days |
| S6 | No question was asked about matters the artifacts settle |

## Assessor answer

One JSON object: `{"obligations": {"P01": {"score": 2, "where": "...", "note": "..."}, ...}, "wrong": [...], "overspecification": [{"category": "X2", "definition": "...", "statement": "..."}], "structure": {...}, "seeded": {"S1": true, ...}, "summary": "..."}`.
