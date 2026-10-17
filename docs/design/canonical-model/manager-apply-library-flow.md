# Library Apply propagation and integration

This walkthrough applies the [fixed Manager process](manager-apply-concept.md#tasks-and-delegation) to [M1](library-example/README.md). It replaces the earlier root-planned order and parent-authored prompts under [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process). The exchanges explain the owner-confirmed Explore, local planning and limited-delivery process, not an implemented scheduler or a real successful Apply. Successful deliveries consume shared execution capacity but no rejection allowance; exact technical bindings must enforce the selected process.

## Starting point

The Orchestrator captures the committed model, configuration, registry and repository, compiles the model and derives its complete local Manager tree. In M1, library, members, catalog and lending are top-level Packages; reminders belongs to lending. The library Package is not the root and is not the parent of members or catalog.

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
| 1 Receive at root | Orchestrator records the operation and supplies root context. Root starts no work before a message arrives. | Orchestrator derives identity, context and mandatory participation. Root judges its own integration responsibility. |
| 2 Propagate to every branch | Orchestrator records and delivers the unchanged operation on root-to-library, root-to-members, root-to-catalog and root-to-lending edges. Delivery failure remains visible; root output cannot remove a branch. | Orchestrator applies the fixed propagation rule. |
| 3 Propagate at lending | The same instruction goes from lending to reminders, independently of whether lending asks to delegate. Each Manager receives its own complete compiled context; reminders is not folded into lending's local Definitions. | Orchestrator preserves tree routing ([M1](manager-apply-core.md#messages)) and scope. |
| 4 Explore, then plan locally | Explore runs children before parents: reminders before lending, and every top-level Package before root ([T1](manager-apply-core.md#manager-tasks)). All Managers inspect their area without changing artifacts and report changes, prerequisites and questions; one that finds nothing to change submits a no-change result directly ([T3](manager-apply-core.md#manager-tasks)). Parents see their direct children's reports. Only after every task is explored may writing start ([T2](manager-apply-core.md#manager-tasks)). Each Manager then plans locally; no global Plan barrier delays an independent ready branch. | Agent interprets local obligations; Orchestrator enforces participation, the initial write barrier and recorded readiness. |
| 5 Resolve prerequisites | For a missing project skeleton, lending addresses a question to library, which its context names through the callable interface. The Orchestrator forwards it through root without dispatching root ([M2](manager-apply-core.md#messages)). Library owns the callable interface and required project files. Its limited normal delivery receives independent review and Orchestrator acceptance into the aggregate, so lending's next step sees it ([D3](manager-apply-core.md#deliveries-and-composition)). Library's final wiring remains open. | Orchestrator forwards along edges, enforces reviewed scope and records the aggregate version each step saw. Library chooses its owned implementation; no root-authored global work plan is needed. |
| 6 Work and seal | Lending works on the current aggregate, reserves new paths and proposes registrations. Orchestrator observes the full actual delta and protected inputs and seals candidate C1. | Lending implements; Orchestrator observes and checks authority. |
| 7 Review and correct | Fresh reviewer first records expectations from M1, then sees C1 and actual checks. Suppose code, tests and docs all permit six loans. The five-loan model rule produces a finding. Lending corrects it; candidate C2 receives current independent review. | Reviewer judges meaning; Orchestrator enforces exact subjects, finite rejection limits and evidence. |
| 8 Integrate reminders | Reminders supplies its independently reviewed result or no-change result. Once reminders is done and lending's local delivery is accepted, lending's integration step receives the composed aggregate with a summary and evidence ([T9](manager-apply-core.md#manager-tasks)), may request targeted detail, and integrates it with local work. A fresh integration review checks due dates and exclusion of returned loans. | Lending judges composition; Orchestrator refuses completion with missing children or stale evidence. |
| 9 Integrate at root | Root receives current reviewed results from all four direct branches. Library finishes required wiring using current interfaces. Root's fresh integration review checks cross-area behavior, including suspension preventing borrowing while return remains possible. | Root and reviewer judge the aggregate; Orchestrator requires all areas and evidence. |
| 10 Final verification and publication | Required aggregate checks, Orchestrator evidence check, source preflight, any configured owner approval, guarded write and actual read-back complete the operation. | Orchestrator establishes procedural completeness and actual effects. The owner's later commit accepts the result. |

This is an illustrative successful exchange sequence, not a prescribed AI plan. A dependency can change which work is ready. The Orchestrator admits the oldest ready task first ([T4](manager-apply-core.md#manager-tasks)) and runs one step at a time ([S5](manager-apply-core.md#steps)) within budget limits. After the initial Explore barrier, a waiting lending Manager does not prevent ready members work or library's prerequisite work. In this model the skeleton owner is a sibling reached through root. No Manager can be declared complete merely because its prerequisite was useful or its files did not change.

For budget accounting, library's successful skeleton and successful final delivery leave its local rejection count at zero while consuming shared execution capacity. Lending's rejected six-loan candidate adds one local rejection; its successful five-loan correction leaves that count at one. An integration rejection belongs to the integrating Manager's separate integration counter; any later rejected local correction belongs to its owner's local counter. Success and newly named tasks never reset these counts. With the default limit, two rejections trigger parent assessment of the repair plan and the third makes the task incomplete, which finishes the run Incomplete ([T14](manager-apply-core.md#manager-tasks)).

## Questions across areas

If lending needs a catalog API correction, it addresses the question to catalog. The Orchestrator forwards it along lending, root and catalog and records each hop; root is not dispatched and sees the forwarded question in its next step's context ([M2](manager-apply-core.md#messages)). The reply returns along the same edges. A reminder question to members travels reminders, lending, root and members in the same way. No intermediate Manager spends a relay turn. Explicit model references help name the addressee; they grant no write privileges.

If the responsible area is unknown, the Manager asks its parent, which replies or names a better addressee in its subtree ([M3](manager-apply-core.md#messages)). Pending replies stay visible; the request cannot disappear because a Manager guesses it is irrelevant. A missing necessary model relationship remains a modeling error. Inquiry does not manufacture that relationship or silently augment Apply context to compensate for it.

## What each handoff carries

**Operation handoff:** unchanged operation identity and bytes, Orchestrator-bound run and Manager responsibility, direct parent-child route, separately compiled context, resolved settings and current candidate basis. A later message cannot narrow the original obligation.

**Parent hint:** separate attributed advice with its grounds, such as "catalog reports a changed interface; clarify it before implementing the consumer". It cannot add a binding rule, rewrite Apply or change ownership. Summaries carry results, interfaces, prerequisites, findings, uncertainty and evidence, with targeted details available on request.

**Question or dependency:** originating Manager, concrete question, relevant model facts and current basis, addressee and recorded hops, outstanding replies and the result needed to resume. The [issue shape](manager-apply-core.md#issues) holds its fields.

**Work proposal:** summary, Definition assessments, proposed registry changes and open work. Orchestrator supplies the observed delta and sealed candidate. No-change has the same assessment and independent review duties. A prerequisite contribution names only its supported reviewed scope and keeps final obligations open.

**Review proposal:** exact subject and purpose, verdict, findings, model grounds and observed evidence references. A pass cannot accept changed bytes or an incomplete subtree.

**Accepted delivery:** Orchestrator-issued exact model/candidate/registry bindings, complete reviewed scope, child delivery references and evidence. A limited scope may be usable while the Manager remains open; dependent implementation waits until the Orchestrator has accepted this delivery into the aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). Local success does not replace integration or final verification.

## Sessions, integration and observation

Each step runs in its own container, starts from the current aggregate plus the Manager's own unaccepted changes, and continues the Manager's conversation or receives an Orchestrator-built handoff ([S1–S4](manager-apply-core.md#steps), [D3](manager-apply-core.md#deliveries-and-composition)). Independent reviewers start fresh. Uncertain activity blocks the workspace ([S2](manager-apply-core.md#steps)); no automatic resend repairs a lost response.

Parents begin with direct-child summaries and evidence. Targeted reads remain possible when necessary; they do not receive every descendant transcript by default. The Orchestrator records supplied messages, context, observed effects and exact candidate transitions. Metrics derive from those records: waits, requests between areas, rejected submissions and integration findings can reveal recurring coordination costs. Metrics do not automatically reorganize Packages or prove benefit.

Orchestrator composition follows [D2](manager-apply-core.md#deliveries-and-composition). A parent can edit its own integration artifacts, never a child's files; a correction is an issue addressed to the owner ([D5](manager-apply-core.md#deliveries-and-composition)). Changed assumptions or content require renewed affected assessment, and unknown consequences broaden review. An unresolved circular wait makes the task incomplete and finishes the run Incomplete ([T14](manager-apply-core.md#manager-tasks)), and a needed model change does too ([W3](manager-apply-core.md#while-a-run-waits)); neither becomes an endless message loop.

## Consuming a delivery without losing local work

Suppose lending has its own unfinished changes D, recorded after its last writing step ([S4](manager-apply-core.md#steps)), when the Orchestrator accepts a catalog delivery into the aggregate. Acceptance never touches a running step ([S6](manager-apply-core.md#steps)). Lending's next step starts from the current aggregate, which now holds the catalog delivery, with D replayed onto it ([D3](manager-apply-core.md#deliveries-and-composition)). Exclusive ownership keeps D's paths unchanged in the aggregate; if one had changed, that would be an Orchestrator error that stops the run ([D2](manager-apply-core.md#deliveries-and-composition)), never a silent merge or loss of D.

That step's input names the changed catalog interface, lending's own changes and the required reassessment, in its continued conversation or an Orchestrator-built handoff ([S3](manager-apply-core.md#steps)). An old delayed tool request cannot write to the new workspace, and the resulting review candidate is frozen separately. Blocking findings against an accepted delivery block the affected work ([I3](manager-apply-core.md#issues)); [D4](manager-apply-core.md#deliveries-and-composition) decides whether lending's local review must be renewed. If materialization is interrupted, no further lending step starts until journal and workspace observations establish the actual state; unknown state stays blocked, without a blind replay ([S2](manager-apply-core.md#steps)).

## Concrete refusal cases

| Case | Required result |
| --- | --- |
| Root omits lending in its output | Orchestrator still propagates to lending and reminders; a missing observed receipt cannot count as completion. |
| Lending proposes writing while members has not reported its initial Explore assessment | Orchestrator refuses writing until every task is explored ([T2](manager-apply-core.md#manager-tasks)). |
| Library's skeleton has a reviewer pass but no Orchestrator acceptance into the aggregate | Lending's dependent implementation stays blocked; unrelated eligible work may proceed. |
| An Agent calls a changed input irrelevant and reuses its old review | Orchestrator requires an unchanged complete bound subject or renewed assessment; changed aggregate integration remains current. |
| Parent rewrites Apply to cover borrowing but not returns | Refuse narrowed operation; the full compiled local responsibility remains binding. |
| Empty or one-child Package is optimized out | Refuse completion without that Manager's explicit assessed result and required review. |
| Root addresses reminders, or lending addresses catalog | No direct delivery: the Orchestrator forwards the question along the tree path and records each hop without dispatching the Managers in between ([M2](manager-apply-core.md#messages)). |
| A child waits for a parent-owned prerequisite | Permit ready owned parent work; do not require child completion first. |
| A useful skeleton contribution is reported as final completion | Refuse until the owner's remaining duties and current review/integration are complete. |
| Lending writes unreserved or protected inputs | Refuse the candidate; later registration cannot authorize the earlier write. |
| C1 passed but C2 is delivered | Refuse the mismatched subject and require current assessment. |
| Reminders or an outstanding dependency reply is unresolved | Do not accept the lending subtree or root aggregate. |
| A necessary relationship is absent from the model | Record the modeling error; do not infer context or edit the model during Apply. |
| All Agents say done but checks or integration evidence are missing | Refuse success. |
| Source inputs drift or publication is partial | Preserve observations; do not return Applied. |

These are controlled process cases for later implementation. They neither authorize real provider runs nor replace the two real library Applies.

## First Apply after model adoption

Status: owner-accepted workflow, 17 October 2026 ([source](../project-capability-skills-20261009.md#owner-source)). This is a hypothetical existing-project scenario, not an observed defect in Library M1 or an executed Apply. It adds no implementation or first-Apply delivery scope. The [Extract completion criterion](commands.md#completion-and-adoption-with-known-deviations) permits adoption of a reviewed model with known realization deviations; artifact reconciliation is the next separately authorized operation.

Suppose the owner has adopted the model, responsibilities and exact file accounting for an existing library application. The model permits five open loans, but the code still admits a sixth. Return behavior and other parts already conform. The owner's ordinary request is: "Reconcile the existing application with the adopted model and correct the known deviations. Preserve valid realization and its existing interfaces."

The starting aggregate is the captured existing realization, not an empty replacement or a freshly generated skeleton. A construction prerequisite in the earlier exchange example is needed only when an actual obligation is missing; adoption does not itself require reconstruction. The outer coding agent selects the accepted model revision, current source and configured execution authority, commissions Apply and relays status or decisions. It does not repair artifacts outside Markitect to bypass the process.

### Assess the whole responsibility, change what needs correction

1. **Bind and hand over.** Bind the adopted model, actual realization, configuration and registry. The [existing finding structure](manager-apply-execution.md#extract-findings-at-the-first-apply) carries known Extract deviations and source evidence into the current responsibility; model acceptance is not artifact acceptance.
2. **Explore every area read-only.** Every local Manager, including root, examines its full obligations and current artifacts. All initial Explore assessments precede artifact writing. Known findings are useful inputs, not an exclusive task list; absence of an Extract finding is no proof of correctness.
3. **Keep or correct with reasons.** Managers propose justified no-change results or changes addressing actual obligations and deviations. Each writing Manager changes only its allowed artifacts; another area's correction goes through its responsible Manager. The model is not weakened to excuse a defect, and missing intended meaning becomes an owner question.
4. **Review current subjects.** The configured independent reviewers assess both change and no-change proposals against the model and current evidence. Existing suitable checks may be reused after judging their assertions and applicability. Actual required observations must still be available; tests themselves can be faulty realizations.
5. **Integrate and publish.** Parents assess the composed result and cross-area behavior. Changed interfaces or assumptions can require renewed affected assessment even when a consumer's bytes remain unchanged. Required aggregate checks, final evidence, source preflight, configured owner approval, guarded publication and read-back remain mandatory; the owner's later commit accepts the result.

| Observed condition against the adopted model | Appropriate outcome |
| --- | --- |
| Lending admits a sixth loan against the five-loan limit. | Correct admission and supply distinguishing evidence: the fifth remains possible under valid conditions and the next loan is refused at the limit. Assess the other applicable lending obligations too. |
| Return code correctly permits a suspended member to return, but a test expects refusal. | Correct the faulty test and retain conforming return code. A green test cannot override the model. |
| Catalog behavior and its required interfaces already conform. | Retain the realization with an explained no-change assessment and independent review. |
| Documentation already states the accepted five-loan policy accurately. | Keep it; generating a different wording is not required by adoption. |
| An agent prefers other internal function names without a violated obligation. | Record a useful suggestion as advisory if appropriate. It creates no repair duty or blocking conformity finding. |

No-change means a supported judgment about obligations and actual realization, not merely an empty diff or an old approval. Conversely, a valid existing design is not wrong because an agent would implement it differently. Review asks which obligation or demonstrated deviation justifies a material change. A coherent local restructuring needed for a correction can be justified; there is no smallest-diff-at-any-cost rule. Cleanup or Refactor can separately seek improvements in conforming artifacts under their own authorized objectives.

For the loan-limit correction, reviewers derive distinguishing expectations from the model, inspect the actual candidate and check evidence that the boundary now holds without losing the other required behavior. The responsible Manager proposes closure of the original finding; independent review establishes resolution on the current subject. Local review does not replace parent integration. Ownership and subject bindings are mechanically checked by the Orchestrator; the substance of a justification remains review judgment, not a proof from diff size.

### Completion after adoption is stricter than model readiness

Extract could finish with adequately supported known realization deviations. Apply cannot report success while its blocking deviations, necessary checks or required current assessments remain open. Insufficient authority or budget leaves retained candidates and findings with an honest Incomplete result; it cannot waive evidence, trigger unlimited retries or turn adoption into a pass for the artifacts. A necessary model change remains separate from this artifact-only operation.

The concise result distinguishes changed and retained areas, the model obligations and evidence supporting each judgment, the actual resolution of known findings, open work and what was actually published. Preservation means keeping conforming artifacts; it does not mean omitting them from the first reconciliation or exempting them from integration assessment. These remain design requirements until the complete executable path and its evidence exist.
