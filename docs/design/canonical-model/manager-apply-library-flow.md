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

## First business-rule change: from five to seven

Status: owner-accepted working-day walkthrough, 17 October 2026 ([source](../project-capability-skills-20261009.md#owner-source)). Assume the first reconciliation above has succeeded and the existing application conforms to the adopted five-loan model. This is a hypothetical continuation, not evidence of an executed Apply. It changes no model fixture, implementation or delivery scope; the [delivery demonstration](../../implementation-plan.md#target-and-baseline) retains its separate five-to-three change.

The owner's ordinary request can be: "Increase the limit from five to seven simultaneously open loans per member. Preserve its existing scope and all other conditions. Update the model first, then have Markitect reconcile the application. Ask only about material conflicts or missing authority." This is a change of intended behavior, rather than a defect repair against unchanged intent. A discussion-only request authorizes neither the model edit nor execution; this document records the workflow only.

### Coding agent: maintain and adopt the intended model

The outer coding agent uses `markitect-design` to locate the authoritative Definition and inspect its meaning, references and constraints. In the library example, the canonical owner is [lending/loan-policy](library-example/m1/model/lending/loan-policy.yaml), using [maxOpenLoans](library-example/m1/model/lending/max-open-loans.yaml): the resulting count after successful borrowing must not exceed the limit. The proposed value change is only:

```diff
-  maxOpenLoans: 5
+  maxOpenLoans: 7
```

The scope remains per member, counting open loans as already defined. Loan duration, suspension, return behavior, ownership and other conditions retain their accepted meaning. The agent does not replace unrelated occurrences of five or add a new policy merely to make the edit seem complete. Consequential conflicts with another binding statement become a concrete owner question with sources and a recommendation.

Review checks that the proposed model expresses the requested change without unintended changes of meaning. The agent uses `markitect-check` for structure and accounting, refreshes the official readable view through its supported writer, and commits the authorized model revision under the project's existing adoption policy and delegated authority. The owner's clear instruction already settles seven; no additional confirmation is needed for that same decision. A commit supplies the specification basis, not proof of human review.

At this boundary the application may still enforce five. Report that difference honestly; a valid new model is not evidence that artifacts conform. Commit the selected model and leave a clean captured source before commissioning Apply. Do not ask artifact workers to change the model during realization.

### Markitect: realize the new rule in the existing application

Through `markitect-apply`, the coding agent commissions the supported workflow against the committed new model, actual source, configuration and existing finite execution grant. It observes progress and relays decisions; it does not make application repairs outside Markitect or construct a second Manager scheduler. An ordinary assignment can authorize both model work and subsequent realization without requiring a separate human prompt for each skill.

The [fixed process](#messages-down-reviewed-results-up) still includes every local Manager, including root, with read-only Explore completed throughout the tree before writing. The numeric change does not authorize omitting Managers based only on a predicted impact list. Each Manager assesses its complete responsibility, changes its own artifacts where needed and supplies justified no-change results for conforming parts. Parents integrate their local work and direct children's results; configured independent reviewers assess both changes and retained realization.

| Area | Expected assessment and possible change |
| --- | --- |
| Lending | Enforce the seven-loan admission limit while retaining the other borrowing and return obligations. |
| Tests | Update expectations that intentionally depended on five; establish that the sixth and seventh loans succeed when other prerequisites hold, and an eighth is refused with no observable state change. |
| User documentation | Update statements of the current binding limit; preserve unrelated correct guidance and historical records in their original scope. |
| Catalog, membership and reminders | Assess relevant interactions; retain conforming realization with reasons and review. Suspension and due-date behavior do not change merely because the limit increased. |
| Parent integration | Assess the composed application and required cross-area behavior with current evidence. Local approval alone does not establish aggregate success. |

A former test requiring refusal of the sixth loan now expresses superseded intent. Change that expectation deliberately and preserve its purpose of testing the admission boundary at the new limit; do not remove boundary protection merely to obtain green tests. Reviewers derive expectations from the new model and examine actual assertions and observations. Prior passes cannot automatically approve changed meaning. Any earlier finding retains its provenance through the [existing issue structure](manager-apply-execution.md#extract-findings-at-the-first-apply); its applicability or resolution needs current grounds rather than silent deletion.

### Owner decisions and the completion boundary

| Situation | Owner involvement |
| --- | --- |
| The model unambiguously expresses the same rule with seven instead of five. | The instruction already decides the change. The agent performs authorized model maintenance and Markitect handles realization, review and bounded rework. |
| A conflicting binding rule or consequential ambiguity changes who or what the limit covers. | Decide the source-bound question and concrete proposed meaning. Do not let an agent silently choose an exception or broaden the request. |
| Realization exposes a necessary additional model change or exceptional tradeoff. | Decide the separate intent proposal. Apply keeps the model read-only and cannot claim completion while a necessary unresolved decision remains. |
| Required provider authority, budget or publication approval is missing. | Supply the specific missing grant or choice if desired; otherwise retain an honest incomplete result. Existing valid grants need no repeated approval. |
| Publication succeeds and the project requires result acceptance. | Follow the existing acceptance policy; the owner's later commit accepts the published artifacts. Apply itself never commits, merges or pushes. An outer agent may handle repository delivery only within separately granted authority. |

Routine implementation choices, ordinary review findings and bounded corrections stay with the responsible Managers and reviewers. They do not create a standing owner approval round. Required aggregate checks, current independent assessments, preflight, any configured approval, guarded publication and actual read-back still apply. The final report identifies the selected model/source, changed and retained areas with their reasons, evidence for the new boundary and preserved behavior, open findings and actual publication. It distinguishes model adoption, realization, publication and owner acceptance; this documentation claims none of them was executed.

## Defect repair: an eighth loan despite the seven-loan limit

Status: owner-accepted working-day walkthrough, 18 October 2026 ([source](../project-capability-skills-20261009.md#owner-source)). Continue the hypothetical seven-loan scenario above: the owner reports that an eighth open loan is admitted. This is not an observed defect in the executable library fixtures, an executed repair or new implementation scope. The accepted intent remains seven; no model edit or artificial model-version change is needed to repair a realization deviation.

The owner's ordinary request can be: "Investigate and fix the eighth-loan admission through Markitect. Keep the seven-loan model and correct existing behavior. Show that seven loans remain possible and an eighth is refused without changing state." The request authorizes investigation and repair within existing execution grants. A discussion-only request, including this documentation task, executes neither.

### Coding agent: bind the report and commission the repair

The outer coding agent identifies the applicable committed model, the reported application version or repository revision, relevant configuration, starting data and reproduction steps. Keep the affected application and source bindings explicit; do not assume the observed version is the candidate under investigation. "An eighth loan was admitted" is a reported observation; "the comparison is off by one" is a diagnosis that still needs evidence.

For an authorized repair, `markitect-apply` commissions the ordinary investigation, realization and review workflow against the unchanged model. A separate `verify` run is not a mandatory preliminary approval round. If the owner asks only for assessment, use `markitect-verify`, retain the actual findings and make no repairs. The coding agent relays progress and genuine questions; it neither patches application artifacts outside Markitect nor introduces a second scheduler. Model structure passing `check` cannot establish correct loan admission.

### Markitect: investigate and carry the finding through the existing process

Use the [existing issue shape](manager-apply-core.md#issues) and [finding rules](manager-apply-execution.md#evidence-need-survives-budget-pressure), not another repair ledger. Preserve the original report as provenance and supply its source basis and reproduction information to Explore and review. A user report is not an authenticated independent reviewer verdict. A confirmed deviation records the affected Definitions, exact model and realization subject, observed evidence, reason, responsible Manager, severity and required correction or observation. A confirmed admission above the applicable limit is blocking for a conforming Apply result.

The illustrative reproduction starts with seven open loans for an otherwise eligible member and another available copy. Borrowing that copy should refuse without observable state change; instead, the reported application creates an eighth loan. Managers investigate the actual cause, such as a faulty boundary check or open-loan count, rather than implementing a guessed diagnosis. If the case cannot be reproduced or adequately assessed, retain the evidence gap; lack of reproduction alone does not establish conformity or close the report as resolved.

Every local Manager, including root, still participates, and all initial read-only Explore assessments precede writing. Findings guide investigation without replacing full responsibility. The responsible Manager corrects its owned realization and suitable tests; necessary work in another area goes to that area's Manager. Correct artifacts remain with explained no-change assessments and required review. The model stays read-only: raising the limit to eight, weakening the rule or changing a test to expect the defect cannot count as repair.

### Independent evidence and closure

Configured independent reviewers derive expectations from the unchanged seven-loan model and examine the candidate, actual assertions and recorded observations. For a reproduced defect, retain the failing observation on its original basis and evidence that the corrected candidate addresses that case. The distinguishing checks include:

| Starting condition | Required evidence on the corrected candidate |
| --- | --- |
| Six open loans; all other borrowing prerequisites hold. | The seventh succeeds with the required normal effects. |
| Seven open loans; another copy is available and other prerequisites hold. | The eighth is refused; no loan is created, the copy remains available and no observable state changes. |
| Other applicable borrowing or return conditions. | Suspension, availability, counting rules and return behavior still satisfy the model; fixing the boundary does not waive them. |

A green suite with a wrong assertion or no exercised boundary is insufficient. The responsible Manager proposes resolution; only current independent review can establish closure under the existing finding rules. Old approval, an empty diff or generic exit code cannot replace required case evidence. Parents independently assess composition and cross-area behavior; local review does not replace integration, required aggregate checks, preflight, configured approval, guarded publication or actual read-back. No successful Apply may retain this blocking deviation or missing required assessment.

### Owner decisions and existing affected data

| Situation | Owner involvement |
| --- | --- |
| The seven-loan rule is clear and the application violates it. | No renewed decision about the limit. Diagnosis, routine implementation choices, review and bounded correction stay within the authorized repair. |
| Investigation finds conflicting intended rules or a necessary additional model change. | Receive the concrete source-bound question and recommendation. Decide the separate intent proposal; Apply cannot change the model to settle it. |
| Required authority, budget or configured publication approval is missing. | Decide the specific missing grant or choice. Existing valid authority needs no repeated approval; unresolved limits leave an honest incomplete result with retained findings and evidence. |
| Eight loans already exist because of the defect. | The code repair does not itself authorize deleting loans, forcing returns or editing live data. Any needed data correction follows established rules and its own explicit scope and authority; unresolved policy becomes a concrete owner question. |
| The result is ready for publication or acceptance. | Follow the existing project policy. Apply never commits, merges or pushes; the owner's later commit accepts the published artifacts, and any outer-agent repository delivery needs its own grant. |

If the defect is confirmed after a previously successful Apply, examine the relevant prior evidence to explain which case, assertion or assessment missed it. Propose a concrete distinguishing improvement within the existing test/review responsibility, rather than inventing a general approval institution. Preserve what the earlier evidence actually established; do not claim that a hypothetical walkthrough or prior green CI demonstrated defect-free behavior.

The final report separates reported and reproduced behavior, confirmed cause or remaining uncertainty, changed and retained areas, current independent evidence, finding resolution, outstanding decisions and actual publication. This record specifies that workflow only; no model fixture, application, live data or execution capability is changed here.
