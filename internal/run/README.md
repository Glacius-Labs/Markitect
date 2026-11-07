# The process of a Run

`internal/run` holds the process of a Run in the tool, per [draft concept 4.0](../../docs/concepts/v4-concept.md). `markitect apply` and `markitect verify` open a Run and say which agent to start. The owner's own Claude Code session starts it, as a subagent in the owner's folder. The agents only call `markitect`, and `markitect submit` is the one door: it says what is next each time and decides whether the work counts. Nothing here starts or drives an agent, so forgetting a step can't land and compaction can't lose the process.

Beside the Run, and to be tried before anything is removed, `markitect gate` checks work done without one and without Git ([DEC-096](../../docs/concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](../../docs/concepts/register.md#dec-097-markitect-uses-no-git), [plan](../../docs/concepts/lean-tool-plan.md)): [below](#without-a-run-gate).

| Package | Role |
| --- | --- |
| `internal/run` | The process: `apply`, `verify`, the checklist, the answers, `submit`, the hook on `answer` and `checklist`, the Result and landing; and without a Run, `gate`, the certificate and the answers bound to files. |
| `internal/adapters/rungit` | Git work in the user's own repository ([below](#git)). |
| `internal/adapters/repo` | Without a Run: the walk of the project folder and its `.gitignore` files (`Folder`). |
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

## Without a Run: gate

The tool computes and checks the project folder, and whoever drives the agents (a skill, a workflow or a person) does the rest, every Git step included ([DEC-096](../../docs/concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](../../docs/concepts/register.md#dec-097-markitect-uses-no-git)). These commands are built beside the Run and change nothing of it. While a Run is open in the folder, `checklist` and `answer` are the Run's, as above, and `gate` refuses; that look into the Run store is the one thing here that still uses Git, and it goes with the Run. Outside a Git repository the commands just work.

**The folder.** The project folder is the nearest folder with `.markitect/manifest.yaml`. Its files come from walking it, without what the project's `.gitignore` files exclude, read as plain text, nested ones included ([`repo.Folder`](../adapters/repo/folder.go)). Patterns follow Git: `!` brings a file back unless its folder is excluded, a trailing `/**` holds what is inside a folder and not the folder, and `[!x]` means "not x". Git's own folder is not walked. Another repository, a folder whose `.git` is a folder with a `HEAD` or a file naming one, is one entry, as in Git, so that `check` names it until it is registered or ignored; its files are not walked, and a change inside it is not seen. A symbolic link counts as where it points, as Git stores it. Global Git settings are out of scope, and so are files Git tracks although a `.gitignore` excludes them: the walk does not see those, and a registry row for one is an error.

**The certificate.** When `gate` passes, it writes `gate-<time>-<state>.json` under `.markitect/results/`: the SHA-256 of every accounted file and every Definition, the text of the files under `.markitect/` the spec and its registry are read from (not the results, not the installed tool, no binary file), and each item's answer. Line endings do not count in a text file's hash, so a checkout on another system matches. The Result page `gate-<time>-<state>.md` lies beside it. The newest certificate is the base of the next change: one that another names as its base is older than that one, whatever the clocks said, and of the rest, as after a merge of two certified branches, the one made last counts, by the time it records, never by file times; "changed" means a different hash, and a Definition's old text comes from the copy, so `checklist` shows what a changed or removed Definition said before. With no certificate the list holds every Definition, Findings or not. After a merge the rest of what the other branch covered is reviewed again. The results folder counts in no certificate's hashes: it holds the records. A certificate is a record, not a proof: during the work a changed certificate stops `gate`, but one planted by hand while no work is under way is taken as it is.

**The work** lives outside the project, in the user's cache folder (`markitect/work/`, one file per project folder). `checklist` begins it and holds the hashes it began with of what the work may not change: the spec, the harness, the `.gitignore` files and the certificates. Without `--base`, `--full` or `--findings`, `checklist` goes on with the work under way, and begins the change from the newest certificate when there is none or a newer certificate ended it; when what the work may not change changed under it, it refuses instead of beginning again, so that no builder can change the spec and start over. With those flags it names a change, and begins it afresh unless it is the one under way: that is how the owner begins the work again after changing the spec. `--start`, the skills' call, is the owner's step too: when what the work may not change changed, it begins the work again from the newest certificate and says so. The hook stops a subagent from calling `checklist` with any of these flags, so an agent that changed the spec cannot start over with its change as the owner's; it gets `checklist` as it is, or with `--review`. `--base` takes a certificate, its `.json` or its `.md`, as a path in the project or a name in the results folder. A base certificate gone from the folder, as after a change of branch, is a changed certificate like any other. The refusal names no command that begins the work again, since an agent reads it too. `checklist --review` and `answer` without work under way say there is nothing to do and to end, so a reviewer of an ended Run is not turned into a builder. The list grows as a Run's does: an item stays on it once a state put it there.

**The answers** are the same lines as in a Run. Each pointer must be a file the walk finds. An answer is bound to the hashes of the files its item names and the files it points to, and falls away when one of them changes; put back, it stands again. A `wrong` answer stands only on the state it was given on: the work went back for it. When every item has an answer, that state of the folder is a finished review.

**`gate`** does the first thing that applies, and each refusal prints the one next step:

| | It finds | It says, and exits |
|---|---|---|
| 1 | The spec (everything under `.markitect/` except the registry pages and the results), the harness, a `.gitignore` file or a certificate changed since `checklist` began the work, or a link leaves the folder | Put them back; a change to the spec is its own step, before the work, and the owner begins the work again by naming the change. Exit 1 |
| 2 | A file that changed since the base certificate and lies in no item of the list | The files: list them on the registry so that they are reviewed, or put them back. Exit 1 |
| 3 | An error from `check`, in the spec or the realization | The errors. Exit 1 |
| 4 | Items without an answer for the files as they are | Start the reviewer, which takes its task from `checklist --review`. Exit 1 |
| 5 | A finished review with a `spec problem` | The question for the owner; the Result says so. Exit 1 |
| 6 | A `wrong` answer | The items, and which repair round of two this is. After two rounds it refuses, and the Result says so. Exit 1 |
| 7 | Nothing left | Writes the certificate and the Result under `.markitect/results/`, ends the work and says to commit the work with both in one commit. Exit 0 |

Before row 1: without work under way, a folder that matches its newest certificate has nothing to check (exit 0), and any other has no work to check (exit 1). With work under way, an empty list on a folder that matches the base certificate has nothing to check either; work on Findings has a list. Rows 5 and 6 read every finished review of the work, not only this state's, so a spec problem, or a third review that sends something back, is said before row 4 asks for another review, as in `submit`. A repair round is a finished review that found something wrong: calling `gate` again on the same state uses none, and a review with a spec problem uses none either. `gate` keeps only these two rounds, a rule about state; time and the number of calls are the driver's. Row 2 needs a base: with no certificate every Definition is on the list. A change of a file that realizes a Definition the list did not reach puts that Definition on the list, so row 2 names what lies in no item at all, such as a file of the ignore file. A Result page of a refusal is named by the state and the outcome, so calling `gate` again rewrites it. `gate` never commits; the driver commits the work together with the certificate.

**The default driver.** The `/markitect-apply` and `/markitect-verify` skills call `checklist --start`, with `--review` for a review, which begins the work, or goes on with the work under way, and prints the driver's step instead of an agent's task: start the builder, or the reviewer, and run `gate` (or `gate --findings`) when it has ended; work whose review is complete, or that `gate` has stopped, goes straight to `gate`. A review with an empty list leaves no work behind. The builder goes through `gate` itself and starts the reviewer as its own subagent when `gate` says so, so it keeps its context through the repairs; the owner's session runs `gate` again when the builder has ended and starts whichever agent it names. A builder started again sees in its `checklist` what the last review found wrong. The session commits the work with the certificate.

**`gate --verify`** checks that the folder matches its newest certificate and names the files that do not, for a checkout in CI or anywhere no agent reaches. **`gate --findings`** closes a review of finished work, such as `checklist --base CERTIFICATE --review` and its answers: the Result (`gate-…-verify.md`) and, with Findings, the Findings page (`gate-…-findings.md`) go into the folder, the input of `checklist --findings`, and no certificate. **`impact --base`** also takes a certificate, by its `.json` or `.md` path. **`status`** adds, when no Run is open, how the work stands for the files as they are, and which files differ from the newest certificate; that part reads and changes nothing.

## Tests

`run_test.go` runs whole Runs over real Git, with the test playing the agents: every row of `submit`, repair rounds, landing over the work in the folder, one Run per folder, the hook on `answer`, the limits and `verify`. `gate_test.go` does the same without a Run, on plain folders without Git: the rows of `submit` that `gate` keeps, the certificate and the newest one after a merge, answers that fall away when their files change, `--verify` and `--findings`, and `gate` beside an open Run.
