# Delivery plan: finish the Apply and Extract evidence

This is the delivery and coordination owner. It names the current target, the work streams with their write scopes, and the integration rules. The GitHub milestone issues hold the bounded work and dependencies; chats and PRs hold progress and evidence. Do not create a second status file or dashboard. The plan that led to the first reliable Apply, with its A1 to A6 queue and zones, remains in [Git at Main f15ea723](https://github.com/Glacius-Labs/Markitect/blob/f15ea723d519d541c7122568b078cbd49bd00cf1/docs/implementation-plan.md).

## Where we are

The original implementation baseline was Main `f15ea723`; later integrated changes and their exact runtime evidence stay bound to their delivery PRs and reports.

- Main holds the recursive Apply of [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) with execute-first ([DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify)), enforced ownership and the outside oracle ([DEC-059](concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)) and the owner-realizes rule ([DEC-060](concepts/register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)), plus a first bounded Extract. The [runtime contract](design/canonical-model/manager-apply-runtime.md) describes what is implemented.
- Three timed library demonstrations verified both Applies in one Verify round each ([record](validation/recursive-apply-repetitions-20261019.md)). The first Extract runs work as a mechanism; the extracted model is not yet adoptable as binding intent ([record](validation/library-extract-20261019.md)).
- The current task is to finish and assess the bounded evidence below. An integrated implementation, a successful source check and owner acceptance remain distinct.

## Target and baseline

**Current scope, owner direction of 19 October 2026: finish the Apply and Extract evidence before widening scope** ([original instruction](concepts/sources/coordinator-assignment-20261019.md#checkpoint-evidence-first)). Only [Apply #492](https://github.com/Glacius-Labs/Markitect/issues/492) and [Extract #495](https://github.com/Glacius-Labs/Markitect/issues/495) are active delivery objectives.

- **Apply:** show a model change, repair of drift with unchanged intent, a justified owner question with the answer taken over, and budget/deadline/interruption outcomes with visible Incomplete, retained evidence/work and the supported clean restart. Confirm guarded publication, read-back and unchanged protected inputs. Reuse existing successful cases and state any remaining proof on the final integrated build.
- **Extract:** finish the already-running comparison on fixed old/new builds. Assess missing project/delivery obligations, incidental implementation choices made binding, and wrong or unnecessary escalations. A no-change round trip alone does not establish model quality. Report improvement only where the comparison demonstrates it.

Preserve running attempts, evidence and completed candidates. Change only what fixes an observed gap of these proofs. Do not add roles, review stages, process architecture or a new test matrix for hypothetical later needs. Do not multiply similar runs without a concrete unanswered question. Keep source checks, real-run evidence, Main integration and owner acceptance separate.

Finish each bounded series with a short result: what passed, failed or remains unproved; the exact builds; and the smallest justified correction, if any. A negative or inconclusive result ends that series too; it does not automatically start a redesign or more runs. After both reports are assessed, the owner selects the next scope. This is the current scope decision under the existing working method, not another permanent approval round or runtime mechanism.

**Later candidates, not an automatically advancing queue:** Apply without the harness and the `--host` entry, the Roombook/Readinglog change series, self-model and standard library, an Extract adoption product and between-round structure changes, new Skills/UI, delegated model maintenance, release and marketing. Their issues remain blocked by #492 and #495. Finished PRs and fixtures retain their state; their completion starts no follow-up work.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) stays the language baseline. Compatibility is not required: obsolete product paths, formats, tests and documents are deleted together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

## Work streams

<a id="architect-and-coordinator"></a>

Three sessions work in parallel. This continues the owner's [division of 17 October 2026](concepts/sources/architecture-review-20261017.md#division-of-work) ("Konzept und Herzstück" for the core, "die Dinge aussenrum" for the Coordinator). On 19 October 2026 the owner [assigned the Coordinator everything around the core and originally selected Apply without the harness](concepts/sources/coordinator-assignment-20261019.md). The later [scope decision](#target-and-baseline) now takes priority. The write scopes below are the sessions' agreement within that assignment; the owner can change them. Each session owns its write scope; anything outside it goes through the owning session.

| Stream | Write scope | Next |
| --- | --- | --- |
| **Apply core** | `internal/run/`, `internal/apply/` except Extract, `internal/realization/`, `internal/organization/`, the `apply` wiring in the CLI and bootstrap, the [runtime contract](design/canonical-model/manager-apply-runtime.md) except its Extract section, the run and apply guides | Finish and assess #492 on frozen builds: reuse existing successes, complete the missing owner-question and budget/deadline/interruption/restart evidence, and fix only observed blockers. |
| **Extract** | `internal/apply/extract*.go`, the Extract sections of the runtime contract and the [command design](design/canonical-model/commands.md), the Extract harness scripts, the register entry for the Extract concept decisions | Finish the existing fixed-build comparison and report #495 against the three known defect classes. Separate adoption/handoff products and structure changes stay deferred. |
| **Coordinator (around the core)** | This plan, the vision, workflow and operations guides, the CLI and bootstrap outside the `apply` wiring, Host setup, `packaging/`, the adapters below the ports, `experiments/test-projects/`, the self-model, release tooling and repository hygiene | Support only an observed blocker of #492/#495; preserve existing candidates. The former sequence below is deferred. |

The Coordinator's earlier sequence (cleanup, runtime image, Host setup, broader examples, self-model, standard library and release/marketing) is retained in [the original assignment](concepts/sources/coordinator-assignment-20261019.md). It does not dispatch the next item during this checkpoint. Existing work and ownership are preserved; a preparation optimization such as #491/#506 is not a prerequisite for the two proofs.

## How we work

The owner-adopted [working method](development/working-method.md) applies: learn through complete workflows and give short owner accounts at milestones ([DEC-038](concepts/register.md#dec-038-learn-through-complete-workflows)).

### Sessions

Each session owns its scope, its pull requests and its merges. Implementation work runs in an isolated worktree from the integrated base, with one bounded task, owned paths and acceptance criteria. Sessions coordinate directly; actual state, candidate SHAs, reviews and results live in each chat and PR.

**Shared files** have one writer at a time and are announced to the other sessions before an edit: `go.mod`, `internal/adapters/cli/command.go`, `internal/bootstrap/main.go`, the model and registry shards outside a stream's own packages, the architecture and module gates and the code map. The run configuration, `.markitect/execution.yaml` and the Host profile formats change only after notice to the Coordinator, because Host setup produces them. Port changes belong to the Apply core.

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
