# Delivery plan: toward a usable owner pilot

This is the delivery and coordination owner. It names the current target, the work streams with their write scopes, and the integration rules. The GitHub milestone issues hold the bounded work and dependencies; chats and PRs hold progress and evidence. Do not create a second status file or dashboard. The plan that led to the first reliable Apply, with its A1 to A6 queue and zones, remains in [Git at Main f15ea723](https://github.com/Glacius-Labs/Markitect/blob/f15ea723d519d541c7122568b078cbd49bd00cf1/docs/implementation-plan.md).

## Where we are

The original implementation baseline was Main `f15ea723`; later integrated changes and their exact runtime evidence stay bound to their delivery PRs and reports.

- Main holds the recursive Apply of [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) with execute-first ([DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify)), enforced ownership and the outside oracle ([DEC-059](concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)), the owner-realizes rule ([DEC-060](concepts/register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)), a strict initial File check ([DEC-064](concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams); it supersedes the repairable start drift of DEC-062) and model defects that end the run without owner questions ([DEC-063](concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). It also holds the bounded Extract with [DEC-061](concepts/register.md#dec-061-what-an-extracted-model-contains) and its round-2 refinements. The [runtime contract](design/canonical-model/manager-apply-runtime.md) describes what is implemented.
- The Apply and Extract evidence series of 19 and 20 October 2026 are closed. The [closing record](validation/apply-extract-closeout-20261020.md) states what passed, failed or remains unproved, on which build. An integrated implementation, a successful source check, a real run and owner acceptance remain distinct; owner acceptance is recorded for neither series.

## Accepted Apply refinement and its implementation

On 20 October 2026 the owner [adopted the complete workflow discussion](concepts/sources/terminology-and-file-accounting-20261020.md#owner-adoption-and-implementation-preparation). [DEC-064](concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams) and the [accepted Apply workflow](design/canonical-model/manager-apply-workflow.md) now govern the target. They supersede initial accounting repair and the blanket question ban in the implemented baseline. Plan, Prepare and Implement are explicit recursive Passes; fresh independent Verify remains separate. Naming, local registry/ownership checks, lifecycle/events, tools and bounded local teams are part of the same accepted change.

The [Claude handoff](development/claude-apply-implementation-handoff-20261020.md) prepares one bounded implementation assignment, including shared CLI/bootstrap/Run contracts, relevant canonical self-model and registries, container/provider adapters, tests and current guides. Implementation starts from fetched Main containing DEC-064 and its polished common base process, with the actual full starting SHA recorded from Git. The historical preparation base was Main `a0928a1b7a95ebbc23836e5a5f472b29d233466e`. No Claude session or product-code writer is started by this preparation. The future implementation session uses its own worktree and claims the named shared seams as one writer, after checking actual live scopes. Preserve the separate trial preparation stream and frozen closeout evidence. Provider runs still need a separate explicit finite grant; no release or unlimited series is authorized.

The implementation session started on 20 October 2026 from Main `b59abad30409601075f4a597457b707cbf47ba72`, later composed with the three accepted clarifications of Main `6594fee87e3394e5ddc23267ebed730ce514247b`, in its own worktree on `dev/manager-pass-process`. Its PR carries the review, checks and evidence; until it is merged, the [runtime contract](design/canonical-model/manager-apply-runtime.md) describes that branch. `experiments/apply-trials/` still uses the earlier execution format (`layout`, `rounds`, `--host-profile`) and its drift-repair expectations; its owning stream adjusts the selected cases before a new run.

## Target and baseline

**Destination, accepted by the owner on 20 October 2026: a usable owner pilot** ([roadmap](https://github.com/Glacius-Labs/Markitect/issues/501#issuecomment-6284960943), [source](concepts/sources/coordinator-assignment-20261019.md#accepted-pilot-roadmap-20-october-2026)). The next cross-milestone delivery target is a reproducibly installable, versioned pilot build that the owner can use in an existing project while Markitect development continues. Apply is the critical path; dependable Extract is strongly desired in parallel. The owner may start an Apply-only pilot with an explicitly reviewed model if Extract is not ready.

The sequence and priorities below follow the comment's wording; the comment governs.

1. Assess and close the current evidence: [#492](https://github.com/Glacius-Labs/Markitect/issues/492) completes the canonical Library decisions and one frozen-build M1 → A1 → M2 → A2 → no-change workflow ([closing record](validation/apply-extract-closeout-20261020.md)). [#524](https://github.com/Glacius-Labs/Markitect/issues/524) explains the current process and discusses simplification candidates read-only. [#523](https://github.com/Glacius-Labs/Markitect/issues/523) prepares external trial cases only.
2. Reach a clean, coherent source, documentation and operation baseline. Apply any process refinement only when concrete findings justify it, preserve required properties, and check the affected workflow. Run selected prepared trials to establish a supported scope: model-only realization, actual model changes, unchanged-model drift, complexity and distributed inconsistency, honest model-defect stops, and preserved conforming areas. Do not start an unlimited matrix or redesign loop.
3. In a bounded parallel Extract stream, complete the workflow existing repository → reviewable model proposal → owner correction and adoption → conforming no-change Apply → genuine model change. Reuse #496 and #499; the completed comparison [#495](https://github.com/Glacius-Labs/Markitect/issues/495) alone does not establish this outcome.
4. Make the complete owner path comfortable: setup, modeling, Apply, progress and results, and failure handling. Reuse #500 and #501. Choose Skills, Agents or UI under #502 for demonstrated user friction; no comprehensive UI is a prerequisite for first use.
5. Prepare and verify a pinned pilot build and its setup path with #503. Select the owner's actual project early, assess its language, runtime, size and model needs, and validate one real change carried forward from its existing state. Record corrections, intervention, failures and remaining limits. The owner deliberately updates the pilot version while core development continues separately.

Milestone priorities:

- **M1** supplies the modeling foundations needed by the selected pilot project; finishing every proposed standard-library Kind is not a blanket pilot prerequisite.
- **M2** supplies dependable Apply within the demonstrated supported scope.
- **M3** supplies reliable onboarding through Extract when ready; reviewed model authoring is the explicit fallback for the first pilot.
- **M4** supplies a complete usable workflow and reproducible pilot setup.
- **M5**, delegated model maintenance, follows initial real use.

The older full-workflow dependencies in #501 remain the broader roadmap context. They do not require all Extract capability or all example series before a deliberately bounded Apply-only pilot; the exact pilot criteria are reconciled when the project is selected. Source checks, real runtime and behavior evidence, packaging and owner acceptance stay distinct.

**Execution boundary:** the earlier #492/#523/#524 scopes below retain their recorded boundaries and evidence. The owner has now accepted the process refinement and requested a prepared Claude assignment as described above; that assignment is ready to start in a separate session. Preparation does not start that session, broader provider series or a public release.

Where the evidence stands, in short (details and builds in the [closing record](validation/apply-extract-closeout-20261020.md)):

- **Apply:** a model change, drift repair and a repeated Apply with `no_change` are shown on the final Apply code, on one library example with Claude Haiku. A model defect that ends the run, budget, deadline, interruption and restart are shown only on earlier builds. Other projects and other agent families are unproved.
- **Extract:** two projects complete reliably, and Extract asks only necessary questions. The library models are not adoptable without owner correction: they bind internal wiring, mostly miss the test-coverage obligation and keep policy values as prose. Adoption, the handoff of deviations to Apply and a round trip on an existing repository are not implemented or unproved. The round-2 clarification of DEC-061 awaits the owner's confirmation.

Preserve attempts, evidence, retained work volumes and completed candidates. Keep source checks, real-run evidence, Main integration and owner acceptance separate. The other milestone issues stay blocked until a bounded assignment selects them. Finished PRs and fixtures retain their state; their completion starts no follow-up work.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) stays the language baseline. Compatibility is not required: obsolete product paths, formats, tests and documents are deleted together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

## Work streams

<a id="architect-and-coordinator"></a>

On 20 October 2026 the owner ended the Apply core, Extract and Coordinator sessions of the checkpoint and [divided the remaining work](concepts/sources/coordinator-assignment-20261019.md#closeout-and-division-20-october-2026) as below. The earlier division of 17 and 19 October, with the three streams' write scopes, is retained in [the plan at Main 0c28097c](https://github.com/Glacius-Labs/Markitect/blob/0c28097c030bc5d031b501002bc85634cd66a4c8/docs/implementation-plan.md#work-streams). Each session owns its write scope; anything outside it goes through the owning session. The product code has no active writer until a bounded assignment names one.

| Stream | Write scope | Task |
| --- | --- | --- |
| **Closeout (#492)** | This plan, the issue texts of #492, #495 and their follow-ups, the [closing record](validation/apply-extract-closeout-20261020.md), the Library example model and its demonstration harness `experiments/recursive-apply-demo/` | The closing record, one bounded confirmation run and the issue reconciliation; no further runs. |
| **Trial-case preparation ([#523](https://github.com/Glacius-Labs/Markitect/issues/523))** | `experiments/apply-trials/`, its entries in the examples registry shard and its catalog rows in the experiment guides | Trial cases and independent oracles only; it runs no Apply and no provider. 13 cases are prepared and self-checked in [`experiments/apply-trials/`](../experiments/apply-trials/README.md) (#526), with no Apply result yet. Which cases run belongs to the next bounded assignment. |
| **Process review ([#524](https://github.com/Glacius-Labs/Markitect/issues/524))** | Read-only, in the owner's chat | Explains the current Apply workflow and assesses simplification candidates. |
| **Accepted Apply implementation (in delivery on `dev/manager-pass-process`)** | Apply/model/organization/realization/Run, required CLI/bootstrap and container/provider seams, corresponding self-model/registry/guides/tests as bounded by the [handoff](development/claude-apply-implementation-handoff-20261020.md) | Implement DEC-064 in a separate Claude worktree; verify live shared-file scopes before writing. No provider spending or release is granted here. |

The owner has selected the DEC-064 refinement from the conversation; its implementation assignment is prepared in the linked handoff. Larger measurement series still require a bounded selection. The Coordinator's earlier sequence (cleanup, runtime image, Host setup, broader examples, self-model, standard library and release/marketing) remains in [the prior delivery plan](https://github.com/Glacius-Labs/Markitect/blob/30bc6e87e0e7cf26df7f290003b7941d3d375df4/docs/implementation-plan.md#work-streams) and dispatches nothing.

## How we work

The owner-adopted [working method](development/working-method.md) applies: learn through complete workflows and give short owner accounts at milestones ([DEC-038](concepts/register.md#dec-038-learn-through-complete-workflows)).

### Sessions

Each session owns its scope, its pull requests and its merges. Implementation work runs in an isolated worktree from the integrated base, with one bounded task, owned paths and acceptance criteria. Sessions coordinate directly; actual state, candidate SHAs, reviews and results live in each chat and PR.

**Shared files** have one writer at a time and are announced to the other sessions before an edit: `go.mod`, `internal/adapters/cli/command.go`, `internal/bootstrap/main.go`, the model and registry shards outside a stream's own packages, the architecture and module gates and the code map. The run configuration, `.markitect/execution.yaml` and the Runtime profile formats change only within an assignment that names their writer, because Host setup produces them. Port changes likewise need an assignment for the Apply core.

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
