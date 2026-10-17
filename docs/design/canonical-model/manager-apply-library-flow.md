# Library Apply propagation and integration

This walkthrough applies the [fixed Manager process](manager-apply-concept.md#tasks-and-delegation) to [M1](library-example/README.md). It replaces the earlier root-planned order and parent-authored prompts under [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process). The exchanges explain the owner-confirmed Explore, local planning and limited-delivery process, not an implemented scheduler or a real successful Apply. Review-budget treatment remains an open decision; exact technical bindings must enforce the selected process.

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
| 4 Explore, then plan locally | All Managers inspect their area without changing artifacts and report changes, prerequisites and questions. Parents summarize their direct children. Only after every initial assessment is recorded may writing start. Each Manager then plans locally; no global Plan barrier delays an independent ready branch. | Agent interprets local obligations; Host enforces participation, the initial write barrier and recorded readiness. |
| 5 Resolve prerequisites | For a missing project skeleton, lending sends a request to root, which routes it to library. Library owns the callable interface and required project files. Its limited normal delivery receives independent review and Host acceptance, then is composed into lending's base. Library's final wiring remains open. | Host enforces edges, reviewed scope and exact consumer basis. Library chooses its owned implementation; no root-authored global work plan is needed. |
| 6 Work and seal | Lending works on an allowed current base, reserves new paths and proposes registrations. Host observes the full actual delta and protected inputs and seals candidate C1. | Lending implements; Host observes and checks authority. |
| 7 Review and correct | Fresh reviewer first records expectations from M1, then sees C1 and actual checks. Suppose code, tests and docs all permit six loans. The five-loan model rule produces a finding. Lending corrects it; candidate C2 receives current independent review. | Reviewer judges meaning; Host enforces exact subjects, finite rounds and evidence. |
| 8 Integrate reminders | Reminders supplies its independently reviewed result or no-change result. Lending receives a summary and evidence, may request targeted detail, and integrates it with local work. A fresh integration review checks due dates and exclusion of returned loans. | Lending judges composition; Host refuses completion with missing children or stale evidence. |
| 9 Integrate at root | Root receives current reviewed results from all four direct branches. Library finishes required wiring using current interfaces. Root's fresh integration review checks cross-area behavior, including suspension preventing borrowing while return remains possible. | Root and reviewer judge the aggregate; Host requires all areas and evidence. |
| 10 Final verification and publication | Required aggregate checks, Host evidence check, source preflight, any configured owner approval, guarded write and actual read-back complete the operation. | Host establishes procedural completeness and actual effects. The owner's later commit accepts the result. |

This is an illustrative successful exchange sequence, not a prescribed AI plan. A dependency can change which work is ready. The Host chooses the oldest eligible item, with stable ties, within one-writer and budget limits. Explore, planning and reviews on immutable subjects can run concurrently. After the initial Explore barrier, a waiting lending Manager does not prevent ready members work or library's prerequisite work. Waiting holds no active write grant. In this model the skeleton owner is a sibling reached through root. No Manager can be declared complete merely because its prerequisite was useful or its files did not change.

## Questions across areas

If lending needs a catalog API correction, the route is lending to root to catalog, with the reply returning catalog to root to lending. Lending cannot call catalog directly. A reminder question to members travels reminders to lending to root to members and back along those edges. Explicit model references provide context, not communication or write privileges.

If the responsible area is unknown, the receiving parent uses explicit facts or a bounded inquiry through its direct branches. Recipients and pending replies stay visible; the request cannot disappear because a Manager guesses it is irrelevant. A missing necessary model relationship remains a modeling error. Inquiry does not manufacture that relationship or silently augment Apply context to compensate for it.

## What each handoff carries

**Operation handoff:** unchanged operation identity and bytes, Host-bound run and Manager responsibility, direct parent-child route, separately compiled context, resolved settings and current candidate basis. A later message cannot narrow the original obligation.

**Parent hint:** separate attributed advice with its grounds, such as "catalog reports a changed interface; clarify it before implementing the consumer". It cannot add a binding rule, rewrite Apply or change ownership. Summaries carry results, interfaces, prerequisites, findings, uncertainty and evidence, with targeted details available on request.

**Question or dependency:** originating Manager, concrete question, relevant model facts and current basis, direct recipient, outstanding replies and the result needed to resume. Exact record schemas follow process clarification, not this example.

**Work proposal:** summary, Definition assessments, proposed registry changes and open work. Host supplies the observed delta and sealed candidate. No-change has the same assessment and independent review duties. A prerequisite contribution names only its supported reviewed scope and keeps final obligations open.

**Review proposal:** exact subject and purpose, verdict, findings, model grounds and observed evidence references. A pass cannot accept changed bytes or an incomplete subtree.

**Accepted delivery:** Host-issued exact model/candidate/registry bindings, complete reviewed scope, child delivery references and evidence. A limited scope may be usable while the Manager remains open; dependent implementation waits until this delivery is actually composed into its base. Local success does not replace integration or final verification.

## Sessions, integration and observation

Managers are message-driven sessions. Known sessions can continue only on current bound inputs and workspace; otherwise an explicitly related replacement receives recorded context. Independent reviewers start fresh. Eager or lazy physical startup does not change mandatory participation. Uncertain activity blocks handoff; no automatic resend repairs a lost response.

Parents begin with direct-child summaries and evidence. Targeted reads remain possible when necessary; they do not receive every descendant transcript by default. The Host records supplied messages, context, observed effects and exact candidate transitions. Metrics derive from those records: waits, requests between areas, correction rounds and integration findings can reveal recurring coordination costs. Metrics do not automatically reorganize Packages or prove benefit.

Host composition preserves exact bases and ownership. A parent can edit its own integration artifacts, never a child's files. Corrections return through the tree to the owner. Changed assumptions or content require renewed affected assessment, and unknown consequences broaden review. An unresolved circular wait or model decision becomes visible incomplete work rather than an endless message loop.

## Concrete refusal cases

| Case | Required result |
| --- | --- |
| Root omits lending in its output | Host still propagates to lending and reminders; a missing observed receipt cannot count as completion. |
| Lending proposes writing while reminders has not reported its initial Explore assessment | Host refuses writing until the complete initial assessment census is present. |
| Library's skeleton has a reviewer pass but no Host acceptance or composition into lending's base | Lending's dependent implementation stays blocked; unrelated eligible work may proceed. |
| An Agent calls a changed input irrelevant and reuses its old review | Host requires an unchanged complete bound subject or renewed assessment; changed aggregate integration remains current. |
| Parent rewrites Apply to cover borrowing but not returns | Refuse narrowed operation; the full compiled local responsibility remains binding. |
| Empty or one-child Package is optimized out | Refuse completion without that Manager's explicit assessed result and required review. |
| Root sends directly to reminders, or lending sends to catalog | Refuse the non-adjacent call and use the required tree route. |
| A child waits for a parent-owned prerequisite | Permit ready owned parent work; do not require child completion first. |
| A useful skeleton contribution is reported as final completion | Refuse until the owner's remaining duties and current review/integration are complete. |
| Lending writes unreserved or protected inputs | Refuse the candidate; later registration cannot authorize the earlier write. |
| C1 passed but C2 is delivered | Refuse the mismatched subject and require current assessment. |
| Reminders or an outstanding dependency reply is unresolved | Do not accept the lending subtree or root aggregate. |
| A necessary relationship is absent from the model | Record the modeling error; do not infer context or edit the model during Apply. |
| All Agents say done but checks or integration evidence are missing | Refuse success. |
| Source inputs drift or publication is partial | Preserve observations; do not return Applied. |

These are controlled process cases for later implementation. They neither authorize real provider runs nor replace the two real library Applies.
