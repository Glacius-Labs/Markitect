# Independent library evaluation

Date: 16 October 2026. Proposed behavioral assessment for the [callable interface](m1/model/library/callable-interface.yaml) of M1 and [two-Apply Playground](../manager-apply-playground.md). These are reviewed expectations to implement in an external evaluator, not recorded test results. M1 remains the source of business intent; this bounded experiment does not create a second specification required of every Markitect project.

## Independent inputs and oracle

Author expected outcomes from the canonical Definitions before inspecting generated implementation. Freeze two explicit expectation profiles: M1 permits five open loans; M2 permits three. Both use a fourteen-day period and the same remaining rules. Bind each profile to its actual canonical model digest. M2 changes only `spec.maxOpenLoans` in [loan-policy](m1/model/lending/loan-policy.yaml); it is materialized by the runner as a separate input, not pre-authored as another divergent model tree.

The evaluator is a separate Python standard-library process. It launches only the built library program and talks through the public JSON Lines interface. It imports neither generated code nor generated tests, does not call Markitect's domain logic, and does not calculate expected policy values by reading the implementation or whichever model happens to be current. Its data and expected outcomes stay outside all Agent write scopes. The public interface and behavioral requirements are visible to Agents; private case data is not an additional requirement.

The newer [expectation-maintenance clarification](../../../concepts/reflections/working-day-demonstration-20261016.md#keep-expectations-subordinate-to-the-model) distinguishes these frozen experiment controls from routine project tests. Routine checks may take structured parameters from the exact selected model and receive delegated, independently reviewed updates when semantics change. They remain realizations of canonical intent. This experiment keeps its deliberate five/three controls fixed; it does not require the owner to maintain another living business specification or add a second model interpreter.

The runner verifies evaluator, case and expectation hashes before and after each evaluation. It launches assessment only after all Apply sessions are terminal and their workspaces are quiescent. The library child receives neither protected files nor credentials. Process/container enforcement must be reported as actually established; a read-only mandate alone is not OS isolation. A hash mismatch invalidates the affected evidence even if assertions passed.

Use fresh, varied identifiers and names; include exact-case distinctions, two physical copies with the same title, and multiple members. Fix and record case data or the deterministic seed before the runs. The evaluator cannot demand an undocumented result ordering, error wording or loan-ID format. It checks that a reason is nonempty; independent code review checks whether refusals explain a real failing condition. Single-fault cases make that judgment clear without adding error precedence.

## Observation method

Each independent case starts a fresh empty process and uses normal registrations to arrange its state. Within a case, capture each returned loan ID as a symbolic handle, check all record fields, then query state through the inspection operations. Compare lists by unique IDs as sets, rejecting duplicates. Check both the reported count and the actual list length.

For a refusal, idempotent status update, repeated return or read-only assessment, compare the observable state before and after: all known members and copies, all previously returned loan IDs, every member's open-loan list/count and overdue notices at fixed dates. Check unrecognized IDs still refuse. This observes the API's state, not private memory; it cannot prove absence of unobservable garbage. Independent source inspection complements the protocol checks.

Request IDs are unique in normal test traces; a separate transport case establishes that repeated correlation IDs do not supply business idempotency. All time inputs are explicit. The evaluator has finite process/request/output budgets from the frozen manifest and records stdout, stderr, exit state, request/result pairs and each assertion. A timeout, malformed response or crash fails the appropriate case; it is not converted into an expected business refusal.

## Required case families

Each row expands to executable cases before real Apply. The canonical references own their full meaning; the table identifies the obligations to exercise. Shared cases run against both A1/M1 and A2/M2.

| Family | Required observations | Canonical basis |
| --- | --- | --- |
| E01 Membership | Registration creates an active record; duplicate ID refuses without replacing name/status; suspend/reactivate; setting current status succeeds unchanged; unknown member/status refuses; inspection is read-only. | [member](m1/model/members/member.yaml), [registration](m1/model/members/register-member.yaml), [status](m1/model/members/set-member-status.yaml), [inspection](m1/model/members/inspect-member.yaml) |
| E02 Copies | New copy is available; duplicate ID preserves original; same title with distinct IDs succeeds; borrowing/return derive availability from the actual open loan; another member cannot borrow an unavailable copy. | [copy](m1/model/catalog/copy.yaml), [registration](m1/model/catalog/register-copy.yaml), [inspection](m1/model/catalog/inspect-copy.yaml), [availability](m1/model/catalog/availability.yaml) |
| E03 Successful loan | Exactly one new stable loan ID, correct member/copy/dates, open state and null return date; count increments and copy becomes unavailable; another member/copy is unchanged. Repeating the same copy refuses without another loan. | [loan](m1/model/lending/loan.yaml), [borrow](m1/model/lending/borrow.yaml) |
| E04 Admission boundaries | Requests up to the selected limit succeed; the next refuses unchanged. Closed loans do not count; overdue open loans do count. An overdue loan adds no separate admission ban. Counts are per member. | [policy](m1/model/lending/loan-policy.yaml), [borrow](m1/model/lending/borrow.yaml) |
| E05 Suspension | A suspended member cannot borrow; existing loan records/due dates remain unchanged; suspended members can return; reactivation allows borrowing subject to all ordinary conditions. | [eligibility](m1/model/members/eligibility.yaml), [borrow](m1/model/lending/borrow.yaml), [return](m1/model/lending/return.yaml) |
| E06 Calendar arithmetic | Fourteen calendar days: `2026-10-01` becomes `2026-10-15`, `2028-02-20` becomes `2028-03-05`, `2026-12-25` becomes `2027-01-08`. Weekends add no special rule. | [policy](m1/model/lending/loan-policy.yaml), [loan](m1/model/lending/loan.yaml) |
| E07 Returns | Return closes exactly its loan, preserves borrow/due dates, records the supplied date, decreases count and releases the copy. Return at/after due date and at the borrowing limit succeeds. A date before borrowing refuses. | [return](m1/model/lending/return.yaml), [availability](m1/model/catalog/availability.yaml) |
| E08 Repeated old return | First return, borrow the same copy again, then return the old loan again: `alreadyReturned`, original return date preserved, later loan remains open and copy stays unavailable. Include a valid different repeat date and an invalid before-borrow date. | [return](m1/model/lending/return.yaml), [availability](m1/model/catalog/availability.yaml) |
| E09 Loan inspection | Known loan reports exact current data, including a closed loan; member open list excludes closed loans and gives zero/empty for a new member. Unknown loan/member refuses. All reads preserve state. | [inspect-loans](m1/model/lending/inspect-loans.yaml) |
| E10 Overdue assessment | Due on assessment date is excluded; due the previous day is included; exactly one notice per current open overdue loan, with correct fields; suspended member included; closed loans excluded even at an assessment date earlier than their return date; repeated assessment is read-only and equivalent. | [assess-overdue](m1/model/lending/reminders/assess-overdue.yaml) |
| E11 Invalid domain input | Required argument missing, wrong JSON type, empty ID, whitespace-only name/title, invalid date, unknown requested record. Include invalid leap date `2026-02-29`. Each refuses understandably, preserves state and permits the next valid request. Differently spelled valid IDs remain distinct. | [inputs](m1/model/library/inputs.yaml) and each affected operation |
| E12 Callable transport | Exact correlation, one response per line, no banners, continued handling after malformed input/unknown operation, normal EOF exit; use only the documented operations and record shapes. | [callable-interface](m1/model/library/callable-interface.yaml), required by [realization](m1/model/library/realization.yaml) |

Exercise E11 across each operation's required inputs, rather than testing validation only on registration. Isolate the reason being tested where possible; on intentionally combined failures, any actual failing condition is sufficient. Include a return while suspended, an overdue return and a return at the admission limit. Do not silently substitute an unmodeled renewal, fee, historical assessment or current-date rule.

## The M1 to M2 witness

Use exactly the same operation sequence and identifiers for this paired case, in a fresh A1 process and a fresh A2 process:

1. Register active member `member-limit` and copies `copy-1` through `copy-6`.
2. Borrow copies 1, 2 and 3 on `2026-10-01`. Both versions have three open loans, due `2026-10-15`.
3. Request copy 4 on the same date.

| Observation after step 3 | A1 against M1 | A2 against M2 |
| --- | --- | --- |
| Request result | Success, with a new loan | Refusal, with a reason |
| Open-loan count | 4 | 3 |
| Copy 4 | Unavailable | Available |
| First three loans | Open, unchanged | Open, unchanged |

Continue a separate limit case through five successful loans and a refused sixth for M1; through three successful loans and a refused fourth for M2. In both versions, return one current loan, then borrow another available copy: one new loan succeeds and the next above-limit request refuses. Other shared families remain unchanged.

Run the dedicated fourth-loan witness in both directions as a discrimination check: A1 must fail the M2 witness and A2 must fail the M1 witness. In each direction, setup and the first three loans must succeed; the mismatch must concern the fourth request's admission result and resulting count/availability. A crash, timeout or unrelated failure is not a successful discrimination check. These are expected mismatches in separate report fields, not failed primary evaluations. If both artifacts pass both witness profiles, the experiment has not demonstrated the intended policy change.

## Existing loans and the process boundary

M1's [policy](m1/model/lending/loan-policy.yaml) says a lower limit must not remove existing loans or prevent their return. A fresh M2 process cannot reach a state with four open loans through this public API. No ordinary test arrangement can create that state without bypassing M2's rule. R2 preserving the A1 source tree does not preserve a running library's records.

The first demonstration therefore establishes these separately:

- Black-box evidence: M2 refuses the fourth loan without changing the first three; existing reachable loans remain returnable at the limit, while overdue and while suspended.
- Independent source review: admission checks apply to new borrowing; inspection, return and initialization contain no forced-return/deletion rule derived from the lower limit. Record the relevant source findings against A2's exact hash. This is review evidence, not a runtime upgrade test.
- Unproven here: carrying four or five live A1 loans into A2 and then returning them. This requires a separately modeled state-transfer/upgrade capability or a deliberately expanded state-fixture contract. Do not add hidden state injection or claim restart persistence to close this gap.

This keeps the existing simple domain scope. If live data across upgrades becomes a product goal, extend the model and public interface deliberately and add that independent case before claiming migration correctness.

## Additional assessment and completion

The external behavior suite is one evidence class. Also run the candidate's own tests, inspect actual source/diff, check usage documentation against the build/run contract, and examine registry/Definition assessments and the real Orchestrator history. These checks answer different questions; generated test success or a Manager's report does not replace the external oracle. Human acceptance is a separate recorded decision.

Before trusting an evaluator, exercise it against controlled wrong transcripts/program stubs: always-five policy against M2, always-three against M1, off-by-one due dates, mutation on refusal, duplicate notices and an old return that closes a newer loan. Keep these negative checks outside the two real artifacts. Their success proves evaluator discrimination only, not a real Apply or complete defect detection.

The [Playground report](../manager-apply-playground.md#report-and-failure-contract) must state every unexecuted family and the process-boundary limitation. Passing requires all applicable assertions to execute; skipped or unreachable cases are not silently counted as passed. A failure is preserved with its exact inputs, trace and source identity.

Return to [M1](README.md) or the [callable interface](m1/model/library/callable-interface.yaml).
