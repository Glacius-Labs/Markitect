# Delivery plan: a usable Apply

This is the delivery and coordination owner. It names the current target, the work streams with their write scopes, and the integration rules. Chats and PRs hold progress and evidence; there is no status file, backlog or dashboard. The plan that led to the first reliable Apply, with its A1 to A6 queue and zones, remains in [Git at Main f15ea723](https://github.com/Glacius-Labs/Markitect/blob/f15ea723d519d541c7122568b078cbd49bd00cf1/docs/implementation-plan.md).

## Where we are

Status, 19 October 2026 (Main `f15ea723`):

- Main holds the recursive Apply of [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) with execute-first ([DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify)), enforced ownership and the outside oracle ([DEC-059](concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)) and the owner-realizes rule ([DEC-060](concepts/register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)), plus a first bounded Extract. The [runtime contract](design/canonical-model/manager-apply-runtime.md) describes what is implemented.
- Three timed library demonstrations verified both Applies in one Verify round each ([record](validation/recursive-apply-repetitions-20261019.md)). The first Extract runs work as a mechanism; the extracted model is not yet adoptable as binding intent ([record](validation/library-extract-20261019.md)).
- Not yet established: Apply without the demonstration harness, a repair of injected drift under an unchanged model, a real owner question, an interrupted run, a second example, a release and owner acceptance.

## Target and baseline

**Next milestone: Apply without the harness.** An owner prepares the Host once and then runs `markitect apply` and `markitect extract` on their own repository with product commands only. Today the scripts of the [library demonstration](../experiments/recursive-apply-demo/README.md) build the images, bind the Host profile and write the run configuration; that becomes product.

The milestone is demonstrated on the [Roombook](../experiments/test-projects/roombook/README.md) (empty start, case 1) and [Readinglog](../experiments/test-projects/readinglog/README.md) (existing application, case 2) change series, run through product commands only, including a real owner question and an interrupted run. Each Apply is assessed afterwards by an independent oracle outside the process, as in [DEC-059](concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle). A release follows only with explicit owner authority.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) stays the language baseline. Compatibility is not required: obsolete product paths, formats, tests and documents are deleted together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)).

## Work streams

<a id="architect-and-coordinator"></a>

Three sessions work in parallel. This continues the owner's [division of 17 October 2026](concepts/sources/architecture-review-20261017.md#division-of-work) ("Konzept und Herzstück" for the core, "die Dinge aussenrum" for the Coordinator). On 19 October 2026 the owner [assigned the Coordinator everything around the core and selected the milestone above](concepts/sources/coordinator-assignment-20261019.md). The write scopes below are the sessions' agreement within that assignment; the owner can change them. Each session owns its write scope; anything outside it goes through the owning session.

| Stream | Write scope | Next |
| --- | --- | --- |
| **Apply core** | `internal/run/`, `internal/apply/` except Extract, `internal/realization/`, `internal/organization/`, the `apply` wiring in the CLI and bootstrap, the [runtime contract](design/canonical-model/manager-apply-runtime.md) except its Extract section, the run and apply guides | Library scenarios on a frozen build: baseline, repair of injected drift with an unchanged model, a real owner question, budget and deadline end, abort. Engine fixes for the observed waste and rough edges. |
| **Extract** | `internal/apply/extract*.go`, the Extract sections of the runtime contract and the [command design](design/canonical-model/commands.md), the Extract harness scripts, the register entry for the Extract concept decisions | Project-level obligations, intent versus realization choice, proposed checks, a readable adoption subject and the handoff of known deviations to Apply. |
| **Coordinator (around the core)** | This plan, the vision, workflow and operations guides, the CLI and bootstrap outside the `apply` wiring, Host setup, `packaging/`, the adapters below the ports, `experiments/test-projects/`, the self-model, release tooling and repository hygiene | See the sequence below. |

The Coordinator's sequence:

1. **Cleanup.** This plan, superseded design documents to [history](history/README.md), stale status lines, merged worktrees and branches.
2. **Runtime image.** Python 3 in the base image and a C toolchain in the Go example image, so that `go test -race` works.
3. **Host setup.** A product command that builds and binds the images and writes today's Host profile and run configuration, so that `apply` and `extract` need no harness. It produces the existing formats and changes none of them.
4. **Evidence.** Roombook and Readinglog through product commands only, with their oracles.
5. **Self-model.** Adopt the accepted [preparation-to-publication proposal](design/canonical-model/reconciliation-model-proposal.md) into `.markitect/`.
6. **Standard library.** Implement the selected [standard-library design](design/canonical-model/README.md#standard-library-design).
7. **Release and marketing,** after the milestone and with explicit owner authority.

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
