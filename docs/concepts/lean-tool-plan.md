# Plan: a lean Markitect

**Status:** written on 6 November 2026 at the owner's request, after a discussion of what Markitect should own. The owner accepted it the same day in two steps: the tool computes and checks, a driver acts, and the command that checks is called `gate` ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)); and Markitect uses no Git at all, so agents or people do every Git step ([DEC-097](register.md#dec-097-markitect-uses-no-git)). The open [decisions](#decisions-for-the-owner) have recommendations. Built and tried, it revised [draft concept 4.0](v4-concept.md) on 7 November 2026, which now describes release 0.20.0. [DEC-001](register.md#dec-001-delegated-realization-is-central-to-the-product) stands: delegated realization stays the purpose, and the tool no longer runs it.

## The idea

Markitect computes and checks a folder. It does not drive agents and does not touch Git.

- **It computes:** the spec compiles, every file is accounted for, what a change reaches, the complete checklist for that change, and how good the spec is.
- **It checks:** every item has an answer with a pointer that exists; every file that changed lies in a reviewed item; the spec did not change while the work was done; `check` passes. Then it writes a certificate, the Result.
- **Everything else is the driver:** who builds, who reviews, in what order and how often, and every Git step: commits, branches, merges, diffs. Skills, agents, a workflow script or a person. Markitect ships one default driver, as skills and agent files, and any other may replace it.

Delegated realization stays the purpose. What changes is where it lives: the tool owns the obligations and the check, the host owns the agents and the repository.

## Why

**The value that can be shown today is on the deterministic side.** The spec language, the compiler, file accounting, impact and `lint` work and are tested. The benefit of the delegated method is still unmeasured, by the vision's own account ([current basis](../vision.md#current-basis-and-unproven-benefit)).

**The weight is in the run and in Git, not in the spec.** Lines of Go, without tests, on Main at `96273ee3d`:

| Part | Lines |
| --- | ---: |
| The spec side: `model`, `realization`, `project`, `analysis`, `libraries` | about 2,830 |
| The run engine: `internal/run` | about 3,020 |
| Git for runs: `internal/adapters/rungit` | about 660 |
| Reading the repository through Git: `internal/adapters/repo` | about 470 |
| What the agents are told: the harness's skill and agent files, and the task texts in `internal/run/strategy.go` | about 170 |

Within `internal/run`, the run lifecycle (`engine.go`, 640 lines), `submit` (530), the store (320), landing (240) and the trailers (210) are most of it.

**The failures came from that machinery.** Background sessions, containers, landing in the owner's folder while they work, Git on Windows, and now a builder that has to start the reviewer as a nested subagent, which is not yet shown to work ([concept 4.0 §14](v4-concept.md#14-what-is-not-yet-shown)).

**A lean tool is easier to judge and to use.** Any host can drive it, with any version control or none, so the comparison with a strong agent and ordinary documentation can swap only the driver. Its tests need folders, not repositories.

## What stays

| Command | What it does | Change |
| --- | --- | --- |
| `check`, `lock`, `refs`, `graph`, `lint` | The spec and its realization record | The file set comes from the folder, not from Git (below) |
| `impact [--base RESULT]` | What a change reaches | The base is a certificate, not a commit |
| `init` | Writes the project's files | Makes no repository and commits nothing |
| `install`, `update`, `version`, `licenses` | Setup and upgrade | `update` writes and commits nothing |
| `checklist [--base RESULT] [--full] [--findings FILE] [--review]` | The list for the change between the newest certificate and the folder; `--review` prints the reviewer's task | Needs no open run |
| `answer [FILE]` | Stores a reviewer's answers for the folder's current state; each pointer must exist in the current files | Bound to the files it points at, not to a run |
| `gate [--verify] [--findings] [--spec-problem WHY]` | The checks below; writes the certificate; `--spec-problem` is the builder's report (decision 7) | New, replaces `submit` and `stop` |
| `hook pretool` | Only the reviewer may store answers | None |
| `status` | Read-only: the answers for the current state, and the files that changed since the newest certificate | No run to report on |

**`gate`** does the first thing that applies:

| | It finds | It says, and exits |
|---|---|---|
| 1 | The spec (except registry pages) changed since `checklist` started, or the harness changed | Put them back; a change to the spec is its own step, before the work. Exit 1 |
| 2 | The builder's report of a spec problem, not yet answered (decision 7) | Start a reviewer to confirm it. Exit 1 |
| 3 | A `spec problem`, confirmed or found by a review | The question for the owner. The Result says so. Exit 1 |
| 4 | A file that changed since the base certificate and lies in no reviewed item | The files; list them for review or revert them. Exit 1 |
| 5 | An error from `check` | The errors. Exit 1 |
| 6 | Items without a valid answer for the current files | Which; start a reviewer. Exit 1 |
| 7 | A `wrong` answer | The items and what is wrong; repair. After two repair rounds it refuses, and the Result says so. Exit 1 |
| 8 | Nothing left | Writes the certificate under `.markitect/results/`. Exit 0. The driver commits the work and the certificate together |

`gate --verify` checks that the folder matches its newest certificate, for a checkout in CI or anywhere no agent reaches. `gate --findings` closes a review of finished work (today's `verify`): it writes a Findings page into the folder and no certificate.

## What goes

- `apply` and `verify` as commands that open a run; `submit`; `stop`.
- The run store and its lock, and "one run per folder".
- The deadline and the ceiling of 30 calls: the driver's business. `gate` keeps only the two repair rounds, a rule about state, not time.
- Every use of Git: committing the spec, run branches `markitect/runs/*`, kept runs, landing and merging, the `Run-*` trailers, `git init` and the commits of `init` and `update`, and reading files and history through Git.
- The notice lines about runs that closed.

Rough estimate, to be confirmed by doing it: `internal/run` from about 3,000 to about 1,000 lines; `rungit` goes; `repo` from about 470 to about 150 (a folder walk and a `.gitignore` reader); `runs.go` in the CLI to a third. The release tooling (`internal/tooling/release`, `internal/guardedwrite`) packages Markitect's own source from Git and is not part of the product; it stays.

## Without Git

**The file set.** Markitect walks the project folder. It skips what the project's `.gitignore` files exclude, read as plain text, including nested ones, and what its own ignore file lists. Global Git settings are out of scope. Where this differs from Git's view, the difference shows up as a `file.unregistered` error, never silently. A second, hand-kept list of ignored files is what this avoids.

**The certificate.** When `gate` passes, it writes the Result under `.markitect/results/`: each item's answer and pointers, the SHA-256 of every accounted file and every Definition, and a copy of the spec text it certified. The newest certificate is the default base of the next `checklist` and `impact`. "Changed" means a different hash; a Definition's old text comes from the copy, so a checklist can show what changed in it. With no certificate, the checklist is full.

**Merges.** Two branches may each certify. After they are merged, the newer certificate counts, and what only the other covered is reviewed again. That costs some work and loses nothing.

**Changes nobody reviewed.** The tool knows no "committed", only the folder. So `gate` refuses any file that changed since the base and lies in no reviewed item (row 4): the owner's own edits on the side, or a builder that touched more than its items, are named instead of certified.

**The work state.** Between `checklist` and `gate`, the hashes of the spec the checklist started from and the answers live outside the project, in the user's cache folder, keyed by the project's path. An answer binds to the hashes of the files it points at and falls away when one of them changes.

## The default driver

`/markitect-apply`, a skill with the same name as today, run in the owner's session:

1. Run `markitect checklist --start`, which begins the work and names the agent to start. If the spec has errors, stop and say so.
2. Start `markitect-builder`. It takes its task from `markitect checklist` and goes through `markitect gate`, which has it start `markitect-reviewer` as its own subagent; Claude Code allows subagents three levels deep by default. So the builder keeps its context through the repairs.
3. When it ends, run `markitect gate` and do what it printed: start the reviewer, or the builder again, which sees in its checklist what the last review found wrong.
4. When `gate` passes, commit the work and the certificate in one commit on the current branch, push nothing, and report.

Every step comes from what the tool prints, so the process stays in its output and the skill stays "do what it printed". Who starts the reviewer is a choice of the driver, not of the tool: step 3 also covers a builder that cannot start one.

`/markitect-verify` is `markitect checklist --review --start`, with `--base CERTIFICATE` or `--full`, the reviewer's answers, and `markitect gate --findings`.

Later drivers: a workflow script, which holds the loop and the limits in code; another provider.

## What we give up, and what holds instead

| What the owner wants | Today | Lean, without Git |
| --- | --- | --- |
| What is committed is complete | The tool commits only after its checks | The driver commits after `gate`. A commit made around it is visible afterwards: `status` and `gate --verify` name the files that do not match the newest certificate, and a CI check can refuse it |
| The work doesn't mix with the owner's | `apply` refuses uncommitted changes; one run per folder; a landing stops at an edit of the owner | `gate` refuses every changed file that lies in no reviewed item |
| A run has a limit | Deadline and ceiling in the tool | Two repair rounds in `gate`; time and calls in the driver |
| The branch moved during the work | The landing merges, if clean | Ordinary Git, by the driver or the owner |
| A crash in the middle | The run is stored and resumes | The answers are stored per file state; run `gate` again |
| A pull request against `main` | `impact --base main` | The base is the certificate the branch carries; Git's view is the driver's |

## Order of work

1. **Decide.** Done for the boundary, the name and Git ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)); the rest is below. The concept follows once step 3 holds.
2. **Build beside the old.** Add the folder walk, the certificate, `gate` and the file-bound answers next to today's engine, whose Git use stays until step 4. Port the tests of `submit`'s table to plain folders. Nothing is removed yet. Built on 6 November 2026, with decisions 3, 5 and 6 below as recommended, so that the owner can still change them; [the work](../../internal/run/README.md) describes it.
3. **Try it.** Switch the default driver to the lean commands and run a few changes on Makerspace, with planted defects. Compare with today's flow: items answered, planted defects caught, interventions by the owner, time. The skills were switched on 7 November 2026, as [above](#the-default-driver). The trials ran the same day on three Makerspace changes in both flows ([record](../validation/lean-trials-20261107.md)): the lean flow held, nested reviewer included, and caught the planted bug; it lacks a builder's door for a spec problem, which `stop` was and step 4 removes; decision 7 [below](#decisions-for-the-owner) settles how a builder reports one.
4. **Remove.** If it holds: first the builder's report of a spec problem, confirmed by a reviewer (decision 7), replaces `stop`; then `stop`, the run lifecycle, landing, run branches, trailers and every use of Git in the product go; docs, concept, code map; a release. Done on 7 November 2026 and released as 0.20.0: the builder reports with `gate --spec-problem` ([#995](https://github.com/Glacius-Labs/Markitect/pull/995)), and `apply`, `verify`, `submit`, `stop`, the run store, landing, run branches, the `Run-*` trailers, the full-check count and every use of Git in the product went ([#996](https://github.com/Glacius-Labs/Markitect/pull/996)). `internal/run` went from about 5,000 lines to about 2,900, tests aside and `gate`, the certificate and the report included; `rungit` went, and the folder walk of `repo` became `internal/adapters/folder`. The product's tests need no Git, so the whole Go suite takes about a minute on Windows instead of about 50. [The work](../../internal/run/README.md) describes what holds now, and [draft concept 4.0](v4-concept.md) was revised to match the same day. `tools/evaluation` still drives the concept 3.0 engine of the archive branch, unchanged by this step; whether it goes stays open ([draft 4.0 §16, question 7](v4-concept.md#16-questions-for-the-owner)).
5. **Later.** A workflow driver; a CI check with `gate --verify`.

## Decisions for the owner

1. **The boundary:** the tool computes and checks, a driver acts. Decided on 6 November 2026.
2. **Git:** none in the product; agents or people do every Git step. Decided on 6 November 2026. This also settles that `gate` makes no commit.
3. **Limits:** recommended: two repair rounds in `gate`, time and the number of calls in the driver.
4. **The name:** `gate`, decided on 6 November 2026, after weighing `certify`. `submit` was set aside because it meant landing.
5. **Where the work state lives:** recommended: the user's cache folder, keyed by the project's path, so the project holds only the spec and its certificates.
6. **How much of the spec the certificate copies:** recommended: the text of every Definition, since the spec is small and a checklist that shows what changed helps the builder and the reviewer.
7. **How a builder reports a spec problem:** the builder records the problem with the tool, and `gate` has a reviewer confirm it before it tells the owner, without counting a repair round, so no agent decides about its own work. Decided on 7 November 2026, after [finding 1](../validation/lean-trials-20261107.md#findings) of the trials. Built the same day as `gate --spec-problem "WHY"` ([the builder's report](../../internal/run/README.md#gate)).
