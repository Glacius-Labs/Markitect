# Delivery plan: Markitect

<a id="v1-implementation-organization"></a>

## Target and current state

No concept is the target for realization right now ([DEC-094](concepts/register.md#dec-094-a-fresh-start-for-realization)). The aim is one clean and simple version that works reliably on Sonnet, no longer proof that one version is better. Main holds the model language and its compiler, the model commands (`init`, `check`, `lock`, `refs`, `graph`, `impact`, `lint`), the skills and Markitect Desktop with its chats. It has no `apply` or `verify` and no evaluation suite. This plan owns the steps and the integration rules. The [architecture](architecture.md) owns concrete packages and contracts.

## Steps

| Step | Result | Done when |
|---|---|---|
| 1. Fresh start | The Command engine, the evaluation suite and the pages that described them are removed. They stay at the [archive point](archive.md) `archive/main-pre-native-20261104` | Main builds, its tests pass, and no maintained page names them as current |
| 2. Concept for `apply` | The owner and the Project discussion session talk through how `apply` should work, starting from a native version: a harness of skills and rules per provider, with the agent following the process | The owner accepts a concept, recorded in the register |
| 3. Build it | The concept's first version on Claude, on Sonnet | The concept's own conditions |
| 4. A simple evaluation | The owner and the session talk through a much simpler evaluation, then it is built and used to make the version reliable | The owner's reading |

Steps 2 and 4 are conversations first: nothing is designed or built for them before the talk.

## Earlier plans

The steps of concepts 1.0, 2.0 and 3.0 (T1 to T9, U1 to U5, the comparison studies, the model track and the repository cleanups) are in [the plan at the archive point](https://github.com/Glacius-Labs/Markitect/blob/10b074e4fe82c70c8219fb2fa5a0bacee4986ee1/docs/implementation-plan.md).

## Markitect Desktop (owner decision 31 October 2026)

The owner decided on Markitect Desktop, the app over the CLI that users know as Markitect ([DEC-089](concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli)). Its [design](design/markitect-desktop.md#delivery) owns the slices and their order, and the [Desktop session](workstreams/desktop.md) builds them in its own scope: `desktop/`, `internal/adapters/desktop` and `cmd/markitect-desktop`. Shared files, such as `internal/bootstrap`, the CLI's `desktop` verb, the [architecture rule table](../internal/tooling/architecture/rules.go) with the code map, the source packager and CI, change one writer at a time. The Runs area waits for the new concept.

<a id="work-streams"></a>

## Work streams

Writers run in parallel as far as is useful and their exclusive scopes are disjoint ([DEC-070](concepts/register.md#dec-070-parallel-writers-where-useful)). Each dispatch records the exact integrated SHA, branch and worktree, dirty state, writer and exclusive scope.

## Integration

Keep the existing development policy: focused PR, meaningful local checks, independent complete-diff review, freeze the reviewed head, merge the exact head without bypassing protection, then read Main back. Recompose and review substantive overlapping changes. CI runs on Linux, on request (`gh workflow run ci.yaml --ref BRANCH`), with Linux quality and the race detector in about five minutes; the Windows job is being removed ([PR #760](https://github.com/Glacius-Labs/Markitect/pull/760)); the dispatching session checks its result once ([DEC-077](concepts/register.md#dec-077-development-pace-and-linux-first-execution)). Final or release evidence remains separate. Absent hosted execution is never reported green. A quiet machine is needed only for counted study runs; during a native run nobody runs the full test suite or Docker image builds. The day's owner decisions are recorded in one batched PR.

Use isolated worktrees from verified integrated bases, non-protected scoped branches, preserve unrelated work and revalidate ownership before changes. No parallel Go commands or source edits during a gate in the same checkout. [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns routine checks. Markitect's own repository keeps no self-model or registry ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); a change updates its canonical documentation page, and generated pages such as the code map are regenerated, not edited.

## Historical references

Historical work orders are outside the normal implementation entry and grant no work or execution. The S0–S5 rebuild plan and its evidence log are in [Main `334e7c31`](https://github.com/Glacius-Labs/Markitect/blob/334e7c3133ff294c0b827e623d80185e3d3ac9af/docs/implementation-plan.md). The earlier integrated plan is at [source snapshot 447ac3b](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/docs/implementation-plan.md); [Run 08](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/apply-assignments-luna-20261023.md) retains its own scope. The [pre-cutover archive](archive.md#bundle-of-the-pre-cutover-archive-point) stays verified and is not deleted.
