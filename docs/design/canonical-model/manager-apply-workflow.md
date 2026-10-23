# Apply through Agent workspaces and reviewed handoffs

**Accepted target, 22 October 2026:** the owner adopted this replacement in [DEC-067](../../concepts/register.md#dec-067-agent-workspaces-and-reviewed-handoffs), with [verbatim source and scope](../../concepts/sources/agent-workspaces-and-handoffs-20261022.md). This page owns the target behavior. The [delivery plan](../../implementation-plan.md#apply-rebuild-plan) owns the rebuild sequence. This change is documentation only: the current executable and canonical self-model still describe the shared-workspace Pass implementation, whose behavior remains in the [runtime contract](manager-apply-runtime.md).

## Why the process changes

The [earlier process review](https://github.com/Glacius-Labs/Markitect/blob/a4a88c0345acfc84deeb37a3804c0f9f31de6575/docs/design/canonical-model/manager-apply-workflow.md#proportionate-process-review-22-october-2026) recorded protocol refusals, incomplete reviews and expensive probe coordination. Current source combines a shared work volume, accepted partial stands, dependency invalidation and short container-bound Turns. The owner wants the original collaboration model: each Agent works independently, hands its result to its parent, and the parent reviews and integrates it.

The challenged assumption is that more Host coordination and evidence bookkeeping necessarily improve semantic correctness. Responsibility and independent judgment remain; per-write enforcement and a formal proof protocol are not prerequisites. Improvement means completing the existing preservation/repair workflow, detecting real defects and false findings, with fewer protocol repairs and less duplicate work. Fewer fields or container starts alone do not establish improvement.

## One mandate per run

A run has one precise purpose. The same collaboration mechanism serves the following mandates; it does not prescribe Plan/Prepare/Implement Passes or a universal report envelope.

| Mandate | Work and result |
| --- | --- |
| Model assessment | Assess the fixed model for material contradictions and missing decisions. Return grounded problems or no identified blockers; do not implement or search for optional improvements. |
| Implement | Realize the fixed model across code, tests, documentation and configuration. Preserve conforming artifacts, test necessary changes and integrate the children. Local planning belongs to this work. |
| Verify | Independently assess one named integrated stand against the model, including the actual assertions of its tests. Return acceptance, concrete defects or unresolved required matters. |
| Improvement analysis | Only when explicitly commissioned, propose alternatives with benefits and costs. It grants no implementation authority. |

Apply retains model assessment before realization and an independent Verify of the final integrated result. They are separately assigned runs with distinct contexts and outputs, not hidden extra duties of Implement. This specifies no new CLI commands or configuration format. One owner invocation may sequence them, but all work and corrections consume its original total allowance; separate runs never reset a grant.

A confirmed material model problem stops the affected work and is reported through Root without publication. Corrections to intent happen outside execution, followed by a new model assessment. Ordinary implementation freedom stays with its owner. No external answer channel or preloaded owner decision is introduced.

## Agents, Managers and containers

The model supplies one ordinary Root and one accountable Manager for every required local Package, including empty and intermediate Packages. A Manager is an Agent responsible for its own area and for coordinating, reviewing and integrating its direct children. Required responsibilities cannot disappear because a parent considers them irrelevant.

At the beginning of Implement, the Host starts all required Manager containers across the whole tree concurrently, each with an independent repository working copy. A Manager's conversation starts only with its first task. Its parent supplies the model context, expected result, responsibility, fixed starting Git stand, contacts and budget. The same container, copy and conversation remain available through questions, waiting and corrections after Verify, until Apply ends or aborts. Model assessment and each independent Verify use fresh, separate Agents and conversations. This lifecycle clarification is part of the owner's 22 October 2026 acceptance criteria in the [delivery plan](../../implementation-plan.md#apply-rebuild-plan); the dated implementation and run evidence retain their own scope.

There are no domain Turns: no Turn budget, productive/communication Turn modes, repeated-progress classifier, report-to-continue ritual or container restart on every provider response. Provider requests, context management and completion events remain adapter details. An idle or finished provider response alone neither completes the assignment nor accepts its result.

Agents use ordinary file, shell, build and test tools in their own working copies. Responsibility, keeping intent fixed and independent review are instructions assessed at handoff. Per-file write overlays, read-only reviewer mounts, denied-write gates and technical freezing of a container are not required. The container remains a practical execution environment with explicit credentials and resource limits; this decision does not remove ordinary credential handling or cancellation.

## Delegation and questions

A Manager establishes sibling interfaces in its own workspace and conversation before giving its Go, without a separate plan run or stage. It then gives bounded tasks to all required direct children concurrently. Before the first or a later child task, the Host updates that child's existing copy to the exact integrated parent commit: a child without its own changes switches to it; otherwise normal Git merge preserves own work and leaves conflicts for the child to resolve. If later work needs a completed sibling result, the common parent may reassign the child after reviewing and integrating that sibling. The Agent, container and conversation continue. The Host delivers assignments and messages; it does not choose a semantic plan or replace the Manager's integration judgment. The same assignment mechanism handles additional explicitly commissioned helpers, without a separate helper-only Turn protocol.

The initial parent task explicitly names each required child as a separate Markitect Manager with its own container and working copy. Only the Host's `delegate_child` performs that delegation; a native provider subagent is no substitute. A leaf Manager receives its own bounded responsibility without a child-delegation mandate or child-coordination tools.

Questions and answers travel along direct parent/child edges. A cross-area question goes through the common parent, which settles it or consults the relevant child. If a Manager cannot decide within its model mandate, it escalates to its own parent. Root reports a material unresolved intent decision outside the run; it cannot invent policy.

A question needs an explicit signal so the Host can route it to the parent and return the answer. Ordinary work completion needs no separate Agent handoff tool or status message protocol.

A question to a working child is queued in arrival order and delivered after its current step. The parent tool receives an immediate queue acknowledgement; its provider response ends and releases the parent queue before the Host waits. The child answer resumes the same parent task or review in an ordinary follow-up message. This lets the parent answer another child's pending question while it waits, and avoids a crossed parent/child lock wait. Provisional review decisions are applied only after the queued answers have been delivered. Consultation, continuation and any decision correction share the original message allowance and timeout; cancellation and Close release all waiters.

A needed interface or contract may be supplied early as a named Git stand or an explicit decision. There is no universal basis submission or parent preapproval before local work. If a supplied foundation changes, the parent gives affected children a concrete update or correction task. The Host does not maintain a global basis/result dependency graph or automatically invalidate a tree of work.

## Handoffs and parent integration

When an Agent finishes its work, its current Git working stand is the handoff and its last message summarizes the result, relevant tests and open matters. The Host may secure that stand with an ordinary internal Git commit so the parent can inspect a fixed result against its starting commit. Further work creates another stand; it cannot alter an earlier result commit. No separate submission tool, mandatory JSON or status word is needed to receive the work. The parent decides acceptance or correction from the actual stand and summary. Internal work commits do not commit or push the owner's repository.

The Agent may test with its normal tools. The Host retains actual available commands, outputs and logs, distinguishing them from prose claims. Tests performed before later changes remain earlier observations. There is no mandatory per-Definition evidence matrix, custom receipt language or preregistration of every test expectation. The parent judges the assertions and whether the evidence is relevant to the submitted stand.

The parent inspects the child's actual changes, reviews its model conformance and tests, and integrates the result into its own working copy. An accepted handoff is not the mechanical overlay of an area onto a shared global volume. Git integration exposes conflicts; the parent routes necessary corrections to the owning child rather than silently taking over its responsibility. It checks the composed interfaces and shared invariants and runs relevant integration checks. It need not routinely repeat all child tests.

Children work concurrently. The parent reviews and integrates their fixed results serially in actual completion order, starting with the first result while other children may still be running. Child questions queue while that parent is reviewing or integrating another result. A returned correction runs in its owning child and rejoins the result queue; cancellation releases question waiters. Each result records the parent commit delivered for its task as its review base. Updating a copy for a later task cannot rewrite an earlier fixed result or its base.

Acceptance or return is a substantive parent decision. A return states the concrete gap and expected correction. The child corrects its own stand and hands it back. The parent checks the correction and affected integration; it need not restart every other Agent. A broken test script is corrected and reviewed with its retained observations. A real failure remains open until the responsible reviewer accepts its correction or explains why the observation was a false finding. No separate dispute/disposition state machine or complete global Verify solely to settle a script setup error is required.

The Host binds `accept_child` to the exact handoff currently presented for that child; the Manager supplies no commit argument. Invalid arguments receive concrete tool feedback. A missing, wrong-child, multiple or invalid delegation decision receives up to two ordinary feedback messages in the same assignment under its existing allowance; continued invalid decisions then end incomplete. Host diagnostics have fixed byte and entry bounds with visible omission markers. Material child answers reach the parent in full or fail with the provider channel's context-limit error before a decision is applied. The retained RunID04 trace exposed the need for this boundary when an Agent's own commit differed from the Host's fixed handoff. This correction preserves both Git identities and removes model-side selection of the integration commit; it does not establish acceptance of the failed run.

Each parent passes its reviewed, integrated result upward in the same way. The next parent reviews its integration changes too. Registry and file accounting remain part of a conforming result; responsibility is not permission to edit the model or ignore policy. Existing strict initial accounting stays in place. Removing per-write enforcement does not authorize optional cleanup or omit ordinary review of changed paths.

## Root verification and final transfer

Root integrates the entire required responsibility tree. A fresh independent Agent receives the fixed model, Root's result commit and the relevant child results, tests and open issues under a Verify mandate. It assesses the complete integrated result and challenges unsupported claims and cross-area behavior. It may receive focused assistance where needed; a second mandatory mirror of the entire Manager tree is not prescribed.

Verification judges material model obligations across all governed artifact kinds. Removing the old evidence protocol does not make green tests or a Manager's completion claim sufficient. Concrete unresolved required matters prevent acceptance. Root routes genuine corrections to the responsible owners, integrates them and submits the revised result for independent verification; the reviewer chooses relevant follow-up based on what changed. Mandatory project checks must pass on the final result after relevant changes.

Only the independently accepted Root result may be transferred to the owner's repository. Use the ordinary Git workflow: detect intervening user changes, expose conflicts and confirm what was transferred. No automatic owner commit, push, merge, release or deployment is authorized. A fresh review or unchanged earlier test cannot approve a different result stand. A partial or failed transfer is not success.

## Host responsibility and limits

The Host provides containers and working copies, delivery of assignments/questions/handoffs, ordinary Git integration support, retained results/logs and finite execution/cancellation. Managers judge meaning, responsibility and integration. Normal coordination is assignment, work, waiting, handoff, review and correction; this is not a new multi-axis state machine or schema specification.

Keep an overall deadline, assignment/command timeouts and resource limits. Waiting causes no repeated provider calls merely to announce unchanged progress. Limits do not depend on artificial Turn counts. Aborted work, observations and internal Git results remain inspectable; an interruption is incomplete, not silent success or automatic replay. No new recovery subsystem is commissioned.

## Library example

Root asks Library to realize the public application. Library supplies the callable contract and delegates lending behavior to Lending and storage to Storage on their own starting stands. A storage question travels to Library, which decides within the model or escalates. Each child tests and hands back its result commit. Library reviews and integrates both, checks borrowing through the public interface and returns its integrated stand to Root. Root integrates project-wide files and submits the result to fresh Verify. A discovered lending defect goes back to Lending; corrected integration is verified before transfer. A conforming no-change result follows the same responsibilities without invented edits.

## Implementation and evidence boundary

This target replaces the common Pass mechanism, shared writable tree, per-Turn containers, general basis/result invalidation, Turn progress/empty counters, mandatory probe receipts and automatic whole-tree correction loops of DEC-064. Keep canonical model authority, complete file accounting, required Manager responsibility, parent/child communication, preservation, substantive review/integration, fresh final verification, finite limits and visible incomplete work.

The [rebuild plan](../../implementation-plan.md#apply-rebuild-plan) names source seams, dependencies, removals and focused acceptance. Canonical self-model Rules, execution formats, generated views and executable guides migrate with the implementing slices; they are deliberately unchanged by this documentation-only decision. Extract has its own authority and workflow: inspect its shared consumers before removing old code, without extending this assignment to an Extract redesign. Cockpit, setup and reporting must follow actual implemented capabilities, not display this target as available.

## Earlier process links

The following anchors retain incoming references to the replaced 20-22 October Pass design. They are historical routes, not additional rules. The complete former text remains at [Main a4a88c0](https://github.com/Glacius-Labs/Markitect/blob/a4a88c0345acfc84deeb37a3804c0f9f31de6575/docs/design/canonical-model/manager-apply-workflow.md). Its original source statements and evidence keep their dates and outcomes. The current executable remains described by the runtime contract until rebuilt.

<a id="active-counterexample-assessment-in-the-same-pass"></a>
<a id="apply-workflow-and-recursive-manager-passes"></a>
<a id="authored-local-registries-and-one-derived-view"></a>
<a id="complete-workflow"></a>
<a id="configurable-local-manager-teams"></a>
<a id="conformance-mandate-and-observation-kinds"></a>
<a id="deferred-proposal-reuse-manager-working-containers"></a>
<a id="disposition-of-the-previous-claude-proposals"></a>
<a id="evaluation-that-can-refute-this-refinement"></a>
<a id="evidence-backed-acceptance-after-the-codex-retest"></a>
<a id="evidence-driven-hardening-adopted-20-october-2026"></a>
<a id="implement-pass"></a>
<a id="library-walkthrough"></a>
<a id="mandatory-model-assessment-and-closed-runs"></a>
<a id="model-context-delivery-and-completeness"></a>
<a id="observation-assumption-and-intended-improvement"></a>
<a id="one-contract-different-purposes"></a>
<a id="one-recursive-pass-mechanism"></a>
<a id="ownership-and-file-accounting"></a>
<a id="per-turn-container-baseline"></a>
<a id="plan-pass"></a>
<a id="prepare-pass"></a>
<a id="proportionate-process-review-22-october-2026"></a>
<a id="provisional-direct-baseline-21-october-2026"></a>
<a id="public-policy-implementation-freedom-and-s-q"></a>
<a id="purpose-and-vocabulary"></a>
<a id="questions-and-escalation"></a>
<a id="required-organization-and-explicit-areas"></a>
<a id="safe-acceptance-while-other-managers-work"></a>
<a id="semantic-coverage-and-parent-integration"></a>
<a id="shared-result-and-pass-identity"></a>
<a id="states-reports-and-events"></a>
<a id="strategy-boundary-and-future-extension"></a>
<a id="supersession-and-implementation-boundaries"></a>
<a id="the-base-process"></a>
<a id="tool-produced-evidence-without-false-proof"></a>
<a id="tools-and-method-guidance"></a>
<a id="verify-pass-and-correction"></a>

Earlier Pass, Turn, team and evidence rules are superseded by the assignment and handoff process above.
