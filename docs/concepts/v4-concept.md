# Markitect concept 4.0 (draft)

**You model. Markitect's tools check the spec and work out what a change reaches, an ordinary agent, started by the owner's own session, makes the project match, and one tool command decides whether the work counts.**

| | |
|---|---|
| **Version** | 4.0 – draft, not accepted |
| **Date** | 4 November 2026 |
| **Basis** | The [owner's statements](sources/fresh-start-20261104.md) of 4 November 2026, concept 3.0, a workflow trial on one real change, the Claude Code documentation and the help of version 2.1.318 as read that day, and independent reviews of this draft and an earlier one. |
| **State** | A draft for the owner ([OQ-012](register.md#open-questions)). [Concept 3.0](v3-concept.md) stays the target until this one is accepted. Main carries its build since 6 November 2026 ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)), for the owner to try by using it ([what real sessions showed](../validation/native-version-20261104.md)). On 6 November 2026 the owner chose a leaner tool that computes and checks and leaves the agents to a driver ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)); its [plan](lean-tool-plan.md) will revise §5 to §9 once it is built and tried. |

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
| `install` | Writes the personal skill, once per machine. Optional: `init` says the same and gives a project its own skills ([§9](#9-the-harness)) | New |
| `init` | Sets up a project: `.markitect/`, the first page of the spec, the harness, and `AGENTS.md` and `CLAUDE.md` if they are missing. An empty folder is enough: it becomes a Git repository. Run again, it refreshes the harness and never touches spec pages | Writes the reviewer agent and the new skills, and commits what it writes, so the first `apply` finds a clean folder. It writes no file that would hold nothing yet, and it ends by saying what Markitect is and pointing to `/markitect-start` |
| `lock` | Pins the Libraries. It holds a fingerprint of each one, as `go.sum` does for modules, and `check` refuses a Library that no longer matches | None |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself | Stale entries and unused Definitions become errors, except in a project with nothing built yet. `--spec` is new |
| `refs TARGET` | Who links to a Definition or Package, and which files realize it | None |
| `impact` | What a change reaches ([below](#how-impact-works)) | `--base` defaults to the last Result. The per-Package lists go |
| `analyze` | Measures the spec's design ([§4](#4-is-the-spec-good)). `--severity` filters its findings, and `--graph` draws the Package dependency graph | New. Replaces `lint` and `graph`. Comes after `apply` is reliable ([§13](#13-order-of-work)) |
| `update` | Installs the newest release over the binary, then lets the new binary bring what an earlier build wrote in line with it: the personal skill if there is one, and in a project the harness, which it commits, and the lock, which it rewrites and leaves uncommitted for the owner to read. `--check` changes nothing and exits 1 if anything is out of date; `--files` leaves the binary alone; `--force` replaces a development build | New, on the owner's word of 6 November 2026 ([§16](#16-questions-for-the-owner), [DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reuses what `init` and `lock` write and replaces neither |
| `version` | The build, and whether the personal skill and the project match it | Adds the status lines |
| `licenses` | As named | None |

The owner proposed `update` to replace `lock` and `init`'s refreshing. An earlier version of this draft held it back, because the two commands already did that work. On 6 November 2026 the owner asked for it, with a `version` that says what is out of date, since an upgraded binary left the personal skill and each project's harness behind with nothing to say so. So `update` is thin: it runs `init`'s writers and `lock` over what differs, compares files without regard to line endings, commits the harness but not the lock, and never touches a spec file. Commands that read a project print one line when its skills and agents are old.

The same day the owner added releases and the binary's own upgrade, so that `update` does what `claude update` does ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reads the newest tag from the redirect of GitHub's releases page, downloads the binary for the system, checks it against the checksums file of that release and replaces the running executable: by renaming over it on Linux and macOS, and on Windows, which lets a program rename its own executable but not overwrite it, by moving the old file aside first. The new binary then finishes the update, since only it knows the files it ships. This is the only place where Markitect uses the network, and only when the owner runs `update`; without a network it says so and still refreshes the files. A development build older than the newest release (`0.0.0-dev`, or `0.15.0-dev` while `0.15.0` is out) is not replaced unless `--force` says so, so a build from source is safe from a stray `update`; one newer than the release (`0.16.0-dev` after `0.15.0`) counts as up to date. The download is checked against checksums from the same release over HTTPS, so it protects against a damaged or swapped file but not against a release published by someone who controls the repository; signed attestations would add that.

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

**Run state** lives under the repository's Git directory, in `markitect/`, and never reaches the working tree. Work that did not land is copied to `markitect/runs/<run>`, where the owner can look at it, and the Result says why. It also stays in the owner's folder, uncommitted.

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
3. **Say which agent to start.** `apply` saves the run and prints one step: start the `markitect-builder` agent with the message "Build run NAME". The apply skill has the owner's own session do exactly that. The builder is a subagent: it starts with fresh context, in the owner's folder, and knows only the spec, the list and what the tool prints.
4. **Implement.** The builder takes the list from `markitect checklist` and makes the project match the spec, keeping the registry current. It does not change the spec. If the spec contradicts itself or leaves open something that changes behaviour at the project's boundary, it runs `markitect stop` with why. A gap with no such effect is its own to decide. If the machine lacks something, such as a program or a service, it says so with `markitect stop --environment`, which asks the owner nothing about the spec.
5. **Submit.** When it thinks it is done, it calls `markitect submit`, and follows what that says ([§6](#6-the-one-door-submit)).
6. **Review.** `submit` tells it to start the `markitect-reviewer` agent, a subagent of its own. The reviewer starts with fresh context, has no tool to edit files and cannot start further agents. It takes the list from `markitect checklist`, reads the Definitions and the code, and stores one line per item with `markitect answer` ([concept 3.0 §8](v3-concept.md#8-the-checklist)).
7. **Repair.** A `wrong` answer goes back to the builder, which fixes it and calls `submit` again. The next reviewer gets the findings, the new diff and the items whose locations that diff touched; other answers stand. At most two repair rounds.
8. **Land.** When nothing is left to do, `submit` writes the Result, commits it with the work and brings the owner's branch up. The builder is told it has landed, and ends. The owner's session sees it end and reports.

**Bringing the branch up.** `submit` builds what will land as commits, without touching the folder: the run's commit if the branch has not moved, otherwise a merge, which must be clean and pass `check --spec`. Later commits stay in the next `impact`, so the next run realizes them. Then it moves the recorded branch, only if it is still where it was. Where it is checked out, the files follow as in a fast-forward: a file that already holds the landing's version, such as the work the builder wrote in that folder, stays as it is, and a file with an edit of its own stops the move. The files change before the branch moves, so a landing cut off part way runs again. Where it is not checked out, only the ref moves. A short wait and a retry cover Git's own locks. Nothing is pushed: the owner pushes when they want to, to any remote or none.

**If landing fails,** the run closes as kept on `markitect/runs/<run>`, the Result says why, and the builder is told to finish. The work also stays in the folder, uncommitted. The next `apply` tries that branch once more. If that fails too, it leaves the branch where it is and starts a fresh run.

**One run per folder.** The agents work in the owner's folder, so while a run is open, a second `apply` or `verify` there starts nothing and says how to go on with it. The owner leaves the folder alone meanwhile; the session that started the run is busy with it anyway.

**Telling the owner.** The owner's session waits for the builder, so it sees the end. If the builder ends while the run is still open, `apply` has told the session to start it again with the same message. Every `markitect` command also prints one line about a run that closed unread.

**Why not a background session.** The first build of this draft had `apply` start `claude --bg` in a worktree of its own, with generated settings and hooks, and it landed real changes ([record](../validation/native-version-20261104.md)). On 5 November 2026 the owner chose agents started by their own session instead: Markitect should be just a way of agentic coding, without that machinery. What the choice costs:

- the owner's session is busy until the run ends, and the run stops if that session closes;
- the agents work under that session's permission mode, settings, plugins and memory, so they may ask for permission, and they differ from one person to the next;
- an agent that ends early is started again by the owner's session, as `apply` says, instead of being sent back by a hook.

What it keeps: fresh context for the builder and for the reviewer, which is the test that the spec alone is enough, and every refusal of `submit`.

## 6. The one door: `submit`

`submit` looks at the current state and does the first thing that applies:

| | It finds | It does | It tells the agent |
|---|---|---|---|
| 0 | The folder is no longer on the run's branch at its start, because of a commit made during the run or another branch checked out | Refuses | Undo the commits with `git reset --soft`, or check the branch out again |
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
| A run doesn't stop to wait for someone | Nothing: it works under the owner's session's permission mode. The builder accepts file edits without asking | Weak by choice: the owner is there to answer |
| The work doesn't mix with the owner's own | One run per folder, and the owner leaves the folder alone while it is open. A landing stops at a file with an edit of its own | Good for what lands; nothing keeps the owner from editing |
| A run doesn't just stop | `apply` tells the owner's session to start the builder again while the run is open | Prompt text, not structure. An early end lands nothing, so it costs time, not correctness |
| The agent doesn't review itself | A hook in the project's settings lets only a reviewer call `answer`; it runs in subagents too | Good, not proof: it reads the command's text and the agent's name |
| The reviewer judges for itself | Its agent file says to take everything from `markitect checklist` | Weak: the builder writes the message that starts it. Reviews given a maker's claims did no worse on planted bugs ([17 of 18 blind, 18 of 18 with claims](../validation/k-checker-test-20261103.md)), though that tested claims beside bugs, not a message written to steer. No hook is added until this is seen |
| Nothing goes to a remote | The task says the tool handles Git | Catches ordinary use only: the owner's own settings decide what a shell command may do |
| A commit that bypassed `submit` is noticed | It stays in the next `impact`, and `status` counts commits since the last Result | Seen afterwards, not prevented |
| The reviewer really read the code | Pointers checked for existence | Not provable. It was not provable in the Docker version either |
| The code is right | The reviewer and the project's tests | Not guaranteed by any version |
| A run has a limit | The deadline and the ceiling of [§6](#6-the-one-door-submit), held by the tool | Good while the agent calls tools. The provider's own budget and turn flags work only with `-p`, and there is no spending limit |

**Anyone who wants more** can run `markitect check` and the project's tests where the agent can't reach them: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these. Answers stay on the machine, and a landing's Result carries them.

**What we give up against concept 3.0:**

- prevention of a deliberate bypass;
- isolation from the user's machine: agents run with the user's rights, so a run must not read untrusted input;
- one operating system: agents work on the user's own, Windows included, with its paths and shells;
- a reviewer in its own session as another user, with a throwaway folder. It now shares the owner's folder. `submit` catches what it changes in files Git would take, and code it runs can still change its stored answers;
- review of a clean copy: it reads the folder as it stands, ignored leftovers included;
- a reviewer's task written by the Command, and a measure of how much it checked.

## 8. Other Commands

Every Command is one short declaration over the same tools: which agent starts, where its items come from, and whether it lands. The table in [concept 3.0 §12](v3-concept.md#12-other-commands) stands. A skill for a Command is one line that calls it, so the same Command runs from a chat, a terminal or a schedule.

**`verify`** changes nothing the owner has. It opens a review of the last commit and says to start the `markitect-reviewer` agent, which the owner's session does. The reviewer reads the folder, so `verify` refuses one with uncommitted changes. It doesn't commit on the owner's branch. With no Result yet, it is `--full`. Like `apply`, it holds the folder while it is open.

`submit` works differently for it:

| It finds | It does |
|---|---|
| Items without an answer | Lists them, under the same ceiling and deadline |
| Otherwise | Runs `check`. A wrong answer becomes a Finding, and what names no Definition, such as an error from `check`, is listed in the Result. It writes the Result and a Findings page on `markitect/runs/<run>` and closes the run |

`apply --findings` and the count of changes since the last full check read them there.

`--full` starts with one reviewer for the whole list, and the first `apply` on a project is `--full` too. That is untested at size: the 5× build used 51 reviewers, up to eight at a time, each in a clean copy. A split per Package may be needed, and the runs of [§13](#13-order-of-work) decide that.

`apply` comes first, then `verify`. `cleanup`, `bughunt`, `security` and `improve` follow once those work. Modeling and Extract stay skills in the owner's own session, as they are, and so does `markitect-start`, the guide a new project begins with. It asks what the project is, one question at a time, and starts the spec with the owner.

## 9. The harness

For Claude Code, `markitect init` writes these project files, so they are versioned with the project, and the tool accounts for them itself.

| File | Holds |
|---|---|
| `.claude/skills/markitect-apply/SKILL.md` | One injected command, `` !`markitect apply` ``, and "do what it printed". Its front matter sets `allowed-tools`, so the command runs without a prompt, and `disable-model-invocation: true`, so a modeling agent can't start a run by itself. The provider runs the command before the model sees the skill, and any non-zero exit aborts the skill, so "run open" and "nothing to do" exit 0 |
| `.claude/skills/markitect-verify/SKILL.md` | The same for `verify` |
| `.claude/agents/markitect-builder.md` | The builder: take the task from `markitect checklist`, and be done when `markitect submit` says so. All tools, including starting the reviewer; model `sonnet`, effort `high`, file edits without asking |
| `.claude/agents/markitect-reviewer.md` | Effort, the reviewer's task, and tools without edit or agent tools |
| `.claude/settings.json` | The hook on `answer`, below, added to whatever settings the project already has |
| `AGENTS.md`, `CLAUDE.md` | What kind of project this is, for any session: the spec is the authority, changes start there, and `markitect` is on the path. `CLAUDE.md` holds only `@AGENTS.md`, so that it works in every session of the provider. They hold no process, since a run reads them too. Written only if missing |

**The agents' settings** come from their files, and the manifest's `model`, if it names one, which `apply` passes on when it says which agent to start. Everything else is the owner's session: its permission mode, settings, plugins and memory. No settings are generated per run.

**The hook.** `markitect hook pretool`, in the project's settings, runs before a shell command that calls `markitect answer`, and refuses it unless the agent is the reviewer. Hooks in project settings run inside subagents too, and their input names the agent. A run may not change the file.

The first draft's seal is gone. `answer` checks and stores the answers itself, so no hook has to catch a reviewer as it finishes.

**One file lies outside the project.** A session that runs `init` started before the project's files existed, so it knows only what the tool prints, and Claude Code doesn't watch a skills folder that wasn't there when the session started. So help and `init` say what Markitect is, and `markitect install` writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, with the same text from the binary, which reaches a session in any folder. It allows no command in advance. `init` never writes to the home folder; it only tips when the skill is missing or old. `markitect update` installs the newest release and refreshes the skill and the project's files, and `markitect version` says whether they match. The skills and agents call `markitect` by name, so `version`, `init`, `install` and `update` warn when the `PATH` finds no `markitect` or another one.

**The plugin does both.** `plugin/` is a Claude Code plugin, `markitect`, in this repository. The marketplace that lists it, `glacius-labs`, is the small repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins), whose entry pins the plugin to the tag of a release: the plugin that a user gets is the one at the tagged commit, whatever `main` holds. It carries the personal skill as `markitect:getting-started` (the same text, pinned by a test, under a name that does not repeat the plugin's, since Claude Code names a plugin's skill `plugin:skill`), a launcher, `bin/markitect`, which Claude Code puts on the `PATH` of its shell, and two hooks. The launcher reads the plugin's version, downloads the binary of the release with that version into the plugin's data folder, checks it against the release's checksums and runs it, with `MARKITECT_PLUGIN` set so that the binary knows who started it: `update` then leaves the binary to the plugin (`claude plugin update markitect@glacius-labs`), and the `PATH` warning stays quiet, since the launcher is what the `PATH` leads to. A session-start hook has the launcher fetch the binary in the background and outside the shell's sandbox, which may block the network, but only in a project that has a `.markitect/` folder: in any other project the plugin stays out of the way, and the binary is fetched at the first call. The other hook runs `markitect hook pretool` before `markitect answer`, as the project's hook does, because the project's hook runs in a shell that does not see the plugin's `bin/` and would find no `markitect` there. It looks at the command itself and starts the launcher, and so a download, only for a command that contains `markitect answer`; the `if` filter of the hook alone was seen to start it for other commands too, in 0.17.0. After `init` both run, and both give the same answer; if the launcher fails, this one exits 2 and blocks the call, which is right, since `markitect answer` would fail too. The plugin does not replace `init`: the project's skills, agents and hook still come from `markitect init`.

Skills are kept short, and they hold no steps, so nothing depends on what survives compaction. The harness's agents stay in the project rather than in a plugin: the provider ignores hooks and permission modes in plugin agents. The plugin described below brings only the command, the personal skill and two hooks.

## 10. Strategies

**A strategy is a folder of harness files and nothing else.** There is no code interface to implement. Adding a version means adding a folder, and removing one means deleting it. **It may change** the prompt, agents, skills and settings. **It may not change** the tools or what `submit` refuses.

The first version ships one strategy and no switch for it. The natural second is a saved provider workflow that fixes the order of steps, as in the trial of [§14](#14-what-is-not-yet-shown), added only if the first one fails too often ([§16](#16-questions-for-the-owner)).

## 11. Providers

The first version is Claude only. A provider needs subagents with their own agent files, and a hook before a tool call that names the agent. Codex gets its own strategy once Claude's version is reliable.

No token is needed: sessions run on the user's own stored login in the unmodified program, which removes token passing. Each session uses the user's subscription quota. The legal reading of [concept 3.0 §14, question 10](v3-concept.md#14-decided-on-acceptance) still applies: the owner's own use looks allowed, and other users are not settled. So the owner still asks Anthropic before anyone else runs Markitect.

**The app.** The owner said "we can remove desktop for now". This draft reads that as the whole app, with its spec views and chats. Its chats are the only other use of containers, so this takes Docker out of Markitect altogether, along with the provider adapters, the session helper, the runtime image and the token. It stays recoverable in Git and can come back once the tool has settled. The owner confirmed that reading on 6 November 2026, and the app is gone ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).

## 12. Schedules and continuous work

[DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps) stands: the loop runs outside Markitect, as a scheduled job that calls `markitect`, and there is no daemon in the product. Its budget per task is the run's deadline and ceiling; a budget per day belongs to the job. Government stays a later step ([IDEA-002](register.md#idea-002-government-delegated-model-maintenance)): its mandate would live in the spec, and a fresh reviewer would accept proposals within it.

A schedule now needs a provider session to do what `markitect verify` or `markitect apply` prints, since the tool starts no agent. Two limits stay:

- **The machine must be on and signed in.** A server without a stored login is not covered yet.
- **External signals need isolation.** Issue text can hold instructions aimed at the agent, and a native agent has the user's rights. A run that reads outside input needs a container or a machine with limited rights around it. This wraps the harness and needs no engine.

## 13. Order of work

| Step | Result | Done when |
|---|---|---|
| 1. Probe | The harness by hand on the Library, on Windows, under a bounded dispatch. It tries what [§14](#14-what-is-not-yet-shown) lists | Each works, or the concept is changed |
| 2. Build | `apply`, `checklist`, `answer`, `submit`, `stop`, `status`, `hook` and the stricter `check` in Go, reusing what `internal/run` holds. `init` writes the harness. The engine's container and session code goes, roughly 7,000 lines, once the probe has passed | A Library change lands, planted refusals hold, and a `submit` killed at each step can be run again |
| 3. Make it reliable | The 5× project's 8 change steps on Sonnet, with their hidden cases. Failures are fixed in the tool, not with more prompt text | Two clean passes in a row: every step lands without the owner, all hidden cases pass and no refusal was overridden ([§16](#16-questions-for-the-owner)) |
| 4. The rest | `verify`, then the other Commands, `analyze` and the rename | The owner's reading |

Before step 3, read test 1's nine finished launches. They are in `%LOCALAPPDATA%\markitect-test1`, not in the repository, and their answers are unread. They may say at no new cost whether one reviewer catches known bugs at 5×.

Concept 3.0's tests 2 and 3 fall away, since the aim is no longer to show that this beats something else.

## 14. What is not yet shown

**The background version was tried on Windows on small changes ([record](../validation/native-version-20261104.md)).** That record covers the launcher, settings and hooks the owner has since dropped. For agents started by the owner's own session, nothing has run yet. What a first run has to show:

- the apply skill's printed step makes the owner's session start the builder, and that session starts it again when the run is still open;
- the builder, as a subagent, starts the reviewer as its own subagent and waits for it;
- the hook in the project's settings fires inside a subagent and refuses `answer` to anyone but the reviewer;
- landing over the work in the owner's folder leaves it clean, on Windows line endings too;
- how often the owner's permission mode stops a builder to ask.

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
| [DEC-066](register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [DEC-089](register.md#dec-089-markitect-desktop-the-app-over-the-cli) | Parked: no app for now, as the owner confirmed on 6 November 2026. If it returns, it works over the same commands | The idea of an app over the CLI. The owner being told when a run ends moves to the tool |
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

Each has a recommendation, and none is decided. Already decided by the owner: the name `submit`, that `submit` brings the branch up by itself, that required checks go for now to see whether they are needed, that a run's agents are started by the owner's own session, that the app goes, that `update` comes now, and that releases, the plugin and a binary that updates itself come with it ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)).

1. **Starting the session.** Decided by the owner on 5 November 2026: the owner's own session starts the agents, with no background session ([§5](#5-what-apply-does)).
2. **What a run may do.** Settled with question 1: the agents work under the owner's own permission mode and settings, and the builder accepts file edits without asking.
3. **How the owner hears of the end of a run.** Settled with question 1: the owner's session waits for the builder and sees it end.
4. **A run's limit.** A deadline of 6 hours and 30 `submit`s, with no spending limit, since the provider offers none here. Recommended for now; the numbers are guesses for step 3 to correct.
5. **One version or two.** The owner asked for "the two simplest approaches". Recommended: one now, the workflow as the second only if this one fails too often. The owner also expected a package per version behind an interface; a strategy here is a folder of harness files, with no code interface, because the tool does not change between versions.
6. **The old version.** Settled on 6 November 2026: Main carries this version, and Main until then is kept on the branch `archive/main-concept-3-20261106` ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).
7. **The suite.** Remove its runner and arms, and keep the Makerspace projects and hidden cases, which step 3 runs on. Recommended, and open.
8. **`update`.** Decided by the owner on 6 November 2026: it comes now, with a `version` that reports what is out of date ([§3](#commands-on-the-spec)). The Claude Code plugin that would carry the personal skill and update it with the provider's own mechanism, first held back, comes too: the owner said on 6 November 2026 that releases come anyway, and that the plugin should set up what a user needs ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)).
9. **The app.** Decided by the owner on 6 November 2026: the whole app goes ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).
10. **What "reliable" means.** Step 3's two clean passes on the 8 change steps is a proposal for the talk on evaluation the owner asked for. Is that the finish line?
