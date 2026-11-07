# Plan: Markitect states facts

**Status:** proposed on 7 November 2026, at the owner's request, after a discussion of what Markitect's commands put out. The owner asked for output that is as uniform and clean as is sensible. They also said that the work needs no maximum number of repair rounds. Nothing here is decided until the owner accepts it ([decisions](#decisions-for-the-owner)). It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and on the [lean plan](lean-tool-plan.md) ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)).

## The idea

Markitect is a compiler and build system for a spec. Definitions are the sources, Packages the modules, links the imports, and Kinds the types. The files on the registry are the build targets, the builder and the reviewer are the toolchain, and the certificate is the build stamp. Such a tool states facts: what holds, what is to be done, and by whom. It does not write prompts, and it writes files only where a fact must outlive the call.

Today the commands put out three kinds of thing in no common shape:

- **facts:** errors, lists, hashes;
- **instructions to agents**, which are prompts in all but name: the builder's task, the reviewer's answer format, and "start the markitect-builder agent with exactly this message";
- **files**, which some commands write on their own: the work state in the user's cache folder, Result pages of refusals, the certificate.

The exit codes differ too. `gate` exits 1 when it names the next step, `checklist --start` always exits 0, and `answer` exits 1 while items are still open.

## The rules

1. **One result per command, of facts only.** Every command puts out a result of the same shape ([below](#the-result)). It holds no prose meant to steer an agent.
2. **The tool computes the next step, and the harness says how to do it.** A result names its next step as data: which agent to start and with which message, what to commit, or what to put to the owner. All prose for agents lives in the agent files and skills that `init` and `update` write ([below](#prose-moves-to-the-harness)).
3. **Only the commands that record a fact write, and only into places the tool owns.** Commands that read never write. The tool's own state lives in `.markitect/work/`, and records that outlive the work live in `.markitect/results/` ([below](#who-writes-what)).
4. **One meaning of exit codes everywhere:** 0 when the result's state is `ok`, 1 for any other state, 2 when the command could not run.
5. **No limit on rounds.** `gate` keeps no count limit. How long the work may run is the driver's, and so the owner's.

## The result

Every command, query or not, puts out the same structure. It is text for people and agents, and JSON with `--json` for scripts.

| Field | Holds |
|---|---|
| `command` | The command and its flags, as run |
| `state` | `ok`: what was asked holds, or there is nothing to do. `todo`: an agent has work to do, as `next` says. `owner`: only the owner can go on |
| `summary` | One line of facts, such as "3 items have no answer for the files as they are" |
| `entries` | The facts behind it, typed per command: errors, items, files, links, measures. Each entry has a stable code where it is a finding, as `check`'s errors already do |
| `next` | The next step, or none. One of: `start`, with an agent and the exact message; `commit`, with the files to commit beside the work; `ask`, with the question for the owner; `run`, with a command, such as `markitect update` |
| `written` | The files the command wrote, if any |

The text form puts `state` and `summary` on the first line, the entries under it, and `next` last, on one line. An error that keeps the command from running, such as a usage error or an unreadable folder, is printed to standard error and exits 2, with no result.

**`--out PATH`** writes the result to that file instead of the console, as one file written whole, in UTF-8. Every command takes it, and its exit code does not change. That spares a driver on Windows the shell's own encodings, and a CI job gets a file to keep. A command never decides by itself where its result goes.

### States and next steps, per command

| Command | Question | State | `next` |
|---|---|---|---|
| `check` | Does the project compile? | `ok`, or `owner` for errors in the spec, `todo` for errors only in the realization | None |
| `refs`, `impact`, `lint`, `graph` | Who links to what, what does a change reach, how is the spec built | `ok` | None |
| `version`, `update --check` | Do the binary, the skill and the harness match? | `ok`, or `todo` | `run`: `markitect update` |
| `checklist` | What is on the list of the work, for the builder (`--review`: for the reviewer) | The work's state, as `gate` would give it, without certifying | As `gate` |
| `checklist --start` | The owner's step: begin the work, or go on with it | The work's state | `start` the builder, or with `--review` the reviewer |
| `answer` | Store a reviewer's answers | `ok` when every item has an answer; `todo` while some are open or a line was refused, with the reasons as entries | None, or `start` the reviewer, which the reviewer's own file says means itself |
| `gate` | Is the work done? | `ok` when it passed or nothing changed, `todo` when an agent has work, `owner` for a spec problem, a spec changed under the work, or a spec that does not compile | `commit` with the certificate, `start` an agent, or `ask` |
| `gate --verify` | Does the folder match its newest certificate? | `ok`, or `todo` with the files that differ | None |
| `gate --findings` | Close a review of finished work | `ok`, or `todo` while items are open | `commit` with the Findings |
| `status` | How the work stands, and what differs from the newest certificate | The work's state | As `gate` |
| `init`, `install`, `update` | Set up or refresh the project, the skill or the binary | `ok`, with what they wrote | None |

A state of `todo` from the builder's own `gate` names the builder in `next` when the work goes back for repair. The builder's agent file says that this means itself. So `next` is the same fact whoever asks, and the tool needs to know nothing about its caller.

**The skill's first command.** Claude Code aborts a skill whose injected command exits non-zero. Since `checklist --start` would now exit 1 with work to do, the skill injects `` !`markitect checklist --start $ARGUMENTS || true` ``. The provider's quirk stays in the provider's harness, and the tool keeps one rule.

## Prose moves to the harness

These texts leave `internal/run` and go to the files the harness already has. `init` and `update` write them, and `version` says when a project's copy is older than the binary.

| Today in the tool | Moves to |
|---|---|
| The builder's task (`stateTask`), and what the last review sent back, worded as an instruction | `markitect-builder.md`. `checklist` still lists what was sent back, as entries: item, answer, the reviewer's explanation |
| What a spec is and the three commands to look with (`fixedPart`) | Both agent files |
| The reviewer's task and the answer format (`answerFormat`) | `markitect-reviewer.md` |
| Which agent to start, and what to run when it has ended (`startStep`, `reviewNext`) | `next`, plus one rule in each skill: do what `next` says, and when an agent you started has ended, run `markitect gate` and do what its `next` says |
| Refusal texts that tell an agent what to do ("put them back", "list them on the registry") | Entry codes. The agent file says what to do for each code |

An agent's file is its system prompt, so it outlives the compaction of that agent's context. The owner's session keeps only the skill's rule, and the concrete step arrives fresh in every result's `next`. So the session needs to remember less than today.

## Who writes what

| Command | Writes | Where |
|---|---|---|
| `check`, `refs`, `impact`, `lint`, `graph`, `status`, `version`, `licenses`, `checklist`, `checklist --review`, `gate --verify` | Nothing | |
| `checklist --start` | Begins the work, or names a change: the base, the hashes of what the work may not change, the list | `.markitect/work/` |
| `answer` | The answers, bound to the hashes of the files they cover | `.markitect/work/` |
| `gate` | The reviewed state, and the builder's report with `--spec-problem` | `.markitect/work/` |
| `gate`, when it passes | The certificate, and its Result as a Markdown page beside it; then it removes the work | `.markitect/results/` |
| `gate --findings` | The Findings as data, their page beside them, and the Result; then it removes the work | `.markitect/results/` |
| `lock`, `init`, `install`, `update` | The lock, the harness, the personal skill, the binary | Where each belongs today |
| Any command, with `--out PATH` | Its result | `PATH` |

**What changes:**

- A plain `checklist` never begins work. Only `--start`, the owner's step, does. `--base`, `--full` and `--findings` without `--start` show that change and record nothing. The hook then needs only to refuse a subagent's `--start`, instead of four flags.
- A refusal of `gate` writes no Result page. Its result says why, and the work's state in `.markitect/work/` keeps the history. Only facts that outlive the work go into `results/`: a certificate and Findings. Each is data, a JSON file, with its Markdown rendering beside it.
- The Result beside the certificate still records the number of repair rounds, as a fact.

### The work's state moves into the project

Today it lives in the user's cache folder, under a hash of the project's path. It moves to `.markitect/work/`, under four conditions:

1. **The tool is its only writer.** The hook refuses Write and Edit there for every agent, as it refuses `answer` to every agent but the reviewer.
2. **It is outside the walk and the hashes.** `init` writes `.markitect/.gitignore` with `work/`, so the walk skips it like any ignored file, and Git leaves it out of commits. `gate` refuses a change of that ignore file as it refuses one of any `.gitignore` file.
3. **It is a journal that only grows.** It holds one line per event: the work begun, a batch of answers, a reviewed state, a report and its answer. The current state and the number of rounds are read from it, so nothing is counted in a field that an edit could reset.
4. **One writer at a time.** A lock file in the same folder, as the cache has today.

What this buys: the owner sees the state where the spec is. It survives moving or renaming the folder, and every worktree has its own. All of Markitect's data then lives under `.markitect/`. What it costs: an agent sees the folder and could write it through the shell. That is the same class of gap as a call of `./bin/markitect` past the hook: the guard catches mistakes, not intent ([concept 4.0 §7](v4-concept.md#7-how-sure-we-are)).

## No limit on rounds

`gate` no longer refuses after two repair rounds, and the builder's reports of a spec problem are no longer capped at two that were turned down. One rule stays: a report cannot be repeated on a state on which a reviewer turned one down. It is not a limit but the end of a loop that would cost a review and change nothing. The default driver sets no limit of its own, so the work goes on until `gate` passes or needs the owner, or until the owner stops it. The Result says how many rounds it took.

## What this leaves alone

The checks themselves: `gate`'s rows apart from the round limit, the certificate's content, how answers bind to files, the folder walk, and impact. Also the hook's rule on `answer`, the commit by the driver, and the absence of Git in the tool. This plan changes what the tool says and where it keeps its state. It does not change what it checks.

## Decisions for the owner

Each has a recommendation.

1. **One result shape** with the states `ok`, `todo` and `owner`, the field `next`, and the exit codes 0, 1 and 2. Recommended.
2. **The tool computes `next`; the prose lives in the harness.** Recommended. The alternative, facts only with no `next`, leaves every driver to work out the step from the state, which is the process in prose again.
3. **The work's state in `.markitect/work/`**, under the four conditions above. Recommended. The alternative is to keep it in the user's cache folder: hidden, so less likely to be edited by mistake, but invisible to the owner and lost when the folder moves.
4. **Only certificates and Findings go into `results/`, each as data with its page.** Refusals write nothing. Recommended.
5. **`--out PATH` on every command.** Recommended: cheap, and one rule for every command.
6. **A plain `checklist` begins nothing.** Recommended.
7. **No limit on repair rounds**, decided by the owner on 7 November 2026, **and none on rejected reports** either, keeping only the rule against repeating a report on the same state. Recommended for the reports.

Once accepted, the decisions go into the register as one entry. Concept 4.0's §3, §5, §6 and §9, the [project workflow](../guides/project-workflow.md#7-apply-and-verify) and [the work](../../internal/run/README.md) follow in the same changes as the code.

## Order of work

1. **Decide** the points above.
2. **The result.** One result type in `internal/run` and the CLI, its text and JSON forms, `--out`, and the exit codes for every command. The current texts stay as the summaries for now. One PR.
3. **Prose to the harness, and `next`.** Move the texts into the agent files and skills, and make the skills follow `next`. Remove the round limit and the cap on reports. One PR and a release, since the harness and the binary change together.
4. **The work's state into the project.** The journal in `.markitect/work/`, the ignore file `init` writes, the hook's guard on that folder, and `update` moving a project's work under way out of the cache. One PR.
5. **Try it** on a small project, with planted defects, under a bounded dispatch. The evaluation projects are gone from Main; the Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks) for that.
