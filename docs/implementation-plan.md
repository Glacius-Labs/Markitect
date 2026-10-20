# Delivery plan: stabilize Apply on the new Pass core

This is the delivery and coordination owner. It names the current target, the work streams with their write scopes, and the integration rules. The GitHub milestone issues hold the bounded work and dependencies; chats and PRs hold progress and evidence. Do not create a second status file or dashboard. The plan that led to the first reliable Apply, with its A1 to A6 queue and zones, remains in [Git at Main f15ea723](https://github.com/Glacius-Labs/Markitect/blob/f15ea723d519d541c7122568b078cbd49bd00cf1/docs/implementation-plan.md).

## Where we are

Main `780dccb7eda4f10e70d1f5513f5e43bd031346b1` integrated the DEC-064 implementation in [#530](https://github.com/Glacius-Labs/Markitect/pull/530). Apply now uses one recursive Pass process for Plan, Prepare, Implement and independent Verify, with explicit parent acceptance, checked ownership/local registries and guarded publication/read-back. The [accepted workflow](design/canonical-model/manager-apply-workflow.md), [DEC-064](concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams) and [runtime contract](design/canonical-model/manager-apply-runtime.md) own its behavior. The implementation handoff is historical preparation, not an outstanding implementation assignment.

- **Current real-run evidence:** the [first Pass-process report](validation/apply-pass-process-20261020.md) records model-only realization, semantic drift repair, a model-error stop, a Verify/correction/fresh-Verify cycle and Sonnet completing a six-Manager example. It also records a Verified depot result that retained the known valid-name defect, optional changes, substantial coordination overhead and Haiku exhausting its turn budget on serial dependencies. Most cases ran once per development build. These results do not establish stable Apply.
- **Independent trial inputs:** #526 prepared 13 cases. The trial stream is correcting faulty references/oracle coverage and translating the selected inputs and expectations to DEC-064. Its unmerged work and any earlier-core series are not evidence for current Main. A reference labelled conforming must actually conform before its changes can be called overreach.
- **Historical closeout:** [#492](https://github.com/Glacius-Labs/Markitect/issues/492) and [#495](https://github.com/Glacius-Labs/Markitect/issues/495) are closed. Their [closing record](validation/apply-extract-closeout-20261020.md) retains the earlier core's Library model-change and no-change proof and Extract comparison. Those results are not a successful rerun on the rewritten shared core. Extract has not been exercised in a real run on the Pass implementation, and the earlier extracted models still need owner correction.

Source checks, real-run evidence, Main integration, packaging and owner acceptance remain separate. The current source tests passed locally; #530 reports no hosted CI or release, and a failing Windows packaged-source smoke. Its origin outside #530 does not remove its relevance to a later Windows pilot.

## Current checkpoint: concrete Apply corrections and new-core evidence

**Owner direction, 20 October 2026: finish the concrete Apply weaknesses and the evidence on the new core before expanding** ([verbatim instruction](concepts/sources/coordinator-assignment-20261019.md#apply-pass-stabilization-first-20-october-2026)). This takes precedence over the deferred pilot sequence below. It selects no new process architecture and starts no additional session or provider series by itself.

**Current phase: owner assessment and concept refinement.** The owner clarified that all Claude sessions have ended ([source](concepts/sources/coordinator-assignment-20261019.md#assessment-and-concept-refinement-20-october-2026)). Preserve their candidates, attempts and self-check/review outputs, without claiming unfinished work passed. No implementation writer or provider series is active or dispatched by this plan. The findings below remain priorities for a later bounded assignment; the ongoing discussion may refine their solution and order. Do not resume an old session or start a replacement automatically.

Recorded follow-up needs, subject to that assessment:

1. **Finish the independent trial-kit correction under [#523](https://github.com/Glacius-Labs/Markitect/issues/523).** Assess and preserve the existing review/self-check outputs, candidates and attempts; distinguish completed and missing evidence. Correct the valid hyphen-leading-name references, make the oracle distinguish the known failures and partial repairs, and align selected packets with current profiles, ownership and Pass budgets. Under DEC-064 the initial File check is strict: accounting defects refuse before work, while registered semantic drift remains Apply work. Clarification is distinct from an actual erroneous or contradictory model. Reuse the existing projects; no new complexity families, parameter sweep or broad old-core series.
2. **Close the observed correctness/preservation weakness in [#531](https://github.com/Glacius-Labs/Markitect/issues/531).** Explain and reproduce how a Verified publication retained the genuine depot defect. Correct the smallest responsible source or mandate behavior. Required repair and model-required evidence must be distinguishable from optional hardening, restructuring or extra tests. Prove preservation on an independently checked conforming stand, not on a faulty reference. Protect genuine finding closure, model authority, ownership and immutable Verify.
3. **Dispose of the observed dependency/wait overhead in [#532](https://github.com/Glacius-Labs/Markitect/issues/532).** Use existing events/transcripts to separate productive work, submission/parent acceptance, turn/container latency and blocked waits. Correct only demonstrated unnecessary serial dependencies or repeated unproductive blocked work while preserving required questions and prerequisites. Report actual time/turns and unknown usage/costs honestly; no generic scheduler redesign, model tournament or arbitrary speed promise.
4. **Close one bounded integrated-build evidence assignment in [#533](https://github.com/Glacius-Labs/Markitect/issues/533).** It depends on the independent input corrections and both core findings' dispositions. Freeze product, model, profile and image identities and a small named case list before running. Reuse equivalent existing evidence with an explicit source/input argument. Include model-only to actual model change to conforming no-change, targeted multi-Manager semantic drift/preservation, and exact-build gaps for model-error stop, finite limits/interruption/restart and publication safeguards. Record every selected attempt and remaining uncertainty. Do not repeat the whole matrix until green.

The report closes the bounded series even if negative or incomplete; it must name passed, failed and unproved obligations and each concrete finding's disposition. A failed series is not permission to expand, redesign or keep spending. Further corrective work needs a named observed gap; broader work waits for assessment and the next bounded scope choice. This is the existing owner scope decision, not a new permanent approval mechanism.

**Deferred during this checkpoint:** wider Extract adoption/features or another Extract series; the Roombook/Readinglog product-command series; new Skills/UI; distribution/release/pilot packaging; self-model or standard-library expansion beyond a correction directly required by Apply; new project/complexity families; and delegated model maintenance. Preserve completed candidates and historical evidence. Focused regression checks needed to protect the shared engine are part of an Apply correction, not a new Extract delivery stream.

Reconcile active guide/register/entrypoint evidence status with #530 and the real-run report as part of #533. Current statements that the implementation is branch-only or has no provider runs are stale. Do not rewrite dated old-core reports as evidence for the new process.

## Target and baseline

**Destination remains the usable owner pilot accepted on 20 October 2026** ([roadmap](https://github.com/Glacius-Labs/Markitect/issues/501#issuecomment-6284960943), [source](concepts/sources/coordinator-assignment-20261019.md#accepted-pilot-roadmap-20-october-2026)). The owner can use a reproducibly installable, pinned version in an existing project while Markitect development continues. Apply is the critical path; a reviewed project model can enable an Apply-only pilot if Extract is not ready.

The current checkpoint precedes the remaining pilot route:

1. Reach the coherent, evidenced Apply baseline above, correcting only observed weaknesses.
2. After its assessment, select a bounded Extract adoption and original-repository round-trip workflow through #496/#499. Earlier completion scores and rebuilds do not establish it on the new core.
3. Make setup, modeling, Apply, progress/results and failure handling usable through #500/#501. Choose Skills, Agents or UI under #502 only for demonstrated friction.
4. Prepare and verify the pinned pilot build/setup path through #503. Select the owner's actual project and its supported language/runtime, size and model needs; validate a real change carried forward from its existing state. Public release remains separate.

M1 supplies the modeling foundations needed by that project, not a blanket requirement to finish every proposed Kind. M2 supplies dependable Apply within the demonstrated scope. M3 supplies Extract onboarding when ready, with reviewed model authoring as the first-pilot fallback. M4 supplies the complete usable workflow and reproducible setup. M5, delegated model maintenance, follows initial real use.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) remains the language baseline. Compatibility is not required: obsolete product paths, formats, tests and documents are removed together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

## Work streams

<a id="architect-and-coordinator"></a>

One writer owns each shared scope. Record the actual session, branch, integrated base and reservations before a new source change. This checkpoint does not create replacement sessions or a second core writer. Earlier [17/19 October](https://github.com/Glacius-Labs/Markitect/blob/0c28097c030bc5d031b501002bc85634cd66a4c8/docs/implementation-plan.md#work-streams) and [20 October](https://github.com/Glacius-Labs/Markitect/blob/780dccb7eda4f10e70d1f5513f5e43bd031346b1/docs/implementation-plan.md#work-streams) stream assignments remain historical.

| Stream | Write scope | Current bounded task |
| --- | --- | --- |
| **Plan and issue reconciliation** | This plan, the existing owner-assignment source and GitHub coordination metadata | Record this owner checkpoint and current integration/evidence status; preserve other sessions' candidates. |
| **Independent trial kit (#523)** | `experiments/apply-trials/`, its existing registry/navigation entries and trial reports | Session ended. Assess the preserved DEC-064 adaptation, reference/oracle correction and self-check/review outputs; later work needs a selected assignment. No automatic continuation. |
| **Apply core (#531, #532)** | Existing Apply/Pass/ownership/Verify source and the directly affected contracts, self-model, registry, guides and tests | No active writer. Later assignment: one owning stream, sequential corrections selected from the current assessment. No automatic implementation or expansion. |
| **New-core evidence (#533)** | Frozen run inputs, outside evidence and the existing validation/docs owners | Deferred during assessment and concept refinement, then dependent on corrected inputs/core dispositions and a selected finite assignment with recorded authority and limits. |
| **Historical closeout (#492/#495) and process review (#524)** | Preserve their reports, accepted sources and PRs | Closed results, not active assignments or current-core runtime acceptance. DEC-064 implementation is merged through #530. |

The wider Coordinator, Extract, UX, release and model-maintenance queues remain deferred. Readiness of a fixture or merged implementation alone dispatches no next item.

## How we work

The owner-adopted [working method](development/working-method.md) applies: learn through complete workflows and give short owner accounts at milestones ([DEC-038](concepts/register.md#dec-038-learn-through-complete-workflows)).

### Sessions

Each session owns its scope, its pull requests and its merges. Implementation work runs in an isolated worktree from the integrated base, with one bounded task, owned paths and acceptance criteria. Sessions coordinate directly; actual state, candidate SHAs, reviews and results live in each chat and PR.

**Shared files** have one writer at a time and are announced to the other sessions before an edit: `go.mod`, `internal/adapters/cli/command.go`, `internal/bootstrap/main.go`, the model and registry shards outside a stream's own packages, the architecture and module gates and the code map. The run configuration, `.markitect/execution.yaml` and the Runtime profile formats change only within an assignment that names their writer, because `markitect setup` produces them. Port changes likewise need an assignment for the Apply core.

### Branch names

Use an isolated managed worktree and a non-protected branch from the integrated base. Branch prefixes describe the scope, such as `dev/`, `fix/` or `docs/`. Never stash, reset unrelated work, force-push reviewed branches or implement on Main. Verify path, branch, full SHA and local changes before writing.

### Integration

**Development CI policy, owner decision, 19 October 2026:** automatic CI is paused and the complete pipeline stays available manually. The observed delay was a runner queue of more than twelve minutes for a reviewed documentation change. Focused local validation and independent complete-diff review are the default merge basis; release validation still runs the complete workflow.

1. One focused PR per change. The delivering session obtains an independent complete-diff review and resolves every finding.
2. Freeze the approved head and record focused local validation for it. A changed head needs renewed review and checks.
3. The delivering session merges its own PR with a merge commit and `--match-head-commit`. Never bypass branch protection.
4. Recompose only when GitHub reports conflicts or the PR overlaps substantively with newly merged work; then review the delta and renew the relevant checks.
5. Record the merge in the chat and continue. React when Main is reported red; investigate the actual failure first.

[CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) lists the checks. Source tests and smokes are bounded evidence; they do not establish a provider run, a release or owner acceptance.

### Session and process contracts

Every Agent session and project check runs in its own Linux container ([DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)); the [runtime contract](design/canonical-model/manager-apply-runtime.md#ports-and-adapters) and the [runtime image guide](../packaging/runtime/README.md) own the details. Real provider runs need explicit credentials and the owner's grant; they stay outside the repository, and their evidence is recorded in [validation](validation/README.md).

## Owner decisions

Ask the owner only for genuinely new intent, authority or an exceptional tradeoff; implementation choices within the accepted design proceed. This plan does not authorize unlimited provider spending, studies, releases or deployments.

## Releases and historical evidence

The owner retired the old release pages and downloads on 17 October 2026; tags, identities and attestations are preserved in the [retirement record](history/distribution-retirement-20261017.md). [Operations](operations.md) owns release work. Old [handoffs](development/session-handoff.md) and [surveys](history/surveys/README.md) keep their scope and cannot restart queues.

## Historical roadmap links

These anchors route old references to immutable records.

<a id="astra-assessment-disposition-and-next-method-checkpoint"></a>
[Astra disposition](history/implementation-plan-before-product-integration-20261009.md#astra-assessment-disposition-and-next-method-checkpoint).

<a id="standard-operating-model-checkpoint"></a>
[Earlier operating-model checkpoint](history/implementation-plan-before-product-integration-20261009.md#standard-operating-model-checkpoint).

<a id="read-only-policy-failure-analysis-unreleased-source"></a>
[Earlier policy-failure assessment](history/implementation-plan-before-product-integration-20261009.md#read-only-policy-failure-analysis-unreleased-source).

<a id="v0100--canonical-engineering-model-published"></a>
[v0.10.0 historical release](history/implementation-plan-before-product-integration-20261009.md#v0100--canonical-engineering-model-published).

<a id="knowledge-graph-assessment-and-authoring-decision"></a>
[Earlier knowledge-graph assessment](history/implementation-plan-before-product-integration-20261009.md#knowledge-graph-assessment-and-authoring-decision).
