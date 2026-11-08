# The parallel strategy, run for real — 8 November 2026

The smoke session of 0.25.0 ran `/markitect-build` with `strategy.build parallel` on a small fixture, and nothing was verified or merged. The workflow's builders could not start their reviewer, since Claude Code gives the agents of a workflow no tool to start an agent. The merge step was refused, since it ran `git -C` on the strands' worktrees from the run's worktree, which Claude Code's worktree isolation does not allow. This record is the run of the fix in 0.25.2 ([DEC-103](../concepts/register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow), amended). In it, each strand is steps of the workflow: a builder that commits in its own worktree, a merge of its branch by name, a reviewer on the run's branch, and a step that commits the verdicts.

**What the run shows:**

- **The structure holds on one small project.** The session entered a worktree and started `markitect-parallel`. The workflow ran one wave of two strands, `inventory` and `pricing`, then Root. Each builder committed in its own worktree and returned its folder and branch. The merge step merged the branches by name, and nothing was refused. Three reviewers verified their Packages on the run's branch, and only they ran `markitect mark`. The workflow stopped "empty" after one round, with no strand left unmerged and none unreviewed. The session then merged the run into `main` and removed the worktrees.
- **The planted defect was caught.** The fixture's `pricing/total.go` gave the bulk discount above 10 items, where the spec says 10 or more. The pricing builder found it against the spec and fixed it with tests, and its reviewer verified the fix. As in the sequential smoke run, the builder caught it before review.
- **The list went from 6 to 0.** Afterwards, `markitect status` listed nothing, `check` passed, `go test ./...` passed, the spec was unchanged, and `git status` was clean. `init` had put `.claude/worktrees/` into `.gitignore`.
- **One fault was found and fixed after the run.** The step that commits the verdicts ran `git commit -- .markitect/review.sum`. Git refuses that while the record is a new, untracked file, as it is in a fresh project. The verdicts reached a commit only in the session's final commit. The step now adds the file first. That was checked on a scratch repository and in the simulation, but not in a second agent run.
- **Friction, not a fault.** Claude Code refused two builders' compound `cd … && git add -A && git commit …` commands as too complex to verify. The builders split them and went on. The sequential smoke run saw the same.

## Source and method

**Source.** Branch `fix/parallel-reviewer-merge`, built as `markitect 0.26.0-dev+819a5b66` (Main `32ebe716` with the fix). It ran before the record step's fix and the review's fixes.

**Fixture.** The smoke kit's Shop: a Go module with 6 Definitions in Root and two Packages, `pricing` and `inventory`. Every file is on its registry page, `inventory/reserve` has no code yet, and `pricing/total.go` carries the planted defect. Its tests pass, so only reading against the spec finds it. The project has no record yet, so `status` lists all 6 Definitions. `markitect init` from the build wrote the harness, then `markitect config strategy.build parallel` was set and committed.

**Run.** One headless `claude -p "/markitect-build"` run of Claude Code 2.1.319 on Windows 11:
- model Sonnet 5.5, effort high, permission mode `auto`;
- project settings only, so the hooks were the ones `init` wrote;
- `git push` and `gh` refused;
- `timeout 1800` as the abort;
- the binary first on the `PATH` of that one process only.

It ran from 13:00:14 to 13:03:17 UTC and exited 0. The workflow took 155 seconds with 11 agents: the status step, three builders, two merge steps, three reviewers and two record steps. It was authorized by the coordinating session as an addendum to the smoke dispatch: one run, Sonnet, at most 30 minutes.

**Tests.** `internal/skills/testdata/simulate.mjs` runs the workflow against stub agents under `go test`. It covers:
- waves, batches, Root and the stop rule;
- the order of builder, merge, reviewer and record;
- conflicts, uncommitted work, a builder that changed nothing, and a strand the merge step leaves out;
- a builder that names a wrong branch, and steps that return nothing.

Mutations of the workflow that undo each fix were run, and each made it fail. `internal/bootstrap` tests the line `init` and `update` add to `.gitignore`.

## Limits

- One run, on one small project of 6 Definitions and three strands, on one Windows machine. It shows that the steps fit together under Claude Code's rules. It shows nothing about long lists, several waves, merge conflicts or a builder that fails. So `auto` still chooses sequential.
- The builder fixed the defect before review. The run did not show a reviewer failing an item and a later round repairing it.
- The record step's fix and the review's fixes came after the run.
