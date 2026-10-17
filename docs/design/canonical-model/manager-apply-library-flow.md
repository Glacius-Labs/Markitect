# Library Apply propagation and integration

This walkthrough applies the [fixed Manager process](manager-apply-concept.md#tasks-and-delegation) to [M1](library-example/README.md). It replaces the earlier root-planned order and parent-authored prompts under [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process). The exchanges explain the owner-confirmed Explore, local planning and limited-delivery process, not an implemented scheduler or a real successful Apply. Successful deliveries consume shared execution capacity but no rejection allowance; exact technical bindings must enforce the selected process.

## Starting point

The Host captures the committed model, configuration, registry and repository, compiles the model and derives its complete local Manager tree. In M1, library, members, catalog and lending are top-level Packages; reminders belongs to lending. The library Package is not the root and is not the parent of members or catalog.

```text
root
  library
  members
  catalog
  lending
    reminders
```

The versioned Apply instruction is identical for every Manager: reconcile the artifacts of your responsibility with the bound model, assess existing realization, correct deviations and explain the result. Context, role instructions, exact scope and current messages are rendered separately. The model stays read-only. All local Managers participate; imported vocabulary supplies context without spawning Managers.

## Messages down, reviewed results up

| Step | Exchange | Decision owner |
| --- | --- | --- |
| 1 Receive at root | Host records the operation and supplies root context. Root starts no work before a message arrives. | Host derives identity, context and mandatory participation. Root judges its own integration responsibility. |
| 2 Propagate to every branch | Host records and delivers the unchanged operation on root-to-library, root-to-members, root-to-catalog and root-to-lending edges. Delivery failure remains visible; root output cannot remove a branch. | Host applies the fixed propagation rule. |
| 3 Propagate at lending | The same instruction goes from lending to reminders, independently of whether lending asks to delegate. Each Manager receives its own complete compiled context; reminders is not folded into lending's local Definitions. | Host preserves adjacent routing and scope. |
| 4 Explore, then plan locally | Explore runs children before parents: reminders before lending, and every top-level Package before root ([T1](manager-apply-core.md#manager-tasks)). All Managers inspect their area without changing artifacts and report changes, prerequisites and questions; one that finds nothing to change submits a no-change result directly ([T3](manager-apply-core.md#manager-tasks)). Parents see their direct children's reports. Only after every task is explored may writing start ([T2](manager-apply-core.md#manager-tasks)). Each Manager then plans locally; no global Plan barrier delays an independent ready branch. | Agent interprets local obligations; Host enforces participation, the initial write barrier and recorded readiness. |
| 5 Resolve prerequisites | For a missing project skeleton, lending addresses a question to library, which its context names through the callable interface. The Host forwards it through root without dispatching root ([M2](manager-apply-core.md#messages)). Library owns the callable interface and required project files. Its limited normal delivery receives independent review and Host acceptance into the aggregate, so lending's next step sees it ([D3](manager-apply-core.md#deliveries-and-composition)). Library's final wiring remains open. | Host forwards along edges, enforces reviewed scope and records the aggregate version each step saw. Library chooses its owned implementation; no root-authored global work plan is needed. |
| 6 Work and seal | Lending works on the current aggregate, reserves new paths and proposes registrations. Host observes the full actual delta and protected inputs and seals candidate C1. | Lending implements; Host observes and checks authority. |
| 7 Review and correct | Fresh reviewer first records expectations from M1, then sees C1 and actual checks. Suppose code, tests and docs all permit six loans. The five-loan model rule produces a finding. Lending corrects it; candidate C2 receives current independent review. | Reviewer judges meaning; Host enforces exact subjects, finite rejection limits and evidence. |
| 8 Integrate reminders | Reminders supplies its independently reviewed result or no-change result. Once reminders is done and lending's local delivery is accepted, lending's integration step receives the composed aggregate with a summary and evidence ([T8](manager-apply-core.md#manager-tasks)), may request targeted detail, and integrates it with local work. A fresh integration review checks due dates and exclusion of returned loans. | Lending judges composition; Host refuses completion with missing children or stale evidence. |
| 9 Integrate at root | Root receives current reviewed results from all four direct branches. Library finishes required wiring using current interfaces. Root's fresh integration review checks cross-area behavior, including suspension preventing borrowing while return remains possible. | Root and reviewer judge the aggregate; Host requires all areas and evidence. |
| 10 Final verification and publication | Required aggregate checks, Host evidence check, source preflight, any configured owner approval, guarded write and actual read-back complete the operation. | Host establishes procedural completeness and actual effects. The owner's later commit accepts the result. |

This is an illustrative successful exchange sequence, not a prescribed AI plan. A dependency can change which work is ready. The Host admits the oldest ready task first ([T4](manager-apply-core.md#manager-tasks)) and runs one step at a time ([S5](manager-apply-core.md#steps)) within budget limits. After the initial Explore barrier, a waiting lending Manager does not prevent ready members work or library's prerequisite work. In this model the skeleton owner is a sibling reached through root. No Manager can be declared complete merely because its prerequisite was useful or its files did not change.

For budget accounting, library's successful skeleton and successful final delivery leave its local rejection count at zero while consuming shared execution capacity. Lending's rejected six-loan candidate adds one local rejection; its successful five-loan correction leaves that count at one. An integration rejection belongs to the integrating Manager's separate integration counter; any later rejected local correction belongs to its owner's local counter. Success and newly named tasks never reset these counts. With the default limit, two rejections trigger parent assessment of the repair plan and the third makes the task incomplete, which finishes the run Incomplete ([T10](manager-apply-core.md#manager-tasks)).

## Questions across areas

If lending needs a catalog API correction, it addresses the question to catalog. The Host forwards it along lending, root and catalog and records each hop; root is not dispatched and sees the forwarded question in its next step's context ([M2](manager-apply-core.md#messages)). The reply returns along the same edges. A reminder question to members travels reminders, lending, root and members in the same way. No intermediate Manager spends a relay turn. Explicit model references help name the addressee; they grant no write privileges.

If the responsible area is unknown, the Manager asks its parent, which replies or names a better addressee in its subtree ([M3](manager-apply-core.md#messages)). Pending replies stay visible; the request cannot disappear because a Manager guesses it is irrelevant. A missing necessary model relationship remains a modeling error. Inquiry does not manufacture that relationship or silently augment Apply context to compensate for it.

## What each handoff carries

**Operation handoff:** unchanged operation identity and bytes, Host-bound run and Manager responsibility, direct parent-child route, separately compiled context, resolved settings and current candidate basis. A later message cannot narrow the original obligation.

**Parent hint:** separate attributed advice with its grounds, such as "catalog reports a changed interface; clarify it before implementing the consumer". It cannot add a binding rule, rewrite Apply or change ownership. Summaries carry results, interfaces, prerequisites, findings, uncertainty and evidence, with targeted details available on request.

**Question or dependency:** originating Manager, concrete question, relevant model facts and current basis, addressee and recorded hops, outstanding replies and the result needed to resume. The [issue shape](manager-apply-core.md#issues) holds its fields.

**Work proposal:** summary, Definition assessments, proposed registry changes and open work. Host supplies the observed delta and sealed candidate. No-change has the same assessment and independent review duties. A prerequisite contribution names only its supported reviewed scope and keeps final obligations open.

**Review proposal:** exact subject and purpose, verdict, findings, model grounds and observed evidence references. A pass cannot accept changed bytes or an incomplete subtree.

**Accepted delivery:** Host-issued exact model/candidate/registry bindings, complete reviewed scope, child delivery references and evidence. A limited scope may be usable while the Manager remains open; dependent implementation waits until the Host has accepted this delivery into the aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). Local success does not replace integration or final verification.

## Sessions, integration and observation

Each step runs in its own container, starts from the current aggregate plus the Manager's own unaccepted changes, and continues the Manager's conversation or receives a Host-built handoff ([S1–S4](manager-apply-core.md#steps), [D3](manager-apply-core.md#deliveries-and-composition)). Independent reviewers start fresh. Uncertain activity blocks the workspace ([S2](manager-apply-core.md#steps)); no automatic resend repairs a lost response.

Parents begin with direct-child summaries and evidence. Targeted reads remain possible when necessary; they do not receive every descendant transcript by default. The Host records supplied messages, context, observed effects and exact candidate transitions. Metrics derive from those records: waits, requests between areas, rejected submissions and integration findings can reveal recurring coordination costs. Metrics do not automatically reorganize Packages or prove benefit.

Host composition follows [D2](manager-apply-core.md#deliveries-and-composition). A parent can edit its own integration artifacts, never a child's files; a correction is an issue addressed to the owner ([D5](manager-apply-core.md#deliveries-and-composition)). Changed assumptions or content require renewed affected assessment, and unknown consequences broaden review. An unresolved circular wait makes the task incomplete and finishes the run Incomplete ([T10](manager-apply-core.md#manager-tasks)), and a needed model change does too ([W3](manager-apply-core.md#while-a-run-waits)); neither becomes an endless message loop.

## Consuming a delivery without losing local work

Suppose lending has its own unfinished changes D, recorded after its last writing step ([S4](manager-apply-core.md#steps)), when the Host accepts a catalog delivery into the aggregate. Acceptance never touches a running step ([S6](manager-apply-core.md#steps)). Lending's next step starts from the current aggregate, which now holds the catalog delivery, with D replayed onto it ([D3](manager-apply-core.md#deliveries-and-composition)). Exclusive ownership keeps D's paths unchanged in the aggregate; if one had changed, that would be a Host error that stops the run ([D2](manager-apply-core.md#deliveries-and-composition)), never a silent merge or loss of D.

That step's input names the changed catalog interface, lending's own changes and the required reassessment, in its continued conversation or a Host-built handoff ([S3](manager-apply-core.md#steps)). An old delayed tool request cannot write to the new workspace, and the resulting review candidate is frozen separately. Blocking findings against an accepted delivery block the affected work ([I3](manager-apply-core.md#issues)); [D4](manager-apply-core.md#deliveries-and-composition) decides whether lending's local review must be renewed. If materialization is interrupted, no further lending step starts until journal and workspace observations establish the actual state; unknown state stays blocked, without a blind replay ([S2](manager-apply-core.md#steps)).

## Concrete refusal cases

| Case | Required result |
| --- | --- |
| Root omits lending in its output | Host still propagates to lending and reminders; a missing observed receipt cannot count as completion. |
| Lending proposes writing while members has not reported its initial Explore assessment | Host refuses writing until every task is explored ([T2](manager-apply-core.md#manager-tasks)). |
| Library's skeleton has a reviewer pass but no Host acceptance into the aggregate | Lending's dependent implementation stays blocked; unrelated eligible work may proceed. |
| An Agent calls a changed input irrelevant and reuses its old review | Host requires an unchanged complete bound subject or renewed assessment; changed aggregate integration remains current. |
| Parent rewrites Apply to cover borrowing but not returns | Refuse narrowed operation; the full compiled local responsibility remains binding. |
| Empty or one-child Package is optimized out | Refuse completion without that Manager's explicit assessed result and required review. |
| Root addresses reminders, or lending addresses catalog | No direct delivery: the Host forwards the question along the tree path and records each hop without dispatching the Managers in between ([M2](manager-apply-core.md#messages)). |
| A child waits for a parent-owned prerequisite | Permit ready owned parent work; do not require child completion first. |
| A useful skeleton contribution is reported as final completion | Refuse until the owner's remaining duties and current review/integration are complete. |
| Lending writes unreserved or protected inputs | Refuse the candidate; later registration cannot authorize the earlier write. |
| C1 passed but C2 is delivered | Refuse the mismatched subject and require current assessment. |
| Reminders or an outstanding dependency reply is unresolved | Do not accept the lending subtree or root aggregate. |
| A necessary relationship is absent from the model | Record the modeling error; do not infer context or edit the model during Apply. |
| All Agents say done but checks or integration evidence are missing | Refuse success. |
| Source inputs drift or publication is partial | Preserve observations; do not return Applied. |

These are controlled process cases for later implementation. They neither authorize real provider runs nor replace the two real library Applies.
