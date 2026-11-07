# Plan: Markitect states facts

**Status:** proposed on 7 November 2026, at the owner's request, after a discussion of what Markitect's commands put out. The owner asked for output that is as uniform and clean as is sensible, and decided that the work needs no maximum number of repair rounds. An independent architect's review shaped this version. Nothing else here is decided until the owner accepts it ([decisions](#decisions-for-the-owner)). It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and the [lean plan](lean-tool-plan.md) ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)).

## The idea

Markitect is a compiler and build system for a spec. Definitions are the sources, Packages the modules, links the imports, and Kinds the types. The files on the registry are the build targets, the builder and the reviewer are the toolchain, and the certificate is the build stamp. Such a tool states facts: what holds, what does not, and what step comes next. It writes no prompts. It writes a file only where a fact must outlive the call.

Today the commands put out three kinds of thing in no common shape:

- **facts:** errors, lists and hashes;
- **instructions to agents**, which are prompts in all but name: the builder's task, the reviewer's answer format, "start the markitect-builder agent with exactly this message";
- **files** some commands write on their own: Result pages of refusals beside the certificate, besides the work's state in the user's cache folder.

The exit codes differ too. `gate` exits 1 when it names the next step, `checklist --start` always exits 0, and `answer` exits 1 while items are open.

## The rules

1. **One result per command, made of facts.** Every command puts out a result of the same shape ([below](#the-result)), with no prose meant to steer an agent.
2. **The next step is data, and only the tool's two steering commands compute it.** `checklist --start` and `gate` name the next step: which agent to start and with which message, what to commit, or what to ask the owner. All prose for agents lives in the agent files and skills that `init` and `update` write ([below](#prose-moves-to-the-harness)).
3. **Commands that read never write.** Only the commands that record a fact write: `checklist --start`, `answer` and `gate`. They write into the work's state, as today, and `gate` writes into the project only the records that outlive the work ([below](#who-writes-what)).
4. **One meaning of exit codes:** 0 when the result is `ok`, 1 when it is not, 2 when the command could not run. One command is an exception, for a reason of the provider's ([below](#the-skills-first-command)).
5. **No limit on rounds.** `gate` keeps no count limit. How long the work may run is up to the driver, and so up to the owner. Two state rules remain against loops that would change nothing ([below](#no-limit-on-rounds)).

## The result

Every command, query or not, puts out the same structure: text for people and agents, and JSON with `--json` for scripts.

| Field | Holds |
|---|---|
| `command` | The command and its flags, as run |
| `ok` | Whether what the command checks holds, or what it was to do is done |
| `summary` | One line of facts, such as "3 items have no answer for the files as they are" |
| `entries` | The facts behind it: errors, items, files, links, measures. A finding has a stable code and a one-line factual message, as `check`'s errors already do |
| `next` | The next step, or none. `start` names an agent and the exact message; `commit` names the files to commit with the work; `ask` gives the question for the owner; `run` names a command, such as `markitect gate` or `markitect update` |
| `written` | The files the command wrote, if any |

The text form puts `ok` and the summary on the first line, the entries under it, and `next` last, on one line. An error that keeps a command from running, such as a usage error or an unreadable folder, goes to standard error and exits 2.

There is no third state. Whether an agent or the owner acts next is what `next` says, and a field that repeated it would disagree with it somewhere.

**`--out PATH`** writes the result to that file instead of the console, as one file written whole, in UTF-8; the exit code does not change. It first removes an older file at that path, so a run that could not finish leaves no stale result behind. A path inside the project must be one the walk ignores, or `gate` sees an unlisted file. Every command takes it except `hook`, whose output is the provider's protocol, and `licenses`.

### Per command

| Command | `ok` means | `next` |
|---|---|---|
| `check` | The project compiles. The entries keep today's two groups: the spec, which the owner settles, and the realization, which the work settles | None |
| `refs`, `impact`, `lint`, `graph`, `licenses` | The answer is printed | None |
| `version`, `update --check` | The binary, the personal skill and the harness match | `run`: `markitect update`, when they do not |
| `checklist` | The list is printed. It is a view, for the builder, or with `--review` for the reviewer, of the work under way. With `--base`, `--full` or `--findings` it shows that change instead, and records nothing | None. The builder's file says when to run `gate` |
| `checklist --start` | The work began or went on | `start` the builder, or with `--review` the reviewer; or `run`: `markitect gate`, when the review is complete or the work stopped |
| `answer` | Every item has an answer for the files as they are | None. The entries name what is open and why a line was refused |
| `gate` | The work passed, or nothing changed | `commit` with the certificate and its Result; `start` the reviewer or the builder; or `ask`, for a spec problem or a spec that does not compile |
| `gate --verify` | The folder matches its newest certificate | None |
| `gate --findings` | The review of finished work is closed | `commit` with the Findings and the Result; or `start` the reviewer while items are open |
| `status` | The folder matches its newest certificate and no work is under way | `run`: `markitect gate`, when work is under way |
| `init`, `install`, `update`, `lock` | They wrote what they list | None |

Only `gate` and `checklist --start` compute a step for an agent. A plain `checklist` and `status` don't. Before any building, `gate`'s rows cannot tell "nothing built yet" from "built, waiting for review", so a view that guessed would send the builder to the reviewer.

**`gate` and `answer` with no work under way** give `ask`: there is nothing to check here, and the owner decides. They never name `checklist --start`. Otherwise an agent that lost the work and then followed `next` would begin the work again from a spec it changed.

**What `gate`'s rows give.** Row 1 (what the work may not change has changed) gives `start` the builder, with the files as entries, as today's "put them back" does. Only the owner begins the work again after a change of the spec, with `/markitect-apply`. A spec problem, and a spec that does not compile, give `ask`.

### The skill's first command

Claude Code aborts a skill when its injected command exits non-zero. So `checklist --start` exits 0 whenever it produced a result, `ok` or not, and prints why it could not begin, such as an error in the spec or a missing harness, on standard output, as it does today. This is the one exception to the exit codes, named as one. The alternative, `|| true` in the skill, would hide the message of an exit 2, would not parse in PowerShell 5.1, and would widen the skill's allowed tools.

## Prose moves to the harness

These texts leave `internal/run` and go to the files the harness already has. `init` and `update` write them.

| Today in the tool | Moves to |
|---|---|
| The builder's task (`stateTask`) and what the last review sent back, worded as an instruction | `markitect-builder.md`. `checklist` still lists what was sent back, as entries: the item, the answer and the reviewer's explanation |
| What a spec is, and the three commands to look with (`fixedPart`) | Both agent files |
| The reviewer's task and the answer format (`answerFormat`) | `markitect-reviewer.md` |
| Which agent to start, and what to run when it has ended (`startStep`, `reviewNext`) | `next`, plus one rule in each skill: do what `next` says, and when an agent you started has ended, run `markitect gate` and do what its `next` says |
| Refusal texts that tell an agent what to do, such as "put them back" or "list them on the registry" | Entry codes with a one-line factual message. The agent files say what to do for each code |

**Compaction.** An agent's file is its system prompt, so it outlives the compaction of that agent's context. The owner's session keeps only the skill's one rule, and the concrete step arrives fresh in every `next`.

**A harness older than the binary.** The plugin updates the binary, while a project's agent files stay at the version `init` or `update` wrote. So the harness carries the version that wrote it, and `checklist --start` refuses a harness of another version, with `run`: `markitect update`. A Go test checks that the embedded agent files explain every entry code the tool can emit.

**"Start the builder" when the builder asks.** When `gate` sends the work back, its `next` starts the builder, whoever called it. The builder's file says that this means itself, and its tools allow it to start only the reviewer (`Agent(markitect-reviewer)`), so the rule is held in structure, not only in prose.

## Who writes what

| Command | Writes |
|---|---|
| `check`, `refs`, `impact`, `lint`, `graph`, `status`, `version`, `licenses`, `checklist` and `checklist --review` without `--start`, `gate --verify` | Nothing |
| `checklist --start` | Begins the work or names a change, in the work's state |
| `answer` | The answers, bound to the hashes of the files they cover, in the work's state |
| `gate` | The reviewed state and the builder's reports, in the work's state. When it passes: the certificate and its Result page under `.markitect/results/`, then it ends the work |
| `gate --findings` | The Findings as data, their page and the Result under `.markitect/results/`, then it ends the review |
| `lock`, `init`, `install`, `update` | The lock, the harness, the personal skill and the binary, as today |
| Any command but `hook` and `licenses`, with `--out PATH` | Its result |

**The work's state stays in the user's cache folder**, one file per project folder, as today. Moving it into `.markitect/` was weighed and rejected. Anything there outside the fixed layout is a spec error ([`project.Load`](../../internal/project/project.go)). The hook would grow a guard on Write and Edit. `update` would need a migration, `git clean -X` would wipe the work, and an `rm` through the shell would still pass. Every worktree already has its own state, since the key is the folder's path.

**What changes:**

- A plain `checklist` never begins work. Only `--start` does, the step every driver takes to begin: the skill, a person or a workflow. It refuses where the Claude harness is incomplete, as today, and `markitect update` sets that right. A driver for another provider brings its own guard and its own check of it.
- A refusal of `gate` writes nothing into the project. Its result says why, and the work's state keeps the history. Only facts that outlive the work go into `results/`: a certificate and Findings, each as data with its Markdown page beside it. The Result still records how many repair rounds the work took.
- The hook refuses a subagent's `checklist --start`, the one call that writes, instead of four flags, and refuses `answer` to every agent but the reviewer, as today.

## No limit on rounds

`gate` no longer refuses after two repair rounds, and the builder's reports of a spec problem are no longer capped at two that were turned down. The default driver sets no limit of its own: the work goes on until `gate` passes or asks the owner, or until the owner stops it. The Result says how many rounds it took.

Two rules about state, not counts, remain against loops that cost a review and change nothing:

- **Reports wait for a review.** After a reviewer turned a report down, the builder can report again only once a full review has finished. Today's rule, "not on the same state", is bypassed by any edit, so it goes.
- **Disagreement has no rule yet.** A builder and a reviewer that disagree on an item without end are stopped by the owner. Unattended work, the scheduled jobs of [concept 4.0 §12](v4-concept.md#12-schedules-and-continuous-work), counted on the two rounds as their budget, so that work brings a bound back. A candidate: an item sent back on two reviews in a row goes to the owner.

What goes: `repairRounds`, the refusal after the rounds in `gate` and the "stopped" case of `checklist --start`, the cap on rejected reports, and the field and checks that served them.

## What this leaves alone

The checks themselves: `gate`'s rows apart from the round limit, the certificate's content, how answers bind to files, the folder walk and impact. Also the place of the work's state, the hook on `answer`, the commit by the driver, and a tool without Git. This plan changes what the tool says and what it writes into the project, not what it checks.

## Decisions for the owner

Each has a recommendation.

1. **One result shape:** `ok`, a summary, entries with codes, `next` and `written`, in text and JSON. Exit codes are 0, 1 and 2, with `checklist --start` always 0. Recommended.
2. **The next step as data from `gate` and `checklist --start`; the prose in the harness**, with a version check of the harness and a test that every entry code is explained. Recommended. The alternative, facts with no `next`, would leave every driver to work out the step itself, and that is the process in prose again.
3. **The work's state stays in the user's cache folder.** Recommended, for the reasons above.
4. **Only certificates and Findings go into `results/`, each as data with its page; refusals write nothing.** Recommended.
5. **`--out PATH` on every command but `hook` and `licenses`.** Recommended: it costs little, and it gives every command one rule.
6. **A plain `checklist` begins nothing; `--start` begins the work for every driver.** Recommended.
7. **No limit on repair rounds**, decided by the owner on 7 November 2026. **No cap on rejected reports**, with the rule that a report waits for a full review. **No rule for disagreement until unattended work needs one.** Recommended.

Once accepted, the decisions go into the register as one entry. The pages that describe them change in the same pull requests as the code: concept 4.0 §3, §5, §6 and §7 (its limits row), §9 and §12; the [project workflow](../guides/project-workflow.md#7-apply-and-verify); and [the work](../../internal/run/README.md).

## Order of work

1. **Decide** the points above.
2. **The result.** One result type in `internal/run` and the CLI, with its text and JSON forms, `--out`, and the exit codes for every command. Today's texts stay as summaries for now. One PR.
3. **`next`, prose to the harness, no limit.** `gate` and `checklist --start` name the next step. The texts move into the agent files and skills, together with the harness's version stamp and the code test. The builder's Agent tool narrows to the reviewer. The round limit and the cap on reports go, and the rule for reports comes in. `checklist` becomes read-only, refusals stop writing pages, and the hook narrows to `--start`. One PR and a release, since the harness and the binary change together.
4. **Try it** on a small project with planted defects, under a bounded dispatch. The evaluation projects are gone from Main; the Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks) for it.
