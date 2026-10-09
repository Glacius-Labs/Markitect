# Public holdout and independent check procedure

Version: 0.1. Freeze this procedure with the rubric and checks before any actor trial.

## Boundary

A holdout is a private evaluation of whether the candidate satisfies requirements already released in the public brief or a task card. It may vary input values, call order, process restart timing, concurrency schedule, and equivalent implementation-level fault placement. It may not introduce a new product rule, endpoint, response promise, security requirement, style preference, or failure condition absent from the public contract. If a proposed check cannot be traced to a released sentence, it is excluded from scoring or published as a new study version before any affected trial.

The private oracle records each assertion's source requirement, applicable task cutoff, cell applicability, expected observable behavior, and reason the assertion is held out. Private data stays outside actor repositories and their ordinary handoff artifacts. Filesystem separation alone is not treated as access control; the study operator must inspect actual runner/tool visibility and record any exposure. Any actor access to private cases or results contaminates the affected cell and is reported.

## Construction and execution

1. Derive checks from the released contract. Freeze public checks, private check definitions, rubric, start fixture SHAs, and task-release mapping before trials. Record digests in the private study ledger.
2. Give all arms the same public check sources and invocation instructions at the same task cutoff. Do not expose holdout values, schedules, mutations, expected outputs, or pass/fail feedback during implementation.
3. Run public checks for feedback only where the task contract allows. Run private holdout checks after the candidate is frozen for scoring. Capture the candidate SHA, runner/tool version, commands, raw outputs, and check digest.
4. For task 6, first freeze the task-5 candidate. The operator injects one behaviorally equivalent violation of a released invariant into each cell, selected without exposing a guaranteed path or symbol. Keep the mapping from requirement to mutation to observed failure private. Then release the same repair card. A blinded reviewer checks equivalence across cells before scoring. A failed equivalence check makes the affected cell non-comparable; it must not be repaired by changing the expected behavior after seeing the actor's response.
5. Report public and holdout findings by requirement and task cutoff. A failed hard invariant remains visible even if the weighted dimensions score well.

## Independence and leakage controls

The assessor works from frozen candidates and the private oracle, not actor explanations alone. Where feasible, candidate identifiers are blinded and order randomized. The assessor records observed results separately from interpretation. No assessor gives hidden-case feedback before the relevant candidate is frozen. If hidden results are disclosed and later work continues, mark that cell and subsequent affected work as contaminated; do not describe it as an independent holdout.

A holdout pass establishes only that tested examples satisfy the cited public requirements under the recorded runner. It does not establish unrestricted correctness, general software quality, or absence of untested defects.

## Prospective evaluation correction
See operator-evidence-contract.md for exact real-predecessor/same-database continuity, phase completion and structured attribution. This clarifies existing duties; no historical results are rescored. All rubric weights remain unchanged.
