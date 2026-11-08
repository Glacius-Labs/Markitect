# Delivery plan: Markitect

This plan owns the steps from the current source to [concept 4.0](concepts/v4-concept.md), how work is split between sessions, and how it is integrated. The [architecture](architecture.md) owns packages and contracts, and the [register](concepts/register.md) the decisions. The plan's history, from V1 through concepts 2.0 and 3.0 to the lean tool, is archived at [`d85e0fa1`](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/implementation-plan.md).

## Where Main is

Main builds concept 4.0, the one concept ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-one-concept)). The latest release, 0.25.3, carries:

- the Markdown spec language in `.markitect/spec/`, moved from `.markitect/model/` by `markitect update` ([DEC-080](concepts/register.md#dec-080-the-spec-is-typed-documentation-in-markdown)), with `check`, `tidy`, `vet`, `list` and `doc`, and `vet`'s warnings and notes ([DEC-102](concepts/register.md#dec-102-what-vet-measures));
- how links are written: a link is never decorative, a record that others name gets an identity Definition of its own, a Package's surface is its visibility, and `doc PACKAGE` shows a Package's public Definitions apart from its internal ones ([DEC-104](concepts/register.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them));
- the record of what was verified, `.markitect/review.sum`, with `status` and `mark` ([DEC-099](concepts/register.md#dec-099-a-record-of-what-was-verified));
- the skills `markitect-design`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and `/markitect-schedule`, four agents whose rights a hook holds, and the builder in a worktree of its own ([DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role));
- the strategies and the shipped parallel workflow, `harness.yaml` at user and project level, `markitect config` and `/markitect-config` ([DEC-103](concepts/register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)); since 0.25.2 a parallel strand's builder commits in its own worktree, and the workflow merges it by branch name and has it reviewed on the run's branch, as one real run on a small fixture showed, before the review's fixes ([record](validation/parallel-strategy-20261108.md)); `auto` still chooses sequential;
- releases with the Claude Code plugin and `markitect update` ([DEC-095](concepts/register.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)); since 0.25.1 the plugin's `markitect` resolves by name in Claude Code's Bash and PowerShell tools on Windows too ([record](validation/plugin-path-windows-20261108.md)); since 0.25.3 the project's hooks call `markitect` only where their shell finds one and leave the guard to the plugin, so a session with the plugin alone shows no hook error ([the hook](../internal/run/README.md#the-hook)).

All of this is covered by tests. No real agent run of the record, the skills or the roles has been read yet ([concept 4.0, §13](concepts/v4-concept.md#13-what-is-not-yet-shown)).

## Current steps

| Step | Result | Done when |
|---|---|---|
| 1. Try it | The owner uses it on a real project. A bounded dispatch tries the record, the skills and the roles on a small project with planted defects, including a weakened test that served as evidence and a builder's edit of the spec. For the parallel strategy it first probes, on Haiku, what the workflow assumes of Claude Code (a skill starts a saved workflow; a builder's worktree branches from the session's worktree; its reviewer works in that worktree; the hook sees the workflow agents' types), then builds a fixture project of a few Packages in parallel, with one planted bug and one planted contradiction. The Makerspace project can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks) | The planted defects are caught or their misses are explained, and what the run showed is recorded under `docs/validation/` |
| 2. Make it reliable | The 5× project's eight change steps on Sonnet, with their hidden cases, which are at an [archive point](archive.md#removed-with-the-evaluation-tools). Failures are fixed in the tool contract, not with more prompt text ([DEC-051](concepts/register.md#dec-051-agents-get-tasks-not-process)) | Two clean passes in a row: every step ends with nothing left to verify without the owner, every hidden case passes, and no refusal of the hook was overridden. Whether that is the finish line is open ([OQ-014](concepts/register.md#open-questions)) |
| 3. The rest | The `security` and `improve` skills; a CI check with `markitect status` that ships with Markitect; `vet`'s zone and risk measures once real records exist; the rest of the rename of "model" to "spec" in the code, after the folder `.markitect/spec/` ([DEC-080](concepts/register.md#dec-080-the-spec-is-typed-documentation-in-markdown)), with no compatibility kept; a driver for Codex once Claude's is reliable | The owner's reading |

Before step 2, read the nine finished launches of the earlier reviewing test at the 5× size. They are in `%LOCALAPPDATA%\markitect-test1`, not in the repository, and their answers are unread. They may say at no new cost whether one reviewer catches known bugs at that size.

## Authorizations

- Real agent runs of the skills, and any container or provider run, need a concrete bounded dispatch under the [standing evidence authorization](concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded.
- Releases follow the [release operations](../packaging/README.md#release-operations); the owner allowed releasing on 6 November 2026 ([DEC-095](concepts/register.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)). No deployment.
- Source tests and smokes are bounded evidence, not agent-run, release or owner acceptance.

## Work streams

Writing sessions run in parallel as far as is useful and their exclusive file scopes are disjoint ([DEC-070](concepts/register.md#dec-070-parallel-writers-where-useful)). Each dispatch records the exact integrated SHA, branch and worktree, local changes, writer and exclusive paths before writing. Shared contracts and generated files, such as the architecture rule table and the [code map](development/code-map.md), have one writer at a time, named in the dispatch. The coordinating session owns the concept, this plan, reviews and integration, and does not edit a writer's files.

## Integration

Keep the development policy: a focused PR, meaningful local checks, an independent complete-diff review, freeze the reviewed head, merge the exact head without bypassing protection, then read Main back. Recompose and review substantive overlapping changes.

CI runs on Linux, on request (`gh workflow run ci.yaml --ref BRANCH`), with quality checks and the race detector in about five minutes; the dispatching session checks its result once ([DEC-077](concepts/register.md#dec-077-linux-ci-run-on-request)). Absent hosted execution is never reported green. Release evidence stays separate.

Use isolated worktrees from verified integrated bases and non-protected scoped branches, preserve unrelated work, and never use `git stash` across worktrees. No parallel Go commands or source edits during a gate in the same checkout. [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns routine checks. Markitect's own repository keeps no spec of itself ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-an-ordinary-software-project)): a change updates its canonical documentation page, and generated pages such as the code map are regenerated, not edited.
