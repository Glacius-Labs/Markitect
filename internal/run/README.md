# The process of a Run

`internal/run` holds the process of a Run in the tool, per [draft concept 4.0](../../docs/concepts/v4-concept.md). `markitect apply` and `markitect verify` start a background session and end. The session only calls `markitect`, and `markitect submit` is the one door: it says what is next each time and decides whether the work counts. Nothing here drives a session, so forgetting a step can't land and compaction can't lose the process.

| Package | Role |
| --- | --- |
| `internal/run` | The process: `apply`, `verify`, the checklist, the answers, `submit`, the hooks, the Result and landing. It starts no process itself; its ports do. |
| `internal/adapters/rungit` | Git work in the user's own repository ([below](#git)). |
| `internal/adapters/hostcheck` | The required checks on the user's machine ([below](#required-checks)). |
| `internal/adapters/claude` | The provider's background sessions ([below](#the-session)). |
| `internal/bootstrap` | Builds the engine for the command line. |

## A Run

1. **`apply`** refuses a detached HEAD and a change outside `.markitect/` that is not committed, since it would be silently left out. It compiles the spec, and an error in the spec stops it with no agent. It commits the spec on the owner's branch and works out the base and the list ([impact and checklists](../../docs/guides/model-language.md#impact-and-checklists)). Nothing affected ends it.
2. **The session** starts in a worktree of its own, on that commit, and `apply` returns. The owner keeps modeling. An open Run on the branch means a new `apply` starts nothing: one that is working is left alone, and one whose session has ended is resumed in its own worktree, with its repair count and a new deadline. `verify` does the same with an open verify. What a session calls works only in a Run's worktree.
3. **`checklist`**, on its first call, tries the required checks on the start. A check that cannot start ends the Run before the work is spent. One that fails is told to the implementer and the Result as failing before the work. It prints the task again each time, so compaction can't lose it.
4. **`submit`** is below. It tells the implementer when to start the `markitect-reviewer` agent, which answers with `markitect answer`.

`markitect stop WHY` ends a Run without landing, and waits for a `submit` that is running, which may be landing. From the session it records a question for the owner, or with `--environment` says the machine lacks something; from the owner it ends the session first. Every other command tells the owner in one line about a Run that closed or stopped working (`Notice`).

The next `apply` removes the worktrees of closed Runs. The provider locks a worktree while its session runs, so `submit` cannot.

## The one door

`submit` looks at the current state and does the first thing that applies:

| | It finds | It does | It tells the agent |
|---|---|---|---|
| 1 | Anything under `.markitect/` changed except the registry pages, anything in the harness, or a link that leaves the repository | Refuses | Put these back, or `markitect stop` |
| 2 | A reviewed state, and the only difference from it is new files not listed | Refuses | List them if they are the work, otherwise remove them |
| 3 | An error from `check`, in the spec or the realization | Refuses | The errors |
| 4 | A required check that cannot start | Closes the Run as `incomplete` | Stop |
| 5 | A required check that fails | Refuses | The check and its output |
| 6 | No complete review for this state | Commits the state on the Run's branch | Start the reviewer, with the items |
| 7 | A `spec problem` | Closes the Run as `spec-problem` | The Result holds the question |
| 8 | A `wrong` answer | Counts one repair round per reviewed state; the third closes the Run as `refused` | The items and what is wrong |
| 9 | Nothing left | Writes the Result, commits and brings the branch up | Landed, or kept and why |

A review that finished on an earlier state still counts when the implementer repaired before calling `submit` again. A repair review asks only about what has no answer, is open, or lies in a file the new diff touched; other answers stand.

**Limits.** At most two repair rounds, 30 calls of `submit` and, by default, 6 hours. A second `submit` during a slow one answers "still running" and does not count. The lock names its process, so a `submit` that was killed can be run again.

## The answers

A reviewer answers one line per item on standard input: `KEY: ANSWER POINTERS -- EXPLANATION`. The words are `ok`, `wrong`, `not shown` and `spec problem`; all but `ok` need an explanation after `--`, and `ok` needs a pointer. `answer` refuses what it cannot accept at once and stores nothing from that call: a line it cannot read, an item not on the list, a location or test that does not exist in the commit under review. It says what is still missing. How much a reviewer really read is not measured, and the Result says so.

## The hooks

They are in the settings a session starts with, so a modeling session gets none of them. `markitect hook` does nothing unless the Run is open.

| Hook | Does |
|---|---|
| `Stop` | Sends the session back while the Run is open and no reviewer or `submit` is running. After five blocks since the last `submit` it closes the Run and lets the session end |
| `StopFailure` | Records the cause when a turn ends on an API error, such as a usage limit |
| `PreToolUse` | Past the deadline it stops the session and closes the Run. On the shell it refuses `markitect answer` unless the agent is the reviewer |

## Landing

`submit` builds what will land off to the side: the work if the owner's branch has not moved, otherwise a merge, which must be clean and pass `check --spec` and the required checks. The landing commit's `Run-Work` trailer names the work, which is the next Run's base, so later commits stay in the next `impact`. A Result that did not land says so in its trailers, so merging such a branch by hand does not make it realized work. A branch the owner moved back keeps the Run from landing. It then moves the branch, only if it is still where it was: `merge --ff-only` where it is checked out, so uncommitted edits stay, or a ref update where it is not.

If that fails after three tries, the Run closes as `kept` on `markitect/runs/<run>`, and the next `apply` tries once more. If that fails too it starts fresh. The base has not moved, so nothing is lost. Nothing is ever pushed.

## Git

`internal/adapters/rungit` works in the repository's common Git directory, which every worktree shares. `Candidate` makes a commit of every file Git would take through a temporary index, so a worktree, its index and HEAD stay as they are. The tool's commits run no hooks of the project. Only the spec commit in `apply` and what `init` writes use the user's identity and hooks. Run state lives in `<git dir>/markitect/runs/`.

## Required checks

`internal/adapters/hostcheck` runs each check with `sh -c` in a clean export of the state, stopping at the first failure, with the variables of a session and of Git taken out. A pass is kept per state; a failure runs again. Exit 126 or 127 counts as "could not start", the environment's failure, only when the shell does not know the command's first word and it is no path into the repository; a program a script calls is the work's, and so is a folder the work has not made yet. A check that exits and leaves a child holding its output has passed.

## The session

`internal/adapters/claude` starts `claude --bg --worktree <run>` on the user's stored login. The session runs in `dontAsk`, with file edits and any shell command allowed and a deny rule on `git push` and `gh`. Its settings are generated per Run under the run state: `worktree.baseRef` set to `head`, auto memory off, the user's own `CLAUDE.md` excluded, subagents forced onto the session's model and a long shell time limit. Only a session the provider reports as working counts as working. The model and effort come from `execution.yaml`, with `sonnet` and `high` as the defaults.

## The Result

The tool writes the Result from what happened, never an agent. A landing's Result is part of its commit, under `.markitect/results/`. Any other stays on `markitect/runs/<run>`, with `verify`'s Findings page next to it. A landing does not delete a verify's branch. **The full-check count** says how many changes have landed since the last full check; from 10 on, the Result says one is due.

## Tests

`run_test.go` runs whole Runs over real Git, with a provider that makes the worktree and a test that plays the agent: every row of `submit`, repair rounds, landing on a moved branch, conflicts, uncommitted edits, the hooks, the limits and `verify`.
