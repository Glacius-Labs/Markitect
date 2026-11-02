# Arms S and F: pilot dispatch — 2 November 2026

A small pilot of two simpler ways to apply a model change, against the Manager tree (M) and plain Claude Code (K). The owner, through the Topics session: "Yes lets take this one as the third arm", "No commiting, pulling, pushing and so on", "I think we should definately strive to get them to work in parallel", and on its size: "Lets make it smaller. We just want to get the idea how complex each of those approaches turns out to be and which of them is worth pursuing." It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). **These runs are not counted, and they pick an approach to pursue; they prove nothing.** The new project stays sealed.

**Status: approved by Build on 2 November 2026.** It has not started.

## Question

How complex does each approach turn out to be, and which of them is worth pursuing?

## The arms

Both are [`one_agent_arm.py`](../../tools/evaluation/one_agent_arm.py), on one graph: impact, make, gate, fresh check, land, with a bounded loop from a refusal back to make. The host holds impact, the generated prompts, the gate, starting the checker, the loop's limit, all Git and the landing. Markitect starts no container: the suite's one sandbox per launch wraps the arm, as with K. No agent commits, pushes or merges; no Manager has a branch, path rights or messages.

- **S:** make is one session with the whole checklist, each reached Definition once, and its context. It may use subagents, as K may.
- **F:** make is one prompt per working Package with its bounded context, in parallel in one folder. When they have all ended, one Root prompt integrates. A rejection goes back to Root alone.
- **The gate, after make has ended:** the model untouched, a verdict per item, `markitect check` passes, and the required checks pass on the whole repository.
- **The checker:** a fresh session the host starts, with the model, the repository at the submission, the verdicts and the diff. A rejection starts a new maker with the defects, up to 6 attempts in a row. A confirmed `model-problem` ends the Run without landing.
- **Answers** are files outside the repository, so they never land.

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

None yet.
