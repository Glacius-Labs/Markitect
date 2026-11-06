# Plan: a lean Markitect

**Status:** written on 6 November 2026 at the owner's request, after a discussion of what Markitect should own. The owner accepted the direction the same day ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)): the tool computes and checks, a driver acts, and the command that checks is called `gate`. The other [decisions](#decisions-for-the-owner) have recommendations and are open. Once built, this revises [draft concept 4.0](v4-concept.md) §5 to §9. [DEC-001](register.md#dec-001-delegated-realization-is-central-to-the-product) stands: delegated realization stays the purpose, and the tool no longer runs it.

## The idea

Markitect computes and checks. It does not drive agents and does not move branches.

- **It computes:** the spec compiles, every file is accounted for, what a change reaches, the complete checklist for that change, and how good the spec is.
- **It checks:** every item has an answer with a pointer that exists, for exactly the state that is to be committed; nothing the work may not touch was touched; `check` passes. Then it writes the Result.
- **Everything else is the driver:** who builds, who reviews, in what order, how often. Skills, agents, a workflow script or a person. Markitect ships one default driver, as skills and agent files, and any other may replace it.

Delegated realization stays the purpose. What changes is where it lives: the tool owns the obligations and the check before a commit, the host owns the agents.

## Why

**The value that can be shown today is on the deterministic side.** The spec language, the compiler, file accounting, impact and `lint` work and are tested. The benefit of the delegated method is still unmeasured, by the vision's own account ([current basis](../vision.md#current-basis-and-unproven-benefit)).

**The weight is in the run, not in the spec.** Lines of Go, without tests, on Main at `96273ee3d`:

| Part | Lines |
| --- | ---: |
| The spec side: `model`, `realization`, `project`, `analysis`, `libraries` | about 2,830 |
| The run engine: `internal/run` | about 3,020 |
| Git for runs: `internal/adapters/rungit` | about 660 |
| What the agents are told: the harness's skill and agent files, and the task texts in `internal/run/strategy.go` | about 170 |

Within `internal/run`, the run lifecycle (`engine.go`, 640 lines), `submit` (530), the store (320), landing (240) and the trailers (210) are most of it. What tells the agents what to do is small; the machinery around a run is the bulk.

**The failures came from that machinery.** Background sessions, containers, landing in the owner's folder while they work, Git on Windows, and now a builder that has to start the reviewer as a nested subagent, which is not yet shown to work ([concept 4.0 §14](v4-concept.md#14-what-is-not-yet-shown)).

**A lean tool is easier to judge.** Any host can drive it, so the comparison with a strong agent and ordinary documentation can swap only the driver.

## What stays

| Command | What it does | Change |
| --- | --- | --- |
| `init`, `check`, `lock`, `refs`, `graph`, `impact`, `lint` | The spec and its realization record | None |
| `install`, `update`, `version`, `licenses` | Setup and upgrade | None |
| `checklist [--base REV] [--full] [--findings FILE] [--review]` | The list for the change between the last certified commit and the folder; `--review` prints the reviewer's task | Needs no open run; finds its base in history |
| `answer [FILE]` | Stores a reviewer's answers for the folder's current state; each pointer must exist in a snapshot of that state | Keyed by the state, not by a run |
| `gate [--commit]` | `submit`'s checks without the landing (below) | New, replaces `submit` |
| `hook pretool` | Only the reviewer may store answers | None |
| `status` | Read-only: the answers for the current state, and the commits since the last certified one | No run to report on |

**`gate`** does the first thing that applies, as `submit` does today, minus what a run needs:

| | It finds | It says, and exits |
|---|---|---|
| 1 | The spec (except registry pages) or the harness differs from `HEAD` | Put them back; spec changes are committed before the work, by a person or the modeling skill. Exit 1 |
| 2 | A reviewed state, and the only difference is new files not listed | List or remove them. Exit 1 |
| 3 | An error from `check` | The errors. Exit 1 |
| 4 | Items without an answer for this state | Which; start a reviewer. Exit 1 |
| 5 | A `spec problem` | The question for the owner. The Result says so. Exit 1 |
| 6 | A `wrong` answer | The items and what is wrong; repair. After two repair rounds it refuses, and the Result says so. Exit 1 |
| 7 | Nothing left | Writes the Result under `.markitect/results/` in the folder. Exit 0. With `--commit`, commits the work and the Result on the checked-out branch, and nothing else |

## What goes

- `apply` and `verify` as commands that open a run; `submit`; `stop`.
- The run store and its lock, and "one run per folder".
- The deadline and the ceiling of 30 calls (the driver's business; `gate` keeps only the two repair rounds, which are a rule about state, not time).
- Run branches `markitect/runs/*`, kept runs and their retries.
- Landing: building the landing off to the side, merging into a moved branch, bringing the files of the folder up, retries.
- The `Run-*` trailers, replaced by the Result in the commit.
- The notice lines about runs that closed.

Rough estimate, to be confirmed by doing it: `internal/run` from about 3,000 to about 1,000 lines, `rungit` from about 650 to about 250, and `runs.go` in the CLI to a third.

## The state that remains

- **In the repository, versioned:** the spec and the Results.
- **Outside the folder, under `<git dir>/markitect/answers/`:** the answers, keyed by the base, `HEAD` and the snapshot of the folder. An answer whose files the next change touches falls away, as a repair review works today.
- **Certification:** a commit that adds a passing Result whose recorded tree is the commit's tree, apart from the Result, is certified. The newest certified ancestor is the default base of the next `checklist`. A landing commit with the `Run-Work` trailer counts as certified, so existing projects keep their base.

A commit without a Result is not prevented, as today: it stays in the next checklist, `status` counts it, and a CI job can run `markitect gate --verify COMMIT` where no agent reaches.

## The default driver

`/markitect-apply`, a skill with the same name as today, run in the owner's session:

1. If the spec has uncommitted changes, stop and say so.
2. Start `markitect-builder`. It takes its task from `markitect checklist`.
3. When it ends, run `markitect gate` and do what it printed: start `markitect-reviewer`, which answers with `markitect answer`, or start the builder again with the items that are wrong.
4. When `gate` passes, run `markitect gate --commit`, and report.

Every step comes from what `gate` prints, so the process stays in the tool's output and the skill stays "do what it printed". If a subagent can start another, the builder may start the reviewer itself, as today, and keep its context through the repairs. That is a choice of the driver, not of the tool.

`/markitect-verify` becomes `markitect checklist --base HEAD~1 --review`, the reviewer's answers, and `markitect gate --findings`, which writes the Findings page into the folder and commits nothing.

Later drivers: a workflow script, which holds the loop and the limits in code; another provider.

## What we give up, and what holds instead

| What the owner wants | Today | Lean |
| --- | --- | --- |
| What is committed is complete | The tool commits only after its checks | The same with `gate --commit`. A commit made around it is visible afterwards, and a CI check can refuse it |
| A run has a limit | Deadline and ceiling in the tool | Two repair rounds in `gate`; time and calls in the driver |
| The work doesn't mix with the owner's | One run per folder, a landing that stops at an edit of the owner | `gate` snapshots the whole folder; the owner keeps their edits out of it while a driver works |
| The branch moved during the work | The landing merges, if clean | Ordinary Git: the commit goes on the checked-out branch, and the owner merges or rebases as usual |
| A crash in the middle | The run is stored and resumes | The answers are stored per state; run `gate` again |

## Order of work

1. **Decide.** Done for the boundary and the name ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)); the other decisions are below. The concept follows once step 3 holds.
2. **Build beside the old.** Add `gate`, the certified base and the state-keyed answers next to today's engine. Port the tests of `submit`'s table. Nothing is removed yet.
3. **Try it.** Switch the default driver to the lean commands and run a few changes on Makerspace, with planted defects. Compare with today's flow: items answered, planted defects caught, interventions by the owner, time.
4. **Remove.** If it holds: the run lifecycle, landing, run branches and trailers go; docs, concept, code map; a release.
5. **Later.** A workflow driver; a CI check of certified commits.

## Decisions for the owner

1. **The boundary:** the tool computes and checks, a driver acts. Decided by the owner on 6 November 2026.
2. **`gate --commit`:** should the tool still make the commit? Recommended: yes, a plain commit on the checked-out branch, so "what is committed is complete" stays the tool's, at almost no cost.
3. **Limits:** recommended: two repair rounds in `gate`, time and the number of calls in the driver.
4. **The name:** `gate`, decided by the owner on 6 November 2026, after weighing `certify`. `submit` was set aside because it meant landing.
5. **Where answers live:** recommended: outside the folder, in the Git directory, as today.
