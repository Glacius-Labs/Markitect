# Library Apply delegation and integration

This walkthrough traces one Apply of [M1](library-example/README.md) through the [Apply concept](manager-apply-concept.md) and the [port contracts](manager-apply-port-contracts.md). It shows the order of exchanges and who decides each step.

## Starting point

The Host has captured and compiled M1, bound configuration and repository inputs, and derived the ordinary root plus the library, members, catalog, lending and lending/reminders Managers. The root receives the user's Apply instruction. All tasks inherit artifact-only reconciliation and a read-only canonical model.

The root's recorded order put library first: its Manager owns the Go module, entry point and callable interface that `library/callable-interface` requires. Members and catalog followed, then lending. For this focused trace, B0 denotes the current unpublished candidate, which already contains the accepted library, member and catalog deliveries from earlier tasks in the same run. This does not imply that shared library obligations or the whole project have passed. All other required Package deliveries remain part of the root's completion criteria. C1, C2, I1 and I2 below are illustrative candidate labels, not recorded content digests.

## Root to lending and back

| Step | Exchange and result | Decision owner |
| --- | --- | --- |
| 1 Root receives context | Its task includes the operation goal, bound M1, Package directory, shared rules, current candidate and available child summaries. It receives no required descendant file-edit list. | Host prepares and records the actual input; root decides how to delegate. |
| 2 Root delegates | Root asks lending to reconcile its area, including reminders, as code, tests and documentation against M1. It can cite `lending/borrow`, `lending/loan-policy`, `members/eligibility` and `catalog/availability`. The Host still includes every local lending Definition and required descendant responsibility. | Root supplies outcome and basis; Host checks the direct parent/child relation and fixes scope, identity and candidate base B0. |
| 3 Root yields | Delegation immediately returns a recorded child task ID. Root finishes its current step with the child pending. The Host schedules the child; no provider tool call waits synchronously for an entire subtree to finish. | Host owns queue and waiting state. |
| 4 Lending receives context | Its eight local Definitions include the LendingPolicy Kind and Properties, policy, loan, borrowing, return and inspection. See also exposes membership, availability and shared delivery rules, with reasons. Reminders is a child responsibility, not inherited local content. | Organization renders supplied facts; Host records the packet and starts the Manager session. |
| 5 Lending works | It reads its existing realization and reserves new paths through Host tools before writing. Existing file ownership comes from the registry. It implements in a private workspace and proposes its Definition assessments and realization explanation. | Lending chooses local implementation; Host validates reservations and scope. |
| 6 Host seals C1 | After the writing step ends, Host captures actual bytes and the registry delta, checks protected inputs and ownership, and creates immutable candidate C1. Required checks run against that exact candidate. | Host validates observed changes, not the Agent's claimed diff. |
| 7 Fresh local review | A fresh reviewer receives C1, M1, assessments and actual check evidence. Suppose code, tests and guide all permit six loans. The reviewer reports the conflict with `loan-policy.maxOpenLoans`, whose M1 value is five. Passing generated tests cannot discharge that finding. | Reviewer judges meaning; Host validates the result's identity, purpose, subject and evidence references. |
| 8 Rework and review | Lending receives that finding and revises from the recorded C1 state. Host seals C2, runs current checks and starts another fresh review. A pass belongs to C2 and local delivery scope. Attempts remain bounded. | Lending repairs; reviewer reassesses; Host controls transitions and budgets. |
| 9 Reminders delivery | Lending delegates to its reminders child through the same protocol, using a current base and an outcome about open overdue loans at an explicit date. The child supplies its own reviewed result. | Lending issues the prompt; Host dispatches and validates; the child owns its work. |
| 10 Lending integrates | Host composes the accepted local and reminders deltas with explicit bases into I1. Lending assesses their combined behavior. A fresh integration reviewer examines I1, including whether reminders consume the current loan/due-date contract and exclude returned loans. | Host checks composition; lending and its integration reviewer judge the aggregate. |
| 11 Host accepts the subtree | Only when local work, descendants, required reviews/checks, assessments and questions are complete does Host issue an accepted lending delivery. It carries I1, registry revision, evidence and a concise report. | Host issues the delivery; the Agent cannot self-issue acceptance. |
| 12 Root resumes and integrates | Root receives the accepted delivery and can inspect details by identity. Host composes it with other required deliveries into I2. Root assesses system integration; a fresh integration reviewer checks the aggregate, including suspension preventing borrowing. | Root coordinates and judges integration; Host validates composition and evidence. |
| 13 Verify and publish | The complete current aggregate receives required checks and the Host's final evidence check over coverage, the Definition census and all review bindings. Apply rechecks source inputs, requests guarded publication of artifacts and registry, and records actual read-back. | Apply owns this final lifecycle; the publisher observes physical writes. |

The real order is chosen through recorded tasks and ready work. A Manager waiting for another area does not prevent the Host from running other ready tasks. Shared library obligations can require a later assessment against the assembled result; the root cannot mark that Manager complete merely because the other areas worked locally.

## What each handoff carries

**Delegation proposal:** receiving Manager, intended outcome, model basis and relevant constraints. Sender identity comes from the actual session. Host adds run/task/parent identities, attempt, complete responsibility, inherited restrictions and the current candidate. A task's cited basis is a starting point, not permission to omit its other Definitions.

**Work proposal:** summary, Definition assessments, registry proposal references, child results and open questions. Host adds the observed delta and sealed candidate. A no-change proposal has the same obligations; empty file changes alone are insufficient.

**Review proposal:** exact subject and purpose, verdict, findings, model grounds and references to observed evidence. It does not contain accepted delivery authority. Changes requested lead to another work attempt, a model issue stays open, and exhausted attempts terminate incomplete.

**Accepted delivery:** Host-issued identity, exact model/candidate/registry bindings, complete reviewed scope, assessments, child delivery IDs, checks/reviews and report. A local pass supports local scope; an accepted lending subtree also requires reminders and integration. Neither replaces root integration or the final evidence check.

## Context and session continuity

Root and lending are ordinary Managers with different responsibilities. Their logical identities survive steps and attempts. Provider session identities are separate. A Manager can continue a known session with recorded supplied history; a reviewer starts fresh, without the implementation conversation. Rework adds the grounded finding and candidate basis rather than silently replaying the previous step.

If a session or workspace binding cannot be safely continued, use an explicit new session with compiled prior reports and a recorded relationship to the earlier attempt. Do not claim the new session has inherited an unknown provider conversation. An uncertain writer must be reconciled or stopped before another writer receives that workspace.

The exact initial context, custom instructions, supplied additions, actual outgoing request, observed responses, tool exchanges and candidate differences enter the same run history. Summaries sent upward reference those records. This gives the root a useful overview without requiring the whole descendant transcript in its initial prompt.

## Integration rules

Host composition applies validated child changes to an exact base and binds the registry to the resulting artifact tree. It does not choose a semantic conflict resolution. A conflicting or outdated delivery returns to its owner for a current candidate and review. A parent does not repair a conflict by editing child-owned files.

If integration needs an authored change, perform an ordinary owned work/review step and then review the new aggregate. Local reviews and resolved findings remain evidence for their original subjects. Renew affected assessments when their content or assumptions change; unknown consequences broaden review. There is no automatic assertion that an unchanged file makes all earlier semantic evidence current.

## Concrete refusal cases

| Case | Required result |
| --- | --- |
| Root tries to delegate directly to lending/reminders | Refuse the invalid parent/child route; lending coordinates its child. |
| Root narrows the task to borrowing and omits returns | Host retains lending's complete reconciliation responsibility; omission prevents completion. |
| Lending writes an unreserved new file or a model/config/ignore input | Refuse the candidate. A later claim cannot silently authorize an already observed unauthorized write. |
| A Kind or Property is labelled vocabulary-only | Require a reason and concrete uses where its obligations are assessed; the numeric policy cannot disappear behind that label. |
| Reviewer passes C1 but the delivered bytes are C2 | Refuse the mismatched subject; assess C2. |
| Reminders remains unresolved while lending's local code passes | Do not accept the lending subtree. |
| A member rule needs a missing link to lending | Record a source-grounded question through the common parent. Apply cannot add the link; unresolved required assessment prevents completion. |
| A tool request or provider response is observed twice | Use recorded correlation IDs; do not create another child or another work attempt for a duplicate. |
| A step times out after possibly being sent | Record uncertainty and inspect the known session/step; do not automatically send the work again. |
| Final evidence is complete but repository inputs changed | Return stale before publication. |
| Publication writes only part of the candidate | Preserve actual writes, report partial failure and inspect recovery state; do not return Applied. |

These cases are the first controlled-adapter tests of the [port contracts](manager-apply-port-contracts.md); real role sessions and the two real Applies follow separately.
