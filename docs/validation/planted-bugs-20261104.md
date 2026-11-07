# Test 1: planted bugs on the 5× project, 4 November 2026

This is test 1 of [concept 3.0 §13](../concepts/v3-concept.md#13-how-we-know-it-works), under step U4 of the [delivery plan](../implementation-plan.md#concept-30-build). It puts known bugs into a finished 5× program and has reviewers check it with no word about them. No result is recorded yet.

## Questions

- **Q1:** does a reviewer still catch known bugs at this size?
- **Q2:** is one reviewer per Package better than one overall?
- **Q3:** does a reviewer without a shell catch as many as one with a shell?
- **Q4:** how often does a check raise a false alarm on the same program without the bugs?

The answers bear on how `--full` works and on whether reviewers keep their shell. If no setting catches the bugs, reviewing is rethought. Which way each decision goes stays with the owner, on the page these launches end in.

## What is checked

The program is the default's own [first build at 5×](default-arm-20261104.md), kept as `large-w0-default-sonnet`, which passes all 327 hidden cases.

**The bugs.** A fresh author planted 20 bugs, with no part in the program or in this test's reviews: 6 in one module's own rules, 6 that cross modules, 4 in the order of refusals, 2 where an effect goes missing and 2 small ones, across 18 modules. They were written from the spec alone, and the list stays outside the repository until the launches end. It is `bugs.json`, SHA-256 `235fac088c8c6a78f1542d88b011fe15508f77fd89144ab70664425e37a070dd`.

| Each bug must | Shown by |
|---|---|
| Contradict the spec | The author names the page and the rule |
| Survive the program's own checks | With all 20 in, it builds, its tests pass and `markitect check` reports no errors. No test was changed |
| Be real by the hidden cases | With all 20 in, 27 of the 327 hidden cases fail. Each bug alone, on a fresh copy of the clean program, fails the cases its author named, and its tests still pass. Build checked this itself |

**What this set is.** The program's tests stopped about 92 % of some 4,800 automatic changes and almost every hand-written one that touched an explicit rule, boundary or example. About 65 automatic changes got past those tests and still failed a hidden case, and the author rewrote 20 of them as plausible mistakes. So these bugs come from what the tests don't reach, and they say nothing about bugs the tests would have stopped. None sits in `training` or `purchasing`, where nothing got through.

**Two submissions, alike.** Each is a new repository of two commits with the same names, dates and messages: the owner's model, then one commit of the work. They differ only in those 18 files, so neither a reviewer's history nor a stray object points at a bug. The earlier landing's Result is out of both, since it would tell a reviewer what an earlier review answered.

| Submission | Commit |
|---|---|
| `clean` | `6fa49cc31a4dbd2221e851b6b1ee09572bf963c7` |
| `planted` | `fe277033f2c886cb2f3050358e8c2240fd95cf40` |

## Launches

Twelve checks, each one `markitect verify --full` on a fresh container: four settings, each on `planted` twice and on `clean` once.

| Setting | Reviewers | Shell |
|---|---|---|
| Per Package, with a shell | One per Package, 8 at a time, as `--full` works today | Yes |
| One reviewer, with a shell | One for the whole list, through `--one-reviewer` | Yes |
| Per Package, no shell | As the first | No, through `MARKITECT_REVIEWER_SHELL=off` |
| One reviewer, no shell | As the second | No |

The order goes round the four settings on `planted`, then on `clean`, then on `planted` again, in the table's order. The runner stops after the first launch, so that someone sees it did what this page says before the rest start.

Without a shell a reviewer can't run a test or the program. It still sees whether the required checks passed, which the Command ran. The Command also holds it to more: with no command in its transcript, every file it points at must have been opened with `Read`, or it is asked once to correct that. This is part of what the setting does, not something to separate from it. Neither setting gets a diff: `verify` has no earlier commit to compare with.

## How the answers are read

This rule is fixed before the first review.

For each planted launch and each of the 20 bugs, one word:

| Word | Meaning |
|---|---|
| **Caught** | An item is answered `wrong`, and its explanation and pointer name this bug: what the code does here, and that the spec says otherwise. The words need not match the list, and the item need not be the one the bug was planted under |
| **Noticed** | No `wrong` names it, but a `not shown` or `spec problem` answer or a reviewer's closing message does. A `not shown` would have let it land |
| **Refused for something else** | The item where the bug lives is `wrong`, for a reason that isn't this bug. In an `apply` it would go back, but the reviewer did not see the bug |
| **Missed** | Anything else, including `ok` on that item |
| **Not answered** | The item has no accepted answer, because the launch ended without one. The reviewer's last written answers are recovered from its transcript and read by the same rule, shown apart from the accepted ones. A setting can't catch a bug this way in the totals, since the Command would not have accepted its answers |

One `wrong` that names two bugs catches both, and two `wrong` answers that name one bug count once. Two pairs of bugs share an item (`access/admit-guest` and `ledger/postings`), so a `wrong` naming one of a pair leaves the other at **Refused for something else** unless it names that too.

For every launch, each other `wrong` is read against the spec and the code: **real** if the code departs from the spec, **false alarm** if it does not. A real one on `clean` is a true defect of the default's build that the hidden cases don't hold, and it is no mark against the reviewer. `spec problem` answers are counted apart.

**Who reads.** A fresh reader takes the bug list, this rule and every launch's answers other than `ok`, under neutral labels, without being told the setting or which launches are `clean`. Any `caught` on a `clean` launch shows the reading went wrong, since those launches can't hold the bugs. The reviewers' closing messages come in a second step, after that reading is written, and can only add **Noticed**: they tend to show the setting, since a reviewer without a shell says what it could not run, and an answer can hint at it too. Build then checks each `caught` and `noticed` by hand against the code, and any disagreement goes on the page with the reason.

**What goes on the page.** Bug by launch, in one table. Per setting: how many of the 40 chances were caught, by kind of bug; false alarms and real findings on `clean`; time, cost and how much checking the transcripts show. There is no overall score. Two launches per setting can't give a rate, and the same 20 bugs recur, so only a wide gap says much.

## Dispatch

| Item | Value |
|---|---|
| Source | Branch `dev/u4-test1` at the commit that adds this page, with `tools` frozen by `git archive` into the evidence folder and `markitect` built from it |
| Image | `markitect-runtime-go:u1-dev`, `sha256:4f4da6c2687a65f81702b7e0349ef8dec990ea2bcaab6efcc139225cdb8a42bc`: `packaging/runtime` and its Go example, with Claude Code 2.1.308 and Go 1.27.1 |
| Sandbox image, for the suite's gateway | `sha256:67e2cb76d76c27fd20b183e8d209e6407c49f758c35a454168b5b37cc2e24ce8` |
| Provider | `claude-sonnet-5-5` at effort `high`, the session's subagents forced onto it, and the host refuses any other model |
| Login | The owner's token, in `MARKITECT_CLAUDE_TOKEN` in the runner's own environment only, read from the user's registry by the launcher and never printed or written. Every other `ANTHROPIC_*` and `CLAUDE_*` variable is cleared for the host |
| Limits | The Command's container: 4 CPUs, 8 GiB, 2,048 processes, 1 GiB of temporary space. A session's limit is 3 hours, and a launch's deadline 14,400 seconds. A launch that runs out is recorded as it ended, since that is what the setting did at this size |
| Launch bound | 16 launches. A void launch goes again once; twelve are planned |
| Expected use | About 20 times K's W0 on Sonnet, and about 12 points on the token's weekly reading. In the default's W0 the per-Package reviewers cost about 1.6 K, and that launch took 2 points for 3.4 K. A single reviewer's cost is not known |
| Abort | A login failure, the token found in the evidence, an escape, or the weekly rise past 30 points, at which the runner is stopped by hand. The runner's own stop stands at a weekly reading of 90 |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, taken by the suite. Other Docker work is checked first and stays off |
| Cleanup | The Command removes its container and volumes and the suite its network and gateway; afterwards nothing labelled with the evaluation may remain |
| Evidence | Outside the repository, in `%LOCALAPPDATA%/markitect-test1/`: the runner's work folder, the submissions and the bug list. Each launch that counts goes into the [lasting store](https://github.com/Glacius-Labs/Markitect/blob/2fd2d33747de2e69e5f4968432792d543003041a/tools/evaluation/kept.py) with its Result |

**What this can't show.** It uses one program and one set of bugs, and the bugs come from the blind spots of that program's own tests. A real implementer's mistakes may look different. It checks a whole program at rest, not a change, so test 2 is where reviewing a change gets its evidence.
