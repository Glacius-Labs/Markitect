# Markitect concept 4.0 (draft)

**You model. Markitect's tools check the spec and work out what a change reaches, an ordinary agent in a background session of its own makes the project match, and one tool command decides whether the work counts.**

| | |
|---|---|
| **Version** | 4.0 – draft, not accepted |
| **Date** | 4 November 2026 |
| **Basis** | The [owner's statements](sources/fresh-start-20261104.md) of 4 November 2026, concept 3.0, a workflow trial on one real change, the Claude Code documentation and the help of version 2.1.318 as read that day, and independent reviews of this draft and an earlier one. |
| **State** | A draft for the owner ([OQ-012](register.md#open-questions)). [Concept 3.0](v3-concept.md) stays the target until this one is accepted. It is built on the branch `dev/native-apply` for the owner to try by using it ([what real sessions showed](../validation/native-version-20261104.md)). |

It uses the words of [concept 3.0](v3-concept.md#2-words): spec, implementer, reviewer. Code and older documents still say model, maker and checker.

## 1. The aim

Make one clean and simple version work reliably on Sonnet. The aim is no longer to prove that one version is better than another.

The problem has not changed: a project holds what is, and what should be lives in people's heads. [Concept 3.0 §1](v3-concept.md#1-what-markitect-is-for) states it as six problems, and they stand. What changes is how much machinery sits around the agent.

In the owner's words, the point is that "the spec as is gets applied reliably", with no babysitting, review or other slowing down, so that people "concentrate on what agents cant do by themselves: design and model". Every choice below is judged by that.

**The way of working:**

1. A person designs or changes the spec in an ordinary agent session.
2. When the spec looks right, the person uses the apply skill.
3. A session of its own applies the spec as it stood at that moment, on a branch and in a folder of its own. It asks nothing, and nobody has to watch.
4. The person goes on modeling. The session works on a frozen copy, so nobody models in a spec that is being applied.
5. When the final check passes, the person's branch moves forward by itself. They read the Result when they want to, and answer only when the spec has a problem.
6. Other Commands, such as a bug hunt, a cleanup or a check that spec and code match, also run on a schedule.

## 2. What Markitect is

| Part | What it is | Changes with the provider? |
|---|---|---|
| **Spec** | Typed documentation in Markdown, with the registry. Unchanged ([concept 3.0 §3 and §4](v3-concept.md#3-the-spec)) | No |
| **Tools** | One program, `markitect`. It holds every fact, every rule that can be checked without judgment, and the process | No |
| **Harness** | One reviewer agent, a few one-line skills and the instruction files that `markitect init` writes, and the prompt and settings a Command generates for each session | Yes |

**One goal and one door.** A session is told: make the project match the spec, and you are done when `markitect submit` says so. Each time it is called, `submit` says what is next: fix what `check` reports, start the reviewer, fix these wrong items, or landed.

This is the main difference from the first draft of this concept, which kept the steps in a skill and added hooks to catch what an agent forgot. With the steps in `submit`:

- a forgotten step cannot land, and the agent is told what is missing;
- compaction cannot lose the process, because the agent gets its next step from the tool each time;
- the prompt stays a task and an answer format, as [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) asks;
- only a few small hook checks are needed.

What is left to the agent is what needs judgment: how to build it, and whether the code meets the spec.

There is no container, no Docker image and no token. No Command stays running while the agent works: `apply` starts a session and ends, and the session only calls `markitect` like any other tool. [§5](#5-what-apply-does) says why this is the native version the owner asked for.

## 3. The tools

### Commands on the spec

None of these starts an agent. `--repo DIR` names a folder inside the project's repository (default: the current folder), and `--json` prints JSON for scripts.

| Command | Does | Change from today |
|---|---|---|
| `install` | Writes the personal skill, once per machine ([§9](#9-the-harness)) | New |
| `init` | Sets up a project: `.markitect/`, the first page of the spec, the harness, and `AGENTS.md` and `CLAUDE.md` if they are missing. An empty folder is enough: it becomes a Git repository. Run again, it refreshes the harness and never touches spec pages | Writes the reviewer agent and the new skills, and commits what it writes, so the first `apply` finds a clean folder. It writes no file that would hold nothing yet, and it ends by saying what Markitect is and pointing to `/markitect-start` |
| `lock` | Pins the Libraries. It holds a fingerprint of each one, as `go.sum` does for modules, and `check` refuses a Library that no longer matches | None |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself | Stale entries and unused Definitions become errors, except in a project with nothing built yet. `--spec` is new |
| `refs TARGET` | Who links to a Definition or Package, and which files realize it | None |
| `impact` | What a change reaches ([below](#how-impact-works)) | `--base` defaults to the last Result. The per-Package lists go |
| `analyze` | Measures the spec's design ([§4](#4-is-the-spec-good)). `--severity` filters its findings, and `--graph` draws the Package dependency graph | New. Replaces `lint` and `graph`. Comes after `apply` is reliable ([§13](#13-order-of-work)) |
| `version`, `licenses` | As named | None |

The owner proposed `update`, to replace `lock` and `init`'s refreshing. This draft holds it back: the two commands already do that work, and a new name makes nothing more reliable ([§16](#16-questions-for-the-owner)).

### What `check` refuses

`check` passes or fails, with no warnings: a project either compiles or it doesn't. Its errors come in two groups, and it prints them apart. One case passes: when the registry lists no file and the only errors are Definitions nothing realizes, it says that the spec compiles and nothing is built yet. `submit` makes no such exception.

| Group | Errors | Who settles them |
|---|---|---|
| **The spec** | A Kind or link that does not resolve, a Package cycle, a Library that does not match the lock, and the rest of today's errors in the spec | The owner, in the spec |
| **The realization** | A file neither listed nor ignored, a registry entry for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes | The implementer, in the work |

So `check` goes red the moment the owner deletes a Definition or adds one nothing uses yet, and stays red until `apply` has realized the change. That is how it should read: spec and code don't match yet. `--spec` answers the narrower question "is the spec itself valid?", which is what a modeling session asks.

`markitect apply` stops only on errors in the spec. Errors in the realization become checklist items, and `submit` refuses until none is left.

On finished Makerspace projects, at steps c5 and c8, there are no stale entries and one unused Definition, `documentation`, out of about 115. So a finished project can meet this rule. A rule that applies everywhere still needs a file that realizes it, or a link from something that does.

### How `impact` works

Like an incremental build, it works out what a change touches, so that only that gets reworked and reviewed. It compares a base commit with the working tree.

| Distance | Reached | On the checklist |
|---|---|---|
| 0 | Definitions that changed, Definitions whose files changed, and Definitions a registry row gained or lost | Yes |
| 1 | Definitions that link to a changed one or are of a changed Kind | Yes |
| 2 and more | Everything further along the links, with distance and reason | No |

It doesn't follow the code's own imports. A full `verify` now and then is the safety net. The list is rebuilt after the work, so a file touched outside the first impact adds its Definitions. On Makerspace's change steps, one step reached 33 of the 36 existing files that truly changed, and deeper reach 35 ([measurement](../validation/impact-reach-20261103.md)). How far every link reaches says something about the spec too ([§4](#4-is-the-spec-good)).

### Commands of a run

They are thin. The checklist, answer parsing, Result page and trailers exist today in `internal/run`. Run state on disk and a `submit` that can be killed and run again are new, and the riskiest part to build.

| Command | Called by | Does |
|---|---|---|
| `markitect apply` | The owner, the apply skill or a schedule | Compiles the spec, commits a spec change that is not committed yet, works out the base and the list, starts the session and returns once it is up. Nothing to do means no session ([§5](#5-what-apply-does)) |
| `markitect verify` | The same | Starts a reviewer's session on the last commit and returns ([§8](#8-other-commands)) |
| `markitect checklist` | Implementer and reviewer | Prints the task and the list, rebuilt from the current files and registry. For a repair review, it prints only what is to be checked again |
| `markitect answer` | Reviewer | Checks its answers and stores them against the state under review. A wrong location or an unknown item is refused at once, and it prints what is still missing |
| `markitect submit` | The session | The one door ([§6](#6-the-one-door-submit)) |
| `markitect stop WHY` | The session or the owner | Ends the session if it is still running, then ends the run without landing: commits what is there with a Result on `markitect/runs/<run>` |
| `markitect status` | The owner | Whether a run is open and how its session stands, the last Result, and the commits since it |
| `markitect hook` | Hooks only | The checks of [§9](#9-the-harness) |

**The base.** The last Result on the branch names the work it realized, as today. With no Result yet, the first `apply` is `--full` and says so. Commits made beside `submit` stay in the next `impact`, so a hand fix or a bypass still gets reviewed by the next run.

**State of the work** means a Git tree of every file Git would take, so ignored build output does not count.

**Run state** lives under the repository's Git directory, in `markitect/`, which every worktree shares, and never reaches the working tree. The worktree's own branch is `worktree-<run>`, the provider's name for it. Work that did not land is copied to `markitect/runs/<run>`, where the owner can look at it, and the Result says why.

**There are no required checks.** The owner took them out on 4 November 2026, to see whether they are needed. A project that wants tests states that in the spec, with the command that runs them, and the reviewer checks it and may run them. So no proof in a run comes from outside an agent any more, apart from `check` and what `submit` refuses, and that is what this trial tests. If runs land work that a test would have caught, the checks come back. `execution.yaml` goes with them, and the session's `model` and `effort` move to the manifest.

**The ignore file** is `.markitect/ignore`, written like a `.gitignore`: one pattern per line and `#` for a comment. Its patterns stay the registry's, and it has no `!`. A reason is a comment now and no longer required, which is safe because a run can't change the file. The tool accounts for what `init` installs by itself, so a new project has no ignore file.

## 4. Is the spec good?

A spec is judged like code ([founding intent](sources/founding-intent-20261026.md)), so it gets the same kind of analysis code gets. `check` is the compiler: is it valid? `analyze` is the static analyzer: is it well designed? It only ever warns, since granularity stays the author's choice ([DEC-005](register.md#dec-005-model-granularity-is-the-authors-responsibility)), and its thresholds are provisional.

| Measure | Code analogy | Warns about | State |
|---|---|---|---|
| Efferent and afferent coupling, instability, per Package | Clean Architecture | A Package depending on too many others, or on a less stable one | Exists in `lint` |
| Size of a Definition and a Package | Long functions, large modules | God Definitions | Exists in `lint` |
| Packages with no paths, or no Definitions and one child | Empty namespaces | Structure that holds nothing | Exists in `lint` |
| Fan-in of a Definition, including at Root | God class, `utils` file, interface segregation | One page that too many depend on | New. `lint.reach` skips Root today, so it misses the worst case below |
| Blast radius: how far every link reaches from a Definition, as median, 90th percentile and worst | Transitive dependents | A change that could touch most of the spec | New |
| Links inside a Package against links leaving it | Cohesion | A Package whose parts belong elsewhere | New |
| Change hotspots from Git history | Churn | What keeps changing, with how far it reaches | Later |
| Size against what tests cover | CRAP | Large, unverified Definitions | Later: needs tests bound to Definitions ([ENH-003](register.md#enh-003-test-to-statement-binding-and-test-review)) |
| The spec's dependencies against the code's imports | Architecture tests | Code that has drifted from the design | Later |

**Package cycles stay a compile error**, as in Go. The details of `analyze`, its measures, thresholds and output, are still to be discussed with the owner.

**What this finds on Makerspace, measured on 4 November.** Following every link reached over 85 % of the spec on four of its eight change steps. That looked like a ball of mud, and at the Package level it is not one: there are no Package cycles, dependencies point towards a stable core, and a typical Definition reaches 8 of 113. But it does show a real design smell:

| | Makerspace, 113 Definitions | 5× project, 413 Definitions |
|---|---|---|
| Direct links to one page, `values` | 60 | 268 |
| Definitions that page can reach through every link | 97 | 392 |
| Definitions reaching over half the spec | 9, all at Root | 17, of which 15 at Root |
| Worst reach if the top three hubs pass nothing on | 46 | 188 |

`values` is one page of thirteen unrelated value formats, and most links to it name no format. A Definition that uses only dates still depends, on paper, on the format of money. Treating the top three hubs as stopping points brings the four steps that reached 85 to 88 % down to between 9 and 38 %. In code this would be a `utils` file, and the remedy would be to split it. Whether splitting it really narrows reach, and whether that helps agents, is untested: cutting hubs in a calculation is not the same as a better spec.

**What analysis can't say:** whether the spec is right. No threshold has been checked against outcomes yet, so a clean report doesn't show that a spec leads to better work.

## 5. What `apply` does

The owner models until the spec looks right, then uses `/markitect-apply`, from the same session or another, or types `markitect apply`. From there it runs without them.

1. **Check.** `apply` compiles the spec. An error in the spec stops here, no session starts, and the owner sees why. A change outside `.markitect/` that is not committed also stops it, because it would be silently left out, and it says which files.
2. **Commit and list.** It records the owner's branch, commits the spec change on it, and works out the base and the checklist. Nothing affected means "nothing to do", and it ends.
3. **Start.** It starts a background session in a new worktree, based on that commit. Roughly:

   ```text
   claude --bg --worktree <run> --model <model> --effort <effort>
          --permission-mode dontAsk --settings <the run's settings>
          --setting-sources project --strict-mcp-config
          "<the task, with the run's name>"
   ```

   It waits until the provider reports the session working, records its ID and returns. If the session doesn't start, it closes the run and says why.
4. **Implement.** The implementer takes the list from `markitect checklist` and makes the project match the spec, keeping the registry current. It does not change the spec. If the spec contradicts itself or leaves open something that changes behaviour at the project's boundary, it runs `markitect stop` with why. A gap with no such effect is its own to decide. If the machine lacks something, such as a program or a service, it says so with `markitect stop --environment`, which asks the owner nothing about the spec.
5. **Submit.** When it thinks it is done, it calls `markitect submit`, and follows what that says ([§6](#6-the-one-door-submit)).
6. **Review.** `submit` tells it to start the `markitect-reviewer` agent. The reviewer starts with fresh context, has no tool to edit files and cannot start further agents. It takes the list from `markitect checklist`, reads the Definitions and the code, and stores one line per item with `markitect answer` ([concept 3.0 §8](v3-concept.md#8-the-checklist)).
7. **Repair.** A `wrong` answer goes back to the implementer, which fixes it and calls `submit` again. The next reviewer gets the findings, the new diff and the items whose locations that diff touched; other answers stand. At most two repair rounds.
8. **Land.** When nothing is left to do, `submit` writes the Result, commits it with the work and brings the owner's branch up. The session is told it has landed, and finishes.

**Bringing the branch up.** `submit` builds what will land off to the side: the run's commit if the branch has not moved, otherwise a merge, which must be clean and pass `check --spec`. Later commits stay in the next `impact`, so the next run realizes them. Then it moves the recorded branch, only if the branch is still where it was: `merge --ff-only` where it is checked out, so the owner's uncommitted edits stay, or a ref update where it is not. A short wait and a retry cover Git's own locks. Nothing is pushed: the owner pushes when they want to, to any remote or none.

**If landing fails,** for example because uncommitted edits overlap, the run closes as kept on `markitect/runs/<run>`, the Result says why, and the session is told to finish. The next `apply` commits the spec and tries that branch once more. If that fails too, it leaves the branch where it is and starts a fresh run. The base has not moved, so nothing is lost, and the work is done again.

**One run per branch.** A second `apply` during a run says that a run is open. The owner can keep modeling meanwhile, and the next `apply` commits what they have.

**Telling the owner.** `apply` has returned, so nothing waits for the end of a run. Every `markitect` command, the apply skill included, prints one line about a run that closed unread or died. A desktop notice on close is a small addition ([§16](#16-questions-for-the-owner)). This matters most for what no hook can send back: a usage limit ends the session with an API error, for which the provider fires StopFailure, which can only record the cause.

**Is this native?** Yes, in what the owner asked for: "I use the apply skill", "Claude opens a branch and applies the changes maybe in its own session". The difference from `claude -p`, which the owner turned down, is real:

| | `claude -p` | `claude --bg` |
|---|---|---|
| What it is | The command-line form of the provider's Agent SDK | The provider's ordinary session, sent to the background, listed in `claude agents` and open to attach |
| Login | The documentation says `--bare` will become its default, and bare mode doesn't use a subscription login | The same stored credentials as interactive sessions |
| Who waits for it | A script that drives it | Nobody: `apply` has ended long before the work does |

A forked subagent of the owner's own session would look more native, and was the first draft's answer. It is worse on the owner's own terms: the run would hang on that session, and that session's auto mode would be passed on to the agents. It stays the fallback if a session cannot start another ([§14](#14-what-is-not-yet-shown)).

**It asks nothing.** Agents can't ask the owner a question, and they run in the `dontAsk` permission mode, which refuses what is not allowed instead of waiting. That matters more in the background, where a prompt would stop the session until someone came back. Auto mode doesn't fit: after 3 blocks in a row or 20 in total it goes back to prompting. So the session may edit files and use any shell command, and [§16](#16-questions-for-the-owner) puts that to the owner. A short list of commands won't do: Makerspace c6 deletes a test file, and `dontAsk` would refuse `rm`. The provider's protected paths, such as `.git`, `.husky` or `.vscode`, stay refused for the edit tools.

**After a reboot or a crash.** The work is in the run's worktree, not in the owner's folder, and the run is still open. Only a session the provider reports as working counts as in progress: a finished session's process lives on for about an hour, and that doesn't count. So `apply` ends such a process, resumes the conversation in the background in the run's own worktree with the same settings, and records the ID that comes back, since the provider may start a copy. The repair count stands, and the deadline starts again. A run whose worktree is gone is not resumed. `markitect stop` ends the run instead. Only `submit` and `stop` close a run. A run stays open until someone acts, unless a schedule resumes it.

**Cleaning up.** The provider holds a lock on a worktree while its session runs, so `submit` can't remove it. The next `apply` stops the sessions of closed runs and removes their worktrees.

## 6. The one door: `submit`

`submit` looks at the current state and does the first thing that applies:

| | It finds | It does | It tells the agent |
|---|---|---|---|
| 1 | Anything under `.markitect/` changed except the registry pages, or anything in the harness, including `AGENTS.md` and `CLAUDE.md` | Refuses | Put these back, or `markitect stop` if the spec can't be met |
| 2 | A reviewed state, and the only difference from it is new files not listed | Refuses | List them if they are the work, otherwise remove them |
| 3 | An error from `markitect check`, in the spec or the realization | Refuses | The errors |
| 4 | No complete review for this state | Commits the state on the run's branch | Start `markitect-reviewer`, and the items that need answers |
| 5 | A `spec problem` | Closes the run | Nothing lands, and the Result holds the question |
| 6 | A `wrong` answer | Counts one repair round per reviewed state; the third closes the run | The items and what is wrong |
| 7 | Nothing left | Writes the Result, commits and brings the branch up | Landed, or kept and why |

`not shown` may land, and the Result lists it. A change after a review is new work: it passes through rows 3 and 4, and the new review covers the items it touched.

**Limits.** Every `submit` counts towards a ceiling, for now 30, and reaching it closes the run. That also covers a reviewer that keeps leaving items open. A run also has a deadline, for now 6 hours as in today's engine, and the hooks of [§9](#9-the-harness) enforce it.

**Slow checks.** `submit` can outlast a shell call's time limit, after which the provider moves the command to the background and the agent sees no verdict. So `submit` takes a lock, a second call answers "still running" and doesn't count towards the ceiling, no limit closes the run while the lock is held, the run's settings raise the time limit, and a landing that was killed can be run again.

**The Result** is written by the tool, as in [concept 3.0 §10](v3-concept.md#10-the-result): each item's answer and pointers, what could not be tested, and how many changes have landed since the last full check. A landing's Result goes into its commit under `.markitect/results/`. The first version doesn't measure how much checking really happened from the reviewer's transcript, and says "not measured". The provider says that format can change between versions, and reviewers read mostly through the shell, so the measure says little for its cost.

## 7. How sure we are

The tool can't stop a determined agent that runs as the user on the user's machine. This concept guards against forgetting and shortcuts, which is what has been seen, and against deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec compiles before work starts | `apply` runs before any session exists | Firm |
| What `submit` lands is complete: spec unchanged, project compiles, every item answered for the work | The tool | Firm for what `submit` lands. A merge with later commits gets `check --spec`, not another review |
| A run doesn't stop to wait for someone | `dontAsk`, set by `apply` on that session | Firm by the documentation. Nothing has run under it yet: the trial and today's adapter start in `bypassPermissions`. The owner's own mode no longer reaches the run |
| The owner's folder stays out of it | The worktree. The provider blocks edits, command folders and Git commands aimed at the main checkout | Good. It reads the command's text, so a `cp` or a shell redirect into the checkout is not refused. On PowerShell only the command folder is checked |
| The session doesn't just end | A Stop hook sends it back while the run is open | Catches forgetting. The hook closes the run after five blocks. An API error or a closed session gets past it, and then the owner is told ([§5](#5-what-apply-does)) |
| The agent doesn't review itself | A hook lets only a reviewer call `answer` | Good, not proof: it reads the command's text and the agent's name |
| The reviewer judges for itself | Its agent file says to take everything from `markitect checklist` | Weak: the implementer writes the message that starts it. Reviews given a maker's claims did no worse on planted bugs ([17 of 18 blind, 18 of 18 with claims](../validation/k-checker-test-20261103.md)), though that tested claims beside bugs, not a message written to steer. No hook is added until this is seen |
| Nothing goes to a remote | The task says the tool handles Git, and a deny rule covers `git push` and `gh` | Catches ordinary use. Rules match text, so `git -C . push` gets past it |
| A commit that bypassed `submit` is noticed | It stays in the next `impact`, and `status` counts commits since the last Result | Seen afterwards, not prevented |
| The reviewer really read the code | Pointers checked for existence | Not provable. It was not provable in the Docker version either |
| The code is right | The reviewer and the project's tests | Not guaranteed by any version |
| A run has a limit | The deadline and the ceiling of [§6](#6-the-one-door-submit), held by the tool | Good while the agent calls tools. The provider's own budget and turn flags work only with `-p`, and there is no spending limit |

**Anyone who wants more** can run `markitect check` and the project's tests where the agent can't reach them: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these. Answers stay on the machine, and a landing's Result carries them.

**What we give up against concept 3.0:**

- prevention of a deliberate bypass;
- isolation from the user's machine: agents run with the user's rights, so a run must not read untrusted input;
- one operating system: agents work on the user's own, Windows included, with its paths and shells;
- a reviewer in its own session as another user, with a throwaway folder. It now shares the worktree. `submit` catches what it changes in files Git would take, and code it runs can still change its stored answers;
- review of a clean copy: it reads the folder as it stands, ignored leftovers included;
- a reviewer's task written by the Command, and a measure of how much it checked.

## 8. Other Commands

Every Command is one short declaration over the same tools: which agent starts, where its items come from, and whether it lands. The table in [concept 3.0 §12](v3-concept.md#12-other-commands) stands. A skill for a Command is one line that calls it, so the same Command runs from a chat, a terminal or a schedule.

**`verify`** changes nothing the owner has. It starts the reviewer as the main agent of a background session, in a worktree of the last commit, with no edit tools. A second `verify` resumes one that stopped working, and `markitect stop` ends it. It doesn't commit on the owner's branch, doesn't switch anything and doesn't mind a dirty folder; the Result says that uncommitted work was not looked at. With no Result yet, it is `--full`. It never holds the branch, so a scheduled `verify` doesn't block an `apply`.

`submit` works differently for it:

| It finds | It does |
|---|---|
| Items without an answer | Lists them, under the same ceiling and deadline |
| Otherwise | Runs `check`. A wrong answer becomes a Finding, and what names no Definition, such as an error from `check`, is listed in the Result. It writes the Result and a Findings page on `markitect/runs/<run>` and closes the run |

`apply --findings` and the count of changes since the last full check read them there.

`--full` starts with one reviewer for the whole list, and the first `apply` on a project is `--full` too. That is untested at size: the 5× build used 51 reviewers, up to eight at a time, each in a clean copy. A split per Package may be needed, and the runs of [§13](#13-order-of-work) decide that.

`apply` comes first, then `verify`. `cleanup`, `bughunt`, `security` and `improve` follow once those work. Modeling and Extract stay skills in the owner's own session, as they are, and so does `markitect-start`, the guide a new project begins with. It asks what the project is, one question at a time, and starts the spec with the owner.

## 9. The harness

For Claude Code, `markitect init` writes these project files, so they are versioned with the project, and the tool accounts for them itself. It also adds `.claude/worktrees/` to `.gitignore`.

| File | Holds |
|---|---|
| `.claude/skills/markitect-apply/SKILL.md` | One injected command, `` !`markitect apply` ``, and "report what it printed". Its front matter sets `allowed-tools`, so the command runs without a prompt outside auto mode too, and `disable-model-invocation: true`, so a modeling agent can't start a run by itself. The provider runs the command before the model sees the skill, and any non-zero exit aborts the skill, so "run open" and "nothing to do" exit 0 |
| `.claude/skills/markitect-verify/SKILL.md` | The same for `verify` |
| `.claude/agents/markitect-reviewer.md` | Effort, the reviewer's task, and tools without edit or agent tools. Its model is the one forced on subagents below |
| `AGENTS.md`, `CLAUDE.md` | What kind of project this is, for any session: the spec is the authority, changes start there, and `markitect` is on the path. `CLAUDE.md` holds only `@AGENTS.md`, so that it works in every session of the provider. They hold no process, since a run reads them too. Written only if missing |

**The implementer is a plain session,** so it keeps the provider's default system prompt, as every launch so far did. An agent file would replace that prompt. Its task is the first prompt: the goal, what the spec is, its authority, the tools and that the tool handles Git. `markitect checklist` prints that again, so compaction can't lose it. Model and effort for the session come from the manifest.

**The run's settings** are not in the project. `apply` generates them for that session, in the run state, so a modeling session gets none of it:

- `worktree.baseRef` set to `head`. The default branches from the remote's default branch, and would miss the spec commit that was just made;
- `dontAsk`, with file edits and any shell command allowed, and a deny rule on `git push` and `gh`. By default the provider has an isolated background session push its branch and open a draft pull request when it finishes, which the owner doesn't want ("We dont want to rely on github");
- subagents forced onto the chosen model: one earlier launch on Haiku started 68 subagents on Opus;
- auto memory off, so that modeling notes don't reach a run and a run's notes don't reach the next;
- a longer time limit for shell calls;
- the hooks below.

`--setting-sources project` and `--strict-mcp-config` keep the personal settings, plugins and MCP servers of whoever runs it out of the run. They do not keep the user's own `CLAUDE.md` out of a background session, so the settings also exclude it with `claudeMdExcludes`.

**The hooks,** all `markitect hook`, which does nothing unless a run is open:

| Hook | Does |
|---|---|
| Stop | Sends the session back while the run is open and no reviewer is running. The input lists background tasks, so the wait for a reviewer passes. After five blocks since the last `submit` it closes the run and lets the session end, since the provider's own count of eight starts again at any tool call. While `submit` holds its lock, it lets the turn end |
| StopFailure | Records the cause when a turn ends on an API error, such as a usage limit |
| PreToolUse | On every tool call, past the deadline, stops the session and closes the run. On the shell, refuses `markitect answer` unless the input's `agent_type` is the reviewer's |

The first draft's seal is gone. `answer` checks and stores the answers itself, so no hook has to catch a reviewer as it finishes.

**One file lies outside the project.** A session that runs `init` started before the project's files existed, so it knows only what the tool prints, and Claude Code doesn't watch a skills folder that wasn't there when the session started. So help and `init` say what Markitect is, and `markitect install` writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, with the same text from the binary, which reaches a session in any folder. It allows no command in advance, and `--setting-sources project` keeps it out of runs. `init` never writes to the home folder.

Skills are kept short, and they hold no steps, so nothing depends on what survives compaction. Plugins do not fit: the provider ignores hooks and permission modes in plugin agents.

## 10. Strategies

**A strategy is a folder of harness files and nothing else.** There is no code interface to implement. Adding a version means adding a folder, and removing one means deleting it. **It may change** the prompt, agents, skills and settings. **It may not change** the tools or what `submit` refuses.

The first version ships one strategy and no switch for it. The natural second is a saved provider workflow that fixes the order of steps, as in the trial of [§14](#14-what-is-not-yet-shown), added only if the first one fails too often ([§16](#16-questions-for-the-owner)).

## 11. Providers

The first version is Claude only. A provider needs a way to start a background session in a worktree with settings, a way to resume it, hooks on stopping and on tool calls, and an agent type for the reviewer. Codex gets its own strategy once Claude's version is reliable.

No token is needed: sessions run on the user's own stored login in the unmodified program, which removes token passing. Each session uses the user's subscription quota. The legal reading of [concept 3.0 §14, question 10](v3-concept.md#14-decided-on-acceptance) still applies: the owner's own use looks allowed, and other users are not settled. So the owner still asks Anthropic before anyone else runs Markitect, now including whether a tool may start `claude --bg` on a user's login.

**The app.** The owner said "we can remove desktop for now". This draft reads that as the whole app, with its spec views and chats. Its chats are the only other use of containers, so this takes Docker out of Markitect altogether, along with the provider adapters, the session helper, the runtime image and the token. It stays recoverable in Git and can come back once the tool has settled. That reading is not confirmed ([§16](#16-questions-for-the-owner)).

## 12. Schedules and continuous work

[DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps) stands: the loop runs outside Markitect, as a scheduled job that calls `markitect`, and there is no daemon in the product. Its budget per task is the run's deadline and ceiling; a budget per day belongs to the job. Government stays a later step ([IDEA-002](register.md#idea-002-government-delegated-model-maintenance)): its mandate would live in the spec, and a fresh reviewer would accept proposals within it.

Because a Command is a plain command, a schedule needs nothing new: `markitect verify` starts the same background session, and `markitect apply` resumes a dead run. `claude -p` is not involved, so its coming `--bare` default doesn't matter. Two limits stay:

- **The machine must be on and signed in.** Background sessions are local and stop when it shuts down, and in a script `claude --bg` fails on a folder that was never trusted. A server without a stored login is not covered yet.
- **External signals need isolation.** Issue text can hold instructions aimed at the agent, and a native agent has the user's rights. A run that reads outside input needs a container or a machine with limited rights around it. This wraps the harness and needs no engine.

## 13. Order of work

| Step | Result | Done when |
|---|---|---|
| 1. Probe | The harness by hand on the Library, on Windows, under a bounded dispatch. It tries what [§14](#14-what-is-not-yet-shown) lists | Each works, or the concept is changed |
| 2. Build | `apply`, `checklist`, `answer`, `submit`, `stop`, `status`, `hook` and the stricter `check` in Go, reusing what `internal/run` holds. `init` writes the harness. The engine's container and session code goes, roughly 7,000 lines, once the probe has passed | A Library change lands, planted refusals hold, and a `submit` killed at each step can be run again |
| 3. Make it reliable | The 5× project's 8 change steps on Sonnet, with their hidden cases. Failures are fixed in the tool, not with more prompt text | Two clean passes in a row: every step lands without the owner, all hidden cases pass and no refusal was overridden ([§16](#16-questions-for-the-owner)) |
| 4. The rest | `verify`, then the other Commands, `analyze`, the rename, and `update` if still wanted | The owner's reading |

Before step 3, read test 1's nine finished launches. They are in `%LOCALAPPDATA%\markitect-test1`, not in the repository, and their answers are unread. They may say at no new cost whether one reviewer catches known bugs at 5×.

Concept 3.0's tests 2 and 3 fall away, since the aim is no longer to show that this beats something else.

## 14. What is not yet shown

**Tried on Windows on small changes ([record](../validation/native-version-20261104.md)),** so the list below is what the probe set out to show. These hold: the skill's injected command starts a session from Manual mode and from a linked worktree, `baseRef`, the flags and settings together, the reviewer with its narrowed tools, the Stop hook, no push, landing from inside the worktree, and personal rules staying out once `claudeMdExcludes` is set. Writes to protected paths are refused, also through the shell. A resume, the deadline's `continue: false` and a real usage limit are tested only without a provider.

What the probe set out to show:

- the skill's injected command can start `claude --bg` from inside a session, in Manual mode as well as auto, and from a session already in a linked worktree. If not, the owner types `markitect apply` in a terminal, or the fallback is a forked subagent with `background: true` and `isolation: worktree`;
- `--worktree` with `baseRef` set to `head` starts from the spec commit. The workflow trial's first attempt failed on worktree isolation, on a temp path's spelling;
- `--model`, `--effort`, `--permission-mode dontAsk`, `--settings` and `--setting-sources` hold together in a background session, also after a restart and compaction, and it never shows "Needs input";
- a shell write to a provider's protected path works, or is refused in a way the agent can report. A spec that asks for `.husky` or `.vscode` files could otherwise fail with no defect in the spec;
- the session starts the reviewer and waits for it, the reviewer gets its narrowed tools, and `agent_type` reaches the hook. A subagent has its own prompt, where the reviewers so far were full sessions on the default one;
- the Stop hook fires in a background session, sends it back, and lets the wait for a reviewer pass;
- a PreToolUse hook's `continue: false` really ends the session;
- the provider's habit of pushing and opening a draft pull request at the end stays off;
- `submit`, run from inside the worktree, can bring the owner's branch up, and survives outliving a shell call's time limit;
- a stopped session resumes with its settings, and the states in `claude agents --json` tell a live run from a dead one;
- personal rules such as `~/.claude/CLAUDE.md` really stay out, which no flag is documented to do;
- with the deny rule on the shell, a Windows session works through Git Bash alone. The provider turns PowerShell off then, which helps the checks of [§7](#7-how-sure-we-are), but no run has worked that way.

**Background sessions are a research preview,** so their interface can change. Only the launcher in `apply` depends on them.

**What the evidence does and doesn't cover:**

- **A plain Sonnet session has done the work from the typed spec, once.** Concept 3.0's default built the 5× project in one launch, with one implementer session and one repair round, and passed all 327 hidden cases ([record](../validation/default-arm-20261104.md)). A reviewer found a real bug there. That is one launch and not a rate, in a container, driven by the Command, and its change steps have not run. So the implementation doesn't look like the main risk; the process around it does.
- **How often a native agent skips steps is unmeasured.** The owner has seen one forget things. With `submit` as the door, a skipped step should show as a refusal, not a wrong landing. Whether that keeps runs from ending in nothing is what step 3 finds out.
- **The workflow trial is weak evidence.** On 4 November a saved script fixed the order, run through `claude -p` in `bypassPermissions`, with a throwaway check on a local repository. Of four attempts, two ended early: a refused worktree, and a parser that stopped on a helper's extra sentence. In the third the planted bug became a build error, so the reviewer never met it. The fourth started at review, on the second attempt's work plus the plant: the reviewer caught it once, and it landed. It used no skills and no background session. Its evidence is in `C:\Users\Consiliari\wf-trial\evidence`, outside the repository.
- **Reviewing at 5× has no read result.** Test 1 stopped when the machine went down, with nine of twelve launches run.

## 15. What this overturns

| Entry | Becomes | What stays from it |
|---|---|---|
| [DEC-091](register.md#dec-091-concept-30-is-the-target) | Superseded as the target, once this is accepted | Concept 3.0 as the record; its six problems, words, checklist, answers and Result |
| [DEC-037](register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers) | Superseded: agents and checks no longer run in containers | A reviewer gets no write tools |
| [DEC-013](register.md#dec-013-linux-first-for-tests-and-the-playground), [DEC-077](register.md#dec-077-development-pace-and-linux-first-execution) | Amended: agents work on the user's own system, so Windows becomes a main path | Linux tests for Markitect's own code |
| [DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) | No remote involved: a run has a local branch, `submit` brings the owner's branch up, and the owner pushes | Work lands without the owner reviewing a diff; Git as the abstraction, not a host |
| [DEC-080](register.md#dec-080-the-model-is-typed-documentation) | Amended: "Definitions nothing links to and no file realizes" move from warnings of `lint` to errors of `check`, and the command list changes to `check`, `lock`, `refs`, `impact`, `analyze` | The language, and everything else measured staying a warning |
| [DEC-083](register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers) | No containers and no host | Native on Windows, Linux and macOS |
| [DEC-087](register.md#dec-087-required-checks-gate-merged-states-not-each-submission) | Removed for now, by the owner, to see whether they are needed. `execution.yaml` goes too | A merged state must still pass `check --spec` |
| [DEC-066](register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [DEC-089](register.md#dec-089-markitect-desktop-the-app-over-the-cli) | Parked if the owner confirms: no app for now. If it returns, it works over the same commands | The idea of an app over the CLI. The owner being told when a run ends moves to the tool |
| [DEC-092](register.md#dec-092-a-command-uses-the-users-own-sign-in), [DEC-093](register.md#dec-093-a-command-passes-the-users-own-token) | No token and nothing passed on: the session uses the stored login. DEC-092's fallback was to be built only if `claude -p` lost subscription login; that route is no longer used | The unmodified CLI, and the question to Anthropic. The pinned CLI goes, so a change of the provider's defaults can arrive unannounced |
| [The guide's build decisions](../guides/model-language.md#build-decisions) on stale realization, confirmed by the owner on 28 October 2026, and on a required check's environment | Stale entries are errors, not warnings, and the `environment` field goes | The Command that cleans them up still starts: they are errors in the realization |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 1 | The rename no longer waits on test 3. It comes in step 4, once the sealed project's use is settled | One step, with no compatibility kept |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 7 | `apply` commits an uncommitted spec instead of refusing it | A change outside the spec is still refused, for the same reason |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 9 | Services a project needs come from the user's machine, not an image | A run that lacks one says so with `markitect stop --environment` |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 12 | One run per branch | The rule for a branch that moved, in a simpler form |
| [Concept 3.0 §6, §9, §11](v3-concept.md#6-what-apply-does) and [§13](v3-concept.md#13-how-we-know-it-works) | The Command driving Runs, the sandbox, and tests 2 and 3 go. Skills stay optional reading, so they hold no process | The rule that a strategy can't change the check |
| [Concept 3.0 §8 and §10](v3-concept.md#8-the-checklist) | One reviewer first instead of one per Package at `--full`. A pointer no longer has to show as opened in the transcript, and the Result doesn't measure depth | The answers, pointers, and what each answer covers |

**Stays unchanged:** [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process), now served by `submit` in place of an Orchestrator, and the working rule in `AGENTS.md` on fixing failures in the tool contract; the spec's authority and escalation to the owner ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)), explicitness, every file accounted for, removed intent removing its realization, delegated realization, quality measures as warnings ([ENH-002](register.md#enh-002-model-quality-linter-and-metrics)), continuous change as a later loop, and honest results, finite budgets and provider neutrality from concept 1.0.

## 16. Questions for the owner

Each has a recommendation, and none is decided. Already decided by the owner: the name `submit`, own worktree, that `submit` brings the branch up by itself, that required checks go for now to see whether they are needed, and that a plugin waits while the CLI carries the setup.

1. **Starting the session.** Is `markitect apply`, behind the apply skill, starting the provider's background session native enough? Recommended: yes, for the reasons in [§5](#5-what-apply-does). The alternative is a forked subagent of the owner's session, with the costs named there.
2. **What a run may do.** An unattended agent may run any shell command as the user, with the worktree checks and the deny rule around it. Recommended for specs and repositories the owner trusts, since anything narrower refuses real work. The alternative is auto mode, whose classifier reviews each action, with a hook that denies what would prompt; it is safer and refuses more, so it fails runs more often.
3. **How the owner hears of the end of a run.** One line from every `markitect` command is in this draft. Recommended: also a desktop notice on close, or the owner has to go and look.
4. **A run's limit.** A deadline of 6 hours and 30 `submit`s, with no spending limit, since the provider offers none here. Recommended for now; the numbers are guesses for step 3 to correct.
5. **One version or two.** The owner asked for "the two simplest approaches". Recommended: one now, the workflow as the second only if this one fails too often. The owner also expected a package per version behind an interface; a strategy here is a folder of harness files, with no code interface, because the tool does not change between versions.
6. **The old version.** Keep the engine until the probe passes, then remove it in step 2. Recommended, since the backup branch `archive/main-pre-native-20261104` holds it anyway. The owner earlier said to keep it for now and discuss.
7. **The suite.** Remove its runner and arms, and keep the Makerspace projects and hidden cases, which step 3 runs on. Recommended, and open.
8. **`update`.** The owner proposed it. Hold it back until the tool has settled, as [§3](#commands-on-the-spec) does? Recommended.
9. **The app.** Does "remove desktop" mean the whole app, which takes Docker out of Markitect? This draft assumes yes.
10. **What "reliable" means.** Step 3's two clean passes on the 8 change steps is a proposal for the talk on evaluation the owner asked for. Is that the finish line?
