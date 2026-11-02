# Arms S and F: pilot dispatch — 2 November 2026

A small pilot of two simpler ways to apply a model change, against the Manager tree (M) and plain Claude Code (K): one where no agent touches Git, and one where agents work in parallel. It is kept small on purpose, to learn how complex each approach turns out to be and which of them is worth pursuing. It runs under the standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). **These runs are not counted, and they pick an approach to pursue; they prove nothing.** The new project stays sealed.

**Status: approved by Build on 2 November 2026.** It started at 19:44 UTC the same day ([Amendment 1](#amendment-1-2-november-2026-the-source-and-the-start)). Build stopped it at 21:44 UTC under the dispatch's stop rule, with S's five launches done and F's first three ([Amendment 2](#amendment-2-2-november-2026-the-reading)).

## Question

How complex does each approach turn out to be, and which of them is worth pursuing?

## The arms

Both are [`one_agent_arm.py`](../../tools/evaluation/one_agent_arm.py), on one graph: impact, make, gate, fresh check, land, with a bounded loop from a refusal back to make. The host holds impact, the generated prompts, the gate, starting the checker, the loop's limit, all Git and the landing. Markitect starts no container: the suite's one sandbox per launch wraps the arm, as with K. No agent commits, pushes or merges; no Manager has a branch, path rights or messages.

- **S:** make is one session with the whole checklist, each reached Definition once, and its context. It may use subagents, as K may.
- **F:** make is one prompt per working Package with its bounded context, in parallel in one folder. When they have all ended, one Root prompt integrates. A rejection goes back to Root alone.
- **The gate, after make has ended:** the model untouched, a verdict per item, `markitect check` passes, and the required checks pass on the whole repository.
- **The checker:** a fresh session the host starts, with the model, the repository at the submission, the verdicts and the diff. A rejection starts a new maker with the defects, up to 6 attempts in a row. A confirmed `model-problem` ends the Run without landing.
- **Answers** are files in `.arm/`, which Git excludes, so they never land.

## Dispatch

| Item | Value |
| --- | --- |
| Source | The head of branch `dev/arms-s-f-pilot`, named in Amendment 1 before the first launch. `markitect.exe` and `markitect-remote` are built from it outside the work folder, and `markitect` for Linux goes to `src/tools/evaluation/arm-bin/`, where the arm reaches it. The suite's `tools/` tree comes from it by `git archive` |
| Images | The series' sandbox, by ID: `sha256:67e2cb76…`. The Go Manager image `sha256:4f664eb8…` is named for the runner's check, and no launch here uses it |
| Cases | One repetition each, 5 per arm, S first, then F: the Library plants `library-m2-limit-contradiction` (must not land) and `library-m2-transition-gap` (must land); the Makerspace chain W0 and C1, the greenfield build and one change across areas; and the Makerspace fixed start C2 on the old start, whose layout differs from the model's, the whole-program move |
| Launches | 10, launch bound 15, relaunches 2, as the runner counts them |
| Beside them | The M and K rows of the same tasks in the [counted Windows series](counted-series-protocol-20261030.md#amendment-9-2-november-2026-the-windows-series-ended-and-linux-is-held), read only, and M's C2 r1 from the [re-measure](fixed-starts-remeasure-20261102.md). No new M or K launch |
| Model and effort | `claude:claude-sonnet-5-5`, effort high, as M and K |
| Quota | The runner's own stops: a refused request or a seven-day use of 90 %. A guess of 2 to 3 points |
| Runner | `pilot.py` in Build's notes folder `arms-pilot/`, whose SHA-256 Amendment 1 names. It runs the tools' runner, with no contrast, no review and no verify launch. Started detached (`Win32_Process`), without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables |
| Guard, lock, login | As in the series: the quiet-engine guard, `NATIVE-RUN.lock` and the refresh of the account login. No API key |
| Deadline | As in the series: 5,400 seconds per Makerspace launch, 2,700 per Library launch |
| Stop | The runner's own stops, or the owner's word. Build stops an arm that clearly fails the chain and tells the owner: W0 or C1 not landed, or a regression of W0's cases at C1. A broken arm gets what is cheap to fix, recorded as complexity, and the launch runs again under the bound. If a fix turns into real design work, Build stops and tells the owner |
| Cleanup | The runner's final check, by the suite's labels. **No** `docker image prune`, `docker system prune` or `docker rmi`. **Nothing** under `markitect-counted\` or `markitect-fixed-starts\` is written |
| The new project | Not touched. Build opens none of its material |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-arms-pilot\work`, with the runs, judgments, ledger, log, reports and `read.md` |
| Platform | Windows, natively, on Docker Desktop's engine. Build checks the engine and the process list before the start, and the owner's other Docker work stays off |
| Time | About 3 hours of launches: S about 1, F about 1.5 to 2 |

## Reading

Complexity first, then results, in one short table per arm against M and K, with Build's view of which is worth pursuing. It is an amendment to this page.

- **Complexity per arm:** the lines of host code and prompts it needed, each thing that had to be added or worked around to make it run, and what could go from today's host if it were adopted.
- **Behavior per case:** landed, hidden cases failed, module packages held, agent starts, wall time, cost, rejections, builds failing during make, and subagent use.

**Limits:**

- One repetition. In the series, the same step on the same start cost 22 agent starts and $3 in one repetition, and 160 and $31 in the other.
- M's C2 did not land in either repetition of the series, and it landed once in the re-measure with the path fix. Those are M before and after that fix, not one number.
- The chain through C1 says little about drift over time, since it holds only one change.
- The two arms cover what the model's Definitions reach. The evaluation's own hidden cases judge them; no blinded review and no `verify --full` runs here.
- A run this small cannot show what the hierarchy is for: a project too big for one agent.

## Amendments

### Amendment 1, 2 November 2026: the source and the start

- **Source:** `016fc5db302581135e3b2d382cada877f5d0dd19` on `dev/arms-s-f-pilot`. Its `tools/` tree is in the work folder's `src/`. The binaries were built from the parent merge `24bc6bd9`; the commit between them changes only Python files.
- **Runner:** `pilot.py`, SHA-256 prefix `981e40e3dd83d779`, started detached at 19:44:16 UTC, after the engine showed no container and the process list no other run. The series runner's SHA-256 is the series' own, `2f32715b…`.
- **Inputs:** the three prepared tasks that have a counterpart in the series were byte for byte the series' in a dry preparation.
- **An independent review before the first launch** found these, all fixed in that commit, and they count as complexity: answers outside the working directory might have been refused, so they moved to an excluded folder; the cost did not reach the report; a check in a folder that does not exist yet voided the launch; F's Packages could use up the integrator's time; and a review with an item missing failed whole. It left `EDITABLE` as it is: as for M, a maker cannot change the ignore page.

### Amendment 2, 2 November 2026: the reading

**Corrected the same evening.** The Topics session read the checkers' answers, and Build checked what it found against the evidence. The first version of this reading said S "did everything right" and that F's checkers disputed mostly evidence, and it proposed only narrowing the re-check. The points below now say what the answers show.

**How it ended.** F's W0 used its 6 attempts and did not land, which meets the stop rule. The runner had just started F's C1 from the same empty start, which would have rebuilt the program and not tested C1's change, so Build wrote `STOP` and killed that launch's sandbox after about a minute. Its record is no result of F. F's C2 did not run. The final check found no leftovers, no credential and no engine error. Seven-day use went from 6 % to 8 %.

**Complexity**

| | S | F | M, today's host |
|---|---|---|---|
| Code | About 530 lines of Python, prompts included (about 70), on helpers of K's runner and M's arm | About 40 lines more | About 14,700 lines of Go without tests, outside the model language |
| Containers it starts | None | None | One per agent start, and a remote per Run |
| Agent Git | None | None | Branch per Manager, hooks, merges |
| Fixed before launch | 5 review findings | The same | |
| Fixed during launches | None | None | |

Worked around to make it run: a process group per session, since K's stop would end the sessions beside it; answers in an excluded folder, since a session might not write outside its working directory; required checks in a clean checkout, since `go build` leaves a binary behind; and a small reader for `execution.yaml`, since no command prints it. If S were adopted, Build estimates that about two thirds of that Go could go (the remote and its hooks, per-agent containers, the agent daemon, most of the Manager schedule), and that a product version needs about 1,000 to 1,500 lines back. That is an estimate from reading the code, not a count. The arm also throws away a stopped Run's work, where M keeps it on a Run branch, so a product version must add that.

**Behavior, one repetition each.** M and K are the series' rows for the same tasks, and Failed counts hidden cases. A dash means it did not run.

| Case | S | F | M | K |
|---|---|---|---|---|
| Library plant, contradiction: must stop | Stopped, 2 min, $0.39, 3 agents | Stopped, 3 min, $0.69, 6 agents | Stopped, 1 min (r1), 5 min and $0.62 (r2) | Stopped, under 1 min, $0.07 |
| Library plant, gap: must land | Landed, 0 of 19 failed, 1 min, $0.30, 2 agents | Landed, 0 of 19, 2 min, $0.59, 4 agents | Landed, 0 of 19, 4 min, $0.70, 10 agents | Landed, 0 of 19, under 1 min, $0.15 |
| Makerspace W0 | Landed, 0 of 54, 9 packages, 27 min, $7.21, 2 agents | **Not landed**, 6 rejections, 56 min, $16.24, 21 agents | Landed, 0 of 54, 9 packages, 26 min, $9.41, 28 agents | Landed, 0 of 54, 9 packages, 12 min, $2.08 |
| Makerspace C1 | Landed, 0 of 55, 2.5 min, $0.58, 2 agents | — | Landed, 0 of 55, 8 min, $3.24, 24 agents | Landed, 0 of 55, 2 min, $0.56 |
| C2, layout differs from the model | Landed, 0 of 57, 9 packages, 14 min, $3.27, 2 agents | — | Series: not landed, 21 of 57, 31 min, $24.55, 101 agents. With the path fix: landed, 0 of 57, 9 packages, 54 min, $16.37, 68 agents | Landed, 0 of 57, 9 packages, 8 min, $1.49 |

Builds and tests failing during make: S 2 of 75, F 0 of 53. Subagent calls: S 5, all on W0; F 0. Checker rejections: S 0, F 7; the gate refused S once, for a stray binary.

Go delivered on W0, as code lines, test lines and test functions: S 2,465, 6,942 and 228; K 2,471, 1,528 and 38; M 4,163, 3,834 and 112. The hidden cases do not see tests. That they explain much of S's cost over K is an inference, not measured.

**What it shows:**

- **S got every hidden case right, and more simply than M.** It matched M's results on every case here, at about 0.2 to 0.8 of its cost. That holds for the hidden cases and its own checker only, as the next points show.
- **F's fan-out worked. Its checker loop did not.** On W0 the nine Packages ran side by side with no failed build, the longest for 11 minutes, and Root's integration took 1 minute. That is a little over half of S's 21-minute make, at a cost of $5.00 against $6.04. Then six fresh checkers cost $9.17 and disputed 1 to 3 of 108 items each, 11 in all, while the build and tests passed. Since a stopped Run delivers nothing, F's 54 failed cases are the empty start, and its code has no hidden score.
- **No F checker named wrong behavior.** 8 of the 11 disputes are evidence or test gaps, and in 6 of those the checker itself says the code looks right or probably right. The other 3 are one rule read two ways: the `Interface` Kind says other modules use nothing else of a module, and the maker was told breach or an open model (round 1), no model problem and fine (round 2), then breach again (round 4). Rounds 3, 5 and 6 did not dispute it. 4 of the 8 gaps are behavior across modules, which no Package prompt owns, so that part is F's own.
- **S's clean pass is partly one reader's ruling.** S's code has the same kind of exports, such as `billing.InvoicesOfMonth` and `members.Members`. Its checker saw them and let them through: "the model gives no other way to get them, so I do not hold it against the realization." F's round 4 checker held the opposite, that the reports could be built from what the interfaces list. The interface pages name the records other modules may use and not always how to get them, so the model may be unclear here. A checker settled that on its own authority, where a model defect should escalate ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)), and the arm's result records no finding of it.
- **That loop is S's too.** S's checker approved W0 the first time, on one repetition. A fresh checker that re-reads everything each round can keep finding something, so S may meet the same stop on another launch.
- **The checker's worth stays open.** It caught nothing in S's five launches, and on F's W0 it named no wrong behavior. On F's contradiction plant it rightly refused model-problem verdicts on items the contradiction did not reach.
- **K had the same hidden results on all five, at about 0.2 to 1.0 of S's cost.** It was about level on C1 and under half on the rest. This pilot does not show that S is better than K. What S adds is a verdict with evidence for each reached Definition, a current registry and a second reader.

**Build's view:** S is worth pursuing, and F is a small option to keep for a project too large for one session, which this pilot cannot test. If F stays, its Root prompt should own the behavior across modules and its tests.

Before more launches, fix the loop, and not by narrowing it alone. On paper, re-checking only the disputed items and what changed lands F at round 3. So does the contract Topics proposes to the owner: a checker answers `wrong` or `not-shown`, only `wrong` refuses, and `not-shown` lands as a recorded finding. Both land there because round 3's checker accepted a rule that round 4's refused, so neither settles a rule read two ways. Build would do both, and have whatever a checker waives land as a recorded finding too, so that a person sees the ruling. That changes the checker's answer and not only its prompt, and nothing has tested it on the plants. Concept 2.0 stays as it is until the owner decides.

### Amendment 3, 2 November 2026: the checker change and a second set of launches

The owner approved that change and a new W0 for both arms on 2 November 2026. These launches are as descriptive as the first ones, and they are not counted.

**The change** in [`one_agent_arm.py`](../../tools/evaluation/one_agent_arm.py), about 40 lines more:

- **The answers.** A checker answers each verdict `confirmed`, `wrong`, `not-shown` or `waived`. Only `wrong` refuses. `not-shown` and `waived` land, and the Result lists them under `findings`, so that a person sees them.
- **The rounds.** A ruling stands for the Run. A later round re-checks what was refused and the items whose verdict the maker changed since, sees the diff since the last review, and reopens another item only if that diff breaks it.
- **A model problem.** A `model-problem` verdict the checker confirms ends the Run at once, so that no later maker can take it back. Any other answer refuses it, since nothing was made for that item. A checker is told never to waive a contradiction in the model.

An independent review before the first launch found four defects, all fixed before the source below, and they count as complexity: a confirmed model problem could be taken back and the contradiction land; S's later makers were told to answer every item again, which undid the narrowing; a gate refusal dropped the checker's reason from the next maker's prompt; and the prompt misstated what refuses a `model-problem` verdict.

| Item | Value |
| --- | --- |
| Source | The head of branch `dev/arms-s-f-pilot`, named below before the first launch. Only Python has changed since the first set, so the programs are that set's copies |
| Images | As in the first set |
| Cases | Both arms: the two Library plants and the Makerspace chain W0 and C1. F also: the fixed start C2, which it never reached. S's C2 does not run again, since its checker refused nothing there and the new one can refuse only less |
| Launches | 9, launch bound 13, relaunches 2, as the runner counts them |
| Beside them | The first set's rows, the series' M and K rows and M's C2 from the [re-measure](fixed-starts-remeasure-20261102.md), read only. No new M or K launch |
| Model and effort | As in the first set |
| Quota | The runner's own stops. A guess of 2 to 3 points |
| Runner | `pilot.py` in Build's notes folder `arms-pilot-2/`, whose SHA-256 is named below. Started detached, without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables |
| Guard, lock, login | As in the first set. No API key |
| Deadline | As in the first set |
| Stop | The runner's own stops, or the owner's word. Build stops at once if the contradiction plant lands or the gap plant does not, since the change would then be unsafe. Build stops an arm if W0 or C1 does not land, or if W0's cases regress at C1, and tells the owner |
| Cleanup | The runner's final check, by the suite's labels. **No** `docker image prune`, `docker system prune` or `docker rmi`. **Nothing** under `markitect-counted\`, `markitect-fixed-starts\` or `markitect-arms-pilot\` is written |
| The new project | Not touched. Build opens none of its material |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-arms-pilot-2\work` |
| Platform | Windows, natively, on Docker Desktop's engine. Build checks the engine and the process list before the start |
| Time | About 2 hours |

**The reading** adds, per launch, the findings it landed and whether any of them shows in the hidden cases. That is the price of this change: what it lets through.
