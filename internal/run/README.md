# The process of a Run

`internal/run` holds the process of a Run in the tool, per [draft concept 4.0](../../docs/concepts/v4-concept.md). `markitect apply` and `markitect verify` open a Run and say which agent to start. The owner's own Claude Code session starts it, as a subagent in the owner's folder. The agents only call `markitect`, and `markitect submit` is the one door: it says what is next each time and decides whether the work counts. Nothing here starts or drives an agent, so forgetting a step can't land and compaction can't lose the process.

| Package | Role |
| --- | --- |
| `internal/run` | The process: `apply`, `verify`, the checklist, the answers, `submit`, the hook on `answer`, the Result and landing. |
| `internal/adapters/rungit` | Git work in the user's own repository ([below](#git)). |
| `internal/bootstrap` | Builds the engine for the command line. |

## A Run

1. **`apply`** refuses a detached HEAD and a change outside `.markitect/` that is not committed, since it would be silently left out. It compiles the spec, and an error in the spec stops it with no agent. It commits the spec on the owner's branch and works out the base and the list ([impact and checklists](../../docs/guides/model-language.md#impact-and-checklists)). Nothing affected ends it.
2. **The builder** is the next step `apply` prints: start the `markitect-builder` agent with "Build run NAME". The apply skill has the owner's session do that. The builder works in the owner's folder, with fresh context. When it ends, the owner's session runs `markitect status`, and while the Run is open it starts the builder again. One Run at a time is open per folder; a second `apply` or `verify` there says how to go on with it. What the agents call finds the open Run of their folder.
3. **`checklist`** prints the task and the list each time, so compaction can't lose them.
4. **`submit`** is below. It tells the builder when to start the `markitect-reviewer` agent, a subagent of its own, which answers with `markitect answer`.

`markitect stop WHY` ends a Run without landing, and waits for a `submit` that is running, which may be landing. The Result holds why as a question for the owner; with `--environment` it says the machine lacks something. The work goes on `markitect/runs/<run>` and also stays in the folder, uncommitted, for the owner to keep or drop. Every other command tells the owner in one line about a Run that closed (`Notice`).

## The one door

`submit` looks at the current state and does the first thing that applies:

| | It finds | It does | It tells the agent |
|---|---|---|---|
| 0 | The folder is no longer on the Run's branch at its start: a commit made during the Run, or another branch checked out | Refuses; a landing that moved the branch but was not recorded is recorded | Undo the commits with `git reset --soft`, or check the branch out again |
| 1 | Anything under `.markitect/` changed except the registry pages, anything in the harness, including `AGENTS.md` and `CLAUDE.md`, or a link that leaves the repository | Refuses | Put these back, or `markitect stop` |
| 2 | A reviewed state, and the only difference from it is new files not listed | Refuses | List them if they are the work, otherwise remove them |
| 3 | An error from `check`, in the spec or the realization | Refuses | The errors |
| 4 | No complete review for this state | Makes a commit of the folder, on no branch | Start the reviewer, with the items |
| 5 | A `spec problem` | Closes the Run as `spec-problem` | The Result holds the question |
| 6 | A `wrong` answer | Counts one repair round per reviewed state; the third closes the Run as `refused` | The items and what is wrong |
| 7 | Nothing left | Writes the Result, commits and brings the branch up | Landed, or kept and why |

A review that finished on an earlier state still counts when the implementer repaired before calling `submit` again. A repair review asks only about what has no answer, is open, or lies in a file the new diff touched; other answers stand.

**Limits.** At most two repair rounds, 30 calls of `submit` and, by default, 6 hours. A second `submit` during a slow one answers "still running" and does not count. The lock names its process, so a `submit` that was killed can be run again.

## The answers

A reviewer answers one line per item on standard input: `KEY: ANSWER POINTERS -- EXPLANATION`. The words are `ok`, `wrong`, `not shown` and `spec problem`; all but `ok` need an explanation after `--`, and `ok` needs a pointer. `answer` refuses what it cannot accept at once and stores nothing from that call: a line it cannot read, an item not on the list, a location or test that does not exist in the commit under review. It says what is still missing. How much a reviewer really read is not measured, and the Result says so.

## The hook

One hook, in the project's `.claude/settings.json`, which `init` writes: before a shell command that runs `markitect answer`, `markitect hook pretool` refuses it unless the agent is the reviewer, so no agent marks its own work as done. Hooks in project settings run inside subagents too, and their input names the agent. A Run may not change the file.

## Landing

`submit` builds what will land off to the side: the work if the owner's branch has not moved, otherwise a merge, which must be clean and pass `check --spec`. The landing commit's `Run-Work` trailer names the work, which is the next Run's base, so later commits stay in the next `impact`. A Result that did not land says so in its trailers, so merging such a branch by hand does not make it realized work. A branch the owner moved back keeps the Run from landing. It then moves the branch, only if it is still where it was. Where it is checked out, the files follow as in a fast-forward, a file that already holds the landing's version, such as the work the builder wrote in that folder, stays as it is, and a file with an edit of its own stops the move. The files change first, the branch moves after them and the index last, so a landing cut off part way runs again, and one cut off after the branch moved only brings the index up; one whose branch cannot move puts the files back. Every attempt of one landing writes the same Result page. A commit on the branch counts as the Run's landing only with a passed review, the reviewed work as its parent and the Run's Result page. Where it is not checked out, only the ref moves.

If that fails after three tries, the Run closes as `kept` on `markitect/runs/<run>`, and the next `apply` tries once more, before it looks for uncommitted changes, since the work is still in the folder. A retry stopped while the folder holds other changes does not count. If that fails too it starts fresh. The base has not moved, so nothing is lost. Nothing is ever pushed.

## Git

`internal/adapters/rungit` works in the repository's common Git directory. `Candidate` makes a commit of every file Git would take in the folder through a temporary index, so the folder, its index and HEAD stay as they are until landing. The tool's commits run no hooks of the project. Only the spec commit in `apply` and what `init` writes use the user's identity and hooks. Run state lives in `<git dir>/markitect/runs/`.

## The agents

Both are agent files `init` writes. The builder may edit files and start agents; the reviewer has no edit or agent tools. Each defaults to `sonnet` with effort `high`, and `apply` names the manifest's `model` when it says which agent to start. Everything else comes from the owner's session: its permission mode, settings, plugins and memory.

## The Result

The tool writes the Result from what happened, never an agent. A landing's Result is part of its commit, under `.markitect/results/`. Any other stays on `markitect/runs/<run>`, with `verify`'s Findings page next to it. A landing does not delete a verify's branch. **The full-check count** says how many changes have landed since the last full check; from 10 on, the Result says one is due.

## Tests

`run_test.go` runs whole Runs over real Git, with the test playing the agents: every row of `submit`, repair rounds, landing over the work in the folder, one Run per folder, the hook on `answer`, the limits and `verify`.
